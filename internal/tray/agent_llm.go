//go:build !nogui && !headless && !linux

package tray

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mcpproxy-go/internal/config"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"
)

// LLMAgent is a universal agent powered by LLM (OpenAI) that can handle all tasks
type LLMAgent struct {
	logger        *zap.Logger
	llmClient     LLMClient
	serverManager interface {
		GetServerTools(serverName string) ([]map[string]interface{}, error)
		EnableServer(serverName string, enabled bool) error
		GetAllServers() ([]map[string]interface{}, error)
		ReloadConfiguration() error
		GetConfigPath() string
		GetLogDir() string
		GetGitHubURL() string
	}
}

// NewLLMAgent creates a new LLM-powered agent with config-based client
func NewLLMAgent(logger *zap.Logger, llmConfig *config.LLMConfig, serverManager interface {
	GetServerTools(serverName string) ([]map[string]interface{}, error)
	EnableServer(serverName string, enabled bool) error
	GetAllServers() ([]map[string]interface{}, error)
	ReloadConfiguration() error
	GetConfigPath() string
	GetLogDir() string
	GetGitHubURL() string
}) *LLMAgent {
	// Create LLM client based on configuration
	llmClient := NewLLMClientFromConfig(llmConfig)

	logger.Info("LLM Agent initialized",
		zap.String("provider", func() string {
			if llmConfig != nil {
				return llmConfig.Provider
			}
			return "openai (default)"
		}()),
		zap.String("model", func() string {
			if llmConfig != nil {
				return llmConfig.Model
			}
			return "gpt-4o-mini (default)"
		}()))

	return &LLMAgent{
		logger:        logger,
		llmClient:     llmClient,
		serverManager: serverManager,
	}
}

// ProcessMessage processes a user message using LLM
func (a *LLMAgent) ProcessMessage(ctx context.Context, message ChatMessage, session *ChatSession) (*ChatMessage, error) {
	a.logger.Info("LLM agent processing message",
		zap.String("session_id", session.ID),
		zap.String("server", session.ServerName),
		zap.String("content_preview", truncateString(message.Content, 50)))

	// Prune conversation context to avoid token limits
	// GPT-4o-mini has 128k context, we reserve ~25k for system prompt + tools + response
	const maxContextTokens = 100000
	prunedMessages := a.pruneConversationContext(session.Messages, maxContextTokens)

	// Temporarily update session with pruned messages for context building
	originalMessages := session.Messages
	session.Messages = prunedMessages

	// Build context from chat history
	conversationContext := a.buildConversationContext(session)

	// Restore original messages (pruning was just for API call)
	session.Messages = originalMessages

	// Create comprehensive prompt with server context
	prompt := a.buildPrompt(message.Content, session.ServerName, conversationContext)

	// Define available tools
	tools := a.getTools()

	// Create tool executor
	toolExecutor := a.createToolExecutor(session.ServerName)

	// Get LLM response with tool support
	response, toolCalls, err := a.llmClient.AnalyzeWithTools(prompt, tools, toolExecutor)
	if err != nil {
		a.logger.Error("LLM analysis failed", zap.Error(err))
		return nil, fmt.Errorf("LLM analysis failed: %w", err)
	}

	// Format tool calls for display
	formattedResponse := response
	if len(toolCalls) > 0 {
		formattedResponse += a.formatToolCalls(toolCalls)
	}

	// Detect config changes
	configChanged := false
	for _, tc := range toolCalls {
		if tc.ToolName == "write_config" && tc.Error == "" {
			configChanged = true
			break
		}
	}

	// Create response message
	metadata := map[string]interface{}{
		"model": "gpt-4o-mini",
	}

	// Add tool calls to metadata if present
	if len(toolCalls) > 0 {
		metadata["tool_calls"] = toolCalls
	}

	// Mark config changes
	if configChanged {
		metadata["config_changed"] = true
	}

	responseMsg := &ChatMessage{
		ID:        generateMessageID(),
		Role:      "assistant",
		Content:   formattedResponse,
		AgentType: "llm",
		Timestamp: time.Now(),
		Metadata:  metadata,
	}

	return responseMsg, nil
}

// formatToolCalls formats tool call records for display in the chat
func (a *LLMAgent) formatToolCalls(toolCalls []ToolCallRecord) string {
	if len(toolCalls) == 0 {
		return ""
	}

	var output strings.Builder
	output.WriteString("\n\n🔧 **MCP Server Tool Calls:**\n")

	for i, tc := range toolCalls {
		output.WriteString(fmt.Sprintf("\n**%d. %s**\n", i+1, tc.ToolName))

		// Arguments
		if len(tc.Arguments) > 0 {
			output.WriteString("   📥 Arguments:\n")
			for key, value := range tc.Arguments {
				// Truncate long values
				valueStr := fmt.Sprintf("%v", value)
				if len(valueStr) > 100 {
					valueStr = valueStr[:97] + "..."
				}
				output.WriteString(fmt.Sprintf("      • %s: %s\n", key, valueStr))
			}
		}

		// Result or Error
		if tc.Error != "" {
			output.WriteString(fmt.Sprintf("   ❌ Error: %s\n", tc.Error))
		} else {
			// Truncate long results
			result := tc.Result
			if len(result) > 200 {
				result = result[:197] + "..."
			}
			output.WriteString(fmt.Sprintf("   ✅ Result: %s\n", result))
		}

		// Duration
		output.WriteString(fmt.Sprintf("   ⏱️  Duration: %v\n", tc.Duration.Round(time.Millisecond)))
	}

	return output.String()
}

// estimateTokenCount estimates the number of tokens in a string
// Using rough approximation: 1 token ≈ 4 characters
func estimateTokenCount(text string) int {
	return len(text) / 4
}

// estimateMessageTokens estimates tokens for a chat message including metadata
func estimateMessageTokens(msg ChatMessage) int {
	tokens := estimateTokenCount(msg.Content)

	// Add tokens for tool calls in metadata
	if msg.Metadata != nil {
		if toolCalls, ok := msg.Metadata["tool_calls"].([]ToolCallRecord); ok {
			for _, tc := range toolCalls {
				tokens += estimateTokenCount(tc.ToolName)

				// Arguments
				if argsJSON, err := json.Marshal(tc.Arguments); err == nil {
					tokens += estimateTokenCount(string(argsJSON))
				}

				// Result
				tokens += estimateTokenCount(tc.Result)
				tokens += estimateTokenCount(tc.Error)
			}
		}
	}

	return tokens
}

// pruneConversationContext intelligently reduces conversation history to fit token limits
// Strategy:
// 1. Keep system message and most recent messages
// 2. Summarize middle messages
// 3. Remove detailed tool call data from older messages
// 4. Preserve important context markers (config changes, errors)
func (a *LLMAgent) pruneConversationContext(messages []ChatMessage, maxTokens int) []ChatMessage {
	const (
		systemMessageTokens = 1000 // Reserve for system message
		recentMessageCount  = 5    // Always keep the last N messages
	)
	targetTokens := maxTokens - systemMessageTokens - 1000 // Safety margin

	// Calculate total tokens
	totalTokens := 0
	messageTokens := make([]int, len(messages))
	for i, msg := range messages {
		tokens := estimateMessageTokens(msg)
		messageTokens[i] = tokens
		totalTokens += tokens
	}

	a.logger.Info("Context pruning analysis",
		zap.Int("total_messages", len(messages)),
		zap.Int("total_tokens", totalTokens),
		zap.Int("max_tokens", maxTokens),
		zap.Int("target_tokens", targetTokens))

	// If we're under the limit, return as-is
	if totalTokens <= targetTokens {
		a.logger.Info("Context within limits, no pruning needed")
		return messages
	}

	// Strategy: Keep first message (if system) + recent messages + summarize/compress middle
	prunedMessages := []ChatMessage{}
	currentTokens := 0

	// 1. Keep the first message if it's important context
	if len(messages) > 0 && (messages[0].Role == "system" || messages[0].AgentType == "llm") {
		prunedMessages = append(prunedMessages, messages[0])
		currentTokens += messageTokens[0]
	}

	// 2. Always keep the most recent messages
	recentStartIdx := len(messages) - recentMessageCount
	if recentStartIdx < 1 {
		recentStartIdx = 1
	}

	// 3. Process middle messages with compression
	middleMessages := []ChatMessage{}
	if recentStartIdx > 1 {
		for i := 1; i < recentStartIdx; i++ {
			compressed := a.compressMessage(messages[i])
			compressedTokens := estimateMessageTokens(compressed)

			// Only include if we have room and it's important
			if currentTokens+compressedTokens <= targetTokens-recentMessageCount*500 {
				middleMessages = append(middleMessages, compressed)
				currentTokens += compressedTokens
			}
		}

		// If we have too many middle messages, create a summary
		if len(middleMessages) > 10 {
			summary := a.createConversationSummary(middleMessages)
			prunedMessages = append(prunedMessages, summary)
			currentTokens += estimateMessageTokens(summary)
		} else {
			prunedMessages = append(prunedMessages, middleMessages...)
		}
	}

	// 4. Add recent messages (these are kept in full detail)
	for i := recentStartIdx; i < len(messages); i++ {
		prunedMessages = append(prunedMessages, messages[i])
		currentTokens += messageTokens[i]
	}

	newTotalTokens := 0
	for _, msg := range prunedMessages {
		newTotalTokens += estimateMessageTokens(msg)
	}

	a.logger.Info("Context pruning complete",
		zap.Int("original_messages", len(messages)),
		zap.Int("pruned_messages", len(prunedMessages)),
		zap.Int("original_tokens", totalTokens),
		zap.Int("pruned_tokens", newTotalTokens),
		zap.Int("tokens_saved", totalTokens-newTotalTokens))

	return prunedMessages
}

// compressMessage removes detailed information while keeping essential content
func (a *LLMAgent) compressMessage(msg ChatMessage) ChatMessage {
	compressed := msg

	// Remove detailed tool call information from metadata
	if compressed.Metadata != nil {
		if toolCalls, ok := compressed.Metadata["tool_calls"].([]ToolCallRecord); ok && len(toolCalls) > 0 {
			// Create compressed version with just tool names
			compressedCalls := []map[string]interface{}{}
			for _, tc := range toolCalls {
				compressedCalls = append(compressedCalls, map[string]interface{}{
					"tool_name": tc.ToolName,
					"status":    func() string {
						if tc.Error != "" {
							return "error"
						}
						return "success"
					}(),
				})
			}
			compressed.Metadata["tool_calls_summary"] = compressedCalls
			delete(compressed.Metadata, "tool_calls")
		}
	}

	// Truncate very long content
	if len(compressed.Content) > 500 {
		compressed.Content = compressed.Content[:497] + "..."
	}

	return compressed
}

// createConversationSummary creates a single summary message from multiple messages
func (a *LLMAgent) createConversationSummary(messages []ChatMessage) ChatMessage {
	userQuestions := 0
	agentResponses := 0
	configChanges := 0
	toolCallCount := 0

	for _, msg := range messages {
		if msg.Role == "user" {
			userQuestions++
		} else if msg.AgentType == "llm" {
			agentResponses++
		}

		if msg.Metadata != nil {
			if configChanged, ok := msg.Metadata["config_changed"].(bool); ok && configChanged {
				configChanges++
			}
			if toolCalls, ok := msg.Metadata["tool_calls"].([]ToolCallRecord); ok {
				toolCallCount += len(toolCalls)
			}
		}
	}

	summaryContent := fmt.Sprintf(
		"[Earlier conversation summary: %d user questions, %d agent responses, %d config changes, %d tool calls]",
		userQuestions, agentResponses, configChanges, toolCallCount,
	)

	// Add important topics if present
	topics := []string{}
	for _, msg := range messages {
		if strings.Contains(strings.ToLower(msg.Content), "error") {
			topics = append(topics, "error handling")
			break
		}
	}
	for _, msg := range messages {
		if strings.Contains(strings.ToLower(msg.Content), "config") {
			topics = append(topics, "configuration")
			break
		}
	}

	if len(topics) > 0 {
		summaryContent += fmt.Sprintf(" Topics: %s", strings.Join(topics, ", "))
	}

	return ChatMessage{
		ID:        "summary_" + fmt.Sprintf("%d", time.Now().Unix()),
		Role:      "assistant",
		Content:   summaryContent,
		AgentType: "system",
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"is_summary":      true,
			"summarized_count": len(messages),
		},
	}
}

// buildConversationContext creates a summary of recent conversation
func (a *LLMAgent) buildConversationContext(session *ChatSession) string {
	if len(session.Messages) <= 1 {
		return ""
	}

	var contextParts []string
	// Get last 5 messages for context (exclude the current one)
	startIdx := len(session.Messages) - 6
	if startIdx < 0 {
		startIdx = 0
	}

	for i := startIdx; i < len(session.Messages)-1; i++ {
		msg := session.Messages[i]
		contextParts = append(contextParts, fmt.Sprintf("%s: %s", msg.Role, truncateString(msg.Content, 100)))
	}

	if len(contextParts) == 0 {
		return ""
	}

	return "\n\nRecent conversation:\n" + strings.Join(contextParts, "\n")
}

// buildPrompt creates a comprehensive prompt for the LLM
func (a *LLMAgent) buildPrompt(userMessage, serverName, conversationContext string) string {
	// Get comprehensive server information
	var serverInfo string
	if a.serverManager != nil {
		servers, err := a.serverManager.GetAllServers()
		if err == nil {
			for _, srv := range servers {
				if name, ok := srv["name"].(string); ok && name == serverName {
					// Basic server information
					serverInfo = fmt.Sprintf("\n\n=== Server Information ===\n- Name: %s\n- Enabled: %v\n- Protocol: %v",
						name,
						srv["enabled"],
						srv["protocol"])

					// Connection details
					if url, ok := srv["url"].(string); ok && url != "" {
						serverInfo += fmt.Sprintf("\n- URL: %s", url)
					}
					if cmd, ok := srv["command"].(string); ok && cmd != "" {
						serverInfo += fmt.Sprintf("\n- Command: %s", cmd)
					}
					if args, ok := srv["args"].([]interface{}); ok && len(args) > 0 {
						argsStr := make([]string, len(args))
						for i, arg := range args {
							argsStr[i] = fmt.Sprintf("%v", arg)
						}
						serverInfo += fmt.Sprintf("\n- Arguments: %s", strings.Join(argsStr, " "))
					}

					// Working directory
					if workingDir, ok := srv["working_dir"].(string); ok && workingDir != "" {
						serverInfo += fmt.Sprintf("\n- Working Directory: %s", workingDir)
					}

					// Status and state with visual indicators
					enabled, _ := srv["enabled"].(bool)
					connected, _ := srv["connected"].(bool)
					connecting, _ := srv["connecting"].(bool)
					quarantined, _ := srv["quarantined"].(bool)
					sleeping, _ := srv["sleeping"].(bool)

					var statusIcon string
					var statusText string

					if quarantined {
						statusIcon = "🔒"
						statusText = "QUARANTINED"
					} else if !enabled {
						statusIcon = "⏸️"
						statusText = "DISABLED"
					} else if sleeping {
						statusIcon = "💤"
						statusText = "SLEEPING (Lazy Loading)"
					} else if connected {
						statusIcon = "🟢"
						statusText = "CONNECTED"
					} else if connecting {
						statusIcon = "🟡"
						statusText = "CONNECTING..."
					} else {
						statusIcon = "🔴"
						statusText = "DISCONNECTED"
					}

					serverInfo += fmt.Sprintf("\n- Status: %s %s", statusIcon, statusText)

					if connectionState, ok := srv["connection_state"].(string); ok && connectionState != "" {
						serverInfo += fmt.Sprintf("\n- Connection State: %s", connectionState)
					}

					// Tool count
					if toolCount, ok := srv["tool_count"].(int); ok && toolCount > 0 {
						serverInfo += fmt.Sprintf("\n- Available Tools: %d", toolCount)
					} else if toolCount, ok := srv["tool_count"].(float64); ok && toolCount > 0 {
						serverInfo += fmt.Sprintf("\n- Available Tools: %d", int(toolCount))
					}

					// Error information
					if lastError, ok := srv["last_error"].(string); ok && lastError != "" {
						serverInfo += fmt.Sprintf("\n- Last Error: %s", lastError)
					}

					// Configuration path
					configPath := a.serverManager.GetConfigPath()
					if configPath != "" {
						serverInfo += fmt.Sprintf("\n- Config File: %s", configPath)
					}

					// Repository URL if available
					if repoURL, ok := srv["repository_url"].(string); ok && repoURL != "" {
						serverInfo += fmt.Sprintf("\n- Repository: %s", repoURL)
					} else if homepage, ok := srv["homepage"].(string); ok && homepage != "" {
						serverInfo += fmt.Sprintf("\n- Homepage: %s", homepage)
					}

					// Docker isolation info
					if isolated, ok := srv["docker_isolated"].(bool); ok && isolated {
						serverInfo += "\n- Docker Isolation: Enabled"
						if image, ok := srv["docker_image"].(string); ok && image != "" {
							serverInfo += fmt.Sprintf("\n- Docker Image: %s", image)
						}
					}

					// Quarantine status
					if quarantined, ok := srv["quarantined"].(bool); ok && quarantined {
						serverInfo += "\n- ⚠️  QUARANTINED: Server is quarantined for security review"
					}

					break
				}
			}
		}

		// Get available tools for this server with detailed status check
		tools, err := a.serverManager.GetServerTools(serverName)
		if err == nil && len(tools) > 0 {
			serverInfo += fmt.Sprintf("\n\n=== Available Tools (%d) ===", len(tools))
			for i, tool := range tools {
				if i >= 10 {
					serverInfo += fmt.Sprintf("\n... and %d more tools", len(tools)-10)
					break
				}
				if name, ok := tool["name"].(string); ok {
					toolDesc := ""
					if desc, ok := tool["description"].(string); ok && desc != "" {
						toolDesc = fmt.Sprintf(" - %s", truncateString(desc, 100))
					}
					serverInfo += fmt.Sprintf("\n  %d. %s%s", i+1, name, toolDesc)
				}
			}
		} else {
			// Provide detailed reason why tools are not available
			serverInfo += "\n\n=== Tool Status ==="

			// Check server configuration for detailed diagnostics
			if servers, err := a.serverManager.GetAllServers(); err == nil {
				for _, srv := range servers {
					if name, ok := srv["name"].(string); ok && name == serverName {
						enabled, _ := srv["enabled"].(bool)
						connected, _ := srv["connected"].(bool)
						quarantined, _ := srv["quarantined"].(bool)
						connecting, _ := srv["connecting"].(bool)

						if quarantined {
							serverInfo += "\n⚠️  Tools unavailable: Server is QUARANTINED"
							serverInfo += "\n   → Action required: Unquarantine the server from the tray menu to access tools"
						} else if !enabled {
							serverInfo += "\n⚠️  Tools unavailable: Server is DISABLED"
							serverInfo += "\n   → Action required: Enable the server from the tray menu to access tools"
						} else if connecting {
							serverInfo += "\n⏳ Tools unavailable: Server is currently CONNECTING"
							serverInfo += "\n   → Please wait for the connection to complete"
						} else if !connected {
							serverInfo += "\n⚠️  Tools unavailable: Server is NOT CONNECTED"
							serverInfo += "\n   → Possible reasons:"
							serverInfo += "\n      - Server configuration issues"
							serverInfo += "\n      - Network connectivity problems"
							serverInfo += "\n      - Server process not running"
							if lastErr, ok := srv["last_error"].(string); ok && lastErr != "" {
								serverInfo += fmt.Sprintf("\n      - Last error: %s", lastErr)
							}
							serverInfo += "\n   → Try restarting the server from the tray menu"
						} else {
							// Server is enabled and connected but still no tools
							serverInfo += fmt.Sprintf("\n⚠️  Tools unavailable: %v", err)
							serverInfo += "\n   → The server appears to be connected but tool retrieval failed"
							serverInfo += "\n   → This might indicate an MCP protocol issue with the server"
						}
						break
					}
				}
			} else if err != nil {
				serverInfo += fmt.Sprintf("\n⚠️  Could not retrieve tools: %v", err)
				serverInfo += "\n   → Unable to determine server status"
			}
		}
	}

	// Add available resources information
	resourcesInfo := "\n\n=== Available Resources ==="
	if a.serverManager != nil {
		logDir := a.serverManager.GetLogDir()
		if logDir != "" {
			resourcesInfo += fmt.Sprintf("\n📄 Server Log File: %s/server-%s.log", logDir, serverName)
			resourcesInfo += "\n   You can request to read this log file to diagnose issues"
		}

		configPath := a.serverManager.GetConfigPath()
		if configPath != "" {
			resourcesInfo += fmt.Sprintf("\n⚙️ Configuration File: %s", configPath)
			resourcesInfo += "\n   You can request to read this config file to understand server setup"
		}

		// Check if there's a repository URL
		githubURL := ""
		if servers, err := a.serverManager.GetAllServers(); err == nil {
			for _, srv := range servers {
				if name, ok := srv["name"].(string); ok && name == serverName {
					if repoURL, ok := srv["repository_url"].(string); ok && repoURL != "" {
						githubURL = repoURL
					}
					break
				}
			}
		}
		if githubURL == "" {
			githubURL = a.serverManager.GetGitHubURL()
		}
		if githubURL != "" {
			resourcesInfo += fmt.Sprintf("\n🔗 GitHub Repository: %s", githubURL)
			resourcesInfo += "\n   You can request to fetch and analyze the README or documentation"
		}
	}

	prompt := fmt.Sprintf(`You are an expert diagnostic agent for MCP (Model Context Protocol) servers.

Your capabilities include:
1. Configuration Analysis and Updates - Analyze and fix server configurations
2. Log Analysis - Diagnose issues from logs and error messages
3. Documentation Analysis - Help understand and implement server features from GitHub
4. Installation Help - Guide users through server setup and installation
5. Testing Assistance - Help test server functionality and troubleshoot issues
6. Server Management - Start, stop, enable, disable servers
7. Tool Testing - Test individual tools with appropriate parameters
8. General Troubleshooting - Solve any MCP server-related problems

=== Context ===
Current Server: %s%s%s

User Question: %s%s

=== Instructions ===
Please provide a clear, actionable response that:
- Directly addresses the user's question
- Analyzes server configuration and status if relevant
- Identifies potential issues from error messages or status
- Provides specific steps or solutions when applicable
- Includes relevant code examples or configuration snippets when helpful
- Suggests tool tests with example parameters if testing is needed
- References GitHub repository documentation when available
- Recommends next steps or follow-up actions

IMPORTANT: You have direct access to the following tools that you can call autonomously:

Available Tools:
- read_config: Read the mcp_config.json configuration file to analyze current server setup
- write_config: Write/update the mcp_config.json file (automatically creates backups and reloads configuration)
- read_log: Read server log files to view MCP communication, errors, and diagnostic information
- read_github: Fetch documentation or README from GitHub repository URLs

You can and should use these tools directly whenever needed. DO NOT suggest that the user should manually read files or execute commands.

When diagnosing issues or providing assistance:
1. Use read_log tool to investigate MCP communication issues and view error details
2. Use read_config tool to analyze current configuration and identify setup problems
3. Use write_config tool to fix configuration issues (automatic backup and reload)
4. Use read_github tool to fetch repository documentation and understand server requirements
5. Always use tools proactively - don't ask the user to do things you can do yourself

Example: When asked "Can you read the config?", immediately call the read_config tool instead of explaining how the user could read it manually.

If the server has errors:
1. Explain what the error means
2. Identify the root cause based on configuration and status
3. Provide step-by-step fix instructions
4. Suggest verification steps after fix

If testing tools:
1. Explain what the tool does
2. Suggest appropriate test parameters
3. Explain expected results
4. Help interpret actual results`,
		serverName,
		serverInfo,
		resourcesInfo,
		userMessage,
		conversationContext)

	return prompt
}

// GetCapabilities returns all capabilities of the LLM agent
func (a *LLMAgent) GetCapabilities() []string {
	return []string{
		"Configuration analysis and updates",
		"Log analysis and troubleshooting",
		"Documentation analysis and guidance",
		"Installation and setup assistance",
		"Server testing and validation",
		"General MCP server diagnostics",
		"Code generation and examples",
		"Best practices and recommendations",
	}
}

// GetAgentType returns the agent type
func (a *LLMAgent) GetAgentType() AgentType {
	return AgentType("llm")
}

// CanHandle determines if this agent can handle a message (always true for LLM agent)
func (a *LLMAgent) CanHandle(message ChatMessage) bool {
	// LLM agent can handle any message
	return true
}

// truncateString truncates a string to maxLength and adds ellipsis if needed
func truncateString(s string, maxLength int) string {
	if len(s) <= maxLength {
		return s
	}
	return s[:maxLength] + "..."
}

// getTools returns the tool definitions for OpenAI function calling
func (a *LLMAgent) getTools() []Tool {
	return []Tool{
		{
			Type: "function",
			Function: ToolFunction{
				Name:        "read_config",
				Description: "Read the MCP server configuration file (mcp_config.json) to analyze current settings and identify configuration issues",
				Parameters: map[string]interface{}{
					"type":       "object",
					"properties": map[string]interface{}{},
					"required":   []string{},
				},
			},
		},
		{
			Type: "function",
			Function: ToolFunction{
				Name:        "write_config",
				Description: "Write/update the MCP server configuration file with corrected settings. Automatically creates backups and reloads configuration.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"content": map[string]interface{}{
							"type":        "string",
							"description": "Complete JSON configuration content to write to mcp_config.json",
						},
					},
					"required": []string{"content"},
				},
			},
		},
		{
			Type: "function",
			Function: ToolFunction{
				Name:        "read_log",
				Description: "Read the server log file to view MCP communication, errors, and diagnostic information",
				Parameters: map[string]interface{}{
					"type":       "object",
					"properties": map[string]interface{}{},
					"required":   []string{},
				},
			},
		},
		{
			Type: "function",
			Function: ToolFunction{
				Name:        "read_github",
				Description: "Fetch documentation or README from a GitHub repository URL to understand server requirements and features",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"url": map[string]interface{}{
							"type":        "string",
							"description": "GitHub repository URL to fetch documentation from",
						},
					},
					"required": []string{"url"},
				},
			},
		},
		{
			Type: "function",
			Function: ToolFunction{
				Name:        "restart_server",
				Description: "Restart a specific MCP server by disabling and re-enabling it. This will reconnect the server and reload its configuration.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"server_name": map[string]interface{}{
							"type":        "string",
							"description": "Name of the server to restart",
						},
					},
					"required": []string{"server_name"},
				},
			},
		},
		{
			Type: "function",
			Function: ToolFunction{
				Name:        "call_tool",
				Description: "Call a specific tool on an MCP server with the given arguments. This allows testing server tools and executing operations. All MCP communication is automatically saved as context.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"server_name": map[string]interface{}{
							"type":        "string",
							"description": "Name of the MCP server",
						},
						"tool_name": map[string]interface{}{
							"type":        "string",
							"description": "Name of the tool to call (without server prefix)",
						},
						"arguments": map[string]interface{}{
							"type":        "object",
							"description": "Arguments to pass to the tool",
						},
					},
					"required": []string{"server_name", "tool_name", "arguments"},
				},
			},
		},
		{
			Type: "function",
			Function: ToolFunction{
				Name:        "get_server_status",
				Description: "Get detailed status information about a specific MCP server including connection state, enabled status, tools count, errors, and configuration details.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"server_name": map[string]interface{}{
							"type":        "string",
							"description": "Name of the server to query",
						},
					},
					"required": []string{"server_name"},
				},
			},
		},
		{
			Type: "function",
			Function: ToolFunction{
				Name:        "test_server_tools",
				Description: "Automatically analyze and test all tools from a specific MCP server. This generates test cases based on each tool's schema, executes them, and provides a comprehensive test report with success rates. Use 'simple' mode for basic testing or 'comprehensive' mode for thorough testing.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"server_name": map[string]interface{}{
							"type":        "string",
							"description": "Name of the MCP server to test",
						},
						"test_mode": map[string]interface{}{
							"type":        "string",
							"description": "Test mode: 'simple' for basic tests, 'comprehensive' for thorough tests",
							"enum":        []string{"simple", "comprehensive"},
							"default":     "simple",
						},
					},
					"required": []string{"server_name"},
				},
			},
		},
		{
			Type: "function",
			Function: ToolFunction{
				Name:        "list_all_servers",
				Description: "Get a comprehensive list of all configured MCP servers with their status, connection state, tool counts, and configuration details. This provides a complete overview of the entire MCP server ecosystem.",
				Parameters: map[string]interface{}{
					"type":       "object",
					"properties": map[string]interface{}{},
					"required":   []string{},
				},
			},
		},
		{
			Type: "function",
			Function: ToolFunction{
				Name:        "list_all_tools",
				Description: "Get a comprehensive list of all available tools from all MCP servers. This shows which tools are available from each server, including their descriptions and input schemas. Only includes tools from enabled and connected servers.",
				Parameters: map[string]interface{}{
					"type":       "object",
					"properties": map[string]interface{}{},
					"required":   []string{},
				},
			},
		},
	}
}

// createToolExecutor creates a function that executes tool calls by making HTTP requests to the endpoints
func (a *LLMAgent) createToolExecutor(serverName string) ToolExecutor {
	return func(toolName string, arguments map[string]interface{}) (string, error) {
		a.logger.Info("Executing tool",
			zap.String("tool", toolName),
			zap.Any("arguments", arguments))

		// Determine the endpoint based on tool name
		var endpoint string
		var requestBody interface{}

		switch toolName {
		case "read_config":
			endpoint = "http://localhost:8080/chat/read-config"
			requestBody = map[string]interface{}{}

		case "write_config":
			endpoint = "http://localhost:8080/chat/write-config"
			content, ok := arguments["content"].(string)
			if !ok {
				return "", fmt.Errorf("write_config requires 'content' parameter")
			}
			requestBody = map[string]interface{}{
				"content": content,
			}

		case "read_log":
			endpoint = "http://localhost:8080/chat/read-log"
			requestBody = map[string]interface{}{}

		case "read_github":
			endpoint = "http://localhost:8080/chat/read-github"
			url, ok := arguments["url"].(string)
			if !ok {
				return "", fmt.Errorf("read_github requires 'url' parameter")
			}
			requestBody = map[string]interface{}{
				"url": url,
			}

		case "restart_server":
			endpoint = "http://localhost:8080/chat/restart-server"
			serverNameParam, ok := arguments["server_name"].(string)
			if !ok {
				return "", fmt.Errorf("restart_server requires 'server_name' parameter")
			}
			requestBody = map[string]interface{}{
				"server_name": serverNameParam,
			}

		case "call_tool":
			endpoint = "http://localhost:8080/chat/call-tool"
			serverNameParam, ok1 := arguments["server_name"].(string)
			toolNameParam, ok2 := arguments["tool_name"].(string)
			toolArguments, ok3 := arguments["arguments"].(map[string]interface{})
			if !ok1 || !ok2 || !ok3 {
				return "", fmt.Errorf("call_tool requires 'server_name', 'tool_name', and 'arguments' parameters")
			}
			requestBody = map[string]interface{}{
				"server_name": serverNameParam,
				"tool_name":   toolNameParam,
				"arguments":   toolArguments,
			}

		case "get_server_status":
			endpoint = "http://localhost:8080/chat/get-server-status"
			serverNameParam, ok := arguments["server_name"].(string)
			if !ok {
				return "", fmt.Errorf("get_server_status requires 'server_name' parameter")
			}
			requestBody = map[string]interface{}{
				"server_name": serverNameParam,
			}

		case "test_server_tools":
			endpoint = "http://localhost:8080/chat/test-server-tools"
			serverNameParam, ok := arguments["server_name"].(string)
			if !ok {
				return "", fmt.Errorf("test_server_tools requires 'server_name' parameter")
			}

			// Get test_mode parameter (optional, defaults to "simple")
			testMode := "simple"
			if mode, ok := arguments["test_mode"].(string); ok {
				testMode = mode
			}

			requestBody = map[string]interface{}{
				"server_name": serverNameParam,
				"test_mode":   testMode,
			}

		case "list_all_servers":
			endpoint = "http://localhost:8080/chat/list-all-servers"
			requestBody = map[string]interface{}{}

		case "list_all_tools":
			endpoint = "http://localhost:8080/chat/list-all-tools"
			requestBody = map[string]interface{}{}

		default:
			return "", fmt.Errorf("unknown tool: %s", toolName)
		}

		// Make HTTP request to endpoint
		jsonData, err := json.Marshal(requestBody)
		if err != nil {
			return "", fmt.Errorf("failed to marshal request: %w", err)
		}

		resp, err := http.Post(endpoint, "application/json", bytes.NewBuffer(jsonData))
		if err != nil {
			return "", fmt.Errorf("failed to call endpoint: %w", err)
		}
		defer resp.Body.Close()

		// Read response
		var result map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return "", fmt.Errorf("failed to decode response: %w", err)
		}

		// Check for success
		success, _ := result["success"].(bool)
		if !success {
			errorMsg, _ := result["error"].(string)
			return "", fmt.Errorf("tool execution failed: %s", errorMsg)
		}

		// Extract and return the content
		if content, ok := result["content"].(string); ok {
			return content, nil
		}

		// For write_config, return the success message
		if message, ok := result["message"].(string); ok {
			return message, nil
		}

		return "Tool executed successfully", nil
	}
}

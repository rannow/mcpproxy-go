//go:build !nogui && !headless && !linux

package tray

import (
	"context"
	"fmt"
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
	}
}

// NewLLMAgent creates a new LLM-powered agent
func NewLLMAgent(logger *zap.Logger, serverManager interface {
	GetServerTools(serverName string) ([]map[string]interface{}, error)
	EnableServer(serverName string, enabled bool) error
	GetAllServers() ([]map[string]interface{}, error)
	ReloadConfiguration() error
	GetConfigPath() string
}) *LLMAgent {
	return &LLMAgent{
		logger:        logger,
		llmClient:     NewOpenAIClient(),
		serverManager: serverManager,
	}
}

// ProcessMessage processes a user message using LLM
func (a *LLMAgent) ProcessMessage(ctx context.Context, message ChatMessage, session *ChatSession) (*ChatMessage, error) {
	a.logger.Info("LLM agent processing message",
		zap.String("session_id", session.ID),
		zap.String("server", session.ServerName),
		zap.String("content_preview", truncateString(message.Content, 50)))

	// Build context from chat history
	conversationContext := a.buildConversationContext(session)

	// Create comprehensive prompt with server context
	prompt := a.buildPrompt(message.Content, session.ServerName, conversationContext)

	// Get LLM response
	response, err := a.llmClient.Analyze(prompt)
	if err != nil {
		a.logger.Error("LLM analysis failed", zap.Error(err))
		return nil, fmt.Errorf("LLM analysis failed: %w", err)
	}

	// Create response message
	responseMsg := &ChatMessage{
		ID:        generateMessageID(),
		Role:      "assistant",
		Content:   response,
		AgentType: "llm",
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"model": "gpt-4o-mini",
		},
	}

	return responseMsg, nil
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
Current Server: %s%s

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

//go:build !nogui && !headless && !linux

package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"
)

// MCP Communication structure to capture protocol messages
type MCPCommunication struct {
	Timestamp  string                 `json:"timestamp"`
	Direction  string                 `json:"direction"` // "request" or "response"
	Server     string                 `json:"server"`
	Tool       string                 `json:"tool"`
	Request    map[string]interface{} `json:"request,omitempty"`
	Response   interface{}            `json:"response,omitempty"`
	Error      string                 `json:"error,omitempty"`
}

// OpenAI Function Calling structures
type openAITool struct {
	Type     string               `json:"type"`
	Function openAIToolFunction   `json:"function"`
}

type openAIToolFunction struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

type openAIToolCall struct {
	ID       string                     `json:"id"`
	Type     string                     `json:"type"`
	Function openAIToolCallFunction    `json:"function"`
}

type openAIToolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// OpenAI Request with tools support
type openAIRequestWithTools struct {
	Model       string          `json:"model"`
	Messages    []openAIMessage `json:"messages"`
	Temperature float64         `json:"temperature"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	Tools       []openAITool    `json:"tools,omitempty"`
}

type openAIMessage struct {
	Role       string           `json:"role"`
	Content    string           `json:"content,omitempty"`
	ToolCalls  []openAIToolCall `json:"tool_calls,omitempty"`
	ToolCallID string           `json:"tool_call_id,omitempty"`
}

type openAIResponseWithTools struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int           `json:"index"`
		Message openAIMessage `json:"message"`
		Finish  string        `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error,omitempty"`
}

// getTools returns tool definitions for OpenAI (built-in tools only)
func (s *Server) getTools() []openAITool {
	return s.getToolsForServer("")
}

// getToolsForServer returns tool definitions including server-specific tools
func (s *Server) getToolsForServer(serverName string) []openAITool {
	tools := []openAITool{
		{
			Type: "function",
			Function: openAIToolFunction{
				Name:        "read_config",
				Description: "Read the mcp_config.json configuration file to analyze current server setup and configuration",
				Parameters: map[string]interface{}{
					"type":       "object",
					"properties": map[string]interface{}{},
					"required":   []string{},
				},
			},
		},
		{
			Type: "function",
			Function: openAIToolFunction{
				Name:        "write_config",
				Description: "Write/update the mcp_config.json file with corrected settings. Automatically creates backups and reloads configuration.",
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
			Function: openAIToolFunction{
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
			Function: openAIToolFunction{
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
	}

	// If a server name is specified, add server-specific tools
	if serverName != "" {
		client, exists := s.upstreamManager.GetClient(serverName)
		if exists && client != nil {
			// Get tools from the upstream server
			serverTools, err := client.ListTools(context.Background())
			if err == nil && len(serverTools) > 0 {
				s.logger.Info("Adding server-specific tools to chat",
					zap.String("server", serverName),
					zap.Int("tool_count", len(serverTools)))

				// Convert MCP tools to OpenAI format
				for _, tool := range serverTools {
					// Convert input schema to OpenAI parameters format
					params := map[string]interface{}{
						"type":       "object",
						"properties": map[string]interface{}{},
						"required":   []string{},
					}

					// Parse the ParamsJSON back to a schema map
					if tool.ParamsJSON != "" {
						var inputSchema map[string]interface{}
						if err := json.Unmarshal([]byte(tool.ParamsJSON), &inputSchema); err == nil {
							// Extract properties and required fields from input schema
							if props, ok := inputSchema["properties"].(map[string]interface{}); ok {
								params["properties"] = props
							}
							if req, ok := inputSchema["required"].([]interface{}); ok {
								requiredFields := make([]string, 0, len(req))
								for _, r := range req {
									if str, ok := r.(string); ok {
										requiredFields = append(requiredFields, str)
									}
								}
								params["required"] = requiredFields
							}
						}
					}

					// Add server prefix to tool name to avoid conflicts
					// Use underscore instead of colon to comply with OpenAI pattern ^[a-zA-Z0-9_-]+$
					toolName := serverName + "_" + tool.Name

					tools = append(tools, openAITool{
						Type: "function",
						Function: openAIToolFunction{
							Name:        toolName,
							Description: tool.Description,
							Parameters:  params,
						},
					})
				}
			}
		}
	}

	return tools
}

// executeTool executes a tool call by making HTTP request to the endpoint
// Returns: result string, MCP communication details, error
func (s *Server) executeTool(toolName string, arguments map[string]interface{}) (string, *MCPCommunication, error) {
	s.logger.Info("Executing tool",
		zap.String("tool", toolName),
		zap.Any("arguments", arguments))

	timestamp := time.Now().Format(time.RFC3339)

	// Check if this is a server-specific tool (format: "server_tool")
	if len(toolName) > 0 && toolName != "read_config" && toolName != "write_config" && toolName != "read_log" && toolName != "read_github" {
		// Try to parse as server_tool format
		// Since server names can contain underscores (e.g., "applescript_execute"),
		// we find the first underscore and check if the prefix is a known server
		underscoreIndex := strings.Index(toolName, "_")

		if underscoreIndex > 0 {
			serverName := toolName[:underscoreIndex]
			actualToolName := toolName[underscoreIndex+1:]

			s.logger.Info("Executing server-specific tool",
				zap.String("server", serverName),
				zap.String("tool", actualToolName))

			// Get client for the server
			client, exists := s.upstreamManager.GetClient(serverName)
			if !exists || client == nil {
				mcpComm := &MCPCommunication{
					Timestamp: timestamp,
					Direction: "request",
					Server:    serverName,
					Tool:      actualToolName,
					Request:   arguments,
					Error:     fmt.Sprintf("server not found: %s", serverName),
				}
				return "", mcpComm, fmt.Errorf("server not found: %s", serverName)
			}

			// Capture MCP request
			mcpComm := &MCPCommunication{
				Timestamp: timestamp,
				Direction: "request",
				Server:    serverName,
				Tool:      actualToolName,
				Request:   arguments,
			}

			// Call the tool on the upstream server
			result, err := client.CallTool(context.Background(), actualToolName, arguments)
			if err != nil {
				mcpComm.Error = err.Error()
				return "", mcpComm, fmt.Errorf("failed to call server tool: %w", err)
			}

			// Capture MCP response
			mcpComm.Direction = "request-response"
			mcpComm.Response = result.Content

			// Format the result as a string
			if len(result.Content) > 0 {
				// Convert MCP content to JSON representation
				jsonResult, err := json.MarshalIndent(result.Content, "", "  ")
				if err == nil && len(jsonResult) > 0 {
					return string(jsonResult), mcpComm, nil
				}
			}

			return "Tool executed successfully with no output", mcpComm, nil
		}
	}

	// Built-in tools
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
			return "", nil, fmt.Errorf("write_config requires 'content' parameter")
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
			return "", nil, fmt.Errorf("read_github requires 'url' parameter")
		}
		requestBody = map[string]interface{}{
			"url": url,
		}

	default:
		return "", nil, fmt.Errorf("unknown tool: %s", toolName)
	}

	// Make HTTP request to endpoint
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := http.Post(endpoint, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", nil, fmt.Errorf("failed to call endpoint: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Check for success
	success, _ := result["success"].(bool)
	if !success {
		errorMsg, _ := result["error"].(string)
		return "", nil, fmt.Errorf("tool execution failed: %s", errorMsg)
	}

	// Extract and return the content
	if content, ok := result["content"].(string); ok {
		return content, nil, nil
	}

	// For write_config, return the success message
	if message, ok := result["message"].(string); ok {
		return message, nil, nil
	}

	return "Tool executed successfully", nil, nil
}

// callOpenAIWithTools makes a request to OpenAI API with tools support
// Returns: response string, MCP communications, error
func (s *Server) callOpenAIWithTools(apiKey string, messages []chatMessage, serverName string) (string, []MCPCommunication, error) {
	// Convert chatMessage to openAIMessage
	openAIMessages := make([]openAIMessage, len(messages))
	for i, msg := range messages {
		openAIMessages[i] = openAIMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	// Collect all MCP communications
	var mcpCommunications []MCPCommunication

	tools := s.getToolsForServer(serverName)
	maxIterations := 10 // Increased from 5 to handle complex tool chains

	for i := 0; i < maxIterations; i++ {
		// Prepare request
		request := openAIRequestWithTools{
			Model:       "gpt-4o-mini",
			Messages:    openAIMessages,
			Temperature: 0.7,
			MaxTokens:   2000,
			Tools:       tools,
		}

		// Marshal request to JSON
		jsonData, err := json.Marshal(request)
		if err != nil {
			return "", mcpCommunications, fmt.Errorf("failed to marshal request: %w", err)
		}

		// Create HTTP request with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonData))
		if err != nil {
			return "", mcpCommunications, fmt.Errorf("failed to create request: %w", err)
		}

		// Set headers
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))

		// Send request
		client := &http.Client{
			Timeout: 60 * time.Second,
		}

		resp, err := client.Do(req)
		if err != nil {
			return "", mcpCommunications, fmt.Errorf("request failed: %w", err)
		}
		defer resp.Body.Close()

		// Read response body
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", mcpCommunications, fmt.Errorf("failed to read response: %w", err)
		}

		// Check for non-200 status codes
		if resp.StatusCode != http.StatusOK {
			return "", mcpCommunications, fmt.Errorf("OpenAI API error (status %d): %s", resp.StatusCode, string(body))
		}

		// Parse response
		var openAIResp openAIResponseWithTools
		if err := json.Unmarshal(body, &openAIResp); err != nil {
			return "", mcpCommunications, fmt.Errorf("failed to parse response: %w", err)
		}

		// Check for API errors
		if openAIResp.Error != nil {
			return "", mcpCommunications, fmt.Errorf("OpenAI API error: %s (type: %s, code: %s)",
				openAIResp.Error.Message,
				openAIResp.Error.Type,
				openAIResp.Error.Code)
		}

		// Extract response
		if len(openAIResp.Choices) == 0 {
			return "", mcpCommunications, fmt.Errorf("no response choices from OpenAI")
		}

		choice := openAIResp.Choices[0]

		// Log token usage
		s.logger.Debug("OpenAI API call completed",
			zap.String("model", openAIResp.Model),
			zap.Int("prompt_tokens", openAIResp.Usage.PromptTokens),
			zap.Int("completion_tokens", openAIResp.Usage.CompletionTokens),
			zap.Int("total_tokens", openAIResp.Usage.TotalTokens),
			zap.String("finish_reason", choice.Finish))

		// Check finish reason
		if choice.Finish == "stop" {
			return choice.Message.Content, mcpCommunications, nil
		}

		if choice.Finish == "tool_calls" {
			// Add assistant message with tool calls to history
			openAIMessages = append(openAIMessages, choice.Message)

			// Execute each tool call
			for _, toolCall := range choice.Message.ToolCalls {
				// Log raw arguments for debugging
				s.logger.Info("Parsing tool call arguments",
					zap.String("tool", toolCall.Function.Name),
					zap.String("arguments_raw", toolCall.Function.Arguments),
					zap.Int("arguments_length", len(toolCall.Function.Arguments)))

				// Parse tool arguments
				var args map[string]interface{}
				if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
					s.logger.Error("Failed to parse tool arguments",
						zap.String("tool", toolCall.Function.Name),
						zap.String("arguments", toolCall.Function.Arguments),
						zap.Error(err))
					return "", mcpCommunications, fmt.Errorf("failed to parse tool arguments: %w", err)
				}

				// Execute the tool and capture MCP communication
				result, mcpComm, err := s.executeTool(toolCall.Function.Name, args)
				if err != nil {
					result = fmt.Sprintf("Error executing tool: %v", err)
				}

				// Add MCP communication if available
				if mcpComm != nil {
					mcpCommunications = append(mcpCommunications, *mcpComm)
				}

				// Add tool result to messages
				openAIMessages = append(openAIMessages, openAIMessage{
					Role:       "tool",
					Content:    result,
					ToolCallID: toolCall.ID,
				})
			}

			// Continue to next iteration with tool results
			continue
		}

		if choice.Finish == "length" {
			// Response was truncated due to max_tokens limit
			s.logger.Warn("AI response truncated due to max_tokens limit",
				zap.Int("prompt_tokens", openAIResp.Usage.PromptTokens),
				zap.Int("completion_tokens", openAIResp.Usage.CompletionTokens),
				zap.Int("total_tokens", openAIResp.Usage.TotalTokens))

			// If there are tool calls despite truncation, execute them
			if len(choice.Message.ToolCalls) > 0 {
				s.logger.Info("Executing tool calls despite truncation",
					zap.Int("tool_call_count", len(choice.Message.ToolCalls)))

				// Add assistant message with tool calls to history
				openAIMessages = append(openAIMessages, choice.Message)

				// Execute each tool call
				for _, toolCall := range choice.Message.ToolCalls {
					// Log raw arguments for debugging
					s.logger.Info("Parsing tool call arguments (truncated response)",
						zap.String("tool", toolCall.Function.Name),
						zap.String("arguments_raw", toolCall.Function.Arguments),
						zap.Int("arguments_length", len(toolCall.Function.Arguments)))

					// Parse tool arguments
					var args map[string]interface{}
					if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
						s.logger.Error("Failed to parse tool arguments (truncated response)",
							zap.String("tool", toolCall.Function.Name),
							zap.String("arguments", toolCall.Function.Arguments),
							zap.Error(err))
						return "", mcpCommunications, fmt.Errorf("failed to parse tool arguments: %w", err)
					}

					// Execute the tool and capture MCP communication
					result, mcpComm, err := s.executeTool(toolCall.Function.Name, args)
					if err != nil {
						result = fmt.Sprintf("Error executing tool: %v", err)
					}

					// Add MCP communication if available
					if mcpComm != nil {
						mcpCommunications = append(mcpCommunications, *mcpComm)
					}

					// Add tool result to messages
					openAIMessages = append(openAIMessages, openAIMessage{
						Role:       "tool",
						Content:    result,
						ToolCallID: toolCall.ID,
					})
				}

				// Continue to next iteration with tool results
				continue
			}

			// If there's content but no tool calls, return it with warning
			if choice.Message.Content != "" {
				// Add a note about truncation to the response
				truncatedContent := choice.Message.Content + "\n\n[Note: Response was truncated due to length limits. Please ask me to continue if you need more information.]"
				return truncatedContent, mcpCommunications, nil
			}

			return "", mcpCommunications, fmt.Errorf("response truncated and no content available")
		}

		// If we get here with content, return it
		if choice.Message.Content != "" {
			return choice.Message.Content, mcpCommunications, nil
		}

		return "", mcpCommunications, fmt.Errorf("unexpected finish reason: %s", choice.Finish)
	}

	// If we reached max iterations, return a helpful message with the context
	s.logger.Warn("Maximum tool call iterations reached",
		zap.Int("iterations", maxIterations),
		zap.String("server", serverName))

	// Try to extract any partial content from the last message
	lastContent := ""
	if len(openAIMessages) > 0 {
		lastMsg := openAIMessages[len(openAIMessages)-1]
		if lastMsg.Role == "assistant" && lastMsg.Content != "" {
			lastContent = lastMsg.Content
		}
	}

	if lastContent != "" {
		return lastContent + "\n\n[Note: Response incomplete - maximum tool call iterations reached. The task may require simplification or breaking into smaller steps.]", mcpCommunications, nil
	}

	return "", mcpCommunications, fmt.Errorf("maximum tool call iterations reached after %d attempts without producing a final response", maxIterations)
}

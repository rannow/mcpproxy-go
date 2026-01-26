//go:build !nogui && !headless && !linux

package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"mcpproxy-go/internal/config"

	"go.uber.org/zap"
)

// handleChatReadConfig reads the MCP configuration file
func (s *Server) handleChatReadConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	configPath := s.GetConfigPath()
	if configPath == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Config path not available",
		})
		return
	}

	content, err := os.ReadFile(configPath)
	if err != nil {
		s.logger.Error("Failed to read config file",
			zap.String("path", configPath),
			zap.Error(err))
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Failed to read config: %v", err),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"content": string(content),
		"path":    configPath,
	})
}

// handleChatWriteConfig writes the MCP configuration file
func (s *Server) handleChatWriteConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		Content string `json:"content"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		s.logger.Error("Failed to decode write-config request", zap.Error(err))
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Invalid request format",
		})
		return
	}

	if request.Content == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Config content is required",
		})
		return
	}

	configPath := s.GetConfigPath()
	if configPath == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Config path not available",
		})
		return
	}

	// Parse and validate the new configuration
	var newConfig config.Config
	if err := json.Unmarshal([]byte(request.Content), &newConfig); err != nil {
		s.logger.Error("Failed to parse config JSON", zap.Error(err))
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Invalid JSON format: %v", err),
		})
		return
	}

	// Save configuration (automatically creates backup)
	if err := config.SaveConfig(&newConfig, configPath); err != nil {
		s.logger.Error("Failed to save config file",
			zap.String("path", configPath),
			zap.Error(err))
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Failed to save config: %v", err),
		})
		return
	}

	s.logger.Info("Config file saved successfully", zap.String("path", configPath))

	// Trigger configuration reload
	if err := s.ReloadConfiguration(); err != nil {
		s.logger.Warn("Failed to reload configuration after save", zap.Error(err))
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"path":    configPath,
			"warning": fmt.Sprintf("Config saved but reload failed: %v", err),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"path":    configPath,
		"message": "Configuration saved and reloaded successfully",
	})
}

// handleChatReadLog reads the server log file
func (s *Server) handleChatReadLog(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	logDir := s.GetLogDir()
	if logDir == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Log directory not available",
		})
		return
	}

	// Read main.log
	logPath := fmt.Sprintf("%s/main.log", logDir)
	content, err := os.ReadFile(logPath)
	if err != nil {
		s.logger.Error("Failed to read log file",
			zap.String("path", logPath),
			zap.Error(err))
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Failed to read log: %v", err),
		})
		return
	}

	// Return last 200 lines (approximately 50KB)
	lines := strings.Split(string(content), "\n")
	if len(lines) > 200 {
		lines = lines[len(lines)-200:]
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"content": strings.Join(lines, "\n"),
		"path":    logPath,
	})
}

// handleChatReadGitHub fetches content from a GitHub URL
func (s *Server) handleChatReadGitHub(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		URL string `json:"url"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		s.logger.Error("Failed to decode read-github request", zap.Error(err))
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Invalid request format",
		})
		return
	}

	if request.URL == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "URL is required",
		})
		return
	}

	// Fetch content from GitHub
	resp, err := http.Get(request.URL)
	if err != nil {
		s.logger.Error("Failed to fetch GitHub content",
			zap.String("url", request.URL),
			zap.Error(err))
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Failed to fetch URL: %v", err),
		})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("GitHub returned status %d", resp.StatusCode),
		})
		return
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		s.logger.Error("Failed to read GitHub response",
			zap.String("url", request.URL),
			zap.Error(err))
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Failed to read response: %v", err),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"content": string(content),
		"url":     request.URL,
	})
}

// handleChatRestartServer restarts a specific MCP server
func (s *Server) handleChatRestartServer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		ServerName string `json:"server_name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		s.logger.Error("Failed to decode restart-server request", zap.Error(err))
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Invalid request format",
		})
		return
	}

	if request.ServerName == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Server name is required",
		})
		return
	}

	if err := s.RestartServer(request.ServerName); err != nil {
		s.logger.Error("Failed to restart server",
			zap.String("server", request.ServerName),
			zap.Error(err))
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Failed to restart server: %v", err),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Server '%s' restarted successfully", request.ServerName),
		"server":  request.ServerName,
	})
}

// handleChatCallTool calls a tool on a specific MCP server
func (s *Server) handleChatCallTool(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		ServerName string                 `json:"server_name"`
		ToolName   string                 `json:"tool_name"`
		Arguments  map[string]interface{} `json:"arguments"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		s.logger.Error("Failed to decode call-tool request", zap.Error(err))
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Invalid request format",
		})
		return
	}

	if request.ServerName == "" || request.ToolName == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Server name and tool name are required",
		})
		return
	}

	// MED-002: Use centralized timeout to prevent hanging API requests
	ctx, cancel := context.WithTimeout(r.Context(), config.ToolCallTimeout)
	defer cancel()

	var result interface{}
	var err error

	// Route MCPProxy internal tools to the built-in handler instead of upstream manager
	if request.ServerName == "MCPProxy" {
		result, err = s.mcpProxy.CallBuiltInTool(ctx, request.ToolName, request.Arguments)
	} else {
		// Construct full tool name with server prefix for upstream servers
		fullToolName := fmt.Sprintf("%s:%s", request.ServerName, request.ToolName)
		result, err = s.upstreamManager.CallTool(ctx, fullToolName, request.Arguments)
	}
	if err != nil {
		// Check if it was a timeout error for better error messaging
		errorMsg := fmt.Sprintf("Failed to call tool: %v", err)
		if ctx.Err() == context.DeadlineExceeded {
			errorMsg = fmt.Sprintf("Tool call timed out after %v: %v", config.ToolCallTimeout, err)
			s.logger.Warn("Tool call timed out",
				zap.String("server", request.ServerName),
				zap.String("tool", request.ToolName),
				zap.Duration("timeout", config.ToolCallTimeout),
				zap.Error(err))
		} else {
			s.logger.Error("Failed to call tool",
				zap.String("server", request.ServerName),
				zap.String("tool", request.ToolName),
				zap.Error(err))
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   errorMsg,
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"result":  result,
		"server":  request.ServerName,
		"tool":    request.ToolName,
	})
}

// handleChatGetServerStatus gets the status of a specific MCP server
func (s *Server) handleChatGetServerStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		ServerName string `json:"server_name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		s.logger.Error("Failed to decode get-server-status request", zap.Error(err))
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Invalid request format",
		})
		return
	}

	if request.ServerName == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Server name is required",
		})
		return
	}

	// Get all servers and find the requested one
	allServers, err := s.GetAllServers()
	if err != nil {
		s.logger.Error("Failed to get server list", zap.Error(err))
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Failed to get server list: %v", err),
		})
		return
	}

	for _, server := range allServers {
		if name, ok := server["name"].(string); ok && name == request.ServerName {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"status":  server,
			})
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": false,
		"error":   fmt.Sprintf("Server '%s' not found", request.ServerName),
	})
}

// handleChatTestServerTools analyzes and tests all tools from a specific MCP server
func (s *Server) handleChatTestServerTools(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		ServerName string `json:"server_name"`
		TestMode   string `json:"test_mode"` // "simple" or "comprehensive"
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		s.logger.Error("Failed to decode test-server-tools request", zap.Error(err))
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Invalid request format",
		})
		return
	}

	if request.ServerName == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Server name is required",
		})
		return
	}

	// Default to simple test mode
	if request.TestMode == "" {
		request.TestMode = "simple"
	}

	s.logger.Info("Starting tool test generation and execution",
		zap.String("server", request.ServerName),
		zap.String("test_mode", request.TestMode))

	// Get the client for the specified server
	client, exists := s.upstreamManager.GetClient(request.ServerName)
	if !exists {
		s.logger.Error("Server not found",
			zap.String("server", request.ServerName))
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Server '%s' not found", request.ServerName),
		})
		return
	}

	// Get all tools from the server
	// MED-002: Use centralized timeout to prevent hanging API requests
	listCtx, listCancel := context.WithTimeout(r.Context(), config.ListToolsTimeout)
	defer listCancel()
	tools, err := client.ListTools(listCtx)
	if err != nil {
		s.logger.Error("Failed to list tools from server",
			zap.String("server", request.ServerName),
			zap.Error(err))
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Failed to list tools: %v", err),
		})
		return
	}

	if len(tools) == 0 {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": fmt.Sprintf("Server '%s' has no tools to test", request.ServerName),
			"results":  []interface{}{},
		})
		return
	}

	s.logger.Info("Found tools to test",
		zap.String("server", request.ServerName),
		zap.Int("tool_count", len(tools)))

	// Generate and execute tests for each tool
	testResults := make([]map[string]interface{}, 0, len(tools))

	for _, tool := range tools {
		toolName := tool.Name
		s.logger.Debug("Testing tool",
			zap.String("server", request.ServerName),
			zap.String("tool", toolName))

		testResult := map[string]interface{}{
			"tool_name":   toolName,
			"description": tool.Description,
		}

		// Analyze tool schema to determine test cases
		testCases := s.generateTestCases(tool, request.TestMode)
		testResult["test_cases"] = len(testCases)

		// Execute test cases
		caseResults := make([]map[string]interface{}, 0, len(testCases))
		successCount := 0

		for i, testCase := range testCases {
			caseResult := map[string]interface{}{
				"case_number":  i + 1,
				"description":  testCase.Description,
				"arguments":    testCase.Arguments,
			}

			// Call the tool with test arguments
			// MED-002: Use centralized timeout for each tool call
			callCtx, callCancel := context.WithTimeout(r.Context(), config.ToolCallTimeout)
			fullToolName := fmt.Sprintf("%s:%s", request.ServerName, toolName)
			result, err := s.upstreamManager.CallTool(callCtx, fullToolName, testCase.Arguments)
			callCancel()

			if err != nil {
				caseResult["status"] = "error"
				if callCtx.Err() == context.DeadlineExceeded {
					caseResult["error"] = fmt.Sprintf("Tool call timed out after %v: %v", config.ToolCallTimeout, err)
				} else {
					caseResult["error"] = err.Error()
				}
			} else {
				caseResult["status"] = "success"
				caseResult["result"] = result
				successCount++
			}

			caseResults = append(caseResults, caseResult)
		}

		testResult["results"] = caseResults
		testResult["success_rate"] = fmt.Sprintf("%d/%d", successCount, len(testCases))
		testResults = append(testResults, testResult)
	}

	s.logger.Info("Completed tool testing",
		zap.String("server", request.ServerName),
		zap.Int("tools_tested", len(testResults)))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":      true,
		"server":       request.ServerName,
		"tools_tested": len(testResults),
		"results":      testResults,
	})
}

// TestCase represents a single test case for a tool
type TestCase struct {
	Description string
	Arguments   map[string]interface{}
}

// generateTestCases creates test cases based on tool schema
func (s *Server) generateTestCases(tool *config.ToolMetadata, testMode string) []TestCase {
	// Parse the ParamsJSON to get the schema
	if tool.ParamsJSON == "" {
		// Tool has no parameters, create a simple test
		return []TestCase{
			{
				Description: "No parameters required",
				Arguments:   map[string]interface{}{},
			},
		}
	}

	var inputSchema map[string]interface{}
	if err := json.Unmarshal([]byte(tool.ParamsJSON), &inputSchema); err != nil {
		s.logger.Warn("Failed to parse tool params JSON",
			zap.String("tool", tool.Name),
			zap.Error(err))
		return []TestCase{
			{
				Description: "Unable to parse schema, using empty arguments",
				Arguments:   map[string]interface{}{},
			},
		}
	}

	properties, hasProps := inputSchema["properties"].(map[string]interface{})
	required, _ := inputSchema["required"].([]interface{})

	if !hasProps || len(properties) == 0 {
		// No properties defined, test with empty arguments
		return []TestCase{
			{
				Description: "Empty arguments",
				Arguments:   map[string]interface{}{},
			},
		}
	}

	// Build required parameters map
	requiredMap := make(map[string]bool)
	for _, r := range required {
		if reqStr, ok := r.(string); ok {
			requiredMap[reqStr] = true
		}
	}

	testCases := []TestCase{}

	// Test Case 1: Minimal valid input (only required parameters)
	minimalArgs := make(map[string]interface{})
	for propName, propValue := range properties {
		if requiredMap[propName] {
			propMap, ok := propValue.(map[string]interface{})
			if ok {
				minimalArgs[propName] = s.getDefaultValue(propMap)
			}
		}
	}

	if len(minimalArgs) > 0 || len(requiredMap) == 0 {
		testCases = append(testCases, TestCase{
			Description: "Minimal required parameters",
			Arguments:   minimalArgs,
		})
	}

	// For comprehensive mode, add more test cases
	if testMode == "comprehensive" {
		// Test Case 2: All parameters with defaults
		allArgs := make(map[string]interface{})
		for propName, propValue := range properties {
			propMap, ok := propValue.(map[string]interface{})
			if ok {
				allArgs[propName] = s.getDefaultValue(propMap)
			}
		}

		if len(allArgs) > len(minimalArgs) {
			testCases = append(testCases, TestCase{
				Description: "All parameters with default values",
				Arguments:   allArgs,
			})
		}
	}

	return testCases
}

// getDefaultValue returns a sensible default value based on JSON schema type
func (s *Server) getDefaultValue(schema map[string]interface{}) interface{} {
	// Check for default value in schema
	if defaultVal, hasDefault := schema["default"]; hasDefault {
		return defaultVal
	}

	// Check for enum values
	if enumVals, hasEnum := schema["enum"].([]interface{}); hasEnum && len(enumVals) > 0 {
		return enumVals[0]
	}

	// Generate based on type
	typeVal, hasType := schema["type"].(string)
	if !hasType {
		return nil
	}

	switch typeVal {
	case "string":
		// Check if there's a pattern or example
		if example, hasExample := schema["example"].(string); hasExample {
			return example
		}
		return "test_value"
	case "number", "integer":
		if example, hasExample := schema["example"]; hasExample {
			return example
		}
		return 0
	case "boolean":
		return false
	case "array":
		return []interface{}{}
	case "object":
		return map[string]interface{}{}
	default:
		return nil
	}
}

// handleChatContext provides context statistics for the current chat session
func (s *Server) handleChatContext(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get session_id from query parameter
	sessionID := r.URL.Query().Get("session_id")
	if sessionID == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "session_id query parameter is required",
		})
		return
	}

	// Get the session messages from global sessions manager
	messages := sessions.getMessages(sessionID)
	if messages == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":            "Session not found",
			"total_messages":   0,
			"estimated_tokens": 0,
			"max_tokens":       128000,
			"target_tokens":    40000,
			"pruning_active":   false,
		})
		return
	}

	// Calculate context statistics
	totalMessages := len(messages)
	estimatedTokens := 0

	// Estimate tokens for all messages (conservative: 1 token ≈ 3 characters)
	for _, msg := range messages {
		estimatedTokens += len(msg.Content) / 3
	}

	// Check if pruning is active (estimate > target threshold)
	const maxTokens = 128000
	const targetTokens = 40000 // More aggressive pruning to avoid context overflow
	pruningActive := estimatedTokens > targetTokens

	// Build response
	response := map[string]interface{}{
		"total_messages":   totalMessages,
		"estimated_tokens": estimatedTokens,
		"max_tokens":       maxTokens,
		"target_tokens":    targetTokens,
		"pruning_active":   pruningActive,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleChatListAllServers returns a list of all configured MCP servers
func (s *Server) handleChatListAllServers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	servers, err := s.GetAllServers()
	if err != nil {
		s.logger.Error("Failed to get all servers", zap.Error(err))
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Failed to get servers: %v", err),
		})
		return
	}

	// Format server information for the AI agent
	serverList := make([]map[string]interface{}, 0, len(servers))
	for _, srv := range servers {
		serverInfo := map[string]interface{}{
			"name":     srv["name"],
			"enabled":  srv["enabled"],
			"protocol": srv["protocol"],
		}

		// Add connection details
		if url, ok := srv["url"].(string); ok && url != "" {
			serverInfo["url"] = url
		}
		if cmd, ok := srv["command"].(string); ok && cmd != "" {
			serverInfo["command"] = cmd
		}
		if args, ok := srv["args"].([]interface{}); ok && len(args) > 0 {
			serverInfo["args"] = args
		}
		if workingDir, ok := srv["working_dir"].(string); ok && workingDir != "" {
			serverInfo["working_dir"] = workingDir
		}

		// Add status information
		if connected, ok := srv["connected"].(bool); ok {
			serverInfo["connected"] = connected
		}
		if connecting, ok := srv["connecting"].(bool); ok {
			serverInfo["connecting"] = connecting
		}
		if quarantined, ok := srv["quarantined"].(bool); ok {
			serverInfo["quarantined"] = quarantined
		}
		if sleeping, ok := srv["sleeping"].(bool); ok {
			serverInfo["sleeping"] = sleeping
		}

		// Add tool count
		if toolCount, ok := srv["tool_count"].(int); ok {
			serverInfo["tool_count"] = toolCount
		} else if toolCount, ok := srv["tool_count"].(float64); ok {
			serverInfo["tool_count"] = int(toolCount)
		}

		// Add error information
		if lastError, ok := srv["last_error"].(string); ok && lastError != "" {
			serverInfo["last_error"] = lastError
		}

		// Add repository information
		if repoURL, ok := srv["repository_url"].(string); ok && repoURL != "" {
			serverInfo["repository_url"] = repoURL
		}

		serverList = append(serverList, serverInfo)
	}

	content, err := json.MarshalIndent(serverList, "", "  ")
	if err != nil {
		s.logger.Error("Failed to marshal server list", zap.Error(err))
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Failed to format server list: %v", err),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"content": string(content),
		"count":   len(serverList),
	})
}

// handleChatListAllTools returns a list of all tools from all servers
func (s *Server) handleChatListAllTools(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	servers, err := s.GetAllServers()
	if err != nil {
		s.logger.Error("Failed to get all servers", zap.Error(err))
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Failed to get servers: %v", err),
		})
		return
	}

	// Collect tools from all servers
	allTools := make(map[string]interface{})
	totalToolCount := 0

	for _, srv := range servers {
		serverName, ok := srv["name"].(string)
		if !ok {
			continue
		}

		// Only fetch tools from enabled and connected servers
		enabled, _ := srv["enabled"].(bool)
		connected, _ := srv["connected"].(bool)
		quarantined, _ := srv["quarantined"].(bool)

		if !enabled || !connected || quarantined {
			allTools[serverName] = map[string]interface{}{
				"status": "unavailable",
				"reason": func() string {
					if quarantined {
						return "server is quarantined"
					} else if !enabled {
						return "server is disabled"
					} else if !connected {
						return "server is not connected"
					}
					return "unknown"
				}(),
				"tools": []interface{}{},
			}
			continue
		}

		// Get tools for this server
		tools, err := s.GetServerTools(serverName)
		if err != nil {
			allTools[serverName] = map[string]interface{}{
				"status": "error",
				"error":  err.Error(),
				"tools":  []interface{}{},
			}
			continue
		}

		// Format tools with essential information
		formattedTools := make([]map[string]interface{}, 0, len(tools))
		for _, tool := range tools {
			toolInfo := map[string]interface{}{
				"name": tool["name"],
			}
			if desc, ok := tool["description"].(string); ok && desc != "" {
				toolInfo["description"] = desc
			}
			if schema, ok := tool["inputSchema"].(map[string]interface{}); ok {
				toolInfo["inputSchema"] = schema
			}
			formattedTools = append(formattedTools, toolInfo)
		}

		allTools[serverName] = map[string]interface{}{
			"status": "available",
			"count":  len(formattedTools),
			"tools":  formattedTools,
		}
		totalToolCount += len(formattedTools)
	}

	content, err := json.MarshalIndent(allTools, "", "  ")
	if err != nil {
		s.logger.Error("Failed to marshal tools list", zap.Error(err))
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Failed to format tools list: %v", err),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":      true,
		"content":      string(content),
		"total_tools":  totalToolCount,
		"server_count": len(servers),
	})
}

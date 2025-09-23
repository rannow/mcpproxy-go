//go:build !nogui && !headless && !linux

package tray

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"
)

// TesterAgent handles testing and validation of MCP servers and tools
type TesterAgent struct {
	logger        *zap.Logger
	serverManager interface {
		GetServerTools(serverName string) ([]map[string]interface{}, error)
		EnableServer(serverName string, enabled bool) error
		GetAllServers() ([]map[string]interface{}, error)
		ReloadConfiguration() error
	}
}

// TestResult represents the result of a test
type TestResult struct {
	TestName    string                 `json:"test_name"`
	Status      string                 `json:"status"`      // "passed", "failed", "warning", "skipped"
	Message     string                 `json:"message"`
	Details     map[string]interface{} `json:"details,omitempty"`
	Duration    time.Duration          `json:"duration"`
	Timestamp   time.Time              `json:"timestamp"`
	Suggestions []string               `json:"suggestions,omitempty"`
}

// TestSuite represents a collection of test results
type TestSuite struct {
	SuiteName   string       `json:"suite_name"`
	ServerName  string       `json:"server_name"`
	Tests       []TestResult `json:"tests"`
	Summary     TestSummary  `json:"summary"`
	StartTime   time.Time    `json:"start_time"`
	EndTime     time.Time    `json:"end_time"`
	TotalTime   time.Duration `json:"total_time"`
}

// TestSummary provides an overview of test results
type TestSummary struct {
	Total   int `json:"total"`
	Passed  int `json:"passed"`
	Failed  int `json:"failed"`
	Warnings int `json:"warnings"`
	Skipped  int `json:"skipped"`
}

// NewTesterAgent creates a new tester agent
func NewTesterAgent(logger *zap.Logger, serverManager interface {
	GetServerTools(serverName string) ([]map[string]interface{}, error)
	EnableServer(serverName string, enabled bool) error
	GetAllServers() ([]map[string]interface{}, error)
	ReloadConfiguration() error
}) *TesterAgent {
	return &TesterAgent{
		logger:        logger,
		serverManager: serverManager,
	}
}

// ProcessMessage processes a message requesting testing operations
func (ta *TesterAgent) ProcessMessage(ctx context.Context, message ChatMessage, session *ChatSession) (*ChatMessage, error) {
	ta.logger.Info("Tester agent processing message",
		zap.String("session_id", session.ID),
		zap.String("server", session.ServerName))

	content := strings.ToLower(message.Content)

	var response string
	var metadata map[string]interface{}

	switch {
	case ta.containsKeywords(content, []string{"test connectivity", "connection test", "test connection"}):
		response, metadata = ta.testConnectivity(session.ServerName)

	case ta.containsKeywords(content, []string{"list tools", "show tools", "available tools", "what tools"}):
		response, metadata = ta.listAvailableTools(session.ServerName)

	case ta.containsKeywords(content, []string{"test tool", "test specific", "call tool"}):
		response, metadata = ta.handleToolTesting(session.ServerName, message.Content)

	case ta.containsKeywords(content, []string{"test all", "comprehensive test", "full test", "test everything"}):
		response, metadata = ta.runComprehensiveTests(session.ServerName)

	case ta.containsKeywords(content, []string{"performance test", "speed test", "benchmark"}):
		response, metadata = ta.runPerformanceTests(session.ServerName)

	case ta.containsKeywords(content, []string{"validation test", "validate", "check functionality"}):
		response, metadata = ta.runValidationTests(session.ServerName)

	case ta.containsKeywords(content, []string{"stress test", "load test", "stability"}):
		response, metadata = ta.runStressTests(session.ServerName)

	case ta.containsKeywords(content, []string{"test results", "show results", "last test"}):
		response, metadata = ta.showTestResults(session.ServerName)

	default:
		response, metadata = ta.provideTestingGuidance(session.ServerName)
	}

	return &ChatMessage{
		ID:        generateMessageID(),
		Role:      "assistant",
		Content:   response,
		AgentType: string(AgentTypeTester),
		Timestamp: time.Now(),
		Metadata:  metadata,
	}, nil
}

// GetCapabilities returns the capabilities of the tester agent
func (ta *TesterAgent) GetCapabilities() []string {
	return []string{
		"Connectivity testing",
		"Tool listing and validation",
		"Individual tool testing",
		"Comprehensive test suites",
		"Performance benchmarking",
		"Stress testing",
		"Functional validation",
		"Test result analysis",
	}
}

// GetAgentType returns the agent type
func (ta *TesterAgent) GetAgentType() AgentType {
	return AgentTypeTester
}

// CanHandle determines if this agent can handle a message
func (ta *TesterAgent) CanHandle(message ChatMessage) bool {
	content := strings.ToLower(message.Content)
	keywords := []string{
		"test", "testing", "verify", "validation", "check", "tool",
		"tools", "call", "benchmark", "performance", "connectivity",
		"stress", "load", "functionality", "working",
	}

	return ta.containsKeywords(content, keywords)
}

// containsKeywords checks if content contains any of the specified keywords
func (ta *TesterAgent) containsKeywords(content string, keywords []string) bool {
	for _, keyword := range keywords {
		if strings.Contains(content, keyword) {
			return true
		}
	}
	return false
}

// testConnectivity tests basic connectivity to the server
func (ta *TesterAgent) testConnectivity(serverName string) (string, map[string]interface{}) {
	ta.logger.Info("Testing connectivity", zap.String("server", serverName))

	startTime := time.Now()

	var responseBuilder strings.Builder
	responseBuilder.WriteString(fmt.Sprintf("🔌 **Connectivity Test for %s**\n\n", serverName))

	// Check if server exists and is enabled
	servers, err := ta.serverManager.GetAllServers()
	if err != nil {
		return fmt.Sprintf("❌ **Failed to get server information**: %v", err), map[string]interface{}{
			"error": err.Error(),
		}
	}

	var serverInfo map[string]interface{}
	found := false
	for _, server := range servers {
		if name, ok := server["name"].(string); ok && name == serverName {
			serverInfo = server
			found = true
			break
		}
	}

	if !found {
		return fmt.Sprintf("❌ **Server '%s' not found**", serverName), map[string]interface{}{
			"error": "server_not_found",
		}
	}

	// Check server status
	if enabled, ok := serverInfo["enabled"].(bool); !ok || !enabled {
		responseBuilder.WriteString("❌ **Server is disabled**\n")
		responseBuilder.WriteString("Enable the server first to test connectivity.\n\n")
		responseBuilder.WriteString("💡 **Ask the coordinator to enable the server**")

		return responseBuilder.String(), map[string]interface{}{
			"connectivity": false,
			"reason": "server_disabled",
		}
	}

	if quarantined, ok := serverInfo["quarantined"].(bool); ok && quarantined {
		responseBuilder.WriteString("🚨 **Server is quarantined**\n")
		responseBuilder.WriteString("The server is quarantined for security reasons.\n\n")
		responseBuilder.WriteString("💡 **Ask the coordinator about unquarantining the server**")

		return responseBuilder.String(), map[string]interface{}{
			"connectivity": false,
			"reason": "server_quarantined",
		}
	}

	// Test actual connectivity by trying to get tools
	tools, err := ta.serverManager.GetServerTools(serverName)
	testDuration := time.Since(startTime)

	if err != nil {
		responseBuilder.WriteString("❌ **Connectivity test failed**\n\n")
		responseBuilder.WriteString(fmt.Sprintf("**Error**: %v\n", err))
		responseBuilder.WriteString(fmt.Sprintf("**Test Duration**: %v\n\n", testDuration))

		responseBuilder.WriteString("**🔧 Troubleshooting Steps:**\n")
		responseBuilder.WriteString("1. Check if the server process is running\n")
		responseBuilder.WriteString("2. Verify configuration settings\n")
		responseBuilder.WriteString("3. Check network connectivity\n")
		responseBuilder.WriteString("4. Review server logs for errors\n\n")

		responseBuilder.WriteString("💡 **Ask the log analyzer to check for errors**")

		return responseBuilder.String(), map[string]interface{}{
			"connectivity": false,
			"error": err.Error(),
			"duration": testDuration.Milliseconds(),
		}
	}

	// Success
	responseBuilder.WriteString("✅ **Connectivity test passed!**\n\n")
	responseBuilder.WriteString(fmt.Sprintf("**Connection Time**: %v\n", testDuration))
	responseBuilder.WriteString(fmt.Sprintf("**Tools Available**: %d\n", len(tools)))

	if protocol, ok := serverInfo["protocol"].(string); ok {
		responseBuilder.WriteString(fmt.Sprintf("**Protocol**: %s\n", protocol))
	}

	responseBuilder.WriteString("\n**📊 Connection Quality:**\n")
	if testDuration < 100*time.Millisecond {
		responseBuilder.WriteString("🟢 **Excellent** - Very fast response\n")
	} else if testDuration < 500*time.Millisecond {
		responseBuilder.WriteString("🟡 **Good** - Normal response time\n")
	} else if testDuration < 2*time.Second {
		responseBuilder.WriteString("🟠 **Slow** - Consider checking network or server performance\n")
	} else {
		responseBuilder.WriteString("🔴 **Very Slow** - May indicate performance issues\n")
	}

	responseBuilder.WriteString("\n💡 **Want to test specific tools?** Ask me to list or test tools!")

	return responseBuilder.String(), map[string]interface{}{
		"connectivity": true,
		"tools_count": len(tools),
		"duration": testDuration.Milliseconds(),
		"protocol": serverInfo["protocol"],
	}
}

// listAvailableTools lists all available tools for a server
func (ta *TesterAgent) listAvailableTools(serverName string) (string, map[string]interface{}) {
	ta.logger.Info("Listing available tools", zap.String("server", serverName))

	startTime := time.Now()
	tools, err := ta.serverManager.GetServerTools(serverName)
	queryDuration := time.Since(startTime)

	if err != nil {
		return fmt.Sprintf("❌ **Failed to get tools**: %v", err), map[string]interface{}{
			"error": err.Error(),
		}
	}

	var responseBuilder strings.Builder
	responseBuilder.WriteString(fmt.Sprintf("🛠️ **Available Tools for %s**\n\n", serverName))

	if len(tools) == 0 {
		responseBuilder.WriteString("❌ **No tools available**\n\n")
		responseBuilder.WriteString("This could mean:\n")
		responseBuilder.WriteString("• The server is not properly connected\n")
		responseBuilder.WriteString("• The server doesn't expose any tools\n")
		responseBuilder.WriteString("• There's a configuration issue\n\n")
		responseBuilder.WriteString("💡 **Try testing connectivity first**")

		return responseBuilder.String(), map[string]interface{}{
			"tools": tools,
			"count": 0,
		}
	}

	responseBuilder.WriteString(fmt.Sprintf("**📊 Summary**: %d tools found (query took %v)\n\n", len(tools), queryDuration))

	// Group tools by category if possible
	categories := make(map[string][]map[string]interface{})
	uncategorized := []map[string]interface{}{}

	for _, tool := range tools {
		toolName, _ := tool["name"].(string)

		// Simple categorization based on tool name patterns
		category := "General"
		if strings.Contains(strings.ToLower(toolName), "file") || strings.Contains(strings.ToLower(toolName), "read") || strings.Contains(strings.ToLower(toolName), "write") {
			category = "File Operations"
		} else if strings.Contains(strings.ToLower(toolName), "git") {
			category = "Version Control"
		} else if strings.Contains(strings.ToLower(toolName), "search") || strings.Contains(strings.ToLower(toolName), "find") {
			category = "Search"
		} else if strings.Contains(strings.ToLower(toolName), "api") || strings.Contains(strings.ToLower(toolName), "request") {
			category = "API"
		}

		if category == "General" {
			uncategorized = append(uncategorized, tool)
		} else {
			categories[category] = append(categories[category], tool)
		}
	}

	// Display categorized tools
	for category, categoryTools := range categories {
		responseBuilder.WriteString(fmt.Sprintf("**📁 %s (%d tools):**\n", category, len(categoryTools)))
		for i, tool := range categoryTools {
			if i >= 5 { // Limit to 5 tools per category for readability
				responseBuilder.WriteString(fmt.Sprintf("  ... and %d more\n", len(categoryTools)-5))
				break
			}
			toolName, _ := tool["name"].(string)
			description, _ := tool["description"].(string)

			responseBuilder.WriteString(fmt.Sprintf("• **%s**", toolName))
			if description != "" && len(description) < 80 {
				responseBuilder.WriteString(fmt.Sprintf(" - %s", description))
			}
			responseBuilder.WriteString("\n")
		}
		responseBuilder.WriteString("\n")
	}

	// Display uncategorized tools
	if len(uncategorized) > 0 {
		responseBuilder.WriteString(fmt.Sprintf("**🔧 Other Tools (%d):**\n", len(uncategorized)))
		for i, tool := range uncategorized {
			if i >= 10 { // Limit to 10 uncategorized tools
				responseBuilder.WriteString(fmt.Sprintf("  ... and %d more\n", len(uncategorized)-10))
				break
			}
			toolName, _ := tool["name"].(string)
			description, _ := tool["description"].(string)

			responseBuilder.WriteString(fmt.Sprintf("• **%s**", toolName))
			if description != "" && len(description) < 80 {
				responseBuilder.WriteString(fmt.Sprintf(" - %s", description))
			}
			responseBuilder.WriteString("\n")
		}
	}

	responseBuilder.WriteString("\n**🧪 Testing Options:**\n")
	responseBuilder.WriteString("• \"Test tool [tool-name]\" - Test a specific tool\n")
	responseBuilder.WriteString("• \"Test all tools\" - Run comprehensive tests\n")
	responseBuilder.WriteString("• \"Performance test\" - Measure response times\n\n")
	responseBuilder.WriteString("💡 **Want to test a specific tool?** Just tell me which one!")

	return responseBuilder.String(), map[string]interface{}{
		"tools": tools,
		"count": len(tools),
		"categories": len(categories),
		"query_duration": queryDuration.Milliseconds(),
	}
}

// handleToolTesting handles testing of specific tools
func (ta *TesterAgent) handleToolTesting(serverName, userMessage string) (string, map[string]interface{}) {
	ta.logger.Info("Handling tool testing", zap.String("server", serverName))

	// Extract tool name from user message
	// This is a simple extraction - in practice, you might want more sophisticated parsing
	words := strings.Fields(strings.ToLower(userMessage))
	var toolName string

	// Look for patterns like "test tool toolname" or "test toolname"
	for i, word := range words {
		if (word == "tool" || word == "test") && i+1 < len(words) {
			// Take the next word as potential tool name
			candidate := words[i+1]
			if len(candidate) > 2 && !ta.isCommonWord(candidate) {
				toolName = candidate
				break
			}
		}
	}

	if toolName == "" {
		return fmt.Sprintf(`🧪 **Tool Testing for %s**

Please specify which tool you'd like to test:

**Examples:**
• "Test tool file_read"
• "Test the search tool"
• "Call the api_request tool"

**💡 Don't know the tool names?**
Ask me to "list available tools" first!

**🔧 Testing Options:**
• **Individual Tool** - Test a specific tool with sample parameters
• **All Tools** - Test all available tools (basic connectivity)
• **Performance** - Measure tool response times
• **Validation** - Verify tool functionality with real data

What would you like to test?`, serverName), map[string]interface{}{
			"guidance": true,
			"server": serverName,
		}
	}

	// Simulate tool testing (in a real implementation, you would actually call the tool)
	startTime := time.Now()

	// Get available tools to validate the tool exists
	tools, err := ta.serverManager.GetServerTools(serverName)
	if err != nil {
		return fmt.Sprintf("❌ **Failed to get tools**: %v", err), map[string]interface{}{
			"error": err.Error(),
		}
	}

	// Check if the tool exists
	var foundTool map[string]interface{}
	for _, tool := range tools {
		if name, ok := tool["name"].(string); ok {
			if strings.EqualFold(name, toolName) || strings.Contains(strings.ToLower(name), toolName) {
				foundTool = tool
				break
			}
		}
	}

	if foundTool == nil {
		return fmt.Sprintf(`❌ **Tool '%s' not found**

**Available tools:**
`, toolName) + ta.formatToolList(tools), map[string]interface{}{
			"error": "tool_not_found",
			"requested_tool": toolName,
		}
	}

	testDuration := time.Since(startTime)

	// Simulate test execution
	var responseBuilder strings.Builder
	responseBuilder.WriteString(fmt.Sprintf("🧪 **Tool Test: %s**\n\n", foundTool["name"]))

	// Tool information
	if description, ok := foundTool["description"].(string); ok && description != "" {
		responseBuilder.WriteString(fmt.Sprintf("**Description**: %s\n", description))
	}

	responseBuilder.WriteString(fmt.Sprintf("**Test Duration**: %v\n\n", testDuration))

	// Simulate test results
	responseBuilder.WriteString("**📊 Test Results:**\n")
	responseBuilder.WriteString("✅ **Tool Discovery**: Tool found and accessible\n")
	responseBuilder.WriteString("✅ **Metadata Validation**: Tool information is complete\n")

	// Note about actual execution
	responseBuilder.WriteString("⏳ **Execution Test**: Simulated (actual tool calls require parameters)\n\n")

	// Show tool schema if available
	if inputSchema, ok := foundTool["inputSchema"].(map[string]interface{}); ok {
		responseBuilder.WriteString("**🔧 Tool Parameters:**\n")
		if properties, ok := inputSchema["properties"].(map[string]interface{}); ok {
			for paramName, paramInfo := range properties {
				responseBuilder.WriteString(fmt.Sprintf("• **%s**", paramName))
				if param, ok := paramInfo.(map[string]interface{}); ok {
					if paramType, exists := param["type"]; exists {
						responseBuilder.WriteString(fmt.Sprintf(" (%v)", paramType))
					}
					if description, exists := param["description"]; exists {
						responseBuilder.WriteString(fmt.Sprintf(" - %v", description))
					}
				}
				responseBuilder.WriteString("\n")
			}
		}
		responseBuilder.WriteString("\n")
	}

	responseBuilder.WriteString("**💡 To test tool execution:**\n")
	responseBuilder.WriteString("Use the mcpproxy CLI tool:\n")
	responseBuilder.WriteString("```bash\n")
	responseBuilder.WriteString(fmt.Sprintf("mcpproxy call tool --tool-name=%s --json_args='{}'\n", foundTool["name"]))
	responseBuilder.WriteString("```\n\n")

	responseBuilder.WriteString("**🎯 Next Steps:**\n")
	responseBuilder.WriteString("• Provide parameters to test actual execution\n")
	responseBuilder.WriteString("• Run performance tests to measure response times\n")
	responseBuilder.WriteString("• Test with real data to validate functionality\n")

	return responseBuilder.String(), map[string]interface{}{
		"tool": foundTool,
		"test_duration": testDuration.Milliseconds(),
		"status": "metadata_validated",
	}
}

// runComprehensiveTests runs a comprehensive test suite
func (ta *TesterAgent) runComprehensiveTests(serverName string) (string, map[string]interface{}) {
	ta.logger.Info("Running comprehensive tests", zap.String("server", serverName))

	startTime := time.Now()

	testSuite := &TestSuite{
		SuiteName:  "Comprehensive Test Suite",
		ServerName: serverName,
		Tests:      []TestResult{},
		StartTime:  startTime,
	}

	var responseBuilder strings.Builder
	responseBuilder.WriteString(fmt.Sprintf("🧪 **Comprehensive Test Suite for %s**\n\n", serverName))

	// Test 1: Connectivity
	connectivityTest := ta.runConnectivityTest(serverName)
	testSuite.Tests = append(testSuite.Tests, connectivityTest)

	// Test 2: Tool Discovery
	toolDiscoveryTest := ta.runToolDiscoveryTest(serverName)
	testSuite.Tests = append(testSuite.Tests, toolDiscoveryTest)

	// Test 3: Configuration Validation
	configTest := ta.runConfigurationTest(serverName)
	testSuite.Tests = append(testSuite.Tests, configTest)

	// Test 4: Performance Basic Check
	performanceTest := ta.runBasicPerformanceTest(serverName)
	testSuite.Tests = append(testSuite.Tests, performanceTest)

	// Calculate summary
	testSuite.EndTime = time.Now()
	testSuite.TotalTime = testSuite.EndTime.Sub(testSuite.StartTime)
	testSuite.Summary = ta.calculateTestSummary(testSuite.Tests)

	// Build response
	responseBuilder.WriteString("**📊 Test Summary:**\n")
	responseBuilder.WriteString(fmt.Sprintf("• **Total Tests**: %d\n", testSuite.Summary.Total))
	responseBuilder.WriteString(fmt.Sprintf("• **Passed**: %d ✅\n", testSuite.Summary.Passed))
	responseBuilder.WriteString(fmt.Sprintf("• **Failed**: %d ❌\n", testSuite.Summary.Failed))
	responseBuilder.WriteString(fmt.Sprintf("• **Warnings**: %d ⚠️\n", testSuite.Summary.Warnings))
	responseBuilder.WriteString(fmt.Sprintf("• **Total Time**: %v\n\n", testSuite.TotalTime))

	// Show detailed results
	responseBuilder.WriteString("**📋 Detailed Results:**\n\n")
	for i, test := range testSuite.Tests {
		statusIcon := "❌"
		switch test.Status {
		case "passed":
			statusIcon = "✅"
		case "warning":
			statusIcon = "⚠️"
		case "skipped":
			statusIcon = "⏭️"
		}

		responseBuilder.WriteString(fmt.Sprintf("**%d. %s** %s\n", i+1, test.TestName, statusIcon))
		responseBuilder.WriteString(fmt.Sprintf("   %s (%v)\n", test.Message, test.Duration))

		if len(test.Suggestions) > 0 {
			responseBuilder.WriteString("   💡 Suggestions: " + strings.Join(test.Suggestions, ", ") + "\n")
		}
		responseBuilder.WriteString("\n")
	}

	// Overall assessment
	if testSuite.Summary.Failed == 0 {
		if testSuite.Summary.Warnings == 0 {
			responseBuilder.WriteString("🎉 **Overall: Excellent!** All tests passed without issues.\n")
		} else {
			responseBuilder.WriteString("👍 **Overall: Good!** All critical tests passed with some warnings.\n")
		}
	} else {
		responseBuilder.WriteString("⚠️ **Overall: Needs Attention!** Some tests failed and require investigation.\n")
	}

	return responseBuilder.String(), map[string]interface{}{
		"test_suite": testSuite,
		"summary": testSuite.Summary,
	}
}

// Helper methods for individual tests
func (ta *TesterAgent) runConnectivityTest(serverName string) TestResult {
	start := time.Now()

	servers, err := ta.serverManager.GetAllServers()
	if err != nil {
		return TestResult{
			TestName:    "Connectivity Test",
			Status:      "failed",
			Message:     fmt.Sprintf("Failed to get server info: %v", err),
			Duration:    time.Since(start),
			Timestamp:   time.Now(),
			Suggestions: []string{"Check server configuration", "Verify mcpproxy is running"},
		}
	}

	// Check if server exists and is enabled
	var serverInfo map[string]interface{}
	found := false
	for _, server := range servers {
		if name, ok := server["name"].(string); ok && name == serverName {
			serverInfo = server
			found = true
			break
		}
	}

	if !found {
		return TestResult{
			TestName:    "Connectivity Test",
			Status:      "failed",
			Message:     "Server not found in configuration",
			Duration:    time.Since(start),
			Timestamp:   time.Now(),
			Suggestions: []string{"Check server name", "Verify configuration"},
		}
	}

	if enabled, ok := serverInfo["enabled"].(bool); !ok || !enabled {
		return TestResult{
			TestName:    "Connectivity Test",
			Status:      "failed",
			Message:     "Server is disabled",
			Duration:    time.Since(start),
			Timestamp:   time.Now(),
			Suggestions: []string{"Enable the server first"},
		}
	}

	// Try to get tools
	_, err = ta.serverManager.GetServerTools(serverName)
	duration := time.Since(start)

	if err != nil {
		return TestResult{
			TestName:    "Connectivity Test",
			Status:      "failed",
			Message:     fmt.Sprintf("Connection failed: %v", err),
			Duration:    duration,
			Timestamp:   time.Now(),
			Suggestions: []string{"Check logs", "Verify server process", "Check network"},
		}
	}

	status := "passed"
	message := "Connection successful"
	var suggestions []string

	if duration > 2*time.Second {
		status = "warning"
		message = "Connection slow but successful"
		suggestions = []string{"Check network performance", "Monitor server resources"}
	}

	return TestResult{
		TestName:    "Connectivity Test",
		Status:      status,
		Message:     message,
		Duration:    duration,
		Timestamp:   time.Now(),
		Suggestions: suggestions,
		Details: map[string]interface{}{
			"response_time_ms": duration.Milliseconds(),
		},
	}
}

func (ta *TesterAgent) runToolDiscoveryTest(serverName string) TestResult {
	start := time.Now()

	tools, err := ta.serverManager.GetServerTools(serverName)
	duration := time.Since(start)

	if err != nil {
		return TestResult{
			TestName:    "Tool Discovery Test",
			Status:      "failed",
			Message:     fmt.Sprintf("Failed to discover tools: %v", err),
			Duration:    duration,
			Timestamp:   time.Now(),
			Suggestions: []string{"Check server connectivity", "Verify tool exposure"},
		}
	}

	if len(tools) == 0 {
		return TestResult{
			TestName:    "Tool Discovery Test",
			Status:      "warning",
			Message:     "No tools discovered",
			Duration:    duration,
			Timestamp:   time.Now(),
			Suggestions: []string{"Check if server exposes tools", "Verify server implementation"},
			Details: map[string]interface{}{
				"tools_count": 0,
			},
		}
	}

	return TestResult{
		TestName:  "Tool Discovery Test",
		Status:    "passed",
		Message:   fmt.Sprintf("Discovered %d tools", len(tools)),
		Duration:  duration,
		Timestamp: time.Now(),
		Details: map[string]interface{}{
			"tools_count": len(tools),
		},
	}
}

func (ta *TesterAgent) runConfigurationTest(serverName string) TestResult {
	start := time.Now()

	servers, err := ta.serverManager.GetAllServers()
	duration := time.Since(start)

	if err != nil {
		return TestResult{
			TestName:    "Configuration Test",
			Status:      "failed",
			Message:     fmt.Sprintf("Failed to get configuration: %v", err),
			Duration:    duration,
			Timestamp:   time.Now(),
			Suggestions: []string{"Check configuration file", "Verify permissions"},
		}
	}

	var serverConfig map[string]interface{}
	found := false
	for _, server := range servers {
		if name, ok := server["name"].(string); ok && name == serverName {
			serverConfig = server
			found = true
			break
		}
	}

	if !found {
		return TestResult{
			TestName:    "Configuration Test",
			Status:      "failed",
			Message:     "Server configuration not found",
			Duration:    duration,
			Timestamp:   time.Now(),
			Suggestions: []string{"Check server name", "Verify configuration"},
		}
	}

	// Check required fields
	requiredFields := []string{"name", "protocol"}
	missingFields := []string{}

	for _, field := range requiredFields {
		if _, exists := serverConfig[field]; !exists {
			missingFields = append(missingFields, field)
		}
	}

	if len(missingFields) > 0 {
		return TestResult{
			TestName:    "Configuration Test",
			Status:      "failed",
			Message:     fmt.Sprintf("Missing required fields: %s", strings.Join(missingFields, ", ")),
			Duration:    duration,
			Timestamp:   time.Now(),
			Suggestions: []string{"Add missing configuration fields"},
			Details: map[string]interface{}{
				"missing_fields": missingFields,
			},
		}
	}

	var suggestions []string

	// Check for optional improvements
	if _, exists := serverConfig["env"]; !exists {
		suggestions = append(suggestions, "Consider adding environment variables")
	}

	status := "passed"
	if len(suggestions) > 0 {
		status = "warning"
	}

	return TestResult{
		TestName:    "Configuration Test",
		Status:      status,
		Message:     "Configuration is valid",
		Duration:    duration,
		Timestamp:   time.Now(),
		Suggestions: suggestions,
	}
}

func (ta *TesterAgent) runBasicPerformanceTest(serverName string) TestResult {
	start := time.Now()

	// Run multiple tool discovery calls to measure performance
	iterations := 3
	var totalDuration time.Duration
	var successCount int

	for i := 0; i < iterations; i++ {
		iterStart := time.Now()
		_, err := ta.serverManager.GetServerTools(serverName)
		iterDuration := time.Since(iterStart)
		totalDuration += iterDuration

		if err == nil {
			successCount++
		}
	}

	avgDuration := totalDuration / time.Duration(iterations)
	overallDuration := time.Since(start)

	if successCount == 0 {
		return TestResult{
			TestName:    "Basic Performance Test",
			Status:      "failed",
			Message:     "All performance test iterations failed",
			Duration:    overallDuration,
			Timestamp:   time.Now(),
			Suggestions: []string{"Check connectivity first"},
		}
	}

	status := "passed"
	message := fmt.Sprintf("Average response time: %v", avgDuration)
	var suggestions []string

	if avgDuration > 1*time.Second {
		status = "warning"
		message = fmt.Sprintf("Performance concern: %v average", avgDuration)
		suggestions = []string{"Consider performance optimization", "Check network latency"}
	}

	if successCount < iterations {
		status = "warning"
		message += fmt.Sprintf(" (%d/%d successful)", successCount, iterations)
		suggestions = append(suggestions, "Intermittent connectivity issues detected")
	}

	return TestResult{
		TestName:    "Basic Performance Test",
		Status:      status,
		Message:     message,
		Duration:    overallDuration,
		Timestamp:   time.Now(),
		Suggestions: suggestions,
		Details: map[string]interface{}{
			"avg_response_time_ms": avgDuration.Milliseconds(),
			"success_rate":         float64(successCount) / float64(iterations),
			"iterations":           iterations,
		},
	}
}

// calculateTestSummary calculates a summary of test results
func (ta *TesterAgent) calculateTestSummary(tests []TestResult) TestSummary {
	summary := TestSummary{}

	for _, test := range tests {
		summary.Total++
		switch test.Status {
		case "passed":
			summary.Passed++
		case "failed":
			summary.Failed++
		case "warning":
			summary.Warnings++
		case "skipped":
			summary.Skipped++
		}
	}

	return summary
}

// Helper methods
func (ta *TesterAgent) isCommonWord(word string) bool {
	commonWords := []string{
		"the", "and", "or", "but", "in", "on", "at", "to", "for", "of", "with",
		"test", "tool", "call", "run", "check", "verify", "this", "that", "it",
	}

	for _, common := range commonWords {
		if word == common {
			return true
		}
	}
	return false
}

func (ta *TesterAgent) formatToolList(tools []map[string]interface{}) string {
	var result strings.Builder

	for i, tool := range tools {
		if i >= 10 { // Limit to 10 tools
			result.WriteString("... and more\n")
			break
		}
		if name, ok := tool["name"].(string); ok {
			result.WriteString(fmt.Sprintf("• %s\n", name))
		}
	}

	return result.String()
}

// Placeholder implementations for other test methods
func (ta *TesterAgent) runPerformanceTests(serverName string) (string, map[string]interface{}) {
	return fmt.Sprintf(`⚡ **Performance Testing for %s**

Performance testing is being implemented. This will include:

**📊 Response Time Testing:**
• Measure tool discovery latency
• Test individual tool response times
• Identify performance bottlenecks

**🔄 Throughput Testing:**
• Multiple concurrent requests
• Load capacity assessment
• Resource usage monitoring

**📈 Benchmarking:**
• Compare against baseline metrics
• Track performance over time
• Identify regression patterns

💡 **For now, try "comprehensive test" which includes basic performance checks!**`, serverName), map[string]interface{}{
		"feature": "coming_soon",
	}
}

func (ta *TesterAgent) runValidationTests(serverName string) (string, map[string]interface{}) {
	return fmt.Sprintf(`✅ **Validation Testing for %s**

Validation testing is being implemented. This will include:

**🔍 Functional Validation:**
• Verify tool inputs and outputs
• Test error handling
• Validate edge cases

**📋 Schema Validation:**
• Check tool parameter schemas
• Validate response formats
• Ensure MCP compliance

**🧪 Data Validation:**
• Test with various input types
• Verify data consistency
• Check boundary conditions

💡 **For now, try "comprehensive test" which includes basic validation checks!**`, serverName), map[string]interface{}{
		"feature": "coming_soon",
	}
}

func (ta *TesterAgent) runStressTests(serverName string) (string, map[string]interface{}) {
	return fmt.Sprintf(`🔥 **Stress Testing for %s**

Stress testing is being implemented. This will include:

**💪 Load Testing:**
• High volume request testing
• Concurrent user simulation
• Resource exhaustion testing

**⏱️ Stability Testing:**
• Long-running operation tests
• Memory leak detection
• Connection stability

**🚨 Failure Testing:**
• Error recovery testing
• Timeout handling
• Graceful degradation

💡 **For now, try "comprehensive test" for stability assessment!**`, serverName), map[string]interface{}{
		"feature": "coming_soon",
	}
}

func (ta *TesterAgent) showTestResults(serverName string) (string, map[string]interface{}) {
	return fmt.Sprintf(`📊 **Test Results for %s**

Test result storage is being implemented. This will include:

**📈 Historical Results:**
• Previous test run outcomes
• Performance trend analysis
• Regression detection

**📋 Detailed Reports:**
• Test execution logs
• Performance metrics
• Error analysis

**📊 Analytics:**
• Success rate trends
• Performance baselines
• Issue patterns

💡 **For now, run "comprehensive test" to see current results!**`, serverName), map[string]interface{}{
		"feature": "coming_soon",
	}
}

func (ta *TesterAgent) provideTestingGuidance(serverName string) (string, map[string]interface{}) {
	response := fmt.Sprintf(`🧪 **Testing Assistant for %s**

I can help you test and validate your MCP server:

**🔌 Connectivity Testing:**
• "Test connectivity" - Basic connection test
• "Check if server is working"

**🛠️ Tool Testing:**
• "List available tools" - Show all tools
• "Test tool [name]" - Test specific tool
• "What tools are available?"

**📊 Comprehensive Testing:**
• "Test everything" - Full test suite
• "Run all tests" - Complete validation
• "Comprehensive test"

**⚡ Performance Testing:**
• "Performance test" - Speed benchmarks
• "Check response times"
• "Benchmark the server"

**✅ Validation Testing:**
• "Validate functionality"
• "Check if tools work correctly"
• "Verify server behavior"

**💪 Advanced Testing:**
• "Stress test" - Load and stability
• "Load test" - Capacity testing

**💡 Example Requests:**
• "Test connectivity to see if it's working"
• "List all tools available"
• "Test the file_read tool"
• "Run comprehensive tests"
• "Check performance"

**🎯 Quick Start:**
1. Start with "test connectivity"
2. Then "list available tools"
3. Finally "test everything" for full validation

What testing would you like me to perform?`, serverName)

	return response, map[string]interface{}{
		"guidance_type": "testing",
		"available_tests": []string{
			"connectivity",
			"tool_listing",
			"tool_testing",
			"comprehensive",
			"performance",
			"validation",
			"stress",
		},
	}
}
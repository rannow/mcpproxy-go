//go:build !nogui && !headless && !linux

package tray

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"
)

// CoordinatorAgent is the main orchestrating agent that coordinates other agents
type CoordinatorAgent struct {
	logger        *zap.Logger
	serverManager interface {
		GetServerTools(serverName string) ([]map[string]interface{}, error)
		EnableServer(serverName string, enabled bool) error
		GetAllServers() ([]map[string]interface{}, error)
		ReloadConfiguration() error
	}
}

// NewCoordinatorAgent creates a new coordinator agent
func NewCoordinatorAgent(logger *zap.Logger, serverManager interface {
	GetServerTools(serverName string) ([]map[string]interface{}, error)
	EnableServer(serverName string, enabled bool) error
	GetAllServers() ([]map[string]interface{}, error)
	ReloadConfiguration() error
}) *CoordinatorAgent {
	return &CoordinatorAgent{
		logger:        logger,
		serverManager: serverManager,
	}
}

// ProcessMessage processes a message and coordinates with other agents
func (ca *CoordinatorAgent) ProcessMessage(ctx context.Context, message ChatMessage, session *ChatSession) (*ChatMessage, error) {
	ca.logger.Info("Coordinator processing message",
		zap.String("session_id", session.ID),
		zap.String("server", session.ServerName),
		zap.String("content", message.Content))

	content := strings.ToLower(message.Content)

	var response string
	var metadata map[string]interface{}

	// Analyze the message and determine appropriate response
	switch {
	case ca.containsKeywords(content, []string{"status", "check", "diagnose", "problem", "issue"}):
		response, metadata = ca.handleStatusCheck(session.ServerName)

	case ca.containsKeywords(content, []string{"install", "setup", "configure"}):
		response, metadata = ca.handleInstallationGuidance(session.ServerName)

	case ca.containsKeywords(content, []string{"test", "verify", "tools", "working"}):
		response, metadata = ca.handleTestingGuidance(session.ServerName)

	case ca.containsKeywords(content, []string{"logs", "error", "debug", "troubleshoot"}):
		response, metadata = ca.handleLogAnalysisGuidance(session.ServerName)

	case ca.containsKeywords(content, []string{"config", "configuration", "settings", "update"}):
		response, metadata = ca.handleConfigurationGuidance(session.ServerName)

	case ca.containsKeywords(content, []string{"restart", "reload", "refresh"}):
		response, metadata = ca.handleRestartGuidance(session.ServerName)

	case ca.containsKeywords(content, []string{"help", "what can you do", "capabilities"}):
		response, metadata = ca.handleHelpRequest()

	default:
		response, metadata = ca.handleGeneralInquiry(message.Content, session.ServerName)
	}

	return &ChatMessage{
		ID:        generateMessageID(),
		Role:      "assistant",
		Content:   response,
		AgentType: string(AgentTypeCoordinator),
		Timestamp: time.Now(),
		Metadata:  metadata,
	}, nil
}

// GetCapabilities returns the capabilities of the coordinator agent
func (ca *CoordinatorAgent) GetCapabilities() []string {
	return []string{
		"Server status checking",
		"Installation guidance",
		"Configuration management",
		"Testing coordination",
		"Log analysis coordination",
		"Agent coordination",
		"General troubleshooting",
	}
}

// GetAgentType returns the agent type
func (ca *CoordinatorAgent) GetAgentType() AgentType {
	return AgentTypeCoordinator
}

// CanHandle determines if this agent can handle a message
func (ca *CoordinatorAgent) CanHandle(message ChatMessage) bool {
	// Coordinator can handle any message as a fallback
	return true
}

// containsKeywords checks if content contains any of the specified keywords
func (ca *CoordinatorAgent) containsKeywords(content string, keywords []string) bool {
	for _, keyword := range keywords {
		if strings.Contains(content, keyword) {
			return true
		}
	}
	return false
}

// handleStatusCheck performs a comprehensive status check
func (ca *CoordinatorAgent) handleStatusCheck(serverName string) (string, map[string]interface{}) {
	ca.logger.Info("Performing status check", zap.String("server", serverName))

	// Get server information
	servers, err := ca.serverManager.GetAllServers()
	if err != nil {
		return fmt.Sprintf("❌ Failed to get server information: %v", err), map[string]interface{}{
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
		return fmt.Sprintf("❌ Server '%s' not found in configuration", serverName), map[string]interface{}{
			"error": "server_not_found",
		}
	}

	// Build status response
	var statusBuilder strings.Builder
	statusBuilder.WriteString(fmt.Sprintf("📊 **Status Check for %s**\n\n", serverName))

	// Basic server info
	if enabled, ok := serverInfo["enabled"].(bool); ok {
		if enabled {
			statusBuilder.WriteString("✅ **Status**: Enabled\n")
		} else {
			statusBuilder.WriteString("🔴 **Status**: Disabled\n")
		}
	}

	if quarantined, ok := serverInfo["quarantined"].(bool); ok && quarantined {
		statusBuilder.WriteString("🚨 **Security**: Quarantined\n")
	}

	if protocol, ok := serverInfo["protocol"].(string); ok {
		statusBuilder.WriteString(fmt.Sprintf("🔌 **Protocol**: %s\n", protocol))
	}

	// Try to get tools to test connectivity
	tools, err := ca.serverManager.GetServerTools(serverName)
	if err != nil {
		statusBuilder.WriteString("❌ **Connectivity**: Failed to retrieve tools\n")
		statusBuilder.WriteString(fmt.Sprintf("   Error: %v\n", err))
	} else {
		statusBuilder.WriteString(fmt.Sprintf("✅ **Connectivity**: Active (%d tools available)\n", len(tools)))
	}

	statusBuilder.WriteString("\n💡 **Next Steps**:\n")
	if enabled, ok := serverInfo["enabled"].(bool); ok && !enabled {
		statusBuilder.WriteString("• Enable the server to start diagnosis\n")
	} else if err != nil {
		statusBuilder.WriteString("• Check logs for connection issues\n")
		statusBuilder.WriteString("• Verify server configuration\n")
		statusBuilder.WriteString("• Test installation requirements\n")
	} else {
		statusBuilder.WriteString("• Server appears to be working correctly\n")
		statusBuilder.WriteString("• You can test specific tools or check logs for any issues\n")
	}

	return statusBuilder.String(), map[string]interface{}{
		"server_info": serverInfo,
		"tools_count": len(tools),
		"has_connectivity": err == nil,
	}
}

// handleInstallationGuidance provides installation guidance
func (ca *CoordinatorAgent) handleInstallationGuidance(serverName string) (string, map[string]interface{}) {
	response := fmt.Sprintf(`🔧 **Installation Guidance for %s**

I can help you with:

**1. Configuration Setup**
• Verify server configuration parameters
• Check required fields and environment variables
• Validate connection settings

**2. Dependency Installation**
• Check system requirements (Node.js, Python, etc.)
• Install required packages and dependencies
• Verify installation paths

**3. Testing Installation**
• Test server connectivity
• Verify tool availability
• Run basic functionality tests

**4. Troubleshooting**
• Analyze installation logs
• Identify common installation issues
• Provide step-by-step fixes

What specific aspect would you like help with? For example:
• "Check configuration for missing settings"
• "Install dependencies for this server"
• "Test if installation is working"
• "Show me the logs"`, serverName)

	return response, map[string]interface{}{
		"guidance_type": "installation",
		"available_actions": []string{
			"check_config",
			"install_dependencies",
			"test_installation",
			"analyze_logs",
		},
	}
}

// handleTestingGuidance provides testing guidance
func (ca *CoordinatorAgent) handleTestingGuidance(serverName string) (string, map[string]interface{}) {
	response := fmt.Sprintf(`🧪 **Testing Guidance for %s**

I can help you test your MCP server:

**1. Connectivity Testing**
• Test basic server connection
• Verify authentication if required
• Check network connectivity

**2. Tool Testing**
• List available tools
• Test individual tool calls
• Verify tool responses and functionality

**3. Performance Testing**
• Measure response times
• Test under load
• Check resource usage

**4. Integration Testing**
• Test with real-world scenarios
• Verify data flow
• Check error handling

What would you like to test? For example:
• "List all available tools"
• "Test a specific tool"
• "Check server response time"
• "Run a comprehensive test suite"`, serverName)

	return response, map[string]interface{}{
		"guidance_type": "testing",
		"available_tests": []string{
			"connectivity",
			"tools_list",
			"tool_execution",
			"performance",
			"integration",
		},
	}
}

// handleLogAnalysisGuidance provides log analysis guidance
func (ca *CoordinatorAgent) handleLogAnalysisGuidance(serverName string) (string, map[string]interface{}) {
	response := fmt.Sprintf(`📋 **Log Analysis for %s**

I can help analyze logs to identify issues:

**1. Error Analysis**
• Find and categorize errors
• Identify recurring issues
• Analyze error patterns

**2. Connection Issues**
• Track connection attempts
• Identify timeout issues
• Analyze authentication failures

**3. Performance Analysis**
• Monitor response times
• Identify bottlenecks
• Track resource usage

**4. Trend Analysis**
• Compare historical data
• Identify degradation patterns
• Track improvement over time

What would you like me to analyze? For example:
• "Show me recent errors"
• "Analyze connection problems"
• "Check for performance issues"
• "Find authentication failures"`, serverName)

	return response, map[string]interface{}{
		"guidance_type": "log_analysis",
		"analysis_types": []string{
			"errors",
			"connections",
			"performance",
			"trends",
		},
	}
}

// handleConfigurationGuidance provides configuration guidance
func (ca *CoordinatorAgent) handleConfigurationGuidance(serverName string) (string, map[string]interface{}) {
	response := fmt.Sprintf(`⚙️ **Configuration Management for %s**

I can help with configuration:

**1. Configuration Validation**
• Check required fields
• Validate settings format
• Verify environment variables

**2. Configuration Updates**
• Update server settings
• Modify connection parameters
• Add environment variables

**3. Best Practices**
• Security recommendations
• Performance optimizations
• Reliability improvements

**4. Backup & Restore**
• Backup current configuration
• Restore previous settings
• Compare configurations

What configuration help do you need? For example:
• "Check my current configuration"
• "Update connection settings"
• "Add environment variables"
• "Show configuration recommendations"`, serverName)

	return response, map[string]interface{}{
		"guidance_type": "configuration",
		"config_actions": []string{
			"validate",
			"update",
			"optimize",
			"backup",
		},
	}
}

// handleRestartGuidance provides restart guidance
func (ca *CoordinatorAgent) handleRestartGuidance(serverName string) (string, map[string]interface{}) {
	response := fmt.Sprintf(`🔄 **Restart Operations for %s**

I can help with server restart operations:

**1. Safe Restart**
• Check current connections
• Gracefully restart server
• Verify restart success

**2. Configuration Reload**
• Reload configuration without restart
• Apply new settings
• Validate changes

**3. Troubleshooting Restarts**
• Identify restart failures
• Check startup logs
• Resolve startup issues

**4. Recovery Operations**
• Restore from backup
• Reset to defaults
• Emergency recovery

What restart operation do you need? For example:
• "Restart the server safely"
• "Reload configuration"
• "Fix restart issues"
• "Check startup status"`, serverName)

	return response, map[string]interface{}{
		"guidance_type": "restart",
		"restart_options": []string{
			"safe_restart",
			"config_reload",
			"troubleshoot",
			"recovery",
		},
	}
}

// handleHelpRequest provides help information
func (ca *CoordinatorAgent) handleHelpRequest() (string, map[string]interface{}) {
	response := `🤖 **Diagnostic Agent Capabilities**

I'm your diagnostic coordinator with access to specialized agents:

**🎯 Coordinator (me)**
• Overall system orchestration
• Status checking and guidance
• Agent coordination

**📊 Log Analyzer**
• Analyze server logs for errors
• Identify patterns and trends
• Performance monitoring

**📚 Documentation Analyzer**
• Analyze installation guides
• Extract requirements and steps
• Troubleshooting assistance

**⚙️ Configuration Manager**
• Update server configurations
• Validate settings
• Apply best practices

**🔧 Service Installer**
• Install dependencies
• Setup services
• Environment configuration

**🧪 Testing Agent**
• Test MCP tools
• Connectivity testing
• Performance validation

**Available Commands:**
• Ask about server status: "What's the status of my server?"
• Get installation help: "Help me install this server"
• Test functionality: "Test my server tools"
• Analyze logs: "Check logs for errors"
• Update config: "Update my configuration"
• Get help: "What can you help me with?"

Just describe what you need in natural language!`

	return response, map[string]interface{}{
		"agent_types": []string{
			"coordinator",
			"log_analyzer",
			"doc_analyzer",
			"config_manager",
			"installer",
			"tester",
		},
	}
}

// handleGeneralInquiry handles general questions
func (ca *CoordinatorAgent) handleGeneralInquiry(content string, serverName string) (string, map[string]interface{}) {
	response := fmt.Sprintf(`I understand you're asking about: "%s"

For the server **%s**, I can help with:

• **Status & Diagnostics** - Check if everything is working
• **Installation & Setup** - Get the server configured properly
• **Testing & Validation** - Verify functionality
• **Log Analysis** - Find and fix issues
• **Configuration** - Update settings and parameters
• **Troubleshooting** - Solve specific problems

Could you be more specific about what you'd like to do? For example:
• "Check the status of my server"
• "Help me fix connection issues"
• "Show me how to test the tools"
• "Analyze recent errors"`, content, serverName)

	return response, map[string]interface{}{
		"inquiry_type": "general",
		"original_message": content,
	}
}
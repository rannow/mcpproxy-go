//go:build !nogui && !headless && !linux

package tray

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"go.uber.org/zap"
	"mcpproxy-go/internal/config"
)

// ConfigUpdateAgent handles configuration updates and management
type ConfigUpdateAgent struct {
	logger        *zap.Logger
	serverManager interface {
		GetServerTools(serverName string) ([]map[string]interface{}, error)
		EnableServer(serverName string, enabled bool) error
		GetAllServers() ([]map[string]interface{}, error)
		ReloadConfiguration() error
		GetConfigPath() string
	}
}

// ConfigUpdateRequest represents a configuration update request
type ConfigUpdateRequest struct {
	ServerName string                 `json:"server_name"`
	Updates    map[string]interface{} `json:"updates"`
	Action     string                 `json:"action"` // "update", "add_env", "remove_env", "validate"
}

// ConfigValidationResult contains validation results
type ConfigValidationResult struct {
	Valid          bool     `json:"valid"`
	Errors         []string `json:"errors"`
	Warnings       []string `json:"warnings"`
	Suggestions    []string `json:"suggestions"`
	MissingFields  []string `json:"missing_fields"`
	OptionalFields []string `json:"optional_fields"`
}

// NewConfigUpdateAgent creates a new configuration update agent
func NewConfigUpdateAgent(logger *zap.Logger, serverManager interface {
	GetServerTools(serverName string) ([]map[string]interface{}, error)
	EnableServer(serverName string, enabled bool) error
	GetAllServers() ([]map[string]interface{}, error)
	ReloadConfiguration() error
	GetConfigPath() string
}) *ConfigUpdateAgent {
	return &ConfigUpdateAgent{
		logger:        logger,
		serverManager: serverManager,
	}
}

// ProcessMessage processes a message requesting configuration updates
func (ca *ConfigUpdateAgent) ProcessMessage(ctx context.Context, message ChatMessage, session *ChatSession) (*ChatMessage, error) {
	ca.logger.Info("Config update agent processing message",
		zap.String("session_id", session.ID),
		zap.String("server", session.ServerName))

	content := strings.ToLower(message.Content)

	var response string
	var metadata map[string]interface{}

	switch {
	case ca.containsKeywords(content, []string{"validate", "check config", "verify config"}):
		response, metadata = ca.validateConfiguration(session.ServerName)

	case ca.containsKeywords(content, []string{"show config", "current config", "display config"}):
		response, metadata = ca.showCurrentConfiguration(session.ServerName)

	case ca.containsKeywords(content, []string{"add environment", "add env", "set env"}):
		response, metadata = ca.handleEnvironmentVariables(session.ServerName, message.Content, "add")

	case ca.containsKeywords(content, []string{"remove env", "delete env", "unset env"}):
		response, metadata = ca.handleEnvironmentVariables(session.ServerName, message.Content, "remove")

	case ca.containsKeywords(content, []string{"update config", "modify config", "change config"}):
		response, metadata = ca.handleConfigurationUpdate(session.ServerName, message.Content)

	case ca.containsKeywords(content, []string{"backup config", "save config", "backup"}):
		response, metadata = ca.backupConfiguration(session.ServerName)

	case ca.containsKeywords(content, []string{"restore config", "restore backup"}):
		response, metadata = ca.restoreConfiguration(session.ServerName)

	case ca.containsKeywords(content, []string{"optimize config", "improve config", "best practices"}):
		response, metadata = ca.optimizeConfiguration(session.ServerName)

	default:
		response, metadata = ca.provideConfigurationGuidance(session.ServerName)
	}

	return &ChatMessage{
		ID:        generateMessageID(),
		Role:      "assistant",
		Content:   response,
		AgentType: string(AgentTypeConfigUpdate),
		Timestamp: time.Now(),
		Metadata:  metadata,
	}, nil
}

// GetCapabilities returns the capabilities of the config update agent
func (ca *ConfigUpdateAgent) GetCapabilities() []string {
	return []string{
		"Configuration validation",
		"Environment variable management",
		"Configuration updates",
		"Configuration backup/restore",
		"Configuration optimization",
		"Field validation",
		"Best practices recommendations",
	}
}

// GetAgentType returns the agent type
func (ca *ConfigUpdateAgent) GetAgentType() AgentType {
	return AgentTypeConfigUpdate
}

// CanHandle determines if this agent can handle a message
func (ca *ConfigUpdateAgent) CanHandle(message ChatMessage) bool {
	content := strings.ToLower(message.Content)
	keywords := []string{
		"config", "configuration", "settings", "update", "modify",
		"environment", "env", "validate", "backup", "restore",
		"optimize", "change", "set", "add", "remove",
	}

	return ca.containsKeywords(content, keywords)
}

// containsKeywords checks if content contains any of the specified keywords
func (ca *ConfigUpdateAgent) containsKeywords(content string, keywords []string) bool {
	for _, keyword := range keywords {
		if strings.Contains(content, keyword) {
			return true
		}
	}
	return false
}

// validateConfiguration validates the current server configuration
func (ca *ConfigUpdateAgent) validateConfiguration(serverName string) (string, map[string]interface{}) {
	ca.logger.Info("Validating configuration", zap.String("server", serverName))

	// Get current server configuration
	servers, err := ca.serverManager.GetAllServers()
	if err != nil {
		return fmt.Sprintf("❌ **Failed to get server configuration**: %v", err), map[string]interface{}{
			"error": err.Error(),
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
		return fmt.Sprintf("❌ **Server '%s' not found in configuration**", serverName), map[string]interface{}{
			"error": "server_not_found",
		}
	}

	// Validate the configuration
	validation := ca.performConfigValidation(serverConfig)

	var responseBuilder strings.Builder
	responseBuilder.WriteString(fmt.Sprintf("🔍 **Configuration Validation for %s**\n\n", serverName))

	if validation.Valid {
		responseBuilder.WriteString("✅ **Configuration is valid!**\n\n")
		responseBuilder.WriteString("All required fields are present and properly configured.\n")
	} else {
		responseBuilder.WriteString("❌ **Configuration has issues:**\n\n")

		if len(validation.Errors) > 0 {
			responseBuilder.WriteString("**🔴 Errors (must fix):**\n")
			for _, error := range validation.Errors {
				responseBuilder.WriteString(fmt.Sprintf("• %s\n", error))
			}
			responseBuilder.WriteString("\n")
		}

		if len(validation.MissingFields) > 0 {
			responseBuilder.WriteString("**📋 Missing Required Fields:**\n")
			for _, field := range validation.MissingFields {
				responseBuilder.WriteString(fmt.Sprintf("• %s\n", field))
			}
			responseBuilder.WriteString("\n")
		}
	}

	if len(validation.Warnings) > 0 {
		responseBuilder.WriteString("**🟡 Warnings (recommended to fix):**\n")
		for _, warning := range validation.Warnings {
			responseBuilder.WriteString(fmt.Sprintf("• %s\n", warning))
		}
		responseBuilder.WriteString("\n")
	}

	if len(validation.Suggestions) > 0 {
		responseBuilder.WriteString("**💡 Suggestions:**\n")
		for _, suggestion := range validation.Suggestions {
			responseBuilder.WriteString(fmt.Sprintf("• %s\n", suggestion))
		}
		responseBuilder.WriteString("\n")
	}

	if len(validation.OptionalFields) > 0 {
		responseBuilder.WriteString("**🔧 Optional improvements:**\n")
		for _, field := range validation.OptionalFields {
			responseBuilder.WriteString(fmt.Sprintf("• Consider adding %s\n", field))
		}
	}

	return responseBuilder.String(), map[string]interface{}{
		"validation": validation,
		"config":     serverConfig,
	}
}

// showCurrentConfiguration displays the current server configuration
func (ca *ConfigUpdateAgent) showCurrentConfiguration(serverName string) (string, map[string]interface{}) {
	ca.logger.Info("Showing current configuration", zap.String("server", serverName))

	servers, err := ca.serverManager.GetAllServers()
	if err != nil {
		return fmt.Sprintf("❌ **Failed to get server configuration**: %v", err), map[string]interface{}{
			"error": err.Error(),
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
		return fmt.Sprintf("❌ **Server '%s' not found in configuration**", serverName), map[string]interface{}{
			"error": "server_not_found",
		}
	}

	var responseBuilder strings.Builder
	responseBuilder.WriteString(fmt.Sprintf("⚙️ **Current Configuration for %s**\n\n", serverName))

	// Format configuration as readable JSON
	configJSON, err := json.MarshalIndent(serverConfig, "", "  ")
	if err != nil {
		return fmt.Sprintf("❌ **Failed to format configuration**: %v", err), map[string]interface{}{
			"error": err.Error(),
		}
	}

	responseBuilder.WriteString("```json\n")
	responseBuilder.WriteString(string(configJSON))
	responseBuilder.WriteString("\n```\n\n")

	// Add key information
	responseBuilder.WriteString("**📋 Key Settings:**\n")
	if enabled, ok := serverConfig["enabled"].(bool); ok {
		status := "Disabled"
		if enabled {
			status = "Enabled"
		}
		responseBuilder.WriteString(fmt.Sprintf("• **Status**: %s\n", status))
	}

	if protocol, ok := serverConfig["protocol"].(string); ok {
		responseBuilder.WriteString(fmt.Sprintf("• **Protocol**: %s\n", protocol))
	}

	if env, ok := serverConfig["env"].(map[string]interface{}); ok && len(env) > 0 {
		responseBuilder.WriteString(fmt.Sprintf("• **Environment Variables**: %d configured\n", len(env)))
	}

	responseBuilder.WriteString("\n💡 **Need to update something?** Just tell me what you want to change!")

	return responseBuilder.String(), map[string]interface{}{
		"config": serverConfig,
	}
}

// handleEnvironmentVariables handles adding or removing environment variables
func (ca *ConfigUpdateAgent) handleEnvironmentVariables(serverName, userMessage, action string) (string, map[string]interface{}) {
	ca.logger.Info("Handling environment variables",
		zap.String("server", serverName),
		zap.String("action", action))

	if action == "add" {
		return ca.addEnvironmentVariable(serverName, userMessage)
	} else if action == "remove" {
		return ca.removeEnvironmentVariable(serverName, userMessage)
	}

	return "❌ Invalid environment variable action", map[string]interface{}{
		"error": "invalid_action",
	}
}

// addEnvironmentVariable adds an environment variable
func (ca *ConfigUpdateAgent) addEnvironmentVariable(serverName, userMessage string) (string, map[string]interface{}) {
	// Try to extract key=value from the message
	// This is a simple extraction - in a real implementation, you might want more sophisticated parsing
	envRegex := regexp.MustCompile(`(\w+)\s*=\s*([^\s]+)`)
	matches := envRegex.FindStringSubmatch(userMessage)

	if len(matches) < 3 {
		return fmt.Sprintf(`❌ **Please specify the environment variable in the format:**

` + "`" + `KEY=value` + "`" + `

**Examples:**
• "Add env API_KEY=your-api-key-here"
• "Set environment TIMEOUT=30"
• "Add env DATABASE_URL=postgresql://..."

**Common Variables:**
• API_KEY - API authentication key
• TIMEOUT - Request timeout in seconds
• DEBUG - Enable debug mode (true/false)
• BASE_URL - Base URL for API endpoints`), map[string]interface{}{
			"guidance": true,
		}
	}

	key := matches[1]
	value := matches[2]

	// Load current configuration
	configPath := ca.serverManager.GetConfigPath()
	if configPath == "" {
		return "❌ **Configuration path not available**", map[string]interface{}{
			"error": "no_config_path",
		}
	}

	globalConfig, err := config.LoadFromFile(configPath)
	if err != nil {
		return fmt.Sprintf("❌ **Failed to load configuration**: %v", err), map[string]interface{}{
			"error": err.Error(),
		}
	}

	// Find and update the server
	found := false
	for i, server := range globalConfig.Servers {
		if server.Name == serverName {
			if server.Env == nil {
				server.Env = make(map[string]string)
			}
			server.Env[key] = value
			globalConfig.Servers[i] = server
			found = true
			break
		}
	}

	if !found {
		return fmt.Sprintf("❌ **Server '%s' not found in configuration**", serverName), map[string]interface{}{
			"error": "server_not_found",
		}
	}

	// Save the updated configuration
	if err := config.SaveToFile(globalConfig, configPath); err != nil {
		return fmt.Sprintf("❌ **Failed to save configuration**: %v", err), map[string]interface{}{
			"error": err.Error(),
		}
	}

	// Reload configuration
	if err := ca.serverManager.ReloadConfiguration(); err != nil {
		ca.logger.Warn("Failed to reload configuration", zap.Error(err))
	}

	return fmt.Sprintf(`✅ **Environment variable added successfully!**

**Added:**
• **%s** = `+"`%s`"+`

The configuration has been updated and reloaded. The server will use this environment variable on its next restart.

💡 **Want to restart the server now?** Ask the coordinator to restart it!`, key, value), map[string]interface{}{
		"added_env": map[string]string{key: value},
		"success":   true,
	}
}

// removeEnvironmentVariable removes an environment variable
func (ca *ConfigUpdateAgent) removeEnvironmentVariable(serverName, userMessage string) (string, map[string]interface{}) {
	// Extract the key to remove
	words := strings.Fields(userMessage)
	var keyToRemove string

	for _, word := range words {
		if len(word) > 2 && !ca.isCommonWord(strings.ToLower(word)) {
			keyToRemove = word
			break
		}
	}

	if keyToRemove == "" {
		return fmt.Sprintf(`❌ **Please specify which environment variable to remove:**

**Examples:**
• "Remove env API_KEY"
• "Delete environment TIMEOUT"
• "Unset DEBUG"

**Current environment variables:**
Use "show config" to see what's currently set.`), map[string]interface{}{
			"guidance": true,
		}
	}

	// Load current configuration
	configPath := ca.serverManager.GetConfigPath()
	if configPath == "" {
		return "❌ **Configuration path not available**", map[string]interface{}{
			"error": "no_config_path",
		}
	}

	globalConfig, err := config.LoadFromFile(configPath)
	if err != nil {
		return fmt.Sprintf("❌ **Failed to load configuration**: %v", err), map[string]interface{}{
			"error": err.Error(),
		}
	}

	// Find and update the server
	found := false
	removed := false
	for i, server := range globalConfig.Servers {
		if server.Name == serverName {
			if server.Env != nil {
				if _, exists := server.Env[keyToRemove]; exists {
					delete(server.Env, keyToRemove)
					globalConfig.Servers[i] = server
					removed = true
				}
			}
			found = true
			break
		}
	}

	if !found {
		return fmt.Sprintf("❌ **Server '%s' not found in configuration**", serverName), map[string]interface{}{
			"error": "server_not_found",
		}
	}

	if !removed {
		return fmt.Sprintf("❌ **Environment variable '%s' not found**", keyToRemove), map[string]interface{}{
			"error": "env_not_found",
		}
	}

	// Save the updated configuration
	if err := config.SaveToFile(globalConfig, configPath); err != nil {
		return fmt.Sprintf("❌ **Failed to save configuration**: %v", err), map[string]interface{}{
			"error": err.Error(),
		}
	}

	// Reload configuration
	if err := ca.serverManager.ReloadConfiguration(); err != nil {
		ca.logger.Warn("Failed to reload configuration", zap.Error(err))
	}

	return fmt.Sprintf(`✅ **Environment variable removed successfully!**

**Removed:**
• **%s**

The configuration has been updated and reloaded. The change will take effect on the server's next restart.

💡 **Want to restart the server now?** Ask the coordinator to restart it!`, keyToRemove), map[string]interface{}{
		"removed_env": keyToRemove,
		"success":     true,
	}
}

// handleConfigurationUpdate handles general configuration updates
func (ca *ConfigUpdateAgent) handleConfigurationUpdate(serverName, userMessage string) (string, map[string]interface{}) {
	return fmt.Sprintf(`⚙️ **Configuration Update for %s**

I can help you update various configuration settings:

**🔧 Available Updates:**

**1. Environment Variables**
• Add: "Add env API_KEY=your-key"
• Remove: "Remove env API_KEY"

**2. Basic Settings**
• Enable/Disable: Ask coordinator to enable/disable
• Protocol: Currently requires manual editing

**3. Advanced Settings**
• Working directory
• Command arguments
• Timeout settings

**💡 What would you like to update?**

**Examples:**
• "Add environment variable API_KEY=abc123"
• "Remove environment variable DEBUG"
• "Show me current configuration"
• "Validate my configuration"

For complex updates, I can guide you through manual editing.`, serverName), map[string]interface{}{
		"guidance_type": "configuration_update",
		"available_updates": []string{
			"environment_variables",
			"basic_settings",
			"validation",
		},
	}
}

// backupConfiguration creates a backup of the current configuration
func (ca *ConfigUpdateAgent) backupConfiguration(serverName string) (string, map[string]interface{}) {
	ca.logger.Info("Backing up configuration", zap.String("server", serverName))

	configPath := ca.serverManager.GetConfigPath()
	if configPath == "" {
		return "❌ **Configuration path not available**", map[string]interface{}{
			"error": "no_config_path",
		}
	}

	// Create backup filename with timestamp
	timestamp := time.Now().Format("20060102-150405")
	backupPath := fmt.Sprintf("%s.backup.%s", configPath, timestamp)

	// Copy configuration file
	configData, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Sprintf("❌ **Failed to read configuration**: %v", err), map[string]interface{}{
			"error": err.Error(),
		}
	}

	if err := os.WriteFile(backupPath, configData, 0644); err != nil {
		return fmt.Sprintf("❌ **Failed to create backup**: %v", err), map[string]interface{}{
			"error": err.Error(),
		}
	}

	return fmt.Sprintf(`✅ **Configuration backup created!**

**Backup Location:**
`+"`%s`"+`

**Backup Time:** %s

This backup contains the complete configuration for all servers. You can restore it later if needed.

💡 **To restore:** Say "restore config" and I'll help you!`, backupPath, time.Now().Format("January 2, 2006 at 3:04 PM")), map[string]interface{}{
		"backup_path": backupPath,
		"timestamp":   timestamp,
		"success":     true,
	}
}

// restoreConfiguration provides guidance for restoring configuration
func (ca *ConfigUpdateAgent) restoreConfiguration(serverName string) (string, map[string]interface{}) {
	ca.logger.Info("Configuration restore requested", zap.String("server", serverName))

	configPath := ca.serverManager.GetConfigPath()
	if configPath == "" {
		return "❌ **Configuration path not available**", map[string]interface{}{
			"error": "no_config_path",
		}
	}

	// List available backups
	configDir := filepath.Dir(configPath)
	configName := filepath.Base(configPath)

	pattern := fmt.Sprintf("%s.backup.*", configName)
	matches, err := filepath.Glob(filepath.Join(configDir, pattern))
	if err != nil {
		return fmt.Sprintf("❌ **Failed to search for backups**: %v", err), map[string]interface{}{
			"error": err.Error(),
		}
	}

	var responseBuilder strings.Builder
	responseBuilder.WriteString("🔄 **Configuration Restore**\n\n")

	if len(matches) == 0 {
		responseBuilder.WriteString("❌ **No backup files found.**\n\n")
		responseBuilder.WriteString("Create a backup first using \"backup config\" before making changes.\n")
	} else {
		responseBuilder.WriteString(fmt.Sprintf("✅ **Found %d backup(s):**\n\n", len(matches)))

		for i, match := range matches {
			if i >= 5 { // Show max 5 backups
				responseBuilder.WriteString("... and more\n")
				break
			}

			fileName := filepath.Base(match)
			// Extract timestamp from filename
			parts := strings.Split(fileName, ".")
			if len(parts) >= 3 {
				timestamp := parts[len(parts)-1]
				responseBuilder.WriteString(fmt.Sprintf("• **%s** - %s\n", fileName, timestamp))
			} else {
				responseBuilder.WriteString(fmt.Sprintf("• **%s**\n", fileName))
			}
		}

		responseBuilder.WriteString("\n⚠️ **Manual Restore Required**\n")
		responseBuilder.WriteString("To restore a backup:\n")
		responseBuilder.WriteString(fmt.Sprintf("1. Stop mcpproxy server\n"))
		responseBuilder.WriteString(fmt.Sprintf("2. Copy backup file to: `%s`\n", configPath))
		responseBuilder.WriteString("3. Restart mcpproxy server\n\n")
		responseBuilder.WriteString("💡 **Need help with any step?** Just ask!")
	}

	return responseBuilder.String(), map[string]interface{}{
		"backups":     matches,
		"config_path": configPath,
	}
}

// optimizeConfiguration provides configuration optimization suggestions
func (ca *ConfigUpdateAgent) optimizeConfiguration(serverName string) (string, map[string]interface{}) {
	ca.logger.Info("Optimizing configuration", zap.String("server", serverName))

	// Get current configuration for analysis
	servers, err := ca.serverManager.GetAllServers()
	if err != nil {
		return fmt.Sprintf("❌ **Failed to get server configuration**: %v", err), map[string]interface{}{
			"error": err.Error(),
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
		return fmt.Sprintf("❌ **Server '%s' not found in configuration**", serverName), map[string]interface{}{
			"error": "server_not_found",
		}
	}

	// Analyze configuration for optimization opportunities
	optimizations := ca.analyzeConfigurationForOptimizations(serverConfig)

	var responseBuilder strings.Builder
	responseBuilder.WriteString(fmt.Sprintf("🚀 **Configuration Optimization for %s**\n\n", serverName))

	if len(optimizations) == 0 {
		responseBuilder.WriteString("✅ **Configuration looks good!**\n\n")
		responseBuilder.WriteString("No obvious optimizations detected. Your configuration follows best practices.\n")
	} else {
		responseBuilder.WriteString("💡 **Optimization Opportunities:**\n\n")

		for i, opt := range optimizations {
			if i >= 8 { // Limit to 8 suggestions
				break
			}
			responseBuilder.WriteString(fmt.Sprintf("**%d. %s**\n", i+1, opt))
		}
	}

	responseBuilder.WriteString("\n**📋 Best Practices:**\n")
	responseBuilder.WriteString("• Use environment variables for sensitive data\n")
	responseBuilder.WriteString("• Enable quarantine for new servers\n")
	responseBuilder.WriteString("• Set appropriate timeouts\n")
	responseBuilder.WriteString("• Use descriptive server names\n")
	responseBuilder.WriteString("• Keep configuration backed up\n")

	responseBuilder.WriteString("\n💡 **Want to apply any of these optimizations?** Just ask!")

	return responseBuilder.String(), map[string]interface{}{
		"optimizations": optimizations,
		"config":        serverConfig,
	}
}

// provideConfigurationGuidance provides general configuration guidance
func (ca *ConfigUpdateAgent) provideConfigurationGuidance(serverName string) (string, map[string]interface{}) {
	response := fmt.Sprintf(`⚙️ **Configuration Management for %s**

I can help you manage your server configuration:

**🔍 Configuration Analysis:**
• "Show current configuration"
• "Validate my configuration"
• "Check for optimization opportunities"

**🛠️ Configuration Updates:**
• "Add environment variable KEY=value"
• "Remove environment variable KEY"
• "Update timeout settings"

**💾 Backup & Restore:**
• "Backup my configuration"
• "Show available backups"
• "Help me restore configuration"

**🚀 Optimization:**
• "Optimize my configuration"
• "Show best practices"
• "Security recommendations"

**💡 Example Requests:**
• "Add env API_KEY=abc123"
• "Show me my current config"
• "Validate configuration"
• "Backup config before changes"
• "What can I optimize?"

**🔒 Security Tips:**
• Use environment variables for API keys
• Don't store secrets in plain text
• Keep backups of working configurations
• Test changes in a safe environment

What configuration help do you need?`, serverName)

	return response, map[string]interface{}{
		"guidance_type": "configuration_management",
		"available_actions": []string{
			"validate",
			"show_config",
			"add_env",
			"remove_env",
			"backup",
			"restore",
			"optimize",
		},
	}
}

// performConfigValidation validates a server configuration
func (ca *ConfigUpdateAgent) performConfigValidation(serverConfig map[string]interface{}) *ConfigValidationResult {
	validation := &ConfigValidationResult{
		Valid:     true,
		Errors:    []string{},
		Warnings:  []string{},
		Suggestions: []string{},
		MissingFields: []string{},
		OptionalFields: []string{},
	}

	// Check required fields
	requiredFields := []string{"name", "protocol"}
	for _, field := range requiredFields {
		if _, exists := serverConfig[field]; !exists {
			validation.Valid = false
			validation.Errors = append(validation.Errors, fmt.Sprintf("Missing required field: %s", field))
			validation.MissingFields = append(validation.MissingFields, field)
		}
	}

	// Check protocol-specific requirements
	if protocol, ok := serverConfig["protocol"].(string); ok {
		if protocol == "stdio" {
			if _, exists := serverConfig["command"]; !exists {
				validation.Valid = false
				validation.Errors = append(validation.Errors, "stdio protocol requires 'command' field")
				validation.MissingFields = append(validation.MissingFields, "command")
			}
		} else if protocol == "http" || protocol == "sse" {
			if _, exists := serverConfig["url"]; !exists {
				validation.Valid = false
				validation.Errors = append(validation.Errors, fmt.Sprintf("%s protocol requires 'url' field", protocol))
				validation.MissingFields = append(validation.MissingFields, "url")
			}
		}
	}

	// Check for common issues
	if enabled, ok := serverConfig["enabled"].(bool); ok && !enabled {
		validation.Warnings = append(validation.Warnings, "Server is currently disabled")
	}

	if quarantined, ok := serverConfig["quarantined"].(bool); ok && quarantined {
		validation.Warnings = append(validation.Warnings, "Server is quarantined for security")
	}

	// Check for optional improvements
	if _, exists := serverConfig["env"]; !exists {
		validation.OptionalFields = append(validation.OptionalFields, "environment variables")
	}

	if _, exists := serverConfig["working_dir"]; !exists {
		validation.OptionalFields = append(validation.OptionalFields, "working directory")
	}

	// Generate suggestions
	if len(validation.Errors) == 0 && len(validation.Warnings) == 0 {
		validation.Suggestions = append(validation.Suggestions, "Configuration looks good!")
	} else {
		validation.Suggestions = append(validation.Suggestions, "Review and fix identified issues")
	}

	return validation
}

// analyzeConfigurationForOptimizations analyzes configuration for optimization opportunities
func (ca *ConfigUpdateAgent) analyzeConfigurationForOptimizations(serverConfig map[string]interface{}) []string {
	var optimizations []string

	// Check for potential improvements
	if env, exists := serverConfig["env"]; !exists || env == nil {
		optimizations = append(optimizations, "Consider adding environment variables for configuration")
	}

	if _, exists := serverConfig["working_dir"]; !exists {
		optimizations = append(optimizations, "Add working_dir to control where the server runs")
	}

	if quarantined, ok := serverConfig["quarantined"].(bool); !ok || !quarantined {
		optimizations = append(optimizations, "Consider quarantining new servers for security review")
	}

	if protocol, ok := serverConfig["protocol"].(string); ok && protocol == "stdio" {
		if args, exists := serverConfig["args"]; !exists || args == nil {
			optimizations = append(optimizations, "Consider adding command arguments for better control")
		}
	}

	// Check for security improvements
	if env, ok := serverConfig["env"].(map[string]interface{}); ok {
		for key, value := range env {
			if strings.Contains(strings.ToLower(key), "key") || strings.Contains(strings.ToLower(key), "token") {
				if str, ok := value.(string); ok && len(str) < 10 {
					optimizations = append(optimizations, fmt.Sprintf("Environment variable %s appears to have a short value - verify it's correct", key))
				}
			}
		}
	}

	return optimizations
}

// isCommonWord checks if a word is a common English word that shouldn't be treated as an env var name
func (ca *ConfigUpdateAgent) isCommonWord(word string) bool {
	commonWords := []string{
		"the", "and", "or", "but", "in", "on", "at", "to", "for", "of", "with", "by",
		"remove", "delete", "add", "set", "env", "environment", "variable", "config",
		"configuration", "update", "change", "modify", "from", "this", "that", "it",
	}

	for _, common := range commonWords {
		if word == common {
			return true
		}
	}
	return false
}
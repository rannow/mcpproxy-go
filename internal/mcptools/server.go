// Package mcptools provides MCP tools for managing mcpproxy servers
package mcptools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	"mcpproxy-go/internal/config"
	"mcpproxy-go/internal/upstream"
	"mcpproxy-go/internal/upstream/types"
)

// MCPToolsServer provides MCP tools for server management
type MCPToolsServer struct {
	manager      *upstream.Manager
	config       *config.Config
	configLoader ConfigLoader
	logger       *zap.Logger
	mu           sync.RWMutex
}

// ConfigLoader interface for loading and saving configuration
type ConfigLoader interface {
	SaveConfig(cfg *config.Config) error
	ReloadConfig() (*config.Config, error)
}

// NewMCPToolsServer creates a new MCP tools server
func NewMCPToolsServer(manager *upstream.Manager, cfg *config.Config, loader ConfigLoader, logger *zap.Logger) *MCPToolsServer {
	return &MCPToolsServer{
		manager:      manager,
		config:       cfg,
		configLoader: loader,
		logger:       logger,
	}
}

// ToolDefinition represents an MCP tool definition
type ToolDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

// GetTools returns all available MCP management tools
func (s *MCPToolsServer) GetTools() []ToolDefinition {
	return []ToolDefinition{
		s.addServerTool(),
		s.removeServerTool(),
		s.listServersTool(),
		s.getServerTool(),
		s.updateServerTool(),
		s.connectServerTool(),
		s.disconnectServerTool(),
		s.reconnectServerTool(),
		s.testConnectionTool(),
		s.testProtocolTool(),
		s.healthCheckTool(),
		s.getConfigTool(),
		s.validateConfigTool(),
		s.diagnoseTool(),
		s.getLogsTool(),
		// Tool testing capabilities
		s.listServerToolsTool(),
		s.callServerToolTool(),
		s.testServerToolTool(),
	}
}

// CallTool executes a management tool
func (s *MCPToolsServer) CallTool(ctx context.Context, name string, args map[string]interface{}) (interface{}, error) {
	s.logger.Debug("CallTool invoked", zap.String("tool", name), zap.Any("args", args))

	switch name {
	case "mcpproxy_add_server":
		return s.handleAddServer(ctx, args)
	case "mcpproxy_remove_server":
		return s.handleRemoveServer(ctx, args)
	case "mcpproxy_list_servers":
		return s.handleListServers(ctx, args)
	case "mcpproxy_get_server":
		return s.handleGetServer(ctx, args)
	case "mcpproxy_update_server":
		return s.handleUpdateServer(ctx, args)
	case "mcpproxy_connect_server":
		return s.handleConnectServer(ctx, args)
	case "mcpproxy_disconnect_server":
		return s.handleDisconnectServer(ctx, args)
	case "mcpproxy_reconnect_server":
		return s.handleReconnectServer(ctx, args)
	case "mcpproxy_test_connection":
		return s.handleTestConnection(ctx, args)
	case "mcpproxy_test_protocol":
		return s.handleTestProtocol(ctx, args)
	case "mcpproxy_health_check":
		return s.handleHealthCheck(ctx, args)
	case "mcpproxy_get_config":
		return s.handleGetConfig(ctx, args)
	case "mcpproxy_validate_config":
		return s.handleValidateConfig(ctx, args)
	case "mcpproxy_diagnose":
		return s.handleDiagnose(ctx, args)
	case "mcpproxy_get_logs":
		return s.handleGetLogs(ctx, args)
	case "mcpproxy_list_server_tools":
		return s.handleListServerTools(ctx, args)
	case "mcpproxy_call_server_tool":
		return s.handleCallServerTool(ctx, args)
	case "mcpproxy_test_server_tool":
		return s.handleTestServerTool(ctx, args)
	default:
		return nil, fmt.Errorf("unknown tool: %s", name)
	}
}

// Tool Definitions

func (s *MCPToolsServer) addServerTool() ToolDefinition {
	return ToolDefinition{
		Name:        "mcpproxy_add_server",
		Description: "Add a new MCP server to mcpproxy configuration",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"name": map[string]interface{}{
					"type":        "string",
					"description": "Unique server identifier",
				},
				"protocol": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"stdio", "http", "sse", "streamable-http"},
					"description": "Transport protocol",
				},
				"command": map[string]interface{}{
					"type":        "string",
					"description": "Command for stdio servers (e.g., 'npx', 'python', 'uvx')",
				},
				"args": map[string]interface{}{
					"type":        "array",
					"items":       map[string]interface{}{"type": "string"},
					"description": "Command arguments for stdio servers",
				},
				"url": map[string]interface{}{
					"type":        "string",
					"description": "URL for HTTP/SSE servers",
				},
				"env": map[string]interface{}{
					"type":                 "object",
					"additionalProperties": map[string]interface{}{"type": "string"},
					"description":          "Environment variables",
				},
				"headers": map[string]interface{}{
					"type":                 "object",
					"additionalProperties": map[string]interface{}{"type": "string"},
					"description":          "HTTP headers for HTTP servers",
				},
				"startup_mode": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"active", "disabled", "lazy_loading"},
					"default":     "active",
					"description": "Server startup behavior",
				},
				"description": map[string]interface{}{
					"type":        "string",
					"description": "Human-readable server description",
				},
				"working_dir": map[string]interface{}{
					"type":        "string",
					"description": "Working directory for stdio servers",
				},
				"test_before_add": map[string]interface{}{
					"type":        "boolean",
					"default":     true,
					"description": "Test connection before adding",
				},
			},
			"required": []string{"name", "protocol"},
		},
	}
}

func (s *MCPToolsServer) removeServerTool() ToolDefinition {
	return ToolDefinition{
		Name:        "mcpproxy_remove_server",
		Description: "Remove an MCP server from mcpproxy",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"name": map[string]interface{}{
					"type":        "string",
					"description": "Server name to remove",
				},
				"force": map[string]interface{}{
					"type":        "boolean",
					"default":     false,
					"description": "Force removal even if server is connected",
				},
			},
			"required": []string{"name"},
		},
	}
}

func (s *MCPToolsServer) listServersTool() ToolDefinition {
	return ToolDefinition{
		Name:        "mcpproxy_list_servers",
		Description: "List all configured MCP servers with status",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"filter": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"all", "connected", "disconnected", "disabled", "error"},
					"default":     "all",
					"description": "Filter servers by status",
				},
				"include_tools": map[string]interface{}{
					"type":        "boolean",
					"default":     false,
					"description": "Include tool count for each server",
				},
			},
		},
	}
}

func (s *MCPToolsServer) getServerTool() ToolDefinition {
	return ToolDefinition{
		Name:        "mcpproxy_get_server",
		Description: "Get detailed information about a specific server",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"name": map[string]interface{}{
					"type":        "string",
					"description": "Server name",
				},
			},
			"required": []string{"name"},
		},
	}
}

func (s *MCPToolsServer) updateServerTool() ToolDefinition {
	return ToolDefinition{
		Name:        "mcpproxy_update_server",
		Description: "Update server configuration",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"name": map[string]interface{}{
					"type":        "string",
					"description": "Server name to update",
				},
				"updates": map[string]interface{}{
					"type":        "object",
					"description": "Fields to update",
				},
				"reconnect": map[string]interface{}{
					"type":        "boolean",
					"default":     false,
					"description": "Reconnect after update if config changed",
				},
			},
			"required": []string{"name", "updates"},
		},
	}
}

func (s *MCPToolsServer) connectServerTool() ToolDefinition {
	return ToolDefinition{
		Name:        "mcpproxy_connect_server",
		Description: "Connect to a specific MCP server",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"name": map[string]interface{}{
					"type":        "string",
					"description": "Server name to connect",
				},
				"timeout": map[string]interface{}{
					"type":        "integer",
					"default":     30,
					"description": "Connection timeout in seconds",
				},
			},
			"required": []string{"name"},
		},
	}
}

func (s *MCPToolsServer) disconnectServerTool() ToolDefinition {
	return ToolDefinition{
		Name:        "mcpproxy_disconnect_server",
		Description: "Disconnect from a specific MCP server",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"name": map[string]interface{}{
					"type":        "string",
					"description": "Server name to disconnect",
				},
			},
			"required": []string{"name"},
		},
	}
}

func (s *MCPToolsServer) reconnectServerTool() ToolDefinition {
	return ToolDefinition{
		Name:        "mcpproxy_reconnect_server",
		Description: "Disconnect and reconnect to a server",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"name": map[string]interface{}{
					"type":        "string",
					"description": "Server name to reconnect",
				},
			},
			"required": []string{"name"},
		},
	}
}

func (s *MCPToolsServer) testConnectionTool() ToolDefinition {
	return ToolDefinition{
		Name:        "mcpproxy_test_connection",
		Description: "Test connectivity to an MCP server",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"name": map[string]interface{}{
					"type":        "string",
					"description": "Existing server name to test",
				},
				"config": map[string]interface{}{
					"type":        "object",
					"description": "Or provide ad-hoc server config to test",
				},
				"timeout": map[string]interface{}{
					"type":        "integer",
					"default":     10,
					"description": "Test timeout in seconds",
				},
			},
		},
	}
}

func (s *MCPToolsServer) testProtocolTool() ToolDefinition {
	return ToolDefinition{
		Name:        "mcpproxy_test_protocol",
		Description: "Verify MCP protocol compliance",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"name": map[string]interface{}{
					"type":        "string",
					"description": "Server name to test",
				},
				"tests": map[string]interface{}{
					"type":        "array",
					"items":       map[string]interface{}{"type": "string"},
					"default":     []string{"initialize", "tools_list"},
					"description": "Protocol tests to run",
				},
			},
			"required": []string{"name"},
		},
	}
}

func (s *MCPToolsServer) healthCheckTool() ToolDefinition {
	return ToolDefinition{
		Name:        "mcpproxy_health_check",
		Description: "Run health check on all or specific servers",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"servers": map[string]interface{}{
					"type":        "array",
					"items":       map[string]interface{}{"type": "string"},
					"description": "Server names to check (empty = all)",
				},
				"fix_issues": map[string]interface{}{
					"type":        "boolean",
					"default":     false,
					"description": "Attempt to fix issues (reconnect failed servers)",
				},
			},
		},
	}
}

func (s *MCPToolsServer) getConfigTool() ToolDefinition {
	return ToolDefinition{
		Name:        "mcpproxy_get_config",
		Description: "Get mcpproxy configuration",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"section": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"all", "servers", "logging", "security", "docker", "llm"},
					"default":     "all",
					"description": "Configuration section to retrieve",
				},
			},
		},
	}
}

func (s *MCPToolsServer) validateConfigTool() ToolDefinition {
	return ToolDefinition{
		Name:        "mcpproxy_validate_config",
		Description: "Validate mcpproxy configuration file",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"config_path": map[string]interface{}{
					"type":        "string",
					"description": "Path to config file (default: auto-detect)",
				},
				"strict": map[string]interface{}{
					"type":        "boolean",
					"default":     false,
					"description": "Enable strict validation",
				},
			},
		},
	}
}

func (s *MCPToolsServer) diagnoseTool() ToolDefinition {
	return ToolDefinition{
		Name:        "mcpproxy_diagnose",
		Description: "Run diagnostics on mcpproxy setup",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"checks": map[string]interface{}{
					"type":        "array",
					"items":       map[string]interface{}{"type": "string"},
					"default":     []string{"config", "servers"},
					"description": "Diagnostic checks to run",
				},
				"server": map[string]interface{}{
					"type":        "string",
					"description": "Focus diagnostics on specific server",
				},
			},
		},
	}
}

func (s *MCPToolsServer) getLogsTool() ToolDefinition {
	return ToolDefinition{
		Name:        "mcpproxy_get_logs",
		Description: "Retrieve mcpproxy logs",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"lines": map[string]interface{}{
					"type":        "integer",
					"default":     100,
					"description": "Number of log lines to retrieve",
				},
				"level": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"debug", "info", "warn", "error"},
					"description": "Filter by log level",
				},
				"server": map[string]interface{}{
					"type":        "string",
					"description": "Filter by server name",
				},
			},
		},
	}
}

// Tool Testing Tool Definitions

func (s *MCPToolsServer) listServerToolsTool() ToolDefinition {
	return ToolDefinition{
		Name:        "mcpproxy_list_server_tools",
		Description: "List all tools available on a specific MCP server with their schemas",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"server": map[string]interface{}{
					"type":        "string",
					"description": "Name of the MCP server to list tools from",
				},
				"filter": map[string]interface{}{
					"type":        "string",
					"description": "Optional filter to match tool names (supports wildcards)",
				},
				"include_schema": map[string]interface{}{
					"type":        "boolean",
					"default":    true,
					"description": "Include full input schema for each tool",
				},
			},
			"required": []string{"server"},
		},
	}
}

func (s *MCPToolsServer) callServerToolTool() ToolDefinition {
	return ToolDefinition{
		Name:        "mcpproxy_call_server_tool",
		Description: "Call a specific tool on an MCP server with given arguments",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"server": map[string]interface{}{
					"type":        "string",
					"description": "Name of the MCP server",
				},
				"tool": map[string]interface{}{
					"type":        "string",
					"description": "Name of the tool to call",
				},
				"arguments": map[string]interface{}{
					"type":        "object",
					"description": "Arguments to pass to the tool (as JSON object)",
				},
				"timeout": map[string]interface{}{
					"type":        "integer",
					"default":    30,
					"description": "Timeout in seconds for the tool call",
				},
			},
			"required": []string{"server", "tool"},
		},
	}
}

func (s *MCPToolsServer) testServerToolTool() ToolDefinition {
	return ToolDefinition{
		Name:        "mcpproxy_test_server_tool",
		Description: "Test a tool on an MCP server with test cases and validate results",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"server": map[string]interface{}{
					"type":        "string",
					"description": "Name of the MCP server",
				},
				"tool": map[string]interface{}{
					"type":        "string",
					"description": "Name of the tool to test",
				},
				"test_cases": map[string]interface{}{
					"type":        "array",
					"description": "Array of test cases to run",
					"items": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"name": map[string]interface{}{
								"type":        "string",
								"description": "Test case name",
							},
							"arguments": map[string]interface{}{
								"type":        "object",
								"description": "Arguments to pass to the tool",
							},
							"expect_error": map[string]interface{}{
								"type":        "boolean",
								"description": "Whether the tool should return an error",
							},
							"expect_contains": map[string]interface{}{
								"type":        "string",
								"description": "String that should be present in the result",
							},
							"expect_not_contains": map[string]interface{}{
								"type":        "string",
								"description": "String that should NOT be present in the result",
							},
						},
					},
				},
				"timeout": map[string]interface{}{
					"type":        "integer",
					"default":    30,
					"description": "Timeout in seconds for each test case",
				},
				"stop_on_failure": map[string]interface{}{
					"type":        "boolean",
					"default":    false,
					"description": "Stop testing after first failure",
				},
			},
			"required": []string{"server", "tool", "test_cases"},
		},
	}
}

// Tool Handlers

func (s *MCPToolsServer) handleAddServer(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	name, _ := args["name"].(string)
	protocol, _ := args["protocol"].(string)

	if name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if protocol == "" {
		return nil, fmt.Errorf("protocol is required")
	}

	// Protocol-specific validation
	switch protocol {
	case "stdio":
		if cmd, ok := args["command"].(string); !ok || cmd == "" {
			return nil, fmt.Errorf("command is required for stdio protocol")
		}
	case "http", "sse", "streamable-http":
		if url, ok := args["url"].(string); !ok || url == "" {
			return nil, fmt.Errorf("url is required for %s protocol", protocol)
		}
	}

	// Check if manager is available
	if s.manager == nil {
		return nil, fmt.Errorf("server manager not available")
	}

	// Check if server already exists
	servers := s.manager.ListServers()
	if _, exists := servers[name]; exists {
		return nil, fmt.Errorf("server '%s' already exists", name)
	}

	// Build server config
	serverConfig := &config.ServerConfig{
		Name:        name,
		Protocol:    protocol,
		StartupMode: "active",
		Created:     time.Now(),
	}

	// Set optional fields
	if cmd, ok := args["command"].(string); ok {
		serverConfig.Command = cmd
	}
	if argsSlice, ok := args["args"].([]interface{}); ok {
		for _, a := range argsSlice {
			if str, ok := a.(string); ok {
				serverConfig.Args = append(serverConfig.Args, str)
			}
		}
	}
	if url, ok := args["url"].(string); ok {
		serverConfig.URL = url
	}
	if env, ok := args["env"].(map[string]interface{}); ok {
		serverConfig.Env = make(map[string]string)
		for k, v := range env {
			if str, ok := v.(string); ok {
				serverConfig.Env[k] = str
			}
		}
	}
	if headers, ok := args["headers"].(map[string]interface{}); ok {
		serverConfig.Headers = make(map[string]string)
		for k, v := range headers {
			if str, ok := v.(string); ok {
				serverConfig.Headers[k] = str
			}
		}
	}
	if mode, ok := args["startup_mode"].(string); ok {
		serverConfig.StartupMode = mode
	}
	if desc, ok := args["description"].(string); ok {
		serverConfig.Description = desc
	}
	if wd, ok := args["working_dir"].(string); ok {
		serverConfig.WorkingDir = wd
	}

	// Validate protocol-specific requirements
	if protocol == "stdio" && serverConfig.Command == "" {
		return nil, fmt.Errorf("command is required for stdio protocol")
	}
	if (protocol == "http" || protocol == "sse" || protocol == "streamable-http") && serverConfig.URL == "" {
		return nil, fmt.Errorf("url is required for %s protocol", protocol)
	}

	// Test connection before adding (if requested)
	testBeforeAdd := true
	if test, ok := args["test_before_add"].(bool); ok {
		testBeforeAdd = test
	}

	if testBeforeAdd {
		result, err := s.testServerConfig(ctx, serverConfig, 10*time.Second)
		if err != nil {
			return map[string]interface{}{
				"success": false,
				"error":   fmt.Sprintf("Connection test failed: %v", err),
				"test":    result,
			}, nil
		}
	}

	// Add server to manager
	if err := s.manager.AddServerConfig(name, serverConfig); err != nil {
		return nil, fmt.Errorf("failed to add server: %w", err)
	}

	// Update config and save
	s.mu.Lock()
	s.config.Servers = append(s.config.Servers, serverConfig)
	s.mu.Unlock()

	if s.configLoader != nil {
		if err := s.configLoader.SaveConfig(s.config); err != nil {
			s.logger.Warn("Failed to save config", zap.Error(err))
		}
	}

	// Connect if startup_mode is active
	var connectResult string
	if serverConfig.StartupMode == "active" {
		if err := s.manager.AddServer(name, serverConfig); err != nil {
			connectResult = fmt.Sprintf("Added but connection failed: %v", err)
		} else {
			connectResult = "Connected successfully"
		}
	} else {
		connectResult = "Added (not connected due to startup_mode)"
	}

	return map[string]interface{}{
		"success":        true,
		"name":           name,
		"protocol":       protocol,
		"startup_mode":   serverConfig.StartupMode,
		"connect_result": connectResult,
	}, nil
}

func (s *MCPToolsServer) handleRemoveServer(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	name, _ := args["name"].(string)
	force, _ := args["force"].(bool)

	if name == "" {
		return nil, fmt.Errorf("name is required")
	}

	// Check if server exists
	servers := s.manager.ListServers()
	serverConfig, exists := servers[name]
	if !exists {
		return nil, fmt.Errorf("server '%s' not found", name)
	}

	// Check if connected and force not set
	client, clientExists := s.manager.GetClient(name)
	if clientExists && client.IsConnected() && !force {
		return nil, fmt.Errorf("server '%s' is connected. Use force=true to remove", name)
	}

	// Remove from manager
	s.manager.RemoveServer(name)

	// Update config
	s.mu.Lock()
	var newServers []*config.ServerConfig
	for _, srv := range s.config.Servers {
		if srv.Name != name {
			newServers = append(newServers, srv)
		}
	}
	s.config.Servers = newServers
	s.mu.Unlock()

	// Save config
	if s.configLoader != nil {
		if err := s.configLoader.SaveConfig(s.config); err != nil {
			s.logger.Warn("Failed to save config", zap.Error(err))
		}
	}

	return map[string]interface{}{
		"success":        true,
		"name":           name,
		"was_connected":  clientExists && client.IsConnected(),
		"previous_state": serverConfig.StartupMode,
	}, nil
}

func (s *MCPToolsServer) handleListServers(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	filter, _ := args["filter"].(string)
	if filter == "" {
		filter = "all"
	}
	includeTools, _ := args["include_tools"].(bool)

	servers := s.manager.ListServers()
	result := make([]map[string]interface{}, 0)

	for name, cfg := range servers {
		client, clientExists := s.manager.GetClient(name)

		// Determine status
		var status string
		var lastError string
		if !clientExists {
			status = "unknown"
		} else if cfg.IsDisabled() {
			status = "disabled"
		} else if client.IsConnected() {
			status = "connected"
		} else if client.IsConnecting() {
			status = "connecting"
		} else {
			info := client.GetConnectionInfo()
			if info.State == types.StateError {
				status = "error"
				if info.LastError != nil {
					lastError = info.LastError.Error()
				}
			} else {
				status = "disconnected"
			}
		}

		// Apply filter
		if filter != "all" && status != filter {
			continue
		}

		serverInfo := map[string]interface{}{
			"name":         name,
			"protocol":     cfg.Protocol,
			"status":       status,
			"startup_mode": cfg.StartupMode,
		}

		if cfg.URL != "" {
			serverInfo["url"] = cfg.URL
		}
		if cfg.Command != "" {
			serverInfo["command"] = cfg.Command
		}
		if cfg.Description != "" {
			serverInfo["description"] = cfg.Description
		}
		if lastError != "" {
			serverInfo["last_error"] = lastError
		}

		// Include tool count if requested
		if includeTools && clientExists && client.IsConnected() {
			tools, err := client.ListTools(ctx)
			if err == nil {
				serverInfo["tool_count"] = len(tools)
			}
		} else if includeTools && cfg.ToolCount > 0 {
			serverInfo["tool_count"] = cfg.ToolCount
			serverInfo["tool_count_cached"] = true
		}

		result = append(result, serverInfo)
	}

	return map[string]interface{}{
		"servers":       result,
		"total":         len(result),
		"filter_applied": filter,
	}, nil
}

func (s *MCPToolsServer) handleGetServer(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	name, _ := args["name"].(string)
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}

	servers := s.manager.ListServers()
	cfg, exists := servers[name]
	if !exists {
		return nil, fmt.Errorf("server '%s' not found", name)
	}

	client, clientExists := s.manager.GetClient(name)

	result := map[string]interface{}{
		"name":           name,
		"protocol":       cfg.Protocol,
		"startup_mode":   cfg.StartupMode,
		"created":        cfg.Created,
		"updated":        cfg.Updated,
		"ever_connected": cfg.EverConnected,
	}

	if cfg.URL != "" {
		result["url"] = cfg.URL
	}
	if cfg.Command != "" {
		result["command"] = cfg.Command
		result["args"] = cfg.Args
	}
	if cfg.Description != "" {
		result["description"] = cfg.Description
	}
	if cfg.WorkingDir != "" {
		result["working_dir"] = cfg.WorkingDir
	}
	if len(cfg.Env) > 0 {
		// Mask sensitive values
		maskedEnv := make(map[string]string)
		for k := range cfg.Env {
			if strings.Contains(strings.ToLower(k), "key") ||
				strings.Contains(strings.ToLower(k), "secret") ||
				strings.Contains(strings.ToLower(k), "token") ||
				strings.Contains(strings.ToLower(k), "password") {
				maskedEnv[k] = "***"
			} else {
				maskedEnv[k] = cfg.Env[k]
			}
		}
		result["env"] = maskedEnv
	}
	if len(cfg.Headers) > 0 {
		// Mask authorization headers
		maskedHeaders := make(map[string]string)
		for k, v := range cfg.Headers {
			if strings.ToLower(k) == "authorization" {
				maskedHeaders[k] = "***"
			} else {
				maskedHeaders[k] = v
			}
		}
		result["headers"] = maskedHeaders
	}
	if !cfg.LastSuccessfulConnection.IsZero() {
		result["last_successful_connection"] = cfg.LastSuccessfulConnection
	}
	if cfg.ToolCount > 0 {
		result["tool_count"] = cfg.ToolCount
	}
	if cfg.HealthCheck {
		result["health_check"] = true
	}
	if cfg.AutoDisableReason != "" {
		result["auto_disable_reason"] = cfg.AutoDisableReason
	}

	// Connection state
	if clientExists {
		info := client.GetConnectionInfo()
		result["connection"] = map[string]interface{}{
			"state":        info.State.String(),
			"connected":    client.IsConnected(),
			"connecting":   client.IsConnecting(),
			"retry_count":  info.RetryCount,
			"should_retry": client.ShouldRetry(),
		}
		if info.LastError != nil {
			result["connection"].(map[string]interface{})["last_error"] = info.LastError.Error()
		}
		if info.ServerName != "" {
			result["connection"].(map[string]interface{})["server_name"] = info.ServerName
		}
		if info.ServerVersion != "" {
			result["connection"].(map[string]interface{})["server_version"] = info.ServerVersion
		}
	}

	return result, nil
}

func (s *MCPToolsServer) handleUpdateServer(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	name, _ := args["name"].(string)
	updates, _ := args["updates"].(map[string]interface{})
	reconnect, _ := args["reconnect"].(bool)

	if name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if updates == nil {
		return nil, fmt.Errorf("updates is required")
	}

	// Find server in config
	s.mu.Lock()
	var targetServer *config.ServerConfig
	for _, srv := range s.config.Servers {
		if srv.Name == name {
			targetServer = srv
			break
		}
	}
	s.mu.Unlock()

	if targetServer == nil {
		return nil, fmt.Errorf("server '%s' not found", name)
	}

	// Apply updates
	changed := false
	if desc, ok := updates["description"].(string); ok {
		targetServer.Description = desc
		changed = true
	}
	if mode, ok := updates["startup_mode"].(string); ok {
		targetServer.StartupMode = mode
		changed = true
	}
	if hc, ok := updates["health_check"].(bool); ok {
		targetServer.HealthCheck = hc
		changed = true
	}
	if env, ok := updates["env"].(map[string]interface{}); ok {
		targetServer.Env = make(map[string]string)
		for k, v := range env {
			if str, ok := v.(string); ok {
				targetServer.Env[k] = str
			}
		}
		changed = true
	}
	if headers, ok := updates["headers"].(map[string]interface{}); ok {
		targetServer.Headers = make(map[string]string)
		for k, v := range headers {
			if str, ok := v.(string); ok {
				targetServer.Headers[k] = str
			}
		}
		changed = true
	}

	if !changed {
		return map[string]interface{}{
			"success": true,
			"name":    name,
			"changed": false,
			"message": "No changes applied",
		}, nil
	}

	targetServer.Updated = time.Now()

	// Save config
	if s.configLoader != nil {
		if err := s.configLoader.SaveConfig(s.config); err != nil {
			s.logger.Warn("Failed to save config", zap.Error(err))
		}
	}

	// Reconnect if requested
	var reconnectResult string
	if reconnect {
		client, exists := s.manager.GetClient(name)
		if exists {
			_ = client.Disconnect()
			if err := s.manager.AddServer(name, targetServer); err != nil {
				reconnectResult = fmt.Sprintf("Reconnect failed: %v", err)
			} else {
				reconnectResult = "Reconnected successfully"
			}
		}
	}

	result := map[string]interface{}{
		"success": true,
		"name":    name,
		"changed": true,
	}
	if reconnectResult != "" {
		result["reconnect_result"] = reconnectResult
	}

	return result, nil
}

func (s *MCPToolsServer) handleConnectServer(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	name, _ := args["name"].(string)
	timeoutSec, _ := args["timeout"].(float64)
	if timeoutSec == 0 {
		timeoutSec = 30
	}

	if name == "" {
		return nil, fmt.Errorf("name is required")
	}

	client, exists := s.manager.GetClient(name)
	if !exists {
		return nil, fmt.Errorf("server '%s' not found", name)
	}

	if client.IsConnected() {
		return map[string]interface{}{
			"success":         true,
			"name":            name,
			"already_connected": true,
		}, nil
	}

	timeout := time.Duration(timeoutSec) * time.Second
	connCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	start := time.Now()
	err := client.Connect(connCtx)
	elapsed := time.Since(start)

	if err != nil {
		return map[string]interface{}{
			"success":      false,
			"name":         name,
			"error":        err.Error(),
			"elapsed_ms":   elapsed.Milliseconds(),
		}, nil
	}

	return map[string]interface{}{
		"success":    true,
		"name":       name,
		"connected":  client.IsConnected(),
		"elapsed_ms": elapsed.Milliseconds(),
	}, nil
}

func (s *MCPToolsServer) handleDisconnectServer(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	name, _ := args["name"].(string)
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}

	client, exists := s.manager.GetClient(name)
	if !exists {
		return nil, fmt.Errorf("server '%s' not found", name)
	}

	wasConnected := client.IsConnected()
	err := client.Disconnect()

	if err != nil {
		return map[string]interface{}{
			"success":       false,
			"name":          name,
			"error":         err.Error(),
			"was_connected": wasConnected,
		}, nil
	}

	return map[string]interface{}{
		"success":       true,
		"name":          name,
		"was_connected": wasConnected,
	}, nil
}

func (s *MCPToolsServer) handleReconnectServer(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	name, _ := args["name"].(string)
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}

	client, exists := s.manager.GetClient(name)
	if !exists {
		return nil, fmt.Errorf("server '%s' not found", name)
	}

	wasConnected := client.IsConnected()

	// Disconnect
	_ = client.Disconnect()

	// Wait briefly
	time.Sleep(100 * time.Millisecond)

	// Reconnect
	connCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	start := time.Now()
	err := client.Connect(connCtx)
	elapsed := time.Since(start)

	if err != nil {
		return map[string]interface{}{
			"success":       false,
			"name":          name,
			"error":         err.Error(),
			"was_connected": wasConnected,
			"elapsed_ms":    elapsed.Milliseconds(),
		}, nil
	}

	return map[string]interface{}{
		"success":       true,
		"name":          name,
		"was_connected": wasConnected,
		"connected":     client.IsConnected(),
		"elapsed_ms":    elapsed.Milliseconds(),
	}, nil
}

func (s *MCPToolsServer) handleTestConnection(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	name, _ := args["name"].(string)
	configMap, _ := args["config"].(map[string]interface{})
	timeoutSec, _ := args["timeout"].(float64)
	if timeoutSec == 0 {
		timeoutSec = 10
	}
	timeout := time.Duration(timeoutSec) * time.Second

	var serverConfig *config.ServerConfig

	if name != "" {
		// Test existing server
		servers := s.manager.ListServers()
		cfg, exists := servers[name]
		if !exists {
			return nil, fmt.Errorf("server '%s' not found", name)
		}
		serverConfig = cfg
	} else if configMap != nil {
		// Test ad-hoc config
		serverConfig = &config.ServerConfig{
			Name:        "test-server",
			StartupMode: "active",
		}
		if protocol, ok := configMap["protocol"].(string); ok {
			serverConfig.Protocol = protocol
		}
		if cmd, ok := configMap["command"].(string); ok {
			serverConfig.Command = cmd
		}
		if argsSlice, ok := configMap["args"].([]interface{}); ok {
			for _, a := range argsSlice {
				if str, ok := a.(string); ok {
					serverConfig.Args = append(serverConfig.Args, str)
				}
			}
		}
		if url, ok := configMap["url"].(string); ok {
			serverConfig.URL = url
		}
	} else {
		return nil, fmt.Errorf("either name or config is required")
	}

	result, err := s.testServerConfig(ctx, serverConfig, timeout)
	if err != nil {
		return map[string]interface{}{
			"success": false,
			"error":   err.Error(),
			"result":  result,
		}, nil
	}

	return map[string]interface{}{
		"success": true,
		"result":  result,
	}, nil
}

func (s *MCPToolsServer) handleTestProtocol(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	name, _ := args["name"].(string)
	testsSlice, _ := args["tests"].([]interface{})

	if name == "" {
		return nil, fmt.Errorf("name is required")
	}

	tests := []string{"initialize", "tools_list"}
	if len(testsSlice) > 0 {
		tests = nil
		for _, t := range testsSlice {
			if str, ok := t.(string); ok {
				tests = append(tests, str)
			}
		}
	}

	client, exists := s.manager.GetClient(name)
	if !exists {
		return nil, fmt.Errorf("server '%s' not found", name)
	}

	results := make(map[string]interface{})

	for _, test := range tests {
		switch test {
		case "initialize":
			// Check if we can get server info (requires successful initialize)
			info := client.GetServerInfo()
			if info != nil {
				results["initialize"] = map[string]interface{}{
					"success":          true,
					"protocol_version": info.ProtocolVersion,
					"server_name":      info.ServerInfo.Name,
					"server_version":   info.ServerInfo.Version,
				}
			} else if client.IsConnected() {
				results["initialize"] = map[string]interface{}{
					"success": true,
					"message": "Connected but no server info available",
				}
			} else {
				results["initialize"] = map[string]interface{}{
					"success": false,
					"error":   "Server not connected",
				}
			}

		case "tools_list":
			if !client.IsConnected() {
				results["tools_list"] = map[string]interface{}{
					"success": false,
					"error":   "Server not connected",
				}
				continue
			}
			tools, err := client.ListTools(ctx)
			if err != nil {
				results["tools_list"] = map[string]interface{}{
					"success": false,
					"error":   err.Error(),
				}
			} else {
				toolNames := make([]string, len(tools))
				for i, t := range tools {
					toolNames[i] = t.Name
				}
				results["tools_list"] = map[string]interface{}{
					"success":    true,
					"tool_count": len(tools),
					"tools":      toolNames,
				}
			}

		case "tool_call":
			results["tool_call"] = map[string]interface{}{
				"success": false,
				"error":   "tool_call test requires a specific tool name",
			}

		default:
			results[test] = map[string]interface{}{
				"success": false,
				"error":   fmt.Sprintf("Unknown test: %s", test),
			}
		}
	}

	// Determine overall success
	allPassed := true
	for _, r := range results {
		if rm, ok := r.(map[string]interface{}); ok {
			if success, ok := rm["success"].(bool); ok && !success {
				allPassed = false
				break
			}
		}
	}

	return map[string]interface{}{
		"success": allPassed,
		"name":    name,
		"tests":   results,
	}, nil
}

func (s *MCPToolsServer) handleHealthCheck(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	serversSlice, _ := args["servers"].([]interface{})
	fixIssues, _ := args["fix_issues"].(bool)

	// Check if manager is available
	if s.manager == nil {
		return map[string]interface{}{
			"total":           0,
			"healthy_count":   0,
			"unhealthy_count": 0,
			"servers":         []map[string]interface{}{},
			"message":         "No manager available",
		}, nil
	}

	var serverNames []string
	if len(serversSlice) > 0 {
		for _, s := range serversSlice {
			if str, ok := s.(string); ok {
				serverNames = append(serverNames, str)
			}
		}
	} else {
		serverNames = s.manager.GetAllServerNames()
	}

	results := make([]map[string]interface{}, 0)
	healthyCount := 0
	unhealthyCount := 0
	fixedCount := 0

	for _, name := range serverNames {
		client, exists := s.manager.GetClient(name)
		if !exists {
			results = append(results, map[string]interface{}{
				"name":    name,
				"healthy": false,
				"error":   "Server not found",
			})
			unhealthyCount++
			continue
		}

		servers := s.manager.ListServers()
		cfg := servers[name]

		if cfg.IsDisabled() {
			results = append(results, map[string]interface{}{
				"name":         name,
				"healthy":      false,
				"status":       "disabled",
				"startup_mode": cfg.StartupMode,
			})
			unhealthyCount++
			continue
		}

		if client.IsConnected() {
			results = append(results, map[string]interface{}{
				"name":    name,
				"healthy": true,
				"status":  "connected",
			})
			healthyCount++
			continue
		}

		// Unhealthy - not connected
		info := client.GetConnectionInfo()
		result := map[string]interface{}{
			"name":    name,
			"healthy": false,
			"status":  info.State.String(),
		}
		if info.LastError != nil {
			result["last_error"] = info.LastError.Error()
		}

		// Try to fix if requested
		if fixIssues {
			connCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
			err := client.Connect(connCtx)
			cancel()

			if err == nil && client.IsConnected() {
				result["fixed"] = true
				result["healthy"] = true
				result["status"] = "connected"
				healthyCount++
				fixedCount++
			} else {
				result["fix_attempted"] = true
				result["fix_error"] = err.Error()
				unhealthyCount++
			}
		} else {
			unhealthyCount++
		}

		results = append(results, result)
	}

	return map[string]interface{}{
		"servers":         results,
		"total":           len(results),
		"healthy_count":   healthyCount,
		"unhealthy_count": unhealthyCount,
		"fixed_count":     fixedCount,
	}, nil
}

func (s *MCPToolsServer) handleGetConfig(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	section, _ := args["section"].(string)
	if section == "" {
		section = "all"
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	switch section {
	case "servers":
		serverConfigs := make([]map[string]interface{}, len(s.config.Servers))
		for i, srv := range s.config.Servers {
			serverConfigs[i] = map[string]interface{}{
				"name":         srv.Name,
				"protocol":     srv.Protocol,
				"startup_mode": srv.StartupMode,
			}
			if srv.URL != "" {
				serverConfigs[i]["url"] = srv.URL
			}
			if srv.Command != "" {
				serverConfigs[i]["command"] = srv.Command
			}
		}
		return map[string]interface{}{"servers": serverConfigs}, nil

	case "logging":
		if s.config.Logging != nil {
			return map[string]interface{}{
				"level":          s.config.Logging.Level,
				"enable_file":    s.config.Logging.EnableFile,
				"enable_console": s.config.Logging.EnableConsole,
				"filename":       s.config.Logging.Filename,
			}, nil
		}
		return map[string]interface{}{"logging": nil}, nil

	case "security":
		return map[string]interface{}{
			"read_only_mode":      s.config.ReadOnlyMode,
			"disable_management":  s.config.DisableManagement,
			"allow_server_add":    s.config.AllowServerAdd,
			"allow_server_remove": s.config.AllowServerRemove,
		}, nil

	case "docker":
		if s.config.DockerIsolation != nil {
			return map[string]interface{}{
				"enabled":       s.config.DockerIsolation.Enabled,
				"network_mode":  s.config.DockerIsolation.NetworkMode,
				"memory_limit":  s.config.DockerIsolation.MemoryLimit,
				"cpu_limit":     s.config.DockerIsolation.CPULimit,
			}, nil
		}
		return map[string]interface{}{"docker": nil}, nil

	case "llm":
		if s.config.LLM != nil {
			return map[string]interface{}{
				"provider":    s.config.LLM.Provider,
				"model":       s.config.LLM.Model,
				"temperature": s.config.LLM.Temperature,
				"max_tokens":  s.config.LLM.MaxTokens,
			}, nil
		}
		return map[string]interface{}{"llm": nil}, nil

	default: // "all"
		// Return sanitized config
		return map[string]interface{}{
			"listen":                     s.config.Listen,
			"data_dir":                   s.config.DataDir,
			"server_count":               len(s.config.Servers),
			"top_k":                      s.config.TopK,
			"tools_limit":                s.config.ToolsLimit,
			"enable_lazy_loading":        s.config.EnableLazyLoading,
			"max_concurrent_connections": s.config.MaxConcurrentConnections,
		}, nil
	}
}

func (s *MCPToolsServer) handleValidateConfig(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	strict, _ := args["strict"].(bool)

	s.mu.RLock()
	cfg := s.config
	s.mu.RUnlock()

	issues := make([]string, 0)
	warnings := make([]string, 0)

	// Validate basic config
	if err := cfg.Validate(); err != nil {
		issues = append(issues, fmt.Sprintf("Validation error: %v", err))
	}

	// Check servers
	for _, srv := range cfg.Servers {
		if srv.Name == "" {
			issues = append(issues, "Server with empty name found")
		}
		if srv.Protocol == "" {
			issues = append(issues, fmt.Sprintf("Server '%s' has no protocol", srv.Name))
		}
		if srv.Protocol == "stdio" && srv.Command == "" {
			issues = append(issues, fmt.Sprintf("Server '%s' (stdio) has no command", srv.Name))
		}
		if (srv.Protocol == "http" || srv.Protocol == "sse" || srv.Protocol == "streamable-http") && srv.URL == "" {
			issues = append(issues, fmt.Sprintf("Server '%s' (%s) has no URL", srv.Name, srv.Protocol))
		}

		// Warnings (strict mode)
		if strict {
			if srv.Description == "" {
				warnings = append(warnings, fmt.Sprintf("Server '%s' has no description", srv.Name))
			}
			if srv.StartupMode == "" {
				warnings = append(warnings, fmt.Sprintf("Server '%s' has no startup_mode (defaults to active)", srv.Name))
			}
		}
	}

	valid := len(issues) == 0

	result := map[string]interface{}{
		"valid":  valid,
		"issues": issues,
	}
	if strict {
		result["warnings"] = warnings
	}

	return result, nil
}

func (s *MCPToolsServer) handleDiagnose(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	checksSlice, _ := args["checks"].([]interface{})
	serverName, _ := args["server"].(string)

	checks := []string{"config", "servers"}
	if len(checksSlice) > 0 {
		checks = nil
		for _, c := range checksSlice {
			if str, ok := c.(string); ok {
				checks = append(checks, str)
			}
		}
	}

	results := make(map[string]interface{})

	for _, check := range checks {
		switch check {
		case "config":
			// Validate config
			s.mu.RLock()
			err := s.config.Validate()
			s.mu.RUnlock()

			if err != nil {
				results["config"] = map[string]interface{}{
					"status": "error",
					"error":  err.Error(),
				}
			} else {
				results["config"] = map[string]interface{}{
					"status":       "ok",
					"server_count": len(s.config.Servers),
				}
			}

		case "servers":
			stats := s.manager.GetStats()
			serverResults := make([]map[string]interface{}, 0)

			if serverName != "" {
				// Diagnose specific server
				client, exists := s.manager.GetClient(serverName)
				if !exists {
					serverResults = append(serverResults, map[string]interface{}{
						"name":   serverName,
						"status": "not_found",
					})
				} else {
					info := client.GetConnectionInfo()
					serverResults = append(serverResults, map[string]interface{}{
						"name":        serverName,
						"state":       info.State.String(),
						"connected":   client.IsConnected(),
						"retry_count": info.RetryCount,
					})
				}
			} else {
				// Diagnose all servers
				for name := range s.manager.ListServers() {
					client, _ := s.manager.GetClient(name)
					if client != nil {
						info := client.GetConnectionInfo()
						serverResults = append(serverResults, map[string]interface{}{
							"name":      name,
							"state":     info.State.String(),
							"connected": client.IsConnected(),
						})
					}
				}
			}

			results["servers"] = map[string]interface{}{
				"status":  "ok",
				"stats":   stats,
				"details": serverResults,
			}

		case "network":
			results["network"] = map[string]interface{}{
				"status":  "ok",
				"message": "Network diagnostics not yet implemented",
			}

		case "permissions":
			results["permissions"] = map[string]interface{}{
				"status":  "ok",
				"message": "Permission diagnostics not yet implemented",
			}

		case "logs":
			results["logs"] = map[string]interface{}{
				"status":  "ok",
				"message": "Log diagnostics not yet implemented",
			}

		default:
			results[check] = map[string]interface{}{
				"status": "unknown",
				"error":  fmt.Sprintf("Unknown check: %s", check),
			}
		}
	}

	return map[string]interface{}{
		"checks":  results,
		"summary": fmt.Sprintf("Ran %d diagnostic checks", len(checks)),
	}, nil
}

func (s *MCPToolsServer) handleGetLogs(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	// This is a placeholder - actual log retrieval would need access to log files
	return map[string]interface{}{
		"success": false,
		"message": "Log retrieval not yet implemented. Check ~/.mcpproxy/logs/ directory",
	}, nil
}

// Tool Testing Handlers

func (s *MCPToolsServer) handleListServerTools(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	serverName, _ := args["server"].(string)
	filter, _ := args["filter"].(string)
	includeSchema := true
	if val, ok := args["include_schema"].(bool); ok {
		includeSchema = val
	}

	if serverName == "" {
		return nil, fmt.Errorf("server name is required")
	}

	if s.manager == nil {
		return nil, fmt.Errorf("manager not available")
	}

	// Get client for the server
	client, exists := s.manager.GetClient(serverName)
	if !exists {
		return nil, fmt.Errorf("server '%s' not found", serverName)
	}

	if !client.IsConnected() {
		return nil, fmt.Errorf("server '%s' is not connected", serverName)
	}

	// List tools from the server
	tools, err := client.ListTools(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list tools: %w", err)
	}

	// Build response
	toolList := make([]map[string]interface{}, 0)
	for _, tool := range tools {
		// Apply filter if specified
		if filter != "" && !matchWildcard(tool.Name, filter) {
			continue
		}

		toolInfo := map[string]interface{}{
			"name":        tool.Name,
			"description": tool.Description,
			"server":      tool.ServerName,
		}

		// ParamsJSON contains the schema as JSON string
		if includeSchema && tool.ParamsJSON != "" {
			var schema map[string]interface{}
			if err := json.Unmarshal([]byte(tool.ParamsJSON), &schema); err == nil {
				toolInfo["input_schema"] = schema
			}
		}

		toolList = append(toolList, toolInfo)
	}

	return map[string]interface{}{
		"server":     serverName,
		"tool_count": len(toolList),
		"tools":      toolList,
	}, nil
}

func (s *MCPToolsServer) handleCallServerTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	serverName, _ := args["server"].(string)
	toolName, _ := args["tool"].(string)
	toolArgs, _ := args["arguments"].(map[string]interface{})
	timeoutSec := 30
	if val, ok := args["timeout"].(float64); ok {
		timeoutSec = int(val)
	}

	if serverName == "" {
		return nil, fmt.Errorf("server name is required")
	}
	if toolName == "" {
		return nil, fmt.Errorf("tool name is required")
	}

	if s.manager == nil {
		return nil, fmt.Errorf("manager not available")
	}

	// Get client for the server
	client, exists := s.manager.GetClient(serverName)
	if !exists {
		return nil, fmt.Errorf("server '%s' not found", serverName)
	}

	if !client.IsConnected() {
		return nil, fmt.Errorf("server '%s' is not connected", serverName)
	}

	// Create timeout context
	timeout := time.Duration(timeoutSec) * time.Second
	callCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Call the tool
	startTime := time.Now()
	result, err := client.CallTool(callCtx, toolName, toolArgs)
	duration := time.Since(startTime)

	response := map[string]interface{}{
		"server":      serverName,
		"tool":        toolName,
		"duration_ms": duration.Milliseconds(),
	}

	if err != nil {
		response["success"] = false
		response["error"] = err.Error()
		return response, nil
	}

	response["success"] = true

	// Extract result content
	if result != nil {
		response["is_error"] = result.IsError

		if len(result.Content) > 0 {
			contents := make([]map[string]interface{}, 0)
			for _, content := range result.Content {
				// Marshal content to JSON to extract fields
				contentBytes, err := json.Marshal(content)
				if err != nil {
					continue
				}
				var contentMap map[string]interface{}
				if err := json.Unmarshal(contentBytes, &contentMap); err != nil {
					continue
				}
				contents = append(contents, contentMap)
			}
			response["content"] = contents
		}
	}

	return response, nil
}

func (s *MCPToolsServer) handleTestServerTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	serverName, _ := args["server"].(string)
	toolName, _ := args["tool"].(string)
	testCasesRaw, _ := args["test_cases"].([]interface{})
	timeoutSec := 30
	if val, ok := args["timeout"].(float64); ok {
		timeoutSec = int(val)
	}
	stopOnFailure, _ := args["stop_on_failure"].(bool)

	if serverName == "" {
		return nil, fmt.Errorf("server name is required")
	}
	if toolName == "" {
		return nil, fmt.Errorf("tool name is required")
	}
	if len(testCasesRaw) == 0 {
		return nil, fmt.Errorf("at least one test case is required")
	}

	if s.manager == nil {
		return nil, fmt.Errorf("manager not available")
	}

	// Get client for the server
	client, exists := s.manager.GetClient(serverName)
	if !exists {
		return nil, fmt.Errorf("server '%s' not found", serverName)
	}

	if !client.IsConnected() {
		return nil, fmt.Errorf("server '%s' is not connected", serverName)
	}

	timeout := time.Duration(timeoutSec) * time.Second

	// Run test cases
	testResults := make([]map[string]interface{}, 0)
	passedCount := 0
	failedCount := 0

	for i, tcRaw := range testCasesRaw {
		tc, ok := tcRaw.(map[string]interface{})
		if !ok {
			continue
		}

		testName, _ := tc["name"].(string)
		if testName == "" {
			testName = fmt.Sprintf("test_%d", i+1)
		}
		testArgs, _ := tc["arguments"].(map[string]interface{})
		expectError, _ := tc["expect_error"].(bool)
		expectContains, _ := tc["expect_contains"].(string)
		expectNotContains, _ := tc["expect_not_contains"].(string)

		// Run the test
		callCtx, cancel := context.WithTimeout(ctx, timeout)
		startTime := time.Now()
		result, err := client.CallTool(callCtx, toolName, testArgs)
		duration := time.Since(startTime)
		cancel()

		testResult := map[string]interface{}{
			"name":        testName,
			"duration_ms": duration.Milliseconds(),
		}

		// Evaluate test result
		passed := true
		var failureReason string

		if err != nil {
			if expectError {
				testResult["got_expected_error"] = true
			} else {
				passed = false
				failureReason = fmt.Sprintf("unexpected error: %s", err.Error())
			}
			testResult["error"] = err.Error()
		} else {
			if expectError {
				passed = false
				failureReason = "expected error but got success"
			}

			// Check result content
			if result != nil && len(result.Content) > 0 {
				var resultText string
				for _, content := range result.Content {
					// Marshal content to JSON to extract fields
					contentBytes, err := json.Marshal(content)
					if err != nil {
						continue
					}
					var contentMap map[string]interface{}
					if err := json.Unmarshal(contentBytes, &contentMap); err != nil {
						continue
					}
					// Extract text if type is "text"
					if contentType, ok := contentMap["type"].(string); ok && contentType == "text" {
						if text, ok := contentMap["text"].(string); ok {
							resultText += text
						}
					}
				}

				testResult["result_preview"] = truncateString(resultText, 200)

				if expectContains != "" && !strings.Contains(resultText, expectContains) {
					passed = false
					failureReason = fmt.Sprintf("expected result to contain '%s'", expectContains)
				}

				if expectNotContains != "" && strings.Contains(resultText, expectNotContains) {
					passed = false
					failureReason = fmt.Sprintf("expected result NOT to contain '%s'", expectNotContains)
				}
			}
		}

		testResult["passed"] = passed
		if !passed {
			testResult["failure_reason"] = failureReason
			failedCount++
		} else {
			passedCount++
		}

		testResults = append(testResults, testResult)

		// Stop if requested and test failed
		if stopOnFailure && !passed {
			break
		}
	}

	return map[string]interface{}{
		"server":       serverName,
		"tool":         toolName,
		"total_tests":  len(testResults),
		"passed":       passedCount,
		"failed":       failedCount,
		"success_rate": fmt.Sprintf("%.1f%%", float64(passedCount)/float64(len(testResults))*100),
		"test_results": testResults,
	}, nil
}

// matchWildcard performs simple wildcard matching (* for any characters)
func matchWildcard(s, pattern string) bool {
	if pattern == "" || pattern == "*" {
		return true
	}
	// Simple prefix/suffix matching
	if strings.HasPrefix(pattern, "*") && strings.HasSuffix(pattern, "*") {
		return strings.Contains(s, pattern[1:len(pattern)-1])
	}
	if strings.HasPrefix(pattern, "*") {
		return strings.HasSuffix(s, pattern[1:])
	}
	if strings.HasSuffix(pattern, "*") {
		return strings.HasPrefix(s, pattern[:len(pattern)-1])
	}
	return s == pattern
}

// truncateString truncates a string to maxLen and adds "..." if truncated
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// Helper methods

func (s *MCPToolsServer) testServerConfig(ctx context.Context, cfg *config.ServerConfig, timeout time.Duration) (map[string]interface{}, error) {
	result := map[string]interface{}{
		"protocol":   cfg.Protocol,
		"timeout_ms": timeout.Milliseconds(),
	}

	// For now, return a basic test result
	// Full implementation would create a temporary client and test connection
	result["test_type"] = "basic"
	result["message"] = "Connection testing implemented via manager"

	return result, nil
}

// MarshalJSON for tool definitions
func (t ToolDefinition) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"name":        t.Name,
		"description": t.Description,
		"inputSchema": t.InputSchema,
	})
}

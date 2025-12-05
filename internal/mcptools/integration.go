// Package mcptools provides MCP tools for managing mcpproxy servers
package mcptools

import (
	"context"
	"encoding/json"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
	"go.uber.org/zap"

	"mcpproxy-go/internal/config"
	"mcpproxy-go/internal/upstream"
)

// RegisterTools registers all mcptools with an MCP server
func RegisterTools(
	mcpServer *mcpserver.MCPServer,
	manager *upstream.Manager,
	cfg *config.Config,
	loader ConfigLoader,
	logger *zap.Logger,
) *MCPToolsServer {
	toolsServer := NewMCPToolsServer(manager, cfg, loader, logger)

	// Register each tool with the MCP server
	for _, toolDef := range toolsServer.GetTools() {
		tool := createMCPTool(toolDef)
		mcpServer.AddTool(tool, createToolHandler(toolsServer, toolDef.Name))
	}

	logger.Info("Registered mcptools with MCP server",
		zap.Int("tool_count", len(toolsServer.GetTools())))

	return toolsServer
}

// createMCPTool creates an MCP tool from a ToolDefinition
func createMCPTool(def ToolDefinition) mcp.Tool {
	opts := []mcp.ToolOption{
		mcp.WithDescription(def.Description),
	}

	// Extract properties from input schema
	if props, ok := def.InputSchema["properties"].(map[string]interface{}); ok {
		required := make(map[string]bool)
		if req, ok := def.InputSchema["required"].([]string); ok {
			for _, r := range req {
				required[r] = true
			}
		}

		for name, propDef := range props {
			propMap, ok := propDef.(map[string]interface{})
			if !ok {
				continue
			}

			propType, _ := propMap["type"].(string)
			propDesc, _ := propMap["description"].(string)
			isRequired := required[name]

			switch propType {
			case "string":
				stringOpts := []mcp.PropertyOption{mcp.Description(propDesc)}
				if isRequired {
					stringOpts = append(stringOpts, mcp.Required())
				}
				if enum, ok := propMap["enum"].([]string); ok {
					stringOpts = append(stringOpts, mcp.Enum(enum...))
				}
				opts = append(opts, mcp.WithString(name, stringOpts...))

			case "integer", "number":
				numOpts := []mcp.PropertyOption{mcp.Description(propDesc)}
				if isRequired {
					numOpts = append(numOpts, mcp.Required())
				}
				opts = append(opts, mcp.WithNumber(name, numOpts...))

			case "boolean":
				boolOpts := []mcp.PropertyOption{mcp.Description(propDesc)}
				if isRequired {
					boolOpts = append(boolOpts, mcp.Required())
				}
				opts = append(opts, mcp.WithBoolean(name, boolOpts...))

			case "array":
				// For arrays, we'll accept them as JSON strings
				arrayOpts := []mcp.PropertyOption{mcp.Description(propDesc + " (as JSON array)")}
				if isRequired {
					arrayOpts = append(arrayOpts, mcp.Required())
				}
				opts = append(opts, mcp.WithString(name, arrayOpts...))

			case "object":
				// For objects, we'll accept them as JSON strings
				objOpts := []mcp.PropertyOption{mcp.Description(propDesc + " (as JSON object)")}
				if isRequired {
					objOpts = append(objOpts, mcp.Required())
				}
				opts = append(opts, mcp.WithString(name, objOpts...))
			}
		}
	}

	return mcp.NewTool(def.Name, opts...)
}

// createToolHandler creates an MCP tool handler for a specific tool
func createToolHandler(server *MCPToolsServer, toolName string) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Convert request arguments to map
		args := make(map[string]interface{})
		if request.Params.Arguments != nil {
			if argsMap, ok := request.Params.Arguments.(map[string]interface{}); ok {
				args = argsMap
			}
		}

		result, err := server.CallTool(ctx, toolName, args)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		// Format result as JSON
		jsonResult, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return mcp.NewToolResultText(formatResult(result)), nil
		}
		return mcp.NewToolResultText(string(jsonResult)), nil
	}
}

// formatResult formats the tool result as a string
func formatResult(result interface{}) string {
	if result == nil {
		return "{}"
	}

	switch v := result.(type) {
	case string:
		return v
	case map[string]interface{}:
		return formatMap(v, 0)
	case []interface{}:
		return formatSlice(v)
	default:
		return "{\"result\": \"unknown type\"}"
	}
}

func formatMap(m map[string]interface{}, indent int) string {
	if len(m) == 0 {
		return "{}"
	}

	result := "{\n"
	i := 0
	for k, v := range m {
		result += spaces(indent+2) + "\"" + k + "\": "
		result += formatValue(v, indent+2)
		if i < len(m)-1 {
			result += ","
		}
		result += "\n"
		i++
	}
	result += spaces(indent) + "}"
	return result
}

func formatSlice(s []interface{}) string {
	if len(s) == 0 {
		return "[]"
	}

	result := "[\n"
	for i, v := range s {
		result += "  " + formatValue(v, 2)
		if i < len(s)-1 {
			result += ","
		}
		result += "\n"
	}
	result += "]"
	return result
}

func formatValue(v interface{}, indent int) string {
	switch val := v.(type) {
	case nil:
		return "null"
	case bool:
		if val {
			return "true"
		}
		return "false"
	case string:
		return "\"" + escapeString(val) + "\""
	case int, int64, float64:
		return formatNumber(val)
	case []interface{}:
		return formatSlice(val)
	case map[string]interface{}:
		return formatMap(val, indent)
	case []string:
		items := make([]interface{}, len(val))
		for i, s := range val {
			items[i] = s
		}
		return formatSlice(items)
	case []map[string]interface{}:
		items := make([]interface{}, len(val))
		for i, m := range val {
			items[i] = m
		}
		return formatSlice(items)
	default:
		return "\"<unknown>\""
	}
}

func formatNumber(v interface{}) string {
	switch n := v.(type) {
	case int:
		return itoa(n)
	case int64:
		return itoa64(n)
	case float64:
		if n == float64(int64(n)) {
			return itoa64(int64(n))
		}
		return ftoa(n)
	default:
		return "0"
	}
}

func escapeString(s string) string {
	result := ""
	for _, c := range s {
		switch c {
		case '"':
			result += "\\\""
		case '\\':
			result += "\\\\"
		case '\n':
			result += "\\n"
		case '\r':
			result += "\\r"
		case '\t':
			result += "\\t"
		default:
			result += string(c)
		}
	}
	return result
}

func spaces(n int) string {
	result := ""
	for i := 0; i < n; i++ {
		result += " "
	}
	return result
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}
	digits := ""
	for n > 0 {
		digits = string(rune('0'+n%10)) + digits
		n /= 10
	}
	if neg {
		digits = "-" + digits
	}
	return digits
}

func itoa64(n int64) string {
	if n == 0 {
		return "0"
	}
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}
	digits := ""
	for n > 0 {
		digits = string(rune('0'+n%10)) + digits
		n /= 10
	}
	if neg {
		digits = "-" + digits
	}
	return digits
}

func ftoa(f float64) string {
	// Simple float formatting - for production use strconv.FormatFloat
	intPart := int64(f)
	fracPart := f - float64(intPart)
	if fracPart < 0 {
		fracPart = -fracPart
	}

	result := itoa64(intPart)
	if fracPart > 0.0001 {
		result += "."
		// Add up to 4 decimal places
		for i := 0; i < 4 && fracPart > 0.0001; i++ {
			fracPart *= 10
			digit := int(fracPart)
			result += string(rune('0' + digit))
			fracPart -= float64(digit)
		}
	}
	return result
}

// ToolInfo provides information about available tools
type ToolInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Category    string `json:"category"`
}

// GetToolInfo returns categorized tool information
func GetToolInfo() []ToolInfo {
	return []ToolInfo{
		// Server Management
		{Name: "mcpproxy_add_server", Description: "Add a new MCP server", Category: "server-management"},
		{Name: "mcpproxy_remove_server", Description: "Remove an MCP server", Category: "server-management"},
		{Name: "mcpproxy_list_servers", Description: "List all servers", Category: "server-management"},
		{Name: "mcpproxy_get_server", Description: "Get server details", Category: "server-management"},
		{Name: "mcpproxy_update_server", Description: "Update server config", Category: "server-management"},

		// Connection Management
		{Name: "mcpproxy_connect_server", Description: "Connect to server", Category: "connection"},
		{Name: "mcpproxy_disconnect_server", Description: "Disconnect from server", Category: "connection"},
		{Name: "mcpproxy_reconnect_server", Description: "Reconnect to server", Category: "connection"},

		// Testing
		{Name: "mcpproxy_test_connection", Description: "Test server connectivity", Category: "testing"},
		{Name: "mcpproxy_test_protocol", Description: "Test protocol compliance", Category: "testing"},
		{Name: "mcpproxy_health_check", Description: "Run health checks", Category: "testing"},

		// Configuration
		{Name: "mcpproxy_get_config", Description: "Get configuration", Category: "configuration"},
		{Name: "mcpproxy_validate_config", Description: "Validate configuration", Category: "configuration"},

		// Diagnostics
		{Name: "mcpproxy_diagnose", Description: "Run diagnostics", Category: "diagnostics"},
		{Name: "mcpproxy_get_logs", Description: "Retrieve logs", Category: "diagnostics"},

		// Tool Testing
		{Name: "mcpproxy_list_server_tools", Description: "List server tools with schemas", Category: "tool-testing"},
		{Name: "mcpproxy_call_server_tool", Description: "Call a tool on a server", Category: "tool-testing"},
		{Name: "mcpproxy_test_server_tool", Description: "Run test cases against a tool", Category: "tool-testing"},
	}
}

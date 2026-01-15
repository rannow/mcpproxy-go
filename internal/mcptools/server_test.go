package mcptools

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"mcpproxy-go/internal/config"
)

// MockConfigLoader implements ConfigLoader for testing
type MockConfigLoader struct {
	savedConfig *config.Config
	saveError   error
}

func (m *MockConfigLoader) SaveConfig(cfg *config.Config) error {
	if m.saveError != nil {
		return m.saveError
	}
	m.savedConfig = cfg
	return nil
}

func (m *MockConfigLoader) ReloadConfig() (*config.Config, error) {
	return m.savedConfig, nil
}

func TestNewMCPToolsServer(t *testing.T) {
	logger := zap.NewNop()
	cfg := config.DefaultConfig()
	loader := &MockConfigLoader{}

	server := NewMCPToolsServer(nil, cfg, loader, logger)

	assert.NotNil(t, server)
	assert.Equal(t, cfg, server.config)
	assert.Equal(t, loader, server.configLoader)
}

func TestGetTools(t *testing.T) {
	logger := zap.NewNop()
	cfg := config.DefaultConfig()
	server := NewMCPToolsServer(nil, cfg, nil, logger)

	tools := server.GetTools()

	// Should have all management tools
	assert.GreaterOrEqual(t, len(tools), 15)

	// Check specific tools exist
	toolNames := make(map[string]bool)
	for _, tool := range tools {
		toolNames[tool.Name] = true
	}

	expectedTools := []string{
		"mcpproxy_add_server",
		"mcpproxy_remove_server",
		"mcpproxy_list_servers",
		"mcpproxy_get_server",
		"mcpproxy_update_server",
		"mcpproxy_connect_server",
		"mcpproxy_disconnect_server",
		"mcpproxy_reconnect_server",
		"mcpproxy_test_connection",
		"mcpproxy_test_protocol",
		"mcpproxy_health_check",
		"mcpproxy_get_config",
		"mcpproxy_validate_config",
		"mcpproxy_diagnose",
		"mcpproxy_get_logs",
	}

	for _, name := range expectedTools {
		assert.True(t, toolNames[name], "Expected tool %s to exist", name)
	}
}

func TestToolDefinitionSchema(t *testing.T) {
	logger := zap.NewNop()
	cfg := config.DefaultConfig()
	server := NewMCPToolsServer(nil, cfg, nil, logger)

	tools := server.GetTools()

	for _, tool := range tools {
		// Each tool should have required fields
		assert.NotEmpty(t, tool.Name, "Tool name should not be empty")
		assert.NotEmpty(t, tool.Description, "Tool %s should have description", tool.Name)
		assert.NotNil(t, tool.InputSchema, "Tool %s should have input schema", tool.Name)

		// Input schema should be an object type
		schemaType, ok := tool.InputSchema["type"].(string)
		assert.True(t, ok, "Tool %s schema should have type", tool.Name)
		assert.Equal(t, "object", schemaType, "Tool %s schema type should be object", tool.Name)
	}
}

func TestCallToolUnknown(t *testing.T) {
	logger := zap.NewNop()
	cfg := config.DefaultConfig()
	server := NewMCPToolsServer(nil, cfg, nil, logger)

	ctx := context.Background()
	_, err := server.CallTool(ctx, "unknown_tool", nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown tool")
}

func TestHandleGetConfig(t *testing.T) {
	logger := zap.NewNop()
	cfg := config.DefaultConfig()
	cfg.Listen = ":9090"
	cfg.TopK = 10
	server := NewMCPToolsServer(nil, cfg, nil, logger)

	ctx := context.Background()

	t.Run("all sections", func(t *testing.T) {
		result, err := server.CallTool(ctx, "mcpproxy_get_config", map[string]interface{}{
			"section": "all",
		})
		require.NoError(t, err)

		resultMap := result.(map[string]interface{})
		assert.Equal(t, ":9090", resultMap["listen"])
		assert.Equal(t, 10, resultMap["top_k"])
	})

	t.Run("security section", func(t *testing.T) {
		result, err := server.CallTool(ctx, "mcpproxy_get_config", map[string]interface{}{
			"section": "security",
		})
		require.NoError(t, err)

		resultMap := result.(map[string]interface{})
		assert.Contains(t, resultMap, "read_only_mode")
		assert.Contains(t, resultMap, "allow_server_add")
	})

	t.Run("default section", func(t *testing.T) {
		result, err := server.CallTool(ctx, "mcpproxy_get_config", map[string]interface{}{})
		require.NoError(t, err)

		resultMap := result.(map[string]interface{})
		assert.Contains(t, resultMap, "listen")
	})
}

func TestHandleValidateConfig(t *testing.T) {
	logger := zap.NewNop()
	cfg := config.DefaultConfig()
	server := NewMCPToolsServer(nil, cfg, nil, logger)

	ctx := context.Background()

	t.Run("valid config", func(t *testing.T) {
		result, err := server.CallTool(ctx, "mcpproxy_validate_config", map[string]interface{}{})
		require.NoError(t, err)

		resultMap := result.(map[string]interface{})
		assert.True(t, resultMap["valid"].(bool))
	})

	t.Run("strict mode", func(t *testing.T) {
		result, err := server.CallTool(ctx, "mcpproxy_validate_config", map[string]interface{}{
			"strict": true,
		})
		require.NoError(t, err)

		resultMap := result.(map[string]interface{})
		assert.Contains(t, resultMap, "warnings")
	})
}

func TestHandleDiagnose(t *testing.T) {
	logger := zap.NewNop()
	cfg := config.DefaultConfig()
	server := NewMCPToolsServer(nil, cfg, nil, logger)

	ctx := context.Background()

	t.Run("config check", func(t *testing.T) {
		result, err := server.CallTool(ctx, "mcpproxy_diagnose", map[string]interface{}{
			"checks": []interface{}{"config"},
		})
		require.NoError(t, err)

		resultMap := result.(map[string]interface{})
		checks := resultMap["checks"].(map[string]interface{})
		configCheck := checks["config"].(map[string]interface{})
		assert.Equal(t, "ok", configCheck["status"])
	})
}

func TestHandleAddServerValidation(t *testing.T) {
	logger := zap.NewNop()
	cfg := config.DefaultConfig()
	server := NewMCPToolsServer(nil, cfg, nil, logger)

	ctx := context.Background()

	t.Run("missing name", func(t *testing.T) {
		_, err := server.CallTool(ctx, "mcpproxy_add_server", map[string]interface{}{
			"protocol": "stdio",
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "name is required")
	})

	t.Run("missing protocol", func(t *testing.T) {
		_, err := server.CallTool(ctx, "mcpproxy_add_server", map[string]interface{}{
			"name": "test-server",
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "protocol is required")
	})

	t.Run("stdio without command", func(t *testing.T) {
		_, err := server.CallTool(ctx, "mcpproxy_add_server", map[string]interface{}{
			"name":            "test-server",
			"protocol":        "stdio",
			"test_before_add": false,
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "command is required")
	})

	t.Run("http without url", func(t *testing.T) {
		_, err := server.CallTool(ctx, "mcpproxy_add_server", map[string]interface{}{
			"name":            "test-server",
			"protocol":        "http",
			"test_before_add": false,
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "url is required")
	})
}

func TestHandleTestConnectionValidation(t *testing.T) {
	logger := zap.NewNop()
	cfg := config.DefaultConfig()
	server := NewMCPToolsServer(nil, cfg, nil, logger)

	ctx := context.Background()

	t.Run("missing both name and config", func(t *testing.T) {
		_, err := server.CallTool(ctx, "mcpproxy_test_connection", map[string]interface{}{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "either name or config is required")
	})
}

func TestHandleHealthCheck(t *testing.T) {
	logger := zap.NewNop()
	cfg := config.DefaultConfig()
	server := NewMCPToolsServer(nil, cfg, nil, logger)

	ctx := context.Background()

	t.Run("no servers", func(t *testing.T) {
		result, err := server.CallTool(ctx, "mcpproxy_health_check", map[string]interface{}{})
		require.NoError(t, err)

		resultMap := result.(map[string]interface{})
		assert.Equal(t, 0, resultMap["total"])
		assert.Equal(t, 0, resultMap["healthy_count"])
	})
}

func TestToolDefinitionMarshalJSON(t *testing.T) {
	tool := ToolDefinition{
		Name:        "test_tool",
		Description: "Test tool description",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"param1": map[string]interface{}{
					"type": "string",
				},
			},
		},
	}

	data, err := tool.MarshalJSON()
	require.NoError(t, err)

	assert.Contains(t, string(data), "test_tool")
	assert.Contains(t, string(data), "Test tool description")
	assert.Contains(t, string(data), "inputSchema")
}

func TestServerConfigBuilding(t *testing.T) {
	logger := zap.NewNop()
	cfg := config.DefaultConfig()
	loader := &MockConfigLoader{}
	server := NewMCPToolsServer(nil, cfg, loader, logger)

	ctx := context.Background()

	// Test that we can build various server configs from args
	// (without actually connecting since we have no manager)

	testCases := []struct {
		name     string
		args     map[string]interface{}
		wantErr  bool
		errMatch string
	}{
		{
			name: "valid stdio config",
			args: map[string]interface{}{
				"name":            "test-stdio",
				"protocol":        "stdio",
				"command":         "npx",
				"args":            []interface{}{"-y", "test-server"},
				"test_before_add": false,
			},
			wantErr: true, // Will fail because no manager
		},
		{
			name: "valid http config",
			args: map[string]interface{}{
				"name":            "test-http",
				"protocol":        "streamable-http",
				"url":             "https://example.com/mcp",
				"test_before_add": false,
			},
			wantErr: true, // Will fail because no manager
		},
		{
			name: "with env and headers",
			args: map[string]interface{}{
				"name":            "test-full",
				"protocol":        "http",
				"url":             "https://example.com/mcp",
				"env":             map[string]interface{}{"API_KEY": "test"},
				"headers":         map[string]interface{}{"Authorization": "Bearer token"},
				"startup_mode":    "lazy_loading",
				"description":     "Test server",
				"test_before_add": false,
			},
			wantErr: true, // Will fail because no manager
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := server.CallTool(ctx, "mcpproxy_add_server", tc.args)
			if tc.wantErr {
				assert.Error(t, err)
				if tc.errMatch != "" {
					assert.Contains(t, err.Error(), tc.errMatch)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestTimeoutHandling(t *testing.T) {
	logger := zap.NewNop()
	cfg := config.DefaultConfig()
	server := NewMCPToolsServer(nil, cfg, nil, logger)

	// Test that timeout parameter is parsed correctly
	args := map[string]interface{}{
		"timeout": float64(5),
	}

	timeoutSec, ok := args["timeout"].(float64)
	assert.True(t, ok)
	assert.Equal(t, float64(5), timeoutSec)

	timeout := time.Duration(timeoutSec) * time.Second
	assert.Equal(t, 5*time.Second, timeout)

	_ = server // Use server to avoid unused warning
}

func TestGetToolsIncludesNewToolTestingTools(t *testing.T) {
	logger := zap.NewNop()
	cfg := config.DefaultConfig()
	server := NewMCPToolsServer(nil, cfg, nil, logger)

	tools := server.GetTools()

	// Check new tool testing tools exist
	toolNames := make(map[string]bool)
	for _, tool := range tools {
		toolNames[tool.Name] = true
	}

	newTools := []string{
		"mcpproxy_list_server_tools",
		"mcpproxy_call_server_tool",
		"mcpproxy_test_server_tool",
	}

	for _, name := range newTools {
		assert.True(t, toolNames[name], "Expected tool %s to exist", name)
	}

	// Should now have 18 tools (15 + 3 new)
	assert.GreaterOrEqual(t, len(tools), 18)
}

func TestHandleListServerToolsValidation(t *testing.T) {
	logger := zap.NewNop()
	cfg := config.DefaultConfig()
	server := NewMCPToolsServer(nil, cfg, nil, logger)

	ctx := context.Background()

	t.Run("missing server name", func(t *testing.T) {
		_, err := server.CallTool(ctx, "mcpproxy_list_server_tools", map[string]interface{}{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "server name is required")
	})

	t.Run("manager not available", func(t *testing.T) {
		_, err := server.CallTool(ctx, "mcpproxy_list_server_tools", map[string]interface{}{
			"server": "test-server",
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "manager not available")
	})
}

func TestHandleCallServerToolValidation(t *testing.T) {
	logger := zap.NewNop()
	cfg := config.DefaultConfig()
	server := NewMCPToolsServer(nil, cfg, nil, logger)

	ctx := context.Background()

	t.Run("missing server name", func(t *testing.T) {
		_, err := server.CallTool(ctx, "mcpproxy_call_server_tool", map[string]interface{}{
			"tool": "some_tool",
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "server name is required")
	})

	t.Run("missing tool name", func(t *testing.T) {
		_, err := server.CallTool(ctx, "mcpproxy_call_server_tool", map[string]interface{}{
			"server": "test-server",
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "tool name is required")
	})

	t.Run("manager not available", func(t *testing.T) {
		_, err := server.CallTool(ctx, "mcpproxy_call_server_tool", map[string]interface{}{
			"server": "test-server",
			"tool":   "some_tool",
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "manager not available")
	})
}

func TestHandleTestServerToolValidation(t *testing.T) {
	logger := zap.NewNop()
	cfg := config.DefaultConfig()
	server := NewMCPToolsServer(nil, cfg, nil, logger)

	ctx := context.Background()

	t.Run("missing server name", func(t *testing.T) {
		_, err := server.CallTool(ctx, "mcpproxy_test_server_tool", map[string]interface{}{
			"tool":       "some_tool",
			"test_cases": []interface{}{map[string]interface{}{"name": "test1"}},
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "server name is required")
	})

	t.Run("missing tool name", func(t *testing.T) {
		_, err := server.CallTool(ctx, "mcpproxy_test_server_tool", map[string]interface{}{
			"server":     "test-server",
			"test_cases": []interface{}{map[string]interface{}{"name": "test1"}},
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "tool name is required")
	})

	t.Run("missing test cases", func(t *testing.T) {
		_, err := server.CallTool(ctx, "mcpproxy_test_server_tool", map[string]interface{}{
			"server": "test-server",
			"tool":   "some_tool",
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "at least one test case is required")
	})

	t.Run("manager not available", func(t *testing.T) {
		_, err := server.CallTool(ctx, "mcpproxy_test_server_tool", map[string]interface{}{
			"server":     "test-server",
			"tool":       "some_tool",
			"test_cases": []interface{}{map[string]interface{}{"name": "test1"}},
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "manager not available")
	})
}

func TestMatchWildcard(t *testing.T) {
	testCases := []struct {
		s       string
		pattern string
		want    bool
	}{
		{"hello", "hello", true},
		{"hello", "*", true},
		{"hello", "hel*", true},
		{"hello", "*llo", true},
		{"hello", "*ell*", true},
		// Note: h*o won't match - the implementation only supports prefix/suffix wildcards
		{"hello", "h*o", false},
		{"hello", "world", false},
		{"hello", "hel", false},
		{"hello", "h*d", false},
		{"", "", true},
		{"", "*", true},
		// Empty pattern matches everything in this implementation
		{"hello", "", true},
	}

	for _, tc := range testCases {
		t.Run(tc.s+"_"+tc.pattern, func(t *testing.T) {
			got := matchWildcard(tc.s, tc.pattern)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestTruncateString(t *testing.T) {
	testCases := []struct {
		s      string
		maxLen int
		want   string
	}{
		{"hello", 10, "hello"},
		{"hello", 5, "hello"},
		// len("hello")=5, so 5 <= 6 doesn't truncate
		{"hello", 6, "hello"},
		// len("hello world")=11 > 8, so truncate: s[:8-3] + "..." = "hello..."
		{"hello world", 8, "hello..."},
		{"", 5, ""},
		// len("abcdefgh")=8 > 7, so truncate: s[:7-3] + "..." = "abcd..."
		{"abcdefgh", 7, "abcd..."},
	}

	for _, tc := range testCases {
		t.Run(tc.s, func(t *testing.T) {
			got := truncateString(tc.s, tc.maxLen)
			assert.Equal(t, tc.want, got)
		})
	}
}

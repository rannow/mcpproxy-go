package logs

import (
	"context"
	"mcpproxy-go/internal/config"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCommunicationLogger_Disabled(t *testing.T) {
	// Test with disabled communication logging
	logConfig := &config.LogConfig{
		Communication: &config.CommunicationLogConfig{
			Enabled: false,
		},
	}

	logger, err := NewCommunicationLogger(logConfig)
	require.NoError(t, err)
	assert.False(t, logger.IsEnabled())
}

func TestNewCommunicationLogger_Enabled(t *testing.T) {
	// Create temp directory for test logs
	tempDir, err := os.MkdirTemp("", "mcpproxy-comm-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Test with enabled communication logging
	logConfig := &config.LogConfig{
		Level:      "debug",
		LogDir:     tempDir,
		MaxSize:    1,
		MaxBackups: 3,
		MaxAge:     7,
		Compress:   false,
		Communication: &config.CommunicationLogConfig{
			Enabled:         true,
			Filename:        "test-communication.log",
			LogRequests:     true,
			LogResponses:    true,
			LogToolCalls:    true,
			LogErrors:       true,
			IncludePayload:  true,
			MaxPayloadSize:  1024,
			IncludeHeaders:  true,
			FilterSensitive: true,
		},
	}

	logger, err := NewCommunicationLogger(logConfig)
	require.NoError(t, err)
	assert.True(t, logger.IsEnabled())
	assert.NotNil(t, logger.GetConfig())

	// Cleanup
	require.NoError(t, logger.Close())
}

func TestCommunicationLogger_LogRequest(t *testing.T) {
	// Create temp directory for test logs
	tempDir, err := os.MkdirTemp("", "mcpproxy-comm-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	logConfig := &config.LogConfig{
		Level:      "debug",
		LogDir:     tempDir,
		MaxSize:    1,
		MaxBackups: 3,
		MaxAge:     7,
		Compress:   false,
		Communication: &config.CommunicationLogConfig{
			Enabled:         true,
			Filename:        "test-communication.log",
			LogRequests:     true,
			LogResponses:    true,
			LogToolCalls:    true,
			LogErrors:       true,
			IncludePayload:  true,
			MaxPayloadSize:  1024,
			IncludeHeaders:  false,
			FilterSensitive: false,
		},
	}

	logger, err := NewCommunicationLogger(logConfig)
	require.NoError(t, err)
	defer logger.Close()

	ctx := context.Background()
	payload := map[string]interface{}{
		"name": "test:tool",
		"args": map[string]interface{}{
			"message": "Hello World",
		},
	}
	headers := map[string]interface{}{
		"Content-Type": "application/json",
	}

	// Log a request
	logger.LogRequest(ctx, "call_tool", payload, headers, "test-request-123")

	// Verify log file was created
	logFilePath := filepath.Join(tempDir, "test-communication.log")
	assert.FileExists(t, logFilePath)

	// Read and verify log content
	content, err := os.ReadFile(logFilePath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "communication_event")
	assert.Contains(t, string(content), "test-request-123")
	assert.Contains(t, string(content), "call_tool")
	assert.Contains(t, string(content), "incoming")
}

func TestCommunicationLogger_LogToolCall(t *testing.T) {
	// Create temp directory for test logs
	tempDir, err := os.MkdirTemp("", "mcpproxy-comm-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	logConfig := &config.LogConfig{
		Level:      "debug",
		LogDir:     tempDir,
		MaxSize:    1,
		MaxBackups: 3,
		MaxAge:     7,
		Compress:   false,
		Communication: &config.CommunicationLogConfig{
			Enabled:         true,
			Filename:        "test-communication.log",
			LogRequests:     true,
			LogResponses:    true,
			LogToolCalls:    true,
			LogErrors:       true,
			IncludePayload:  true,
			MaxPayloadSize:  1024,
			IncludeHeaders:  false,
			FilterSensitive: false,
		},
	}

	logger, err := NewCommunicationLogger(logConfig)
	require.NoError(t, err)
	defer logger.Close()

	ctx := context.Background()
	payload := map[string]interface{}{
		"message": "Hello upstream server",
	}

	// Log a tool call
	logger.LogToolCall(ctx, "test-server", "echo_tool", payload, nil, "test-call-456")

	// Verify log file was created
	logFilePath := filepath.Join(tempDir, "test-communication.log")
	assert.FileExists(t, logFilePath)

	// Read and verify log content
	content, err := os.ReadFile(logFilePath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "tool_call")
	assert.Contains(t, string(content), "test-server")
	assert.Contains(t, string(content), "echo_tool")
	assert.Contains(t, string(content), "test-call-456")
	assert.Contains(t, string(content), "outgoing")
}

func TestCommunicationLogger_LogError(t *testing.T) {
	// Create temp directory for test logs
	tempDir, err := os.MkdirTemp("", "mcpproxy-comm-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	logConfig := &config.LogConfig{
		Level:      "debug",
		LogDir:     tempDir,
		MaxSize:    1,
		MaxBackups: 3,
		MaxAge:     7,
		Compress:   false,
		Communication: &config.CommunicationLogConfig{
			Enabled:         true,
			Filename:        "test-communication.log",
			LogRequests:     true,
			LogResponses:    true,
			LogToolCalls:    true,
			LogErrors:       true,
			IncludePayload:  true,
			MaxPayloadSize:  1024,
			IncludeHeaders:  false,
			FilterSensitive: false,
		},
	}

	logger, err := NewCommunicationLogger(logConfig)
	require.NoError(t, err)
	defer logger.Close()

	ctx := context.Background()
	payload := map[string]interface{}{
		"invalid": "data",
	}

	// Log an error
	logger.LogError(ctx, "Connection failed to upstream server", "test-server", "failed_tool", "call_tool", payload, "test-error-789")

	// Verify log file was created
	logFilePath := filepath.Join(tempDir, "test-communication.log")
	assert.FileExists(t, logFilePath)

	// Read and verify log content
	content, err := os.ReadFile(logFilePath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "error")
	assert.Contains(t, string(content), "Connection failed to upstream server")
	assert.Contains(t, string(content), "test-server")
	assert.Contains(t, string(content), "failed_tool")
	assert.Contains(t, string(content), "test-error-789")
	assert.Contains(t, string(content), "internal")
}

func TestCommunicationLogger_PayloadTruncation(t *testing.T) {
	// Create temp directory for test logs
	tempDir, err := os.MkdirTemp("", "mcpproxy-comm-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	logConfig := &config.LogConfig{
		Level:      "debug",
		LogDir:     tempDir,
		MaxSize:    1,
		MaxBackups: 3,
		MaxAge:     7,
		Compress:   false,
		Communication: &config.CommunicationLogConfig{
			Enabled:         true,
			Filename:        "test-communication.log",
			LogRequests:     true,
			LogResponses:    true,
			LogToolCalls:    true,
			LogErrors:       true,
			IncludePayload:  true,
			MaxPayloadSize:  50, // Small payload size to test truncation
			IncludeHeaders:  false,
			FilterSensitive: false,
		},
	}

	logger, err := NewCommunicationLogger(logConfig)
	require.NoError(t, err)
	defer logger.Close()

	ctx := context.Background()
	// Create a large payload that will be truncated
	payload := map[string]interface{}{
		"large_data": "This is a very long string that should be truncated because it exceeds the maximum payload size configured for communication logging in this test case",
		"more_data":  "Additional data to make the payload even larger",
	}

	// Log the request with large payload
	logger.LogRequest(ctx, "call_tool", payload, nil, "test-truncation")

	// Verify log file was created
	logFilePath := filepath.Join(tempDir, "test-communication.log")
	assert.FileExists(t, logFilePath)

	// Read and verify log content contains truncation indicator
	content, err := os.ReadFile(logFilePath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "truncated")
	assert.Contains(t, string(content), "true") // truncated field should be true
}

func TestCommunicationLogger_SensitiveDataFiltering(t *testing.T) {
	// Create temp directory for test logs
	tempDir, err := os.MkdirTemp("", "mcpproxy-comm-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	logConfig := &config.LogConfig{
		Level:      "debug",
		LogDir:     tempDir,
		MaxSize:    1,
		MaxBackups: 3,
		MaxAge:     7,
		Compress:   false,
		Communication: &config.CommunicationLogConfig{
			Enabled:         true,
			Filename:        "test-communication.log",
			LogRequests:     true,
			LogResponses:    true,
			LogToolCalls:    true,
			LogErrors:       true,
			IncludePayload:  true,
			MaxPayloadSize:  1024,
			IncludeHeaders:  true,
			FilterSensitive: true, // Enable sensitive data filtering
		},
	}

	logger, err := NewCommunicationLogger(logConfig)
	require.NoError(t, err)
	defer logger.Close()

	ctx := context.Background()
	// Create payload with sensitive data
	payload := map[string]interface{}{
		"api_key":     "secret-api-key-12345",
		"password":    "my-secret-password",
		"token":       "bearer-token-xyz",
		"normal_data": "this is safe data",
	}
	headers := map[string]interface{}{
		"Authorization": "Bearer secret-token",
		"Content-Type":  "application/json",
	}

	// Log the request with sensitive data
	logger.LogRequest(ctx, "call_tool", payload, headers, "test-filtering")

	// Verify log file was created
	logFilePath := filepath.Join(tempDir, "test-communication.log")
	assert.FileExists(t, logFilePath)

	// Read and verify sensitive data is filtered
	content, err := os.ReadFile(logFilePath)
	require.NoError(t, err)

	// Sensitive data should be filtered out
	assert.NotContains(t, string(content), "secret-api-key-12345")
	assert.NotContains(t, string(content), "my-secret-password")
	assert.NotContains(t, string(content), "bearer-token-xyz")
	assert.NotContains(t, string(content), "secret-token")

	// Normal data should still be present
	assert.Contains(t, string(content), "this is safe data")
	assert.Contains(t, string(content), "application/json")

	// Filtered markers should be present
	assert.Contains(t, string(content), "[FILTERED]")
}

func TestCommunicationLogger_WithDuration(t *testing.T) {
	// Create temp directory for test logs
	tempDir, err := os.MkdirTemp("", "mcpproxy-comm-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	logConfig := &config.LogConfig{
		Level:      "debug",
		LogDir:     tempDir,
		MaxSize:    1,
		MaxBackups: 3,
		MaxAge:     7,
		Compress:   false,
		Communication: &config.CommunicationLogConfig{
			Enabled:         true,
			Filename:        "test-communication.log",
			LogRequests:     true,
			LogResponses:    true,
			LogToolCalls:    true,
			LogErrors:       true,
			IncludePayload:  true,
			MaxPayloadSize:  1024,
			IncludeHeaders:  false,
			FilterSensitive: false,
		},
	}

	logger, err := NewCommunicationLogger(logConfig)
	require.NoError(t, err)
	defer logger.Close()

	ctx := context.Background()
	payload := map[string]interface{}{
		"result": "success",
	}
	duration := 250 * time.Millisecond

	// Log a tool response with duration
	logger.LogToolResponse(ctx, "test-server", "echo_tool", payload, duration, "test-duration")

	// Verify log file was created
	logFilePath := filepath.Join(tempDir, "test-communication.log")
	assert.FileExists(t, logFilePath)

	// Read and verify duration is logged
	content, err := os.ReadFile(logFilePath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "tool_response")
	assert.Contains(t, string(content), "250ms") // Duration as string
	assert.Contains(t, string(content), "test-duration")
}

func TestDefaultCommunicationLogConfig(t *testing.T) {
	config := config.DefaultCommunicationLogConfig()

	assert.False(t, config.Enabled) // Should be disabled by default
	assert.Equal(t, "communication.log", config.Filename)
	assert.True(t, config.LogRequests)
	assert.True(t, config.LogResponses)
	assert.True(t, config.LogToolCalls)
	assert.True(t, config.LogErrors)
	assert.True(t, config.IncludePayload)
	assert.Equal(t, 10240, config.MaxPayloadSize) // 10KB
	assert.False(t, config.IncludeHeaders) // Headers disabled by default for privacy
	assert.True(t, config.FilterSensitive) // Filter sensitive data by default
}
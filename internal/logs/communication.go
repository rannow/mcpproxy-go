package logs

import (
	"context"
	"encoding/json"
	"fmt"
	"mcpproxy-go/internal/config"
	"regexp"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// CommunicationLogger handles logging of communication between mcpproxy and upstream servers
type CommunicationLogger struct {
	logger    *zap.Logger
	config    *config.CommunicationLogConfig
	enabled   bool
	sensitive *regexp.Regexp
}

// CommunicationEvent represents a logged communication event
type CommunicationEvent struct {
	Timestamp   time.Time              `json:"timestamp"`
	Type        string                 `json:"type"`        // "request", "response", "tool_call", "error"
	Direction   string                 `json:"direction"`   // "incoming", "outgoing"
	ServerName  string                 `json:"server_name,omitempty"`
	ToolName    string                 `json:"tool_name,omitempty"`
	Method      string                 `json:"method,omitempty"`
	Headers     map[string]interface{} `json:"headers,omitempty"`
	Payload     interface{}            `json:"payload,omitempty"`
	PayloadSize int                    `json:"payload_size,omitempty"`
	Truncated   bool                   `json:"truncated,omitempty"`
	Error       string                 `json:"error,omitempty"`
	Duration    *time.Duration         `json:"duration,omitempty"`
	RequestID   string                 `json:"request_id,omitempty"`
}

// NewCommunicationLogger creates a new communication logger
func NewCommunicationLogger(logConfig *config.LogConfig) (*CommunicationLogger, error) {
	if logConfig == nil || logConfig.Communication == nil || !logConfig.Communication.Enabled {
		return &CommunicationLogger{
			enabled: false,
		}, nil
	}

	commConfig := logConfig.Communication

	// Create file core for communication logging
	fileLogConfig := &config.LogConfig{
		Level:         logConfig.Level,
		EnableFile:    true,
		EnableConsole: false, // Communication logs only go to file
		Filename:      commConfig.Filename,
		LogDir:        logConfig.LogDir,
		MaxSize:       logConfig.MaxSize,
		MaxBackups:    logConfig.MaxBackups,
		MaxAge:        logConfig.MaxAge,
		Compress:      logConfig.Compress,
		JSONFormat:    true, // Always use JSON format for communication logs
	}

	// Parse log level
	var level zapcore.Level
	switch logConfig.Level {
	case LogLevelTrace:
		level = zap.DebugLevel
	case LogLevelDebug:
		level = zap.DebugLevel
	case LogLevelInfo:
		level = zap.InfoLevel
	case LogLevelWarn:
		level = zap.WarnLevel
	case LogLevelError:
		level = zap.ErrorLevel
	default:
		level = zap.InfoLevel
	}

	// Create file core
	fileCore, err := createFileCore(fileLogConfig, level)
	if err != nil {
		return nil, fmt.Errorf("failed to create communication log file core: %w", err)
	}

	// Create logger
	logger := zap.New(fileCore, zap.AddCaller())

	// Compile sensitive data regex if filtering is enabled
	var sensitiveRegex *regexp.Regexp
	if commConfig.FilterSensitive {
		// Pattern to match common sensitive fields
		pattern := `(?i)(password|secret|key|token|authorization|auth|credential|private|api_key|api-key|bearer|jwt)`
		sensitiveRegex = regexp.MustCompile(pattern)
	}

	return &CommunicationLogger{
		logger:    logger,
		config:    commConfig,
		enabled:   true,
		sensitive: sensitiveRegex,
	}, nil
}

// LogRequest logs an incoming request
func (cl *CommunicationLogger) LogRequest(ctx context.Context, method string, payload interface{}, headers map[string]interface{}, requestID string) {
	if !cl.enabled || !cl.config.LogRequests {
		return
	}

	event := CommunicationEvent{
		Timestamp: time.Now(),
		Type:      "request",
		Direction: "incoming",
		Method:    method,
		RequestID: requestID,
	}

	cl.addPayloadAndHeaders(&event, payload, headers)
	cl.logEvent(&event)
}

// LogResponse logs an outgoing response
func (cl *CommunicationLogger) LogResponse(ctx context.Context, method string, payload interface{}, headers map[string]interface{}, duration time.Duration, requestID string) {
	if !cl.enabled || !cl.config.LogResponses {
		return
	}

	event := CommunicationEvent{
		Timestamp: time.Now(),
		Type:      "response",
		Direction: "outgoing",
		Method:    method,
		Duration:  &duration,
		RequestID: requestID,
	}

	cl.addPayloadAndHeaders(&event, payload, headers)
	cl.logEvent(&event)
}

// LogToolCall logs a tool call to an upstream server
func (cl *CommunicationLogger) LogToolCall(ctx context.Context, serverName, toolName string, payload interface{}, headers map[string]interface{}, requestID string) {
	if !cl.enabled || !cl.config.LogToolCalls {
		return
	}

	event := CommunicationEvent{
		Timestamp:  time.Now(),
		Type:       "tool_call",
		Direction:  "outgoing",
		ServerName: serverName,
		ToolName:   toolName,
		RequestID:  requestID,
	}

	cl.addPayloadAndHeaders(&event, payload, headers)
	cl.logEvent(&event)
}

// LogToolResponse logs a tool response from an upstream server
func (cl *CommunicationLogger) LogToolResponse(ctx context.Context, serverName, toolName string, payload interface{}, duration time.Duration, requestID string) {
	if !cl.enabled || !cl.config.LogToolCalls {
		return
	}

	event := CommunicationEvent{
		Timestamp:  time.Now(),
		Type:       "tool_response",
		Direction:  "incoming",
		ServerName: serverName,
		ToolName:   toolName,
		Duration:   &duration,
		RequestID:  requestID,
	}

	cl.addPayloadAndHeaders(&event, payload, nil)
	cl.logEvent(&event)
}

// LogError logs a communication error
func (cl *CommunicationLogger) LogError(ctx context.Context, errorMsg string, serverName, toolName, method string, payload interface{}, requestID string) {
	if !cl.enabled || !cl.config.LogErrors {
		return
	}

	event := CommunicationEvent{
		Timestamp:  time.Now(),
		Type:       "error",
		Direction:  "internal",
		ServerName: serverName,
		ToolName:   toolName,
		Method:     method,
		Error:      errorMsg,
		RequestID:  requestID,
	}

	cl.addPayloadAndHeaders(&event, payload, nil)
	cl.logEvent(&event)
}

// addPayloadAndHeaders adds payload and headers to the event, handling size limits and filtering
func (cl *CommunicationLogger) addPayloadAndHeaders(event *CommunicationEvent, payload interface{}, headers map[string]interface{}) {
	// Add headers if enabled and provided
	if cl.config.IncludeHeaders && headers != nil {
		if cl.config.FilterSensitive {
			event.Headers = cl.filterSensitiveData(headers)
		} else {
			event.Headers = headers
		}
	}

	// Add payload if enabled and provided
	if cl.config.IncludePayload && payload != nil {
		// Convert payload to JSON to measure size
		payloadBytes, err := json.Marshal(payload)
		if err != nil {
			// If we can't marshal, just store the error
			event.Payload = fmt.Sprintf("marshal_error: %v", err)
			return
		}

		event.PayloadSize = len(payloadBytes)

		// Check size limit
		if cl.config.MaxPayloadSize > 0 && event.PayloadSize > cl.config.MaxPayloadSize {
			// Truncate payload
			truncatedBytes := payloadBytes[:cl.config.MaxPayloadSize]

			// Try to unmarshal truncated bytes to maintain JSON structure
			var truncatedPayload interface{}
			if err := json.Unmarshal(truncatedBytes, &truncatedPayload); err != nil {
				// If truncated bytes don't form valid JSON, store as string
				event.Payload = fmt.Sprintf("truncated_payload: %s...", string(truncatedBytes))
			} else {
				event.Payload = truncatedPayload
			}
			event.Truncated = true
		} else {
			// Use full payload
			if cl.config.FilterSensitive {
				event.Payload = cl.filterSensitiveDataInPayload(payload)
			} else {
				event.Payload = payload
			}
		}
	}
}

// filterSensitiveData removes sensitive information from headers/metadata
func (cl *CommunicationLogger) filterSensitiveData(data map[string]interface{}) map[string]interface{} {
	if cl.sensitive == nil {
		return data
	}

	filtered := make(map[string]interface{})
	for key, value := range data {
		if cl.sensitive.MatchString(key) {
			filtered[key] = "[FILTERED]"
		} else {
			// For string values, also check the value content
			if str, ok := value.(string); ok && cl.sensitive.MatchString(str) {
				filtered[key] = "[FILTERED]"
			} else {
				filtered[key] = value
			}
		}
	}
	return filtered
}

// filterSensitiveDataInPayload removes sensitive information from payload
func (cl *CommunicationLogger) filterSensitiveDataInPayload(payload interface{}) interface{} {
	if cl.sensitive == nil {
		return payload
	}

	return cl.filterRecursive(payload)
}

// filterRecursive recursively filters sensitive data in nested structures
func (cl *CommunicationLogger) filterRecursive(data interface{}) interface{} {
	switch v := data.(type) {
	case map[string]interface{}:
		filtered := make(map[string]interface{})
		for key, value := range v {
			if cl.sensitive.MatchString(key) {
				filtered[key] = "[FILTERED]"
			} else {
				filtered[key] = cl.filterRecursive(value)
			}
		}
		return filtered
	case []interface{}:
		filtered := make([]interface{}, len(v))
		for i, item := range v {
			filtered[i] = cl.filterRecursive(item)
		}
		return filtered
	case string:
		if cl.sensitive.MatchString(v) {
			return "[FILTERED]"
		}
		return v
	default:
		return v
	}
}

// logEvent logs the communication event
func (cl *CommunicationLogger) logEvent(event *CommunicationEvent) {
	var durationField zap.Field
	if event.Duration != nil {
		durationField = zap.String("duration", event.Duration.String())
	} else {
		durationField = zap.String("duration", "")
	}

	cl.logger.Info("communication_event",
		zap.String("type", event.Type),
		zap.String("direction", event.Direction),
		zap.String("server_name", event.ServerName),
		zap.String("tool_name", event.ToolName),
		zap.String("method", event.Method),
		zap.Any("headers", event.Headers),
		zap.Any("payload", event.Payload),
		zap.Int("payload_size", event.PayloadSize),
		zap.Bool("truncated", event.Truncated),
		zap.String("error", event.Error),
		durationField,
		zap.String("request_id", event.RequestID),
	)
}

// Close closes the communication logger
func (cl *CommunicationLogger) Close() error {
	if cl.logger != nil {
		return cl.logger.Sync()
	}
	return nil
}

// IsEnabled returns whether communication logging is enabled
func (cl *CommunicationLogger) IsEnabled() bool {
	return cl.enabled
}

// GetConfig returns the communication log configuration
func (cl *CommunicationLogger) GetConfig() *config.CommunicationLogConfig {
	return cl.config
}
// Package types provides type definitions for upstream connection management.
// MED-001: ConnectionState type extracted from types.go for better separation of concerns.
package types

import "time"

// ConnectionState represents the runtime state of an upstream connection (in-memory only)
// This is separate from ServerState which is persisted configuration
type ConnectionState int

const (
	// StateDisconnected indicates the upstream is not connected
	StateDisconnected ConnectionState = iota
	// StateConnecting indicates the upstream is attempting to connect
	StateConnecting
	// StateAuthenticating indicates the upstream is performing OAuth authentication
	StateAuthenticating
	// StateDiscovering indicates the upstream is discovering available tools
	StateDiscovering
	// StateReady indicates the upstream is connected and ready for requests
	StateReady
	// StateError indicates the upstream encountered an error
	StateError
)

// String returns the string representation of the connection state
// MED-004: ConnectionState uses Title Case which is suitable for both API and UI display
// No separate DisplayString() needed since String() already returns human-readable format
func (s ConnectionState) String() string {
	switch s {
	case StateDisconnected:
		return "Disconnected"
	case StateConnecting:
		return "Connecting"
	case StateAuthenticating:
		return "Authenticating"
	case StateDiscovering:
		return "Discovering"
	case StateReady:
		return "Ready"
	case StateError:
		return "Error"
	default:
		return "Unknown"
	}
}

// ConnectionInfo holds information about the current connection state
type ConnectionInfo struct {
	State                ConnectionState `json:"state"`
	LastError            error           `json:"last_error,omitempty"`
	RetryCount           int             `json:"retry_count"`
	LastRetryTime        time.Time       `json:"last_retry_time,omitempty"`
	ServerName           string          `json:"server_name,omitempty"`
	ServerVersion        string          `json:"server_version,omitempty"`
	LastOAuthAttempt     time.Time       `json:"last_oauth_attempt,omitempty"`
	OAuthRetryCount      int             `json:"oauth_retry_count"`
	IsOAuthError         bool            `json:"is_oauth_error"`
	FirstAttemptTime     time.Time       `json:"first_attempt_time,omitempty"` // When first connection attempt was made
	ConnectedAt          time.Time       `json:"connected_at,omitempty"`       // When connection was successfully established
	ConsecutiveFailures  int             `json:"consecutive_failures"`         // Number of consecutive failures
	AutoDisabled         bool            `json:"auto_disabled"`                // Whether server was auto-disabled
	AutoDisableReason    string          `json:"auto_disable_reason,omitempty"`// Reason for auto-disable
	AutoDisableThreshold int             `json:"auto_disable_threshold"`       // Threshold for auto-disable (default: DefaultAutoDisableThreshold)
	LastSuccessTime      time.Time       `json:"last_success_time,omitempty"`  // Last successful connection
}

package storage

import (
	"encoding/json"
	"mcpproxy-go/internal/config"
	"time"
)

// Bucket names for bbolt database
const (
	UpstreamsBucket       = "upstreams"
	ToolStatsBucket       = "toolstats"
	ToolHashBucket        = "toolhash"
	ToolMetadataBucket    = "tool_metadata"    // Store complete tool metadata for lazy loading
	OAuthTokenBucket      = "oauth_tokens" //nolint:gosec // bucket name, not a credential
	OAuthCompletionBucket = "oauth_completion"
	MetaBucket            = "meta"
	CacheBucket           = "cache"
	CacheStatsBucket      = "cache_stats"
)

// Meta keys
const (
	SchemaVersionKey = "schema"
)

// Current schema version
const CurrentSchemaVersion = 1

// UpstreamRecord represents an upstream server record in storage
type UpstreamRecord struct {
	ID            string                  `json:"id"`
	Name          string                  `json:"name"`
	Description   string                  `json:"description,omitempty"`
	URL           string                  `json:"url,omitempty"`
	Protocol      string                  `json:"protocol,omitempty"` // stdio, http, sse, streamable-http, auto
	Command       string                  `json:"command,omitempty"`
	Args          []string                `json:"args,omitempty"`
	WorkingDir    string                  `json:"working_dir,omitempty"` // Working directory for stdio servers
	Env           map[string]string       `json:"env,omitempty"`
	Headers       map[string]string       `json:"headers,omitempty"` // For HTTP authentication
	OAuth         *config.OAuthConfig     `json:"oauth,omitempty"`   // OAuth configuration
	RepositoryURL string                  `json:"repository_url,omitempty"` // GitHub/Repository URL for the MCP server
	Enabled       bool                    `json:"enabled"`
	Quarantined   bool                    `json:"quarantined"` // Security quarantine status
	Created       time.Time               `json:"created"`
	Updated       time.Time               `json:"updated"`
	Isolation     *config.IsolationConfig `json:"isolation,omitempty"` // Per-server isolation settings
	GroupID       int                     `json:"group_id,omitempty"`
	GroupName     string                  `json:"group_name,omitempty"`

	// Connection history for prioritization
	EverConnected            bool      `json:"ever_connected,omitempty"`
	LastSuccessfulConnection time.Time `json:"last_successful_connection,omitempty"`
	ToolCount                int       `json:"tool_count,omitempty"`

	// Lazy loading and health check configuration
	StartOnBoot              bool      `json:"start_on_boot,omitempty"`
	HealthCheck              bool      `json:"health_check,omitempty"`

	// Auto-disable state (for servers automatically disabled due to failures)
	AutoDisabled         bool   `json:"auto_disabled,omitempty"`
	AutoDisableReason    string `json:"auto_disable_reason,omitempty"`
	AutoDisableThreshold int    `json:"auto_disable_threshold,omitempty"`
}

// ToolStatRecord represents tool usage statistics
type ToolStatRecord struct {
	ToolName string    `json:"tool_name"`
	Count    uint64    `json:"count"`
	LastUsed time.Time `json:"last_used"`
}

// ToolHashRecord represents a tool hash for change detection
type ToolHashRecord struct {
	ToolName string    `json:"tool_name"`
	Hash     string    `json:"hash"`
	Updated  time.Time `json:"updated"`
}

// ToolMetadataRecord represents complete tool metadata stored in database
// This enables lazy loading by providing tools from DB without calling ListTools
type ToolMetadataRecord struct {
	ServerID    string                 `json:"server_id"`
	ToolName    string                 `json:"tool_name"`     // Unprefixed tool name
	PrefixedName string                `json:"prefixed_name"` // server_id:tool_name
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"input_schema,omitempty"`
	Created     time.Time              `json:"created"`
	Updated     time.Time              `json:"updated"`
}

// OAuthTokenRecord represents stored OAuth tokens for a server
type OAuthTokenRecord struct {
	ServerName   string    `json:"server_name"`
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	TokenType    string    `json:"token_type"`
	ExpiresAt    time.Time `json:"expires_at"`
	Scopes       []string  `json:"scopes,omitempty"`
	Created      time.Time `json:"created"`
	Updated      time.Time `json:"updated"`
}

// OAuthCompletionEvent represents an OAuth completion event for cross-process notification
type OAuthCompletionEvent struct {
	ServerName  string     `json:"server_name"`
	CompletedAt time.Time  `json:"completed_at"`
	ProcessedAt *time.Time `json:"processed_at,omitempty"` // Nil if not yet processed by server
}

// MarshalBinary implements encoding.BinaryMarshaler
func (u *UpstreamRecord) MarshalBinary() ([]byte, error) {
	return json.Marshal(u)
}

// UnmarshalBinary implements encoding.BinaryUnmarshaler
func (u *UpstreamRecord) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, u)
}

// MarshalBinary implements encoding.BinaryMarshaler
func (t *ToolStatRecord) MarshalBinary() ([]byte, error) {
	return json.Marshal(t)
}

// UnmarshalBinary implements encoding.BinaryUnmarshaler
func (t *ToolStatRecord) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, t)
}

// MarshalBinary implements encoding.BinaryMarshaler
func (h *ToolHashRecord) MarshalBinary() ([]byte, error) {
	return json.Marshal(h)
}

// UnmarshalBinary implements encoding.BinaryUnmarshaler
func (h *ToolHashRecord) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, h)
}

// MarshalBinary implements encoding.BinaryMarshaler
func (o *OAuthTokenRecord) MarshalBinary() ([]byte, error) {
	return json.Marshal(o)
}

// UnmarshalBinary implements encoding.BinaryUnmarshaler
func (o *OAuthTokenRecord) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, o)
}

// MarshalBinary implements encoding.BinaryMarshaler
func (e *OAuthCompletionEvent) MarshalBinary() ([]byte, error) {
	return json.Marshal(e)
}

// UnmarshalBinary implements encoding.BinaryUnmarshaler
func (e *OAuthCompletionEvent) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, e)
}

// MarshalBinary implements encoding.BinaryMarshaler
func (t *ToolMetadataRecord) MarshalBinary() ([]byte, error) {
	return json.Marshal(t)
}

// UnmarshalBinary implements encoding.BinaryUnmarshaler
func (t *ToolMetadataRecord) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, t)
}

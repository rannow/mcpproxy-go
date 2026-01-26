package config

import (
	"encoding/json"
	"fmt"
	"mcpproxy-go/internal/secureenv"
	"os"
	"path/filepath"
	"time"
)

const (
	defaultPort = ":8080"
)

// Duration is a wrapper around time.Duration that can be marshaled to/from JSON
type Duration time.Duration

// MarshalJSON implements json.Marshaler interface
func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(d).String())
}

// UnmarshalJSON implements json.Unmarshaler interface
func (d *Duration) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	parsed, err := time.ParseDuration(s)
	if err != nil {
		return fmt.Errorf("invalid duration format: %w", err)
	}

	*d = Duration(parsed)
	return nil
}

// Duration returns the underlying time.Duration
func (d Duration) Duration() time.Duration {
	return time.Duration(d)
}

// Config represents the main configuration structure
type Config struct {
	Listen            string          `json:"listen" mapstructure:"listen"`
	DataDir           string          `json:"data_dir" mapstructure:"data-dir"`
	EnableTray        bool            `json:"enable_tray" mapstructure:"tray"`
	DebugSearch       bool            `json:"debug_search" mapstructure:"debug-search"`
	Servers           []*ServerConfig `json:"mcpServers" mapstructure:"servers"`
	TopK              int             `json:"top_k" mapstructure:"top-k"`
	ToolsLimit        int             `json:"tools_limit" mapstructure:"tools-limit"`
	ToolResponseLimit int             `json:"tool_response_limit" mapstructure:"tool-response-limit"`
	CallToolTimeout   Duration        `json:"call_tool_timeout" mapstructure:"call-tool-timeout"`

	// Environment configuration for secure variable filtering
	Environment *secureenv.EnvConfig `json:"environment,omitempty" mapstructure:"environment"`

	// Logging configuration
	Logging *LogConfig `json:"logging,omitempty" mapstructure:"logging"`

	// Security settings
	ReadOnlyMode      bool `json:"read_only_mode" mapstructure:"read-only-mode"`
	DisableManagement bool `json:"disable_management" mapstructure:"disable-management"`
	AllowServerAdd    bool `json:"allow_server_add" mapstructure:"allow-server-add"`
	AllowServerRemove bool `json:"allow_server_remove" mapstructure:"allow-server-remove"`

	// Prompts settings
	EnablePrompts bool `json:"enable_prompts" mapstructure:"enable-prompts"`

	// Repository detection settings
	CheckServerRepo bool `json:"check_server_repo" mapstructure:"check-server-repo"`

	// Docker isolation settings
	DockerIsolation *DockerIsolationConfig `json:"docker_isolation,omitempty" mapstructure:"docker-isolation"`

	// Registries configuration for MCP server discovery
	Registries []RegistryEntry `json:"registries,omitempty" mapstructure:"registries"`

	// Groups configuration
	Groups []GroupConfig `json:"groups,omitempty" mapstructure:"groups"`

	// Server group assignments
	ServerGroupAssignments map[string]string `json:"server_group_assignments,omitempty" mapstructure:"server-group-assignments"`

	// GitHub repository URL for the project
	GitHubURL string `json:"github_url,omitempty" mapstructure:"github-url"`

	// Startup script configuration, executed when mcpproxy starts
	StartupScript *StartupScriptConfig `json:"startup_script,omitempty" mapstructure:"startup-script"`

	// Maximum number of concurrent server connections during startup
	MaxConcurrentConnections int `json:"max_concurrent_connections" mapstructure:"max-concurrent-connections"`

	// Lazy loading configuration - only connect to servers when their tools are called
	EnableLazyLoading bool `json:"enable_lazy_loading" mapstructure:"enable-lazy-loading"`

	// Tool cache TTL in seconds (default: 300 = 5 minutes)
	ToolCacheTTL int `json:"tool_cache_ttl" mapstructure:"tool-cache-ttl"`

	// LLM configuration for AI Diagnostic Agent
	LLM *LLMConfig `json:"llm,omitempty" mapstructure:"llm"`

	// Auto-disable threshold - number of consecutive failures before auto-disabling (default: 3)
	AutoDisableThreshold int `json:"auto_disable_threshold,omitempty" mapstructure:"auto-disable-threshold"`

	// PersistAutoDisableToConfig controls whether auto-disable state is saved to config file.
	// When false (default), auto-disable state is only stored in database, keeping servers as "active" in config.
	// When true, auto-disable state is written to both database AND config file.
	PersistAutoDisableToConfig bool `json:"persist_auto_disable_to_config,omitempty" mapstructure:"persist-auto-disable-to-config"`

	// Semantic search configuration
	SemanticSearch *SemanticSearchConfig `json:"semantic_search,omitempty" mapstructure:"semantic-search"`
}

// SemanticSearchConfig represents semantic search configuration
type SemanticSearchConfig struct {
	Enabled       bool    `json:"enabled" mapstructure:"enabled"`               // Enable semantic search
	HybridMode    bool    `json:"hybrid_mode" mapstructure:"hybrid-mode"`       // Combine BM25 and semantic search
	HybridWeight  float64 `json:"hybrid_weight" mapstructure:"hybrid-weight"`   // Weight for semantic search in hybrid mode (0.0-1.0)
	MinSimilarity float32 `json:"min_similarity" mapstructure:"min-similarity"` // Minimum similarity threshold (0.0-1.0)
}

// LLMConfig represents LLM provider configuration for AI Diagnostic Agent
type LLMConfig struct {
	Provider string `json:"provider" mapstructure:"provider"` // "openai", "anthropic", "ollama"
	Model    string `json:"model,omitempty" mapstructure:"model"` // Model name (e.g., "gpt-4o-mini", "claude-3-5-sonnet-20241022", "llama2")

	// API Keys - prefer config over environment variables for GUI apps
	OpenAIKey    string `json:"openai_api_key,omitempty" mapstructure:"openai_api_key"`
	AnthropicKey string `json:"anthropic_api_key,omitempty" mapstructure:"anthropic_api_key"`

	// Ollama specific settings
	OllamaURL string `json:"ollama_url,omitempty" mapstructure:"ollama_url"` // Default: "http://localhost:11434"

	// General settings
	Temperature float64 `json:"temperature,omitempty" mapstructure:"temperature"` // 0.0 - 1.0, default: 0.7
	MaxTokens   int     `json:"max_tokens,omitempty" mapstructure:"max_tokens"`   // Default: 2000
}

// LogConfig represents logging configuration
type LogConfig struct {
	Level                string              `json:"level" mapstructure:"level"`
	EnableFile           bool                `json:"enable_file" mapstructure:"enable-file"`
	EnableConsole        bool                `json:"enable_console" mapstructure:"enable-console"`
	Filename             string              `json:"filename" mapstructure:"filename"`
	LogDir               string              `json:"log_dir,omitempty" mapstructure:"log-dir"` // Custom log directory
	MaxSize              int                 `json:"max_size" mapstructure:"max-size"`         // MB
	MaxBackups           int                 `json:"max_backups" mapstructure:"max-backups"`   // number of backup files
	MaxAge               int                 `json:"max_age" mapstructure:"max-age"`           // days
	Compress             bool                `json:"compress" mapstructure:"compress"`
	JSONFormat           bool                `json:"json_format" mapstructure:"json-format"`
	Communication        *CommunicationLogConfig `json:"communication,omitempty" mapstructure:"communication"` // Communication logging configuration
}

// CommunicationLogConfig represents communication logging configuration
type CommunicationLogConfig struct {
	Enabled           bool   `json:"enabled" mapstructure:"enabled"`                       // Enable communication logging
	Filename          string `json:"filename" mapstructure:"filename"`                     // Communication log filename
	LogRequests       bool   `json:"log_requests" mapstructure:"log-requests"`             // Log incoming requests
	LogResponses      bool   `json:"log_responses" mapstructure:"log-responses"`           // Log outgoing responses
	LogToolCalls      bool   `json:"log_tool_calls" mapstructure:"log-tool-calls"`         // Log tool calls to upstream servers
	LogErrors         bool   `json:"log_errors" mapstructure:"log-errors"`                 // Log communication errors
	IncludePayload    bool   `json:"include_payload" mapstructure:"include-payload"`       // Include full payload in logs
	MaxPayloadSize    int    `json:"max_payload_size" mapstructure:"max-payload-size"`     // Maximum payload size to log (bytes)
	IncludeHeaders    bool   `json:"include_headers" mapstructure:"include-headers"`       // Include HTTP headers in logs
	FilterSensitive   bool   `json:"filter_sensitive" mapstructure:"filter-sensitive"`     // Filter sensitive data like API keys
}

// StartupScriptConfig represents configuration for an optional startup script that
// runs when mcpproxy launches. The script can be managed via tray and MCP tools.
type StartupScriptConfig struct {
    Enabled     bool              `json:"enabled" mapstructure:"enabled"`
    Path        string            `json:"path,omitempty" mapstructure:"path"`               // Script file path or shell command
    Shell       string            `json:"shell,omitempty" mapstructure:"shell"`             // Shell to execute with -c (default: /bin/bash)
    Args        []string          `json:"args,omitempty" mapstructure:"args"`               // Optional extra args to append after -c
    WorkingDir  string            `json:"working_dir,omitempty" mapstructure:"working_dir"`
    Env         map[string]string `json:"env,omitempty" mapstructure:"env"`
    Timeout     Duration          `json:"timeout,omitempty" mapstructure:"timeout"`         // Optional max runtime before forced stop (0 = no timeout)
}

// ServerConfig represents upstream MCP server configuration
type ServerConfig struct {
	Name          string            `json:"name,omitempty" mapstructure:"name"`
	Description   string            `json:"description,omitempty" mapstructure:"description"`
	URL           string            `json:"url,omitempty" mapstructure:"url"`
	Protocol      string            `json:"protocol,omitempty" mapstructure:"protocol"` // stdio, http, sse, streamable-http, auto
	Command       string            `json:"command,omitempty" mapstructure:"command"`
	Args          []string          `json:"args,omitempty" mapstructure:"args"`
	WorkingDir    string            `json:"working_dir,omitempty" mapstructure:"working_dir"` // Working directory for stdio servers
	Env           map[string]string `json:"env,omitempty" mapstructure:"env"`
	Headers       map[string]string `json:"headers,omitempty" mapstructure:"headers"`        // For HTTP servers
	OAuth         *OAuthConfig      `json:"oauth,omitempty" mapstructure:"oauth"`            // OAuth configuration
	RepositoryURL string            `json:"repository_url,omitempty" mapstructure:"repository_url"` // GitHub/Repository URL for the MCP server

	// Unified startup mode - determines server behavior at startup
	// Values: "active", "disabled", "quarantined", "auto_disabled", "lazy_loading"
	StartupMode   string            `json:"startup_mode,omitempty" mapstructure:"startup_mode"`

	Created       time.Time         `json:"created" mapstructure:"created"`
	Updated       time.Time         `json:"updated,omitempty" mapstructure:"updated"`
	Isolation                 *IsolationConfig  `json:"isolation,omitempty" mapstructure:"isolation"` // Per-server isolation settings
	GroupID                   int               `json:"group_id,omitempty" mapstructure:"group_id"`       // Assigned group ID (new format)
	GroupName                 string            `json:"group_name,omitempty" mapstructure:"group_name"`   // Assigned group name (legacy)

	// Connection history for prioritization
	EverConnected             bool      `json:"ever_connected,omitempty" mapstructure:"ever_connected"`                         // Has this server ever successfully connected
	LastSuccessfulConnection  time.Time `json:"last_successful_connection,omitempty" mapstructure:"last_successful_connection"` // Last successful connection time
	ToolCount                 int       `json:"tool_count,omitempty" mapstructure:"tool_count"`                                 // Number of tools discovered from this server

	// Lazy loading and connection behavior flags
	HealthCheck               bool      `json:"health_check" mapstructure:"health_check"`           // Perform regular health checks (default: false)

	// Auto-disable threshold - per-server override (0 = use global default)
	AutoDisableThreshold      int       `json:"auto_disable_threshold,omitempty" mapstructure:"auto_disable_threshold"` // Number of consecutive failures before auto-disabling

	// Connection timeout - per-server override (0 = use global default: 60s)
	// Useful for slow-starting servers like uvx-based AWS servers
	ConnectionTimeout         Duration  `json:"connection_timeout,omitempty" mapstructure:"connection_timeout"` // Connection timeout for this server

	// Auto-disable state - persisted across restarts
	AutoDisableReason         string    `json:"auto_disable_reason,omitempty" mapstructure:"auto_disable_reason"` // Reason for auto-disable

	// NOTE: "Stopped" field has been REMOVED - it was runtime-only state that should NOT be persisted
	// Use StateManager.IsUserStopped() / SetUserStopped() for runtime-only stopped state
	// When app restarts, all servers return to their original startup_mode (no persisted "stopped" state)
}

// ShouldConnectOnStartup determines if the server should connect when mcpproxy starts
// based on the StartupMode field
func (s *ServerConfig) ShouldConnectOnStartup() bool {
	return s.StartupMode == "active"
}

// IsQuarantined determines if the server is quarantined
func (s *ServerConfig) IsQuarantined() bool {
	return s.StartupMode == "quarantined"
}

// IsDisabled determines if the server is disabled
func (s *ServerConfig) IsDisabled() bool {
	return s.StartupMode == "disabled" || s.StartupMode == "auto_disabled"
}

// GetConnectionTimeout returns the effective connection timeout for this server.
// If a per-server timeout is configured (ConnectionTimeout > 0), it uses that.
// Otherwise, it returns the global DefaultConnectionTimeout.
func (s *ServerConfig) GetConnectionTimeout() time.Duration {
	if s.ConnectionTimeout > 0 {
		return time.Duration(s.ConnectionTimeout)
	}
	return DefaultConnectionTimeout
}

// MarshalJSON implements custom JSON marshaling to exclude deprecated fields
// This ensures that old boolean fields (enabled, quarantined, auto_disabled)
// are never written to the config file, even if they exist in memory from
// reading old configs.
func (s *ServerConfig) MarshalJSON() ([]byte, error) {
	// Create a type alias to avoid infinite recursion
	type Alias ServerConfig

	// Marshal to JSON using default behavior
	data, err := json.Marshal((*Alias)(s))
	if err != nil {
		return nil, err
	}

	// Unmarshal to map to filter out deprecated fields
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}

	// Remove deprecated fields
	delete(m, "enabled")
	delete(m, "quarantined")
	delete(m, "auto_disabled")
	delete(m, "start_on_boot")
	delete(m, "stopped")

	// Marshal back to JSON without deprecated fields
	return json.Marshal(m)
}

// OAuthConfig represents OAuth configuration for a server
type OAuthConfig struct {
	ClientID     string   `json:"client_id,omitempty" mapstructure:"client_id"`
	ClientSecret string   `json:"client_secret,omitempty" mapstructure:"client_secret"`
	RedirectURI  string   `json:"redirect_uri,omitempty" mapstructure:"redirect_uri"`
	Scopes       []string `json:"scopes,omitempty" mapstructure:"scopes"`
	PKCEEnabled  bool     `json:"pkce_enabled,omitempty" mapstructure:"pkce_enabled"`
}

// DockerIsolationConfig represents global Docker isolation settings
type DockerIsolationConfig struct {
	Enabled       bool              `json:"enabled" mapstructure:"enabled"`                       // Global enable/disable for Docker isolation
	DefaultImages map[string]string `json:"default_images" mapstructure:"default_images"`         // Map of runtime type to Docker image
	Registry      string            `json:"registry,omitempty" mapstructure:"registry"`           // Custom registry (defaults to docker.io)
	NetworkMode   string            `json:"network_mode,omitempty" mapstructure:"network_mode"`   // Docker network mode (default: bridge)
	MemoryLimit   string            `json:"memory_limit,omitempty" mapstructure:"memory_limit"`   // Memory limit for containers
	CPULimit      string            `json:"cpu_limit,omitempty" mapstructure:"cpu_limit"`         // CPU limit for containers
	Timeout       Duration          `json:"timeout,omitempty" mapstructure:"timeout"`             // Container startup timeout
	ExtraArgs     []string          `json:"extra_args,omitempty" mapstructure:"extra_args"`       // Additional docker run arguments
	LogDriver     string            `json:"log_driver,omitempty" mapstructure:"log_driver"`       // Docker log driver (default: json-file)
	LogMaxSize    string            `json:"log_max_size,omitempty" mapstructure:"log_max_size"`   // Maximum size of log files (default: 100m)
	LogMaxFiles   string            `json:"log_max_files,omitempty" mapstructure:"log_max_files"` // Maximum number of log files (default: 3)
}

// IsolationConfig represents per-server isolation settings
type IsolationConfig struct {
	Enabled     bool     `json:"enabled" mapstructure:"enabled"`                       // Enable Docker isolation for this server
	Image       string   `json:"image,omitempty" mapstructure:"image"`                 // Custom Docker image (overrides default)
	NetworkMode string   `json:"network_mode,omitempty" mapstructure:"network_mode"`   // Custom network mode for this server
	ExtraArgs   []string `json:"extra_args,omitempty" mapstructure:"extra_args"`       // Additional docker run arguments for this server
	WorkingDir  string   `json:"working_dir,omitempty" mapstructure:"working_dir"`     // Custom working directory in container
	LogDriver   string   `json:"log_driver,omitempty" mapstructure:"log_driver"`       // Docker log driver override for this server
	LogMaxSize  string   `json:"log_max_size,omitempty" mapstructure:"log_max_size"`   // Maximum size of log files override
	LogMaxFiles string   `json:"log_max_files,omitempty" mapstructure:"log_max_files"` // Maximum number of log files override
}

// GroupConfig represents a server group configuration
type GroupConfig struct {
	ID          int    `json:"id" mapstructure:"id"`
	Name        string `json:"name" mapstructure:"name"`
	Description string `json:"description,omitempty" mapstructure:"description"`
	Color       string `json:"color" mapstructure:"color"`
	Icon        string `json:"icon_emoji,omitempty" mapstructure:"icon_emoji"` // Emoji icon for the group
	Enabled     bool   `json:"enabled" mapstructure:"enabled"`
}

// AvailableGroupIcons returns a list of available emoji icons for groups
var AvailableGroupIcons = []string{
	"🌐", // Browser/Web
	"🔧", // Tools/Configuration
	"🧪", // Testing/Experimental
	"🗄️", // Database/Storage
	"☁️", // Cloud/Web Services
	"🎯", // Target/Goals
	"💼", // Business/Professional
	"🔔", // Notifications/Alerts
	"🏠", // Home/Default
	"🖥️", // Computer/Services
	"📊", // Analytics/Data
	"🔒", // Security
	"⚡", // Performance/Speed
	"🎨", // Design/UI
	"📱", // Mobile
	"🌟", // Featured/Important
	"🔍", // Search/Discovery
	"💾", // Storage/Backup
	"🚀", // Launch/Deployment
	"📁", // Files/Documents
	"🔗", // Integration/Links
	"⚙️", // Settings/Configuration
	"📝", // Documentation/Notes
	"🎭", // Testing/QA
	"🌈", // Diverse/Mixed
	"🔐", // Authentication
	"📡", // Network/Communication
	"🎮", // Gaming/Interactive
	"🏗️", // Building/Construction
	"🔬", // Research/Science
	"📈", // Growth/Metrics
	"🌍", // Global/International
	"🎪", // Entertainment
	"🔊", // Audio/Sound
	"📸", // Media/Images
	"🎥", // Video/Streaming
	"📚", // Libraries/Knowledge
	"💡", // Ideas/Innovation
	"🛠️", // Tools/Utilities
	"🎁", // Packages/Resources
}

// RegistryEntry represents a registry in the configuration
type RegistryEntry struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	URL         string      `json:"url"`
	ServersURL  string      `json:"servers_url,omitempty"`
	Tags        []string    `json:"tags,omitempty"`
	Protocol    string      `json:"protocol,omitempty"`
	Count       interface{} `json:"count,omitempty"` // number or string
}

// CursorMCPConfig represents the structure for Cursor IDE MCP configuration
type CursorMCPConfig struct {
	MCPServers map[string]CursorServerConfig `json:"mcpServers"`
}

// CursorServerConfig represents a single server configuration in Cursor format
type CursorServerConfig struct {
	Command       string            `json:"command,omitempty"`
	Args          []string          `json:"args,omitempty"`
	Env           map[string]string `json:"env,omitempty"`
	URL           string            `json:"url,omitempty"`
	Headers       map[string]string `json:"headers,omitempty"`
	RepositoryURL string            `json:"repository_url,omitempty"`
}

// ConvertFromCursorFormat converts Cursor IDE format to our internal format
func ConvertFromCursorFormat(cursorConfig *CursorMCPConfig) []*ServerConfig {
	var servers []*ServerConfig

	for name, serverConfig := range cursorConfig.MCPServers {
		server := &ServerConfig{
			Name:          name,
			StartupMode:   "active",  // Default to active for Cursor imports
			Created:       time.Now(),
			RepositoryURL: serverConfig.RepositoryURL,
		}

		if serverConfig.Command != "" {
			server.Command = serverConfig.Command
			server.Args = serverConfig.Args
			server.Env = serverConfig.Env
			server.Protocol = "stdio"
		} else if serverConfig.URL != "" {
			server.URL = serverConfig.URL
			server.Headers = serverConfig.Headers
			server.Protocol = "http"
		}

		servers = append(servers, server)
	}

	return servers
}

// ToolMetadata represents tool information stored in the index
type ToolMetadata struct {
	Name        string    `json:"name"`
	ServerName  string    `json:"server_name"`
	Description string    `json:"description"`
	ParamsJSON  string    `json:"params_json"`
	Hash        string    `json:"hash"`
	Created     time.Time `json:"created"`
	Updated     time.Time `json:"updated"`
}

// ToolRegistration represents a tool registration
type ToolRegistration struct {
	Name         string                 `json:"name"`
	Description  string                 `json:"description"`
	InputSchema  map[string]interface{} `json:"input_schema"`
	ServerName   string                 `json:"server_name"`
	OriginalName string                 `json:"original_name"`
}

// SearchResult represents a search result with score
type SearchResult struct {
	Tool  *ToolMetadata `json:"tool"`
	Score float64       `json:"score"`
}

// ToolStats represents tool statistics
type ToolStats struct {
	TotalTools int             `json:"total_tools"`
	TopTools   []ToolStatEntry `json:"top_tools"`
}

// ToolStatEntry represents a single tool stat entry
type ToolStatEntry struct {
	ToolName string `json:"tool_name"`
	Count    uint64 `json:"count"`
}

// DefaultCommunicationLogConfig returns default communication logging configuration
func DefaultCommunicationLogConfig() *CommunicationLogConfig {
	return &CommunicationLogConfig{
		Enabled:           false, // Disabled by default to avoid excessive logging
		Filename:          "communication.log",
		LogRequests:       true,
		LogResponses:      true,
		LogToolCalls:      true,
		LogErrors:         true,
		IncludePayload:    true,
		MaxPayloadSize:    10240, // 10KB default limit
		IncludeHeaders:    false, // Headers disabled by default for privacy
		FilterSensitive:   true,  // Filter sensitive data by default
	}
}

// DefaultDockerIsolationConfig returns default Docker isolation configuration
func DefaultDockerIsolationConfig() *DockerIsolationConfig {
	return &DockerIsolationConfig{
		Enabled: false, // Disabled by default for backward compatibility
		DefaultImages: map[string]string{
			// Python environments - using full images for Git and build tool support
			"python":  "python:3.11",
			"python3": "python:3.11",
			"uvx":     "python:3.11", // Full image needed for git+https:// installs
			"pip":     "python:3.11",
			"pipx":    "python:3.11",

			// Node.js environments - using full images for Git and native module support
			"node": "node:20",
			"npm":  "node:20",
			"npx":  "node:20", // Full image needed for git dependencies and native modules
			"yarn": "node:20",

			// Go binaries
			"go": "golang:1.21-alpine",

			// Rust binaries
			"cargo": "rust:1.75-slim",
			"rustc": "rust:1.75-slim",

			// Generic binary execution
			"binary": "alpine:3.18",

			// Shell/script execution
			"sh":   "alpine:3.18",
			"bash": "alpine:3.18",

			// Ruby
			"ruby": "ruby:3.2-alpine",
			"gem":  "ruby:3.2-alpine",

			// PHP
			"php":      "php:8.2-cli-alpine",
			"composer": "php:8.2-cli-alpine",
		},
		Registry:    "docker.io",                // Default Docker Hub registry
		NetworkMode: "bridge",                   // Default Docker network mode
		MemoryLimit: "512m",                     // Default memory limit
		CPULimit:    "1.0",                      // Default CPU limit (1 core)
		Timeout:     Duration(30 * time.Second), // 30 second startup timeout
		ExtraArgs:   []string{},                 // No extra args by default
		LogDriver:   "",                         // Use Docker system default (empty = no override)
		LogMaxSize:  "100m",                     // Default maximum log file size (only used if json-file driver is set)
		LogMaxFiles: "3",                        // Default maximum number of log files (only used if json-file driver is set)
	}
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		Listen:            defaultPort,
		DataDir:           "", // Will be set to ~/.mcpproxy by loader
		EnableTray:        true,
		DebugSearch:       false,
		Servers:           []*ServerConfig{},
		TopK:              5,
		ToolsLimit:        15,
		ToolResponseLimit: 20000,                     // Default 20000 characters
		CallToolTimeout:   Duration(2 * time.Minute), // Default 2 minutes for tool calls

		// Default secure environment configuration
		Environment: secureenv.DefaultEnvConfig(),

		// Default logging configuration
		Logging: &LogConfig{
			Level:         "info",
			EnableFile:    false, // Changed: Console by default
			EnableConsole: true,
			Filename:      "main.log",
			MaxSize:       10, // 10MB
			MaxBackups:    5,  // 5 backup files
			MaxAge:        30, // 30 days
			Compress:      true,
			JSONFormat:    false, // Use console format for readability
			Communication: DefaultCommunicationLogConfig(),
		},

		// Security defaults - permissive by default for compatibility
		ReadOnlyMode:      false,
		DisableManagement: false,
		AllowServerAdd:    true,
		AllowServerRemove: true,

		// Prompts enabled by default
		EnablePrompts: true,

		// Repository detection enabled by default
		CheckServerRepo: true,

		// Default Docker isolation settings
		DockerIsolation: DefaultDockerIsolationConfig(),

		// Default registries for MCP server discovery
		Registries: []RegistryEntry{
			{
				ID:          "pulse",
				Name:        "Pulse MCP",
				Description: "Browse and discover MCP use-cases, servers, clients, and news",
				URL:         "https://www.pulsemcp.com/",
				ServersURL:  "https://api.pulsemcp.com/v0beta/servers",
				Tags:        []string{"verified"},
				Protocol:    "custom/pulse",
			},
			{
				ID:          "docker-mcp-catalog",
				Name:        "Docker MCP Catalog",
				Description: "A collection of secure, high-quality MCP servers as docker images",
				URL:         "https://hub.docker.com/catalogs/mcp",
				ServersURL:  "https://hub.docker.com/v2/repositories/mcp/",
				Tags:        []string{"verified"},
				Protocol:    "custom/docker",
			},
			{
				ID:          "fleur",
				Name:        "Fleur",
				Description: "Fleur is the app store for Claude",
				URL:         "https://www.fleurmcp.com/",
				ServersURL:  "https://raw.githubusercontent.com/fleuristes/app-registry/refs/heads/main/apps.json",
				Tags:        []string{"verified"},
				Protocol:    "custom/fleur",
			},
			{
				ID:          "azure-mcp-demo",
				Name:        "Azure MCP Registry Demo",
				Description: "A reference implementation of MCP registry using Azure API Center",
				URL:         "https://demo.registry.azure-mcp.net/",
				ServersURL:  "https://demo.registry.azure-mcp.net/v0/servers",
				Tags:        []string{"verified", "demo", "azure", "reference"},
				Protocol:    "mcp/v0",
			},
			{
				ID:          "remote-mcp-servers",
				Name:        "Remote MCP Servers",
				Description: "Community-maintained list of remote Model Context Protocol servers",
				URL:         "https://remote-mcp-servers.com/",
				ServersURL:  "https://remote-mcp-servers.com/api/servers",
				Tags:        []string{"verified", "community", "remote"},
				Protocol:    "custom/remote",
			},
		},

		// Default GitHub repository URL
		GitHubURL: "https://github.com/smart-mcp-proxy/mcpproxy-go",

		// Default startup script configuration (disabled by default)
		StartupScript: &StartupScriptConfig{
			Enabled:    false,
			Shell:      "/bin/bash",
			Path:       "",
			Args:       []string{},
			WorkingDir: "",
			Env:        map[string]string{},
			Timeout:    Duration(0),
		},

		// Default concurrent connections: 10 servers at once (reduced to avoid resource contention)
		MaxConcurrentConnections: 10,

		// Default LLM configuration (tries environment variables as fallback)
		LLM: &LLMConfig{
			Provider:    "openai",      // Default to OpenAI
			Model:       "gpt-4o-mini", // Cost-effective model
			Temperature: 0.7,
			MaxTokens:   2000,
			OllamaURL:   "http://localhost:11434", // Default Ollama endpoint
		},

		// Default semantic search configuration
		SemanticSearch: &SemanticSearchConfig{
			Enabled:       false, // Disabled by default (opt-in)
			HybridMode:    true,  // When enabled, use hybrid mode by default
			HybridWeight:  0.5,   // Equal weight between BM25 and semantic (0.0-1.0)
			MinSimilarity: 0.1,   // Minimum similarity threshold (0.0-1.0)
		},
	}
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Listen == "" {
		c.Listen = defaultPort
	}
	if c.TopK <= 0 {
		c.TopK = 5
	}
	if c.ToolsLimit <= 0 {
		c.ToolsLimit = 15
	}
	if c.ToolResponseLimit < 0 {
		c.ToolResponseLimit = 0 // 0 means disabled
	}
	if c.CallToolTimeout.Duration() <= 0 {
		c.CallToolTimeout = Duration(2 * time.Minute) // Default to 2 minutes
	}
	if c.MaxConcurrentConnections <= 0 {
		c.MaxConcurrentConnections = 10 // Default to 10 concurrent connections (reduced to avoid resource contention)
	}

	// Ensure Environment config is not nil
	if c.Environment == nil {
		c.Environment = secureenv.DefaultEnvConfig()
	}

	// Ensure DockerIsolation config is not nil
	if c.DockerIsolation == nil {
		c.DockerIsolation = DefaultDockerIsolationConfig()
	}

	// Ensure Logging config is not nil
	if c.Logging == nil {
		c.Logging = &LogConfig{
			Level:         "info",
			EnableFile:    false,
			EnableConsole: true,
			Filename:      "main.log",
			MaxSize:       10,
			MaxBackups:    5,
			MaxAge:        30,
			Compress:      true,
			JSONFormat:    false,
			Communication: DefaultCommunicationLogConfig(),
		}
	}

	// Ensure Communication config is not nil
	if c.Logging.Communication == nil {
		c.Logging.Communication = DefaultCommunicationLogConfig()
	}

	// Ensure StartupScript defaults
	if c.StartupScript == nil {
		c.StartupScript = &StartupScriptConfig{Enabled: false, Shell: "/bin/bash", Args: []string{}, Env: map[string]string{}}
	}
	if c.StartupScript.Shell == "" {
		c.StartupScript.Shell = "/bin/bash"
	}
	// Default startup script path under data dir
	if c.StartupScript.Path == "" {
		// Ensure DataDir is set before this (loader guarantees this)
		if c.DataDir != "" {
			c.StartupScript.Path = filepath.Join(c.DataDir, "startup-script.sh")
		}
	}
	// Create default startup script file if path is set but file does not exist
	if c.StartupScript.Path != "" {
		if _, err := os.Stat(c.StartupScript.Path); os.IsNotExist(err) {
			// Ensure directory exists
			_ = os.MkdirAll(filepath.Dir(c.StartupScript.Path), 0755)
			content := []byte("#!/bin/bash\n# mcpproxy startup script\n# Add your initialization commands below\n\nexit 0\n")
			_ = os.WriteFile(c.StartupScript.Path, content, 0755)
		} else if err == nil {
			// Ensure executable bit is set (best-effort)
			_ = os.Chmod(c.StartupScript.Path, 0755)
		}
	}

	// Validate communication log configuration
	if c.Logging.Communication.MaxPayloadSize < 0 {
		c.Logging.Communication.MaxPayloadSize = 10240 // Default 10KB
	}
	if c.Logging.Communication.Filename == "" {
		c.Logging.Communication.Filename = "communication.log"
	}

	// Ensure LLM config is not nil
	if c.LLM == nil {
		c.LLM = &LLMConfig{
			Provider:    "openai",
			Model:       "gpt-4o-mini",
			Temperature: 0.7,
			MaxTokens:   2000,
			OllamaURL:   "http://localhost:11434",
		}
	}

	// Validate LLM configuration
	if c.LLM.Provider == "" {
		c.LLM.Provider = "openai"
	}
	if c.LLM.Model == "" {
		// Set default model based on provider
		switch c.LLM.Provider {
		case "openai":
			c.LLM.Model = "gpt-4o-mini"
		case "anthropic":
			c.LLM.Model = "claude-3-5-sonnet-20241022"
		case "ollama":
			c.LLM.Model = "llama2"
		default:
			c.LLM.Model = "gpt-4o-mini"
		}
	}
	if c.LLM.Temperature <= 0 {
		c.LLM.Temperature = 0.7
	}
	if c.LLM.MaxTokens <= 0 {
		c.LLM.MaxTokens = 2000
	}
	if c.LLM.OllamaURL == "" {
		c.LLM.OllamaURL = "http://localhost:11434"
	}

	// Ensure SemanticSearch config is not nil
	if c.SemanticSearch == nil {
		c.SemanticSearch = &SemanticSearchConfig{
			Enabled:       false,
			HybridMode:    true,
			HybridWeight:  0.5,
			MinSimilarity: 0.1,
		}
	}

	// Validate semantic search configuration
	if c.SemanticSearch.HybridWeight < 0 {
		c.SemanticSearch.HybridWeight = 0
	}
	if c.SemanticSearch.HybridWeight > 1 {
		c.SemanticSearch.HybridWeight = 1
	}
	if c.SemanticSearch.MinSimilarity < 0 {
		c.SemanticSearch.MinSimilarity = 0
	}
	if c.SemanticSearch.MinSimilarity > 1 {
		c.SemanticSearch.MinSimilarity = 1
	}

	// Validate server configurations
	for _, server := range c.Servers {
		// Validate startup_mode if set
		if server.StartupMode != "" {
			if err := ValidateStartupMode(server.StartupMode); err != nil {
				return fmt.Errorf("server %s: %w", server.Name, err)
			}
		}
	}

	return nil
}

// ValidateStartupMode validates that a startup_mode value is valid
func ValidateStartupMode(mode string) error {
	validModes := map[string]bool{
		"active":        true,
		"disabled":      true,
		"quarantined":   true,
		"auto_disabled": true,
		"lazy_loading":  true,
	}

	if !validModes[mode] {
		return fmt.Errorf("invalid startup_mode: %s (must be one of: active, disabled, quarantined, auto_disabled, lazy_loading)", mode)
	}

	return nil
}

// MarshalJSON implements json.Marshaler interface
func (c *Config) MarshalJSON() ([]byte, error) {
	type Alias Config
	return json.Marshal((*Alias)(c))
}

// UnmarshalJSON implements json.Unmarshaler interface
func (c *Config) UnmarshalJSON(data []byte) error {
	type Alias Config
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(c),
	}
	return json.Unmarshal(data, aux)
}

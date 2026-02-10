package tray

import (
	"context"

	"mcpproxy-go/internal/config"
	"mcpproxy-go/internal/events"
	"mcpproxy-go/internal/upstream"
)

// ServerInterface defines the interface for server control
// This interface is used across all build configurations (GUI and headless)
type ServerInterface interface {
	IsRunning() bool
	GetListenAddress() string
	GetUpstreamStats() map[string]interface{}
	StartServer(ctx context.Context) error
	StopServer() error
	GetStatus() interface{}            // Returns server status for display
	StatusChannel() <-chan interface{} // Channel for status updates

	// Quarantine management methods
	GetQuarantinedServers() ([]map[string]interface{}, error)
	UnquarantineServer(serverName string) error

	// Upstream manager access for diagnostics
	GetManager() *upstream.Manager

	// Server management methods for tray menu
	EnableServer(serverName string, enabled bool) error
	QuarantineServer(serverName string, quarantined bool) error
	DeleteServer(serverName string, deleteFromConfig bool) error // Delete server from runtime and optionally from config
	StopUpstreamServer(serverName string) error                  // Stop individual upstream server (sets Stopped=true)
	UnstopUpstreamServer(serverName string) error                // Unstop individual upstream server (sets Stopped=false)
	GetAllServers() ([]map[string]interface{}, error)
	GetServerTools(serverName string) ([]map[string]interface{}, error)

	// Config management for file watching
	ReloadConfiguration() error
	ShouldSkipConfigReload() bool // Check if config change was programmatic (skip reload)
	GetConfigPath() string
	GetLogDir() string
	GetGitHubURL() string
	GetLLMConfig() *config.LLMConfig

	// OAuth control
	TriggerOAuthLogin(serverName string) error

	// Startup script control
	StartStartupScript(ctx context.Context) error
	StopStartupScript() error
	RestartStartupScript(ctx context.Context) error
	GetStartupScriptStatus() map[string]interface{}

	// Event bus for event-driven synchronization
	GetEventBus() *events.EventBus
}

// Package config provides configuration types and utilities for mcpproxy.
// MED-002: Centralized timeout constants to eliminate magic numbers.
package config

import "time"

// Shutdown & Cleanup Timeouts
const (
	// TrayQuitTimeout is the maximum time to wait for graceful shutdown
	// before forcing termination. Accounts for Docker (5s) + Process Groups (2s).
	TrayQuitTimeout = 10 * time.Second

	// TrayKillTimeout is the absolute maximum before force kill
	TrayKillTimeout = 12 * time.Second

	// DockerCleanupTimeout is the time allowed for Docker containers to stop
	DockerCleanupTimeout = 5 * time.Second

	// ProcessGroupWaitTimeout is the time to wait for process groups to terminate
	ProcessGroupWaitTimeout = 2 * time.Second

	// ProcessMonitorShutdownTimeout is the time to wait for process monitor to stop
	// HIGH-005: Increased to allow Docker containers to stop gracefully
	ProcessMonitorShutdownTimeout = 7 * time.Second

	// ServerDisconnectTimeout is the max time to wait for a server to disconnect
	ServerDisconnectTimeout = 10 * time.Second
)

// Connection Timeouts
const (
	// DefaultConnectionTimeout is the default timeout for establishing connections
	// Increased from 30s to 60s to allow NPX-based servers more time to download packages
	DefaultConnectionTimeout = 60 * time.Second

	// HTTPConnectionTimeout is the timeout for HTTP/SSE connections
	HTTPConnectionTimeout = 180 * time.Second

	// HTTPIdleConnTimeout is the idle connection timeout for HTTP transports
	HTTPIdleConnTimeout = 90 * time.Second

	// QuickOperationTimeout is used for quick health checks and status queries
	QuickOperationTimeout = 10 * time.Second
)

// Retry & Backoff Configuration
const (
	// MaxBackoffDelay is the maximum delay between retry attempts
	MaxBackoffDelay = 30 * time.Second

	// InitialBackoffDelay is the starting delay for exponential backoff
	InitialBackoffDelay = 1 * time.Second

	// TokenReconnectCooldown prevents rapid reconnection attempts
	TokenReconnectCooldown = 10 * time.Second
)

// Health Check & Monitoring Intervals
const (
	// HealthCheckInterval is how often to check client health
	HealthCheckInterval = 5 * time.Second

	// AutoRecoveryCheckInterval is how often to check for auto-recovery
	AutoRecoveryCheckInterval = 60 * time.Second

	// ListToolsTimeout is the timeout for listing tools from a server
	ListToolsTimeout = 30 * time.Second

	// ToolCallTimeout is the timeout for individual tool calls via API
	// This prevents hanging API requests when upstream servers don't respond
	ToolCallTimeout = 60 * time.Second
)

// OAuth Timeouts
const (
	// OAuthReadHeaderTimeout prevents Slowloris attacks
	OAuthReadHeaderTimeout = 10 * time.Second

	// OAuthReadTimeout is the extended timeout for OAuth discovery
	OAuthReadTimeout = 30 * time.Second

	// OAuthWriteTimeout is the extended timeout for OAuth responses
	OAuthWriteTimeout = 30 * time.Second

	// OAuthServerShutdownTimeout is the graceful shutdown timeout for OAuth server
	OAuthServerShutdownTimeout = 5 * time.Second

	// OAuthCompletionWindow is how long after completion to consider OAuth recent
	OAuthCompletionWindow = 5 * time.Minute
)

// Restart & Recovery
const (
	// RestartWaitDelay is the minimum wait between stop and start during restart
	RestartWaitDelay = 500 * time.Millisecond

	// StateRestorationDelay is the delay before restoring server states after start
	StateRestorationDelay = 3 * time.Second
)

// Long-running Operations
const (
	// BatchOperationTimeout is the timeout for batch/parallel operations
	BatchOperationTimeout = 2 * time.Minute

	// LongRunningOperationTimeout is for operations that may take a long time
	LongRunningOperationTimeout = 30 * time.Minute
)

// Retry Configuration
const (
	// MaxConnectionRetries is the maximum number of retry attempts for failed connections
	MaxConnectionRetries = 5

	// MaxBackoffMinutes is the maximum backoff for general retries
	MaxBackoffMinutes = 5 * time.Minute

	// StartupGracePeriod is the time after first connection attempt during which
	// auto-disable is suppressed. This allows slow-starting servers (NPX, Docker)
	// to initialize without being prematurely disabled.
	// FIX: Prevents false-positive auto-disable for servers that take time to start
	StartupGracePeriod = 2 * time.Minute

	// NPXServerStartupTimeout is the extended timeout for NPX-based servers
	// which need to download packages on first run
	NPXServerStartupTimeout = 90 * time.Second
)

// OAuth Backoff Intervals (much longer than standard retries)
const (
	// OAuthBackoffLevel1 is the initial backoff after first OAuth failure
	OAuthBackoffLevel1 = 5 * time.Minute

	// OAuthBackoffLevel2 is the backoff after second OAuth failure
	OAuthBackoffLevel2 = 15 * time.Minute

	// OAuthBackoffLevel3 is the backoff after third OAuth failure
	OAuthBackoffLevel3 = 1 * time.Hour

	// OAuthBackoffLevel4 is the backoff after fourth OAuth failure
	OAuthBackoffLevel4 = 4 * time.Hour

	// OAuthBackoffMax is the maximum backoff for OAuth retries
	OAuthBackoffMax = 24 * time.Hour
)

// Event Bus Buffer Sizes
const (
	// EventChannelBufferSize is the buffer size for individual event subscriptions
	EventChannelBufferSize = 100

	// EventChannelBufferSizeAll is the buffer size for subscribing to all events
	EventChannelBufferSizeAll = 500
)

// HTTP Transport Configuration
const (
	// MaxIdleConns is the maximum number of idle HTTP connections
	MaxIdleConns = 10

	// MaxIdleConnsPerHost is the maximum idle connections per host
	MaxIdleConnsPerHost = 5
)

// Application State Machine
const (
	// StableStateTimeout is the default timeout for stable state transitions
	StableStateTimeout = 30 * time.Second

	// StateTransitionDelay is the delay during state transitions
	StateTransitionDelay = 2 * time.Second
)

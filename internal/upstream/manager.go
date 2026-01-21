package upstream

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	"mcpproxy-go/internal/config"
	"mcpproxy-go/internal/events"
	"mcpproxy-go/internal/logs"
	"mcpproxy-go/internal/oauth"
	"mcpproxy-go/internal/storage"
	"mcpproxy-go/internal/transport"
	"mcpproxy-go/internal/upstream/core"
	"mcpproxy-go/internal/upstream/managed"
	"mcpproxy-go/internal/upstream/types"
)

// Manager manages connections to multiple upstream MCP servers
type Manager struct {
	clients         map[string]*managed.Client
	mu              sync.RWMutex
	logger          *zap.Logger
	logConfig       *config.LogConfig
	globalConfig    *config.Config
	storage         *storage.BoltDB
	storageManager  *storage.Manager // For persisting state changes
	notificationMgr *NotificationManager
	eventBus        *events.EventBus // Event bus for publishing state changes

	// tokenReconnect keeps last reconnect trigger time per server when detecting
	// newly available OAuth tokens without explicit DB events (e.g., when CLI
	// cannot write due to DB lock). Prevents rapid retrigger loops.
	tokenReconnect map[string]time.Time

	// onServerAutoDisable callback to notify server when a server is auto-disabled
	onServerAutoDisable func(serverName string, reason string)
}

// NewManager creates a new upstream manager
func NewManager(logger *zap.Logger, globalConfig *config.Config, storage *storage.BoltDB) *Manager {
	manager := &Manager{
		clients:         make(map[string]*managed.Client),
		logger:          logger,
		globalConfig:    globalConfig,
		storage:         storage,
		notificationMgr: NewNotificationManager(),
		tokenReconnect:  make(map[string]time.Time),
	}

	// Set up OAuth completion callback to trigger connection retries (in-process)
	tokenManager := oauth.GetTokenStoreManager()
	tokenManager.SetOAuthCompletionCallback(func(serverName string) {
		logger.Info("OAuth completion callback triggered, attempting connection retry",
			zap.String("server", serverName))
		if err := manager.RetryConnection(serverName); err != nil {
			logger.Warn("Failed to trigger connection retry after OAuth completion",
				zap.String("server", serverName),
				zap.Error(err))
		}
	})

	// Start database event monitor for cross-process OAuth completion notifications
	if storage != nil {
		go manager.startOAuthEventMonitor()
	}

	// Start health check monitor for servers with health_check enabled
	go manager.startHealthCheckMonitor()

	return manager
}

// SetLogConfig sets the logging configuration for upstream server loggers
func (m *Manager) SetLogConfig(logConfig *config.LogConfig) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logConfig = logConfig
}

// AddNotificationHandler adds a notification handler to receive state change notifications
func (m *Manager) AddNotificationHandler(handler NotificationHandler) {
	m.notificationMgr.AddHandler(handler)
}

// SetEventBus sets the event bus for publishing state change events
func (m *Manager) SetEventBus(eventBus *events.EventBus) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.eventBus = eventBus
	m.logger.Info("Event bus configured for upstream manager")
}

// SetServerAutoDisableCallback sets the callback to be invoked when a server is auto-disabled
func (m *Manager) SetServerAutoDisableCallback(callback func(serverName string, reason string)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onServerAutoDisable = callback
}

// SetStorageManager sets the storage manager for persisting state changes
func (m *Manager) SetStorageManager(storageManager *storage.Manager) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.storageManager = storageManager
	m.logger.Info("Storage manager configured for upstream manager")
}

// AddServerConfig adds a server configuration without connecting
func (m *Manager) AddServerConfig(id string, serverConfig *config.ServerConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if existing client exists and if config has changed
	if existingClient, exists := m.clients[id]; exists {
		existingConfig := existingClient.Config

		// Compare configurations to determine if reconnection is needed
		configChanged := existingConfig.URL != serverConfig.URL ||
			existingConfig.Protocol != serverConfig.Protocol ||
			existingConfig.Command != serverConfig.Command ||
			!equalStringSlices(existingConfig.Args, serverConfig.Args) ||
			!equalStringMaps(existingConfig.Env, serverConfig.Env) ||
			!equalStringMaps(existingConfig.Headers, serverConfig.Headers) ||
			existingConfig.StartupMode != serverConfig.StartupMode

		if configChanged {
			m.logger.Info("Server configuration changed, disconnecting existing client",
				zap.String("id", id),
				zap.String("name", serverConfig.Name),
				zap.String("current_state", existingClient.GetState().String()),
				zap.Bool("is_connected", existingClient.IsConnected()))
			_ = existingClient.Disconnect()
			delete(m.clients, id)
		} else {
			m.logger.Debug("Server configuration unchanged, keeping existing client",
				zap.String("id", id),
				zap.String("name", serverConfig.Name),
				zap.String("current_state", existingClient.GetState().String()),
				zap.Bool("is_connected", existingClient.IsConnected()))
			// Update the client's config reference to the new config but don't recreate the client
			existingClient.Config = serverConfig

			// Restore auto-disabled state from updated config (fix for config reload bug)
			if serverConfig.StartupMode == "auto_disabled" {
				existingClient.StateManager.SetAutoDisabled("Restored from config")
				m.logger.Info("Restored auto-disabled state during config update",
					zap.String("server", serverConfig.Name),
					zap.String("startup_mode", serverConfig.StartupMode))
			}

			return nil
		}
	}

	// Create new client but don't connect yet
	client, err := managed.NewClient(id, serverConfig, m.logger, m.logConfig, m.globalConfig, m.storage)
	if err != nil {
		return fmt.Errorf("failed to create client for server %s: %w", serverConfig.Name, err)
	}

	// Configure auto-disable threshold (per-server or global default)
	threshold := serverConfig.AutoDisableThreshold
	if threshold == 0 {
		// Use global default if per-server not specified
		threshold = m.globalConfig.AutoDisableThreshold
		if threshold == 0 {
			// Final fallback to consolidated constant (HIGH-003)
			threshold = types.DefaultAutoDisableThreshold
		}
	}
	client.StateManager.SetAutoDisableThreshold(threshold)
	m.logger.Debug("Configured auto-disable threshold",
		zap.String("server", serverConfig.Name),
		zap.Int("threshold", threshold))

	// Restore auto-disable state from config (if server was previously auto-disabled)
	if serverConfig.StartupMode == "auto_disabled" {
		client.StateManager.SetAutoDisabled("Restored from config")
		m.logger.Info("Restored auto-disabled state from config",
			zap.String("server", serverConfig.Name),
			zap.String("startup_mode", serverConfig.StartupMode))
	}

	// Set up notification callback for state changes
	if m.notificationMgr != nil {
		notifierCallback := StateChangeNotifier(m.notificationMgr, serverConfig.Name)
		// Combine with existing callback if present
		existingCallback := client.StateManager.GetStateChangeCallback()
		client.StateManager.SetStateChangeCallback(func(oldState, newState types.ConnectionState, info *types.ConnectionInfo) {
			// Call existing callback first (for logging)
			if existingCallback != nil {
				existingCallback(oldState, newState, info)
			}
			// Then call notification callback
			notifierCallback(oldState, newState, info)

			// Publish event to event bus if available
			m.mu.RLock()
			eventBus := m.eventBus
			m.mu.RUnlock()

			if eventBus != nil {
				eventBus.Publish(events.Event{
					Type:       events.EventStateChange,
					ServerName: serverConfig.Name,
					Data: events.StateChangeData{
						OldState: oldState,
						NewState: newState,
						Info:     info,
					},
					Timestamp: time.Now(),
				})
			}
		})
	}

	// Set up auto-disable callback to persist config changes
	if m.onServerAutoDisable != nil {
		client.SetAutoDisableCallback(func(serverName string, reason string) {
			// Call the manager's callback which will trigger server config save
			m.onServerAutoDisable(serverName, reason)
		})
	}

	// Set storage manager for persisting state changes
	if m.storageManager != nil {
		client.SetStorageManager(m.storageManager)
		m.logger.Debug("Configured storage manager for client",
			zap.String("server", serverConfig.Name))
	}

	m.clients[id] = client
	m.logger.Info("Added upstream server configuration",
		zap.String("id", id),
		zap.String("name", serverConfig.Name))

	return nil
}

// Helper functions for comparing slices and maps
func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func equalStringMaps(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

// AddServer adds a new upstream server and connects to it (legacy method)
func (m *Manager) AddServer(id string, serverConfig *config.ServerConfig) error {
	if err := m.AddServerConfig(id, serverConfig); err != nil {
		return err
	}

	// Check startup mode and skip connection if not active
	if serverConfig.StartupMode == "disabled" {
		m.logger.Debug("Skipping connection for disabled server",
			zap.String("id", id),
			zap.String("name", serverConfig.Name))
		return nil
	}

	if serverConfig.StartupMode == "quarantined" {
		m.logger.Debug("Skipping connection for quarantined server",
			zap.String("id", id),
			zap.String("name", serverConfig.Name))
		return nil
	}

	if serverConfig.StartupMode == "auto_disabled" {
		m.logger.Debug("Skipping connection for auto-disabled server",
			zap.String("id", id),
			zap.String("name", serverConfig.Name))
		return nil
	}

	// Check if client exists and is already connected
	if client, exists := m.GetClient(id); exists {
		// Check runtime-only userStopped flag (NEVER persisted)
		// If user manually stopped this server via tray, skip connection
		if client.StateManager.IsUserStopped() {
			m.logger.Debug("Skipping connection for user-stopped server (runtime-only state)",
				zap.String("id", id),
				zap.String("name", serverConfig.Name))
			return nil
		}

		if client.IsConnected() {
			m.logger.Debug("Server is already connected, skipping connection attempt",
				zap.String("id", id),
				zap.String("name", serverConfig.Name))
			return nil
		}

		// MED-002: Connect to server with per-server or default timeout
		effectiveTimeout := serverConfig.GetConnectionTimeout()
		ctx, cancel := context.WithTimeout(context.Background(), effectiveTimeout)
		defer cancel()
		if err := client.Connect(ctx); err != nil {
			return fmt.Errorf("failed to connect to server %s: %w", serverConfig.Name, err)
		}
	} else {
		m.logger.Error("Client not found after AddServerConfig - this should not happen",
			zap.String("id", id),
			zap.String("name", serverConfig.Name))
	}

	return nil
}

// RemoveServer removes an upstream server
func (m *Manager) RemoveServer(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if client, exists := m.clients[id]; exists {
		m.logger.Info("Removing upstream server",
			zap.String("id", id),
			zap.String("state", client.GetState().String()))
		// Remove from map immediately to prevent further operations
		delete(m.clients, id)
		// Disconnect asynchronously to avoid blocking if connection is in progress
		// This prevents 30s delay when removing a connecting server
		go func(c *managed.Client) {
			_ = c.Disconnect()
		}(client)
	}
}

// GetClient returns a client by ID
func (m *Manager) GetClient(id string) (*managed.Client, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	client, exists := m.clients[id]
	return client, exists
}

// GetAllClients returns all clients
func (m *Manager) GetAllClients() map[string]*managed.Client {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]*managed.Client)
	for id, client := range m.clients {
		result[id] = client
	}
	return result
}

// GetAllServerNames returns a slice of all configured server names
func (m *Manager) GetAllServerNames() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, 0, len(m.clients))
	for name := range m.clients {
		names = append(names, name)
	}
	return names
}

// DiscoverTools discovers all tools from all connected upstream servers
func (m *Manager) DiscoverTools(ctx context.Context) ([]*config.ToolMetadata, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var allTools []*config.ToolMetadata
	connectedCount := 0

	for id, client := range m.clients {
		if client.Config.IsDisabled() {
			continue
		}
		if !client.IsConnected() {
			m.logger.Debug("Skipping disconnected client", zap.String("id", id), zap.String("state", client.GetState().String()))
			continue
		}
		connectedCount++

		tools, err := client.ListTools(ctx)
		if err != nil {
			m.logger.Error("Failed to list tools from client",
				zap.String("id", id),
				zap.Error(err))
			continue
		}

		if tools != nil {
			allTools = append(allTools, tools...)
		}
	}

	m.logger.Info("Discovered tools from upstream servers",
		zap.Int("total_tools", len(allTools)),
		zap.Int("connected_servers", connectedCount))

	return allTools, nil
}

// CallTool calls a tool on the appropriate upstream server
func (m *Manager) CallTool(ctx context.Context, toolName string, args map[string]interface{}) (interface{}, error) {
	// Parse tool name to extract server and tool components
	parts := strings.SplitN(toolName, ":", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid tool name format: %s (expected server:tool)", toolName)
	}

	serverName := parts[0]
	actualToolName := parts[1]

	m.mu.RLock()
	defer m.mu.RUnlock()

	// Find the client for this server
	var targetClient *managed.Client
	for _, client := range m.clients {
		if client.Config.Name == serverName {
			targetClient = client
			break
		}
	}

	if targetClient == nil {
		return nil, fmt.Errorf("no client found for server: %s", serverName)
	}

	if targetClient.Config.IsDisabled() {
		return nil, fmt.Errorf("client for server %s is disabled (startup_mode: %s)", serverName, targetClient.Config.StartupMode)
	}

	// Check connection status and provide detailed error information
	if !targetClient.IsConnected() {
		state := targetClient.GetState()
		if targetClient.IsConnecting() {
			return nil, fmt.Errorf("server '%s' is currently connecting - please wait for connection to complete (state: %s)", serverName, state.String())
		}

		// Lazy loading: Try to connect the server if it has tools in DB
		if m.globalConfig.EnableLazyLoading && targetClient.Config.ToolCount > 0 {
			m.logger.Info("Lazy loading: Connecting to server on-demand",
				zap.String("server", serverName),
				zap.String("tool", actualToolName),
				zap.Int("tool_count", targetClient.Config.ToolCount))

			// Release read lock temporarily to allow Connect() to acquire write lock if needed
			m.mu.RUnlock()

			// Attempt to connect
			err := targetClient.Connect(ctx)

			// Reacquire read lock
			m.mu.RLock()

			if err != nil {
				m.logger.Error("Lazy loading: Failed to connect server on-demand",
					zap.String("server", serverName),
					zap.Error(err))
				return nil, fmt.Errorf("lazy loading failed for server '%s': connection attempt failed: %w", serverName, err)
			}

			// Wait a moment for the connection to stabilize
			if !targetClient.IsConnected() {
				return nil, fmt.Errorf("lazy loading failed for server '%s': server did not become connected after connection attempt", serverName)
			}

			m.logger.Info("Lazy loading: Server connected successfully",
				zap.String("server", serverName))

			// Continue to tool execution below
		} else {
			// Not lazy loading or no tools in DB - return error as before
			// Include last error if available with enhanced context
			if lastError := targetClient.GetLastError(); lastError != nil {
				// Enrich OAuth-related errors at source
				lastErrStr := lastError.Error()
				if strings.Contains(lastErrStr, "OAuth authentication failed") ||
					strings.Contains(lastErrStr, "Dynamic Client Registration") ||
					strings.Contains(lastErrStr, "authorization required") {
					return nil, fmt.Errorf("server '%s' requires OAuth authentication but is not properly configured. OAuth setup failed: %s. Please configure OAuth credentials manually or use a Personal Access Token - check mcpproxy logs for detailed setup instructions", serverName, lastError.Error())
				}

				if strings.Contains(lastErrStr, "OAuth metadata unavailable") {
					return nil, fmt.Errorf("server '%s' does not provide valid OAuth configuration endpoints. This server may not support OAuth or requires manual authentication setup: %s", serverName, lastError.Error())
				}

				return nil, fmt.Errorf("server '%s' is not connected (state: %s) - connection failed with error: %s", serverName, state.String(), lastError.Error())
			}

			return nil, fmt.Errorf("server '%s' is not connected (state: %s) - use 'upstream_servers' tool to check server configuration", serverName, state.String())
		}
	}

	// Call the tool on the upstream server with enhanced error handling
	result, err := targetClient.CallTool(ctx, actualToolName, args)
	if err != nil {
		// Enrich errors at source with server context
		errStr := err.Error()

		// OAuth-related errors
		if strings.Contains(errStr, "OAuth authentication failed") ||
			strings.Contains(errStr, "authorization required") ||
			strings.Contains(errStr, "invalid_token") ||
			strings.Contains(errStr, "Unauthorized") {
			return nil, fmt.Errorf("server '%s' authentication failed for tool '%s'. OAuth/token authentication required but not properly configured. Check server authentication settings and ensure valid credentials are available: %w", serverName, actualToolName, err)
		}

		// Permission/scope errors
		if strings.Contains(errStr, "insufficient_scope") || strings.Contains(errStr, "access_denied") {
			return nil, fmt.Errorf("server '%s' denied access to tool '%s' due to insufficient permissions or scopes. Check OAuth scopes configuration or token permissions: %w", serverName, actualToolName, err)
		}

		// Rate limiting
		if strings.Contains(errStr, "429") || strings.Contains(errStr, "rate limit") || strings.Contains(errStr, "too many requests") {
			return nil, fmt.Errorf("server '%s' rate limit exceeded for tool '%s'. Please wait before making more requests or check API quotas: %w", serverName, actualToolName, err)
		}

		// Connection issues
		if strings.Contains(errStr, "connection refused") || strings.Contains(errStr, "no such host") {
			return nil, fmt.Errorf("server '%s' connection failed for tool '%s'. Check if the server URL is correct and the server is running: %w", serverName, actualToolName, err)
		}

		// Tool-specific errors
		if strings.Contains(errStr, "tool not found") || strings.Contains(errStr, "unknown tool") {
			return nil, fmt.Errorf("tool '%s' not found on server '%s'. Use 'retrieve_tools' to see available tools: %w", actualToolName, serverName, err)
		}

		// Generic error with helpful context
		return nil, fmt.Errorf("tool '%s' on server '%s' failed: %w. Check server configuration, authentication, and tool parameters", actualToolName, serverName, err)
	}

	return result, nil
}

// clientJob represents a client connection job for parallel processing
type clientJob struct {
	id     string
	client *managed.Client
}

// ConnectAll connects to all configured servers using two-phase strategy:
// Phase 1: Initial connection attempts for all servers
// Phase 2: Retry failed servers (up to 5 retries max)
func (m *Manager) ConnectAll(ctx context.Context) error {
	m.mu.RLock()
	clients := make(map[string]*managed.Client)
	for id, client := range m.clients {
		clients[id] = client
	}
	m.mu.RUnlock()

	// Collect clients that need to connect

	var jobs []clientJob

	for id, client := range clients {
		m.logger.Debug("Evaluating client for connection",
			zap.String("id", id),
			zap.String("name", client.Config.Name),
			zap.String("startup_mode", client.Config.StartupMode),
			zap.Bool("should_connect", client.Config.ShouldConnectOnStartup()),
			zap.Bool("is_connected", client.IsConnected()),
			zap.Bool("is_connecting", client.IsConnecting()),
			zap.String("current_state", client.GetState().String()))

		if !client.Config.ShouldConnectOnStartup() {
			m.logger.Debug("Skipping client (startup_mode prevents connection)",
				zap.String("id", id),
				zap.String("name", client.Config.Name),
				zap.String("startup_mode", client.Config.StartupMode))

			if client.IsConnected() {
				m.logger.Info("Disconnecting client that should not connect on startup",
					zap.String("id", id),
					zap.String("name", client.Config.Name),
					zap.String("startup_mode", client.Config.StartupMode))
				_ = client.Disconnect()
			}
			continue
		}

		// Check runtime-only userStopped flag (NOT persisted)
		// If user manually stopped this server via tray, skip connection
		if client.StateManager.IsUserStopped() {
			m.logger.Debug("Skipping user-stopped server (runtime-only state)",
				zap.String("id", id),
				zap.String("name", client.Config.Name))

			if client.IsConnected() {
				m.logger.Info("Disconnecting user-stopped server",
					zap.String("id", id),
					zap.String("name", client.Config.Name))
				_ = client.Disconnect()
			}
			continue
		}

		// Lazy loading optimization: Skip connection for servers with cached tools
		// These servers will connect on-demand when a tool call is made
		// ConnectionState remains Disconnected until first tool call
		if m.globalConfig.EnableLazyLoading && client.Config.ToolCount > 0 && client.Config.StartupMode == "lazy_loading" && client.Config.EverConnected {
			m.logger.Debug("Skipping connection for lazy loading server with cached tools",
				zap.String("id", id),
				zap.String("name", client.Config.Name),
				zap.Int("tool_count", client.Config.ToolCount),
				zap.String("startup_mode", client.Config.StartupMode))
			continue
		}

		// Check connection eligibility
		if client.IsConnected() {
			m.logger.Debug("Client already connected, skipping",
				zap.String("id", id),
				zap.String("name", client.Config.Name))
			continue
		}

		if client.IsConnecting() {
			m.logger.Debug("Client already connecting, skipping",
				zap.String("id", id),
				zap.String("name", client.Config.Name))
			continue
		}

		jobs = append(jobs, clientJob{
			id:     id,
			client: client,
		})
	}

	if len(jobs) == 0 {
		m.logger.Debug("No clients need connection")
		return nil
	}

	// Get concurrency limit from config
	maxConcurrent := m.globalConfig.MaxConcurrentConnections
	if maxConcurrent <= 0 {
		maxConcurrent = 10 // Fallback default
	}

	// Build map of eligible clients for scheduler
	eligibleClients := make(map[string]*managed.Client)
	for _, job := range jobs {
		eligibleClients[job.id] = job.client
	}

	m.logger.Info("🚀 Starting connection scheduler",
		zap.Int("total_clients", len(eligibleClients)),
		zap.Int("worker_count", maxConcurrent))

	// Use queue-based scheduler for startup connections
	// Benefits: constant 10 active workers, individual timeouts, retry queue
	scheduler := NewConnectionScheduler(m, maxConcurrent, m.logger)
	result := scheduler.Start(eligibleClients)

	m.logger.Info("✅ ConnectAll completed",
		zap.Int("total_attempted", result.TotalJobs),
		zap.Int("successful", result.Successful),
		zap.Int("failed", result.Failed),
		zap.Int("retried", result.Retried),
		zap.Duration("duration", result.Duration))

	return nil
}

// connectPhase performs a single phase of connection attempts
func (m *Manager) connectPhase(ctx context.Context, jobs []clientJob, maxConcurrent int, timeout time.Duration, phase string) []clientJob {
	semaphore := make(chan struct{}, maxConcurrent)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var failedJobs []clientJob

	for _, job := range jobs {
		wg.Add(1)
		semaphore <- struct{}{} // Acquire

		go func(j clientJob) {
			defer wg.Done()
			defer func() { <-semaphore }() // Release

			// Use per-server timeout if configured, otherwise use the phase default
			effectiveTimeout := j.client.Config.GetConnectionTimeout()

			// Create timeout context for this connection attempt
			connCtx, cancel := context.WithTimeout(ctx, effectiveTimeout)
			defer cancel()

			m.logger.Info("Connecting server",
				zap.String("phase", phase),
				zap.String("id", j.id),
				zap.String("name", j.client.Config.Name),
				zap.Duration("timeout", effectiveTimeout))

			if err := j.client.Connect(connCtx); err != nil {
				m.logger.Warn("Connection failed",
					zap.String("phase", phase),
					zap.String("id", j.id),
					zap.String("name", j.client.Config.Name),
					zap.Error(err))

				// Add to failed jobs for retry
				mu.Lock()
				failedJobs = append(failedJobs, j)
				mu.Unlock()
			} else {
				m.logger.Info("✅ Connection successful",
					zap.String("phase", phase),
					zap.String("id", j.id),
					zap.String("name", j.client.Config.Name))
			}
		}(job)
	}

	wg.Wait()
	return failedJobs
}

// retryFailedServers performs up to MaxConnectionRetries for failed servers
func (m *Manager) retryFailedServers(ctx context.Context, failedJobs []clientJob, maxConcurrent int) {
	maxRetries := config.MaxConnectionRetries

	for retry := 1; retry <= maxRetries; retry++ {
		if len(failedJobs) == 0 {
			m.logger.Info("No more failed servers to retry")
			break
		}

		m.logger.Info("Retry attempt",
			zap.Int("retry", retry),
			zap.Int("max_retries", maxRetries),
			zap.Int("servers_to_retry", len(failedJobs)))

		// MED-002: Exponential backoff delay before retry with centralized max
		backoffDelay := time.Duration(1<<uint(retry-1)) * time.Second
		if backoffDelay > config.MaxBackoffDelay {
			backoffDelay = config.MaxBackoffDelay
		}

		m.logger.Info("Waiting before retry",
			zap.Duration("backoff", backoffDelay),
			zap.Int("retry", retry))
		time.Sleep(backoffDelay)

		// MED-002: Retry failed servers with centralized timeout
		failedJobs = m.connectPhase(ctx, failedJobs, maxConcurrent, config.DefaultConnectionTimeout, fmt.Sprintf("retry-%d", retry))

		// Check if we're on the last retry
		if retry == maxRetries && len(failedJobs) > 0 {
			m.logger.Warn("⚠️ Max retries reached, disabling/quarantining persistent failures",
				zap.Int("failed_servers", len(failedJobs)))

			for _, job := range failedJobs {
				m.handlePersistentFailure(job.id, job.client)
			}
		}
	}
}

// handlePersistentFailure disables or quarantines a server after max retries
func (m *Manager) handlePersistentFailure(id string, client *managed.Client) {
	// Get connection info for detailed logging
	info := client.StateManager.GetConnectionInfo()
	errorMsg := "Unknown error"
	if info.LastError != nil {
		errorMsg = info.LastError.Error()
	}

	m.logger.Error("🚫 Server persistently failing after max retries",
		zap.String("id", id),
		zap.String("name", client.Config.Name),
		zap.String("final_state", client.GetState().String()),
		zap.Int("consecutive_failures", info.ConsecutiveFailures),
		zap.String("error", errorMsg))

	// Update config to auto_disabled
	reason := fmt.Sprintf("Server automatically disabled after %d startup failures", info.ConsecutiveFailures)
	client.Config.StartupMode = "auto_disabled"
	client.Config.AutoDisableReason = reason

	// Mark as auto-disabled in StateManager (for tray UI and status)
	client.StateManager.SetAutoDisabled(reason)

	// Write detailed failure information to failed_servers.log for web UI display
	dataDir := m.globalConfig.DataDir

	// Use detailed logging with error categorization
	if err := logs.LogServerFailureDetailed(
		dataDir,
		client.Config.Name,
		errorMsg,
		info.ConsecutiveFailures,
		info.FirstAttemptTime,
	); err != nil {
		m.logger.Error("Failed to write detailed failure to failed_servers.log",
			zap.String("server", client.Config.Name),
			zap.Error(err))
		// Fallback to simple logging
		if fallbackErr := logs.LogServerFailure(dataDir, client.Config.Name, reason); fallbackErr != nil {
			m.logger.Error("Failed to write simple failure log as fallback",
				zap.String("server", client.Config.Name),
				zap.Error(fallbackErr))
		}
	}

	// Persist to storage (storage layer still uses legacy fields for backward compatibility)
	if m.storage != nil {
		if err := m.storage.SaveUpstream(&storage.UpstreamRecord{
			ID:                       client.Config.Name,
			Name:                     client.Config.Name,
			URL:                      client.Config.URL,
			Protocol:                 client.Config.Protocol,
			Command:                  client.Config.Command,
			Args:                     client.Config.Args,
			WorkingDir:               client.Config.WorkingDir,
			Env:                      client.Config.Env,
			Headers:                  client.Config.Headers,
			OAuth:                    client.Config.OAuth,
			RepositoryURL:            client.Config.RepositoryURL,
			ServerState:              "auto_disabled",
			Created:                  client.Config.Created,
			Updated:                  time.Now(),
			Isolation:                client.Config.Isolation,
			GroupID:                  client.Config.GroupID,
			GroupName:                client.Config.GroupName,
			EverConnected:            client.Config.EverConnected,
			LastSuccessfulConnection: client.Config.LastSuccessfulConnection,
			ToolCount:                client.Config.ToolCount,
			AutoDisableThreshold:     client.Config.AutoDisableThreshold,
		}); err != nil {
			m.logger.Error("Failed to persist disabled server state",
				zap.String("server", client.Config.Name),
				zap.Error(err))
		}
	}

	// Disconnect client
	_ = client.Disconnect()

	// Trigger auto-disable callback to persist config changes
	if m.onServerAutoDisable != nil {
		m.onServerAutoDisable(client.Config.Name, reason)
	}

	m.logger.Warn("Server has been disabled due to persistent connection failures. Enable manually after fixing the issue.",
		zap.String("name", client.Config.Name),
		zap.String("reason", reason))
}

// DisconnectAll disconnects from all servers in parallel to avoid blocking
// CRIT-005: Now aggregates all errors instead of returning only the last one
// TIMEOUT-FIX: Execute disconnects in parallel with timeout to prevent API hangs
func (m *Manager) DisconnectAll() error {
	m.mu.RLock()
	clients := make([]*managed.Client, 0, len(m.clients))
	for _, client := range m.clients {
		clients = append(clients, client)
	}
	m.mu.RUnlock()

	if len(clients) == 0 {
		return nil
	}

	// TIMEOUT-FIX: Disconnect all clients in parallel with timeout
	type disconnectResult struct {
		serverName string
		err        error
	}

	results := make(chan disconnectResult, len(clients))
	timeout := 10 * time.Second // Maximum wait time for all disconnects

	for _, client := range clients {
		go func(c *managed.Client) {
			err := c.Disconnect()
			results <- disconnectResult{serverName: c.Config.Name, err: err}
		}(client)
	}

	// Collect results with timeout
	var errs []error
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	for i := 0; i < len(clients); i++ {
		select {
		case result := <-results:
			if result.err != nil {
				errs = append(errs, fmt.Errorf("%s: %w", result.serverName, result.err))
				m.logger.Warn("Failed to disconnect client",
					zap.String("server", result.serverName),
					zap.Error(result.err))
			}
		case <-timer.C:
			m.logger.Warn("DisconnectAll timeout reached, some clients may not have disconnected cleanly",
				zap.Int("remaining", len(clients)-i),
				zap.Duration("timeout", timeout))
			errs = append(errs, fmt.Errorf("timeout waiting for %d clients to disconnect", len(clients)-i))
			goto done
		}
	}

done:
	// Return aggregated errors using errors.Join (Go 1.20+)
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

// HasDockerContainers checks if any connected servers are running Docker containers
func (m *Manager) HasDockerContainers() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, client := range m.clients {
		if client.IsDockerCommand() {
			return true
		}
	}
	return false
}

// GetStats returns statistics about upstream connections
func (m *Manager) GetStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	connectedCount := 0
	connectingCount := 0
	totalCount := len(m.clients)

	serverStatus := make(map[string]interface{})
	for id, client := range m.clients {
		// Get detailed connection info from state manager
		connectionInfo := client.GetConnectionInfo()

		status := map[string]interface{}{
			"state":        connectionInfo.State.String(),
			"connected":    connectionInfo.State == types.StateReady,
			"connecting":   client.IsConnecting(),
			"retry_count":  connectionInfo.RetryCount,
			"should_retry": client.ShouldRetry(),
			"name":         client.Config.Name,
			"url":          client.Config.URL,
			"protocol":     client.Config.Protocol,
		}

		if connectionInfo.State == types.StateReady {
			connectedCount++
		}

		if client.IsConnecting() {
			connectingCount++
		}

		if !connectionInfo.LastRetryTime.IsZero() {
			status["last_retry_time"] = connectionInfo.LastRetryTime
		}

		if connectionInfo.LastError != nil {
			status["last_error"] = connectionInfo.LastError.Error()
		}

		if connectionInfo.ServerName != "" {
			status["server_name"] = connectionInfo.ServerName
		}

		if connectionInfo.ServerVersion != "" {
			status["server_version"] = connectionInfo.ServerVersion
		}

		// Only call GetServerInfo on connected clients to avoid blocking on connecting clients
		// GetServerInfo requires c.mu.RLock which would block if Connect holds c.mu.Lock
		if connectionInfo.State == types.StateReady {
			if client.GetServerInfo() != nil {
				info := client.GetServerInfo()
				status["protocol_version"] = info.ProtocolVersion
			}
		}

		serverStatus[id] = status
	}

	return map[string]interface{}{
		"connected_servers":  connectedCount,
		"connecting_servers": connectingCount,
		"total_servers":      totalCount,
		"servers":            serverStatus,
		"total_tools":        m.GetTotalToolCount(),
	}
}

// GetTotalToolCount returns the total number of tools across all servers
// This uses cached tool counts to avoid blocking network calls during status updates
// Tool counts are updated when ListTools is called on each client
func (m *Manager) GetTotalToolCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	totalTools := 0
	for _, client := range m.clients {
		if client.Config.IsDisabled() || !client.IsConnected() {
			continue
		}

		// Use cached tool count from Config instead of making network calls
		// The tool count is updated when ListTools() succeeds on the client
		totalTools += client.Config.ToolCount
	}
	return totalTools
}

// ListServers returns information about all registered servers
func (m *Manager) ListServers() map[string]*config.ServerConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()

	servers := make(map[string]*config.ServerConfig)
	for id, client := range m.clients {
		servers[id] = client.Config
	}
	return servers
}

// RetryConnection triggers a connection retry for a specific server
// This is typically called after OAuth completion to immediately use new tokens
func (m *Manager) RetryConnection(serverName string) error {
	m.mu.RLock()
	client, exists := m.clients[serverName]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("server not found: %s", serverName)
	}

	// If the client is already connected or connecting, do not force a
	// reconnect. This prevents Ready→Disconnected flapping when duplicate
	// OAuth completion events arrive.
	if client.IsConnected() {
		m.logger.Info("Skipping retry: client already connected",
			zap.String("server", serverName),
			zap.String("state", client.GetState().String()))
		return nil
	}
	if client.IsConnecting() {
		m.logger.Info("Skipping retry: client already connecting",
			zap.String("server", serverName),
			zap.String("state", client.GetState().String()))
		return nil
	}

	// Log detailed state prior to retry and token availability in persistent store
	// This helps diagnose cases where the core client reports "already connected"
	// while the managed state is Error/Disconnected.
	state := client.GetState().String()
	isConnected := client.IsConnected()
	isConnecting := client.IsConnecting()

	// Check persistent token presence (daemon uses BBolt-backed token store)
	var hasToken bool
	var tokenExpires time.Time
	if m.storage != nil {
		ts := oauth.NewPersistentTokenStore(client.Config.Name, client.Config.URL, m.storage)
		if tok, err := ts.GetToken(); err == nil && tok != nil {
			hasToken = true
			tokenExpires = tok.ExpiresAt
		}
	}

	m.logger.Info("Triggering connection retry after OAuth completion",
		zap.String("server", serverName),
		zap.String("state", state),
		zap.Bool("is_connected", isConnected),
		zap.Bool("is_connecting", isConnecting),
		zap.Bool("has_persistent_token", hasToken),
		zap.Time("token_expires_at", tokenExpires))

	// Trigger connection attempt in background to avoid blocking
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), config.BatchOperationTimeout)
		defer cancel()

		// Important: Ensure a clean reconnect only if not already connected.
		// Managed state guards above should make this idempotent.
		if derr := client.Disconnect(); derr != nil {
			m.logger.Debug("Disconnect before retry returned",
				zap.String("server", serverName),
				zap.Error(derr))
		}

		if err := client.Connect(ctx); err != nil {
			m.logger.Warn("Connection retry after OAuth failed",
				zap.String("server", serverName),
				zap.Error(err))
		} else {
			m.logger.Info("Connection retry after OAuth succeeded",
				zap.String("server", serverName))
		}
	}()

	return nil
}

// startOAuthEventMonitor monitors the database for OAuth completion events from CLI processes
func (m *Manager) startOAuthEventMonitor() {
	m.logger.Info("Starting OAuth event monitor for cross-process notifications")

	ticker := time.NewTicker(config.HealthCheckInterval) // Check every 5 seconds
	defer ticker.Stop()

	for range ticker.C {
		if err := m.processOAuthEvents(); err != nil {
			m.logger.Warn("Failed to process OAuth events", zap.Error(err))
		}

		// Also scan for newly available tokens to handle cases where the CLI
		// could not write a DB event due to a lock. If we see a persisted
		// token for an errored OAuth server, trigger a reconnect once.
		m.scanForNewTokens()
	}
}

// processOAuthEvents checks for and processes unprocessed OAuth completion events
func (m *Manager) processOAuthEvents() error {
	if m.storage == nil {
		m.logger.Debug("processOAuthEvents: no storage available, skipping")
		return nil
	}

	m.logger.Debug("processOAuthEvents: checking for OAuth completion events...")
	events, err := m.storage.GetUnprocessedOAuthCompletionEvents()
	if err != nil {
		m.logger.Error("processOAuthEvents: failed to get events", zap.Error(err))
		return fmt.Errorf("failed to get OAuth completion events: %w", err)
	}

	if len(events) == 0 {
		m.logger.Debug("processOAuthEvents: no unprocessed events found")
		return nil
	}

	m.logger.Info("processOAuthEvents: found unprocessed OAuth completion events", zap.Int("count", len(events)))

	for _, event := range events {
		m.logger.Info("Processing OAuth completion event from database",
			zap.String("server", event.ServerName),
			zap.Time("completed_at", event.CompletedAt))

		// Skip retry if client is already connected/connecting to avoid flapping
		m.mu.RLock()
		c, exists := m.clients[event.ServerName]
		m.mu.RUnlock()
		if exists && (c.IsConnected() || c.IsConnecting()) {
			m.logger.Info("Skipping retry for OAuth event: client already connected/connecting",
				zap.String("server", event.ServerName),
				zap.String("state", c.GetState().String()))
		} else {
			// Trigger connection retry
			if err := m.RetryConnection(event.ServerName); err != nil {
				m.logger.Warn("Failed to retry connection for OAuth completion event",
					zap.String("server", event.ServerName),
					zap.Error(err))
			} else {
				m.logger.Info("Successfully triggered connection retry for OAuth completion event",
					zap.String("server", event.ServerName))
			}
		}

		// Mark event as processed
		if err := m.storage.MarkOAuthCompletionEventProcessed(event.ServerName, event.CompletedAt); err != nil {
			m.logger.Error("Failed to mark OAuth completion event as processed",
				zap.String("server", event.ServerName),
				zap.Error(err))
		}

		// Clean up old events periodically (when processing events)
		if err := m.storage.CleanupOldOAuthCompletionEvents(); err != nil {
			m.logger.Warn("Failed to cleanup old OAuth completion events", zap.Error(err))
		}
	}

	return nil
}

// scanForNewTokens checks persistent token store for each client in Error state
// and triggers a reconnect if a token is present. This complements DB-based
// events and handles DB lock scenarios.
func (m *Manager) scanForNewTokens() {
	if m.storage == nil {
		return
	}

	m.mu.RLock()
	clients := make(map[string]*managed.Client, len(m.clients))
	for id, c := range m.clients {
		clients[id] = c
	}
	m.mu.RUnlock()

	now := time.Now()
	for id, c := range clients {
		// Only consider servers that should connect on startup but aren't currently connected
		if c.Config.IsDisabled() || c.IsConnected() {
			continue
		}

		state := c.GetState()
		// Focus on Error state likely due to OAuth/authorization
		if state != types.StateError {
			continue
		}

		// Rate-limit triggers per server
		if last, ok := m.tokenReconnect[id]; ok && now.Sub(last) < config.TokenReconnectCooldown {
			continue
		}

		// Check for a persisted token
		ts := oauth.NewPersistentTokenStore(c.Config.Name, c.Config.URL, m.storage)
		tok, err := ts.GetToken()
		if err != nil || tok == nil {
			continue
		}

		m.logger.Info("Detected persisted OAuth token; triggering reconnect",
			zap.String("server", c.Config.Name),
			zap.Time("token_expires_at", tok.ExpiresAt))

		// Remember trigger time and retry connection
		m.tokenReconnect[id] = now
		_ = m.RetryConnection(c.Config.Name)
	}
}

// StartManualOAuth performs an in-process OAuth flow for the given server.
// This avoids cross-process DB locking by using the daemon's storage directly.
func (m *Manager) StartManualOAuth(serverName string, force bool) error {
	m.mu.RLock()
	client, exists := m.clients[serverName]
	m.mu.RUnlock()
	if !exists {
		return fmt.Errorf("server not found: %s", serverName)
	}

	cfg := client.Config
	m.logger.Info("Starting in-process manual OAuth",
		zap.String("server", cfg.Name),
		zap.Bool("force", force))

	// Preflight: if server does not appear to require OAuth, avoid starting
	// OAuth flow and return an informative error (tray will show it).
	// Attempt a short no-auth initialize to confirm.
	if !oauth.ShouldUseOAuth(cfg) && !force {
		m.logger.Info("OAuth not applicable based on config (no headers, protocol)", zap.String("server", cfg.Name))
		return fmt.Errorf("OAuth is not supported or not required for server '%s'", cfg.Name)
	}

	// Create a transient core client that uses the daemon's storage
	coreClient, err := core.NewClientWithOptions(cfg.Name, cfg, m.logger, m.logConfig, m.globalConfig, m.storage, false)
	if err != nil {
		return fmt.Errorf("failed to create core client for OAuth: %w", err)
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), config.LongRunningOperationTimeout)
		defer cancel()

		if force {
			coreClient.ClearOAuthState()
		}

		// Preflight no-auth check: try a quick connect without OAuth to
		// determine if authorization is actually required. If initialize
		// succeeds, inform and return early.
		if !force {
			cpy := *cfg
			cpy.Headers = cfg.Headers // preserve headers
			// Try HTTP/SSE path with no OAuth
			noAuthTransport := transport.DetermineTransportType(&cpy)
			if noAuthTransport == "http" || noAuthTransport == "streamable-http" || noAuthTransport == "sse" {
				m.logger.Info("Running preflight no-auth initialize to check OAuth requirement", zap.String("server", cfg.Name))
				testClient, err2 := core.NewClientWithOptions(cfg.Name, &cpy, m.logger, m.logConfig, m.globalConfig, m.storage, false)
				if err2 == nil {
					tctx, tcancel := context.WithTimeout(ctx, config.QuickOperationTimeout)
					_ = testClient.Connect(tctx)
					tcancel()
					if testClient.GetServerInfo() != nil {
						m.logger.Info("Preflight succeeded without OAuth; skipping OAuth flow", zap.String("server", cfg.Name))
						return
					}
				}
			}
		}

		m.logger.Info("Triggering OAuth flow (in-process)", zap.String("server", cfg.Name))
		if err := coreClient.ForceOAuthFlow(ctx); err != nil {
			m.logger.Warn("In-process OAuth flow failed",
				zap.String("server", cfg.Name),
				zap.Error(err))
			return
		}
		m.logger.Info("In-process OAuth flow completed successfully",
			zap.String("server", cfg.Name))
		// Immediately attempt reconnect with new tokens
		if err := m.RetryConnection(cfg.Name); err != nil {
			m.logger.Warn("Failed to trigger reconnect after in-process OAuth",
				zap.String("server", cfg.Name),
				zap.Error(err))
		}
	}()

	return nil
}

// startHealthCheckMonitor monitors server health and attempts reconnection for servers with health_check enabled
func (m *Manager) startHealthCheckMonitor() {
	m.logger.Info("Starting health check monitor for servers with health_check enabled")

	ticker := time.NewTicker(config.AutoRecoveryCheckInterval) // Check every 60 seconds
	defer ticker.Stop()

	for range ticker.C {
		m.performHealthChecks()
	}
}

// performHealthChecks checks the health of all servers and attempts reconnection for disconnected ones
// Servers with HealthCheck=true get active health checks, all servers get reconnection attempts
// Uses PARALLEL reconnections with a worker pool for efficiency
func (m *Manager) performHealthChecks() {
	m.mu.RLock()
	allClients := make(map[string]*managed.Client)
	for id, client := range m.clients {
		allClients[id] = client
	}
	m.mu.RUnlock()

	if len(allClients) == 0 {
		return
	}

	// Collect servers that need reconnection
	type reconnectJob struct {
		id     string
		client *managed.Client
	}
	var toReconnect []reconnectJob

	var healthCheckCount int
	for id, client := range allClients {
		if client.Config.HealthCheck {
			healthCheckCount++
		}

		// Skip if disabled or auto-disabled
		if client.Config.IsDisabled() {
			continue
		}
		if client.StateManager.IsAutoDisabled() {
			continue
		}
		// Skip if user manually stopped
		if client.StateManager.IsUserStopped() {
			continue
		}

		// Check connection status - reconnect ALL disconnected servers
		if !client.IsConnected() {
			// Check if we should retry based on backoff
			if !client.ShouldRetry() {
				m.logger.Debug("Health check: Skipping reconnection (backoff not elapsed)",
					zap.String("id", id),
					zap.String("name", client.Config.Name),
					zap.String("state", client.GetState().String()))
				continue
			}
			toReconnect = append(toReconnect, reconnectJob{id: id, client: client})
		} else if client.Config.HealthCheck {
			// Only log healthy status for servers with explicit health check
			m.logger.Debug("Health check: Server is healthy",
				zap.String("id", id),
				zap.String("name", client.Config.Name))
		}
	}

	m.logger.Debug("Performing health checks",
		zap.Int("total_servers", len(allClients)),
		zap.Int("servers_with_health_check", healthCheckCount),
		zap.Int("disconnected_servers", len(toReconnect)))

	if len(toReconnect) == 0 {
		return
	}

	// Use parallel reconnection with worker pool
	// Use MaxConcurrentConnections from globalConfig (default 20)
	maxWorkers := 20
	if m.globalConfig != nil && m.globalConfig.MaxConcurrentConnections > 0 {
		maxWorkers = m.globalConfig.MaxConcurrentConnections
	}
	if maxWorkers > len(toReconnect) {
		maxWorkers = len(toReconnect)
	}

	m.logger.Info("Health check: Starting parallel reconnection",
		zap.Int("servers_to_reconnect", len(toReconnect)),
		zap.Int("max_workers", maxWorkers))

	// Create job channel and wait group
	jobChan := make(chan reconnectJob, len(toReconnect))
	var wg sync.WaitGroup

	// Start workers
	for i := 0; i < maxWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for job := range jobChan {
				state := job.client.GetState()
				m.logger.Info("Health check: Server not connected, attempting reconnection",
					zap.Int("worker_id", workerID),
					zap.String("id", job.id),
					zap.String("name", job.client.Config.Name),
					zap.String("state", state.String()),
					zap.Bool("health_check_enabled", job.client.Config.HealthCheck))

				// Attempt to reconnect with per-server or default timeout
				effectiveTimeout := job.client.Config.GetConnectionTimeout()
				ctx, cancel := context.WithTimeout(context.Background(), effectiveTimeout)
				err := job.client.Connect(ctx)
				cancel()

				if err != nil {
					m.logger.Warn("Health check: Failed to reconnect server",
						zap.Int("worker_id", workerID),
						zap.String("id", job.id),
						zap.String("name", job.client.Config.Name),
						zap.Error(err))
				} else {
					m.logger.Info("Health check: Successfully reconnected server",
						zap.Int("worker_id", workerID),
						zap.String("id", job.id),
						zap.String("name", job.client.Config.Name))
				}
			}
		}(i)
	}

	// Send all jobs to workers
	for _, job := range toReconnect {
		jobChan <- job
	}
	close(jobChan)

	// Wait for all workers to complete
	wg.Wait()

	m.logger.Info("Health check: Parallel reconnection completed",
		zap.Int("servers_attempted", len(toReconnect)))
}

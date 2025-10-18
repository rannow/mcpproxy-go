package upstream

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	"mcpproxy-go/internal/config"
	"mcpproxy-go/internal/events"
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
	notificationMgr *NotificationManager
	eventBus        *events.EventBus // Event bus for publishing state changes

	// tokenReconnect keeps last reconnect trigger time per server when detecting
	// newly available OAuth tokens without explicit DB events (e.g., when CLI
	// cannot write due to DB lock). Prevents rapid retrigger loops.
	tokenReconnect map[string]time.Time
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
			existingConfig.Enabled != serverConfig.Enabled ||
			existingConfig.Quarantined != serverConfig.Quarantined

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
			return nil
		}
	}

	// Create new client but don't connect yet
	client, err := managed.NewClient(id, serverConfig, m.logger, m.logConfig, m.globalConfig, m.storage)
	if err != nil {
		return fmt.Errorf("failed to create client for server %s: %w", serverConfig.Name, err)
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

	if !serverConfig.Enabled {
		m.logger.Debug("Skipping connection for disabled server",
			zap.String("id", id),
			zap.String("name", serverConfig.Name))
		return nil
	}

	// Check if client exists and is already connected
	if client, exists := m.GetClient(id); exists {
		if client.IsConnected() {
			m.logger.Debug("Server is already connected, skipping connection attempt",
				zap.String("id", id),
				zap.String("name", serverConfig.Name))
			return nil
		}

		// Connect to server with timeout to prevent hanging
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
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
		_ = client.Disconnect()
		delete(m.clients, id)
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
		if !client.Config.Enabled {
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

	if !targetClient.Config.Enabled {
		return nil, fmt.Errorf("client for server %s is disabled", serverName)
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

// ConnectAll connects to all configured servers that should retry
func (m *Manager) ConnectAll(ctx context.Context) error {
	m.mu.RLock()
	clients := make(map[string]*managed.Client)
	for id, client := range m.clients {
		clients[id] = client
	}
	m.mu.RUnlock()

	// Collect clients that need to connect, prioritizing fast protocols
	type clientJob struct {
		id       string
		client   *managed.Client
		priority int // Lower number = higher priority
	}

	var jobs []clientJob

	for id, client := range clients {
		m.logger.Debug("Evaluating client for connection",
			zap.String("id", id),
			zap.String("name", client.Config.Name),
			zap.Bool("enabled", client.Config.Enabled),
			zap.Bool("is_connected", client.IsConnected()),
			zap.Bool("is_connecting", client.IsConnecting()),
			zap.String("current_state", client.GetState().String()),
			zap.Bool("quarantined", client.Config.Quarantined))

		if !client.Config.Enabled {
			m.logger.Debug("Skipping disabled client",
				zap.String("id", id),
				zap.String("name", client.Config.Name))

			if client.IsConnected() {
				m.logger.Info("Disconnecting disabled client", zap.String("id", id), zap.String("name", client.Config.Name))
				_ = client.Disconnect()
			}
			continue
		}

		// Lazy loading check: Skip servers that have tools in DB and don't have StartOnBoot flag
		if m.globalConfig.EnableLazyLoading && client.Config.ToolCount > 0 && !client.Config.StartOnBoot {
			m.logger.Debug("Skipping server due to lazy loading (tools in DB, start_on_boot=false)",
				zap.String("id", id),
				zap.String("name", client.Config.Name),
				zap.Int("tool_count", client.Config.ToolCount),
				zap.Bool("start_on_boot", client.Config.StartOnBoot))
			continue
		}

		// Check connection eligibility with detailed logging
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

		// Assign priority based on connection history:
		// Priority 1 (highest): Servers that have connected before AND have tools
		// Priority 2 (medium): Servers that have never connected before
		// Priority 3 (lowest): Servers that last had an error
		priority := 2 // Default: never connected before

		if client.Config.EverConnected && client.Config.ToolCount > 0 {
			// Highest priority: has connected before and has tools
			priority = 1
		} else {
			// Check if last connection attempt had an error
			connectionStatus := client.GetConnectionStatus()
			if lastError, ok := connectionStatus["last_error"].(string); ok && lastError != "" {
				// Lowest priority: last connection had an error
				priority = 3
			}
		}

		jobs = append(jobs, clientJob{
			id:       id,
			client:   client,
			priority: priority,
		})
	}

	if len(jobs) == 0 {
		m.logger.Debug("No clients need connection")
		return nil
	}

	// Sort jobs by priority (lower number = higher priority = connect first)
	sort.Slice(jobs, func(i, j int) bool {
		return jobs[i].priority < jobs[j].priority
	})

	// Helper function to count jobs by priority
	countByPriority := func(priority int) int {
		count := 0
		for _, job := range jobs {
			if job.priority == priority {
				count++
			}
		}
		return count
	}

	// Get concurrency limit from config
	maxConcurrent := m.globalConfig.MaxConcurrentConnections
	if maxConcurrent <= 0 {
		maxConcurrent = 20 // Fallback default
	}

	m.logger.Info("ConnectAll starting with controlled concurrency and priority-based ordering",
		zap.Int("total_clients", len(jobs)),
		zap.Int("max_concurrent", maxConcurrent),
		zap.Int("priority_1_servers", countByPriority(1)),
		zap.Int("priority_2_servers", countByPriority(2)),
		zap.Int("priority_3_servers", countByPriority(3)))

	// Create semaphore channel to limit concurrent connections
	semaphore := make(chan struct{}, maxConcurrent)

	var wg sync.WaitGroup

	// Connect to all clients with concurrency control
	for _, job := range jobs {
		wg.Add(1)

		// Acquire semaphore
		semaphore <- struct{}{}

		go func(id string, c *managed.Client) {
			defer wg.Done()
			defer func() { <-semaphore }() // Release semaphore

			m.logger.Info("Attempting to connect client",
				zap.String("id", id),
				zap.String("name", c.Config.Name),
				zap.String("url", c.Config.URL),
				zap.String("command", c.Config.Command),
				zap.String("protocol", c.Config.Protocol))

			if err := c.Connect(ctx); err != nil {
				m.logger.Error("Failed to connect to upstream server",
					zap.String("id", id),
					zap.String("name", c.Config.Name),
					zap.String("state", c.GetState().String()),
					zap.Error(err))
			} else {
				m.logger.Info("Successfully initiated connection to upstream server",
					zap.String("id", id),
					zap.String("name", c.Config.Name))
			}
		}(job.id, job.client)
	}

	wg.Wait()
	m.logger.Info("ConnectAll completed",
		zap.Int("attempted", len(jobs)))

	return nil
}

// DisconnectAll disconnects from all servers
func (m *Manager) DisconnectAll() error {
	m.mu.RLock()
	clients := make([]*managed.Client, 0, len(m.clients))
	for _, client := range m.clients {
		clients = append(clients, client)
	}
	m.mu.RUnlock()

	var lastError error
	for _, client := range clients {
		if err := client.Disconnect(); err != nil {
			lastError = err
		}
	}

	return lastError
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

		if client.GetServerInfo() != nil {
			info := client.GetServerInfo()
			status["protocol_version"] = info.ProtocolVersion
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
// This is optimized to avoid network calls during shutdown for performance
func (m *Manager) GetTotalToolCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	totalTools := 0
	for _, client := range m.clients {
		if !client.Config.Enabled || !client.IsConnected() {
			continue
		}

		// Quick check if client is actually reachable before making network call
		if !client.IsConnected() {
			continue
		}

		// Use timeout for UI status updates (30 seconds for SSE servers)
		// This allows time for SSE servers to establish connections and respond
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

		m.logger.Debug("Starting ListTools for tool counting",
			zap.Duration("timeout", 30*time.Second))
		tools, err := client.ListTools(ctx)
		cancel()
		if err == nil && tools != nil {
			totalTools += len(tools)
		}
		// Silently ignore errors during tool counting to avoid noise during shutdown
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
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
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

	ticker := time.NewTicker(5 * time.Second) // Check every 5 seconds
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
		// Only consider enabled, HTTP/SSE servers not currently connected
		if !c.Config.Enabled || c.IsConnected() {
			continue
		}

		state := c.GetState()
		// Focus on Error state likely due to OAuth/authorization
		if state != types.StateError {
			continue
		}

		// Rate-limit triggers per server
		if last, ok := m.tokenReconnect[id]; ok && now.Sub(last) < 10*time.Second {
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
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
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
					tctx, tcancel := context.WithTimeout(ctx, 10*time.Second)
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

	ticker := time.NewTicker(60 * time.Second) // Check every 60 seconds
	defer ticker.Stop()

	for range ticker.C {
		m.performHealthChecks()
	}
}

// performHealthChecks checks the health of all servers with HealthCheck flag enabled
func (m *Manager) performHealthChecks() {
	m.mu.RLock()
	clients := make(map[string]*managed.Client)
	for id, client := range m.clients {
		// Only check servers with HealthCheck enabled
		if client.Config.HealthCheck {
			clients[id] = client
		}
	}
	m.mu.RUnlock()

	if len(clients) == 0 {
		return
	}

	m.logger.Debug("Performing health checks", zap.Int("servers_with_health_check", len(clients)))

	for id, client := range clients {
		// Skip if not enabled
		if !client.Config.Enabled {
			continue
		}

		// Check connection status
		if !client.IsConnected() {
			state := client.GetState()
			m.logger.Info("Health check: Server not connected, attempting reconnection",
				zap.String("id", id),
				zap.String("name", client.Config.Name),
				zap.String("state", state.String()))

			// Attempt to reconnect
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			err := client.Connect(ctx)
			cancel()

			if err != nil {
				m.logger.Warn("Health check: Failed to reconnect server",
					zap.String("id", id),
					zap.String("name", client.Config.Name),
					zap.Error(err))
			} else {
				m.logger.Info("Health check: Successfully reconnected server",
					zap.String("id", id),
					zap.String("name", client.Config.Name))
			}
		} else {
			m.logger.Debug("Health check: Server is healthy",
				zap.String("id", id),
				zap.String("name", client.Config.Name))
		}
	}
}

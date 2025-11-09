package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/mark3labs/mcp-go/server"
	"go.uber.org/zap"

	"mcpproxy-go/internal/cache"
	"mcpproxy-go/internal/config"
	"mcpproxy-go/internal/events"
	"mcpproxy-go/internal/index"
	"mcpproxy-go/internal/logs"
	"mcpproxy-go/internal/startup"
	"mcpproxy-go/internal/storage"
	"mcpproxy-go/internal/truncate"
	"mcpproxy-go/internal/upstream"
)

// Group represents a server group
type Group struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Icon        string `json:"icon_emoji,omitempty"`
	Color       string `json:"color"`
}

// Global groups storage
var (
	groups     = make(map[string]*Group)
	groupsMutex = sync.RWMutex{}
)

// toolCountCache represents cached tool count information
type toolCountCache struct {
	count      int
	lastUpdate time.Time
}

// Status represents the current status of the server
type Status struct {
	Phase         string                 `json:"phase"`          // Starting, Ready, Error
	Message       string                 `json:"message"`        // Human readable status message
	UpstreamStats map[string]interface{} `json:"upstream_stats"` // Upstream server statistics
	ToolsIndexed  int                    `json:"tools_indexed"`  // Number of tools indexed
	LastUpdated   time.Time              `json:"last_updated"`
}

// Server wraps the MCP proxy server with all its dependencies
type Server struct {
	config          *config.Config
	configPath      string // Store the actual config file path used
	logger          *zap.Logger
	storageManager  *storage.Manager
	indexManager    *index.Manager
	upstreamManager *upstream.Manager
	cacheManager    *cache.Manager
	truncator       *truncate.Truncator
	mcpProxy        *MCPProxyServer
	eventBus        *events.EventBus // Event bus for state change notifications

	// Startup script manager
	startupManager *startup.Manager

	// MCP Inspector manager
	inspectorManager *InspectorManager

	// Semantic search service
	semanticSearchService *SemanticSearchService

	// Server control
	httpServer *http.Server
	running    bool
	mu         sync.RWMutex

	// Separate contexts for different lifecycles
	appCtx       context.Context    // Application-wide context (only cancelled on shutdown)
	appCancel    context.CancelFunc // Application-wide cancel function
	serverCtx    context.Context    // HTTP server context (cancelled on stop/start)
	serverCancel context.CancelFunc // HTTP server cancel function
	shutdown     bool               // Guard against double shutdown

	// Status reporting
	status   Status
	statusMu sync.RWMutex
	statusCh chan Status

	// Tool count cache to avoid excessive ListTools operations
	toolCountCache map[string]*toolCountCache
	toolCountMu    sync.RWMutex
}

// NewServer creates a new server instance
func NewServer(cfg *config.Config, logger *zap.Logger) (*Server, error) {
	return NewServerWithConfigPath(cfg, "", logger)
}

// NewServerWithConfigPath creates a new server instance with explicit config path tracking
func NewServerWithConfigPath(cfg *config.Config, configPath string, logger *zap.Logger) (*Server, error) {
	// Initialize storage manager
	storageManager, err := storage.NewManager(cfg.DataDir, logger.Sugar())
	if err != nil {
		return nil, fmt.Errorf("failed to initialize storage manager: %w", err)
	}

	// Initialize index manager
	indexManager, err := index.NewManager(cfg.DataDir, logger)
	if err != nil {
		storageManager.Close()
		return nil, fmt.Errorf("failed to initialize index manager: %w", err)
	}

	// Initialize upstream manager
	upstreamManager := upstream.NewManager(logger, cfg, storageManager.GetBoltDB())

	// Set logging configuration on upstream manager for per-server logging
	if cfg.Logging != nil {
		upstreamManager.SetLogConfig(cfg.Logging)
	}

	// Initialize cache manager
	cacheManager, err := cache.NewManager(storageManager.GetDB(), logger)
	if err != nil {
		storageManager.Close()
		indexManager.Close()
		return nil, fmt.Errorf("failed to initialize cache manager: %w", err)
	}

	// Initialize truncator
	truncator := truncate.NewTruncator(cfg.ToolResponseLimit)

	// Create a context that will be used for background operations
	ctx, cancel := context.WithCancel(context.Background())

	// Initialize event bus for state change notifications
	eventBus := events.NewEventBus()

    server := &Server{
		config:          cfg,
		configPath:      configPath,
		logger:          logger,
		storageManager:  storageManager,
		indexManager:    indexManager,
		upstreamManager: upstreamManager,
		cacheManager:    cacheManager,
		truncator:       truncator,
		eventBus:        eventBus,
		appCtx:          ctx,
		appCancel:       cancel,
		statusCh:        make(chan Status, 10), // Buffered channel for status updates
		toolCountCache:  make(map[string]*toolCountCache), // Initialize tool count cache
		status: Status{
			Phase:       "Initializing",
			Message:     "Server is initializing...",
			LastUpdated: time.Now(),
		},
	}

	// Initialize startup script manager
	server.startupManager = startup.NewManager(cfg.StartupScript, logger.Sugar())

	// Initialize MCP Inspector manager
	server.inspectorManager = NewInspectorManager(logger.Sugar())

	// Initialize semantic search service
	semanticSearchURL := os.Getenv("SEMANTIC_SEARCH_URL")
	if semanticSearchURL == "" {
		semanticSearchURL = "http://127.0.0.1:8081"
	}
	server.semanticSearchService = NewSemanticSearchService(semanticSearchURL, logger)

	// Check if semantic search is available
	if server.semanticSearchService.IsAvailable(context.Background()) {
		logger.Info("Semantic search service available", zap.String("url", semanticSearchURL))
	} else {
		logger.Warn("Semantic search service not available - semantic_search_tools will not work",
			zap.String("url", semanticSearchURL),
			zap.String("help", "Start with: python3 agent/semantic_search_api.py"))
	}

	// Create MCP proxy server
	mcpProxy := NewMCPProxyServer(storageManager, indexManager, upstreamManager, cacheManager, truncator, logger, server, cfg.DebugSearch, cfg)

	server.mcpProxy = mcpProxy

	// Initialize groups from config
	server.initGroupsFromConfig()

	// Migrate legacy group_name -> group_id if present
	server.migrateLegacyGroupNamesToIDs()

	// Initialize server-group assignments from config
	server.initServerGroupAssignments()

	// Setup event bridge to connect StateManager to EventBus
	server.setupEventBridge()

	// Start background initialization immediately
	go server.backgroundInitialization()

	return server, nil
}

// GetStatus returns the current server status
func (s *Server) GetStatus() interface{} {
	s.statusMu.RLock()
	defer s.statusMu.RUnlock()
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Create a map representation of the status for the tray
	statusMap := map[string]interface{}{
		"running":        s.running,
		"listen_addr":    s.GetListenAddress(),
		"phase":          s.status.Phase,
		"message":        s.status.Message,
		"upstream_stats": s.status.UpstreamStats,
		"tools_indexed":  s.status.ToolsIndexed,
		"last_updated":   s.status.LastUpdated,
	}

	return statusMap
}

// TriggerOAuthLogin starts an in-process OAuth flow for the given server name.
// Used by the tray to avoid cross-process DB locking issues during OAuth.
func (s *Server) TriggerOAuthLogin(serverName string) error {
	s.logger.Info("Tray requested OAuth login", zap.String("server", serverName))
	if s.upstreamManager == nil {
		return fmt.Errorf("upstream manager not initialized")
	}
	if err := s.upstreamManager.StartManualOAuth(serverName, true); err != nil {
		s.logger.Error("Failed to start in-process OAuth", zap.String("server", serverName), zap.Error(err))
		return err
	}
	return nil
}

// StatusChannel returns a channel that receives status updates
func (s *Server) StatusChannel() <-chan interface{} {
	// Create a new channel that converts Status to interface{}
	ch := make(chan interface{}, 10)
	go func() {
		defer close(ch)
		for status := range s.statusCh {
			ch <- status
		}
	}()
	return ch
}

// updateStatus updates the current status and notifies subscribers
func (s *Server) updateStatus(phase, message string) {
	s.statusMu.Lock()
	s.status.Phase = phase
	s.status.Message = message
	s.status.LastUpdated = time.Now()
	s.status.UpstreamStats = s.upstreamManager.GetStats()
	s.status.ToolsIndexed = s.getIndexedToolCount()
	status := s.status
	s.statusMu.Unlock()

	// Non-blocking send to status channel
	select {
	case s.statusCh <- status:
	default:
		// If channel is full, skip this update
	}

	s.logger.Info("Status updated", zap.String("phase", phase), zap.String("message", message))
}

// getIndexedToolCount returns the number of indexed tools
func (s *Server) getIndexedToolCount() int {
	stats := s.upstreamManager.GetStats()
	if totalTools, ok := stats["total_tools"].(int); ok {
		return totalTools
	}
	return 0
}

// backgroundInitialization handles server initialization in the background
func (s *Server) backgroundInitialization() {
	s.updateStatus("Loading", "Loading configuration and connecting to servers...")

	// Load configured servers from storage and add to upstream manager
	if err := s.loadConfiguredServers(); err != nil {
		s.logger.Error("Failed to load configured servers", zap.Error(err))
		s.updateStatus("Error", fmt.Sprintf("Failed to load servers: %v", err))
		return
	}

	// Start background connection attempts using application context
	s.updateStatus("Connecting", "Connecting to upstream servers...")
	s.mu.RLock()
	appCtx := s.appCtx // Use application context, not server context
	s.mu.RUnlock()
	go s.backgroundConnections(appCtx)

	// Start background tool discovery and indexing using application context
	s.mu.RLock()
	appCtx = s.appCtx // Use application context, not server context
	s.mu.RUnlock()
	go s.backgroundToolIndexing(appCtx)

	// Only set "Ready" status if the server is not already running
	// If server is running, don't override the "Running" status
	s.mu.RLock()
	isRunning := s.running
	s.mu.RUnlock()

	if !isRunning {
		s.updateStatus("Ready", "Server is ready (connections continue in background)")
	}
}

// backgroundConnections handles connecting to upstream servers with retry logic
func (s *Server) backgroundConnections(ctx context.Context) {
	// Initial connection attempt
	s.connectAllWithRetry(ctx)

	// Start periodic reconnection attempts for failed connections (less aggressive)
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.connectAllWithRetry(ctx)
		case <-ctx.Done():
			s.logger.Info("Background connections stopped due to context cancellation")
			return
		}
	}
}

// connectAllWithRetry attempts to connect to all servers with exponential backoff
func (s *Server) connectAllWithRetry(ctx context.Context) {
	stats := s.upstreamManager.GetStats()
	connectedCount := 0
	totalCount := 0

	if serverStats, ok := stats["servers"].(map[string]interface{}); ok {
		totalCount = len(serverStats)
		for _, serverStat := range serverStats {
			if stat, ok := serverStat.(map[string]interface{}); ok {
				if connected, ok := stat["connected"].(bool); ok && connected {
					connectedCount++
				}
			}
		}
	}

	if connectedCount < totalCount {
		// Only update status to "Connecting" if server is not running
		// If server is running, don't override the "Running" status
		s.mu.RLock()
		isRunning := s.running
		s.mu.RUnlock()

		if !isRunning {
			s.updateStatus("Connecting", fmt.Sprintf("Connected to %d/%d servers, retrying...", connectedCount, totalCount))
		}

		// Try to connect with timeout
		connectCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		if err := s.upstreamManager.ConnectAll(connectCtx); err != nil {
			s.logger.Warn("Some upstream servers failed to connect", zap.Error(err))
		}
	}
}

// backgroundToolIndexing handles initial tool discovery for servers that need it
// Tools are ONLY loaded at startup for servers with StartOnBoot=true or when lazy loading is disabled
func (s *Server) backgroundToolIndexing(ctx context.Context) {
	// Wait for connections to establish
	select {
	case <-time.After(2 * time.Second):
	case <-ctx.Done():
		s.logger.Info("Background tool indexing stopped during initial delay")
		return
	}

	// ONLY load tools for servers that should start on boot OR when lazy loading is disabled
	if s.config.EnableLazyLoading {
		s.logger.Info("Lazy loading enabled - skipping automatic tool discovery",
			zap.String("note", "Tools will be loaded on-demand or for servers with StartOnBoot=true"))

		// Load tools ONLY for servers with StartOnBoot=true
		if err := s.loadToolsForStartOnBootServers(ctx); err != nil {
			s.logger.Error("Failed to load tools for StartOnBoot servers", zap.Error(err))
		}
	} else {
		// Lazy loading disabled - load all tools
		s.logger.Info("Lazy loading disabled - loading tools for all connected servers")
		if err := s.discoverAndIndexTools(ctx); err != nil {
			s.logger.Error("Failed to discover and index tools", zap.Error(err))
		}
	}

	// NOTE: Removed periodic re-indexing ticker
	// Tools should only be reloaded via:
	// 1. Manual reload from tray UI
	// 2. Server-specific health checks (if configured)
	// 3. Lazy loading wake-up when tool is called
}

// loadToolsForStartOnBootServers loads tools ONLY for servers with StartOnBoot=true
// This respects lazy loading while still allowing specific servers to load at startup
func (s *Server) loadToolsForStartOnBootServers(ctx context.Context) error {
	s.logger.Info("Loading tools for StartOnBoot servers only")

	// Get all server names from upstream manager
	serverNames := s.upstreamManager.GetAllServerNames()

	var toolsToIndex []*config.ToolMetadata
	startOnBootCount := 0

	for _, serverName := range serverNames {
		client, exists := s.upstreamManager.GetClient(serverName)
		if !exists {
			continue
		}

		// ONLY load tools if StartOnBoot is true
		if !client.Config.StartOnBoot {
			s.logger.Debug("Skipping server (StartOnBoot=false)",
				zap.String("server", serverName))
			continue
		}

		// Check if server is connected
		if !client.IsConnected() {
			s.logger.Debug("Skipping server (not connected)",
				zap.String("server", serverName),
				zap.Bool("start_on_boot", client.Config.StartOnBoot))
			continue
		}

		s.logger.Info("Loading tools for StartOnBoot server",
			zap.String("server", serverName))

		// Call ListTools for this specific server
		tools, err := client.ListTools(ctx)
		if err != nil {
			s.logger.Error("Failed to list tools for StartOnBoot server",
				zap.String("server", serverName),
				zap.Error(err))
			continue
		}

		// Save tools to database for lazy loading
		if err := s.storageManager.SaveToolMetadata(serverName, tools); err != nil {
			s.logger.Error("Failed to save tool metadata to database",
				zap.String("server", serverName),
				zap.Error(err))
			// Continue anyway - tools will still be indexed
		}

		// Prefix tools with server name for indexing
		for _, tool := range tools {
			prefixedTool := &config.ToolMetadata{
				Name:        fmt.Sprintf("%s:%s", serverName, tool.Name),
				ServerName:  serverName,
				Description: tool.Description,
				ParamsJSON:  tool.ParamsJSON,
				Hash:        tool.Hash,
				Created:     tool.Created,
				Updated:     tool.Updated,
			}
			toolsToIndex = append(toolsToIndex, prefixedTool)
		}

		startOnBootCount++
		s.logger.Info("Loaded and saved tools from StartOnBoot server",
			zap.String("server", serverName),
			zap.Int("tool_count", len(tools)))
	}

	// Index all collected tools
	if len(toolsToIndex) > 0 {
		if err := s.indexManager.BatchIndexTools(toolsToIndex); err != nil {
			return fmt.Errorf("failed to index StartOnBoot tools: %w", err)
		}
		s.logger.Info("Successfully indexed StartOnBoot tools",
			zap.Int("server_count", startOnBootCount),
			zap.Int("total_tools", len(toolsToIndex)))
	} else {
		s.logger.Info("No StartOnBoot servers found or all sleeping")
	}

	return nil
}

// loadConfiguredServers synchronizes the storage and upstream manager from the current config.
// This is the source of truth when configuration is loaded from disk.
//
//nolint:unparam // function designed to be best-effort, always returns nil by design
func (s *Server) loadConfiguredServers() error {
	s.logger.Info("Synchronizing servers from configuration (config as source of truth)")

	// Initialize server-group assignments from config
	s.initServerGroupAssignments()

	// Get current state for comparison
	currentUpstreams := s.upstreamManager.GetAllServerNames()
	storedServers, err := s.storageManager.ListUpstreamServers()
	if err != nil {
		s.logger.Error("Failed to get stored servers for sync", zap.Error(err))
		storedServers = []*config.ServerConfig{} // Continue with empty list
	}

	// Create a stable copy of the servers list to avoid race conditions during config reload
	s.mu.RLock()
	serversCopy := make([]*config.ServerConfig, len(s.config.Servers))
	copy(serversCopy, s.config.Servers)
	s.mu.RUnlock()

	// Create maps for efficient lookups
	configuredServers := make(map[string]*config.ServerConfig)
	storedServerMap := make(map[string]*config.ServerConfig)

	for _, serverCfg := range serversCopy {
		configuredServers[serverCfg.Name] = serverCfg
	}

	for _, storedServer := range storedServers {
		storedServerMap[storedServer.Name] = storedServer
	}

	// Sync config to storage and upstream manager (with parallel startup)
	s.mu.RLock()
	maxConcurrent := s.config.MaxConcurrentConnections
	s.mu.RUnlock()

	if maxConcurrent <= 0 {
		maxConcurrent = 20 // Default to 20 concurrent connections
	}

	s.logger.Info("Starting server sync",
		zap.Int("total_servers", len(serversCopy)),
		zap.Int("max_concurrent", maxConcurrent))

	// Create semaphore for concurrency control
	semaphore := make(chan struct{}, maxConcurrent)
	var wg sync.WaitGroup
	var mu sync.Mutex
	errorCount := 0

	for i := range serversCopy {
		serverCfg := serversCopy[i] // Use the stable copy to avoid race conditions

		// Check if server state has changed
		storedServer, existsInStorage := storedServerMap[serverCfg.Name]
		hasChanged := !existsInStorage ||
			storedServer.Enabled != serverCfg.Enabled ||
			storedServer.Quarantined != serverCfg.Quarantined ||
			storedServer.URL != serverCfg.URL ||
			storedServer.Command != serverCfg.Command ||
			storedServer.Protocol != serverCfg.Protocol

		if hasChanged {
			s.logger.Info("Server configuration changed, updating storage",
				zap.String("server", serverCfg.Name),
				zap.Bool("new", !existsInStorage),
				zap.Bool("enabled_changed", existsInStorage && storedServer.Enabled != serverCfg.Enabled),
				zap.Bool("quarantined_changed", existsInStorage && storedServer.Quarantined != serverCfg.Quarantined))
		}

		// Always sync config to storage (ensures consistency) - sequential for DB writes
		if err := s.storageManager.SaveUpstreamServer(serverCfg); err != nil {
			s.logger.Error("Failed to save/update server in storage", zap.Error(err), zap.String("server", serverCfg.Name))
			continue
		}

		// Sync to upstream manager based on enabled status - parallel
		if serverCfg.Enabled {
			wg.Add(1)
			go func(cfg *config.ServerConfig) {
				defer wg.Done()

				// Acquire semaphore
				semaphore <- struct{}{}
				defer func() { <-semaphore }() // Release

				// Add server to upstream manager regardless of quarantine status
				// Quarantined servers are kept connected for inspection but blocked for execution
				if err := s.upstreamManager.AddServer(cfg.Name, cfg); err != nil {
					mu.Lock()
					errorCount++
					mu.Unlock()
					s.logger.Error("Failed to add/update upstream server", zap.Error(err), zap.String("server", cfg.Name))
				}

				if cfg.Quarantined {
					s.logger.Info("Server is quarantined but kept connected for security inspection", zap.String("server", cfg.Name))
				}
			}(serverCfg)
		} else {
			// Remove from upstream manager only if disabled (not quarantined)
			s.upstreamManager.RemoveServer(serverCfg.Name)
			s.logger.Info("Server is disabled, removing from active connections", zap.String("server", serverCfg.Name))
		}
	}

	// Wait for all parallel operations to complete
	wg.Wait()

	s.logger.Info("Server sync completed",
		zap.Int("total_servers", len(s.config.Servers)),
		zap.Int("errors", errorCount))

	// Remove servers that are no longer in config (comprehensive cleanup)
	serversToRemove := []string{}

	// Check upstream manager
	for _, serverName := range currentUpstreams {
		if _, exists := configuredServers[serverName]; !exists {
			serversToRemove = append(serversToRemove, serverName)
		}
	}

	// Check storage for orphaned servers
	for _, storedServer := range storedServers {
		if _, exists := configuredServers[storedServer.Name]; !exists {
			// Add to removal list if not already there
			found := false
			for _, name := range serversToRemove {
				if name == storedServer.Name {
					found = true
					break
				}
			}
			if !found {
				serversToRemove = append(serversToRemove, storedServer.Name)
			}
		}
	}

	// Perform comprehensive cleanup for removed servers
	for _, serverName := range serversToRemove {
		s.logger.Info("Removing server no longer in config", zap.String("server", serverName))

		// Remove from upstream manager
		s.upstreamManager.RemoveServer(serverName)

		// Remove from storage
		if err := s.storageManager.DeleteUpstreamServer(serverName); err != nil {
			s.logger.Error("Failed to delete server from storage", zap.Error(err), zap.String("server", serverName))
		}

		// Remove tools from search index
		if err := s.indexManager.DeleteServerTools(serverName); err != nil {
			s.logger.Error("Failed to delete server tools from index", zap.Error(err), zap.String("server", serverName))
		} else {
			s.logger.Info("Removed server tools from search index", zap.String("server", serverName))
		}
	}

	if len(serversToRemove) > 0 {
		s.logger.Info("Comprehensive server cleanup completed",
			zap.Int("removed_count", len(serversToRemove)),
			zap.Strings("removed_servers", serversToRemove))
	}

	s.logger.Info("Server synchronization completed",
		zap.Int("configured_servers", len(s.config.Servers)),
		zap.Int("removed_servers", len(serversToRemove)))

	return nil
}

// Start starts the MCP proxy server
func (s *Server) Start(ctx context.Context) error {
	s.logger.Info("Starting MCP proxy server - HTTP server will start immediately")

	// Handle graceful shutdown when context is cancelled (for full application shutdown only)
	go func() {
		<-ctx.Done()
		s.logger.Info("Main context cancelled, shutting down server")
		// First shutdown the HTTP server
		if err := s.StopServer(); err != nil {
			s.logger.Error("Error stopping server during context cancellation", zap.Error(err))
		}
		// Then shutdown the rest (only for full application shutdown, not server restarts)
		// We distinguish this by checking if the cancelled context is the application context
		s.mu.Lock()
		alreadyShutdown := s.shutdown
		isAppContext := (ctx == s.appCtx)
		s.mu.Unlock()

		if !alreadyShutdown && isAppContext {
			s.logger.Info("Application context cancelled, performing full shutdown")
			if err := s.Shutdown(); err != nil {
				s.logger.Error("Error during context-triggered shutdown", zap.Error(err))
			}
		} else if !isAppContext {
			s.logger.Info("Server context cancelled, server stop completed")
		}

		s.logger.Info("SERVER SHUTDOWN SEQUENCE COMPLETED")
		_ = s.logger.Sync()
	}()

	// Run background initialization tasks AFTER HTTP server starts
	go func() {
		s.logger.Info("Starting background initialization tasks")

		// Clean up any orphaned Docker containers from previous runs
		s.cleanupOrphanedDockerContainers(ctx)

		// Start startup script (best-effort) in background
		if s.startupManager != nil {
			if err := s.startupManager.Start(ctx); err != nil {
				s.logger.Warn("Failed to start startup script", zap.Error(err))
			} else {
				s.logger.Info("Startup script initialization completed")
			}
		}

		s.logger.Info("Background initialization tasks completed")
	}()

    // Determine transport mode based on listen address
	if s.config.Listen != "" && s.config.Listen != ":0" {
		// Start the MCP server in HTTP mode (Streamable HTTP) IMMEDIATELY
		s.logger.Info("Starting MCP HTTP server IMMEDIATELY (before upstream connections)",
			zap.String("transport", "streamable-http"),
			zap.String("listen", s.config.Listen))

		// Update status to show server is now running
		s.updateStatus("Running", fmt.Sprintf("Server is running on %s", s.config.Listen))

		// Create Streamable HTTP server with custom routing
		streamableServer := server.NewStreamableHTTPServer(s.mcpProxy.GetMCPServer())

		// Create custom HTTP server for handling multiple routes
		s.logger.Info("About to call startCustomHTTPServer")
		if err := s.startCustomHTTPServer(streamableServer); err != nil {
			return fmt.Errorf("MCP Streamable HTTP server error: %w", err)
		}
	} else {
		// Start the MCP server in stdio mode
		s.logger.Info("Starting MCP server", zap.String("transport", "stdio"))

		// Update status to show server is now running
		s.updateStatus("Running", "Server is running in stdio mode")

		// Serve using stdio (standard MCP transport)
		if err := server.ServeStdio(s.mcpProxy.GetMCPServer()); err != nil {
			return fmt.Errorf("MCP server error: %w", err)
		}
	}

	return nil
}

// discoverAndIndexTools discovers tools from upstream servers and indexes them
func (s *Server) discoverAndIndexTools(ctx context.Context) error {
	s.logger.Info("Discovering and indexing tools...")

	tools, err := s.upstreamManager.DiscoverTools(ctx)
	if err != nil {
		return fmt.Errorf("failed to discover tools: %w", err)
	}

	if len(tools) == 0 {
		s.logger.Warn("No tools discovered from upstream servers")
		return nil
	}

	// Group tools by server for database storage
	toolsByServer := make(map[string][]*config.ToolMetadata)
	for _, tool := range tools {
		serverID := tool.ServerName
		if serverID == "" {
			// If ServerName is not set, skip this tool
			s.logger.Warn("Tool has no ServerName, skipping database save",
				zap.String("tool", tool.Name))
			continue
		}
		toolsByServer[serverID] = append(toolsByServer[serverID], tool)
	}

	// Save tools to database by server
	for serverID, serverTools := range toolsByServer {
		if err := s.storageManager.SaveToolMetadata(serverID, serverTools); err != nil {
			s.logger.Error("Failed to save tool metadata to database",
				zap.String("server", serverID),
				zap.Int("tool_count", len(serverTools)),
				zap.Error(err))
			// Continue anyway - tools will still be indexed
		} else {
			s.logger.Info("Saved tool metadata to database",
				zap.String("server", serverID),
				zap.Int("tool_count", len(serverTools)))
		}
	}

	// Index tools
	if err := s.indexManager.BatchIndexTools(tools); err != nil {
		return fmt.Errorf("failed to index tools: %w", err)
	}

	s.logger.Info("Successfully discovered, saved, and indexed tools",
		zap.Int("total_tools", len(tools)),
		zap.Int("servers", len(toolsByServer)))
	return nil
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown() error {
	s.mu.Lock()
	if s.shutdown {
		s.mu.Unlock()
		s.logger.Info("Server already shutdown, skipping")
		return nil
	}
	s.shutdown = true
	httpServer := s.httpServer
	s.mu.Unlock()

	s.logger.Info("Shutting down MCP proxy server...")

    // First stop startup script and its subprocesses
    if s.startupManager != nil {
        s.logger.Info("Stopping startup script")
        if err := s.startupManager.Stop(); err != nil {
            s.logger.Warn("Failed to stop startup script", zap.Error(err))
        }
    }

    // Stop MCP Inspector if running
    if s.inspectorManager != nil {
        s.logger.Info("Stopping MCP Inspector")
        if err := s.inspectorManager.Stop(); err != nil {
            s.logger.Warn("Failed to stop MCP Inspector", zap.Error(err))
        }
    }

    // Gracefully shutdown HTTP server first to stop accepting new connections
	if httpServer != nil {
		s.logger.Info("Gracefully shutting down HTTP server...")
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := httpServer.Shutdown(ctx); err != nil {
			s.logger.Warn("HTTP server forced shutdown due to timeout", zap.Error(err))
			// Force close if graceful shutdown times out
			httpServer.Close()
		} else {
			s.logger.Info("HTTP server shutdown completed gracefully")
		}
	}

	// Cancel the server context to stop all background operations
	if s.appCancel != nil {
		s.logger.Info("Cancelling server context to stop background operations")
		s.appCancel()
	}

	// Disconnect upstream servers
	if err := s.upstreamManager.DisconnectAll(); err != nil {
		s.logger.Error("Failed to disconnect upstream servers", zap.Error(err))
	}

	// Close managers
	if s.cacheManager != nil {
		s.cacheManager.Close()
	}

	if err := s.indexManager.Close(); err != nil {
		s.logger.Error("Failed to close index manager", zap.Error(err))
	}

	if err := s.storageManager.Close(); err != nil {
		s.logger.Error("Failed to close storage manager", zap.Error(err))
	}

	s.logger.Info("MCP proxy server shutdown complete")
	return nil
}

// IsRunning returns whether the server is currently running
func (s *Server) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// GetEventBus returns the event bus for subscribing to events
func (s *Server) GetEventBus() *events.EventBus {
	return s.eventBus
}

// setupEventBridge connects StateManager callbacks to the EventBus
// This should be called after upstream manager is initialized
func (s *Server) setupEventBridge() {
	s.logger.Info("Setting up event bridge for state change notifications")

	// Pass the event bus to the upstream manager
	// It will then publish events for all state changes
	s.upstreamManager.SetEventBus(s.eventBus)

	s.logger.Info("Event bridge ready - state changes will be published to event bus")
}

// GetListenAddress returns the address the server is listening on
func (s *Server) GetListenAddress() string {
	return s.config.Listen
}

// GetUpstreamStats returns statistics about upstream servers
func (s *Server) GetUpstreamStats() map[string]interface{} {
	stats := s.upstreamManager.GetStats()

	// Enhance stats with tool counts per server
	if servers, ok := stats["servers"].(map[string]interface{}); ok {
		for id, serverInfo := range servers {
			if serverMap, ok := serverInfo.(map[string]interface{}); ok {
				// Get tool count for this server
				toolCount := s.getServerToolCount(id)
				serverMap["tool_count"] = toolCount
			}
		}
	}

	return stats
}

// GetAllServers returns information about all upstream servers for tray UI
func (s *Server) GetAllServers() ([]map[string]interface{}, error) {
	// Check if storage manager is available
	if s.storageManager == nil {
		return []map[string]interface{}{}, nil
	}

	servers, err := s.storageManager.ListUpstreamServers()
	if err != nil {
		// Handle database closed gracefully
		if strings.Contains(err.Error(), "database not open") || strings.Contains(err.Error(), "closed") {
			s.logger.Debug("Database not available for GetAllServers, returning empty list")
			return []map[string]interface{}{}, nil
		}
		return nil, err
	}

	// Build a lookup from config to enrich with group info (authoritative for assignments)
	configByName := make(map[string]*config.ServerConfig, len(s.config.Servers))
	for _, cfg := range s.config.Servers {
		configByName[cfg.Name] = cfg
	}

	// Debug: Log server count discrepancy
	configServerCount := len(s.config.Servers)
	dbServerCount := len(servers)
	if configServerCount != dbServerCount {
		s.logger.Warn("Server count mismatch detected", 
			zap.Int("config_servers", configServerCount),
			zap.Int("db_servers", dbServerCount),
			zap.Int("missing", configServerCount-dbServerCount))
	}

	var result []map[string]interface{}
	for _, server := range servers {
		// Get connection status and tool count from upstream manager
		var connected bool
		var connecting bool
		var sleeping bool
		var connectionState string
		var lastError string
		var toolCount int

		if s.upstreamManager != nil {
			if client, exists := s.upstreamManager.GetClient(server.Name); exists {
				connectionStatus := client.GetConnectionStatus()
				if c, ok := connectionStatus["connected"].(bool); ok {
					connected = c
				}
				if c, ok := connectionStatus["connecting"].(bool); ok {
					connecting = c
				}
				if state, ok := connectionStatus["state"].(string); ok {
					connectionState = state
					sleeping = (state == "Sleeping")
				}
				if e, ok := connectionStatus["last_error"].(string); ok {
					lastError = e
				}

				if connected {
					toolCount = s.getServerToolCount(server.Name)
				}
			}
		}

		// Enrich with group assignment from config (if present)
		var groupID int
		if cfg, ok := configByName[server.Name]; ok && cfg != nil {
			groupID = cfg.GroupID
		}

		// Get description, start_on_boot, health_check from config
		var description string
		var startOnBoot bool
		var healthCheck bool
		if cfg, ok := configByName[server.Name]; ok && cfg != nil {
			description = cfg.Description
			startOnBoot = cfg.StartOnBoot
			healthCheck = cfg.HealthCheck
		}

		result = append(result, map[string]interface{}{
			"name":            server.Name,
			"description":     description,
			"url":             server.URL,
			"command":         server.Command,
			"args":            server.Args,
			"working_dir":     server.WorkingDir,
			"env":             server.Env,
			"protocol":        server.Protocol,
			"repository_url":  server.RepositoryURL,
			"enabled":         server.Enabled,
			"quarantined":     server.Quarantined,
			"created":         server.Created,
			"connected":       connected,
			"connecting":      connecting,
			"sleeping":        sleeping,
			"connection_state": connectionState,
			"tool_count":      toolCount,
			"last_error":      lastError,
			"group_id":        groupID,
			"start_on_boot":   startOnBoot,
			"health_check":    healthCheck,
		})
	}

	return result, nil
}

// GetServerTools returns tools for a specific server for configuration dialog
func (s *Server) GetServerTools(serverName string) ([]map[string]interface{}, error) {
	if s.upstreamManager == nil {
		return []map[string]interface{}{}, nil
	}

	// Get the specific upstream client for this server
	client, exists := s.upstreamManager.GetClient(serverName)
	if !exists || client == nil {
		return nil, fmt.Errorf("server %s not found or not connected", serverName)
	}

	// Get tools from the client
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	tools, err := client.ListTools(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get tools for server %s: %w", serverName, err)
	}

	var result []map[string]interface{}
	for _, tool := range tools {
		toolMap := map[string]interface{}{
			"name":        tool.Name,
			"description": tool.Description,
		}
		result = append(result, toolMap)
	}

	return result, nil
}

// GetQuarantinedServers returns information about quarantined servers for tray UI
func (s *Server) GetQuarantinedServers() ([]map[string]interface{}, error) {
	s.logger.Debug("GetQuarantinedServers called")

	// Check if storage manager is available
	if s.storageManager == nil {
		s.logger.Warn("Storage manager is nil in GetQuarantinedServers")
		return []map[string]interface{}{}, nil
	}

	s.logger.Debug("Calling storage manager ListQuarantinedUpstreamServers")
	quarantinedServers, err := s.storageManager.ListQuarantinedUpstreamServers()
	if err != nil {
		// Handle database closed gracefully
		if strings.Contains(err.Error(), "database not open") || strings.Contains(err.Error(), "closed") {
			s.logger.Debug("Database not available for GetQuarantinedServers, returning empty list")
			return []map[string]interface{}{}, nil
		}
		s.logger.Error("Failed to get quarantined servers from storage", zap.Error(err))
		return nil, err
	}

	s.logger.Debug("Retrieved quarantined servers from storage",
		zap.Int("count", len(quarantinedServers)))

	var result []map[string]interface{}
	for _, server := range quarantinedServers {
		serverMap := map[string]interface{}{
			"name":        server.Name,
			"url":         server.URL,
			"command":     server.Command,
			"protocol":    server.Protocol,
			"enabled":     server.Enabled,
			"quarantined": server.Quarantined,
			"created":     server.Created,
		}
		result = append(result, serverMap)

		s.logger.Debug("Added quarantined server to result",
			zap.String("server", server.Name),
			zap.Bool("quarantined", server.Quarantined))
	}

	s.logger.Debug("GetQuarantinedServers completed",
		zap.Int("total_result_count", len(result)))

	return result, nil
}

// UnquarantineServer removes a server from quarantine via tray UI
func (s *Server) UnquarantineServer(serverName string) error {
	return s.QuarantineServer(serverName, false)
}

// EnableServer enables/disables a server and ensures all state is synchronized.
// It acts as the entry point for changes originating from the UI or API.
func (s *Server) EnableServer(serverName string, enabled bool) error {
	s.logger.Info("Request to change server enabled state",
		zap.String("server", serverName),
		zap.Bool("enabled", enabled))

	// First, update the authoritative record in storage.
	if err := s.storageManager.EnableUpstreamServer(serverName, enabled); err != nil {
		s.logger.Error("Failed to update server enabled state in storage", zap.Error(err))
		return fmt.Errorf("failed to update server '%s' in storage: %w", serverName, err)
	}

	// Now that storage is updated, save the configuration to disk.
	// This ensures the file reflects the authoritative state.
	if err := s.SaveConfiguration(); err != nil {
		s.logger.Error("Failed to save configuration after state change", zap.Error(err))
		// Don't return here; the primary state is updated. The file watcher will eventually sync.
	}

	// Publish config change event for tray to react
	action := "disabled"
	if enabled {
		action = "enabled"
	}
	s.eventBus.Publish(events.Event{
		Type:       events.EventConfigChange,
		ServerName: serverName,
		Data: events.ConfigChangeData{
			Action: action,
		},
	})
	s.logger.Debug("Published config change event",
		zap.String("server", serverName),
		zap.String("action", action))

	// The file watcher in the tray will detect the change to the config file and
	// trigger ReloadConfiguration(), which calls loadConfiguredServers().
	// This completes the loop by updating the running state (upstreamManager) from the new config.
	s.logger.Info("Successfully persisted server state change. Relying on file watcher to sync running state.",
		zap.String("server", serverName))

	return nil
}

// RestartServer restarts an individual upstream server by disabling and re-enabling it
func (s *Server) RestartServer(serverName string) error {
	s.logger.Info("Request to restart server", zap.String("server", serverName))

	// First disable the server
	if err := s.EnableServer(serverName, false); err != nil {
		return fmt.Errorf("failed to disable server during restart: %w", err)
	}

	// Give it a moment to fully disconnect
	time.Sleep(500 * time.Millisecond)

	// Re-enable the server
	if err := s.EnableServer(serverName, true); err != nil {
		return fmt.Errorf("failed to re-enable server during restart: %w", err)
	}

	s.logger.Info("Successfully restarted server", zap.String("server", serverName))
	return nil
}

// QuarantineServer quarantines/unquarantines a server
func (s *Server) QuarantineServer(serverName string, quarantined bool) error {
	s.logger.Info("Request to change server quarantine state",
		zap.String("server", serverName),
		zap.Bool("quarantined", quarantined))

	if err := s.storageManager.QuarantineUpstreamServer(serverName, quarantined); err != nil {
		s.logger.Error("Failed to update server quarantine state in storage", zap.Error(err))
		return fmt.Errorf("failed to update quarantine state for server '%s' in storage: %w", serverName, err)
	}

	if err := s.SaveConfiguration(); err != nil {
		s.logger.Error("Failed to save configuration after quarantine state change", zap.Error(err))
	}

	// Publish config change event for tray to react
	action := "unquarantined"
	if quarantined {
		action = "quarantined"
	}
	s.eventBus.Publish(events.Event{
		Type:       events.EventConfigChange,
		ServerName: serverName,
		Data: events.ConfigChangeData{
			Action: action,
		},
	})
	s.logger.Debug("Published config change event",
		zap.String("server", serverName),
		zap.String("action", action))

	s.logger.Info("Successfully persisted server quarantine state change",
		zap.String("server", serverName),
		zap.Bool("quarantined", quarantined))

	return nil
}

// getServerToolCount returns the number of tools for a specific server
// Uses shorter timeout and better error handling for UI status display
func (s *Server) getServerToolCount(serverID string) int {
	client, exists := s.upstreamManager.GetClient(serverID)
	if !exists || !client.IsConnected() {
		return 0
	}

	// Check cache first
	s.toolCountMu.RLock()
	cached, hasCached := s.toolCountCache[serverID]
	s.toolCountMu.RUnlock()

	// Determine cache TTL (default 5 minutes if not configured)
	cacheTTL := time.Duration(300) * time.Second
	if s.config.ToolCacheTTL > 0 {
		cacheTTL = time.Duration(s.config.ToolCacheTTL) * time.Second
	}

	// Return cached value if it exists and is still valid
	if hasCached && time.Since(cached.lastUpdate) < cacheTTL {
		s.logger.Debug("Returning cached tool count",
			zap.String("server_id", serverID),
			zap.Int("count", cached.count),
			zap.Duration("age", time.Since(cached.lastUpdate)))
		return cached.count
	}

	// Cache miss or expired - read from database first (avoids ListTools calls)
	s.logger.Debug("Reading tool count from database (cache miss or expired)",
		zap.String("server_id", serverID))

	// Try to get tool count from database first
	dbTools, err := s.storageManager.GetToolMetadata(serverID)
	if err == nil && len(dbTools) > 0 {
		// Database has tools for this server - use that count
		count := len(dbTools)

		// Update cache with database value
		s.toolCountMu.Lock()
		s.toolCountCache[serverID] = &toolCountCache{
			count:      count,
			lastUpdate: time.Now(),
		}
		s.toolCountMu.Unlock()

		s.logger.Debug("Retrieved tool count from database",
			zap.String("server_id", serverID),
			zap.Int("count", count))

		return count
	}

	// Database has no tools for this server
	// NOTE: We do NOT call ListTools here anymore
	// Tools should be loaded via:
	// 1. Startup (if StartOnBoot=true or lazy loading disabled)
	// 2. Manual reload from tray UI
	// 3. Health check intervals (if configured)

	s.logger.Debug("No tools found in database for server",
		zap.String("server_id", serverID),
		zap.Error(err))

	// Return 0 for now - tools will be loaded via proper triggers
	count := 0
	s.toolCountMu.Lock()
	s.toolCountCache[serverID] = &toolCountCache{
		count:      count,
		lastUpdate: time.Now(),
	}
	s.toolCountMu.Unlock()

	s.logger.Debug("Updated tool count cache",
		zap.String("server_id", serverID),
		zap.Int("count", count))

	return count
}

// Helper functions for error classification
func isTimeoutError(err error) bool {
	errStr := err.Error()
	return strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "deadline exceeded") ||
		strings.Contains(errStr, "context canceled")
}

func isConnectionError(err error) bool {
	errStr := err.Error()
	return strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "no such host") ||
		strings.Contains(errStr, "connection reset") ||
		strings.Contains(errStr, "broken pipe")
}

// StartServer starts the server if it's not already running
func (s *Server) StartServer(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("server is already running")
	}

	// Cancel the old context before creating a new one to avoid race conditions
	if s.serverCancel != nil {
		s.serverCancel()
	}

	s.serverCtx, s.serverCancel = context.WithCancel(ctx)

	go func() {
		var serverError error

		defer func() {
			s.mu.Lock()
			s.running = false
			s.mu.Unlock()

			// Only send "Stopped" status if there was no error
			// If there was an error, the error status should remain
			if serverError == nil || serverError == context.Canceled {
				s.updateStatus("Stopped", "Server has stopped")
			}
		}()

		s.mu.Lock()
		s.running = true
		s.mu.Unlock()

		// Notify about server start
		s.updateStatus("Starting", "Server is starting...")

		serverError = s.Start(s.serverCtx)
		if serverError != nil && serverError != context.Canceled {
			s.logger.Error("Server error during background start", zap.Error(serverError))
			s.updateStatus("Error", fmt.Sprintf("Server error: %v", serverError))
		}
	}()

	return nil
}

// StopServer stops the server if it's running
func (s *Server) StopServer() error {
	s.logger.Info("STOPSERVER CALLED - STARTING SHUTDOWN SEQUENCE")
	_ = s.logger.Sync()

	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		// Return nil instead of error to prevent race condition logs
		s.logger.Debug("Server stop requested but server is not running")
		return nil
	}

	// Notify about server stopping
	s.logger.Info("STOPSERVER - Server is running, proceeding with stop")
	_ = s.logger.Sync()

	// Disconnect upstream servers FIRST to ensure Docker containers are cleaned up
	// Do this before canceling contexts to avoid interruption
	s.logger.Info("STOPSERVER - Disconnecting upstream servers EARLY")
	_ = s.logger.Sync()
	if err := s.upstreamManager.DisconnectAll(); err != nil {
		s.logger.Error("STOPSERVER - Failed to disconnect upstream servers early", zap.Error(err))
		_ = s.logger.Sync()
	} else {
		s.logger.Info("STOPSERVER - Successfully disconnected all upstream servers early")
		_ = s.logger.Sync()
	}

	// Add a brief wait to ensure Docker containers have time to be cleaned up
	// Only wait if there are actually Docker containers running
	if s.upstreamManager.HasDockerContainers() {
		s.logger.Info("STOPSERVER - Docker containers detected, waiting for cleanup to complete")
		_ = s.logger.Sync()
		time.Sleep(3 * time.Second)
		s.logger.Info("STOPSERVER - Docker container cleanup wait completed")
		_ = s.logger.Sync()
	} else {
		s.logger.Debug("STOPSERVER - No Docker containers detected, skipping cleanup wait")
		_ = s.logger.Sync()
	}

	s.updateStatus("Stopping", "Server is stopping...")

	// Cancel the server context after cleanup
	s.logger.Info("STOPSERVER - Cancelling server context")
	_ = s.logger.Sync()
	if s.serverCancel != nil {
		s.serverCancel()
	}

	// Gracefully shutdown HTTP server if it exists
	s.logger.Info("STOPSERVER - Shutting down HTTP server")
	_ = s.logger.Sync()
	if s.httpServer != nil {
		// Give the server 5 seconds to shutdown gracefully
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
			s.logger.Warn("STOPSERVER - Failed to gracefully shutdown HTTP server, forcing close", zap.Error(err))
			// Force close if graceful shutdown fails
			if closeErr := s.httpServer.Close(); closeErr != nil {
				s.logger.Error("STOPSERVER - Error forcing HTTP server close", zap.Error(closeErr))
			}
		} else {
			s.logger.Info("STOPSERVER - HTTP server shutdown successfully")
			_ = s.logger.Sync()
		}
		s.httpServer = nil
	}

	s.logger.Info("STOPSERVER - HTTP server cleanup completed")
	_ = s.logger.Sync()

    // Upstream servers already disconnected early in this method
	s.logger.Info("STOPSERVER - Upstream servers already disconnected early")
	_ = s.logger.Sync()

    // Stop startup script as part of stop sequence
    if s.startupManager != nil {
        s.logger.Info("STOPSERVER - Stopping startup script")
        _ = s.logger.Sync()
        if err := s.startupManager.Stop(); err != nil {
            s.logger.Warn("STOPSERVER - Failed to stop startup script", zap.Error(err))
            _ = s.logger.Sync()
        }
    }

    // Set running to false immediately after server is shut down
	s.running = false

	// Notify about server stopped with explicit status update
	s.updateStatus("Stopped", "Server has been stopped")

	s.logger.Info("STOPSERVER - All operations completed successfully")
	_ = s.logger.Sync() // Final log flush

	return nil
}

// startCustomHTTPServer creates a custom HTTP server that handles MCP endpoints
func (s *Server) startCustomHTTPServer(streamableServer *server.StreamableHTTPServer) error {
	mux := http.NewServeMux()

	// Create a logging wrapper for debugging client connections
	loggingHandler := func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Log incoming request with connection details
			s.logger.Debug("MCP client request received",
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.String("remote_addr", r.RemoteAddr),
				zap.String("user_agent", r.UserAgent()),
				zap.String("content_type", r.Header.Get("Content-Type")),
				zap.String("connection", r.Header.Get("Connection")),
				zap.Int64("content_length", r.ContentLength),
			)

			// Create response writer wrapper to capture status and errors
			wrappedWriter := &responseWriter{ResponseWriter: w, statusCode: 200}

			// Handle the request
			handler.ServeHTTP(wrappedWriter, r)

			duration := time.Since(start)

			// Log response with timing and status
			if wrappedWriter.statusCode >= 400 {
				s.logger.Warn("MCP client request completed with error",
					zap.String("method", r.Method),
					zap.String("path", r.URL.Path),
					zap.String("remote_addr", r.RemoteAddr),
					zap.Int("status_code", wrappedWriter.statusCode),
					zap.Duration("duration", duration),
				)
			} else {
				s.logger.Debug("MCP client request completed successfully",
					zap.String("method", r.Method),
					zap.String("path", r.URL.Path),
					zap.String("remote_addr", r.RemoteAddr),
					zap.Int("status_code", wrappedWriter.statusCode),
					zap.Duration("duration", duration),
				)
			}
		})
	}

	// Standard MCP endpoint according to the specification
	mux.Handle("/mcp", loggingHandler(streamableServer))
	mux.Handle("/mcp/", loggingHandler(streamableServer)) // Handle trailing slash

	// Legacy endpoints for backward compatibility
	mux.Handle("/v1/tool_code", loggingHandler(streamableServer))
	mux.Handle("/v1/tool-code", loggingHandler(streamableServer)) // Alias for python client

	// Root dashboard handler
	mux.HandleFunc("/", s.handleDashboard)

	// Metrics web interface
	mux.HandleFunc("/metrics", s.handleMetricsWeb)
	mux.HandleFunc("/api/metrics/current", s.handleMetricsAPI)

	// Comprehensive resources web interface
	mux.HandleFunc("/resources", s.handleResourcesWeb)
	mux.HandleFunc("/api/resources/current", s.handleResourcesAPI)
	mux.HandleFunc("/api/resources/history", s.handleResourcesHistoryAPI)

	// Server overview web interface
	mux.HandleFunc("/servers", s.handleServersWeb)
	mux.HandleFunc("/api/servers/status", s.handleServersStatusAPI)
	mux.HandleFunc("/api/servers", s.handleServersAPI)
	mux.HandleFunc("/api/servers/", s.handleServerConfigOrToolsAPI)

	// Server diagnostic chat interface
	mux.HandleFunc("/server/chat", s.handleServerChat)
	mux.HandleFunc("/api/chat/sessions", s.handleChatSession)
	mux.HandleFunc("/api/chat/sessions/", s.handleChatMessage)

	// Chat tool endpoints for OpenAI Function Calling
	mux.HandleFunc("/chat/read-config", s.handleChatReadConfig)
	mux.HandleFunc("/chat/write-config", s.handleChatWriteConfig)
	mux.HandleFunc("/chat/read-log", s.handleChatReadLog)
	mux.HandleFunc("/chat/read-github", s.handleChatReadGitHub)
	mux.HandleFunc("/chat/restart-server", s.handleChatRestartServer)
	mux.HandleFunc("/chat/call-tool", s.handleChatCallTool)
	mux.HandleFunc("/chat/get-server-status", s.handleChatGetServerStatus)
	mux.HandleFunc("/chat/test-server-tools", s.handleChatTestServerTools)
	mux.HandleFunc("/chat/list-all-servers", s.handleChatListAllServers)
	mux.HandleFunc("/chat/list-all-tools", s.handleChatListAllTools)
	mux.HandleFunc("/chat/context", s.handleChatContext)

	// Group management web interface
	mux.HandleFunc("/groups", s.handleGroupsWeb)
	mux.HandleFunc("/assignments", s.handleAssignmentWeb)
	mux.HandleFunc("/api/groups", s.handleGroupsAPI)
	mux.HandleFunc("/api/groups/", s.handleGroupsAPI)

	// Server assignment endpoints
	mux.HandleFunc("/api/toggle-group-servers", s.handleToggleGroupServers)
	mux.HandleFunc("/api/assign-server", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			s.handleAssignServer(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/unassign-server", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			s.handleUnassignServer(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/assignments", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			s.handleGetAssignments(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// MCP Inspector endpoints
	mux.HandleFunc("/api/inspector/start", s.handleInspectorStart)
	mux.HandleFunc("/api/inspector/stop", s.handleInspectorStop)
	mux.HandleFunc("/api/inspector/status", s.handleInspectorStatus)

	// Fast action endpoints for diagnostic agent
	mux.HandleFunc("/api/fast-action", s.handleFastAction)

	// File/path opening endpoint
	mux.HandleFunc("/api/open-path", s.handleOpenPath)

	// Agent API endpoints for Python MCP agent integration
	mux.HandleFunc("/api/v1/agent/servers", s.handleAgentListServers)
	mux.HandleFunc("/api/v1/agent/servers/", func(w http.ResponseWriter, r *http.Request) {
		// Route to either server details or server config based on path
		if strings.HasSuffix(r.URL.Path, "/logs") {
			s.handleAgentServerLogs(w, r)
		} else if strings.HasSuffix(r.URL.Path, "/config") {
			s.handleAgentServerConfig(w, r)
		} else {
			s.handleAgentServerDetails(w, r)
		}
	})
	mux.HandleFunc("/api/v1/agent/logs/main", s.handleAgentMainLogs)
	mux.HandleFunc("/api/v1/agent/registries/search", s.handleAgentSearchRegistries)
	mux.HandleFunc("/api/v1/agent/install", s.handleAgentInstallServer)

	s.mu.Lock()
	s.httpServer = &http.Server{
		Addr:              s.config.Listen,
		Handler:           mux,
		ReadHeaderTimeout: 60 * time.Second,  // Increased for better client compatibility
		ReadTimeout:       120 * time.Second, // Full request read timeout
		WriteTimeout:      120 * time.Second, // Response write timeout
		IdleTimeout:       180 * time.Second, // Keep-alive timeout for persistent connections
		MaxHeaderBytes:    1 << 20,           // 1MB max header size
		// Enable connection state tracking for better debugging
		ConnState: s.logConnectionState,
	}
	s.running = true
	s.mu.Unlock()

	s.logger.Info("Starting MCP HTTP server with enhanced client stability",
		zap.String("address", s.config.Listen),
		zap.Strings("endpoints", []string{"/mcp", "/mcp/", "/v1/tool_code", "/v1/tool-code"}),
		zap.Duration("read_timeout", 120*time.Second),
		zap.Duration("write_timeout", 120*time.Second),
		zap.Duration("idle_timeout", 180*time.Second),
		zap.String("features", "connection_tracking,graceful_shutdown,enhanced_logging"),
	)
	if err := s.httpServer.ListenAndServe(); err != http.ErrServerClosed {
		s.logger.Error("HTTP server error", zap.Error(err))
		s.mu.Lock()
		s.running = false
		s.mu.Unlock()
		s.updateStatus("Error", fmt.Sprintf("Server failed: %v", err))
		return err
	}

	s.logger.Info("HTTP server stopped")
	return nil
}

// responseWriter wraps http.ResponseWriter to capture the status code
type responseWriter struct {
	http.ResponseWriter
	statusCode    int
	headerWritten bool
}

func (rw *responseWriter) WriteHeader(code int) {
	if !rw.headerWritten {
		rw.statusCode = code
		rw.headerWritten = true
		rw.ResponseWriter.WriteHeader(code)
	}
}

// logConnectionState logs HTTP connection state changes for debugging client issues
func (s *Server) logConnectionState(conn net.Conn, state http.ConnState) {
	switch state {
	case http.StateNew:
		s.logger.Debug("New client connection established",
			zap.String("remote_addr", conn.RemoteAddr().String()),
			zap.String("state", "new"))
	case http.StateActive:
		s.logger.Debug("Client connection active",
			zap.String("remote_addr", conn.RemoteAddr().String()),
			zap.String("state", "active"))
	case http.StateIdle:
		s.logger.Debug("Client connection idle",
			zap.String("remote_addr", conn.RemoteAddr().String()),
			zap.String("state", "idle"))
	case http.StateHijacked:
		s.logger.Debug("Client connection hijacked (likely for upgrade)",
			zap.String("remote_addr", conn.RemoteAddr().String()),
			zap.String("state", "hijacked"))
	case http.StateClosed:
		s.logger.Debug("Client connection closed",
			zap.String("remote_addr", conn.RemoteAddr().String()),
			zap.String("state", "closed"))
	}
}

// SaveConfiguration saves the current configuration to the persistent config file
func (s *Server) SaveConfiguration() error {
	configPath := s.GetConfigPath()
	if configPath == "" {
		s.logger.Warn("Configuration file path is not available, cannot save configuration")
		return fmt.Errorf("configuration file path is not available")
	}

	s.logger.Debug("Saving configuration to file (merged)", zap.String("path", configPath))

	// Ensure we have the latest server list from the storage manager
	latestServers, err := s.storageManager.ListUpstreamServers()
	if err != nil {
		s.logger.Error("Failed to get latest server list from storage for saving", zap.Error(err))
		return err
	}
	s.config.Servers = latestServers

	// Sync groups from in-memory storage to config (structure only)
	s.syncGroupsToConfig()

	// Sync server-group assignments to config (fills GroupID on s.config.Servers)
	s.syncServerGroupAssignments()

	// Load existing JSON to preserve unknown fields
	existingBytes, readErr := os.ReadFile(configPath)
	if readErr != nil {
		// If file missing or unreadable, fall back to full save
		s.logger.Warn("Could not read existing config, falling back to full save", zap.Error(readErr))
		return config.SaveConfig(s.config, configPath)
	}

	var existing map[string]interface{}
	if err := json.Unmarshal(existingBytes, &existing); err != nil {
		s.logger.Warn("Existing config not JSON-decodable, falling back to full save", zap.Error(err))
		return config.SaveConfig(s.config, configPath)
	}

	// --- Merge Groups ---
	// Build lookup from existing groups by id or name to preserve extra fields
	existingGroups := map[string]map[string]interface{}{} // key by name
	if eg, ok := existing["groups"].([]interface{}); ok {
		for _, gi := range eg {
			if gm, ok := gi.(map[string]interface{}); ok {
				if name, ok := gm["name"].(string); ok && name != "" {
					existingGroups[name] = gm
				}
			}
		}
	}

	mergedGroups := make([]map[string]interface{}, 0, len(s.config.Groups))
	for _, g := range s.config.Groups {
		mg := map[string]interface{}{
			"id":      g.ID,
			"name":    g.Name,
			"color":   g.Color,
			"enabled": g.Enabled,
		}

		// Add description from config if present
		if g.Description != "" {
			mg["description"] = g.Description
		}

		// Add icon_emoji if present in config
		if g.Icon != "" {
			mg["icon_emoji"] = g.Icon
			s.logger.Debug("[ICON PRESERVE] Using icon from config",
				zap.String("group", g.Name),
				zap.String("icon", g.Icon))
		}

		// Preserve fields from previous config if current config doesn't have them
		if prev, ok := existingGroups[g.Name]; ok {
			// Preserve description if not set in config but exists in prev
			if g.Description == "" {
				if desc, ok := prev["description"].(string); ok && desc != "" {
					mg["description"] = desc
					s.logger.Debug("[ICON PRESERVE] Preserved description from previous config",
						zap.String("group", g.Name),
						zap.String("description", desc))
				}
			}

			// CRITICAL FIX: Preserve icon_emoji if not set in config but exists in prev
			if g.Icon == "" {
				if iconEmoji, ok := prev["icon_emoji"].(string); ok && iconEmoji != "" {
					mg["icon_emoji"] = iconEmoji
					s.logger.Warn("[ICON PRESERVE] Restored icon_emoji from previous config!",
						zap.String("group", g.Name),
						zap.String("icon_emoji", iconEmoji))
				} else {
					s.logger.Warn("[ICON PRESERVE] No icon found in config or previous data!",
						zap.String("group", g.Name))
				}
			}
		} else if g.Icon == "" {
			// No previous config and no icon in current config
			s.logger.Warn("[ICON PRESERVE] New group without icon",
				zap.String("group", g.Name))
		}

		mergedGroups = append(mergedGroups, mg)
	}
	existing["groups"] = mergedGroups

	// --- Merge Servers ---
	// Create lookup of existing servers by name
	existingServersByName := map[string]map[string]interface{}{}
	existingOrder := []string{}
	if es, ok := existing["mcpServers"].([]interface{}); ok {
		for _, si := range es {
			if sm, ok := si.(map[string]interface{}); ok {
				if name, ok := sm["name"].(string); ok && name != "" {
					existingServersByName[name] = sm
					existingOrder = append(existingOrder, name)
				}
			}
		}
	}

	// Lookups for latest server configs by name
	latestByName := map[string]*config.ServerConfig{}
	for _, sc := range s.config.Servers {
		latestByName[sc.Name] = sc
	}

	// Build set of servers whose group assignment is being updated now
	updatedServers := map[string]bool{}
	assignmentsMutex.RLock()
	for srvName := range serverGroupAssignments {
		updatedServers[srvName] = true
	}
	assignmentsMutex.RUnlock()

	// Build group name -> id lookup
	groupNameToID := map[string]int{}
	groupsMutex.RLock()
	for name, g := range groups {
		if g != nil {
			groupNameToID[name] = g.ID
		}
	}
	groupsMutex.RUnlock()

	// Helper to compute final group fields for a given server name, starting from current values
	computeGroupFields := func(name string, currentID int, currentName string) (int, string) {
		if updatedServers[name] {
			assignmentsMutex.RLock()
			if gname, ok := serverGroupAssignments[name]; ok {
				if gid, ok2 := groupNameToID[gname]; ok2 {
					assignmentsMutex.RUnlock()
					return gid, gname
				}
				assignmentsMutex.RUnlock()
				return currentID, gname
			}
			assignmentsMutex.RUnlock()
		}
		return currentID, currentName
	}

	// Start from existing entries to preserve unknown fields and ordering
	mergedServers := make([]map[string]interface{}, 0, max(len(existingOrder), len(latestByName)))
	seen := map[string]bool{}
	for _, name := range existingOrder {
		existingEntry := existingServersByName[name]
		// Start with a shallow copy of existing
		m := map[string]interface{}{}
		for k, v := range existingEntry {
			m[k] = v
		}
		// Overlay known fields from latest if available
		if sc, ok := latestByName[name]; ok {
			m["name"] = sc.Name
			m["description"] = sc.Description
			m["url"] = sc.URL
			m["protocol"] = sc.Protocol
			m["command"] = sc.Command
			m["args"] = sc.Args
			m["working_dir"] = sc.WorkingDir
			m["env"] = sc.Env
			m["headers"] = sc.Headers
			m["oauth"] = sc.OAuth
			m["repository_url"] = sc.RepositoryURL
			m["enabled"] = sc.Enabled
			m["quarantined"] = sc.Quarantined
			m["created"] = sc.Created
			m["updated"] = sc.Updated
			m["isolation"] = sc.Isolation
		}
		// Compute final group fields using current values as base
		prevID := 0
		if v, ok := m["group_id"].(float64); ok { prevID = int(v) } else if vi, ok := m["group_id"].(int); ok { prevID = vi }
		finalID, _ := computeGroupFields(name, prevID, "")
		// Safety: never downgrade a non-zero existing group_id to 0 during merge
		if finalID == 0 && prevID > 0 {
			finalID = prevID
		}
		m["group_id"] = finalID
		// Remove legacy group_name
		delete(m, "group_name")

		mergedServers = append(mergedServers, m)
		seen[name] = true
	}

	// Append any new servers not in existing
	for name, sc := range latestByName {
		if seen[name] { continue }
		m := map[string]interface{}{
			"name":           sc.Name,
			"description":    sc.Description,
			"url":            sc.URL,
			"protocol":       sc.Protocol,
			"command":        sc.Command,
			"args":           sc.Args,
			"working_dir":    sc.WorkingDir,
			"env":            sc.Env,
			"headers":        sc.Headers,
			"oauth":          sc.OAuth,
			"repository_url": sc.RepositoryURL,
			"enabled":        sc.Enabled,
			"quarantined":    sc.Quarantined,
			"created":        sc.Created,
			"updated":        sc.Updated,
			"isolation":      sc.Isolation,
		}
		// Group fields for new server: compute from assignment (if any) else 0/""
		finalID, _ := computeGroupFields(name, sc.GroupID, "")
		m["group_id"] = finalID
		// Remove legacy group_name
		// (not set)
		mergedServers = append(mergedServers, m)
	}

	existing["mcpServers"] = mergedServers

	// Preserve all other top-level fields in existing as-is (they already are in 'existing')

	// Write merged JSON back to file
	out, err := json.MarshalIndent(existing, "", "  ")
	if err != nil {
		s.logger.Error("Failed to marshal merged configuration", zap.Error(err))
		return err
	}
	// Debug: log resulting group assignments
	if ms, ok := existing["mcpServers"].([]map[string]interface{}); ok {
		for _, e := range ms {
			name, _ := e["name"].(string)
			gid := 0
			if v, ok := e["group_id"].(float64); ok { gid = int(v) } else if vi, ok := e["group_id"].(int); ok { gid = vi }
			s.logger.Debug("Merged server group assignment", zap.String("server", name), zap.Int("group_id", gid))
		}
	}
	if err := os.WriteFile(configPath, out, 0600); err != nil {
		s.logger.Error("Failed to write merged configuration", zap.Error(err))
		return err
	}

	s.logger.Info("Configuration saved with merge strategy", zap.Int("servers", len(mergedServers)), zap.Int("groups", len(mergedGroups)))
	return nil
}

// syncGroupsToConfig syncs groups from in-memory storage to config
func (s *Server) syncGroupsToConfig() {
	groupsMutex.RLock()
	defer groupsMutex.RUnlock()

	s.logger.Debug("[GROUPS DEBUG] syncGroupsToConfig called",
		zap.Int("in_memory_groups_count", len(groups)))

	// Log all in-memory groups
	for name, group := range groups {
		s.logger.Debug("[GROUPS DEBUG] In-memory group found",
			zap.String("name", name),
			zap.String("group_name", group.Name),
			zap.Int("id", group.ID),
			zap.String("icon", group.Icon),
			zap.String("color", group.Color))
	}

	// Convert in-memory groups to config format
	configGroups := make([]config.GroupConfig, 0, len(groups))
	for _, group := range groups {
		configGroups = append(configGroups, config.GroupConfig{
			ID:          group.ID,
			Name:        group.Name,
			Description: group.Description,
			Icon:        group.Icon,
			Color:       group.Color,
			Enabled:     true, // Groups are enabled by default
		})
		s.logger.Debug("[GROUPS DEBUG] Converting group to config format",
			zap.String("name", group.Name),
			zap.String("description", group.Description),
			zap.String("icon", group.Icon),
			zap.String("color", group.Color))
	}

	s.logger.Debug("[GROUPS DEBUG] About to overwrite config.Groups",
		zap.Int("old_config_groups", len(s.config.Groups)),
		zap.Int("new_config_groups", len(configGroups)))

	s.config.Groups = configGroups
	s.logger.Debug("Synced groups to config", zap.Int("count", len(configGroups)))
}

// syncServerGroupAssignments syncs server-group assignments to config
func (s *Server) syncServerGroupAssignments() {
	assignmentsMutex.RLock()
	defer assignmentsMutex.RUnlock()

	// Update each server's group_id field in the config
	for i := range s.config.Servers {
		serverName := s.config.Servers[i].Name
		if groupName, exists := serverGroupAssignments[serverName]; exists {
			// Find group ID by name
			groupsMutex.RLock()
			if group, groupExists := groups[groupName]; groupExists {
				s.config.Servers[i].GroupID = group.ID
				s.config.Servers[i].GroupName = "" // Clear legacy field
			}
			groupsMutex.RUnlock()
		}
		// REMOVED ELSE CLAUSE - preserve existing group_id if not in assignments map
	}

	s.logger.Debug("Synced server-group assignments to config",
		zap.Int("assignments", len(serverGroupAssignments)))
}

// initGroupsFromConfig initializes in-memory groups from config
func (s *Server) initGroupsFromConfig() {
	groupsMutex.Lock()
	defer groupsMutex.Unlock()

	s.logger.Debug("[GROUPS DEBUG] initGroupsFromConfig called",
		zap.Int("config_groups_count", len(s.config.Groups)))

	// Log all config groups
	for i, configGroup := range s.config.Groups {
		s.logger.Debug("[GROUPS DEBUG] Config group found",
			zap.Int("index", i),
			zap.String("name", configGroup.Name),
			zap.String("color", configGroup.Color),
			zap.Bool("enabled", configGroup.Enabled))
	}

	// Clear existing groups
	groups = make(map[string]*Group)

	// Load groups from config
	loadedCount := 0
	for _, configGroup := range s.config.Groups {
		if configGroup.Enabled {
			groups[configGroup.Name] = &Group{
				ID:          configGroup.ID,
				Name:        configGroup.Name,
				Description: configGroup.Description,
				Icon:        configGroup.Icon,
				Color:       configGroup.Color,
			}
			loadedCount++
			s.logger.Debug("[GROUPS DEBUG] Loaded group into memory",
				zap.Int("id", configGroup.ID),
				zap.String("name", configGroup.Name),
				zap.String("description", configGroup.Description),
				zap.String("icon", configGroup.Icon),
				zap.String("color", configGroup.Color))
		}
	}

	s.logger.Debug("[GROUPS DEBUG] Groups loaded from config",
		zap.Int("loaded_count", loadedCount),
		zap.Int("total_in_memory", len(groups)))

	// Ensure default groups exist if no groups in config
	if len(groups) == 0 {
		s.logger.Debug("[GROUPS DEBUG] No groups loaded, creating defaults")
		groups["AWS Services"] = &Group{Name: "AWS Services", Color: "#ff9900"}
		groups["Development"] = &Group{Name: "Development", Color: "#28a745"}
		groups["Production"] = &Group{Name: "Production", Color: "#dc3545"}
		s.logger.Debug("[GROUPS DEBUG] Created default groups", zap.Int("default_count", len(groups)))
	}

	s.logger.Debug("Initialized groups from config", zap.Int("count", len(groups)))
}

// initServerGroupAssignments initializes server-group assignments from config
func (s *Server) initServerGroupAssignments() {
	assignmentsMutex.Lock()
	defer assignmentsMutex.Unlock()

	// Clear existing assignments
	serverGroupAssignments = make(map[string]string)

	// Detect if any server uses group_id (>0). If yes, we consider IDs authoritative and ignore legacy group_name fields.
	idMode := false
	for _, srv := range s.config.Servers {
		if srv.GroupID > 0 {
			idMode = true
			break
		}
	}

	// Load assignments from config
	for _, server := range s.config.Servers {
		// ID mode: use IDs only; group_id=0 means unassigned
		if idMode {
			if server.GroupID > 0 {
				groupsMutex.RLock()
				for name, g := range groups {
					if g != nil && g.ID == server.GroupID {
						serverGroupAssignments[server.Name] = name
						break
					}
				}
				groupsMutex.RUnlock()
			}
			continue
		}

		// Legacy mode (no IDs anywhere): fall back to GroupName
		if server.GroupName != "" {
			serverGroupAssignments[server.Name] = server.GroupName
		}
	}

	s.logger.Info("Initialized server-group assignments from config",
		zap.Int("assignments", len(serverGroupAssignments)),
		zap.Bool("id_mode", idMode))
}

// migrateLegacyGroupNamesToIDs converts any server with only group_name set into group_id in-memory and saves the config
func (s *Server) migrateLegacyGroupNamesToIDs() {
	changed := false
	// Build lookup of group name -> id
	groupsMutex.RLock()
	nameToID := make(map[string]int)
	for name, g := range groups {
		if g != nil && g.ID > 0 {
			nameToID[name] = g.ID
		}
	}
	groupsMutex.RUnlock()

	for _, srv := range s.config.Servers {
		if srv.GroupID == 0 && srv.GroupName != "" {
			if id, ok := nameToID[srv.GroupName]; ok {
				srv.GroupID = id
				srv.GroupName = ""
				changed = true
			}
		}
	}
	if changed {
		_ = s.SaveConfiguration()
		s.logger.Info("Migrated legacy group_name to group_id in config")
	}
}

// getGroups returns a copy of all groups (thread-safe)
func (s *Server) getGroups() map[string]*Group {
	groupsMutex.RLock()
	defer groupsMutex.RUnlock()
	
	result := make(map[string]*Group)
	for k, v := range groups {
		result[k] = v
	}
	return result
}

// setGroup sets a group (thread-safe)
func (s *Server) setGroup(name string, group *Group) {
	groupsMutex.Lock()
	defer groupsMutex.Unlock()
	groups[name] = group
}

// deleteGroup deletes a group (thread-safe)
func (s *Server) deleteGroup(name string) {
	groupsMutex.Lock()
	defer groupsMutex.Unlock()
	delete(groups, name)
}

// ReloadConfiguration reloads the configuration from disk
func (s *Server) ReloadConfiguration() error {
	s.logger.Info("Reloading configuration from disk")

	// Store old config for comparison
	oldServerCount := len(s.config.Servers)

	// Preserve current server-group assignments before config reload
	assignmentsMutex.RLock()
	savedAssignments := make(map[string]string)
	for serverName, groupName := range serverGroupAssignments {
		savedAssignments[serverName] = groupName
	}
	assignmentsMutex.RUnlock()

	// Load configuration from file
	s.mu.RLock()
	dataDir := s.config.DataDir
	s.mu.RUnlock()

	configPath := config.GetConfigPath(dataDir)
	newConfig, err := config.LoadFromFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to reload config: %w", err)
	}

	// Update internal config with write lock to prevent race conditions
	s.mu.Lock()
	s.config = newConfig
	s.mu.Unlock()

	// Migrate legacy names to IDs after reload
	s.migrateLegacyGroupNamesToIDs()

	// NOTE: Do not restore preserved assignments here. We want the file to be authoritative
	// on reload (including clearing assignments where group_id == 0). The assignments map
	// will be rebuilt from s.config by loadConfiguredServers -> initServerGroupAssignments.
	//
	// Previously we restored savedAssignments here, which could mask file changes.

	s.logger.Debug("Preserved assignments snapshot (not restored)",
		zap.Int("preserved_assignments", len(savedAssignments)))

	// Reload configured servers (this is where the comprehensive sync happens)
	s.logger.Debug("About to call loadConfiguredServers")
	if err := s.loadConfiguredServers(); err != nil {
		s.logger.Error("loadConfiguredServers failed", zap.Error(err))
		return fmt.Errorf("failed to reload servers: %w", err)
	}
	s.logger.Debug("loadConfiguredServers completed successfully")

	// Trigger immediate reconnection for servers that were disconnected during config reload
	s.logger.Debug("Starting goroutine for immediate reconnection after config reload")
	go func() {
		s.mu.RLock()
		ctx := s.appCtx // Use application context instead of server context
		s.mu.RUnlock()

		s.logger.Debug("Inside reconnection goroutine", zap.Bool("ctx_is_nil", ctx == nil))
		if ctx == nil {
			s.logger.Error("Application context is nil, cannot trigger reconnection")
			return
		}

		s.logger.Info("Triggering immediate reconnection after config reload")

		// Connect all servers that should be connected
		connectCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		if err := s.upstreamManager.ConnectAll(connectCtx); err != nil {
			s.logger.Warn("Some servers failed to reconnect after config reload", zap.Error(err))
		}

		// NOTE: Removed automatic tool re-indexing after config reload
		// Tools will be loaded based on:
		// 1. StartOnBoot flag for individual servers
		// 2. EnableLazyLoading global setting
		// 3. Manual reload requests from tray UI
		// 4. Server-specific health checks (if configured)
		s.logger.Info("Config reload complete - tool loading will follow lazy loading policy")
	}()

	s.logger.Info("Configuration reload completed",
		zap.String("path", configPath),
		zap.Int("old_server_count", oldServerCount),
		zap.Int("new_server_count", len(newConfig.Servers)),
		zap.Int("server_delta", len(newConfig.Servers)-oldServerCount))

	return nil
}

// OnUpstreamServerChange should be called when upstream servers are modified
func (s *Server) OnUpstreamServerChange() {
	// NOTE: Removed automatic tool re-indexing on server changes
	// Tools will be loaded based on:
	// 1. StartOnBoot flag for individual servers
	// 2. EnableLazyLoading global setting
	// 3. Manual reload requests from tray UI
	// 4. Server-specific health checks (if configured)
	// This prevents excessive ListTools calls when servers are modified
	s.logger.Info("Upstream server configuration changed")

	go func() {
		// Clean up any orphaned tools in index that are no longer from active servers
		// This handles edge cases where servers were removed abruptly
		s.cleanupOrphanedIndexEntries()
	}()

	// Update status
	s.updateStatus(s.status.Phase, "Upstream servers updated")
}

// cleanupOrphanedIndexEntries removes index entries for servers that are no longer active
func (s *Server) cleanupOrphanedIndexEntries() {
	s.logger.Debug("Checking for orphaned index entries")

	// Get list of active server names
	activeServers := s.upstreamManager.GetAllServerNames()
	activeServerMap := make(map[string]bool)
	for _, serverName := range activeServers {
		activeServerMap[serverName] = true
	}

	// For now, we rely on the batch indexing to effectively replace all content
	// In a more sophisticated implementation, we could:
	// 1. Query the index for all unique server names
	// 2. Compare against active servers
	// 3. Remove orphaned entries
	// This is left as a future enhancement since batch indexing handles most cases

	s.logger.Debug("Orphaned index cleanup completed",
		zap.Int("active_servers", len(activeServers)))
}

// GetConfigPath returns the path to the configuration file for file watching
func (s *Server) GetConfigPath() string {
	// If we have the actual config path that was used, return that
	if s.configPath != "" {
		return s.configPath
	}
	// Otherwise fall back to the default path
	return config.GetConfigPath(s.config.DataDir)
}

// GetLogDir returns the log directory path for tray UI
func (s *Server) GetLogDir() string {
	if s.config.Logging != nil && s.config.Logging.LogDir != "" {
		return s.config.Logging.LogDir
	}
	// Return OS-specific default log directory if not configured
	if defaultLogDir, err := logs.GetLogDir(); err == nil {
		return defaultLogDir
	}
	// Last resort fallback to data directory
	return s.config.DataDir
}

// GetGitHubURL returns the configured GitHub URL
func (s *Server) GetGitHubURL() string {
	if s.config.GitHubURL != "" {
		return s.config.GitHubURL
	}
	// Fallback to default GitHub URL if not configured
	return "https://github.com/smart-mcp-proxy/mcpproxy-go"
}

// GetLLMConfig returns the LLM configuration for the AI Diagnostic Agent
func (s *Server) GetLLMConfig() *config.LLMConfig {
	if s.config == nil {
		return nil
	}
	return s.config.LLM
}

// --- Startup Script Management (exposed for tray/MCP) ---

// StartStartupScript starts the configured startup script if enabled
func (s *Server) StartStartupScript(ctx context.Context) error {
    if s.startupManager == nil {
        return fmt.Errorf("startup manager not initialized")
    }
    return s.startupManager.Start(ctx)
}

// StopStartupScript stops the startup script and child processes
func (s *Server) StopStartupScript() error {
    if s.startupManager == nil {
        return nil
    }
    return s.startupManager.Stop()
}

// RestartStartupScript restarts the startup script
func (s *Server) RestartStartupScript(ctx context.Context) error {
    if s.startupManager == nil {
        return fmt.Errorf("startup manager not initialized")
    }
    return s.startupManager.Restart(ctx)
}

// GetStartupScriptStatus returns status information about the startup script
func (s *Server) GetStartupScriptStatus() map[string]interface{} {
    if s.startupManager == nil {
        return map[string]interface{}{"enabled": false, "running": false}
    }
    return s.startupManager.Status()
}

// UpdateStartupScript updates startup script configuration and persists it
func (s *Server) UpdateStartupScript(cfg *config.StartupScriptConfig) error {
    if cfg == nil {
        return fmt.Errorf("nil startup script config")
    }
    // Basic validation
    if err := startup.ValidateConfig(cfg); err != nil {
        return err
    }
    // Update in-memory
    s.config.StartupScript = cfg
    if s.startupManager != nil {
        s.startupManager.UpdateConfig(cfg)
    } else {
        s.startupManager = startup.NewManager(cfg, s.logger.Sugar())
    }
    // Persist to disk
    if err := s.SaveConfiguration(); err != nil {
        return err
    }
    return nil
}

// cleanupOrphanedDockerContainers finds and removes Docker containers left over from previous crashes
func (s *Server) cleanupOrphanedDockerContainers(ctx context.Context) {
	s.logger.Info("Checking for orphaned Docker containers from previous runs...")

	// Search for containers with mcpproxy.server label
	listCmd := exec.CommandContext(ctx, "docker", "ps", "-a", "--filter", "label=mcpproxy.server", "--format", "{{.ID}}\t{{.Names}}\t{{.Label \"mcpproxy.server\"}}")
	output, err := listCmd.Output()
	if err != nil {
		s.logger.Warn("Failed to list Docker containers for orphan cleanup", zap.Error(err))
		return
	}

	orphanedContainers := strings.TrimSpace(string(output))
	if orphanedContainers == "" {
		s.logger.Debug("No orphaned Docker containers found")
		return
	}

	// Parse container list
	lines := strings.Split(orphanedContainers, "\n")
	s.logger.Info("Found orphaned Docker containers from previous runs",
		zap.Int("count", len(lines)))

	// Kill each orphaned container
	cleanedCount := 0
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 3)
		if len(parts) < 3 {
			continue
		}

		containerID := parts[0]
		containerName := parts[1]
		serverName := parts[2]

		s.logger.Info("Cleaning up orphaned Docker container",
			zap.String("container_id", containerID),
			zap.String("container_name", containerName),
			zap.String("server", serverName))

		// Try graceful stop first
		stopCmd := exec.CommandContext(ctx, "docker", "stop", containerID)
		if err := stopCmd.Run(); err != nil {
			// Force kill if stop fails
			s.logger.Debug("Graceful stop failed, force killing container",
				zap.String("container_id", containerID),
				zap.Error(err))
			killCmd := exec.CommandContext(ctx, "docker", "kill", containerID)
			if err := killCmd.Run(); err != nil {
				s.logger.Warn("Failed to kill orphaned container",
					zap.String("container_id", containerID),
					zap.Error(err))
				continue
			}
		}

		cleanedCount++
		s.logger.Info("Successfully cleaned up orphaned container",
			zap.String("container_id", containerID),
			zap.String("server", serverName))
	}

	if cleanedCount > 0 {
		s.logger.Info("Orphaned Docker container cleanup completed",
			zap.Int("cleaned", cleanedCount),
			zap.Int("total_found", len(lines)))
	}
}

// handleServersAPI returns a JSON list of all servers for the chat page sidebar
func (s *Server) handleServersAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	servers, err := s.GetAllServers()
	if err != nil {
		s.logger.Error("Failed to get all servers for API", zap.Error(err))
		http.Error(w, fmt.Sprintf("Failed to get servers: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"servers": servers,
	}); err != nil {
		s.logger.Error("Failed to encode servers JSON", zap.Error(err))
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// handleServerConfigOrToolsAPI handles both GET /api/servers/{name}/tools and PUT /api/servers/{name}/config
func (s *Server) handleServerConfigOrToolsAPI(w http.ResponseWriter, r *http.Request) {
	// Extract server name from URL path
	path := r.URL.Path
	prefix := "/api/servers/"
	if !strings.HasPrefix(path, prefix) {
		http.Error(w, "Invalid URL path", http.StatusBadRequest)
		return
	}

	remainder := strings.TrimPrefix(path, prefix)
	parts := strings.Split(remainder, "/")
	if len(parts) < 2 {
		http.Error(w, "Invalid URL path", http.StatusBadRequest)
		return
	}

	serverName := parts[0]
	endpoint := parts[1]

	if serverName == "" {
		http.Error(w, "Server name is required", http.StatusBadRequest)
		return
	}

	// Route based on method and endpoint
	if endpoint == "tools" && r.Method == http.MethodGet {
		s.handleGetServerTools(w, r, serverName)
	} else if endpoint == "config" && r.Method == http.MethodPut {
		s.handleUpdateServerConfig(w, r, serverName)
	} else {
		http.Error(w, "Method not allowed or invalid endpoint", http.StatusMethodNotAllowed)
	}
}

// handleGetServerTools returns a JSON list of tools for a specific server
func (s *Server) handleGetServerTools(w http.ResponseWriter, r *http.Request, serverName string) {
	s.logger.Debug("Getting tools for server", zap.String("server", serverName))

	tools, err := s.GetServerTools(serverName)
	if err != nil {
		s.logger.Error("Failed to get server tools for API",
			zap.String("server", serverName),
			zap.Error(err))
		http.Error(w, fmt.Sprintf("Failed to get tools: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"tools": tools,
	}); err != nil {
		s.logger.Error("Failed to encode tools JSON", zap.Error(err))
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// handleUpdateServerConfig updates a server's configuration
func (s *Server) handleUpdateServerConfig(w http.ResponseWriter, r *http.Request, serverName string) {
	s.logger.Info("Updating server configuration", zap.String("server", serverName))

	// Parse request body
	var updateData struct {
		Name          string                 `json:"name"`
		Enabled       bool                   `json:"enabled"`
		Protocol      string                 `json:"protocol"`
		Command       string                 `json:"command"`
		WorkingDir    string                 `json:"working_dir"`
		URL           string                 `json:"url"`
		RepositoryURL string                 `json:"repository_url"`
		Quarantined   bool                   `json:"quarantined"`
		Args          []string               `json:"args"`
		Env           map[string]string      `json:"env"`
	}

	if err := json.NewDecoder(r.Body).Decode(&updateData); err != nil {
		s.logger.Error("Failed to parse update request", zap.Error(err))
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	// Get existing server configuration
	servers, err := s.storageManager.ListUpstreamServers()
	if err != nil {
		s.logger.Error("Failed to get servers from storage", zap.Error(err))
		http.Error(w, "Failed to load server configuration", http.StatusInternalServerError)
		return
	}

	var serverConfig *config.ServerConfig
	for _, srv := range servers {
		if srv.Name == serverName {
			serverConfig = srv
			break
		}
	}

	if serverConfig == nil {
		http.Error(w, "Server not found", http.StatusNotFound)
		return
	}

	// Update fields
	serverConfig.Enabled = updateData.Enabled
	serverConfig.Protocol = updateData.Protocol
	serverConfig.Command = updateData.Command
	serverConfig.WorkingDir = updateData.WorkingDir
	serverConfig.URL = updateData.URL
	serverConfig.RepositoryURL = updateData.RepositoryURL
	serverConfig.Quarantined = updateData.Quarantined
	serverConfig.Args = updateData.Args
	serverConfig.Env = updateData.Env
	serverConfig.Updated = time.Now()

	// Save to storage
	if err := s.storageManager.SaveUpstreamServer(serverConfig); err != nil {
		s.logger.Error("Failed to save server configuration", zap.Error(err))
		http.Error(w, "Failed to save configuration", http.StatusInternalServerError)
		return
	}

	// Save to config file
	if err := s.SaveConfiguration(); err != nil {
		s.logger.Error("Failed to save configuration file", zap.Error(err))
		http.Error(w, "Failed to save configuration file", http.StatusInternalServerError)
		return
	}

	// Publish config change event
	action := "updated"
	s.eventBus.Publish(events.Event{
		Type:       events.EventConfigChange,
		ServerName: serverName,
		Data: events.ConfigChangeData{
			Action: action,
		},
	})

	// Return success
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Configuration updated successfully",
		"server":  serverConfig,
	})

	s.logger.Info("Server configuration updated successfully", zap.String("server", serverName))
}

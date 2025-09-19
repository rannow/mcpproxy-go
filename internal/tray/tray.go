//go:build !nogui && !headless && !linux

package tray

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"fyne.io/systray"
	"github.com/fsnotify/fsnotify"
	"github.com/inconshreveable/go-update"
	"go.uber.org/zap"
	"golang.org/x/mod/semver"

	"mcpproxy-go/internal/config"
	"mcpproxy-go/internal/server"
	// "mcpproxy-go/internal/upstream/cli" // replaced by in-process OAuth
)

const (
	repo      = "smart-mcp-proxy/mcpproxy-go" // Actual repository
	osDarwin  = "darwin"
	osWindows = "windows"
	trueStr   = "true"
)

//go:embed icon-mono-44.png
var iconData []byte

// ServerGroup represents a custom group of servers with color coding
type ServerGroup struct {
	Name        string   `json:"name"`
	Color       string   `json:"color"`       // Color emoji or hex code
	ColorEmoji  string   `json:"color_emoji"` // Color emoji for display
	Description string   `json:"description"`
	ServerNames []string `json:"server_names"`
	Enabled     bool     `json:"enabled"`
}

// Predefined color options for groups
var GroupColors = []struct {
	Emoji string
	Name  string
	Code  string
}{
	{"🔴", "Red", "#FF0000"},
	{"🟠", "Orange", "#FFA500"},
	{"🟡", "Yellow", "#FFFF00"},
	{"🟢", "Green", "#00FF00"},
	{"🔵", "Blue", "#0000FF"},
	{"🟣", "Purple", "#800080"},
	{"🟤", "Brown", "#A52A2A"},
	{"⚫", "Black", "#000000"},
	{"⚪", "White", "#FFFFFF"},
	{"🔸", "Diamond Orange", "#FFB366"},
	{"🔹", "Diamond Blue", "#66B2FF"},
	{"⭐", "Star", "#FFD700"},
}

// GitHubRelease represents a GitHub release
type GitHubRelease struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

// ServerInterface defines the interface for server control
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

	// Server management methods for tray menu
	EnableServer(serverName string, enabled bool) error
	QuarantineServer(serverName string, quarantined bool) error
	GetAllServers() ([]map[string]interface{}, error)
	GetServerTools(serverName string) ([]map[string]interface{}, error)

	// Config management for file watching
	ReloadConfiguration() error
	GetConfigPath() string
	GetLogDir() string
	GetGitHubURL() string

	// OAuth control
	TriggerOAuthLogin(serverName string) error
}

// App represents the system tray application
type App struct {
	server   ServerInterface
	logger   *zap.SugaredLogger
	version  string
	shutdown func()

	// Menu items for dynamic updates
	statusItem          *systray.MenuItem
	startStopItem       *systray.MenuItem
	serverCountItem     *systray.MenuItem

	// Status-based server menus
	connectedServersMenu    *systray.MenuItem
	disconnectedServersMenu *systray.MenuItem
	stoppedServersMenu      *systray.MenuItem
	disabledServersMenu     *systray.MenuItem
	quarantineMenu          *systray.MenuItem

	// Legacy upstream menu (for backward compatibility if needed)
	// upstreamServersMenu *systray.MenuItem - REMOVED

	// Managers for proper synchronization
	stateManager *ServerStateManager
	menuManager  *MenuManager
	syncManager  *SynchronizationManager
	diagnosticAgent *DiagnosticAgent

	// Autostart manager
	autostartManager *AutostartManager
	autostartItem    *systray.MenuItem

	// Config file watching
	configWatcher *fsnotify.Watcher
	configPath    string

	// Context for background operations
	ctx    context.Context
	cancel context.CancelFunc

	// Legacy fields for compatibility during transition
	lastRunningState bool // Track last known server running state

	// Menu tracking fields for dynamic updates
	forceRefresh      bool                         // Force menu refresh flag
	menusInitialized  bool                         // Track if menus have been initialized
	coreMenusReady    bool                         // Track if core menu items are ready
	lastServerList    []string                     // Track last known server list for change detection
	serverMenus       map[string]*systray.MenuItem // Track server menu items
	serverActionMenus map[string]*systray.MenuItem // Track server action menu items

	// Server count state
	serverCountFromConfig int  // Static server count from config
	serverCountInitialized bool // Flag to prevent overwriting the static count

	// Quarantine menu tracking fields
	lastQuarantineList    []string                     // Track last known quarantine list for change detection
	quarantineServerMenus map[string]*systray.MenuItem // Track quarantine server menu items

	// Group management fields
	groupManagementMenu *systray.MenuItem            // Group management menu
	groupedServersMenu  *systray.MenuItem            // Menu showing servers organized by groups
	serverGroups        map[string]*ServerGroup      // Custom server groups
	groupMenuItems      map[string]*systray.MenuItem // Group menu items
	groupServerMenus    map[string]*systray.MenuItem // Group-based server list menus
}

// New creates a new tray application
func New(server ServerInterface, logger *zap.SugaredLogger, version string, shutdown func()) *App {
	app := &App{
		server:   server,
		logger:   logger,
		version:  version,
		shutdown: shutdown,
	}

	// Initialize managers (will be fully set up in onReady)
	app.stateManager = NewServerStateManager(server, logger)

	// Initialize autostart manager
	if autostartManager, err := NewAutostartManager(); err != nil {
		logger.Warn("Failed to initialize autostart manager", zap.Error(err))
	} else {
		app.autostartManager = autostartManager
	}

	// Initialize menu tracking maps
	app.serverMenus = make(map[string]*systray.MenuItem)
	app.serverActionMenus = make(map[string]*systray.MenuItem)
	app.quarantineServerMenus = make(map[string]*systray.MenuItem)
	app.lastServerList = []string{}
	app.lastQuarantineList = []string{}

	// Initialize group management
	app.serverGroups = make(map[string]*ServerGroup)
	app.groupMenuItems = make(map[string]*systray.MenuItem)
	app.groupServerMenus = make(map[string]*systray.MenuItem)

	return app
}

// Run starts the system tray application
func (a *App) Run(ctx context.Context) error {
	a.logger.Info("Starting system tray application")
	a.ctx, a.cancel = context.WithCancel(ctx)
	defer a.cancel()

	// Initialize config file watcher
	if err := a.initConfigWatcher(); err != nil {
		a.logger.Warn("Failed to initialize config file watcher", zap.Error(err))
	}

	// Start background auto-update checker (daily)
	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				a.checkForUpdates()
			case <-ctx.Done():
				return
			}
		}
	}()

	// Start background status updater (every 5 seconds for more responsive UI)
	// Wait for menu initialization to complete before starting updates
	go func() {
		a.logger.Debug("Waiting for core menu items to be initialized...")
		// Wait for menu items to be initialized using the flag
		for !a.coreMenusReady {
			select {
			case <-ctx.Done():
				return
			default:
				time.Sleep(100 * time.Millisecond) // Check every 100ms
			}
		}

		a.logger.Debug("Core menu items ready, starting status updater")
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				a.updateStatus()
			case <-ctx.Done():
				return
			}
		}
	}()

	// Start config file watcher
	if a.configWatcher != nil {
		go a.watchConfigFile()
	}

	// Listen for real-time status updates
	if a.server != nil {
		go func() {
			a.logger.Debug("Waiting for core menu items before processing real-time status updates...")
			// Wait for menu items to be initialized using the flag
			for !a.coreMenusReady {
				select {
				case <-ctx.Done():
					return
				default:
					time.Sleep(100 * time.Millisecond) // Check every 100ms
				}
			}

			a.logger.Debug("Core menu items ready, starting real-time status updates")
			statusCh := a.server.StatusChannel()
			for {
				select {
				case status := <-statusCh:
					a.updateStatusFromData(status)
				case <-ctx.Done():
					return
				}
			}
		}()
	}

	// Monitor context cancellation and quit systray when needed
	go func() {
		<-ctx.Done()
		a.logger.Info("Context cancelled, quitting systray")
		a.cleanup()
		systray.Quit()
	}()

	// Start systray - this is a blocking call that must run on main thread
	systray.Run(a.onReady, a.onExit)

	return ctx.Err()
}

// initConfigWatcher initializes the config file watcher
func (a *App) initConfigWatcher() error {
	if a.server == nil {
		return fmt.Errorf("server interface not available")
	}

	configPath := a.server.GetConfigPath()
	if configPath == "" {
		return fmt.Errorf("config path not available")
	}

	a.configPath = configPath

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create file watcher: %w", err)
	}

	a.configWatcher = watcher

	// Watch the config file
	if err := a.configWatcher.Add(configPath); err != nil {
		a.configWatcher.Close()
		return fmt.Errorf("failed to watch config file %s: %w", configPath, err)
	}

	a.logger.Info("Config file watcher initialized", zap.String("path", configPath))
	return nil
}

// watchConfigFile watches for config file changes and reloads configuration
func (a *App) watchConfigFile() {
	defer a.configWatcher.Close()

	for {
		select {
		case event, ok := <-a.configWatcher.Events:
			if !ok {
				return
			}

			if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) {
				a.logger.Debug("Config file changed, reloading configuration", zap.String("event", event.String()))

				// Add a small delay to ensure file write is complete
				time.Sleep(500 * time.Millisecond)

				if err := a.server.ReloadConfiguration(); err != nil {
					a.logger.Error("Failed to reload configuration", zap.Error(err))
				} else {
					a.logger.Debug("Configuration reloaded successfully")
					// Force a menu refresh after config reload
					a.forceRefresh = true
					a.refreshMenusImmediate()
				}
			}

		case err, ok := <-a.configWatcher.Errors:
			if !ok {
				return
			}
			a.logger.Error("Config file watcher error", zap.Error(err))

		case <-a.ctx.Done():
			return
		}
	}
}

// cleanup performs cleanup operations
func (a *App) cleanup() {
	if a.configWatcher != nil {
		a.configWatcher.Close()
	}
	a.cancel()
}

func (a *App) onReady() {
	systray.SetIcon(iconData)
	// On macOS, also set as template icon for better system integration
	if runtime.GOOS == osDarwin {
		systray.SetTemplateIcon(iconData, iconData)
	}
	a.updateTooltip()

	// --- Initialize Menu Items ---
	a.logger.Debug("Initializing tray menu items")
	a.statusItem = systray.AddMenuItem("Status: Initializing...", "Proxy server status")
	a.statusItem.Disable() // Initially disabled as it's just for display
	a.startStopItem = systray.AddMenuItem("Start Server", "Start the proxy server")
	a.serverCountItem = systray.AddMenuItem("📊 Servers: Loading...", "Total number of configured servers")
	a.serverCountItem.Disable() // Display only

	// Mark core menu items as ready - this will release waiting goroutines
	a.coreMenusReady = true
	a.logger.Debug("Core menu items initialized successfully - background processes can now start")
	systray.AddSeparator()

	// --- Status-Based Server Menus ---
	a.connectedServersMenu = systray.AddMenuItem("🟢 Connected Servers", "Connected and ready servers")
	a.disconnectedServersMenu = systray.AddMenuItem("🔴 Disconnected Servers", "Servers that are enabled but not connected")
	a.stoppedServersMenu = systray.AddMenuItem("⏹️ Stopped Servers", "Servers that have been stopped")
	a.disabledServersMenu = systray.AddMenuItem("⏸️ Disabled Servers", "Servers that are disabled")
	a.quarantineMenu = systray.AddMenuItem("🔒 Quarantined Servers", "Servers in security quarantine")
	systray.AddSeparator()

	// --- Legacy and Group Menus ---
	a.groupedServersMenu = systray.AddMenuItem("🏷️ Grouped Servers", "View servers organized by groups")
	a.groupManagementMenu = systray.AddMenuItem("🌐 Group Management", "Open web interface to manage server groups")
	systray.AddSeparator()

	// --- Initialize Managers ---
	a.menuManager = NewMenuManager(a.connectedServersMenu, a.disconnectedServersMenu, a.stoppedServersMenu, a.disabledServersMenu, a.quarantineMenu, nil, a.logger)
	a.syncManager = NewSynchronizationManager(a.stateManager, a.menuManager, a.logger)
	a.diagnosticAgent = NewDiagnosticAgent(a.logger.Desugar())

	// --- Set Action Callback ---
	// Centralized action handler for all menu-driven server actions
	a.menuManager.SetActionCallback(a.handleServerAction)

	// --- Set Server Count Callback ---
	// Set callback but with protection against smaller counts
	a.menuManager.SetServerCountCallback(a.updateServerCountDisplay)

	// --- Set Server Groups Reference ---
	// Allow MenuManager to access server groups for color display
	a.menuManager.SetServerGroups(&a.serverGroups)

	// --- Load Groups for Server Menus ---
	a.logger.Info("TRAY INIT: Loading groups for server assignment menus")
	a.loadGroupsForServerMenus()

	// --- Initialize Server Count Display ---
	// Load initial server count from config
	a.updateServerCountFromConfig()

	// --- Other Menu Items ---
	updateItem := systray.AddMenuItem("Check for Updates...", "Check for a new version of the proxy")
	openConfigItem := systray.AddMenuItem("Open config dir", "Open the configuration directory")
	editConfigItem := systray.AddMenuItem("Edit config", "Edit the configuration file")
	openLogsItem := systray.AddMenuItem("Open logs dir", "Open the logs directory")
	githubItem := systray.AddMenuItem("🔗 GitHub Repository", "Open the project on GitHub")
	systray.AddSeparator()

	// --- Autostart Menu Item (macOS only) ---
	if runtime.GOOS == osDarwin && a.autostartManager != nil {
		a.autostartItem = systray.AddMenuItem("Start at Login", "Start mcpproxy automatically when you log in")
		a.updateAutostartMenuItem()
		systray.AddSeparator()
	}

	quitItem := systray.AddMenuItem("Quit", "Quit the application")
	forceQuitItem := systray.AddMenuItem("🚨 Force Quit", "Force quit if application is hanging")

	// --- Set Initial State & Start Sync ---
	a.updateStatus()

	if err := a.syncManager.SyncNow(); err != nil {
		a.logger.Error("Initial menu sync failed", zap.Error(err))
	}

	a.syncManager.Start()

	// --- Click Handlers ---
	go func() {
		for {
			select {
			case <-a.startStopItem.ClickedCh:
				a.handleStartStop()
			case <-updateItem.ClickedCh:
				go a.checkForUpdates()
			case <-openConfigItem.ClickedCh:
				a.openConfigDir()
			case <-editConfigItem.ClickedCh:
				a.editConfigFile()
			case <-openLogsItem.ClickedCh:
				a.openLogsDir()
			case <-githubItem.ClickedCh:
				a.openGitHubRepository()
			case <-a.groupedServersMenu.ClickedCh:
				a.handleGroupedServers()
			case <-a.groupManagementMenu.ClickedCh:
				a.openGroupManagementWeb()
			case <-a.connectedServersMenu.ClickedCh:
				a.handleServerMenuClick("connected")
			case <-a.disconnectedServersMenu.ClickedCh:
				a.handleServerMenuClick("disconnected")
			case <-a.stoppedServersMenu.ClickedCh:
				a.handleServerMenuClick("stopped")
			case <-a.disabledServersMenu.ClickedCh:
				a.handleServerMenuClick("disabled")
			case <-a.quarantineMenu.ClickedCh:
				a.handleServerMenuClick("quarantine")
			case <-quitItem.ClickedCh:
				a.logger.Info("Quit item clicked, shutting down")
				
				// Force quit with timeout to prevent hanging
				go func() {
					// Set a maximum time for graceful shutdown
					timeout := time.After(3 * time.Second)
					done := make(chan bool, 1)
					
					go func() {
						if a.shutdown != nil {
							a.shutdown()
						}
						done <- true
					}()
					
					select {
					case <-done:
						a.logger.Info("Graceful shutdown completed")
					case <-timeout:
						a.logger.Warn("Graceful shutdown timed out, forcing exit")
						os.Exit(0) // Force exit if graceful shutdown hangs
					}
				}()
				return
			case <-forceQuitItem.ClickedCh:
				a.logger.Warn("Force quit requested - exiting immediately")
				os.Exit(0)
			case <-a.ctx.Done():
				return
			}
		}
	}()

	// --- Autostart Click Handler (separate goroutine for macOS) ---
	if runtime.GOOS == osDarwin && a.autostartItem != nil {
		go func() {
			for {
				select {
				case <-a.autostartItem.ClickedCh:
					a.handleAutostartToggle()
				case <-a.ctx.Done():
					return
				}
			}
		}()
	}

	a.logger.Info("System tray is ready - menu items fully initialized")
}

// updateTooltip updates the tooltip based on the server's running state
func (a *App) updateTooltip() {
	if a.server == nil {
		systray.SetTooltip("mcpproxy is stopped")
		a.updateServerCountFromConfig()
		return
	}

	// Get full status and use comprehensive tooltip
	statusData := a.server.GetStatus()
	if status, ok := statusData.(map[string]interface{}); ok {
		a.updateTooltipFromStatusData(status)
	} else {
		// Fallback to basic tooltip if status format is unexpected
		if a.server.IsRunning() {
			systray.SetTooltip(fmt.Sprintf("mcpproxy is running on %s", a.server.GetListenAddress()))
		} else {
			systray.SetTooltip("mcpproxy is stopped")
			a.updateServerCountFromConfig()
		}
	}
}

// updateStatusFromData updates menu items based on real-time status data from the server
func (a *App) updateStatusFromData(statusData interface{}) {
	// Handle different status data formats
	var status map[string]interface{}
	var ok bool

	switch v := statusData.(type) {
	case map[string]interface{}:
		status = v
		ok = true
	case server.Status:
		// Convert Status struct to map for consistent handling
		status = map[string]interface{}{
			"running":     a.server != nil && a.server.IsRunning(),
			"listen_addr": "",
			"phase":       v.Phase,
			"message":     v.Message,
		}
		if a.server != nil {
			status["listen_addr"] = a.server.GetListenAddress()
		}
		ok = true
	default:
		// Try to handle basic server state even with unexpected format
		a.logger.Debug("Received status data in unexpected format, using fallback",
			zap.String("type", fmt.Sprintf("%T", statusData)))

		// Fallback to basic server state
		if a.server != nil {
			status = map[string]interface{}{
				"running":     a.server.IsRunning(),
				"listen_addr": a.server.GetListenAddress(),
				"phase":       "Unknown",
				"message":     "Status format unknown",
			}
			ok = true
		} else {
			// No server available, can't determine status
			return
		}
	}

	if !ok {
		a.logger.Warn("Unable to process status data, skipping update")
		return
	}

	// Check if core menu items are ready to prevent nil pointer dereference
	if !a.coreMenusReady {
		a.logger.Debug("Core menu items not ready yet, skipping status update from data")
		return
	}

	// Debug logging to track status updates
	running, _ := status["running"].(bool)
	phase, _ := status["phase"].(string)
	serverRunning := a.server != nil && a.server.IsRunning()

	a.logger.Debug("Updating tray status",
		zap.Bool("status_running", running),
		zap.Bool("server_is_running", serverRunning),
		zap.String("phase", phase),
		zap.Any("status_data", status))

	// Use the actual server running state as the authoritative source
	actuallyRunning := serverRunning

	// Update running status and start/stop button
	if actuallyRunning {
		listenAddr, _ := status["listen_addr"].(string)
		if listenAddr != "" {
			a.statusItem.SetTitle(fmt.Sprintf("Status: Running (%s)", listenAddr))
		} else {
			a.statusItem.SetTitle("Status: Running")
		}
		a.startStopItem.SetTitle("Stop Server")
		a.logger.Debug("Set tray to running state")
	} else {
		a.statusItem.SetTitle("Status: Stopped")
		a.startStopItem.SetTitle("Start Server")
		a.logger.Debug("Set tray to stopped state")
	}

	// Update tooltip
	a.updateTooltipFromStatusData(status)

	// Update server menus using the manager (only if server is running)
	if a.syncManager != nil {
		if actuallyRunning {
			a.syncManager.SyncDelayed()
		} else {
			// When server is stopped, preserve the last known server list but update connection status
			// This prevents the UI from showing (0/0) when the server is temporarily stopped
			// The menu items will still be visible but will show disconnected status
			a.logger.Debug("Server stopped, preserving menu state with disconnected status")
			// DON'T clear menus - this causes the (0/0) flickering issue
			// DON'T clear quarantine menu - quarantine data is persistent storage,
			// not runtime connection data. Users should manage quarantined servers
			// even when server is stopped.
			// a.menuManager.UpdateQuarantineMenu([]map[string]interface{}{})
		}
	}
}

// updateTooltipFromStatusData updates the tray tooltip from status data map
func (a *App) updateTooltipFromStatusData(status map[string]interface{}) {
	running, _ := status["running"].(bool)

	if !running {
		systray.SetTooltip("mcpproxy is stopped")
		return
	}

	// Build comprehensive tooltip for running server
	listenAddr, _ := status["listen_addr"].(string)
	phase, _ := status["phase"].(string)
	toolsIndexed, _ := status["tools_indexed"].(int)

	// Get upstream stats for connected servers and total tools
	upstreamStats, _ := status["upstream_stats"].(map[string]interface{})

	var connectedServers, totalServers, totalTools int
	if upstreamStats != nil {
		if connected, ok := upstreamStats["connected_servers"].(int); ok {
			connectedServers = connected
		}
		if total, ok := upstreamStats["total_servers"].(int); ok {
			totalServers = total
		}
		if tools, ok := upstreamStats["total_tools"].(int); ok {
			totalTools = tools
		}
	}

	// Build multi-line tooltip with comprehensive information
	var tooltipLines []string

	// Main status line
	tooltipLines = append(tooltipLines, fmt.Sprintf("mcpproxy (%s) - %s", phase, listenAddr))

	// Server connection status
	if totalServers > 0 {
		tooltipLines = append(tooltipLines, fmt.Sprintf("Servers: %d/%d connected", connectedServers, totalServers))
	} else {
		tooltipLines = append(tooltipLines, "Servers: none configured")
	}

	// Tool count - this is the key information the user wanted
	if totalTools > 0 {
		tooltipLines = append(tooltipLines, fmt.Sprintf("Tools: %d available", totalTools))
	} else if toolsIndexed > 0 {
		// Fallback to indexed count if total tools not available
		tooltipLines = append(tooltipLines, fmt.Sprintf("Tools: %d indexed", toolsIndexed))
	} else {
		tooltipLines = append(tooltipLines, "Tools: none available")
	}

	tooltip := strings.Join(tooltipLines, "\n")
	systray.SetTooltip(tooltip)

	// Note: We no longer update server count here because we want to show
	// the static count from config, not the dynamic count from server status
}

// updateServerCountDisplay updates the server count menu item
func (a *App) updateServerCountDisplay(totalServers int) {
	a.logger.Debug("updateServerCountDisplay called", zap.Int("total_servers", totalServers))

	if a.serverCountItem == nil {
		a.logger.Debug("serverCountItem is nil, skipping update")
		return
	}

	// If we have a config count and this is a smaller dynamic count, ignore it
	if a.serverCountFromConfig > 0 && totalServers < a.serverCountFromConfig {
		a.logger.Debug("Ignoring smaller dynamic count, keeping config count", 
			zap.Int("config_count", a.serverCountFromConfig), 
			zap.Int("dynamic_count", totalServers))
		totalServers = a.serverCountFromConfig
	}

	var displayText string
	if totalServers > 0 {
		displayText = fmt.Sprintf("📊 Servers: %d total", totalServers)
	} else {
		displayText = "📊 Servers: none configured"
	}

	a.logger.Debug("Setting server count display", zap.String("display_text", displayText))
	a.serverCountItem.SetTitle(displayText)
	a.serverCountItem.SetTooltip(fmt.Sprintf("%d servers configured in total", totalServers))
}

// updateServerCountFromConfig reads the config file and updates server count display
func (a *App) updateServerCountFromConfig() {
	a.logger.Debug("updateServerCountFromConfig called")
	
	// Try to load config from default location
	homeDir, err := os.UserHomeDir()
	if err != nil {
		a.logger.Warnf("Failed to get home directory: %v", err)
		return
	}

	configPath := filepath.Join(homeDir, ".mcpproxy", "mcp_config.json")
	a.logger.Debugf("Loading config from: %s", configPath)
	
	cfg, err := config.LoadFromFile(configPath)
	if err != nil {
		a.logger.Debugf("Failed to load config for server count: %v", err)
		return
	}

	serverCount := len(cfg.Servers)
	a.logger.Debugf("Loaded %d servers from config", serverCount)
	
	// Store the config count
	a.serverCountFromConfig = serverCount
	a.serverCountInitialized = true
	
	a.updateServerCountDisplay(serverCount)
}

// updateServersMenuFromStatusData is a legacy method, functionality is now in MenuManager
func (a *App) updateServersMenuFromStatusData(_ map[string]interface{}) {
	// This function is kept for reference during transition but the primary
	// logic is now handled by the MenuManager and SynchronizationManager.
	// We trigger a sync instead of manually updating here.
	if a.syncManager != nil {
		a.syncManager.SyncDelayed()
	}
}

// updateStatus updates the status menu item and tooltip
func (a *App) updateStatus() {
	if a.server == nil {
		return
	}

	// Check if core menu items are ready
	if !a.coreMenusReady {
		a.logger.Debug("Core menu items not ready yet, skipping status update")
		return
	}

	statusData := a.server.GetStatus()
	a.updateStatusFromData(statusData)
}

// updateServersMenu is a legacy method, now triggers a sync
func (a *App) updateServersMenu() {
	if a.syncManager != nil {
		a.syncManager.SyncDelayed()
	}
}

// handleStartStop toggles the server's running state
func (a *App) handleStartStop() {
	if a.server.IsRunning() {
		a.logger.Info("Stopping server from tray")

		// Save server states before stopping
		if err := a.saveServerStatesForStop(); err != nil {
			a.logger.Error("Failed to save server states", zap.Error(err))
		}

		// Immediately update UI to show stopping state and disable button
		if a.statusItem != nil {
			a.statusItem.SetTitle("Status: Stopping...")
		}
		if a.startStopItem != nil {
			a.startStopItem.SetTitle("Stopping...")
			a.startStopItem.Disable() // Prevent multiple clicks
		}

		// Stop the server with timeout protection
		go func() {
			done := make(chan bool, 1)
			
			// Run stop operation in separate goroutine
			go func() {
				defer func() {
					if r := recover(); r != nil {
						a.logger.Error("Panic during server stop", zap.Any("panic", r))
					}
					done <- true
				}()
				
				if err := a.server.StopServer(); err != nil {
					a.logger.Error("Failed to stop server", zap.Error(err))
				}
			}()

			// Wait for completion or timeout
			select {
			case <-done:
				a.logger.Info("Server stop operation completed")
			case <-time.After(5 * time.Second):
				a.logger.Warn("Server stop operation timed out after 5 seconds")
			}

			// Always restore UI state
			if a.startStopItem != nil {
				a.startStopItem.Enable()
			}
			a.updateStatus()
		}()
	} else {
		a.logger.Info("Starting server from tray")

		// Immediately update UI to show starting state and disable button
		if a.statusItem != nil {
			a.statusItem.SetTitle("Status: Starting...")
		}
		if a.startStopItem != nil {
			a.startStopItem.SetTitle("Starting...")
			a.startStopItem.Disable() // Prevent multiple clicks
		}

		// Start the server with timeout protection
		go func() {
			done := make(chan bool, 1)
			
			// Run start operation in separate goroutine
			go func() {
				defer func() {
					if r := recover(); r != nil {
						a.logger.Error("Panic during server start", zap.Any("panic", r))
					}
					done <- true
				}()
				
				if err := a.server.StartServer(a.ctx); err != nil {
					a.logger.Error("Failed to start server", zap.Error(err))
				}
			}()

			// Wait for completion or timeout
			select {
			case <-done:
				a.logger.Info("Server start operation completed")
				// Restore server states after successful start
				if err := a.restoreServerStatesAfterStart(); err != nil {
					a.logger.Error("Failed to restore server states", zap.Error(err))
				}
			case <-time.After(10 * time.Second):
				a.logger.Warn("Server start operation timed out after 10 seconds")
			}

			// Always restore UI state
			if a.startStopItem != nil {
				a.startStopItem.Enable()
			}
			a.updateStatus()
		}()
	}
}

// onExit is called when the application is quitting
func (a *App) onExit() {
	a.logger.Info("Tray application exiting")
	a.cleanup()
	if a.cancel != nil {
		a.cancel()
	}
}

// checkForUpdates checks for new releases on GitHub
func (a *App) checkForUpdates() {
	// Check if auto-update is disabled by environment variable
	if os.Getenv("MCPPROXY_DISABLE_AUTO_UPDATE") == trueStr {
		a.logger.Info("Auto-update disabled by environment variable")
		return
	}

	// Disable auto-update for app bundles by default (DMG installation should be manual)
	if a.isAppBundle() && os.Getenv("MCPPROXY_UPDATE_APP_BUNDLE") != trueStr {
		a.logger.Info("Auto-update disabled for app bundle installations (use DMG for updates)")
		return
	}

	// Check if notification-only mode is enabled
	notifyOnly := os.Getenv("MCPPROXY_UPDATE_NOTIFY_ONLY") == trueStr

	a.statusItem.SetTitle("Checking for updates...")
	defer a.updateStatus() // Restore original status after check

	release, err := a.getLatestRelease()
	if err != nil {
		a.logger.Error("Failed to get latest release", zap.Error(err))
		return
	}

	latestVersion := strings.TrimPrefix(release.TagName, "v")
	if semver.Compare("v"+a.version, "v"+latestVersion) >= 0 {
		a.logger.Info("You are running the latest version", zap.String("version", a.version))
		return
	}

	if notifyOnly {
		a.logger.Info("Update available - notification only mode",
			zap.String("current", a.version),
			zap.String("latest", latestVersion),
			zap.String("url", fmt.Sprintf("https://github.com/%s/releases/tag/%s", repo, release.TagName)))

		// You could add desktop notification here if desired
		a.statusItem.SetTitle(fmt.Sprintf("Update available: %s", latestVersion))
		return
	}

	downloadURL, err := a.findAssetURL(release)
	if err != nil {
		a.logger.Error("Failed to find asset for your system", zap.Error(err))
		return
	}

	if err := a.downloadAndApplyUpdate(downloadURL); err != nil {
		a.logger.Error("Update failed", zap.Error(err))
	}
}

// getLatestRelease fetches the latest release information from GitHub
func (a *App) getLatestRelease() (*GitHubRelease, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo)
	resp, err := http.Get(url) // #nosec G107 -- URL is constructed from known repo constant
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}
	return &release, nil
}

// findAssetURL finds the correct asset URL for the current system
func (a *App) findAssetURL(release *GitHubRelease) (string, error) {
	// Check if this is a Homebrew installation to avoid conflicts
	if a.isHomebrewInstallation() {
		return "", fmt.Errorf("auto-update disabled for Homebrew installations - use 'brew upgrade mcpproxy' instead")
	}

	// Determine file extension based on platform
	var extension string
	switch runtime.GOOS {
	case osWindows:
		extension = ".zip"
	default: // macOS, Linux
		extension = ".tar.gz"
	}

	// Try latest assets first (for website integration)
	latestSuffix := fmt.Sprintf("latest-%s-%s%s", runtime.GOOS, runtime.GOARCH, extension)
	for _, asset := range release.Assets {
		if strings.HasSuffix(asset.Name, latestSuffix) {
			return asset.BrowserDownloadURL, nil
		}
	}

	// Fallback to versioned assets
	versionedSuffix := fmt.Sprintf("-%s-%s%s", runtime.GOOS, runtime.GOARCH, extension)
	for _, asset := range release.Assets {
		if strings.HasSuffix(asset.Name, versionedSuffix) {
			return asset.BrowserDownloadURL, nil
		}
	}

	return "", fmt.Errorf("no suitable asset found for %s-%s (tried %s and %s)",
		runtime.GOOS, runtime.GOARCH, latestSuffix, versionedSuffix)
}

// isHomebrewInstallation checks if this is a Homebrew installation
func (a *App) isHomebrewInstallation() bool {
	execPath, err := os.Executable()
	if err != nil {
		return false
	}

	// Check if running from Homebrew path
	return strings.Contains(execPath, "/opt/homebrew/") ||
		strings.Contains(execPath, "/usr/local/Homebrew/") ||
		strings.Contains(execPath, "/home/linuxbrew/")
}

// isAppBundle checks if running from macOS app bundle
func (a *App) isAppBundle() bool {
	if runtime.GOOS != osDarwin {
		return false
	}

	execPath, err := os.Executable()
	if err != nil {
		return false
	}

	return strings.Contains(execPath, ".app/Contents/MacOS/")
}

// downloadAndApplyUpdate downloads and applies the update
func (a *App) downloadAndApplyUpdate(url string) error {
	resp, err := http.Get(url) // #nosec G107 -- URL is from GitHub releases API which is trusted
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if strings.HasSuffix(url, ".zip") {
		return a.applyZipUpdate(resp.Body)
	} else if strings.HasSuffix(url, ".tar.gz") {
		return a.applyTarGzUpdate(resp.Body)
	}

	return update.Apply(resp.Body, update.Options{})
}

// applyZipUpdate extracts and applies an update from a zip archive
func (a *App) applyZipUpdate(body io.Reader) error {
	tmpfile, err := os.CreateTemp("", "update-*.zip")
	if err != nil {
		return err
	}
	defer os.Remove(tmpfile.Name())
	defer tmpfile.Close()

	_, err = io.Copy(tmpfile, body)
	if err != nil {
		return err
	}

	r, err := zip.OpenReader(tmpfile.Name())
	if err != nil {
		return err
	}
	defer r.Close()

	executablePath, err := os.Executable()
	if err != nil {
		return err
	}

	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}

		err = update.Apply(rc, update.Options{TargetPath: executablePath})
		rc.Close()
		return err
	}

	return fmt.Errorf("no file found in zip archive to apply")
}

// applyTarGzUpdate extracts and applies an update from a tar.gz archive
func (a *App) applyTarGzUpdate(body io.Reader) error {
	// For tar.gz files, we need to extract and find the binary
	tmpfile, err := os.CreateTemp("", "update-*.tar.gz")
	if err != nil {
		return err
	}
	defer os.Remove(tmpfile.Name())
	defer tmpfile.Close()

	_, err = io.Copy(tmpfile, body)
	if err != nil {
		return err
	}

	// Open the tar.gz file and extract the binary
	if _, err := tmpfile.Seek(0, 0); err != nil {
		return fmt.Errorf("failed to seek to beginning of file: %w", err)
	}

	gzr, err := gzip.NewReader(tmpfile)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// Look for the mcpproxy binary (could be mcpproxy or mcpproxy.exe)
		if strings.HasSuffix(header.Name, "mcpproxy") || strings.HasSuffix(header.Name, "mcpproxy.exe") {
			executablePath, err := os.Executable()
			if err != nil {
				return err
			}

			return update.Apply(tr, update.Options{TargetPath: executablePath})
		}
	}

	return fmt.Errorf("no mcpproxy binary found in tar.gz archive")
}

// openConfigDir opens the directory containing the configuration file
func (a *App) openConfigDir() {
	if a.configPath == "" {
		a.logger.Warn("Config path is not set, cannot open")
		return
	}

	configDir := filepath.Dir(a.configPath)
	a.openDirectory(configDir, "config directory")
}

// openLogsDir opens the logs directory
func (a *App) openLogsDir() {
	if a.server == nil {
		a.logger.Warn("Server interface not available, cannot open logs directory")
		return
	}

	logDir := a.server.GetLogDir()
	if logDir == "" {
		a.logger.Warn("Log directory path is not set, cannot open")
		return
	}

	a.openDirectory(logDir, "logs directory")
}

// editConfigFile opens the configuration file in the default editor
func (a *App) editConfigFile() {
	if a.configPath == "" {
		a.logger.Warn("Config path is not set, cannot open")
		return
	}
	a.openFile(a.configPath, "config file")
}

// openGitHubRepository opens the GitHub repository URL in the default browser
func (a *App) openGitHubRepository() {
	// Notify sync manager of user activity for adaptive frequency
	if a.syncManager != nil {
		a.syncManager.NotifyUserActivity()
	}

	githubURL := a.server.GetGitHubURL()

	a.logger.Info("Opening GitHub repository", zap.String("url", githubURL))
	a.openFile(githubURL, "GitHub repository")
}

// openServerLog opens the log file for a specific server
func (a *App) openServerLog(serverName string) error {
	if a.server == nil {
		return fmt.Errorf("server interface not available")
	}
	logDir := a.server.GetLogDir()
	if logDir == "" {
		return fmt.Errorf("log directory path is not set")
	}
	logPath := filepath.Join(logDir, fmt.Sprintf("server-%s.log", serverName))
	a.openFile(logPath, "server log")
	return nil
}

// openServerRepo opens the repository/URL for a specific server
func (a *App) openServerRepo(serverName string) error {
	// Notify sync manager of user activity for adaptive frequency
	if a.syncManager != nil {
		a.syncManager.NotifyUserActivity()
	}

	allServers, err := a.stateManager.GetAllServers()
	if err != nil {
		return fmt.Errorf("failed to get servers: %w", err)
	}
	var url string
	for _, srv := range allServers {
		if name, ok := srv["name"].(string); ok && name == serverName {
			// Prefer repository_url over url
			if repoURL, ok := srv["repository_url"].(string); ok && repoURL != "" {
				url = repoURL
			} else if u, ok := srv["url"].(string); ok && u != "" {
				// Fallback to server URL for HTTP servers
				url = u
			}
			break
		}
	}
	if url == "" {
		return fmt.Errorf("no repository URL for server %s", serverName)
	}
	a.logger.Info("Opening server repository", zap.String("server", serverName), zap.String("url", url))
	a.openFile(url, "server repository")
	return nil
}

// openDirectory opens a directory using the OS-specific file manager
func (a *App) openDirectory(dirPath, dirType string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case osDarwin:
		cmd = exec.Command("open", dirPath)
	case "linux":
		cmd = exec.Command("xdg-open", dirPath)
	case osWindows:
		cmd = exec.Command("explorer", dirPath)
	default:
		a.logger.Warn("Unsupported OS for opening directory", zap.String("os", runtime.GOOS))
		return
	}

	if err := cmd.Run(); err != nil {
		a.logger.Error("Failed to open directory", zap.Error(err), zap.String("dir_type", dirType), zap.String("path", dirPath))
	} else {
		a.logger.Info("Successfully opened directory", zap.String("dir_type", dirType), zap.String("path", dirPath))
	}
}

// openFile opens a file or URL using the OS-specific handler
func (a *App) openFile(path, fileType string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case osDarwin:
		cmd = exec.Command("open", path)
	case "linux":
		cmd = exec.Command("xdg-open", path)
	case osWindows:
		cmd = exec.Command("cmd", "/c", "start", "", path)
	default:
		a.logger.Warn("Unsupported OS for opening file", zap.String("os", runtime.GOOS))
		return
	}

	if err := cmd.Run(); err != nil {
		a.logger.Error("Failed to open file", zap.Error(err), zap.String("file_type", fileType), zap.String("path", path))
	} else {
		a.logger.Info("Successfully opened file", zap.String("file_type", fileType), zap.String("path", path))
	}
}

// openGroupManagementWeb opens the web interface for group management
func (a *App) openGroupManagementWeb() {
	url := "http://localhost:8080/groups"
	a.openFile(url, "group management web interface")
}

// refreshMenusDelayed refreshes menus after a delay using the synchronization manager
func (a *App) refreshMenusDelayed() {
	if a.syncManager != nil {
		a.syncManager.SyncDelayed()
	} else {
		a.logger.Warn("Sync manager not initialized for delayed refresh")
	}
}

// refreshMenusImmediate refreshes menus immediately using the synchronization manager
func (a *App) refreshMenusImmediate() {
	if err := a.syncManager.SyncNow(); err != nil {
		a.logger.Error("Failed to refresh menus immediately", zap.Error(err))
	}
}

// handleServerMenuClick handles lazy loading when server menus are clicked
func (a *App) handleServerMenuClick(menuType string) {
	a.logger.Info("Server menu clicked, loading data", zap.String("menu_type", menuType))
	
	// Mark menu as open to prevent updates while user is interacting
	if a.menuManager != nil {
		a.menuManager.SetMenuOpen()
		
		// Set a timer to mark menu as closed after user interaction timeout
		go func() {
			time.Sleep(30 * time.Second)
			if a.menuManager != nil {
				a.menuManager.SetMenuClosed()
			}
		}()
	}
	
	// Notify sync manager of user activity for adaptive frequency
	if a.syncManager != nil {
		a.syncManager.NotifyUserActivity()
	}
	
	// Trigger immediate sync to load current data
	if err := a.syncManager.SyncNow(); err != nil {
		a.logger.Error("Failed to sync menu data on click", zap.Error(err))
	}
}
func (a *App) handleServerAction(serverName, action string) {
	var err error
	a.logger.Info("Handling server action", zap.String("server", serverName), zap.String("action", action))

	// Notify sync manager of user activity for adaptive frequency
	if a.syncManager != nil {
		a.syncManager.NotifyUserActivity()
	}

	switch action {
	case "toggle_enable":
		allServers, getErr := a.stateManager.GetAllServers()
		if getErr != nil {
			a.logger.Error("Failed to get servers for toggle action", zap.Error(getErr))
			return
		}

		var serverEnabled bool
		found := false
		for _, server := range allServers {
			if name, ok := server["name"].(string); ok && name == serverName {
				if enabled, ok := server["enabled"].(bool); ok {
					serverEnabled = enabled
					found = true
					break
				}
			}
		}

		if !found {
			a.logger.Error("Server not found for toggle action", zap.String("server", serverName))
			return
		}
		err = a.syncManager.HandleServerEnable(serverName, !serverEnabled)

	case "oauth_login":
		err = a.handleOAuthLogin(serverName)

	case "quarantine":
		err = a.syncManager.HandleServerQuarantine(serverName, true)

	case "unquarantine":
		err = a.syncManager.HandleServerUnquarantine(serverName)

	case "open_log":
		err = a.openServerLog(serverName)

	case "open_repo":
		err = a.openServerRepo(serverName)

	case "configure":
		err = a.handleServerConfiguration(serverName)

	default:
		// Check if it's a group action
		if strings.HasPrefix(action, "assign_to_group:") {
			groupName := strings.TrimPrefix(action, "assign_to_group:")
			err = a.handleAssignServerToGroup(serverName, groupName)
		} else if strings.HasPrefix(action, "remove_from_group:") {
			groupName := strings.TrimPrefix(action, "remove_from_group:")
			err = a.handleRemoveServerFromGroup(serverName, groupName)
		} else {
			a.logger.Warn("Unknown server action requested", zap.String("action", action))
		}
	}

	if err != nil {
		a.logger.Error("Failed to handle server action",
			zap.String("server", serverName),
			zap.String("action", action),
			zap.Error(err))
	}
}

// handleOAuthLogin handles OAuth authentication for a server from the tray menu
func (a *App) handleOAuthLogin(serverName string) error {
	a.logger.Info("Starting OAuth login from tray menu", zap.String("server", serverName))

	// Get server information from the state manager (same source as tray menu)
	allServers, err := a.stateManager.GetAllServers()
	if err != nil {
		a.logger.Error("Failed to get servers for OAuth login",
			zap.String("server", serverName),
			zap.Error(err))
		return fmt.Errorf("failed to get servers: %w", err)
	}

	// Debug: List all available servers
	var availableServerNames []string
	for _, server := range allServers {
		if name, ok := server["name"].(string); ok {
			availableServerNames = append(availableServerNames, name)
		}
	}
	a.logger.Info("Available servers from state manager",
		zap.String("requested_server", serverName),
		zap.Strings("available_servers", availableServerNames))

	// Find the requested server
	var targetServer map[string]interface{}
	for _, server := range allServers {
		if name, ok := server["name"].(string); ok && name == serverName {
			targetServer = server
			break
		}
	}

	if targetServer == nil {
		err := fmt.Errorf("server '%s' not found in available servers", serverName)
		a.logger.Error("Server not found for OAuth login",
			zap.String("server", serverName),
			zap.Strings("available_servers", availableServerNames))
		return err
	}

	a.logger.Info("Found server for OAuth",
		zap.String("server", serverName),
		zap.Any("server_data", targetServer))

	// Load the config file that mcpproxy is using
	configPath := a.server.GetConfigPath()
	if configPath == "" {
		err := fmt.Errorf("config path not available")
		a.logger.Error("Failed to get config path for OAuth login",
			zap.String("server", serverName),
			zap.Error(err))
		return err
	}

	a.logger.Info("Loading config file for OAuth",
		zap.String("server", serverName),
		zap.String("config_path", configPath))

	globalConfig, err := config.LoadFromFile(configPath)
	if err != nil {
		a.logger.Error("Failed to load server configuration for OAuth login",
			zap.String("server", serverName),
			zap.String("config_path", configPath),
			zap.Error(err))
		return fmt.Errorf("failed to load server configuration: %w", err)
	}

	// Debug: Check if server exists in config
	var serverFound bool
	for _, server := range globalConfig.Servers {
		if server.Name == serverName {
			serverFound = true
			break
		}
	}

	a.logger.Info("Server lookup in config",
		zap.String("server", serverName),
		zap.Bool("found_in_config", serverFound),
		zap.String("config_path", configPath))

	a.logger.Info("Config loaded for OAuth",
		zap.String("server", serverName),
		zap.Int("total_servers", len(globalConfig.Servers)))

	// Trigger OAuth inside the running daemon to avoid DB lock conflicts
	a.logger.Info("Triggering in-process OAuth from tray", zap.String("server", serverName))
	if err := a.server.TriggerOAuthLogin(serverName); err != nil {
		return fmt.Errorf("failed to trigger OAuth: %w", err)
	}
	return nil
}

// stringSlicesEqual compares two string slices for equality
func stringSlicesEqual(a, b []string) bool {
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

// updateAutostartMenuItem updates the autostart menu item based on current state
func (a *App) updateAutostartMenuItem() {
	if a.autostartItem == nil || a.autostartManager == nil {
		return
	}

	if a.autostartManager.IsEnabled() {
		a.autostartItem.SetTitle("☑️ Start at Login")
		a.autostartItem.SetTooltip("mcpproxy will start automatically when you log in (click to disable)")
	} else {
		a.autostartItem.SetTitle("Start at Login")
		a.autostartItem.SetTooltip("Start mcpproxy automatically when you log in (click to enable)")
	}
}

// handleAutostartToggle handles toggling the autostart functionality
func (a *App) handleAutostartToggle() {
	if a.autostartManager == nil {
		a.logger.Warn("Autostart manager not available")
		return
	}

	a.logger.Info("Toggling autostart functionality")

	if err := a.autostartManager.Toggle(); err != nil {
		a.logger.Error("Failed to toggle autostart", zap.Error(err))
		return
	}

	// Update the menu item to reflect the new state
	a.updateAutostartMenuItem()

	// Log the new state
	if a.autostartManager.IsEnabled() {
		a.logger.Info("Autostart enabled - mcpproxy will start automatically at login")
	} else {
		a.logger.Info("Autostart disabled - mcpproxy will not start automatically at login")
	}
}

// handleGroupedServers handles clicks on the grouped servers menu
func (a *App) handleGroupedServers() {
	a.logger.Info("Opening grouped servers view")

	// Notify sync manager of user activity for adaptive frequency
	if a.syncManager != nil {
		a.syncManager.NotifyUserActivity()
	}

	// Clear existing group server menu items if they exist
	if a.groupedServersMenu != nil {
		// Remove all submenu items
		a.groupedServersMenu.Hide()
		a.groupedServersMenu.Show()
	}

	// Fetch groups from the web interface
	groups, err := a.fetchGroupsFromAPI()
	if err != nil {
		a.logger.Error("Failed to fetch groups", zap.Error(err))
		// Show error item
		errorItem := a.groupedServersMenu.AddSubMenuItem("❌ Failed to load groups", "Error loading groups from web interface")
		errorItem.Disable()
		return
	}

	// Fetch server assignments
	assignments, err := a.fetchServerAssignments()
	if err != nil {
		a.logger.Error("Failed to fetch server assignments", zap.Error(err))
		assignments = make(map[string]string) // Empty assignments on error
	}

	// Create menu items for each group
	for _, group := range groups {
		groupName := group["name"].(string)
		
		// Get servers assigned to this group
		groupServers := make([]string, 0)
		for serverName, assignedGroup := range assignments {
			if assignedGroup == groupName {
				groupServers = append(groupServers, serverName)
			}
		}

		// Create group menu item
		groupItem := a.groupedServersMenu.AddSubMenuItem(
			fmt.Sprintf("🏷️ %s (%d)", groupName, len(groupServers)), 
			fmt.Sprintf("Servers in group '%s'", groupName))

		// Add servers in this group
		if len(groupServers) == 0 {
			emptyItem := groupItem.AddSubMenuItem("📭 No servers assigned", "Use the 'groups' MCP tool to assign servers")
			emptyItem.Disable()
		} else {
			for _, serverName := range groupServers {
				serverItem := groupItem.AddSubMenuItem(
					fmt.Sprintf("📋 %s", serverName),
					fmt.Sprintf("Server: %s", serverName))
				
				// Add server actions
				go func(sName string, item *systray.MenuItem) {
					for range item.ClickedCh {
						a.handleServerAction(sName, "view")
					}
				}(serverName, serverItem)
			}
		}
	}
}

// handleGroupManagement handles clicks on the group management menu
func (a *App) handleGroupManagement() {
	a.logger.Info("Opening group management interface")

	// Notify sync manager of user activity for adaptive frequency
	if a.syncManager != nil {
		a.syncManager.NotifyUserActivity()
	}

	// Clear existing group menu items if they exist
	if a.groupManagementMenu != nil {
		// Remove all submenu items
		for _, item := range a.groupMenuItems {
			if item != nil {
				// Note: systray doesn't have a direct way to remove items, so we hide them
				item.Hide()
			}
		}
		a.groupMenuItems = make(map[string]*systray.MenuItem)
	}

	// Add group management options and show existing groups
	a.logger.Info("TRAY INIT: About to add group management items")
	createGroupItem := a.groupManagementMenu.AddSubMenuItem("➕ Create New Group", "Create a new server group with color assignment")
	manageGroupsItem := a.groupManagementMenu.AddSubMenuItem("⚙️ Manage Groups", "Edit existing server groups")
	a.groupManagementMenu.AddSeparator()

	// Show existing groups from API
	a.logger.Info("TRAY INIT: About to call refreshGroupsMenu")
	a.refreshGroupsMenu()
	
	// Also schedule a delayed refresh in case server wasn't ready initially
	go func() {
		time.Sleep(3 * time.Second)
		a.logger.Info("TRAY INIT: Delayed refreshGroupsMenu call")
		a.refreshGroupsMenu()
	}()

	// Handle clicks on group management items
	go func() {
		for {
			select {
			case <-createGroupItem.ClickedCh:
				a.handleCreateGroup()
			case <-manageGroupsItem.ClickedCh:
				a.handleManageGroups()
			case <-a.ctx.Done():
				return
			}
		}
	}()
}

// saveGroupsToConfig saves the current groups to the configuration file
func (a *App) saveGroupsToConfig() error {
	if a.server == nil {
		return fmt.Errorf("server interface not available")
	}

	configPath := a.server.GetConfigPath()
	if configPath == "" {
		return fmt.Errorf("config path not available")
	}

	// Read the current config file as JSON
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	var configData map[string]interface{}
	if err := json.Unmarshal(data, &configData); err != nil {
		return fmt.Errorf("failed to parse config JSON: %w", err)
	}

	// Convert serverGroups to config format
	groups := make([]map[string]interface{}, 0, len(a.serverGroups))
	for _, group := range a.serverGroups {
		if group.Enabled {
			groups = append(groups, map[string]interface{}{
				"name":        group.Name,
				"description": group.Description,
				"color":       group.Color,
				"enabled":     group.Enabled,
			})
		}
	}

	// Update the groups in the config
	configData["groups"] = groups

	// Update server group assignments
	if servers, ok := configData["mcpServers"].([]interface{}); ok {
		for _, serverInterface := range servers {
			if server, ok := serverInterface.(map[string]interface{}); ok {
				if serverName, ok := server["name"].(string); ok {
					// Find which group this server belongs to
					server["group_name"] = "" // Reset first
					for groupName, group := range a.serverGroups {
						for _, assignedServerName := range group.ServerNames {
							if assignedServerName == serverName {
								server["group_name"] = groupName
								break
							}
						}
						if server["group_name"] != "" {
							break
						}
					}
				}
			}
		}
	}

	// Write the updated config back to file
	updatedData, err := json.MarshalIndent(configData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config JSON: %w", err)
	}

	if err := os.WriteFile(configPath, updatedData, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	a.logger.Info("Groups saved to configuration", 
		zap.Int("group_count", len(groups)),
		zap.String("config_path", configPath))

	return nil
}

// handleCreateGroup handles creating a new server group
func (a *App) handleCreateGroup() {
	a.logger.Info("Opening group creation interface")
	
	// Open the web interface for group management
	url := "http://localhost:8080/groups"
	a.openFile(url, "group management web interface")
}

// handleManageGroups handles managing existing groups
func (a *App) handleManageGroups() {
	a.logger.Info("Managing existing server groups")

	if len(a.serverGroups) == 0 {
		a.logger.Info("No groups exist to manage")
		return
	}

	// For now, just log the existing groups
	// In a full implementation, this would open a management interface
	for name, group := range a.serverGroups {
		a.logger.Info("Existing group",
			zap.String("name", name),
			zap.String("color", group.ColorEmoji),
			zap.Int("servers", len(group.ServerNames)),
			zap.Bool("enabled", group.Enabled))
	}
}

// refreshGroupsMenu refreshes the groups submenu with current groups
func (a *App) refreshGroupsMenu() {
	a.logger.Info("refreshGroupsMenu called - START", zap.Int("local_groups_count", len(a.serverGroups)))
	
	// Clear existing group menu items
	for _, item := range a.groupMenuItems {
		if item != nil {
			item.Hide()
		}
	}
	a.groupMenuItems = make(map[string]*systray.MenuItem)

	// If no local groups exist, try to sync with config groups first, then API groups
	if len(a.serverGroups) == 0 {
		a.logger.Debug("No local groups found, attempting to sync with config and API")
		
		// Check if server is running first
		if !a.server.IsRunning() {
			a.logger.Debug("Server not running yet, showing placeholder")
			noGroupsItem := a.groupManagementMenu.AddSubMenuItem("⏳ Loading groups...", "Server is starting up")
			noGroupsItem.Disable()
			a.groupMenuItems["loading"] = noGroupsItem
			return
		}
		
		// Try to get groups from config first, then fall back to API
		groupsLoaded := false
		
		// TODO: Load from config when available
		// For now, fetch from API as fallback
		if apiGroups, err := a.fetchGroupsFromAPI(); err == nil && len(apiGroups) > 0 {
			a.logger.Debug("Successfully fetched API groups, populating local groups", zap.Int("count", len(apiGroups)))
			for _, apiGroup := range apiGroups {
				if name, ok := apiGroup["name"].(string); ok {
					color, _ := apiGroup["color"].(string)
					if color == "" { color = "#007bff" }
					
					// Create local ServerGroup from API group
					a.serverGroups[name] = &ServerGroup{
						Name: name,
						Description: "Synced from API",
						Color: color,
						ColorEmoji: a.getColorEmojiForHex(color),
						ServerNames: make([]string, 0),
						Enabled: true,
					}
					a.logger.Debug("Added API group to local groups", zap.String("name", name), zap.String("color", color))
				}
			}
			groupsLoaded = true
			// Update MenuManager with new groups
			if a.menuManager != nil {
				a.menuManager.SetServerGroups(&a.serverGroups)
				// Trigger a refresh of all server menus to show the groups
				if a.syncManager != nil {
					a.syncManager.SyncDelayed()
				}
			}
		}
		
		if !groupsLoaded {
			a.logger.Error("Failed to load groups from config or API")
		}
	}

	if len(a.serverGroups) == 0 {
		// Show "no groups" message if API fetch failed
		noGroupsItem := a.groupManagementMenu.AddSubMenuItem("📋 No groups available", "Groups will appear here when server is ready")
		noGroupsItem.Disable()
		a.groupMenuItems["no_groups"] = noGroupsItem
		return
	}

	// Add separator before groups list
	if len(a.serverGroups) > 0 {
		a.groupManagementMenu.AddSeparator()
	}

	// Add each group to the menu
	for groupName, group := range a.serverGroups {
		if group.Enabled {
			groupTitle := fmt.Sprintf("%s %s (%d servers)", group.ColorEmoji, groupName, len(group.ServerNames))
			groupItem := a.groupManagementMenu.AddSubMenuItem(groupTitle, group.Description)

			// Add submenu options for this group
			addServerItem := groupItem.AddSubMenuItem("➕ Add Server", "Add a server to this group")
			editGroupItem := groupItem.AddSubMenuItem("✏️ Edit Group", "Edit group name and color")
			deleteGroupItem := groupItem.AddSubMenuItem("🗑️ Delete Group", "Delete this group")

			a.groupMenuItems[groupName] = groupItem

			// Handle group-specific actions
			go func(gName string, addServer, editGroup, deleteGroup *systray.MenuItem) {
				for {
					select {
					case <-addServer.ClickedCh:
						a.handleAddServerToGroup(gName)
					case <-editGroup.ClickedCh:
						a.handleEditGroup(gName)
					case <-deleteGroup.ClickedCh:
						a.handleDeleteGroup(gName)
					case <-a.ctx.Done():
						return
					}
				}
			}(groupName, addServerItem, editGroupItem, deleteGroupItem)
		}
	}
}

// refreshGroupedServersMenu refreshes the grouped servers submenu showing all servers organized by group
func (a *App) refreshGroupedServersMenu() {
	// Clear existing group server menu items
	for _, item := range a.groupServerMenus {
		if item != nil {
			item.Hide()
		}
	}
	a.groupServerMenus = make(map[string]*systray.MenuItem)

	// Get all servers
	allServers, err := a.stateManager.GetAllServers()
	if err != nil {
		a.logger.Error("Failed to get servers for grouped menu", zap.Error(err))
		return
	}

	if len(a.serverGroups) == 0 {
		// Show "no groups" message
		noGroupsItem := a.groupedServersMenu.AddSubMenuItem("📋 No groups created", "Create server groups to organize your servers")
		noGroupsItem.Disable()
		a.groupServerMenus["no_groups"] = noGroupsItem
		return
	}

	// Add separator before groups list
	if len(a.serverGroups) > 0 {
		a.groupedServersMenu.AddSeparator()
	}

	// Add each group with its servers
	for groupName, group := range a.serverGroups {
		if !group.Enabled {
			continue
		}

		// Count servers in this group that actually exist
		var groupServers []map[string]interface{}
		for _, serverName := range group.ServerNames {
			for _, server := range allServers {
				if name, ok := server["name"].(string); ok && name == serverName {
					groupServers = append(groupServers, server)
					break
				}
			}
		}

		groupTitle := fmt.Sprintf("%s %s (%d servers)", group.ColorEmoji, groupName, len(groupServers))
		groupItem := a.groupedServersMenu.AddSubMenuItem(groupTitle, group.Description)
		a.groupServerMenus[groupName] = groupItem

		if len(groupServers) == 0 {
			// No servers in this group
			emptyItem := groupItem.AddSubMenuItem("📭 No servers in group", "Add servers to this group")
			emptyItem.Disable()
		} else {
			// Add each server in the group
			for _, server := range groupServers {
				serverName, _ := server["name"].(string)
				enabled, _ := server["enabled"].(bool)
				quarantined, _ := server["quarantined"].(bool)
				state, _ := server["state"].(string)

				// Determine server status icon
				var statusIcon string
				var statusText string
				if quarantined {
					statusIcon = "🚨"
					statusText = "Quarantined"
				} else if !enabled {
					statusIcon = "⏸️"
					statusText = "Disabled"
				} else {
					switch state {
					case "ready":
						statusIcon = "🟢"
						statusText = "Connected"
					case "connecting":
						statusIcon = "🟡"
						statusText = "Connecting"
					case "error":
						statusIcon = "🔴"
						statusText = "Error"
					default:
						statusIcon = "⚫"
						statusText = "Disconnected"
					}
				}

				serverItemTitle := fmt.Sprintf("%s %s", statusIcon, serverName)
				serverItem := groupItem.AddSubMenuItem(serverItemTitle, fmt.Sprintf("Server status: %s", statusText))

				// Add server action options
				toggleItem := serverItem.AddSubMenuItem("🔄 Toggle Enable/Disable", "Enable or disable this server")
				viewItem := serverItem.AddSubMenuItem("👁️ View Details", "View server information")

				// Add repository link if available
				if repoURL, hasRepo := server["repository_url"].(string); hasRepo && repoURL != "" {
					repoItem := serverItem.AddSubMenuItem("🔗 Repository", "Open server repository")

					// Handle repository link clicks
					go func(url string) {
						for {
							select {
							case <-repoItem.ClickedCh:
								a.openFile(url, "repository")
							case <-a.ctx.Done():
								return
							}
						}
					}(repoURL)
				}

				// Handle server action clicks
				go func(sName string, toggle, view *systray.MenuItem) {
					for {
						select {
						case <-toggle.ClickedCh:
							a.handleServerAction(sName, "toggle")
						case <-view.ClickedCh:
							a.handleServerAction(sName, "view")
						case <-a.ctx.Done():
							return
						}
					}
				}(serverName, toggleItem, viewItem)
			}
		}
	}

	// Add ungrouped servers if any exist
	var ungroupedServers []map[string]interface{}
	for _, server := range allServers {
		serverName, _ := server["name"].(string)
		isGrouped := false

		// Check if server belongs to any group
		for _, group := range a.serverGroups {
			for _, groupedServerName := range group.ServerNames {
				if groupedServerName == serverName {
					isGrouped = true
					break
				}
			}
			if isGrouped {
				break
			}
		}

		if !isGrouped {
			ungroupedServers = append(ungroupedServers, server)
		}
	}

	if len(ungroupedServers) > 0 {
		a.groupedServersMenu.AddSeparator()
		ungroupedTitle := fmt.Sprintf("📂 Ungrouped Servers (%d)", len(ungroupedServers))
		ungroupedItem := a.groupedServersMenu.AddSubMenuItem(ungroupedTitle, "Servers not assigned to any group")
		a.groupServerMenus["ungrouped"] = ungroupedItem

		for _, server := range ungroupedServers {
			serverName, _ := server["name"].(string)
			enabled, _ := server["enabled"].(bool)
			quarantined, _ := server["quarantined"].(bool)
			state, _ := server["state"].(string)

			// Determine server status icon
			var statusIcon string
			var statusText string
			if quarantined {
				statusIcon = "🚨"
				statusText = "Quarantined"
			} else if !enabled {
				statusIcon = "⏸️"
				statusText = "Disabled"
			} else {
				switch state {
				case "ready":
					statusIcon = "🟢"
					statusText = "Connected"
				case "connecting":
					statusIcon = "🟡"
					statusText = "Connecting"
				case "error":
					statusIcon = "🔴"
					statusText = "Error"
				default:
					statusIcon = "⚫"
					statusText = "Disconnected"
				}
			}

			serverItemTitle := fmt.Sprintf("%s %s", statusIcon, serverName)
			serverItem := ungroupedItem.AddSubMenuItem(serverItemTitle, fmt.Sprintf("Server status: %s", statusText))

			// Add server action options
			toggleItem := serverItem.AddSubMenuItem("🔄 Toggle Enable/Disable", "Enable or disable this server")
			viewItem := serverItem.AddSubMenuItem("👁️ View Details", "View server information")

			// Add repository link if available
			if repoURL, hasRepo := server["repository_url"].(string); hasRepo && repoURL != "" {
				repoItem := serverItem.AddSubMenuItem("🔗 Repository", "Open server repository")

				// Handle repository link clicks
				go func(url string) {
					for {
						select {
						case <-repoItem.ClickedCh:
							a.openFile(url, "repository")
						case <-a.ctx.Done():
							return
						}
					}
				}(repoURL)
			}

			// Handle server action clicks
			go func(sName string, toggle, view *systray.MenuItem) {
				for {
					select {
					case <-toggle.ClickedCh:
						a.handleServerAction(sName, "toggle")
					case <-view.ClickedCh:
						a.handleServerAction(sName, "view")
					case <-a.ctx.Done():
						return
					}
				}
			}(serverName, toggleItem, viewItem)
		}
	}
}

// handleAddServerToGroup handles adding a server to a specific group
func (a *App) handleAddServerToGroup(groupName string) {
	a.logger.Info("Adding server to group", zap.String("group", groupName))

	group, exists := a.serverGroups[groupName]
	if !exists {
		a.logger.Error("Group not found", zap.String("group", groupName))
		return
	}

	// Get available servers
	allServers, err := a.stateManager.GetAllServers()
	if err != nil {
		a.logger.Error("Failed to get servers for group assignment", zap.Error(err))
		return
	}

	// For demonstration, add the first available server that's not already in any group
	for _, server := range allServers {
		if name, ok := server["name"].(string); ok {
			// Check if server is already in this group
			alreadyInGroup := false
			for _, existingServer := range group.ServerNames {
				if existingServer == name {
					alreadyInGroup = true
					break
				}
			}

			if !alreadyInGroup {
				// Add server to group
				group.ServerNames = append(group.ServerNames, name)
				a.logger.Info("Added server to group",
					zap.String("server", name),
					zap.String("group", groupName))

				// Refresh the menu to show updated server count
				a.refreshGroupsMenu()
				break
			}
		}
	}
}

// handleEditGroup handles editing a group's properties
func (a *App) handleEditGroup(groupName string) {
	a.logger.Info("Opening edit menu for group", zap.String("group", groupName))

	group, exists := a.serverGroups[groupName]
	if !exists {
		a.logger.Error("Group not found for editing", zap.String("group", groupName))
		return
	}

	// Open the edit submenu for this group
	a.openGroupEditMenu(groupName, group)
}

// openGroupEditMenu opens a detailed edit menu for a group
func (a *App) openGroupEditMenu(groupName string, group *ServerGroup) {
	// Clear the main groups menu temporarily to show edit interface
	if a.groupManagementMenu != nil {
		// Hide existing items
		for _, item := range a.groupMenuItems {
			if item != nil {
				item.Hide()
			}
		}
		a.groupMenuItems = make(map[string]*systray.MenuItem)
	}

	// Create edit interface
	editTitle := a.groupManagementMenu.AddSubMenuItem(
		fmt.Sprintf("✏️ Editing: %s %s", group.ColorEmoji, groupName),
		"Edit group properties")
	editTitle.Disable()

	a.groupManagementMenu.AddSeparator()

	// Name editing - show current name and options to change
	nameSection := a.groupManagementMenu.AddSubMenuItem("📝 Change Name", "Edit the group name")
	currentNameItem := nameSection.AddSubMenuItem(
		fmt.Sprintf("Current: %s", groupName),
		"Current group name")
	currentNameItem.Disable()

	// Predefined name suggestions
	nameSection.AddSeparator()
	nameOptions := []string{"Work", "Personal", "Development", "Production", "Testing", "AWS", "Databases", "AI/ML", "Monitoring", "Custom-" + fmt.Sprintf("%d", len(a.serverGroups)+1)}

	for _, nameOption := range nameOptions {
		if nameOption != groupName { // Don't show current name as option
			nameItem := nameSection.AddSubMenuItem("→ " + nameOption, fmt.Sprintf("Rename group to '%s'", nameOption))

			go func(oldName, newName string, item *systray.MenuItem) {
				for range item.ClickedCh {
					a.handleRenameGroup(oldName, newName)
					// Return to main groups menu after rename
					a.refreshGroupsMenu()
				}
			}(groupName, nameOption, nameItem)
		}
	}

	// Color editing
	colorSection := a.groupManagementMenu.AddSubMenuItem("🎨 Change Color", "Change the group color")
	currentColorItem := colorSection.AddSubMenuItem(
		fmt.Sprintf("Current: %s %s", group.ColorEmoji, getColorName(group.ColorEmoji)),
		"Current group color")
	currentColorItem.Disable()

	colorSection.AddSeparator()

	// Show all available colors except current one
	for _, colorOption := range GroupColors {
		if colorOption.Emoji != group.ColorEmoji {
			colorItem := colorSection.AddSubMenuItem(
				fmt.Sprintf("%s %s", colorOption.Emoji, colorOption.Name),
				fmt.Sprintf("Change color to %s", colorOption.Name))

			go func(gName string, color struct{ Emoji, Name, Code string }, item *systray.MenuItem) {
				for range item.ClickedCh {
					a.handleChangeGroupColor(gName, color)
					// Return to main groups menu after color change
					a.refreshGroupsMenu()
				}
			}(groupName, colorOption, colorItem)
		}
	}

	// Action buttons
	a.groupManagementMenu.AddSeparator()

	// Done button to return to main menu
	doneItem := a.groupManagementMenu.AddSubMenuItem("✅ Done", "Finish editing and return to main menu")
	go func(item *systray.MenuItem) {
		for range item.ClickedCh {
			a.refreshGroupsMenu()
		}
	}(doneItem)

	// Delete button
	deleteItem := a.groupManagementMenu.AddSubMenuItem("🗑️ Delete Group", fmt.Sprintf("Delete group '%s'", groupName))
	go func(gName string, item *systray.MenuItem) {
		for range item.ClickedCh {
			a.handleDeleteGroup(gName)
		}
	}(groupName, deleteItem)
}

// getColorName returns the color name for a given emoji
func getColorName(emoji string) string {
	for _, color := range GroupColors {
		if color.Emoji == emoji {
			return color.Name
		}
	}
	return "Unknown"
}

// handleRenameGroup renames a group
func (a *App) handleRenameGroup(oldName, newName string) {
	a.logger.Info("Renaming group", zap.String("old_name", oldName), zap.String("new_name", newName))

	// Check if old group exists
	group, exists := a.serverGroups[oldName]
	if !exists {
		a.logger.Error("Group not found for renaming", zap.String("old_name", oldName))
		return
	}

	// Check if new name already exists
	if _, exists := a.serverGroups[newName]; exists {
		a.logger.Warn("Group name already exists", zap.String("new_name", newName))
		return
	}

	// Update group name
	group.Name = newName
	group.Description = fmt.Sprintf("Custom group for organizing servers - %s", getColorName(group.ColorEmoji))

	// Move group in map
	a.serverGroups[newName] = group
	delete(a.serverGroups, oldName)

	// Save groups to configuration file
	if err := a.saveGroupsToConfig(); err != nil {
		a.logger.Error("Failed to save groups to configuration", zap.Error(err))
	}

	a.logger.Info("Group renamed successfully",
		zap.String("old_name", oldName),
		zap.String("new_name", newName))

	// Refresh menus to show changes
	if a.syncManager != nil {
		a.syncManager.SyncDelayed() // Refresh server menus to update group references
	}
}

// handleChangeGroupColor changes a group's color
func (a *App) handleChangeGroupColor(groupName string, newColor struct{ Emoji, Name, Code string }) {
	a.logger.Info("Changing group color",
		zap.String("group", groupName),
		zap.String("new_color", newColor.Name))

	group, exists := a.serverGroups[groupName]
	if !exists {
		a.logger.Error("Group not found for color change", zap.String("group", groupName))
		return
	}

	// Update group color
	group.Color = newColor.Code
	group.ColorEmoji = newColor.Emoji
	group.Description = fmt.Sprintf("Custom group for organizing servers - %s", newColor.Name)

	// Save groups to configuration file
	if err := a.saveGroupsToConfig(); err != nil {
		a.logger.Error("Failed to save groups to configuration", zap.Error(err))
	}

	a.logger.Info("Group color changed successfully",
		zap.String("group", groupName),
		zap.String("new_color", newColor.Name),
		zap.String("new_emoji", newColor.Emoji))

	// Refresh menus to show changes
	if a.syncManager != nil {
		a.syncManager.SyncDelayed() // Refresh server menus to show new colors
	}
}

// handleDeleteGroup handles deleting a group
func (a *App) handleDeleteGroup(groupName string) {
	a.logger.Info("Deleting group", zap.String("group", groupName))

	// Remove from groups map
	delete(a.serverGroups, groupName)

	// Remove from menu items
	if item, exists := a.groupMenuItems[groupName]; exists {
		item.Hide()
		delete(a.groupMenuItems, groupName)
	}

	// Save groups to configuration file
	if err := a.saveGroupsToConfig(); err != nil {
		a.logger.Error("Failed to save groups to configuration", zap.Error(err))
	}

	a.logger.Info("Group deleted successfully", zap.String("group", groupName))

	// Refresh the menu
	a.refreshGroupsMenu()
}

// handleAssignServerToGroup assigns a server to a specific group
func (a *App) handleAssignServerToGroup(serverName, groupName string) error {
	a.logger.Info("Assigning server to group", zap.String("server", serverName), zap.String("group", groupName))

	// Check if group exists
	group, exists := a.serverGroups[groupName]
	if !exists {
		return fmt.Errorf("group '%s' not found", groupName)
	}

	// Remove server from any existing groups first
	a.removeServerFromAllGroups(serverName)

	// Add server to the target group
	group.ServerNames = append(group.ServerNames, serverName)

	// Save groups to configuration file
	if err := a.saveGroupsToConfig(); err != nil {
		a.logger.Error("Failed to save groups to configuration", zap.Error(err))
	}

	a.logger.Info("Server assigned to group successfully",
		zap.String("server", serverName),
		zap.String("group", groupName),
		zap.Int("group_size", len(group.ServerNames)))

	// Refresh menus to show changes
	a.refreshGroupsMenu()
	if a.syncManager != nil {
		a.syncManager.SyncDelayed() // Refresh server menus to show group colors
	}

	return nil
}

// handleRemoveServerFromGroup removes a server from a specific group
func (a *App) handleRemoveServerFromGroup(serverName, groupName string) error {
	a.logger.Info("Removing server from group", zap.String("server", serverName), zap.String("group", groupName))

	// Check if group exists
	group, exists := a.serverGroups[groupName]
	if !exists {
		return fmt.Errorf("group '%s' not found", groupName)
	}

	// Remove server from the group
	for i, existingServer := range group.ServerNames {
		if existingServer == serverName {
			// Remove server from slice
			group.ServerNames = append(group.ServerNames[:i], group.ServerNames[i+1:]...)
			break
		}
	}

	// Save groups to configuration file
	if err := a.saveGroupsToConfig(); err != nil {
		a.logger.Error("Failed to save groups to configuration", zap.Error(err))
	}

	a.logger.Info("Server removed from group successfully",
		zap.String("server", serverName),
		zap.String("group", groupName),
		zap.Int("group_size", len(group.ServerNames)))

	// Refresh menus to show changes
	a.refreshGroupsMenu()
	if a.syncManager != nil {
		a.syncManager.SyncDelayed() // Refresh server menus to remove group colors
	}

	return nil
}

// removeServerFromAllGroups removes a server from all groups (helper function)
func (a *App) removeServerFromAllGroups(serverName string) {
	for _, group := range a.serverGroups {
		for i, existingServer := range group.ServerNames {
			if existingServer == serverName {
				// Remove server from slice
				group.ServerNames = append(group.ServerNames[:i], group.ServerNames[i+1:]...)
				break
			}
		}
	}
}

// handleServerConfiguration opens the configuration dialog for a server
func (a *App) handleServerConfiguration(serverName string) error {
	a.logger.Info("Opening configuration dialog for server", zap.String("server", serverName))

	// Notify sync manager of user activity for adaptive frequency
	if a.syncManager != nil {
		a.syncManager.NotifyUserActivity()
	}

	// Get current server configuration
	allServers, err := a.stateManager.GetAllServers()
	if err != nil {
		return fmt.Errorf("failed to get servers for configuration: %w", err)
	}

	var targetServer *config.ServerConfig
	for _, srv := range allServers {
		if name, ok := srv["name"].(string); ok && name == serverName {
			// Convert map back to ServerConfig
			serverJSON, err := json.Marshal(srv)
			if err != nil {
				return fmt.Errorf("failed to marshal server data: %w", err)
			}

			targetServer = &config.ServerConfig{}
			if err := json.Unmarshal(serverJSON, targetServer); err != nil {
				return fmt.Errorf("failed to unmarshal server data: %w", err)
			}
			break
		}
	}

	if targetServer == nil {
		return fmt.Errorf("server '%s' not found", serverName)
	}

	// Create and show configuration dialog
	dialog := NewServerConfigDialog(a.logger, targetServer, serverName)
	
	// Set diagnostic agent and server manager
	dialog.diagnosticAgent = a.diagnosticAgent
	if a.server != nil {
		dialog.serverManager = a.server
	}

	// Define save callback
	onSave := func(updatedServer *config.ServerConfig) error {
		a.logger.Info("Saving server configuration",
			zap.String("old_name", serverName),
			zap.String("new_name", updatedServer.Name))

		// Load current configuration
		configPath := a.server.GetConfigPath()
		if configPath == "" {
			return fmt.Errorf("config path not available")
		}

		globalConfig, err := config.LoadFromFile(configPath)
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		// Find and update the server in the configuration
		var found bool
		for i, server := range globalConfig.Servers {
			if server.Name == serverName {
				// Update the server configuration
				globalConfig.Servers[i] = updatedServer
				found = true
				break
			}
		}

		if !found {
			return fmt.Errorf("server '%s' not found in configuration", serverName)
		}

		// Save the updated configuration
		if err := config.SaveToFile(globalConfig, configPath); err != nil {
			return fmt.Errorf("failed to save configuration: %w", err)
		}

		// Trigger configuration reload in the server
		if err := a.server.ReloadConfiguration(); err != nil {
			a.logger.Warn("Failed to reload configuration after update", zap.Error(err))
			// Don't return error here as the save operation succeeded
		}

		a.logger.Info("Server configuration saved and reloaded successfully",
			zap.String("server", updatedServer.Name))

		return nil
	}

	// Define cancel callback
	onCancel := func() {
		a.logger.Info("Server configuration dialog cancelled", zap.String("server", serverName))
	}

	// Show the dialog
	return dialog.Show(a.ctx, onSave, onCancel)
}
// fetchGroupsFromAPI fetches groups from the web interface API
func (a *App) fetchGroupsFromAPI() ([]map[string]interface{}, error) {
	listenAddr := a.server.GetListenAddress()
	if listenAddr == "" {
		return nil, fmt.Errorf("server listen address not available")
	}
	
	// Ensure we have a proper URL
	if !strings.HasPrefix(listenAddr, "http") {
		listenAddr = "http://localhost" + listenAddr
	}
	
	url := listenAddr + "/api/groups"
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch groups from %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	success, ok := response["success"].(bool)
	if !ok || !success {
		return nil, fmt.Errorf("API returned error: %v", response["error"])
	}

	groups, ok := response["groups"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid groups data in response")
	}

	result := make([]map[string]interface{}, len(groups))
	for i, group := range groups {
		if groupMap, ok := group.(map[string]interface{}); ok {
			result[i] = groupMap
		}
	}

	return result, nil
}

// loadGroupsForServerMenus loads groups from API for server assignment menus
func (a *App) loadGroupsForServerMenus() {
	a.logger.Info("loadGroupsForServerMenus called - START")
	
	// Check if server is running first
	if !a.server.IsRunning() {
		a.logger.Info("Server not running yet, scheduling delayed group load")
		go func() {
			time.Sleep(5 * time.Second)
			a.loadGroupsForServerMenus()
		}()
		return
	}
	
	// Fetch API groups and populate local groups
	if apiGroups, err := a.fetchGroupsFromAPI(); err == nil && len(apiGroups) > 0 {
		a.logger.Info("Successfully fetched API groups for server menus", zap.Int("count", len(apiGroups)))
		for _, apiGroup := range apiGroups {
			if name, ok := apiGroup["name"].(string); ok {
				color, _ := apiGroup["color"].(string)
				if color == "" { color = "#007bff" }
				
				// Create local ServerGroup from API group
				a.serverGroups[name] = &ServerGroup{
					Name: name,
					Description: "Available for server assignment",
					Color: color,
					ColorEmoji: a.getColorEmojiForHex(color),
					ServerNames: make([]string, 0),
					Enabled: true,
				}
				a.logger.Info("Added API group for server assignment", zap.String("name", name), zap.String("color", color))
			}
		}
		
		// Update MenuManager with new groups
		if a.menuManager != nil {
			a.menuManager.SetServerGroups(&a.serverGroups)
			a.logger.Info("Updated MenuManager with groups", zap.Int("group_count", len(a.serverGroups)))
			
			// Trigger a refresh of all server menus to show the groups
			if a.syncManager != nil {
				a.syncManager.SyncDelayed()
				a.logger.Info("Triggered delayed sync to refresh server menus with groups")
			}
		}
	} else {
		a.logger.Error("Failed to fetch API groups for server menus", zap.Error(err))
	}
}

func (a *App) syncWithAPIGroups() {
	apiGroups, err := a.fetchGroupsFromAPI()
	if err != nil {
		a.logger.Error("Failed to sync with API groups", zap.Error(err))
		return
	}

	// Convert API groups to tray groups
	for _, apiGroup := range apiGroups {
		name, ok := apiGroup["name"].(string)
		if !ok {
			continue
		}
		
		color, ok := apiGroup["color"].(string)
		if !ok {
			color = "#007bff" // Default color
		}

		// Create tray group from API group
		newGroup := &ServerGroup{
			Name:        name,
			Description: fmt.Sprintf("Synced from API: %s", name),
			Color:       color,
			ColorEmoji:  a.getColorEmoji(color),
			ServerNames: make([]string, 0),
			Enabled:     true,
		}

		a.serverGroups[name] = newGroup
	}

	a.logger.Info("Synchronized with API groups", zap.Int("count", len(apiGroups)))
}

// getColorEmoji returns an emoji for a given hex color
func (a *App) getColorEmoji(hexColor string) string {
	// Map common hex colors to emojis
	colorMap := map[string]string{
		"#ff9900": "🟠", // AWS Orange
		"#28a745": "🟢", // Green
		"#dc3545": "🔴", // Red
		"#007bff": "🔵", // Blue
		"#6f42c1": "🟣", // Purple
		"#fd7e14": "🟠", // Orange
		"#20c997": "🟢", // Teal (green)
		"#e83e8c": "🟣", // Pink (purple)
		"#6c757d": "⚫", // Gray (black)
		"#343a40": "⚫", // Dark
	}

	// Convert to lowercase for comparison
	hexColor = strings.ToLower(hexColor)
	
	if emoji, exists := colorMap[hexColor]; exists {
		return emoji
	}
	
	// Default to blue circle for unknown colors
	return "🔵"
}

// fetchServerAssignments fetches server-to-group assignments
func (a *App) fetchServerAssignments() (map[string]string, error) {
	// For now, return empty assignments since we don't have persistent storage yet
	// TODO: Implement actual assignment fetching from the groups API
	return make(map[string]string), nil
}

// assignServerToGroup assigns a server to a group
func (a *App) assignServerToGroup(serverName, groupName string) {
	a.logger.Info("Assigning server to group", zap.String("server", serverName), zap.String("group", groupName))
	
	// TODO: Implement actual server-to-group assignment
	// For now, just log the action
	
	a.logger.Info("Server assignment completed", zap.String("server", serverName), zap.String("group", groupName))
}

// ServerState represents the state of a server before stop
type ServerState struct {
	Name      string `json:"name"`
	Enabled   bool   `json:"enabled"`
	Connected bool   `json:"connected"`
}

// saveServerStatesForStop saves current server states before stopping
func (a *App) saveServerStatesForStop() error {
	if a.stateManager == nil {
		return nil
	}

	// Get all current servers
	servers, err := a.stateManager.GetAllServers()
	if err != nil {
		return err
	}

	// Create state snapshot
	states := make([]ServerState, 0, len(servers))
	for _, server := range servers {
		name, _ := server["name"].(string)
		enabled, _ := server["enabled"].(bool)
		connected, _ := server["connected"].(bool)

		states = append(states, ServerState{
			Name:      name,
			Enabled:   enabled,
			Connected: connected,
		})
	}

	// Save to temporary file
	stateFile := filepath.Join(os.TempDir(), "mcpproxy_server_states.json")
	data, err := json.Marshal(states)
	if err != nil {
		return err
	}

	if err := os.WriteFile(stateFile, data, 0644); err != nil {
		return err
	}

	a.logger.Info("Server states saved for stop operation", 
		zap.String("file", stateFile), 
		zap.Int("servers", len(states)))

	return nil
}

// restoreServerStatesAfterStart restores server states after starting
func (a *App) restoreServerStatesAfterStart() error {
	stateFile := filepath.Join(os.TempDir(), "mcpproxy_server_states.json")
	
	// Check if state file exists
	if _, err := os.Stat(stateFile); os.IsNotExist(err) {
		a.logger.Debug("No server states file found, skipping restoration")
		return nil
	}

	// Read saved states
	data, err := os.ReadFile(stateFile)
	if err != nil {
		return err
	}

	var states []ServerState
	if err := json.Unmarshal(data, &states); err != nil {
		return err
	}

	a.logger.Info("Restoring server states after start", zap.Int("servers", len(states)))

	// Wait a moment for server to be fully ready
	go func() {
		// Give the server time to initialize
		time.Sleep(3 * time.Second)

		// Trigger a sync to restore the UI to the saved states
		if a.syncManager != nil {
			if err := a.syncManager.SyncNow(); err != nil {
				a.logger.Error("Failed to sync after state restoration", zap.Error(err))
			} else {
				a.logger.Info("Server states restored successfully")
			}
		}

		// Clean up the state file
		if err := os.Remove(stateFile); err != nil {
			a.logger.Warn("Failed to remove state file", zap.Error(err))
		}
	}()

	return nil
}

// getColorEmojiForHex returns emoji for hex color
func (a *App) getColorEmojiForHex(hex string) string {
	switch strings.ToLower(hex) {
	case "#ff9900": return "🟠" // AWS Orange
	case "#28a745": return "🟢" // Green  
	case "#dc3545": return "🔴" // Red
	default: return "🔵" // Blue
	}
}

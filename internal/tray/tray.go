//go:build !nogui && !headless && !linux

package tray

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
	"hash/fnv"
	"strconv"

	"github.com/getlantern/systray"
	"github.com/fsnotify/fsnotify"
	"github.com/inconshreveable/go-update"
	"go.uber.org/zap"
	"golang.org/x/mod/semver"

	"mcpproxy-go/internal/events"

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
	ID          int      `json:"id"`
	Name        string   `json:"name"`
	Icon        string   `json:"icon_emoji"`  // Icon emoji for display
	Color       string   `json:"color"`       // Color hex code
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
	StopUpstreamServer(serverName string) error   // Stop individual upstream server (sets Stopped=true)
	UnstopUpstreamServer(serverName string) error // Unstop individual upstream server (sets Stopped=false)
	GetAllServers() ([]map[string]interface{}, error)
	GetServerTools(serverName string) ([]map[string]interface{}, error)

	// Config management for file watching
	ReloadConfiguration() error
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

// App represents the system tray application
type App struct {
	server    ServerInterface
	logger    *zap.SugaredLogger
	version   string
	buildTime string
	shutdown  func()

	// Menu items for dynamic updates
	statusItem       *systray.MenuItem
	stopStartAllItem *systray.MenuItem
	serverCountItem  *systray.MenuItem

	// Status-based server menus
	connectedServersMenu    *systray.MenuItem
	disconnectedServersMenu *systray.MenuItem
	sleepingServersMenu     *systray.MenuItem
	disabledServersMenu     *systray.MenuItem
	autoDisabledServersMenu *systray.MenuItem
	quarantineMenu          *systray.MenuItem

	// Legacy upstream menu (for backward compatibility if needed)
	// upstreamServersMenu *systray.MenuItem - REMOVED

	// Managers for proper synchronization
	stateManager    *ServerStateManager
	menuManager     *MenuManager
	syncManager     *SynchronizationManager
	eventManager    *EventManager // Event-based synchronization
	diagnosticAgent *DiagnosticAgent

	// Chat system for multi-agent diagnostics
	chatSystem *ChatSystem

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
	resourceMonitorMenu *systray.MenuItem            // Resource monitor menu
	serverGroups        map[string]*ServerGroup      // Custom server groups
	groupMenuItems      map[string]*systray.MenuItem // Group menu items
}

// New creates a new tray application
func New(server ServerInterface, logger *zap.SugaredLogger, version string, buildTime string, shutdown func()) *App {
	app := &App{
		server:    server,
		logger:    logger,
		version:   version,
		buildTime: buildTime,
		shutdown:  shutdown,
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

	return app
}

// Helper functions for group ID management
func (a *App) getGroupByID(id int) *ServerGroup {
	for _, group := range a.serverGroups {
		if group.ID == id {
			return group
		}
	}
	return nil
}

func (a *App) getGroupByName(name string) *ServerGroup {
	for _, group := range a.serverGroups {
		if group.Name == name {
			return group
		}
	}
	return nil
}

func (a *App) getNextGroupID() int {
	maxID := 0
	for _, group := range a.serverGroups {
		if group.ID > maxID {
			maxID = group.ID
		}
	}
	return maxID + 1
}

func (a *App) getGroupKeyByID(id int) string {
	for key, group := range a.serverGroups {
		if group.ID == id {
			return key
		}
	}
	return ""
}

// Run starts the system tray application
// CRITICAL: On macOS Tahoe, systray.Run() must be called IMMEDIATELY without
// any goroutines or complex setup beforehand. All initialization is deferred
// to onReady() callback to ensure proper GUI event loop initialization.
func (a *App) Run(ctx context.Context) error {
	a.logger.Info("Starting system tray application - calling systray.Run() immediately")
	a.ctx = ctx // Store context reference for use in onReady

	// Start systray FIRST - this is a blocking call that must run on main thread
	// All other initialization happens in onReady() callback
	systray.Run(a.onReady, a.onExit)

	return ctx.Err()
}

// startBackgroundServices initializes watchers and goroutines AFTER systray is running
// This is called from onReady() to ensure GUI is fully initialized first
func (a *App) startBackgroundServices() {
	// Create cancellable context from stored parent context
	a.ctx, a.cancel = context.WithCancel(a.ctx)

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
			case <-a.ctx.Done():
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
				case <-a.ctx.Done():
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
				case <-a.ctx.Done():
					return
				}
			}
		}()
	}

	// Monitor context cancellation and quit systray when needed
	go func() {
		<-a.ctx.Done()
		a.logger.Info("Context cancelled, quitting systray")
		a.cleanup()
		systray.Quit()
	}()

	a.logger.Info("Background services started successfully")
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
	a.logger.Info("🎨 TRAY: onReady() called - starting icon setup", zap.Int("icon_bytes", len(iconData)))

	if len(iconData) == 0 {
		a.logger.Error("🚨 CRITICAL: Icon data is EMPTY!")
	} else {
		a.logger.Info("✅ Icon data loaded successfully", zap.Int("size", len(iconData)))
	}

	// On macOS 26.1 Tahoe, set title FIRST to force NSStatusItem creation
	// This is a workaround for potential initialization timing issues
	if runtime.GOOS == osDarwin {
		a.logger.Info("🍎 macOS Tahoe workaround: Setting title first to force status item visibility")
		systray.SetTitle("")
		a.logger.Info("✅ systray.SetTitle('') called FIRST for macOS visibility (icon only)")
		// Small delay to let NSStatusItem stabilize
		time.Sleep(100 * time.Millisecond)
	}

	systray.SetIcon(iconData)
	a.logger.Info("✅ systray.SetIcon() called")

	// On macOS, also set as template icon for better system integration
	if runtime.GOOS == osDarwin {
		systray.SetTemplateIcon(iconData, iconData)
		a.logger.Info("✅ systray.SetTemplateIcon() called for macOS")
		// Small delay after icon setup to ensure rendering
		time.Sleep(50 * time.Millisecond)
		// Force title update again after icon to ensure visibility
		systray.SetTitle("")
		a.logger.Info("✅ systray.SetTitle('') called again after icon for visibility (icon only)")
	}
	a.updateServerCountFromConfig()

	// --- Initialize Menu Items ---
	a.logger.Debug("Initializing tray menu items")
	a.statusItem = systray.AddMenuItem("Status: Initializing...", "")
	a.statusItem.Disable() // Initially disabled as it's just for display
	a.stopStartAllItem = systray.AddMenuItem("⏸️ Stop All Servers", "Stop all running servers (keeps MCPProxy running)")
	a.serverCountItem = systray.AddMenuItem("📊 Servers: Loading...", "")
	a.serverCountItem.Disable() // Display only

	// Mark core menu items as ready - this will release waiting goroutines
	a.coreMenusReady = true
	a.logger.Debug("Core menu items initialized successfully - background processes can now start")

	// Start background services NOW that systray is initialized and core menus are ready
	// This was moved from Run() to ensure systray.Run() is called first on macOS Tahoe
	a.startBackgroundServices()

	systray.AddSeparator()

	// --- Status-Based Server Menus ---
	a.connectedServersMenu = systray.AddMenuItem("🟢 Connected Servers", "")
	a.disconnectedServersMenu = systray.AddMenuItem("🔴 Disconnected Servers", "")
	a.sleepingServersMenu = systray.AddMenuItem("💤 Sleeping Servers", "Servers with lazy loading enabled")
	a.disabledServersMenu = systray.AddMenuItem("⏹️ Disabled Servers", "Manually disabled servers")
	a.autoDisabledServersMenu = systray.AddMenuItem("🚫 Auto-Disabled Servers", "Servers automatically disabled due to failures")
	a.quarantineMenu = systray.AddMenuItem("🔒 Quarantined Servers", "")
	systray.AddSeparator()

	// --- Group Management Menu ---
	a.groupManagementMenu = systray.AddMenuItem("🌐 Group Management", "")

	// --- MCPProxy Management Menu ---
	a.resourceMonitorMenu = systray.AddMenuItem("📊 MCPProxy Management", "View system resources and metrics")
	systray.AddSeparator()

	// --- Initialize Managers ---
	a.menuManager = NewMenuManager(a.connectedServersMenu, a.disconnectedServersMenu, a.sleepingServersMenu, a.disabledServersMenu, a.autoDisabledServersMenu, a.quarantineMenu, nil, a.logger)
	a.syncManager = NewSynchronizationManager(a.stateManager, a.menuManager, a.logger)
	a.diagnosticAgent = NewDiagnosticAgent(a.logger.Desugar(), a.server.GetLLMConfig())

	// Initialize event-based synchronization
	if eventBus := a.server.GetEventBus(); eventBus != nil {
		a.logger.Info("Event-based synchronization enabled")
		a.eventManager = NewEventManager(eventBus, a.syncManager, a.menuManager, a.logger)
		a.logger.Info("EventManager initialized successfully")
	} else {
		a.logger.Warn("Server does not provide EventBus, event-based sync not available")
	}

	// Initialize chat system
	storage, err := NewFileChatStorage(a.logger.Desugar(), "")
	if err != nil {
		a.logger.Warnf("Failed to initialize chat storage: %v", err)
	}

	// Get LLM configuration from server
	llmConfig := a.server.GetLLMConfig()
	a.chatSystem = NewChatSystem(a.logger.Desugar(), storage, llmConfig, a.server)

	// --- Set Action Callback ---
	// Centralized action handler for all menu-driven server actions
	a.menuManager.SetActionCallback(a.handleServerAction)

	// --- Set Server Count Callback ---
	// Set callback but with protection against smaller counts
	a.menuManager.SetServerCountCallback(a.updateServerCountDisplay)

	// --- Load Groups for Server Menus ---
	a.logger.Info("TRAY INIT: Loading groups for server assignment menus")
	a.loadGroupsForServerMenus()

	// --- Set Server Groups Reference ---
	// Allow MenuManager to access server groups for color display
	a.menuManager.SetServerGroups(&a.serverGroups)

	// --- Initialize Server Count Display ---
	// Load initial server count from config
	a.updateServerCountFromConfig()

    // --- Other Menu Items ---
    openConfigItem := systray.AddMenuItem("Open config dir", "")
    editConfigItem := systray.AddMenuItem("Edit config", "")
    reloadConfigItem := systray.AddMenuItem("🔄 Reload Config", "")
    openLogsItem := systray.AddMenuItem("Open logs dir", "")
    githubItem := systray.AddMenuItem("🔗 GitHub Repository", "")
    // Startup script submenu
    startupMenu := systray.AddMenuItem("🚀 Startup Script", "Manage startup script")
    startupStatusItem := startupMenu.AddSubMenuItem("Status: Loading...", "")
    startupStartItem := startupMenu.AddSubMenuItem("Start", "")
    startupStopItem := startupMenu.AddSubMenuItem("Stop", "")
    startupRestartItem := startupMenu.AddSubMenuItem("Restart", "")
    startupEditItem := startupMenu.AddSubMenuItem("✏️ Edit Script", "Open startup script file for editing")

	// Version information
	versionTitle := fmt.Sprintf("ℹ️ Version %s", a.version)
	if a.buildTime != "unknown" && a.buildTime != "" {
		versionTitle = fmt.Sprintf("ℹ️ Version %s (%s)", a.version, a.buildTime)
	}
	versionItem := systray.AddMenuItem(versionTitle, "")
	versionItem.Disable() // Display only
	systray.AddSeparator()

	// --- Autostart Menu Item (macOS only) ---
	if runtime.GOOS == osDarwin && a.autostartManager != nil {
		a.autostartItem = systray.AddMenuItem("Start at Login", "")
		a.updateAutostartMenuItem()
		systray.AddSeparator()
	}

	quitItem := systray.AddMenuItem("Quit", "")

	// --- Set Initial State & Start Sync ---
	a.updateStatus()

	if err := a.syncManager.SyncNow(); err != nil {
		a.logger.Error("Initial menu sync failed", zap.Error(err))
	}

	a.syncManager.Start()

    // Helper to refresh startup status
    refreshStartup := func() {
        if a.server != nil {
            status := a.server.GetStartupScriptStatus()
            title := "Status: Disabled"
            pathVal := ""
            if status != nil {
                if enabled, ok := status["enabled"].(bool); ok && enabled {
                    if running, ok2 := status["running"].(bool); ok2 && running {
                        title = "Status: Running"
                    } else {
                        title = "Status: Stopped"
                    }
                }
                if p, ok := status["path"].(string); ok { pathVal = p }
            }
            startupStatusItem.SetTitle(title)
            if pathVal != "" {
                startupEditItem.Enable()
            } else {
                startupEditItem.Disable()
            }
        }
    }
    refreshStartup()

    // --- Click Handlers ---
	go func() {
		for {
			select {
			case <-a.stopStartAllItem.ClickedCh:
				a.handleStopStartAll()
			case <-openConfigItem.ClickedCh:
				a.openConfigDir()
			case <-editConfigItem.ClickedCh:
				a.editConfigFile()
			case <-reloadConfigItem.ClickedCh:
				a.handleReloadConfig()
			case <-openLogsItem.ClickedCh:
				a.openLogsDir()
            case <-githubItem.ClickedCh:
				a.openGitHubRepository()
            case <-startupStartItem.ClickedCh:
                if a.server != nil { _ = a.server.StartStartupScript(a.ctx) }
                refreshStartup()
            case <-startupStopItem.ClickedCh:
                if a.server != nil { _ = a.server.StopStartupScript() }
                refreshStartup()
            case <-startupRestartItem.ClickedCh:
                if a.server != nil { _ = a.server.RestartStartupScript(a.ctx) }
                refreshStartup()
            case <-startupEditItem.ClickedCh:
                if a.server != nil {
                    status := a.server.GetStartupScriptStatus()
                    if status != nil {
                        if p, ok := status["path"].(string); ok && p != "" {
                            // Basic ~ expansion
                            if strings.HasPrefix(p, "~") {
                                if home, err := os.UserHomeDir(); err == nil {
                                    p = filepath.Join(home, strings.TrimPrefix(p, "~"))
                                }
                            }
                            a.openFile(p, "startup script")
                        } else {
                            // Fallback: open config to set path
                            a.editConfigFile()
                        }
                    }
                }
			case <-a.groupManagementMenu.ClickedCh:
				a.openGroupManagementWeb()
		case <-a.resourceMonitorMenu.ClickedCh:
			a.openResourceMonitor()
			case <-a.connectedServersMenu.ClickedCh:
				a.handleServerMenuClick("connected")
			case <-a.disconnectedServersMenu.ClickedCh:
				a.handleServerMenuClick("disconnected")
			case <-a.disabledServersMenu.ClickedCh:
				a.handleServerMenuClick("disabled")
			case <-a.quarantineMenu.ClickedCh:
				a.handleServerMenuClick("quarantine")
			case <-quitItem.ClickedCh:
				a.logger.Info("Quit item clicked, shutting down")

				// Force quit with timeout to prevent hanging
				go func() {
					// Set a maximum time for graceful shutdown
					// MED-002: Using centralized timeout constants
					timeout := time.After(config.TrayQuitTimeout)
					killTimeout := time.After(config.TrayKillTimeout)
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
						os.Exit(0)
					case <-timeout:
						a.logger.Warn("Graceful shutdown timed out after 10s, forcing exit with os.Exit(0)")
						os.Exit(0) // Force exit if graceful shutdown hangs

						// If os.Exit(0) doesn't work (should never happen), wait for kill timeout
						select {
						case <-killTimeout:
							a.logger.Error("os.Exit(0) failed, sending SIGKILL to self")
							// Last resort: kill -9 self
							pid := os.Getpid()
							killCmd := exec.Command("kill", "-9", fmt.Sprintf("%d", pid))
							_ = killCmd.Run()
						}
					}
				}()
				return
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
	appState, _ := status["app_state"].(string)
	serverRunning := a.server != nil && a.server.IsRunning()

	a.logger.Debug("Updating tray status",
		zap.Bool("status_running", running),
		zap.Bool("server_is_running", serverRunning),
		zap.String("phase", phase),
		zap.String("app_state", appState),
		zap.Any("status_data", status))

	// Use the actual server running state as the authoritative source
	actuallyRunning := serverRunning

	// Update running status and start/stop button
	if actuallyRunning {
		listenAddr, _ := status["listen_addr"].(string)

		// Show app state in status message
		switch appState {
		case "starting":
			a.statusItem.SetTitle("Status: Starting...")
		case "degraded":
			if listenAddr != "" {
				a.statusItem.SetTitle(fmt.Sprintf("Status: Degraded (%s)", listenAddr))
			} else {
				a.statusItem.SetTitle("Status: Degraded")
			}
		case "stopping":
			a.statusItem.SetTitle("Status: Stopping...")
		default: // "running" or empty
			if listenAddr != "" {
				a.statusItem.SetTitle(fmt.Sprintf("Status: Running (%s)", listenAddr))
			} else {
				a.statusItem.SetTitle("Status: Running")
			}
		}
		a.logger.Debug("Set tray to running state", zap.String("app_state", appState))
	} else {
		a.statusItem.SetTitle("Status: Stopped")
		a.logger.Debug("Set tray to stopped state")
	}


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

	// Update the stop/start all button based on server states
	a.updateStopStartButton()
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

// updateStatus updates the status menu item
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

// getStoppableServerCount counts servers that can be stopped (enabled, not quarantined, not auto-disabled, not already stopped)
func (a *App) getStoppableServerCount() int {
	servers, err := a.server.GetAllServers()
	if err != nil {
		return 0
	}

	count := 0
	for _, serverMap := range servers {
		enabled, _ := serverMap["enabled"].(bool)
		quarantined, _ := serverMap["quarantined"].(bool)
		autoDisabled, _ := serverMap["auto_disabled"].(bool)
		stopped, _ := serverMap["stopped"].(bool)

		// Count servers that are enabled, not quarantined, not auto-disabled, and not already stopped
		if enabled && !quarantined && !autoDisabled && !stopped {
			count++
		}
	}
	return count
}

// getStoppedServerCount counts currently stopped servers
func (a *App) getStoppedServerCount() int {
	servers, err := a.server.GetAllServers()
	if err != nil {
		return 0
	}

	count := 0
	for _, serverMap := range servers {
		stopped, _ := serverMap["stopped"].(bool)
		if stopped {
			count++
		}
	}
	return count
}

// updateStopStartButton updates the stop/start all button based on current server states
func (a *App) updateStopStartButton() {
	if a.stopStartAllItem == nil {
		return
	}

	stoppedCount := a.getStoppedServerCount()
	stoppableCount := a.getStoppableServerCount()

	if stoppableCount == 0 && stoppedCount > 0 {
		// All stoppable servers are stopped - show "Start All Servers"
		a.stopStartAllItem.SetTitle("▶️ Start All Servers")
		a.stopStartAllItem.SetTooltip("Start all stopped servers")
		a.stopStartAllItem.Enable()
	} else if stoppedCount > 0 {
		// Some servers stopped, some running - show "Start All Servers"
		a.stopStartAllItem.SetTitle("▶️ Start All Servers")
		a.stopStartAllItem.SetTooltip("Start all stopped servers")
		a.stopStartAllItem.Enable()
	} else {
		// No stopped servers - show "Stop All Servers"
		a.stopStartAllItem.SetTitle("⏸️ Stop All Servers")
		a.stopStartAllItem.SetTooltip("Stop all running servers (keeps MCPProxy running)")
		a.stopStartAllItem.Enable()
	}
}

// handleStopStartAll handles both stop and start operations based on current state
func (a *App) handleStopStartAll() {
	stoppedCount := a.getStoppedServerCount()

	// Determine operation based on current state
	if stoppedCount > 0 {
		// Start all stopped servers
		a.handleStartAllServersNew()
	} else {
		// Stop all servers
		a.handleStopAllServersNew()
	}
}

// handleStopAllServersNew stops all running servers (sets Stopped=true, keeps Enabled=true)
func (a *App) handleStopAllServersNew() {
	a.logger.Info("Stopping all upstream servers from tray")

	// Get all servers
	servers, err := a.server.GetAllServers()
	if err != nil {
		a.logger.Error("Failed to get all servers", zap.Error(err))
		return
	}

	// Count stoppable servers
	total := a.getStoppableServerCount()
	if total == 0 {
		a.logger.Info("No servers to stop")
		return
	}

	// Disable button during operation
	if a.stopStartAllItem != nil {
		a.stopStartAllItem.Disable()
	}

	// Stop each stoppable server with progress updates
	stopped := 0
	var mu sync.Mutex

	for _, serverMap := range servers {
		name, ok := serverMap["name"].(string)
		if !ok {
			continue
		}

		enabled, _ := serverMap["enabled"].(bool)
		quarantined, _ := serverMap["quarantined"].(bool)
		autoDisabled, _ := serverMap["auto_disabled"].(bool)
		alreadyStopped, _ := serverMap["stopped"].(bool)

		// Skip servers that can't be stopped
		if !enabled || quarantined || autoDisabled || alreadyStopped {
			continue
		}

		// Stop the server
		go func(serverName string) {
			if err := a.server.StopUpstreamServer(serverName); err != nil {
				a.logger.Error("Failed to stop server", zap.String("server", serverName), zap.Error(err))
			} else {
				mu.Lock()
				stopped++
				current := stopped
				mu.Unlock()

				// Update button with progress
				if a.stopStartAllItem != nil {
					a.stopStartAllItem.SetTitle(fmt.Sprintf("⏸️ Stopping (%d/%d)...", current, total))
				}
				a.logger.Info("Stopped server", zap.String("server", serverName), zap.Int("progress", current), zap.Int("total", total))
			}
		}(name)
	}

	// Wait for all operations to complete (with timeout)
	go func() {
		timeout := time.After(30 * time.Second)
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-timeout:
				a.logger.Warn("Stop all servers operation timed out")

				// Sync and update button after sync completes
				if a.syncManager != nil {
					go func() {
						// Wait for SyncDelayed() to complete (1 second + buffer)
						time.Sleep(1200 * time.Millisecond)
						a.updateStopStartButton()
					}()
					a.syncManager.SyncDelayed()
				} else {
					a.updateStopStartButton()
				}
				return
			case <-ticker.C:
				mu.Lock()
				current := stopped
				mu.Unlock()

				if current >= total {
					a.logger.Info("All servers stopped", zap.Int("count", stopped))

					// Sync and update button after sync completes
					if a.syncManager != nil {
						go func() {
							// Wait for SyncDelayed() to complete (1 second + buffer)
							time.Sleep(1200 * time.Millisecond)
							a.updateStopStartButton()
						}()
						a.syncManager.SyncDelayed()
					} else {
						a.updateStopStartButton()
					}
					return
				}
			}
		}
	}()
}

// handleStartAllServersNew starts all stopped servers (sets Stopped=false)
func (a *App) handleStartAllServersNew() {
	a.logger.Info("Starting all stopped servers from tray")

	// Get all servers
	servers, err := a.server.GetAllServers()
	if err != nil {
		a.logger.Error("Failed to get all servers", zap.Error(err))
		return
	}

	// Count stopped servers
	total := a.getStoppedServerCount()
	if total == 0 {
		a.logger.Info("No stopped servers to start")
		return
	}

	// Disable button during operation
	if a.stopStartAllItem != nil {
		a.stopStartAllItem.Disable()
	}

	// Start each stopped server with progress updates
	started := 0
	var mu sync.Mutex

	for _, serverMap := range servers {
		name, ok := serverMap["name"].(string)
		if !ok {
			continue
		}

		stopped, _ := serverMap["stopped"].(bool)
		if !stopped {
			continue
		}

		// Start the server
		go func(serverName string) {
			if err := a.server.UnstopUpstreamServer(serverName); err != nil {
				a.logger.Error("Failed to start server", zap.String("server", serverName), zap.Error(err))
			} else {
				mu.Lock()
				started++
				current := started
				mu.Unlock()

				// Update button with progress
				if a.stopStartAllItem != nil {
					a.stopStartAllItem.SetTitle(fmt.Sprintf("▶️ Starting (%d/%d)...", current, total))
				}
				a.logger.Info("Started server", zap.String("server", serverName), zap.Int("progress", current), zap.Int("total", total))
			}
		}(name)
	}

	// Wait for all operations to complete (with timeout)
	go func() {
		timeout := time.After(30 * time.Second)
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-timeout:
				a.logger.Warn("Start all servers operation timed out")

				// Sync and update button after sync completes
				if a.syncManager != nil {
					go func() {
						// Wait for SyncDelayed() to complete (1 second + buffer)
						time.Sleep(1200 * time.Millisecond)
						a.updateStopStartButton()
					}()
					a.syncManager.SyncDelayed()
				} else {
					a.updateStopStartButton()
				}
				return
			case <-ticker.C:
				mu.Lock()
				current := started
				mu.Unlock()

				if current >= total {
					a.logger.Info("All stopped servers started", zap.Int("count", started))

					// Sync and update button after sync completes
					if a.syncManager != nil {
						go func() {
							// Wait for SyncDelayed() to complete (1 second + buffer)
							time.Sleep(1200 * time.Millisecond)
							a.updateStopStartButton()
						}()
						a.syncManager.SyncDelayed()
					} else {
						a.updateStopStartButton()
					}
					return
				}
			}
		}
	}()
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

// handleReloadConfig reloads the configuration from file
func (a *App) handleReloadConfig() {
	// Notify sync manager of user activity for adaptive frequency
	if a.syncManager != nil {
		a.syncManager.NotifyUserActivity()
	}

	a.logger.Info("Manual configuration reload requested from tray menu")

	if a.server == nil {
		a.logger.Warn("Server interface not available, cannot reload configuration")
		return
	}

	// Trigger configuration reload in the server
	if err := a.server.ReloadConfiguration(); err != nil {
		a.logger.Error("Failed to reload configuration", zap.Error(err))
		return
	}

	a.logger.Info("Configuration reloaded successfully from tray menu")

	// Force a menu refresh after config reload to show any new servers or groups
	if a.syncManager != nil {
		if err := a.syncManager.SyncNow(); err != nil {
			a.logger.Error("Failed to sync menus after config reload", zap.Error(err))
		}
	}

	// Reload groups from config to reflect any changes
	a.refreshGroupsMenu()
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

// openResourceMonitor opens the web interface for resource monitoring
func (a *App) openResourceMonitor() {
	url := "http://localhost:8080/"
	a.openFile(url, "dashboard web interface")
}

// refreshMenusDelayed refreshes menus after a delay using the synchronization manager
func (a *App) refreshMenusDelayed() {
	// Reload groups from config when refreshing menus
	if a.loadGroupsFromConfig() {
		a.populateServerNamesFromConfig()
		if a.menuManager != nil {
			a.menuManager.SetServerGroups(&a.serverGroups)
		}
	}
	
	if a.syncManager != nil {
		a.syncManager.SyncDelayed()
	} else {
		a.logger.Warn("Sync manager not initialized for delayed refresh")
	}
}

// refreshMenusImmediate refreshes menus immediately using the synchronization manager
func (a *App) refreshMenusImmediate() {
	// Reload groups from config when refreshing menus
	if a.loadGroupsFromConfig() {
		a.populateServerNamesFromConfig()
		if a.menuManager != nil {
			a.menuManager.SetServerGroups(&a.serverGroups)
		}
		// Update Group Management submenus after config reload
		a.updateGroupManagementSubmenus()
	}

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

	case "force_enable":
		// Force-enable a server (used by "Enable All" to avoid race conditions)
		a.logger.Info("Force-enabling server", zap.String("server", serverName))
		err = a.syncManager.HandleServerEnable(serverName, true)

	case "force_disable":
		// Force-disable a server (for consistency with force_enable)
		a.logger.Info("Force-disabling server", zap.String("server", serverName))
		err = a.syncManager.HandleServerEnable(serverName, false)

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

	case "restart":
		err = a.handleServerRestart(serverName)

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
	} else {
		a.autostartItem.SetTitle("Start at Login")
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
	migrateGroupsItem := a.groupManagementMenu.AddSubMenuItem("🔄 Migrate to IDs", "Add IDs to existing groups in config")
	// Note: getlantern/systray doesn't support AddSeparator on submenus
	// a.groupManagementMenu.AddSeparator()

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
			case <-migrateGroupsItem.ClickedCh:
				a.handleMigrateGroups()
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

	// IMPORTANT: Populate ServerNames from current config FIRST
	// This ensures we have the complete list of server assignments
	// before saving, preventing accidental reset of group_ids
	a.populateServerNamesFromConfig()

	// Read the current config file as JSON
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	var configData map[string]interface{}
	if err := json.Unmarshal(data, &configData); err != nil {
		return fmt.Errorf("failed to parse config JSON: %w", err)
	}

	// Convert serverGroups to config format with IDs
	groups := make([]map[string]interface{}, 0, len(a.serverGroups))
	for _, group := range a.serverGroups {
		if group.Enabled {
			groups = append(groups, map[string]interface{}{
				"id":          group.ID,
				"name":        group.Name,
				"description": group.Description,
				"icon_emoji":  group.Icon,
				"color":       group.Color,
				"enabled":     group.Enabled,
			})
		}
	}

	// Update the groups in the config
	configData["groups"] = groups

	// Build map of server -> group ID from current group assignments
	serverToGroupID := make(map[string]int)
	for _, group := range a.serverGroups {
		for _, assignedServerName := range group.ServerNames {
			serverToGroupID[assignedServerName] = group.ID
		}
	}

	// Update server group assignments using group IDs
	if servers, ok := configData["mcpServers"].([]interface{}); ok {
		for _, serverInterface := range servers {
			if server, ok := serverInterface.(map[string]interface{}); ok {
				if serverName, ok := server["name"].(string); ok {
					delete(server, "group_name") // Remove old field

					// Remove deprecated fields that should not be written to config
					delete(server, "enabled")
					delete(server, "quarantined")
					delete(server, "auto_disabled")
					delete(server, "start_on_boot")
					delete(server, "stopped")

					// Set group_id from map, or 0 if not in any group
					if groupID, exists := serverToGroupID[serverName]; exists {
						server["group_id"] = groupID
					} else {
						server["group_id"] = 0
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
						zap.Int("servers", len(group.ServerNames)),
			zap.Bool("enabled", group.Enabled))
	}
}

// handleMigrateGroups manually migrates config to use group IDs
func (a *App) handleMigrateGroups() {
	a.logger.Info("Manual migration to group IDs requested")
	
	if a.server == nil {
		a.logger.Error("Server interface not available for migration")
		return
	}

	configPath := a.server.GetConfigPath()
	if configPath == "" {
		a.logger.Error("Config path not available for migration")
		return
	}

	// Force reload groups from config
	a.loadGroupsFromConfig()
	
	// Force migration
	a.forceMigrateConfigToIDs()
	
	a.logger.Info("Manual migration completed")
}

// forceMigrateConfigToIDs forces migration even if groups already have IDs
func (a *App) forceMigrateConfigToIDs() {
	// Assign IDs to all groups (reassign if needed)
	for _, group := range a.serverGroups {
		if group.ID == 0 {
			group.ID = a.getNextGroupID()
		}
	}
	
	// Force save config
	if err := a.saveGroupsToConfig(); err != nil {
		a.logger.Error("Failed to save migrated config", zap.Error(err))
	} else {
		a.logger.Info("Successfully migrated config to use group IDs", zap.Int("groups", len(a.serverGroups)))
	}
}

// refreshGroupsMenu refreshes the groups submenu with current groups
func (a *App) refreshGroupsMenu() {
	a.logger.Info("refreshGroupsMenu called - START", zap.Int("local_groups_count", len(a.serverGroups)))
	
	// FORCE reload from config first to ensure we have the latest colors
	a.logger.Info("Force reloading groups from config to get latest colors")
	a.loadGroupsFromConfig()
	
	// Clear existing group menu items
	for _, item := range a.groupMenuItems {
		if item != nil {
			item.Hide()
		}
	}
	a.groupMenuItems = make(map[string]*systray.MenuItem)

	// If no local groups exist, try to load from config
	if len(a.serverGroups) == 0 {
		a.logger.Debug("No local groups found, attempting to load from config")
		
		// Load from config
		if a.loadGroupsFromConfig() {
			a.logger.Debug("Successfully loaded groups from config", zap.Int("count", len(a.serverGroups)))
			
			// Populate server assignments from config
			a.populateServerNamesFromConfig()
			
			// Update MenuManager with new groups
			if a.menuManager != nil {
				a.menuManager.SetServerGroups(&a.serverGroups)
				// Trigger a refresh of all server menus to show the groups
				if a.syncManager != nil {
					a.syncManager.SyncDelayed()
				}
			}
		} else {
			a.logger.Error("Failed to load groups from config")
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
	// Note: getlantern/systray doesn't support AddSeparator on submenus
	// if len(a.serverGroups) > 0 {
	// 	a.groupManagementMenu.AddSeparator()
	// }

	// Add each group to the menu
	for groupName, group := range a.serverGroups {
		if group.Enabled {
			icon := group.Icon
			groupTitle := fmt.Sprintf("%s %s (%d servers)", icon, groupName, len(group.ServerNames))
			groupItem := a.groupManagementMenu.AddSubMenuItem(groupTitle, group.Description)

			a.groupMenuItems[groupName] = groupItem
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
	icon := group.Icon
	editTitle := a.groupManagementMenu.AddSubMenuItem(
		fmt.Sprintf("✏️ Editing: %s %s", icon, groupName),
		"Edit group properties")
	editTitle.Disable()

	// Note: getlantern/systray doesn't support AddSeparator on submenus
	// a.groupManagementMenu.AddSeparator()

	// Name editing - show current name and options to change
	nameSection := a.groupManagementMenu.AddSubMenuItem("📝 Change Name", "Edit the group name")
	currentNameItem := nameSection.AddSubMenuItem(
		fmt.Sprintf("Current: %s", groupName),
		"Current group name")
	currentNameItem.Disable()

	// Predefined name suggestions
	// Note: getlantern/systray doesn't support AddSeparator on submenus
	// nameSection.AddSeparator()
	nameOptions := []string{"Work", "Personal", "Development", "Production", "Testing", "AWS", "Databases", "AI/ML", "Monitoring", "Custom-" + fmt.Sprintf("%d", len(a.serverGroups)+1)}

	for _, nameOption := range nameOptions {
		if nameOption != groupName { // Don't show current name as option
			nameItem := nameSection.AddSubMenuItem("→ " + nameOption, "")

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
		fmt.Sprintf("Current: %s %s", group.Icon, getColorName(group.Icon)),
		"Current group color")
	currentColorItem.Disable()

	// Note: getlantern/systray doesn't support AddSeparator on submenus
	// colorSection.AddSeparator()

	// Show all available colors except current one
	for _, colorOption := range GroupColors {
		if colorOption.Emoji != group.Icon {
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
	// Note: getlantern/systray doesn't support AddSeparator on submenus
	// a.groupManagementMenu.AddSeparator()

	// Done button to return to main menu
	doneItem := a.groupManagementMenu.AddSubMenuItem("✅ Done", "Finish editing and return to main menu")
	go func(item *systray.MenuItem) {
		for range item.ClickedCh {
			a.refreshGroupsMenu()
		}
	}(doneItem)

	// Delete button
	deleteItem := a.groupManagementMenu.AddSubMenuItem("🗑️ Delete Group", "")
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
	group.Description = fmt.Sprintf("Custom group for organizing servers - %s", getColorName(group.Icon))

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
	a.updateGroupManagementSubmenus() // Update Group Management submenus to reflect rename
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
	a.updateGroupManagementSubmenus() // Update Group Management submenus to reflect color change
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
	a.updateGroupManagementSubmenus() // Update Group Management submenus to reflect deletion
	if a.syncManager != nil {
		a.syncManager.SyncDelayed() // Refresh server menus to remove group icons
	}
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
	a.updateGroupManagementSubmenus() // Update Group Management submenus to reflect assignment
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
	a.updateGroupManagementSubmenus() // Update Group Management submenus to reflect removal
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
	
	// Set diagnostic agent, chat system, and server manager
	dialog.diagnosticAgent = a.diagnosticAgent
	dialog.chatSystem = a.chatSystem
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

// handleServerRestart restarts a server by disabling and re-enabling it
func (a *App) handleServerRestart(serverName string) error {
	a.logger.Info("Restarting server", zap.String("server", serverName))

	// Notify sync manager of user activity for adaptive frequency
	if a.syncManager != nil {
		a.syncManager.NotifyUserActivity()
	}

	// Get current server configuration to check if it's enabled
	allServers, err := a.stateManager.GetAllServers()
	if err != nil {
		return fmt.Errorf("failed to get servers for restart: %w", err)
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
		return fmt.Errorf("server '%s' not found", serverName)
	}

	// Only restart if the server is currently enabled
	if !serverEnabled {
		a.logger.Warn("Cannot restart disabled server", zap.String("server", serverName))
		return fmt.Errorf("server '%s' is disabled and cannot be restarted", serverName)
	}

	// Restart by disabling then re-enabling
	a.logger.Info("Disabling server for restart", zap.String("server", serverName))
	if err := a.syncManager.HandleServerEnable(serverName, false); err != nil {
		return fmt.Errorf("failed to disable server for restart: %w", err)
	}

	// Add a small delay to ensure the server is fully stopped
	time.Sleep(500 * time.Millisecond)

	a.logger.Info("Re-enabling server after restart", zap.String("server", serverName))
	if err := a.syncManager.HandleServerEnable(serverName, true); err != nil {
		return fmt.Errorf("failed to re-enable server after restart: %w", err)
	}

	a.logger.Info("Server restart completed successfully", zap.String("server", serverName))
	return nil
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

// loadGroupsForServerMenus loads groups from config for server assignment menus
func (a *App) loadGroupsForServerMenus() {
	a.logger.Info("loadGroupsForServerMenus called - START")
	
	// Load groups from config to get correct colors and assignments
	if a.loadGroupsFromConfig() {
		a.logger.Info("Successfully loaded groups from config", zap.Int("group_count", len(a.serverGroups)))
		
		// Populate ServerNames from config assignments
		a.populateServerNamesFromConfig()

		// Update MenuManager with groups (if menuManager is available)
		if a.menuManager != nil {
			a.menuManager.SetServerGroups(&a.serverGroups)
			a.logger.Info("Updated MenuManager with groups", zap.Int("group_count", len(a.serverGroups)))

			// Create/update Group Management submenus
			a.updateGroupManagementSubmenus()

			// Trigger a refresh of all server menus to show the groups
			if a.syncManager != nil {
				a.syncManager.SyncDelayed()
				a.logger.Info("Triggered delayed sync to refresh server menus with groups")
			}
		} else {
			a.logger.Debug("MenuManager not available yet, groups will be set later")
		}
	} else {
		a.logger.Error("Failed to load groups from config for server menus")
	}
}

// populateServerNamesFromConfig reads the config file and populates ServerNames for each group
func (a *App) populateServerNamesFromConfig() {
	if a.server == nil {
		a.logger.Error("Server interface not available for config reading")
		return
	}

	configPath := a.server.GetConfigPath()
	if configPath == "" {
		a.logger.Error("Config path not available")
		return
	}

	// Read the current config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		a.logger.Error("Failed to read config file for group population", zap.Error(err))
		return
	}

	var configData map[string]interface{}
	if err := json.Unmarshal(data, &configData); err != nil {
		a.logger.Error("Failed to parse config JSON for group population", zap.Error(err))
		return
	}

	// Clear existing server assignments
	for _, group := range a.serverGroups {
		group.ServerNames = make([]string, 0)
	}

	// Read server assignments from config
	if servers, ok := configData["mcpServers"].([]interface{}); ok {
		for _, serverInterface := range servers {
			if server, ok := serverInterface.(map[string]interface{}); ok {
				if serverName, ok := server["name"].(string); ok {
					// Check for group_id (new format) or group_name (legacy format)
					var targetGroup *ServerGroup
					
					if groupID, ok := server["group_id"].(float64); ok && groupID != 0 {
						// New format: use group_id
						targetGroup = a.getGroupByID(int(groupID))
					} else if groupName, ok := server["group_name"].(string); ok && groupName != "" {
						// Legacy format: use group_name
						targetGroup = a.getGroupByName(groupName)
					}
					
					if targetGroup != nil {
						// Check if server is not already in the group
						found := false
						for _, existingServer := range targetGroup.ServerNames {
							if existingServer == serverName {
								found = true
								break
							}
						}
						if !found {
							targetGroup.ServerNames = append(targetGroup.ServerNames, serverName)
							a.logger.Debug("Added server to group from config",
								zap.String("server", serverName),
								zap.String("group", targetGroup.Name),
								zap.Int("group_id", targetGroup.ID))
						}
					}
				}
			}
		}
	}

	a.logger.Info("Populated server names from config", zap.Int("groups_count", len(a.serverGroups)))
}

// loadGroupsFromConfig loads groups from the configuration file
func (a *App) loadGroupsFromConfig() bool {
	if a.server == nil {
		a.logger.Error("Server interface not available for config reading")
		return false
	}

	configPath := a.server.GetConfigPath()
	if configPath == "" {
		a.logger.Error("Config path not available")
		return false
	}

	// Read the current config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		a.logger.Debug("Failed to read config file for group loading", zap.Error(err))
		return false
	}

	var configData map[string]interface{}
	if err := json.Unmarshal(data, &configData); err != nil {
		a.logger.Error("Failed to parse config JSON for group loading", zap.Error(err))
		return false
	}

	// Load groups from config
	if groups, ok := configData["groups"].([]interface{}); ok && len(groups) > 0 {
		a.logger.Debug("Found groups in config", zap.Int("count", len(groups)))
		
		for _, groupInterface := range groups {
			if group, ok := groupInterface.(map[string]interface{}); ok {
				name, nameOk := group["name"].(string)
				if !nameOk || name == "" {
					continue
				}

				// Get ID from config, or generate one if missing
				id, idOk := group["id"].(float64) // JSON numbers are float64
				if !idOk {
					id = float64(a.getNextGroupID())
				}

				description, _ := group["description"].(string)
				color, _ := group["color"].(string)
				icon, _ := group["icon_emoji"].(string)
				enabled, _ := group["enabled"].(bool)
				
				// Set defaults
				if description == "" {
					description = fmt.Sprintf("Custom group: %s", name)
				}
				if color == "" {
					color = "#6c757d"
				}
				
				a.serverGroups[name] = &ServerGroup{
					ID:          int(id),
					Name:        name,
					Description: description,
					Icon:        icon,
					Color:       color,
					ServerNames: make([]string, 0),
					Enabled:     enabled,
				}

				a.logger.Debug("Loaded group from config",
					zap.String("name", name),
					zap.Int("id", int(id)),
					zap.String("color", color),
					zap.Bool("enabled", enabled))
			}
		}

		a.logger.Info("Successfully loaded groups from config", zap.Int("count", len(a.serverGroups)))
		// Migrate config if needed (add IDs to groups without them)
		a.migrateConfigToIDs()
		
		return len(a.serverGroups) > 0
	}

	a.logger.Debug("No groups found in config file")
	return false
}

// migrateConfigToIDs adds IDs to existing groups in config file if they don't have them
func (a *App) migrateConfigToIDs() {
	needsMigration := false
	
	// Check if any group is missing an ID
	for _, group := range a.serverGroups {
		if group.ID == 0 {
			group.ID = a.getNextGroupID()
			needsMigration = true
			a.logger.Info("Assigned ID to existing group", 
				zap.String("name", group.Name), 
				zap.Int("id", group.ID))
		}
	}
	
	// Save config if migration was needed
	if needsMigration {
		if err := a.saveGroupsToConfig(); err != nil {
			a.logger.Error("Failed to save migrated config", zap.Error(err))
		} else {
			a.logger.Info("Successfully migrated config to use group IDs")
		}
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
			color = "#6c757d" // Default color
		}

		// Create tray group from API group
		newGroup := &ServerGroup{
			ID:          a.getNextGroupID(),
			Name:        name,
			Description: fmt.Sprintf("Synced from API: %s", name),
			Color:       color,
			ServerNames: make([]string, 0),
			Enabled:     true,
		}

		a.serverGroups[name] = newGroup
	}

	a.logger.Info("Synchronized with API groups", zap.Int("count", len(apiGroups)))
}

// fetchServerAssignments fetches server-to-group assignments
func (a *App) fetchServerAssignments() (map[string]string, error) {
	baseURL := "http://localhost:8080"
	resp, err := http.Get(baseURL + "/api/assignments")
	if err != nil {
		a.logger.Error("Failed to fetch server assignments from API", zap.Error(err))
		return make(map[string]string), err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("API returned status %d", resp.StatusCode)
		a.logger.Error("Server assignments API returned error", zap.Error(err))
		return make(map[string]string), err
	}

	var response struct {
		Success     bool `json:"success"`
		Assignments []struct {
			ServerName string `json:"server_name"`
			GroupName  string `json:"group_name"`
		} `json:"assignments"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		a.logger.Error("Failed to decode server assignments response", zap.Error(err))
		return make(map[string]string), err
	}

	if !response.Success {
		err := fmt.Errorf("API returned success=false")
		a.logger.Error("Server assignments API returned failure", zap.Error(err))
		return make(map[string]string), err
	}

	// Convert to map
	assignments := make(map[string]string)
	for _, assignment := range response.Assignments {
		assignments[assignment.ServerName] = assignment.GroupName
	}

	a.logger.Info("Successfully fetched server assignments",
		zap.Int("count", len(assignments)),
		zap.Any("assignments", assignments))

	return assignments, nil
}

// assignServerToGroup assigns a server to a group
func (a *App) assignServerToGroup(serverName, groupName string) {
	a.logger.Info("Assigning server to group", zap.String("server", serverName), zap.String("group", groupName))

	// Prepare assignment data
	assignmentData := map[string]string{
		"server_name": serverName,
		"group_name":  groupName,
	}

	jsonData, err := json.Marshal(assignmentData)
	if err != nil {
		a.logger.Error("Failed to marshal assignment data", zap.Error(err))
		return
	}

	// Send assignment request to API
	baseURL := "http://localhost:8080"
	resp, err := http.Post(baseURL+"/api/assign-server", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		a.logger.Error("Failed to send server assignment request", zap.Error(err))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		a.logger.Error("Server assignment API returned error", zap.Int("status", resp.StatusCode))
		return
	}

	var response struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
		Error   string `json:"error,omitempty"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		a.logger.Error("Failed to decode assignment response", zap.Error(err))
		return
	}

	if !response.Success {
		a.logger.Error("Server assignment failed", zap.String("error", response.Error))
		return
	}

	a.logger.Info("Server assignment completed successfully",
		zap.String("server", serverName),
		zap.String("group", groupName),
		zap.String("message", response.Message))
}

// ServerSnapshot represents a snapshot of server state before stop operations.
// HIGH-001: Renamed from ServerState to ServerSnapshot to avoid confusion with types.ServerState
type ServerSnapshot struct {
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
	states := make([]ServerSnapshot, 0, len(servers))
	for _, server := range servers {
		name, _ := server["name"].(string)
		enabled, _ := server["enabled"].(bool)
		connected, _ := server["connected"].(bool)

		states = append(states, ServerSnapshot{
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

	var states []ServerSnapshot
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

// updateGroupManagementSubmenus creates dynamic submenus for Group Management
// showing each group with its corresponding servers loaded from config file
func (a *App) updateGroupManagementSubmenus() {
	a.logger.Info("Updating Group Management submenus")

	// ENSURE: Load groups from config to get correct colors before creating menus
	a.loadGroupsFromConfig()
	a.logger.Debug("Reloaded groups from config for Group Management", zap.Int("groups_count", len(a.serverGroups)))

	// Clear existing group management menu items
	if a.groupManagementMenu != nil {
		// Remove all submenu items
		for _, item := range a.groupMenuItems {
			if item != nil {
				item.Hide()
			}
		}
		a.groupMenuItems = make(map[string]*systray.MenuItem)
	}

	// Check if server groups are available
	if len(a.serverGroups) == 0 {
		a.logger.Info("No groups available for Group Management menu")
		noGroupsItem := a.groupManagementMenu.AddSubMenuItem("📋 No groups available", "Create groups to manage servers")
		noGroupsItem.Disable()
		a.groupMenuItems["no_groups"] = noGroupsItem
		return
	}

	// Get server assignments to show which servers belong to each group
	assignments, err := a.fetchServerAssignments()
	if err != nil {
		a.logger.Error("Failed to fetch server assignments for Group Management", zap.Error(err))
	}

	// Create submenu for each group showing its servers
	for groupName, group := range a.serverGroups {
		if !group.Enabled {
			continue // Skip disabled groups
		}

		// Get servers assigned to this group
		var assignedServers []string
		for serverName, assignedGroup := range assignments {
			if assignedGroup == groupName {
				assignedServers = append(assignedServers, serverName)
			}
		}

		// Create group submenu title with server count
		icon := group.Icon
		groupTitle := fmt.Sprintf("%s %s (%d servers)", icon, groupName, len(assignedServers))
		groupItem := a.groupManagementMenu.AddSubMenuItem(groupTitle, "")

		// Add assigned servers as submenus under each group
		if len(assignedServers) > 0 {
			// Add a separator before server list
			groupItem.AddSubMenuItem("───── Servers ─────", "").Disable()

			for _, serverName := range assignedServers {
				serverTitle := fmt.Sprintf("🖥️  %s", serverName)
				serverItem := groupItem.AddSubMenuItem(serverTitle, "")

				// Add options for this server
				removeServerItem := serverItem.AddSubMenuItem("➖ Remove from Group", "")

				// Handle remove server click
				go func(sName, gName string, item *systray.MenuItem) {
					for range item.ClickedCh {
						a.handleRemoveServerFromGroup(sName, gName)
						// Refresh the Group Management submenus after change
						a.updateGroupManagementSubmenus()
					}
				}(serverName, groupName, removeServerItem)
			}

			// Add separator before group actions
			groupItem.AddSubMenuItem("───── Actions ─────", "").Disable()

			// Add Enable All Servers button
			enableAllItem := groupItem.AddSubMenuItem("✅ Enable All Servers", "Enable all servers in this group")
			go func(gName string, sNames []string, item *systray.MenuItem) {
				for range item.ClickedCh {
					a.handleEnableAllServersInGroup(gName, sNames)
				}
			}(groupName, assignedServers, enableAllItem)

			// Add Disable All Servers button
			disableAllItem := groupItem.AddSubMenuItem("🚫 Disable All Servers", "Disable all servers in this group")
			go func(gName string, sNames []string, item *systray.MenuItem) {
				for range item.ClickedCh {
					a.handleDisableAllServersInGroup(gName, sNames)
				}
			}(groupName, assignedServers, disableAllItem)
		} else {
			// No servers assigned to this group
			noServersItem := groupItem.AddSubMenuItem("📭 No servers assigned", "Assign servers to this group")
			noServersItem.Disable()
		}

		// Store the group menu item
		a.groupMenuItems[groupName] = groupItem
	}

	a.logger.Info("Group Management submenus updated",
		zap.Int("groups_count", len(a.serverGroups)),
		zap.Int("assignments_count", len(assignments)))
}

// handleEnableAllServersInGroup enables all servers assigned to a group
func (a *App) handleEnableAllServersInGroup(groupName string, serverNames []string) {
	if len(serverNames) == 0 {
		a.logger.Warn("No servers to enable in group", zap.String("group", groupName))
		return
	}

	a.logger.Info("Enabling all servers in group",
		zap.String("group", groupName),
		zap.Int("server_count", len(serverNames)))

	successCount := 0
	for _, serverName := range serverNames {
		// Use the server manager to enable the server (updates config + storage)
		if err := a.server.EnableServer(serverName, true); err != nil {
			a.logger.Error("Failed to enable server in group",
				zap.String("server", serverName),
				zap.String("group", groupName),
				zap.Error(err))
		} else {
			successCount++
		}
	}

	a.logger.Info("Completed enabling servers in group",
		zap.String("group", groupName),
		zap.Int("success_count", successCount),
		zap.Int("total_count", len(serverNames)))

	// Trigger menu refresh
	if a.syncManager != nil {
		a.syncManager.SyncDelayed()
	}
}

// handleDisableAllServersInGroup disables all servers assigned to a group
func (a *App) handleDisableAllServersInGroup(groupName string, serverNames []string) {
	if len(serverNames) == 0 {
		a.logger.Warn("No servers to disable in group", zap.String("group", groupName))
		return
	}

	a.logger.Info("Disabling all servers in group",
		zap.String("group", groupName),
		zap.Int("server_count", len(serverNames)))

	successCount := 0
	for _, serverName := range serverNames {
		// Use the server manager to disable the server (updates config + storage)
		if err := a.server.EnableServer(serverName, false); err != nil {
			a.logger.Error("Failed to disable server in group",
				zap.String("server", serverName),
				zap.String("group", groupName),
				zap.Error(err))
		} else {
			successCount++
		}
	}

	a.logger.Info("Completed disabling servers in group",
		zap.String("group", groupName),
		zap.Int("success_count", successCount),
		zap.Int("total_count", len(serverNames)))

	// Trigger menu refresh
	if a.syncManager != nil {
		a.syncManager.SyncDelayed()
	}
}

// generateColorForGroup creates a deterministic vibrant color from group name
func (a *App) generateColorForGroup(name string) string {
	h := fnv.New32a()
	_, _ = h.Write([]byte(strings.ToLower(strings.TrimSpace(name))))
	seed := h.Sum32()
	// Map to hue [0,360), use fixed saturation/value for visibility
	hue := float64(seed%360)
	sat := 0.75
	val := 0.85
	r, g, b := hsvToRgb(hue/360.0, sat, val)
	return fmt.Sprintf("#%02x%02x%02x", int(math.Round(r*255)), int(math.Round(g*255)), int(math.Round(b*255)))
}

// parseHexColor supports #rgb, #rgba, #rrggbb, #rrggbbaa
func parseHexColor(s string) (float64, float64, float64) {
	s = strings.TrimSpace(strings.ToLower(s))
	if strings.HasPrefix(s, "#") {
		s = s[1:]
	}
	var r, g, b int64
	if len(s) == 3 {
		// #rgb
		r, _ = strconv.ParseInt(strings.Repeat(string(s[0]), 2), 16, 64)
		g, _ = strconv.ParseInt(strings.Repeat(string(s[1]), 2), 16, 64)
		b, _ = strconv.ParseInt(strings.Repeat(string(s[2]), 2), 16, 64)
	} else if len(s) >= 6 {
		r, _ = strconv.ParseInt(s[0:2], 16, 64)
		g, _ = strconv.ParseInt(s[2:4], 16, 64)
		b, _ = strconv.ParseInt(s[4:6], 16, 64)
	} else {
		return 0.42, 0.42, 0.42
	}
	return float64(r) / 255.0, float64(g) / 255.0, float64(b) / 255.0
}

func rgbToHsv(r, g, b float64) (float64, float64, float64) {
	max := math.Max(r, math.Max(g, b))
	min := math.Min(r, math.Min(g, b))
	d := max - min
	var h float64
	if d == 0 {
		h = 0
	} else if max == r {
		h = math.Mod(((g-b)/d), 6)
	} else if max == g {
		h = ((b-r)/d + 2)
	} else {
		h = ((r-g)/d + 4)
	}
	h = h / 6
	if h < 0 {
		h += 1
	}
	var s float64
	if max == 0 { s = 0 } else { s = d / max }
	v := max
	return h, s, v
}

func hsvToRgb(h, s, v float64) (float64, float64, float64) {
	if s == 0 { return v, v, v }
	h = h * 6
	i := math.Floor(h)
	f := h - i
	p := v * (1 - s)
	q := v * (1 - s*f)
	t := v * (1 - s*(1-f))
	switch int(i) % 6 {
	case 0:
		return v, t, p
	case 1:
		return q, v, p
	case 2:
		return p, v, t
	case 3:
		return p, q, v
	case 4:
		return t, p, v
	default:
		return v, p, q
	}
}

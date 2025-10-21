//go:build !nogui && !headless && !linux

package tray

import (
	"context"
	"crypto/md5"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"fyne.io/systray"
	"go.uber.org/zap"
)

const (
	actionEnable  = "enable"
	actionDisable = "disable"
	textEnable    = "Enable"
	textDisable   = "Disable"
)

// ServerStateManager manages server state synchronization between storage, config, and menu
type ServerStateManager struct {
	server ServerInterface
	logger *zap.SugaredLogger
	mu     sync.RWMutex

	// Current state tracking
	allServers           []map[string]interface{}
	quarantinedServers   []map[string]interface{}
	lastUpdate           time.Time
	lastQuarantineUpdate time.Time // Separate timestamp for quarantine data
}

// NewServerStateManager creates a new server state manager
func NewServerStateManager(server ServerInterface, logger *zap.SugaredLogger) *ServerStateManager {
	return &ServerStateManager{
		server: server,
		logger: logger,
	}
}

// RefreshState forces a refresh of server state from the server
func (m *ServerStateManager) RefreshState() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Get all servers
	allServers, err := m.server.GetAllServers()
	if err != nil {
		m.logger.Error("RefreshState failed to get all servers", zap.Error(err))
		return fmt.Errorf("failed to get all servers: %w", err)
	}

	// Get quarantined servers
	quarantinedServers, err := m.server.GetQuarantinedServers()
	if err != nil {
		m.logger.Error("RefreshState failed to get quarantined servers", zap.Error(err))
		return fmt.Errorf("failed to get quarantined servers: %w", err)
	}

	m.allServers = allServers
	m.quarantinedServers = quarantinedServers
	m.lastUpdate = time.Now()
	m.lastQuarantineUpdate = time.Now()

	m.logger.Debug("Server state refreshed",
		zap.Int("all_servers", len(allServers)),
		zap.Int("quarantined_servers", len(quarantinedServers)))

	return nil
}

// GetAllServers returns cached or fresh server list
func (m *ServerStateManager) GetAllServers() ([]map[string]interface{}, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return cached data if available and recent (only if THIS data has been loaded before)
	if time.Since(m.lastUpdate) < 2*time.Second && !m.lastUpdate.IsZero() && m.allServers != nil {
		return m.allServers, nil
	}

	// Get fresh data but handle database errors gracefully
	servers, err := m.server.GetAllServers()
	if err != nil {
		// If database is closed, return cached data if available
		if strings.Contains(err.Error(), "database not open") || strings.Contains(err.Error(), "closed") {
			if len(m.allServers) > 0 {
				m.logger.Debug("Database not available, returning cached server data")
				return m.allServers, nil
			}
			// No cached data available, return cached data or fallback to avoid UI flickering
			m.logger.Debug("Database not available and no cached data, preserving UI state")
			// Return error to indicate data is not available, let caller handle gracefully
			return nil, fmt.Errorf("database not available and no cached data: %w", err)
		}
		m.logger.Error("Failed to get fresh all servers data", zap.Error(err))
		return nil, err
	}

	// Only update cache if we got valid data (non-empty or intentionally empty)
	// This prevents overwriting good cached data with temporary empty results
	if servers != nil {
		m.allServers = servers
		m.lastUpdate = time.Now()
	}
	return servers, nil
}

// GetQuarantinedServers returns cached or fresh quarantined server list
func (m *ServerStateManager) GetQuarantinedServers() ([]map[string]interface{}, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return cached data if available and recent (only if data has been loaded before)
	if time.Since(m.lastQuarantineUpdate) < 2*time.Second && !m.lastQuarantineUpdate.IsZero() {
		return m.quarantinedServers, nil
	}

	// Get fresh data but handle database errors gracefully
	servers, err := m.server.GetQuarantinedServers()
	if err != nil {
		// If database is closed, return cached data if available
		if strings.Contains(err.Error(), "database not open") || strings.Contains(err.Error(), "closed") {
			if len(m.quarantinedServers) > 0 {
				m.logger.Debug("Database not available, returning cached quarantine data")
				return m.quarantinedServers, nil
			}
			// No cached data available, return error to preserve UI state
			m.logger.Debug("Database not available and no cached data, preserving quarantine UI state")
			return nil, fmt.Errorf("database not available and no cached data: %w", err)
		}
		m.logger.Error("Failed to get fresh quarantined servers data", zap.Error(err))
		return nil, err
	}

	// Only update cache if we got valid data
	if servers != nil {
		m.quarantinedServers = servers
		m.lastQuarantineUpdate = time.Now()
	}
	return servers, nil
}

// GetServerByName returns data for a specific server by name
func (m *ServerStateManager) GetServerByName(serverName string) (map[string]interface{}, error) {
	// Get all servers (uses cache if recent)
	allServers, err := m.GetAllServers()
	if err != nil {
		return nil, fmt.Errorf("failed to get servers: %w", err)
	}

	// Find the server by name
	for _, server := range allServers {
		if name, ok := server["name"].(string); ok && name == serverName {
			return server, nil
		}
	}

	return nil, fmt.Errorf("server not found: %s", serverName)
}

// QuarantineServer quarantines a server and ensures all state is synchronized
func (m *ServerStateManager) QuarantineServer(serverName string, quarantined bool) error {
	m.logger.Info("QuarantineServer called",
		zap.String("server", serverName),
		zap.Bool("quarantined", quarantined))

	// Update the server quarantine status
	if err := m.server.QuarantineServer(serverName, quarantined); err != nil {
		return fmt.Errorf("failed to quarantine server: %w", err)
	}

	// Force state refresh immediately after the change
	if err := m.RefreshState(); err != nil {
		m.logger.Error("Failed to refresh state after quarantine change", zap.Error(err))
		// Don't return error here as the quarantine operation itself succeeded
	}

	m.logger.Info("Server quarantine status updated successfully",
		zap.String("server", serverName),
		zap.Bool("quarantined", quarantined))

	return nil
}

// UnquarantineServer removes a server from quarantine and ensures all state is synchronized
func (m *ServerStateManager) UnquarantineServer(serverName string) error {
	m.logger.Info("UnquarantineServer called", zap.String("server", serverName))

	// Update the server quarantine status
	if err := m.server.UnquarantineServer(serverName); err != nil {
		return fmt.Errorf("failed to unquarantine server: %w", err)
	}

	// Force state refresh immediately after the change
	if err := m.RefreshState(); err != nil {
		m.logger.Error("Failed to refresh state after unquarantine change", zap.Error(err))
		// Don't return error here as the unquarantine operation itself succeeded
	}

	m.logger.Info("Server unquarantine completed successfully", zap.String("server", serverName))

	return nil
}

// EnableServer enables/disables a server and ensures all state is synchronized
func (m *ServerStateManager) EnableServer(serverName string, enabled bool) error {
	action := actionDisable
	if enabled {
		action = actionEnable
	}

	m.logger.Info("EnableServer called",
		zap.String("server", serverName),
		zap.String("action", action))

	// Update the server enable status
	if err := m.server.EnableServer(serverName, enabled); err != nil {
		return fmt.Errorf("failed to %s server: %w", action, err)
	}

	// Force state refresh immediately after the change
	if err := m.RefreshState(); err != nil {
		m.logger.Error("Failed to refresh state after enable change", zap.Error(err))
		// Don't return error here as the enable operation itself succeeded
	}

	m.logger.Info("Server enable status updated successfully",
		zap.String("server", serverName),
		zap.String("action", action))

	return nil
}

// MenuManager manages tray menu state and prevents duplications
type MenuManager struct {
	logger *zap.SugaredLogger
	mu     sync.RWMutex

	// Menu references - Status-based menus
	connectedServersMenu    *systray.MenuItem
	disconnectedServersMenu *systray.MenuItem
	sleepingServersMenu     *systray.MenuItem
	stoppedServersMenu      *systray.MenuItem
	disabledServersMenu     *systray.MenuItem
	quarantineMenu          *systray.MenuItem

	// Legacy menu reference
	upstreamServersMenu *systray.MenuItem

	// Menu tracking to prevent duplicates - Status-based tracking
	connectedMenuItems      map[string]*systray.MenuItem // server name -> connected menu item
	disconnectedMenuItems   map[string]*systray.MenuItem // server name -> disconnected menu item
	sleepingMenuItems       map[string]*systray.MenuItem // server name -> sleeping menu item
	stoppedMenuItems        map[string]*systray.MenuItem // server name -> stopped menu item
	disabledMenuItems       map[string]*systray.MenuItem // server name -> disabled menu item
	quarantineMenuItems     map[string]*systray.MenuItem // server name -> quarantine menu item

	// Legacy tracking
	serverMenuItems       map[string]*systray.MenuItem // server name -> legacy menu item

	// Action item tracking (shared across all status menus)
	serverActionItems     map[string]*systray.MenuItem // server name -> enable/disable action menu item
	serverQuarantineItems map[string]*systray.MenuItem // server name -> quarantine action menu item
	serverOAuthItems      map[string]*systray.MenuItem // server name -> OAuth login menu item
	serverLogItems        map[string]*systray.MenuItem // server name -> open log menu item
	serverRepoItems       map[string]*systray.MenuItem // server name -> open repo menu item
	serverConfigItems     map[string]*systray.MenuItem // server name -> configure menu item
	serverRestartItems    map[string]*systray.MenuItem // server name -> restart menu item
	quarantineInfoEmpty   *systray.MenuItem            // "No servers" info item
	quarantineInfoHelp    *systray.MenuItem            // "Click to unquarantine" help item

	// Header items and separators tracking
	headerItems           []*systray.MenuItem          // All header items (Connected, Disconnected, etc.)
	separatorItems        []*systray.MenuItem          // All separator items

	// State tracking to detect changes
	lastServerNames     []string
	lastQuarantineNames []string
	menusInitialized    bool

	// Menu state tracking to prevent unnecessary recreation
	lastMenuState     string
	lastMenuStateHash string
	menuUpdateMutex   sync.Mutex
	menuDataCached    bool // Flag to track if menu data is loaded
	isMenuOpen        bool // Track if user has menu open
	menuOpenTime      time.Time

	// Server groups for color display
	serverGroups *map[string]*ServerGroup // Reference to server groups from App

	// Event handler callbacks
	onServerAction     func(serverName string, action string) // callback for server actions
	onServerCountUpdate func(totalCount int)                    // callback for server count updates
}

// NewMenuManager creates a new menu manager
func NewMenuManager(connectedMenu, disconnectedMenu, sleepingMenu, stoppedMenu, disabledMenu, quarantineMenu, upstreamMenu *systray.MenuItem, logger *zap.SugaredLogger) *MenuManager {
	return &MenuManager{
		logger:                  logger,
		connectedServersMenu:    connectedMenu,
		disconnectedServersMenu: disconnectedMenu,
		sleepingServersMenu:     sleepingMenu,
		stoppedServersMenu:      stoppedMenu,
		disabledServersMenu:     disabledMenu,
		quarantineMenu:          quarantineMenu,
		upstreamServersMenu:     upstreamMenu,

		// Status-based tracking maps
		connectedMenuItems:      make(map[string]*systray.MenuItem),
		disconnectedMenuItems:   make(map[string]*systray.MenuItem),
		sleepingMenuItems:       make(map[string]*systray.MenuItem),
		stoppedMenuItems:        make(map[string]*systray.MenuItem),
		disabledMenuItems:       make(map[string]*systray.MenuItem),
		quarantineMenuItems:     make(map[string]*systray.MenuItem),

		// Legacy tracking
		serverMenuItems:         make(map[string]*systray.MenuItem),

		// Action item tracking
		serverActionItems:       make(map[string]*systray.MenuItem),
		serverQuarantineItems:   make(map[string]*systray.MenuItem),
		serverOAuthItems:        make(map[string]*systray.MenuItem),
		serverLogItems:          make(map[string]*systray.MenuItem),
		serverRepoItems:         make(map[string]*systray.MenuItem),
		serverConfigItems:       make(map[string]*systray.MenuItem),
		serverRestartItems:      make(map[string]*systray.MenuItem),
		headerItems:             []*systray.MenuItem{},
		separatorItems:          []*systray.MenuItem{},
	}
}

// SetActionCallback sets the callback function for server actions
func (m *MenuManager) SetActionCallback(callback func(serverName string, action string)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onServerAction = callback
}

// SetServerCountCallback sets the callback function for server count updates
func (m *MenuManager) SetServerCountCallback(callback func(totalCount int)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onServerCountUpdate = callback
}

// SetServerGroups sets the reference to server groups for color display
func (m *MenuManager) SetServerGroups(groups *map[string]*ServerGroup) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.serverGroups = groups
}

// removeServersFromWrongMenus removes servers from status menus where they no longer belong
func (m *MenuManager) removeServersFromWrongMenus(allCurrentServers map[string]string) {
	m.logger.Debug("removeServersFromWrongMenus called",
		zap.Int("total_servers", len(allCurrentServers)))

	// Check each status menu and remove servers that shouldn't be there anymore
	statusMenus := map[string]map[string]*systray.MenuItem{
		"connected":     m.connectedMenuItems,
		"disconnected":  m.disconnectedMenuItems,
		"stopped":       m.stoppedMenuItems,
		"disabled":      m.disabledMenuItems,
		"quarantined":   m.quarantineMenuItems,
	}

	for currentStatus, menuItems := range statusMenus {
		m.logger.Debug("Checking status menu",
			zap.String("status", currentStatus),
			zap.Int("menu_items_count", len(menuItems)))

		for serverName, menuItem := range menuItems {
			correctStatus, serverExists := allCurrentServers[serverName]

			m.logger.Debug("Checking server placement",
				zap.String("server", serverName),
				zap.String("current_menu", currentStatus),
				zap.String("correct_status", correctStatus),
				zap.Bool("server_exists", serverExists))

			if !serverExists {
				// Server no longer exists at all, remove it
				m.logger.Debug("Removing non-existent server from menu",
					zap.String("server", serverName),
					zap.String("current_menu", currentStatus))
				menuItem.Hide()
				delete(menuItems, serverName)
				m.cleanupServerActionItems(serverName)
			} else if correctStatus != currentStatus {
				// Server exists but should be in a different status menu
				m.logger.Debug("Moving server from wrong status menu",
					zap.String("server", serverName),
					zap.String("from_status", currentStatus),
					zap.String("to_status", correctStatus))
				menuItem.Hide()
				delete(menuItems, serverName)
				m.cleanupServerActionItems(serverName)
			}
		}
	}
}

// UpdateStatusBasedMenus updates servers grouped by status into separate menus
func (m *MenuManager) UpdateStatusBasedMenus(servers []map[string]interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Stability check: Don't clear existing menus if we get empty servers and we already have servers
	if len(servers) == 0 && (len(m.connectedMenuItems) > 0 || len(m.disconnectedMenuItems) > 0 || len(m.sleepingMenuItems) > 0 || len(m.stoppedMenuItems) > 0 || len(m.disabledMenuItems) > 0) {
		m.logger.Debug("Received empty server list but existing menu items present, preserving UI state")
		return
	}

	// Group servers by status
	var connected, disconnected, sleeping, stopped, disabled, quarantined []map[string]interface{}
	connectedCount, disconnectedCount, sleepingCount, stoppedCount, disabledCount, quarantinedCount := 0, 0, 0, 0, 0, 0

	for _, server := range servers {
		enabled, _ := server["enabled"].(bool)
		serverConnected, _ := server["connected"].(bool)
		serverConnecting, _ := server["connecting"].(bool)
		serverSleeping, _ := server["sleeping"].(bool)
		serverQuarantined, _ := server["quarantined"].(bool)

		// Check if server has a stopped status (when server is not connecting, not sleeping, and not connected but enabled)
		serverStopped := enabled && !serverConnected && !serverConnecting && !serverSleeping

		if serverQuarantined {
			quarantined = append(quarantined, server)
			quarantinedCount++
		} else if !enabled {
			disabled = append(disabled, server)
			disabledCount++
		} else if serverConnected {
			connected = append(connected, server)
			connectedCount++
		} else if serverSleeping {
			sleeping = append(sleeping, server)
			sleepingCount++
		} else if serverStopped {
			stopped = append(stopped, server)
			stoppedCount++
		} else {
			disconnected = append(disconnected, server)
			disconnectedCount++
		}
	}

	// Update menu titles with counts
	if m.connectedServersMenu != nil {
		m.connectedServersMenu.SetTitle(fmt.Sprintf("🟢 Connected Servers (%d)", connectedCount))
	}
	if m.disconnectedServersMenu != nil {
		m.disconnectedServersMenu.SetTitle(fmt.Sprintf("🔴 Disconnected Servers (%d)", disconnectedCount))
	}
	if m.sleepingServersMenu != nil {
		m.sleepingServersMenu.SetTitle(fmt.Sprintf("💤 Sleeping Servers (%d)", sleepingCount))
	}
	if m.stoppedServersMenu != nil {
		m.stoppedServersMenu.SetTitle(fmt.Sprintf("⏹️ Stopped Servers (%d)", stoppedCount))
	}
	if m.disabledServersMenu != nil {
		m.disabledServersMenu.SetTitle(fmt.Sprintf("⏸️ Disabled Servers (%d)", disabledCount))
	}
	if m.quarantineMenu != nil {
		m.quarantineMenu.SetTitle(fmt.Sprintf("🔒 Quarantined Servers (%d)", quarantinedCount))
	}

	// First, collect all servers that should be in each status menu
	allCurrentServers := make(map[string]string) // serverName -> status
	for _, server := range connected {
		if name, ok := server["name"].(string); ok {
			allCurrentServers[name] = "connected"
		}
	}
	for _, server := range disconnected {
		if name, ok := server["name"].(string); ok {
			allCurrentServers[name] = "disconnected"
		}
	}
	for _, server := range sleeping {
		if name, ok := server["name"].(string); ok {
			allCurrentServers[name] = "sleeping"
		}
	}
	for _, server := range stopped {
		if name, ok := server["name"].(string); ok {
			allCurrentServers[name] = "stopped"
		}
	}
	for _, server := range disabled {
		if name, ok := server["name"].(string); ok {
			allCurrentServers[name] = "disabled"
		}
	}
	for _, server := range quarantined {
		if name, ok := server["name"].(string); ok {
			allCurrentServers[name] = "quarantined"
		}
	}

	// Remove servers from wrong status menus before updating
	m.removeServersFromWrongMenus(allCurrentServers)

	// Update each status-based menu
	m.updateMenuForStatus(m.connectedServersMenu, connected, m.connectedMenuItems, "connected")
	m.updateMenuForStatus(m.disconnectedServersMenu, disconnected, m.disconnectedMenuItems, "disconnected")
	m.updateMenuForStatus(m.sleepingServersMenu, sleeping, m.sleepingMenuItems, "sleeping")
	m.updateMenuForStatus(m.stoppedServersMenu, stopped, m.stoppedMenuItems, "stopped")
	m.updateMenuForStatus(m.disabledServersMenu, disabled, m.disabledMenuItems, "disabled")
	m.updateMenuForStatus(m.quarantineMenu, quarantined, m.quarantineMenuItems, "quarantined")

	m.logger.Debug("Status-based menus updated",
		zap.Int("connected", connectedCount),
		zap.Int("disconnected", disconnectedCount),
		zap.Int("sleeping", sleepingCount),
		zap.Int("stopped", stoppedCount),
		zap.Int("disabled", disabledCount),
		zap.Int("quarantined", quarantinedCount))

	// Update server count via callback
	totalCount := len(servers)
	m.logger.Debug("Attempting to update server count via callback", zap.Int("total_count", totalCount))
	if m.onServerCountUpdate != nil {
		m.logger.Debug("Calling onServerCountUpdate callback", zap.Int("total_count", totalCount))
		m.onServerCountUpdate(totalCount)
	} else {
		m.logger.Debug("onServerCountUpdate callback is nil")
	}
}

// UpdateSingleServer updates a single server in the appropriate status menu
func (m *MenuManager) UpdateSingleServer(serverName string, serverData map[string]interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Determine current status
	enabled, _ := serverData["enabled"].(bool)
	connected, _ := serverData["connected"].(bool)
	connecting, _ := serverData["connecting"].(bool)
	sleeping, _ := serverData["sleeping"].(bool)
	quarantined, _ := serverData["quarantined"].(bool)

	// Check if server has a stopped status (when server is not connecting, not connected, not sleeping, but enabled)
	stopped := enabled && !connected && !connecting && !sleeping

	// Find the correct status menu
	var targetMenu map[string]*systray.MenuItem
	var parentMenu *systray.MenuItem
	var statusName string

	if quarantined {
		targetMenu = m.quarantineMenuItems
		parentMenu = m.quarantineMenu
		statusName = "quarantined"
	} else if !enabled {
		targetMenu = m.disabledMenuItems
		parentMenu = m.disabledServersMenu
		statusName = "disabled"
	} else if connected {
		targetMenu = m.connectedMenuItems
		parentMenu = m.connectedServersMenu
		statusName = "connected"
	} else if sleeping {
		targetMenu = m.sleepingMenuItems
		parentMenu = m.sleepingServersMenu
		statusName = "sleeping"
	} else if stopped {
		targetMenu = m.stoppedMenuItems
		parentMenu = m.stoppedServersMenu
		statusName = "stopped"
	} else {
		targetMenu = m.disconnectedMenuItems
		parentMenu = m.disconnectedServersMenu
		statusName = "disconnected"
	}

	// Ensure group info is attached if missing
	m.attachGroupInfo(serverName, serverData)

	// Remove server from all OTHER status menus
	m.removeServerFromWrongMenus(serverName, statusName)

	// Update or create in the correct menu
	if menuItem, exists := targetMenu[serverName]; exists {
		// Update existing
		m.updateServerMenuItem(menuItem, serverData, serverName)
		m.logger.Debug("Updated existing server menu item",
			zap.String("server", serverName),
			zap.String("status", statusName))
	} else {
		// Create new
		if parentMenu != nil {
			menuItem := m.createServerMenuItem(parentMenu, serverData, serverName)
			targetMenu[serverName] = menuItem
			m.logger.Debug("Created new server menu item",
				zap.String("server", serverName),
				zap.String("status", statusName))
		}
	}

	// Update counts in menu titles
	m.updateMenuCounts()
}

// removeServerFromWrongMenus removes server from all status menus except the correct one
func (m *MenuManager) removeServerFromWrongMenus(serverName, correctStatus string) {
	statusMenus := map[string]map[string]*systray.MenuItem{
		"connected":     m.connectedMenuItems,
		"disconnected":  m.disconnectedMenuItems,
		"sleeping":      m.sleepingMenuItems,
		"stopped":       m.stoppedMenuItems,
		"disabled":      m.disabledMenuItems,
		"quarantined":   m.quarantineMenuItems,
	}

	for status, menuItems := range statusMenus {
		if status != correctStatus {
			if menuItem, exists := menuItems[serverName]; exists {
				menuItem.Hide()
				delete(menuItems, serverName)
				m.logger.Debug("Removed server from wrong status menu",
					zap.String("server", serverName),
					zap.String("wrong_status", status),
					zap.String("correct_status", correctStatus))
			}
		}
	}
}

// updateMenuCounts updates the counts in menu titles
func (m *MenuManager) updateMenuCounts() {
	if m.connectedServersMenu != nil {
		m.connectedServersMenu.SetTitle(fmt.Sprintf("🟢 Connected Servers (%d)", len(m.connectedMenuItems)))
	}
	if m.disconnectedServersMenu != nil {
		m.disconnectedServersMenu.SetTitle(fmt.Sprintf("🔴 Disconnected Servers (%d)", len(m.disconnectedMenuItems)))
	}
	if m.sleepingServersMenu != nil {
		m.sleepingServersMenu.SetTitle(fmt.Sprintf("💤 Sleeping Servers (%d)", len(m.sleepingMenuItems)))
	}
	if m.stoppedServersMenu != nil {
		m.stoppedServersMenu.SetTitle(fmt.Sprintf("⏹️ Stopped Servers (%d)", len(m.stoppedMenuItems)))
	}
	if m.disabledServersMenu != nil {
		m.disabledServersMenu.SetTitle(fmt.Sprintf("⏸️ Disabled Servers (%d)", len(m.disabledMenuItems)))
	}
	if m.quarantineMenu != nil {
		m.quarantineMenu.SetTitle(fmt.Sprintf("🔒 Quarantined Servers (%d)", len(m.quarantineMenuItems)))
	}
}

// updateMenuForStatus updates a specific status menu with its servers
func (m *MenuManager) updateMenuForStatus(menu *systray.MenuItem, servers []map[string]interface{}, menuItems map[string]*systray.MenuItem, status string) {
	if menu == nil {
		return
	}

	// Sort servers alphabetically
	sort.Slice(servers, func(i, j int) bool {
		nameI, _ := servers[i]["name"].(string)
		nameJ, _ := servers[j]["name"].(string)
		return nameI < nameJ
	})

	// Create a map for efficient lookup
	currentServerMap := make(map[string]map[string]interface{})
	var currentServerNames []string
	for _, server := range servers {
		if name, ok := server["name"].(string); ok {
			// Ensure group info is attached if missing
			m.attachGroupInfo(name, server)
			currentServerMap[name] = server
			currentServerNames = append(currentServerNames, name)
		}
	}

	// Remove servers that are no longer in this status
	for serverName, menuItem := range menuItems {
		if _, exists := currentServerMap[serverName]; !exists {
			m.logger.Debug("Removing server from menu", zap.String("server", serverName), zap.String("status", status))
			menuItem.Hide()
			delete(menuItems, serverName)
			// Also clean up from action items
			m.cleanupServerActionItems(serverName)
		}
	}

	// Add or update servers in this status
	for _, serverName := range currentServerNames {
		server := currentServerMap[serverName]
		if existingItem, exists := menuItems[serverName]; exists {
			// Update existing item
			m.logger.Debug("Updating existing server menu item", zap.String("server", serverName), zap.String("status", status))
			m.updateServerMenuItem(existingItem, server, serverName)
		} else {
			// Create new item
			m.logger.Debug("Creating new server menu item", zap.String("server", serverName), zap.String("status", status))
			menuItem := m.createServerMenuItem(menu, server, serverName)
			menuItems[serverName] = menuItem
		}
	}

	// Add info message for quarantine menu
	if status == "quarantined" {
		if len(servers) == 0 {
			// Empty state message
			if m.quarantineInfoEmpty == nil {
				m.quarantineInfoEmpty = menu.AddSubMenuItem("ℹ️ No servers quarantined", "")
				m.quarantineInfoEmpty.Disable()
			}
		} else {
			// Remove empty state message and add help text
			if m.quarantineInfoEmpty != nil {
				m.quarantineInfoEmpty.Hide()
				m.quarantineInfoEmpty = nil
			}
			if m.quarantineInfoHelp == nil {
				m.quarantineInfoHelp = menu.AddSubMenuItem("💡 Click server to unquarantine", "")
				m.quarantineInfoHelp.Disable()
			}
		}
	}
}

// Legacy function for backward compatibility - now delegates to status-based menus
func (m *MenuManager) UpdateUpstreamServersMenu(servers []map[string]interface{}) {
	// Use the new status-based menu system
	m.UpdateStatusBasedMenus(servers)

	// Also update the legacy menu if it exists (for backward compatibility)
	m.updateLegacyUpstreamMenu(servers)
}

// createServerMenuItem creates a menu item for a server
func (m *MenuManager) createServerMenuItem(parentMenu *systray.MenuItem, server map[string]interface{}, serverName string) *systray.MenuItem {
	status := m.getServerStatusDisplay(server)
	menuItem := parentMenu.AddSubMenuItem(status, "")

	// Special handling for quarantined servers - they should be clickable to unquarantine
	quarantined, _ := server["quarantined"].(bool)
	if quarantined {
		// Set up direct click handler for unquarantining
		go func(name string, item *systray.MenuItem) {
			for range item.ClickedCh {
				if m.onServerAction != nil {
					// Run in a new goroutine to avoid blocking the event channel
					go m.onServerAction(name, "unquarantine")
				}
			}
		}(serverName, menuItem)
	} else {
		// For non-quarantined servers, create action submenus
		m.createServerActionSubmenus(menuItem, server)
	}

	return menuItem
}

// updateServerMenuItem updates an existing server menu item
func (m *MenuManager) updateServerMenuItem(menuItem *systray.MenuItem, server map[string]interface{}, serverName string) {
	status := m.getServerStatusDisplay(server)
	menuItem.SetTitle(status)
	m.updateServerActionMenus(serverName, server)
	menuItem.Show()
}

// cleanupServerActionItems removes action items for a server
func (m *MenuManager) cleanupServerActionItems(serverName string) {
	// Clean up action items for this server
	if actionItem, ok := m.serverActionItems[serverName]; ok {
		actionItem.Hide()
		delete(m.serverActionItems, serverName)
	}
	if quarantineItem, ok := m.serverQuarantineItems[serverName]; ok {
		quarantineItem.Hide()
		delete(m.serverQuarantineItems, serverName)
	}
	if oauthItem, ok := m.serverOAuthItems[serverName]; ok {
		oauthItem.Hide()
		delete(m.serverOAuthItems, serverName)
	}
	if logItem, ok := m.serverLogItems[serverName]; ok {
		logItem.Hide()
		delete(m.serverLogItems, serverName)
	}
	if repoItem, ok := m.serverRepoItems[serverName]; ok {
		repoItem.Hide()
		delete(m.serverRepoItems, serverName)
	}
	if configItem, ok := m.serverConfigItems[serverName]; ok {
		configItem.Hide()
		delete(m.serverConfigItems, serverName)
	}
	if restartItem, ok := m.serverRestartItems[serverName]; ok {
		restartItem.Hide()
		delete(m.serverRestartItems, serverName)
	}
}

// updateLegacyUpstreamMenu maintains the old combined menu for backward compatibility
func (m *MenuManager) updateLegacyUpstreamMenu(servers []map[string]interface{}) {
	if m.upstreamServersMenu == nil {
		return // Skip if legacy menu not available
	}

	// Stability check: Don't clear existing menus if we get empty servers and we already have servers
	// This prevents UI flickering when database is temporarily unavailable
	if len(servers) == 0 && len(m.serverMenuItems) > 0 {
		m.logger.Debug("Received empty server list but existing menu items present, preserving UI state")
		return
	}

	// --- Update Title ---
	connectedServers := 0
	disconnectedServers := 0
	disabledServers := 0
	quarantinedServers := 0

	for _, server := range servers {
		enabled, _ := server["enabled"].(bool)
		connected, _ := server["connected"].(bool)
		quarantined, _ := server["quarantined"].(bool)

		if quarantined {
			quarantinedServers++
		} else if !enabled {
			disabledServers++
		} else if connected {
			connectedServers++
		} else {
			disconnectedServers++
		}
	}

	menuTitle := fmt.Sprintf("Upstream Servers (🟢%d 🔴%d ⏸️%d 🔒%d)", connectedServers, disconnectedServers, disabledServers, quarantinedServers)
	if m.upstreamServersMenu != nil {
		m.upstreamServersMenu.SetTitle(menuTitle)
	}

	// --- Create a map for efficient lookup of current servers ---
	currentServerMap := make(map[string]map[string]interface{})
	var currentServerNames []string
	for _, server := range servers {
		if name, ok := server["name"].(string); ok {
			currentServerMap[name] = server
			currentServerNames = append(currentServerNames, name)
		}
	}

	// Sort servers by status categories with better grouping
	var connected, disconnected, disabled, quarantined []string
	for _, name := range currentServerNames {
		server := currentServerMap[name]
		enabled, _ := server["enabled"].(bool)
		serverConnected, _ := server["connected"].(bool)
		serverQuarantined, _ := server["quarantined"].(bool)

		if serverQuarantined {
			quarantined = append(quarantined, name)
		} else if !enabled {
			disabled = append(disabled, name)
		} else if serverConnected {
			connected = append(connected, name)
		} else {
			disconnected = append(disconnected, name)
		}
	}

	// Sort each category alphabetically
	sort.Strings(connected)
	sort.Strings(disconnected)
	sort.Strings(disabled)
	sort.Strings(quarantined)

	// Combine in priority order: Connected, Disconnected, Disabled, Quarantined
	currentServerNames = append(connected, disconnected...)
	currentServerNames = append(currentServerNames, disabled...)
	currentServerNames = append(currentServerNames, quarantined...)

	// --- Check if we need to rebuild the menu using state hash ---
	if !m.shouldUpdateMenus(servers) {
		// No changes detected - skip menu recreation to prevent focus interruption
		m.logger.Debug("Menu state unchanged, skipping menu recreation")
		return
	}

	m.logger.Info("Menu state changed, rebuilding upstream servers menu")

	// Hide all existing menu items
	for serverName, menuItem := range m.serverMenuItems {
		menuItem.Hide()
		// Also hide sub-menu items
		if actionItem, ok := m.serverActionItems[serverName]; ok {
			actionItem.Hide()
		}
		if quarantineActionItem, ok := m.serverQuarantineItems[serverName]; ok {
			quarantineActionItem.Hide()
		}
		if oauthItem, ok := m.serverOAuthItems[serverName]; ok {
			oauthItem.Hide()
		}
		if logItem, ok := m.serverLogItems[serverName]; ok {
			logItem.Hide()
		}
		if repoItem, ok := m.serverRepoItems[serverName]; ok {
			repoItem.Hide()
		}
		if configItem, ok := m.serverConfigItems[serverName]; ok {
			configItem.Hide()
		}
	}

	// Hide all header items and separators
	for _, headerItem := range m.headerItems {
		headerItem.Hide()
	}
	for _, separatorItem := range m.separatorItems {
		separatorItem.Hide()
	}

	// Clear the tracking maps
	m.serverMenuItems = make(map[string]*systray.MenuItem)
	m.serverActionItems = make(map[string]*systray.MenuItem)
	m.serverQuarantineItems = make(map[string]*systray.MenuItem)
	m.serverOAuthItems = make(map[string]*systray.MenuItem)
	m.serverLogItems = make(map[string]*systray.MenuItem)
	m.serverRepoItems = make(map[string]*systray.MenuItem)
	m.serverConfigItems = make(map[string]*systray.MenuItem)
	m.headerItems = []*systray.MenuItem{}
	m.separatorItems = []*systray.MenuItem{}

	// Create section headers and servers grouped by status
	m.createGroupedServerMenus(currentServerNames, currentServerMap, connected, disconnected, disabled, quarantined)

	// Update server count via callback
	totalCount := len(servers)
	m.logger.Debug("Attempting to update server count via callback", zap.Int("total_count", totalCount))
	if m.onServerCountUpdate != nil {
		m.logger.Debug("Calling onServerCountUpdate callback", zap.Int("total_count", totalCount))
		m.onServerCountUpdate(totalCount)
	} else {
		m.logger.Debug("onServerCountUpdate callback is nil")
	}
}

// UpdateQuarantineMenu updates the quarantine menu using Hide/Show to prevent duplicates
// NOTE: This function is now a no-op since quarantine handling is done by UpdateStatusBasedMenus
func (m *MenuManager) UpdateQuarantineMenu(quarantinedServers []map[string]interface{}) {
	m.logger.Debug("UpdateQuarantineMenu called - delegating to status-based menu system",
		zap.Int("quarantined_count", len(quarantinedServers)))
	// The quarantine menu is now handled by UpdateStatusBasedMenus
	// This function is kept for backward compatibility but does nothing
}

// GetServerMenuItem returns the menu item for a server (for action handling)
func (m *MenuManager) GetServerMenuItem(serverName string) *systray.MenuItem {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.serverMenuItems[serverName]
}

// GetQuarantineMenuItem returns the quarantine menu item for a server (for action handling)
func (m *MenuManager) GetQuarantineMenuItem(serverName string) *systray.MenuItem {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.quarantineMenuItems[serverName]
}

// ForceRefresh clears all menu tracking to force recreation (handles systray limitations)
func (m *MenuManager) ForceRefresh() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.logger.Warn("ForceRefresh is called, which is deprecated. Check for misuse.")
	// This function is now a no-op to prevent the duplication issue.
	// The new Hide/Show logic should be used instead.
}

// calculateMenuStateHash creates a hash of the current menu state to detect changes
func (m *MenuManager) calculateMenuStateHash(servers []map[string]interface{}) string {
	var stateBuilder strings.Builder
	
	// Sort servers by name for consistent hashing
	serverNames := make([]string, 0, len(servers))
	serverMap := make(map[string]map[string]interface{})
	
	for _, server := range servers {
		if name, ok := server["name"].(string); ok {
			serverNames = append(serverNames, name)
			serverMap[name] = server
		}
	}
	sort.Strings(serverNames)
	
	// Build state string with server status and key properties
	for _, name := range serverNames {
		server := serverMap[name]
		enabled, _ := server["enabled"].(bool)
		connected, _ := server["connected"].(bool)
		quarantined, _ := server["quarantined"].(bool)
		
		stateBuilder.WriteString(fmt.Sprintf("%s:%t:%t:%t;", name, enabled, connected, quarantined))
	}
	
	// Calculate MD5 hash
	hash := md5.Sum([]byte(stateBuilder.String()))
	return fmt.Sprintf("%x", hash)
}

// shouldUpdateMenus checks if menus need to be updated based on state changes
func (m *MenuManager) shouldUpdateMenus(servers []map[string]interface{}) bool {
	m.menuUpdateMutex.Lock()
	defer m.menuUpdateMutex.Unlock()
	
	// Don't update if menu is currently open (user is interacting with it)
	if m.isMenuOpen && time.Since(m.menuOpenTime) < 30*time.Second {
		m.logger.Debug("Menu is open, skipping update to prevent interruption")
		return false
	}
	
	newHash := m.calculateMenuStateHash(servers)
	
	// For lazy loading: if data is already cached and hash hasn't changed, don't update
	if m.menuDataCached && newHash == m.lastMenuStateHash {
		m.logger.Debug("Menu data cached and unchanged, skipping update")
		return false
	}
	
	if newHash != m.lastMenuStateHash {
		m.lastMenuStateHash = newHash
		m.menuDataCached = true // Mark data as cached after update
		// Add small delay to prevent focus interruption during rapid updates
		time.Sleep(50 * time.Millisecond)
		return true
	}
	return false
}

// SetMenuOpen marks the menu as opened by user
func (m *MenuManager) SetMenuOpen() {
	m.menuUpdateMutex.Lock()
	defer m.menuUpdateMutex.Unlock()
	
	m.isMenuOpen = true
	m.menuOpenTime = time.Now()
	m.logger.Debug("Menu marked as open")
}

// SetMenuClosed marks the menu as closed by user
func (m *MenuManager) SetMenuClosed() {
	m.menuUpdateMutex.Lock()
	defer m.menuUpdateMutex.Unlock()
	
	m.isMenuOpen = false
	m.logger.Debug("Menu marked as closed")
}

func (m *MenuManager) createGroupedServerMenus(currentServerNames []string, currentServerMap map[string]map[string]interface{}, connected, disconnected, disabled, quarantined []string) {
	var lastHeaderItem *systray.MenuItem

	// Connected Servers Section
	if len(connected) > 0 {
		lastHeaderItem = m.upstreamServersMenu.AddSubMenuItem("🟢 Connected", "")
		lastHeaderItem.Disable() // Make it non-clickable header
		m.headerItems = append(m.headerItems, lastHeaderItem) // Track header item
		for _, serverName := range connected {
			serverData := currentServerMap[serverName]
			m.logger.Debug("Creating menu item for connected server", zap.String("server", serverName))
			status := m.getServerStatusDisplay(serverData)
			serverMenuItem := m.upstreamServersMenu.AddSubMenuItem("  "+status, "") // Indent with spaces
			m.serverMenuItems[serverName] = serverMenuItem
			m.createServerActionSubmenus(serverMenuItem, serverData)
		}
	}

	// Disconnected Servers Section
	if len(disconnected) > 0 {
		if lastHeaderItem != nil {
			separatorItem := m.upstreamServersMenu.AddSubMenuItem("──────────────", "")
			m.separatorItems = append(m.separatorItems, separatorItem) // Track separator
		}
		lastHeaderItem = m.upstreamServersMenu.AddSubMenuItem("🔴 Disconnected", "")
		lastHeaderItem.Disable() // Make it non-clickable header
		m.headerItems = append(m.headerItems, lastHeaderItem) // Track header item
		for _, serverName := range disconnected {
			serverData := currentServerMap[serverName]
			m.logger.Debug("Creating menu item for disconnected server", zap.String("server", serverName))
			status := m.getServerStatusDisplay(serverData)
			serverMenuItem := m.upstreamServersMenu.AddSubMenuItem("  "+status, "") // Indent with spaces
			m.serverMenuItems[serverName] = serverMenuItem
			m.createServerActionSubmenus(serverMenuItem, serverData)
		}
	}

	// Disabled Servers Section
	if len(disabled) > 0 {
		if lastHeaderItem != nil {
			separatorItem := m.upstreamServersMenu.AddSubMenuItem("──────────────", "")
			m.separatorItems = append(m.separatorItems, separatorItem) // Track separator
		}
		lastHeaderItem = m.upstreamServersMenu.AddSubMenuItem("⏸️ Disabled", "")
		lastHeaderItem.Disable() // Make it non-clickable header
		m.headerItems = append(m.headerItems, lastHeaderItem) // Track header item
		for _, serverName := range disabled {
			serverData := currentServerMap[serverName]
			m.logger.Debug("Creating menu item for disabled server", zap.String("server", serverName))
			status := m.getServerStatusDisplay(serverData)
			serverMenuItem := m.upstreamServersMenu.AddSubMenuItem("  "+status, "") // Indent with spaces
			m.serverMenuItems[serverName] = serverMenuItem
			m.createServerActionSubmenus(serverMenuItem, serverData)
		}
	}

	// Quarantined Servers Section
	if len(quarantined) > 0 {
		if lastHeaderItem != nil {
			separatorItem := m.upstreamServersMenu.AddSubMenuItem("──────────────", "")
			m.separatorItems = append(m.separatorItems, separatorItem) // Track separator
		}
		lastHeaderItem = m.upstreamServersMenu.AddSubMenuItem("🔒 Quarantined", "")
		lastHeaderItem.Disable() // Make it non-clickable header
		m.headerItems = append(m.headerItems, lastHeaderItem) // Track header item
		for _, serverName := range quarantined {
			serverData := currentServerMap[serverName]
			m.logger.Debug("Creating menu item for quarantined server", zap.String("server", serverName))
			status := m.getServerStatusDisplay(serverData)
			serverMenuItem := m.upstreamServersMenu.AddSubMenuItem("  "+status, "") // Indent with spaces
			m.serverMenuItems[serverName] = serverMenuItem
			m.createServerActionSubmenus(serverMenuItem, serverData)
		}
	}
}

// getServerStatusDisplay returns display text for a server
func (m *MenuManager) getServerStatusDisplay(server map[string]interface{}) (displayText string) {
	serverName, _ := server["name"].(string)
	enabled, _ := server["enabled"].(bool)
	connected, _ := server["connected"].(bool)
	connecting, _ := server["connecting"].(bool)
	sleeping, _ := server["sleeping"].(bool)
	quarantined, _ := server["quarantined"].(bool)

	var statusIcon string

	if quarantined {
		statusIcon = "🔒"
	} else if !enabled {
		statusIcon = "⏸️"
	} else if sleeping {
		statusIcon = "💤"
	} else if connected {
		statusIcon = "🟢"
	} else if connecting {
		statusIcon = "🟡"
	} else if enabled && !connected && !connecting && !sleeping {
		statusIcon = "⏹️"
	} else {
		statusIcon = "🔴"
	}

	var groupIcon string
	if m.serverGroups != nil {
		if g := m.findGroupForServer(serverName, server); g != nil && g.Enabled {
			// Use group Icon if available
			groupIcon = g.Icon
		}
	}

	if groupIcon != "" {
		displayText = fmt.Sprintf("%s %s %s", statusIcon, groupIcon, serverName)
	} else {
		displayText = fmt.Sprintf("%s %s", statusIcon, serverName)
	}

	return
}

func (m *MenuManager) findGroupForServer(serverName string, server map[string]interface{}) *ServerGroup {
	// Prefer explicit group_id
	if v, ok := server["group_id"]; ok {
		switch id := v.(type) {
		case int:
			for _, g := range *m.serverGroups { if g.ID == id { m.logger.Debug("Group match via group_id (int)", zap.String("server", serverName), zap.Int("group_id", id), zap.String("group", g.Name)); return g } }
		case float64:
			gid := int(id)
			for _, g := range *m.serverGroups { if g.ID == gid { m.logger.Debug("Group match via group_id (float64)", zap.String("server", serverName), zap.Int("group_id", gid), zap.String("group", g.Name)); return g } }
		}
	}
	// Then group_name
	// if gname, ok := server["group_name"].(string); ok && gname != "" {
	// 	ln := strings.ToLower(strings.TrimSpace(gname))
	// 	for name, g := range *m.serverGroups {
	// 		if strings.ToLower(strings.TrimSpace(name)) == ln { m.logger.Debug("Group match via group_name", zap.String("server", serverName), zap.String("group_name", gname), zap.String("group", g.Name)); return g }
	// 	}
	// }
	// Fallback: scan assignments
	lnServer := strings.ToLower(strings.TrimSpace(serverName))
	for _, g := range *m.serverGroups {
		if !g.Enabled { continue }
		for _, s := range g.ServerNames {
			if strings.ToLower(strings.TrimSpace(s)) == lnServer { m.logger.Debug("Group match via assignment list", zap.String("server", serverName), zap.String("group", g.Name)); return g }
		}
	}
	m.logger.Debug("No group found for server", zap.String("server", serverName))
	return nil
}

// serverSupportsOAuth determines if a server supports OAuth authentication
func (m *MenuManager) serverSupportsOAuth(server map[string]interface{}) bool {
	// Get server URL
	serverURL, ok := server["url"].(string)
	if !ok || serverURL == "" {
		// For stdio servers without URL, check if they have OAuth configuration
		if _, hasOAuth := server["oauth"]; hasOAuth {
			return true
		}
		return false // stdio servers typically don't support OAuth
	}

	// Check if it's an HTTP/HTTPS server (OAuth is typically used with HTTP-based APIs)
	urlLower := strings.ToLower(serverURL)
	if !strings.HasPrefix(urlLower, "http://") && !strings.HasPrefix(urlLower, "https://") {
		return false
	}

	// Check for OAuth-related URLs patterns
	if strings.Contains(urlLower, "oauth") || strings.Contains(urlLower, "auth") {
		return true
	}

	// For common MCP servers that we know support OAuth
	oauthDomains := []string{
		"sentry.dev",
		"github.com",
		"gitlab.com",
		"google.com",
		"googleapis.com",
		"microsoft.com",
		"oauth.com",
	}

	for _, domain := range oauthDomains {
		if strings.Contains(urlLower, domain) {
			return true
		}
	}

	// For any HTTP/HTTPS server, show OAuth option since it might support it
	// Users can try it and it will fail gracefully if not supported
	return true
}

// createServerActionSubmenus creates action submenus for a server (enable/disable, quarantine, OAuth login)
func (m *MenuManager) createServerActionSubmenus(serverMenuItem *systray.MenuItem, server map[string]interface{}) {
	serverName, _ := server["name"].(string)
	if serverName == "" {
		m.logger.Warn("createServerActionSubmenus: empty server name")
		return
	}

	m.logger.Debug("Creating action submenus for server", zap.String("server", serverName))

	enabled, _ := server["enabled"].(bool)
	quarantined, _ := server["quarantined"].(bool)

	// Enable/Disable action
	var enableText string
	if enabled {
		enableText = textDisable
	} else {
		enableText = textEnable
	}
	enableItem := serverMenuItem.AddSubMenuItem(enableText, "")
	m.serverActionItems[serverName] = enableItem
	m.logger.Debug("Added enable/disable menu item", zap.String("server", serverName), zap.String("text", enableText))

	// OAuth Login action (only for servers that support OAuth)
	if m.serverSupportsOAuth(server) && !quarantined {
		oauthItem := serverMenuItem.AddSubMenuItem("🔐 OAuth Login", "")
		m.serverOAuthItems[serverName] = oauthItem
		m.logger.Debug("Added OAuth menu item", zap.String("server", serverName))

		// Set up OAuth login click handler
		go func(name string, item *systray.MenuItem) {
			for range item.ClickedCh {
				if m.onServerAction != nil {
					// Run in new goroutines to avoid blocking the event channel
					go m.onServerAction(name, "oauth_login")
				}
			}
		}(serverName, oauthItem)
	} else {
		m.logger.Debug("Skipping OAuth menu item", zap.String("server", serverName), zap.Bool("supports_oauth", m.serverSupportsOAuth(server)), zap.Bool("quarantined", quarantined))
	}

	// Quarantine action (only if not already quarantined)
	if !quarantined {
		quarantineItem := serverMenuItem.AddSubMenuItem("Move to Quarantine", "")
		m.serverQuarantineItems[serverName] = quarantineItem
		m.logger.Debug("Added quarantine menu item", zap.String("server", serverName))

		// Set up quarantine click handler
		go func(name string, item *systray.MenuItem) {
			for range item.ClickedCh {
				if m.onServerAction != nil {
					// Run in new goroutines to avoid blocking the event channel
					go m.onServerAction(name, "quarantine")
				}
			}
		}(serverName, quarantineItem)
	} else {
		m.logger.Debug("Skipping quarantine menu item (already quarantined)", zap.String("server", serverName))
	}

	// Configuration editor action
	configItem := serverMenuItem.AddSubMenuItem("⚙️ Configure", "")
	m.serverConfigItems[serverName] = configItem
	m.logger.Debug("Added configure menu item", zap.String("server", serverName))
	go func(name string, item *systray.MenuItem) {
		for range item.ClickedCh {
			if m.onServerAction != nil {
				go m.onServerAction(name, "configure")
			}
		}
	}(serverName, configItem)

	// Restart action (only for enabled, non-quarantined servers)
	if enabled && !quarantined {
		restartItem := serverMenuItem.AddSubMenuItem("🔄 Restart Server", "")
		m.serverRestartItems[serverName] = restartItem
		m.logger.Debug("Added restart menu item", zap.String("server", serverName))

		// Set up restart click handler
		go func(name string, item *systray.MenuItem) {
			for range item.ClickedCh {
				if m.onServerAction != nil {
					// Run in new goroutines to avoid blocking the event channel
					go m.onServerAction(name, "restart")
				}
			}
		}(serverName, restartItem)
	} else {
		m.logger.Debug("Skipping restart menu item", zap.String("server", serverName), zap.Bool("enabled", enabled), zap.Bool("quarantined", quarantined))
	}

	// Log viewer action
	logItem := serverMenuItem.AddSubMenuItem("📄 Open Log", "")
	m.serverLogItems[serverName] = logItem
	go func(name string, item *systray.MenuItem) {
		for range item.ClickedCh {
			if m.onServerAction != nil {
				go m.onServerAction(name, "open_log")
			}
		}
	}(serverName, logItem)

	// Repository action if repository URL is available
	var hasRepositoryURL bool
	if repoURL, ok := server["repository_url"].(string); ok && repoURL != "" {
		hasRepositoryURL = true
	} else if urlStr, ok := server["url"].(string); ok && urlStr != "" {
		// Fallback to server URL for HTTP servers if no repository URL is set
		hasRepositoryURL = true
	}

	if hasRepositoryURL {
		repoItem := serverMenuItem.AddSubMenuItem("🔗 Open Repository", "")
		m.serverRepoItems[serverName] = repoItem
		go func(name string, item *systray.MenuItem) {
			for range item.ClickedCh {
				if m.onServerAction != nil {
					go m.onServerAction(name, "open_repo")
				}
			}
		}(serverName, repoItem)
	}

	// Add separator before group actions
	serverMenuItem.AddSeparator()

	// Group management actions
	m.createGroupActionsSubmenu(serverMenuItem, serverName)

	// Set up enable/disable click handler
	go func(name string, item *systray.MenuItem) {
		for range item.ClickedCh {
			if m.onServerAction != nil {
				// The best approach is to have the sync manager handle the toggle.
				// We send a generic 'toggle_enable' action and let the handler determine the state.
				go m.onServerAction(name, "toggle_enable")
			}
		}
	}(serverName, enableItem)
}

// createGroupActionsSubmenu creates group-related actions for a server
func (m *MenuManager) createGroupActionsSubmenu(serverMenuItem *systray.MenuItem, serverName string) {
	// Find current group assignment for this server
	var currentGroup *ServerGroup
	var currentGroupName string
	if m.serverGroups != nil {
		for groupName, group := range *m.serverGroups {
			if group.Enabled {
				for _, groupServerName := range group.ServerNames {
					if groupServerName == serverName {
						currentGroup = group
						currentGroupName = groupName
						break
					}
				}
			}
			if currentGroup != nil {
				break
			}
		}
	}

	// Create the assign to group submenu directly
	assignSubmenu := serverMenuItem.AddSubMenuItem("📋 Assign to Group", "")

	// Show current group status if assigned
	if currentGroup != nil {
		currentGroupItem := assignSubmenu.AddSubMenuItem(
			fmt.Sprintf("%s Currently in: %s", currentGroup.Icon, currentGroupName),
			fmt.Sprintf("Server is currently assigned to group '%s'", currentGroupName))
		currentGroupItem.Disable() // Make it non-clickable info

		// Add option to remove from current group
		removeFromGroupItem := assignSubmenu.AddSubMenuItem("❌ Remove from Group", "")
		go func(name, group string, item *systray.MenuItem) {
			for range item.ClickedCh {
				if m.onServerAction != nil {
					go m.onServerAction(name, fmt.Sprintf("remove_from_group:%s", group))
				}
			}
		}(serverName, currentGroupName, removeFromGroupItem)

		assignSubmenu.AddSeparator()
	}

	// Add options to assign to different groups
	if m.serverGroups != nil && len(*m.serverGroups) > 0 {
		// List all available groups (except current one)
		for groupName, group := range *m.serverGroups {
			if group.Enabled && groupName != currentGroupName {
				groupItem := assignSubmenu.AddSubMenuItem(
					fmt.Sprintf("%s %s (%d servers)", group.Icon, groupName, len(group.ServerNames)),
					fmt.Sprintf("Assign server to group '%s'", groupName))

				go func(name, group string, item *systray.MenuItem) {
					for range item.ClickedCh {
						if m.onServerAction != nil {
							go m.onServerAction(name, fmt.Sprintf("assign_to_group:%s", group))
						}
					}
				}(serverName, groupName, groupItem)
			}
		}
	} else {
		// No groups available
		noGroupsItem := assignSubmenu.AddSubMenuItem("📋 No groups available", "")
		noGroupsItem.Disable()
	}
}

// updateServerActionMenus updates the action submenu items for an existing server
func (m *MenuManager) updateServerActionMenus(serverName string, server map[string]interface{}) {
	enabled, _ := server["enabled"].(bool)
	quarantined, _ := server["quarantined"].(bool)

	// Update enable/disable action menu text
	if actionItem, exists := m.serverActionItems[serverName]; exists {
		var enableText string
		if enabled {
			enableText = textDisable
		} else {
			enableText = textEnable
		}
		actionItem.SetTitle(enableText)

		m.logger.Debug("Updated action menu for server",
			zap.String("server", serverName),
			zap.String("action", enableText))
	}

	// Update restart menu visibility - only show for enabled, non-quarantined servers
	if restartItem, exists := m.serverRestartItems[serverName]; exists {
		if enabled && !quarantined {
			restartItem.Show()
		} else {
			restartItem.Hide()
		}
		m.logger.Debug("Updated restart menu visibility for server",
			zap.String("server", serverName),
			zap.Bool("enabled", enabled),
			zap.Bool("quarantined", quarantined),
			zap.Bool("visible", enabled && !quarantined))
	}
}

// SynchronizationManager coordinates between state manager and menu manager
type SynchronizationManager struct {
	stateManager *ServerStateManager
	menuManager  *MenuManager
	logger       *zap.SugaredLogger

	// Background sync control
	ctx       context.Context
	cancel    context.CancelFunc
	syncTimer *time.Timer

	// User activity tracking for adaptive sync
	lastUserActivity time.Time
	activityMu       sync.RWMutex
}

// NewSynchronizationManager creates a new synchronization manager
func NewSynchronizationManager(stateManager *ServerStateManager, menuManager *MenuManager, logger *zap.SugaredLogger) *SynchronizationManager {
	ctx, cancel := context.WithCancel(context.Background())
	return &SynchronizationManager{
		stateManager:     stateManager,
		menuManager:      menuManager,
		logger:           logger,
		ctx:              ctx,
		cancel:           cancel,
		lastUserActivity: time.Now(),
	}
}

// Start begins background synchronization
// NOTE: This now only performs initial sync. Events handle updates.
func (m *SynchronizationManager) Start() {
	// Perform initial menu population
	if err := m.performInitialSync(); err != nil {
		m.logger.Error("Initial sync failed", zap.Error(err))
	}

	m.logger.Info("Initial synchronization completed, now using event-based updates")
}

// performInitialSync performs one-time initial menu population
func (m *SynchronizationManager) performInitialSync() error {
	m.logger.Info("Performing initial menu sync")

	allServers, err := m.stateManager.GetAllServers()
	if err != nil {
		return fmt.Errorf("failed to get all servers: %w", err)
	}

	// Populate menu initial
	m.menuManager.UpdateUpstreamServersMenu(allServers)

	quarantinedServers, err := m.stateManager.GetQuarantinedServers()
	if err != nil {
		return fmt.Errorf("failed to get quarantined servers: %w", err)
	}

	m.menuManager.UpdateQuarantineMenu(quarantinedServers)

	m.logger.Info("Initial menu populated",
		zap.Int("all_servers", len(allServers)),
		zap.Int("quarantined_servers", len(quarantinedServers)))

	return nil
}

// Stop stops background synchronization
func (m *SynchronizationManager) Stop() {
	if m.cancel != nil {
		m.cancel()
	}
	if m.syncTimer != nil {
		m.syncTimer.Stop()
	}
}

// SyncNow performs immediate synchronization
func (m *SynchronizationManager) SyncNow() error {
	return m.performSync()
}

// SyncDelayed schedules a delayed synchronization to batch updates
func (m *SynchronizationManager) SyncDelayed() {
	if m.syncTimer != nil {
		m.syncTimer.Stop()
	}
	m.syncTimer = time.AfterFunc(1*time.Second, func() {
		if err := m.performSync(); err != nil {
			m.logger.Error("Delayed sync failed", zap.Error(err))
		}
	})
}

// NotifyUserActivity records user interaction to enable adaptive sync frequency
func (m *SynchronizationManager) NotifyUserActivity() {
	m.activityMu.Lock()
	m.lastUserActivity = time.Now()
	m.activityMu.Unlock()

	// Also trigger an immediate sync to ensure up-to-date menu when user is active
	m.SyncDelayed()
}

// syncLoop is DEPRECATED and replaced by event-based synchronization
// This method is kept for backward compatibility but does nothing.
// Events from the EventBus now trigger menu updates automatically.
func (m *SynchronizationManager) syncLoop() {
	m.logger.Warn("syncLoop called but is deprecated - using event-based sync instead")
	// Event-based synchronization handles all updates now
	// This method intentionally does nothing
	<-m.ctx.Done()
}

// performSync performs the actual synchronization
func (m *SynchronizationManager) performSync() error {
	// Check if the state manager's server is available and running
	// If not, skip the sync to avoid database errors
	//
	// FIXME: remove this if no issue with DB connection
	//
	// if m.stateManager.server != nil && !m.stateManager.server.IsRunning() {
	// 	m.logger.Debug("Server is stopped, skipping synchronization")
	// 	return nil
	// }

	// Get current state with error handling for database issues
	allServers, err := m.stateManager.GetAllServers()
	if err != nil {
		// Check if it's a database closed error and handle gracefully
		if strings.Contains(err.Error(), "database not available") {
			m.logger.Debug("Database not available, skipping servers menu update to preserve UI state")
			// Don't update servers menu to preserve current state
		} else {
			m.logger.Error("Failed to get all servers", zap.Error(err))
			return fmt.Errorf("failed to get all servers: %w", err)
		}
	} else {
		// Only update menu if we have valid data
		m.menuManager.UpdateUpstreamServersMenu(allServers)
	}

	quarantinedServers, err := m.stateManager.GetQuarantinedServers()
	if err != nil {
		// Check if it's a database closed error and handle gracefully
		if strings.Contains(err.Error(), "database not available") {
			m.logger.Debug("Database not available, skipping quarantine menu update to preserve UI state")
			// Don't update quarantine menu to preserve current state
		} else {
			m.logger.Error("Failed to get quarantined servers", zap.Error(err))
			return fmt.Errorf("failed to get quarantined servers: %w", err)
		}
	} else {
		// Only update menu if we have valid data
		m.menuManager.UpdateQuarantineMenu(quarantinedServers)
	}

	return nil
}

// HandleServerQuarantine handles server quarantine with full synchronization
func (m *SynchronizationManager) HandleServerQuarantine(serverName string, quarantined bool) error {
	m.logger.Info("Handling server quarantine",
		zap.String("server", serverName),
		zap.Bool("quarantined", quarantined))

	// Update state
	if err := m.stateManager.QuarantineServer(serverName, quarantined); err != nil {
		return err
	}

	// Force immediate sync
	return m.SyncNow()
}

// HandleServerUnquarantine handles server unquarantine with full synchronization
func (m *SynchronizationManager) HandleServerUnquarantine(serverName string) error {
	m.logger.Info("Handling server unquarantine", zap.String("server", serverName))

	// Update state
	if err := m.stateManager.UnquarantineServer(serverName); err != nil {
		return err
	}

	// Force immediate sync
	return m.SyncNow()
}

// HandleServerEnable handles server enable/disable with full synchronization
func (m *SynchronizationManager) HandleServerEnable(serverName string, enabled bool) error {
	action := "disable"
	if enabled {
		action = "enable"
	}
	m.logger.Info("Handling server enable/disable",
		zap.String("server", serverName),
		zap.String("action", action))

	// Update state
	if err := m.stateManager.EnableServer(serverName, enabled); err != nil {
		return err
	}

	// Force immediate sync
	return m.SyncNow()
}

// Note: stringSlicesEqual function is defined in tray.go

// attachGroupInfo populates group_id/group_name on server map if missing using serverGroups assignments
func (m *MenuManager) attachGroupInfo(serverName string, server map[string]interface{}) {
	if m.serverGroups == nil { return }
	if _, ok := server["group_id"]; ok { return }
	if gn, ok := server["group_name"].(string); ok && gn != "" { return }
	ln := strings.ToLower(strings.TrimSpace(serverName))
	for _, g := range *m.serverGroups {
		if !g.Enabled { continue }
		for _, s := range g.ServerNames {
			if strings.ToLower(strings.TrimSpace(s)) == ln {
				server["group_id"] = g.ID
				// stop setting legacy group_name
				m.logger.Debug("Attached group info to server", zap.String("server", serverName), zap.Int("group_id", g.ID))
				return
			}
		}
	}
}

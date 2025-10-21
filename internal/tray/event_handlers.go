//go:build !nogui && !headless && !linux

package tray

import (
	"time"

	"mcpproxy-go/internal/events"
	"mcpproxy-go/internal/upstream/types"

	"go.uber.org/zap"
)

// EventManager manages event subscriptions for the tray
type EventManager struct {
	eventBus    *events.EventBus
	syncManager *SynchronizationManager
	menuManager *MenuManager
	logger      *zap.SugaredLogger

	// Debouncing for menu updates
	menuUpdateChan     chan string // serverName or empty for full sync
	menuUpdateDebounce time.Duration
	stopChan           chan struct{}
}

// NewEventManager creates a new event manager for tray
func NewEventManager(
	eventBus *events.EventBus,
	syncManager *SynchronizationManager,
	menuManager *MenuManager,
	logger *zap.SugaredLogger,
) *EventManager {
	em := &EventManager{
		eventBus:           eventBus,
		syncManager:        syncManager,
		menuManager:        menuManager,
		logger:             logger,
		menuUpdateChan:     make(chan string, 100),
		menuUpdateDebounce: 100 * time.Millisecond,
		stopChan:           make(chan struct{}),
	}

	// Subscribe to events
	em.subscribeToEvents()

	// Start debounced menu updater
	go em.debouncedMenuUpdater()

	return em
}

// Stop stops the event manager
func (em *EventManager) Stop() {
	close(em.stopChan)
}

// subscribeToEvents subscribes to all relevant events
func (em *EventManager) subscribeToEvents() {
	// State change events (server connected/disconnected)
	em.eventBus.Subscribe(events.EventStateChange, em.handleStateChange)

	// Config change events (enabled/disabled/quarantined)
	em.eventBus.Subscribe(events.EventConfigChange, em.handleConfigChange)

	// Tools discovered events (update counts)
	em.eventBus.Subscribe(events.EventToolsDiscovered, em.handleToolsDiscovered)

	em.logger.Info("Event subscriptions initialized",
		zap.Int("state_change_subscribers", em.eventBus.SubscriberCount(events.EventStateChange)),
		zap.Int("config_change_subscribers", em.eventBus.SubscriberCount(events.EventConfigChange)),
		zap.Int("tools_discovered_subscribers", em.eventBus.SubscriberCount(events.EventToolsDiscovered)))
}

// handleStateChange handles server state changes
func (em *EventManager) handleStateChange(event events.Event) {
	data, ok := event.Data.(events.StateChangeData)
	if !ok {
		em.logger.Error("Invalid state change data")
		return
	}

	em.logger.Info("State change event received",
		zap.String("server", event.ServerName),
		zap.String("old_state", data.OldState.String()),
		zap.String("new_state", data.NewState.String()))

	// Trigger menu update only for significant state changes
	if em.isSignificantStateChange(data.OldState, data.NewState) {
		em.triggerMenuUpdate(event.ServerName)
	}
}

// handleConfigChange handles configuration changes
func (em *EventManager) handleConfigChange(event events.Event) {
	data, ok := event.Data.(events.ConfigChangeData)
	if !ok {
		em.logger.Error("Invalid config change data")
		return
	}

	em.logger.Info("Config change event received",
		zap.String("server", event.ServerName),
		zap.String("action", data.Action))

	// Always trigger menu update for config changes
	em.triggerMenuUpdate(event.ServerName)
}

// handleToolsDiscovered handles tools discovery completion
func (em *EventManager) handleToolsDiscovered(event events.Event) {
	data, ok := event.Data.(events.ToolsDiscoveredData)
	if !ok {
		em.logger.Error("Invalid tools discovered data")
		return
	}

	em.logger.Debug("Tools discovered event received",
		zap.String("server", event.ServerName),
		zap.Int("count", data.Count))

	// Trigger menu update to show tool count
	em.triggerMenuUpdate(event.ServerName)
}

// isSignificantStateChange checks if state change requires menu update
func (em *EventManager) isSignificantStateChange(oldState, newState types.ConnectionState) bool {
	// Only update menu if:
	// - Connected/Disconnected status changes
	// - Error state changes
	// - Connecting state changes (for visual feedback)

	oldConnected := oldState == types.StateReady
	newConnected := newState == types.StateReady

	oldError := oldState == types.StateError
	newError := newState == types.StateError

	oldConnecting := oldState == types.StateConnecting || oldState == types.StateAuthenticating || oldState == types.StateDiscovering
	newConnecting := newState == types.StateConnecting || newState == types.StateAuthenticating || newState == types.StateDiscovering

	return oldConnected != newConnected || oldError != newError || oldConnecting != newConnecting
}

// triggerMenuUpdate requests a debounced menu update for a specific server
func (em *EventManager) triggerMenuUpdate(serverName string) {
	select {
	case em.menuUpdateChan <- serverName:
		em.logger.Debug("Menu update queued", zap.String("server", serverName))
	default:
		// Channel full, update already pending
		em.logger.Debug("Menu update channel full, skipping", zap.String("server", serverName))
	}
}

// debouncedMenuUpdater batches menu updates to prevent rapid rebuilds
func (em *EventManager) debouncedMenuUpdater() {
	var timer *time.Timer
	var pendingUpdates = make(map[string]bool) // serverName -> true

	for {
		select {
		case serverName := <-em.menuUpdateChan:
			// Add to pending updates
			pendingUpdates[serverName] = true

			// Cancel existing timer
			if timer != nil {
				timer.Stop()
			}

			// Wait for debounce period
			timer = time.AfterFunc(em.menuUpdateDebounce, func() {
				// Process all pending updates
				em.processPendingUpdates(pendingUpdates)

				// Clear pending updates
				pendingUpdates = make(map[string]bool)
			})

		case <-em.stopChan:
			if timer != nil {
				timer.Stop()
			}
			return
		}
	}
}

// processPendingUpdates processes all pending menu updates
func (em *EventManager) processPendingUpdates(pendingUpdates map[string]bool) {
	if len(pendingUpdates) == 0 {
		return
	}

	em.logger.Debug("Processing pending menu updates",
		zap.Int("count", len(pendingUpdates)),
		zap.Strings("servers", em.mapKeysToSlice(pendingUpdates)))

	// If we have many pending updates (>5), do a full sync instead
	if len(pendingUpdates) > 5 {
		em.logger.Info("Many pending updates, performing full sync",
			zap.Int("count", len(pendingUpdates)))

		if err := em.syncManager.SyncNow(); err != nil {
			em.logger.Error("Full menu sync failed", zap.Error(err))
		}
		return
	}

	// Otherwise, update individual servers
	for serverName := range pendingUpdates {
		if err := em.updateSingleServerInMenu(serverName); err != nil {
			em.logger.Error("Failed to update server in menu",
				zap.String("server", serverName),
				zap.Error(err))
		}
	}
}

// updateSingleServerInMenu updates a single server in the menu
func (em *EventManager) updateSingleServerInMenu(serverName string) error {
	// Get data for this specific server
	serverData, err := em.syncManager.stateManager.GetServerByName(serverName)
	if err != nil {
		return err
	}

	// Update only this menu item
	em.menuManager.UpdateSingleServer(serverName, serverData)

	em.logger.Debug("Single server menu updated",
		zap.String("server", serverName))

	return nil
}

// mapKeysToSlice converts map keys to slice (helper function)
func (em *EventManager) mapKeysToSlice(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

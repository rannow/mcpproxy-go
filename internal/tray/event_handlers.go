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

	// Event channels
	stateChangeChan     <-chan events.Event
	configChangeChan    <-chan events.Event
	toolsDiscoveredChan <-chan events.Event

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
	em.stateChangeChan = em.eventBus.Subscribe(events.EventStateChange)

	// Config change events (enabled/disabled/quarantined)
	em.configChangeChan = em.eventBus.Subscribe(events.EventConfigChange)

	// Tools updated events (update counts)
	em.toolsDiscoveredChan = em.eventBus.Subscribe(events.ToolsUpdated)

	// Start goroutines to handle events from channels
	go em.handleStateChangeEvents()
	go em.handleConfigChangeEvents()
	go em.handleToolsDiscoveredEvents()

	em.logger.Info("Event subscriptions initialized",
		zap.Int("state_change_subscribers", em.eventBus.SubscriberCount(events.EventStateChange)),
		zap.Int("config_change_subscribers", em.eventBus.SubscriberCount(events.EventConfigChange)),
		zap.Int("tools_updated_subscribers", em.eventBus.SubscriberCount(events.ToolsUpdated)))
}

// handleStateChangeEvents listens for state change events and processes them
func (em *EventManager) handleStateChangeEvents() {
	for {
		select {
		case event, ok := <-em.stateChangeChan:
			if !ok {
				em.logger.Info("State change channel closed")
				return
			}

			data, ok := event.Data.(events.StateChangeData)
			if !ok {
				em.logger.Error("Invalid state change data")
				continue
			}

			em.logger.Info("State change event received",
				zap.String("server", event.ServerName),
				zap.Any("old_state", data.OldState),
				zap.Any("new_state", data.NewState))

			// Trigger menu update only for significant state changes
			if em.isSignificantStateChange(data.OldState, data.NewState) {
				em.triggerMenuUpdate(event.ServerName)
			}

		case <-em.stopChan:
			em.logger.Info("State change event handler stopped")
			return
		}
	}
}

// handleConfigChangeEvents listens for config change events and processes them
func (em *EventManager) handleConfigChangeEvents() {
	for {
		select {
		case event, ok := <-em.configChangeChan:
			if !ok {
				em.logger.Info("Config change channel closed")
				return
			}

			data, ok := event.Data.(events.ConfigChangeData)
			if !ok {
				em.logger.Error("Invalid config change data")
				continue
			}

			em.logger.Info("Config change event received",
				zap.String("server", event.ServerName),
				zap.String("action", data.Action))

			// Always trigger menu update for config changes
			em.triggerMenuUpdate(event.ServerName)

		case <-em.stopChan:
			em.logger.Info("Config change event handler stopped")
			return
		}
	}
}

// handleToolsDiscoveredEvents listens for tools discovered events and processes them
func (em *EventManager) handleToolsDiscoveredEvents() {
	for {
		select {
		case event, ok := <-em.toolsDiscoveredChan:
			if !ok {
				em.logger.Info("Tools discovered channel closed")
				return
			}

			// ToolsUpdated event data is typically map[string]interface{} with count
			if dataMap, ok := event.Data.(map[string]interface{}); ok {
				if count, ok := dataMap["count"].(int); ok {
					em.logger.Debug("Tools discovered event received",
						zap.String("server", event.ServerName),
						zap.Int("count", count))

					// Trigger menu update to show tool count
					em.triggerMenuUpdate(event.ServerName)
				}
			}

		case <-em.stopChan:
			em.logger.Info("Tools discovered event handler stopped")
			return
		}
	}
}

// isSignificantStateChange checks if state change requires menu update
func (em *EventManager) isSignificantStateChange(oldState, newState interface{}) bool {
	// Try to convert to ConnectionState for comparison
	oldConnState, oldOk := oldState.(types.ConnectionState)
	newConnState, newOk := newState.(types.ConnectionState)

	// If we can't convert, assume it's significant (be conservative)
	if !oldOk || !newOk {
		return true
	}

	// Only update menu if:
	// - Connected/Disconnected status changes
	// - Error state changes
	// - Connecting state changes (for visual feedback)

	oldConnected := oldConnState == types.StateReady
	newConnected := newConnState == types.StateReady

	oldError := oldConnState == types.StateError
	newError := newConnState == types.StateError

	oldConnecting := oldConnState == types.StateConnecting || oldConnState == types.StateAuthenticating || oldConnState == types.StateDiscovering
	newConnecting := newConnState == types.StateConnecting || newConnState == types.StateAuthenticating || newConnState == types.StateDiscovering

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

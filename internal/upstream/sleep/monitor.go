package sleep

import (
	"sync"
	"time"

	"mcpproxy-go/internal/upstream/types"

	"go.uber.org/zap"
)

// InactivityMonitor tracks server activity and transitions inactive servers to sleep mode.
// It monitors the last tool call timestamp for each server and automatically puts servers
// to sleep after the configured inactivity timeout (InactivitySleepMs).
type InactivityMonitor struct {
	mu               sync.RWMutex
	lastActivity     map[string]time.Time // serverName -> last activity timestamp
	logger           *zap.Logger
	inactivityPeriod time.Duration // Configured inactivity timeout
	checkInterval    time.Duration // How often to check for inactive servers

	// Manager access for state transitions
	getClientFunc func(serverName string) ClientInterface

	// Control channels
	stopCh chan struct{}
	doneCh chan struct{}
}

// ClientInterface defines the minimal interface needed from managed.Client
// This avoids circular dependencies between packages
type ClientInterface interface {
	GetStateManager() *types.StateManager
	GetName() string
}

// NewInactivityMonitor creates a new inactivity monitor with the specified configuration.
// getClientFunc should return the managed.Client for a given server name.
func NewInactivityMonitor(
	inactivityPeriod time.Duration,
	getClientFunc func(serverName string) ClientInterface,
	logger *zap.Logger,
) *InactivityMonitor {
	// Check interval is 10% of inactivity period, minimum 30s, maximum 5min
	checkInterval := inactivityPeriod / 10
	if checkInterval < 30*time.Second {
		checkInterval = 30 * time.Second
	}
	if checkInterval > 5*time.Minute {
		checkInterval = 5 * time.Minute
	}

	return &InactivityMonitor{
		lastActivity:     make(map[string]time.Time),
		logger:           logger,
		inactivityPeriod: inactivityPeriod,
		checkInterval:    checkInterval,
		getClientFunc:    getClientFunc,
		stopCh:           make(chan struct{}),
		doneCh:           make(chan struct{}),
	}
}

// Start begins monitoring server activity in a background goroutine.
// It periodically checks for inactive servers and transitions them to sleep.
func (im *InactivityMonitor) Start() {
	im.logger.Info("Starting inactivity monitor",
		zap.Duration("inactivity_period", im.inactivityPeriod),
		zap.Duration("check_interval", im.checkInterval))

	go im.monitorLoop()
}

// Stop gracefully shuts down the inactivity monitor.
func (im *InactivityMonitor) Stop() {
	im.logger.Info("Stopping inactivity monitor")
	close(im.stopCh)
	<-im.doneCh
	im.logger.Info("Inactivity monitor stopped")
}

// RecordActivity records that a server was active at the current time.
// This should be called whenever a tool call is made to a server.
func (im *InactivityMonitor) RecordActivity(serverName string) {
	im.mu.Lock()
	defer im.mu.Unlock()

	now := time.Now()
	im.lastActivity[serverName] = now

	im.logger.Debug("Recorded server activity",
		zap.String("server", serverName),
		zap.Time("timestamp", now))
}

// GetLastActivity returns the last activity timestamp for a server.
// Returns zero time if server has no recorded activity.
func (im *InactivityMonitor) GetLastActivity(serverName string) time.Time {
	im.mu.RLock()
	defer im.mu.RUnlock()
	return im.lastActivity[serverName]
}

// RemoveServer removes activity tracking for a server (e.g., when server is removed).
func (im *InactivityMonitor) RemoveServer(serverName string) {
	im.mu.Lock()
	defer im.mu.Unlock()
	delete(im.lastActivity, serverName)

	im.logger.Debug("Removed server from activity tracking",
		zap.String("server", serverName))
}

// monitorLoop is the main monitoring goroutine that checks for inactive servers.
func (im *InactivityMonitor) monitorLoop() {
	defer close(im.doneCh)

	ticker := time.NewTicker(im.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-im.stopCh:
			return
		case <-ticker.C:
			im.checkInactivity()
		}
	}
}

// checkInactivity scans all tracked servers and transitions inactive ones to sleep.
func (im *InactivityMonitor) checkInactivity() {
	im.mu.RLock()
	now := time.Now()

	// Build list of servers to check (avoid holding lock during state transitions)
	var serversToCheck []struct {
		name         string
		lastActivity time.Time
	}
	for serverName, lastActivity := range im.lastActivity {
		serversToCheck = append(serversToCheck, struct {
			name         string
			lastActivity time.Time
		}{serverName, lastActivity})
	}
	im.mu.RUnlock()

	// Check each server for inactivity
	for _, server := range serversToCheck {
		timeSinceActivity := now.Sub(server.lastActivity)

		if timeSinceActivity >= im.inactivityPeriod {
			im.transitionToSleep(server.name, timeSinceActivity)
		}
	}
}

// transitionToSleep attempts to transition a server to sleep mode.
func (im *InactivityMonitor) transitionToSleep(serverName string, inactiveDuration time.Duration) {
	// Get client via callback
	client := im.getClientFunc(serverName)
	if client == nil {
		im.logger.Warn("Cannot transition to sleep: client not found",
			zap.String("server", serverName))
		return
	}

	stateManager := client.GetStateManager()
	if stateManager == nil {
		im.logger.Warn("Cannot transition to sleep: state manager not available",
			zap.String("server", serverName))
		return
	}

	// Check current state - only transition if active or lazy_loading
	currentState := stateManager.GetServerState()
	if currentState != types.StateActive && currentState != types.StateLazyLoading {
		// Server is not in a state that should sleep (e.g., already disabled, quarantined)
		im.logger.Debug("Skipping sleep transition: server not in active/lazy_loading state",
			zap.String("server", serverName),
			zap.String("current_state", string(currentState)))
		return
	}

	// Transition to sleep
	err := stateManager.TransitionServerState(types.StateSleep)
	if err != nil {
		im.logger.Error("Failed to transition server to sleep",
			zap.String("server", serverName),
			zap.Duration("inactive_duration", inactiveDuration),
			zap.Error(err))
		return
	}

	im.logger.Info("Transitioned server to sleep due to inactivity",
		zap.String("server", serverName),
		zap.Duration("inactive_duration", inactiveDuration),
		zap.Time("last_activity", im.GetLastActivity(serverName)))

	// Remove from tracking since it's now asleep
	// It will be re-added when woken up and used again
	im.RemoveServer(serverName)
}

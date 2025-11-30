// Package types provides type definitions for upstream connection management.
// MED-001: StateManager extracted from types.go for better separation of concerns.
// This file contains the core StateManager struct and its primary methods.
package types

import (
	"fmt"
	"sync"
	"time"

	"mcpproxy-go/internal/config"
	"mcpproxy-go/internal/events"
)

// StateManager manages the state transitions for an upstream connection
type StateManager struct {
	mu sync.RWMutex

	// Runtime connection state (in-memory only)
	currentState    ConnectionState
	lastError       error
	retryCount      int
	lastRetryTime   time.Time
	serverName      string
	serverVersion   string
	lastOAuthAttempt time.Time
	oauthRetryCount  int
	isOAuthError     bool
	firstAttemptTime time.Time // Time of first connection attempt
	connectedAt      time.Time // Time when connection was established

	// Auto-disable tracking
	consecutiveFailures  int    // Consecutive failures counter
	autoDisabled         bool   // Auto-disable flag
	autoDisableReason    string // Reason for auto-disable
	autoDisableThreshold int    // Threshold for auto-disable (default: DefaultAutoDisableThreshold)
	lastSuccessTime      time.Time // Last successful connection time

	// Runtime-only UI state (NOT persisted)
	// IMPORTANT: This field should NEVER be saved to config or database
	// When app restarts, all userStopped flags are cleared and servers return to their original startup_mode
	userStopped bool // User manually stopped via tray UI (runtime-only, never persisted)

	// Persisted configuration state (stored in database)
	serverState ServerState // Current server state (active, disabled, quarantined, etc.)

	// Event bus for publishing state changes
	eventBus *events.Bus

	// Callbacks for state transitions
	onStateChange func(oldState, newState ConnectionState, info *ConnectionInfo)
}

// NewStateManager creates a new state manager
func NewStateManager() *StateManager {
	return &StateManager{
		currentState:         StateDisconnected,
		serverState:          StateActive, // Default to active until config loaded
		autoDisableThreshold: DefaultAutoDisableThreshold, // HIGH-003: Use consolidated constant
	}
}

// SetStateChangeCallback sets a callback function that will be called on state changes
func (sm *StateManager) SetStateChangeCallback(callback func(oldState, newState ConnectionState, info *ConnectionInfo)) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.onStateChange = callback
}

// GetStateChangeCallback returns the current state change callback
func (sm *StateManager) GetStateChangeCallback() func(oldState, newState ConnectionState, info *ConnectionInfo) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.onStateChange
}

// GetState returns the current connection state
func (sm *StateManager) GetState() ConnectionState {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.currentState
}

// GetConnectionInfo returns detailed connection information
func (sm *StateManager) GetConnectionInfo() ConnectionInfo {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	return ConnectionInfo{
		State:                sm.currentState,
		LastError:            sm.lastError,
		RetryCount:           sm.retryCount,
		LastRetryTime:        sm.lastRetryTime,
		ServerName:           sm.serverName,
		ServerVersion:        sm.serverVersion,
		LastOAuthAttempt:     sm.lastOAuthAttempt,
		OAuthRetryCount:      sm.oauthRetryCount,
		IsOAuthError:         sm.isOAuthError,
		FirstAttemptTime:     sm.firstAttemptTime,
		ConnectedAt:          sm.connectedAt,
		ConsecutiveFailures:  sm.consecutiveFailures,
		AutoDisabled:         sm.autoDisabled,
		AutoDisableReason:    sm.autoDisableReason,
		AutoDisableThreshold: sm.autoDisableThreshold,
		LastSuccessTime:      sm.lastSuccessTime,
	}
}

// TransitionTo transitions to a new state
// HIGH-007: Validates transition but allows it for backward compatibility.
// Invalid transitions are logged via event bus for observability.
func (sm *StateManager) TransitionTo(newState ConnectionState) {
	sm.mu.Lock()
	oldState := sm.currentState

	// Validate transition
	if err := sm.ValidateTransition(oldState, newState); err != nil {
		// HIGH-007: Log invalid transitions via event bus for observability
		// We allow the transition for backward compatibility but emit a warning event
		if sm.eventBus != nil && sm.serverName != "" {
			go sm.eventBus.Publish(events.Event{
				Type:       events.ServerStateChanged,
				ServerName: sm.serverName,
				Data: map[string]interface{}{
					"warning":    "invalid_state_transition",
					"error":      err.Error(),
					"from_state": oldState.String(),
					"to_state":   newState.String(),
					"allowed":    true, // Allowed for backward compat
					"source":     "connection_state_manager",
				},
			})
		}
	}

	sm.currentState = newState

	// Record first connection attempt
	if newState == StateConnecting && sm.firstAttemptTime.IsZero() {
		sm.firstAttemptTime = time.Now()
	}

	// Record successful connection time
	if newState == StateReady {
		sm.connectedAt = time.Now()
		sm.lastSuccessTime = time.Now()
		sm.lastError = nil
		sm.retryCount = 0
		sm.consecutiveFailures = 0 // Reset consecutive failures on success
		sm.isOAuthError = false
	}

	info := sm.buildConnectionInfo()
	callback := sm.onStateChange
	sm.mu.Unlock()

	// Publish event to event bus (non-blocking)
	sm.publishConnectionStateChange(oldState, newState, &info)

	// HIGH-004: Call the callback asynchronously to avoid deadlocks
	// All callback invocations are now standardized to use goroutines
	if callback != nil {
		go callback(oldState, newState, &info)
	}
}

// SetError sets an error and transitions to error state
func (sm *StateManager) SetError(err error) {
	sm.mu.Lock()

	oldState := sm.currentState
	sm.currentState = StateError
	sm.lastError = err
	sm.retryCount++
	sm.consecutiveFailures++ // Increment consecutive failures
	sm.lastRetryTime = time.Now()

	info := sm.buildConnectionInfo()
	callback := sm.onStateChange
	sm.mu.Unlock()

	// Publish event to event bus (non-blocking)
	sm.publishConnectionStateChange(oldState, StateError, &info)

	// HIGH-004: Call the callback asynchronously to avoid deadlocks
	// All callback invocations are now standardized to use goroutines
	if callback != nil {
		go callback(oldState, StateError, &info)
	}
}

// SetServerInfo sets the server information
func (sm *StateManager) SetServerInfo(name, version string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.serverName = name
	sm.serverVersion = version
}

// IsState checks if the current state matches the given state
func (sm *StateManager) IsState(state ConnectionState) bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.currentState == state
}

// IsReady returns true if the connection is ready for requests
func (sm *StateManager) IsReady() bool {
	return sm.IsState(StateReady)
}

// IsConnecting returns true if the connection is in progress
func (sm *StateManager) IsConnecting() bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.currentState == StateConnecting || sm.currentState == StateAuthenticating || sm.currentState == StateDiscovering
}

// ValidateTransition validates if a state transition is allowed
func (sm *StateManager) ValidateTransition(from, to ConnectionState) error {
	// Define valid transitions
	validTransitions := map[ConnectionState][]ConnectionState{
		StateDisconnected:   {StateConnecting},
		StateConnecting:     {StateAuthenticating, StateDiscovering, StateReady, StateError, StateDisconnected}, // Allow direct to Ready for OAuth flows
		StateAuthenticating: {StateConnecting, StateDiscovering, StateReady, StateError, StateDisconnected},
		StateDiscovering:    {StateReady, StateError, StateDisconnected},
		StateReady:          {StateError, StateDisconnected},
		StateError:          {StateConnecting, StateDisconnected},
	}

	allowed, exists := validTransitions[from]
	if !exists {
		return fmt.Errorf("invalid source state: %s", from)
	}

	for _, validTo := range allowed {
		if validTo == to {
			return nil
		}
	}

	return fmt.Errorf("invalid transition from %s to %s", from, to)
}

// Reset resets the state manager to disconnected state
func (sm *StateManager) Reset() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	oldState := sm.currentState
	sm.currentState = StateDisconnected
	sm.lastError = nil
	sm.retryCount = 0
	sm.lastRetryTime = time.Time{}
	sm.serverName = ""
	sm.serverVersion = ""
	sm.lastOAuthAttempt = time.Time{}
	sm.oauthRetryCount = 0
	sm.isOAuthError = false
	sm.firstAttemptTime = time.Time{}
	sm.connectedAt = time.Time{}
	// NOTE: Do NOT reset consecutiveFailures, autoDisabled, or lastSuccessTime
	// These should persist across disconnections for proper auto-disable logic

	info := sm.buildConnectionInfo()
	callback := sm.onStateChange

	// HIGH-004: Call the callback asynchronously to avoid deadlocks
	// All callback invocations are now standardized to use goroutines
	if callback != nil {
		go callback(oldState, StateDisconnected, &info)
	}
}

// buildConnectionInfo creates a ConnectionInfo from current state (must be called with lock held)
func (sm *StateManager) buildConnectionInfo() ConnectionInfo {
	return ConnectionInfo{
		State:                sm.currentState,
		LastError:            sm.lastError,
		RetryCount:           sm.retryCount,
		LastRetryTime:        sm.lastRetryTime,
		ServerName:           sm.serverName,
		ServerVersion:        sm.serverVersion,
		LastOAuthAttempt:     sm.lastOAuthAttempt,
		OAuthRetryCount:      sm.oauthRetryCount,
		IsOAuthError:         sm.isOAuthError,
		FirstAttemptTime:     sm.firstAttemptTime,
		ConnectedAt:          sm.connectedAt,
		ConsecutiveFailures:  sm.consecutiveFailures,
		AutoDisabled:         sm.autoDisabled,
		AutoDisableReason:    sm.autoDisableReason,
		AutoDisableThreshold: sm.autoDisableThreshold,
		LastSuccessTime:      sm.lastSuccessTime,
	}
}

// ============================================================================
// Runtime-Only UI State Methods (NOT Persisted)
// ============================================================================

// IsUserStopped returns whether the user manually stopped this server via tray UI
// IMPORTANT: This is runtime-only state, never persisted to config or database
// When app restarts, all userStopped flags are cleared
func (sm *StateManager) IsUserStopped() bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.userStopped
}

// SetUserStopped sets whether the user manually stopped this server via tray UI
// IMPORTANT: This is runtime-only state, never persisted to config or database
// When app restarts, all userStopped flags are cleared
func (sm *StateManager) SetUserStopped(stopped bool) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.userStopped = stopped
}

// ============================================================================
// ServerState Management Methods (Persisted Configuration State)
// ============================================================================

// GetServerState returns the current server state (persisted configuration state)
func (sm *StateManager) GetServerState() ServerState {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.serverState
}

// GetServerStateUnsafe returns the server state without acquiring a lock.
// CRIT-003: This should ONLY be called when the caller already holds a lock
// or when atomic access is guaranteed by the caller's lock hierarchy.
func (sm *StateManager) GetServerStateUnsafe() ServerState {
	return sm.serverState
}

// SetServerState sets the server state and publishes an event if event bus is configured
func (sm *StateManager) SetServerState(newState ServerState) {
	sm.mu.Lock()
	oldState := sm.serverState
	sm.serverState = newState
	eventBus := sm.eventBus
	serverName := sm.serverName
	sm.mu.Unlock()

	// Publish event if state changed and event bus is configured
	if oldState != newState && eventBus != nil && serverName != "" {
		event := events.Event{
			Type: events.ServerStateChanged,
			Data: map[string]interface{}{
				"server_name": serverName,
				"old_state":   string(oldState),
				"new_state":   string(newState),
				"timestamp":   time.Now(),
			},
		}
		eventBus.Publish(event)
	}
}

// TransitionServerState validates and transitions to a new server state
// Returns error if transition is invalid
func (sm *StateManager) TransitionServerState(newState ServerState) error {
	// Validate the new state
	if err := ValidateServerState(string(newState)); err != nil {
		return err
	}

	// All transitions are allowed for now (business logic enforced at higher level)
	// Later we can add specific transition validation rules here

	sm.SetServerState(newState)

	return nil
}

// TransitionServerStateUnsafe performs the state transition without acquiring locks.
// CRIT-003: This should ONLY be called when the caller already holds a lock
// and has validated the transition is allowed.
func (sm *StateManager) TransitionServerStateUnsafe(newState ServerState) {
	oldState := sm.serverState
	sm.serverState = newState

	// Capture values for event publishing outside of any lock
	eventBus := sm.eventBus
	serverName := sm.serverName

	// Publish event asynchronously if state changed and event bus is configured
	if oldState != newState && eventBus != nil && serverName != "" {
		go func() {
			event := events.Event{
				Type: events.ServerStateChanged,
				Data: map[string]interface{}{
					"server_name": serverName,
					"old_state":   string(oldState),
					"new_state":   string(newState),
					"timestamp":   time.Now(),
				},
			}
			eventBus.Publish(event)
		}()
	}
}

// SetEventBus configures the event bus for publishing state changes
func (sm *StateManager) SetEventBus(eventBus *events.Bus) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.eventBus = eventBus
}

// ============================================================================
// Enhanced State Change Publishing (ConnectionState + Events)
// ============================================================================

// publishConnectionStateChange publishes connection state change events
func (sm *StateManager) publishConnectionStateChange(oldState, newState ConnectionState, info *ConnectionInfo) {
	sm.mu.RLock()
	eventBus := sm.eventBus
	serverName := sm.serverName
	sm.mu.RUnlock()

	if eventBus != nil && serverName != "" {
		event := events.Event{
			Type: events.ServerStateChanged,
			Data: map[string]interface{}{
				"server_name":     serverName,
				"old_conn_state":  oldState.String(),
				"new_conn_state":  newState.String(),
				"timestamp":       time.Now(),
				"connection_info": info,
			},
		}
		eventBus.Publish(event)
	}
}

// ============================================================================
// Retry & Backoff Logic
// ============================================================================

// ShouldRetry returns true if the connection should be retried based on exponential backoff
func (sm *StateManager) ShouldRetry() bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if sm.currentState != StateError {
		return false
	}

	if sm.retryCount == 0 {
		return true
	}

	// Calculate exponential backoff
	// Ensure retry count is valid and within safe range to avoid overflow
	retryCount := sm.retryCount - 1
	if retryCount < 0 {
		retryCount = 0
	}
	if retryCount > 30 { // Cap at 30 to prevent overflow in 64-bit systems
		retryCount = 30
	}
	backoffDuration := time.Duration(1<<uint(retryCount)) * time.Second //nolint:gosec // retryCount is bounds-checked above
	if backoffDuration > config.MaxBackoffMinutes {
		backoffDuration = config.MaxBackoffMinutes
	}

	return time.Since(sm.lastRetryTime) >= backoffDuration
}

// SetOAuthError sets an OAuth-specific error with longer backoff periods
func (sm *StateManager) SetOAuthError(err error) {
	sm.mu.Lock()

	oldState := sm.currentState
	sm.currentState = StateError
	sm.lastError = err
	sm.isOAuthError = true
	sm.oauthRetryCount++
	sm.lastOAuthAttempt = time.Now()
	sm.lastRetryTime = time.Now()

	info := sm.buildConnectionInfo()
	callback := sm.onStateChange
	sm.mu.Unlock()

	// HIGH-004: Call the callback asynchronously to avoid deadlocks
	// All callback invocations are now standardized to use goroutines
	if callback != nil {
		go callback(oldState, StateError, &info)
	}
}

// ShouldRetryOAuth returns true if OAuth should be retried with much longer backoff intervals
func (sm *StateManager) ShouldRetryOAuth() bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if !sm.isOAuthError || sm.currentState != StateError {
		return false
	}

	if sm.oauthRetryCount == 0 {
		return true
	}

	// OAuth has much longer backoff intervals: 5min, 15min, 1h, 4h, 24h
	var backoffDuration time.Duration
	switch {
	case sm.oauthRetryCount <= 1:
		backoffDuration = config.OAuthBackoffLevel1
	case sm.oauthRetryCount <= 2:
		backoffDuration = config.OAuthBackoffLevel2
	case sm.oauthRetryCount <= 3:
		backoffDuration = config.OAuthBackoffLevel3
	case sm.oauthRetryCount <= 4:
		backoffDuration = config.OAuthBackoffLevel4
	default:
		backoffDuration = config.OAuthBackoffMax // Max backoff for OAuth: 24 hours
	}

	return time.Since(sm.lastOAuthAttempt) >= backoffDuration
}

// IsOAuthError returns true if the last error was OAuth-related
func (sm *StateManager) IsOAuthError() bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.isOAuthError
}

// ============================================================================
// Auto-Disable Logic
// ============================================================================

// ShouldAutoDisable returns true if consecutive failures exceed threshold
// FIX: Added startup grace period check to prevent premature auto-disable
// for slow-starting servers (NPX, Docker)
func (sm *StateManager) ShouldAutoDisable() bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	// Don't auto-disable if already disabled or if threshold is 0 (disabled feature)
	if sm.autoDisabled || sm.autoDisableThreshold <= 0 {
		return false
	}

	// FIX: Check if we're still in the startup grace period
	// This prevents premature auto-disable for servers that take time to start
	if !sm.firstAttemptTime.IsZero() {
		gracePeriodEnd := sm.firstAttemptTime.Add(config.StartupGracePeriod)
		if time.Now().Before(gracePeriodEnd) {
			// Still in grace period - don't auto-disable yet
			// But only if we haven't exceeded double the threshold (obvious failure)
			if sm.consecutiveFailures < sm.autoDisableThreshold*2 {
				return false
			}
		}
	}

	return sm.consecutiveFailures >= sm.autoDisableThreshold
}

// SetAutoDisabled marks the server as auto-disabled with reason
func (sm *StateManager) SetAutoDisabled(reason string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.autoDisabled = true
	sm.autoDisableReason = reason
}

// IsAutoDisabled returns true if the server was auto-disabled
func (sm *StateManager) IsAutoDisabled() bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.autoDisabled
}

// GetAutoDisableReason returns the reason for auto-disable
func (sm *StateManager) GetAutoDisableReason() string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.autoDisableReason
}

// SetAutoDisableThreshold sets the threshold for auto-disable
func (sm *StateManager) SetAutoDisableThreshold(threshold int) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.autoDisableThreshold = threshold
}

// GetConsecutiveFailures returns the current consecutive failures count
func (sm *StateManager) GetConsecutiveFailures() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.consecutiveFailures
}

// ResetAutoDisable clears the auto-disable state (for manual re-enable)
func (sm *StateManager) ResetAutoDisable() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.autoDisabled = false
	sm.autoDisableReason = ""
	sm.consecutiveFailures = 0
}

// ResetConsecutiveFailures resets only the consecutive failure counter
// This is called after a successful connection to ensure clean slate
// FIX: Separate from ResetAutoDisable to allow resetting failures without
// clearing the entire auto-disable state
func (sm *StateManager) ResetConsecutiveFailures() {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.consecutiveFailures = 0
}

// IsInGracePeriod returns true if the server is still in the startup grace period
// During this period, auto-disable is suppressed to allow slow-starting servers time
func (sm *StateManager) IsInGracePeriod() bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if sm.firstAttemptTime.IsZero() {
		return false
	}

	gracePeriodEnd := sm.firstAttemptTime.Add(config.StartupGracePeriod)
	return time.Now().Before(gracePeriodEnd)
}

// GetGracePeriodRemaining returns the remaining time in the startup grace period
func (sm *StateManager) GetGracePeriodRemaining() time.Duration {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if sm.firstAttemptTime.IsZero() {
		return 0
	}

	gracePeriodEnd := sm.firstAttemptTime.Add(config.StartupGracePeriod)
	remaining := gracePeriodEnd.Sub(time.Now())
	if remaining < 0 {
		return 0
	}
	return remaining
}

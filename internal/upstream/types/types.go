package types

import (
	"fmt"
	"sync"
	"time"

	"mcpproxy-go/internal/events"
)

// ServerState represents the persisted configuration state of a server
// This aligns with the startup_mode field in ServerConfig and is stored in the database
//
// State Transitions:
//
// Normal flow (user-initiated):
//   active ←→ disabled        (user enable/disable via UI/API)
//   active ←→ lazy_loading    (user changes startup mode)
//   lazy_loading ←→ disabled  (user enable/disable)
//
// Quarantine flow (security-triggered):
//   [any state] → quarantined (automatic on security detection)
//   quarantined → [original]  (manual approval via UI only)
//
// Auto-disable flow (failure-triggered):
//   active → auto_disabled    (automatic after connection failures)
//   auto_disabled → active    (manual re-enable or group enable)
//   lazy_loading → auto_disabled (automatic after connection failures)
//   auto_disabled → lazy_loading (manual re-enable with lazy mode)
//
// Group operations:
//   Group enable: clears auto_disabled for all servers in group → attempts reconnection
//   Group disable: sets all servers to disabled state
//
// Stability guarantees:
//   - Stable states (active, disabled, lazy_loading): Won't change automatically
//   - Unstable states (quarantined, auto_disabled): Can be cleared or changed
//
type ServerState string

const (
	// StateActive - server starts immediately on boot
	StateActive ServerState = "active"
	// StateDisabledConfig - server is disabled by user configuration
	StateDisabledConfig ServerState = "disabled"
	// StateQuarantined - server is quarantined for security reasons
	StateQuarantined ServerState = "quarantined"
	// StateAutoDisabled - server was automatically disabled after repeated failures
	StateAutoDisabled ServerState = "auto_disabled"
	// StateLazyLoading - server enabled but doesn't start on boot (lazy loaded)
	StateLazyLoading ServerState = "lazy_loading"
)

// String returns the string representation of the server state
func (s ServerState) String() string {
	return string(s)
}

// IsStable returns true if the state represents a stable configuration state
// Stable states don't change automatically - they require user or system intervention
func (s ServerState) IsStable() bool {
	switch s {
	case StateActive, StateDisabledConfig, StateLazyLoading:
		return true
	case StateQuarantined, StateAutoDisabled:
		return false // These can be cleared/changed
	default:
		return false
	}
}

// IsEnabled returns true if the server is enabled (active or lazy loading)
func (s ServerState) IsEnabled() bool {
	return s == StateActive || s == StateLazyLoading
}

// IsDisabled returns true if the server is disabled in any form
func (s ServerState) IsDisabled() bool {
	return s == StateDisabledConfig || s == StateQuarantined || s == StateAutoDisabled
}

// ValidateServerState validates that a server state string is valid
func ValidateServerState(state string) error {
	validStates := map[string]bool{
		string(StateActive):         true,
		string(StateDisabledConfig): true,
		string(StateQuarantined):    true,
		string(StateAutoDisabled):   true,
		string(StateLazyLoading):    true,
	}

	if !validStates[state] {
		return fmt.Errorf("invalid server state: %s (must be one of: active, disabled, quarantined, auto_disabled, lazy_loading)", state)
	}

	return nil
}

// ConnectionState represents the runtime state of an upstream connection (in-memory only)
// This is separate from ServerState which is persisted configuration
type ConnectionState int

const (
	// StateDisconnected indicates the upstream is not connected
	StateDisconnected ConnectionState = iota
	// StateConnecting indicates the upstream is attempting to connect
	StateConnecting
	// StateAuthenticating indicates the upstream is performing OAuth authentication
	StateAuthenticating
	// StateDiscovering indicates the upstream is discovering available tools
	StateDiscovering
	// StateReady indicates the upstream is connected and ready for requests
	StateReady
	// StateError indicates the upstream encountered an error
	StateError
)

// String returns the string representation of the connection state
func (s ConnectionState) String() string {
	switch s {
	case StateDisconnected:
		return "Disconnected"
	case StateConnecting:
		return "Connecting"
	case StateAuthenticating:
		return "Authenticating"
	case StateDiscovering:
		return "Discovering"
	case StateReady:
		return "Ready"
	case StateError:
		return "Error"
	default:
		return "Unknown"
	}
}

// ConnectionInfo holds information about the current connection state
type ConnectionInfo struct {
	State                ConnectionState `json:"state"`
	LastError            error           `json:"last_error,omitempty"`
	RetryCount           int             `json:"retry_count"`
	LastRetryTime        time.Time       `json:"last_retry_time,omitempty"`
	ServerName           string          `json:"server_name,omitempty"`
	ServerVersion        string          `json:"server_version,omitempty"`
	LastOAuthAttempt     time.Time       `json:"last_oauth_attempt,omitempty"`
	OAuthRetryCount      int             `json:"oauth_retry_count"`
	IsOAuthError         bool            `json:"is_oauth_error"`
	FirstAttemptTime     time.Time       `json:"first_attempt_time,omitempty"` // When first connection attempt was made
	ConnectedAt          time.Time       `json:"connected_at,omitempty"`       // When connection was successfully established
	ConsecutiveFailures  int             `json:"consecutive_failures"`         // Number of consecutive failures
	AutoDisabled         bool            `json:"auto_disabled"`                // Whether server was auto-disabled
	AutoDisableReason    string          `json:"auto_disable_reason,omitempty"`// Reason for auto-disable
	AutoDisableThreshold int             `json:"auto_disable_threshold"`       // Threshold for auto-disable (default: 10)
	LastSuccessTime      time.Time       `json:"last_success_time,omitempty"`  // Last successful connection
}

// StateManager manages the state transitions for an upstream connection
type StateManager struct {
	mu                   sync.RWMutex

	// Runtime connection state (in-memory only)
	currentState         ConnectionState
	lastError            error
	retryCount           int
	lastRetryTime        time.Time
	serverName           string
	serverVersion        string
	lastOAuthAttempt     time.Time
	oauthRetryCount      int
	isOAuthError         bool
	firstAttemptTime     time.Time // Time of first connection attempt
	connectedAt          time.Time // Time when connection was established
	consecutiveFailures  int       // Consecutive failures counter
	autoDisabled         bool      // Auto-disable flag
	autoDisableReason    string    // Reason for auto-disable
	autoDisableThreshold int       // Threshold for auto-disable (default: 10)
	lastSuccessTime      time.Time // Last successful connection time

	// Runtime-only UI state (NOT persisted)
	// IMPORTANT: This field should NEVER be saved to config or database
	// When app restarts, all userStopped flags are cleared and servers return to their original startup_mode
	userStopped          bool      // User manually stopped via tray UI (runtime-only, never persisted)

	// Persisted configuration state (stored in database)
	serverState          ServerState // Current server state (active, disabled, quarantined, etc.)

	// Event bus for publishing state changes
	eventBus             *events.Bus

	// Callbacks for state transitions
	onStateChange func(oldState, newState ConnectionState, info *ConnectionInfo)
}

// NewStateManager creates a new state manager
func NewStateManager() *StateManager {
	return &StateManager{
		currentState:         StateDisconnected,
		serverState:          StateActive, // Default to active until config loaded
		autoDisableThreshold: 3,           // Default threshold (user-friendly, overridden by config)
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
func (sm *StateManager) TransitionTo(newState ConnectionState) {
	sm.mu.Lock()
	oldState := sm.currentState

	// Validate transition
	if err := sm.ValidateTransition(oldState, newState); err != nil {
		// For now, log the validation error but allow the transition
		// In the future, we might want to be stricter
		fmt.Printf("Invalid state transition: %v (from %s to %s)\n", err, oldState.String(), newState.String())
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

	info := ConnectionInfo{
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

	callback := sm.onStateChange
	sm.mu.Unlock()

	// Publish event to event bus (non-blocking)
	sm.publishConnectionStateChange(oldState, newState, &info)

	// Call the callback outside the lock to avoid deadlocks
	if callback != nil {
		callback(oldState, newState, &info)
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

	info := ConnectionInfo{
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

	callback := sm.onStateChange
	sm.mu.Unlock()

	// Publish event to event bus (non-blocking)
	sm.publishConnectionStateChange(oldState, StateError, &info)

	// Call the callback outside the lock to avoid deadlocks
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
	maxBackoff := 5 * time.Minute
	if backoffDuration > maxBackoff {
		backoffDuration = maxBackoff
	}

	return time.Since(sm.lastRetryTime) >= backoffDuration
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

	info := ConnectionInfo{
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

	callback := sm.onStateChange

	// Call the callback outside the lock to avoid deadlocks
	if callback != nil {
		go callback(oldState, StateDisconnected, &info)
	}
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

	info := ConnectionInfo{
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

	callback := sm.onStateChange
	sm.mu.Unlock()

	// Call the callback outside the lock to avoid deadlocks
	if callback != nil {
		callback(oldState, StateError, &info)
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
		backoffDuration = 5 * time.Minute
	case sm.oauthRetryCount <= 2:
		backoffDuration = 15 * time.Minute
	case sm.oauthRetryCount <= 3:
		backoffDuration = 1 * time.Hour
	case sm.oauthRetryCount <= 4:
		backoffDuration = 4 * time.Hour
	default:
		backoffDuration = 24 * time.Hour // Max backoff for OAuth: 24 hours
	}

	return time.Since(sm.lastOAuthAttempt) >= backoffDuration
}

// IsOAuthError returns true if the last error was OAuth-related
func (sm *StateManager) IsOAuthError() bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.isOAuthError
}

// ShouldAutoDisable returns true if consecutive failures exceed threshold
func (sm *StateManager) ShouldAutoDisable() bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	// Don't auto-disable if already disabled or if threshold is 0 (disabled feature)
	if sm.autoDisabled || sm.autoDisableThreshold <= 0 {
		return false
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
				"server_name":       serverName,
				"old_conn_state":    oldState.String(),
				"new_conn_state":    newState.String(),
				"timestamp":         time.Now(),
				"connection_info":   info,
			},
		}
		eventBus.Publish(event)
	}
}

package managed

import (
	"fmt"
	"sync"

	"mcpproxy-go/internal/events"
	"mcpproxy-go/internal/storage"
	"mcpproxy-go/internal/upstream/types"
)

// StateMachine manages server state transitions and enforces valid state changes.
// It coordinates between runtime connection states and persisted configuration states,
// handling auto-disable logic and state persistence.
type StateMachine struct {
	mu sync.RWMutex

	// Core dependencies
	stateManager *types.StateManager
	storage      *storage.Manager
	eventBus     *events.Bus
	serverName   string

	// Auto-disable tracking
	consecutiveFailures  int
	autoDisableThreshold int

	// Startup mode flag - when true, auto-disable is suspended
	// This ensures servers are only auto-disabled after ALL waves complete
	startupModeActive bool

	// PersistAutoDisableToConfig controls whether auto-disable state is saved to config file.
	// When false (default), auto-disable state is only stored in database.
	// When true, auto-disable state is written to both database AND config file.
	persistAutoDisableToConfig bool
}

// validTransitions defines allowed state transitions.
// A missing entry means the state can transition to any other state.
// This prevents invalid transitions like quarantined → auto_disabled.
var validTransitions = map[types.ServerState][]types.ServerState{
	// Active can transition to any state
	types.StateActive: {
		types.StateDisabledConfig,
		types.StateQuarantined,
		types.StateAutoDisabled,
		types.StateLazyLoading,
	},

	// Disabled can transition back to active or lazy_loading
	types.StateDisabledConfig: {
		types.StateActive,
		types.StateLazyLoading,
		types.StateQuarantined, // Can quarantine a disabled server
	},

	// Quarantined must be manually approved to active
	types.StateQuarantined: {
		types.StateActive,
		types.StateDisabledConfig, // Can disable quarantined server
	},

	// Auto-disabled can recover to active or be manually disabled
	types.StateAutoDisabled: {
		types.StateActive,        // Manual recovery
		types.StateDisabledConfig, // Convert to permanent disable
	},

	// Lazy loading can transition to active or disabled
	types.StateLazyLoading: {
		types.StateActive,
		types.StateDisabledConfig,
		types.StateQuarantined,    // Can quarantine lazy server
		types.StateAutoDisabled,   // Can auto-disable if failures occur
	},
}

// NewStateMachine creates a new state machine for managing server states.
// persistAutoDisableToConfig controls whether auto-disable state is saved to config file.
// When false (default), auto-disable state is only stored in database.
func NewStateMachine(
	stateManager *types.StateManager,
	storage *storage.Manager,
	eventBus *events.Bus,
	serverName string,
	persistAutoDisableToConfig bool,
) *StateMachine {
	return &StateMachine{
		stateManager:               stateManager,
		storage:                    storage,
		eventBus:                   eventBus,
		serverName:                 serverName,
		autoDisableThreshold:       types.DefaultAutoDisableThreshold, // HIGH-003: Use consolidated constant
		persistAutoDisableToConfig: persistAutoDisableToConfig,
	}
}

// SetAutoDisableThreshold updates the threshold for auto-disable.
func (sm *StateMachine) SetAutoDisableThreshold(threshold int) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.autoDisableThreshold = threshold
}

// SetStartupMode enables or disables startup mode.
// When startup mode is active, auto-disable is suspended to ensure servers
// are only auto-disabled after ALL waves complete in the connection scheduler.
func (sm *StateMachine) SetStartupMode(active bool) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.startupModeActive = active
}

// IsStartupModeActive returns whether startup mode is currently active.
func (sm *StateMachine) IsStartupModeActive() bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.startupModeActive
}

// CanTransitionTo checks if a transition from current state to newState is valid.
// CRIT-004: Changed from permissive to deny-by-default policy for security
func (sm *StateMachine) CanTransitionTo(newState types.ServerState) bool {
	sm.mu.RLock()
	currentState := sm.stateManager.GetServerState()
	sm.mu.RUnlock()

	// Same state transition is always allowed (no-op)
	if currentState == newState {
		return true
	}

	// Check if transition is defined in validTransitions
	allowedStates, ok := validTransitions[currentState]
	if !ok {
		// CRIT-004: Deny by default - only explicitly listed transitions are allowed
		// This prevents unintended state changes when new states are added
		return false
	}

	// Check if newState is in the allowed list
	for _, allowed := range allowedStates {
		if allowed == newState {
			return true
		}
	}

	return false
}

// TransitionTo attempts to transition to a new server state with validation.
// Returns an error if the transition is invalid.
// CRIT-003: Fixed race condition by using unsafe methods that don't acquire nested locks
func (sm *StateMachine) TransitionTo(newState types.ServerState) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// CRIT-003: Use GetServerStateUnsafe to avoid nested lock acquisition
	// The StateMachine lock protects the entire transition operation
	currentState := sm.stateManager.GetServerStateUnsafe()

	// Validate transition
	if !sm.canTransitionToLocked(currentState, newState) {
		return fmt.Errorf("invalid state transition: %s → %s", currentState, newState)
	}

	// CRIT-003: Use TransitionServerStateUnsafe to avoid nested lock acquisition
	// Validation is already done above, so we can safely transition
	sm.stateManager.TransitionServerStateUnsafe(newState)

	// Persist to storage
	if err := sm.persistStateLocked(newState); err != nil {
		// Log error but don't fail the transition
		// The state is already changed in memory
		return fmt.Errorf("state transitioned but failed to persist: %w", err)
	}

	// Reset failure count on successful transition to active
	if newState == types.StateActive {
		sm.consecutiveFailures = 0
	}

	return nil
}

// canTransitionToLocked checks transition validity (must be called with lock held).
// CRIT-004: Changed from permissive to deny-by-default policy for security
func (sm *StateMachine) canTransitionToLocked(currentState, newState types.ServerState) bool {
	// Same state transition is always allowed (no-op)
	if currentState == newState {
		return true
	}

	// Check if transition is defined in validTransitions
	allowedStates, ok := validTransitions[currentState]
	if !ok {
		// CRIT-004: Deny by default - only explicitly listed transitions are allowed
		// This prevents unintended state changes when new states are added
		return false
	}

	// Check if newState is in the allowed list
	for _, allowed := range allowedStates {
		if allowed == newState {
			return true
		}
	}

	return false
}

// HandleConnectionFailure increments failure count and triggers auto-disable if threshold reached.
func (sm *StateMachine) HandleConnectionFailure() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	currentState := sm.stateManager.GetServerState()

	// Only auto-disable if currently active or lazy_loading
	if currentState != types.StateActive && currentState != types.StateLazyLoading {
		return nil
	}

	sm.consecutiveFailures++

	// Check if threshold reached - but only trigger auto-disable if NOT in startup mode
	// During startup, we count failures but defer auto-disable until all waves complete
	if !sm.startupModeActive && sm.consecutiveFailures >= sm.autoDisableThreshold {
		// Transition to auto-disabled
		if err := sm.stateManager.TransitionServerState(types.StateAutoDisabled); err != nil {
			return fmt.Errorf("failed to auto-disable server: %w", err)
		}

		// Persist auto-disable state
		if err := sm.persistAutoDisableLocked(); err != nil {
			return fmt.Errorf("failed to persist auto-disable state: %w", err)
		}

		// Reset failure count
		sm.consecutiveFailures = 0
	}

	return nil
}

// ResetFailures resets the consecutive failure counter.
func (sm *StateMachine) ResetFailures() {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.consecutiveFailures = 0
}

// GetConsecutiveFailures returns the current consecutive failure count.
func (sm *StateMachine) GetConsecutiveFailures() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.consecutiveFailures
}

// persistStateLocked persists the current state to storage (must be called with lock held).
func (sm *StateMachine) persistStateLocked(state types.ServerState) error {
	if sm.storage == nil {
		return nil // No storage configured, skip persistence
	}

	// Map types.ServerState to storage string representation
	var stateStr string
	switch state {
	case types.StateActive:
		stateStr = "active"
	case types.StateDisabledConfig:
		stateStr = "disabled"
	case types.StateQuarantined:
		stateStr = "quarantined"
	case types.StateAutoDisabled:
		stateStr = "auto_disabled"
	case types.StateLazyLoading:
		stateStr = "lazy_loading"
	default:
		stateStr = "active" // Default fallback
	}

	// Persist to database only (not config file) for runtime state changes
	// This preserves the original startup_mode in config while tracking runtime state in DB
	if err := sm.storage.UpdateUpstreamServerState(sm.serverName, stateStr); err != nil {
		return fmt.Errorf("failed to persist state to database: %w", err)
	}

	// Emit state change event for Tray synchronization
	if sm.eventBus != nil {
		sm.eventBus.Publish(events.Event{
			Type:       events.ServerStateChanged,
			ServerName: sm.serverName,
			OldState:   "", // Not tracked here, StateManager handles this
			NewState:   stateStr,
			Data: map[string]interface{}{
				"server_state": stateStr,
				"source":       "state_machine",
			},
		})
	}

	return nil
}

// persistAutoDisableLocked persists auto-disable state to storage (must be called with lock held).
// Depending on persistAutoDisableToConfig flag:
// - When true: uses two-phase commit that updates BOTH database AND config file
// - When false (default): only updates database, keeping servers as "active" in config file
func (sm *StateMachine) persistAutoDisableLocked() error {
	if sm.storage == nil {
		return nil // No storage configured, skip persistence
	}

	if sm.persistAutoDisableToConfig {
		// Use UpdateServerState which handles two-phase commit (DB + config)
		// This ensures both database and config file are updated atomically
		reason := fmt.Sprintf("Auto-disabled after %d consecutive connection failures", sm.autoDisableThreshold)
		if err := sm.storage.UpdateServerState(sm.serverName, reason); err != nil {
			return fmt.Errorf("failed to persist auto-disable state: %w", err)
		}
		// Note: Event emission is handled by storage.UpdateServerState via events.ServerAutoDisabled
	} else {
		// Only update database, keep config file unchanged
		// This allows servers to remain "active" in config while being auto-disabled at runtime
		if err := sm.storage.UpdateUpstreamServerState(sm.serverName, "auto_disabled"); err != nil {
			return fmt.Errorf("failed to persist auto-disable state to database: %w", err)
		}

		// Emit event manually since UpdateUpstreamServerState doesn't emit events
		if sm.eventBus != nil {
			sm.eventBus.Publish(events.Event{
				Type:       events.ServerAutoDisabled,
				ServerName: sm.serverName,
				NewState:   "auto_disabled",
				Data: map[string]interface{}{
					"reason":                "consecutive_failures",
					"threshold":             sm.autoDisableThreshold,
					"persist_to_config":     false,
					"config_state_unchanged": true,
				},
			})
		}
	}

	return nil
}

// EnterLazyLoading transitions the server to lazy_loading state.
// This is used when a server should be available but not actively connected.
func (sm *StateMachine) EnterLazyLoading() error {
	return sm.TransitionTo(types.StateLazyLoading)
}

// ExitLazyLoading transitions the server from lazy_loading back to active.
// This is used when a lazy-loaded server needs to become active.
func (sm *StateMachine) ExitLazyLoading() error {
	sm.mu.Lock()
	currentState := sm.stateManager.GetServerState()
	sm.mu.Unlock()

	if currentState != types.StateLazyLoading {
		return fmt.Errorf("cannot exit lazy loading: current state is %s", currentState)
	}

	return sm.TransitionTo(types.StateActive)
}

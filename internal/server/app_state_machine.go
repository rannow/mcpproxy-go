package server

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"

	"mcpproxy-go/internal/config"
	"mcpproxy-go/internal/events"
	"mcpproxy-go/internal/upstream"
	"mcpproxy-go/internal/upstream/types"
)

// AppStateMachine manages application-level state transitions and enforces valid state changes.
// It coordinates server states to determine overall application health and availability.
type AppStateMachine struct {
	mu sync.RWMutex

	// Core dependencies
	logger          *zap.Logger
	eventBus        *events.Bus
	upstreamManager *upstream.Manager

	// Current state
	currentState AppState

	// Configuration
	stableTimeout time.Duration
}

// validAppTransitions defines allowed application state transitions.
// A missing entry means no specific restrictions (permissive).
var validAppTransitions = map[AppState][]AppState{
	// Starting can transition to running or degraded
	AppStateStarting: {
		AppStateRunning,
		AppStateDegraded,
		AppStateStopping, // Can stop during startup
	},

	// Running can degrade or be stopped
	AppStateRunning: {
		AppStateDegraded,
		AppStateStopping,
	},

	// Degraded can recover to running or be stopped
	AppStateDegraded: {
		AppStateRunning,
		AppStateStopping,
	},

	// Stopping can only go to stopped
	AppStateStopping: {
		AppStateStopped,
	},

	// Stopped can restart
	AppStateStopped: {
		AppStateStarting,
	},
}

// NewAppStateMachine creates a new application state machine.
func NewAppStateMachine(
	logger *zap.Logger,
	eventBus *events.Bus,
	upstreamManager *upstream.Manager,
) *AppStateMachine {
	return &AppStateMachine{
		logger:          logger,
		eventBus:        eventBus,
		upstreamManager: upstreamManager,
		currentState:    AppStateStarting,
		stableTimeout:   config.StableStateTimeout, // Default 30 second timeout
	}
}

// SetStableTimeout updates the timeout for waiting for stable state.
func (asm *AppStateMachine) SetStableTimeout(timeout time.Duration) {
	asm.mu.Lock()
	defer asm.mu.Unlock()
	asm.stableTimeout = timeout
}

// GetState returns the current application state.
func (asm *AppStateMachine) GetState() AppState {
	asm.mu.RLock()
	defer asm.mu.RUnlock()
	return asm.currentState
}

// CanTransitionTo checks if a transition from current state to newState is valid.
func (asm *AppStateMachine) CanTransitionTo(newState AppState) bool {
	asm.mu.RLock()
	currentState := asm.currentState
	asm.mu.RUnlock()

	// Check if transition is defined in validAppTransitions
	allowedStates, ok := validAppTransitions[currentState]
	if !ok {
		// If not defined, allow transition (permissive default)
		return true
	}

	// Check if newState is in the allowed list
	for _, allowed := range allowedStates {
		if allowed == newState {
			return true
		}
	}

	return false
}

// TransitionTo attempts to transition to a new application state with validation.
// Returns an error if the transition is invalid.
func (asm *AppStateMachine) TransitionTo(newState AppState) error {
	asm.mu.Lock()
	defer asm.mu.Unlock()

	oldState := asm.currentState

	// Check if this is a no-op transition
	if oldState == newState {
		return nil
	}

	// Validate transition
	if !asm.canTransitionToLocked(oldState, newState) {
		return fmt.Errorf("invalid app state transition: %s → %s", oldState, newState)
	}

	// Perform the transition
	asm.currentState = newState

	// Log the transition
	asm.logger.Info("Application state changed",
		zap.String("old_state", oldState.String()),
		zap.String("new_state", newState.String()))

	// Publish event
	if asm.eventBus != nil {
		asm.eventBus.Publish(events.Event{
			Type: events.EventAppStateChange,
			Data: events.AppStateChangeData{
				OldState: oldState.String(),
				NewState: newState.String(),
			},
		})
	}

	return nil
}

// canTransitionToLocked checks transition validity (must be called with lock held).
func (asm *AppStateMachine) canTransitionToLocked(currentState, newState AppState) bool {
	// Check if transition is defined in validAppTransitions
	allowedStates, ok := validAppTransitions[currentState]
	if !ok {
		// If not defined, allow transition (permissive default)
		return true
	}

	// Check if newState is in the allowed list
	for _, allowed := range allowedStates {
		if allowed == newState {
			return true
		}
	}

	return false
}

// CheckServerStability evaluates all server states and determines application state.
// Returns the recommended application state based on server health.
func (asm *AppStateMachine) CheckServerStability() AppState {
	clients := asm.upstreamManager.GetAllClients()

	// No servers configured means we're running
	if len(clients) == 0 {
		return AppStateRunning
	}

	enabledCount := 0
	readyCount := 0
	errorCount := 0
	stableCount := 0

	for _, client := range clients {
		if client.Config.StartupMode == "disabled" || client.Config.StartupMode == "quarantined" {
			continue
		}
		enabledCount++

		state := client.GetState()
		serverState := client.StateManager.GetServerState()

		switch state {
		case types.StateReady:
			readyCount++
			if serverState.IsStable() {
				stableCount++
			}
		case types.StateError:
			errorCount++
		}
	}

	// No enabled servers means we're running
	if enabledCount == 0 {
		return AppStateRunning
	}

	// All servers stable and ready
	if stableCount == enabledCount {
		return AppStateRunning
	}

	// Some servers in error or not ready
	if errorCount > 0 || readyCount < enabledCount {
		// If we're still starting, stay in starting
		asm.mu.RLock()
		currentState := asm.currentState
		asm.mu.RUnlock()

		if currentState == AppStateStarting {
			return AppStateStarting
		}
		return AppStateDegraded
	}

	// Default to running
	return AppStateRunning
}

// UpdateState checks server stability and transitions to the appropriate state.
// This should be called when server states change.
func (asm *AppStateMachine) UpdateState() error {
	recommendedState := asm.CheckServerStability()
	return asm.TransitionTo(recommendedState)
}

// WaitForStableState waits for the application to reach a stable state (Running or Stopped).
// Returns an error if the timeout is exceeded.
func (asm *AppStateMachine) WaitForStableState(ctx context.Context) error {
	// Create a context with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, asm.stableTimeout)
	defer cancel()

	// Subscribe to app state change events
	eventChan := asm.eventBus.Subscribe(events.EventAppStateChange)
	defer func() {
		// Unsubscribe when done
		if asm.eventBus != nil {
			asm.eventBus.Unsubscribe(events.EventAppStateChange, eventChan)
		}
	}()

	// Check current state immediately
	currentState := asm.GetState()
	if asm.isStableState(currentState) {
		return nil
	}

	// Wait for stable state or timeout
	for {
		select {
		case <-timeoutCtx.Done():
			return fmt.Errorf("timeout waiting for stable state: current state is %s", asm.GetState())

		case event := <-eventChan:
			// Check if the new state is stable
			if data, ok := event.Data.(events.AppStateChangeData); ok {
				newState := AppState(data.NewState)
				if asm.isStableState(newState) {
					return nil
				}
			}
		}
	}
}

// isStableState returns true if the state is considered stable (terminal).
func (asm *AppStateMachine) isStableState(state AppState) bool {
	return state == AppStateRunning || state == AppStateStopped
}

// StartServers transitions to Starting state and triggers server connections.
// Returns immediately after initiating the startup process.
func (asm *AppStateMachine) StartServers() error {
	// Transition to starting state
	if err := asm.TransitionTo(AppStateStarting); err != nil {
		return fmt.Errorf("failed to transition to starting: %w", err)
	}

	asm.logger.Info("Starting all enabled upstream servers...")

	// Background task to check state after delay
	go func() {
		time.Sleep(config.StateTransitionDelay)
		if err := asm.UpdateState(); err != nil {
			asm.logger.Error("Failed to update app state after startup", zap.Error(err))
		}
	}()

	return nil
}

// StopServers transitions through Stopping to Stopped state.
func (asm *AppStateMachine) StopServers() error {
	// Transition to stopping state
	if err := asm.TransitionTo(AppStateStopping); err != nil {
		return fmt.Errorf("failed to transition to stopping: %w", err)
	}

	asm.logger.Info("Stopping all upstream servers...")

	// Disconnect all servers
	if err := asm.upstreamManager.DisconnectAll(); err != nil {
		asm.logger.Error("Error disconnecting servers", zap.Error(err))
		return err
	}

	// Transition to stopped state
	if err := asm.TransitionTo(AppStateStopped); err != nil {
		return fmt.Errorf("failed to transition to stopped: %w", err)
	}

	asm.logger.Info("All upstream servers stopped")
	return nil
}

package managed

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mcpproxy-go/internal/events"
	"mcpproxy-go/internal/upstream/types"
)

func TestNewStateMachine(t *testing.T) {
	stateManager := types.NewStateManager()
	stateManager.SetServerInfo("test-server", "1.0.0")
	eventBus := events.NewBus()

	sm := NewStateMachine(stateManager, nil, eventBus, "test-server")

	assert.NotNil(t, sm)
	assert.Equal(t, "test-server", sm.serverName)
	assert.Equal(t, types.DefaultAutoDisableThreshold, sm.autoDisableThreshold, "default threshold should match types.DefaultAutoDisableThreshold")
}

func TestStateMachine_SetAutoDisableThreshold(t *testing.T) {
	stateManager := types.NewStateManager()
	sm := NewStateMachine(stateManager, nil, nil, "test")

	sm.SetAutoDisableThreshold(5)
	assert.Equal(t, 5, sm.autoDisableThreshold)
}

func TestStateMachine_CanTransitionTo(t *testing.T) {
	tests := []struct {
		name         string
		currentState types.ServerState
		newState     types.ServerState
		canTransition bool
	}{
		// Active transitions
		{"active → disabled", types.StateActive, types.StateDisabledConfig, true},
		{"active → quarantined", types.StateActive, types.StateQuarantined, true},
		{"active → auto_disabled", types.StateActive, types.StateAutoDisabled, true},
		{"active → lazy_loading", types.StateActive, types.StateLazyLoading, true},

		// Disabled transitions
		{"disabled → active", types.StateDisabledConfig, types.StateActive, true},
		{"disabled → lazy_loading", types.StateDisabledConfig, types.StateLazyLoading, true},
		{"disabled → quarantined", types.StateDisabledConfig, types.StateQuarantined, true},
		{"disabled → auto_disabled", types.StateDisabledConfig, types.StateAutoDisabled, false},

		// Quarantined transitions
		{"quarantined → active", types.StateQuarantined, types.StateActive, true},
		{"quarantined → disabled", types.StateQuarantined, types.StateDisabledConfig, true},
		{"quarantined → auto_disabled", types.StateQuarantined, types.StateAutoDisabled, false},
		{"quarantined → lazy_loading", types.StateQuarantined, types.StateLazyLoading, false},

		// Auto-disabled transitions
		{"auto_disabled → active", types.StateAutoDisabled, types.StateActive, true},
		{"auto_disabled → disabled", types.StateAutoDisabled, types.StateDisabledConfig, true},
		{"auto_disabled → quarantined", types.StateAutoDisabled, types.StateQuarantined, false},
		{"auto_disabled → lazy_loading", types.StateAutoDisabled, types.StateLazyLoading, false},

		// Lazy loading transitions
		{"lazy_loading → active", types.StateLazyLoading, types.StateActive, true},
		{"lazy_loading → disabled", types.StateLazyLoading, types.StateDisabledConfig, true},
		{"lazy_loading → quarantined", types.StateLazyLoading, types.StateQuarantined, true},
		{"lazy_loading → auto_disabled", types.StateLazyLoading, types.StateAutoDisabled, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stateManager := types.NewStateManager()
			stateManager.SetServerState(tt.currentState)

			sm := NewStateMachine(stateManager, nil, nil, "test")
			canTransition := sm.CanTransitionTo(tt.newState)

			assert.Equal(t, tt.canTransition, canTransition,
				"transition %s → %s: expected %v, got %v",
				tt.currentState, tt.newState, tt.canTransition, canTransition)
		})
	}
}

func TestStateMachine_TransitionTo(t *testing.T) {
	stateManager := types.NewStateManager()
	stateManager.SetServerInfo("test-server", "1.0.0")
	eventBus := events.NewBus()

	sm := NewStateMachine(stateManager, nil, eventBus, "test-server")

	// Valid transition
	err := sm.TransitionTo(types.StateDisabledConfig)
	assert.NoError(t, err)
	assert.Equal(t, types.StateDisabledConfig, stateManager.GetServerState())

	// Another valid transition
	err = sm.TransitionTo(types.StateActive)
	assert.NoError(t, err)
	assert.Equal(t, types.StateActive, stateManager.GetServerState())

	// Invalid transition: active → auto_disabled is valid but let's test quarantined → auto_disabled
	stateManager.SetServerState(types.StateQuarantined)
	err = sm.TransitionTo(types.StateAutoDisabled)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid state transition")
	assert.Equal(t, types.StateQuarantined, stateManager.GetServerState(), "state should not change on invalid transition")
}

func TestStateMachine_TransitionTo_ResetsFailuresOnActive(t *testing.T) {
	stateManager := types.NewStateManager()
	stateManager.SetServerInfo("test-server", "1.0.0")

	sm := NewStateMachine(stateManager, nil, nil, "test-server")

	// Transition to disabled first
	err := sm.TransitionTo(types.StateDisabledConfig)
	require.NoError(t, err)

	// Simulate failures
	sm.consecutiveFailures = 2

	// Transition to active should reset failures
	err = sm.TransitionTo(types.StateActive)
	assert.NoError(t, err)
	assert.Equal(t, 0, sm.consecutiveFailures)
}

func TestStateMachine_HandleConnectionFailure(t *testing.T) {
	stateManager := types.NewStateManager()
	stateManager.SetServerInfo("test-server", "1.0.0")
	eventBus := events.NewBus()

	sm := NewStateMachine(stateManager, nil, eventBus, "test-server")
	sm.SetAutoDisableThreshold(3)

	// Initial state is active
	assert.Equal(t, types.StateActive, stateManager.GetServerState())

	// First failure
	err := sm.HandleConnectionFailure()
	assert.NoError(t, err)
	assert.Equal(t, 1, sm.GetConsecutiveFailures())
	assert.Equal(t, types.StateActive, stateManager.GetServerState())

	// Second failure
	err = sm.HandleConnectionFailure()
	assert.NoError(t, err)
	assert.Equal(t, 2, sm.GetConsecutiveFailures())
	assert.Equal(t, types.StateActive, stateManager.GetServerState())

	// Third failure - should trigger auto-disable
	err = sm.HandleConnectionFailure()
	assert.NoError(t, err)
	assert.Equal(t, 0, sm.GetConsecutiveFailures(), "failures should reset after auto-disable")
	assert.Equal(t, types.StateAutoDisabled, stateManager.GetServerState())
}

func TestStateMachine_HandleConnectionFailure_OnlyWhenActiveOrLazy(t *testing.T) {
	stateManager := types.NewStateManager()
	stateManager.SetServerInfo("test-server", "1.0.0")

	sm := NewStateMachine(stateManager, nil, nil, "test-server")
	sm.SetAutoDisableThreshold(1) // Low threshold for quick test

	// Set to disabled state
	stateManager.SetServerState(types.StateDisabledConfig)

	// Failure should be ignored for disabled servers (early return, no count increment)
	err := sm.HandleConnectionFailure()
	assert.NoError(t, err)
	assert.Equal(t, types.StateDisabledConfig, stateManager.GetServerState(), "state should remain disabled")
	assert.Equal(t, 0, sm.GetConsecutiveFailures(), "failure count should not increment for disabled servers")

	// Test that quarantined state also ignores failures
	stateManager.SetServerState(types.StateQuarantined)

	err = sm.HandleConnectionFailure()
	assert.NoError(t, err)
	assert.Equal(t, types.StateQuarantined, stateManager.GetServerState(), "state should remain quarantined")
	assert.Equal(t, 0, sm.GetConsecutiveFailures(), "failure count should not increment for quarantined servers")
}

func TestStateMachine_HandleConnectionFailure_LazyLoading(t *testing.T) {
	stateManager := types.NewStateManager()
	stateManager.SetServerInfo("test-server", "1.0.0")
	eventBus := events.NewBus()

	sm := NewStateMachine(stateManager, nil, eventBus, "test-server")
	sm.SetAutoDisableThreshold(2)

	// Set to lazy_loading state
	stateManager.SetServerState(types.StateLazyLoading)

	// Failures should trigger auto-disable
	err := sm.HandleConnectionFailure()
	assert.NoError(t, err)
	err = sm.HandleConnectionFailure()
	assert.NoError(t, err)

	assert.Equal(t, types.StateAutoDisabled, stateManager.GetServerState())
}

func TestStateMachine_ResetFailures(t *testing.T) {
	stateManager := types.NewStateManager()
	sm := NewStateMachine(stateManager, nil, nil, "test")

	sm.consecutiveFailures = 5
	sm.ResetFailures()

	assert.Equal(t, 0, sm.GetConsecutiveFailures())
}

func TestStateMachine_GetConsecutiveFailures(t *testing.T) {
	stateManager := types.NewStateManager()
	sm := NewStateMachine(stateManager, nil, nil, "test")

	assert.Equal(t, 0, sm.GetConsecutiveFailures())

	sm.consecutiveFailures = 3
	assert.Equal(t, 3, sm.GetConsecutiveFailures())
}

func TestStateMachine_EnterLazyLoading(t *testing.T) {
	stateManager := types.NewStateManager()
	stateManager.SetServerInfo("test-server", "1.0.0")
	eventBus := events.NewBus()

	sm := NewStateMachine(stateManager, nil, eventBus, "test-server")

	// Enter lazy loading from active
	err := sm.EnterLazyLoading()
	assert.NoError(t, err)
	assert.Equal(t, types.StateLazyLoading, stateManager.GetServerState())
}

func TestStateMachine_ExitLazyLoading(t *testing.T) {
	stateManager := types.NewStateManager()
	stateManager.SetServerInfo("test-server", "1.0.0")
	eventBus := events.NewBus()

	sm := NewStateMachine(stateManager, nil, eventBus, "test-server")

	// Set to lazy loading first
	stateManager.SetServerState(types.StateLazyLoading)

	// Exit lazy loading
	err := sm.ExitLazyLoading()
	assert.NoError(t, err)
	assert.Equal(t, types.StateActive, stateManager.GetServerState())
}

func TestStateMachine_ExitLazyLoading_Error(t *testing.T) {
	stateManager := types.NewStateManager()
	stateManager.SetServerInfo("test-server", "1.0.0")

	sm := NewStateMachine(stateManager, nil, nil, "test-server")

	// Try to exit lazy loading when not in that state
	err := sm.ExitLazyLoading()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot exit lazy loading")
	assert.Equal(t, types.StateActive, stateManager.GetServerState())
}

func TestStateMachine_ThreadSafety(t *testing.T) {
	stateManager := types.NewStateManager()
	stateManager.SetServerInfo("test-server", "1.0.0")
	eventBus := events.NewBus()

	sm := NewStateMachine(stateManager, nil, eventBus, "test-server")

	// Concurrent transitions
	done := make(chan bool)
	go func() {
		for i := 0; i < 50; i++ {
			_ = sm.TransitionTo(types.StateDisabledConfig)
			_ = sm.TransitionTo(types.StateActive)
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 50; i++ {
			_ = sm.HandleConnectionFailure()
			sm.ResetFailures()
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 50; i++ {
			_ = sm.GetConsecutiveFailures()
			_ = sm.CanTransitionTo(types.StateLazyLoading)
		}
		done <- true
	}()

	// Wait for all goroutines
	<-done
	<-done
	<-done

	// Verify no race conditions (test passes if no panic)
}

func TestStateMachine_EventBusIntegration(t *testing.T) {
	stateManager := types.NewStateManager()
	stateManager.SetServerInfo("test-server", "1.0.0")
	eventBus := events.NewBus()
	stateManager.SetEventBus(eventBus)

	sm := NewStateMachine(stateManager, nil, eventBus, "test-server")

	// Subscribe to events
	eventChan := eventBus.Subscribe(events.ServerStateChanged)

	// Perform transition
	err := sm.TransitionTo(types.StateDisabledConfig)
	require.NoError(t, err)

	// CRIT-003: Event is now published asynchronously, so we need a timeout
	// instead of default case to allow the goroutine to publish the event
	select {
	case event := <-eventChan:
		assert.Equal(t, events.ServerStateChanged, event.Type)
		data, ok := event.Data.(map[string]interface{})
		require.True(t, ok, "Event data should be a map")
		assert.Equal(t, "test-server", data["server_name"])
		assert.Equal(t, string(types.StateActive), data["old_state"])
		assert.Equal(t, string(types.StateDisabledConfig), data["new_state"])
	case <-time.After(1 * time.Second):
		t.Fatal("expected ServerStateChanged event within timeout")
	}
}

package types

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mcpproxy-go/internal/events"
)

func TestStateManager_ServerState(t *testing.T) {
	sm := NewStateManager()
	sm.SetServerInfo("test-server", "1.0.0")

	// Test initial state
	assert.Equal(t, StateActive, sm.GetServerState(), "initial state should be active")

	// Test SetServerState
	sm.SetServerState(StateDisabledConfig)
	assert.Equal(t, StateDisabledConfig, sm.GetServerState())

	sm.SetServerState(StateQuarantined)
	assert.Equal(t, StateQuarantined, sm.GetServerState())

	sm.SetServerState(StateAutoDisabled)
	assert.Equal(t, StateAutoDisabled, sm.GetServerState())

	sm.SetServerState(StateLazyLoading)
	assert.Equal(t, StateLazyLoading, sm.GetServerState())
}

func TestStateManager_TransitionServerState(t *testing.T) {
	sm := NewStateManager()
	sm.SetServerInfo("test-server", "1.0.0")

	tests := []struct {
		name      string
		newState  ServerState
		wantErr   bool
	}{
		{"valid transition to disabled", StateDisabledConfig, false},
		{"valid transition to quarantined", StateQuarantined, false},
		{"valid transition to auto_disabled", StateAutoDisabled, false},
		{"valid transition to lazy_loading", StateLazyLoading, false},
		{"valid transition to active", StateActive, false},
		{"invalid state", ServerState("invalid"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := sm.TransitionServerState(tt.newState)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.newState, sm.GetServerState())
			}
		})
	}
}

func TestStateManager_SetEventBus(t *testing.T) {
	sm := NewStateManager()
	sm.SetServerInfo("test-server", "1.0.0")

	bus := events.NewBus()
	require.NotNil(t, bus)

	sm.SetEventBus(bus)

	// EventBus should be set internally
	// We can verify by checking events are published (next test)
}

func TestStateManager_ServerStateChangeEvents(t *testing.T) {
	sm := NewStateManager()
	sm.SetServerInfo("test-server", "1.0.0")

	bus := events.NewBus()
	sm.SetEventBus(bus)

	// Subscribe to events
	eventChan := bus.Subscribe(events.ServerStateChanged)

	// Change state and verify event
	sm.SetServerState(StateDisabledConfig)

	select {
	case event := <-eventChan:
		assert.Equal(t, events.ServerStateChanged, event.Type)
		data, ok := event.Data.(map[string]interface{})
		if !ok {
			t.Fatalf("expected Data to be map[string]interface{}, got %T", event.Data)
		}
		assert.Equal(t, "test-server", data["server_name"])
		assert.Equal(t, string(StateActive), data["old_state"])
		assert.Equal(t, string(StateDisabledConfig), data["new_state"])
		assert.NotNil(t, data["timestamp"])
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for event")
	}

	// Change state again
	sm.SetServerState(StateQuarantined)

	select {
	case event := <-eventChan:
		assert.Equal(t, events.ServerStateChanged, event.Type)
		data, ok := event.Data.(map[string]interface{})
		if !ok {
			t.Fatalf("expected Data to be map[string]interface{}, got %T", event.Data)
		}
		assert.Equal(t, "test-server", data["server_name"])
		assert.Equal(t, string(StateDisabledConfig), data["old_state"])
		assert.Equal(t, string(StateQuarantined), data["new_state"])
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for event")
	}
}

func TestStateManager_ConnectionStateChangeEvents(t *testing.T) {
	sm := NewStateManager()
	sm.SetServerInfo("test-server", "1.0.0")

	bus := events.NewBus()
	sm.SetEventBus(bus)

	// Subscribe to events
	eventChan := bus.Subscribe(events.ServerStateChanged)

	// Transition connection state
	sm.TransitionTo(StateConnecting)

	select {
	case event := <-eventChan:
		assert.Equal(t, events.ServerStateChanged, event.Type)
		data, ok := event.Data.(map[string]interface{})
		if !ok {
			t.Fatalf("expected Data to be map[string]interface{}, got %T", event.Data)
		}
		assert.Equal(t, "test-server", data["server_name"])
		assert.Equal(t, "Disconnected", data["old_conn_state"])
		assert.Equal(t, "Connecting", data["new_conn_state"])
		assert.NotNil(t, data["connection_info"])
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for connection state event")
	}

	// Transition to Ready
	sm.TransitionTo(StateReady)

	select {
	case event := <-eventChan:
		data, ok := event.Data.(map[string]interface{})
		if !ok {
			t.Fatalf("expected Data to be map[string]interface{}, got %T", event.Data)
		}
		assert.Equal(t, "Connecting", data["old_conn_state"])
		assert.Equal(t, "Ready", data["new_conn_state"])
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for ready state event")
	}
}

func TestStateManager_NoEventWithoutServerName(t *testing.T) {
	sm := NewStateManager()
	// Note: NOT setting server name

	bus := events.NewBus()
	sm.SetEventBus(bus)

	// Subscribe to events
	eventChan := bus.Subscribe(events.ServerStateChanged)

	// Change state - should NOT publish event
	sm.SetServerState(StateDisabledConfig)

	select {
	case <-eventChan:
		t.Fatal("should not receive event without server name")
	case <-time.After(50 * time.Millisecond):
		// Expected - no event
	}
}

func TestStateManager_NoEventWithoutEventBus(t *testing.T) {
	sm := NewStateManager()
	sm.SetServerInfo("test-server", "1.0.0")
	// Note: NOT setting event bus

	// Change state - should not panic
	sm.SetServerState(StateDisabledConfig)
	assert.Equal(t, StateDisabledConfig, sm.GetServerState())

	// Transition state - should not panic
	sm.TransitionTo(StateConnecting)
	assert.Equal(t, StateConnecting, sm.GetState())
}

func TestStateManager_SameStateNoEvent(t *testing.T) {
	sm := NewStateManager()
	sm.SetServerInfo("test-server", "1.0.0")

	bus := events.NewBus()
	sm.SetEventBus(bus)

	// Subscribe to events
	eventChan := bus.Subscribe(events.ServerStateChanged)

	// Set same state - should NOT publish event
	sm.SetServerState(StateActive)

	select {
	case <-eventChan:
		t.Fatal("should not receive event when state doesn't change")
	case <-time.After(50 * time.Millisecond):
		// Expected - no event for same state
	}
}

func TestStateManager_SetErrorPublishesEvent(t *testing.T) {
	sm := NewStateManager()
	sm.SetServerInfo("test-server", "1.0.0")

	bus := events.NewBus()
	sm.SetEventBus(bus)

	// Subscribe to events
	eventChan := bus.Subscribe(events.ServerStateChanged)

	// Set error
	testErr := assert.AnError
	sm.SetError(testErr)

	select {
	case event := <-eventChan:
		assert.Equal(t, events.ServerStateChanged, event.Type)
		data, ok := event.Data.(map[string]interface{})
		if !ok {
			t.Fatalf("expected Data to be map[string]interface{}, got %T", event.Data)
		}
		assert.Equal(t, "test-server", data["server_name"])
		assert.Equal(t, "Disconnected", data["old_conn_state"])
		assert.Equal(t, "Error", data["new_conn_state"])
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for error state event")
	}
}

func TestStateManager_ThreadSafety(t *testing.T) {
	sm := NewStateManager()
	sm.SetServerInfo("test-server", "1.0.0")

	bus := events.NewBus()
	sm.SetEventBus(bus)

	// Subscribe to events
	eventChan := bus.Subscribe(events.ServerStateChanged)

	// Concurrent state changes
	done := make(chan bool)
	go func() {
		for i := 0; i < 50; i++ {
			sm.SetServerState(StateActive)
			sm.SetServerState(StateDisabledConfig)
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 50; i++ {
			sm.TransitionTo(StateConnecting)
			sm.TransitionTo(StateReady)
			sm.TransitionTo(StateDisconnected)
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 50; i++ {
			_ = sm.GetServerState()
			_ = sm.GetState()
		}
		done <- true
	}()

	// Drain events to prevent channel blocking
	go func() {
		for range eventChan {
			// Just drain the events
		}
	}()

	// Wait for all goroutines
	<-done
	<-done
	<-done

	// Verify no race conditions (test passes if no panic)
}

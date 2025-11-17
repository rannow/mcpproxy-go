package server

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"mcpproxy-go/internal/config"
	"mcpproxy-go/internal/events"
	"mcpproxy-go/internal/upstream"
)

func TestNewAppStateMachine(t *testing.T) {
	logger := zap.NewNop()
	eventBus := events.NewBus()
	upstreamManager := upstream.NewManager(logger, nil, nil)

	asm := NewAppStateMachine(logger, eventBus, upstreamManager)

	assert.NotNil(t, asm)
	assert.Equal(t, AppStateStarting, asm.GetState(), "initial state should be starting")
	assert.Equal(t, 30*time.Second, asm.stableTimeout, "default timeout should be 30 seconds")
}

func TestAppStateMachine_SetStableTimeout(t *testing.T) {
	logger := zap.NewNop()
	eventBus := events.NewBus()
	upstreamManager := upstream.NewManager(logger, nil, nil)

	asm := NewAppStateMachine(logger, eventBus, upstreamManager)

	newTimeout := 10 * time.Second
	asm.SetStableTimeout(newTimeout)

	assert.Equal(t, newTimeout, asm.stableTimeout)
}

func TestAppStateMachine_CanTransitionTo(t *testing.T) {
	tests := []struct {
		name          string
		currentState  AppState
		newState      AppState
		canTransition bool
	}{
		// Starting transitions
		{"starting → running", AppStateStarting, AppStateRunning, true},
		{"starting → degraded", AppStateStarting, AppStateDegraded, true},
		{"starting → stopping", AppStateStarting, AppStateStopping, true},
		{"starting → stopped", AppStateStarting, AppStateStopped, false},

		// Running transitions
		{"running → degraded", AppStateRunning, AppStateDegraded, true},
		{"running → stopping", AppStateRunning, AppStateStopping, true},
		{"running → starting", AppStateRunning, AppStateStarting, false},
		{"running → stopped", AppStateRunning, AppStateStopped, false},

		// Degraded transitions
		{"degraded → running", AppStateDegraded, AppStateRunning, true},
		{"degraded → stopping", AppStateDegraded, AppStateStopping, true},
		{"degraded → starting", AppStateDegraded, AppStateStarting, false},
		{"degraded → stopped", AppStateDegraded, AppStateStopped, false},

		// Stopping transitions
		{"stopping → stopped", AppStateStopping, AppStateStopped, true},
		{"stopping → running", AppStateStopping, AppStateRunning, false},
		{"stopping → starting", AppStateStopping, AppStateStarting, false},
		{"stopping → degraded", AppStateStopping, AppStateDegraded, false},

		// Stopped transitions
		{"stopped → starting", AppStateStopped, AppStateStarting, true},
		{"stopped → running", AppStateStopped, AppStateRunning, false},
		{"stopped → degraded", AppStateStopped, AppStateDegraded, false},
		{"stopped → stopping", AppStateStopped, AppStateStopping, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := zap.NewNop()
			eventBus := events.NewBus()
			upstreamManager := upstream.NewManager(logger, nil, nil)

			asm := NewAppStateMachine(logger, eventBus, upstreamManager)
			asm.currentState = tt.currentState

			canTransition := asm.CanTransitionTo(tt.newState)

			assert.Equal(t, tt.canTransition, canTransition,
				"transition %s → %s: expected %v, got %v",
				tt.currentState, tt.newState, tt.canTransition, canTransition)
		})
	}
}

func TestAppStateMachine_TransitionTo(t *testing.T) {
	logger := zap.NewNop()
	eventBus := events.NewBus()
	upstreamManager := upstream.NewManager(logger, nil, nil)

	asm := NewAppStateMachine(logger, eventBus, upstreamManager)

	// Valid transition
	err := asm.TransitionTo(AppStateRunning)
	assert.NoError(t, err)
	assert.Equal(t, AppStateRunning, asm.GetState())

	// Another valid transition
	err = asm.TransitionTo(AppStateDegraded)
	assert.NoError(t, err)
	assert.Equal(t, AppStateDegraded, asm.GetState())

	// Invalid transition: degraded → stopped
	err = asm.TransitionTo(AppStateStopped)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid app state transition")
	assert.Equal(t, AppStateDegraded, asm.GetState(), "state should not change on invalid transition")
}

func TestAppStateMachine_TransitionTo_NoOp(t *testing.T) {
	logger := zap.NewNop()
	eventBus := events.NewBus()
	upstreamManager := upstream.NewManager(logger, nil, nil)

	asm := NewAppStateMachine(logger, eventBus, upstreamManager)

	// Subscribe to events
	eventChan := eventBus.Subscribe(events.EventAppStateChange)

	// Drain any initial events
	select {
	case <-eventChan:
	case <-time.After(50 * time.Millisecond):
	}

	// Transition to same state (no-op)
	err := asm.TransitionTo(AppStateStarting)
	assert.NoError(t, err)

	// Verify no event was published
	select {
	case event := <-eventChan:
		t.Fatalf("unexpected event: %+v", event)
	case <-time.After(100 * time.Millisecond):
		// Expected - no event
	}
}

func TestAppStateMachine_TransitionTo_PublishesEvent(t *testing.T) {
	logger := zap.NewNop()
	eventBus := events.NewBus()
	upstreamManager := upstream.NewManager(logger, nil, nil)

	asm := NewAppStateMachine(logger, eventBus, upstreamManager)

	// Subscribe to events
	eventChan := eventBus.Subscribe(events.EventAppStateChange)

	// Perform transition
	err := asm.TransitionTo(AppStateRunning)
	require.NoError(t, err)

	// Verify event was published
	select {
	case event := <-eventChan:
		assert.Equal(t, events.EventAppStateChange, event.Type)
		data, ok := event.Data.(events.AppStateChangeData)
		require.True(t, ok, "event data should be AppStateChangeData")
		assert.Equal(t, "starting", data.OldState)
		assert.Equal(t, "running", data.NewState)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("expected EventAppStateChange event")
	}
}

func TestAppStateMachine_ThreadSafety(t *testing.T) {
	logger := zap.NewNop()
	eventBus := events.NewBus()
	upstreamManager := upstream.NewManager(logger, nil, nil)

	asm := NewAppStateMachine(logger, eventBus, upstreamManager)

	// Concurrent state changes
	var wg sync.WaitGroup
	states := []AppState{AppStateRunning, AppStateDegraded, AppStateStopping, AppStateStopped}

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			state := states[idx%len(states)]
			// Ignore errors since not all transitions are valid
			_ = asm.TransitionTo(state)
		}(i)
	}

	// Concurrent reads
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = asm.GetState()
			_ = asm.CanTransitionTo(AppStateRunning)
		}()
	}

	// Wait for all goroutines with timeout
	done := make(chan bool)
	go func() {
		wg.Wait()
		done <- true
	}()

	select {
	case <-done:
		// Success - no race conditions
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for concurrent operations")
	}
}

func TestAppStateMachine_CheckServerStability_NoServers(t *testing.T) {
	logger := zap.NewNop()
	eventBus := events.NewBus()
	upstreamManager := upstream.NewManager(logger, nil, nil)

	asm := NewAppStateMachine(logger, eventBus, upstreamManager)

	// With no servers, app should be running
	state := asm.CheckServerStability()
	assert.Equal(t, AppStateRunning, state)
}

func TestAppStateMachine_CheckServerStability_AllDisabled(t *testing.T) {
	logger := zap.NewNop()
	cfg := &config.Config{
		Listen:  ":8080",
		DataDir: t.TempDir(),
		Servers: []*config.ServerConfig{
			{
				Name:    "server1",
				StartupMode: "disabled",
			},
			{
				Name:    "server2",
				StartupMode: "disabled",
			},
		},
	}

	eventBus := events.NewBus()
	upstreamManager := upstream.NewManager(logger, cfg, nil)

	asm := NewAppStateMachine(logger, eventBus, upstreamManager)

	// With all servers disabled, app should be running
	state := asm.CheckServerStability()
	assert.Equal(t, AppStateRunning, state)
}

func TestAppStateMachine_UpdateState(t *testing.T) {
	logger := zap.NewNop()
	eventBus := events.NewBus()
	upstreamManager := upstream.NewManager(logger, nil, nil)

	asm := NewAppStateMachine(logger, eventBus, upstreamManager)

	// Update state with no servers should transition to running
	err := asm.UpdateState()
	assert.NoError(t, err)
	assert.Equal(t, AppStateRunning, asm.GetState())
}

func TestAppStateMachine_IsStableState(t *testing.T) {
	logger := zap.NewNop()
	eventBus := events.NewBus()
	upstreamManager := upstream.NewManager(logger, nil, nil)

	asm := NewAppStateMachine(logger, eventBus, upstreamManager)

	tests := []struct {
		state    AppState
		isStable bool
	}{
		{AppStateStarting, false},
		{AppStateRunning, true},
		{AppStateDegraded, false},
		{AppStateStopping, false},
		{AppStateStopped, true},
	}

	for _, tt := range tests {
		t.Run(tt.state.String(), func(t *testing.T) {
			assert.Equal(t, tt.isStable, asm.isStableState(tt.state))
		})
	}
}

func TestAppStateMachine_WaitForStableState_AlreadyStable(t *testing.T) {
	logger := zap.NewNop()
	eventBus := events.NewBus()
	upstreamManager := upstream.NewManager(logger, nil, nil)

	asm := NewAppStateMachine(logger, eventBus, upstreamManager)

	// Set to running (stable)
	err := asm.TransitionTo(AppStateRunning)
	require.NoError(t, err)

	// Should return immediately
	ctx := context.Background()
	err = asm.WaitForStableState(ctx)
	assert.NoError(t, err)
}

func TestAppStateMachine_WaitForStableState_Timeout(t *testing.T) {
	logger := zap.NewNop()
	eventBus := events.NewBus()
	upstreamManager := upstream.NewManager(logger, nil, nil)

	asm := NewAppStateMachine(logger, eventBus, upstreamManager)
	asm.SetStableTimeout(100 * time.Millisecond)

	// Transition to degraded (not stable)
	err := asm.TransitionTo(AppStateRunning)
	require.NoError(t, err)
	err = asm.TransitionTo(AppStateDegraded)
	require.NoError(t, err)

	// Should timeout
	ctx := context.Background()
	err = asm.WaitForStableState(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "timeout waiting for stable state")
}

func TestAppStateMachine_WaitForStableState_BecomesStable(t *testing.T) {
	logger := zap.NewNop()
	eventBus := events.NewBus()
	upstreamManager := upstream.NewManager(logger, nil, nil)

	asm := NewAppStateMachine(logger, eventBus, upstreamManager)
	asm.SetStableTimeout(2 * time.Second)

	// Start in degraded (not stable)
	err := asm.TransitionTo(AppStateRunning)
	require.NoError(t, err)
	err = asm.TransitionTo(AppStateDegraded)
	require.NoError(t, err)

	// Background transition to stable state
	go func() {
		time.Sleep(200 * time.Millisecond)
		_ = asm.TransitionTo(AppStateRunning)
	}()

	// Should wait and then return when stable
	ctx := context.Background()
	err = asm.WaitForStableState(ctx)
	assert.NoError(t, err)
	assert.Equal(t, AppStateRunning, asm.GetState())
}

func TestAppStateMachine_StartServers(t *testing.T) {
	logger := zap.NewNop()
	eventBus := events.NewBus()
	upstreamManager := upstream.NewManager(logger, nil, nil)

	asm := NewAppStateMachine(logger, eventBus, upstreamManager)

	// Set to stopped first
	asm.currentState = AppStateStopped

	// Subscribe to events
	eventChan := eventBus.Subscribe(events.EventAppStateChange)

	// Start servers
	err := asm.StartServers()
	assert.NoError(t, err)

	// Should transition to starting immediately
	assert.Equal(t, AppStateStarting, asm.GetState())

	// Verify event
	select {
	case event := <-eventChan:
		data, ok := event.Data.(events.AppStateChangeData)
		require.True(t, ok)
		assert.Equal(t, "stopped", data.OldState)
		assert.Equal(t, "starting", data.NewState)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("expected EventAppStateChange event")
	}

	// Wait for background task to update state (2 second delay + buffer)
	time.Sleep(2500 * time.Millisecond)

	// Should eventually transition to running (no servers configured)
	finalState := asm.GetState()
	assert.Equal(t, AppStateRunning, finalState)
}

func TestAppStateMachine_StopServers(t *testing.T) {
	logger := zap.NewNop()
	eventBus := events.NewBus()
	upstreamManager := upstream.NewManager(logger, nil, nil)

	asm := NewAppStateMachine(logger, eventBus, upstreamManager)

	// Set to running first
	err := asm.TransitionTo(AppStateRunning)
	require.NoError(t, err)

	// Subscribe to events
	eventChan := eventBus.Subscribe(events.EventAppStateChange)

	// Drain initial event
	select {
	case <-eventChan:
	case <-time.After(50 * time.Millisecond):
	}

	// Stop servers
	err = asm.StopServers()
	assert.NoError(t, err)

	// Final state should be stopped
	assert.Equal(t, AppStateStopped, asm.GetState())

	// Verify events were published (Stopping → Stopped)
	receivedEvents := []events.Event{}
	timeout := time.After(200 * time.Millisecond)
eventLoop:
	for {
		select {
		case event := <-eventChan:
			receivedEvents = append(receivedEvents, event)
			if len(receivedEvents) >= 2 {
				break eventLoop
			}
		case <-timeout:
			break eventLoop
		}
	}

	// Should have received at least stopping and stopped events
	require.GreaterOrEqual(t, len(receivedEvents), 2)

	// Find stopping event
	var stoppingEvent *events.Event
	for i := range receivedEvents {
		data, ok := receivedEvents[i].Data.(events.AppStateChangeData)
		if ok && data.NewState == "stopping" {
			stoppingEvent = &receivedEvents[i]
			break
		}
	}
	require.NotNil(t, stoppingEvent, "should have received stopping event")

	// Find stopped event
	var stoppedEvent *events.Event
	for i := range receivedEvents {
		data, ok := receivedEvents[i].Data.(events.AppStateChangeData)
		if ok && data.NewState == "stopped" {
			stoppedEvent = &receivedEvents[i]
			break
		}
	}
	require.NotNil(t, stoppedEvent, "should have received stopped event")
}

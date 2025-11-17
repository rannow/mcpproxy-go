package server

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"mcpproxy-go/internal/config"
	"mcpproxy-go/internal/events"
	"mcpproxy-go/internal/upstream/types"
)

func TestAppState_String(t *testing.T) {
	tests := []struct {
		name  string
		state AppState
		want  string
	}{
		{"starting", AppStateStarting, "starting"},
		{"running", AppStateRunning, "running"},
		{"degraded", AppStateDegraded, "degraded"},
		{"stopping", AppStateStopping, "stopping"},
		{"stopped", AppStateStopped, "stopped"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.state.String())
		})
	}
}

func TestServer_GetAppState(t *testing.T) {
	logger := zap.NewNop()
	cfg := &config.Config{
		Listen:  ":8080",
		DataDir: t.TempDir(),
		Servers: []*config.ServerConfig{},
	}

	server, err := NewServerWithConfigPath(cfg, "", logger)
	require.NoError(t, err)
	defer server.Shutdown()

	// Initial state should be starting
	state := server.GetAppState()
	assert.Equal(t, AppStateStarting, state)
}

func TestServer_SetAppState(t *testing.T) {
	logger := zap.NewNop()
	cfg := &config.Config{
		Listen:  ":8080",
		DataDir: t.TempDir(),
		Servers: []*config.ServerConfig{},
	}

	server, err := NewServerWithConfigPath(cfg, "", logger)
	require.NoError(t, err)
	defer server.Shutdown()

	// Subscribe to app state change events
	eventChan := server.eventBus.Subscribe(events.EventAppStateChange)

	// Set new state
	server.setAppState(AppStateRunning)

	// Verify state was set
	assert.Equal(t, AppStateRunning, server.GetAppState())

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

func TestServer_SetAppState_NoEventOnSameState(t *testing.T) {
	logger := zap.NewNop()
	cfg := &config.Config{
		Listen:  ":8080",
		DataDir: t.TempDir(),
		Servers: []*config.ServerConfig{},
	}

	server, err := NewServerWithConfigPath(cfg, "", logger)
	require.NoError(t, err)
	defer server.Shutdown()

	// Subscribe to app state change events
	eventChan := server.eventBus.Subscribe(events.EventAppStateChange)

	// Drain any initial events
	select {
	case <-eventChan:
	case <-time.After(50 * time.Millisecond):
	}

	// Set state to same value
	server.setAppState(AppStateStarting)

	// Verify no event was published
	select {
	case event := <-eventChan:
		t.Fatalf("unexpected event: %+v", event)
	case <-time.After(100 * time.Millisecond):
		// Expected - no event
	}
}

func TestServer_SetAppState_ThreadSafe(t *testing.T) {
	logger := zap.NewNop()
	cfg := &config.Config{
		Listen:  ":8080",
		DataDir: t.TempDir(),
		Servers: []*config.ServerConfig{},
	}

	server, err := NewServerWithConfigPath(cfg, "", logger)
	require.NoError(t, err)
	defer server.Shutdown()

	// Concurrent state changes
	var wg sync.WaitGroup
	states := []AppState{AppStateRunning, AppStateDegraded, AppStateStopping, AppStateStopped}

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			state := states[idx%len(states)]
			server.setAppState(state)
		}(i)
	}

	// Concurrent reads
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = server.GetAppState()
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

func TestServer_CheckAndUpdateAppState_NoServers(t *testing.T) {
	logger := zap.NewNop()
	cfg := &config.Config{
		Listen:  ":8080",
		DataDir: t.TempDir(),
		Servers: []*config.ServerConfig{},
	}

	server, err := NewServerWithConfigPath(cfg, "", logger)
	require.NoError(t, err)
	defer server.Shutdown()

	// With no servers, app should be running
	server.checkAndUpdateAppState()
	assert.Equal(t, AppStateRunning, server.GetAppState())
}

func TestServer_CheckAndUpdateAppState_AllDisabled(t *testing.T) {
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

	server, err := NewServerWithConfigPath(cfg, "", logger)
	require.NoError(t, err)
	defer server.Shutdown()

	// With all servers disabled, app should be running
	server.checkAndUpdateAppState()
	assert.Equal(t, AppStateRunning, server.GetAppState())
}

func TestServer_StopAllServers(t *testing.T) {
	logger := zap.NewNop()
	cfg := &config.Config{
		Listen:  ":8080",
		DataDir: t.TempDir(),
		Servers: []*config.ServerConfig{},
	}

	server, err := NewServerWithConfigPath(cfg, "", logger)
	require.NoError(t, err)
	defer server.Shutdown()

	// Subscribe to app state change events
	eventChan := server.eventBus.Subscribe(events.EventAppStateChange)

	// Drain initial events
	select {
	case <-eventChan:
	case <-time.After(50 * time.Millisecond):
	}

	// Stop all servers
	err = server.StopAllServers()
	assert.NoError(t, err)

	// Verify final state is stopped
	assert.Equal(t, AppStateStopped, server.GetAppState())

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

func TestServer_StartAllServers(t *testing.T) {
	logger := zap.NewNop()
	cfg := &config.Config{
		Listen:  ":8080",
		DataDir: t.TempDir(),
		Servers: []*config.ServerConfig{},
	}

	server, err := NewServerWithConfigPath(cfg, "", logger)
	require.NoError(t, err)
	defer server.Shutdown()

	// First transition to a different state so we can see the transition to starting
	server.setAppState(AppStateRunning)

	// Subscribe to app state change events
	eventChan := server.eventBus.Subscribe(events.EventAppStateChange)

	// Start all servers
	err = server.StartAllServers()
	assert.NoError(t, err)

	// Should transition to starting state immediately
	select {
	case event := <-eventChan:
		data, ok := event.Data.(events.AppStateChangeData)
		require.True(t, ok, "event data should be AppStateChangeData")
		assert.Equal(t, "starting", data.NewState)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("expected EventAppStateChange event for starting state")
	}

	// Wait for goroutine to update state (2 second delay in implementation)
	time.Sleep(2500 * time.Millisecond)

	// Should eventually transition to running (no servers configured)
	state := server.GetAppState()
	assert.Equal(t, AppStateRunning, state)
}

// mockClient is a test helper for creating test clients
type mockClient struct {
	config       *config.ServerConfig
	state        types.ConnectionState
	stateManager *types.StateManager
}

func (mc *mockClient) GetState() types.ConnectionState {
	return mc.state
}

func TestServer_CheckAndUpdateAppState_AllReady(t *testing.T) {
	// This test would require mocking the upstream manager
	// to return test clients - skipping for now as it requires
	// significant refactoring to make upstream manager injectable
	t.Skip("Requires upstream manager mocking - to be implemented")
}

func TestServer_CheckAndUpdateAppState_SomeErrors(t *testing.T) {
	// This test would require mocking the upstream manager
	// Skipping for now
	t.Skip("Requires upstream manager mocking - to be implemented")
}

func TestServer_CheckAndUpdateAppState_Degraded(t *testing.T) {
	// This test would require mocking the upstream manager
	// Skipping for now
	t.Skip("Requires upstream manager mocking - to be implemented")
}

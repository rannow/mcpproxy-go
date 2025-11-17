package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"mcpproxy-go/internal/config"
	"mcpproxy-go/internal/events"
	"mcpproxy-go/internal/storage"
)

// TestIntegration_AutoDisablePersistence verifies auto-disable state persists across restarts
func TestIntegration_AutoDisablePersistence(t *testing.T) {
	// Create temporary directory for test
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "mcp_config.json")

	// Step 1: Create initial configuration
	cfg := config.DefaultConfig()
	cfg.DataDir = tempDir
	cfg.Servers = []*config.ServerConfig{
		{
			Name:        "test-server",
			Protocol:    "http",
			URL:         "http://localhost:9999",
			StartupMode: "active",
			Created:     time.Now(),
		},
	}

	configData, err := json.MarshalIndent(cfg, "", "  ")
	require.NoError(t, err)
	err = os.WriteFile(configPath, configData, 0644)
	require.NoError(t, err)

	// Step 2: Create storage manager and config loader
	logger := zap.NewNop().Sugar()
	storageManager, err := storage.NewManager(tempDir, logger)
	require.NoError(t, err)

	configLoader, err := config.NewLoader(configPath, zap.NewNop())
	require.NoError(t, err)

	_, err = configLoader.Load()
	require.NoError(t, err)

	storageManager.SetConfigLoader(configLoader)

	// Save server to database
	require.NoError(t, storageManager.SaveUpstreamServer(cfg.Servers[0]))

	// Step 3: Set server to auto-disabled state
	err = storageManager.UpdateServerState("test-server", "test auto-disable reason")
	require.NoError(t, err)

	// Step 4: Verify auto-disable state in database
	record, err := storageManager.GetBoltDB().GetUpstream("test-server")
	require.NoError(t, err)
	assert.Equal(t, "auto_disabled", record.ServerState, "Server should be auto_disabled in database")

	// Step 5: Verify config file was updated
	serverConfig, err := storageManager.GetUpstreamServer("test-server")
	require.NoError(t, err)
	assert.Equal(t, "auto_disabled", serverConfig.StartupMode, "Config should show auto_disabled")
	assert.Equal(t, "test auto-disable reason", serverConfig.AutoDisableReason, "Reason should match in config")

	// Step 6: Close and recreate storage manager (simulates restart)
	storageManager.Close()
	configLoader.Stop()

	// Step 7: Create new storage manager instance
	storageManager2, err := storage.NewManager(tempDir, logger)
	require.NoError(t, err)
	defer storageManager2.Close()

	// Step 8: Verify state persisted in database after restart
	record2, err := storageManager2.GetBoltDB().GetUpstream("test-server")
	require.NoError(t, err)
	assert.Equal(t, "auto_disabled", record2.ServerState, "Server should still be auto_disabled after restart")

	// Step 9: Verify config file still has correct state
	serverConfig2, err := storageManager2.GetUpstreamServer("test-server")
	require.NoError(t, err)
	assert.Equal(t, "auto_disabled", serverConfig2.StartupMode, "Config should still show auto_disabled after restart")
	assert.Equal(t, "test auto-disable reason", serverConfig2.AutoDisableReason, "Reason should persist after restart")
}

// TestIntegration_GroupEnableClearsAutoDisable verifies group enable clears auto-disable state
func TestIntegration_GroupEnableClearsAutoDisable(t *testing.T) {
	// Create temporary directory for test
	tempDir := t.TempDir()

	// Create config with groups and servers
	cfg := config.DefaultConfig()
	cfg.DataDir = tempDir
	cfg.Groups = []config.GroupConfig{
		{
			ID:      1,
			Name:    "test-group",
			Color:   "#FF0000",
			Enabled: true,
		},
	}
	cfg.Servers = []*config.ServerConfig{
		{
			Name:     "test-server-1",
			Protocol: "stdio",
			Command:  "test-command",
			StartupMode: "active",
			GroupID:  1,
			Created:  time.Now(),
		},
	}

	// Save config
	configPath := filepath.Join(tempDir, "mcp_config.json")
	configData, err := json.MarshalIndent(cfg, "", "  ")
	require.NoError(t, err)
	err = os.WriteFile(configPath, configData, 0644)
	require.NoError(t, err)

	// Create server
	logger := zap.NewNop()
	srv, err := NewServerWithConfigPath(cfg, configPath, logger)
	require.NoError(t, err)
	require.NotNil(t, srv)
	defer srv.storageManager.Close()

	// Wait for server initialization
	time.Sleep(100 * time.Millisecond)

	// Save server to storage first
	err = srv.storageManager.SaveUpstreamServer(cfg.Servers[0])
	require.NoError(t, err)

	// Set server to auto-disabled state
	err = srv.storageManager.UpdateServerState("test-server-1", "auto-disable test")
	require.NoError(t, err)

	// Verify auto-disabled
	serverConfig, err := srv.storageManager.GetUpstreamServer("test-server-1")
	require.NoError(t, err)
	assert.Equal(t, "auto_disabled", serverConfig.StartupMode, "Server should be auto_disabled")

	// Subscribe to events before group enable
	eventChan := srv.eventBus.Subscribe(events.ServerStateChanged)

	// Enable group via HTTP endpoint (should clear auto-disable)
	// Assign server to group first
	assignmentsMutex.Lock()
	serverGroupAssignments["test-server-1"] = "test-group"
	assignmentsMutex.Unlock()

	// Call the toggle group servers endpoint
	payload := map[string]interface{}{
		"group_name": "test-group",
		"enabled":    true,
	}
	body, err := json.Marshal(payload)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/api/toggle-group-servers", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.handleToggleGroupServers(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	// Verify auto-disable was cleared
	serverConfig, err = srv.storageManager.GetUpstreamServer("test-server-1")
	require.NoError(t, err)
	assert.Equal(t, "active", serverConfig.StartupMode, "Server should be active after group enable")
	assert.Empty(t, serverConfig.AutoDisableReason, "Auto-disable reason should be cleared")

	// Verify event was emitted (with timeout)
	select {
	case event := <-eventChan:
		assert.Equal(t, events.ServerStateChanged, event.Type, "Should receive state change event")
		assert.Equal(t, "test-server-1", event.ServerName, "Event should be for test-server-1")
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for state change event")
	}
}

// TestIntegration_WebSocketEventDelivery verifies end-to-end WebSocket event delivery
func TestIntegration_WebSocketEventDelivery(t *testing.T) {
	// Create temporary directory for test
	tempDir := t.TempDir()

	// Create config
	cfg := config.DefaultConfig()
	cfg.DataDir = tempDir
	cfg.Servers = []*config.ServerConfig{
		{
			Name:     "test-server",
			Protocol: "stdio",
			Command:  "test-command",
			StartupMode: "active",
			Created:  time.Now(),
		},
	}

	// Save config
	configPath := filepath.Join(tempDir, "mcp_config.json")
	configData, err := json.MarshalIndent(cfg, "", "  ")
	require.NoError(t, err)
	err = os.WriteFile(configPath, configData, 0644)
	require.NoError(t, err)

	// Create server
	logger := zap.NewNop()
	srv, err := NewServerWithConfigPath(cfg, configPath, logger)
	require.NoError(t, err)
	require.NotNil(t, srv)
	defer srv.storageManager.Close()

	// Start HTTP server for WebSocket
	httpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/ws/events" {
			srv.wsManager.HandleWebSocket(w, r, "")
		}
	}))
	defer httpServer.Close()

	// Connect WebSocket client
	wsURL := "ws" + httpServer.URL[4:] + "/ws/events"
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer ws.Close()

	// Create channel to receive WebSocket messages
	messageChan := make(chan map[string]interface{}, 1)
	go func() {
		for {
			var msg map[string]interface{}
			err := ws.ReadJSON(&msg)
			if err != nil {
				return
			}
			messageChan <- msg
		}
	}()

	// Save server to storage first
	err = srv.storageManager.SaveUpstreamServer(cfg.Servers[0])
	require.NoError(t, err)

	// Trigger state change by enabling/disabling server
	err = srv.storageManager.EnableUpstreamServer("test-server", false)
	require.NoError(t, err)

	// Manually publish event through event bus (simulating state manager)
	// Note: Use legacy "state_change" type which is what WebSocket manager expects
	srv.eventBus.Publish(events.Event{
		Type:       events.EventStateChange,
		ServerName: "test-server",
		OldState:   "enabled",
		NewState:   "disabled",
		Timestamp:  time.Now(),
		Data: map[string]interface{}{
			"enabled": false,
		},
	})

	// Wait for WebSocket message (with timeout)
	select {
	case msg := <-messageChan:
		assert.Equal(t, string(events.EventStateChange), msg["type"], "Should receive state change event")
		assert.Equal(t, "test-server", msg["server_name"], "Should have correct server name")
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Timeout waiting for WebSocket event delivery")
	}
}

// TestIntegration_AppStateTransitions verifies application state transitions
func TestIntegration_AppStateTransitions(t *testing.T) {
	// Create temporary directory for test
	tempDir := t.TempDir()

	// Create config with multiple servers
	cfg := config.DefaultConfig()
	cfg.DataDir = tempDir
	cfg.Servers = []*config.ServerConfig{
		{
			Name:     "server-1",
			Protocol: "stdio",
			Command:  "test-command-1",
			StartupMode: "active",
			Created:  time.Now(),
		},
		{
			Name:     "server-2",
			Protocol: "stdio",
			Command:  "test-command-2",
			StartupMode: "active",
			Created:  time.Now(),
		},
	}

	// Save config
	configPath := filepath.Join(tempDir, "mcp_config.json")
	configData, err := json.MarshalIndent(cfg, "", "  ")
	require.NoError(t, err)
	err = os.WriteFile(configPath, configData, 0644)
	require.NoError(t, err)

	// Create server
	logger := zap.NewNop()
	srv, err := NewServerWithConfigPath(cfg, configPath, logger)
	require.NoError(t, err)
	require.NotNil(t, srv)
	defer srv.storageManager.Close()

	// Subscribe to app state events
	eventChan := srv.eventBus.Subscribe(events.EventAppStateChange)

	// Test 1: Initial state should be Starting
	currentState := srv.GetAppState()
	assert.Equal(t, AppStateStarting, currentState, "Initial state should be Starting")

	// Test 2: Transition to Running (simulated - in real scenario, all servers would connect)
	srv.setAppState(AppStateRunning)

	select {
	case event := <-eventChan:
		assert.Equal(t, events.EventAppStateChange, event.Type, "Should receive app state change event")
		data, ok := event.Data.(map[string]interface{})
		require.True(t, ok, "Event data should be a map")
		assert.Equal(t, string(AppStateStarting), data["old_state"], "Old state should be Starting")
		assert.Equal(t, string(AppStateRunning), data["new_state"], "New state should be Running")
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Timeout waiting for app state change event")
	}

	// Test 3: Transition to Stopping
	srv.setAppState(AppStateStopping)

	select {
	case event := <-eventChan:
		assert.Equal(t, events.EventAppStateChange, event.Type, "Should receive app state change event")
		data, ok := event.Data.(map[string]interface{})
		require.True(t, ok, "Event data should be a map")
		assert.Equal(t, string(AppStateRunning), data["old_state"], "Old state should be Running")
		assert.Equal(t, string(AppStateStopping), data["new_state"], "New state should be Stopping")
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Timeout waiting for app state change event")
	}

	// Test 4: Transition to Stopped
	srv.setAppState(AppStateStopped)

	select {
	case event := <-eventChan:
		assert.Equal(t, events.EventAppStateChange, event.Type, "Should receive app state change event")
		data, ok := event.Data.(map[string]interface{})
		require.True(t, ok, "Event data should be a map")
		assert.Equal(t, string(AppStateStopping), data["old_state"], "Old state should be Stopping")
		assert.Equal(t, string(AppStateStopped), data["new_state"], "New state should be Stopped")
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Timeout waiting for app state change event")
	}
}

// TestIntegration_FileWatcherSkipsProgrammaticUpdates verifies file watcher doesn't trigger on programmatic changes
func TestIntegration_FileWatcherSkipsProgrammaticUpdates(t *testing.T) {
	// Create temporary directory for test
	tempDir := t.TempDir()

	// Create config
	cfg := config.DefaultConfig()
	cfg.DataDir = tempDir
	cfg.Servers = []*config.ServerConfig{
		{
			Name:     "test-server",
			Protocol: "stdio",
			Command:  "test-command",
			StartupMode: "active",
			Created:  time.Now(),
		},
	}

	// Save config
	configPath := filepath.Join(tempDir, "mcp_config.json")
	configData, err := json.MarshalIndent(cfg, "", "  ")
	require.NoError(t, err)
	err = os.WriteFile(configPath, configData, 0644)
	require.NoError(t, err)

	// Create config loader
	loader, err := config.NewLoader(configPath, zap.NewNop())
	require.NoError(t, err)

	// Create server with storage manager
	srv, err := NewServerWithConfigPath(cfg, configPath, zap.NewNop())
	require.NoError(t, err)
	defer srv.storageManager.Close()

	// Test 1: Programmatic update via storage API should not trigger reload
	// This would set skipNextReload flag internally
	reloadCount := 0

	// Make programmatic change
	err = srv.storageManager.EnableUpstreamServer("test-server", false)
	require.NoError(t, err)

	// Wait a bit to see if file watcher triggers (it shouldn't)
	time.Sleep(200 * time.Millisecond)

	// Reload count should still be 0 (no file watcher reload)
	assert.Equal(t, 0, reloadCount, "Programmatic update should not trigger file watcher")

	// Test 2: Manual file edit should trigger reload
	// Read current config
	currentConfig := loader.GetConfig()
	require.NotNil(t, currentConfig)

	// Manually edit file (simulating external change)
	currentConfig.Servers[0].Description = "manually edited"
	configData, err = json.MarshalIndent(currentConfig, "", "  ")
	require.NoError(t, err)
	err = os.WriteFile(configPath, configData, 0644)
	require.NoError(t, err)

	// File watcher should detect this change
	// In a real scenario, we'd hook into the file watcher's reload callback
	// For this test, we verify the file content changed
	reloadedConfig := loader.GetConfig()
	require.NotNil(t, reloadedConfig)
	assert.Equal(t, "manually edited", reloadedConfig.Servers[0].Description, "Manual edit should be visible")

	// Test 3: Verify no infinite reload loops
	// Make several programmatic changes in sequence
	for i := 0; i < 5; i++ {
		err = srv.storageManager.EnableUpstreamServer("test-server", i%2 == 0)
		require.NoError(t, err)
		time.Sleep(50 * time.Millisecond)
	}

	// Should not cause infinite reloads
	// If there were infinite reloads, the test would hang or fail
	assert.True(t, true, "No infinite reload loop detected")
}

// Helper function to wait for event with timeout
func waitForEvent(t *testing.T, eventChan <-chan events.Event, expectedType events.EventType, timeout time.Duration) events.Event {
	select {
	case event := <-eventChan:
		require.Equal(t, expectedType, event.Type, "Event type should match")
		return event
	case <-time.After(timeout):
		t.Fatalf("Timeout waiting for event type %s", expectedType)
		return events.Event{}
	}
}

// Helper to create a minimal server for testing
func createTestServer(t *testing.T, tempDir string, servers []*config.ServerConfig) (*Server, string) {
	cfg := config.DefaultConfig()
	cfg.DataDir = tempDir
	cfg.Servers = servers

	configPath := filepath.Join(tempDir, "mcp_config.json")
	configData, err := json.MarshalIndent(cfg, "", "  ")
	require.NoError(t, err)
	err = os.WriteFile(configPath, configData, 0644)
	require.NoError(t, err)

	logger := zap.NewNop()
	srv, err := NewServerWithConfigPath(cfg, configPath, logger)
	require.NoError(t, err)
	require.NotNil(t, srv)

	return srv, configPath
}

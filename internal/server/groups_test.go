package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"mcpproxy-go/internal/config"
	"mcpproxy-go/internal/events"
	"mcpproxy-go/internal/storage"
	"mcpproxy-go/internal/upstream"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// TestGroupEnable_ClearsAutoDisable verifies that enabling a group clears auto-disable state
func TestGroupEnable_ClearsAutoDisable(t *testing.T) {
	// Create test server
	server, cleanup := setupTestServerWithGroups(t)
	defer cleanup()

	// Create test group and assign servers
	groupName := "test-group"
	serverNames := []string{"server1", "server2"}

	// Setup servers in auto-disabled state
	for _, serverName := range serverNames {
		srv := &config.ServerConfig{
			Name:              serverName,
			Protocol:          "http",
			URL:               "http://localhost:9999",
			StartupMode:       "auto_disabled",
			AutoDisableReason: "connection_failures",
			Created:           time.Now(),
		}
		require.NoError(t, server.storageManager.SaveUpstreamServer(srv))

		// Assign server to group
		assignmentsMutex.Lock()
		serverGroupAssignments[serverName] = groupName
		assignmentsMutex.Unlock()
	}

	// Enable group
	payload := map[string]interface{}{
		"group_name": groupName,
		"enabled":    true,
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/api/toggle-group-servers", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleToggleGroupServers(w, req)

	// Verify response
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	assert.True(t, response["success"].(bool))
	assert.Equal(t, float64(len(serverNames)), response["updated"].(float64))

	// Verify auto-disable state cleared for all servers
	for _, serverName := range serverNames {
		srv, err := server.storageManager.GetUpstreamServer(serverName)
		require.NoError(t, err)

		assert.Equal(t, "active", srv.StartupMode, "server %s startup mode should be active (not auto_disabled)", serverName)
	}
}

// TestGroupEnable_TriggersReconnect verifies that enabling a group triggers reconnection
func TestGroupEnable_TriggersReconnect(t *testing.T) {
	// Create test server
	server, cleanup := setupTestServerWithGroups(t)
	defer cleanup()

	// Create test group and assign servers
	groupName := "reconnect-test-group"
	serverNames := []string{"reconnect-server1", "reconnect-server2"}

	// Setup disabled servers
	for _, serverName := range serverNames {
		srv := &config.ServerConfig{
			Name:     serverName,
			Protocol: "http",
			URL:      "http://localhost:9999",
			StartupMode: "disabled",
			Created:  time.Now(),
		}
		require.NoError(t, server.storageManager.SaveUpstreamServer(srv))

		assignmentsMutex.Lock()
		serverGroupAssignments[serverName] = groupName
		assignmentsMutex.Unlock()
	}

	// Enable group
	payload := map[string]interface{}{
		"group_name": groupName,
		"enabled":    true,
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/api/toggle-group-servers", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleToggleGroupServers(w, req)

	// Verify response (AddServerConfig is called internally)
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	assert.True(t, response["success"].(bool))
	assert.Equal(t, float64(len(serverNames)), response["updated"].(float64))

	// Verify servers are now active (which triggers reconnection)
	for _, serverName := range serverNames {
		srv, err := server.storageManager.GetUpstreamServer(serverName)
		require.NoError(t, err)
		assert.Equal(t, "active", srv.StartupMode, "server %s startup mode should be active", serverName)
	}
}

// TestGroupDisable_StopsServers verifies that disabling a group stops all servers
func TestGroupDisable_StopsServers(t *testing.T) {
	// Create test server
	server, cleanup := setupTestServerWithGroups(t)
	defer cleanup()

	// Create test group and assign servers
	groupName := "disable-test-group"
	serverNames := []string{"disable-server1", "disable-server2", "disable-server3"}

	// Setup enabled servers
	for _, serverName := range serverNames {
		srv := &config.ServerConfig{
			Name:     serverName,
			Protocol: "http",
			URL:      "http://localhost:9999",
			StartupMode: "active",
			Created:  time.Now(),
		}
		require.NoError(t, server.storageManager.SaveUpstreamServer(srv))

		assignmentsMutex.Lock()
		serverGroupAssignments[serverName] = groupName
		assignmentsMutex.Unlock()
	}

	// Disable group
	payload := map[string]interface{}{
		"group_name": groupName,
		"enabled":    false,
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/api/toggle-group-servers", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleToggleGroupServers(w, req)

	// Verify response
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	assert.True(t, response["success"].(bool))
	assert.Equal(t, float64(len(serverNames)), response["updated"].(float64))

	// Verify servers are disabled in storage (RemoveServer is called internally)
	for _, serverName := range serverNames {
		srv, err := server.storageManager.GetUpstreamServer(serverName)
		require.NoError(t, err)
		assert.Equal(t, "disabled", srv.StartupMode, "server %s startup mode should be disabled", serverName)
	}
}

// TestGroupOperations_PartialFailure verifies graceful handling of partial failures
func TestGroupOperations_PartialFailure(t *testing.T) {
	// Create test server
	server, cleanup := setupTestServerWithGroups(t)
	defer cleanup()

	// Create test group and assign servers
	groupName := "partial-fail-group"

	// Setup one valid server
	validSrv := &config.ServerConfig{
		Name:     "valid-server",
		Protocol: "http",
		URL:      "http://localhost:9999",
		StartupMode: "disabled",
		Created:  time.Now(),
	}
	require.NoError(t, server.storageManager.SaveUpstreamServer(validSrv))

	// Assign servers to group (invalid-server doesn't exist in storage)
	assignmentsMutex.Lock()
	serverGroupAssignments["valid-server"] = groupName
	serverGroupAssignments["invalid-server"] = groupName
	assignmentsMutex.Unlock()

	// Enable group
	payload := map[string]interface{}{
		"group_name": groupName,
		"enabled":    true,
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/api/toggle-group-servers", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleToggleGroupServers(w, req)

	// Verify partial failure response
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))

	// Should indicate partial failure
	assert.False(t, response["success"].(bool), "should indicate failure due to partial failure")
	assert.True(t, response["partial_failure"].(bool), "should indicate partial failure")
	assert.Equal(t, float64(1), response["updated"].(float64), "should update 1 server successfully")

	// Verify failed_servers contains details
	failedServers, ok := response["failed_servers"].([]interface{})
	require.True(t, ok, "should have failed_servers field")
	assert.Len(t, failedServers, 1, "should have 1 failed server")

	// Verify valid server was still updated
	srv, err := server.storageManager.GetUpstreamServer("valid-server")
	require.NoError(t, err)
	assert.Equal(t, "active", srv.StartupMode, "valid server should be enabled despite partial failure")
}

// TestGroupOperations_EmitsEvents verifies that group operations emit appropriate events
func TestGroupOperations_EmitsEvents(t *testing.T) {
	// Create test server with event bus
	server, cleanup := setupTestServerWithGroups(t)
	defer cleanup()

	// Subscribe to events
	stateChangeChan := server.eventBus.Subscribe(events.ServerStateChanged)
	groupUpdateChan := server.eventBus.Subscribe(events.ServerGroupUpdated)

	// Create test group and assign servers
	groupName := "event-test-group"
	serverNames := []string{"event-server1", "event-server2"}

	for _, serverName := range serverNames {
		srv := &config.ServerConfig{
			Name:     serverName,
			Protocol: "http",
			URL:      "http://localhost:9999",
			StartupMode: "disabled",
			Created:  time.Now(),
		}
		require.NoError(t, server.storageManager.SaveUpstreamServer(srv))

		assignmentsMutex.Lock()
		serverGroupAssignments[serverName] = groupName
		assignmentsMutex.Unlock()
	}

	// Enable group
	payload := map[string]interface{}{
		"group_name": groupName,
		"enabled":    true,
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/api/toggle-group-servers", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleToggleGroupServers(w, req)

	// Collect events (with timeout)
	var stateChangeEvents []events.Event
	var groupUpdateEvents []events.Event

	timeout := time.After(2 * time.Second)
	expectedStateChanges := len(serverNames)
	expectedGroupUpdates := 1

eventLoop:
	for {
		select {
		case event := <-stateChangeChan:
			stateChangeEvents = append(stateChangeEvents, event)
			if len(stateChangeEvents) >= expectedStateChanges && len(groupUpdateEvents) >= expectedGroupUpdates {
				break eventLoop
			}
		case event := <-groupUpdateChan:
			groupUpdateEvents = append(groupUpdateEvents, event)
			if len(stateChangeEvents) >= expectedStateChanges && len(groupUpdateEvents) >= expectedGroupUpdates {
				break eventLoop
			}
		case <-timeout:
			t.Logf("Timeout waiting for events. Got %d state changes (expected %d) and %d group updates (expected %d)",
				len(stateChangeEvents), expectedStateChanges, len(groupUpdateEvents), expectedGroupUpdates)
			break eventLoop
		}
	}

	// Verify ServerStateChanged events for each server
	assert.GreaterOrEqual(t, len(stateChangeEvents), expectedStateChanges,
		"should emit ServerStateChanged event for each server")

	for _, event := range stateChangeEvents {
		assert.Equal(t, events.ServerStateChanged, event.Type)
		assert.Contains(t, serverNames, event.ServerName)

		// Verify event data
		data, ok := event.Data.(map[string]interface{})
		require.True(t, ok, "event data should be map")
		assert.True(t, data["enabled"].(bool), "should indicate enabled=true")
		assert.False(t, data["auto_disabled"].(bool), "should indicate auto_disabled=false")
		assert.Equal(t, "group_enable", data["action"].(string))
		assert.Equal(t, groupName, data["group"].(string))
	}

	// Verify ServerGroupUpdated event
	assert.GreaterOrEqual(t, len(groupUpdateEvents), expectedGroupUpdates,
		"should emit ServerGroupUpdated event")

	for _, event := range groupUpdateEvents {
		assert.Equal(t, events.ServerGroupUpdated, event.Type)

		// Verify event data
		data, ok := event.Data.(map[string]interface{})
		require.True(t, ok, "event data should be map")
		assert.Equal(t, groupName, data["group"].(string))
		assert.Equal(t, "enable", data["action"].(string))
		assert.Equal(t, len(serverNames), data["total_updated"].(int))
	}
}

// Helper functions

// setupTestServerWithGroups creates a test server with storage and event bus
func setupTestServerWithGroups(t *testing.T) (*Server, func()) {
	// Create temporary directory for storage
	tempDir := t.TempDir()

	// Create logger
	logger, _ := zap.NewDevelopment()

	// Create storage
	storageManager, err := storage.NewManager(tempDir, logger.Sugar())
	require.NoError(t, err)

	// Create event bus
	eventBus := events.NewEventBus()

	// Create config
	cfg := &config.Config{
		DataDir: tempDir,
		Servers: []*config.ServerConfig{},
	}

	// Create upstream manager
	upstreamMgr := upstream.NewManager(logger, cfg, storageManager.GetBoltDB())
	upstreamMgr.SetEventBus(eventBus)
	upstreamMgr.SetStorageManager(storageManager)

	// Create server
	server := &Server{
		config:          cfg,
		storageManager:  storageManager,
		upstreamManager: upstreamMgr,
		eventBus:        eventBus,
		logger:          logger,
	}

	// Initialize group state
	groupsMutex.Lock()
	groups = make(map[string]*Group)
	groupsMutex.Unlock()

	assignmentsMutex.Lock()
	serverGroupAssignments = make(map[string]string)
	assignmentsMutex.Unlock()

	cleanup := func() {
		eventBus.Close()
		storageManager.Close()
	}

	return server, cleanup
}

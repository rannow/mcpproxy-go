package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"mcpproxy-go/internal/events"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestNewWebSocketManager(t *testing.T) {
	eventBus := events.NewEventBus()
	defer eventBus.Close()

	logger := zap.NewNop().Sugar()

	manager := NewWebSocketManager(eventBus, logger)
	assert.NotNil(t, manager)
	assert.NotNil(t, manager.eventBus)
	assert.NotNil(t, manager.logger)
	assert.NotNil(t, manager.connections)
	assert.NotNil(t, manager.register)
	assert.NotNil(t, manager.unregister)
	assert.NotNil(t, manager.stopChan)

	// Cleanup
	manager.Stop()
}

func TestWebSocketConnection(t *testing.T) {
	eventBus := events.NewEventBus()
	defer eventBus.Close()

	logger := zap.NewNop().Sugar()
	manager := NewWebSocketManager(eventBus, logger)
	defer manager.Stop()

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		manager.HandleWebSocket(w, r, "")
	}))
	defer server.Close()

	// Connect WebSocket client
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer conn.Close()

	// Wait for connection to be registered
	time.Sleep(100 * time.Millisecond)

	// Verify connection count
	assert.Equal(t, 1, manager.GetActiveConnections())

	// Close connection
	conn.Close()

	// Wait for unregistration
	time.Sleep(100 * time.Millisecond)

	// Verify connection count decreased
	assert.Equal(t, 0, manager.GetActiveConnections())
}

func TestWebSocketEventBroadcast(t *testing.T) {
	eventBus := events.NewEventBus()
	defer eventBus.Close()

	logger := zap.NewNop().Sugar()
	manager := NewWebSocketManager(eventBus, logger)
	defer manager.Stop()

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		manager.HandleWebSocket(w, r, "")
	}))
	defer server.Close()

	// Connect WebSocket client
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer conn.Close()

	// Wait for connection to be registered
	time.Sleep(100 * time.Millisecond)

	// Publish an event
	testEvent := events.Event{
		Type:       events.EventStateChange,
		ServerName: "test-server",
		OldState:   "connecting",
		NewState:   "ready",
		Data: map[string]interface{}{
			"test": "data",
		},
	}
	eventBus.Publish(testEvent)

	// Read the event from WebSocket
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, message, err := conn.ReadMessage()
	require.NoError(t, err)

	// Unmarshal and verify
	var receivedEvent events.Event
	err = json.Unmarshal(message, &receivedEvent)
	require.NoError(t, err)

	assert.Equal(t, events.EventStateChange, receivedEvent.Type)
	assert.Equal(t, "test-server", receivedEvent.ServerName)
	assert.Equal(t, "connecting", receivedEvent.OldState)
	assert.Equal(t, "ready", receivedEvent.NewState)
}

func TestWebSocketServerFilter(t *testing.T) {
	eventBus := events.NewEventBus()
	defer eventBus.Close()

	logger := zap.NewNop().Sugar()
	manager := NewWebSocketManager(eventBus, logger)
	defer manager.Stop()

	// Create test server with filter
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Filter for "server-a" only
		manager.HandleWebSocket(w, r, "server-a")
	}))
	defer server.Close()

	// Connect WebSocket client
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer conn.Close()

	// Wait for connection to be registered
	time.Sleep(100 * time.Millisecond)

	// Publish event for server-a (should be received)
	eventBus.Publish(events.Event{
		Type:       events.EventStateChange,
		ServerName: "server-a",
		NewState:   "ready",
	})

	// Read the event (should succeed)
	conn.SetReadDeadline(time.Now().Add(1 * time.Second))
	_, message, err := conn.ReadMessage()
	require.NoError(t, err)

	var receivedEvent events.Event
	err = json.Unmarshal(message, &receivedEvent)
	require.NoError(t, err)
	assert.Equal(t, "server-a", receivedEvent.ServerName)

	// Publish event for server-b (should NOT be received)
	eventBus.Publish(events.Event{
		Type:       events.EventStateChange,
		ServerName: "server-b",
		NewState:   "ready",
	})

	// Try to read (should timeout since event is filtered)
	conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	_, _, err = conn.ReadMessage()
	assert.Error(t, err) // Should timeout
}

func TestWebSocketMultipleClients(t *testing.T) {
	eventBus := events.NewEventBus()
	defer eventBus.Close()

	logger := zap.NewNop().Sugar()
	manager := NewWebSocketManager(eventBus, logger)
	defer manager.Stop()

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		manager.HandleWebSocket(w, r, "")
	}))
	defer server.Close()

	// Connect multiple clients
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	conn1, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer conn1.Close()

	conn2, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer conn2.Close()

	conn3, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer conn3.Close()

	// Wait for connections to be registered
	time.Sleep(100 * time.Millisecond)

	// Verify connection count
	assert.Equal(t, 3, manager.GetActiveConnections())

	// Publish an event
	testEvent := events.Event{
		Type:       events.EventConfigChange,
		ServerName: "test-server",
		NewState:   "updated",
	}
	eventBus.Publish(testEvent)

	// All clients should receive the event
	for i, conn := range []*websocket.Conn{conn1, conn2, conn3} {
		conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		_, message, err := conn.ReadMessage()
		require.NoError(t, err, "client %d should receive event", i+1)

		var receivedEvent events.Event
		err = json.Unmarshal(message, &receivedEvent)
		require.NoError(t, err)
		assert.Equal(t, events.EventConfigChange, receivedEvent.Type)
	}
}

func TestWebSocketPingPong(t *testing.T) {
	eventBus := events.NewEventBus()
	defer eventBus.Close()

	logger := zap.NewNop().Sugar()
	manager := NewWebSocketManager(eventBus, logger)
	defer manager.Stop()

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		manager.HandleWebSocket(w, r, "")
	}))
	defer server.Close()

	// Connect WebSocket client with pong handler
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer conn.Close()

	// Set pong handler (just to verify connection health mechanism works)
	conn.SetPongHandler(func(string) error {
		return nil
	})

	// Wait for ping (pingPeriod is 54 seconds, so we won't wait that long in test)
	// Instead, just verify connection stays alive
	time.Sleep(200 * time.Millisecond)

	// Connection should still be active
	assert.Equal(t, 1, manager.GetActiveConnections())
}

func TestWebSocketManagerStop(t *testing.T) {
	eventBus := events.NewEventBus()
	defer eventBus.Close()

	logger := zap.NewNop().Sugar()
	manager := NewWebSocketManager(eventBus, logger)

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		manager.HandleWebSocket(w, r, "")
	}))
	defer server.Close()

	// Connect multiple clients
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	conn1, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer conn1.Close()

	conn2, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer conn2.Close()

	// Wait for connections
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, 2, manager.GetActiveConnections())

	// Stop manager
	manager.Stop()

	// Wait for cleanup
	time.Sleep(100 * time.Millisecond)

	// All connections should be closed
	assert.Equal(t, 0, manager.GetActiveConnections())
}

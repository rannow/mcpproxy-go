package server

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"mcpproxy-go/internal/config"
	"mcpproxy-go/internal/events"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// TestWebSocketRoutesIntegration tests the WebSocket routes with a full server instance
func TestWebSocketRoutesIntegration(t *testing.T) {
	// Create test config with a specific test port
	cfg := config.DefaultConfig()
	cfg.Listen = "127.0.0.1:18765" // Fixed test port
	cfg.DataDir = t.TempDir()

	// Create logger
	logger := zap.NewNop()

	// Create server
	srv, err := NewServer(cfg, logger)
	require.NoError(t, err)
	defer srv.Shutdown()

	// Start server
	ctx := context.Background()
	err = srv.StartServer(ctx)
	require.NoError(t, err)

	// Wait for server to start
	time.Sleep(200 * time.Millisecond)

	// Get server address
	addr := "127.0.0.1:18765"

	// Test /ws/events endpoint
	t.Run("ws_events_endpoint", func(t *testing.T) {
		wsURL := "ws://" + addr + "/ws/events"
		ws, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
		require.NoError(t, err)
		require.Equal(t, http.StatusSwitchingProtocols, resp.StatusCode)
		defer ws.Close()

		// Publish an event
		srv.eventBus.Publish(events.Event{
			Type:       events.ServerStateChanged,
			ServerName: "test-server",
			Data: map[string]interface{}{
				"old_state": "disconnected",
				"new_state": "connected",
			},
		})

		// Read event from WebSocket
		ws.SetReadDeadline(time.Now().Add(2 * time.Second))
		_, message, err := ws.ReadMessage()
		require.NoError(t, err)

		// Parse event
		var receivedEvent events.Event
		err = json.Unmarshal(message, &receivedEvent)
		require.NoError(t, err)

		// Verify event
		assert.Equal(t, events.ServerStateChanged, receivedEvent.Type)
		assert.Equal(t, "test-server", receivedEvent.ServerName)
	})

	// Test /ws/servers endpoint with server filter
	t.Run("ws_servers_endpoint_with_filter", func(t *testing.T) {
		wsURL := "ws://" + addr + "/ws/servers?server=server-a"
		ws, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
		require.NoError(t, err)
		require.Equal(t, http.StatusSwitchingProtocols, resp.StatusCode)
		defer ws.Close()

		// Publish events for different servers
		srv.eventBus.Publish(events.Event{
			Type:       events.ServerStateChanged,
			ServerName: "server-a",
			Data:       map[string]interface{}{"state": "connected"},
		})

		srv.eventBus.Publish(events.Event{
			Type:       events.ServerStateChanged,
			ServerName: "server-b",
			Data:       map[string]interface{}{"state": "disconnected"},
		})

		// Read event from WebSocket (should only get server-a event)
		ws.SetReadDeadline(time.Now().Add(2 * time.Second))
		_, message, err := ws.ReadMessage()
		require.NoError(t, err)

		// Parse event
		var receivedEvent events.Event
		err = json.Unmarshal(message, &receivedEvent)
		require.NoError(t, err)

		// Verify only server-a event was received
		assert.Equal(t, "server-a", receivedEvent.ServerName)

		// Try to read another message - should timeout (no server-b event)
		ws.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
		_, _, err = ws.ReadMessage()
		assert.Error(t, err) // Timeout error expected
	})

	// Test multiple concurrent connections
	t.Run("multiple_connections", func(t *testing.T) {
		wsURL := "ws://" + addr + "/ws/events"

		initialCount := srv.wsManager.GetActiveConnections()

		// Connect 3 clients
		clients := make([]*websocket.Conn, 3)
		for i := 0; i < 3; i++ {
			ws, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
			require.NoError(t, err)
			require.Equal(t, http.StatusSwitchingProtocols, resp.StatusCode)
			defer ws.Close()
			clients[i] = ws
		}

		// Wait for all connections to register
		time.Sleep(100 * time.Millisecond)

		// Verify connection count (initial + 3 new)
		assert.Equal(t, initialCount+3, srv.wsManager.GetActiveConnections())

		// Publish event
		srv.eventBus.Publish(events.Event{
			Type:       events.ToolsUpdated,
			ServerName: "test-server",
			Data:       map[string]interface{}{"count": 42},
		})

		// All 3 clients should receive the event
		for i, ws := range clients {
			ws.SetReadDeadline(time.Now().Add(2 * time.Second))
			_, message, err := ws.ReadMessage()
			require.NoError(t, err, "Client %d should receive message", i)

			var receivedEvent events.Event
			err = json.Unmarshal(message, &receivedEvent)
			require.NoError(t, err)
			assert.Equal(t, events.ToolsUpdated, receivedEvent.Type)
		}
	})

	// Test connection cleanup on close
	t.Run("connection_cleanup", func(t *testing.T) {
		wsURL := "ws://" + addr + "/ws/events"

		initialCount := srv.wsManager.GetActiveConnections()

		ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		require.NoError(t, err)

		// Wait for connection registration
		time.Sleep(100 * time.Millisecond)
		assert.Equal(t, initialCount+1, srv.wsManager.GetActiveConnections())

		// Close connection
		ws.Close()

		// Wait for cleanup (longer wait for cleanup to complete)
		time.Sleep(300 * time.Millisecond)
		assert.Equal(t, initialCount, srv.wsManager.GetActiveConnections())
	})
}

// TestWebSocketShutdown tests that WebSocket connections are properly closed on server shutdown
func TestWebSocketShutdown(t *testing.T) {
	// Create test config
	cfg := config.DefaultConfig()
	cfg.Listen = "127.0.0.1:18766"
	cfg.DataDir = t.TempDir()

	// Create logger
	logger := zap.NewNop()

	// Create server
	srv, err := NewServer(cfg, logger)
	require.NoError(t, err)

	// Start server
	ctx := context.Background()
	err = srv.StartServer(ctx)
	require.NoError(t, err)

	// Wait for server to start
	time.Sleep(200 * time.Millisecond)

	addr := "127.0.0.1:18766"
	wsURL := "ws://" + addr + "/ws/events"

	// Connect client
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)

	// Wait for connection
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, 1, srv.wsManager.GetActiveConnections())

	// Shutdown server
	err = srv.Shutdown()
	require.NoError(t, err)

	// Try to read from connection - should fail with close error
	ws.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, _, err = ws.ReadMessage()
	assert.Error(t, err)
	assert.True(t, websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway, websocket.CloseAbnormalClosure))

	ws.Close()
}

// TestWebSocketPingPongIntegration tests ping/pong mechanism in real server
func TestWebSocketPingPongIntegration(t *testing.T) {
	// Create test config
	cfg := config.DefaultConfig()
	cfg.Listen = "127.0.0.1:18767"
	cfg.DataDir = t.TempDir()

	logger := zap.NewNop()
	srv, err := NewServer(cfg, logger)
	require.NoError(t, err)
	defer srv.Shutdown()

	ctx := context.Background()
	err = srv.StartServer(ctx)
	require.NoError(t, err)

	time.Sleep(200 * time.Millisecond)

	addr := "127.0.0.1:18767"
	wsURL := "ws://" + addr + "/ws/events"

	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer ws.Close()

	// Set up pong handler
	pongReceived := make(chan bool, 1)
	ws.SetPongHandler(func(string) error {
		pongReceived <- true
		return nil
	})

	// Start reading messages (needed for pong handler to be called)
	go func() {
		for {
			_, _, err := ws.ReadMessage()
			if err != nil {
				return
			}
		}
	}()

	// Send ping
	err = ws.WriteMessage(websocket.PingMessage, nil)
	require.NoError(t, err)

	// Wait for pong (with timeout)
	select {
	case <-pongReceived:
		// Success
	case <-time.After(2 * time.Second):
		t.Fatal("Did not receive pong response")
	}
}

// TestWebSocketInvalidUpgrade tests that invalid upgrade requests are rejected
func TestWebSocketInvalidUpgrade(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Listen = "127.0.0.1:18768"
	cfg.DataDir = t.TempDir()

	logger := zap.NewNop()
	srv, err := NewServer(cfg, logger)
	require.NoError(t, err)
	defer srv.Shutdown()

	ctx := context.Background()
	err = srv.StartServer(ctx)
	require.NoError(t, err)

	time.Sleep(200 * time.Millisecond)

	addr := "127.0.0.1:18768"

	// Try regular HTTP request to WebSocket endpoint
	resp, err := http.Get("http://" + addr + "/ws/events")
	require.NoError(t, err)
	defer resp.Body.Close()

	// Should fail with bad request or upgrade required
	assert.True(t, resp.StatusCode == http.StatusBadRequest ||
		resp.StatusCode == http.StatusUpgradeRequired)
}

// TestWebSocketEventTypes tests that all event types are properly broadcast
func TestWebSocketEventTypes(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Listen = "127.0.0.1:18769"
	cfg.DataDir = t.TempDir()

	logger := zap.NewNop()
	srv, err := NewServer(cfg, logger)
	require.NoError(t, err)
	defer srv.Shutdown()

	ctx := context.Background()
	err = srv.StartServer(ctx)
	require.NoError(t, err)

	time.Sleep(200 * time.Millisecond)

	addr := "127.0.0.1:18769"
	wsURL := "ws://" + addr + "/ws/events"

	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer ws.Close()

	// Test different event types
	eventTypes := []events.EventType{
		events.ServerStateChanged,
		events.ServerConfigChanged,
		events.ServerAutoDisabled,
		events.ServerGroupUpdated,
		events.ToolsUpdated,
		events.ToolCalled,
		events.ConnectionEstablished,
		events.ConnectionLost,
	}

	for _, eventType := range eventTypes {
		t.Run(string(eventType), func(t *testing.T) {
			// Publish event
			srv.eventBus.Publish(events.Event{
				Type:       eventType,
				ServerName: "test-server",
				Data:       map[string]interface{}{"test": "data"},
			})

			// Read event
			ws.SetReadDeadline(time.Now().Add(2 * time.Second))
			_, message, err := ws.ReadMessage()
			require.NoError(t, err)

			var receivedEvent events.Event
			err = json.Unmarshal(message, &receivedEvent)
			require.NoError(t, err)

			assert.Equal(t, eventType, receivedEvent.Type)
			assert.Equal(t, "test-server", receivedEvent.ServerName)
		})
	}
}

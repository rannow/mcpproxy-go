package server

import (
	"encoding/json"
	"net/http"
	"reflect"
	"sync"
	"time"

	"mcpproxy-go/internal/events"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

const (
	// WebSocket settings
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512 * 1024 // 512 KB
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins for now - can be restricted later
		return true
	},
}

// WebSocketManager manages WebSocket connections and event broadcasting
type WebSocketManager struct {
	eventBus    *events.EventBus
	logger      *zap.SugaredLogger
	connections map[*websocket.Conn]*wsClient
	mu          sync.RWMutex
	register    chan *wsClient
	unregister  chan *wsClient
	stopChan    chan struct{}
}

// wsClient represents a WebSocket client connection
type wsClient struct {
	conn         *websocket.Conn
	send         chan []byte
	manager      *WebSocketManager
	eventChannels []<-chan events.Event // Multiple channels for different event types
	filterServer string // If set, only send events for this server
	stopChan     chan struct{}
}

// NewWebSocketManager creates a new WebSocket manager
func NewWebSocketManager(eventBus *events.EventBus, logger *zap.SugaredLogger) *WebSocketManager {
	manager := &WebSocketManager{
		eventBus:    eventBus,
		logger:      logger,
		connections: make(map[*websocket.Conn]*wsClient),
		register:    make(chan *wsClient),
		unregister:  make(chan *wsClient),
		stopChan:    make(chan struct{}),
	}

	// Start the connection manager
	go manager.run()

	return manager
}

// run manages client registration and event broadcasting
func (m *WebSocketManager) run() {
	for {
		select {
		case client := <-m.register:
			m.mu.Lock()
			m.connections[client.conn] = client
			m.mu.Unlock()
			m.logger.Info("WebSocket client registered",
				zap.Int("total_clients", len(m.connections)))

		case client := <-m.unregister:
			m.mu.Lock()
			if _, ok := m.connections[client.conn]; ok {
				delete(m.connections, client.conn)
				close(client.send)
			}
			m.mu.Unlock()
			m.logger.Info("WebSocket client unregistered",
				zap.Int("total_clients", len(m.connections)))

		case <-m.stopChan:
			m.mu.Lock()
			for conn, client := range m.connections {
				close(client.send)
				conn.Close()
			}
			m.connections = make(map[*websocket.Conn]*wsClient)
			m.mu.Unlock()
			return
		}
	}
}

// Stop stops the WebSocket manager and closes all connections
func (m *WebSocketManager) Stop() {
	close(m.stopChan)
}

// HandleWebSocket handles WebSocket connection upgrades
func (m *WebSocketManager) HandleWebSocket(w http.ResponseWriter, r *http.Request, filterServer string) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		m.logger.Error("Failed to upgrade WebSocket connection", zap.Error(err))
		return
	}

	// Subscribe to all relevant event types
	eventChannels := []<-chan events.Event{
		m.eventBus.Subscribe(events.ServerStateChanged),
		m.eventBus.Subscribe(events.ServerConfigChanged),
		m.eventBus.Subscribe(events.ServerAutoDisabled),
		m.eventBus.Subscribe(events.ServerGroupUpdated),
		m.eventBus.Subscribe(events.EventStateChange),
		m.eventBus.Subscribe(events.EventConfigChange),
		m.eventBus.Subscribe(events.AppStateChanged),
		m.eventBus.Subscribe(events.ToolsUpdated),
		m.eventBus.Subscribe(events.ToolCalled),
		m.eventBus.Subscribe(events.ConnectionEstablished),
		m.eventBus.Subscribe(events.ConnectionLost),
	}

	client := &wsClient{
		conn:          conn,
		send:          make(chan []byte, 256),
		manager:       m,
		eventChannels: eventChannels,
		filterServer:  filterServer,
		stopChan:      make(chan struct{}),
	}

	// Register the client
	m.register <- client

	// Start goroutines for reading and writing
	go client.writePump()
	go client.readPump()
	go client.eventPump()
}

// GetActiveConnections returns the number of active WebSocket connections
func (m *WebSocketManager) GetActiveConnections() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.connections)
}

// readPump pumps messages from the WebSocket connection to handle pongs
func (c *wsClient) readPump() {
	defer func() {
		close(c.stopChan) // Signal event pump to stop
		c.manager.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	// Read messages (mostly just to handle pongs and detect disconnects)
	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.manager.logger.Error("WebSocket read error", zap.Error(err))
			}
			break
		}
	}
}

// writePump pumps messages from the send channel to the WebSocket connection
func (c *wsClient) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Channel closed
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				c.manager.logger.Error("WebSocket write error", zap.Error(err))
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// eventPump reads events from all subscribed channels and forwards them to the client
func (c *wsClient) eventPump() {
	defer func() {
		c.manager.logger.Debug("Event pump stopped for WebSocket client")
	}()

	// Build select cases for all event channels
	cases := make([]reflect.SelectCase, len(c.eventChannels)+1)
	for i, ch := range c.eventChannels {
		cases[i] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(ch)}
	}
	// Add stop channel
	cases[len(c.eventChannels)] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(c.stopChan)}

	for {
		chosen, value, ok := reflect.Select(cases)

		// Check if it's the stop channel
		if chosen == len(c.eventChannels) {
			// Stop channel closed
			return
		}

		if !ok {
			// Channel closed, remove from cases
			cases[chosen].Chan = reflect.ValueOf(nil)
			continue
		}

		// Extract event
		event, ok := value.Interface().(events.Event)
		if !ok {
			c.manager.logger.Error("Failed to cast event from channel")
			continue
		}

		// Apply server filter if set
		if c.filterServer != "" && event.ServerName != c.filterServer {
			continue
		}

		// Marshal event to JSON
		data, err := json.Marshal(event)
		if err != nil {
			c.manager.logger.Error("Failed to marshal event", zap.Error(err))
			continue
		}

		// Try to send to client
		select {
		case c.send <- data:
			// Event sent successfully
		default:
			// Channel full, drop event
			c.manager.logger.Warn("WebSocket send buffer full, dropping event",
				zap.String("event_type", string(event.Type)))
		}
	}
}

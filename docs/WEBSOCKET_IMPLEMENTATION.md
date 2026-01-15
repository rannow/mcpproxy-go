# WebSocket Real-Time Updates Implementation

## Overview

This document describes the WebSocket implementation in MCPProxy that enables real-time event-driven updates for server status monitoring, replacing the previous polling-based approach.

## Architecture

### Backend Components

#### 1. WebSocketManager (`internal/server/websocket.go`)

Central component managing all WebSocket connections and event broadcasting.

**Key Features:**
- Centralized connection tracking and lifecycle management
- Multiple event channel subscriptions (11 event types)
- Server-specific event filtering via query parameters
- Automatic connection cleanup and graceful shutdown
- Ping/pong health monitoring (60-second timeout)

**Event Types Subscribed:**
- `ServerStateChanged` - Server connection state changes
- `ServerConfigChanged` - Server configuration updates
- `ServerAutoDisabled` - Automatic server disabling
- `ServerGroupUpdated` - Server group membership changes
- `EventStateChange` - Legacy state change compatibility
- `EventConfigChange` - Legacy config change compatibility
- `AppStateChanged` - Application-wide state changes
- `ToolsUpdated` - Server tool discovery updates
- `ToolCalled` - Tool execution events
- `ConnectionEstablished` - Server connection established
- `ConnectionLost` - Server connection lost

#### 2. WebSocket Client (`wsClient` struct)

Individual client connection handler with three concurrent goroutines:

- **readPump**: Handles incoming messages (primarily for pong responses)
- **writePump**: Sends outgoing messages and ping keepalives
- **eventPump**: Listens to event channels and forwards to client

**Features:**
- 256-message send buffer per client
- Non-blocking event delivery (drops events if buffer full)
- Server-specific filtering support
- Graceful shutdown coordination

#### 3. HTTP Routes (`internal/server/server.go`)

Two WebSocket endpoints registered:

```go
// All events
GET /ws/events

// Server-specific events (filtered)
GET /ws/servers?server=<name>
```

### Frontend Components

#### Dashboard WebSocket Client (`internal/server/servers_web.go`)

JavaScript WebSocket client with robust error handling and reconnection logic.

**Key Features:**
- Automatic WebSocket protocol selection (ws:// vs wss://)
- Exponential backoff reconnection (5 attempts, max 32s delay)
- Fallback to polling if WebSocket fails completely
- Event-driven UI updates (replaces 5s polling with real-time)
- Reduced fallback polling to 30s (from 5s)

**Event Handling:**
```javascript
switch(eventData.type) {
    case 'server_state_changed':
    case 'state_change':
    case 'server_config_changed':
    case 'config_change':
    case 'server_auto_disabled':
    case 'connection_established':
    case 'connection_lost':
    case 'tools_updated':
        // Refresh server list
        refreshServers();
        break;
}
```

## Event Flow

```
Server Event Occurs
    ↓
EventBus.Publish(event)
    ↓
WebSocketManager receives event (via subscribed channels)
    ↓
eventPump processes event for each connected client
    ↓
Server filter applied (if specified)
    ↓
Event marshaled to JSON
    ↓
Non-blocking send to client's send channel
    ↓
writePump sends JSON to WebSocket connection
    ↓
Browser receives event
    ↓
JavaScript event handler processes event
    ↓
UI refreshes server status (AJAX call to /api/servers/status)
```

## Connection Lifecycle

### Server Startup

1. Create WebSocketManager with EventBus and logger
2. Subscribe to 11 event types
3. Start connection management goroutine
4. Register HTTP routes for WebSocket upgrade
5. Wait for client connections

### Client Connection

1. Browser connects to `/ws/events` or `/ws/servers?server=name`
2. HTTP upgrade to WebSocket protocol
3. Create wsClient with event channel subscriptions
4. Register client with manager
5. Start 3 client goroutines (read, write, event pumps)
6. Initial data load via AJAX

### Client Disconnection

1. Connection close detected (network failure, explicit close, etc.)
2. Signal stopChan to terminate eventPump
3. Unregister client from manager
4. Close send channel
5. Cleanup client resources

### Server Shutdown

1. Call `wsManager.Stop()`
2. Close stopChan (terminates manager goroutine)
3. Close all client send channels
4. Close all WebSocket connections
5. Clear connection map

## Configuration

No configuration required - WebSocket functionality is always enabled.

**WebSocket Settings (constants in `websocket.go`):**
```go
writeWait      = 10 * time.Second  // Write timeout
pongWait       = 60 * time.Second  // Pong timeout
pingPeriod     = 54 * time.Second  // Ping interval (9/10 of pongWait)
maxMessageSize = 512 KB            // Max message size
```

## Testing

### Unit Tests (`websocket_test.go`)

6 tests covering core WebSocket functionality:
- Manager initialization
- Connection establishment
- Event broadcasting
- Server filtering
- Multiple concurrent clients
- Ping/pong mechanism

### Integration Tests (`websocket_integration_test.go`)

5 test suites with full server instances:
- Route integration (events & server filtering)
- Server shutdown behavior
- Ping/pong with real server
- Invalid upgrade rejection
- All 8 event types

**Test Ports:**
- 18765: Main integration tests
- 18766: Shutdown tests
- 18767: Ping/pong tests
- 18768: Invalid upgrade tests
- 18769: Event type tests

## Performance Improvements

### Before (Polling)

- 5-second poll interval for all clients
- N clients = N requests every 5 seconds
- Server CPU load proportional to client count
- Fixed 5-second delay for updates
- Unnecessary network traffic when no changes occur

### After (WebSocket)

- Real-time event delivery (instant updates)
- Reduced polling to 30-second fallback
- 6x reduction in baseline network traffic
- Server pushes only when events occur
- Scales better with concurrent clients

**Example Metrics:**
- 10 clients polling every 5s = 120 requests/minute
- 10 clients with WebSocket + 30s fallback = 20 requests/minute
- **83% reduction in baseline traffic**

## Debugging

### Browser Console

The WebSocket client logs all activity to the browser console:

```javascript
// Connection events
console.log('Connecting to WebSocket:', wsUrl);
console.log('WebSocket connected');
console.log('WebSocket disconnected');

// Received events
console.log('WebSocket event received:', eventData.type, eventData);

// Reconnection attempts
console.log('Reconnecting in 2000ms (attempt 1/5)');

// Errors
console.error('WebSocket error:', error);
console.error('Max reconnection attempts reached, falling back to polling');
```

### Server Logs

WebSocket activity is logged with structured logging:

```go
// Connection events
logger.Info("WebSocket client registered", zap.Int("total_clients", len(m.connections)))
logger.Info("WebSocket client unregistered", zap.Int("total_clients", len(m.connections)))

// Errors
logger.Error("Failed to upgrade WebSocket connection", zap.Error(err))
logger.Error("WebSocket read error", zap.Error(err))
logger.Warn("WebSocket send buffer full, dropping event", zap.String("event_type", string(event.Type)))
```

## Security Considerations

### CORS

Currently configured to allow all origins:

```go
CheckOrigin: func(r *http.Request) bool {
    return true // Allow all origins for now
}
```

**Production Recommendation:** Restrict to specific allowed origins.

### Authentication

No authentication currently required for WebSocket connections.

**Production Recommendation:** Implement session-based or token-based authentication before WebSocket upgrade.

### Resource Limits

- Maximum message size: 512 KB
- Send buffer size: 256 messages per client
- Events dropped if client buffer full (prevents memory exhaustion)
- Connection count tracking available via `GetActiveConnections()`

## Future Enhancements

1. **Selective Event Subscriptions**
   - Allow clients to subscribe to specific event types
   - Reduce unnecessary event traffic

2. **Connection Compression**
   - Enable permessage-deflate for bandwidth reduction
   - Particularly useful for high-frequency events

3. **Message Batching**
   - Batch multiple events in single WebSocket message
   - Reduce overhead for rapid event sequences

4. **Server-Side Event Filtering**
   - More granular filtering beyond server name
   - Filter by event type, severity, etc.

5. **Connection Metrics Dashboard**
   - Active connection count
   - Event delivery rates
   - Dropped event statistics

## Related Files

- `internal/server/websocket.go` - WebSocket manager implementation
- `internal/server/websocket_test.go` - Unit tests
- `internal/server/websocket_integration_test.go` - Integration tests
- `internal/server/server.go` - Server integration and route registration
- `internal/server/servers_web.go` - Frontend WebSocket client
- `internal/events/bus.go` - Event bus implementation
- `internal/tray/event_handlers.go` - Tray event subscription example

## Migration Notes

The WebSocket implementation is **additive** - existing polling functionality remains as a fallback mechanism. No configuration changes or data migrations required.

Clients automatically detect and use WebSocket connections with graceful fallback to polling if WebSocket is unavailable or fails after reconnection attempts.

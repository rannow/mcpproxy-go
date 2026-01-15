# MCPProxy Event-Driven Architecture

## System Design Principle

**MCPProxy uses event-driven architecture with WebSocket push notifications, NOT polling.**

## Core Pattern

```
Event Occurs → EventBus.Publish() → Subscribers Notified → UI Updates in Real-Time
```

## Key Components

### 1. Event Bus (`internal/events/bus.go`)

Thread-safe pub/sub system for application-wide events.

- **Publishers**: Upstream managers, config loaders, state managers
- **Subscribers**: WebSocket manager, tray UI, log handlers
- **Events**: 11 types including state changes, config updates, tool discoveries

### 2. WebSocket Manager (`internal/server/websocket.go`)

Real-time event broadcaster to web clients.

- **Subscribes to**: All 11 event types from EventBus
- **Broadcasts to**: Connected browser clients via WebSocket
- **Features**: Server filtering, connection lifecycle, health monitoring

### 3. Web Dashboard (`internal/server/servers_web.go`)

Browser client with WebSocket connection.

- **Connects to**: `/ws/events` or `/ws/servers?server=name`
- **Receives**: Real-time events from server
- **Updates**: UI immediately when events occur
- **Fallback**: 30-second polling if WebSocket unavailable

### 4. System Tray (`internal/tray/event_handlers.go`)

Native UI event subscriber.

- **Subscribes to**: State changes, config changes, tool updates
- **Debounces**: Menu updates (100ms) to prevent rapid rebuilds
- **Updates**: Tray menu items based on server state

## Event Flow Example

```
Server Connection State Changes
    ↓
UpstreamManager detects state change
    ↓
EventBus.Publish(ServerStateChanged)
    ↓
┌─────────────────────┬─────────────────────┐
│                     │                     │
WebSocketManager    TrayEventManager    LogHandler
    ↓                   ↓                   ↓
Browser UI Updates  Menu Updates        Log Entry
(instant)           (debounced 100ms)   (immediate)
```

## Event Types

### Server Events
- `ServerStateChanged` - Connection state (Ready, Connecting, Error, Disconnected)
- `ServerConfigChanged` - Configuration updates (enabled, disabled, quarantined)
- `ServerAutoDisabled` - Automatic disabling due to failures
- `ServerGroupUpdated` - Group membership changes

### Application Events
- `AppStateChanged` - Application-wide state changes
- `EventStateChange` - Legacy state change compatibility
- `EventConfigChange` - Legacy config change compatibility

### Tool Events
- `ToolsUpdated` - Server tool discovery/refresh
- `ToolCalled` - Tool execution tracking

### Connection Events
- `ConnectionEstablished` - Server connection established
- `ConnectionLost` - Server connection lost

## Anti-Patterns to Avoid

### ❌ Polling for State Changes

```javascript
// WRONG - Don't do this
setInterval(() => {
    fetch('/api/servers/status')
        .then(updateUI);
}, 1000);  // Wasteful polling
```

### ✅ Event-Driven Updates

```javascript
// CORRECT - Use WebSocket events
ws.onmessage = (event) => {
    const data = JSON.parse(event.data);
    if (data.type === 'server_state_changed') {
        refreshServers();  // Only fetch when needed
    }
};
```

### ❌ Storing State in Multiple Places

```go
// WRONG - Don't maintain separate state
type TrayUI struct {
    serverStates map[string]string  // Duplicate state
}
```

### ✅ Single Source of Truth

```go
// CORRECT - Read from authoritative source
func (t *TrayUI) updateMenu() {
    servers, _ := t.stateManager.GetAllServers()  // Read from storage
    // Update menu based on current state
}
```

## Performance Benefits

### Before (Polling-based)

- **Update Latency**: 1-5 seconds (poll interval)
- **Network Traffic**: Constant, regardless of activity
- **Server Load**: Proportional to client count × poll frequency
- **Scalability**: Poor (N clients = N× requests)

### After (Event-driven)

- **Update Latency**: <100ms (instant event delivery)
- **Network Traffic**: Minimal, only when events occur
- **Server Load**: Event-driven, scales with activity not clients
- **Scalability**: Good (1 event = broadcast to all clients)

**Example Metrics:**
- 10 clients polling every 5s = 120 requests/minute
- 10 clients with WebSocket = 1-2 requests/minute (only events)
- **95%+ reduction in baseline traffic**

## Implementation Guidelines

### For New Features

1. **Publish Events**: When state changes, publish to EventBus
2. **Subscribe**: Have UI components subscribe to relevant events
3. **React**: Update UI in response to events, not on schedule
4. **Single Source**: Always read from authoritative storage/state

### Example: Adding New Server State

```go
// 1. Detect state change
if oldState != newState {
    // 2. Publish event
    s.eventBus.Publish(events.Event{
        Type:       events.ServerStateChanged,
        ServerName: serverName,
        Data: events.StateChangeData{
            OldState: oldState,
            NewState: newState,
        },
    })
}

// 3. Subscribers automatically notified
// - WebSocket clients receive update
// - Tray menu refreshes
// - Logs recorded
```

## Testing Event-Driven Code

### Unit Tests

Test event publishing and subscription:

```go
func TestEventPublishing(t *testing.T) {
    eventBus := events.NewEventBus()
    defer eventBus.Close()

    // Subscribe
    ch := eventBus.Subscribe(events.ServerStateChanged)

    // Publish
    eventBus.Publish(events.Event{
        Type: events.ServerStateChanged,
        ServerName: "test-server",
    })

    // Verify received
    select {
    case event := <-ch:
        assert.Equal(t, "test-server", event.ServerName)
    case <-time.After(1 * time.Second):
        t.Fatal("Event not received")
    }
}
```

### Integration Tests

Test WebSocket event delivery:

```go
func TestWebSocketEventDelivery(t *testing.T) {
    // Create server with WebSocket
    srv, _ := NewServer(cfg, logger)
    srv.StartServer(ctx)

    // Connect WebSocket client
    ws, _, _ := websocket.DefaultDialer.Dial(wsURL, nil)

    // Publish event
    srv.eventBus.Publish(events.Event{
        Type: events.ServerStateChanged,
        ServerName: "test",
    })

    // Verify client receives event
    _, message, _ := ws.ReadMessage()
    var event events.Event
    json.Unmarshal(message, &event)
    assert.Equal(t, events.ServerStateChanged, event.Type)
}
```

## Related Documentation

- [WebSocket Implementation](WEBSOCKET_IMPLEMENTATION.md) - Detailed WebSocket architecture
- [Event Bus](../internal/events/bus.go) - Event bus implementation
- [WebSocket Manager](../internal/server/websocket.go) - WebSocket broadcasting
- [Tray Events](../internal/tray/event_handlers.go) - Native UI event handling

## Key Takeaways

1. **Events, Not Polling**: System uses event-driven architecture throughout
2. **Single Source of Truth**: State lives in storage, not in-memory caches
3. **Real-Time Updates**: WebSocket provides instant UI updates
4. **Scalable**: Event broadcasting scales better than polling
5. **Testable**: Event-driven code is easier to test and verify

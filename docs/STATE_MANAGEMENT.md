# State Management System

## Overview

MCPProxy implements an event-driven state management architecture with real-time WebSocket updates and persistent storage. The system eliminates polling in favor of efficient event-based communication, providing a single source of truth for all server states.

### Key Features

- **Event-Driven Architecture**: Zero polling, all updates via events
- **Real-Time Updates**: WebSocket streaming for live state changes
- **Single Source of Truth**: All state persisted in storage layer
- **State Machine Design**: Valid transitions enforced at runtime
- **Two-Phase Persistence**: Atomic updates across database and config file
- **Auto-Disable Protection**: Prevents flapping servers from degrading system

## Architecture Components

```
┌─────────────────────────────────────────────────────────────┐
│                     Application Layer                        │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │  Tray UI     │  │  Web UI      │  │  MCP Tools   │      │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘      │
│         │                 │                  │              │
│         └─────────────────┴──────────────────┘              │
│                           │                                 │
└───────────────────────────┼─────────────────────────────────┘
                            │
┌───────────────────────────┼─────────────────────────────────┐
│                     Event Bus Layer                          │
│         ┌─────────────────┴─────────────────┐               │
│         │   EventBus (Pub/Sub)              │               │
│         │   - 11 Event Types                │               │
│         │   - Buffered Channels             │               │
│         │   - Non-blocking Publish          │               │
│         └────────┬──────────────────────────┘               │
│                  │                                           │
└──────────────────┼───────────────────────────────────────────┘
                   │
┌──────────────────┼───────────────────────────────────────────┐
│            State Management Layer                            │
│  ┌───────────────┴──────────────┐  ┌────────────────────┐  │
│  │  AppStateMachine             │  │  ServerStateMachine │  │
│  │  - Application States        │  │  - Per-Server States│  │
│  │  - Valid Transitions         │  │  - Auto-Disable     │  │
│  │  - Health Checks             │  │  - Failure Tracking │  │
│  └───────────────┬──────────────┘  └────────┬───────────┘  │
│                  │                           │              │
└──────────────────┼───────────────────────────┼──────────────┘
                   │                           │
┌──────────────────┼───────────────────────────┼──────────────┐
│                Storage Layer (Single Source of Truth)        │
│  ┌──────────────┴───────────────────────────┴───────────┐  │
│  │  Two-Phase Commit:                                    │  │
│  │  1. BBolt Database (config.db)                        │  │
│  │  2. Configuration File (mcp_config.json)              │  │
│  │     - Automatic Rollback on Failure                   │  │
│  └───────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

## Startup Modes

MCPProxy uses a unified `startup_mode` field to replace multiple boolean flags. This simplifies configuration and prevents conflicting states.

### Mode Definitions

| Mode | Value | Description | Startup Behavior |
|------|-------|-------------|------------------|
| **Active** | `active` | Server starts on boot, auto-reconnects on failure | Immediate connection on mcpproxy start |
| **Lazy Loading** | `lazy_loading` | Server enabled but waits for first tool call | Connection deferred until needed |
| **Disabled** | `disabled` | Server disabled by user | No connection, tools unavailable |
| **Quarantined** | `quarantined` | Server in security quarantine | No tool execution, security analysis required |
| **Auto-Disabled** | `auto_disabled` | Automatically disabled after repeated failures | No connection until manually cleared |

### Startup Mode Configuration

**JSON Configuration** (`mcp_config.json`):
```json
{
  "mcpServers": [
    {
      "name": "example-server",
      "startup_mode": "active",
      "url": "http://localhost:3000"
    },
    {
      "name": "lazy-server",
      "startup_mode": "lazy_loading",
      "command": "npx",
      "args": ["mcp-server"]
    }
  ]
}
```

### Legacy Compatibility

The system maintains backward compatibility with legacy boolean flags:

**Legacy Fields** (DEPRECATED):
- `enabled` → Use `startup_mode: "active"` or `"disabled"`
- `quarantined` → Use `startup_mode: "quarantined"`
- `start_on_boot` → Use `startup_mode: "active"` or `"lazy_loading"`
- `auto_disabled` → Use `startup_mode: "auto_disabled"`

**Migration Logic**:
```go
// Automatic migration on config load
if server.StartupMode == "" {
    if server.Quarantined {
        server.StartupMode = "quarantined"
    } else if server.AutoDisabled {
        server.StartupMode = "auto_disabled"
    } else if server.Enabled {
        server.StartupMode = "active"
    } else {
        server.StartupMode = "disabled"
    }
}
```

## State Machine

### Server State Transitions

The state machine enforces valid transitions to prevent invalid states.

```
┌─────────────┐
│   Active    │◄──────────────┐
│ (Starting   │               │
│ on boot)    │               │
└──────┬──────┘               │
       │                      │
       │ Failure × 3          │ Manual
       │                      │ Enable
       ▼                      │
┌─────────────┐               │
│Auto-Disabled│───────────────┘
│ (Blocked    │
│ until clear)│
└──────┬──────┘
       │
       │ Manual
       │ Disable
       ▼
┌─────────────┐     Manual      ┌─────────────┐
│  Disabled   │◄───────────────►│ Quarantined │
│ (User       │     Quarantine  │ (Security   │
│ config)     │                 │ block)      │
└─────────────┘                 └─────────────┘
       ▲
       │
       │ User
       │ Enable
       │
┌──────┴──────┐
│Lazy Loading │
│ (Deferred   │
│ start)      │
└─────────────┘
```

### Valid Transitions Table

| From State | To States | Conditions |
|-----------|-----------|------------|
| `active` | `disabled`, `quarantined`, `auto_disabled`, `lazy_loading` | Any time |
| `disabled` | `active`, `lazy_loading`, `quarantined` | User action |
| `quarantined` | `active`, `disabled` | Manual approval required |
| `auto_disabled` | `active`, `disabled` | Must clear auto-disable state |
| `lazy_loading` | `active`, `disabled`, `quarantined`, `auto_disabled` | User action or failure |

### Auto-Disable Logic

**Threshold Configuration**:
```json
{
  "auto_disable_threshold": 3,  // Global default
  "mcpServers": [
    {
      "name": "flaky-server",
      "auto_disable_threshold": 5,  // Per-server override
      "startup_mode": "active"
    }
  ]
}
```

**Auto-Disable Sequence**:
1. Server connection fails
2. Increment consecutive failure counter
3. If counter >= threshold (default: 3):
   - Transition to `auto_disabled` state
   - Persist state to disk (two-phase commit)
   - Emit `ServerAutoDisabled` event
   - Reset failure counter

**Clearing Auto-Disable**:
```bash
# Via MCP tool
mcpproxy call tool --tool-name=upstream_servers \
  --json_args='{"operation":"patch","name":"server-name","patch_json":"{\"startup_mode\":\"active\"}"}'

# Or via group operations (clears all auto-disabled servers in group)
mcpproxy call tool --tool-name=groups \
  --json_args='{"operation":"enable_group","group_name":"Production"}'
```

## Event System

### EventBus Architecture

The EventBus implements a non-blocking publish/subscribe pattern with buffered channels.

**Key Features**:
- **Buffered Channels**: 100 events per subscriber (prevents blocking)
- **Non-Blocking Publish**: Drops events if subscriber buffer full
- **Thread-Safe**: Concurrent publishers and subscribers supported
- **Auto-Cleanup**: Closed subscribers automatically removed

### Event Types (11 Total)

| Event Type | Trigger | Data Structure |
|-----------|---------|----------------|
| `server_state_changed` | Server state transition | `{server_name, old_state, new_state}` |
| `server_config_changed` | Server configuration update | `{server_name, action: created/updated/deleted}` |
| `server_auto_disabled` | Auto-disable triggered | `{server_name, reason, threshold}` |
| `server_group_updated` | Group membership change | `{group_id, server_names[], action}` |
| `state_change` | Legacy compatibility | Same as `server_state_changed` |
| `app_state_changed` | Application state change | `{old_state, new_state}` |
| `app_state_change` | Alias for consistency | Same as `app_state_changed` |
| `config_change` | Config file reload | `{action: reloaded}` |
| `tools_updated` | Tool index rebuild | `{server_name, tool_count}` |
| `tool_called` | Tool execution | `{tool_name, server_name, duration}` |
| `connection_established` | Server connected | `{server_name, timestamp}` |
| `connection_lost` | Server disconnected | `{server_name, error, timestamp}` |

### Event Data Structures

```go
// Generic event envelope
type Event struct {
    Type       EventType   `json:"type"`
    ServerName string      `json:"server_name,omitempty"`
    OldState   string      `json:"old_state,omitempty"`
    NewState   string      `json:"new_state,omitempty"`
    Timestamp  time.Time   `json:"timestamp"`
    Data       interface{} `json:"data,omitempty"`
}

// State change events
type StateChangeData struct {
    ServerName string      `json:"server_name,omitempty"`
    OldState   interface{} `json:"old_state"`
    NewState   interface{} `json:"new_state"`
    Info       interface{} `json:"info,omitempty"`
}

// App state events
type AppStateChangeData struct {
    OldState string `json:"old_state"`
    NewState string `json:"new_state"`
}

// Config change events
type ConfigChangeData struct {
    Action string `json:"action"` // "created", "updated", "deleted"
}
```

### Subscribing to Events (Go)

```go
// Subscribe to specific event type
eventChan := eventBus.Subscribe(events.ServerStateChanged)
defer eventBus.Unsubscribe(events.ServerStateChanged, eventChan)

for event := range eventChan {
    fmt.Printf("Server %s: %s → %s\n",
        event.ServerName, event.OldState, event.NewState)
}

// Subscribe to all events
allEventsChan := eventBus.SubscribeAll()
defer eventBus.Unsubscribe(events.EventType("*"), allEventsChan)

for event := range allEventsChan {
    // Process any event type
    fmt.Printf("Event: %s\n", event.Type)
}
```

## WebSocket API

### Endpoints

**1. All Events Stream**
```
GET /ws/events
```
Streams all event types to the client.

**2. Server-Filtered Stream**
```
GET /ws/servers?server=<server-name>
```
Streams events for a specific server only.

### Event Format (JSON)

```json
{
  "type": "server_state_changed",
  "server_name": "example-server",
  "old_state": "active",
  "new_state": "auto_disabled",
  "timestamp": "2025-11-15T17:30:00Z",
  "data": {
    "server_name": "example-server",
    "old_state": "active",
    "new_state": "auto_disabled",
    "info": {
      "reason": "connection_failures",
      "threshold": 3
    }
  }
}
```

### JavaScript Client Example

```javascript
// Connect to WebSocket
const ws = new WebSocket('ws://localhost:8080/ws/events');

ws.onopen = () => {
    console.log('Connected to event stream');
};

ws.onmessage = (event) => {
    const data = JSON.parse(event.data);

    switch (data.type) {
        case 'server_state_changed':
            console.log(`Server ${data.server_name}: ${data.old_state} → ${data.new_state}`);
            updateServerUI(data.server_name, data.new_state);
            break;

        case 'server_auto_disabled':
            console.warn(`Server ${data.server_name} auto-disabled`);
            showAutoDisableNotification(data.server_name);
            break;

        case 'app_state_changed':
            console.log(`App state: ${data.data.old_state} → ${data.data.new_state}`);
            updateAppStatus(data.data.new_state);
            break;
    }
};

ws.onerror = (error) => {
    console.error('WebSocket error:', error);
};

ws.onclose = () => {
    console.log('Disconnected from event stream');
    // Implement reconnection logic
    setTimeout(() => reconnect(), 5000);
};

// Helper functions
function updateServerUI(serverName, newState) {
    const serverElement = document.getElementById(`server-${serverName}`);
    if (serverElement) {
        serverElement.className = `server-status ${newState}`;
        serverElement.textContent = newState;
    }
}

function showAutoDisableNotification(serverName) {
    // Show toast notification
    showToast(`Server ${serverName} has been auto-disabled due to repeated failures`, 'warning');
}

function updateAppStatus(newState) {
    document.getElementById('app-status').textContent = newState;
}
```

### Reconnection Logic

```javascript
class WebSocketManager {
    constructor(url) {
        this.url = url;
        this.ws = null;
        this.reconnectDelay = 1000;
        this.maxReconnectDelay = 30000;
    }

    connect() {
        this.ws = new WebSocket(this.url);

        this.ws.onopen = () => {
            console.log('WebSocket connected');
            this.reconnectDelay = 1000; // Reset delay on successful connect
        };

        this.ws.onmessage = (event) => {
            this.handleMessage(JSON.parse(event.data));
        };

        this.ws.onerror = (error) => {
            console.error('WebSocket error:', error);
        };

        this.ws.onclose = () => {
            console.log('WebSocket closed, reconnecting...');
            this.reconnect();
        };
    }

    reconnect() {
        setTimeout(() => {
            this.connect();
            // Exponential backoff
            this.reconnectDelay = Math.min(this.reconnectDelay * 2, this.maxReconnectDelay);
        }, this.reconnectDelay);
    }

    handleMessage(data) {
        // Dispatch to appropriate handler
        const handler = this.handlers[data.type];
        if (handler) {
            handler(data);
        }
    }
}

// Usage
const wsManager = new WebSocketManager('ws://localhost:8080/ws/events');
wsManager.connect();
```

## Storage & Persistence

### Two-Phase Commit

All state changes use a two-phase commit to ensure consistency between the database and configuration file.

**Phase 1: Database Update**
```go
// Update BBolt database
if err := storage.UpdateServerStartupMode(name, newMode, reason); err != nil {
    return fmt.Errorf("database update failed: %w", err)
}
```

**Phase 2: Config File Update**
```go
// Update mcp_config.json
if err := config.SaveToFile(); err != nil {
    // ROLLBACK: Revert database changes
    storage.UpdateServerStartupMode(name, oldMode, "rollback")
    return fmt.Errorf("config file update failed, rolled back: %w", err)
}
```

### Automatic Rollback

If the config file write fails, the database changes are automatically rolled back:

```go
func (m *Manager) UpdateServerState(name string, newState types.ServerState) error {
    // Get current state (for rollback)
    oldState := m.GetServerState(name)

    // Phase 1: Update database
    if err := m.storage.UpdateServerStartupMode(name, string(newState), ""); err != nil {
        return err
    }

    // Phase 2: Update config file
    if err := m.config.UpdateServerMode(name, string(newState)); err != nil {
        // Rollback database
        m.storage.UpdateServerStartupMode(name, string(oldState), "rollback")
        return fmt.Errorf("config update failed, rolled back: %w", err)
    }

    return nil
}
```

### Transaction Safety

**Guarantees**:
- Atomic updates across database and config file
- Automatic rollback on failure
- No partial state changes
- Event emission only after successful commit

**Error Handling**:
```go
// Example: Enable server with auto-disable clear
func EnableServer(name string) error {
    // Start transaction
    tx := storage.Begin()
    defer tx.Rollback() // Auto-rollback if not committed

    // Clear auto-disable
    if err := tx.ClearAutoDisable(name); err != nil {
        return err
    }

    // Update startup mode
    if err := tx.UpdateStartupMode(name, "active"); err != nil {
        return err
    }

    // Update config file
    if err := config.UpdateServerMode(name, "active"); err != nil {
        return err // Rollback triggered
    }

    // Commit transaction
    return tx.Commit()
}
```

### ClearAutoDisable Operation

**Purpose**: Clears auto-disabled state and allows server to reconnect.

**Usage**:
```go
// Via storage API
if err := storage.ClearAutoDisable(serverName); err != nil {
    return fmt.Errorf("failed to clear auto-disable: %w", err)
}

// Automatically called when:
// 1. User manually enables server
// 2. Group enable operation
// 3. Server configuration updated
```

**Implementation**:
```go
func (s *Manager) ClearAutoDisable(name string) error {
    // Remove auto-disable reason
    // Set startup_mode to "active"
    // Emit ServerConfigChanged event
    return s.UpdateServerStartupMode(name, "active", "")
}
```

## Group Operations

### Enable Group (Clears Auto-Disable)

When enabling a group, all auto-disabled servers within the group are automatically cleared:

```go
func (m *Manager) EnableGroup(groupID int) error {
    servers := m.GetServersInGroup(groupID)

    for _, server := range servers {
        // Check if auto-disabled
        if server.StartupMode == "auto_disabled" {
            // Clear auto-disable state
            if err := m.storage.ClearAutoDisable(server.Name); err != nil {
                return fmt.Errorf("failed to clear auto-disable for %s: %w", server.Name, err)
            }
        }

        // Enable server
        if err := m.EnableServer(server.Name); err != nil {
            return fmt.Errorf("failed to enable %s: %w", server.Name, err)
        }
    }

    // Emit group updated event
    m.eventBus.Publish(events.Event{
        Type: events.ServerGroupUpdated,
        Data: map[string]interface{}{
            "group_id": groupID,
            "action":   "enabled",
        },
    })

    return nil
}
```

### Disable Group (Stops All Servers)

```go
func (m *Manager) DisableGroup(groupID int) error {
    servers := m.GetServersInGroup(groupID)

    for _, server := range servers {
        // Set startup_mode to disabled
        if err := m.storage.UpdateServerStartupMode(server.Name, "disabled", ""); err != nil {
            return fmt.Errorf("failed to disable %s: %w", server.Name, err)
        }

        // Stop server if running
        if err := m.StopServer(server.Name); err != nil {
            return fmt.Errorf("failed to stop %s: %w", server.Name, err)
        }
    }

    // Emit group updated event
    m.eventBus.Publish(events.Event{
        Type: events.ServerGroupUpdated,
        Data: map[string]interface{}{
            "group_id": groupID,
            "action":   "disabled",
        },
    })

    return nil
}
```

### Partial Failure Handling

Group operations are designed to be resilient:

```go
func (m *Manager) EnableGroup(groupID int) error {
    servers := m.GetServersInGroup(groupID)
    errors := make([]error, 0)

    for _, server := range servers {
        if err := m.EnableServer(server.Name); err != nil {
            // Log error but continue
            m.logger.Error("Failed to enable server in group",
                zap.String("server", server.Name),
                zap.Error(err))
            errors = append(errors, err)
        }
    }

    // Return combined errors if any
    if len(errors) > 0 {
        return fmt.Errorf("group enable completed with %d errors", len(errors))
    }

    return nil
}
```

## Troubleshooting

### Auto-Disable Not Persisting

**Symptoms**:
- Server auto-disables but returns to `active` after restart
- Auto-disable state not visible in config file

**Diagnosis**:
```bash
# Check database state
sqlite3 ~/.mcpproxy/config.db "SELECT name, startup_mode, auto_disable_reason FROM servers;"

# Check config file
cat ~/.mcpproxy/mcp_config.json | jq '.mcpServers[] | select(.name == "server-name")'
```

**Solutions**:
1. Verify two-phase commit completed:
   ```bash
   # Check logs for rollback messages
   grep "rollback" ~/.mcpproxy/logs/main.log
   ```

2. Check file permissions:
   ```bash
   ls -la ~/.mcpproxy/mcp_config.json
   # Should be writable by user
   ```

3. Verify storage operations:
   ```go
   // Enable debug logging
   "logging": {
       "level": "debug",
       "enable_file": true
   }
   ```

### Events Not Firing

**Symptoms**:
- UI not updating in real-time
- WebSocket receives no messages

**Diagnosis**:
```bash
# Check WebSocket connections
netstat -an | grep 8080

# Check event bus status
curl http://localhost:8080/api/events/stats
```

**Solutions**:
1. Verify WebSocket connection:
   ```javascript
   console.log('WebSocket state:', ws.readyState);
   // 0 = CONNECTING, 1 = OPEN, 2 = CLOSING, 3 = CLOSED
   ```

2. Check event bus subscribers:
   ```go
   fmt.Printf("Subscribers: %d\n", eventBus.TotalSubscribers())
   ```

3. Enable event logging:
   ```go
   eventBus.Subscribe(events.ServerStateChanged)
   for event := range eventChan {
       log.Printf("Event: %+v", event)
   }
   ```

### WebSocket Connection Issues

**Symptoms**:
- WebSocket fails to connect
- Frequent disconnections

**Solutions**:
1. Check CORS settings:
   ```go
   upgrader := websocket.Upgrader{
       CheckOrigin: func(r *http.Request) bool {
           return true // Or implement proper CORS check
       },
   }
   ```

2. Verify endpoint:
   ```bash
   # Test WebSocket endpoint
   wscat -c ws://localhost:8080/ws/events
   ```

3. Check buffer sizes:
   ```go
   // Increase buffer if events are being dropped
   ch := make(chan Event, 500) // Default is 100
   ```

### State Desync Between Database and Config

**Symptoms**:
- Database shows different state than config file
- Inconsistent behavior after restart

**Diagnosis**:
```bash
# Compare states
echo "Database state:"
sqlite3 ~/.mcpproxy/config.db "SELECT name, startup_mode FROM servers;"

echo "Config file state:"
cat ~/.mcpproxy/mcp_config.json | jq '.mcpServers[] | {name, startup_mode}'
```

**Solutions**:
1. Force config reload:
   ```bash
   # Trigger file watcher by touching config
   touch ~/.mcpproxy/mcp_config.json
   ```

2. Manual sync:
   ```go
   // Re-save config from database
   config := m.config.Clone()
   if err := config.SaveToFile(); err != nil {
       log.Fatal(err)
   }
   ```

3. Repair from database (authoritative):
   ```bash
   # Export database to config
   mcpproxy tools export-config
   ```

## Best Practices

### Event Handling

1. **Always unsubscribe** when done to prevent memory leaks:
   ```go
   eventChan := eventBus.Subscribe(eventType)
   defer eventBus.Unsubscribe(eventType, eventChan)
   ```

2. **Use buffered channels** to prevent blocking publishers:
   ```go
   // EventBus uses 100-event buffer by default
   // Increase if processing is slow
   ```

3. **Handle dropped events** gracefully:
   ```go
   select {
   case event := <-eventChan:
       processEvent(event)
   default:
       // Channel empty, no event available
   }
   ```

### State Management

1. **Always check transition validity**:
   ```go
   if !stateMachine.CanTransitionTo(newState) {
       return fmt.Errorf("invalid transition: %s → %s", currentState, newState)
   }
   ```

2. **Use two-phase commit** for all state changes:
   ```go
   // Update database first
   storage.Update()
   // Then config file
   config.Save()
   // Rollback database if config save fails
   ```

3. **Emit events after commit**:
   ```go
   // Persist state first
   if err := storage.UpdateState(); err != nil {
       return err
   }
   // Then emit event
   eventBus.Publish(event)
   ```

### WebSocket Clients

1. **Implement reconnection logic**:
   ```javascript
   function reconnect() {
       setTimeout(() => connect(), Math.min(delay * 2, 30000));
   }
   ```

2. **Use heartbeat/ping to detect disconnects**:
   ```javascript
   setInterval(() => {
       if (ws.readyState === WebSocket.OPEN) {
           ws.send(JSON.stringify({type: 'ping'}));
       }
   }, 30000);
   ```

3. **Buffer events during reconnection**:
   ```javascript
   const eventBuffer = [];
   ws.onclose = () => {
       bufferEvents = true;
   };
   ws.onopen = () => {
       bufferEvents = false;
       eventBuffer.forEach(processEvent);
       eventBuffer.length = 0;
   };
   ```

## Related Documentation

- [ARCHITECTURE_WEBSOCKET.md](./ARCHITECTURE_WEBSOCKET.md) - WebSocket implementation details
- [STATE_MANAGEMENT_REFACTOR.md](./STATE_MANAGEMENT_REFACTOR.md) - Refactor planning document
- [STATE_MANAGEMENT_TASKS.md](./STATE_MANAGEMENT_TASKS.md) - Implementation task breakdown
- [CLAUDE.md](../CLAUDE.md) - Development guide with state management section

# State Management System Refactor - Comprehensive Plan

## Executive Summary

This document describes a comprehensive refactor of mcpproxy-go's server state management system to:
1. **Separate configuration flags from runtime states** for clearer semantics
2. **Introduce application-level states** (Starting, Running, Stopping, Stopped)
3. **Implement event-driven architecture** replacing polling with real-time updates
4. **Fix auto-disable persistence** ensuring state survives restarts
5. **Improve group operations** to properly clear auto-disabled servers

## Current Architecture Problems

### Problem 1: Mixed Configuration and State
**Current**: Multiple boolean flags serve dual purposes (config + state)
- `Enabled bool` - User wants server active (config) OR server is currently active (state)
- `Quarantined bool` - Security flag (config) AND runtime status (state)
- `AutoDisabled bool` - Was auto-disabled (config) AND is currently auto-disabled (state)
- `StartOnBoot bool` - Startup behavior (config)
- `Stopped bool` - Temporary stop (runtime-only, but not clearly separated)

**Impact**:
- Unclear when to persist vs when to keep in-memory
- File watcher conflicts during programmatic updates
- Group operations don't know which flags to update
- Restart behavior is unpredictable

### Problem 2: No Application-Level State
**Current**: Only server-level states exist (Ready, Connecting, Error, etc.)

**Missing**: Overall application state indicating:
- Are all servers starting up? (Starting)
- Are all servers in stable states? (Running)
- User clicked "Stop All"? (Stopping → Stopped)

**Impact**:
- Tray UI doesn't know if startup is complete
- No way to coordinate "Stop All" operation
- Can't distinguish "startup in progress" from "servers failing"

### Problem 3: Polling-Based UI Updates
**Current**:
- Tray polls server state every X seconds
- Web interface requires manual refresh
- File watcher triggers config reloads (race conditions)

**Impact**:
- UI lag (up to polling interval)
- Unnecessary resource usage
- State synchronization issues
- File watcher conflicts with programmatic updates

### Problem 4: Auto-Disable Persistence Issues
**Current**:
- Auto-disable state tracked in multiple places (StateManager, config, database)
- No clear synchronization mechanism
- Group enable doesn't properly clear auto-disabled state (recent bug fixes: 0d6cc38, 60d739b)

**Impact**:
- Auto-disable state sometimes lost on restart
- Group operations leave servers auto-disabled
- Inconsistent state across storage layers

## Proposed Architecture

### 1. Configuration Schema Changes

#### 1.1 Replace Multiple Flags with `startup_mode`

**Remove these flags from `ServerConfig`**:
```go
// OLD (to be removed)
Enabled       bool   // Replaced by startup_mode
Quarantined   bool   // Replaced by startup_mode
StartOnBoot   bool   // Replaced by startup_mode
AutoDisabled  bool   // Replaced by startup_mode
```

**Add new field**:
```go
// NEW
StartupMode string  // "active", "disabled", "quarantined", "auto_disabled", "lazy_loading"
```

**Startup Mode Values**:
- `"active"` - Server should start on boot and connect immediately
- `"disabled"` - User has disabled this server (won't start)
- `"quarantined"` - Security quarantine (won't start, requires manual approval)
- `"auto_disabled"` - System disabled due to failures (can be cleared by user)
- `"lazy_loading"` - Don't connect on startup, wait for tool request

**Migration Logic** (backward compatibility):
```go
func MigrateToStartupMode(config *ServerConfig) string {
    if config.Quarantined {
        return "quarantined"
    }
    if config.AutoDisabled {
        return "auto_disabled"
    }
    if !config.Enabled {
        return "disabled"
    }
    if config.StartOnBoot {
        return "active"
    }
    return "lazy_loading"  // Default for enabled but not start-on-boot
}
```

#### 1.2 Keep These Configuration Fields
```go
// Persistence flags (kept)
AutoDisableReason    string  // Why was server auto-disabled
AutoDisableThreshold int     // Per-server failure threshold override

// Runtime-only flags (NOT persisted to config file)
Stopped              bool    // Temporary user stop via tray (in-memory only)
```

### 2. Runtime State System

#### 2.1 Server Runtime States (StateManager)

**New State Enum** (`internal/upstream/types/types.go`):
```go
type ServerState string

const (
    // Configuration-driven states
    StateDisabled      ServerState = "disabled"       // startup_mode="disabled"
    StateAutoDisabled  ServerState = "auto_disabled"  // startup_mode="auto_disabled"
    StateQuarantined   ServerState = "quarantined"    // startup_mode="quarantined"

    // Runtime states
    StateStopped       ServerState = "stopped"        // User clicked stop
    StateSleeping      ServerState = "sleeping"       // Lazy loading, tools cached
    StateConnected     ServerState = "connected"      // Active MCP connection
    StateDisconnected  ServerState = "disconnected"   // Attempting connection
    StateConnecting    ServerState = "connecting"     // In connection process
    StateAuthenticating ServerState = "authenticating" // OAuth flow
    StateDiscovering   ServerState = "discovering"    // Fetching tools
    StateError         ServerState = "error"          // Connection failed
)
```

**StateManager Fields** (`internal/upstream/types/types.go`):
```go
type StateManager struct {
    mu                  sync.RWMutex
    currentState        ServerState    // Current runtime state
    consecutiveFailures int            // Failure counter for auto-disable
    lastSuccessTime     time.Time      // Last successful connection
    lastError           error          // Most recent error
    eventBus            *events.Bus    // NEW: Event bus for notifications
}
```

#### 2.2 Application-Level States

**New App State Enum** (`internal/server/server.go`):
```go
type AppState string

const (
    AppStateStarting  AppState = "starting"  // Servers transitioning to target states
    AppStateRunning   AppState = "running"   // All servers in stable states
    AppStateStopping  AppState = "stopping"  // User requested stop, servers shutting down
    AppStateStopped   AppState = "stopped"   // All servers stopped
)
```

**Server struct additions**:
```go
type Server struct {
    // ... existing fields ...

    // NEW: Application state management
    appState      AppState
    appStateMu    sync.RWMutex
    eventBus      *events.Bus  // Event bus for state changes
}
```

**Stability Check Logic**:
```go
// Server is considered "stable" if in one of these states
func (s ServerState) IsStable() bool {
    switch s {
    case StateConnected, StateQuarantined, StateAutoDisabled,
         StateSleeping, StateDisabled, StateStopped:
        return true
    default:
        return false
    }
}

// App transitions to Running when ALL servers are stable
func (s *Server) updateAppState() {
    allStable := true
    for _, upstream := range s.upstreams {
        if !upstream.GetState().IsStable() {
            allStable = false
            break
        }
    }

    if allStable && s.appState == AppStateStarting {
        s.setAppState(AppStateRunning)
    }
}
```

### 3. Event-Driven Architecture

#### 3.1 Internal Event Bus

**New Package**: `internal/events/bus.go`

**Event Types**:
```go
type EventType string

const (
    ServerStateChanged  EventType = "server_state_changed"
    ServerConfigChanged EventType = "server_config_changed"
    AppStateChanged     EventType = "app_state_changed"
    ServerAutoDisabled  EventType = "server_auto_disabled"
    ServerGroupUpdated  EventType = "server_group_updated"
    ToolsUpdated        EventType = "tools_updated"
)
```

**Event Structure**:
```go
type Event struct {
    Type       EventType              `json:"type"`
    ServerName string                 `json:"server_name,omitempty"`
    OldState   string                 `json:"old_state,omitempty"`
    NewState   string                 `json:"new_state,omitempty"`
    Timestamp  time.Time              `json:"timestamp"`
    Data       map[string]interface{} `json:"data,omitempty"`
}
```

**Event Bus Implementation**:
```go
type Bus struct {
    mu          sync.RWMutex
    subscribers map[EventType][]chan Event
}

func NewBus() *Bus {
    return &Bus{
        subscribers: make(map[EventType][]chan Event),
    }
}

func (b *Bus) Subscribe(eventType EventType) <-chan Event {
    b.mu.Lock()
    defer b.mu.Unlock()

    ch := make(chan Event, 100) // Buffered to prevent blocking
    b.subscribers[eventType] = append(b.subscribers[eventType], ch)
    return ch
}

func (b *Bus) Publish(event Event) {
    b.mu.RLock()
    defer b.mu.RUnlock()

    event.Timestamp = time.Now()
    for _, ch := range b.subscribers[event.Type] {
        select {
        case ch <- event:
        default:
            // Drop event if channel is full (prevent blocking)
        }
    }
}
```

#### 3.2 WebSocket Integration

**New File**: `internal/server/websocket.go`

**Endpoints**:
- `GET /ws/events` - Subscribe to all events
- `GET /ws/servers` - Subscribe to server state changes only

**Implementation**:
```go
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
    conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        return
    }
    defer conn.Close()

    // Subscribe to events
    eventChan := s.eventBus.Subscribe(ServerStateChanged)
    defer close(eventChan)

    for event := range eventChan {
        if err := conn.WriteJSON(event); err != nil {
            return
        }
    }
}
```

## Success Criteria

✅ Single `startup_mode` field replaces 4 boolean flags
✅ Auto-disable state persists and survives restart
✅ Group enable operation clears auto-disable for all servers
✅ Tray shows app state (Starting/Running/Stopping/Stopped)
✅ WebSocket delivers real-time state updates
✅ No file watcher conflicts during programmatic updates
✅ All existing configs migrate automatically
✅ Backward compatibility with old API maintained
✅ Zero polling in tray UI (100% event-driven)
✅ All unit/integration/E2E tests passing
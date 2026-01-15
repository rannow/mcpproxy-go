# State Flow: Config → Runtime → AppState

## Overview

This document explains how `startup_mode` (persisted in config) translates to runtime `ConnectionState` (in-memory) and how the overall `AppState` aggregates all server states.

## Three-Tier State System

```
┌──────────────────────────────────────────────────────────────┐
│                   PERSISTENT LAYER                           │
│                  (Config File + Database)                    │
│                                                              │
│  startup_mode: "active" | "lazy_loading" | "disabled" |     │
│                "quarantined" | "auto_disabled"               │
│                                                              │
│  Files: ~/.mcpproxy/mcp_config.json                         │
│         ~/.mcpproxy/config.db (BBolt)                       │
└──────────────────────────────────────────────────────────────┘
                          │
                          │ Loads at startup
                          │ (server.go:630-688)
                          ↓
┌──────────────────────────────────────────────────────────────┐
│                   IN-MEMORY RUNTIME LAYER                    │
│                  (Per-Server StateManager)                   │
│                                                              │
│  ConnectionState: Disconnected | Connecting |                │
│                   Authenticating | Discovering |             │
│                   Ready | Error                              │
│                                                              │
│  ServerState: Mirrors startup_mode for state management     │
│                                                              │
│  Location: internal/upstream/types/types.go:163-192         │
└──────────────────────────────────────────────────────────────┘
                          │
                          │ Aggregated by
                          │ (app_state_machine.go:179-237)
                          ↓
┌──────────────────────────────────────────────────────────────┐
│                   APPLICATION STATE LAYER                    │
│                  (Overall Health Summary)                    │
│                                                              │
│  AppState: starting | running | degraded |                   │
│            stopping | stopped                                │
│                                                              │
│  Logic: All enabled servers ready → running                 │
│         Some servers in error → degraded                     │
│         Shutdown initiated → stopping → stopped              │
│                                                              │
│  Location: internal/server/app_state_machine.go             │
└──────────────────────────────────────────────────────────────┘
```

## Startup Flow (Config → Runtime)

### Step 1: Load Configuration (server.go:630-688)

```go
// For each server in config.Servers:
for i := range s.config.Servers {
    serverCfg := s.config.Servers[i]

    // Step 1a: Persist to database
    s.storageManager.SaveUpstreamServer(serverCfg)

    // Step 1b: Add to upstream manager (creates runtime state)
    s.upstreamManager.AddServer(serverCfg.Name, serverCfg)
}
```

**Key Points**:
- All servers are added to upstream manager, even disabled ones
- startup_mode determines whether connection is attempted
- Disabled/quarantined servers tracked but not connected

### Step 2: Upstream Manager Creates Runtime State (manager.go:270-320)

```go
func (m *Manager) AddServer(id string, serverConfig *config.ServerConfig) error {
    // Create client with StateManager
    m.AddServerConfig(id, serverConfig)

    // Check startup_mode and decide connection behavior
    if serverConfig.StartupMode == "disabled" {
        return nil  // Don't connect
    }

    if serverConfig.StartupMode == "quarantined" {
        return nil  // Don't connect
    }

    if serverConfig.StartupMode == "auto_disabled" {
        return nil  // Don't connect
    }

    // For "active" and "lazy_loading", attempt connection
    client, _ := m.GetClient(id)
    client.Connect(ctx)  // Sets ConnectionState transitions
}
```

**State Mapping**:

| startup_mode | Initial ConnectionState | Connection Attempted? |
|--------------|-------------------------|----------------------|
| `active` | Disconnected → Connecting | ✅ Yes |
| `lazy_loading` | Disconnected (server_state="stopped" in DB) | ❌ No (waits for tool call) |
| `disabled` | Disconnected | ❌ No |
| `quarantined` | Disconnected | ❌ No |
| `auto_disabled` | Disconnected | ❌ No |

### Step 3: StateManager Manages Runtime State (types.go:163-192)

```go
type StateManager struct {
    // Runtime connection state (NOT persisted)
    currentState ConnectionState  // Disconnected, Connecting, Ready, etc.

    // Configuration state (mirrors startup_mode)
    serverState ServerState       // active, disabled, quarantined, etc.

    // Auto-disable tracking
    consecutiveFailures int
    autoDisabled bool
}
```

**Connection State Transitions**:
```
startup_mode="active":
  Disconnected → Connecting → Authenticating → Discovering → Ready
                     ↓
                   Error (retry with backoff)

startup_mode="lazy_loading":
  Disconnected (DB server_state="stopped") → (tool call) → Connecting → Ready
```

**Failure Handling**:
```
Ready → Error → Error → Error (3x failures)
    ↓
StateManager.consecutiveFailures == 3
    ↓
Callback: updateServerConfigEnabled()
    ↓
startup_mode changed to "auto_disabled" in config
    ↓
Connection stopped, state persisted
```

## App State Aggregation (AppStateMachine)

### CheckServerStability Logic (app_state_machine.go:179-237)

```go
func (asm *AppStateMachine) CheckServerStability() AppState {
    clients := asm.upstreamManager.GetAllClients()

    // Count servers by state
    enabledCount := 0    // startup_mode != "disabled" && != "quarantined"
    readyCount := 0      // ConnectionState == Ready
    errorCount := 0      // ConnectionState == Error
    stableCount := 0     // ConnectionState == Ready && ServerState.IsStable()

    for _, client := range clients {
        // Skip disabled and quarantined servers
        if client.Config.StartupMode == "disabled" ||
           client.Config.StartupMode == "quarantined" {
            continue
        }

        enabledCount++

        state := client.GetState()  // ConnectionState
        serverState := client.StateManager.GetServerState()

        switch state {
        case types.StateReady:
            readyCount++
            if serverState.IsStable() {  // active, lazy_loading (NOT auto_disabled)
                stableCount++
            }
        case types.StateError:
            errorCount++
        }
    }

    // Decision logic
    if stableCount == enabledCount {
        return AppStateRunning     // All enabled servers stable
    }

    if errorCount > 0 || readyCount < enabledCount {
        return AppStateDegraded    // Some servers in error
    }

    return AppStateRunning
}
```

### AppState Determination Examples

**Example 1: All Active Servers Connected**
```
Server A: startup_mode="active", ConnectionState=Ready
Server B: startup_mode="active", ConnectionState=Ready
Server C: startup_mode="disabled", ConnectionState=Disconnected

Result:
- enabledCount = 2 (A, B; C skipped)
- stableCount = 2 (both Ready)
- AppState = Running ✅
```

**Example 2: Mixed States**
```
Server A: startup_mode="active", ConnectionState=Ready
Server B: startup_mode="active", ConnectionState=Error
Server C: startup_mode="lazy_loading", ConnectionState=Disconnected (DB server_state="stopped")

Result:
- enabledCount = 3 (all enabled)
- stableCount = 1 (only A ready)
- errorCount = 1 (B)
- AppState = Degraded ⚠️
```

**Example 3: Auto-Disabled Server**
```
Server A: startup_mode="active", ConnectionState=Ready
Server B: startup_mode="auto_disabled", ConnectionState=Disconnected
Server C: startup_mode="quarantined", ConnectionState=Disconnected

Result:
- enabledCount = 1 (only A; B auto_disabled, C quarantined both skipped)
- stableCount = 1 (A ready)
- AppState = Running ✅
```

**Example 4: All Servers Stopped (Tray "Stop All")**
```
Current (Broken - uses Stopped boolean):
Server A: startup_mode="active", Stopped=true, ConnectionState=Disconnected
Server B: startup_mode="active", Stopped=true, ConnectionState=Disconnected

Result:
- enabledCount = 2 (counts as enabled due to startup_mode!)
- stableCount = 0 (none ready)
- AppState = Degraded ❌ (WRONG!)

Proposed (Fixed - with "stopped" state):
Server A: startup_mode="stopped", ConnectionState=Disconnected
Server B: startup_mode="stopped", ConnectionState=Disconnected

Result:
- enabledCount = 0 (both stopped, skipped like disabled)
- AppState = Running ✅ (no enabled servers to monitor)
```

## State Persistence and Synchronization

### Two-Phase Commit (storage/manager.go)

```go
// When startup_mode changes:
1. Update in-memory config (s.config.Servers[i].StartupMode = newMode)
2. Save to BBolt database (storage.SaveUpstreamServer)
3. Save to JSON config file (config.SaveConfig)
4. On error: Rollback both database and file
```

### Event-Driven Updates (events/bus.go)

```go
// When ConnectionState changes:
StateManager.SetState(newState)
    ↓
eventBus.Publish(ServerStateChanged)
    ↓
WebSocket: /ws/events broadcasts to clients
    ↓
Tray UI updates menu icons
```

### Config Reload (config/loader.go)

```go
// File watcher detects mcp_config.json changes:
1. Reload config from disk
2. For each server:
   a. Compare old vs new startup_mode
   b. If changed, update StateManager.serverState
   c. If changed to "disabled", disconnect
   d. If changed to "active", attempt connection
3. Publish ConfigChangeData event
```

## Critical State Consistency Rules

### Rule 1: Single Source of Truth
```go
// startup_mode is authoritative
assert(config.Servers[i].StartupMode == storage.GetServer(name).StartupMode)
assert(NO boolean flags like Stopped, Enabled, Quarantined, AutoDisabled)
```

### Rule 2: Runtime Mirrors Config
```go
// StateManager.serverState mirrors startup_mode
assert(stateManager.serverState == convertStartupMode(config.StartupMode))
```

### Rule 3: Connection State Independent
```go
// ConnectionState can change freely without affecting startup_mode
// Example: Server can be "active" but temporarily in "Error" state
assert(config.StartupMode == "active" && connectionState == Error)  // Valid!
```

### Rule 4: AppState Derived, Never Stored
```go
// AppState is ALWAYS computed from server states, never persisted
appState := appStateMachine.CheckServerStability()  // Recomputed on demand
```

## State Transition Flowchart

```
┌─────────────────────────────────────────────────────────────┐
│                    User Action / Event                      │
└─────────────────────────────────────────────────────────────┘
                          │
                          ↓
┌─────────────────────────────────────────────────────────────┐
│              Update startup_mode in Config                   │
│  (via tray UI, API call, or manual config edit)            │
└─────────────────────────────────────────────────────────────┘
                          │
                          ↓
┌─────────────────────────────────────────────────────────────┐
│                Two-Phase Persistence                         │
│  1. BBolt database (storage.SaveUpstreamServer)             │
│  2. JSON config file (config.SaveConfig)                    │
│  3. On error → Rollback both                                │
└─────────────────────────────────────────────────────────────┘
                          │
                          ↓
┌─────────────────────────────────────────────────────────────┐
│         Update StateManager.serverState (in-memory)         │
│  stateManager.SetServerState(newStartupMode)                │
└─────────────────────────────────────────────────────────────┘
                          │
                          ↓
┌─────────────────────────────────────────────────────────────┐
│            Connection State Transition                       │
│  if newMode == "active":    Connect()                       │
│  if newMode == "disabled":  Disconnect()                    │
│  if newMode == "stopped":   Disconnect() (proposed)         │
└─────────────────────────────────────────────────────────────┘
                          │
                          ↓
┌─────────────────────────────────────────────────────────────┐
│                Publish Events                                │
│  1. ServerStateChanged (startup_mode change)                │
│  2. ConnectionEstablished / ConnectionLost                  │
│  3. ConfigChangeData (for tray UI)                          │
└─────────────────────────────────────────────────────────────┘
                          │
                          ↓
┌─────────────────────────────────────────────────────────────┐
│         AppStateMachine.UpdateState()                       │
│  Aggregate all server states → new AppState                 │
│  Publish AppStateChanged event                              │
└─────────────────────────────────────────────────────────────┘
                          │
                          ↓
┌─────────────────────────────────────────────────────────────┐
│              UI Updates (Event-Driven)                       │
│  1. WebSocket clients receive events (/ws/events)          │
│  2. Tray menu icons update                                  │
│  3. Web dashboard refreshes                                 │
└─────────────────────────────────────────────────────────────┘
```

## Files and Code References

### Configuration Loading
- **Entry Point**: [internal/server/server.go:630-688](internal/server/server.go#L630-L688)
- **Config Struct**: [internal/config/config.go:180-217](internal/config/config.go#L180-L217)
- **Database Save**: [internal/storage/manager.go](internal/storage/manager.go)

### Runtime State Management
- **StateManager**: [internal/upstream/types/types.go:163-192](internal/upstream/types/types.go#L163-L192)
- **ConnectionInfo**: [internal/upstream/types/types.go:142-160](internal/upstream/types/types.go#L142-L160)
- **State Transitions**: [internal/upstream/managed/client.go](internal/upstream/managed/client.go)

### Connection Logic
- **Upstream Manager**: [internal/upstream/manager.go:270-320](internal/upstream/manager.go#L270-L320)
- **Add Server**: [internal/upstream/manager.go:113-142](internal/upstream/manager.go#L113-L142)
- **Background Reconnect**: [internal/upstream/manager.go:562-625](internal/upstream/manager.go#L562-L625)

### App State Aggregation
- **AppStateMachine**: [internal/server/app_state_machine.go:17-30](internal/server/app_state_machine.go#L17-L30)
- **CheckServerStability**: [internal/server/app_state_machine.go:179-237](internal/server/app_state_machine.go#L179-L237)
- **Transition Logic**: [internal/server/app_state_machine.go:94-175](internal/server/app_state_machine.go#L94-L175)

### Event System
- **Event Bus**: [internal/events/bus.go](internal/events/bus.go)
- **Event Types**: [internal/events/events.go](internal/events/events.go)
- **WebSocket Streaming**: [internal/server/websocket.go](internal/server/websocket.go)

## Current Issues and Fixes Needed

### Issue 1: Stopped Boolean Field

**Location**: [internal/config/config.go:217](internal/config/config.go#L217)

**Problem**: Creates flag combinations violating single-state principle
```go
// CURRENT (Broken):
startup_mode: "active"
Stopped: true           // FLAG COMBINATION!

// PROPOSED (Fixed):
startup_mode: "stopped"  // SINGLE STATE!
```

**Impact on State Flow**:
- **Current**: AppStateMachine counts stopped servers as enabled (degraded state incorrectly)
- **Fixed**: AppStateMachine skips stopped servers like disabled ones (correct state)

### Issue 2: CheckServerStability Needs Update

**Location**: [internal/server/app_state_machine.go:193-194](internal/server/app_state_machine.go#L193-L194)

**Current Code**:
```go
if client.Config.StartupMode == "disabled" || client.Config.StartupMode == "quarantined" {
    continue
}
```

**After Fix**:
```go
if client.Config.StartupMode == "disabled" ||
   client.Config.StartupMode == "quarantined" ||
   client.Config.StartupMode == "stopped" {  // ADD THIS
    continue
}
```

### Issue 3: Connection Manager Needs Update

**Location**: [internal/upstream/manager.go:277-296](internal/upstream/manager.go#L277-L296)

**Add Check**:
```go
if serverConfig.StartupMode == "stopped" {
    m.logger.Debug("Skipping connection for stopped server",
        zap.String("id", id),
        zap.String("name", serverConfig.Name))
    return nil
}
```

## Validation and Testing

### Unit Tests Needed

1. **State Flow Test**:
```go
func TestStateFlow_ConfigToRuntime(t *testing.T) {
    // Test: startup_mode="active" → ConnectionState=Connecting
    // Test: startup_mode="lazy_loading" → ConnectionState=Disconnected (DB server_state="stopped")
    // Test: startup_mode="disabled" → ConnectionState=Disconnected (no connect)
}
```

2. **App State Aggregation Test**:
```go
func TestAppState_AllServersReady(t *testing.T) {
    // All active servers → AppState=Running
}

func TestAppState_SomeErrors(t *testing.T) {
    // Some errors → AppState=Degraded
}

func TestAppState_StoppedServers(t *testing.T) {
    // All stopped → AppState=Running (no enabled servers)
}
```

3. **Persistence Test**:
```go
func TestPersistence_StartupModeChanges(t *testing.T) {
    // Change startup_mode → Verify DB and file both updated
    // Restart application → Verify startup_mode preserved
}
```

### Integration Tests

```bash
# E2E test with production config
go test -v ./internal/server -run TestE2E_StateFlow

# Test auto-disable persistence
go test -v ./internal/server -run TestE2E_AutoDisablePersistence

# Test AppState transitions
go test -v ./internal/server -run TestE2E_AppStateTransitions
```

---

**Document Version**: 1.0
**Last Updated**: 2025-01-16
**Status**: Complete - Pending Implementation of Stopped Field Fix

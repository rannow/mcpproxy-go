# State Management Architecture

## Overview

MCPProxy implements a three-tier state management system with clear separation between application-level, server configuration, and runtime connection states.

## State Hierarchy

```
┌─────────────────────────────────────────────────────────┐
│                     AppState                            │
│              (Application-Level)                        │
│                                                         │
│  starting → running → degraded → stopping → stopped    │
└─────────────────────────────────────────────────────────┘
                          │
                          │ manages
                          ↓
┌─────────────────────────────────────────────────────────┐
│                  ServerState                            │
│          (Per-Server Configuration)                     │
│                                                         │
│  active ←→ lazy_loading ←→ disabled                    │
│     ↓                                                   │
│  auto_disabled (after failures)                        │
│     ↓                                                   │
│  quarantined (security)                                │
│                                                         │
│  ** ISSUE: Stopped boolean creates flag combinations   │
└─────────────────────────────────────────────────────────┘
                          │
                          │ controls
                          ↓
┌─────────────────────────────────────────────────────────┐
│               ConnectionState                           │
│            (Per-Server Runtime)                         │
│                                                         │
│  Disconnected → Connecting → Authenticating →          │
│  Discovering → Ready                                    │
│       ↓                                                 │
│  Error (temporary)                                      │
└─────────────────────────────────────────────────────────┘
```

## 1. AppState (Application-Level)

**Location**: `internal/server/server.go`

**Purpose**: Tracks the overall health and lifecycle of the mcpproxy application.

**Values**:
- `starting` - Application is initializing
- `running` - All systems operational
- `degraded` - Some servers failing but app functional
- `stopping` - Graceful shutdown in progress
- `stopped` - Application has stopped

**Transitions**:
```
starting → running       (successful startup)
running → degraded       (server failures detected)
degraded → running       (servers recovered)
running → stopping       (shutdown initiated)
degraded → stopping      (shutdown initiated)
stopping → stopped       (shutdown complete)
stopped → starting       (restart)
```

**Storage**: In-memory only (not persisted)

**Events**: `AppStateChanged` published on transitions

## 2. ServerState (Per-Server Configuration)

**Location**: `internal/upstream/types/types.go`, `internal/config/config.go`

**Purpose**: Represents the persisted configuration state of each MCP server.

**Current Values** (from startup_mode):
- `active` - Start on boot, auto-reconnect on failure
- `lazy_loading` - Start on first tool call, remain disconnected until needed
- `disabled` - User disabled, will not connect
- `quarantined` - Security quarantine, tools blocked
- `auto_disabled` - Automatically disabled after N connection failures

**State Transition Rules**:

### Normal Flow (User-Initiated)
```
active ←→ disabled           (user enable/disable via UI/API)
active ←→ lazy_loading       (user changes startup mode)
lazy_loading ←→ disabled     (user enable/disable)
```

### Quarantine Flow (Security-Triggered)
```
[any state] → quarantined    (automatic on security detection)
quarantined → [original]     (manual approval via UI only)
```

### Auto-Disable Flow (Failure-Triggered)
```
active → auto_disabled           (automatic after N failures)
auto_disabled → active           (manual re-enable or group enable)
lazy_loading → auto_disabled     (automatic after N failures)
auto_disabled → lazy_loading     (manual re-enable with lazy mode)
```

**Storage**:
- BBolt database (`internal/storage/`)
- JSON config file (`~/.mcpproxy/mcp_config.json`)
- Two-phase commit with rollback support

**Events**: `ServerStateChanged`, `ServerAutoDisabled` published on transitions

**Helper Methods**:
```go
func (s ServerState) IsStable() bool
func (s ServerState) IsEnabled() bool
func (s ServerState) IsDisabled() bool
```

### Stopped Field Issue

**Current Implementation**:
```go
// internal/config/config.go:217
Stopped bool `json:"stopped,omitempty" mapstructure:"stopped"`
```

**Problem**: Creates flag combinations AND persists runtime-only state:
- `startup_mode="active" + Stopped=true` (flag combination violation)
- `startup_mode="lazy_loading" + Stopped=true` (flag combination violation)
- **Stopped state is persisted to disk** (architecture violation - should be runtime-only)

This violates TWO principles:
1. **Single-State Principle**: Each server should have exactly ONE state
2. **Runtime-Only Principle**: Transient UI state should NOT be persisted

**Usage**:
- Tray UI: "Stop All Servers" button sets `Stopped=true`
- Tray UI: "Start All Servers" button sets `Stopped=false`
- Connection logic: Skips servers where `Stopped=true`
- **INCORRECTLY persisted to disk** across restarts

**Recommended Fix**: Remove Stopped field entirely, track in StateManager as runtime-only
```go
// Remove from config.ServerConfig:
Stopped bool  // DELETE - should NOT be persisted

// Add to StateManager (runtime-only):
type StateManager struct {
    // ... existing fields ...
    userStopped bool  // Runtime-only: user manually stopped via tray
}
```

**Rationale**:
- "stopped" is a **transient UI state**, not a configuration preference
- When app restarts, all servers should return to their original startup_mode
- User expectation: "Stop All Servers" is temporary, not a configuration change
- Persistence violates principle of least surprise

**State Transitions with Runtime-Only Stopped**:
```
Runtime State (NOT persisted):
active + userStopped=true   (user clicked "Stop All Servers")
→ On restart: reverts to active (userStopped cleared)

lazy_loading + userStopped=true  (user clicked "Stop All Servers")
→ On restart: reverts to lazy_loading (userStopped cleared)
```

## 3. ConnectionState (Per-Server Runtime)

**Location**: `internal/upstream/types/types.go`

**Purpose**: Tracks the real-time connection status of each MCP server.

**Values**:
- `Disconnected` - Not connected, not attempting to connect
- `Connecting` - Attempting to establish connection
- `Authenticating` - Performing OAuth authentication
- `Discovering` - Discovering available tools from server
- `Ready` - Connected and ready for requests
- `Error` - Temporary error state (will retry)

**Transitions**:
```
Disconnected → Connecting → Authenticating → Discovering → Ready
                ↓                ↓              ↓          ↓
              Error ──────────────────────────────────────┘
```

**Storage**: In-memory only (StateManager, not persisted)

**Events**: `ConnectionEstablished`, `ConnectionLost` published on transitions

**Relationship to ServerState**:
- `startup_mode="disabled"` → ConnectionState remains `Disconnected`
- `startup_mode="active"` → ConnectionState cycles through Connecting → Ready
- `startup_mode="lazy_loading"` → ConnectionState is `Disconnected`, database server_state set to "stopped" if tools cached
- `startup_mode="quarantined"` → ConnectionState can be Ready but tools blocked
- `startup_mode="auto_disabled"` → ConnectionState is `Disconnected`, no retry

## State Coordination

### Startup Flow
```
1. AppState: starting
2. Load ServerState from disk (BBolt + JSON)
3. For each server with startup_mode="active":
   a. ConnectionState: Disconnected → Connecting
   b. ConnectionState: Connecting → Authenticating (if OAuth)
   c. ConnectionState: Authenticating → Discovering
   d. ConnectionState: Discovering → Ready
   e. ServerState: remains "active"
4. For each server with startup_mode="lazy_loading":
   a. Load tools from database
   b. Set database server_state to "stopped" (ConnectionState remains Disconnected)
   c. ServerState (config): remains "lazy_loading"
5. AppState: starting → running (or degraded if failures)
```

### Failure Handling
```
1. Connection fails N times (N = auto_disable_threshold)
2. ServerState: active → auto_disabled
3. ConnectionState: Error → Disconnected
4. Persist ServerState to disk (BBolt + JSON)
5. Event: ServerAutoDisabled published
6. AppState: May transition to degraded if critical servers affected
```

### User Stop Flow (Current - Broken)
```
1. User clicks "Stop All Servers" in tray
2. For each server:
   a. Set Stopped=true (CREATES FLAG COMBINATION!)
   b. Persist to disk
   c. Disconnect connection
   d. ConnectionState: Ready → Disconnected
   e. ServerState: UNCHANGED (still "active" but Stopped=true)
3. Event: ConfigChangeData{Action: "stopped"}
```

### User Stop Flow (Proposed - Fixed)
```
1. User clicks "Stop All Servers" in tray
2. For each server:
   a. ServerState: active → stopped (SINGLE STATE!)
   b. Persist to disk (BBolt + JSON)
   c. Disconnect connection
   d. ConnectionState: Ready → Disconnected
3. Event: ServerStateChanged published
```

## Persistence Strategy

### Two-Phase Commit
```
Phase 1: Update BBolt database
Phase 2: Update JSON config file
On Error: Rollback both phases
```

**Critical**: Both storage layers must stay synchronized

**Verification**:
```go
// After any state change:
assert(storage.ServerState == configFile.StartupMode)
assert(NO boolean flags like Stopped, Enabled, Quarantined)
```

## Event System

**Event Types**:
- `AppStateChanged` - Application state transitions
- `ServerStateChanged` - Server state transitions
- `ServerAutoDisabled` - Automatic disable triggered
- `ConnectionEstablished` - Connection ready
- `ConnectionLost` - Connection failed
- `ConfigChangeData` - Configuration updated

**WebSocket Streaming**:
- All events published to WebSocket clients
- Real-time UI updates without polling
- Two endpoints:
  - `/ws/events` - All events
  - `/ws/servers?server=name` - Server-filtered events

## Configuration File Format

**Current (Broken)**:
```json
{
  "mcpServers": [
    {
      "name": "github-server",
      "startup_mode": "active",
      "stopped": true,              ← FLAG COMBINATION!
      "url": "https://api.github.com/mcp"
    }
  ]
}
```

**Proposed (Fixed)**:
```json
{
  "mcpServers": [
    {
      "name": "github-server",
      "startup_mode": "stopped",    ← SINGLE STATE!
      "url": "https://api.github.com/mcp"
    }
  ]
}
```

## Migration Path

### Phase 1: Add userStopped to StateManager (runtime-only)
```go
// internal/upstream/types/types.go
type StateManager struct {
    // ... existing fields ...

    // Runtime-only state (NOT persisted)
    userStopped bool  // User manually stopped via tray UI
}

// Helper method
func (sm *StateManager) IsUserStopped() bool {
    return sm.userStopped
}

func (sm *StateManager) SetUserStopped(stopped bool) {
    sm.userStopped = stopped
}
```

### Phase 2: Update migration logic to CLEAR Stopped field
```go
// internal/config/migration.go
func migrateToStartupMode(server *ServerConfig) {
    // IMPORTANT: If Stopped=true, do NOT persist it
    // Instead, clear it so server returns to original startup_mode on restart
    if server.Stopped {
        // Log that we're clearing the stopped state
        log.Info("Clearing runtime-only 'stopped' state from config",
            zap.String("server", server.Name),
            zap.String("startup_mode", server.StartupMode))

        server.Stopped = false  // Clear - should never be persisted
        // Note: StateManager will track userStopped at runtime
    }

    // ... existing migration logic ...
}
```

### Phase 3: Remove Stopped field from ServerConfig
```go
// internal/config/config.go
// DELETE (should NOT be persisted):
// Stopped bool `json:"stopped,omitempty" mapstructure:"stopped"`
```

### Phase 4: Update tray Stop/Start logic
```go
// internal/tray/event_handlers.go
func handleStopAllServers(upstreamManager *upstream.Manager) {
    for _, client := range upstreamManager.GetClients() {
        // Set runtime-only flag (NOT persisted)
        client.StateManager.SetUserStopped(true)
        client.Disconnect()
    }
    // NO config file update - runtime-only change
}

func handleStartAllServers(upstreamManager *upstream.Manager) {
    for _, client := range upstreamManager.GetClients() {
        // Clear runtime-only flag
        client.StateManager.SetUserStopped(false)

        // Reconnect based on original startup_mode
        if client.Config.StartupMode == "active" {
            client.Connect(ctx)
        }
    }
    // NO config file update - runtime-only change
}

### Phase 4: Update all usages
- Replace `serverConfig.Stopped` checks with `serverConfig.StartupMode == "stopped"`
- Update tray Stop/Start buttons to set startup_mode
- Update connection logic to skip "stopped" servers
- Update web UI to display "stopped" state

## Validation Rules

**Single-State Integrity**:
```go
// Rule 1: No boolean flags that modify startup_mode
assert(serverConfig.Stopped == undefined)  // After migration
assert(serverConfig.Enabled == undefined)  // Already removed
assert(serverConfig.Quarantined == undefined)  // Already removed
assert(serverConfig.AutoDisabled == undefined)  // Already removed

// Rule 2: startup_mode is the sole source of truth
assert(serverConfig.StartupMode in ["active", "lazy_loading", "disabled",
                                     "quarantined", "auto_disabled", "stopped"])

// Rule 3: Helper methods must derive from startup_mode only
assert(serverConfig.IsEnabled() == (startup_mode in ["active", "lazy_loading"]))
assert(serverConfig.IsDisabled() == (startup_mode in ["disabled", "auto_disabled", "stopped"]))
```

## Testing Strategy

### Unit Tests
```go
// Test single-state integrity
func TestServerState_NoFlagCombinations(t *testing.T)

// Test all state transitions
func TestServerState_ValidTransitions(t *testing.T)

// Test stopped state
func TestServerState_StoppedByUser(t *testing.T)
```

### Integration Tests
```go
// Test tray stop/start buttons
func TestTray_StopStartAllServers(t *testing.T)

// Test persistence across restarts
func TestServerState_PersistsStopped(t *testing.T)
```

### E2E Tests
```bash
# Test production config with stopped servers
go test -v ./internal/server -run TestE2E_ProductionConfigStartup
```

## References

**Key Files**:
- `internal/upstream/types/types.go` - ServerState and ConnectionState definitions
- `internal/config/config.go` - ServerConfig struct (contains Stopped field issue)
- `internal/server/server.go` - AppState and server management
- `internal/server/app_state_machine.go` - AppState transitions
- `internal/upstream/manager.go` - Connection management (checks Stopped)
- `internal/tray/tray.go` - UI that sets Stopped field
- `docs/STATE_MANAGEMENT.md` - Event-driven architecture documentation

**Design Documents**:
- `docs/STATE_MANAGEMENT.md` - Overall state management design
- `docs/archive/state-management/STATE_MANAGEMENT_REFACTOR.md` - Original refactor plan
- `docs/reports/state-refactoring-analysis.md` - Implementation analysis

## Field Naming and Tray UI Display

### Field Name Conventions Across Layers

The state management system uses different field names at each architectural layer for clarity and to prevent accidental layer mixing:

| Layer | Field Name | Location | Type | Purpose |
|-------|-----------|----------|------|---------|
| **Config** | `startup_mode` | `config.ServerConfig` | `string` | User-facing configuration (persisted to JSON) |
| **Database** | `server_state` | `storage.UpstreamRecord` | `string` | Persisted runtime state (BBolt database) |
| **Runtime** | `userStopped` | `types.StateManager` | `bool` | Runtime-only UI state (NOT persisted) |

**Why Different Names?**

1. **Clarity**: Makes it immediately obvious which architectural layer you're working in
2. **Type Safety**: Prevents accidental mixing of config and storage fields
3. **Maintainability**: Easy to search and find references to each layer
4. **Searchability**: `grep "startup_mode"` finds config layer, `grep "server_state"` finds storage layer

**Field Mapping**:
```
config.ServerConfig.StartupMode (JSON: "startup_mode")
    ↓ persisted to database ↓
storage.UpstreamRecord.ServerState (JSON: "server_state")
    ↓ loaded at runtime ↓
types.StateManager.GetServerState() → "active"|"lazy_loading"|"disabled"|"quarantined"|"auto_disabled"
```

### Complete State Value Reference

#### AppState (Application-Level)
**Location**: `internal/server/server.go` (in-memory only)
**Values**:
- `starting` - Application initializing, servers connecting
- `running` - All systems operational, no critical issues
- `degraded` - Some servers failing but application functional
- `stopping` - Graceful shutdown in progress
- `stopped` - Application has stopped

#### ServerState (Per-Server Configuration)
**Config Field**: `startup_mode` (in `config.ServerConfig`)
**Database Field**: `server_state` (in `storage.UpstreamRecord`)
**Values**:
- `active` - Start on boot, auto-reconnect on failure
- `lazy_loading` - Start on first tool call, remain disconnected until needed (database server_state may be "stopped")
- `disabled` - User disabled, will not connect
- `quarantined` - Security quarantine, tools blocked from execution
- `auto_disabled` - Automatically disabled after N connection failures

#### ConnectionState (Per-Server Runtime)
**Location**: `types.StateManager` (in-memory only)
**Values**:
- `Disconnected` - Not connected, not attempting to connect
- `Connecting` - Attempting to establish connection
- `Authenticating` - Performing OAuth authentication
- `Discovering` - Discovering available tools from server
- `Ready` - Connected and ready for tool requests
- `Error` - Temporary error state (will retry based on configuration)

### Tray UI State Display Mapping

The system tray displays state information in multiple menu items. Here's the complete mapping:

#### Main Status Item (Lines 783-803 in `internal/tray/tray.go`)
**Displays**: `AppState` (application-level state)

**Format**:
```
Status: Starting...      → AppState="starting"
Status: Running (addr)   → AppState="running"
Status: Degraded (addr)  → AppState="degraded"
Status: Stopping...      → AppState="stopping"
Status: Stopped          → AppState="stopped"
```

**Code Reference**:
```go
switch appState {
case "starting":
    a.statusItem.SetTitle("Status: Starting...")
case "degraded":
    a.statusItem.SetTitle(fmt.Sprintf("Status: Degraded (%s)", listenAddr))
case "stopping":
    a.statusItem.SetTitle("Status: Stopping...")
default: // "running" or empty
    a.statusItem.SetTitle(fmt.Sprintf("Status: Running (%s)", listenAddr))
}
```

#### Start/Stop Server Button (Lines 799, 803)
**Displays**: Server running state (derived from AppState)

**Format**:
```
Stop Server   → Server is running
Start Server  → Server is stopped
```

#### Stop/Start All Servers Button (Lines 1073-1083)
**Displays**: Operation progress with server count

**Format During Operations**:
```
Stopping Servers... (5/10)   → Bulk stop in progress
Starting Servers... (5/10)   → Bulk start in progress
Stop All Servers             → Default state
Start All Servers            → Default state
```

**Code Reference**:
```go
// During stop operation
title := fmt.Sprintf("Stopping Servers... (%d/%d)", i+1, total)
a.stopAllItem.SetTitle(title)

// During start operation
title := fmt.Sprintf("Starting Servers... (%d/%d)", i+1, total)
a.startAllItem.SetTitle(title)
```

#### Individual Server Menu Items
**Displays**: `ConnectionState` + `ServerState` combination

**Format**:
```
[server-name] - Ready              → ConnectionState="Ready", active server
[server-name] - Connecting         → ConnectionState="Connecting"
[server-name] - Error              → ConnectionState="Error"
[server-name] - Disabled           → ServerState="disabled"
[server-name] - Quarantined        → ServerState="quarantined"
[server-name] - Auto-Disabled      → ServerState="auto_disabled"
```

#### Startup Script Submenu (Line 528)
**Displays**: Startup script execution state

**Format**:
```
Startup Script: Running    → Script is executing
Startup Script: Stopped    → Script is not running
Startup Script             → Status unknown
```

#### Progress Indicators (Lines 1158, 1264)
**Displays**: Temporary operation progress

**Format**:
```
[operation-description]... (N/Total)   → Progress during bulk operations
```

### State Display Flow Diagram

```
User Views Tray
    │
    ├─ Main Status Item ──────────────────► AppState (starting/running/degraded/stopping/stopped)
    │
    ├─ Start/Stop Button ─────────────────► Derived from AppState (running → "Stop Server")
    │
    ├─ Server List ───────────────────────► Individual servers show:
    │      │                                  - ConnectionState (Ready/Connecting/Error)
    │      │                                  - ServerState (disabled/quarantined/auto_disabled)
    │      │
    │      └─ Per-Server Actions ──────────► Enable/Disable/Quarantine operations
    │                                         (modify ServerState → persist to config + DB)
    │
    ├─ Stop/Start All Servers ────────────► Progress indicator during bulk operations
    │                                         (shows count: "5/10 servers")
    │
    └─ Startup Script ────────────────────► Script execution state (Running/Stopped)
```

### State Persistence Mapping

| State Type | Tray Display | Config File Field | Database Field | Runtime Only |
|-----------|--------------|-------------------|----------------|--------------|
| **AppState** | Main status item | ❌ Not persisted | ❌ Not persisted | ✅ In-memory |
| **ServerState** | Server list items | ✅ `startup_mode` | ✅ `server_state` | ❌ |
| **ConnectionState** | Server list items | ❌ Not persisted | ❌ Not persisted | ✅ In-memory |
| **userStopped** | Stop/Start buttons | ❌ Not persisted | ❌ Not persisted | ✅ In-memory |

**Key Principle**:
- **Persisted states** (ServerState): Survive application restart, stored in both config.json and BBolt DB
- **Runtime states** (AppState, ConnectionState, userStopped): Reset on application restart, exist only in memory

**Example State Transitions Visible in Tray**:

1. **User Disables Server**:
   - Tray: "[server-name] - Disabled"
   - Config: `startup_mode: "disabled"`
   - Database: `server_state: "disabled"`
   - Persisted: ✅ Yes (survives restart)

2. **User Clicks "Stop All Servers"**:
   - Tray: Button shows "Start All Servers"
   - Runtime: `userStopped = true` for all servers
   - Config: ❌ Not modified
   - Database: ❌ Not modified
   - Persisted: ❌ No (resets on restart)

3. **Server Auto-Disabled After Failures**:
   - Tray: "[server-name] - Auto-Disabled (5 startup failures)"
   - Config: `startup_mode: "auto_disabled"`
   - Database: `server_state: "auto_disabled"`
   - Persisted: ✅ Yes (survives restart)

4. **Application Starting Up**:
   - Tray: "Status: Starting..."
   - AppState: `starting`
   - Persisted: ❌ No (runtime-only)

---

**Document Version**: 1.1
**Last Updated**: 2025-01-16
**Status**: Analysis Complete - Implementation Pending

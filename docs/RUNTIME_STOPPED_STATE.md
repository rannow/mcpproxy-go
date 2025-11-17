# Runtime-Only Stopped State Implementation

## Overview

This document describes the implementation of runtime-only "stopped" state for MCPProxy servers, addressing the issue where transient UI state was incorrectly persisted to configuration files.

## Problem Statement

**Previous Behavior**:
- `Stopped` boolean field was persisted in `config.ServerConfig`
- Created flag combinations: `startup_mode="active" + Stopped=true`
- Violated single-state principle (servers should have ONE state)
- User expectation violated: "Stop All Servers" persisted across restarts

**User Request**:
> "the persisten do not have a stoped state as this is only needed if the app starts then alle server are stopped the app status stopped should not be persisted"

## Solution Architecture

### Layer Separation

**IMPORTANT**: The codebase now uses different field names at each layer for clarity:

| Layer | Field Name | Location | Purpose |
|-------|-----------|----------|---------|
| **Config** | `startup_mode` | `config.ServerConfig` | User-facing configuration |
| **Database** | `server_state` | `storage.UpstreamRecord` | Persisted runtime state |
| **Runtime** | `userStopped` | `types.StateManager` | Runtime-only UI state (NOT persisted) |

### Persistent Layer

**Config File** (`startup_mode`):
- `active` - Start on boot, auto-reconnect on failure
- `lazy_loading` - Start on first tool call only
- `disabled` - User disabled, no connection
- `quarantined` - Security quarantine, no tool execution
- `auto_disabled` - Disabled after N connection failures

**Database** (`server_state`):
- Same values as `startup_mode`
- Automatically mapped: `config.StartupMode` ↔ `storage.ServerState`

**NO "stopped" value in either** - it's purely runtime

### Runtime Layer (In-Memory StateManager)

**Added Field**:
```go
// internal/upstream/types/types.go:187
userStopped bool  // Runtime-only: user manually stopped via tray UI
```

**Helper Methods**:
```go
func (sm *StateManager) IsUserStopped() bool
func (sm *StateManager) SetUserStopped(stopped bool)
```

## Data Flow and Mapping

### Config ↔ Database Mapping

**Automatic Translation** in storage layer:

```go
// Save: Config → Database
func (m *Manager) SaveUpstreamServer(serverConfig *config.ServerConfig) error {
    record := &UpstreamRecord{
        // ... other fields ...
        ServerState: serverConfig.StartupMode,  // config.StartupMode → storage.ServerState
    }
    return m.db.SaveUpstream(record)
}

// Load: Database → Config
func (m *Manager) GetUpstreamServer(name string) (*config.ServerConfig, error) {
    record, err := m.db.GetUpstream(name)
    // ... error handling ...

    return &config.ServerConfig{
        // ... other fields ...
        StartupMode: record.ServerState,  // storage.ServerState → config.StartupMode
    }, nil
}
```

**Why Different Names?**:
1. **Clarity**: Makes it obvious which layer you're in
2. **Type Safety**: Prevents accidental mixing of layers
3. **Maintainability**: Clear separation of concerns
4. **Searchability**: Easy to find config vs storage vs runtime references

## Implementation Changes

### 1. Migration Logic

**File**: [internal/config/migration.go](../internal/config/migration.go)

**Changes**:
- `serverNeedsMigration()` now checks for `Stopped` field
- `migrateServer()` clears `Stopped` field if present
- Migration logs when clearing runtime-only state

**Behavior**:
```go
// If config has Stopped=true
if server.Stopped {
    server.Stopped = false  // Clear - should never be persisted
    // StateManager will track userStopped at runtime if needed
}
```

### 2. StateManager Enhancement

**File**: [internal/upstream/types/types.go](../internal/upstream/types/types.go)

**Added**:
- `userStopped bool` field with comprehensive documentation
- `IsUserStopped()` getter method
- `SetUserStopped(bool)` setter method

**Documentation**:
```go
// Runtime-only UI state (NOT persisted)
// IMPORTANT: This field should NEVER be saved to config or database
// When app restarts, all userStopped flags are cleared and servers return to their original startup_mode
userStopped bool  // User manually stopped via tray UI (runtime-only, never persisted)
```

### 3. Documentation Updates

**Files Updated**:
- [docs/STATE_ARCHITECTURE.md](STATE_ARCHITECTURE.md) - Updated Stopped field analysis and migration path
- [docs/RUNTIME_STOPPED_STATE.md](RUNTIME_STOPPED_STATE.md) - This document

## Behavior

### User Stops Server via Tray

```
1. User clicks "Stop All Servers" in tray
2. For each server:
   - client.StateManager.SetUserStopped(true)
   - client.Disconnect()
3. NO config file update
4. NO database update
```

### User Starts Server via Tray

```
1. User clicks "Start All Servers" in tray
2. For each server:
   - client.StateManager.SetUserStopped(false)
   - If startup_mode == "active":
       client.Connect(ctx)
3. NO config file update
4. NO database update
```

### Application Restart

```
1. Load config from disk (Stopped field cleared by migration)
2. Create StateManager for each server
3. userStopped field defaults to false (never persisted)
4. Servers return to their original startup_mode behavior:
   - "active" → Connect immediately
   - "lazy_loading" → Wait for first tool call
   - "disabled" → Stay disconnected
```

## Connection Logic Integration

**File**: [internal/upstream/manager.go](../internal/upstream/manager.go)

**Recommended Check** (to be implemented):
```go
func (m *Manager) shouldConnect(serverConfig *config.ServerConfig, client *Client) bool {
    // Check runtime-only stopped state
    if client.StateManager.IsUserStopped() {
        return false
    }

    // Check persisted startup_mode
    if serverConfig.StartupMode == "disabled" { return false }
    if serverConfig.StartupMode == "quarantined" { return false }
    if serverConfig.StartupMode == "auto_disabled" { return false }

    return true
}
```

## Testing Strategy

### Unit Tests

**Test Runtime-Only Behavior**:
```go
func TestStateManager_UserStoppedRuntimeOnly(t *testing.T) {
    sm := NewStateManager()

    // Verify default state
    assert.False(t, sm.IsUserStopped())

    // Set userStopped
    sm.SetUserStopped(true)
    assert.True(t, sm.IsUserStopped())

    // Clear userStopped
    sm.SetUserStopped(false)
    assert.False(t, sm.IsUserStopped())
}
```

**Test Migration Clears Stopped**:
```go
func TestMigration_ClearsStoppedField(t *testing.T) {
    server := &ServerConfig{
        Name:        "test-server",
        StartupMode: "active",
        Stopped:     true,  // Should be cleared
    }

    migrateServer(server)

    assert.False(t, server.Stopped, "Stopped field should be cleared")
    assert.Equal(t, "active", server.StartupMode)
}
```

### Integration Tests

**Test Persistence**:
```go
func TestIntegration_StoppedNotPersisted(t *testing.T) {
    // 1. Start server with active mode
    // 2. Set userStopped=true via StateManager
    // 3. Save config to disk
    // 4. Reload config
    // 5. Verify Stopped field not in saved config
    // 6. Verify new StateManager has userStopped=false
}
```

### E2E Tests

**Test Restart Behavior**:
```bash
# 1. Start mcpproxy
./mcpproxy serve &

# 2. Stop all servers via tray UI
# (userStopped=true for all servers)

# 3. Restart mcpproxy
pkill mcpproxy
./mcpproxy serve &

# 4. Verify servers return to original startup_mode
# - active servers reconnect
# - lazy_loading servers wait
# - disabled servers stay disconnected
```

## Validation

### Single-State Integrity

**Rule**: Each server has exactly ONE state (no flag combinations)

**Verification**:
```go
// ✅ VALID: Single state from startup_mode
startup_mode = "active"

// ❌ INVALID: Flag combination (OLD behavior)
startup_mode = "active" + Stopped = true

// ✅ VALID: Runtime-only tracking (NEW behavior)
startup_mode = "active" + userStopped = true (in StateManager, not config)
```

### Persistence Verification

**Rule**: Runtime-only state should NEVER be persisted

**Checks**:
- [ ] `Stopped` field removed from `config.ServerConfig` JSON tags
- [ ] `userStopped` field NOT in `storage.UpstreamRecord`
- [ ] Migration clears any persisted `Stopped` values
- [ ] Config save operations don't include `userStopped`

### Restart Behavior Verification

**Rule**: On restart, all servers return to their original startup_mode

**Test Cases**:
1. **Active Server Stopped by User**:
   - Before restart: `startup_mode="active"`, `userStopped=true`
   - After restart: `startup_mode="active"`, `userStopped=false` → Connects

2. **Lazy Loading Server Stopped by User**:
   - Before restart: `startup_mode="lazy_loading"`, `userStopped=true`
   - After restart: `startup_mode="lazy_loading"`, `userStopped=false` → Sleeps

3. **Disabled Server** (never stopped by user):
   - Before restart: `startup_mode="disabled"`, `userStopped=false`
   - After restart: `startup_mode="disabled"`, `userStopped=false` → Stays disabled

## Remaining Work

### Phase 1: Migration (✅ COMPLETE)
- ✅ Add `userStopped` field to StateManager
- ✅ Implement migration logic to clear `Stopped` field
- ✅ Add helper methods (IsUserStopped, SetUserStopped)
- ✅ Update documentation

### Phase 2: Integration (⏳ PENDING)
- [ ] Update tray Stop/Start logic to use `SetUserStopped()`
- [ ] Update connection logic to check `IsUserStopped()`
- [ ] Remove `Stopped` checks from existing code
- [ ] Update web UI to display userStopped state

### Phase 3: Cleanup (⏳ PENDING)
- [ ] Remove `Stopped` field from `config.ServerConfig`
- [ ] Remove `Stopped` from JSON tags
- [ ] Fix all test files referencing `Stopped`

### Phase 4: Testing (⏳ PENDING)
- [ ] Add unit tests for runtime-only behavior
- [ ] Add integration tests for persistence
- [ ] Add E2E tests for restart behavior
- [ ] Run full test suite

## References

**Documentation**:
- [STATE_ARCHITECTURE.md](STATE_ARCHITECTURE.md) - Three-tier state hierarchy
- [STATE_FLOW.md](STATE_FLOW.md) - Complete state flow from config to runtime

**Key Files**:
- `internal/config/config.go` - Config layer with `startup_mode` field
- `internal/storage/models.go` - Database layer with `server_state` field
- `internal/storage/manager.go` - Mapping logic between config and database
- `internal/config/migration.go` - Migration logic to clear Stopped
- `internal/upstream/types/types.go` - StateManager with `userStopped` field
- `internal/tray/event_handlers.go` - Tray UI handlers (to be updated)
- `internal/upstream/manager.go` - Connection logic (to be updated)

---

**Document Version**: 1.0
**Last Updated**: 2025-01-16
**Status**: Phase 1 Complete, Phase 2-4 Pending

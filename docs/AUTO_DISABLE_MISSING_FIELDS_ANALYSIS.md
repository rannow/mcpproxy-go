# Auto-Disable Missing Fields Bug - Root Cause Analysis

## Executive Summary

**Problem**: 76 servers showing as "Disabled" (enabled=false, auto_disabled=false) instead of "Auto-Disabled" (enabled=false, auto_disabled=true).

**Root Cause**: Three `storage.SaveUpstream()` calls are missing the `AutoDisabled`, `AutoDisableReason`, and `AutoDisableThreshold` fields, causing them to be reset to default values (false/"") in the database.

**Impact**: Servers that fail during connection are being auto-disabled correctly in memory and config, but the database record is missing the auto-disable flags, causing tray UI to categorize them incorrectly.

## Evidence

### Failed Servers Log
- **Total failed**: 89 servers
- **Timeout failures**: 88 servers (context deadline exceeded)
- **OAuth failures**: 1 server
- **All had 3 consecutive failures** (matching auto-disable threshold)

### Current State (from mcp_config.json)
- **76 servers**: enabled=false, auto_disabled=false ❌ (PROBLEM)
- **13 servers**: enabled=false, auto_disabled=true ✅ (CORRECT)
- **70 servers**: enabled=true, auto_disabled=false ✅ (WORKING)

### Tray Categorization Logic
```go
// internal/tray/managers.go:442-451
if server.Enabled {
    connectedServers = append(connectedServers, server)
} else if server.AutoDisabled {
    autoDisabledServers = append(autoDisabledServers, server)
} else {
    disabledServers = append(disabledServers, server)  // ← 76 servers here
}
```

## Root Cause Analysis

### Code Inspection Findings

#### ✅ CORRECT: Auto-Disable Callback (internal/server/server.go:206-229)
```go
upstreamManager.SetServerAutoDisableCallback(func(serverName string, reason string) {
    // Update server config to set enabled=false AND auto_disabled fields
    for i := range server.config.Servers {
        if server.config.Servers[i].Name == serverName {
            server.config.Servers[i].Enabled = false        // ✅
            server.config.Servers[i].AutoDisabled = true    // ✅
            server.config.Servers[i].AutoDisableReason = reason // ✅
            break
        }
    }
    // Save configuration to disk
    server.SaveConfiguration() // ✅ Saves to mcp_config.json
})
```

#### ✅ CORRECT: Auto-Disable Check (internal/upstream/managed/client.go:502-556)
```go
func (mc *Client) checkAndHandleAutoDisable() {
    if !mc.StateManager.ShouldAutoDisable() {
        return
    }

    // Update in-memory state
    mc.StateManager.SetAutoDisabled(reason) // ✅

    // Update config
    mc.Config.AutoDisabled = true          // ✅
    mc.Config.AutoDisableReason = reason   // ✅

    // Trigger callback (which saves to mcp_config.json)
    if mc.onAutoDisable != nil {
        mc.onAutoDisable(mc.Config.Name, reason) // ✅
    }
}
```

#### ❌ BUG 1: handlePersistentFailure Storage Save (internal/upstream/manager.go:788-816)
```go
// Persist to storage
if m.storage != nil {
    if err := m.storage.SaveUpstream(&storage.UpstreamRecord{
        ID:                       client.Config.Name,
        Name:                     client.Config.Name,
        URL:                      client.Config.URL,
        Protocol:                 client.Config.Protocol,
        Command:                  client.Config.Command,
        Args:                     client.Config.Args,
        WorkingDir:               client.Config.WorkingDir,
        Env:                      client.Config.Env,
        Headers:                  client.Config.Headers,
        OAuth:                    client.Config.OAuth,
        RepositoryURL:            client.Config.RepositoryURL,
        Enabled:                  false, // DISABLED
        Quarantined:              client.Config.Quarantined,
        Created:                  client.Config.Created,
        Updated:                  time.Now(),
        Isolation:                client.Config.Isolation,
        GroupID:                  client.Config.GroupID,
        GroupName:                client.Config.GroupName,
        EverConnected:            client.Config.EverConnected,
        LastSuccessfulConnection: client.Config.LastSuccessfulConnection,
        ToolCount:                client.Config.ToolCount,
        // ❌ MISSING: AutoDisabled
        // ❌ MISSING: AutoDisableReason
        // ❌ MISSING: AutoDisableThreshold
    }); err != nil {
        // ...
    }
}
```

**Analysis**: This code is called when a server fails during startup (handlePersistentFailure). It correctly:
1. Sets `client.Config.Enabled = false` (line 756)
2. Calls `client.StateManager.SetAutoDisabled(reason)` (line 760)
3. Sets `client.Config.AutoDisabled = true` (line 763)
4. Sets `client.Config.AutoDisableReason = reason` (line 764)

However, when saving to storage (line 790), it **does NOT include** the AutoDisabled fields, causing the database to reset them to default values (false/"").

#### ❌ BUG 2: Connection History Storage Save (internal/upstream/managed/client.go:156-185)
```go
// Persist connection history to storage
if mc.storage != nil {
    if err := mc.storage.SaveUpstream(&storage.UpstreamRecord{
        ID:                       mc.Config.Name,
        Name:                     mc.Config.Name,
        URL:                      mc.Config.URL,
        Protocol:                 mc.Config.Protocol,
        Command:                  mc.Config.Command,
        Args:                     mc.Config.Args,
        WorkingDir:               mc.Config.WorkingDir,
        Env:                      mc.Config.Env,
        Headers:                  mc.Config.Headers,
        OAuth:                    mc.Config.OAuth,
        RepositoryURL:            mc.Config.RepositoryURL,
        Enabled:                  mc.Config.Enabled,
        Quarantined:              mc.Config.Quarantined,
        Created:                  mc.Config.Created,
        Updated:                  time.Now(),
        Isolation:                mc.Config.Isolation,
        GroupID:                  mc.Config.GroupID,
        GroupName:                mc.Config.GroupName,
        EverConnected:            mc.Config.EverConnected,
        LastSuccessfulConnection: mc.Config.LastSuccessfulConnection,
        ToolCount:                mc.Config.ToolCount,
        // ❌ MISSING: AutoDisabled
        // ❌ MISSING: AutoDisableReason
        // ❌ MISSING: AutoDisableThreshold
    }); err != nil {
        // ...
    }
}
```

**Analysis**: This code runs when a server successfully connects and persists connection history. If a server was previously auto-disabled and then re-enabled, this save would **ERASE** the auto-disable state from the database.

#### ❌ BUG 3: Tool Count Storage Save (internal/upstream/managed/client.go:373-401)
```go
// Persist tool count to storage
if mc.storage != nil {
    if err := mc.storage.SaveUpstream(&storage.UpstreamRecord{
        ID:                       mc.Config.Name,
        Name:                     mc.Config.Name,
        URL:                      mc.Config.URL,
        Protocol:                 mc.Config.Protocol,
        Command:                  mc.Config.Command,
        Args:                     mc.Config.Args,
        WorkingDir:               mc.Config.WorkingDir,
        Env:                      mc.Config.Env,
        Headers:                  mc.Config.Headers,
        OAuth:                    mc.Config.OAuth,
        RepositoryURL:            mc.Config.RepositoryURL,
        Enabled:                  mc.Config.Enabled,
        Quarantined:              mc.Config.Quarantined,
        Created:                  mc.Config.Created,
        Updated:                  time.Now(),
        Isolation:                mc.Config.Isolation,
        GroupID:                  mc.Config.GroupID,
        GroupName:                mc.Config.GroupName,
        EverConnected:            mc.Config.EverConnected,
        LastSuccessfulConnection: mc.Config.LastSuccessfulConnection,
        ToolCount:                mc.Config.ToolCount,
        // ❌ MISSING: AutoDisabled
        // ❌ MISSING: AutoDisableReason
        // ❌ MISSING: AutoDisableThreshold
    }); err != nil {
        // ...
    }
}
```

**Analysis**: This code runs after listing tools to update the tool count. If called on a server that was auto-disabled, it would **ERASE** the auto-disable state.

### Storage Model Verification

The storage model DOES include the auto-disable fields:

```go
// internal/storage/models.go:61-64
type UpstreamRecord struct {
    // ... other fields ...

    // Auto-disable state (for servers automatically disabled due to failures)
    AutoDisabled         bool   `json:"auto_disabled,omitempty"`
    AutoDisableReason    string `json:"auto_disable_reason,omitempty"`
    AutoDisableThreshold int    `json:"auto_disable_threshold,omitempty"`
}
```

## Why This Causes the Problem

### Failure Sequence for 76 Servers

1. **Server fails to connect** (timeout after 3 attempts)
2. **Auto-disable triggered correctly**:
   - `client.Config.Enabled = false` ✅
   - `client.Config.AutoDisabled = true` ✅
   - `client.Config.AutoDisableReason = reason` ✅
3. **Callback invoked** → Updates `mcp_config.json` ✅
4. **Storage save (BUG 1)** → Saves to database WITHOUT auto-disable fields ❌
   - Database now shows: `enabled=false, auto_disabled=false`
5. **Next restart**: Config reloads from `mcp_config.json` (which has correct values)
6. **Tray UI queries database** → Gets `auto_disabled=false` → Categorizes as "Disabled" ❌

### Why 13 Servers Show Correctly

These 13 servers were likely:
- Auto-disabled BEFORE the recent bugs were introduced
- Auto-disabled via a different code path that correctly saves to storage
- Never had their storage record overwritten by the buggy save operations

## Fix Implementation Plan

### Required Changes

#### File 1: internal/upstream/manager.go:788-816
**Location**: `handlePersistentFailure()` function, storage save section

**Current Code**:
```go
if err := m.storage.SaveUpstream(&storage.UpstreamRecord{
    // ... existing fields ...
    Enabled:                  false, // DISABLED
    // ... more fields ...
    ToolCount:                client.Config.ToolCount,
}); err != nil {
```

**Fixed Code**:
```go
if err := m.storage.SaveUpstream(&storage.UpstreamRecord{
    // ... existing fields ...
    Enabled:                  false, // DISABLED
    // ... more fields ...
    ToolCount:                client.Config.ToolCount,
    AutoDisabled:             client.Config.AutoDisabled,
    AutoDisableReason:        client.Config.AutoDisableReason,
    AutoDisableThreshold:     client.Config.AutoDisableThreshold,
}); err != nil {
```

**Lines**: Insert after line 811 (before closing brace)

---

#### File 2: internal/upstream/managed/client.go:156-185
**Location**: `Connect()` function, connection history persistence section

**Current Code**:
```go
if err := mc.storage.SaveUpstream(&storage.UpstreamRecord{
    // ... existing fields ...
    ToolCount:                mc.Config.ToolCount,
}); err != nil {
```

**Fixed Code**:
```go
if err := mc.storage.SaveUpstream(&storage.UpstreamRecord{
    // ... existing fields ...
    ToolCount:                mc.Config.ToolCount,
    AutoDisabled:             mc.Config.AutoDisabled,
    AutoDisableReason:        mc.Config.AutoDisableReason,
    AutoDisableThreshold:     mc.Config.AutoDisableThreshold,
}); err != nil {
```

**Lines**: Insert after line 179 (before closing brace)

---

#### File 3: internal/upstream/managed/client.go:373-401
**Location**: `ListTools()` function, tool count persistence section

**Current Code**:
```go
if err := mc.storage.SaveUpstream(&storage.UpstreamRecord{
    // ... existing fields ...
    ToolCount:                mc.Config.ToolCount,
}); err != nil {
```

**Fixed Code**:
```go
if err := mc.storage.SaveUpstream(&storage.UpstreamRecord{
    // ... existing fields ...
    ToolCount:                mc.Config.ToolCount,
    AutoDisabled:             mc.Config.AutoDisabled,
    AutoDisableReason:        mc.Config.AutoDisableReason,
    AutoDisableThreshold:     mc.Config.AutoDisableThreshold,
}); err != nil {
```

**Lines**: Insert after line 395 (before closing brace)

---

### Post-Fix Cleanup

After applying the fixes, the 76 servers will need to be re-auto-disabled to correct their database state:

**Option 1: Automated Script** (Recommended)
```bash
# Trigger re-evaluation of all failed servers
# This will read from mcp_config.json and update database
mcpproxy call tool --tool-name=upstream_servers --json_args='{"operation":"list"}'
```

**Option 2: Manual Config Reload**
```bash
# Restart mcpproxy - will reload from mcp_config.json
pkill mcpproxy
./mcpproxy serve
```

**Option 3: Database Update Script**
Create a migration script to update the 76 servers directly in the database based on mcp_config.json values.

## Testing Plan

### Test Case 1: New Server Auto-Disable
1. Add a new server that will fail to connect
2. Wait for 3 consecutive failures
3. Verify database shows: `enabled=false, auto_disabled=true, auto_disable_reason="..."`
4. Verify tray shows server in "Auto-Disabled Servers" category

### Test Case 2: Connection History Update
1. Take an auto-disabled server
2. Re-enable it manually
3. Let it connect successfully
4. Verify database PRESERVES auto-disable history (should clear it on re-enable)

### Test Case 3: Tool Count Update
1. Connect to a working server
2. List tools (triggers tool count update)
3. Verify database preserves all server state including auto-disable fields

### Test Case 4: Tray Categorization
1. After fixes, restart mcpproxy
2. Verify tray shows:
   - 70 servers in "Connected Servers"
   - 89 servers in "Auto-Disabled Servers" (76 + 13)
   - 0 servers in "Disabled Servers"

## Success Criteria

- ✅ All three `SaveUpstream()` calls include auto-disable fields
- ✅ No servers show as "Disabled" (only "Connected" or "Auto-Disabled")
- ✅ Database state matches config file state
- ✅ Tray UI correctly categorizes all servers
- ✅ Auto-disable state persists across restarts
- ✅ Re-enabling an auto-disabled server clears the auto-disable state

## Timeline

- **Code Changes**: 15 minutes (3 simple additions)
- **Testing**: 30 minutes (comprehensive testing)
- **Cleanup**: 5 minutes (restart to reload state)
- **Total**: ~50 minutes

## Risk Assessment

**Risk Level**: LOW
- Changes are minimal (adding 3 missing fields)
- No logic changes, only data persistence
- Existing auto-disable logic is working correctly
- Only affects database state, not runtime behavior

**Mitigation**:
- Test on development environment first
- Keep backup of config.db before restart
- Monitor failed_servers.log after fixes

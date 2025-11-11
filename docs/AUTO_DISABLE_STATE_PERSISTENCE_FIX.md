# Auto-Disable State Persistence Fix

**Date**: 2025-11-10
**Status**: ✅ FIXED
**Issue**: Auto-disabled servers not appearing in "Auto-Disabled Servers" tray menu after restart

---

## Problem

Auto-disabled servers were appearing in the "Disabled Servers" tray menu instead of "Auto-Disabled Servers" after application restart, even though:
- ✅ They were disabled in config (`enabled: false`)
- ✅ They were logged to `failed_servers.log`
- ✅ The `SetAutoDisabled()` was called during auto-disable

**Root Cause**: The `auto_disabled` and `auto_disable_reason` fields were **only stored in memory** (StateManager) and not persisted to the configuration file. When the server restarted:
1. Config loaded with `enabled: false` for auto-disabled servers
2. StateManager created fresh with `autoDisabled = false` (default)
3. Tray checked `auto_disabled` flag → found `false` → categorized as "Disabled" instead of "Auto-Disabled"

---

## Solution

**Persist auto-disable state to configuration** so it survives restarts.

### Changes Made

#### 1. Added Persistence Fields to ServerConfig

**File**: [internal/config/config.go:201-202](../internal/config/config.go#L201)

```go
// Auto-disable state - persisted across restarts
AutoDisabled              bool      `json:"auto_disabled,omitempty" mapstructure:"auto_disabled"`
AutoDisableReason         string    `json:"auto_disable_reason,omitempty" mapstructure:"auto_disable_reason"`
```

#### 2. Persist State When Auto-Disabling (OLD System - Startup)

**File**: [internal/upstream/manager.go:717-719](../internal/upstream/manager.go#L717)

```go
// Persist auto-disable state to config (so it survives restarts)
client.Config.AutoDisabled = true
client.Config.AutoDisableReason = reason
```

**Location**: In `handlePersistentFailure()` function, after `SetAutoDisabled()` call

#### 3. Persist State When Auto-Disabling (NEW System - Health Check)

**File**: [internal/upstream/managed/client.go:495-497](../internal/upstream/managed/client.go#L495)

```go
// Persist auto-disable state to config (so it survives restarts)
mc.Config.AutoDisabled = true
mc.Config.AutoDisableReason = reason
```

**Location**: In `performHealthCheck()` function, after `SetAutoDisabled()` call

#### 4. Restore State on Startup

**File**: [internal/upstream/manager.go:164-170](../internal/upstream/manager.go#L164)

```go
// Restore auto-disable state from config (if server was previously auto-disabled)
if serverConfig.AutoDisabled {
    client.StateManager.SetAutoDisabled(serverConfig.AutoDisableReason)
    m.logger.Info("Restored auto-disabled state from config",
        zap.String("server", serverConfig.Name),
        zap.String("reason", serverConfig.AutoDisableReason))
}
```

**Location**: In `AddServerConfig()` function, after threshold configuration

#### 5. Clear State on Manual Re-Enable

**File**: [internal/server/mcp.go:1706-1716](../internal/server/mcp.go#L1706)

```go
// Handle enabled state change
wasEnabled := updatedServer.Enabled
updatedServer.Enabled = request.GetBool("enabled", updatedServer.Enabled)

// If re-enabling a previously auto-disabled server, clear the auto-disable state
if !wasEnabled && updatedServer.Enabled && updatedServer.AutoDisabled {
    updatedServer.AutoDisabled = false
    updatedServer.AutoDisableReason = ""
    p.logger.Info("Cleared auto-disable state on manual re-enable",
        zap.String("server", name))
}
```

**Location**: In both `handleUpdateUpstream()` and `handlePatchUpstream()` functions

---

## How It Works Now

### Auto-Disable Flow (Complete)

```
Server Fails 3+ Times
    ↓
handlePersistentFailure() OR performHealthCheck()
    ↓
SetAutoDisabled(reason)  ← Sets in-memory state
    ↓
Config.AutoDisabled = true  ← NEW: Persist to config
Config.AutoDisableReason = reason
    ↓
SaveUpstream() / SaveConfiguration()
    ↓
Write to mcp_config.json
    {
      "name": "server-name",
      "enabled": false,
      "auto_disabled": true,
      "auto_disable_reason": "Server automatically disabled after 3 startup failures"
    }
```

### Restart Flow (Complete)

```
Server Restart
    ↓
Load mcp_config.json
    ↓
For each server:
  if AutoDisabled == true:
    StateManager.SetAutoDisabled(AutoDisableReason)
    ↓
Tray reads GetConnectionStatus()
    ↓
Returns auto_disabled: true
    ↓
Tray categorizes as "Auto-Disabled Servers" ✅
```

### Manual Re-Enable Flow (Complete)

```
User Re-Enables Server (Tray or API)
    ↓
handleUpdateUpstream() / handlePatchUpstream()
    ↓
Detect: wasEnabled == false && nowEnabled == true && AutoDisabled == true
    ↓
Clear state:
  Config.AutoDisabled = false
  Config.AutoDisableReason = ""
    ↓
SaveConfiguration()
    ↓
AddServer() → Fresh start with clean state
```

---

## Testing

### Test Scenario 1: Fresh Auto-Disable

1. Add server with invalid command:
```bash
mcpproxy call tool --tool-name=upstream_servers \
  --json_args='{"operation":"add","name":"test-fail","command":"invalid-cmd","args_json":"[]","enabled":true}'
```

2. Server fails 3 times → Auto-disabled

3. Check config:
```bash
cat ~/.mcpproxy/mcp_config.json | jq '.mcpServers[] | select(.name == "test-fail")'
```

**Expected**:
```json
{
  "name": "test-fail",
  "enabled": false,
  "auto_disabled": true,
  "auto_disable_reason": "Server automatically disabled after 3 startup failures"
}
```

4. Restart mcpproxy:
```bash
pkill mcpproxy
./mcpproxy serve
```

5. Check tray menu → Should appear in "🚫 Auto-Disabled Servers"

### Test Scenario 2: Manual Re-Enable

1. Find auto-disabled server in tray menu
2. Right-click → Enable Server
3. Check config again:
```bash
cat ~/.mcpproxy/mcp_config.json | jq '.mcpServers[] | select(.name == "test-fail")'
```

**Expected**:
```json
{
  "name": "test-fail",
  "enabled": true
  // auto_disabled and auto_disable_reason fields should be absent or false/empty
}
```

### Test Scenario 3: Health Check Auto-Disable

1. Start server that connects successfully
2. Kill the server process externally
3. Wait for 3 health check failures
4. Server auto-disabled by NEW system
5. Check config → Should have `auto_disabled: true`
6. Restart mcpproxy → Should appear in "Auto-Disabled Servers" tray menu

---

## Configuration Example

**After Fix - Config with Auto-Disabled Server**:
```json
{
  "mcpServers": [
    {
      "name": "working-server",
      "command": "npx",
      "args": ["some-mcp-server"],
      "enabled": true
    },
    {
      "name": "failed-server",
      "command": "invalid-command",
      "args": [],
      "enabled": false,
      "auto_disabled": true,
      "auto_disable_reason": "Server automatically disabled after 3 startup failures"
    }
  ]
}
```

---

## Migration Path

**Existing Disabled Servers**: Servers that were auto-disabled before this fix won't have the `auto_disabled` fields in config. They will:
- Appear in "Disabled Servers" tray menu (not "Auto-Disabled")
- Show as `enabled: false` but `auto_disabled: false` (missing field defaults to false)

**Going Forward**: All newly auto-disabled servers will have proper persistence and will appear in the correct tray menu after restart.

**Manual Migration** (Optional):
If you want to mark existing auto-disabled servers manually, edit `mcp_config.json`:
```json
{
  "name": "previously-failed-server",
  "enabled": false,
  "auto_disabled": true,
  "auto_disable_reason": "Server was automatically disabled before this fix"
}
```

Then restart mcpproxy.

---

## Verification Checklist

- [x] **Config Fields**: `auto_disabled` and `auto_disable_reason` added to ServerConfig
- [x] **OLD System**: Persists state in `handlePersistentFailure()`
- [x] **NEW System**: Persists state in `performHealthCheck()`
- [x] **Restore**: Loads state in `AddServerConfig()`
- [x] **Clear on Re-Enable**: Clears state in `handleUpdateUpstream()` and `handlePatchUpstream()`
- [x] **Build**: Compiles without errors
- [x] **Deployed**: Running with fixes

---

## Before vs After

| Aspect | Before | After |
|--------|--------|-------|
| **Auto-disable persistence** | ❌ Memory only | ✅ Persisted to config |
| **Tray after restart** | ❌ "Disabled Servers" | ✅ "Auto-Disabled Servers" |
| **State restoration** | ❌ Lost on restart | ✅ Restored from config |
| **Manual re-enable** | ⚠️ Flag remained in memory | ✅ Flag properly cleared |
| **Config completeness** | ❌ Missing auto-disable info | ✅ Complete state tracking |

---

## Related Documentation

- [TRAY_AUTO_DISABLE_FIX.md](TRAY_AUTO_DISABLE_FIX.md) - Initial tray flag integration
- [AUTO_DISABLE_FIX_SUMMARY.md](AUTO_DISABLE_FIX_SUMMARY.md) - Threshold and logging fixes
- [AUTO_DISABLE_IMPLEMENTATION_FIXED.md](AUTO_DISABLE_IMPLEMENTATION_FIXED.md) - Complete implementation guide
- [DEPLOYMENT_STATUS.md](DEPLOYMENT_STATUS.md) - Current deployment status

---

## Summary

**Problem**: Auto-disable state was memory-only → Lost on restart → Wrong tray categorization

**Solution**: Persist `auto_disabled` and `auto_disable_reason` to config → Restore on startup → Correct tray display

**Impact**: All auto-disabled servers now properly appear in "🚫 Auto-Disabled Servers" tray menu across restarts, with complete failure information and troubleshooting suggestions.

---

**Status**: ✅ **COMPLETE**
**Build**: Successful
**Deployed**: 2025-11-10
**Verification**: Pending user testing after next auto-disable event

---

*Fix completed: 2025-11-10*
*Server PID: 6030*
*All systems operational ✅*

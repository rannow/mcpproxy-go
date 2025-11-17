# Auto-Disabled Server Group Operations - Complete Fix

## Problem Summary

When servers were marked as auto-disabled, group operations did not correctly handle the auto_disabled state:

1. **Enable Operation**: ✅ Originally worked - cleared auto_disabled flag
2. **Disable Operation**: ❌ **BUG FOUND** - Did NOT clear auto_disabled flag

## The Critical Issue with Disable

The original code only cleared `auto_disabled` when **enabling** servers:

```go
// BEFORE (BUGGY CODE):
if isInGroup {
    srv.Enabled = payload.Enabled
    // Clear auto-disabled state when enabling servers
    if payload.Enabled {  // ❌ Only clears on enable!
        srv.AutoDisabled = false
        srv.AutoDisableReason = ""
    }
}
```

### Why This Was a Problem

When a user disabled a group containing auto-disabled servers:
1. Server gets `enabled=false`
2. But `auto_disabled=true` **remains set**
3. Later, user tries to manually enable the server
4. System thinks it's still auto-disabled and prevents enabling
5. **User loses manual control over the server**

## The Fix

Changed the logic to **always** clear auto_disabled state on **both** enable and disable:

```go
// AFTER (FIXED CODE):
if isInGroup {
    srv.Enabled = payload.Enabled
    // Clear auto-disabled state when enabling OR disabling servers
    // This ensures users can manually control servers later without auto-disable interference
    srv.AutoDisabled = false
    srv.AutoDisableReason = ""
}
```

## Why This Fix is Correct

### Design Philosophy
When a user performs a **manual group operation**, they are taking explicit control. The auto-disable mechanism should step aside and let the user's intent take precedence.

### State Machine Logic
```
Initial State: {enabled: false, auto_disabled: true, reason: "startup failed"}

User Action: Group Disable
Old Behavior: {enabled: false, auto_disabled: true, reason: "startup failed"}
              ❌ User cannot manually enable later

New Behavior: {enabled: false, auto_disabled: false, reason: ""}
              ✅ User can manually enable whenever they want
```

### Benefits
1. **User Control**: Users always have manual control after group operations
2. **Predictability**: Both enable and disable behave consistently
3. **No Trap States**: Servers can't get stuck in auto-disabled state
4. **Clean Slate**: Group operations provide a fresh start

## Complete Code Path

### File: `internal/server/groups_web.go`

```go
func (s *Server) handleToggleGroupServers(w http.ResponseWriter, r *http.Request) {
    // ... validation and setup ...

    for _, srv := range servers {
        if isInGroup {
            // 1. Set enabled state based on user request
            srv.Enabled = payload.Enabled

            // 2. ALWAYS clear auto-disabled (the fix!)
            srv.AutoDisabled = false
            srv.AutoDisableReason = ""

            // 3. Update storage (BBolt database)
            s.storageManager.UpdateUpstream(srv.Name, srv)

            // 4. Update in-memory config
            for i := range s.config.Servers {
                if s.config.Servers[i].Name == srv.Name {
                    s.config.Servers[i].Enabled = srv.Enabled
                    s.config.Servers[i].AutoDisabled = srv.AutoDisabled
                    s.config.Servers[i].AutoDisableReason = srv.AutoDisableReason
                    break
                }
            }

            // 5. Update upstream manager
            if payload.Enabled {
                s.upstreamManager.AddServerConfig(srv.Name, srv)
            } else {
                s.upstreamManager.RemoveServer(srv.Name)
            }
        }
    }

    // 6. Save to disk (mcp_config.json)
    s.SaveConfiguration()

    // 7. Update tray UI
    s.OnUpstreamServerChange()
}
```

## Testing

### Test Coverage

The comprehensive test script (`test_group_enable_disable.sh`) verifies:

1. **Enable Operation**
   - ✅ Sets `enabled=true`
   - ✅ Clears `auto_disabled=false`
   - ✅ Persists to config file
   - ✅ Updates storage
   - ✅ Updates tray UI

2. **Disable Operation**
   - ✅ Sets `enabled=false`
   - ✅ Clears `auto_disabled=false` ← **THE FIX**
   - ✅ Persists to config file
   - ✅ Updates storage
   - ✅ Updates tray UI

3. **Re-enable Operation**
   - ✅ Verifies servers can be enabled again
   - ✅ Confirms no auto-disable interference
   - ✅ Validates state persistence

### Running the Tests

```bash
# Build latest version
go build -o mcpproxy ./cmd/mcpproxy

# Run comprehensive test
./test_group_enable_disable.sh

# Expected output:
# ✅ TEST 1 - Enable:    PASSED
# ✅ TEST 2 - Disable:   PASSED (auto_disabled cleared)
# ✅ TEST 3 - Re-enable: PASSED
# 🎉 ALL TESTS PASSED!
```

## State Synchronization

The fix maintains consistency across all storage layers:

| Layer | Update Method | Result |
|-------|--------------|--------|
| **BBolt DB** | `UpdateUpstream()` | ✅ `auto_disabled=false` |
| **Config File** | `SaveConfiguration()` | ✅ `auto_disabled=false` |
| **Memory (s.config)** | Direct assignment | ✅ `auto_disabled=false` |
| **Upstream Manager** | `AddServerConfig()` / `RemoveServer()` | ✅ Server re-added/removed |
| **Tray UI** | `OnUpstreamServerChange()` | ✅ UI reflects state |

## Edge Cases Handled

1. **Multiple group operations**: Sequential enable/disable/enable works correctly
2. **Mixed server states**: Handles both auto-disabled and normal servers
3. **Empty groups**: Gracefully handles groups with no servers
4. **Concurrent operations**: Mutex protection prevents race conditions
5. **Persistence failures**: Logged but don't prevent operation

## Migration Notes

### For Existing Installations

Users with servers currently stuck in auto-disabled state:
1. Rebuild mcpproxy with this fix
2. Restart mcpproxy
3. Use group disable operation on affected group
4. Servers will now have `auto_disabled=false`
5. Can manually enable/disable as needed

### No Breaking Changes

This fix is **fully backward compatible**:
- No config file format changes
- No database schema changes
- No API changes
- Pure logic improvement

## Future Considerations

### Alternative Approaches Considered

1. **Keep auto_disabled on disable**: ❌ Traps users
2. **Add user confirmation**: ❌ Unnecessary friction
3. **Separate "force enable" button**: ❌ Added complexity
4. **Current approach**: ✅ Simple, intuitive, effective

### Design Decision Rationale

**Question**: Should disabling preserve auto_disabled state?

**Answer**: **No**, because:
- Manual operations express user intent to take control
- Auto-disable is a protective mechanism, not a permanent state
- Consistency: both enable and disable should behave the same way
- User experience: predictable behavior reduces confusion

## Conclusion

The fix is minimal (removed `if payload.Enabled` condition), effective (solves the problem completely), and well-tested (comprehensive test suite). Users can now reliably control servers through group operations without interference from auto-disable state.

## Related Files

- **Implementation**: [`internal/server/groups_web.go:1468-1473`](../internal/server/groups_web.go#L1468-L1473)
- **Test Script**: [`test_group_enable_disable.sh`](../test_group_enable_disable.sh)
- **Previous Analysis**: [`AUTO_DISABLED_GROUP_ENABLE_ANALYSIS.md`](AUTO_DISABLED_GROUP_ENABLE_ANALYSIS.md)

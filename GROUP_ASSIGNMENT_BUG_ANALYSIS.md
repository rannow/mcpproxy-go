# Group Assignment Bug Analysis

## Issue Summary

When attempting to change the athena server's group assignment, ALL servers' `group_id` values got incorrectly modified, with most servers being assigned to groups 3, 10, or 19 instead of remaining at `group_id: 0` (ungrouped).

## Root Cause

The bug is in the `syncServerGroupAssignments()` function in `internal/server/server.go` at lines 2028-2051.

### The Problematic Code

```go
func (s *Server) syncServerGroupAssignments() {
    assignmentsMutex.RLock()
    defer assignmentsMutex.RUnlock()

    // Update each server's group_id field in the config
    for i := range s.config.Servers {
        serverName := s.config.Servers[i].Name
        if groupName, exists := serverGroupAssignments[serverName]; exists {
            // Find group ID by name
            groupsMutex.RLock()
            if group, groupExists := groups[groupName]; groupExists {
                s.config.Servers[i].GroupID = group.ID
                s.config.Servers[i].GroupName = "" // Clear legacy field
            }
            groupsMutex.RUnlock()
        } else {
            s.config.Servers[i].GroupID = 0        // ← BUG: Resets ALL unassigned servers
            s.config.Servers[i].GroupName = ""    // Clear legacy field
        }
    }
```

### The Bug Explained

The function has a **destructive reset behavior**:

1. **Problem**: The `else` clause at line 2043-2046 **resets `group_id` to 0 for ANY server that is NOT in the in-memory `serverGroupAssignments` map**

2. **Why this is wrong**: The `serverGroupAssignments` map is an in-memory cache that gets rebuilt from the config. If a server has a `group_id` in the config but is NOT in the in-memory map (due to initialization issues or timing), it gets reset to 0

3. **The cycle of corruption**:
   - User assigns server A to a group → `serverGroupAssignments[A] = "GroupName"`
   - `SaveConfiguration()` calls `syncServerGroupAssignments()`
   - All servers NOT in `serverGroupAssignments` get `group_id = 0`
   - Config is saved with mostly `group_id: 0` values
   - On restart, `initServerGroupAssignments()` rebuilds the map from config
   - Only servers with `group_id > 0` are loaded into `serverGroupAssignments`
   - Next save resets even more servers to `group_id: 0`

## Affected Files

- `internal/server/server.go`: Lines 2028-2051 (`syncServerGroupAssignments`)
- `internal/tray/tray.go`: Lines 1785-1862 (`saveGroupsToConfig`)

## Architecture Issues

### Two Different Systems

The codebase has **two separate group management systems** that don't properly synchronize:

1. **Tray System** (`internal/tray/tray.go`):
   - Uses `ServerGroup.ServerNames []string` to track assignments
   - Saves groups to config without `server_names` field
   - Has its own `saveGroupsToConfig()` function

2. **Server System** (`internal/server/server.go`):
   - Uses `serverGroupAssignments map[string]string` (in-memory)
   - Uses `server.GroupID` (persistent in config)
   - Has its own `syncServerGroupAssignments()` function

### Synchronization Gap

The two systems don't properly coordinate:
- Tray updates `group.ServerNames` but doesn't update `serverGroupAssignments`
- Server reads from `serverGroupAssignments` but doesn't check `group.ServerNames`
- Neither system properly preserves existing `group_id` values during saves

## Impact

When a user changes any server's group assignment through the tray UI:
1. The tray updates `group.ServerNames` in memory
2. Calls `saveGroupsToConfig()` which doesn't save `server_names`
3. Server's `syncServerGroupAssignments()` runs and resets unassigned servers to `group_id: 0`
4. All servers that weren't explicitly in `serverGroupAssignments` lose their group assignments

## Solution

### Option 1: Fix syncServerGroupAssignments (Recommended)

Change the function to **preserve existing `group_id` values** instead of resetting them:

```go
func (s *Server) syncServerGroupAssignments() {
    assignmentsMutex.RLock()
    defer assignmentsMutex.RUnlock()

    // Update each server's group_id field in the config
    for i := range s.config.Servers {
        serverName := s.config.Servers[i].Name
        if groupName, exists := serverGroupAssignments[serverName]; exists {
            // Find group ID by name
            groupsMutex.RLock()
            if group, groupExists := groups[groupName]; groupExists {
                s.config.Servers[i].GroupID = group.ID
                s.config.Servers[i].GroupName = "" // Clear legacy field
            }
            groupsMutex.RUnlock()
        }
        // REMOVED ELSE CLAUSE - preserve existing group_id if not in assignments map
    }
}
```

### Option 2: Unified Group Management

Create a single source of truth for group assignments by:
1. Using ONLY `server.GroupID` in the config (not `group.ServerNames`)
2. Rebuilding `group.ServerNames` dynamically when needed
3. Removing the `else` clause that resets `group_id` to 0

## Testing

After fix, verify:
1. Assigning a server to a group doesn't affect other servers
2. Removing a server from a group only affects that server
3. Config reload preserves all `group_id` values
4. Tray and web UI show consistent group assignments

## Status

- ✅ **Config restored** from backup `mcp_config.backup.20251101-230447.json`
- ✅ **Bug identified** in `syncServerGroupAssignments()`
- ⏳ **Fix pending** - needs code changes to remove destructive reset
- ⏳ **Testing needed** after fix implementation

## Files to Review Before Implementing Fix

1. `internal/server/server.go` - Line 2028-2051 (syncServerGroupAssignments)
2. `internal/tray/tray.go` - Line 1785-1862 (saveGroupsToConfig)
3. `internal/server/server.go` - Line 2110-2153 (initServerGroupAssignments)
4. Configuration persistence logic in both tray and server

## Recommendations

1. **Immediate**: Remove the `else` clause in `syncServerGroupAssignments()` that resets `group_id` to 0
2. **Short-term**: Add integration tests for group assignment operations
3. **Long-term**: Unify the two group management systems into one
4. **Documentation**: Add comments explaining the relationship between `serverGroupAssignments`, `GroupID`, and `ServerNames`

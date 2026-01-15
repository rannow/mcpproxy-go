# Auto-Disable Bug Fix - Exact Code Changes

## Summary

Three `storage.SaveUpstream()` calls are missing the `AutoDisabled`, `AutoDisableReason`, and `AutoDisableThreshold` fields. This causes the database to reset these fields to default values, making 76 auto-disabled servers appear as manually disabled.

## Root Cause

When servers are auto-disabled:
1. ✅ Config file (`mcp_config.json`) is updated correctly with auto-disable fields
2. ✅ In-memory state is updated correctly
3. ❌ Database (`config.db`) is NOT updated with auto-disable fields

This causes a state mismatch where the tray UI (which reads from database) shows servers as "Disabled" instead of "Auto-Disabled".

## Code Changes

### Change 1: internal/upstream/manager.go

**File**: `/Users/hrannow/Library/CloudStorage/OneDrive-Persönlich/workspace/mcp-server/mcpproxy-go/internal/upstream/manager.go`

**Function**: `handlePersistentFailure()`

**Line**: 811 (after `ToolCount:`)

**Current Code** (lines 788-816):
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
		}); err != nil {
			m.logger.Error("Failed to persist disabled server state",
				zap.String("server", client.Config.Name),
				zap.Error(err))
		}
	}
```

**Fixed Code**:
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
			AutoDisabled:             client.Config.AutoDisabled,         // ADD THIS
			AutoDisableReason:        client.Config.AutoDisableReason,    // ADD THIS
			AutoDisableThreshold:     client.Config.AutoDisableThreshold, // ADD THIS
		}); err != nil {
			m.logger.Error("Failed to persist disabled server state",
				zap.String("server", client.Config.Name),
				zap.Error(err))
		}
	}
```

**Add these 3 lines after line 811**:
```go
			AutoDisabled:             client.Config.AutoDisabled,
			AutoDisableReason:        client.Config.AutoDisableReason,
			AutoDisableThreshold:     client.Config.AutoDisableThreshold,
```

---

### Change 2: internal/upstream/managed/client.go (Connection History)

**File**: `/Users/hrannow/Library/CloudStorage/OneDrive-Persönlich/workspace/mcp-server/mcpproxy-go/internal/upstream/managed/client.go`

**Function**: `Connect()`

**Line**: 179 (after `ToolCount:`)

**Current Code** (lines 156-185):
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
		}); err != nil {
			mc.logger.Warn("Failed to persist connection history to storage",
				zap.String("server", mc.Config.Name),
				zap.Error(err))
		}
	}
```

**Fixed Code**:
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
			AutoDisabled:             mc.Config.AutoDisabled,         // ADD THIS
			AutoDisableReason:        mc.Config.AutoDisableReason,    // ADD THIS
			AutoDisableThreshold:     mc.Config.AutoDisableThreshold, // ADD THIS
		}); err != nil {
			mc.logger.Warn("Failed to persist connection history to storage",
				zap.String("server", mc.Config.Name),
				zap.Error(err))
		}
	}
```

**Add these 3 lines after line 179**:
```go
			AutoDisabled:             mc.Config.AutoDisabled,
			AutoDisableReason:        mc.Config.AutoDisableReason,
			AutoDisableThreshold:     mc.Config.AutoDisableThreshold,
```

---

### Change 3: internal/upstream/managed/client.go (Tool Count)

**File**: `/Users/hrannow/Library/CloudStorage/OneDrive-Persönlich/workspace/mcp-server/mcpproxy-go/internal/upstream/managed/client.go`

**Function**: `ListTools()`

**Line**: 395 (after `ToolCount:`)

**Current Code** (lines 372-401):
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
		}); err != nil {
			mc.logger.Warn("Failed to persist tool count to storage",
				zap.String("server", mc.Config.Name),
				zap.Error(err))
		}
	}
```

**Fixed Code**:
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
			AutoDisabled:             mc.Config.AutoDisabled,         // ADD THIS
			AutoDisableReason:        mc.Config.AutoDisableReason,    // ADD THIS
			AutoDisableThreshold:     mc.Config.AutoDisableThreshold, // ADD THIS
		}); err != nil {
			mc.logger.Warn("Failed to persist tool count to storage",
				zap.String("server", mc.Config.Name),
				zap.Error(err))
		}
	}
```

**Add these 3 lines after line 395**:
```go
			AutoDisabled:             mc.Config.AutoDisabled,
			AutoDisableReason:        mc.Config.AutoDisableReason,
			AutoDisableThreshold:     mc.Config.AutoDisableThreshold,
```

---

## Verification

After applying these changes:

1. **Build and restart**:
   ```bash
   go build -o mcpproxy ./cmd/mcpproxy
   pkill mcpproxy
   ./mcpproxy serve
   ```

2. **Verify database state**:
   - Check that auto-disabled servers now have `auto_disabled=true` in database
   - Tray UI should show servers in "Auto-Disabled Servers" category

3. **Test new failures**:
   - Add a server that will fail
   - Wait for auto-disable (3 failures)
   - Verify database has correct auto-disable fields

## Impact

- **Files Changed**: 2 files (3 locations)
- **Lines Added**: 9 lines total (3 lines × 3 locations)
- **Lines Modified**: 0
- **Risk**: LOW (only adding missing fields, no logic changes)

## Expected Outcome

After restart:
- ✅ 76 "Disabled" servers → Move to "Auto-Disabled Servers"
- ✅ 13 "Auto-Disabled" servers → Remain in "Auto-Disabled Servers"
- ✅ 70 "Connected" servers → Remain in "Connected Servers"
- ✅ Final tray categories: 70 Connected, 89 Auto-Disabled, 0 Disabled

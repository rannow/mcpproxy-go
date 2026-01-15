# Server Disabled Conditions - Complete Reference

This document explains **all conditions and code locations** where a server is set to "disabled" state in mcpproxy.

## Disabled State Types

There are **3 types** of disabled states:

1. **`disabled`** - Manual disable by user
2. **`auto_disabled`** - Automatic disable after repeated failures
3. **`quarantined`** - Security quarantine (treated similarly to disabled)

---

## 🔍 Condition 1: Manual User Disable via Storage API

**Location**: `internal/storage/manager.go:387`

**Condition**:
```go
if !enabled {
    cfg.Servers[i].StartupMode = "disabled"
    cfg.Servers[i].AutoDisableReason = "" // Clear auto-disable reason
}
```

**Trigger**: Called when user explicitly disables a server through:
- Web UI toggle
- System tray menu
- API call to `EnableUpstreamServer(name, false)`

**Code Flow**:
```
storage.EnableUpstreamServer(name, false)
  → cfg.Servers[i].StartupMode = "disabled"
  → Save to config file
  → Update database
```

---

## 🔍 Condition 2: Automatic Disable After Repeated Failures

**Location**: `internal/upstream/manager.go:787`

**Condition**:
```go
// After max consecutive failures
reason := fmt.Sprintf("Server automatically disabled after %d startup failures",
    info.ConsecutiveFailures)
client.Config.StartupMode = "auto_disabled"
client.Config.AutoDisableReason = reason
```

**Trigger**: When a server fails to start **multiple times consecutively**
- Default threshold: configured per server or global setting
- Tracks consecutive failures across retries
- Only counts startup/connection failures

**Code Flow**:
```
Connection fails repeatedly
  → ConsecutiveFailures reaches threshold
  → client.Config.StartupMode = "auto_disabled"
  → client.StateManager.SetAutoDisabled(reason)
  → Persist to storage (storage.SaveUpstreamServerAutoDisabled)
  → Log to failed_servers.log
  → Publish ServerAutoDisabled event
```

**Related Code**:
```go
// storage/manager.go:606
cfg.Servers[i].StartupMode = "auto_disabled"
cfg.Servers[i].AutoDisableReason = reason
```

---

## 🔍 Condition 3: Group Disable Operation

**Location**: `internal/server/groups_web.go:1519-1523`

**Condition**:
```go
// When disabling a group, disable all servers in group
} else {
    // Disable server: Stop it first, then update storage
    s.upstreamManager.RemoveServer(serverName)

    // Use EnableUpstreamServer(name, false) to disable
    updateErr = s.storageManager.EnableUpstreamServer(serverName, false)
}
```

**Trigger**: When a user disables an entire group
- All servers in the group are disabled
- Calls same storage API as manual disable

**Code Flow**:
```
Group disable request
  → For each server in group:
    → upstreamManager.RemoveServer(serverName)
    → storageManager.EnableUpstreamServer(serverName, false)
    → cfg.Servers[i].StartupMode = "disabled"
    → Publish ServerStateChanged event
```

---

## 🔍 Condition 4: Startup Mode Check (Skip Connection)

**Location**: `internal/upstream/manager.go:277-282`

**Condition**:
```go
// Check startup mode and skip connection if not active
if serverConfig.StartupMode == "disabled" {
    m.logger.Debug("Skipping connection for disabled server",
        zap.String("id", id),
        zap.String("name", serverConfig.Name))
    return nil
}
```

**Trigger**: When adding a server or during startup
- Server already has `StartupMode == "disabled"` in config
- Connection attempt is skipped entirely

**Similar Checks**:
```go
// Line 291: Skip auto-disabled servers
if serverConfig.StartupMode == "auto_disabled" {
    m.logger.Debug("Skipping connection for auto-disabled server", ...)
    return nil
}

// Line 284: Skip quarantined servers
if serverConfig.StartupMode == "quarantined" {
    m.logger.Debug("Skipping connection for quarantined server", ...)
    return nil
}
```

---

## 🔍 Condition 5: Status Check Returns Disabled

**Location**: `internal/server/mcp.go:2278-2282`

**Condition**:
```go
// Check if server is disabled first
for _, serverConfig := range p.config.Servers {
    if serverConfig.Name == serverName &&
       serverConfig.StartupMode == "disabled" ||
       serverConfig.StartupMode == "quarantined" {
        return statusDisabled, messageServerDisabled
    }
}
```

**Trigger**: During connection monitoring/status checks
- Returns `"disabled"` status to clients
- Message: `"Server is disabled and will not connect"`

**Also at**:
- `mcp.go:1576`: Direct status assignment
- `mcp.go:1745`: Another status check location
- `mcp.go:2296`: Disconnected state check

---

## 🔍 Condition 6: State Machine Exclusion

**Location**: `internal/server/app_state_machine.go:193`

**Condition**:
```go
for _, client := range clients {
    if client.Config.StartupMode == "disabled" ||
       client.Config.StartupMode == "quarantined" {
        continue  // Skip in health calculations
    }
    enabledCount++
    // ...
}
```

**Trigger**: During application state evaluation
- Disabled servers don't count toward app health
- Not considered in Running/Degraded state calculations

---

## 📋 Complete Code Flow Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                    DISABLED STATE TRIGGERS                   │
└─────────────────────────────────────────────────────────────┘
                              │
        ┌────────────────────┼────────────────────┐
        │                    │                    │
   ┌────▼────┐        ┌──────▼──────┐     ┌──────▼──────┐
   │  Manual │        │   Auto      │     │   Group     │
   │ Disable │        │  Disable    │     │  Disable    │
   └────┬────┘        └──────┬──────┘     └──────┬──────┘
        │                    │                    │
        │              ┌─────▼─────┐              │
        │              │ Failures  │              │
        │              │ > Threshold│              │
        │              └─────┬─────┘              │
        │                    │                    │
   ┌────▼────────────────────▼────────────────────▼────┐
   │     storage.EnableUpstreamServer(name, false)     │
   │              OR                                     │
   │     client.Config.StartupMode = "auto_disabled"   │
   └────┬──────────────────────────────────────────────┘
        │
   ┌────▼────────────────────────────────────────────┐
   │  Update Config: StartupMode = "disabled"        │
   │                 OR "auto_disabled"               │
   └────┬───────────────────────────────────────────┘
        │
   ┌────▼────────────────────────────────────────────┐
   │  Persist to:                                     │
   │    - Config file (config.json)                  │
   │    - Database (BBolt)                           │
   │    - Memory (s.config.Servers)                  │
   └────┬───────────────────────────────────────────┘
        │
   ┌────▼────────────────────────────────────────────┐
   │  Effects:                                        │
   │    - Skip connection attempts                   │
   │    - Return "disabled" status                   │
   │    - Exclude from health checks                 │
   │    - Show in tray as disabled                   │
   │    - Publish state change events                │
   └─────────────────────────────────────────────────┘
```

---

## 🎯 Key Code Locations Summary

| Location | Purpose | Type |
|----------|---------|------|
| `storage/manager.go:387` | Set manual disabled state | Manual |
| `storage/manager.go:606` | Set auto-disabled state | Auto |
| `upstream/manager.go:787` | Auto-disable after failures | Auto |
| `upstream/manager.go:277` | Skip disabled on connect | Check |
| `upstream/manager.go:291` | Skip auto-disabled on connect | Check |
| `server/mcp.go:2280` | Return disabled status | Status |
| `server/mcp.go:1576` | Direct status assignment | Status |
| `server/app_state_machine.go:193` | Exclude from health | Health |
| `server/groups_web.go:1519` | Group disable operation | Manual |

---

## 🔄 Re-enabling a Disabled Server

**Manual Disable → Enable**:
```go
storage.EnableUpstreamServer(name, true)
  → cfg.Servers[i].StartupMode = "active"
  → Clear AutoDisableReason
```

**Auto-Disable → Enable**:
```go
// mcp.go:1721 and 1825
if !wasEnabled && updatedServer.StartupMode == "active" &&
   previousMode == "auto_disabled" {
    updatedServer.StartupMode = "active"
    p.logger.Info("Cleared auto-disable state on manual re-enable")
}
```

---

## 🔍 Debugging Tips

**To check why a server is disabled**:

1. **Check config file**:
   ```bash
   cat ~/.mcpproxy/config.json | jq '.mcpServers[] | select(.name=="SERVER_NAME")'
   ```

2. **Check startup_mode field**:
   - `"disabled"` = Manual disable
   - `"auto_disabled"` = Automatic disable
   - `"quarantined"` = Security quarantine

3. **Check auto_disable_reason**:
   ```json
   {
     "name": "my-server",
     "startup_mode": "auto_disabled",
     "auto_disable_reason": "Server automatically disabled after 3 startup failures"
   }
   ```

4. **Check logs**:
   ```bash
   # Auto-disable logs
   grep "automatically disabled" ~/.mcpproxy/logs/mcpproxy.log

   # Manual disable logs
   grep "Skipping connection for disabled server" ~/.mcpproxy/logs/mcpproxy.log

   # Failed servers log
   cat ~/.mcpproxy/failed_servers.log
   ```

5. **Check database**:
   ```bash
   # Using storage API
   curl http://localhost:8080/api/servers/SERVER_NAME
   ```

---

## 📊 State Transition Diagram

```
    ┌─────────┐
    │ Active  │
    └────┬────┘
         │
    ┌────▼─────────────────────────────────┐
    │                                      │
┌───▼────┐                          ┌──────▼──────┐
│Manual  │                          │  Failures   │
│Disable │                          │ > Threshold │
└───┬────┘                          └──────┬──────┘
    │                                      │
┌───▼────────┐                    ┌────────▼────────┐
│ "disabled" │                    │ "auto_disabled" │
└───┬────────┘                    └────────┬────────┘
    │                                      │
    │         ┌────────────┐               │
    └─────────► Re-enable  ◄───────────────┘
              └─────┬──────┘
                    │
              ┌─────▼────┐
              │  Active  │
              └──────────┘
```

---

## 🚨 Important Notes

1. **Disabled vs Auto-Disabled**:
   - `disabled`: User action, can be re-enabled anytime
   - `auto_disabled`: System action, should be investigated before re-enabling

2. **Quarantined**:
   - Treated same as disabled in most checks
   - Security-related, requires manual review

3. **User Stopped (Runtime-Only)**:
   - NOT persisted to config
   - Temporary state, reset on restart
   - See `client.StateManager.IsUserStopped()`

4. **Event Publishing**:
   - `ServerStateChanged` - On manual disable/enable
   - `ServerAutoDisabled` - On automatic disable
   - Used by tray UI and WebSocket clients

---

## 📝 Related Files

- **Config**: `internal/config/config.go`
- **Storage**: `internal/storage/manager.go`
- **Upstream**: `internal/upstream/manager.go`
- **Server**: `internal/server/mcp.go`, `internal/server/groups_web.go`
- **State**: `internal/server/app_state_machine.go`
- **Types**: `internal/upstream/types/types.go`

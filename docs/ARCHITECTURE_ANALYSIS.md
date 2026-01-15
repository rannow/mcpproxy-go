# MCPProxy Architecture Analysis: Status Reporting vs Functionality Discrepancy

## Executive Summary

**Problem**: 71 servers configured, status reports show many as "connected", but only a fraction are actually functional for tool execution.

**Root Cause**: Three-layer architecture with separate code paths for:
1. Tool Discovery (read-only, cached, optimistic)
2. Tool Execution (real-time, strict validation)
3. Status Reporting (mixed sources, inconsistent state)

**Impact**: Users see "connected" servers that fail when attempting to use tools, creating confusion and unreliable system behavior.

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────┐
│                         CLIENT (Claude Code)                         │
│                                                                       │
│  Uses MCP tools: retrieve_tools, call_tool, upstream_servers        │
└───────────────────────┬─────────────────────────────────────────────┘
                        │
                        ↓
┌─────────────────────────────────────────────────────────────────────┐
│                       MCPPROXY SERVER                                │
│                    (internal/server/mcp.go)                          │
│                                                                       │
│  ┌──────────────────┐  ┌──────────────────┐  ┌──────────────────┐ │
│  │ retrieve_tools   │  │    call_tool     │  │upstream_servers  │ │
│  │  (Discovery)     │  │   (Execution)    │  │  (Status API)    │ │
│  └─────┬────────────┘  └─────┬────────────┘  └─────┬────────────┘ │
└────────┼────────────────────┼─────────────────────┼────────────────┘
         │                    │                     │
         │                    │                     │
    [1] Index/Cache      [2] Direct Call      [3] Manager Stats
         │                    │                     │
         ↓                    ↓                     ↓
┌────────────────┐   ┌──────────────────┐   ┌────────────────────┐
│ Index Manager  │   │ Upstream Manager │   │ Upstream Manager   │
│ (BleveSearch)  │   │   CallTool()     │   │   GetStats()       │
└────────────────┘   └──────────────────┘   └────────────────────┘
         │                    │                     │
         │                    │                     │
         ↓                    ↓                     ↓
┌────────────────┐   ┌──────────────────┐   ┌────────────────────┐
│ Tool Metadata  │   │ Managed Client   │   │ StateManager +     │
│   (Cached)     │   │  (Live Check)    │   │ ConnectionInfo     │
└────────────────┘   └──────────────────┘   └────────────────────┘
```

---

## Three Critical Code Paths

### Path 1: Tool Discovery (retrieve_tools)

**Location**: `internal/server/mcp.go:handleRetrieveTools()`

**Flow**:
```go
1. Client calls retrieve_tools
2. MCPProxyServer.handleRetrieveTools()
3. index.SearchTools(query) → BleveSearch
4. Returns cached ToolMetadata from database
5. NO CONNECTION CHECK
6. NO LIVE VALIDATION
```

**Data Source**:
- BBolt database (`~/.mcpproxy/data/mcpproxy.db`)
- Tool metadata persisted from previous successful connections
- Uses Bleve full-text search index

**Key Code**:
```go
// internal/server/mcp.go:506
func (p *MCPProxyServer) handleRetrieveTools(...) {
    // Search index for matching tools
    results, err := p.index.SearchTools(ctx, query, limit, includeStats)

    // Filter quarantined servers
    filteredResults := filterQuarantinedServers(results)

    // Return cached tools - NO CONNECTION CHECK!
    return formatToolResults(filteredResults)
}
```

**Issue**: Tools are returned from cache even if server is currently disconnected or non-functional.

---

### Path 2: Tool Execution (call_tool)

**Location**: `internal/server/mcp.go:handleCallTool()`

**Flow**:
```go
1. Client calls call_tool with "server:tool"
2. MCPProxyServer.handleCallTool()
3. upstreamManager.CallTool()
4. manager.GetClient(serverName) → Find managed.Client
5. CHECK: client.Config.IsDisabled()
6. CHECK: client.IsConnected() ← STRICT VALIDATION
7. If lazy_loading: Attempt on-demand connection
8. client.CallTool() → coreClient.CallTool()
9. Real MCP protocol call to upstream server
```

**Data Source**:
- Live connection state from `managed.Client`
- Real-time MCP protocol communication
- `StateManager.GetState()` for connection status

**Key Code**:
```go
// internal/upstream/manager.go:416
func (m *Manager) CallTool(ctx context.Context, toolName string, args map[string]interface{}) {
    // Find client for server
    targetClient := findClientByName(serverName)

    // STRICT VALIDATION
    if targetClient.Config.IsDisabled() {
        return fmt.Errorf("server disabled")
    }

    if !targetClient.IsConnected() {
        // Try lazy loading
        if globalConfig.EnableLazyLoading && targetClient.Config.ToolCount > 0 {
            err := targetClient.Connect(ctx)
            if err != nil {
                return fmt.Errorf("lazy loading failed: %w", err)
            }
        } else {
            return fmt.Errorf("server not connected")
        }
    }

    // Execute tool on live connection
    return targetClient.CallTool(ctx, actualToolName, args)
}
```

**Issue**: This path has STRICT validation that Path 1 lacks.

---

### Path 3: Status Reporting (upstream_servers list)

**Location**: `internal/server/mcp.go:handleListUpstreams()`

**Flow**:
```go
1. Client calls upstream_servers with operation="list"
2. MCPProxyServer.handleListUpstreams()
3. upstreamManager.GetStats()
4. For each client:
   a. connectionInfo = client.GetConnectionInfo() ← StateManager
   b. connected = (connectionInfo.State == StateReady)
   c. toolCount = GetTotalToolCount() ← May timeout/fail
5. Return aggregated status
```

**Data Source**:
- Mixed: StateManager (runtime) + Config (persisted)
- `GetTotalToolCount()` attempts live `ListTools()` calls
- 30-second timeout for SSE servers

**Key Code**:
```go
// internal/server/mcp.go:1046
func (p *MCPProxyServer) handleListUpstreams() {
    stats := p.upstreamManager.GetStats()

    for serverName, status := range stats["servers"] {
        serverData := map[string]interface{}{
            "name":      serverName,
            "connected": status["connected"],  // From StateManager
            "state":     status["state"],      // From StateManager
            // ... other fields
        }
    }
}

// internal/upstream/manager.go:892
func (m *Manager) GetStats() map[string]interface{} {
    for id, client := range m.clients {
        connectionInfo := client.GetConnectionInfo()
        status := map[string]interface{}{
            "state":      connectionInfo.State.String(),
            "connected":  connectionInfo.State == types.StateReady,  // ← Optimistic
            "connecting": client.IsConnecting(),
        }
    }
}
```

**Issue**: `StateManager.State == StateReady` does NOT guarantee functional connection for tool execution.

---

## State Management Architecture

### Three-Tier State System

```
┌─────────────────────────────────────────────────────────┐
│                     AppState                            │
│              (Application-Level)                        │
│  starting → running → degraded → stopping → stopped    │
└─────────────────────────────────────────────────────────┘
                          │
                          ↓
┌─────────────────────────────────────────────────────────┐
│                  ServerState                            │
│          (Per-Server Configuration - PERSISTED)         │
│                                                         │
│  active ←→ lazy_loading ←→ disabled                    │
│     ↓                                                   │
│  auto_disabled (after failures)                        │
│     ↓                                                   │
│  quarantined (security)                                │
└─────────────────────────────────────────────────────────┘
                          │
                          ↓
┌─────────────────────────────────────────────────────────┐
│               ConnectionState                           │
│        (Per-Server Runtime - IN-MEMORY ONLY)           │
│                                                         │
│  Disconnected → Connecting → Authenticating →          │
│  Discovering → Ready → Error                            │
└─────────────────────────────────────────────────────────┘
```

### Connection State vs Functional State

**Problem**: `ConnectionState.Ready` means different things in different contexts:

1. **Optimistic Interpretation** (Status API):
   - `StateReady` = "Server reached Ready state at some point"
   - May not reflect current functional status
   - Used by `GetStats()` for status reporting

2. **Strict Interpretation** (Tool Execution):
   - `IsConnected()` checks `StateManager.IsReady()`
   - Additional checks for disabled/quarantined status
   - Attempts lazy loading if needed
   - Used by `CallTool()` for actual execution

**Code Evidence**:
```go
// OPTIMISTIC (Status Reporting)
// internal/upstream/manager.go:906
status := map[string]interface{}{
    "connected": connectionInfo.State == types.StateReady,  // ← Just checks state
}

// STRICT (Tool Execution)
// internal/upstream/manager.go:442-505
if targetClient.Config.IsDisabled() {
    return fmt.Errorf("server disabled")
}
if !targetClient.IsConnected() {
    // Try lazy loading...
    if !targetClient.IsConnected() {
        return fmt.Errorf("server not connected")
    }
}
```

---

## Data Sources and Persistence

### 1. BBolt Database

**Location**: `~/.mcpproxy/data/mcpproxy.db`

**Stores**:
- `UpstreamRecord` with fields:
  - `server_state` (persisted ServerState)
  - `tool_count` (cached from last successful connection)
  - `ever_connected` (boolean)
  - `last_successful_connection` (timestamp)

**Used By**:
- Tool Discovery (via Bleve index)
- Lazy Loading decisions
- Connection history tracking

### 2. JSON Config File

**Location**: `~/.mcpproxy/mcp_config.json`

**Stores**:
- `ServerConfig` with fields:
  - `startup_mode` (active/lazy_loading/disabled/quarantined/auto_disabled)
  - `enabled` (boolean - deprecated but still present)
  - `stopped` (boolean - runtime state incorrectly persisted)
  - `tool_count` (duplicated from database)

**Issue**: Flag combinations create ambiguity:
```json
{
  "name": "github-server",
  "startup_mode": "active",
  "stopped": true,         // ← Creates inconsistency
  "tool_count": 42         // ← May be stale
}
```

### 3. In-Memory State

**Location**: `managed.Client.StateManager`

**Stores**:
- `ConnectionState` (Disconnected/Connecting/Ready/Error)
- `ConnectionInfo` (last error, retry count, timestamps)
- `userStopped` (boolean - runtime-only flag)

**Issue**: Not synchronized with persisted state in real-time.

---

## Identified Design Issues

### Issue 1: Cached Tools Without Connection Validation

**Problem**: `retrieve_tools` returns cached tools from database without checking if server is currently connected.

**Impact**: Users see tools that cannot be executed.

**Evidence**:
```go
// internal/server/mcp.go:506
func (p *MCPProxyServer) handleRetrieveTools() {
    // Returns tools from Bleve index (cached in DB)
    results, err := p.index.SearchTools(ctx, query, limit, includeStats)
    // NO CONNECTION CHECK HERE!
}
```

**Fix Needed**: Add connection status to tool metadata:
```go
type ToolMetadata struct {
    // ... existing fields
    ServerConnected bool      `json:"server_connected"`
    ServerState     string    `json:"server_state"`
    LastVerified    time.Time `json:"last_verified"`
}
```

### Issue 2: Optimistic Status Reporting

**Problem**: `upstream_servers list` reports `connected=true` based on `StateManager.State == StateReady`, which doesn't guarantee functional connectivity.

**Impact**: Dashboards and UIs show servers as "connected" when they cannot execute tools.

**Evidence**:
```go
// internal/upstream/manager.go:906
status := map[string]interface{}{
    "connected": connectionInfo.State == types.StateReady,  // ← No functional check
}
```

**Fix Needed**: Add functional verification:
```go
status := map[string]interface{}{
    "connected":            connectionInfo.State == types.StateReady,
    "functionally_ready":   client.IsConnected() && !client.Config.IsDisabled(),
    "can_execute_tools":    verifyToolExecution(client),
}
```

### Issue 3: Separate Registries for Tools and Status

**Problem**: Tool metadata in Bleve index is separate from runtime connection state in StateManager.

**Impact**: Stale tool data persists after server disconnection.

**Evidence**:
- Tools indexed on successful connection
- Index not updated on disconnection
- No TTL on cached tool metadata

**Fix Needed**: Implement cache invalidation:
```go
// On server disconnection
func (mc *Client) Disconnect() error {
    // ... existing disconnect logic

    // Mark tools as stale in index
    p.index.MarkServerStale(mc.Config.Name)

    // Or remove tools from index
    // p.index.RemoveServerTools(mc.Config.Name)
}
```

### Issue 4: Lazy Loading False Positives

**Problem**: Lazy loading checks `Config.ToolCount > 0` to decide if server can connect on-demand, but this count may be stale.

**Impact**: Failed lazy loading attempts when tools no longer exist.

**Evidence**:
```go
// internal/upstream/manager.go:454
if m.globalConfig.EnableLazyLoading && targetClient.Config.ToolCount > 0 {
    // Assumes tools still exist because ToolCount > 0
    err := targetClient.Connect(ctx)
}
```

**Fix Needed**: Add timestamp validation:
```go
if m.globalConfig.EnableLazyLoading &&
   targetClient.Config.ToolCount > 0 &&
   time.Since(targetClient.Config.LastSuccessfulConnection) < 24*time.Hour {
    // Only attempt lazy loading if connection was recent
}
```

### Issue 5: Health Check Call to ListTools

**Problem**: `GetTotalToolCount()` calls `client.ListTools(ctx)` with 30-second timeout for each server, causing slow status API responses.

**Impact**: Status API can take minutes to respond with many servers configured.

**Evidence**:
```go
// internal/upstream/manager.go:974
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
tools, err := client.ListTools(ctx)
```

**Fix Needed**: Use cached tool counts from database:
```go
func (m *Manager) GetTotalToolCount() int {
    for _, client := range m.clients {
        // Use cached count instead of live call
        totalTools += client.Config.ToolCount
    }
}
```

---

## State Transition Analysis

### Successful Connection Flow

```
1. ConnectAll() initiated
2. For each client:
   a. StateManager: Disconnected → Connecting
   b. coreClient.Connect() performs MCP initialize
   c. StateManager: Connecting → Ready
   d. ListTools() called to discover tools
   e. Tools indexed in Bleve + persisted to BBolt
   f. Config.ToolCount updated
3. Server reported as "connected" in status API
```

### Disconnection Flow (Current - Incomplete)

```
1. Disconnect() called
2. coreClient.Disconnect()
3. StateManager.Reset() → Disconnected
4. Tools remain in Bleve index ← NOT REMOVED
5. Config.ToolCount unchanged ← STALE
6. retrieve_tools still returns cached tools ← MISLEADING
7. Status API reports "connected": false ← CORRECT
```

### Disconnection Flow (Proposed - Complete)

```
1. Disconnect() called
2. coreClient.Disconnect()
3. StateManager.Reset() → Disconnected
4. index.MarkServerStale(serverName) ← ADD THIS
5. Publish StateChange event via EventBus ← ALREADY EXISTS
6. retrieve_tools filters stale tools ← ADD THIS
7. Status API reports "connected": false
```

---

## Connection Status "Who Sets What"

### ServerState (Persisted Configuration)

**Set By**: User actions, auto-disable system
- User enables/disables server → `startup_mode` changes
- Auto-disable after N failures → `startup_mode = "auto_disabled"`
- User quarantines server → `startup_mode = "quarantined"`

**Persistence**: Both BBolt DB and JSON config file

### ConnectionState (Runtime Status)

**Set By**: Connection lifecycle methods
- `Connect()` → `Disconnected → Connecting → Ready`
- `Disconnect()` → `Ready → Disconnected`
- Connection errors → `Error` state
- OAuth flows → `Authenticating` state

**Persistence**: In-memory only (StateManager)

### "connected" Field in Status API

**Set By**: Status aggregation logic
```go
// internal/upstream/manager.go:906
status["connected"] = connectionInfo.State == types.StateReady
```

**Issue**: This is a snapshot of StateManager state, not a live functional check.

---

## Health Check System

### Current Implementation

**Location**: `internal/upstream/manager.go:startHealthCheckMonitor()`

**Flow**:
```go
1. Ticker runs every 60 seconds
2. For each client with HealthCheck=true:
   a. Check IsConnected()
   b. If not connected, call Connect()
3. Log results
```

**Issue**: Only checks `IsConnected()`, doesn't verify tool execution capability.

### Proposed Enhancement

```go
func (m *Manager) performHealthChecks() {
    for _, client := range clients {
        // Current check
        if !client.IsConnected() {
            client.Connect(ctx)
            continue
        }

        // NEW: Verify functional readiness
        if err := verifyToolExecution(client); err != nil {
            m.logger.Warn("Health check failed - server appears connected but tools fail",
                zap.String("server", client.Config.Name),
                zap.Error(err))
            // Trigger reconnection
            client.Disconnect()
            client.Connect(ctx)
        }
    }
}

func verifyToolExecution(client *managed.Client) error {
    // Try a lightweight tool call (e.g., list_tools with timeout)
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    tools, err := client.ListTools(ctx)
    if err != nil {
        return fmt.Errorf("tool execution check failed: %w", err)
    }

    if len(tools) == 0 {
        return fmt.Errorf("no tools returned")
    }

    return nil
}
```

---

## Recommendations

### Immediate Fixes (High Priority)

1. **Add Connection Status to Tool Metadata**
   - Include `server_connected`, `server_state` in ToolMetadata
   - Filter disconnected servers in `retrieve_tools` results
   - Show warning in tool descriptions for stale tools

2. **Fix Status API to Report Functional Status**
   - Add `functionally_ready` field
   - Distinguish between "StateReady" and "can execute tools"
   - Include last verification timestamp

3. **Implement Cache Invalidation**
   - Remove or mark stale tools on disconnection
   - Add TTL to cached tool metadata
   - Periodically verify cached tools

### Short-Term Improvements (Medium Priority)

4. **Enhance Health Checks**
   - Add functional verification (lightweight tool call)
   - Update tool metadata when health check fails
   - Trigger cache invalidation on repeated failures

5. **Fix Lazy Loading Validation**
   - Add timestamp check before lazy loading
   - Verify tools still exist after connection
   - Update ToolCount if tools changed

6. **Remove Flag Combinations**
   - Eliminate `stopped` field from ServerConfig
   - Use only `startup_mode` for state
   - Implement runtime-only `userStopped` flag

### Long-Term Architectural Changes (Low Priority)

7. **Unified State Source**
   - Single source of truth for server status
   - Real-time synchronization between layers
   - Event-driven updates instead of polling

8. **Tool Metadata Versioning**
   - Track when tools were last verified
   - Automatic re-discovery on major version changes
   - Tool signature validation

9. **Graceful Degradation**
   - Show partially functional servers
   - Per-tool availability status
   - Automatic fallback to alternative servers

---

## Testing Strategy

### Unit Tests Needed

```go
// Test tool discovery returns only connected servers
func TestRetrieveTools_OnlyConnectedServers(t *testing.T)

// Test status API reports functional readiness
func TestStatusAPI_FunctionalReadiness(t *testing.T)

// Test cache invalidation on disconnect
func TestDisconnect_InvalidatesToolCache(t *testing.T)

// Test lazy loading with stale tool counts
func TestLazyLoading_StaleToolCount(t *testing.T)
```

### Integration Tests Needed

```go
// Test end-to-end: connect → discover → disconnect → verify stale
func TestE2E_ToolLifecycle(t *testing.T)

// Test health checks detect non-functional connections
func TestHealthCheck_DetectsFailures(t *testing.T)

// Test status consistency across all three paths
func TestStatusConsistency_AllPaths(t *testing.T)
```

---

## References

**Key Files**:
- `internal/server/mcp.go` - MCP tool handlers (retrieve_tools, call_tool, upstream_servers)
- `internal/upstream/manager.go` - Connection management, tool discovery, status aggregation
- `internal/upstream/managed/client.go` - Per-server state management
- `internal/index/bleve.go` - Tool metadata search index
- `docs/STATE_ARCHITECTURE.md` - State management design documentation

**Related Issues**:
- Flag combinations (`stopped` + `startup_mode`)
- Cached tools without connection validation
- Optimistic status reporting
- Slow status API due to ListTools calls

---

**Document Version**: 1.0
**Created**: 2025-01-29
**Status**: Analysis Complete - Recommendations Pending Implementation

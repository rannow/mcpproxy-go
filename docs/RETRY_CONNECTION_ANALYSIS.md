# Retry and Connection System - Technical Analysis

## Executive Summary

This document provides a comprehensive technical analysis of the retry mechanism, connection flow, timeout handling, and error propagation in the mcpproxy-go MCP proxy codebase.

---

## Table of Contents

1. [Retry Mechanism Architecture](#retry-mechanism-architecture)
2. [Connection Flow](#connection-flow)
3. [Timeout Strategy](#timeout-strategy)
4. [Backoff Algorithms](#backoff-algorithms)
5. [Auto-Disable System](#auto-disable-system)
6. [State Machine](#state-machine)
7. [Error Handling & Propagation](#error-handling--propagation)
8. [Health Check & Recovery](#health-check--recovery)

---

## 1. Retry Mechanism Architecture

### 1.1 Two-Phase Connection Strategy

The system implements a sophisticated two-phase connection strategy in `manager.go:ConnectAll()`:

**Phase 1: Initial Connection Attempts**
- Parallel connection attempts for all eligible servers
- Uses semaphore-based concurrency control (max 10 concurrent connections by default)
- Each server gets `DefaultConnectionTimeout` (60 seconds) to establish connection
- Failed servers are collected for Phase 2

**Phase 2: Exponential Backoff Retry**
- Failed servers from Phase 1 are retried up to `MaxConnectionRetries` (5 attempts)
- Exponential backoff delay between retries: 1s, 2s, 4s, 8s, 16s
- Maximum backoff capped at `MaxBackoffDelay` (30 seconds)
- Servers that fail all retries are auto-disabled (if threshold reached)

```go
// manager.go:659-678
func (m *Manager) ConnectAll(ctx context.Context) error {
    // Phase 1: Initial connection attempts
    failedJobs := m.connectPhase(ctx, jobs, maxConcurrent, config.DefaultConnectionTimeout, "initial")

    // Phase 2: Retry failed servers
    if len(failedJobs) > 0 {
        m.retryFailedServers(ctx, failedJobs, maxConcurrent)
    }

    return nil
}
```

### 1.2 Retry Triggers

**Automatic Retry Triggers:**
1. **Connection Errors**: Detected by `client.go:isConnectionError()`
   - Connection refused, timeout, network unreachable
   - SSE/HTTP transport failures
   - Broken pipes, context cancellations

2. **Background Health Checks**: `client.go:performHealthCheck()`
   - Runs every 30 seconds via `backgroundHealthCheck()`
   - Triggers reconnection for servers in Error state
   - Respects exponential backoff timing

3. **OAuth Token Availability**: `manager.go:scanForNewTokens()`
   - Monitors for newly available OAuth tokens every 5 seconds
   - Triggers reconnection when token appears for errored OAuth servers
   - Rate-limited per server (10s cooldown)

**Manual Retry Triggers:**
- `manager.go:RetryConnection(serverName)` - Called via API/Tray
- OAuth completion callbacks (in-process or cross-process via DB events)

### 1.3 Retry Decision Logic

```go
// state_manager.go:442-470
func (sm *StateManager) ShouldRetry() bool {
    if sm.currentState != StateError {
        return false  // Only retry from Error state
    }

    // Exponential backoff calculation
    backoffDuration := time.Duration(1<<uint(retryCount)) * time.Second
    if backoffDuration > config.MaxBackoffMinutes {
        backoffDuration = config.MaxBackoffMinutes  // Cap at 5 minutes
    }

    return time.Since(sm.lastRetryTime) >= backoffDuration
}
```

**Special OAuth Retry Logic:**
OAuth errors use extended backoff intervals to avoid rapid token exhaustion:
- 1st retry: 5 minutes
- 2nd retry: 15 minutes
- 3rd retry: 1 hour
- 4th retry: 4 hours
- 5+ retry: 24 hours (max)

---

## 2. Connection Flow

### 2.1 MCP Initialize Process

**Step-by-Step Connection Sequence:**

```
1. StateManager.TransitionTo(StateConnecting)
   ↓
2. coreClient.Connect(ctx) [with 60s timeout]
   ↓
   ├─ Transport layer initialization (stdio/http/sse)
   ├─ MCP Initialize handshake
   ├─ OAuth authentication (if required)
   └─ Server capability discovery
   ↓
3a. SUCCESS → StateManager.TransitionTo(StateReady)
   ├─ Update server info (name, version)
   ├─ Reset failure counters
   ├─ Persist connection history
   └─ Start background monitoring

3b. FAILURE → StateManager.SetError(err)
   ├─ Increment consecutiveFailures
   ├─ Apply backoff timing
   ├─ Check auto-disable threshold
   └─ Schedule retry attempt
```

### 2.2 Connection States

**Runtime Connection States** (in-memory only):
- `StateDisconnected` - Initial state, no connection
- `StateConnecting` - Connection attempt in progress
- `StateAuthenticating` - OAuth authentication in progress
- `StateDiscovering` - Tool discovery in progress
- `StateReady` - Connected and operational
- `StateError` - Connection failed, retrying with backoff

**Persisted Server States** (database + config):
- `active` - Should connect on startup
- `disabled` - Manually disabled, never connects
- `quarantined` - Untrusted, requires manual approval
- `auto_disabled` - Automatically disabled after repeated failures
- `lazy_loading` - Connects on-demand when tool is called

### 2.3 Concurrent Connection Protection

**Reconnection Mutex** (`client.go:694-711`):
```go
mc.reconnectMu.Lock()
if mc.reconnectInProgress {
    mc.reconnectMu.Unlock()
    return  // Prevent duplicate reconnection attempts
}
mc.reconnectInProgress = true
mc.reconnectMu.Unlock()
defer func() {
    mc.reconnectMu.Lock()
    mc.reconnectInProgress = false
    mc.reconnectMu.Unlock()
}()
```

This prevents:
- Duplicate Docker container launches
- Concurrent process spawns
- Race conditions in state transitions

---

## 3. Timeout Strategy

### 3.1 Timeout Hierarchy

**Connection Timeouts** (from `config/timeouts.go`):

| Operation | Timeout | Purpose |
|-----------|---------|---------|
| `DefaultConnectionTimeout` | 60s | Standard connection establishment |
| `HTTPConnectionTimeout` | 180s | HTTP/SSE transport connections |
| `QuickOperationTimeout` | 10s | Health checks, status queries |
| `ListToolsTimeout` | 30s | Tool discovery operations |
| `BatchOperationTimeout` | 2m | Parallel connection operations |
| `LongRunningOperationTimeout` | 30m | OAuth flows, manual operations |

**Retry & Backoff Timeouts:**

| Backoff Stage | Duration | Context |
|---------------|----------|---------|
| `InitialBackoffDelay` | 1s | First retry attempt |
| `MaxBackoffDelay` | 30s | Maximum per-retry delay |
| `MaxBackoffMinutes` | 5m | Overall backoff cap |
| `TokenReconnectCooldown` | 10s | OAuth token detection rate limit |

**OAuth Extended Backoff:**

| Attempt | Backoff |
|---------|---------|
| 1 | 5 minutes |
| 2 | 15 minutes |
| 3 | 1 hour |
| 4 | 4 hours |
| 5+ | 24 hours |

### 3.2 Timeout Application During Connection

**Phase 1 - Initial Connection** (`manager.go:681-727`):
```go
func (m *Manager) connectPhase(ctx context.Context, jobs []clientJob,
                                maxConcurrent int, timeout time.Duration,
                                phase string) []clientJob {

    for _, job := range jobs {
        go func(j clientJob) {
            // Create per-connection timeout context
            connCtx, cancel := context.WithTimeout(ctx, timeout)
            defer cancel()

            // Attempt connection with timeout enforcement
            if err := j.client.Connect(connCtx); err != nil {
                // Collect failed job for retry
                failedJobs = append(failedJobs, j)
            }
        }(job)
    }

    return failedJobs
}
```

**Phase 2 - Retry with Exponential Backoff** (`manager.go:729-768`):
```go
func (m *Manager) retryFailedServers(ctx context.Context,
                                     failedJobs []clientJob,
                                     maxConcurrent int) {

    for retry := 1; retry <= MaxConnectionRetries; retry++ {
        // Calculate exponential backoff: 1s, 2s, 4s, 8s, 16s
        backoffDelay := time.Duration(1<<uint(retry-1)) * time.Second
        if backoffDelay > config.MaxBackoffDelay {
            backoffDelay = config.MaxBackoffDelay  // Cap at 30s
        }

        time.Sleep(backoffDelay)

        // Retry with same timeout as Phase 1
        failedJobs = m.connectPhase(ctx, failedJobs, maxConcurrent,
                                     config.DefaultConnectionTimeout,
                                     fmt.Sprintf("retry-%d", retry))

        if retry == MaxConnectionRetries && len(failedJobs) > 0 {
            // Auto-disable servers that failed all retries
            for _, job := range failedJobs {
                m.handlePersistentFailure(job.id, job.client)
            }
        }
    }
}
```

### 3.3 Context Propagation

All timeout enforcement uses Go's `context.Context` pattern:

1. **Parent Context**: `context.Background()` at application startup
2. **Timeout Context**: `context.WithTimeout(parent, duration)`
3. **Cancellation**: Automatic when timeout expires or manual `cancel()`
4. **Propagation**: Passed through entire call stack to transport layer

---

## 4. Backoff Algorithms

### 4.1 Standard Exponential Backoff

**Algorithm** (`state_manager.go:442-470`):
```go
// Exponential backoff: 2^retryCount seconds
backoffDuration := time.Duration(1 << uint(retryCount)) * time.Second

// Cap at maximum backoff (5 minutes)
if backoffDuration > config.MaxBackoffMinutes {
    backoffDuration = config.MaxBackoffMinutes
}

// Allow retry if enough time has elapsed
return time.Since(sm.lastRetryTime) >= backoffDuration
```

**Progression:**
- Retry 0: Immediate (first attempt)
- Retry 1: 1 second (2^0)
- Retry 2: 2 seconds (2^1)
- Retry 3: 4 seconds (2^2)
- Retry 4: 8 seconds (2^3)
- Retry 5: 16 seconds (2^4)
- Retry 6+: 30 seconds (capped at MaxBackoffDelay)
- Retry 10+: 5 minutes (capped at MaxBackoffMinutes)

**Overflow Protection:**
```go
// Cap retry count to prevent overflow in 64-bit systems
if retryCount > 30 {
    retryCount = 30
}
```

### 4.2 OAuth Extended Backoff

**Algorithm** (`state_manager.go:495-524`):
```go
func (sm *StateManager) ShouldRetryOAuth() bool {
    var backoffDuration time.Duration

    switch {
    case sm.oauthRetryCount <= 1:
        backoffDuration = config.OAuthBackoffLevel1  // 5 minutes
    case sm.oauthRetryCount <= 2:
        backoffDuration = config.OAuthBackoffLevel2  // 15 minutes
    case sm.oauthRetryCount <= 3:
        backoffDuration = config.OAuthBackoffLevel3  // 1 hour
    case sm.oauthRetryCount <= 4:
        backoffDuration = config.OAuthBackoffLevel4  // 4 hours
    default:
        backoffDuration = config.OAuthBackoffMax     // 24 hours
    }

    return time.Since(sm.lastOAuthAttempt) >= backoffDuration
}
```

**Rationale:**
- OAuth tokens may have rate limits
- Avoid rapid token exhaustion
- Give time for manual intervention
- Prevent account lockouts

### 4.3 Startup Retry Backoff

During application startup (`manager.go:729-768`):

**Phase 2 Retry Sequence:**
```
Attempt 1: Wait 1s  → Retry all failed servers
Attempt 2: Wait 2s  → Retry remaining failures
Attempt 3: Wait 4s  → Retry remaining failures
Attempt 4: Wait 8s  → Retry remaining failures
Attempt 5: Wait 16s → Final retry (or capped at 30s)
           ↓
    Auto-disable persistent failures
```

---

## 5. Auto-Disable System

### 5.1 Failure Tracking

**Consecutive Failure Counter** (`state_manager.go:175`):
```go
func (sm *StateManager) SetError(err error) {
    sm.consecutiveFailures++  // Increment on each error
    sm.retryCount++
    sm.lastRetryTime = time.Now()
}
```

**Reset on Success** (`state_manager.go:149`):
```go
if newState == StateReady {
    sm.consecutiveFailures = 0  // Reset counter
    sm.lastSuccessTime = time.Now()
}
```

### 5.2 Auto-Disable Threshold

**Default Threshold**: 3 consecutive failures

**Configuration:**
- **Global**: `config.AutoDisableThreshold` (default: 3)
- **Per-Server**: `ServerConfig.AutoDisableThreshold` (overrides global)
- **Disable Feature**: Set threshold to 0

**Check Logic** (`state_manager.go:537-548`):
```go
func (sm *StateManager) ShouldAutoDisable() bool {
    // Don't auto-disable if already disabled or threshold is 0
    if sm.autoDisabled || sm.autoDisableThreshold <= 0 {
        return false
    }

    return sm.consecutiveFailures >= sm.autoDisableThreshold
}
```

### 5.3 Auto-Disable Execution

**Trigger Points:**
1. After each connection failure (`client.go:125-138`)
2. During health check cycles (`client.go:593-594`)
3. After max retries during startup (`manager.go:759-766`)

**Auto-Disable Handler** (`client.go:523-589`):
```go
func (mc *Client) checkAndHandleAutoDisable() {
    if !mc.StateManager.ShouldAutoDisable() {
        return
    }

    reason := fmt.Sprintf("Server automatically disabled after %d consecutive failures",
                          info.ConsecutiveFailures)

    // 1. Persist to storage (database + config file)
    if mc.storageManager != nil {
        mc.storageManager.UpdateServerState(mc.Config.Name, reason)
    }

    // 2. Update state machine
    mc.StateManager.SetAutoDisabled(reason)

    // 3. Log detailed failure information
    logs.LogServerFailureDetailed(dataDir, serverName, errorMsg,
                                   consecutiveFailures, firstAttemptTime)

    // 4. Trigger callback for config persistence
    if mc.onAutoDisable != nil {
        mc.onAutoDisable(mc.Config.Name, reason)
    }
}
```

### 5.4 Persistence Strategy

**Two-Phase Commit:**
1. **Database Update**: `storage.UpdateUpstreamServerState()`
2. **Config File Update**: `storage.UpdateServerState()` triggers config save
3. **Event Emission**: `events.ServerAutoDisabled` published to event bus

**State Synchronization:**
- Runtime state: `StateManager.autoDisabled` flag
- Config state: `ServerConfig.StartupMode = "auto_disabled"`
- Database state: `UpstreamRecord.ServerState = "auto_disabled"`

### 5.5 Recovery from Auto-Disable

**Manual Recovery:**
1. User re-enables server via Tray/Web UI
2. Triggers `StateManager.ResetAutoDisable()`
3. Clears `consecutiveFailures` counter
4. Updates config: `startup_mode: "active"`
5. Attempts fresh connection

**Automatic Recovery:**
- No automatic recovery from auto-disabled state
- Requires explicit user intervention
- Prevents infinite failure loops

---

## 6. State Machine

### 6.1 Dual State System

The codebase maintains TWO independent state machines:

#### 6.1.1 Connection State Machine (Runtime)
**Location**: `types/state_manager.go`
**Storage**: In-memory only (not persisted)
**States**: `ConnectionState` enum

```
StateDisconnected → StateConnecting → StateAuthenticating → StateDiscovering → StateReady
                         ↓                    ↓                    ↓              ↓
                    StateError ←──────────────────────────────────────────────────┘
                         ↓
                    StateConnecting (retry with backoff)
```

**Valid Transitions** (`state_manager.go:219-243`):
```go
validTransitions := map[ConnectionState][]ConnectionState{
    StateDisconnected:   {StateConnecting},
    StateConnecting:     {StateAuthenticating, StateDiscovering, StateReady, StateError, StateDisconnected},
    StateAuthenticating: {StateConnecting, StateDiscovering, StateReady, StateError, StateDisconnected},
    StateDiscovering:    {StateReady, StateError, StateDisconnected},
    StateReady:          {StateError, StateDisconnected},
    StateError:          {StateConnecting, StateDisconnected},
}
```

#### 6.1.2 Server State Machine (Persisted)
**Location**: `types/server_state.go`
**Storage**: Database + config file
**States**: `ServerState` enum

```
StateActive ⇄ StateDisabledConfig
    ↓              ↓
    ↓         StateQuarantined
    ↓              ↓
StateAutoDisabled ←┘
    ↓
StateActive (manual recovery only)
```

**Valid Transitions** (`managed/state_machine.go:32-67`):
```go
validTransitions := map[types.ServerState][]types.ServerState{
    StateActive: {
        StateDisabledConfig,
        StateQuarantined,
        StateAutoDisabled,
        StateLazyLoading,
    },
    StateDisabledConfig: {
        StateActive,
        StateLazyLoading,
        StateQuarantined,
    },
    StateQuarantined: {
        StateActive,
        StateDisabledConfig,
    },
    StateAutoDisabled: {
        StateActive,        // Manual recovery
        StateDisabledConfig, // Convert to permanent disable
    },
    StateLazyLoading: {
        StateActive,
        StateDisabledConfig,
        StateQuarantined,
        StateAutoDisabled,
    },
}
```

### 6.2 State Transition Safety

**Validation** (`managed/state_machine.go:94-120`):
```go
func (sm *StateMachine) CanTransitionTo(newState types.ServerState) bool {
    currentState := sm.stateManager.GetServerState()

    // Same state transition is always allowed (no-op)
    if currentState == newState {
        return true
    }

    // Check if transition is defined in validTransitions
    allowedStates, ok := validTransitions[currentState]
    if !ok {
        return false  // Deny by default
    }

    // Check if newState is in the allowed list
    for _, allowed := range allowedStates {
        if allowed == newState {
            return true
        }
    }

    return false
}
```

**Concurrency Protection** (`managed/state_machine.go:125-154`):
- Uses mutex locks to prevent race conditions
- Atomic read-modify-write operations
- Unsafe variants for nested lock scenarios (`GetServerStateUnsafe()`, `TransitionServerStateUnsafe()`)

### 6.3 State Synchronization

**Event-Driven Updates:**
```go
// Connection state changes trigger callbacks
StateManager.onStateChange(oldState, newState, info)

// Server state changes publish events
eventBus.Publish(events.ServerStateChanged)

// Tray UI subscribes to events for real-time updates
```

---

## 7. Error Handling & Propagation

### 7.1 Error Classification

**Connection Errors** (`client.go:766-800`):
```go
connectionErrors := []string{
    "connection refused",
    "no such host",
    "connection reset",
    "broken pipe",
    "network is unreachable",
    "timeout",
    "deadline exceeded",
    "context canceled",
    "SSE stream disconnected",
    "stream disconnected",
    "Failed to reconnect SSE stream",
    "Maximum reconnection attempts",
    "ECONNREFUSED",
}
```

**OAuth Errors** (`client.go:826-851`):
```go
oauthErrors := []string{
    "invalid_token",
    "invalid_grant",
    "access_denied",
    "unauthorized",
    "401",
    "Missing or invalid access token",
    "OAuth authentication failed",
    "oauth timeout",
    "oauth error",
}
```

**OAuth Authorization Requirements** (`client.go:802-823`):
```go
authRequiredErrors := []string{
    "OAuth authorization during MCP init failed",
    "OAuth authorization not implemented",
    "OAuth authorization required",
    "authorization required",
}
```

### 7.2 Error Enrichment

**Context Enhancement** (`manager.go:488-545`):

Errors are enriched at the source with helpful context:

```go
func (m *Manager) CallTool(ctx context.Context, toolName string, args map[string]interface{}) (interface{}, error) {
    // ... connection check ...

    result, err := targetClient.CallTool(ctx, actualToolName, args)
    if err != nil {
        errStr := err.Error()

        // OAuth errors
        if strings.Contains(errStr, "OAuth authentication failed") {
            return nil, fmt.Errorf(
                "server '%s' authentication failed for tool '%s'. " +
                "OAuth/token authentication required but not properly configured. " +
                "Check server authentication settings: %w",
                serverName, actualToolName, err)
        }

        // Rate limiting
        if strings.Contains(errStr, "429") || strings.Contains(errStr, "rate limit") {
            return nil, fmt.Errorf(
                "server '%s' rate limit exceeded for tool '%s'. " +
                "Please wait before making more requests: %w",
                serverName, actualToolName, err)
        }

        // Connection issues
        if strings.Contains(errStr, "connection refused") {
            return nil, fmt.Errorf(
                "server '%s' connection failed for tool '%s'. " +
                "Check if the server URL is correct and the server is running: %w",
                serverName, actualToolName, err)
        }

        // Generic with context
        return nil, fmt.Errorf(
            "tool '%s' on server '%s' failed: %w. " +
            "Check server configuration, authentication, and tool parameters",
            actualToolName, serverName, err)
    }

    return result, nil
}
```

### 7.3 Error Logging Strategy

**Log Level Determination** (`client.go:417-438`):

```go
func (mc *Client) CallTool(ctx context.Context, toolName string, args map[string]interface{}) (*mcp.CallToolResult, error) {
    result, err := mc.coreClient.CallTool(ctx, toolName, args)
    if err != nil {
        if mc.isConnectionError(err) {
            if mc.isNormalReconnectionError(err) {
                // Use WARN for expected reconnections
                mc.logger.Warn("Tool call failed due to connection loss, will attempt reconnection",
                    zap.String("server", mc.Config.Name),
                    zap.String("tool", toolName),
                    zap.String("error_type", "normal_reconnection"),
                    zap.Error(err))
            } else {
                // Use ERROR for unexpected connection failures
                mc.logger.Error("Tool call failed with connection error",
                    zap.String("server", mc.Config.Name),
                    zap.String("tool", toolName),
                    zap.Error(err))
            }
            mc.StateManager.SetError(err)
        } else {
            // Non-connection errors always logged as ERROR
            mc.logger.Error("Tool call failed",
                zap.String("server", mc.Config.Name),
                zap.String("tool", toolName),
                zap.Error(err))
        }
        return nil, err
    }

    return result, nil
}
```

**Detailed Failure Logging** (`client.go:567-583`):

For auto-disabled servers, detailed failure information is logged:

```go
logs.LogServerFailureDetailed(
    dataDir,
    mc.Config.Name,
    errorMsg,
    info.ConsecutiveFailures,
    info.FirstAttemptTime,
)
```

This creates entries in `failed_servers.log` with:
- Server name
- Error message with categorization
- Consecutive failure count
- First attempt timestamp
- Failure timestamp

### 7.4 Error Recovery

**Automatic Recovery Triggers:**
1. Background health checks (every 30s)
2. OAuth token availability detection (every 5s)
3. Manual retry via API/Tray

**Recovery Flow:**
```
Error State
    ↓
ShouldRetry() check (with backoff)
    ↓
tryReconnect()
    ↓
Disconnect (cleanup)
    ↓
Reset StateManager
    ↓
Connect (new attempt)
    ↓
Success → StateReady
Failure → StateError (repeat with longer backoff)
```

---

## 8. Health Check & Recovery

### 8.1 Background Health Monitoring

**Health Check Lifecycle** (`client.go:488-521`):

```go
func (mc *Client) startBackgroundMonitoring() {
    mc.monitoringWG.Add(1)
    go func() {
        defer mc.monitoringWG.Done()

        ticker := time.NewTicker(30 * time.Second)
        defer ticker.Stop()

        for {
            select {
            case <-ticker.C:
                mc.performHealthCheck()
            case <-mc.stopMonitoring:
                return
            }
        }
    }()
}
```

**Health Check Frequency:**
- Background checks: Every 30 seconds per client
- Auto-recovery checks: Every 60 seconds for servers with `HealthCheck` flag
- OAuth token scanning: Every 5 seconds globally

### 8.2 Health Check Logic

**Health Check Decision Tree** (`client.go:591-691`):

```
1. Check if auto-disabled → Skip
   ↓
2. Check if OAuth error
   ├─ Yes → Check OAuth backoff timer
   │         ├─ Elapsed → tryReconnect()
   │         └─ Not elapsed → Skip
   └─ No → Continue
       ↓
3. Check if Error state + ShouldRetry()
   ├─ Yes → tryReconnect()
   └─ No → Continue
       ↓
4. Check if connected
   ├─ No → Skip
   └─ Yes → Continue
       ↓
5. Check if Docker server → Skip (avoid interference)
   ↓
6. Check if HealthCheck enabled
   ├─ No → Skip (opt-in only)
   └─ Yes → Perform active health check
       ↓
7. Try ListTools with 5s timeout
   ├─ Connection error → StateManager.SetError()
   └─ Success → Continue monitoring
```

**Active Health Check Protection:**
- Only performed if `ServerConfig.HealthCheck == true` (opt-in)
- Docker servers are skipped to avoid container interference
- Concurrent ListTools calls are prevented via mutex

### 8.3 Auto-Recovery System

**Manager-Level Health Checks** (`manager.go:1302-1367`):

```go
func (m *Manager) startHealthCheckMonitor() {
    ticker := time.NewTicker(config.AutoRecoveryCheckInterval) // 60 seconds
    defer ticker.Stop()

    for range ticker.C {
        m.performHealthChecks()
    }
}

func (m *Manager) performHealthChecks() {
    for id, client := range clients {
        // Only check servers with HealthCheck enabled
        if !client.Config.HealthCheck {
            continue
        }

        // Skip disabled servers
        if client.Config.IsDisabled() {
            continue
        }

        // Check connection status
        if !client.IsConnected() {
            ctx, cancel := context.WithTimeout(context.Background(),
                                              config.DefaultConnectionTimeout)
            err := client.Connect(ctx)
            cancel()

            if err != nil {
                m.logger.Warn("Health check: Failed to reconnect server",
                    zap.String("name", client.Config.Name),
                    zap.Error(err))
            } else {
                m.logger.Info("Health check: Successfully reconnected server",
                    zap.String("name", client.Config.Name))
            }
        }
    }
}
```

**Recovery Strategy:**
- Separate from per-client background monitoring
- Operates at manager level for servers with `HealthCheck` flag
- Uses standard connection timeout (60s)
- Logs success/failure for observability

### 8.4 Reconnection Protection

**Concurrent Reconnection Prevention** (`client.go:694-711`):

```go
func (mc *Client) tryReconnect() {
    mc.reconnectMu.Lock()
    if mc.reconnectInProgress {
        mc.reconnectMu.Unlock()
        mc.logger.Debug("Reconnection already in progress, skipping duplicate attempt")
        return
    }
    mc.reconnectInProgress = true
    mc.reconnectMu.Unlock()

    defer func() {
        mc.reconnectMu.Lock()
        mc.reconnectInProgress = false
        mc.reconnectMu.Unlock()
    }()

    // Perform reconnection...
}
```

**Protection Benefits:**
- Prevents duplicate Docker container spawns
- Avoids concurrent process creation
- Eliminates race conditions in cleanup

### 8.5 OAuth Event Monitoring

**Cross-Process OAuth Completion** (`manager.go:1091-1220`):

The system monitors for OAuth completion events from CLI processes:

```go
func (m *Manager) startOAuthEventMonitor() {
    ticker := time.NewTicker(config.HealthCheckInterval) // 5 seconds
    defer ticker.Stop()

    for range ticker.C {
        // Check database for OAuth completion events
        m.processOAuthEvents()

        // Scan for newly available tokens (handles DB lock scenarios)
        m.scanForNewTokens()
    }
}
```

**Event Processing:**
1. Query database for unprocessed OAuth events
2. Skip retry if server already connected/connecting
3. Trigger `RetryConnection()` for each event
4. Mark event as processed
5. Clean up old events (>24 hours)

**Token Scanning:**
1. Check persistent token store for each errored server
2. Rate-limit reconnection attempts (10s cooldown per server)
3. Trigger reconnection when new token detected
4. Avoid rapid retry loops

---

## Summary

The mcpproxy-go retry and connection system implements:

1. **Two-Phase Connection Strategy**: Initial parallel attempts + exponential backoff retries
2. **Intelligent Backoff**: Standard (1s-5m) for general errors, extended (5m-24h) for OAuth
3. **Auto-Disable Protection**: Automatic disabling after threshold failures (default: 3)
4. **Dual State Machines**: Runtime connection state + persisted server configuration state
5. **Comprehensive Health Checks**: Background monitoring every 30s with opt-in active checks
6. **Timeout Enforcement**: Context-based timeouts at every layer (60s default connection)
7. **Error Enrichment**: Context-aware error messages with actionable guidance
8. **Concurrent Protection**: Mutexes prevent duplicate reconnections and race conditions
9. **OAuth-Specific Handling**: Extended backoff, token monitoring, cross-process event handling
10. **Persistence Strategy**: Two-phase commit for config + database state synchronization

The system is designed for resilience, observability, and automatic recovery while preventing resource exhaustion and infinite retry loops.

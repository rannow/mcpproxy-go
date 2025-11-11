# Auto-Disable Implementation Status

## Overview

This document tracks the implementation of automatic server disabling after consecutive failures, as recommended in [SERVER_STARTUP_ANALYSIS.md](SERVER_STARTUP_ANALYSIS.md).

## Completed Components

### 1. Data Structures (✅ Complete)

**File**: `internal/upstream/types/types.go`

Extended `ConnectionInfo` and `StateManager` structs with new fields:

```go
type ConnectionInfo struct {
    // ... existing fields ...
    ConsecutiveFailures  int       `json:"consecutive_failures"`
    AutoDisabled         bool      `json:"auto_disabled"`
    AutoDisableReason    string    `json:"auto_disable_reason,omitempty"`
    AutoDisableThreshold int       `json:"auto_disable_threshold"`
    LastSuccessTime      time.Time `json:"last_success_time,omitempty"`
}
```

**Key Features**:
- Default threshold: 10 consecutive failures (configurable)
- Tracks consecutive failures across connection/disconnection cycles
- Preserves failure history in `Reset()` method for accurate tracking
- Records last successful connection time

### 2. State Management (✅ Complete)

**File**: `internal/upstream/types/types.go`

Added helper methods:

1. `ShouldAutoDisable()` - Check if consecutive failures exceed threshold
2. `SetAutoDisabled(reason string)` - Mark server as auto-disabled
3. `IsAutoDisabled()` - Check if auto-disabled
4. `GetAutoDisableReason()` - Get disable reason
5. `SetAutoDisableThreshold(threshold int)` - Configure threshold
6. `GetConsecutiveFailures()` - Get failure count
7. `ResetAutoDisable()` - Clear auto-disable state for manual re-enable

**Integration Points**:
- `SetError()` increments consecutive failures on each error
- `TransitionTo(StateReady)` resets consecutive failures on success
- `Reset()` preserves failure tracking (does NOT reset consecutiveFailures)

### 3. Health Check Integration (✅ Complete)

**File**: `internal/upstream/managed/client.go`

Added auto-disable check in `performHealthCheck()`:

```go
func (mc *Client) performHealthCheck() {
    // Check if server should be auto-disabled due to consecutive failures
    if mc.StateManager.ShouldAutoDisable() {
        info := mc.StateManager.GetConnectionInfo()
        reason := fmt.Sprintf("Server automatically disabled after %d consecutive failures (threshold: %d)",
            info.ConsecutiveFailures, info.AutoDisableThreshold)

        mc.StateManager.SetAutoDisabled(reason)
        mc.logger.Warn("Server auto-disabled due to consecutive failures",
            zap.String("server", mc.Config.Name),
            zap.Int("consecutive_failures", info.ConsecutiveFailures),
            zap.Int("threshold", info.AutoDisableThreshold),
            zap.String("reason", reason))

        // TODO: Trigger configuration update to persist auto-disable state
        // TODO: Generate failure report with diagnostic information

        return
    }

    // Skip health checks if server is already auto-disabled
    if mc.StateManager.IsAutoDisabled() {
        return
    }

    // ... rest of health check logic ...
}
```

**Behavior**:
- Checks threshold before each retry attempt
- Logs warning when server is auto-disabled
- Stops health checks for auto-disabled servers
- Prevents resource waste on repeatedly failing servers

## Pending Components

### 4. Configuration Persistence (⏳ In Progress)

**Requirement**: Update `mcp_config.json` to set `enabled: false` when server is auto-disabled

**Implementation Plan**:

```go
// Add auto-disable callback to managed.Client
type Client struct {
    // ... existing fields ...
    onAutoDisable func(serverName string, reason string) // New callback
}

// In upstream/manager.go - AddServerConfig()
client.SetAutoDisableCallback(func(serverName string, reason string) {
    // 1. Update server config in storage
    cfg := m.getServerConfig(serverName)
    cfg.Enabled = false
    m.storage.UpdateUpstreamServer(serverName, cfg)

    // 2. Trigger server config save via event bus or callback
    if m.server != nil {
        m.server.SaveConfiguration()
        m.server.OnUpstreamServerChange()
    }
})
```

**Files to Modify**:
- `internal/upstream/managed/client.go` - Add callback field and setter
- `internal/upstream/manager.go` - Set up callback in `AddServerConfig()`
- `internal/server/server.go` - Ensure `SaveConfiguration()` is called

### 5. Failure Report Generation (⏳ Pending)

**Requirement**: Generate detailed diagnostic report when server is auto-disabled

**Implementation Plan**:

```go
type FailureReport struct {
    ServerName           string                 `json:"server_name"`
    Protocol             string                 `json:"protocol"`
    Command              string                 `json:"command,omitempty"`
    URL                  string                 `json:"url,omitempty"`
    ConsecutiveFailures  int                    `json:"consecutive_failures"`
    Threshold            int                    `json:"threshold"`
    AutoDisableTime      time.Time              `json:"auto_disable_time"`
    AutoDisableReason    string                 `json:"auto_disable_reason"`
    LastSuccessTime      time.Time              `json:"last_success_time,omitempty"`
    ErrorHistory         []FailureInstance      `json:"error_history"`
    DiagnosticInfo       map[string]interface{} `json:"diagnostic_info"`
}

type FailureInstance struct {
    Timestamp       time.Time `json:"timestamp"`
    Error           string    `json:"error"`
    ErrorType       string    `json:"error_type"`
    RetryCount      int       `json:"retry_count"`
    IsOAuthError    bool      `json:"is_oauth_error"`
}
```

**Features**:
- Error history with timestamps and types
- Diagnostic information:
  - Missing packages detection (npm, pip, etc.)
  - Timeout analysis
  - OAuth-specific issues
  - Docker container failures
  - Network connectivity problems
- Storage in BBolt database
- Retrieval via MCP tool or web UI

**Files to Create/Modify**:
- `internal/upstream/types/failure_report.go` - Report structures
- `internal/upstream/managed/client.go` - Report generation in auto-disable path
- `internal/storage/failure_reports.go` - BBolt storage for reports
- `internal/server/mcp.go` - MCP tool for retrieving reports

### 6. Package/Dependency Detection (⏳ Pending)

**Requirement**: Detect missing packages and dependencies in error messages

**Implementation Plan**:

```go
// Error pattern detection
var packageErrorPatterns = map[string]*regexp.Regexp{
    "npm": regexp.MustCompile(`(?i)(npm|node|package\.json|not found|ENOENT|Cannot find module)`),
    "pip": regexp.MustCompile(`(?i)(pip|python|requirements\.txt|ModuleNotFoundError|ImportError)`),
    "docker": regexp.MustCompile(`(?i)(docker|container|image not found|pull access denied)`),
    "git": regexp.MustCompile(`(?i)(git|repository|clone failed|not a git repository)`),
    "oauth": regexp.MustCompile(`(?i)(oauth|authorization|token|401|unauthorized|forbidden)`),
    "network": regexp.MustCompile(`(?i)(connection refused|timeout|network|dial tcp|ECONNREFUSED)`),
}

func analyzeError(err error) DiagnosticInfo {
    errStr := err.Error()
    info := DiagnosticInfo{
        ErrorType: "unknown",
        Suggestions: []string{},
    }

    for errorType, pattern := range packageErrorPatterns {
        if pattern.MatchString(errStr) {
            info.ErrorType = errorType
            info.Suggestions = getSuggestions(errorType, errStr)
            break
        }
    }

    return info
}

func getSuggestions(errorType string, errorMessage string) []string {
    switch errorType {
    case "npm":
        return []string{
            "Ensure Node.js and npm are installed",
            "Run 'npm install' in the working directory",
            "Check package.json for correct package names",
        }
    case "pip":
        return []string{
            "Ensure Python and pip are installed",
            "Run 'pip install -r requirements.txt'",
            "Check if the module is available in PyPI",
        }
    // ... more cases
    }
    return []string{}
}
```

**Files to Create/Modify**:
- `internal/upstream/diagnostics/error_analyzer.go` - Error pattern analysis
- `internal/upstream/managed/client.go` - Call analyzer in error handling
- `internal/upstream/types/failure_report.go` - Include diagnostics in report

## Testing Plan

### Unit Tests

1. **StateManager Auto-Disable Logic**
   - Test `ShouldAutoDisable()` threshold checking
   - Test `SetAutoDisabled()` state changes
   - Test consecutive failure tracking across disconnections
   - Test threshold configuration

2. **Managed Client Health Check**
   - Test auto-disable triggers after threshold
   - Test health check skipping for auto-disabled servers
   - Test failure reset on successful connection

### Integration Tests

1. **End-to-End Auto-Disable Flow**
   - Start server with failing upstream
   - Verify consecutive failure increments
   - Verify auto-disable after threshold
   - Verify configuration persistence
   - Verify failure report generation

2. **Recovery Testing**
   - Test manual re-enable after auto-disable
   - Test successful connection after re-enable
   - Test failure counter reset on success

## Configuration

### Default Threshold

The default auto-disable threshold is **10 consecutive failures**, set in `NewStateManager()`:

```go
func NewStateManager() *StateManager {
    return &StateManager{
        currentState:         StateDisconnected,
        autoDisableThreshold: 10, // Default threshold
    }
}
```

### Customization

Users can customize the threshold via:

1. **Environment Variable** (future):
   ```bash
   export MCPPROXY_AUTO_DISABLE_THRESHOLD=5
   ```

2. **Configuration File** (future):
   ```json
   {
     "auto_disable_threshold": 5,
     "mcpServers": [...]
   }
   ```

3. **Per-Server Configuration** (future):
   ```json
   {
     "mcpServers": [
       {
         "name": "flaky-server",
         "auto_disable_threshold": 20,
         ...
       }
     ]
   }
   ```

## Benefits

1. **Resource Efficiency**: Prevents wasted retry attempts on persistently failing servers
2. **System Stability**: Reduces log noise and resource consumption
3. **Operational Visibility**: Clear indication of problematic servers
4. **Diagnostic Value**: Failure reports aid troubleshooting
5. **Graceful Degradation**: System continues functioning with remaining healthy servers

## Next Steps

1. ✅ Complete basic auto-disable implementation
2. ⏳ Add configuration persistence callback
3. ⏳ Implement failure report generation
4. ⏳ Add package/dependency error detection
5. Add comprehensive unit tests
6. Add integration tests
7. Update user documentation

## Related Documents

- [SERVER_STARTUP_ANALYSIS.md](SERVER_STARTUP_ANALYSIS.md) - Original analysis and recommendations
- [CLAUDE.md](../CLAUDE.md) - Project overview and development guidelines

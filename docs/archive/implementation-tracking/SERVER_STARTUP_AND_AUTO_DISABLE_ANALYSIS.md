# Server Startup and Auto-Disable System Analysis

## Executive Summary

Comprehensive analysis of MCPProxy's server startup, activation, and auto-disable mechanisms, including findings and recommendations for improved failure tracking and reporting.

## Current System Analysis

### 1. Server Startup and Activation ✅ WORKING CORRECTLY

**Location**: `internal/server/server.go:597-659`

**Process Flow**:
1. `loadConfiguredServers()` called during server initialization
2. Reads all servers from config (`s.config.Servers`)
3. **Filters out disabled servers** (line 635-639):
   ```go
   if cfg.Enabled || cfg.Quarantined {
       // Add to upstream manager
   } else {
       s.upstreamManager.RemoveServer(serverCfg.Name)
       s.logger.Info("Server is disabled, removing from active connections")
   }
   ```
4. Only enabled servers are added to upstream manager
5. Parallel connection attempts with semaphore (max 20 concurrent)

**Verification**: ✅ **CONFIRMED - Only enabled servers are started**
- Disabled servers explicitly removed from upstream manager
- Log message confirms: "Server is disabled, removing from active connections"
- Quarantined servers kept connected but execution blocked

---

### 2. Auto-Disable Implementation ✅ PARTIALLY IMPLEMENTED

**Location**: `internal/upstream/managed/client.go:481-514`

**Current Behavior**:
- **Threshold**: 10 consecutive failures (configurable via `AutoDisableThreshold`)
- **Trigger Point**: Health check loop (`performHealthCheck()`)
- **Actions on Auto-Disable**:
  1. ✅ Sets auto-disabled flag in state manager
  2. ✅ Logs warning to main log
  3. ✅ **Writes to `failed_servers.log`** (lines 495-501)
  4. ✅ **Persists `enabled: false` to config** (via callback lines 503-506)
  5. ✅ Stops health checks for disabled server

**Failure Tracking**:
- `SetError()` increments `consecutiveFailures` counter (types.go:215)
- Counter reset on successful connection (types.go:174)
- Counter preserved across disconnection/reconnection cycles

---

### 3. Startup Failure Handling ⚠️ NEEDS ENHANCEMENT

**Current Behavior**:
- **Initial Connection**: `manager.go:228-249` calls `Connect()`
- **Connection Errors**: Logged but NOT counted toward auto-disable
- **Health Check**: Only monitors AFTER initial connection established

**Gap Identified**:
- Startup failures during initial `Connect()` do NOT increment `consecutiveFailures`
- Only errors during health checks trigger auto-disable
- Server that fails 10 times at startup will keep retrying indefinitely

**Why This Happens**:
```go
// manager.go:247 - Startup connection
if err := client.Connect(ctx); err != nil {
    return fmt.Errorf("failed to connect: %w", err)
    // ❌ Does NOT call SetError() - failure not counted
}

// managed/client.go:120 - During Connect()
} else {
    mc.StateManager.SetError(err)  // ✅ Sets error
}
// ❌ But parent AddServer() just logs and returns error
```

---

### 4. Failed Servers Log Format and Content ✅ FUNCTIONAL, ⚠️ NEEDS ENHANCEMENT

**Current Log Format** (`internal/logs/failure_logger.go:27-29`):
```
timestamp [LEVEL] Server "name" failed: reason
```

**Example**:
```
2025-11-09 23:19:28	[ERROR]	Server "test-server" failed: Server automatically disabled after 10 consecutive failures (threshold: 10)
```

**Current Limitations**:
1. ❌ **Generic reason** - Only says "X consecutive failures"
2. ❌ **No error details** - Doesn't capture WHAT failed (timeout, missing package, OAuth, etc.)
3. ❌ **No categorization** - Can't filter by error type
4. ❌ **No timestamps for first failure** - Can't determine failure duration
5. ✅ **HTML rendering works** - Displayed correctly at `/failed-servers`

---

### 5. Log Backup and Cleanup ⚠️ NOT IMPLEMENTED

**Current State**:
- ❌ **No backup on restart** - Log file persists across restarts
- ❌ **No cleanup** - Old entries accumulate indefinitely
- ❌ **No rotation** - Single file grows without bounds

**What Should Happen** (per user request):
1. On startup: Backup existing `failed_servers.log` to timestamped file
2. Clear/truncate the main log file
3. Optionally keep last N backups (e.g., 5)

---

## Recommended Improvements

### Priority 1: Enhanced Error Logging (HIGH)

**Goal**: Capture detailed diagnostic information for each failure

**Implementation**:
```go
// internal/logs/failure_logger.go - Enhanced log format
type FailureLogEntry struct {
    Timestamp    string
    ServerName   string
    ErrorType    string // "timeout", "missing_package", "oauth", "config", "network"
    ErrorMessage string
    FailureCount int
    FirstFailure string // When failures started
    Details      map[string]string // Additional context
}

func LogServerFailureDetailed(dataDir, serverName string, info *types.ConnectionInfo) error {
    // Analyze last error to categorize
    errorType, suggestions := categorizeError(info.LastError)

    // Format: timestamp [ERROR] Server "name" | Type: <type> | Count: <n> | Error: <msg> | Suggestions: <hints>
    logLine := fmt.Sprintf("%s\t[ERROR]\tServer \"%s\" | Type: %s | Count: %d | First: %s | Error: %s | Suggestions: %s\n",
        timestamp, serverName, errorType, info.ConsecutiveFailures,
        info.FirstAttemptTime.Format("2006-01-02 15:04:05"),
        info.LastError.Error(), strings.Join(suggestions, "; "))
}

func categorizeError(err error) (string, []string) {
    if err == nil {
        return "unknown", []string{}
    }

    errStr := strings.ToLower(err.Error())

    // Timeout errors
    if strings.Contains(errStr, "timeout") || strings.Contains(errStr, "deadline exceeded") {
        return "timeout", []string{
            "Check if server process starts correctly",
            "Increase timeout in configuration",
            "Verify network connectivity",
        }
    }

    // Missing package errors
    if strings.Contains(errStr, "cannot find module") ||
       strings.Contains(errStr, "modulenotfounderror") ||
       strings.Contains(errStr, "command not found") {
        return "missing_package", []string{
            "Run 'npm install' or 'pip install' in working directory",
            "Verify package.json or requirements.txt exists",
            "Check if npx/uvx is installed",
        }
    }

    // OAuth errors
    if strings.Contains(errStr, "oauth") ||
       strings.Contains(errStr, "unauthorized") ||
       strings.Contains(errStr, "401") {
        return "oauth", []string{
            "Run: mcpproxy auth login --server=" + serverName,
            "Check API token is valid",
            "Verify OAuth configuration",
        }
    }

    // Configuration errors
    if strings.Contains(errStr, "config") ||
       strings.Contains(errStr, "invalid") ||
       strings.Contains(errStr, "missing required") {
        return "config", []string{
            "Verify server configuration in mcp_config.json",
            "Check required environment variables are set",
            "Review server documentation for setup requirements",
        }
    }

    // Network errors
    if strings.Contains(errStr, "connection refused") ||
       strings.Contains(errStr, "network") ||
       strings.Contains(errStr, "dial tcp") {
        return "network", []string{
            "Check server URL is correct",
            "Verify firewall settings",
            "Test network connectivity",
        }
    }

    return "unknown", []string{"Check server logs for details"}
}
```

---

### Priority 2: Startup Failure Tracking (HIGH)

**Goal**: Count initial connection failures toward auto-disable threshold

**Implementation**:
```go
// internal/upstream/manager.go:228-249 - Modified AddServer()
func (m *Manager) AddServer(id string, serverConfig *config.ServerConfig) error {
    // ... existing code ...

    if !serverConfig.Enabled {
        m.logger.Debug("Skipping connection for disabled server")
        return nil
    }

    // Check if client exists and is already connected
    if client, exists := m.GetClient(id); exists {
        if client.IsConnected() {
            return nil
        }

        // Connect to server with timeout
        ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
        defer cancel()
        if err := client.Connect(ctx); err != nil {
            // ✅ NEW: Log startup failure for tracking
            dataDir := m.getDataDir() // Get from config
            if failureErr := logs.LogServerFailureDetailed(dataDir, serverConfig.Name, client.StateManager.GetConnectionInfo()); failureErr != nil {
                m.logger.Error("Failed to log startup failure", zap.Error(failureErr))
            }

            // ✅ NEW: Check if should auto-disable after startup failure
            if client.StateManager.ShouldAutoDisable() {
                info := client.StateManager.GetConnectionInfo()
                reason := fmt.Sprintf("Server failed to start %d times (threshold: %d)",
                    info.ConsecutiveFailures, info.AutoDisableThreshold)
                client.StateManager.SetAutoDisabled(reason)

                // Persist disabled state
                if m.onServerAutoDisable != nil {
                    m.onServerAutoDisable(serverConfig.Name, reason)
                }

                m.logger.Warn("Server auto-disabled after startup failures",
                    zap.String("server", serverConfig.Name),
                    zap.Int("failures", info.ConsecutiveFailures))
            }

            return fmt.Errorf("failed to connect to server %s: %w", serverConfig.Name, err)
        }
    }

    return nil
}
```

---

### Priority 3: Log Backup and Rotation (MEDIUM)

**Goal**: Backup and clean log on startup, maintain history

**Implementation**:
```go
// internal/logs/failure_logger.go - Add backup function

func BackupAndClearFailureLog(dataDir string) error {
    if dataDir == "" {
        dataDir = filepath.Join(os.Getenv("HOME"), ".mcpproxy")
    }

    logPath := filepath.Join(dataDir, "failed_servers.log")

    // Check if log exists
    if _, err := os.Stat(logPath); os.IsNotExist(err) {
        // No log to backup
        return nil
    }

    // Create backup with timestamp
    timestamp := time.Now().Format("20060102-150405")
    backupPath := filepath.Join(dataDir, fmt.Sprintf("failed_servers.backup.%s.log", timestamp))

    // Copy current log to backup
    input, err := os.ReadFile(logPath)
    if err != nil {
        return fmt.Errorf("failed to read log for backup: %w", err)
    }

    if err := os.WriteFile(backupPath, input, 0644); err != nil {
        return fmt.Errorf("failed to create backup: %w", err)
    }

    // Truncate original log
    if err := os.Truncate(logPath, 0); err != nil {
        return fmt.Errorf("failed to truncate log: %w", err)
    }

    // Clean up old backups (keep last 5)
    if err := cleanOldBackups(dataDir, 5); err != nil {
        // Log but don't fail
        fmt.Printf("Warning: failed to clean old backups: %v\n", err)
    }

    return nil
}

func cleanOldBackups(dataDir string, keepCount int) error {
    files, err := filepath.Glob(filepath.Join(dataDir, "failed_servers.backup.*.log"))
    if err != nil {
        return err
    }

    // Sort by modification time (oldest first)
    sort.Slice(files, func(i, j int) bool {
        statI, _ := os.Stat(files[i])
        statJ, _ := os.Stat(files[j])
        return statI.ModTime().Before(statJ.ModTime())
    })

    // Remove oldest backups if exceeding keepCount
    if len(files) > keepCount {
        for _, file := range files[:len(files)-keepCount] {
            os.Remove(file)
        }
    }

    return nil
}

// Call from server startup (server.go:backgroundInitialization)
func (s *Server) backgroundInitialization() {
    // Backup and clear failed servers log on startup
    if err := logs.BackupAndClearFailureLog(s.config.DataDir); err != nil {
        s.logger.Warn("Failed to backup/clear failure log", zap.Error(err))
    }

    // ... rest of initialization ...
}
```

---

### Priority 4: Enhanced HTML Display (LOW)

**Goal**: Better visualization with error type filtering and suggestions

**Implementation**: Update `internal/server/failed_servers.go`
- Add color-coded error type badges
- Display categorized suggestions
- Add filter buttons for error types
- Show failure count and duration
- Link to detailed server logs

---

## Testing Plan

### 1. Startup Failure Test
```bash
# Create server config with invalid command
# Restart mcpproxy 10 times
# Verify: Server auto-disabled and logged
```

### 2. Runtime Failure Test
```bash
# Start server successfully
# Force 10 consecutive health check failures
# Verify: Server auto-disabled and logged with details
```

### 3. Log Backup Test
```bash
# Create failed_servers.log with entries
# Restart mcpproxy
# Verify: Backup created, original cleared
# Verify: Old backups cleaned (keep 5)
```

### 4. Error Categorization Test
```bash
# Test each error type:
# - Timeout (slow server)
# - Missing package (invalid npm module)
# - OAuth (no token)
# - Config (missing env var)
# - Network (invalid URL)
# Verify: Correct categorization and suggestions
```

---

## Implementation Priority

1. **HIGH** - Enhanced error logging with categorization
2. **HIGH** - Startup failure tracking toward auto-disable
3. **MEDIUM** - Log backup and rotation on startup
4. **LOW** - Enhanced HTML display with filtering

---

## Summary of Findings

| Component | Status | Notes |
|-----------|--------|-------|
| Only enabled servers start | ✅ WORKING | Correctly filters disabled servers |
| Auto-disable after failures | ✅ WORKING | Triggers at threshold (default 10) |
| Failed servers logged | ✅ WORKING | Writes to failed_servers.log |
| Config persistence | ✅ WORKING | Sets enabled=false on auto-disable |
| HTML display | ✅ WORKING | Renders log entries correctly |
| Detailed error info | ⚠️ NEEDS WORK | Only generic "X failures" message |
| Error categorization | ❌ MISSING | No error type detection |
| Startup failure tracking | ⚠️ PARTIAL | Health checks only, not initial connect |
| Log backup on restart | ❌ MISSING | No backup/rotation implemented |
| Suggestion system | ❌ MISSING | No diagnostic hints provided |

---

## Next Steps

1. Implement enhanced error logging with categorization
2. Add startup failure tracking to auto-disable logic
3. Implement log backup and rotation
4. Update HTML display with enhanced information
5. Add comprehensive tests for all scenarios
6. Update user documentation

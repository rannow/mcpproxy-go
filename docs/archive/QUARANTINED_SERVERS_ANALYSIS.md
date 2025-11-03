# Quarantined Servers Analysis & Recovery Plan

**Analysis Date**: 2025-10-31
**Total Quarantined Servers**: 4
**Status**: All failures caused by concurrent startup overload (Oct 17)

---

## Executive Summary

**Good News**: None of the quarantined servers have actual security issues or malicious behavior. All 4 servers were auto-quarantined due to repeated connection failures during the concurrent startup overload period (Oct 17, 2025).

**Root Cause**: Resource exhaustion from attempting to start 20+ NPX processes simultaneously:
- `fork/exec /bin/zsh: resource temporarily unavailable` (process limit exceeded)
- `context deadline exceeded` (30s timeout insufficient under load)

**Recommendation**: Safe to unquarantine all 4 servers now that concurrency has been reduced from 20 to 5.

---

## Quarantined Servers Analysis

### 1. bigquery-ergut
**Package**: `@ergut/mcp-bigquery-server`
**Status**: ❌ Disabled & Quarantined
**Failure Pattern**: Resource exhaustion → timeout failures
**Security Risk**: ✅ None detected

**Error Timeline**:
- Oct 17, 17:45: Fork/exec failures (process limit)
- Oct 17, 20:01-22:53: Multiple timeout failures (30s)

**Recovery Assessment**: ✅ **SAFE TO ENABLE**
- No malicious behavior detected
- Official MCP package from ergut
- Failures consistent with system resource exhaustion
- Note: Alternative `bigquery-lucashild` server already enabled and working

**Recommendation**:
- Option A: Enable and test (may conflict with bigquery-lucashild)
- Option B: Keep disabled (redundant with bigquery-lucashild)

---

### 2. gdrive
**Package**: `@modelcontextprotocol/server-gdrive`
**Status**: ❌ Disabled & Quarantined
**Failure Pattern**: Resource exhaustion → timeout failures
**Security Risk**: ✅ None detected

**Error Timeline**:
- Oct 17, 17:45: Fork/exec failures (process limit)
- Oct 17, 20:03-20:52: Multiple timeout failures (30s)

**Recovery Assessment**: ✅ **SAFE TO ENABLE**
- Official Anthropic MCP server
- No security concerns
- Failures consistent with system resource exhaustion
- Note: `gdrive-server` is duplicate configuration

**Recommendation**: Enable ONE of gdrive/gdrive-server (same package)

---

### 3. gdrive-server
**Package**: `@modelcontextprotocol/server-gdrive`
**Status**: ❌ Disabled & Quarantined
**Failure Pattern**: Resource exhaustion → timeout failures
**Security Risk**: ✅ None detected

**Configuration Difference**:
- gdrive: uses `npx -y` flag
- gdrive-server: uses `npx --yes` flag
- Same package, different NPX flag syntax

**Recovery Assessment**: ✅ **SAFE TO ENABLE**
- Duplicate of `gdrive` server
- Official Anthropic MCP server
- No security concerns

**Recommendation**:
- Enable ONLY ONE: Choose either `gdrive` OR `gdrive-server`
- Prefer `gdrive` (shorter name, `-y` is more common)
- Delete the duplicate to avoid confusion

---

### 4. test-weather-server
**Package**: `@modelcontextprotocol/server-weather`
**Status**: ❌ Disabled & Quarantined
**Failure Pattern**: Resource exhaustion → timeout failures
**Security Risk**: ✅ None detected

**Error Timeline**: Similar pattern to other servers

**Recovery Assessment**: ✅ **SAFE TO ENABLE**
- Official Anthropic weather demo server
- No security concerns
- Likely used for testing purposes

**Recommendation**:
- Enable if weather functionality needed
- Otherwise keep disabled (test/demo server)

---

## Recovery Plan

### Immediate Actions (Safe to Execute Now)

#### Step 1: Remove Duplicate gdrive Configuration
**Action**: Delete `gdrive-server` (keep `gdrive`)
**Rationale**: Identical package, avoid confusion and resource waste
**Risk**: None (duplicate configuration)

#### Step 2: Unquarantine Safe Servers
**Action**: Set `quarantined: false` for:
- ✅ gdrive (primary Google Drive integration)
- ⚠️ bigquery-ergut (if needed, but conflicts with bigquery-lucashild)
- ⚠️ test-weather-server (demo/test server, low priority)

#### Step 3: Enable Based on Need
**High Priority**:
- `gdrive`: Enable for Google Drive functionality

**Low Priority** (enable only if needed):
- `bigquery-ergut`: Only if bigquery-lucashild insufficient
- `test-weather-server`: Only for weather feature testing

---

## Automated Recovery Script

### Option A: Conservative (Enable gdrive only)
```bash
#!/bin/bash
CONFIG_FILE="$HOME/.mcpproxy/mcp_config.json"
BACKUP_DIR="$HOME/.mcpproxy/backups"
mkdir -p "$BACKUP_DIR"

# Backup current config
cp "$CONFIG_FILE" "$BACKUP_DIR/config-before-unquarantine-$(date +%Y%m%d-%H%M%S).json"

# Remove gdrive-server duplicate
jq 'del(.mcpServers[] | select(.name == "gdrive-server"))' "$CONFIG_FILE" > "$CONFIG_FILE.tmp"
mv "$CONFIG_FILE.tmp" "$CONFIG_FILE"

# Unquarantine and enable gdrive
jq '.mcpServers |= map(
  if .name == "gdrive"
  then .quarantined = false | .enabled = true
  else .
  end
)' "$CONFIG_FILE" > "$CONFIG_FILE.tmp"
mv "$CONFIG_FILE.tmp" "$CONFIG_FILE"

echo "✅ Unquarantined and enabled gdrive"
echo "✅ Removed duplicate gdrive-server"
echo "📁 Backup: $BACKUP_DIR/config-before-unquarantine-*.json"
```

### Option B: Aggressive (Enable all quarantined servers)
```bash
#!/bin/bash
CONFIG_FILE="$HOME/.mcpproxy/mcp_config.json"
BACKUP_DIR="$HOME/.mcpproxy/backups"
mkdir -p "$BACKUP_DIR"

# Backup current config
cp "$CONFIG_FILE" "$BACKUP_DIR/config-before-unquarantine-all-$(date +%Y%m%d-%H%M%S).json"

# Remove gdrive-server duplicate
jq 'del(.mcpServers[] | select(.name == "gdrive-server"))' "$CONFIG_FILE" > "$CONFIG_FILE.tmp"
mv "$CONFIG_FILE.tmp" "$CONFIG_FILE"

# Unquarantine all servers EXCEPT gdrive-server (deleted)
jq '.mcpServers |= map(
  if .quarantined == true and .name != "gdrive-server"
  then .quarantined = false | .enabled = true
  else .
  end
)' "$CONFIG_FILE" > "$CONFIG_FILE.tmp"
mv "$CONFIG_FILE.tmp" "$CONFIG_FILE"

echo "✅ Unquarantined all servers:"
echo "  - gdrive"
echo "  - bigquery-ergut"
echo "  - test-weather-server"
echo "✅ Removed duplicate gdrive-server"
echo "📁 Backup: $BACKUP_DIR/config-before-unquarantine-all-*.json"
```

---

## Security Verification

### Why These Servers Are Safe

1. **All Official Packages**:
   - `@modelcontextprotocol/*`: Official Anthropic packages
   - `@ergut/mcp-bigquery-server`: Reputable third-party package

2. **No Tool Poisoning Indicators**:
   - No hidden instructions in tool descriptions
   - No data exfiltration attempts detected
   - Standard MCP protocol compliance

3. **Failure Pattern Analysis**:
   - All failures occurred during Oct 17 resource exhaustion event
   - Same error patterns across all servers (fork/exec → timeout)
   - No server-specific anomalies or suspicious behavior

4. **Quarantine Reason**: Auto-quarantine after 6+ consecutive failures
   - Not quarantined due to security analysis
   - Not quarantined due to malicious behavior
   - Quarantined as safety precaution after repeated failures

---

## Post-Recovery Testing

After unquarantining servers, verify:

1. **Connection Success**:
   ```bash
   tail -f ~/Library/Logs/mcpproxy/main.log | grep -E "(gdrive|bigquery-ergut|test-weather)"
   ```

2. **No Timeout Errors** (with concurrency=5, should succeed):
   ```bash
   grep "context deadline exceeded" ~/Library/Logs/mcpproxy/server-gdrive.log | tail -5
   ```

3. **Tool Discovery**:
   ```bash
   ./mcpproxy tools list --server=gdrive
   ```

---

## Recommendations Summary

### ✅ Safe to Execute Immediately
1. **Remove duplicate**: Delete `gdrive-server` configuration
2. **Unquarantine gdrive**: Safe, official Anthropic package
3. **Enable gdrive**: Restore Google Drive functionality

### ⚠️ Evaluate Before Enabling
1. **bigquery-ergut**: Check if `bigquery-lucashild` sufficient
2. **test-weather-server**: Enable only if weather testing needed

### 📊 Expected Outcome
- **Before**: 4 quarantined servers, 0% Google Drive access
- **After**: 0-1 quarantined servers, Google Drive functional
- **Startup time**: Still ~60-90s (concurrency=5 working correctly)
- **Success rate**: 90%+ (vs 0% during Oct 17 overload)

---

## Conclusion

**All quarantined servers are safe to unquarantine.** The quarantine was triggered by system resource exhaustion during concurrent startup overload, not by security concerns or malicious behavior.

**Recommended Action**: Execute Option A (conservative) recovery script to restore Google Drive functionality while removing the duplicate configuration.

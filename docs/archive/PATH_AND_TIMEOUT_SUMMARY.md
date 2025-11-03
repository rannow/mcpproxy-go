# PATH Configuration and Timeout Issues Summary

**Date**: 2025-10-31 18:08
**Status**: PATH fixes applied ✅ | Timeout issues remain ⏳

---

## Completed PATH Fixes

### ✅ UVX Servers (27 servers)
**Issue**: `uvx` command not found - located at `~/.local/bin/uvx` but not in daemon PATH
**Fix Applied**: Added comprehensive PATH to all 27 uvx servers via `scripts/fix-uvx-path.sh`
**Result**: UVX servers now starting successfully (verified in logs)

**Example Working Server**:
- `mcp-imap-server` - Successfully initialized
- AWS Labs servers - Large active log files (8-9MB)

### ✅ PIPX Servers (4 servers) - PATH Fixed, Timeout Remains
**Issue**: `pipx` command not found - located at `/opt/homebrew/bin/pipx` but not in daemon PATH
**Fix Applied**: Added comprehensive PATH to all 4 pipx servers via `scripts/fix-pipx-path.sh`
**Current Status**: PATH resolved, but servers still timing out (see Timeout Issues section)

**Servers**:
1. bigquery-lucashild - `pipx run mcp-server-bigquery`
2. duckdb-ktanaka - `pipx run mcp-server-duckdb`
3. motherduck-duckdb - `pipx run mcp-server-motherduck`
4. toolfront-database - `pipx run toolfront`

### PATH Configuration Applied
```bash
PATH="/Users/hrannow/.local/bin:/Users/hrannow/.amplify/bin:/Users/hrannow/.pyenv/bin:/Users/hrannow/.codeium/windsurf/bin:/opt/homebrew/bin:/opt/homebrew/sbin:/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin:/Users/hrannow/.rvm/bin"
```

**Includes**:
- `~/.local/bin` - uvx, uv, user Python tools
- `~/.amplify/bin` - AWS Amplify CLI
- `~/.pyenv/bin` - Python version manager
- `~/.codeium/windsurf/bin` - Windsurf IDE tools
- `/opt/homebrew/bin` - Homebrew packages (pipx, npm, etc.)
- `/opt/homebrew/sbin` - Homebrew system binaries
- Standard system paths
- `~/.rvm/bin` - Ruby version manager

---

## Remaining Timeout Issues

### ⏳ PIPX Servers - Package Installation Timeout

**Root Cause**: `pipx run` downloads and installs packages on-demand if not already installed. This can take >60 seconds on first execution.

**Error Pattern**:
```
MCP initialize JSON-RPC call failed
Error: transport error: context deadline exceeded
```

**Similar to NPX Issue**: Same problem as NPX-based servers - need pre-installation.

### Solution Options for PIPX Servers

#### Option 1: Pre-install PIPX Packages (Recommended)
```bash
# Install packages globally with pipx
pipx install mcp-server-bigquery
pipx install mcp-server-duckdb
pipx install mcp-server-motherduck
pipx install toolfront

# Verify installations
pipx list
```

**Benefits**:
- ✅ Packages cached locally, no download on connection
- ✅ Faster startup times (<5s instead of >60s)
- ✅ More reliable connections (no network dependency)

**After Installation**: Change config from `pipx run` to direct package commands:
```json
{
  "name": "bigquery-lucashild",
  "command": "mcp-server-bigquery",  // Instead of "pipx"
  "args": [],  // Remove "run mcp-server-bigquery"
  "env": { ... }
}
```

#### Option 2: Increase Connection Timeout
```json
{
  "connection_timeout": 120  // Increase from 60 to 120 seconds
}
```

**Pros**: Allows first-time package downloads
**Cons**:
- Slower startup times on first run
- Network-dependent
- Still fails if package download takes >120s

#### Option 3: Disable PIPX Servers
If these database MCP servers aren't needed:
```bash
jq '.mcpServers |= map(if .command == "pipx" then .enabled = false else . end)' \
  ~/.mcpproxy/mcp_config.json > ~/.mcpproxy/mcp_config.json.tmp
mv ~/.mcpproxy/mcp_config.json.tmp ~/.mcpproxy/mcp_config.json
```

---

## Docker Servers - Separate Issue

**Status**: ❌ Docker daemon not running

**Affected Servers** (5 total):
1. MCP_DOCKER
2. aws-mcp-server
3. crawl4ai-rag (2x duplicate entries)
4. k8s-mcp-server

**Root Cause**: Docker Desktop not running - "Cannot connect to the Docker daemon"

**Solution**: See [DOCKER_SERVERS_DIAGNOSTIC.md](DOCKER_SERVERS_DIAGNOSTIC.md) for detailed analysis

---

## Current Server Status

### Successfully Connected (64+ servers, 40%+)
- ✅ All uvx servers with PATH fix
- ✅ All npx servers (pre-installed)
- ✅ HTTP/SSE based servers
- ✅ Other stdio servers with accessible commands

### Failing Due to Timeout (4+ servers)
- ⏳ 4 pipx servers (`pipx run` needs pre-install or longer timeout)
- ⏳ Some NPX servers (if packages not pre-installed)

### Failing Due to Docker (5 servers)
- ❌ All Docker-based servers (daemon not running)

---

## Expected Improvement After PIPX Fix

| Action | Current | After Fix | Change |
|--------|---------|-----------|---------|
| **Option 1: Pre-install pipx packages** | 64 (40%) | 68 (42%) | +4 (+6%) |
| **Option 2: Increase timeout to 120s** | 64 (40%) | 68 (42%) | +4 (+6%) |
| **Option 3: Disable pipx servers** | 64 (40%) | 64 (40%) | 0 (cleaner logs) |

**Combined with Docker fix**: +9 servers total (64 → 73, 45% success rate)

---

## Scripts Created

1. **scripts/fix-uvx-path.sh** ✅ - Add PATH to uvx servers
2. **scripts/update-comprehensive-path.sh** ✅ - Upgrade to comprehensive PATH
3. **scripts/fix-pipx-path.sh** ✅ - Add PATH to pipx servers
4. **scripts/diagnose-and-recover.sh** - General diagnostic tools
5. **scripts/fix-startup-timeout.sh** - Concurrency optimization

---

## Recommendation

**Immediate Action** (< 5 minutes):
```bash
# Pre-install pipx packages
pipx install mcp-server-bigquery
pipx install mcp-server-duckdb
pipx install mcp-server-motherduck
pipx install toolfront

# Verify installations
pipx list

# Update config to use direct commands (optional but faster)
# Or keep current config and let pipx run use installed packages

# Restart mcpproxy
ps aux | grep '[m]cpproxy serve' | awk '{print $2}' | xargs kill -9
cd ~/Library/CloudStorage/OneDrive-Persönlich/workspace/mcp-server/mcpproxy-go
./mcpproxy serve
```

**Expected Result**: +4 pipx servers connecting successfully

**Medium-term** (optional):
- Start Docker Desktop for +5 Docker servers
- OR disable Docker servers if not needed

---

## Configuration Backups

All backups created before PATH modifications:
```
~/.mcpproxy/mcp_config.json.backup-before-uvx-path-fix-*
~/.mcpproxy/mcp_config.json.backup-before-pipx-path-fix-*
```

---

## Verification Commands

```bash
# Check uvx servers
tail -f ~/Library/Logs/mcpproxy/main.log | grep -E "(mcp-imap-server|awslabs)"

# Check pipx servers
tail -f ~/Library/Logs/mcpproxy/main.log | grep -E "(bigquery|duckdb|motherduck|toolfront)"

# Check Docker servers
tail -f ~/Library/Logs/mcpproxy/main.log | grep -E "(aws-mcp-server|crawl4ai|k8s-mcp)"

# Count successfully connected servers
tail -1000 ~/Library/Logs/mcpproxy/main.log | grep -c "Successfully retrieved tools"
```

---

## Summary

**PATH Fixes Complete**: ✅ 31 servers now have correct PATH (27 uvx + 4 pipx)
**UVX Servers Working**: ✅ PATH fix successful
**PIPX Servers**: ✅ PATH fixed, ⏳ awaiting package pre-installation
**Docker Servers**: ❌ Waiting for Docker Desktop or disable decision

**Next Step**: Pre-install pipx packages for +4 servers

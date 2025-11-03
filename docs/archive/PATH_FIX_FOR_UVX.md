# PATH Fix for UVX Servers

## Issue Found

✅ **`uvx` is installed**: Located at `~/.local/bin/uvx` (version 0.7.2)
❌ **`uvx` not in PATH**: The `~/.local/bin` directory is not in the PATH when mcpproxy spawns child processes
❌ **18 servers failing**: All servers using `uvx` command cannot start

## Root Cause

1. `~/.local/bin` is configured in `~/.zshrc` (line 244)
2. But when mcpproxy starts as a background daemon, it doesn't source `~/.zshrc`
3. Child processes spawned by mcpproxy inherit mcpproxy's PATH, which doesn't include `~/.local/bin`
4. Result: All `uvx` commands fail with "command not found"

## Affected Servers (18 total)

All servers using `uvx` command:
- awslabs.amazon-rekognition-mcp-server
- awslabs.cloudwatch-logs-mcp-server
- awslabs.cost-analysis-mcp-server
- awslabs.ecs-mcp-server
- cognee
- docs-mcp-server
- infinity-swiss
- markdownify-mcp
- mcp-anthropic-claude
- mcp-browser-tools
- mcp-code-executor
- mcp-openai
- mcp-perplexity
- neon-mcp-server
- ragie-mcp-server
- serena
- travel-planner-mcp-server
- video-editing-mcp

## Solutions

### Option 1: Start mcpproxy with Correct PATH (Temporary)

When starting mcpproxy manually, export PATH first:

```bash
export PATH="/Users/hrannow/.local/bin:$PATH"
cd /Users/hrannow/Library/CloudStorage/OneDrive-Persönlich/workspace/mcp-server/mcpproxy-go
./mcpproxy serve
```

⚠️ **Problem**: This only works for current session. System tray restart won't include the PATH.

### Option 2: Modify mcpproxy Config to Include PATH (Recommended)

Add PATH environment variable to each `uvx` server in `~/.mcpproxy/mcp_config.json`:

```json
{
  "name": "server-name",
  "command": "uvx",
  "args": ["package-name"],
  "env": {
    "PATH": "/Users/hrannow/.local/bin:/usr/local/bin:/usr/bin:/bin"
  }
}
```

✅ **Benefit**: Works permanently, survives restarts
❌ **Downside**: Need to update 18 server configurations

### Option 3: Create Symbolic Link (Quick Fix)

Create symlink in `/usr/local/bin` which is already in PATH:

```bash
sudo ln -s /Users/hrannow/.local/bin/uvx /usr/local/bin/uvx
sudo ln -s /Users/hrannow/.local/bin/uv /usr/local/bin/uv
```

✅ **Benefit**: Simple, permanent, no config changes needed
✅ **Benefit**: All servers automatically work
⚠️ **Risk**: Requires sudo, modifies system directories

### Option 4: Fix mcpproxy Code (Best Long-term Solution)

Modify mcpproxy to respect user's shell PATH when spawning child processes.

**Suggested Code Location**: `internal/upstream/core/` or `internal/upstream/managed/`

**Implementation**: When executing stdio commands, mcpproxy should:
1. Read user's shell PATH from `~/.zshrc` or `~/.bashrc`
2. Merge with current PATH
3. Pass to child processes via env

✅ **Benefit**: Proper solution, works for all users
❌ **Downside**: Requires code changes and testing

## Recommended Action Plan

### Immediate (< 5 minutes)
1. Create symbolic links:
```bash
sudo ln -s /Users/hrannow/.local/bin/uvx /usr/local/bin/uvx
sudo ln -s /Users/hrannow/.local/bin/uv /usr/local/bin/uv
```

2. Verify:
```bash
which uvx  # Should show /usr/local/bin/uvx
uvx --version  # Should show version 0.7.2
```

3. Restart mcpproxy:
```bash
ps aux | grep "[m]cpproxy serve" | awk '{print $2}' | xargs kill -9
cd /Users/hrannow/Library/CloudStorage/OneDrive-Persönlich/workspace/mcp-server/mcpproxy-go
./mcpproxy serve
```

4. Wait 2 minutes and check connection count:
```bash
tail -1000 ~/Library/Logs/mcpproxy/main.log | grep -c "Successfully retrieved tools"
```

**Expected Result**: +18 servers (from 64 → 82, 51% success rate)

### Short-term (Future Enhancement)
Add feature to mcpproxy to respect user's shell PATH configuration.

## Testing the Fix

After applying symlinks, test with a single uvx server:

```bash
# Test uvx command directly
uvx --help

# Check mcpproxy logs for uvx server
tail -f ~/Library/Logs/mcpproxy/server-cognee.log

# Watch for successful connection
tail -f ~/Library/Logs/mcpproxy/main.log | grep cognee
```

## Expected Impact

| Metric | Before Fix | After Fix | Change |
|--------|-----------|-----------|---------|
| Connected Servers | 64 (40%) | 82 (51%) | +18 (+28%) |
| Package Issue Servers | 18 | 0 | -18 (-100%) |
| Quick-Fixable Servers | 98 | 80 | -18 |

## Alternative: Manual Per-Server Fix Script

If you prefer not to use symlinks, I can create a script to automatically add PATH to all 18 uvx server configurations:

```bash
./scripts/fix-uvx-path.sh
```

This would modify `~/.mcpproxy/mcp_config.json` to add PATH env var to each uvx server.

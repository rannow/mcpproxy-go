# Docker MCP Servers Diagnostic Report

**Date**: 2025-10-31 18:05
**Status**: ❌ **Docker Daemon Not Running**

---

## Issue Summary

**All Docker-based MCP servers are failing** due to Docker daemon not being available.

### Error Found
```
Cannot connect to the Docker daemon at unix:///Users/hrannow/.docker/run/docker.sock
Is the docker daemon running?
```

---

## Affected Servers (5 total)

1. **MCP_DOCKER** - `docker mcp gateway run --verbose`
2. **aws-mcp-server** - `docker run -i --rm ghcr.io/alexei-led/aws-mcp-server:latest`
3. **crawl4ai-rag** (2 instances) - `docker run --rm -i mcp/crawl4ai`
4. **k8s-mcp-server** - Docker-based Kubernetes MCP server

### Error Pattern

All servers show the same timeout pattern:
```
MCP initialize JSON-RPC call failed
Error: transport error: context deadline exceeded
```

**Root Cause**: Not actually a timeout issue - servers can't start because Docker daemon isn't running.

---

## Current State

### Log Evidence
```bash
# From server-aws-mcp-server.log:
2025-10-31T17:47:40.333+01:00 | ERROR | MCP initialize JSON-RPC call failed
Error: transport error: context deadline exceeded

# Multiple retry attempts every few minutes
# All failing with same error
```

### Docker Daemon Status
```bash
$ docker run --help
Cannot connect to the Docker daemon at unix:///Users/hrannow/.docker/run/docker.sock
```

---

## Solution

### Option 1: Start Docker Desktop (Recommended)

1. **Open Docker Desktop application**
   - Location: `/Applications/Docker.app`
   - Or: Spotlight search "Docker"

2. **Wait for Docker to start**
   - Status indicator in menu bar will turn green
   - Usually takes 30-60 seconds

3. **Verify Docker is running**:
   ```bash
   docker ps
   ```
   Should show running containers (or empty list if none running)

4. **Restart mcpproxy**:
   ```bash
   ps aux | grep '[m]cpproxy serve' | awk '{print $2}' | xargs kill -9
   cd ~/Library/CloudStorage/OneDrive-Persönlich/workspace/mcp-server/mcpproxy-go
   ./mcpproxy serve
   ```

5. **Expected result**: +5 Docker servers connecting successfully

### Option 2: Disable Docker Servers (If Not Needed)

If you don't need Docker-based MCP servers, disable them in config:

```bash
jq '.mcpServers |= map(if .command == "docker" then .enabled = false else . end)' \
  ~/.mcpproxy/mcp_config.json > ~/.mcpproxy/mcp_config.json.tmp
mv ~/.mcpproxy/mcp_config.json.tmp ~/.mcpproxy/mcp_config.json
```

This will prevent retry attempts and reduce log noise.

---

## Expected Impact

### With Docker Running

| Metric | Before | After | Change |
|--------|--------|-------|---------|
| Docker Servers | 0/5 (0%) | 5/5 (100%) | +5 |
| Total Connected | ~65 (40%) | ~70 (43%) | +5 (+3%) |
| Timeout Errors | 5 servers | 0 servers | -5 |

### With Docker Disabled

| Metric | Value |
|--------|-------|
| Docker Servers | 5 disabled |
| Log Noise | Eliminated |
| Resource Usage | Reduced (no retry attempts) |

---

## Docker Server Details

### 1. aws-mcp-server
**Purpose**: AWS service integration
**Image**: `ghcr.io/alexei-led/aws-mcp-server:latest`
**Command**: `docker run -i --rm -v /Users/hrannow/.aws:/home/appuser/.aws:ro`
**Size**: ~7.6MB log file (many retry attempts)

### 2. MCP_DOCKER
**Purpose**: Docker container management via MCP
**Command**: `docker mcp gateway run --verbose`
**Note**: Requires Docker daemon for container operations

### 3. crawl4ai-rag (2 instances)
**Purpose**: Web crawling with RAG (Retrieval-Augmented Generation)
**Image**: `mcp/crawl4ai`
**Command**: `docker run --rm -i -e TRANSPORT -e OPENAI_API_KEY -e SUPABASE_URL -e SUPABASE_SERVICE_KEY`
**Note**: Duplicate entry in config (same server twice)

### 4. k8s-mcp-server
**Purpose**: Kubernetes cluster management
**Status**: Docker-based, requires daemon

---

## Why Docker Servers Take Longer

Docker-based MCP servers are inherently slower to start because they need to:

1. **Pull images** (if not cached) - can take 30-120 seconds
2. **Create container** - 2-5 seconds
3. **Start container** - 5-15 seconds
4. **Initialize MCP server inside container** - 10-30 seconds
5. **Respond to initialization request** - 1-5 seconds

**Total**: 48-177 seconds for first run, 18-55 seconds for cached images

Current timeout (60s) is **insufficient for first-time Docker image pulls**.

---

## Additional Recommendations

### If Using Docker Servers

1. **Pre-pull all Docker images** to reduce startup time:
   ```bash
   docker pull ghcr.io/alexei-led/aws-mcp-server:latest
   docker pull mcp/crawl4ai
   # ... other images
   ```

2. **Increase timeout for Docker servers** to 120s:
   - This would require code changes to allow per-server timeout configuration
   - Current timeout (60s) applies to all servers

3. **Monitor Docker resources**:
   ```bash
   docker stats
   ```

### If Not Using Docker Servers

1. **Disable Docker servers** to reduce resource usage and log noise
2. **Remove duplicate crawl4ai-rag entry** from config
3. **Consider alternatives**: Many Docker-based servers have native (NPM/Python) equivalents

---

## Technical Details

### Docker Socket Location
- **Expected**: `unix:///var/run/docker.sock` (standard)
- **macOS**: `unix:///Users/hrannow/.docker/run/docker.sock` (Docker Desktop)
- **Status**: Not accessible (daemon not running)

### Connection Attempts
- **Frequency**: Every 2-5 minutes (exponential backoff)
- **Max Retries**: Unlimited (continues until success or manual stop)
- **Impact**: Log file growth, resource usage, retry queue delays

---

## Summary

**Problem**: ❌ Docker daemon not running
**Impact**: 5 Docker servers failing (3% of total servers)
**Solution**: Start Docker Desktop OR disable Docker servers
**Expected Result**: +5 servers if Docker started, cleaner logs if disabled

**Recommendation**: **Start Docker Desktop** if you need these services, otherwise **disable** to reduce noise and resource usage.

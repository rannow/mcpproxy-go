# MCP Skills Documentation

Comprehensive documentation for the MCP Server management skills in mcpproxy-go.

## Overview

This project includes four interconnected skills for managing MCP (Model Context Protocol) servers:

| Skill | Purpose | Use When |
|-------|---------|----------|
| **MCP Server Installer** | Interactive server installation | Adding new MCP servers |
| **MCP Tool Tester** | Comprehensive functional testing | Validating server functionality |
| **MCP Server Debugger** | Diagnosis and automated repair | Fixing server issues |
| **MCP Batch Installer** | Bulk installation from tables | Installing multiple servers |

---

## Skill Ecosystem Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                     MCP Batch Installer                              │
│                  (Orchestrates all skills)                           │
│                                                                      │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐          │
│  │   Installer  │ → │    Tester    │ → │   Debugger   │           │
│  │   (Install)  │    │   (Verify)   │    │    (Fix)     │           │
│  └──────────────┘    └──────────────┘    └──────────────┘          │
└─────────────────────────────────────────────────────────────────────┘

Standalone Usage:
┌──────────────┐    ┌──────────────┐    ┌──────────────┐
│   Installer  │    │    Tester    │    │   Debugger   │
│  (Add server)│    │ (Test tools) │    │ (Fix issues) │
└──────────────┘    └──────────────┘    └──────────────┘
```

---

## 1. MCP Server Installer

**Location:** `.claude/skills/mcp-server-installer/SKILL.md`

### Purpose

Guides users through installing MCP servers interactively by:
- Identifying server type and transport method
- Searching documentation for requirements
- Checking prerequisites (Node.js, Python, Docker)
- Asking for missing configuration
- Validating before adding
- Verifying installation success

### Workflow

```
┌─────────────┐   ┌─────────────┐   ┌─────────────┐   ┌─────────────┐
│  IDENTIFY   │ → │   SEARCH    │ → │    CHECK    │ → │   GATHER    │
│ Server Type │   │    Docs     │   │   Prereqs   │   │   Config    │
└─────────────┘   └─────────────┘   └─────────────┘   └─────────────┘
                                                              ↓
┌─────────────┐   ┌─────────────┐   ┌─────────────┐
│   VERIFY    │ ← │   INSTALL   │ ← │  VALIDATE   │
│ Connection  │   │  To Proxy   │   │   Config    │
└─────────────┘   └─────────────┘   └─────────────┘
```

### Supported Transports

| Transport | Command | Example |
|-----------|---------|---------|
| **STDIO (NPX)** | `npx` | `npx -y @modelcontextprotocol/server-filesystem /tmp` |
| **STDIO (UVX)** | `uvx` | `uvx mcp-server-fetch` |
| **STDIO (Python)** | `python` | `python -m mcp_server` |
| **HTTP/HTTPS** | URL | `http://localhost:8080/mcp` |
| **SSE** | URL | `http://localhost:3000/sse` |
| **Docker** | `docker` | With isolation configuration |

### Interactive Questions

The skill uses `AskUserQuestion` to gather:
- Server type (Official, Community, Custom, HTTP)
- Transport method (STDIO, Docker, HTTP)
- Paths for filesystem servers
- API tokens for authenticated servers
- Database connection strings
- Docker isolation settings

### Quick Templates

```json
// Filesystem Server
{
  "name": "filesystem",
  "command": "npx",
  "args": ["-y", "@modelcontextprotocol/server-filesystem", "/private/tmp"]
}

// GitHub Server
{
  "name": "github",
  "command": "npx",
  "args": ["-y", "@modelcontextprotocol/server-github"],
  "env": { "GITHUB_PERSONAL_ACCESS_TOKEN": "ghp_..." }
}

// HTTP Server
{
  "name": "my-api",
  "url": "http://localhost:8080/mcp",
  "transport": "http"
}
```

---

## 2. MCP Tool Tester

**Location:** `.claude/skills/mcp-tool-tester/SKILL.md`

### Purpose

Systematically tests all tools on an MCP server using **real data operations**:
- Discovers all tools and their schemas
- Creates test data before testing
- Executes full CRUD cycles
- Verifies operations with follow-up checks
- Tests parameter validation
- Cleans up test data

### Core Principle

> **Tests must use REAL data operations, not just parameter validation!**

| ❌ Wrong Approach | ✅ Correct Approach |
|-------------------|---------------------|
| Only test missing param rejection | Create real file, read it back, verify content |
| Skip happy path if setup needed | Create test data first, then test operations |
| Mark "PASS" without execution | Execute operation, verify result, then mark |

### Test Workflow

```
┌─────────────┐   ┌─────────────┐   ┌─────────────┐   ┌─────────────┐
│  DISCOVER   │ → │    SETUP    │ → │    TEST     │ → │   VERIFY    │
│    Tools    │   │  Test Data  │   │    CRUD     │   │   Results   │
└─────────────┘   └─────────────┘   └─────────────┘   └─────────────┘
                                                              ↓
                                    ┌─────────────┐   ┌─────────────┐
                                    │   REPORT    │ ← │   CLEANUP   │
                                    │   Results   │   │  Test Data  │
                                    └─────────────┘   └─────────────┘
```

### Test Types

| Test Type | Description | Example |
|-----------|-------------|---------|
| **Happy Path** | Valid operation with real data | Create file, verify content |
| **Validation** | Missing/invalid parameters | Call without required param |
| **Verification** | Confirm mutation worked | Read after write |
| **Edge Cases** | Special characters, empty values | UTF-8 content, empty strings |

### Filesystem Server Test Pattern

```yaml
1_setup:
  - create_directory: /private/tmp/mcp-test
  - write_file: test-file.txt with "Test UTF-8: äöü €"

2_read_operations:
  - list_directory → verify files listed
  - read_file → verify content matches
  - get_file_info → verify size > 0

3_mutation_operations:
  - edit_file → then read_file to verify change
  - move_file → then read from new location

4_validation_tests:
  - read_file({}) → expect "path Required"

5_cleanup:
  - delete all test files and directories
```

### Report Format

```markdown
## Summary
| Category | Tested | Passed | Failed |
|----------|--------|--------|--------|
| Create   | 3      | 3      | 0      |
| Read     | 5      | 5      | 0      |
| Update   | 2      | 2      | 0      |
| Delete   | 1      | 1      | 0      |

## Detailed Results
| Tool | Test | Input | Output | Verified | Status |
|------|------|-------|--------|----------|--------|
| write_file | Create | {path, content} | Success | Read back ✓ | ✅ PASS |
```

---

## 3. MCP Server Debugger

**Location:** `.claude/skills/mcp-server-debugger/SKILL.md`

### Purpose

Systematically debugs MCP server issues by:
- Detecting problem type (connection, runtime, config)
- Collecting diagnostic information
- Analyzing root cause with pattern matching
- Applying automatic fixes when possible
- Guiding users through manual fixes
- Verifying fixes worked

### Debug Workflow

```
┌─────────────┐   ┌─────────────┐   ┌─────────────┐
│   DETECT    │ → │   COLLECT   │ → │   ANALYZE   │
│   Problem   │   │ Diagnostics │   │  Patterns   │
└─────────────┘   └─────────────┘   └─────────────┘
                                          ↓
┌─────────────┐   ┌─────────────┐   ┌─────────────┐
│   VERIFY    │ ← │     FIX     │ ← │  DIAGNOSE   │
│    Fix      │   │   (Auto/    │   │ Root Cause  │
│             │   │   Manual)   │   │             │
└─────────────┘   └─────────────┘   └─────────────┘
```

### Problem Categories

| Category | Symptoms | Priority |
|----------|----------|----------|
| **Connection** | State: Failed/Disconnected | Critical |
| **Runtime** | Command not found, crash | Critical |
| **Configuration** | Wrong args, missing env | High |
| **Permissions** | Access denied | High |
| **Network** | Timeout, refused | High |
| **Authentication** | 401/403 errors | Medium |

### Error Pattern Database

| Error Pattern | Root Cause | Auto-Fix? |
|---------------|------------|-----------|
| `spawn ENOENT` | Command not found | ❌ Install runtime |
| `EACCES permission denied` | No execute permission | ✅ chmod +x |
| `ECONNREFUSED` | Server not running | ❌ Start server |
| `Parent directory: /tmp` | macOS symlink issue | ✅ Use /private/tmp |
| `Cannot find module` | npm package missing | ✅ npx -y reinstall |
| `Invalid token` | Bad credentials | ❌ Ask user |

### Automatic Fixes

```python
# macOS /tmp Symlink Fix
mcp__MCPProxy__upstream_servers(
    operation: "patch",
    name: "filesystem",
    patch_json: '{"args": ["-y", "@modelcontextprotocol/server-filesystem", "/private/tmp"]}'
)

# NPX Package Reinstall
npm cache clean --force
npx -y @modelcontextprotocol/server-{name}

# Port Conflict Resolution
lsof -i :8080  # Find process
kill -9 {PID}  # Or change port in config
```

### Diagnosis Report

```markdown
## Diagnosis Report

**Server:** filesystem
**State:** Failed
**Problem Category:** Configuration

### Symptoms
- Server fails to start
- Error: Parent directory does not exist: /tmp

### Root Cause
macOS uses /tmp as symlink to /private/tmp.
The server requires the actual path.

### Evidence
```
Error: ENOENT: no such file or directory, stat '/tmp'
```

### Recommended Fix
Change path from /tmp to /private/tmp

### Auto-Fix Available?
Yes - Will apply automatically
```

---

## 4. MCP Batch Installer

**Location:** `.claude/skills/mcp-batch-installer/SKILL.md`

### Purpose

Orchestrates batch installation from a Markdown table:
- Parses Markdown table with server definitions
- Installs each server 1-by-1
- Tests each server after installation
- Attempts fixes up to 3 times
- Removes unfixable servers
- Generates detailed report
- Updates source table with status

### Workflow

```
┌─────────────┐   ┌─────────────┐   ┌─────────────┐   ┌─────────────┐
│    PARSE    │ → │   INSTALL   │ → │    TEST     │ → │     FIX     │
│    Table    │   │   Server    │   │   Server    │   │  (3 tries)  │
└─────────────┘   └─────────────┘   └─────────────┘   └─────────────┘
                                                              ↓
┌─────────────┐   ┌─────────────┐   ┌─────────────┐
│   UPDATE    │ ← │   REPORT    │ ← │   CLEANUP   │
│    Table    │   │  Generate   │   │  Unfixable  │
└─────────────┘   └─────────────┘   └─────────────┘
```

### Expected Table Format

```markdown
| Name | Transport | Command | Args | URL | Env Vars | Description |
|------|-----------|---------|------|-----|----------|-------------|
| filesystem | stdio | npx | -y @modelcontextprotocol/server-filesystem /private/tmp | | | Files |
| github | stdio | npx | -y @modelcontextprotocol/server-github | | GITHUB_TOKEN=ghp_xxx | GitHub |
| my-api | http | | | http://localhost:8080/mcp | | Custom API |
```

### Required Columns

| Column | Required | Description |
|--------|----------|-------------|
| Name | ✅ YES | Unique server identifier |
| Transport | ✅ YES | `stdio`, `http`, `sse`, `docker` |
| Command | For STDIO | Command (npx, uvx, python) |
| Args | For STDIO | Command arguments |
| URL | For HTTP | Server URL |

### Processing Flow

```python
for server in parsed_servers:
    # 1. Install
    install_result = install_server(server)

    # 2. Test
    test_result = test_server(server.name)

    # 3. Fix if needed (max 3 attempts)
    if not test_result.passed:
        for attempt in range(1, 4):
            fix_server(server.name)
            if test_server(server.name).passed:
                break

    # 4. Remove if unfixable
    if not test_result.passed:
        remove_server(server.name)
```

### Status Values

| Status | Meaning |
|--------|---------|
| ✅ Installed | Working on first attempt |
| 🔧 Fixed | Working after fix attempts |
| ❌ Removed: {error} | Removed after 3 failed attempts |
| ⏭️ Skipped: {reason} | Skipped (missing config) |

### Report Output

```markdown
# MCP Batch Installation Report

**Generated:** 2026-01-03T10:00:00Z
**Source:** docs/mcp-servers.md

## Summary
| Status | Count |
|--------|-------|
| ✅ Installed | 3 |
| 🔧 Fixed | 1 |
| ❌ Removed | 1 |

## Successfully Installed
- filesystem (11 tools)
- postgres (6 tools)

## Fixed Servers
- brave-search: Changed /tmp to /private/tmp

## Removed Servers
- my-api: ECONNREFUSED (server not running)
```

### Table Update

**Before:**
```markdown
| Name | Transport | Command |
|------|-----------|---------|
| filesystem | stdio | npx |
| my-api | http | |
```

**After:**
```markdown
| Name | Transport | Command | Status |
|------|-----------|---------|--------|
| filesystem | stdio | npx | ✅ Installed |
| my-api | http | | ❌ Removed: ECONNREFUSED |
```

---

## Skill Integration

### Using Skills Together

```
User wants to add multiple MCP servers
                ↓
        ┌───────────────────┐
        │  Batch Installer  │
        └───────────────────┘
                ↓
    ┌───────────────────────────┐
    │  For each server:         │
    │  1. Installer → Install   │
    │  2. Tester → Verify       │
    │  3. Debugger → Fix        │
    └───────────────────────────┘
                ↓
        ┌───────────────────┐
        │  Report + Update  │
        └───────────────────┘
```

### Standalone Usage

| Scenario | Use Skill |
|----------|-----------|
| Add one new server | MCP Server Installer |
| Test existing server | MCP Tool Tester |
| Fix broken server | MCP Server Debugger |
| Add many servers | MCP Batch Installer |

---

## Quick Reference

### MCP Tools Used

```python
# List servers
mcp__MCPProxy__upstream_servers(operation: "list")

# Add server
mcp__MCPProxy__upstream_servers(
    operation: "add",
    name: "name",
    command: "npx",
    args_json: '["-y", "package"]'
)

# Get logs
mcp__MCPProxy__upstream_servers(
    operation: "tail_log",
    name: "name",
    lines: 200
)

# Patch server
mcp__MCPProxy__upstream_servers(
    operation: "patch",
    name: "name",
    patch_json: '{"key": "value"}'
)

# Remove server
mcp__MCPProxy__upstream_servers(
    operation: "remove",
    name: "name"
)

# Discover tools
mcp__MCPProxy__retrieve_tools(
    query: "server:name",
    limit: 100
)

# Call tool
mcp__MCPProxy__call_tool(
    name: "server:tool",
    args_json: '{"param": "value"}'
)
```

### Common Fixes Reference

| Issue | Fix |
|-------|-----|
| `/tmp` on macOS | Use `/private/tmp` |
| `npx` not found | Install Node.js 18+ |
| `uvx` not found | `pip install uv` |
| Port in use | Kill process or change port |
| Invalid token | Regenerate and update |
| Package not found | `npm cache clean` + retry |

---

## File Locations

```
.claude/skills/
├── mcp-server-installer/
│   └── SKILL.md              # Installation skill
├── mcp-tool-tester/
│   └── SKILL.md              # Testing skill
├── mcp-server-debugger/
│   └── SKILL.md              # Debugging skill
└── mcp-batch-installer/
    └── SKILL.md              # Batch operations skill

docs/
├── MCP-SERVER-INSTALLATION-GUIDE.md  # Installation reference
└── MCP-SKILLS-DOCUMENTATION.md       # This file
```

---

## Version History

| Version | Date | Changes |
|---------|------|---------|
| 1.0 | 2026-01-03 | Initial release of all four skills |

---

**Documentation Version:** 1.0
**Last Updated:** 2026-01-03
**Author:** Claude Code with SuperClaude Framework

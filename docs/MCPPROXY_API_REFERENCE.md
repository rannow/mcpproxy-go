# MCPProxy API Reference

Comprehensive API documentation for mcpproxy-go HTTP REST API.

**Base URL**: `http://localhost:8080`
**Version**: v1.0
**Last Updated**: 2026-01-07

---

## Table of Contents

1. [Overview](#overview)
2. [Server Management API](#server-management-api)
3. [Agent API v1 (Recommended)](#agent-api-v1-recommended)
4. [Fast Action API](#fast-action-api)
5. [Group Management API](#group-management-api)
6. [Metrics & Monitoring API](#metrics--monitoring-api)
7. [Chat API (Diagnostic Agent)](#chat-api-diagnostic-agent)
8. [WebSocket API](#websocket-api)
9. [Tray Status API](#tray-status-api) ⭐ NEW
10. [System Integration API](#system-integration-api)

---

## Overview

### Authentication
Currently, no authentication is required for local API access.

### Response Format
All API responses are JSON unless otherwise noted:
```json
{
  "success": true,
  "message": "Operation completed",
  "data": { ... }
}
```

### Error Responses
```json
{
  "error": "Error description",
  "success": false
}
```

### HTTP Status Codes
| Code | Meaning |
|------|---------|
| 200 | Success |
| 400 | Bad Request - Invalid parameters |
| 404 | Not Found - Resource doesn't exist |
| 405 | Method Not Allowed |
| 500 | Internal Server Error |

---

## Server Management API

### List All Servers
```http
GET /api/servers
```

**Response** (200):
```json
{
  "servers": [
    {
      "name": "github",
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-github"],
      "env": {"GITHUB_TOKEN": "..."},
      "protocol": "stdio",
      "startup_mode": "active",
      "connection_state": "Ready",
      "tool_count": 17,
      "auto_disabled": false,
      "quarantined": false
    }
  ]
}
```

---

### Get Server Status (All Servers)
```http
GET /api/servers/status
```

**Response** (200):
```json
{
  "servers": [
    {
      "name": "github",
      "status": "Ready",
      "connected": true,
      "connecting": false,
      "retry_count": 0,
      "time_to_connection": "2.5m",
      "protocol": "stdio",
      "command": "npx",
      "tool_count": 17,
      "startup_mode": "active"
    }
  ]
}
```

**Status Values**:
- `Ready` - Connected and operational
- `Connecting` - Connection in progress
- `Disconnected` - Not connected
- `Auto-Disabled` - Disabled after repeated failures
- `Error` - Connection error

---

### Get Server Tools
```http
GET /api/servers/{server_name}/tools
```

**Example**:
```bash
curl http://localhost:8080/api/servers/github/tools
```

**Response** (200):
```json
{
  "tools": [
    {
      "name": "create_or_update_file",
      "description": "Create or update a single file in a GitHub repository"
    },
    {
      "name": "search_repositories",
      "description": "Search for GitHub repositories"
    }
  ]
}
```

**Error** (500):
```json
{
  "error": "Failed to get tools: server not found or not connected"
}
```

---

### Update Server Configuration (Standard API)
```http
PUT /api/servers/{server_name}/config
```

**Request Body**:
```json
{
  "enabled": true,
  "name": "github",
  "protocol": "stdio",
  "command": "npx",
  "args": ["-y", "@modelcontextprotocol/server-github"],
  "env": {"GITHUB_TOKEN": "..."},
  "working_dir": "",
  "url": "",
  "quarantined": false
}
```

**Response** (200):
```json
{
  "success": true,
  "message": "Configuration updated successfully",
  "server": {
    "name": "github",
    "startup_mode": "active",
    "updated": "2026-01-07T07:37:59.007534-06:00"
  }
}
```

---

## Agent API v1 (Recommended)

The Agent API v1 is the recommended interface for programmatic server management. It supports partial updates via PATCH.

### List All Servers
```http
GET /api/v1/agent/servers
```

**Response** (200):
```json
{
  "servers": [
    {
      "name": "github",
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-github"],
      "protocol": "stdio",
      "startup_mode": "active",
      "url": "",
      "working_dir": "",
      "status": {
        "state": "Ready"
      }
    }
  ]
}
```

---

### Get Server Details
```http
GET /api/v1/agent/servers/{server_name}
```

**Example**:
```bash
curl http://localhost:8080/api/v1/agent/servers/github
```

**Response** (200):
```json
{
  "name": "github",
  "command": "npx",
  "args": ["-y", "@modelcontextprotocol/server-github"],
  "env": {"GITHUB_PERSONAL_ACCESS_TOKEN": "ghp_..."},
  "protocol": "stdio",
  "startup_mode": "active",
  "url": "",
  "working_dir": "",
  "status": {
    "state": "Ready"
  },
  "tools": {
    "count": 17
  }
}
```

---

### Get Server Configuration
```http
GET /api/v1/agent/servers/{server_name}/config
```

**Response** (200):
```json
{
  "name": "github",
  "command": "npx",
  "args": ["-y", "@modelcontextprotocol/server-github"],
  "env": {"GITHUB_PERSONAL_ACCESS_TOKEN": "ghp_..."},
  "protocol": "stdio",
  "startup_mode": "active",
  "url": "",
  "working_dir": ""
}
```

---

### Update Server Configuration (Partial Update)
```http
PATCH /api/v1/agent/servers/{server_name}/config
```

**Request Body** (only include fields to update):
```json
{
  "startup_mode": "active"
}
```

**Updateable Fields**:
- `url` - Server URL (for HTTP/SSE servers)
- `command` - Command to run (for stdio servers)
- `args` - Command arguments array
- `env` - Environment variables object
- `protocol` - "stdio", "http", "sse"
- `startup_mode` - "active", "disabled", "quarantined", "lazy_loading"
- `working_dir` - Working directory path

**Response** (200):
```json
{
  "success": true,
  "message": "Configuration updated successfully",
  "needs_restart": true,
  "startup_mode": "active"
}
```

**Example - Re-enable Auto-Disabled Server**:
```bash
curl -X PATCH http://localhost:8080/api/v1/agent/servers/confluence/config \
  -H "Content-Type: application/json" \
  -d '{"startup_mode": "active"}'
```

---

### Get Server Logs
```http
GET /api/v1/agent/servers/{server_name}/logs
```

**Query Parameters**:
- `lines` - Number of lines to return (default: 100, max: 1000)

**Response** (200):
```json
{
  "server": "github",
  "logs": [
    {
      "timestamp": "2026-01-07T07:00:00-06:00",
      "level": "INFO",
      "message": "Server started successfully"
    }
  ],
  "count": 1,
  "limited": false
}
```

---

### Get Main MCPProxy Logs
```http
GET /api/v1/agent/logs/main
```

**Query Parameters**:
- `lines` - Number of lines to return (default: 100)

**Response** (200):
```json
{
  "logs": [...],
  "count": 100,
  "limited": true
}
```

---

### Search Registries
```http
GET /api/v1/agent/registries/search?query={search_term}
```

**Example**:
```bash
curl "http://localhost:8080/api/v1/agent/registries/search?query=github"
```

**Response** (200):
```json
{
  "query": "github",
  "results": [...],
  "message": "Registry search integration pending"
}
```

---

## Fast Action API

### Execute Fast Action
```http
POST /api/fast-action
```

**Available Actions**:

#### Check Disabled Servers
```json
{
  "action": "check_disabled_servers"
}
```

**Response**:
```json
{
  "success": true,
  "message": "📊 Analyzed 7 disabled servers",
  "details": {
    "count": 7,
    "report": [
      {
        "name": "confluence",
        "analysis": {
          "success": true,
          "message": "❌ Server is disabled\n✅ Command 'npx' found",
          "details": {
            "command": "npx",
            "command_found": true,
            "connection_state": "Disconnected",
            "enabled": false,
            "startup_mode": "auto_disabled"
          }
        }
      }
    ]
  }
}
```

#### Check Server Startup
```json
{
  "action": "check_startup",
  "server": "github"
}
```

**Note**: Use `"server"` field, not `"server_name"`.

#### Test Server Locally
```json
{
  "action": "test_local",
  "server": "github"
}
```

#### Check Documentation
```json
{
  "action": "check_docs",
  "server": "github"
}
```

#### Preload Packages
```json
{
  "action": "preload_packages",
  "server": "github"
}
```

---

## Group Management API

### List Groups
```http
GET /api/groups
```

**Response** (200):
```json
{
  "success": true,
  "groups": [
    {
      "id": 1,
      "name": "Development",
      "description": "Custom group: Development",
      "color": "#28a745"
    },
    {
      "id": 2,
      "name": "Production",
      "description": "Custom group: Production",
      "color": "#dc3545"
    }
  ],
  "assignments": []
}
```

---

### Create Group
```http
POST /api/groups
```

**Request Body**:
```json
{
  "name": "my-group",
  "description": "Group description"
}
```

**Response** (200):
```json
{
  "success": true,
  "message": "Group 'my-group' created successfully"
}
```

---

### Get Assignments
```http
GET /api/assignments
```

**Response** (200):
```json
{
  "success": true,
  "assignments": [
    {
      "server_name": "github",
      "group_name": "Development"
    }
  ]
}
```

---

### Assign Server to Group
```http
POST /api/assign-server
```

**Request Body**:
```json
{
  "server_name": "github",
  "group_id": 1
}
```

**Response** (200):
```json
{
  "success": true,
  "message": "Server 'github' assigned to group 'Development'"
}
```

---

### Unassign Server from Group
```http
POST /api/unassign-server
```

**Request Body**:
```json
{
  "server_name": "github"
}
```

**Response** (200):
```json
{
  "success": true,
  "message": "Server 'github' unassigned from group"
}
```

---

### Toggle Group Servers
```http
POST /api/toggle-group-servers
```

**Request Body**:
```json
{
  "group_id": 1,
  "enabled": true
}
```

---

## Metrics & Monitoring API

### Get Current Metrics
```http
GET /api/metrics/current
```

**Response** (200):
```json
{
  "timestamp": "2026-01-07T07:37:58.983396-06:00",
  "uptime": "1h0m0s",
  "go_version": "go1.25.1",
  "num_goroutines": 1414,
  "num_cpu": 16,
  "memory_stats": {
    "Alloc": 4523248,
    "TotalAlloc": 937935176,
    "Sys": 144072736,
    "HeapAlloc": 4523248,
    "HeapSys": 128942080,
    "HeapInuse": 8355840,
    "HeapObjects": 26966,
    "StackInuse": 5275648
  }
}
```

---

### Get Memory/Diagnostic Content
```http
GET /api/memory
```

**Response** (200):
```json
{
  "content": "# Diagnostic Agent Memory\n\nThis file stores common problems..."
}
```

---

## Chat API (Diagnostic Agent)

These endpoints are designed for OpenAI Function Calling integration.

### Get Server Status
```http
POST /chat/get-server-status
```

**Request Body**:
```json
{
  "server": "github"
}
```

---

### List All Servers
```http
POST /chat/list-all-servers
```

**Request Body**:
```json
{}
```

**Response** (200):
```json
{
  "content": "[{\"name\": \"github\", \"status\": \"Ready\", ...}]"
}
```

---

### List All Tools
```http
POST /chat/list-all-tools
```

**Note**: May timeout for large tool lists. Use `/api/servers/{name}/tools` instead.

---

### Read Config
```http
POST /chat/read-config
```

---

### Write Config
```http
POST /chat/write-config
```

---

### Read Log
```http
POST /chat/read-log
```

---

### Restart Server
```http
POST /chat/restart-server
```

**Request Body**:
```json
{
  "server": "github"
}
```

---

### Call Tool
```http
POST /chat/call-tool
```

**Request Body**:
```json
{
  "server": "github",
  "tool": "search_repositories",
  "args": {"query": "test"}
}
```

---

## WebSocket API

### Events Stream
```
ws://localhost:8080/ws/events
```

Provides real-time events for all servers.

---

### Server-Specific Stream
```
ws://localhost:8080/ws/servers?server={server_name}
```

Provides real-time events for a specific server.

**Message Format**:
```json
{
  "type": "status_change",
  "server": "github",
  "data": {
    "previous": "Connecting",
    "current": "Ready"
  },
  "timestamp": "2026-01-07T07:00:00Z"
}
```

---

## Tray Status API

### Get Tray Menu Categories
```http
GET /api/tray/status
```

Returns server status categories exactly as displayed in the system tray menu. Use this endpoint to programmatically verify tray functionality.

**Response** (200):
```json
{
  "connected": {
    "count": 36,
    "servers": ["github", "gitlab", "docker", "..."]
  },
  "disconnected": {
    "count": 0,
    "servers": []
  },
  "sleeping": {
    "count": 0,
    "servers": []
  },
  "disabled": {
    "count": 0,
    "servers": []
  },
  "auto_disabled": {
    "count": 7,
    "servers": ["confluence", "exa", "gdrive", "..."]
  },
  "quarantined": {
    "count": 0,
    "servers": []
  },
  "total": 43,
  "timestamp": "2026-01-07T08:33:22.065937-06:00"
}
```

**Categories**:
| Category | Icon | Description |
|----------|------|-------------|
| `connected` | 🟢 | Servers with `connection_state == "Ready"` (not disabled/quarantined/auto_disabled) |
| `disconnected` | 🔴 | Servers with `connection_state != "Ready"` (not disabled/quarantined/auto_disabled) |
| `sleeping` | 💤 | Servers with `startup_mode == "lazy_loading"` AND `connection_state != "Ready"` |
| `disabled` | ⏸️ | Servers with `startup_mode == "disabled"` |
| `auto_disabled` | 🚫 | Servers with `startup_mode == "auto_disabled"` |
| `quarantined` | 🔒 | Servers with `startup_mode == "quarantined"` |

**Example - Test Tray Functionality**:
```bash
# Get tray status
curl http://localhost:8080/api/tray/status | jq

# Verify connected count matches tray menu
curl -s http://localhost:8080/api/tray/status | jq '.connected.count'

# List all auto-disabled servers
curl -s http://localhost:8080/api/tray/status | jq '.auto_disabled.servers[]'
```

---

## System Integration API

### Open Path in System
```http
POST /api/open-path
```

Opens a path in the system file manager (Finder/Explorer).

**Request Body**:
```json
{
  "path": "/tmp"
}
```

**Response** (200):
```json
{
  "status": "success",
  "path": "/tmp"
}
```

---

### Launch MCP Inspector
```http
POST /api/launch-inspector
```

---

## Startup Mode Reference

| Mode | Behavior |
|------|----------|
| `active` | Server connects on mcpproxy startup |
| `disabled` | Server is disabled, will not connect |
| `quarantined` | Server blocked for security review |
| `auto_disabled` | Disabled after repeated connection failures |
| `lazy_loading` | Connects on first tool call |

### Re-enabling Auto-Disabled Servers

```bash
# Via Agent API (recommended)
curl -X PATCH http://localhost:8080/api/v1/agent/servers/{server_name}/config \
  -H "Content-Type: application/json" \
  -d '{"startup_mode": "active"}'
```

---

## Common Use Cases

### 1. List all available tools
```bash
# Get server list
curl http://localhost:8080/api/servers

# Get tools for each server
curl http://localhost:8080/api/servers/github/tools
```

### 2. Check why server is failing
```bash
# Get server status
curl http://localhost:8080/api/v1/agent/servers/github

# Get server logs
curl http://localhost:8080/api/v1/agent/servers/github/logs

# Run diagnostics
curl -X POST http://localhost:8080/api/fast-action \
  -H "Content-Type: application/json" \
  -d '{"action": "check_startup", "server": "github"}'
```

### 3. Re-enable auto-disabled server
```bash
# Check current config
curl http://localhost:8080/api/v1/agent/servers/confluence/config

# Re-enable
curl -X PATCH http://localhost:8080/api/v1/agent/servers/confluence/config \
  -H "Content-Type: application/json" \
  -d '{"startup_mode": "active"}'
```

### 4. Update environment variables
```bash
curl -X PATCH http://localhost:8080/api/v1/agent/servers/exa/config \
  -H "Content-Type: application/json" \
  -d '{"env": {"EXA_API_KEY": "your-new-key"}}'
```

---

## Test Results Summary

| Category | Success Rate |
|----------|-------------|
| Agent API v1 | 100% |
| Server Tools API | 50% (depends on server state) |
| Fast Action API | 100% |
| Groups API | 83% |
| Basic Endpoints | 100% |
| Chat API | 67% |
| WebSocket | Expected behavior |
| **Tray Status API** | **100%** ⭐ NEW |

---

## Notes

1. **Timeouts**: Some endpoints may timeout for operations involving many servers. The default timeout is 30 seconds.

2. **WebSocket**: WebSocket endpoints require proper upgrade headers and cannot be tested with standard HTTP requests.

3. **Fast Action**: Use `"server"` field, not `"server_name"` for server-specific actions.

4. **Auto-Disabled Servers**: Servers are auto-disabled after 7 consecutive startup failures.

5. **Tray Status API**: The `/api/tray/status` endpoint returns categories matching the system tray menu exactly. Use this for scripted tray functionality testing.

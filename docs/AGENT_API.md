# Agent API Documentation

RESTful API endpoints for AI agent integration with mcpproxy. These endpoints enable programmatic management of MCP servers, diagnostics, configuration, and log analysis.

## Base URL

```
http://localhost:8080/api/v1/agent
```

## Endpoints

### 1. List All Servers

Get a list of all configured MCP servers with their status.

**Endpoint:** `GET /api/v1/agent/servers`

**Response:**
```json
{
  "servers": [
    {
      "name": "github-server",
      "url": "https://api.github.com/mcp",
      "command": "",
      "args": [],
      "protocol": "http",
      "enabled": true,
      "quarantined": false,
      "working_dir": "",
      "status": {
        "state": "Ready",
        "connected": true
      }
    }
  ],
  "total": 1
}
```

**Example:**
```bash
curl http://localhost:8080/api/v1/agent/servers
```

---

### 2. Get Server Details

Get detailed information about a specific MCP server.

**Endpoint:** `GET /api/v1/agent/servers/{name}`

**Path Parameters:**
- `name` (required): Server name

**Response:**
```json
{
  "name": "github-server",
  "url": "https://api.github.com/mcp",
  "command": "",
  "args": [],
  "env": {},
  "protocol": "http",
  "enabled": true,
  "quarantined": false,
  "working_dir": "",
  "status": {
    "state": "Ready",
    "connected": true
  },
  "tools": {
    "count": 12
  }
}
```

**Example:**
```bash
curl http://localhost:8080/api/v1/agent/servers/github-server
```

---

### 3. Get Server Logs

Retrieve logs for a specific MCP server.

**Endpoint:** `GET /api/v1/agent/servers/{name}/logs`

**Path Parameters:**
- `name` (required): Server name

**Query Parameters:**
- `lines` (optional, default: 100, max: 1000): Number of log lines to retrieve
- `filter` (optional): Filter logs by pattern (case-insensitive)

**Response:**
```json
{
  "server": "github-server",
  "logs": [
    {
      "timestamp": "2025-10-31T12:00:00Z",
      "level": "INFO",
      "message": "Server connected successfully"
    },
    {
      "timestamp": "2025-10-31T12:05:00Z",
      "level": "ERROR",
      "message": "OAuth token expired"
    }
  ],
  "count": 2,
  "limited": false
}
```

**Examples:**
```bash
# Get last 100 logs
curl http://localhost:8080/api/v1/agent/servers/github-server/logs

# Get last 50 logs
curl "http://localhost:8080/api/v1/agent/servers/github-server/logs?lines=50"

# Filter for errors only
curl "http://localhost:8080/api/v1/agent/servers/github-server/logs?filter=error"
```

---

### 4. Get Main Logs

Retrieve main mcpproxy logs.

**Endpoint:** `GET /api/v1/agent/logs/main`

**Query Parameters:**
- `lines` (optional, default: 100, max: 1000): Number of log lines to retrieve
- `filter` (optional): Filter logs by pattern (case-insensitive)

**Response:**
```json
{
  "logs": [
    {
      "timestamp": "2025-10-31T12:00:00Z",
      "level": "INFO",
      "message": "MCPProxy started"
    }
  ],
  "count": 1,
  "limited": false
}
```

**Examples:**
```bash
# Get last 100 logs
curl http://localhost:8080/api/v1/agent/logs/main

# Get last 200 logs with error filter
curl "http://localhost:8080/api/v1/agent/logs/main?lines=200&filter=error"
```

---

### 5. Get Server Configuration

Retrieve configuration for a specific server.

**Endpoint:** `GET /api/v1/agent/servers/{name}/config`

**Path Parameters:**
- `name` (required): Server name

**Response:**
```json
{
  "name": "github-server",
  "url": "https://api.github.com/mcp",
  "command": "",
  "args": [],
  "env": {
    "GITHUB_TOKEN": "***"
  },
  "protocol": "http",
  "enabled": true,
  "quarantined": false,
  "working_dir": ""
}
```

**Example:**
```bash
curl http://localhost:8080/api/v1/agent/servers/github-server/config
```

---

### 6. Update Server Configuration

Update configuration for a specific server.

**Endpoint:** `PATCH /api/v1/agent/servers/{name}/config`

**Path Parameters:**
- `name` (required): Server name

**Request Body:**
```json
{
  "enabled": true,
  "url": "https://new-url.com/mcp",
  "env": {
    "NEW_VAR": "value"
  },
  "working_dir": "/path/to/directory"
}
```

**Response:**
```json
{
  "success": true,
  "message": "Configuration updated successfully",
  "needs_restart": true,
  "server_enabled": true
}
```

**Examples:**
```bash
# Enable a server
curl -X PATCH http://localhost:8080/api/v1/agent/servers/github-server/config \
  -H "Content-Type: application/json" \
  -d '{"enabled": true}'

# Update URL and environment variables
curl -X PATCH http://localhost:8080/api/v1/agent/servers/my-server/config \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://new-api.example.com/mcp",
    "env": {
      "API_KEY": "new-key",
      "API_SECRET": "new-secret"
    }
  }'
```

---

### 7. Search MCP Server Registries

Search for MCP servers in configured registries.

**Endpoint:** `GET /api/v1/agent/registries/search`

**Query Parameters:**
- `query` (required): Search query
- `registry` (optional): Specific registry to search

**Response:**
```json
{
  "results": [],
  "query": "weather",
  "message": "Registry search integration pending"
}
```

**Example:**
```bash
curl "http://localhost:8080/api/v1/agent/registries/search?query=weather"
```

---

### 8. Install MCP Server

Install a new MCP server from a registry.

**Endpoint:** `POST /api/v1/agent/install`

**Request Body:**
```json
{
  "server_id": "weather-api",
  "name": "my-weather-server",
  "config": {
    "url": "https://weather-api.com/mcp",
    "enabled": true
  }
}
```

**Response:**
```json
{
  "success": false,
  "message": "Server installation integration pending"
}
```

**Example:**
```bash
curl -X POST http://localhost:8080/api/v1/agent/install \
  -H "Content-Type: application/json" \
  -d '{
    "server_id": "weather-api",
    "name": "weather-server",
    "config": {
      "enabled": true
    }
  }'
```

---

## Error Responses

All endpoints return standard HTTP error codes:

- `400 Bad Request`: Invalid request parameters
- `404 Not Found`: Server not found
- `405 Method Not Allowed`: Invalid HTTP method
- `500 Internal Server Error`: Server error

**Error Response Format:**
```json
{
  "error": "Server 'invalid-server' not found"
}
```

Or plain text:
```
Server 'invalid-server' not found
```

---

## Python Client Example

Using the Python agent's `MCPProxyClient`:

```python
from mcp_agent.tools.diagnostic import MCPProxyClient

# Initialize client
client = MCPProxyClient(base_url="http://localhost:8080")

# List servers
servers = await client.get("/api/v1/agent/servers")
print(f"Found {servers['total']} servers")

# Get server details
server = await client.get("/api/v1/agent/servers/github-server")
print(f"Server status: {server['status']['state']}")

# Get logs
logs = await client.get_server_logs("github-server", lines=50, filter_pattern="error")
print(f"Found {len(logs)} error log entries")

# Update config
result = await client.patch(
    "/api/v1/agent/servers/github-server/config",
    json={"enabled": True}
)
print(f"Config updated: {result['message']}")
```

---

## Log Entry Format

Logs are returned as an array of entries. Each entry can be:

**Structured (JSON):**
```json
{
  "timestamp": "2025-10-31T12:00:00Z",
  "level": "ERROR",
  "message": "Connection failed",
  "context": {
    "server": "github-server",
    "error": "timeout"
  }
}
```

**Plain Text:**
```json
{
  "timestamp": "2025-10-31T12:00:00Z",
  "level": "INFO",
  "message": "Server started",
  "raw": "2025-10-31T12:00:00Z\tINFO\tServer started"
}
```

---

## Integration with Python Agent

The Python MCP agent uses these endpoints to:

1. **Discover Servers**: List all servers and their status
2. **Diagnose Issues**: Analyze logs for errors and patterns
3. **Monitor Health**: Check server connectivity and tool availability
4. **Modify Configuration**: Update server settings dynamically
5. **Install Servers**: Add new servers from registries

See [agent/README.md](../agent/README.md) for the Python agent documentation.

---

## Testing

Use the provided test script:

```bash
chmod +x agent/test_agent_api.sh
./agent/test_agent_api.sh
```

Or test individual endpoints with curl:

```bash
# List all servers
curl http://localhost:8080/api/v1/agent/servers | jq

# Get specific server with pretty printing
curl http://localhost:8080/api/v1/agent/servers/my-server | jq

# Get last 20 error logs
curl "http://localhost:8080/api/v1/agent/logs/main?lines=20&filter=error" | jq '.logs'
```

---

## Security Considerations

1. **No Authentication**: Currently, the agent API has no authentication. It's designed for local use only.
2. **Local Access Only**: Only bind to localhost in production
3. **Future Enhancement**: Authentication tokens and rate limiting will be added
4. **Config Changes**: Configuration updates persist to disk and may require server restart

---

## Future Enhancements

- [ ] Authentication/authorization
- [ ] Rate limiting
- [ ] WebSocket support for real-time log streaming
- [ ] Complete registry search integration
- [ ] Server installation automation
- [ ] Bulk operations
- [ ] Pagination for large result sets
- [ ] Server metrics and performance data

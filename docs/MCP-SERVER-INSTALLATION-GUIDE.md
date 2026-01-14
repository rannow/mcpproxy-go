# MCP Server Installation Guide

Comprehensive guide for installing and configuring MCP (Model Context Protocol) servers with all available transport methods, authentication options, and prerequisites.

---

## Table of Contents

1. [Transport Types Overview](#transport-types-overview)
2. [STDIO Transport](#stdio-transport)
   - [NPX (Node.js)](#npx-nodejs)
   - [UVX (Python)](#uvx-python)
   - [Direct Binary](#direct-binary)
   - [Python Direct](#python-direct)
   - [Node.js Direct](#nodejs-direct)
3. [HTTP/SSE Transport](#httpsse-transport)
   - [HTTP](#http)
   - [HTTPS with TLS](#https-with-tls)
   - [Server-Sent Events (SSE)](#server-sent-events-sse)
   - [Streamable HTTP](#streamable-http)
4. [Docker Transport](#docker-transport)
   - [Basic Docker](#basic-docker)
   - [Docker with Isolation](#docker-with-isolation)
   - [Docker Compose](#docker-compose)
5. [Authentication Methods](#authentication-methods)
   - [API Key Headers](#api-key-headers)
   - [Bearer Token](#bearer-token)
   - [OAuth 2.0](#oauth-20)
   - [Basic Auth](#basic-auth)
6. [Prerequisites by Server Type](#prerequisites-by-server-type)
7. [Configuration Examples](#configuration-examples)
8. [Troubleshooting](#troubleshooting)

---

## Transport Types Overview

| Transport | Protocol | Use Case | Prerequisites |
|-----------|----------|----------|---------------|
| **stdio** | JSON-RPC over stdin/stdout | Local processes | Runtime (Node/Python) |
| **http** | HTTP POST | Local/Remote servers | Running HTTP server |
| **https** | HTTP POST + TLS | Secure remote servers | TLS certificates |
| **sse** | Server-Sent Events | Streaming responses | SSE-capable server |
| **streamable-http** | Bidirectional HTTP | Complex streaming | Compatible server |

---

## STDIO Transport

STDIO is the most common transport for local MCP servers. The server runs as a subprocess communicating via stdin/stdout.

### NPX (Node.js)

**Prerequisites:**
- Node.js 18+ installed
- npm/npx available in PATH

**Installation:**
```bash
# Verify Node.js installation
node --version  # Should be 18+
npx --version
```

**Configuration:**
```json
{
  "name": "filesystem-server",
  "command": "npx",
  "args": [
    "-y",
    "@modelcontextprotocol/server-filesystem",
    "/path/to/allowed/directory"
  ],
  "auto_start": true
}
```

**Common NPX MCP Servers:**

| Server | Package | Purpose |
|--------|---------|---------|
| Filesystem | `@modelcontextprotocol/server-filesystem` | File operations |
| GitHub | `@modelcontextprotocol/server-github` | GitHub API |
| GitLab | `@modelcontextprotocol/server-gitlab` | GitLab API |
| Slack | `@modelcontextprotocol/server-slack` | Slack integration |
| Google Drive | `@modelcontextprotocol/server-gdrive` | Google Drive access |
| Postgres | `@modelcontextprotocol/server-postgres` | PostgreSQL queries |
| SQLite | `@modelcontextprotocol/server-sqlite` | SQLite database |
| Puppeteer | `@modelcontextprotocol/server-puppeteer` | Browser automation |
| Brave Search | `@modelcontextprotocol/server-brave-search` | Web search |
| Fetch | `@modelcontextprotocol/server-fetch` | HTTP requests |
| Memory | `@modelcontextprotocol/server-memory` | Knowledge graph |
| Sequential Thinking | `@modelcontextprotocol/server-sequential-thinking` | Reasoning |
| Everything | `@modelcontextprotocol/server-everything` | Demo/testing |

**With Environment Variables:**
```json
{
  "name": "github-server",
  "command": "npx",
  "args": ["-y", "@modelcontextprotocol/server-github"],
  "env": {
    "GITHUB_PERSONAL_ACCESS_TOKEN": "ghp_xxxxxxxxxxxx"
  }
}
```

---

### UVX (Python)

**Prerequisites:**
- Python 3.10+ installed
- uv package manager installed

**Installation:**
```bash
# Install uv (recommended Python package manager)
curl -LsSf https://astral.sh/uv/install.sh | sh

# Or via pip
pip install uv

# Or via Homebrew (macOS)
brew install uv

# Verify installation
uv --version
uvx --version
```

**Configuration:**
```json
{
  "name": "mcp-server-fetch",
  "command": "uvx",
  "args": ["mcp-server-fetch"],
  "auto_start": true
}
```

**Common UVX MCP Servers:**

| Server | Package | Purpose |
|--------|---------|---------|
| Fetch | `mcp-server-fetch` | HTTP requests |
| Time | `mcp-server-time` | Time/timezone |
| Git | `mcp-server-git` | Git operations |
| Sqlite | `mcp-server-sqlite` | SQLite database |
| Filesystem | `mcp-server-filesystem` | File operations |
| Sentry | `mcp-server-sentry` | Error tracking |
| Raygun | `mcp-server-raygun` | Error monitoring |

**With Dependencies:**
```json
{
  "name": "custom-python-server",
  "command": "uvx",
  "args": [
    "--from", "git+https://github.com/user/mcp-server.git",
    "mcp-server"
  ]
}
```

---

### Direct Binary

**Prerequisites:**
- Compiled binary for your OS/architecture
- Binary in PATH or absolute path specified

**Configuration:**
```json
{
  "name": "custom-binary-server",
  "command": "/usr/local/bin/my-mcp-server",
  "args": ["--config", "/etc/mcp/config.yaml"],
  "auto_start": true
}
```

**Go Binary Example:**
```json
{
  "name": "go-mcp-server",
  "command": "./bin/mcp-server",
  "args": ["serve", "--port", "0"],
  "working_dir": "/opt/mcp-servers/go-server"
}
```

**Rust Binary Example:**
```json
{
  "name": "rust-mcp-server",
  "command": "/usr/local/bin/mcp-rust-server",
  "args": ["--stdio"]
}
```

---

### Python Direct

**Prerequisites:**
- Python 3.10+ installed
- Required packages installed (pip/poetry/uv)

**Installation:**
```bash
# Using pip
pip install mcp-server-package

# Using poetry
poetry add mcp-server-package

# Using uv
uv pip install mcp-server-package
```

**Configuration:**
```json
{
  "name": "python-mcp-server",
  "command": "python",
  "args": ["-m", "mcp_server_package"],
  "env": {
    "PYTHONPATH": "/path/to/custom/modules"
  }
}
```

**With Virtual Environment:**
```json
{
  "name": "venv-mcp-server",
  "command": "/path/to/venv/bin/python",
  "args": ["-m", "my_mcp_server"],
  "working_dir": "/path/to/project"
}
```

**Poetry Project:**
```json
{
  "name": "poetry-mcp-server",
  "command": "poetry",
  "args": ["run", "python", "-m", "my_mcp_server"],
  "working_dir": "/path/to/poetry/project"
}
```

---

### Node.js Direct

**Prerequisites:**
- Node.js 18+ installed
- Package installed globally or locally

**Global Installation:**
```bash
npm install -g @modelcontextprotocol/server-filesystem
```

**Configuration (Global):**
```json
{
  "name": "global-node-server",
  "command": "mcp-server-filesystem",
  "args": ["/allowed/path"]
}
```

**Local Installation:**
```bash
cd /path/to/project
npm install @modelcontextprotocol/server-filesystem
```

**Configuration (Local):**
```json
{
  "name": "local-node-server",
  "command": "node",
  "args": ["node_modules/.bin/mcp-server-filesystem", "/allowed/path"],
  "working_dir": "/path/to/project"
}
```

---

## HTTP/SSE Transport

HTTP-based transports connect to running servers over the network.

### HTTP

**Prerequisites:**
- MCP server running and accessible
- Network connectivity to server

**Configuration:**
```json
{
  "name": "http-mcp-server",
  "url": "http://localhost:8055/mcp",
  "transport": "http",
  "auto_start": true
}
```

**With Custom Port:**
```json
{
  "name": "remote-mcp-server",
  "url": "http://192.168.1.100:3000/api/mcp",
  "transport": "http"
}
```

---

### HTTPS with TLS

**Prerequisites:**
- Valid TLS certificate (or self-signed for development)
- Server configured for HTTPS

**Configuration:**
```json
{
  "name": "secure-mcp-server",
  "url": "https://mcp.example.com/api",
  "transport": "https"
}
```

**Self-Signed Certificate (Development):**
```json
{
  "name": "dev-secure-server",
  "url": "https://localhost:8443/mcp",
  "transport": "https",
  "headers": {
    "X-Skip-Verify": "true"
  }
}
```

---

### Server-Sent Events (SSE)

**Prerequisites:**
- SSE-capable MCP server
- Stable network connection (for streaming)

**Configuration:**
```json
{
  "name": "sse-mcp-server",
  "url": "http://localhost:8080/sse",
  "transport": "sse"
}
```

**With Reconnection:**
```json
{
  "name": "sse-with-retry",
  "url": "http://localhost:8080/events",
  "transport": "sse",
  "headers": {
    "X-Reconnect-Interval": "5000"
  }
}
```

---

### Streamable HTTP

**Prerequisites:**
- Server supporting bidirectional streaming
- HTTP/2 or chunked transfer encoding

**Configuration:**
```json
{
  "name": "streamable-server",
  "url": "http://localhost:9000/stream",
  "transport": "streamable-http"
}
```

---

## Docker Transport

Run MCP servers in isolated Docker containers.

### Basic Docker

**Prerequisites:**
- Docker installed and running
- Docker daemon accessible

**Installation:**
```bash
# Verify Docker installation
docker --version
docker ps
```

**Configuration:**
```json
{
  "name": "docker-mcp-server",
  "command": "docker",
  "args": [
    "run", "--rm", "-i",
    "--name", "mcp-filesystem",
    "mcp/filesystem-server",
    "/data"
  ]
}
```

**With Volume Mounts:**
```json
{
  "name": "docker-with-volumes",
  "command": "docker",
  "args": [
    "run", "--rm", "-i",
    "-v", "/host/path:/container/path:ro",
    "-v", "/tmp:/tmp",
    "mcp/server-image"
  ]
}
```

**With Environment Variables:**
```json
{
  "name": "docker-with-env",
  "command": "docker",
  "args": [
    "run", "--rm", "-i",
    "-e", "API_KEY=secret",
    "-e", "DEBUG=true",
    "mcp/api-server"
  ]
}
```

---

### Docker with Isolation

MCPProxy supports automatic Docker isolation for stdio servers.

**Configuration:**
```json
{
  "name": "isolated-server",
  "command": "npx",
  "args": ["-y", "@modelcontextprotocol/server-filesystem", "/data"],
  "isolation": {
    "enabled": true,
    "image": "node:20-slim",
    "memory_limit": "512m",
    "cpu_limit": "1.0",
    "network_mode": "none",
    "working_dir": "/app"
  }
}
```

**Global Isolation Settings:**
```json
{
  "docker_isolation": {
    "enabled": true,
    "memory_limit": "512m",
    "cpu_limit": "1.0",
    "network_mode": "bridge",
    "default_images": {
      "npx": "node:20",
      "node": "node:20",
      "python": "python:3.11",
      "uvx": "python:3.11",
      "pip": "python:3.11",
      "go": "golang:1.21-alpine",
      "cargo": "rust:1.75-slim"
    }
  }
}
```

---

### Docker Compose

**Prerequisites:**
- Docker Compose installed
- docker-compose.yml file

**docker-compose.yml:**
```yaml
version: '3.8'
services:
  mcp-filesystem:
    image: mcp/filesystem-server
    volumes:
      - /data:/data:ro
    stdin_open: true
    tty: true

  mcp-postgres:
    image: mcp/postgres-server
    environment:
      - DATABASE_URL=postgresql://user:pass@db:5432/mydb
    depends_on:
      - db
    stdin_open: true

  db:
    image: postgres:15
    environment:
      POSTGRES_USER: user
      POSTGRES_PASSWORD: pass
      POSTGRES_DB: mydb
```

**Configuration:**
```json
{
  "name": "compose-mcp-server",
  "command": "docker",
  "args": [
    "compose", "-f", "/path/to/docker-compose.yml",
    "run", "--rm", "-T",
    "mcp-filesystem"
  ]
}
```

---

## Authentication Methods

### API Key Headers

**Configuration:**
```json
{
  "name": "api-key-server",
  "url": "https://api.example.com/mcp",
  "transport": "https",
  "headers": {
    "X-API-Key": "your-api-key-here"
  }
}
```

**Multiple Headers:**
```json
{
  "name": "multi-header-server",
  "url": "https://api.example.com/mcp",
  "headers": {
    "X-API-Key": "key123",
    "X-Client-ID": "client456",
    "X-Request-Source": "mcpproxy"
  }
}
```

---

### Bearer Token

**Configuration:**
```json
{
  "name": "bearer-auth-server",
  "url": "https://api.example.com/mcp",
  "transport": "https",
  "headers": {
    "Authorization": "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
  }
}
```

---

### OAuth 2.0

**Prerequisites:**
- OAuth provider configured
- Client credentials or authorization code

**Configuration:**
```json
{
  "name": "oauth-server",
  "url": "https://api.example.com/mcp",
  "transport": "https",
  "oauth": {
    "client_id": "your-client-id",
    "client_secret": "your-client-secret",
    "token_url": "https://auth.example.com/oauth/token",
    "scopes": ["read", "write"]
  }
}
```

**With Refresh Token:**
```json
{
  "name": "oauth-refresh-server",
  "url": "https://api.example.com/mcp",
  "oauth": {
    "client_id": "client-id",
    "client_secret": "client-secret",
    "token_url": "https://auth.example.com/token",
    "refresh_token": "refresh-token-here",
    "scopes": ["mcp:read", "mcp:write"]
  }
}
```

---

### Basic Auth

**Configuration:**
```json
{
  "name": "basic-auth-server",
  "url": "https://api.example.com/mcp",
  "transport": "https",
  "headers": {
    "Authorization": "Basic dXNlcm5hbWU6cGFzc3dvcmQ="
  }
}
```

**Note:** Base64 encode `username:password` for the Authorization header.

---

## Prerequisites by Server Type

### Official Anthropic Servers

| Server | Prerequisites | Installation |
|--------|--------------|--------------|
| Filesystem | Node.js 18+ | `npx -y @modelcontextprotocol/server-filesystem` |
| GitHub | Node.js 18+, GitHub Token | `npx -y @modelcontextprotocol/server-github` |
| Postgres | Node.js 18+, PostgreSQL | `npx -y @modelcontextprotocol/server-postgres` |
| SQLite | Node.js 18+ | `npx -y @modelcontextprotocol/server-sqlite` |
| Puppeteer | Node.js 18+, Chrome/Chromium | `npx -y @modelcontextprotocol/server-puppeteer` |
| Brave Search | Node.js 18+, Brave API Key | `npx -y @modelcontextprotocol/server-brave-search` |

### Python-Based Servers

| Server | Prerequisites | Installation |
|--------|--------------|--------------|
| mcp-server-fetch | Python 3.10+, uv | `uvx mcp-server-fetch` |
| mcp-server-git | Python 3.10+, uv, git | `uvx mcp-server-git` |
| mcp-server-time | Python 3.10+, uv | `uvx mcp-server-time` |

### Database Servers

| Server | Prerequisites | Installation |
|--------|--------------|--------------|
| PostgreSQL | PostgreSQL running, connection string | `npx @modelcontextprotocol/server-postgres` |
| SQLite | SQLite library | `npx @modelcontextprotocol/server-sqlite` |
| MySQL | MySQL/MariaDB running | Community server |
| MongoDB | MongoDB running | Community server |

### Cloud Service Servers

| Server | Prerequisites | Installation |
|--------|--------------|--------------|
| AWS | AWS CLI, credentials | Community server |
| GCP | gcloud CLI, credentials | Community server |
| Azure | Azure CLI, credentials | Community server |
| Cloudflare | Cloudflare API token | Community server |

---

## Configuration Examples

### Complete mcpproxy Configuration

```json
{
  "listen": ":8080",
  "data_dir": "/Users/username/.mcpproxy",

  "docker_isolation": {
    "enabled": false,
    "memory_limit": "512m",
    "cpu_limit": "1.0",
    "network_mode": "bridge",
    "default_images": {
      "npx": "node:20",
      "uvx": "python:3.11"
    }
  },

  "mcpServers": [
    {
      "name": "filesystem",
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-filesystem", "/private/tmp"],
      "auto_start": true
    },
    {
      "name": "github",
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-github"],
      "env": {
        "GITHUB_PERSONAL_ACCESS_TOKEN": "ghp_xxxxx"
      },
      "auto_start": true
    },
    {
      "name": "fetch-python",
      "command": "uvx",
      "args": ["mcp-server-fetch"],
      "auto_start": true
    },
    {
      "name": "remote-api",
      "url": "https://api.example.com/mcp",
      "transport": "https",
      "headers": {
        "Authorization": "Bearer token123"
      },
      "auto_start": true
    },
    {
      "name": "local-http",
      "url": "http://localhost:8055/mcp",
      "transport": "http",
      "auto_start": true
    }
  ]
}
```

### Claude Desktop Configuration

```json
{
  "mcpServers": {
    "filesystem": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-filesystem", "/Users/username/Documents"]
    },
    "github": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-github"],
      "env": {
        "GITHUB_PERSONAL_ACCESS_TOKEN": "ghp_xxxxx"
      }
    },
    "memory": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-memory"]
    }
  }
}
```

---

## Troubleshooting

### Common Issues

| Problem | Cause | Solution |
|---------|-------|----------|
| "command not found" | Runtime not installed | Install Node.js/Python |
| "npx: command not found" | Node.js not in PATH | Add to PATH or use full path |
| "uvx: command not found" | uv not installed | Install with `pip install uv` |
| "ECONNREFUSED" | HTTP server not running | Start the server first |
| "certificate verify failed" | Invalid TLS cert | Use valid cert or skip verification |
| "permission denied" | File permissions | Check file/directory permissions |
| "Docker not running" | Docker daemon stopped | Start Docker daemon |
| "Container not found" | Image not pulled | `docker pull image:tag` |

### Checking Server Status

```bash
# Check if npx servers work
npx -y @modelcontextprotocol/server-everything

# Check if uvx servers work
uvx mcp-server-time

# Check HTTP server
curl http://localhost:8055/health

# Check Docker
docker ps
docker logs container-name
```

### Debug Mode

Enable verbose logging:
```json
{
  "logging": {
    "level": "debug",
    "enable_console": true
  }
}
```

---

## Quick Reference

### Transport Selection Guide

| Scenario | Recommended Transport |
|----------|----------------------|
| Local development | stdio (npx/uvx) |
| Production local | stdio with Docker isolation |
| Remote server | https with auth |
| Streaming responses | sse |
| High-security | https + Docker + OAuth |

### Command Cheat Sheet

```bash
# Install Node.js servers
npx -y @modelcontextprotocol/server-<name>

# Install Python servers
uvx mcp-server-<name>

# Run with Docker
docker run --rm -i mcp/<server>

# Check server health
curl http://localhost:8080/api/servers
```

---

**Version:** 1.0
**Last Updated:** 2026-01-03
**Author:** MCP Tool Tester

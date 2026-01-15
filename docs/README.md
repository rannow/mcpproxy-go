# MCPProxy Documentation

Welcome to the MCPProxy documentation. This guide will help you understand, configure, and use MCPProxy effectively.

## 📚 Documentation Structure

### Getting Started
- **[Setup Guide](setup.md)** - Complete setup instructions for all platforms and clients
- **[Security](SECURITY.md)** - Security best practices and guidelines
- **[Auto-Update](AUTOUPDATE.md)** - Automatic update system documentation

### Core Features
- **[Docker Isolation](docker-isolation.md)** - Secure server isolation with Docker containers
- **[OAuth Integration](mcp-go-oauth.md)** - OAuth 2.1 authentication for MCP servers
- **[Semantic Search](SEMANTIC_SEARCH.md)** - Hybrid BM25 search with configurable modes
- **[Repository Detection](repository-detection.md)** - Automatic package detection and installation
- **[Logging System](logging.md)** - Logging configuration and debugging
- **[Search Servers](search_servers.md)** - MCP server registry search and discovery
- **[LLM Configuration](LLM_CONFIGURATION.md)** - Multi-provider LLM configuration for AI diagnostic agent

### Architecture & Design
- **[State Management](STATE_MANAGEMENT.md)** - Event-driven state management with WebSocket support
- **[State Architecture](STATE_ARCHITECTURE.md)** - Three-tier state hierarchy (AppState, ServerState, ConnectionState)
- **[State Flow](STATE_FLOW.md)** - Complete flow from config → runtime → app state aggregation
- **[Runtime Stopped State](RUNTIME_STOPPED_STATE.md)** - Implementation of runtime-only stopped state (not persisted)
- **[WebSocket Implementation](WEBSOCKET_IMPLEMENTATION.md)** - Real-time event streaming architecture
- **[System Design](architecture/DESIGN.md)** - Overall architecture and design decisions
- **[Memory Management](architecture/MEMORY.md)** - Memory handling and optimization
- **[Menu Enhancement](architecture/MENU_UPDATE_ENHANCEMENT.md)** - System tray menu updates
- **[Agent API](AGENT_API.md)** - Background agent architecture and endpoints
- **[MCP Agent Design](MCP_AGENT_DESIGN.md)** - Agent communication patterns

### Testing & Quality
- **[E2E Production Scale Testing](E2E_PRODUCTION_SCALE_TESTING.md)** - Production-scale testing strategy
- **[Code Review Guidelines](code-review.md)** - Code review best practices

### Reports & Troubleshooting
- **[Server Connection Issues](reports/server_connection_issues.md)** - Known issues and solutions
- **[Tray Investigation Report](reports/tray-investigation-2025-10-17.md)** - System tray analysis
- **[Project Cleanup Report](reports/project-cleanup-2025-10-17.md)** - Repository cleanup summary

### Historical Documentation
See [archive/](archive/) folder for historical design documents, diagnostic reports, and implementation progress tracking.

## 🚀 Quick Start

1. **Install MCPProxy**
   ```bash
   # macOS (Homebrew)
   brew install smart-mcp-proxy/mcpproxy/mcpproxy

   # Go Install
   go install github.com/smart-mcp-proxy/mcpproxy-go/cmd/mcpproxy@latest
   ```

2. **Start the Server**
   ```bash
   mcpproxy serve
   ```

3. **Configure Your Client**
   - See [Setup Guide](setup.md) for detailed client configuration

## 📖 Key Concepts

### MCP Proxy Architecture
MCPProxy acts as an intelligent proxy between MCP clients (like Cursor IDE, VS Code, Claude Desktop) and upstream MCP servers. It provides:

- **Unified Interface**: Single endpoint for multiple MCP servers
- **Tool Discovery**: Intelligent BM25 search and indexing
- **Security**: Docker isolation and quarantine system
- **Performance**: Caching and connection pooling
- **OAuth Support**: Built-in RFC 8252 compliant authentication
- **Real-Time Updates**: WebSocket event streaming
- **State Management**: Event-driven architecture with persistence

### Configuration
Configuration file location: `~/.mcpproxy/mcp_config.json`

Basic structure:
```json
{
  "listen": ":8080",
  "enable_tray": true,
  "check_server_repo": true,
  "docker_isolation": {
    "enabled": true,
    "memory_limit": "512m",
    "cpu_limit": "1.0"
  },
  "mcpServers": []
}
```

### Startup Modes
Servers use `startup_mode` instead of multiple boolean flags:
- `active` - Start on boot, auto-reconnect on failure
- `lazy_loading` - Start on first tool call only
- `disabled` - User disabled, no connection
- `quarantined` - Security quarantine, no tool execution
- `auto_disabled` - Disabled after connection failures (requires manual re-enable)

## 🔧 Common Tasks

### Adding an MCP Server
Edit `~/.mcpproxy/mcp_config.json`:
```json
{
  "mcpServers": [
    {
      "name": "my-server",
      "command": "python",
      "args": ["-m", "my_mcp_server"],
      "protocol": "stdio",
      "startup_mode": "active",
      "working_dir": "/path/to/project"
    }
  ]
}
```

### Managing Server State
```bash
# Via tool calls (recommended)
mcpproxy call tool --tool-name=upstream_servers \
  --json_args='{"operation":"update","name":"server-name","startup_mode":"active"}'

# Enable all servers in a group
mcpproxy call tool --tool-name=groups \
  --json_args='{"operation":"enable_group","group_name":"Production"}'
```

### Debugging
```bash
# View logs
tail -f ~/Library/Logs/mcpproxy/main.log

# Debug specific server
tail -f ~/Library/Logs/mcpproxy/server-{name}.log

# Start with debug logging
./mcpproxy serve --log-level=debug

# Test OAuth flow
mcpproxy auth login --server=ServerName --log-level=debug
```

### Searching MCP Servers
```bash
# Search Smithery registry
mcpproxy call tool --tool-name=search_servers \
  --json_args='{"registry":"smithery","search":"database"}'

# Search with semantic search (hybrid mode)
mcpproxy call tool --tool-name=retrieve_tools \
  --json_args='{"query":"create github issue","mode":"hybrid"}'
```

### WebSocket Event Monitoring
```bash
# Connect to all events
ws://localhost:8080/ws/events

# Filter by server
ws://localhost:8080/ws/servers?server=github-server
```

## 🛡️ Security

See [Security Documentation](SECURITY.md) for:
- Docker isolation best practices
- Network security configuration
- Secure credential management
- Quarantine system details
- OAuth security considerations

## 📊 Monitoring & Observability

- **Logs**: Per-server logs in `~/Library/Logs/mcpproxy/` (macOS) or `~/.mcpproxy/logs/` (Linux)
- **Events**: Real-time WebSocket event streams
- **Metrics**: Tool usage statistics via `tools_stat` endpoint
- **Health Checks**: Application and server-level health monitoring

## 🆘 Getting Help

- **Documentation**: Read the relevant guide above
- **Issues**: [GitHub Issues](https://github.com/smart-mcp-proxy/mcpproxy-go/issues)
- **Logs**: Check `~/Library/Logs/mcpproxy/main.log` for errors
- **Debug Mode**: Run with `--log-level=debug` for detailed output

## 📝 Contributing

See our [Contributing Guide](../CONTRIBUTING.md) for:
- Development setup
- Code style guidelines
- Pull request process
- Testing requirements

---

**Last updated**: November 2025
**Current Version**: See latest git tags for version information

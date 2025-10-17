# MCPProxy Documentation

Welcome to the MCPProxy documentation. This guide will help you understand, configure, and use MCPProxy effectively.

## 📚 Documentation Structure

### Getting Started
- **[Setup Guide](setup.md)** - Complete setup instructions for all platforms and clients
- **[Security](SECURITY.md)** - Security best practices and guidelines
- **[Auto-Update](AUTOUPDATE.md)** - Automatic update system documentation

### Features & Configuration
- **[Docker Isolation](docker-isolation.md)** - Secure server isolation with Docker
- **[Repository Detection](repository-detection.md)** - Automatic package detection and installation
- **[Logging System](logging.md)** - Logging configuration and debugging
- **[OAuth Integration](mcp-go-oauth.md)** - OAuth authentication for MCP servers
- **[Search Servers](search_servers.md)** - MCP server registry search and discovery

### Architecture & Design
- **[System Design](architecture/DESIGN.md)** - Overall architecture and design decisions
- **[Memory Management](architecture/MEMORY.md)** - Memory handling and optimization
- **[Menu Enhancement](architecture/MENU_UPDATE_ENHANCEMENT.md)** - System tray menu updates

### Reports & Troubleshooting
- **[Server Connection Issues](reports/server_connection_issues.md)** - Known issues and solutions

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
MCPProxy acts as an intelligent proxy between MCP clients (like Cursor IDE, VS Code) and upstream MCP servers. It provides:

- **Unified Interface**: Single endpoint for multiple MCP servers
- **Tool Discovery**: Intelligent tool search and indexing
- **Security**: Docker isolation for stdio servers
- **Performance**: Caching and connection pooling
- **OAuth Support**: Built-in authentication handling

### Configuration
Configuration file location: `~/.mcpproxy/mcp_config.json`

Basic structure:
```json
{
  "listen": ":8080",
  "enable_tray": true,
  "check_server_repo": true,
  "docker_isolation": {
    "enabled": false
  },
  "mcpServers": []
}
```

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
      "type": "stdio",
      "enabled": true
    }
  ]
}
```

### Debugging
```bash
# View logs
mcpproxy serve --log-level=debug --tray=false

# Test individual server
mcpproxy tools list --server=my-server --log-level=trace
```

### Searching MCP Servers
```bash
# Search Smithery registry
mcpproxy search-servers --registry smithery --search database

# Search MCP Pulse registry
mcpproxy search-servers --registry pulse --search weather --limit 5
```

## 🛡️ Security

See [Security Documentation](SECURITY.md) for:
- Docker isolation best practices
- Network security configuration
- Secure credential management
- Read-only mode settings

## 🆘 Getting Help

- **Documentation**: Read the relevant guide above
- **Issues**: [GitHub Issues](https://github.com/smart-mcp-proxy/mcpproxy-go/issues)
- **Community**: Join our Discord/Slack
- **Website**: [mcpproxy.app](https://mcpproxy.app)

## 📝 Contributing

See our [Contributing Guide](../CONTRIBUTING.md) for:
- Development setup
- Code style guidelines
- Pull request process
- Testing requirements

---

*Last updated: October 2025*

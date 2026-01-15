# Standalone MCP Server Agent Usage

## Overview

The AgentClient now supports launching and communicating with MCP servers independently, without routing through the main mcpproxy server. This enables direct server interaction for diagnostic purposes, testing, and standalone operations.

## Features

### 1. **Standalone Server Management**
Launch and manage MCP servers independently:
- Process lifecycle management (start, stop, monitor)
- State tracking (stopped, starting, running, stopping, error)
- Automatic cleanup and resource management

### 2. **Direct MCP Communication**
Communicate directly with MCP servers:
- Tool discovery via ListTools
- Tool execution via CallTool
- Server info retrieval
- Batch tool operations

### 3. **Markdown Report Generation**
Generate comprehensive markdown reports:
- Diagnostic reports with issues and recommendations
- Tool discovery reports with schemas
- Server analysis reports with metrics

## Architecture

### Core Components

#### `StandaloneServerManager` (`agent_standalone.go`)
Manages independent MCP server processes without main mcpproxy:
```go
type StandaloneServerManager struct {
    servers      map[string]*StandaloneServer
    logger       *zap.Logger
    globalConfig *config.Config
    logConfig    *config.LogConfig
}
```

#### `StandaloneMCPClient` (`agent_mcp.go`)
Provides direct MCP protocol communication:
```go
type StandaloneMCPClient struct {
    server *StandaloneServer
    logger *zap.Logger
}
```

#### `ReportGenerator` (`agent_reporter.go`)
Generates markdown reports from findings:
```go
type ReportGenerator struct {
    logger    *zap.Logger
    templates map[ReportType]string
}
```

## Usage Examples

### Example 1: Launch Standalone Server

```go
// Create AgentClient
agentClient := tray.NewAgentClient("http://localhost:5000", logger)

// Configure server
serverConfig := &config.ServerConfig{
    Name:     "test-server",
    Command:  "npx",
    Args:     []string{"@modelcontextprotocol/server-filesystem"},
    Protocol: "stdio",
    Enabled:  true,
}

// Launch standalone server
server, err := agentClient.LaunchStandaloneServer(serverConfig)
if err != nil {
    log.Fatal(err)
}

log.Printf("Server launched: %s (State: %s)", server.ID, server.GetState())
```

### Example 2: Discover Tools

```go
// Get the standalone server
server, exists := agentClient.GetStandaloneServer(serverID)
if !exists {
    log.Fatal("Server not found")
}

// Discover tools
result, err := agentClient.PerformStandaloneToolDiscovery(server)
if err != nil {
    log.Fatal(err)
}

log.Printf("Discovered %d tools in %v", len(result.Tools), result.Duration)
for _, toolName := range result.ToolNames {
    log.Printf("  - %s", toolName)
}
```

### Example 3: Call a Tool

```go
// Call a tool directly
args := map[string]interface{}{
    "path": "/home/user/projects",
}

result, err := agentClient.CallStandaloneTool(server, "list_directory", args)
if err != nil {
    log.Fatal(err)
}

log.Printf("Tool execution result: %+v", result.Result)
```

### Example 4: Generate Tool Discovery Report

```go
// Discover and catalog tools
catalog, err := agentClient.DiscoverToolCatalog(server)
if err != nil {
    log.Fatal(err)
}

// Generate markdown report
markdown, err := agentClient.GenerateToolDiscoveryMarkdown(catalog)
if err != nil {
    log.Fatal(err)
}

// Save report
err = agentClient.SaveMarkdownReport(markdown, "tool-discovery.md")
if err != nil {
    log.Fatal(err)
}
```

### Example 5: Generate Server Analysis Report

```go
// Track multiple operations
operations := []*tray.MCPOperationResult{}

// Perform various operations
listResult, _ := agentClient.PerformStandaloneToolDiscovery(server)
operations = append(operations, listResult)

callResult, _ := agentClient.CallStandaloneTool(server, "some_tool", args)
operations = append(operations, callResult)

// Generate comprehensive analysis
markdown, err := agentClient.GenerateServerAnalysisMarkdown(server, operations)
if err != nil {
    log.Fatal(err)
}

// Save to reports directory
filepath, err := agentClient.SaveMarkdownReportToDir(
    markdown,
    "./reports",
    "server-analysis",
)
log.Printf("Report saved to: %s", filepath)
```

### Example 6: Batch Tool Calls

```go
// Create batch tool calls
calls := []tray.StandaloneToolCall{
    {
        ToolName: "list_directory",
        Arguments: map[string]interface{}{"path": "/home"},
    },
    {
        ToolName: "read_file",
        Arguments: map[string]interface{}{"path": "/etc/hosts"},
    },
}

// Create MCP client for batch operations
mcpClient := tray.NewStandaloneMCPClient(server)

// Execute batch
results, err := mcpClient.BatchCallTools(context.Background(), calls)
if err != nil {
    log.Fatal(err)
}

// Process results
for i, result := range results {
    if result.Success {
        log.Printf("Call %d succeeded in %v", i+1, result.Duration)
    } else {
        log.Printf("Call %d failed: %v", i+1, result.Error)
    }
}
```

## API Reference

### AgentClient Methods

#### Standalone Server Management
```go
// Launch a new standalone MCP server
LaunchStandaloneServer(cfg *config.ServerConfig) (*StandaloneServer, error)

// Stop a standalone server
StopStandaloneServer(serverID string) error

// Get server by ID
GetStandaloneServer(serverID string) (*StandaloneServer, bool)

// List all standalone servers
ListStandaloneServers() []*StandaloneServer

// Cleanup all standalone servers
CleanupStandaloneServers() error
```

#### Direct MCP Operations
```go
// Perform tool discovery
PerformStandaloneToolDiscovery(server *StandaloneServer) (*MCPOperationResult, error)

// Call a tool
CallStandaloneTool(server *StandaloneServer, toolName string, args map[string]interface{}) (*MCPOperationResult, error)

// Discover and catalog tools
DiscoverToolCatalog(server *StandaloneServer) (*ToolCatalog, error)

// Test server connection
TestStandaloneConnection(server *StandaloneServer) error
```

#### Report Generation
```go
// Generate diagnostic report markdown
GenerateDiagnosticMarkdown(report *DiagnosticReport) (string, error)

// Generate tool discovery report markdown
GenerateToolDiscoveryMarkdown(catalog *ToolCatalog) (string, error)

// Generate server analysis report markdown
GenerateServerAnalysisMarkdown(server *StandaloneServer, operations []*MCPOperationResult) (string, error)

// Save markdown report to file
SaveMarkdownReport(report string, filename string) error

// Save markdown report to directory with timestamp
SaveMarkdownReportToDir(report string, dir string, prefix string) (string, error)
```

## Server States

Standalone servers track their lifecycle through these states:

- **stopped**: Server is not running
- **starting**: Server is initializing
- **running**: Server is active and accepting requests
- **stopping**: Server is shutting down
- **error**: Server encountered an error

## Report Types

Three types of markdown reports are supported:

### 1. Diagnostic Report
- Executive summary
- Issues detected
- Recommendations
- Configuration analysis
- Log analysis
- Repository analysis
- AI analysis
- Suggested configuration

### 2. Tool Discovery Report
- Server metadata
- Tool catalog with names, descriptions, and schemas
- Discovery metrics (duration, tool count)

### 3. Server Analysis Report
- Server configuration
- Uptime and state
- Operations summary (success rate, average duration)
- Detailed operation logs
- Performance metrics

## Best Practices

1. **Resource Management**: Always cleanup standalone servers when done:
   ```go
   defer agentClient.CleanupStandaloneServers()
   ```

2. **Error Handling**: Check server state before operations:
   ```go
   if !server.IsRunning() {
       log.Printf("Server not running: %s", server.GetError())
   }
   ```

3. **Timeouts**: Use context with timeout for operations:
   ```go
   ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
   defer cancel()
   result, err := mcpClient.ListTools(ctx)
   ```

4. **Logging**: Standalone servers use the existing logging infrastructure:
   - Server-specific logs in `~/.mcpproxy/logs/server-{name}.log`
   - Main logs in `~/.mcpproxy/logs/main.log`

5. **Report Organization**: Use the directory-based save for reports:
   ```go
   filepath, _ := agentClient.SaveMarkdownReportToDir(markdown, "./reports", "diagnostic")
   // Generates: ./reports/diagnostic-20250101-150405.md
   ```

## Integration with AgentClient

The standalone functionality integrates seamlessly with the existing AgentClient:

```go
// Create single client instance
agentClient := tray.NewAgentClient("http://localhost:5000", logger)

// Use existing Python agent operations
diagReport, _ := agentClient.DiagnoseServer(serverName)

// Use new standalone operations
standaloneServer, _ := agentClient.LaunchStandaloneServer(serverConfig)
toolCatalog, _ := agentClient.DiscoverToolCatalog(standaloneServer)

// Generate reports from both
pythonAgentMarkdown, _ := agentClient.GenerateDiagnosticMarkdown(diagReport)
standaloneMarkdown, _ := agentClient.GenerateToolDiscoveryMarkdown(toolCatalog)
```

## Troubleshooting

### Server Won't Start
- Check server configuration (command, args, protocol)
- Verify server binary is installed and accessible
- Check logs: `server.GetError()` or `~/.mcpproxy/logs/server-{name}.log`

### Tool Discovery Fails
- Ensure server is in "running" state: `server.IsRunning()`
- Check if server supports tools capability
- Verify server logs for initialization errors

### Connection Timeouts
- Increase timeout in context: `context.WithTimeout(ctx, 60*time.Second)`
- Check if server process is responsive
- Verify no firewall or network issues

### Report Generation Errors
- Ensure valid data structures (non-nil fields)
- Check write permissions for output directory
- Verify markdown template compatibility

## Summary

The standalone MCP server agent provides:
- ✅ Independent server management
- ✅ Direct MCP protocol communication
- ✅ Comprehensive markdown reporting
- ✅ Integration with existing AgentClient
- ✅ Full lifecycle management
- ✅ Detailed metrics and diagnostics

This enables powerful diagnostic and testing workflows without requiring the main mcpproxy routing infrastructure.

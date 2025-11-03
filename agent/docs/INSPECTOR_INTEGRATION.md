# MCP Inspector Integration

Live MCP protocol debugging through conversational agent control.

## Overview

The Python MCP agent now has built-in control of the MCP Inspector, allowing users to:
- **Start/stop the inspector** through natural language chat
- **Monitor live MCP protocol** interactions in real-time
- **Watch tool calls and responses** as they happen
- **Debug server connections** with visual feedback

## Architecture

```
User Chat Input
    ↓
Agent Request Analysis
    ↓
Inspector Task Detection
    ↓
Inspector Tools (Python)
    ↓
HTTP API Call (mcpproxy Go server)
    ↓
InspectorManager (Go)
    ↓
npx @modelcontextprotocol/inspector
    ↓
Web UI (Browser)
```

### Components

1. **InspectorTools** (`mcp_agent/tools/inspector.py`)
   - Python wrapper for inspector control
   - HTTP client for mcpproxy API
   - Browser integration for automatic opening

2. **Agent Graph Integration** (`mcp_agent/graph/agent_graph.py`)
   - Inspector task type detection
   - Dedicated inspector node in workflow
   - Conversational response generation

3. **Go Inspector Manager** (`internal/server/inspector.go`)
   - Process lifecycle management
   - Port detection and URL generation
   - Auth token integration

## Usage

### Starting the Inspector

**Natural Language**:
```
User: "Start the inspector so I can watch the interaction"
User: "Open the inspector"
User: "Launch inspector to see what's happening"
User: "Show me the live MCP protocol"
```

**Agent Response**:
```
Completed: Starting MCP Inspector

Actions Taken:
  - Started MCP Inspector

Recommendations:
  - Inspector is now running at: http://localhost:5173/?MCP_PROXY_AUTH_TOKEN=abc123...
  - You can watch the live MCP protocol interaction in your browser
```

### Checking Inspector Status

**Natural Language**:
```
User: "Is the inspector running?"
User: "Check inspector status"
User: "What's the inspector URL?"
```

**Agent Response** (if running):
```
Completed: Checking Inspector Status

Actions Taken:
  - Checked Inspector status

Recommendations:
  - Inspector is running at: http://localhost:5173/?MCP_PROXY_AUTH_TOKEN=abc123...
```

**Agent Response** (if not running):
```
Completed: Checking Inspector Status

Actions Taken:
  - Checked Inspector status

Recommendations:
  - Inspector is not currently running
  - Use 'start inspector' to launch it
```

### Stopping the Inspector

**Natural Language**:
```
User: "Stop the inspector"
User: "Close inspector"
User: "End inspector session"
```

**Agent Response**:
```
Completed: Stopping MCP Inspector

Actions Taken:
  - Stopped MCP Inspector
```

## Conversational Workflow

### Complete Example

```python
from mcp_agent.tools.inspector import InspectorTools
from mcp_agent.graph.agent_graph import MCPAgentGraph, AgentInput

# Initialize agent with inspector tools
tools_registry = {
    "inspector": InspectorTools(base_url="http://localhost:8080"),
    # ... other tools
}

agent = MCPAgentGraph(tools_registry)
thread_id = "my-session"

# User wants to debug an issue
result = await agent.run(
    AgentInput(request="Start the inspector so I can see what's happening"),
    thread_id=thread_id
)
print(result.recommendations[0])
# "Inspector is now running at: http://localhost:5173/?MCP_PROXY_AUTH_TOKEN=..."

# User opens browser, watches live protocol
# Agent continues to process requests
# User sees all MCP tool calls and responses in real-time

# Later, user is done debugging
result = await agent.run(
    AgentInput(request="Stop the inspector now"),
    thread_id=thread_id
)
# Inspector process cleanly terminates
```

## Features

### 1. Live Protocol Visualization
- **Real-time MCP messages**: See every request and response
- **Tool call inspection**: Examine parameters and return values
- **Server connection status**: Monitor connection health
- **Error debugging**: Identify protocol-level issues

### 2. Natural Language Control
- **Conversational interface**: No commands to memorize
- **Intent detection**: Keywords like "start", "stop", "status"
- **Contextual help**: Agent suggests next steps

### 3. Automatic Browser Integration
- **Auto-open**: Browser launches automatically on start
- **Auth token included**: URL contains authentication
- **One-click access**: No manual URL copying

### 4. Session Persistence
- **Inspector state**: Survives across chat sessions
- **Thread isolation**: Different sessions can have different inspector states
- **Cleanup on exit**: Graceful shutdown handling

## Technical Details

### Keyword Detection

The agent detects inspector requests using these keywords:

**Inspector Task**:
- `inspector`, `inspect`, `watch`, `visualize`, `live debug`

**Start Action**:
- `start`, `open`, `launch`, `show`

**Stop Action**:
- `stop`, `close`, `end`

**Status Action**:
- `status`, `running`, `check`

### HTTP API Endpoints

InspectorTools communicates with:
- `POST /inspector/start` - Start inspector process
- `POST /inspector/stop` - Stop inspector process
- `GET /inspector/status` - Get current status

### Response Format

```python
class InspectorStartResponse(BaseModel):
    success: bool  # True if started successfully
    message: str   # Status message
    url: str | None  # Inspector URL with auth token

class InspectorStopResponse(BaseModel):
    success: bool  # True if stopped successfully
    message: str   # Status message

class InspectorStatus(BaseModel):
    running: bool  # True if currently running
    url: str | None  # Inspector URL if running
```

## Testing

### Running Tests

```bash
cd agent
python3 test_inspector_integration.py
```

### Expected Output

```
======================================================================
Testing MCP Inspector Integration with Python Agent
======================================================================

✓ Agent initialized with inspector tools

======================================================================
Test 1: Starting MCP Inspector
======================================================================
User: 'Start the inspector so I can watch the interaction'

Agent Response: Completed: Starting MCP Inspector
Actions Taken: ['Started MCP Inspector']
Recommendations:
  - Inspector is now running at: http://localhost:5173/?MCP_PROXY_AUTH_TOKEN=...
  - You can watch the live MCP protocol interaction in your browser

======================================================================
Test 2: Checking Inspector Status
======================================================================
User: 'Is the inspector running?'

Agent Response: Completed: Checking Inspector Status
Actions Taken: ['Checked Inspector status']
Recommendations:
  - Inspector is running at: http://localhost:5173/?MCP_PROXY_AUTH_TOKEN=...

======================================================================
Integration Test Summary
======================================================================
✅ Inspector can be started via natural language
✅ Inspector status can be checked conversationally
✅ Inspector can be stopped via chat
✅ User can watch live MCP protocol interaction

The agent now has full control of the MCP Inspector!
======================================================================
```

## Use Cases

### 1. Debugging Server Issues
```
User: "I'm having issues with the GitHub server, can you start the inspector?"
Agent: Starts inspector
User: Opens browser, watches live MCP protocol
Agent: "List all GitHub tools"
User: Sees the tool discovery protocol in real-time
User: Identifies the connection issue
```

### 2. Learning MCP Protocol
```
User: "I want to learn how MCP works, can you show me?"
Agent: Starts inspector
User: "Call a simple tool"
Agent: Executes tool
User: Watches the request/response cycle in the inspector
User: Understands the protocol flow
```

### 3. Performance Monitoring
```
User: "Let's monitor the MCP server performance"
Agent: Starts inspector
User: Monitors response times
User: Identifies slow tools
User: "Stop the inspector"
Agent: Cleans up
```

## Error Handling

### Inspector Already Running
```
User: "Start the inspector"
Agent: "MCP Inspector is already running at: ..."
```

### Inspector Not Installed
```
Error: Failed to start inspector: npx @modelcontextprotocol/inspector not found
Solution: Install with: npm install -g @modelcontextprotocol/inspector
```

### mcpproxy Not Running
```
Error: Failed to connect to mcpproxy server at http://localhost:8080
Solution: Start mcpproxy: ./mcpproxy serve
```

## Future Enhancements

### Planned Features
- [ ] **Event streaming**: Real-time protocol events in chat
- [ ] **Automated testing**: Generate test cases from inspector data
- [ ] **Protocol recording**: Save and replay MCP sessions
- [ ] **Multi-inspector**: Support multiple inspectors for different servers
- [ ] **Performance analytics**: Automated performance reports

### Integration Ideas
- **Claude Flow coordination**: Multi-agent inspector orchestration
- **Automated debugging**: Agent suggests fixes based on inspector data
- **Live documentation**: Generate docs from observed protocol
- **Testing workflows**: Record user interactions as test cases

## Troubleshooting

### Problem: Browser doesn't open
**Cause**: Headless environment or browser access blocked

**Solution**:
```python
# Disable auto-open
result = await inspector_tools.start_inspector(open_browser=False)
# Manually open the URL
print(f"Open this URL: {result.url}")
```

### Problem: Port already in use
**Cause**: Previous inspector process still running

**Solution**:
```bash
# Find and kill inspector process
pkill -f "mcp-inspector"
# Or restart mcpproxy
./mcpproxy serve
```

### Problem: Inspector URL not detected
**Cause**: Output parsing issue or slow startup

**Solution**:
- Wait 5-10 seconds for full startup
- Check mcpproxy logs: `tail -f ~/Library/Logs/mcpproxy/main.log`
- Manually check inspector status: `curl http://localhost:8080/inspector/status`

## Best Practices

### 1. Start Inspector Early
Start the inspector before complex debugging sessions:
```
User: "I need to debug multiple servers, start the inspector first"
Agent: Starts inspector
User: Proceeds with debugging while watching protocol
```

### 2. Use Descriptive Requests
Be specific about what you want to inspect:
```
Good: "Start the inspector so I can see the GitHub API calls"
Better than: "Inspector"
```

### 3. Clean Up After Sessions
Always stop the inspector when done:
```
User: "I'm done debugging, stop the inspector"
Agent: Cleans up inspector process
```

### 4. Combine with Other Agent Features
Use inspector alongside other agent capabilities:
```
User: "Start the inspector and diagnose the filesystem server"
Agent: Starts inspector, runs diagnostics
User: Watches diagnostic protocol in real-time
```

## Related Documentation

- [Session Memory](SESSION_MEMORY.md) - Agent session persistence
- [Context Compaction](../CONTEXT_COMPACTION_SUMMARY.md) - Token optimization
- [Python Agent Integration](../PYTHON_AGENT_INTEGRATION_SUMMARY.md) - Overall architecture
- [Go Inspector Implementation](../../internal/server/inspector.go) - Backend details

## Summary

The MCP Inspector integration provides:
- ✅ **Natural language control** of debugging tools
- ✅ **Live protocol visualization** in browser
- ✅ **Conversational workflow** for debugging
- ✅ **Automatic lifecycle management** of inspector process
- ✅ **Session-aware** inspector state
- ✅ **Production-ready** error handling and cleanup

**The agent now enables interactive, visual MCP protocol debugging through simple conversation!**

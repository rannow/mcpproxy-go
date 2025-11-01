# MCP Server Management Agent

An intelligent AI agent for managing, debugging, and optimizing MCP (Model Context Protocol) servers within the mcpproxy ecosystem.

## 🎯 Features

- **🔍 Intelligent Diagnostics**: Analyze logs, identify errors, and suggest fixes
- **✅ Automated Testing**: Validate server functionality and tool responses
- **📦 Server Discovery**: Search and install new MCP servers from registries
- **⚙️ Configuration Management**: Read, modify, and validate server configurations
- **📊 Log Analysis**: AI-powered analysis of mcpproxy and server-specific logs
- **🚀 Startup Management**: Manage startup scripts, dependencies, and Docker services
- **📚 Documentation Access**: Search and retrieve MCP server documentation
- **🤖 Autonomous Operation**: Self-learning agent with memory and context retention

## 🏗️ Architecture

This agent uses a **hybrid LangGraph + PydanticAI** architecture:

- **LangGraph**: State machine orchestration and workflow management
- **PydanticAI**: Type-safe tool definitions and LLM-agnostic execution
- **SQLite Memory**: Persistent conversation history and knowledge base
- **MCPProxy Integration**: REST API for seamless integration

## 📋 Prerequisites

- Python 3.11+
- mcpproxy running with agent API enabled
- Anthropic API key (or other LLM provider)

## 🚀 Quick Start

### 1. Installation

```bash
cd agent/

# Using Poetry (recommended)
poetry install

# Or using pip
python -m venv venv
source venv/bin/activate  # On Windows: venv\Scripts\activate
pip install -r requirements.txt
```

### 2. Configuration

Create a `.env` file:

```bash
# LLM Configuration
ANTHROPIC_API_KEY=your-api-key-here

# MCPProxy Configuration
MCPPROXY_URL=http://localhost:8080
MCPPROXY_API_TOKEN=optional-token

# Agent Configuration
AGENT_AUTO_APPROVE_SAFE_FIXES=false
AGENT_LOG_LEVEL=info
```

### 3. Start MCPProxy with Agent API

First, you need to extend mcpproxy with the agent API endpoints. See the implementation guide in `docs/MCP_AGENT_DESIGN.md`.

```bash
# Start mcpproxy (with agent API enabled)
./mcpproxy serve --enable-agent-api
```

### 4. Run the Agent

```bash
# Interactive mode
poetry run python -m mcp_agent.cli chat --interactive

# Single command
poetry run python -m mcp_agent.cli diagnose github-server

# Auto-fix mode
poetry run python -m mcp_agent.cli diagnose github-server --auto-fix

# Test a server
poetry run python -m mcp_agent.cli test my-server --tool get_weather

# Search for servers
poetry run python -m mcp_agent.cli search "weather api"

# Check status
poetry run python -m mcp_agent.cli status --server github-server
```

## 📖 Usage Examples

### Diagnose a Failing Server

```bash
$ poetry run python -m mcp_agent.cli chat "The github-server keeps failing"

🤖 Analyzing server...

Diagnosis Complete
──────────────────

Server: github-server
Status: Error (OAuth token expired)

Issues Found:
• OAuth authentication failed (last 15 minutes)
• Connection timeout errors (12 occurrences)
• Tool execution failures: create_issue (3), get_repo (2)

Recommended Fixes:
1. Re-authenticate OAuth
   Command: mcpproxy auth login --server=github-server
   Risk: Low | Requires approval: Yes

2. Increase timeout values
   Update config: {"timeout": "120s"}
   Risk: Low | Requires approval: No

Would you like me to apply these fixes? (y/n):
```

### Interactive Chat Mode

```bash
$ poetry run python -m mcp_agent.cli chat --interactive

MCP Agent Interactive Mode
Type 'exit' to quit

You: List all servers with errors

Agent: I found 3 servers with errors:
1. github-server: OAuth expired
2. sentry-server: Connection timeout
3. weather-api: Rate limited

You: Fix the github server

Agent: I've analyzed github-server. The issue is an expired OAuth token.

Suggested fix:
• Re-authenticate with GitHub OAuth
• Command: mcpproxy auth login --server=github-server

Should I proceed? (yes/no)

You: yes

Agent: ✅ Successfully re-authenticated github-server
Monitoring for 60 seconds... Server is now stable.

You: exit

Goodbye!
```

### Search and Install New Server

```bash
$ poetry run python -m mcp_agent.cli search "weather forecast"

Found 5 MCP servers:

1. weather-api (v1.2.0)
   Description: Real-time weather data from OpenWeatherMap
   Registry: smithery
   Install: npx @weather/mcp-server

2. forecast-io (v2.0.1)
   Description: Weather forecasts and historical data
   Registry: mcprun
   Install: uvx forecast-io-mcp

...

$ poetry run python -m mcp_agent.cli install weather-api

Installing weather-api from smithery...
✅ Installed successfully
📝 Configuration created at ~/.mcpproxy/mcp_config.json
🔧 Testing server connection... OK
✅ weather-api is ready to use
```

## 🛠️ Development

### Project Structure

```
agent/
├── mcp_agent/
│   ├── agents/          # Specialized agent implementations
│   ├── tools/           # PydanticAI tool definitions
│   │   ├── diagnostic.py    # Debug and diagnostics
│   │   ├── config.py        # Configuration management
│   │   ├── discovery.py     # Server discovery
│   │   ├── testing.py       # Testing tools
│   │   ├── logs.py          # Log analysis
│   │   ├── docs.py          # Documentation
│   │   └── startup.py       # Startup management
│   ├── graph/           # LangGraph state machines
│   │   └── agent_graph.py   # Main orchestration graph
│   ├── utils/           # Utilities
│   └── cli.py           # CLI interface
├── tests/               # Test suite
├── docs/                # Documentation
├── pyproject.toml       # Poetry configuration
└── README.md
```

### Running Tests

```bash
# Run all tests
poetry run pytest

# Run with coverage
poetry run pytest --cov=mcp_agent tests/

# Run specific test file
poetry run pytest tests/test_diagnostic.py -v
```

### Code Quality

```bash
# Type checking
poetry run mypy mcp_agent/

# Linting
poetry run ruff check mcp_agent/

# Formatting
poetry run black mcp_agent/
```

## 🔧 Configuration

### Agent Configuration File

Create `~/.mcpproxy/agent_config.json`:

```json
{
  "llm": {
    "provider": "anthropic",
    "model": "claude-3-5-sonnet-20241022",
    "api_key_env": "ANTHROPIC_API_KEY",
    "fallback_models": ["claude-3-haiku-20240307"]
  },
  "capabilities": {
    "auto_fix": true,
    "auto_install": false,
    "require_approval": true,
    "max_retries": 3
  },
  "memory": {
    "type": "sqlite",
    "path": "~/.mcpproxy/agent_memory.db",
    "max_conversations": 100,
    "retention_days": 30
  },
  "logging": {
    "level": "info",
    "file": "~/.mcpproxy/logs/agent.log"
  }
}
```

## 🧠 How It Works

### 1. Request Analysis

The agent analyzes your request to determine:
- Task type (diagnose, test, configure, install, etc.)
- Target server
- Required tools
- Approval requirements

### 2. State Machine Execution

LangGraph state machine orchestrates the workflow:

```
ANALYZE REQUEST → CHECK STATUS → DIAGNOSE → SUGGEST FIXES
                                              ↓
                                    AWAIT APPROVAL (if needed)
                                              ↓
                                    EXECUTE FIXES → MONITOR → REPORT
```

### 3. Tool Execution

PydanticAI tools interact with mcpproxy:
- Type-safe API calls
- Automatic validation
- Error handling and retries

### 4. AI-Powered Analysis

LLM analyzes results and provides:
- Root cause identification
- Fix suggestions
- Configuration recommendations
- Documentation references

### 5. Memory & Learning

Agent remembers:
- Previous issues and solutions
- Server configurations
- User preferences
- Successful fix patterns

## 📊 Monitoring & Observability

The agent provides detailed logging and metrics:

```bash
# View agent logs
tail -f ~/.mcpproxy/logs/agent.log

# View agent memory database
sqlite3 ~/.mcpproxy/agent_memory.db "SELECT * FROM conversations LIMIT 10;"

# Agent metrics (if enabled)
curl http://localhost:8081/metrics
```

## 🔒 Security

- **Authentication**: All API calls use token authentication
- **Approval Flow**: High-risk operations require user approval
- **Audit Log**: All agent actions are logged
- **Sandboxing**: Server installations run in isolated environments
- **Input Validation**: All inputs validated before processing
- **Rate Limiting**: Prevents excessive API calls

## 🚧 Roadmap

- [x] Framework selection and architecture design
- [x] Core diagnostic tools
- [x] Configuration management
- [x] LangGraph state machine
- [ ] Complete all tool implementations
- [ ] MCPProxy REST API endpoints
- [ ] Testing suite
- [ ] Web UI (Streamlit/Gradio)
- [ ] Multi-modal capabilities
- [ ] Predictive maintenance
- [ ] Self-learning improvements

## 📚 Documentation

- [Design Document](../docs/MCP_AGENT_DESIGN.md) - Complete technical design
- [API Reference](docs/api.md) - API documentation (TBD)
- [Contributing Guide](docs/CONTRIBUTING.md) - How to contribute (TBD)

## 🤝 Contributing

Contributions are welcome! See the implementation task list in `docs/MCP_AGENT_DESIGN.md` for areas that need work.

## 📄 License

[Same as mcpproxy]

## 🙏 Acknowledgments

Built with:
- [LangGraph](https://github.com/langchain-ai/langgraph) - Agent orchestration
- [PydanticAI](https://github.com/pydantic/pydantic-ai) - Type-safe AI tools
- [Rich](https://github.com/Textualize/rich) - Beautiful CLI
- [Typer](https://github.com/tiangolo/typer) - CLI framework

---

**Note**: This is a proof-of-concept implementation. Many features are still under development. See `docs/MCP_AGENT_DESIGN.md` for the complete implementation plan.

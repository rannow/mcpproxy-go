# MCP Server Management Agent - Design Document

## Executive Summary

This document outlines the design for an intelligent AI agent system to manage, debug, and optimize MCP (Model Context Protocol) servers within the mcpproxy ecosystem.

## 1. Framework Evaluation & Recommendation

### Framework Comparison Matrix

| Framework | Multi-Agent | Tool Integration | State Mgmt | LLM Flexibility | Go Integration | Complexity | Recommendation |
|-----------|-------------|------------------|------------|-----------------|----------------|------------|----------------|
| **LangGraph** | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐ | Medium | **PRIMARY** |
| **PydanticAI** | ⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐ | Low | **SECONDARY** |
| **CrewAI** | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐ | Medium | Alternative |
| **AutoGen** | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐ | High | Alternative |

### Recommended Architecture: **Hybrid LangGraph + PydanticAI**

#### Why This Combination?

1. **LangGraph** for orchestration and state management
   - Graph-based workflow perfect for MCP server lifecycle management
   - Built-in state persistence and checkpointing
   - Excellent debugging and observability
   - Native support for complex agent interactions

2. **PydanticAI** for tool definitions and validation
   - Type-safe tool definitions with Pydantic models
   - Easy integration with existing systems
   - Excellent for structured data handling (configs, logs)
   - LLM-agnostic design (Claude, GPT-4, Gemini, local models)

3. **Integration Strategy**
   - LangGraph manages the overall agent workflow
   - PydanticAI handles individual tool execution
   - Go mcpproxy exposes REST API for agent interaction
   - Python agent system communicates via HTTP/JSON-RPC

## 2. System Architecture

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                   MCP Management Agent System                │
├─────────────────────────────────────────────────────────────┤
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │  Diagnostic  │  │   Config     │  │  Discovery   │      │
│  │    Agent     │  │   Manager    │  │    Agent     │      │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘      │
│         │                  │                  │              │
│  ┌──────┴──────────────────┴──────────────────┴───────┐     │
│  │        LangGraph Orchestration Layer               │     │
│  │  (State Management, Workflow Control, Memory)      │     │
│  └──────────────────────┬─────────────────────────────┘     │
│                         │                                    │
│  ┌──────────────────────┴─────────────────────────────┐     │
│  │         PydanticAI Tool Execution Layer            │     │
│  │  (Type-Safe Tools, Validation, Error Handling)     │     │
│  └──────────────────────┬─────────────────────────────┘     │
└─────────────────────────┼─────────────────────────────────┘
                          │
                          ↓
┌─────────────────────────────────────────────────────────────┐
│                    MCPProxy Integration Layer                │
├─────────────────────────────────────────────────────────────┤
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐   │
│  │   REST   │  │  MCP     │  │  Config  │  │   Logs   │   │
│  │   API    │  │  Tools   │  │   API    │  │   API    │   │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘  └────┬─────┘   │
├───────┴─────────────┴─────────────┴─────────────┴──────────┤
│              mcpproxy (Go) - Existing System                 │
└─────────────────────────────────────────────────────────────┘
```

### Component Architecture

#### 1. Agent Layer (LangGraph State Machine)

```python
States:
- IDLE: Waiting for tasks
- ANALYZING: Examining server state
- DIAGNOSING: Identifying issues
- TESTING: Validating server functionality
- CONFIGURING: Modifying configurations
- INSTALLING: Adding new servers
- MONITORING: Observing server health
- REPORTING: Generating insights
```

#### 2. Specialized Agents

| Agent | Purpose | Key Capabilities |
|-------|---------|------------------|
| **Diagnostic Agent** | Debug & troubleshoot | Log analysis, error pattern detection, root cause analysis |
| **Config Manager** | Configuration management | Read/write configs, validate settings, backup/restore |
| **Discovery Agent** | Find & install servers | Search registries, evaluate servers, install & configure |
| **Testing Agent** | Validate functionality | Tool testing, integration tests, health checks |
| **Documentation Agent** | Knowledge retrieval | Read docs, extract information, provide guidance |
| **Startup Manager** | System initialization | Manage startup scripts, dependencies, Docker/services |

## 3. Core Capabilities & Functions

### 3.1 Debug & Diagnostics

```python
class DiagnosticTools:
    def analyze_server_logs(
        server_name: str,
        time_range: Optional[str] = None,
        error_patterns: Optional[List[str]] = None
    ) -> LogAnalysisResult:
        """Analyze server logs for errors and patterns"""

    def identify_connection_issues(
        server_name: str
    ) -> ConnectionDiagnostic:
        """Diagnose connection problems"""

    def analyze_tool_failures(
        server_name: str,
        tool_name: Optional[str] = None
    ) -> ToolFailureAnalysis:
        """Analyze tool execution failures"""

    def suggest_fixes(
        diagnostic_results: Dict[str, Any]
    ) -> List[Fix]:
        """AI-powered fix suggestions"""
```

### 3.2 Testing & Validation

```python
class TestingTools:
    def test_server_connection(
        server_name: str
    ) -> ConnectionTestResult:
        """Test server connectivity"""

    def test_tool_execution(
        server_name: str,
        tool_name: str,
        test_args: Dict[str, Any]
    ) -> ToolTestResult:
        """Execute tool with test arguments"""

    def validate_server_health(
        server_name: str
    ) -> HealthCheckResult:
        """Comprehensive health check"""

    def run_integration_tests(
        server_name: str
    ) -> IntegrationTestResult:
        """Run full integration test suite"""
```

### 3.3 Discovery & Installation

```python
class DiscoveryTools:
    def search_mcp_registries(
        query: str,
        registry: Optional[str] = None
    ) -> List[MCPServerInfo]:
        """Search available MCP server registries"""

    def get_server_details(
        server_id: str
    ) -> ServerDetails:
        """Get detailed server information"""

    def install_server(
        server_id: str,
        config: ServerConfig
    ) -> InstallationResult:
        """Install and configure new MCP server"""

    def check_dependencies(
        server_id: str
    ) -> DependencyCheck:
        """Check required dependencies"""
```

### 3.4 Configuration Management

```python
class ConfigTools:
    def read_server_config(
        server_name: str
    ) -> ServerConfig:
        """Read server configuration"""

    def update_server_config(
        server_name: str,
        updates: Dict[str, Any],
        validate: bool = True
    ) -> ConfigUpdateResult:
        """Update server configuration"""

    def validate_config(
        config: Dict[str, Any]
    ) -> ValidationResult:
        """Validate configuration structure"""

    def backup_config(
        server_name: Optional[str] = None
    ) -> BackupResult:
        """Backup configurations"""

    def restore_config(
        backup_id: str
    ) -> RestoreResult:
        """Restore from backup"""
```

### 3.5 Log Analysis

```python
class LogTools:
    def read_main_logs(
        lines: int = 100,
        filter: Optional[str] = None
    ) -> LogEntries:
        """Read mcpproxy main logs"""

    def read_server_logs(
        server_name: str,
        lines: int = 100,
        filter: Optional[str] = None
    ) -> LogEntries:
        """Read server-specific logs"""

    def analyze_error_patterns(
        logs: LogEntries
    ) -> ErrorPatternAnalysis:
        """AI-powered error pattern analysis"""

    def tail_logs_realtime(
        server_name: Optional[str] = None,
        callback: Callable[[str], None]
    ) -> None:
        """Real-time log monitoring"""
```

### 3.6 Documentation & Knowledge

```python
class DocumentationTools:
    def search_mcp_docs(
        query: str,
        server_name: Optional[str] = None
    ) -> List[DocSection]:
        """Search MCP documentation"""

    def get_tool_documentation(
        server_name: str,
        tool_name: str
    ) -> ToolDocumentation:
        """Get tool-specific documentation"""

    def fetch_external_docs(
        url: str
    ) -> DocumentContent:
        """Fetch and parse external documentation"""

    def extract_code_examples(
        documentation: str
    ) -> List[CodeExample]:
        """Extract code examples from docs"""
```

### 3.7 Startup & Service Management

```python
class StartupTools:
    def read_startup_script(
        server_name: str
    ) -> StartupScript:
        """Read server startup configuration"""

    def update_startup_script(
        server_name: str,
        script_updates: Dict[str, Any]
    ) -> UpdateResult:
        """Modify startup script"""

    def manage_docker_services(
        server_name: str,
        action: Literal["start", "stop", "restart", "status"]
    ) -> ServiceResult:
        """Manage Docker services"""

    def check_system_dependencies(
        requirements: List[str]
    ) -> DependencyStatus:
        """Check system-level dependencies"""

    def install_dependencies(
        packages: List[str],
        manager: Literal["npm", "pip", "apt", "brew"]
    ) -> InstallResult:
        """Install required dependencies"""
```

## 4. MCPProxy Integration Requirements

### 4.1 New REST API Endpoints (Go Implementation)

```go
// Add to internal/server/server.go

// Agent API endpoints
func (s *Server) setupAgentAPI() {
    s.router.HandleFunc("/api/v1/agent/servers", s.handleListServers)
    s.router.HandleFunc("/api/v1/agent/servers/{name}", s.handleServerDetails)
    s.router.HandleFunc("/api/v1/agent/servers/{name}/logs", s.handleServerLogs)
    s.router.HandleFunc("/api/v1/agent/servers/{name}/test", s.handleTestServer)
    s.router.HandleFunc("/api/v1/agent/servers/{name}/config", s.handleServerConfig)
    s.router.HandleFunc("/api/v1/agent/install", s.handleInstallServer)
    s.router.HandleFunc("/api/v1/agent/logs/main", s.handleMainLogs)
    s.router.HandleFunc("/api/v1/agent/registries/search", s.handleSearchRegistries)
    s.router.HandleFunc("/api/v1/agent/startup/{name}", s.handleStartupScript)
}
```

### 4.2 Agent Configuration

```json
{
  "agent": {
    "enabled": true,
    "port": 8081,
    "auth_token": "secure-token-here",
    "llm": {
      "provider": "anthropic",
      "model": "claude-3-5-sonnet-20241022",
      "api_key_env": "ANTHROPIC_API_KEY",
      "fallback_models": ["claude-3-haiku-20240307"]
    },
    "memory": {
      "type": "sqlite",
      "path": "~/.mcpproxy/agent_memory.db",
      "max_conversations": 100
    },
    "capabilities": {
      "auto_fix": true,
      "auto_install": false,
      "require_approval": true
    }
  }
}
```

## 5. Implementation Task List

### Phase 1: Foundation (Week 1-2)

- [ ] **Task 1.1**: Set up Python project structure
  - Create virtual environment
  - Install LangGraph, PydanticAI, httpx, rich
  - Set up project layout

- [ ] **Task 1.2**: Design PydanticAI tool schemas
  - Define all tool models with Pydantic
  - Create validation logic
  - Document tool interfaces

- [ ] **Task 1.3**: Implement MCPProxy REST API (Go)
  - Add new API endpoints
  - Implement handlers for logs, configs, servers
  - Add authentication middleware

- [ ] **Task 1.4**: Create HTTP client layer (Python)
  - Build typed HTTP client for mcpproxy API
  - Implement error handling and retries
  - Add response validation

### Phase 2: Core Agent Development (Week 3-4)

- [ ] **Task 2.1**: Implement LangGraph state machine
  - Define states and transitions
  - Create state persistence layer
  - Implement checkpointing

- [ ] **Task 2.2**: Build Diagnostic Agent
  - Log analysis tools
  - Error pattern detection
  - Fix suggestion system

- [ ] **Task 2.3**: Build Config Manager Agent
  - Config read/write tools
  - Validation logic
  - Backup/restore functionality

- [ ] **Task 2.4**: Build Discovery Agent
  - Registry search integration
  - Server installation logic
  - Dependency checking

### Phase 3: Advanced Features (Week 5-6)

- [ ] **Task 3.1**: Testing Agent implementation
  - Tool testing framework
  - Health check system
  - Integration test runner

- [ ] **Task 3.2**: Documentation Agent
  - Doc search and retrieval
  - Information extraction
  - Context building for LLM

- [ ] **Task 3.3**: Startup Manager
  - Startup script manipulation
  - Docker/service management
  - Dependency installation

- [ ] **Task 3.4**: Memory & Learning System
  - Conversation history
  - Problem-solution database
  - Pattern learning

### Phase 4: Integration & Testing (Week 7-8)

- [ ] **Task 4.1**: End-to-end integration
  - Connect all agents
  - Test full workflows
  - Performance optimization

- [ ] **Task 4.2**: CLI Interface
  - Interactive CLI using Rich
  - Command-line tools
  - Output formatting

- [ ] **Task 4.3**: Web UI (Optional)
  - Streamlit/Gradio interface
  - Real-time monitoring
  - Visual workflow display

- [ ] **Task 4.4**: Testing & Documentation
  - Unit tests (pytest)
  - Integration tests
  - User documentation
  - API documentation

### Phase 5: Deployment & Monitoring (Week 9-10)

- [ ] **Task 5.1**: Deployment packaging
  - Docker container
  - Installation scripts
  - Configuration templates

- [ ] **Task 5.2**: Monitoring & Observability
  - Metrics collection
  - Performance monitoring
  - Error tracking

- [ ] **Task 5.3**: Security hardening
  - Authentication/authorization
  - Input validation
  - Rate limiting

- [ ] **Task 5.4**: Production readiness
  - Load testing
  - Documentation review
  - Release preparation

## 6. Technology Stack

### Python Components
```
langgraph==0.2.0          # Agent orchestration
pydantic==2.5.0           # Data validation
pydantic-ai==0.0.13       # AI agent framework
httpx==0.25.0             # HTTP client
rich==13.7.0              # CLI formatting
typer==0.9.0              # CLI framework
sqlite3                   # Memory persistence
pytest==7.4.0             # Testing
```

### Go Components (mcpproxy extensions)
```
net/http                  # REST API
gorilla/mux              # Routing
```

## 7. Example Agent Workflow

### Scenario: Debug a Failing MCP Server

```python
# User: "The github-server keeps failing, can you diagnose it?"

1. ANALYZING State
   - Agent reads server status → "Error: Connection timeout"
   - Agent reads server logs → Identifies OAuth token expiration

2. DIAGNOSING State
   - Agent analyzes error patterns
   - Agent checks server configuration
   - Agent identifies root cause: "OAuth token expired"

3. SUGGESTING State
   - Agent suggests: "Re-authenticate OAuth"
   - Agent provides command: "mcpproxy auth login --server=github-server"

4. (User approves) FIXING State
   - Agent executes re-authentication
   - Agent verifies connection restored

5. MONITORING State
   - Agent monitors for 5 minutes
   - Agent confirms stable operation

6. REPORTING State
   - Agent summarizes: "Fixed OAuth authentication for github-server"
   - Agent updates knowledge base with solution
```

## 8. Security Considerations

1. **Authentication**: All agent API calls require token authentication
2. **Authorization**: Configurable capabilities (auto-fix, auto-install, etc.)
3. **Approval Flow**: Critical operations require user approval
4. **Audit Log**: All agent actions logged for review
5. **Sandboxing**: Server installations run in isolated environments
6. **Rate Limiting**: Prevent excessive API calls
7. **Input Validation**: All inputs validated before processing

## 9. Success Metrics

- **Diagnostic Accuracy**: >90% correct root cause identification
- **Fix Success Rate**: >80% successful automatic fixes
- **Response Time**: <30s for common diagnostics
- **User Approval Rate**: >70% of suggested fixes approved
- **System Stability**: No agent-caused mcpproxy crashes

## 10. Future Enhancements

- Multi-modal capabilities (analyze screenshots, diagrams)
- Predictive maintenance (anticipate issues before they occur)
- Automated testing suite generation
- Performance optimization recommendations
- Natural language query interface
- Integration with monitoring systems (Prometheus, Grafana)
- Self-learning from successful fixes
- Collaborative multi-agent problem solving

---

## Appendix A: Quick Start Guide

```bash
# 1. Install Python agent
cd agent/
python -m venv venv
source venv/bin/activate
pip install -r requirements.txt

# 2. Configure
cp config.example.json config.json
# Edit config.json with your settings

# 3. Start mcpproxy with agent API
./mcpproxy serve --enable-agent-api

# 4. Start agent
python -m mcp_agent --config config.json

# 5. Interact
python -m mcp_agent cli "Diagnose all servers"
```

## Appendix B: Development Commands

```bash
# Run tests
pytest tests/ -v

# Type checking
mypy mcp_agent/

# Linting
ruff check mcp_agent/

# Format code
black mcp_agent/

# Run agent with debug logging
python -m mcp_agent --log-level=debug
```

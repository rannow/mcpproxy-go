# MCP Agent Migration Plan

## Overview
Replace the Go-based LLM and diagnostic agents in the tray with the new Python agent API.

## Current Architecture

### Go Implementation (To Be Removed)
- `internal/tray/agent_llm.go` (1063 lines) - OpenAI/Anthropic-based LLM agent
- `internal/tray/diagnostic_agent.go` (622 lines) - Server diagnostic agent
- `internal/tray/llm_client.go` (26KB) - LLM API client

### Python Implementation (New)
- `agent/mcp_agent/tools/diagnostic.py` (400+ lines) - Diagnostic tools
- `agent/mcp_agent/tools/config.py` (200+ lines) - Configuration management
- `agent/mcp_agent/tools/discovery.py` - Server discovery
- `agent/mcp_agent/tools/testing.py` - Server testing
- `agent/mcp_agent/tools/logs.py` - Log analysis
- `agent/mcp_agent/tools/docs.py` - Documentation tools
- `agent/mcp_agent/tools/startup.py` - Startup management
- `agent/mcp_agent/graph/agent_graph.py` (300+ lines) - LangGraph orchestration
- `internal/server/agent_api.go` (450+ lines) - REST API endpoints

## Migration Strategy

### Phase 1: Testing Infrastructure ✅ COMPLETED
- [x] Set up pytest infrastructure
- [x] Create 32 unit tests for DiagnosticTools
- [x] Fix dependency issues (typing-inspection, langgraph)
- [x] All tests passing (100% success rate)

### Phase 2: Complete Test Coverage (IN PROGRESS)
- [ ] Unit tests for ConfigTools
- [ ] Unit tests for DiscoveryTools
- [ ] Unit tests for TestingTools
- [ ] Unit tests for LogTools
- [ ] Unit tests for DocumentationTools
- [ ] Unit tests for StartupTools
- [ ] Run coverage report (target: 80%+)

### Phase 3: Integration Testing
- [ ] Integration tests for LangGraph state machine
- [ ] Integration tests for agent workflow
- [ ] E2E test scenarios (diagnose → suggest → apply fixes)

### Phase 4: Python Agent API Integration
- [ ] Create Go HTTP client for Python agent API
- [ ] Replace LLMAgent in tray with Python agent client
- [ ] Replace DiagnosticAgent with Python agent client
- [ ] Update chat system to use new agent
- [ ] Test tray integration

### Phase 5: Code Cleanup
- [ ] Remove `internal/tray/agent_llm.go`
- [ ] Remove `internal/tray/diagnostic_agent.go`
- [ ] Remove `internal/tray/llm_client.go`
- [ ] Update imports and dependencies
- [ ] Clean up unused code

### Phase 6: Documentation and Verification
- [ ] Update CLAUDE.md with new architecture
- [ ] Update agent documentation
- [ ] Integration testing
- [ ] Deployment verification

## API Endpoint Mapping

### Current Go Endpoints → Python API
| Go Function | Python Endpoint | Status |
|-------------|----------------|--------|
| `read_config` tool | `/api/v1/agent/servers/{name}/config` | ✅ Available |
| `write_config` tool | `/api/v1/agent/servers/{name}/config` (PATCH) | ✅ Available |
| `read_log` tool | `/api/v1/agent/servers/{name}/logs` | ✅ Available |
| `read_github` tool | Python `DocumentationTools.get_server_readme()` | ✅ Available |
| `restart_server` tool | Python `ConfigTools.update_server_config()` | ✅ Available |
| `call_tool` tool | Direct MCP proxy calls | ✅ Available |
| `get_server_status` tool | `/api/v1/agent/servers/{name}` | ✅ Available |
| `test_server_tools` tool | Python `TestingTools.run_test_suite()` | ✅ Available |
| `list_all_servers` tool | `/api/v1/agent/servers` | ✅ Available |
| `list_all_tools` tool | Python `TestingTools` + server enumeration | ✅ Available |

## Benefits of Migration

### Technical Improvements
1. **Better Architecture**: Modular tool-based design with clear separation of concerns
2. **Type Safety**: Pydantic models for all data structures
3. **Advanced AI**: LangGraph state machine for complex workflows
4. **Better Testing**: Comprehensive pytest infrastructure with 32+ unit tests
5. **Maintainability**: Python is easier to extend and maintain than Go for agent logic

### Functional Improvements
1. **More Capabilities**: 7 tool modules vs 2 Go agents
2. **Better Diagnostics**: Comprehensive log analysis and error detection
3. **Configuration Management**: Advanced config validation and updates
4. **Documentation Integration**: Direct GitHub README fetching and analysis
5. **Testing Tools**: Automated test suite generation and execution

### Operational Improvements
1. **REST API**: Clean HTTP API for easy integration
2. **Language Flexibility**: Use any LLM provider (OpenAI, Anthropic, local models)
3. **Memory Management**: Persistent memory across sessions
4. **Observable**: Better logging and debugging

## Risk Mitigation

### Rollback Strategy
1. Keep Go agent code temporarily in separate branch
2. Feature flag to switch between Go and Python agents
3. Gradual migration with A/B testing

### Testing Strategy
1. Unit tests for all Python tools (target: 80%+ coverage)
2. Integration tests for API endpoints
3. E2E tests for complete workflows
4. Manual testing in tray UI

### Performance Considerations
1. Python agent runs as HTTP service (already running)
2. REST API adds minimal latency (~10-50ms)
3. Async operations for long-running tasks
4. Caching for frequently accessed data

## Timeline

- **Week 1**: Complete test coverage and integration tests ⏳
- **Week 2**: Integrate Python agent into tray
- **Week 3**: Remove Go agent code and cleanup
- **Week 4**: Documentation and deployment

## Success Criteria

- [ ] 80%+ test coverage for Python agent
- [ ] All unit tests passing
- [ ] Integration tests passing
- [ ] E2E tests passing
- [ ] Tray UI working with Python agent
- [ ] No Go agent code remaining
- [ ] Documentation updated
- [ ] Performance metrics acceptable (<100ms API latency)

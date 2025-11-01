# MCP Agent Test Suite

Comprehensive test suite for the MCP Agent application using pytest.

## Test Structure

```
tests/
├── __init__.py           # Test package initialization
├── conftest.py           # Shared fixtures and configuration
├── unit/                 # Unit tests for individual components
│   ├── __init__.py
│   └── test_diagnostic_tools.py
├── integration/          # Integration tests for workflows
│   ├── __init__.py
│   └── test_agent_workflow.py
└── e2e/                  # End-to-end test scenarios
    ├── __init__.py
    └── test_diagnostic_scenarios.py
```

## Running Tests

### All Tests
```bash
pytest
```

### By Test Type
```bash
# Unit tests only
pytest -m unit

# Integration tests only
pytest -m integration

# End-to-end tests only
pytest -m e2e
```

### By Component
```bash
# Diagnostic tools tests
pytest -m diagnostic

# LangGraph workflow tests
pytest -m graph

# Configuration tests
pytest -m config
```

### With Coverage
```bash
# Run with coverage report
pytest --cov=mcp_agent

# Generate HTML coverage report
pytest --cov=mcp_agent --cov-report=html

# View coverage in browser
open htmlcov/index.html
```

### Parallel Execution
```bash
# Install pytest-xdist
pip install pytest-xdist

# Run tests in parallel
pytest -n auto
```

### Specific Test File
```bash
pytest tests/unit/test_diagnostic_tools.py -v
```

### Specific Test Class
```bash
pytest tests/unit/test_diagnostic_tools.py::TestDiagnosticToolsLogAnalysis -v
```

### Specific Test Function
```bash
pytest tests/unit/test_diagnostic_tools.py::TestDiagnosticToolsLogAnalysis::test_analyze_server_logs_basic -v
```

## Test Markers

Tests are organized using pytest markers:

- `@pytest.mark.unit`: Unit tests for individual components
- `@pytest.mark.integration`: Integration tests for workflows
- `@pytest.mark.e2e`: End-to-end test scenarios
- `@pytest.mark.slow`: Long-running tests
- `@pytest.mark.requires_api`: Tests requiring mcpproxy API
- `@pytest.mark.requires_llm`: Tests requiring LLM API access

Component-specific markers:
- `@pytest.mark.diagnostic`: Diagnostic tools tests
- `@pytest.mark.config`: Configuration management tests
- `@pytest.mark.discovery`: Server discovery tests
- `@pytest.mark.testing`: Testing tools tests
- `@pytest.mark.logs`: Log analysis tests
- `@pytest.mark.docs`: Documentation tools tests
- `@pytest.mark.startup`: Startup script tests
- `@pytest.mark.graph`: LangGraph state machine tests

### Filtering by Markers
```bash
# Run only unit tests
pytest -m unit

# Run diagnostic tests
pytest -m diagnostic

# Run tests that don't require API
pytest -m "not requires_api"

# Combine markers (unit AND diagnostic)
pytest -m "unit and diagnostic"

# Combine markers (unit OR integration)
pytest -m "unit or integration"
```

## Test Fixtures

### Available Fixtures

#### Session-Scoped (Shared Across All Tests)
- `event_loop`: Async event loop for tests
- `base_url`: Base URL for mcpproxy API
- `api_token`: Mock API token

#### Function-Scoped (Fresh Instance Per Test)
- `mock_httpx_client`: Mock httpx.AsyncClient
- `mock_mcpproxy_client`: Mock MCPProxyClient
- `configured_mock_client`: Pre-configured client with responses
- `failed_server_mock_client`: Client configured for failure scenarios

#### Test Data
- `sample_server_config`: Sample HTTP server configuration
- `sample_stdio_server_config`: Sample stdio server configuration
- `sample_log_entries`: Sample log entries
- `sample_server_status`: Healthy server status
- `sample_failed_server_status`: Failed server status
- `sample_tool_list`: Sample tool list
- `sample_oauth_config`: OAuth configuration

#### HTTP Responses
- `mock_server_logs_response`: Mock logs endpoint response
- `mock_server_status_response`: Mock status endpoint response
- `mock_failed_server_status_response`: Mock failed status response
- `mock_tools_list_response`: Mock tools list response
- `mock_error_response`: Mock 500 error response
- `mock_auth_error_response`: Mock 401 error response

#### LangGraph State
- `initial_agent_state`: Initial agent state
- `diagnostic_state_with_results`: State after diagnostics

### Using Fixtures in Tests
```python
@pytest.mark.unit
def test_example(configured_mock_client, sample_log_entries):
    """Example test using fixtures."""
    tools = DiagnosticTools(configured_mock_client)
    # Test implementation
```

## Writing Tests

### Unit Test Template
```python
import pytest
from mcp_agent.tools.diagnostic import DiagnosticTools

@pytest.mark.unit
@pytest.mark.diagnostic
class TestMyFeature:
    """Test my feature."""

    @pytest.mark.asyncio
    async def test_basic_functionality(self, configured_mock_client):
        """Test basic functionality."""
        tools = DiagnosticTools(configured_mock_client)

        result = await tools.my_method("test-input")

        assert result is not None
        assert result.some_field == expected_value
```

### Integration Test Template
```python
import pytest
from mcp_agent.graph.agent_graph import MCPAgentGraph, AgentInput

@pytest.mark.integration
@pytest.mark.graph
class TestMyWorkflow:
    """Test my workflow."""

    @pytest.fixture
    def tools_registry(self, configured_mock_client):
        return {"diagnostic": DiagnosticTools(configured_mock_client)}

    @pytest.mark.asyncio
    async def test_workflow_execution(self, tools_registry):
        """Test workflow execution."""
        agent = MCPAgentGraph(tools_registry)

        result = await agent.run(AgentInput(
            request="Test request",
            server_name="test-server",
        ))

        assert result is not None
```

### E2E Test Template
```python
import pytest

@pytest.mark.e2e
@pytest.mark.diagnostic
class TestMyScenario:
    """Test complete user scenario."""

    @pytest.mark.asyncio
    async def test_complete_workflow(self, tools_registry):
        """Test complete workflow from input to output."""
        # Setup
        # Execute
        # Verify
```

## Coverage Goals

- **Overall**: 80%+ coverage
- **Unit tests**: 90%+ coverage
- **Integration tests**: 70%+ coverage
- **E2E tests**: Focus on critical paths

### Checking Coverage
```bash
# Run with coverage
pytest --cov=mcp_agent

# Generate detailed report
pytest --cov=mcp_agent --cov-report=term-missing

# Find uncovered lines
pytest --cov=mcp_agent --cov-report=html
open htmlcov/index.html
```

## Continuous Integration

### GitHub Actions Example
```yaml
name: Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest

    steps:
    - uses: actions/checkout@v2

    - name: Set up Python
      uses: actions/setup-python@v2
      with:
        python-version: '3.11'

    - name: Install dependencies
      run: |
        pip install -r requirements.txt

    - name: Run tests
      run: |
        pytest --cov=mcp_agent --cov-report=xml

    - name: Upload coverage
      uses: codecov/codecov-action@v2
```

## Troubleshooting

### Common Issues

#### Tests Hanging
```bash
# Use timeout to prevent hanging tests
pytest --timeout=30
```

#### Import Errors
```bash
# Install package in development mode
pip install -e .
```

#### Async Test Issues
```bash
# Ensure pytest-asyncio is installed
pip install pytest-asyncio

# Check pytest.ini has asyncio_mode = auto
```

#### Coverage Not Generated
```bash
# Ensure pytest-cov is installed
pip install pytest-cov

# Check that paths are correct
pytest --cov=mcp_agent --cov-report=term
```

## Best Practices

1. **Test Isolation**: Each test should be independent
2. **Clear Naming**: Use descriptive test names
3. **Arrange-Act-Assert**: Structure tests clearly
4. **Mock External Dependencies**: Don't make real API calls in unit tests
5. **Use Fixtures**: Reuse common setup code
6. **Test Edge Cases**: Include error conditions and boundary cases
7. **Keep Tests Fast**: Unit tests should run in milliseconds
8. **Document Intent**: Add docstrings explaining what's being tested
9. **One Assertion Per Test**: Focus each test on one behavior
10. **Use Markers**: Tag tests appropriately for filtering

## Additional Resources

- [pytest documentation](https://docs.pytest.org/)
- [pytest-asyncio](https://pytest-asyncio.readthedocs.io/)
- [pytest-cov](https://pytest-cov.readthedocs.io/)
- [Testing best practices](https://docs.python-guide.org/writing/tests/)

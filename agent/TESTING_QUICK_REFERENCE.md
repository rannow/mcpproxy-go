# MCP Agent Testing Quick Reference Card

## 🚀 Quick Commands

```bash
# Run all tests
pytest

# Run with coverage
pytest --cov=mcp_agent --cov-report=html && open htmlcov/index.html

# Run unit tests only
pytest -m unit -v

# Run specific test file
pytest tests/unit/test_diagnostic_tools.py -v

# Run specific test class
pytest tests/unit/test_diagnostic_tools.py::TestDiagnosticToolsLogAnalysis -v

# Run specific test
pytest tests/unit/test_diagnostic_tools.py::TestMCPProxyClient::test_client_initialization -v
```

## 🏷️ Test Markers

```bash
# By test type
pytest -m unit              # Unit tests
pytest -m integration       # Integration tests
pytest -m e2e               # End-to-end tests

# By component
pytest -m diagnostic        # Diagnostic tools
pytest -m config            # Config tools
pytest -m graph             # LangGraph workflows

# By requirements
pytest -m "not slow"        # Skip slow tests
pytest -m "not requires_api" # Skip API tests
```

## 📁 Test Structure

```
tests/
├── conftest.py                  # Shared fixtures
├── unit/                        # Unit tests (32 tests ✅)
│   └── test_diagnostic_tools.py
├── integration/                 # Integration tests (12 tests)
│   └── test_agent_workflow.py
└── e2e/                         # E2E tests (10 scenarios)
    └── test_diagnostic_scenarios.py
```

## 🔧 Common Fixtures

```python
# Mock clients
configured_mock_client           # Pre-configured MCPProxyClient
failed_server_mock_client        # Client for failure scenarios
mock_httpx_client               # Raw httpx mock

# Test data
sample_server_config            # HTTP server config
sample_stdio_server_config      # stdio server config
sample_log_entries              # Log entries
sample_server_status            # Server status

# LangGraph state
initial_agent_state             # Initial state
diagnostic_state_with_results   # After diagnostics
```

## ✍️ Writing Tests Template

```python
import pytest
from mcp_agent.tools.diagnostic import DiagnosticTools

@pytest.mark.unit
@pytest.mark.diagnostic
class TestMyFeature:
    """Test description."""

    @pytest.mark.asyncio
    async def test_basic_functionality(self, configured_mock_client):
        """Test what this does."""
        # Arrange
        tools = DiagnosticTools(configured_mock_client)

        # Act
        result = await tools.my_method("input")

        # Assert
        assert result is not None
        assert result.field == expected_value
```

## 📊 Coverage

```bash
# Generate coverage report
make test-cov-html

# Check coverage threshold (80%)
pytest --cov=mcp_agent --cov-fail-under=80
```

## 🛠️ Makefile Shortcuts

```bash
make test               # Run all tests
make test-unit          # Unit tests
make test-integration   # Integration tests
make test-e2e           # E2E tests
make test-cov           # With coverage
make test-cov-html      # HTML coverage report
make lint               # Run linters
make format             # Format code
make clean              # Clean generated files
```

## 🐛 Debugging

```bash
# Verbose output
pytest -vv

# Show print statements
pytest -s

# Stop on first failure
pytest -x

# Drop into debugger on failure
pytest --pdb

# Re-run only failed tests
pytest --lf
```

## 📚 Key Files

- `pytest.ini` - Configuration
- `tests/conftest.py` - Fixtures (682 lines)
- `tests/README.md` - Full documentation
- `Makefile` - Convenient commands
- `.github/workflows/test.yml.example` - CI/CD template

## 💡 Pro Tips

1. **Use fixtures**: Don't repeat setup code
2. **One assertion per test**: Keep tests focused
3. **Descriptive names**: `test_oauth_failure_suggests_reauth`
4. **Mark appropriately**: Use `@pytest.mark.unit`, etc.
5. **Mock external calls**: Don't make real API calls
6. **Test edge cases**: Include error scenarios
7. **Keep tests fast**: Unit tests should run in ms

## 🎯 Success Criteria

- ✅ Unit tests: 90%+ coverage
- ✅ Integration tests: 70%+ coverage
- ✅ All tests pass in CI/CD
- ✅ Tests run in <10 seconds locally

## 📖 More Info

See `tests/README.md` for comprehensive documentation.

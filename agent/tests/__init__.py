"""MCP Agent Test Suite.

This package contains comprehensive tests for the MCP Agent application.

Test Organization:
- unit/: Unit tests for individual tools and components
- integration/: Integration tests for LangGraph workflows
- e2e/: End-to-end test scenarios

Test Markers:
- @pytest.mark.unit: Unit tests
- @pytest.mark.integration: Integration tests
- @pytest.mark.e2e: End-to-end tests
- @pytest.mark.slow: Long-running tests
- @pytest.mark.requires_api: Tests requiring mcpproxy API
- @pytest.mark.requires_llm: Tests requiring LLM API access

Running Tests:
- All tests: pytest
- Unit tests only: pytest -m unit
- Integration tests: pytest -m integration
- E2E tests: pytest -m e2e
- Specific tool: pytest -m diagnostic
- With coverage: pytest --cov=mcp_agent
- Parallel: pytest -n auto (requires pytest-xdist)
"""

__version__ = "0.1.0"

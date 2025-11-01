"""End-to-end tests for MCP Agent scenarios.

E2E tests verify complete user workflows from input to output.
These tests may require running mcpproxy API and should be marked
with @pytest.mark.requires_api.

Test Coverage:
- Complete diagnostic workflows
- Server installation and configuration
- Error recovery scenarios
- Multi-step agent interactions
- Real API integration (when available)
"""

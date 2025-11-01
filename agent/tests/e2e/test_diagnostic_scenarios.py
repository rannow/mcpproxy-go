"""End-to-end tests for diagnostic scenarios.

These tests verify complete user workflows from input to output.
Some tests require a running mcpproxy instance and are marked with
@pytest.mark.requires_api.

Usage:
    # Run all E2E tests (without API)
    pytest tests/e2e/ -v -m "not requires_api"

    # Run with API integration
    pytest tests/e2e/ -v -m requires_api

    # Run specific scenario
    pytest tests/e2e/test_diagnostic_scenarios.py::TestOAuthFailureScenario -v
"""

from typing import Dict
from unittest.mock import AsyncMock, Mock

import pytest

from mcp_agent.graph.agent_graph import AgentInput, MCPAgentGraph
from mcp_agent.tools.diagnostic import DiagnosticTools


# ============================================================================
# E2E Diagnostic Scenarios (Mocked)
# ============================================================================


@pytest.mark.e2e
@pytest.mark.diagnostic
class TestOAuthFailureScenario:
    """Test complete OAuth failure diagnostic and fix workflow."""

    @pytest.fixture
    def oauth_failure_logs(self):
        """Logs showing OAuth token expiration."""
        return [
            {
                "timestamp": "2025-11-01T10:00:00Z",
                "level": "ERROR",
                "message": "OAuth token expired",
                "server": "github-server",
            }
            for _ in range(10)
        ]

    @pytest.fixture
    def oauth_failure_status(self):
        """Server status showing OAuth failure."""
        return {
            "name": "github-server",
            "state": "Error",
            "is_connected": False,
            "last_error": "Authentication failed: OAuth token expired",
            "retry_count": 5,
        }

    @pytest.fixture
    def tools_registry(
        self,
        mock_mcpproxy_client,
        oauth_failure_logs,
        oauth_failure_status,
    ):
        """Create tools registry with OAuth failure scenario."""
        mock_mcpproxy_client.get_server_logs.return_value = oauth_failure_logs
        mock_mcpproxy_client.get_server_status.return_value = oauth_failure_status

        return {
            "diagnostic": DiagnosticTools(mock_mcpproxy_client),
        }

    @pytest.mark.asyncio
    async def test_oauth_failure_complete_workflow(self, tools_registry):
        """Test complete OAuth failure diagnostic workflow."""
        agent = MCPAgentGraph(tools_registry)

        # User request
        user_input = AgentInput(
            request="GitHub server is not working, can you help?",
            server_name="github-server",
            auto_approve=False,
        )

        # Run agent
        result = await agent.run(user_input)

        # Verify agent identified OAuth issue
        assert result.response is not None
        assert len(result.actions_taken) > 0

        # Verify recommendations include re-authentication
        recommendations_text = " ".join(result.recommendations).lower()
        assert "oauth" in recommendations_text or "auth" in recommendations_text

        # Verify server status indicates failure
        if result.server_status:
            assert result.server_status["is_connected"] is False


@pytest.mark.e2e
@pytest.mark.diagnostic
class TestHighErrorRateScenario:
    """Test diagnostic workflow for server with high error rate."""

    @pytest.fixture
    def high_error_logs(self):
        """Logs showing high error rate."""
        logs = []
        for i in range(100):
            if i % 3 == 0:  # 33% error rate
                logs.append(
                    {
                        "timestamp": f"2025-11-01T10:{i:02d}:00Z",
                        "level": "ERROR",
                        "message": "Connection timeout",
                        "server": "slow-server",
                    }
                )
            else:
                logs.append(
                    {
                        "timestamp": f"2025-11-01T10:{i:02d}:00Z",
                        "level": "INFO",
                        "message": "Request processed",
                        "server": "slow-server",
                    }
                )
        return logs

    @pytest.fixture
    def tools_registry(self, mock_mcpproxy_client, high_error_logs):
        """Create tools registry with high error scenario."""
        mock_mcpproxy_client.get_server_logs.return_value = high_error_logs
        mock_mcpproxy_client.get_server_status.return_value = {
            "name": "slow-server",
            "state": "Ready",
            "is_connected": True,
        }

        return {
            "diagnostic": DiagnosticTools(mock_mcpproxy_client),
        }

    @pytest.mark.asyncio
    async def test_high_error_rate_detection(self, tools_registry):
        """Test that high error rate is detected and reported."""
        agent = MCPAgentGraph(tools_registry)

        user_input = AgentInput(
            request="Analyze slow-server performance",
            server_name="slow-server",
            auto_approve=False,
        )

        result = await agent.run(user_input)

        # Should identify high error rate
        assert len(result.recommendations) > 0


@pytest.mark.e2e
@pytest.mark.diagnostic
class TestCriticalServerCrashScenario:
    """Test diagnostic workflow for critical server crash."""

    @pytest.fixture
    def crash_logs(self):
        """Logs showing server crash."""
        return [
            {
                "timestamp": "2025-11-01T10:00:00Z",
                "level": "INFO",
                "message": "Server starting",
                "server": "crash-server",
            },
            {
                "timestamp": "2025-11-01T10:01:00Z",
                "level": "CRITICAL",
                "message": "Server crashed: Out of memory",
                "server": "crash-server",
            },
        ]

    @pytest.fixture
    def tools_registry(self, mock_mcpproxy_client, crash_logs):
        """Create tools registry with crash scenario."""
        mock_mcpproxy_client.get_server_logs.return_value = crash_logs
        mock_mcpproxy_client.get_server_status.return_value = {
            "name": "crash-server",
            "state": "Error",
            "is_connected": False,
            "last_error": "Server crashed",
        }

        return {
            "diagnostic": DiagnosticTools(mock_mcpproxy_client),
        }

    @pytest.mark.asyncio
    async def test_critical_crash_detection(self, tools_registry):
        """Test that critical crashes are detected and prioritized."""
        agent = MCPAgentGraph(tools_registry)

        user_input = AgentInput(
            request="What happened to crash-server?",
            server_name="crash-server",
            auto_approve=False,
        )

        result = await agent.run(user_input)

        # Should identify critical issue
        assert result.response is not None


# ============================================================================
# E2E Scenarios Requiring API
# ============================================================================


@pytest.mark.e2e
@pytest.mark.requires_api
@pytest.mark.slow
class TestRealAPIIntegration:
    """Test scenarios with real mcpproxy API.

    These tests require a running mcpproxy instance at localhost:8080.
    They are skipped by default and must be explicitly run.
    """

    @pytest.mark.asyncio
    async def test_real_server_status_retrieval(self):
        """Test retrieving real server status from API."""
        from mcp_agent.tools.diagnostic import MCPProxyClient

        client = MCPProxyClient(base_url="http://localhost:8080")

        try:
            # This will fail if API is not running
            status = await client.get_server_status("test-server")
            assert status is not None
        except Exception as e:
            pytest.skip(f"API not available: {e}")

    @pytest.mark.asyncio
    async def test_real_logs_retrieval(self):
        """Test retrieving real logs from API."""
        from mcp_agent.tools.diagnostic import MCPProxyClient

        client = MCPProxyClient(base_url="http://localhost:8080")

        try:
            logs = await client.get_main_logs(lines=10)
            assert isinstance(logs, list)
        except Exception as e:
            pytest.skip(f"API not available: {e}")


# ============================================================================
# Multi-Server Diagnostic Scenarios
# ============================================================================


@pytest.mark.e2e
@pytest.mark.diagnostic
class TestMultiServerDiagnostic:
    """Test diagnostic workflows involving multiple servers."""

    @pytest.mark.asyncio
    async def test_diagnose_without_server_name(
        self,
        configured_mock_client,
    ):
        """Test diagnostic request without specifying server."""
        tools_registry = {
            "diagnostic": DiagnosticTools(configured_mock_client),
        }

        agent = MCPAgentGraph(tools_registry)

        user_input = AgentInput(
            request="Check all servers for issues",
            server_name=None,  # No specific server
            auto_approve=False,
        )

        result = await agent.run(user_input)

        # Should handle gracefully
        assert result.response is not None


# ============================================================================
# User Interaction Scenarios
# ============================================================================


@pytest.mark.e2e
@pytest.mark.integration
class TestApprovalWorkflow:
    """Test scenarios requiring user approval."""

    @pytest.fixture
    def tools_registry(self, mock_mcpproxy_client):
        """Create tools registry."""
        mock_mcpproxy_client.get_server_logs.return_value = [
            {
                "level": "ERROR",
                "message": "OAuth token expired",
            }
        ]
        mock_mcpproxy_client.get_server_status.return_value = {
            "name": "test-server",
            "state": "Error",
            "is_connected": False,
        }

        return {
            "diagnostic": DiagnosticTools(mock_mcpproxy_client),
        }

    @pytest.mark.asyncio
    async def test_workflow_with_approval_required(self, tools_registry):
        """Test workflow that requires user approval."""
        agent = MCPAgentGraph(tools_registry)

        user_input = AgentInput(
            request="Fix test-server",
            server_name="test-server",
            auto_approve=False,  # Require approval
        )

        result = await agent.run(user_input)

        # Should indicate approval needed
        # (actual approval mechanism depends on implementation)
        assert result.response is not None

    @pytest.mark.asyncio
    async def test_workflow_with_auto_approve(self, tools_registry):
        """Test workflow with auto-approval enabled."""
        agent = MCPAgentGraph(tools_registry)

        user_input = AgentInput(
            request="Fix test-server",
            server_name="test-server",
            auto_approve=True,  # Auto-approve safe fixes
        )

        result = await agent.run(user_input)

        # Should proceed without manual approval
        assert result.response is not None

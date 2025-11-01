"""Unit tests for DiagnosticTools.

This module provides comprehensive unit tests for the DiagnosticTools class,
demonstrating best practices for testing async code, mocking HTTP clients,
and verifying tool behavior.

Test Organization:
- TestMCPProxyClient: Tests for the HTTP client
- TestDiagnosticToolsLogAnalysis: Tests for log analysis functionality
- TestDiagnosticToolsConnectionDiagnostics: Tests for connection diagnostics
- TestDiagnosticToolsToolFailureAnalysis: Tests for tool failure analysis
- TestDiagnosticToolsFixSuggestions: Tests for fix suggestion generation

Usage:
    # Run all diagnostic tests
    pytest tests/unit/test_diagnostic_tools.py -v

    # Run specific test class
    pytest tests/unit/test_diagnostic_tools.py::TestDiagnosticToolsLogAnalysis -v

    # Run with coverage
    pytest tests/unit/test_diagnostic_tools.py --cov=mcp_agent.tools.diagnostic
"""

from datetime import datetime
from typing import Any, Dict, List
from unittest.mock import AsyncMock, Mock, patch

import httpx
import pytest
from httpx import Response

from mcp_agent.tools.diagnostic import (
    ConnectionDiagnostic,
    DiagnosticTools,
    Fix,
    LogAnalysisResult,
    MCPProxyClient,
    SeverityLevel,
    ToolFailureAnalysis,
)


# ============================================================================
# MCPProxyClient Tests
# ============================================================================


@pytest.mark.unit
@pytest.mark.diagnostic
class TestMCPProxyClient:
    """Test MCPProxyClient HTTP client."""

    @pytest.mark.asyncio
    async def test_client_initialization(self, base_url, api_token):
        """Test client initializes with correct configuration."""
        client = MCPProxyClient(base_url=base_url, api_token=api_token)

        assert client.base_url == base_url
        assert client.headers["Authorization"] == f"Bearer {api_token}"
        assert isinstance(client.client, httpx.AsyncClient)

    @pytest.mark.asyncio
    async def test_client_initialization_without_token(self, base_url):
        """Test client initializes without API token."""
        client = MCPProxyClient(base_url=base_url)

        assert client.base_url == base_url
        assert "Authorization" not in client.headers

    @pytest.mark.asyncio
    async def test_get_server_logs_success(
        self,
        mock_httpx_client,
        sample_log_entries,
    ):
        """Test successful server logs retrieval."""
        # Configure mock response
        mock_response = Response(
            status_code=200,
            json=sample_log_entries,
            request=Mock(),
        )
        mock_httpx_client.get.return_value = mock_response

        # Create client with mocked HTTP client
        client = MCPProxyClient()
        client.client = mock_httpx_client

        # Execute
        logs = await client.get_server_logs("test-server", lines=100)

        # Verify
        assert logs == sample_log_entries
        mock_httpx_client.get.assert_called_once_with(
            "/api/v1/agent/servers/test-server/logs",
            params={"lines": 100},
        )

    @pytest.mark.asyncio
    async def test_get_server_logs_with_filter(
        self,
        mock_httpx_client,
        sample_log_entries,
    ):
        """Test server logs retrieval with filter pattern."""
        mock_response = Response(
            status_code=200,
            json=sample_log_entries,
            request=Mock(),
        )
        mock_httpx_client.get.return_value = mock_response

        client = MCPProxyClient()
        client.client = mock_httpx_client

        logs = await client.get_server_logs(
            "test-server",
            lines=50,
            filter_pattern="ERROR",
        )

        assert logs == sample_log_entries
        mock_httpx_client.get.assert_called_once_with(
            "/api/v1/agent/servers/test-server/logs",
            params={"lines": 50, "filter": "ERROR"},
        )

    @pytest.mark.asyncio
    async def test_get_server_logs_http_error(self, mock_httpx_client):
        """Test server logs retrieval handles HTTP errors."""
        mock_httpx_client.get.side_effect = httpx.HTTPStatusError(
            "500 Internal Server Error",
            request=Mock(),
            response=Response(status_code=500, request=Mock()),
        )

        client = MCPProxyClient()
        client.client = mock_httpx_client

        with pytest.raises(httpx.HTTPStatusError):
            await client.get_server_logs("test-server")

    @pytest.mark.asyncio
    async def test_get_server_status_success(
        self,
        mock_httpx_client,
        sample_server_status,
    ):
        """Test successful server status retrieval."""
        mock_response = Response(
            status_code=200,
            json=sample_server_status,
            request=Mock(),
        )
        mock_httpx_client.get.return_value = mock_response

        client = MCPProxyClient()
        client.client = mock_httpx_client

        status = await client.get_server_status("test-server")

        assert status == sample_server_status
        mock_httpx_client.get.assert_called_once_with(
            "/api/v1/agent/servers/test-server"
        )

    @pytest.mark.asyncio
    async def test_get_main_logs_success(
        self,
        mock_httpx_client,
        sample_log_entries,
    ):
        """Test successful main logs retrieval."""
        mock_response = Response(
            status_code=200,
            json=sample_log_entries,
            request=Mock(),
        )
        mock_httpx_client.get.return_value = mock_response

        client = MCPProxyClient()
        client.client = mock_httpx_client

        logs = await client.get_main_logs(lines=200)

        assert logs == sample_log_entries
        mock_httpx_client.get.assert_called_once_with(
            "/api/v1/agent/logs/main",
            params={"lines": 200},
        )


# ============================================================================
# DiagnosticTools Log Analysis Tests
# ============================================================================


@pytest.mark.unit
@pytest.mark.diagnostic
class TestDiagnosticToolsLogAnalysis:
    """Test DiagnosticTools log analysis functionality."""

    @pytest.mark.asyncio
    async def test_analyze_server_logs_basic(self, configured_mock_client):
        """Test basic server log analysis."""
        tools = DiagnosticTools(configured_mock_client)

        result = await tools.analyze_server_logs("test-server")

        # Verify result structure
        assert isinstance(result, LogAnalysisResult)
        assert result.server_name == "test-server"
        assert result.total_entries > 0
        assert result.error_count >= 0
        assert result.warning_count >= 0
        assert isinstance(result.patterns, list)
        assert isinstance(result.recommendations, list)
        assert isinstance(result.critical_issues, list)

        # Verify client was called
        configured_mock_client.get_server_logs.assert_called_once_with(
            "test-server",
            lines=500,
        )

    @pytest.mark.asyncio
    async def test_analyze_server_logs_counts_errors_correctly(
        self,
        mock_mcpproxy_client,
    ):
        """Test that error and warning counts are accurate."""
        # Create specific log entries with known counts
        test_logs = [
            {"level": "INFO", "message": "Info 1"},
            {"level": "ERROR", "message": "Error 1"},
            {"level": "ERROR", "message": "Error 2"},
            {"level": "WARN", "message": "Warning 1"},
            {"level": "WARN", "message": "Warning 2"},
            {"level": "WARN", "message": "Warning 3"},
            {"level": "INFO", "message": "Info 2"},
        ]
        mock_mcpproxy_client.get_server_logs.return_value = test_logs

        tools = DiagnosticTools(mock_mcpproxy_client)
        result = await tools.analyze_server_logs("test-server")

        assert result.total_entries == 7
        assert result.error_count == 2
        assert result.warning_count == 3

    @pytest.mark.asyncio
    async def test_analyze_server_logs_detects_patterns(
        self,
        mock_mcpproxy_client,
    ):
        """Test that common patterns are detected in logs."""
        # Create logs with repeated error pattern
        test_logs = [
            {"level": "ERROR", "message": "Authentication failed"},
            {"level": "ERROR", "message": "Authentication failed"},
            {"level": "ERROR", "message": "Authentication failed"},
            {"level": "ERROR", "message": "Timeout occurred"},
            {"level": "INFO", "message": "Normal operation"},
        ]
        mock_mcpproxy_client.get_server_logs.return_value = test_logs

        tools = DiagnosticTools(mock_mcpproxy_client)
        result = await tools.analyze_server_logs("test-server")

        # Verify patterns were detected
        assert len(result.patterns) > 0

        # Verify most common pattern is auth failure
        auth_pattern = next(
            (p for p in result.patterns if "Authentication failed" in p["pattern"]),
            None,
        )
        assert auth_pattern is not None
        assert auth_pattern["occurrences"] == 3

    @pytest.mark.asyncio
    async def test_analyze_server_logs_generates_recommendations(
        self,
        mock_mcpproxy_client,
    ):
        """Test that appropriate recommendations are generated."""
        # Create logs with OAuth errors
        test_logs = [
            {"level": "ERROR", "message": "OAuth token expired"} for _ in range(15)
        ]
        mock_mcpproxy_client.get_server_logs.return_value = test_logs

        tools = DiagnosticTools(mock_mcpproxy_client)
        result = await tools.analyze_server_logs("test-server")

        # Verify recommendations exist
        assert len(result.recommendations) > 0

        # Should recommend re-authentication for OAuth issues
        assert any(
            "re-authentication" in rec.lower() or "oauth" in rec.lower()
            for rec in result.recommendations
        )

    @pytest.mark.asyncio
    async def test_analyze_server_logs_identifies_critical_issues(
        self,
        mock_mcpproxy_client,
    ):
        """Test that critical issues are properly identified."""
        test_logs = [
            {"level": "INFO", "message": "Normal operation"},
            {"level": "CRITICAL", "message": "Server crashed: Out of memory"},
            {"level": "ERROR", "message": "Minor error"},
            {"level": "CRITICAL", "message": "Data corruption detected"},
        ]
        mock_mcpproxy_client.get_server_logs.return_value = test_logs

        tools = DiagnosticTools(mock_mcpproxy_client)
        result = await tools.analyze_server_logs("test-server")

        # Verify critical issues were identified
        assert len(result.critical_issues) == 2
        assert "Out of memory" in result.critical_issues[0]
        assert "Data corruption" in result.critical_issues[1]

    @pytest.mark.asyncio
    async def test_analyze_server_logs_with_time_range(
        self,
        configured_mock_client,
    ):
        """Test log analysis with time range parameter."""
        tools = DiagnosticTools(configured_mock_client)

        result = await tools.analyze_server_logs(
            "test-server",
            time_range="24h",
        )

        assert isinstance(result, LogAnalysisResult)
        # Note: time_range is currently informational, not implemented in filtering

    @pytest.mark.asyncio
    async def test_analyze_server_logs_with_error_patterns(
        self,
        configured_mock_client,
    ):
        """Test log analysis with specific error patterns."""
        tools = DiagnosticTools(configured_mock_client)

        result = await tools.analyze_server_logs(
            "test-server",
            error_patterns=["timeout", "authentication"],
        )

        assert isinstance(result, LogAnalysisResult)
        # Note: error_patterns is currently informational


# ============================================================================
# DiagnosticTools Connection Diagnostics Tests
# ============================================================================


@pytest.mark.unit
@pytest.mark.diagnostic
class TestDiagnosticToolsConnectionDiagnostics:
    """Test DiagnosticTools connection diagnostics functionality."""

    @pytest.mark.asyncio
    async def test_identify_connection_issues_connected_server(
        self,
        mock_mcpproxy_client,
        sample_server_status,
    ):
        """Test connection diagnostics for healthy server."""
        mock_mcpproxy_client.get_server_status.return_value = sample_server_status

        tools = DiagnosticTools(mock_mcpproxy_client)
        result = await tools.identify_connection_issues("test-server")

        # Verify result structure
        assert isinstance(result, ConnectionDiagnostic)
        assert result.server_name == "test-server"
        assert result.is_connected is True
        assert result.connection_state == "Ready"
        assert result.last_error is None
        assert result.retry_count == 0
        assert len(result.suggestions) == 0  # No suggestions for healthy server

    @pytest.mark.asyncio
    async def test_identify_connection_issues_auth_failure(
        self,
        mock_mcpproxy_client,
    ):
        """Test connection diagnostics suggests re-auth for auth failures."""
        failed_status = {
            "state": "Error",
            "last_error": "Authentication failed: Invalid OAuth token",
            "retry_count": 3,
        }
        mock_mcpproxy_client.get_server_status.return_value = failed_status

        tools = DiagnosticTools(mock_mcpproxy_client)
        result = await tools.identify_connection_issues("test-server")

        assert result.is_connected is False
        assert result.connection_state == "Error"
        assert "auth" in result.last_error.lower()

        # Verify re-authentication suggestion
        assert any(
            "re-authenticate" in suggestion.lower()
            for suggestion in result.suggestions
        )

    @pytest.mark.asyncio
    async def test_identify_connection_issues_timeout(
        self,
        mock_mcpproxy_client,
    ):
        """Test connection diagnostics suggests fixes for timeouts."""
        failed_status = {
            "state": "Error",
            "last_error": "Connection timeout after 30 seconds",
            "retry_count": 2,
        }
        mock_mcpproxy_client.get_server_status.return_value = failed_status

        tools = DiagnosticTools(mock_mcpproxy_client)
        result = await tools.identify_connection_issues("test-server")

        assert result.is_connected is False

        # Verify timeout-related suggestions
        assert any(
            "network" in suggestion.lower() or "timeout" in suggestion.lower()
            for suggestion in result.suggestions
        )

    @pytest.mark.asyncio
    async def test_identify_connection_issues_high_retry_count(
        self,
        mock_mcpproxy_client,
    ):
        """Test connection diagnostics handles high retry counts."""
        failed_status = {
            "state": "Connecting",
            "last_error": "Connection refused",
            "retry_count": 10,
        }
        mock_mcpproxy_client.get_server_status.return_value = failed_status

        tools = DiagnosticTools(mock_mcpproxy_client)
        result = await tools.identify_connection_issues("test-server")

        assert result.retry_count == 10

        # Should suggest reviewing logs or timeout settings
        assert any(
            "timeout" in suggestion.lower() or "log" in suggestion.lower()
            for suggestion in result.suggestions
        )


# ============================================================================
# DiagnosticTools Tool Failure Analysis Tests
# ============================================================================


@pytest.mark.unit
@pytest.mark.diagnostic
class TestDiagnosticToolsToolFailureAnalysis:
    """Test DiagnosticTools tool failure analysis functionality."""

    @pytest.mark.asyncio
    async def test_analyze_tool_failures_basic(self, mock_mcpproxy_client):
        """Test basic tool failure analysis."""
        failure_logs = [
            {"message": "Tool execution failed: create_issue"},
            {"message": "Error executing tool: list_repos"},
        ]
        mock_mcpproxy_client.get_server_logs.return_value = failure_logs

        tools = DiagnosticTools(mock_mcpproxy_client)
        result = await tools.analyze_tool_failures("test-server")

        # Verify result structure
        assert isinstance(result, ToolFailureAnalysis)
        assert result.server_name == "test-server"
        assert result.failure_count == 2
        assert isinstance(result.common_errors, list)
        assert isinstance(result.root_causes, list)
        assert isinstance(result.suggested_fixes, list)

    @pytest.mark.asyncio
    async def test_analyze_tool_failures_specific_tool(
        self,
        mock_mcpproxy_client,
    ):
        """Test tool failure analysis for specific tool."""
        failure_logs = [
            {"message": "Tool execution failed: create_issue - Invalid parameters"}
        ]
        mock_mcpproxy_client.get_server_logs.return_value = failure_logs

        tools = DiagnosticTools(mock_mcpproxy_client)
        result = await tools.analyze_tool_failures(
            "test-server",
            tool_name="create_issue",
        )

        assert result.tool_name == "create_issue"
        assert result.failure_count >= 0

    @pytest.mark.asyncio
    async def test_analyze_tool_failures_identifies_root_causes(
        self,
        mock_mcpproxy_client,
    ):
        """Test that root causes are identified from error patterns."""
        failure_logs = [
            {"message": "Authentication failed when calling tool"},
            {"message": "Tool not found: missing_tool"},
            {"message": "Connection timeout during tool execution"},
        ]
        mock_mcpproxy_client.get_server_logs.return_value = failure_logs

        tools = DiagnosticTools(mock_mcpproxy_client)
        result = await tools.analyze_tool_failures("test-server")

        # Should identify multiple root causes
        assert len(result.root_causes) > 0
        root_causes_text = " ".join(result.root_causes).lower()

        # Check for expected root cause categories
        assert (
            "authentication" in root_causes_text
            or "not found" in root_causes_text
            or "timeout" in root_causes_text
        )

    @pytest.mark.asyncio
    async def test_analyze_tool_failures_suggests_fixes(
        self,
        mock_mcpproxy_client,
    ):
        """Test that appropriate fixes are suggested."""
        failure_logs = [
            {"message": "Authentication failed when calling tool"} for _ in range(5)
        ]
        mock_mcpproxy_client.get_server_logs.return_value = failure_logs

        tools = DiagnosticTools(mock_mcpproxy_client)
        result = await tools.analyze_tool_failures("test-server")

        # Should suggest re-authentication
        assert len(result.suggested_fixes) > 0
        fixes_text = " ".join(result.suggested_fixes).lower()
        assert "auth" in fixes_text or "authentication" in fixes_text


# ============================================================================
# DiagnosticTools Fix Suggestions Tests
# ============================================================================


@pytest.mark.unit
@pytest.mark.diagnostic
class TestDiagnosticToolsFixSuggestions:
    """Test DiagnosticTools fix suggestion generation."""

    @pytest.mark.asyncio
    async def test_suggest_fixes_oauth_expired(self, configured_mock_client):
        """Test fix suggestions for expired OAuth token."""
        diagnostic_results = {
            "server_name": "test-server",
            "oauth_expired": True,
        }

        tools = DiagnosticTools(configured_mock_client)
        fixes = await tools.suggest_fixes(diagnostic_results)

        # Verify fix structure
        assert isinstance(fixes, list)
        assert all(isinstance(fix, Fix) for fix in fixes)

        # Find OAuth fix
        oauth_fix = next(
            (f for f in fixes if "oauth" in f.issue.lower()),
            None,
        )
        assert oauth_fix is not None
        assert oauth_fix.fix_type == "authentication"
        assert len(oauth_fix.commands) > 0
        assert "mcpproxy auth login" in oauth_fix.commands[0]
        assert oauth_fix.risk_level == SeverityLevel.INFO
        assert oauth_fix.requires_approval is True

    @pytest.mark.asyncio
    async def test_suggest_fixes_invalid_config(self, configured_mock_client):
        """Test fix suggestions for invalid configuration."""
        diagnostic_results = {
            "server_name": "test-server",
            "config_invalid": True,
        }

        tools = DiagnosticTools(configured_mock_client)
        fixes = await tools.suggest_fixes(diagnostic_results)

        # Find config fix
        config_fix = next(
            (f for f in fixes if "config" in f.issue.lower()),
            None,
        )
        assert config_fix is not None
        assert config_fix.fix_type == "configuration"
        assert len(config_fix.commands) > 0
        assert config_fix.risk_level == SeverityLevel.WARNING

    @pytest.mark.asyncio
    async def test_suggest_fixes_empty_results(self, configured_mock_client):
        """Test fix suggestions with no issues detected."""
        diagnostic_results = {
            "server_name": "test-server",
        }

        tools = DiagnosticTools(configured_mock_client)
        fixes = await tools.suggest_fixes(diagnostic_results)

        # Should return empty list for healthy server
        assert isinstance(fixes, list)
        # May be empty or contain general suggestions

    @pytest.mark.asyncio
    async def test_suggest_fixes_multiple_issues(self, configured_mock_client):
        """Test fix suggestions for multiple issues."""
        diagnostic_results = {
            "server_name": "test-server",
            "oauth_expired": True,
            "config_invalid": True,
        }

        tools = DiagnosticTools(configured_mock_client)
        fixes = await tools.suggest_fixes(diagnostic_results)

        # Should have multiple fixes
        assert len(fixes) >= 2

        # Verify both fix types are present
        fix_types = {fix.fix_type for fix in fixes}
        assert "authentication" in fix_types
        assert "configuration" in fix_types


# ============================================================================
# Private Method Tests
# ============================================================================


@pytest.mark.unit
@pytest.mark.diagnostic
class TestDiagnosticToolsPrivateMethods:
    """Test DiagnosticTools private helper methods."""

    def test_detect_patterns(self, configured_mock_client):
        """Test pattern detection in logs."""
        logs = [
            {"level": "ERROR", "message": "Connection timeout"},
            {"level": "ERROR", "message": "Connection timeout"},
            {"level": "ERROR", "message": "Connection timeout"},
            {"level": "ERROR", "message": "Auth failed"},
            {"level": "INFO", "message": "Normal operation"},
        ]

        tools = DiagnosticTools(configured_mock_client)
        patterns = tools._detect_patterns(logs)

        # Should detect the timeout pattern
        assert len(patterns) > 0
        timeout_pattern = patterns[0]
        assert timeout_pattern["pattern"] == "Connection timeout"
        assert timeout_pattern["occurrences"] == 3

    def test_generate_recommendations_high_errors(
        self,
        configured_mock_client,
    ):
        """Test recommendations for high error counts."""
        tools = DiagnosticTools(configured_mock_client)
        patterns = []
        recommendations = tools._generate_recommendations(
            patterns,
            error_count=50,
            warning_count=5,
        )

        assert len(recommendations) > 0
        assert any("error rate" in rec.lower() for rec in recommendations)

    def test_generate_recommendations_oauth_pattern(
        self,
        configured_mock_client,
    ):
        """Test recommendations for OAuth patterns."""
        tools = DiagnosticTools(configured_mock_client)
        patterns = [
            {"pattern": "OAuth token expired", "occurrences": 5, "severity": "high"}
        ]
        recommendations = tools._generate_recommendations(
            patterns,
            error_count=5,
            warning_count=0,
        )

        assert any(
            "oauth" in rec.lower() or "auth" in rec.lower()
            for rec in recommendations
        )

    def test_extract_common_errors(self, configured_mock_client):
        """Test extraction of common errors."""
        logs = [
            {"message": "Error A"},
            {"message": "Error A"},
            {"message": "Error B"},
        ]

        tools = DiagnosticTools(configured_mock_client)
        common_errors = tools._extract_common_errors(logs)

        assert len(common_errors) == 2
        assert common_errors[0]["error"] == "Error A"
        assert common_errors[0]["count"] == 2

    def test_identify_root_causes(self, configured_mock_client):
        """Test root cause identification."""
        common_errors = [
            {"error": "Authentication failed", "count": 5},
            {"error": "Connection timeout", "count": 3},
        ]

        tools = DiagnosticTools(configured_mock_client)
        root_causes = tools._identify_root_causes(common_errors)

        assert "Authentication failure" in root_causes
        assert "Connection timeout" in root_causes

    def test_suggest_fixes_from_root_causes(self, configured_mock_client):
        """Test fix suggestions from root causes."""
        root_causes = ["Authentication failure", "Connection timeout"]

        tools = DiagnosticTools(configured_mock_client)
        fixes = tools._suggest_fixes(root_causes)

        assert len(fixes) == 2
        assert any("auth" in fix.lower() for fix in fixes)
        assert any("timeout" in fix.lower() or "network" in fix.lower() for fix in fixes)

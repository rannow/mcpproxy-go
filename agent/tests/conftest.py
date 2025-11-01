"""Pytest configuration and shared fixtures for MCP Agent tests."""

import asyncio
from datetime import datetime
from typing import Any, AsyncGenerator, Dict, List
from unittest.mock import AsyncMock, MagicMock, Mock

import httpx
import pytest
from httpx import AsyncClient, Response


# ============================================================================
# Pytest Configuration
# ============================================================================


def pytest_configure(config):
    """Configure pytest with custom settings."""
    config.addinivalue_line(
        "markers",
        "unit: Unit tests for individual components",
    )
    config.addinivalue_line(
        "markers",
        "integration: Integration tests for workflows",
    )
    config.addinivalue_line(
        "markers",
        "e2e: End-to-end test scenarios",
    )


# ============================================================================
# Session-Scoped Fixtures (Shared Across All Tests)
# ============================================================================


@pytest.fixture(scope="session")
def event_loop():
    """Create event loop for async tests."""
    loop = asyncio.get_event_loop_policy().new_event_loop()
    yield loop
    loop.close()


@pytest.fixture(scope="session")
def base_url() -> str:
    """Base URL for mcpproxy API."""
    return "http://localhost:8080"


@pytest.fixture(scope="session")
def api_token() -> str:
    """Mock API token for testing."""
    return "test-api-token-12345"


# ============================================================================
# Function-Scoped Fixtures (Fresh Instance Per Test)
# ============================================================================


@pytest.fixture
def mock_httpx_client() -> AsyncMock:
    """Create a mock httpx.AsyncClient for testing."""
    client = AsyncMock(spec=AsyncClient)
    client.base_url = "http://localhost:8080"
    client.headers = {}
    return client


@pytest.fixture
def mock_mcpproxy_client(mock_httpx_client):
    """Create a mock MCPProxyClient for testing."""
    from mcp_agent.tools.diagnostic import MCPProxyClient

    client = Mock(spec=MCPProxyClient)
    client.base_url = "http://localhost:8080"
    client.headers = {}
    client.client = mock_httpx_client

    # Mock common methods
    client.get_server_logs = AsyncMock()
    client.get_server_status = AsyncMock()
    client.get_main_logs = AsyncMock()

    return client


# ============================================================================
# Test Data Fixtures
# ============================================================================


@pytest.fixture
def sample_server_config() -> Dict[str, Any]:
    """Sample MCP server configuration."""
    return {
        "name": "test-server",
        "url": "https://api.test.com/mcp",
        "protocol": "http",
        "enabled": True,
        "quarantined": False,
        "headers": {
            "Authorization": "Bearer test-token",
        },
    }


@pytest.fixture
def sample_stdio_server_config() -> Dict[str, Any]:
    """Sample stdio MCP server configuration."""
    return {
        "name": "test-stdio-server",
        "command": "npx",
        "args": ["@modelcontextprotocol/server-test"],
        "protocol": "stdio",
        "enabled": True,
        "quarantined": False,
        "env": {
            "API_KEY": "test-api-key",
        },
        "working_dir": "/home/user/projects/test",
    }


@pytest.fixture
def sample_log_entries() -> List[Dict[str, Any]]:
    """Sample log entries for testing."""
    return [
        {
            "timestamp": "2025-11-01T10:00:00Z",
            "level": "INFO",
            "server": "test-server",
            "message": "Server started successfully",
            "context": {},
        },
        {
            "timestamp": "2025-11-01T10:01:00Z",
            "level": "ERROR",
            "server": "test-server",
            "message": "Authentication failed: Invalid token",
            "context": {"error_code": "auth_failed"},
        },
        {
            "timestamp": "2025-11-01T10:02:00Z",
            "level": "WARN",
            "server": "test-server",
            "message": "Connection timeout after 30s",
            "context": {"retry_count": 3},
        },
        {
            "timestamp": "2025-11-01T10:03:00Z",
            "level": "ERROR",
            "server": "test-server",
            "message": "Authentication failed: Invalid token",
            "context": {"error_code": "auth_failed"},
        },
        {
            "timestamp": "2025-11-01T10:04:00Z",
            "level": "CRITICAL",
            "server": "test-server",
            "message": "Server crashed: Out of memory",
            "context": {},
        },
    ]


@pytest.fixture
def sample_server_status() -> Dict[str, Any]:
    """Sample server status response."""
    return {
        "name": "test-server",
        "state": "Ready",
        "is_connected": True,
        "last_error": None,
        "retry_count": 0,
        "uptime": 3600,
        "tools_count": 15,
    }


@pytest.fixture
def sample_failed_server_status() -> Dict[str, Any]:
    """Sample failed server status response."""
    return {
        "name": "test-server",
        "state": "Error",
        "is_connected": False,
        "last_error": "Authentication failed: OAuth token expired",
        "retry_count": 5,
        "uptime": 0,
        "tools_count": 0,
    }


@pytest.fixture
def sample_tool_list() -> List[Dict[str, Any]]:
    """Sample tool list response."""
    return [
        {
            "name": "test-server:create_issue",
            "description": "Create a new issue in the issue tracker",
            "input_schema": {
                "type": "object",
                "properties": {
                    "title": {"type": "string"},
                    "description": {"type": "string"},
                },
                "required": ["title"],
            },
        },
        {
            "name": "test-server:list_issues",
            "description": "List all issues",
            "input_schema": {
                "type": "object",
                "properties": {
                    "status": {"type": "string", "enum": ["open", "closed", "all"]},
                },
            },
        },
    ]


@pytest.fixture
def sample_oauth_config() -> Dict[str, Any]:
    """Sample OAuth configuration."""
    return {
        "authorization_url": "https://auth.test.com/oauth/authorize",
        "token_url": "https://auth.test.com/oauth/token",
        "client_id": "test-client-id",
        "scopes": ["read", "write"],
    }


# ============================================================================
# HTTP Response Fixtures
# ============================================================================


@pytest.fixture
def mock_server_logs_response(sample_log_entries) -> Response:
    """Mock HTTP response for server logs endpoint."""
    return Response(
        status_code=200,
        json=sample_log_entries,
        request=Mock(),
    )


@pytest.fixture
def mock_server_status_response(sample_server_status) -> Response:
    """Mock HTTP response for server status endpoint."""
    return Response(
        status_code=200,
        json=sample_server_status,
        request=Mock(),
    )


@pytest.fixture
def mock_failed_server_status_response(sample_failed_server_status) -> Response:
    """Mock HTTP response for failed server status endpoint."""
    return Response(
        status_code=200,
        json=sample_failed_server_status,
        request=Mock(),
    )


@pytest.fixture
def mock_tools_list_response(sample_tool_list) -> Response:
    """Mock HTTP response for tools list endpoint."""
    return Response(
        status_code=200,
        json={"tools": sample_tool_list},
        request=Mock(),
    )


@pytest.fixture
def mock_error_response() -> Response:
    """Mock HTTP error response."""
    return Response(
        status_code=500,
        json={"error": "Internal server error", "message": "Something went wrong"},
        request=Mock(),
    )


@pytest.fixture
def mock_auth_error_response() -> Response:
    """Mock HTTP 401 authentication error response."""
    return Response(
        status_code=401,
        json={"error": "Unauthorized", "message": "Invalid or expired token"},
        request=Mock(),
    )


# ============================================================================
# MCPProxyClient Response Fixtures
# ============================================================================


@pytest.fixture
def configured_mock_client(
    mock_mcpproxy_client,
    sample_log_entries,
    sample_server_status,
):
    """MCPProxyClient mock configured with default responses."""
    mock_mcpproxy_client.get_server_logs.return_value = sample_log_entries
    mock_mcpproxy_client.get_server_status.return_value = sample_server_status
    mock_mcpproxy_client.get_main_logs.return_value = sample_log_entries
    return mock_mcpproxy_client


@pytest.fixture
def failed_server_mock_client(
    mock_mcpproxy_client,
    sample_log_entries,
    sample_failed_server_status,
):
    """MCPProxyClient mock configured for failed server scenario."""
    # Add more error logs
    error_logs = sample_log_entries + [
        {
            "timestamp": "2025-11-01T10:05:00Z",
            "level": "ERROR",
            "server": "test-server",
            "message": "Connection timeout after 30s",
            "context": {"retry_count": 6},
        }
        for _ in range(10)
    ]

    mock_mcpproxy_client.get_server_logs.return_value = error_logs
    mock_mcpproxy_client.get_server_status.return_value = sample_failed_server_status
    mock_mcpproxy_client.get_main_logs.return_value = error_logs
    return mock_mcpproxy_client


# ============================================================================
# LangGraph State Fixtures
# ============================================================================


@pytest.fixture
def initial_agent_state() -> Dict[str, Any]:
    """Initial state for LangGraph agent."""
    return {
        "user_request": "Debug test-server",
        "conversation_history": [{"role": "user", "content": "Debug test-server"}],
        "current_task": "",
        "task_type": "diagnose",
        "target_server": "test-server",
        "server_status": None,
        "diagnostic_results": None,
        "test_results": None,
        "config_changes": None,
        "suggested_fixes": [],
        "requires_approval": False,
        "approval_granted": False,
        "next_action": None,
        "error": None,
        "completed": False,
    }


@pytest.fixture
def diagnostic_state_with_results(initial_agent_state, sample_server_status) -> Dict[str, Any]:
    """Agent state after diagnostic analysis."""
    state = initial_agent_state.copy()
    state.update(
        {
            "current_task": "Diagnosing server issues",
            "server_status": sample_server_status,
            "diagnostic_results": {
                "log_analysis": {
                    "server_name": "test-server",
                    "total_entries": 100,
                    "error_count": 15,
                    "warning_count": 30,
                    "patterns": [
                        {
                            "pattern": "Authentication failed",
                            "occurrences": 10,
                            "severity": "high",
                        }
                    ],
                    "recommendations": ["Re-authenticate with OAuth provider"],
                    "critical_issues": [],
                },
                "connection_status": {
                    "server_name": "test-server",
                    "is_connected": True,
                    "connection_state": "Ready",
                    "last_error": None,
                    "retry_count": 0,
                    "suggestions": [],
                },
            },
        }
    )
    return state


# ============================================================================
# Utility Fixtures
# ============================================================================


@pytest.fixture
def mock_datetime():
    """Mock datetime for consistent timestamps."""
    mock_dt = MagicMock()
    mock_dt.now.return_value = datetime(2025, 11, 1, 10, 0, 0)
    mock_dt.utcnow.return_value = datetime(2025, 11, 1, 10, 0, 0)
    return mock_dt


@pytest.fixture
def capture_logs(caplog):
    """Fixture to capture and analyze logs."""
    import logging

    caplog.set_level(logging.DEBUG)
    return caplog


# ============================================================================
# Async Helper Fixtures
# ============================================================================


@pytest.fixture
async def async_mock_context():
    """Async context manager for testing."""

    class AsyncContextManager:
        async def __aenter__(self):
            return self

        async def __aexit__(self, exc_type, exc_val, exc_tb):
            return None

    return AsyncContextManager()

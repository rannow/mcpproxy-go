"""Unit tests for TestingTools."""

import pytest
from unittest.mock import AsyncMock, MagicMock, patch
from httpx import Response, HTTPError

from mcp_agent.tools.testing import (
    TestingTools,
    TestStatus,
    ConnectionTestResult,
    ToolTestResult,
    HealthCheckResult,
    TestSuite,
)


# Fixtures

@pytest.fixture
def testing_tools():
    """Create TestingTools instance."""
    return TestingTools(base_url="http://localhost:8080")


@pytest.fixture
def sample_server_status():
    """Sample server status response."""
    return {
        "status": {
            "connected": True,
            "state": "Ready"
        },
        "tools": {
            "count": 10
        }
    }


@pytest.fixture
def sample_server_config():
    """Sample server configuration."""
    return {
        "name": "github-server",
        "enabled": True,
        "quarantined": False,
        "protocol": "http",
        "url": "https://api.github.com"
    }


# Enum Tests

class TestTestStatus:
    """Test TestStatus enum."""

    def test_status_values(self):
        """Test all status values."""
        assert TestStatus.PASSED == "passed"
        assert TestStatus.FAILED == "failed"
        assert TestStatus.SKIPPED == "skipped"
        assert TestStatus.ERROR == "error"


# Pydantic Model Tests

class TestConnectionTestResult:
    """Test ConnectionTestResult model."""

    def test_connection_success(self):
        """Test successful connection result."""
        result = ConnectionTestResult(
            server_name="github-server",
            connected=True,
            state="Ready",
            response_time_ms=150.5,
            tool_count=10,
            error=None
        )

        assert result.server_name == "github-server"
        assert result.connected is True
        assert result.state == "Ready"
        assert result.response_time_ms == 150.5
        assert result.tool_count == 10
        assert result.error is None

    def test_connection_failure(self):
        """Test failed connection result."""
        result = ConnectionTestResult(
            server_name="github-server",
            connected=False,
            state="Error",
            response_time_ms=500.0,
            error="Connection timeout"
        )

        assert result.connected is False
        assert result.state == "Error"
        assert result.error == "Connection timeout"
        assert result.tool_count is None


class TestToolTestResult:
    """Test ToolTestResult model."""

    def test_tool_test_passed(self):
        """Test passed tool test result."""
        result = ToolTestResult(
            tool_name="github:create_issue",
            status=TestStatus.PASSED,
            execution_time_ms=250.0,
            response={"issue_id": 123},
            test_args={"title": "Test"}
        )

        assert result.tool_name == "github:create_issue"
        assert result.status == TestStatus.PASSED
        assert result.execution_time_ms == 250.0
        assert result.response == {"issue_id": 123}
        assert result.test_args == {"title": "Test"}
        assert result.error is None

    def test_tool_test_failed(self):
        """Test failed tool test result."""
        result = ToolTestResult(
            tool_name="github:create_issue",
            status=TestStatus.FAILED,
            execution_time_ms=100.0,
            error="Invalid arguments",
            test_args={}
        )

        assert result.status == TestStatus.FAILED
        assert result.error == "Invalid arguments"
        assert result.response is None


class TestHealthCheckResult:
    """Test HealthCheckResult model."""

    def test_health_check_healthy(self):
        """Test healthy server check result."""
        result = HealthCheckResult(
            server_name="github-server",
            healthy=True,
            checks_passed=5,
            checks_failed=0,
            details={
                "connectivity": "✓ Server connected",
                "state": "✓ Server ready"
            },
            warnings=[]
        )

        assert result.healthy is True
        assert result.checks_passed == 5
        assert result.checks_failed == 0
        assert len(result.warnings) == 0

    def test_health_check_unhealthy(self):
        """Test unhealthy server check result."""
        result = HealthCheckResult(
            server_name="github-server",
            healthy=False,
            checks_passed=2,
            checks_failed=3,
            details={
                "connectivity": "✗ Not connected"
            },
            warnings=["Server response time exceeds 1000ms"]
        )

        assert result.healthy is False
        assert result.checks_failed == 3
        assert len(result.warnings) == 1


class TestTestSuite:
    """Test TestSuite model."""

    def test_test_suite(self):
        """Test test suite model."""
        tool_result = ToolTestResult(
            tool_name="test:tool",
            status=TestStatus.PASSED,
            execution_time_ms=100.0,
            test_args={}
        )

        suite = TestSuite(
            server_name="github-server",
            total_tests=10,
            passed=8,
            failed=1,
            skipped=1,
            errors=0,
            duration_ms=1500.0,
            results=[tool_result]
        )

        assert suite.total_tests == 10
        assert suite.passed == 8
        assert suite.failed == 1
        assert suite.skipped == 1
        assert suite.errors == 0
        assert suite.duration_ms == 1500.0
        assert len(suite.results) == 1


# TestingTools Tests

class TestTestingToolsInit:
    """Test TestingTools initialization."""

    def test_init_default(self):
        """Test initialization with default values."""
        tools = TestingTools()

        assert tools.base_url == "http://localhost:8080"
        assert tools.client is not None

    def test_init_custom_base_url(self):
        """Test initialization with custom base URL."""
        tools = TestingTools(base_url="http://custom:9000")

        assert tools.base_url == "http://custom:9000"


class TestTestServerConnection:
    """Test test_server_connection method."""

    @pytest.mark.asyncio
    async def test_connection_success(self, testing_tools, sample_server_status):
        """Test successful server connection test."""
        mock_response = AsyncMock(spec=Response)
        mock_response.json.return_value = sample_server_status
        mock_response.raise_for_status = MagicMock()

        testing_tools.client.get = AsyncMock(return_value=mock_response)

        result = await testing_tools.test_server_connection("github-server")

        assert isinstance(result, ConnectionTestResult)
        assert result.server_name == "github-server"
        assert result.connected is True
        assert result.state == "Ready"
        assert result.tool_count == 10
        assert result.response_time_ms is not None
        assert result.error is None

    @pytest.mark.asyncio
    async def test_connection_failure(self, testing_tools):
        """Test failed server connection test."""
        testing_tools.client.get = AsyncMock(
            side_effect=HTTPError("Connection failed")
        )

        result = await testing_tools.test_server_connection("github-server")

        assert result.connected is False
        assert result.state == "Error"
        assert result.error is not None
        assert "Connection failed" in result.error

    @pytest.mark.asyncio
    async def test_connection_disconnected_state(self, testing_tools):
        """Test connection with disconnected server state."""
        disconnected_status = {
            "status": {
                "connected": False,
                "state": "Disconnected"
            },
            "tools": {
                "count": 0
            }
        }

        mock_response = AsyncMock(spec=Response)
        mock_response.json.return_value = disconnected_status
        mock_response.raise_for_status = MagicMock()

        testing_tools.client.get = AsyncMock(return_value=mock_response)

        result = await testing_tools.test_server_connection("github-server")

        assert result.connected is False
        assert result.state == "Disconnected"
        assert result.tool_count == 0


class TestTestToolExecution:
    """Test test_tool_execution method."""

    @pytest.mark.asyncio
    async def test_tool_execution_success(self, testing_tools, sample_server_status):
        """Test successful tool execution test."""
        mock_response = AsyncMock(spec=Response)
        mock_response.json.return_value = sample_server_status
        mock_response.raise_for_status = MagicMock()

        testing_tools.client.get = AsyncMock(return_value=mock_response)

        result = await testing_tools.test_tool_execution(
            server_name="github",
            tool_name="create_issue",
            test_args={"title": "Test"}
        )

        assert isinstance(result, ToolTestResult)
        assert result.tool_name == "github:create_issue"
        assert result.status == TestStatus.PASSED
        assert result.execution_time_ms is not None
        assert result.test_args == {"title": "Test"}
        assert result.error is None

    @pytest.mark.asyncio
    async def test_tool_execution_no_args(self, testing_tools, sample_server_status):
        """Test tool execution without test arguments."""
        mock_response = AsyncMock(spec=Response)
        mock_response.json.return_value = sample_server_status
        mock_response.raise_for_status = MagicMock()

        testing_tools.client.get = AsyncMock(return_value=mock_response)

        result = await testing_tools.test_tool_execution(
            server_name="github",
            tool_name="list_repos"
        )

        assert result.test_args == {}
        assert result.status == TestStatus.PASSED

    @pytest.mark.asyncio
    async def test_tool_execution_error(self, testing_tools):
        """Test tool execution with error."""
        testing_tools.client.get = AsyncMock(
            side_effect=HTTPError("Server not found")
        )

        result = await testing_tools.test_tool_execution(
            server_name="github",
            tool_name="create_issue",
            test_args={"title": "Test"}
        )

        assert result.status == TestStatus.ERROR
        assert result.error is not None
        assert "Server not found" in result.error


class TestRunHealthCheck:
    """Test run_health_check method."""

    @pytest.mark.asyncio
    async def test_health_check_all_pass(self, testing_tools, sample_server_status, sample_server_config):
        """Test health check with all checks passing."""
        # Mock connection test
        with patch.object(testing_tools, 'test_server_connection', new_callable=AsyncMock) as mock_conn:
            mock_conn.return_value = ConnectionTestResult(
                server_name="github-server",
                connected=True,
                state="Ready",
                response_time_ms=150.0,
                tool_count=10
            )

            # Mock config check
            mock_config_response = AsyncMock(spec=Response)
            mock_config_response.status_code = 200
            mock_config_response.json.return_value = sample_server_config

            testing_tools.client.get = AsyncMock(return_value=mock_config_response)

            result = await testing_tools.run_health_check("github-server")

            assert isinstance(result, HealthCheckResult)
            assert result.healthy is True
            assert result.checks_failed == 0
            assert result.checks_passed >= 4
            assert "connectivity" in result.details
            assert "state" in result.details
            assert "response_time" in result.details
            assert "tools" in result.details

    @pytest.mark.asyncio
    async def test_health_check_connection_failed(self, testing_tools):
        """Test health check with connection failure."""
        with patch.object(testing_tools, 'test_server_connection', new_callable=AsyncMock) as mock_conn:
            mock_conn.return_value = ConnectionTestResult(
                server_name="github-server",
                connected=False,
                state="Error",
                error="Connection timeout"
            )

            # Mock config check
            testing_tools.client.get = AsyncMock(side_effect=HTTPError("Config error"))

            result = await testing_tools.run_health_check("github-server")

            assert result.healthy is False
            assert result.checks_failed > 0

    @pytest.mark.asyncio
    async def test_health_check_slow_response_warning(self, testing_tools, sample_server_config):
        """Test health check with slow response time warning."""
        with patch.object(testing_tools, 'test_server_connection', new_callable=AsyncMock) as mock_conn:
            mock_conn.return_value = ConnectionTestResult(
                server_name="github-server",
                connected=True,
                state="Ready",
                response_time_ms=1500.0,  # Slow response
                tool_count=10
            )

            mock_config_response = AsyncMock(spec=Response)
            mock_config_response.status_code = 200
            mock_config_response.json.return_value = sample_server_config

            testing_tools.client.get = AsyncMock(return_value=mock_config_response)

            result = await testing_tools.run_health_check("github-server")

            assert result.healthy is True  # Still healthy but with warning
            assert any("response time" in w.lower() for w in result.warnings)

    @pytest.mark.asyncio
    async def test_health_check_no_tools_warning(self, testing_tools, sample_server_config):
        """Test health check with no tools warning."""
        with patch.object(testing_tools, 'test_server_connection', new_callable=AsyncMock) as mock_conn:
            mock_conn.return_value = ConnectionTestResult(
                server_name="github-server",
                connected=True,
                state="Ready",
                response_time_ms=150.0,
                tool_count=0  # No tools
            )

            mock_config_response = AsyncMock(spec=Response)
            mock_config_response.status_code = 200
            mock_config_response.json.return_value = sample_server_config

            testing_tools.client.get = AsyncMock(return_value=mock_config_response)

            result = await testing_tools.run_health_check("github-server")

            assert any("no registered tools" in w.lower() for w in result.warnings)

    @pytest.mark.asyncio
    async def test_health_check_disabled_server(self, testing_tools):
        """Test health check for disabled server."""
        with patch.object(testing_tools, 'test_server_connection', new_callable=AsyncMock) as mock_conn:
            mock_conn.return_value = ConnectionTestResult(
                server_name="github-server",
                connected=True,
                state="Ready",
                response_time_ms=150.0,
                tool_count=10
            )

            disabled_config = {
                "name": "github-server",
                "enabled": False,
                "quarantined": False
            }

            mock_config_response = AsyncMock(spec=Response)
            mock_config_response.status_code = 200
            mock_config_response.json.return_value = disabled_config

            testing_tools.client.get = AsyncMock(return_value=mock_config_response)

            result = await testing_tools.run_health_check("github-server")

            assert any("disabled" in w.lower() for w in result.warnings)

    @pytest.mark.asyncio
    async def test_health_check_quarantined_warning(self, testing_tools):
        """Test health check for quarantined server."""
        with patch.object(testing_tools, 'test_server_connection', new_callable=AsyncMock) as mock_conn:
            mock_conn.return_value = ConnectionTestResult(
                server_name="github-server",
                connected=True,
                state="Ready",
                response_time_ms=150.0,
                tool_count=10
            )

            quarantined_config = {
                "name": "github-server",
                "enabled": True,
                "quarantined": True
            }

            mock_config_response = AsyncMock(spec=Response)
            mock_config_response.status_code = 200
            mock_config_response.json.return_value = quarantined_config

            testing_tools.client.get = AsyncMock(return_value=mock_config_response)

            result = await testing_tools.run_health_check("github-server")

            assert any("quarantined" in w.lower() for w in result.warnings)


class TestRunTestSuite:
    """Test run_test_suite method."""

    @pytest.mark.asyncio
    async def test_test_suite_all_passed(self, testing_tools):
        """Test suite with all tests passing."""
        with patch.object(testing_tools, 'test_server_connection', new_callable=AsyncMock) as mock_conn:
            mock_conn.return_value = ConnectionTestResult(
                server_name="github-server",
                connected=True,
                state="Ready",
                response_time_ms=150.0,
                tool_count=10
            )

            with patch.object(testing_tools, 'test_tool_execution', new_callable=AsyncMock) as mock_tool:
                mock_tool.return_value = ToolTestResult(
                    tool_name="github:test",
                    status=TestStatus.PASSED,
                    execution_time_ms=100.0,
                    test_args={}
                )

                tool_tests = [
                    {"tool_name": "create_issue", "args": {}},
                    {"tool_name": "list_repos", "args": {}}
                ]

                result = await testing_tools.run_test_suite(
                    server_name="github-server",
                    tool_tests=tool_tests
                )

                assert isinstance(result, TestSuite)
                assert result.total_tests == 2
                assert result.passed == 2
                assert result.failed == 0
                assert result.errors == 0
                assert len(result.results) == 2

    @pytest.mark.asyncio
    async def test_test_suite_server_disconnected(self, testing_tools):
        """Test suite with disconnected server."""
        with patch.object(testing_tools, 'test_server_connection', new_callable=AsyncMock) as mock_conn:
            mock_conn.return_value = ConnectionTestResult(
                server_name="github-server",
                connected=False,
                state="Error",
                error="Connection failed"
            )

            tool_tests = [
                {"tool_name": "create_issue", "args": {}},
                {"tool_name": "list_repos", "args": {}}
            ]

            result = await testing_tools.run_test_suite(
                server_name="github-server",
                tool_tests=tool_tests
            )

            # All tool tests should be skipped
            assert result.skipped == 2
            assert len(result.results) == 0

    @pytest.mark.asyncio
    async def test_test_suite_no_tool_tests(self, testing_tools):
        """Test suite without tool tests."""
        with patch.object(testing_tools, 'test_server_connection', new_callable=AsyncMock) as mock_conn:
            mock_conn.return_value = ConnectionTestResult(
                server_name="github-server",
                connected=True,
                state="Ready",
                response_time_ms=150.0,
                tool_count=10
            )

            result = await testing_tools.run_test_suite(
                server_name="github-server",
                tool_tests=None
            )

            assert result.total_tests == 0
            assert len(result.results) == 0

    @pytest.mark.asyncio
    async def test_test_suite_mixed_results(self, testing_tools):
        """Test suite with mixed pass/fail results."""
        with patch.object(testing_tools, 'test_server_connection', new_callable=AsyncMock) as mock_conn:
            mock_conn.return_value = ConnectionTestResult(
                server_name="github-server",
                connected=True,
                state="Ready",
                response_time_ms=150.0,
                tool_count=10
            )

            test_results = [
                ToolTestResult(
                    tool_name="github:test1",
                    status=TestStatus.PASSED,
                    execution_time_ms=100.0,
                    test_args={}
                ),
                ToolTestResult(
                    tool_name="github:test2",
                    status=TestStatus.FAILED,
                    execution_time_ms=50.0,
                    error="Test failed",
                    test_args={}
                ),
                ToolTestResult(
                    tool_name="github:test3",
                    status=TestStatus.ERROR,
                    execution_time_ms=25.0,
                    error="Server error",
                    test_args={}
                )
            ]

            call_count = 0
            async def mock_tool_exec(*args, **kwargs):
                nonlocal call_count
                result = test_results[call_count]
                call_count += 1
                return result

            with patch.object(testing_tools, 'test_tool_execution', side_effect=mock_tool_exec):
                tool_tests = [
                    {"tool_name": "test1", "args": {}},
                    {"tool_name": "test2", "args": {}},
                    {"tool_name": "test3", "args": {}}
                ]

                result = await testing_tools.run_test_suite(
                    server_name="github-server",
                    tool_tests=tool_tests
                )

                assert result.passed == 1
                assert result.failed == 1
                assert result.errors == 1

    @pytest.mark.asyncio
    async def test_test_suite_skip_invalid_test(self, testing_tools):
        """Test suite skipping invalid test configuration."""
        with patch.object(testing_tools, 'test_server_connection', new_callable=AsyncMock) as mock_conn:
            mock_conn.return_value = ConnectionTestResult(
                server_name="github-server",
                connected=True,
                state="Ready",
                response_time_ms=150.0,
                tool_count=10
            )

            # Tool test missing tool_name
            tool_tests = [
                {"args": {}},  # Missing tool_name
                {"tool_name": "valid_tool", "args": {}}
            ]

            with patch.object(testing_tools, 'test_tool_execution', new_callable=AsyncMock) as mock_tool:
                mock_tool.return_value = ToolTestResult(
                    tool_name="github:valid_tool",
                    status=TestStatus.PASSED,
                    execution_time_ms=100.0,
                    test_args={}
                )

                result = await testing_tools.run_test_suite(
                    server_name="github-server",
                    tool_tests=tool_tests
                )

                assert result.skipped == 1
                assert result.passed == 1


class TestValidateServerQuarantine:
    """Test validate_server_quarantine method."""

    @pytest.mark.asyncio
    async def test_validate_not_quarantined(self, testing_tools, sample_server_config):
        """Test validation of non-quarantined server."""
        mock_response = AsyncMock(spec=Response)
        mock_response.json.return_value = sample_server_config
        mock_response.raise_for_status = MagicMock()

        testing_tools.client.get = AsyncMock(return_value=mock_response)

        result = await testing_tools.validate_server_quarantine("github-server")

        assert result["server_name"] == "github-server"
        assert result["is_quarantined"] is False
        assert result["should_quarantine"] is False
        assert result["recommendation"] == "Safe to use"

    @pytest.mark.asyncio
    async def test_validate_quarantined_server(self, testing_tools):
        """Test validation of quarantined server."""
        quarantined_config = {
            "name": "suspicious-server",
            "enabled": False,
            "quarantined": True
        }

        mock_response = AsyncMock(spec=Response)
        mock_response.json.return_value = quarantined_config
        mock_response.raise_for_status = MagicMock()

        testing_tools.client.get = AsyncMock(return_value=mock_response)

        result = await testing_tools.validate_server_quarantine("suspicious-server")

        assert result["is_quarantined"] is True
        assert result["should_quarantine"] is True
        assert result["recommendation"] == "Keep quarantined"
        assert len(result["reasons"]) > 0

    @pytest.mark.asyncio
    async def test_validate_disabled_server(self, testing_tools):
        """Test validation of disabled server."""
        disabled_config = {
            "name": "github-server",
            "enabled": False,
            "quarantined": False
        }

        mock_response = AsyncMock(spec=Response)
        mock_response.json.return_value = disabled_config
        mock_response.raise_for_status = MagicMock()

        testing_tools.client.get = AsyncMock(return_value=mock_response)

        result = await testing_tools.validate_server_quarantine("github-server")

        assert any("disabled" in reason.lower() for reason in result["reasons"])

    @pytest.mark.asyncio
    async def test_validate_error_suggests_quarantine(self, testing_tools):
        """Test validation error suggests quarantine."""
        testing_tools.client.get = AsyncMock(
            side_effect=HTTPError("Cannot connect")
        )

        result = await testing_tools.validate_server_quarantine("github-server")

        assert result["should_quarantine"] is True
        assert result["recommendation"] == "Quarantine until validated"
        assert len(result["reasons"]) > 0


class TestClose:
    """Test close method."""

    @pytest.mark.asyncio
    async def test_close(self, testing_tools):
        """Test closing HTTP client."""
        testing_tools.client.aclose = AsyncMock()

        await testing_tools.close()

        testing_tools.client.aclose.assert_called_once()

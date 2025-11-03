"""Unit tests for LogTools."""

import pytest
from unittest.mock import AsyncMock, MagicMock
from httpx import Response, HTTPError
from datetime import datetime

from mcp_agent.tools.logs import (
    LogTools,
    LogEntry,
    LogAnalysis,
    LogQueryResult,
)


# Fixtures

@pytest.fixture
def log_tools():
    """Create LogTools instance."""
    return LogTools(base_url="http://localhost:8080")


@pytest.fixture
def sample_log_entries():
    """Sample log entries."""
    return [
        {
            "timestamp": "2025-01-15T10:30:00Z",
            "level": "INFO",
            "message": "Server started successfully",
            "raw": "2025-01-15T10:30:00Z INFO Server started successfully"
        },
        {
            "timestamp": "2025-01-15T10:31:00Z",
            "level": "ERROR",
            "message": "Connection failed: timeout",
            "raw": "2025-01-15T10:31:00Z ERROR Connection failed: timeout"
        },
        {
            "timestamp": "2025-01-15T10:32:00Z",
            "level": "WARN",
            "message": "Retry attempt 1",
            "raw": "2025-01-15T10:32:00Z WARN Retry attempt 1"
        }
    ]


@pytest.fixture
def sample_log_analysis():
    """Sample log analysis result."""
    return {
        "total_entries": 100,
        "error_count": 10,
        "warning_count": 5,
        "info_count": 80,
        "debug_count": 5,
        "most_common_errors": [
            "Connection failed: timeout",
            "Authentication failed",
            "Rate limit exceeded"
        ],
        "most_common_warnings": [
            "Retry attempt 1",
            "Cache miss"
        ],
        "time_range": "2025-01-15T10:00:00Z to 2025-01-15T11:00:00Z",
        "patterns_detected": [
            "High error rate: 10 errors in last 100 entries",
            "Connection failures detected"
        ]
    }


# Pydantic Model Tests

class TestLogEntry:
    """Test LogEntry model."""

    def test_log_entry_full(self):
        """Test log entry with all fields."""
        entry = LogEntry(
            timestamp="2025-01-15T10:30:00Z",
            level="ERROR",
            message="Connection failed",
            raw="2025-01-15T10:30:00Z ERROR Connection failed",
            context={"retry_count": 3}
        )

        assert entry.timestamp == "2025-01-15T10:30:00Z"
        assert entry.level == "ERROR"
        assert entry.message == "Connection failed"
        assert entry.raw == "2025-01-15T10:30:00Z ERROR Connection failed"
        assert entry.context == {"retry_count": 3}

    def test_log_entry_minimal(self):
        """Test log entry with minimal fields."""
        entry = LogEntry(
            message="Test message"
        )

        assert entry.timestamp is None
        assert entry.level is None
        assert entry.message == "Test message"
        assert entry.raw is None
        assert entry.context is None


class TestLogAnalysis:
    """Test LogAnalysis model."""

    def test_log_analysis_full(self, sample_log_analysis):
        """Test log analysis with all fields."""
        analysis = LogAnalysis(**sample_log_analysis)

        assert analysis.total_entries == 100
        assert analysis.error_count == 10
        assert analysis.warning_count == 5
        assert analysis.info_count == 80
        assert analysis.debug_count == 5
        assert len(analysis.most_common_errors) == 3
        assert len(analysis.most_common_warnings) == 2
        assert len(analysis.patterns_detected) == 2

    def test_log_analysis_minimal(self):
        """Test log analysis with minimal fields."""
        analysis = LogAnalysis(
            total_entries=50,
            error_count=0,
            warning_count=0,
            info_count=50,
            debug_count=0,
            most_common_errors=[],
            most_common_warnings=[]
        )

        assert analysis.total_entries == 50
        assert analysis.error_count == 0
        assert analysis.most_common_errors == []
        assert analysis.most_common_warnings == []
        assert analysis.patterns_detected == []


class TestLogQueryResult:
    """Test LogQueryResult model."""

    def test_log_query_result_with_entries(self, sample_log_entries):
        """Test log query result with entries."""
        entries = [LogEntry(**e) for e in sample_log_entries]
        result = LogQueryResult(
            server_name="github-server",
            logs=entries,
            count=3,
            limited=False,
            filter_applied="error"
        )

        assert len(result.logs) == 3
        assert result.count == 3
        assert result.limited is False
        assert result.server_name == "github-server"
        assert result.filter_applied == "error"

    def test_log_query_result_empty(self):
        """Test log query result with no entries."""
        result = LogQueryResult(
            logs=[],
            count=0,
            limited=False
        )

        assert len(result.logs) == 0
        assert result.count == 0
        assert result.server_name is None


# LogTools Tests

class TestLogToolsInit:
    """Test LogTools initialization."""

    def test_init_default(self):
        """Test initialization with default values."""
        tools = LogTools()

        assert tools.base_url == "http://localhost:8080"
        assert tools.client is not None

    def test_init_custom(self):
        """Test initialization with custom base URL."""
        tools = LogTools(base_url="http://custom:9000")

        assert tools.base_url == "http://custom:9000"


class TestReadMainLogs:
    """Test read_main_logs method."""

    @pytest.mark.asyncio
    async def test_read_main_logs_success(self, log_tools, sample_log_entries):
        """Test successful main logs read."""
        mock_response = AsyncMock(spec=Response)
        mock_response.json.return_value = {
            "logs": sample_log_entries,
            "count": 3,
            "limited": False
        }
        mock_response.raise_for_status = MagicMock()

        log_tools.client.get = AsyncMock(return_value=mock_response)

        result = await log_tools.read_main_logs(lines=100)

        assert isinstance(result, LogQueryResult)
        assert len(result.logs) == 3
        assert result.count == 3
        assert result.logs[0].level == "INFO"
        assert result.logs[1].level == "ERROR"

    @pytest.mark.asyncio
    async def test_read_main_logs_with_filter(self, log_tools):
        """Test reading main logs with filter pattern."""
        error_logs = [
            {
                "timestamp": "2025-01-15T10:31:00Z",
                "level": "ERROR",
                "message": "Connection failed",
                "raw": "2025-01-15T10:31:00Z ERROR Connection failed"
            }
        ]

        mock_response = AsyncMock(spec=Response)
        mock_response.json.return_value = {
            "logs": error_logs,
            "count": 1,
            "limited": False
        }
        mock_response.raise_for_status = MagicMock()

        log_tools.client.get = AsyncMock(return_value=mock_response)

        result = await log_tools.read_main_logs(lines=100, filter_pattern="error")

        assert len(result.logs) == 1
        assert result.logs[0].level == "ERROR"
        assert result.filter_applied == "error"

        # Verify API call included filter
        call_args = log_tools.client.get.call_args
        assert "filter" in call_args[1]["params"]
        assert call_args[1]["params"]["filter"] == "error"

    @pytest.mark.asyncio
    async def test_read_main_logs_http_error(self, log_tools):
        """Test read main logs with HTTP error."""
        log_tools.client.get = AsyncMock(
            side_effect=HTTPError("Connection failed")
        )

        result = await log_tools.read_main_logs()

        assert isinstance(result, LogQueryResult)
        assert len(result.logs) == 0
        assert result.count == 0


class TestReadServerLogs:
    """Test read_server_logs method."""

    @pytest.mark.asyncio
    async def test_read_server_logs_success(self, log_tools, sample_log_entries):
        """Test successful server logs read."""
        mock_response = AsyncMock(spec=Response)
        mock_response.json.return_value = {
            "logs": sample_log_entries,
            "count": 3,
            "limited": False
        }
        mock_response.raise_for_status = MagicMock()

        log_tools.client.get = AsyncMock(return_value=mock_response)

        result = await log_tools.read_server_logs("github-server", lines=50)

        assert isinstance(result, LogQueryResult)
        assert len(result.logs) == 3
        assert result.server_name == "github-server"

        # Verify API call
        call_args = log_tools.client.get.call_args
        assert "github-server" in call_args[0][0]

    @pytest.mark.asyncio
    async def test_read_server_logs_with_filter(self, log_tools):
        """Test reading server logs with filter pattern."""
        mock_response = AsyncMock(spec=Response)
        mock_response.json.return_value = {
            "logs": [],
            "count": 0,
            "limited": False
        }
        mock_response.raise_for_status = MagicMock()

        log_tools.client.get = AsyncMock(return_value=mock_response)

        await log_tools.read_server_logs(
            "github-server",
            filter_pattern="timeout"
        )

        # Verify API call included filter parameter
        call_args = log_tools.client.get.call_args
        assert "filter" in call_args[1]["params"]

    @pytest.mark.asyncio
    async def test_read_server_logs_http_error(self, log_tools):
        """Test read server logs with HTTP error."""
        log_tools.client.get = AsyncMock(
            side_effect=HTTPError("Server not found")
        )

        result = await log_tools.read_server_logs("nonexistent")

        assert len(result.logs) == 0


class TestAnalyzeLogs:
    """Test analyze_logs method."""

    @pytest.mark.asyncio
    async def test_analyze_logs_success(self, log_tools, sample_log_entries):
        """Test successful log analysis."""
        # Mock read_server_logs to return sample entries
        mock_response = AsyncMock(spec=Response)
        mock_response.json.return_value = {
            "logs": sample_log_entries,
            "count": 3,
            "limited": False
        }
        mock_response.raise_for_status = MagicMock()
        log_tools.client.get = AsyncMock(return_value=mock_response)

        result = await log_tools.analyze_logs("github-server", lines=100)

        assert isinstance(result, LogAnalysis)
        assert result.total_entries == 3
        # Sample has 1 error, 1 warn, 1 info
        assert result.error_count == 1
        assert result.warning_count == 1
        assert result.info_count == 1

    @pytest.mark.asyncio
    async def test_analyze_logs_main_logs(self, log_tools, sample_log_entries):
        """Test log analysis for main logs."""
        mock_response = AsyncMock(spec=Response)
        mock_response.json.return_value = {
            "logs": sample_log_entries,
            "count": 3,
            "limited": False
        }
        mock_response.raise_for_status = MagicMock()
        log_tools.client.get = AsyncMock(return_value=mock_response)

        result = await log_tools.analyze_logs(server_name=None, lines=100)

        assert isinstance(result, LogAnalysis)
        assert result.total_entries == 3

    @pytest.mark.asyncio
    async def test_analyze_logs_http_error(self, log_tools):
        """Test analyze logs with HTTP error."""
        log_tools.client.get = AsyncMock(
            side_effect=HTTPError("Analysis failed")
        )

        result = await log_tools.analyze_logs("github-server")

        assert result.total_entries == 0
        assert result.error_count == 0


class TestSearchLogsForPattern:
    """Test search_logs_for_pattern method."""

    @pytest.mark.asyncio
    async def test_search_logs_success(self, log_tools):
        """Test successful log pattern search."""
        matching_entries = [
            {
                "timestamp": "2025-01-15T10:31:00Z",
                "level": "ERROR",
                "message": "Connection failed: timeout",
                "raw": "2025-01-15T10:31:00Z ERROR Connection failed: timeout"
            }
        ]

        mock_response = AsyncMock(spec=Response)
        mock_response.json.return_value = {
            "logs": matching_entries,
            "count": 1,
            "limited": False
        }
        mock_response.raise_for_status = MagicMock()

        log_tools.client.get = AsyncMock(return_value=mock_response)

        result = await log_tools.search_logs_for_pattern(
            "timeout",
            server_name="github-server"
        )

        assert isinstance(result, list)
        assert len(result) == 1
        assert "timeout" in result[0].message.lower()

    @pytest.mark.asyncio
    async def test_search_logs_no_server_name(self, log_tools):
        """Test searching logs across all servers."""
        mock_response = AsyncMock(spec=Response)
        mock_response.json.return_value = {
            "logs": [],
            "count": 0,
            "limited": False
        }
        mock_response.raise_for_status = MagicMock()

        log_tools.client.get = AsyncMock(return_value=mock_response)

        result = await log_tools.search_logs_for_pattern("error")

        assert isinstance(result, list)
        assert len(result) == 0

    @pytest.mark.asyncio
    async def test_search_logs_http_error(self, log_tools):
        """Test search logs with HTTP error."""
        log_tools.client.get = AsyncMock(
            side_effect=HTTPError("Search failed")
        )

        result = await log_tools.search_logs_for_pattern("error")

        assert isinstance(result, list)
        assert len(result) == 0


class TestGetErrorSummary:
    """Test get_error_summary method."""

    @pytest.mark.asyncio
    async def test_get_error_summary_success(self, log_tools):
        """Test successful error summary retrieval."""
        # Create sample log entries with errors
        error_logs = [
            {
                "timestamp": "2025-01-15T10:30:00Z",
                "level": "ERROR",
                "message": "Connection failed",
                "raw": "2025-01-15T10:30:00Z ERROR Connection failed"
            },
            {
                "timestamp": "2025-01-15T10:31:00Z",
                "level": "ERROR",
                "message": "Connection failed",
                "raw": "2025-01-15T10:31:00Z ERROR Connection failed"
            }
        ]

        mock_response = AsyncMock(spec=Response)
        mock_response.json.return_value = {
            "logs": error_logs,
            "count": 2,
            "limited": False
        }
        mock_response.raise_for_status = MagicMock()
        log_tools.client.get = AsyncMock(return_value=mock_response)

        result = await log_tools.get_error_summary(server_name="github-server")

        assert result["total_errors"] == 2
        assert result["error_rate"] == 100.0  # 2 errors out of 2 total
        assert "Connection failed" in result["most_common_errors"]
        assert "severity" in result

    @pytest.mark.asyncio
    async def test_get_error_summary_no_errors(self, log_tools):
        """Test error summary with no errors."""
        info_logs = [
            {
                "timestamp": "2025-01-15T10:30:00Z",
                "level": "INFO",
                "message": "Server started",
                "raw": "2025-01-15T10:30:00Z INFO Server started"
            }
        ]

        mock_response = AsyncMock(spec=Response)
        mock_response.json.return_value = {
            "logs": info_logs,
            "count": 1,
            "limited": False
        }
        mock_response.raise_for_status = MagicMock()
        log_tools.client.get = AsyncMock(return_value=mock_response)

        result = await log_tools.get_error_summary()

        assert result["total_errors"] == 0
        assert result["severity"] == "low"

    @pytest.mark.asyncio
    async def test_get_error_summary_http_error(self, log_tools):
        """Test error summary with HTTP error."""
        log_tools.client.get = AsyncMock(
            side_effect=HTTPError("Summary failed")
        )

        result = await log_tools.get_error_summary()

        assert result["total_errors"] == 0
        assert result["error_rate"] == 0


class TestClose:
    """Test close method."""

    @pytest.mark.asyncio
    async def test_close(self, log_tools):
        """Test closing HTTP client."""
        log_tools.client.aclose = AsyncMock()

        await log_tools.close()

        log_tools.client.aclose.assert_called_once()

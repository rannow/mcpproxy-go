"""Log analysis tools."""

import httpx
import re
from typing import Optional, List, Dict, Any
from pydantic import BaseModel
from datetime import datetime
from collections import Counter


class LogEntry(BaseModel):
    """Structured log entry."""
    timestamp: Optional[str] = None
    level: Optional[str] = None
    message: str
    raw: Optional[str] = None
    context: Optional[Dict[str, Any]] = None


class LogAnalysis(BaseModel):
    """Analysis of log entries."""
    total_entries: int
    error_count: int
    warning_count: int
    info_count: int
    debug_count: int
    most_common_errors: List[str]
    most_common_warnings: List[str]
    time_range: Optional[str] = None
    patterns_detected: List[str] = []


class LogQueryResult(BaseModel):
    """Result of log query."""
    server_name: Optional[str] = None
    logs: List[LogEntry]
    count: int
    limited: bool
    filter_applied: Optional[str] = None


class LogTools:
    """Tools for reading and analyzing logs."""

    def __init__(self, base_url: str = "http://localhost:8080"):
        """Initialize log tools.

        Args:
            base_url: Base URL for mcpproxy agent API
        """
        self.base_url = base_url
        self.client = httpx.AsyncClient(timeout=30.0)

    async def read_main_logs(
        self,
        lines: int = 100,
        filter_pattern: Optional[str] = None
    ) -> LogQueryResult:
        """Read mcpproxy main logs.

        Args:
            lines: Number of log lines to retrieve (max 1000)
            filter_pattern: Optional pattern to filter logs (case-insensitive)

        Returns:
            LogQueryResult with log entries
        """
        try:
            params = {"lines": min(lines, 1000)}
            if filter_pattern:
                params["filter"] = filter_pattern

            response = await self.client.get(
                f"{self.base_url}/api/v1/agent/logs/main",
                params=params
            )
            response.raise_for_status()
            data = response.json()

            # Parse log entries
            log_entries = []
            for entry in data.get("logs", []):
                log_entries.append(LogEntry(
                    timestamp=entry.get("timestamp"),
                    level=entry.get("level"),
                    message=entry.get("message", ""),
                    raw=entry.get("raw"),
                    context=entry.get("context")
                ))

            return LogQueryResult(
                server_name=None,
                logs=log_entries,
                count=data.get("count", len(log_entries)),
                limited=data.get("limited", False),
                filter_applied=filter_pattern
            )

        except httpx.HTTPError as e:
            # Return empty result on error
            return LogQueryResult(
                logs=[],
                count=0,
                limited=False,
                filter_applied=filter_pattern
            )

    async def read_server_logs(
        self,
        server_name: str,
        lines: int = 100,
        filter_pattern: Optional[str] = None
    ) -> LogQueryResult:
        """Read server-specific logs.

        Args:
            server_name: Name of server to read logs from
            lines: Number of log lines to retrieve (max 1000)
            filter_pattern: Optional pattern to filter logs

        Returns:
            LogQueryResult with log entries
        """
        try:
            params = {"lines": min(lines, 1000)}
            if filter_pattern:
                params["filter"] = filter_pattern

            response = await self.client.get(
                f"{self.base_url}/api/v1/agent/servers/{server_name}/logs",
                params=params
            )
            response.raise_for_status()
            data = response.json()

            # Parse log entries
            log_entries = []
            for entry in data.get("logs", []):
                log_entries.append(LogEntry(
                    timestamp=entry.get("timestamp"),
                    level=entry.get("level"),
                    message=entry.get("message", ""),
                    raw=entry.get("raw"),
                    context=entry.get("context")
                ))

            return LogQueryResult(
                server_name=server_name,
                logs=log_entries,
                count=data.get("count", len(log_entries)),
                limited=data.get("limited", False),
                filter_applied=filter_pattern
            )

        except httpx.HTTPError as e:
            # Return empty result on error
            return LogQueryResult(
                server_name=server_name,
                logs=[],
                count=0,
                limited=False,
                filter_applied=filter_pattern
            )

    async def analyze_logs(
        self,
        server_name: Optional[str] = None,
        lines: int = 500
    ) -> LogAnalysis:
        """Analyze logs for patterns and issues.

        Args:
            server_name: Server to analyze (None for main logs)
            lines: Number of recent log lines to analyze

        Returns:
            LogAnalysis with summary and insights
        """
        # Read logs
        if server_name:
            result = await self.read_server_logs(server_name, lines=lines)
        else:
            result = await self.read_main_logs(lines=lines)

        logs = result.logs

        # Count by level
        error_count = 0
        warning_count = 0
        info_count = 0
        debug_count = 0

        errors = []
        warnings = []
        timestamps = []

        for entry in logs:
            level = (entry.level or "").upper()

            if level == "ERROR":
                error_count += 1
                errors.append(entry.message)
            elif level in ["WARN", "WARNING"]:
                warning_count += 1
                warnings.append(entry.message)
            elif level == "INFO":
                info_count += 1
            elif level == "DEBUG":
                debug_count += 1

            if entry.timestamp:
                timestamps.append(entry.timestamp)

        # Find most common errors and warnings
        error_counter = Counter(errors)
        warning_counter = Counter(warnings)

        most_common_errors = [msg for msg, _ in error_counter.most_common(5)]
        most_common_warnings = [msg for msg, _ in warning_counter.most_common(5)]

        # Detect patterns
        patterns = []
        if error_count > 10:
            patterns.append(f"High error rate: {error_count} errors in last {lines} entries")
        if warning_count > 20:
            patterns.append(f"Elevated warnings: {warning_count} warnings detected")

        # Detect specific error patterns
        for entry in logs:
            msg = entry.message.lower()
            if "connection" in msg and "failed" in msg:
                patterns.append("Connection failures detected")
                break

        for entry in logs:
            msg = entry.message.lower()
            if "timeout" in msg:
                patterns.append("Timeout issues detected")
                break

        for entry in logs:
            msg = entry.message.lower()
            if "auth" in msg or "oauth" in msg:
                patterns.append("Authentication-related messages")
                break

        # Time range
        time_range = None
        if timestamps and len(timestamps) > 1:
            time_range = f"{timestamps[0]} to {timestamps[-1]}"

        return LogAnalysis(
            total_entries=len(logs),
            error_count=error_count,
            warning_count=warning_count,
            info_count=info_count,
            debug_count=debug_count,
            most_common_errors=most_common_errors,
            most_common_warnings=most_common_warnings,
            time_range=time_range,
            patterns_detected=list(set(patterns))
        )

    async def search_logs_for_pattern(
        self,
        pattern: str,
        server_name: Optional[str] = None,
        lines: int = 500,
        case_sensitive: bool = False
    ) -> List[LogEntry]:
        """Search logs for specific pattern.

        Args:
            pattern: Regex pattern or string to search for
            server_name: Server to search (None for main logs)
            lines: Number of lines to search through
            case_sensitive: Whether search should be case-sensitive

        Returns:
            List of matching log entries
        """
        # Read logs
        if server_name:
            result = await self.read_server_logs(server_name, lines=lines)
        else:
            result = await self.read_main_logs(lines=lines)

        # Compile regex pattern
        flags = 0 if case_sensitive else re.IGNORECASE
        try:
            regex = re.compile(pattern, flags)
        except re.error:
            # If pattern is invalid regex, treat as literal string
            pattern_escaped = re.escape(pattern)
            regex = re.compile(pattern_escaped, flags)

        # Filter matching entries
        matching_entries = []
        for entry in result.logs:
            # Search in message and raw fields
            if regex.search(entry.message):
                matching_entries.append(entry)
            elif entry.raw and regex.search(entry.raw):
                matching_entries.append(entry)

        return matching_entries

    async def get_error_summary(
        self,
        server_name: Optional[str] = None,
        lines: int = 1000
    ) -> Dict[str, Any]:
        """Get summary of errors from logs.

        Args:
            server_name: Server to analyze (None for main logs)
            lines: Number of lines to analyze

        Returns:
            Dict with error summary and recommendations
        """
        analysis = await self.analyze_logs(server_name=server_name, lines=lines)

        summary = {
            "total_errors": analysis.error_count,
            "total_warnings": analysis.warning_count,
            "error_rate": (analysis.error_count / analysis.total_entries * 100) if analysis.total_entries > 0 else 0,
            "most_common_errors": analysis.most_common_errors,
            "most_common_warnings": analysis.most_common_warnings,
            "patterns": analysis.patterns_detected,
            "severity": "high" if analysis.error_count > 10 else "medium" if analysis.error_count > 0 else "low"
        }

        # Generate recommendations
        recommendations = []
        if analysis.error_count > 10:
            recommendations.append("High error rate - immediate investigation required")
        if "Connection failures detected" in analysis.patterns_detected:
            recommendations.append("Check network connectivity and server status")
        if "Timeout issues detected" in analysis.patterns_detected:
            recommendations.append("Investigate timeout causes - may need timeout adjustment")
        if "Authentication-related messages" in analysis.patterns_detected:
            recommendations.append("Check OAuth configuration and token validity")

        summary["recommendations"] = recommendations

        return summary

    async def close(self):
        """Close HTTP client."""
        await self.client.aclose()

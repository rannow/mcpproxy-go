"""Diagnostic tools for MCP server debugging and analysis."""

from typing import Optional, List, Dict, Any
from datetime import datetime
from pydantic import BaseModel, Field
from pydantic_ai import Agent, RunContext
import httpx
from enum import Enum


class SeverityLevel(str, Enum):
    """Error severity levels."""
    CRITICAL = "critical"
    ERROR = "error"
    WARNING = "warning"
    INFO = "info"


class LogEntry(BaseModel):
    """Individual log entry."""
    timestamp: datetime
    level: str
    server: Optional[str] = None
    message: str
    context: Dict[str, Any] = Field(default_factory=dict)


class LogAnalysisResult(BaseModel):
    """Result of log analysis."""
    server_name: str
    total_entries: int
    error_count: int
    warning_count: int
    patterns: List[Dict[str, Any]]
    recommendations: List[str]
    critical_issues: List[str]


class ConnectionDiagnostic(BaseModel):
    """Connection diagnostic result."""
    server_name: str
    is_connected: bool
    connection_state: str
    last_error: Optional[str] = None
    retry_count: int
    suggestions: List[str]


class ToolFailureAnalysis(BaseModel):
    """Analysis of tool execution failures."""
    server_name: str
    tool_name: Optional[str]
    failure_count: int
    common_errors: List[Dict[str, Any]]
    root_causes: List[str]
    suggested_fixes: List[str]


class Fix(BaseModel):
    """Suggested fix for an issue."""
    issue: str
    fix_type: str
    description: str
    commands: List[str]
    risk_level: SeverityLevel
    requires_approval: bool = True


class MCPProxyClient:
    """HTTP client for mcpproxy API."""

    def __init__(self, base_url: str = "http://localhost:8080", api_token: Optional[str] = None):
        self.base_url = base_url
        self.headers = {}
        if api_token:
            self.headers["Authorization"] = f"Bearer {api_token}"
        self.client = httpx.AsyncClient(base_url=base_url, headers=self.headers)

    async def get_server_logs(
        self,
        server_name: str,
        lines: int = 100,
        filter_pattern: Optional[str] = None
    ) -> List[Dict[str, Any]]:
        """Get server-specific logs."""
        params = {"lines": lines}
        if filter_pattern:
            params["filter"] = filter_pattern

        response = await self.client.get(
            f"/api/v1/agent/servers/{server_name}/logs",
            params=params
        )
        response.raise_for_status()
        return response.json()

    async def get_server_status(self, server_name: str) -> Dict[str, Any]:
        """Get server status."""
        response = await self.client.get(f"/api/v1/agent/servers/{server_name}")
        response.raise_for_status()
        return response.json()

    async def get_main_logs(
        self,
        lines: int = 100,
        filter_pattern: Optional[str] = None
    ) -> List[Dict[str, Any]]:
        """Get main mcpproxy logs."""
        params = {"lines": lines}
        if filter_pattern:
            params["filter"] = filter_pattern

        response = await self.client.get("/api/v1/agent/logs/main", params=params)
        response.raise_for_status()
        return response.json()


class DiagnosticTools:
    """Tools for diagnosing MCP server issues."""

    def __init__(self, mcpproxy_client: MCPProxyClient):
        self.client = mcpproxy_client
        self.agent = Agent(
            "openai:gpt-4",  # Can be configured to use Claude, Gemini, etc.
            deps_type=MCPProxyClient,
        )

    async def analyze_server_logs(
        self,
        server_name: str,
        time_range: Optional[str] = None,
        error_patterns: Optional[List[str]] = None,
    ) -> LogAnalysisResult:
        """
        Analyze server logs for errors and patterns.

        Args:
            server_name: Name of the MCP server
            time_range: Time range for log analysis (e.g., "1h", "24h")
            error_patterns: Specific error patterns to look for

        Returns:
            LogAnalysisResult with findings and recommendations
        """
        # Fetch logs from mcpproxy
        logs = await self.client.get_server_logs(server_name, lines=500)

        # Count errors and warnings
        error_count = sum(1 for log in logs if log.get("level") == "ERROR")
        warning_count = sum(1 for log in logs if log.get("level") == "WARN")

        # Use LLM to analyze patterns
        analysis_prompt = f"""
        Analyze these server logs for {server_name}:

        Total entries: {len(logs)}
        Errors: {error_count}
        Warnings: {warning_count}

        Recent logs:
        {logs[:50]}

        Identify:
        1. Common error patterns
        2. Root causes
        3. Recommendations for fixes
        4. Critical issues requiring immediate attention
        """

        # This would use the LLM to analyze
        # For now, return structured data
        patterns = self._detect_patterns(logs)
        recommendations = self._generate_recommendations(patterns, error_count, warning_count)
        critical_issues = self._identify_critical_issues(logs)

        return LogAnalysisResult(
            server_name=server_name,
            total_entries=len(logs),
            error_count=error_count,
            warning_count=warning_count,
            patterns=patterns,
            recommendations=recommendations,
            critical_issues=critical_issues,
        )

    async def identify_connection_issues(
        self,
        server_name: str,
    ) -> ConnectionDiagnostic:
        """
        Diagnose connection problems for a specific server.

        Args:
            server_name: Name of the MCP server

        Returns:
            ConnectionDiagnostic with status and suggestions
        """
        status = await self.client.get_server_status(server_name)

        connection_state = status.get("state", "unknown")
        is_connected = connection_state == "Ready"
        last_error = status.get("last_error")
        retry_count = status.get("retry_count", 0)

        suggestions = []
        if not is_connected:
            if "auth" in str(last_error).lower():
                suggestions.append("Re-authenticate: mcpproxy auth login --server=" + server_name)
            if "timeout" in str(last_error).lower():
                suggestions.append("Check network connectivity and server availability")
            if retry_count > 5:
                suggestions.append("Consider increasing timeout or checking server logs")

        return ConnectionDiagnostic(
            server_name=server_name,
            is_connected=is_connected,
            connection_state=connection_state,
            last_error=last_error,
            retry_count=retry_count,
            suggestions=suggestions,
        )

    async def analyze_tool_failures(
        self,
        server_name: str,
        tool_name: Optional[str] = None,
    ) -> ToolFailureAnalysis:
        """
        Analyze tool execution failures.

        Args:
            server_name: Name of the MCP server
            tool_name: Specific tool to analyze (optional)

        Returns:
            ToolFailureAnalysis with failure patterns and fixes
        """
        # Get logs filtered for tool failures
        logs = await self.client.get_server_logs(
            server_name,
            lines=200,
            filter_pattern="tool.*fail|error.*executing"
        )

        failure_count = len(logs)
        common_errors = self._extract_common_errors(logs)
        root_causes = self._identify_root_causes(common_errors)
        suggested_fixes = self._suggest_fixes(root_causes)

        return ToolFailureAnalysis(
            server_name=server_name,
            tool_name=tool_name,
            failure_count=failure_count,
            common_errors=common_errors,
            root_causes=root_causes,
            suggested_fixes=suggested_fixes,
        )

    async def suggest_fixes(
        self,
        diagnostic_results: Dict[str, Any],
    ) -> List[Fix]:
        """
        AI-powered fix suggestions based on diagnostic results.

        Args:
            diagnostic_results: Results from diagnostic analysis

        Returns:
            List of Fix suggestions with commands and risk levels
        """
        fixes = []

        # Example fix suggestions based on common patterns
        if diagnostic_results.get("oauth_expired"):
            fixes.append(Fix(
                issue="OAuth token expired",
                fix_type="authentication",
                description="Re-authenticate with OAuth provider",
                commands=[
                    f"mcpproxy auth login --server={diagnostic_results['server_name']}"
                ],
                risk_level=SeverityLevel.INFO,
                requires_approval=True,
            ))

        if diagnostic_results.get("config_invalid"):
            fixes.append(Fix(
                issue="Invalid configuration detected",
                fix_type="configuration",
                description="Reset configuration to defaults",
                commands=[
                    f"mcpproxy config validate --server={diagnostic_results['server_name']}",
                    f"mcpproxy config reset --server={diagnostic_results['server_name']}",
                ],
                risk_level=SeverityLevel.WARNING,
                requires_approval=True,
            ))

        return fixes

    def _detect_patterns(self, logs: List[Dict[str, Any]]) -> List[Dict[str, Any]]:
        """Detect common patterns in logs."""
        patterns = []

        # Simple pattern detection (would use LLM in production)
        error_messages = [log.get("message", "") for log in logs if log.get("level") == "ERROR"]

        if error_messages:
            # Group similar errors
            from collections import Counter
            error_counts = Counter(error_messages)

            for error, count in error_counts.most_common(5):
                patterns.append({
                    "pattern": error,
                    "occurrences": count,
                    "severity": "high" if count > 10 else "medium"
                })

        return patterns

    def _generate_recommendations(
        self,
        patterns: List[Dict[str, Any]],
        error_count: int,
        warning_count: int
    ) -> List[str]:
        """Generate recommendations based on analysis."""
        recommendations = []

        if error_count > 10:
            recommendations.append("High error rate detected - investigate server stability")

        if warning_count > 50:
            recommendations.append("Many warnings - review configuration and dependencies")

        for pattern in patterns:
            if "oauth" in pattern["pattern"].lower():
                recommendations.append("OAuth issues detected - consider re-authentication")
            if "timeout" in pattern["pattern"].lower():
                recommendations.append("Timeout issues detected - check network and increase timeout values")

        return recommendations

    def _identify_critical_issues(self, logs: List[Dict[str, Any]]) -> List[str]:
        """Identify critical issues requiring immediate attention."""
        critical = []

        for log in logs:
            if log.get("level") == "CRITICAL":
                critical.append(log.get("message", "Unknown critical issue"))

        return critical

    def _extract_common_errors(self, logs: List[Dict[str, Any]]) -> List[Dict[str, Any]]:
        """Extract and group common errors."""
        from collections import Counter

        error_messages = [log.get("message", "") for log in logs]
        error_counts = Counter(error_messages)

        return [
            {"error": error, "count": count}
            for error, count in error_counts.most_common(10)
        ]

    def _identify_root_causes(self, common_errors: List[Dict[str, Any]]) -> List[str]:
        """Identify root causes from common errors."""
        root_causes = []

        for error_info in common_errors:
            error = error_info["error"].lower()
            if "authentication" in error or "auth" in error:
                root_causes.append("Authentication failure")
            elif "timeout" in error:
                root_causes.append("Connection timeout")
            elif "not found" in error:
                root_causes.append("Resource not found")

        return list(set(root_causes))

    def _suggest_fixes(self, root_causes: List[str]) -> List[str]:
        """Suggest fixes for identified root causes."""
        fixes = []

        for cause in root_causes:
            if "authentication" in cause.lower():
                fixes.append("Re-authenticate with OAuth provider")
            elif "timeout" in cause.lower():
                fixes.append("Increase timeout values or check network connectivity")
            elif "not found" in cause.lower():
                fixes.append("Verify resource paths and configuration")

        return fixes

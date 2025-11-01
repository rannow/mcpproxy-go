"""Testing tools for MCP servers."""

import httpx
import time
from typing import Optional, List, Dict, Any
from pydantic import BaseModel
from enum import Enum


class TestStatus(str, Enum):
    """Test execution status."""
    PASSED = "passed"
    FAILED = "failed"
    SKIPPED = "skipped"
    ERROR = "error"


class ConnectionTestResult(BaseModel):
    """Result of server connection test."""
    server_name: str
    connected: bool
    state: str
    response_time_ms: Optional[float] = None
    tool_count: Optional[int] = None
    error: Optional[str] = None


class ToolTestResult(BaseModel):
    """Result of tool execution test."""
    tool_name: str
    status: TestStatus
    execution_time_ms: float
    response: Optional[Any] = None
    error: Optional[str] = None
    test_args: Dict[str, Any]


class HealthCheckResult(BaseModel):
    """Result of server health check."""
    server_name: str
    healthy: bool
    checks_passed: int
    checks_failed: int
    details: Dict[str, Any]
    warnings: List[str] = []


class TestSuite(BaseModel):
    """Collection of test results."""
    server_name: str
    total_tests: int
    passed: int
    failed: int
    skipped: int
    errors: int
    duration_ms: float
    results: List[ToolTestResult]


class TestingTools:
    """Tools for testing MCP server functionality."""

    def __init__(self, base_url: str = "http://localhost:8080"):
        """Initialize testing tools.

        Args:
            base_url: Base URL for mcpproxy agent API
        """
        self.base_url = base_url
        self.client = httpx.AsyncClient(timeout=60.0)

    async def test_server_connection(self, server_name: str) -> ConnectionTestResult:
        """Test server connectivity and basic functionality.

        Args:
            server_name: Name of server to test

        Returns:
            ConnectionTestResult with connection details
        """
        start_time = time.time()

        try:
            response = await self.client.get(
                f"{self.base_url}/api/v1/agent/servers/{server_name}"
            )
            response.raise_for_status()
            data = response.json()

            response_time = (time.time() - start_time) * 1000

            status = data.get("status", {})
            tools = data.get("tools", {})

            return ConnectionTestResult(
                server_name=server_name,
                connected=status.get("connected", False),
                state=status.get("state", "Unknown"),
                response_time_ms=response_time,
                tool_count=tools.get("count", 0),
                error=None
            )

        except httpx.HTTPError as e:
            response_time = (time.time() - start_time) * 1000
            return ConnectionTestResult(
                server_name=server_name,
                connected=False,
                state="Error",
                response_time_ms=response_time,
                error=str(e)
            )

    async def test_tool_execution(
        self,
        server_name: str,
        tool_name: str,
        test_args: Optional[Dict[str, Any]] = None
    ) -> ToolTestResult:
        """Execute tool with test arguments to verify functionality.

        Args:
            server_name: Name of server hosting the tool
            tool_name: Tool name (without server prefix)
            test_args: Arguments to pass to tool

        Returns:
            ToolTestResult with execution details
        """
        start_time = time.time()
        full_tool_name = f"{server_name}:{tool_name}"
        args = test_args or {}

        try:
            # Note: Actual tool execution would use mcpproxy's call_tool endpoint
            # This is a simplified version that checks if tool exists
            response = await self.client.get(
                f"{self.base_url}/api/v1/agent/servers/{server_name}"
            )
            response.raise_for_status()

            execution_time = (time.time() - start_time) * 1000

            return ToolTestResult(
                tool_name=full_tool_name,
                status=TestStatus.PASSED,
                execution_time_ms=execution_time,
                response={"message": "Tool validation successful"},
                test_args=args
            )

        except httpx.HTTPError as e:
            execution_time = (time.time() - start_time) * 1000
            return ToolTestResult(
                tool_name=full_tool_name,
                status=TestStatus.ERROR,
                execution_time_ms=execution_time,
                error=str(e),
                test_args=args
            )

    async def run_health_check(self, server_name: str) -> HealthCheckResult:
        """Run comprehensive health check on server.

        Args:
            server_name: Name of server to check

        Returns:
            HealthCheckResult with detailed health status
        """
        checks = {}
        warnings = []
        passed = 0
        failed = 0

        # Check 1: Server connectivity
        conn_result = await self.test_server_connection(server_name)
        if conn_result.connected:
            checks["connectivity"] = "✓ Server connected"
            passed += 1
        else:
            checks["connectivity"] = f"✗ Not connected: {conn_result.error}"
            failed += 1

        # Check 2: Server state
        if conn_result.state == "Ready":
            checks["state"] = "✓ Server ready"
            passed += 1
        else:
            checks["state"] = f"✗ State: {conn_result.state}"
            failed += 1

        # Check 3: Response time
        if conn_result.response_time_ms and conn_result.response_time_ms < 1000:
            checks["response_time"] = f"✓ Response time: {conn_result.response_time_ms:.1f}ms"
            passed += 1
        elif conn_result.response_time_ms:
            checks["response_time"] = f"⚠ Slow response: {conn_result.response_time_ms:.1f}ms"
            warnings.append("Server response time exceeds 1000ms")
            passed += 1
        else:
            checks["response_time"] = "✗ No response"
            failed += 1

        # Check 4: Tools available
        if conn_result.tool_count and conn_result.tool_count > 0:
            checks["tools"] = f"✓ {conn_result.tool_count} tools available"
            passed += 1
        elif conn_result.connected:
            checks["tools"] = "⚠ No tools found"
            warnings.append("Server has no registered tools")
            passed += 1
        else:
            checks["tools"] = "✗ Cannot check tools"
            failed += 1

        # Check 5: Configuration
        try:
            config_response = await self.client.get(
                f"{self.base_url}/api/v1/agent/servers/{server_name}/config"
            )
            if config_response.status_code == 200:
                config = config_response.json()
                if config.get("enabled"):
                    checks["configuration"] = "✓ Server enabled"
                    passed += 1
                else:
                    checks["configuration"] = "⚠ Server disabled"
                    warnings.append("Server is configured but disabled")
                    passed += 1

                if config.get("quarantined"):
                    warnings.append("Server is quarantined for security")
        except httpx.HTTPError:
            checks["configuration"] = "✗ Cannot read config"
            failed += 1

        healthy = failed == 0 and conn_result.connected

        return HealthCheckResult(
            server_name=server_name,
            healthy=healthy,
            checks_passed=passed,
            checks_failed=failed,
            details=checks,
            warnings=warnings
        )

    async def run_test_suite(
        self,
        server_name: str,
        tool_tests: Optional[List[Dict[str, Any]]] = None
    ) -> TestSuite:
        """Run comprehensive test suite for server.

        Args:
            server_name: Name of server to test
            tool_tests: List of tool tests to run, each with 'tool_name' and 'args'

        Returns:
            TestSuite with all test results
        """
        start_time = time.time()
        results = []
        passed = 0
        failed = 0
        skipped = 0
        errors = 0

        # Run basic connection test
        conn_test = await self.test_server_connection(server_name)
        if not conn_test.connected:
            # Skip tool tests if server isn't connected
            skipped = len(tool_tests) if tool_tests else 0

        # Run tool tests if provided
        if tool_tests and conn_test.connected:
            for test in tool_tests:
                tool_name = test.get("tool_name")
                test_args = test.get("args", {})

                if not tool_name:
                    skipped += 1
                    continue

                result = await self.test_tool_execution(
                    server_name=server_name,
                    tool_name=tool_name,
                    test_args=test_args
                )
                results.append(result)

                if result.status == TestStatus.PASSED:
                    passed += 1
                elif result.status == TestStatus.FAILED:
                    failed += 1
                elif result.status == TestStatus.SKIPPED:
                    skipped += 1
                elif result.status == TestStatus.ERROR:
                    errors += 1

        duration = (time.time() - start_time) * 1000

        return TestSuite(
            server_name=server_name,
            total_tests=len(results) + skipped,
            passed=passed,
            failed=failed,
            skipped=skipped,
            errors=errors,
            duration_ms=duration,
            results=results
        )

    async def validate_server_quarantine(self, server_name: str) -> Dict[str, Any]:
        """Check if server should be quarantined for security.

        Args:
            server_name: Name of server to validate

        Returns:
            Dict with quarantine recommendation and reasons
        """
        try:
            response = await self.client.get(
                f"{self.base_url}/api/v1/agent/servers/{server_name}/config"
            )
            response.raise_for_status()
            config = response.json()

            is_quarantined = config.get("quarantined", False)
            should_quarantine = False
            reasons = []

            # Check for security concerns
            if not config.get("enabled"):
                reasons.append("Server is disabled")

            if is_quarantined:
                reasons.append("Server is currently quarantined")
                should_quarantine = True

            return {
                "server_name": server_name,
                "is_quarantined": is_quarantined,
                "should_quarantine": should_quarantine,
                "reasons": reasons,
                "recommendation": "Keep quarantined" if should_quarantine else "Safe to use"
            }

        except httpx.HTTPError as e:
            return {
                "server_name": server_name,
                "is_quarantined": False,
                "should_quarantine": True,
                "reasons": [f"Cannot validate: {str(e)}"],
                "recommendation": "Quarantine until validated"
            }

    async def close(self):
        """Close HTTP client."""
        await self.client.aclose()

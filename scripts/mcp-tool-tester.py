#!/usr/bin/env python3
"""
MCP Proxy Tool Tester
=====================
Tests all MCP tools via the MCPProxy HTTP API with timeout handling.

Usage:
    python scripts/mcp-tool-tester.py [--timeout 10] [--output logs/mcp-test.log]
"""

import argparse
import json
import time
import os
from datetime import datetime
from dataclasses import dataclass, field
from typing import Optional, Any
import urllib.request
import urllib.error
import socket

# Configuration
DEFAULT_PROXY_URL = "http://localhost:8080"
DEFAULT_TIMEOUT = 10  # seconds
DEFAULT_LOG_FILE = "logs/mcp-tool-test.log"


@dataclass
class TestResult:
    """Result of a single tool test"""
    server_name: str
    tool_name: str
    success: bool
    duration_ms: float
    request: dict
    response: Optional[Any] = None
    error: Optional[str] = None
    timestamp: str = field(default_factory=lambda: datetime.now().isoformat())


class MCPProxyTester:
    """MCP Proxy test client with timeout handling"""

    def __init__(self, proxy_url: str, timeout: int, log_file: str):
        self.proxy_url = proxy_url.rstrip('/')
        self.timeout = timeout
        self.log_file = log_file
        self.results: list[TestResult] = []
        self.log_handle = None

    def log(self, message: str, level: str = "INFO"):
        """Write to both console and log file"""
        timestamp = datetime.now().strftime("%Y-%m-%d %H:%M:%S")
        formatted = f"[{timestamp}] [{level}] {message}"
        print(formatted)
        if self.log_handle:
            self.log_handle.write(formatted + "\n")
            self.log_handle.flush()

    def http_request(self, endpoint: str, method: str = "GET", data: Optional[dict] = None) -> tuple[Any, float, Optional[str]]:
        """Make HTTP request with timeout, returns (response, duration_ms, error)"""
        url = f"{self.proxy_url}{endpoint}"
        start_time = time.time()

        try:
            if data is not None:
                req = urllib.request.Request(
                    url,
                    data=json.dumps(data).encode('utf-8'),
                    headers={'Content-Type': 'application/json'},
                    method=method
                )
            else:
                req = urllib.request.Request(url, method=method)

            with urllib.request.urlopen(req, timeout=self.timeout) as response:
                body = response.read().decode('utf-8')
                duration_ms = (time.time() - start_time) * 1000

                try:
                    return json.loads(body), duration_ms, None
                except json.JSONDecodeError:
                    return body, duration_ms, None

        except urllib.error.HTTPError as e:
            duration_ms = (time.time() - start_time) * 1000
            try:
                error_body = e.read().decode('utf-8')
                return None, duration_ms, f"HTTP {e.code}: {error_body[:200]}"
            except:
                return None, duration_ms, f"HTTP {e.code}: {e.reason}"

        except urllib.error.URLError as e:
            duration_ms = (time.time() - start_time) * 1000
            return None, duration_ms, f"URL Error: {e.reason}"

        except socket.timeout:
            duration_ms = (time.time() - start_time) * 1000
            return None, duration_ms, f"Timeout after {self.timeout}s"

        except Exception as e:
            duration_ms = (time.time() - start_time) * 1000
            return None, duration_ms, f"Error: {str(e)}"

    def get_config(self) -> Optional[dict]:
        """Get MCPProxy configuration"""
        self.log("Fetching MCPProxy configuration...")
        response, duration, error = self.http_request("/chat/read-config", "POST", {})

        if error:
            self.log(f"Failed to get config: {error}", "ERROR")
            return None

        if isinstance(response, dict) and 'content' in response:
            try:
                config = json.loads(response['content'])
                self.log(f"Config loaded in {duration:.1f}ms")
                return config
            except:
                pass

        self.log(f"Config response in {duration:.1f}ms")
        return response

    def get_servers(self, config: dict) -> list[dict]:
        """Extract active servers from config"""
        servers = config.get('mcpServers', config.get('servers', []))
        active_servers = [
            s for s in servers
            if s.get('startup_mode', 'active') == 'active'
        ]
        self.log(f"Found {len(active_servers)} active servers out of {len(servers)} total")
        return active_servers

    def call_tool(self, server_name: str, tool_name: str, arguments: Optional[dict] = None) -> TestResult:
        """Call a specific tool and return test result"""
        if arguments is None:
            arguments = {}

        request_data = {
            "server_name": server_name,
            "tool_name": tool_name,
            "arguments": arguments
        }

        self.log(f"Testing {server_name}:{tool_name}...")
        response, duration, error = self.http_request("/chat/call-tool", "POST", request_data)

        result = TestResult(
            server_name=server_name,
            tool_name=tool_name,
            success=error is None and response is not None,
            duration_ms=duration,
            request=request_data,
            response=response if not error else None,
            error=error
        )

        status = "✅" if result.success else "❌"
        self.log(f"  {status} {duration:.1f}ms - {error or 'Success'}")

        return result

    def get_server_tools(self, server_name: str) -> list[str]:
        """Get list of tools for a specific server via retrieve_tools"""
        # Try to get tools via the retrieve_tools endpoint
        request_data = {
            "server_name": server_name,
            "tool_name": "list_tools",
            "arguments": {}
        }

        # This might not work directly, so we'll use the config tool count
        return []

    def test_mcpproxy_tools(self) -> list[TestResult]:
        """Test MCPProxy's own management tools"""
        self.log("\n" + "="*60)
        self.log("Testing MCPProxy Management Tools")
        self.log("="*60)

        # MCPProxy management tools that don't require upstream servers
        management_tools = [
            ("MCPProxy", "list_registries", {}),
            ("MCPProxy", "list_available_groups", {}),
            ("MCPProxy", "groups", {"operation": "list_groups"}),
            ("MCPProxy", "upstream_servers", {"operation": "list"}),
            ("MCPProxy", "quarantine_security", {"operation": "list_quarantined"}),
            ("MCPProxy", "retrieve_tools", {"query": "list", "limit": 5}),
        ]

        results = []
        for server, tool, args in management_tools:
            # Use the MCP endpoint format
            request_data = {
                "server_name": server,
                "tool_name": tool,
                "arguments": args
            }

            self.log(f"Testing MCPProxy:{tool}...")
            response, duration, error = self.http_request("/chat/call-tool", "POST", request_data)

            result = TestResult(
                server_name=server,
                tool_name=tool,
                success=error is None and response is not None,
                duration_ms=duration,
                request=request_data,
                response=response if not error else None,
                error=error
            )

            status = "✅" if result.success else "❌"
            self.log(f"  {status} {duration:.1f}ms - {error or 'Success'}")
            results.append(result)
            self.results.append(result)

        return results

    def test_upstream_servers(self, servers: list[dict], max_servers: int = 10) -> list[TestResult]:
        """Test tools from upstream MCP servers"""
        self.log("\n" + "="*60)
        self.log(f"Testing Upstream Server Tools (max {max_servers} servers)")
        self.log("="*60)

        # Sample tools to test for different server types
        test_tools = {
            "context7": ("resolve-library-id", {"libraryName": "react"}),
            "github": ("get_me", {}),
            "docker": ("docker_list_containers", {}),
            "postgres": ("query", {"sql": "SELECT 1"}),
            "archon": ("find_projects", {}),
            "gmail": ("list_labels", {}),
            "playwright": ("browser_navigate", {"url": "about:blank"}),
            "supabase": ("list_tables", {}),
            "neo4j": ("read_query", {"query": "RETURN 1"}),
            "qdrant": ("list_collections", {}),
            "exa": ("search", {"query": "test"}),
            "reddit": ("get_subreddit", {"subreddit": "programming"}),
            "confluence": ("search", {"query": "test"}),
            "taskmaster": ("list_tasks", {}),
            "basic-memory": ("list_memories", {}),
        }

        # Default test tool for unknown servers
        default_tools = [
            ("list", {}),
            ("get_status", {}),
            ("health", {}),
            ("ping", {}),
        ]

        results = []
        tested = 0

        for server in servers:
            if tested >= max_servers:
                break

            server_name = server.get('name', '')
            if not server_name:
                continue

            # Find appropriate tool to test
            if server_name in test_tools:
                tool_name, args = test_tools[server_name]
            else:
                # Try first available tool or default
                tool_name, args = default_tools[0]

            result = self.call_tool(server_name, tool_name, args)
            results.append(result)
            self.results.append(result)
            tested += 1

        return results

    def write_detailed_log(self):
        """Write detailed results to log file"""
        self.log("\n" + "="*60)
        self.log("DETAILED REQUEST/RESPONSE LOG")
        self.log("="*60)

        for i, result in enumerate(self.results, 1):
            self.log(f"\n--- Test #{i}: {result.server_name}:{result.tool_name} ---")
            self.log(f"Timestamp: {result.timestamp}")
            self.log(f"Duration: {result.duration_ms:.1f}ms")
            self.log(f"Success: {result.success}")
            self.log(f"Request: {json.dumps(result.request, indent=2)}")

            if result.response:
                response_str = json.dumps(result.response, indent=2) if isinstance(result.response, (dict, list)) else str(result.response)
                # Truncate long responses
                if len(response_str) > 1000:
                    response_str = response_str[:1000] + "\n... (truncated)"
                self.log(f"Response: {response_str}")

            if result.error:
                self.log(f"Error: {result.error}")

    def write_summary(self):
        """Write summary statistics"""
        self.log("\n" + "="*60)
        self.log("TEST SUMMARY")
        self.log("="*60)

        total = len(self.results)
        successful = sum(1 for r in self.results if r.success)
        failed = total - successful

        self.log(f"\nTotal Tests: {total}")
        self.log(f"Successful:  {successful} ({100*successful/total:.1f}%)" if total > 0 else "Successful: 0")
        self.log(f"Failed:      {failed} ({100*failed/total:.1f}%)" if total > 0 else "Failed: 0")

        if self.results:
            durations = [r.duration_ms for r in self.results]
            self.log(f"\nTiming Statistics:")
            self.log(f"  Min:     {min(durations):.1f}ms")
            self.log(f"  Max:     {max(durations):.1f}ms")
            self.log(f"  Average: {sum(durations)/len(durations):.1f}ms")
            self.log(f"  Total:   {sum(durations):.1f}ms")

        # Results by server
        self.log(f"\nResults by Server:")
        self.log(f"{'Server':<30} {'Tool':<30} {'Status':<10} {'Duration':<12}")
        self.log("-" * 85)

        for result in sorted(self.results, key=lambda r: r.duration_ms, reverse=True):
            status = "✅ OK" if result.success else "❌ FAIL"
            self.log(f"{result.server_name:<30} {result.tool_name:<30} {status:<10} {result.duration_ms:>8.1f}ms")

        # Timeout analysis
        timeouts = [r for r in self.results if r.error and "Timeout" in r.error]
        if timeouts:
            self.log(f"\n⚠️  Timeouts: {len(timeouts)} requests exceeded {self.timeout}s timeout")
            for r in timeouts:
                self.log(f"  - {r.server_name}:{r.tool_name}")

    def run(self, max_upstream_servers: int = 10):
        """Run all tests"""
        # Ensure log directory exists
        log_dir = os.path.dirname(self.log_file)
        if log_dir and not os.path.exists(log_dir):
            os.makedirs(log_dir)

        with open(self.log_file, 'w') as f:
            self.log_handle = f

            self.log("="*60)
            self.log("MCP PROXY TOOL TESTER")
            self.log("="*60)
            self.log(f"Proxy URL: {self.proxy_url}")
            self.log(f"Timeout: {self.timeout}s")
            self.log(f"Log File: {self.log_file}")
            self.log(f"Started: {datetime.now().isoformat()}")

            # Get configuration
            config = self.get_config()
            if not config:
                self.log("Failed to get configuration, aborting", "ERROR")
                return

            # Get active servers
            servers = self.get_servers(config)

            # Test MCPProxy management tools
            self.test_mcpproxy_tools()

            # Test upstream server tools
            self.test_upstream_servers(servers, max_upstream_servers)

            # Write detailed log
            self.write_detailed_log()

            # Write summary
            self.write_summary()

            self.log(f"\nCompleted: {datetime.now().isoformat()}")
            self.log(f"Log saved to: {self.log_file}")

        self.log_handle = None


def main():
    parser = argparse.ArgumentParser(
        description="Test MCP Proxy tools with timeout handling",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
    python scripts/mcp-tool-tester.py
    python scripts/mcp-tool-tester.py --timeout 5 --max-servers 5
    python scripts/mcp-tool-tester.py --output logs/test-$(date +%Y%m%d).log
        """
    )
    parser.add_argument(
        "--url",
        default=DEFAULT_PROXY_URL,
        help=f"MCPProxy URL (default: {DEFAULT_PROXY_URL})"
    )
    parser.add_argument(
        "--timeout", "-t",
        type=int,
        default=DEFAULT_TIMEOUT,
        help=f"Request timeout in seconds (default: {DEFAULT_TIMEOUT})"
    )
    parser.add_argument(
        "--output", "-o",
        default=DEFAULT_LOG_FILE,
        help=f"Output log file (default: {DEFAULT_LOG_FILE})"
    )
    parser.add_argument(
        "--max-servers", "-m",
        type=int,
        default=10,
        help="Maximum number of upstream servers to test (default: 10)"
    )

    args = parser.parse_args()

    tester = MCPProxyTester(
        proxy_url=args.url,
        timeout=args.timeout,
        log_file=args.output
    )

    tester.run(max_upstream_servers=args.max_servers)


if __name__ == "__main__":
    main()

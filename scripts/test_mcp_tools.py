#!/usr/bin/env python3
"""
MCP Tools Timeout Tester
Tests all mcpproxy tools with a configurable timeout.
Reports which tools respond, timeout, or fail.
"""

import asyncio
import json
import sys
import time
from datetime import datetime
from typing import Any

# Configuration
MCP_SERVER_URL = "http://localhost:8080"
TIMEOUT_SECONDS = 60
STDIO_MODE = True  # Use stdio transport to mcpproxy


class MCPClient:
    """Simple MCP client using stdio transport"""

    def __init__(self):
        self.request_id = 0
        self.process = None
        self.reader = None
        self.writer = None

    async def connect_stdio(self, command: list[str]):
        """Connect to MCP server via stdio"""
        self.process = await asyncio.create_subprocess_exec(
            *command,
            stdin=asyncio.subprocess.PIPE,
            stdout=asyncio.subprocess.PIPE,
            stderr=asyncio.subprocess.PIPE
        )
        self.reader = self.process.stdout
        self.writer = self.process.stdin

        # Initialize
        await self.send_request("initialize", {
            "protocolVersion": "2024-11-05",
            "capabilities": {},
            "clientInfo": {"name": "mcp-tool-tester", "version": "1.0.0"}
        })

        # Send initialized notification
        await self.send_notification("notifications/initialized", {})

    async def connect_http(self, url: str):
        """Connect to MCP server via HTTP/SSE"""
        import aiohttp
        self.session = aiohttp.ClientSession()
        self.url = url

    async def send_notification(self, method: str, params: dict):
        """Send a notification (no response expected)"""
        message = {
            "jsonrpc": "2.0",
            "method": method,
            "params": params
        }
        data = json.dumps(message) + "\n"
        self.writer.write(data.encode())
        await self.writer.drain()

    async def send_request(self, method: str, params: dict, timeout: float = TIMEOUT_SECONDS) -> dict:
        """Send request and wait for response with timeout"""
        self.request_id += 1
        request = {
            "jsonrpc": "2.0",
            "id": self.request_id,
            "method": method,
            "params": params
        }

        data = json.dumps(request) + "\n"
        self.writer.write(data.encode())
        await self.writer.drain()

        # Wait for response with timeout
        try:
            response_line = await asyncio.wait_for(
                self.reader.readline(),
                timeout=timeout
            )
            if response_line:
                return json.loads(response_line.decode())
            return {"error": {"message": "Empty response"}}
        except asyncio.TimeoutError:
            return {"error": {"message": f"TIMEOUT after {timeout}s"}}
        except Exception as e:
            return {"error": {"message": str(e)}}

    async def list_tools(self) -> list[dict]:
        """Get list of available tools"""
        response = await self.send_request("tools/list", {})
        if "result" in response and "tools" in response["result"]:
            return response["result"]["tools"]
        return []

    async def call_tool(self, name: str, arguments: dict, timeout: float = TIMEOUT_SECONDS) -> dict:
        """Call a tool with timeout"""
        return await self.send_request("tools/call", {
            "name": name,
            "arguments": arguments
        }, timeout=timeout)

    async def close(self):
        """Close connection"""
        if self.process:
            self.process.terminate()
            await self.process.wait()


def get_test_arguments(tool_name: str) -> dict:
    """Get safe test arguments for each tool"""
    test_args = {
        # Server management tools
        "upstream_servers": {"operation": "list"},
        "quarantine_security": {"operation": "list_quarantined"},
        "groups": {"operation": "list_groups"},
        "list_available_groups": {},
        "retrieve_tools": {"query": "test", "limit": 5},
        "list_registries": {},
        "search_servers": {"registry": "smithery", "limit": 3},
        "startup_script": {"operation": "status"},
        "read_cache": {"key": "nonexistent", "limit": 10},
    }
    return test_args.get(tool_name, {})


def should_skip_tool(tool_name: str) -> bool:
    """Check if tool should be skipped (destructive or requires setup)"""
    skip_patterns = [
        "call_tool",  # Meta-tool, needs specific target
    ]
    return any(pattern in tool_name for pattern in skip_patterns)


async def test_all_tools():
    """Main test function"""
    print("=" * 70)
    print(f"MCP Tools Timeout Tester - {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}")
    print(f"Timeout: {TIMEOUT_SECONDS}s per tool")
    print("=" * 70)
    print()

    client = MCPClient()

    # Connect to mcpproxy
    print("[*] Connecting to mcpproxy via stdio...")
    mcpproxy_path = "/Users/hrannow/Library/CloudStorage/OneDrive-Persönlich/workspace/mcp-server/mcpproxy-go/mcpproxy"
    config_path = "/Users/hrannow/Library/CloudStorage/OneDrive-Persönlich/workspace/mcp-server/mcpproxy-go/config-link/mcp_config_tested.json"

    try:
        await client.connect_stdio([mcpproxy_path, "mcp", "-c", config_path])
        print("[+] Connected successfully")
    except Exception as e:
        print(f"[-] Failed to connect: {e}")
        return

    print()
    print("[*] Fetching tool list...")
    tools = await client.list_tools()

    if not tools:
        print("[-] No tools found or failed to fetch tools")
        await client.close()
        return

    # Filter to only mcpproxy tools
    mcpproxy_tools = [t for t in tools if t["name"].startswith(("upstream_servers", "quarantine", "groups", "list_", "retrieve_tools", "search_servers", "startup_script", "read_cache", "call_tool"))]

    print(f"[+] Found {len(tools)} total tools, {len(mcpproxy_tools)} mcpproxy management tools")
    print()

    # Test results
    results = {
        "success": [],
        "timeout": [],
        "error": [],
        "skipped": []
    }

    print("-" * 70)
    print(f"{'Tool Name':<40} {'Status':<15} {'Time':<10}")
    print("-" * 70)

    for tool in mcpproxy_tools:
        tool_name = tool["name"]

        if should_skip_tool(tool_name):
            results["skipped"].append(tool_name)
            print(f"{tool_name:<40} {'SKIPPED':<15} {'-':<10}")
            continue

        args = get_test_arguments(tool_name)

        start_time = time.time()
        response = await client.call_tool(tool_name, args, timeout=TIMEOUT_SECONDS)
        elapsed = time.time() - start_time

        if "error" in response:
            error_msg = response["error"].get("message", str(response["error"]))
            if "TIMEOUT" in error_msg:
                results["timeout"].append(tool_name)
                status = "TIMEOUT"
            else:
                results["error"].append((tool_name, error_msg[:50]))
                status = "ERROR"
        else:
            results["success"].append(tool_name)
            status = "OK"

        print(f"{tool_name:<40} {status:<15} {elapsed:.2f}s")

    print("-" * 70)
    print()

    # Summary
    print("=" * 70)
    print("SUMMARY")
    print("=" * 70)
    print(f"  Success:  {len(results['success'])}")
    print(f"  Timeout:  {len(results['timeout'])}")
    print(f"  Error:    {len(results['error'])}")
    print(f"  Skipped:  {len(results['skipped'])}")
    print()

    if results["timeout"]:
        print("TIMEOUT Tools (potential bugs):")
        for tool in results["timeout"]:
            print(f"  - {tool}")
        print()

    if results["error"]:
        print("ERROR Tools:")
        for tool, error in results["error"]:
            print(f"  - {tool}: {error}")
        print()

    await client.close()
    print("[*] Test completed")


if __name__ == "__main__":
    asyncio.run(test_all_tools())

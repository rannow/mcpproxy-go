#!/usr/bin/env python3
"""
MCP Tools HTTP Tester
Tests mcpproxy tools via HTTP API with timeout monitoring.
"""

import json
import requests
import time
from datetime import datetime
from concurrent.futures import ThreadPoolExecutor, TimeoutError as FuturesTimeoutError

# Configuration
BASE_URL = "http://localhost:8080"
TIMEOUT_SECONDS = 60

def call_mcp_tool(tool_name: str, arguments: dict) -> tuple[str, float, dict]:
    """Call MCP tool via HTTP and return status, time, response"""
    url = f"{BASE_URL}/mcp"

    request_body = {
        "jsonrpc": "2.0",
        "id": int(time.time() * 1000),
        "method": "tools/call",
        "params": {
            "name": tool_name,
            "arguments": arguments
        }
    }

    start_time = time.time()
    try:
        response = requests.post(
            url,
            json=request_body,
            headers={"Content-Type": "application/json"},
            timeout=TIMEOUT_SECONDS
        )
        elapsed = time.time() - start_time

        if response.status_code == 200:
            data = response.json()
            if "error" in data:
                return "ERROR", elapsed, data["error"]
            return "OK", elapsed, data.get("result", {})
        else:
            return "HTTP_ERROR", elapsed, {"status": response.status_code, "text": response.text[:200]}

    except requests.exceptions.Timeout:
        return "TIMEOUT", TIMEOUT_SECONDS, {}
    except requests.exceptions.ConnectionError as e:
        return "CONN_ERROR", time.time() - start_time, {"error": str(e)[:100]}
    except Exception as e:
        return "EXCEPTION", time.time() - start_time, {"error": str(e)[:100]}


def list_tools_via_http() -> list[dict]:
    """Get tool list via HTTP"""
    url = f"{BASE_URL}/mcp"
    request_body = {
        "jsonrpc": "2.0",
        "id": 1,
        "method": "tools/list",
        "params": {}
    }

    try:
        response = requests.post(
            url,
            json=request_body,
            headers={"Content-Type": "application/json"},
            timeout=30
        )
        if response.status_code == 200:
            data = response.json()
            return data.get("result", {}).get("tools", [])
    except Exception as e:
        print(f"Failed to list tools: {e}")
    return []


def get_test_arguments(tool_name: str) -> dict:
    """Get safe test arguments for each tool"""
    test_args = {
        # Server management - READ operations
        "upstream_servers": {"operation": "list"},
        "quarantine_security": {"operation": "list_quarantined"},
        "groups": {"operation": "list_groups"},
        "list_available_groups": {},
        "retrieve_tools": {"query": "calculator", "limit": 3},
        "list_registries": {},
        "startup_script": {"operation": "status"},

        # Server management - WRITE operations (test with dummy data)
        "upstream_servers_add": {"operation": "add", "name": "test-timeout-check", "command": "echo", "args_json": '["hello"]', "protocol": "stdio"},
        "upstream_servers_remove": {"operation": "remove", "name": "test-timeout-check"},
    }
    return test_args.get(tool_name, {})


# Define specific test cases for upstream_servers operations
TEST_CASES = [
    ("upstream_servers", {"operation": "list"}, "List servers"),
    ("upstream_servers", {"operation": "add", "name": "test-tool-tester", "command": "echo", "args_json": '["test"]', "protocol": "stdio"}, "Add server"),
    ("upstream_servers", {"operation": "remove", "name": "test-tool-tester"}, "Remove server"),
    ("quarantine_security", {"operation": "list_quarantined"}, "List quarantined"),
    ("groups", {"operation": "list_groups"}, "List groups"),
    ("list_available_groups", {}, "Available groups"),
    ("retrieve_tools", {"query": "calculator", "limit": 3}, "Search tools"),
    ("list_registries", {}, "List registries"),
    ("startup_script", {"operation": "status"}, "Startup status"),
]


def main():
    print("=" * 80)
    print(f"MCP Tools HTTP Tester - {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}")
    print(f"Server: {BASE_URL}")
    print(f"Timeout: {TIMEOUT_SECONDS}s per request")
    print("=" * 80)
    print()

    # Check server health first
    print("[*] Checking server connectivity...")
    try:
        resp = requests.get(f"{BASE_URL}/api/servers", timeout=5)
        if resp.status_code == 200:
            servers = resp.json().get("servers", [])
            print(f"[+] Server is up, {len(servers)} upstream servers configured")
        else:
            print(f"[-] Server returned {resp.status_code}")
    except Exception as e:
        print(f"[-] Cannot connect to server: {e}")
        return

    print()
    print("-" * 80)
    print(f"{'Test Case':<35} {'Tool':<25} {'Status':<12} {'Time':<8}")
    print("-" * 80)

    results = {"OK": 0, "TIMEOUT": 0, "ERROR": 0, "CONN_ERROR": 0}
    failed_tests = []

    for tool_name, args, description in TEST_CASES:
        status, elapsed, response = call_mcp_tool(tool_name, args)
        results[status] = results.get(status, 0) + 1

        time_str = f"{elapsed:.2f}s" if elapsed < TIMEOUT_SECONDS else f">{TIMEOUT_SECONDS}s"
        print(f"{description:<35} {tool_name:<25} {status:<12} {time_str:<8}")

        if status != "OK":
            error_detail = ""
            if isinstance(response, dict):
                error_detail = response.get("message", response.get("error", str(response)))[:60]
            failed_tests.append((description, tool_name, status, error_detail))

    print("-" * 80)
    print()

    # Summary
    print("=" * 80)
    print("SUMMARY")
    print("=" * 80)
    total = sum(results.values())
    print(f"  Total tests:     {total}")
    print(f"  Success (OK):    {results.get('OK', 0)}")
    print(f"  Timeout:         {results.get('TIMEOUT', 0)}")
    print(f"  Errors:          {results.get('ERROR', 0)}")
    print(f"  Conn Errors:     {results.get('CONN_ERROR', 0)}")
    print()

    if failed_tests:
        print("FAILED TESTS (potential bugs):")
        print("-" * 80)
        for desc, tool, status, error in failed_tests:
            print(f"  [{status}] {desc}")
            print(f"          Tool: {tool}")
            if error:
                print(f"          Error: {error}")
            print()

    print("[*] Test completed")


if __name__ == "__main__":
    main()

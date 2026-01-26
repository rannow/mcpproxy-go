#!/usr/bin/env python3
"""
MCP Tools HTTP Tester - Extended Version
Tests mcpproxy tools via HTTP API with:
- Full tool listing from MCP Proxy response
- All operations for all tools
- Detailed request/response logging
"""

import json
import requests
import time
from datetime import datetime
from typing import Any

# Configuration
BASE_URL = "http://localhost:8080"
TIMEOUT_SECONDS = 60
VERBOSE = True  # Show request/response details

def log_request(method: str, url: str, body: dict):
    """Log the HTTP request"""
    if VERBOSE:
        print(f"\n  📤 REQUEST: {method} {url}")
        print(f"     Body: {json.dumps(body, indent=2, ensure_ascii=False)[:500]}")

def log_response(status_code: int, elapsed: float, data: Any):
    """Log the HTTP response"""
    if VERBOSE:
        print(f"  📥 RESPONSE: {status_code} ({elapsed*1000:.1f}ms)")
        response_str = json.dumps(data, indent=2, ensure_ascii=False) if isinstance(data, (dict, list)) else str(data)
        if len(response_str) > 800:
            response_str = response_str[:800] + "...[truncated]"
        print(f"     Data: {response_str}")

def call_chat_api(endpoint: str, body: dict = None) -> tuple[str, float, dict]:
    """Call Chat API endpoint and return status, time, response"""
    url = f"{BASE_URL}{endpoint}"
    request_body = body or {}

    log_request("POST", url, request_body)
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
            log_response(response.status_code, elapsed, data)
            if "error" in data:
                return "ERROR", elapsed, data
            return "OK", elapsed, data
        else:
            log_response(response.status_code, elapsed, response.text[:200])
            return "HTTP_ERROR", elapsed, {"status": response.status_code, "text": response.text[:200]}

    except requests.exceptions.Timeout:
        print(f"  ⏱️ TIMEOUT after {TIMEOUT_SECONDS}s")
        return "TIMEOUT", TIMEOUT_SECONDS, {}
    except requests.exceptions.ConnectionError as e:
        print(f"  ❌ CONNECTION ERROR: {str(e)[:100]}")
        return "CONN_ERROR", time.time() - start_time, {"error": str(e)[:100]}
    except Exception as e:
        print(f"  ❌ EXCEPTION: {str(e)[:100]}")
        return "EXCEPTION", time.time() - start_time, {"error": str(e)[:100]}


def call_mcp_tool(tool_name: str, arguments: dict, server_name: str = "MCPProxy") -> tuple[str, float, dict]:
    """Call MCP tool via Chat API and return status, time, response"""
    body = {
        "tool_name": tool_name,
        "arguments": arguments,
        "server_name": server_name
    }
    status, elapsed, result = call_chat_api("/chat/call-tool", body)
    if status == "OK":
        # Extract tool result from chat API response
        return status, elapsed, result.get("result", result)
    return status, elapsed, result


def list_tools_via_http() -> tuple[list[dict], dict]:
    """Get tool list via Chat API with full schema.
    Returns (tools_list, servers_info) where servers_info contains status per server.
    """
    status, elapsed, result = call_chat_api("/chat/list-all-tools", {})
    if status == "OK":
        content = result.get("content", "{}")
        try:
            servers_data = json.loads(content)
            all_tools = []
            servers_info = {}

            for server_name, server_info in servers_data.items():
                server_status = server_info.get("status", "unknown")
                servers_info[server_name] = {
                    "status": server_status,
                    "reason": server_info.get("reason", ""),
                    "tool_count": len(server_info.get("tools", []))
                }

                # Only include tools from available servers
                if server_status == "available":
                    tools = server_info.get("tools", [])
                    # Add server name to each tool for context
                    for tool in tools:
                        tool["_server"] = server_name
                    all_tools.extend(tools)

            return all_tools, servers_info
        except json.JSONDecodeError as e:
            print(f"  ❌ JSON parse error: {e}")
            return [], {}
    return [], {}


def get_tool_operations(tool: dict) -> list[tuple[str, dict, str, str]]:
    """
    Extract all possible operations/test cases from tool schema.
    Returns list of (tool_name, arguments, description, server_name)
    """
    tool_name = tool.get("name", "")

    # CRITICAL: _server MUST be set by list_tools_via_http()
    # Never default to "MCPProxy" - it doesn't exist as a server
    if "_server" not in tool:
        print(f"  ⚠️ WARNING: Tool '{tool_name}' missing _server field!")
        server_name = "UNKNOWN_SERVER"  # Will fail explicitly rather than silently
    else:
        server_name = tool["_server"]

    schema = tool.get("inputSchema", {})
    properties = schema.get("properties", {})

    operations = []

    # Check if tool has "operation" property (enum-based operations)
    if "operation" in properties:
        op_schema = properties["operation"]
        if "enum" in op_schema:
            for op in op_schema["enum"]:
                args = {"operation": op}
                # Add required arguments for specific operations
                args = enrich_operation_args(tool_name, op, args, properties)
                operations.append((tool_name, args, f"{tool_name}:{op}", server_name))
        else:
            # Single default operation
            operations.append((tool_name, {}, f"{tool_name}:default", server_name))
    else:
        # Tool without operation enum - use default/safe args
        args = get_default_test_args(tool_name, properties)
        operations.append((tool_name, args, f"{tool_name}", server_name))

    return operations


def enrich_operation_args(tool_name: str, operation: str, args: dict, properties: dict) -> dict:
    """Add required/useful arguments for specific operations"""

    # upstream_servers operations
    if tool_name == "upstream_servers":
        if operation == "add":
            args.update({
                "name": "test-api-tester",
                "command": "echo",
                "args_json": '["test"]',
                "protocol": "stdio"
            })
        elif operation == "remove":
            args["name"] = "test-api-tester"
        elif operation == "enable":
            args["name"] = "test-api-tester"
        elif operation == "disable":
            args["name"] = "test-api-tester"
        elif operation == "status":
            args["name"] = "test-api-tester"

    # groups operations
    elif tool_name == "groups":
        if operation in ["add_to_group", "remove_from_group", "set_server_groups"]:
            args["server_name"] = "test-server"
            args["group_name"] = "test-group"
        elif operation in ["create_group", "delete_group"]:
            args["group_name"] = "test-group"

    # quarantine_security operations
    elif tool_name == "quarantine_security":
        if operation == "quarantine":
            args["server_name"] = "test-server"
            args["reason"] = "testing"
        elif operation == "unquarantine":
            args["server_name"] = "test-server"
        elif operation == "is_quarantined":
            args["server_name"] = "test-server"

    # startup_script operations
    elif tool_name == "startup_script":
        if operation == "set_order":
            args["order_json"] = '["server1", "server2"]'
        elif operation == "set_enabled":
            args["enabled"] = True

    # retrieve_tools - add search params
    elif tool_name == "retrieve_tools":
        if "query" not in args:
            args["query"] = "calculator"
            args["limit"] = 3

    return args


def get_default_test_args(tool_name: str, properties: dict) -> dict:
    """Get default safe test arguments for tools without operation enum"""
    defaults = {
        "list_registries": {},
        "list_available_groups": {},
        "retrieve_tools": {"query": "test", "limit": 3},
    }
    return defaults.get(tool_name, {})


def print_tool_schema(tool: dict):
    """Pretty print tool schema"""
    print(f"\n  📋 Tool: {tool.get('name', 'unknown')}")
    print(f"     Description: {tool.get('description', 'N/A')[:100]}")
    schema = tool.get("inputSchema", {})
    properties = schema.get("properties", {})
    required = schema.get("required", [])

    if properties:
        print("     Parameters:")
        for prop_name, prop_schema in properties.items():
            req_marker = "*" if prop_name in required else " "
            prop_type = prop_schema.get("type", "any")
            enum_vals = prop_schema.get("enum", [])
            if enum_vals:
                print(f"       {req_marker} {prop_name}: {prop_type} = {enum_vals}")
            else:
                print(f"       {req_marker} {prop_name}: {prop_type}")


def main():
    print("=" * 90)
    print(f"MCP Tools HTTP Tester (Extended) - {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}")
    print(f"Server: {BASE_URL}")
    print(f"Timeout: {TIMEOUT_SECONDS}s per request")
    print("=" * 90)

    # Check server health first
    print("\n[1] Checking server connectivity...")
    try:
        resp = requests.get(f"{BASE_URL}/api/servers", timeout=5)
        if resp.status_code == 200:
            servers = resp.json().get("servers", [])
            print(f"    ✅ Server is up, {len(servers)} upstream servers configured")
        else:
            print(f"    ❌ Server returned {resp.status_code}")
            return
    except Exception as e:
        print(f"    ❌ Cannot connect to server: {e}")
        return

    # PHASE 1: Get and display all tools from MCP Proxy
    print("\n" + "=" * 90)
    print("[2] LISTING ALL TOOLS FROM MCP PROXY")
    print("=" * 90)

    tools, servers_info = list_tools_via_http()

    # Display server status summary
    if servers_info:
        available = sum(1 for s in servers_info.values() if s["status"] == "available")
        unavailable = len(servers_info) - available
        print(f"\n    📊 Server Status: {available} available, {unavailable} unavailable")
        if unavailable > 0:
            print("    Unavailable servers:")
            for name, info in servers_info.items():
                if info["status"] != "available":
                    print(f"      - {name}: {info['status']} ({info.get('reason', 'unknown')})")

    if not tools:
        print("    ❌ No tools returned from MCP Proxy")
        return

    print(f"\n    📦 Found {len(tools)} tools:")

    for tool in tools:
        print_tool_schema(tool)

    # PHASE 2: Generate all test operations
    print("\n" + "=" * 90)
    print("[3] GENERATING TEST OPERATIONS FOR ALL TOOLS")
    print("=" * 90)

    all_operations = []
    for tool in tools:
        ops = get_tool_operations(tool)
        all_operations.extend(ops)
        print(f"    {tool.get('name', 'unknown')}: {len(ops)} operations")

    print(f"\n    Total: {len(all_operations)} test operations")

    # PHASE 3: Execute all operations
    print("\n" + "=" * 90)
    print("[4] EXECUTING ALL TOOL OPERATIONS")
    print("=" * 90)

    results = {"OK": 0, "TIMEOUT": 0, "ERROR": 0, "CONN_ERROR": 0, "HTTP_ERROR": 0, "EXCEPTION": 0}
    failed_tests = []
    successful_tests = []

    for i, (tool_name, args, description, server_name) in enumerate(all_operations, 1):
        print(f"\n{'─' * 90}")
        print(f"[{i}/{len(all_operations)}] Testing: {description} (server: {server_name})")
        print(f"{'─' * 90}")

        status, elapsed, response = call_mcp_tool(tool_name, args, server_name)
        results[status] = results.get(status, 0) + 1

        time_str = f"{elapsed*1000:.1f}ms" if elapsed < TIMEOUT_SECONDS else f">{TIMEOUT_SECONDS}s"

        if status == "OK":
            print(f"    ✅ SUCCESS ({time_str})")
            successful_tests.append((description, time_str))
        else:
            print(f"    ❌ {status} ({time_str})")
            error_detail = ""
            if isinstance(response, dict):
                error_detail = response.get("message", response.get("error", str(response)))[:80]
            failed_tests.append((description, tool_name, status, error_detail, args))

    # PHASE 4: Summary
    print("\n" + "=" * 90)
    print("[5] TEST SUMMARY")
    print("=" * 90)

    total = sum(results.values())
    success_rate = (results.get('OK', 0) / total * 100) if total > 0 else 0

    print(f"""
    Total tests:       {total}
    ✅ Success (OK):   {results.get('OK', 0)} ({success_rate:.1f}%)
    ⏱️ Timeout:        {results.get('TIMEOUT', 0)}
    ❌ Errors:         {results.get('ERROR', 0)}
    🔌 Conn Errors:    {results.get('CONN_ERROR', 0)}
    🌐 HTTP Errors:    {results.get('HTTP_ERROR', 0)}
    💥 Exceptions:     {results.get('EXCEPTION', 0)}
    """)

    if successful_tests:
        print("    SUCCESSFUL TESTS:")
        print("    " + "-" * 50)
        for desc, time_str in successful_tests:
            print(f"      ✅ {desc} ({time_str})")

    if failed_tests:
        print("\n    FAILED TESTS (potential bugs):")
        print("    " + "-" * 50)
        for desc, tool, status, error, args in failed_tests:
            print(f"      ❌ [{status}] {desc}")
            print(f"         Tool: {tool}")
            print(f"         Args: {json.dumps(args, ensure_ascii=False)[:100]}")
            if error:
                print(f"         Error: {error}")

    print("\n" + "=" * 90)
    print(f"[*] Test completed at {datetime.now().strftime('%H:%M:%S')}")
    print("=" * 90)


if __name__ == "__main__":
    main()

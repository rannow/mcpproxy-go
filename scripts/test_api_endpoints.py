#!/usr/bin/env python3
"""
MCP Proxy API Endpoint Tester
Comprehensive test for all HTTP API endpoints with request/response logging.
"""

import json
import requests
import time
from datetime import datetime
from typing import Any, Optional
from dataclasses import dataclass
from enum import Enum

# Configuration
BASE_URL = "http://localhost:8080"
TIMEOUT_SECONDS = 30
VERBOSE = True

class HttpMethod(Enum):
    GET = "GET"
    POST = "POST"
    PUT = "PUT"
    DELETE = "DELETE"

@dataclass
class ApiEndpoint:
    path: str
    method: HttpMethod
    description: str
    body: Optional[dict] = None
    query_params: Optional[dict] = None
    category: str = "misc"
    expected_status: int = 200

# All API endpoints discovered from server.go
# NOTE: MCP protocol endpoints (/mcp, /v1/tool_code, /v1/tool-code) require proper
# MCP client initialization (Start + Initialize with protocol version) and cannot
# be tested with simple HTTP POST. Use test_mcp_tools_http.py for MCP protocol testing.
API_ENDPOINTS = [

    # ============================================
    # Chat API Endpoints (for AI assistants)
    # ============================================
    ApiEndpoint("/chat/read-config", HttpMethod.POST, "Read MCP Proxy config",
                body={}, category="chat"),
    ApiEndpoint("/chat/call-tool", HttpMethod.POST, "Call tool via chat API",
                {"tool_name": "list_registries", "arguments": {}, "server_name": "MCPProxy"},
                category="chat"),
    ApiEndpoint("/chat/read-log", HttpMethod.POST, "Read log file",
                body={"lines": 50}, category="chat"),
    ApiEndpoint("/chat/get-server-status", HttpMethod.POST, "Get all server status",
                body={}, category="chat"),
    ApiEndpoint("/chat/list-all-servers", HttpMethod.POST, "List all configured servers",
                body={}, category="chat"),
    ApiEndpoint("/chat/list-all-tools", HttpMethod.POST, "List all available tools",
                body={}, category="chat"),
    ApiEndpoint("/chat/context", HttpMethod.GET, "Get context for AI assistant",
                category="chat"),

    # ============================================
    # REST API Endpoints
    # ============================================
    ApiEndpoint("/api/servers", HttpMethod.GET, "List servers (REST API)",
                category="api"),
    ApiEndpoint("/api/servers/status", HttpMethod.GET, "Servers status overview",
                category="api"),
    ApiEndpoint("/api/groups", HttpMethod.GET, "List groups",
                category="api"),
    ApiEndpoint("/api/metrics/current", HttpMethod.GET, "Current metrics",
                category="api"),
    ApiEndpoint("/api/resources/current", HttpMethod.GET, "Current resources",
                category="api"),
    ApiEndpoint("/api/memory", HttpMethod.GET, "Memory usage",
                category="api"),
    ApiEndpoint("/api/tray/status", HttpMethod.GET, "Tray status",
                category="api"),
    ApiEndpoint("/api/assignments", HttpMethod.GET, "Server assignments",
                category="api"),

    # ============================================
    # Agent API Endpoints
    # ============================================
    ApiEndpoint("/api/v1/agent/servers", HttpMethod.GET, "Agent: List servers",
                category="agent"),
    ApiEndpoint("/api/v1/agent/logs/main", HttpMethod.GET, "Agent: Main logs",
                query_params={"lines": "20"},
                category="agent"),
    ApiEndpoint("/api/v1/agent/registries/search", HttpMethod.GET, "Agent: Search registries",
                query_params={"query": "calculator", "limit": "3"},
                category="agent"),

]


def log_separator(char: str = "─", width: int = 90):
    print(char * width)


def call_api(endpoint: ApiEndpoint) -> tuple[str, float, int, Any]:
    """
    Call API endpoint and return (status, elapsed_time, http_code, response_data)
    """
    url = f"{BASE_URL}{endpoint.path}"
    if endpoint.query_params:
        url += "?" + "&".join(f"{k}={v}" for k, v in endpoint.query_params.items())

    start_time = time.time()

    try:
        # Log request
        if VERBOSE:
            print(f"\n  📤 REQUEST: {endpoint.method.value} {url}")
            if endpoint.body:
                body_str = json.dumps(endpoint.body, indent=2, ensure_ascii=False)
                if len(body_str) > 300:
                    body_str = body_str[:300] + "...[truncated]"
                print(f"     Body: {body_str}")

        # Make request
        if endpoint.method == HttpMethod.GET:
            response = requests.get(url, timeout=TIMEOUT_SECONDS)
        elif endpoint.method == HttpMethod.POST:
            response = requests.post(url, json=endpoint.body,
                                    headers={"Content-Type": "application/json"},
                                    timeout=TIMEOUT_SECONDS)
        elif endpoint.method == HttpMethod.PUT:
            response = requests.put(url, json=endpoint.body,
                                   headers={"Content-Type": "application/json"},
                                   timeout=TIMEOUT_SECONDS)
        elif endpoint.method == HttpMethod.DELETE:
            response = requests.delete(url, timeout=TIMEOUT_SECONDS)
        else:
            return "UNSUPPORTED", 0, 0, {"error": f"Unsupported method: {endpoint.method}"}

        elapsed = time.time() - start_time

        # Parse response
        try:
            data = response.json()
        except:
            data = response.text[:500] if response.text else ""

        # Log response
        if VERBOSE:
            print(f"  📥 RESPONSE: {response.status_code} ({elapsed*1000:.1f}ms)")
            response_str = json.dumps(data, indent=2, ensure_ascii=False) if isinstance(data, (dict, list)) else str(data)
            if len(response_str) > 600:
                response_str = response_str[:600] + "...[truncated]"
            print(f"     Data: {response_str}")

        # Determine status
        if response.status_code == endpoint.expected_status:
            return "OK", elapsed, response.status_code, data
        elif response.status_code < 400:
            return "UNEXPECTED_STATUS", elapsed, response.status_code, data
        else:
            return "HTTP_ERROR", elapsed, response.status_code, data

    except requests.exceptions.Timeout:
        print(f"  ⏱️ TIMEOUT after {TIMEOUT_SECONDS}s")
        return "TIMEOUT", TIMEOUT_SECONDS, 0, {}
    except requests.exceptions.ConnectionError as e:
        print(f"  ❌ CONNECTION ERROR: {str(e)[:100]}")
        return "CONN_ERROR", time.time() - start_time, 0, {"error": str(e)[:100]}
    except Exception as e:
        print(f"  💥 EXCEPTION: {str(e)[:100]}")
        return "EXCEPTION", time.time() - start_time, 0, {"error": str(e)[:100]}


def main():
    print("=" * 90)
    print(f"MCP Proxy API Endpoint Tester - {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}")
    print(f"Server: {BASE_URL}")
    print(f"Timeout: {TIMEOUT_SECONDS}s per request")
    print(f"Total Endpoints: {len(API_ENDPOINTS)}")
    print("=" * 90)

    # Group endpoints by category
    categories = {}
    for ep in API_ENDPOINTS:
        if ep.category not in categories:
            categories[ep.category] = []
        categories[ep.category].append(ep)

    print("\n📋 API Endpoints by Category:")
    for cat, endpoints in sorted(categories.items()):
        print(f"   {cat}: {len(endpoints)} endpoints")

    # Check server connectivity
    print("\n" + "=" * 90)
    print("[1] CHECKING SERVER CONNECTIVITY")
    print("=" * 90)

    try:
        resp = requests.get(f"{BASE_URL}/api/servers", timeout=5)
        if resp.status_code == 200:
            print(f"    ✅ Server is up and responding")
        else:
            print(f"    ⚠️ Server returned status {resp.status_code}")
    except Exception as e:
        print(f"    ❌ Cannot connect to server: {e}")
        print("    Please ensure mcpproxy is running on port 8080")
        return

    # Test all endpoints
    results = {"OK": 0, "TIMEOUT": 0, "HTTP_ERROR": 0, "CONN_ERROR": 0,
               "EXCEPTION": 0, "UNEXPECTED_STATUS": 0}
    failed_tests = []
    successful_tests = []

    for category, endpoints in sorted(categories.items()):
        print("\n" + "=" * 90)
        print(f"[{category.upper()}] Testing {len(endpoints)} endpoints")
        print("=" * 90)

        for i, endpoint in enumerate(endpoints, 1):
            log_separator()
            print(f"[{i}/{len(endpoints)}] {endpoint.method.value} {endpoint.path}")
            print(f"    Description: {endpoint.description}")
            log_separator()

            status, elapsed, http_code, response = call_api(endpoint)
            results[status] = results.get(status, 0) + 1

            time_str = f"{elapsed*1000:.1f}ms"

            if status == "OK":
                print(f"    ✅ SUCCESS ({time_str}) HTTP {http_code}")
                successful_tests.append((endpoint.path, endpoint.description, time_str, http_code))
            else:
                print(f"    ❌ {status} ({time_str}) HTTP {http_code}")
                error_detail = ""
                if isinstance(response, dict):
                    error_detail = response.get("error", response.get("message", str(response)))[:80]
                elif isinstance(response, str):
                    error_detail = response[:80]
                failed_tests.append((endpoint.path, endpoint.description, status, http_code, error_detail))

    # Summary
    print("\n" + "=" * 90)
    print("[SUMMARY] API ENDPOINT TEST RESULTS")
    print("=" * 90)

    total = sum(results.values())
    success_rate = (results.get('OK', 0) / total * 100) if total > 0 else 0

    print(f"""
    Total endpoints tested:  {total}
    ✅ Success (OK):         {results.get('OK', 0)} ({success_rate:.1f}%)
    ⏱️ Timeout:              {results.get('TIMEOUT', 0)}
    ❌ HTTP Errors:          {results.get('HTTP_ERROR', 0)}
    🔌 Connection Errors:    {results.get('CONN_ERROR', 0)}
    ⚠️ Unexpected Status:    {results.get('UNEXPECTED_STATUS', 0)}
    💥 Exceptions:           {results.get('EXCEPTION', 0)}
    """)

    if successful_tests:
        print("    SUCCESSFUL ENDPOINTS:")
        print("    " + "-" * 70)
        for path, desc, time_str, code in successful_tests:
            print(f"      ✅ {path:<35} HTTP {code} ({time_str})")

    if failed_tests:
        print("\n    FAILED ENDPOINTS:")
        print("    " + "-" * 70)
        for path, desc, status, code, error in failed_tests:
            print(f"      ❌ {path:<35} [{status}] HTTP {code}")
            print(f"         {desc}")
            if error:
                print(f"         Error: {error}")

    # List all discovered endpoints for reference
    print("\n" + "=" * 90)
    print("[REFERENCE] ALL API ENDPOINTS")
    print("=" * 90)

    for category, endpoints in sorted(categories.items()):
        print(f"\n  [{category.upper()}]")
        for ep in endpoints:
            print(f"    {ep.method.value:<6} {ep.path:<40} - {ep.description[:40]}")

    print("\n" + "=" * 90)
    print(f"[*] Test completed at {datetime.now().strftime('%H:%M:%S')}")
    print("=" * 90)


if __name__ == "__main__":
    main()

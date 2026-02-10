#!/usr/bin/env python3
"""
Test script for MCPProxy upstream_servers patch command.
Tests the patch API endpoint and verifies server configuration updates.
"""

import requests
import json
import time
from datetime import datetime

BASE_URL = "http://localhost:8080"

def log(msg):
    """Print timestamped log message."""
    print(f"[{datetime.now().strftime('%H:%M:%S')}] {msg}")

def call_tool(tool_name: str, arguments: dict) -> dict:
    """Call MCPProxy tool via chat API."""
    payload = {
        "tool_name": tool_name,
        "arguments": arguments,
        "server_name": "MCPProxy"
    }
    try:
        response = requests.post(
            f"{BASE_URL}/chat/call-tool",
            json=payload,
            timeout=30
        )
        return response.json()
    except Exception as e:
        return {"error": str(e)}

def get_server_info(server_name: str) -> dict:
    """Get info for a specific server."""
    result = call_tool("upstream_servers", {"operation": "list"})
    try:
        text = result.get("result", {}).get("content", [{}])[0].get("text", "{}")
        data = json.loads(text)
        servers = data.get("servers", [])
        for server in servers:
            if server.get("name") == server_name:
                return server
    except:
        pass
    return {}

def test_patch_operation(server_name: str, patch_data: dict) -> dict:
    """Test patch operation on a server."""
    log(f"Testing PATCH on '{server_name}' with: {json.dumps(patch_data)}")

    # Get current state
    before = get_server_info(server_name)
    if not before:
        return {"success": False, "error": f"Server '{server_name}' not found"}

    log(f"  Before: startup_mode={before.get('startup_mode')}, args={before.get('args')}")

    # Execute patch
    result = call_tool("upstream_servers", {
        "operation": "patch",
        "name": server_name,
        "patch_json": json.dumps(patch_data)
    })

    # Check result
    if "error" in result:
        return {"success": False, "error": result["error"]}

    # Get after state
    time.sleep(1)  # Wait for update
    after = get_server_info(server_name)
    log(f"  After: startup_mode={after.get('startup_mode')}, args={after.get('args')}")

    # Verify changes
    changes_applied = True
    for key, value in patch_data.items():
        if after.get(key) != value:
            log(f"  ❌ Field '{key}' not updated: expected={value}, got={after.get(key)}")
            changes_applied = False
        else:
            log(f"  ✅ Field '{key}' updated successfully")

    return {
        "success": changes_applied,
        "before": before,
        "after": after,
        "result": result
    }

def test_update_operation(server_name: str, update_fields: dict) -> dict:
    """Test update operation (alternative to patch)."""
    log(f"Testing UPDATE on '{server_name}' with: {json.dumps(update_fields)}")

    # Get current state
    before = get_server_info(server_name)
    if not before:
        return {"success": False, "error": f"Server '{server_name}' not found"}

    log(f"  Before: startup_mode={before.get('startup_mode')}")

    # Build update arguments
    args = {"operation": "update", "name": server_name}
    args.update(update_fields)

    result = call_tool("upstream_servers", args)

    # Check result
    if "error" in result:
        return {"success": False, "error": result["error"]}

    # Get after state
    time.sleep(1)
    after = get_server_info(server_name)
    log(f"  After: startup_mode={after.get('startup_mode')}")

    return {
        "success": True,
        "before": before,
        "after": after,
        "result": result
    }

def main():
    print("=" * 70)
    print("MCPProxy Patch Command Test Script")
    print("=" * 70)

    # Test servers
    test_servers = ["ecs-mcp-server", "mcp-lambda-handler", "cloudwatch-mcp-server"]

    print("\n1. Checking current server states...")
    for server in test_servers:
        info = get_server_info(server)
        if info:
            log(f"  {server}: startup_mode={info.get('startup_mode')}, state={info.get('connection_status', {}).get('state')}")
            log(f"    args={info.get('args')}")
        else:
            log(f"  {server}: NOT FOUND")

    print("\n2. Testing PATCH operation...")
    # Test patch with startup_mode change
    result = test_patch_operation("ecs-mcp-server", {"startup_mode": "active"})
    print(f"   Result: {'SUCCESS' if result.get('success') else 'FAILED'}")
    if not result.get('success'):
        print(f"   Error: {result.get('error')}")
        print(f"   Raw result: {json.dumps(result.get('result', {}), indent=2)[:500]}")

    print("\n3. Testing UPDATE operation (alternative)...")
    result = test_update_operation("mcp-lambda-handler", {"enabled": True})
    print(f"   Result: {'SUCCESS' if result.get('success') else 'FAILED'}")
    if not result.get('success'):
        print(f"   Error: {result.get('error')}")

    print("\n4. Direct API test for patch...")
    # Try direct HTTP endpoint
    try:
        # Check if there's a direct patch endpoint
        response = requests.get(f"{BASE_URL}/api/servers", timeout=5)
        log(f"  /api/servers: {response.status_code}")
    except Exception as e:
        log(f"  /api/servers: {e}")

    try:
        # Try patch via servers endpoint
        response = requests.patch(
            f"{BASE_URL}/api/servers/ecs-mcp-server",
            json={"startup_mode": "active"},
            timeout=5
        )
        log(f"  PATCH /api/servers/ecs-mcp-server: {response.status_code} - {response.text[:200]}")
    except Exception as e:
        log(f"  PATCH failed: {e}")

    print("\n5. Final state check...")
    for server in test_servers:
        info = get_server_info(server)
        if info:
            log(f"  {server}: startup_mode={info.get('startup_mode')}, state={info.get('connection_status', {}).get('state')}")

    print("\n" + "=" * 70)
    print("Test completed")
    print("=" * 70)

if __name__ == "__main__":
    main()

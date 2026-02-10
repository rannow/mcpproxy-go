#!/usr/bin/env python3
"""
Diagnostic script to trace tool routing issue.
Tests whether _server field is correctly propagated.
"""

import json
import requests

BASE_URL = "http://localhost:8080"

def diagnose():
    print("=" * 70)
    print("DIAGNOSTIC: Tool Routing Analysis")
    print("=" * 70)

    # Step 1: Get raw response from list-all-tools
    print("\n[1] Fetching /chat/list-all-tools response...")
    try:
        resp = requests.post(f"{BASE_URL}/chat/list-all-tools", json={}, timeout=30)
        if resp.status_code != 200:
            print(f"    ERROR: HTTP {resp.status_code}")
            return

        data = resp.json()
        content = data.get("content", "{}")
        servers_data = json.loads(content)

        print(f"    Response keys: {list(servers_data.keys())[:10]}...")
        print(f"    Total servers: {len(servers_data)}")

    except Exception as e:
        print(f"    ERROR: {e}")
        return

    # Step 2: Analyze server names in response
    print("\n[2] Analyzing server names in response...")
    available_servers = []
    for server_name, server_info in servers_data.items():
        if server_info.get("status") == "available":
            tool_count = len(server_info.get("tools", []))
            available_servers.append((server_name, tool_count))

    print(f"    Available servers: {len(available_servers)}")
    print("    First 10 available servers with tool counts:")
    for name, count in available_servers[:10]:
        print(f"      - {name}: {count} tools")

    # Step 3: Check if MCPProxy is the only server or one of many
    print("\n[3] Checking MCPProxy presence...")
    if "MCPProxy" in servers_data:
        mcpproxy_tools = servers_data["MCPProxy"].get("tools", [])
        print(f"    MCPProxy has {len(mcpproxy_tools)} tools")
        if mcpproxy_tools:
            print(f"    MCPProxy tool names: {[t.get('name') for t in mcpproxy_tools[:5]]}")
    else:
        print("    MCPProxy NOT found in response keys!")

    # Step 4: Pick a non-MCPProxy tool and test routing
    print("\n[4] Testing tool routing with non-MCPProxy tool...")
    test_tool = None
    test_server = None

    for server_name, server_info in servers_data.items():
        if server_name != "MCPProxy" and server_info.get("status") == "available":
            tools = server_info.get("tools", [])
            if tools:
                test_tool = tools[0]
                test_server = server_name
                break

    if test_tool and test_server:
        tool_name = test_tool.get("name", "unknown")
        print(f"    Selected tool: {tool_name}")
        print(f"    From server: {test_server}")

        # Test calling with correct server_name
        print(f"\n[5] Calling tool with server_name='{test_server}'...")
        try:
            call_resp = requests.post(
                f"{BASE_URL}/chat/call-tool",
                json={
                    "tool_name": tool_name,
                    "arguments": {},
                    "server_name": test_server
                },
                timeout=30
            )
            print(f"    HTTP Status: {call_resp.status_code}")
            call_data = call_resp.json()
            if "error" in call_data:
                print(f"    ERROR: {call_data.get('error', call_data.get('message', 'unknown'))[:100]}")
            else:
                print(f"    SUCCESS: Tool call returned result")
                result_preview = str(call_data)[:200]
                print(f"    Result preview: {result_preview}...")
        except Exception as e:
            print(f"    EXCEPTION: {e}")

        # Test calling with server_name="MCPProxy" (should fail)
        print(f"\n[6] Calling SAME tool with server_name='MCPProxy' (expect failure)...")
        try:
            call_resp = requests.post(
                f"{BASE_URL}/chat/call-tool",
                json={
                    "tool_name": tool_name,
                    "arguments": {},
                    "server_name": "MCPProxy"
                },
                timeout=30
            )
            print(f"    HTTP Status: {call_resp.status_code}")
            call_data = call_resp.json()
            if "error" in call_data:
                print(f"    EXPECTED ERROR: {call_data.get('error', call_data.get('message', 'unknown'))[:100]}")
            else:
                print(f"    UNEXPECTED SUCCESS (routing may be working differently)")
        except Exception as e:
            print(f"    EXCEPTION: {e}")
    else:
        print("    No non-MCPProxy tools found to test!")

    print("\n" + "=" * 70)
    print("DIAGNOSIS COMPLETE")
    print("=" * 70)

if __name__ == "__main__":
    diagnose()

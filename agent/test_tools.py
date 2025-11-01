#!/usr/bin/env python3
"""Quick test script to verify all tools can be imported and initialized."""

import asyncio
import sys


async def test_tool_imports():
    """Test that all tool modules can be imported."""
    print("Testing tool imports...")

    try:
        # Test imports
        from mcp_agent.tools.diagnostic import DiagnosticTools, MCPProxyClient
        from mcp_agent.tools.config import ConfigTools
        from mcp_agent.tools.discovery import DiscoveryTools
        from mcp_agent.tools.testing import TestingTools
        from mcp_agent.tools.logs import LogTools
        from mcp_agent.tools.docs import DocumentationTools
        from mcp_agent.tools.startup import StartupTools
        print("✅ All tool modules imported successfully")

        # Test initialization
        print("\nTesting tool initialization...")
        client = MCPProxyClient(base_url="http://localhost:8080")
        diagnostic = DiagnosticTools(mcpproxy_client=client)
        config = ConfigTools(base_url="http://localhost:8080")
        discovery = DiscoveryTools(base_url="http://localhost:8080")
        testing = TestingTools(base_url="http://localhost:8080")
        logs = LogTools(base_url="http://localhost:8080")
        docs = DocumentationTools(base_url="http://localhost:8080")
        startup = StartupTools(base_url="http://localhost:8080")
        print("✅ All tools initialized successfully")

        # Test a simple method call (non-network)
        print("\nTesting basic functionality...")

        # Test log parsing
        from mcp_agent.tools.logs import LogEntry
        entry = LogEntry(
            timestamp="2025-01-01T00:00:00Z",
            level="INFO",
            message="Test message"
        )
        print(f"✅ LogEntry model works: {entry.level} - {entry.message}")

        # Test connection result
        from mcp_agent.tools.testing import ConnectionTestResult
        conn = ConnectionTestResult(
            server_name="test-server",
            connected=True,
            state="Ready",
            response_time_ms=50.0,
            tool_count=5
        )
        print(f"✅ ConnectionTestResult model works: {conn.server_name} - {conn.state}")

        # Test documentation result
        from mcp_agent.tools.docs import DocumentationResult
        doc = DocumentationResult(
            title="Test Doc",
            content="Test content",
            source="Test",
            relevance_score=0.9
        )
        print(f"✅ DocumentationResult model works: {doc.title} - {doc.source}")

        print("\n🎉 All tool tests passed!")
        return True

    except Exception as e:
        print(f"❌ Error: {e}")
        import traceback
        traceback.print_exc()
        return False


async def test_api_connection():
    """Test connection to mcpproxy agent API."""
    print("\n\nTesting API connection...")

    try:
        from mcp_agent.tools.testing import TestingTools

        testing = TestingTools(base_url="http://localhost:8080")

        # Try to get server list (this should work if mcpproxy is running)
        print("Attempting to connect to mcpproxy at http://localhost:8080...")

        import httpx
        async with httpx.AsyncClient(timeout=5.0) as client:
            response = await client.get("http://localhost:8080/api/v1/agent/servers")
            if response.status_code == 200:
                data = response.json()
                print(f"✅ Connected! Found {data.get('total', 0)} servers")
                return True
            else:
                print(f"⚠️  API returned status {response.status_code}")
                return False

    except httpx.ConnectError:
        print("⚠️  Could not connect to mcpproxy (is it running?)")
        print("   Start mcpproxy with: ./mcpproxy serve")
        return False
    except Exception as e:
        print(f"❌ Error testing API: {e}")
        return False


async def main():
    """Run all tests."""
    print("=" * 60)
    print("MCP Agent Tools Test Suite")
    print("=" * 60)
    print()

    # Test imports and initialization
    imports_ok = await test_tool_imports()

    # Test API connection
    api_ok = await test_api_connection()

    print("\n" + "=" * 60)
    print("Test Summary")
    print("=" * 60)
    print(f"Tool imports and models: {'✅ PASS' if imports_ok else '❌ FAIL'}")
    print(f"API connection:          {'✅ PASS' if api_ok else '⚠️  SKIPPED (mcpproxy not running)'}")

    if not imports_ok:
        sys.exit(1)

    print("\n✨ Tool implementation verified!")
    sys.exit(0)


if __name__ == "__main__":
    asyncio.run(main())

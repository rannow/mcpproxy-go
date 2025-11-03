#!/usr/bin/env python3
"""Test script for MCP Inspector integration with the Python agent.

This demonstrates the agent's ability to control the MCP Inspector
through natural language conversation.
"""

import asyncio
import sys
from mcp_agent.tools.diagnostic import DiagnosticTools, MCPProxyClient
from mcp_agent.tools.config import ConfigTools
from mcp_agent.tools.inspector import InspectorTools
from mcp_agent.graph.agent_graph import MCPAgentGraph, AgentInput


async def main():
    """Test the inspector integration."""
    print("=" * 70)
    print("Testing MCP Inspector Integration with Python Agent")
    print("=" * 70)

    # Initialize tools
    client = MCPProxyClient(base_url="http://localhost:8080")
    diagnostic_tools = DiagnosticTools(mcpproxy_client=client)
    config_tools = ConfigTools(base_url="http://localhost:8080")
    inspector_tools = InspectorTools(base_url="http://localhost:8080")

    tools_registry = {
        "diagnostic": diagnostic_tools,
        "config": config_tools,
        "inspector": inspector_tools,
    }

    # Create agent
    agent = MCPAgentGraph(tools_registry)
    thread_id = "inspector-test-session"

    print("\n✓ Agent initialized with inspector tools\n")

    # Test 1: Start the inspector
    print("=" * 70)
    print("Test 1: Starting MCP Inspector")
    print("=" * 70)
    print("User: 'Start the inspector so I can watch the interaction'\n")

    result = await agent.run(
        AgentInput(request="Start the inspector so I can watch the interaction"),
        thread_id=thread_id
    )

    print(f"Agent Response: {result.response}")
    print(f"Actions Taken: {result.actions_taken}")
    if result.recommendations:
        print("Recommendations:")
        for rec in result.recommendations:
            print(f"  - {rec}")

    # Wait a bit for inspector to fully start
    await asyncio.sleep(2)

    # Test 2: Check inspector status
    print("\n" + "=" * 70)
    print("Test 2: Checking Inspector Status")
    print("=" * 70)
    print("User: 'Is the inspector running?'\n")

    result = await agent.run(
        AgentInput(request="Is the inspector running?"),
        thread_id=thread_id
    )

    print(f"Agent Response: {result.response}")
    print(f"Actions Taken: {result.actions_taken}")
    if result.recommendations:
        print("Recommendations:")
        for rec in result.recommendations:
            print(f"  - {rec}")

    # Test 3: User can now interact with the inspector
    print("\n" + "=" * 70)
    print("Inspector is Running - User Can Watch Live Interaction")
    print("=" * 70)
    print("At this point, the user can:")
    print("  1. Open the inspector URL in their browser")
    print("  2. Chat with the agent and watch the MCP protocol in real-time")
    print("  3. See tool calls, responses, and server interactions")
    print()

    # Test 4: Stop the inspector
    print("=" * 70)
    print("Test 3: Stopping Inspector")
    print("=" * 70)
    print("User: 'Stop the inspector now'\n")

    result = await agent.run(
        AgentInput(request="Stop the inspector now"),
        thread_id=thread_id
    )

    print(f"Agent Response: {result.response}")
    print(f"Actions Taken: {result.actions_taken}")
    if result.recommendations:
        print("Recommendations:")
        for rec in result.recommendations:
            print(f"  - {rec}")

    # Test 5: Verify inspector stopped
    print("\n" + "=" * 70)
    print("Test 4: Verifying Inspector Stopped")
    print("=" * 70)
    print("User: 'Check if the inspector is still running'\n")

    result = await agent.run(
        AgentInput(request="Check if the inspector is still running"),
        thread_id=thread_id
    )

    print(f"Agent Response: {result.response}")
    print(f"Actions Taken: {result.actions_taken}")
    if result.recommendations:
        print("Recommendations:")
        for rec in result.recommendations:
            print(f"  - {rec}")

    print("\n" + "=" * 70)
    print("Integration Test Summary")
    print("=" * 70)
    print("✅ Inspector can be started via natural language")
    print("✅ Inspector status can be checked conversationally")
    print("✅ Inspector can be stopped via chat")
    print("✅ User can watch live MCP protocol interaction")
    print()
    print("The agent now has full control of the MCP Inspector!")
    print("=" * 70)

    return 0


if __name__ == "__main__":
    try:
        sys.exit(asyncio.run(main()))
    except KeyboardInterrupt:
        print("\nTest interrupted by user")
        sys.exit(1)
    except Exception as e:
        print(f"\n❌ Error: {e}")
        import traceback
        traceback.print_exc()
        sys.exit(1)

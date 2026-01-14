import asyncio
from mcp_agent.tools.web_search import WebSearchTools
from mcp_agent.tools.shell import ShellTools
from mcp_agent.graph.agent_graph import MCPAgentGraph, AgentInput

class MockDiagnosticTools:
    pass

class MockConfigTools:
    pass

async def test_capabilities():
    print("--- Testing Agent Capabilities ---")
    
    # 1. Test Web Search
    print("\n[1] Testing Web Search...")
    search_tool = WebSearchTools()
    shell_tool = ShellTools()
    
    tools_registry = {
        "diagnostic": MockDiagnosticTools(),
        "config": MockConfigTools(),
        "web_search": search_tool,
        "shell": shell_tool
    }
    
    agent = MCPAgentGraph(tools_registry)
    
    # Test "research" routing
    print("  Query: 'search for current time in Tokyo'")
    input_data = AgentInput(request="search for current time in Tokyo")
    result = await agent.run(input_data)
    print(f"  Response: {result.response[:100]}...")
    if result.actions_taken:
        print(f"  Action taken: {result.actions_taken[0]}")
    
    # 2. Test Execution
    print("\n[2] Testing Program Execution...")
    print("  Query: 'run echo Hello World'")
    input_data = AgentInput(request="run echo Hello World")
    result = await agent.run(input_data)
    print(f"  Response: {result.response[:100]}...")
    if result.actions_taken:
        print(f"  Action taken: {result.actions_taken[0]}")

    # 3. Test Inspector
    print("\n[3] Testing Inspector...")
    print("  Query: 'start inspector'")
    input_data = AgentInput(request="start inspector")
    # Mock specific tools for this test
    from mcp_agent.tools.inspector import InspectorTools, InspectorStartResponse
    
    class MockInspectorTools:
        async def start_inspector(self, open_browser=False):
             return InspectorStartResponse(success=True, url="http://localhost:5173", message="Started")
    
    tools_registry["inspector"] = MockInspectorTools()
    agent = MCPAgentGraph(tools_registry)

    result = await agent.run(input_data)
    print(f"  Response: {result.response[:100]}...")
    if result.actions_taken:
        print(f"  Action taken: {result.actions_taken[0]}")

if __name__ == "__main__":
    asyncio.run(test_capabilities())

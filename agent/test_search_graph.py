import asyncio
from mcp_agent.tools.web_search import WebSearchTools
from mcp_agent.graph.agent_graph import MCPAgentGraph, AgentInput

class MockDiagnosticTools:
    pass

class MockConfigTools:
    pass

async def test_search():
    print("Testing Web Search Tool directly...")
    tool = WebSearchTools()
    results = await tool.search_web("python programming")
    print(f"Found {len(results)} results")
    for r in results[:1]:
         print(f"Sample: {r.title} - {r.href}")

    print("\nTesting Agent Graph Routing...")
    tools_registry = {
        "diagnostic": MockDiagnosticTools(),
        "config": MockConfigTools(),
        "web_search": tool
    }
    
    agent = MCPAgentGraph(tools_registry)
    
    # Test "research" routing
    input_data = AgentInput(request="search for current time in Tokyo")
    result = await agent.run(input_data)
    
    print(f"Agent Response: {result.response}")
    print(f"Actions Taken: {result.actions_taken}")
    print(f"Recommendations: {result.recommendations[:2]}")

if __name__ == "__main__":
    asyncio.run(test_search())

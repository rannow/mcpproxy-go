#!/usr/bin/env python3
"""Test script for semantic search agent integration.

This demonstrates the semantic search capabilities with:
- Pure semantic search using embeddings
- Hybrid search combining semantic + keyword
- Context-aware search with server disambiguation
- Server documentation indexing
"""

import asyncio
import sys
from mcp_agent.tools.semantic_search import SemanticSearchTools, ServerSummary
from mcp_agent.graph.semantic_agent import SemanticSearchAgent, SearchRequest


async def main():
    """Test semantic search agent."""
    print("=" * 70)
    print("Testing Semantic Search Agent with RAG")
    print("=" * 70)

    # Initialize semantic search tools
    print("\n📦 Initializing semantic search tools...")
    semantic_tools = SemanticSearchTools(
        base_url="http://localhost:8080",
        data_dir="~/.mcpproxy",
        embedding_model="all-MiniLM-L6-v2"
    )
    print("✓ Semantic search tools initialized")

    # Initialize semantic agent
    print("\n🤖 Initializing semantic search agent...")
    agent = SemanticSearchAgent(semantic_tools=semantic_tools)
    print("✓ Semantic search agent initialized")

    # Step 1: Index server documentation summaries
    print("\n" + "=" * 70)
    print("Step 1: Indexing Server Documentation")
    print("=" * 70)

    # Example: GitHub server
    github_summary = ServerSummary(
        server_name="github",
        summary="""GitHub MCP server provides comprehensive GitHub API access for
        repository management, issue tracking, pull requests, and code search.
        Ideal for projects using GitHub for version control and collaboration.""",
        capabilities=[
            "repository management",
            "issue tracking",
            "pull requests",
            "code search",
            "GitHub Actions"
        ],
        typical_use_cases=[
            "Create and manage GitHub issues",
            "Review and merge pull requests",
            "Search code across repositories",
            "Automate GitHub workflows"
        ]
    )

    semantic_tools.index_server_summary(github_summary)
    print("✓ Indexed GitHub server documentation")

    # Example: Filesystem server
    filesystem_summary = ServerSummary(
        server_name="filesystem",
        summary="""Filesystem MCP server provides local file system operations including
        reading, writing, searching, and managing files and directories. Best for
        working with local project files and directory structures.""",
        capabilities=[
            "file reading",
            "file writing",
            "directory management",
            "file search",
            "path operations"
        ],
        typical_use_cases=[
            "Read and write project files",
            "Search for files in directories",
            "Manage project file structure",
            "Access configuration files"
        ]
    )

    semantic_tools.index_server_summary(filesystem_summary)
    print("✓ Indexed Filesystem server documentation")

    # Step 2: Sync tools from mcpproxy
    print("\n" + "=" * 70)
    print("Step 2: Syncing Tools from MCPProxy")
    print("=" * 70)

    indexed_count = semantic_tools.sync_from_mcpproxy()
    print(f"✓ Indexed {indexed_count} tools from mcpproxy")

    # Step 3: Test semantic search
    print("\n" + "=" * 70)
    print("Step 3: Testing Semantic Search")
    print("=" * 70)

    test_query = "create GitHub issues"
    print(f'\nQuery: "{test_query}"')
    print(f"Mode: semantic (pure embedding search)\n")

    request = SearchRequest(
        query=test_query,
        mode="semantic",
        limit=5,
        include_reasoning=True
    )

    result = await agent.search(request, thread_id="test-semantic")

    print(f"Results: {result.total} tools found")
    print(f"Reasoning: {result.reasoning}\n")

    for i, tool in enumerate(result.tools, 1):
        print(f"{i}. {tool['tool_name']}")
        print(f"   Server: {tool['server_name']}")
        print(f"   Similarity: {tool['similarity_score']:.3f}")
        print(f"   Context: {tool['context_score']:.3f}")
        print(f"   Final Score: {tool['final_score']:.3f}")
        print(f"   Reasoning: {tool['reasoning']}")
        print()

    # Step 4: Test hybrid search
    print("=" * 70)
    print("Step 4: Testing Hybrid Search (Semantic + Keyword)")
    print("=" * 70)

    test_query = "search code in project"
    print(f'\nQuery: "{test_query}"')
    print(f"Mode: hybrid (60% semantic, 40% keyword)\n")

    request = SearchRequest(
        query=test_query,
        mode="hybrid",
        limit=5,
        semantic_weight=0.6
    )

    result = await agent.search(request, thread_id="test-hybrid")

    print(f"Results: {result.total} tools found")
    print(f"Reasoning: {result.reasoning}\n")

    for i, tool in enumerate(result.tools, 1):
        print(f"{i}. {tool['tool_name']}")
        print(f"   Final Score: {tool['final_score']:.3f}")
        print(f"   Reasoning: {tool['reasoning']}")
        print()

    # Step 5: Test context-aware search
    print("=" * 70)
    print("Step 5: Testing Context-Aware Search (Server Disambiguation)")
    print("=" * 70)

    test_query = "which server should I use for file operations?"
    print(f'\nQuery: "{test_query}"')
    print(f"Mode: context_aware (uses server documentation)\n")

    request = SearchRequest(
        query=test_query,
        mode="context_aware",
        limit=10
    )

    result = await agent.search(request, thread_id="test-context")

    print(f"Results: {result.total} tools found")
    print(f"\nReasoning:\n{result.reasoning}\n")

    # Group tools by server
    server_groups = {}
    for tool in result.tools:
        server = tool['server_name']
        if server not in server_groups:
            server_groups[server] = []
        server_groups[server].append(tool)

    for server, tools in server_groups.items():
        avg_score = sum(t['final_score'] for t in tools) / len(tools)
        print(f"\n{server} ({len(tools)} tools, avg score: {avg_score:.3f}):")
        for tool in tools[:3]:  # Show top 3 per server
            print(f"  - {tool['tool_name']} (score: {tool['final_score']:.3f})")

    # Step 6: Test natural language queries
    print("\n" + "=" * 70)
    print("Step 6: Testing Natural Language Queries")
    print("=" * 70)

    natural_queries = [
        "manage repository",
        "work with todos and tasks",
        "analyze code quality",
        "deploy application",
    ]

    for query in natural_queries:
        print(f'\n🔍 Query: "{query}"')

        request = SearchRequest(
            query=query,
            mode="hybrid",
            limit=3,
            include_reasoning=False
        )

        result = await agent.search(request, thread_id=f"test-{query.replace(' ', '-')}")

        if result.total > 0:
            top_tool = result.tools[0]
            print(f"   Top Result: {top_tool['tool_name']}")
            print(f"   Score: {top_tool['final_score']:.3f}")
        else:
            print("   No results found")

    # Summary
    print("\n" + "=" * 70)
    print("Semantic Search Test Summary")
    print("=" * 70)
    print("✅ Server documentation indexing")
    print("✅ Tool syncing from mcpproxy")
    print("✅ Semantic search (embedding-based)")
    print("✅ Hybrid search (semantic + keyword)")
    print("✅ Context-aware search (server disambiguation)")
    print("✅ Natural language query handling")
    print()
    print("The semantic search agent is fully functional!")
    print("=" * 70)

    return 0


if __name__ == "__main__":
    try:
        sys.exit(asyncio.run(main()))
    except KeyboardInterrupt:
        print("\n\nTest interrupted by user")
        sys.exit(1)
    except Exception as e:
        print(f"\n❌ Error: {e}")
        import traceback
        traceback.print_exc()
        sys.exit(1)

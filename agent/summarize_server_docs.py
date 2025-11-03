#!/usr/bin/env python3
"""Automatically generate and index MCP server documentation summaries.

Fetches documentation from various sources and creates structured summaries
for semantic search agent to use when disambiguating similar tools.
"""

import argparse
import asyncio
import sys
from typing import List, Dict, Optional
import httpx
from bs4 import BeautifulSoup

from mcp_agent.tools.semantic_search import SemanticSearchTools, ServerSummary


class DocumentationSummarizer:
    """Fetch and summarize MCP server documentation."""

    def __init__(self):
        self.client = httpx.AsyncClient(timeout=30.0, follow_redirects=True)

    async def fetch_url(self, url: str) -> Optional[str]:
        """Fetch content from URL."""
        try:
            response = await self.client.get(url)
            response.raise_for_status()
            return response.text
        except Exception as e:
            print(f"Failed to fetch {url}: {e}")
            return None

    def extract_text(self, html: str) -> str:
        """Extract text content from HTML."""
        soup = BeautifulSoup(html, 'html.parser')

        # Remove script and style elements
        for script in soup(["script", "style", "nav", "footer"]):
            script.decompose()

        # Get text
        text = soup.get_text()

        # Clean up text
        lines = (line.strip() for line in text.splitlines())
        chunks = (phrase.strip() for line in lines for phrase in line.split("  "))
        text = ' '.join(chunk for chunk in chunks if chunk)

        return text

    def generate_summary(self, text: str, max_length: int = 500) -> str:
        """Generate a summary from text.

        This is a simple extraction-based summary. For production,
        you could use an LLM to generate better summaries.
        """
        # For now, take the first meaningful paragraphs
        sentences = text.split('.')
        summary = ""

        for sentence in sentences:
            if len(summary) + len(sentence) < max_length:
                summary += sentence.strip() + ". "
            else:
                break

        return summary.strip()

    async def summarize_from_url(
        self,
        server_name: str,
        doc_url: str
    ) -> Optional[ServerSummary]:
        """Fetch and summarize documentation from URL."""
        print(f"\n📄 Fetching documentation for {server_name}...")
        print(f"   URL: {doc_url}")

        html = await self.fetch_url(doc_url)
        if not html:
            return None

        text = self.extract_text(html)
        summary = self.generate_summary(text)

        print(f"   ✓ Generated summary ({len(summary)} chars)")

        return ServerSummary(
            server_name=server_name,
            summary=summary,
            capabilities=[],  # Could be extracted from docs
            typical_use_cases=[]  # Could be extracted from docs
        )

    async def close(self):
        """Close HTTP client."""
        await self.client.aclose()


# Predefined server documentation sources
KNOWN_SERVERS = {
    "github": {
        "name": "github",
        "summary": """GitHub MCP server provides comprehensive GitHub API access for
        repository management, issue tracking, pull requests, code search, and GitHub Actions.
        Ideal for projects using GitHub for version control and team collaboration.""",
        "capabilities": [
            "repository management",
            "issue tracking",
            "pull requests",
            "code search",
            "GitHub Actions",
            "webhooks",
            "team management"
        ],
        "use_cases": [
            "Create and manage GitHub issues",
            "Review and merge pull requests",
            "Search code across repositories",
            "Automate GitHub workflows",
            "Manage repository settings",
            "Handle GitHub notifications"
        ]
    },
    "filesystem": {
        "name": "filesystem",
        "summary": """Filesystem MCP server provides local file system operations including
        reading, writing, searching, and managing files and directories. Best for working
        with local project files and directory structures.""",
        "capabilities": [
            "file reading",
            "file writing",
            "directory management",
            "file search",
            "path operations",
            "file permissions"
        ],
        "use_cases": [
            "Read and write project files",
            "Search for files in directories",
            "Manage project file structure",
            "Access configuration files",
            "Navigate directory trees",
            "Check file permissions"
        ]
    },
    "git": {
        "name": "git",
        "summary": """Git MCP server provides Git version control operations for managing
        repositories, commits, branches, and history. Enables direct Git operations
        without GitHub/GitLab API dependencies.""",
        "capabilities": [
            "commit management",
            "branch operations",
            "history inspection",
            "diff operations",
            "merge operations",
            "tag management"
        ],
        "use_cases": [
            "Create commits and branches",
            "View commit history",
            "Inspect file diffs",
            "Merge branches",
            "Manage tags and releases",
            "Cherry-pick commits"
        ]
    },
    "postgres": {
        "name": "postgres",
        "summary": """PostgreSQL MCP server provides database access for querying,
        schema management, and data operations. Supports complex SQL queries,
        transactions, and database administration tasks.""",
        "capabilities": [
            "SQL queries",
            "schema management",
            "data manipulation",
            "transactions",
            "database introspection",
            "performance monitoring"
        ],
        "use_cases": [
            "Execute SQL queries",
            "Manage database schema",
            "Insert and update data",
            "Analyze query performance",
            "Inspect database structure",
            "Manage database users"
        ]
    },
    "sqlite": {
        "name": "sqlite",
        "summary": """SQLite MCP server provides lightweight database access for local
        data storage and querying. Perfect for embedded databases, testing, and
        single-user applications.""",
        "capabilities": [
            "SQL queries",
            "database creation",
            "data operations",
            "schema management",
            "lightweight storage"
        ],
        "use_cases": [
            "Query local databases",
            "Create and manage SQLite databases",
            "Test database operations",
            "Store application data locally",
            "Perform data analysis"
        ]
    },
    "brave-search": {
        "name": "brave-search",
        "summary": """Brave Search MCP server provides privacy-focused web search capabilities.
        Access search results, news, and web content through Brave's search API.""",
        "capabilities": [
            "web search",
            "news search",
            "privacy-focused",
            "search results",
            "content retrieval"
        ],
        "use_cases": [
            "Search the web privately",
            "Find recent news articles",
            "Research topics online",
            "Gather web information",
            "Access search results programmatically"
        ]
    },
    "fetch": {
        "name": "fetch",
        "summary": """Fetch MCP server provides HTTP client capabilities for making web
        requests, downloading content, and accessing APIs. Supports various HTTP methods
        and authentication schemes.""",
        "capabilities": [
            "HTTP requests",
            "API calls",
            "content download",
            "authentication",
            "header management"
        ],
        "use_cases": [
            "Call REST APIs",
            "Download web content",
            "Test API endpoints",
            "Fetch remote data",
            "Access web services"
        ]
    },
    "slack": {
        "name": "slack",
        "summary": """Slack MCP server provides integration with Slack workspaces for
        messaging, channel management, and team communication. Enables automated
        Slack interactions and notifications.""",
        "capabilities": [
            "send messages",
            "channel management",
            "user management",
            "file sharing",
            "team communication"
        ],
        "use_cases": [
            "Send Slack messages",
            "Create and manage channels",
            "Share files in Slack",
            "Automate notifications",
            "Manage team communications"
        ]
    }
}


async def main():
    """Main function."""
    parser = argparse.ArgumentParser(
        description="Generate and index MCP server documentation summaries"
    )
    parser.add_argument(
        "servers",
        nargs="*",
        help="Server names to index (default: all known servers)"
    )
    parser.add_argument(
        "--url",
        help="Documentation URL for custom server"
    )
    parser.add_argument(
        "--custom-name",
        help="Name for custom server (required with --url)"
    )
    parser.add_argument(
        "--list",
        action="store_true",
        help="List all known servers"
    )

    args = parser.parse_args()

    # List known servers
    if args.list:
        print("Known MCP servers:")
        for name in sorted(KNOWN_SERVERS.keys()):
            print(f"  - {name}")
        return 0

    # Initialize semantic search tools
    print("🚀 Initializing semantic search tools...")
    semantic_tools = SemanticSearchTools()
    print("✓ Semantic search tools initialized\n")

    # Custom server from URL
    if args.url:
        if not args.custom_name:
            print("Error: --custom-name required with --url")
            return 1

        summarizer = DocumentationSummarizer()

        try:
            summary = await summarizer.summarize_from_url(
                server_name=args.custom_name,
                doc_url=args.url
            )

            if summary:
                semantic_tools.index_server_summary(summary)
                print(f"✓ Indexed {args.custom_name} from {args.url}")
            else:
                print(f"✗ Failed to summarize {args.custom_name}")
                return 1

        finally:
            await summarizer.close()

        return 0

    # Index known servers
    servers_to_index = args.servers if args.servers else list(KNOWN_SERVERS.keys())

    print("=" * 70)
    print("Indexing MCP Server Documentation Summaries")
    print("=" * 70)

    indexed_count = 0

    for server_name in servers_to_index:
        if server_name not in KNOWN_SERVERS:
            print(f"\n⚠️  Unknown server: {server_name}")
            print(f"   Known servers: {', '.join(KNOWN_SERVERS.keys())}")
            continue

        server_data = KNOWN_SERVERS[server_name]

        summary = ServerSummary(
            server_name=server_data["name"],
            summary=server_data["summary"],
            capabilities=server_data["capabilities"],
            typical_use_cases=server_data["use_cases"]
        )

        try:
            semantic_tools.index_server_summary(summary)
            print(f"\n✓ Indexed {server_name}")
            print(f"   Capabilities: {', '.join(server_data['capabilities'][:3])}...")
            print(f"   Use cases: {len(server_data['use_cases'])} examples")
            indexed_count += 1

        except Exception as e:
            print(f"\n✗ Failed to index {server_name}: {e}")

    # Summary
    print("\n" + "=" * 70)
    print(f"Summary: Indexed {indexed_count}/{len(servers_to_index)} servers")
    print("=" * 70)

    if indexed_count > 0:
        print("\n📊 Server summaries indexed in ChromaDB")
        print("   Location: ~/.mcpproxy/chroma_db/")
        print("\n🔍 Semantic search can now use server context for disambiguation")
        print("\nExample queries that benefit from server context:")
        print("   - 'which server for file operations?'")
        print("   - 'best tool for GitHub issues'")
        print("   - 'database queries'")
        print("   - 'web search and fetch'")

    return 0


if __name__ == "__main__":
    try:
        sys.exit(asyncio.run(main()))
    except KeyboardInterrupt:
        print("\n\nInterrupted by user")
        sys.exit(1)
    except Exception as e:
        print(f"\n❌ Error: {e}")
        import traceback
        traceback.print_exc()
        sys.exit(1)

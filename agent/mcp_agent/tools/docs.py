"""Documentation tools."""

import httpx
from typing import Optional, List, Dict, Any
from pydantic import BaseModel
from bs4 import BeautifulSoup
import re


class DocumentationResult(BaseModel):
    """Documentation search result."""
    title: str
    content: str
    source: str
    url: Optional[str] = None
    relevance_score: float = 0.0


class ToolDocumentation(BaseModel):
    """Documentation for MCP tool."""
    tool_name: str
    description: str
    parameters: Dict[str, Any]
    examples: List[str] = []
    server_name: str


class DocumentationTools:
    """Tools for searching and retrieving documentation."""

    def __init__(self, base_url: str = "http://localhost:8080"):
        """Initialize documentation tools.

        Args:
            base_url: Base URL for mcpproxy agent API
        """
        self.base_url = base_url
        self.client = httpx.AsyncClient(timeout=30.0, follow_redirects=True)

        # Common MCP documentation sources
        self.mcp_docs_urls = {
            "spec": "https://spec.modelcontextprotocol.io/",
            "github": "https://github.com/modelcontextprotocol/",
            "quickstart": "https://modelcontextprotocol.io/quickstart"
        }

    async def search_mcp_docs(
        self,
        query: str,
        server_name: Optional[str] = None,
        limit: int = 5
    ) -> List[DocumentationResult]:
        """Search MCP documentation and server docs.

        Args:
            query: Search query
            server_name: Optional specific server to search docs for
            limit: Maximum results to return

        Returns:
            List of documentation results
        """
        results = []

        # If server specified, get tool documentation from that server
        if server_name:
            tool_docs = await self._get_server_tool_docs(server_name, query)
            results.extend(tool_docs[:limit])

        # Search general MCP documentation
        if len(results) < limit:
            general_docs = await self._search_general_mcp_docs(query)
            results.extend(general_docs[:limit - len(results)])

        return results

    async def _get_server_tool_docs(
        self,
        server_name: str,
        query: Optional[str] = None
    ) -> List[DocumentationResult]:
        """Get tool documentation from specific server.

        Args:
            server_name: Server to get tool docs from
            query: Optional query to filter tools

        Returns:
            List of tool documentation results
        """
        results = []

        try:
            # Get server details with tool info
            response = await self.client.get(
                f"{self.base_url}/api/v1/agent/servers/{server_name}"
            )
            response.raise_for_status()
            data = response.json()

            # Extract tool information
            # Note: This is simplified - actual implementation would need
            # mcpproxy to expose tool descriptions via API
            tool_count = data.get("tools", {}).get("count", 0)

            if tool_count > 0:
                results.append(DocumentationResult(
                    title=f"{server_name} Tools",
                    content=f"Server has {tool_count} available tools. Use retrieve_tools to search them.",
                    source=server_name,
                    url=data.get("url"),
                    relevance_score=1.0
                ))

        except httpx.HTTPError:
            pass

        return results

    async def _search_general_mcp_docs(self, query: str) -> List[DocumentationResult]:
        """Search general MCP documentation.

        Args:
            query: Search query

        Returns:
            List of documentation results
        """
        results = []

        # Add MCP spec documentation
        results.append(DocumentationResult(
            title="MCP Specification",
            content=f"Official Model Context Protocol specification. Covers protocol design, message formats, and implementation guidelines related to: {query}",
            source="MCP Spec",
            url=self.mcp_docs_urls["spec"],
            relevance_score=0.9
        ))

        # Add GitHub documentation
        results.append(DocumentationResult(
            title="MCP GitHub Repository",
            content=f"Official MCP repository with code examples, server implementations, and community contributions related to: {query}",
            source="GitHub",
            url=self.mcp_docs_urls["github"],
            relevance_score=0.8
        ))

        # Add quickstart guide
        if any(keyword in query.lower() for keyword in ["start", "setup", "install", "getting"]):
            results.append(DocumentationResult(
                title="MCP Quickstart Guide",
                content="Step-by-step guide to getting started with Model Context Protocol",
                source="Quickstart",
                url=self.mcp_docs_urls["quickstart"],
                relevance_score=0.95
            ))

        return results

    async def fetch_external_docs(
        self,
        url: str,
        extract_text: bool = True
    ) -> str:
        """Fetch external documentation from URL.

        Args:
            url: URL to fetch documentation from
            extract_text: Whether to extract clean text from HTML

        Returns:
            Documentation content
        """
        try:
            response = await self.client.get(url)
            response.raise_for_status()

            content = response.text

            # If HTML, extract clean text
            if extract_text and 'html' in response.headers.get('content-type', '').lower():
                content = self._extract_text_from_html(content)

            return content

        except httpx.HTTPError as e:
            return f"Error fetching documentation: {str(e)}"

    def _extract_text_from_html(self, html: str) -> str:
        """Extract clean text from HTML.

        Args:
            html: HTML content

        Returns:
            Extracted text
        """
        try:
            soup = BeautifulSoup(html, 'html.parser')

            # Remove script and style elements
            for script in soup(["script", "style"]):
                script.decompose()

            # Get text
            text = soup.get_text()

            # Clean up whitespace
            lines = (line.strip() for line in text.splitlines())
            chunks = (phrase.strip() for line in lines for phrase in line.split("  "))
            text = '\n'.join(chunk for chunk in chunks if chunk)

            return text

        except Exception:
            return html

    async def get_tool_help(
        self,
        server_name: str,
        tool_name: str
    ) -> Optional[ToolDocumentation]:
        """Get help documentation for specific tool.

        Args:
            server_name: Server hosting the tool
            tool_name: Tool name (without server prefix)

        Returns:
            Tool documentation if available
        """
        # Note: This would require mcpproxy to expose tool schemas via API
        # For now, return basic structure

        try:
            response = await self.client.get(
                f"{self.base_url}/api/v1/agent/servers/{server_name}"
            )
            response.raise_for_status()

            return ToolDocumentation(
                tool_name=f"{server_name}:{tool_name}",
                description=f"Tool from {server_name} server",
                parameters={},
                examples=[],
                server_name=server_name
            )

        except httpx.HTTPError:
            return None

    async def get_server_readme(self, server_name: str) -> Optional[str]:
        """Get README or documentation for MCP server.

        Args:
            server_name: Server to get README for

        Returns:
            README content if available
        """
        try:
            # Try common GitHub patterns
            github_urls = [
                f"https://raw.githubusercontent.com/modelcontextprotocol/servers/main/{server_name}/README.md",
                f"https://raw.githubusercontent.com/{server_name}/main/README.md",
            ]

            for url in github_urls:
                try:
                    content = await self.fetch_external_docs(url, extract_text=False)
                    if content and "Error fetching" not in content and len(content) > 100:
                        return content
                except:
                    continue

            return None

        except Exception:
            return None

    async def search_examples(
        self,
        query: str,
        limit: int = 5
    ) -> List[DocumentationResult]:
        """Search for code examples and usage patterns.

        Args:
            query: What to search for
            limit: Maximum examples to return

        Returns:
            List of example documentation
        """
        results = []

        # Add common example patterns
        examples = {
            "connection": "Example: Connecting to MCP server using stdio or HTTP protocol",
            "authentication": "Example: OAuth authentication flow for MCP servers",
            "tool call": "Example: Calling MCP tools with proper parameter formatting",
            "error handling": "Example: Handling MCP protocol errors and retries",
            "logging": "Example: Configuring logging for MCP servers"
        }

        query_lower = query.lower()
        for topic, example in examples.items():
            if topic in query_lower or query_lower in topic:
                results.append(DocumentationResult(
                    title=f"Example: {topic.title()}",
                    content=example,
                    source="MCP Examples",
                    relevance_score=0.85
                ))

        return results[:limit]

    async def close(self):
        """Close HTTP client."""
        await self.client.aclose()

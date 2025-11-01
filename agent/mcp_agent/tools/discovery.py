"""Discovery tools for finding and installing MCP servers."""

import httpx
from typing import Optional, List, Dict, Any
from pydantic import BaseModel


class MCPServerInfo(BaseModel):
    """Information about an MCP server from registry."""
    id: str
    name: str
    description: str
    author: Optional[str] = "Unknown"
    version: Optional[str] = "latest"
    registry: str
    install_command: Optional[str] = None
    dependencies: List[str] = []
    url: Optional[str] = None
    command: Optional[str] = None
    args: Optional[List[str]] = None
    protocol: Optional[str] = None


class InstallResult(BaseModel):
    """Result of server installation."""
    success: bool
    message: str
    server_name: Optional[str] = None
    needs_restart: bool = False
    warnings: List[str] = []


class SearchResult(BaseModel):
    """Registry search results."""
    results: List[MCPServerInfo]
    query: str
    total_found: int
    registries_searched: List[str]


class DiscoveryTools:
    """Tools for discovering and installing MCP servers."""

    def __init__(self, base_url: str = "http://localhost:8080"):
        """Initialize discovery tools.

        Args:
            base_url: Base URL for mcpproxy agent API
        """
        self.base_url = base_url
        self.client = httpx.AsyncClient(timeout=30.0)

    async def search_mcp_registries(
        self,
        query: str,
        registry: Optional[str] = None,
        limit: int = 20
    ) -> SearchResult:
        """Search MCP server registries for available servers.

        Args:
            query: Search query (e.g., "github", "database", "weather")
            registry: Specific registry to search (optional)
            limit: Maximum results to return

        Returns:
            SearchResult with found servers and metadata
        """
        try:
            # Use mcpproxy agent API to search registries
            params = {"query": query}
            if registry:
                params["registry"] = registry

            response = await self.client.get(
                f"{self.base_url}/api/v1/agent/registries/search",
                params=params
            )
            response.raise_for_status()
            data = response.json()

            # Parse results into MCPServerInfo objects
            servers = []
            for result in data.get("results", [])[:limit]:
                servers.append(MCPServerInfo(
                    id=result.get("id", result.get("name", "unknown")),
                    name=result.get("name", "Unknown"),
                    description=result.get("description", "No description"),
                    author=result.get("author"),
                    version=result.get("version"),
                    registry=result.get("registry", registry or "unknown"),
                    install_command=result.get("install_command"),
                    dependencies=result.get("dependencies", []),
                    url=result.get("url"),
                    command=result.get("command"),
                    args=result.get("args"),
                    protocol=result.get("protocol", "http")
                ))

            return SearchResult(
                results=servers,
                query=query,
                total_found=len(servers),
                registries_searched=[registry] if registry else ["all"]
            )

        except httpx.HTTPError as e:
            # Return empty result on error
            return SearchResult(
                results=[],
                query=query,
                total_found=0,
                registries_searched=[],
            )

    async def install_server(
        self,
        server_id: str,
        name: Optional[str] = None,
        config: Optional[Dict[str, Any]] = None,
        auto_enable: bool = True
    ) -> InstallResult:
        """Install new MCP server from registry.

        Args:
            server_id: Server identifier from registry search
            name: Custom name for the server (defaults to server_id)
            config: Additional configuration options
            auto_enable: Whether to enable server after installation

        Returns:
            InstallResult with success status and details
        """
        try:
            server_name = name or server_id
            install_config = config or {}

            # Ensure server is enabled if requested
            if auto_enable and "enabled" not in install_config:
                install_config["enabled"] = True

            # Use mcpproxy agent API to install server
            payload = {
                "server_id": server_id,
                "name": server_name,
                "config": install_config
            }

            response = await self.client.post(
                f"{self.base_url}/api/v1/agent/install",
                json=payload
            )
            response.raise_for_status()
            data = response.json()

            return InstallResult(
                success=data.get("success", False),
                message=data.get("message", "Installation completed"),
                server_name=server_name,
                needs_restart=data.get("needs_restart", False),
                warnings=data.get("warnings", [])
            )

        except httpx.HTTPError as e:
            return InstallResult(
                success=False,
                message=f"Installation failed: {str(e)}",
                server_name=name or server_id
            )

    async def check_server_exists(self, server_name: str) -> bool:
        """Check if a server is already installed.

        Args:
            server_name: Name of server to check

        Returns:
            True if server exists, False otherwise
        """
        try:
            response = await self.client.get(
                f"{self.base_url}/api/v1/agent/servers/{server_name}"
            )
            return response.status_code == 200
        except httpx.HTTPError:
            return False

    async def get_install_recommendations(
        self,
        purpose: str,
        limit: int = 5
    ) -> List[MCPServerInfo]:
        """Get recommended servers for a specific purpose.

        Args:
            purpose: What you want to do (e.g., "work with GitHub", "analyze code")
            limit: Maximum recommendations to return

        Returns:
            List of recommended servers
        """
        # Search based on purpose keywords
        result = await self.search_mcp_registries(query=purpose, limit=limit)
        return result.results

    async def close(self):
        """Close HTTP client."""
        await self.client.aclose()

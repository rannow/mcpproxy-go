"""Unit tests for DiscoveryTools."""

import pytest
from unittest.mock import AsyncMock, MagicMock, patch
from httpx import Response, HTTPError, RequestError

from mcp_agent.tools.discovery import (
    DiscoveryTools,
    MCPServerInfo,
    InstallResult,
    SearchResult,
)


# Fixtures

@pytest.fixture
def discovery_tools():
    """Create DiscoveryTools instance."""
    return DiscoveryTools(base_url="http://localhost:8080")


@pytest.fixture
def sample_server_info():
    """Sample MCP server info."""
    return {
        "id": "github-mcp",
        "name": "GitHub MCP Server",
        "description": "MCP server for GitHub API",
        "author": "GitHub Inc.",
        "version": "1.0.0",
        "registry": "npm",
        "install_command": "npx @github/mcp-server",
        "dependencies": ["node>=18"],
        "url": "https://github.com/github/mcp-server",
        "command": "npx",
        "args": ["@github/mcp-server"],
        "protocol": "http"
    }


@pytest.fixture
def sample_search_results(sample_server_info):
    """Sample search results."""
    return {
        "results": [
            sample_server_info,
            {
                **sample_server_info,
                "id": "gitlab-mcp",
                "name": "GitLab MCP Server",
                "description": "MCP server for GitLab API"
            }
        ],
        "query": "git",
        "total_found": 2
    }


@pytest.fixture
def sample_install_result():
    """Sample installation result."""
    return {
        "success": True,
        "message": "Server installed successfully",
        "needs_restart": True,
        "warnings": ["Server requires authentication"]
    }


# Pydantic Model Tests

class TestMCPServerInfo:
    """Test MCPServerInfo model."""

    def test_server_info_full(self, sample_server_info):
        """Test server info with all fields."""
        info = MCPServerInfo(**sample_server_info)

        assert info.id == "github-mcp"
        assert info.name == "GitHub MCP Server"
        assert info.description == "MCP server for GitHub API"
        assert info.author == "GitHub Inc."
        assert info.version == "1.0.0"
        assert info.registry == "npm"
        assert info.install_command == "npx @github/mcp-server"
        assert info.dependencies == ["node>=18"]
        assert info.url == "https://github.com/github/mcp-server"
        assert info.command == "npx"
        assert info.args == ["@github/mcp-server"]
        assert info.protocol == "http"

    def test_server_info_minimal(self):
        """Test server info with minimal fields."""
        info = MCPServerInfo(
            id="test-server",
            name="Test Server",
            description="A test server",
            registry="test"
        )

        assert info.id == "test-server"
        assert info.name == "Test Server"
        assert info.description == "A test server"
        assert info.author == "Unknown"
        assert info.version == "latest"
        assert info.registry == "test"
        assert info.install_command is None
        assert info.dependencies == []
        assert info.url is None
        assert info.command is None
        assert info.args is None
        assert info.protocol is None


class TestInstallResult:
    """Test InstallResult model."""

    def test_install_result_success(self):
        """Test successful installation result."""
        result = InstallResult(
            success=True,
            message="Installation completed",
            server_name="github-server",
            needs_restart=True,
            warnings=["Warning 1", "Warning 2"]
        )

        assert result.success is True
        assert result.message == "Installation completed"
        assert result.server_name == "github-server"
        assert result.needs_restart is True
        assert len(result.warnings) == 2

    def test_install_result_failure(self):
        """Test failed installation result."""
        result = InstallResult(
            success=False,
            message="Installation failed: dependency error"
        )

        assert result.success is False
        assert "failed" in result.message.lower()
        assert result.server_name is None
        assert result.needs_restart is False
        assert result.warnings == []


class TestSearchResult:
    """Test SearchResult model."""

    def test_search_result_with_results(self, sample_server_info):
        """Test search result with found servers."""
        server1 = MCPServerInfo(**sample_server_info)
        server2 = MCPServerInfo(**{**sample_server_info, "id": "test2"})

        result = SearchResult(
            results=[server1, server2],
            query="github",
            total_found=2,
            registries_searched=["npm", "pypi"]
        )

        assert len(result.results) == 2
        assert result.query == "github"
        assert result.total_found == 2
        assert result.registries_searched == ["npm", "pypi"]

    def test_search_result_empty(self):
        """Test search result with no results."""
        result = SearchResult(
            results=[],
            query="nonexistent",
            total_found=0,
            registries_searched=[]
        )

        assert len(result.results) == 0
        assert result.query == "nonexistent"
        assert result.total_found == 0
        assert result.registries_searched == []


# DiscoveryTools Tests

class TestDiscoveryToolsInit:
    """Test DiscoveryTools initialization."""

    def test_init_default(self):
        """Test initialization with default values."""
        tools = DiscoveryTools()

        assert tools.base_url == "http://localhost:8080"
        assert tools.client is not None

    def test_init_custom_base_url(self):
        """Test initialization with custom base URL."""
        tools = DiscoveryTools(base_url="http://custom:9000")

        assert tools.base_url == "http://custom:9000"
        assert tools.client is not None


class TestSearchMCPRegistries:
    """Test search_mcp_registries method."""

    @pytest.mark.asyncio
    async def test_search_success(self, discovery_tools, sample_search_results):
        """Test successful registry search."""
        mock_response = AsyncMock(spec=Response)
        mock_response.json.return_value = sample_search_results
        mock_response.raise_for_status = MagicMock()

        discovery_tools.client.get = AsyncMock(return_value=mock_response)

        result = await discovery_tools.search_mcp_registries(
            query="git",
            limit=20
        )

        assert isinstance(result, SearchResult)
        assert len(result.results) == 2
        assert result.query == "git"
        assert result.total_found == 2
        assert result.registries_searched == ["all"]

        discovery_tools.client.get.assert_called_once_with(
            "http://localhost:8080/api/v1/agent/registries/search",
            params={"query": "git"}
        )

    @pytest.mark.asyncio
    async def test_search_with_specific_registry(self, discovery_tools, sample_search_results):
        """Test search in specific registry."""
        mock_response = AsyncMock(spec=Response)
        mock_response.json.return_value = sample_search_results
        mock_response.raise_for_status = MagicMock()

        discovery_tools.client.get = AsyncMock(return_value=mock_response)

        result = await discovery_tools.search_mcp_registries(
            query="git",
            registry="npm",
            limit=20
        )

        assert isinstance(result, SearchResult)
        assert result.registries_searched == ["npm"]

        discovery_tools.client.get.assert_called_once_with(
            "http://localhost:8080/api/v1/agent/registries/search",
            params={"query": "git", "registry": "npm"}
        )

    @pytest.mark.asyncio
    async def test_search_with_limit(self, discovery_tools):
        """Test search with result limit."""
        many_results = {
            "results": [
                {
                    "id": f"server-{i}",
                    "name": f"Server {i}",
                    "description": f"Description {i}",
                    "registry": "npm"
                }
                for i in range(50)
            ]
        }

        mock_response = AsyncMock(spec=Response)
        mock_response.json.return_value = many_results
        mock_response.raise_for_status = MagicMock()

        discovery_tools.client.get = AsyncMock(return_value=mock_response)

        result = await discovery_tools.search_mcp_registries(
            query="test",
            limit=10
        )

        # Should respect limit
        assert len(result.results) == 10
        assert result.total_found == 10

    @pytest.mark.asyncio
    async def test_search_http_error(self, discovery_tools):
        """Test search with HTTP error."""
        discovery_tools.client.get = AsyncMock(
            side_effect=HTTPError("Connection failed")
        )

        result = await discovery_tools.search_mcp_registries(query="git")

        # Should return empty result on error
        assert isinstance(result, SearchResult)
        assert len(result.results) == 0
        assert result.total_found == 0
        assert result.registries_searched == []

    @pytest.mark.asyncio
    async def test_search_no_results(self, discovery_tools):
        """Test search with no results."""
        mock_response = AsyncMock(spec=Response)
        mock_response.json.return_value = {"results": []}
        mock_response.raise_for_status = MagicMock()

        discovery_tools.client.get = AsyncMock(return_value=mock_response)

        result = await discovery_tools.search_mcp_registries(query="nonexistent")

        assert len(result.results) == 0
        assert result.total_found == 0


class TestInstallServer:
    """Test install_server method."""

    @pytest.mark.asyncio
    async def test_install_success(self, discovery_tools, sample_install_result):
        """Test successful server installation."""
        mock_response = AsyncMock(spec=Response)
        mock_response.json.return_value = sample_install_result
        mock_response.raise_for_status = MagicMock()

        discovery_tools.client.post = AsyncMock(return_value=mock_response)

        result = await discovery_tools.install_server(
            server_id="github-mcp",
            name="my-github-server"
        )

        assert isinstance(result, InstallResult)
        assert result.success is True
        assert result.server_name == "my-github-server"
        assert result.needs_restart is True
        assert len(result.warnings) == 1

        # Verify API call
        call_args = discovery_tools.client.post.call_args
        assert call_args[0][0] == "http://localhost:8080/api/v1/agent/install"
        payload = call_args[1]["json"]
        assert payload["server_id"] == "github-mcp"
        assert payload["name"] == "my-github-server"
        assert payload["config"]["enabled"] is True

    @pytest.mark.asyncio
    async def test_install_with_custom_config(self, discovery_tools, sample_install_result):
        """Test installation with custom configuration."""
        mock_response = AsyncMock(spec=Response)
        mock_response.json.return_value = sample_install_result
        mock_response.raise_for_status = MagicMock()

        discovery_tools.client.post = AsyncMock(return_value=mock_response)

        custom_config = {
            "env": {"API_KEY": "secret"},
            "enabled": False
        }

        result = await discovery_tools.install_server(
            server_id="github-mcp",
            config=custom_config,
            auto_enable=False
        )

        # Verify custom config was used
        call_args = discovery_tools.client.post.call_args
        payload = call_args[1]["json"]
        assert payload["config"]["env"]["API_KEY"] == "secret"
        assert payload["config"]["enabled"] is False

    @pytest.mark.asyncio
    async def test_install_auto_enable(self, discovery_tools, sample_install_result):
        """Test auto-enable functionality."""
        mock_response = AsyncMock(spec=Response)
        mock_response.json.return_value = sample_install_result
        mock_response.raise_for_status = MagicMock()

        discovery_tools.client.post = AsyncMock(return_value=mock_response)

        result = await discovery_tools.install_server(
            server_id="github-mcp",
            auto_enable=True
        )

        # Verify enabled=True was added to config
        call_args = discovery_tools.client.post.call_args
        payload = call_args[1]["json"]
        assert payload["config"]["enabled"] is True

    @pytest.mark.asyncio
    async def test_install_default_name(self, discovery_tools, sample_install_result):
        """Test installation with default server name."""
        mock_response = AsyncMock(spec=Response)
        mock_response.json.return_value = sample_install_result
        mock_response.raise_for_status = MagicMock()

        discovery_tools.client.post = AsyncMock(return_value=mock_response)

        result = await discovery_tools.install_server(server_id="github-mcp")

        # Name should default to server_id
        call_args = discovery_tools.client.post.call_args
        payload = call_args[1]["json"]
        assert payload["name"] == "github-mcp"

    @pytest.mark.asyncio
    async def test_install_http_error(self, discovery_tools):
        """Test installation with HTTP error."""
        discovery_tools.client.post = AsyncMock(
            side_effect=HTTPError("Installation failed")
        )

        result = await discovery_tools.install_server(
            server_id="github-mcp",
            name="test-server"
        )

        assert result.success is False
        assert "failed" in result.message.lower()
        assert result.server_name == "test-server"

    @pytest.mark.asyncio
    async def test_install_request_error(self, discovery_tools):
        """Test installation with request error."""
        discovery_tools.client.post = AsyncMock(
            side_effect=RequestError("Network error")
        )

        result = await discovery_tools.install_server(server_id="github-mcp")

        assert result.success is False
        assert result.server_name == "github-mcp"


class TestCheckServerExists:
    """Test check_server_exists method."""

    @pytest.mark.asyncio
    async def test_server_exists(self, discovery_tools):
        """Test checking for existing server."""
        mock_response = AsyncMock(spec=Response)
        mock_response.status_code = 200

        discovery_tools.client.get = AsyncMock(return_value=mock_response)

        exists = await discovery_tools.check_server_exists("github-server")

        assert exists is True

        discovery_tools.client.get.assert_called_once_with(
            "http://localhost:8080/api/v1/agent/servers/github-server"
        )

    @pytest.mark.asyncio
    async def test_server_not_exists_404(self, discovery_tools):
        """Test checking for non-existent server (404)."""
        mock_response = AsyncMock(spec=Response)
        mock_response.status_code = 404

        discovery_tools.client.get = AsyncMock(return_value=mock_response)

        exists = await discovery_tools.check_server_exists("nonexistent")

        assert exists is False

    @pytest.mark.asyncio
    async def test_server_not_exists_error(self, discovery_tools):
        """Test checking for server with HTTP error."""
        discovery_tools.client.get = AsyncMock(
            side_effect=HTTPError("Connection failed")
        )

        exists = await discovery_tools.check_server_exists("github-server")

        assert exists is False


class TestGetInstallRecommendations:
    """Test get_install_recommendations method."""

    @pytest.mark.asyncio
    async def test_get_recommendations(self, discovery_tools, sample_search_results):
        """Test getting install recommendations."""
        mock_response = AsyncMock(spec=Response)
        mock_response.json.return_value = sample_search_results
        mock_response.raise_for_status = MagicMock()

        discovery_tools.client.get = AsyncMock(return_value=mock_response)

        recommendations = await discovery_tools.get_install_recommendations(
            purpose="work with GitHub",
            limit=5
        )

        assert isinstance(recommendations, list)
        assert len(recommendations) == 2
        assert all(isinstance(r, MCPServerInfo) for r in recommendations)

        # Verify search was called with purpose
        call_args = discovery_tools.client.get.call_args
        assert "work with GitHub" in str(call_args)

    @pytest.mark.asyncio
    async def test_get_recommendations_with_limit(self, discovery_tools):
        """Test recommendations with custom limit."""
        many_results = {
            "results": [
                {
                    "id": f"server-{i}",
                    "name": f"Server {i}",
                    "description": f"Description {i}",
                    "registry": "npm"
                }
                for i in range(20)
            ]
        }

        mock_response = AsyncMock(spec=Response)
        mock_response.json.return_value = many_results
        mock_response.raise_for_status = MagicMock()

        discovery_tools.client.get = AsyncMock(return_value=mock_response)

        recommendations = await discovery_tools.get_install_recommendations(
            purpose="database",
            limit=3
        )

        # Should respect limit
        assert len(recommendations) <= 3


class TestClose:
    """Test close method."""

    @pytest.mark.asyncio
    async def test_close(self, discovery_tools):
        """Test closing HTTP client."""
        discovery_tools.client.aclose = AsyncMock()

        await discovery_tools.close()

        discovery_tools.client.aclose.assert_called_once()

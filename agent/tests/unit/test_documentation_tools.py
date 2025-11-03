"""Unit tests for DocumentationTools."""

import pytest
from unittest.mock import AsyncMock, MagicMock
from httpx import Response, HTTPError

from mcp_agent.tools.docs import (
    DocumentationTools,
    DocumentationResult,
    ToolDocumentation,
)


# Fixtures

@pytest.fixture
def doc_tools():
    """Create DocumentationTools instance."""
    return DocumentationTools(base_url="http://localhost:8080")


@pytest.fixture
def sample_doc_result():
    """Sample documentation result."""
    return {
        "title": "MCP Quickstart Guide",
        "content": "# Getting Started\n\nThis guide will help you get started with MCP...",
        "source": "Quickstart",
        "url": "https://modelcontextprotocol.io/quickstart",
        "relevance_score": 0.95
    }


@pytest.fixture
def sample_tool_doc():
    """Sample tool documentation."""
    return {
        "tool_name": "github-server:create_issue",
        "description": "Create a new GitHub issue",
        "parameters": {
            "title": "Issue title",
            "body": "Issue description"
        },
        "examples": [
            '{"title": "Bug found", "body": "Description..."}',
            '{"title": "Feature request", "body": "Feature description..."}'
        ],
        "server_name": "github-server"
    }


# Pydantic Model Tests

class TestDocumentationResult:
    """Test DocumentationResult model."""

    def test_documentation_result_full(self, sample_doc_result):
        """Test documentation result with all fields."""
        result = DocumentationResult(**sample_doc_result)

        assert result.title == "MCP Quickstart Guide"
        assert "Getting Started" in result.content
        assert result.url == "https://modelcontextprotocol.io/quickstart"
        assert result.source == "Quickstart"
        assert result.relevance_score == 0.95

    def test_documentation_result_minimal(self):
        """Test documentation result with minimal fields."""
        result = DocumentationResult(
            title="Test Doc",
            content="Test content",
            source="test"
        )

        assert result.title == "Test Doc"
        assert result.content == "Test content"
        assert result.source == "test"
        assert result.url is None
        assert result.relevance_score == 0.0


class TestToolDocumentation:
    """Test ToolDocumentation model."""

    def test_tool_documentation_full(self, sample_tool_doc):
        """Test tool documentation with all fields."""
        doc = ToolDocumentation(**sample_tool_doc)

        assert doc.tool_name == "github-server:create_issue"
        assert doc.description == "Create a new GitHub issue"
        assert "title" in doc.parameters
        assert len(doc.examples) == 2
        assert doc.server_name == "github-server"

    def test_tool_documentation_minimal(self):
        """Test tool documentation with minimal fields."""
        doc = ToolDocumentation(
            tool_name="test:tool",
            description="Test tool",
            parameters={},
            server_name="test-server"
        )

        assert doc.tool_name == "test:tool"
        assert doc.description == "Test tool"
        assert doc.parameters == {}
        assert doc.examples == []
        assert doc.server_name == "test-server"


# DocumentationTools Tests

class TestDocumentationToolsInit:
    """Test DocumentationTools initialization."""

    def test_init_default(self):
        """Test initialization with default values."""
        tools = DocumentationTools()

        assert tools.base_url == "http://localhost:8080"
        assert tools.client is not None
        assert "spec" in tools.mcp_docs_urls
        assert "github" in tools.mcp_docs_urls
        assert "quickstart" in tools.mcp_docs_urls

    def test_init_custom(self):
        """Test initialization with custom base URL."""
        tools = DocumentationTools(base_url="http://custom:9000")

        assert tools.base_url == "http://custom:9000"


class TestSearchMCPDocs:
    """Test search_mcp_docs method."""

    @pytest.mark.asyncio
    async def test_search_mcp_docs_general(self, doc_tools):
        """Test general MCP docs search."""
        result = await doc_tools.search_mcp_docs(query="protocol")

        assert isinstance(result, list)
        assert len(result) > 0
        assert all(isinstance(doc, DocumentationResult) for doc in result)
        # Should return general MCP docs (spec, github, etc.)
        assert any("MCP" in doc.title for doc in result)

    @pytest.mark.asyncio
    async def test_search_mcp_docs_with_server(self, doc_tools):
        """Test MCP docs search with server filter."""
        # Mock server response
        server_data = {
            "name": "github-server",
            "tools": {"count": 5},
            "url": "https://github.com/example/mcp"
        }

        mock_response = AsyncMock(spec=Response)
        mock_response.json.return_value = server_data
        mock_response.raise_for_status = MagicMock()

        doc_tools.client.get = AsyncMock(return_value=mock_response)

        result = await doc_tools.search_mcp_docs(
            query="create",
            server_name="github-server"
        )

        assert isinstance(result, list)
        # Should include both server-specific and general docs
        assert len(result) > 0

    @pytest.mark.asyncio
    async def test_search_mcp_docs_limit(self, doc_tools):
        """Test MCP docs search with limit."""
        result = await doc_tools.search_mcp_docs(query="test", limit=2)

        assert len(result) <= 2


class TestFetchExternalDocs:
    """Test fetch_external_docs method."""

    @pytest.mark.asyncio
    async def test_fetch_external_docs_html(self, doc_tools):
        """Test fetching HTML documentation."""
        html_content = """
        <html>
        <head><title>Test Doc</title></head>
        <body>
            <h1>Documentation</h1>
            <p>Content here</p>
        </body>
        </html>
        """

        mock_response = AsyncMock(spec=Response)
        mock_response.text = html_content
        mock_response.headers = {"content-type": "text/html"}
        mock_response.raise_for_status = MagicMock()

        doc_tools.client.get = AsyncMock(return_value=mock_response)

        result = await doc_tools.fetch_external_docs("https://example.com/docs")

        assert isinstance(result, str)
        assert "Documentation" in result
        assert len(result) > 0

    @pytest.mark.asyncio
    async def test_fetch_external_docs_plain_text(self, doc_tools):
        """Test fetching plain text documentation."""
        text_content = "Plain text documentation content"

        mock_response = AsyncMock(spec=Response)
        mock_response.text = text_content
        mock_response.headers = {"content-type": "text/plain"}
        mock_response.raise_for_status = MagicMock()

        doc_tools.client.get = AsyncMock(return_value=mock_response)

        result = await doc_tools.fetch_external_docs(
            "https://example.com/readme.txt",
            extract_text=False
        )

        assert result == text_content

    @pytest.mark.asyncio
    async def test_fetch_external_docs_http_error(self, doc_tools):
        """Test fetch with HTTP error."""
        doc_tools.client.get = AsyncMock(
            side_effect=HTTPError("Not found")
        )

        result = await doc_tools.fetch_external_docs("https://example.com/404")

        assert isinstance(result, str)
        assert "Error fetching documentation" in result


class TestGetToolHelp:
    """Test get_tool_help method."""

    @pytest.mark.asyncio
    async def test_get_tool_help_success(self, doc_tools):
        """Test successful tool help retrieval."""
        server_data = {
            "name": "github-server",
            "tools": {"count": 5}
        }

        mock_response = AsyncMock(spec=Response)
        mock_response.json.return_value = server_data
        mock_response.raise_for_status = MagicMock()

        doc_tools.client.get = AsyncMock(return_value=mock_response)

        result = await doc_tools.get_tool_help("github-server", "create_issue")

        assert isinstance(result, ToolDocumentation)
        assert result.tool_name == "github-server:create_issue"
        assert result.server_name == "github-server"

    @pytest.mark.asyncio
    async def test_get_tool_help_http_error(self, doc_tools):
        """Test tool help with HTTP error."""
        doc_tools.client.get = AsyncMock(
            side_effect=HTTPError("Server not found")
        )

        result = await doc_tools.get_tool_help("nonexistent", "tool")

        assert result is None


class TestGetServerReadme:
    """Test get_server_readme method."""

    @pytest.mark.asyncio
    async def test_get_server_readme_success(self, doc_tools):
        """Test successful README retrieval."""
        readme_content = "# GitHub MCP Server\n\nThis server provides GitHub integration." + " " * 100

        # Mock fetch_external_docs since that's what get_server_readme calls
        doc_tools.fetch_external_docs = AsyncMock(return_value=readme_content)

        result = await doc_tools.get_server_readme("github-server")

        assert isinstance(result, str)
        assert "GitHub MCP Server" in result

    @pytest.mark.asyncio
    async def test_get_server_readme_not_found(self, doc_tools):
        """Test README not found."""
        doc_tools.client.get = AsyncMock(
            side_effect=HTTPError("Not found")
        )

        result = await doc_tools.get_server_readme("nonexistent")

        assert result is None


class TestSearchExamples:
    """Test search_examples method."""

    @pytest.mark.asyncio
    async def test_search_examples_success(self, doc_tools):
        """Test successful example search."""
        result = await doc_tools.search_examples(query="connection")

        assert isinstance(result, list)
        assert len(result) > 0
        assert all(isinstance(ex, DocumentationResult) for ex in result)
        # Should find connection example
        assert any("connection" in ex.title.lower() for ex in result)

    @pytest.mark.asyncio
    async def test_search_examples_with_limit(self, doc_tools):
        """Test example search with limit."""
        result = await doc_tools.search_examples(query="error", limit=2)

        assert len(result) <= 2

    @pytest.mark.asyncio
    async def test_search_examples_no_match(self, doc_tools):
        """Test example search with no matches."""
        result = await doc_tools.search_examples(query="veryrarequery12345")

        assert isinstance(result, list)
        # May be empty or have low relevance results
        assert len(result) == 0


class TestExtractTextFromHTML:
    """Test _extract_text_from_html method."""

    def test_extract_text_simple(self, doc_tools):
        """Test text extraction from simple HTML."""
        html = "<html><body><p>Hello World</p></body></html>"

        text = doc_tools._extract_text_from_html(html)

        assert "Hello World" in text

    def test_extract_text_with_headings(self, doc_tools):
        """Test text extraction preserves headings."""
        html = """
        <html>
        <body>
            <h1>Title</h1>
            <p>Content</p>
            <h2>Subtitle</h2>
            <p>More content</p>
        </body>
        </html>
        """

        text = doc_tools._extract_text_from_html(html)

        assert "Title" in text
        assert "Subtitle" in text
        assert "Content" in text

    def test_extract_text_strips_scripts(self, doc_tools):
        """Test script tags are removed."""
        html = """
        <html>
        <body>
            <p>Visible text</p>
            <script>console.log('hidden');</script>
        </body>
        </html>
        """

        text = doc_tools._extract_text_from_html(html)

        assert "Visible text" in text
        assert "console.log" not in text

    def test_extract_text_error_handling(self, doc_tools):
        """Test error handling for invalid HTML."""
        invalid_html = "<<<invalid>>>"

        # Should return original HTML on error
        text = doc_tools._extract_text_from_html(invalid_html)

        assert isinstance(text, str)


class TestClose:
    """Test close method."""

    @pytest.mark.asyncio
    async def test_close(self, doc_tools):
        """Test closing HTTP client."""
        doc_tools.client.aclose = AsyncMock()

        await doc_tools.close()

        doc_tools.client.aclose.assert_called_once()

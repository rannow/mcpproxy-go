#!/usr/bin/env python3
"""FastAPI service exposing semantic search agent for mcpproxy.

Provides HTTP API endpoints for semantic tool discovery that can be
called from the Go mcpproxy server.
"""

import asyncio
import os
import sys
from typing import Optional
from contextlib import asynccontextmanager

from fastapi import FastAPI, HTTPException
from fastapi.responses import JSONResponse
from pydantic import BaseModel, Field
import uvicorn

from mcp_agent.tools.semantic_search import SemanticSearchTools, ServerSummary
from mcp_agent.graph.semantic_agent import SemanticSearchAgent, SearchRequest, SearchResponse


# Global instances
semantic_tools: Optional[SemanticSearchTools] = None
semantic_agent: Optional[SemanticSearchAgent] = None


@asynccontextmanager
async def lifespan(app: FastAPI):
    """Lifespan context manager for initialization and cleanup."""
    global semantic_tools, semantic_agent

    # Initialize on startup
    print("🚀 Initializing semantic search service...")

    base_url = os.getenv("MCPPROXY_URL", "http://localhost:8080")
    data_dir = os.getenv("MCPPROXY_DATA_DIR", "~/.mcpproxy")
    embedding_model = os.getenv("EMBEDDING_MODEL", "all-MiniLM-L6-v2")

    try:
        # Initialize semantic search tools
        semantic_tools = SemanticSearchTools(
            base_url=base_url,
            data_dir=data_dir,
            embedding_model=embedding_model
        )
        print(f"✓ Semantic search tools initialized")
        print(f"  - MCPProxy: {base_url}")
        print(f"  - Data directory: {data_dir}")
        print(f"  - Embedding model: {embedding_model}")

        # Initialize semantic agent
        semantic_agent = SemanticSearchAgent(
            semantic_tools=semantic_tools,
            use_postgres=False  # Can be enabled via env var
        )
        print(f"✓ Semantic search agent initialized")

        # Initial sync from mcpproxy
        print("📥 Syncing tools from mcpproxy...")
        indexed_count = semantic_tools.sync_from_mcpproxy()
        print(f"✓ Indexed {indexed_count} tools")

    except Exception as e:
        print(f"❌ Initialization failed: {e}")
        import traceback
        traceback.print_exc()
        sys.exit(1)

    yield

    # Cleanup on shutdown
    print("🛑 Shutting down semantic search service...")


app = FastAPI(
    title="MCPProxy Semantic Search API",
    description="Semantic tool discovery with RAG for mcpproxy",
    version="1.0.0",
    lifespan=lifespan
)


class HealthResponse(BaseModel):
    """Health check response."""
    status: str
    tools_indexed: int
    servers_indexed: int


class SyncResponse(BaseModel):
    """Tool sync response."""
    success: bool
    tools_indexed: int
    message: str


class ServerSummaryRequest(BaseModel):
    """Request to index server summary."""
    server_name: str
    summary: str
    capabilities: list[str] = Field(default_factory=list)
    typical_use_cases: list[str] = Field(default_factory=list)


@app.get("/health", response_model=HealthResponse)
async def health_check():
    """Health check endpoint."""
    if not semantic_tools:
        raise HTTPException(status_code=503, detail="Service not initialized")

    # Get collection counts
    try:
        tools_count = semantic_tools.tools_collection.count()
        servers_count = semantic_tools.servers_collection.count()

        return HealthResponse(
            status="healthy",
            tools_indexed=tools_count,
            servers_indexed=servers_count
        )
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Health check failed: {str(e)}")


@app.post("/api/semantic-search", response_model=SearchResponse)
async def semantic_search(request: SearchRequest):
    """Perform semantic search for tools.

    Args:
        request: Search request with query and parameters

    Returns:
        Ranked tools with reasoning
    """
    if not semantic_agent:
        raise HTTPException(status_code=503, detail="Service not initialized")

    try:
        result = await semantic_agent.search(request)
        return result

    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Search failed: {str(e)}")


@app.post("/api/sync-tools", response_model=SyncResponse)
async def sync_tools():
    """Sync tool index from mcpproxy.

    Fetches all current tools and re-indexes them with embeddings.
    """
    if not semantic_tools:
        raise HTTPException(status_code=503, detail="Service not initialized")

    try:
        indexed_count = semantic_tools.sync_from_mcpproxy()

        return SyncResponse(
            success=True,
            tools_indexed=indexed_count,
            message=f"Successfully indexed {indexed_count} tools"
        )

    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Sync failed: {str(e)}")


@app.post("/api/index-server")
async def index_server(request: ServerSummaryRequest):
    """Index MCP server documentation summary.

    Args:
        request: Server summary with capabilities and use cases

    Returns:
        Success response
    """
    if not semantic_tools:
        raise HTTPException(status_code=503, detail="Service not initialized")

    try:
        server_summary = ServerSummary(
            server_name=request.server_name,
            summary=request.summary,
            capabilities=request.capabilities,
            typical_use_cases=request.typical_use_cases
        )

        semantic_tools.index_server_summary(server_summary)

        return JSONResponse(
            content={
                "success": True,
                "message": f"Indexed server: {request.server_name}"
            }
        )

    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Index server failed: {str(e)}")


@app.get("/api/servers")
async def list_servers():
    """List indexed MCP servers."""
    if not semantic_tools:
        raise HTTPException(status_code=503, detail="Service not initialized")

    try:
        results = semantic_tools.servers_collection.get(
            include=["metadatas", "documents"]
        )

        if not results or not results['ids']:
            return JSONResponse(content={"servers": []})

        servers = []
        for i, server_id in enumerate(results['ids']):
            metadata = results['metadatas'][i] if results['metadatas'] else {}
            document = results['documents'][i] if results['documents'] else ""

            servers.append({
                "server_name": server_id,
                "summary": metadata.get("summary", ""),
                "capabilities": metadata.get("capabilities", "").split(",") if metadata.get("capabilities") else [],
                "use_cases": metadata.get("use_cases", "").split(",") if metadata.get("use_cases") else []
            })

        return JSONResponse(content={"servers": servers})

    except Exception as e:
        raise HTTPException(status_code=500, detail=f"List servers failed: {str(e)}")


@app.get("/")
async def root():
    """Root endpoint with service information."""
    return {
        "service": "MCPProxy Semantic Search API",
        "version": "1.0.0",
        "status": "running",
        "endpoints": {
            "health": "/health",
            "search": "/api/semantic-search",
            "sync": "/api/sync-tools",
            "index_server": "/api/index-server",
            "list_servers": "/api/servers"
        }
    }


def main():
    """Run the FastAPI server."""
    port = int(os.getenv("SEMANTIC_SEARCH_PORT", "8081"))
    host = os.getenv("SEMANTIC_SEARCH_HOST", "127.0.0.1")

    print(f"🌐 Starting semantic search API on {host}:{port}")

    uvicorn.run(
        app,
        host=host,
        port=port,
        log_level="info"
    )


if __name__ == "__main__":
    main()

"""Semantic search tools for intelligent tool discovery using RAG.

Provides semantic search capabilities with:
- Vector embeddings for tools and server documentation
- RAG pipeline for context-aware tool selection
- Intelligent reranking based on server context
- Multi-server disambiguation
"""

import chromadb
from chromadb.config import Settings
import httpx
import os
from typing import List, Optional, Dict, Any
from pydantic import BaseModel, Field
from sentence_transformers import SentenceTransformer


class ServerSummary(BaseModel):
    """MCP server documentation summary."""
    server_name: str = Field(description="Server name")
    summary: str = Field(description="Documentation summary")
    capabilities: List[str] = Field(default_factory=list, description="Server capabilities")
    typical_use_cases: List[str] = Field(default_factory=list, description="Typical use cases")


class ToolCandidate(BaseModel):
    """Tool candidate with context."""
    tool_name: str = Field(description="Full tool name (server:tool)")
    server_name: str = Field(description="Server name")
    description: str = Field(description="Tool description")
    input_schema: Dict[str, Any] = Field(default_factory=dict, description="Tool parameters")
    similarity_score: float = Field(description="Embedding similarity score")
    context_score: float = Field(description="Server context relevance score")
    final_score: float = Field(description="Combined relevance score")
    reasoning: str = Field(default="", description="Selection reasoning")


class SemanticSearchResult(BaseModel):
    """Semantic search result."""
    query: str
    tools: List[ToolCandidate]
    total: int
    reasoning: str = Field(default="", description="Overall search reasoning")


class SemanticSearchTools:
    """Tools for semantic search with RAG pipeline.

    Uses vector embeddings and server context to intelligently select
    the best tools for a given request, even when multiple tools can
    accomplish the same task.
    """

    def __init__(
        self,
        base_url: str = "http://localhost:8080",
        data_dir: str = "~/.mcpproxy",
        embedding_model: str = "all-MiniLM-L6-v2"
    ):
        """Initialize semantic search tools.

        Args:
            base_url: Base URL of the mcpproxy server
            data_dir: Data directory for ChromaDB storage
            embedding_model: Sentence transformer model name
        """
        self.base_url = base_url.rstrip('/')
        self.client = httpx.Client(timeout=30.0)

        # Expand data directory path
        data_dir = os.path.expanduser(data_dir)
        chroma_dir = os.path.join(data_dir, "chroma_db")

        # Initialize ChromaDB client
        self.chroma_client = chromadb.PersistentClient(
            path=chroma_dir,
            settings=Settings(
                anonymized_telemetry=False,
                allow_reset=True
            )
        )

        # Initialize embedding model
        self.embedding_model = SentenceTransformer(embedding_model)

        # Get or create collections
        self.servers_collection = self.chroma_client.get_or_create_collection(
            name="mcp_servers",
            metadata={"description": "MCP server documentation summaries"}
        )

        self.tools_collection = self.chroma_client.get_or_create_collection(
            name="mcp_tools",
            metadata={"description": "MCP tool descriptions with server context"}
        )

    def _generate_embedding(self, text: str) -> List[float]:
        """Generate embedding for text.

        Args:
            text: Text to embed

        Returns:
            Embedding vector
        """
        embedding = self.embedding_model.encode(text, convert_to_numpy=True)
        return embedding.tolist()

    def index_server_summary(self, server_summary: ServerSummary) -> None:
        """Index MCP server documentation summary.

        Args:
            server_summary: Server summary to index
        """
        # Create searchable text combining all information
        searchable_text = f"""
        Server: {server_summary.server_name}
        Summary: {server_summary.summary}
        Capabilities: {', '.join(server_summary.capabilities)}
        Use Cases: {', '.join(server_summary.typical_use_cases)}
        """.strip()

        # Generate embedding
        embedding = self._generate_embedding(searchable_text)

        # Store in ChromaDB
        self.servers_collection.upsert(
            ids=[server_summary.server_name],
            documents=[searchable_text],
            embeddings=[embedding],
            metadatas=[{
                "server_name": server_summary.server_name,
                "summary": server_summary.summary,
                "capabilities": ",".join(server_summary.capabilities),
                "use_cases": ",".join(server_summary.typical_use_cases)
            }]
        )

    def index_tool(
        self,
        tool_name: str,
        server_name: str,
        description: str,
        input_schema: Dict[str, Any],
        server_context: Optional[str] = None
    ) -> None:
        """Index a tool with server context.

        Args:
            tool_name: Full tool name (server:tool)
            server_name: Server name
            description: Tool description
            input_schema: Tool parameters schema
            server_context: Optional server documentation context
        """
        # Create searchable text with server context
        searchable_text = f"""
        Tool: {tool_name}
        Server: {server_name}
        Description: {description}
        """

        if server_context:
            searchable_text += f"\nServer Context: {server_context}"

        # Add parameter information
        if input_schema and "properties" in input_schema:
            params = ", ".join(input_schema["properties"].keys())
            searchable_text += f"\nParameters: {params}"

        searchable_text = searchable_text.strip()

        # Generate embedding
        embedding = self._generate_embedding(searchable_text)

        # Store in ChromaDB
        self.tools_collection.upsert(
            ids=[tool_name],
            documents=[searchable_text],
            embeddings=[embedding],
            metadatas=[{
                "tool_name": tool_name,
                "server_name": server_name,
                "description": description,
                "has_server_context": bool(server_context)
            }]
        )

    def _get_server_context(self, query: str, top_k: int = 3) -> Dict[str, float]:
        """Retrieve relevant server contexts for query.

        Args:
            query: Search query
            top_k: Number of servers to retrieve

        Returns:
            Dictionary mapping server names to relevance scores
        """
        # Generate query embedding
        query_embedding = self._generate_embedding(query)

        # Search server summaries
        results = self.servers_collection.query(
            query_embeddings=[query_embedding],
            n_results=top_k
        )

        if not results or not results['ids']:
            return {}

        # Extract server relevance scores
        server_scores = {}
        for i, server_id in enumerate(results['ids'][0]):
            distance = results['distances'][0][i] if results['distances'] else 0
            # Convert distance to similarity score (1 = perfect match, 0 = no match)
            similarity = max(0, 1 - distance)
            server_scores[server_id] = similarity

        return server_scores

    async def semantic_search(
        self,
        query: str,
        limit: int = 15,
        include_reasoning: bool = True
    ) -> SemanticSearchResult:
        """Perform semantic search for tools using RAG pipeline.

        Args:
            query: Natural language search query
            limit: Maximum number of tools to return
            include_reasoning: Include selection reasoning

        Returns:
            Semantic search result with ranked tools
        """
        # Step 1: Retrieve relevant server contexts
        server_scores = self._get_server_context(query, top_k=5)

        # Step 2: Retrieve tool candidates using embeddings
        query_embedding = self._generate_embedding(query)

        tool_results = self.tools_collection.query(
            query_embeddings=[query_embedding],
            n_results=min(limit * 3, 50)  # Retrieve more candidates for reranking
        )

        if not tool_results or not tool_results['ids']:
            return SemanticSearchResult(
                query=query,
                tools=[],
                total=0,
                reasoning="No tools found matching the query"
            )

        # Step 3: Rerank tools using server context
        candidates = []

        for i, tool_id in enumerate(tool_results['ids'][0]):
            metadata = tool_results['metadatas'][0][i]
            server_name = metadata['server_name']

            # Calculate similarity score
            distance = tool_results['distances'][0][i] if tool_results['distances'] else 0
            similarity_score = max(0, 1 - distance)

            # Calculate context score (boost tools from relevant servers)
            context_score = server_scores.get(server_name, 0.0)

            # Combined score: 70% similarity, 30% context
            final_score = (0.7 * similarity_score) + (0.3 * context_score)

            # Generate reasoning if requested
            reasoning = ""
            if include_reasoning:
                reasoning = f"Similarity: {similarity_score:.2f}, Server relevance: {context_score:.2f}"
                if context_score > 0.5:
                    reasoning += f" (High relevance from {server_name})"

            candidate = ToolCandidate(
                tool_name=tool_id,
                server_name=server_name,
                description=metadata['description'],
                input_schema={},  # Will be filled from mcpproxy
                similarity_score=similarity_score,
                context_score=context_score,
                final_score=final_score,
                reasoning=reasoning
            )

            candidates.append(candidate)

        # Step 4: Sort by final score and limit results
        candidates.sort(key=lambda x: x.final_score, reverse=True)
        top_candidates = candidates[:limit]

        # Generate overall reasoning
        overall_reasoning = f"Found {len(candidates)} candidates, selected top {len(top_candidates)} based on semantic similarity and server context"

        if server_scores:
            top_servers = sorted(server_scores.items(), key=lambda x: x[1], reverse=True)[:3]
            server_names = [s[0] for s in top_servers]
            overall_reasoning += f". Prioritized servers: {', '.join(server_names)}"

        return SemanticSearchResult(
            query=query,
            tools=top_candidates,
            total=len(top_candidates),
            reasoning=overall_reasoning
        )

    async def hybrid_search(
        self,
        query: str,
        limit: int = 15,
        semantic_weight: float = 0.6
    ) -> SemanticSearchResult:
        """Hybrid search combining semantic and keyword search.

        Args:
            query: Search query
            limit: Maximum results
            semantic_weight: Weight for semantic results (0-1)

        Returns:
            Combined search results
        """
        # Get semantic results
        semantic_results = await self.semantic_search(query, limit=limit)

        # Get keyword results from mcpproxy BM25
        try:
            response = self.client.get(
                f"{self.base_url}/api/tools/search",
                params={"query": query, "limit": limit}
            )
            response.raise_for_status()
            keyword_data = response.json()
            keyword_tools = keyword_data.get("tools", [])
        except Exception as e:
            print(f"Keyword search failed: {e}")
            keyword_tools = []

        # Combine and rerank results
        tool_scores = {}

        # Add semantic scores
        for i, tool in enumerate(semantic_results.tools):
            rank_score = 1.0 - (i / len(semantic_results.tools))
            tool_scores[tool.tool_name] = {
                "semantic": tool.final_score * rank_score * semantic_weight,
                "keyword": 0.0,
                "tool": tool
            }

        # Add keyword scores
        keyword_weight = 1.0 - semantic_weight
        for i, tool in enumerate(keyword_tools):
            tool_name = tool.get("name", "")
            rank_score = 1.0 - (i / len(keyword_tools))

            if tool_name in tool_scores:
                tool_scores[tool_name]["keyword"] = rank_score * keyword_weight
            else:
                # Create candidate from keyword result
                candidate = ToolCandidate(
                    tool_name=tool_name,
                    server_name=tool.get("server", ""),
                    description=tool.get("description", ""),
                    input_schema=tool.get("inputSchema", {}),
                    similarity_score=0.0,
                    context_score=0.0,
                    final_score=rank_score * keyword_weight,
                    reasoning=f"From keyword search (rank {i+1})"
                )
                tool_scores[tool_name] = {
                    "semantic": 0.0,
                    "keyword": rank_score * keyword_weight,
                    "tool": candidate
                }

        # Calculate final scores and sort
        final_results = []
        for tool_name, scores in tool_scores.items():
            combined_score = scores["semantic"] + scores["keyword"]
            tool = scores["tool"]
            tool.final_score = combined_score
            tool.reasoning = f"Hybrid: semantic={scores['semantic']:.2f}, keyword={scores['keyword']:.2f}"
            final_results.append(tool)

        final_results.sort(key=lambda x: x.final_score, reverse=True)
        final_results = final_results[:limit]

        return SemanticSearchResult(
            query=query,
            tools=final_results,
            total=len(final_results),
            reasoning=f"Hybrid search: {len(semantic_results.tools)} semantic + {len(keyword_tools)} keyword results"
        )

    def sync_from_mcpproxy(self) -> int:
        """Sync tool index from mcpproxy.

        Fetches all tools from mcpproxy and indexes them with embeddings.

        Returns:
            Number of tools indexed
        """
        try:
            # Fetch all tools from mcpproxy
            response = self.client.get(f"{self.base_url}/api/tools/list")
            response.raise_for_status()

            data = response.json()
            tools = data.get("tools", [])

            indexed_count = 0

            for tool in tools:
                tool_name = tool.get("name", "")
                server_name = tool.get("server", "")
                description = tool.get("description", "")
                input_schema = tool.get("inputSchema", {})

                if not tool_name or not server_name:
                    continue

                # Get server context if available
                server_context = None
                try:
                    server_results = self.servers_collection.get(
                        ids=[server_name],
                        include=["documents"]
                    )
                    if server_results and server_results['documents']:
                        server_context = server_results['documents'][0]
                except Exception:
                    pass

                # Index the tool
                self.index_tool(
                    tool_name=tool_name,
                    server_name=server_name,
                    description=description,
                    input_schema=input_schema,
                    server_context=server_context
                )

                indexed_count += 1

            return indexed_count

        except Exception as e:
            print(f"Failed to sync from mcpproxy: {e}")
            return 0

    def __del__(self):
        """Cleanup HTTP client."""
        try:
            self.client.close()
        except Exception:
            pass

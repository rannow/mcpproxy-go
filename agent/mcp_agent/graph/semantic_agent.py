"""LangGraph-based semantic search agent for intelligent tool discovery."""

from typing import TypedDict, Literal, List, Optional
from langgraph.graph import StateGraph, END
from pydantic import BaseModel, Field

from .checkpointer import create_checkpointer, get_checkpointer_info


class SemanticSearchState(TypedDict):
    """State for semantic search agent."""
    # User input
    user_query: str
    search_mode: Literal["semantic", "hybrid", "context_aware"]

    # Search parameters
    limit: int
    include_reasoning: bool
    semantic_weight: float  # For hybrid search

    # Results
    candidates: List[dict]
    selected_tools: List[dict]
    reasoning: str

    # Workflow control
    completed: bool
    error: Optional[str]


class SearchRequest(BaseModel):
    """Semantic search request."""
    query: str = Field(description="Natural language search query")
    mode: Literal["semantic", "hybrid", "context_aware"] = Field(
        default="hybrid",
        description="Search mode: semantic (embeddings only), hybrid (semantic + keyword), context_aware (with server disambiguation)"
    )
    limit: int = Field(default=15, description="Maximum tools to return")
    include_reasoning: bool = Field(default=True, description="Include selection reasoning")
    semantic_weight: float = Field(default=0.6, description="Weight for semantic vs keyword (0-1)")


class SearchResponse(BaseModel):
    """Semantic search response."""
    tools: List[dict]
    total: int
    reasoning: str
    mode: str


class SemanticSearchAgent:
    """LangGraph agent for semantic tool search with RAG."""

    def __init__(
        self,
        semantic_tools,
        postgres_url: Optional[str] = None,
        use_postgres: bool = False
    ):
        """Initialize semantic search agent.

        Args:
            semantic_tools: SemanticSearchTools instance
            postgres_url: PostgreSQL connection string
            use_postgres: Enable PostgreSQL checkpointer
        """
        self.semantic_tools = semantic_tools
        self.memory = create_checkpointer(postgres_url, use_postgres)
        self.graph = self._build_graph()

        # Log checkpointer configuration
        checkpointer_info = get_checkpointer_info(self.memory)
        print(f"✓ Semantic agent initialized with {checkpointer_info['type']} checkpointer")

    def _build_graph(self) -> StateGraph:
        """Build the LangGraph state machine."""
        workflow = StateGraph(SemanticSearchState)

        # Add nodes
        workflow.add_node("analyze_query", self._analyze_query)
        workflow.add_node("semantic_search", self._semantic_search)
        workflow.add_node("hybrid_search", self._hybrid_search)
        workflow.add_node("context_aware_search", self._context_aware_search)
        workflow.add_node("rerank_results", self._rerank_results)
        workflow.add_node("finalize", self._finalize)

        # Define workflow
        workflow.set_entry_point("analyze_query")

        workflow.add_conditional_edges(
            "analyze_query",
            self._route_search_mode,
            {
                "semantic": "semantic_search",
                "hybrid": "hybrid_search",
                "context_aware": "context_aware_search",
                "end": END
            }
        )

        workflow.add_edge("semantic_search", "rerank_results")
        workflow.add_edge("hybrid_search", "rerank_results")
        workflow.add_edge("context_aware_search", "rerank_results")
        workflow.add_edge("rerank_results", "finalize")
        workflow.add_edge("finalize", END)

        return workflow.compile(checkpointer=self.memory)

    async def _analyze_query(self, state: SemanticSearchState) -> SemanticSearchState:
        """Analyze search query and set parameters."""
        query = state["user_query"].lower()

        # Detect if query needs context-aware search
        # (multiple servers might have similar tools)
        context_indicators = [
            "which server", "which one", "best for", "recommend",
            "github or", "filesystem or", "database or"
        ]

        if any(indicator in query for indicator in context_indicators):
            state["search_mode"] = "context_aware"

        # Set defaults if not specified
        if "limit" not in state or state["limit"] == 0:
            state["limit"] = 15

        if "include_reasoning" not in state:
            state["include_reasoning"] = True

        if "semantic_weight" not in state or state["semantic_weight"] == 0:
            state["semantic_weight"] = 0.6

        return state

    async def _semantic_search(self, state: SemanticSearchState) -> SemanticSearchState:
        """Perform pure semantic search."""
        try:
            result = await self.semantic_tools.semantic_search(
                query=state["user_query"],
                limit=state["limit"],
                include_reasoning=state["include_reasoning"]
            )

            state["candidates"] = [tool.model_dump() for tool in result.tools]
            state["reasoning"] = result.reasoning

        except Exception as e:
            state["error"] = f"Semantic search failed: {str(e)}"
            state["candidates"] = []

        return state

    async def _hybrid_search(self, state: SemanticSearchState) -> SemanticSearchState:
        """Perform hybrid semantic + keyword search."""
        try:
            result = await self.semantic_tools.hybrid_search(
                query=state["user_query"],
                limit=state["limit"],
                semantic_weight=state["semantic_weight"]
            )

            state["candidates"] = [tool.model_dump() for tool in result.tools]
            state["reasoning"] = result.reasoning

        except Exception as e:
            state["error"] = f"Hybrid search failed: {str(e)}"
            state["candidates"] = []

        return state

    async def _context_aware_search(self, state: SemanticSearchState) -> SemanticSearchState:
        """Perform context-aware search with server disambiguation."""
        try:
            # First get semantic results
            semantic_result = await self.semantic_tools.semantic_search(
                query=state["user_query"],
                limit=state["limit"] * 2,  # Get more candidates for filtering
                include_reasoning=True
            )

            candidates = semantic_result.tools

            # Group by server to identify alternatives
            server_tools = {}
            for tool in candidates:
                server = tool.server_name
                if server not in server_tools:
                    server_tools[server] = []
                server_tools[server].append(tool)

            # Analyze which servers have similar capabilities
            reasoning_parts = [semantic_result.reasoning]

            if len(server_tools) > 1:
                reasoning_parts.append("\n\nServer Analysis:")
                for server, tools in server_tools.items():
                    avg_context_score = sum(t.context_score for t in tools) / len(tools)
                    reasoning_parts.append(
                        f"- {server}: {len(tools)} tools, avg relevance {avg_context_score:.2f}"
                    )

                # Recommend based on context scores
                best_server = max(server_tools.items(), key=lambda x: sum(t.context_score for t in x[1]))[0]
                reasoning_parts.append(
                    f"\nRecommendation: {best_server} appears most relevant based on server context"
                )

            state["candidates"] = [tool.model_dump() for tool in candidates[:state["limit"]]]
            state["reasoning"] = "\n".join(reasoning_parts)

        except Exception as e:
            state["error"] = f"Context-aware search failed: {str(e)}"
            state["candidates"] = []

        return state

    async def _rerank_results(self, state: SemanticSearchState) -> SemanticSearchState:
        """Rerank results based on multiple factors."""
        # Results are already ranked by the search methods
        # This node can add additional reranking logic if needed

        # For now, just ensure results are sorted by final_score
        candidates = state["candidates"]
        candidates.sort(key=lambda x: x.get("final_score", 0), reverse=True)
        state["selected_tools"] = candidates

        return state

    async def _finalize(self, state: SemanticSearchState) -> SemanticSearchState:
        """Finalize search results."""
        state["completed"] = True
        return state

    def _route_search_mode(self, state: SemanticSearchState) -> str:
        """Route to appropriate search method."""
        if state.get("error"):
            return "end"

        mode = state.get("search_mode", "hybrid")

        if mode == "semantic":
            return "semantic"
        elif mode == "hybrid":
            return "hybrid"
        elif mode == "context_aware":
            return "context_aware"
        else:
            return "hybrid"  # Default to hybrid

    async def search(
        self,
        request: SearchRequest,
        thread_id: str = "default"
    ) -> SearchResponse:
        """Execute semantic search.

        Args:
            request: Search request parameters
            thread_id: Thread ID for session persistence

        Returns:
            Search response with ranked tools
        """
        config = {"configurable": {"thread_id": thread_id}}

        # Create initial state
        initial_state: SemanticSearchState = {
            "user_query": request.query,
            "search_mode": request.mode,
            "limit": request.limit,
            "include_reasoning": request.include_reasoning,
            "semantic_weight": request.semantic_weight,
            "candidates": [],
            "selected_tools": [],
            "reasoning": "",
            "completed": False,
            "error": None
        }

        # Execute graph
        final_state = await self.graph.ainvoke(initial_state, config)

        # Build response
        return SearchResponse(
            tools=final_state.get("selected_tools", []),
            total=len(final_state.get("selected_tools", [])),
            reasoning=final_state.get("reasoning", ""),
            mode=final_state.get("search_mode", request.mode)
        )

# Semantic Search Agent Integration

AI-powered tool discovery using vector embeddings and RAG (Retrieval-Augmented Generation).

## Overview

The semantic search agent provides intelligent tool discovery that goes beyond simple keyword matching. It uses:

- **Vector Embeddings**: Tools and server documentation are encoded as embeddings for semantic similarity
- **Server Context**: MCP server documentation summaries help disambiguate when multiple tools can do the same thing
- **RAG Pipeline**: Retrieves relevant tools and reranks them using server context
- **LangGraph Orchestration**: Multi-step workflow for intelligent tool selection

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│  Claude/LLM → MCPProxy (Go)                                  │
│              ↓                                               │
│         semantic_search_tools (MCP Tool)                     │
│              ↓                                               │
│         HTTP API Call                                        │
└─────────────────────────────────────────────────────────────┘
                       ↓
┌─────────────────────────────────────────────────────────────┐
│  Semantic Search API (Python FastAPI)                        │
│              ↓                                               │
│  ┌─────────────────────────────────────────────────────┐    │
│  │ SemanticSearchAgent (LangGraph)                     │    │
│  │  ├─ Analyze Query                                   │    │
│  │  ├─ Retrieve Candidates (Vector Search)             │    │
│  │  ├─ Get Server Context                              │    │
│  │  ├─ Rerank with Context                             │    │
│  │  └─ Return Ranked Tools                             │    │
│  └─────────────────────────────────────────────────────┘    │
│              ↓                                               │
│  ┌─────────────────────────────────────────────────────┐    │
│  │ SemanticSearchTools                                 │    │
│  │  ├─ SentenceTransformers (Embeddings)               │    │
│  │  ├─ ChromaDB (Vector Storage)                       │    │
│  │  │   ├─ mcp_servers collection                      │    │
│  │  │   └─ mcp_tools collection                        │    │
│  │  └─ Hybrid Search (Semantic + Keyword)              │    │
│  └─────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────┘
```

## Components

### 1. SemanticSearchTools (`mcp_agent/tools/semantic_search.py`)

Core RAG implementation with vector embeddings:

**Key Features**:
- ChromaDB for vector storage (lightweight, embeddable)
- SentenceTransformers for embeddings (`all-MiniLM-L6-v2` by default)
- Two collections:
  - `mcp_servers`: Server documentation summaries with capabilities
  - `mcp_tools`: Tool descriptions with server context
- Hybrid search combining semantic + keyword search
- Automatic sync from mcpproxy

**Methods**:
- `semantic_search()`: Pure embedding-based search
- `hybrid_search()`: Combines semantic + BM25 keyword search
- `index_server_summary()`: Index server documentation
- `index_tool()`: Index tool with server context
- `sync_from_mcpproxy()`: Sync all tools from mcpproxy

### 2. SemanticSearchAgent (`mcp_agent/graph/semantic_agent.py`)

LangGraph orchestration for multi-step search:

**Workflow**:
1. **Analyze Query**: Detect if context-aware search is needed
2. **Search Mode Selection**:
   - `semantic`: Pure embedding search
   - `hybrid`: Semantic + keyword (default)
   - `context_aware`: With server disambiguation
3. **Rerank Results**: Combine similarity + context scores
4. **Finalize**: Return ranked tools with reasoning

**Scoring**:
- Similarity Score: Vector embedding distance (0-1)
- Context Score: Server relevance from documentation (0-1)
- Final Score: `0.7 × similarity + 0.3 × context`

### 3. FastAPI Service (`agent/semantic_search_api.py`)

HTTP API exposing the semantic search agent:

**Endpoints**:
- `POST /api/semantic-search`: Perform semantic search
- `POST /api/sync-tools`: Sync tools from mcpproxy
- `POST /api/index-server`: Index server documentation
- `GET /api/servers`: List indexed servers
- `GET /health`: Health check

**Configuration (Environment Variables)**:
- `MCPPROXY_URL`: mcpproxy base URL (default: `http://localhost:8080`)
- `MCPPROXY_DATA_DIR`: Data directory (default: `~/.mcpproxy`)
- `EMBEDDING_MODEL`: Model name (default: `all-MiniLM-L6-v2`)
- `SEMANTIC_SEARCH_PORT`: API port (default: `8081`)
- `SEMANTIC_SEARCH_HOST`: API host (default: `127.0.0.1`)

### 4. Go Integration (`internal/server/semantic.go`)

MCP tool integration for mcpproxy:

**SemanticSearchService**:
- HTTP client to call Python API
- Automatic health checking
- Tool sync coordination

**MCP Tool**: `semantic_search_tools`
- Natural language queries
- Multiple search modes
- Server context disambiguation
- Reasoning explanation

## Installation & Setup

### 1. Install Python Dependencies

```bash
cd agent
pip install -r requirements.txt

# Additional dependencies for semantic search
pip install sentence-transformers chromadb fastapi uvicorn
```

Or add to `requirements.txt`:
```
sentence-transformers>=2.2.0
chromadb>=0.4.0
fastapi>=0.104.0
uvicorn>=0.24.0
```

### 2. Start Semantic Search API

```bash
# From agent directory
python3 semantic_search_api.py
```

Or with custom configuration:
```bash
export MCPPROXY_URL="http://localhost:8080"
export EMBEDDING_MODEL="all-MiniLM-L6-v2"
export SEMANTIC_SEARCH_PORT="8081"

python3 semantic_search_api.py
```

The service will:
1. Initialize ChromaDB at `~/.mcpproxy/chroma_db`
2. Load sentence transformer model
3. Sync tools from mcpproxy
4. Start HTTP API on port 8081

### 3. Integrate with MCPProxy (Go)

Add to `internal/server/server.go`:

```go
// Add to Server struct (around line 76)
semanticSearchService *SemanticSearchService

// Initialize in NewServer (around line 170)
server.semanticSearchService = NewSemanticSearchService(
    os.Getenv("SEMANTIC_SEARCH_URL"), // Default: http://127.0.0.1:8081
    logger,
)

// Check availability
if server.semanticSearchService.IsAvailable(ctx) {
    logger.Info("Semantic search service available")
} else {
    logger.Warn("Semantic search service not available - tool will not be enabled")
}
```

Add to `internal/server/mcp.go`:

```go
// Add tool definition (around line 147, after retrieve_tools)
semanticSearchTool := createSemanticSearchTool()

// Add to tools array (around line 293)
tools := []mcp.Tool{
    retrieveToolsTool,
    semanticSearchTool, // ADD THIS
    callToolTool,
    // ...
}

// Add case in handleToolCall (around line 350)
case "semantic_search_tools":
    return p.handleSemanticSearch(ctx, request)
```

### 4. Build and Run

```bash
# Build mcpproxy
go build -o mcpproxy ./cmd/mcpproxy

# Start mcpproxy (in one terminal)
./mcpproxy serve

# Start semantic search API (in another terminal)
cd agent
python3 semantic_search_api.py
```

## Usage

### Basic Semantic Search

```python
# Using MCP protocol
{
  "method": "tools/call",
  "params": {
    "name": "semantic_search_tools",
    "arguments": {
      "query": "create GitHub issues",
      "mode": "hybrid",
      "limit": 10,
      "include_reasoning": true
    }
  }
}
```

**Response**:
```json
{
  "tools": [
    {
      "tool_name": "github:create_issue",
      "server_name": "github",
      "description": "Create a new issue in a GitHub repository",
      "similarity_score": 0.92,
      "context_score": 0.85,
      "final_score": 0.90,
      "reasoning": "Similarity: 0.92, Server relevance: 0.85 (High relevance from github)"
    }
  ],
  "total": 1,
  "reasoning": "Found 5 candidates, selected top 1 based on semantic similarity and server context. Prioritized servers: github, gitlab",
  "mode": "hybrid"
}
```

### Search Modes

#### 1. Semantic Mode (Pure Embedding Search)
Best for: Conceptual queries, when you want semantic understanding only

```json
{
  "query": "manage my tasks and todos",
  "mode": "semantic",
  "limit": 15
}
```

#### 2. Hybrid Mode (Semantic + Keyword)
Best for: General queries, balanced approach (recommended default)

```json
{
  "query": "search files in project",
  "mode": "hybrid",
  "semantic_weight": 0.6
}
```

**Semantic Weight**:
- `1.0`: Pure semantic search
- `0.6`: Default (60% semantic, 40% keyword)
- `0.0`: Pure keyword search

#### 3. Context-Aware Mode (With Server Disambiguation)
Best for: When multiple servers offer similar functionality

```json
{
  "query": "which server should I use for file operations?",
  "mode": "context_aware",
  "limit": 15
}
```

This mode:
- Groups tools by server
- Analyzes server relevance from documentation
- Provides recommendations based on context
- Explains why one server is better suited

### Example Queries

**Disambiguation Scenarios**:
```python
# Multiple file servers
"search code in project"
→ Returns: ast-grep, filesystem, git tools
→ Recommends: ast-grep (code-specific search)

# Multiple task managers
"create and track todos"
→ Returns: GitHub issues, todo server, project tools
→ Recommends: Based on project context in server docs

# API operations
"call external web services"
→ Returns: HTTP client, fetch, API tools
→ Recommends: Based on authentication capabilities
```

**Natural Language Queries**:
```python
"manage GitHub repository"
→ Finds all GitHub management tools

"analyze code quality"
→ Finds linting, testing, analysis tools

"deploy to production"
→ Finds CI/CD, Docker, deployment tools

"work with databases"
→ Finds PostgreSQL, SQLite, database tools
```

## Server Documentation Indexing

### Automatic Documentation Summarization

Create summaries of MCP server documentation:

```python
from mcp_agent.tools.semantic_search import SemanticSearchTools, ServerSummary

tools = SemanticSearchTools()

# Index GitHub server
github_summary = ServerSummary(
    server_name="github",
    summary="""GitHub MCP server provides comprehensive GitHub API access.
    Enables repository management, issue tracking, pull requests, and more.
    Ideal for projects using GitHub for version control and collaboration.""",
    capabilities=[
        "repository management",
        "issue tracking",
        "pull requests",
        "code search",
        "GitHub Actions integration"
    ],
    typical_use_cases=[
        "Create and manage GitHub issues",
        "Review and merge pull requests",
        "Search code across repositories",
        "Automate GitHub workflows"
    ]
)

tools.index_server_summary(github_summary)
```

### Via HTTP API

```bash
curl -X POST http://localhost:8081/api/index-server \
  -H "Content-Type: application/json" \
  -d '{
    "server_name": "github",
    "summary": "GitHub MCP server for repository management and collaboration",
    "capabilities": ["issues", "pull_requests", "repositories"],
    "typical_use_cases": ["Create issues", "Manage PRs", "Search code"]
  }'
```

## Performance & Scalability

### Embedding Model Selection

**Fast & Lightweight** (`all-MiniLM-L6-v2`):
- Size: 80MB
- Speed: ~3000 sentences/sec
- Quality: Good for general queries
- **Recommended for most use cases**

**Higher Quality** (`all-mpnet-base-v2`):
- Size: 420MB
- Speed: ~2000 sentences/sec
- Quality: Better semantic understanding

**Multilingual** (`paraphrase-multilingual-MiniLM-L12-v2`):
- Size: 420MB
- Languages: 50+ languages
- Use case: International tool discovery

### ChromaDB Storage

**Advantages**:
- Embedded (no separate server)
- Fast vector similarity search
- Automatic persistence
- Small footprint (~50MB for 1000 tools)

**Storage Location**: `~/.mcpproxy/chroma_db/`

### Caching Strategy

Tools and server summaries are cached in ChromaDB:
- **Tools**: Indexed once, updated on change
- **Server Summaries**: Manually updated via API
- **Embeddings**: Computed once, reused

**Sync Frequency**:
- Initial: On semantic API startup
- Manual: Via `/api/sync-tools` endpoint
- Triggered: When mcpproxy adds new tools (future enhancement)

## Comparison: Semantic vs Keyword Search

| Feature | Keyword (retrieve_tools) | Semantic (semantic_search_tools) |
|---------|-------------------------|----------------------------------|
| **Algorithm** | BM25 keyword matching | Vector embeddings + RAG |
| **Understanding** | Exact words only | Conceptual meaning |
| **Context** | No server context | Uses server documentation |
| **Disambiguation** | First match wins | Intelligent selection |
| **Speed** | Very fast (~10ms) | Fast (~50-100ms) |
| **Accuracy** | Good for exact terms | Better for natural language |
| **Use When** | You know exact tool name | Exploring, natural queries |

**Best Practice**: Use hybrid mode to get benefits of both!

## Troubleshooting

### Service Not Available

```
Error: Semantic search service not available
```

**Solution**:
1. Check if semantic API is running: `curl http://localhost:8081/health`
2. Start the service: `python3 agent/semantic_search_api.py`
3. Verify connection: Check `SEMANTIC_SEARCH_URL` environment variable

### No Tools Found

```
{
  "tools": [],
  "total": 0,
  "reasoning": "No tools found matching the query"
}
```

**Solution**:
1. Sync tools: `curl -X POST http://localhost:8081/api/sync-tools`
2. Check mcpproxy tools: `curl http://localhost:8080/api/tools/list`
3. Verify ChromaDB: Check `~/.mcpproxy/chroma_db/` exists

### Poor Quality Results

**Solution**:
1. **Index server summaries**: Provide context via `/api/index-server`
2. **Use hybrid mode**: Combines semantic + keyword for better results
3. **Adjust semantic_weight**: Try different values (0.4-0.8)
4. **Try different embedding model**: Experiment with `all-mpnet-base-v2`

### Performance Issues

**Solution**:
1. **Reduce limit**: Request fewer tools (10-15 instead of 50)
2. **Use semantic mode**: Faster than hybrid (no keyword search)
3. **Upgrade hardware**: Embedding models benefit from better CPU
4. **Consider GPU**: Use sentence-transformers with CUDA for faster encoding

## Advanced Usage

### Custom Embedding Models

```python
from mcp_agent.tools.semantic_search import SemanticSearchTools

# Use different model
tools = SemanticSearchTools(
    embedding_model="all-mpnet-base-v2"  # Higher quality
)
```

### Programmatic Access

```python
from mcp_agent.tools.semantic_search import SemanticSearchTools
from mcp_agent.graph.semantic_agent import SemanticSearchAgent, SearchRequest

# Initialize
tools = SemanticSearchTools()
agent = SemanticSearchAgent(tools)

# Search
request = SearchRequest(
    query="create GitHub issues",
    mode="hybrid",
    limit=10
)

result = await agent.search(request)

for tool in result.tools:
    print(f"{tool['tool_name']}: {tool['final_score']:.2f}")
    print(f"  Reasoning: {tool['reasoning']}")
```

### Integration with Other Agents

```python
from mcp_agent.graph.agent_graph import MCPAgentGraph
from mcp_agent.tools.semantic_search import SemanticSearchTools

# Add to existing agent
tools_registry = {
    "diagnostic": diagnostic_tools,
    "semantic": SemanticSearchTools(),  # Add semantic search
}

agent = MCPAgentGraph(tools_registry)
```

## Future Enhancements

### Planned Features
- [ ] Automatic server documentation summarization from URLs
- [ ] Real-time tool sync (webhook from mcpproxy)
- [ ] Multi-language embedding support
- [ ] Tool usage feedback for ranking improvement
- [ ] Vector search optimization with FAISS
- [ ] Cached embedding service
- [ ] Tool recommendation API
- [ ] A/B testing semantic vs keyword

## Summary

The semantic search agent provides:

✅ **Natural Language Understanding** - Query with concepts, not keywords
✅ **Server Context Awareness** - Intelligently disambiguate similar tools
✅ **RAG Pipeline** - Retrieve and rerank with documentation context
✅ **Hybrid Search** - Combine semantic and keyword for best results
✅ **Production Ready** - Fast, scalable, embeddable ChromaDB
✅ **Easy Integration** - Simple HTTP API + MCP tool

**Recommended workflow**:
1. Start with `hybrid` mode for balanced results
2. Use `context_aware` mode when servers offer similar tools
3. Index server documentation summaries for better disambiguation
4. Adjust `semantic_weight` based on your queries
5. Monitor performance and sync tools regularly

**The semantic search agent makes tool discovery intelligent, contextual, and conversational!** 🚀

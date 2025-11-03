# Integration Patch for Semantic Search Agent

This document shows the exact changes needed to integrate semantic search into MCPProxy.

## Files to Modify

### 1. `internal/server/server.go`

**Add semantic search service field** (around line 76, after `inspectorManager`):

```go
// Add to Server struct
type Server struct {
	// ... existing fields ...

	// MCP Inspector manager
	inspectorManager *InspectorManager

	// Semantic search service (ADD THIS)
	semanticSearchService *SemanticSearchService

	// Server control
	httpServer *http.Server
	// ... rest of struct ...
}
```

**Initialize semantic search service** (around line 170, after inspector manager):

```go
// Initialize MCP Inspector manager
server.inspectorManager = NewInspectorManager(logger.Sugar())

// Initialize semantic search service (ADD THIS)
semanticSearchURL := os.Getenv("SEMANTIC_SEARCH_URL")
if semanticSearchURL == "" {
	semanticSearchURL = "http://127.0.0.1:8081"
}

server.semanticSearchService = NewSemanticSearchService(semanticSearchURL, logger)

// Check if semantic search is available
if server.semanticSearchService.IsAvailable(context.Background()) {
	logger.Info("Semantic search service available", zap.String("url", semanticSearchURL))
} else {
	logger.Warn("Semantic search service not available - semantic_search_tools will not work",
		zap.String("url", semanticSearchURL),
		zap.String("help", "Start with: python3 agent/semantic_search_api.py"))
}

// Create MCP proxy server
mcpProxy := NewMCPProxyServer(storageManager, indexManager, upstreamManager, cacheManager, truncator, logger, server, cfg.DebugSearch, cfg)
```

### 2. `internal/server/mcp.go`

**Add semantic search tool** (around line 147, after `retrieve_tools`):

```go
// retrieve_tools - THE PRIMARY TOOL FOR DISCOVERING TOOLS
retrieveToolsTool := mcp.NewTool("retrieve_tools",
	// ... existing definition ...
)

// semantic_search_tools - AI-POWERED SEMANTIC SEARCH (ADD THIS)
semanticSearchTool := createSemanticSearchTool()

// call_tool - Execute tools discovered via retrieve_tools
callToolTool := mcp.NewTool("call_tool",
	// ... existing definition ...
)
```

**Add to tools array** (around line 293):

```go
tools := []mcp.Tool{
	retrieveToolsTool,
	semanticSearchTool, // ADD THIS LINE
	callToolTool,
	upstreamServersTool,
	toolsStatTool,
	// ... rest of tools ...
}
```

**Add handler case** (around line 350, in `handleToolCall`):

```go
func (p *MCPProxyServer) handleToolCall(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	toolName := request.Params.Name

	switch toolName {
	case "retrieve_tools":
		return p.handleRetrieveTools(ctx, request)

	case "semantic_search_tools": // ADD THIS CASE
		return p.handleSemanticSearch(ctx, request)

	case "call_tool":
		return p.handleCallTool(ctx, request)

	// ... rest of cases ...
	}
}
```

## Environment Variables

Add to your environment or `.env` file:

```bash
# Semantic search API URL (optional, defaults to http://127.0.0.1:8081)
export SEMANTIC_SEARCH_URL="http://127.0.0.1:8081"

# For semantic API itself
export MCPPROXY_URL="http://localhost:8080"
export EMBEDDING_MODEL="all-MiniLM-L6-v2"
export SEMANTIC_SEARCH_PORT="8081"
```

## Build & Run Steps

### 1. Install Python Dependencies

```bash
cd agent

# Install base requirements
pip install -r requirements.txt

# Install semantic search requirements
pip install -r requirements-semantic.txt
```

### 2. Start Semantic Search API (Terminal 1)

```bash
cd agent
python3 semantic_search_api.py
```

You should see:
```
🚀 Initializing semantic search service...
✓ Semantic search tools initialized
  - MCPProxy: http://localhost:8080
  - Data directory: ~/.mcpproxy
  - Embedding model: all-MiniLM-L6-v2
✓ Semantic search agent initialized
📥 Syncing tools from mcpproxy...
✓ Indexed 42 tools
INFO:     Started server process [12345]
INFO:     Waiting for application startup.
INFO:     Application startup complete.
INFO:     Uvicorn running on http://127.0.0.1:8081
```

### 3. Build MCPProxy

```bash
# From mcpproxy-go root
go build -o mcpproxy ./cmd/mcpproxy
```

### 4. Start MCPProxy (Terminal 2)

```bash
./mcpproxy serve
```

You should see:
```
INFO    Semantic search service available    {"url": "http://127.0.0.1:8081"}
INFO    MCP server listening    {"address": "stdio"}
INFO    HTTP server listening   {"address": ":8080"}
```

## Testing the Integration

### 1. Test via curl

```bash
# Health check
curl http://localhost:8081/health

# Test semantic search directly
curl -X POST http://localhost:8081/api/semantic-search \
  -H "Content-Type: application/json" \
  -d '{
    "query": "create GitHub issues",
    "mode": "hybrid",
    "limit": 5,
    "include_reasoning": true,
    "semantic_weight": 0.6
  }'
```

### 2. Test via MCP Protocol

Use the MCP test client or Claude Code:

```json
{
  "jsonrpc": "2.0",
  "method": "tools/call",
  "params": {
    "name": "semantic_search_tools",
    "arguments": {
      "query": "search files in project",
      "mode": "context_aware",
      "limit": 10,
      "include_reasoning": true
    }
  },
  "id": 1
}
```

### 3. Run Test Script

```bash
cd agent
python3 test_semantic_search.py
```

## Verification Checklist

- [ ] Semantic search API starts successfully
- [ ] MCPProxy detects semantic search service
- [ ] `semantic_search_tools` appears in tools list
- [ ] Health check returns 200 OK
- [ ] Semantic search returns results
- [ ] Server summaries can be indexed
- [ ] Tools sync from mcpproxy
- [ ] ChromaDB storage created at `~/.mcpproxy/chroma_db/`

## Troubleshooting

### Service Not Available

If you see:
```
WARN    Semantic search service not available
```

**Check**:
1. Is Python API running? `curl http://localhost:8081/health`
2. Correct port? Default is 8081
3. Firewall blocking connection?
4. Check logs: `python3 semantic_search_api.py`

### No Tools Indexed

If semantic search returns empty results:

```bash
# Manually trigger sync
curl -X POST http://localhost:8081/api/sync-tools

# Check how many tools were indexed
curl http://localhost:8081/health
```

### Import Errors

If Python API fails to start with import errors:

```bash
# Reinstall dependencies
pip install -r agent/requirements-semantic.txt

# Check sentence-transformers
python3 -c "import sentence_transformers; print('OK')"

# Check chromadb
python3 -c "import chromadb; print('OK')"
```

### Performance Issues

If searches are slow (>1 second):

1. **Use faster model**: Change to `all-MiniLM-L6-v2` (default, fastest)
2. **Reduce limit**: Request fewer tools (5-10 instead of 50)
3. **Use semantic mode**: Skip keyword search for speed
4. **Check ChromaDB**: Ensure `~/.mcpproxy/chroma_db/` is on SSD

## Integration Complete!

After applying this patch, your MCPProxy will have:

✅ **semantic_search_tools** - AI-powered tool discovery
✅ **Hybrid search** - Combines semantic + keyword
✅ **Server context** - Intelligent disambiguation
✅ **RAG pipeline** - Retrieval-augmented generation
✅ **Natural language** - Query with concepts, not keywords

The semantic search agent is now fully integrated! 🎉

# Semantic Search for MCPProxy

## Overview

MCPProxy now includes **semantic search** capabilities that complement the existing BM25 keyword search. Semantic search understands the **meaning** of your queries, not just keywords, providing more relevant tool discovery.

## Features

### 🔍 Search Modes

1. **BM25 Only** (Default)
   - Fast keyword-based search
   - Exact and fuzzy matching
   - Optimized for tool names and descriptions

2. **Semantic Only**
   - Understanding-based search
   - Finds tools by meaning, not just keywords
   - Better for natural language queries

3. **Hybrid Mode** (Recommended)
   - Combines BM25 + Semantic search
   - Best of both worlds
   - Adjustable weight between keyword and semantic

### ⚡ Performance

- **BM25**: Sub-millisecond search
- **Semantic**: ~10-50ms search
- **Hybrid**: ~20-70ms search
- **Indexing**: Both indices updated in parallel

## Configuration

### Enable Semantic Search

Add to your `mcp_config.json`:

```json
{
  "semantic_search": {
    "enabled": true,
    "hybrid_mode": true,
    "hybrid_weight": 0.5,
    "min_similarity": 0.1
  }
}
```

### Configuration Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `enabled` | boolean | `false` | Enable semantic search |
| `hybrid_mode` | boolean | `true` | Combine BM25 and semantic |
| `hybrid_weight` | float (0-1) | `0.5` | Weight for semantic search (0=BM25 only, 1=Semantic only) |
| `min_similarity` | float (0-1) | `0.1` | Minimum similarity threshold for results |

### Hybrid Weight Examples

```json
// Prefer BM25 (70% keyword, 30% semantic)
{"hybrid_weight": 0.3}

// Balanced (50% keyword, 50% semantic)
{"hybrid_weight": 0.5}

// Prefer Semantic (30% keyword, 70% semantic)
{"hybrid_weight": 0.7}
```

## How It Works

### Search Flow

```
User Query: "create a github issue"
    ↓
┌─────────────────┐     ┌──────────────────┐
│   BM25 Search   │     │ Semantic Search  │
│  (Keywords)     │     │   (Meaning)      │
└────────┬────────┘     └────────┬─────────┘
         │                       │
         │  Reciprocal Rank     │
         │     Fusion (RRF)     │
         └───────────┬───────────┘
                     ↓
         ┌──────────────────────┐
         │  Combined Results    │
         │  (Ranked by Score)   │
         └──────────────────────┘
```

### Reciprocal Rank Fusion (RRF)

The hybrid mode uses **RRF** to combine results from both search engines:

```
score = (bm25_weight / (60 + rank_bm25)) + (semantic_weight / (60 + rank_semantic))
```

This ensures:
- Top results from both engines are prioritized
- No single engine dominates
- Rank position matters more than raw scores

## Usage Examples

### Basic Search

```bash
# With semantic search enabled
mcpproxy call tool --tool-name=retrieve_tools \
  --json_args='{"query":"create a github issue","limit":10}'
```

### Debug Mode

```bash
# See which search backend is being used
mcpproxy call tool --tool-name=retrieve_tools \
  --json_args='{"query":"weather api","limit":10,"debug":true}'
```

**Debug Output**:
```json
{
  "tools": [...],
  "debug": {
    "search_backend": "Hybrid (BM25 + Semantic, weight: 0.50)",
    "semantic_enabled": true,
    "total_indexed_tools": 150
  }
}
```

### Query Comparison

| Query Type | BM25 Result | Semantic Result |
|------------|-------------|-----------------|
| "github create issue" | ✅ Exact match | ✅ Understands intent |
| "make a bug report on github" | ❌ No keyword match | ✅ Understands meaning |
| "create_issue" | ✅ Exact tool name | ⚠️ May rank lower |

**Hybrid mode combines both strengths!**

## Architecture

### Components

1. **Embedding Service** (`internal/semantic/embeddings.go`)
   - Converts text to vector embeddings
   - TF-IDF based (native Go implementation)
   - 384-dimensional vectors

2. **Semantic Index** (`internal/semantic/index.go`)
   - Stores tool embeddings
   - Cosine similarity search
   - JSON persistence

3. **Index Manager** (`internal/index/manager.go`)
   - Coordinates BM25 and semantic indices
   - Handles hybrid search logic
   - RRF score combination

### Data Flow

```
Tool Added
    ↓
IndexTool()
    ├─→ BM25 Index (Bleve)
    └─→ Semantic Index (Embeddings + Cosine Similarity)

Search Query
    ↓
SearchTools()
    ├─→ BM25 Search (keyword matching)
    ├─→ Semantic Search (meaning understanding)
    └─→ Hybrid Fusion (RRF combination)
    ↓
Ranked Results
```

## Performance Considerations

### Indexing

- **BM25**: ~1ms per tool
- **Semantic**: ~5ms per tool (embedding generation)
- **Batch Indexing**: Parallelized for efficiency

### Search

- **BM25**: <1ms for most queries
- **Semantic**: 10-50ms depending on index size
- **Hybrid**: Sum of both + fusion overhead (~5ms)

### Storage

- **BM25 Index**: `.bleve/` directory (~1MB per 1000 tools)
- **Semantic Index**: `semantic_index.json` (~500KB per 1000 tools)

## Advanced Configuration

### Disable for Specific Queries

Semantic search is automatically used for all queries when enabled. To disable temporarily, use BM25-optimized queries:

```bash
# Exact tool name (BM25 excels here)
mcpproxy call tool --tool-name=retrieve_tools \
  --json_args='{"query":"github:create_issue"}'
```

### Tuning Similarity Threshold

Adjust `min_similarity` to control result quality:

```json
// Strict: Only very similar results
{"min_similarity": 0.5}

// Relaxed: More diverse results
{"min_similarity": 0.1}

// No filter: All results
{"min_similarity": 0.0}
```

## Troubleshooting

### Semantic Search Not Working

1. **Check Configuration**:
   ```bash
   cat ~/.mcpproxy/mcp_config.json | grep -A5 semantic_search
   ```

2. **Verify Index Created**:
   ```bash
   ls -lh ~/.mcpproxy/semantic_index.json
   ```

3. **Check Logs**:
   ```bash
   tail -f ~/Library/Logs/mcpproxy/main.log | grep -i semantic
   ```

### Poor Search Results

1. **Try Different Hybrid Weights**:
   - Start with 0.5 (balanced)
   - Increase for more semantic influence
   - Decrease for more keyword influence

2. **Adjust Minimum Similarity**:
   - Lower threshold for more results
   - Higher threshold for better quality

3. **Rebuild Index**:
   ```bash
   rm ~/.mcpproxy/semantic_index.json
   # Restart mcpproxy - index will rebuild automatically
   ```

## Migration Guide

### From BM25-Only Setup

1. **Add Configuration** to `mcp_config.json`:
   ```json
   {
     "semantic_search": {
       "enabled": true,
       "hybrid_mode": true,
       "hybrid_weight": 0.5
     }
   }
   ```

2. **Restart MCPProxy**:
   ```bash
   pkill mcpproxy
   ./mcpproxy serve
   ```

3. **Verify Activation**:
   ```bash
   mcpproxy call tool --tool-name=retrieve_tools \
     --json_args='{"query":"test","limit":1,"debug":true}' \
     | grep search_backend
   ```

### Reverting to BM25-Only

Simply set `"enabled": false` in config:

```json
{
  "semantic_search": {
    "enabled": false
  }
}
```

## Future Enhancements

### Planned Features

- [ ] **Neural Embeddings**: Integration with sentence-transformers via HTTP
- [ ] **Multilingual Support**: Cross-language tool discovery
- [ ] **User Feedback**: Learning from search behavior
- [ ] **Query Expansion**: Automatic synonym and related term expansion
- [ ] **Caching**: Embedding cache for frequently searched terms

### Integration Opportunities

- **External Embedding Services**: Connect to existing embedding APIs
- **Vector Databases**: FAISS, Pinecone, Weaviate integration
- **Custom Models**: Fine-tune embeddings on your tool corpus

## References

- **BM25 Algorithm**: [Wikipedia](https://en.wikipedia.org/wiki/Okapi_BM25)
- **Reciprocal Rank Fusion**: [Paper](https://plg.uwaterloo.ca/~gvcormac/cormacksigir09-rrf.pdf)
- **Cosine Similarity**: [Wikipedia](https://en.wikipedia.org/wiki/Cosine_similarity)
- **TF-IDF**: [Wikipedia](https://en.wikipedia.org/wiki/Tf%E2%80%93idf)

---

**Questions or Feedback?** Open an issue on GitHub!

package index

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"mcpproxy-go/internal/config"
	"mcpproxy-go/internal/semantic"

	"go.uber.org/zap"
)

// Manager provides a unified interface for indexing operations
type Manager struct {
	bleveIndex    *BleveIndex
	semanticIndex *semantic.SemanticIndex
	mu            sync.RWMutex
	logger        *zap.Logger
	config        *config.SemanticSearchConfig
}

// NewManager creates a new index manager
func NewManager(dataDir string, logger *zap.Logger, semanticConfig *config.SemanticSearchConfig) (*Manager, error) {
	bleveIndex, err := NewBleveIndex(dataDir, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create Bleve index: %w", err)
	}

	// Create semantic index if enabled
	var semanticIndex *semantic.SemanticIndex
	if semanticConfig != nil && semanticConfig.Enabled {
		semanticIndex, err = semantic.NewSemanticIndex(dataDir, logger, true)
		if err != nil {
			logger.Warn("Failed to create semantic index, falling back to BM25 only",
				zap.Error(err))
			semanticIndex = nil
		}
	}

	return &Manager{
		bleveIndex:    bleveIndex,
		semanticIndex: semanticIndex,
		logger:        logger,
		config:        semanticConfig,
	}, nil
}

// Close closes the index manager
func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var err error
	if m.bleveIndex != nil {
		err = m.bleveIndex.Close()
	}
	if m.semanticIndex != nil {
		if semanticErr := m.semanticIndex.Close(); semanticErr != nil {
			m.logger.Warn("Failed to close semantic index", zap.Error(semanticErr))
		}
	}
	return err
}

// IndexTool indexes a single tool in both BM25 and semantic indices
func (m *Manager) IndexTool(toolMeta *config.ToolMetadata) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Index in BM25 (primary index)
	if err := m.bleveIndex.IndexTool(toolMeta); err != nil {
		return err
	}

	// Index in semantic index if enabled
	if m.semanticIndex != nil && m.semanticIndex.IsEnabled() {
		ctx := context.Background()
		if err := m.semanticIndex.IndexTool(ctx, toolMeta); err != nil {
			m.logger.Warn("Failed to index tool in semantic index",
				zap.String("tool", toolMeta.Name),
				zap.Error(err))
			// Don't fail the entire indexing operation
		}
	}

	return nil
}

// BatchIndexTools indexes multiple tools efficiently in both indices
func (m *Manager) BatchIndexTools(tools []*config.ToolMetadata) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Index in BM25 (primary index)
	if err := m.bleveIndex.BatchIndex(tools); err != nil {
		return err
	}

	// Index in semantic index if enabled
	if m.semanticIndex != nil && m.semanticIndex.IsEnabled() {
		ctx := context.Background()
		if err := m.semanticIndex.BatchIndexTools(ctx, tools); err != nil {
			m.logger.Warn("Failed to batch index tools in semantic index",
				zap.Int("count", len(tools)),
				zap.Error(err))
			// Don't fail the entire indexing operation
		}
	}

	return nil
}

// SearchTools searches for tools matching the query with optional semantic search
func (m *Manager) SearchTools(query string, limit int) ([]*config.SearchResult, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if limit <= 0 {
		limit = 20 // default limit
	}

	// Check if semantic search is enabled and hybrid mode is active
	useSemanticSearch := m.semanticIndex != nil &&
		m.semanticIndex.IsEnabled() &&
		m.config != nil &&
		m.config.Enabled

	if !useSemanticSearch {
		// Fall back to BM25 only
		return m.bleveIndex.SearchTools(query, limit)
	}

	// Perform hybrid search (BM25 + Semantic)
	if m.config.HybridMode {
		return m.hybridSearch(query, limit)
	}

	// Semantic search only
	ctx := context.Background()
	semanticResults, err := m.semanticIndex.Search(ctx, query, limit)
	if err != nil {
		m.logger.Warn("Semantic search failed, falling back to BM25",
			zap.Error(err))
		return m.bleveIndex.SearchTools(query, limit)
	}

	// Convert semantic results to config.SearchResult
	results := make([]*config.SearchResult, len(semanticResults))
	for i, sr := range semanticResults {
		results[i] = &config.SearchResult{
			Tool:  sr.Tool,
			Score: float64(sr.Score),
		}
	}

	return results, nil
}

// hybridSearch combines BM25 and semantic search results
func (m *Manager) hybridSearch(query string, limit int) ([]*config.SearchResult, error) {
	// Get more results from both engines to ensure good coverage
	searchLimit := limit * 3

	// Get BM25 results
	bm25Results, err := m.bleveIndex.SearchTools(query, searchLimit)
	if err != nil {
		return nil, fmt.Errorf("BM25 search failed: %w", err)
	}

	// Get semantic results
	ctx := context.Background()
	semanticResults, err := m.semanticIndex.Search(ctx, query, searchLimit)
	if err != nil {
		m.logger.Warn("Semantic search failed in hybrid mode, using BM25 only",
			zap.Error(err))
		// Return BM25 results limited to requested limit
		if len(bm25Results) > limit {
			return bm25Results[:limit], nil
		}
		return bm25Results, nil
	}

	// Combine results using reciprocal rank fusion (RRF)
	combined := m.reciprocalRankFusion(bm25Results, semanticResults, limit)

	// Filter by minimum similarity if set
	if m.config.MinSimilarity > 0 {
		filtered := make([]*config.SearchResult, 0, len(combined))
		for _, result := range combined {
			if result.Score >= float64(m.config.MinSimilarity) {
				filtered = append(filtered, result)
			}
		}
		combined = filtered
	}

	m.logger.Debug("Hybrid search completed",
		zap.String("query", query),
		zap.Int("bm25_results", len(bm25Results)),
		zap.Int("semantic_results", len(semanticResults)),
		zap.Int("combined_results", len(combined)),
		zap.Float64("hybrid_weight", m.config.HybridWeight))

	return combined, nil
}

// reciprocalRankFusion combines results from multiple search engines
// RRF formula: score = sum(1 / (k + rank)) where k is a constant (typically 60)
func (m *Manager) reciprocalRankFusion(bm25Results []*config.SearchResult, semanticResults []*semantic.SearchResult, limit int) []*config.SearchResult {
	const k = 60.0 // RRF constant

	// Create score map by tool name
	scores := make(map[string]float32)
	toolMap := make(map[string]*config.ToolMetadata)

	// Add BM25 scores (weighted by 1 - hybridWeight)
	bm25Weight := float32(1.0 - m.config.HybridWeight)
	for rank, result := range bm25Results {
		score := bm25Weight / (k + float32(rank+1))
		scores[result.Tool.Name] = score
		toolMap[result.Tool.Name] = result.Tool
	}

	// Add semantic scores (weighted by hybridWeight)
	semanticWeight := float32(m.config.HybridWeight)
	for rank, result := range semanticResults {
		score := semanticWeight / (k + float32(rank+1))
		if existing, ok := scores[result.Tool.Name]; ok {
			scores[result.Tool.Name] = existing + score
		} else {
			scores[result.Tool.Name] = score
			toolMap[result.Tool.Name] = result.Tool
		}
	}

	// Convert to slice and sort by combined score
	type scoredTool struct {
		name  string
		score float32
		tool  *config.ToolMetadata
	}

	scoredTools := make([]scoredTool, 0, len(scores))
	for name, score := range scores {
		scoredTools = append(scoredTools, scoredTool{
			name:  name,
			score: score,
			tool:  toolMap[name],
		})
	}

	// Sort by score (descending)
	sort.Slice(scoredTools, func(i, j int) bool {
		return scoredTools[i].score > scoredTools[j].score
	})

	// Limit results
	if len(scoredTools) > limit {
		scoredTools = scoredTools[:limit]
	}

	// Convert to SearchResult
	results := make([]*config.SearchResult, len(scoredTools))
	for i, st := range scoredTools {
		results[i] = &config.SearchResult{
			Tool:  st.tool,
			Score: float64(st.score),
		}
	}

	return results
}

// Search searches for tools matching the query (alias for SearchTools)
func (m *Manager) Search(query string, limit int) ([]*config.SearchResult, error) {
	return m.SearchTools(query, limit)
}

// DeleteTool removes a tool from both indices
func (m *Manager) DeleteTool(serverName, toolName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Delete from BM25 index
	if err := m.bleveIndex.DeleteTool(serverName, toolName); err != nil {
		return err
	}

	// Delete from semantic index if enabled
	if m.semanticIndex != nil && m.semanticIndex.IsEnabled() {
		if err := m.semanticIndex.DeleteTool(serverName, toolName); err != nil {
			m.logger.Warn("Failed to delete tool from semantic index",
				zap.String("server", serverName),
				zap.String("tool", toolName),
				zap.Error(err))
		}
	}

	return nil
}

// DeleteServerTools removes all tools from a specific server in both indices
func (m *Manager) DeleteServerTools(serverName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Delete from BM25 index
	if err := m.bleveIndex.DeleteServerTools(serverName); err != nil {
		return err
	}

	// Delete from semantic index if enabled
	if m.semanticIndex != nil && m.semanticIndex.IsEnabled() {
		if err := m.semanticIndex.DeleteServerTools(serverName); err != nil {
			m.logger.Warn("Failed to delete server tools from semantic index",
				zap.String("server", serverName),
				zap.Error(err))
		}
	}

	return nil
}

// GetDocumentCount returns the number of indexed documents
func (m *Manager) GetDocumentCount() (uint64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.bleveIndex.GetDocumentCount()
}

// RebuildIndex rebuilds the entire index
func (m *Manager) RebuildIndex() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.bleveIndex.RebuildIndex()
}

// GetStats returns indexing statistics
func (m *Manager) GetStats() (map[string]interface{}, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	docCount, err := m.bleveIndex.GetDocumentCount()
	if err != nil {
		return nil, err
	}

	stats := map[string]interface{}{
		"document_count": docCount,
		"index_type":     "bleve",
		"search_backend": "BM25",
	}

	return stats, nil
}

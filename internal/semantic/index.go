package semantic

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"go.uber.org/zap"
	"mcpproxy-go/internal/config"
)

// SemanticIndex manages semantic search over tools
type SemanticIndex struct {
	embedding  *EmbeddingService
	logger     *zap.Logger
	dataDir    string
	documents  map[string]*EmbeddingDocument
	mu         sync.RWMutex
	indexPath  string
	enabled    bool
}

// SearchResult represents a semantic search result
type SearchResult struct {
	Tool       *config.ToolMetadata
	Score      float32
	Similarity float32
}

// NewSemanticIndex creates a new semantic index
func NewSemanticIndex(dataDir string, logger *zap.Logger, enabled bool) (*SemanticIndex, error) {
	if !enabled {
		logger.Info("Semantic search is disabled")
		return &SemanticIndex{
			logger:  logger,
			enabled: false,
		}, nil
	}

	embedding, err := NewEmbeddingService(logger)
	if err != nil {
		logger.Warn("Failed to create embedding service, semantic search disabled",
			zap.Error(err))
		return &SemanticIndex{
			logger:  logger,
			enabled: false,
		}, nil
	}

	indexPath := filepath.Join(dataDir, "semantic_index.json")

	idx := &SemanticIndex{
		embedding:  embedding,
		logger:     logger,
		dataDir:    dataDir,
		documents:  make(map[string]*EmbeddingDocument),
		indexPath:  indexPath,
		enabled:    true,
	}

	// Try to load existing index
	if err := idx.Load(); err != nil {
		logger.Info("No existing semantic index found, will create new one",
			zap.String("path", indexPath))
	} else {
		logger.Info("Loaded existing semantic index",
			zap.Int("documents", len(idx.documents)),
			zap.String("path", indexPath))
	}

	return idx, nil
}

// IsEnabled returns whether semantic search is enabled
func (idx *SemanticIndex) IsEnabled() bool {
	return idx.enabled
}

// IndexTool indexes a tool for semantic search
func (idx *SemanticIndex) IndexTool(ctx context.Context, tool *config.ToolMetadata) error {
	if !idx.enabled {
		return nil
	}

	idx.mu.Lock()
	defer idx.mu.Unlock()

	// Create searchable text combining all relevant fields
	searchableText := fmt.Sprintf("%s %s %s",
		tool.Name,
		tool.Description,
		tool.ParamsJSON)

	// Generate embedding
	embedding, err := idx.embedding.Embed(ctx, searchableText)
	if err != nil {
		return fmt.Errorf("failed to generate embedding: %w", err)
	}

	// Create document ID
	docID := fmt.Sprintf("%s:%s", tool.ServerName, tool.Name)

	// Store document
	idx.documents[docID] = &EmbeddingDocument{
		ID:        docID,
		Text:      searchableText,
		Embedding: embedding,
		Metadata: map[string]interface{}{
			"server_name": tool.ServerName,
			"tool_name":   tool.Name,
			"description": tool.Description,
			"params_json": tool.ParamsJSON,
			"hash":        tool.Hash,
		},
	}

	idx.logger.Debug("Indexed tool for semantic search",
		zap.String("tool", docID),
		zap.Int("embedding_dim", len(embedding)))

	return nil
}

// BatchIndexTools indexes multiple tools efficiently
func (idx *SemanticIndex) BatchIndexTools(ctx context.Context, tools []*config.ToolMetadata) error {
	if !idx.enabled {
		return nil
	}

	for _, tool := range tools {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if err := idx.IndexTool(ctx, tool); err != nil {
				idx.logger.Warn("Failed to index tool",
					zap.String("tool", tool.Name),
					zap.Error(err))
				// Continue with other tools
			}
		}
	}

	// Save index after batch indexing
	if err := idx.Save(); err != nil {
		idx.logger.Warn("Failed to save semantic index after batch indexing",
			zap.Error(err))
	}

	return nil
}

// Search performs semantic search
func (idx *SemanticIndex) Search(ctx context.Context, query string, limit int) ([]*SearchResult, error) {
	if !idx.enabled {
		return nil, fmt.Errorf("semantic search is disabled")
	}

	idx.mu.RLock()
	defer idx.mu.RUnlock()

	if len(idx.documents) == 0 {
		return []*SearchResult{}, nil
	}

	// Generate query embedding
	queryEmbedding, err := idx.embedding.Embed(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	// Calculate similarities
	type scoredDoc struct {
		doc        *EmbeddingDocument
		similarity float32
	}
	scoredDocs := make([]scoredDoc, 0, len(idx.documents))

	for _, doc := range idx.documents {
		similarity := CosineSimilarity(queryEmbedding, doc.Embedding)
		scoredDocs = append(scoredDocs, scoredDoc{
			doc:        doc,
			similarity: similarity,
		})
	}

	// Sort by similarity (descending)
	sort.Slice(scoredDocs, func(i, j int) bool {
		return scoredDocs[i].similarity > scoredDocs[j].similarity
	})

	// Limit results
	if limit > len(scoredDocs) {
		limit = len(scoredDocs)
	}
	scoredDocs = scoredDocs[:limit]

	// Convert to search results
	results := make([]*SearchResult, len(scoredDocs))
	for i, sd := range scoredDocs {
		// Extract metadata
		toolName, _ := sd.doc.Metadata["tool_name"].(string)
		serverName, _ := sd.doc.Metadata["server_name"].(string)
		description, _ := sd.doc.Metadata["description"].(string)
		paramsJSON, _ := sd.doc.Metadata["params_json"].(string)
		hash, _ := sd.doc.Metadata["hash"].(string)

		fullToolName := fmt.Sprintf("%s:%s", serverName, toolName)

		results[i] = &SearchResult{
			Tool: &config.ToolMetadata{
				Name:        fullToolName,
				ServerName:  serverName,
				Description: description,
				ParamsJSON:  paramsJSON,
				Hash:        hash,
			},
			Score:      sd.similarity, // Use similarity as score
			Similarity: sd.similarity,
		}
	}

	idx.logger.Debug("Semantic search completed",
		zap.String("query", query),
		zap.Int("results", len(results)),
		zap.Int("total_docs", len(idx.documents)))

	return results, nil
}

// DeleteTool removes a tool from the index
func (idx *SemanticIndex) DeleteTool(serverName, toolName string) error {
	if !idx.enabled {
		return nil
	}

	idx.mu.Lock()
	defer idx.mu.Unlock()

	docID := fmt.Sprintf("%s:%s", serverName, toolName)
	delete(idx.documents, docID)

	idx.logger.Debug("Deleted tool from semantic index", zap.String("tool", docID))
	return nil
}

// DeleteServerTools removes all tools from a server
func (idx *SemanticIndex) DeleteServerTools(serverName string) error {
	if !idx.enabled {
		return nil
	}

	idx.mu.Lock()
	defer idx.mu.Unlock()

	count := 0
	for docID := range idx.documents {
		// Extract server name from docID (format: "server:tool")
		if extractServerName(docID) == serverName {
			delete(idx.documents, docID)
			count++
		}
	}

	idx.logger.Info("Deleted tools from semantic index",
		zap.String("server", serverName),
		zap.Int("count", count))

	return nil
}

// GetDocumentCount returns the number of indexed documents
func (idx *SemanticIndex) GetDocumentCount() int {
	if !idx.enabled {
		return 0
	}

	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return len(idx.documents)
}

// Save persists the index to disk
func (idx *SemanticIndex) Save() error {
	if !idx.enabled {
		return nil
	}

	idx.mu.RLock()
	defer idx.mu.RUnlock()

	// Marshal documents to JSON
	data, err := json.MarshalIndent(idx.documents, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal index: %w", err)
	}

	// Write to file
	if err := os.WriteFile(idx.indexPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write index file: %w", err)
	}

	idx.logger.Debug("Saved semantic index",
		zap.String("path", idx.indexPath),
		zap.Int("documents", len(idx.documents)))

	return nil
}

// Load loads the index from disk
func (idx *SemanticIndex) Load() error {
	if !idx.enabled {
		return nil
	}

	// Read file
	data, err := os.ReadFile(idx.indexPath)
	if err != nil {
		return fmt.Errorf("failed to read index file: %w", err)
	}

	// Unmarshal documents
	documents := make(map[string]*EmbeddingDocument)
	if err := json.Unmarshal(data, &documents); err != nil {
		return fmt.Errorf("failed to unmarshal index: %w", err)
	}

	idx.mu.Lock()
	idx.documents = documents
	idx.mu.Unlock()

	idx.logger.Debug("Loaded semantic index",
		zap.String("path", idx.indexPath),
		zap.Int("documents", len(documents)))

	return nil
}

// Close cleans up resources
func (idx *SemanticIndex) Close() error {
	if !idx.enabled {
		return nil
	}

	// Save index before closing
	if err := idx.Save(); err != nil {
		idx.logger.Warn("Failed to save index during close", zap.Error(err))
	}

	if idx.embedding != nil {
		return idx.embedding.Close()
	}

	return nil
}

// extractServerName extracts server name from docID (format: "server:tool")
func extractServerName(docID string) string {
	parts := splitOnce(docID, ":")
	if len(parts) == 2 {
		return parts[0]
	}
	return ""
}

// splitOnce splits a string on the first occurrence of sep
func splitOnce(s, sep string) []string {
	idx := 0
	for i, ch := range s {
		if string(ch) == sep {
			idx = i
			break
		}
	}
	if idx == 0 {
		return []string{s}
	}
	return []string{s[:idx], s[idx+1:]}
}

package semantic

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"sync"

	"go.uber.org/zap"
)

// EmbeddingService provides text embedding functionality
type EmbeddingService struct {
	logger     *zap.Logger
	dimension  int
	mu         sync.RWMutex
	modelReady bool
}

// NewEmbeddingService creates a new embedding service
// Uses a simple but effective TF-IDF based approach for Go-native implementation
func NewEmbeddingService(logger *zap.Logger) (*EmbeddingService, error) {
	svc := &EmbeddingService{
		logger:     logger,
		dimension:  384, // Standard dimension for sentence transformers
		modelReady: true,
	}

	logger.Info("Embedding service initialized",
		zap.Int("dimension", svc.dimension),
		zap.String("model", "TF-IDF with cosine similarity"))

	return svc, nil
}

// Embed converts text to a vector representation
func (s *EmbeddingService) Embed(ctx context.Context, text string) ([]float32, error) {
	s.mu.RLock()
	if !s.modelReady {
		s.mu.RUnlock()
		return nil, fmt.Errorf("embedding model not ready")
	}
	s.mu.RUnlock()

	// Simple but effective embedding: TF-IDF style vector
	// In production, you'd use sentence-transformers via HTTP or gRPC
	embedding := s.createSimpleEmbedding(text)

	return embedding, nil
}

// BatchEmbed embeds multiple texts efficiently
func (s *EmbeddingService) BatchEmbed(ctx context.Context, texts []string) ([][]float32, error) {
	embeddings := make([][]float32, len(texts))

	for i, text := range texts {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			embedding, err := s.Embed(ctx, text)
			if err != nil {
				return nil, fmt.Errorf("failed to embed text %d: %w", i, err)
			}
			embeddings[i] = embedding
		}
	}

	return embeddings, nil
}

// GetDimension returns the embedding dimension
func (s *EmbeddingService) GetDimension() int {
	return s.dimension
}

// IsReady returns whether the service is ready
func (s *EmbeddingService) IsReady() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.modelReady
}

// Close cleans up resources
func (s *EmbeddingService) Close() error {
	s.mu.Lock()
	s.modelReady = false
	s.mu.Unlock()
	return nil
}

// createSimpleEmbedding creates a simple TF-IDF style embedding
// This is a placeholder for a real embedding model (like sentence-transformers)
func (s *EmbeddingService) createSimpleEmbedding(text string) []float32 {
	// Normalize and tokenize
	tokens := s.tokenize(text)

	// Create frequency map
	freq := make(map[string]int)
	for _, token := range tokens {
		freq[token]++
	}

	// Create fixed-size vector
	embedding := make([]float32, s.dimension)

	// Use hash-based positioning for each token
	for token, count := range freq {
		hash := s.hashToken(token)
		for i := 0; i < 3; i++ { // Use multiple positions for robustness
			pos := (hash + i*17) % s.dimension
			embedding[pos] += float32(count) / float32(len(tokens))
		}
	}

	// Normalize the vector
	return s.normalize(embedding)
}

// tokenize splits text into tokens
func (s *EmbeddingService) tokenize(text string) []string {
	text = strings.ToLower(text)
	// Simple tokenization: split on non-alphanumeric
	tokens := strings.FieldsFunc(text, func(r rune) bool {
		return !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'))
	})
	return tokens
}

// hashToken creates a hash for a token
func (s *EmbeddingService) hashToken(token string) int {
	hash := 0
	for _, ch := range token {
		hash = hash*31 + int(ch)
	}
	if hash < 0 {
		hash = -hash
	}
	return hash
}

// normalize normalizes a vector to unit length
func (s *EmbeddingService) normalize(vec []float32) []float32 {
	var sum float32
	for _, v := range vec {
		sum += v * v
	}

	if sum == 0 {
		return vec
	}

	norm := float32(math.Sqrt(float64(sum)))
	normalized := make([]float32, len(vec))
	for i, v := range vec {
		normalized[i] = v / norm
	}

	return normalized
}

// CosineSimilarity computes cosine similarity between two vectors
func CosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0
	}

	var dotProduct, normA, normB float32
	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / float32(math.Sqrt(float64(normA)*float64(normB)))
}

// EmbeddingDocument represents a document with its embedding
type EmbeddingDocument struct {
	ID        string                 `json:"id"`
	Text      string                 `json:"text"`
	Embedding []float32              `json:"embedding"`
	Metadata  map[string]interface{} `json:"metadata"`
}

// MarshalJSON implements custom JSON marshaling
func (e *EmbeddingDocument) MarshalJSON() ([]byte, error) {
	type Alias EmbeddingDocument
	return json.Marshal(&struct {
		*Alias
		EmbeddingSize int `json:"embedding_size"`
	}{
		Alias:         (*Alias)(e),
		EmbeddingSize: len(e.Embedding),
	})
}

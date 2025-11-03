package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// SemanticSearchService provides semantic search capabilities via HTTP API
type SemanticSearchService struct {
	baseURL    string
	httpClient *http.Client
	logger     *zap.Logger
}

// HealthResponse represents the health check response from semantic search API
type HealthResponse struct {
	Status        string `json:"status"`
	ToolsIndexed  int    `json:"tools_indexed"`
	ServersIndexed int   `json:"servers_indexed"`
}

// NewSemanticSearchService creates a new semantic search service client
func NewSemanticSearchService(baseURL string, logger *zap.Logger) *SemanticSearchService {
	return &SemanticSearchService{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		logger: logger,
	}
}

// IsAvailable checks if the semantic search service is available and healthy
func (s *SemanticSearchService) IsAvailable(ctx context.Context) bool {
	if s == nil {
		return false
	}

	// Create request with context
	req, err := http.NewRequestWithContext(ctx, "GET", s.baseURL+"/health", nil)
	if err != nil {
		s.logger.Debug("Failed to create health check request",
			zap.Error(err),
			zap.String("url", s.baseURL))
		return false
	}

	// Perform health check
	resp, err := s.httpClient.Do(req)
	if err != nil {
		s.logger.Debug("Semantic search health check failed",
			zap.Error(err),
			zap.String("url", s.baseURL))
		return false
	}
	defer resp.Body.Close()

	// Check HTTP status
	if resp.StatusCode != http.StatusOK {
		s.logger.Debug("Semantic search health check returned non-OK status",
			zap.Int("status", resp.StatusCode),
			zap.String("url", s.baseURL))
		return false
	}

	// Parse response
	var health HealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		s.logger.Debug("Failed to parse health check response",
			zap.Error(err),
			zap.String("url", s.baseURL))
		return false
	}

	// Check status field
	if health.Status != "healthy" {
		s.logger.Debug("Semantic search service reports unhealthy status",
			zap.String("status", health.Status),
			zap.String("url", s.baseURL))
		return false
	}

	s.logger.Debug("Semantic search service is healthy",
		zap.String("url", s.baseURL),
		zap.Int("tools_indexed", health.ToolsIndexed),
		zap.Int("servers_indexed", health.ServersIndexed))

	return true
}

// GetBaseURL returns the base URL of the semantic search service
func (s *SemanticSearchService) GetBaseURL() string {
	if s == nil {
		return ""
	}
	return s.baseURL
}

// Close cleans up resources (currently no-op, reserved for future use)
func (s *SemanticSearchService) Close() error {
	// Future: close any persistent connections if needed
	return nil
}

// String returns a string representation of the service
func (s *SemanticSearchService) String() string {
	if s == nil {
		return "<nil SemanticSearchService>"
	}
	return fmt.Sprintf("SemanticSearchService{baseURL: %s}", s.baseURL)
}

package managed

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sync"
	"time"

	"mcpproxy-go/internal/config"
)

// CachedTools represents cached tool data with metadata
type CachedTools struct {
	Tools     []*config.ToolMetadata
	FetchedAt time.Time
	Hash      string // Hash of tool list for invalidation
}

// ToolCache manages cached tool lists with TTL and invalidation
type ToolCache struct {
	mu    sync.RWMutex
	cache map[string]*CachedTools
	ttl   time.Duration
}

// NewToolCache creates a new tool cache with specified TTL
func NewToolCache(ttl time.Duration) *ToolCache {
	if ttl <= 0 {
		ttl = 5 * time.Minute // Default 5 minutes
	}

	return &ToolCache{
		cache: make(map[string]*CachedTools),
		ttl:   ttl,
	}
}

// Get retrieves cached tools if valid
func (tc *ToolCache) Get(serverID string) ([]*config.ToolMetadata, bool) {
	tc.mu.RLock()
	defer tc.mu.RUnlock()

	cached, ok := tc.cache[serverID]
	if !ok {
		return nil, false
	}

	// Check TTL
	if time.Since(cached.FetchedAt) > tc.ttl {
		return nil, false
	}

	return cached.Tools, true
}

// Set stores tools in cache with current timestamp
func (tc *ToolCache) Set(serverID string, tools []*config.ToolMetadata) {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	hash := tc.hashTools(tools)
	tc.cache[serverID] = &CachedTools{
		Tools:     tools,
		FetchedAt: time.Now(),
		Hash:      hash,
	}
}

// Invalidate removes cached tools for a server
func (tc *ToolCache) Invalidate(serverID string) {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	delete(tc.cache, serverID)
}

// InvalidateAll clears all cached tools
func (tc *ToolCache) InvalidateAll() {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	tc.cache = make(map[string]*CachedTools)
}

// IsStale checks if cached tools have changed (based on hash)
func (tc *ToolCache) IsStale(serverID string, newTools []*config.ToolMetadata) bool {
	tc.mu.RLock()
	defer tc.mu.RUnlock()

	cached, ok := tc.cache[serverID]
	if !ok {
		return true
	}

	newHash := tc.hashTools(newTools)
	return cached.Hash != newHash
}

// hashTools creates a hash of the tool list for change detection
func (tc *ToolCache) hashTools(tools []*config.ToolMetadata) string {
	// Create a deterministic representation of tools
	type toolSignature struct {
		Name        string
		Description string
		ParamsJSON  string
	}

	signatures := make([]toolSignature, len(tools))
	for i, tool := range tools {
		signatures[i] = toolSignature{
			Name:        tool.Name,
			Description: tool.Description,
			ParamsJSON:  tool.ParamsJSON,
		}
	}

	// Convert to JSON for hashing
	data, err := json.Marshal(signatures)
	if err != nil {
		return ""
	}

	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// GetStats returns cache statistics
func (tc *ToolCache) GetStats() map[string]interface{} {
	tc.mu.RLock()
	defer tc.mu.RUnlock()

	stats := make(map[string]interface{})
	stats["total_cached"] = len(tc.cache)
	stats["ttl_seconds"] = tc.ttl.Seconds()

	// Count valid vs expired entries
	valid := 0
	expired := 0
	now := time.Now()

	for _, cached := range tc.cache {
		if now.Sub(cached.FetchedAt) <= tc.ttl {
			valid++
		} else {
			expired++
		}
	}

	stats["valid_entries"] = valid
	stats["expired_entries"] = expired

	return stats
}

// CleanupExpired removes expired entries from cache
func (tc *ToolCache) CleanupExpired() int {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	removed := 0
	now := time.Now()

	for serverID, cached := range tc.cache {
		if now.Sub(cached.FetchedAt) > tc.ttl {
			delete(tc.cache, serverID)
			removed++
		}
	}

	return removed
}

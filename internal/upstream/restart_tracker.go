// Package upstream provides upstream connection management.
// MED-005: RestartTracker detects and prevents restart loops.
package upstream

import (
	"sync"
	"time"

	"go.uber.org/zap"
)

// RestartTrackerConfig configures restart loop detection
type RestartTrackerConfig struct {
	// MaxRestarts is the maximum number of restarts allowed within the time window
	MaxRestarts int
	// TimeWindow is the time window for counting restarts
	TimeWindow time.Duration
	// CooldownPeriod is the minimum time between restart attempts after hitting the limit
	CooldownPeriod time.Duration
}

// DefaultRestartTrackerConfig returns the default configuration
func DefaultRestartTrackerConfig() RestartTrackerConfig {
	return RestartTrackerConfig{
		MaxRestarts:    3,
		TimeWindow:     5 * time.Minute,
		CooldownPeriod: 10 * time.Minute,
	}
}

// restartRecord tracks a single restart event
type restartRecord struct {
	Timestamp time.Time
	Reason    string
}

// serverRestartState tracks restart state for a single server
type serverRestartState struct {
	Restarts       []restartRecord
	LoopDetected   bool
	LoopDetectedAt time.Time
	TotalRestarts  int64 // Lifetime counter
}

// RestartTracker tracks restart events and detects restart loops
type RestartTracker struct {
	mu      sync.RWMutex
	config  RestartTrackerConfig
	servers map[string]*serverRestartState
	logger  *zap.Logger

	// Callback when restart loop is detected
	onLoopDetected func(serverName string, restartCount int, window time.Duration)
}

// NewRestartTracker creates a new restart tracker
func NewRestartTracker(logger *zap.Logger, config RestartTrackerConfig) *RestartTracker {
	return &RestartTracker{
		config:  config,
		servers: make(map[string]*serverRestartState),
		logger:  logger.Named("restart-tracker"),
	}
}

// SetLoopDetectedCallback sets the callback for when a restart loop is detected
func (rt *RestartTracker) SetLoopDetectedCallback(callback func(serverName string, restartCount int, window time.Duration)) {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	rt.onLoopDetected = callback
}

// RecordRestart records a restart event for a server
// Returns true if the restart should proceed, false if blocked due to loop detection
func (rt *RestartTracker) RecordRestart(serverName, reason string) bool {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	now := time.Now()

	// Get or create server state
	state, exists := rt.servers[serverName]
	if !exists {
		state = &serverRestartState{
			Restarts: make([]restartRecord, 0, rt.config.MaxRestarts+1),
		}
		rt.servers[serverName] = state
	}

	// Check if we're in cooldown period
	if state.LoopDetected {
		cooldownEnd := state.LoopDetectedAt.Add(rt.config.CooldownPeriod)
		if now.Before(cooldownEnd) {
			rt.logger.Warn("Restart blocked - server in cooldown after restart loop",
				zap.String("server", serverName),
				zap.Duration("remaining_cooldown", cooldownEnd.Sub(now)))
			return false
		}
		// Cooldown expired, reset loop detection
		state.LoopDetected = false
		state.Restarts = nil
		rt.logger.Info("Cooldown expired, restart loop detection reset",
			zap.String("server", serverName))
	}

	// Clean up old restarts outside the time window
	cutoff := now.Add(-rt.config.TimeWindow)
	cleaned := make([]restartRecord, 0, len(state.Restarts))
	for _, r := range state.Restarts {
		if r.Timestamp.After(cutoff) {
			cleaned = append(cleaned, r)
		}
	}
	state.Restarts = cleaned

	// Add new restart record
	state.Restarts = append(state.Restarts, restartRecord{
		Timestamp: now,
		Reason:    reason,
	})
	state.TotalRestarts++

	// Check if we've hit the restart limit
	if len(state.Restarts) > rt.config.MaxRestarts {
		state.LoopDetected = true
		state.LoopDetectedAt = now

		rt.logger.Error("Restart loop detected",
			zap.String("server", serverName),
			zap.Int("restart_count", len(state.Restarts)),
			zap.Duration("window", rt.config.TimeWindow),
			zap.Duration("cooldown", rt.config.CooldownPeriod))

		// Trigger callback
		if rt.onLoopDetected != nil {
			go rt.onLoopDetected(serverName, len(state.Restarts), rt.config.TimeWindow)
		}

		return false
	}

	rt.logger.Debug("Restart recorded",
		zap.String("server", serverName),
		zap.String("reason", reason),
		zap.Int("recent_restarts", len(state.Restarts)),
		zap.Int("max_restarts", rt.config.MaxRestarts))

	return true
}

// CanRestart checks if a server can be restarted without actually recording the event
func (rt *RestartTracker) CanRestart(serverName string) bool {
	rt.mu.RLock()
	defer rt.mu.RUnlock()

	state, exists := rt.servers[serverName]
	if !exists {
		return true
	}

	// Check cooldown
	if state.LoopDetected {
		cooldownEnd := state.LoopDetectedAt.Add(rt.config.CooldownPeriod)
		if time.Now().Before(cooldownEnd) {
			return false
		}
	}

	// Count recent restarts
	now := time.Now()
	cutoff := now.Add(-rt.config.TimeWindow)
	count := 0
	for _, r := range state.Restarts {
		if r.Timestamp.After(cutoff) {
			count++
		}
	}

	return count < rt.config.MaxRestarts
}

// GetServerStats returns restart statistics for a server
func (rt *RestartTracker) GetServerStats(serverName string) (recentRestarts int, totalRestarts int64, inCooldown bool, cooldownRemaining time.Duration) {
	rt.mu.RLock()
	defer rt.mu.RUnlock()

	state, exists := rt.servers[serverName]
	if !exists {
		return 0, 0, false, 0
	}

	// Count recent restarts
	now := time.Now()
	cutoff := now.Add(-rt.config.TimeWindow)
	for _, r := range state.Restarts {
		if r.Timestamp.After(cutoff) {
			recentRestarts++
		}
	}

	totalRestarts = state.TotalRestarts

	// Check cooldown
	if state.LoopDetected {
		cooldownEnd := state.LoopDetectedAt.Add(rt.config.CooldownPeriod)
		if now.Before(cooldownEnd) {
			inCooldown = true
			cooldownRemaining = cooldownEnd.Sub(now)
		}
	}

	return
}

// ResetServer resets tracking for a specific server
func (rt *RestartTracker) ResetServer(serverName string) {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	delete(rt.servers, serverName)
	rt.logger.Info("Restart tracking reset for server", zap.String("server", serverName))
}

// ResetAll resets all restart tracking
func (rt *RestartTracker) ResetAll() {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	rt.servers = make(map[string]*serverRestartState)
	rt.logger.Info("All restart tracking reset")
}

// GetAllStats returns restart statistics for all tracked servers
func (rt *RestartTracker) GetAllStats() map[string]struct {
	RecentRestarts    int
	TotalRestarts     int64
	InCooldown        bool
	CooldownRemaining time.Duration
} {
	rt.mu.RLock()
	defer rt.mu.RUnlock()

	result := make(map[string]struct {
		RecentRestarts    int
		TotalRestarts     int64
		InCooldown        bool
		CooldownRemaining time.Duration
	})

	now := time.Now()
	cutoff := now.Add(-rt.config.TimeWindow)

	for name, state := range rt.servers {
		stats := struct {
			RecentRestarts    int
			TotalRestarts     int64
			InCooldown        bool
			CooldownRemaining time.Duration
		}{
			TotalRestarts: state.TotalRestarts,
		}

		// Count recent restarts
		for _, r := range state.Restarts {
			if r.Timestamp.After(cutoff) {
				stats.RecentRestarts++
			}
		}

		// Check cooldown
		if state.LoopDetected {
			cooldownEnd := state.LoopDetectedAt.Add(rt.config.CooldownPeriod)
			if now.Before(cooldownEnd) {
				stats.InCooldown = true
				stats.CooldownRemaining = cooldownEnd.Sub(now)
			}
		}

		result[name] = stats
	}

	return result
}

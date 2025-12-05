package upstream

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestNewRestartTracker(t *testing.T) {
	logger := zap.NewNop()
	config := DefaultRestartTrackerConfig()

	tracker := NewRestartTracker(logger, config)

	assert.NotNil(t, tracker)
	assert.Equal(t, 3, tracker.config.MaxRestarts)
	assert.Equal(t, 5*time.Minute, tracker.config.TimeWindow)
	assert.Equal(t, 10*time.Minute, tracker.config.CooldownPeriod)
}

func TestRecordRestart_AllowsNormalRestarts(t *testing.T) {
	logger := zap.NewNop()
	config := RestartTrackerConfig{
		MaxRestarts:    3,
		TimeWindow:     5 * time.Minute,
		CooldownPeriod: 10 * time.Minute,
	}
	tracker := NewRestartTracker(logger, config)

	// First 3 restarts should be allowed
	assert.True(t, tracker.RecordRestart("test-server", "manual"))
	assert.True(t, tracker.RecordRestart("test-server", "manual"))
	assert.True(t, tracker.RecordRestart("test-server", "manual"))
}

func TestRecordRestart_DetectsRestartLoop(t *testing.T) {
	logger := zap.NewNop()
	config := RestartTrackerConfig{
		MaxRestarts:    3,
		TimeWindow:     5 * time.Minute,
		CooldownPeriod: 100 * time.Millisecond, // Short for testing
	}
	tracker := NewRestartTracker(logger, config)

	var loopDetected atomic.Bool
	tracker.SetLoopDetectedCallback(func(serverName string, restartCount int, window time.Duration) {
		loopDetected.Store(true)
		assert.Equal(t, "test-server", serverName)
		assert.Equal(t, 4, restartCount)
	})

	// First 3 restarts allowed
	assert.True(t, tracker.RecordRestart("test-server", "error"))
	assert.True(t, tracker.RecordRestart("test-server", "error"))
	assert.True(t, tracker.RecordRestart("test-server", "error"))

	// 4th restart should trigger loop detection
	assert.False(t, tracker.RecordRestart("test-server", "error"))

	// Wait for async callback to complete
	time.Sleep(10 * time.Millisecond)
	assert.True(t, loopDetected.Load(), "Loop should be detected")
}

func TestRecordRestart_CooldownExpires(t *testing.T) {
	logger := zap.NewNop()
	config := RestartTrackerConfig{
		MaxRestarts:    2,
		TimeWindow:     5 * time.Minute,
		CooldownPeriod: 50 * time.Millisecond, // Very short for testing
	}
	tracker := NewRestartTracker(logger, config)

	// Trigger loop detection
	assert.True(t, tracker.RecordRestart("test-server", "error"))
	assert.True(t, tracker.RecordRestart("test-server", "error"))
	assert.False(t, tracker.RecordRestart("test-server", "error")) // Blocked

	// Wait for cooldown to expire
	time.Sleep(60 * time.Millisecond)

	// Should be allowed again
	assert.True(t, tracker.RecordRestart("test-server", "after cooldown"))
}

func TestCanRestart(t *testing.T) {
	logger := zap.NewNop()
	config := RestartTrackerConfig{
		MaxRestarts:    2,
		TimeWindow:     5 * time.Minute,
		CooldownPeriod: 100 * time.Millisecond,
	}
	tracker := NewRestartTracker(logger, config)

	// Initially should allow
	assert.True(t, tracker.CanRestart("test-server"))

	// After some restarts, still can
	tracker.RecordRestart("test-server", "1")
	assert.True(t, tracker.CanRestart("test-server"))

	// After hitting limit
	tracker.RecordRestart("test-server", "2")
	tracker.RecordRestart("test-server", "3") // This triggers loop
	assert.False(t, tracker.CanRestart("test-server"))
}

func TestGetServerStats(t *testing.T) {
	logger := zap.NewNop()
	config := RestartTrackerConfig{
		MaxRestarts:    3,
		TimeWindow:     5 * time.Minute,
		CooldownPeriod: 100 * time.Millisecond,
	}
	tracker := NewRestartTracker(logger, config)

	// No stats initially
	recent, total, inCooldown, remaining := tracker.GetServerStats("test-server")
	assert.Equal(t, 0, recent)
	assert.Equal(t, int64(0), total)
	assert.False(t, inCooldown)
	assert.Equal(t, time.Duration(0), remaining)

	// After some restarts
	tracker.RecordRestart("test-server", "1")
	tracker.RecordRestart("test-server", "2")

	recent, total, inCooldown, remaining = tracker.GetServerStats("test-server")
	assert.Equal(t, 2, recent)
	assert.Equal(t, int64(2), total)
	assert.False(t, inCooldown)
}

func TestResetServer(t *testing.T) {
	logger := zap.NewNop()
	config := RestartTrackerConfig{
		MaxRestarts:    2,
		TimeWindow:     5 * time.Minute,
		CooldownPeriod: time.Hour, // Long cooldown
	}
	tracker := NewRestartTracker(logger, config)

	// Trigger loop detection
	tracker.RecordRestart("test-server", "1")
	tracker.RecordRestart("test-server", "2")
	tracker.RecordRestart("test-server", "3") // Triggers loop

	assert.False(t, tracker.CanRestart("test-server"))

	// Reset should clear everything
	tracker.ResetServer("test-server")

	assert.True(t, tracker.CanRestart("test-server"))

	recent, total, inCooldown, _ := tracker.GetServerStats("test-server")
	assert.Equal(t, 0, recent)
	assert.Equal(t, int64(0), total)
	assert.False(t, inCooldown)
}

func TestResetAll(t *testing.T) {
	logger := zap.NewNop()
	config := DefaultRestartTrackerConfig()
	tracker := NewRestartTracker(logger, config)

	// Add some data
	tracker.RecordRestart("server1", "1")
	tracker.RecordRestart("server2", "1")
	tracker.RecordRestart("server3", "1")

	// Reset all
	tracker.ResetAll()

	// All servers should be reset
	stats := tracker.GetAllStats()
	assert.Empty(t, stats)
}

func TestGetAllStats(t *testing.T) {
	logger := zap.NewNop()
	config := DefaultRestartTrackerConfig()
	tracker := NewRestartTracker(logger, config)

	tracker.RecordRestart("server1", "1")
	tracker.RecordRestart("server1", "2")
	tracker.RecordRestart("server2", "1")

	stats := tracker.GetAllStats()

	require.Len(t, stats, 2)
	assert.Equal(t, 2, stats["server1"].RecentRestarts)
	assert.Equal(t, int64(2), stats["server1"].TotalRestarts)
	assert.Equal(t, 1, stats["server2"].RecentRestarts)
	assert.Equal(t, int64(1), stats["server2"].TotalRestarts)
}

func TestMultipleServersIndependent(t *testing.T) {
	logger := zap.NewNop()
	config := RestartTrackerConfig{
		MaxRestarts:    2,
		TimeWindow:     5 * time.Minute,
		CooldownPeriod: time.Hour,
	}
	tracker := NewRestartTracker(logger, config)

	// Server1 hits limit
	tracker.RecordRestart("server1", "1")
	tracker.RecordRestart("server1", "2")
	tracker.RecordRestart("server1", "3") // Triggers loop

	// Server2 should still work
	assert.False(t, tracker.CanRestart("server1"))
	assert.True(t, tracker.CanRestart("server2"))
	assert.True(t, tracker.RecordRestart("server2", "1"))
}

func TestConcurrentAccess(t *testing.T) {
	logger := zap.NewNop()
	config := RestartTrackerConfig{
		MaxRestarts:    100,
		TimeWindow:     5 * time.Minute,
		CooldownPeriod: time.Hour,
	}
	tracker := NewRestartTracker(logger, config)

	// Concurrent restarts
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 10; j++ {
				tracker.RecordRestart("test-server", "concurrent")
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Should have recorded 100 restarts
	_, total, _, _ := tracker.GetServerStats("test-server")
	assert.Equal(t, int64(100), total)
}

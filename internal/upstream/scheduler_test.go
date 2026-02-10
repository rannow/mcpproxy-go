package upstream

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	"mcpproxy-go/internal/upstream/managed"
)

// TestNewConnectionScheduler tests scheduler creation
func TestNewConnectionScheduler(t *testing.T) {
	logger := zap.NewNop()

	tests := []struct {
		name            string
		workerCount     int
		expectedWorkers int
	}{
		{"default workers", 0, 10},
		{"negative workers", -5, 10},
		{"custom workers", 5, 5},
		{"large workers", 20, 20},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheduler := NewConnectionScheduler(nil, tt.workerCount, logger)
			assert.NotNil(t, scheduler)
			assert.Equal(t, tt.expectedWorkers, scheduler.workerCount)
			// Verify context is initialized
			assert.NotNil(t, scheduler.ctx)
			assert.NotNil(t, scheduler.cancel)
		})
	}
}

// TestSchedulerEmptyClients tests scheduler with no clients
func TestSchedulerEmptyClients(t *testing.T) {
	logger := zap.NewNop()
	scheduler := NewConnectionScheduler(nil, 10, logger)

	result := scheduler.Start(nil)

	assert.NotNil(t, result)
	assert.Equal(t, 0, result.TotalJobs)
	assert.Equal(t, 0, result.Successful)
	assert.Equal(t, 0, result.Failed)
	assert.Equal(t, 0, result.Retried)
}

// TestSchedulerResult tests the SchedulerResult struct
func TestSchedulerResult(t *testing.T) {
	result := &SchedulerResult{
		Duration:   100 * time.Millisecond,
		TotalJobs:  10,
		Successful: 8,
		Failed:     2,
		Retried:    3,
	}

	assert.Equal(t, 100*time.Millisecond, result.Duration)
	assert.Equal(t, 10, result.TotalJobs)
	assert.Equal(t, 8, result.Successful)
	assert.Equal(t, 2, result.Failed)
	assert.Equal(t, 3, result.Retried)
}

// TestSchedulerMetrics tests atomic metric counters
func TestSchedulerMetrics(t *testing.T) {
	logger := zap.NewNop()
	scheduler := NewConnectionScheduler(nil, 10, logger)

	// Initial metrics should be zero
	total, successful, failed, retrying := scheduler.GetMetrics()
	assert.Equal(t, int64(0), total)
	assert.Equal(t, int64(0), successful)
	assert.Equal(t, int64(0), failed)
	assert.Equal(t, int64(0), retrying) // Always 0 - no longer tracked

	// Manually increment metrics
	atomic.AddInt64(&scheduler.totalAttempts, 5)
	atomic.AddInt64(&scheduler.successful, 3)
	atomic.AddInt64(&scheduler.failed, 1)
	// Note: retrying is no longer tracked as separate metric

	total, successful, failed, retrying = scheduler.GetMetrics()
	assert.Equal(t, int64(5), total)
	assert.Equal(t, int64(3), successful)
	assert.Equal(t, int64(1), failed)
	assert.Equal(t, int64(0), retrying) // Always 0 - no longer tracked
}

// TestSchedulerStop tests graceful shutdown
func TestSchedulerStop(t *testing.T) {
	logger := zap.NewNop()
	scheduler := NewConnectionScheduler(nil, 10, logger)

	// Stop should cancel the context
	done := make(chan struct{})
	go func() {
		scheduler.Stop()
		close(done)
	}()

	select {
	case <-done:
		// Success - stop completed
	case <-time.After(2 * time.Second):
		t.Fatal("Scheduler stop timed out")
	}

	// Verify context was cancelled
	select {
	case <-scheduler.ctx.Done():
		// Expected - context is cancelled
	default:
		t.Fatal("Context should be cancelled after Stop()")
	}
}

// TestConnectionJob tests job struct
func TestConnectionJob(t *testing.T) {
	job := &connectionJob{
		id:     "test-server",
		client: nil,
	}

	assert.Equal(t, "test-server", job.id)
	assert.Nil(t, job.client)
}

// TestConnectionResult tests result struct
func TestConnectionResult(t *testing.T) {
	job := &connectionJob{
		id: "test-server",
	}

	result := &connectionResult{
		job:     job,
		success: true,
		err:     nil,
		elapsed: 100 * time.Millisecond,
	}

	assert.True(t, result.success)
	assert.Nil(t, result.err)
	assert.Equal(t, 100*time.Millisecond, result.elapsed)
	assert.Equal(t, "test-server", result.job.id)
}

// TestSchedulerFields tests scheduler struct fields
func TestSchedulerFields(t *testing.T) {
	logger := zap.NewNop()
	scheduler := NewConnectionScheduler(nil, 10, logger)

	// Verify scheduler is properly initialized
	assert.NotNil(t, scheduler)
	assert.Equal(t, 10, scheduler.workerCount)
	assert.NotNil(t, scheduler.logger)
	assert.NotNil(t, scheduler.ctx)
	assert.NotNil(t, scheduler.cancel)
}

// TestSchedulerContextCancellation tests context handling
func TestSchedulerContextCancellation(t *testing.T) {
	logger := zap.NewNop()
	scheduler := NewConnectionScheduler(nil, 10, logger)

	// Context should not be cancelled initially
	select {
	case <-scheduler.ctx.Done():
		t.Fatal("Context should not be cancelled initially")
	default:
		// Expected
	}

	// Cancel the context
	scheduler.cancel()

	// Context should be cancelled now
	select {
	case <-scheduler.ctx.Done():
		// Expected
	default:
		t.Fatal("Context should be cancelled after cancel()")
	}
}

// TestWorkerCountBehavior tests that worker count is properly enforced
func TestWorkerCountBehavior(t *testing.T) {
	tests := []struct {
		input    int
		expected int
	}{
		{0, 10},
		{-1, 10},
		{-100, 10},
		{1, 1},
		{5, 5},
		{10, 10},
		{50, 50},
	}

	for _, tt := range tests {
		scheduler := NewConnectionScheduler(nil, tt.input, zap.NewNop())
		assert.Equal(t, tt.expected, scheduler.workerCount,
			"input %d should result in %d workers", tt.input, tt.expected)
	}
}

// TestSchedulerWorkerCount tests that workerCount is properly set
func TestSchedulerWorkerCount(t *testing.T) {
	scheduler := NewConnectionScheduler(nil, 10, zap.NewNop())
	assert.Equal(t, 10, scheduler.workerCount)
}

// TestSchedulerWithEmptyMap tests Start with empty but non-nil map
func TestSchedulerWithEmptyMap(t *testing.T) {
	logger := zap.NewNop()
	scheduler := NewConnectionScheduler(nil, 10, logger)

	// Empty map (not nil)
	clients := make(map[string]*managed.Client)
	result := scheduler.Start(clients)

	assert.NotNil(t, result)
	assert.Equal(t, 0, result.TotalJobs)
	assert.Equal(t, 0, result.Successful)
	assert.Equal(t, 0, result.Failed)
}

// BenchmarkSchedulerCreation benchmarks scheduler creation
func BenchmarkSchedulerCreation(b *testing.B) {
	logger := zap.NewNop()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = NewConnectionScheduler(nil, 10, logger)
	}
}

// BenchmarkMetricsAccess benchmarks atomic metric access
func BenchmarkMetricsAccess(b *testing.B) {
	scheduler := NewConnectionScheduler(nil, 10, zap.NewNop())
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		scheduler.GetMetrics()
	}
}

// TestSchedulerIntegration is a simple integration test
func TestSchedulerIntegration(t *testing.T) {
	t.Skip("Skipping integration test - requires mock server setup")

	// This test would require:
	// 1. Mock MCP servers that can be connected to
	// 2. Proper Manager instance
	// 3. More complex test infrastructure
	//
	// For now, unit tests cover the core logic.
}

package upstream

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"

	"mcpproxy-go/internal/config"
	"mcpproxy-go/internal/upstream/managed"
)

// ConnectionScheduler manages startup connections using a worker pool pattern.
// It maintains a constant number of active connection workers and processes
// servers from a queue system, with failed servers retried after all others.
type ConnectionScheduler struct {
	manager     *Manager
	workerCount int
	logger      *zap.Logger

	// Channels for job distribution
	primaryQueue chan *connectionJob
	retryQueue   chan *connectionJob
	results      chan *connectionResult

	// Synchronization
	wg     sync.WaitGroup
	ctx    context.Context
	cancel context.CancelFunc

	// Metrics (atomic for thread safety)
	totalAttempts int64
	successful    int64
	failed        int64
	retrying      int64

	// Configuration
	maxRetries int
}

// connectionJob represents a server connection task
type connectionJob struct {
	id      string
	client  *managed.Client
	isRetry bool
	attempt int
}

// connectionResult represents the outcome of a connection attempt
type connectionResult struct {
	job     *connectionJob
	success bool
	err     error
	elapsed time.Duration
}

// NewConnectionScheduler creates a new scheduler with the specified worker count
func NewConnectionScheduler(manager *Manager, workerCount int, logger *zap.Logger) *ConnectionScheduler {
	if workerCount <= 0 {
		workerCount = 10 // Default to 10 concurrent workers
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &ConnectionScheduler{
		manager:      manager,
		workerCount:  workerCount,
		logger:       logger,
		primaryQueue: make(chan *connectionJob, 100), // Buffer for primary queue
		retryQueue:   make(chan *connectionJob, 100), // Buffer for retry queue
		results:      make(chan *connectionResult, 100),
		ctx:          ctx,
		cancel:       cancel,
		maxRetries:   config.MaxConnectionRetries,
	}
}

// Start begins the connection scheduling process for all eligible clients
func (s *ConnectionScheduler) Start(clients map[string]*managed.Client) *SchedulerResult {
	startTime := time.Now()
	s.logger.Info("Starting connection scheduler",
		zap.Int("worker_count", s.workerCount),
		zap.Int("total_clients", len(clients)))

	// Count eligible clients and queue them
	var eligibleCount int
	for id, client := range clients {
		if !client.Config.ShouldConnectOnStartup() {
			s.logger.Debug("Skipping server (not configured for startup connect)",
				zap.String("server", id))
			continue
		}
		if client.IsConnected() {
			s.logger.Debug("Skipping server (already connected)",
				zap.String("server", id))
			continue
		}

		eligibleCount++
		s.primaryQueue <- &connectionJob{
			id:      id,
			client:  client,
			isRetry: false,
			attempt: 1,
		}
	}

	if eligibleCount == 0 {
		s.logger.Info("No eligible clients for connection")
		return &SchedulerResult{
			Duration:    time.Since(startTime),
			TotalJobs:   0,
			Successful:  0,
			Failed:      0,
			Retried:     0,
		}
	}

	s.logger.Info("Queued clients for connection",
		zap.Int("eligible_count", eligibleCount))

	// Start workers
	for i := 0; i < s.workerCount; i++ {
		s.wg.Add(1)
		go s.worker(i)
	}

	// Start result collector
	done := make(chan struct{})
	go s.collectResults(eligibleCount, done)

	// Wait for all jobs to complete
	<-done

	// Signal workers to stop
	s.cancel()
	s.wg.Wait()

	result := &SchedulerResult{
		Duration:    time.Since(startTime),
		TotalJobs:   eligibleCount,
		Successful:  int(atomic.LoadInt64(&s.successful)),
		Failed:      int(atomic.LoadInt64(&s.failed)),
		Retried:     int(atomic.LoadInt64(&s.retrying)),
	}

	s.logger.Info("Connection scheduler completed",
		zap.Duration("duration", result.Duration),
		zap.Int("successful", result.Successful),
		zap.Int("failed", result.Failed),
		zap.Int("retried", result.Retried))

	return result
}

// worker processes jobs from the queues
func (s *ConnectionScheduler) worker(id int) {
	defer s.wg.Done()

	s.logger.Debug("Worker started", zap.Int("worker_id", id))

	for {
		select {
		case <-s.ctx.Done():
			s.logger.Debug("Worker stopping", zap.Int("worker_id", id))
			return

		case job := <-s.primaryQueue:
			s.processJob(id, job)

		default:
			// Primary queue empty, try retry queue
			select {
			case <-s.ctx.Done():
				return
			case job := <-s.primaryQueue:
				// Check primary queue again with higher priority
				s.processJob(id, job)
			case job := <-s.retryQueue:
				s.processJob(id, job)
			case <-time.After(100 * time.Millisecond):
				// Brief wait before checking again
			}
		}
	}
}

// processJob handles a single connection job
func (s *ConnectionScheduler) processJob(workerID int, job *connectionJob) {
	atomic.AddInt64(&s.totalAttempts, 1)

	// Get individual timeout for this server
	timeout := job.client.Config.GetConnectionTimeout()
	if timeout <= 0 {
		timeout = 30 * time.Second // Default timeout
	}

	ctx, cancel := context.WithTimeout(s.ctx, timeout)
	defer cancel()

	s.logger.Debug("Worker starting connection",
		zap.Int("worker_id", workerID),
		zap.String("server", job.id),
		zap.Int("attempt", job.attempt),
		zap.Bool("is_retry", job.isRetry),
		zap.Duration("timeout", timeout))

	startTime := time.Now()
	err := job.client.Connect(ctx)
	elapsed := time.Since(startTime)

	result := &connectionResult{
		job:     job,
		success: err == nil,
		err:     err,
		elapsed: elapsed,
	}

	if err != nil {
		s.logger.Warn("Connection failed",
			zap.Int("worker_id", workerID),
			zap.String("server", job.id),
			zap.Int("attempt", job.attempt),
			zap.Duration("elapsed", elapsed),
			zap.Error(err))

		// Queue for retry if under max retries
		if job.attempt < s.maxRetries {
			job.attempt++
			job.isRetry = true
			atomic.AddInt64(&s.retrying, 1)

			// Non-blocking send to retry queue
			select {
			case s.retryQueue <- job:
				s.logger.Debug("Queued for retry",
					zap.String("server", job.id),
					zap.Int("next_attempt", job.attempt))
			default:
				s.logger.Warn("Retry queue full, dropping retry",
					zap.String("server", job.id))
				atomic.AddInt64(&s.failed, 1)
			}
		} else {
			atomic.AddInt64(&s.failed, 1)
			s.logger.Error("Max retries exceeded",
				zap.String("server", job.id),
				zap.Int("attempts", job.attempt))
		}
	} else {
		atomic.AddInt64(&s.successful, 1)
		s.logger.Info("Connection successful",
			zap.Int("worker_id", workerID),
			zap.String("server", job.id),
			zap.Duration("elapsed", elapsed))
	}

	// Send result
	select {
	case s.results <- result:
	default:
		// Results channel full, log but continue
		s.logger.Debug("Results channel full, dropping result",
			zap.String("server", job.id))
	}
}

// collectResults monitors results and determines completion
func (s *ConnectionScheduler) collectResults(totalJobs int, done chan<- struct{}) {
	defer close(done)

	processed := 0
	timeout := time.NewTimer(5 * time.Minute) // Overall timeout
	defer timeout.Stop()

	for {
		select {
		case <-timeout.C:
			s.logger.Warn("Scheduler timeout reached",
				zap.Int("processed", processed),
				zap.Int("total", totalJobs))
			return

		case result := <-s.results:
			if result.success || result.job.attempt >= s.maxRetries {
				processed++
				s.logger.Debug("Job completed",
					zap.String("server", result.job.id),
					zap.Bool("success", result.success),
					zap.Int("processed", processed),
					zap.Int("total", totalJobs))
			}

			// Check if all jobs are done
			if processed >= totalJobs {
				s.logger.Debug("All jobs processed",
					zap.Int("processed", processed))
				return
			}

		case <-s.ctx.Done():
			return
		}
	}
}

// Stop cancels all pending operations
func (s *ConnectionScheduler) Stop() {
	s.cancel()
	s.wg.Wait()
}

// SchedulerResult contains the outcome of a scheduling run
type SchedulerResult struct {
	Duration   time.Duration
	TotalJobs  int
	Successful int
	Failed     int
	Retried    int
}

// GetMetrics returns current scheduler metrics
func (s *ConnectionScheduler) GetMetrics() (total, successful, failed, retrying int64) {
	return atomic.LoadInt64(&s.totalAttempts),
		atomic.LoadInt64(&s.successful),
		atomic.LoadInt64(&s.failed),
		atomic.LoadInt64(&s.retrying)
}

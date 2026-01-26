package upstream

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"

	"mcpproxy-go/internal/upstream/managed"
)

// Wave timeout configuration: exponential backoff 20s, 40s, 80s, 160s, 320s
var waveTimeouts = []time.Duration{
	20 * time.Second,
	40 * time.Second,
	80 * time.Second,
	160 * time.Second,
	320 * time.Second,
}

// ConnectionScheduler manages startup connections using a wave-based approach.
// Each wave processes ALL servers with a specific timeout before moving to the next wave.
// Wave 1: All servers with 10s timeout
// Wave 2: Failed servers with 20s timeout
// Wave 3: Failed servers with 40s timeout
// Wave 4: Failed servers with 80s timeout
// Wave 5: Failed servers with 160s timeout
type ConnectionScheduler struct {
	manager     *Manager
	workerCount int
	logger      *zap.Logger

	// Synchronization
	ctx    context.Context
	cancel context.CancelFunc

	// Metrics (atomic for thread safety)
	totalAttempts int64
	successful    int64
	failed        int64
}

// connectionJob represents a server connection task
type connectionJob struct {
	id     string
	client *managed.Client
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
		manager:     manager,
		workerCount: workerCount,
		logger:      logger,
		ctx:         ctx,
		cancel:      cancel,
	}
}

// Start begins the wave-based connection scheduling process for all eligible clients
func (s *ConnectionScheduler) Start(clients map[string]*managed.Client) *SchedulerResult {
	overallStartTime := time.Now()
	s.logger.Info("═══════════════════════════════════════════════════════════════")
	s.logger.Info("STARTUP: Connection scheduler starting",
		zap.Int("worker_count", s.workerCount),
		zap.Int("total_clients", len(clients)),
		zap.Int("max_waves", len(waveTimeouts)))

	// Collect eligible clients
	var eligibleJobs []*connectionJob
	for id, client := range clients {
		if !client.Config.ShouldConnectOnStartup() {
			s.logger.Debug("STARTUP: Skipping server (not configured for startup connect)",
				zap.String("server", id))
			continue
		}
		if client.IsConnected() {
			s.logger.Debug("STARTUP: Skipping server (already connected)",
				zap.String("server", id))
			continue
		}

		eligibleJobs = append(eligibleJobs, &connectionJob{
			id:     id,
			client: client,
		})
	}

	if len(eligibleJobs) == 0 {
		s.logger.Info("STARTUP: No eligible clients for connection")
		return &SchedulerResult{
			Duration:   time.Since(overallStartTime),
			TotalJobs:  0,
			Successful: 0,
			Failed:     0,
			Retried:    0,
		}
	}

	s.logger.Info("STARTUP: Eligible clients collected",
		zap.Int("eligible_count", len(eligibleJobs)))

	// Process waves
	pendingJobs := eligibleJobs
	var totalRetried int

	// Collect timing data across all waves
	var allConnectionTimes []time.Duration
	var successConnectionTimes []time.Duration

	for wave := 0; wave < len(waveTimeouts) && len(pendingJobs) > 0; wave++ {
		timeout := waveTimeouts[wave]
		waveStartTime := time.Now()

		s.logger.Info("───────────────────────────────────────────────────────────────")
		s.logger.Info("STARTUP: Starting wave",
			zap.Int("wave", wave+1),
			zap.Int("max_waves", len(waveTimeouts)),
			zap.Duration("timeout", timeout),
			zap.Int("servers_to_process", len(pendingJobs)))

		// Process this wave
		waveResult := s.processWave(wave+1, pendingJobs, timeout)

		// Collect timing data
		allConnectionTimes = append(allConnectionTimes, waveResult.allTimes...)
		successConnectionTimes = append(successConnectionTimes, waveResult.successTimes...)

		waveElapsed := time.Since(waveStartTime)
		successCount := len(pendingJobs) - len(waveResult.failedJobs)

		s.logger.Info("STARTUP: Wave completed",
			zap.Int("wave", wave+1),
			zap.Duration("wave_duration", waveElapsed),
			zap.Int("successful", successCount),
			zap.Int("failed", len(waveResult.failedJobs)),
			zap.Int("remaining_waves", len(waveTimeouts)-wave-1))

		// Failed jobs become pending for next wave
		if len(waveResult.failedJobs) > 0 && wave < len(waveTimeouts)-1 {
			totalRetried += len(waveResult.failedJobs)
			s.logger.Info("STARTUP: Servers queued for next wave",
				zap.Int("count", len(waveResult.failedJobs)),
				zap.Duration("next_timeout", waveTimeouts[wave+1]))
		}

		pendingJobs = waveResult.failedJobs
	}

	// Mark any remaining as finally failed
	for _, job := range pendingJobs {
		atomic.AddInt64(&s.failed, 1)
		s.logger.Error("STARTUP: Max retries exceeded",
			zap.String("server", job.id),
			zap.Int("attempts", len(waveTimeouts)))
	}

	overallDuration := time.Since(overallStartTime)

	// Calculate timing metrics
	minAll, maxAll, avgAll := calculateTimingMetrics(allConnectionTimes)
	minSuccess, maxSuccess, avgSuccess := calculateTimingMetrics(successConnectionTimes)

	result := &SchedulerResult{
		Duration:       overallDuration,
		TotalJobs:      len(eligibleJobs),
		Successful:     int(atomic.LoadInt64(&s.successful)),
		Failed:         int(atomic.LoadInt64(&s.failed)),
		Retried:        totalRetried,
		MinConnectTime: minAll,
		MaxConnectTime: maxAll,
		AvgConnectTime: avgAll,
		SuccessMinTime: minSuccess,
		SuccessMaxTime: maxSuccess,
		SuccessAvgTime: avgSuccess,
	}

	s.logger.Info("═══════════════════════════════════════════════════════════════")
	s.logger.Info("STARTUP: Connection scheduler completed",
		zap.Duration("total_duration", result.Duration),
		zap.Int("total_servers", result.TotalJobs),
		zap.Int("successful", result.Successful),
		zap.Int("failed", result.Failed),
		zap.Int("total_retried", result.Retried))
	if len(allConnectionTimes) > 0 {
		s.logger.Info("STARTUP: Connection timing metrics (all attempts)",
			zap.Duration("min", result.MinConnectTime),
			zap.Duration("max", result.MaxConnectTime),
			zap.Duration("avg", result.AvgConnectTime))
	}
	if len(successConnectionTimes) > 0 {
		s.logger.Info("STARTUP: Connection timing metrics (successful only)",
			zap.Duration("min", result.SuccessMinTime),
			zap.Duration("max", result.SuccessMaxTime),
			zap.Duration("avg", result.SuccessAvgTime))
	}
	s.logger.Info("═══════════════════════════════════════════════════════════════")

	return result
}

// waveResults contains the outcome of a single wave
type waveResults struct {
	failedJobs     []*connectionJob
	allTimes       []time.Duration
	successTimes   []time.Duration
}

// processWave processes all jobs in parallel with the specified timeout
// Returns the list of jobs that failed and timing metrics
func (s *ConnectionScheduler) processWave(waveNum int, jobs []*connectionJob, timeout time.Duration) *waveResults {
	if len(jobs) == 0 {
		return &waveResults{}
	}

	// Create channels for job distribution and results
	jobChan := make(chan *connectionJob, len(jobs))
	resultChan := make(chan *connectionResult, len(jobs))

	// Queue all jobs
	for _, job := range jobs {
		jobChan <- job
	}
	close(jobChan)

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < s.workerCount; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			s.waveWorker(workerID, waveNum, timeout, jobChan, resultChan)
		}(i)
	}

	// Wait for all workers to finish
	wg.Wait()
	close(resultChan)

	// Collect results with timing metrics
	results := &waveResults{
		allTimes:     make([]time.Duration, 0, len(jobs)),
		successTimes: make([]time.Duration, 0),
	}

	for result := range resultChan {
		results.allTimes = append(results.allTimes, result.elapsed)
		if !result.success {
			results.failedJobs = append(results.failedJobs, result.job)
		} else {
			results.successTimes = append(results.successTimes, result.elapsed)
		}
	}

	return results
}

// waveWorker processes jobs from the channel with the specified timeout
func (s *ConnectionScheduler) waveWorker(workerID, waveNum int, timeout time.Duration, jobs <-chan *connectionJob, results chan<- *connectionResult) {
	for job := range jobs {
		atomic.AddInt64(&s.totalAttempts, 1)

		ctx, cancel := context.WithTimeout(s.ctx, timeout)

		s.logger.Debug("STARTUP: Worker starting connection",
			zap.Int("worker_id", workerID),
			zap.Int("wave", waveNum),
			zap.String("server", job.id),
			zap.Duration("timeout", timeout))

		startTime := time.Now()
		err := job.client.Connect(ctx)
		elapsed := time.Since(startTime)
		cancel()

		result := &connectionResult{
			job:     job,
			success: err == nil,
			err:     err,
			elapsed: elapsed,
		}

		if err != nil {
			s.logger.Warn("STARTUP: Connection failed",
				zap.Int("worker_id", workerID),
				zap.Int("wave", waveNum),
				zap.String("server", job.id),
				zap.Duration("elapsed", elapsed),
				zap.Duration("timeout", timeout),
				zap.Error(err))
		} else {
			atomic.AddInt64(&s.successful, 1)
			s.logger.Info("STARTUP: Connection successful",
				zap.Int("worker_id", workerID),
				zap.Int("wave", waveNum),
				zap.String("server", job.id),
				zap.Duration("elapsed", elapsed))
		}

		results <- result
	}
}

// Stop cancels all pending operations
func (s *ConnectionScheduler) Stop() {
	s.cancel()
}

// SchedulerResult contains the outcome of a scheduling run
type SchedulerResult struct {
	Duration   time.Duration
	TotalJobs  int
	Successful int
	Failed     int
	Retried    int

	// Connection timing metrics
	MinConnectTime time.Duration
	MaxConnectTime time.Duration
	AvgConnectTime time.Duration
	// Separate metrics for successful vs failed connections
	SuccessMinTime time.Duration
	SuccessMaxTime time.Duration
	SuccessAvgTime time.Duration
}

// calculateTimingMetrics computes min, max, and average from a slice of durations
func calculateTimingMetrics(times []time.Duration) (min, max, avg time.Duration) {
	if len(times) == 0 {
		return 0, 0, 0
	}

	min = times[0]
	max = times[0]
	var total time.Duration

	for _, t := range times {
		if t < min {
			min = t
		}
		if t > max {
			max = t
		}
		total += t
	}

	avg = total / time.Duration(len(times))
	return min, max, avg
}

// GetMetrics returns current scheduler metrics
func (s *ConnectionScheduler) GetMetrics() (total, successful, failed, retrying int64) {
	return atomic.LoadInt64(&s.totalAttempts),
		atomic.LoadInt64(&s.successful),
		atomic.LoadInt64(&s.failed),
		0 // No longer tracking retrying as separate metric
}

// Package startup provides sequential server startup management.
// Servers are started ONE at a time with configurable delays, timeouts, and retries.
package startup

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"mcpproxy-go/internal/config"
	"mcpproxy-go/internal/upstream/managed"
)

// ServerJob represents a single server to connect.
type ServerJob struct {
	ID     string
	Client *managed.Client
}

// Result contains the outcome of the sequential startup run.
type Result struct {
	Duration   time.Duration
	TotalJobs  int
	Successful int
	Failed     int
	Retried    int
}

// Manager handles sequential ONE-at-a-time server startup.
// No parallelism, no waves, no exponential backoff.
type Manager struct {
	cfg    *config.StartupConfig
	logger *zap.Logger

	// onAutoDisable is called when a server exceeds max retries
	onAutoDisable func(serverID string, reason string)
}

// NewManager creates a startup manager with the given config.
func NewManager(cfg *config.StartupConfig, logger *zap.Logger) *Manager {
	if cfg == nil {
		def := config.DefaultStartupConfig()
		cfg = &def
	}
	cfg.Validate()
	return &Manager{
		cfg:    cfg,
		logger: logger,
	}
}

// SetAutoDisableCallback sets the callback invoked when a server is auto-disabled.
func (m *Manager) SetAutoDisableCallback(fn func(serverID string, reason string)) {
	m.onAutoDisable = fn
}

// Start processes all jobs sequentially, ONE at a time.
// Between each server it waits InterServerDelay.
// Failed servers are retried up to MaxRetries with fixed RetryDelay.
func (m *Manager) Start(ctx context.Context, jobs []ServerJob) *Result {
	start := time.Now()
	res := &Result{TotalJobs: len(jobs)}

	if len(jobs) == 0 {
		return res
	}

	m.logger.Info("STARTUP: Sequential startup beginning",
		zap.Int("servers", len(jobs)),
		zap.Duration("inter_server_delay", m.cfg.InterServerDelay()),
		zap.Int("max_retries", m.cfg.MaxRetries))

	// First pass: try each server once
	var failedJobs []ServerJob
	for i, job := range jobs {
		if ctx.Err() != nil {
			m.logger.Warn("STARTUP: Context cancelled, aborting")
			break
		}

		// Inter-server delay (skip before first server)
		if i > 0 {
			m.logger.Debug("STARTUP: Inter-server delay",
				zap.Duration("delay", m.cfg.InterServerDelay()))
			select {
			case <-time.After(m.cfg.InterServerDelay()):
			case <-ctx.Done():
				m.logger.Warn("STARTUP: Context cancelled during delay")
				break
			}
		}

		success := m.connectServer(ctx, job)
		if success {
			res.Successful++
		} else {
			failedJobs = append(failedJobs, job)
		}
	}

	// Retry loop: fixed delay, up to MaxRetries
	for retry := 1; retry <= m.cfg.MaxRetries && len(failedJobs) > 0; retry++ {
		if ctx.Err() != nil {
			break
		}

		m.logger.Info("STARTUP: Retry round",
			zap.Int("retry", retry),
			zap.Int("max_retries", m.cfg.MaxRetries),
			zap.Int("servers_to_retry", len(failedJobs)),
			zap.Duration("retry_delay", m.cfg.RetryDelay()))

		// Fixed delay before retry round
		select {
		case <-time.After(m.cfg.RetryDelay()):
		case <-ctx.Done():
			break
		}

		var stillFailed []ServerJob
		for i, job := range failedJobs {
			if ctx.Err() != nil {
				break
			}
			if i > 0 {
				select {
				case <-time.After(m.cfg.InterServerDelay()):
				case <-ctx.Done():
					break
				}
			}

			res.Retried++
			success := m.connectServer(ctx, job)
			if success {
				res.Successful++
			} else {
				stillFailed = append(stillFailed, job)
			}
		}
		failedJobs = stillFailed
	}

	// Auto-disable servers that still failed after all retries
	for _, job := range failedJobs {
		res.Failed++
		reason := fmt.Sprintf("Auto-disabled after %d startup retries", m.cfg.MaxRetries)
		m.logger.Error("STARTUP: Server exceeded max retries, auto-disabling",
			zap.String("server", job.ID),
			zap.String("name", job.Client.Config.Name),
			zap.Int("max_retries", m.cfg.MaxRetries))
		if m.onAutoDisable != nil {
			m.onAutoDisable(job.ID, reason)
		}
	}

	res.Duration = time.Since(start)

	m.logger.Info("STARTUP: Sequential startup completed",
		zap.Duration("total_duration", res.Duration),
		zap.Int("total", res.TotalJobs),
		zap.Int("successful", res.Successful),
		zap.Int("failed", res.Failed),
		zap.Int("retried", res.Retried))

	return res
}

// connectServer attempts to connect a single server with the appropriate timeout.
func (m *Manager) connectServer(ctx context.Context, job ServerJob) bool {
	timeout := m.cfg.CalculateTimeout(
		job.Client.Config.StartupTested,
		job.Client.Config.StartupTimeMs,
	)

	m.logger.Info("STARTUP: Connecting server",
		zap.String("server", job.ID),
		zap.String("name", job.Client.Config.Name),
		zap.Duration("timeout", timeout),
		zap.Bool("startup_tested", job.Client.Config.StartupTested),
		zap.Int("startup_time_ms", job.Client.Config.StartupTimeMs))

	connCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	startTime := time.Now()
	err := job.Client.Connect(connCtx)
	elapsed := time.Since(startTime)

	if err != nil {
		m.logger.Warn("STARTUP: Connection failed",
			zap.String("server", job.ID),
			zap.String("name", job.Client.Config.Name),
			zap.Duration("elapsed", elapsed),
			zap.Duration("timeout", timeout),
			zap.Error(err))
		return false
	}

	m.logger.Info("STARTUP: Connection successful",
		zap.String("server", job.ID),
		zap.String("name", job.Client.Config.Name),
		zap.Duration("elapsed", elapsed))
	return true
}

// Package shutdown provides coordinated shutdown management for mcpproxy.
// MED-003: Centralized shutdown coordination to ensure proper cleanup order and timeout handling.
package shutdown

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"mcpproxy-go/internal/config"

	"go.uber.org/zap"
)

// Phase represents a shutdown phase with ordered execution
type Phase int

const (
	// PhaseConnections - Stop accepting new connections
	PhaseConnections Phase = iota
	// PhaseWebSockets - Close WebSocket connections
	PhaseWebSockets
	// PhaseUpstreams - Disconnect upstream servers
	PhaseUpstreams
	// PhaseProcesses - Stop background processes and monitors
	PhaseProcesses
	// PhaseStorage - Close storage and caches
	PhaseStorage
	// PhaseCleanup - Final cleanup
	PhaseCleanup
)

// String returns the human-readable phase name
// MED-004: Phase uses Title Case which is suitable for both logging and UI display
func (p Phase) String() string {
	switch p {
	case PhaseConnections:
		return "Connections"
	case PhaseWebSockets:
		return "WebSockets"
	case PhaseUpstreams:
		return "Upstreams"
	case PhaseProcesses:
		return "Processes"
	case PhaseStorage:
		return "Storage"
	case PhaseCleanup:
		return "Cleanup"
	default:
		return "Unknown"
	}
}

// ShutdownFunc is a function that performs shutdown work
// It receives a context for timeout/cancellation and returns an error if shutdown failed
type ShutdownFunc func(ctx context.Context) error

// Handler represents a registered shutdown handler
type Handler struct {
	Name     string
	Phase    Phase
	Priority int // Higher priority = executed first within same phase
	Fn       ShutdownFunc
	Timeout  time.Duration // 0 = use default
}

// Coordinator manages coordinated shutdown across all components
type Coordinator struct {
	mu       sync.RWMutex
	handlers map[Phase][]*Handler
	logger   *zap.Logger

	// State
	shutdownOnce   sync.Once
	shutdownDone   chan struct{}
	shutdownErr    error
	isShuttingDown atomic.Bool

	// Configuration
	defaultTimeout time.Duration
	totalTimeout   time.Duration

	// Progress tracking
	progressCh chan Progress
}

// Progress represents shutdown progress information
type Progress struct {
	Phase     Phase
	Handler   string
	Completed bool
	Error     error
	Duration  time.Duration
}

// NewCoordinator creates a new shutdown coordinator
func NewCoordinator(logger *zap.Logger) *Coordinator {
	return &Coordinator{
		handlers:       make(map[Phase][]*Handler),
		logger:         logger.Named("shutdown"),
		shutdownDone:   make(chan struct{}),
		defaultTimeout: config.ServerDisconnectTimeout,
		totalTimeout:   config.TrayQuitTimeout,
		progressCh:     make(chan Progress, 100),
	}
}

// Register adds a shutdown handler
func (c *Coordinator) Register(h *Handler) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if h.Timeout == 0 {
		h.Timeout = c.defaultTimeout
	}

	c.handlers[h.Phase] = append(c.handlers[h.Phase], h)

	// Sort by priority (higher first)
	handlers := c.handlers[h.Phase]
	for i := len(handlers) - 1; i > 0; i-- {
		if handlers[i].Priority > handlers[i-1].Priority {
			handlers[i], handlers[i-1] = handlers[i-1], handlers[i]
		}
	}

	c.logger.Debug("Registered shutdown handler",
		zap.String("name", h.Name),
		zap.String("phase", h.Phase.String()),
		zap.Int("priority", h.Priority))
}

// RegisterFunc is a convenience method to register a simple shutdown function
func (c *Coordinator) RegisterFunc(name string, phase Phase, fn ShutdownFunc) {
	c.Register(&Handler{
		Name:  name,
		Phase: phase,
		Fn:    fn,
	})
}

// Unregister removes a shutdown handler by name
func (c *Coordinator) Unregister(name string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for phase, handlers := range c.handlers {
		for i, h := range handlers {
			if h.Name == name {
				c.handlers[phase] = append(handlers[:i], handlers[i+1:]...)
				c.logger.Debug("Unregistered shutdown handler", zap.String("name", name))
				return
			}
		}
	}
}

// IsShuttingDown returns true if shutdown is in progress
func (c *Coordinator) IsShuttingDown() bool {
	return c.isShuttingDown.Load()
}

// Done returns a channel that is closed when shutdown is complete
func (c *Coordinator) Done() <-chan struct{} {
	return c.shutdownDone
}

// Progress returns a channel for receiving shutdown progress updates
func (c *Coordinator) Progress() <-chan Progress {
	return c.progressCh
}

// Shutdown initiates coordinated shutdown
// It is safe to call multiple times - only the first call will execute
func (c *Coordinator) Shutdown(ctx context.Context) error {
	c.shutdownOnce.Do(func() {
		c.isShuttingDown.Store(true)
		c.shutdownErr = c.executeShutdown(ctx)
		close(c.shutdownDone)
		close(c.progressCh)
	})

	return c.shutdownErr
}

// executeShutdown performs the actual shutdown sequence
func (c *Coordinator) executeShutdown(ctx context.Context) error {
	c.logger.Info("Starting coordinated shutdown")
	startTime := time.Now()

	// Create timeout context for entire shutdown
	shutdownCtx, cancel := context.WithTimeout(ctx, c.totalTimeout)
	defer cancel()

	var allErrors []error

	// Execute phases in order
	phases := []Phase{
		PhaseConnections,
		PhaseWebSockets,
		PhaseUpstreams,
		PhaseProcesses,
		PhaseStorage,
		PhaseCleanup,
	}

	for _, phase := range phases {
		if err := c.executePhase(shutdownCtx, phase); err != nil {
			allErrors = append(allErrors, fmt.Errorf("phase %s: %w", phase.String(), err))
			// Continue with other phases even if one fails
		}

		// Check if context was cancelled
		if shutdownCtx.Err() != nil {
			c.logger.Warn("Shutdown timeout reached, aborting remaining phases",
				zap.Duration("elapsed", time.Since(startTime)))
			allErrors = append(allErrors, fmt.Errorf("shutdown timeout: %w", shutdownCtx.Err()))
			break
		}
	}

	duration := time.Since(startTime)
	if len(allErrors) > 0 {
		c.logger.Warn("Shutdown completed with errors",
			zap.Duration("duration", duration),
			zap.Int("error_count", len(allErrors)))
		return errors.Join(allErrors...)
	}

	c.logger.Info("Shutdown completed successfully",
		zap.Duration("duration", duration))
	return nil
}

// executePhase runs all handlers for a specific phase
func (c *Coordinator) executePhase(ctx context.Context, phase Phase) error {
	c.mu.RLock()
	handlers := make([]*Handler, len(c.handlers[phase]))
	copy(handlers, c.handlers[phase])
	c.mu.RUnlock()

	if len(handlers) == 0 {
		return nil
	}

	c.logger.Info("Executing shutdown phase",
		zap.String("phase", phase.String()),
		zap.Int("handler_count", len(handlers)))

	var phaseErrors []error

	for _, h := range handlers {
		if err := c.executeHandler(ctx, h); err != nil {
			phaseErrors = append(phaseErrors, fmt.Errorf("%s: %w", h.Name, err))
		}
	}

	if len(phaseErrors) > 0 {
		return errors.Join(phaseErrors...)
	}
	return nil
}

// executeHandler runs a single handler with timeout
func (c *Coordinator) executeHandler(ctx context.Context, h *Handler) error {
	startTime := time.Now()

	// Create handler-specific timeout
	handlerCtx, cancel := context.WithTimeout(ctx, h.Timeout)
	defer cancel()

	c.logger.Debug("Executing shutdown handler",
		zap.String("name", h.Name),
		zap.String("phase", h.Phase.String()))

	// Execute handler
	errCh := make(chan error, 1)
	go func() {
		errCh <- h.Fn(handlerCtx)
	}()

	var err error
	select {
	case err = <-errCh:
		// Handler completed
	case <-handlerCtx.Done():
		err = fmt.Errorf("handler timeout after %v", h.Timeout)
	}

	duration := time.Since(startTime)

	// Send progress update (non-blocking)
	select {
	case c.progressCh <- Progress{
		Phase:     h.Phase,
		Handler:   h.Name,
		Completed: err == nil,
		Error:     err,
		Duration:  duration,
	}:
	default:
		// Progress channel full, skip
	}

	if err != nil {
		c.logger.Warn("Shutdown handler failed",
			zap.String("name", h.Name),
			zap.Duration("duration", duration),
			zap.Error(err))
		return err
	}

	c.logger.Debug("Shutdown handler completed",
		zap.String("name", h.Name),
		zap.Duration("duration", duration))
	return nil
}

// SetTotalTimeout sets the total timeout for the entire shutdown sequence
func (c *Coordinator) SetTotalTimeout(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.totalTimeout = d
}

// SetDefaultTimeout sets the default timeout for individual handlers
func (c *Coordinator) SetDefaultTimeout(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.defaultTimeout = d
}

// GetHandlerCount returns the number of registered handlers
func (c *Coordinator) GetHandlerCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	count := 0
	for _, handlers := range c.handlers {
		count += len(handlers)
	}
	return count
}

// GetPhaseHandlers returns the handlers registered for a specific phase
func (c *Coordinator) GetPhaseHandlers(phase Phase) []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var names []string
	for _, h := range c.handlers[phase] {
		names = append(names, h.Name)
	}
	return names
}

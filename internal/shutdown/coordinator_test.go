package shutdown

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestNewCoordinator(t *testing.T) {
	logger := zap.NewNop()
	c := NewCoordinator(logger)

	if c == nil {
		t.Fatal("NewCoordinator returned nil")
	}

	if c.GetHandlerCount() != 0 {
		t.Errorf("Expected 0 handlers, got %d", c.GetHandlerCount())
	}

	if c.IsShuttingDown() {
		t.Error("Expected IsShuttingDown to be false initially")
	}
}

func TestRegisterHandler(t *testing.T) {
	logger := zap.NewNop()
	c := NewCoordinator(logger)

	c.RegisterFunc("test-handler", PhaseConnections, func(ctx context.Context) error {
		return nil
	})

	if c.GetHandlerCount() != 1 {
		t.Errorf("Expected 1 handler, got %d", c.GetHandlerCount())
	}

	handlers := c.GetPhaseHandlers(PhaseConnections)
	if len(handlers) != 1 || handlers[0] != "test-handler" {
		t.Errorf("Expected test-handler, got %v", handlers)
	}
}

func TestRegisterMultipleHandlers(t *testing.T) {
	logger := zap.NewNop()
	c := NewCoordinator(logger)

	c.Register(&Handler{
		Name:     "low-priority",
		Phase:    PhaseUpstreams,
		Priority: 1,
		Fn:       func(ctx context.Context) error { return nil },
	})

	c.Register(&Handler{
		Name:     "high-priority",
		Phase:    PhaseUpstreams,
		Priority: 10,
		Fn:       func(ctx context.Context) error { return nil },
	})

	handlers := c.GetPhaseHandlers(PhaseUpstreams)
	if len(handlers) != 2 {
		t.Fatalf("Expected 2 handlers, got %d", len(handlers))
	}

	// High priority should be first
	if handlers[0] != "high-priority" {
		t.Errorf("Expected high-priority first, got %s", handlers[0])
	}
}

func TestUnregisterHandler(t *testing.T) {
	logger := zap.NewNop()
	c := NewCoordinator(logger)

	c.RegisterFunc("test-handler", PhaseConnections, func(ctx context.Context) error {
		return nil
	})

	c.Unregister("test-handler")

	if c.GetHandlerCount() != 0 {
		t.Errorf("Expected 0 handlers after unregister, got %d", c.GetHandlerCount())
	}
}

func TestShutdownExecutesHandlers(t *testing.T) {
	logger := zap.NewNop()
	c := NewCoordinator(logger)

	var executed atomic.Int32

	c.RegisterFunc("handler1", PhaseConnections, func(ctx context.Context) error {
		executed.Add(1)
		return nil
	})

	c.RegisterFunc("handler2", PhaseUpstreams, func(ctx context.Context) error {
		executed.Add(1)
		return nil
	})

	c.RegisterFunc("handler3", PhaseStorage, func(ctx context.Context) error {
		executed.Add(1)
		return nil
	})

	err := c.Shutdown(context.Background())
	if err != nil {
		t.Errorf("Shutdown returned error: %v", err)
	}

	if executed.Load() != 3 {
		t.Errorf("Expected 3 handlers executed, got %d", executed.Load())
	}

	if !c.IsShuttingDown() {
		t.Error("Expected IsShuttingDown to be true after shutdown")
	}
}

func TestShutdownPhasesInOrder(t *testing.T) {
	logger := zap.NewNop()
	c := NewCoordinator(logger)

	var order []Phase
	var mu atomic.Value
	mu.Store(order)

	addPhase := func(p Phase) {
		current := mu.Load().([]Phase)
		mu.Store(append(current, p))
	}

	c.RegisterFunc("connections", PhaseConnections, func(ctx context.Context) error {
		addPhase(PhaseConnections)
		return nil
	})

	c.RegisterFunc("upstreams", PhaseUpstreams, func(ctx context.Context) error {
		addPhase(PhaseUpstreams)
		return nil
	})

	c.RegisterFunc("storage", PhaseStorage, func(ctx context.Context) error {
		addPhase(PhaseStorage)
		return nil
	})

	c.RegisterFunc("cleanup", PhaseCleanup, func(ctx context.Context) error {
		addPhase(PhaseCleanup)
		return nil
	})

	_ = c.Shutdown(context.Background())

	result := mu.Load().([]Phase)
	expected := []Phase{PhaseConnections, PhaseUpstreams, PhaseStorage, PhaseCleanup}

	if len(result) != len(expected) {
		t.Fatalf("Expected %d phases, got %d", len(expected), len(result))
	}

	for i, p := range expected {
		if result[i] != p {
			t.Errorf("Phase %d: expected %s, got %s", i, p.String(), result[i].String())
		}
	}
}

func TestShutdownHandlerError(t *testing.T) {
	logger := zap.NewNop()
	c := NewCoordinator(logger)

	expectedErr := errors.New("handler error")

	c.RegisterFunc("failing-handler", PhaseConnections, func(ctx context.Context) error {
		return expectedErr
	})

	c.RegisterFunc("success-handler", PhaseUpstreams, func(ctx context.Context) error {
		return nil
	})

	err := c.Shutdown(context.Background())
	if err == nil {
		t.Error("Expected error from shutdown")
	}

	// Should contain the handler error
	if !errors.Is(err, expectedErr) {
		t.Errorf("Expected error to contain %v, got %v", expectedErr, err)
	}
}

func TestShutdownTimeout(t *testing.T) {
	logger := zap.NewNop()
	c := NewCoordinator(logger)
	c.SetTotalTimeout(100 * time.Millisecond)

	c.RegisterFunc("slow-handler", PhaseConnections, func(ctx context.Context) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(5 * time.Second):
			return nil
		}
	})

	start := time.Now()
	err := c.Shutdown(context.Background())
	duration := time.Since(start)

	if err == nil {
		t.Error("Expected timeout error")
	}

	if duration > 500*time.Millisecond {
		t.Errorf("Shutdown took too long: %v", duration)
	}
}

func TestShutdownOnlyOnce(t *testing.T) {
	logger := zap.NewNop()
	c := NewCoordinator(logger)

	var count atomic.Int32

	c.RegisterFunc("counter", PhaseConnections, func(ctx context.Context) error {
		count.Add(1)
		return nil
	})

	// Call shutdown multiple times
	_ = c.Shutdown(context.Background())
	_ = c.Shutdown(context.Background())
	_ = c.Shutdown(context.Background())

	if count.Load() != 1 {
		t.Errorf("Expected handler to run once, ran %d times", count.Load())
	}
}

func TestProgressChannel(t *testing.T) {
	logger := zap.NewNop()
	c := NewCoordinator(logger)

	c.RegisterFunc("progress-handler", PhaseConnections, func(ctx context.Context) error {
		return nil
	})

	progressCh := c.Progress()

	go func() {
		_ = c.Shutdown(context.Background())
	}()

	// Should receive progress update
	select {
	case progress := <-progressCh:
		if progress.Handler != "progress-handler" {
			t.Errorf("Expected progress-handler, got %s", progress.Handler)
		}
		if !progress.Completed {
			t.Error("Expected progress.Completed to be true")
		}
	case <-time.After(time.Second):
		t.Error("Timeout waiting for progress update")
	}
}

func TestDoneChannel(t *testing.T) {
	logger := zap.NewNop()
	c := NewCoordinator(logger)

	c.RegisterFunc("handler", PhaseConnections, func(ctx context.Context) error {
		return nil
	})

	go func() {
		_ = c.Shutdown(context.Background())
	}()

	select {
	case <-c.Done():
		// Success
	case <-time.After(time.Second):
		t.Error("Timeout waiting for done channel")
	}
}

func TestPhaseString(t *testing.T) {
	tests := []struct {
		phase    Phase
		expected string
	}{
		{PhaseConnections, "Connections"},
		{PhaseWebSockets, "WebSockets"},
		{PhaseUpstreams, "Upstreams"},
		{PhaseProcesses, "Processes"},
		{PhaseStorage, "Storage"},
		{PhaseCleanup, "Cleanup"},
		{Phase(99), "Unknown"},
	}

	for _, tt := range tests {
		if got := tt.phase.String(); got != tt.expected {
			t.Errorf("Phase(%d).String() = %s, want %s", tt.phase, got, tt.expected)
		}
	}
}

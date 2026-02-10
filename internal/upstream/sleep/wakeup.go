package sleep

import (
	"context"
	"fmt"
	"sync"
	"time"

	"mcpproxy-go/internal/upstream/types"

	"go.uber.org/zap"
)

// WakeupManager handles wake-on-call functionality for sleeping servers.
// It intercepts tool calls to sleeping servers, wakes them up transparently,
// and waits for connection readiness before forwarding the call.
type WakeupManager struct {
	mu            sync.RWMutex
	wakeTimeoutMs int
	getClientFunc func(serverName string) ClientInterface
	logger        *zap.Logger
}

// NewWakeupManager creates a new wakeup manager with the specified configuration.
// wakeTimeoutMs is the maximum time to wait for a server to wake up and be ready.
// getClientFunc provides access to clients via the Manager's callback.
func NewWakeupManager(
	wakeTimeoutMs int,
	getClientFunc func(serverName string) ClientInterface,
	logger *zap.Logger,
) *WakeupManager {
	return &WakeupManager{
		wakeTimeoutMs: wakeTimeoutMs,
		getClientFunc: getClientFunc,
		logger:        logger,
	}
}

// InterceptToolCall checks if a server is sleeping and wakes it up if needed.
// This should be called before forwarding any tool call to a server.
// Returns nil if server is awake or successfully woken, error otherwise.
func (wm *WakeupManager) InterceptToolCall(serverName string) error {
	client := wm.getClientFunc(serverName)
	if client == nil {
		return fmt.Errorf("server not found: %s", serverName)
	}

	stateManager := client.GetStateManager()
	if stateManager == nil {
		return fmt.Errorf("state manager not available for server: %s", serverName)
	}

	// Check if server is sleeping
	currentState := stateManager.GetServerState()
	if currentState != types.StateSleep {
		// Server not sleeping, no wake needed
		return nil
	}

	// Wake server: Sleep → Active transition
	wm.logger.Info("Waking sleeping server for tool call",
		zap.String("server", serverName),
		zap.Time("wake_time", time.Now()))

	err := stateManager.TransitionServerState(types.StateActive)
	if err != nil {
		return fmt.Errorf("failed to transition server to active: %w", err)
	}

	// Wait for server to be ready with timeout
	ctx, cancel := context.WithTimeout(context.Background(),
		time.Duration(wm.wakeTimeoutMs)*time.Millisecond)
	defer cancel()

	return wm.waitForReady(ctx, client)
}

// waitForReady waits for a server to reach Connected state after waking up.
// It polls the connection state at 100ms intervals until ready or timeout.
func (wm *WakeupManager) waitForReady(ctx context.Context, client ClientInterface) error {
	serverName := client.GetName()
	stateManager := client.GetStateManager()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	startTime := time.Now()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for server %s to wake (waited %v): %w",
				serverName, time.Since(startTime), ctx.Err())

		case <-ticker.C:
			connState := stateManager.GetState()
			if connState == types.StateReady {
				duration := time.Since(startTime)
				wm.logger.Info("Server ready after wake-up",
					zap.String("server", serverName),
					zap.Duration("wake_duration", duration))
				return nil
			}

			// Log intermediate states for debugging
			wm.logger.Debug("Waiting for server to wake",
				zap.String("server", serverName),
				zap.String("current_state", string(connState)),
				zap.Duration("elapsed", time.Since(startTime)))
		}
	}
}

// IsServerSleeping checks if a server is currently in sleep mode.
// This can be used to determine if a tool call will trigger a wake-up.
func (wm *WakeupManager) IsServerSleeping(serverName string) bool {
	client := wm.getClientFunc(serverName)
	if client == nil {
		return false
	}

	stateManager := client.GetStateManager()
	if stateManager == nil {
		return false
	}

	return stateManager.GetServerState() == types.StateSleep
}

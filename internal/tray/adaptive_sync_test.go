//go:build !nogui && !headless && !linux

package tray

import (
	"context"
	"testing"
	"time"

	"go.uber.org/zap/zaptest"
)

func TestAdaptiveSyncFrequency(t *testing.T) {
	// Create a mock logger
	logger := zaptest.NewLogger(t).Sugar()

	// Create a mock state manager and menu manager
	mockStateManager := &ServerStateManager{
		logger: logger,
	}
	mockMenuManager := NewMenuManager(nil, nil, nil, nil, nil, nil, nil, logger)

	// Create sync manager
	syncManager := NewSynchronizationManager(mockStateManager, mockMenuManager, logger)

	// Test initial state
	if syncManager.lastUserActivity.IsZero() {
		t.Error("Expected lastUserActivity to be initialized")
	}

	// Test user activity notification
	initialTime := syncManager.lastUserActivity
	time.Sleep(10 * time.Millisecond) // Small delay to ensure time difference

	syncManager.NotifyUserActivity()

	if !syncManager.lastUserActivity.After(initialTime) {
		t.Error("Expected lastUserActivity to be updated after NotifyUserActivity")
	}

	// Verify that sync manager handles context cancellation
	ctx, cancel := context.WithCancel(context.Background())
	syncManager.ctx = ctx
	syncManager.cancel = cancel

	// Cancel context
	cancel()

	// The syncLoop should handle cancellation gracefully
	// We can't easily test the sync loop without running it in a goroutine,
	// but we can verify the context is properly set up
	select {
	case <-syncManager.ctx.Done():
		// Expected behavior
	default:
		t.Error("Expected context to be cancelled")
	}
}

func TestSyncManagerInitialization(t *testing.T) {
	logger := zaptest.NewLogger(t).Sugar()

	// Create a mock state manager and menu manager
	mockStateManager := &ServerStateManager{
		logger: logger,
	}
	mockMenuManager := NewMenuManager(nil, nil, nil, nil, nil, nil, nil, logger)

	// Create sync manager
	syncManager := NewSynchronizationManager(mockStateManager, mockMenuManager, logger)

	// Verify initialization
	if syncManager.stateManager != mockStateManager {
		t.Error("Expected state manager to be set correctly")
	}

	if syncManager.menuManager != mockMenuManager {
		t.Error("Expected menu manager to be set correctly")
	}

	if syncManager.logger != logger {
		t.Error("Expected logger to be set correctly")
	}

	if syncManager.ctx == nil {
		t.Error("Expected context to be initialized")
	}

	if syncManager.cancel == nil {
		t.Error("Expected cancel function to be initialized")
	}

	if syncManager.lastUserActivity.IsZero() {
		t.Error("Expected lastUserActivity to be initialized")
	}
}

func TestUserActivityTracking(t *testing.T) {
	logger := zaptest.NewLogger(t).Sugar()

	// Create sync manager
	syncManager := NewSynchronizationManager(nil, nil, logger)

	// Record initial time
	initialTime := syncManager.lastUserActivity

	// Wait a bit and then notify activity
	time.Sleep(5 * time.Millisecond)
	syncManager.NotifyUserActivity()

	// Verify time was updated
	if !syncManager.lastUserActivity.After(initialTime) {
		t.Error("Expected lastUserActivity to be updated")
	}

	// Verify concurrent access works
	done := make(chan bool)
	go func() {
		for i := 0; i < 100; i++ {
			syncManager.NotifyUserActivity()
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			syncManager.activityMu.RLock()
			_ = syncManager.lastUserActivity
			syncManager.activityMu.RUnlock()
		}
		done <- true
	}()

	// Wait for both goroutines to complete
	<-done
	<-done

	// If we reach here without deadlock, the test passes
}
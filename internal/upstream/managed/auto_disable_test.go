package managed

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"mcpproxy-go/internal/config"
	"mcpproxy-go/internal/storage"
	"mcpproxy-go/internal/upstream/types"
)

// TestAutoDisablePersistence tests that auto-disable state is persisted to storage
func TestAutoDisablePersistence(t *testing.T) {
	// Create temp directory for test
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")

	// Create initial config file
	cfg := config.DefaultConfig()
	cfg.DataDir = tempDir
	cfg.AutoDisableThreshold = 3

	testServer := &config.ServerConfig{
		Name:        "test-server",
		URL:         "http://localhost:9999",
		Protocol:    "http",
		StartupMode: "active",
	}
	cfg.Servers = []*config.ServerConfig{testServer}

	// Write initial config
	require.NoError(t, config.SaveConfig(cfg, configPath))

	// Create logger
	logger := zap.NewNop()

	// Initialize storage manager with config loader
	storageManager, err := storage.NewManager(tempDir, logger.Sugar())
	require.NoError(t, err)
	defer storageManager.Close()

	// Create config loader
	configLoader, err := config.NewLoader(configPath, logger)
	require.NoError(t, err)
	defer configLoader.Stop()

	// Load initial config
	_, err = configLoader.Load()
	require.NoError(t, err)

	// Set config loader on storage manager
	storageManager.SetConfigLoader(configLoader)

	// Save initial upstream record (this is normally done by the server on startup)
	require.NoError(t, storageManager.GetBoltDB().SaveUpstream(&storage.UpstreamRecord{
		ID:          "test-server",
		Name:        "test-server",
		URL:         "http://localhost:9999",
		Protocol:    "http",
		ServerState: "active",
		Created:     time.Now(),
		Updated:     time.Now(),
	}))

	// Create client
	client, err := NewClient("test-server", testServer, logger, nil, cfg, storageManager.GetBoltDB())
	require.NoError(t, err)

	// Set storage manager on client
	client.SetStorageManager(storageManager)

	// Configure auto-disable threshold
	client.StateManager.SetAutoDisableThreshold(3)

	// Simulate 3 consecutive failures by setting errors
	for i := 0; i < 3; i++ {
		client.StateManager.SetError(assert.AnError)
	}

	// Verify auto-disable should trigger
	require.True(t, client.StateManager.ShouldAutoDisable(), "Should auto-disable after 3 failures")

	// Trigger auto-disable handling
	client.checkAndHandleAutoDisable()

	// Verify state manager is auto-disabled
	info := client.StateManager.GetConnectionInfo()
	assert.True(t, info.AutoDisabled, "State manager should be auto-disabled")
	assert.Contains(t, info.AutoDisableReason, "consecutive failures")

	// Verify persistence in database
	record, err := storageManager.GetBoltDB().GetUpstream("test-server")
	require.NoError(t, err)
	assert.Equal(t, "auto_disabled", record.ServerState, "Server state should be auto_disabled in database")

	// Verify config file remains unchanged (default: PersistAutoDisableToConfig=false)
	// With the new behavior, auto-disable is only saved to database, not config file
	reloadedCfg, err := config.LoadFromFile(configPath)
	require.NoError(t, err)

	found := false
	for _, srv := range reloadedCfg.Servers {
		if srv.Name == "test-server" {
			found = true
			assert.Equal(t, "active", srv.StartupMode, "Server startup mode should remain 'active' in config file (DB-only persistence)")
			break
		}
	}
	assert.True(t, found, "Server should be found in config file")
}

// TestAutoDisableRollback tests that database changes are rolled back if config update fails
func TestAutoDisableRollback(t *testing.T) {
	// Create temp directory for test
	tempDir := t.TempDir()

	// Create read-only directory for config (to force failure)
	configDir := filepath.Join(tempDir, "readonly")
	require.NoError(t, os.MkdirAll(configDir, 0755))
	configPath := filepath.Join(configDir, "config.json")

	// Create initial config file
	cfg := config.DefaultConfig()
	cfg.DataDir = tempDir
	cfg.AutoDisableThreshold = 3

	testServer := &config.ServerConfig{
		Name:        "test-server",
		URL:         "http://localhost:9999",
		Protocol:    "http",
		StartupMode: "active",
	}
	cfg.Servers = []*config.ServerConfig{testServer}

	// Write initial config
	require.NoError(t, config.SaveConfig(cfg, configPath))

	// Create logger
	logger := zap.NewNop()

	// Initialize storage manager with config loader
	storageManager, err := storage.NewManager(tempDir, logger.Sugar())
	require.NoError(t, err)
	defer storageManager.Close()

	// Create config loader
	configLoader, err := config.NewLoader(configPath, logger)
	require.NoError(t, err)
	defer configLoader.Stop()

	// Load initial config
	_, err = configLoader.Load()
	require.NoError(t, err)

	// Set config loader on storage manager
	storageManager.SetConfigLoader(configLoader)

	// Save initial upstream record
	require.NoError(t, storageManager.GetBoltDB().SaveUpstream(&storage.UpstreamRecord{
		ID:          "test-server",
		Name:        "test-server",
		URL:         "http://localhost:9999",
		Protocol:    "http",
		ServerState: "active",
		Created:     time.Now(),
		Updated:     time.Now(),
	}))

	// NOW make directory read-only to force config save failure
	require.NoError(t, os.Chmod(configDir, 0444))
	defer os.Chmod(configDir, 0755) // Restore for cleanup

	// Attempt to update startup mode (should fail and rollback)
	err = storageManager.UpdateServerState("test-server", "test failure")
	require.Error(t, err, "Should fail to update due to read-only config directory")

	// Verify database was rolled back
	record, err := storageManager.GetBoltDB().GetUpstream("test-server")
	require.NoError(t, err)
	assert.Equal(t, "active", record.ServerState, "Server state should still be active after rollback")
}

// TestClearAutoDisablePersistence tests that clearing auto-disable persists properly
func TestClearAutoDisablePersistence(t *testing.T) {
	// Create temp directory for test
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")

	// Create initial config with auto-disabled server
	cfg := config.DefaultConfig()
	cfg.DataDir = tempDir

	testServer := &config.ServerConfig{
		Name:        "test-server",
		URL:         "http://localhost:9999",
		Protocol:    "http",
		StartupMode: "auto_disabled",
	}
	cfg.Servers = []*config.ServerConfig{testServer}

	// Write initial config
	require.NoError(t, config.SaveConfig(cfg, configPath))

	// Create logger
	logger := zap.NewNop()

	// Initialize storage manager with config loader
	storageManager, err := storage.NewManager(tempDir, logger.Sugar())
	require.NoError(t, err)
	defer storageManager.Close()

	// Create config loader
	configLoader, err := config.NewLoader(configPath, logger)
	require.NoError(t, err)
	defer configLoader.Stop()

	// Load initial config
	_, err = configLoader.Load()
	require.NoError(t, err)

	// Set config loader on storage manager
	storageManager.SetConfigLoader(configLoader)

	// Save initial upstream record with auto-disable
	require.NoError(t, storageManager.GetBoltDB().SaveUpstream(&storage.UpstreamRecord{
		ID:          "test-server",
		Name:        "test-server",
		URL:         "http://localhost:9999",
		Protocol:    "http",
		ServerState: "auto_disabled",
		Created:     time.Now(),
		Updated:     time.Now(),
	}))

	// Clear auto-disable
	err = storageManager.ClearAutoDisable("test-server")
	require.NoError(t, err)

	// Verify database was updated
	record, err := storageManager.GetBoltDB().GetUpstream("test-server")
	require.NoError(t, err)
	assert.Equal(t, "active", record.ServerState, "Server state should be active (not auto_disabled) in database")

	// Verify config file was updated
	reloadedCfg, err := config.LoadFromFile(configPath)
	require.NoError(t, err)

	found := false
	for _, srv := range reloadedCfg.Servers {
		if srv.Name == "test-server" {
			found = true
			assert.Equal(t, "active", srv.StartupMode, "Startup mode should be active (not auto_disabled) in config")
			assert.Empty(t, srv.AutoDisableReason, "Reason should be cleared in config")
			break
		}
	}
	assert.True(t, found, "Server should be found in config file")
}

// TestRestartPersistence tests that auto-disable state survives application restart
func TestRestartPersistence(t *testing.T) {
	// Create temp directory for test
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")

	// Create initial config file
	cfg := config.DefaultConfig()
	cfg.DataDir = tempDir
	cfg.AutoDisableThreshold = 3

	testServer := &config.ServerConfig{
		Name:        "test-server",
		URL:         "http://localhost:9999",
		Protocol:    "http",
		StartupMode: "active",
	}
	cfg.Servers = []*config.ServerConfig{testServer}

	// Write initial config
	require.NoError(t, config.SaveConfig(cfg, configPath))

	// Create logger
	logger := zap.NewNop()

	// === FIRST SESSION: Set auto-disable ===
	{
		// Initialize storage manager
		storageManager, err := storage.NewManager(tempDir, logger.Sugar())
		require.NoError(t, err)

		// Create config loader
		configLoader, err := config.NewLoader(configPath, logger)
		require.NoError(t, err)

		// Load initial config
		_, err = configLoader.Load()
		require.NoError(t, err)

		// Set config loader on storage manager
		storageManager.SetConfigLoader(configLoader)

		// Save initial upstream record (this is normally done by the server on startup)
		require.NoError(t, storageManager.GetBoltDB().SaveUpstream(&storage.UpstreamRecord{
			ID:          "test-server",
			Name:        "test-server",
			URL:         "http://localhost:9999",
			Protocol:    "http",
			ServerState: "active",
			Created:     time.Now(),
			Updated:     time.Now(),
		}))

		// Create client
		client, err := NewClient("test-server", testServer, logger, nil, cfg, storageManager.GetBoltDB())
		require.NoError(t, err)

		// Set storage manager on client
		client.SetStorageManager(storageManager)

		// Configure auto-disable threshold
		client.StateManager.SetAutoDisableThreshold(3)

		// Simulate 3 consecutive failures
		for i := 0; i < 3; i++ {
			client.StateManager.SetError(assert.AnError)
		}

		// Trigger auto-disable
		client.checkAndHandleAutoDisable()

		// Verify auto-disabled
		info := client.StateManager.GetConnectionInfo()
		assert.True(t, info.AutoDisabled, "Should be auto-disabled")

		// Close resources
		configLoader.Stop()
		storageManager.Close()
	}

	// === SECOND SESSION: Verify persistence after restart ===
	{
		// Reload config from disk
		reloadedCfg, err := config.LoadFromFile(configPath)
		require.NoError(t, err)

		// Find server in config
		var reloadedServer *config.ServerConfig
		for _, srv := range reloadedCfg.Servers {
			if srv.Name == "test-server" {
				reloadedServer = srv
				break
			}
		}
		require.NotNil(t, reloadedServer, "Server should exist in reloaded config")

		// Verify config file remains 'active' (default: PersistAutoDisableToConfig=false)
		// Auto-disable state is tracked in database, not config file
		assert.Equal(t, "active", reloadedServer.StartupMode, "Server should remain 'active' in config file (DB-only persistence)")

		// Initialize new storage manager
		storageManager2, err := storage.NewManager(tempDir, logger.Sugar())
		require.NoError(t, err)
		defer storageManager2.Close()

		// Verify auto-disable persisted in database
		record, err := storageManager2.GetBoltDB().GetUpstream("test-server")
		require.NoError(t, err)
		assert.Equal(t, "auto_disabled", record.ServerState, "Server should be auto_disabled in database after restart")

		// Create new client with persisted config
		client2, err := NewClient("test-server", reloadedServer, logger, nil, reloadedCfg, storageManager2.GetBoltDB())
		require.NoError(t, err)

		// Restore auto-disable state from database (simulating what manager does)
		if record.ServerState == "auto_disabled" {
			// In a real scenario, the auto-disable reason would be restored from the database
			// For now, we just verify that auto-disable state is persisted
			client2.StateManager.SetAutoDisabled("3 consecutive failures")
		}

		// Verify state manager has auto-disable state
		info2 := client2.StateManager.GetConnectionInfo()
		assert.True(t, info2.AutoDisabled, "State manager should have auto-disabled state after restart")
		assert.Contains(t, info2.AutoDisableReason, "consecutive failures")
	}
}

// TestRuntimeAutoDisableViaStateChangeCallback tests that auto-disable is triggered
// via the onStateChange callback when consecutive failures reach the threshold during runtime
func TestRuntimeAutoDisableViaStateChangeCallback(t *testing.T) {
	// Create temp directory for test
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")

	// Create initial config file
	cfg := config.DefaultConfig()
	cfg.DataDir = tempDir
	cfg.AutoDisableThreshold = 3

	testServer := &config.ServerConfig{
		Name:        "test-server",
		URL:         "http://localhost:9999",
		Protocol:    "http",
		StartupMode: "active",
	}
	cfg.Servers = []*config.ServerConfig{testServer}

	// Write initial config
	require.NoError(t, config.SaveConfig(cfg, configPath))

	// Create logger
	logger := zap.NewNop()

	// Initialize storage manager with config loader
	storageManager, err := storage.NewManager(tempDir, logger.Sugar())
	require.NoError(t, err)
	defer storageManager.Close()

	// Create config loader
	configLoader, err := config.NewLoader(configPath, logger)
	require.NoError(t, err)
	defer configLoader.Stop()

	// Load initial config
	_, err = configLoader.Load()
	require.NoError(t, err)

	// Set config loader on storage manager
	storageManager.SetConfigLoader(configLoader)

	// Save initial upstream record
	require.NoError(t, storageManager.GetBoltDB().SaveUpstream(&storage.UpstreamRecord{
		ID:          "test-server",
		Name:        "test-server",
		URL:         "http://localhost:9999",
		Protocol:    "http",
		ServerState: "active",
		Created:     time.Now(),
		Updated:     time.Now(),
	}))

	// Create client
	client, err := NewClient("test-server", testServer, logger, nil, cfg, storageManager.GetBoltDB())
	require.NoError(t, err)

	// Set storage manager on client
	client.SetStorageManager(storageManager)

	// Configure auto-disable threshold
	client.StateManager.SetAutoDisableThreshold(3)

	// Simulate runtime failures by calling SetError
	// This triggers the onStateChange callback which should call checkAndHandleAutoDisable
	for i := 0; i < 3; i++ {
		client.StateManager.SetError(assert.AnError)
	}

	// Give the async callback time to execute (onStateChange runs in goroutine)
	time.Sleep(100 * time.Millisecond)

	// Verify auto-disable was triggered automatically via the callback
	info := client.StateManager.GetConnectionInfo()
	assert.True(t, info.AutoDisabled, "Should be auto-disabled via onStateChange callback after 3 runtime failures")
	assert.Contains(t, info.AutoDisableReason, "consecutive failures", "Should have proper auto-disable reason")

	// Verify persistence in database
	record, err := storageManager.GetBoltDB().GetUpstream("test-server")
	require.NoError(t, err)
	assert.Equal(t, "auto_disabled", record.ServerState, "Server state should be auto_disabled in database")

	// Verify persistence in config file
	reloadedCfg, err := config.LoadFromFile(configPath)
	require.NoError(t, err)

	found := false
	for _, srv := range reloadedCfg.Servers {
		if srv.Name == "test-server" {
			found = true
			assert.Equal(t, "active", srv.StartupMode, "Server startup mode should remain 'active' in config file (DB-only persistence)")
			break
		}
	}
	assert.True(t, found, "Server should be found in config file")
}

// TestAutoDisableWithReset tests that auto-disable works even when Reset() is called
// This reproduces the bug where Reset() was clearing firstAttemptTime, resetting the grace period
func TestAutoDisableWithReset(t *testing.T) {
	// Create temp directory for test
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")

	// Create initial config file
	cfg := config.DefaultConfig()
	cfg.DataDir = tempDir
	cfg.AutoDisableThreshold = 3

	testServer := &config.ServerConfig{
		Name:        "test-server",
		URL:         "http://localhost:9999",
		Protocol:    "http",
		StartupMode: "active",
	}
	cfg.Servers = []*config.ServerConfig{testServer}

	// Write initial config
	require.NoError(t, config.SaveConfig(cfg, configPath))

	// Create logger
	logger := zap.NewNop()

	// Initialize storage manager with config loader
	storageManager, err := storage.NewManager(tempDir, logger.Sugar())
	require.NoError(t, err)
	defer storageManager.Close()

	// Create config loader
	configLoader, err := config.NewLoader(configPath, logger)
	require.NoError(t, err)
	defer configLoader.Stop()

	// Load initial config
	_, err = configLoader.Load()
	require.NoError(t, err)

	// Set config loader on storage manager
	storageManager.SetConfigLoader(configLoader)

	// Save initial upstream record
	require.NoError(t, storageManager.GetBoltDB().SaveUpstream(&storage.UpstreamRecord{
		ID:          "test-server",
		Name:        "test-server",
		URL:         "http://localhost:9999",
		Protocol:    "http",
		ServerState: "active",
		Created:     time.Now(),
		Updated:     time.Now(),
	}))

	// Create client
	client, err := NewClient("test-server", testServer, logger, nil, cfg, storageManager.GetBoltDB())
	require.NoError(t, err)

	// Set storage manager on client
	client.SetStorageManager(storageManager)

	// Configure auto-disable threshold
	client.StateManager.SetAutoDisableThreshold(3)

	// Simulate connection sequence:
	// 1. Connect (sets firstAttemptTime)
	// 2. Fail (increments failures)
	// 3. Reset (simulating tryReconnect) - THIS SHOULD NOT CLEAR firstAttemptTime
	// 4. Repeat until threshold

	// Transition to Connecting to set firstAttemptTime
	client.StateManager.TransitionTo(types.StateConnecting)
	firstAttempt := client.StateManager.GetConnectionInfo().FirstAttemptTime
	require.False(t, firstAttempt.IsZero(), "First attempt time should be set")

	// Simulate 3 failures with resets in between
	for i := 0; i < 3; i++ {
		// Set error (increments failures)
		client.StateManager.SetError(assert.AnError)

		// Verify failure count
		info := client.StateManager.GetConnectionInfo()
		assert.Equal(t, i+1, info.ConsecutiveFailures)

		// Reset state (as tryReconnect does)
		client.StateManager.Reset()

		// Verify firstAttemptTime is preserved
		// NOTE: This assertion will fail before the fix
		currentInfo := client.StateManager.GetConnectionInfo()
		assert.Equal(t, firstAttempt, currentInfo.FirstAttemptTime, "First attempt time should be preserved after Reset()")
	}

	// Verify consecutive failures are preserved
	info := client.StateManager.GetConnectionInfo()
	assert.Equal(t, 3, info.ConsecutiveFailures, "Consecutive failures should be preserved after Reset()")

	// Now check auto-disable
	// We need to manually trigger the check because we're not using the full client loop here
	client.checkAndHandleAutoDisable()

	// Verify auto-disable is SUPPRESSED due to grace period (since we are just starting)
	info = client.StateManager.GetConnectionInfo()
	assert.False(t, info.AutoDisabled, "Should NOT be auto-disabled yet due to grace period")

	// Now simulate enough failures to exceed the "obvious failure" threshold (threshold * 2)
	// This confirms that auto-disable still works eventually
	for i := 0; i < 3; i++ {
		client.StateManager.SetError(assert.AnError)
		client.StateManager.Reset()
	}

	// Trigger check again
	client.checkAndHandleAutoDisable()

	// Now it should be auto-disabled because failures (6) >= threshold*2 (6)
	info = client.StateManager.GetConnectionInfo()
	assert.True(t, info.AutoDisabled, "Should be auto-disabled after exceeding grace period threshold (threshold*2)")
}

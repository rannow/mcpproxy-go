package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"mcpproxy-go/internal/config"
)

func TestManager_ClearAutoDisable(t *testing.T) {
	logger := zap.NewNop().Sugar()
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")

	// Create initial config with auto-disabled server
	cfg := config.DefaultConfig()
	cfg.DataDir = tempDir
	cfg.Servers = []*config.ServerConfig{
		{
			Name:        "test-server",
			Protocol:    "http",
			URL:         "http://localhost:8080",
			StartupMode: "auto_disabled",
			Created:     time.Now(),
		},
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(configPath, data, 0644))

	// Create manager and config loader
	manager, err := NewManager(tempDir, logger)
	require.NoError(t, err)
	defer manager.Close()

	// Save server to database
	require.NoError(t, manager.SaveUpstreamServer(cfg.Servers[0]))

	// Create and set config loader
	zapLogger := zap.NewNop()
	loader, err := config.NewLoader(configPath, zapLogger)
	require.NoError(t, err)
	defer loader.Stop()

	_, err = loader.Load()
	require.NoError(t, err)

	manager.SetConfigLoader(loader)

	// Clear auto-disable
	err = manager.ClearAutoDisable("test-server")
	require.NoError(t, err)

	// Verify database was updated
	record, err := manager.GetBoltDB().GetUpstream("test-server")
	require.NoError(t, err)
	assert.Equal(t, "active", record.ServerState, "Database server state should be active (not auto_disabled)")

	// Verify config file was updated
	updatedCfg := loader.GetConfig()
	found := false
	for _, server := range updatedCfg.Servers {
		if server.Name == "test-server" {
			found = true
			assert.Equal(t, "active", server.StartupMode, "Config startup mode should be active (not auto_disabled)")
			break
		}
	}
	assert.True(t, found, "Server should exist in config")
}

func TestManager_UpdateServerState(t *testing.T) {
	logger := zap.NewNop().Sugar()
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")

	// Create initial config
	cfg := config.DefaultConfig()
	cfg.DataDir = tempDir
	cfg.Servers = []*config.ServerConfig{
		{
			Name:        "test-server",
			Protocol:    "http",
			URL:         "http://localhost:8080",
			StartupMode: "active",
			Created:     time.Now(),
		},
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(configPath, data, 0644))

	// Create manager and config loader
	manager, err := NewManager(tempDir, logger)
	require.NoError(t, err)
	defer manager.Close()

	require.NoError(t, manager.SaveUpstreamServer(cfg.Servers[0]))

	zapLogger := zap.NewNop()
	loader, err := config.NewLoader(configPath, zapLogger)
	require.NoError(t, err)
	defer loader.Stop()

	_, err = loader.Load()
	require.NoError(t, err)

	manager.SetConfigLoader(loader)

	// Update startup mode to disabled with reason
	err = manager.UpdateServerState("test-server", "Too many failures")
	require.NoError(t, err)

	// Verify database was updated
	record, err := manager.GetBoltDB().GetUpstream("test-server")
	require.NoError(t, err)
	assert.Equal(t, "auto_disabled", record.ServerState, "Database server state should be auto_disabled")

	// Verify config file was updated
	updatedCfg := loader.GetConfig()
	found := false
	for _, server := range updatedCfg.Servers {
		if server.Name == "test-server" {
			found = true
			assert.Equal(t, "auto_disabled", server.StartupMode, "Config startup mode should be auto_disabled")
			break
		}
	}
	assert.True(t, found, "Server should exist in config")
}

func TestManager_UpdateServerState_Rollback(t *testing.T) {
	logger := zap.NewNop().Sugar()
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")

	// Create initial config
	cfg := config.DefaultConfig()
	cfg.DataDir = tempDir
	cfg.Servers = []*config.ServerConfig{
		{
			Name:        "test-server",
			Protocol:    "http",
			URL:         "http://localhost:8080",
			StartupMode: "active",
			Created:     time.Now(),
		},
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(configPath, data, 0644))

	// Create manager
	manager, err := NewManager(tempDir, logger)
	require.NoError(t, err)
	defer manager.Close()

	require.NoError(t, manager.SaveUpstreamServer(cfg.Servers[0]))

	// Create read-only directory for config to force failure
	tempDir2 := t.TempDir()
	readOnlyPath := filepath.Join(tempDir2, "config.json")
	err = os.Chmod(tempDir2, 0444)
	require.NoError(t, err)
	defer os.Chmod(tempDir2, 0755)

	zapLogger := zap.NewNop()
	loader2, err := config.NewLoader(readOnlyPath, zapLogger)
	require.NoError(t, err)
	defer loader2.Stop()

	manager.SetConfigLoader(loader2)

	// Try to update - should fail on config file write
	err = manager.UpdateServerState("test-server", "Test reason")
	assert.Error(t, err, "Should fail to write config file")

	// Verify database was rolled back
	record, err := manager.GetBoltDB().GetUpstream("test-server")
	require.NoError(t, err)
	assert.Equal(t, "active", record.ServerState, "Database server state should be rolled back to active")
}

func TestManager_EnableUpstreamServer_WithConfigLoader(t *testing.T) {
	logger := zap.NewNop().Sugar()
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")

	// Create initial config with disabled and auto-disabled server
	cfg := config.DefaultConfig()
	cfg.DataDir = tempDir
	cfg.Servers = []*config.ServerConfig{
		{
			Name:        "test-server",
			Protocol:    "http",
			URL:         "http://localhost:8080",
			StartupMode: "auto_disabled",
			Created:     time.Now(),
		},
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(configPath, data, 0644))

	// Create manager and config loader
	manager, err := NewManager(tempDir, logger)
	require.NoError(t, err)
	defer manager.Close()

	require.NoError(t, manager.SaveUpstreamServer(cfg.Servers[0]))

	zapLogger := zap.NewNop()
	loader, err := config.NewLoader(configPath, zapLogger)
	require.NoError(t, err)
	defer loader.Stop()

	_, err = loader.Load()
	require.NoError(t, err)

	manager.SetConfigLoader(loader)

	// Enable server - should also clear auto-disable
	err = manager.EnableUpstreamServer("test-server", true)
	require.NoError(t, err)

	// Verify database
	record, err := manager.GetBoltDB().GetUpstream("test-server")
	require.NoError(t, err)
	assert.Equal(t, "active", record.ServerState, "Server state should be active (enabled and auto-disable cleared)")

	// Verify config file
	updatedCfg := loader.GetConfig()
	found := false
	for _, server := range updatedCfg.Servers {
		if server.Name == "test-server" {
			found = true
			assert.Equal(t, "active", server.StartupMode, "Config startup mode should be active (enabled and auto-disable cleared)")
			break
		}
	}
	assert.True(t, found, "Server should exist in config")
}

func TestManager_StopServer(t *testing.T) {
	logger := zap.NewNop().Sugar()
	tempDir := t.TempDir()

	// Create manager
	manager, err := NewManager(tempDir, logger)
	require.NoError(t, err)
	defer manager.Close()

	// Create test server
	serverCfg := &config.ServerConfig{
		Name:        "test-server",
		Protocol:    "http",
		URL:         "http://localhost:8080",
		StartupMode: "active",
		Created:     time.Now(),
	}
	require.NoError(t, manager.SaveUpstreamServer(serverCfg))

	// Stop server
	err = manager.StopServer("test-server")
	require.NoError(t, err)

	// Verify server still exists in database
	record, err := manager.GetBoltDB().GetUpstream("test-server")
	require.NoError(t, err)
	assert.NotNil(t, record)
}

func TestManager_StartServer(t *testing.T) {
	logger := zap.NewNop().Sugar()
	tempDir := t.TempDir()

	// Create manager
	manager, err := NewManager(tempDir, logger)
	require.NoError(t, err)
	defer manager.Close()

	// Create enabled server
	serverCfg := &config.ServerConfig{
		Name:        "test-server",
		Protocol:    "http",
		URL:         "http://localhost:8080",
		StartupMode: "active",
		Created:     time.Now(),
	}
	require.NoError(t, manager.SaveUpstreamServer(serverCfg))

	// Start server
	err = manager.StartServer("test-server")
	require.NoError(t, err)

	// Verify server exists
	record, err := manager.GetBoltDB().GetUpstream("test-server")
	require.NoError(t, err)
	assert.NotNil(t, record)
	assert.Equal(t, "active", record.ServerState)
}

func TestManager_StartServer_Disabled(t *testing.T) {
	logger := zap.NewNop().Sugar()
	tempDir := t.TempDir()

	// Create manager
	manager, err := NewManager(tempDir, logger)
	require.NoError(t, err)
	defer manager.Close()

	// Create disabled server
	serverCfg := &config.ServerConfig{
		Name:        "test-server",
		Protocol:    "http",
		URL:         "http://localhost:8080",
		StartupMode: "disabled",
		Created:     time.Now(),
	}
	require.NoError(t, manager.SaveUpstreamServer(serverCfg))

	// Try to start disabled server - should fail
	err = manager.StartServer("test-server")
	assert.Error(t, err, "Should not be able to start disabled server")
	assert.Contains(t, err.Error(), "disabled")
}

// TestManager_ConfigFilePriority tests that config file values have priority over database values
// EXCEPT for auto_disabled state which is a runtime protection mechanism
func TestManager_ConfigFilePriority(t *testing.T) {
	logger := zap.NewNop().Sugar()
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")

	// Create config file with server set to "active"
	cfg := config.DefaultConfig()
	cfg.DataDir = tempDir
	cfg.Servers = []*config.ServerConfig{
		{
			Name:        "test-server",
			Protocol:    "http",
			URL:         "http://localhost:8080",
			StartupMode: "active", // Config says active
			Created:     time.Now(),
		},
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(configPath, data, 0644))

	// Create manager
	manager, err := NewManager(tempDir, logger)
	require.NoError(t, err)
	defer manager.Close()

	// Save server to database with "disabled" state (simulating old database state)
	disabledServer := &config.ServerConfig{
		Name:        "test-server",
		Protocol:    "http",
		URL:         "http://localhost:8080",
		StartupMode: "disabled", // Database says disabled
		Created:     time.Now(),
	}
	require.NoError(t, manager.SaveUpstreamServer(disabledServer))

	// Verify database has disabled state
	record, err := manager.GetBoltDB().GetUpstream("test-server")
	require.NoError(t, err)
	assert.Equal(t, "disabled", record.ServerState, "Database should have disabled state")

	// Create and set config loader
	zapLogger := zap.NewNop()
	loader, err := config.NewLoader(configPath, zapLogger)
	require.NoError(t, err)
	defer loader.Stop()

	_, err = loader.Load()
	require.NoError(t, err)

	manager.SetConfigLoader(loader)

	// Now test ListUpstreamServers - config file should take priority
	servers, err := manager.ListUpstreamServers()
	require.NoError(t, err)
	require.Len(t, servers, 1)
	assert.Equal(t, "active", servers[0].StartupMode,
		"Config file startup_mode (active) should override database server_state (disabled)")

	// Also test GetUpstreamServer
	server, err := manager.GetUpstreamServer("test-server")
	require.NoError(t, err)
	assert.Equal(t, "active", server.StartupMode,
		"Config file startup_mode (active) should override database server_state (disabled)")
}

// TestManager_AutoDisabledPreserved tests that auto_disabled state from database is preserved
// even when config file says active (runtime protection mechanism)
func TestManager_AutoDisabledPreserved(t *testing.T) {
	logger := zap.NewNop().Sugar()
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")

	// Create config file with server set to "active"
	cfg := config.DefaultConfig()
	cfg.DataDir = tempDir
	cfg.Servers = []*config.ServerConfig{
		{
			Name:        "test-server",
			Protocol:    "http",
			URL:         "http://localhost:8080",
			StartupMode: "active", // Config says active
			Created:     time.Now(),
		},
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(configPath, data, 0644))

	// Create manager
	manager, err := NewManager(tempDir, logger)
	require.NoError(t, err)
	defer manager.Close()

	// Save server to database with "auto_disabled" state (simulating runtime auto-disable)
	autoDisabledServer := &config.ServerConfig{
		Name:        "test-server",
		Protocol:    "http",
		URL:         "http://localhost:8080",
		StartupMode: "auto_disabled", // Database says auto_disabled (runtime protection)
		Created:     time.Now(),
	}
	require.NoError(t, manager.SaveUpstreamServer(autoDisabledServer))

	// Verify database has auto_disabled state
	record, err := manager.GetBoltDB().GetUpstream("test-server")
	require.NoError(t, err)
	assert.Equal(t, "auto_disabled", record.ServerState, "Database should have auto_disabled state")

	// Create and set config loader
	zapLogger := zap.NewNop()
	loader, err := config.NewLoader(configPath, zapLogger)
	require.NoError(t, err)
	defer loader.Stop()

	_, err = loader.Load()
	require.NoError(t, err)

	manager.SetConfigLoader(loader)

	// Now test ListUpstreamServers - auto_disabled from database should be preserved
	servers, err := manager.ListUpstreamServers()
	require.NoError(t, err)
	require.Len(t, servers, 1)
	assert.Equal(t, "auto_disabled", servers[0].StartupMode,
		"Database auto_disabled state should be preserved (runtime protection overrides config)")

	// Also test GetUpstreamServer
	server, err := manager.GetUpstreamServer("test-server")
	require.NoError(t, err)
	assert.Equal(t, "auto_disabled", server.StartupMode,
		"Database auto_disabled state should be preserved (runtime protection overrides config)")
}

// TestManager_ConfigAutoDisabledOverrides tests that when config file explicitly sets auto_disabled,
// the config file takes priority (user has confirmed the auto-disable)
func TestManager_ConfigAutoDisabledOverrides(t *testing.T) {
	logger := zap.NewNop().Sugar()
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")

	// Create config file with server set to "auto_disabled" (user confirmed)
	cfg := config.DefaultConfig()
	cfg.DataDir = tempDir
	cfg.Servers = []*config.ServerConfig{
		{
			Name:        "test-server",
			Protocol:    "http",
			URL:         "http://localhost:8080",
			StartupMode: "auto_disabled", // Config says auto_disabled (user confirmed)
			Created:     time.Now(),
		},
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(configPath, data, 0644))

	// Create manager
	manager, err := NewManager(tempDir, logger)
	require.NoError(t, err)
	defer manager.Close()

	// Save server to database with "auto_disabled" state
	autoDisabledServer := &config.ServerConfig{
		Name:        "test-server",
		Protocol:    "http",
		URL:         "http://localhost:8080",
		StartupMode: "auto_disabled",
		Created:     time.Now(),
	}
	require.NoError(t, manager.SaveUpstreamServer(autoDisabledServer))

	// Create and set config loader
	zapLogger := zap.NewNop()
	loader, err := config.NewLoader(configPath, zapLogger)
	require.NoError(t, err)
	defer loader.Stop()

	_, err = loader.Load()
	require.NoError(t, err)

	manager.SetConfigLoader(loader)

	// Now test ListUpstreamServers - both config and db say auto_disabled, result should be auto_disabled
	servers, err := manager.ListUpstreamServers()
	require.NoError(t, err)
	require.Len(t, servers, 1)
	assert.Equal(t, "auto_disabled", servers[0].StartupMode,
		"When config says auto_disabled and db says auto_disabled, result should be auto_disabled")
}

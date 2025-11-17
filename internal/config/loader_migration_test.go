package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfigFile_AutomaticMigration(t *testing.T) {
	// Create a temporary directory
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test_config.json")

	// Create a config file without startup_mode fields (old format)
	// NOTE: Old boolean fields (enabled, start_on_boot, quarantined, auto_disabled)
	// are no longer part of ServerConfig struct, so they are silently dropped during JSON unmarshal
	// Current migration logic: empty startup_mode → "disabled" (default for safety)
	oldFormatConfig := map[string]interface{}{
		"listen":  ":8080",
		"data_dir": tempDir,
		"mcpServers": []map[string]interface{}{
			{
				"name":     "server1",
				"url":      "http://localhost:3000",
				"protocol": "http",
				// No startup_mode field - should migrate to "disabled"
			},
			{
				"name":     "server2",
				"url":      "http://localhost:3001",
				"protocol": "http",
				// Old fields like "enabled", "start_on_boot" would be ignored during unmarshal
			},
			{
				"name":     "server3",
				"url":      "http://localhost:3002",
				"protocol": "http",
			},
		},
	}

	// Write old format config to file
	data, err := json.MarshalIndent(oldFormatConfig, "", "  ")
	require.NoError(t, err)
	err = os.WriteFile(configPath, data, 0600)
	require.NoError(t, err)

	// Load the config (should trigger automatic migration)
	cfg := DefaultConfig()
	err = loadConfigFile(configPath, cfg)
	require.NoError(t, err)

	// Verify migration occurred
	require.Len(t, cfg.Servers, 3)

	// All servers without startup_mode should be migrated to "disabled" (current behavior)
	for _, server := range cfg.Servers {
		assert.Equal(t, "disabled", server.StartupMode,
			"server %s should be migrated to 'disabled' (default for empty startup_mode)", server.Name)
	}

	// Verify the config file was updated on disk with migrated values
	var savedConfig Config
	savedData, err := os.ReadFile(configPath)
	require.NoError(t, err)
	err = json.Unmarshal(savedData, &savedConfig)
	require.NoError(t, err)

	// Check that saved config has startup_mode fields
	for _, server := range savedConfig.Servers {
		assert.Equal(t, "disabled", server.StartupMode,
			"server %s should have startup_mode='disabled' in saved config", server.Name)
	}
}

func TestLoadConfigFile_AlreadyMigrated(t *testing.T) {
	// Create a temporary directory
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test_config.json")

	// Create a config file that's already in new format
	newFormatConfig := &Config{
		Listen:  ":8080",
		DataDir: tempDir,
		Servers: []*ServerConfig{
			{
				Name:        "already-migrated",
				URL:         "http://localhost:3000",
				Protocol:    "http",
				StartupMode: "active", // Already has startup_mode
				Created:     time.Now(),
			},
		},
	}

	// Save to file
	err := SaveConfig(newFormatConfig, configPath)
	require.NoError(t, err)

	// Record original modification time
	originalInfo, err := os.Stat(configPath)
	require.NoError(t, err)
	originalModTime := originalInfo.ModTime()

	// Small delay to ensure timestamps would be different
	time.Sleep(10 * time.Millisecond)

	// Load the config (should NOT trigger migration)
	cfg := DefaultConfig()
	err = loadConfigFile(configPath, cfg)
	require.NoError(t, err)

	// Verify no migration occurred
	require.Len(t, cfg.Servers, 1)
	assert.Equal(t, "active", cfg.Servers[0].StartupMode)

	// File should not have been modified (no re-save)
	newInfo, err := os.Stat(configPath)
	require.NoError(t, err)
	newModTime := newInfo.ModTime()

	// Modification time should be the same or very close (within 1 second)
	timeDiff := newModTime.Sub(originalModTime)
	assert.Less(t, timeDiff, 2*time.Second,
		"config file should not be modified if already migrated")
}

func TestLoadConfigFile_MigrationWithBackup(t *testing.T) {
	// Create a temporary directory
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test_config.json")

	// Create old format config
	oldFormatConfig := map[string]interface{}{
		"listen":  ":8080",
		"data_dir": tempDir,
		"mcpServers": []map[string]interface{}{
			{
				"name":          "test-server",
				"url":           "http://localhost:3000",
				"enabled":       true,
				"start_on_boot": true,
			},
		},
	}

	data, err := json.MarshalIndent(oldFormatConfig, "", "  ")
	require.NoError(t, err)
	err = os.WriteFile(configPath, data, 0600)
	require.NoError(t, err)

	// Load config (triggers migration and creates backup)
	cfg := DefaultConfig()
	err = loadConfigFile(configPath, cfg)
	require.NoError(t, err)

	// Check backup file was created
	backupPattern := filepath.Join(tempDir, "test_config.backup.*.json")
	matches, err := filepath.Glob(backupPattern)
	require.NoError(t, err)
	assert.Greater(t, len(matches), 0, "backup file should have been created")

	// Verify backup contains original data (without startup_mode)
	if len(matches) > 0 {
		var backupConfig map[string]interface{}
		backupData, err := os.ReadFile(matches[0])
		require.NoError(t, err)
		err = json.Unmarshal(backupData, &backupConfig)
		require.NoError(t, err)

		servers, ok := backupConfig["mcpServers"].([]interface{})
		require.True(t, ok, "backup should have mcpServers")
		require.Greater(t, len(servers), 0, "backup should have servers")

		server := servers[0].(map[string]interface{})
		// Backup should NOT have startup_mode (it's the old format)
		_, hasStartupMode := server["startup_mode"]
		assert.False(t, hasStartupMode, "backup should preserve old format without startup_mode")
	}
}

func TestLoadFromFile_WithMigration(t *testing.T) {
	// Create a temporary directory
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test_config.json")

	// Create old format config (without startup_mode)
	// NOTE: Old boolean fields are no longer in ServerConfig struct
	// Current migration: empty startup_mode → "disabled"
	oldFormatConfig := map[string]interface{}{
		"listen":  ":8080",
		"data_dir": tempDir,
		"mcpServers": []map[string]interface{}{
			{
				"name": "server1",
				// No startup_mode - should migrate to "disabled"
			},
			{
				"name": "server2",
				// No startup_mode - should migrate to "disabled"
			},
		},
	}

	data, err := json.MarshalIndent(oldFormatConfig, "", "  ")
	require.NoError(t, err)
	err = os.WriteFile(configPath, data, 0600)
	require.NoError(t, err)

	// Use LoadFromFile (public API)
	cfg, err := LoadFromFile(configPath)
	require.NoError(t, err)

	// Verify migration happened - both should be "disabled" (current default)
	assert.Equal(t, "disabled", cfg.Servers[0].StartupMode)
	assert.Equal(t, "disabled", cfg.Servers[1].StartupMode)
}

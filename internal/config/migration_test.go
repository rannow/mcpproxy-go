package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestMigrateConfig_NoMigrationNeeded(t *testing.T) {
	config := &Config{
		Servers: []*ServerConfig{
			{
				Name:        "already-migrated",
				StartupMode: "active",
			},
		},
	}

	result := MigrateConfig(config, nil)
	if result.Migrated {
		t.Error("expected no migration for already migrated config")
	}
	if result.ServersChanged != 0 {
		t.Errorf("expected 0 servers changed, got %d", result.ServersChanged)
	}
}

func TestMigrateConfig_EmptyStartupMode(t *testing.T) {
	server := &ServerConfig{
		Name:        "test-server",
		StartupMode: "", // Empty startup mode needs migration
	}

	migrateServer(server)

	if server.StartupMode != "disabled" {
		t.Errorf("expected startup_mode='disabled' for empty startup_mode, got '%s'", server.StartupMode)
	}
}

func TestMigrateConfig_StartupModeAlreadySet(t *testing.T) {
	server := &ServerConfig{
		Name:        "test-server",
		StartupMode: "lazy_loading",
	}

	needsMigration := serverNeedsMigration(server)
	if needsMigration {
		t.Error("expected no migration needed for server with startup_mode already set")
	}
}

func TestMigrateConfig_QuarantinedMode(t *testing.T) {
	server := &ServerConfig{
		Name:        "test-server",
		StartupMode: "quarantined",
	}

	needsMigration := serverNeedsMigration(server)
	if needsMigration {
		t.Error("expected no migration needed for server with startup_mode='quarantined'")
	}
}

func TestMigrateConfig_AutoDisabledMode(t *testing.T) {
	server := &ServerConfig{
		Name:        "test-server",
		StartupMode: "auto_disabled",
	}

	needsMigration := serverNeedsMigration(server)
	if needsMigration {
		t.Error("expected no migration needed for server with startup_mode='auto_disabled'")
	}
}

func TestMigrateConfig_DisabledMode(t *testing.T) {
	server := &ServerConfig{
		Name:        "test-server",
		StartupMode: "disabled",
	}

	needsMigration := serverNeedsMigration(server)
	if needsMigration {
		t.Error("expected no migration needed for server with startup_mode='disabled'")
	}
}

func TestMigrateConfig_ActiveMode(t *testing.T) {
	server := &ServerConfig{
		Name:        "test-server",
		StartupMode: "active",
	}

	needsMigration := serverNeedsMigration(server)
	if needsMigration {
		t.Error("expected no migration needed for server with startup_mode='active'")
	}
}

func TestMigrateConfig_MultipleServers(t *testing.T) {
	config := &Config{
		Servers: []*ServerConfig{
			{Name: "server1", StartupMode: "active"},
			{Name: "server2", StartupMode: "lazy_loading"},
			{Name: "server3", StartupMode: "disabled"},
			{Name: "server4", StartupMode: "quarantined"},
			{Name: "server5", StartupMode: "auto_disabled"},
		},
	}

	result := MigrateConfig(config, nil)

	// All servers already have startup_mode set, so no migration should occur
	if result.Migrated {
		t.Error("expected no migration to occur for already migrated config")
	}
	if result.ServersChanged != 0 {
		t.Errorf("expected 0 servers changed, got %d", result.ServersChanged)
	}
}

func TestMigrateConfig_EmptyStartupModes(t *testing.T) {
	config := &Config{
		Servers: []*ServerConfig{
			{Name: "server1", StartupMode: ""},
			{Name: "server2", StartupMode: ""},
			{Name: "server3", StartupMode: ""},
		},
	}

	result := MigrateConfig(config, nil)

	if !result.Migrated {
		t.Error("expected migration to occur for servers with empty startup_mode")
	}
	if result.ServersChanged != 3 {
		t.Errorf("expected 3 servers changed, got %d", result.ServersChanged)
	}

	// All servers should be migrated to "disabled" (the default)
	for _, server := range config.Servers {
		if server.StartupMode != "disabled" {
			t.Errorf("server %s: expected startup_mode='disabled', got '%s'",
				server.Name, server.StartupMode)
		}
	}
}

func TestValidateStartupMode(t *testing.T) {
	validModes := []string{"active", "disabled", "quarantined", "auto_disabled", "lazy_loading"}

	for _, mode := range validModes {
		if err := ValidateStartupMode(mode); err != nil {
			t.Errorf("expected mode '%s' to be valid, got error: %v", mode, err)
		}
	}

	invalidModes := []string{"", "invalid", "ACTIVE", "Active", "enabled"}

	for _, mode := range invalidModes {
		if err := ValidateStartupMode(mode); err == nil {
			t.Errorf("expected mode '%s' to be invalid, but validation passed", mode)
		}
	}
}

func TestCreateBackup(t *testing.T) {
	// Create temporary config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "mcp_config.json")
	content := []byte(`{"test": "data"}`)

	if err := os.WriteFile(configPath, content, 0600); err != nil {
		t.Fatalf("failed to create test config: %v", err)
	}

	// Create backup
	backupPath, err := CreateBackup(configPath, nil)
	if err != nil {
		t.Fatalf("CreateBackup failed: %v", err)
	}

	// Verify backup exists
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		t.Error("backup file was not created")
	}

	// Verify backup content matches original
	backupContent, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatalf("failed to read backup: %v", err)
	}

	if string(backupContent) != string(content) {
		t.Error("backup content doesn't match original")
	}

	// Verify backup filename format
	if !strings.Contains(backupPath, ".backup-") {
		t.Errorf("backup filename doesn't contain .backup-: %s", backupPath)
	}
}

func TestRollbackMigration(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "mcp_config.json")
	originalContent := []byte(`{"version": "old"}`)
	newContent := []byte(`{"version": "new"}`)

	// Create original config
	if err := os.WriteFile(configPath, originalContent, 0600); err != nil {
		t.Fatalf("failed to create test config: %v", err)
	}

	// Create backup
	backupPath, err := CreateBackup(configPath, nil)
	if err != nil {
		t.Fatalf("CreateBackup failed: %v", err)
	}

	// Modify config (simulate migration)
	if err := os.WriteFile(configPath, newContent, 0600); err != nil {
		t.Fatalf("failed to modify config: %v", err)
	}

	// Verify config is modified
	currentContent, _ := os.ReadFile(configPath)
	if string(currentContent) != string(newContent) {
		t.Error("config was not modified as expected")
	}

	// Rollback
	if err := RollbackMigration(configPath, backupPath, nil); err != nil {
		t.Fatalf("RollbackMigration failed: %v", err)
	}

	// Verify config is restored
	restoredContent, _ := os.ReadFile(configPath)
	if string(restoredContent) != string(originalContent) {
		t.Error("config was not restored to original content")
	}
}

func TestCleanOldBackups(t *testing.T) {
	tempDir := t.TempDir()

	// Create some backup files with different ages
	now := time.Now()

	// Recent backup (should not be deleted)
	recentBackup := filepath.Join(tempDir, "mcp_config.json.backup-20250115-120000")
	if err := os.WriteFile(recentBackup, []byte("recent"), 0600); err != nil {
		t.Fatalf("failed to create recent backup: %v", err)
	}
	if err := os.Chtimes(recentBackup, now, now); err != nil {
		t.Fatalf("failed to set recent backup time: %v", err)
	}

	// Old backup (should be deleted)
	oldBackup := filepath.Join(tempDir, "mcp_config.json.backup-20240101-120000")
	if err := os.WriteFile(oldBackup, []byte("old"), 0600); err != nil {
		t.Fatalf("failed to create old backup: %v", err)
	}
	oldTime := now.Add(-90 * 24 * time.Hour) // 90 days ago
	if err := os.Chtimes(oldBackup, oldTime, oldTime); err != nil {
		t.Fatalf("failed to set old backup time: %v", err)
	}

	// Non-backup file (should not be touched)
	normalFile := filepath.Join(tempDir, "mcp_config.json")
	if err := os.WriteFile(normalFile, []byte("normal"), 0600); err != nil {
		t.Fatalf("failed to create normal file: %v", err)
	}

	// Clean backups older than 30 days
	if err := CleanOldBackups(tempDir, 30*24*time.Hour, nil); err != nil {
		t.Fatalf("CleanOldBackups failed: %v", err)
	}

	// Verify recent backup still exists
	if _, err := os.Stat(recentBackup); os.IsNotExist(err) {
		t.Error("recent backup was deleted (should have been kept)")
	}

	// Verify old backup was deleted
	if _, err := os.Stat(oldBackup); !os.IsNotExist(err) {
		t.Error("old backup still exists (should have been deleted)")
	}

	// Verify normal file wasn't touched
	if _, err := os.Stat(normalFile); os.IsNotExist(err) {
		t.Error("normal file was deleted (should have been kept)")
	}
}

func TestDescribeOldFormat(t *testing.T) {
	tests := []struct {
		name     string
		server   *ServerConfig
		expected string
	}{
		{
			name:     "empty_startup_mode",
			server:   &ServerConfig{StartupMode: ""},
			expected: "startup_mode not set (migration needed)",
		},
		{
			name:     "quarantined_set",
			server:   &ServerConfig{StartupMode: "quarantined"},
			expected: "unknown migration reason",
		},
		{
			name:     "auto_disabled_set",
			server:   &ServerConfig{StartupMode: "auto_disabled"},
			expected: "unknown migration reason",
		},
		{
			name:     "disabled_set",
			server:   &ServerConfig{StartupMode: "disabled"},
			expected: "unknown migration reason",
		},
		{
			name:     "active_set",
			server:   &ServerConfig{StartupMode: "active"},
			expected: "unknown migration reason",
		},
		{
			name:     "lazy_loading_set",
			server:   &ServerConfig{StartupMode: "lazy_loading"},
			expected: "unknown migration reason",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := describeOldFormat(tt.server)
			if result != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestMigrateConfig_Idempotent(t *testing.T) {
	config := &Config{
		Servers: []*ServerConfig{
			{Name: "test", StartupMode: ""},
		},
	}

	// First migration
	result1 := MigrateConfig(config, nil)
	if !result1.Migrated {
		t.Error("expected first migration to occur")
	}
	if config.Servers[0].StartupMode != "disabled" {
		t.Errorf("expected startup_mode='disabled', got '%s'", config.Servers[0].StartupMode)
	}

	// Second migration should be idempotent
	result2 := MigrateConfig(config, nil)
	if result2.Migrated {
		t.Error("expected second migration to be skipped (idempotent)")
	}
}

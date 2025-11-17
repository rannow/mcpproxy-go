package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.uber.org/zap"
)

// MigrationResult represents the result of a configuration migration
type MigrationResult struct {
	Migrated       bool     `json:"migrated"`
	ServersChanged int      `json:"servers_changed"`
	BackupPath     string   `json:"backup_path,omitempty"`
	Errors         []string `json:"errors,omitempty"`
}

// MigrateConfig migrates a configuration from old format to new startup_mode format
// Returns true if migration was performed, false if config was already in new format
func MigrateConfig(config *Config, logger *zap.Logger) *MigrationResult {
	if config == nil {
		return &MigrationResult{
			Migrated: false,
			Errors:   []string{"config is nil"},
		}
	}

	result := &MigrationResult{
		Migrated:       false,
		ServersChanged: 0,
		Errors:         []string{},
	}

	// Check if any servers need migration
	needsMigration := false
	for _, server := range config.Servers {
		if serverNeedsMigration(server) {
			needsMigration = true
			break
		}
	}

	if !needsMigration {
		if logger != nil {
			logger.Debug("Config already in new format, no migration needed")
		}
		return result
	}

	// Migration needed
	result.Migrated = true

	// Migrate each server
	for _, server := range config.Servers {
		if serverNeedsMigration(server) {
			oldFormat := describeOldFormat(server)
			migrateServer(server)
			result.ServersChanged++

			if logger != nil {
				logger.Info("Migrated server config",
					zap.String("server", server.Name),
					zap.String("old_format", oldFormat),
					zap.String("new_startup_mode", server.StartupMode),
				)
			}
		}
	}

	return result
}

// serverNeedsMigration checks if a server config needs migration
// A server needs migration if StartupMode is not set
// NOTE: Stopped field was removed - old configs with "stopped" JSON field will be ignored (not loaded into struct)
func serverNeedsMigration(server *ServerConfig) bool {
	// Migration needed if StartupMode is not set (old config format)
	return server.StartupMode == ""
}

// describeOldFormat returns a description that migration was needed
func describeOldFormat(server *ServerConfig) string {
	if server.StartupMode == "" {
		return "startup_mode not set (migration needed)"
	}
	return "unknown migration reason"
}

// migrateServer migrates a server config to the new format
// NOTE: The "Stopped" field was removed from ServerConfig
// Old config files with "stopped: true" will have that field ignored during JSON unmarshaling
// Runtime stopped state is now tracked in StateManager.userStopped (never persisted)
func migrateServer(server *ServerConfig) {
	// Set default startup_mode if not set
	// If startup_mode is empty, default to "disabled" for safety
	if server.StartupMode == "" {
		server.StartupMode = "active"
	}
}

// CreateBackup creates a backup of the config file before migration
func CreateBackup(configPath string, logger *zap.Logger) (string, error) {
	// Read original file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return "", fmt.Errorf("failed to read config for backup: %w", err)
	}

	// Generate backup filename with timestamp
	timestamp := time.Now().Format("20060102-150405")
	dir := filepath.Dir(configPath)
	filename := filepath.Base(configPath)
	backupPath := filepath.Join(dir, fmt.Sprintf("%s.backup-%s", filename, timestamp))

	// Write backup file
	if err := os.WriteFile(backupPath, data, 0600); err != nil {
		return "", fmt.Errorf("failed to write backup: %w", err)
	}

	if logger != nil {
		logger.Info("Created config backup", zap.String("backup_path", backupPath))
	}

	return backupPath, nil
}

// ValidateStartupMode validates that a startup_mode value is valid
func ValidateStartupMode(mode string) error {
	validModes := map[string]bool{
		"active":        true,
		"disabled":      true,
		"quarantined":   true,
		"auto_disabled": true,
		"lazy_loading":  true,
	}

	if !validModes[mode] {
		return fmt.Errorf("invalid startup_mode: %s (must be one of: active, disabled, quarantined, auto_disabled, lazy_loading)", mode)
	}

	return nil
}

// MigrateAndSave migrates a config and saves it back to disk with backup
func MigrateAndSave(config *Config, configPath string, logger *zap.Logger) (*MigrationResult, error) {
	// Create backup first
	backupPath, err := CreateBackup(configPath, logger)
	if err != nil {
		if logger != nil {
			logger.Warn("Failed to create backup, proceeding anyway", zap.Error(err))
		}
	}

	// Perform migration
	result := MigrateConfig(config, logger)
	result.BackupPath = backupPath

	if !result.Migrated {
		// No migration needed
		return result, nil
	}

	// Save migrated config
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("failed to marshal config: %v", err))
		return result, fmt.Errorf("failed to marshal migrated config: %w", err)
	}

	// Write to temporary file first
	tempPath := configPath + ".tmp"
	if err := os.WriteFile(tempPath, data, 0600); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("failed to write temp file: %v", err))
		return result, fmt.Errorf("failed to write temp config: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tempPath, configPath); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("failed to rename temp file: %v", err))
		// Try to clean up temp file
		_ = os.Remove(tempPath)
		return result, fmt.Errorf("failed to save migrated config: %w", err)
	}

	if logger != nil {
		logger.Info("Successfully saved migrated config",
			zap.String("path", configPath),
			zap.Int("servers_migrated", result.ServersChanged),
		)
	}

	return result, nil
}

// RollbackMigration rolls back to a backup file
func RollbackMigration(configPath, backupPath string, logger *zap.Logger) error {
	if backupPath == "" {
		return fmt.Errorf("no backup path provided")
	}

	// Read backup
	data, err := os.ReadFile(backupPath)
	if err != nil {
		return fmt.Errorf("failed to read backup: %w", err)
	}

	// Write to config path
	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to restore backup: %w", err)
	}

	if logger != nil {
		logger.Info("Rolled back to backup",
			zap.String("config_path", configPath),
			zap.String("backup_path", backupPath),
		)
	}

	return nil
}

// CleanOldBackups removes backup files older than specified duration
func CleanOldBackups(configDir string, maxAge time.Duration, logger *zap.Logger) error {
	entries, err := os.ReadDir(configDir)
	if err != nil {
		return fmt.Errorf("failed to read config directory: %w", err)
	}

	now := time.Now()
	removed := 0

	for _, entry := range entries {
		// Look for backup files (*.backup-*)
		if entry.IsDir() || !isBackupFile(entry.Name()) {
			continue
		}

		fullPath := filepath.Join(configDir, entry.Name())
		info, err := entry.Info()
		if err != nil {
			if logger != nil {
				logger.Warn("Failed to get file info", zap.String("file", fullPath), zap.Error(err))
			}
			continue
		}

		// Check if older than maxAge
		if now.Sub(info.ModTime()) > maxAge {
			if err := os.Remove(fullPath); err != nil {
				if logger != nil {
					logger.Warn("Failed to remove old backup", zap.String("file", fullPath), zap.Error(err))
				}
			} else {
				removed++
				if logger != nil {
					logger.Debug("Removed old backup", zap.String("file", fullPath))
				}
			}
		}
	}

	if logger != nil && removed > 0 {
		logger.Info("Cleaned old backups", zap.Int("removed", removed))
	}

	return nil
}

// isBackupFile checks if a filename is a backup file
func isBackupFile(filename string) bool {
	return strings.Contains(filename, ".backup-")
}

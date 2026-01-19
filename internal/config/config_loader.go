package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/fsnotify/fsnotify"
	"go.uber.org/zap"
)

// Loader manages configuration loading, watching, and atomic updates.
type Loader struct {
	mu             sync.Mutex
	configPath     string
	config         *Config
	watcher        *fsnotify.Watcher
	skipNextReload bool
	onChange       func(*Config) error
	logger         *zap.Logger
	stopChan       chan struct{}
}

// NewLoader creates a new configuration loader with file watching.
func NewLoader(configPath string, logger *zap.Logger) (*Loader, error) {
	// Create file watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create file watcher: %w", err)
	}

	loader := &Loader{
		configPath: configPath,
		watcher:    watcher,
		logger:     logger,
		stopChan:   make(chan struct{}),
	}

	return loader, nil
}

// Load loads the initial configuration from file.
func (l *Loader) Load() (*Config, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	cfg, err := LoadFromFile(l.configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	l.config = cfg
	return cfg, nil
}

// StartWatching starts watching the configuration file for changes.
// The onChange callback is called when the configuration file changes.
func (l *Loader) StartWatching(onChange func(*Config) error) error {
	l.mu.Lock()
	l.onChange = onChange
	l.mu.Unlock()

	// Add the config file to the watcher
	if err := l.watcher.Add(l.configPath); err != nil {
		return fmt.Errorf("failed to watch config file: %w", err)
	}

	// Start watching in background
	go l.watchLoop()

	l.logger.Info("Started watching configuration file",
		zap.String("path", l.configPath))

	return nil
}

// watchLoop runs the file watching loop.
func (l *Loader) watchLoop() {
	for {
		select {
		case event, ok := <-l.watcher.Events:
			if !ok {
				return
			}

			// Handle file modifications
			if event.Op&fsnotify.Write == fsnotify.Write {
				l.handleFileChange()
			}

		case err, ok := <-l.watcher.Errors:
			if !ok {
				return
			}
			l.logger.Error("File watcher error", zap.Error(err))

		case <-l.stopChan:
			return
		}
	}
}

// handleFileChange handles configuration file changes.
func (l *Loader) handleFileChange() {
	l.mu.Lock()

	// Check if we should skip this reload
	if l.skipNextReload {
		l.logger.Debug("Skipping file reload (programmatic change)")
		l.skipNextReload = false
		l.mu.Unlock()
		return
	}

	l.mu.Unlock()

	// Reload configuration
	l.logger.Info("Configuration file changed, reloading...")

	cfg, err := LoadFromFile(l.configPath)
	if err != nil {
		l.logger.Error("Failed to reload configuration",
			zap.String("path", l.configPath),
			zap.Error(err))
		return
	}

	l.mu.Lock()
	oldConfig := l.config
	l.config = cfg
	onChange := l.onChange
	l.mu.Unlock()

	// Call onChange callback if set
	if onChange != nil {
		if err := onChange(cfg); err != nil {
			l.logger.Error("Failed to apply configuration changes",
				zap.Error(err))

			// Rollback to old config
			l.mu.Lock()
			l.config = oldConfig
			l.mu.Unlock()
			return
		}
	}

	l.logger.Info("Configuration reloaded successfully")
}

// UpdateConfigAtomic performs an atomic configuration update.
// The updateFn receives the current config and should return the modified config.
// Uses temp file + atomic rename pattern to ensure atomicity.
func (l *Loader) UpdateConfigAtomic(updateFn func(*Config) (*Config, error)) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Create a deep copy of current config to avoid in-place modification
	configCopy, err := l.copyConfig(l.config)
	if err != nil {
		return fmt.Errorf("failed to copy config: %w", err)
	}

	// Call update function with the copy
	newConfig, err := updateFn(configCopy)
	if err != nil {
		return fmt.Errorf("update function failed: %w", err)
	}

	// Validate new config
	if err := newConfig.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Write to temporary file
	tempPath := l.configPath + ".tmp"
	if err := l.writeConfigToFile(newConfig, tempPath); err != nil {
		return fmt.Errorf("failed to write temp config: %w", err)
	}

	// Set flag to skip next reload (this is our own change)
	l.skipNextReload = true

	// Atomic rename
	if err := os.Rename(tempPath, l.configPath); err != nil {
		l.skipNextReload = false
		os.Remove(tempPath) // Clean up temp file
		return fmt.Errorf("failed to rename config file: %w", err)
	}

	// Update in-memory config
	l.config = newConfig

	l.logger.Info("Configuration updated atomically",
		zap.String("path", l.configPath))

	return nil
}

// copyConfig creates a deep copy of the configuration using JSON marshal/unmarshal.
func (l *Loader) copyConfig(cfg *Config) (*Config, error) {
	data, err := json.Marshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	var copy Config
	if err := json.Unmarshal(data, &copy); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &copy, nil
}

// ShouldSkipReload checks if the next config file change should be skipped.
// Returns true if the change was programmatic (e.g., auto-disable) and resets the flag.
// This is thread-safe and should be called by external config watchers.
func (l *Loader) ShouldSkipReload() bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.skipNextReload {
		l.skipNextReload = false
		return true
	}
	return false
}

// writeConfigToFile writes configuration to a file.
func (l *Loader) writeConfigToFile(cfg *Config, path string) error {
	// Marshal config to JSON with proper formatting
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write file with proper permissions
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// GetConfig returns the current configuration (thread-safe).
func (l *Loader) GetConfig() *Config {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.config
}

// Stop stops the file watcher and cleanup resources.
func (l *Loader) Stop() error {
	// Signal watch loop to stop
	close(l.stopChan)

	// Close watcher
	if err := l.watcher.Close(); err != nil {
		return fmt.Errorf("failed to close watcher: %w", err)
	}

	l.logger.Info("Stopped configuration file watcher")
	return nil
}

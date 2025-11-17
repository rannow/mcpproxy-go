package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestNewLoader(t *testing.T) {
	logger := zap.NewNop()
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")

	// Create a test config file
	cfg := DefaultConfig()
	data, err := json.MarshalIndent(cfg, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(configPath, data, 0644))

	loader, err := NewLoader(configPath, logger)
	require.NoError(t, err)
	assert.NotNil(t, loader)
	assert.Equal(t, configPath, loader.configPath)
	assert.NotNil(t, loader.watcher)
	assert.NotNil(t, loader.logger)

	// Clean up
	err = loader.Stop()
	assert.NoError(t, err)
}

func TestLoader_Load(t *testing.T) {
	logger := zap.NewNop()
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")

	// Create a test config file
	testConfig := DefaultConfig()
	testConfig.Listen = ":9999"
	data, err := json.MarshalIndent(testConfig, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(configPath, data, 0644))

	loader, err := NewLoader(configPath, logger)
	require.NoError(t, err)
	defer loader.Stop()

	// Load config
	cfg, err := loader.Load()
	require.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.Equal(t, ":9999", cfg.Listen)
}

func TestLoader_GetConfig(t *testing.T) {
	logger := zap.NewNop()
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")

	// Create a test config file
	cfg := DefaultConfig()
	data, err := json.MarshalIndent(cfg, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(configPath, data, 0644))

	loader, err := NewLoader(configPath, logger)
	require.NoError(t, err)
	defer loader.Stop()

	// Load config
	_, err = loader.Load()
	require.NoError(t, err)

	// Get config (thread-safe)
	retrievedCfg := loader.GetConfig()
	assert.NotNil(t, retrievedCfg)
	assert.Equal(t, cfg.Listen, retrievedCfg.Listen)
}

func TestLoader_UpdateConfigAtomic(t *testing.T) {
	logger := zap.NewNop()
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")

	// Create initial config
	initialCfg := DefaultConfig()
	initialCfg.Listen = ":8080"
	data, err := json.MarshalIndent(initialCfg, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(configPath, data, 0644))

	loader, err := NewLoader(configPath, logger)
	require.NoError(t, err)
	defer loader.Stop()

	_, err = loader.Load()
	require.NoError(t, err)

	// Update config atomically
	err = loader.UpdateConfigAtomic(func(cfg *Config) (*Config, error) {
		cfg.Listen = ":9090"
		return cfg, nil
	})
	require.NoError(t, err)

	// Verify in-memory config updated
	updatedCfg := loader.GetConfig()
	assert.Equal(t, ":9090", updatedCfg.Listen)

	// Verify file was updated
	fileData, err := os.ReadFile(configPath)
	require.NoError(t, err)

	var fileCfg Config
	err = json.Unmarshal(fileData, &fileCfg)
	require.NoError(t, err)
	assert.Equal(t, ":9090", fileCfg.Listen)

	// Verify no temp file left behind
	tempPath := configPath + ".tmp"
	_, err = os.Stat(tempPath)
	assert.True(t, os.IsNotExist(err), "temp file should be cleaned up")
}

func TestLoader_UpdateConfigAtomic_InvalidConfig(t *testing.T) {
	logger := zap.NewNop()
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")

	// Create initial config
	initialCfg := DefaultConfig()
	initialCfg.Listen = ":8080"
	initialCfg.TopK = 5
	data, err := json.MarshalIndent(initialCfg, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(configPath, data, 0644))

	loader, err := NewLoader(configPath, logger)
	require.NoError(t, err)
	defer loader.Stop()

	_, err = loader.Load()
	require.NoError(t, err)

	// Try to update with config that would fail - set invalid CallToolTimeout
	err = loader.UpdateConfigAtomic(func(cfg *Config) (*Config, error) {
		// Create invalid config with negative TopK which becomes 0 and is invalid
		cfg.TopK = -100
		cfg.CallToolTimeout = Duration(-1 * time.Second) // Negative timeout
		return cfg, nil
	})

	// Note: Validate() sets defaults rather than strictly validating
	// So this test just verifies the update mechanism works
	// The validation will set TopK to 5 and CallToolTimeout to 2 minutes
	require.NoError(t, err)

	// Verify config was "fixed" by Validate
	currentCfg := loader.GetConfig()
	assert.Equal(t, 5, currentCfg.TopK, "TopK should be set to default")
}

func TestLoader_UpdateConfigAtomic_UpdateFunctionError(t *testing.T) {
	logger := zap.NewNop()
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")

	// Create initial config
	cfg := DefaultConfig()
	data, err := json.MarshalIndent(cfg, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(configPath, data, 0644))

	loader, err := NewLoader(configPath, logger)
	require.NoError(t, err)
	defer loader.Stop()

	_, err = loader.Load()
	require.NoError(t, err)

	// Update function returns error
	testErr := assert.AnError
	err = loader.UpdateConfigAtomic(func(cfg *Config) (*Config, error) {
		return nil, testErr
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "update function failed")
}

func TestLoader_UpdateConfigAtomic_Rollback(t *testing.T) {
	logger := zap.NewNop()
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")

	// Create initial config
	initialCfg := DefaultConfig()
	initialCfg.Listen = ":8080"
	data, err := json.MarshalIndent(initialCfg, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(configPath, data, 0644))

	loader, err := NewLoader(configPath, logger)
	require.NoError(t, err)
	defer loader.Stop()

	_, err = loader.Load()
	require.NoError(t, err)

	// Make directory read-only to force write failure
	tempDir2 := t.TempDir()
	err = os.Chmod(tempDir2, 0444)
	require.NoError(t, err)
	defer os.Chmod(tempDir2, 0755) // Restore for cleanup

	readOnlyPath := filepath.Join(tempDir2, "config.json")
	loader2, err := NewLoader(readOnlyPath, logger)
	require.NoError(t, err)
	defer loader2.Stop()

	loader2.config = initialCfg

	// Try to update - should fail on write
	err = loader2.UpdateConfigAtomic(func(cfg *Config) (*Config, error) {
		cfg.Listen = ":9090"
		return cfg, nil
	})
	assert.Error(t, err)

	// Verify original config unchanged
	assert.Equal(t, ":8080", loader2.GetConfig().Listen)
}

func TestLoader_ConcurrentUpdates(t *testing.T) {
	logger := zap.NewNop()
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")

	// Create initial config
	cfg := DefaultConfig()
	cfg.Listen = ":8080"
	data, err := json.MarshalIndent(cfg, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(configPath, data, 0644))

	loader, err := NewLoader(configPath, logger)
	require.NoError(t, err)
	defer loader.Stop()

	_, err = loader.Load()
	require.NoError(t, err)

	// Concurrent updates
	var wg sync.WaitGroup
	ports := []string{":9000", ":9001", ":9002", ":9003", ":9004"}

	for _, port := range ports {
		wg.Add(1)
		go func(p string) {
			defer wg.Done()
			err := loader.UpdateConfigAtomic(func(cfg *Config) (*Config, error) {
				cfg.Listen = p
				return cfg, nil
			})
			assert.NoError(t, err)
		}(port)
	}

	// Wait for all updates
	done := make(chan bool)
	go func() {
		wg.Wait()
		done <- true
	}()

	select {
	case <-done:
		// Success - no race conditions
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for concurrent updates")
	}

	// Verify config was updated (should be one of the ports)
	finalCfg := loader.GetConfig()
	found := false
	for _, port := range ports {
		if finalCfg.Listen == port {
			found = true
			break
		}
	}
	assert.True(t, found, "final config should have one of the updated ports")
}

func TestLoader_FileWatching(t *testing.T) {
	logger := zap.NewNop()
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")

	// Create initial config
	initialCfg := DefaultConfig()
	initialCfg.Listen = ":8080"
	data, err := json.MarshalIndent(initialCfg, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(configPath, data, 0644))

	loader, err := NewLoader(configPath, logger)
	require.NoError(t, err)
	defer loader.Stop()

	_, err = loader.Load()
	require.NoError(t, err)

	// Track onChange calls
	changeCount := 0
	var mu sync.Mutex
	onChange := func(cfg *Config) error {
		mu.Lock()
		changeCount++
		mu.Unlock()
		return nil
	}

	// Start watching
	err = loader.StartWatching(onChange)
	require.NoError(t, err)

	// Modify config file externally
	modifiedCfg := DefaultConfig()
	modifiedCfg.Listen = ":9999"
	data, err = json.MarshalIndent(modifiedCfg, "", "  ")
	require.NoError(t, err)
	err = os.WriteFile(configPath, data, 0644)
	require.NoError(t, err)

	// Wait for file watcher to trigger
	time.Sleep(500 * time.Millisecond)

	// Verify onChange was called
	mu.Lock()
	count := changeCount
	mu.Unlock()
	assert.Greater(t, count, 0, "onChange should have been called")

	// Verify config was reloaded
	reloadedCfg := loader.GetConfig()
	assert.Equal(t, ":9999", reloadedCfg.Listen)
}

func TestLoader_SkipNextReload(t *testing.T) {
	logger := zap.NewNop()
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")

	// Create initial config
	initialCfg := DefaultConfig()
	initialCfg.Listen = ":8080"
	data, err := json.MarshalIndent(initialCfg, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(configPath, data, 0644))

	loader, err := NewLoader(configPath, logger)
	require.NoError(t, err)
	defer loader.Stop()

	_, err = loader.Load()
	require.NoError(t, err)

	// Track onChange calls
	changeCount := 0
	var mu sync.Mutex
	onChange := func(cfg *Config) error {
		mu.Lock()
		changeCount++
		mu.Unlock()
		return nil
	}

	// Start watching
	err = loader.StartWatching(onChange)
	require.NoError(t, err)

	// Wait for watcher to be ready
	time.Sleep(100 * time.Millisecond)

	// Update atomically (should skip reload)
	err = loader.UpdateConfigAtomic(func(cfg *Config) (*Config, error) {
		cfg.Listen = ":9090"
		return cfg, nil
	})
	require.NoError(t, err)

	// Wait to ensure watcher would have triggered if it were going to
	time.Sleep(500 * time.Millisecond)

	// Verify onChange was NOT called (programmatic change skipped)
	mu.Lock()
	count := changeCount
	mu.Unlock()
	assert.Equal(t, 0, count, "onChange should not be called for programmatic changes")

	// Verify config was updated
	assert.Equal(t, ":9090", loader.GetConfig().Listen)
}

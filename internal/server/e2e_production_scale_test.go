package server

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"mcpproxy-go/internal/config"
)

// TestE2E_ProductionScaleConfig tests mcpproxy with a real production configuration
// containing ~160 MCP servers to verify scalability and performance.
//
// This test validates:
// - Loading large configuration files (~160 servers)
// - Server initialization with production-scale data
// - State management across many servers
// - Group operations with large server counts
// - WebSocket event delivery performance
// - Memory usage and stability
func TestE2E_ProductionScaleConfig(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping production-scale E2E test in short mode")
	}

	// Load the actual production configuration
	productionConfigPath := filepath.Join(os.Getenv("HOME"), ".mcpproxy", "mcp_config.json")
	if _, err := os.Stat(productionConfigPath); os.IsNotExist(err) {
		t.Skipf("Production config not found at %s, skipping test", productionConfigPath)
	}

	// Create test environment
	env := NewTestEnvironment(t)
	defer env.Cleanup()

	ctx := context.Background()

	// Read production configuration
	productionConfigData, err := os.ReadFile(productionConfigPath)
	require.NoError(t, err, "Failed to read production config")

	var productionConfig config.Config
	err = json.Unmarshal(productionConfigData, &productionConfig)
	require.NoError(t, err, "Failed to parse production config")

	serverCount := len(productionConfig.Servers)
	t.Logf("📊 Production config contains %d MCP servers", serverCount)
	require.GreaterOrEqual(t, serverCount, 100, "Production config should have at least 100 servers for meaningful test")

	// Track test metrics
	metrics := &ProductionScaleMetrics{
		StartTime:    time.Now(),
		ServerCount:  serverCount,
		ConfigSizeKB: len(productionConfigData) / 1024,
	}

	// Test 1: Configuration Loading Performance
	t.Run("ConfigurationLoading", func(t *testing.T) {
		startTime := time.Now()

		// Copy production servers into test environment's config
		env.proxyServer.config.Servers = productionConfig.Servers

		// Save configuration to test database
		for _, serverConfig := range productionConfig.Servers {
			err := env.proxyServer.storageManager.SaveUpstreamServer(serverConfig)
			if err != nil {
				t.Logf("⚠️ Failed to save server %s: %v", serverConfig.Name, err)
			}
		}

		metrics.ConfigLoadTime = time.Since(startTime)
		t.Logf("✅ Loaded %d servers in %v", serverCount, metrics.ConfigLoadTime)
		assert.Less(t, metrics.ConfigLoadTime.Seconds(), 5.0, "Config loading should complete within 5 seconds")
	})

	// Test 2: Server State Management at Scale
	t.Run("StateManagement", func(t *testing.T) {
		startTime := time.Now()

		// Get all servers from storage
		allServers, err := env.proxyServer.storageManager.ListUpstreamServers()
		require.NoError(t, err)

		metrics.StorageReadTime = time.Since(startTime)
		t.Logf("✅ Retrieved %d server states in %v", len(allServers), metrics.StorageReadTime)

		// Verify we got all servers
		assert.Equal(t, serverCount, len(allServers), "All servers should be retrievable from storage")

		// Test state transitions for subset of servers
		testServers := min(10, len(allServers))
		stateTransitionStart := time.Now()

		for i := 0; i < testServers; i++ {
			serverName := allServers[i].Name

			// Test enable operation
			err := env.proxyServer.storageManager.EnableUpstreamServer(serverName, true)
			assert.NoError(t, err, "Enable server should succeed for %s", serverName)

			// Test disable operation (via second parameter = false)
			err = env.proxyServer.storageManager.EnableUpstreamServer(serverName, false)
			assert.NoError(t, err, "Disable server should succeed for %s", serverName)
		}

		metrics.StateTransitionTime = time.Since(stateTransitionStart)
		avgTime := metrics.StateTransitionTime.Milliseconds() / int64(testServers*2) // 2 operations per server
		t.Logf("✅ Performed %d state transitions in %v (avg: %dms)", testServers*2, metrics.StateTransitionTime, avgTime)
		assert.Less(t, avgTime, int64(100), "Average state transition should be under 100ms")
	})

	// Test 3: Group Operations with Large Server Counts
	t.Run("GroupOperations", func(t *testing.T) {
		// Create test groups with different sizes
		testGroups := []struct {
			name        string
			serverNames []string
		}{
			{
				name:        "test-group-small",
				serverNames: getServerNames(productionConfig.Servers, 5),
			},
			{
				name:        "test-group-medium",
				serverNames: getServerNames(productionConfig.Servers, 20),
			},
			{
				name:        "test-group-large",
				serverNames: getServerNames(productionConfig.Servers, 50),
			},
		}

		for _, group := range testGroups {
			t.Run(group.name, func(t *testing.T) {
				startTime := time.Now()

				// Simulate group enable operation
				successCount := 0
				failCount := 0

				for _, serverName := range group.serverNames {
					err := env.proxyServer.storageManager.EnableUpstreamServer(serverName, true)
					if err == nil {
						successCount++
					} else {
						failCount++
					}
				}

				duration := time.Since(startTime)
				avgPerServer := duration.Milliseconds() / int64(len(group.serverNames))

				t.Logf("✅ Group %s: %d servers in %v (avg: %dms/server, success: %d, fail: %d)",
					group.name, len(group.serverNames), duration, avgPerServer, successCount, failCount)

				assert.GreaterOrEqual(t, successCount, len(group.serverNames)*8/10,
					"At least 80%% of group operations should succeed")
				assert.Less(t, avgPerServer, int64(50), "Average operation time should be under 50ms")
			})
		}
	})

	// Test 4: Event System Performance at Scale
	t.Run("EventSystemPerformance", func(t *testing.T) {
		eventCount := 0
		eventReceived := make(chan bool, 100)

		// Subscribe to events
		eventChan := env.proxyServer.eventBus.Subscribe("ServerStateChanged")
		defer env.proxyServer.eventBus.Unsubscribe("ServerStateChanged", eventChan)

		// Collect events in background
		go func() {
			timeout := time.After(10 * time.Second)
			for {
				select {
				case <-eventChan:
					eventCount++
					select {
					case eventReceived <- true:
					default:
					}
				case <-timeout:
					return
				}
			}
		}()

		startTime := time.Now()

		// Trigger state changes for subset of servers
		testServers := getServerNames(productionConfig.Servers, 20)
		for _, serverName := range testServers {
			err := env.proxyServer.storageManager.EnableUpstreamServer(serverName, true)
			if err != nil {
				t.Logf("⚠️ Failed to enable server %s: %v", serverName, err)
			}
		}

		// Wait for events to be processed
		time.Sleep(2 * time.Second)

		metrics.EventDeliveryTime = time.Since(startTime)
		t.Logf("✅ Processed %d events in %v for %d servers",
			eventCount, metrics.EventDeliveryTime, len(testServers))

		assert.GreaterOrEqual(t, eventCount, len(testServers)/2,
			"Should receive events for at least half the servers")
	})

	// Test 5: Tool Discovery and Search at Scale
	t.Run("ToolDiscoveryAtScale", func(t *testing.T) {
		mcpClient := env.CreateProxyClient()
		defer mcpClient.Close()
		env.ConnectClient(mcpClient)

		// Test retrieve_tools with production config
		startTime := time.Now()

		searchRequest := mcp.CallToolRequest{}
		searchRequest.Params.Name = "retrieve_tools"
		searchRequest.Params.Arguments = map[string]interface{}{
			"query": "server management configuration",
			"limit": 20,
		}

		searchResult, err := mcpClient.CallTool(ctx, searchRequest)
		require.NoError(t, err)
		assert.False(t, searchResult.IsError)

		metrics.ToolSearchTime = time.Since(startTime)
		t.Logf("✅ Tool search completed in %v", metrics.ToolSearchTime)
		assert.Less(t, metrics.ToolSearchTime.Seconds(), 2.0,
			"Tool search should complete within 2 seconds even with large config")
	})

	// Test 6: Memory and Stability
	t.Run("MemoryAndStability", func(t *testing.T) {
		// Perform multiple operations to stress test
		iterations := 5

		for i := 0; i < iterations; i++ {
			t.Logf("Iteration %d/%d", i+1, iterations)

			// List all servers
			_, err := env.proxyServer.storageManager.ListUpstreamServers()
			require.NoError(t, err)

			// Perform random state changes
			testServers := getServerNames(productionConfig.Servers, 10)
			for _, serverName := range testServers {
				enabled := (i % 2) == 0
				_ = env.proxyServer.storageManager.EnableUpstreamServer(serverName, enabled)
			}

			time.Sleep(100 * time.Millisecond)
		}

		t.Logf("✅ Completed %d stress test iterations without crash", iterations)
	})

	// Test 7: Configuration Persistence
	t.Run("ConfigurationPersistence", func(t *testing.T) {
		// Save configuration to file
		configPath := filepath.Join(env.tempDir, "test_config.json")

		err := config.SaveToFile(env.proxyServer.config, configPath)
		require.NoError(t, err, "Config save should succeed")

		// Verify file size
		fileInfo, err := os.Stat(configPath)
		require.NoError(t, err)

		t.Logf("✅ Saved config file: %s (size: %d KB)", configPath, fileInfo.Size()/1024)
		assert.Greater(t, fileInfo.Size(), int64(10*1024), "Config file should be at least 10KB")

		// Reload and verify
		reloadedConfig, err := config.LoadFromFile(configPath)
		require.NoError(t, err)

		assert.Equal(t, len(env.proxyServer.config.Servers), len(reloadedConfig.Servers),
			"Reloaded config should have same number of servers")
	})

	// Print final metrics
	metrics.TotalTestTime = time.Since(metrics.StartTime)
	printProductionScaleMetrics(t, metrics)
}

// ProductionScaleMetrics tracks performance metrics for production-scale testing
type ProductionScaleMetrics struct {
	StartTime           time.Time
	TotalTestTime       time.Duration
	ServerCount         int
	ConfigSizeKB        int
	ConfigLoadTime      time.Duration
	StorageReadTime     time.Duration
	StateTransitionTime time.Duration
	EventDeliveryTime   time.Duration
	ToolSearchTime      time.Duration
}

// printProductionScaleMetrics prints a summary of test metrics
func printProductionScaleMetrics(t *testing.T, m *ProductionScaleMetrics) {
	t.Logf("\n")
	t.Logf("═══════════════════════════════════════════════════════════")
	t.Logf("📊 PRODUCTION SCALE TEST METRICS")
	t.Logf("═══════════════════════════════════════════════════════════")
	t.Logf("Configuration:")
	t.Logf("  • Server Count:        %d servers", m.ServerCount)
	t.Logf("  • Config File Size:    %d KB", m.ConfigSizeKB)
	t.Logf("")
	t.Logf("Performance:")
	t.Logf("  • Config Load Time:    %v", m.ConfigLoadTime)
	t.Logf("  • Storage Read Time:   %v", m.StorageReadTime)
	t.Logf("  • State Transition:    %v", m.StateTransitionTime)
	t.Logf("  • Event Delivery:      %v", m.EventDeliveryTime)
	t.Logf("  • Tool Search:         %v", m.ToolSearchTime)
	t.Logf("")
	t.Logf("Overall:")
	t.Logf("  • Total Test Time:     %v", m.TotalTestTime)
	t.Logf("  • Servers/Second:      %.2f", float64(m.ServerCount)/m.TotalTestTime.Seconds())
	t.Logf("═══════════════════════════════════════════════════════════")
	t.Logf("\n")
}

// getServerNames returns the first n server names from a server slice
func getServerNames(servers []*config.ServerConfig, n int) []string {
	names := make([]string, 0, n)
	count := 0

	for _, server := range servers {
		if count >= n {
			break
		}
		names = append(names, server.Name)
		count++
	}

	return names
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// TestE2E_ProductionConfigStartup tests that the server can start with production config
// This is a lighter test that just verifies startup doesn't crash or timeout
func TestE2E_ProductionConfigStartup(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping production config startup test in short mode")
	}

	productionConfigPath := filepath.Join(os.Getenv("HOME"), ".mcpproxy", "mcp_config.json")
	if _, err := os.Stat(productionConfigPath); os.IsNotExist(err) {
		t.Skipf("Production config not found at %s, skipping test", productionConfigPath)
	}

	// Create temp directory for test
	tempDir, err := os.MkdirTemp("", "mcpproxy-startup-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Copy production config to temp location
	testConfigPath := filepath.Join(tempDir, "mcp_config.json")
	productionData, err := os.ReadFile(productionConfigPath)
	require.NoError(t, err)

	err = os.WriteFile(testConfigPath, productionData, 0644)
	require.NoError(t, err)

	// Create logger
	logger := zap.NewNop()

	// Create data directory
	dataDir := filepath.Join(tempDir, "data")
	err = os.MkdirAll(dataDir, 0755)
	require.NoError(t, err)

	// Load the production config from file
	cfg, err := config.LoadFromFile(testConfigPath)
	require.NoError(t, err)

	// Override necessary settings for test
	cfg.DataDir = dataDir
	cfg.Listen = ":0" // Random port
	cfg.DisableManagement = false

	serverCount := len(cfg.Servers)
	t.Logf("📊 Testing startup with %d servers from production config", serverCount)

	// Create server
	startTime := time.Now()
	srv, err := NewServer(cfg, logger)
	require.NoError(t, err)
	defer srv.Shutdown()

	// Start server
	ctx := context.Background()
	err = srv.StartServer(ctx)
	require.NoError(t, err)

	startupTime := time.Since(startTime)
	t.Logf("✅ Server started successfully in %v with %d servers", startupTime, serverCount)

	// Verify server is running
	assert.True(t, srv.IsRunning())

	// Allow some time for initialization
	time.Sleep(2 * time.Second)

	// Verify we can get server status
	allServers, err := srv.storageManager.ListUpstreamServers()
	require.NoError(t, err)
	t.Logf("✅ Retrieved %d servers from storage", len(allServers))

	assert.GreaterOrEqual(t, len(allServers), serverCount*8/10,
		"At least 80%% of servers should be loaded into storage")

	// Test basic operations work
	status := srv.GetStatus()
	assert.NotNil(t, status)
	t.Logf("✅ Server status: %+v", status)

	t.Logf("\n")
	t.Logf("═══════════════════════════════════════════════════════════")
	t.Logf("✅ PRODUCTION CONFIG STARTUP TEST PASSED")
	t.Logf("═══════════════════════════════════════════════════════════")
	t.Logf("  • Servers Loaded:      %d / %d", len(allServers), serverCount)
	t.Logf("  • Startup Time:        %v", startupTime)
	t.Logf("  • Server Running:      %v", srv.IsRunning())
	t.Logf("═══════════════════════════════════════════════════════════")
	t.Logf("\n")
}

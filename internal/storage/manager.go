package storage

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"mcpproxy-go/internal/config"
	"mcpproxy-go/internal/events"

	"go.etcd.io/bbolt"
	"go.uber.org/zap"
)

// Manager provides a unified interface for storage operations
type Manager struct {
	db           *BoltDB
	configLoader *config.Loader
	eventBus     *events.Bus
	mu           sync.RWMutex
	logger       *zap.SugaredLogger
}

// NewManager creates a new storage manager
func NewManager(dataDir string, logger *zap.SugaredLogger) (*Manager, error) {
	db, err := NewBoltDB(dataDir, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create bolt database: %w", err)
	}

	return &Manager{
		db:     db,
		logger: logger,
	}, nil
}

// Close closes the storage manager
func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.db != nil {
		return m.db.Close()
	}
	return nil
}

// GetDB returns the underlying BBolt database for direct access
func (m *Manager) GetDB() *bbolt.DB {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.db != nil {
		return m.db.db
	}
	return nil
}

// GetBoltDB returns the wrapped BoltDB instance for higher-level operations
func (m *Manager) GetBoltDB() *BoltDB {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.db
}

// SetConfigLoader sets the config loader for two-phase commit operations
func (m *Manager) SetConfigLoader(loader *config.Loader) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.configLoader = loader
}

// SetEventBus sets the event bus for publishing state change events
func (m *Manager) SetEventBus(eventBus *events.Bus) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.eventBus = eventBus
}

// Upstream operations

// SaveUpstreamServer saves an upstream server configuration
func (m *Manager) SaveUpstreamServer(serverConfig *config.ServerConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	record := &UpstreamRecord{
		ID:                       serverConfig.Name, // Use name as ID for simplicity
		Name:                     serverConfig.Name,
		Description:              serverConfig.Description,
		URL:                      serverConfig.URL,
		Protocol:                 serverConfig.Protocol,
		Command:                  serverConfig.Command,
		Args:                     serverConfig.Args,
		WorkingDir:               serverConfig.WorkingDir,
		Env:                      serverConfig.Env,
		Headers:                  serverConfig.Headers,
		OAuth:                    serverConfig.OAuth,
		RepositoryURL:            serverConfig.RepositoryURL,
		Created:                  serverConfig.Created,
		Updated:                  time.Now(),
		Isolation:                serverConfig.Isolation,
		GroupID:                  serverConfig.GroupID,
		GroupName:                serverConfig.GroupName,
		EverConnected:            serverConfig.EverConnected,
		LastSuccessfulConnection: serverConfig.LastSuccessfulConnection,
		ToolCount:                serverConfig.ToolCount,
		HealthCheck:              serverConfig.HealthCheck,
		AutoDisableThreshold:     serverConfig.AutoDisableThreshold,
		ServerState:              serverConfig.StartupMode,       // Map config.StartupMode → storage.ServerState
		AutoDisableReason:        serverConfig.AutoDisableReason, // Save auto-disable reason
	}

	return m.db.SaveUpstream(record)
}

// GetUpstreamServer retrieves an upstream server by name
func (m *Manager) GetUpstreamServer(name string) (*config.ServerConfig, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	record, err := m.db.GetUpstream(name)
	if err != nil {
		return nil, err
	}

	// Determine startup mode with config file priority
	// PRIORITY: Config file has priority over database, EXCEPT for auto_disabled state
	startupMode := record.ServerState
	dbServerState := record.ServerState

	if m.configLoader != nil {
		if cfg := m.configLoader.GetConfig(); cfg != nil {
			for _, s := range cfg.Servers {
				if s.Name == name {
					configStartupMode := s.StartupMode
					if configStartupMode == "" {
						configStartupMode = "active" // Default if not specified in config
					}

					// Special case: If database has auto_disabled, keep it (runtime protection)
					// UNLESS config explicitly sets auto_disabled (user confirmed)
					if dbServerState == "auto_disabled" && configStartupMode != "auto_disabled" {
						// Keep auto_disabled from database
						m.logger.Debug("GetUpstreamServer: keeping auto_disabled from database",
							zap.String("server", name),
							zap.String("config_startup_mode", configStartupMode))
					} else {
						// Config file takes priority
						startupMode = configStartupMode
					}
					break
				}
			}
		}
	}

	// Default to "active" if still empty
	if startupMode == "" {
		startupMode = "active"
	}

	return &config.ServerConfig{
		Name:                     record.Name,
		Description:              record.Description,
		URL:                      record.URL,
		Protocol:                 record.Protocol,
		Command:                  record.Command,
		Args:                     record.Args,
		WorkingDir:               record.WorkingDir,
		Env:                      record.Env,
		Headers:                  record.Headers,
		OAuth:                    record.OAuth,
		RepositoryURL:            record.RepositoryURL,
		Created:                  record.Created,
		Updated:                  record.Updated,
		Isolation:                record.Isolation,
		GroupID:                  record.GroupID,
		GroupName:                record.GroupName,
		EverConnected:            record.EverConnected,
		LastSuccessfulConnection: record.LastSuccessfulConnection,
		ToolCount:                record.ToolCount,
		HealthCheck:              record.HealthCheck,
		AutoDisableThreshold:     record.AutoDisableThreshold,
		StartupMode:              startupMode,             // Use config-prioritized startup mode
		AutoDisableReason:        record.AutoDisableReason, // Include auto-disable reason
	}, nil
}

// ListUpstreamServers returns all upstream servers
func (m *Manager) ListUpstreamServers() ([]*config.ServerConfig, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	records, err := m.db.ListUpstreams()
	if err != nil {
		return nil, err
	}

	var servers []*config.ServerConfig
	for _, record := range records {
		// Map database server_state to config startup_mode
		// PRIORITY: Config file has priority over database, EXCEPT for auto_disabled state
		// which is a runtime state that should persist across restarts
		startupMode := record.ServerState
		dbServerState := record.ServerState

		m.logger.Debug("ListUpstreamServers processing record",
			zap.String("server", record.Name),
			zap.String("db_server_state", record.ServerState),
			zap.String("initial_startup_mode", startupMode))

		// CONFIG FILE HAS PRIORITY: Always check config file for startup_mode
		// The config file is the source of truth for user-defined states (active, disabled, quarantined, lazy_loading)
		// Only auto_disabled from database should override config, as it's a runtime protection mechanism
		if m.configLoader != nil {
			if cfg := m.configLoader.GetConfig(); cfg != nil {
				for _, s := range cfg.Servers {
					if s.Name == record.Name {
						configStartupMode := s.StartupMode
						if configStartupMode == "" {
							configStartupMode = "active" // Default if not specified in config
						}

						// Special case: If database has auto_disabled, keep it (runtime protection)
						// UNLESS config explicitly sets auto_disabled (user confirmed)
						if dbServerState == "auto_disabled" && configStartupMode != "auto_disabled" {
							m.logger.Info("Server is auto_disabled in database - keeping runtime state",
								zap.String("server", record.Name),
								zap.String("db_server_state", dbServerState),
								zap.String("config_startup_mode", configStartupMode),
								zap.String("result", "keeping auto_disabled"))
							// Keep auto_disabled from database
						} else {
							// Config file takes priority for all other cases
							if startupMode != configStartupMode {
								m.logger.Info("Config file takes priority over database",
									zap.String("server", record.Name),
									zap.String("db_server_state", dbServerState),
									zap.String("config_startup_mode", configStartupMode),
									zap.String("result", configStartupMode))
							}
							startupMode = configStartupMode
						}
						break
					}
				}
			}
		}

		// Final fallback: default to "active" if still empty
		// This handles cases where database has no server_state and configLoader is not available
		if startupMode == "" {
			startupMode = "active"
			m.logger.Debug("Using default startup_mode as fallback",
				zap.String("server", record.Name),
				zap.String("default_startup_mode", "active"),
				zap.String("reason", "database server_state is empty and config fallback unavailable"))
		}

		servers = append(servers, &config.ServerConfig{
			Name:                     record.Name,
			Description:              record.Description,
			URL:                      record.URL,
			Protocol:                 record.Protocol,
			Command:                  record.Command,
			Args:                     record.Args,
			WorkingDir:               record.WorkingDir,
			Env:                      record.Env,
			Headers:                  record.Headers,
			OAuth:                    record.OAuth,
			RepositoryURL:            record.RepositoryURL,
			Created:                  record.Created,
			Updated:                  record.Updated,
			Isolation:                record.Isolation,
			GroupID:                  record.GroupID,
			GroupName:                record.GroupName,
			EverConnected:            record.EverConnected,
			LastSuccessfulConnection: record.LastSuccessfulConnection,
			ToolCount:                record.ToolCount,
			HealthCheck:              record.HealthCheck,
			AutoDisableThreshold:     record.AutoDisableThreshold,
			StartupMode:              startupMode, // Use fallback value if database was empty
			AutoDisableReason:        record.AutoDisableReason,
		})
	}

	return servers, nil
}

// ListQuarantinedUpstreamServers returns all quarantined upstream servers
func (m *Manager) ListQuarantinedUpstreamServers() ([]*config.ServerConfig, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	m.logger.Debug("ListQuarantinedUpstreamServers called")

	records, err := m.db.ListUpstreams()
	if err != nil {
		m.logger.Errorw("Failed to list upstreams for quarantine filtering",
			"error", err)
		return nil, err
	}

	m.logger.Debugw("Retrieved all upstream records for quarantine filtering",
		"total_records", len(records))

	var quarantinedServers []*config.ServerConfig
	for _, record := range records {
		m.logger.Debugw("Checking server quarantine status",
			"server", record.Name,
			"server_state", record.ServerState)

		if record.ServerState == "quarantined" {
			quarantinedServers = append(quarantinedServers, &config.ServerConfig{
				Name:                     record.Name,
				Description:              record.Description,
				URL:                      record.URL,
				Protocol:                 record.Protocol,
				Command:                  record.Command,
				Args:                     record.Args,
				WorkingDir:               record.WorkingDir,
				Env:                      record.Env,
				Headers:                  record.Headers,
				OAuth:                    record.OAuth,
				RepositoryURL:            record.RepositoryURL,
				Created:                  record.Created,
				Updated:                  record.Updated,
				Isolation:                record.Isolation,
				GroupID:                  record.GroupID,
				GroupName:                record.GroupName,
				EverConnected:            record.EverConnected,
				LastSuccessfulConnection: record.LastSuccessfulConnection,
				ToolCount:                record.ToolCount,
				HealthCheck:              record.HealthCheck,
				StartupMode:              record.ServerState,
				AutoDisableReason:        record.AutoDisableReason,
			})

			m.logger.Debugw("Added server to quarantined list",
				"server", record.Name,
				"total_quarantined_so_far", len(quarantinedServers))
		}
	}

	m.logger.Debugw("ListQuarantinedUpstreamServers completed",
		"total_quarantined", len(quarantinedServers))

	return quarantinedServers, nil
}

// ListQuarantinedTools returns tools from quarantined servers with full descriptions for security analysis
func (m *Manager) ListQuarantinedTools(serverName string) ([]map[string]interface{}, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Check if server is quarantined
	server, err := m.GetUpstreamServer(serverName)
	if err != nil {
		return nil, err
	}

	if !server.IsQuarantined() {
		return nil, fmt.Errorf("server '%s' is not quarantined", serverName)
	}

	// NOTE: This returns placeholder data. Actual implementation would need to:
	// 1. Connect to the quarantined upstream server via UpstreamManager
	// 2. Retrieve tool descriptions with full schemas for LLM security analysis
	// 3. Include input schemas and security analysis prompts
	// Feature Backlog: Implement GetQuarantinedServerTools with actual server connection
	tools := []map[string]interface{}{
		{
			"message":        fmt.Sprintf("Server '%s' is quarantined. The actual tool descriptions should be retrieved from the upstream manager for security analysis.", serverName),
			"server":         serverName,
			"status":         "quarantined",
			"implementation": "PLACEHOLDER",
			"next_steps":     "The upstream manager should be used to connect to this server and retrieve actual tool descriptions with full schemas for LLM security analysis",
			"security_note":  "Real implementation needs to: 1) Connect to quarantined server, 2) Retrieve all tools with descriptions, 3) Include input schemas, 4) Add security analysis prompts, 5) Return quoted tool descriptions for LLM inspection",
		},
	}

	return tools, nil
}

// DeleteUpstreamServer deletes an upstream server
func (m *Manager) DeleteUpstreamServer(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.db.DeleteUpstream(name)
}

// EnableUpstreamServer enables/disables an upstream server using server_state
// When enabling, sets server_state to "active", when disabling sets to "auto_disabled"
func (m *Manager) EnableUpstreamServer(name string, enabled bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	record, err := m.db.GetUpstream(name)
	if err != nil {
		return err
	}

	// Store old value for rollback
	oldServerState := record.ServerState

	// Set appropriate server_state
	if enabled {
		record.ServerState = "active"
		record.AutoDisableReason = "" // Clear auto-disable reason when enabling
	} else {
		record.ServerState = "auto_disabled"
		// Keep auto-disable reason when disabling (it may have been set by auto-disable logic)
	}
	record.Updated = time.Now()

	// Phase 1: Update database
	if err := m.db.SaveUpstream(record); err != nil {
		return fmt.Errorf("failed to save to database: %w", err)
	}

	// Phase 2: Update config file
	if m.configLoader != nil {
		if err := m.configLoader.UpdateConfigAtomic(func(cfg *config.Config) (*config.Config, error) {
			for i, server := range cfg.Servers {
				if server.Name == name {
					if enabled {
						cfg.Servers[i].StartupMode = "active"
						cfg.Servers[i].AutoDisableReason = "" // Clear auto-disable reason when enabling
					} else {
						cfg.Servers[i].StartupMode = "auto_disabled"
						// Keep auto-disable reason when disabling (it may have been set by auto-disable logic)
					}
					break
				}
			}
			return cfg, nil
		}); err != nil {
			// Rollback database change
			record.ServerState = oldServerState
			if rollbackErr := m.db.SaveUpstream(record); rollbackErr != nil {
				m.logger.Errorw("Failed to rollback database changes",
					"server", name,
					"error", rollbackErr)
			}
			return fmt.Errorf("failed to update config file: %w", err)
		}
	}

	m.logger.Infow("Updated server state via EnableUpstreamServer",
		"server", name,
		"enabled", enabled,
		"server_state", record.ServerState)

	return nil
}

// QuarantineUpstreamServer sets the quarantine status of an upstream server using server_state
func (m *Manager) QuarantineUpstreamServer(name string, quarantined bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.logger.Debugw("QuarantineUpstreamServer called",
		"server", name,
		"quarantined", quarantined)

	record, err := m.db.GetUpstream(name)
	if err != nil {
		m.logger.Errorw("Failed to get upstream record for quarantine operation",
			"server", name,
			"error", err)
		return err
	}

	// Store old value for rollback
	oldServerState := record.ServerState

	m.logger.Debugw("Retrieved upstream record for quarantine",
		"server", name,
		"current_server_state", record.ServerState,
		"new_quarantined", quarantined)

	// Set appropriate server_state
	if quarantined {
		record.ServerState = "quarantined"
	} else {
		// When un-quarantining, set to active
		record.ServerState = "active"
	}
	record.Updated = time.Now()

	// Phase 1: Update database
	if err := m.db.SaveUpstream(record); err != nil {
		m.logger.Errorw("Failed to save quarantine status to database",
			"server", name,
			"quarantined", quarantined,
			"error", err)
		return err
	}

	// Phase 2: Update config file
	if m.configLoader != nil {
		if err := m.configLoader.UpdateConfigAtomic(func(cfg *config.Config) (*config.Config, error) {
			for i, server := range cfg.Servers {
				if server.Name == name {
					if quarantined {
						cfg.Servers[i].StartupMode = "quarantined"
					} else {
						cfg.Servers[i].StartupMode = "active"
					}
					break
				}
			}
			return cfg, nil
		}); err != nil {
			// Rollback database changes
			record.ServerState = oldServerState
			if rollbackErr := m.db.SaveUpstream(record); rollbackErr != nil {
				m.logger.Errorw("Failed to rollback database changes",
					"server", name,
					"error", rollbackErr)
			}
			return fmt.Errorf("failed to update config file: %w", err)
		}
	}

	m.logger.Debugw("Successfully saved quarantine status to database",
		"server", name,
		"server_state", record.ServerState)

	return nil
}

// ClearAutoDisable clears the auto-disable state for a server using server_state
// This method updates both the database and the config file in a two-phase commit
func (m *Manager) ClearAutoDisable(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Phase 1: Update database (can rollback if config fails)
	record, err := m.db.GetUpstream(name)
	if err != nil {
		return fmt.Errorf("failed to get upstream record: %w", err)
	}

	// Store old value for rollback
	oldServerState := record.ServerState

	// Clear auto-disable by setting server_state to active and clearing reason
	record.ServerState = "active"
	record.AutoDisableReason = "" // Clear the auto-disable reason
	record.Updated = time.Now()

	if err := m.db.SaveUpstream(record); err != nil {
		return fmt.Errorf("failed to save to database: %w", err)
	}

	// Phase 2: Update config file
	if m.configLoader != nil {
		if err := m.configLoader.UpdateConfigAtomic(func(cfg *config.Config) (*config.Config, error) {
			// Find server in config
			for i, server := range cfg.Servers {
				if server.Name == name {
					cfg.Servers[i].StartupMode = "active"
					cfg.Servers[i].AutoDisableReason = "" // Clear the reason in config too
					break
				}
			}
			return cfg, nil
		}); err != nil {
			// Rollback database changes
			record.ServerState = oldServerState
			if rollbackErr := m.db.SaveUpstream(record); rollbackErr != nil {
				m.logger.Errorw("Failed to rollback database changes",
					"server", name,
					"error", rollbackErr)
			}
			return fmt.Errorf("failed to update config file: %w", err)
		}
	}

	m.logger.Infow("Cleared auto-disable state",
		"server", name,
		"server_state", "active")

	return nil
}

// UpdateUpstreamServerState updates ONLY the database server_state WITHOUT changing config startup_mode
// IMPORTANT: This is database-only persistence - config startup_mode remains unchanged
// Use case: Lazy loading servers where startup_mode="lazy_loading" but server_state="stopped"
func (m *Manager) UpdateUpstreamServerState(id, serverState string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Only update database (do NOT touch config file)
	record, err := m.db.GetUpstream(id)
	if err != nil {
		return fmt.Errorf("failed to get upstream record: %w", err)
	}

	// Update server_state in database only
	record.ServerState = serverState
	record.Updated = time.Now()

	if err := m.db.SaveUpstream(record); err != nil {
		return fmt.Errorf("failed to save to database: %w", err)
	}

	m.logger.Infow("Updated database server_state (config unchanged)",
		"server_id", id,
		"server_state", serverState)

	return nil
}

// UpdateServerState sets a server to auto_disabled state after threshold failures
// This function is ONLY called after ShouldAutoDisable() threshold check passes
// The threshold (e.g., 5 consecutive failures) is validated by StateManager.ShouldAutoDisable()
// For manual enable/disable, use EnableUpstreamServer instead
func (m *Manager) UpdateServerState(name string, reason string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Phase 1: Update database
	record, err := m.db.GetUpstream(name)
	if err != nil {
		return fmt.Errorf("failed to get upstream record: %w", err)
	}

	// Store old values for rollback
	oldServerState := record.ServerState

	// Always set to auto_disabled state when this function is called
	record.ServerState = "auto_disabled"
	record.AutoDisableReason = reason // Optional - can be empty string
	record.Updated = time.Now()

	if err := m.db.SaveUpstream(record); err != nil {
		return fmt.Errorf("failed to save to database: %w", err)
	}

	// Phase 2: Update config file
	if m.configLoader != nil {
		if err := m.configLoader.UpdateConfigAtomic(func(cfg *config.Config) (*config.Config, error) {
			// Find server in config
			for i, server := range cfg.Servers {
				if server.Name == name {
					// Always set to auto_disabled state (this function is ONLY for auto-disable)
					cfg.Servers[i].StartupMode = "auto_disabled"
					cfg.Servers[i].AutoDisableReason = reason // Optional - can be empty
					break
				}
			}
			return cfg, nil
		}); err != nil {
			// Rollback database changes
			record.ServerState = oldServerState
			if rollbackErr := m.db.SaveUpstream(record); rollbackErr != nil {
				m.logger.Errorw("Failed to rollback database changes",
					"server", name,
					"error", rollbackErr)
			}
			return fmt.Errorf("failed to update config file: %w", err)
		}
	}

	m.logger.Infow("Updated server server state",
		"server", name,
		"server_state", record.ServerState,
		"reason", reason)

	// Publish ServerAutoDisabled event
	if m.eventBus != nil {
		m.eventBus.Publish(events.Event{
			Type:       events.ServerAutoDisabled,
			ServerName: name,
			OldState:   oldServerState,
			NewState:   "auto_disabled",
			Data: map[string]interface{}{
				"reason":    reason,
				"timestamp": time.Now(),
			},
		})
	}

	return nil
}

// StopServer sets the runtime stopped flag for a server
// This only updates in-memory state, not persistent storage
func (m *Manager) StopServer(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Get record to verify it exists
	_, err := m.db.GetUpstream(name)
	if err != nil {
		return fmt.Errorf("server not found: %w", err)
	}

	// Note: This is a placeholder for runtime state management
	// The actual runtime state will be managed by the state machine
	m.logger.Infow("Server stop requested (runtime state)",
		"server", name)

	return nil
}

// StartServer triggers connection attempt for a server
// This only updates in-memory state, not persistent storage
func (m *Manager) StartServer(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Get record to verify it exists and is enabled
	record, err := m.db.GetUpstream(name)
	if err != nil {
		return fmt.Errorf("server not found: %w", err)
	}

	if record.ServerState == "disabled" || record.ServerState == "quarantined" || record.ServerState == "auto_disabled" {
		return fmt.Errorf("server is disabled (mode: %s), cannot start", record.ServerState)
	}

	// Note: This is a placeholder for runtime state management
	// The actual connection will be triggered by the state machine
	m.logger.Infow("Server start requested (runtime state)",
		"server", name)

	return nil
}

// Tool statistics operations

// IncrementToolUsage increments the usage count for a tool
func (m *Manager) IncrementToolUsage(toolName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.logger.Debugf("Incrementing usage for tool: %s", toolName)
	return m.db.IncrementToolStats(toolName)
}

// GetToolUsage retrieves usage statistics for a tool
func (m *Manager) GetToolUsage(toolName string) (*ToolStatRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.db.GetToolStats(toolName)
}

// GetToolStatistics returns aggregated tool statistics
func (m *Manager) GetToolStatistics(topN int) (*config.ToolStats, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	records, err := m.db.ListToolStats()
	if err != nil {
		return nil, err
	}

	// Sort by usage count (descending)
	sort.Slice(records, func(i, j int) bool {
		return records[i].Count > records[j].Count
	})

	// Limit to topN
	if topN > 0 && len(records) > topN {
		records = records[:topN]
	}

	// Convert to config format
	var topTools []config.ToolStatEntry
	for _, record := range records {
		topTools = append(topTools, config.ToolStatEntry{
			ToolName: record.ToolName,
			Count:    record.Count,
		})
	}

	return &config.ToolStats{
		TotalTools: len(records),
		TopTools:   topTools,
	}, nil
}

// Tool hash operations

// SaveToolHash saves a tool hash for change detection
func (m *Manager) SaveToolHash(toolName, hash string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.db.SaveToolHash(toolName, hash)
}

// GetToolHash retrieves a tool hash
func (m *Manager) GetToolHash(toolName string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.db.GetToolHash(toolName)
}

// HasToolChanged checks if a tool has changed based on its hash
func (m *Manager) HasToolChanged(toolName, currentHash string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	storedHash, err := m.db.GetToolHash(toolName)
	if err != nil {
		// If hash doesn't exist, consider it changed (new tool)
		return true, nil
	}

	return storedHash != currentHash, nil
}

// DeleteToolHash deletes a tool hash
func (m *Manager) DeleteToolHash(toolName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.db.DeleteToolHash(toolName)
}

// Maintenance operations

// Backup creates a backup of the database
func (m *Manager) Backup(destPath string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.db.Backup(destPath)
}

// GetSchemaVersion returns the current schema version
func (m *Manager) GetSchemaVersion() (uint64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.db.GetSchemaVersion()
}

// GetStats returns storage statistics
func (m *Manager) GetStats() (map[string]interface{}, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return map[string]interface{}{
		"upstreams": "managed",
		"tools":     "indexed",
	}, nil
}

// Alias methods for compatibility with MCP server expectations

// ListUpstreams is an alias for ListUpstreamServers
func (m *Manager) ListUpstreams() ([]*config.ServerConfig, error) {
	return m.ListUpstreamServers()
}

// AddUpstream adds an upstream server and returns its ID
func (m *Manager) AddUpstream(serverConfig *config.ServerConfig) (string, error) {
	err := m.SaveUpstreamServer(serverConfig)
	if err != nil {
		return "", err
	}
	return serverConfig.Name, nil // Use name as ID
}

// RemoveUpstream removes an upstream server by ID/name
func (m *Manager) RemoveUpstream(id string) error {
	return m.DeleteUpstreamServer(id)
}

// UpdateUpstream updates an upstream server configuration
func (m *Manager) UpdateUpstream(id string, serverConfig *config.ServerConfig) error {
	// Ensure the ID matches the name
	serverConfig.Name = id
	return m.SaveUpstreamServer(serverConfig)
}

// GetToolStats gets tool statistics formatted for MCP responses
func (m *Manager) GetToolStats(topN int) ([]map[string]interface{}, error) {
	stats, err := m.GetToolStatistics(topN)
	if err != nil {
		return nil, err
	}

	var result []map[string]interface{}
	for _, tool := range stats.TopTools {
		result = append(result, map[string]interface{}{
			"tool_name": tool.ToolName,
			"count":     tool.Count,
		})
	}

	return result, nil
}

// SaveToolMetadata saves tool metadata to the database for lazy loading
// Tools are stored with key: {serverID}:{toolName}
func (m *Manager) SaveToolMetadata(serverID string, tools []*config.ToolMetadata) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.db.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(ToolMetadataBucket))
		if bucket == nil {
			return fmt.Errorf("tool metadata bucket not found")
		}

		now := time.Now()

		for _, tool := range tools {
			record := &ToolMetadataRecord{
				ServerID:     serverID,
				ToolName:     tool.Name,
				PrefixedName: fmt.Sprintf("%s:%s", serverID, tool.Name),
				Description:  tool.Description,
				InputSchema:  map[string]interface{}{}, // Store as empty map for now
				Created:      now,
				Updated:      now,
			}

			// If tool has ParamsJSON, store it in InputSchema as a marker
			if tool.ParamsJSON != "" {
				record.InputSchema = map[string]interface{}{
					"_params_json": tool.ParamsJSON,
				}
			}

			key := fmt.Sprintf("%s:%s", serverID, tool.Name)
			data, err := record.MarshalBinary()
			if err != nil {
				return fmt.Errorf("failed to marshal tool metadata: %w", err)
			}

			if err := bucket.Put([]byte(key), data); err != nil {
				return fmt.Errorf("failed to save tool metadata: %w", err)
			}
		}

		m.logger.Infof("Saved %d tool metadata records for server %s", len(tools), serverID)
		return nil
	})
}

// GetToolMetadata retrieves tool metadata for a specific server from the database
func (m *Manager) GetToolMetadata(serverID string) ([]*config.ToolMetadata, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var tools []*config.ToolMetadata

	err := m.db.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(ToolMetadataBucket))
		if bucket == nil {
			return fmt.Errorf("tool metadata bucket not found")
		}

		// Iterate through all tools for this server
		prefix := []byte(serverID + ":")
		cursor := bucket.Cursor()

		for k, v := cursor.Seek(prefix); k != nil && len(k) >= len(prefix) && string(k[:len(prefix)]) == string(prefix); k, v = cursor.Next() {
			var record ToolMetadataRecord
			if err := record.UnmarshalBinary(v); err != nil {
				m.logger.Warnf("Failed to unmarshal tool metadata for key %s: %v", string(k), err)
				continue
			}

			// Extract ParamsJSON from InputSchema if it exists
			paramsJSON := ""
			if pj, ok := record.InputSchema["_params_json"].(string); ok {
				paramsJSON = pj
			}

			tools = append(tools, &config.ToolMetadata{
				Name:        record.PrefixedName, // Use prefixed name for consistency
				ServerName:  record.ServerID,
				Description: record.Description,
				ParamsJSON:  paramsJSON,
				Hash:        "",
				Created:     record.Created,
				Updated:     record.Updated,
			})
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	m.logger.Debugf("Retrieved %d tool metadata records for server %s from database", len(tools), serverID)
	return tools, nil
}

// GetAllToolMetadata retrieves all tool metadata from the database
func (m *Manager) GetAllToolMetadata() ([]*config.ToolMetadata, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var tools []*config.ToolMetadata

	err := m.db.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(ToolMetadataBucket))
		if bucket == nil {
			return fmt.Errorf("tool metadata bucket not found")
		}

		return bucket.ForEach(func(k, v []byte) error {
			var record ToolMetadataRecord
			if err := record.UnmarshalBinary(v); err != nil {
				m.logger.Warnf("Failed to unmarshal tool metadata for key %s: %v", string(k), err)
				return nil // Continue to next record
			}

			// Extract ParamsJSON from InputSchema if it exists
			paramsJSON := ""
			if pj, ok := record.InputSchema["_params_json"].(string); ok {
				paramsJSON = pj
			}

			tools = append(tools, &config.ToolMetadata{
				Name:        record.PrefixedName,
				ServerName:  record.ServerID,
				Description: record.Description,
				ParamsJSON:  paramsJSON,
				Hash:        "",
				Created:     record.Created,
				Updated:     record.Updated,
			})
			return nil
		})
	})

	if err != nil {
		return nil, err
	}

	m.logger.Debugf("Retrieved %d total tool metadata records from database", len(tools))
	return tools, nil
}

// DeleteServerToolMetadata deletes all tool metadata for a specific server
func (m *Manager) DeleteServerToolMetadata(serverID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.db.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(ToolMetadataBucket))
		if bucket == nil {
			return fmt.Errorf("tool metadata bucket not found")
		}

		// Delete all tools with this server prefix
		prefix := []byte(serverID + ":")
		cursor := bucket.Cursor()

		keysToDelete := [][]byte{}
		for k, _ := cursor.Seek(prefix); k != nil && len(k) > 0 && string(k[:len(prefix)]) == string(prefix); k, _ = cursor.Next() {
			// Copy the key since it will be invalid after cursor moves
			keyCopy := make([]byte, len(k))
			copy(keyCopy, k)
			keysToDelete = append(keysToDelete, keyCopy)
		}

		// Delete the keys
		for _, key := range keysToDelete {
			if err := bucket.Delete(key); err != nil {
				return fmt.Errorf("failed to delete tool metadata key %s: %w", string(key), err)
			}
		}

		m.logger.Infof("Deleted %d tool metadata records for server %s", len(keysToDelete), serverID)
		return nil
	})
}

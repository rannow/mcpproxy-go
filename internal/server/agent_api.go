package server

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"go.uber.org/zap"

	"mcpproxy-go/internal/config"
)

// Agent API handlers for Python MCP agent integration
// These endpoints provide programmatic access to server management,
// diagnostics, configuration, and logs for AI agent automation

// handleAgentListServers lists all configured MCP servers with their status
// GET /api/v1/agent/servers
func (s *Server) handleAgentListServers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	servers := make([]map[string]interface{}, 0)

	for _, serverConfig := range s.config.Servers {
		client, exists := s.upstreamManager.GetClient(serverConfig.Name)

		statusInfo := map[string]interface{}{
			"state":     "Unknown",
			"connected": false,
		}

		if exists && client != nil {
			statusInfo["state"] = "Ready"
			statusInfo["connected"] = true
		}

		serverInfo := map[string]interface{}{
			"name":        serverConfig.Name,
			"url":         serverConfig.URL,
			"command":     serverConfig.Command,
			"args":        serverConfig.Args,
			"protocol":    serverConfig.Protocol,
			"enabled":     serverConfig.Enabled,
			"quarantined": serverConfig.Quarantined,
			"working_dir": serverConfig.WorkingDir,
			"status":      statusInfo,
		}

		servers = append(servers, serverInfo)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"servers": servers,
		"total":   len(servers),
	})
}

// handleAgentServerDetails gets detailed information about a specific server
// GET /api/v1/agent/servers/{name}
func (s *Server) handleAgentServerDetails(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract server name from path: /api/v1/agent/servers/{name}
	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/v1/agent/servers/"), "/")
	if len(pathParts) == 0 || pathParts[0] == "" {
		http.Error(w, "Server name required", http.StatusBadRequest)
		return
	}
	serverName := pathParts[0]

	// Find server config
	var serverConfig *config.ServerConfig
	for i := range s.config.Servers {
		if s.config.Servers[i].Name == serverName {
			serverConfig = s.config.Servers[i]
			break
		}
	}

	if serverConfig == nil {
		http.Error(w, fmt.Sprintf("Server '%s' not found", serverName), http.StatusNotFound)
		return
	}

	// Get server client
	client, exists := s.upstreamManager.GetClient(serverName)

	statusInfo := map[string]interface{}{
		"state":     "Unknown",
		"connected": false,
	}

	toolCount := 0
	if exists && client != nil {
		statusInfo["state"] = "Ready"
		statusInfo["connected"] = true

		// Get tool count from client
		if tools, err := client.ListTools(r.Context()); err == nil {
			toolCount = len(tools)
		}
	}

	serverInfo := map[string]interface{}{
		"name":        serverConfig.Name,
		"url":         serverConfig.URL,
		"command":     serverConfig.Command,
		"args":        serverConfig.Args,
		"env":         serverConfig.Env,
		"protocol":    serverConfig.Protocol,
		"enabled":     serverConfig.Enabled,
		"quarantined": serverConfig.Quarantined,
		"working_dir": serverConfig.WorkingDir,
		"status":      statusInfo,
		"tools": map[string]interface{}{
			"count": toolCount,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(serverInfo)
}

// handleAgentServerLogs retrieves logs for a specific server
// GET /api/v1/agent/servers/{name}/logs?lines=100&filter=error
func (s *Server) handleAgentServerLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract server name from path
	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/v1/agent/servers/"), "/")
	if len(pathParts) < 2 || pathParts[0] == "" {
		http.Error(w, "Server name required", http.StatusBadRequest)
		return
	}
	serverName := pathParts[0]

	// Parse query parameters
	query := r.URL.Query()
	lines := 100
	if l := query.Get("lines"); l != "" {
		fmt.Sscanf(l, "%d", &lines)
	}
	if lines > 1000 {
		lines = 1000 // Limit to prevent excessive data
	}

	filterPattern := query.Get("filter")

	// Get server log file path
	logDir := s.config.DataDir
	if logDir == "" {
		logDir = filepath.Join(os.Getenv("HOME"), ".mcpproxy")
	}
	logFile := filepath.Join(logDir, "logs", fmt.Sprintf("server-%s.log", serverName))

	// Read log file
	logEntries, err := s.readLogFile(logFile, lines, filterPattern)
	if err != nil {
		s.logger.Warn("Failed to read server log",
			zap.String("server", serverName),
			zap.Error(err))
		// Return empty array instead of error for better UX
		logEntries = []map[string]interface{}{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"server":  serverName,
		"logs":    logEntries,
		"count":   len(logEntries),
		"limited": len(logEntries) == lines,
	})
}

// handleAgentMainLogs retrieves main mcpproxy logs
// GET /api/v1/agent/logs/main?lines=100&filter=error
func (s *Server) handleAgentMainLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse query parameters
	query := r.URL.Query()
	lines := 100
	if l := query.Get("lines"); l != "" {
		fmt.Sscanf(l, "%d", &lines)
	}
	if lines > 1000 {
		lines = 1000
	}

	filterPattern := query.Get("filter")

	// Get main log file path
	logDir := s.config.DataDir
	if logDir == "" {
		logDir = filepath.Join(os.Getenv("HOME"), ".mcpproxy")
	}
	logFile := filepath.Join(logDir, "logs", "main.log")

	// Read log file
	logEntries, err := s.readLogFile(logFile, lines, filterPattern)
	if err != nil {
		s.logger.Warn("Failed to read main log", zap.Error(err))
		logEntries = []map[string]interface{}{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"logs":    logEntries,
		"count":   len(logEntries),
		"limited": len(logEntries) == lines,
	})
}

// handleAgentServerConfig gets server configuration
// GET /api/v1/agent/servers/{name}/config
func (s *Server) handleAgentServerConfig(w http.ResponseWriter, r *http.Request) {
	// Extract server name from path
	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/v1/agent/servers/"), "/")
	if len(pathParts) < 2 || pathParts[0] == "" {
		http.Error(w, "Server name required", http.StatusBadRequest)
		return
	}
	serverName := pathParts[0]

	switch r.Method {
	case http.MethodGet:
		s.handleAgentGetServerConfig(w, r, serverName)
	case http.MethodPatch:
		s.handleAgentPatchServerConfig(w, r, serverName)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleAgentGetServerConfig retrieves server configuration
func (s *Server) handleAgentGetServerConfig(w http.ResponseWriter, r *http.Request, serverName string) {
	// Find server config
	var serverConfig *config.ServerConfig
	for i := range s.config.Servers {
		if s.config.Servers[i].Name == serverName {
			serverConfig = s.config.Servers[i]
			break
		}
	}

	if serverConfig == nil {
		http.Error(w, fmt.Sprintf("Server '%s' not found", serverName), http.StatusNotFound)
		return
	}

	configData := map[string]interface{}{
		"name":        serverConfig.Name,
		"url":         serverConfig.URL,
		"command":     serverConfig.Command,
		"args":        serverConfig.Args,
		"env":         serverConfig.Env,
		"protocol":    serverConfig.Protocol,
		"enabled":     serverConfig.Enabled,
		"quarantined": serverConfig.Quarantined,
		"working_dir": serverConfig.WorkingDir,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(configData)
}

// handleAgentPatchServerConfig updates server configuration
// PATCH /api/v1/agent/servers/{name}/config
func (s *Server) handleAgentPatchServerConfig(w http.ResponseWriter, r *http.Request, serverName string) {
	// Parse request body
	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	// Find server config
	var serverConfig *config.ServerConfig
	var serverIndex int
	for i := range s.config.Servers {
		if s.config.Servers[i].Name == serverName {
			serverConfig = s.config.Servers[i]
			serverIndex = i
			break
		}
	}

	if serverConfig == nil {
		http.Error(w, fmt.Sprintf("Server '%s' not found", serverName), http.StatusNotFound)
		return
	}

	// Apply updates
	needsRestart := false

	if url, ok := updates["url"].(string); ok {
		serverConfig.URL = url
		needsRestart = true
	}
	if command, ok := updates["command"].(string); ok {
		serverConfig.Command = command
		needsRestart = true
	}
	if args, ok := updates["args"].([]interface{}); ok {
		strArgs := make([]string, len(args))
		for i, arg := range args {
			strArgs[i] = fmt.Sprint(arg)
		}
		serverConfig.Args = strArgs
		needsRestart = true
	}
	if env, ok := updates["env"].(map[string]interface{}); ok {
		envMap := make(map[string]string)
		for k, v := range env {
			envMap[k] = fmt.Sprint(v)
		}
		serverConfig.Env = envMap
		needsRestart = true
	}
	if protocol, ok := updates["protocol"].(string); ok {
		serverConfig.Protocol = protocol
		needsRestart = true
	}
	if enabled, ok := updates["enabled"].(bool); ok {
		serverConfig.Enabled = enabled
		needsRestart = true
	}
	if quarantined, ok := updates["quarantined"].(bool); ok {
		serverConfig.Quarantined = quarantined
	}
	if workingDir, ok := updates["working_dir"].(string); ok {
		serverConfig.WorkingDir = workingDir
		needsRestart = true
	}

	// Update config in memory
	s.config.Servers[serverIndex] = serverConfig

	// Save config to file
	if err := config.SaveConfig(s.config, s.configPath); err != nil {
		s.logger.Error("Failed to save config after update",
			zap.String("server", serverName),
			zap.Error(err))
		http.Error(w, fmt.Sprintf("Failed to save configuration: %v", err), http.StatusInternalServerError)
		return
	}

	// If needs restart and server is enabled, restart it
	if needsRestart && serverConfig.Enabled {
		s.logger.Info("Restarting server after configuration update",
			zap.String("server", serverName))
		// This would trigger a server restart
		// For now, just log it - the file watcher will handle the reload
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":        true,
		"message":        "Configuration updated successfully",
		"needs_restart":  needsRestart,
		"server_enabled": serverConfig.Enabled,
	})
}

// readLogFile reads the last N lines from a log file with optional filtering
func (s *Server) readLogFile(logPath string, lines int, filterPattern string) ([]map[string]interface{}, error) {
	file, err := os.Open(logPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Read all lines first
	var allLines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		// Apply filter if specified
		if filterPattern != "" && !strings.Contains(strings.ToLower(line), strings.ToLower(filterPattern)) {
			continue
		}

		allLines = append(allLines, line)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// Get last N lines
	startIndex := 0
	if len(allLines) > lines {
		startIndex = len(allLines) - lines
	}
	selectedLines := allLines[startIndex:]

	// Parse log entries
	logEntries := make([]map[string]interface{}, 0, len(selectedLines))
	for _, line := range selectedLines {
		entry := s.parseLogLine(line)
		logEntries = append(logEntries, entry)
	}

	return logEntries, nil
}

// parseLogLine attempts to parse a log line into structured data
func (s *Server) parseLogLine(line string) map[string]interface{} {
	// Try to parse as JSON (zap structured logging)
	var jsonEntry map[string]interface{}
	if err := json.Unmarshal([]byte(line), &jsonEntry); err == nil {
		return jsonEntry
	}

	// Fallback: parse as plain text
	// Expected format: timestamp level message
	parts := strings.SplitN(line, "\t", 3)
	entry := map[string]interface{}{
		"raw": line,
	}

	if len(parts) >= 3 {
		entry["timestamp"] = parts[0]
		entry["level"] = parts[1]
		entry["message"] = parts[2]
	} else {
		entry["message"] = line
	}

	return entry
}

// handleAgentSearchRegistries searches MCP server registries
// GET /api/v1/agent/registries/search?query=weather&registry=smithery
func (s *Server) handleAgentSearchRegistries(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	query := r.URL.Query()
	searchQuery := query.Get("query")
	registry := query.Get("registry")

	if searchQuery == "" {
		http.Error(w, "Query parameter required", http.StatusBadRequest)
		return
	}

	// This would integrate with the existing search_servers MCP tool
	// For now, return a placeholder response
	s.logger.Info("Agent registry search",
		zap.String("query", searchQuery),
		zap.String("registry", registry))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"results": []map[string]interface{}{},
		"query":   searchQuery,
		"message": "Registry search integration pending",
	})
}

// handleAgentInstallServer installs a new MCP server
// POST /api/v1/agent/install
func (s *Server) handleAgentInstallServer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var installRequest struct {
		ServerID string                 `json:"server_id"`
		Name     string                 `json:"name"`
		Config   map[string]interface{} `json:"config"`
	}

	if err := json.NewDecoder(r.Body).Decode(&installRequest); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	s.logger.Info("Agent server installation request",
		zap.String("server_id", installRequest.ServerID),
		zap.String("name", installRequest.Name))

	// This would integrate with the installation logic
	// For now, return a placeholder
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": false,
		"message": "Server installation integration pending",
	})
}
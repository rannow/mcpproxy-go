package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"mcpproxy-go/internal/config"

	"go.uber.org/zap"
)

// InspectorStartRequest represents the request body for starting the inspector with a specific server
type InspectorStartRequest struct {
	ServerName string `json:"server_name"`
}

// handleInspectorStart launches the MCP Inspector connected to mcpproxy for a specific server
func (s *Server) handleInspectorStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req InspectorStartRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.logger.Error("Failed to decode inspector start request", zap.Error(err))
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Invalid request body",
		})
		return
	}

	// Verify the server exists
	allServers, err := s.storageManager.ListUpstreamServers()
	if err != nil {
		s.logger.Error("Failed to list servers for inspector", zap.Error(err))
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Failed to retrieve server list",
		})
		return
	}

	// Find the requested server
	var targetServer *config.ServerConfig
	for i, server := range allServers {
		if server.Name == req.ServerName {
			targetServer = allServers[i]
			break
		}
	}

	if targetServer == nil {
		s.logger.Error("Server not found for inspector", zap.String("server", req.ServerName))
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("Server '%s' not found", req.ServerName),
		})
		return
	}

	// Create inspector config that connects to mcpproxy
	// The inspector will connect to mcpproxy, which aggregates all servers
	inspectorConfig := make(map[string]interface{})
	mcpServers := make(map[string]interface{})

	// Build mcpproxy URL from listen address
	listenAddr := s.GetListenAddress()
	proxyURL := fmt.Sprintf("http://localhost%s/mcp", listenAddr)

	// Configure inspector to connect to mcpproxy server
	serverConfig := make(map[string]interface{})
	serverConfig["url"] = proxyURL

	// Use a descriptive name that indicates this is mcpproxy aggregating the target server
	serverName := fmt.Sprintf("mcpproxy-%s", req.ServerName)
	mcpServers[serverName] = serverConfig
	inspectorConfig["mcpServers"] = mcpServers

	// Write config to temporary file with unique name to allow multiple instances
	tempDir := os.TempDir()
	configPath := filepath.Join(tempDir, fmt.Sprintf("mcp_inspector_%s.json", req.ServerName))

	configData, err := json.MarshalIndent(inspectorConfig, "", "  ")
	if err != nil {
		s.logger.Error("Failed to marshal inspector config", zap.Error(err))
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Failed to create inspector configuration",
		})
		return
	}

	if err := os.WriteFile(configPath, configData, 0644); err != nil {
		s.logger.Error("Failed to write inspector config", zap.Error(err))
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Failed to write configuration file",
		})
		return
	}

	s.logger.Info("Created MCP Inspector configuration for mcpproxy access",
		zap.String("target_server", req.ServerName),
		zap.String("proxy_url", proxyURL),
		zap.String("config_path", configPath))

	// Launch MCP Inspector in background
	cmd := exec.Command("npx", "@modelcontextprotocol/inspector", configPath)
	cmd.Dir = tempDir

	// Capture output for logging
	output := &strings.Builder{}
	cmd.Stdout = output
	cmd.Stderr = output

	if err := cmd.Start(); err != nil {
		s.logger.Error("Failed to start MCP Inspector", zap.Error(err))
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("Failed to launch inspector: %v", err),
		})
		return
	}

	s.logger.Info("MCP Inspector started",
		zap.String("target_server", req.ServerName),
		zap.String("proxy_url", proxyURL),
		zap.Int("pid", cmd.Process.Pid),
		zap.String("config", configPath))

	// Return success response with URL (inspector will open automatically)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("MCP Inspector launched for server '%s' via mcpproxy. Browser should open automatically.", req.ServerName),
		"url":     "http://localhost:6274", // Default inspector port
		"pid":     cmd.Process.Pid,
	})
}

// handleLaunchInspector launches the MCP Inspector connected to mcpproxy (all servers)
func (s *Server) handleLaunchInspector(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Create inspector config that connects to mcpproxy
	// The inspector will connect to mcpproxy, which aggregates all servers
	inspectorConfig := make(map[string]interface{})
	mcpServers := make(map[string]interface{})

	// Build mcpproxy URL from listen address
	listenAddr := s.GetListenAddress()
	proxyURL := fmt.Sprintf("http://localhost%s/mcp", listenAddr)

	// Configure inspector to connect to mcpproxy server
	serverConfig := make(map[string]interface{})
	serverConfig["url"] = proxyURL

	// Use mcpproxy as the server name
	mcpServers["mcpproxy"] = serverConfig
	inspectorConfig["mcpServers"] = mcpServers

	// Write config to temporary file
	tempDir := os.TempDir()
	configPath := filepath.Join(tempDir, "mcp_inspector_all_servers.json")

	configData, err := json.MarshalIndent(inspectorConfig, "", "  ")
	if err != nil {
		s.logger.Error("Failed to marshal inspector config", zap.Error(err))
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Failed to create inspector configuration",
		})
		return
	}

	if err := os.WriteFile(configPath, configData, 0644); err != nil {
		s.logger.Error("Failed to write inspector config", zap.Error(err))
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Failed to write configuration file",
		})
		return
	}

	s.logger.Info("Created MCP Inspector configuration for mcpproxy",
		zap.String("proxy_url", proxyURL),
		zap.String("config_path", configPath))

	// Launch MCP Inspector in background
	cmd := exec.Command("npx", "@modelcontextprotocol/inspector", configPath)
	cmd.Dir = tempDir

	// Capture output for logging
	output := &strings.Builder{}
	cmd.Stdout = output
	cmd.Stderr = output

	if err := cmd.Start(); err != nil {
		s.logger.Error("Failed to start MCP Inspector", zap.Error(err))
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Failed to launch inspector: %v", err),
		})
		return
	}

	s.logger.Info("MCP Inspector started for mcpproxy",
		zap.String("proxy_url", proxyURL),
		zap.Int("pid", cmd.Process.Pid),
		zap.String("config", configPath))

	// Return success response
	// Note: The inspector will open in browser automatically
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "MCP Inspector launched successfully via mcpproxy. Browser should open automatically.",
		"url":     "http://localhost:6274", // Default inspector port
		"pid":     cmd.Process.Pid,
	})
}

//go:build !nogui && !headless && !linux

package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"go.uber.org/zap"
)

// FastActionRequest represents a fast action request
type FastActionRequest struct {
	Action     string `json:"action"`
	ServerName string `json:"server_name"`
}

// FastActionResponse represents a fast action response
type FastActionResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// handleFastAction handles fast action button requests
func (s *Server) handleFastAction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req FastActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var response FastActionResponse

	switch req.Action {
	case "check_startup":
		response = s.checkServerStartup(req.ServerName)
	case "test_local":
		response = s.testServerLocally(req.ServerName)
	case "check_docs":
		response = s.checkDocumentation(req.ServerName)
	case "preload_packages":
		response = s.preloadPackages(req.ServerName)
	case "check_disabled_servers":
		response = s.checkDisabledServers()
	default:
		response = FastActionResponse{
			Success: false,
			Error:   "Unknown action",
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// checkServerStartup analyzes why a server is not starting
func (s *Server) checkServerStartup(serverName string) FastActionResponse {
	s.logger.Info("Checking server startup", zap.String("server", serverName))

	servers, err := s.GetAllServers()
	if err != nil {
		return FastActionResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to get servers: %v", err),
		}
	}

	var serverConfig map[string]interface{}
	for _, srv := range servers {
		if name, ok := srv["name"].(string); ok && name == serverName {
			serverConfig = srv
			break
		}
	}

	if serverConfig == nil {
		return FastActionResponse{
			Success: false,
			Error:   "Server not found",
		}
	}

	issues := []string{}
	details := make(map[string]interface{})

	// Check if server is enabled
	enabled, _ := serverConfig["enabled"].(bool)
	details["enabled"] = enabled
	if !enabled {
		issues = append(issues, "❌ Server is disabled")
	} else {
		issues = append(issues, "✅ Server is enabled")
	}

	// Check protocol
	protocol, _ := serverConfig["protocol"].(string)
	details["protocol"] = protocol

	if protocol == "stdio" {
		// Check command exists
		command, _ := serverConfig["command"].(string)
		details["command"] = command

		if command == "" {
			issues = append(issues, "❌ No command specified")
		} else {
			// Check if command is in PATH
			_, err := exec.LookPath(command)
			if err != nil {
				issues = append(issues, fmt.Sprintf("❌ Command '%s' not found in PATH", command))
				details["command_found"] = false
			} else {
				issues = append(issues, fmt.Sprintf("✅ Command '%s' found", command))
				details["command_found"] = true
			}
		}

		// Check working directory
		workingDir, _ := serverConfig["working_dir"].(string)
		if workingDir != "" {
			details["working_dir"] = workingDir
			if _, err := os.Stat(workingDir); os.IsNotExist(err) {
				issues = append(issues, fmt.Sprintf("❌ Working directory does not exist: %s", workingDir))
				details["working_dir_exists"] = false
			} else {
				issues = append(issues, "✅ Working directory exists")
				details["working_dir_exists"] = true
			}
		}
	} else if protocol == "http" || protocol == "sse" || protocol == "streamable-http" {
		// Check URL
		url, _ := serverConfig["url"].(string)
		details["url"] = url
		if url == "" {
			issues = append(issues, "❌ No URL specified")
		} else {
			issues = append(issues, fmt.Sprintf("✅ URL configured: %s", url))
		}
	}

	// Check last error
	lastError, _ := serverConfig["last_error"].(string)
	if lastError != "" {
		issues = append(issues, fmt.Sprintf("❌ Last error: %s", lastError))
		details["last_error"] = lastError
	}

	// Check connection state
	connectionState, _ := serverConfig["connection_state"].(string)
	details["connection_state"] = connectionState
	if connectionState == "Ready" {
		issues = append(issues, "✅ Server is connected and ready")
	} else {
		issues = append(issues, fmt.Sprintf("⚠️  Connection state: %s", connectionState))
	}

	// Check quarantine status
	quarantined, _ := serverConfig["quarantined"].(bool)
	details["quarantined"] = quarantined
	if quarantined {
		issues = append(issues, "⚠️  Server is quarantined for security review")
	}

	// Read recent log entries
	logPath := filepath.Join(os.ExpandEnv("$HOME"), "Library", "Logs", "mcpproxy", fmt.Sprintf("server-%s.log", serverName))
	if _, err := os.Stat(logPath); err == nil {
		// Read last 20 lines of log
		cmd := exec.Command("tail", "-n", "20", logPath)
		if output, err := cmd.Output(); err == nil {
			details["recent_logs"] = string(output)
		}
	}

	message := strings.Join(issues, "\n")
	return FastActionResponse{
		Success: true,
		Message: message,
		Details: details,
	}
}

// testServerLocally tests starting the server locally
func (s *Server) testServerLocally(serverName string) FastActionResponse {
	s.logger.Info("Testing server locally", zap.String("server", serverName))

	servers, err := s.GetAllServers()
	if err != nil {
		return FastActionResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to get servers: %v", err),
		}
	}

	var serverConfig map[string]interface{}
	for _, srv := range servers {
		if name, ok := srv["name"].(string); ok && name == serverName {
			serverConfig = srv
			break
		}
	}

	if serverConfig == nil {
		return FastActionResponse{
			Success: false,
			Error:   "Server not found",
		}
	}

	protocol, _ := serverConfig["protocol"].(string)
	if protocol != "stdio" {
		return FastActionResponse{
			Success: false,
			Error:   "Only stdio servers can be tested locally",
		}
	}

	command, _ := serverConfig["command"].(string)
	args, _ := serverConfig["args"].([]interface{})
	workingDir, _ := serverConfig["working_dir"].(string)

	if command == "" {
		return FastActionResponse{
			Success: false,
			Error:   "No command specified",
		}
	}

	// Build command arguments
	cmdArgs := []string{}
	for _, arg := range args {
		if argStr, ok := arg.(string); ok {
			cmdArgs = append(cmdArgs, argStr)
		}
	}

	// Create command
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, command, cmdArgs...)
	if workingDir != "" {
		cmd.Dir = workingDir
	}

	// Set environment variables
	cmd.Env = os.Environ()
	if envMap, ok := serverConfig["env"].(map[string]interface{}); ok {
		for key, value := range envMap {
			if valStr, ok := value.(string); ok {
				cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, valStr))
			}
		}
	}

	// Capture output
	output, err := cmd.CombinedOutput()

	details := make(map[string]interface{})
	details["command"] = command
	details["args"] = cmdArgs
	details["working_dir"] = workingDir
	details["output"] = string(output)

	if err != nil {
		return FastActionResponse{
			Success: false,
			Message: fmt.Sprintf("Server test failed: %v", err),
			Details: details,
			Error:   err.Error(),
		}
	}

	return FastActionResponse{
		Success: true,
		Message: "✅ Server started successfully locally",
		Details: details,
	}
}

// checkDocumentation fetches and analyzes server documentation
func (s *Server) checkDocumentation(serverName string) FastActionResponse {
	s.logger.Info("Checking documentation", zap.String("server", serverName))

	servers, err := s.GetAllServers()
	if err != nil {
		return FastActionResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to get servers: %v", err),
		}
	}

	var serverConfig map[string]interface{}
	for _, srv := range servers {
		if name, ok := srv["name"].(string); ok && name == serverName {
			serverConfig = srv
			break
		}
	}

	if serverConfig == nil {
		return FastActionResponse{
			Success: false,
			Error:   "Server not found",
		}
	}

	repoURL, _ := serverConfig["repository_url"].(string)
	if repoURL == "" {
		return FastActionResponse{
			Success: false,
			Error:   "No repository URL configured",
		}
	}

	// Try to fetch README from GitHub
	readmeURL := ""
	if strings.Contains(repoURL, "github.com") {
		// Convert GitHub URL to raw README URL
		repoURL = strings.TrimSuffix(repoURL, "/")
		repoURL = strings.TrimSuffix(repoURL, ".git")
		readmeURL = strings.Replace(repoURL, "github.com", "raw.githubusercontent.com", 1) + "/main/README.md"
	}

	details := make(map[string]interface{})
	details["repository_url"] = repoURL
	details["readme_url"] = readmeURL

	if readmeURL != "" {
		// Fetch README
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, "GET", readmeURL, nil)
		if err != nil {
			return FastActionResponse{
				Success: false,
				Message: "Failed to create request",
				Error:   err.Error(),
			}
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			// Try master branch instead
			readmeURL = strings.Replace(readmeURL, "/main/", "/master/", 1)
			req, _ = http.NewRequestWithContext(ctx, "GET", readmeURL, nil)
			resp, err = http.DefaultClient.Do(req)
			if err != nil {
				return FastActionResponse{
					Success: false,
					Message: "Failed to fetch README",
					Error:   err.Error(),
				}
			}
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			// Read README content (limit to 10KB)
			readme := make([]byte, 10240)
			n, _ := resp.Body.Read(readme)
			details["readme_content"] = string(readme[:n])

			return FastActionResponse{
				Success: true,
				Message: "✅ Documentation fetched successfully",
				Details: details,
			}
		} else {
			return FastActionResponse{
				Success: false,
				Message: fmt.Sprintf("Failed to fetch README (status: %d)", resp.StatusCode),
				Details: details,
			}
		}
	}

	return FastActionResponse{
		Success: false,
		Message: "Could not determine README URL",
		Details: details,
	}
}

// preloadPackages installs required packages for the server
func (s *Server) preloadPackages(serverName string) FastActionResponse {
	s.logger.Info("Preloading packages", zap.String("server", serverName))

	servers, err := s.GetAllServers()
	if err != nil {
		return FastActionResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to get servers: %v", err),
		}
	}

	var serverConfig map[string]interface{}
	for _, srv := range servers {
		if name, ok := srv["name"].(string); ok && name == serverName {
			serverConfig = srv
			break
		}
	}

	if serverConfig == nil {
		return FastActionResponse{
			Success: false,
			Error:   "Server not found",
		}
	}

	command, _ := serverConfig["command"].(string)
	args, _ := serverConfig["args"].([]interface{})

	results := []string{}
	details := make(map[string]interface{})

	// Determine package manager based on command
	switch command {
	case "uvx":
		// Install with uv
		if len(args) > 0 {
			packageName, _ := args[0].(string)
			results = append(results, fmt.Sprintf("Installing Python package with uv: %s", packageName))

			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()

			cmd := exec.CommandContext(ctx, "uv", "tool", "install", packageName)
			output, err := cmd.CombinedOutput()
			details["uv_output"] = string(output)

			if err != nil {
				results = append(results, fmt.Sprintf("❌ Failed to install: %v", err))
			} else {
				results = append(results, "✅ Package installed successfully")
			}
		}

	case "npx":
		// Install with npm
		if len(args) > 0 {
			packageName, _ := args[0].(string)
			results = append(results, fmt.Sprintf("Installing npm package: %s", packageName))

			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()

			cmd := exec.CommandContext(ctx, "npm", "install", "-g", packageName)
			output, err := cmd.CombinedOutput()
			details["npm_output"] = string(output)

			if err != nil {
				results = append(results, fmt.Sprintf("❌ Failed to install: %v", err))
			} else {
				results = append(results, "✅ Package installed successfully")
			}
		}

	case "python", "python3":
		// Check if package is specified in args
		packageName := ""
		for _, arg := range args {
			if argStr, ok := arg.(string); ok && !strings.HasPrefix(argStr, "-") {
				packageName = argStr
				break
			}
		}

		if packageName != "" {
			results = append(results, fmt.Sprintf("Installing Python package with pip: %s", packageName))

			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()

			cmd := exec.CommandContext(ctx, "pip", "install", packageName)
			output, err := cmd.CombinedOutput()
			details["pip_output"] = string(output)

			if err != nil {
				results = append(results, fmt.Sprintf("❌ Failed to install: %v", err))
			} else {
				results = append(results, "✅ Package installed successfully")
			}
		}

	default:
		return FastActionResponse{
			Success: false,
			Message: fmt.Sprintf("Unsupported command: %s", command),
		}
	}

	message := strings.Join(results, "\n")
	return FastActionResponse{
		Success: true,
		Message: message,
		Details: details,
	}
}

// checkDisabledServers checks all disabled servers and creates a report
func (s *Server) checkDisabledServers() FastActionResponse {
	s.logger.Info("Checking all disabled servers")

	servers, err := s.GetAllServers()
	if err != nil {
		return FastActionResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to get servers: %v", err),
		}
	}

	disabledServers := []map[string]interface{}{}
	for _, srv := range servers {
		if enabled, ok := srv["enabled"].(bool); ok && !enabled {
			disabledServers = append(disabledServers, srv)
		}
	}

	if len(disabledServers) == 0 {
		return FastActionResponse{
			Success: true,
			Message: "✅ No disabled servers found",
			Details: map[string]interface{}{
				"count": 0,
			},
		}
	}

	// Analyze each disabled server
	report := []map[string]interface{}{}
	for _, srv := range disabledServers {
		serverName, _ := srv["name"].(string)
		analysis := s.checkServerStartup(serverName)

		report = append(report, map[string]interface{}{
			"server":   serverName,
			"analysis": analysis,
		})
	}

	message := fmt.Sprintf("📊 Analyzed %d disabled servers", len(disabledServers))

	return FastActionResponse{
		Success: true,
		Message: message,
		Details: map[string]interface{}{
			"count":  len(disabledServers),
			"report": report,
		},
	}
}

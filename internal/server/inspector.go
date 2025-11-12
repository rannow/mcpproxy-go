package server

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

// InspectorManager manages the MCP Inspector process
type InspectorManager struct {
	mu      sync.Mutex
	cmd     *exec.Cmd
	cancel  context.CancelFunc
	running bool
	port    int
	url     string // Full inspector URL with auth token
	logger  *zap.SugaredLogger
}

// NewInspectorManager creates a new inspector manager
func NewInspectorManager(logger *zap.SugaredLogger) *InspectorManager {
	return &InspectorManager{
		port:   5173, // Default MCP Inspector port
		logger: logger,
	}
}

// Start starts the MCP Inspector process without server configuration
func (im *InspectorManager) Start() error {
	return im.StartWithServer("", "")
}

// StartWithServer starts the MCP Inspector with a specific server configuration
func (im *InspectorManager) StartWithServer(serverName, configPath string) error {
	im.mu.Lock()
	defer im.mu.Unlock()

	if im.running {
		im.logger.Info("MCP Inspector is already running, stopping first")
		if err := im.stopLocked(); err != nil {
			im.logger.Warn("Failed to stop existing inspector", zap.Error(err))
		}
		// Small delay to ensure port is released
		im.mu.Unlock()
		time.Sleep(500 * time.Millisecond)
		im.mu.Lock()
	}

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	im.cancel = cancel

	// Build command arguments
	args := []string{"@modelcontextprotocol/inspector"}

	// Add server configuration if provided
	if configPath != "" && serverName != "" {
		// CRITICAL: MCP Inspector expects a numeric index, not a server name
		// Find the server's index in the config file
		serverIndex, err := im.findServerIndex(configPath, serverName)
		if err != nil {
			cancel()
			im.logger.Error("Failed to find server index in config",
				zap.String("server", serverName),
				zap.String("config", configPath),
				zap.Error(err))
			return fmt.Errorf("failed to find server '%s' in config: %w", serverName, err)
		}

		args = append(args, "--config", configPath, "--server", fmt.Sprintf("%d", serverIndex))
		im.logger.Info("Starting MCP Inspector with server configuration",
			zap.String("server", serverName),
			zap.Int("index", serverIndex),
			zap.String("config", configPath))
	} else {
		im.logger.Info("Starting MCP Inspector without server configuration")
	}

	// Start MCP Inspector using npx
	im.cmd = exec.CommandContext(ctx, "npx", args...)

	// Capture stdout and stderr to detect the actual port
	stdout, err := im.cmd.StdoutPipe()
	if err != nil {
		cancel()
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := im.cmd.StderrPipe()
	if err != nil {
		cancel()
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	im.logger.Info("Starting MCP Inspector",
		zap.String("command", "npx @modelcontextprotocol/inspector"))

	// Start the process
	if err := im.cmd.Start(); err != nil {
		cancel()
		im.logger.Error("Failed to start MCP Inspector", zap.Error(err))
		return fmt.Errorf("failed to start MCP Inspector: %w", err)
	}

	im.running = true

	// Channel to signal when port is detected
	portDetected := make(chan bool, 1)

	// Monitor stdout for port information
	go im.monitorOutput(stdout, portDetected)

	// Monitor stderr for errors and port information
	go im.monitorOutput(stderr, portDetected)

	// Monitor the process in a goroutine
	go func() {
		err := im.cmd.Wait()
		im.mu.Lock()
		im.running = false
		im.mu.Unlock()

		if err != nil && ctx.Err() == nil {
			im.logger.Warn("MCP Inspector process exited unexpectedly", zap.Error(err))
		} else {
			im.logger.Info("MCP Inspector process stopped")
		}
	}()

	// Wait for port detection or timeout
	select {
	case <-portDetected:
		im.logger.Info("MCP Inspector started successfully",
			zap.Int("port", im.port),
			zap.String("url", im.url))
	case <-time.After(10 * time.Second):
		im.logger.Warn("Timeout waiting for MCP Inspector port detection, trying lsof fallback",
			zap.Int("default_port", 5173))

		// Fallback: Use lsof to find inspector ports and test which is the client
		if clientPort, clientURL := im.detectPortsViaLsof(); clientPort > 0 {
			im.mu.Lock()
			im.port = clientPort
			im.url = clientURL
			im.mu.Unlock()
			im.logger.Info("Detected inspector client port via lsof",
				zap.Int("port", clientPort),
				zap.String("url", clientURL))
		} else {
			im.logger.Warn("Failed to detect inspector port, using default",
				zap.Int("default_port", 5173))
		}
	}

	return nil
}

// monitorOutput monitors the output stream and detects the inspector URL
func (im *InspectorManager) monitorOutput(reader io.Reader, portDetected chan bool) {
	scanner := bufio.NewScanner(reader)
	// Regex to match port from "Proxy server listening on localhost:6277" line
	portRegex := regexp.MustCompile(`(?:Proxy server listening on|listening on)\s+(?:localhost|127\.0\.0\.1):(\d{4,5})`)
	// Regex to match session token from "Session token: abc123..." line
	tokenRegex := regexp.MustCompile(`Session token:\s+([a-f0-9]{64})`)

	var detectedPort int
	var sessionToken string

	for scanner.Scan() {
		line := scanner.Text()
		im.logger.Debug("Inspector output", zap.String("line", line))

		// Look for the "Proxy server listening on localhost:XXXX" line
		if portMatches := portRegex.FindStringSubmatch(line); len(portMatches) > 1 {
			if port, err := strconv.Atoi(portMatches[1]); err == nil {
				detectedPort = port
				im.logger.Info("Detected MCP Inspector port", zap.Int("port", port))
			}
		}

		// Look for the "Session token: ..." line
		if tokenMatches := tokenRegex.FindStringSubmatch(line); len(tokenMatches) > 1 {
			sessionToken = tokenMatches[1]
			im.logger.Info("Detected MCP Inspector session token", zap.String("token", sessionToken[:16]+"..."))
		}

		// Once we have both port and token, construct the full URL
		if detectedPort > 0 && sessionToken != "" {
			im.mu.Lock()
			im.port = detectedPort
			im.url = fmt.Sprintf("http://localhost:%d/?MCP_PROXY_AUTH_TOKEN=%s", detectedPort, sessionToken)
			im.logger.Info("Constructed MCP Inspector client URL",
				zap.String("url", im.url),
				zap.Int("port", im.port))
			select {
			case portDetected <- true:
			default:
			}
			im.mu.Unlock()
			// We have what we need, can stop monitoring
			break
		}

		// Log error lines for debugging
		if strings.Contains(strings.ToLower(line), "error") {
			im.logger.Warn("Inspector output (error)", zap.String("line", line))
		}
	}
}

// Stop stops the MCP Inspector process
func (im *InspectorManager) Stop() error {
	im.mu.Lock()
	defer im.mu.Unlock()
	return im.stopLocked()
}

// stopLocked stops the inspector without locking (must be called with lock held)
func (im *InspectorManager) stopLocked() error {
	if !im.running {
		return nil
	}

	im.logger.Info("Stopping MCP Inspector")

	if im.cancel != nil {
		im.cancel()
	}

	if im.cmd != nil && im.cmd.Process != nil {
		if err := im.cmd.Process.Kill(); err != nil {
			im.logger.Warn("Failed to kill MCP Inspector process", zap.Error(err))
		}
	}

	im.running = false
	im.logger.Info("MCP Inspector stopped")

	return nil
}

// IsRunning returns whether the inspector is running
func (im *InspectorManager) IsRunning() bool {
	im.mu.Lock()
	defer im.mu.Unlock()
	return im.running
}

// GetURL returns the inspector URL (with auth token if detected)
func (im *InspectorManager) GetURL() string {
	im.mu.Lock()
	defer im.mu.Unlock()

	// Return full URL with auth token if detected
	if im.url != "" {
		return im.url
	}

	// Fallback to basic URL if auth token URL not detected
	return fmt.Sprintf("http://localhost:%d", im.port)
}

// handleInspectorStart starts the MCP Inspector and redirects to it
func (s *Server) handleInspectorStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request body for server configuration
	var req struct {
		ServerName string `json:"server_name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// If no body or invalid JSON, start without server configuration
		s.logger.Info("Received request to start MCP Inspector without server configuration")
	} else {
		s.logger.Info("Received request to start MCP Inspector with server",
			zap.String("server", req.ServerName))
	}

	// Get config path for inspector
	configPath := s.GetConfigPath()
	if configPath == "" {
		configPath = s.config.DataDir + "/mcp_config.json"
	}

	// Start the inspector with server configuration if provided
	if err := s.inspectorManager.StartWithServer(req.ServerName, configPath); err != nil {
		s.logger.Error("Failed to start MCP Inspector", zap.Error(err))
		http.Error(w, fmt.Sprintf("Failed to start MCP Inspector: %v", err), http.StatusInternalServerError)
		return
	}

	// Wait for URL detection (monitoring goroutines may still be running)
	// Poll for up to 5 seconds to get the actual detected URL
	finalURL := s.inspectorManager.GetURL()
	for i := 0; i < 10; i++ {
		url := s.inspectorManager.GetURL()
		if url != fmt.Sprintf("http://localhost:%d", 5173) {
			// Got a non-default URL, use it
			finalURL = url
			break
		}
		time.Sleep(500 * time.Millisecond)
	}

	// Return success response with redirect URL
	response := map[string]interface{}{
		"success": true,
		"message": "MCP Inspector started successfully",
		"url":     finalURL,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleInspectorStop stops the MCP Inspector
func (s *Server) handleInspectorStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.logger.Info("Received request to stop MCP Inspector")

	if err := s.inspectorManager.Stop(); err != nil {
		s.logger.Error("Failed to stop MCP Inspector", zap.Error(err))
		http.Error(w, fmt.Sprintf("Failed to stop MCP Inspector: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"success": true,
		"message": "MCP Inspector stopped successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// detectPortsViaLsof detects the inspector client port using lsof
func (im *InspectorManager) detectPortsViaLsof() (int, string) {
	if im.cmd == nil || im.cmd.Process == nil {
		return 0, ""
	}

	// Wait a bit for the process to fully start and bind to ports
	time.Sleep(2 * time.Second)

	// Get all listening ports from process tree
	pid := im.cmd.Process.Pid
	cmd := exec.Command("lsof", "-nP", "-iTCP", "-sTCP:LISTEN", "-a", "-p", fmt.Sprintf("%d", pid))
	output, err := cmd.Output()
	if err != nil {
		im.logger.Debug("Failed to run lsof for inspector PID", zap.Int("pid", pid), zap.Error(err))

		// Try to find inspector processes by name
		cmd = exec.Command("lsof", "-nP", "-iTCP", "-sTCP:LISTEN")
		output, err = cmd.Output()
		if err != nil {
			return 0, ""
		}
	}

	// Parse lsof output to find ports
	var ports []int
	lines := strings.Split(string(output), "\n")
	portRegex := regexp.MustCompile(`:(\d{4,5})\s+\(LISTEN\)`)

	for _, line := range lines {
		if strings.Contains(line, "node") && strings.Contains(line, "inspector") {
			if matches := portRegex.FindStringSubmatch(line); len(matches) > 1 {
				if port, err := strconv.Atoi(matches[1]); err == nil && port > 1024 {
					ports = append(ports, port)
				}
			}
		}
	}

	// Test each port to find the client UI
	for _, port := range ports {
		if url := im.testPort(port); url != "" {
			return port, url
		}
	}

	return 0, ""
}

// testPort tests if a port serves the inspector client UI
func (im *InspectorManager) testPort(port int) string {
	url := fmt.Sprintf("http://localhost:%d/", port)
	resp, err := http.Get(url)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	// Client UI should return 200 OK and HTML content
	if resp.StatusCode == http.StatusOK {
		contentType := resp.Header.Get("Content-Type")
		if strings.Contains(contentType, "html") ||
		   strings.Contains(contentType, "text/html") {
			// Found the client UI!
			im.logger.Debug("Found inspector client UI",
				zap.Int("port", port),
				zap.String("content_type", contentType))
			return url
		}
	}

	return ""
}

// handleInspectorStatus returns the inspector status
func (s *Server) handleInspectorStatus(w http.ResponseWriter, r *http.Request) {
	running := s.inspectorManager.IsRunning()

	response := map[string]interface{}{
		"running": running,
		"url":     s.inspectorManager.GetURL(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleOpenPath opens a file or folder path in the system default application
func (s *Server) handleOpenPath(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		Path string `json:"path"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		s.logger.Error("Failed to decode open-path request", zap.Error(err))
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Invalid request format",
		})
		return
	}

	if request.Path == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Path is required",
		})
		return
	}

	// Open the path using the system default application
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", request.Path)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", "", request.Path)
	case "linux":
		cmd = exec.Command("xdg-open", request.Path)
	default:
		s.logger.Error("Unsupported operating system", zap.String("os", runtime.GOOS))
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Unsupported operating system",
		})
		return
	}

	if err := cmd.Start(); err != nil {
		s.logger.Error("Failed to open path",
			zap.String("path", request.Path),
			zap.Error(err))
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Failed to open path: %v", err),
		})
		return
	}

	s.logger.Info("Opened path successfully",
		zap.String("path", request.Path),
		zap.String("os", runtime.GOOS))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	})
}

// findServerIndex finds the index of a server in the config file by name
// MCP Inspector requires a numeric index (0-based) not a server name
func (im *InspectorManager) findServerIndex(configPath, serverName string) (int, error) {
	// Read the config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return -1, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse the config JSON to find the server index
	var cfg struct {
		Servers []struct {
			Name string `json:"name"`
		} `json:"mcpServers"`
	}

	if err := json.Unmarshal(data, &cfg); err != nil {
		return -1, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Find the server by name and return its index
	for i, server := range cfg.Servers {
		if server.Name == serverName {
			return i, nil
		}
	}

	return -1, fmt.Errorf("server '%s' not found in config file", serverName)
}

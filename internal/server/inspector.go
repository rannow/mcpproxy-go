package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
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
	logger  *zap.SugaredLogger
}

// NewInspectorManager creates a new inspector manager
func NewInspectorManager(logger *zap.SugaredLogger) *InspectorManager {
	return &InspectorManager{
		port:   5173, // Default MCP Inspector port
		logger: logger,
	}
}

// Start starts the MCP Inspector process
func (im *InspectorManager) Start() error {
	im.mu.Lock()
	defer im.mu.Unlock()

	if im.running {
		im.logger.Info("MCP Inspector is already running")
		return nil
	}

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	im.cancel = cancel

	// Start MCP Inspector using npx
	im.cmd = exec.CommandContext(ctx, "npx", "@modelcontextprotocol/inspector")

	im.logger.Info("Starting MCP Inspector",
		zap.String("command", "npx @modelcontextprotocol/inspector"),
		zap.Int("port", im.port))

	// Start the process
	if err := im.cmd.Start(); err != nil {
		cancel()
		im.logger.Error("Failed to start MCP Inspector", zap.Error(err))
		return fmt.Errorf("failed to start MCP Inspector: %w", err)
	}

	im.running = true

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

	// Wait a bit for the inspector to start
	time.Sleep(2 * time.Second)

	im.logger.Info("MCP Inspector started successfully",
		zap.Int("port", im.port),
		zap.String("url", fmt.Sprintf("http://localhost:%d", im.port)))

	return nil
}

// Stop stops the MCP Inspector process
func (im *InspectorManager) Stop() error {
	im.mu.Lock()
	defer im.mu.Unlock()

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

// GetURL returns the inspector URL
func (im *InspectorManager) GetURL() string {
	return fmt.Sprintf("http://localhost:%d", im.port)
}

// handleInspectorStart starts the MCP Inspector and redirects to it
func (s *Server) handleInspectorStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.logger.Info("Received request to start MCP Inspector")

	// Start the inspector
	if err := s.inspectorManager.Start(); err != nil {
		s.logger.Error("Failed to start MCP Inspector", zap.Error(err))
		http.Error(w, fmt.Sprintf("Failed to start MCP Inspector: %v", err), http.StatusInternalServerError)
		return
	}

	// Return success response with redirect URL
	response := map[string]interface{}{
		"success": true,
		"message": "MCP Inspector started successfully",
		"url":     s.inspectorManager.GetURL(),
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

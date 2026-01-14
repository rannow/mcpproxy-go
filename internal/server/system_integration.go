package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"path/filepath"
	"runtime"

	"go.uber.org/zap"
)

// OpenPathRequest represents a request to open a file or directory
type OpenPathRequest struct {
	Path string `json:"path"`
}

// handleOpenPath opens a file or directory in the system's default application
func (s *Server) handleOpenPath(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req OpenPathRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.logger.Error("Failed to decode open path request", zap.Error(err))
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Path == "" {
		http.Error(w, "Path is required", http.StatusBadRequest)
		return
	}

	// Clean the path to prevent path traversal attacks
	cleanPath := filepath.Clean(req.Path)

	// Determine the command based on OS
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin": // macOS
		cmd = exec.Command("open", cleanPath)
	case "linux":
		cmd = exec.Command("xdg-open", cleanPath)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", "", cleanPath)
	default:
		s.logger.Error("Unsupported operating system", zap.String("os", runtime.GOOS))
		http.Error(w, "Unsupported operating system", http.StatusInternalServerError)
		return
	}

	// Execute the command
	if err := cmd.Start(); err != nil {
		s.logger.Error("Failed to open path",
			zap.String("path", cleanPath),
			zap.String("os", runtime.GOOS),
			zap.Error(err))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": fmt.Sprintf("Failed to open path: %v", err),
		})
		return
	}

	s.logger.Info("Successfully opened path",
		zap.String("path", cleanPath),
		zap.String("os", runtime.GOOS))

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "success",
		"path":   cleanPath,
	})
}

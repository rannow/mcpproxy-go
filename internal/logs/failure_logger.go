package logs

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// LogServerFailure writes a server failure entry to the failed_servers.log file
// This is used by the /failed-servers web UI to display servers that have been
// automatically disabled due to consecutive failures.
func LogServerFailure(dataDir, serverName, reason string) error {
	if dataDir == "" {
		dataDir = filepath.Join(os.Getenv("HOME"), ".mcpproxy")
	}

	logPath := filepath.Join(dataDir, "failed_servers.log")

	// Open file in append mode, create if it doesn't exist
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open failed_servers.log: %w", err)
	}
	defer f.Close()

	// Format: timestamp [LEVEL] Server "name" failed: reason
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	logLine := fmt.Sprintf("%s\t[ERROR]\tServer \"%s\" failed: %s\n", timestamp, serverName, reason)

	_, err = f.WriteString(logLine)
	if err != nil {
		return fmt.Errorf("failed to write to failed_servers.log: %w", err)
	}

	return nil
}

// LogServerFailureDetailed writes a detailed server failure entry with error categorization
func LogServerFailureDetailed(dataDir, serverName string, errorMsg string, failureCount int, firstFailureTime time.Time) error {
	if dataDir == "" {
		dataDir = filepath.Join(os.Getenv("HOME"), ".mcpproxy")
	}

	logPath := filepath.Join(dataDir, "failed_servers.log")

	// Categorize the error
	errorType, suggestions := categorizeError(errorMsg)

	// Open file in append mode, create if it doesn't exist
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open failed_servers.log: %w", err)
	}
	defer f.Close()

	// Enhanced format with error categorization
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	firstFailure := firstFailureTime.Format("2006-01-02 15:04:05")

	logLine := fmt.Sprintf("%s\t[ERROR]\tServer \"%s\" | Type: %s | Count: %d | First: %s | Error: %s | Suggestions: %s\n",
		timestamp, serverName, errorType, failureCount, firstFailure, errorMsg, strings.Join(suggestions, "; "))

	_, err = f.WriteString(logLine)
	if err != nil {
		return fmt.Errorf("failed to write to failed_servers.log: %w", err)
	}

	return nil
}

// categorizeError analyzes an error message and returns error type and suggestions
func categorizeError(errMsg string) (string, []string) {
	if errMsg == "" {
		return "unknown", []string{"No error details available"}
	}

	errStr := strings.ToLower(errMsg)

	// Timeout errors
	if strings.Contains(errStr, "timeout") || strings.Contains(errStr, "deadline exceeded") {
		return "timeout", []string{
			"Check if server process starts correctly",
			"Increase timeout in configuration",
			"Verify network connectivity",
		}
	}

	// Missing package errors
	if strings.Contains(errStr, "cannot find module") ||
		strings.Contains(errStr, "modulenotfounderror") ||
		strings.Contains(errStr, "command not found") ||
		strings.Contains(errStr, "no such file") ||
		strings.Contains(errStr, "enoent") {
		return "missing_package", []string{
			"Run 'npm install' or 'pip install' in working directory",
			"Verify package.json or requirements.txt exists",
			"Check if npx/uvx is installed and in PATH",
		}
	}

	// OAuth errors
	if strings.Contains(errStr, "oauth") ||
		strings.Contains(errStr, "unauthorized") ||
		strings.Contains(errStr, "401") ||
		strings.Contains(errStr, "authentication") {
		return "oauth", []string{
			"Run: mcpproxy auth login --server=<name>",
			"Check API token is valid and not expired",
			"Verify OAuth configuration in mcp_config.json",
		}
	}

	// Configuration errors
	if strings.Contains(errStr, "config") ||
		strings.Contains(errStr, "invalid") ||
		strings.Contains(errStr, "missing required") ||
		strings.Contains(errStr, "env") {
		return "config", []string{
			"Verify server configuration in mcp_config.json",
			"Check required environment variables are set",
			"Review server documentation for setup requirements",
		}
	}

	// Network errors
	if strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "network") ||
		strings.Contains(errStr, "dial tcp") ||
		strings.Contains(errStr, "no route to host") {
		return "network", []string{
			"Check server URL is correct and accessible",
			"Verify firewall settings allow connections",
			"Test network connectivity to the server",
		}
	}

	// Permission errors
	if strings.Contains(errStr, "permission denied") ||
		strings.Contains(errStr, "access denied") {
		return "permission", []string{
			"Check file/directory permissions",
			"Verify executable has correct permissions",
			"May need to run with elevated privileges",
		}
	}

	return "unknown", []string{"Check server-specific logs for details"}
}

// BackupAndClearFailureLog creates a timestamped backup of the failure log and clears it
func BackupAndClearFailureLog(dataDir string) error {
	if dataDir == "" {
		dataDir = filepath.Join(os.Getenv("HOME"), ".mcpproxy")
	}

	logPath := filepath.Join(dataDir, "failed_servers.log")

	// Check if log exists
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		// No log to backup
		return nil
	}

	// Read current log
	content, err := os.ReadFile(logPath)
	if err != nil {
		return fmt.Errorf("failed to read log for backup: %w", err)
	}

	// Only backup if there's content
	if len(content) > 0 {
		// Create backup with timestamp
		timestamp := time.Now().Format("20060102-150405")
		backupPath := filepath.Join(dataDir, fmt.Sprintf("failed_servers.backup.%s.log", timestamp))

		if err := os.WriteFile(backupPath, content, 0644); err != nil {
			return fmt.Errorf("failed to create backup: %w", err)
		}

		// Clean up old backups (keep last 5)
		if err := cleanOldBackups(dataDir, 5); err != nil {
			// Log but don't fail
			fmt.Printf("Warning: failed to clean old backups: %v\n", err)
		}
	}

	// Truncate original log (or remove if empty)
	if err := os.Truncate(logPath, 0); err != nil {
		// Try to remove instead
		if rmErr := os.Remove(logPath); rmErr != nil {
			return fmt.Errorf("failed to clear log: %w (truncate: %v, remove: %v)", err, err, rmErr)
		}
	}

	return nil
}

// cleanOldBackups removes old backup files, keeping only the most recent N backups
func cleanOldBackups(dataDir string, keepCount int) error {
	pattern := filepath.Join(dataDir, "failed_servers.backup.*.log")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("failed to list backup files: %w", err)
	}

	// Sort by modification time (oldest first)
	type fileInfo struct {
		path    string
		modTime time.Time
	}

	fileInfos := make([]fileInfo, 0, len(files))
	for _, file := range files {
		stat, err := os.Stat(file)
		if err != nil {
			continue // Skip files we can't stat
		}
		fileInfos = append(fileInfos, fileInfo{
			path:    file,
			modTime: stat.ModTime(),
		})
	}

	sort.Slice(fileInfos, func(i, j int) bool {
		return fileInfos[i].modTime.Before(fileInfos[j].modTime)
	})

	// Remove oldest backups if exceeding keepCount
	if len(fileInfos) > keepCount {
		for i := 0; i < len(fileInfos)-keepCount; i++ {
			if err := os.Remove(fileInfos[i].path); err != nil {
				fmt.Printf("Warning: failed to remove old backup %s: %v\n", fileInfos[i].path, err)
			}
		}
	}

	return nil
}

// ClearFailureLog clears the failed_servers.log file
// This can be used when manually re-enabling servers or clearing old failures
func ClearFailureLog(dataDir string) error {
	if dataDir == "" {
		dataDir = filepath.Join(os.Getenv("HOME"), ".mcpproxy")
	}

	logPath := filepath.Join(dataDir, "failed_servers.log")

	// Remove the file if it exists
	err := os.Remove(logPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to clear failed_servers.log: %w", err)
	}

	return nil
}

// RemoveServerFromFailureLog removes a specific server's entries from the log
// This is useful when a server is manually re-enabled or fixed
func RemoveServerFromFailureLog(dataDir, serverName string) error {
	if dataDir == "" {
		dataDir = filepath.Join(os.Getenv("HOME"), ".mcpproxy")
	}

	logPath := filepath.Join(dataDir, "failed_servers.log")

	// Read all lines
	content, err := os.ReadFile(logPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // File doesn't exist, nothing to do
		}
		return fmt.Errorf("failed to read failed_servers.log: %w", err)
	}

	// Filter out lines containing this server
	lines := make([]string, 0)
	for _, line := range splitLines(string(content)) {
		if line == "" {
			continue
		}
		// Skip lines that contain this server name
		if !containsServerName(line, serverName) {
			lines = append(lines, line+"\n")
		}
	}

	// Write back the filtered content
	err = os.WriteFile(logPath, []byte(joinLines(lines)), 0644)
	if err != nil {
		return fmt.Errorf("failed to write filtered log: %w", err)
	}

	return nil
}

// Helper functions
func splitLines(s string) []string {
	var lines []string
	start := 0
	for i, c := range s {
		if c == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func joinLines(lines []string) string {
	result := ""
	for _, line := range lines {
		result += line
	}
	return result
}

func containsServerName(line, serverName string) bool {
	// Check if the line contains the server name in quotes
	searchStr := fmt.Sprintf("\"%s\"", serverName)
	return contains(line, searchStr)
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && indexSubstring(s, substr) >= 0
}

func indexSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			if s[i+j] != substr[j] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}

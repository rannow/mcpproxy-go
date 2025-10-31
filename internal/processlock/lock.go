package processlock

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"go.uber.org/zap"
)

const (
	defaultPIDFile = "mcpproxy.pid"
)

// ProcessLock manages process-level locking to ensure only one instance runs
type ProcessLock struct {
	pidFile string
	logger  *zap.Logger
}

// New creates a new ProcessLock instance
func New(dataDir string, logger *zap.Logger) *ProcessLock {
	return &ProcessLock{
		pidFile: filepath.Join(dataDir, defaultPIDFile),
		logger:  logger,
	}
}

// Acquire attempts to acquire the process lock
func (p *ProcessLock) Acquire(listenAddr string) error {
	// Check if port is already in use
	if err := p.checkPort(listenAddr); err != nil {
		return fmt.Errorf("port check failed: %w", err)
	}

	// Check if PID file exists
	if _, err := os.Stat(p.pidFile); err == nil {
		// PID file exists, check if process is running
		pid, err := p.readPID()
		if err != nil {
			p.logger.Warn("Failed to read PID file, removing stale lock",
				zap.String("pid_file", p.pidFile),
				zap.Error(err))
			os.Remove(p.pidFile)
		} else if p.isProcessRunning(pid) {
			return fmt.Errorf("another mcpproxy instance is already running (PID: %d)", pid)
		} else {
			p.logger.Warn("Removing stale PID file from dead process",
				zap.Int("pid", pid),
				zap.String("pid_file", p.pidFile))
			os.Remove(p.pidFile)
		}
	}

	// Write current PID
	if err := p.writePID(); err != nil {
		return fmt.Errorf("failed to write PID file: %w", err)
	}

	p.logger.Info("Process lock acquired",
		zap.Int("pid", os.Getpid()),
		zap.String("pid_file", p.pidFile))

	return nil
}

// Release releases the process lock
func (p *ProcessLock) Release() error {
	if err := os.Remove(p.pidFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove PID file: %w", err)
	}

	p.logger.Info("Process lock released",
		zap.Int("pid", os.Getpid()),
		zap.String("pid_file", p.pidFile))

	return nil
}

// checkPort checks if the listen address port is already in use
func (p *ProcessLock) checkPort(listenAddr string) error {
	// Parse listen address
	host, port, err := net.SplitHostPort(listenAddr)
	if err != nil {
		// If no port specified, assume default format
		if !strings.Contains(listenAddr, ":") {
			return fmt.Errorf("invalid listen address format: %s", listenAddr)
		}
		return fmt.Errorf("failed to parse listen address: %w", err)
	}

	// Try to listen on the port
	addr := net.JoinHostPort(host, port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		// Port is in use
		return fmt.Errorf("port %s is already in use by another process", addr)
	}
	listener.Close()

	return nil
}

// readPID reads the PID from the PID file
func (p *ProcessLock) readPID() (int, error) {
	data, err := os.ReadFile(p.pidFile)
	if err != nil {
		return 0, err
	}

	pidStr := strings.TrimSpace(string(data))
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		return 0, fmt.Errorf("invalid PID in file: %s", pidStr)
	}

	return pid, nil
}

// writePID writes the current PID to the PID file
func (p *ProcessLock) writePID() error {
	pid := os.Getpid()
	return os.WriteFile(p.pidFile, []byte(fmt.Sprintf("%d\n", pid)), 0644)
}

// isProcessRunning checks if a process with the given PID is running
func (p *ProcessLock) isProcessRunning(pid int) bool {
	// Send signal 0 to check if process exists
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// On Unix systems, FindProcess always succeeds, so we need to send a signal
	err = process.Signal(syscall.Signal(0))
	if err != nil {
		// Process doesn't exist or we don't have permission to signal it
		return false
	}

	return true
}

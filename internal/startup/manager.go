package startup

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"go.uber.org/zap"
	"mcpproxy-go/internal/config"
)

// Manager manages the lifecycle of an optional startup script/command
type Manager struct {
	mu     sync.Mutex
	cmd    *exec.Cmd
	cfg    *config.StartupScriptConfig
	log    *zap.SugaredLogger
	start  time.Time
}

func NewManager(cfg *config.StartupScriptConfig, logger *zap.SugaredLogger) *Manager {
	return &Manager{cfg: cfg, log: logger}
}

// UpdateConfig updates the runtime configuration. Caller persists to disk.
func (m *Manager) UpdateConfig(cfg *config.StartupScriptConfig) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cfg = cfg
}

// Start launches the configured script/command if enabled and not already running.
func (m *Manager) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.cfg == nil || !m.cfg.Enabled {
		return nil
	}
	if m.cmd != nil {
		return fmt.Errorf("startup script already running")
	}
	if m.cfg.Path == "" {
		return fmt.Errorf("startup script path/command not configured")
	}

	// Build command: shell -c "<Path>" plus optional args
	shell := m.cfg.Shell
	if strings.TrimSpace(shell) == "" {
		shell = "/bin/bash"
	}

	args := []string{"-c", m.cfg.Path}
	if len(m.cfg.Args) > 0 {
		args = append(args, m.cfg.Args...)
	}

	cmd := exec.CommandContext(ctx, shell, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()
	for k, v := range m.cfg.Env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}
	if m.cfg.WorkingDir != "" {
		cmd.Dir = m.cfg.WorkingDir
	}

	// Ensure we can kill all children on Unix by creating a new process group
	if runtime.GOOS != "windows" {
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	// Handle optional timeout
	if m.cfg.Timeout.Duration() > 0 {
		go func(timeout time.Duration, c *exec.Cmd) {
			select {
			case <-time.After(timeout):
				m.log.Warn("Startup script timeout reached, stopping")
				_ = m.killProcessTree(c)
			case <-ctx.Done():
				// Context cancelled elsewhere
			}
		}(m.cfg.Timeout.Duration(), cmd)
	}

	m.cmd = cmd
	m.start = time.Now()
	m.log.Info("Startup script started",
		zap.String("shell", shell),
		zap.String("command", m.cfg.Path),
		zap.String("dir", cmd.Dir))

	// Reap when finished
	go func(c *exec.Cmd) {
		_ = c.Wait()
		m.mu.Lock()
		defer m.mu.Unlock()
		m.cmd = nil
	}(cmd)

	return nil
}

// Stop terminates the startup script and its subprocesses.
func (m *Manager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.cmd == nil {
		return nil
	}
	return m.killProcessTree(m.cmd)
}

// Restart stops (if running) and starts again.
func (m *Manager) Restart(ctx context.Context) error {
	if err := m.Stop(); err != nil {
		return err
	}
	return m.Start(ctx)
}

// Status returns process status info
func (m *Manager) Status() map[string]interface{} {
	m.mu.Lock()
	defer m.mu.Unlock()

	running := m.cmd != nil
	pid := 0
	if running && m.cmd.Process != nil {
		pid = m.cmd.Process.Pid
	}

	return map[string]interface{}{
		"enabled":  m.cfg != nil && m.cfg.Enabled,
		"running":  running,
		"pid":      pid,
		"path":     func() string { if m.cfg!=nil {return m.cfg.Path}; return "" }(),
		"shell":    func() string { if m.cfg!=nil {return m.cfg.Shell}; return "" }(),
		"since":    m.start,
	}
}

// killProcessTree attempts to kill the whole process tree for the command
func (m *Manager) killProcessTree(c *exec.Cmd) error {
	if c == nil || c.Process == nil {
		return nil
	}
	// Windows: use taskkill to terminate tree
	if runtime.GOOS == "windows" {
		// Best-effort: taskkill /T /F /PID <pid>
		_ = exec.Command("taskkill", "/T", "/F", "/PID", fmt.Sprintf("%d", c.Process.Pid)).Run()
		return c.Process.Kill()
	}
	// Unix: kill process group
	pgid, err := syscall.Getpgid(c.Process.Pid)
	if err != nil {
		// Fallback to direct kill
		_ = c.Process.Kill()
		return err
	}
	// Negative pgid sends to the group
	if err := syscall.Kill(-pgid, syscall.SIGKILL); err != nil {
		// Fallback
		_ = c.Process.Kill()
		return err
	}
	return nil
}

// ValidateConfig performs basic checks
func ValidateConfig(cfg *config.StartupScriptConfig) error {
	if cfg == nil {
		return errors.New("config is nil")
	}
	if cfg.Enabled && strings.TrimSpace(cfg.Path) == "" {
		return fmt.Errorf("startup script path is required when enabled")
	}
	return nil
}



//go:build unix || darwin || linux

package startup

import (
	"fmt"
	"os/exec"
	"syscall"
)

// setupProcessGroup configures the process to create its own process group
func setupProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

// killProcessGroup kills the entire process group
func killProcessGroup(cmd *exec.Cmd) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}

	// Get process group ID
	pgid, err := syscall.Getpgid(cmd.Process.Pid)
	if err != nil {
		// Fallback to direct kill
		_ = cmd.Process.Kill()
		return fmt.Errorf("failed to get process group: %w", err)
	}

	// Negative pgid sends signal to the entire process group
	if err := syscall.Kill(-pgid, syscall.SIGKILL); err != nil {
		// Fallback to direct kill
		_ = cmd.Process.Kill()
		return fmt.Errorf("failed to kill process group: %w", err)
	}

	return nil
}

//go:build windows

package startup

import (
	"fmt"
	"os/exec"
)

// setupProcessGroup is a no-op on Windows (process groups are handled differently)
func setupProcessGroup(cmd *exec.Cmd) {
	// No-op on Windows
}

// killProcessGroup uses taskkill to terminate the process tree on Windows
func killProcessGroup(cmd *exec.Cmd) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}

	// Use taskkill with /T (tree) flag to kill process and all children
	killCmd := exec.Command("taskkill", "/T", "/F", "/PID", fmt.Sprintf("%d", cmd.Process.Pid))
	if err := killCmd.Run(); err != nil {
		// Fallback to direct kill
		_ = cmd.Process.Kill()
		return fmt.Errorf("taskkill failed: %w", err)
	}

	return nil
}

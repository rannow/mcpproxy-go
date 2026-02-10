// Package types provides type definitions for upstream connection management.
package types

// GlobalStatus represents the overall system status of the proxy.
// It tracks the lifecycle of all server connections as a unified state.
type GlobalStatus string

const (
	// StatusStopped - all servers disconnected
	StatusStopped GlobalStatus = "stopped"
	// StatusStarting - servers are being connected
	StatusStarting GlobalStatus = "starting"
	// StatusTesting - diagnostic agent is testing servers
	StatusTesting GlobalStatus = "testing"
	// StatusRunning - all active servers connected
	StatusRunning GlobalStatus = "running"
	// StatusStopping - servers are being disconnected
	StatusStopping GlobalStatus = "stopping"
)

// String returns the string representation of the global status.
func (g GlobalStatus) String() string {
	return string(g)
}

// DisplayString returns a human-readable representation of the global status.
func (g GlobalStatus) DisplayString() string {
	switch g {
	case StatusStopped:
		return "Stopped"
	case StatusStarting:
		return "Starting"
	case StatusTesting:
		return "Testing"
	case StatusRunning:
		return "Running"
	case StatusStopping:
		return "Stopping"
	default:
		return "Unknown"
	}
}

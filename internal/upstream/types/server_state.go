// Package types provides type definitions for upstream connection management.
// MED-001: ServerState type extracted from types.go for better separation of concerns.
package types

import "fmt"

// ServerState represents the persisted configuration state of a server
// This aligns with the startup_mode field in ServerConfig and is stored in the database
//
// State Transitions:
//
// Normal flow (user-initiated):
//
//	active <-> disabled        (user enable/disable via UI/API)
//	active <-> lazy_loading    (user changes startup mode)
//	lazy_loading <-> disabled  (user enable/disable)
//
// Quarantine flow (security-triggered):
//
//	[any state] -> quarantined (automatic on security detection)
//	quarantined -> [original]  (manual approval via UI only)
//
// Auto-disable flow (failure-triggered):
//
//	active -> auto_disabled    (automatic after connection failures)
//	auto_disabled -> active    (manual re-enable or group enable)
//	lazy_loading -> auto_disabled (automatic after connection failures)
//	auto_disabled -> lazy_loading (manual re-enable with lazy mode)
//
// Group operations:
//
//	Group enable: clears auto_disabled for all servers in group -> attempts reconnection
//	Group disable: sets all servers to disabled state
//
// Stability guarantees:
//   - Stable states (active, disabled, lazy_loading): Won't change automatically
//   - Unstable states (quarantined, auto_disabled): Can be cleared or changed
type ServerState string

const (
	// DefaultAutoDisableThreshold is the default number of consecutive failures
	// before a server is automatically disabled. HIGH-003: Consolidated to single constant.
	// Increased from 5 to 7 to reduce false positives with slow-starting NPX servers.
	DefaultAutoDisableThreshold = 7

	// StateActive - server starts immediately on boot
	StateActive ServerState = "active"
	// StateDisabledConfig - server is disabled by user configuration
	StateDisabledConfig ServerState = "disabled"
	// StateQuarantined - server is quarantined for security reasons
	StateQuarantined ServerState = "quarantined"
	// StateAutoDisabled - server was automatically disabled after repeated failures
	StateAutoDisabled ServerState = "auto_disabled"
	// StateLazyLoading - server enabled but doesn't start on boot (lazy loaded)
	StateLazyLoading ServerState = "lazy_loading"
)

// String returns the string representation of the server state (technical/API format)
// MED-004: Consistent with API serialization format (snake_case)
func (s ServerState) String() string {
	return string(s)
}

// DisplayString returns a human-readable representation of the server state
// MED-004: For UI display purposes (Title Case with proper spacing)
func (s ServerState) DisplayString() string {
	switch s {
	case StateActive:
		return "Active"
	case StateDisabledConfig:
		return "Disabled"
	case StateQuarantined:
		return "Quarantined"
	case StateAutoDisabled:
		return "Auto-Disabled"
	case StateLazyLoading:
		return "Lazy Loading"
	default:
		return "Unknown"
	}
}

// IsStable returns true if the state represents a stable configuration state
// Stable states don't change automatically - they require user or system intervention
func (s ServerState) IsStable() bool {
	switch s {
	case StateActive, StateDisabledConfig, StateLazyLoading:
		return true
	case StateQuarantined, StateAutoDisabled:
		return false // These can be cleared/changed
	default:
		return false
	}
}

// IsEnabled returns true if the server is enabled (active or lazy loading)
func (s ServerState) IsEnabled() bool {
	return s == StateActive || s == StateLazyLoading
}

// IsDisabled returns true if the server is disabled in any form
func (s ServerState) IsDisabled() bool {
	return s == StateDisabledConfig || s == StateQuarantined || s == StateAutoDisabled
}

// ValidateServerState validates that a server state string is valid
func ValidateServerState(state string) error {
	validStates := map[string]bool{
		string(StateActive):         true,
		string(StateDisabledConfig): true,
		string(StateQuarantined):    true,
		string(StateAutoDisabled):   true,
		string(StateLazyLoading):    true,
	}

	if !validStates[state] {
		return fmt.Errorf("invalid server state: %s (must be one of: active, disabled, quarantined, auto_disabled, lazy_loading)", state)
	}

	return nil
}

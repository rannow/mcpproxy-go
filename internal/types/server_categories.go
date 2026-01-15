// Package types provides shared types used across multiple packages
package types

import "time"

// ServerStatusCategories represents servers grouped by their status
// This is the SINGLE SOURCE OF TRUTH structure for server categorization
// Both API and Tray MUST use this structure to ensure consistency
//
// Categorization priority (in order):
// 1. Quarantined (startup_mode == "quarantined")
// 2. Auto-Disabled (startup_mode == "auto_disabled")
// 3. Disabled (startup_mode == "disabled")
// 4. Sleeping (startup_mode == "lazy_loading" AND connection_state != "Ready")
// 5. Connected (NOT disabled/quarantined/auto_disabled AND connection_state == "Ready")
// 6. Disconnected (NOT disabled/quarantined/auto_disabled AND connection_state != "Ready")
type ServerStatusCategories struct {
	Connected    []string  `json:"connected"`     // Connected Servers
	Disconnected []string  `json:"disconnected"`  // Disconnected Servers
	Sleeping     []string  `json:"sleeping"`      // Sleeping Servers
	Disabled     []string  `json:"disabled"`      // Disabled Servers
	AutoDisabled []string  `json:"auto_disabled"` // Auto-Disabled Servers
	Quarantined  []string  `json:"quarantined"`   // Quarantined Servers
	Total        int       `json:"total"`
	Timestamp    time.Time `json:"timestamp"`
}

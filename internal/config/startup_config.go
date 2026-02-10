// Package config provides configuration types and utilities for mcpproxy.
// StartupConfig centralizes all startup-related timing configuration.
package config

import "time"

// StartupConfig contains all configurable timing settings for the startup process.
// All values are in milliseconds for JSON compatibility.
// The startup process uses individual sequential server startup (ONE at a time).
type StartupConfig struct {
	// InterServerDelayMs is the delay between starting each server (default: 5000ms = 5s)
	InterServerDelayMs int `json:"inter_server_delay_ms" mapstructure:"inter_server_delay_ms"`

	// DefaultTimeoutMs is the timeout for servers that haven't been tested yet (default: 60000ms = 60s)
	DefaultTimeoutMs int `json:"default_timeout_ms" mapstructure:"default_timeout_ms"`

	// TestedTimeoutBufferMs is added to the measured startup time for tested servers (default: 30000ms = 30s)
	TestedTimeoutBufferMs int `json:"tested_timeout_buffer_ms" mapstructure:"tested_timeout_buffer_ms"`

	// MaxRetries is the maximum number of retry attempts before auto-disabling (default: 5)
	MaxRetries int `json:"max_retries" mapstructure:"max_retries"`

	// RetryDelayMs is the FIXED delay between retry attempts - NO exponential backoff (default: 10000ms = 10s)
	RetryDelayMs int `json:"retry_delay_ms" mapstructure:"retry_delay_ms"`

	// InactivitySleepMs is the inactivity timeout before putting a server to sleep (default: 600000ms = 10min)
	InactivitySleepMs int `json:"inactivity_sleep_ms" mapstructure:"inactivity_sleep_ms"`

	// KeepAliveIntervalMs is the interval for keep-alive pings (default: 30000ms = 30s)
	KeepAliveIntervalMs int `json:"keepalive_interval_ms" mapstructure:"keepalive_interval_ms"`

	// WakeTimeoutMs is the timeout for waking a sleeping server (default: 30000ms = 30s)
	WakeTimeoutMs int `json:"wake_timeout_ms" mapstructure:"wake_timeout_ms"`
}

// Default values for StartupConfig
const (
	DefaultInterServerDelayMs    = 5000   // 5 seconds
	DefaultDefaultTimeoutMs      = 60000  // 60 seconds
	DefaultTestedTimeoutBufferMs = 30000  // 30 seconds
	DefaultMaxRetries            = 5
	DefaultRetryDelayMs          = 10000  // 10 seconds (FIXED, no backoff)
	DefaultInactivitySleepMs     = 600000 // 10 minutes
	DefaultKeepAliveIntervalMs   = 30000  // 30 seconds
	DefaultWakeTimeoutMs         = 30000  // 30 seconds
)

// DefaultStartupConfig returns a StartupConfig with default values
func DefaultStartupConfig() StartupConfig {
	return StartupConfig{
		InterServerDelayMs:    DefaultInterServerDelayMs,
		DefaultTimeoutMs:      DefaultDefaultTimeoutMs,
		TestedTimeoutBufferMs: DefaultTestedTimeoutBufferMs,
		MaxRetries:            DefaultMaxRetries,
		RetryDelayMs:          DefaultRetryDelayMs,
		InactivitySleepMs:     DefaultInactivitySleepMs,
		KeepAliveIntervalMs:   DefaultKeepAliveIntervalMs,
		WakeTimeoutMs:         DefaultWakeTimeoutMs,
	}
}

// InterServerDelay returns the inter-server delay as a time.Duration
func (c *StartupConfig) InterServerDelay() time.Duration {
	return time.Duration(c.InterServerDelayMs) * time.Millisecond
}

// DefaultTimeout returns the default timeout as a time.Duration
func (c *StartupConfig) DefaultTimeout() time.Duration {
	return time.Duration(c.DefaultTimeoutMs) * time.Millisecond
}

// TestedTimeoutBuffer returns the tested timeout buffer as a time.Duration
func (c *StartupConfig) TestedTimeoutBuffer() time.Duration {
	return time.Duration(c.TestedTimeoutBufferMs) * time.Millisecond
}

// RetryDelay returns the retry delay as a time.Duration
func (c *StartupConfig) RetryDelay() time.Duration {
	return time.Duration(c.RetryDelayMs) * time.Millisecond
}

// InactivitySleep returns the inactivity sleep timeout as a time.Duration
func (c *StartupConfig) InactivitySleep() time.Duration {
	return time.Duration(c.InactivitySleepMs) * time.Millisecond
}

// KeepAliveInterval returns the keep-alive interval as a time.Duration
func (c *StartupConfig) KeepAliveInterval() time.Duration {
	return time.Duration(c.KeepAliveIntervalMs) * time.Millisecond
}

// WakeTimeout returns the wake timeout as a time.Duration
func (c *StartupConfig) WakeTimeout() time.Duration {
	return time.Duration(c.WakeTimeoutMs) * time.Millisecond
}

// CalculateTimeout returns the appropriate timeout for a server based on whether it has been tested.
// If startupTested is true, uses startupTimeMs + TestedTimeoutBuffer.
// Otherwise, uses DefaultTimeout.
func (c *StartupConfig) CalculateTimeout(startupTested bool, startupTimeMs int) time.Duration {
	if startupTested && startupTimeMs > 0 {
		return time.Duration(startupTimeMs)*time.Millisecond + c.TestedTimeoutBuffer()
	}
	return c.DefaultTimeout()
}

// Validate validates the StartupConfig and sets defaults for zero values
func (c *StartupConfig) Validate() {
	if c.InterServerDelayMs <= 0 {
		c.InterServerDelayMs = DefaultInterServerDelayMs
	}
	if c.DefaultTimeoutMs <= 0 {
		c.DefaultTimeoutMs = DefaultDefaultTimeoutMs
	}
	if c.TestedTimeoutBufferMs <= 0 {
		c.TestedTimeoutBufferMs = DefaultTestedTimeoutBufferMs
	}
	if c.MaxRetries <= 0 {
		c.MaxRetries = DefaultMaxRetries
	}
	if c.RetryDelayMs <= 0 {
		c.RetryDelayMs = DefaultRetryDelayMs
	}
	if c.InactivitySleepMs <= 0 {
		c.InactivitySleepMs = DefaultInactivitySleepMs
	}
	if c.KeepAliveIntervalMs <= 0 {
		c.KeepAliveIntervalMs = DefaultKeepAliveIntervalMs
	}
	if c.WakeTimeoutMs <= 0 {
		c.WakeTimeoutMs = DefaultWakeTimeoutMs
	}
}

package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestServerState_String(t *testing.T) {
	tests := []struct {
		name  string
		state ServerState
		want  string
	}{
		{"active", StateActive, "active"},
		{"disabled", StateDisabledConfig, "disabled"},
		{"quarantined", StateQuarantined, "quarantined"},
		{"auto_disabled", StateAutoDisabled, "auto_disabled"},
		{"lazy_loading", StateLazyLoading, "lazy_loading"},
		{"empty", ServerState(""), ""},
		{"unknown", ServerState("unknown"), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.state.String())
		})
	}
}

func TestServerState_IsStable(t *testing.T) {
	tests := []struct {
		name   string
		state  ServerState
		stable bool
	}{
		{"active is stable", StateActive, true},
		{"disabled is stable", StateDisabledConfig, true},
		{"lazy_loading is stable", StateLazyLoading, true},
		{"quarantined is not stable", StateQuarantined, false},
		{"auto_disabled is not stable", StateAutoDisabled, false},
		{"empty is not stable", ServerState(""), false},
		{"unknown is not stable", ServerState("unknown"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.stable, tt.state.IsStable(),
				"state %s: expected IsStable()=%v", tt.state, tt.stable)
		})
	}
}

func TestServerState_IsEnabled(t *testing.T) {
	tests := []struct {
		name    string
		state   ServerState
		enabled bool
	}{
		{"active is enabled", StateActive, true},
		{"lazy_loading is enabled", StateLazyLoading, true},
		{"disabled is not enabled", StateDisabledConfig, false},
		{"quarantined is not enabled", StateQuarantined, false},
		{"auto_disabled is not enabled", StateAutoDisabled, false},
		{"empty is not enabled", ServerState(""), false},
		{"unknown is not enabled", ServerState("unknown"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.enabled, tt.state.IsEnabled(),
				"state %s: expected IsEnabled()=%v", tt.state, tt.enabled)
		})
	}
}

func TestServerState_IsDisabled(t *testing.T) {
	tests := []struct {
		name     string
		state    ServerState
		disabled bool
	}{
		{"active is not disabled", StateActive, false},
		{"lazy_loading is not disabled", StateLazyLoading, false},
		{"disabled is disabled", StateDisabledConfig, true},
		{"quarantined is disabled", StateQuarantined, true},
		{"auto_disabled is disabled", StateAutoDisabled, true},
		{"empty is not disabled", ServerState(""), false},
		{"unknown is not disabled", ServerState("unknown"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.disabled, tt.state.IsDisabled(),
				"state %s: expected IsDisabled()=%v", tt.state, tt.disabled)
		})
	}
}

func TestValidateServerState(t *testing.T) {
	tests := []struct {
		name    string
		state   string
		wantErr bool
	}{
		{"valid active", "active", false},
		{"valid disabled", "disabled", false},
		{"valid quarantined", "quarantined", false},
		{"valid auto_disabled", "auto_disabled", false},
		{"valid lazy_loading", "lazy_loading", false},
		{"empty is invalid", "", true},
		{"unknown is invalid", "unknown", true},
		{"uppercase is invalid", "ACTIVE", true},
		{"mixed case is invalid", "Active", true},
		{"whitespace is invalid", " active ", true},
		{"partial match is invalid", "act", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateServerState(tt.state)
			if tt.wantErr {
				assert.Error(t, err, "state %q should be invalid", tt.state)
				assert.Contains(t, err.Error(), "invalid server state",
					"error message should mention invalid state")
			} else {
				assert.NoError(t, err, "state %q should be valid", tt.state)
			}
		})
	}
}

func TestServerState_Consistency(t *testing.T) {
	// Test that IsEnabled() and IsDisabled() are mutually exclusive for valid states
	validStates := []ServerState{
		StateActive,
		StateDisabledConfig,
		StateQuarantined,
		StateAutoDisabled,
		StateLazyLoading,
	}

	for _, state := range validStates {
		t.Run(string(state), func(t *testing.T) {
			enabled := state.IsEnabled()
			disabled := state.IsDisabled()

			// For valid states, exactly one should be true
			assert.NotEqual(t, enabled, disabled,
				"state %s: IsEnabled()=%v and IsDisabled()=%v should be mutually exclusive",
				state, enabled, disabled)
		})
	}
}

func TestServerState_StableVsEnabled(t *testing.T) {
	// Test relationship between IsStable() and IsEnabled()
	// Stable states should be either all enabled or all disabled
	// Non-stable states (quarantined, auto_disabled) should be disabled

	tests := []struct {
		state   ServerState
		stable  bool
		enabled bool
	}{
		{StateActive, true, true},
		{StateDisabledConfig, true, false},
		{StateLazyLoading, true, true},
		{StateQuarantined, false, false},
		{StateAutoDisabled, false, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.state), func(t *testing.T) {
			assert.Equal(t, tt.stable, tt.state.IsStable(),
				"state %s stability mismatch", tt.state)
			assert.Equal(t, tt.enabled, tt.state.IsEnabled(),
				"state %s enabled mismatch", tt.state)

			// Non-stable states should always be disabled
			if !tt.stable {
				assert.False(t, tt.state.IsEnabled(),
					"non-stable state %s should be disabled", tt.state)
				assert.True(t, tt.state.IsDisabled(),
					"non-stable state %s should report as disabled", tt.state)
			}
		})
	}
}

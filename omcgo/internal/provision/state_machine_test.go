package provision

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateTransition(t *testing.T) {
	tests := []struct {
		name    string
		current ProvisioningState
		target  ProvisioningState
		wantErr bool
	}{
		// Happy path.
		{"discovered -> identifying", StateDiscovered, StateIdentifying, false},
		{"identifying -> matching", StateIdentifying, StateMatching, false},
		{"matching -> configuring", StateMatching, StateConfiguring, false},
		{"matching -> discovering", StateMatching, StateDiscovering, false},
		{"matching -> syncing", StateMatching, StateSyncing, false},
		{"configuring -> verifying", StateConfiguring, StateVerifying, false},
		{"verifying -> completed", StateVerifying, StateCompleted, false},

		// Failure transitions.
		{"discovered -> failed", StateDiscovered, StateFailed, false},
		{"identifying -> failed", StateIdentifying, StateFailed, false},
		{"matching -> failed", StateMatching, StateFailed, false},
		{"configuring -> failed", StateConfiguring, StateFailed, false},
		{"verifying -> failed", StateVerifying, StateFailed, false},

		// Invalid transitions.
		{"discovered -> completed", StateDiscovered, StateCompleted, true},
		{"discovered -> configuring", StateDiscovered, StateConfiguring, true},
		{"completed -> identifying", StateCompleted, StateIdentifying, true},
		{"completed -> failed", StateCompleted, StateFailed, true},
		{"failed -> identifying", StateFailed, StateIdentifying, true},
		{"verifying -> matching", StateVerifying, StateMatching, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTransition(tt.current, tt.target)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestIsTerminal(t *testing.T) {
	assert.True(t, IsTerminal(StateCompleted))
	assert.True(t, IsTerminal(StateFailed))
	assert.False(t, IsTerminal(StateDiscovered))
	assert.False(t, IsTerminal(StateConfiguring))
}

func TestNextState(t *testing.T) {
	tests := []struct {
		current ProvisioningState
		want    ProvisioningState
	}{
		{StateDiscovered, StateIdentifying},
		{StateIdentifying, StateMatching},
		{StateMatching, StateConfiguring},
		{StateConfiguring, StateVerifying},
		{StateVerifying, StateCompleted},
		{StateCompleted, StateCompleted},
		{StateFailed, StateFailed},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.want, NextState(tt.current), "NextState(%s)", tt.current)
	}
}

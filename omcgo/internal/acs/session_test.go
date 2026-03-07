package acs

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSession_ValidTransitions(t *testing.T) {
	tests := []struct {
		from SessionState
		to   SessionState
		ok   bool
	}{
		{StateIdle, StateInformReceived, true},
		{StateInformReceived, StateProcessing, true},
		{StateInformReceived, StateComplete, true},
		{StateProcessing, StateRPCPending, true},
		{StateProcessing, StateComplete, true},
		{StateRPCPending, StateRPCResponse, true},
		{StateRPCResponse, StateProcessing, true},
		{StateRPCResponse, StateComplete, true},
		// Invalid
		{StateIdle, StateComplete, false},
		{StateInformReceived, StateRPCPending, false},
		{StateComplete, StateProcessing, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.from)+"->"+string(tt.to), func(t *testing.T) {
			s := &Session{State: tt.from}
			err := s.TransitionTo(tt.to)
			if tt.ok {
				assert.NoError(t, err)
				assert.Equal(t, tt.to, s.State)
			} else {
				assert.Error(t, err)
				assert.Equal(t, tt.from, s.State)
			}
		})
	}
}

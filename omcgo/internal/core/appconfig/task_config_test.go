package appconfig

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestTaskConfig_EffectiveTerminalRedisTTL(t *testing.T) {
	tests := []struct {
		name string
		raw  time.Duration
		want time.Duration
	}{
		{name: "unset uses production default", raw: 0, want: 15 * time.Minute},
		{name: "negative uses production default", raw: -time.Minute, want: 15 * time.Minute},
		{name: "unsafe value is clamped", raw: time.Minute, want: 10 * time.Minute},
		{name: "configured value is honored", raw: 20 * time.Minute, want: 20 * time.Minute},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := TaskConfig{TerminalRedisTTL: tc.raw}
			require.Equal(t, tc.want, cfg.EffectiveTerminalRedisTTL())
		})
	}
}

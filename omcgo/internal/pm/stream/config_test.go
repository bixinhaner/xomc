package stream

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestConfigFromEnv(t *testing.T) {
	t.Setenv("PM_AGGREGATION_ENABLED", "false")
	t.Setenv("PM_AGGREGATION_CLOSE_GRACE", "7m")
	t.Setenv("PM_AGGREGATION_CONSUMER_CONCURRENCY", "12")
	t.Setenv("PM_AGGREGATION_WINDOW_TTL", "50h")

	cfg := ConfigFromEnv()
	require.False(t, cfg.Enabled)
	require.Equal(t, 7*time.Minute, cfg.CloseGrace)
	require.Equal(t, 12, cfg.ConsumerConcurrency)
	require.Equal(t, 50*time.Hour, cfg.WindowTTL)
}

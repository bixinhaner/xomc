package stream

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestDefaultConfigUsesTwelveMinuteHourlyCloseGrace(t *testing.T) {
	cfg := DefaultConfig()

	require.Equal(t, 12*time.Minute, cfg.CloseGrace)
	require.Equal(t, 15*time.Minute, cfg.DailyCloseGrace)
	require.Equal(t, 30*time.Minute, cfg.WeeklyCloseGrace)
	require.Equal(t, 30*time.Minute, cfg.MonthlyCloseGrace)
	require.Equal(t, 32, cfg.FinalizeConcurrency)
	require.Equal(t, 24*time.Hour, cfg.OutboxRetention)
	require.Equal(t, 45*24*time.Hour, cfg.ReplayRetention)
	require.Equal(t, 500, cfg.CleanupBatch)
	require.Equal(t, time.Hour, cfg.CleanupInterval)
	require.Equal(t, 30*time.Second, cfg.CleanupMaxDuration)
	require.False(t, cfg.CleanupVacuum)
	require.True(t, cfg.RedisV2WriteEnabled,
		"new releases must use compact v2 Redis state by default")
}

func TestConfigFromEnvBoundsRuntimeCleanup(t *testing.T) {
	t.Setenv("PM_AGGREGATION_CLEANUP_BATCH", "999999")
	t.Setenv("PM_AGGREGATION_CLEANUP_INTERVAL", "1s")
	t.Setenv("PM_AGGREGATION_CLEANUP_MAX_DURATION", "24h")
	t.Setenv("PM_AGGREGATION_CLEANUP_VACUUM", "true")

	cfg := ConfigFromEnv()

	require.Equal(t, maximumCleanupBatch, cfg.CleanupBatch)
	require.Equal(t, time.Minute, cfg.CleanupInterval)
	require.Equal(t, 5*time.Minute, cfg.CleanupMaxDuration)
	require.True(t, cfg.CleanupVacuum)
}

func TestConfigFromEnvCanDisableRedisV2WritesForRollback(t *testing.T) {
	t.Setenv("PM_AGGREGATION_REDIS_V2_WRITE_ENABLED", "false")

	cfg := ConfigFromEnv()

	require.False(t, cfg.RedisV2WriteEnabled)
}

func TestConfigFromEnv(t *testing.T) {
	t.Setenv("PM_AGGREGATION_ENABLED", "false")
	t.Setenv("PM_AGGREGATION_CLOSE_GRACE", "7m")
	t.Setenv("PM_AGGREGATION_CONSUMER_CONCURRENCY", "12")
	t.Setenv("PM_AGGREGATION_WINDOW_TTL", "50h")
	t.Setenv("PM_AGGREGATION_OUTBOX_RETENTION", "30m")
	t.Setenv("PM_AGGREGATION_REPLAY_RETENTION", "2880h")
	t.Setenv("PM_AGGREGATION_REDIS_V2_WRITE_ENABLED", "true")

	cfg := ConfigFromEnv()
	require.False(t, cfg.Enabled)
	require.Equal(t, 12*time.Minute, cfg.CloseGrace)
	require.Equal(t, 12, cfg.ConsumerConcurrency)
	require.Equal(t, 50*time.Hour, cfg.WindowTTL)
	require.Equal(t, time.Hour, cfg.OutboxRetention)
	require.Equal(t, 90*24*time.Hour, cfg.ReplayRetention)
	require.True(t, cfg.RedisV2WriteEnabled)
}

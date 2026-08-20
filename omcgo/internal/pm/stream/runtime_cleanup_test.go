package stream

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRuntimeCleanerBoundsDirectConfiguration(t *testing.T) {
	cleaner := NewRuntimeCleaner(nil, nil, nil, nil).SetConfig(RuntimeCleanupConfig{
		BatchSize: 999999, Interval: time.Second,
		MaxDuration: 24 * time.Hour, VacuumEnabled: true,
	})

	require.Equal(t, maximumCleanupBatch, cleaner.config.BatchSize)
	require.Equal(t, time.Minute, cleaner.config.Interval)
	require.Equal(t, 5*time.Minute, cleaner.config.MaxDuration)
	require.True(t, cleaner.config.VacuumEnabled)
}

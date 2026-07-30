package components

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
)

func TestNewPersistentQueueMetricsPrimesAllBoundedSeries(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := NewPersistentQueueMetrics(reg)

	require.Equal(t, float64(0), testutil.ToFloat64(m.Pending.WithLabelValues("device_tasks", persistentQueueStatusPending)))
	require.Equal(t, float64(0), testutil.ToFloat64(m.OldestAgeSeconds.WithLabelValues("dead_letters", persistentQueueStatusDeadLetter)))
	require.Equal(t, float64(0), testutil.ToFloat64(m.Up.WithLabelValues("async_jobs")))

	families, err := reg.Gather()
	require.NoError(t, err)
	names := make(map[string]bool, len(families))
	for _, family := range families {
		names[family.GetName()] = true
	}
	require.True(t, names["omc_persistent_queue_pending"])
	require.True(t, names["omc_persistent_queue_oldest_age_seconds"])
	require.NotNil(t, m.ObserverFailuresTotal)
}

func TestNormalizePersistentQueueStatus(t *testing.T) {
	tests := map[string]string{
		"queued":     persistentQueueStatusPending,
		"delivering": persistentQueueStatusRunning,
		"done":       persistentQueueStatusSucceeded,
		"expired":    persistentQueueStatusFailed,
		"zombie":     persistentQueueStatusDeadLetter,
	}
	for raw, want := range tests {
		got, err := normalizePersistentQueueStatus(raw)
		require.NoError(t, err, raw)
		require.Equal(t, want, got, raw)
	}
	_, err := normalizePersistentQueueStatus("unexpected")
	require.Error(t, err)
}

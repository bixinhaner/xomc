package components

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
)

func TestNewPersistentQueueMetricsPrimesAllBoundedSeries(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := NewPersistentQueueMetrics(reg)

	require.Equal(t, float64(0), testutil.ToFloat64(m.Pending.WithLabelValues("device_tasks", persistentQueueStatusPending)))
	require.Equal(t, float64(0), testutil.ToFloat64(m.OldestAgeSeconds.WithLabelValues("dead_letters", persistentQueueStatusDeadLetter)))
	require.Equal(t, float64(0), testutil.ToFloat64(m.OverdueOldestAgeSeconds.WithLabelValues("device_tasks", persistentQueueStatusSent)))
	require.Equal(t, float64(0), testutil.ToFloat64(m.Up.WithLabelValues("async_jobs")))

	families, err := reg.Gather()
	require.NoError(t, err)
	names := make(map[string]bool, len(families))
	for _, family := range families {
		names[family.GetName()] = true
	}
	require.True(t, names["omc_persistent_queue_pending"])
	require.True(t, names["omc_persistent_queue_oldest_age_seconds"])
	require.True(t, names["omc_persistent_queue_overdue_oldest_age_seconds"])
	require.NotNil(t, m.ObserverFailuresTotal)
}

func TestPersistentQueueQueriesUseRowExpiryOnlyForDeviceTasks(t *testing.T) {
	for _, descriptor := range persistentQueueQueries {
		if descriptor.name == "device_tasks" {
			require.Contains(t, descriptor.query, "expires_at < now()")
			require.Contains(t, descriptor.query, "now() - expires_at")
			continue
		}
		require.NotContains(t, descriptor.query, "expires_at", descriptor.name)
		require.True(t, strings.Contains(descriptor.query, "0::double precision"), descriptor.name)
	}
}

func TestDeviceTaskOverdueQueryUsesEachRowsExpiry(t *testing.T) {
	dsn := os.Getenv("TEST_PG_URL")
	if dsn == "" {
		t.Skip("TEST_PG_URL not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	require.NoError(t, pool.Ping(ctx))

	var productionQuery string
	for _, descriptor := range persistentQueueQueries {
		if descriptor.name == "device_tasks" {
			productionQuery = descriptor.query
			break
		}
	}
	require.NotEmpty(t, productionQuery)
	query := `WITH device_tasks(status, created_at, expires_at) AS (
		VALUES
			('sent', now() - interval '3 days', now() + interval '4 days'),
			('sent', now() - interval '10 minutes', now() - interval '2 minutes'),
			('sent', now() - interval '20 minutes', now() - interval '9 minutes')
	) ` + productionQuery

	var status string
	var count int64
	var oldestAge, overdueOldestAge float64
	err = pool.QueryRow(ctx, query).Scan(&status, &count, &oldestAge, &overdueOldestAge)
	require.NoError(t, err)
	require.Equal(t, persistentQueueStatusSent, status)
	require.Equal(t, int64(3), count)
	require.Greater(t, oldestAge, (72*time.Hour - time.Minute).Seconds())
	require.InDelta(t, (9 * time.Minute).Seconds(), overdueOldestAge, 2)
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

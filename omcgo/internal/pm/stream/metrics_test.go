package stream

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
)

func TestMetricsExposeStreamingHealthWithoutHighCardinalityLabels(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := NewMetrics(registry)
	metrics.Ready.Set(1)
	metrics.DuplicateEventsTotal.Inc()
	metrics.WindowsFinalizedTotal.WithLabelValues("complete").Inc()
	metrics.BuiltinReconcileRunsTotal.Inc()
	metrics.BuiltinReconcileErrorsTotal.Inc()
	metrics.BuiltinVersionsChangedTotal.Add(2)
	metrics.BuiltinDefinitionsEmpty.Set(3)
	metrics.FinalizeClaims.Inc()
	metrics.FinalizeInflight.Set(2)
	metrics.FinalizeOldestDueSeconds.Set(3600)
	metrics.FinalizeClaimConflictsTotal.Inc()
	metrics.RedisSampledActiveWindows.WithLabelValues("hourly").Set(3)
	metrics.RedisSampledKeys.WithLabelValues("hourly").Set(12)
	metrics.RedisSampledEstimatedBytes.WithLabelValues("hourly").Set(4096)
	metrics.RedisSweeperDeletedTotal.Inc()
	metrics.RedisWriteErrorsTotal.Inc()
	metrics.RuntimeCleanupRowsTotal.WithLabelValues("outbox").Add(10)
	metrics.RuntimeCleanupErrorsTotal.WithLabelValues("outbox").Inc()
	metrics.RuntimeCleanupDuration.WithLabelValues("outbox").Observe(0.01)
	metrics.RuntimeCleanupBacklog.WithLabelValues("outbox").Set(20)
	metrics.DailyVersionExpectedSlotsMismatchTotal.Inc()
	metrics.ResultReplaceSeconds.Observe(0.01)
	metrics.WindowsPreparedTotal.Inc()
	metrics.WindowsPublishedTotal.Add(2)
	metrics.PublicationDuration.Observe(0.02)
	metrics.DeviceHourReplayReadsTotal.Inc()
	metrics.DeviceHourReplayVersionsTotal.Add(2)
	metrics.DeviceHourReplayErrorsTotal.Inc()

	require.Equal(t, float64(1), testutil.ToFloat64(metrics.Ready))
	require.Equal(t, float64(1), testutil.ToFloat64(metrics.DuplicateEventsTotal))
	require.Equal(t, float64(1), testutil.ToFloat64(
		metrics.WindowsFinalizedTotal.WithLabelValues("complete"),
	))
	require.Equal(t, float64(1), testutil.ToFloat64(metrics.BuiltinReconcileRunsTotal))
	require.Equal(t, float64(1), testutil.ToFloat64(metrics.BuiltinReconcileErrorsTotal))
	require.Equal(t, float64(2), testutil.ToFloat64(metrics.BuiltinVersionsChangedTotal))
	require.Equal(t, float64(3), testutil.ToFloat64(metrics.BuiltinDefinitionsEmpty))
	require.Equal(t, float64(1), testutil.ToFloat64(metrics.FinalizeClaims))
	require.Equal(t, float64(2), testutil.ToFloat64(metrics.FinalizeInflight))
	require.Equal(t, float64(3600), testutil.ToFloat64(metrics.FinalizeOldestDueSeconds))
	require.Equal(t, float64(1), testutil.ToFloat64(metrics.FinalizeClaimConflictsTotal))
	require.Equal(t, float64(1), testutil.ToFloat64(metrics.DailyVersionExpectedSlotsMismatchTotal))
	require.Equal(t, float64(1), testutil.ToFloat64(metrics.WindowsPreparedTotal))
	require.Equal(t, float64(2), testutil.ToFloat64(metrics.WindowsPublishedTotal))
	require.Equal(t, float64(1), testutil.ToFloat64(metrics.DeviceHourReplayReadsTotal))
	require.Equal(t, float64(2), testutil.ToFloat64(metrics.DeviceHourReplayVersionsTotal))
	require.Equal(t, float64(1), testutil.ToFloat64(metrics.DeviceHourReplayErrorsTotal))

	families, err := registry.Gather()
	require.NoError(t, err)
	names := make(map[string]struct{}, len(families))
	for _, family := range families {
		names[family.GetName()] = struct{}{}
	}
	for _, name := range []string{
		"omc_pm_aggregation_finalize_claims_total",
		"omc_pm_aggregation_finalize_inflight",
		"omc_pm_aggregation_finalize_oldest_due_seconds",
		"omc_pm_aggregation_finalize_claim_conflicts_total",
		"omc_pm_aggregation_redis_sampled_active_windows",
		"omc_pm_aggregation_redis_sampled_keys",
		"omc_pm_aggregation_redis_sampled_estimated_bytes",
		"omc_pm_aggregation_redis_sweeper_deleted_total",
		"omc_pm_aggregation_redis_write_errors_total",
		"omc_pm_aggregation_runtime_cleanup_rows_total",
		"omc_pm_aggregation_runtime_cleanup_errors_total",
		"omc_pm_aggregation_runtime_cleanup_duration_seconds",
		"omc_pm_aggregation_runtime_cleanup_backlog",
		"omc_pm_aggregation_daily_version_expected_slots_mismatch_total",
		"omc_pm_aggregation_result_replace_seconds",
		"omc_pm_aggregation_rebuild_snapshot_pages_total",
		"omc_pm_aggregation_windows_prepared_total",
		"omc_pm_aggregation_windows_published_total",
		"omc_pm_aggregation_publication_duration_seconds",
		"omc_pm_aggregation_device_hour_replay_reads_total",
		"omc_pm_aggregation_device_hour_replay_versions_total",
		"omc_pm_aggregation_device_hour_replay_errors_total",
	} {
		require.Contains(t, names, name)
	}
}

func TestRebuildSnapshotScanHistogramCoversOperationalThresholds(t *testing.T) {
	registry := prometheus.NewRegistry()
	NewMetrics(registry)

	families, err := registry.Gather()
	require.NoError(t, err)
	for _, family := range families {
		if family.GetName() != "omc_pm_aggregation_rebuild_snapshot_scan_seconds" {
			continue
		}
		bounds := make([]float64, 0, len(family.Metric[0].Histogram.Bucket))
		for _, bucket := range family.Metric[0].Histogram.Bucket {
			bounds = append(bounds, bucket.GetUpperBound())
		}
		require.Contains(t, bounds, float64(15))
		require.Contains(t, bounds, float64(30))
		require.Contains(t, bounds, float64(60))
		return
	}
	t.Fatal("rebuild snapshot scan histogram not registered")
}

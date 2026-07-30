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

	families, err := registry.Gather()
	require.NoError(t, err)
	names := make(map[string]struct{}, len(families))
	for _, family := range families {
		names[family.GetName()] = struct{}{}
	}
	for _, name := range []string{
		"omc_pm_aggregation_finalize_claims",
		"omc_pm_aggregation_finalize_inflight",
		"omc_pm_aggregation_finalize_oldest_due_seconds",
		"omc_pm_aggregation_finalize_claim_conflicts_total",
		"omc_pm_aggregation_redis_sampled_active_windows",
		"omc_pm_aggregation_redis_sampled_keys",
		"omc_pm_aggregation_redis_sampled_estimated_bytes",
		"omc_pm_aggregation_redis_sweeper_deleted_total",
		"omc_pm_aggregation_redis_write_errors_total",
	} {
		require.Contains(t, names, name)
	}
}

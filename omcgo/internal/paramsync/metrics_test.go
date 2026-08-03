package paramsync

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
)

func TestConvergenceMetricsAreRegisteredAndObservable(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := NewMetrics(registry)

	metrics.RunsReadyButNotFinalized.Set(2)
	metrics.RunCounterDrift.Set(3)
	metrics.RunCounterDriftOldestIdle.Set(301)
	metrics.ActiveRunOldestAge.Set(61)
	metrics.ReconcileFinalized.WithLabelValues("succeeded").Inc()
	metrics.ReconcileDuration.Observe(0.25)
	metrics.RunsBlocked.WithLabelValues("terminal_task_missing_result").Set(4)
	metrics.RunsBlockedOldestIdle.WithLabelValues("terminal_task_missing_result").Set(302)
	metrics.AutomaticReservedRuns.Set(32)
	metrics.AutomaticQueuedRequests.Set(100)
	metrics.AutomaticOldestQueueAge.Set(120)

	count, err := testutil.GatherAndCount(registry,
		"param_sync_runs_ready_but_not_finalized",
		"param_sync_run_counter_drift",
		"param_sync_run_counter_drift_oldest_idle_seconds",
		"param_sync_active_run_oldest_age_seconds",
		"param_sync_reconcile_finalized_total",
		"param_sync_reconcile_duration_seconds",
		"param_sync_runs_blocked",
		"param_sync_runs_blocked_oldest_idle_seconds",
		"param_sync_automatic_admission_reserved_runs",
		"param_sync_automatic_admission_queued_requests",
		"param_sync_automatic_admission_oldest_queue_age_seconds",
	)
	require.NoError(t, err)
	require.Equal(t, 11, count)
	require.Equal(t, float64(4), testutil.ToFloat64(
		metrics.RunsBlocked.WithLabelValues("terminal_task_missing_result"),
	))
	require.Equal(t, float64(302), testutil.ToFloat64(
		metrics.RunsBlockedOldestIdle.WithLabelValues("terminal_task_missing_result"),
	))
	require.Equal(t, float64(100), testutil.ToFloat64(metrics.AutomaticQueuedRequests))
}

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

	require.Equal(t, float64(1), testutil.ToFloat64(metrics.Ready))
	require.Equal(t, float64(1), testutil.ToFloat64(metrics.DuplicateEventsTotal))
	require.Equal(t, float64(1), testutil.ToFloat64(
		metrics.WindowsFinalizedTotal.WithLabelValues("complete"),
	))
	require.Equal(t, float64(1), testutil.ToFloat64(metrics.BuiltinReconcileRunsTotal))
	require.Equal(t, float64(1), testutil.ToFloat64(metrics.BuiltinReconcileErrorsTotal))
	require.Equal(t, float64(2), testutil.ToFloat64(metrics.BuiltinVersionsChangedTotal))
	require.Equal(t, float64(3), testutil.ToFloat64(metrics.BuiltinDefinitionsEmpty))
}

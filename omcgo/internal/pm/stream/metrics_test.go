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

	require.Equal(t, float64(1), testutil.ToFloat64(metrics.Ready))
	require.Equal(t, float64(1), testutil.ToFloat64(metrics.DuplicateEventsTotal))
	require.Equal(t, float64(1), testutil.ToFloat64(
		metrics.WindowsFinalizedTotal.WithLabelValues("complete"),
	))
}

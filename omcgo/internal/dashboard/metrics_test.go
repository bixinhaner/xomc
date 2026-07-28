package dashboard

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDashboardMetricsRegistersLowCardinalityCollectors(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := NewMetrics(registry)

	metrics.ObserveGuardResult("series:lte:weekly:key", "miss")
	metrics.ObserveQuery("series", "weekly", "success", 25*time.Millisecond)
	metrics.SetInflight(2)
	metrics.ObserveMissing("lte", "weekly", 1)
	metrics.ObserveIncomplete("lte", "weekly", 1)
	metrics.SetRollupLag("lte", "weekly", 60)

	families, err := registry.Gather()
	require.NoError(t, err)
	names := make(map[string]bool, len(families))
	for _, family := range families {
		names[family.GetName()] = true
	}
	for _, name := range []string{
		"dashboard_kpi_query_duration_seconds",
		"dashboard_kpi_query_inflight",
		"dashboard_kpi_query_cache_total",
		"dashboard_kpi_missing_result_total",
		"dashboard_kpi_incomplete_window_total",
		"pm_network_rollup_lag_seconds",
	} {
		assert.True(t, names[name], name)
	}
}

func TestDashboardMetricsNilRegistryIsSafe(t *testing.T) {
	metrics := NewMetrics(nil)
	require.NotPanics(t, func() {
		metrics.ObserveGuardResult("summary:latest-hourly", "timeout")
		metrics.ObserveQuery("summary", "hourly", "timeout", time.Second)
		metrics.SetInflight(1)
		metrics.ObserveMissing("nr", "daily", 1)
		metrics.ObserveIncomplete("nr", "daily", 1)
		metrics.SetRollupLag("nr", "daily", 120)
	})
}

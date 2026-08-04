package dashboard

import (
	"strings"
	"testing"
	"time"

	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/pm/metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildNetworkRollupSeriesSQLUsesOnlyPublishedNetworkResults(t *testing.T) {
	start := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(12 * 7 * 24 * time.Hour)

	query, args, err := buildNetworkRollupSeriesSQL(NetworkRollupQuery{
		Technology:  model.TechNR,
		Granularity: metrics.GranularityWeekly,
		MetricPaths: []string{"K2", " K1 ", "K2"},
		StartTime:   start,
		EndTime:     end,
	})
	require.NoError(t, err)

	assert.Contains(t, query, "FROM pm_aggregation_results r")
	assert.Contains(t, query, "JOIN pm_aggregation_publications published_revision")
	assert.Contains(t, query, "published_revision.status = 'published'")
	assert.Contains(t, query, "published_revision.revision = r.revision")
	assert.Contains(t, query, "DISTINCT ON (r.technology, r.metric_path, r.window_start)")
	assert.Contains(t, query, "r.dimension =")
	assert.Contains(t, query, "r.metric_type =")
	assert.Contains(t, query, "r.task_id IN")
	assert.Contains(t, query, "r.technology =")
	assert.Contains(t, query, "r.granularity =")
	assert.Contains(t, query, "r.metric_path IN")
	assert.Contains(t, query, "r.window_start >=")
	assert.Contains(t, query, "r.window_start <")
	assert.Contains(t, query, "ORDER BY r.technology, r.metric_path, r.window_start, r.created_at DESC")
	assert.NotContains(t, query, "pm_metric_values")
	assert.NotContains(t, query, "pm_measurement_anchors")
	assert.NotContains(t, query, "pm_metrics_")

	require.Contains(t, args, model.TechNR)
	require.Contains(t, args, metrics.GranularityWeekly)
	require.Contains(t, args, string(metrics.MetricTypeKPI))
	require.Contains(t, args, "K1")
	require.Contains(t, args, "K2")
	assert.Equal(t, 0, strings.Count(query, "K1"), "metric values must remain parameters")
}

func TestBuildNetworkRollupSeriesSQLUsesPublishedCounterResults(t *testing.T) {
	start := time.Date(2026, 8, 3, 13, 0, 0, 0, time.FixedZone("UTC+8", 8*60*60))
	end := start.Add(24 * time.Hour)

	query, args, err := buildNetworkRollupSeriesSQL(NetworkRollupQuery{
		Technology:  model.TechNR,
		Granularity: metrics.GranularityHourly,
		MetricType:  metrics.MetricTypeCounter,
		MetricPaths: []string{"C010070004"},
		StartTime:   start,
		EndTime:     end,
	})
	require.NoError(t, err)

	assert.Contains(t, query, "FROM pm_aggregation_results r")
	assert.Contains(t, query, "JOIN pm_aggregation_publications published_revision")
	assert.NotContains(t, query, "pm_metrics_hourly")
	require.Contains(t, args, string(metrics.MetricTypeCounter))
	require.Contains(t, args, "C010070004")
}

func TestBuildNetworkRollupSeriesSQLRejectsInvalidMetricType(t *testing.T) {
	start := time.Now().UTC().Add(-time.Hour)
	end := time.Now().UTC()

	_, _, err := buildNetworkRollupSeriesSQL(NetworkRollupQuery{
		Technology: model.TechNR, Granularity: metrics.GranularityHourly,
		MetricType: metrics.MetricType("unknown"), MetricPaths: []string{"C010070004"},
		StartTime: start, EndTime: end,
	})
	require.ErrorContains(t, err, "metric type")
}

func TestBuildNetworkRollupSeriesSQLAcceptsOnlyDashboardGranularities(t *testing.T) {
	start := time.Now().UTC().Add(-time.Hour)
	end := time.Now().UTC()

	for _, granularity := range []metrics.Granularity{
		metrics.GranularityHourly,
		metrics.GranularityDaily,
		metrics.GranularityWeekly,
	} {
		_, _, err := buildNetworkRollupSeriesSQL(NetworkRollupQuery{
			Technology: model.TechLTE, Granularity: granularity,
			MetricPaths: []string{"K1"}, StartTime: start, EndTime: end,
		})
		require.NoError(t, err, granularity)
	}

	for _, granularity := range []metrics.Granularity{
		metrics.Granularity15Min,
		metrics.GranularityMonthly,
	} {
		_, _, err := buildNetworkRollupSeriesSQL(NetworkRollupQuery{
			Technology: model.TechLTE, Granularity: granularity,
			MetricPaths: []string{"K1"}, StartTime: start, EndTime: end,
		})
		require.Error(t, err, granularity)
	}
}

func TestBuildLatestNetworkHourlySQLSelectsLatestPerTechnologyAndMetric(t *testing.T) {
	end := time.Date(2026, 7, 29, 0, 0, 0, 0, time.UTC)
	query, args, err := buildLatestNetworkHourlySQL(end.Add(-24*time.Hour), end)
	require.NoError(t, err)

	assert.Contains(t, query, "FROM pm_aggregation_results r")
	assert.Contains(t, query, "published_revision.revision = r.revision")
	assert.Contains(t, query, "DISTINCT ON (r.technology, r.metric_path)")
	assert.Contains(t, query, "r.dimension =")
	assert.Contains(t, query, "r.metric_type =")
	assert.Contains(t, query, "r.granularity =")
	assert.Contains(t, query, "ORDER BY r.technology, r.metric_path, r.window_start DESC, r.created_at DESC")
	assert.NotContains(t, query, "pm_metric_values")
	require.Contains(t, args, metrics.GranularityHourly)
}

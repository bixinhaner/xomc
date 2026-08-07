package dashboard

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/jsonx"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/pm/metrics"
	pmstream "github.com/omcgo/omcgo/internal/pm/stream"
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
	assert.NotContains(t, query, "DISTINCT ON (r.technology, r.metric_path, r.window_start)")
	assert.NotContains(t, query, "pm_aggregation_version_metrics")
	assert.Contains(t, query, "LEFT JOIN pm_metric_dictionary dictionary")
	assert.NotContains(t, query, "pm_aggregation_version_counters")
	assert.Contains(t, query, "r.dimension =")
	assert.Contains(t, query, "r.metric_type =")
	assert.Contains(t, query, "r.task_id IN")
	assert.Contains(t, query, "r.technology =")
	assert.Contains(t, query, "r.granularity =")
	assert.Contains(t, query, "r.metric_path IN")
	assert.Contains(t, query, "r.window_start >=")
	assert.Contains(t, query, "r.window_start <")
	assert.Contains(t, query, "ORDER BY r.technology, r.metric_path, r.window_start, r.version_effective_from, r.created_at")
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

func TestMergeNetworkRollupVersionSlicesCoversIssue266DaySum(t *testing.T) {
	windowStart := time.Date(2026, 8, 4, 0, 0, 0, 0, time.FixedZone("UTC+8", 8*60*60))
	formula := "(C000060011+C000060022)/1000"
	points := []NetworkRollupPoint{
		{
			Technology: model.TechLTE, MetricPath: "K900010015",
			Granularity: metrics.GranularityDaily,
			WindowStart: windowStart, WindowEnd: windowStart.Add(24 * time.Hour),
			Value: 35287.03464, Aggregation: pmstream.AggregationFormula,
			Formula: formula, SampleCount: 39, StatisType: "sum",
			DefinitionComplete:   true,
			VersionEffectiveFrom: windowStart.Add(19 * time.Hour),
		},
		{
			Technology: model.TechLTE, MetricPath: "K900010015",
			Granularity: metrics.GranularityDaily,
			WindowStart: windowStart, WindowEnd: windowStart.Add(24 * time.Hour),
			Value: 159736.99992, Aggregation: pmstream.AggregationFormula,
			Formula: formula, SampleCount: 164, StatisType: "sum",
			DefinitionComplete:   true,
			VersionEffectiveFrom: windowStart.Add(20 * time.Hour),
		},
	}

	merged := mergeNetworkRollupVersionSlices(points)

	require.Len(t, merged, 1)
	assert.InDelta(t, 195024.03456, float64(merged[0].Value), 0.000001)
	assert.EqualValues(t, 203, merged[0].SampleCount)
}

func TestMergeNetworkRollupVersionSlicesDoesNotCrossFormulaChange(t *testing.T) {
	start := time.Date(2026, 8, 4, 0, 0, 0, 0, time.UTC)
	old := NetworkRollupPoint{
		Technology: model.TechLTE, MetricPath: "K1", Granularity: metrics.GranularityDaily,
		WindowStart: start, WindowEnd: start.Add(24 * time.Hour), Value: 10,
		Aggregation: pmstream.AggregationFormula, Formula: "C1/1000",
		StatisType: "sum", DefinitionComplete: true,
		VersionEffectiveFrom: start.Add(time.Hour),
	}
	latest := old
	latest.Value = 20
	latest.Formula = "C1/1024"
	latest.VersionEffectiveFrom = start.Add(2 * time.Hour)

	merged := mergeNetworkRollupVersionSlices([]NetworkRollupPoint{old, latest})

	require.Len(t, merged, 1)
	assert.Equal(t, jsonx.Float(20), merged[0].Value)
}

func TestMergeNetworkRollupVersionSlicesHandlesAllValueAggregationTypes(t *testing.T) {
	start := time.Date(2026, 8, 4, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name        string
		statisType  string
		first       float64
		second      float64
		firstCount  int64
		secondCount int64
		want        float64
	}{
		{name: "sum", statisType: "sum", first: 10, second: 20, firstCount: 1, secondCount: 3, want: 30},
		{name: "avg weighted by valid samples", statisType: "avg", first: 10, second: 20, firstCount: 1, secondCount: 3, want: 17.5},
		{name: "min", statisType: "min", first: 10, second: 20, firstCount: 1, secondCount: 3, want: 10},
		{name: "max", statisType: "max", first: 10, second: 20, firstCount: 1, secondCount: 3, want: 20},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			first := NetworkRollupPoint{
				Technology: model.TechLTE, MetricPath: "K1", Granularity: metrics.GranularityDaily,
				WindowStart: start, WindowEnd: start.Add(24 * time.Hour), Value: jsonx.Float(tt.first),
				Aggregation: pmstream.AggregationFormula, Formula: "C1", StatisType: tt.statisType,
				DefinitionComplete: true,
				SampleCount:        tt.firstCount, Complete: true, VersionEffectiveFrom: start.Add(time.Hour),
			}
			second := first
			second.Value = jsonx.Float(tt.second)
			second.SampleCount = tt.secondCount
			second.VersionEffectiveFrom = start.Add(2 * time.Hour)

			merged := mergeNetworkRollupVersionSlices([]NetworkRollupPoint{first, second})

			require.Len(t, merged, 1)
			assert.InDelta(t, tt.want, float64(merged[0].Value), 0.000001)
			assert.EqualValues(t, tt.firstCount+tt.secondCount, merged[0].SampleCount)
		})
	}
}

func TestMergeNetworkRollupVersionSlicesHandlesCounterOperations(t *testing.T) {
	start := time.Date(2026, 8, 4, 0, 0, 0, 0, time.UTC)
	first := NetworkRollupPoint{
		Technology: model.TechLTE, MetricPath: "C1", Granularity: metrics.GranularityDaily,
		WindowStart: start, WindowEnd: start.Add(24 * time.Hour), Value: 10,
		Aggregation: pmstream.AggregationAvg, SampleCount: 1,
		VersionEffectiveFrom: start.Add(time.Hour),
	}
	second := first
	second.Value = 20
	second.SampleCount = 3
	second.VersionEffectiveFrom = start.Add(2 * time.Hour)

	merged := mergeNetworkRollupVersionSlices([]NetworkRollupPoint{first, second})

	require.Len(t, merged, 1)
	assert.InDelta(t, 17.5, float64(merged[0].Value), 0.000001)
}

func TestMergeNetworkRollupVersionSlicesStopsAtSemanticChange(t *testing.T) {
	start := time.Date(2026, 8, 4, 0, 0, 0, 0, time.UTC)
	point := func(value float64, formula string, hour int) NetworkRollupPoint {
		return NetworkRollupPoint{
			Technology: model.TechLTE, MetricPath: "K1", Granularity: metrics.GranularityDaily,
			WindowStart: start, WindowEnd: start.Add(24 * time.Hour), Value: jsonx.Float(value),
			Aggregation: pmstream.AggregationFormula, Formula: formula, StatisType: "sum",
			DefinitionComplete:   true,
			VersionEffectiveFrom: start.Add(time.Duration(hour) * time.Hour),
		}
	}

	merged := mergeNetworkRollupVersionSlices([]NetworkRollupPoint{
		point(10, "C1", 1), point(20, "C1/1000", 2), point(30, "C1", 3),
	})

	require.Len(t, merged, 1)
	assert.Equal(t, jsonx.Float(30), merged[0].Value)
}

func TestMergeNetworkRollupVersionSlicesDoesNotCrossStatisTypeChange(t *testing.T) {
	start := time.Date(2026, 8, 4, 0, 0, 0, 0, time.UTC)
	old := NetworkRollupPoint{
		Technology: model.TechLTE, MetricPath: "K1", Granularity: metrics.GranularityDaily,
		WindowStart: start, WindowEnd: start.Add(24 * time.Hour), Value: 10,
		Aggregation: pmstream.AggregationFormula, Formula: "C1", StatisType: "sum",
		DefinitionComplete: true,
		SampleCount:        1, VersionEffectiveFrom: start.Add(time.Hour),
	}
	latest := old
	latest.Value = 20
	latest.StatisType = "avg"
	latest.SampleCount = 3
	latest.VersionEffectiveFrom = start.Add(2 * time.Hour)

	merged := mergeNetworkRollupVersionSlices([]NetworkRollupPoint{old, latest})

	require.Len(t, merged, 1)
	assert.Equal(t, jsonx.Float(20), merged[0].Value)
	assert.EqualValues(t, 3, merged[0].SampleCount)
}

type fakeNetworkTaskVersionLoader struct {
	versions []*pmstream.TaskVersionSnapshot
	loaded   []uuid.UUID
}

func (f *fakeNetworkTaskVersionLoader) LoadVersionsByID(
	_ context.Context,
	versionIDs []uuid.UUID,
) ([]*pmstream.TaskVersionSnapshot, error) {
	f.loaded = append([]uuid.UUID(nil), versionIDs...)
	return f.versions, nil
}

func TestEnrichKPIDefinitionsReusesPMVersionRules(t *testing.T) {
	versionID := uuid.New()
	loader := &fakeNetworkTaskVersionLoader{versions: []*pmstream.TaskVersionSnapshot{{
		VersionID: versionID,
		Metrics: map[string]pmstream.MetricRule{
			"K1": {
				MetricPath: "K1", MetricType: "kpi", Formula: "C2/C1",
				Dependencies: []string{"C2", "C1"},
			},
		},
		Counters: map[string]pmstream.CounterRule{
			"C1": {MetricPath: "C1", Aggregation: pmstream.AggregationSum},
			"C2": {MetricPath: "C2", Aggregation: pmstream.AggregationAvg},
		},
	}}}
	points := []NetworkRollupPoint{{MetricPath: "K1", TaskVersionID: versionID}}
	repo := &NetworkRollupRepository{definitions: loader}

	err := repo.enrichKPIDefinitions(context.Background(), points)

	require.NoError(t, err)
	require.Equal(t, []uuid.UUID{versionID}, loader.loaded)
	assert.Equal(t, "C2/C1", points[0].Formula)
	assert.Equal(t, []string{"C2", "C1"}, points[0].Dependencies)
	assert.Equal(t, []string{"C1:sum", "C2:avg"}, points[0].CounterSignature)
	assert.True(t, points[0].DefinitionComplete)
}

func TestEnrichKPIDefinitionsLeavesMissingCounterRuleIncomplete(t *testing.T) {
	versionID := uuid.New()
	loader := &fakeNetworkTaskVersionLoader{versions: []*pmstream.TaskVersionSnapshot{{
		VersionID: versionID,
		Metrics: map[string]pmstream.MetricRule{
			"K1": {
				MetricPath: "K1", MetricType: "kpi", Formula: "C1+C2",
				Dependencies: []string{"C1", "C2"},
			},
		},
		Counters: map[string]pmstream.CounterRule{
			"C1": {MetricPath: "C1", Aggregation: pmstream.AggregationSum},
		},
	}}}
	points := []NetworkRollupPoint{{MetricPath: "K1", TaskVersionID: versionID}}
	repo := &NetworkRollupRepository{definitions: loader}

	err := repo.enrichKPIDefinitions(context.Background(), points)

	require.NoError(t, err)
	assert.False(t, points[0].DefinitionComplete)
	assert.Empty(t, points[0].Formula)
	assert.Empty(t, points[0].CounterSignature)
}

func TestMergeNetworkRollupVersionSlicesDoesNotCrossCounterRuleChange(t *testing.T) {
	start := time.Date(2026, 8, 4, 0, 0, 0, 0, time.UTC)
	first := NetworkRollupPoint{
		MetricPath: "K1", Granularity: metrics.GranularityDaily,
		WindowStart: start, WindowEnd: start.Add(24 * time.Hour), Value: 10,
		Aggregation: pmstream.AggregationFormula, Formula: "C1", StatisType: "sum",
		Dependencies: []string{"C1"}, CounterSignature: []string{"C1:sum"},
		DefinitionComplete: true, VersionEffectiveFrom: start.Add(time.Hour),
	}
	latest := first
	latest.Value = 20
	latest.CounterSignature = []string{"C1:avg"}
	latest.VersionEffectiveFrom = start.Add(2 * time.Hour)

	merged := mergeNetworkRollupVersionSlices([]NetworkRollupPoint{first, latest})

	require.Len(t, merged, 1)
	assert.Equal(t, jsonx.Float(20), merged[0].Value)
}

func TestMergeNetworkRollupVersionSlicesDoesNotMergeIncompleteDefinitions(t *testing.T) {
	start := time.Date(2026, 8, 4, 0, 0, 0, 0, time.UTC)
	first := NetworkRollupPoint{
		MetricPath: "K1", Granularity: metrics.GranularityDaily,
		WindowStart: start, WindowEnd: start.Add(24 * time.Hour), Value: 10,
		Aggregation: pmstream.AggregationFormula, StatisType: "sum",
		VersionEffectiveFrom: start.Add(time.Hour),
	}
	latest := first
	latest.Value = 20
	latest.VersionEffectiveFrom = start.Add(2 * time.Hour)

	merged := mergeNetworkRollupVersionSlices([]NetworkRollupPoint{first, latest})

	require.Len(t, merged, 1)
	assert.Equal(t, jsonx.Float(20), merged[0].Value)
}

type fakeNetworkCounterRollups struct {
	payloads    []pmstream.RollupPayload
	granularity pmstream.Granularity
	entityKeys  []string
}

func (f *fakeNetworkCounterRollups) ListSnapshots(
	_ context.Context,
	versionID uuid.UUID,
	entityKey string,
	granularity pmstream.Granularity,
	start time.Time,
	end time.Time,
) ([]pmstream.RollupPayload, error) {
	f.granularity = granularity
	f.entityKeys = append(f.entityKeys, entityKey)
	var out []pmstream.RollupPayload
	for _, payload := range f.payloads {
		if payload.TaskVersionID != versionID || payload.EntityKey != entityKey ||
			payload.WindowStart.Before(start) || !payload.WindowStart.Before(end) {
			continue
		}
		out = append(out, payload)
	}
	return out, nil
}

func TestRecomputePercentVersionSlicesUsesComposedCounters(t *testing.T) {
	start := time.Date(2026, 8, 4, 0, 0, 0, 0, time.UTC)
	firstVersion := uuid.New()
	secondVersion := uuid.New()
	point := func(versionID uuid.UUID, value float64, hour int) NetworkRollupPoint {
		return NetworkRollupPoint{
			Technology: model.TechLTE, MetricPath: "K_PCT", Granularity: metrics.GranularityDaily,
			WindowStart: start, WindowEnd: start.Add(24 * time.Hour), Value: jsonx.Float(value),
			Aggregation: pmstream.AggregationFormula, Formula: "C_NUM/C_DEN*100",
			Dependencies:     []string{"C_NUM", "C_DEN"},
			CounterSignature: []string{"C_DEN:sum", "C_NUM:sum"}, StatisType: "pct",
			DefinitionComplete: true,
			TaskVersionID:      versionID, SampleCount: 4, Complete: true,
			VersionEffectiveFrom: start.Add(time.Duration(hour) * time.Hour),
		}
	}
	points := []NetworkRollupPoint{
		point(firstVersion, 100, 19),
		point(secondVersion, 9.090909, 20),
	}
	payload := func(versionID uuid.UUID, hour int, numerator, denominator float64) pmstream.RollupPayload {
		return pmstream.RollupPayload{
			TaskVersionID: versionID, EntityKey: "network",
			WindowStart: start.Add(time.Duration(hour) * time.Hour),
			Values: []pmstream.ContributionValue{
				{MetricPath: "C_NUM", Operation: pmstream.AggregationSum, Sum: numerator, Count: 4, Composed: true},
				{MetricPath: "C_DEN", Operation: pmstream.AggregationSum, Sum: denominator, Count: 4, Composed: true},
			},
		}
	}
	rollups := &fakeNetworkCounterRollups{payloads: []pmstream.RollupPayload{
		payload(firstVersion, 19, 10, 10),
		payload(secondVersion, 20, 90, 990),
	}}
	repo := &NetworkRollupRepository{counterRollups: rollups}
	merged := mergeNetworkRollupVersionSlices(points)

	err := repo.recomputePercentVersionSlices(context.Background(), points, merged)

	require.NoError(t, err)
	require.Len(t, merged, 1)
	assert.InDelta(t, 10, float64(merged[0].Value), 0.000001)
	assert.EqualValues(t, 8, merged[0].SampleCount)
	assert.Equal(t, pmstream.GranularityHourly, rollups.granularity)
	assert.Equal(t, []string{"network", "network"}, rollups.entityKeys)
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
	assert.NotContains(t, query, "pm_aggregation_version_metrics")
	assert.NotContains(t, query, "pm_metric_dictionary")
	assert.NotContains(t, query, "pm_aggregation_version_counters")
	assert.NotContains(t, query, "pm_metrics_hourly")
	require.Contains(t, args, string(metrics.MetricTypeCounter))
	require.Contains(t, args, "C010070004")
}

func TestBuildNetworkRollupSeriesSQLKeepsHourlyKPIQueryLightweight(t *testing.T) {
	start := time.Date(2026, 8, 6, 10, 0, 0, 0, time.FixedZone("UTC+8", 8*60*60))
	end := start.Add(24 * time.Hour)

	query, _, err := buildNetworkRollupSeriesSQL(NetworkRollupQuery{
		Technology: model.TechLTE, Granularity: metrics.GranularityHourly,
		MetricType: metrics.MetricTypeKPI, MetricPaths: []string{"K900010015"},
		StartTime: start, EndTime: end,
	})
	require.NoError(t, err)

	assert.Contains(t, query, "JOIN pm_aggregation_publications published_revision")
	assert.NotContains(t, query, "pm_aggregation_version_metrics")
	assert.NotContains(t, query, "pm_metric_dictionary")
	assert.NotContains(t, query, "pm_aggregation_version_counters")
}

func TestBuildNetworkRollupSeriesSQLKeepsDailyKPIQueryInsideTSDB(t *testing.T) {
	start := time.Date(2026, 8, 4, 0, 0, 0, 0, time.FixedZone("UTC+8", 8*60*60))
	end := start.Add(24 * time.Hour)

	query, _, err := buildNetworkRollupSeriesSQL(NetworkRollupQuery{
		Technology: model.TechLTE, Granularity: metrics.GranularityDaily,
		MetricType: metrics.MetricTypeKPI, MetricPaths: []string{"K900010015"},
		StartTime: start, EndTime: end,
	})
	require.NoError(t, err)

	assert.NotContains(t, query, "pm_aggregation_version_metrics")
	assert.Contains(t, query, "pm_metric_dictionary")
	assert.NotContains(t, query, "pm_aggregation_version_counters")
}

func TestBuildNetworkRollupSeriesSQLKeepsDailyCounterQueryLightweight(t *testing.T) {
	start := time.Date(2026, 8, 4, 0, 0, 0, 0, time.FixedZone("UTC+8", 8*60*60))
	end := start.Add(24 * time.Hour)

	query, _, err := buildNetworkRollupSeriesSQL(NetworkRollupQuery{
		Technology: model.TechLTE, Granularity: metrics.GranularityDaily,
		MetricType: metrics.MetricTypeCounter, MetricPaths: []string{"C000060011"},
		StartTime: start, EndTime: end,
	})
	require.NoError(t, err)

	assert.NotContains(t, query, "pm_aggregation_version_metrics")
	assert.NotContains(t, query, "pm_metric_dictionary")
	assert.NotContains(t, query, "pm_aggregation_version_counters")
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
	assert.NotContains(t, query, "metric_rule")
	assert.NotContains(t, query, "pm_metric_values")
	require.Contains(t, args, metrics.GranularityHourly)
}

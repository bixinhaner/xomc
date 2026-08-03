package stream

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestFinalizationCoverageUsesClosedVersionSlotsForResultCompleteness(t *testing.T) {
	windowStart := time.Date(2026, 7, 28, 8, 0, 0, 0, time.UTC)
	closedAt := windowStart.Add(16 * time.Hour)
	version := &TaskVersionSnapshot{
		EffectiveFrom: windowStart.Add(5 * time.Hour),
		EffectiveTo:   &closedAt,
	}
	key := WindowKey{
		Granularity: GranularityDaily,
		Start:       windowStart,
		End:         windowStart.Add(24 * time.Hour),
	}
	state := WindowState{
		ExpectedSlots:       19,
		ReceivedSlots:       11,
		SourceExpectedSlots: 11,
		SourceReceivedSlots: 11,
	}

	state, result := finalizationCoverageFor(key, version, state, time.UTC)

	require.EqualValues(t, 11, state.ExpectedSlots)
	require.EqualValues(t, 11, result.VersionExpectedSlots)
	require.False(t, result.PeriodComplete)
	require.True(t, result.DailyVersionExpectedSlotsMismatch)
}

func TestFinalizationCoverageDoesNotFlagNaturalDailyVersion(t *testing.T) {
	windowStart := time.Date(2026, 7, 28, 8, 0, 0, 0, time.UTC)
	version := &TaskVersionSnapshot{EffectiveFrom: windowStart}
	key := WindowKey{Granularity: GranularityDaily, Start: windowStart, End: windowStart.Add(24 * time.Hour)}

	_, coverage := finalizationCoverageFor(key, version, WindowState{ExpectedSlots: 24}, time.UTC)

	require.False(t, coverage.DailyVersionExpectedSlotsMismatch)
}

func TestDailyVersionExpectedSlotsMismatchOnlyCountsInitialPublication(t *testing.T) {
	coverage := finalizationCoverage{DailyVersionExpectedSlotsMismatch: true}
	require.True(t, shouldRecordDailyVersionExpectedSlotsMismatch(1, coverage))
	require.False(t, shouldRecordDailyVersionExpectedSlotsMismatch(2, coverage))
	require.False(t, shouldRecordDailyVersionExpectedSlotsMismatch(1, finalizationCoverage{}))
}

func TestFinalizerUsesConfiguredLocationForDSTVersionBoundary(t *testing.T) {
	location, err := time.LoadLocation("America/New_York")
	require.NoError(t, err)
	window, err := WindowFor(
		time.Date(2026, 3, 2, 0, 0, 0, 0, location),
		GranularityWeekly,
		location,
	)
	require.NoError(t, err)
	key := WindowKey{Granularity: GranularityWeekly, Start: window.Start, End: window.End}
	version := &TaskVersionSnapshot{EffectiveFrom: key.End}
	finalizer := NewFinalizer(nil, nil, nil).SetLocation(location)

	state, _ := finalizer.finalizationCoverageFor(
		key, version, WindowState{ExpectedSlots: 7},
	)

	require.EqualValues(t, 7, state.ExpectedSlots)
}

func TestFinalizerLocationProviderReflectsRuntimeChanges(t *testing.T) {
	tokyo, err := time.LoadLocation("Asia/Tokyo")
	require.NoError(t, err)
	newYork, err := time.LoadLocation("America/New_York")
	require.NoError(t, err)
	current := tokyo
	finalizer := NewFinalizer(nil, nil, nil).SetLocationProvider(func() *time.Location {
		return current
	})

	require.Equal(t, "Asia/Tokyo", finalizer.finalizationLocation().String())

	current = newYork
	require.Equal(t, "America/New_York", finalizer.finalizationLocation().String())
}

func TestBuildFinalizedMetricsCalculatesKPIFromAggregatedRawCounters(t *testing.T) {
	base := ContributionValue{
		Dimension: DimensionNetwork, DimensionKey: "network", DimensionName: "Network",
	}
	numerator := base
	numerator.MetricPath = "C_NUM"
	numerator.MetricType = "counter"
	numerator.Operation = AggregationSum
	denominator := base
	denominator.MetricPath = "C_DEN"
	denominator.MetricType = "counter"
	denominator.Operation = AggregationSum
	averageCounter := base
	averageCounter.MetricPath = "C_AVG"
	averageCounter.MetricType = "counter"
	averageCounter.Operation = AggregationAvg

	version := &TaskVersionSnapshot{Metrics: map[string]MetricRule{
		"K_RATIO": {
			MetricID: "K_RATIO", MetricPath: "K_RATIO", MetricType: "kpi",
			Aggregation: AggregationFormula, Formula: "C_NUM/C_DEN",
			Dependencies: []string{"C_NUM", "C_DEN"},
		},
		"C_AVG": {
			MetricID: "C_AVG", MetricPath: "C_AVG", MetricType: "counter",
			Aggregation: AggregationAvg, Dependencies: []string{"C_AVG"},
		},
	}}
	state := WindowState{Accumulators: []Accumulator{
		{Definition: numerator, Sum: 110, Count: 2, Min: 10, Max: 100},
		{Definition: denominator, Sum: 101, Count: 2, Min: 1, Max: 100},
		{Definition: averageCounter, Sum: 30, Count: 3, Min: 5, Max: 15},
	}}

	metrics, incomplete, err := buildFinalizedMetrics(version, state)
	require.NoError(t, err)
	require.False(t, incomplete)
	require.Len(t, metrics, 2)

	byID := make(map[string]finalizedMetric, len(metrics))
	for _, metric := range metrics {
		byID[metric.MetricID] = metric
	}
	require.InDelta(t, 110.0/101.0, byID["K_RATIO"].Value, 1e-12)
	require.NotEqual(t, (10.0+0.1)/2.0, byID["K_RATIO"].Value,
		"the rollup must not average child KPI ratios")
	require.InDelta(t, 10.0, byID["C_AVG"].Value, 1e-12)
	require.EqualValues(t, 3, byID["C_AVG"].SampleCount)
}

func TestCompactCounterValuesCarriesOnlyComposedCounterState(t *testing.T) {
	definition := ContributionValue{
		Dimension: DimensionNetwork, DimensionKey: "network",
		MetricPath: "C001", MetricType: "counter", Operation: AggregationSum,
		Value: 999,
	}
	version := &TaskVersionSnapshot{Counters: map[string]CounterRule{
		"C001": {MetricPath: "C001", Aggregation: AggregationAvg},
	}}
	values := compactCounterValues(version, []Accumulator{{
		Definition: definition, Sum: 30, Count: 3, Min: 5, Max: 15,
	}})
	require.Len(t, values, 1)
	require.True(t, values[0].Composed)
	require.Zero(t, values[0].Value)
	require.Equal(t, AggregationAvg, values[0].Operation)
	require.EqualValues(t, 30, values[0].Sum)
	require.EqualValues(t, 3, values[0].Count)
	require.Equal(t, []Granularity{GranularityDaily}, rollupTargets(GranularityHourly))
	require.Equal(t,
		[]Granularity{GranularityWeekly, GranularityMonthly},
		rollupTargets(GranularityDaily),
	)
}

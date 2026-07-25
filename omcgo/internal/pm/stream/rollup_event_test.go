package stream

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestBuildRollupPayloadsCarriesOnlyCompactCounters(t *testing.T) {
	taskID := uuid.New()
	versionID := uuid.New()
	deviceID := uuid.New()
	start := time.Date(2026, 7, 25, 8, 0, 0, 0, time.UTC)
	version := &TaskVersionSnapshot{
		TaskID: taskID, VersionID: versionID, Enabled: true,
		EffectiveFrom: start.Add(-time.Hour),
		Granularities: []Granularity{
			GranularityHourly, GranularityDaily, GranularityWeekly, GranularityMonthly,
		},
		Members: map[uuid.UUID][]TaskMember{
			deviceID: {{DeviceID: deviceID, DimensionKey: "network"}},
		},
		Counters: map[string]CounterRule{
			"C1": {MetricPath: "C1", Aggregation: AggregationSum},
		},
	}
	state := WindowState{
		ExpectedSlots: 4, ReceivedSlots: 4,
		Accumulators: []Accumulator{
			{
				Definition: ContributionValue{
					Dimension: DimensionNetwork, DimensionKey: "network",
					MetricPath: "C1", MetricType: "counter", Operation: AggregationSum,
				},
				Sum: 10, Count: 4, Min: 1, Max: 4,
			},
			{
				Definition: ContributionValue{
					Dimension: DimensionNetwork, DimensionKey: "network",
					MetricPath: "K1", MetricType: "kpi", Operation: AggregationFormula,
				},
				Sum: 99, Count: 4, Min: 1, Max: 99,
			},
		},
	}
	key := WindowKey{
		TaskID: taskID, TaskVersionID: versionID, Granularity: GranularityHourly,
		Start: start, End: start.Add(time.Hour),
	}

	first, err := buildRollupPayloads(key, CloseComplete, state, version, 500)
	require.NoError(t, err)
	second, err := buildRollupPayloads(key, CloseComplete, state, version, 500)
	require.NoError(t, err)
	require.Len(t, first, 1)
	require.Equal(t, first[0].EventID, second[0].EventID)
	require.True(t, first[0].Complete)
	require.Len(t, first[0].Values, 1)
	require.Equal(t, "C1", first[0].Values[0].MetricPath)
	require.Equal(t, 10.0, first[0].Values[0].Sum)
	require.Equal(t, int64(4), first[0].Values[0].Count)
	require.True(t, first[0].Values[0].Composed)
}

func TestRollupContributionsHourlyToDailyAndDailyToWeekMonth(t *testing.T) {
	taskID := uuid.New()
	versionID := uuid.New()
	deviceID := uuid.New()
	start := time.Date(2026, 7, 27, 8, 0, 0, 0, time.UTC)
	version := &TaskVersionSnapshot{
		TaskID: taskID, VersionID: versionID, Enabled: true,
		EffectiveFrom: start.AddDate(0, -2, 0),
		Granularities: []Granularity{
			GranularityHourly, GranularityDaily, GranularityWeekly, GranularityMonthly,
		},
		Members: map[uuid.UUID][]TaskMember{
			deviceID: {{DeviceID: deviceID, DimensionKey: "network"}},
		},
	}
	value := ContributionValue{
		Dimension: DimensionNetwork, DimensionKey: "network",
		MetricPath: "C1", MetricType: "counter", Operation: AggregationSum,
		Sum: 10, Count: 4, Min: 1, Max: 4, Composed: true,
	}
	hourly := RollupPayload{
		SchemaVersion: SchemaVersion, EventID: uuid.New(), TaskID: taskID,
		TaskVersionID: versionID, SourceGranularity: GranularityHourly,
		WindowStart: start, WindowEnd: start.Add(time.Hour),
		SourceExpectedSlots: 4, SourceReceivedSlots: 4, Complete: true,
		ChunkCount: 1, Values: []ContributionValue{value},
	}
	daily, err := rollupContributions(hourly, version, time.UTC)
	require.NoError(t, err)
	require.Len(t, daily, 1)
	require.Equal(t, GranularityDaily, daily[0].Key.Granularity)
	require.Equal(t, int64(24), daily[0].ExpectedSlots)

	dailyPayload := hourly
	dailyPayload.EventID = uuid.New()
	dailyPayload.SourceGranularity = GranularityDaily
	dailyPayload.WindowStart = start.Truncate(24 * time.Hour)
	dailyPayload.WindowEnd = dailyPayload.WindowStart.Add(24 * time.Hour)
	parents, err := rollupContributions(dailyPayload, version, time.UTC)
	require.NoError(t, err)
	require.Len(t, parents, 2)
	require.Equal(t, GranularityWeekly, parents[0].Key.Granularity)
	require.Equal(t, GranularityMonthly, parents[1].Key.Granularity)
	require.Equal(t, int64(7), parents[0].ExpectedSlots)
	require.Equal(t, int64(31), parents[1].ExpectedSlots)
}

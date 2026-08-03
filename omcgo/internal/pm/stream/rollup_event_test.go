package stream

import (
	"encoding/json"
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
		EntityKey: "network", Start: start, End: start.Add(time.Hour),
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
	encoded, err := json.Marshal(first[0])
	require.NoError(t, err)
	require.NotContains(t, string(encoded), "DimensionKey")
	require.NotContains(t, string(encoded), "source_expected_slots")
}

func TestBuildRollupPayloadsTimeoutClosePreservesDataCompleteness(t *testing.T) {
	start := time.Date(2026, 7, 28, 8, 0, 0, 0, time.UTC)
	version := &TaskVersionSnapshot{
		TaskID: uuid.New(), VersionID: uuid.New(), Enabled: true,
		Counters: map[string]CounterRule{
			"C1": {MetricPath: "C1", Aggregation: AggregationSum},
		},
	}
	key := WindowKey{
		TaskID: version.TaskID, TaskVersionID: version.VersionID,
		EntityKey: "network", Granularity: GranularityHourly,
		Start: start, End: start.Add(time.Hour),
	}
	state := WindowState{
		ExpectedSlots: 4, ReceivedSlots: 4,
		Accumulators: []Accumulator{{
			Definition: ContributionValue{
				Dimension: DimensionNetwork, DimensionKey: "network",
				MetricPath: "C1", MetricType: "counter", Operation: AggregationSum,
			},
			Sum: 10, Count: 4, Min: 1, Max: 4,
		}},
	}

	payloads, err := buildRollupPayloads(key, CloseTimeout, state, version, 500)

	require.NoError(t, err)
	require.Len(t, payloads, 1)
	require.True(t, payloads[0].Complete)
}

func TestBuildDeviceRollupPayloadUsesWindowEntity(t *testing.T) {
	start := time.Date(2026, 7, 28, 8, 0, 0, 0, time.UTC)
	firstDevice := uuid.New()
	version := &TaskVersionSnapshot{
		TaskID: uuid.New(), VersionID: uuid.New(), Enabled: true, DevicePipeline: true,
		RollupVersionID: uuid.New(),
		Counters:        map[string]CounterRule{"C1": {MetricPath: "C1", Aggregation: AggregationSum}},
	}
	accumulator := func(deviceID uuid.UUID) Accumulator {
		return Accumulator{
			Definition: ContributionValue{
				Dimension: DimensionDevice, DimensionKey: deviceID.String(),
				MetricPath: "C1", MetricType: "counter", Operation: AggregationSum,
			},
			Sum: 10, Count: 4, Min: 1, Max: 4,
		}
	}
	state := WindowState{
		ExpectedSlots: 4, ReceivedSlots: 4,
		Accumulators: []Accumulator{accumulator(firstDevice)},
	}
	key := WindowKey{
		TaskID: version.TaskID, TaskVersionID: version.VersionID,
		EntityKey:   firstDevice.String(),
		Granularity: GranularityHourly, Start: start, End: start.Add(time.Hour),
	}

	payloads, err := buildRollupPayloads(key, CloseComplete, state, version, 500)

	require.NoError(t, err)
	require.Len(t, payloads, 1)
	require.Equal(t, firstDevice.String(), payloads[0].EntityKey)
	require.Equal(t, version.RollupVersionID, payloads[0].TaskVersionID)
	for _, payload := range payloads {
		require.Equal(t, 1, payload.ChunkCount)
		require.Len(t, payload.Values, 1)
	}
}

func TestChunkRollupValuesHonorsEncodedByteLimit(t *testing.T) {
	values := []ContributionValue{
		{Dimension: DimensionDevice, DimensionKey: "a", MetricPath: "C1", MetricType: "counter", Count: 1, Composed: true},
		{Dimension: DimensionDevice, DimensionKey: "b", MetricPath: "C2", MetricType: "counter", Count: 1, Composed: true},
	}
	first, err := json.Marshal(values[0])
	require.NoError(t, err)

	chunks, err := chunkRollupValues(values, 500, len(first)+3)

	require.NoError(t, err)
	require.Len(t, chunks, 2)
	require.Len(t, chunks[0], 1)
	require.Len(t, chunks[1], 1)
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
		EntityKey:   deviceID.String(),
		WindowStart: start, WindowEnd: start.Add(time.Hour),
		SourceExpectedSlots: 4, SourceReceivedSlots: 4, Complete: true,
		ChunkCount: 1, Values: []ContributionValue{value},
	}
	daily, err := rollupContributions(hourly, version, time.UTC)
	require.NoError(t, err)
	require.Len(t, daily, 1)
	require.Equal(t, GranularityDaily, daily[0].Key.Granularity)
	require.Equal(t, int64(24), daily[0].ExpectedSlots)
	require.Equal(t, hourly.EntityKey, daily[0].Key.EntityKey)
	require.Equal(t, hourly.SourceExpectedSlots, daily[0].SourceExpectedSlots)

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

func TestRollupContributionsUseDynamicMatcherLocationForDailyAndWeeklyBoundaries(t *testing.T) {
	tokyo, err := time.LoadLocation("Asia/Tokyo")
	require.NoError(t, err)
	newYork, err := time.LoadLocation("America/New_York")
	require.NoError(t, err)
	shanghai, err := time.LoadLocation("Asia/Shanghai")
	require.NoError(t, err)

	current := tokyo
	matcher := NewMatcherWithLocationProvider(func() *time.Location {
		return current
	})
	version := &TaskVersionSnapshot{
		TaskID: uuid.New(), VersionID: uuid.New(), Enabled: true,
		EffectiveFrom: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		Granularities: []Granularity{GranularityHourly, GranularityDaily, GranularityWeekly},
	}
	value := ContributionValue{
		Dimension: DimensionNetwork, DimensionKey: "network",
		MetricPath: "C1", MetricType: "counter", Operation: AggregationSum,
		Sum: 10, Count: 1, Min: 10, Max: 10, Composed: true,
	}
	hourly := RollupPayload{
		SchemaVersion: SchemaVersion, EventID: uuid.New(),
		TaskID: version.TaskID, TaskVersionID: version.VersionID,
		SourceGranularity: GranularityHourly, EntityKey: "network",
		WindowStart:         time.Date(2026, 7, 29, 16, 30, 0, 0, time.UTC),
		WindowEnd:           time.Date(2026, 7, 29, 17, 30, 0, 0, time.UTC),
		SourceExpectedSlots: 1, SourceReceivedSlots: 1, Complete: true,
		ChunkCount: 1, Values: []ContributionValue{value},
	}

	dailyTokyo, err := rollupContributions(hourly, version, matcher.Location())
	require.NoError(t, err)
	require.Len(t, dailyTokyo, 1)
	require.True(t, dailyTokyo[0].Key.Start.Equal(time.Date(2026, 7, 30, 0, 0, 0, 0, tokyo)))

	current = newYork
	dailyNewYork, err := rollupContributions(hourly, version, matcher.Location())
	require.NoError(t, err)
	require.Len(t, dailyNewYork, 1)
	require.True(t, dailyNewYork[0].Key.Start.Equal(time.Date(2026, 7, 29, 0, 0, 0, 0, newYork)))
	require.False(t, dailyNewYork[0].Key.Start.Equal(time.Date(2026, 7, 29, 0, 0, 0, 0, time.UTC)))
	require.False(t, dailyNewYork[0].Key.Start.Equal(time.Date(2026, 7, 30, 0, 0, 0, 0, shanghai)))
	require.False(t, dailyTokyo[0].Key.Start.Equal(dailyNewYork[0].Key.Start))

	dailyPayload := hourly
	dailyPayload.EventID = uuid.New()
	dailyPayload.SourceGranularity = GranularityDaily
	dailyPayload.WindowStart = time.Date(2026, 7, 29, 15, 0, 0, 0, time.UTC)
	dailyPayload.WindowEnd = dailyPayload.WindowStart.Add(24 * time.Hour)

	current = tokyo
	weeklyTokyo, err := rollupContributions(dailyPayload, version, matcher.Location())
	require.NoError(t, err)
	require.Len(t, weeklyTokyo, 1)
	require.Equal(t, GranularityWeekly, weeklyTokyo[0].Key.Granularity)
	require.True(t, weeklyTokyo[0].Key.Start.Equal(time.Date(2026, 7, 27, 0, 0, 0, 0, tokyo)))

	current = newYork
	weeklyNewYork, err := rollupContributions(dailyPayload, version, matcher.Location())
	require.NoError(t, err)
	require.Len(t, weeklyNewYork, 1)
	require.True(t, weeklyNewYork[0].Key.Start.Equal(time.Date(2026, 7, 27, 0, 0, 0, 0, newYork)))
	require.False(t, weeklyNewYork[0].Key.Start.Equal(time.Date(2026, 7, 27, 0, 0, 0, 0, time.UTC)))
	require.False(t, weeklyNewYork[0].Key.Start.Equal(time.Date(2026, 7, 27, 0, 0, 0, 0, shanghai)))
	require.False(t, weeklyTokyo[0].Key.Start.Equal(weeklyNewYork[0].Key.Start))
}

func TestRollupContributionsClipsRuleDayAtVersionBoundary(t *testing.T) {
	day := time.Date(2026, 7, 28, 0, 0, 0, 0, time.UTC)
	change := day.Add(13 * time.Hour)
	oldVersion := &TaskVersionSnapshot{
		TaskID: uuid.New(), VersionID: uuid.New(), Enabled: true,
		EffectiveFrom: day.Add(-24 * time.Hour), EffectiveTo: &change,
		Granularities: []Granularity{GranularityHourly, GranularityDaily},
	}
	newVersion := &TaskVersionSnapshot{
		TaskID: uuid.New(), VersionID: uuid.New(), Enabled: true,
		EffectiveFrom: change,
		Granularities: []Granularity{GranularityHourly, GranularityDaily},
	}
	payloadFor := func(version *TaskVersionSnapshot, start time.Time) RollupPayload {
		return RollupPayload{
			SchemaVersion: SchemaVersion, EventID: uuid.New(),
			TaskID: version.TaskID, TaskVersionID: version.VersionID,
			SourceGranularity: GranularityHourly, EntityKey: "network",
			WindowStart: start, WindowEnd: start.Add(time.Hour),
			SourceExpectedSlots: 1, SourceReceivedSlots: 1, Complete: true,
			ChunkCount: 1,
			Values: []ContributionValue{{
				Dimension: DimensionNetwork, DimensionKey: "network",
				MetricPath: "C1", MetricType: "counter", Operation: AggregationSum,
				Sum: 1, Count: 1, Min: 1, Max: 1, Composed: true,
			}},
		}
	}

	oldContributions, err := rollupContributions(
		payloadFor(oldVersion, change.Add(-time.Hour)), oldVersion, time.UTC,
	)
	require.NoError(t, err)
	newContributions, err := rollupContributions(
		payloadFor(newVersion, change), newVersion, time.UTC,
	)
	require.NoError(t, err)

	require.EqualValues(t, 13, oldContributions[0].ExpectedSlots)
	require.EqualValues(t, 11, newContributions[0].ExpectedSlots)
}

func TestExpectedSlotsForFinalizationClipsVersionBoundaryAfterWindowOpened(t *testing.T) {
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

	got := expectedSlotsForFinalization(key, version, 19, time.UTC)

	require.EqualValues(t, 11, got)
	devicePipeline := *version
	devicePipeline.DevicePipeline = true
	require.EqualValues(t, 19, expectedSlotsForFinalization(key, &devicePipeline, 19, time.UTC))
}

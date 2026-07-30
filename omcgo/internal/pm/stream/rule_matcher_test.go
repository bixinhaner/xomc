package stream

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestMatchDeviceHourRulesFansOneDeviceIntoEveryGroup(t *testing.T) {
	deviceID := uuid.MustParse("10000000-0000-4000-8000-000000000001")
	ruleVersion := &TaskVersionSnapshot{
		TaskID: uuid.New(), VersionID: uuid.New(), Enabled: true,
		Technology: "lte", Dimension: DimensionDeviceGroup,
		EffectiveFrom: time.Date(2026, 7, 28, 0, 0, 0, 0, time.UTC),
		Granularities: []Granularity{GranularityHourly, GranularityDaily},
		Counters: map[string]CounterRule{
			"C1": {MetricPath: "C1", Aggregation: AggregationSum},
		},
		Members: map[uuid.UUID][]TaskMember{
			deviceID: {
				{DeviceID: deviceID, DimensionKey: "DeviceGroup=A", DimensionName: "A"},
				{DeviceID: deviceID, DimensionKey: "DeviceGroup=B", DimensionName: "B"},
			},
		},
	}
	snapshot := BuildTaskSnapshot([]*TaskVersionSnapshot{ruleVersion})
	start := time.Date(2026, 7, 28, 8, 0, 0, 0, time.UTC)
	payload := RollupPayload{
		SchemaVersion: SchemaVersion, EventID: uuid.New(),
		TaskID: uuid.New(), TaskVersionID: uuid.New(),
		SourceGranularity: GranularityHourly, EntityKey: deviceID.String(),
		WindowStart: start, WindowEnd: start.Add(time.Hour),
		SourceExpectedSlots: 4, SourceReceivedSlots: 4, Complete: true,
		ChunkCount: 1,
		Values: []ContributionValue{{
			Dimension: DimensionDevice, DimensionKey: deviceID.String(),
			Technology: "lte",
			MetricPath: "C1", MetricType: "counter", Operation: AggregationSum,
			Sum: 10, Count: 4, Min: 1, Max: 4, Composed: true,
		}},
	}

	contributions, err := matchDeviceHourRules(payload, snapshot, time.UTC)

	require.NoError(t, err)
	require.Len(t, contributions, 2)
	keys := []string{
		contributions[0].Values[0].DimensionKey,
		contributions[1].Values[0].DimensionKey,
	}
	require.ElementsMatch(t, []string{"DeviceGroup=A", "DeviceGroup=B"}, keys)
	require.NotEqual(t, contributions[0].SourceFileID, contributions[1].SourceFileID)
	for _, contribution := range contributions {
		require.Equal(t, ruleVersion.VersionID, contribution.Key.TaskVersionID)
		require.Equal(t, GranularityHourly, contribution.Key.Granularity)
		require.EqualValues(t, 1, contribution.ExpectedSlots)
		require.Equal(t, contribution.Values[0].DimensionKey, contribution.Key.EntityKey)
		require.True(t, contribution.Rollup)
	}
}

func TestMatchDeviceHourRuleWindowSelectsOnlyTargetVersionAndDimension(t *testing.T) {
	deviceID := uuid.New()
	target := &TaskVersionSnapshot{
		TaskID: uuid.New(), VersionID: uuid.New(), Enabled: true,
		Technology: "lte", Dimension: DimensionDeviceGroup,
		Granularities: []Granularity{GranularityHourly},
		Counters: map[string]CounterRule{
			"C1": {MetricPath: "C1", Aggregation: AggregationSum},
		},
		Members: map[uuid.UUID][]TaskMember{
			deviceID: {
				{DeviceID: deviceID, DimensionKey: "DeviceGroup=A", DimensionName: "A"},
				{DeviceID: deviceID, DimensionKey: "DeviceGroup=B", DimensionName: "B"},
			},
		},
	}
	unrelated := &TaskVersionSnapshot{
		TaskID: uuid.New(), VersionID: uuid.New(), Enabled: true,
		Technology: "lte", Dimension: DimensionNetwork,
		Granularities: []Granularity{GranularityHourly},
		Members: map[uuid.UUID][]TaskMember{
			deviceID: {{DeviceID: deviceID, DimensionKey: "network"}},
		},
	}
	start := time.Date(2026, 7, 28, 8, 0, 0, 0, time.UTC)
	payload := RollupPayload{
		SchemaVersion: SchemaVersion, EventID: uuid.New(),
		TaskID: uuid.New(), TaskVersionID: uuid.New(),
		SourceGranularity: GranularityHourly, EntityKey: deviceID.String(),
		WindowStart: start, WindowEnd: start.Add(time.Hour),
		SourceExpectedSlots: 4, SourceReceivedSlots: 4, Complete: true,
		ChunkCount: 1,
		Values: []ContributionValue{{
			Technology: "lte", MetricPath: "C1", MetricType: "counter",
			Operation: AggregationSum, Sum: 10, Count: 4, Composed: true,
		}},
	}
	snapshot := BuildTaskSnapshot([]*TaskVersionSnapshot{target, unrelated})

	contributions, err := matchDeviceHourRuleWindow(payload, snapshot, WindowKey{
		TaskID: target.TaskID, TaskVersionID: target.VersionID,
		EntityKey: "DeviceGroup=B", Granularity: GranularityHourly,
		Start: start, End: start.Add(time.Hour),
	})

	require.NoError(t, err)
	require.Len(t, contributions, 1)
	require.Equal(t, target.VersionID, contributions[0].Key.TaskVersionID)
	require.Equal(t, "DeviceGroup=B", contributions[0].Key.EntityKey)
	require.Equal(t, "B", contributions[0].Values[0].DimensionName)

	contributions, err = matchDeviceHourRuleWindow(payload, snapshot, WindowKey{
		TaskVersionID: target.VersionID, EntityKey: "DeviceGroup=missing",
		Granularity: GranularityHourly,
	})
	require.NoError(t, err)
	require.Empty(t, contributions)
}

func TestMatchDeviceHourRulesPreservesEmptySourceChunk(t *testing.T) {
	deviceID := uuid.New()
	version := &TaskVersionSnapshot{
		TaskID: uuid.New(), VersionID: uuid.New(), Enabled: true,
		Technology: "lte", Dimension: DimensionNetwork,
		EffectiveFrom: time.Date(2026, 7, 28, 0, 0, 0, 0, time.UTC),
		Granularities: []Granularity{GranularityHourly},
		Counters: map[string]CounterRule{
			"SELECTED": {MetricPath: "SELECTED", Aggregation: AggregationSum},
		},
		Members: map[uuid.UUID][]TaskMember{
			deviceID: {{DeviceID: deviceID, DimensionKey: "network"}},
		},
	}
	start := time.Date(2026, 7, 28, 8, 0, 0, 0, time.UTC)
	payload := RollupPayload{
		SchemaVersion: SchemaVersion, EventID: uuid.New(), TaskID: uuid.New(),
		TaskVersionID: uuid.New(), SourceGranularity: GranularityHourly,
		EntityKey: deviceID.String(), WindowStart: start, WindowEnd: start.Add(time.Hour),
		SourceExpectedSlots: 4, SourceReceivedSlots: 4, Complete: true,
		ChunkIndex: 1, ChunkCount: 2,
		Values: []ContributionValue{{
			Dimension: DimensionDevice, DimensionKey: deviceID.String(),
			Technology: "lte", MetricPath: "NOT_SELECTED", MetricType: "counter",
			Operation: AggregationSum, Sum: 1, Count: 1, Composed: true,
		}},
	}

	contributions, err := matchDeviceHourRules(
		payload, BuildTaskSnapshot([]*TaskVersionSnapshot{version}), time.UTC,
	)

	require.NoError(t, err)
	require.Len(t, contributions, 1)
	require.Empty(t, contributions[0].Values)
	require.Equal(t, 1, contributions[0].RollupChunkIndex)
	require.Equal(t, 2, contributions[0].RollupChunkCount)
}

func TestMatchDeviceHourRulesMergesSameDimensionMembersPerSourceChunk(t *testing.T) {
	deviceID := uuid.New()
	version := &TaskVersionSnapshot{
		TaskID: uuid.New(), VersionID: uuid.New(), Enabled: true,
		Technology: "lte", Dimension: DimensionBand,
		EffectiveFrom: time.Date(2026, 7, 28, 0, 0, 0, 0, time.UTC),
		Granularities: []Granularity{GranularityHourly},
		Counters: map[string]CounterRule{
			"C1": {MetricPath: "C1", Aggregation: AggregationSum},
		},
		Members: map[uuid.UUID][]TaskMember{
			deviceID: {
				{DeviceID: deviceID, DimensionKey: "Band=3", DimensionName: "Band=3", ObjectLDN: "1"},
				{DeviceID: deviceID, DimensionKey: "Band=3", DimensionName: "Band=3", ObjectLDN: "2"},
			},
		},
	}
	start := time.Date(2026, 7, 28, 8, 0, 0, 0, time.UTC)
	payload := RollupPayload{
		SchemaVersion: SchemaVersion, EventID: uuid.New(), TaskID: uuid.New(),
		TaskVersionID: uuid.New(), SourceGranularity: GranularityHourly,
		EntityKey: deviceID.String(), WindowStart: start, WindowEnd: start.Add(time.Hour),
		SourceExpectedSlots: 4, SourceReceivedSlots: 4, Complete: true,
		ChunkCount: 1,
		Values: []ContributionValue{{
			Dimension: DimensionDevice, DimensionKey: deviceID.String(),
			ObjectLDN:  "Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.Cellid=2",
			Technology: "lte", MetricPath: "C1", MetricType: "counter",
			Operation: AggregationSum, Sum: 10, Count: 4, Composed: true,
		}},
	}

	contributions, err := matchDeviceHourRules(
		payload, BuildTaskSnapshot([]*TaskVersionSnapshot{version}), time.UTC,
	)

	require.NoError(t, err)
	require.Len(t, contributions, 1)
	require.Equal(t, "Band=3", contributions[0].Key.EntityKey)
	require.Len(t, contributions[0].Values, 1)
	require.Equal(t, "C1", contributions[0].Values[0].MetricPath)
	require.EqualValues(t, 1, contributions[0].ExpectedSlots)
}

func TestIsDeviceHourPayloadAcceptsOnlyStableDeviceRollup(t *testing.T) {
	stableID := uuid.New()
	ruleID := uuid.New()
	snapshot := BuildTaskSnapshot([]*TaskVersionSnapshot{
		{TaskID: uuid.New(), VersionID: stableID, DeviceRollup: true},
		{TaskID: uuid.New(), VersionID: ruleID},
	})
	payload := RollupPayload{
		TaskVersionID: stableID, SourceGranularity: GranularityHourly,
	}

	require.True(t, isDeviceHourPayload(payload, snapshot))
	payload.TaskVersionID = ruleID
	require.False(t, isDeviceHourPayload(payload, snapshot))
	payload.TaskVersionID = stableID
	payload.SourceGranularity = GranularityDaily
	require.False(t, isDeviceHourPayload(payload, snapshot))
}

func TestTaskSnapshotCachesDistinctDimensionMemberCounts(t *testing.T) {
	first, second := uuid.New(), uuid.New()
	version := &TaskVersionSnapshot{
		TaskID: uuid.New(), VersionID: uuid.New(),
		Members: map[uuid.UUID][]TaskMember{
			first: {
				{DeviceID: first, DimensionKey: "network"},
				{DeviceID: first, DimensionKey: "Band=3"},
				{DeviceID: first, DimensionKey: "Band=3"},
			},
			second: {
				{DeviceID: second, DimensionKey: "network"},
			},
		},
	}

	BuildTaskSnapshot([]*TaskVersionSnapshot{version})

	require.EqualValues(t, 2, ruleDimensionMemberCount(version, "network"))
	require.EqualValues(t, 1, ruleDimensionMemberCount(version, "Band=3"))
	require.EqualValues(t, 0, ruleDimensionMemberCount(version, "missing"))
}

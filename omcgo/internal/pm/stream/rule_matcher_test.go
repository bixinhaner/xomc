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

func TestMatchDeviceHourRulesDropsSourceObjectLDNForNetworkRollup(t *testing.T) {
	deviceID := uuid.New()
	version := pctRollupVersion(deviceID, DimensionNetwork, []TaskMember{
		{DimensionKey: "network", DimensionName: "Network"},
	})
	start := time.Date(2026, 7, 28, 8, 0, 0, 0, time.UTC)
	payload := pctRollupPayload(deviceID, start)

	contributions, err := matchDeviceHourRules(
		payload, BuildTaskSnapshot([]*TaskVersionSnapshot{version}), time.UTC,
	)

	require.NoError(t, err)
	requirePctRollupMetric(t, version, contributions, DimensionNetwork, "network", "")
}

func TestMatchDeviceHourRulesDropsSourceObjectLDNForDimensionRollups(t *testing.T) {
	tests := []struct {
		name         string
		dimension    Dimension
		dimensionKey string
	}{
		{
			name:         "product",
			dimension:    DimensionProduct,
			dimensionKey: "Product=LTE",
		},
		{
			name:         "device group",
			dimension:    DimensionDeviceGroup,
			dimensionKey: "DeviceGroup=A",
		},
		{
			name:         "aggregate group",
			dimension:    DimensionAggregateGroup,
			dimensionKey: "AggregateGroup=North",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deviceID := uuid.New()
			version := pctRollupVersion(deviceID, tt.dimension, []TaskMember{
				{DimensionKey: tt.dimensionKey, DimensionName: tt.dimensionKey},
			})
			start := time.Date(2026, 7, 28, 8, 0, 0, 0, time.UTC)
			payload := pctRollupPayload(deviceID, start)

			contributions, err := matchDeviceHourRules(
				payload, BuildTaskSnapshot([]*TaskVersionSnapshot{version}), time.UTC,
			)

			require.NoError(t, err)
			requirePctRollupMetric(t, version, contributions, tt.dimension, tt.dimensionKey, "")
		})
	}
}

func TestMatchDeviceHourRulesCalculatesPctAfterBandRollup(t *testing.T) {
	deviceID := uuid.New()
	version := pctRollupVersion(deviceID, DimensionBand, []TaskMember{
		{DimensionKey: "Band=3", DimensionName: "Band=3", ObjectLDN: "10497"},
		{DimensionKey: "Band=3", DimensionName: "Band=3", ObjectLDN: "10498"},
	})
	start := time.Date(2026, 7, 28, 8, 0, 0, 0, time.UTC)
	payload := pctRollupPayload(deviceID, start)

	contributions, err := matchDeviceHourRules(
		payload, BuildTaskSnapshot([]*TaskVersionSnapshot{version}), time.UTC,
	)

	require.NoError(t, err)
	requirePctRollupMetric(t, version, contributions, DimensionBand, "Band=3", "Band=3")
}

func TestMatchDeviceHourRulesCalculatesPctFromSummedDependencies(t *testing.T) {
	deviceID := uuid.New()
	version := pctRollupVersion(deviceID, DimensionNetwork, []TaskMember{
		{DimensionKey: "network", DimensionName: "Network"},
	})
	start := time.Date(2026, 7, 28, 8, 0, 0, 0, time.UTC)
	payload := pctRollupPayload(deviceID, start)
	for i := range payload.Values {
		value := &payload.Values[i]
		if value.MetricPath == "C_DEN" &&
			value.ObjectLDN == "Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.Cellid=10498" {
			value.Sum = 300
			value.Min = 300
			value.Max = 300
		}
	}

	contributions, err := matchDeviceHourRules(
		payload, BuildTaskSnapshot([]*TaskVersionSnapshot{version}), time.UTC,
	)

	require.NoError(t, err)
	requirePctRollupMetricValue(
		t, version, contributions, DimensionNetwork, "network", "",
		30.0/400.0,
	)
}

func pctRollupVersion(
	deviceID uuid.UUID,
	dimension Dimension,
	members []TaskMember,
) *TaskVersionSnapshot {
	for i := range members {
		members[i].DeviceID = deviceID
	}
	return &TaskVersionSnapshot{
		TaskID: uuid.New(), VersionID: uuid.New(), Enabled: true,
		Technology: "lte", Dimension: dimension,
		EffectiveFrom: time.Date(2026, 7, 28, 0, 0, 0, 0, time.UTC),
		Granularities: []Granularity{GranularityHourly},
		Counters: map[string]CounterRule{
			"C_NUM": {MetricPath: "C_NUM", Aggregation: AggregationSum},
			"C_DEN": {MetricPath: "C_DEN", Aggregation: AggregationSum},
		},
		Metrics: map[string]MetricRule{
			"K_AVAIL": {
				MetricID: "K_AVAIL", MetricPath: "K_AVAIL", MetricType: "kpi",
				Aggregation: AggregationFormula, Formula: "C_NUM/C_DEN",
				Dependencies: []string{"C_NUM", "C_DEN"},
			},
		},
		Members: map[uuid.UUID][]TaskMember{
			deviceID: members,
		},
	}
}

func pctRollupPayload(deviceID uuid.UUID, start time.Time) RollupPayload {
	return RollupPayload{
		SchemaVersion: SchemaVersion, EventID: uuid.New(), TaskID: uuid.New(),
		TaskVersionID: uuid.New(), SourceGranularity: GranularityHourly,
		EntityKey: deviceID.String(), WindowStart: start, WindowEnd: start.Add(time.Hour),
		SourceExpectedSlots: 4, SourceReceivedSlots: 4, Complete: true,
		ChunkCount: 1,
		Values: []ContributionValue{
			{
				Dimension: DimensionDevice, DimensionKey: deviceID.String(),
				ObjectLDN:  "Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.Cellid=10497",
				Technology: "lte",
				MetricPath: "C_NUM", MetricType: "counter", Operation: AggregationSum,
				Sum: 10, Count: 1, Min: 10, Max: 10, Composed: true,
			},
			{
				Dimension: DimensionDevice, DimensionKey: deviceID.String(),
				ObjectLDN:  "Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.Cellid=10497",
				Technology: "lte",
				MetricPath: "C_DEN", MetricType: "counter", Operation: AggregationSum,
				Sum: 100, Count: 1, Min: 100, Max: 100, Composed: true,
			},
			{
				Dimension: DimensionDevice, DimensionKey: deviceID.String(),
				ObjectLDN:  "Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.Cellid=10498",
				Technology: "lte",
				MetricPath: "C_NUM", MetricType: "counter", Operation: AggregationSum,
				Sum: 20, Count: 1, Min: 20, Max: 20, Composed: true,
			},
			{
				Dimension: DimensionDevice, DimensionKey: deviceID.String(),
				ObjectLDN:  "Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.Cellid=10498",
				Technology: "lte",
				MetricPath: "C_DEN", MetricType: "counter", Operation: AggregationSum,
				Sum: 100, Count: 1, Min: 100, Max: 100, Composed: true,
			},
		},
	}
}

func requirePctRollupMetric(
	t *testing.T,
	version *TaskVersionSnapshot,
	contributions []Contribution,
	expectedDimension Dimension,
	expectedDimensionKey string,
	expectedObjectLDN string,
) {
	t.Helper()
	requirePctRollupMetricValue(
		t, version, contributions, expectedDimension, expectedDimensionKey, expectedObjectLDN,
		30.0/200.0,
	)
}

func requirePctRollupMetricValue(
	t *testing.T,
	version *TaskVersionSnapshot,
	contributions []Contribution,
	expectedDimension Dimension,
	expectedDimensionKey string,
	expectedObjectLDN string,
	expectedValue float64,
) {
	t.Helper()
	require.Len(t, contributions, 1)
	require.Equal(t, expectedDimensionKey, contributions[0].Key.EntityKey)
	require.Len(t, contributions[0].Values, 4)
	accumulatorsByDefinitionID := make(map[string]*Accumulator)
	for _, value := range contributions[0].Values {
		require.Equal(t, expectedDimension, value.Dimension)
		require.Equal(t, expectedDimensionKey, value.DimensionKey)
		require.Equal(t, expectedObjectLDN, value.ObjectLDN)
		id, err := accumulatorDefinitionID(value)
		require.NoError(t, err)
		accumulator := accumulatorsByDefinitionID[id]
		if accumulator == nil {
			accumulator = &Accumulator{Definition: value}
			accumulatorsByDefinitionID[id] = accumulator
		}
		accumulator.Sum += value.Sum
		accumulator.Count += value.Count
	}
	require.Len(t, accumulatorsByDefinitionID, 2, "pct dependencies should group by counter, not source object")

	state := WindowState{Accumulators: make([]Accumulator, 0, len(accumulatorsByDefinitionID))}
	for _, accumulator := range accumulatorsByDefinitionID {
		state.Accumulators = append(state.Accumulators, *accumulator)
	}
	metrics, incomplete, err := buildFinalizedMetrics(version, state)
	require.NoError(t, err)
	require.False(t, incomplete)
	require.Len(t, metrics, 1)
	require.Equal(t, "K_AVAIL", metrics[0].MetricID)
	require.InDelta(t, expectedValue, metrics[0].Value, 1e-12)
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
	require.Equal(t, "Band=3", contributions[0].Values[0].ObjectLDN)
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

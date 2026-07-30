package stream

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/stretchr/testify/require"
)

func TestMatcherUsesImmutableVersionWindowBoundary(t *testing.T) {
	deviceID := uuid.New()
	taskID := uuid.New()
	oldID := uuid.New()
	newID := uuid.New()
	changeAt := time.Date(2026, 7, 25, 1, 15, 0, 0, time.UTC)
	oldVersion := &TaskVersionSnapshot{
		TaskID: taskID, VersionID: oldID, VersionNo: 1, Enabled: true,
		Technology: "lte", Dimension: DimensionDevice,
		DevicePipeline: true,
		Granularities:  []Granularity{GranularityHourly},
		EffectiveFrom:  time.Date(2026, 7, 25, 0, 0, 0, 0, time.UTC),
		EffectiveTo:    &changeAt,
		Metrics: map[string]MetricRule{
			"K001": {
				MetricID: "K001", MetricPath: "K001", MetricType: "kpi",
				Aggregation: AggregationFormula, Formula: "C001", Dependencies: []string{"C001"},
			},
		},
		Counters: map[string]CounterRule{"C001": {MetricPath: "C001", Aggregation: AggregationSum}},
		Members: map[uuid.UUID][]TaskMember{
			deviceID: {{DeviceID: deviceID, DeviceSN: "SN-1", DimensionKey: deviceID.String(), DimensionName: "SN-1"}},
		},
	}
	newVersion := *oldVersion
	newVersion.VersionID = newID
	newVersion.VersionNo = 2
	newVersion.EffectiveFrom = changeAt
	newVersion.EffectiveTo = nil
	snapshot := BuildTaskSnapshot([]*TaskVersionSnapshot{oldVersion, &newVersion})

	payload := validNormalizedEvent()
	payload.DeviceID = deviceID
	payload.DeviceSN = "SN-1"
	payload.WindowStart = time.Date(2026, 7, 25, 1, 30, 0, 0, time.UTC)
	payload.WindowEnd = payload.WindowStart.Add(slotDuration)

	contributions, err := NewMatcher(time.UTC).Match(payload, snapshot)
	require.NoError(t, err)
	require.Len(t, contributions, 1)
	require.Equal(t, oldID, contributions[0].Key.TaskVersionID,
		"the old immutable version must finish a parent window that was already open")

	payload.WindowStart = time.Date(2026, 7, 25, 2, 0, 0, 0, time.UTC)
	payload.WindowEnd = payload.WindowStart.Add(slotDuration)
	contributions, err = NewMatcher(time.UTC).Match(payload, snapshot)
	require.NoError(t, err)
	require.Len(t, contributions, 1)
	require.Equal(t, newID, contributions[0].Key.TaskVersionID)
}

func TestMatcherSkipsWindowsAtOrAfterPlannedEnd(t *testing.T) {
	deviceID := uuid.New()
	plannedEndAt := time.Date(2026, 7, 25, 2, 0, 0, 0, time.UTC)
	version := &TaskVersionSnapshot{
		TaskID: uuid.New(), VersionID: uuid.New(), Enabled: true, Technology: "lte",
		Dimension: DimensionDevice, DevicePipeline: true,
		Granularities: []Granularity{GranularityHourly},
		EffectiveFrom: time.Date(2026, 7, 25, 0, 0, 0, 0, time.UTC),
		PlannedEndAt:  &plannedEndAt,
		Metrics: map[string]MetricRule{
			"K001": {
				MetricID: "K001", MetricPath: "K001", MetricType: "kpi",
				Aggregation: AggregationFormula, Formula: "C001", Dependencies: []string{"C001"},
			},
		},
		Counters: map[string]CounterRule{"C001": {MetricPath: "C001", Aggregation: AggregationSum}},
		Members: map[uuid.UUID][]TaskMember{
			deviceID: {{DeviceID: deviceID, DeviceSN: "SN-1", DimensionKey: deviceID.String(), DimensionName: "SN-1"}},
		},
	}
	snapshot := BuildTaskSnapshot([]*TaskVersionSnapshot{version})
	payload := validNormalizedEvent()
	payload.DeviceID = deviceID
	payload.DeviceSN = "SN-1"

	payload.WindowStart = time.Date(2026, 7, 25, 1, 30, 0, 0, time.UTC)
	payload.WindowEnd = payload.WindowStart.Add(slotDuration)
	contributions, err := NewMatcher(time.UTC).Match(payload, snapshot)
	require.NoError(t, err)
	require.Len(t, contributions, 1)

	payload.WindowStart = time.Date(2026, 7, 25, 2, 0, 0, 0, time.UTC)
	payload.WindowEnd = payload.WindowStart.Add(slotDuration)
	contributions, err = NewMatcher(time.UTC).Match(payload, snapshot)
	require.NoError(t, err)
	require.Empty(t, contributions)
}

func TestMatcherFiltersMetricAndObjectLDNWithoutDatabaseReads(t *testing.T) {
	deviceID := uuid.New()
	ldn := "Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.1"
	version := &TaskVersionSnapshot{
		TaskID: uuid.New(), VersionID: uuid.New(), Enabled: true, Technology: "lte",
		Dimension: DimensionDevice, DevicePipeline: true,
		Granularities: []Granularity{GranularityHourly},
		EffectiveFrom: time.Date(2026, 7, 25, 0, 0, 0, 0, time.UTC),
		ObjectLDNs:    map[string]struct{}{ldn: {}},
		Metrics: map[string]MetricRule{
			"K001": {
				MetricID: "K001", MetricPath: "K001", MetricType: "kpi",
				Aggregation: AggregationFormula, Formula: "C001", Dependencies: []string{"C001"},
			},
		},
		Counters: map[string]CounterRule{"C001": {MetricPath: "C001", Aggregation: AggregationSum}},
		Members: map[uuid.UUID][]TaskMember{
			deviceID: {{DeviceID: deviceID, DeviceSN: "SN-1", DimensionKey: deviceID.String(), DimensionName: "SN-1"}},
		},
	}
	payload := validNormalizedEvent()
	payload.DeviceID = deviceID
	payload.Measurements = append(payload.Measurements, event.PMAggregationMeasurement{
		ObjectLDN: "not-selected",
		Metrics:   []event.PMAggregationMetric{{MetricPath: "K001", MetricType: "kpi", Value: 99}},
	})
	payload.Measurements[0].ObjectLDN = ldn
	payload.Measurements[0].Metrics = append(payload.Measurements[0].Metrics,
		event.PMAggregationMetric{MetricPath: "K999", MetricType: "kpi", Value: 99})

	contributions, err := NewMatcher(time.UTC).Match(payload, BuildTaskSnapshot([]*TaskVersionSnapshot{version}))
	require.NoError(t, err)
	require.Len(t, contributions, 1)
	require.Len(t, contributions[0].Values, 1)
	require.Equal(t, "C001", contributions[0].Values[0].MetricPath)
	require.EqualValues(t, 4, contributions[0].ExpectedSlots)
	require.Equal(t, deviceID.String(), contributions[0].Key.EntityKey)
}

func TestMatcherCreatesIndependentDeviceWindows(t *testing.T) {
	firstID, secondID := uuid.New(), uuid.New()
	version := &TaskVersionSnapshot{
		TaskID: uuid.New(), VersionID: uuid.New(), Enabled: true,
		Technology: "lte", Dimension: DimensionDevice, DevicePipeline: true,
		Granularities: []Granularity{GranularityHourly},
		EffectiveFrom: time.Date(2026, 7, 25, 0, 0, 0, 0, time.UTC),
		Counters: map[string]CounterRule{
			"C001": {MetricPath: "C001", Aggregation: AggregationSum},
		},
		Members: map[uuid.UUID][]TaskMember{
			firstID:  {{DeviceID: firstID, DimensionKey: firstID.String()}},
			secondID: {{DeviceID: secondID, DimensionKey: secondID.String()}},
		},
	}
	snapshot := BuildTaskSnapshot([]*TaskVersionSnapshot{version})
	matcher := NewMatcher(time.UTC)
	first := validNormalizedEvent()
	first.DeviceID = firstID
	second := first
	second.DeviceID = secondID

	firstContributions, err := matcher.Match(first, snapshot)
	require.NoError(t, err)
	secondContributions, err := matcher.Match(second, snapshot)
	require.NoError(t, err)

	require.Len(t, firstContributions, 1)
	require.Len(t, secondContributions, 1)
	require.NotEqual(t, firstContributions[0].Key.EntityKey, secondContributions[0].Key.EntityKey)
	require.EqualValues(t, 4, firstContributions[0].ExpectedSlots)
	require.EqualValues(t, 4, secondContributions[0].ExpectedSlots)
}

func TestMatcherDoesNotRecoverMonthlyWindowFromOriginalCounters(t *testing.T) {
	deviceID := uuid.New()
	version := &TaskVersionSnapshot{
		TaskID: uuid.New(), VersionID: uuid.New(), Enabled: true,
		Dimension: DimensionNetwork,
		Granularities: []Granularity{
			GranularityHourly, GranularityDaily, GranularityWeekly, GranularityMonthly,
		},
		EffectiveFrom: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		Counters: map[string]CounterRule{
			"C001": {MetricPath: "C001", Aggregation: AggregationSum},
		},
		Members: map[uuid.UUID][]TaskMember{
			deviceID: {{DeviceID: deviceID, DeviceSN: "SN-1", DimensionKey: "network"}},
		},
	}
	snapshot := BuildTaskSnapshot([]*TaskVersionSnapshot{version})
	payload := validNormalizedEvent()
	payload.DeviceID = deviceID
	payload.WindowStart = time.Date(2026, 7, 25, 3, 15, 0, 0, time.UTC)
	payload.WindowEnd = payload.WindowStart.Add(slotDuration)
	matcher := NewMatcher(time.UTC)
	contributions, err := matcher.MatchGranularity(payload, snapshot, GranularityMonthly)
	require.NoError(t, err)
	require.Empty(t, contributions)
}

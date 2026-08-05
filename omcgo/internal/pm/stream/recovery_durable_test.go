package stream

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReplayDurableDeviceHourSourcesRecoversIssue266MissingQuarter(t *testing.T) {
	deviceID := uuid.MustParse("30f3d50f-e1c4-4c3a-89b1-ac9da9a51350")
	taskID := uuid.New()
	versionID := uuid.New()
	hourStart := time.Date(2026, 8, 4, 21, 0, 0, 0, time.FixedZone("UTC+8", 8*60*60))
	version := &TaskVersionSnapshot{
		TaskID: taskID, VersionID: versionID, Enabled: true,
		Technology: "lte", Dimension: DimensionDevice, DevicePipeline: true,
		Granularities: []Granularity{GranularityHourly}, EffectiveFrom: hourStart,
		Counters: map[string]CounterRule{
			"C000060011": {MetricPath: "C000060011", Aggregation: AggregationSum},
			"C000060022": {MetricPath: "C000060022", Aggregation: AggregationSum},
		},
		Members: map[uuid.UUID][]TaskMember{
			deviceID: {{DeviceID: deviceID, DeviceSN: "120200024100AA00002", DimensionKey: deviceID.String()}},
		},
	}
	snapshot := BuildTaskSnapshot([]*TaskVersionSnapshot{version})
	key := WindowKey{
		TaskID: taskID, TaskVersionID: versionID, EntityKey: deviceID.String(),
		Granularity: GranularityHourly, Start: hourStart, End: hourStart.Add(time.Hour),
	}
	values := [][2]float64{
		{2325393.92, 113.33},
		{4026735.25, 74.55},
		{2089345.25, 44.11},
		{3109685.12, 100.56},
	}
	payloads := make([]event.PMAggregationNormalizedPayload, 0, len(values))
	for index, value := range values {
		payload := validNormalizedEvent()
		payload.EventID = uuid.New()
		payload.SourceFileID = uuid.New()
		payload.DeviceID = deviceID
		payload.DeviceSN = "120200024100AA00002"
		payload.Technology = "lte"
		payload.WindowStart = hourStart.Add(time.Duration(index) * slotDuration)
		payload.WindowEnd = payload.WindowStart.Add(slotDuration)
		payload.Measurements = []event.PMAggregationMeasurement{{
			ObjectLDN: "Cellid=1",
			Metrics: []event.PMAggregationMetric{
				{MetricPath: "C000060011", MetricType: "counter", StatisType: "sum", Value: value[0]},
				{MetricPath: "C000060022", MetricType: "counter", StatisType: "sum", Value: value[1]},
			},
		}}
		payloads = append(payloads, payload)
	}

	seen := map[string]struct{}{}
	var kpi float64
	for index := 0; index < 3; index++ {
		seen[payloads[index].SourceFileID.String()] = struct{}{}
		kpi += (values[index][0] + values[index][1]) / 1000
	}
	visit := func(
		_ context.Context, start, end time.Time,
		fn func(event.PMAggregationNormalizedPayload) error,
	) error {
		require.True(t, start.Equal(hourStart))
		require.True(t, end.Equal(hourStart.Add(time.Hour)))
		for _, payload := range payloads {
			if err := fn(payload); err != nil {
				return err
			}
		}
		return nil
	}
	accumulate := func(_ context.Context, contribution Contribution) (AccumulateResult, error) {
		if _, duplicate := seen[contribution.SourceFileID]; duplicate {
			return AccumulateResult{Duplicate: true, ReceivedSlots: int64(len(seen)), Complete: len(seen) == 4}, nil
		}
		seen[contribution.SourceFileID] = struct{}{}
		for _, value := range contribution.Values {
			kpi += value.Value / 1000
		}
		return AccumulateResult{ReceivedSlots: int64(len(seen)), Complete: len(seen) == 4}, nil
	}

	received, complete, matched, err := replayDurableDeviceHourSources(
		context.Background(), key, snapshot, NewMatcher(hourStart.Location()), visit, accumulate,
	)

	require.NoError(t, err)
	require.True(t, matched)
	require.True(t, complete)
	assert.EqualValues(t, 4, received)
	assert.InDelta(t, 11551.49209, kpi, 0.000001)
	assert.InDelta(t, 3109.78568,
		(values[3][0]+values[3][1])/1000, 0.000001,
		"the exact 21:45 contribution from issue #266 must be restored")
}

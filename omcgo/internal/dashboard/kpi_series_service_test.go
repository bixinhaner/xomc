package dashboard

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/jsonx"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/pm/metrics"
	pmstream "github.com/omcgo/omcgo/internal/pm/stream"
	"github.com/stretchr/testify/require"
)

type recordingNetworkRollupReader struct {
	queries []NetworkRollupQuery
	points  []NetworkRollupPoint
	err     error
}

func (r *recordingNetworkRollupReader) ListSeries(_ context.Context, query NetworkRollupQuery) ([]NetworkRollupPoint, error) {
	r.queries = append(r.queries, query)
	return r.points, r.err
}

func (r *recordingNetworkRollupReader) ListLatestHourly(context.Context, time.Time, time.Time) ([]NetworkRollupPoint, error) {
	return r.points, r.err
}

type fixedNetworkProgressReader struct {
	expectedTaskID uuid.UUID
	result         pmstream.ProgressQueryResult
	err            error
}

func (r *fixedNetworkProgressReader) Query(
	_ context.Context,
	taskID uuid.UUID,
	_, _ time.Time,
) (pmstream.ProgressQueryResult, error) {
	if taskID != r.expectedTaskID {
		return pmstream.ProgressQueryResult{}, errors.New("unexpected network progress task")
	}
	return r.result, r.err
}

func TestGetKPITimeSeriesSnapshotAppendsOnlyRequestedNetworkPartial(t *testing.T) {
	start := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(31 * 24 * time.Hour)
	taskID, ok := pmstream.BuiltinNetworkTaskID("lte")
	require.True(t, ok)
	reader := &recordingNetworkRollupReader{points: []NetworkRollupPoint{{
		Technology: model.TechLTE, MetricPath: "K1",
		Granularity: metrics.GranularityDaily,
		WindowStart: start, WindowEnd: start.Add(24 * time.Hour),
		Value: jsonx.Float(10), Complete: true,
	}}}
	progress := &fixedNetworkProgressReader{
		expectedTaskID: taskID,
		result: pmstream.ProgressQueryResult{
			Rows: []pmstream.ProgressResult{
				{
					TaskID: taskID, Granularity: pmstream.GranularityDaily,
					WindowStart: start.Add(30 * 24 * time.Hour),
					WindowEnd:   end,
					Dimension:   pmstream.DimensionNetwork, DimensionKey: "network",
					MetricPath: "K1", MetricType: "kpi", Value: 42, Partial: true,
					ReceivedSlots: 4, ExpectedSlots: 24,
				},
				{
					TaskID: taskID, Granularity: pmstream.GranularityDaily,
					WindowStart: start.Add(30 * 24 * time.Hour),
					WindowEnd:   end,
					Dimension:   pmstream.DimensionDevice, DimensionKey: "device-1",
					MetricPath: "K1", MetricType: "kpi", Value: 999, Partial: true,
				},
				{
					TaskID: taskID, Granularity: pmstream.GranularityDaily,
					WindowStart: start.Add(30 * 24 * time.Hour),
					WindowEnd:   end,
					Dimension:   pmstream.DimensionNetwork, DimensionKey: "network",
					MetricPath: "K2", MetricType: "kpi", Value: 888, Partial: true,
				},
			},
			Periods: []pmstream.PeriodProgress{{
				Granularity: pmstream.GranularityDaily,
				WindowStart: start.Add(30 * 24 * time.Hour), WindowEnd: end,
				EntityKey: "network", ReceivedSlots: 4, ExpectedSlots: 24,
			}},
		},
	}
	service := &Service{networkRollups: reader}
	service.SetNetworkProgressReader(progress)

	snapshot, _, err := service.GetKPITimeSeriesSnapshotWithMetadata(
		context.Background(), []string{"K1"}, model.TechLTE,
		metrics.GranularityDaily, start, end,
	)

	require.NoError(t, err)
	require.Equal(t, "available", snapshot.ProgressState)
	require.Len(t, snapshot.Series["K1"], 2)
	require.Equal(t, jsonx.Float(10), snapshot.Series["K1"][0].Value)
	require.False(t, snapshot.Series["K1"][0].Partial)
	require.Equal(t, jsonx.Float(42), snapshot.Series["K1"][1].Value)
	require.True(t, snapshot.Series["K1"][1].Partial)
	require.Len(t, snapshot.PeriodProgress, 1)
	require.EqualValues(t, 4, snapshot.PeriodProgress[0].ReceivedSlots)
	require.EqualValues(t, 24, snapshot.PeriodProgress[0].ExpectedSlots)
	require.NotContains(t, snapshot.Series, "K2")
}

func TestGetKPITimeSeriesSnapshotKeepsPublishedSeriesWhenProgressUnavailable(t *testing.T) {
	start := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(31 * 24 * time.Hour)
	taskID, ok := pmstream.BuiltinNetworkTaskID("lte")
	require.True(t, ok)
	reader := &recordingNetworkRollupReader{points: []NetworkRollupPoint{{
		Technology: model.TechLTE, MetricPath: "K1",
		Granularity: metrics.GranularityDaily,
		WindowStart: start, WindowEnd: start.Add(24 * time.Hour),
		Value: jsonx.Float(10), Complete: true,
	}}}
	service := &Service{networkRollups: reader}
	service.SetNetworkProgressReader(&fixedNetworkProgressReader{
		expectedTaskID: taskID,
		err:            errors.New("redis timeout"),
	})

	snapshot, _, err := service.GetKPITimeSeriesSnapshotWithMetadata(
		context.Background(), []string{"K1"}, model.TechLTE,
		metrics.GranularityDaily, start, end,
	)

	require.NoError(t, err)
	require.Equal(t, "unavailable", snapshot.ProgressState)
	require.Len(t, snapshot.Series["K1"], 1)
	require.Equal(t, jsonx.Float(10), snapshot.Series["K1"][0].Value)
	require.Empty(t, snapshot.PeriodProgress)
}

func TestGetKPITimeSeries_RoutesRequestedGranularity(t *testing.T) {
	start := time.Date(2026, 7, 13, 0, 0, 0, 0, time.UTC)
	end := start.Add(12 * 7 * 24 * time.Hour)

	for _, granularity := range []metrics.Granularity{
		metrics.GranularityHourly,
		metrics.GranularityDaily,
		metrics.GranularityWeekly,
	} {
		t.Run(string(granularity), func(t *testing.T) {
			reader := &recordingNetworkRollupReader{}
			service := &Service{networkRollups: reader}

			_, err := service.GetKPITimeSeries(
				context.Background(), []string{"K1"}, model.TechLTE, granularity, start, end,
			)

			require.NoError(t, err)
			require.Len(t, reader.queries, 1)
			require.Equal(t, granularity, reader.queries[0].Granularity)
			require.Equal(t, model.TechLTE, reader.queries[0].Technology)
			require.Equal(t, []string{"K1"}, reader.queries[0].MetricPaths)
		})
	}
}

func TestGetKPITimeSeriesReturnsPublishedValuesWithoutFillingMissingBuckets(t *testing.T) {
	start := time.Date(2026, 7, 13, 0, 0, 0, 0, time.UTC)
	reader := &recordingNetworkRollupReader{points: []NetworkRollupPoint{
		{
			Technology: model.TechNR, MetricPath: "K1",
			Granularity: metrics.GranularityWeekly,
			WindowStart: start.Add(7 * 24 * time.Hour), WindowEnd: start.Add(14 * 24 * time.Hour),
			Value: jsonx.Float(42), Complete: false, MissingSlots: 1,
		},
	}}
	service := &Service{networkRollups: reader}

	got, err := service.GetKPITimeSeries(
		context.Background(), []string{"K1", "K2"}, model.TechNR,
		metrics.GranularityWeekly, start, start.Add(12*7*24*time.Hour),
	)

	require.NoError(t, err)
	require.Len(t, got["K1"], 1)
	require.Equal(t, jsonx.Float(42), got["K1"][0].Value)
	require.Empty(t, got["K2"], "missing buckets and metrics must remain empty")
	require.Len(t, reader.queries, 1, "reader errors or gaps must never trigger a fallback query")
}

func TestCloneKPITimeSeriesResponsePreservesEmptyArrays(t *testing.T) {
	source := KPITimeSeriesResponse{
		"K_WITH_DATA": {{Time: time.Now(), Value: jsonx.Float(42)}},
		"K_NO_DATA":   {},
	}

	cloned := cloneKPITimeSeriesResponse(source)

	require.NotNil(t, cloned["K_NO_DATA"], "JSON contract requires [] instead of null")
	require.Empty(t, cloned["K_NO_DATA"])
	require.Equal(t, source["K_WITH_DATA"], cloned["K_WITH_DATA"])
}

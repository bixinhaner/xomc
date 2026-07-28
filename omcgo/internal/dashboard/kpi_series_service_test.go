package dashboard

import (
	"context"
	"testing"
	"time"

	"github.com/omcgo/omcgo/internal/core/jsonx"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/pm/metrics"
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

package dashboard

import (
	"context"
	"testing"
	"time"

	pmaggregator "github.com/omcgo/omcgo/internal/pm/aggregator"
	"github.com/omcgo/omcgo/internal/pm/metrics"
	"github.com/stretchr/testify/require"
)

type recordingDashboardKPIAggregator struct {
	requests []pmaggregator.QueryRequest
	rows     []pmaggregator.Row
}

func (a *recordingDashboardKPIAggregator) Query(_ context.Context, request pmaggregator.QueryRequest) ([]pmaggregator.Row, error) {
	a.requests = append(a.requests, request)
	return a.rows, nil
}

func TestFetchNetworkKCodeSeries_DoesNotQuery15MinTailWhenHourlyBucketMissing(t *testing.T) {
	aggregator := &recordingDashboardKPIAggregator{}
	service := &Service{pmAggregator: aggregator}
	start := time.Date(2026, 7, 13, 0, 0, 0, 0, time.UTC)
	end := start.Add(2*time.Hour + 30*time.Minute)

	points, err := service.fetchNetworkKCodeSeries(context.Background(), []string{"K1"}, "", start, end)

	require.NoError(t, err)
	require.Empty(t, points)
	require.Len(t, aggregator.requests, 1)
	require.Equal(t, metrics.GranularityHourly, aggregator.requests[0].Granularity)
}

func TestGetKPITimeSeries_RoutesRequestedGranularity(t *testing.T) {
	start := time.Date(2026, 7, 13, 0, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)

	for _, granularity := range []metrics.Granularity{
		metrics.GranularityHourly,
		metrics.GranularityDaily,
	} {
		t.Run(string(granularity), func(t *testing.T) {
			aggregator := &recordingDashboardKPIAggregator{}
			service := &Service{pmAggregator: aggregator}

			_, err := service.GetKPITimeSeries(
				context.Background(), []string{"K1"}, "", granularity, start, end,
			)

			require.NoError(t, err)
			require.Len(t, aggregator.requests, 1)
			require.Equal(t, granularity, aggregator.requests[0].Granularity)
			require.Equal(t, pmaggregator.DimensionNetwork, aggregator.requests[0].Dimension)
		})
	}
}

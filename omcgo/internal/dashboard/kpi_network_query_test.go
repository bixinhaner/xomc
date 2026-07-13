package dashboard

import (
	"encoding/json"
	"math"
	"testing"
	"time"

	"github.com/omcgo/omcgo/internal/core/jsonx"
	"github.com/omcgo/omcgo/internal/core/model"
	pmaggregator "github.com/omcgo/omcgo/internal/pm/aggregator"
	"github.com/omcgo/omcgo/internal/pm/metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildNetworkKPISeriesRequest_Basic(t *testing.T) {
	start := time.Date(2026, 6, 13, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 6, 13, 12, 0, 0, 0, time.UTC)
	codes := []string{"K900010015", "C000060216"}

	req := buildNetworkKPIHourlySeriesRequest(codes, "", start, end)

	assert.Equal(t, metrics.GranularityHourly, req.Granularity)
	assert.Equal(t, pmaggregator.DimensionNetwork, req.Dimension)
	assert.Equal(t, []string{"K900010015", "C000060216"}, req.MetricPaths)
	assert.Equal(t, start, req.StartTime)
	assert.Equal(t, end, req.EndTime)
	assert.Zero(t, req.Limit)
	assert.Zero(t, req.Offset)

	codes[0] = "mutated"
	assert.Equal(t, []string{"K900010015", "C000060216"}, req.MetricPaths)

	dailyReq := buildNetworkKPIDailySeriesRequest([]string{"K900010015"}, "", start, end)
	assert.Equal(t, metrics.GranularityDaily, dailyReq.Granularity)
	assert.Equal(t, pmaggregator.DimensionNetwork, dailyReq.Dimension)
}

func TestBuildNetworkKPISeriesRequest_FiltersSelectedTechnology(t *testing.T) {
	start := time.Date(2026, 7, 13, 0, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)

	req := buildNetworkKPIDailySeriesRequest(
		[]string{"KGSM0101"},
		model.TechGSM,
		start,
		end,
	)

	assert.Equal(t, []string{string(model.TechGSM)}, req.Technologies)
}

func TestNetworkRowsToDailySeriesPoints_UsesBucketStart(t *testing.T) {
	start := time.Date(2026, 6, 12, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 6, 13, 0, 0, 0, 0, time.UTC)
	points := networkRowsToDailySeriesPoints([]pmaggregator.Row{{
		MetricPath:  "K900010015",
		MetricValue: jsonx.Float(3.23),
		Time:        start,
		EndTime:     end,
	}}, start, end)

	require.Len(t, points, 1)
	assert.Equal(t, start, points[0].time)
}

func TestNetworkRowsToSeriesPoints_UsesBucketStartForChartPoint(t *testing.T) {
	start := time.Date(2026, 6, 13, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 6, 13, 17, 10, 0, 0, time.UTC)
	bucketStart := time.Date(2026, 6, 13, 16, 0, 0, 0, time.UTC)
	bucketEnd := time.Date(2026, 6, 13, 17, 0, 0, 0, time.UTC)

	points := networkRowsToSeriesPoints([]pmaggregator.Row{{
		MetricPath:  "K900010015",
		MetricValue: jsonx.Float(42),
		Time:        bucketStart,
		EndTime:     bucketEnd,
	}}, start, end)

	require.Len(t, points, 1)
	assert.Equal(t, bucketStart, points[0].time)
	assert.Equal(t, jsonx.Float(42), points[0].value)
}

func TestNetworkRowsToSeriesPoints_UsesHalfOpenWindow(t *testing.T) {
	start := time.Date(2026, 6, 12, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 6, 13, 0, 0, 0, 0, time.UTC)
	points := networkRowsToSeriesPoints([]pmaggregator.Row{
		{MetricPath: "K", Time: start, EndTime: start.Add(time.Hour)},
		{MetricPath: "K", Time: end, EndTime: end.Add(time.Hour)},
	}, start, end)

	require.Len(t, points, 1)
	assert.Equal(t, start, points[0].time)
}

func TestDashboardKPITrendCompareWindow_UsesDisplayEndBoundary(t *testing.T) {
	now := time.Date(2026, 6, 13, 17, 10, 0, 0, time.UTC)

	compareWith, start, end := dashboardKPITrendCompareWindow(now, "yesterday")
	assert.Equal(t, "yesterday", compareWith)
	assert.Equal(t, time.Date(2026, 6, 12, 0, 0, 0, 0, time.UTC), start)
	assert.Equal(t, time.Date(2026, 6, 13, 0, 0, 0, 0, time.UTC), end)

	compareWith, start, end = dashboardKPITrendCompareWindow(now, "last_week")
	assert.Equal(t, "last_week", compareWith)
	assert.Equal(t, time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC), start)
	assert.Equal(t, time.Date(2026, 6, 8, 0, 0, 0, 0, time.UTC), end)

	loc := time.FixedZone("UTC+8", 8*60*60)
	now = time.Date(2026, 6, 13, 17, 10, 0, 0, loc)
	_, start, end = dashboardKPITrendCompareWindow(now, "yesterday")
	assert.Equal(t, time.Date(2026, 6, 12, 0, 0, 0, 0, loc), start)
	assert.Equal(t, time.Date(2026, 6, 13, 0, 0, 0, 0, loc), end)
}

func TestKPITimeSeriesEntry_NullMetricValueSerializesAsJSONNull(t *testing.T) {
	body, err := json.Marshal(KPITimeSeriesEntry{
		Time:  time.Date(2026, 7, 7, 10, 0, 0, 0, time.UTC),
		Value: jsonxFloatNaN(),
	})
	require.NoError(t, err)

	assert.JSONEq(t, `{"time":"2026-07-07T10:00:00Z","value":null}`, string(body))
}

func jsonxFloatNaN() jsonx.Float {
	return jsonx.Float(math.NaN())
}

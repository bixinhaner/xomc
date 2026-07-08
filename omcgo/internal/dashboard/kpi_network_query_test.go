package dashboard

import (
	"encoding/json"
	"math"
	"testing"
	"time"

	"github.com/omcgo/omcgo/internal/core/jsonx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildNetworkKPISeriesQuery_Basic(t *testing.T) {
	start := time.Date(2026, 6, 13, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 6, 13, 12, 0, 0, 0, time.UTC)

	q, args, err := buildNetworkKPISeriesQuery([]string{"K900010015", "C000060216"}, start, end)
	require.NoError(t, err)

	assert.Contains(t, q, "FROM pm_metrics_hourly_cagg")
	assert.Contains(t, q, "CASE MIN(statis_type)")
	assert.Contains(t, q, "WHEN 'sum' THEN SUM(sum_val)")
	assert.Contains(t, q, "WHEN 'avg' THEN AVG(avg_val)")
	assert.NotContains(t, q, "sample_count")

	// 编号过滤。
	assert.Contains(t, q, "metric_path = ANY($1)")
	assert.Contains(t, q, "bucket_time >= $2")
	assert.Contains(t, q, "bucket_time <= $3")
	assert.Contains(t, q, "ORDER BY metric_path, bucket_time ASC")

	require.Len(t, args, 3)
	assert.Equal(t, []string{"K900010015", "C000060216"}, args[0])
	assert.Equal(t, start, args[1])
	assert.Equal(t, end, args[2])
}

func TestKPITimeSeriesEntry_NullMetricValueSerializesAsJSONNull(t *testing.T) {
	body, err := json.Marshal(KPITimeSeriesEntry{
		Time:  "2026-07-07T10:00:00Z",
		Value: jsonxFloatNaN(),
	})
	require.NoError(t, err)

	assert.JSONEq(t, `{"time":"2026-07-07T10:00:00Z","value":null}`, string(body))
}

func jsonxFloatNaN() jsonx.Float {
	return jsonx.Float(math.NaN())
}

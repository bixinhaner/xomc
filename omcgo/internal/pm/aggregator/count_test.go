package aggregator

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/pm/metrics"
)

// T-0194 C2：Aggregator.Count 返回与 Query 同过滤下命中真实总数（忽略 Limit/Offset）。

// device 维度：直查 COUNT(*)（无聚合），WHERE 含过滤项、不带 LIMIT/OFFSET/ORDER BY。
func Test_Count_Device(t *testing.T) {
	var gotSQL string
	db := &stubDB{}
	db.queryRowFn = func(ctx context.Context, sql string, args ...any) pgx.Row {
		gotSQL = sql
		return countRow{n: 1234}
	}
	a := New(db, nil, nil)
	n, err := a.Count(context.Background(), QueryRequest{
		Granularity: metrics.GranularityHourly,
		Dimension:   DimensionDevice,
		DeviceSNs:   []string{"SN-1"},
		Limit:       100,
		Offset:      50,
	})
	require.NoError(t, err)
	assert.Equal(t, 1234, n)
	assert.True(t, strings.HasPrefix(strings.TrimSpace(gotSQL), "SELECT COUNT(*)"), gotSQL)
	assert.Contains(t, gotSQL, "pm_metrics_hourly")
	assert.NotContains(t, gotSQL, "LIMIT")
	assert.NotContains(t, gotSQL, "OFFSET")
	assert.NotContains(t, gotSQL, "ORDER BY")
}

// device_group 维度：GROUP BY 子查询包成 COUNT(*) FROM (...) sub，数的是分组数。
func Test_Count_DeviceGroup_SubqueryCount(t *testing.T) {
	var gotSQL string
	db := &stubDB{}
	db.queryRowFn = func(ctx context.Context, sql string, args ...any) pgx.Row {
		gotSQL = sql
		return countRow{n: 7}
	}
	a := New(db, nil, nil)
	n, err := a.Count(context.Background(), QueryRequest{
		Granularity:    metrics.GranularityDaily,
		Dimension:      DimensionDeviceGroup,
		DeviceGroupIDs: []uuid.UUID{uuid.New()},
		Limit:          100,
	})
	require.NoError(t, err)
	assert.Equal(t, 7, n)
	assert.Contains(t, gotSQL, "SELECT COUNT(*) FROM (")
	assert.Contains(t, gotSQL, "GROUP BY device_group_id")
	assert.Contains(t, gotSQL, ") sub")
}

// network 维度：GROUP BY metric_path/granularity/time 子查询计数。
func Test_Count_Network_SubqueryCount(t *testing.T) {
	var gotSQL string
	db := &stubDB{}
	db.queryRowFn = func(ctx context.Context, sql string, args ...any) pgx.Row {
		gotSQL = sql
		return countRow{n: 42}
	}
	a := New(db, nil, nil)
	n, err := a.Count(context.Background(), QueryRequest{
		Granularity: metrics.GranularityHourly,
		Dimension:   DimensionNetwork,
		Limit:       5000,
	})
	require.NoError(t, err)
	assert.Equal(t, 42, n)
	assert.Contains(t, gotSQL, "SELECT COUNT(*) FROM (")
	assert.Contains(t, gotSQL, "GROUP BY metric_path, granularity, time")
}

func Test_DiscoverObjectLDNs_IgnoresRequestedMetricPath(t *testing.T) {
	var gotSQL string
	db := &stubDB{}
	db.queryFn = func(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
		gotSQL = sql
		return &fakeRows{rows: [][]any{{"Cellid=1"}, {"Cellid=2"}}}, nil
	}
	a := New(db, nil, nil)
	start := time.Date(2026, 7, 14, 14, 45, 0, 0, time.UTC)

	ldns, err := a.DiscoverObjectLDNs(context.Background(), QueryRequest{
		Granularity: metrics.Granularity15Min,
		Dimension:   DimensionDevice,
		DeviceSNs:   []string{"SN-1"},
		MetricPaths: []string{"K-MISSING"},
		StartTime:   start,
		EndTime:     start.Add(3 * time.Hour),
		Limit:       5000,
	})

	require.NoError(t, err)
	assert.Equal(t, []string{"Cellid=1", "Cellid=2"}, ldns)
	assert.Contains(t, gotSQL, "SELECT DISTINCT object_ldn")
	assert.Contains(t, gotSQL, "FROM pm_metrics")
	assert.Contains(t, gotSQL, "object_ldn <> ''")
	assert.Contains(t, gotSQL, "device_sn")
	assert.Contains(t, gotSQL, "ORDER BY object_ldn")
	assert.NotContains(t, gotSQL, "metric_path")
	assert.NotContains(t, gotSQL, "LIMIT")
}

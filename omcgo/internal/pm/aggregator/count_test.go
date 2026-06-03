package aggregator

import (
	"context"
	"strings"
	"testing"

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

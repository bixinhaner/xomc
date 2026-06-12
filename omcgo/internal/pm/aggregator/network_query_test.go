package aggregator

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/pm/metrics"
)

// networkRow 是 queryNetworkTable SELECT 的一行（path, type, value, statis, gran, time×4）。
// 注意：network 维度 SELECT 不含任何实体键列（无 device/product/object_ldn）。
func networkRow(path, mtype string, val float64, statis, gran string, t time.Time) []any {
	return []any{path, mtype, val, statis, gran, t, t, t, t}
}

// Test_queryNetworkTable_SQLShape：断言全网汇总 SQL 结构：
// 只按 metric_path/granularity/time GROUP BY（无 object_ldn、无 product_id 实体键），
// 走 CASE statis_type 路由算子。
func Test_queryNetworkTable_SQLShape(t *testing.T) {
	db := &recordingDB{
		// T-0191：不再有 precheck；call#0 = 主聚合查询（只聚 counter）
		results: []pgx.Rows{&fakeRows{}},
	}
	a := New(db, nil, nil)
	_, err := a.queryNetworkTable(context.Background(), "pm_metrics_hourly", QueryRequest{
		Granularity: metrics.GranularityHourly,
		MetricPaths: []string{"C000060011"},
		MetricType:  counterType(),
	})
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(db.sqls), 1, "主查询")

	sql := db.sqls[0]
	// 分组键仅 metric_path/granularity/time（全网无实体键）
	assert.Contains(t, sql, "GROUP BY metric_path, granularity, time")
	assert.NotContains(t, sql, "object_ldn", "全网汇总不带 object_ldn 实体键")
	assert.NotContains(t, sql, "product_id", "全网汇总不带 product_id 分组")
	assert.NotContains(t, sql, "device_sn", "全网汇总不带 device_sn 分组")
	// statis_type 路由算子在
	assert.Contains(t, sql, "WHEN 'sum' THEN SUM(metric_value)")
}

// Test_queryNetworkTable_TechnologyFilter：制式过滤通过 (device_oui, device_sn) 子查询收口。
func Test_queryNetworkTable_TechnologyFilter(t *testing.T) {
	db := &recordingDB{results: []pgx.Rows{&fakeRows{}}}
	a := New(db, nil, nil)
	_, err := a.queryNetworkTable(context.Background(), "pm_metrics_hourly", QueryRequest{
		Granularity:  metrics.GranularityHourly,
		MetricPaths:  []string{"C000060011"},
		MetricType:   counterType(),
		Technologies: []string{"lte"},
	})
	require.NoError(t, err)
	sql := db.sqls[0]
	assert.True(t,
		strings.Contains(sql, "SELECT oui, serial_number FROM device_dim WHERE technology"),
		"制式过滤走 device_dim 影子表子查询收口（跨库分离）")
	// args 含制式值
	foundTech := false
	for _, a := range db.argsLog[0] {
		if ss, ok := a.([]string); ok {
			for _, s := range ss {
				if s == "lte" {
					foundTech = true
				}
			}
		}
	}
	assert.True(t, foundTech, "args 含制式过滤值 lte")
}

// Test_queryNetworkTable_AggregatesToOneBus：全网每指标每时间桶汇总成一条总线。
// 结果行无 object_ldn、DeviceSN='AGGREGATED'。
func Test_queryNetworkTable_AggregatesToOneBus(t *testing.T) {
	now := time.Date(2026, 5, 30, 10, 0, 0, 0, time.UTC)
	db := &recordingDB{
		results: []pgx.Rows{
			// 全网所有设备/小区的 C000060011 在 10:00 桶汇总成一条（值=全网合计）
			&fakeRows{rows: [][]any{
				networkRow("C000060011", "counter", 99999, "sum", "hourly", now),
			}},
		},
	}
	a := New(db, nil, nil)
	rows, err := a.queryNetworkTable(context.Background(), "pm_metrics_hourly", QueryRequest{
		Granularity: metrics.GranularityHourly,
		MetricPaths: []string{"C000060011"},
		MetricType:  counterType(),
	})
	require.NoError(t, err)
	require.Len(t, rows, 1, "全网每指标每时间桶一条总线")
	assert.Equal(t, "AGGREGATED", rows[0].DeviceSN)
	assert.Equal(t, "", rows[0].DeviceOUI)
	assert.Nil(t, rows[0].ObjectLDN, "全网总线无实体键 object_ldn")
	assert.Equal(t, float64(99999), rows[0].MetricValue)
	require.NotNil(t, rows[0].StatisType)
	assert.Equal(t, metrics.StatisType("sum"), *rows[0].StatisType)
}

// Test_queryNetworkTable_EmptyData：无数据不报错，返回空。
func Test_queryNetworkTable_EmptyData(t *testing.T) {
	db := &recordingDB{
		results: []pgx.Rows{
			&fakeRows{}, // 主查询无行
		},
	}
	a := New(db, nil, nil)
	rows, err := a.queryNetworkTable(context.Background(), "pm_metrics_hourly", QueryRequest{
		Granularity: metrics.GranularityHourly,
		MetricPaths: []string{"C000060011"},
		MetricType:  counterType(),
	})
	require.NoError(t, err, "空数据不报错")
	assert.Empty(t, rows)
}

// Test_Query_NetworkDimension_EndToEnd：经 Query 入口走 network 分支并完成 DisplayName 回填。
// T-0191：Query 入口现先调 resolveKPIMetadata（call#0，无派生 KPI 返空），再调 network 主查询（call#1）。
func Test_Query_NetworkDimension_EndToEnd(t *testing.T) {
	now := time.Date(2026, 5, 30, 10, 0, 0, 0, time.UTC)
	db := &recordingDB{
		results: []pgx.Rows{
			&fakeRows{}, // resolveKPIMetadata：无派生 KPI
			&fakeRows{rows: [][]any{
				networkRow("C000060011", "counter", 99999, "sum", "hourly", now),
			}},
		},
	}
	a := New(db, nil, nil)
	rows, err := a.Query(context.Background(), QueryRequest{
		Granularity: metrics.GranularityHourly,
		Dimension:   DimensionNetwork,
		MetricPaths: []string{"C000060011"},
		MetricType:  counterType(),
	})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "AGGREGATED", rows[0].DeviceSN)
	assert.Equal(t, "C000060011", rows[0].DisplayName, "counter 行 DisplayName=metric_path")
}

// Test_SelectTable_NetworkDimension：network 维度走 device 维度同源表（全粒度，含 15min）。
func Test_SelectTable_NetworkDimension(t *testing.T) {
	cases := []struct {
		gran metrics.Granularity
		want string
	}{
		{metrics.Granularity15Min, "pm_metrics"},
		{metrics.GranularityHourly, "pm_metrics_hourly"},
		{metrics.GranularityDaily, "pm_metrics_daily"},
		{metrics.GranularityWeekly, "pm_metrics_weekly"},
		{metrics.GranularityMonthly, "pm_metrics_monthly"},
	}
	for _, c := range cases {
		t.Run(string(c.gran), func(t *testing.T) {
			got, err := SelectTable(c.gran, DimensionNetwork)
			require.NoError(t, err)
			assert.Equal(t, c.want, got)
		})
	}
}

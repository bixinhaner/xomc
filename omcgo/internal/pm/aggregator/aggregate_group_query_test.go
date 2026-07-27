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

// aggGroupRow 是 queryAggregateGroupTable SELECT 的一行（单条聚合，无 object_ldn）：
// metric_path, metric_type, metric_value, statis_type, granularity, time×4。
func aggGroupRow(path, mtype string, val float64, statis, gran string, t time.Time) []any {
	return []any{path, mtype, val, statis, gran, t, t, t, t}
}

// Test_queryAggregateGroupTable_SQLShape：临时聚合组改为单条聚合后，
// 生成的 SQL 分组键只含 metric_path/granularity/time，SELECT 与 GROUP BY 都不再含 object_ldn。
func Test_queryAggregateGroupTable_SQLShape(t *testing.T) {
	db := &recordingDB{
		results: []pgx.Rows{&fakeRows{}}, // call#0 = 主聚合查询
	}
	a := New(db, nil, nil)
	_, err := a.queryAggregateGroupTable(context.Background(), "pm_metrics_hourly", QueryRequest{
		Granularity: metrics.GranularityHourly,
		Dimension:   DimensionAggregateGroup,
		DeviceSNs:   []string{"SN1", "SN2"},
		MetricPaths: []string{"C1"},
		StartTime:   time.Unix(0, 0),
		EndTime:     time.Unix(1<<31, 0),
	})
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(db.sqls), 1, "主查询")

	sql := db.sqls[0]
	assert.Contains(t, sql,
		`DISTINCT ON (device_oui, device_sn, metric_path, granularity, "time", object_ldn)`,
		"聚合前应按设备自然键保留最新补报")
	assert.Contains(t, sql, "ingest_time DESC")
	// 分组键只剩三列。
	gb := sql[strings.LastIndex(sql, "GROUP BY"):]
	assert.Contains(t, gb, "metric_path")
	assert.Contains(t, gb, "granularity")
	assert.Contains(t, gb, "time")
	assert.NotContains(t, gb, "object_ldn",
		"物理目标查询可在内层保留实体键，外层聚合不得按 object_ldn 分组")
}

// Test_queryAggregateGroupTable_CollapsesAllCells：多设备多小区在 DB 端按
// metric_path×granularity×time GROUP BY 折叠为一行（DB 不再带 object_ldn 分组），
// 断言 Go 侧承接为单行、ObjectLDN 置空、合计值正确、设备身份为聚合占位。
func Test_queryAggregateGroupTable_CollapsesAllCells(t *testing.T) {
	now := time.Date(2026, 5, 30, 10, 0, 0, 0, time.UTC)
	db := &recordingDB{
		results: []pgx.Rows{
			// 两台设备 × 两小区（共 4 条样本，各 100）同一 metric×gran×time，
			// SQL GROUP BY 去 object_ldn 后折叠成 1 行，sum=400。
			&fakeRows{rows: [][]any{
				aggGroupRow("C1", "counter", 400, "sum", "hourly", now),
			}},
		},
	}
	a := New(db, nil, nil)
	rows, err := a.queryAggregateGroupTable(context.Background(), "pm_metrics_hourly", QueryRequest{
		Granularity: metrics.GranularityHourly,
		Dimension:   DimensionAggregateGroup,
		DeviceSNs:   []string{"SN1", "SN2"},
		MetricPaths: []string{"C1"},
	})
	require.NoError(t, err)
	require.Len(t, rows, 1, "所有设备所有小区合成一条")
	assert.Nil(t, rows[0].ObjectLDN, "单条聚合无小区 → ObjectLDN 置 nil")
	assert.Equal(t, "AGGREGATED", rows[0].DeviceSN)
	assert.Equal(t, "", rows[0].DeviceOUI)
	assert.Equal(t, float64(400), float64(rows[0].MetricValue), "全部设备全部小区合计")
	require.NotNil(t, rows[0].StatisType)
	assert.Equal(t, metrics.StatisType("sum"), *rows[0].StatisType)
}

// Test_queryAggregateGroupTable_ScanErrorPropagates：DB Query 失败时错误向上传播（失败路径）。
func Test_queryAggregateGroupTable_ScanErrorPropagates(t *testing.T) {
	db := &recordingDB{
		errs: []error{assertAnErr},
	}
	a := New(db, nil, nil)
	_, err := a.queryAggregateGroupTable(context.Background(), "pm_metrics_hourly", QueryRequest{
		Granularity: metrics.GranularityHourly,
		Dimension:   DimensionAggregateGroup,
		MetricPaths: []string{"C1"},
	})
	require.Error(t, err, "DB 查询失败应向上传播")
	assert.Contains(t, err.Error(), "aggregate_group")
}

var assertAnErr = errAggGroupQuery{}

type errAggGroupQuery struct{}

func (errAggGroupQuery) Error() string { return "boom" }

package aggregator

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/pm/metrics"
)

// recordingDB 记录每次 Query 的 SQL/args，并按调用序返回预置结果。
type recordingDB struct {
	sqls       []string
	argsLog    [][]any
	results    []pgx.Rows // 按调用序返回
	errs       []error
	rowResults []int // QueryRow 按调用序返回的 count
}

func (r *recordingDB) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	return pgconn.NewCommandTag(""), nil
}

// QueryRow 记录 SQL/args 并返回预置 count（rowResults 按调用序，缺省 0）。
func (r *recordingDB) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	idx := len(r.sqls)
	r.sqls = append(r.sqls, sql)
	r.argsLog = append(r.argsLog, args)
	n := 0
	if idx < len(r.rowResults) {
		n = r.rowResults[idx]
	}
	return countRow{n: n, sql: sql, args: args}
}

func (r *recordingDB) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	idx := len(r.sqls)
	r.sqls = append(r.sqls, sql)
	r.argsLog = append(r.argsLog, args)
	if idx < len(r.errs) && r.errs[idx] != nil {
		return nil, r.errs[idx]
	}
	if idx < len(r.results) {
		return r.results[idx], nil
	}
	return &fakeRows{}, nil
}

// bandRow 是 queryBandTable SELECT 的一行（band, path, type, value, statis, gran, time×4）。
func bandRow(band, path, mtype string, val float64, statis, gran string, t time.Time) []any {
	return []any{band, path, mtype, val, statis, gran, t, t, t, t}
}

// counterType 是 band 测试用的指标类型过滤。
func counterType() *metrics.MetricType {
	ct := metrics.MetricTypeCounter
	return &ct
}

// Test_queryBandTable_SQLShape：断言生成 SQL 含 band 聚合关键结构（KPI/时序库物理分离后）：
// 直接 JOIN 本库影子表 device_dim + cell_band_dim（小区→band 映射已由 worker 同步任务从主库
// device_parameters 的 CellIdentity/FreqBandIndicator 配对派生入 cell_band_dim，不再现拼 CTE）、
// object_ldn 的 Cellid 正则抽取、INNER JOIN（未命中跳过）、按 band GROUP BY。
func Test_queryBandTable_SQLShape(t *testing.T) {
	db := &recordingDB{
		// T-0191：不再有 precheck 调用；call#0 = 主聚合查询（只聚 counter）
		results: []pgx.Rows{&fakeRows{}},
	}
	a := New(db, nil, nil)
	mt := counterType()
	_, err := a.queryBandTable(context.Background(), "pm_metrics_hourly", QueryRequest{
		Granularity: metrics.GranularityHourly,
		MetricPaths: []string{"L.Cell.A"},
		MetricType:  mt,
	})
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(db.sqls), 1, "主查询")

	sql := db.sqls[0]
	// 跨库分离后小区→band 映射改读时序库影子表，不再有 device_parameters 现拼 CTE / 路径后缀。
	assert.NotContains(t, sql, "WITH cell_band AS", "影子表替代 CTE，不应再有 cell_band CTE")
	for _, a := range db.argsLog[0] {
		if s, ok := a.(string); ok {
			assert.False(t, strings.HasSuffix(s, ".CellConfig.LTE.RAN.RF.FreqBandIndicator"),
				"不再以 band 路径后缀作参数")
			assert.False(t, strings.HasSuffix(s, ".CellConfig.LTE.RAN.Common.CellIdentity"),
				"不再以 CellIdentity 路径后缀作参数")
		}
	}
	// 直接 JOIN 影子表 device_dim + cell_band_dim
	assert.Contains(t, sql, "JOIN device_dim d")
	assert.Contains(t, sql, "JOIN cell_band_dim cb")
	assert.NotContains(t, sql, "LEFT JOIN cell_band_dim")
	// cell_band_dim.cell_id 与 PM object_ldn 抽出的 Cellid 等值
	assert.Contains(t, sql, "substring(m.object_ldn FROM 'Cellid=([0-9]+)')")
	// 按 band 分组
	assert.Contains(t, sql, "GROUP BY cb.band")
}

// Test_queryBandTable_JoinHit_LabelsBand：join 命中 → 结果行 ObjectLDN='Band=<值>'。
func Test_queryBandTable_JoinHit_LabelsBand(t *testing.T) {
	now := time.Date(2026, 5, 30, 10, 0, 0, 0, time.UTC)
	db := &recordingDB{
		results: []pgx.Rows{
			&fakeRows{rows: [][]any{
				bandRow("42", "L.Cell.A", "counter", 1500, "sum", "hourly", now),
			}},
		},
	}
	a := New(db, nil, nil)
	rows, err := a.queryBandTable(context.Background(), "pm_metrics_hourly", QueryRequest{
		Granularity: metrics.GranularityHourly,
		MetricPaths: []string{"L.Cell.A"},
		MetricType:  counterType(),
	})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.NotNil(t, rows[0].ObjectLDN)
	assert.Equal(t, "Band=42", *rows[0].ObjectLDN, "命中频段 → ObjectLDN 标 Band=42")
	assert.Equal(t, "AGGREGATED", rows[0].DeviceSN)
	assert.Equal(t, float64(1500), rows[0].MetricValue)
	require.NotNil(t, rows[0].StatisType)
	assert.Equal(t, metrics.StatisType("sum"), *rows[0].StatisType)
}

// Test_queryBandTable_JoinMiss_SkipsCell：小区无频段参数 → INNER JOIN 不产行（兜底=跳过）。
// 这里以「DB 返回空结果集」模拟 join 未命中（SQL 侧 INNER JOIN 已剔除该小区）。
func Test_queryBandTable_JoinMiss_SkipsCell(t *testing.T) {
	db := &recordingDB{
		results: []pgx.Rows{
			&fakeRows{}, // 主查询无命中行
		},
	}
	a := New(db, nil, nil)
	rows, err := a.queryBandTable(context.Background(), "pm_metrics_hourly", QueryRequest{
		Granularity: metrics.GranularityHourly,
		MetricPaths: []string{"L.Cell.A"},
		MetricType:  counterType(),
	})
	require.NoError(t, err)
	assert.Empty(t, rows, "无频段参数的小区被 INNER JOIN 跳过，不产 band 行")
}

// Test_queryBandTable_MultiCellSameBand：多小区聚到同一 band → DB 已 GROUP BY band 折叠成一行。
// 断言 Go 侧正确承接该聚合行（值=多小区合计 sum），并标 Band。
func Test_queryBandTable_MultiCellSameBand(t *testing.T) {
	now := time.Date(2026, 5, 30, 10, 0, 0, 0, time.UTC)
	db := &recordingDB{
		results: []pgx.Rows{
			// 两小区（111172245 + 111172145）同 band=42，SQL GROUP BY 后折叠为 1 行，
			// metric_value = 两小区 sum 合计（800+700）。
			&fakeRows{rows: [][]any{
				bandRow("42", "L.Cell.A", "counter", 1500, "sum", "hourly", now),
			}},
		},
	}
	a := New(db, nil, nil)
	rows, err := a.queryBandTable(context.Background(), "pm_metrics_hourly", QueryRequest{
		Granularity: metrics.GranularityHourly,
		MetricPaths: []string{"L.Cell.A"},
		MetricType:  counterType(),
	})
	require.NoError(t, err)
	require.Len(t, rows, 1, "多小区同 band 经 SQL GROUP BY 折叠为一行")
	assert.Equal(t, "Band=42", *rows[0].ObjectLDN)
	assert.Equal(t, float64(1500), rows[0].MetricValue, "同 band 多小区合计")
}

// Test_Query_BandDimension_EndToEnd：经 Query 入口（不再返 ErrBandNotImplemented），
// 走 band 分支并完成 DisplayName 回填。
// T-0191：Query 入口现先调 resolveKPIMetadata（call#0，无派生 KPI 返空），再调 band 主查询（call#1）。
func Test_Query_BandDimension_EndToEnd(t *testing.T) {
	now := time.Date(2026, 5, 30, 10, 0, 0, 0, time.UTC)
	db := &recordingDB{
		results: []pgx.Rows{
			&fakeRows{}, // resolveKPIMetadata：无派生 KPI
			&fakeRows{rows: [][]any{
				bandRow("42", "L.Cell.A", "counter", 1500, "sum", "hourly", now),
			}},
		},
	}
	a := New(db, nil, nil)
	rows, err := a.Query(context.Background(), QueryRequest{
		Granularity: metrics.GranularityHourly,
		Dimension:   DimensionBand,
		MetricPaths: []string{"L.Cell.A"},
		MetricType:  counterType(),
	})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "Band=42", *rows[0].ObjectLDN)
	assert.Equal(t, "L.Cell.A", rows[0].DisplayName, "counter 行 DisplayName=metric_path")
}

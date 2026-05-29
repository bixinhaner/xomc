package aggregator

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/pm/kpi/router"
	"github.com/omcgo/omcgo/internal/pm/metrics"
)

// ---------------------------------------------------------------------------
// buildCountersSQL：SQL 文本断言（不依赖 DB）
// ---------------------------------------------------------------------------

func Test_buildCountersSQL_HourlyFromPmMetrics(t *testing.T) {
	w := WindowSpec{
		Granularity: metrics.GranularityHourly,
		Start:       time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC),
		End:         time.Date(2026, 5, 22, 11, 0, 0, 0, time.UTC),
	}
	sql, args := buildCountersSQL("pm_metrics", "pm_metrics_hourly", w)

	// CASE WHEN 四路聚合（BUG-D 补 min）
	assert.Contains(t, sql, "WHEN 'sum' THEN SUM(m.metric_value)")
	assert.Contains(t, sql, "WHEN 'avg' THEN AVG(m.metric_value)")
	assert.Contains(t, sql, "WHEN 'max' THEN MAX(m.metric_value)")
	assert.Contains(t, sql, "WHEN 'min' THEN MIN(m.metric_value)")
	// pct/NULL counter 不进聚合
	assert.Contains(t, sql, "AND m.statis_type IN ('sum','avg','max','min')")
	// hourly 目标含 id 列（hypertable）
	assert.Contains(t, sql, "INSERT INTO pm_metrics_hourly (id, device_oui")
	assert.Contains(t, sql, "gen_random_uuid()")
	// hourly 冲突目标含 time 列
	assert.Contains(t, sql, "ON CONFLICT (device_oui, device_sn, metric_path, granularity, end_time, time)")
	// source 表名正确
	assert.Contains(t, sql, "FROM pm_metrics m")
	// args 顺序：granularity / bucket_start / bucket_end / where_start / where_end
	assert.Equal(t, []any{"hourly", w.Start, w.End, w.Start, w.End}, args)
}

func Test_buildCountersSQL_DailyFromHourly(t *testing.T) {
	w := WindowSpec{
		Granularity: metrics.GranularityDaily,
		Start:       time.Date(2026, 5, 22, 0, 0, 0, 0, time.UTC),
		End:         time.Date(2026, 5, 23, 0, 0, 0, 0, time.UTC),
	}
	sql, args := buildCountersSQL("pm_metrics_hourly", "pm_metrics_daily", w)

	// daily 目标无 id 列
	assert.Contains(t, sql, "INSERT INTO pm_metrics_daily (device_oui")
	assert.NotContains(t, sql, "gen_random_uuid()")
	// daily 冲突目标无 time 列（PK 是 5 列）
	assert.Contains(t, sql, "ON CONFLICT (device_oui, device_sn, metric_path, granularity, end_time)")
	// source = hourly
	assert.Contains(t, sql, "FROM pm_metrics_hourly m")
	// granularity 参数
	assert.Equal(t, "daily", args[0])
}

func Test_buildCountersSQL_GroupHourlyHasGroupIDConflict(t *testing.T) {
	w := WindowSpec{
		Granularity: metrics.GranularityHourly,
		Start:       time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC),
		End:         time.Date(2026, 5, 22, 11, 0, 0, 0, time.UTC),
	}
	// device_group 维度的 hourly 表（实际由 device_group.go 的 SQL 构造，此处仅断言 conflict target 切换正确）
	_ = w
	assert.Equal(t,
		"(device_group_id, metric_path, granularity, end_time, time)",
		conflictTargetForTable("pm_group_metrics_hourly"))
	assert.Equal(t,
		"(device_group_id, metric_path, granularity, end_time)",
		conflictTargetForTable("pm_group_metrics_daily"))
}

// ---------------------------------------------------------------------------
// AggregateCounters：通过 stub DB 验证 args + sql 拼接交给底层 Exec
// ---------------------------------------------------------------------------

type stubDB struct {
	execSQL  string
	execArgs []any
	execTag  pgconn.CommandTag
	execErr  error

	queryFn func(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

func (s *stubDB) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	s.execSQL = sql
	s.execArgs = args
	return s.execTag, s.execErr
}

func (s *stubDB) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	if s.queryFn != nil {
		return s.queryFn(ctx, sql, args...)
	}
	return nil, errors.New("stubDB.Query unimplemented")
}

func Test_AggregateCounters_PassesThroughToExec(t *testing.T) {
	db := &stubDB{execTag: pgconn.NewCommandTag("INSERT 0 3")}
	a := New(db, nil, nil)
	w := WindowSpec{
		Granularity: metrics.GranularityHourly,
		Start:       time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC),
		End:         time.Date(2026, 5, 22, 11, 0, 0, 0, time.UTC),
	}

	n, err := a.AggregateCounters(context.Background(), "pm_metrics", "pm_metrics_hourly", w)
	require.NoError(t, err)
	assert.Equal(t, 3, n)
	assert.Contains(t, db.execSQL, "INSERT INTO pm_metrics_hourly")
	assert.Equal(t, []any{"hourly", w.Start, w.End, w.Start, w.End}, db.execArgs)
}

// ---------------------------------------------------------------------------
// KPI 求值路径：用 stub router + stub Query/Exec 验证 evaluator 路径
// ---------------------------------------------------------------------------

type stubKPIRouter struct {
	byDevice map[string]*router.KPIRoute
}

func (s *stubKPIRouter) LookupByDevice(ctx context.Context, sn string) (*router.KPIRoute, error) {
	if r, ok := s.byDevice[sn]; ok {
		return r, nil
	}
	return nil, errors.New("not found")
}

// fakeRows 实现 pgx.Rows 的最小子集，模拟 SELECT 结果
type fakeRows struct {
	rows [][]any
	idx  int
	err  error
}

func (f *fakeRows) Next() bool                           { f.idx++; return f.idx <= len(f.rows) }
func (f *fakeRows) Scan(dest ...any) error               { return scanInto(f.rows[f.idx-1], dest) }
func (f *fakeRows) Close()                               {}
func (f *fakeRows) Err() error                           { return f.err }
func (f *fakeRows) CommandTag() pgconn.CommandTag                { return pgconn.NewCommandTag("") }
func (f *fakeRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (f *fakeRows) Values() ([]any, error)               { return nil, nil }
func (f *fakeRows) RawValues() [][]byte                  { return nil }
func (f *fakeRows) Conn() *pgx.Conn                      { return nil }

// scanInto copies row values into dest pointers, supporting string and float64.
func scanInto(row []any, dest []any) error {
	for i, d := range dest {
		switch dp := d.(type) {
		case *string:
			*dp = row[i].(string)
		case *float64:
			*dp = row[i].(float64)
		default:
			return errors.New("scanInto: unsupported dest type")
		}
	}
	return nil
}

func Test_AggregateKPIs_EvaluatesFormulaAndInserts(t *testing.T) {
	// device (oui=A, sn=S1) 关联一个 KPI：avail_rate = numerator / denominator
	rt := &router.KPIRoute{
		KPIs: []router.KPIDef{
			{
				IndicatorID:  "K1",
				Name:         "L.Cell.Avail.Rate",
				StatisType:   "pct",
				Formula:      "numerator / denominator",
				Dependencies: []string{"numerator", "denominator"},
			},
		},
	}
	kr := &stubKPIRouter{byDevice: map[string]*router.KPIRoute{"S1": rt}}

	queryCalls := 0
	db := &stubDB{
		execTag: pgconn.NewCommandTag("INSERT 0 1"),
		queryFn: func(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
			queryCalls++
			switch queryCalls {
			case 1: // listDevicesInBucket
				return &fakeRows{rows: [][]any{{"A", "S1"}}}, nil
			case 2: // loadCountersForDevice
				return &fakeRows{rows: [][]any{
					{"numerator", float64(80)},
					{"denominator", float64(100)},
				}}, nil
			}
			return nil, errors.New("unexpected Query")
		},
	}

	a := New(db, kr, nil)
	w := WindowSpec{
		Granularity: metrics.GranularityHourly,
		Start:       time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC),
		End:         time.Date(2026, 5, 22, 11, 0, 0, 0, time.UTC),
	}
	n, err := a.AggregateKPIs(context.Background(), "pm_metrics_hourly", w)
	require.NoError(t, err)
	assert.Equal(t, 1, n)
	// 求值结果 0.8 应出现在 Exec args 中（KPI 行 metric_value 位）
	require.Len(t, db.execArgs, 8) // oui, sn, path, val, stype, gran, bktStart, bktEnd
	// metric_path 落库用 K 编号(IndicatorID)，不再用显示名
	assert.Equal(t, "K1", db.execArgs[2])
	assert.Equal(t, float64(0.8), db.execArgs[3])
	assert.Equal(t, "pct", db.execArgs[4])
	assert.Equal(t, "hourly", db.execArgs[5])
}

// 同显示名、不同编号的两个 KPI（device 级 + plmn 级变体）在同一桶内，
// metric_path 必须各自落各自的 K 编号——否则一条多行 INSERT 内撞 ON CONFLICT 唯一键，
// PostgreSQL 报 21000，整设备整桶 KPI 全失败（聚合层 KPI 长期 0 行的根因）。
func Test_AggregateKPIs_SameDisplayNameUsesDistinctIndicatorID(t *testing.T) {
	rt := &router.KPIRoute{
		KPIs: []router.KPIDef{
			{
				IndicatorID:  "K900010029",
				Name:         "RRC连接建立成功率",
				StatisType:   "pct",
				Formula:      "numerator / denominator",
				Dependencies: []string{"numerator", "denominator"},
			},
			{
				// plmn 级变体：显示名完全相同，编号不同
				IndicatorID:  "K900010059",
				Name:         "RRC连接建立成功率",
				StatisType:   "pct",
				Formula:      "numerator / denominator",
				Dependencies: []string{"numerator", "denominator"},
			},
		},
	}
	kr := &stubKPIRouter{byDevice: map[string]*router.KPIRoute{"S1": rt}}

	queryCalls := 0
	db := &stubDB{
		execTag: pgconn.NewCommandTag("INSERT 0 2"),
		queryFn: func(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
			queryCalls++
			switch queryCalls {
			case 1:
				return &fakeRows{rows: [][]any{{"A", "S1"}}}, nil
			case 2:
				return &fakeRows{rows: [][]any{
					{"numerator", float64(95)},
					{"denominator", float64(100)},
				}}, nil
			}
			return nil, errors.New("unexpected Query")
		},
	}

	a := New(db, kr, nil)
	w := WindowSpec{
		Granularity: metrics.GranularityHourly,
		Start:       time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC),
		End:         time.Date(2026, 5, 22, 11, 0, 0, 0, time.UTC),
	}
	_, err := a.AggregateKPIs(context.Background(), "pm_metrics_hourly", w)
	require.NoError(t, err)
	// 两行各 8 个参数；两个 path 位（idx 2、idx 10）必须是不同编号，不能都是同一显示名
	require.Len(t, db.execArgs, 16)
	assert.Equal(t, "K900010029", db.execArgs[2])
	assert.Equal(t, "K900010059", db.execArgs[10])
	assert.NotEqual(t, db.execArgs[2], db.execArgs[10],
		"两个同显示名 KPI 的 metric_path 必须按编号区分，否则撞 ON CONFLICT 唯一键")
}

// backfillDisplayNames：KPI 行按编号回填指标库 cn_name，counter 行用 metric_path 本身，
// 查不到的编号回退用编号。
func Test_backfillDisplayNames(t *testing.T) {
	db := &stubDB{
		queryFn: func(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
			// 模拟三表 UNION 查询，只 K900010029 命中友好名，K_UNKNOWN 查不到
			return &fakeRows{rows: [][]any{
				{"K900010029", "RRC连接建立成功率"},
			}}, nil
		},
	}
	a := New(db, nil, nil)
	rows := []Row{
		{MetricPath: "K900010029", MetricType: metrics.MetricTypeKPI},
		{MetricPath: "K_UNKNOWN", MetricType: metrics.MetricTypeKPI},
		{MetricPath: "L.Cell.RrcConn", MetricType: metrics.MetricTypeCounter},
	}
	a.backfillDisplayNames(context.Background(), rows)
	assert.Equal(t, "RRC连接建立成功率", rows[0].DisplayName, "命中编号回填 cn_name")
	assert.Equal(t, "K_UNKNOWN", rows[1].DisplayName, "查不到的编号回退用编号本身")
	assert.Equal(t, "L.Cell.RrcConn", rows[2].DisplayName, "counter 行用 metric_path")
}

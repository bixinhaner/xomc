package aggregator

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
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
	// hourly 冲突目标含 time 列 + 小区/PLMN（T-A 物化分层）
	assert.Contains(t, sql, "ON CONFLICT (device_oui, device_sn, metric_path, granularity, end_time, time, object_ldn)")
	// 小区/PLMN 进 GROUP BY 尾部、且不再用 MIN 取代表值拍平
	assert.Contains(t, sql, "GROUP BY m.device_oui, m.device_sn, m.metric_path, m.statis_type, m.object_ldn")
	assert.NotContains(t, sql, "MIN(m.object_ldn)", "小区进分组后不应再 MIN 取代表值")
	assert.Contains(t, sql, "m.object_ldn,", "object_ldn 应原样带入 SELECT")
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
	// daily 冲突目标无 time 列，但 PK 尾部含 object_ldn（T-A 物化分层，6 列）
	assert.Contains(t, sql, "ON CONFLICT (device_oui, device_sn, metric_path, granularity, end_time, object_ldn)")
	// 小区/PLMN 进 GROUP BY、不再 MIN 拍平
	assert.Contains(t, sql, "GROUP BY m.device_oui, m.device_sn, m.metric_path, m.statis_type, m.object_ldn")
	assert.NotContains(t, sql, "MIN(m.object_ldn)")
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
	// device_group 四表本次不分小区（决策 #1）——冲突列绝不含 object_ldn
	for _, gt := range []string{
		"pm_group_metrics_hourly", "pm_group_metrics_daily",
		"pm_group_metrics_weekly", "pm_group_metrics_monthly",
	} {
		assert.NotContains(t, conflictTargetForTable(gt), "object_ldn",
			"group 表冲突列不应含 object_ldn："+gt)
	}
}

// 设备级四表的 ON CONFLICT 冲突列尾部都必须含 object_ldn（与迁移 000020 唯一键配套）。
func Test_conflictTargetForTable_DeviceTablesIncludeObjectLdn(t *testing.T) {
	assert.Equal(t,
		"(device_oui, device_sn, metric_path, granularity, end_time, time, object_ldn)",
		conflictTargetForTable("pm_metrics_hourly"))
	for _, dt := range []string{"pm_metrics_daily", "pm_metrics_weekly", "pm_metrics_monthly"} {
		assert.Equal(t,
			"(device_oui, device_sn, metric_path, granularity, end_time, object_ldn)",
			conflictTargetForTable(dt), dt)
	}
}

// buildKPIInsertSQL（T-B）：object_ldn 由硬写 '' 改为写实体实际 object_ldn（参数化 $N），
// 设备级实体（object_ldn==""）仍落空串；object_ldn 绝不写 NULL（NOT NULL 约束）。
// 覆盖 hourly（带 id 列）与 daily（不带 id）两种目标表。
func Test_buildKPIInsertSQL_WritesEntityObjectLdn(t *testing.T) {
	w := WindowSpec{
		Granularity: metrics.GranularityHourly,
		Start:       time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC),
		End:         time.Date(2026, 5, 22, 11, 0, 0, 0, time.UTC),
	}
	rows := []kpiRow{{path: "K1", value: 0.8, stype: "pct"}}

	// PLMN 实体：object_ldn 必须作为实参写入（不再硬写空串字面量）。
	plmnSQL, plmnArgs := buildKPIInsertSQL("pm_metrics_hourly",
		entityKey{oui: "A", sn: "S1", objectLdn: "Cellid=111172245,PLMN=46068"}, w, rows)
	assert.Contains(t, plmnArgs, "Cellid=111172245,PLMN=46068",
		"KPI 行 object_ldn 必须写实体实际小区/PLMN")
	assert.NotContains(t, plmnSQL, "NOW(), '', NULL::jsonb)",
		"object_ldn 已参数化，不应再是硬写空串字面量")
	assert.NotContains(t, plmnSQL, "NOW(), NULL, NULL::jsonb)", "object_ldn 绝不写 NULL")
	assert.Contains(t, plmnSQL, "gen_random_uuid()")
	assert.Contains(t, plmnSQL,
		"ON CONFLICT (device_oui, device_sn, metric_path, granularity, end_time, time, object_ldn)")

	// 设备级实体（object_ldn==""）：实参仍是空串，落库 object_ldn=''（保持 T-A 前行为）。
	dailyW := WindowSpec{
		Granularity: metrics.GranularityDaily,
		Start:       time.Date(2026, 5, 22, 0, 0, 0, 0, time.UTC),
		End:         time.Date(2026, 5, 23, 0, 0, 0, 0, time.UTC),
	}
	devSQL, devArgs := buildKPIInsertSQL("pm_metrics_daily",
		entityKey{oui: "A", sn: "S1", objectLdn: ""}, dailyW, rows)
	assert.Contains(t, devArgs, "", "设备级实体 object_ldn 实参为空串")
	assert.NotContains(t, devSQL, "NOW(), NULL, NULL::jsonb)")
	assert.NotContains(t, devSQL, "gen_random_uuid()", "daily 无 id 列")
	assert.Contains(t, devSQL,
		"ON CONFLICT (device_oui, device_sn, metric_path, granularity, end_time, object_ldn)")
	// daily 每行 9 个参数（含 object_ldn 实参）
	assert.Len(t, devArgs, 9)
}

// ---------------------------------------------------------------------------
// AggregateCounters：通过 stub DB 验证 args + sql 拼接交给底层 Exec
// ---------------------------------------------------------------------------

type stubDB struct {
	execSQL  string
	execArgs []any
	execTag  pgconn.CommandTag
	execErr  error

	queryFn    func(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	queryRowFn func(ctx context.Context, sql string, args ...any) pgx.Row
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

func (s *stubDB) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	if s.queryRowFn != nil {
		return s.queryRowFn(ctx, sql, args...)
	}
	return errRow{err: errors.New("stubDB.QueryRow unimplemented")}
}

// countRow / errRow 是 PgQuerier.QueryRow 的轻量 stub：把一个固定 int（或 error）喂给 Scan。
type countRow struct {
	n    int
	sql  string
	args []any
}

func (r countRow) Scan(dest ...any) error {
	if len(dest) == 1 {
		if p, ok := dest[0].(*int); ok {
			*p = r.n
		}
	}
	return nil
}

type errRow struct{ err error }

func (r errRow) Scan(dest ...any) error { return r.err }

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

// scanInto copies row values into dest pointers, supporting string, float64,
// time.Time, and nullable *string (dest **string).
func scanInto(row []any, dest []any) error {
	for i, d := range dest {
		switch dp := d.(type) {
		case *string:
			*dp = row[i].(string)
		case *float64:
			*dp = row[i].(float64)
		case *time.Time:
			*dp = row[i].(time.Time)
		case **string:
			if row[i] == nil {
				*dp = nil
			} else {
				v := row[i].(string)
				*dp = &v
			}
		case *uuid.UUID:
			switch v := row[i].(type) {
			case uuid.UUID:
				*dp = v
			case string:
				*dp = uuid.MustParse(v)
			case nil:
				*dp = uuid.Nil
			default:
				return errors.New("scanInto: unsupported uuid source")
			}
		case *[]byte:
			if row[i] == nil {
				*dp = nil
			} else if b, ok := row[i].([]byte); ok {
				*dp = b
			} else {
				return errors.New("scanInto: unsupported []byte source")
			}
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
			case 1: // listEntitiesInBucket: (oui, sn, object_ldn) —— 设备级实体（空串）
				return &fakeRows{rows: [][]any{{"A", "S1", ""}}}, nil
			case 2: // loadCountersByObjectLdn: (object_ldn, metric_path, value)
				return &fakeRows{rows: [][]any{
					{"", "numerator", float64(80)},
					{"", "denominator", float64(100)},
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
	require.Len(t, db.execArgs, 9) // oui, sn, path, val, stype, gran, bktStart, bktEnd, object_ldn
	// metric_path 落库用 K 编号(IndicatorID)，不再用显示名
	assert.Equal(t, "K1", db.execArgs[2])
	assert.Equal(t, float64(0.8), db.execArgs[3])
	assert.Equal(t, "pct", db.execArgs[4])
	assert.Equal(t, "hourly", db.execArgs[5])
	// 设备级实体 object_ldn 落空串
	assert.Equal(t, "", db.execArgs[8])
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
			case 1: // listEntitiesInBucket: (oui, sn, object_ldn)
				return &fakeRows{rows: [][]any{{"A", "S1", ""}}}, nil
			case 2: // loadCountersByObjectLdn: (object_ldn, metric_path, value)
				return &fakeRows{rows: [][]any{
					{"", "numerator", float64(95)},
					{"", "denominator", float64(100)},
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
	// 两行各 9 个参数；两个 path 位（idx 2、idx 11）必须是不同编号，不能都是同一显示名
	require.Len(t, db.execArgs, 18)
	assert.Equal(t, "K900010029", db.execArgs[2])
	assert.Equal(t, "K900010059", db.execArgs[11])
	assert.NotEqual(t, db.execArgs[2], db.execArgs[11],
		"两个同显示名 KPI 的 metric_path 必须按编号区分，否则撞 ON CONFLICT 唯一键")
}

// ---------------------------------------------------------------------------
// 路子 A：两个读本桶 counter 的 helper SQL 谓词断言（精确命中本桶，不再半开区间）
// ---------------------------------------------------------------------------

// listEntitiesInBucket / loadCountersByObjectLdn 读的是 target 表（每行即完整桶，
// end_time=w.End）。修复前用源表式半开区间 end_time>=w.Start AND end_time<w.End，
// 会把本桶行（end_time=w.End）排除、反而命中上一桶（end_time=本桶 w.Start）。
// 修复后必须精确命中本桶：end_time = w.End（叠加 time = w.Start 自证），
// 且 args 携带 w.End（不再是半开的 w.Start/w.End 对）。

func Test_buildListEntitiesInBucketSQL_PreciseBucketMatch_Hourly(t *testing.T) {
	w := WindowSpec{
		Granularity: metrics.GranularityHourly,
		Start:       time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC),
		End:         time.Date(2026, 5, 22, 11, 0, 0, 0, time.UTC),
	}
	sql, args := buildListEntitiesInBucketSQL("pm_metrics_hourly", w)

	// T-B：实体枚举含 object_ldn——每个小区/PLMN 各成一独立实体
	assert.Contains(t, sql, "SELECT DISTINCT device_oui, device_sn, object_ldn")
	// 精确命中本桶：end_time = $N（不再是半开 >= ... AND < ...）
	assert.Contains(t, sql, "end_time = $2")
	assert.Contains(t, sql, "time = $3")
	assert.NotContains(t, sql, "end_time >=", "不应再有半开下界")
	assert.NotContains(t, sql, "end_time <", "不应再有半开上界")
	assert.Contains(t, sql, "FROM pm_metrics_hourly")
	// args：granularity / w.End / w.Start —— 桶尾 w.End 进了谓词，不再是半开 w.Start/w.End 对
	assert.Equal(t, []any{"hourly", w.End, w.Start}, args)
}

func Test_buildListEntitiesInBucketSQL_PreciseBucketMatch_AllGranularities(t *testing.T) {
	cases := []struct {
		name        string
		granularity metrics.Granularity
		target      string
		start, end  time.Time
	}{
		{
			name:        "daily",
			granularity: metrics.GranularityDaily,
			target:      "pm_metrics_daily",
			start:       time.Date(2026, 5, 22, 0, 0, 0, 0, time.UTC),
			end:         time.Date(2026, 5, 23, 0, 0, 0, 0, time.UTC),
		},
		{
			name:        "weekly",
			granularity: metrics.GranularityWeekly,
			target:      "pm_metrics_weekly",
			start:       time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC),
			end:         time.Date(2026, 5, 25, 0, 0, 0, 0, time.UTC),
		},
		{
			name:        "monthly",
			granularity: metrics.GranularityMonthly,
			target:      "pm_metrics_monthly",
			start:       time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
			end:         time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			w := WindowSpec{Granularity: c.granularity, Start: c.start, End: c.end}
			sql, args := buildListEntitiesInBucketSQL(c.target, w)
			assert.Contains(t, sql, "SELECT DISTINCT device_oui, device_sn, object_ldn")
			assert.Contains(t, sql, "end_time = $2")
			assert.NotContains(t, sql, "end_time >=")
			assert.NotContains(t, sql, "end_time <")
			assert.Equal(t, []any{string(c.granularity), c.end, c.start}, args)
		})
	}
}

func Test_buildLoadCountersByObjectLdnSQL_PreciseBucketMatch_Hourly(t *testing.T) {
	w := WindowSpec{
		Granularity: metrics.GranularityHourly,
		Start:       time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC),
		End:         time.Date(2026, 5, 22, 11, 0, 0, 0, time.UTC),
	}
	sql, args := buildLoadCountersByObjectLdnSQL("pm_metrics_hourly", "A", "S1", w)

	// T-B：按 (object_ldn, metric_path) 精确带出每行计数器，不再折回设备级（无 GROUP BY metric_path）
	assert.Contains(t, sql, "SELECT object_ldn, metric_path, metric_value")
	assert.NotContains(t, sql, "GROUP BY metric_path", "T-B 拆掉折回层，不再按 metric_path 聚合")
	assert.NotContains(t, sql, "CASE MIN(statis_type)", "T-B 不再折回设备级")
	assert.Contains(t, sql, "end_time = $4")
	assert.Contains(t, sql, "time = $5")
	assert.NotContains(t, sql, "end_time >=", "不应再有半开下界")
	assert.NotContains(t, sql, "end_time <", "不应再有半开上界")
	assert.Contains(t, sql, "FROM pm_metrics_hourly")
	// args：oui / sn / granularity / w.End / w.Start
	assert.Equal(t, []any{"A", "S1", "hourly", w.End, w.Start}, args)
}

func Test_buildLoadCountersByObjectLdnSQL_PreciseBucketMatch_Daily(t *testing.T) {
	w := WindowSpec{
		Granularity: metrics.GranularityDaily,
		Start:       time.Date(2026, 5, 22, 0, 0, 0, 0, time.UTC),
		End:         time.Date(2026, 5, 23, 0, 0, 0, 0, time.UTC),
	}
	sql, args := buildLoadCountersByObjectLdnSQL("pm_metrics_daily", "A", "S1", w)

	assert.Contains(t, sql, "SELECT object_ldn, metric_path, metric_value")
	assert.Contains(t, sql, "end_time = $4")
	assert.NotContains(t, sql, "end_time >=")
	assert.NotContains(t, sql, "end_time <")
	assert.Contains(t, sql, "FROM pm_metrics_daily")
	assert.Equal(t, []any{"A", "S1", "daily", w.End, w.Start}, args)
}

// ---------------------------------------------------------------------------
// T-B：countersForEntity 跨层级配对——PLMN 实体配同小区基础行的小区级计数器
// ---------------------------------------------------------------------------

// 基础小区实体只取本行小区级计数器（无需配 PLMN）。
func Test_countersForEntity_BaseCell_OwnCounters(t *testing.T) {
	byLdn := map[string]map[string]float64{
		"Cellid=1": {"cellNum": 30, "cellDen": 60},
		"Cellid=2": {"cellNum": 50, "cellDen": 40},
	}
	got := countersForEntity(byLdn, "Cellid=1")
	assert.Equal(t, map[string]float64{"cellNum": 30, "cellDen": 60}, got,
		"基础小区实体只取本小区计数器，不混入别的小区")
}

// 验收 #2：两个不同小区互不串味——各取本小区计数。
func Test_countersForEntity_TwoCells_NoCrossContamination(t *testing.T) {
	byLdn := map[string]map[string]float64{
		"Cellid=1": {"num": 30, "den": 60},
		"Cellid=2": {"num": 50, "den": 40},
	}
	c1 := countersForEntity(byLdn, "Cellid=1")
	c2 := countersForEntity(byLdn, "Cellid=2")
	assert.Equal(t, float64(30), c1["num"])
	assert.Equal(t, float64(60), c1["den"])
	assert.Equal(t, float64(50), c2["num"])
	assert.Equal(t, float64(40), c2["den"])
}

// 验收 #3：PLMN 实体跨层级配对——本 PLMN 行的 PLMN 级计数 ∪ 同小区基础行的小区级计数。
func Test_countersForEntity_PLMN_CrossLayerPairing(t *testing.T) {
	byLdn := map[string]map[string]float64{
		// 基础小区行：只上报小区级计数器
		"Cellid=111172245":            {"cellLevelCnt": 1000},
		// 两个 PLMN 行：各只上报 PLMN 级计数器
		"Cellid=111172245,PLMN=00101": {"plmnLevelCnt": 200},
		"Cellid=111172245,PLMN=46068": {"plmnLevelCnt": 350},
	}
	// PLMN=00101 实体：拿到本 PLMN 级 + 基础小区的小区级
	p1 := countersForEntity(byLdn, "Cellid=111172245,PLMN=00101")
	assert.Equal(t, float64(200), p1["plmnLevelCnt"], "本 PLMN 级计数")
	assert.Equal(t, float64(1000), p1["cellLevelCnt"], "跨层级配上基础小区的小区级计数")

	// PLMN=46068 实体：配同一基础小区，各自 PLMN 级值不串
	p2 := countersForEntity(byLdn, "Cellid=111172245,PLMN=46068")
	assert.Equal(t, float64(350), p2["plmnLevelCnt"])
	assert.Equal(t, float64(1000), p2["cellLevelCnt"])

	// 基础小区实体：只有小区级，不带 PLMN 级
	base := countersForEntity(byLdn, "Cellid=111172245")
	assert.Equal(t, float64(1000), base["cellLevelCnt"])
	_, hasPlmn := base["plmnLevelCnt"]
	assert.False(t, hasPlmn, "基础小区实体不应混入 PLMN 级计数器")
}

// 自身行的值优先于配上来的小区级值（防御：两类 metric_path 互斥，撞键时实体自身胜出）。
func Test_countersForEntity_OwnValueWinsOnKeyCollision(t *testing.T) {
	byLdn := map[string]map[string]float64{
		"Cellid=5":          {"shared": 100},
		"Cellid=5,PLMN=001": {"shared": 7},
	}
	p := countersForEntity(byLdn, "Cellid=5,PLMN=001")
	assert.Equal(t, float64(7), p["shared"], "实体自身行的值优先")
}

// 设备级实体（object_ldn==""）只取空串行计数器，不配对。
func Test_countersForEntity_DeviceLevel_EmptyLdn(t *testing.T) {
	byLdn := map[string]map[string]float64{
		"":         {"a": 1, "b": 2},
		"Cellid=1": {"c": 9},
	}
	got := countersForEntity(byLdn, "")
	assert.Equal(t, map[string]float64{"a": 1, "b": 2}, got)
}

// NR/空小区实体（parseObjectLDN 提不出 cellID）不 panic，按本行取数降级。
func Test_countersForEntity_NonParsableLdn_NoPanic(t *testing.T) {
	byLdn := map[string]map[string]float64{
		"NRCellDU=Cell0": {"x": 5},
	}
	assert.NotPanics(t, func() {
		got := countersForEntity(byLdn, "NRCellDU=Cell0")
		assert.Equal(t, float64(5), got["x"])
	})
}

// buildLoadCountersByObjectLdnSQL 的 SQL 形态在上方精确桶匹配测试已覆盖（不再折回设备级）。

// ---------------------------------------------------------------------------
// T-B 行为单测：用 object_ldn-aware 智能桩证明 KPI 按小区/PLMN 各算一条、不串味、跨层级配对
// ---------------------------------------------------------------------------

// cellRow 带 object_ldn，供智能桩按实体（设备 + 小区/PLMN）枚举与取数。
type cellRow struct {
	oui, sn   string
	path      string
	value     float64
	objectLdn string
	endTime   time.Time
	timeCol   time.Time
}

// cellAwareDB 复现 T-B 后 target 表：每 (object_ldn, metric_path) 一行。
//   - listEntitiesInBucket（3 args）：DISTINCT (oui, sn, object_ldn) 实体
//   - loadCountersByObjectLdn（5 args）：按本桶返回 (object_ldn, metric_path, value)，不折回
// 每次 Exec 累积 args，供断言多实体多次插入。
type cellAwareDB struct {
	rows     []cellRow
	execTag  pgconn.CommandTag
	execArgs [][]any
}

func (db *cellAwareDB) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	db.execArgs = append(db.execArgs, args)
	return db.execTag, nil
}

func (db *cellAwareDB) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return errRow{err: errors.New("cellAwareDB.QueryRow unimplemented")}
}

func (db *cellAwareDB) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	// listEntitiesInBucket: args = [granularity, end_time, time]
	if len(args) == 3 {
		want := args[1].(time.Time)
		seen := map[entityKey]bool{}
		var out [][]any
		for _, r := range db.rows {
			if r.endTime.Equal(want) {
				k := entityKey{r.oui, r.sn, r.objectLdn}
				if !seen[k] {
					seen[k] = true
					out = append(out, []any{r.oui, r.sn, r.objectLdn})
				}
			}
		}
		return &fakeRows{rows: out}, nil
	}
	// loadCountersByObjectLdn: args = [oui, sn, granularity, end_time, time]
	// 按本桶/本设备返回每 (object_ldn, metric_path) 一行（T-B 不折回设备级）。
	if len(args) == 5 {
		oui := args[0].(string)
		sn := args[1].(string)
		want := args[3].(time.Time)
		var out [][]any
		for _, r := range db.rows {
			if r.oui == oui && r.sn == sn && r.endTime.Equal(want) {
				out = append(out, []any{r.objectLdn, r.path, r.value})
			}
		}
		return &fakeRows{rows: out}, nil
	}
	return nil, errors.New("cellAwareDB.Query unexpected args len")
}

// findKPIArgs 从累积的 Exec args 里找出 object_ldn 实参等于 wantLdn 的那次插入（每次 9 参/行）。
func findKPIArgs(execArgs [][]any, wantLdn string) []any {
	for _, args := range execArgs {
		if len(args) == 9 && args[8] == wantLdn {
			return args
		}
	}
	return nil
}

// findKPIArgsByPath 在累积的 Exec args 里定位「object_ldn 实参 == wantLdn 且 metric_path == wantPath」
// 的那一行（每行 9 参：oui,sn,path,val,stype,gran,bktStart,bktEnd,object_ldn）。
// 当某实体一次 Exec 插入多行 KPI 时（一次 Exec 累 9×N 参），逐行 9 参切片匹配。
func findKPIArgsByPath(execArgs [][]any, wantLdn, wantPath string) []any {
	for _, args := range execArgs {
		for off := 0; off+9 <= len(args); off += 9 {
			row := args[off : off+9]
			if row[8] == wantLdn && row[2] == wantPath {
				return row
			}
		}
	}
	return nil
}

// 验收 #1/#2：同设备两个不同小区各有不同 numerator/denominator →
// KPI 按每个 object_ldn 各枚举一条，各用本小区计数算，互不串味（不再折回设备级 0.8）。
func Test_AggregateKPIs_PerCell_NoCrossContamination(t *testing.T) {
	curStart := time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC)
	curEnd := time.Date(2026, 5, 22, 11, 0, 0, 0, time.UTC)

	db := &cellAwareDB{
		execTag: pgconn.NewCommandTag("INSERT 0 1"),
		rows: []cellRow{
			// 小区1：30/60 = 0.5
			{oui: "A", sn: "S1", path: "numerator", value: 30, objectLdn: "Cellid=1", endTime: curEnd, timeCol: curStart},
			{oui: "A", sn: "S1", path: "denominator", value: 60, objectLdn: "Cellid=1", endTime: curEnd, timeCol: curStart},
			// 小区2：50/40 = 1.25
			{oui: "A", sn: "S1", path: "numerator", value: 50, objectLdn: "Cellid=2", endTime: curEnd, timeCol: curStart},
			{oui: "A", sn: "S1", path: "denominator", value: 40, objectLdn: "Cellid=2", endTime: curEnd, timeCol: curStart},
		},
	}
	kr := &stubKPIRouter{byDevice: map[string]*router.KPIRoute{"S1": avilRateRoute()}}
	a := New(db, kr, nil)
	w := WindowSpec{Granularity: metrics.GranularityHourly, Start: curStart, End: curEnd}

	n, err := a.AggregateKPIs(context.Background(), "pm_metrics_hourly", w)
	require.NoError(t, err)
	assert.Equal(t, 2, n, "两个小区各产一条 KPI 行")

	c1 := findKPIArgs(db.execArgs, "Cellid=1")
	require.NotNil(t, c1, "应有 Cellid=1 实体的 KPI 行")
	assert.Equal(t, float64(0.5), c1[3], "小区1 KPI=30/60，不混入小区2")

	c2 := findKPIArgs(db.execArgs, "Cellid=2")
	require.NotNil(t, c2, "应有 Cellid=2 实体的 KPI 行")
	assert.Equal(t, float64(1.25), c2[3], "小区2 KPI=50/40，不混入小区1")
}

// 验收 #1/#3：基础小区 + 两个 PLMN 三实体；混合公式（小区级 numerator + PLMN 级 denominator）→
// PLMN 实体跨层级配对算出正确结果；基础小区实体缺 PLMN 级入参按规则跳过、不报错。
func Test_AggregateKPIs_PLMN_CrossLayerPairing(t *testing.T) {
	curStart := time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC)
	curEnd := time.Date(2026, 5, 22, 11, 0, 0, 0, time.UTC)

	// 混合公式 KPI：numerator 只在基础小区上报（小区级），denominator 只在 PLMN 行上报（PLMN 级）。
	mixedRoute := &router.KPIRoute{
		KPIs: []router.KPIDef{{
			IndicatorID:  "Kmix",
			Name:         "Mixed",
			StatisType:   "pct",
			Formula:      "numerator / denominator",
			Dependencies: []string{"numerator", "denominator"},
		}},
	}
	db := &cellAwareDB{
		execTag: pgconn.NewCommandTag("INSERT 0 1"),
		rows: []cellRow{
			// 基础小区行：只有小区级 numerator=1000
			{oui: "A", sn: "S1", path: "numerator", value: 1000, objectLdn: "Cellid=9", endTime: curEnd, timeCol: curStart},
			// PLMN 行：各只有 PLMN 级 denominator
			{oui: "A", sn: "S1", path: "denominator", value: 200, objectLdn: "Cellid=9,PLMN=00101", endTime: curEnd, timeCol: curStart},
			{oui: "A", sn: "S1", path: "denominator", value: 500, objectLdn: "Cellid=9,PLMN=46068", endTime: curEnd, timeCol: curStart},
		},
	}
	kr := &stubKPIRouter{byDevice: map[string]*router.KPIRoute{"S1": mixedRoute}}
	a := New(db, kr, nil)
	w := WindowSpec{Granularity: metrics.GranularityHourly, Start: curStart, End: curEnd}

	n, err := a.AggregateKPIs(context.Background(), "pm_metrics_hourly", w)
	require.NoError(t, err)
	// 基础小区实体缺 denominator（PLMN 级）→ Evaluate 缺依赖跳过；两个 PLMN 实体各配上小区级 numerator 算出
	assert.Equal(t, 2, n, "仅两个 PLMN 实体算出 KPI（基础小区缺 PLMN 级入参被跳过）")

	p1 := findKPIArgs(db.execArgs, "Cellid=9,PLMN=00101")
	require.NotNil(t, p1)
	assert.Equal(t, float64(5), p1[3], "PLMN=00101：基础小区级 1000 / 本 PLMN 级 200 = 5")

	p2 := findKPIArgs(db.execArgs, "Cellid=9,PLMN=46068")
	require.NotNil(t, p2)
	assert.Equal(t, float64(2), p2[3], "PLMN=46068：1000 / 500 = 2，不混入另一 PLMN 的 200")

	// 基础小区实体不应产 Kmix 行（缺 PLMN 级 denominator）
	assert.Nil(t, findKPIArgs(db.execArgs, "Cellid=9"), "基础小区缺 PLMN 级入参，跳过不报错")
}

// 运行栈缺陷修复（自身层级门槛）：纯小区级 KPI（公式只引用基础小区上报的小区级计数器）
// 只能在基础小区实体落库，绝不因跨层级配对把基础小区计数合并进 PLMN map 后被无差别求值
// 而"泄漏"到 PLMN 行（同编号、同值）。
//
// 构造：
//   - 基础小区 Cellid=7 上报小区级 cellNum=40 / cellDen=80（纯小区级 KPI 的全部依赖）
//   - 两个 PLMN 行 PLMN=00101 / PLMN=46068 上报 PLMN 级 plmnNum=10 / plmnDen=20
//   - 纯小区级 KPI（依赖 cellNum/cellDen）+ 纯 PLMN 级 KPI（依赖 plmnNum/plmnDen）
//
// 断言：
//   - 纯小区级 KPI 只在 Cellid=7 落库；两个 PLMN 实体均不得出现纯小区级 KPI（修掉泄漏）
//   - 纯 PLMN 级 KPI 仍在两个 PLMN 实体落库（门槛不误杀）
func Test_AggregateKPIs_PureCellKPI_DoesNotLeakToPLMN(t *testing.T) {
	curStart := time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC)
	curEnd := time.Date(2026, 5, 22, 11, 0, 0, 0, time.UTC)

	route := &router.KPIRoute{
		KPIs: []router.KPIDef{
			{
				// 纯小区级 KPI：依赖全在基础小区上报的小区级计数器
				IndicatorID:  "KcellOnly",
				Name:         "PureCell",
				StatisType:   "pct",
				Formula:      "cellNum / cellDen",
				Dependencies: []string{"cellNum", "cellDen"},
			},
			{
				// 纯 PLMN 级 KPI：依赖全在 PLMN 行上报的 PLMN 级计数器
				IndicatorID:  "KplmnOnly",
				Name:         "PurePLMN",
				StatisType:   "pct",
				Formula:      "plmnNum / plmnDen",
				Dependencies: []string{"plmnNum", "plmnDen"},
			},
		},
	}
	db := &cellAwareDB{
		execTag: pgconn.NewCommandTag("INSERT 0 1"),
		rows: []cellRow{
			// 基础小区行：小区级计数器
			{oui: "A", sn: "S1", path: "cellNum", value: 40, objectLdn: "Cellid=7", endTime: curEnd, timeCol: curStart},
			{oui: "A", sn: "S1", path: "cellDen", value: 80, objectLdn: "Cellid=7", endTime: curEnd, timeCol: curStart},
			// PLMN 行：PLMN 级计数器
			{oui: "A", sn: "S1", path: "plmnNum", value: 10, objectLdn: "Cellid=7,PLMN=00101", endTime: curEnd, timeCol: curStart},
			{oui: "A", sn: "S1", path: "plmnDen", value: 20, objectLdn: "Cellid=7,PLMN=00101", endTime: curEnd, timeCol: curStart},
			{oui: "A", sn: "S1", path: "plmnNum", value: 10, objectLdn: "Cellid=7,PLMN=46068", endTime: curEnd, timeCol: curStart},
			{oui: "A", sn: "S1", path: "plmnDen", value: 20, objectLdn: "Cellid=7,PLMN=46068", endTime: curEnd, timeCol: curStart},
		},
	}
	kr := &stubKPIRouter{byDevice: map[string]*router.KPIRoute{"S1": route}}
	a := New(db, kr, nil)
	w := WindowSpec{Granularity: metrics.GranularityHourly, Start: curStart, End: curEnd}

	n, err := a.AggregateKPIs(context.Background(), "pm_metrics_hourly", w)
	require.NoError(t, err)
	// 基础小区 1 行（纯小区级）+ 两个 PLMN 各 1 行（纯 PLMN 级）= 3 行，无泄漏的多出行
	assert.Equal(t, 3, n, "纯小区级 KPI 只产 1 行（基础小区），不在 PLMN 行重复")

	// 纯小区级 KPI 在基础小区落库
	base := findKPIArgsByPath(db.execArgs, "Cellid=7", "KcellOnly")
	require.NotNil(t, base, "纯小区级 KPI 应在基础小区实体落库")
	assert.Equal(t, float64(0.5), base[3], "40/80=0.5")

	// 纯小区级 KPI 绝不出现在任何 PLMN 实体（修掉泄漏）
	assert.Nil(t, findKPIArgsByPath(db.execArgs, "Cellid=7,PLMN=00101", "KcellOnly"),
		"纯小区级 KPI 不得泄漏到 PLMN=00101")
	assert.Nil(t, findKPIArgsByPath(db.execArgs, "Cellid=7,PLMN=46068", "KcellOnly"),
		"纯小区级 KPI 不得泄漏到 PLMN=46068")

	// 纯 PLMN 级 KPI 仍在两个 PLMN 实体落库（门槛不误杀）
	p1 := findKPIArgsByPath(db.execArgs, "Cellid=7,PLMN=00101", "KplmnOnly")
	require.NotNil(t, p1, "纯 PLMN 级 KPI 应在 PLMN=00101 落库")
	assert.Equal(t, float64(0.5), p1[3], "10/20=0.5")
	p2 := findKPIArgsByPath(db.execArgs, "Cellid=7,PLMN=46068", "KplmnOnly")
	require.NotNil(t, p2, "纯 PLMN 级 KPI 应在 PLMN=46068 落库")

	// 纯 PLMN 级 KPI 不应出现在基础小区（基础小区无 PLMN 级自身计数 → 门槛不过）
	assert.Nil(t, findKPIArgsByPath(db.execArgs, "Cellid=7", "KplmnOnly"),
		"纯 PLMN 级 KPI 不应在基础小区实体落库")
}

// kpiDependsOnOwnCounters 单测：门槛判据——公式依赖与实体自身行计数器有交集才落库。
func Test_kpiDependsOnOwnCounters(t *testing.T) {
	ownCell := map[string]float64{"cellNum": 1, "cellDen": 1}
	ownPlmn := map[string]float64{"plmnNum": 1, "plmnDen": 1}

	pureCell := router.KPIDef{Dependencies: []string{"cellNum", "cellDen"}}
	purePlmn := router.KPIDef{Dependencies: []string{"plmnNum", "plmnDen"}}
	mixed := router.KPIDef{Dependencies: []string{"cellNum", "plmnDen"}}

	// 纯小区级 KPI：在基础小区自身集里依赖齐全 → 门槛过；在 PLMN 自身集里无交集 → 门槛不过
	assert.True(t, kpiDependsOnOwnCounters(pureCell, ownCell), "纯小区级 KPI 在基础小区门槛过")
	assert.False(t, kpiDependsOnOwnCounters(pureCell, ownPlmn), "纯小区级 KPI 在 PLMN 门槛不过（修掉泄漏）")

	// 纯 PLMN 级 KPI：在 PLMN 自身集里门槛过；在基础小区自身集里门槛不过
	assert.True(t, kpiDependsOnOwnCounters(purePlmn, ownPlmn))
	assert.False(t, kpiDependsOnOwnCounters(purePlmn, ownCell))

	// 混合公式：只要引用了 PLMN 自身至少一个计数 → 在 PLMN 实体门槛过（配对补小区级入参）
	assert.True(t, kpiDependsOnOwnCounters(mixed, ownPlmn), "混合公式引用 PLMN 自身计数 → 门槛过")
	// 混合公式在基础小区：引用了小区级 cellNum → 门槛过（但缺 PLMN 级入参会在 Evaluate 阶段跳过）
	assert.True(t, kpiDependsOnOwnCounters(mixed, ownCell))

	// 无 Dependencies（无法判定层级归属）→ 保守落库（不门槛拦），保持原行为
	assert.True(t, kpiDependsOnOwnCounters(router.KPIDef{}, ownCell),
		"无依赖清单时不被门槛拦截（保守）")
	// 空自身集（理论上不会传入有 KPI 的实体）→ 门槛不过
	assert.False(t, kpiDependsOnOwnCounters(pureCell, map[string]float64{}))
}

// ---------------------------------------------------------------------------
// 路子 B：行为单测——智能桩按 end_time 精确过滤，证明 KPI 取本桶值、孤立桶也能算
// ---------------------------------------------------------------------------

// bucketRow 是智能桩里 target 表的一行（带 end_time，供按桶精确过滤）。
type bucketRow struct {
	oui, sn  string
	path     string
	value    float64
	endTime  time.Time
	timeCol  time.Time
}

// preciseBucketDB 模拟"按 end_time 精确过滤"的 target 表：
// listDevicesInBucket / loadCountersForDevice 传入 w.End（args 第二/第四位），
// 只返回 end_time == 该值的行——复现真实 PG 谓词行为，从而验证修复后只读到本桶。
type preciseBucketDB struct {
	rows    []bucketRow
	execTag pgconn.CommandTag
	execSQL string
	execArg []any
}

func (db *preciseBucketDB) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	db.execSQL = sql
	db.execArg = args
	return db.execTag, nil
}

func (db *preciseBucketDB) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return errRow{err: errors.New("preciseBucketDB.QueryRow unimplemented")}
}

func (db *preciseBucketDB) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	// listEntitiesInBucket: args = [granularity, end_time, time] → (oui, sn, object_ldn)
	// loadCountersByObjectLdn: args = [oui, sn, granularity, end_time, time] → (object_ldn, metric_path, value)
	if len(args) == 3 {
		want := args[1].(time.Time)
		seen := map[entityKey]bool{}
		var out [][]any
		for _, r := range db.rows {
			if r.endTime.Equal(want) {
				k := entityKey{r.oui, r.sn, ""} // 设备级实体（object_ldn 空串）
				if !seen[k] {
					seen[k] = true
					out = append(out, []any{r.oui, r.sn, ""})
				}
			}
		}
		return &fakeRows{rows: out}, nil
	}
	if len(args) == 5 {
		oui := args[0].(string)
		sn := args[1].(string)
		want := args[3].(time.Time)
		var out [][]any
		for _, r := range db.rows {
			if r.oui == oui && r.sn == sn && r.endTime.Equal(want) {
				out = append(out, []any{"", r.path, r.value})
			}
		}
		return &fakeRows{rows: out}, nil
	}
	return nil, errors.New("preciseBucketDB.Query unexpected args len")
}

func avilRateRoute() *router.KPIRoute {
	return &router.KPIRoute{
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
}

// 本桶有计数 + 上一桶有不同计数 → KPI 必须取本桶值（修复前会错取上一桶）。
func Test_AggregateKPIs_TakesCurrentBucket_NotPrevious(t *testing.T) {
	prevEnd := time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC) // 上一桶桶尾 = 本桶桶起
	curStart := time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC)
	curEnd := time.Date(2026, 5, 22, 11, 0, 0, 0, time.UTC)

	db := &preciseBucketDB{
		execTag: pgconn.NewCommandTag("INSERT 0 1"),
		rows: []bucketRow{
			// 上一桶 [09:00,10:00)：numerator=10, denominator=100 → 若被错读 KPI=0.1
			{oui: "A", sn: "S1", path: "numerator", value: 10, endTime: prevEnd, timeCol: prevEnd.Add(-time.Hour)},
			{oui: "A", sn: "S1", path: "denominator", value: 100, endTime: prevEnd, timeCol: prevEnd.Add(-time.Hour)},
			// 本桶 [10:00,11:00)：numerator=80, denominator=100 → 正确 KPI=0.8
			{oui: "A", sn: "S1", path: "numerator", value: 80, endTime: curEnd, timeCol: curStart},
			{oui: "A", sn: "S1", path: "denominator", value: 100, endTime: curEnd, timeCol: curStart},
		},
	}
	kr := &stubKPIRouter{byDevice: map[string]*router.KPIRoute{"S1": avilRateRoute()}}
	a := New(db, kr, nil)
	w := WindowSpec{Granularity: metrics.GranularityHourly, Start: curStart, End: curEnd}

	n, err := a.AggregateKPIs(context.Background(), "pm_metrics_hourly", w)
	require.NoError(t, err)
	assert.Equal(t, 1, n)
	require.Len(t, db.execArg, 9)
	assert.Equal(t, float64(0.8), db.execArg[3], "KPI 必须取本桶值 0.8，而非上一桶 0.1")
}

// 孤立桶：上一桶为空、本桶有计数 → 本桶 KPI 仍能算出（修复前因谓词排除本桶行而算不出）。
func Test_AggregateKPIs_IsolatedBucket_StillComputes(t *testing.T) {
	curStart := time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC)
	curEnd := time.Date(2026, 5, 22, 11, 0, 0, 0, time.UTC)

	db := &preciseBucketDB{
		execTag: pgconn.NewCommandTag("INSERT 0 1"),
		rows: []bucketRow{
			// 仅本桶有数据，上一桶完全为空（孤立桶）
			{oui: "A", sn: "S1", path: "numerator", value: 75, endTime: curEnd, timeCol: curStart},
			{oui: "A", sn: "S1", path: "denominator", value: 100, endTime: curEnd, timeCol: curStart},
		},
	}
	kr := &stubKPIRouter{byDevice: map[string]*router.KPIRoute{"S1": avilRateRoute()}}
	a := New(db, kr, nil)
	w := WindowSpec{Granularity: metrics.GranularityHourly, Start: curStart, End: curEnd}

	n, err := a.AggregateKPIs(context.Background(), "pm_metrics_hourly", w)
	require.NoError(t, err)
	assert.Equal(t, 1, n, "孤立桶本桶 KPI 仍应算出 1 行")
	require.Len(t, db.execArg, 9)
	assert.Equal(t, float64(0.75), db.execArg[3])
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

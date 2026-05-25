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

	// CASE WHEN 三路聚合
	assert.Contains(t, sql, "WHEN 'sum' THEN SUM(m.metric_value)")
	assert.Contains(t, sql, "WHEN 'avg' THEN AVG(m.metric_value)")
	assert.Contains(t, sql, "WHEN 'max' THEN MAX(m.metric_value)")
	// pct/NULL counter 不进聚合
	assert.Contains(t, sql, "AND m.statis_type IN ('sum','avg','max')")
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
	assert.Equal(t, "L.Cell.Avail.Rate", db.execArgs[2])
	assert.Equal(t, float64(0.8), db.execArgs[3])
	assert.Equal(t, "pct", db.execArgs[4])
	assert.Equal(t, "hourly", db.execArgs[5])
}

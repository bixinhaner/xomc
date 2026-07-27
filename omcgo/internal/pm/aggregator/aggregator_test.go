package aggregator

import (
	"context"
	dbsql "database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/core/jsonx"
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
	// 结果值规范化：写入前必须解析指标元数据，并按 unit/statis_type + number 配置处理。
	assert.Contains(t, sql, "WITH indicator_meta AS")
	assert.Contains(t, sql, "dictionary_indicator_meta AS")
	assert.Contains(t, sql, "COALESCE(d.unit_id, l.unit_id)")
	assert.Contains(t, sql, "COALESCE(d.statis_type, l.statis_type)")
	assert.Contains(t, sql, "LEFT JOIN indicator_meta im ON im.id = m.metric_path")
	assert.Contains(t, sql, "missing PM indicator metadata for")
	assert.Contains(t, sql, "WHEN btrim(im.unit_id) = 'number'")
	assert.Contains(t, sql, "btrim($6::text)")
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
	// args 顺序：granularity / bucket_start / bucket_end / where_start / where_end / number_process
	assert.Equal(t, []any{"hourly", w.Start, w.End, w.Start, w.End, ""}, args)
}

// #516 分区裁剪：buildCountersSQL 的源筛选必须按**分区列** time 框半开窗口
// [w.Start, w.End)，不再按非分区列 start_time（亦不按 end_time）。源表/上级表均为
// TimescaleDB 超表、按 time 列分区，改用分区列过滤后只命中目标分片、走索引（分区裁剪）。
// 等价性由 #479（time == start_time）背书，查的是同一批源行。
// 横跨小时/日/周/月各粒度（设备级源筛选统一治本）。args 顺序不变（$4=w.Start, $5=w.End）。
func Test_buildCountersSQL_FramesBy_StartTime_AllGranularities(t *testing.T) {
	cases := []struct {
		name       string
		gran       metrics.Granularity
		source     string
		target     string
		start, end time.Time
	}{
		{
			name: "hourly", gran: metrics.GranularityHourly,
			source: "pm_metrics", target: "pm_metrics_hourly",
			start: time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC),
			end:   time.Date(2026, 5, 22, 11, 0, 0, 0, time.UTC),
		},
		{
			name: "daily", gran: metrics.GranularityDaily,
			source: "pm_metrics_hourly", target: "pm_metrics_daily",
			start: time.Date(2026, 5, 22, 0, 0, 0, 0, time.UTC),
			end:   time.Date(2026, 5, 23, 0, 0, 0, 0, time.UTC),
		},
		{
			name: "weekly", gran: metrics.GranularityWeekly,
			source: "pm_metrics_daily", target: "pm_metrics_weekly",
			start: time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC),
			end:   time.Date(2026, 5, 25, 0, 0, 0, 0, time.UTC),
		},
		{
			name: "monthly", gran: metrics.GranularityMonthly,
			source: "pm_metrics_daily", target: "pm_metrics_monthly",
			start: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
			end:   time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			w := WindowSpec{Granularity: c.gran, Start: c.start, End: c.end}
			sql, args := buildCountersSQL(c.source, c.target, w)
			// 绿（#516）：按分区列 time 半开框桶 → 分区裁剪
			assert.Contains(t, sql, "AND m.time >= $4", "源筛选下界应按分区列 time")
			assert.Contains(t, sql, "AND m.time <  $5", "源筛选上界应按分区列 time")
			// 红：绝不再按非分区列 start_time 框桶（全表扫根因）
			assert.NotContains(t, sql, "AND m.start_time >= $4", "源筛选不应再按非分区列 start_time 框桶")
			assert.NotContains(t, sql, "AND m.start_time <  $5", "源筛选不应再按非分区列 start_time 框桶")
			// 红：也绝不按 end_time 框桶（旧法导致偏一格）
			assert.NotContains(t, sql, "AND m.end_time >= $4", "源筛选不应再按 end_time 框桶")
			assert.NotContains(t, sql, "AND m.end_time <  $5", "源筛选不应再按 end_time 框桶")
			// args 顺序：$4/$5 仍是 w.Start/w.End，$6 是 number 处理配置。
			assert.Equal(t, []any{string(c.gran), w.Start, w.End, w.Start, w.End, ""}, args)
		})
	}
}

// #479 改动一 行为红绿：窗口归属语义。
// 桶 end_time = start + 桶宽。按起点语义窗口 [w.Start, w.End)：
//   - 起点落入窗口的源桶 → 归属本窗口（绿）。
//   - 边界源桶（其 end_time = w.End，即下一窗口起点）按 start_time 框桶时归本窗口、
//     而不会被下一窗口（[w.End, w.End+宽)）算走（红：旧按 end_time 框会把它算到下一窗口 → 偏一格）。
//
// framedByStart 复现 SQL 谓词 start_time >= w.Start AND start_time < w.End。
func framedByStart(bucketStart, wStart, wEnd time.Time) bool {
	return !bucketStart.Before(wStart) && bucketStart.Before(wEnd)
}

// framedByEnd 复现修复前的旧谓词 end_time >= w.Start AND end_time < w.End（用于红对照）。
func framedByEnd(bucketEnd, wStart, wEnd time.Time) bool {
	return !bucketEnd.Before(wStart) && bucketEnd.Before(wEnd)
}

func Test_WindowAttribution_StartTimeFraming_AllGranularities(t *testing.T) {
	cases := []struct {
		name               string
		bktStart, bktEnd   time.Time // 源桶自身的起止
		curStart, curEnd   time.Time // 本窗口
		nextStart, nextEnd time.Time // 下一窗口
	}{
		{
			name:      "hourly",
			bktStart:  time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC),
			bktEnd:    time.Date(2026, 5, 22, 11, 0, 0, 0, time.UTC),
			curStart:  time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC),
			curEnd:    time.Date(2026, 5, 22, 11, 0, 0, 0, time.UTC),
			nextStart: time.Date(2026, 5, 22, 11, 0, 0, 0, time.UTC),
			nextEnd:   time.Date(2026, 5, 22, 12, 0, 0, 0, time.UTC),
		},
		{
			name:      "daily",
			bktStart:  time.Date(2026, 5, 22, 0, 0, 0, 0, time.UTC),
			bktEnd:    time.Date(2026, 5, 23, 0, 0, 0, 0, time.UTC),
			curStart:  time.Date(2026, 5, 22, 0, 0, 0, 0, time.UTC),
			curEnd:    time.Date(2026, 5, 23, 0, 0, 0, 0, time.UTC),
			nextStart: time.Date(2026, 5, 23, 0, 0, 0, 0, time.UTC),
			nextEnd:   time.Date(2026, 5, 24, 0, 0, 0, 0, time.UTC),
		},
		{
			name:      "weekly",
			bktStart:  time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC),
			bktEnd:    time.Date(2026, 5, 25, 0, 0, 0, 0, time.UTC),
			curStart:  time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC),
			curEnd:    time.Date(2026, 5, 25, 0, 0, 0, 0, time.UTC),
			nextStart: time.Date(2026, 5, 25, 0, 0, 0, 0, time.UTC),
			nextEnd:   time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name:      "monthly",
			bktStart:  time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
			bktEnd:    time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
			curStart:  time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
			curEnd:    time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
			nextStart: time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
			nextEnd:   time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// 绿：按 start_time 框桶 → 源桶归本窗口、不归下一窗口（无偏移）
			assert.True(t, framedByStart(c.bktStart, c.curStart, c.curEnd),
				"源桶起点落本窗口 → 应归本窗口")
			assert.False(t, framedByStart(c.bktStart, c.nextStart, c.nextEnd),
				"源桶起点不在下一窗口 → 不应归下一窗口")
			// 红对照：旧按 end_time 框 → 源桶被错算到下一窗口（end_time = 下一窗口起点）
			assert.False(t, framedByEnd(c.bktEnd, c.curStart, c.curEnd),
				"旧 end_time 框：源桶 end_time = 本窗口上界，半开区间排除本窗口（漏本窗口）")
			assert.True(t, framedByEnd(c.bktEnd, c.nextStart, c.nextEnd),
				"旧 end_time 框：源桶被错算到下一窗口 → 整体偏一格（这正是被修复的缺陷）")
		})
	}
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
	assert.Equal(t, "", args[5])
}

func Test_buildCountersSQL_PreservesNullAggregationSemantics(t *testing.T) {
	w := WindowSpec{
		Granularity: metrics.GranularityHourly,
		Start:       time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC),
		End:         time.Date(2026, 5, 22, 11, 0, 0, 0, time.UTC),
	}
	sql, _ := buildCountersSQL("pm_metrics", "pm_metrics_hourly", w)

	assert.Contains(t, sql, "WHEN 'sum' THEN SUM(m.metric_value)")
	assert.Contains(t, sql, "WHEN 'avg' THEN AVG(m.metric_value)")
	assert.Contains(t, sql, "WHEN 'max' THEN MAX(m.metric_value)")
	assert.Contains(t, sql, "WHEN 'min' THEN MIN(m.metric_value)")
	assert.NotContains(t, sql, "COALESCE(m.metric_value", "缺值不能在聚合前被当作 0")
	assert.NotContains(t, sql, "m.metric_value IS NOT NULL", "全 NULL 窗口仍应产出 NULL 聚合行")
}

func Test_buildCountersSQL_AcceptsNumberProcessArgument(t *testing.T) {
	w := WindowSpec{
		Granularity: metrics.GranularityHourly,
		Start:       time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC),
		End:         time.Date(2026, 5, 22, 11, 0, 0, 0, time.UTC),
	}
	sql, args := buildCountersSQLWithNumberProcess("pm_metrics", "pm_metrics_hourly", w, "intDown")

	assert.Contains(t, sql, "btrim($6::text)")
	assert.Equal(t, "intDown", args[5])
}

func Test_buildCountersSQL_GroupHourlyHasGroupIDConflict(t *testing.T) {
	w := WindowSpec{
		Granularity: metrics.GranularityHourly,
		Start:       time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC),
		End:         time.Date(2026, 5, 22, 11, 0, 0, 0, time.UTC),
	}
	// device_group 维度的 hourly 表（实际由 device_group.go 的 SQL 构造，此处仅断言 conflict target 切换正确）
	_ = w
	// 设备组制式治本 B 方案：冲突列尾部含 technology（与迁移 000026 唯一键逐字一致）。
	assert.Equal(t,
		"(device_group_id, metric_path, granularity, end_time, time, technology)",
		conflictTargetForTable("pm_group_metrics_hourly"))
	assert.Equal(t,
		"(device_group_id, metric_path, granularity, end_time, technology)",
		conflictTargetForTable("pm_group_metrics_daily"))
	// device_group 四表本次不分小区（决策 #1）——冲突列绝不含 object_ldn；但都必须含 technology
	for _, gt := range []string{
		"pm_group_metrics_hourly", "pm_group_metrics_daily",
		"pm_group_metrics_weekly", "pm_group_metrics_monthly",
	} {
		assert.NotContains(t, conflictTargetForTable(gt), "object_ldn",
			"group 表冲突列不应含 object_ldn："+gt)
		assert.Contains(t, conflictTargetForTable(gt), "technology",
			"group 表冲突列必须含 technology："+gt)
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

// buildKPIInsertSQL（T-B）：object_ldn 由硬写 ” 改为写实体实际 object_ldn（参数化 $N），
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
	assert.Equal(t, []any{"hourly", w.Start, w.End, w.Start, w.End, ""}, db.execArgs)
}

// #516 死判（设备维度）：第一段 counter 聚合的区间过滤谓词列必须是**分区列** time，
// 不是桶起始时间列 start_time。源表是 TimescaleDB 超表、按 time 列分区，按分区列过滤
// 才能命中目标分片走索引（分区裁剪）；按非分区列 start_time 过滤会全表扫（本单根因）。
// 仅断言 WHERE 谓词列；GROUP BY / 写入列 / 算子路由不在本断言范围。
func Test_buildCountersSQL_IntervalFilterUsesPartitionColumn_NotStartTime(t *testing.T) {
	w := WindowSpec{
		Granularity: metrics.GranularityHourly,
		Start:       time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC),
		End:         time.Date(2026, 5, 22, 11, 0, 0, 0, time.UTC),
	}
	sql, _ := buildCountersSQL("pm_metrics", "pm_metrics_hourly", w)
	// 谓词列为分区列 time
	assert.Contains(t, sql, "AND m.time >= $4", "区间过滤下界应为分区列 time")
	assert.Contains(t, sql, "AND m.time <  $5", "区间过滤上界应为分区列 time")
	// 谓词列不再是桶起始时间列 start_time（亦不是 end_time）
	assert.NotContains(t, sql, "AND m.start_time >= $4", "区间过滤不应再用非分区列 start_time")
	assert.NotContains(t, sql, "AND m.start_time <  $5", "区间过滤不应再用非分区列 start_time")
	assert.NotContains(t, sql, "AND m.end_time >= $4")
	assert.NotContains(t, sql, "AND m.end_time <  $5")
}

// #516 死判（设备组维度）：第二处第一段 counter 聚合（设备→设备组）的区间过滤谓词列
// 同样必须是分区列 time，不是 start_time。两处同型一起改。写入列 m.time/m.start_time/
// m.end_time（SELECT/GROUP BY）保持不动，故只断言 WHERE 子句的具体谓词字符串。
func Test_buildDeviceGroupSQL_IntervalFilterUsesPartitionColumn_NotStartTime(t *testing.T) {
	w := WindowSpec{
		Granularity: metrics.GranularityHourly,
		Start:       time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC),
		End:         time.Date(2026, 5, 22, 11, 0, 0, 0, time.UTC),
	}
	sql, _ := buildDeviceGroupSQL("pm_metrics_hourly", "pm_group_metrics_hourly", w)
	assert.Contains(t, sql, "WITH indicator_meta AS")
	assert.Contains(t, sql, "legacy_indicator_meta AS")
	assert.Contains(t, sql, "dictionary_indicator_meta AS")
	assert.Contains(t, sql, "SELECT metric_path AS id,")
	assert.Contains(t, sql, "NULLIF(btrim(unit), '') AS unit_id")
	assert.Contains(t, sql, "FROM pm_metric_dictionary")
	assert.Contains(t, sql, "WHERE metric_type = 'counter'")
	assert.Contains(t, sql, "FULL OUTER JOIN legacy_indicator_meta")
	assert.Contains(t, sql, "COALESCE(d.unit_id, l.unit_id)")
	assert.Contains(t, sql, "COALESCE(d.statis_type, l.statis_type)")
	assert.NotContains(t, sql, "MIN(unit_id)")
	assert.NotContains(t, sql, "MIN(statis_type)")
	assert.Contains(t, sql, "LEFT JOIN indicator_meta im ON im.id = m.metric_path")
	assert.Contains(t, sql, "missing PM indicator metadata for")
	assert.Contains(t, sql, "WHEN btrim(im.unit_id) = 'number'")
	assert.Contains(t, sql, "btrim($4::text)")
	assert.Contains(t, sql, "AND m.time >= $2", "区间过滤下界应为分区列 time")
	assert.Contains(t, sql, "AND m.time <  $3", "区间过滤上界应为分区列 time")
	assert.NotContains(t, sql, "AND m.start_time >= $2", "区间过滤不应再用非分区列 start_time")
	assert.NotContains(t, sql, "AND m.start_time <  $3", "区间过滤不应再用非分区列 start_time")
	assert.NotContains(t, sql, "AND m.end_time >= $2")
	assert.NotContains(t, sql, "AND m.end_time <  $3")
	// 写入列不动：SELECT/GROUP BY 仍取源行三时刻
	assert.Contains(t, sql, "m.time,\n    m.start_time,\n    m.end_time,", "写入列保持不动")
}

// #516 失败/空路径（设备维度）：窗口内无源行 → DB Exec 返回 0 行，AggregateCounters
// 返回 (0, nil) 不报错（INSERT ... SELECT 命中 0 源行是合法空结果，非错误）。
func Test_AggregateCounters_EmptyWindow_WritesZeroRows_NoError(t *testing.T) {
	db := &stubDB{execTag: pgconn.NewCommandTag("INSERT 0 0")}
	a := New(db, nil, nil)
	w := WindowSpec{
		Granularity: metrics.GranularityHourly,
		Start:       time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC),
		End:         time.Date(2026, 5, 22, 11, 0, 0, 0, time.UTC),
	}
	n, err := a.AggregateCounters(context.Background(), "pm_metrics", "pm_metrics_hourly", w)
	require.NoError(t, err)
	assert.Equal(t, 0, n, "窗口内无源行应写 0 行")
}

func Test_AggregateCounters_ReadsNumberProcessBeforeExec(t *testing.T) {
	db := &stubDB{execTag: pgconn.NewCommandTag("INSERT 0 1")}
	a := New(db, nil, nil)
	a.SetNumberProcessLookup(func(ctx context.Context) (string, error) {
		return "intDown", nil
	})
	w := WindowSpec{
		Granularity: metrics.GranularityHourly,
		Start:       time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC),
		End:         time.Date(2026, 5, 22, 11, 0, 0, 0, time.UTC),
	}

	_, err := a.AggregateCounters(context.Background(), "pm_metrics", "pm_metrics_hourly", w)
	require.NoError(t, err)
	require.Len(t, db.execArgs, 6)
	assert.Equal(t, "intDown", db.execArgs[5])
}

// #516 失败/空路径（设备组维度）：窗口内无源行 → AggregateDeviceGroup 返回 (0, nil)。
func Test_AggregateDeviceGroup_EmptyWindow_WritesZeroRows_NoError(t *testing.T) {
	db := &stubDB{execTag: pgconn.NewCommandTag("INSERT 0 0")}
	a := New(db, nil, nil)
	w := WindowSpec{
		Granularity: metrics.GranularityHourly,
		Start:       time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC),
		End:         time.Date(2026, 5, 22, 11, 0, 0, 0, time.UTC),
	}
	n, err := a.AggregateDeviceGroup(context.Background(), "pm_metrics_hourly", "pm_group_metrics_hourly", w)
	require.NoError(t, err)
	assert.Equal(t, 0, n, "窗口内无源行应写 0 行")
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

func (f *fakeRows) Next() bool                                   { f.idx++; return f.idx <= len(f.rows) }
func (f *fakeRows) Scan(dest ...any) error                       { return scanInto(f.rows[f.idx-1], dest) }
func (f *fakeRows) Close()                                       {}
func (f *fakeRows) Err() error                                   { return f.err }
func (f *fakeRows) CommandTag() pgconn.CommandTag                { return pgconn.NewCommandTag("") }
func (f *fakeRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (f *fakeRows) Values() ([]any, error)                       { return nil, nil }
func (f *fakeRows) RawValues() [][]byte                          { return nil }
func (f *fakeRows) Conn() *pgx.Conn                              { return nil }

// scanInto copies row values into dest pointers, supporting string, float64,
// time.Time, and nullable *string (dest **string).
func scanInto(row []any, dest []any) error {
	for i, d := range dest {
		switch dp := d.(type) {
		case *string:
			*dp = row[i].(string)
		case *float64:
			*dp = row[i].(float64)
		case *dbsql.NullFloat64:
			if row[i] == nil {
				*dp = dbsql.NullFloat64{}
			} else {
				*dp = dbsql.NullFloat64{Float64: row[i].(float64), Valid: true}
			}
		case *jsonx.Float:
			if err := dp.Scan(row[i]); err != nil {
				return err
			}
		case *time.Time:
			*dp = row[i].(time.Time)
		case **string:
			if row[i] == nil {
				*dp = nil
			} else {
				v := row[i].(string)
				*dp = &v
			}
		case **uuid.UUID:
			if row[i] == nil {
				*dp = nil
			} else {
				switch v := row[i].(type) {
				case uuid.UUID:
					*dp = &v
				case string:
					id := uuid.MustParse(v)
					*dp = &id
				default:
					return errors.New("scanInto: unsupported nullable uuid source")
				}
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

func Test_queryDeviceTable_NullMetricValueKeepsRowAndSerializesNull(t *testing.T) {
	now := time.Date(2026, 7, 7, 10, 0, 0, 0, time.UTC)
	db := &recordingDB{
		results: []pgx.Rows{
			&fakeRows{rows: [][]any{
				{
					"48BF74", "SN-899", "C000010002", "counter", nil,
					"sum", "15min", now, now, now.Add(15 * time.Minute), now, "Cellid=1", nil,
				},
			}},
		},
	}
	a := New(db, nil, nil)

	rows, err := a.queryDeviceTable(context.Background(), "pm_metrics", QueryRequest{
		Dimension:   DimensionDevice,
		Granularity: metrics.Granularity15Min,
		DeviceSNs:   []string{"SN-899"},
		MetricPaths: []string{"C000010002"},
		Limit:       10,
	})
	require.NoError(t, err)
	require.Len(t, rows, 1, "metric_value=NULL 的 DB 行仍应作为缺值指标返回")
	assert.True(t, math.IsNaN(float64(rows[0].MetricValue)), "SQL NULL 应映射为 NaN 供 JSON 层输出 null")

	body, err := json.Marshal(rows[0])
	require.NoError(t, err)
	assert.Contains(t, string(body), `"metric_value":null`)
}

func Test_AggregateKPIs_EvaluatesFormulaAndInserts(t *testing.T) {
	// device (oui=A, sn=S1) 关联一个 KPI：avail_rate = numerator / denominator
	rt := &router.KPIRoute{
		KPIs: []router.KPIDef{
			{
				IndicatorID:  "K1",
				Name:         "L.Cell.Avail.Rate",
				Unit:         "%",
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
			case 2: // #516 整桶分批载入: (oui, sn, object_ldn, metric_path, value)
				return &fakeRows{rows: [][]any{
					{"A", "S1", "", "numerator", float64(80)},
					{"A", "S1", "", "denominator", float64(100)},
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
				Unit:         "%",
				StatisType:   "pct",
				Formula:      "numerator / denominator",
				Dependencies: []string{"numerator", "denominator"},
			},
			{
				// plmn 级变体：显示名完全相同，编号不同
				IndicatorID:  "K900010059",
				Name:         "RRC连接建立成功率",
				Unit:         "%",
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
			case 2: // #516 整桶分批载入: (oui, sn, object_ldn, metric_path, value)
				return &fakeRows{rows: [][]any{
					{"A", "S1", "", "numerator", float64(95)},
					{"A", "S1", "", "denominator", float64(100)},
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

func Test_AggregateKPIs_NormalizesBeforeInsert(t *testing.T) {
	rt := &router.KPIRoute{
		KPIs: []router.KPIDef{
			{
				IndicatorID:  "Knumber",
				Name:         "NumberKPI",
				Unit:         "number",
				StatisType:   "avg",
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
			case 1:
				return &fakeRows{rows: [][]any{{"A", "S1", ""}}}, nil
			case 2:
				return &fakeRows{rows: [][]any{
					{"A", "S1", "", "Knumber", float64(2.5), "avg"},
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
	require.Len(t, db.execArgs, 9)
	assert.Equal(t, float64(2.5), db.execArgs[3], "avg KPI 即使 unit=number 也必须保留平均值小数")
}

func Test_AggregateKPIs_AvgDerivedRollsUpExisting15MinKPIValues(t *testing.T) {
	rt := &router.KPIRoute{
		KPIs: []router.KPIDef{
			{
				IndicatorID:  "KAVG001",
				Name:         "AverageKPI",
				Unit:         "number",
				StatisType:   "avg",
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
			case 1:
				return &fakeRows{rows: [][]any{{"A", "S1", ""}}}, nil
			case 2:
				assert.Contains(t, sql, "FROM pm_metrics m", "avg/sum/max/min KPI 应从 15min KPI 源表聚合")
				assert.Contains(t, sql, "m.metric_type = 'kpi'")
				assert.Contains(t, sql, "WHEN 'avg' THEN AVG(m.metric_value)")
				assert.NotContains(t, sql, "numerator", "avg 派生 KPI 不应按公式依赖 counter 重算")
				return &fakeRows{rows: [][]any{
					{"A", "S1", "", "KAVG001", float64(40), "avg"},
				}}, nil
			}
			return nil, errors.New("unexpected Query")
		},
	}

	a := New(db, kr, nil)
	w := WindowSpec{
		Granularity: metrics.GranularityDaily,
		Start:       time.Date(2026, 5, 22, 0, 0, 0, 0, time.UTC),
		End:         time.Date(2026, 5, 23, 0, 0, 0, 0, time.UTC),
	}
	n, err := a.AggregateKPIs(context.Background(), "pm_metrics_daily", w)
	require.NoError(t, err)
	assert.Equal(t, 1, n)
	require.Len(t, db.execArgs, 9)
	assert.Equal(t, "KAVG001", db.execArgs[2])
	assert.Equal(t, float64(40), db.execArgs[3])
	assert.Equal(t, "avg", db.execArgs[4])
	assert.Equal(t, "daily", db.execArgs[5])
}

func Test_AggregateKPIs_DirectRollupDoesNotWriteKPIOutsideDeviceRoute(t *testing.T) {
	avgRoute := &router.KPIRoute{
		KPIs: []router.KPIDef{{
			IndicatorID: "KAVG001",
			Name:        "AverageKPI",
			Unit:        "number",
			StatisType:  "avg",
			Formula:     "numerator / denominator",
		}},
	}
	kr := &stubKPIRouter{byDevice: map[string]*router.KPIRoute{
		"S1": avgRoute,
		"S2": {KPIs: nil},
	}}

	queryCalls := 0
	db := &stubDB{
		execTag: pgconn.NewCommandTag("INSERT 0 1"),
		queryFn: func(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
			queryCalls++
			switch queryCalls {
			case 1:
				return &fakeRows{rows: [][]any{{"A", "S1", ""}, {"A", "S2", ""}}}, nil
			case 2:
				return &fakeRows{rows: [][]any{
					{"A", "S1", "", "KAVG001", float64(40), "avg"},
					{"A", "S2", "", "KAVG001", float64(90), "avg"},
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
	require.Len(t, db.execArgs, 9)
	assert.Equal(t, "S1", db.execArgs[1])
	assert.Equal(t, float64(40), db.execArgs[3])
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
	// #516：精确命中本桶改用分区列（桶头）time = $2，不再用非分区桶尾时刻列 end_time。
	assert.Contains(t, sql, "time = $2")
	assert.NotContains(t, sql, "end_time", "#516 精确谓词应为分区列 time（桶头），不再含桶尾时刻列 end_time")
	assert.NotContains(t, sql, "time >=", "不应再有半开下界")
	assert.NotContains(t, sql, "time <", "不应再有半开上界")
	assert.Contains(t, sql, "FROM pm_metrics_hourly")
	// args：granularity / w.Start（分区列桶头）—— 不再带桶尾 w.End
	assert.Equal(t, []any{"hourly", w.Start}, args)
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
			// #516：精确谓词列 == 分区列（桶头）time，非桶尾时刻列 end_time
			assert.Contains(t, sql, "time = $2")
			assert.NotContains(t, sql, "end_time")
			assert.NotContains(t, sql, "time >=")
			assert.NotContains(t, sql, "time <")
			assert.Equal(t, []any{string(c.granularity), c.start}, args)
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
	// #516：精确命中本桶改用分区列（桶头）time = $4，不再用桶尾时刻列 end_time。
	assert.Contains(t, sql, "time = $4")
	assert.NotContains(t, sql, "end_time", "#516 精确谓词应为分区列 time（桶头），不再含桶尾时刻列 end_time")
	assert.NotContains(t, sql, "time >=", "不应再有半开下界")
	assert.NotContains(t, sql, "time <", "不应再有半开上界")
	assert.Contains(t, sql, "FROM pm_metrics_hourly")
	// args：oui / sn / granularity / w.Start（分区列桶头）—— 不再带桶尾 w.End
	assert.Equal(t, []any{"A", "S1", "hourly", w.Start}, args)
}

func Test_buildLoadCountersByObjectLdnSQL_PreciseBucketMatch_Daily(t *testing.T) {
	w := WindowSpec{
		Granularity: metrics.GranularityDaily,
		Start:       time.Date(2026, 5, 22, 0, 0, 0, 0, time.UTC),
		End:         time.Date(2026, 5, 23, 0, 0, 0, 0, time.UTC),
	}
	sql, args := buildLoadCountersByObjectLdnSQL("pm_metrics_daily", "A", "S1", w)

	assert.Contains(t, sql, "SELECT object_ldn, metric_path, metric_value")
	// #516：精确谓词列 == 分区列（桶头）time，非桶尾时刻列 end_time
	assert.Contains(t, sql, "time = $4")
	assert.NotContains(t, sql, "end_time")
	assert.NotContains(t, sql, "time >=")
	assert.NotContains(t, sql, "time <")
	assert.Contains(t, sql, "FROM pm_metrics_daily")
	assert.Equal(t, []any{"A", "S1", "daily", w.Start}, args)
}

// ---------------------------------------------------------------------------
// countersForEntity 只取本行计数器（不做跨层级配对）
// ---------------------------------------------------------------------------

// 基础小区实体只取本行小区级计数器。
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

// PLMN 实体只取本行 PLMN 级计数，不合并同 cellID 基础小区行的小区级计数器。
// 真机实测「零重叠」：PLMN 级与小区级计数器 metric_path 互斥，配对从未真"必需"。
func Test_countersForEntity_PLMN_OwnRowOnly_NoCrossLayerMerge(t *testing.T) {
	byLdn := map[string]map[string]float64{
		// 基础小区行：只上报小区级计数器
		"Cellid=111172245": {"cellLevelCnt": 1000},
		// 两个 PLMN 行：各只上报 PLMN 级计数器
		"Cellid=111172245,PLMN=00101": {"plmnLevelCnt": 200},
		"Cellid=111172245,PLMN=46068": {"plmnLevelCnt": 350},
	}
	// PLMN=00101 实体：只拿本行 PLMN 级，不再混入基础小区的小区级
	p1 := countersForEntity(byLdn, "Cellid=111172245,PLMN=00101")
	assert.Equal(t, float64(200), p1["plmnLevelCnt"], "本 PLMN 级计数")
	_, hasCellInP1 := p1["cellLevelCnt"]
	assert.False(t, hasCellInP1, "PLMN 实体不合并基础小区的小区级计数器")

	// PLMN=46068 实体：同理只拿本行
	p2 := countersForEntity(byLdn, "Cellid=111172245,PLMN=46068")
	assert.Equal(t, float64(350), p2["plmnLevelCnt"])
	_, hasCellInP2 := p2["cellLevelCnt"]
	assert.False(t, hasCellInP2)

	// 基础小区实体：只有小区级，不带 PLMN 级（行为不变）
	base := countersForEntity(byLdn, "Cellid=111172245")
	assert.Equal(t, float64(1000), base["cellLevelCnt"])
	_, hasPlmn := base["plmnLevelCnt"]
	assert.False(t, hasPlmn, "基础小区实体不应混入 PLMN 级计数器")
}

// 撞键时直接看本行值（无跨层级合并，断言「本行值」即可）。
func Test_countersForEntity_OwnRowValue(t *testing.T) {
	byLdn := map[string]map[string]float64{
		"Cellid=5":          {"shared": 100},
		"Cellid=5,PLMN=001": {"shared": 7},
	}
	p := countersForEntity(byLdn, "Cellid=5,PLMN=001")
	assert.Equal(t, float64(7), p["shared"], "本行值（不再合并基础小区行）")
}

// 设备级实体（object_ldn==""）只取空串行计数器（行为不变）。
func Test_countersForEntity_DeviceLevel_EmptyLdn(t *testing.T) {
	byLdn := map[string]map[string]float64{
		"":         {"a": 1, "b": 2},
		"Cellid=1": {"c": 9},
	}
	got := countersForEntity(byLdn, "")
	assert.Equal(t, map[string]float64{"a": 1, "b": 2}, got)
}

// NR/空小区 / 无法解析的 LDN 串不 panic，按本行降级。
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
//   - listEntitiesInBucket（2 args）：DISTINCT (oui, sn, object_ldn) 实体
//   - loadCountersByObjectLdn（4 args）：按本桶返回 (object_ldn, metric_path, value)，不折回
//
// #516：精确过滤按分区列（桶头）time = w.Start（args 末位），不再按桶尾时刻列 end_time。
// 每次 Exec 累积 args，供断言多实体多次插入。
type cellAwareDB struct {
	rows     []cellRow
	execTag  pgconn.CommandTag
	execArgs [][]any
}

func (db *cellAwareDB) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	db.execArgs = append(db.execArgs, args)
	// #516 批量化：一次 Exec 插入跨实体多行（每行 9 参），RowsAffected 须反映实际行数，
	// 否则 AggregateKPIs 返回的总行数（从 RowsAffected 累加）会与实际写入行数不符。
	return pgconn.NewCommandTag(fmt.Sprintf("INSERT 0 %d", len(args)/9)), nil
}

func (db *cellAwareDB) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return errRow{err: errors.New("cellAwareDB.QueryRow unimplemented")}
}

func (db *cellAwareDB) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	// #516：守住改动方向——SQL 必须用分区列 time 精确等值且不再含桶尾时刻列 end_time。
	if !strings.Contains(sql, "time = $") || strings.Contains(sql, "end_time") {
		return nil, errors.New("cellAwareDB: SQL 未按 #516 用分区列 time 精确等值（或仍含 end_time）")
	}
	// listEntitiesInBucket: args = [granularity, time(桶头)]
	if strings.Contains(sql, "SELECT DISTINCT device_oui, device_sn, object_ldn") {
		want := args[1].(time.Time)
		seen := map[entityKey]bool{}
		var out [][]any
		for _, r := range db.rows {
			if r.timeCol.Equal(want) {
				k := entityKey{r.oui, r.sn, r.objectLdn}
				if !seen[k] {
					seen[k] = true
					out = append(out, []any{r.oui, r.sn, r.objectLdn})
				}
			}
		}
		return &fakeRows{rows: out}, nil
	}
	// #516 整桶分批载入: args = [granularity, time(桶头), oui1, sn1, oui2, sn2, ...]
	//   → (oui, sn, object_ldn, metric_path, value)，按本桶 + (oui,sn) IN 集合过滤。
	// 守住批量化方向：必须是复合 IN 的整桶查询、非逐设备单查（含 device_sn IN 形态）。
	if strings.Contains(sql, "SELECT device_oui, device_sn, object_ldn, metric_path, metric_value") {
		if !strings.Contains(sql, "(device_oui, device_sn) IN") {
			return nil, errors.New("cellAwareDB: 整桶载入 SQL 必须用 (oui,sn) 复合 IN 批量，非逐设备单查")
		}
		want := args[1].(time.Time)
		// 收集本批设备集合（从 args[2:] 成对取 oui/sn）。
		type pair struct{ oui, sn string }
		inSet := map[pair]bool{}
		for i := 2; i+1 < len(args); i += 2 {
			inSet[pair{args[i].(string), args[i+1].(string)}] = true
		}
		var out [][]any
		for _, r := range db.rows {
			if r.timeCol.Equal(want) && inSet[pair{r.oui, r.sn}] {
				out = append(out, []any{r.oui, r.sn, r.objectLdn, r.path, r.value})
			}
		}
		return &fakeRows{rows: out}, nil
	}
	return nil, errors.New("cellAwareDB.Query unexpected sql")
}

// findKPIArgs 从累积的 Exec args 里找出 object_ldn 实参等于 wantLdn 的那一行（每行 9 参）。
// #516 批量化后一次 Exec 可含跨实体多行（9×N 参），逐行 9 参切片匹配 object_ldn。
func findKPIArgs(execArgs [][]any, wantLdn string) []any {
	for _, args := range execArgs {
		for off := 0; off+9 <= len(args); off += 9 {
			row := args[off : off+9]
			if row[8] == wantLdn {
				return row
			}
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

// 基础小区 + 两个 PLMN 三实体；混合公式（分子小区级 + 分母 PLMN 级）→
// PLMN 实体只拿本行 PLMN 级 denominator、缺基础小区 numerator → Evaluate
// 缺依赖 skip 不落库；基础小区实体只有 numerator、缺 PLMN 级 denominator → 同样 skip。
// 三个实体最终都不产 Kmix 行——决策 1 兜底：偷懒跨层级公式在错误层级行天然被求值器过滤。
func Test_AggregateKPIs_PLMN_OwnRowOnly_MixedFormulaSkipped(t *testing.T) {
	curStart := time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC)
	curEnd := time.Date(2026, 5, 22, 11, 0, 0, 0, time.UTC)

	// 混合公式 KPI：numerator 只在基础小区上报（小区级），denominator 只在 PLMN 行上报（PLMN 级）。
	mixedRoute := &router.KPIRoute{
		KPIs: []router.KPIDef{{
			IndicatorID:  "Kmix",
			Name:         "Mixed",
			Unit:         "%",
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
	// 混合公式在三个层级实体都缺一边入参 → 全部 skip 不落库
	assert.Equal(t, 0, n, "混合公式在三层级均缺入参、自然全 skip")

	assert.Nil(t, findKPIArgs(db.execArgs, "Cellid=9,PLMN=00101"),
		"PLMN 实体只有 denominator、缺基础小区 numerator → Evaluate skip")
	assert.Nil(t, findKPIArgs(db.execArgs, "Cellid=9,PLMN=46068"),
		"PLMN 实体只有 denominator、缺基础小区 numerator → Evaluate skip")
	assert.Nil(t, findKPIArgs(db.execArgs, "Cellid=9"),
		"基础小区实体只有 numerator、缺 PLMN 级 denominator → Evaluate skip")
}

// 纯小区级 KPI 在 PLMN 行不出现——PLMN 实体的计数器 map 里没有小区级计数器，
// 求值器 Evaluate 缺依赖 skip 不落库。
//
// 构造：
//   - 基础小区 Cellid=7 上报小区级 cellNum=40 / cellDen=80（纯小区级 KPI 的全部依赖）
//   - 两个 PLMN 行 PLMN=00101 / PLMN=46068 上报 PLMN 级 plmnNum=10 / plmnDen=20
//   - 纯小区级 KPI（依赖 cellNum/cellDen）+ 纯 PLMN 级 KPI（依赖 plmnNum/plmnDen）
//
// 断言：
//   - 纯小区级 KPI 只在 Cellid=7 落库；两个 PLMN 实体均不得出现纯小区级 KPI（无泄漏）
//   - 纯 PLMN 级 KPI 仍在两个 PLMN 实体落库（基础小区行无 plmn* 计数 → 基础小区实体自然不出 KplmnOnly）
func Test_AggregateKPIs_PureCellKPI_DoesNotLeakToPLMN(t *testing.T) {
	curStart := time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC)
	curEnd := time.Date(2026, 5, 22, 11, 0, 0, 0, time.UTC)

	route := &router.KPIRoute{
		KPIs: []router.KPIDef{
			{
				// 纯小区级 KPI：依赖全在基础小区上报的小区级计数器
				IndicatorID:  "KcellOnly",
				Name:         "PureCell",
				Unit:         "%",
				StatisType:   "pct",
				Formula:      "cellNum / cellDen",
				Dependencies: []string{"cellNum", "cellDen"},
			},
			{
				// 纯 PLMN 级 KPI：依赖全在 PLMN 行上报的 PLMN 级计数器
				IndicatorID:  "KplmnOnly",
				Name:         "PurePLMN",
				Unit:         "%",
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

	// 纯 PLMN 级 KPI 仍在两个 PLMN 实体落库（基础小区行无 plmn* 计数 → 基础小区实体不出 KplmnOnly）
	p1 := findKPIArgsByPath(db.execArgs, "Cellid=7,PLMN=00101", "KplmnOnly")
	require.NotNil(t, p1, "纯 PLMN 级 KPI 应在 PLMN=00101 落库")
	assert.Equal(t, float64(0.5), p1[3], "10/20=0.5")
	p2 := findKPIArgsByPath(db.execArgs, "Cellid=7,PLMN=46068", "KplmnOnly")
	require.NotNil(t, p2, "纯 PLMN 级 KPI 应在 PLMN=46068 落库")

	// 纯 PLMN 级 KPI 不应出现在基础小区（基础小区行无 plmnNum/plmnDen → Evaluate skip）
	assert.Nil(t, findKPIArgsByPath(db.execArgs, "Cellid=7", "KplmnOnly"),
		"纯 PLMN 级 KPI 不应在基础小区实体落库")
}

// 聚焦死判：构造同一 cellID 下基础小区行（X=cellNum=100/cellDen=200）+ PLMN 行
// （Y=plmnNum=10/plmnDen=20），纯 PLMN 级 KPI (plmnNum/plmnDen) → 断言 PLMN 实体 KPI
// **仅用 Y 算出 = 10/20 = 0.5**，不混入 X 的 cellNum/cellDen。
// 任务卡 hardCheck `own-row-only` 的最直白单测形态。
func Test_AggregateKPIs_PLMN_OnlyOwnRowCounters(t *testing.T) {
	curStart := time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC)
	curEnd := time.Date(2026, 5, 22, 11, 0, 0, 0, time.UTC)

	route := &router.KPIRoute{
		KPIs: []router.KPIDef{{
			IndicatorID:  "KplmnOnly",
			Name:         "PurePLMN",
			Unit:         "%",
			StatisType:   "pct",
			Formula:      "plmnNum / plmnDen",
			Dependencies: []string{"plmnNum", "plmnDen"},
		}},
	}
	db := &cellAwareDB{
		execTag: pgconn.NewCommandTag("INSERT 0 1"),
		rows: []cellRow{
			// X：基础小区行 cellNum=100 / cellDen=200（不参与 PLMN 实体求值）
			{oui: "A", sn: "S1", path: "cellNum", value: 100, objectLdn: "Cellid=42", endTime: curEnd, timeCol: curStart},
			{oui: "A", sn: "S1", path: "cellDen", value: 200, objectLdn: "Cellid=42", endTime: curEnd, timeCol: curStart},
			// Y：PLMN 行 plmnNum=10 / plmnDen=20
			{oui: "A", sn: "S1", path: "plmnNum", value: 10, objectLdn: "Cellid=42,PLMN=46001", endTime: curEnd, timeCol: curStart},
			{oui: "A", sn: "S1", path: "plmnDen", value: 20, objectLdn: "Cellid=42,PLMN=46001", endTime: curEnd, timeCol: curStart},
		},
	}
	kr := &stubKPIRouter{byDevice: map[string]*router.KPIRoute{"S1": route}}
	a := New(db, kr, nil)
	w := WindowSpec{Granularity: metrics.GranularityHourly, Start: curStart, End: curEnd}

	n, err := a.AggregateKPIs(context.Background(), "pm_metrics_hourly", w)
	require.NoError(t, err)
	// 仅 PLMN 实体产 1 行 KPI（基础小区缺 plmnNum/plmnDen → skip）
	assert.Equal(t, 1, n, "PLMN 实体只用本行计数器算出 1 行 KPI")

	plmn := findKPIArgsByPath(db.execArgs, "Cellid=42,PLMN=46001", "KplmnOnly")
	require.NotNil(t, plmn, "PLMN 实体应落库 KplmnOnly")
	assert.Equal(t, float64(0.5), plmn[3], "PLMN 实体 KPI 仅用本行 Y：10/20=0.5，不混入基础小区 X 的 100/200")

	// 基础小区实体不出该 KPI（缺 plmn* 入参）
	assert.Nil(t, findKPIArgsByPath(db.execArgs, "Cellid=42", "KplmnOnly"),
		"基础小区缺 PLMN 级入参 → Evaluate skip")
}

// ---------------------------------------------------------------------------
// 路子 B：行为单测——智能桩按分区列（桶头）time 精确过滤，证明 KPI 取本桶值、孤立桶也能算
// ---------------------------------------------------------------------------

// bucketRow 是智能桩里 target 表的一行（带 time 桶头列，供按分区列精确过滤）。
type bucketRow struct {
	oui, sn string
	path    string
	value   float64
	endTime time.Time
	timeCol time.Time // 分区列（桶头）：#516 后按此列精确等值过滤
}

// preciseBucketDB 模拟"按分区列（桶头）time 精确过滤"的 target 表：
// #516 后 listEntitiesInBucket / loadCountersByObjectLdn 传入 w.Start（= 桶头 time，args 末位），
// 只返回 time(桶头) == 该值的行——复现真实 PG 谓词行为，验证修复后只读本桶。
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
	// #516 后参数布局：
	//   listEntitiesInBucket:     args = [granularity, time(桶头)]            → (oui, sn, object_ldn)
	//   loadCountersByObjectLdn:  args = [oui, sn, granularity, time(桶头)]   → (object_ldn, metric_path, value)
	// 末位均为分区列（桶头）time = w.Start，按它精确等值过滤——复现 #516 改后 WHERE time = $N 行为。
	// 守住改动方向：SQL 必须用分区列 time 精确等值且不再含桶尾时刻列 end_time。
	if !strings.Contains(sql, "time = $") || strings.Contains(sql, "end_time") {
		return nil, errors.New("preciseBucketDB: SQL 未按 #516 用分区列 time 精确等值（或仍含 end_time）")
	}
	// listEntitiesInBucket: args = [granularity, time(桶头)]
	if strings.Contains(sql, "SELECT DISTINCT device_oui, device_sn, object_ldn") {
		want := args[1].(time.Time)
		seen := map[entityKey]bool{}
		var out [][]any
		for _, r := range db.rows {
			if r.timeCol.Equal(want) {
				k := entityKey{r.oui, r.sn, ""} // 设备级实体（object_ldn 空串）
				if !seen[k] {
					seen[k] = true
					out = append(out, []any{r.oui, r.sn, ""})
				}
			}
		}
		return &fakeRows{rows: out}, nil
	}
	// #516 整桶分批载入: args = [granularity, time(桶头), oui1, sn1, ...] → (oui, sn, object_ldn, metric_path, value)
	if strings.Contains(sql, "SELECT device_oui, device_sn, object_ldn, metric_path, metric_value") {
		if !strings.Contains(sql, "(device_oui, device_sn) IN") {
			return nil, errors.New("preciseBucketDB: 整桶载入 SQL 必须用 (oui,sn) 复合 IN 批量，非逐设备单查")
		}
		want := args[1].(time.Time)
		type pair struct{ oui, sn string }
		inSet := map[pair]bool{}
		for i := 2; i+1 < len(args); i += 2 {
			inSet[pair{args[i].(string), args[i+1].(string)}] = true
		}
		var out [][]any
		for _, r := range db.rows {
			if r.timeCol.Equal(want) && inSet[pair{r.oui, r.sn}] {
				out = append(out, []any{r.oui, r.sn, "", r.path, r.value})
			}
		}
		return &fakeRows{rows: out}, nil
	}
	return nil, errors.New("preciseBucketDB.Query unexpected sql")
}

func avilRateRoute() *router.KPIRoute {
	return &router.KPIRoute{
		KPIs: []router.KPIDef{
			{
				IndicatorID:  "K1",
				Name:         "L.Cell.Avail.Rate",
				Unit:         "%",
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

// ---------------------------------------------------------------------------
// #516 边界测试（关键）：第二段精确过滤换分区列后，查目标桶只命中本桶，不漏不串相邻桶。
//
// 构造相邻三桶（prev / target / next，桶头各差一个桶宽）的源行，s516FixtureDB 按查询里
// 传入的「分区列 time 桶头值」精确等值过滤夹具（模拟 #516 改后的 WHERE time = $N 单分片命中）。
// 断言：用 target 桶窗口查实体枚举 / 取计数器，只拿到 target 桶的实体与计数器，
// 不漏（target 全在）、不串（prev/next 一个都不串入）。
// 这道测试守的是「桶头精确等值 → 唯一本桶」的不变量；若误改回半开区间或桶尾时刻列，会漏/串。
// ---------------------------------------------------------------------------

// s516FixtureRow 是 target 表（pm_metrics_hourly 等）夹具里的一行计数器。
type s516FixtureRow struct {
	oui, sn, objectLdn, metricPath string
	value                          float64
	bucketHead                     time.Time // 该行所属桶的桶头（== time 列，#479 后 time==start_time==桶头）
}

// s516FixtureDB 是按「分区列 time 桶头」精确过滤的夹具型 PgQuerier，
// 专用于 #516 边界测试：只实现 listEntities / loadCounters 两条 SELECT 的过滤语义。
type s516FixtureDB struct {
	fixture []s516FixtureRow
}

func (b *s516FixtureDB) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	return pgconn.NewCommandTag(""), nil
}
func (b *s516FixtureDB) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return countRow{n: 0}
}

// Query 模拟 #516 改后两条 SELECT 的分区列精确过滤：
//   - listEntities SQL：args = [granularity, bucketHead]，最后一个 arg 即分区列 time 桶头；
//   - loadCounters SQL：args = [oui, sn, granularity, bucketHead]，最后一个 arg 即分区列 time 桶头。
//
// 仅按桶头精确等值（外加 loadCounters 的 oui/sn）过滤夹具，正是 #516 改后 WHERE time = $N 的行为。
func (b *s516FixtureDB) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	// 断言被测 SQL 确实用分区列 time 精确等值、且不含桶尾时刻列 end_time（守住改动方向）。
	if !strings.Contains(sql, "time = $") || strings.Contains(sql, "end_time") {
		return nil, errors.New("s516FixtureDB: SQL 未按 #516 用分区列 time 精确等值（或仍含 end_time）")
	}
	head, _ := args[len(args)-1].(time.Time)

	if strings.Contains(sql, "SELECT DISTINCT device_oui, device_sn, object_ldn") {
		// listEntitiesInBucket：按桶头过滤后去重 (oui, sn, object_ldn)
		seen := map[entityKey]struct{}{}
		var out [][]any
		for _, r := range b.fixture {
			if !r.bucketHead.Equal(head) {
				continue
			}
			k := entityKey{oui: r.oui, sn: r.sn, objectLdn: r.objectLdn}
			if _, dup := seen[k]; dup {
				continue
			}
			seen[k] = struct{}{}
			out = append(out, []any{r.oui, r.sn, r.objectLdn})
		}
		return &fakeRows{rows: out}, nil
	}

	// loadCountersByObjectLdn：args = [oui, sn, granularity, bucketHead]
	oui, _ := args[0].(string)
	sn, _ := args[1].(string)
	var out [][]any
	for _, r := range b.fixture {
		if !r.bucketHead.Equal(head) || r.oui != oui || r.sn != sn {
			continue
		}
		out = append(out, []any{r.objectLdn, r.metricPath, r.value})
	}
	return &fakeRows{rows: out}, nil
}

// 三相邻桶夹具：prev/target/next 桶头各差 1 小时。每桶放各自独立的实体与计数器值。
func adjacentThreeBucketFixture(prevHead, targetHead, nextHead time.Time) []s516FixtureRow {
	return []s516FixtureRow{
		// prev 桶（桶头 = targetHead - 1h）
		{oui: "A", sn: "S1", objectLdn: "Cellid=1", metricPath: "c", value: 11, bucketHead: prevHead},
		{oui: "A", sn: "S1", objectLdn: "PREV_ONLY", metricPath: "c", value: 12, bucketHead: prevHead},
		// target 桶（本桶，应被精确命中）
		{oui: "A", sn: "S1", objectLdn: "Cellid=1", metricPath: "c", value: 21, bucketHead: targetHead},
		{oui: "A", sn: "S1", objectLdn: "Cellid=2", metricPath: "c", value: 22, bucketHead: targetHead},
		// next 桶（桶头 = targetHead + 1h）
		{oui: "A", sn: "S1", objectLdn: "Cellid=1", metricPath: "c", value: 31, bucketHead: nextHead},
		{oui: "A", sn: "S1", objectLdn: "NEXT_ONLY", metricPath: "c", value: 32, bucketHead: nextHead},
	}
}

func Test_listEntitiesInBucket_DoesNotLeakToAdjacentBuckets(t *testing.T) {
	targetHead := time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC)
	prevHead := targetHead.Add(-time.Hour)
	nextHead := targetHead.Add(time.Hour)
	db := &s516FixtureDB{fixture: adjacentThreeBucketFixture(prevHead, targetHead, nextHead)}
	a := New(db, nil, nil)

	w := WindowSpec{Granularity: metrics.GranularityHourly, Start: targetHead, End: targetHead.Add(time.Hour)}
	ents, err := a.listEntitiesInBucket(context.Background(), "pm_metrics_hourly", w)
	require.NoError(t, err)

	got := map[string]struct{}{}
	for _, e := range ents {
		got[e.objectLdn] = struct{}{}
	}
	// 不漏：target 桶两实体都在
	assert.Contains(t, got, "Cellid=1", "应命中本桶 Cellid=1")
	assert.Contains(t, got, "Cellid=2", "应命中本桶 Cellid=2")
	// 不串：相邻桶独有实体一个都不串入
	assert.NotContains(t, got, "PREV_ONLY", "不应串到前一桶")
	assert.NotContains(t, got, "NEXT_ONLY", "不应串到后一桶")
	assert.Len(t, ents, 2, "目标桶恰两个实体，无相邻桶混入")
}

func Test_loadCountersByObjectLdn_DoesNotLeakToAdjacentBuckets(t *testing.T) {
	targetHead := time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC)
	prevHead := targetHead.Add(-time.Hour)
	nextHead := targetHead.Add(time.Hour)
	db := &s516FixtureDB{fixture: adjacentThreeBucketFixture(prevHead, targetHead, nextHead)}
	a := New(db, nil, nil)

	w := WindowSpec{Granularity: metrics.GranularityHourly, Start: targetHead, End: targetHead.Add(time.Hour)}
	byLdn, err := a.loadCountersByObjectLdn(context.Background(), "pm_metrics_hourly", "A", "S1", w)
	require.NoError(t, err)

	// 不漏 + 取的是本桶的值（target Cellid=1=21, Cellid=2=22），不是相邻桶的 11/31。
	require.Contains(t, byLdn, "Cellid=1")
	require.Contains(t, byLdn, "Cellid=2")
	assert.Equal(t, 21.0, byLdn["Cellid=1"]["c"], "应取本桶值 21，非前桶 11 / 后桶 31")
	assert.Equal(t, 22.0, byLdn["Cellid=2"]["c"], "应取本桶值 22")
	// 不串：相邻桶独有实体不出现
	assert.NotContains(t, byLdn, "PREV_ONLY", "不应串到前一桶")
	assert.NotContains(t, byLdn, "NEXT_ONLY", "不应串到后一桶")
	assert.Len(t, byLdn, 2, "目标桶恰两个实体计数器，无相邻桶混入")
}

func Test_loadCountersByObjectLdn_SkipsNullMetricValues(t *testing.T) {
	bucket := time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC)
	db := &recordingDB{results: []pgx.Rows{&fakeRows{rows: [][]any{
		{"Cellid=1", "C-null", nil},
		{"Cellid=1", "C-real", 42.0},
	}}}}
	a := New(db, nil, nil)

	byLdn, err := a.loadCountersByObjectLdn(context.Background(), "pm_metrics_hourly", "A", "S1",
		WindowSpec{Granularity: metrics.GranularityHourly, Start: bucket, End: bucket.Add(time.Hour)})

	require.NoError(t, err)
	require.Contains(t, byLdn, "Cellid=1")
	assert.NotContains(t, byLdn["Cellid=1"], "C-null", "NULL counter 应按缺依赖处理，不进入 KPI 入参")
	assert.Equal(t, 42.0, byLdn["Cellid=1"]["C-real"])
}

func Test_loadCountersForDevices_SkipsNullMetricValues(t *testing.T) {
	bucket := time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC)
	db := &recordingDB{results: []pgx.Rows{&fakeRows{rows: [][]any{
		{"A", "S1", "Cellid=1", "C-null", nil},
		{"A", "S1", "Cellid=1", "C-real", 7.0},
	}}}}
	a := New(db, nil, nil)

	byDevice, err := a.loadCountersForDevices(context.Background(), "pm_metrics_hourly", []deviceKey{{"A", "S1"}},
		WindowSpec{Granularity: metrics.GranularityHourly, Start: bucket, End: bucket.Add(time.Hour)})

	require.NoError(t, err)
	byLdn := byDevice[deviceKey{"A", "S1"}]
	require.Contains(t, byLdn, "Cellid=1")
	assert.NotContains(t, byLdn["Cellid=1"], "C-null", "NULL counter 应按缺依赖处理，不进入 KPI 入参")
	assert.Equal(t, 7.0, byLdn["Cellid=1"]["C-real"])
}

// ===========================================================================
// #516 阶段3：第二段 N+1 批量化 —— 调用形状 / 等价 / 分批控内存 / 失败路径
//
// 核心命题：第二段从"逐设备取计数器 + 逐实体单条 INSERT"改为"按设备分批整桶载入 +
// 全桶批量写"。下列测试守：①取计数器是整桶批量查询（复合 IN）非逐设备循环单查；
// ②KPI 写入是批量插入（一次多行 VALUES）非逐实体单条；③同一桶批量化前后 KPI 行集合逐字段
// 完全一致；④多设备大桶分批控内存生效；⑤某设备路由/计数器缺失时 WARN 跳过不阻塞整体。
// ===========================================================================

// s3RecordingDB 记录每次 Query / Exec 的 SQL 与 args，供断言"调用形状/批数/单条行数"。
// counters：deviceKey → object_ldn → metric_path → value（整桶真实数据，按桶头 time 过滤）。
type s3RecordingDB struct {
	entities  []entityKey                                 // 本桶实体（listEntitiesInBucket 返回）
	counters  map[deviceKey]map[string]map[string]float64 // 整桶计数器
	loadSQLs  []string                                    // 每次"取计数器"查询的 SQL
	loadArgs  [][]any                                     // 每次"取计数器"查询的 args
	execSQLs  []string                                    // 每次 KPI 写入的 SQL
	execArgs  [][]any                                     // 每次 KPI 写入的 args
	listCount int                                         // listEntitiesInBucket 调用次数
}

func (db *s3RecordingDB) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	db.execSQLs = append(db.execSQLs, sql)
	db.execArgs = append(db.execArgs, args)
	return pgconn.NewCommandTag(fmt.Sprintf("INSERT 0 %d", len(args)/9)), nil
}
func (db *s3RecordingDB) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return errRow{err: errors.New("s3RecordingDB.QueryRow unimplemented")}
}
func (db *s3RecordingDB) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	if strings.Contains(sql, "SELECT DISTINCT device_oui, device_sn, object_ldn") {
		db.listCount++
		var out [][]any
		for _, e := range db.entities {
			out = append(out, []any{e.oui, e.sn, e.objectLdn})
		}
		return &fakeRows{rows: out}, nil
	}
	if strings.Contains(sql, "SELECT device_oui, device_sn, object_ldn, metric_path, metric_value") {
		db.loadSQLs = append(db.loadSQLs, sql)
		db.loadArgs = append(db.loadArgs, args)
		// 从复合 IN 还原本批设备集合（args[2:] 成对）。
		type pair struct{ oui, sn string }
		inSet := map[pair]bool{}
		for i := 2; i+1 < len(args); i += 2 {
			inSet[pair{args[i].(string), args[i+1].(string)}] = true
		}
		var out [][]any
		for dk, byLdn := range db.counters {
			if !inSet[pair{dk.oui, dk.sn}] {
				continue
			}
			for ldn, m := range byLdn {
				for path, v := range m {
					out = append(out, []any{dk.oui, dk.sn, ldn, path, v})
				}
			}
		}
		return &fakeRows{rows: out}, nil
	}
	return nil, errors.New("s3RecordingDB.Query unexpected sql: " + sql)
}

// 调用形状死判：取计数器是整桶批量查询（复合 IN）、非逐设备循环单查；KPI 写入是批量多行 INSERT。
func Test_AggregateKPIs_S3_CallShape_BatchLoadAndBatchInsert(t *testing.T) {
	curStart := time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC)
	curEnd := curStart.Add(time.Hour)
	w := WindowSpec{Granularity: metrics.GranularityHourly, Start: curStart, End: curEnd}

	// 3 个设备各一个设备级实体，各有 numerator/denominator → 各产 1 行 KPI。
	db := &s3RecordingDB{
		entities: []entityKey{
			{oui: "A", sn: "S1", objectLdn: ""},
			{oui: "A", sn: "S2", objectLdn: ""},
			{oui: "A", sn: "S3", objectLdn: ""},
		},
		counters: map[deviceKey]map[string]map[string]float64{
			{"A", "S1"}: {"": {"numerator": 10, "denominator": 100}},
			{"A", "S2"}: {"": {"numerator": 20, "denominator": 100}},
			{"A", "S3"}: {"": {"numerator": 30, "denominator": 100}},
		},
	}
	kr := &stubKPIRouter{byDevice: map[string]*router.KPIRoute{
		"S1": avilRateRoute(), "S2": avilRateRoute(), "S3": avilRateRoute(),
	}}
	a := New(db, kr, nil)

	n, err := a.AggregateKPIs(context.Background(), "pm_metrics_hourly", w)
	require.NoError(t, err)
	assert.Equal(t, 3, n, "3 个设备各 1 行 KPI")

	// ① 取计数器：整桶批量（3 设备 ≤ batchSize → 1 次查询），且用复合 IN，非逐设备单查（3 次）。
	require.Len(t, db.loadSQLs, 1, "3 设备应只发 1 次整桶批量取计数器查询，非逐设备 3 次")
	assert.Contains(t, db.loadSQLs[0], "(device_oui, device_sn) IN", "取计数器必须用 (oui,sn) 复合 IN 批量")
	assert.NotContains(t, db.loadSQLs[0], "device_sn  = $2", "不应是逐设备单查（单 device_sn 等值）")

	// ② KPI 写入：批量插入（1 次 Exec 含 3 行，非逐实体 3 次单条 INSERT）。
	require.Len(t, db.execSQLs, 1, "3 实体 KPI 应只发 1 次批量 INSERT，非逐实体 3 次")
	assert.Len(t, db.execArgs[0], 27, "1 次 Exec 含 3 行 × 9 参 = 27 参（多行 VALUES）")
	assert.Equal(t, 1, db.listCount, "listEntitiesInBucket 仍只 1 次")
}

// 等价死判（关键）：同一桶数据，批量化路径产出的 KPI 行集合（实体 object_ldn / metric_path /
// metric_value）必须与"逐实体求值"参照实现逐字段完全一致。
// 参照实现：直接复用 countersForEntity + evalKPIs（求值逻辑不动），按实体独立算一遍，
// 与 AggregateKPIs 批量路径产出的 Exec 行做集合比对。
func Test_AggregateKPIs_S3_EquivalentRowSet_BeforeAfterBatching(t *testing.T) {
	curStart := time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC)
	curEnd := curStart.Add(time.Hour)
	w := WindowSpec{Granularity: metrics.GranularityHourly, Start: curStart, End: curEnd}

	// 混合数据：两设备、各含基础小区 + 两 PLMN（覆盖跨层级配对 + 自身门槛），纯小区级 + 纯 PLMN 级 KPI。
	route := &router.KPIRoute{KPIs: []router.KPIDef{
		{IndicatorID: "KcellOnly", Name: "PureCell", Unit: "%", StatisType: "pct", Formula: "cellNum / cellDen", Dependencies: []string{"cellNum", "cellDen"}},
		{IndicatorID: "KplmnOnly", Name: "PurePLMN", Unit: "%", StatisType: "pct", Formula: "plmnNum / plmnDen", Dependencies: []string{"plmnNum", "plmnDen"}},
	}}
	counters := map[deviceKey]map[string]map[string]float64{
		{"A", "S1"}: {
			"Cellid=7":            {"cellNum": 40, "cellDen": 80},
			"Cellid=7,PLMN=00101": {"plmnNum": 10, "plmnDen": 20},
			"Cellid=7,PLMN=46068": {"plmnNum": 15, "plmnDen": 30},
		},
		{"A", "S2"}: {
			"Cellid=9":            {"cellNum": 5, "cellDen": 25},
			"Cellid=9,PLMN=00101": {"plmnNum": 7, "plmnDen": 14},
		},
	}
	entities := []entityKey{
		{oui: "A", sn: "S1", objectLdn: "Cellid=7"},
		{oui: "A", sn: "S1", objectLdn: "Cellid=7,PLMN=00101"},
		{oui: "A", sn: "S1", objectLdn: "Cellid=7,PLMN=46068"},
		{oui: "A", sn: "S2", objectLdn: "Cellid=9"},
		{oui: "A", sn: "S2", objectLdn: "Cellid=9,PLMN=00101"},
	}

	// ── 参照实现（"批量化前"逐实体求值，等价对照） ──
	type kpiKey struct{ oui, sn, ldn, path string }
	want := map[kpiKey]float64{}
	refA := New(&s3RecordingDB{}, nil, nil) // 仅借 evalKPIs（不触 DB）
	for _, ent := range entities {
		byLdn := counters[deviceKey{ent.oui, ent.sn}]
		c := countersForEntity(byLdn, ent.objectLdn)
		rows, err := refA.evalKPIs(ent, route.KPIs, c, "")
		require.NoError(t, err)
		for _, r := range rows {
			want[kpiKey{ent.oui, ent.sn, ent.objectLdn, r.path}] = r.value
		}
	}
	require.NotEmpty(t, want, "参照实现应产出若干 KPI 行")

	// ── 批量化路径（实际被测）──
	db := &s3RecordingDB{entities: entities, counters: counters}
	kr := &stubKPIRouter{byDevice: map[string]*router.KPIRoute{"S1": route, "S2": route}}
	a := New(db, kr, nil)
	n, err := a.AggregateKPIs(context.Background(), "pm_metrics_hourly", w)
	require.NoError(t, err)

	// 从批量 Exec 还原产出行集合（每行 9 参：oui,sn,path,val,...,object_ldn[idx8]）。
	got := map[kpiKey]float64{}
	for _, args := range db.execArgs {
		for off := 0; off+9 <= len(args); off += 9 {
			row := args[off : off+9]
			got[kpiKey{row[0].(string), row[1].(string), row[8].(string), row[2].(string)}] = row[3].(float64)
		}
	}

	// 逐字段完全一致（实体 object_ldn + metric_path + metric_value）。
	assert.Equal(t, want, got, "批量化前后 KPI 行集合必须逐字段完全一致")
	assert.Equal(t, len(want), n, "返回行数应等于产出 KPI 行数")
}

// 分批控内存死判：多设备大桶（设备数 > loadCountersBatchSize）→ 取计数器按设备分多批，
// 每批一次查询且每批设备数 ≤ batchSize（不一次性吞整桶），全设备 KPI 仍全部算出。
func Test_AggregateKPIs_S3_BatchesDevicesToBoundMemory(t *testing.T) {
	curStart := time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC)
	curEnd := curStart.Add(time.Hour)
	w := WindowSpec{Granularity: metrics.GranularityHourly, Start: curStart, End: curEnd}

	// 构造 loadCountersBatchSize*2 + 1 个设备 → 至少 3 批。
	nDev := loadCountersBatchSize*2 + 1
	entities := make([]entityKey, 0, nDev)
	counters := map[deviceKey]map[string]map[string]float64{}
	routes := map[string]*router.KPIRoute{}
	for i := 0; i < nDev; i++ {
		sn := fmt.Sprintf("S%05d", i)
		entities = append(entities, entityKey{oui: "A", sn: sn, objectLdn: ""})
		counters[deviceKey{"A", sn}] = map[string]map[string]float64{"": {"numerator": float64(i + 1), "denominator": 100}}
		routes[sn] = avilRateRoute()
	}
	db := &s3RecordingDB{entities: entities, counters: counters}
	a := New(db, &stubKPIRouter{byDevice: routes}, nil)

	n, err := a.AggregateKPIs(context.Background(), "pm_metrics_hourly", w)
	require.NoError(t, err)
	assert.Equal(t, nDev, n, "全部设备 KPI 都应算出")

	// 取计数器分批：批数 = ceil(nDev / batchSize) = 3，且绝不是 1 次整桶吞下、也不是逐设备 nDev 次。
	wantBatches := (nDev + loadCountersBatchSize - 1) / loadCountersBatchSize
	assert.Equal(t, wantBatches, len(db.loadSQLs), "应按 loadCountersBatchSize 分批取计数器（控内存）")
	assert.Greater(t, len(db.loadSQLs), 1, "多设备大桶不应一次性吞整桶（须分批）")
	assert.Less(t, len(db.loadSQLs), nDev, "也不应退化为逐设备单查")
	// 每批设备数 ≤ batchSize（从复合 IN 的 args 还原：(len-2)/2 个设备）。
	for _, args := range db.loadArgs {
		devInBatch := (len(args) - 2) / 2
		assert.LessOrEqual(t, devInBatch, loadCountersBatchSize, "单批设备数不得超过 batchSize")
	}
}

// 失败路径①：某设备 KPI 路由缺失（LookupByDevice 报错）→ WARN 跳过该设备，其余设备照常产出。
func Test_AggregateKPIs_S3_SkipsDeviceWithMissingRoute_NotBlocking(t *testing.T) {
	curStart := time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC)
	w := WindowSpec{Granularity: metrics.GranularityHourly, Start: curStart, End: curStart.Add(time.Hour)}

	db := &s3RecordingDB{
		entities: []entityKey{
			{oui: "A", sn: "OK1", objectLdn: ""},
			{oui: "A", sn: "NOROUTE", objectLdn: ""}, // 路由查不到
			{oui: "A", sn: "OK2", objectLdn: ""},
		},
		counters: map[deviceKey]map[string]map[string]float64{
			{"A", "OK1"}:     {"": {"numerator": 50, "denominator": 100}},
			{"A", "NOROUTE"}: {"": {"numerator": 99, "denominator": 100}},
			{"A", "OK2"}:     {"": {"numerator": 70, "denominator": 100}},
		},
	}
	// NOROUTE 不在 byDevice → LookupByDevice 返回 error。
	kr := &stubKPIRouter{byDevice: map[string]*router.KPIRoute{"OK1": avilRateRoute(), "OK2": avilRateRoute()}}
	a := New(db, kr, nil)

	n, err := a.AggregateKPIs(context.Background(), "pm_metrics_hourly", w)
	require.NoError(t, err, "单设备路由缺失不应让整体报错")
	assert.Equal(t, 2, n, "仅 OK1/OK2 产 KPI，NOROUTE 被跳过")
	// NOROUTE 不应出现在任何写入行（按 sn 检查）。
	for _, args := range db.execArgs {
		for off := 0; off+9 <= len(args); off += 9 {
			assert.NotEqual(t, "NOROUTE", args[off+1], "NOROUTE 设备不应有 KPI 写入行")
		}
	}
}

// 失败路径②：某设备计数器整桶载入没拿到（数据态缺失）→ WARN 跳过该设备不阻塞整体。
// 用 missingCountersDB：listEntities 返回 3 设备，但整桶载入只回 2 个设备的计数器。
func Test_AggregateKPIs_S3_SkipsDeviceWithMissingCounters_NotBlocking(t *testing.T) {
	curStart := time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC)
	w := WindowSpec{Granularity: metrics.GranularityHourly, Start: curStart, End: curStart.Add(time.Hour)}

	db := &s3RecordingDB{
		entities: []entityKey{
			{oui: "A", sn: "HAS1", objectLdn: ""},
			{oui: "A", sn: "NOCNT", objectLdn: ""}, // 列在实体里，但整桶载入无其计数器
			{oui: "A", sn: "HAS2", objectLdn: ""},
		},
		counters: map[deviceKey]map[string]map[string]float64{
			{"A", "HAS1"}: {"": {"numerator": 40, "denominator": 100}},
			{"A", "HAS2"}: {"": {"numerator": 60, "denominator": 100}},
			// 故意不放 NOCNT
		},
	}
	kr := &stubKPIRouter{byDevice: map[string]*router.KPIRoute{
		"HAS1": avilRateRoute(), "NOCNT": avilRateRoute(), "HAS2": avilRateRoute(),
	}}
	a := New(db, kr, nil)

	n, err := a.AggregateKPIs(context.Background(), "pm_metrics_hourly", w)
	require.NoError(t, err, "单设备计数器缺失不应让整体报错")
	assert.Equal(t, 2, n, "仅 HAS1/HAS2 产 KPI，NOCNT 计数器缺失被跳过")
	for _, args := range db.execArgs {
		for off := 0; off+9 <= len(args); off += 9 {
			assert.NotEqual(t, "NOCNT", args[off+1], "NOCNT 设备无计数器，不应有 KPI 写入行")
		}
	}
}

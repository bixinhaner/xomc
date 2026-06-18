package aggregator

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/pm/metrics"
)

// SQL 文本断言：device_group 维度聚合 SQL 必须 JOIN device_dim + device_group_member_dim
// （KPI/时序库物理分离后用本库影子表），SELECT 列 + ON CONFLICT 含 device_group_id。

func Test_buildDeviceGroupSQL_HourlyGroupTableHasIDAndJoins(t *testing.T) {
	w := WindowSpec{
		Granularity: metrics.GranularityHourly,
		Start:       time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC),
		End:         time.Date(2026, 5, 22, 11, 0, 0, 0, time.UTC),
	}
	sql, args := buildDeviceGroupSQL("pm_metrics_hourly", "pm_group_metrics_hourly", w)

	// 双 JOIN（影子表）
	assert.Contains(t, sql, "JOIN device_dim d")
	assert.Contains(t, sql, "JOIN device_group_member_dim dgm")
	assert.Contains(t, sql, "ON d.oui = m.device_oui AND d.serial_number = m.device_sn")
	assert.Contains(t, sql, "ON dgm.device_id = d.id")

	// GROUP BY 维度键含制式 + 源桶时刻（issue #395：按「组 × 制式 × 指标 × 源桶时刻」分组，
	// 每个源小时各成一行，消除「整窗压成单点」）
	assert.Contains(t, sql, "GROUP BY dgm.group_id, d.technology, m.metric_path, m.statis_type, m.time, m.start_time, m.end_time")
	// SELECT 带出 d.technology，insertCols 含 technology 列
	assert.Contains(t, sql, "d.technology")

	// issue #395：写入 time/start_time/end_time 取源行自身桶时刻（m.time/m.start_time/m.end_time），
	// 而非窗口起点 $2 —— 与设备单维度聚合表逐档对齐、无 1 小时偏移
	assert.Contains(t, sql, "m.time,\n    m.start_time,\n    m.end_time,")

	// hourly group 表含 id 列（hypertable）+ technology 列
	assert.Contains(t, sql, "INSERT INTO pm_group_metrics_hourly (id, device_group_id, technology")
	assert.Contains(t, sql, "gen_random_uuid()")

	// ON CONFLICT 含 device_group_id + technology（不含 device_oui/sn），列序与迁移 000026 唯一键一致
	assert.Contains(t, sql, "ON CONFLICT (device_group_id, metric_path, granularity, end_time, time, technology)")

	// issue #395：time 来自源行（m.time）后不再需要 bucketStart 占位，args 仅余
	// granularity / whereStart / whereEnd 三参。
	assert.Equal(t, []any{"hourly", w.Start, w.End}, args)
}

// #516 分区裁剪：device_group 源筛选必须按**分区列** time 框半开窗口 [w.Start, w.End)，
// 不再按非分区列 start_time（亦不按 end_time）。deviceTarget 为 TimescaleDB 超表、按 time
// 列分区，改用分区列过滤后只命中目标分片、走索引。等价性由 #479（time == start_time）背书。
// 写入列（SELECT/GROUP BY 的 m.time/m.start_time/m.end_time）不动，仅换 WHERE 谓词列。
// 横跨小时/日/周/月各粒度；args 顺序不变（$2=w.Start, $3=w.End）。
func Test_buildDeviceGroupSQL_FramesBy_StartTime_AllGranularities(t *testing.T) {
	cases := []struct {
		name       string
		gran       metrics.Granularity
		deviceTbl  string
		groupTbl   string
		start, end time.Time
	}{
		{
			name: "hourly", gran: metrics.GranularityHourly,
			deviceTbl: "pm_metrics_hourly", groupTbl: "pm_group_metrics_hourly",
			start: time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC),
			end:   time.Date(2026, 5, 22, 11, 0, 0, 0, time.UTC),
		},
		{
			name: "daily", gran: metrics.GranularityDaily,
			deviceTbl: "pm_metrics_daily", groupTbl: "pm_group_metrics_daily",
			start: time.Date(2026, 5, 22, 0, 0, 0, 0, time.UTC),
			end:   time.Date(2026, 5, 23, 0, 0, 0, 0, time.UTC),
		},
		{
			name: "weekly", gran: metrics.GranularityWeekly,
			deviceTbl: "pm_metrics_weekly", groupTbl: "pm_group_metrics_weekly",
			start: time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC),
			end:   time.Date(2026, 5, 25, 0, 0, 0, 0, time.UTC),
		},
		{
			name: "monthly", gran: metrics.GranularityMonthly,
			deviceTbl: "pm_metrics_monthly", groupTbl: "pm_group_metrics_monthly",
			start: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
			end:   time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			w := WindowSpec{Granularity: c.gran, Start: c.start, End: c.end}
			sql, args := buildDeviceGroupSQL(c.deviceTbl, c.groupTbl, w)
			// 绿（#516）：WHERE 谓词按分区列 time 半开框桶 → 分区裁剪
			assert.Contains(t, sql, "AND m.time >= $2", "源筛选下界应按分区列 time")
			assert.Contains(t, sql, "AND m.time <  $3", "源筛选上界应按分区列 time")
			// 红：WHERE 绝不再按非分区列 start_time 框桶（全表扫根因）
			assert.NotContains(t, sql, "AND m.start_time >= $2", "源筛选不应再按非分区列 start_time 框桶")
			assert.NotContains(t, sql, "AND m.start_time <  $3", "源筛选不应再按非分区列 start_time 框桶")
			// 红：也绝不按 end_time 框桶
			assert.NotContains(t, sql, "AND m.end_time >= $2", "源筛选不应再按 end_time 框桶")
			assert.NotContains(t, sql, "AND m.end_time <  $3", "源筛选不应再按 end_time 框桶")
			// 写入 time/start_time/end_time 仍取源行自身桶时刻（逐档对齐、无偏移；写入列不动）
			assert.Contains(t, sql, "m.time,\n    m.start_time,\n    m.end_time,")
			assert.Equal(t, []any{string(c.gran), w.Start, w.End}, args)
		})
	}
}

// issue #395 专项：组聚合必须按源行自身桶时刻分桶并取时刻，确保
//  1. 写入 time/start_time/end_time 取源行（m.time/...）而非窗口起点 → 与设备单维度无 1 小时偏移
//  2. GROUP BY 含源桶时刻 → 跨多小时窗口产出多行（每源小时各一行），不被压成单点
func Test_buildDeviceGroupSQL_Issue395_BucketsBySourceTime_NoOffset(t *testing.T) {
	w := WindowSpec{
		Granularity: metrics.GranularityHourly,
		// 故意用一个跨多小时的宽窗（>=3 小时），验证不再依赖窗口起点写时刻
		Start: time.Date(2026, 6, 14, 4, 0, 0, 0, time.UTC),
		End:   time.Date(2026, 6, 14, 8, 0, 0, 0, time.UTC),
	}
	sql, args := buildDeviceGroupSQL("pm_metrics_hourly", "pm_group_metrics_hourly", w)

	// 死判 no-time-offset：写入的 time/start_time/end_time 取源行三列，不出现把 time 硬编码成
	// 窗口起点的 "$2,\n    $2," 旧模式
	assert.Contains(t, sql, "m.time,\n    m.start_time,\n    m.end_time,",
		"写入时刻应取源行 m.time/m.start_time/m.end_time")
	assert.NotContains(t, sql, "$1,\n    $2,\n    $2,\n    $3,",
		"不应再把 time/start_time 硬编码成窗口起点 $2")

	// 死判 multi-bucket-rows：GROUP BY 含源桶时刻 → 每源小时各成一行
	assert.Contains(t, sql, "m.time, m.start_time, m.end_time",
		"GROUP BY 必须含源桶时刻才能逐小时分桶")

	// #516 分区裁剪：源筛选改按分区列 time 半开区间框桶（与 buildCountersSQL 同步），
	// 获得分区裁剪 + 索引。不再按非分区列 start_time（全表扫根因），亦不按 end_time。
	assert.Contains(t, sql, "AND m.time >= $2")
	assert.Contains(t, sql, "AND m.time <  $3")
	assert.NotContains(t, sql, "AND m.start_time >= $2", "源筛选不应再按非分区列 start_time 框桶")
	assert.NotContains(t, sql, "AND m.start_time <  $3", "源筛选不应再按非分区列 start_time 框桶")
	assert.NotContains(t, sql, "AND m.end_time >= $2", "源筛选不应再按 end_time 框桶")
	assert.NotContains(t, sql, "AND m.end_time <  $3", "源筛选不应再按 end_time 框桶")

	// args 去掉了多余的 bucketStart 占位，仅 granularity + where 区间
	assert.Equal(t, []any{"hourly", w.Start, w.End}, args)
}

func Test_buildDeviceGroupSQL_DailyGroupTableNoID(t *testing.T) {
	w := WindowSpec{
		Granularity: metrics.GranularityDaily,
		Start:       time.Date(2026, 5, 22, 0, 0, 0, 0, time.UTC),
		End:         time.Date(2026, 5, 23, 0, 0, 0, 0, time.UTC),
	}
	sql, _ := buildDeviceGroupSQL("pm_metrics_daily", "pm_group_metrics_daily", w)

	// daily group 表无 id 列；insertCols 含 technology
	assert.Contains(t, sql, "INSERT INTO pm_group_metrics_daily (device_group_id, technology")
	assert.NotContains(t, sql, "gen_random_uuid()")
	// daily 冲突目标 PK 5 列（无 time，尾部含 technology），列序与迁移 000026 主键一致
	assert.Contains(t, sql, "ON CONFLICT (device_group_id, metric_path, granularity, end_time, technology)")
}

// AggregateDeviceGroup 通过 stub DB 验证 SQL + args 透传到 Exec。
// （真实 DB 验证留 Task 5 集成测试。）

func Test_AggregateDeviceGroup_PassesThroughToExec(t *testing.T) {
	db := &stubDB{execTag: pgconn.NewCommandTag("INSERT 0 2")}
	a := New(db, nil, nil)
	w := WindowSpec{
		Granularity: metrics.GranularityHourly,
		Start:       time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC),
		End:         time.Date(2026, 5, 22, 11, 0, 0, 0, time.UTC),
	}
	n, err := a.AggregateDeviceGroup(context.Background(), "pm_metrics_hourly", "pm_group_metrics_hourly", w)
	require.NoError(t, err)
	assert.Equal(t, 2, n)
	assert.Contains(t, db.execSQL, "INSERT INTO pm_group_metrics_hourly")
	assert.Contains(t, db.execSQL, "JOIN device_group_member_dim dgm")
	assert.Equal(t, []any{"hourly", w.Start, w.End}, db.execArgs)
}

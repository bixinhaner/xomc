package adhoc

import (
	"strings"
	"testing"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// T-0194：编辑任务 UPDATE SQL 守门口径单测（纯函数，不依赖 DB）。
//
// 覆盖：
//   1. 自建任务全字段进 SET（task_name/device_sns/metric_paths/granularities/cron_expr/object_ldns/window_*）。
//   2. 内置任务只 SET metric_paths + updated_at，其余结构性列不进 SET（保持原值）。
//   3. mode/technology/dimension/is_builtin/expire_days 任何情况都不进 SET。
//   4. WHERE 限定 task_subtype='adhoc_aggregation'。

func Test_buildUpdateSQL_Adhoc_AllFields(t *testing.T) {
	id := uuid.New()
	cronExpr := "5 * * * *"
	req := UpdateRequest{
		IsBuiltin:     false,
		Name:          "edited",
		CronExpr:      &cronExpr,
		DeviceSNs:     []string{"S1", "S2"},
		MetricPaths:   []string{"K1", "K2"},
		Granularities: []string{"daily"},
		ObjectLDNs:    []string{"LDN-A"},
		WindowStart:   time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC),
		WindowEnd:     time.Date(2026, 5, 22, 11, 0, 0, 0, time.UTC),
		PlannedEndAt:  ptrTime(time.Date(2026, 8, 22, 11, 0, 0, 0, time.UTC)),
	}
	sql, _, err := buildUpdateSQL(id, req)
	require.NoError(t, err)

	setClause := sql
	if idx := strings.Index(sql, "WHERE"); idx >= 0 {
		setClause = sql[:idx]
	}
	for _, col := range []string{"metric_paths", "task_name", "device_sns", "granularities", "cron_expr", "object_ldns", "window_start", "window_end", "planned_end_at", "updated_at"} {
		assert.Contains(t, setClause, col, "自建任务应更新 %s", col)
	}
	assert.Contains(t, setClause, "visibility", "自建任务应允许更新 visibility")
	// 结构性不可改字段绝不出现在 SET
	for _, col := range []string{"mode =", "technology =", "dimension =", "is_builtin =", "expire_days ="} {
		assert.NotContains(t, setClause, col, "不可改字段 %s 不应进 SET", col)
	}
	assert.Contains(t, sql, "task_subtype", "WHERE 应限定 task_subtype")
}

func ptrTime(t time.Time) *time.Time {
	return &t
}

func Test_buildUpdateSQL_Adhoc_ResetCursorWhenGranularityChanges(t *testing.T) {
	id := uuid.New()
	cronExpr := "5 * * * *"
	req := UpdateRequest{
		IsBuiltin:     false,
		Mode:          ModeContinuous,
		Name:          "edited",
		CronExpr:      &cronExpr,
		ResetCursor:   true,
		DeviceSNs:     []string{"S1"},
		MetricPaths:   []string{"K1"},
		Granularities: []string{"hourly"},
		LastFireAt:    time.Date(2026, 7, 7, 10, 0, 0, 0, time.UTC),
	}
	sql, _, err := buildUpdateSQL(id, req)
	require.NoError(t, err)

	setClause := sql
	if idx := strings.Index(sql, "WHERE"); idx >= 0 {
		setClause = sql[:idx]
	}
	assert.Contains(t, setClause, "cron_expr", "改粒度应同步更新调度表达式")
	assert.Contains(t, setClause, "last_fire_at", "改粒度应重置调度游标")
}

func Test_buildUpdateSQL_Adhoc_OneshotRequeuesTerminalTask(t *testing.T) {
	id := uuid.New()
	req := UpdateRequest{
		IsBuiltin:       false,
		Mode:            ModeOneshot,
		RequeueTerminal: true,
		Name:            "edited",
		DeviceSNs:       []string{"S1"},
		MetricPaths:     []string{"K1"},
		Granularities:   []string{"hourly"},
		WindowStart:     time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC),
		WindowEnd:       time.Date(2026, 5, 22, 11, 0, 0, 0, time.UTC),
	}
	sql, _, err := buildUpdateSQL(id, req)
	require.NoError(t, err)

	setClause := sql
	if idx := strings.Index(sql, "WHERE"); idx >= 0 {
		setClause = sql[:idx]
	}
	assert.Contains(t, setClause, "status = CASE WHEN status IN ('succeeded','failed') THEN 'pending' ELSE status END", "已完成/失败的自建 oneshot 编辑后应重新进入 pending")
	assert.Contains(t, setClause, "progress = CASE WHEN status IN ('succeeded','failed') THEN 0 ELSE progress END", "重新排队时应重置进度，pending/running 不应被改动")
}

func Test_buildUpdateSQL_Adhoc_OneshotDoesNotRequeueWhenExecutionInputsUnchanged(t *testing.T) {
	id := uuid.New()
	req := UpdateRequest{
		IsBuiltin:       false,
		Mode:            ModeOneshot,
		RequeueTerminal: false,
		Name:            "renamed",
		DeviceSNs:       []string{"S1"},
		MetricPaths:     []string{"K1"},
		Granularities:   []string{"hourly"},
		WindowStart:     time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC),
		WindowEnd:       time.Date(2026, 5, 22, 11, 0, 0, 0, time.UTC),
	}
	sql, _, err := buildUpdateSQL(id, req)
	require.NoError(t, err)

	setClause := sql
	if idx := strings.Index(sql, "WHERE"); idx >= 0 {
		setClause = sql[:idx]
	}
	assert.NotContains(t, setClause, "status = CASE", "仅改名称/可见性等非执行输入时不应重新排队")
	assert.NotContains(t, setClause, "progress = CASE", "不重新排队时不应重置进度")
}

func Test_buildUpdateSQL_Adhoc_ContinuousDoesNotRequeue(t *testing.T) {
	id := uuid.New()
	cronExpr := "5 * * * *"
	req := UpdateRequest{
		IsBuiltin:     false,
		Mode:          ModeContinuous,
		Name:          "edited",
		CronExpr:      &cronExpr,
		DeviceSNs:     []string{"S1"},
		MetricPaths:   []string{"K1"},
		Granularities: []string{"hourly"},
	}
	sql, _, err := buildUpdateSQL(id, req)
	require.NoError(t, err)

	setClause := sql
	if idx := strings.Index(sql, "WHERE"); idx >= 0 {
		setClause = sql[:idx]
	}
	assert.NotContains(t, setClause, "status = CASE", "continuous 编辑保持既有 scheduled/pending 调度语义")
	assert.NotContains(t, setClause, "progress = CASE", "continuous 编辑不走 oneshot 重新排队逻辑")
}

func Test_buildUpdateSQL_Builtin_OnlyMetricPaths(t *testing.T) {
	id := uuid.New()
	req := UpdateRequest{
		IsBuiltin:     true,
		Mode:          ModeOneshot,
		Name:          "ignored",          // 内置不应进 SET
		DeviceSNs:     []string{"X1"},     // 内置不应进 SET
		MetricPaths:   []string{"K9"},     // 唯一可改
		Granularities: []string{"hourly"}, // 内置不应进 SET
	}
	sql, _, err := buildUpdateSQL(id, req)
	require.NoError(t, err)

	assert.Contains(t, sql, "metric_paths", "内置应更新 metric_paths")
	assert.Contains(t, sql, "updated_at", "内置应更新 updated_at")
	// 结构性列不进 SET（device_sns/task_name/granularities/object_ldns/window_*）
	setClause := sql
	if idx := strings.Index(sql, "WHERE"); idx >= 0 {
		setClause = sql[:idx]
	}
	for _, col := range []string{"task_name", "device_sns", "granularities", "cron_expr", "object_ldns", "window_start", "window_end", "planned_end_at"} {
		assert.NotContains(t, setClause, col, "内置任务不应更新 %s", col)
	}
	assert.NotContains(t, setClause, "visibility", "内置任务不应更新 visibility")
	assert.NotContains(t, setClause, "status = CASE", "内置 oneshot 不应被重新排队")
	assert.NotContains(t, setClause, "progress = CASE", "内置 oneshot 不应重置进度")
}

func Test_plannedEndValue_CustomContinuous_DefaultsToCreateTimePlus30Days(t *testing.T) {
	value := plannedEndValue(CreateRequest{Mode: ModeContinuous})

	sqlizer, ok := value.(sq.Sqlizer)
	require.True(t, ok)
	sql, args, err := sqlizer.ToSql()
	require.NoError(t, err)
	assert.Empty(t, args)
	assert.Equal(t, "NOW() + INTERVAL '30 days'", sql)
}

func Test_plannedEndValue_BuiltinIgnoresExplicitPlannedEndAt(t *testing.T) {
	planned := time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC)

	assert.Nil(t, plannedEndValue(CreateRequest{
		Mode:         ModeContinuous,
		IsBuiltin:    true,
		PlannedEndAt: &planned,
	}))
}

func Test_plannedEndValue_NonContinuousDoesNotSetLifecycleEnd(t *testing.T) {
	planned := time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC)

	assert.Nil(t, plannedEndValue(CreateRequest{
		Mode:         ModeOneshot,
		PlannedEndAt: &planned,
	}))
}

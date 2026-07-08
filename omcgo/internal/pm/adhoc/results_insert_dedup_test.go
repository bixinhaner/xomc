package adhoc

import (
	"math"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// T-0194 Part A：InsertResults 同批去重 + ON CONFLICT 守门口径单测（纯函数，不依赖 DB）。

func ptrStr(s string) *string { return &s }

// 成功路径：业务键各不相同时全部保留，顺序不变。
func Test_dedupResultRows_AllDistinctKept(t *testing.T) {
	task := uuid.New()
	t0 := time.Date(2026, 6, 3, 10, 0, 0, 0, time.UTC)
	rows := []ResultRow{
		{TaskID: task, Granularity: "hourly", MetricPath: "K1", DeviceSN: "SN-A", Time: t0, MetricValue: 1},
		{TaskID: task, Granularity: "hourly", MetricPath: "K1", DeviceSN: "SN-B", Time: t0, MetricValue: 2},
		{TaskID: task, Granularity: "hourly", MetricPath: "K2", DeviceSN: "SN-A", Time: t0, MetricValue: 3},
	}
	out := dedupResultRows(rows)
	require.Len(t, out, 3)
	assert.Equal(t, float64(1), out[0].MetricValue)
	assert.Equal(t, float64(2), out[1].MetricValue)
	assert.Equal(t, float64(3), out[2].MetricValue)
}

// 边界路径：同业务键重复时去重保留最后一条（后者覆盖前者），位置不变、行数不增。
// 这是防 PG「command cannot affect row a second time」的关键。
func Test_dedupResultRows_DuplicateKeyLastWins(t *testing.T) {
	task := uuid.New()
	t0 := time.Date(2026, 6, 3, 10, 0, 0, 0, time.UTC)
	rows := []ResultRow{
		{TaskID: task, Granularity: "hourly", MetricPath: "K1", DeviceSN: "SN-A", Time: t0, MetricValue: 10},
		{TaskID: task, Granularity: "hourly", MetricPath: "K1", DeviceSN: "SN-B", Time: t0, MetricValue: 20},
		// 与第 0 条同业务键，值更新
		{TaskID: task, Granularity: "hourly", MetricPath: "K1", DeviceSN: "SN-A", Time: t0, MetricValue: 99},
	}
	out := dedupResultRows(rows)
	require.Len(t, out, 2)
	// 保首次出现位置，值取最后一条
	assert.Equal(t, "SN-A", out[0].DeviceSN)
	assert.Equal(t, float64(99), out[0].MetricValue)
	assert.Equal(t, "SN-B", out[1].DeviceSN)
}

// 空值列（device_oui/sn/product_id/object_ldn）兜空串后判同键：
// product 维度两行（device 列空、product_id 同值、object_ldn 同空）视为同键去重，
// 验证 COALESCE 兜底口径不让空值逃逸。
func Test_dedupResultRows_NullColumnsCoalesced(t *testing.T) {
	task := uuid.New()
	pid := uuid.New()
	t0 := time.Date(2026, 6, 3, 10, 0, 0, 0, time.UTC)
	rows := []ResultRow{
		{TaskID: task, Granularity: "daily", MetricPath: "C1", ProductID: pid, Time: t0, MetricValue: 5},
		// device_oui/sn 空、object_ldn nil、product_id 同 → 同业务键
		{TaskID: task, Granularity: "daily", MetricPath: "C1", ProductID: pid, Time: t0, MetricValue: 8},
	}
	out := dedupResultRows(rows)
	require.Len(t, out, 1)
	assert.Equal(t, float64(8), out[0].MetricValue)
}

// object_ldn 不同 → 不同键，都保留（product/band/aggregate_group 多对象不被误并）。
func Test_dedupResultRows_DifferentObjectLDNKept(t *testing.T) {
	task := uuid.New()
	t0 := time.Date(2026, 6, 3, 10, 0, 0, 0, time.UTC)
	rows := []ResultRow{
		{TaskID: task, Granularity: "daily", MetricPath: "C1", ObjectLDN: ptrStr("Band=1"), Time: t0, MetricValue: 5},
		{TaskID: task, Granularity: "daily", MetricPath: "C1", ObjectLDN: ptrStr("Band=3"), Time: t0, MetricValue: 8},
	}
	out := dedupResultRows(rows)
	require.Len(t, out, 2)
}

// Time 为零时回退 EndTime 参与键（与落库 t 的取值口径一致）。
func Test_dedupResultRows_ZeroTimeFallsBackToEndTime(t *testing.T) {
	task := uuid.New()
	end := time.Date(2026, 6, 3, 11, 0, 0, 0, time.UTC)
	rows := []ResultRow{
		{TaskID: task, Granularity: "hourly", MetricPath: "K1", DeviceSN: "SN-A", EndTime: end, MetricValue: 1},
		{TaskID: task, Granularity: "hourly", MetricPath: "K1", DeviceSN: "SN-A", EndTime: end, MetricValue: 2},
	}
	out := dedupResultRows(rows)
	require.Len(t, out, 1)
	assert.Equal(t, float64(2), out[0].MetricValue)
}

// nil / 单元素切片直接原样返回（无 panic）。
func Test_dedupResultRows_NilAndSingle(t *testing.T) {
	assert.Nil(t, dedupResultRows(nil))
	one := []ResultRow{{TaskID: uuid.New(), MetricPath: "K1"}}
	assert.Len(t, dedupResultRows(one), 1)
}

// ON CONFLICT 冲突目标表达式必须与 migrations/000018 的唯一索引 8 列逐字一致，
// 且 DO UPDATE 用 EXCLUDED 覆盖值类列、不更新冲突键 time。
func Test_onConflictResultsBusiness_MatchesIndexExpression(t *testing.T) {
	c := onConflictResultsBusiness
	// 8 列冲突目标，与索引表达式逐字一致
	assert.Contains(t, c, `ON CONFLICT (task_id, granularity, metric_path, COALESCE(device_oui, ''), COALESCE(device_sn, ''), COALESCE(product_id::text, ''), COALESCE(object_ldn, ''), "time")`)
	// DO UPDATE 覆盖值类列
	assert.Contains(t, c, "DO UPDATE SET")
	for _, col := range []string{"metric_value", "metric_type", "statis_type", "start_time", "end_time", "extra"} {
		assert.Contains(t, c, "EXCLUDED."+col)
	}
	// 冲突键 time 不出现在 SET 中
	assert.NotContains(t, c, "SET ... time =")
	assert.False(t, strings.Contains(c, `"time" = EXCLUDED`))
}

func Test_buildInsertResultsSQL_NullMetricValueBindsNull(t *testing.T) {
	task := uuid.New()
	t0 := time.Date(2026, 6, 3, 10, 0, 0, 0, time.UTC)
	rows := []ResultRow{{
		TaskID: task, Granularity: "hourly", MetricPath: "C-NULL",
		DeviceSN: "SN-A", MetricType: "counter", MetricValue: math.NaN(),
		Time: t0, StartTime: t0, EndTime: t0,
	}}

	_, args, err := buildInsertResultsSQL(rows)

	require.NoError(t, err)
	require.Len(t, args, len(resultInsertCols))
	assert.Nil(t, args[6], "metric_value 的 NaN 应绑定为 SQL NULL，而不是写 PostgreSQL NaN")
}

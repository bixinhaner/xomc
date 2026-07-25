package dashboard

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 成功路径：时间窗内按 metric_path 各取最新一行（DISTINCT ON），LIMIT 给到 latestKPISummaryLimit。
func TestBuildLatestKPIPerNameQuery_Basic(t *testing.T) {
	start := time.Date(2026, 6, 24, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 6, 25, 0, 0, 0, 0, time.UTC)

	q, args, err := buildLatestKPIPerNameQuery(start, end)
	require.NoError(t, err)

	// 只读物理 KPI 值，不展开缺失指标集合。
	assert.Contains(t, q, "FROM pm_metric_values")
	assert.Contains(t, q, "JOIN pm_measurement_anchors")
	// DISTINCT ON (metric_path)：每指标编号一行（HD01 修复点）。
	assert.Contains(t, q, "DISTINCT ON (d.metric_path)")
	// 只读 KPI 类，counter 不进首页卡。
	assert.Contains(t, q, "metric_type = $1")
	// 时间窗。
	assert.Contains(t, q, `a."time" >= $2`)
	assert.Contains(t, q, `a."time" <= $3`)
	// ORDER BY 首键必须是 metric_path（PG DISTINCT ON 硬约束），再 time DESC 取最新。
	assert.Contains(t, q, `ORDER BY d.metric_path, a."time" DESC`)
	// 按"不同指标数"上界（非"行数"上界），防 OOM。
	assert.Contains(t, q, "LIMIT 500")

	// args 顺序：[metric_type, start, end]
	require.Len(t, args, 3)
	assert.Equal(t, "kpi", args[0])
	assert.Equal(t, start, args[1])
	assert.Equal(t, end, args[2])
}

// 不限 granularity / 不限 device：首页卡是全网视图，同指标 15min/hourly 同时存在时取最新。
func TestBuildLatestKPIPerNameQuery_NoGranularityOrDeviceFilter(t *testing.T) {
	q, _, err := buildLatestKPIPerNameQuery(time.Time{}, time.Time{})
	require.NoError(t, err)
	assert.NotContains(t, q, "granularity")
	assert.NotContains(t, q, "device_sn")
	assert.NotContains(t, q, "device_oui")
	assert.NotContains(t, q, "object_ldn")
}

// DISTINCT ON 关键不变量：ORDER BY 首键必须 = DISTINCT ON 列，否则 PG 报错。
func TestBuildLatestKPIPerNameQuery_DistinctOnOrderingInvariant(t *testing.T) {
	q, _, err := buildLatestKPIPerNameQuery(time.Time{}, time.Time{})
	require.NoError(t, err)
	distinctIdx := strings.Index(q, "DISTINCT ON (d.metric_path)")
	orderIdx := strings.Index(q, "ORDER BY d.metric_path")
	require.Positive(t, distinctIdx)
	require.Positive(t, orderIdx)
	require.Greater(t, orderIdx, distinctIdx, "ORDER BY must follow SELECT DISTINCT ON")
}

// 上界常量明确（500 远大于系统总指标数，又给硬上界防 OOM）。
func TestLatestKPISummaryLimit_Fixed(t *testing.T) {
	assert.Equal(t, 500, latestKPISummaryLimit)
}

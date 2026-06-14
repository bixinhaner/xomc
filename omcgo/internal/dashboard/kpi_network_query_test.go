package dashboard

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 成功路径：按指标编号读全网预聚合结果表，SQL 含三条 network 任务过滤 + 编号 ANY + 小时粒度 + 时间窗。
func TestBuildNetworkKPISeriesQuery_Basic(t *testing.T) {
	start := time.Date(2026, 6, 13, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 6, 13, 12, 0, 0, 0, time.UTC)

	q, args, err := buildNetworkKPISeriesQuery([]string{"K900010015", "C000060216"}, start, end)
	require.NoError(t, err)

	// 读预聚合结果表，而非原始 pm_metrics。
	assert.Contains(t, q, "FROM pm_adhoc_aggregation_results")
	assert.NotContains(t, q, "pm_metrics")
	// 三条 network 任务过滤（覆盖全制式）。
	assert.Contains(t, q, "task_id = ANY($1)")
	// 编号过滤。
	assert.Contains(t, q, "metric_path = ANY($2)")
	// 小时粒度（内置 network 任务按小时聚合）。
	assert.Contains(t, q, "granularity = $3")
	// 时间窗。
	assert.Contains(t, q, "time >= $4")
	assert.Contains(t, q, "time <= $5")
	// 排序便于回填时序。
	assert.Contains(t, q, "ORDER BY metric_path, time ASC")

	// args 顺序：[networkTaskIDs, codes, granularity, start, end]
	require.Len(t, args, 5)
	assert.Equal(t, networkAggregationTaskIDs, args[0])
	assert.Equal(t, []string{"K900010015", "C000060216"}, args[1])
	assert.Equal(t, networkResultGranularity, args[2])
	assert.Equal(t, start, args[3])
	assert.Equal(t, end, args[4])
}

// 不限 metric_type：counter 与 KPI 同表读（一面板一时刻一条线，无量纲互压）。
func TestBuildNetworkKPISeriesQuery_NoMetricTypeFilter(t *testing.T) {
	q, _, err := buildNetworkKPISeriesQuery([]string{"C999999999"}, time.Time{}, time.Time{})
	require.NoError(t, err)
	assert.NotContains(t, q, "metric_type")
}

// 三条内置 network 任务 ID 固定且齐全（LTE/NR/GSM 各一），首页全网线只读这三条。
func TestNetworkAggregationTaskIDs_Fixed(t *testing.T) {
	assert.Equal(t, []string{
		"0184dddd-0001-4000-8000-000000000001",
		"0184dddd-0001-4000-8000-000000000002",
		"0184dddd-0001-4000-8000-000000000003",
	}, networkAggregationTaskIDs)
	assert.Equal(t, "hourly", networkResultGranularity)
}

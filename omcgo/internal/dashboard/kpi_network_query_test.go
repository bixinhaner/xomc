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

// 成功路径（回退口径，issue #359）：预聚合表缺数据时直读原始明细 pm_metrics 现场汇成全网线。
// SQL 含 metric_path/time GROUP BY + statis_type 路由的 CASE 算子 + 15min 粒度 + 时间窗。
func TestBuildRawNetworkKPISeriesQuery_Basic(t *testing.T) {
	start := time.Date(2026, 6, 13, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 6, 13, 12, 0, 0, 0, time.UTC)

	q, args, err := buildRawNetworkKPISeriesQuery([]string{"K900010015", "C000060216"}, start, end)
	require.NoError(t, err)

	// 回退读原始明细 pm_metrics，而非预聚合表。
	assert.Contains(t, q, "FROM pm_metrics")
	assert.NotContains(t, q, "pm_adhoc_aggregation_results")
	// 现场汇成全网一条线：按 metric_path + time GROUP BY，不带任何设备/小区实体键。
	assert.Contains(t, q, "GROUP BY metric_path, time")
	assert.NotContains(t, q, "device_sn")
	assert.NotContains(t, q, "object_ldn")
	// 算子按 statis_type 路由（sum/avg/max/min；未知按 sum）。
	assert.Contains(t, q, "CASE MIN(statis_type)")
	assert.Contains(t, q, "WHEN 'sum' THEN SUM(metric_value)")
	assert.Contains(t, q, "WHEN 'avg' THEN AVG(metric_value)")
	// 只读 15min 原始明细。
	assert.Contains(t, q, "granularity = ")
	// 编号过滤 + 时间窗。
	assert.Contains(t, q, "metric_path = ANY(")
	assert.Contains(t, q, "time >= ")
	assert.Contains(t, q, "time <= ")
	// 排序便于回填时序。
	assert.Contains(t, q, "ORDER BY metric_path, time ASC")

	// args 顺序：[codes, granularity, start, end]
	require.Len(t, args, 4)
	assert.Equal(t, []string{"K900010015", "C000060216"}, args[0])
	assert.Equal(t, rawFallbackGranularity, args[1])
	assert.Equal(t, start, args[2])
	assert.Equal(t, end, args[3])
}

// 回退口径只读 15min 原始明细（与性能仪表板默认模板 15min 容错对齐）。
func TestBuildRawNetworkKPISeriesQuery_Granularity15min(t *testing.T) {
	assert.Equal(t, "15min", rawFallbackGranularity)

	_, args, err := buildRawNetworkKPISeriesQuery([]string{"C999999999"}, time.Time{}, time.Time{})
	require.NoError(t, err)
	require.Len(t, args, 4)
	assert.Equal(t, "15min", args[1])
}

// 尾部补点查询与整体回退查询使用相同 SQL 模板，仅时间窗不同。
// （trailingStart = min(latestByCode + 1h)，endTime = now；由调用方 fetchNetworkKCodeSeries 计算）
func TestBuildRawNetworkKPISeriesQuery_TrailingEdgeWindow(t *testing.T) {
	// 模拟：最新小时桶为 16:00，尾部补点时窗为 17:00–17:30。
	trailingStart := time.Date(2026, 6, 13, 17, 0, 0, 0, time.UTC)
	now := time.Date(2026, 6, 13, 17, 30, 0, 0, time.UTC)

	q, args, err := buildRawNetworkKPISeriesQuery([]string{"K900010015"}, trailingStart, now)
	require.NoError(t, err)

	// 与整体回退查询相同模板（读 pm_metrics 15min，按 metric_path+time GROUP BY）。
	assert.Contains(t, q, "FROM pm_metrics")
	assert.Contains(t, q, "GROUP BY metric_path, time")
	assert.Contains(t, q, "granularity = ")

	// 时间窗参数为 trailingStart ~ now。
	require.Len(t, args, 4)
	assert.Equal(t, trailingStart, args[2])
	assert.Equal(t, now, args[3])
}

// 跨制式混合场景：不同 code 有不同的最新小时桶时间，尾部查询的 trailingStart 取各 code
// 「下一桶起点」的最小值；合并时按 per-code 过滤，避免与现有小时数据重叠。
// 本测验证 SQL 模板在更早的 trailingStart 下仍能正确构建（逻辑本身不变）。
func TestBuildRawNetworkKPISeriesQuery_CrossTechEarlierTrailingStart(t *testing.T) {
	// LTE 最新桶 16:00 → 候选 17:00
	// GSM 最新桶 14:00 → 候选 15:00 → 全局 trailingStart 取 15:00
	trailingStart := time.Date(2026, 6, 13, 15, 0, 0, 0, time.UTC)
	now := time.Date(2026, 6, 13, 17, 30, 0, 0, time.UTC)

	q, args, err := buildRawNetworkKPISeriesQuery([]string{"K900010015", "KGSM0101"}, trailingStart, now)
	require.NoError(t, err)

	// SQL 模板不变，时间窗覆盖两个 code 的缺口。
	assert.Contains(t, q, "FROM pm_metrics")
	assert.Contains(t, q, "GROUP BY metric_path, time")
	require.Len(t, args, 4)
	assert.Equal(t, trailingStart, args[2])
	assert.Equal(t, now, args[3])
}

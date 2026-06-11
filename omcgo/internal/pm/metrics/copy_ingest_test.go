package metrics

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// dedupeByNaturalKey 是 copy 模式（plain COPY 无 ON CONFLICT）的文件内幂等闸：撞 uq_pm_metrics_natural
// 的重复行必须先在内存折叠成一行（last-wins），否则整批 COPY 失败。以下用例锁住其语义。

func Test_dedupeByNaturalKey_CollapsesSameKey_LastWins(t *testing.T) {
	// 同一自然键（同 path + 同 cell + 同窗）出现两次：多个上报名经白名单改写命中同一 IndicatorID，
	// 或厂商重复上报同 measType。必须折叠成一行，取最后写入的值（复刻 ON CONFLICT DO UPDATE）。
	first := metricWithLDN("C000060216", "cell-1", true)
	first.MetricValue = 100
	second := metricWithLDN("C000060216", "cell-1", true)
	second.MetricValue = 250 // 后写

	out := dedupeByNaturalKey([]PMMetric{first, second})

	require.Len(t, out, 1, "同自然键必须折叠成一行")
	assert.EqualValues(t, 250, out[0].MetricValue, "last-wins：保留最后写入的值")
}

func Test_dedupeByNaturalKey_DistinctCells_NoCollapse(t *testing.T) {
	// 不同 cell（object_ldn 不同）= 不同自然键，必须各自保留——否则会丢小区数据。
	a := metricWithLDN("C000060216", "cell-1", true)
	b := metricWithLDN("C000060216", "cell-2", true)
	c := metricWithLDN("C000060216", "", false) // object_ldn nil → '' 是独立桶

	out := dedupeByNaturalKey([]PMMetric{a, b, c})

	require.Len(t, out, 3, "不同 object_ldn 是不同自然键，不得折叠")
}

func Test_dedupeByNaturalKey_DistinctPaths_NoCollapse(t *testing.T) {
	// 不同 metric_path（不同 counter / counter vs KPI 编号）= 不同自然键，全部保留。
	out := dedupeByNaturalKey([]PMMetric{
		metricWithLDN("C000060216", "cell-1", true),
		metricWithLDN("C000060217", "cell-1", true),
		metricWithLDN("K000000001", "cell-1", true), // KPI 编号
	})
	require.Len(t, out, 3, "不同 metric_path 不得折叠")
}

func Test_dedupeByNaturalKey_PreservesFirstSeenOrder(t *testing.T) {
	// 折叠仅替换值、保持首次出现的相对顺序，便于排查（不打乱行序）。
	first := metricWithLDN("C1", "cell-1", true)
	mid := metricWithLDN("C2", "cell-1", true)
	dupOfFirst := metricWithLDN("C1", "cell-1", true)
	dupOfFirst.MetricValue = 999

	out := dedupeByNaturalKey([]PMMetric{first, mid, dupOfFirst})

	require.Len(t, out, 2)
	assert.Equal(t, "C1", out[0].MetricPath, "C1 保持在首位（首次出现位置）")
	assert.EqualValues(t, 999, out[0].MetricValue, "C1 取最后值")
	assert.Equal(t, "C2", out[1].MetricPath)
}

func Test_dedupeByNaturalKey_Empty(t *testing.T) {
	assert.Empty(t, dedupeByNaturalKey(nil))
	assert.Empty(t, dedupeByNaturalKey([]PMMetric{}))
}

package metrics

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/core/model"
)

func metricWithLDN(path, ldn string, setLDN bool) PMMetric {
	t := time.Date(2026, 5, 25, 17, 0, 0, 0, time.UTC)
	m := PMMetric{
		DeviceOUI: "48BF74", DeviceSN: "1202000240194DP0015",
		MetricPath: path, MetricType: MetricTypeCounter, MetricValue: 123,
		Granularity: Granularity15Min, Time: t,
		StartTime: t.Add(-15 * time.Minute), EndTime: t, IngestTime: t,
	}
	if setLDN {
		m.ObjectLDN = &ldn
	}
	return m
}

// MetricFromKPIValue 必须给 KPI 行补出完整 15min 窗口（start = end - 15min），否则
// pm_metrics.start_time == end_time，前端悬浮框「开始/结束」显示同一时刻（#199 / #208）。

func Test_MetricFromKPIValue_StartIsEndMinus15Min(t *testing.T) {
	end := time.Date(2026, 6, 13, 10, 15, 0, 0, time.UTC)
	v := model.KPIValue{
		Time:        end,
		OUI:         "0019C0",
		DeviceSN:    "SN-1",
		CellID:      "cell-1",
		IndicatorID: "K900010015",
		KPIValue:    42.5,
	}

	m := MetricFromKPIValue(v)

	assert.Equal(t, MetricTypeKPI, m.MetricType)
	assert.Equal(t, end, m.EndTime, "EndTime 仍为窗口止点 v.Time")
	assert.Equal(t, end.Add(-15*time.Minute), m.StartTime, "StartTime 推导为 end - 15min")
	// #479 改动二：time 统一为桶起点（= start_time），不再写桶结束时刻。
	assert.Equal(t, end.Add(-15*time.Minute), m.Time, "Time 必须为桶起点（= start_time）")
	assert.Equal(t, m.StartTime, m.Time, "不变量：time == start_time")
	assert.NotEqual(t, m.StartTime, m.EndTime, "start 必须严格早于 end，杜绝起止相同")
	require.NotNil(t, m.ObjectLDN)
	assert.Equal(t, "cell-1", *m.ObjectLDN)
}

func Test_MetricFromKPIValue_NoCellID_LdnNil(t *testing.T) {
	// 失败/边界路径：无小区（CellID 空）时 ObjectLDN 必须为 nil，窗口推导仍成立。
	end := time.Date(2026, 6, 13, 10, 15, 0, 0, time.UTC)
	m := MetricFromKPIValue(model.KPIValue{Time: end, IndicatorID: "K1"})

	assert.Nil(t, m.ObjectLDN, "无 CellID 时 object_ldn 为 nil")
	assert.Equal(t, end.Add(-15*time.Minute), m.StartTime)
	assert.Equal(t, end, m.EndTime)
	assert.Equal(t, m.StartTime, m.Time, "不变量：time == start_time")
}

func Test_MetricFromKPIValue_CarriesStatisType(t *testing.T) {
	m := MetricFromKPIValue(model.KPIValue{
		Time:        time.Date(2026, 7, 20, 10, 15, 0, 0, time.UTC),
		IndicatorID: "KAVG001",
		KPIValue:    12.3,
		StatisType:  "avg",
	})

	require.NotNil(t, m.StatisType)
	assert.Equal(t, StatisAvg, *m.StatisType)
}

func Test_MetricFromCounter_StartWindow_NoRegression(t *testing.T) {
	// counter 行起止本就正确（start = end - granularity），确保未被 KPI 改动波及。
	end := time.Date(2026, 6, 13, 10, 15, 0, 0, time.UTC)

	withGran := MetricFromCounter(model.PMCounter{
		Time: end, CounterName: "C1", CounterValue: 7, Granularity: 15,
	})
	assert.Equal(t, MetricTypeCounter, withGran.MetricType)
	assert.Equal(t, end.Add(-15*time.Minute), withGran.StartTime, "granularity>0 时 start = end - granularity")
	assert.Equal(t, end, withGran.EndTime)
	// #479 改动二：time 统一为桶起点（= start_time）。
	assert.Equal(t, withGran.StartTime, withGran.Time, "不变量：time == start_time（granularity>0）")

	// granularity=0（未知粒度）时退回 start == end，保持既有行为。
	noGran := MetricFromCounter(model.PMCounter{Time: end, CounterName: "C1", CounterValue: 7})
	assert.Equal(t, end, noGran.StartTime, "granularity=0 时 start 退回 end（不推导窗口）")
	assert.Equal(t, end, noGran.EndTime)
	assert.Equal(t, noGran.StartTime, noGran.Time, "不变量：time == start_time（granularity=0）")
}

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

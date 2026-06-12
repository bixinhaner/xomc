package counter

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/pm/metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeMetricsRepo 让 BatchInsert 返回预置错误，验证 wrapper 错误透传（issue #14）。
type fakeMetricsRepo struct {
	batchErr error
	gotCount int
}

func (f *fakeMetricsRepo) Insert(ctx context.Context, m metrics.PMMetric) error { return nil }
func (f *fakeMetricsRepo) BatchInsert(ctx context.Context, ms []metrics.PMMetric) error {
	f.gotCount = len(ms)
	return f.batchErr
}
func (f *fakeMetricsRepo) Query(ctx context.Context, q metrics.QueryRequest) ([]metrics.PMMetric, error) {
	return nil, nil
}
func (f *fakeMetricsRepo) Count(ctx context.Context, q metrics.QueryRequest) (int64, error) {
	return 0, nil
}

func ldn(s string) *string { return &s }

// groupCountersByCell 等价性：单次查询按 cell 分桶 == 逐 cell QueryForKPI 过滤求和。
func Test_groupCountersByCell_PerCellSums(t *testing.T) {
	ms := []metrics.PMMetric{
		{ObjectLDN: ldn("cell-1"), MetricPath: "A", MetricValue: 10},
		{ObjectLDN: ldn("cell-1"), MetricPath: "A", MetricValue: 5}, // 同 cell 同 counter 累加
		{ObjectLDN: ldn("cell-1"), MetricPath: "B", MetricValue: 7},
		{ObjectLDN: ldn("cell-2"), MetricPath: "A", MetricValue: 100},
		{ObjectLDN: ldn("cell-3"), MetricPath: "A", MetricValue: 999}, // cell-3 未请求 → 丢弃
	}
	out := groupCountersByCell(ms, []string{"cell-1", "cell-2"}, 900)

	assert.Equal(t, 15.0, out["cell-1"]["A"])
	assert.Equal(t, 7.0, out["cell-1"]["B"])
	assert.Equal(t, 100.0, out["cell-2"]["A"])
	assert.Equal(t, 900.0, out["cell-1"]["period_seconds"], "period 注入每个请求 cell")
	assert.Equal(t, 900.0, out["cell-2"]["period_seconds"])
	_, has3 := out["cell-3"]
	assert.False(t, has3, "未请求的 cell 不应出现")
}

// 请求的 cell 没有任何 counter 行时，仍得到只含 period_seconds 的桶（与旧空结果一致）。
func Test_groupCountersByCell_EmptyCellKeepsPeriod(t *testing.T) {
	out := groupCountersByCell(nil, []string{"cell-x"}, 900)
	require.NotNil(t, out["cell-x"])
	assert.Equal(t, 900.0, out["cell-x"]["period_seconds"])
	_, hasA := out["cell-x"]["A"]
	assert.False(t, hasA)
}

// period<=0 时不注入 period_seconds（复刻旧 duration>0 守卫）。
func Test_groupCountersByCell_NoPeriodWhenZero(t *testing.T) {
	out := groupCountersByCell([]metrics.PMMetric{
		{ObjectLDN: ldn("cell-1"), MetricPath: "A", MetricValue: 3},
	}, []string{"cell-1"}, 0)
	assert.Equal(t, 3.0, out["cell-1"]["A"])
	_, hasPeriod := out["cell-1"]["period_seconds"]
	assert.False(t, hasPeriod)
}

// cellID=="" 的历史语义：跨全部行求和（含真实 cell 行 + 无 cell 行），且 ldn=="" 的行不被重复累加。
func Test_groupCountersByCell_EmptyCellIDSumsAll(t *testing.T) {
	ms := []metrics.PMMetric{
		{ObjectLDN: ldn("cell-1"), MetricPath: "A", MetricValue: 10},
		{ObjectLDN: ldn("cell-2"), MetricPath: "A", MetricValue: 20},
		{ObjectLDN: nil, MetricPath: "A", MetricValue: 3}, // 无 cell 行
	}
	// 同时请求真实 cell 与 ""：真实 cell 只拿自己的行，"" 拿全部行之和。
	out := groupCountersByCell(ms, []string{"cell-1", ""}, 900)
	assert.Equal(t, 10.0, out["cell-1"]["A"], "真实 cell 只累加自身行")
	assert.Equal(t, 33.0, out[""]["A"], "\"\" 桶 = 全部行求和（10+20+3），无重复累加")
}

// 等价性：GroupParsedCountersByCell(内存 PMCounter) 与 groupCountersByCell(DB PMMetric)
// 对同一份数据产出完全相同的分桶——保证免回读快路径与回读路径语义一致。
func Test_GroupParsedCountersByCell_EqualsDBGrouping(t *testing.T) {
	// 同一份逻辑数据的两种表示：内存 PMCounter vs DB PMMetric。
	counters := []model.PMCounter{
		{CellID: "cell-1", CounterName: "A", CounterValue: 10},
		{CellID: "cell-1", CounterName: "A", CounterValue: 5}, // 同 (cell,name) 求和
		{CellID: "cell-1", CounterName: "B", CounterValue: 7},
		{CellID: "cell-2", CounterName: "A", CounterValue: 100},
		{CellID: "", CounterName: "A", CounterValue: 3}, // 无 cell 行
	}
	ms := make([]metrics.PMMetric, len(counters))
	for i, c := range counters {
		ldn := c.CellID
		ms[i] = metrics.PMMetric{ObjectLDN: &ldn, MetricPath: c.CounterName, MetricValue: c.CounterValue}
	}
	cellIDs := []string{"cell-1", "cell-2", ""}

	fromMem := GroupParsedCountersByCell(counters, cellIDs, 900)
	fromDB := groupCountersByCell(ms, cellIDs, 900)
	assert.Equal(t, fromDB, fromMem, "内存分桶须与 DB 分桶逐桶相等")

	// 抽查关键值，避免两者同错。
	assert.Equal(t, 15.0, fromMem["cell-1"]["A"])
	assert.Equal(t, 7.0, fromMem["cell-1"]["B"])
	assert.Equal(t, 100.0, fromMem["cell-2"]["A"])
	assert.Equal(t, 118.0, fromMem[""]["A"], "\"\" 桶=全部 A 行求和 10+5+100+3")
	assert.Equal(t, 900.0, fromMem["cell-1"]["period_seconds"])
}

// --- #208 同窗口多文件 counter 去重（KPI 读侧 SUM 不翻倍）---

// 构造同一个 device 维度 KPI 行的两份来源：同设备同 15min 窗口、同 metric_path/cell，但来自
// 两个不同文件（不同 ingest_time、不同行 id）。删 uq_pm_metrics_natural（#256）后两份都落库，
// KPI 读侧若直接 SUM 会翻倍（~2x）。dedupeCountersByNaturalKey 须把它们折叠成 ingest_time 最新
// 的一条，使后续 SUM 等于「单文件值」，不双计。
func Test_dedupeCountersByNaturalKey_SameWindowTwoFiles_NoDouble(t *testing.T) {
	win := time.Date(2026, 6, 13, 10, 15, 0, 0, time.UTC)
	older := win.Add(30 * time.Second)  // 文件 1 入库时刻
	newer := win.Add(120 * time.Second) // 文件 2（补传换名）入库时刻，更晚
	mk := func(val float64, ingest time.Time) metrics.PMMetric {
		return metrics.PMMetric{
			ID:          uuid.New(), // 不同文件 = 不同行 id
			DeviceOUI:   "00A0C9",
			DeviceSN:    "SN-1",
			MetricPath:  "A",
			Granularity: metrics.Granularity15Min,
			Time:        win,
			EndTime:     win,
			IngestTime:  ingest,
			ObjectLDN:   ldn("cell-1"),
			MetricValue: val,
		}
	}
	// 文件 1 报 A=100，文件 2（补传，值修正/相同）报 A=100。两行同自然键。
	ms := []metrics.PMMetric{mk(100, older), mk(100, newer)}

	deduped := dedupeCountersByNaturalKey(ms)
	require.Len(t, deduped, 1, "同自然键两份须折叠成一条")
	assert.Equal(t, 100.0, deduped[0].MetricValue, "保留值（last-wins），不相加")
	assert.Equal(t, newer, deduped[0].IngestTime, "保留 ingest_time 最新一条（最后入库文件）")

	// 模拟 QueryForKPI 的 SUM 行为（dedup 后逐行累加），断言不翻倍。
	var sum float64
	for _, m := range deduped {
		sum += m.MetricValue
	}
	assert.Equal(t, 100.0, sum, "KPI 读侧 SUM 等于单文件值，不是 200（翻倍）")

	// 反证：不去重直接 SUM 会翻倍——锁住「去重确实在防翻倍」。
	var rawSum float64
	for _, m := range ms {
		rawSum += m.MetricValue
	}
	assert.Equal(t, 200.0, rawSum, "未去重时直接 SUM 翻倍（回归基线）")
}

// last-wins 取值：两份不同值时保留 ingest_time 最新一条的值（不是相加，也不是保留旧值）。
func Test_dedupeCountersByNaturalKey_LastWinsByIngestTime(t *testing.T) {
	win := time.Date(2026, 6, 13, 10, 15, 0, 0, time.UTC)
	mk := func(val float64, ingest time.Time) metrics.PMMetric {
		return metrics.PMMetric{
			DeviceOUI: "00A0C9", DeviceSN: "SN-1", MetricPath: "A",
			Granularity: metrics.Granularity15Min, Time: win, EndTime: win,
			IngestTime: ingest, ObjectLDN: ldn("cell-1"), MetricValue: val,
		}
	}
	t0 := win.Add(10 * time.Second)
	t1 := win.Add(20 * time.Second)
	// 乱序：先给更晚 ingest 的修正值，再给更早的旧值，验证与切片顺序无关、只看 ingest_time。
	deduped := dedupeCountersByNaturalKey([]metrics.PMMetric{mk(42, t1), mk(7, t0)})
	require.Len(t, deduped, 1)
	assert.Equal(t, 42.0, deduped[0].MetricValue, "保留 ingest_time 最新（t1=42），与出现顺序无关")
}

// 不同自然键不折叠：同设备同窗但不同 cell / 不同 metric_path 是合法独立行，必须各自保留。
func Test_dedupeCountersByNaturalKey_DistinctKeysKept(t *testing.T) {
	win := time.Date(2026, 6, 13, 10, 15, 0, 0, time.UTC)
	base := metrics.PMMetric{
		DeviceOUI: "00A0C9", DeviceSN: "SN-1", Granularity: metrics.Granularity15Min,
		Time: win, EndTime: win, IngestTime: win,
	}
	mkA := base
	mkA.MetricPath, mkA.ObjectLDN, mkA.MetricValue = "A", ldn("cell-1"), 10
	mkB := base
	mkB.MetricPath, mkB.ObjectLDN, mkB.MetricValue = "B", ldn("cell-1"), 20 // 不同 metric_path
	mkC := base
	mkC.MetricPath, mkC.ObjectLDN, mkC.MetricValue = "A", ldn("cell-2"), 30 // 不同 cell
	deduped := dedupeCountersByNaturalKey([]metrics.PMMetric{mkA, mkB, mkC})
	assert.Len(t, deduped, 3, "三条不同自然键全保留，不误折叠")
}

// 端到端等价：dedup → groupCountersByCell（QueryForKPICells 的 SUM 路径）对「两文件重复行」产出
// 与「单文件」完全一致的分桶，证明翻倍在分桶 SUM 这一层也被消除。
func Test_dedupeThenGroup_TwoFilesEqualsSingle(t *testing.T) {
	win := time.Date(2026, 6, 13, 10, 15, 0, 0, time.UTC)
	row := func(val float64, ingest time.Time) metrics.PMMetric {
		return metrics.PMMetric{
			DeviceOUI: "00A0C9", DeviceSN: "SN-1", MetricPath: "A",
			Granularity: metrics.Granularity15Min, Time: win, EndTime: win,
			IngestTime: ingest, ObjectLDN: ldn("cell-1"), MetricValue: val,
		}
	}
	single := []metrics.PMMetric{row(100, win)}
	twoFiles := []metrics.PMMetric{row(100, win.Add(10 * time.Second)), row(100, win.Add(60 * time.Second))}

	gotSingle := groupCountersByCell(dedupeCountersByNaturalKey(single), []string{"cell-1"}, 900)
	gotTwo := groupCountersByCell(dedupeCountersByNaturalKey(twoFiles), []string{"cell-1"}, 900)

	assert.Equal(t, 100.0, gotSingle["cell-1"]["A"])
	assert.Equal(t, gotSingle, gotTwo, "去重后两文件与单文件分桶结果须完全一致（KPI 不翻倍）")
	assert.Equal(t, 100.0, gotTwo["cell-1"]["A"], "两文件重复行求和仍为单文件值 100，不是 200")
}

// CounterFilter 与 ListRequest 集成
func Test_CounterFilter_WithListRequest(t *testing.T) {
	filter := CounterFilter{
		ListRequest: model.ListRequest{
			Page:     2,
			PageSize: 10,
			SortBy:   "time",
			SortDir:  "asc",
		},
	}

	assert.Equal(t, 10, filter.Offset(), "page 2, size 10 should offset 10")
	assert.Equal(t, 10, filter.Limit())
}

func Test_CounterFilter_DefaultListRequest(t *testing.T) {
	filter := CounterFilter{
		ListRequest: model.DefaultListRequest(),
	}

	assert.Equal(t, 0, filter.Offset(), "page 1 should offset 0")
	assert.Equal(t, 20, filter.Limit())
}

// AggregatedCounter struct 字段稳定性
func Test_AggregatedCounter_Struct(t *testing.T) {
	now := time.Now()
	deviceID := uuid.New()

	ac := AggregatedCounter{
		Bucket:       now,
		DeviceID:     deviceID,
		CellID:       "cell-1",
		CounterGroup: "LTE.CellMeasReport",
		CounterName:  "PRB.UlAvailProcMeas",
		SumValue:     100.0,
		AvgValue:     25.0,
		MinValue:     10.0,
		MaxValue:     50.0,
		SampleCount:  4,
	}

	assert.Equal(t, now, ac.Bucket)
	assert.Equal(t, deviceID, ac.DeviceID)
	assert.Equal(t, "cell-1", ac.CellID)
	assert.Equal(t, 100.0, ac.SumValue)
	assert.Equal(t, 25.0, ac.AvgValue)
	assert.Equal(t, 10.0, ac.MinValue)
	assert.Equal(t, 50.0, ac.MaxValue)
	assert.Equal(t, int64(4), ac.SampleCount)
}

// G3 字段双向转换：PMCounter ↔ PMMetric 保证无字段丢失
// T-0164-P3 fix: 业务键改用 TR-069 标准 (OUI, DeviceSN) 双键
func Test_counterToMetric_BasicFields(t *testing.T) {
	deviceID := uuid.New()
	now := time.Now()
	c := model.PMCounter{
		Time:         now,
		DeviceID:     deviceID,
		OUI:          "48BF74",
		DeviceSN:     "1202000240194DP0026",
		CellID:       "cell-1",
		CounterGroup: "LTE.CellMeasReport",
		CounterName:  "PRB.UlAvailProcMeas",
		CounterValue: 42.5,
		Granularity:  15,
	}
	m := counterToMetric(c)

	assert.Equal(t, "48BF74", m.DeviceOUI)
	assert.Equal(t, "1202000240194DP0026", m.DeviceSN)
	assert.Equal(t, "PRB.UlAvailProcMeas", m.MetricPath)
	assert.Equal(t, "counter", string(m.MetricType))
	assert.Equal(t, 42.5, m.MetricValue)
	assert.Equal(t, "15min", string(m.Granularity))
	assert.Equal(t, now, m.EndTime)
	assert.Equal(t, now.Add(-15*time.Minute), m.StartTime)
	if assert.NotNil(t, m.ObjectLDN) {
		assert.Equal(t, "cell-1", *m.ObjectLDN)
	}
	assert.Equal(t, "LTE.CellMeasReport", m.Extra["counter_group"])
	assert.Equal(t, deviceID.String(), m.Extra["device_id"])
}

func Test_counterRoundTrip_PreservesCoreFields(t *testing.T) {
	deviceID := uuid.New()
	now := time.Now().UTC().Truncate(time.Second)
	original := model.PMCounter{
		Time:         now,
		DeviceID:     deviceID,
		OUI:          "48BF74",
		DeviceSN:     "1202000240194DP0026",
		CellID:       "cell-7",
		CounterGroup: "LTE.CellMeasReport",
		CounterName:  "PRB.UlAvailProcMeas",
		CounterValue: 7.5,
		Granularity:  15,
	}
	m := counterToMetric(original)
	got := metricToCounter(m)

	assert.Equal(t, deviceID, got.DeviceID, "DeviceID 应从 extra 反查")
	assert.Equal(t, "48BF74", got.OUI)
	assert.Equal(t, "1202000240194DP0026", got.DeviceSN)
	assert.Equal(t, "cell-7", got.CellID)
	assert.Equal(t, "LTE.CellMeasReport", got.CounterGroup)
	assert.Equal(t, "PRB.UlAvailProcMeas", got.CounterName)
	assert.Equal(t, 7.5, got.CounterValue)
	assert.Equal(t, 15, got.Granularity)
}

// issue #14: ErrLateArrival 必须穿透 PgCounterRepository.BatchInsert 包装层不被改写，
// 这样上层 collector 的 errors.Is(err, metrics.ErrLateArrival) 才能识别并降级跳过。
func Test_BatchInsert_PropagatesLateArrivalSentinel(t *testing.T) {
	fake := &fakeMetricsRepo{batchErr: metrics.ErrLateArrival}
	repo := &PgCounterRepository{metricsRepo: fake}

	err := repo.BatchInsert(context.Background(), []model.PMCounter{
		{OUI: "48BF74", DeviceSN: "SN1", CounterName: "c1", CounterValue: 1, Granularity: 15, Time: time.Now()},
	})
	require.Error(t, err)
	assert.True(t, errors.Is(err, metrics.ErrLateArrival), "迟到数据 sentinel 必须穿透 wrapper")
	assert.Equal(t, 1, fake.gotCount, "counter 应转换为 1 条 metric 下传")
}

// 空切片不调用下游、不报错（保持既有快速返回行为）。
func Test_BatchInsert_EmptyNoop(t *testing.T) {
	fake := &fakeMetricsRepo{batchErr: errors.New("should not be called")}
	repo := &PgCounterRepository{metricsRepo: fake}
	require.NoError(t, repo.BatchInsert(context.Background(), nil))
	assert.Equal(t, 0, fake.gotCount)
}

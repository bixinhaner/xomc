package collector

import (
	"context"
	"errors"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/pm"
)

// T-0164 G1 BUG-6 真根因复盘 / 方案 D：filterByWhitelist 行为单测。

// fakeWhitelist 让单测脚本化 LookupCounters 返回值。
// PM-P2：map 改为按 report_key 建键、value 带 {编号, statis_type}（CounterMeta）。
type fakeWhitelist struct {
	set map[string]CounterMeta
	err error
}

func (f *fakeWhitelist) LookupCounters(_ context.Context, _ string) (map[string]CounterMeta, error) {
	return f.set, f.err
}

func collectorWithWhitelist(w CounterWhitelist) *PMCollector {
	return &PMCollector{
		counterWhitelist: w,
		logger:           zap.NewNop(),
	}
}

func sample(names ...string) []model.PMCounter {
	out := make([]model.PMCounter, 0, len(names))
	for _, n := range names {
		out = append(out, model.PMCounter{CounterName: n, CellID: "Cell1"})
	}
	return out
}

func names(cs []model.PMCounter) []string {
	out := make([]string, 0, len(cs))
	for _, c := range cs {
		out = append(out, c.CounterName)
	}
	return out
}

// 核心场景（PM-P2）：白名单按 report_key 含 A/B → 命中后 CounterName 被改写成编号，
// 丢 C/D（孤儿）；保留的 counter 应带正确 StatisType。
func TestFilterByWhitelist_DropsOrphans_RewritesToIndicatorID(t *testing.T) {
	c := collectorWithWhitelist(&fakeWhitelist{set: map[string]CounterMeta{
		"L.Cell.Avail": {IndicatorID: "C000010001", Unit: "%", StatisType: "avg"},
		"RRC.AttConn":  {IndicatorID: "C000010002", Unit: "number", StatisType: "sum"},
	}})
	in := sample("L.Cell.Avail", "MR.RIPPRB", "RRC.AttConn", "MR.RECEIVEDIPOWER")
	out := c.filterByWhitelist(context.Background(), "SN-1", "cmcc", "lte", in)
	// 命中后 CounterName 已是编号（不再是上报名）。
	assert.ElementsMatch(t, []string{"C000010001", "C000010002"}, names(out))
	statisByID := map[string]string{}
	for _, c := range out {
		statisByID[c.CounterName] = c.StatisType
	}
	assert.Equal(t, "avg", statisByID["C000010001"])
	assert.Equal(t, "sum", statisByID["C000010002"])
	unitByID := map[string]string{}
	for _, c := range out {
		unitByID[c.CounterName] = c.Unit
	}
	assert.Equal(t, "%", unitByID["C000010001"])
	assert.Equal(t, "number", unitByID["C000010002"])
}

// PM-P2 锚点专项：上报名匹配 report_key（而非 en_name）。构造 report_key≠en_name 的桩：
// 白名单的键是 report_key（上报名），命中后改写为编号。若误用 en_name 建键，PM 文件
// 上报的 report_key 就匹配不到 → 被当孤儿丢弃，本测试即失败。
func TestFilterByWhitelist_MatchesByReportKeyNotEnName(t *testing.T) {
	// 模拟运维把 en_name 改成人性化名字，report_key 仍是上报名。
	// 白名单按 report_key 建键。
	const reportKey = "L.E-RAB.SuccEst.Ratio.RAW" // 设备上报名
	c := collectorWithWhitelist(&fakeWhitelist{set: map[string]CounterMeta{
		reportKey: {IndicatorID: "C000060216", StatisType: "sum"},
		// 故意放一个 en_name 键，验证它不会被命中（否则说明误用 en_name）。
		"E-RAB建立成功率(原始)": {IndicatorID: "C999999999", StatisType: "avg"},
	}})
	in := sample(reportKey) // PM 文件上报的是 report_key
	out := c.filterByWhitelist(context.Background(), "SN-1", "cmcc", "lte", in)
	require.Len(t, out, 1)
	assert.Equal(t, "C000060216", out[0].CounterName, "应按 report_key 命中并改写为对应编号")
	assert.Equal(t, "sum", out[0].StatisType)
}

// PM-P2：上报名未命中 report_key（野计数器）→ 丢弃。
func TestFilterByWhitelist_UnmatchedReportKey_Dropped(t *testing.T) {
	c := collectorWithWhitelist(&fakeWhitelist{set: map[string]CounterMeta{
		"L.Cell.Avail": {IndicatorID: "C000010001", StatisType: "avg"},
	}})
	in := sample("L.Cell.Avail", "WILD.Counter")
	out := c.filterByWhitelist(context.Background(), "SN-1", "cmcc", "lte", in)
	require.Len(t, out, 1)
	assert.Equal(t, "C000010001", out[0].CounterName)
}

// fail-open：whitelist 未注入（nil）→ 不过滤。
func TestFilterByWhitelist_NilWhitelist_NoFilter(t *testing.T) {
	c := collectorWithWhitelist(nil)
	in := sample("anything", "MR.RIPPRB")
	out := c.filterByWhitelist(context.Background(), "SN-1", "cmcc", "lte", in)
	require.Len(t, out, 2)
}

// fail-open：lookup 报错（DB 故障 / ErrProductNotMatched）→ 不过滤，保留全量。
func TestFilterByWhitelist_LookupError_NoFilter(t *testing.T) {
	c := collectorWithWhitelist(&fakeWhitelist{err: errors.New("router orphan")})
	in := sample("L.Cell.Avail", "MR.RIPPRB")
	out := c.filterByWhitelist(context.Background(), "SN-1", "cmcc", "lte", in)
	require.Len(t, out, 2, "lookup 失败必须 fail-open，避免误删全部")
}

// fail-open：lookup 返回空集合（缓存未热 / 产品无 KPI 配置）→ 不过滤。
func TestFilterByWhitelist_EmptyWhitelist_NoFilter(t *testing.T) {
	c := collectorWithWhitelist(&fakeWhitelist{set: map[string]CounterMeta{}})
	in := sample("any", "thing")
	out := c.filterByWhitelist(context.Background(), "SN-1", "cmcc", "lte", in)
	require.Len(t, out, 2, "空白名单 fail-open，防止误删全部")
}

// 边界：空 counters 不应触发 lookup 调用（节省 router 查询）。
func TestFilterByWhitelist_EmptyCounters_Noop(t *testing.T) {
	called := false
	c := collectorWithWhitelist(&trackingWhitelist{onCall: func() { called = true }})
	out := c.filterByWhitelist(context.Background(), "SN-1", "cmcc", "lte", nil)
	assert.Nil(t, out)
	assert.False(t, called, "空 counters 不应触发 LookupCounters")
}

// BUG-6 回归（最直接的场景）：模拟 Baicells 真机 PM 文件 — `MR.RIPPRB`
// × 53 + `MR.RECEIVEDIPOWER` × 53 + 一个已注册 counter。过滤后只剩注册的那个。
func TestFilterByWhitelist_BUG6_BaicellsOrphans(t *testing.T) {
	c := collectorWithWhitelist(&fakeWhitelist{set: map[string]CounterMeta{
		"RRC.AttConnEstab": {IndicatorID: "C000010099", StatisType: "sum"},
	}})
	in := []model.PMCounter{}
	for i := 1; i <= 53; i++ {
		in = append(in, model.PMCounter{CounterName: "MR.RIPPRB", CellID: "Cell1"})
	}
	for i := 1; i <= 53; i++ {
		in = append(in, model.PMCounter{CounterName: "MR.RECEIVEDIPOWER", CellID: "Cell1"})
	}
	in = append(in, model.PMCounter{CounterName: "RRC.AttConnEstab", CellID: "Cell1"})

	out := c.filterByWhitelist(context.Background(), "SN-1", "cmcc", "lte", in)
	require.Len(t, out, 1)
	// PM-P2：命中后 CounterName 已改写成编号。
	assert.Equal(t, "C000010099", out[0].CounterName)
}

// issue #20：被丢弃的孤儿 counter 数应记入 omc_pm_dropped_counters_total
// （reason=whitelist_miss），让"上报名漂移导致大批 counter 被静默丢弃"可告警。
func TestFilterByWhitelist_DroppedCountersMetric(t *testing.T) {
	reg := prometheus.NewRegistry()
	c := collectorWithWhitelist(&fakeWhitelist{set: map[string]CounterMeta{
		"L.Cell.Avail": {IndicatorID: "C000010001", StatisType: "avg"},
	}})
	c.SetMetrics(pm.NewPMMetrics(reg))

	// 命中 1 个，丢弃 2 个孤儿。
	in := sample("L.Cell.Avail", "MR.RIPPRB", "MR.RECEIVEDIPOWER")
	out := c.filterByWhitelist(context.Background(), "SN-1", "cmcc", "lte", in)
	require.Len(t, out, 1)

	got := testutil.ToFloat64(c.metrics.DroppedCountersTotal.WithLabelValues("cmcc", "lte", "whitelist_miss"))
	assert.Equal(t, float64(2), got, "应记录 2 个被丢弃的孤儿 counter")
}

type trackingWhitelist struct {
	onCall func()
}

func (t *trackingWhitelist) LookupCounters(_ context.Context, _ string) (map[string]CounterMeta, error) {
	t.onCall()
	return nil, nil
}

// ---------------------------------------------------------------------------
// T-0164 G1 BUG-6 续修：applyPayloadIdentity 行为单测
// ---------------------------------------------------------------------------

// BUG 现象：Baicells 真机 PM 文件 `<managedElement localDn="Station=eNb-{SN}"/>`
// → parser extractDeviceSN fallback 取最后段 → CounterSN = "eNb-1202000240194DP0015"。
// payload.DeviceSN（ACS upload handler 从 URL / 文件名解析）= "1202000240194DP0015"。
// 修复：collector 总是用 payload.DeviceSN 覆盖 parser 的输出。
func TestApplyPayloadIdentity_PayloadOverridesParserPrefix(t *testing.T) {
	counters := []model.PMCounter{
		{CounterName: "L.Cell.Avail", DeviceSN: "eNb-1202000240194DP0015"}, // parser 脏值
		{CounterName: "RRC.AttConn", DeviceSN: "eNb-1202000240194DP0015"},
	}
	applyPayloadIdentity(counters, "48BF74", "1202000240194DP0015")

	for i, c := range counters {
		assert.Equal(t, "48BF74", c.OUI, "row %d OUI 必须由 payload 覆盖", i)
		assert.Equal(t, "1202000240194DP0015", c.DeviceSN, "row %d SN 必须用 payload 不带前缀的标准值", i)
	}
}

// 边界：parser 输出 DeviceSN 为空（XML 缺 managedElement）时 payload 同样覆盖。
func TestApplyPayloadIdentity_EmptyParserSN_PayloadFills(t *testing.T) {
	counters := []model.PMCounter{
		{CounterName: "L.Cell.Avail", DeviceSN: ""},
	}
	applyPayloadIdentity(counters, "48BF74", "SN-001")
	require.Len(t, counters, 1)
	assert.Equal(t, "SN-001", counters[0].DeviceSN)
	assert.Equal(t, "48BF74", counters[0].OUI)
}

// 边界：空切片不 panic。
func TestApplyPayloadIdentity_EmptyCounters_Noop(t *testing.T) {
	applyPayloadIdentity(nil, "48BF74", "SN-001")
	applyPayloadIdentity([]model.PMCounter{}, "48BF74", "SN-001")
}

package collector

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/model"
)

// T-0164 G1 BUG-6 真根因复盘 / 方案 D：filterByWhitelist 行为单测。

// fakeWhitelist 让单测脚本化 LookupCounters 返回值。
type fakeWhitelist struct {
	set map[string]struct{}
	err error
}

func (f *fakeWhitelist) LookupCounters(_ context.Context, _ string) (map[string]struct{}, error) {
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

// 核心场景：白名单含 A/B，丢 C/D（孤儿）。
func TestFilterByWhitelist_DropsOrphans(t *testing.T) {
	c := collectorWithWhitelist(&fakeWhitelist{set: map[string]struct{}{
		"L.Cell.Avail": {},
		"RRC.AttConn":  {},
	}})
	in := sample("L.Cell.Avail", "MR.RIPPRB", "RRC.AttConn", "MR.RECEIVEDIPOWER")
	out := c.filterByWhitelist(context.Background(), "SN-1", in)
	assert.ElementsMatch(t, []string{"L.Cell.Avail", "RRC.AttConn"}, names(out))
}

// fail-open：whitelist 未注入（nil）→ 不过滤。
func TestFilterByWhitelist_NilWhitelist_NoFilter(t *testing.T) {
	c := collectorWithWhitelist(nil)
	in := sample("anything", "MR.RIPPRB")
	out := c.filterByWhitelist(context.Background(), "SN-1", in)
	require.Len(t, out, 2)
}

// fail-open：lookup 报错（DB 故障 / ErrProductNotMatched）→ 不过滤，保留全量。
func TestFilterByWhitelist_LookupError_NoFilter(t *testing.T) {
	c := collectorWithWhitelist(&fakeWhitelist{err: errors.New("router orphan")})
	in := sample("L.Cell.Avail", "MR.RIPPRB")
	out := c.filterByWhitelist(context.Background(), "SN-1", in)
	require.Len(t, out, 2, "lookup 失败必须 fail-open，避免误删全部")
}

// fail-open：lookup 返回空集合（缓存未热 / 产品无 KPI 配置）→ 不过滤。
func TestFilterByWhitelist_EmptyWhitelist_NoFilter(t *testing.T) {
	c := collectorWithWhitelist(&fakeWhitelist{set: map[string]struct{}{}})
	in := sample("any", "thing")
	out := c.filterByWhitelist(context.Background(), "SN-1", in)
	require.Len(t, out, 2, "空白名单 fail-open，防止误删全部")
}

// 边界：空 counters 不应触发 lookup 调用（节省 router 查询）。
func TestFilterByWhitelist_EmptyCounters_Noop(t *testing.T) {
	called := false
	c := collectorWithWhitelist(&trackingWhitelist{onCall: func() { called = true }})
	out := c.filterByWhitelist(context.Background(), "SN-1", nil)
	assert.Nil(t, out)
	assert.False(t, called, "空 counters 不应触发 LookupCounters")
}

// BUG-6 回归（最直接的场景）：模拟 Baicells 真机 PM 文件 — `MR.RIPPRB`
// × 53 + `MR.RECEIVEDIPOWER` × 53 + 一个已注册 counter。过滤后只剩注册的那个。
func TestFilterByWhitelist_BUG6_BaicellsOrphans(t *testing.T) {
	c := collectorWithWhitelist(&fakeWhitelist{set: map[string]struct{}{
		"RRC.AttConnEstab": {},
	}})
	in := []model.PMCounter{}
	for i := 1; i <= 53; i++ {
		in = append(in, model.PMCounter{CounterName: "MR.RIPPRB", CellID: "Cell1"})
	}
	for i := 1; i <= 53; i++ {
		in = append(in, model.PMCounter{CounterName: "MR.RECEIVEDIPOWER", CellID: "Cell1"})
	}
	in = append(in, model.PMCounter{CounterName: "RRC.AttConnEstab", CellID: "Cell1"})

	out := c.filterByWhitelist(context.Background(), "SN-1", in)
	require.Len(t, out, 1)
	assert.Equal(t, "RRC.AttConnEstab", out[0].CounterName)
}

type trackingWhitelist struct {
	onCall func()
}

func (t *trackingWhitelist) LookupCounters(_ context.Context, _ string) (map[string]struct{}, error) {
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

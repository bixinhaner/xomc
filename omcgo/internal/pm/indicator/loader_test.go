package indicator

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/core/appconfig"
)

// 单元测试范围限定为 P2-09 引入的纯函数：
//   - parseEnabledFlag
//   - aggregateEnabledOR
//
// SQL 路径（refreshDefaultEnabledBucket）需要 testcontainers/dockertest，
// 留给集成测试覆盖（P3-03 KPI 端点上线后端到端）。

func TestParseEnabledFlag(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"", true},
		{"true", true},
		{"TRUE", true},
		{"True", true},
		{"1", true},
		{"t", true},
		{"yes", true}, // 容错：默认 true
		{"random", true},
		{"false", false},
		{"FALSE", false},
		{"0", false},
		{"f", false},
		{"  false  ", false}, // trim
		{"no", false},
		{"n", false},
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			assert.Equal(t, c.want, parseEnabledFlag(c.in))
		})
	}
}

func TestAggregateEnabledOR_Empty(t *testing.T) {
	got := aggregateEnabledOR(nil)
	assert.Empty(t, got)

	got = aggregateEnabledOR([]xmlIndicatorModel{{}})
	assert.Empty(t, got)
}

func TestAggregateEnabledOR_SingleFile(t *testing.T) {
	docs := []xmlIndicatorModel{
		{Indicators: []xmlIndicator{
			{ID: "X", Enabled: "true"},
			{ID: "Y", Enabled: "false"},
			{ID: "Z", Enabled: ""}, // 缺省视 true
		}},
	}
	got := aggregateEnabledOR(docs)
	assert.True(t, got["X"])
	assert.False(t, got["Y"])
	assert.True(t, got["Z"])
	assert.Len(t, got, 3)
}

func TestAggregateEnabledOR_MultiFile_TrueWins(t *testing.T) {
	// 同一 id "A" 在文件 1 false / 文件 2 true → 应得 true（OR 合并）
	docs := []xmlIndicatorModel{
		{Indicators: []xmlIndicator{{ID: "A", Enabled: "false"}}},
		{Indicators: []xmlIndicator{{ID: "A", Enabled: "true"}}},
	}
	got := aggregateEnabledOR(docs)
	assert.True(t, got["A"], "OR 合并：任意一个文件 enabled=true → true")
}

func TestAggregateEnabledOR_MultiFile_AllFalseStaysFalse(t *testing.T) {
	docs := []xmlIndicatorModel{
		{Indicators: []xmlIndicator{{ID: "B", Enabled: "false"}}},
		{Indicators: []xmlIndicator{{ID: "B", Enabled: "0"}}},
	}
	got := aggregateEnabledOR(docs)
	assert.False(t, got["B"], "全 false → false")
}

func TestAggregateEnabledOR_MultiFile_OrderAgnostic(t *testing.T) {
	// 反向：true 先出现，再 false 不应该把 true 翻回去
	docs := []xmlIndicatorModel{
		{Indicators: []xmlIndicator{{ID: "C", Enabled: "true"}}},
		{Indicators: []xmlIndicator{{ID: "C", Enabled: "false"}}},
	}
	got := aggregateEnabledOR(docs)
	assert.True(t, got["C"], "OR 不受顺序影响")
}

func TestAggregateEnabledOR_EmptyIDIgnored(t *testing.T) {
	docs := []xmlIndicatorModel{
		{Indicators: []xmlIndicator{
			{ID: "", Enabled: "true"},
			{ID: "ok", Enabled: "true"},
		}},
	}
	got := aggregateEnabledOR(docs)
	assert.NotContains(t, got, "")
	assert.True(t, got["ok"])
	assert.Len(t, got, 1)
}

func TestAggregateEnabledOR_DefaultEnabledTreatedAsTrue(t *testing.T) {
	// 缺省 enabled 属性视为 true（XML 现网约定）
	docs := []xmlIndicatorModel{
		{Indicators: []xmlIndicator{{ID: "D"}}}, // Enabled = "" 缺省
	}
	got := aggregateEnabledOR(docs)
	assert.True(t, got["D"])
}

// #98：reload 重灌 builtin 公式时，已被用户自定义公式覆盖的 (平台, 指标) 必须跳过，
// 避免与保留下来的自定义行重复 / 覆盖用户意图。
func TestExcludeCustomOverridden(t *testing.T) {
	formulas := []formulaRow{
		{PlatformName: "P1", IndicatorID: "I1", Formula: "a", LoadedFrom: "x.xml"},
		{PlatformName: "P1", IndicatorID: "I2", Formula: "b", LoadedFrom: "x.xml"},
		{PlatformName: "P2", IndicatorID: "I1", Formula: "c", LoadedFrom: "y.xml"},
	}

	// 无自定义键 → 原样返回
	assert.Len(t, excludeCustomOverridden(formulas, nil), 3)

	// 自定义覆盖 P1/I1 → 丢弃该 builtin 行，保留其余两条；
	// (平台,指标) 是联合键，P2/I1 不同键不应被误删。
	custom := map[formulaKey]struct{}{{platform: "P1", indicator: "I1"}: {}}
	got := excludeCustomOverridden(formulas, custom)
	assert.Len(t, got, 2)
	hasP1I1, hasP2I1 := false, false
	for _, f := range got {
		if f.PlatformName == "P1" && f.IndicatorID == "I1" {
			hasP1I1 = true
		}
		if f.PlatformName == "P2" && f.IndicatorID == "I1" {
			hasP2I1 = true
		}
	}
	assert.False(t, hasP1I1, "被自定义覆盖的 P1/I1 不应重插")
	assert.True(t, hasP2I1, "P2/I1 是不同键，不应被 P1/I1 的覆盖误删")
}

// findIndicatorByID 在已解析的 XML 文档集合里按 id 找指标（issue #389 阶段1 校验用）。
func findIndicatorByID(docs []xmlIndicatorModel, id string) (xmlIndicator, bool) {
	for _, doc := range docs {
		for _, ind := range doc.Indicators {
			if ind.ID == id {
				return ind, true
			}
		}
	}
	return xmlIndicator{}, false
}

// parseLibFiles 读取并解析给定相对路径（相对 data/indicator-library）的 XML 文件集合。
func parseLibFiles(t *testing.T, rels ...string) []xmlIndicatorModel {
	t.Helper()
	docs := make([]xmlIndicatorModel, 0, len(rels))
	for _, rel := range rels {
		var doc xmlIndicatorModel
		require.NoError(t, readXML(filepath.Join("..", "..", "..", "data", "indicator-library", rel), &doc), rel)
		docs = append(docs, doc)
	}
	return docs
}

// TestStatisDurationAndCellAvailabilityRegistered 守 issue #389 阶段1 死判
// [indicators-registered-3tech]：指标库三制式各自登记「统计时长」计数器（is_counter=1 /
// statisType=sum）与「小区可用率」派生指标（is_counter=0 / statisType=pct / 公式含
// 在服时长÷统计时长×100）。直接解析出厂 XML 资产校验，与 dictloader 落库前的真相源对齐。
func TestStatisDurationAndCellAvailabilityRegistered(t *testing.T) {
	cases := []struct {
		tech       string
		files      []string
		durationID string
		availID    string
		availArith string // 期望小区可用率公式（arithmetic，C 编号口径）
	}{
		{
			tech:       "4G/ENB",
			files:      []string{"enb/BLX.xml", "enb/BM.xml", "enb/MLQ.xml", "enb/BLQ.xml", "enb/MLN.xml"},
			durationID: "C000060273",
			availID:    "K900010076",
			availArith: "C000060216/C000060273*100",
		},
		{
			tech:       "5G/GNB",
			files:      []string{"GNB.xml"},
			durationID: "C010120025",
			availID:    "KGNB0570",
			availArith: "OTHER.CellServiceTime/C010120025*100",
		},
		{
			tech:       "GSM",
			files:      []string{"GSM.xml"},
			durationID: "CGSM0080001",
			availID:    "KGSM0143",
			availArith: "OTHER.CellServiceTime/CGSM0080001*100",
		},
	}
	for _, c := range cases {
		t.Run(c.tech, func(t *testing.T) {
			docs := parseLibFiles(t, c.files...)

			// 统计时长：计数器 + sum + 合成（reportKey=OTHER.StatisDuration）。
			dur, ok := findIndicatorByID(docs, c.durationID)
			require.True(t, ok, "%s 统计时长 %s 应登记", c.tech, c.durationID)
			assert.Equal(t, "1", dur.IsCounter, "统计时长应为计数器")
			assert.Equal(t, "sum", dur.StatisType, "统计时长应累加")
			assert.Equal(t, "OTHER.StatisDuration", dur.ReportKey)
			assert.Equal(t, "统计时长", dur.CnName)

			// 小区可用率：派生 + pct + 公式 = 在服时长÷统计时长×100。
			av, ok := findIndicatorByID(docs, c.availID)
			require.True(t, ok, "%s 小区可用率 %s 应登记", c.tech, c.availID)
			assert.Equal(t, "0", av.IsCounter, "小区可用率应为派生指标")
			assert.Equal(t, "pct", av.StatisType, "小区可用率统计类型应为 pct")
			assert.Equal(t, c.availArith, av.Arithmetic, "%s 小区可用率公式应为在服÷统计×100", c.tech)
			assert.Equal(t, "OTHER.CellServiceTime/OTHER.StatisDuration*100", av.Formula)
			assert.Equal(t, "小区可用率", av.CnName)
		})
	}
}

// ISSUE-389 阶段2 修复：指标库加载成功后必须 bump 下游 KPI 路由缓存，
// 否则新增的「统计时长」计数器 report_key 不在已缓存的旧路由白名单里，落库被丢弃。
// 这里覆盖 bump 调用的三条语义：成功调用、bump 报错不影响加载、nil bumper 安全空跑。

func TestBumpDownstreamCache_InvokesBumperOnSuccess(t *testing.T) {
	called := 0
	l := NewLoader(nil, appcfg(), "", nil).WithCacheBumper(func(_ context.Context) error {
		called++
		return nil
	})
	l.bumpDownstreamCache(context.Background())
	assert.Equal(t, 1, called, "加载成功后应调用一次 cacheBumper（bump KPI 路由 cache_version）")
}

func TestBumpDownstreamCache_BumpErrorDoesNotPanic(t *testing.T) {
	l := NewLoader(nil, appcfg(), "", nil).WithCacheBumper(func(_ context.Context) error {
		return errors.New("redis down")
	})
	// bump 失败仅告警（下游靠 24h TTL 兜底），不得 panic / 不抛错。
	assert.NotPanics(t, func() { l.bumpDownstreamCache(context.Background()) })
}

func TestBumpDownstreamCache_NilBumperIsNoOp(t *testing.T) {
	l := NewLoader(nil, appcfg(), "", nil) // 未注入 bumper（无 Redis 部署）
	assert.NotPanics(t, func() { l.bumpDownstreamCache(context.Background()) })
}

func appcfg() appconfig.IndicatorLoaderConfig { return appconfig.IndicatorLoaderConfig{} }

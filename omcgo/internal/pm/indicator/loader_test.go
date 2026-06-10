package indicator

import (
	"testing"

	"github.com/stretchr/testify/assert"
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

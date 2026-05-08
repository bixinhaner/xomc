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

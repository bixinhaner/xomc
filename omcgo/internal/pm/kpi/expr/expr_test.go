package expr

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParse_DottedCounterNames 覆盖真实平台公式（rela_platform_indicator_formula_enb）里
// 的 3GPP 点分计数名。修复前解析器把 '.' 当数字 / 报 unexpected character，导致全部真实
// 公式解析失败 → 全库零 KPI。
func TestParse_DottedCounterNames(t *testing.T) {
	tests := []struct {
		name    string
		formula string
		values  map[string]float64
		want    float64
	}{
		{
			name:    "two-segment dotted division",
			formula: "MAC.RachSuccess/MAC.RachAttempt*100",
			values:  map[string]float64{"MAC.RachSuccess": 90, "MAC.RachAttempt": 100},
			want:    90.0,
		},
		{
			name:    "three-segment dotted with .Sum",
			formula: "(ERAB.EstabInitSuccNbr.Sum+ERAB.EstabAddSuccNbr.Sum)/(ERAB.EstabInitAttNbr.Sum+ERAB.EstabAddAttNbr.Sum)*100",
			values: map[string]float64{
				"ERAB.EstabInitSuccNbr.Sum": 80,
				"ERAB.EstabAddSuccNbr.Sum":  10,
				"ERAB.EstabInitAttNbr.Sum":  90,
				"ERAB.EstabAddAttNbr.Sum":   10,
			},
			want: 90.0,
		},
		{
			name:    "digit-leading segment Cqi.00",
			formula: "Cqi.00+Cqi.01",
			values:  map[string]float64{"Cqi.00": 3, "Cqi.01": 4},
			want:    7.0,
		},
		{
			name:    "underscore suffix segment",
			formula: "CONTEXT.NbrLeftLastPer_1*2",
			values:  map[string]float64{"CONTEXT.NbrLeftLastPer_1": 5},
			want:    10.0,
		},
		{
			name:    "single dotted counter no operator",
			formula: "DRB.PdcpSduAirLossRateDl.Sum",
			values:  map[string]float64{"DRB.PdcpSduAirLossRateDl.Sum": 42.5},
			want:    42.5,
		},
		{
			name:    "dotted divided by plain number constant",
			formula: "(PDCP.UpOctDl+PDCP.CpOctDl)/1000",
			values:  map[string]float64{"PDCP.UpOctDl": 600, "PDCP.CpOctDl": 400},
			want:    1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := Parse(tt.formula)
			require.NoError(t, err, "formula should parse: %s", tt.formula)
			got, err := f.Evaluate(tt.values)
			require.NoError(t, err)
			assert.InDelta(t, tt.want, got, 0.0001)
		})
	}
}

// TestIdentifiers_DottedNames 确保 Identifiers() 返回完整点分名（router 依赖此结果
// 生成 QueryForKPI 的 metric_path 过滤集；切碎会查不到 counter）。
func TestIdentifiers_DottedNames(t *testing.T) {
	f, err := Parse("MAC.RachSuccess/MAC.RachAttempt*100")
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"MAC.RachSuccess", "MAC.RachAttempt"}, f.Identifiers())
}

// TestParse_NumbersStillWork 回归：纯数字 / 小数常量不能被点分标识符逻辑误伤。
func TestParse_NumbersStillWork(t *testing.T) {
	f, err := Parse("a * 1.5 + 100")
	require.NoError(t, err)
	got, err := f.Evaluate(map[string]float64{"a": 2})
	require.NoError(t, err)
	assert.InDelta(t, 103.0, got, 0.0001)
}

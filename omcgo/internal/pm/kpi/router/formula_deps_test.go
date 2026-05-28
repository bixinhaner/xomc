package router

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestExtractFormulaDeps 验证点分 3GPP 计数名被作为完整依赖提取（而非按 '.' 切碎）。
// 切碎会导致 KPIEngine 用错误的 metric_path 查 counter → 查不到 → 全库零 KPI。
func TestExtractFormulaDeps(t *testing.T) {
	tests := []struct {
		name    string
		formula string
		want    []string
	}{
		{
			name:    "dotted division",
			formula: "MAC.RachSuccess/MAC.RachAttempt*100",
			want:    []string{"MAC.RachSuccess", "MAC.RachAttempt"},
		},
		{
			name:    "three-segment with .Sum and dedup",
			formula: "(ERAB.EstabInitSuccNbr.Sum+ERAB.EstabInitSuccNbr.Sum)/ERAB.EstabInitAttNbr.Sum",
			want:    []string{"ERAB.EstabInitSuccNbr.Sum", "ERAB.EstabInitAttNbr.Sum"},
		},
		{
			name:    "underscore-only names still work",
			formula: "RRC_Conn_Succ / RRC_Conn_Att",
			want:    []string{"RRC_Conn_Succ", "RRC_Conn_Att"},
		},
		{
			name:    "numeric constants excluded",
			formula: "(PDCP.UpOctDl+PDCP.CpOctDl)/1000",
			want:    []string{"PDCP.UpOctDl", "PDCP.CpOctDl"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.ElementsMatch(t, tt.want, extractFormulaDeps(tt.formula))
		})
	}
}

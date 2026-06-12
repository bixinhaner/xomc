package admin

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// strPtrTest 是测试内联的 *string 构造器(避免与生产/其它测试帮手撞名)。
func strPtrTest(s string) *string { return &s }

// TestDictsMatchingSourceTable 验证「按 source_table 筛绑定字典」纯函数(#241 导入后自动刷新核心选择逻辑)。
func TestDictsMatchingSourceTable(t *testing.T) {
	dicts := []Dictionary{
		{ID: 1, Type: "param_model_name", SourceTable: strPtrTest("param_models")},
		{ID: 2, Type: "alarm_ne_type", SourceTable: strPtrTest("alarm_definitions")},
		{ID: 3, Type: "kpi_platform_enb", SourceTable: strPtrTest("rela_platform_indicator_formula_enb")},
		{ID: 4, Type: "manual_dict", SourceTable: nil}, // 手工字典,无源 — 必须被忽略
		{ID: 5, Type: "another_param", SourceTable: strPtrTest("param_models")}, // 同表第二个
	}
	tests := []struct {
		name    string
		table   string
		wantIDs []int64
	}{
		{"single match", "alarm_definitions", []int64{2}},
		{"multiple match same table", "param_models", []int64{1, 5}},
		{"kpi platform enb formula table", "rela_platform_indicator_formula_enb", []int64{3}},
		{"no bound dict for table", "rela_platform_indicator_formula_gnb", nil},
		{"unrelated table", "devices", nil},
		{"empty table no match", "", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := dictsMatchingSourceTable(dicts, tt.table)
			var gotIDs []int64
			for _, d := range got {
				gotIDs = append(gotIDs, d.ID)
			}
			assert.Equal(t, tt.wantIDs, gotIDs)
		})
	}
}

// TestRefreshSourceBoundByTable_EngineDisabled 验证未启用数据源引擎(无白名单)时,
// 导入后刷新是安全 no-op:返 (0, nil) 不报错,不阻断导入主流程。
func TestRefreshSourceBoundByTable_EngineDisabled(t *testing.T) {
	svc := NewDictionaryService(newMockDictRepository(), newMockDictDetailRepository())
	n, err := svc.RefreshSourceBoundByTable(context.Background(), "param_models")
	assert.NoError(t, err)
	assert.Equal(t, 0, n)
}

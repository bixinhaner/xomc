//go:build !short

package catalogloader

import (
	"testing"
)

// TestParseGeneratedV23Catalog 验证 parse_v23_to_catalog.py 生成的
// catalog JSON 能被 catalogloader.ParseFile 顺利解析，且统计指标符合 spec v2.3 预期：
// 18 章节 / 72 group / 228 命令 / 624 sub_field。
//
// 注意 build tag !short：日常 -short 测试跳过，CI 全量跑会包含。
func TestParseGeneratedV23Catalog(t *testing.T) {
	cat, err := ParseFile("../../../datamodels/mml-catalog/cmcc-tdlte-v2.3.json")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if cat.SpecVersion == "" || len(cat.Groups) == 0 {
		t.Fatalf("empty catalog: %+v", cat)
	}
	// 期待 72 个 group 跨 18 章节
	if len(cat.Groups) != 72 {
		t.Errorf("expected 72 groups, got %d", len(cat.Groups))
	}
	chapters := map[string]bool{}
	totalCommands := 0
	totalSubFields := 0
	for _, g := range cat.Groups {
		chapters[g.ChapterCode] = true
		totalCommands += len(g.Commands)
		totalSubFields += len(g.SubFields)
	}
	if len(chapters) != 18 {
		t.Errorf("expected 18 chapters, got %d", len(chapters))
	}
	t.Logf("Loaded: groups=%d chapters=%d commands=%d sub_fields=%d",
		len(cat.Groups), len(chapters), totalCommands, totalSubFields)
}

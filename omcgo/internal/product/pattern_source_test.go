package product

import (
	"strings"
	"testing"
)

// TestIsCustomPattern 锁定 source → 可编辑/删除/移动 的唯一判定口径：
// 仅 'custom' 为真，其余（builtin / 空串 / 未知值）一律只读。
func TestIsCustomPattern(t *testing.T) {
	cases := []struct {
		source string
		want   bool
	}{
		{patternSourceCustom, true},
		{"custom", true},
		{patternSourceBuiltin, false},
		{"builtin", false},
		{"", false},
		{"unknown", false},
		{"CUSTOM", false}, // 大小写敏感，与 DB CHECK 取值一致
	}
	for _, c := range cases {
		if got := isCustomPattern(c.source); got != c.want {
			t.Errorf("isCustomPattern(%q) = %v, want %v", c.source, got, c.want)
		}
	}
}

// TestCustomPatternOrderBase_AboveBuiltinOrders 守护 custom 高位段不与
// products.xml 的 globalOrder 撞 uniq_product_class_patterns_global_order：
// 当前内置正则 globalOrder 仅个位/十位数，留足量级裕度即可保证重灌不冲突。
func TestCustomPatternOrderBase_AboveBuiltinOrders(t *testing.T) {
	const maxPlausibleBuiltinOrder = 100_000 // 远超当前 ~29，含未来扩张裕度
	if customPatternOrderBase <= maxPlausibleBuiltinOrder {
		t.Fatalf("customPatternOrderBase=%d must stay well above any builtin globalOrder (%d)",
			customPatternOrderBase, maxPlausibleBuiltinOrder)
	}
}

// TestDeleteBuiltinPatternsSQL_GuardsUserProducts 锁定 issue #206 的核心修复：
// loader 销毁式 reload 的第一步 DELETE 必须同时满足两条守卫，否则用户自建产品
// （is_builtin=FALSE）的正则会被连带删除、升级后正则列变空、且不会从 products.xml 重插。
//
// 成功路径（修复后语义）：DELETE 同时带 source='builtin' 与
//
//	product_id IN (SELECT id FROM products WHERE is_builtin = TRUE)
//
// 两条限定 → 只删内置产品的内置正则，用户产品正则一律不动。
// 失败路径（旧实现回归守卫）：若退回到「只按 source 删、无产品归属限定」的旧语句，
// 本测试失败——防止有人无意中把守卫去掉再次引入 #206。
func TestDeleteBuiltinPatternsSQL_GuardsUserProducts(t *testing.T) {
	normalized := strings.Join(strings.Fields(deleteBuiltinPatternsSQL), " ")

	// 成功路径：必须仍只删 builtin source（保留 builtin 刷新语义）。
	if !strings.Contains(normalized, "source = 'builtin'") {
		t.Fatalf("reload DELETE must still target source='builtin' (got %q)", normalized)
	}

	// 成功路径：必须有产品归属守卫，把删除限定在内置产品上。
	const guard = "product_id IN (SELECT id FROM products WHERE is_builtin = TRUE)"
	if !strings.Contains(normalized, guard) {
		t.Fatalf("reload DELETE must guard by is_builtin=TRUE product ownership to "+
			"protect user product patterns (issue #206); got %q", normalized)
	}

	// 失败路径回归守卫：禁止退回到无 product_id 限定的全量删（旧 #206 行为）。
	if !strings.Contains(normalized, "product_id") {
		t.Fatalf("reload DELETE must not delete builtin patterns across all products "+
			"(would re-introduce issue #206); got %q", normalized)
	}
}

package product

import "testing"

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

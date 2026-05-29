package appconfig

import "testing"

// TestCustomOverridesEnabled_TriState 验证 T-0178 的 *bool 三态契约:
//   - nil(yaml 不写)→ 默认 true(用户决策 1: 同名 custom 胜出)
//   - &true → 显式 true
//   - &false → 显式 false(builtin 胜出)
//
// 防御性测试:Go bool 零值是 false,如果改回直接 bool 类型,yaml 不写就会
// 静默拿到 false → 与设计意图反转 → 整条 self-healing 链失效。该测试在
// 字段类型改回 bool 时会编译失败(无法对非 nil bool 取 nil)。
func TestCustomOverridesEnabled_TriState(t *testing.T) {
	tr := true
	fa := false

	cases := []struct {
		name string
		ptr  *bool
		want bool
	}{
		{"nil = default true", nil, true},
		{"explicit true", &tr, true},
		{"explicit false", &fa, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := ParamModelLoaderConfig{CustomOverrides: tc.ptr}
			if got := c.CustomOverridesEnabled(); got != tc.want {
				t.Errorf("CustomOverridesEnabled() = %v, want %v (ptr=%v)",
					got, tc.want, tc.ptr)
			}
		})
	}
}

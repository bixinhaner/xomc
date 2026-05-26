package task

import "testing"

func TestIsSupportedPlatform(t *testing.T) {
	cases := map[string]bool{
		"BLQ":      true,
		"MLQ":      true,
		"MLN":      true,
		"BM":       true,
		"BLX":      false, // 不在允许列表（即使其 productClass 解析为 BLQ 也会先解析后判断）
		"BaiBNQ":   false,
		"BSC":      false,
		"":         false,
		"  BLQ  ":  true, // TrimSpace
		"blq":      false, // 大小写敏感（param_models.name 全大写规范）
	}
	for in, want := range cases {
		t.Run(in, func(t *testing.T) {
			got := IsSupportedPlatform(in)
			if got != want {
				t.Errorf("IsSupportedPlatform(%q) = %v, want %v", in, got, want)
			}
		})
	}
}

func TestSupportedPlatforms_HasExpectedFour(t *testing.T) {
	got := SupportedPlatforms()
	if len(got) != 4 {
		t.Errorf("expected exactly 4 supported platforms, got %d: %v", len(got), got)
	}
	expected := map[string]struct{}{"BLQ": {}, "MLQ": {}, "MLN": {}, "BM": {}}
	for _, name := range got {
		if _, ok := expected[name]; !ok {
			t.Errorf("unexpected platform %q in supported list", name)
		}
	}
}

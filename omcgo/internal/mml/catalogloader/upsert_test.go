package catalogloader

import "testing"

// TestIsWriteOperation 验证 §15.3 D34 跳过判定：仅 MOD/ADD/RMV 需要非空 target_paths。
func TestIsWriteOperation(t *testing.T) {
	cases := []struct {
		op   string
		want bool
	}{
		{"LST", false},
		{"MOD", true},
		{"ADD", true},
		{"RMV", true},
		{"DSP", false},
		{"", false},
		{"FOO", false},
	}
	for _, c := range cases {
		if got := isWriteOperation(c.op); got != c.want {
			t.Errorf("isWriteOperation(%q) = %v, want %v", c.op, got, c.want)
		}
	}
}

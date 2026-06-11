package software

import "testing"

func TestCompareFirmwareVersion(t *testing.T) {
	tests := []struct {
		name    string
		a, b    string
		wantCmp int
		wantOK  bool
	}{
		{"equal with V prefix", "V1.0.0", "V1.0.0", 0, true},
		{"equal mixed prefix case", "v2.5.7", "V2.5.7", 0, true},
		{"a older minor", "V1.9.0", "V2.0.0", -1, true},
		{"a newer patch", "V2.5.8", "V2.5.7", 1, true},
		{"shorter equals padded", "V1", "V1.0.0", 0, true},
		{"shorter older", "V1", "V1.0.1", -1, true},
		{"no prefix", "3.0.0", "2.9.9", 1, true},
		{"empty a not comparable", "", "V1.0.0", 0, false},
		{"empty b not comparable", "V1.0.0", "", 0, false},
		{"non-numeric segment not comparable", "V1.0.beta", "V1.0.0", 0, false},
		{"prefix-only not comparable", "V", "V1.0.0", 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmp, ok := compareFirmwareVersion(tt.a, tt.b)
			if ok != tt.wantOK {
				t.Fatalf("compareFirmwareVersion(%q,%q) ok=%v, want %v", tt.a, tt.b, ok, tt.wantOK)
			}
			if ok && cmp != tt.wantCmp {
				t.Fatalf("compareFirmwareVersion(%q,%q) cmp=%d, want %d", tt.a, tt.b, cmp, tt.wantCmp)
			}
		})
	}
}

func TestIsDowngrade(t *testing.T) {
	tests := []struct {
		name             string
		current, target  string
		wantIsDowngrade  bool
	}{
		{"strict downgrade blocked", "V3.0.0", "V2.5.7", true},
		{"same version not downgrade", "V2.0.0", "V2.0.0", false},
		{"upgrade not downgrade", "V1.0.0", "V2.0.0", false},
		{"unparseable target → not downgrade (fail-open guard)", "V3.0.0", "stable", false},
		{"unparseable current → not downgrade", "latest", "V1.0.0", false},
		{"empty target → not downgrade", "V3.0.0", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isDowngrade(tt.current, tt.target); got != tt.wantIsDowngrade {
				t.Fatalf("isDowngrade(current=%q,target=%q)=%v, want %v",
					tt.current, tt.target, got, tt.wantIsDowngrade)
			}
		})
	}
}

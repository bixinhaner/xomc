package collector

import "testing"

func TestEnsurePMFileSize(t *testing.T) {
	tests := []struct {
		name    string
		size    int64
		wantErr bool
	}{
		{"zero", 0, false},
		{"under limit", maxPMFileBytes - 1, false},
		{"at limit", maxPMFileBytes, false},
		{"one over limit", maxPMFileBytes + 1, true},
		{"way over limit", maxPMFileBytes * 4, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ensurePMFileSize(tt.size)
			if (err != nil) != tt.wantErr {
				t.Errorf("ensurePMFileSize(%d) err=%v, wantErr=%v", tt.size, err, tt.wantErr)
			}
		})
	}
}

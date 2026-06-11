package collector

import "testing"

func TestEnsureMRFileSize(t *testing.T) {
	tests := []struct {
		name    string
		size    int64
		wantErr bool
	}{
		{"zero", 0, false},
		{"under limit", maxMRFileBytes - 1, false},
		{"at limit", maxMRFileBytes, false},
		{"one over limit", maxMRFileBytes + 1, true},
		{"way over limit", maxMRFileBytes * 4, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ensureMRFileSize(tt.size)
			if (err != nil) != tt.wantErr {
				t.Errorf("ensureMRFileSize(%d) err=%v, wantErr=%v", tt.size, err, tt.wantErr)
			}
		})
	}
}

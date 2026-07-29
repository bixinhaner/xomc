package stream

import "testing"

func TestQueueWatermarkAllowsTimeoutCloseOnlyAfterBarrierConsumed(t *testing.T) {
	tests := []struct {
		name         string
		graceElapsed bool
		pending      int64
		want         bool
	}{
		{name: "before minimum grace", graceElapsed: false, pending: 0, want: false},
		{name: "queue still has preceding event", graceElapsed: true, pending: 1, want: false},
		{name: "minimum grace and consumed barrier", graceElapsed: true, pending: 0, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := QueueWatermarkAllowsClose(tt.graceElapsed, tt.pending); got != tt.want {
				t.Fatalf("QueueWatermarkAllowsClose(%v, %d) = %v, want %v",
					tt.graceElapsed, tt.pending, got, tt.want)
			}
		})
	}
}

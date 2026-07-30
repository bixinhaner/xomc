package rawcleanup

import (
	"math"
	"testing"
	"time"
)

func TestTargetRateScalesWithTenThousandDevices(t *testing.T) {
	got := TargetRate(10000.0/900.0, 0, 1.5, 5, 100)
	if got != 17 {
		t.Fatalf("rate = %v, want 17 objects/s", got)
	}
	wantInterval := time.Duration(float64(time.Second) * 100 / got)
	if interval := BatchInterval(100, got); interval != wantInterval {
		t.Fatalf("interval = %v", interval)
	}
}

func TestTargetRateUsesObservedTrafficAndBounds(t *testing.T) {
	tests := []struct {
		predicted, observed float64
		want                float64
	}{
		{1, 2, 5},
		{10, 20, 30},
		{200, 1, 100},
	}
	for _, tt := range tests {
		if got := TargetRate(tt.predicted, tt.observed, 1.5, 5, 100); got != tt.want {
			t.Fatalf("TargetRate(%v,%v)=%v want %v", tt.predicted, tt.observed, got, tt.want)
		}
	}
}

func TestPressureDecisionProtectsBusinessTraffic(t *testing.T) {
	atBoundary := time.Date(2026, 7, 27, 8, 14, 0, 0, time.UTC)
	tests := []struct {
		name string
		p    Pressure
		now  time.Time
		rate float64
		want float64
		stop bool
	}{
		{"monitoring fallback caps rate", Pressure{}, time.Date(2026, 7, 27, 8, 7, 0, 0, time.UTC), 80, 20, false},
		{"pm boundary quarters rate", Pressure{MonitoringAvailable: true}, atBoundary, 80, 20, false},
		{"high cpu halves rate", Pressure{MonitoringAvailable: true, CPUPercent: 85}, time.Date(2026, 7, 27, 8, 7, 0, 0, time.UTC), 80, 40, false},
		{"disk pressure halves rate", Pressure{MonitoringAvailable: true, DiskAwaitMillis: 25}, time.Date(2026, 7, 27, 8, 7, 0, 0, time.UTC), 80, 40, false},
		{"disk overload pauses", Pressure{MonitoringAvailable: true, DiskAwaitMillis: 45}, atBoundary, 80, 0, true},
		{"nats backlog pauses", Pressure{MonitoringAvailable: true, NATSBacklog: true}, atBoundary, 80, 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ApplyPressure(tt.rate, tt.p, tt.now)
			if math.Abs(got.Rate-tt.want) > 0.001 || got.Paused != tt.stop {
				t.Fatalf("decision = %+v, want rate=%v paused=%v", got, tt.want, tt.stop)
			}
		})
	}
}

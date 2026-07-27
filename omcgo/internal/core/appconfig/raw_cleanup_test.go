package appconfig

import (
	"testing"
	"time"
)

func TestRawCleanupConfigDefaultsAreSafeAndBounded(t *testing.T) {
	got := (RawCleanupConfig{}).Defaults()
	if got.Mode != RawCleanupModeShadow || got.BatchSize != 100 ||
		got.MinRate != 5 || got.MaxRate != 100 || got.RateHeadroom != 1.5 ||
		got.RecalculateInterval != 5*time.Minute || got.ObjectTimeout != 10*time.Second {
		t.Fatalf("unexpected defaults: %+v", got)
	}
}

func TestRawCleanupConfigNeverAllowsMaxBelowMin(t *testing.T) {
	got := (RawCleanupConfig{MinRate: 20, MaxRate: 10}).Defaults()
	if got.MaxRate != 20 {
		t.Fatalf("max rate = %v, want 20", got.MaxRate)
	}
}

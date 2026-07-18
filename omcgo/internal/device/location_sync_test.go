package device

import (
	"testing"
	"time"
)

func TestLookupGPSCoordinatesAcceptsStandardAlias(t *testing.T) {
	lat, lng, sourcePath, ok := LookupGPSCoordinates(map[string]string{
		"Device.DeviceInfo.SAS.FAP.GPS.LockedLatitude":  "39904200",
		"Device.DeviceInfo.SAS.FAP.GPS.LockedLongitude": "116407400",
	})
	if !ok {
		t.Fatal("expected standard GPS alias to be accepted")
	}
	if lat != 39.9042 || lng != 116.4074 {
		t.Fatalf("unexpected coordinates: got (%v, %v)", lat, lng)
	}
	if sourcePath != "Device.DeviceInfo.SAS.FAP.GPS" {
		t.Fatalf("unexpected source path: %q", sourcePath)
	}
}

func TestCompareLocationsReturnsPendingOnlyForRealDifference(t *testing.T) {
	observedAt := time.Date(2026, 7, 17, 10, 0, 0, 0, time.UTC)
	reported := &ReportedLocation{
		Latitude:   39.9042,
		Longitude:  116.4074,
		ObservedAt: observedAt,
		Version:    3,
	}
	accepted := &Location{Latitude: 39.9042, Longitude: 116.4074}

	result := CompareLocations(accepted, reported)
	if result.Status != LocationSyncInSync {
		t.Fatalf("expected in_sync, got %q", result.Status)
	}
	if result.DistanceMeters == nil || *result.DistanceMeters != 0 {
		t.Fatalf("expected zero distance, got %#v", result.DistanceMeters)
	}
}

func TestLookupGPSCoordinatesRejectsInvalidZeroPair(t *testing.T) {
	if _, _, _, ok := LookupGPSCoordinates(map[string]string{
		"Device.FAP.GPS.LockedLatitude":  "0",
		"Device.FAP.GPS.LockedLongitude": "0",
	}); ok {
		t.Fatal("expected zero coordinate pair to be rejected")
	}
	if _, ok := ParseGPSCoordinate("181000000", 180); ok {
		t.Fatal("expected longitude outside scaled range to be rejected")
	}
}

func TestParseGPSCoordinateRejectsOutOfRangeDecimalDegrees(t *testing.T) {
	for _, tt := range []struct {
		raw    string
		maxAbs float64
	}{
		{raw: "91", maxAbs: 90},
		{raw: "-91", maxAbs: 90},
		{raw: "181", maxAbs: 180},
		{raw: "-181", maxAbs: 180},
	} {
		if got, ok := ParseGPSCoordinate(tt.raw, tt.maxAbs); ok {
			t.Fatalf("expected %q to be rejected for max %v, got %v", tt.raw, tt.maxAbs, got)
		}
	}
}

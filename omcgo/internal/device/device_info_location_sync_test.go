package device

import (
	"strings"
	"testing"
)

func TestDeviceWithInfoSelectColumnsIncludeLatestLocationObservation(t *testing.T) {
	columns := strings.Join(deviceWithInfoSelectColumns(), ",")
	for _, expected := range []string{
		"dlo.latitude AS reported_latitude",
		"dlo.longitude AS reported_longitude",
		"dlo.gps_height AS reported_gps_height",
		"dlo.observed_at AS reported_observed_at",
		"dlo.version AS reported_version",
		"dlo.source_path AS reported_source_path",
	} {
		if !strings.Contains(columns, expected) {
			t.Fatalf("missing location observation select column %q", expected)
		}
	}
}

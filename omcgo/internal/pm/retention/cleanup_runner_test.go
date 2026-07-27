package retention

import "testing"

func TestCleanupTableSpecsLeaveRawFileMetadataToPreciseCleaner(t *testing.T) {
	for _, spec := range cleanupTableSpecs() {
		if spec.name == "pm_files" || spec.name == "pm_ingest_batches" {
			t.Fatalf("legacy retention must not delete exact-path metadata: %+v", spec)
		}
	}
}

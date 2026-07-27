package provider

import "testing"

func TestValidateMinIORetentionDays(t *testing.T) {
	for _, value := range []string{"1", "60", "3650"} {
		if err := validateMinIORetentionDays(value); err != nil {
			t.Fatalf("validateMinIORetentionDays(%q) returned error: %v", value, err)
		}
	}
	for _, value := range []string{"0", "3651", "not-a-number"} {
		if err := validateMinIORetentionDays(value); err == nil {
			t.Fatalf("validateMinIORetentionDays(%q) unexpectedly accepted invalid input", value)
		}
	}
}

func TestValidateMinIOCleanupMode(t *testing.T) {
	for _, value := range []string{"shadow", "fallback", "exclusive"} {
		if err := validateMinIOCleanupMode(value); err != nil {
			t.Fatalf("%s: %v", value, err)
		}
	}
	if err := validateMinIOCleanupMode("disabled"); err == nil {
		t.Fatal("invalid mode accepted")
	}
}

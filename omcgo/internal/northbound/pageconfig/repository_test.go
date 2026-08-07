package pageconfig

import "testing"

func TestNormalizeDeviceTechUsesDeviceStorageValues(t *testing.T) {
	tests := map[string]string{
		"ENB": "LTE",
		"LTE": "LTE",
		"GNB": "NR",
		"NR":  "NR",
		"GSM": "GSM",
	}
	for input, want := range tests {
		if got := normalizeDeviceTech(input); got != want {
			t.Fatalf("normalizeDeviceTech(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestDefaultInventoryCatalogContainsCompleteRadioFields(t *testing.T) {
	fields := NewDefaultCatalog().Fields(FieldFilter{Domain: DomainInventory, ObjectCode: "ENB"})
	if len(fields) != len(stationInventoryFieldSeeds) {
		t.Fatalf("ENB inventory field count = %d, want %d", len(fields), len(stationInventoryFieldSeeds))
	}
	if fields[5].OutputAlias != "Shop ID" || fields[5].SystemField != "device.site_id" {
		t.Fatalf("unexpected Shop ID field: %#v", fields[5])
	}
	if fields[37].OutputAlias != "Product Name" || fields[37].SystemField != "product.name" {
		t.Fatalf("unexpected Product Name field: %#v", fields[37])
	}

	gnb := NewDefaultCatalog().Fields(FieldFilter{Domain: DomainInventory, ObjectCode: "GNB"})
	if gnb[4].OutputAlias != "gNB Name" || gnb[8].OutputAlias != "NCI" || gnb[10].OutputAlias != "NR-ARFCN" {
		t.Fatalf("unexpected GNB aliases: %#v, %#v, %#v", gnb[4], gnb[8], gnb[10])
	}
}

package pageconfig

import "testing"

func TestDefaultFileProfilesMatchLegacyScenarioFormats(t *testing.T) {
	csvCMProfiles := map[string]bool{
		"S0007": true,
		"S0008": true,
	}
	for _, profile := range NewDefaultCatalog().FileProfiles() {
		for _, group := range profile.Groups {
			switch group.Domain {
			case DomainCM:
				want := FormatXML
				if csvCMProfiles[profile.Code] {
					want = FormatCSV
				}
				if group.Format != want {
					t.Fatalf("%s %s CM format = %s, want %s", profile.Code, group.ID, group.Format, want)
				}
			case DomainPM:
				if group.Format != FormatCSV {
					t.Fatalf("%s %s PM format = %s, want CSV", profile.Code, group.ID, group.Format)
				}
			case DomainMR:
				if group.Format != FormatXML {
					t.Fatalf("%s %s MR format = %s, want XML", profile.Code, group.ID, group.Format)
				}
			}
		}
	}
}

func TestBackfillDefaultFileProfileGroupsAlignsLegacyFormatOnce(t *testing.T) {
	defaults := map[string]FileGroup{
		"cm-daily": {
			ID:     "cm-daily",
			Domain: DomainCM,
			Format: FormatXML,
			Objects: []ScenarioObject{{
				Code:    "CP",
				Tech:    "LTE",
				Profile: legacyCMCPXMLProfile,
			}},
		},
	}
	groups := []FileGroup{{
		ID:     "cm-daily",
		Domain: DomainCM,
		Format: FormatCSV,
		Objects: []ScenarioObject{{
			Code:    "CP",
			Tech:    "LTE",
			Profile: legacyCMCPCSVProfile,
		}},
	}}

	if !backfillDefaultFileProfileGroups(groups, defaults, true, false) {
		t.Fatal("expected legacy format alignment to report a change")
	}
	if groups[0].Format != FormatXML || groups[0].Objects[0].Profile != legacyCMCPXMLProfile {
		t.Fatalf("legacy format alignment did not restore XML defaults: %#v", groups[0])
	}

	groups = []FileGroup{{
		ID:     "cm-daily",
		Domain: DomainCM,
		Format: FormatCSV,
		Objects: []ScenarioObject{{
			Code:    "CP",
			Tech:    "LTE",
			Profile: legacyCMCPCSVProfile,
		}},
	}}
	if backfillDefaultFileProfileGroups(groups, defaults, false, false) {
		t.Fatal("did not expect a second legacy format alignment after marker is present")
	}
	if groups[0].Format != FormatCSV || groups[0].Objects[0].Profile != legacyCMCPCSVProfile {
		t.Fatalf("marked profiles should preserve user-selected format: %#v", groups[0])
	}
}

func TestBackfillDefaultFileProfileGroupsAddsPMTechnologyPathForLegacyDefaults(t *testing.T) {
	defaults := map[string]FileGroup{
		"pm-pc-60m-lte": {
			ID:           "pm-pc-60m-lte",
			Domain:       DomainPM,
			Format:       FormatCSV,
			PathTemplate: pathPMTech,
		},
		"pm-pc-60m-gsm": {
			ID:           "pm-pc-60m-gsm",
			Domain:       DomainPM,
			Format:       FormatCSV,
			PathTemplate: pathPMTech,
		},
		"pm-pc-60m-gnb": {
			ID:           "pm-pc-60m-gnb",
			Domain:       DomainPM,
			Format:       FormatCSV,
			PathTemplate: pathPMTech,
		},
		"pm-custom": {
			ID:           "pm-custom",
			Domain:       DomainPM,
			Format:       FormatCSV,
			PathTemplate: pathPMTech,
		},
	}
	groups := []FileGroup{
		{ID: "pm-pc-60m-lte", Domain: DomainPM, Format: FormatCSV, PathTemplate: pathPM},
		{ID: "pm-pc-60m-gsm", Domain: DomainPM, Format: FormatCSV, PathTemplate: pathPM},
		{ID: "pm-pc-60m-gnb", Domain: DomainPM, Format: FormatCSV, PathTemplate: pathPM + "GNB/"},
		{ID: "pm-custom", Domain: DomainPM, Format: FormatCSV, PathTemplate: "/operator/PM/#DateTime#/"},
	}

	if !backfillDefaultFileProfileGroups(groups, defaults, false, true) {
		t.Fatal("expected PM technology path alignment to report a change")
	}
	for _, index := range []int{0, 1, 2} {
		if groups[index].PathTemplate != pathPMTech {
			t.Fatalf("group %s path = %s, want %s", groups[index].ID, groups[index].PathTemplate, pathPMTech)
		}
	}
	if groups[3].PathTemplate != "/operator/PM/#DateTime#/" {
		t.Fatalf("custom PM path should be preserved: %s", groups[3].PathTemplate)
	}

	if backfillDefaultFileProfileGroups(groups, defaults, false, false) {
		t.Fatal("did not expect PM technology path alignment after marker is present")
	}
}

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

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

	if !backfillDefaultFileProfileGroups(groups, defaults, true, false, false) {
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
	if backfillDefaultFileProfileGroups(groups, defaults, false, false, false) {
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

	if !backfillDefaultFileProfileGroups(groups, defaults, false, true, false) {
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

	if backfillDefaultFileProfileGroups(groups, defaults, false, false, false) {
		t.Fatal("did not expect PM technology path alignment after marker is present")
	}
}

func TestBackfillDefaultFileProfileGroupsAddsLegacyPMProfile(t *testing.T) {
	defaults := map[string]FileGroup{
		"pm-15m": {
			ID:     "pm-15m",
			Domain: DomainPM,
			Format: FormatCSV,
			Objects: []ScenarioObject{{
				Code:    "PC",
				Tech:    "LTE",
				Profile: legacyPMS0001PCProfile,
			}},
		},
	}
	groups := []FileGroup{{
		ID:     "pm-15m",
		Domain: DomainPM,
		Format: FormatCSV,
		Objects: []ScenarioObject{{
			Code: "PC",
		}},
	}}

	if !backfillDefaultFileProfileGroups(groups, defaults, false, false, false) {
		t.Fatal("expected legacy PM profile backfill to report a change")
	}
	if groups[0].Objects[0].Tech != "LTE" || groups[0].Objects[0].Profile != legacyPMS0001PCProfile {
		t.Fatalf("legacy PM profile was not restored: %#v", groups[0].Objects[0])
	}

	groups[0].Objects[0].Profile = "pm.custom"
	if backfillDefaultFileProfileGroups(groups, defaults, false, false, false) {
		t.Fatal("custom PM object profile should be preserved")
	}
	if groups[0].Objects[0].Profile != "pm.custom" {
		t.Fatalf("custom PM object profile was overwritten: %#v", groups[0].Objects[0])
	}
}

func TestBackfillDefaultFileProfileGroupsMigratesOldDefaultPMProfiles(t *testing.T) {
	defaults := map[string]FileGroup{
		"pm-15m": {
			ID:     "pm-15m",
			Domain: DomainPM,
			Format: FormatCSV,
			Objects: []ScenarioObject{
				{Code: "PE", Tech: "LTE", Profile: legacyPMS0002PEProfile},
				{Code: "PC", Tech: "LTE", Profile: legacyPMS0002PCProfile},
			},
		},
		"pm-pc-60m-gsm": {
			ID:     "pm-pc-60m-gsm",
			Domain: DomainPM,
			Format: FormatCSV,
			Objects: []ScenarioObject{{
				Code:    "PC",
				Tech:    "GSM",
				Profile: legacyPMS0013GSMPCProfile,
			}},
		},
		"pm-pc-15m-gnb": {
			ID:     "pm-pc-15m-gnb",
			Domain: DomainPM,
			Format: FormatCSV,
			Objects: []ScenarioObject{{
				Code:    "PC",
				Tech:    "GNB",
				Profile: legacyPMS0012GNBPCProfile,
			}},
		},
	}
	groups := []FileGroup{
		{
			ID:     "pm-15m",
			Domain: DomainPM,
			Format: FormatCSV,
			Objects: []ScenarioObject{
				{Code: "PE", Tech: "LTE", Profile: "pm.pe.csv.v1"},
				{Code: "PC", Tech: "LTE", Profile: "pm.pc.csv.v1"},
			},
		},
		{
			ID:     "pm-pc-60m-gsm",
			Domain: DomainPM,
			Format: FormatCSV,
			Objects: []ScenarioObject{{
				Code:    "PC",
				Tech:    "GSM",
				Profile: "pm.pc.gsm.pmresult.csv.v1",
			}},
		},
		{
			ID:     "pm-pc-15m-gnb",
			Domain: DomainPM,
			Format: FormatCSV,
			Objects: []ScenarioObject{{
				Code:    "PC",
				Tech:    "GNB",
				Profile: "pm.pc.gnb.csv.v1",
			}},
		},
	}

	if !backfillDefaultFileProfileGroups(groups, defaults, false, false, true) {
		t.Fatal("expected old default PM profile migration to report a change")
	}
	if groups[0].Objects[0].Profile != legacyPMS0002PEProfile || groups[0].Objects[1].Profile != legacyPMS0002PCProfile {
		t.Fatalf("LTE PE/PC profiles were not migrated: %#v", groups[0].Objects)
	}
	if groups[1].Objects[0].Profile != legacyPMS0013GSMPCProfile {
		t.Fatalf("GSM pmresult profile was not migrated: %#v", groups[1].Objects[0])
	}
	if groups[2].Objects[0].Profile != legacyPMS0012GNBPCProfile {
		t.Fatalf("GNB profile was not migrated: %#v", groups[2].Objects[0])
	}

	groups[0].Objects[1].Profile = "pm.custom"
	if backfillDefaultFileProfileGroups(groups[:1], defaults, false, false, true) {
		t.Fatal("custom PM profile should be preserved during legacy migration")
	}
	if groups[0].Objects[1].Profile != "pm.custom" {
		t.Fatalf("custom PM profile was overwritten: %#v", groups[0].Objects[1])
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

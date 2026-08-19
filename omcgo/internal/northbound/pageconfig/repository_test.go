package pageconfig

import (
	"testing"

	"github.com/stretchr/testify/require"
)

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
	catalog := NewDefaultCatalog()
	require.Equal(t, []string{
		"Serial Number", "Cell Status", "Online Status", "Alarms", "Cell Name",
		"IP Address", "MAC Address", "ECI", "PCI", "Earfcn", "MME Status",
		"Sync Status", "UE Count", "Last Period Time", "Product Type",
		"Hardware Version", "Software Version", "Device Group", "RF Status", "Satellites",
		"Longitude", "Latitude", "Height", "Duplex Mode", "IPsec Address", "PLMN",
		"First Online Time", "TAC", "Model Name", "Cell Active State", "Manufacturer",
		"Device Power", "OMC IP", "Installation Detailed Address",
		"Device Status", "First Period Time", "Product Name", "System Uptime",
		"Accumulated Online Time(s)", "Bandwidth", "Height(m)", "txPower", "eNB ID",
		"Cell ID", "Subframe Assignment", "Special Subframe Patterns", "Root Sequence Index",
	}, outputAliases(catalog.Fields(FieldFilter{Domain: DomainInventory, ObjectCode: "ENB"})))

	require.Equal(t, []string{
		"Online Status", "Alarms", "Serial Number", "Cell Name", "IP Address", "MAC Address",
		"NR Cell ID", "PCI", "Cell Status", "Sync Status", "UE Count", "System Uptime",
		"Last Period Time", "AMF Status", "Duplex Mode", "Product Type", "Product Name",
		"gNB ID", "BandIndicator", "Hardware Version", "Software Version", "NR ARFCN UL",
		"NR ARFCN DL", "Device Group", "RF Status", "Satellites", "Longitude", "Latitude",
		"First Period Time", "TAC", "Model Name", "Manufacturer", "Tx Power",
		"OMC IP", "Installation Detailed Address", "Device Status",
	}, outputAliases(catalog.Fields(FieldFilter{Domain: DomainInventory, ObjectCode: "GNB"})))

	require.Equal(t, []string{
		"Online Status", "Alarms", "Serial Number", "Cell Name", "IP Address", "MAC Address",
		"Cell Status", "Sync Status", "UE Count", "System Uptime", "Last Period Time",
		"Product Type", "Product Name", "Accumulated Online Time", "Ipa Unit Id",
		"Oml Remote Ip", "Oml Remote Ip Bak", "BSC Select", "LAC", "Earfcn",
		"Hardware Version", "Software Version", "Device Group", "RF Status", "Satellites",
		"Longitude", "Latitude", "Height", "First Period Time", "Model Name",
		"Cell Admin State", "Manufacturer", "Tx Power", "OMC IP",
		"Installation Detailed Address", "Device Status",
	}, outputAliases(catalog.Fields(FieldFilter{Domain: DomainInventory, ObjectCode: "GSM"})))

	require.Equal(t, []string{
		"OMC Name", "OMC IP", "Total Devices", "Online Devices", "Active Alarms",
		"Current Connected UEs", "Version",
	}, outputAliases(catalog.Fields(FieldFilter{Domain: DomainInventory, ObjectCode: "OMC"})))

	enbFields := catalog.Fields(FieldFilter{Domain: DomainInventory, ObjectCode: "ENB"})
	require.Equal(t, "inventory.device.active_alarm_count", fieldByAlias(t, enbFields, "Alarms").SystemField)
	require.NotContains(t, outputAliases(enbFields), "Shop ID")
	require.NotContains(t, outputAliases(enbFields), "Site Name")
	require.NotContains(t, outputAliases(enbFields), "KPI Report Status")

	gnbFields := catalog.Fields(FieldFilter{Domain: DomainInventory, ObjectCode: "GNB"})
	require.Equal(t, "device_info.cell_id", fieldByAlias(t, gnbFields, "NR Cell ID").SystemField)
	require.Equal(t, "inventory.gnb.gnb_id", fieldByAlias(t, gnbFields, "gNB ID").SystemField)
	require.Equal(t, "inventory.gnb.amf_status", fieldByAlias(t, gnbFields, "AMF Status").SystemField)
	require.Contains(t, fieldByAlias(t, gnbFields, "gNB ID").Source, "FAPControl.NR.RAN.Common.gNBId")
	require.NotContains(t, outputAliases(gnbFields), "MME Status")
	require.NotContains(t, outputAliases(gnbFields), "Site Name")
	gnbMMEOptionalField, ok := optionalDeviceInfoFieldFromColumn(DomainInventory, "GNB", "mme_status", "text")
	require.True(t, ok)
	require.Equal(t, "AMF Status", gnbMMEOptionalField.OutputAlias)
	require.True(t, isUnsupportedInventoryDeviceInfoField(DomainInventory, "GNB", "mme_status"))
	require.False(t, isUnsupportedInventoryDeviceInfoField(DomainInventory, "ENB", "mme_status"))

	gsmFields := catalog.Fields(FieldFilter{Domain: DomainInventory, ObjectCode: "GSM"})
	require.NotContains(t, outputAliases(gsmFields), "Site Name")

	omcFields := catalog.Fields(FieldFilter{Domain: DomainInventory, ObjectCode: "OMC"})
	require.Equal(t, "inventory.omc.total_devices", fieldByAlias(t, omcFields, "Total Devices").SystemField)
	require.Equal(t, "inventory.omc.online_devices", fieldByAlias(t, omcFields, "Online Devices").SystemField)
	require.Equal(t, "inventory.omc.active_alarms", fieldByAlias(t, omcFields, "Active Alarms").SystemField)
	require.Equal(t, "inventory.omc.current_connected_ues", fieldByAlias(t, omcFields, "Current Connected UEs").SystemField)
	require.Contains(t, fieldByAlias(t, omcFields, "OMC Name").Source, "default display name")
	require.NotContains(t, outputAliases(omcFields), "eNB online")
	require.NotContains(t, outputAliases(omcFields), "eNB active")
	require.NotContains(t, outputAliases(omcFields), "MME status")
	require.NotContains(t, outputAliases(omcFields), "UE Count")
	require.NotContains(t, outputAliases(omcFields), "Active/Standby state")
	require.NotContains(t, outputAliases(omcFields), "Hardware Model")
}

func TestDefaultInventoryProfilesCarryLegacyFieldConfig(t *testing.T) {
	profiles := NewDefaultCatalog().InventoryProfiles()
	byCode := make(map[string]InventoryProfile, len(profiles))
	for _, profile := range profiles {
		byCode[profile.Code] = profile
		require.Equal(t, defaultInventoryProfileFieldsRevision, profile.DefaultFieldsRevision)
		require.False(t, profile.FieldsCustomized)
		require.NotEmpty(t, profile.Fields)
	}

	require.Equal(t,
		outputAliases(NewDefaultCatalog().Fields(FieldFilter{Domain: DomainInventory, ObjectCode: "ENB"})),
		inventoryFieldAliases(byCode["ENB"].Fields),
	)
	require.Equal(t,
		outputAliases(NewDefaultCatalog().Fields(FieldFilter{Domain: DomainInventory, ObjectCode: "GNB"})),
		inventoryFieldAliases(byCode["GNB"].Fields),
	)
	require.NotContains(t, inventoryFieldAliases(byCode["ENB"].Fields), "Operator")
	require.NotContains(t, inventoryFieldAliases(byCode["ENB"].Fields), "Femto Vendor")
	require.NotContains(t, inventoryFieldAliases(byCode["GNB"].Fields), "gNB Name")
	require.NotContains(t, inventoryFieldAliases(byCode["OMC"].Fields), "eNB active number")
	require.NotContains(t, inventoryFieldAliases(byCode["OMC"].Fields), "eNB online")
	require.NotContains(t, inventoryFieldAliases(byCode["OMC"].Fields), "MME status")
	require.NotContains(t, inventoryFieldAliases(byCode["OMC"].Fields), "Hardware Model")
}

func TestInventoryProfileConfigMetadataDistinguishesDefaultsFromUserFields(t *testing.T) {
	fields := []InventoryFieldConfig{{
		Key:         "inventory.enb.1",
		OutputAlias: "Serial Number",
		SystemField: "device.serial_number",
		Enabled:     true,
	}}

	defaultRaw, err := marshalDefaultInventoryProfileConfig(fields)
	require.NoError(t, err)
	var defaultProfile InventoryProfile
	require.NoError(t, unmarshalInventoryProfileConfig(defaultRaw, &defaultProfile))
	require.Equal(t, defaultInventoryProfileFieldsRevision, defaultProfile.DefaultFieldsRevision)
	require.False(t, defaultProfile.FieldsCustomized)

	customRaw, err := marshalInventoryProfileConfig(fields)
	require.NoError(t, err)
	var customProfile InventoryProfile
	require.NoError(t, unmarshalInventoryProfileConfig(customRaw, &customProfile))
	require.Empty(t, customProfile.DefaultFieldsRevision)
	require.True(t, customProfile.FieldsCustomized)
}

func inventoryFieldAliases(fields []InventoryFieldConfig) []string {
	aliases := make([]string, 0, len(fields))
	for _, field := range fields {
		aliases = append(aliases, field.OutputAlias)
	}
	return aliases
}

func fieldByAlias(t *testing.T, fields []FieldDefinition, alias string) FieldDefinition {
	t.Helper()
	for _, field := range fields {
		if field.OutputAlias == alias {
			return field
		}
	}
	t.Fatalf("field %q not found", alias)
	return FieldDefinition{}
}

func TestInventoryDeviceRowUsesLegacyDisplayValues(t *testing.T) {
	row := ExportDataRow{
		"device.is_online":                       "true",
		"device.serial_number":                   "SN-FALLBACK",
		"device.site_name":                       "Site From Device",
		"device.lifecycle_state":                 "commissioned",
		"device.product_class":                   "pBS11004",
		"product.name":                           "RTS",
		"device_info.op_state":                   "1",
		"device_info.sync_status":                "DISP",
		"device_info.rf_status":                  "1",
		"device_info.mme_status":                 "partial",
		"device_info.kpi_status":                 "1",
		"device_info.run_time":                   "29437",
		"device_info.cumulative_online_duration": "22638300",
	}

	enrichInventoryDeviceRow(row)

	require.Equal(t, "Site From Device", row["device_info.device_name"])
	require.Equal(t, "ON", row["inventory.enb.online_status"])
	require.Equal(t, "On", row["inventory.gnb.online_status"])
	require.Equal(t, "Active", row["inventory.enb.cell_status"])
	require.Equal(t, "GPS Synchronizing", row["inventory.enb.sync_status"])
	require.Equal(t, "connected", row["inventory.enb.mme_status"])
	require.Equal(t, "connected", row["inventory.gnb.amf_status"])
	require.Equal(t, "normal", row["inventory.enb.kpi_status"])
	require.Equal(t, "ON", row["inventory.enb.rf_status"])
	require.Equal(t, "ON", row["inventory.enb.cell_active_state"])
	require.Equal(t, "Unblock", row["inventory.gsm.cell_admin_state"])
	require.Equal(t, "Install", row["inventory.enb.device_status"])
	require.Equal(t, "RTS", row["inventory.enb.product_name"])
	require.Equal(t, "0d 8h 10m 37s", row["inventory.enb.system_uptime"])
	require.Equal(t, "262d 0h 25m 0s", row["inventory.enb.accumulated_online_time"])
	row["device.is_online"] = "false"
	enrichInventoryDeviceRow(row)
	require.Equal(t, "OFF", row["inventory.enb.online_status"])
	require.Equal(t, "Inactive", row["inventory.enb.cell_status"])
}

func TestInventoryStatusDisplayMappingsCoverDeviceListValues(t *testing.T) {
	require.Equal(t, "connected", legacyInventoryCoreStatus("partial"))
	require.Equal(t, "connected", legacyInventoryCoreStatus("connected"))
	require.Equal(t, "disconnected", legacyInventoryCoreStatus("disconnected"))

	require.Equal(t, "GPS Synchronized", legacyInventorySyncStatus("SYNCHRONIZED"))
	require.Equal(t, "GPS Synchronized", legacyInventorySyncStatus("LOCKED"))
	require.Equal(t, "GPS Synchronizing", legacyInventorySyncStatus("DISP"))
	require.Equal(t, "Unsynchronized", legacyInventorySyncStatus("error"))
	require.Equal(t, "--", legacyInventorySyncStatus("3"))
}

func TestLegacyInventoryOMCNameFallsBackToDefaultDisplayName(t *testing.T) {
	t.Setenv("OMC_NAME", "")
	t.Setenv("OMC_OMC_NAME", "")
	t.Setenv("MR_OMC_NAME", "")

	require.Equal(t, defaultInventoryOMCName, legacyInventoryOMCName(""))
	require.Equal(t, defaultInventoryOMCName, legacyInventoryOMCName("   "))
	require.Equal(t, "Configured OMC", legacyInventoryOMCName(" Configured OMC "))
}

func TestLegacyInventoryOMCNameEnvOverridesConfiguredValue(t *testing.T) {
	t.Setenv("OMC_NAME", "Env OMC")
	t.Setenv("OMC_OMC_NAME", "")
	t.Setenv("MR_OMC_NAME", "")

	require.Equal(t, "Env OMC", legacyInventoryOMCName("Configured OMC"))
}

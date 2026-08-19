package pageconfig

import (
	"strconv"
	"strings"
)

const (
	pathCM        = "/#FTPRoot#/#Province#/#OMC-R#/CM/#DateTime#/"
	pathPM        = "/#FTPRoot#/#Province#/#OMC-R#/PM/#DateTime#/"
	pathPMTech    = "/#FTPRoot#/#Province#/#OMC-R#/PM/#Tech#/#DateTime#/"
	pathMR        = "/#FTPRoot#/#Province#/#OMC-R#/MR/#DateTime#/"
	pathLOG       = "/#FTPRoot#/LOGS/#Date#/"
	pathInventory = "/#FTPRoot#/#Province#/#OMC-R#/Inventory/#Object#/#DateTime#/"

	nameCM        = "Baicells-#Object#-#LocalHost#-#DataVersion#-#DateTime#[-#Ri#][-#FileID#]"
	namePM        = "Baicells-#Object#-#LocalHost#-#DataVersion#-#DateTime#[-#Ri#]-#DataPeriod#[-#FileID#]"
	nameMR        = "#ModuleType#-Baicells-#Object#-#LocalHost#-#eNBID#-#DateTime#[-#Ri#].xml"
	nameLOG       = "#Object#_#Date#.txt"
	nameInventory = "BaiOMC_#Object#_#DateTime#.csv"

	legacyCMCPXMLProfile   = "cm.cp.xml.v1"
	legacyCMEPXMLProfile   = "cm.ep.xml.v1"
	legacyCMCCXMLProfile   = "cm.cc.xml.v1"
	legacyCMCEXMLProfile   = "cm.ce.xml.v1"
	legacyCMCPCSVProfile   = "cm.cp.csv.v1"
	legacyCMEPCSVProfile   = "cm.ep.csv.v1"
	legacyCMCCCSVProfile   = "cm.cc.csv.v1"
	legacyCMCECSVProfile   = "cm.ce.csv.v1"
	legacyCMCOMSCSVProfile = "cm.coms.csv.v1"

	legacyS0001EPProfile        = "cm.s0001.ep.legacy"
	legacyPMS0001PCProfile      = "pm.s0001.pc.legacy"
	legacyPMS0002PEProfile      = "pm.s0002.pe.legacy"
	legacyPMS0002PCProfile      = "pm.s0002.pc.legacy"
	legacyPMS0003PCProfile      = "pm.s0003.pc.legacy"
	legacyPMS0007PCProfile      = "pm.s0007.pc.legacy"
	legacyPMS0008PCProfile      = "pm.s0008.pc.legacy"
	legacyPMS0009PCProfile      = "pm.s0009.pc.legacy"
	legacyPMS0011GSMPCProfile   = "pm.s0011.pc.gsm.legacy"
	legacyPMS0012GNBPCProfile   = "pm.s0012.pc.gnb.legacy"
	legacyPMS0013GSMPCProfile   = "pm.s0013.pc.gsm.legacy"
	legacyPMS0016LTEPCProfile   = "pm.s0016.pc.lte.legacy"
	legacyPMS0016GSMPCProfile   = "pm.s0016.pc.gsm.legacy"
	legacyCMPlmnCPXMLProfile    = "cm.cp.xml.s0003-s0015.v1"
	legacyCMHeNBCEXMLProfile    = "cm.ce.xml.henb-location.v1"
	legacyCMS0005CCXMLProfile   = "cm.cc.xml.s0005.v1"
	legacyCMS0006CPXMLProfile   = "cm.cp.xml.s0006.v1"
	legacyCMS0007COMSCSVProfile = "cm.coms.csv.s0007.v1"
)

// Catalog is the read model used by the page-config API. It is deliberately
// separate from the existing northbound push/sync data-plane models.
type Catalog struct {
	fileProfiles      []FileProfile
	inventoryProfiles []InventoryProfile
	fields            []FieldDefinition
}

func NewDefaultCatalog() *Catalog {
	return &Catalog{
		fileProfiles:      defaultFileProfiles(),
		inventoryProfiles: defaultInventoryProfiles(),
		fields:            defaultFields(),
	}
}

func (c *Catalog) Overview() Overview {
	return Overview{
		FileProfiles:      len(c.fileProfiles),
		InventoryProfiles: len(c.inventoryProfiles),
		FieldDefinitions:  len(c.fields),
		SupportedDomains:  []Domain{DomainCM, DomainPM, DomainMR, DomainLOG, DomainInventory},
		SupportedPeriods:  []Period{Period15M, Period60M, Period24H, Period7D, Period1MO},
	}
}

func (c *Catalog) FileProfiles() []FileProfile {
	out := make([]FileProfile, len(c.fileProfiles))
	copy(out, c.fileProfiles)
	return out
}

func (c *Catalog) InventoryProfiles() []InventoryProfile {
	out := make([]InventoryProfile, len(c.inventoryProfiles))
	copy(out, c.inventoryProfiles)
	return out
}

func (c *Catalog) Fields(filter FieldFilter) []FieldDefinition {
	out := make([]FieldDefinition, 0, len(c.fields))
	profileOut := make([]FieldDefinition, 0, len(c.fields))
	for _, f := range c.fields {
		if filter.Domain != "" && f.Domain != filter.Domain {
			continue
		}
		if filter.ObjectCode != "" && !strings.EqualFold(f.ObjectCode, filter.ObjectCode) {
			continue
		}
		if filter.Tech != "" && f.Tech != "" && !strings.EqualFold(f.Tech, filter.Tech) {
			continue
		}
		if filter.Profile != "" && f.Profile != "" && strings.EqualFold(f.Profile, filter.Profile) {
			profileOut = append(profileOut, f)
			continue
		}
		if f.Profile != "" {
			continue
		}
		out = append(out, f)
	}
	if filter.Profile != "" && len(profileOut) > 0 {
		return profileOut
	}
	return out
}

func (c *Catalog) HasFieldKey(domain Domain, key string) bool {
	for _, f := range c.fields {
		if f.Domain == domain && f.Key == key {
			return true
		}
	}
	return false
}

func group(id string, domain Domain, format OutputFormat, period Period, startMinute int, path, name string, objects []ScenarioObject) FileGroup {
	return FileGroup{
		ID:                 id,
		Domain:             domain,
		Format:             format,
		Period:             period,
		StartMinute:        startMinute,
		PathTemplate:       path,
		FileNameTemplate:   name,
		CompressionEnabled: true,
		CompressionFormat:  CompressionZip,
		Objects:            objects,
	}
}

func csvSeparator(group FileGroup, separator string) FileGroup {
	group.CSVSeparator = separator
	return group
}

func pipeCSV(group FileGroup) FileGroup {
	return csvSeparator(group, "|")
}

func gzipGroup(group FileGroup) FileGroup {
	group.CompressionFormat = CompressionGz
	return group
}

func obj(codes ...string) []ScenarioObject {
	out := make([]ScenarioObject, 0, len(codes))
	for _, code := range codes {
		out = append(out, ScenarioObject{Code: code})
	}
	return out
}

func legacyPMPCObject(profile string) []ScenarioObject {
	return []ScenarioObject{{Code: "PC", Tech: "LTE", Profile: profile}}
}

func legacyPMPEPCObjects(peProfile, pcProfile string) []ScenarioObject {
	return []ScenarioObject{
		{Code: "PE", Tech: "LTE", Profile: peProfile},
		{Code: "PC", Tech: "LTE", Profile: pcProfile},
	}
}

func cmLTEObjects(format OutputFormat, codes ...string) []ScenarioObject {
	out := make([]ScenarioObject, 0, len(codes))
	for _, code := range codes {
		out = append(out, ScenarioObject{Code: code, Tech: "LTE", Profile: legacyCMProfile(format, code)})
	}
	return out
}

func legacyCMProfile(format OutputFormat, code string) string {
	code = strings.ToUpper(strings.TrimSpace(code))
	switch format {
	case FormatCSV:
		switch code {
		case "CP":
			return legacyCMCPCSVProfile
		case "EP":
			return legacyCMEPCSVProfile
		case "CC":
			return legacyCMCCCSVProfile
		case "CE":
			return legacyCMCECSVProfile
		case "COMS":
			return legacyCMCOMSCSVProfile
		}
	case FormatXML:
		switch code {
		case "CP":
			return legacyCMCPXMLProfile
		case "EP":
			return legacyCMEPXMLProfile
		case "CC":
			return legacyCMCCXMLProfile
		case "CE":
			return legacyCMCEXMLProfile
		}
	}
	return ""
}

func withObjectProfile(objects []ScenarioObject, code, profile string) []ScenarioObject {
	out := make([]ScenarioObject, len(objects))
	copy(out, objects)
	for i := range out {
		if strings.EqualFold(out[i].Code, code) {
			out[i].Profile = profile
		}
	}
	return out
}

func defaultFileProfiles() []FileProfile {
	cmXML := cmLTEObjects(FormatXML, "CP", "EP", "CC", "CE")
	cmCSV := cmLTEObjects(FormatCSV, "CP", "EP", "CC", "CE")
	cmS0001 := withObjectProfile(cmXML, "EP", legacyS0001EPProfile)
	cmS0003 := withObjectProfile(cmXML, "CP", legacyCMPlmnCPXMLProfile)
	cmS0004 := withObjectProfile(cmXML, "CE", legacyCMHeNBCEXMLProfile)
	cmS0005 := withObjectProfile(withObjectProfile(cmXML, "CC", legacyCMS0005CCXMLProfile), "CE", legacyCMHeNBCEXMLProfile)
	cmS0006 := withObjectProfile(cmXML, "CP", legacyCMS0006CPXMLProfile)
	cmS0007 := append(cmLTEObjects(FormatCSV, "CP", "EP", "CC", "CE"), ScenarioObject{Code: "COMS", Tech: "LTE", Profile: legacyCMS0007COMSCSVProfile})
	cmS0008 := append(cmCSV, ScenarioObject{Code: "COMS", Tech: "LTE", Profile: legacyCMCOMSCSVProfile})
	mr := obj("MRO", "MRE", "MRS")
	return []FileProfile{
		fileProfile("S0001", "CM + PM PC + MR standard 15M", "标准场景", "Standard", []string{"baseline"}, []FileGroup{
			group("cm-daily", DomainCM, FormatXML, Period24H, 1, pathCM, nameCM, cmS0001),
			group("pm-15m", DomainPM, FormatCSV, Period15M, 5, pathPMTech, namePM, legacyPMPCObject(legacyPMS0001PCProfile)),
			group("mr-15m", DomainMR, FormatXML, Period15M, 0, pathMR, nameMR, mr),
		}),
		fileProfile("S0002", "CM + PM PE/PC + MR", "电信场景", "Telecom", []string{"PE/PC", "csv |"}, []FileGroup{
			group("cm-daily", DomainCM, FormatXML, Period24H, 1, pathCM, nameCM, cmXML),
			pipeCSV(group("pm-15m", DomainPM, FormatCSV, Period15M, 5, pathPMTech, namePM, legacyPMPEPCObjects(legacyPMS0002PEProfile, legacyPMS0002PCProfile))),
			group("mr-15m", DomainMR, FormatXML, Period15M, 0, pathMR, nameMR, mr),
		}),
		fileProfile("S0003", "SH Telecom", "上海电信", "SH-Tele", []string{"PE/PC", "csv |"}, []FileGroup{
			group("cm-daily", DomainCM, FormatXML, Period24H, 1, pathCM, nameCM, cmS0003),
			pipeCSV(group("pm-15m", DomainPM, FormatCSV, Period15M, 5, pathPMTech, namePM, legacyPMPEPCObjects(legacyPMS0002PEProfile, legacyPMS0003PCProfile))),
			group("mr-15m", DomainMR, FormatXML, Period15M, 0, pathMR, nameMR, mr),
		}),
		fileProfile("S0004", "Shaanxi Telecom", "陕西电信", "SN-Tele", []string{"PE/PC", "alarm cn", "csv |"}, []FileGroup{
			group("cm-daily", DomainCM, FormatXML, Period24H, 1, pathCM, nameCM, cmS0004),
			pipeCSV(group("pm-15m", DomainPM, FormatCSV, Period15M, 5, pathPMTech, namePM, legacyPMPEPCObjects(legacyPMS0002PEProfile, legacyPMS0002PCProfile))),
			group("mr-15m", DomainMR, FormatXML, Period15M, 0, pathMR, nameMR, mr),
		}),
		fileProfile("S0005", "Jiangsu Telecom delayed PM + custom log", "江苏电信", "JS-Tele", []string{"PE/PC", "log type1", "csv |"}, []FileGroup{
			group("cm-daily", DomainCM, FormatXML, Period24H, 1, pathCM, nameCM, cmS0005),
			pipeCSV(group("pm-15m-delayed", DomainPM, FormatCSV, Period15M, 14, pathPMTech, namePM, legacyPMPEPCObjects(legacyPMS0002PEProfile, legacyPMS0002PCProfile))),
			group("mr-15m", DomainMR, FormatXML, Period15M, 0, pathMR, nameMR, mr),
			gzipGroup(group("log-custom-daily", DomainLOG, FormatTXT, Period24H, 5, pathLOG, nameLOG, obj("login", "operation"))),
		}),
		fileProfile("S0006", "CM/PM without MR", "泰国 True", "Thai-True", []string{"no MR"}, []FileGroup{
			group("cm-daily", DomainCM, FormatXML, Period24H, 1, pathCM, nameCM, cmS0006),
			group("pm-15m", DomainPM, FormatCSV, Period15M, 5, pathPMTech, namePM, legacyPMPCObject(legacyPMS0001PCProfile)),
		}),
		fileProfile("S0007", "CM CSV + COMS + PM 60M", "泰国 AIS", "Thai-AIS", []string{"CM CSV", "COMS", "PM 60M"}, []FileGroup{
			group("cm-daily-csv", DomainCM, FormatCSV, Period24H, 0, pathCM, nameCM, cmS0007),
			group("pm-60m", DomainPM, FormatCSV, Period60M, 20, pathPMTech, namePM, legacyPMPCObject(legacyPMS0007PCProfile)),
		}),
		fileProfile("S0008", "CM CSV + COMS + fixed log", "V1 场景", "V1", []string{"CM CSV", "COMS", "log type2"}, []FileGroup{
			group("cm-daily-csv", DomainCM, FormatCSV, Period24H, 1, pathCM, nameCM, cmS0008),
			group("pm-15m", DomainPM, FormatCSV, Period15M, 5, pathPMTech, namePM, legacyPMPCObject(legacyPMS0008PCProfile)),
			group("mr-15m", DomainMR, FormatXML, Period15M, 0, pathMR, nameMR, mr),
			gzipGroup(group("log-fixed-daily", DomainLOG, FormatCSV, Period24H, 5, pathLOG, "#Object#_#PeriodStartTime#-24H.csv", obj("login_fix", "operation_fix"))),
		}),
		fileProfile("S0009", "PC 60M + MR", "老挝电信", "Laos-Tele", []string{"PM 60M"}, []FileGroup{
			group("cm-daily", DomainCM, FormatXML, Period24H, 1, pathCM, nameCM, cmXML),
			group("pm-pc-60m", DomainPM, FormatCSV, Period60M, 0, pathPMTech, namePM, legacyPMPCObject(legacyPMS0009PCProfile)),
			group("mr-15m", DomainMR, FormatXML, Period15M, 0, pathMR, nameMR, mr),
		}),
		fileProfile("S0010", "Object directory output", "上海联通", "SHUcom", []string{"object dirs", "socket"}, []FileGroup{
			group("cm-daily-by-object", DomainCM, FormatXML, Period24H, 1, pathCM+"#Object#/", nameCM, cmXML),
			group("pm-15m-by-object", DomainPM, FormatCSV, Period15M, 5, pathPMTech+"#Object#/", namePM, legacyPMPCObject(legacyPMS0001PCProfile)),
			group("mr-15m-by-object", DomainMR, FormatXML, Period15M, 0, pathMR+"#Object#/", nameMR, mr),
		}),
		fileProfile("S0011", "ISAT MTN standard", "ISAT MTN", "ISAT-NBI-MTN", []string{"standard"}, []FileGroup{
			group("cm-daily", DomainCM, FormatXML, Period24H, 1, pathCM, nameCM, cmXML),
			group("pm-15m", DomainPM, FormatCSV, Period15M, 5, pathPMTech, namePM, []ScenarioObject{{Code: "PC", Tech: "GSM", Profile: legacyPMS0011GSMPCProfile}}),
			group("mr-15m", DomainMR, FormatXML, Period15M, 0, pathMR, nameMR, mr),
		}),
		fileProfile("S0012", "ENB + GNB dual technology", "陕西移动", "Shaanxi Mobile", []string{"ENB/GNB"}, []FileGroup{
			group("cm-daily-lte", DomainCM, FormatXML, Period24H, 1, pathCM, nameCM, cmXML),
			group("cm-daily-gnb", DomainCM, FormatXML, Period24H, 3, pathCM+"GNB/", nameCM, []ScenarioObject{{Code: "CP", Tech: "GNB"}, {Code: "EP", Tech: "GNB"}, {Code: "CC", Tech: "GNB"}, {Code: "CE", Tech: "GNB"}}),
			group("pm-15m-lte", DomainPM, FormatCSV, Period15M, 5, pathPMTech, namePM, legacyPMPCObject(legacyPMS0001PCProfile)),
			group("pm-pc-15m-gnb", DomainPM, FormatCSV, Period15M, 8, pathPMTech, namePM, []ScenarioObject{{Code: "PC", Tech: "GNB", Profile: legacyPMS0012GNBPCProfile}}),
			group("mr-15m", DomainMR, FormatXML, Period15M, 0, pathMR, nameMR, mr),
		}),
		fileProfile("S0013", "pmresult 60M multi-technology", "ZED 场景", "ZED", []string{"ENB/GSM/GNB", "PM 60M", "pmresult"}, []FileGroup{
			group("cm-daily", DomainCM, FormatXML, Period24H, 1, pathCM, nameCM, cmXML),
			group("pm-pc-60m-lte", DomainPM, FormatCSV, Period60M, 25, pathPMTech, "pmresult_152XXX_#DataPeriod#_#PeriodStartTime#_#PeriodEndTime#[-#FileID#]", legacyPMPCObject(legacyPMS0001PCProfile)),
			group("pm-pc-60m-gsm", DomainPM, FormatCSV, Period60M, 20, pathPMTech, "pmresult_#LocalHost#_#DataPeriod#_#PeriodStartTime#_#PeriodEndTime#[-#FileID#]", []ScenarioObject{{Code: "PC", Tech: "GSM", Profile: legacyPMS0013GSMPCProfile}}),
			group("pm-pc-60m-gnb", DomainPM, FormatCSV, Period60M, 30, pathPMTech, "pmresult_XXXXXX_#DataPeriod#_#PeriodStartTime#_#PeriodEndTime#[-#FileID#]", []ScenarioObject{{Code: "PC", Tech: "GNB", Profile: legacyPMS0012GNBPCProfile}}),
			group("mr-15m", DomainMR, FormatXML, Period15M, 0, pathMR, nameMR, mr),
		}),
		fileProfile("S0014", "Indonesia Telkomsel standard", "印尼 Telkomsel", "YinNi_Telkomsel_MNO", []string{"standard"}, []FileGroup{
			group("cm-daily", DomainCM, FormatXML, Period24H, 1, pathCM, nameCM, cmXML),
			group("pm-15m", DomainPM, FormatCSV, Period15M, 5, pathPMTech, namePM, legacyPMPCObject(legacyPMS0001PCProfile)),
			group("mr-15m", DomainMR, FormatXML, Period15M, 0, pathMR, nameMR, mr),
		}),
		fileProfile("S0015", "Philippines DITO", "菲律宾 DITO", "FeiLvBin-DITO", []string{"PE/PC", "csv |"}, []FileGroup{
			group("cm-daily", DomainCM, FormatXML, Period24H, 1, pathCM, nameCM, cmS0003),
			pipeCSV(group("pm-15m", DomainPM, FormatCSV, Period15M, 5, pathPMTech, namePM, legacyPMPEPCObjects(legacyPMS0002PEProfile, legacyPMS0003PCProfile))),
			group("mr-15m", DomainMR, FormatXML, Period15M, 0, pathMR, nameMR, mr),
		}),
		fileProfile("S0016", "ENB + GSM PM", "MTN 场景", "MTN", []string{"ENB/GSM"}, []FileGroup{
			group("cm-daily", DomainCM, FormatXML, Period24H, 1, pathCM, nameCM, cmXML),
			group("pm-15m-lte", DomainPM, FormatCSV, Period15M, 5, pathPMTech, namePM, legacyPMPCObject(legacyPMS0016LTEPCProfile)),
			group("pm-pc-15m-gsm", DomainPM, FormatCSV, Period15M, 5, pathPMTech, "pmresult_#LocalHost#_#DataPeriod#_#PeriodStartTime#_#PeriodEndTime#[-#FileID#]", []ScenarioObject{{Code: "PC", Tech: "GSM", Profile: legacyPMS0016GSMPCProfile}}),
			group("mr-15m", DomainMR, FormatXML, Period15M, 0, pathMR, nameMR, mr),
		}),
		fileProfile("S0017", "Heilongjiang standard", "黑龙江场景", "HLongjianng", []string{"standard"}, []FileGroup{
			group("cm-daily", DomainCM, FormatXML, Period24H, 1, pathCM, nameCM, cmXML),
			group("pm-15m", DomainPM, FormatCSV, Period15M, 5, pathPMTech, namePM, legacyPMPCObject(legacyPMS0001PCProfile)),
			group("mr-15m", DomainMR, FormatXML, Period15M, 0, pathMR, nameMR, mr),
		}),
	}
}

func fileProfile(code, name, scenarioName, scenarioNameEn string, flags []string, groups []FileGroup) FileProfile {
	return FileProfile{
		ID:             strings.ToLower(code),
		Code:           code,
		Name:           name,
		Vendor:         "Baicells",
		ScenarioName:   scenarioName,
		ScenarioNameEn: scenarioNameEn,
		Description:    name,
		Flags:          flags,
		Enabled:        false,
		Status:         StatusTerminated,
		Groups:         groups,
	}
}

func defaultInventoryProfiles() []InventoryProfile {
	return []InventoryProfile{
		inventoryProfile("ENB", "eNB Inventory", "eNB", "LTE"),
		inventoryProfile("GNB", "gNB Inventory", "gNB", "GNB"),
		inventoryProfile("GSM", "GSM Inventory", "GSM", "GSM"),
		inventoryProfile("OMC", "OMC Inventory", "OMC", "OMC"),
	}
}

func inventoryProfile(code, name, objectCode, tech string) InventoryProfile {
	return InventoryProfile{
		ID:                 strings.ToLower(code),
		Code:               code,
		Name:               name,
		ObjectCode:         objectCode,
		Tech:               tech,
		Period:             Period24H,
		StartMinute:        5,
		PathTemplate:       pathInventory,
		FileNameTemplate:   nameInventory,
		CompressionEnabled: true,
		CompressionFormat:  CompressionZip,
		Enabled:            false,
		Status:             StatusTerminated,
	}
}

type inventoryFieldSeed struct {
	outputAlias string
	systemField string
	source      string
	dataType    string
	renderer    string
	cnName      string
}

var stationInventoryFieldSeeds = []inventoryFieldSeed{
	{"Serial Number", "device.serial_number", "devices.serial_number", "string", "quote", "Device serial number"},
	{"Cell Status", "device_info.op_state", "device_info.op_state", "string", "quote", "Cell status"},
	{"Online Status", "device.is_online", "devices.is_online", "bool", "enum", "Online status"},
	{"Alarms", "device_info.active_alarm_count", "device_info.active_alarm_count", "number", "number", "Active alarm count"},
	{"Cell Name", "device_info.device_name", "device_info.device_name / devices.site_name", "string", "quote", "Cell name"},
	{"Shop ID", "device.site_id", "devices.site_id", "string", "quote", "Shop ID"},
	{"IP Address", "device.ip_address", "devices.ip_address", "string", "quote", "IP address"},
	{"MAC Address", "device_info.mac", "device_info.mac", "string", "quote", "MAC address"},
	{"ECI", "device_info.eci", "device_info.eci", "string", "preserve text", "ECI"},
	{"PCI", "device_info.pci", "device_info.pci", "string", "preserve text", "PCI"},
	{"Earfcn", "device_info.freq_point", "device_info.freq_point", "number", "preserve text", "EARFCN"},
	{"MME Status", "device_info.mme_status", "device_info.mme_status", "string", "quote", "MME status"},
	{"KPI Report Status", "device_info.kpi_status", "device_info.kpi_status", "string", "quote", "KPI report status"},
	{"Sync Status", "device_info.sync_status", "device_info.sync_status", "string", "quote", "Sync status"},
	{"UE Count", "device_info.ue_count", "device_info.ue_count", "number", "number", "UE count"},
	{"Last Period Time", "device.last_inform_at", "devices.last_inform_at", "datetime", "preserve text", "Last period time"},
	{"Product Type", "device.product_class", "devices.product_class", "string", "quote", "Product class"},
	{"Hardware Version", "device_info.hardware_version", "device_info.hardware_version", "string", "quote", "Hardware version"},
	{"Software Version", "device.firmware_version", "devices.firmware_version", "string", "quote", "Software version"},
	{"Device Group", "device_groups.name", "device_group_members / device_groups", "string", "quote", "Device group"},
	{"RF Status", "device_info.rf_status", "device_info.rf_status", "string", "quote", "RF status"},
	{"Satellites", "device_info.gps_satellites", "device_info.gps_satellites", "number", "number", "GPS satellites"},
	{"Longitude", "device.longitude", "devices.longitude", "number", "number", "Longitude"},
	{"Latitude", "device.latitude", "devices.latitude", "number", "number", "Latitude"},
	{"Height", "device_info.gps_height", "device_info.gps_height", "number", "number", "GPS height"},
	{"Duplex Mode", "device_info.network_model", "device_info.network_model", "string", "quote", "Duplex mode"},
	{"IPsec Address", "device_info.ipsec_addr", "device_info.ipsec_addr", "string", "quote", "IPsec address"},
	{"PLMN", "device_info.plmn", "device_info.plmn", "string", "preserve text", "PLMN"},
	{"First Online Time", "device_info.first_online_time", "device_info.first_online_time", "datetime", "preserve text", "First online time"},
	{"AMF Status", "device_info.mme_status", "device_info.mme_status as amf_status", "string", "quote", "AMF status"},
	{"TAC", "device_info.tac", "device_info.tac", "string", "quote", "TAC"},
	{"Model Name", "device.model_name", "devices.model_name", "string", "quote", "Model name"},
	{"Manufacturer", "device.manufacturer", "devices.manufacturer", "string", "quote", "Manufacturer"},
	{"Site Name", "device.site_name", "devices.site_name / device_info.device_name", "string", "quote", "Site name"},
	{"Installation Detailed Address", "device_info.address", "device_info.address", "string", "quote", "Installation address"},
	{"Device Status", "device.lifecycle_state", "devices.lifecycle_state + devices.is_online", "string", "enum", "Device status"},
	{"First Period Time", "device_info.first_online_time", "device_info.first_online_time", "datetime", "preserve text", "First period time"},
	{"Product Name", "product.name", "products.product_name / devices.product_class", "string", "quote", "Product name"},
	{"System Uptime", "device_info.run_time", "device_info.run_time", "number", "number", "System uptime"},
	{"Accumulated Online Time(s)", "device_info.cumulative_online_duration", "device_info.cumulative_online_duration", "number", "number", "Accumulated online time"},
	{"Bandwidth", "device_info.bandwidth", "device_info.bandwidth", "number", "number", "Bandwidth"},
	{"Height(m)", "device_info.gps_height", "device_info.gps_height", "number", "number", "GPS height"},
	{"eNB ID", "device_info.enb_id", "device_info.enb_id", "string", "quote", "eNB ID"},
	{"Cell ID", "device_info.cell_id", "device_info.cell_id", "string", "quote", "Cell ID"},
	{"Subframe Assignment", "device_info.subframe_assignment", "device_info.subframe_assignment", "string", "quote", "Subframe assignment"},
	{"Special Subframe Patterns", "device_info.special_subframe", "device_info.special_subframe", "string", "quote", "Special subframe patterns"},
}

var deviceInfoOptionalFieldSeeds = []inventoryFieldSeed{
	{"Device Info Device ID", "device_info.device_id", "device_info.device_id", "string", "quote", "Device info device ID"},
	{"Device Name", "device_info.device_name", "device_info.device_name", "string", "quote", "Device name"},
	{"Address", "device_info.address", "device_info.address", "string", "quote", "Address"},
	{"Remark", "device_info.remark", "device_info.remark", "string", "quote", "Remark"},
	{"Project Status", "device_info.project_status", "device_info.project_status", "string", "quote", "Project status"},
	{"Height", "device_info.height", "device_info.height", "number", "number", "Height"},
	{"ECI", "device_info.eci", "device_info.eci", "string", "preserve text", "ECI"},
	{"PCI", "device_info.pci", "device_info.pci", "string", "preserve text", "PCI"},
	{"Cell ID", "device_info.cell_id", "device_info.cell_id", "string", "quote", "Cell ID"},
	{"Frequency Point", "device_info.freq_point", "device_info.freq_point", "number", "preserve text", "Frequency point"},
	{"Bandwidth", "device_info.bandwidth", "device_info.bandwidth", "number", "number", "Bandwidth"},
	{"Transmit Power", "device_info.transmit_power", "device_info.transmit_power", "number", "number", "Transmit power"},
	{"PLMN", "device_info.plmn", "device_info.plmn", "string", "preserve text", "PLMN"},
	{"RF Status", "device_info.rf_status", "device_info.rf_status", "string", "enum", "RF status"},
	{"Cell Status", "device_info.cell_status", "device_info.cell_status", "string", "enum", "Cell status"},
	{"MME Status", "device_info.mme_status", "device_info.mme_status", "string", "enum", "MME status"},
	{"Sync Status", "device_info.sync_status", "device_info.sync_status", "string", "enum", "Sync status"},
	{"KPI Report Status", "device_info.kpi_status", "device_info.kpi_status", "string", "enum", "KPI report status"},
	{"Number of Cells", "device_info.num_of_cells", "device_info.num_of_cells", "number", "number", "Number of cells"},
	{"GPS Status", "device_info.gps_status", "device_info.gps_status", "string", "enum", "GPS status"},
	{"Alarm Severity", "device_info.alarm_severity", "device_info.alarm_severity", "string", "enum", "Alarm severity"},
	{"License Status", "device_info.license_status", "device_info.license_status", "string", "enum", "License status"},
	{"MAC Address", "device_info.mac", "device_info.mac", "string", "quote", "MAC address"},
	{"Hardware Version", "device_info.hardware_version", "device_info.hardware_version", "string", "quote", "Hardware version"},
	{"First Online Time", "device_info.first_online_time", "device_info.first_online_time", "datetime", "yyyy-MM-dd HH:mm:ss", "First online time"},
	{"Last Online Time", "device_info.last_online_time", "device_info.last_online_time", "datetime", "yyyy-MM-dd HH:mm:ss", "Last online time"},
	{"Last Offline Time", "device_info.last_offline_time", "device_info.last_offline_time", "datetime", "yyyy-MM-dd HH:mm:ss", "Last offline time"},
	{"Run Time", "device_info.run_time", "device_info.run_time", "number", "number", "Run time"},
	{"Creator", "device_info.creator", "device_info.creator", "string", "quote", "Creator"},
	{"Updater", "device_info.updater", "device_info.updater", "string", "quote", "Updater"},
	{"Created At", "device_info.created_at", "device_info.created_at", "datetime", "yyyy-MM-dd HH:mm:ss", "Created at"},
	{"Updated At", "device_info.updated_at", "device_info.updated_at", "datetime", "yyyy-MM-dd HH:mm:ss", "Updated at"},
	{"TAC", "device_info.tac", "device_info.tac", "string", "quote", "TAC"},
	{"Band", "device_info.band", "device_info.band", "string", "quote", "Band"},
	{"UL EARFCN", "device_info.ul_earfcn", "device_info.ul_earfcn", "number", "preserve text", "UL EARFCN"},
	{"Subframe Assignment", "device_info.subframe_assignment", "device_info.subframe_assignment", "string", "quote", "Subframe assignment"},
	{"Special Subframe", "device_info.special_subframe", "device_info.special_subframe", "string", "quote", "Special subframe"},
	{"Root Index", "device_info.root_index", "device_info.root_index", "number", "preserve text", "Root index"},
	{"GPS Satellites", "device_info.gps_satellites", "device_info.gps_satellites", "number", "number", "GPS satellites"},
	{"GPS Height", "device_info.gps_height", "device_info.gps_height", "number", "number", "GPS height"},
	{"Lock Status", "device_info.lock_status", "device_info.lock_status", "string", "enum", "Lock status"},
	{"eNB ID", "device_info.enb_id", "device_info.enb_id", "string", "quote", "eNB ID"},
	{"Network Model", "device_info.network_model", "device_info.network_model", "string", "enum", "Network model"},
	{"LAC", "device_info.lac", "device_info.lac", "string", "quote", "LAC"},
	{"Cumulative Online Duration", "device_info.cumulative_online_duration", "device_info.cumulative_online_duration", "number", "number", "Cumulative online duration"},
	{"Operation State", "device_info.op_state", "device_info.op_state", "string", "enum", "Operation state"},
	{"Admin State", "device_info.admin_state", "device_info.admin_state", "string", "enum", "Admin state"},
	{"IPsec Address", "device_info.ipsec_addr", "device_info.ipsec_addr", "string", "quote", "IPsec address"},
	{"BSC Select", "device_info.bsc_select", "device_info.bsc_select", "string", "enum", "BSC select"},
	{"OML Remote IP", "device_info.oml_remote_ip", "device_info.oml_remote_ip", "string", "quote", "OML remote IP"},
	{"OML Remote IP Backup", "device_info.oml_remote_ip_bak", "device_info.oml_remote_ip_bak", "string", "quote", "OML remote IP backup"},
	{"IPA Unit ID", "device_info.ipa_unit_id", "device_info.ipa_unit_id", "string", "quote", "IPA unit ID"},
	{"UE Count", "device_info.ue_count", "device_info.ue_count", "number", "number", "UE count"},
	{"Active Alarm Count", "device_info.active_alarm_count", "device_info.active_alarm_count", "number", "number", "Active alarm count"},
	{"Name Sync Pending", "device_info.name_sync_pending", "device_info.name_sync_pending", "bool", "enum", "Name sync pending"},
	{"LMT Device Name", "device_info.lmt_device_name", "device_info.lmt_device_name", "string", "quote", "LMT device name"},
	{"Highest Alarm Severity", "device_info.highest_alarm_severity", "device_info.highest_alarm_severity", "number", "number", "Highest alarm severity"},
	{"Highest Severity Alarm Count", "device_info.highest_severity_alarm_count", "device_info.highest_severity_alarm_count", "number", "number", "Highest severity alarm count"},
}

var inventoryAliasesByObject = map[string]map[string]string{
	"GNB": {
		"Cell Name": "gNB Name", "ECI": "NCI", "Earfcn": "NR-ARFCN", "MME Status": "AMF Status",
		"eNB ID": "gNB ID", "Subframe Assignment": "Slot Assignment", "Special Subframe Patterns": "Special Slot Patterns",
	},
	"GSM": {
		"Cell Name": "BTS Name", "ECI": "CGI", "PCI": "BSIC", "Earfcn": "BCCH ARFCN", "MME Status": "BSC Status",
		"eNB ID": "BTS ID", "Subframe Assignment": "Channel Assignment", "Special Subframe Patterns": "Channel Pattern",
	},
}

func inventoryField(objectCode string, index int, seed inventoryFieldSeed) FieldDefinition {
	outputAlias := seed.outputAlias
	if aliases := inventoryAliasesByObject[objectCode]; aliases != nil {
		if alias, ok := aliases[outputAlias]; ok {
			outputAlias = alias
		}
	}
	definition := field(DomainInventory, objectCode, outputAlias, seed.systemField, seed.source, seed.dataType, seed.renderer, seed.cnName)
	definition.Key = strings.ToLower(string(DomainInventory) + "." + objectCode + "." + strconv.Itoa(index+1))
	return definition
}

func optionalDeviceInfoFieldByKey(domain Domain, objectCode, key string) (FieldDefinition, bool) {
	if !supportsOptionalDeviceInfoFields(domain, objectCode) {
		return FieldDefinition{}, false
	}
	prefix := strings.ToLower(string(domain) + "." + objectCode + ".device_info.")
	normalizedKey := strings.ToLower(strings.TrimSpace(key))
	if !strings.HasPrefix(normalizedKey, prefix) {
		return FieldDefinition{}, false
	}
	column := strings.TrimPrefix(normalizedKey, prefix)
	return optionalDeviceInfoFieldFromColumn(domain, objectCode, column, "")
}

func optionalDeviceInfoFieldFromColumn(domain Domain, objectCode, column, dbType string) (FieldDefinition, bool) {
	if !supportsOptionalDeviceInfoFields(domain, objectCode) {
		return FieldDefinition{}, false
	}
	column = strings.TrimSpace(strings.ToLower(column))
	if column == "" {
		return FieldDefinition{}, false
	}
	for _, seed := range deviceInfoOptionalFieldSeeds {
		if strings.EqualFold(seed.systemField, "device_info."+column) {
			return field(domain, objectCode, seed.outputAlias, seed.systemField, seed.source, seed.dataType, seed.renderer, seed.cnName), true
		}
	}
	dataType, renderer := dataTypeForDeviceInfoColumn(dbType)
	outputAlias := titleFromColumn(column)
	return field(domain, objectCode, outputAlias, "device_info."+column, "device_info."+column, dataType, renderer, outputAlias), true
}

func dataTypeForDeviceInfoColumn(dbType string) (string, string) {
	normalized := strings.ToLower(dbType)
	switch {
	case strings.Contains(normalized, "timestamp"), strings.Contains(normalized, "date"), strings.Contains(normalized, "time"):
		return "datetime", "yyyy-MM-dd HH:mm:ss"
	case strings.Contains(normalized, "bool"):
		return "bool", "enum"
	case strings.Contains(normalized, "int"), strings.Contains(normalized, "numeric"), strings.Contains(normalized, "decimal"),
		strings.Contains(normalized, "real"), strings.Contains(normalized, "double"):
		return "number", "number"
	default:
		return "string", "quote"
	}
}

func titleFromColumn(column string) string {
	parts := strings.Fields(strings.ReplaceAll(column, "_", " "))
	for i, part := range parts {
		if len(part) == 0 {
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, " ")
}

func supportsOptionalDeviceInfoFields(domain Domain, objectCode string) bool {
	switch domain {
	case DomainCM:
		for _, code := range []string{"CP", "EP", "CC", "CE", "COMS"} {
			if strings.EqualFold(objectCode, code) {
				return true
			}
		}
	case DomainInventory:
		for _, code := range []string{"ENB", "GNB", "GSM"} {
			if strings.EqualFold(objectCode, code) {
				return true
			}
		}
	}
	return false
}

func defaultFields() []FieldDefinition {
	fields := []FieldDefinition{
		field(DomainCM, "CP", "Serial Number", "device.serial_number", "devices.serial_number", "string", "quote", "Device serial number"),
		field(DomainCM, "CP", "dn", "device.serial_number", "devices.serial_number", "string", "quote", "DN / device serial number"),
		field(DomainCM, "CP", "related_enb_dn", "device.serial_number", "devices.serial_number", "string", "quote", "Related eNB DN"),
		field(DomainCM, "CP", "related_enb_id", "device_info.enb_id", "device_info.enb_id", "string", "quote", "Related eNB ID"),
		field(DomainCM, "CP", "related_enb_userlabel", "device.site_name", "devices.site_name / device_info.device_name", "string", "quote", "Related eNB label"),
		field(DomainCM, "CP", "cel_id", "device_info.cell_id", "device_info.cell_id", "string", "quote", "Cell ID"),
		field(DomainCM, "CP", "cel_userlabel", "device_info.device_name", "device_info.device_name / devices.site_name", "string", "quote", "Cell label"),
		field(DomainCM, "CP", "referenceSignalPower", "device_info.transmit_power", "device_info.transmit_power", "number", "number", "Reference signal power"),
		field(DomainCM, "CP", "Manufacturer", "device.manufacturer", "devices.manufacturer", "string", "quote", "Manufacturer"),
		field(DomainCM, "CP", "Model Name", "device.model_name", "devices.model_name", "string", "quote", "Model name"),
		field(DomainCM, "CP", "Product Type", "device.product_class", "devices.product_class", "string", "quote", "Product class"),
		field(DomainCM, "CP", "Hardware Version", "device_info.hardware_version", "device_info.hardware_version", "string", "quote", "Hardware version"),
		field(DomainCM, "CP", "Software Version", "device.firmware_version", "devices.firmware_version", "string", "quote", "Firmware version"),
		field(DomainCM, "CP", "Device Name", "device_info.device_name", "device_info.device_name", "string", "quote", "Device name"),
		field(DomainCM, "CP", "Site Name", "device.site_name", "devices.site_name", "string", "quote", "Site name"),
		field(DomainCM, "CP", "First Online Time", "device_info.first_online_time", "device_info.first_online_time", "datetime", "yyyy-MM-dd HH:mm:ss", "First online time"),
		field(DomainCM, "CP", "Last Inform", "device.last_inform_at", "devices.last_inform_at", "datetime", "yyyy-MM-dd HH:mm:ss", "Last inform time"),
		field(DomainCM, "CP", "Device Alias", "device.site_name", "devices.site_name / device_info.device_name", "string", "quote", "Device alias"),
		field(DomainCM, "CP", "Management Status", "device.lifecycle_state", "devices.lifecycle_state", "string", "enum", "Management status"),
		field(DomainCM, "CP", "Create Time", "device.created_at", "devices.created_at", "datetime", "yyyy-MM-dd HH:mm:ss", "Create time"),
		field(DomainCM, "EP", "IP Address", "device.ip_address", "devices.ip_address", "string", "quote", "IP address"),
		field(DomainCM, "EP", "dn", "device.serial_number", "devices.serial_number", "string", "quote", "DN / device serial number"),
		field(DomainCM, "EP", "enb_id", "device_info.enb_id", "device_info.enb_id", "string", "quote", "eNB ID"),
		field(DomainCM, "EP", "enb_userlabel", "device.site_name", "devices.site_name / device_info.device_name", "string", "quote", "eNB label"),
		field(DomainCM, "EP", "MAC Address", "device_info.mac", "device_info.mac", "string", "quote", "MAC address"),
		field(DomainCM, "EP", "PLMN", "device_info.plmn", "device_info.plmn", "string", "preserve text", "PLMN"),
		field(DomainCM, "EP", "Sync Status", "device_info.sync_status", "device_info.sync_status", "string", "enum", "Sync status"),
		field(DomainCM, "EP", "Longitude", "device.longitude", "devices.longitude", "number", "number", "Longitude"),
		field(DomainCM, "EP", "Latitude", "device.latitude", "devices.latitude", "number", "number", "Latitude"),
		field(DomainCM, "EP", "Satellites", "device_info.gps_satellites", "device_info.gps_satellites", "number", "number", "GPS satellites"),
		field(DomainCM, "EP", "Height(m)", "device_info.gps_height", "device_info.gps_height", "number", "number", "GPS height"),
		field(DomainCM, "EP", "TAC", "device_info.tac", "device_info.tac", "string", "quote", "TAC"),
		field(DomainCM, "EP", "LAC", "device_info.lac", "device_info.lac", "string", "quote", "LAC"),
		field(DomainCM, "EP", "MME Status", "device_info.mme_status", "device_info.mme_status", "string", "enum", "MME status"),
		field(DomainCM, "EP", "AMF Status", "device_info.mme_status", "device_info.mme_status as amf_status", "string", "enum", "AMF status"),
		field(DomainCM, "EP", "IPsec Address", "device_info.ipsec_addr", "device_info.ipsec_addr", "string", "quote", "IPsec address"),
		field(DomainCM, "CC", "eNB ID", "device_info.enb_id", "device_info.enb_id", "string", "quote", "eNB ID"),
		field(DomainCM, "CC", "dn", "device_info.eci", "device_info.eci", "string", "preserve text", "Cell DN / ECI"),
		field(DomainCM, "CC", "related_enb_dn", "device.serial_number", "devices.serial_number", "string", "quote", "Related eNB DN"),
		field(DomainCM, "CC", "related_enb_id", "device_info.enb_id", "device_info.enb_id", "string", "quote", "Related eNB ID"),
		field(DomainCM, "CC", "related_enb_userlabel", "device.site_name", "devices.site_name / device_info.device_name", "string", "quote", "Related eNB label"),
		field(DomainCM, "CC", "cel_id", "device_info.cell_id", "device_info.cell_id", "string", "quote", "Cell ID"),
		field(DomainCM, "CC", "userlabel", "device_info.device_name", "device_info.device_name / devices.site_name", "string", "quote", "Cell label"),
		field(DomainCM, "CC", "pci", "device_info.pci", "device_info.pci", "string", "preserve text", "PCI"),
		field(DomainCM, "CC", "freq_mode", "device_info.network_model", "device_info.network_model", "string", "enum", "Duplex mode"),
		field(DomainCM, "CC", "bandIndicator", "device_info.band", "device_info.band", "string", "quote", "Band indicator"),
		field(DomainCM, "CC", "tac", "device_info.tac", "device_info.tac", "string", "quote", "TAC"),
		field(DomainCM, "CC", "zc_idx", "device_info.root_index", "device_info.root_index", "number", "number", "Root index"),
		field(DomainCM, "CC", "freq_pointno_ul", "device_info.ul_earfcn", "device_info.ul_earfcn", "number", "preserve text", "UL EARFCN"),
		field(DomainCM, "CC", "freq_pointno_dl", "device_info.freq_point", "device_info.freq_point", "number", "preserve text", "DL EARFCN"),
		field(DomainCM, "CC", "bandwidth_ul", "device_info.bandwidth", "device_info.bandwidth", "number", "number", "UL bandwidth"),
		field(DomainCM, "CC", "bandwidth_dl", "device_info.bandwidth", "device_info.bandwidth", "number", "number", "DL bandwidth"),
		field(DomainCM, "CC", "td_sfassignment", "device_info.subframe_assignment", "device_info.subframe_assignment", "string", "quote", "TDD subframe assignment"),
		field(DomainCM, "CC", "td_specialsfpatterns", "device_info.special_subframe", "device_info.special_subframe", "string", "quote", "TDD special subframe patterns"),
		field(DomainCM, "CC", "ECI", "device_info.eci", "device_info.eci", "string", "preserve text", "ECI"),
		field(DomainCM, "CC", "PCI", "device_info.pci", "device_info.pci", "string", "preserve text", "PCI"),
		field(DomainCM, "CC", "Cell ID", "device_info.cell_id", "device_info.cell_id", "string", "quote", "Cell ID"),
		field(DomainCM, "CC", "Earfcn", "device_info.freq_point", "device_info.freq_point", "number", "preserve text", "EARFCN"),
		field(DomainCM, "CC", "UL Earfcn", "device_info.ul_earfcn", "device_info.ul_earfcn", "number", "preserve text", "UL EARFCN"),
		field(DomainCM, "CC", "Bandwidth", "device_info.bandwidth", "device_info.bandwidth", "number", "number", "Bandwidth"),
		field(DomainCM, "CC", "Band", "device_info.band", "device_info.band", "string", "quote", "Band"),
		field(DomainCM, "CC", "Duplex Mode", "device_info.network_model", "device_info.network_model", "string", "quote", "Duplex mode"),
		field(DomainCM, "CC", "Transmit Power", "device_info.transmit_power", "device_info.transmit_power", "number", "number", "Transmit power"),
		field(DomainCM, "CC", "Subframe Assignment", "device_info.subframe_assignment", "device_info.subframe_assignment", "string", "quote", "Subframe assignment"),
		field(DomainCM, "CC", "Special Subframe Patterns", "device_info.special_subframe", "device_info.special_subframe", "string", "quote", "Special subframe patterns"),
		field(DomainCM, "CC", "UL EARFCN", "device_info.ul_earfcn", "device_info.ul_earfcn", "string", "quote", "UL EARFCN"),
		field(DomainCM, "CE", "Cell Status", "device_info.op_state", "device_info.op_state", "string", "enum", "Cell status"),
		field(DomainCM, "CE", "dn", "device.serial_number", "devices.serial_number", "string", "quote", "DN / device serial number"),
		field(DomainCM, "CE", "enb_id", "device_info.enb_id", "device_info.enb_id", "string", "quote", "eNB ID"),
		field(DomainCM, "CE", "userlabel", "device.site_name", "devices.site_name / device_info.device_name", "string", "quote", "Device label"),
		field(DomainCM, "CE", "enb_model", "device.model_name", "devices.model_name / devices.product_class", "string", "quote", "eNB model"),
		field(DomainCM, "CE", "ip_address", "device.ip_address", "devices.ip_address", "string", "quote", "IP address"),
		field(DomainCM, "CE", "software_version", "device.firmware_version", "devices.firmware_version", "string", "quote", "Software version"),
		field(DomainCM, "CE", "freq_mode", "device_info.network_model", "device_info.network_model", "string", "enum", "Duplex mode"),
		field(DomainCM, "CE", "cel_num", "device_info.num_of_cells", "device_info.num_of_cells", "number", "number", "Cell count"),
		field(DomainCM, "CE", "serialid", "device.serial_number", "devices.serial_number", "string", "quote", "Serial ID"),
		field(DomainCM, "CE", "Online Status", "device.is_online", "devices.is_online", "bool", "enum", "Online status"),
		field(DomainCM, "CE", "Device Status", "device.lifecycle_state", "devices.lifecycle_state + devices.is_online", "string", "enum", "Device status"),
		field(DomainCM, "CE", "RF Status", "device_info.rf_status", "device_info.rf_status", "string", "enum", "RF status"),
		field(DomainCM, "CE", "KPI Report Status", "device_info.kpi_status", "device_info.kpi_status", "string", "enum", "KPI report status"),
		field(DomainCM, "CE", "UE Count", "device_info.ue_count", "device_info.ue_count", "number", "number", "UE count"),
		field(DomainCM, "CE", "Alarms", "device_info.active_alarm_count", "device_info.active_alarm_count", "number", "number", "Active alarm count"),
		field(DomainCM, "CE", "Highest Alarm Severity", "device_info.highest_alarm_severity", "device_info.highest_alarm_severity", "string", "enum", "Highest alarm severity"),
		field(DomainCM, "CE", "System Uptime", "device_info.run_time", "device_info.run_time", "number", "number", "System uptime"),
		field(DomainCM, "CE", "Accumulated Online Time(s)", "device_info.cumulative_online_duration", "device_info.cumulative_online_duration", "number", "number", "Accumulated online time"),
		field(DomainCM, "CE", "GPS Satellites", "device_info.gps_satellites", "device_info.gps_satellites", "number", "number", "GPS satellites"),
		field(DomainCM, "CE", "GPS Height", "device_info.gps_height", "device_info.gps_height", "number", "number", "GPS height"),
		field(DomainCM, "CE", "Run Time", "device_info.run_time", "device_info.run_time", "string", "quote", "Run time"),
		field(DomainCM, "COMS", "Serial Number", "device.serial_number", "devices.serial_number", "string", "quote", "Device serial number"),
		field(DomainCM, "COMS", "Vendor", "device.manufacturer", "devices.manufacturer", "string", "quote", "Vendor"),
		field(DomainCM, "COMS", "Model", "device.model_name", "devices.model_name", "string", "quote", "Model"),
		field(DomainCM, "COMS", "ENODEB_ID", "device_info.enb_id", "device_info.enb_id", "string", "quote", "eNB ID"),
		field(DomainCM, "COMS", "ENODEB_NAME", "device.site_name", "devices.site_name / device_info.device_name", "string", "quote", "eNB name"),
		field(DomainCM, "COMS", "Cell ID", "device_info.cell_id", "device_info.cell_id", "string", "quote", "Cell ID"),
		field(DomainCM, "COMS", "CELL_NAME", "device_info.device_name", "device_info.device_name / devices.site_name", "string", "quote", "Cell name"),
		field(DomainCM, "COMS", "CELL_STATUS", "device_info.op_state", "device_info.op_state", "string", "enum", "Cell status"),
		field(DomainCM, "COMS", "ECI", "device_info.eci", "device_info.eci", "string", "preserve text", "ECI"),
		field(DomainCM, "COMS", "TAC", "device_info.tac", "device_info.tac", "string", "quote", "TAC"),
		field(DomainCM, "COMS", "PCI", "device_info.pci", "device_info.pci", "string", "preserve text", "PCI"),
		field(DomainCM, "COMS", "UL_EARFCN", "device_info.ul_earfcn", "device_info.ul_earfcn", "number", "preserve text", "UL EARFCN"),
		field(DomainCM, "COMS", "DL_EARFCN", "device_info.freq_point", "device_info.freq_point", "number", "preserve text", "DL EARFCN"),
		field(DomainCM, "COMS", "BAND", "device_info.band", "device_info.band", "string", "quote", "Band"),
		field(DomainCM, "COMS", "BANDWIDTH", "device_info.bandwidth", "device_info.bandwidth", "number", "number", "Bandwidth"),
		field(DomainCM, "COMS", "RS_POWER", "device_info.transmit_power", "device_info.transmit_power", "number", "number", "Reference signal power"),
		field(DomainCM, "COMS", "OMC-R", "system.omc_r", "northbound_profile.omc_r", "string", "quote", "OMC-R"),
		field(DomainCM, "COMS", "Province", "system.province", "northbound_profile.province", "string", "quote", "Province"),
		field(DomainCM, "COMS", "LocalHost", "runtime.local_host", "runtime.local_host", "string", "quote", "Local host"),
		field(DomainCM, "COMS", "DataVersion", "runtime.data_version", "northbound_profile.data_version", "string", "quote", "Data version"),
		field(DomainCM, "COMS", "DateTime", "task.window_end", "northbound_export_runs.window_end", "datetime", "yyyyMMddHHmmss", "File time"),
		field(DomainCM, "COMS", "Export Batch ID", "runtime.export_batch_id", "northbound_export_runs.id", "string", "quote", "Export batch ID"),
		field(DomainCM, "COMS", "OMC Name", "system.omc_name", "northbound_profile.omc_name", "string", "quote", "OMC name"),
		field(DomainCM, "COMS", "REGION", "system.province", "northbound_profile.province", "string", "quote", "Region"),
		field(DomainCM, "COMS", "DATE_TIME", "task.window_end", "northbound_export_runs.window_end", "datetime", "yyyyMMddHHmmss", "File time"),
		field(DomainCM, "COMS", "DUPLEXING", "device_info.network_model", "device_info.network_model", "string", "quote", "Duplexing"),
		field(DomainCM, "COMS", "MAXTXPOWER", "device_info.transmit_power", "device_info.transmit_power", "number", "number", "Max transmit power"),
		field(DomainPM, "PC", "Device SN", "pm.device_sn", "pm_metrics.device_sn", "string", "quote", "Device SN"),
		field(DomainPM, "PC", "Metric Path", "pm.metric_path", "pm_metrics.metric_path", "string", "quote", "Metric path"),
		field(DomainPM, "PC", "Metric Type", "pm.metric_type", "pm_metrics.metric_type", "string", "quote", "Metric type"),
		field(DomainPM, "PC", "Metric Value", "pm.metric_value", "pm_metrics.metric_value", "number", "number", "Metric value"),
		field(DomainPM, "PC", "Statis Type", "pm.statis_type", "pm_metrics.statis_type", "string", "quote", "Statis type"),
		field(DomainPM, "PC", "Granularity", "pm.granularity", "pm_metrics.granularity", "string", "quote", "Granularity"),
		field(DomainPM, "PC", "End Time", "pm.end_time", "pm_metrics.end_time", "datetime", "yyyy-MM-dd HH:mm:ss", "End time"),
		field(DomainPM, "PC", "Object LDN", "pm.object_ldn", "pm_metrics.object_ldn", "string", "quote", "Object LDN"),
		field(DomainPM, "PE", "Device SN", "pm.device_sn", "pm_metrics.device_sn", "string", "quote", "Device SN"),
		field(DomainPM, "PE", "Metric Path", "pm.metric_path", "pm_metrics.metric_path", "string", "quote", "Metric path"),
		field(DomainPM, "PE", "Metric Type", "pm.metric_type", "pm_metrics.metric_type", "string", "quote", "Metric type"),
		field(DomainPM, "PE", "Metric Value", "pm.metric_value", "pm_metrics.metric_value", "number", "number", "Metric value"),
		field(DomainPM, "PE", "Statis Type", "pm.statis_type", "pm_metrics.statis_type", "string", "quote", "Statis type"),
		field(DomainPM, "PE", "Granularity", "pm.granularity", "pm_metrics.granularity", "string", "quote", "Granularity"),
		field(DomainPM, "PE", "End Time", "pm.end_time", "pm_metrics.end_time", "datetime", "yyyy-MM-dd HH:mm:ss", "End time"),
		field(DomainPM, "PE", "Object LDN", "pm.object_ldn", "pm_metrics.object_ldn", "string", "quote", "Object LDN"),
		field(DomainMR, "MRO", "file_type", "mr.file_type", "mr_files.file_type", "string", "quote", "MR type"),
		field(DomainMR, "MRO", "device_sn", "mr.device_sn", "mr_files.device_sn", "string", "quote", "Device SN"),
		field(DomainMR, "MRE", "file_type", "mr.file_type", "mr_files.file_type", "string", "quote", "MR type"),
		field(DomainMR, "MRS", "file_type", "mr.file_type", "mr_files.file_type", "string", "quote", "MR type"),
		field(DomainLOG, "login", "log_time", "log.log_time", "sys_login_logs.login_at", "datetime", "yyyy-MM-dd HH:mm:ss", "Log time"),
		field(DomainLOG, "login", "sys_source_name", "log.sys_source_name", "literal baicells omc", "string", "quote", "System source name"),
		field(DomainLOG, "login", "account_name", "log.account_name", "sys_login_logs.username", "string", "quote", "Account name"),
		field(DomainLOG, "login", "terminal_name", "log.terminal_name", "sys_login_logs.browser / os", "string", "quote", "Terminal name"),
		field(DomainLOG, "login", "terminal_ip", "log.terminal_ip", "sys_login_logs.ip_address", "string", "quote", "Terminal IP"),
		field(DomainLOG, "login", "log_result", "log.result", "sys_login_logs.status", "string", "enum", "Log result"),
		field(DomainLOG, "login", "log_start_time", "log.log_start_time", "sys_login_logs.login_at", "datetime", "yyyy-MM-dd HH:mm:ss", "Log start time"),
		field(DomainLOG, "login", "log_end_time", "log.log_end_time", "sys_login_logs.login_at", "datetime", "yyyy-MM-dd HH:mm:ss", "Log end time"),
		field(DomainLOG, "operation", "log_time", "log.log_time", "sys_oper_logs.created_at", "datetime", "yyyy-MM-dd HH:mm:ss", "Log time"),
		field(DomainLOG, "operation", "sys_source_name", "log.sys_source_name", "literal baicells omc", "string", "quote", "System source name"),
		field(DomainLOG, "operation", "terminal_name", "log.terminal_name", "sys_oper_logs.user_agent", "string", "quote", "Terminal name"),
		field(DomainLOG, "operation", "terminal_ip", "log.terminal_ip", "sys_oper_logs.ip_address", "string", "quote", "Terminal IP"),
		field(DomainLOG, "operation", "main_name", "log.main_name", "sys_oper_logs.username", "string", "quote", "Main account"),
		field(DomainLOG, "operation", "sub_account", "log.sub_account", "literal empty", "string", "quote", "Sub account"),
		field(DomainLOG, "operation", "asset_name", "log.asset_name", "literal empty", "string", "quote", "Asset name"),
		field(DomainLOG, "operation", "asset_ip", "log.asset_ip", "literal empty", "string", "quote", "Asset IP"),
		field(DomainLOG, "operation", "asset_port", "log.asset_port", "literal empty", "string", "quote", "Asset port"),
		field(DomainLOG, "operation", "asset_attribute", "log.asset_attribute", "literal empty", "string", "quote", "Asset attribute"),
		field(DomainLOG, "operation", "log_data", "log.detail", "sys_oper_logs.action / detail / status", "string", "quote", "Log data"),
		field(DomainLOG, "login_fix", "ID", "log.id", "sys_login_logs.id", "string", "quote", "ID"),
		field(DomainLOG, "login_fix", "User Name", "log.user_name", "sys_login_logs.username", "string", "quote", "User name"),
		field(DomainLOG, "login_fix", "IP Address", "log.ip_address", "sys_login_logs.ip_address", "string", "quote", "IP address"),
		field(DomainLOG, "login_fix", "Log Name", "log.log_name", "literal LoginLogout", "string", "quote", "Log name"),
		field(DomainLOG, "login_fix", "Record Detail", "log.detail", "sys_login_logs.message", "string", "quote", "Record detail"),
		field(DomainLOG, "login_fix", "Results", "log.result_text", "sys_login_logs.status", "string", "enum", "Results"),
		field(DomainLOG, "login_fix", "Failure Reason", "log.failure_reason", "sys_login_logs.message", "string", "quote", "Failure reason"),
		field(DomainLOG, "login_fix", "Time", "log.login_time", "sys_login_logs.login_at", "datetime", "yyyy-MM-dd HH:mm:ss", "Time"),
		field(DomainLOG, "operation_fix", "User Name", "log.user_name", "sys_oper_logs.username", "string", "quote", "User name"),
		field(DomainLOG, "operation_fix", "IP Address", "log.ip_address", "sys_oper_logs.ip_address", "string", "quote", "IP address"),
		field(DomainLOG, "operation_fix", "Log Name", "log.log_name", "sys_oper_logs.action / module", "string", "quote", "Log name"),
		field(DomainLOG, "operation_fix", "Record Detail", "log.detail", "sys_oper_logs.detail / target", "string", "quote", "Record detail"),
		field(DomainLOG, "operation_fix", "Results", "log.result_text", "sys_oper_logs.status", "string", "enum", "Results"),
		field(DomainLOG, "operation_fix", "Failure Reason", "log.failure_reason", "sys_oper_logs.error_msg", "string", "quote", "Failure reason"),
		field(DomainLOG, "operation_fix", "Op Start Time", "log.op_start_time", "sys_oper_logs.created_at", "datetime", "yyyy-MM-dd HH:mm:ss", "Operation start time"),
		field(DomainLOG, "operation_fix", "Op End Time", "log.op_end_time", "sys_oper_logs.created_at + cost_ms", "datetime", "yyyy-MM-dd HH:mm:ss", "Operation end time"),
	}
	for _, objectCode := range []string{"ENB", "GNB", "GSM"} {
		for index, seed := range stationInventoryFieldSeeds {
			fields = append(fields, inventoryField(objectCode, index, seed))
		}
	}
	fields = append(fields,
		field(DomainInventory, "OMC", "eNB online", "inventory.omc.enb_online", "devices.is_online aggregate", "number", "number", "eNB online"),
		field(DomainInventory, "OMC", "eNB active", "inventory.omc.enb_active", "device_info.op_state / lifecycle aggregate", "number", "number", "eNB active"),
		field(DomainInventory, "OMC", "MME status", "inventory.omc.mme_status", "device_info.mme_status aggregate", "string", "quote", "MME status"),
		field(DomainInventory, "OMC", "UE Count", "inventory.omc.ue_count", "SUM(device_info.ue_count)", "number", "number", "UE count"),
		field(DomainInventory, "OMC", "Version", "inventory.omc.version", "buildinfo.ReleaseVersion", "string", "quote", "Version"),
	)
	fields = appendLegacyCMFields(fields)
	return appendLegacyPMFields(fields)
}

func field(domain Domain, objectCode, outputAlias, systemField, source, dataType, renderer, cnName string) FieldDefinition {
	return FieldDefinition{
		Key:           strings.ToLower(string(domain) + "." + objectCode + "." + systemField),
		Domain:        domain,
		ObjectCode:    objectCode,
		OutputAlias:   outputAlias,
		SystemField:   systemField,
		Source:        source,
		DataType:      dataType,
		Renderer:      renderer,
		CnName:        cnName,
		SupportStatus: SupportSupported,
	}
}

func profileField(profile string, domain Domain, objectCode, outputAlias, systemField, source, dataType, renderer, cnName string) FieldDefinition {
	definition := field(domain, objectCode, outputAlias, systemField, source, dataType, renderer, cnName)
	definition.Profile = profile
	definition.Key = strings.ToLower(string(domain) + "." + objectCode + "." + profile + "." + systemField)
	return definition
}

func legacyCMField(profile, objectCode, outputAlias, systemField, source, dataType, renderer, cnName string) FieldDefinition {
	definition := profileField(profile, DomainCM, objectCode, outputAlias, systemField, source, dataType, renderer, cnName)
	definition.Tech = "LTE"
	return definition
}

type legacyCMFieldSeed struct {
	outputAlias string
	systemField string
	source      string
	dataType    string
	renderer    string
	cnName      string
}

func legacyCMFields(profile, objectCode string, seeds []legacyCMFieldSeed) []FieldDefinition {
	out := make([]FieldDefinition, 0, len(seeds))
	for _, seed := range seeds {
		out = append(out, legacyCMField(profile, objectCode, seed.outputAlias, seed.systemField, seed.source, seed.dataType, seed.renderer, seed.cnName))
	}
	return out
}

func legacyCMEPFields(profile string) []FieldDefinition {
	return legacyCMFields(profile, "EP", legacyCMEPFieldSeeds)
}

var legacyCMCPBaseFieldSeeds = []legacyCMFieldSeed{
	{"dn", "device_info.eci", "device_info.eci", "string", "preserve text", "Cell DN / ECI"},
	{"related_enb_dn", "device.serial_number", "devices.serial_number", "string", "quote", "Related eNB DN"},
	{"related_enb_id", "device_info.enb_id", "device_info.enb_id", "string", "quote", "Related eNB ID"},
	{"related_enb_userlabel", "device.site_name", "devices.site_name / device_info.device_name", "string", "quote", "Related eNB label"},
	{"cel_id", "device_info.cell_id", "device_info.cell_id", "string", "quote", "Cell ID"},
	{"cel_userlabel", "device_info.device_name", "device_info.device_name / devices.site_name", "string", "quote", "Cell label"},
	{"DrxAlgSwitch", "device_param.DrxAlgSwitch", "device_parameters LTE DRX enabled", "bool", "enum", "DRX algorithm switch"},
	{"ShortDrxSwitch", "device_param.ShortDrxSwitch", "device_parameters LTE short DRX enabled", "bool", "enum", "Short DRX switch"},
	{"OnDurationTimer", "device_param.OnDurationTimer", "device_parameters LTE DRX on-duration timer", "string", "quote", "On duration timer"},
	{"DrxInactivityTimer", "device_param.DrxInactivityTimer", "device_parameters LTE DRX inactivity timer", "string", "quote", "DRX inactivity timer"},
	{"DrxReTxTimer", "device_param.DrxReTxTimer", "device_parameters LTE DRX retransmission timer", "string", "quote", "DRX retransmission timer"},
	{"LongDrxCycle", "device_param.LongDrxCycle", "device_parameters LTE long DRX cycle", "string", "quote", "Long DRX cycle"},
	{"ShortDrxCycle", "device_param.ShortDrxCycle", "device_parameters LTE short DRX cycle", "string", "quote", "Short DRX cycle"},
	{"DrxShortCycleTimer", "device_param.DrxShortCycleTimer", "device_parameters LTE short DRX cycle timer", "string", "quote", "DRX short cycle timer"},
	{"UeInactiveTimer", "device_param.UeInactiveTimer", "device_parameters LTE UE inactive timer", "string", "quote", "UE inactive timer"},
	{"T304ForEutran", "device_param.T304ForEutran", "device_parameters LTE T304 EUTRA timer", "string", "quote", "T304 for EUTRAN"},
	{"T310", "device_param.T310", "device_parameters LTE T310 timer", "string", "quote", "T310 timer"},
	{"DefaultPagingCycle", "device_param.DefaultPagingCycle", "device_parameters LTE default paging cycle", "string", "quote", "Default paging cycle"},
	{"SysTimeCfgInd", "device_param.SysTimeCfgInd", "device_parameters LTE system time config indicator", "bool", "enum", "System time config indicator"},
	{"referenceSignalPower", "device_info.transmit_power", "device_info.transmit_power", "number", "number", "Reference signal power"},
	{"PA", "device_param.PA", "device_parameters LTE PDSCH Pa", "string", "quote", "PDSCH PA"},
	{"PB", "device_param.PB", "device_parameters LTE PDSCH Pb", "string", "quote", "PDSCH PB"},
	{"PreambInitRcvTargetPwr", "device_param.PreambInitRcvTargetPwr", "device_parameters LTE preamble initial received target power", "string", "quote", "Preamble initial received target power"},
	{"powerRampingStep", "device_param.powerRampingStep", "device_parameters LTE power ramping step", "string", "quote", "Power ramping step"},
	{"N310", "device_param.N310", "device_parameters LTE N310 timer", "string", "quote", "N310"},
	{"N311", "device_param.N311", "device_parameters LTE N311 timer", "string", "quote", "N311"},
	{"T311", "device_param.T311", "device_parameters LTE T311 timer", "string", "quote", "T311"},
	{"T300", "device_param.T300", "device_parameters LTE T300 timer", "string", "quote", "T300"},
	{"T301", "device_param.T301", "device_parameters LTE T301 timer", "string", "quote", "T301"},
	{"T302", "device_param.T302", "device_parameters LTE T302 timer", "string", "quote", "T302"},
	{"VoLTESwitch", "device_param.VoLTESwitch", "device_parameters LTE VoLTE switch", "string", "quote", "VoLTE switch"},
	{"Lcg", "device_param.Lcg", "device_parameters LTE LCG", "string", "quote", "LCG"},
}

var legacyCMCPPlmnFieldSeeds = append(append([]legacyCMFieldSeed{}, legacyCMCPBaseFieldSeeds...),
	legacyCMFieldSeed{"PlmnIdList", "device_info.plmn", "device_info.plmn", "string", "preserve text", "PLMN ID list"},
)

var legacyCMCPThaiTrueFieldSeeds = append(append([]legacyCMFieldSeed{}, legacyCMCPBaseFieldSeeds...),
	legacyCMFieldSeed{"Shop Id", "device.site_id", "devices.site_id", "string", "quote", "Shop ID"},
	legacyCMFieldSeed{"Latitude", "device.latitude", "devices.latitude", "number", "number", "Latitude"},
	legacyCMFieldSeed{"Longitude", "device.longitude", "devices.longitude", "number", "number", "Longitude"},
	legacyCMFieldSeed{"Height", "device_info.gps_height", "device_info.gps_height", "number", "number", "GPS height"},
	legacyCMFieldSeed{"Site Name", "device.site_name", "devices.site_name / device_info.device_name", "string", "quote", "Site name"},
	legacyCMFieldSeed{"Install Detail Address", "device_info.address", "device_info.address", "string", "quote", "Installation address"},
	legacyCMFieldSeed{"Last Period Time", "device.last_inform_at", "devices.last_inform_at", "datetime", "yyyy-MM-dd HH:mm:ss", "Last period time"},
	legacyCMFieldSeed{"Cell Active State", "nbi.cell_active_state", "derived from device_info.op_state", "string", "enum", "Cell active state"},
	legacyCMFieldSeed{"Cell Admin State", "nbi.cell_admin_state", "derived from device_info.rf_status", "string", "enum", "Cell admin state"},
)

var legacyCMEPFieldSeeds = []legacyCMFieldSeed{
	{"dn", "device.serial_number", "devices.serial_number", "string", "quote", "DN / device serial number"},
	{"enb_id", "device_info.enb_id", "device_info.enb_id", "string", "quote", "eNB ID"},
	{"enb_userlabel", "device.site_name", "devices.site_name / device_info.device_name", "string", "quote", "eNB label"},
	{"ShortDrxSwitch", "device_param.ShortDrxSwitch", "device_parameters LTE short DRX enabled", "bool", "enum", "Short DRX switch"},
	{"OnDurationTimer", "device_param.OnDurationTimer", "device_parameters LTE DRX on-duration timer", "string", "quote", "On duration timer"},
	{"DrxInactivityTimer", "device_param.DrxInactivityTimer", "device_parameters LTE DRX inactivity timer", "string", "quote", "DRX inactivity timer"},
	{"DrxReTxTimer", "device_param.DrxReTxTimer", "device_parameters LTE DRX retransmission timer", "string", "quote", "DRX retransmission timer"},
	{"LongDrxCycle", "device_param.LongDrxCycle", "device_parameters LTE long DRX cycle", "string", "quote", "Long DRX cycle"},
	{"ShortDrxCycle", "device_param.ShortDrxCycle", "device_parameters LTE short DRX cycle", "string", "quote", "Short DRX cycle"},
	{"DrxShortCycleTimer", "device_param.DrxShortCycleTimer", "device_parameters LTE short DRX cycle timer", "string", "quote", "DRX short cycle timer"},
	{"UeInactiveTimer", "device_param.UeInactiveTimer", "device_parameters LTE UE inactive timer", "string", "quote", "UE inactive timer"},
	{"T304ForEutran", "device_param.T304ForEutran", "device_parameters LTE T304 EUTRA timer", "string", "quote", "T304 for EUTRAN"},
	{"T310", "device_param.T310", "device_parameters LTE T310 timer", "string", "quote", "T310 timer"},
	{"DefaultPagingCycle", "device_param.DefaultPagingCycle", "device_parameters LTE default paging cycle", "string", "quote", "Default paging cycle"},
	{"SysTimeCfgInd", "device_param.SysTimeCfgInd", "device_parameters LTE system time config indicator", "bool", "enum", "System time config indicator"},
	{"encrypAlgPriority", "device_param.encrypAlgPriority", "device_parameters LTE allowed ciphering algorithm list", "string", "quote", "Encryption algorithm priority"},
	{"integProtAlgPriority", "device_param.integProtAlgPriority", "device_parameters LTE allowed integrity protection algorithm list", "string", "quote", "Integrity protection algorithm priority"},
	{"Lcg", "device_param.Lcg", "device_parameters LTE LCG", "string", "quote", "LCG"},
	{"VoLTESwitch", "device_param.VoLTESwitch", "device_parameters LTE VoLTE switch", "string", "quote", "VoLTE switch"},
}

var legacyCMCCBaseFieldSeeds = []legacyCMFieldSeed{
	{"dn", "device_info.eci", "device_info.eci", "string", "preserve text", "Cell DN / ECI"},
	{"related_enb_dn", "device.serial_number", "devices.serial_number", "string", "quote", "Related eNB DN"},
	{"related_enb_id", "device_info.enb_id", "device_info.enb_id", "string", "quote", "Related eNB ID"},
	{"related_enb_userlabel", "device.site_name", "devices.site_name / device_info.device_name", "string", "quote", "Related eNB label"},
	{"cel_id", "device_info.cell_id", "device_info.cell_id", "string", "quote", "Cell ID"},
	{"userlabel", "device_info.device_name", "device_info.device_name / devices.site_name", "string", "quote", "Cell label"},
	{"pci", "device_info.pci", "device_info.pci", "string", "preserve text", "PCI"},
	{"freq_mode", "nbi.freq_mode", "derived from device_info.network_model", "string", "enum", "Duplex mode"},
	{"bandIndicator", "device_info.band", "device_info.band", "string", "quote", "Band indicator"},
	{"tac", "device_info.tac", "device_info.tac", "string", "quote", "TAC"},
	{"zc_idx", "device_info.root_index", "device_info.root_index", "number", "number", "Root index"},
	{"freq_pointno_ul", "device_info.ul_earfcn", "device_info.ul_earfcn", "number", "preserve text", "UL EARFCN"},
	{"freq_pointno_dl", "device_info.freq_point", "device_info.freq_point", "number", "preserve text", "DL EARFCN"},
	{"bandwidth_ul", "device_info.bandwidth", "device_info.bandwidth", "number", "number", "UL bandwidth"},
	{"bandwidth_dl", "device_info.bandwidth", "device_info.bandwidth", "number", "number", "DL bandwidth"},
	{"td_sfassignment", "device_info.subframe_assignment", "device_info.subframe_assignment", "string", "quote", "TDD subframe assignment"},
	{"td_specialsfpatterns", "device_info.special_subframe", "device_info.special_subframe", "string", "quote", "TDD special subframe patterns"},
}

var legacyCMS0005CCFieldSeeds = append(append([]legacyCMFieldSeed{}, legacyCMCCBaseFieldSeeds...),
	legacyCMFieldSeed{"mac_address", "device_info.mac", "device_info.mac", "string", "quote", "MAC address"},
	legacyCMFieldSeed{"cell_status", "device_info.op_state", "device_info.op_state", "string", "enum", "Cell status"},
)

var legacyCMCEBaseFieldSeeds = []legacyCMFieldSeed{
	{"dn", "device.serial_number", "devices.serial_number", "string", "quote", "DN / device serial number"},
	{"enb_id", "device_info.enb_id", "device_info.enb_id", "string", "quote", "eNB ID"},
	{"userlabel", "device.site_name", "devices.site_name / device_info.device_name", "string", "quote", "Device label"},
	{"enb_model", "device.model_name", "devices.model_name / devices.product_class", "string", "quote", "eNB model"},
	{"ip_address", "device.ip_address", "devices.ip_address", "string", "quote", "IP address"},
	{"software_version", "device.firmware_version", "devices.firmware_version", "string", "quote", "Software version"},
	{"freq_mode", "nbi.freq_mode", "derived from device_info.network_model", "string", "enum", "Duplex mode"},
	{"cel_num", "device_info.num_of_cells", "device_info.num_of_cells", "number", "number", "Cell count"},
	{"serialid", "device.serial_number", "devices.serial_number", "string", "quote", "Serial ID"},
}

var legacyCMCEHeNBFieldSeeds = []legacyCMFieldSeed{
	{"dn", "device.serial_number", "devices.serial_number", "string", "quote", "DN / device serial number"},
	{"enb_id", "device_info.enb_id", "device_info.enb_id", "string", "quote", "eNB ID"},
	{"userlabel", "device.site_name", "devices.site_name / device_info.device_name", "string", "quote", "Device label"},
	{"enb_model", "device.model_name", "devices.model_name / devices.product_class", "string", "quote", "eNB model"},
	{"ip_address", "device.ip_address", "devices.ip_address", "string", "quote", "IP address"},
	{"software_version", "device.firmware_version", "devices.firmware_version", "string", "quote", "Software version"},
	{"freq_mode", "nbi.freq_mode", "derived from device_info.network_model", "string", "enum", "Duplex mode"},
	{"cel_num", "device_info.num_of_cells", "device_info.num_of_cells", "number", "number", "Cell count"},
	{"HeNB_longitude", "device.longitude", "devices.longitude", "number", "number", "HeNB longitude"},
	{"HeNB_latitude", "device.latitude", "devices.latitude", "number", "number", "HeNB latitude"},
	{"serialid", "device.serial_number", "devices.serial_number", "string", "quote", "Serial ID"},
}

var legacyCMS0007COMSFieldSeeds = []legacyCMFieldSeed{
	{"DATE_TIME", "nbi.coms_date_time", "derived from export window", "datetime", "dd/MM/yyyy HH:mm", "Date time"},
	{"DUPLEXING", "nbi.freq_mode", "derived from device_info.network_model", "string", "enum", "Duplexing"},
	{"TXRX_MODE", "device.model_name", "devices.model_name / devices.product_class", "string", "quote", "Tx/Rx mode"},
	{"ENODEB_ID", "device_info.enb_id", "device_info.enb_id", "string", "quote", "eNB ID"},
	{"ENODEB_NAME", "device.site_name", "devices.site_name / device_info.device_name", "string", "quote", "eNB name"},
	{"CELL_ID", "device_info.cell_id", "device_info.cell_id", "string", "quote", "Cell ID"},
	{"CELL_NAME", "device_info.device_name", "device_info.device_name / devices.site_name", "string", "quote", "Cell name"},
	{"CELL_STATUS", "nbi.cell_status", "derived from device_info.op_state", "string", "enum", "Cell status"},
	{"SITE_CODE", "device.site_id", "devices.site_id", "string", "quote", "Site code"},
	{"SITE_DEPLOYMENT", "device.model_name", "devices.model_name / devices.product_class", "string", "quote", "Site deployment"},
	{"SECTOR_TYPE", "device.model_name", "devices.model_name / devices.product_class", "string", "quote", "Sector type"},
	{"MCC", "nbi.mcc", "derived from device_info.plmn", "string", "quote", "MCC"},
	{"MNC", "nbi.mnc", "derived from device_info.plmn", "string", "quote", "MNC"},
	{"TAC_DEC", "device_info.tac", "device_info.tac", "string", "quote", "TAC decimal"},
	{"PCI", "device_info.pci", "device_info.pci", "string", "preserve text", "PCI"},
	{"UL_EARFCN", "device_info.ul_earfcn", "device_info.ul_earfcn", "number", "preserve text", "UL EARFCN"},
	{"DL_EARFCN", "device_info.freq_point", "device_info.freq_point", "number", "preserve text", "DL EARFCN"},
	{"BAND", "device_info.band", "device_info.band", "string", "quote", "Band"},
	{"BANDWIDTH", "device_info.bandwidth", "device_info.bandwidth", "number", "number", "Bandwidth"},
	{"MAXTXPOWER", "device_info.transmit_power", "device_info.transmit_power", "number", "number", "Max transmit power"},
	{"RS_POWER", "device_info.transmit_power", "device_info.transmit_power", "number", "number", "Reference signal power"},
	{"CELL_RANGE", "device_info.num_of_cells", "device_info.num_of_cells", "number", "number", "Cell range"},
	{"PA", "device_param.PA", "device_parameters LTE PDSCH Pa", "string", "quote", "PDSCH PA"},
	{"PB", "device_param.PB", "device_parameters LTE PDSCH Pb", "string", "quote", "PDSCH PB"},
	{"SFN_NO", "device.serial_number", "devices.serial_number", "string", "quote", "SFN number"},
}

var legacyCMCOMSFieldSeeds = []legacyCMFieldSeed{
	{"Serial Number", "device.serial_number", "devices.serial_number", "string", "quote", "Device serial number"},
	{"Femto ID", "device_info.cell_id", "device_info.cell_id", "string", "quote", "Femto ID"},
	{"BSR Name", "device_info.device_name", "device_info.device_name / devices.site_name", "string", "quote", "BSR name"},
	{"External IP", "device.ip_address", "devices.ip_address", "string", "quote", "External IP"},
	{"Last Sync Date", "device.last_inform_at", "devices.last_inform_at", "datetime", "yyyy-MM-dd HH:mm:ss", "Last sync date"},
	{"Vendor", "device.manufacturer", "devices.manufacturer", "string", "quote", "Vendor"},
	{"Model", "device.model_name", "devices.model_name", "string", "quote", "Model"},
	{"Cell ID", "device_info.cell_id", "device_info.cell_id", "string", "quote", "Cell ID"},
	{"LAC", "device_info.lac", "device_info.lac", "string", "quote", "LAC"},
	{"MCC", "nbi.mcc", "derived from device_info.plmn", "string", "quote", "MCC"},
	{"MNC", "nbi.mnc", "derived from device_info.plmn", "string", "quote", "MNC"},
	{"eNODEB ID", "device_info.enb_id", "device_info.enb_id", "string", "quote", "eNodeB ID"},
	{"TAC", "device_info.tac", "device_info.tac", "string", "quote", "TAC"},
}

func appendLegacyCMFields(fields []FieldDefinition) []FieldDefinition {
	for _, profile := range []string{legacyCMCPXMLProfile, legacyCMCPCSVProfile} {
		fields = append(fields, legacyCMFields(profile, "CP", legacyCMCPBaseFieldSeeds)...)
	}
	fields = append(fields, legacyCMFields(legacyCMPlmnCPXMLProfile, "CP", legacyCMCPPlmnFieldSeeds)...)
	fields = append(fields, legacyCMFields(legacyCMS0006CPXMLProfile, "CP", legacyCMCPThaiTrueFieldSeeds)...)
	for _, profile := range []string{legacyS0001EPProfile, legacyCMEPXMLProfile, legacyCMEPCSVProfile} {
		fields = append(fields, legacyCMEPFields(profile)...)
	}
	for _, profile := range []string{legacyCMCCXMLProfile, legacyCMCCCSVProfile} {
		fields = append(fields, legacyCMFields(profile, "CC", legacyCMCCBaseFieldSeeds)...)
	}
	fields = append(fields, legacyCMFields(legacyCMS0005CCXMLProfile, "CC", legacyCMS0005CCFieldSeeds)...)
	for _, profile := range []string{legacyCMCEXMLProfile, legacyCMCECSVProfile} {
		fields = append(fields, legacyCMFields(profile, "CE", legacyCMCEBaseFieldSeeds)...)
	}
	fields = append(fields, legacyCMFields(legacyCMHeNBCEXMLProfile, "CE", legacyCMCEHeNBFieldSeeds)...)
	fields = append(fields, legacyCMFields(legacyCMS0007COMSCSVProfile, "COMS", legacyCMS0007COMSFieldSeeds)...)
	fields = append(fields, legacyCMFields(legacyCMCOMSCSVProfile, "COMS", legacyCMCOMSFieldSeeds)...)
	return fields
}

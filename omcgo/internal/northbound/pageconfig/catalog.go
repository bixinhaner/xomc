package pageconfig

import (
	"strconv"
	"strings"
)

const (
	pathCM        = "/#FTPRoot#/#Province#/#OMC-R#/CM/#DateTime#/"
	pathPM        = "/#FTPRoot#/#Province#/#OMC-R#/PM/#DateTime#/"
	pathMR        = "/#FTPRoot#/#Province#/#OMC-R#/MR/#DateTime#/"
	pathLOG       = "/#FTPRoot#/#Province#/#OMC-R#/LOGS/#DateTime#/"
	pathInventory = "/#FTPRoot#/#Province#/#OMC-R#/Inventory/#Object#/#DateTime#/"

	nameCM        = "Baicells-#Object#-#LocalHost#-#DataVersion#-#DateTime#[-#Ri#][-#FileID#]"
	namePM        = "Baicells-#Object#-#LocalHost#-#DataVersion#-#DateTime#[-#Ri#]-#DataPeriod#[-#FileID#]"
	nameMR        = "#ModuleType#-Baicells-#Object#-#LocalHost#-#eNBID#-#DateTime#[-#Ri#].xml"
	nameLOG       = "Northbound-log-{login|operation}-#DateTime#.txt"
	nameInventory = "BaiOMC_#Object#_#DateTime#.csv"
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
		out = append(out, f)
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

func obj(codes ...string) []ScenarioObject {
	out := make([]ScenarioObject, 0, len(codes))
	for _, code := range codes {
		out = append(out, ScenarioObject{Code: code})
	}
	return out
}

func defaultFileProfiles() []FileProfile {
	cm := obj("CP", "EP", "CC", "CE")
	cmComs := obj("CP", "EP", "CC", "CE", "COMS")
	mr := obj("MRO", "MRE", "MRS")
	return []FileProfile{
		fileProfile("S0001", "CM + PM PC + MR standard 15M", "Standard", "Standard", []string{"baseline"}, []FileGroup{
			group("cm-daily", DomainCM, FormatXML, Period24H, 1, pathCM, nameCM, cm),
			group("pm-15m", DomainPM, FormatCSV, Period15M, 5, pathPM, namePM, obj("PC")),
			group("mr-15m", DomainMR, FormatXML, Period15M, 0, pathMR, nameMR, mr),
		}),
		fileProfile("S0002", "CM + PM PE/PC + MR", "Telecom", "Telecom", []string{"PE/PC", "csv |"}, []FileGroup{
			group("cm-daily", DomainCM, FormatXML, Period24H, 1, pathCM, nameCM, cm),
			pipeCSV(group("pm-15m", DomainPM, FormatCSV, Period15M, 5, pathPM, namePM, obj("PE", "PC"))),
			group("mr-15m", DomainMR, FormatXML, Period15M, 0, pathMR, nameMR, mr),
		}),
		fileProfile("S0003", "SH Telecom", "SH-Tele", "SH-Tele", []string{"PE/PC", "csv |"}, []FileGroup{
			group("cm-daily", DomainCM, FormatXML, Period24H, 1, pathCM, nameCM, cm),
			pipeCSV(group("pm-15m", DomainPM, FormatCSV, Period15M, 5, pathPM, namePM, obj("PE", "PC"))),
			group("mr-15m", DomainMR, FormatXML, Period15M, 0, pathMR, nameMR, mr),
		}),
		fileProfile("S0004", "Shaanxi Telecom", "SN-Tele", "SN-Tele", []string{"PE/PC", "alarm cn", "csv |"}, []FileGroup{
			group("cm-daily", DomainCM, FormatXML, Period24H, 1, pathCM, nameCM, cm),
			pipeCSV(group("pm-15m", DomainPM, FormatCSV, Period15M, 5, pathPM, namePM, obj("PE", "PC"))),
			group("mr-15m", DomainMR, FormatXML, Period15M, 0, pathMR, nameMR, mr),
		}),
		fileProfile("S0005", "Jiangsu Telecom delayed PM + custom log", "JS-Tele", "JS-Tele", []string{"PE/PC", "log type1", "csv |"}, []FileGroup{
			group("cm-daily", DomainCM, FormatXML, Period24H, 1, pathCM, nameCM, cm),
			pipeCSV(group("pm-15m-delayed", DomainPM, FormatCSV, Period15M, 14, pathPM, namePM, obj("PE", "PC"))),
			group("mr-15m", DomainMR, FormatXML, Period15M, 0, pathMR, nameMR, mr),
			group("log-custom-daily", DomainLOG, FormatTXT, Period24H, 5, pathLOG, nameLOG, obj("login", "operation")),
		}),
		fileProfile("S0006", "CM/PM without MR", "Thai-True", "Thai-True", []string{"no MR"}, []FileGroup{
			group("cm-daily", DomainCM, FormatXML, Period24H, 1, pathCM, nameCM, cm),
			group("pm-15m", DomainPM, FormatCSV, Period15M, 5, pathPM, namePM, obj("PC")),
		}),
		fileProfile("S0007", "CM CSV + COMS + PM 60M", "Thai-AIS", "Thai-AIS", []string{"CM CSV", "COMS", "PM 60M"}, []FileGroup{
			group("cm-daily-csv", DomainCM, FormatCSV, Period24H, 0, pathCM, nameCM, cmComs),
			group("pm-60m", DomainPM, FormatCSV, Period60M, 20, pathPM, namePM, obj("PC")),
		}),
		fileProfile("S0008", "CM CSV + COMS + fixed log", "V1", "V1", []string{"CM CSV", "COMS", "log type2"}, []FileGroup{
			group("cm-daily-csv", DomainCM, FormatCSV, Period24H, 1, pathCM, nameCM, cmComs),
			group("pm-15m", DomainPM, FormatCSV, Period15M, 5, pathPM, namePM, obj("PC")),
			group("mr-15m", DomainMR, FormatXML, Period15M, 0, pathMR, nameMR, mr),
			group("log-fixed-daily", DomainLOG, FormatCSV, Period24H, 5, pathLOG, "Northbound-log-fix-{login|operation}-#DateTime#.csv", obj("login_fix", "operation_fix")),
		}),
		fileProfile("S0009", "PC 60M + MR", "Laos-Tele", "Laos-Tele", []string{"PM 60M"}, []FileGroup{
			group("cm-daily", DomainCM, FormatXML, Period24H, 1, pathCM, nameCM, cm),
			group("pm-pc-60m", DomainPM, FormatCSV, Period60M, 0, pathPM, namePM, obj("PC")),
			group("mr-15m", DomainMR, FormatXML, Period15M, 0, pathMR, nameMR, mr),
		}),
		fileProfile("S0010", "Object directory output", "SHUcom", "SHUcom", []string{"object dirs", "socket"}, []FileGroup{
			group("cm-daily-by-object", DomainCM, FormatXML, Period24H, 1, pathCM+"#Object#/", nameCM, cm),
			group("pm-15m-by-object", DomainPM, FormatCSV, Period15M, 5, pathPM+"#Object#/", namePM, obj("PC")),
			group("mr-15m-by-object", DomainMR, FormatXML, Period15M, 0, pathMR+"#Object#/", nameMR, mr),
		}),
		fileProfile("S0011", "ISAT MTN standard", "ISAT-NBI-MTN", "ISAT-NBI-MTN", []string{"standard"}, []FileGroup{
			group("cm-daily", DomainCM, FormatXML, Period24H, 1, pathCM, nameCM, cm),
			group("pm-15m", DomainPM, FormatCSV, Period15M, 5, pathPM, namePM, obj("PC")),
			group("mr-15m", DomainMR, FormatXML, Period15M, 0, pathMR, nameMR, mr),
		}),
		fileProfile("S0012", "LTE + GNB dual technology", "Shaanxi Mobile", "Shaanxi Mobile", []string{"LTE/GNB"}, []FileGroup{
			group("cm-daily-lte", DomainCM, FormatXML, Period24H, 1, pathCM, nameCM, cm),
			group("cm-daily-gnb", DomainCM, FormatXML, Period24H, 3, pathCM+"GNB/", nameCM, []ScenarioObject{{Code: "CP", Tech: "GNB"}, {Code: "EP", Tech: "GNB"}, {Code: "CC", Tech: "GNB"}, {Code: "CE", Tech: "GNB"}}),
			group("pm-15m-lte", DomainPM, FormatCSV, Period15M, 5, pathPM, namePM, obj("PC")),
			group("pm-pc-15m-gnb", DomainPM, FormatCSV, Period15M, 8, pathPM+"GNB/", namePM, []ScenarioObject{{Code: "PC", Tech: "GNB", Profile: "pm.pc.gnb.csv.v1"}}),
			group("mr-15m", DomainMR, FormatXML, Period15M, 0, pathMR, nameMR, mr),
		}),
		fileProfile("S0013", "pmresult 60M multi-technology", "ZED", "ZED", []string{"LTE/GSM/GNB", "PM 60M", "pmresult"}, []FileGroup{
			group("cm-daily", DomainCM, FormatXML, Period24H, 1, pathCM, nameCM, cm),
			group("pm-pc-60m-lte", DomainPM, FormatCSV, Period60M, 25, pathPM, "pmresult_152XXX_#DataPeriod#_#PeriodStartTime#_#PeriodEndTime#[-#FileID#]", []ScenarioObject{{Code: "PC", Profile: "pm.pc.pmresult.csv.v1"}}),
			group("pm-pc-60m-gsm", DomainPM, FormatCSV, Period60M, 20, pathPM, "pmresult_#LocalHost#_#DataPeriod#_#PeriodStartTime#_#PeriodEndTime#[-#FileID#]", []ScenarioObject{{Code: "PC", Tech: "GSM", Profile: "pm.pc.gsm.pmresult.csv.v1"}}),
			group("pm-pc-60m-gnb", DomainPM, FormatCSV, Period60M, 30, pathPM+"GNB/", "pmresult_XXXXXX_#DataPeriod#_#PeriodStartTime#_#PeriodEndTime#[-#FileID#]", []ScenarioObject{{Code: "PC", Tech: "GNB", Profile: "pm.pc.gnb.csv.v1"}}),
			group("mr-15m", DomainMR, FormatXML, Period15M, 0, pathMR, nameMR, mr),
		}),
		fileProfile("S0014", "Indonesia Telkomsel standard", "YinNi_Telkomsel_MNO", "YinNi_Telkomsel_MNO", []string{"standard"}, []FileGroup{
			group("cm-daily", DomainCM, FormatXML, Period24H, 1, pathCM, nameCM, cm),
			group("pm-15m", DomainPM, FormatCSV, Period15M, 5, pathPM, namePM, obj("PC")),
			group("mr-15m", DomainMR, FormatXML, Period15M, 0, pathMR, nameMR, mr),
		}),
		fileProfile("S0015", "Philippines DITO", "FeiLvBin-DITO", "FeiLvBin-DITO", []string{"PE/PC", "csv |"}, []FileGroup{
			group("cm-daily", DomainCM, FormatXML, Period24H, 1, pathCM, nameCM, cm),
			pipeCSV(group("pm-15m", DomainPM, FormatCSV, Period15M, 5, pathPM, namePM, obj("PE", "PC"))),
			group("mr-15m", DomainMR, FormatXML, Period15M, 0, pathMR, nameMR, mr),
		}),
		fileProfile("S0016", "LTE + GSM PM", "MTN", "MTN", []string{"LTE/GSM"}, []FileGroup{
			group("cm-daily", DomainCM, FormatXML, Period24H, 1, pathCM, nameCM, cm),
			group("pm-15m-lte", DomainPM, FormatCSV, Period15M, 5, pathPM, namePM, obj("PC")),
			group("pm-pc-15m-gsm", DomainPM, FormatCSV, Period15M, 5, pathPM+"gsm/", "pmresult_#LocalHost#_#DataPeriod#_#PeriodStartTime#_#PeriodEndTime#[-#FileID#]", []ScenarioObject{{Code: "PC", Tech: "GSM", Profile: "pm.pc.gsm.pmresult.csv.v1"}}),
			group("mr-15m", DomainMR, FormatXML, Period15M, 0, pathMR, nameMR, mr),
		}),
		fileProfile("S0017", "Heilongjiang standard", "HLongjianng", "HLongjianng", []string{"standard"}, []FileGroup{
			group("cm-daily", DomainCM, FormatXML, Period24H, 1, pathCM, nameCM, cm),
			group("pm-15m", DomainPM, FormatCSV, Period15M, 5, pathPM, namePM, obj("PC")),
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
		field(DomainLOG, "login", "login_time", "log.login_time", "audit_logs.created_at", "datetime", "yyyy-MM-dd HH:mm:ss", "Login time"),
		field(DomainLOG, "login", "username", "log.username", "audit_logs.username", "string", "quote", "Username"),
		field(DomainLOG, "operation", "operation_time", "log.operation_time", "sys_oper_logs.created_at", "datetime", "yyyy-MM-dd HH:mm:ss", "Operation time"),
		field(DomainLOG, "operation", "operator", "log.operator", "sys_oper_logs.username", "string", "quote", "Operator"),
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
	return fields
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

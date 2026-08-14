package provision

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/omcgo/omcgo/internal/config/parammodel"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/quicksettings"
)

// ConfigValidationError identifies a policy configuration that cannot safely
// be converted into concrete TR-069 paths.
type ConfigValidationError struct{ Message string }

func (e *ConfigValidationError) Error() string { return e.Message }

type compiledPolicyConfig struct {
	NetworkType      string
	DataModelVersion string
	Parameters       []ResolvedParameter
	VendorSpecific   string
}

type parameterDefinition struct {
	ID          string
	Template    string
	Type        string
	Required    bool
	Readonly    bool
	MinValue    *int64
	MaxValue    *int64
	EnumOptions []quicksettings.EnumOption
}

var policyFieldAliases = map[string]string{
	// LTE/eNB form fields.
	"cellname":                "CellName",
	"bandssupport":            "BandSupport",
	"bandwidth":               "DLBandWidth",
	"frequency":               "DLEarfcn",
	"subframeassignment":      "SubFrameAssignment",
	"specialsubframepatterns": "SpecialSubFramePatterns",
	"tac":                     "TAC",
	"cellidentity":            "ECI",
	"phycellid":               "PCI",
	"rootsequenceindex":       "RootSequenceIndex",
	"totaltxpower":            "MaxTxPower",
	"wanip":                   "WAN_CONFIG1_IPADDR",
	// NR/gNB form and CMCC workbook fields.
	"gnbname":     "gNBName",
	"gnbid":       "gNBId",
	"gnbidlength": "gNBIdLength",
	// Keep compatibility with the historical CMCC workbook header "*gNB Lenth".
	"gnblenth":                     "gNBIdLength",
	"pci":                          "PCI",
	"freqbandindicator":            "Band",
	"bandindicator":                "Band",
	"nrarfcndl":                    "NRARFCNDL",
	"nrarfcnndl":                   "NRARFCNDL",
	"nrarfcnul":                    "NRARFCNUL",
	"dlbandwidth":                  "DLCarrierBandWidth",
	"ulbandwidth":                  "ULCarrierBandWidth",
	"dlantnum":                     "NumOfTxAntenna",
	"ulantnum":                     "NumOfRxAntenna",
	"ssbfrequency":                 "SsbFrequency",
	"offsettopointa":               "OffsetToPointA",
	"kssb":                         "SsbSubcarrierOffset",
	"ssbsubcarrieroffset":          "SsbSubcarrierOffset",
	"nci":                          "NrcellIdentity",
	"plmnid":                       "PLMNID",
	"amfip":                        "AmfIP1",
	"amfipdefault":                 "AmfIP1",
	"primary":                      "IsPrimaryPlmn",
	"ngubindinterface":             "BindInterface",
	"powermodify":                  "PowerModify",
	"subcarrierspacingul":          "ULSubCarrierSpacing",
	"subcarrierspacingdl":          "DLSubCarrierSpacing",
	"dlultransmissionperiodicity1": "DlULTransmissionPeriodicity",
	"nrofdownlinkslots1":           "NrofDownlinkSlots",
	"nrofdownlinksymbols1":         "NrofDownlinkSymbols",
	"nrofuplinkslots1":             "NrofUplinkSlots",
	"nrofuplinksymbols1":           "NrofUplinkSymbols",
	"dlultransmissionperiodicity2": "Pat2DlULTransmissionPeriodicity",
	"nrofdownlinkslots2":           "Pat2NrofDownlinkSlots",
	"nrofdownlinksymbols2":         "Pat2NrofDownlinkSymbols",
	"nrofuplinkslots2":             "Pat2NrofUplinkSlots",
	"nrofuplinksymbols2":           "Pat2NrofUplinkSymbols",
	"leftauth":                     "TUNNEL_LEFT_AUTH",
	"rightauth":                    "TUNNEL_RIGHT_AUTH",
	"leftsubnet":                   "LEFTSUBNET",
	"fragmentation":                "TUNNEL_FRAGMENTATION",
	// 中国移动默认规划模板 DEVICE 工作表字段。
	"ntpenable":              "Enable",
	"localtimezone":          "LocalTimeZoneName",
	"periodicinformenable":   "PeriodicInformEnable",
	"periodicinforminterval": "PeriodicInformInterval",
	"synchronizationmode":    "tfcsManagerPrimsrc",
	"1588enable":             "X_COM_1588SyncEnable",
	"syncmode":               "X_COM_PTP1588syncsMode",
	"domain":                 "X_COM_PTP1588DomainNum",
	"unicastserveripaddress": "X_COM_PTP1588UnicastAddr",
	"synchronization":        "PpsTimeMode",
	"ppstimemode":            "PpsTimeMode",
}

// Some quick-settings parameter names are repeated in different groups. These
// workbook fields therefore bind to an exact registered standard path instead
// of the ambiguous leaf/name.
var policyFieldPathAliases = map[string]string{
	"ntpenable": "Device.Time.Enable",
	"tac":       "Device.Services.FAPService.{i}.CellConfig.{i}.NR.CN.TA.{i}.TAC",
	"plmnid":    "Device.Services.FAPService.{i}.CellConfig.{i}.NR.CN.TA.{i}.PLMNList.{i}.PLMNID",
	"url":       "Device.ManagementServer.URL",
}

var servingPLMNPattern = regexp.MustCompile(`^\d{5,6}$`)

var ignoredPolicyFields = map[string]struct{}{
	"id": {}, "devicetype": {}, "serialnumber": {}, "updatedby": {}, "updatedat": {},
	"sheetparameters": {}, "customparams": {}, "amflist": {}, "plmnconfiglist": {},
	// Workbook instance-control columns select concrete TR-069 object instances.
	// They are metadata, not parameters to be sent to the device.
	"cellindex": {}, "cellnumber": {}, "btsindex": {}, "plmnindex": {},
	"neighborindex": {}, "carrierindex": {}, "trxindex": {},
	"interfaceindex": {}, "tunnelindex": {},
	// Removed Plug-and-Play planning fields. Ignore historical policy values so
	// they cannot be resolved through a product-specific alias and downlinked.
	"timezoneterm": {}, "omcip": {},
}

// CompilePolicyParameters resolves the current UI/import snapshot against
// quick-settings definitions only. Production execution uses
// CompilePolicyParametersWithMappings to add product parameter-table fallback.
func CompilePolicyParameters(
	policy *PlugAndPlayPolicy,
	device *model.Device,
	paramModel string,
	groups []quicksettings.Group,
) (*compiledPolicyConfig, error) {
	return CompilePolicyParametersWithMappings(policy, device, paramModel, groups, nil)
}

// CompilePolicyParametersWithMappings uses quick-settings definitions first
// and supplements missing definitions from the parameter mappings attached to
// the device's product. The mappings should come from ParamRegistry.GetByProduct
// so discovered mappings remain ahead of the product's default parameter table.
func CompilePolicyParametersWithMappings(
	policy *PlugAndPlayPolicy,
	device *model.Device,
	paramModel string,
	groups []quicksettings.Group,
	mappings []parammodel.ParamMapping,
) (*compiledPolicyConfig, error) {
	if policy == nil || device == nil {
		return nil, &ConfigValidationError{Message: "policy and device are required"}
	}
	if !policy.SelfConfigEnabled {
		return nil, &ConfigValidationError{Message: "policy does not enable parameter self-configuration"}
	}
	if len(groups) == 0 && len(mappings) == 0 {
		return nil, &ConfigValidationError{Message: "device product parameter definitions are unavailable"}
	}

	var root map[string]any
	if err := json.Unmarshal(policy.Config, &root); err != nil {
		return nil, &ConfigValidationError{Message: fmt.Sprintf("decode policy config: %v", err)}
	}
	rows := mapSlice(root["paramConfigList"])
	selected := make([]map[string]any, 0)
	for _, row := range rows {
		if strings.TrimSpace(valueString(row["serialNumber"])) == device.SerialNumber {
			selected = append(selected, row)
		}
	}
	if len(selected) == 0 {
		return nil, &ConfigValidationError{Message: fmt.Sprintf(
			"policy has no parameter configuration for device %s", device.SerialNumber)}
	}
	if len(selected) > 1 {
		return nil, &ConfigValidationError{Message: fmt.Sprintf(
			"policy has multiple top-level parameter configurations for device %s", device.SerialNumber)}
	}

	definitions, aliases := buildParameterDefinitions(groups, mappings)
	networkType, defaultVersion := networkProfile(device, paramModel)
	omitDuplexMode := networkType == "NR"
	compiled := make(map[string]ResolvedParameter)
	for rowIndex, row := range selected {
		compileRow := row
		if omitDuplexMode {
			compileRow = withoutWorkbookFieldMappings(row, "duplexmode")
		}
		rowAliases, err := applyWorkbookParameterMappings(compileRow, definitions, aliases)
		if err != nil {
			return nil, err
		}
		cellIndex := 1
		hasSheetData, err := compileSheetParameters(
			compileRow, definitions, rowAliases, cellIndex, networkType, omitDuplexMode, compiled,
		)
		if err != nil {
			return nil, err
		}
		hasAuthoritativeWorkbook := hasSheetData && len(mapSlice(compileRow["workbookMappings"])) > 0
		for key, raw := range compileRow {
			normalized := normalizeParameterKey(key)
			if _, ignored := ignoredPolicyFields[normalized]; ignored {
				continue
			}
			if omitDuplexMode && normalized == "duplexmode" {
				continue
			}
			if hasAuthoritativeWorkbook {
				continue
			}
			if hasSheetData && isWorkbookSummaryField(normalized) {
				continue
			}
			if !isScalarValue(raw) || valueString(raw) == "" {
				continue
			}
			if err := compileValue(key, raw, "page", fmt.Sprintf("paramConfigList[%d].%s", rowIndex, key),
				cellIndex, 1, networkType, definitions, aliases, compiled); err != nil {
				return nil, err
			}
		}
		if !hasAuthoritativeWorkbook {
			if err := compileListValues(
				compileRow, rowIndex, cellIndex, networkType, definitions, aliases, compiled,
			); err != nil {
				return nil, err
			}
		}
		if err := compileCustomParameters(compileRow, rowIndex, definitions, compiled); err != nil {
			return nil, err
		}
	}
	if len(compiled) == 0 {
		return nil, &ConfigValidationError{Message: fmt.Sprintf(
			"no registered quick-settings parameters were resolved for device %s", device.SerialNumber)}
	}

	params := make([]ResolvedParameter, 0, len(compiled))
	for _, param := range compiled {
		params = append(params, param)
	}
	sort.Slice(params, func(i, j int) bool { return params[i].TRPath < params[j].TRPath })

	version := strings.TrimSpace(valueString(root["dataModelVersion"]))
	if version == "" {
		version = defaultVersion
	}
	vendorSpecific := strings.TrimSpace(valueString(root["vendorSpecific"]))
	if vendorSpecific != "" {
		vendorSpecific = base64.StdEncoding.EncodeToString([]byte(vendorSpecific))
	}
	return &compiledPolicyConfig{
		NetworkType: networkType, DataModelVersion: version,
		Parameters: params, VendorSpecific: vendorSpecific,
	}, nil
}

func withoutWorkbookFieldMappings(row map[string]any, normalizedHeader string) map[string]any {
	mappings, ok := row["workbookMappings"].([]any)
	if !ok || len(mappings) == 0 {
		return row
	}
	filtered := make([]any, 0, len(mappings))
	for _, raw := range mappings {
		mapping, ok := raw.(map[string]any)
		if ok && normalizeParameterKey(valueString(mapping["header"])) == normalizedHeader {
			continue
		}
		filtered = append(filtered, raw)
	}
	if len(filtered) == len(mappings) {
		return row
	}
	copy := make(map[string]any, len(row))
	for key, value := range row {
		copy[key] = value
	}
	copy["workbookMappings"] = filtered
	return copy
}

func applyWorkbookParameterMappings(
	row map[string]any,
	definitions map[string]parameterDefinition,
	base map[string]string,
) (map[string]string, error) {
	aliases := make(map[string]string, len(base))
	for key, value := range base {
		aliases[key] = value
	}
	for _, mapping := range mapSlice(row["workbookMappings"]) {
		header := strings.TrimSpace(valueString(mapping["header"]))
		sheet := strings.TrimSpace(valueString(mapping["sheet"]))
		path := strings.TrimSpace(valueString(mapping["trPath"]))
		if header == "" && path == "" {
			continue
		}
		if header == "" || path == "" {
			return nil, &ConfigValidationError{Message: "workbook parameter mapping requires header and TRPath"}
		}
		if !workbookMappingHasValue(row, sheet, header) {
			continue
		}
		definitionKey := ""
		for key, definition := range definitions {
			if path == definition.Template {
				definitionKey = key
				break
			}
			if concretePathMatchesTemplate(path, definition.Template) {
				definitionKey = "workbook:" + path
				definition.Template = path
				definitions[definitionKey] = definition
				break
			}
		}
		if definitionKey == "" {
			return nil, &ConfigValidationError{Message: fmt.Sprintf(
				"workbook parameter mapping path is not registered in quick settings or product parameter mappings: %s", path)}
		}
		alias := normalizeParameterKey(header)
		if sheet != "" {
			qualifiedAlias := normalizeParameterKey(sheet + "." + header)
			if previous, exists := aliases[qualifiedAlias]; exists && previous != definitionKey {
				return nil, &ConfigValidationError{Message: fmt.Sprintf(
					"workbook parameter column %s maps to conflicting TRPaths", header)}
			}
			aliases[qualifiedAlias] = definitionKey
			continue
		}
		if previous, exists := aliases[alias]; exists && previous != definitionKey {
			return nil, &ConfigValidationError{Message: fmt.Sprintf(
				"workbook parameter column %s maps to conflicting TRPaths", header)}
		}
		aliases[alias] = definitionKey
	}
	return aliases, nil
}

func workbookMappingHasValue(row map[string]any, sheet, header string) bool {
	wantedSheet := normalizeParameterKey(sheet)
	wantedHeader := normalizeParameterKey(header)
	for sheetName, rawRows := range mapValue(row["sheetParameters"]) {
		if wantedSheet != "" && normalizeParameterKey(sheetName) != wantedSheet {
			continue
		}
		for _, sheetRow := range mapSlice(rawRows) {
			for column, value := range sheetRow {
				if normalizeParameterKey(column) == wantedHeader && valueString(value) != "" {
					return true
				}
			}
		}
	}
	return false
}

func buildParameterDefinitions(
	groups []quicksettings.Group,
	mappings []parammodel.ParamMapping,
) (map[string]parameterDefinition, map[string]string) {
	definitions := make(map[string]parameterDefinition)
	aliases := make(map[string]string)
	keysByName := make(map[string][]string)
	quickSettingKeysByName := make(map[string][]string)
	quickSettingKeysByAlias := make(map[string][]string)
	collisions := make(map[string]struct{})
	addKey := func(target map[string][]string, name, definitionKey string) {
		for _, existing := range target[name] {
			if existing == definitionKey {
				return
			}
		}
		target[name] = append(target[name], definitionKey)
	}
	addAlias := func(alias, definitionKey string) {
		aliasKey := normalizeParameterKey(alias)
		if aliasKey == "" {
			return
		}
		if previous, exists := aliases[aliasKey]; exists && previous != definitionKey {
			delete(aliases, aliasKey)
			collisions[aliasKey] = struct{}{}
			return
		}
		if _, collided := collisions[aliasKey]; !collided {
			aliases[aliasKey] = definitionKey
		}
	}
	for _, mapping := range mappings {
		if !mapping.IsSupported || !strings.EqualFold(mapping.EntryType, "parameter") {
			continue
		}
		template := strings.TrimSpace(mapping.StandardPath)
		if template == "" {
			template = strings.TrimSpace(mapping.PrivatePath)
		}
		if template == "" {
			continue
		}
		name := parameterLeaf(template)
		definitions[template] = parameterDefinition{
			ID: name, Template: template, Type: mapping.DataType,
			Readonly: strings.EqualFold(mapping.Access, "READ_ONLY") ||
				strings.EqualFold(mapping.Access, "readonly"),
			MinValue: mapping.MinValue, MaxValue: mapping.MaxValue,
			EnumOptions: mappingEnumOptions(mapping),
		}
		addKey(keysByName, name, template)
		addAlias(name, template)
		addAlias(template, template)
	}
	for _, group := range groups {
		for _, param := range group.Params {
			template := strings.TrimSpace(param.StandardPath)
			if template == "" && group.ObjectPath != "" && param.Leaf != "" {
				template = group.ObjectPath + param.Leaf
			}
			if template == "" {
				continue
			}
			definitionKey := template
			definition := parameterDefinition{
				ID: param.Name, Template: template, Type: param.Type, Required: param.Required,
				Readonly: param.Readonly, MinValue: param.MinValue, MaxValue: param.MaxValue,
				EnumOptions: param.EnumOptions,
			}
			if productDefinition, exists := definitions[definitionKey]; exists {
				// Product mappings describe the device wire contract. Quick settings
				// may add presentation metadata, but must not replace product-specific
				// types, ranges or enum wire values (for example BLQ uses 50 while
				// ENB_DEFAULT uses n50 for the same LTE bandwidth path).
				if productDefinition.Type != "" {
					definition.Type = productDefinition.Type
				}
				definition.Readonly = definition.Readonly || productDefinition.Readonly
				if productDefinition.MinValue != nil {
					definition.MinValue = productDefinition.MinValue
				}
				if productDefinition.MaxValue != nil {
					definition.MaxValue = productDefinition.MaxValue
				}
				if len(productDefinition.EnumOptions) > 0 {
					definition.EnumOptions = productDefinition.EnumOptions
				}
			}
			definitions[definitionKey] = definition
			addKey(keysByName, param.Name, definitionKey)
			addKey(quickSettingKeysByName, param.Name, definitionKey)
			for _, alias := range []string{param.Name, param.TitleZh, param.TitleEn} {
				aliasKey := normalizeParameterKey(alias)
				if aliasKey != "" {
					addKey(quickSettingKeysByAlias, aliasKey, definitionKey)
				}
			}
			addAlias(param.Name, definitionKey)
			addAlias(param.TitleZh, definitionKey)
			addAlias(param.TitleEn, definitionKey)
		}
	}
	// Quick Settings is the curated business-facing subset and therefore wins
	// over ambiguous leaf names in the product's full parameter table. Keep an
	// alias ambiguous only when Quick Settings itself maps it to multiple paths.
	for alias, keys := range quickSettingKeysByAlias {
		if len(keys) == 1 {
			aliases[alias] = keys[0]
		}
	}
	for alias, name := range policyFieldAliases {
		keys := quickSettingKeysByName[name]
		if len(keys) == 0 {
			keys = keysByName[name]
		}
		if len(keys) == 1 {
			aliases[alias] = keys[0]
		}
	}
	for alias, path := range policyFieldPathAliases {
		if _, exists := definitions[path]; exists {
			aliases[alias] = path
		}
	}
	// The shared GSM workbook has business-facing column names, while BTS and
	// BSC expose different paths. Bind those columns only when the selected
	// product model contains the corresponding path.
	if path := "DeviceGSM.Bts.{i}.IpaUnitId"; definitions[path].Template != "" {
		aliases["ipaunitid"] = path
	}
	if path := "DeviceGSM.Bts.{i}.IpaRslIp"; definitions[path].Template != "" {
		aliases["remoteip"] = path
	}
	return definitions, aliases
}

func parameterLeaf(path string) string {
	path = strings.TrimSuffix(strings.TrimSpace(path), ".")
	if index := strings.LastIndex(path, "."); index >= 0 {
		return path[index+1:]
	}
	return path
}

func mappingEnumOptions(mapping parammodel.ParamMapping) []quicksettings.EnumOption {
	if mapping.EnumValues == nil || strings.TrimSpace(*mapping.EnumValues) == "" {
		return nil
	}
	values := strings.Split(*mapping.EnumValues, ",")
	var labels []string
	if mapping.EnumLabels != nil {
		labels = strings.Split(*mapping.EnumLabels, ",")
	}
	options := make([]quicksettings.EnumOption, 0, len(values))
	for index, raw := range values {
		value := strings.TrimSpace(raw)
		if value == "" {
			continue
		}
		label := value
		if index < len(labels) && strings.TrimSpace(labels[index]) != "" {
			label = strings.TrimSpace(labels[index])
		}
		options = append(options, quicksettings.EnumOption{Value: value, Label: label})
	}
	return options
}

func compileSheetParameters(
	row map[string]any,
	definitions map[string]parameterDefinition,
	aliases map[string]string,
	defaultCell int,
	networkType string,
	omitDuplexMode bool,
	compiled map[string]ResolvedParameter,
) (bool, error) {
	sheets, ok := row["sheetParameters"].(map[string]any)
	if !ok || len(sheets) == 0 {
		return false, nil
	}
	strictWorkbookMappings := len(mapSlice(row["workbookMappings"])) > 0
	for sheetName, rawRows := range sheets {
		seenPrimary := make(map[int]struct{})
		sheetRows := mapSlice(rawRows)
		for rowIndex, sheetRow := range sheetRows {
			cellIndex, listIndex, err := workbookRowInstances(
				sheetName, sheetRow, rowIndex, len(sheetRows), defaultCell, networkType,
			)
			if err != nil {
				return true, err
			}
			if isPrimaryInstanceSheet(sheetName, networkType) {
				if _, duplicate := seenPrimary[cellIndex]; duplicate {
					return true, &ConfigValidationError{Message: fmt.Sprintf(
						"duplicate primary instance %d in %s", cellIndex, sheetName)}
				}
				seenPrimary[cellIndex] = struct{}{}
			}
			handled := make(map[string]struct{})
			ipa, hasIPA := valueByNormalizedKey(sheetRow, "ipa")
			unitID, hasUnitID := valueByNormalizedKey(sheetRow, "unitid")
			if !strictWorkbookMappings && hasIPA && hasUnitID && valueString(ipa) != "" && valueString(unitID) != "" {
				if id, exists := aliases["ipaunitid"]; exists && (strings.Contains(definitions[id].Template, "GsmBTSCellDT") ||
					strings.Contains(definitions[id].Template, "DeviceGSM.Bts.{i}")) {
					if err := compileValue("IPAUnitID", fmt.Sprintf("%s-%s", valueString(ipa), valueString(unitID)),
						"import", fmt.Sprintf("%s[%d].IPAUnitID", sheetName, rowIndex+1),
						cellIndex, listIndex, networkType, definitions, aliases, compiled); err != nil {
						return true, err
					}
					handled["ipa"] = struct{}{}
					handled["unitid"] = struct{}{}
				}
			}
			for header, value := range sheetRow {
				normalized := normalizeParameterKey(header)
				if omitDuplexMode && normalized == "duplexmode" {
					continue
				}
				if _, alreadyHandled := handled[normalized]; alreadyHandled {
					continue
				}
				if _, ignored := ignoredPolicyFields[normalized]; ignored {
					continue
				}
				if !isScalarValue(value) || valueString(value) == "" {
					continue
				}
				qualifiedKey := normalizeParameterKey(sheetName + "." + header)
				if strictWorkbookMappings {
					if _, hasQualifiedMapping := aliases[qualifiedKey]; !hasQualifiedMapping {
						continue
					}
				}
				compileKey := header
				if _, exists := aliases[qualifiedKey]; exists {
					compileKey = sheetName + "." + header
				}
				if err := compileValue(compileKey, value, "import",
					fmt.Sprintf("%s[%d].%s", sheetName, rowIndex+1, header),
					cellIndex, listIndex, networkType, definitions, aliases, compiled); err != nil {
					return true, err
				}
			}
		}
	}
	return true, nil
}

func workbookRowInstances(
	sheetName string,
	row map[string]any,
	rowIndex, rowCount, defaultPrimary int,
	networkType string,
) (int, int, error) {
	primary := defaultPrimary
	primaryValue, hasPrimary := firstNormalizedValue(row, "cellindex", "cellnumber", "btsindex")
	if hasPrimary && strings.TrimSpace(valueString(primaryValue)) != "" {
		parsed, err := positiveInstanceIndex(primaryValue)
		if err != nil {
			return 0, 0, &ConfigValidationError{Message: fmt.Sprintf(
				"invalid primary instance in %s[%d]: %v", sheetName, rowIndex+1, err)}
		}
		primary = parsed
	} else if isPrimaryInstanceSheet(sheetName, networkType) && rowCount > 1 {
		return 0, 0, &ConfigValidationError{Message: fmt.Sprintf(
			"%s[%d] requires an explicit instance index", sheetName, rowIndex+1)}
	}

	listIndex := rowIndex + 1
	if raw, ok := firstNormalizedValue(
		row, "plmnindex", "neighborindex", "carrierindex", "trxindex", "interfaceindex", "tunnelindex",
	); ok && strings.TrimSpace(valueString(raw)) != "" {
		parsed, err := positiveInstanceIndex(raw)
		if err != nil {
			return 0, 0, &ConfigValidationError{Message: fmt.Sprintf(
				"invalid child instance in %s[%d]: %v", sheetName, rowIndex+1, err)}
		}
		listIndex = parsed
	}
	return primary, listIndex, nil
}

func firstNormalizedValue(values map[string]any, wanted ...string) (any, bool) {
	for _, key := range wanted {
		if value, exists := valueByNormalizedKey(values, key); exists {
			return value, true
		}
	}
	return nil, false
}

func positiveInstanceIndex(raw any) (int, error) {
	text := strings.TrimSpace(valueString(raw))
	value, err := strconv.Atoi(text)
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("instance index must be a positive integer, got %q", text)
	}
	return value, nil
}

func isPrimaryInstanceSheet(sheetName, networkType string) bool {
	normalized := normalizeParameterKey(sheetName)
	switch networkType {
	case "NR", "LTE":
		return normalized == "cell"
	case "GSM":
		return normalized == "gsm" || normalized == "gsmbts"
	default:
		return false
	}
}

func valueByNormalizedKey(values map[string]any, wanted string) (any, bool) {
	for key, value := range values {
		if normalizeParameterKey(key) == wanted {
			return value, true
		}
	}
	return nil, false
}

func compileListValues(
	row map[string]any,
	rowIndex, cellIndex int,
	networkType string,
	definitions map[string]parameterDefinition,
	aliases map[string]string,
	compiled map[string]ResolvedParameter,
) error {
	for index, item := range mapSlice(row["amfList"]) {
		if value := item["amfIp"]; valueString(value) != "" {
			if err := compileValue("AmfIP1", value, "page",
				fmt.Sprintf("paramConfigList[%d].amfList[%d].amfIp", rowIndex, index),
				cellIndex, index+1, networkType, definitions, aliases, compiled); err != nil {
				return err
			}
		}
	}
	plmnItems := mapSlice(row["plmnConfigList"])
	if definitionKey, usesAggregateServingPLMN := aliases[normalizeParameterKey("ExistPlmnidList")]; usesAggregateServingPLMN {
		values := make([]string, 0, len(plmnItems))
		seen := make(map[string]struct{}, len(plmnItems))
		for _, item := range plmnItems {
			if value := strings.TrimSpace(valueString(item["plmnId"])); value != "" {
				if !servingPLMNPattern.MatchString(value) {
					return &ConfigValidationError{Message: fmt.Sprintf("invalid serving PLMN %q", value)}
				}
				if _, duplicate := seen[value]; duplicate {
					return &ConfigValidationError{Message: fmt.Sprintf("duplicate serving PLMN %q", value)}
				}
				seen[value] = struct{}{}
				values = append(values, value)
			}
		}
		if len(values) > 0 {
			definition := definitions[definitionKey]
			if definition.MaxValue != nil && int64(len(values)) > *definition.MaxValue {
				return &ConfigValidationError{Message: fmt.Sprintf(
					"serving PLMN list exceeds maximum %d", *definition.MaxValue)}
			}
			path, err := resolveInstancePath(definition.Template, cellIndex, 1, networkType)
			if err != nil {
				return &ConfigValidationError{Message: fmt.Sprintf(
					"ExistPlmnidList at paramConfigList[%d].plmnConfigList: %v", rowIndex, err)}
			}
			return putResolved(compiled, ResolvedParameter{
				ParameterID: definition.ID,
				TRPath:      path,
				Value:       strings.Join(values, ","),
				Source:      "page",
				SourceLocation: fmt.Sprintf(
					"paramConfigList[%d].plmnConfigList", rowIndex),
			})
		}
	}
	for index, item := range plmnItems {
		if value := item["plmnId"]; valueString(value) != "" {
			if err := compileValue("PLMNID", value, "page",
				fmt.Sprintf("paramConfigList[%d].plmnConfigList[%d].plmnId", rowIndex, index),
				cellIndex, index+1, networkType, definitions, aliases, compiled); err != nil {
				return err
			}
		}
	}
	return nil
}

func compileCustomParameters(
	row map[string]any,
	rowIndex int,
	definitions map[string]parameterDefinition,
	compiled map[string]ResolvedParameter,
) error {
	for index, item := range mapSlice(row["customParams"]) {
		path := strings.TrimSpace(valueString(item["trPath"]))
		value := valueString(item["value"])
		if path == "" || value == "" {
			continue
		}
		if isDeviceMatchOnlyParameter(path) {
			continue
		}
		if existing, exists := compiled[path]; exists && existing.Source == "import" {
			// A workbook column with an explicit parameter mapping is the
			// canonical public-parameter value. Old editor snapshots may retain
			// the same path in customParams; do not let that stale copy override
			// or conflict with the mapped value.
			continue
		}
		var matched *parameterDefinition
		for _, definition := range definitions {
			if concretePathMatchesTemplate(path, definition.Template) {
				copy := definition
				matched = &copy
				break
			}
		}
		if matched == nil {
			return &ConfigValidationError{Message: fmt.Sprintf(
				"custom parameter path is not registered in quick settings: %s", path)}
		}
		if strings.Contains(path, "{i}") {
			return &ConfigValidationError{Message: fmt.Sprintf("custom parameter path contains unresolved instance: %s", path)}
		}
		if err := putResolved(compiled, ResolvedParameter{
			ParameterID: matched.ID, TRPath: path, Value: value, Source: "page",
			SourceLocation: fmt.Sprintf("paramConfigList[%d].customParams[%d]", rowIndex, index),
		}); err != nil {
			return err
		}
	}
	return nil
}

func compileValue(
	key string,
	raw any,
	source, location string,
	cellIndex, listIndex int,
	networkType string,
	definitions map[string]parameterDefinition,
	aliases map[string]string,
	compiled map[string]ResolvedParameter,
) error {
	normalized := normalizeParameterKey(key)
	if normalized == "serialnumber" {
		return nil
	}
	id, exists := aliases[normalized]
	if !exists {
		if source == "import" {
			return &ConfigValidationError{Message: fmt.Sprintf(
				"imported parameter %s at %s is not registered in quick settings or product parameter mappings",
				key, location)}
		}
		return nil
	}
	definition := definitions[id]
	if definition.Readonly {
		return &ConfigValidationError{Message: fmt.Sprintf("%s is read-only", id)}
	}
	value, err := convertParameterValue(raw, definition)
	if err != nil {
		return &ConfigValidationError{Message: fmt.Sprintf("%s at %s: %v", id, location, err)}
	}
	path, err := resolveInstancePath(definition.Template, cellIndex, listIndex, networkType)
	if err != nil {
		return &ConfigValidationError{Message: fmt.Sprintf("%s at %s: %v", id, location, err)}
	}
	return putResolved(compiled, ResolvedParameter{
		ParameterID: id, TRPath: path, Value: value, Source: source, SourceLocation: location,
	})
}

func isDeviceMatchOnlyParameter(path string) bool {
	return normalizeParameterKey(parameterLeaf(path)) == "serialnumber"
}

func putResolved(target map[string]ResolvedParameter, value ResolvedParameter) error {
	if previous, exists := target[value.TRPath]; exists {
		if previous.Value != value.Value {
			return &ConfigValidationError{Message: fmt.Sprintf(
				"conflicting values for %s (%s=%q, %s=%q)",
				value.TRPath, previous.SourceLocation, previous.Value, value.SourceLocation, value.Value)}
		}
		return nil
	}
	target[value.TRPath] = value
	return nil
}

func resolveInstancePath(template string, cellIndex, listIndex int, networkType string) (string, error) {
	path := template
	fapServiceIndex := 1
	if networkType == "LTE" {
		fapServiceIndex = cellIndex
	}
	replacements := []struct {
		pattern string
		value   int
	}{
		{"FAPService.{i}", fapServiceIndex},
		{"CellConfig.{i}", cellIndex},
		{"NeighborList.NRCell.{i}", listIndex},
		{"NeighborList.LTECell.{i}", listIndex},
		{"Neighbor4G.{i}", listIndex},
		{"Neighbor2G.{i}", listIndex},
		{"InterFreq.Carrier.{i}", listIndex},
		{"Trx.{i}", listIndex},
		{"AMFPoolConfigParam.{i}", listIndex},
		{"ScsSpecificCarrierList.{i}", 1},
		{"MultiFrequencyBandListNRSIB.{i}", 1},
		{"PLMNList.{i}", listIndex},
		{"TA.{i}", 1},
		{"NguIpBind{i}", listIndex},
		{"FAP.Ipsec.{i}", listIndex},
		{"GsmBTSCellDT.{i}", cellIndex},
		{"DeviceGSM.Bts.{i}", cellIndex},
	}
	for _, replacement := range replacements {
		path = strings.ReplaceAll(path, replacement.pattern,
			strings.ReplaceAll(replacement.pattern, "{i}", strconv.Itoa(replacement.value)))
	}
	if strings.Contains(path, "{i}") {
		return "", fmt.Errorf("cannot resolve every instance in TR path template %s", template)
	}
	return path, nil
}

func convertParameterValue(raw any, definition parameterDefinition) (string, error) {
	value := valueString(raw)
	if len(definition.EnumOptions) > 0 {
		matched := false
		for _, option := range definition.EnumOptions {
			if value == option.Value || strings.EqualFold(value, option.Label) {
				value = option.Value
				matched = true
				break
			}
		}
		// Compatibility for saved LTE bandwidth policies created by the old
		// product-agnostic UI. Convert only when the target product explicitly
		// declares the counterpart as a legal enum value.
		if !matched && isLegacyLTEBandwidthValue(value) {
			candidate := strings.TrimPrefix(strings.ToLower(value), "n")
			for _, option := range definition.EnumOptions {
				optionCandidate := strings.TrimPrefix(strings.ToLower(option.Value), "n")
				if isLegacyLTEBandwidthValue(option.Value) && candidate == optionCandidate {
					value = option.Value
					matched = true
					break
				}
			}
		}
		if !matched {
			return "", fmt.Errorf("expected one of the configured enum values")
		}
	}
	if definition.MinValue != nil || definition.MaxValue != nil {
		number, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return "", fmt.Errorf("expected an integer")
		}
		if definition.MinValue != nil && number < *definition.MinValue {
			return "", fmt.Errorf("value is below minimum %d", *definition.MinValue)
		}
		if definition.MaxValue != nil && number > *definition.MaxValue {
			return "", fmt.Errorf("value exceeds maximum %d", *definition.MaxValue)
		}
	}
	return value, nil
}

func isLegacyLTEBandwidthValue(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "25", "50", "75", "100", "n25", "n50", "n75", "n100":
		return true
	default:
		return false
	}
}

func networkProfile(device *model.Device, paramModel string) (string, string) {
	switch device.Technology {
	case model.TechLTE:
		return "LTE", "v1.0"
	case model.TechGSM:
		return "GSM", "v1.0"
	case model.TechNR:
		return "NR", "v1.7"
	}
	if strings.Contains(strings.ToLower(paramModel), "baibnq") {
		return "NR", "v1.7"
	}
	return "LTE", "v1.0"
}

func concretePathMatchesTemplate(path, template string) bool {
	pattern := "^" + regexp.QuoteMeta(template) + "$"
	pattern = strings.ReplaceAll(pattern, regexp.QuoteMeta("{i}"), `[1-9][0-9]*`)
	return regexp.MustCompile(pattern).MatchString(path)
}

func normalizeParameterKey(value string) string {
	value = strings.TrimSpace(strings.TrimPrefix(value, "*"))
	var result strings.Builder
	for _, r := range strings.ToLower(value) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || (r >= '\u4e00' && r <= '\u9fff') {
			result.WriteRune(r)
		}
	}
	return result.String()
}

func valueString(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(typed)
	case bool:
		if typed {
			return "1"
		}
		return "0"
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64)
	case json.Number:
		return typed.String()
	default:
		return strings.TrimSpace(fmt.Sprint(typed))
	}
}

func mapSlice(value any) []map[string]any {
	raw, ok := value.([]any)
	if !ok {
		return nil
	}
	result := make([]map[string]any, 0, len(raw))
	for _, item := range raw {
		if mapped, ok := item.(map[string]any); ok {
			result = append(result, mapped)
		}
	}
	return result
}

func isScalarValue(value any) bool {
	switch value.(type) {
	case string, bool, float64, json.Number:
		return true
	default:
		return false
	}
}

func isWorkbookSummaryField(key string) bool {
	switch key {
	case "cellname", "bandssupport", "bandwidth", "frequency", "subframeassignment":
		return true
	default:
		return false
	}
}

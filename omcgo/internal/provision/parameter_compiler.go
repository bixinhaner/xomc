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
	"gnbname":                      "gNBName",
	"gnbid":                        "gNBId",
	"gnbidlength":                  "gNBIdLength",
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
	"timezoneterm":           "LocalTimeZoneName",
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
}

var servingPLMNPattern = regexp.MustCompile(`^\d{5,6}$`)

var ignoredPolicyFields = map[string]struct{}{
	"id": {}, "devicetype": {}, "serialnumber": {}, "updatedby": {}, "updatedat": {},
	"sheetparameters": {}, "customparams": {}, "amflist": {}, "plmnconfiglist": {},
}

// These planning-workbook fields are shared across product families, but a
// particular device model may not expose a writable TR-069 parameter for one
// of them. Preserve them in the policy while compiling every supported field.
var optionalPlanningFields = map[string]struct{}{
	"ipa": {}, "bindip": {}, "wanip": {}, "synchronization": {}, "omc": {}, "omcip": {},
	// The shared eNB workbook always carries the complete PTP/1588 section,
	// while individual product models expose only the subset they support.
	// Keep unsupported planning values in the policy snapshot and compile the
	// supported ones through quick-settings/product mappings.
	"1588enable": {}, "syncmode": {}, "modeswitch": {}, "domain": {},
	"syncinterval": {}, "delayinterval": {}, "asymmetry": {}, "startuptime": {},
	"unicastserveripaddress": {},
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

	definitions, aliases := buildParameterDefinitions(groups, mappings)
	compiled := make(map[string]ResolvedParameter)
	for rowIndex, row := range selected {
		cellIndex := rowIndex + 1
		hasSheetData, err := compileSheetParameters(row, definitions, aliases, cellIndex, compiled)
		if err != nil {
			return nil, err
		}
		for key, raw := range row {
			normalized := normalizeParameterKey(key)
			if _, ignored := ignoredPolicyFields[normalized]; ignored {
				continue
			}
			if hasSheetData && isWorkbookSummaryField(normalized) {
				continue
			}
			if !isScalarValue(raw) || valueString(raw) == "" {
				continue
			}
			if err := compileValue(key, raw, "page", fmt.Sprintf("paramConfigList[%d].%s", rowIndex, key),
				cellIndex, 1, definitions, aliases, compiled); err != nil {
				return nil, err
			}
		}
		if err := compileListValues(row, rowIndex, cellIndex, definitions, aliases, compiled); err != nil {
			return nil, err
		}
		if err := compileCustomParameters(row, rowIndex, definitions, compiled); err != nil {
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

	networkType, defaultVersion := networkProfile(device, paramModel)
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

func buildParameterDefinitions(
	groups []quicksettings.Group,
	mappings []parammodel.ParamMapping,
) (map[string]parameterDefinition, map[string]string) {
	definitions := make(map[string]parameterDefinition)
	aliases := make(map[string]string)
	keysByName := make(map[string][]string)
	collisions := make(map[string]struct{})
	addNameKey := func(name, definitionKey string) {
		for _, existing := range keysByName[name] {
			if existing == definitionKey {
				return
			}
		}
		keysByName[name] = append(keysByName[name], definitionKey)
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
		addNameKey(name, template)
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
			definitions[definitionKey] = parameterDefinition{
				ID: param.Name, Template: template, Type: param.Type, Required: param.Required,
				Readonly: param.Readonly, MinValue: param.MinValue, MaxValue: param.MaxValue,
				EnumOptions: param.EnumOptions,
			}
			addNameKey(param.Name, definitionKey)
			addAlias(param.Name, definitionKey)
			addAlias(param.TitleZh, definitionKey)
			addAlias(param.TitleEn, definitionKey)
		}
	}
	for alias, name := range policyFieldAliases {
		if keys := keysByName[name]; len(keys) == 1 {
			aliases[alias] = keys[0]
		}
	}
	for alias, path := range policyFieldPathAliases {
		if _, exists := definitions[path]; exists {
			aliases[alias] = path
		}
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
	compiled map[string]ResolvedParameter,
) (bool, error) {
	sheets, ok := row["sheetParameters"].(map[string]any)
	if !ok || len(sheets) == 0 {
		return false, nil
	}
	for sheetName, rawRows := range sheets {
		for rowIndex, sheetRow := range mapSlice(rawRows) {
			cellIndex := defaultCell + rowIndex
			handled := make(map[string]struct{})
			ipa, hasIPA := valueByNormalizedKey(sheetRow, "ipa")
			unitID, hasUnitID := valueByNormalizedKey(sheetRow, "unitid")
			if hasIPA && hasUnitID && valueString(ipa) != "" && valueString(unitID) != "" {
				if id, exists := aliases["ipaunitid"]; exists &&
					strings.Contains(definitions[id].Template, "GsmBTSCellDT") {
					if err := compileValue("IPAUnitID", fmt.Sprintf("%s-%s", valueString(ipa), valueString(unitID)),
						"import", fmt.Sprintf("%s[%d].IPAUnitID", sheetName, rowIndex+1),
						cellIndex, rowIndex+1, definitions, aliases, compiled); err != nil {
						return true, err
					}
					handled["ipa"] = struct{}{}
					handled["unitid"] = struct{}{}
				}
			}
			for header, value := range sheetRow {
				normalized := normalizeParameterKey(header)
				if _, alreadyHandled := handled[normalized]; alreadyHandled {
					continue
				}
				if _, ignored := ignoredPolicyFields[normalized]; ignored {
					continue
				}
				if !isScalarValue(value) || valueString(value) == "" {
					continue
				}
				if _, supported := aliases[normalized]; !supported {
					if _, optional := optionalPlanningFields[normalized]; optional {
						continue
					}
				}
				if err := compileValue(header, value, "import",
					fmt.Sprintf("%s[%d].%s", sheetName, rowIndex+1, header),
					cellIndex, rowIndex+1, definitions, aliases, compiled); err != nil {
					return true, err
				}
			}
		}
	}
	return true, nil
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
	definitions map[string]parameterDefinition,
	aliases map[string]string,
	compiled map[string]ResolvedParameter,
) error {
	for index, item := range mapSlice(row["amfList"]) {
		if value := item["amfIp"]; valueString(value) != "" {
			if err := compileValue("AmfIP1", value, "page",
				fmt.Sprintf("paramConfigList[%d].amfList[%d].amfIp", rowIndex, index),
				cellIndex, index+1, definitions, aliases, compiled); err != nil {
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
			path, err := resolveInstancePath(definition.Template, cellIndex, 1)
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
				cellIndex, index+1, definitions, aliases, compiled); err != nil {
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
	path, err := resolveInstancePath(definition.Template, cellIndex, listIndex)
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

func resolveInstancePath(template string, cellIndex, listIndex int) (string, error) {
	path := template
	replacements := []struct {
		pattern string
		value   int
	}{
		{"FAPService.{i}", 1},
		{"CellConfig.{i}", cellIndex},
		{"AMFPoolConfigParam.{i}", listIndex},
		{"ScsSpecificCarrierList.{i}", 1},
		{"MultiFrequencyBandListNRSIB.{i}", 1},
		{"PLMNList.{i}", listIndex},
		{"TA.{i}", 1},
		{"NguIpBind{i}", listIndex},
		{"FAP.Ipsec.{i}", listIndex},
		{"GsmBTSCellDT.{i}", cellIndex},
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
	for _, option := range definition.EnumOptions {
		if value == option.Value || strings.EqualFold(value, option.Label) {
			value = option.Value
			break
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

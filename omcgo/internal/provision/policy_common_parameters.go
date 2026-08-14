package provision

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/omcgo/omcgo/internal/core/model"
)

type policyAllocationRange struct {
	Start int64 `json:"start"`
	End   int64 `json:"end"`
}

type policyAllocationRule struct {
	Start    int64                   `json:"start"`
	End      int64                   `json:"end"`
	Step     int64                   `json:"step"`
	Reserved []policyAllocationRange `json:"reserved"`
}

func policyHasCommonParameters(policy *PlugAndPlayPolicy) bool {
	if policy == nil || len(policy.Config) == 0 {
		return false
	}
	var root map[string]any
	if json.Unmarshal(policy.Config, &root) != nil {
		return false
	}
	return len(mapValue(root["commonParamConfig"])) > 0
}

func validatePolicyCommonParameters(policy *PlugAndPlayPolicy) error {
	if !policyHasCommonParameters(policy) {
		return nil
	}
	var root map[string]any
	if err := json.Unmarshal(policy.Config, &root); err != nil {
		return &ConfigValidationError{Message: fmt.Sprintf("decode policy config: %v", err)}
	}
	common := mapValue(root["commonParamConfig"])
	if !strings.EqualFold(strings.TrimSpace(valueString(common["deviceType"])), "gNB") {
		return nil
	}
	length, ok := integerValue(common["gnbIdLength"])
	if !ok || length < 22 || length > 32 {
		return &ConfigValidationError{Message: "gNB ID length must be between 22 and 32"}
	}
	gnbRule, hasGNBRule, err := decodeAllocationRule(common["gnbIdAllocation"], 0, 4_294_967_295, "gNB ID")
	if err != nil {
		return err
	}
	_, hasPCIRule, err := decodeAllocationRule(common["pciAllocation"], 0, 1007, "PCI")
	if err != nil {
		return err
	}
	if !hasGNBRule || !hasPCIRule {
		return &ConfigValidationError{Message: "gNB common parameters require gNB ID and PCI allocation rules"}
	}
	maximumByLength := int64(1<<uint(length)) - 1
	if gnbRule.End > maximumByLength {
		return &ConfigValidationError{Message: fmt.Sprintf(
			"gNB ID allocation end exceeds the %d-bit maximum %d", length, maximumByLength)}
	}
	return nil
}

// materializePolicyParameters converts a product-level common parameter rule
// into the existing per-device compiler input. Explicit imported rows are
// overlays, not prerequisites: a matching product can execute without its SN
// appearing in paramConfigList.
func materializePolicyParameters(
	policy *PlugAndPlayPolicy,
	device *model.Device,
	fleet []model.Device,
) (*PlugAndPlayPolicy, error) {
	if policy == nil || device == nil {
		return nil, &ConfigValidationError{Message: "policy and device are required"}
	}
	if err := validatePolicyCommonParameters(policy); err != nil {
		return nil, err
	}
	var root map[string]any
	if err := json.Unmarshal(policy.Config, &root); err != nil {
		return nil, &ConfigValidationError{Message: fmt.Sprintf("decode policy config: %v", err)}
	}
	common := sanitizePolicyCommonParameters(cloneMap(mapValue(root["commonParamConfig"])))
	explicitRows := mapSlice(root["paramConfigList"])
	explicitBySerial := make(map[string]map[string]any, len(explicitRows))
	for _, row := range explicitRows {
		serial := strings.TrimSpace(valueString(row["serialNumber"]))
		if serial != "" {
			explicitBySerial[serial] = row
		}
	}
	if len(common) == 0 {
		if _, exists := explicitBySerial[device.SerialNumber]; !exists {
			return nil, &ConfigValidationError{Message: fmt.Sprintf(
				"policy has no common or device parameter configuration for device %s", device.SerialNumber)}
		}
		return policy, nil
	}

	ordered := append([]model.Device(nil), fleet...)
	foundTarget := false
	for _, item := range ordered {
		if item.SerialNumber == device.SerialNumber {
			foundTarget = true
			break
		}
	}
	if !foundTarget {
		ordered = append(ordered, *device)
	}
	sort.SliceStable(ordered, func(i, j int) bool {
		left, right := ordered[i], ordered[j]
		if !left.CreatedAt.Equal(right.CreatedAt) {
			if left.CreatedAt.Equal(time.Time{}) {
				return false
			}
			if right.CreatedAt.Equal(time.Time{}) {
				return true
			}
			return left.CreatedAt.Before(right.CreatedAt)
		}
		return left.SerialNumber < right.SerialNumber
	})

	gnbRule, hasGNBRule, err := decodeAllocationRule(common["gnbIdAllocation"], 0, 4_294_967_295, "gNB ID")
	if err != nil {
		return nil, err
	}
	pciRule, hasPCIRule, err := decodeAllocationRule(common["pciAllocation"], 0, 1007, "PCI")
	if err != nil {
		return nil, err
	}

	usedGNB := make(map[int64]string)
	usedPCI := make(map[int64]string)
	for serial, row := range explicitBySerial {
		if value, ok := configuredInteger(row, "gnbId", "*gNB ID"); ok {
			if previous := usedGNB[value]; previous != "" && previous != serial {
				return nil, &ConfigValidationError{Message: fmt.Sprintf(
					"gNB ID %d is configured for both %s and %s", value, previous, serial)}
			}
			usedGNB[value] = serial
		}
		if value, ok := configuredInteger(row, "pci", "*PCI"); ok {
			if previous := usedPCI[value]; previous != "" && previous != serial {
				return nil, &ConfigValidationError{Message: fmt.Sprintf(
					"PCI %d is configured for both %s and %s", value, previous, serial)}
			}
			usedPCI[value] = serial
		}
	}

	assignments := make(map[string]map[string]int64, len(ordered))
	for _, item := range ordered {
		serial := strings.TrimSpace(item.SerialNumber)
		if serial == "" {
			continue
		}
		assignment := make(map[string]int64, 2)
		override := explicitBySerial[serial]
		if value, ok := configuredInteger(override, "gnbId", "*gNB ID"); ok {
			assignment["gnbId"] = value
		} else if hasGNBRule {
			value, allocateErr := nextPolicyAllocation(gnbRule, usedGNB)
			if allocateErr != nil {
				return nil, &ConfigValidationError{Message: fmt.Sprintf("allocate gNB ID for %s: %v", serial, allocateErr)}
			}
			usedGNB[value] = serial
			assignment["gnbId"] = value
		}
		if value, ok := configuredInteger(override, "pci", "*PCI"); ok {
			assignment["pci"] = value
		} else if hasPCIRule {
			value, allocateErr := nextPolicyAllocation(pciRule, usedPCI)
			if allocateErr != nil {
				return nil, &ConfigValidationError{Message: fmt.Sprintf("allocate PCI for %s: %v", serial, allocateErr)}
			}
			usedPCI[value] = serial
			assignment["pci"] = value
		}
		assignments[serial] = assignment
	}

	selected := cloneMap(common)
	delete(selected, "gnbIdAllocation")
	delete(selected, "pciAllocation")
	if override := explicitBySerial[device.SerialNumber]; override != nil {
		selected = mergeParameterMaps(selected, override)
	}
	selected["serialNumber"] = device.SerialNumber
	selected["id"] = "common-" + device.ID.String()
	selected["updatedBy"] = "common-rule"
	if assignment := assignments[device.SerialNumber]; assignment != nil {
		if value, exists := assignment["gnbId"]; exists {
			selected["gnbId"] = strconv.FormatInt(value, 10)
		}
		if value, exists := assignment["pci"]; exists {
			selected["pci"] = value
		}
	}
	materializeCommonNetworkParameters(selected)
	root["paramConfigList"] = []any{selected}
	encoded, err := json.Marshal(root)
	if err != nil {
		return nil, &ConfigValidationError{Message: fmt.Sprintf("encode materialized policy config: %v", err)}
	}
	copyPolicy := *policy
	copyPolicy.Config = encoded
	return &copyPolicy, nil
}

func materializeCommonNetworkParameters(selected map[string]any) {
	parameterValues := mapValue(selected["networkParameterValues"])
	interfaces := mapSlice(selected["networkInterfaces"])
	if len(interfaces) == 0 && len(parameterValues) == 0 {
		return
	}
	byPath := make(map[string]map[string]any)
	order := make([]string, 0)
	put := func(path string, raw any) {
		value := valueString(raw)
		if path == "" || value == "" {
			return
		}
		if _, exists := byPath[path]; !exists {
			order = append(order, path)
		}
		byPath[path] = map[string]any{"trPath": path, "value": value}
	}
	mappedWorkbookPaths := make(map[string]struct{})
	for _, mapping := range mapSlice(selected["workbookMappings"]) {
		path := strings.TrimSpace(valueString(mapping["trPath"]))
		sheet := strings.TrimSpace(valueString(mapping["sheet"]))
		header := strings.TrimSpace(valueString(mapping["header"]))
		if path != "" && workbookMappingHasValue(selected, sheet, header) {
			mappedWorkbookPaths[path] = struct{}{}
		}
	}
	for path, value := range parameterValues {
		path = strings.TrimSpace(path)
		if _, mapped := mappedWorkbookPaths[path]; mapped {
			continue
		}
		put(path, value)
	}
	putAddressRows := func(prefix, object string, rows []map[string]any, ipv6 bool) {
		for rowIndex, row := range rows {
			rowPrefix := fmt.Sprintf("%s.%s.%d", prefix, object, rowIndex+1)
			if ipv6 {
				put(rowPrefix+".Origin", row["origin"])
				put(rowPrefix+".PrefixLength", row["prefixLength"])
			} else {
				put(rowPrefix+".AddressingType", row["addressingType"])
				put(rowPrefix+".SubnetMask", row["subnetMask"])
			}
			put(rowPrefix+".IPAddress", row["ipAddress"])
			put(rowPrefix+".DefaultGateway", row["defaultGateway"])
			put(rowPrefix+".PortType", row["portType"])
		}
	}
	for interfaceIndex, item := range interfaces {
		prefix := fmt.Sprintf("Device.Ethernet.Interface.%d", interfaceIndex+1)
		put(prefix+".Name", item["name"])
		putAddressRows(prefix, "IPv4Address", mapSlice(item["ipv4Addresses"]), false)
		putAddressRows(prefix, "IPv6Address", mapSlice(item["ipv6Addresses"]), true)
		for vlanIndex, vlan := range mapSlice(item["vlans"]) {
			vlanPrefix := fmt.Sprintf("%s.VlanInterface.%d", prefix, vlanIndex+1)
			put(vlanPrefix+".Name", vlan["name"])
			put(vlanPrefix+".Id", vlan["id"])
			put(vlanPrefix+".Enable", vlan["enable"])
			putAddressRows(vlanPrefix, "IPv4Address", mapSlice(vlan["ipv4Addresses"]), false)
			putAddressRows(vlanPrefix, "IPv6Address", mapSlice(vlan["ipv6Addresses"]), true)
		}
	}
	// Explicit device custom parameters retain override priority over generated
	// product-level network paths.
	for _, item := range mapSlice(selected["customParams"]) {
		path := strings.TrimSpace(valueString(item["trPath"]))
		if path == "" || valueString(item["value"]) == "" {
			continue
		}
		if _, exists := byPath[path]; !exists {
			order = append(order, path)
		}
		byPath[path] = item
	}
	custom := make([]any, 0, len(order))
	for _, path := range order {
		custom = append(custom, byPath[path])
	}
	selected["customParams"] = custom
	delete(selected, "networkInterfaces")
	delete(selected, "networkParameterValues")
	delete(selected, "networkObjectInstances")
}

func sanitizePolicyCommonParameters(common map[string]any) map[string]any {
	sheets := cloneMap(mapValue(common["sheetParameters"]))
	if len(sheets) == 0 {
		return common
	}
	excluded := map[string][]string{
		"DEVICE":    {"Time Zone Term"},
		"INTERFACE": {"Interface Name", "Address Type", "Prefix Length", "Bear Type", "Vlan Name", "OMC IP"},
		"IPSEC":     {"FORCEENCAPS"},
	}
	for sheetName, fields := range excluded {
		rows := mapSlice(sheets[sheetName])
		if len(rows) == 0 {
			continue
		}
		sanitizedRows := make([]any, 0, len(rows))
		for _, row := range rows {
			sanitized := cloneMap(row)
			for _, field := range fields {
				delete(sanitized, field)
			}
			sanitizedRows = append(sanitizedRows, sanitized)
		}
		sheets[sheetName] = sanitizedRows
	}
	common["sheetParameters"] = sheets
	return common
}

func decodeAllocationRule(raw any, minimum, maximum int64, name string) (policyAllocationRule, bool, error) {
	value := mapValue(raw)
	if len(value) == 0 {
		return policyAllocationRule{}, false, nil
	}
	start, startOK := integerValue(value["start"])
	end, endOK := integerValue(value["end"])
	step, stepOK := integerValue(value["step"])
	if !stepOK {
		step = 1
	}
	if !startOK || !endOK || start < minimum || end > maximum || start > end || step < 1 {
		return policyAllocationRule{}, false, &ConfigValidationError{Message: fmt.Sprintf("invalid %s allocation range", name)}
	}
	rule := policyAllocationRule{Start: start, End: end, Step: step}
	for _, item := range mapSlice(value["reserved"]) {
		reservedStart, okStart := integerValue(item["start"])
		reservedEnd, okEnd := integerValue(item["end"])
		if !okStart || !okEnd || reservedStart < minimum || reservedEnd > maximum || reservedStart > reservedEnd {
			return policyAllocationRule{}, false, &ConfigValidationError{Message: fmt.Sprintf("invalid %s reserved range", name)}
		}
		rule.Reserved = append(rule.Reserved, policyAllocationRange{Start: reservedStart, End: reservedEnd})
	}
	return rule, true, nil
}

func nextPolicyAllocation(rule policyAllocationRule, used map[int64]string) (int64, error) {
	for value := rule.Start; value <= rule.End; value += rule.Step {
		if _, exists := used[value]; exists || allocationReserved(value, rule.Reserved) {
			if value > rule.End-rule.Step {
				break
			}
			continue
		}
		return value, nil
	}
	return 0, fmt.Errorf("range is exhausted")
}

func allocationReserved(value int64, ranges []policyAllocationRange) bool {
	for _, item := range ranges {
		if value >= item.Start && value <= item.End {
			return true
		}
	}
	return false
}

func configuredInteger(row map[string]any, field, sheetHeader string) (int64, bool) {
	if value, ok := integerValue(row[field]); ok {
		return value, true
	}
	sheets := mapValue(row["sheetParameters"])
	cellRows := mapSlice(sheets["CELL"])
	if len(cellRows) == 0 {
		return 0, false
	}
	return integerValue(cellRows[0][sheetHeader])
}

func integerValue(value any) (int64, bool) {
	switch typed := value.(type) {
	case float64:
		converted := int64(typed)
		return converted, float64(converted) == typed
	case int:
		return int64(typed), true
	case int64:
		return typed, true
	case json.Number:
		converted, err := typed.Int64()
		return converted, err == nil
	case string:
		converted, err := strconv.ParseInt(strings.TrimSpace(typed), 10, 64)
		return converted, err == nil
	default:
		return 0, false
	}
}

func mapValue(value any) map[string]any {
	result, _ := value.(map[string]any)
	return result
}

func cloneMap(source map[string]any) map[string]any {
	result := make(map[string]any, len(source))
	for key, value := range source {
		if nested, ok := value.(map[string]any); ok {
			result[key] = cloneMap(nested)
		} else {
			result[key] = value
		}
	}
	return result
}

func mergeParameterMaps(base, override map[string]any) map[string]any {
	result := cloneMap(base)
	for key, value := range override {
		baseMap, baseOK := result[key].(map[string]any)
		overrideMap, overrideOK := value.(map[string]any)
		if baseOK && overrideOK {
			result[key] = mergeParameterMaps(baseMap, overrideMap)
			continue
		}
		result[key] = value
	}
	return result
}

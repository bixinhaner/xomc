package carrier

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/omcgo/omcgo/internal/core/model"
)

// GeofenceParameterRole gives every path one unambiguous business meaning.
// In particular, RF and cell runtime state must never be projected as the
// same control signal.
type GeofenceParameterRole string

const (
	GeofenceRoleAdmin   GeofenceParameterRole = "admin"
	GeofenceRoleAdminRF GeofenceParameterRole = "admin_rf"
	GeofenceRoleRF      GeofenceParameterRole = "rf"
	GeofenceRoleIPSec   GeofenceParameterRole = "ipsec"
	GeofenceRoleOpState GeofenceParameterRole = "op_state"
)

// GeofenceDeactivationCapability is the product-model-resolved contract used
// by the geofence state machine. Controls are writable; Terminals are read-only
// postconditions. Missing or ambiguous mandatory roles fail closed.
type GeofenceDeactivationCapability struct {
	Controls  []GeofenceControlParameter
	Terminals []GeofenceControlParameter
}

var (
	lteAdminPathPattern = regexp.MustCompile(
		`^Device\.Services\.FAPService\.([0-9]+)\.FAPControl\.LTE\.AdminState$`,
	)
	lteOpStatePathPattern = regexp.MustCompile(
		`^Device\.Services\.FAPService\.([0-9]+)\.FAPControl\.LTE\.OpState$`,
	)
	lteRFPathPattern = regexp.MustCompile(
		`^Device\.Services\.FAPService\.([0-9]+)\.FAPControl\.LTE\.RFTxStatus$`,
	)
	nrAdminPathPattern = regexp.MustCompile(
		`^Device\.Services\.FAPService\.1\.CellConfig\.([0-9]+)\.NR\.RAN\.(?:CellEnable\.)?AdminState$`,
	)
	nrOpStatePathPattern = regexp.MustCompile(
		`^Device\.Services\.FAPService\.1\.CellConfig\.([0-9]+)\.NR\.RAN\.OpState$`,
	)
	globalIPSecPathPattern = regexp.MustCompile(
		`^Device\.Services\.FAPService\.Ipsec\.IPSEC_ENABLE$`,
	)
	lteInUsePathPattern = regexp.MustCompile(
		`^Device\.Services\.FAPService\.([0-9]+)\.FAPControl\.LTE\.InUse$`,
	)
)

var geofenceCellCountPaths = []string{
	"Device.Services.FAPService.1.CellConfig.LTE.RAN.CA.PARAMS.NumOfCells",
	"Device.Services.FAPService.1.CellConfig.NR.RAN.CA.PARAMS.NumOfCells",
}

// BuildGeofenceDeactivationCapabilityForSnapshotWithMappings resolves the
// complete mandatory contract: writable Admin, RF and IPSec controls plus a
// read-only OpState terminal for every target cell. It deliberately does not
// infer OpState from RF behavior.
func BuildGeofenceDeactivationCapabilityForSnapshotWithMappings(
	productClass string,
	tech model.Technology,
	enabled bool,
	parameters []model.DeviceParameter,
	mappings []GeofenceControlMapping,
) (GeofenceDeactivationCapability, error) {
	if len(mappings) == 0 {
		return GeofenceDeactivationCapability{}, fmt.Errorf(
			"product class %q has no ParamModel mappings; refusing geofence control fallback",
			productClass,
		)
	}
	value := "0"
	if enabled {
		value = "1"
	}

	rfPaths := selectMappedWritableRFControlPaths(parameters, mappings)
	if len(rfPaths) == 0 {
		return GeofenceDeactivationCapability{}, fmt.Errorf(
			"product class %q has no model-proven writable RF control", productClass,
		)
	}

	opStatePaths := selectCapabilityPaths(
		parameters, mappings, tech, GeofenceRoleOpState, false,
	)
	if len(opStatePaths) == 0 {
		return GeofenceDeactivationCapability{}, fmt.Errorf(
			"product class %q has no model-proven read-only cell runtime state", productClass,
		)
	}
	observedInstances := capabilityInstances(tech, GeofenceRoleOpState, opStatePaths)
	maximumInstances := modelMaximumGeofenceCellInstances(mappings)
	if len(maximumInstances) == 0 {
		maximumInstances = observedInstances
	}
	effectiveInstances := resolveEffectiveGeofenceCellInstances(
		parameters, tech, maximumInstances,
	)
	rfPaths = filterCapabilityPathsByInstances(
		tech, GeofenceRoleRF, rfPaths, effectiveInstances,
	)
	opStatePaths = filterCapabilityPathsByInstances(
		tech, GeofenceRoleOpState, opStatePaths, effectiveInstances,
	)
	adminPaths := filterCapabilityPathsByInstances(
		tech,
		GeofenceRoleAdmin,
		selectCapabilityPaths(parameters, mappings, tech, GeofenceRoleAdmin, true),
		effectiveInstances,
	)
	adminControls := make([]GeofenceControlParameter, 0, len(adminPaths))
	for _, path := range adminPaths {
		adminControls = append(adminControls, GeofenceControlParameter{
			Path: path, Role: GeofenceRoleAdmin, AccessProven: true,
			AppliesToAllCells: mappedAdminAppliesToAllCells(path, mappings),
		})
	}
	if len(adminControls) == 0 {
		adminControls = selectPrivateMappedAdminControls(
			parameters, mappings, tech, effectiveInstances,
		)
		adminPaths = controlParameterPaths(adminControls)
	}
	combinedAdminRF := selectAdminRFControlPaths(productClass, rfPaths, mappings)
	if len(adminPaths) == 0 && len(combinedAdminRF) == 0 {
		return GeofenceDeactivationCapability{}, fmt.Errorf(
			"product class %q has no model-proven writable cell administration control", productClass,
		)
	}
	adminCoverage := adminPaths
	adminRole := GeofenceRoleAdmin
	if len(combinedAdminRF) > 0 {
		adminCoverage = combinedAdminRF
		adminRole = GeofenceRoleAdminRF
	}
	if err := validateCellCapabilityCoverage(
		tech, adminRole, adminCoverage, rfPaths, opStatePaths,
		effectiveInstances, hasAllCellAdminControl(adminControls),
	); err != nil {
		return GeofenceDeactivationCapability{}, fmt.Errorf(
			"product class %q cell capability is incomplete: %w", productClass, err,
		)
	}

	ipsecPaths := selectWritableIPSecControlPaths(parameters, mappings)
	if len(ipsecPaths) == 0 {
		return GeofenceDeactivationCapability{}, fmt.Errorf(
			"product class %q has no model-proven writable IPSec control", productClass,
		)
	}

	capability := GeofenceDeactivationCapability{
		Controls:  make([]GeofenceControlParameter, 0, len(adminControls)+len(rfPaths)+len(ipsecPaths)),
		Terminals: make([]GeofenceControlParameter, 0, len(opStatePaths)),
	}
	// Deactivation order is Admin -> RF -> IPSec. Restoration is reordered by
	// the geofence plan to IPSec -> RF -> Admin.
	if len(combinedAdminRF) > 0 {
		for _, path := range combinedAdminRF {
			capability.Controls = append(capability.Controls, GeofenceControlParameter{
				Path: path, Value: value, Role: GeofenceRoleAdminRF, AccessProven: true,
			})
		}
	} else {
		for _, control := range adminControls {
			control.Value = value
			capability.Controls = append(capability.Controls, control)
		}
	}
	for _, path := range rfPaths {
		if containsSortedPath(combinedAdminRF, path) {
			continue
		}
		capability.Controls = append(capability.Controls, GeofenceControlParameter{
			Path: path, Value: value, Role: GeofenceRoleRF, AccessProven: true,
		})
	}
	for _, path := range ipsecPaths {
		capability.Controls = append(capability.Controls, GeofenceControlParameter{
			Path: path, Value: value, Role: GeofenceRoleIPSec, AccessProven: true,
		})
	}
	for _, path := range opStatePaths {
		capability.Terminals = append(capability.Terminals, GeofenceControlParameter{
			Path: path, Value: value, Role: GeofenceRoleOpState, AccessProven: true,
		})
	}
	return capability, nil
}

// resolveEffectiveGeofenceCellInstances applies the device's current cell-use
// facts when they are reliable. If they are absent or contradictory, the
// caller-approved business fallback is the product model's maximum cell set.
func resolveEffectiveGeofenceCellInstances(
	parameters []model.DeviceParameter,
	tech model.Technology,
	maximum map[int]struct{},
) map[int]struct{} {
	if len(maximum) == 0 {
		return maximum
	}
	if tech == model.TechLTE {
		inUse, observed, valid := geofenceLTEInUseInstances(parameters)
		if valid && len(inUse) > 0 && sameInstances(observed, maximum) {
			return inUse
		}
	}
	if count, observed, valid := geofenceConfiguredCellCount(parameters); observed && valid {
		configured := make(map[int]struct{}, count)
		for instance := 1; instance <= count; instance++ {
			configured[instance] = struct{}{}
		}
		if instancesSubset(configured, maximum) {
			return configured
		}
	}
	return maximum
}

// ResolveEffectiveGeofenceCellInstances is also used when restoring a verified
// action. The maximum input is the immutable cell set owned by that action;
// current InUse/NumOfCells may narrow it, but can never expand it.
func ResolveEffectiveGeofenceCellInstances(
	parameters []model.DeviceParameter,
	tech model.Technology,
	maximum []int,
) []int {
	maximumSet := make(map[int]struct{}, len(maximum))
	for _, instance := range maximum {
		if instance > 0 {
			maximumSet[instance] = struct{}{}
		}
	}
	effective := resolveEffectiveGeofenceCellInstances(parameters, tech, maximumSet)
	result := make([]int, 0, len(effective))
	for instance := range effective {
		result = append(result, instance)
	}
	sort.Ints(result)
	return result
}

// GeofenceCellInstance extracts the physical cell index for a canonical
// geofence role. Device-scoped controls such as IPSec intentionally return no
// instance and are not removed by cell filtering.
func GeofenceCellInstance(
	tech model.Technology,
	role GeofenceParameterRole,
	path string,
) (int, bool) {
	instances := capabilityInstances(tech, role, []string{path})
	if len(instances) != 1 {
		return 0, false
	}
	for instance := range instances {
		return instance, true
	}
	return 0, false
}

func geofenceLTEInUseInstances(
	parameters []model.DeviceParameter,
) (map[int]struct{}, map[int]struct{}, bool) {
	result := make(map[int]struct{})
	observed := make(map[int]struct{})
	for _, parameter := range parameters {
		matches := lteInUsePathPattern.FindStringSubmatch(parameter.ParameterPath)
		if len(matches) != 2 {
			continue
		}
		instance, err := strconv.Atoi(matches[1])
		if err != nil || instance < 1 {
			return nil, observed, false
		}
		observed[instance] = struct{}{}
		enabled, valid := geofenceBooleanValue(parameter.ParameterValue)
		if !valid {
			return nil, observed, false
		}
		if enabled {
			result[instance] = struct{}{}
		}
	}
	return result, observed, true
}

func geofenceConfiguredCellCount(
	parameters []model.DeviceParameter,
) (int, bool, bool) {
	count := 0
	observed := false
	for _, parameter := range parameters {
		if !stringInSet(parameter.ParameterPath, geofenceCellCountPaths) {
			continue
		}
		parsed, err := strconv.Atoi(strings.TrimSpace(parameter.ParameterValue))
		if err != nil || parsed < 1 {
			return 0, true, false
		}
		if observed && count != parsed {
			return 0, true, false
		}
		count, observed = parsed, true
	}
	return count, observed, true
}

func modelMaximumGeofenceCellInstances(
	mappings []GeofenceControlMapping,
) map[int]struct{} {
	enumMaximum := 0
	radioModes := ""
	for _, mapping := range mappings {
		if radioModes == "" {
			radioModes = strings.TrimSpace(mapping.ProductRadioModes)
		}
		if !mapping.IsActive || !mapping.IsSupported || !isParameterEntry(mapping.EntryType) ||
			(!stringInSet(mapping.StandardPath, geofenceCellCountPaths) &&
				!stringInSet(mapping.PrivatePath, geofenceCellCountPaths) &&
				!strings.HasSuffix(mapping.StandardPath, ".CellConfig.LTE.RAN.CA.PARAMS.NumOfCells") &&
				!strings.HasSuffix(mapping.StandardPath, ".CellConfig.NR.RAN.CA.PARAMS.NumOfCells")) {
			continue
		}
		for _, raw := range strings.Split(mapping.EnumValues, ",") {
			value, err := strconv.Atoi(strings.TrimSpace(raw))
			if err == nil && value > enumMaximum {
				enumMaximum = value
			}
		}
	}
	maximum := maximumCellCountForRadioModes(radioModes, enumMaximum)
	instances := make(map[int]struct{}, maximum)
	for instance := 1; instance <= maximum; instance++ {
		instances[instance] = struct{}{}
	}
	return instances
}

func maximumCellCountForRadioModes(radioModes string, enumMaximum int) int {
	modeMaximum := 0
	carrierAggregation := false
	for _, raw := range strings.Split(radioModes, ",") {
		switch strings.ToUpper(strings.TrimSpace(raw)) {
		case "SC":
			if modeMaximum < 1 {
				modeMaximum = 1
			}
		case "DC":
			if modeMaximum < 2 {
				modeMaximum = 2
			}
		case "TC":
			if modeMaximum < 3 {
				modeMaximum = 3
			}
		case "CA":
			carrierAggregation = true
		}
	}
	if carrierAggregation && enumMaximum > modeMaximum {
		return enumMaximum
	}
	if modeMaximum > 0 {
		return modeMaximum
	}
	return enumMaximum
}

func geofenceBooleanValue(value string) (bool, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "on", "enabled", "enable", "active":
		return true, true
	case "0", "false", "off", "disabled", "disable", "inactive":
		return false, true
	default:
		return false, false
	}
}

func instancesSubset(subset, set map[int]struct{}) bool {
	for instance := range subset {
		if _, exists := set[instance]; !exists {
			return false
		}
	}
	return true
}

func sameInstances(left, right map[int]struct{}) bool {
	return len(left) == len(right) && instancesSubset(left, right)
}

func filterCapabilityPathsByInstances(
	tech model.Technology,
	role GeofenceParameterRole,
	paths []string,
	instances map[int]struct{},
) []string {
	result := make([]string, 0, len(paths))
	for _, path := range paths {
		pathInstances := capabilityInstances(tech, role, []string{path})
		for instance := range pathInstances {
			if _, exists := instances[instance]; exists {
				result = append(result, path)
			}
		}
	}
	return result
}

func selectPrivateMappedAdminControls(
	parameters []model.DeviceParameter,
	mappings []GeofenceControlMapping,
	tech model.Technology,
	effective map[int]struct{},
) []GeofenceControlParameter {
	result := make([]GeofenceControlParameter, 0)
	seen := make(map[string]struct{})
	for _, parameter := range parameters {
		for _, mapping := range mappings {
			if !mapping.IsActive || !mapping.IsSupported || !isParameterEntry(mapping.EntryType) ||
				!isReadWriteAccess(mapping.Access) ||
				!pathHasCapabilityRole(mapping.StandardPath, tech, GeofenceRoleAdmin) ||
				!mappingPathMatches(mapping.PrivatePath, parameter.ParameterPath) {
				continue
			}
			standardPath, resolved := instantiateMappedPath(
				mapping.StandardPath, mapping.PrivatePath, parameter.ParameterPath,
			)
			if !resolved && !strings.Contains(mapping.PrivatePath, "{i}") && len(effective) == 1 {
				for instance := range effective {
					standardPath = strings.ReplaceAll(mapping.StandardPath, "{i}", strconv.Itoa(instance))
				}
				resolved = !strings.Contains(standardPath, "{i}")
			}
			if !resolved || !pathHasCapabilityRole(standardPath, tech, GeofenceRoleAdmin) {
				continue
			}
			if _, exists := seen[standardPath]; exists {
				continue
			}
			seen[standardPath] = struct{}{}
			result = append(result, GeofenceControlParameter{
				Path: standardPath, SnapshotPath: parameter.ParameterPath,
				Role: GeofenceRoleAdmin, AccessProven: true,
				AppliesToAllCells: !strings.Contains(mapping.PrivatePath, "{i}") &&
					!pathHasNumericCellInstance(mapping.PrivatePath),
			})
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Path < result[j].Path })
	return result
}

func pathHasNumericCellInstance(path string) bool {
	return lteAdminPathPattern.MatchString(strings.TrimSpace(path)) ||
		nrAdminPathPattern.MatchString(strings.TrimSpace(path))
}

func mappedAdminAppliesToAllCells(
	standardPath string,
	mappings []GeofenceControlMapping,
) bool {
	for _, mapping := range mappings {
		if !mapping.IsActive || !mapping.IsSupported || !isParameterEntry(mapping.EntryType) ||
			!isReadWriteAccess(mapping.Access) ||
			!mappingPathMatches(mapping.StandardPath, standardPath) {
			continue
		}
		if !strings.Contains(mapping.PrivatePath, "{i}") &&
			!pathHasNumericCellInstance(mapping.PrivatePath) {
			return true
		}
	}
	return false
}

func hasAllCellAdminControl(controls []GeofenceControlParameter) bool {
	for _, control := range controls {
		if control.AppliesToAllCells {
			return true
		}
	}
	return false
}

func controlParameterPaths(parameters []GeofenceControlParameter) []string {
	result := make([]string, 0, len(parameters))
	for _, parameter := range parameters {
		result = append(result, parameter.Path)
	}
	return result
}

func stringInSet(target string, values []string) bool {
	for _, value := range values {
		if target == value {
			return true
		}
	}
	return false
}

// selectAdminRFControlPaths recognizes a product-model-proven combined cell
// administration/RF control. The 452 field evidence confirms that private
// AdminCellState=0 transitions all cells to inactive and RF off. This rule is
// derived from the ParamModel private path, not a product-class branch.
func selectAdminRFControlPaths(
	productClass string,
	rfPaths []string,
	mappings []GeofenceControlMapping,
) []string {
	if !isVerified452AdminRFProductClass(productClass) {
		return nil
	}
	result := make([]string, 0)
	for _, rfPath := range rfPaths {
		for _, mapping := range mappings {
			if !mapping.IsActive || !mapping.IsSupported || !isReadWriteAccess(mapping.Access) ||
				!strings.HasSuffix(strings.TrimSpace(mapping.PrivatePath), ".AdminCellState") {
				continue
			}
			if mappingPathMatches(mapping.StandardPath, rfPath) {
				result = append(result, rfPath)
				break
			}
		}
	}
	sort.Strings(result)
	return result
}

// The field-proven 452 device (SN 120200055922C8B0068) is routed to the MLN
// product model. Its AdminCellState=0 evidence must not be inherited by BLN,
// BM, or another product merely because they expose the same path suffix.
func isVerified452AdminRFProductClass(productClass string) bool {
	return strings.HasPrefix(strings.ToUpper(strings.TrimSpace(productClass)), "FAP/MLN/")
}

func containsSortedPath(paths []string, target string) bool {
	index := sort.SearchStrings(paths, target)
	return index < len(paths) && paths[index] == target
}

func selectCapabilityPaths(
	parameters []model.DeviceParameter,
	mappings []GeofenceControlMapping,
	tech model.Technology,
	role GeofenceParameterRole,
	writable bool,
) []string {
	paths := make(map[string]struct{})
	for _, parameter := range parameters {
		if !pathHasCapabilityRole(parameter.ParameterPath, tech, role) {
			continue
		}
		if len(mappings) > 0 {
			if !mappingProvesCapability(mappings, parameter.ParameterPath, tech, role, writable) {
				continue
			}
		} else if parameter.Writable != writable {
			continue
		}
		paths[parameter.ParameterPath] = struct{}{}
	}
	result := make([]string, 0, len(paths))
	for path := range paths {
		result = append(result, path)
	}
	sort.Strings(result)
	return result
}

func mappingProvesCapability(
	mappings []GeofenceControlMapping,
	snapshotPath string,
	tech model.Technology,
	role GeofenceParameterRole,
	writable bool,
) bool {
	for _, mapping := range mappings {
		if !mapping.IsActive || !mapping.IsSupported || !isParameterEntry(mapping.EntryType) {
			continue
		}
		if writable && !isReadWriteAccess(mapping.Access) {
			continue
		}
		if !writable && !isReadOnlyAccess(mapping.Access) {
			continue
		}
		if !mappingPathMatches(mapping.StandardPath, snapshotPath) &&
			!mappingPathMatches(mapping.PrivatePath, snapshotPath) {
			continue
		}
		if pathHasCapabilityRole(mapping.StandardPath, tech, role) {
			return true
		}
	}
	return false
}

func pathHasCapabilityRole(
	path string,
	tech model.Technology,
	role GeofenceParameterRole,
) bool {
	path = strings.TrimSpace(path)
	switch tech {
	case model.TechLTE:
		if role == GeofenceRoleAdmin {
			return templateOrConcreteMatch(lteAdminPathPattern, path)
		}
		return role == GeofenceRoleOpState && templateOrConcreteMatch(lteOpStatePathPattern, path)
	case model.TechNR:
		if role == GeofenceRoleAdmin {
			return templateOrConcreteMatch(nrAdminPathPattern, path)
		}
		return role == GeofenceRoleOpState && templateOrConcreteMatch(nrOpStatePathPattern, path)
	default:
		return false
	}
}

func templateOrConcreteMatch(pattern *regexp.Regexp, path string) bool {
	if pattern.MatchString(path) {
		return true
	}
	return pattern.MatchString(strings.ReplaceAll(path, "{i}", "1"))
}

func selectWritableIPSecControlPaths(
	parameters []model.DeviceParameter,
	mappings []GeofenceControlMapping,
) []string {
	global := make([]string, 0, 1)
	single := make([]string, 0)
	multi := make([]string, 0)
	for _, parameter := range parameters {
		path := parameter.ParameterPath
		if len(mappings) > 0 {
			if !mappingProvesWritablePath(mappings, path) {
				continue
			}
		} else if !parameter.Writable {
			continue
		}
		switch {
		case globalIPSecPathPattern.MatchString(path):
			global = append(global, path)
		case singleIPSecPathPattern.MatchString(path):
			matches := singleIPSecPathPattern.FindStringSubmatch(path)
			globalPath := fmt.Sprintf("Device.FAP.Ipsec.%s.TUNNEL_ENABLE", matches[1])
			single = append(single, globalPath)
		case multiIPSecPathPattern.MatchString(path):
			multi = append(multi, path)
		}
	}
	sort.Strings(global)
	sort.Strings(single)
	sort.Strings(multi)
	if len(multi) > 0 {
		return append(global, multi...)
	}
	return append(global, single...)
}

func mappingProvesWritablePath(mappings []GeofenceControlMapping, path string) bool {
	for _, mapping := range mappings {
		if !mapping.IsActive || !mapping.IsSupported || !isParameterEntry(mapping.EntryType) ||
			!isReadWriteAccess(mapping.Access) {
			continue
		}
		if mappingPathMatches(mapping.StandardPath, path) || mappingPathMatches(mapping.PrivatePath, path) {
			return true
		}
	}
	return false
}

func validateCellCapabilityCoverage(
	tech model.Technology,
	adminRole GeofenceParameterRole,
	admins []string,
	rfPaths []string,
	opStates []string,
	expectedInstances map[int]struct{},
	allCellAdmin bool,
) error {
	adminInstances := capabilityInstances(tech, adminRole, admins)
	rfInstances := capabilityInstances(tech, GeofenceRoleRF, rfPaths)
	opStateInstances := capabilityInstances(tech, GeofenceRoleOpState, opStates)
	if err := validateExpectedGeofenceInstances("OpState", opStateInstances, expectedInstances); err != nil {
		return err
	}
	if !allCellAdmin && len(adminInstances) != len(opStateInstances) {
		return fmt.Errorf("Admin/OpState instance count differs (%d/%d)", len(adminInstances), len(opStateInstances))
	}
	if !allCellAdmin {
		for instance := range adminInstances {
			if _, ok := opStateInstances[instance]; !ok {
				return fmt.Errorf("Admin instance %d has no read-only OpState", instance)
			}
		}
	}
	if adminRole != GeofenceRoleAdminRF {
		if len(rfInstances) != len(opStateInstances) {
			return fmt.Errorf("RF/OpState instance count differs (%d/%d)", len(rfInstances), len(opStateInstances))
		}
		for instance := range opStateInstances {
			if _, ok := rfInstances[instance]; !ok {
				return fmt.Errorf("OpState instance %d has no writable RF control", instance)
			}
		}
	}
	return nil
}

func validateExpectedGeofenceInstances(
	role string,
	actual map[int]struct{},
	expected map[int]struct{},
) error {
	if len(actual) != len(expected) {
		return fmt.Errorf("%s/model maximum instance count differs (%d/%d)", role, len(actual), len(expected))
	}
	for instance := range expected {
		if _, exists := actual[instance]; !exists {
			return fmt.Errorf("%s instance %d is missing from the current snapshot", role, instance)
		}
	}
	return nil
}

func capabilityInstances(
	tech model.Technology,
	role GeofenceParameterRole,
	paths []string,
) map[int]struct{} {
	result := make(map[int]struct{}, len(paths))
	var pattern *regexp.Regexp
	switch {
	case tech == model.TechLTE && role == GeofenceRoleAdminRF:
		pattern = lteRFPathPattern
	case tech == model.TechLTE && role == GeofenceRoleRF:
		pattern = lteRFPathPattern
	case tech == model.TechLTE && role == GeofenceRoleAdmin:
		pattern = lteAdminPathPattern
	case tech == model.TechLTE && role == GeofenceRoleOpState:
		pattern = lteOpStatePathPattern
	case tech == model.TechNR && role == GeofenceRoleAdmin:
		pattern = nrAdminPathPattern
	case tech == model.TechNR && role == GeofenceRoleOpState:
		pattern = nrOpStatePathPattern
	default:
		return result
	}
	for _, path := range paths {
		matches := pattern.FindStringSubmatch(path)
		if len(matches) != 2 {
			continue
		}
		if instance, ok := positiveInstance(matches[1]); ok {
			result[instance] = struct{}{}
		}
	}
	return result
}

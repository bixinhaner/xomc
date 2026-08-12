package carrier

import (
	"fmt"
	"regexp"
	"sort"
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
)

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

	adminPaths := selectCapabilityPaths(parameters, mappings, tech, GeofenceRoleAdmin, true)
	combinedAdminRF := selectAdminRFControlPaths(productClass, rfPaths, mappings)
	if len(adminPaths) == 0 && len(combinedAdminRF) == 0 {
		return GeofenceDeactivationCapability{}, fmt.Errorf(
			"product class %q has no model-proven writable cell administration control", productClass,
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
	adminCoverage := adminPaths
	adminRole := GeofenceRoleAdmin
	if len(combinedAdminRF) > 0 {
		adminCoverage = combinedAdminRF
		adminRole = GeofenceRoleAdminRF
	}
	if err := validateCellCapabilityCoverage(
		tech, adminRole, adminCoverage, rfPaths, opStatePaths,
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
		Controls:  make([]GeofenceControlParameter, 0, len(adminPaths)+len(rfPaths)+len(ipsecPaths)),
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
		for _, path := range adminPaths {
			capability.Controls = append(capability.Controls, GeofenceControlParameter{
				Path: path, Value: value, Role: GeofenceRoleAdmin, AccessProven: true,
			})
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
) error {
	adminInstances := capabilityInstances(tech, adminRole, admins)
	rfInstances := capabilityInstances(tech, GeofenceRoleRF, rfPaths)
	opStateInstances := capabilityInstances(tech, GeofenceRoleOpState, opStates)
	if len(adminInstances) != len(opStateInstances) {
		return fmt.Errorf("Admin/OpState instance count differs (%d/%d)", len(adminInstances), len(opStateInstances))
	}
	for instance := range adminInstances {
		if _, ok := opStateInstances[instance]; !ok {
			return fmt.Errorf("Admin instance %d has no read-only OpState", instance)
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

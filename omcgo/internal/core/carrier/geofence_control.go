package carrier

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/omcgo/omcgo/internal/core/model"
)

type GeofenceControlParameter struct {
	Path              string
	SnapshotPath      string
	Value             string
	Role              GeofenceParameterRole
	AccessProven      bool
	AppliesToAllCells bool
}

// GeofenceControlMapping is the small ParamModel projection needed by the
// geofence capability resolver. The private path is retained for diagnostics;
// control tasks still carry StandardPath and ACS performs the final translation.
type GeofenceControlMapping struct {
	StandardPath      string
	PrivatePath       string
	EntryType         string
	Access            string
	EnumValues        string
	ProductRadioModes string
	IsActive          bool
	IsSupported       bool
}

type GeofenceControlParameterInstanceResolver interface {
	GeofenceControlParametersForInstances(
		productClass string,
		tech model.Technology,
		enabled bool,
		instances []int,
	) ([]GeofenceControlParameter, error)
}

var (
	singleIPSecPathPattern = regexp.MustCompile(
		`^Device\.FAP\.Ipsec\.([0-9]+)\.(?:TUNNEL_ENABLE|TUNNEL_CONFIG_TUNNELENABLE)$`,
	)
	multiIPSecPathPattern = regexp.MustCompile(
		`^Device\.Services\.FAPService\.1\.CellConfig\.LTE\.MultiIpsecConfigParam\.([0-9]+)\.Enable$`,
	)
	rfControlPathPatterns = []*regexp.Regexp{
		regexp.MustCompile(`^Device\.Services\.FAPService\.[0-9]+\.FAPControl\.LTE\.RFTxStatus$`),
		regexp.MustCompile(`^Device\.Services\.FAPService\.[0-9]+\.CellConfig\.LTE\.RAN\.RF\.(?:X_COM_RadioEnable|AdminCellState)$`),
		regexp.MustCompile(`^Device\.DeviceInfo\.SAS\.RadioEnable[0-9]*$`),
		regexp.MustCompile(`^Device\.DeviceInfo\.(?:EU\.[0-9]+\.)?RU\.[0-9]+\.RFTxStatus$`),
	}
	standardMappedRFControlPathPattern = regexp.MustCompile(
		`^Device\.Services\.FAPService\.[0-9]+\.FAPControl\.LTE\.RFTxStatus$`,
	)
)

func BuildGeofenceControlParametersForInstances(
	productClass string,
	tech model.Technology,
	enabled bool,
	instances []int,
) ([]GeofenceControlParameter, error) {
	productClass = strings.ToUpper(strings.TrimSpace(productClass))
	if len(instances) == 0 {
		return nil, fmt.Errorf("FAPService instances are required")
	}
	value := "0"
	if enabled {
		value = "1"
	}
	if isMBS31001ProductClass(productClass) {
		if tech != model.TechLTE {
			return nil, fmt.Errorf("no geofence RF control path for product class %q and technology %q", productClass, tech)
		}
		parameters := []GeofenceControlParameter{{
			Path:  "Device.DeviceInfo.SAS.RadioEnable",
			Value: value,
		}}
		for _, instance := range instances {
			if instance <= 0 {
				return nil, fmt.Errorf("FAPService instance must be positive")
			}
			parameters = append(parameters, GeofenceControlParameter{
				Path:  fmt.Sprintf("Device.FAP.Ipsec.%d.TUNNEL_ENABLE", instance),
				Value: value,
			})
		}
		return parameters, nil
	}

	const rfPathTemplate = "Device.Services.FAPService.%d.FAPControl.LTE.RFTxStatus"
	switch {
	case productClass == "BM" && tech == model.TechGSM:
		return nil, fmt.Errorf("BM GSM RfState is read-only and cannot be used for control")
	case tech == model.TechLTE && (productClass == "BLQ" || productClass == "MLQ" ||
		productClass == "BLN" || productClass == "MLN" || productClass == "BM"):
		// Always enqueue the standard model path. The ACS translator resolves it
		// to the product-private X_COM_RadioEnable or AdminCellState path.
	default:
		return nil, fmt.Errorf("no geofence RF control path for product class %q and technology %q", productClass, tech)
	}

	parameters := []GeofenceControlParameter{
		{Path: "Device.Services.FAPService.Ipsec.IPSEC_ENABLE", Value: value},
	}
	for _, instance := range instances {
		if instance <= 0 {
			return nil, fmt.Errorf("FAPService instance must be positive")
		}
		parameters = append(parameters, GeofenceControlParameter{
			Path:  fmt.Sprintf(rfPathTemplate, instance),
			Value: value,
		})
	}
	return parameters, nil
}

// BuildGeofenceControlParametersForSnapshot resolves LTE control capability
// from the device's parameter snapshot. Returned names remain standard paths;
// ACS translates them to product-private TR-069 paths.
func BuildGeofenceControlParametersForSnapshot(
	productClass string,
	tech model.Technology,
	enabled bool,
	parameters []model.DeviceParameter,
) ([]GeofenceControlParameter, error) {
	return BuildGeofenceControlParametersForSnapshotWithMappings(
		productClass, tech, enabled, parameters, nil,
	)
}

// BuildGeofenceControlParametersForSnapshotWithMappings resolves RF paths from
// the product's ParamModel when available. The device snapshot supplies the
// concrete instances and current values; it must not override a ParamModel
// READ_WRITE mapping with a misleading aggregate switch.
func BuildGeofenceControlParametersForSnapshotWithMappings(
	productClass string,
	tech model.Technology,
	enabled bool,
	parameters []model.DeviceParameter,
	mappings []GeofenceControlMapping,
) ([]GeofenceControlParameter, error) {
	productClass = strings.ToUpper(strings.TrimSpace(productClass))
	if tech != model.TechLTE {
		return nil, fmt.Errorf("no geofence RF control path for product class %q and technology %q", productClass, tech)
	}
	value := "0"
	if enabled {
		value = "1"
	}
	singleInstances := make(map[int]struct{})
	multiInstances := make(map[int]struct{})
	for _, parameter := range parameters {
		if !parameter.Writable {
			continue
		}
		if matches := singleIPSecPathPattern.FindStringSubmatch(parameter.ParameterPath); len(matches) == 2 {
			if instance, ok := positiveInstance(matches[1]); ok {
				singleInstances[instance] = struct{}{}
			}
			continue
		}
		if matches := multiIPSecPathPattern.FindStringSubmatch(parameter.ParameterPath); len(matches) == 2 {
			if instance, ok := positiveInstance(matches[1]); ok {
				multiInstances[instance] = struct{}{}
			}
		}
	}
	var rfPaths []string
	if len(mappings) > 0 {
		rfPaths = selectMappedWritableRFControlPaths(parameters, mappings)
		if len(rfPaths) == 0 {
			return nil, fmt.Errorf("no ParamModel-mapped writable geofence RF control parameter for product class %q", productClass)
		}
	} else {
		rfPaths = selectWritableRFControlPaths(parameters)
	}
	if len(rfPaths) == 0 {
		return nil, fmt.Errorf("no writable geofence RF control parameter for product class %q", productClass)
	}
	parametersOut := make([]GeofenceControlParameter, 0, len(rfPaths)+len(singleInstances)+len(multiInstances))
	for _, path := range rfPaths {
		parametersOut = append(parametersOut, GeofenceControlParameter{Path: path, Value: value})
	}
	if len(multiInstances) > 0 {
		for _, instance := range sortedInstances(multiInstances) {
			parametersOut = append(parametersOut, GeofenceControlParameter{
				Path:  fmt.Sprintf("Device.Services.FAPService.1.CellConfig.LTE.MultiIpsecConfigParam.%d.Enable", instance),
				Value: value,
			})
		}
		return parametersOut, nil
	}
	if len(singleInstances) == 0 {
		return nil, fmt.Errorf("no writable geofence IPSec tunnel instances found for product class %q", productClass)
	}
	for _, instance := range sortedInstances(singleInstances) {
		parametersOut = append(parametersOut, GeofenceControlParameter{
			Path:  fmt.Sprintf("Device.FAP.Ipsec.%d.TUNNEL_ENABLE", instance),
			Value: value,
		})
	}
	return parametersOut, nil
}

func selectWritableRFControlPaths(parameters []model.DeviceParameter) []string {
	for _, pattern := range rfControlPathPatterns[:2] {
		paths := make([]string, 0)
		for _, parameter := range parameters {
			if parameter.Writable && pattern.MatchString(parameter.ParameterPath) {
				paths = append(paths, parameter.ParameterPath)
			}
		}
		if len(paths) > 0 {
			sort.Strings(paths)
			return paths
		}
	}
	// RFTxStatus is the stable standard path for products whose ParamModel
	// translates it to a writable private RF path. Its device snapshot may
	// still mark the standard alias read-only, so prefer it before generic
	// aggregate switches such as SAS.RadioEnable.
	paths := make([]string, 0)
	for _, parameter := range parameters {
		if standardMappedRFControlPathPattern.MatchString(parameter.ParameterPath) {
			paths = append(paths, parameter.ParameterPath)
		}
	}
	if len(paths) > 0 {
		sort.Strings(paths)
		return paths
	}
	for _, pattern := range rfControlPathPatterns[2:] {
		paths := make([]string, 0)
		for _, parameter := range parameters {
			if parameter.Writable && pattern.MatchString(parameter.ParameterPath) {
				paths = append(paths, parameter.ParameterPath)
			}
		}
		if len(paths) > 0 {
			sort.Strings(paths)
			return paths
		}
	}
	return nil
}

func selectMappedWritableRFControlPaths(
	parameters []model.DeviceParameter,
	mappings []GeofenceControlMapping,
) []string {
	paths := make([]string, 0)
	seen := make(map[string]struct{})
	bestPriority := int(^uint(0) >> 1)
	for _, parameter := range parameters {
		if !isRFControlPath(parameter.ParameterPath) {
			continue
		}
		for _, mapping := range mappings {
			standardPath, mapped := mappedStandardRFPath(mapping, parameter.ParameterPath)
			if !mapping.IsActive || !mapping.IsSupported ||
				!isReadWriteAccess(mapping.Access) ||
				!isParameterEntry(mapping.EntryType) ||
				!mapped {
				continue
			}
			priority := rfControlPathPriority(standardPath)
			if priority > bestPriority {
				continue
			}
			if priority < bestPriority {
				paths = paths[:0]
				seen = make(map[string]struct{})
				bestPriority = priority
			}
			if _, ok := seen[standardPath]; ok {
				break
			}
			seen[standardPath] = struct{}{}
			paths = append(paths, standardPath)
			break
		}
	}
	sort.Strings(paths)
	return paths
}

func rfControlPathPriority(path string) int {
	path = strings.TrimSpace(path)
	switch {
	case strings.Contains(path, ".FAPControl.LTE.RFTxStatus"):
		return 10
	case strings.Contains(path, ".CellConfig.LTE.RAN.RF."):
		return 10
	case strings.Contains(path, ".RU.") && strings.HasSuffix(path, ".RFTxStatus"):
		return 20
	case strings.Contains(path, ".SAS.RadioEnable"):
		return 30
	default:
		return 100
	}
}

func isRFControlPath(path string) bool {
	for _, pattern := range rfControlPathPatterns {
		if pattern.MatchString(path) {
			return true
		}
	}
	return false
}

// ResolveWritableRFControlPaths returns the concrete RF control paths exposed
// by a device parameter snapshot. It is shared by geofence and device-access
// RF actions so product-private and multi-instance paths are resolved once.
func ResolveWritableRFControlPaths(parameters []model.DeviceParameter) []string {
	return selectWritableRFControlPaths(parameters)
}

// ResolveWritableRFControlPathsForProduct keeps device-access RF operations on
// the same product-specific control surface as the established geofence flow.
// mBS31001 exposes per-cell X_COM_RadioEnable values for status projection, but
// the durable device control switch is DeviceInfo.SAS.RadioEnable. Writing the
// per-cell status aliases can return SPV/GPV success and then immediately fall
// back, which must not be treated as a successful RF action.
//
// Do not gate this product-specific path on device_parameters.writable. That
// flag comes from an early CPE GetParameterNames snapshot and may disagree with
// the ParamModel READ_WRITE mapping used by the parameter-tree write path (see
// T-0148). Requiring the exact reported path keeps the resolver fail-closed
// without reintroducing the stale snapshot as a second write-authority source.
func ResolveWritableRFControlPathsForProduct(
	productClass string,
	parameters []model.DeviceParameter,
) []string {
	if IsMBS31001ProductClass(productClass) {
		const radioPath = "Device.DeviceInfo.SAS.RadioEnable"
		for _, parameter := range parameters {
			if strings.TrimSpace(parameter.ParameterPath) == radioPath {
				return []string{radioPath}
			}
		}
		return nil
	}
	return selectWritableRFControlPaths(parameters)
}

// IsRFControlPath reports whether path is one of the supported RF control
// parameter families. Callers use it to validate queued security actions.
func IsRFControlPath(path string) bool {
	return isRFControlPath(strings.TrimSpace(path))
}

func isParameterEntry(entryType string) bool {
	return strings.EqualFold(strings.TrimSpace(entryType), "parameter")
}

func isReadWriteAccess(access string) bool {
	access = strings.ToLower(strings.TrimSpace(access))
	access = strings.NewReplacer("_", "", "-", "", " ", "").Replace(access)
	return access == "readwrite" || access == "rw"
}

func isReadOnlyAccess(access string) bool {
	access = strings.ToLower(strings.TrimSpace(access))
	access = strings.NewReplacer("_", "", "-", "", " ", "").Replace(access)
	return access == "readonly" || access == "ro"
}

func mappingPathMatches(template, concrete string) bool {
	template = strings.TrimSpace(template)
	concrete = strings.TrimSpace(concrete)
	if template == "" || concrete == "" {
		return false
	}
	if !strings.Contains(template, "{i}") {
		return template == concrete
	}
	pattern := regexp.QuoteMeta(template)
	pattern = strings.ReplaceAll(pattern, `\{i\}`, `[0-9]+`)
	matched, err := regexp.MatchString("^"+pattern+"$", concrete)
	return err == nil && matched
}

func mappedStandardRFPath(mapping GeofenceControlMapping, snapshotPath string) (string, bool) {
	if mappingPathMatches(mapping.StandardPath, snapshotPath) {
		return snapshotPath, true
	}
	if !mappingPathMatches(mapping.PrivatePath, snapshotPath) {
		return "", false
	}
	return instantiateMappedPath(mapping.StandardPath, mapping.PrivatePath, snapshotPath)
}

func instantiateMappedPath(standardTemplate, privateTemplate, concretePrivate string) (string, bool) {
	if !strings.Contains(standardTemplate, "{i}") {
		return standardTemplate, true
	}
	privatePattern := regexp.QuoteMeta(privateTemplate)
	privatePattern = strings.ReplaceAll(privatePattern, `\{i\}`, `([0-9]+)`)
	pattern, err := regexp.Compile("^" + privatePattern + "$")
	if err != nil {
		return "", false
	}
	matches := pattern.FindStringSubmatch(concretePrivate)
	if len(matches) == 0 {
		return "", false
	}
	standardPath := standardTemplate
	for _, instance := range matches[1:] {
		standardPath = strings.Replace(standardPath, "{i}", instance, 1)
	}
	return standardPath, !strings.Contains(standardPath, "{i}")
}

func DetectGeofenceControlInstances(
	parameters []model.DeviceParameter,
	productClass string,
	tech model.Technology,
) ([]int, error) {
	productClass = strings.ToUpper(strings.TrimSpace(productClass))
	if IsMBS31001ProductClass(productClass) {
		if tech != model.TechLTE {
			return nil, fmt.Errorf("no geofence RF control parameter for product class %q and technology %q", productClass, tech)
		}
		return detectMBS31001IPSecInstances(parameters, productClass)
	}
	var supported bool
	switch {
	case tech == model.TechLTE && (productClass == "BLQ" || productClass == "MLQ" ||
		productClass == "BLN" || productClass == "MLN" || productClass == "BM"):
		supported = true
	default:
		return nil, fmt.Errorf("no geofence RF control parameter for product class %q and technology %q", productClass, tech)
	}
	if !supported {
		return nil, fmt.Errorf("no geofence RF control parameter for product class %q and technology %q", productClass, tech)
	}
	pattern := regexp.MustCompile(
		`^Device\.Services\.FAPService\.([0-9]+)\.(?:` +
			`FAPControl\.LTE\.RFTxStatus|` +
			`CellConfig\.LTE\.RAN\.RF\.(?:X_COM_RadioEnable|AdminCellState)` +
			`)$`,
	)
	instances := make([]int, 0)
	seen := make(map[int]struct{})
	for _, parameter := range parameters {
		if !parameter.Writable {
			continue
		}
		matches := pattern.FindStringSubmatch(parameter.ParameterPath)
		if len(matches) != 2 {
			continue
		}
		var instance int
		if _, err := fmt.Sscanf(matches[1], "%d", &instance); err != nil || instance <= 0 {
			continue
		}
		if _, ok := seen[instance]; ok {
			continue
		}
		seen[instance] = struct{}{}
		instances = append(instances, instance)
	}
	sort.Ints(instances)
	if len(instances) == 0 {
		return nil, fmt.Errorf("no writable geofence RF instances found for product class %q", productClass)
	}
	return instances, nil
}

func isMBS31001ProductClass(productClass string) bool {
	return IsMBS31001ProductClass(productClass)
}

// IsMBS31001ProductClass identifies the 4G mBS31001 product family without
// coupling callers to a specific /SC, /DC, or /CA suffix.
func IsMBS31001ProductClass(productClass string) bool {
	return strings.HasPrefix(strings.ToUpper(strings.TrimSpace(productClass)), "FAP/MBS31001/")
}

func detectMBS31001IPSecInstances(
	parameters []model.DeviceParameter,
	productClass string,
) ([]int, error) {
	const radioPath = "Device.DeviceInfo.SAS.RadioEnable"
	instances := make([]int, 0)
	seen := make(map[int]struct{})
	hasWritableRadio := false
	for _, parameter := range parameters {
		if !parameter.Writable {
			continue
		}
		if parameter.ParameterPath == radioPath {
			hasWritableRadio = true
			continue
		}
		matches := singleIPSecPathPattern.FindStringSubmatch(parameter.ParameterPath)
		if len(matches) != 2 {
			matches = multiIPSecPathPattern.FindStringSubmatch(parameter.ParameterPath)
		}
		if len(matches) != 2 {
			continue
		}
		instance, ok := positiveInstance(matches[1])
		if !ok {
			continue
		}
		if _, ok := seen[instance]; ok {
			continue
		}
		seen[instance] = struct{}{}
		instances = append(instances, instance)
	}
	if !hasWritableRadio {
		return nil, fmt.Errorf("no writable geofence RF control parameter for product class %q", productClass)
	}
	if len(instances) == 0 {
		return nil, fmt.Errorf("no writable geofence IPSec tunnel instances found for product class %q", productClass)
	}
	sort.Ints(instances)
	return instances, nil
}

func positiveInstance(raw string) (int, bool) {
	instance, err := strconv.Atoi(raw)
	return instance, err == nil && instance > 0
}

func sortedInstances(instances map[int]struct{}) []int {
	result := make([]int, 0, len(instances))
	for instance := range instances {
		result = append(result, instance)
	}
	sort.Ints(result)
	return result
}

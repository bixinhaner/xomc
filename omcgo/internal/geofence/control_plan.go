package geofence

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/omcgo/omcgo/internal/core/carrier"
	"github.com/omcgo/omcgo/internal/core/model"
)

var geofenceRFControlPathPattern = regexp.MustCompile(
	`^Device\.Services\.FAPService\.([0-9]+)\.(?:` +
		`FAPControl\.LTE\.RFTxStatus|` +
		`CellConfig\.LTE\.RAN\.RF\.(?:X_COM_RadioEnable|AdminCellState)` +
		`)$`,
)

var geofenceSingleIPSecControlPathPattern = regexp.MustCompile(
	`^Device\.FAP\.Ipsec\.([0-9]+)\.TUNNEL_ENABLE$`,
)

var geofenceMappedStandardRFControlPathPattern = regexp.MustCompile(
	`^(?:` +
		`Device\.Services\.FAPService\.[0-9]+\.FAPControl\.(?:LTE|NR)\.RFTxStatus|` +
		`Device\.Services\.FAPService\.[0-9]+\.CellConfig\.(?:LTE|NR)\.RAN\.RF\.(?:AdminCellState|X_COM_RadioEnable|ForceRadioEnable)|` +
		`Device\.DeviceInfo\.(?:EU\.[0-9]+\.)?RU\.[0-9]+\.RFTxStatus|` +
		`Device\.DeviceInfo\.SAS\.RadioEnable[0-9]*` +
		`)$`,
)

type geofenceControlPlan struct {
	Before    []ControlParameterState
	Requested []ControlParameterState
	Terminals []ControlParameterState
}

func buildGeofenceControlPlan(
	snapshot []model.DeviceParameter,
	targets []carrier.GeofenceControlParameter,
) (geofenceControlPlan, error) {
	return buildGeofenceControlPlanWithTerminal(snapshot, targets, nil, false)
}

func buildGeofenceControlPlanWithRestore(
	snapshot []model.DeviceParameter,
	targets []carrier.GeofenceControlParameter,
	forceRequested bool,
) (geofenceControlPlan, error) {
	return buildGeofenceControlPlanWithTerminal(snapshot, targets, nil, forceRequested)
}

func buildGeofenceControlPlanWithTerminal(
	snapshot []model.DeviceParameter,
	targets []carrier.GeofenceControlParameter,
	terminals []carrier.GeofenceControlParameter,
	forceRequested bool,
) (geofenceControlPlan, error) {
	plan := geofenceControlPlan{
		Before:    make([]ControlParameterState, 0, len(targets)+len(terminals)),
		Requested: make([]ControlParameterState, 0, len(targets)),
		Terminals: make([]ControlParameterState, 0, len(terminals)),
	}
	for _, target := range targets {
		if target.Path == "" || strings.Contains(target.Path, "{i}") {
			return geofenceControlPlan{}, fmt.Errorf(
				"geofence control parameter path is unresolved: %q", target.Path,
			)
		}
		targetValue, err := normalizeControlBoolean(target.Value)
		if err != nil {
			return geofenceControlPlan{}, fmt.Errorf(
				"normalize geofence target %s: %w", target.Path, err,
			)
		}
		snapshotPath := target.SnapshotPath
		if snapshotPath == "" {
			snapshotPath = target.Path
		}
		current, ok := findControlParameterSnapshot(snapshot, snapshotPath)
		if !ok {
			return geofenceControlPlan{}, fmt.Errorf(
				"geofence control parameter %s is missing from the device snapshot", target.Path,
			)
		}
		if !current.Writable && !target.AccessProven &&
			!geofenceMappedStandardRFControlPathPattern.MatchString(target.Path) {
			return geofenceControlPlan{}, fmt.Errorf(
				"geofence control parameter %s is not writable", target.Path,
			)
		}
		currentValue, err := normalizeControlBoolean(current.ParameterValue)
		if err != nil {
			return geofenceControlPlan{}, fmt.Errorf(
				"normalize current geofence parameter %s: %w", target.Path, err,
			)
		}
		plan.Before = append(plan.Before, ControlParameterState{
			Path: target.Path, ObservedPath: target.SnapshotPath,
			Value: currentValue, Role: target.Role,
			AppliesToAllCells: target.AppliesToAllCells,
		})
		if forceRequested || currentValue != targetValue {
			plan.Requested = append(plan.Requested, ControlParameterState{
				Path: target.Path, Value: targetValue, Role: target.Role,
				AppliesToAllCells: target.AppliesToAllCells,
			})
		}
	}
	for _, terminal := range terminals {
		if terminal.Path == "" || strings.Contains(terminal.Path, "{i}") {
			return geofenceControlPlan{}, fmt.Errorf(
				"geofence terminal parameter path is unresolved: %q", terminal.Path,
			)
		}
		targetValue, err := normalizeControlBoolean(terminal.Value)
		if err != nil {
			return geofenceControlPlan{}, fmt.Errorf(
				"normalize geofence terminal target %s: %w", terminal.Path, err,
			)
		}
		current, ok := findControlParameterSnapshot(snapshot, terminal.Path)
		if !ok {
			return geofenceControlPlan{}, fmt.Errorf(
				"geofence terminal parameter %s is missing from the device snapshot", terminal.Path,
			)
		}
		if current.Writable && !terminal.AccessProven {
			return geofenceControlPlan{}, fmt.Errorf(
				"geofence terminal parameter %s is writable; refusing to use it as OpState", terminal.Path,
			)
		}
		currentValue, err := normalizeControlBoolean(current.ParameterValue)
		if err != nil {
			return geofenceControlPlan{}, fmt.Errorf(
				"normalize current geofence terminal %s: %w", terminal.Path, err,
			)
		}
		plan.Before = append(plan.Before, ControlParameterState{
			Path: terminal.Path, Value: currentValue, Role: carrier.GeofenceRoleOpState,
		})
		plan.Terminals = append(plan.Terminals, ControlParameterState{
			Path: terminal.Path, Value: targetValue, Role: carrier.GeofenceRoleOpState,
		})
	}
	return plan, nil
}

func findControlParameterSnapshot(
	snapshot []model.DeviceParameter,
	standardPath string,
) (model.DeviceParameter, bool) {
	for _, parameter := range snapshot {
		if parameter.ParameterPath == standardPath {
			return parameter, true
		}
	}
	want := geofenceRFControlPathPattern.FindStringSubmatch(standardPath)
	if singleIPSec := geofenceSingleIPSecControlPathPattern.FindStringSubmatch(standardPath); len(singleIPSec) == 2 {
		privatePath := "Device.FAP.Ipsec." + singleIPSec[1] + ".TUNNEL_CONFIG_TUNNELENABLE"
		for _, parameter := range snapshot {
			if parameter.ParameterPath == privatePath {
				return parameter, true
			}
		}
	}
	if len(want) != 2 {
		return model.DeviceParameter{}, false
	}
	for _, parameter := range snapshot {
		got := geofenceRFControlPathPattern.FindStringSubmatch(parameter.ParameterPath)
		if len(got) == 2 && got[1] == want[1] {
			return parameter, true
		}
	}
	return model.DeviceParameter{}, false
}

func normalizeControlBoolean(value string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "on", "enabled", "enable", "active":
		return "1", nil
	case "0", "false", "off", "disabled", "disable", "inactive":
		return "0", nil
	default:
		return "", fmt.Errorf("unsupported boolean value %q", value)
	}
}

func marshalGeofenceControlPayload(states []ControlParameterState) ([]byte, error) {
	values := make([]map[string]string, 0, len(states))
	for _, state := range states {
		values = append(values, map[string]string{
			"name": state.Path, "value": state.Value, "type": "xsd:boolean",
		})
	}
	return json.Marshal(map[string]any{"values": values})
}

func restoreTargets(
	before []ControlParameterState,
	changed []ControlParameterState,
) []carrier.GeofenceControlParameter {
	changedPaths := make(map[string]struct{}, len(changed))
	for _, state := range changed {
		changedPaths[state.Path] = struct{}{}
	}
	targets := make([]carrier.GeofenceControlParameter, 0, len(changedPaths))
	for _, state := range before {
		if state.Role == carrier.GeofenceRoleOpState {
			continue
		}
		if _, ok := changedPaths[state.Path]; !ok {
			continue
		}
		targets = append(targets, carrier.GeofenceControlParameter{
			Path: state.Path, SnapshotPath: state.ObservedPath,
			Value: state.Value, Role: state.Role, AccessProven: true,
			AppliesToAllCells: state.AppliesToAllCells,
		})
	}
	sort.SliceStable(targets, func(i, j int) bool {
		return geofenceRestoreRolePriority(targets[i]) < geofenceRestoreRolePriority(targets[j])
	})
	return targets
}

func filterRestoreToEffectiveCells(
	snapshot []model.DeviceParameter,
	tech model.Technology,
	targets []carrier.GeofenceControlParameter,
	terminals []ControlParameterState,
) ([]carrier.GeofenceControlParameter, []ControlParameterState) {
	maximumSet := make(map[int]struct{})
	for _, terminal := range terminals {
		if instance, ok := carrier.GeofenceCellInstance(
			tech, carrier.GeofenceRoleOpState, terminal.Path,
		); ok {
			maximumSet[instance] = struct{}{}
		}
	}
	if len(maximumSet) == 0 {
		return targets, terminals
	}
	maximum := make([]int, 0, len(maximumSet))
	for instance := range maximumSet {
		maximum = append(maximum, instance)
	}
	effectiveList := carrier.ResolveEffectiveGeofenceCellInstances(snapshot, tech, maximum)
	effective := make(map[int]struct{}, len(effectiveList))
	for _, instance := range effectiveList {
		effective[instance] = struct{}{}
	}

	filteredTargets := make([]carrier.GeofenceControlParameter, 0, len(targets))
	for _, target := range targets {
		if target.AppliesToAllCells {
			filteredTargets = append(filteredTargets, target)
			continue
		}
		instance, cellScoped := carrier.GeofenceCellInstance(tech, target.Role, target.Path)
		if !cellScoped {
			filteredTargets = append(filteredTargets, target)
			continue
		}
		if _, exists := effective[instance]; exists {
			filteredTargets = append(filteredTargets, target)
		}
	}
	filteredTerminals := make([]ControlParameterState, 0, len(terminals))
	for _, terminal := range terminals {
		instance, cellScoped := carrier.GeofenceCellInstance(
			tech, carrier.GeofenceRoleOpState, terminal.Path,
		)
		if !cellScoped {
			continue
		}
		if _, exists := effective[instance]; exists {
			filteredTerminals = append(filteredTerminals, terminal)
		}
	}
	return filteredTargets, filteredTerminals
}

func geofenceRestoreRolePriority(parameter carrier.GeofenceControlParameter) int {
	switch parameter.Role {
	case carrier.GeofenceRoleIPSec:
		return 10
	case carrier.GeofenceRoleRF:
		return 20
	case carrier.GeofenceRoleAdmin:
		return 30
	case carrier.GeofenceRoleAdminRF:
		return 30
	}
	if isGeofenceIPSecPath(parameter.Path) {
		return 10
	}
	return 20
}

func originalTerminalTargets(before []ControlParameterState) []ControlParameterState {
	result := make([]ControlParameterState, 0)
	for _, state := range before {
		if state.Role != carrier.GeofenceRoleOpState {
			continue
		}
		result = append(result, state)
	}
	return result
}

func isGeofenceIPSecPath(path string) bool {
	normalized := strings.ToLower(path)
	return strings.Contains(normalized, ".ipsec.") ||
		strings.Contains(normalized, "multiipsecconfigparam")
}

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

type geofenceControlPlan struct {
	Before    []ControlParameterState
	Requested []ControlParameterState
}

func buildGeofenceControlPlan(
	snapshot []model.DeviceParameter,
	targets []carrier.GeofenceControlParameter,
) (geofenceControlPlan, error) {
	plan := geofenceControlPlan{
		Before:    make([]ControlParameterState, 0, len(targets)),
		Requested: make([]ControlParameterState, 0, len(targets)),
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
		current, ok := findControlParameterSnapshot(snapshot, target.Path)
		if !ok {
			return geofenceControlPlan{}, fmt.Errorf(
				"geofence control parameter %s is missing from the device snapshot", target.Path,
			)
		}
		if !current.Writable {
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
			Path: target.Path, Value: currentValue,
		})
		if currentValue != targetValue {
			plan.Requested = append(plan.Requested, ControlParameterState{
				Path: target.Path, Value: targetValue,
			})
		}
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
	case "1", "true", "on", "enabled", "enable":
		return "1", nil
	case "0", "false", "off", "disabled", "disable":
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
		if _, ok := changedPaths[state.Path]; !ok {
			continue
		}
		targets = append(targets, carrier.GeofenceControlParameter{
			Path: state.Path, Value: state.Value,
		})
	}
	sort.SliceStable(targets, func(i, j int) bool {
		// Recovery establishes IPSec before RF when both were changed.
		iIPSec := strings.Contains(targets[i].Path, "IPSEC_ENABLE")
		jIPSec := strings.Contains(targets[j].Path, "IPSEC_ENABLE")
		return iIPSec && !jIPSec
	})
	return targets
}

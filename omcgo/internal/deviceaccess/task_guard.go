package deviceaccess

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/omcgo/omcgo/internal/task"
)

// CarrierResolver resolves the carrier scope for an existing task target.
// It must reject ambiguous ownership instead of falling back to a default carrier.
type CarrierResolver interface {
	ResolveCarrier(ctx context.Context, serialNumber string) (string, error)
}

type AccessTaskGuard struct {
	repository Repository
	carriers   CarrierResolver
	actions    SecurityActionAuthorizer
	snapshot   AccessSnapshotStateReader
	settings   RuntimeSettingsReader
}

// AccessSnapshotStateReader supplies only the hot-path authorization summary.
// The ACS process owns the concrete Redis representation.
type AccessSnapshotStateReader func(ctx context.Context, serialNumber string) (state string, frozen bool, found bool, err error)

func (r AccessSnapshotStateReader) Read(ctx context.Context, serialNumber string) (string, bool, bool, error) {
	return r(ctx, serialNumber)
}

type SecurityActionAuthorizer interface {
	AuthorizeSecurityAction(ctx context.Context, request task.TaskAdmissionRequest) (bool, string, error)
}

func NewAccessTaskGuard(repository Repository, carriers CarrierResolver) *AccessTaskGuard {
	return &AccessTaskGuard{repository: repository, carriers: carriers}
}

func (g *AccessTaskGuard) SetSecurityActionAuthorizer(authorizer SecurityActionAuthorizer) {
	g.actions = authorizer
}

func (g *AccessTaskGuard) SetAccessSnapshotReader(reader AccessSnapshotStateReader) {
	g.snapshot = reader
}

func (g *AccessTaskGuard) SetRuntimeSettingsReader(reader RuntimeSettingsReader) {
	g.settings = reader
}

func (g *AccessTaskGuard) Allow(ctx context.Context, request task.TaskAdmissionRequest) (bool, string, error) {
	serialNumber := request.DeviceSN
	serialNumber = strings.TrimSpace(serialNumber)
	if serialNumber == "" {
		return false, "serial_number_required", ErrSerialNumberRequired
	}
	if g == nil || g.repository == nil || g.carriers == nil {
		return false, "task_guard_dependency_missing", ErrAccessGateDependencyMissing
	}
	carrier, err := g.carriers.ResolveCarrier(ctx, serialNumber)
	if err != nil {
		return false, "carrier_resolution_failed", fmt.Errorf("resolve task carrier: %w", err)
	}
	enabled, err := runtimeAccessEnabled(ctx, g.settings, carrier)
	if err != nil {
		return false, "access_control_settings_unavailable", fmt.Errorf("load device access business switch: %w", err)
	}
	if !enabled {
		switch request.AdmissionClass {
		case "", task.AdmissionClassNormal:
			return true, "access_control_disabled", nil
		case task.AdmissionClassSecurityAction:
			// Turning the business switch off stops every new RF mutation, but
			// an already-dispatched action must still be allowed to finish its
			// read-only GPV verification so the device result is not lost.
			if request.Method == "GetParameterValues" && g.actions != nil {
				return g.actions.AuthorizeSecurityAction(ctx, request)
			}
			return false, "access_control_disabled", nil
		default:
			return false, "access_control_disabled", nil
		}
	}

	switch request.AdmissionClass {
	case task.AdmissionClassAccessProbe:
		if err := validateAccessProbeTask(request, carrier); err != nil {
			return false, "invalid_access_probe_contract", err
		}
		return true, "access_probe_allowed", nil
	case task.AdmissionClassSecurityAction:
		if g.actions == nil {
			return false, "security_action_authorizer_unavailable", nil
		}
		return g.actions.AuthorizeSecurityAction(ctx, request)
	case "", task.AdmissionClassNormal:
		if g.snapshot != nil {
			state, frozen, found, snapshotErr := g.snapshot.Read(ctx, serialNumber)
			if snapshotErr == nil && found {
				if state != string(AccessStateAccepted) || frozen {
					return false, "normal_tasks_frozen_by_access_snapshot", nil
				}
				// An accepted cache entry may lag a newer rejected/revoked durable
				// decision. Positive authorization therefore always verifies the
				// PostgreSQL projection; the cache remains a safe fast-deny path.
			}
			// Redis is an acceleration layer, not the authorization source of
			// truth. A missing/expired snapshot or a transient Redis failure must
			// fall back to the durable projection; otherwise an unchanged accepted
			// decision becomes permanently frozen as soon as its cache TTL expires.
		}
		accessContext, err := g.repository.LoadEvaluationContext(ctx, carrier, serialNumber)
		if err != nil {
			return false, "access_state_unavailable", fmt.Errorf("load task access state: %w", err)
		}
		if accessContext.State != nil && accessContext.State.State == AccessStateAccepted && !accessContext.State.NormalTasksFrozen {
			if g.snapshot != nil {
				return true, "accepted_durable_verified", nil
			}
			return true, "accepted", nil
		}
		return false, "normal_tasks_frozen_by_access_state", nil
	default:
		return false, "unknown_admission_class", nil
	}
}

// 16 serving cells × (TAC + PLMNID + IsPrimary + ECI) plus seven GPS fields.
const maxAccessProbeParameterNames = 80

func validateAccessProbeTask(request task.TaskAdmissionRequest, resolvedCarrier string) error {
	if request.Source != task.TaskSourceDeviceAccess {
		return fmt.Errorf("access probe source must be %q", task.TaskSourceDeviceAccess)
	}
	var params gpsProbeTaskParams
	if err := json.Unmarshal(request.Params, &params); err != nil {
		return fmt.Errorf("decode access probe params: %w", err)
	}
	if strings.TrimSpace(params.SerialNumber) != strings.TrimSpace(request.DeviceSN) {
		return fmt.Errorf("access probe serial number does not match task target")
	}
	if err := validateIdentity(params.Carrier, params.SerialNumber); err != nil {
		return fmt.Errorf("validate access probe identity: %w", err)
	}
	if strings.TrimSpace(params.Carrier) != strings.TrimSpace(resolvedCarrier) {
		return fmt.Errorf("access probe carrier does not match task target")
	}
	if params.EvidenceVersion <= 0 {
		return fmt.Errorf("access probe evidence version must be positive")
	}
	if request.Method == "GetParameterNames" {
		return validateServingCellDiscoveryTask(params)
	}
	if request.Method != "GetParameterValues" {
		return fmt.Errorf("access probe method %q is not allowed", request.Method)
	}
	if len(params.Names) == 0 || len(params.Names) > maxAccessProbeParameterNames {
		return fmt.Errorf("access probe parameter count must be between 1 and %d", maxAccessProbeParameterNames)
	}

	allowed := make(map[string]struct{}, len(params.Paths)+len(params.RadioPaths))
	standardGPSPaths := make(map[string]struct{}, len(gpsPathSpecs))
	for _, spec := range gpsPathSpecs {
		standardGPSPaths[spec.EvidencePath] = struct{}{}
	}
	for _, path := range params.Paths {
		if _, ok := standardGPSPaths[path.StandardPath]; !ok || strings.TrimSpace(path.PrivatePath) == "" {
			return fmt.Errorf("GPS access-evidence path %q is not allowed", path.StandardPath)
		}
		if _, duplicate := allowed[path.PrivatePath]; duplicate {
			return fmt.Errorf("duplicate access-probe path %q", path.PrivatePath)
		}
		allowed[path.PrivatePath] = struct{}{}
	}
	for _, path := range params.RadioPaths {
		if !validRadioEvidencePath(path) {
			return fmt.Errorf("radio access-evidence path %q is not allowed", path.StandardPath)
		}
		if _, duplicate := allowed[path.PrivatePath]; duplicate {
			return fmt.Errorf("duplicate access-probe path %q", path.PrivatePath)
		}
		allowed[path.PrivatePath] = struct{}{}
	}
	if len(allowed) != len(params.Names) {
		return fmt.Errorf("access probe names must exactly match declared evidence paths")
	}
	seen := make(map[string]struct{}, len(params.Names))
	for _, name := range params.Names {
		name = strings.TrimSpace(name)
		if _, ok := allowed[name]; !ok {
			return fmt.Errorf("access probe parameter %q is not declared evidence", name)
		}
		if _, duplicate := seen[name]; duplicate {
			return fmt.Errorf("duplicate access-probe parameter %q", name)
		}
		seen[name] = struct{}{}
	}
	if params.NeedGPS != (len(params.Paths) > 0) {
		return fmt.Errorf("access probe GPS intent does not match evidence paths")
	}
	if params.NeedTAC != hasRadioPathKind(params.RadioPaths, radioPathTAC) {
		return fmt.Errorf("access probe TAC intent does not match evidence paths")
	}
	if err := validateECGIProbeContract(params.RadioPaths, params.NeedECGI); err != nil {
		return err
	}
	return nil
}

func validateECGIProbeContract(paths []RadioParameterPath, needed bool) error {
	type contract struct {
		plmn, primary, servingPLMNs, eci bool
	}
	byInstance := make(map[int]*contract)
	for _, path := range paths {
		if path.Kind != radioPathPLMN && path.Kind != radioPathPLMNPrimary &&
			path.Kind != radioPathServingPLMNs && path.Kind != radioPathECI {
			continue
		}
		item := byInstance[path.FAPInstance]
		if item == nil {
			item = &contract{}
			byInstance[path.FAPInstance] = item
		}
		switch path.Kind {
		case radioPathPLMN:
			item.plmn = true
		case radioPathPLMNPrimary:
			item.primary = true
		case radioPathServingPLMNs:
			item.servingPLMNs = true
		case radioPathECI:
			item.eci = true
		}
	}
	if !needed {
		if len(byInstance) != 0 {
			return fmt.Errorf("access probe ECGI intent does not match evidence paths")
		}
		return nil
	}
	if len(byInstance) == 0 {
		return fmt.Errorf("access probe ECGI intent does not match evidence paths")
	}
	for _, path := range paths {
		if path.Kind == radioPathTAC && byInstance[path.FAPInstance] == nil {
			return fmt.Errorf("access probe ECGI contract is missing for FAPService.%d", path.FAPInstance)
		}
	}
	for instance, item := range byInstance {
		primaryContract := item.plmn && item.primary && !item.servingPLMNs
		servingListContract := item.servingPLMNs && !item.plmn && !item.primary
		if !item.eci || (!primaryContract && !servingListContract) {
			return fmt.Errorf("access probe ECGI contract is incomplete for FAPService.%d", instance)
		}
	}
	return nil
}

func validateServingCellDiscoveryTask(params gpsProbeTaskParams) error {
	path := strings.TrimSpace(params.DiscoveryPath)
	if params.DiscoveryStandardPath != servingCellStandardRoot ||
		path == "" || len(path) > 512 || !strings.HasSuffix(path, ".") ||
		strings.Contains(path, "{i}") || strings.Contains(path, "{n}") {
		return fmt.Errorf("serving-cell discovery path is not allowed")
	}
	if !params.NextLevel || (!params.NeedTAC && !params.NeedECGI) {
		return fmt.Errorf("serving-cell discovery intent is invalid")
	}
	if strings.TrimSpace(params.ProductClass) == "" || len(params.Names) > 0 ||
		len(params.Paths) > 0 || len(params.RadioPaths) > 0 {
		return fmt.Errorf("serving-cell discovery contract contains unexpected fields")
	}
	return nil
}

func validRadioEvidencePath(path RadioParameterPath) bool {
	if path.FAPInstance <= 0 || path.FAPInstance > maxServingCellInstances || strings.TrimSpace(path.PrivatePath) == "" {
		return false
	}
	var match []string
	switch path.Kind {
	case radioPathTAC:
		match = lteTACPathPattern.FindStringSubmatch(path.StandardPath)
	case radioPathPLMN:
		match = ltePLMNPathPattern.FindStringSubmatch(path.StandardPath)
	case radioPathPLMNPrimary:
		match = ltePLMNPrimaryPattern.FindStringSubmatch(path.StandardPath)
	case radioPathServingPLMNs:
		match = lteServingPLMNsPattern.FindStringSubmatch(path.StandardPath)
	case radioPathECI:
		match = lteECIPathPattern.FindStringSubmatch(path.StandardPath)
	default:
		return false
	}
	if len(match) != 2 {
		return false
	}
	return match[1] == fmt.Sprintf("%d", path.FAPInstance)
}

func hasRadioPathKind(paths []RadioParameterPath, kind radioPathKind) bool {
	for _, path := range paths {
		if path.Kind == kind {
			return true
		}
	}
	return false
}

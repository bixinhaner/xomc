package deviceaccess

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	devicepkg "github.com/omcgo/omcgo/internal/device"
	"github.com/omcgo/omcgo/internal/task"
	"github.com/omcgo/omcgo/pkg/tr069"
	"go.uber.org/zap"
)

const (
	gpsProbeExpiresInSeconds = 300
	gpsEvidenceLifetime      = time.Hour
)

type GPSPathResolver interface {
	ResolveMappings(ctx context.Context, productClass, softwareVersion string) ([]GPSParameterPath, error)
}

type RadioPathResolver interface {
	ResolveRadioMappings(
		ctx context.Context,
		productClass string,
		softwareVersion string,
		fapInstances []int,
		needTAC bool,
		needECGI bool,
	) ([]RadioParameterPath, error)
}

type ServingCellDiscoveryResolver interface {
	ResolveServingCellRoot(ctx context.Context, productClass, softwareVersion string) (string, error)
}

type GPSProbeRequest struct {
	Carrier           string `json:"carrier"`
	SerialNumber      string `json:"serial_number"`
	ProductClass      string `json:"product_class,omitempty"`
	SoftwareVersion   string `json:"software_version,omitempty"`
	EvidenceVersion   int64  `json:"evidence_version"`
	SourceID          string `json:"source_id,omitempty"`
	NeedGPS           bool   `json:"need_gps,omitempty"`
	NeedTAC           bool   `json:"need_tac,omitempty"`
	NeedECGI          bool   `json:"need_ecgi,omitempty"`
	FAPInstances      []int  `json:"fap_instances,omitempty"`
	instancesVerified bool
}

type GPSProbePlanner interface {
	EnsureGPSProbe(ctx context.Context, request GPSProbeRequest) error
}

type gpsProbeTaskParams struct {
	Names                 []string             `json:"names"`
	Carrier               string               `json:"carrier"`
	SerialNumber          string               `json:"serial_number"`
	EvidenceVersion       int64                `json:"evidence_version"`
	Paths                 []GPSParameterPath   `json:"evidence_paths"`
	RadioPaths            []RadioParameterPath `json:"radio_evidence_paths,omitempty"`
	NeedGPS               bool                 `json:"need_gps,omitempty"`
	NeedTAC               bool                 `json:"need_tac,omitempty"`
	NeedECGI              bool                 `json:"need_ecgi,omitempty"`
	ProductClass          string               `json:"product_class,omitempty"`
	SoftwareVersion       string               `json:"software_version,omitempty"`
	DiscoveryPath         string               `json:"path,omitempty"`
	DiscoveryStandardPath string               `json:"discovery_standard_path,omitempty"`
	NextLevel             bool                 `json:"next_level,omitempty"`
}

type gpsProbeTaskResult struct {
	PrivateParameterValues  []tr069.ParameterValueStruct `json:"private_parameter_values"`
	StandardParameterValues []tr069.ParameterValueStruct `json:"standard_parameter_values"`
	ParameterInfos          []tr069.ParameterInfoStruct  `json:"parameter_infos"`
}

type gpsEvidenceValue struct {
	Latitude           float64  `json:"latitude"`
	Longitude          float64  `json:"longitude"`
	Height             *float64 `json:"height,omitempty"`
	NumberOfSatellites *int     `json:"number_of_satellites,omitempty"`
	Status             string   `json:"status,omitempty"`
	HorizontalAccuracy *float64 `json:"horizontal_accuracy,omitempty"`
	VerticalAccuracy   *float64 `json:"vertical_accuracy,omitempty"`
}

type GPSProbeService struct {
	paths       GPSPathResolver
	tasks       task.Enqueuer
	repository  Repository
	reevaluator ReevaluationHandler
	logger      *zap.Logger
	now         func() time.Time
}

func NewGPSProbeService(
	paths GPSPathResolver,
	tasks task.Enqueuer,
	repository Repository,
	reevaluator ReevaluationHandler,
	logger *zap.Logger,
) *GPSProbeService {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &GPSProbeService{
		paths:       paths,
		tasks:       tasks,
		repository:  repository,
		reevaluator: reevaluator,
		logger:      logger.Named("device-access-gps-probe"),
		now:         time.Now,
	}
}

func (s *GPSProbeService) SetClock(now func() time.Time) {
	if now != nil {
		s.now = now
	}
}

func (s *GPSProbeService) EnsureGPSProbe(ctx context.Context, request GPSProbeRequest) error {
	if err := validateIdentity(request.Carrier, request.SerialNumber); err != nil {
		return err
	}
	if s == nil || s.paths == nil || s.tasks == nil {
		return fmt.Errorf("create GPS access probe: %w", ErrAccessEvidenceUnavailable)
	}
	if request.EvidenceVersion <= 0 {
		return fmt.Errorf("create GPS access probe: evidence version must be positive")
	}
	if !request.NeedGPS && !request.NeedTAC && !request.NeedECGI {
		request.NeedGPS = true
	}
	if (request.NeedTAC || request.NeedECGI) && !request.instancesVerified {
		return s.enqueueServingCellDiscovery(ctx, request)
	}
	return s.enqueueValueProbe(ctx, request)
}

func (s *GPSProbeService) enqueueServingCellDiscovery(ctx context.Context, request GPSProbeRequest) error {
	resolver, ok := s.paths.(ServingCellDiscoveryResolver)
	if !ok {
		return fmt.Errorf("resolve serving-cell discovery path: %w", ErrAccessEvidenceUnavailable)
	}
	path, err := resolver.ResolveServingCellRoot(ctx, request.ProductClass, request.SoftwareVersion)
	if err != nil {
		return fmt.Errorf("resolve serving-cell discovery path: %w", err)
	}
	params, err := json.Marshal(gpsProbeTaskParams{
		Carrier:               request.Carrier,
		SerialNumber:          request.SerialNumber,
		EvidenceVersion:       request.EvidenceVersion,
		NeedGPS:               request.NeedGPS,
		NeedTAC:               request.NeedTAC,
		NeedECGI:              request.NeedECGI,
		ProductClass:          request.ProductClass,
		SoftwareVersion:       request.SoftwareVersion,
		DiscoveryPath:         path,
		DiscoveryStandardPath: servingCellStandardRoot,
		NextLevel:             true,
	})
	if err != nil {
		return fmt.Errorf("encode serving-cell discovery task: %w", err)
	}
	maxRetries := 1
	if _, err := createProbeTask(ctx, s.tasks, &task.CreateTaskRequest{
		DeviceSN:       request.SerialNumber,
		Method:         "GetParameterNames",
		Params:         params,
		Priority:       5,
		ExpiresIn:      gpsProbeExpiresInSeconds,
		MaxRetries:     &maxRetries,
		CommandKey:     probeCommandKey("discovery", params),
		Source:         task.TaskSourceDeviceAccess,
		SourceID:       request.SourceID,
		Description:    "device access serving-cell discovery",
		AdmissionClass: task.AdmissionClassAccessProbe,
	}); err != nil {
		return fmt.Errorf("enqueue serving-cell discovery: %w", err)
	}
	return nil
}

func (s *GPSProbeService) enqueueValueProbe(ctx context.Context, request GPSProbeRequest) error {
	var mappings []GPSParameterPath
	var err error
	if request.NeedGPS {
		mappings, err = s.paths.ResolveMappings(ctx, request.ProductClass, request.SoftwareVersion)
		if err != nil {
			return fmt.Errorf("resolve GPS access probe paths: %w", err)
		}
		if !hasGPSCoordinatePaths(mappings) {
			return fmt.Errorf("resolve GPS latitude and longitude paths: %w", ErrAccessEvidenceUnavailable)
		}
	}
	var radioMappings []RadioParameterPath
	if request.NeedTAC || request.NeedECGI {
		resolver, ok := s.paths.(RadioPathResolver)
		if !ok {
			return fmt.Errorf("resolve radio access probe paths: %w", ErrAccessEvidenceUnavailable)
		}
		radioMappings, err = resolver.ResolveRadioMappings(
			ctx,
			request.ProductClass,
			request.SoftwareVersion,
			request.FAPInstances,
			request.NeedTAC,
			request.NeedECGI,
		)
		if err != nil {
			return fmt.Errorf("resolve radio access probe paths: %w", err)
		}
	}
	names := make([]string, 0, len(mappings)+len(radioMappings))
	for _, mapping := range mappings {
		names = append(names, mapping.PrivatePath)
	}
	for _, mapping := range radioMappings {
		names = append(names, mapping.PrivatePath)
	}
	params, err := json.Marshal(gpsProbeTaskParams{
		Names:           names,
		Carrier:         request.Carrier,
		SerialNumber:    request.SerialNumber,
		EvidenceVersion: request.EvidenceVersion,
		Paths:           mappings,
		RadioPaths:      radioMappings,
		NeedGPS:         request.NeedGPS,
		NeedTAC:         request.NeedTAC,
		NeedECGI:        request.NeedECGI,
	})
	if err != nil {
		return fmt.Errorf("encode GPS access probe task: %w", err)
	}
	maxRetries := 1
	if _, err := createProbeTask(ctx, s.tasks, &task.CreateTaskRequest{
		DeviceSN:       request.SerialNumber,
		Method:         "GetParameterValues",
		Params:         params,
		Priority:       5,
		ExpiresIn:      gpsProbeExpiresInSeconds,
		MaxRetries:     &maxRetries,
		CommandKey:     probeCommandKey("values", params),
		Source:         task.TaskSourceDeviceAccess,
		SourceID:       request.SourceID,
		Description:    "device access evidence probe",
		AdmissionClass: task.AdmissionClassAccessProbe,
	}); err != nil {
		return fmt.Errorf("enqueue GPS access probe: %w", err)
	}
	return nil
}

func createProbeTask(ctx context.Context, enqueuer task.Enqueuer, request *task.CreateTaskRequest) (*task.Task, error) {
	if reliable, ok := enqueuer.(task.IdempotentEnqueuer); ok {
		return reliable.EnsureTaskByCommandKey(ctx, request)
	}
	return enqueuer.CreateTask(ctx, request)
}

func probeCommandKey(kind string, contract []byte) string {
	digest := sha256.Sum256(contract)
	return fmt.Sprintf("access-evidence:%s:%x", kind, digest[:16])
}

func (s *GPSProbeService) HandleCompleted(ctx context.Context, completed *task.Task) error {
	if completed == nil ||
		completed.Source != task.TaskSourceDeviceAccess ||
		completed.AdmissionClass != task.AdmissionClassAccessProbe ||
		(completed.Method != "GetParameterValues" && completed.Method != "GetParameterNames") ||
		!isTerminalProbeStatus(completed.Status) {
		return nil
	}
	if s == nil || s.repository == nil {
		return fmt.Errorf("project GPS access evidence: %w", ErrAccessEvidenceUnavailable)
	}
	var params gpsProbeTaskParams
	if err := json.Unmarshal(completed.Params, &params); err != nil {
		return fmt.Errorf("decode GPS access probe params: %w", err)
	}
	if err := validateIdentity(params.Carrier, params.SerialNumber); err != nil {
		return fmt.Errorf("validate GPS access probe identity: %w", err)
	}
	if params.SerialNumber != completed.DeviceSN {
		return fmt.Errorf("GPS access probe task serial number mismatch")
	}
	taskID, err := uuid.Parse(completed.ID)
	if err != nil {
		return fmt.Errorf("parse access probe task id: %w", err)
	}
	observedAt := s.now().UTC()
	if completed.Method == "GetParameterNames" {
		return s.handleServingCellDiscovery(ctx, completed, params, taskID, observedAt)
	}
	replacements, projectionErr := successfulProbeEvidence(params, completed.Result, taskID, observedAt)
	if completed.Status != task.TaskStatusCompleted || projectionErr != nil {
		replacements, err = failedProbeEvidence(params, taskID, observedAt)
		if err != nil {
			return err
		}
		if projectionErr != nil {
			s.logger.Warn("access probe completed without usable evidence",
				zap.String("task_id", completed.ID), zap.Error(projectionErr))
		}
	}
	return s.projectProbeEvidence(ctx, params, replacements)
}

func (s *GPSProbeService) handleServingCellDiscovery(
	ctx context.Context,
	completed *task.Task,
	params gpsProbeTaskParams,
	taskID uuid.UUID,
	observedAt time.Time,
) error {
	if completed.Status == task.TaskStatusCompleted {
		instances, err := servingCellInstancesFromDiscovery(params.DiscoveryPath, completed.Result)
		if err == nil {
			err = s.enqueueValueProbe(ctx, GPSProbeRequest{
				Carrier:           params.Carrier,
				SerialNumber:      params.SerialNumber,
				ProductClass:      params.ProductClass,
				SoftwareVersion:   params.SoftwareVersion,
				EvidenceVersion:   params.EvidenceVersion,
				SourceID:          completed.SourceID,
				NeedGPS:           params.NeedGPS,
				NeedTAC:           params.NeedTAC,
				NeedECGI:          params.NeedECGI,
				FAPInstances:      instances,
				instancesVerified: true,
			})
		}
		if err == nil {
			return nil
		}
		s.logger.Warn("serving-cell discovery did not produce a value probe",
			zap.String("task_id", completed.ID), zap.Error(err))
	}
	replacements, err := failedProbeEvidence(params, taskID, observedAt)
	if err != nil {
		return err
	}
	return s.projectProbeEvidence(ctx, params, replacements)
}

func servingCellInstancesFromDiscovery(path string, rawResult json.RawMessage) ([]int, error) {
	path = strings.TrimSpace(path)
	if path == "" || !strings.HasSuffix(path, ".") {
		return nil, fmt.Errorf("serving-cell discovery path is invalid: %w", ErrAccessEvidenceUnavailable)
	}
	var result gpsProbeTaskResult
	if err := json.Unmarshal(rawResult, &result); err != nil {
		return nil, fmt.Errorf("decode serving-cell discovery result: %w", err)
	}
	seen := make(map[int]struct{})
	for _, info := range result.ParameterInfos {
		name := strings.TrimSpace(info.Name)
		if !strings.HasPrefix(name, path) {
			continue
		}
		remainder := strings.TrimPrefix(name, path)
		if !strings.HasSuffix(remainder, ".") || strings.Count(remainder, ".") != 1 {
			continue
		}
		instance, err := strconv.Atoi(strings.TrimSuffix(remainder, "."))
		if err != nil || instance <= 0 {
			continue
		}
		if instance > maxServingCellInstances {
			return nil, fmt.Errorf(
				"serving-cell instance %d exceeds access-probe bound %d: %w",
				instance, maxServingCellInstances, ErrAccessEvidenceUnavailable,
			)
		}
		seen[instance] = struct{}{}
	}
	instances := make([]int, 0, len(seen))
	for instance := range seen {
		instances = append(instances, instance)
	}
	instances = sortedPositiveInstances(instances)
	if len(instances) == 0 {
		return nil, fmt.Errorf("serving-cell discovery returned no bounded instances: %w", ErrAccessEvidenceUnavailable)
	}
	return instances, nil
}

func (s *GPSProbeService) projectProbeEvidence(
	ctx context.Context,
	params gpsProbeTaskParams,
	replacements []EvidenceRecord,
) error {
	current, err := s.repository.LoadEvaluationContext(ctx, params.Carrier, params.SerialNumber)
	if err != nil {
		return fmt.Errorf("load current evidence before access probe projection: %w", err)
	}
	triggerEventID := probeProjectionEventID(replacements)
	if triggerEventID != "" && evidenceContainsProbeProjection(current.Evidence.Records, replacements) {
		return s.reevaluateProbeEvidence(ctx, params, triggerEventID)
	}
	version := params.EvidenceVersion
	if version <= current.Evidence.Version {
		version = current.Evidence.Version + 1
	}
	batch := EvidenceBatch{
		Carrier:      params.Carrier,
		SerialNumber: params.SerialNumber,
		Version:      version,
		Records:      mergeEvidenceRecords(current.Evidence.Records, replacements),
	}
	if _, err := s.repository.AppendEvidence(ctx, batch); err != nil {
		return fmt.Errorf("append access probe evidence: %w", err)
	}
	return s.reevaluateProbeEvidence(ctx, params, triggerEventID)
}

func (s *GPSProbeService) reevaluateProbeEvidence(
	ctx context.Context,
	params gpsProbeTaskParams,
	triggerEventID string,
) error {
	if s.reevaluator == nil {
		return fmt.Errorf("reevaluate access probe evidence: %w", ErrAccessGateDependencyMissing)
	}
	if err := s.reevaluator.Handle(ctx, ReevaluationRequest{
		Carrier:                 params.Carrier,
		SerialNumber:            params.SerialNumber,
		TriggerType:             "access_probe_evidence",
		TriggerEventID:          triggerEventID,
		ObservedProductClass:    params.ProductClass,
		ObservedSoftwareVersion: params.SoftwareVersion,
	}); err != nil {
		return fmt.Errorf("reevaluate after access probe evidence: %w", err)
	}
	return nil
}

func probeProjectionEventID(records []EvidenceRecord) string {
	for _, record := range records {
		if record.TaskID != nil && *record.TaskID != uuid.Nil {
			return "access-probe:" + record.TaskID.String()
		}
	}
	return ""
}

func evidenceContainsProbeProjection(current, replacements []EvidenceRecord) bool {
	if len(replacements) == 0 {
		return false
	}
	wanted := make(map[ConditionType]uuid.UUID, len(replacements))
	for _, record := range replacements {
		if record.TaskID == nil || *record.TaskID == uuid.Nil {
			return false
		}
		wanted[record.Type] = *record.TaskID
	}
	for _, record := range current {
		if taskID, ok := wanted[record.Type]; ok && record.TaskID != nil && *record.TaskID == taskID {
			delete(wanted, record.Type)
		}
	}
	return len(wanted) == 0
}

func isTerminalProbeStatus(status task.TaskStatus) bool {
	switch status {
	case task.TaskStatusCompleted, task.TaskStatusFailed, task.TaskStatusExpired, task.TaskStatusCancelled:
		return true
	default:
		return false
	}
}

func successfulProbeEvidence(
	params gpsProbeTaskParams,
	rawResult json.RawMessage,
	taskID uuid.UUID,
	observedAt time.Time,
) ([]EvidenceRecord, error) {
	var result gpsProbeTaskResult
	if err := json.Unmarshal(rawResult, &result); err != nil {
		return nil, fmt.Errorf("decode GPS access probe result: %w", err)
	}
	replacements := make([]EvidenceRecord, 0, 3)
	if params.NeedGPS || len(params.Paths) > 0 {
		values := result.PrivateParameterValues
		if len(values) == 0 {
			values = result.StandardParameterValues
		}
		normalized, err := normalizeGPSEvidence(params.Paths, values)
		if err != nil {
			return nil, err
		}
		normalizedJSON, err := json.Marshal(normalized)
		if err != nil {
			return nil, fmt.Errorf("encode normalized GPS evidence: %w", err)
		}
		expiresAt := observedAt.Add(gpsEvidenceLifetime)
		replacements = append(replacements, EvidenceRecord{
			Type:            ConditionTypeGPS,
			Status:          EvidenceStatusAvailable,
			NormalizedValue: normalizedJSON,
			ValueHash:       evidenceHash(normalizedJSON),
			Source:          "gpv",
			TaskID:          &taskID,
			ObservedAt:      observedAt,
			ExpiresAt:       &expiresAt,
		})
	}
	if params.NeedTAC || params.NeedECGI {
		values := append([]tr069.ParameterValueStruct(nil), result.PrivateParameterValues...)
		values = append(values, result.StandardParameterValues...)
		normalized, err := normalizeGPVRadioEvidence(
			params.RadioPaths,
			values,
			params.NeedTAC,
			params.NeedECGI,
		)
		if err != nil {
			return nil, err
		}
		radioRecords, err := radioEvidenceRecords(normalized, "gpv", observedAt)
		if err != nil {
			return nil, err
		}
		for i := range radioRecords {
			radioRecords[i].Status = EvidenceStatusAvailable
			radioRecords[i].TaskID = &taskID
		}
		replacements = append(replacements, radioRecords...)
	}
	return replacements, nil
}

func failedProbeEvidence(params gpsProbeTaskParams, taskID uuid.UUID, observedAt time.Time) ([]EvidenceRecord, error) {
	types := make([]ConditionType, 0, 3)
	if params.NeedGPS || len(params.Paths) > 0 {
		types = append(types, ConditionTypeGPS)
	}
	if params.NeedTAC || hasRadioPathKind(params.RadioPaths, radioPathTAC) {
		types = append(types, ConditionTypeTAC)
	}
	if params.NeedECGI || hasRadioPathKind(params.RadioPaths, radioPathPLMN) || hasRadioPathKind(params.RadioPaths, radioPathECI) {
		types = append(types, ConditionTypeECGI)
	}
	if len(types) == 0 {
		return nil, fmt.Errorf("access probe declares no evidence target: %w", ErrAccessEvidenceUnavailable)
	}
	records := make([]EvidenceRecord, 0, len(types))
	for _, evidenceType := range types {
		raw, err := json.Marshal(map[string]string{"status": string(EvidenceStatusCollectionFailed)})
		if err != nil {
			return nil, fmt.Errorf("encode %s collection failure: %w", evidenceType, err)
		}
		records = append(records, EvidenceRecord{
			Type:            evidenceType,
			Status:          EvidenceStatusCollectionFailed,
			NormalizedValue: raw,
			ValueHash:       evidenceHash(append([]byte(string(evidenceType)+":"), raw...)),
			Source:          "gpv",
			TaskID:          &taskID,
			ObservedAt:      observedAt,
		})
	}
	return records, nil
}

func (s *GPSProbeService) OnTaskCompleted(ctx context.Context, completed *task.Task) {
	if err := s.OnTaskCompletedReliable(ctx, completed); err != nil {
		s.logger.Error("project completed GPS access probe", zap.String("task_id", taskID(completed)), zap.Error(err))
	}
}

func (s *GPSProbeService) OnTaskCompletedReliable(ctx context.Context, completed *task.Task) error {
	return s.HandleCompleted(ctx, completed)
}

func hasGPSCoordinatePaths(paths []GPSParameterPath) bool {
	var latitude, longitude bool
	for _, path := range paths {
		switch path.StandardPath {
		case gpsLatitudeStandardPath:
			latitude = strings.TrimSpace(path.PrivatePath) != ""
		case gpsLongitudeStandardPath:
			longitude = strings.TrimSpace(path.PrivatePath) != ""
		}
	}
	return latitude && longitude
}

func normalizeGPSEvidence(paths []GPSParameterPath, parameters []tr069.ParameterValueStruct) (gpsEvidenceValue, error) {
	values := make(map[string]string, len(parameters))
	for _, parameter := range parameters {
		values[parameter.Name] = strings.TrimSpace(parameter.Value)
	}
	byStandard := make(map[string]string, len(paths))
	for _, path := range paths {
		value, ok := values[path.PrivatePath]
		if !ok {
			value, ok = values[path.StandardPath]
		}
		if ok {
			byStandard[path.StandardPath] = value
		}
	}
	latitude, err := requiredCoordinate(byStandard[gpsLatitudeStandardPath], -90, 90, "latitude")
	if err != nil {
		return gpsEvidenceValue{}, err
	}
	longitude, err := requiredCoordinate(byStandard[gpsLongitudeStandardPath], -180, 180, "longitude")
	if err != nil {
		return gpsEvidenceValue{}, err
	}
	return gpsEvidenceValue{
		Latitude:           latitude,
		Longitude:          longitude,
		Height:             optionalFloat(byStandard["Device.FAP.GPS.Height"]),
		NumberOfSatellites: optionalInt(byStandard["Device.FAP.GPS.NumberOfSatellites"]),
		Status:             byStandard["Device.DeviceInfo.GPS_Status"],
		HorizontalAccuracy: optionalFloat(byStandard["Device.DeviceInfo.GPS.horizontalAccuracy"]),
		VerticalAccuracy:   optionalFloat(byStandard["Device.DeviceInfo.GPS.verticalAccuracy"]),
	}, nil
}

func requiredCoordinate(value string, minimum, maximum float64, name string) (float64, error) {
	maxAbs := maximum
	if -minimum > maxAbs {
		maxAbs = -minimum
	}
	parsed, ok := devicepkg.ParseGPSCoordinate(value, maxAbs)
	if !ok || parsed < minimum || parsed > maximum {
		return 0, fmt.Errorf("GPS %s is missing or invalid: %w", name, ErrAccessEvidenceUnavailable)
	}
	return parsed, nil
}

func optionalFloat(value string) *float64 {
	parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	if err != nil {
		return nil
	}
	return &parsed
}

func optionalInt(value string) *int {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || parsed < 0 {
		return nil
	}
	return &parsed
}

func taskID(completed *task.Task) string {
	if completed == nil {
		return ""
	}
	return completed.ID
}

func evidenceHash(value []byte) string {
	digest := sha256.Sum256(value)
	return "sha256:" + hex.EncodeToString(digest[:])
}

var _ GPSProbePlanner = (*GPSProbeService)(nil)
var _ task.TaskCompletionCallback = (*GPSProbeService)(nil)

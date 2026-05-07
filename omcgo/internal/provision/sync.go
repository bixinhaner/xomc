package provision

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/omcgo/omcgo/internal/config/datamodel"
	"github.com/omcgo/omcgo/internal/config/parammodel"
	"github.com/omcgo/omcgo/internal/core/appconfig"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/device"
	"github.com/omcgo/omcgo/internal/product"
	"github.com/omcgo/omcgo/internal/task"
	"github.com/omcgo/omcgo/pkg/tr069"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// SyncService handles batch parameter value synchronization from devices.
//
// T-0098 P2-04：双栈期 dataModel 与 paramRegistry 共存。当 paramRegistryEnabled 且
// productRegistry/paramRegistry 注入、且 device.ProductClass 能命中 product 时，走
// Path B 新栈：删 GPN 阶段 + ParamMapping 列表去重前缀 + Translator 落库 + is_storable
// 过滤；否则降级到既有 dataModel 双阶段路径（StartTwoPhaseSync / HandleGPNResult /
// HandleSyncResult）。
type SyncService struct {
	paramRepo            device.DeviceParameterRepository
	discoveryRepo        ParameterDiscoveryLogRepository
	taskSvc              task.Enqueuer
	planStore            *SyncPlanStore
	paramRegistry        *parammodel.Registry
	productRegistry      *product.Registry
	paramRegistryEnabled bool
	config               appconfig.AutoSyncConfig
	batchSize            int
	logger               *zap.Logger
}

// NewSyncService creates a new SyncService.
func NewSyncService(
	paramRepo device.DeviceParameterRepository,
	discoveryRepo ParameterDiscoveryLogRepository,
	taskSvc task.Enqueuer,
	planStore *SyncPlanStore,
	config appconfig.AutoSyncConfig,
	batchSize int,
	logger *zap.Logger,
) *SyncService {
	if batchSize <= 0 {
		batchSize = 50
	}
	return &SyncService{
		paramRepo:     paramRepo,
		discoveryRepo: discoveryRepo,
		taskSvc:       taskSvc,
		planStore:     planStore,
		config:        config,
		batchSize:     batchSize,
		logger:        logger,
	}
}

// StartSync initiates a full parameter value sync by enqueuing batched
// GetParameterValues commands for the device's root object.
func (s *SyncService) StartSync(ctx context.Context, dev *model.Device, paramPaths []string) error {
	// Update discovery log status to syncing if exists.
	log, _ := s.discoveryRepo.GetByDeviceID(ctx, dev.ID)
	if log != nil {
		_ = s.discoveryRepo.UpdateStatus(ctx, log.ID, DiscoverySyncing, "")
	}

	// Batch parameter paths into GPV commands.
	batches := batchPaths(paramPaths, s.batchSize)
	for i, batch := range batches {
		gpvParams, err := json.Marshal(map[string]interface{}{
			"names": batch,
		})
		if err != nil {
			return fmt.Errorf("marshal GPV batch %d: %w", i, err)
		}

		if _, err := s.taskSvc.CreateTask(ctx, &task.CreateTaskRequest{
			DeviceSN:   dev.SerialNumber,
			Method:     MethodGetParameterValues,
			Params:     gpvParams,
			Priority:   10 + i, // Lower priority than discovery commands.
			CommandKey: fmt.Sprintf("sync-gpv-%s-%d", dev.SerialNumber, i),
			Source:     task.TaskSourceSystem,
		}); err != nil {
			return fmt.Errorf("enqueue GPV batch %d: %w", i, err)
		}
	}

	s.logger.Info("parameter sync started",
		zap.String("device_sn", dev.SerialNumber),
		zap.Int("total_params", len(paramPaths)),
		zap.Int("batches", len(batches)),
	)

	return nil
}

// HandleSyncResult processes a GPV response and upserts parameter values into the database.
func (s *SyncService) HandleSyncResult(ctx context.Context, dev *model.Device,
	paramValues []tr069.ParameterValueStruct) error {

	if len(paramValues) == 0 {
		return nil
	}

	params := make([]model.DeviceParameter, 0, len(paramValues))
	now := time.Now()
	for _, pv := range paramValues {
		params = append(params, model.DeviceParameter{
			DeviceID:       dev.ID,
			ParameterPath:  pv.Name,
			ParameterValue: pv.Value,
			ParameterType:  inferParameterType(pv.Value, pv.Type),
			Writable:       false, // Will be updated from GPN data if available.
			LastUpdatedAt:  now,
		})
	}

	if err := s.paramRepo.BatchUpsert(ctx, dev.ID, params); err != nil {
		return fmt.Errorf("batch upsert parameters: %w", err)
	}

	s.logger.Debug("parameters synced",
		zap.String("device_sn", dev.SerialNumber),
		zap.Int("count", len(params)),
	)

	return nil
}

// StartTwoPhaseSync initiates a two-phase parameter sync using the data model iterator.
// Phase 1: GPN requests discover actual multi-instance objects.
// Phase 2: GPV requests with partial path prefixes fetch all parameter values.
func (s *SyncService) StartTwoPhaseSync(ctx context.Context, dev *model.Device, dm *datamodel.DataModel) error {
	iterator, err := datamodel.NewParameterTreeIterator(dm)
	if err != nil {
		return fmt.Errorf("build parameter tree iterator: %w", err)
	}

	plan := iterator.BuildSyncPlan()

	// If no multi-instance objects, skip Phase 1 and directly use static prefixes.
	if len(plan.Phase1GPNs) == 0 {
		return s.enqueueGPVPrefixes(ctx, dev, plan.StaticPrefixes)
	}

	// Phase 1: Enqueue depth-0 GPN requests.
	depth0GPNs := iterator.ExpandGPNsForDepth(plan, 0, datamodel.InstanceMap{})
	for i, gpn := range depth0GPNs {
		gpnParams, err := json.Marshal(map[string]interface{}{
			"path":       gpn.BasePath,
			"next_level": true,
		})
		if err != nil {
			return fmt.Errorf("marshal GPN %d: %w", i, err)
		}

		if _, err := s.taskSvc.CreateTask(ctx, &task.CreateTaskRequest{
			DeviceSN:   dev.SerialNumber,
			Method:     "GetParameterNames",
			Params:     gpnParams,
			Priority:   5 + i,
			CommandKey: fmt.Sprintf("sync-gpn-%s-%d", dev.SerialNumber, i),
			Source:     task.TaskSourceSystem,
		}); err != nil {
			return fmt.Errorf("enqueue GPN %d: %w", i, err)
		}
	}

	// Save sync plan context to Redis for GPN response continuation.
	planData := syncPlanState{
		StaticPrefixes:  plan.StaticPrefixes,
		Phase1GPNs:      plan.Phase1GPNs,
		Instances:       make(datamodel.InstanceMap),
		PendingGPNCount: len(depth0GPNs),
		MaxDepth:        iterator.MaxDepth(),
		CurrentDepth:    0,
	}

	planJSON, _ := json.Marshal(planData)
	s.saveSyncPlan(ctx, dev.SerialNumber, planJSON)

	s.logger.Info("two-phase sync started",
		zap.String("device_sn", dev.SerialNumber),
		zap.Int("gpn_requests", len(depth0GPNs)),
		zap.Int("static_prefixes", len(plan.StaticPrefixes)),
		zap.Int("max_depth", iterator.MaxDepth()),
	)

	return nil
}

// syncPlanState is the serializable state for a running two-phase sync.
type syncPlanState struct {
	StaticPrefixes  []string                `json:"static_prefixes"`
	Phase1GPNs      []datamodel.GPNRequest  `json:"phase1_gpns"`
	Instances       datamodel.InstanceMap   `json:"instances"`
	PendingGPNCount int                     `json:"pending_gpn_count"`
	MaxDepth        int                     `json:"max_depth"`
	CurrentDepth    int                     `json:"current_depth"`
}

// HandleGPNResult processes a GPN response and continues the two-phase sync.
func (s *SyncService) HandleGPNResult(ctx context.Context, dev *model.Device,
	gpnPath string, paramInfos []tr069.ParameterInfoStruct) error {

	// Load sync plan from Redis.
	planJSON := s.loadSyncPlan(ctx, dev.SerialNumber)
	if planJSON == nil {
		s.logger.Debug("no sync plan found for GPN response, ignoring",
			zap.String("device_sn", dev.SerialNumber),
			zap.String("gpn_path", gpnPath),
		)
		return nil
	}

	var state syncPlanState
	if err := json.Unmarshal(planJSON, &state); err != nil {
		return fmt.Errorf("unmarshal sync plan: %w", err)
	}

	// Parse instance numbers from GPN response.
	instanceNums := parseGPNInstances(gpnPath, paramInfos)
	if len(instanceNums) > 0 {
		state.Instances[gpnPath] = instanceNums
	}

	state.PendingGPNCount--

	// Check if all GPNs for current depth are done.
	if state.PendingGPNCount <= 0 && state.CurrentDepth < state.MaxDepth {
		// Expand next depth GPNs.
		state.CurrentDepth++
		plan := &datamodel.SyncPlan{
			Phase1GPNs:     state.Phase1GPNs,
			StaticPrefixes: state.StaticPrefixes,
		}

		// Rebuild iterator to expand next depth.
		nextGPNs := expandGPNsFromState(plan, state.CurrentDepth, state.Instances)

		if len(nextGPNs) > 0 {
			state.PendingGPNCount = len(nextGPNs)
			for i, gpn := range nextGPNs {
				gpnParams, _ := json.Marshal(map[string]interface{}{
					"path":       gpn.BasePath,
					"next_level": true,
				})
				if _, err := s.taskSvc.CreateTask(ctx, &task.CreateTaskRequest{
					DeviceSN:   dev.SerialNumber,
					Method:     "GetParameterNames",
					Params:     gpnParams,
					Priority:   5 + i,
					CommandKey: fmt.Sprintf("sync-gpn-%s-d%d-%d", dev.SerialNumber, state.CurrentDepth, i),
					Source:     task.TaskSourceSystem,
				}); err != nil {
					s.logger.Error("enqueue depth GPN", zap.Error(err))
				}
			}

			// Save updated state.
			planJSON, _ = json.Marshal(state)
			s.saveSyncPlan(ctx, dev.SerialNumber, planJSON)
			return nil
		}
	}

	if state.PendingGPNCount <= 0 {
		// All GPNs done. Build GPV prefixes and start Phase 2.
		gpvPrefixes := buildGPVPrefixesFromState(state)
		s.clearSyncPlan(ctx, dev.SerialNumber)

		s.logger.Info("GPN discovery complete, starting GPV phase",
			zap.String("device_sn", dev.SerialNumber),
			zap.Int("gpv_prefixes", len(gpvPrefixes)),
			zap.Any("instances", state.Instances),
		)

		return s.enqueueGPVPrefixes(ctx, dev, gpvPrefixes)
	}

	// Save updated state.
	planJSON, _ = json.Marshal(state)
	s.saveSyncPlan(ctx, dev.SerialNumber, planJSON)
	return nil
}

// enqueueGPVPrefixes enqueues GPV commands using partial path prefixes.
func (s *SyncService) enqueueGPVPrefixes(ctx context.Context, dev *model.Device, prefixes []string) error {
	// Update discovery log.
	log, _ := s.discoveryRepo.GetByDeviceID(ctx, dev.ID)
	if log != nil {
		_ = s.discoveryRepo.UpdateStatus(ctx, log.ID, DiscoverySyncing, "")
	}

	batches := batchPaths(prefixes, s.batchSize)
	for i, batch := range batches {
		gpvParams, err := json.Marshal(map[string]interface{}{
			"names": batch,
		})
		if err != nil {
			return fmt.Errorf("marshal GPV prefix batch %d: %w", i, err)
		}

		if _, err := s.taskSvc.CreateTask(ctx, &task.CreateTaskRequest{
			DeviceSN:   dev.SerialNumber,
			Method:     MethodGetParameterValues,
			Params:     gpvParams,
			Priority:   10 + i,
			CommandKey: fmt.Sprintf("sync-gpv-%s-%d", dev.SerialNumber, i),
			Source:     task.TaskSourceSystem,
		}); err != nil {
			return fmt.Errorf("enqueue GPV batch %d: %w", i, err)
		}
	}

	s.logger.Info("GPV prefix sync enqueued",
		zap.String("device_sn", dev.SerialNumber),
		zap.Int("prefixes", len(prefixes)),
		zap.Int("batches", len(batches)),
	)

	return nil
}

// parseGPNInstances extracts instance numbers from a GPN response.
func parseGPNInstances(basePath string, paramInfos []tr069.ParameterInfoStruct) []int {
	instanceSet := make(map[int]bool)
	for _, pi := range paramInfos {
		suffix := strings.TrimPrefix(pi.Name, basePath)
		suffix = strings.TrimSuffix(suffix, ".")
		if num, err := strconv.Atoi(suffix); err == nil {
			instanceSet[num] = true
		}
	}
	instances := make([]int, 0, len(instanceSet))
	for num := range instanceSet {
		instances = append(instances, num)
	}
	return instances
}

// expandGPNsFromState expands GPN requests for a given depth using discovered instances.
func expandGPNsFromState(plan *datamodel.SyncPlan, depth int, instances datamodel.InstanceMap) []datamodel.GPNRequest {
	var result []datamodel.GPNRequest
	for _, gpn := range plan.Phase1GPNs {
		if gpn.Depth != depth {
			continue
		}
		// Replace placeholders with discovered instances.
		templatePath := gpn.TemplatePath
		expandedPaths := expandTemplateWithInstances(templatePath, instances)
		for _, ep := range expandedPaths {
			basePath := datamodel.TemplateToBasePath(ep)
			if !datamodel.ContainsPlaceholder(basePath) {
				result = append(result, datamodel.GPNRequest{
					TemplatePath: ep,
					BasePath:     basePath,
					Depth:        depth,
					NextLevel:    true,
				})
			}
		}
	}
	return result
}

// expandTemplateWithInstances replaces {N} placeholders with all discovered instances.
func expandTemplateWithInstances(templatePath string, instances datamodel.InstanceMap) []string {
	if !datamodel.ContainsPlaceholder(templatePath) {
		return []string{templatePath}
	}

	basePath := datamodel.TemplateToBasePath(templatePath)
	insts, ok := instances[basePath]
	if !ok || len(insts) == 0 {
		return nil
	}

	var result []string
	for _, inst := range insts {
		expanded := datamodel.ReplaceInstance(templatePath, 0, inst)
		// Recursively expand remaining placeholders.
		subPaths := expandTemplateWithInstances(expanded, instances)
		result = append(result, subPaths...)
	}
	return result
}

// buildGPVPrefixesFromState generates GPV prefixes from completed sync state.
func buildGPVPrefixesFromState(state syncPlanState) []string {
	var prefixes []string
	prefixes = append(prefixes, state.StaticPrefixes...)

	// Add instance prefixes for top-level multi-instance objects.
	for basePath, insts := range state.Instances {
		// Only include top-level instances (depth 0 GPNs).
		isDepth0 := false
		for _, gpn := range state.Phase1GPNs {
			if gpn.BasePath == basePath && gpn.Depth == 0 {
				isDepth0 = true
				break
			}
		}
		if isDepth0 {
			for _, inst := range insts {
				prefixes = append(prefixes, fmt.Sprintf("%s%d.", basePath, inst))
			}
		}
	}

	return prefixes
}

// Sync plan state helpers delegate to SyncPlanStore (plain Redis STRING + TTL).

func (s *SyncService) saveSyncPlan(ctx context.Context, deviceSN string, data []byte) {
	if s.planStore == nil {
		return
	}
	if err := s.planStore.Save(ctx, deviceSN, data); err != nil {
		s.logger.Warn("save sync plan", zap.String("device_sn", deviceSN), zap.Error(err))
	}
}

func (s *SyncService) loadSyncPlan(ctx context.Context, deviceSN string) []byte {
	if s.planStore == nil {
		return nil
	}
	return s.planStore.Load(ctx, deviceSN)
}

func (s *SyncService) clearSyncPlan(ctx context.Context, deviceSN string) {
	if s.planStore == nil {
		return
	}
	if err := s.planStore.Clear(ctx, deviceSN); err != nil {
		s.logger.Warn("clear sync plan", zap.String("device_sn", deviceSN), zap.Error(err))
	}
}

// CompleteSyncLog marks the discovery log as completed after all sync batches finish.
func (s *SyncService) CompleteSyncLog(ctx context.Context, deviceID uuid.UUID) error {
	log, err := s.discoveryRepo.GetByDeviceID(ctx, deviceID)
	if err != nil {
		return fmt.Errorf("get discovery log: %w", err)
	}
	if log != nil && log.Status == DiscoverySyncing {
		return s.discoveryRepo.UpdateStatus(ctx, log.ID, DiscoveryCompleted, "")
	}
	return nil
}

// batchPaths splits parameter paths into batches of the given size.
func batchPaths(paths []string, batchSize int) [][]string {
	var batches [][]string
	for i := 0; i < len(paths); i += batchSize {
		end := i + batchSize
		if end > len(paths) {
			end = len(paths)
		}
		batches = append(batches, paths[i:end])
	}
	return batches
}

// inferParameterType tries to determine the TR069 parameter type from value and SOAP type hint.
func inferParameterType(value, soapType string) model.ParameterType {
	switch soapType {
	case "xsd:string", "string":
		return model.ParameterType("string")
	case "xsd:unsignedInt", "unsignedInt":
		return model.ParameterType("unsignedInt")
	case "xsd:int", "int":
		return model.ParameterType("int")
	case "xsd:boolean", "boolean":
		return model.ParameterType("boolean")
	case "xsd:dateTime", "dateTime":
		return model.ParameterType("dateTime")
	default:
		return model.ParameterType("string")
	}
}

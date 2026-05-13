package provision

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

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
// T-0098 P5-01：旧 datamodel 双阶段同步（StartTwoPhaseSync / HandleGPNResult /
// iterator）已全部删除。仅保留 Path B：通过 paramRegistry 获取 MappingSet，按
// is_storable=true 抽取去重对象前缀直接 GPV，CPE 自动展开实例（设计 §1.11）。
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
func (s *SyncService) StartSync(ctx context.Context, dev *model.Device, paramPaths []string, sourceID string) error {
	log, _ := s.discoveryRepo.GetByDeviceID(ctx, dev.ID)
	if log != nil {
		_ = s.discoveryRepo.UpdateStatus(ctx, log.ID, DiscoverySyncing, "")
	}

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
			Priority:   10 + i,
			CommandKey: fmt.Sprintf("sync-gpv-%s-%d", dev.SerialNumber, i),
			Source:     task.TaskSourceSystem,
			SourceID:   sourceID,
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
//
// T-0098 P5-01：legacy 路径——按 device 端 privatePath 直写，无 standardPath 翻译。
// 新栈 Path B 走 HandleSyncResultPathB，会用 Translator 把 privatePath → standardPath。
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
			Writable:       false,
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

// enqueueGPVPrefixes enqueues GPV commands using paths from the param mapping
// dictionary as-is. 供 sync_pathb.go 的 Path B 流程复用：把 ParamMapping 抽出的
// 去重 path 列表分批做 GPV。
//
// 路径形态由 basePrefix 决定（含 "{i}" 截到对象前缀，其余原样）。CPE 收到对象前缀
// 时自动展开子树，收到叶子时返回该叶子的值。
func (s *SyncService) enqueueGPVPrefixes(ctx context.Context, dev *model.Device, prefixes []string, sourceID string) error {
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
			SourceID:   sourceID,
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

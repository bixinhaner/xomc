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
	"github.com/redis/go-redis/v9"
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
	redisClient          redis.UniversalClient
	paramSyncWriter      ParamSyncWriter
	config               appconfig.AutoSyncConfig
	batchSize            int
	logger               *zap.Logger
}

// SetRedisClient 注入 Redis 客户端供 Path B 同步 reason 标签传递与差异日志使用（T-0123/T-0127）。
// nil 表示禁用 reason 标签（差异日志 reason 字段会降级为 "unknown"，仍正常输出）。
func (s *SyncService) SetRedisClient(client redis.UniversalClient) *SyncService {
	s.redisClient = client
	return s
}

// ParamSyncWriter 消费者驱动接口：HandleSyncResultPathB BatchUpsert 成功后回写
// devices.last_param_sync_at（T-0124 设计 §2.6 统一回写口径，不区分触发源）。
// device.DeviceRepository 自然满足；测试可用最小 mock。
type ParamSyncWriter interface {
	UpdateLastParamSyncAt(ctx context.Context, id uuid.UUID, at time.Time) error
}

// SetParamSyncWriter 注入 ParamSyncWriter（T-0124）。nil 表示禁用回写
// （PeriodicSyncer 会因 last_param_sync_at 永远为 NULL 而每轮都重新入队，
// dev/test 环境可接受；生产建议注入 DeviceRepository）。
func (s *SyncService) SetParamSyncWriter(w ParamSyncWriter) *SyncService {
	s.paramSyncWriter = w
	return s
}

// StartManualSync 用户手动触发 Path B 全量同步的便捷 wrapper（T-0126 设计 §4）。
//
// 等价于 StartPathBSync(WithReason("manual"))，存在的意义：让 device 包能通过
// 消费者驱动的 narrow interface（ParamSyncStarter，1 方法）注入本服务，
// 不需要 device 包 import provision.PathBOption 类型（避免 device → provision 循环依赖）。
func (s *SyncService) StartManualSync(ctx context.Context, dev *model.Device, sourceID string) (bool, error) {
	return s.StartPathBSync(ctx, dev, sourceID, WithReason("manual"))
}

// pathBOptions 收集 StartPathBSync 的可选配置（T-0123 引入）。
type pathBOptions struct {
	reason string // "device_online" / "periodic" / "firmware_changed" / "manual" / ""
}

// PathBOption 是 StartPathBSync 的 functional option。
type PathBOption func(*pathBOptions)

// WithReason 设置 Path B 同步触发原因。HandleSyncResultPathB 完成时据此打差异日志（T-0127）。
// reason 通过 Redis 临时映射 provision:syncreason:{deviceID} 传递，TTL=10min。
func WithReason(reason string) PathBOption {
	return func(o *pathBOptions) {
		o.reason = reason
	}
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
//
// Batch 划分（object_param_classifier 引入）：
//   - scalar 参数（无尾点）：按 s.batchSize 批量打包，效率优先
//   - object 前缀（尾点 "."，CPE 枚举实例）：一个 path 一个 GPV 独立成 batch
//     原因：object 0 实例时 CPE 回 SOAP Fault 9005，会拒绝整个 GPV。批量发会让
//     无关联的兄弟 path 一并失败。单独发只让自己挂掉，其它 batch 不受影响。
//
// 不并发收紧：所有 batch 走同一 taskSvc 入队，taskSvc 下游 dispatch 已按 device-level
// 串行调度 RPC（每设备同时只跑一个会话），不需要在这里再做"object batch 并发限流 5"
// 之类的事 —— 设备端速率天然受限。
//
// 失败语义：object 单 path GPV 收到 9005 时，ACS 端会把对应 task 标记为 Fault；
// 这是预期行为（"该对象当前无实例"，不是错误），不会触发 reconcileDeletedPaths
// 误删（reconcile 只在响应成功且含数据的入口跑）。
func (s *SyncService) enqueueGPVPrefixes(ctx context.Context, dev *model.Device, prefixes []string, sourceID string) error {
	log, _ := s.discoveryRepo.GetByDeviceID(ctx, dev.ID)
	if log != nil {
		_ = s.discoveryRepo.UpdateStatus(ctx, log.ID, DiscoverySyncing, "")
	}

	batches := buildGPVBatches(prefixes, s.batchSize)
	scalarPaths, objectPaths := classifyPrefixes(prefixes)
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
		zap.Int("scalar_paths", len(scalarPaths)),
		zap.Int("object_paths", len(objectPaths)),
		zap.Int("batches", len(batches)),
	)

	return nil
}

// buildGPVBatches 把 prefixes 划分为 GPV 批次：
//   - scalar 参数合并到 s.batchSize 大小的批
//   - object 前缀（尾点 "."）每条独立成批（size=1），避免 SOAP Fault 9005 误伤
//
// 输入顺序：scalars 与 objects 在输入中可交错；本函数稳定保留各类内部相对顺序，
// 输出 [scalar 批... , object 单 path 批...]。空切片返回 nil（与 batchPaths 一致）。
func buildGPVBatches(prefixes []string, batchSize int) [][]string {
	scalars, objects := classifyPrefixes(prefixes)
	if batchSize <= 0 {
		batchSize = 50
	}
	var batches [][]string
	batches = append(batches, batchPaths(scalars, batchSize)...)
	for _, p := range objects {
		batches = append(batches, []string{p})
	}
	return batches
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

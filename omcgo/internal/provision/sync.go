package provision

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/omcgo/omcgo/internal/config/parammodel"
	"github.com/omcgo/omcgo/internal/core/appconfig"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/device"
	"github.com/omcgo/omcgo/internal/mml"
	"github.com/omcgo/omcgo/internal/product"
	"github.com/omcgo/omcgo/internal/task"
	"github.com/omcgo/omcgo/pkg/tr069"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// syncGPVTaskExpiresIn 是 Path B / 手动 sync GPV task 的过期秒数（30 分钟）。
//
// 为什么需要单独的 TTL：全局 default_expires_in_seconds=120 适用于交互式 / 单 RPC
// 场景，但单次 sync 在 BSC 等慢设备上会产生 200+ object 前缀 task（如
// DeviceGSM.Bts.{1..256}.），ACS 在一次 inform session 内只能串行 push（CPE 完
// 成一个空 POST 再触发下一个 RPC），按 ~1 task/s 估算 250 task 需要 4~5 分钟才
// 能消化完，并且要跨越多次 inform 周期（设备 inform_interval 通常 300s）。
//
// 沿用 120s 时后半批 task 会被 ExpiredSweeper 抢先标 expired，导致部分 BTS
// 实例无法落库（前端临区/TRX 表显示不全），与首次同步语义不符。1800s 留出
// 完整 TTL 内的 inform 机会，配合 ACS handler 单 task 1s 处理上限完成批量收尾。
const syncGPVTaskExpiresIn = 1800
const syncGPVTaskRetryIntervalSeconds = 30

// SyncService handles batch parameter value synchronization from devices.
//
// T-0098 P5-01：旧 datamodel 双阶段同步（StartTwoPhaseSync / HandleGPNResult /
// iterator）已全部删除。仅保留 Path B：通过 paramRegistry 获取 MappingSet，按
// is_storable=true 抽取去重对象前缀直接 GPV，CPE 自动展开实例（设计 §1.11）。
type SyncService struct {
	paramRepo            device.DeviceParameterRepository
	discoveryRepo        ParameterDiscoveryLogRepository
	taskSvc              task.Enqueuer
	pathBSyncTaskReader  PathBSyncTaskReader
	planStore            *SyncPlanStore
	paramRegistry        *parammodel.Registry
	productRegistry      *product.Registry
	paramRegistryEnabled bool
	unsupportedPathRepo  mml.ProductUnsupportedPathRepository
	redisClient          redis.UniversalClient
	paramSyncWriter      ParamSyncWriter
	deviceInfoRefresher  DeviceInfoRefresher
	deviceNameSyncHook   *DeviceNameSyncHook // Issue #758: 设备名称同步钩子
	config               appconfig.AutoSyncConfig
	batchSize            int
	logger               *zap.Logger
	durableStarter       DurableParamSyncStarter
}

// DurableParamSyncStarter exposes the durable request/run data plane without
// coupling provision to the paramsync package.
type DurableParamSyncStarter interface {
	StartDurableSync(ctx context.Context, dev *model.Device, sourceID, reason string, parameterPaths []string) (handled bool, taskCount int, err error)
}

func (s *SyncService) SetDurableStarter(starter DurableParamSyncStarter) *SyncService {
	s.durableStarter = starter
	return s
}

type syncGPVOpenGuard interface {
	HasOpenSyncGPVTasksByDevice(ctx context.Context, deviceSN string) (bool, error)
}

type syncGPVDeviceLocker interface {
	AcquireSyncGPVDeviceLock(ctx context.Context, deviceSN string) (release func(), err error)
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

// DeviceInfoRefresher 在 Path B 参数全量落库后，把 device_parameters 投影刷新到
// device_info，避免列表/详情读取到旧快照。
type DeviceInfoRefresher interface {
	SyncFromParameters(ctx context.Context, deviceID uuid.UUID, carrierCode model.CarrierCode, tech model.Technology, productClass string) ([]string, error)
}

// PathBSyncTaskReader 查询某设备是否仍有未完成的 sync-gpv 任务。
// 用真实任务状态而不是预估批次数判断收尾，覆盖 ACS SOAP Fault 自愈产生的 -r 重试任务。
type PathBSyncTaskReader interface {
	HasIncompleteSyncGPVTasksByDevice(ctx context.Context, deviceSN string) (bool, error)
}

// SetParamSyncWriter 注入 ParamSyncWriter（T-0124）。nil 表示禁用回写
// （PeriodicSyncer 会因 last_param_sync_at 永远为 NULL 而每轮都重新入队，
// dev/test 环境可接受；生产建议注入 DeviceRepository）。
func (s *SyncService) SetParamSyncWriter(w ParamSyncWriter) *SyncService {
	s.paramSyncWriter = w
	return s
}

// SetDeviceInfoRefresher 注入 device_info 刷新器，使 Path B 同步完成后立即刷新快照列。
func (s *SyncService) SetDeviceInfoRefresher(r DeviceInfoRefresher) *SyncService {
	s.deviceInfoRefresher = r
	return s
}

func (s *SyncService) SetPathBSyncTaskReader(r PathBSyncTaskReader) *SyncService {
	s.pathBSyncTaskReader = r
	return s
}

func (s *SyncService) SetUnsupportedPathRepo(r mml.ProductUnsupportedPathRepository) *SyncService {
	s.unsupportedPathRepo = r
	return s
}

// SetDeviceNameSyncHook 注入设备名称同步钩子（Issue #758）。
// 在 Path B 同步完成后自动检测 LMT 名称与网管名称是否一致，按配置方向同步。
func (s *SyncService) SetDeviceNameSyncHook(h *DeviceNameSyncHook) *SyncService {
	s.deviceNameSyncHook = h
	return s
}

// StartManualSync 用户手动触发 Path B 同步的便捷 wrapper（T-0126 设计 §4）。
//
// 等价于 StartPathBSync(WithReason("manual"))，存在的意义：让 device 包能通过
// 消费者驱动的 narrow interface（ParamSyncStarter，1 方法）注入本服务，
// 不需要 device 包 import provision.PathBOption 类型（避免 device → provision 循环依赖）。
func (s *SyncService) StartManualSync(ctx context.Context, dev *model.Device, sourceID string, parameterPaths []string) (bool, int, error) {
	return s.StartPathBSync(ctx, dev, sourceID, WithReason("manual"), WithParameterPaths(parameterPaths))
}

// pathBOptions 收集 StartPathBSync 的可选配置（T-0123 引入）。
type pathBOptions struct {
	reason         string // "device_online" / "periodic" / "firmware_changed" / "manual" / ""
	parameterPaths []string
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

// WithParameterPaths scopes a manual Path B sync to the given standard paths.
// Empty means full sync, preserving the historical behavior for other callers.
func WithParameterPaths(paths []string) PathBOption {
	return func(o *pathBOptions) {
		o.parameterPaths = append([]string(nil), paths...)
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
	_, err := s.EnqueueGPVBatches(ctx, dev.SerialNumber, paramPaths, sourceID)
	return err
}

// EnqueueGPVBatches 把 paths 拆批入队 GetParameterValues task。对所有同步入口统一：
//   - StartSync（手动 / 北向显式 path 列表）
//   - enqueueGPVPrefixes（Path B / Inform 触发，从 paramRegistry 抽 storable 前缀）
//   - PullConfig（HTTP /config/sync/pull/{deviceId} 北向同步）
//
// 批次划分（object_param_classifier 引入）：
//   - 标量参数（无尾点）按 s.batchSize 批量打包，效率优先
//   - 已展开的实例级对象前缀（如 DeviceGSM.Bts.1.）按 5MB NATS payload 预算自适应合批
//   - 其它对象前缀（尾点 "."，CPE 枚举实例）每条独立成批 size=1
//
// 普通对象前缀单独成批的原因：CPE 展开对象前缀返回所有当前实例的所有参数，单个对象就可能
// 是几十上百个参数；且对象 0 实例时 CPE 回 9005 拒绝整批。混批等同于"一个对象 fault
// 拖垮 49 个兄弟标量参数"。事后有 ACS handler.tryRecoverGPVFault 自愈兜底（处理对象
// 有实例但其中某参数不支持等场景），事前 size=1 是对最常见 fault 模式的预防。
//
// 已展开实例级对象不同：DeviceGSM.Bts.1. 这种 path 只会返回单实例子树，响应规模稳定，
// 可按保守字节估算合批以降低 BSC 256 BTS 场景的串行往返数，同时把估算 NATS payload
// 控制在 5MB 以下。
//
// commandKey 使用 "sync-gpv-{sn}-{i}"，保留 sync-gpv- 前缀供 ACS Fault
// 自愈和 Path B 结果翻译识别。
//
// ExpiresIn=syncGPVTaskExpiresIn（1800s）：保留给慢设备/异常大对象的多批兜底窗口。
// 常规 BSC BTS 对象在 5MB 预算内会一次整对象同步，不再产生 200+ 个实例 task。
//
// 返回入队成功的 task ID 列表，调用方可用于追溯/北向返回。
func (s *SyncService) EnqueueGPVBatches(ctx context.Context, deviceSN string, paramPaths []string, sourceID string) ([]string, error) {
	return s.enqueueGPVBatches(ctx, deviceSN, paramPaths, sourceID, "sync-gpv-")
}

func (s *SyncService) enqueueGPVBatches(
	ctx context.Context,
	deviceSN string,
	paramPaths []string,
	sourceID string,
	commandKeyPrefix string,
) ([]string, error) {
	if deviceSN == "" {
		return nil, fmt.Errorf("EnqueueGPVBatches: empty deviceSN")
	}
	if strings.TrimSpace(sourceID) == "" {
		sourceID = uuid.NewString()
	}
	batches := buildGPVBatches(paramPaths, s.batchSize)
	taskIDs := make([]string, 0, len(batches))
	maxRetries := task.RetryBudgetCoveringExpiry(syncGPVTaskExpiresIn, syncGPVTaskRetryIntervalSeconds)
	for i, batch := range batches {
		gpvParams, err := json.Marshal(map[string]interface{}{
			"names": batch,
		})
		if err != nil {
			return taskIDs, fmt.Errorf("marshal GPV batch %d: %w", i, err)
		}
		t, err := s.taskSvc.CreateTask(ctx, &task.CreateTaskRequest{
			DeviceSN:             deviceSN,
			Method:               MethodGetParameterValues,
			Params:               gpvParams,
			Priority:             10 + i,
			ExpiresIn:            syncGPVTaskExpiresIn,
			MaxRetries:           &maxRetries,
			RetryIntervalSeconds: syncGPVTaskRetryIntervalSeconds,
			CommandKey:           fmt.Sprintf("%s%s-%d", commandKeyPrefix, deviceSN, i),
			Source:               task.TaskSourceSystem,
			SourceID:             sourceID,
		})
		if err != nil {
			return taskIDs, fmt.Errorf("enqueue GPV batch %d: %w", i, err)
		}
		if t != nil {
			taskIDs = append(taskIDs, t.ID)
		}
	}
	s.logger.Info("GPV batches enqueued",
		zap.String("device_sn", deviceSN),
		zap.Int("total_params", len(paramPaths)),
		zap.Int("batches", len(batches)),
	)
	return taskIDs, nil
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

// enqueueGPVPrefixes 把 paramRegistry 抽出的去重前缀分批入队 GPV。供 Path B 同步使用。
//
// 路径形态由 basePrefix 决定（含 "{i}" 截到对象前缀，其余原样）。CPE 收到对象前缀
// 时自动展开子树，收到叶子时返回该叶子的值。批次拆分逻辑由 EnqueueGPVBatches 统一
// 实现（标量合并 + 对象前缀 size=1），ACS handler.tryRecoverGPVFault 在仍发生 fault 时兜底。
func (s *SyncService) enqueueGPVPrefixes(
	ctx context.Context,
	dev *model.Device,
	prefixes []string,
	sourceID string,
) error {
	log, _ := s.discoveryRepo.GetByDeviceID(ctx, dev.ID)
	if log != nil {
		_ = s.discoveryRepo.UpdateStatus(ctx, log.ID, DiscoverySyncing, "")
	}
	_, err := s.EnqueueGPVBatches(ctx, dev.SerialNumber, prefixes, sourceID)
	return err
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

const (
	// natsMaxPayloadBytes 对齐 NATS max_payload=5MB。GPV 批量估算必须低于该值。
	natsMaxPayloadBytes = 5 * 1024 * 1024

	// gpvNATSPayloadBudgetBytes 只使用 75% 的 NATS 上限，给事件 envelope、JSON 元数据、
	// 参数名长度波动和设备返回值波动留余量。
	gpvNATSPayloadBudgetBytes = natsMaxPayloadBytes * 3 / 4

	// expandedObjectPrefixPayloadEstimateBytes 是单个已展开实例级 object GPV 响应的保守估算。
	// BSC BTS 实测约 70 字段，按 avgFieldBytes=60 约 4KB；这里按 12KB 计，256 个实例
	// 约 3MB，仍低于 5MB 并保留 envelope/JSON 元数据余量。
	expandedObjectPrefixPayloadEstimateBytes = 12 * 1024
)

// buildGPVBatches 把 prefixes 划分为 GPV 批次：
//   - scalar 参数合并到 batchSize 大小的批
//   - 已展开实例级 object 前缀按 NATS payload 预算合批
//   - 其它 object 前缀（尾点 "."）每条独立成批（size=1）—— 见 object_param_classifier.go 的说明
//
// 输出 [scalar 批... , instance object 批... , object 单 path 批...]。空切片返回 nil（与 batchPaths 一致）。
func buildGPVBatches(prefixes []string, batchSize int) [][]string {
	scalars, objects := classifyPrefixes(prefixes)
	if batchSize <= 0 {
		batchSize = 50
	}
	instanceObjectBatchSize := maxExpandedObjectPrefixesPerGPV()
	var batches [][]string
	batches = append(batches, batchPaths(scalars, batchSize)...)
	var instanceObjects []string
	for _, p := range objects {
		if isExpandedInstanceObjectPath(p) {
			instanceObjects = append(instanceObjects, p)
			continue
		}
		batches = append(batches, batchPaths(instanceObjects, instanceObjectBatchSize)...)
		instanceObjects = nil
		batches = append(batches, []string{p})
	}
	batches = append(batches, batchPaths(instanceObjects, instanceObjectBatchSize)...)
	return batches
}

// PathBGPVBatches exposes the established Path B GPV isolation and payload
// budgeting rules to the durable parameter-sync scheduler.
func PathBGPVBatches(prefixes []string, batchSize int) [][]string {
	return buildGPVBatches(prefixes, batchSize)
}

func maxExpandedObjectPrefixesPerGPV() int {
	if expandedObjectPrefixPayloadEstimateBytes <= 0 {
		return 1
	}
	budget := gpvNATSPayloadBudgetBytes
	if budget <= 0 || budget > natsMaxPayloadBytes {
		budget = natsMaxPayloadBytes
	}
	maxPrefixes := budget / expandedObjectPrefixPayloadEstimateBytes
	if maxPrefixes < 1 {
		return 1
	}
	return maxPrefixes
}

func estimatedGPVNATSPayloadBytes(prefixes []string) int {
	return len(prefixes) * expandedObjectPrefixPayloadEstimateBytes
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

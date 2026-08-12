package provision

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/config/template"
	"github.com/omcgo/omcgo/internal/core/appconfig"
	"github.com/omcgo/omcgo/internal/core/carrier"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/reliability"
	"github.com/omcgo/omcgo/internal/core/tracing"
	"github.com/omcgo/omcgo/internal/device"
	"github.com/omcgo/omcgo/internal/product"
	"github.com/omcgo/omcgo/internal/task"
	"github.com/omcgo/omcgo/pkg/tr069"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
)

// ProvisioningEngine orchestrates the full auto-provisioning workflow.
//
// T-0098 P5-01：旧 dmRegistry / GPN 二阶段同步 / handleDataModelFileReceived
// 已删除。Path B 同步走 syncService.StartPathBSync（基于 paramRegistry.GetByProduct
// 的 MappingSet）；Path C 模型上传由 modelUploadService.HandleModelFileReceived
// 解析 XML → IntersectService 写 discovered_param_mappings。
// productClassMatcher 是 ProvisioningEngine 路由 productClass → product 装配件
// 的最小依赖（T-0176-PR-D）。生产由 *product.Registry 满足；测试可注入 stub。
// 消费者侧定义避免对 product 内部类型的强耦合。
type productClassMatcher interface {
	MatchProductClass(ctx context.Context, productClass string) (*product.MatchResult, error)
}

// productBinder 是 ProvisioningEngine 把命中的 product 装配件写回 device 行
// 的最小依赖（T-0176-PR-D）。生产由 *product.PgRepository 满足；测试可注入 stub。
type productBinder interface {
	BindDevice(ctx context.Context, deviceID, productID uuid.UUID, paramModelID *uuid.UUID) error
	GetProductIDByDeviceID(ctx context.Context, deviceID uuid.UUID) (*uuid.UUID, error)
}

// deviceCacheInvalidator 是 ProvisioningEngine 写库后失效 SN 缓存的最小依赖
// （T-0176-PR-D）。生产由 *device.DeviceCache 满足；测试可注入 stub。
type deviceCacheInvalidator interface {
	Delete(ctx context.Context, sn string)
}

// RegisteredDeviceSyncStarter submits the Issue #148 new-device trigger
// directly to the durable parameter-sync data plane.
type RegisteredDeviceSyncStarter interface {
	StartRegisteredDeviceSync(
		ctx context.Context,
		dev *model.Device,
		sourceID string,
	) error
}

// DeviceOnlineFullSyncResult preserves the durable request outcome at the
// lifecycle-event boundary. The provision package deliberately keeps these
// fields transport-neutral to avoid depending on the paramsync package, whose
// planner already reuses provision's path-selection helpers.
type DeviceOnlineFullSyncResult struct {
	RequestID  uuid.UUID
	RunID      *uuid.UUID
	Status     string
	ResultCode string
	TaskCount  int
}

// DeviceOnlineFullSyncSubmitter sends an online recovery directly into the
// durable parameter_sync request/run lifecycle. It must not fall back to the
// legacy sync-gpv Path B task materializer.
type DeviceOnlineFullSyncSubmitter interface {
	SubmitDeviceOnlineFullSync(
		ctx context.Context,
		dev *model.Device,
		idempotencyKey string,
		sourceEventID string,
		originEventType string,
	) (*DeviceOnlineFullSyncResult, error)
}

// policyContinuation advances an explicitly executed plug-and-play policy only
// after the existing file-management state machine reports a terminal result.
type policyContinuation interface {
	ContinuePolicy(context.Context, uuid.UUID, uuid.UUID, policyModule) error
}

// automaticPolicyExecutor starts the plug-and-play policy after the existing
// first-registration full parameter synchronization has finished. Parameter
// synchronization is only a sequencing boundary; it is not a provisioning
// task step and its result is not copied into the plug-and-play task.
type automaticPolicyExecutor interface {
	ExecuteAutomaticPolicy(context.Context, uuid.UUID) error
}

type activationStateReader interface {
	GetByDeviceID(context.Context, uuid.UUID) (*device.DeviceInfo, error)
}

type activationStateRefresher interface {
	SyncFromParameters(context.Context, uuid.UUID, model.CarrierCode, model.Technology, string) ([]string, error)
}

const activationCheckDelay = 5 * time.Minute
const activationCheckOrigin = "provision.activation_check"

// activationCheckMaxAttempts counts the initial observation as attempt one.
// The schedule is therefore T+5m, T+10m and T+15m, with no T+20m check.
const activationCheckMaxAttempts = 3

type ProvisioningEngine struct {
	taskRepo           ProvisioningTaskRepository
	deviceService      *device.DeviceService
	templateService    *template.ConfigTemplateService
	carrierRegistry    *carrier.CarrierRegistry
	taskSvc            task.Enqueuer
	eventBus           event.EventBus
	deduper            *event.Deduper
	modelUploadService *ModelUploadService
	syncService        *SyncService
	registeredSync     RegisteredDeviceSyncStarter
	deviceOnlineSync   DeviceOnlineFullSyncSubmitter
	productRegistry    productClassMatcher
	productRepo        productBinder
	// deviceCache 在 lazy bind 写库成功后失效 SN 缓存。
	// nil 表示禁用（不影响主流程，仅留下 stale 缓存的可能 — admin 改 productClass 时
	// 见 §C C2 修复路径）。T-0176-PR-D 注入。
	deviceCache         deviceCacheInvalidator
	redisClient         redis.UniversalClient
	metrics             *Metrics
	config              appconfig.ProvisionConfig
	logger              *zap.Logger
	policyContinuation  policyContinuation
	automaticPolicy     automaticPolicyExecutor
	activationState     activationStateReader
	activationRefresher activationStateRefresher

	gpvWorkersOnce sync.Once
	gpvWorkerChans []chan gpvWorkItem
}

func (e *ProvisioningEngine) SetPolicyContinuation(continuation policyContinuation) {
	e.policyContinuation = continuation
}

func (e *ProvisioningEngine) SetAutomaticPolicyExecutor(executor automaticPolicyExecutor) {
	e.automaticPolicy = executor
}

func (e *ProvisioningEngine) SetActivationStateReader(reader activationStateReader) {
	e.activationState = reader
}

func (e *ProvisioningEngine) SetActivationStateRefresher(refresher activationStateRefresher) {
	e.activationRefresher = refresher
}

// NewProvisioningEngine creates a new ProvisioningEngine with all dependencies.
func NewProvisioningEngine(
	taskRepo ProvisioningTaskRepository,
	deviceService *device.DeviceService,
	templateService *template.ConfigTemplateService,
	carrierRegistry *carrier.CarrierRegistry,
	taskSvc task.Enqueuer,
	eventBus event.EventBus,
	config appconfig.ProvisionConfig,
	logger *zap.Logger,
) *ProvisioningEngine {
	return &ProvisioningEngine{
		taskRepo:        taskRepo,
		deviceService:   deviceService,
		templateService: templateService,
		carrierRegistry: carrierRegistry,
		taskSvc:         taskSvc,
		eventBus:        eventBus,
		config:          config,
		logger:          logger,
	}
}

// SetModelUploadService sets the model upload service for parameter model acquisition.
func (e *ProvisioningEngine) SetModelUploadService(svc *ModelUploadService) {
	e.modelUploadService = svc
}

// SetSyncService sets the sync service for parameter synchronization.
func (e *ProvisioningEngine) SetSyncService(svc *SyncService) {
	e.syncService = svc
}

func (e *ProvisioningEngine) SetRegisteredDeviceSyncStarter(starter RegisteredDeviceSyncStarter) {
	e.registeredSync = starter
}

func (e *ProvisioningEngine) SetDeviceOnlineFullSyncSubmitter(submitter DeviceOnlineFullSyncSubmitter) {
	e.deviceOnlineSync = submitter
}

// SetRedisClient 注入 Redis 客户端供 device.online 节流与 Path B 同步差异日志使用（T-0123）。
// nil 表示禁用 token bucket 节流（仍可正常处理事件，幂等性由下游 GPV/UPSERT 保证）。
func (e *ProvisioningEngine) SetRedisClient(client redis.UniversalClient) {
	e.redisClient = client
}

// SetMetrics 注入 provisioning 指标集合，供 Redis 节流失败计数（MEDIUM-19）。
// nil 表示禁用打点（节流逻辑不变，仅不暴露失败计数）。
func (e *ProvisioningEngine) SetMetrics(m *Metrics) {
	e.metrics = m
}

// throttleSetNX 在节流路径执行 Redis SetNX，带短重试（MEDIUM-19）。
//
// 语义保持 fail-open：返回 (acquired, throttleAvailable)。
//   - throttleAvailable=false（重试耗尽仍失败）→ 调用方放行（继续同步），但运维可
//     从 provision_redis_throttle_failures_total{operation} 与 error 日志看到"节流
//     已失效，重复同步可能发生"，Redis 故障 >5min 应告警；
//   - acquired=false 且 throttleAvailable=true → 命中已有 token，调用方应跳过（节流）。
//
// 重试用 reliability.Retry（指数退退）抹平 Redis 抖动型瞬时故障——抖动期内重试
// 通常即可拿到 token，避免直接 fail-open 放过重复 device.online 触发 GPV 风暴。
// 整体重试时间窗用 1.5s 上限封顶，不拖慢事件处理热路径。
func (e *ProvisioningEngine) throttleSetNX(
	ctx context.Context, key, operation string, ttl time.Duration,
) (acquired bool, throttleAvailable bool) {
	if e.redisClient == nil {
		return true, false
	}

	retryCtx, cancel := context.WithTimeout(ctx, 1500*time.Millisecond)
	defer cancel()

	cfg := reliability.RetryConfig{MaxAttempts: 3, BaseDelay: 100 * time.Millisecond, MaxDelay: 500 * time.Millisecond}
	err := reliability.Retry(retryCtx, cfg, func(c context.Context) error {
		ok, e2 := e.redisClient.SetNX(c, key, "1", ttl).Result()
		if e2 != nil {
			return e2
		}
		acquired = ok
		return nil
	})
	if err != nil {
		// 重试耗尽：fail-open（节流失效，放行），但打点 + Error 日志让运维可见。
		e.metrics.redisThrottleFailure(operation)
		e.logger.Error("redis throttling disabled, duplicates possible",
			zap.String("operation", operation),
			zap.String("key", key),
			zap.Error(err))
		return true, false
	}
	return acquired, true
}

// SetDeduper enables idempotent event handling for the provisioning engine.
// When set, repeated NATS deliveries of the same event ID skip the handler.
// nil 表示关闭去重（向后兼容单进程内存 EventBus 场景）。
func (e *ProvisioningEngine) SetDeduper(d *event.Deduper) {
	e.deduper = d
}

// SetProductBinder 注入产品装配件路由 + 写回能力（B1 修复）。
// HandleBootstrap identify 阶段调 productRegistry 命中产品后,用 productRepo
// 把 product_id / param_model_id 写回 device 行,避免前端"产品中心"设备数永 0
// 和孤儿设备页误判。两个参数任一为 nil 都关闭此功能。
//
// 接收消费者侧 narrow interface（productClassMatcher / productBinder），生产由
// *product.Registry / *product.PgRepository 满足，测试可注入 stub
// （T-0176-PR-D 重构）。
func (e *ProvisioningEngine) SetProductBinder(reg productClassMatcher, repo productBinder) {
	e.productRegistry = reg
	e.productRepo = repo
}

// SetDeviceCache 注入 DeviceCache，让 bindDeviceProductIfNeeded 在写库后失效
// SN 缓存（T-0176-PR-D）。nil 表示禁用 — bind 仍写库，仅跳过 cache.Delete。
func (e *ProvisioningEngine) SetDeviceCache(c deviceCacheInvalidator) {
	e.deviceCache = c
}

// bindDeviceProduct 调 productRegistry 路由 productClass 命中产品后，
// 把 product_id / param_model_id 写回 device 行。返 bound=true 当且仅当
// 实际 BindDevice 写库成功；orphan / 任何 err / 前置校验失败均返 false。
// 任何错误都仅记 warn 不阻断后续 provisioning（孤儿设备由前端孤儿设备页处理）。
//
// T-0176-PR-D：返 bool 让 bindDeviceProductIfNeeded 据此决定是否清 cache。
func (e *ProvisioningEngine) bindDeviceProduct(ctx context.Context, dev *model.Device) (bound bool) {
	if e.productRegistry == nil || e.productRepo == nil || dev == nil || dev.ProductClass == "" {
		return false
	}
	match, err := e.productRegistry.MatchProductClass(ctx, dev.ProductClass)
	if err != nil {
		// ErrOrphan 不算异常 — 这里 warn 即可，调用方据 false 跳过 cache 清理
		e.logger.Warn("bindDeviceProduct: match productClass failed",
			zap.String("device_sn", dev.SerialNumber),
			zap.String("product_class", dev.ProductClass),
			zap.Error(err))
		return false
	}
	if match == nil || match.Product == nil {
		return false
	}
	if err := e.productRepo.BindDevice(ctx, dev.ID, match.Product.ID, match.Product.ParamModelID); err != nil {
		e.logger.Warn("bindDeviceProduct: write product_id failed",
			zap.String("device_sn", dev.SerialNumber),
			zap.String("product_id", match.Product.ID.String()),
			zap.Error(err))
		return false
	}
	e.logger.Info("device bound to product",
		zap.String("device_sn", dev.SerialNumber),
		zap.String("product_id", match.Product.ID.String()),
		zap.String("product_name", match.Product.Name))
	return true
}

// bindDeviceProductIfNeeded 在非 Bootstrap 入口（DeviceOnline / FirmwareChanged /
// FileType11 上传）头部懒补 product 绑定。已绑定时 skip；写库成功后失效 SN cache。
//
// "已绑定"的检测：通过 productRepo.GetProductIDByDeviceID 查 devices.product_id。
// productRepo / productRegistry 任一 nil 时静默跳过（与 bindDeviceProduct 行为一致）。
//
// 任何 err 都不抛出 — 主流程（Path B 同步 / Upload / RPC 派发）不应被审计写库
// 路径阻塞（PR-D 事实修正：product_id 列写入只服务 admin/审计/未来 denorm 消费）。
func (e *ProvisioningEngine) bindDeviceProductIfNeeded(ctx context.Context, dev *model.Device) {
	if dev == nil || e.productRepo == nil || e.productRegistry == nil {
		return
	}
	if dev.ProductClass == "" {
		return
	}
	// 已绑定检测：查 devices.product_id；err / not-null 都 skip
	existing, err := e.productRepo.GetProductIDByDeviceID(ctx, dev.ID)
	if err != nil {
		// 查询失败不阻塞主流程，silent skip（BindDevice 即使重复写也幂等，
		// 但避免主流程额外 DB round trip — 这里宁可让缓存暂时落后一拍）
		e.logger.Debug("bindDeviceProductIfNeeded: lookup product_id failed, skipping",
			zap.String("device_sn", dev.SerialNumber),
			zap.Error(err))
		return
	}
	if existing != nil {
		// 已绑定 → skip
		return
	}
	if !e.bindDeviceProduct(ctx, dev) {
		return
	}
	if e.deviceCache != nil {
		// cache.Delete 内部已有 warn log + silent fail，不再重复包装
		e.deviceCache.Delete(ctx, dev.SerialNumber)
	}
}

// bootstrapEvent represents the data published on device.registered.
type bootstrapEvent struct {
	DeviceID     uuid.UUID `json:"device_id"`
	SerialNumber string    `json:"serial_number"`
	OUI          string    `json:"oui"`
	ProductClass string    `json:"product_class"`
	Carrier      string    `json:"carrier"`
	Technology   string    `json:"technology"`
	Created      bool      `json:"created"`
}

// Subscribe registers the engine to listen for bootstrap and model file events.
func (e *ProvisioningEngine) Subscribe(bus event.EventBus) error {
	bootstrapHandler := func(ctx context.Context, evt event.Event) error {
		e.logger.Info("provisioning engine received device.registered event",
			zap.String("event_id", evt.ID),
			zap.String("subject", evt.Subject),
		)
		var bsEvt bootstrapEvent
		if err := evt.DecodePayload(&bsEvt); err != nil {
			e.logger.Error("decode bootstrap event", zap.Error(err))
			return err
		}
		return e.HandleBootstrap(ctx, bsEvt)
	}
	if e.deduper != nil {
		bootstrapHandler = e.deduper.Wrap("provision-bootstrap", bootstrapHandler)
	}
	_, err := bus.QueueSubscribe(event.SubjectDeviceRegistered, "provisioning", bootstrapHandler)
	if err != nil {
		e.logger.Error("failed to subscribe provisioning engine", zap.Error(err))
		return err
	}
	e.logger.Info("provisioning engine subscribed to device.registered events")

	if _, err := bus.QueueSubscribe(
		event.SubjectDeviceRegistered,
		"device-registered-param-sync",
		e.handleRegisteredDeviceSyncEvent,
	); err != nil {
		return fmt.Errorf("subscribe registered-device parameter sync: %w", err)
	}

	if _, err := bus.QueueSubscribe(
		event.SubjectDeviceTransferComplete,
		"provision-xml-transfer-complete",
		func(ctx context.Context, evt event.Event) error {
			return e.handleXMLTransferCompleteEvent(ctx, evt)
		},
	); err != nil {
		return fmt.Errorf("subscribe plug and play TransferComplete: %w", err)
	}
	if _, err := bus.QueueSubscribe(
		event.SubjectParamSyncRunCompleted,
		"provision-auto-policy-after-registered-sync",
		e.handleRegisteredParamSyncCompleted,
	); err != nil {
		return fmt.Errorf("subscribe registered-device parameter sync completion: %w", err)
	}
	if _, err := bus.QueueSubscribe(
		event.SubjectParamSyncRunFailed,
		"provision-pnp-activation-sync-failed",
		e.handleActivationSyncFailedEvent,
	); err != nil {
		return fmt.Errorf("subscribe plug and play activation sync failure: %w", err)
	}
	if _, err := bus.QueueSubscribe(
		event.SubjectDeviceStartupStageReport,
		"provision-xml-startup-stage",
		func(ctx context.Context, evt event.Event) error {
			var payload device.InformEventPayload
			if err := evt.DecodePayload(&payload); err != nil {
				return fmt.Errorf("decode plug and play startup stage Inform: %w", err)
			}
			return e.HandleStartupStageReport(ctx, payload)
		},
	); err != nil {
		return fmt.Errorf("subscribe plug and play startup stage: %w", err)
	}
	if _, err := bus.QueueSubscribe(
		event.SubjectDeviceStartupResultReport,
		"provision-xml-startup-result",
		func(ctx context.Context, evt event.Event) error {
			var payload device.InformEventPayload
			if err := evt.DecodePayload(&payload); err != nil {
				return fmt.Errorf("decode plug and play startup result Inform: %w", err)
			}
			return e.HandleStartupResultReport(ctx, payload)
		},
	); err != nil {
		return fmt.Errorf("subscribe plug and play startup result: %w", err)
	}
	for _, subscription := range []struct {
		subject string
		queue   string
		failed  bool
	}{
		{event.SubjectUpgradeCompleted, "provision-delegated-upgrade-completed", false},
		{event.SubjectUpgradeFailed, "provision-delegated-upgrade-failed", true},
	} {
		item := subscription
		if _, err := bus.QueueSubscribe(item.subject, item.queue, func(ctx context.Context, evt event.Event) error {
			var payload delegatedUpgradeEvent
			if err := evt.DecodePayload(&payload); err != nil {
				return fmt.Errorf("decode delegated upgrade result: %w", err)
			}
			return e.HandleDelegatedUpgradeResult(ctx, payload, item.failed)
		}); err != nil {
			return fmt.Errorf("subscribe %s: %w", item.subject, err)
		}
	}

	// Subscribe to datamodel.file.received for Path C model upload processing.
	if _, err := bus.QueueSubscribe(event.SubjectDataModelFileReceived, "provision-model-upload", func(ctx context.Context, evt event.Event) error {
		return e.handleDataModelFileReceived(ctx, evt)
	}); err != nil {
		e.logger.Warn("failed to subscribe to datamodel.file.received", zap.Error(err))
	} else {
		e.logger.Info("provisioning engine subscribed to datamodel.file.received events")
	}

	// Subscribe to GPV response for parameter sync processing.
	gpvConfig := e.config.GPVResponse.Defaults()
	if tuner, ok := bus.(interface {
		SetPullTuning(string, event.PullTuning)
	}); ok {
		tuner.SetPullTuning(event.SubjectCommandGetParamsResponse, event.PullTuning{
			BatchSize:     gpvPullBatchSize,
			Concurrency:   gpvConfig.ProvisionConcurrency,
			AckWait:       gpvConfig.AckWait,
			MaxDeliver:    gpvConfig.MaxDeliver,
			MaxAckPending: gpvConfig.MaxAckPending,
		})
	}
	e.ensureGPVWorkersStarted()
	gpvHandler := func(ctx context.Context, evt event.Event) error {
		return e.enqueueGPVResponseEvent(ctx, evt)
	}
	var gpvSub event.Subscription
	if keyedBus, ok := bus.(interface {
		KeyedPullSubscribe(
			subject, queue string,
			queueDepth int,
			keyFunc event.EventKeyFunc,
			handler event.EventHandler,
		) (event.Subscription, error)
	}); ok {
		gpvSub, err = keyedBus.KeyedPullSubscribe(
			event.SubjectCommandGetParamsResponse,
			gpvConfig.ProvisionQueue,
			gpvConfig.ProvisionQueueDepth,
			provisionGPVDeviceKey,
			gpvHandler,
		)
	} else {
		// Non-NATS test buses do not expose ordered keyed pull. Keep their
		// delivery single-threaded so adjacent responses for one device cannot
		// enter the engine shards out of order.
		if tuner, ok := bus.(interface {
			SetPullTuning(string, event.PullTuning)
		}); ok {
			tuner.SetPullTuning(event.SubjectCommandGetParamsResponse, event.PullTuning{
				BatchSize:     gpvPullBatchSize,
				Concurrency:   1,
				AckWait:       gpvConfig.AckWait,
				MaxDeliver:    gpvConfig.MaxDeliver,
				MaxAckPending: 1,
			})
		}
		gpvSub, err = bus.PullSubscribe(
			event.SubjectCommandGetParamsResponse,
			gpvConfig.ProvisionQueue,
			gpvHandler,
		)
	}
	if err != nil {
		e.logger.Warn("failed to subscribe to GPV response", zap.Error(err))
	} else {
		_ = gpvSub
		e.logger.Info("provisioning engine subscribed to GPV response events")
	}

	// T-0098 P5-01：GPN response 订阅删除（two-phase sync 已移除）。

	// 订阅 device.online — 已存在设备从 offline 恢复时触发 durable 全量参数同步。
	onlineHandler := func(ctx context.Context, evt event.Event) error {
		var onlineEvt device.DeviceOnlineEvent
		if err := evt.DecodePayload(&onlineEvt); err != nil {
			e.logger.Error("decode device.online event", zap.Error(err),
				zap.String("event_id", evt.ID))
			return err
		}
		return e.handleDeviceOnline(ctx, onlineEvt, evt.ID)
	}
	if _, err := bus.QueueSubscribe(event.SubjectDeviceOnline, "provision-online-sync", onlineHandler); err != nil {
		e.logger.Warn("failed to subscribe to device.online", zap.Error(err))
	} else {
		e.logger.Info("provisioning engine subscribed to device.online events")
	}

	// 订阅 device.firmware.changed — 固件升级后刷新模型；若同时上线则补交 durable 全量同步。
	firmwareHandler := func(ctx context.Context, evt event.Event) error {
		var fwEvt device.DeviceFirmwareChangedEvent
		if err := evt.DecodePayload(&fwEvt); err != nil {
			e.logger.Error("decode device.firmware.changed event", zap.Error(err),
				zap.String("event_id", evt.ID))
			return err
		}
		return e.handleFirmwareChanged(ctx, fwEvt, evt.ID)
	}
	if _, err := bus.QueueSubscribe(event.SubjectDeviceFirmwareChanged, "provision-firmware-changed", firmwareHandler); err != nil {
		e.logger.Warn("failed to subscribe to device.firmware.changed", zap.Error(err))
	} else {
		e.logger.Info("provisioning engine subscribed to device.firmware.changed events")
	}

	return nil
}

type paramSyncCompletedEvent struct {
	DeviceID      uuid.UUID `json:"device_id"`
	TriggerReason string    `json:"trigger_reason"`
	SyncScope     string    `json:"sync_scope"`
}

func (e *ProvisioningEngine) handleRegisteredParamSyncCompleted(ctx context.Context, evt event.Event) error {
	var completed paramSyncCompletedEvent
	if err := evt.DecodePayload(&completed); err != nil {
		return fmt.Errorf("decode registered-device parameter sync completion: %w", err)
	}
	if completed.DeviceID == uuid.Nil || completed.SyncScope != "full" {
		return nil
	}
	if completed.TriggerReason == "device_online" {
		return e.handleActivationSyncCompleted(ctx, completed.DeviceID)
	}
	if completed.TriggerReason != "device_registered" {
		return nil
	}
	if e.automaticPolicy == nil {
		e.logger.Warn("automatic plug-and-play executor is unavailable after registered-device sync",
			zap.String("device_id", completed.DeviceID.String()))
		return nil
	}
	return e.automaticPolicy.ExecuteAutomaticPolicy(ctx, completed.DeviceID)
}

// handleXMLTransferCompleteEvent only accepts the CommandKey assigned to the
// plug-and-play Download RPC. Generic or fixed CommandKeys cannot be safely
// correlated: the same device may concurrently report unrelated upload,
// download, or historical TransferComplete events.
func (e *ProvisioningEngine) handleXMLTransferCompleteEvent(ctx context.Context, evt event.Event) error {
	var transfer tr069.TransferComplete
	if err := evt.DecodePayload(&transfer); err != nil {
		return fmt.Errorf("decode plug and play TransferComplete: %w", err)
	}
	if !strings.HasPrefix(transfer.CommandKey, "PNPXML_") {
		return nil
	}
	return e.HandleXMLTransferComplete(ctx, transfer)
}

// HandleDeviceOnline 处理已存在设备从 offline 恢复 active 的事件。
//
// 节流：Redis token bucket key=provision:online_sync:{deviceID} TTL=60s，
// 60s 内同一设备的重复 device.online 事件直接跳过（防 ACS 抖动/multiple Inform 触发）。
// 仅 durable 模式提交 scope=full、reason=device_online 的可靠参数同步；其他模式
// fail-closed，不再回退到 legacy sync-gpv Path B。
func (e *ProvisioningEngine) HandleDeviceOnline(ctx context.Context, evt device.DeviceOnlineEvent) error {
	return e.handleDeviceOnline(ctx, evt, "")
}

func (e *ProvisioningEngine) handleDeviceOnline(
	ctx context.Context,
	evt device.DeviceOnlineEvent,
	sourceEventID string,
) error {
	ctx, span := tracing.StartSpan(ctx, tracing.ProvisionTracerName, "Provision HandleDeviceOnline",
		attribute.String("provision.device_sn", evt.SerialNumber),
		attribute.String("provision.device_id", evt.DeviceID.String()),
	)
	defer span.End()

	// Compatibility for CONFIG_RESTORE tasks created before Auto Start File
	// dispatch was introduced. Do this before the generic online-sync throttle so
	// an in-flight legacy PnP task is never skipped by the duplicate-online guard.
	if pt, lookupErr := e.taskRepo.GetByDeviceID(ctx, evt.DeviceID); lookupErr == nil && pt != nil && !IsTerminal(pt.Status) {
		switch pt.CurrentStepName {
		case "wait_device_online":
			pt.Status = StateVerifying
			pt.CurrentStep = 11
			pt.CurrentStepName = "wait_activation_check"
			pt.RetryCount = 0
			pt.MaxRetries = activationCheckMaxAttempts
			if updateErr := e.taskRepo.Update(ctx, pt); updateErr != nil {
				return fmt.Errorf("schedule plug and play activation check: %w", updateErr)
			}
			// The reboot-complete online event is the start of the required
			// five-minute quiet period. Do not launch the generic immediate
			// device-online sync; the activation checker submits a fresh query
			// only after that period has elapsed.
			return nil
		case "wait_activation_check", "wait_activation_sync":
			// Duplicate online events during the observation/query window must
			// not reset the timer or start an early parameter query.
			return nil
		}
	}

	if e.deviceOnlineSync == nil {
		return fmt.Errorf("device.online durable parameter sync submitter unavailable")
	}

	// Token bucket 节流：60s 内重复 device.online 跳过。
	// MEDIUM-19：SetNX 带短重试抹平 Redis 抖动；重试耗尽仍 fail-open（放行+打点）。
	var throttleKey string
	if e.redisClient != nil {
		throttleKey = fmt.Sprintf("provision:online_sync:%s", evt.DeviceID.String())
		acquired, _ := e.throttleSetNX(ctx, throttleKey, "device_online", 60*time.Second)
		if !acquired {
			e.logger.Debug("device.online throttled by token bucket",
				zap.String("device_id", evt.DeviceID.String()),
				zap.String("serial_number", evt.SerialNumber))
			return nil
		}
	}

	dev, err := e.deviceService.GetDevice(ctx, evt.DeviceID)
	if err != nil {
		if e.redisClient != nil && throttleKey != "" {
			_ = e.redisClient.Del(ctx, throttleKey).Err()
		}
		return fmt.Errorf("device.online: lookup device %s: %w", evt.DeviceID, err)
	}
	if dev == nil {
		e.logger.Warn("device.online: device not found, skipping",
			zap.String("device_id", evt.DeviceID.String()))
		return nil
	}

	// T-0176-PR-D 懒补 product 绑定（首次 Bootstrap 错过 / 历史孤儿设备恢复在线场景）。
	// silent skip 任何 err，不阻塞 durable 参数同步主流程。
	e.bindDeviceProductIfNeeded(ctx, dev)

	if err := e.startDeviceOnlineFullSync(
		ctx,
		dev,
		sourceEventID,
		event.SubjectDeviceOnline,
	); err != nil {
		// Submit 未落到 durable request 前不能保留短时防抖键，否则 NATS
		// 重投会被当成重复上线吞掉。
		if e.redisClient != nil && throttleKey != "" {
			_ = e.redisClient.Del(ctx, throttleKey).Err()
		}
		return err
	}
	return nil
}

func (e *ProvisioningEngine) startDeviceOnlineFullSync(
	ctx context.Context,
	dev *model.Device,
	sourceEventID string,
	originEventType string,
) error {
	if dev == nil {
		return nil
	}
	if e.deviceOnlineSync == nil {
		return fmt.Errorf("device.online durable parameter sync unavailable for device %s", dev.ID)
	}
	if strings.TrimSpace(sourceEventID) == "" {
		sourceEventID = uuid.NewString()
	}
	idempotencyKey := "device_online:" + sourceEventID
	if originEventType == event.SubjectDeviceFirmwareChanged {
		idempotencyKey = "device_online:firmware_changed:" + sourceEventID
	}
	result, err := e.deviceOnlineSync.SubmitDeviceOnlineFullSync(
		ctx,
		dev,
		idempotencyKey,
		sourceEventID,
		originEventType,
	)
	if err != nil {
		return fmt.Errorf("submit device.online durable full sync for device %s: %w", dev.ID, err)
	}
	if result == nil {
		return fmt.Errorf("submit device.online durable full sync for device %s: empty result", dev.ID)
	}

	fields := []zap.Field{
		zap.String("device_id", dev.ID.String()),
		zap.String("serial_number", dev.SerialNumber),
		zap.String("idempotency_key", idempotencyKey),
		zap.String("source_event_id", sourceEventID),
		zap.String("origin_event_type", originEventType),
		zap.String("request_id", result.RequestID.String()),
		zap.String("status", result.Status),
		zap.String("result_code", result.ResultCode),
		zap.Int("task_count", result.TaskCount),
	}
	if result.RunID != nil {
		fields = append(fields, zap.String("run_id", result.RunID.String()))
	}

	switch result.Status {
	case "rejected":
		switch result.ResultCode {
		case "ACTIVE_SYNC_EXISTS":
			e.logger.Info("device.online: durable full sync skipped", fields...)
			return nil
		default:
			return fmt.Errorf(
				"device.online durable full sync rejected for device %s: %s",
				dev.ID,
				result.ResultCode,
			)
		}
	case "failed", "timed_out", "cancelled":
		return fmt.Errorf(
			"device.online durable full sync terminal failure for device %s: status=%s code=%s",
			dev.ID,
			result.Status,
			result.ResultCode,
		)
	case "deduplicated":
		e.logger.Info("device.online: durable full sync deduplicated", fields...)
	default:
		e.logger.Info("device.online: durable full sync accepted", fields...)
	}
	return nil
}

// HandleFirmwareChanged 处理设备固件版本变化事件（T-0125）。
//
// 流程（设计方案 §3.2）：
//  1. 若同一 Inform 同时完成 offline→active，提交一次 reason=device_online 的 durable
//     全量同步，补足 device.online 被二选一抑制的上线语义。
//  2. Redis 串行锁 SetNX provision:firmware_handling:{deviceID} TTL=10min，仅防设备升级期间
//     不稳定 swVersion 多次 Inform 引发重复模型上传。锁不主动释放，TTL 自然过期。
//  3. 调 modelUploadService.RequestModelUploadWithParamSync(syncOnTerminal=false)
//     刷新模型/设备元数据。
func (e *ProvisioningEngine) HandleFirmwareChanged(ctx context.Context, evt device.DeviceFirmwareChangedEvent) error {
	return e.handleFirmwareChanged(ctx, evt, "")
}

func (e *ProvisioningEngine) handleFirmwareChanged(
	ctx context.Context,
	evt device.DeviceFirmwareChangedEvent,
	sourceEventID string,
) error {
	ctx, span := tracing.StartSpan(ctx, tracing.ProvisionTracerName, "Provision HandleFirmwareChanged",
		attribute.String("provision.device_sn", evt.SerialNumber),
		attribute.String("provision.device_id", evt.DeviceID.String()),
		attribute.String("provision.old_version", evt.OldVersion),
		attribute.String("provision.new_version", evt.NewVersion),
	)
	defer span.End()

	// 1. 查 device。查询失败必须交给消息系统重投，不能确认并吞掉该事件。
	dev, err := e.deviceService.GetDevice(ctx, evt.DeviceID)
	if err != nil {
		return fmt.Errorf("firmware.changed: lookup device %s: %w", evt.DeviceID, err)
	}
	if dev == nil {
		e.logger.Warn("firmware.changed: device not found, skipping",
			zap.String("device_id", evt.DeviceID.String()))
		return nil
	}

	// T-0176-PR-D 懒补 product 绑定（设备升级后 productClass 不变但历史 orphan 此刻有机会路由）。
	e.bindDeviceProductIfNeeded(ctx, dev)

	// 2. FirmwareChanged 与 device.online 二选一发布；升级 Inform 同时完成
	// offline→active 时，仍需提交一次 durable 全量同步。
	//
	// 这一步必须位于 model-upload 的 10 分钟锁之外：较早的固件变化可能已经
	// 获取模型上传锁，但随后真正的 BecameOnline 仍必须触发恢复同步。
	if evt.BecameOnline {
		if err := e.startDeviceOnlineFullSync(
			ctx,
			dev,
			sourceEventID,
			event.SubjectDeviceFirmwareChanged,
		); err != nil {
			return fmt.Errorf("firmware.changed online full sync: %w", err)
		}
	}

	// 3. Redis 串行锁只保护模型上传，不能抑制上面的在线恢复同步。
	// MEDIUM-19：SetNX 带短重试抹平 Redis 抖动；重试耗尽仍 fail-open（放行+打点）。
	if e.redisClient != nil {
		firmwareLockKey := fmt.Sprintf("provision:firmware_handling:%s", evt.DeviceID.String())
		acquired, _ := e.throttleSetNX(ctx, firmwareLockKey, "firmware_changed", 10*time.Minute)
		if !acquired {
			e.logger.Debug("firmware model upload already in progress, skip",
				zap.String("device_id", evt.DeviceID.String()),
				zap.String("serial_number", evt.SerialNumber))
			return nil
		}
	}

	sourceID := strings.TrimSpace(sourceEventID)
	if sourceID == "" {
		sourceID = uuid.NewString()
	}
	if e.modelUploadService != nil {
		log, uploadErr := e.modelUploadService.RequestModelUploadWithParamSync(ctx, dev, sourceID, false)
		if uploadErr != nil {
			e.logger.Warn("firmware.changed: RequestModelUpload failed",
				zap.String("device_id", evt.DeviceID.String()),
				zap.Error(uploadErr))
		} else if log != nil && log.Status == DiscoveryDiscovering {
			e.logger.Info("firmware.changed: model upload enqueued",
				zap.String("device_id", evt.DeviceID.String()),
				zap.String("discovery_id", log.ID.String()),
				zap.String("old_version", evt.OldVersion),
				zap.String("new_version", evt.NewVersion))
		} else if log != nil {
			e.logger.Info("firmware.changed: model upload skipped",
				zap.String("device_id", evt.DeviceID.String()),
				zap.String("discovery_status", string(log.Status)),
				zap.String("discovery_message", log.ErrorMessage))
		}
	}

	return nil
}

// HandleBootstrap processes a device bootstrap event and initiates the provisioning workflow.
//
// 路由策略（P5-01 后）：
//
//	Path A：模板匹配命中 + AutoConfigure → 经典模板下发流程
//	Path B：AutoSync 启用 + Path B 可用（productClass 命中 product + paramMapping 存在） → 直接 GPV 同步
//	Path C：ModelUpload 启用 + 设备未上传过 → 下发 Upload(FileType=11)
//	否则失败
func (e *ProvisioningEngine) HandleBootstrap(ctx context.Context, evt bootstrapEvent) error {
	ctx, span := tracing.StartSpan(ctx, tracing.ProvisionTracerName, "Provision HandleBootstrap",
		attribute.String("provision.device_sn", evt.SerialNumber),
		attribute.String("provision.device_id", evt.DeviceID.String()),
		attribute.String("provision.carrier", evt.Carrier),
	)
	defer span.End()

	e.logger.Info("provisioning started for device",
		zap.String("device_sn", evt.SerialNumber),
		zap.String("device_id", evt.DeviceID.String()),
	)

	// 0. Reset-on-BOOT: if there is an existing non-terminal task, fail it and start fresh.
	existingTask, _ := e.taskRepo.GetByDeviceID(ctx, evt.DeviceID)
	if existingTask != nil && !IsTerminal(existingTask.Status) {
		e.logger.Warn("cancelling stale provisioning task for reconnected device",
			zap.String("device_sn", evt.SerialNumber),
			zap.String("existing_task_id", existingTask.ID.String()),
			zap.String("existing_status", string(existingTask.Status)),
			zap.Duration("task_age", time.Since(existingTask.CreatedAt)),
		)
		_ = e.failTask(ctx, existingTask,
			fmt.Errorf("device reconnected with BOOT, cancelling stale task in state %s", existingTask.Status))
		if e.modelUploadService != nil {
			e.modelUploadService.CleanupState(ctx, evt.SerialNumber)
		}
	}

	// 1. Create provisioning task.
	task := NewProvisioningTask(evt.DeviceID)
	now := time.Now()
	task.StartedAt = &now
	if err := e.taskRepo.Create(ctx, task); err != nil {
		return fmt.Errorf("create provisioning task: %w", err)
	}

	e.publishEvent(ctx, event.SubjectProvisionStarted, task)

	// 2. Identify device.
	if err := e.transitionTask(ctx, task, StateIdentifying); err != nil {
		return e.failTask(ctx, task, fmt.Errorf("transition to identifying: %w", err))
	}

	dev, err := e.deviceService.GetDevice(ctx, evt.DeviceID)
	if err != nil {
		return e.failTask(ctx, task, fmt.Errorf("get device: %w", err))
	}

	// 2.5 路由产品装配件并回写 device.product_id / param_model_id（B1 修复）。
	// 路由失败/未命中（孤儿设备）不阻断后续 provisioning，仅记 warn。
	// T-0176-PR-D：HandleBootstrap 进的是首次发现链路，无需 cache.Delete（设备行刚 INSERT，
	// cache 还没写入；后续读再触发 GetOrLoad 即可）。返回值丢弃。
	_ = e.bindDeviceProduct(ctx, dev)

	// 3. Try matching template first (Path A).
	if err := e.transitionTask(ctx, task, StateMatching); err != nil {
		return e.failTask(ctx, task, fmt.Errorf("transition to matching: %w", err))
	}

	tmpl, err := e.templateService.Match(ctx, dev)
	if err != nil {
		return e.failTask(ctx, task, fmt.Errorf("match template: %w", err))
	}

	// T-0120-b：模板自带 auto_dispatch=true 时 opt-in 自动走 Path A，
	// 不依赖全局 config.auto_configure 开关；既有 auto_configure 语义保留。
	if tmpl != nil && (e.config.AutoConfigure || tmpl.AutoDispatch) {
		return e.handleTemplateProvisioning(ctx, task, dev, tmpl, evt.SerialNumber)
	}
	if tmpl != nil {
		e.logger.Info("template matched but auto_configure/auto_dispatch disabled, skipping Path A",
			zap.String("device_sn", evt.SerialNumber),
			zap.String("template_id", tmpl.ID.String()),
		)
	}

	// Newly created devices are synchronized by the independent
	// device.registered durable consumer. Re-entering here would submit the same
	// full sync with a different idempotency key.
	if !evt.Created && e.config.AutoSync.Enabled && e.syncService != nil && e.syncService.PathBEnabled(ctx, dev) {
		return e.handleAutoSync(ctx, task, dev)
	}

	// Path C: ModelUpload enabled → request CPE parameter model upload.
	if e.config.ModelUpload.Enabled && e.modelUploadService != nil {
		return e.handleModelUpload(ctx, task, dev)
	}

	// Path D: nothing applicable → fail.
	e.logger.Warn("no provisioning path applicable, skipping",
		zap.String("device_sn", evt.SerialNumber),
	)
	return e.failTask(ctx, task, fmt.Errorf("no provisioning path for device %s", evt.SerialNumber))
}

func (e *ProvisioningEngine) handleRegisteredDeviceSyncEvent(ctx context.Context, evt event.Event) error {
	var registered bootstrapEvent
	if err := evt.DecodePayload(&registered); err != nil {
		return fmt.Errorf("decode registered-device sync event: %w", err)
	}
	if !registered.Created || e.registeredSync == nil {
		return nil
	}
	dev, err := e.deviceService.GetDevice(ctx, registered.DeviceID)
	if err != nil {
		return fmt.Errorf("get registered device for parameter sync: %w", err)
	}
	if dev == nil {
		return nil
	}
	return e.startRegisteredDeviceSync(ctx, registered, dev)
}

func (e *ProvisioningEngine) startRegisteredDeviceSync(
	ctx context.Context,
	evt bootstrapEvent,
	dev *model.Device,
) error {
	if !evt.Created || e.registeredSync == nil {
		return nil
	}
	err := e.registeredSync.StartRegisteredDeviceSync(
		ctx,
		dev,
		"device_registered:"+dev.ID.String(),
	)
	if err != nil {
		return fmt.Errorf("start registered-device parameter sync: %w", err)
	}
	return nil
}

// handleTemplateProvisioning executes the classic provisioning flow (Path A).
func (e *ProvisioningEngine) handleTemplateProvisioning(ctx context.Context, task *ProvisioningTask,
	dev *model.Device, tmpl *template.ConfigTemplate, deviceSN string) error {

	task.TemplateID = &tmpl.ID
	if err := e.taskRepo.Update(ctx, task); err != nil {
		return e.failTask(ctx, task, fmt.Errorf("update task template: %w", err))
	}

	if err := e.transitionTask(ctx, task, StateConfiguring); err != nil {
		return e.failTask(ctx, task, fmt.Errorf("transition to configuring: %w", err))
	}

	// T-0098 P2-05：Path A 模板 Parameters 视为 standardPath，
	// 翻译为 privatePath 后再下发；翻译未解出时退化为原 standardPath。
	var (
		steps    []ProvisioningStep
		buildErr error
	)
	if e.syncService != nil {
		if translator, ok := e.syncService.ResolveTranslator(ctx, dev); ok {
			steps, buildErr = BuildProvisioningStepsTranslated(tmpl, translator)
		} else {
			steps, buildErr = BuildProvisioningSteps(tmpl)
		}
	} else {
		steps, buildErr = BuildProvisioningSteps(tmpl)
	}
	if buildErr != nil {
		return e.failTask(ctx, task, fmt.Errorf("build provisioning steps: %w", buildErr))
	}
	task.TotalSteps = len(steps)
	if err := e.taskRepo.Update(ctx, task); err != nil {
		return e.failTask(ctx, task, fmt.Errorf("update task steps: %w", err))
	}

	if err := EnqueueSteps(ctx, deviceSN, steps, e.taskSvc, task.ID.String()); err != nil {
		return e.failTask(ctx, task, fmt.Errorf("enqueue steps: %w", err))
	}

	e.logger.Info("provisioning steps enqueued",
		zap.String("device_sn", deviceSN),
		zap.Int("steps", len(steps)),
	)

	if err := e.transitionTask(ctx, task, StateVerifying); err != nil {
		return e.failTask(ctx, task, fmt.Errorf("transition to verifying: %w", err))
	}

	return nil
}

// handleModelUpload requests the CPE to upload its parameter model XML (Path C).
func (e *ProvisioningEngine) handleModelUpload(ctx context.Context, task *ProvisioningTask,
	dev *model.Device) error {

	if err := e.transitionTask(ctx, task, StateDiscovering); err != nil {
		return e.failTask(ctx, task, fmt.Errorf("transition to discovering: %w", err))
	}

	_, err := e.modelUploadService.RequestModelUpload(ctx, dev, task.ID.String())
	if err != nil {
		return e.failTask(ctx, task, fmt.Errorf("request model upload: %w", err))
	}

	e.logger.Info("parameter model upload requested",
		zap.String("device_sn", dev.SerialNumber),
		zap.String("task_id", task.ID.String()),
	)

	return nil
}

// handleAutoSync initiates Path B parameter value synchronization.
func (e *ProvisioningEngine) handleAutoSync(ctx context.Context, task *ProvisioningTask,
	dev *model.Device) error {
	if err := e.transitionTask(ctx, task, StateSyncing); err != nil {
		return e.failTask(ctx, task, fmt.Errorf("transition to syncing: %w", err))
	}

	used, _, err := e.syncService.StartPathBSync(ctx, dev, task.ID.String(), WithReason("bootstrap"))
	if err != nil {
		return e.failTask(ctx, task, fmt.Errorf("start path-b sync: %w", err))
	}
	if !used {
		return e.failTask(ctx, task, fmt.Errorf("path-b sync unavailable for device %s", dev.SerialNumber))
	}

	e.logger.Info("path-b auto-sync initiated",
		zap.String("device_sn", dev.SerialNumber),
		zap.String("task_id", task.ID.String()),
	)
	return nil
}

// HandleRPCResult processes an RPC response and advances the provisioning state.
func (e *ProvisioningEngine) HandleRPCResult(ctx context.Context, deviceSN string, method string, success bool, errMsg string) error {
	dev, err := e.deviceService.GetBySerialNumber(ctx, deviceSN)
	if err != nil {
		return fmt.Errorf("find device by SN %s: %w", deviceSN, err)
	}
	if dev == nil {
		return fmt.Errorf("device not found: %s", deviceSN)
	}

	task, err := e.taskRepo.GetByDeviceID(ctx, dev.ID)
	if err != nil {
		return fmt.Errorf("get task for device %s: %w", deviceSN, err)
	}

	if IsTerminal(task.Status) {
		return nil
	}

	task.CurrentStep++
	if err := e.taskRepo.Update(ctx, task); err != nil {
		return fmt.Errorf("update task step: %w", err)
	}

	e.publishEvent(ctx, event.SubjectProvisionStepDone, map[string]interface{}{
		"task_id":      task.ID,
		"device_sn":    deviceSN,
		"method":       method,
		"success":      success,
		"current_step": task.CurrentStep,
		"total_steps":  task.TotalSteps,
	})

	if !success {
		task.RetryCount++
		if task.RetryCount >= task.MaxRetries {
			return e.failTask(ctx, task, fmt.Errorf("step %s failed after %d retries: %s", method, task.RetryCount, errMsg))
		}
		e.logger.Warn("provisioning step failed, will retry",
			zap.String("device_sn", deviceSN),
			zap.String("method", method),
			zap.Int("retry", task.RetryCount),
			zap.String("error", errMsg),
		)
		task.ErrorMessage = errMsg
		return e.taskRepo.Update(ctx, task)
	}

	if task.CurrentStep >= task.TotalSteps {
		return e.completeTask(ctx, task)
	}

	return nil
}

func (e *ProvisioningEngine) transitionTask(ctx context.Context, task *ProvisioningTask, target ProvisioningState) error {
	if err := ValidateTransition(task.Status, target); err != nil {
		return err
	}
	task.Status = target
	return e.taskRepo.UpdateStatus(ctx, task.ID, target, "")
}

// OnTaskCompleted 实现 task.TaskCompletionCallback 接口。
// CompletionRouter 在 NATS 投递 task.completed/task.failed（system source）时调用。
//
// 联动语义（D2 修复）：
//   - device_task.SourceID 携带的是 ProvisioningTask.ID（由 sync.go / model_upload.go
//     在 CreateTask 时设置）。本回调用 SourceID 反查 ProvisioningTask 并按 device_task
//     的终态推动它前进。
//   - **failed**：立即把 ProvisioningTask 置 failed（任意一个 GPV/Upload/SPV 子任务失败
//     都视为整批 provisioning 失败，例如 BAICELLS 9005 Fault）。
//   - **completed**：当前不在此回调内推进 ProvisioningTask（多个子任务/响应分摊推进
//     由 HandleRPCResult 和 handleGPVResponse 各自路径处理）；此回调只对 failed 兜底。
func (e *ProvisioningEngine) OnTaskCompleted(ctx context.Context, t *task.Task) {
	if t == nil {
		return
	}
	if t.Status == task.TaskStatusCompleted {
		if t.Method == "Download" && strings.HasPrefix(t.CommandKey, "PNPXML_") && t.SourceID != "" {
			if id, err := uuid.Parse(t.SourceID); err == nil {
				if pt, getErr := e.taskRepo.GetByID(ctx, id); getErr == nil && pt != nil && !IsTerminal(pt.Status) {
					pt.CurrentStep = 6
					pt.CurrentStepName = "wait_transfer_complete"
					if updateErr := e.taskRepo.Update(ctx, pt); updateErr != nil {
						e.logger.Error("update plug and play XML task progress",
							zap.String("provisioning_task_id", id.String()), zap.Error(updateErr))
						return
					}
				}
			}
			return
		}
		e.maybeFinalizeRecoveredSyncGPV(ctx, t)
		return
	}
	if t.Status != task.TaskStatusFailed && t.Status != task.TaskStatusExpired {
		return
	}
	if t.SourceID == "" {
		return
	}
	ptID, err := uuid.Parse(t.SourceID)
	if err != nil {
		e.logger.Warn("OnTaskCompleted: source_id is not a valid uuid, skipping",
			zap.String("task_id", t.ID),
			zap.String("source_id", t.SourceID),
			zap.Error(err))
		return
	}
	pt, err := e.taskRepo.GetByID(ctx, ptID)
	if err != nil {
		e.logger.Warn("OnTaskCompleted: lookup provisioning task failed",
			zap.String("provisioning_task_id", ptID.String()),
			zap.Error(err))
		return
	}
	if pt == nil || IsTerminal(pt.Status) {
		return
	}
	cause := fmt.Errorf("device task %s (%s) failed: code=%d %s",
		t.ID, t.Method, t.ErrorCode, t.ErrorMessage)
	if failErr := e.failTask(ctx, pt, cause); failErr != nil {
		e.logger.Error("OnTaskCompleted: failTask error",
			zap.String("provisioning_task_id", ptID.String()),
			zap.Error(failErr))
		return
	}
	e.logger.Info("provisioning task failed by device task callback",
		zap.String("provisioning_task_id", ptID.String()),
		zap.String("device_task_id", t.ID),
		zap.String("device_sn", t.DeviceSN),
		zap.Int("error_code", t.ErrorCode))
}

// HandleXMLTransferComplete makes the CPE report authoritative for a
// plug-and-play XML download. DownloadResponse only acknowledges the RPC.
func (e *ProvisioningEngine) HandleXMLTransferComplete(ctx context.Context, transfer tr069.TransferComplete) error {
	const commandPrefix = "PNPXML_"
	if !strings.HasPrefix(transfer.CommandKey, commandPrefix) {
		return nil
	}
	taskID, err := uuid.Parse(strings.TrimPrefix(transfer.CommandKey, commandPrefix))
	if err != nil {
		return nil
	}
	pt, err := e.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return fmt.Errorf("get plug and play task for TransferComplete: %w", err)
	}
	if pt == nil || IsTerminal(pt.Status) {
		return nil
	}
	if pt.CurrentStepName == "wait_device_online" ||
		pt.CurrentStepName == "wait_activation_check" ||
		pt.CurrentStepName == "wait_activation_sync" {
		return nil
	}

	pt.CurrentStep = 6
	pt.CurrentStepName = "wait_transfer_complete"
	if transfer.FaultStruct != nil && (transfer.FaultStruct.FaultCode != 0 || strings.TrimSpace(transfer.FaultStruct.FaultString) != "") {
		if err := e.taskRepo.Update(ctx, pt); err != nil {
			return fmt.Errorf("update failed XML transfer progress: %w", err)
		}
		return e.failTask(ctx, pt, fmt.Errorf("TransferComplete fault %d: %s",
			transfer.FaultStruct.FaultCode, transfer.FaultStruct.FaultString))
	}
	dev, err := e.deviceService.GetDevice(ctx, pt.DeviceID)
	if err != nil {
		return fmt.Errorf("get plug and play device for TransferComplete: %w", err)
	}
	if dev == nil {
		return e.failTask(ctx, pt, fmt.Errorf("plug and play device %s not found after TransferComplete", pt.DeviceID))
	}
	if dev.Technology != model.TechNR {
		// LTE/GSM devices do not reboot automatically after applying the legacy
		// Auto Start File. Queue an explicit Reboot and wait for the subsequent
		// BOOT-driven device.online event before starting the five-minute window.
		pt.Status = StateConfiguring
		pt.CurrentStep = 7
		pt.CurrentStepName = "wait_device_online"
		pt.RetryCount = 0
		if err := e.taskRepo.Update(ctx, pt); err != nil {
			return fmt.Errorf("prepare automatic-start reboot: %w", err)
		}
		noRetries := 0
		rebootTask, err := e.taskSvc.CreateTask(ctx, &task.CreateTaskRequest{
			DeviceSN:    dev.SerialNumber,
			Method:      MethodReboot,
			Params:      json.RawMessage(`{}`),
			Source:      task.TaskSourceSystem,
			SourceID:    pt.ID.String(),
			CommandKey:  "PNPREBOOT_" + pt.ID.String(),
			Description: "plug and play reboot after automatic-start XML",
			MaxRetries:  &noRetries,
		})
		if err != nil {
			return e.failTask(ctx, pt, fmt.Errorf("enqueue automatic-start Reboot: %w", err))
		}
		if rebootTask != nil {
			if id, parseErr := uuid.Parse(rebootTask.ID); parseErr == nil {
				pt.DeviceTaskID = &id
			}
			if err := e.taskRepo.Update(ctx, pt); err != nil {
				return fmt.Errorf("record automatic-start Reboot task: %w", err)
			}
		}
		return nil
	}

	pt.CurrentStep = 6
	pt.CurrentStepName = "wait_startup_stage"
	if err := e.taskRepo.Update(ctx, pt); err != nil {
		return fmt.Errorf("update successful XML transfer progress: %w", err)
	}
	return nil
}

// HandleStartupStageReport advances a PnP XML task from device-reported CMCC
// stages. Connection/file-transfer stages are OMC-derived; Stage 1..3 cover
// validation, configuration and cell activation.
func (e *ProvisioningEngine) HandleStartupStageReport(ctx context.Context, payload device.InformEventPayload) error {
	pt, err := e.activePlugAndPlayTask(ctx, payload)
	if err != nil || pt == nil {
		return err
	}
	stage := informParameterValue(payload.ParameterList, "Stage")
	var step int
	var name string
	switch stage {
	case "1":
		step, name = 7, "parameter_validation"
	case "2":
		step, name = 8, "parameter_configuration"
	case "3":
		step, name = 9, "cell_activation"
	default:
		return fmt.Errorf("unsupported automatic-start Stage %q for device %s", stage, payload.DeviceId.SerialNumber)
	}
	if step < pt.CurrentStep {
		return nil
	}
	pt.Status = StateConfiguring
	pt.CurrentStep, pt.CurrentStepName = step, name
	if err := e.taskRepo.Update(ctx, pt); err != nil {
		return fmt.Errorf("update automatic-start stage %s: %w", stage, err)
	}
	return nil
}

// HandleStartupResultReport closes a PnP XML task only after the authoritative
// device result event. Status=1 means every cell is active; Status=2 requires a
// readable FailureCause.
func (e *ProvisioningEngine) HandleStartupResultReport(ctx context.Context, payload device.InformEventPayload) error {
	pt, err := e.activePlugAndPlayTask(ctx, payload)
	if err != nil || pt == nil {
		return err
	}
	status := informParameterValue(payload.ParameterList, "Status")
	failureCause := informParameterValue(payload.ParameterList, "FailureCause")
	pt.CurrentStep, pt.CurrentStepName = 10, "wait_startup_result"
	if err := e.taskRepo.Update(ctx, pt); err != nil {
		return fmt.Errorf("update automatic-start result progress: %w", err)
	}
	switch status {
	case "1":
		pt.CurrentStep, pt.CurrentStepName = 11, "verify_online"
		if err := e.taskRepo.Update(ctx, pt); err != nil {
			return fmt.Errorf("update automatic-start verification progress: %w", err)
		}
		pt.CurrentStep, pt.CurrentStepName = pt.TotalSteps, "completed"
		if err := e.taskRepo.Update(ctx, pt); err != nil {
			return fmt.Errorf("update successful automatic-start result: %w", err)
		}
		return e.completeTask(ctx, pt)
	case "2":
		if strings.TrimSpace(failureCause) == "" {
			failureCause = "device reported automatic-start failure without FailureCause"
		}
		return e.failTask(ctx, pt, fmt.Errorf("%s", failureCause))
	default:
		return fmt.Errorf("unsupported automatic-start Status %q for device %s", status, payload.DeviceId.SerialNumber)
	}
}

type delegatedUpgradeEvent struct {
	TaskID   string `json:"task_id"`
	DeviceID string `json:"device_id"`
	Reason   string `json:"reason"`
}

// HandleDelegatedUpgradeResult mirrors the authoritative file-management
// upgrade result into the PnP task row. Creating the UFTE task is only an
// acknowledgement and must never be treated as successful completion.
func (e *ProvisioningEngine) HandleDelegatedUpgradeResult(
	ctx context.Context,
	payload delegatedUpgradeEvent,
	failed bool,
) error {
	delegatedTaskID, err := uuid.Parse(strings.TrimSpace(payload.TaskID))
	if err != nil {
		return fmt.Errorf("parse delegated upgrade task id %q: %w", payload.TaskID, err)
	}
	deviceID, err := uuid.Parse(strings.TrimSpace(payload.DeviceID))
	if err != nil {
		return fmt.Errorf("parse delegated upgrade device id %q: %w", payload.DeviceID, err)
	}
	pt, err := e.taskRepo.GetByDelegatedTaskID(ctx, delegatedTaskID, deviceID)
	if err != nil {
		return fmt.Errorf("get delegated plug and play upgrade task: %w", err)
	}
	if pt == nil || IsTerminal(pt.Status) {
		return nil
	}
	if failed {
		reason := strings.TrimSpace(payload.Reason)
		if reason == "" {
			reason = "software upgrade failed"
		}
		return e.failTask(ctx, pt, fmt.Errorf("%s", reason))
	}
	completedModule, canContinue := policyModuleFromStepName(pt.CurrentStepName)
	if completedModule == policyModuleSelfConfig {
		// Compatibility for non-gNB CONFIG_RESTORE tasks already in flight during
		// deployment. New self-configuration tasks use PNPXML_ Download and start
		// this timer in HandleXMLTransferComplete.
		pt.Status = StateVerifying
		pt.CurrentStep = 11
		pt.CurrentStepName = "wait_activation_check"
		pt.RetryCount = 0
		pt.MaxRetries = activationCheckMaxAttempts
		if err := e.taskRepo.Update(ctx, pt); err != nil {
			return fmt.Errorf("start self configuration cell-state timer: %w", err)
		}
		return nil
	}
	pt.CurrentStep = pt.TotalSteps
	pt.CurrentStepName = string(completedModule) + "_completed"
	if err := e.taskRepo.Update(ctx, pt); err != nil {
		return fmt.Errorf("update delegated upgrade completion progress: %w", err)
	}
	if err := e.completeTask(ctx, pt); err != nil {
		return err
	}
	if canContinue && pt.PolicyID != nil && e.policyContinuation != nil {
		return e.policyContinuation.ContinuePolicy(ctx, *pt.PolicyID, pt.DeviceID, completedModule)
	}
	return nil
}

// CheckDueActivationTasks submits a fresh full parameter query for non-gNB
// self-configuration tasks after the five-minute post-reboot-online window.
// The terminal sync event refreshes and evaluates device_info.cell_status.
func (e *ProvisioningEngine) CheckDueActivationTasks(ctx context.Context, now time.Time) error {
	if e.activationState == nil || e.deviceOnlineSync == nil {
		return nil
	}
	items, err := e.taskRepo.ListActivationChecksDue(ctx, now.Add(-activationCheckDelay), 100)
	if err != nil {
		return err
	}
	for i := range items {
		item := &items[i]
		if item.CurrentStepName != "wait_activation_check" {
			continue
		}
		dev, lookupErr := e.deviceService.GetDevice(ctx, item.DeviceID)
		if lookupErr != nil {
			return fmt.Errorf("lookup device for activation query: %w", lookupErr)
		}
		if dev == nil {
			return fmt.Errorf("device %s not found for activation query", item.DeviceID)
		}
		sourceID := fmt.Sprintf("pnp-activation:%s:%d", item.ID, item.RetryCount+1)
		if err := e.startDeviceOnlineFullSync(ctx, dev, sourceID, activationCheckOrigin); err != nil {
			return fmt.Errorf("submit activation parameter query for device %s: %w", item.DeviceID, err)
		}
		item.CurrentStepName = "wait_activation_sync"
		item.ErrorMessage = ""
		if err := e.taskRepo.Update(ctx, item); err != nil {
			return fmt.Errorf("record activation parameter query for device %s: %w", item.DeviceID, err)
		}
	}
	return nil
}

func (e *ProvisioningEngine) handleActivationSyncCompleted(ctx context.Context, deviceID uuid.UUID) error {
	pt, err := e.taskRepo.GetByDeviceID(ctx, deviceID)
	if err != nil {
		return fmt.Errorf("get plug and play task after activation sync: %w", err)
	}
	if pt == nil || IsTerminal(pt.Status) || pt.CurrentStepName != "wait_activation_sync" {
		return nil
	}
	dev, err := e.deviceService.GetDevice(ctx, deviceID)
	if err != nil {
		return fmt.Errorf("get device after activation sync: %w", err)
	}
	if dev == nil {
		return e.recordActivationCheckResult(ctx, pt, nil, fmt.Errorf("device not found after activation sync"))
	}
	if e.activationRefresher != nil {
		if _, err := e.activationRefresher.SyncFromParameters(
			ctx, dev.ID, dev.Carrier, dev.Technology, dev.ProductClass,
		); err != nil {
			return fmt.Errorf("refresh activation state after parameter sync: %w", err)
		}
	}
	info, readErr := e.activationState.GetByDeviceID(ctx, deviceID)
	return e.recordActivationCheckResult(ctx, pt, info, readErr)
}

func (e *ProvisioningEngine) handleActivationSyncFailedEvent(ctx context.Context, evt event.Event) error {
	var failed paramSyncCompletedEvent
	if err := evt.DecodePayload(&failed); err != nil {
		return fmt.Errorf("decode activation parameter sync failure: %w", err)
	}
	if failed.DeviceID == uuid.Nil || failed.TriggerReason != "device_online" || failed.SyncScope != "full" {
		return nil
	}
	pt, err := e.taskRepo.GetByDeviceID(ctx, failed.DeviceID)
	if err != nil {
		return fmt.Errorf("get plug and play task after activation sync failure: %w", err)
	}
	if pt == nil || IsTerminal(pt.Status) || pt.CurrentStepName != "wait_activation_sync" {
		return nil
	}
	return e.recordActivationCheckResult(ctx, pt, nil, fmt.Errorf("activation parameter query failed"))
}

func (e *ProvisioningEngine) recordActivationCheckResult(
	ctx context.Context,
	pt *ProvisioningTask,
	info *device.DeviceInfo,
	readErr error,
) error {
	if info != nil && strings.TrimSpace(info.CellStatus) == string(device.CellStatusNormal) {
		pt.ErrorMessage = ""
		pt.CurrentStep = pt.TotalSteps
		pt.CurrentStepName = "activation_verified"
		return e.completeTask(ctx, pt)
	}
	checkErr := readErr
	if checkErr == nil {
		cellStatus := "unknown"
		if info != nil && strings.TrimSpace(info.CellStatus) != "" {
			cellStatus = strings.TrimSpace(info.CellStatus)
		}
		checkErr = fmt.Errorf("cell status is %s", cellStatus)
	}
	if pt.MaxRetries <= 0 {
		pt.MaxRetries = activationCheckMaxAttempts
	}
	attempt := pt.RetryCount + 1
	pt.RetryCount = attempt
	if attempt < pt.MaxRetries {
		pt.CurrentStepName = "wait_activation_check"
		pt.ErrorMessage = fmt.Sprintf(
			"activation check failed (%d/%d checks), next check in %s: %s",
			pt.RetryCount, pt.MaxRetries, activationCheckDelay, checkErr,
		)
		if err := e.taskRepo.Update(ctx, pt); err != nil {
			return fmt.Errorf("schedule activation check retry for device %s: %w", pt.DeviceID, err)
		}
		return nil
	}
	if err := e.taskRepo.Update(ctx, pt); err != nil {
		return fmt.Errorf("record final activation check for device %s: %w", pt.DeviceID, err)
	}
	return e.failTask(ctx, pt, fmt.Errorf(
		"activation check failed after %d checks: %s", pt.MaxRetries, checkErr,
	))
}

func (e *ProvisioningEngine) activePlugAndPlayTask(ctx context.Context, payload device.InformEventPayload) (*ProvisioningTask, error) {
	if e.deviceService == nil {
		return nil, fmt.Errorf("device service is unavailable")
	}
	dev, err := e.deviceService.GetBySerialNumber(ctx, strings.TrimSpace(payload.DeviceId.SerialNumber))
	if err != nil {
		return nil, fmt.Errorf("get device for automatic-start event: %w", err)
	}
	if dev == nil {
		return nil, nil
	}

	var pt *ProvisioningTask
	for _, evt := range payload.EventStructs {
		const prefix = "PNPXML_"
		if !strings.HasPrefix(evt.CommandKey, prefix) {
			continue
		}
		taskID, parseErr := uuid.Parse(strings.TrimPrefix(evt.CommandKey, prefix))
		if parseErr != nil {
			return nil, fmt.Errorf("parse automatic-start command key %q: %w", evt.CommandKey, parseErr)
		}
		pt, err = e.taskRepo.GetByID(ctx, taskID)
		if err != nil {
			return nil, fmt.Errorf("get automatic-start task by command key: %w", err)
		}
		if pt != nil && pt.DeviceID != dev.ID {
			return nil, fmt.Errorf("automatic-start task %s does not belong to device %s", taskID, payload.DeviceId.SerialNumber)
		}
		break
	}
	if pt == nil {
		pt, err = e.taskRepo.GetByDeviceID(ctx, dev.ID)
	}
	if err != nil {
		return nil, fmt.Errorf("get automatic-start task by device: %w", err)
	}
	if pt == nil || pt.PolicyID == nil || pt.XMLFileID == nil || IsTerminal(pt.Status) {
		return nil, nil
	}
	return pt, nil
}

func informParameterValue(values []tr069.ParameterValueStruct, leaf string) string {
	for _, value := range values {
		name := strings.TrimSpace(value.Name)
		if strings.EqualFold(name, leaf) || strings.HasSuffix(strings.ToLower(name), "."+strings.ToLower(leaf)) {
			return strings.TrimSpace(value.Value)
		}
	}
	return ""
}

func (e *ProvisioningEngine) maybeFinalizeRecoveredSyncGPV(ctx context.Context, t *task.Task) {
	if e == nil || e.syncService == nil || e.deviceService == nil || t == nil {
		return
	}
	if t.Method != "GetParameterValues" || !isFullSyncTrigger(t.CommandKey) || len(t.Result) == 0 {
		return
	}
	var result struct {
		Recovered    bool `json:"recovered"`
		RemainingCnt int  `json:"remaining_cnt"`
	}
	if err := json.Unmarshal(t.Result, &result); err != nil {
		return
	}
	if !result.Recovered || result.RemainingCnt != 0 {
		return
	}
	dev, err := e.deviceService.GetBySerialNumber(ctx, t.DeviceSN)
	if err != nil {
		e.logger.Warn("OnTaskCompleted: lookup device for recovered sync finalize failed",
			zap.String("device_sn", t.DeviceSN),
			zap.String("device_task_id", t.ID),
			zap.Error(err))
		return
	}
	if dev == nil {
		return
	}
	if !e.syncService.shouldFinalizePathBSync(ctx, dev, t.CommandKey) {
		return
	}
	if err := e.syncService.finalizePathBSync(ctx, dev); err != nil {
		e.logger.Warn("OnTaskCompleted: finalize recovered sync-gpv failed",
			zap.String("device_sn", t.DeviceSN),
			zap.String("device_task_id", t.ID),
			zap.String("command_key", t.CommandKey),
			zap.Error(err))
		return
	}
	e.logger.Info("recovered sync-gpv finalized via completion callback",
		zap.String("device_sn", t.DeviceSN),
		zap.String("device_task_id", t.ID),
		zap.String("command_key", t.CommandKey))
}

func (e *ProvisioningEngine) failTask(ctx context.Context, task *ProvisioningTask, cause error) error {
	task.Status = StateFailed
	task.ErrorMessage = cause.Error()
	now := time.Now()
	task.CompletedAt = &now

	if updateErr := e.taskRepo.UpdateStatus(ctx, task.ID, StateFailed, cause.Error()); updateErr != nil {
		e.logger.Error("failed to update task status to failed",
			zap.String("task_id", task.ID.String()),
			zap.Error(updateErr),
		)
	}

	e.publishEvent(ctx, event.SubjectProvisionFailed, map[string]interface{}{
		"task_id":   task.ID,
		"device_id": task.DeviceID,
		"error":     cause.Error(),
	})

	return nil
}

func (e *ProvisioningEngine) completeTask(ctx context.Context, task *ProvisioningTask) error {
	now := time.Now()
	task.Status = StateCompleted
	task.CompletedAt = &now

	if err := e.taskRepo.UpdateStatus(ctx, task.ID, StateCompleted, ""); err != nil {
		return fmt.Errorf("complete task: %w", err)
	}

	if err := e.deviceService.TransitionStatus(ctx, task.DeviceID, model.DeviceActive); err != nil {
		e.logger.Warn("failed to transition device to active",
			zap.String("task_id", task.ID.String()),
			zap.Error(err),
		)
	}

	e.publishEvent(ctx, event.SubjectProvisionCompleted, map[string]interface{}{
		"task_id":   task.ID,
		"device_id": task.DeviceID,
	})

	e.logger.Info("provisioning completed",
		zap.String("task_id", task.ID.String()),
		zap.String("device_id", task.DeviceID.String()),
	)

	return nil
}

func (e *ProvisioningEngine) publishEvent(ctx context.Context, subject string, data interface{}) {
	evt, err := event.NewEvent(subject, data)
	if err != nil {
		e.logger.Error("create event", zap.String("subject", subject), zap.Error(err))
		return
	}
	if err := e.eventBus.Publish(ctx, subject, evt); err != nil {
		e.logger.Warn("publish event", zap.String("subject", subject), zap.Error(err))
	}
}

// handleDataModelFileReceived processes datamodel.file.received events.
//
// 设备 FileType=11 上传 → ACS 存 MinIO 后发本事件；本 handler 拉文件、解析 XML、
// 调 IntersectService 写 discovered_param_mappings。完成后若 AutoSync 启用且
// Path B 可用，触发同步。
func (e *ProvisioningEngine) handleDataModelFileReceived(ctx context.Context, evt event.Event) error {
	var payload dataModelFilePayload
	if err := evt.DecodePayload(&payload); err != nil {
		e.logger.Error("decode datamodel file event", zap.Error(err))
		return nil
	}

	e.logger.Info("received datamodel.file.received",
		zap.String("device_sn", payload.DeviceSN),
		zap.String("path", payload.MinioPath),
		zap.Int64("file_size", payload.FileSize),
	)

	if payload.DeviceSN == "" {
		e.logger.Warn("datamodel file event missing device_sn, skipping")
		return nil
	}

	dev, err := e.deviceService.GetBySerialNumber(ctx, payload.DeviceSN)
	if err != nil {
		e.logger.Error("find device for datamodel file", zap.Error(err), zap.String("device_sn", payload.DeviceSN))
		return nil
	}
	if dev == nil {
		e.logger.Warn("device not found for datamodel file, skipping", zap.String("device_sn", payload.DeviceSN))
		return nil
	}

	// T-0176-PR-D 懒补 product 绑定（FileType11 上传链路也可能命中历史 orphan）。
	e.bindDeviceProductIfNeeded(ctx, dev)

	if e.modelUploadService == nil {
		e.logger.Warn("model upload service not configured, skipping datamodel file")
		return nil
	}

	if err := e.modelUploadService.HandleModelFileReceived(ctx, dev, payload); err != nil {
		e.logger.Error("handle model file received", zap.Error(err), zap.String("device_sn", payload.DeviceSN))
		return nil
	}

	e.logger.Info("model XML processed, discovered mappings written",
		zap.String("device_sn", payload.DeviceSN),
	)

	return nil
}

// gpvResponsePayload is the structured event payload for GetParameterValuesResponse.
type gpvResponsePayload struct {
	DeviceSN        string                       `json:"device_sn"`
	Method          string                       `json:"method"`
	CommandKey      string                       `json:"command_key,omitempty"`
	TaskSource      task.TaskSource              `json:"task_source,omitempty"`
	TaskSourceID    string                       `json:"task_source_id,omitempty"`
	ParameterValues []tr069.ParameterValueStruct `json:"parameter_values"`
}

type gpvWorkItem struct {
	ctx     context.Context
	payload gpvResponsePayload
	result  chan error
}

const (
	gpvPullBatchSize = 64
)

func (e *ProvisioningEngine) ensureGPVWorkersStarted() {
	e.gpvWorkersOnce.Do(func() {
		config := e.config.GPVResponse.Defaults()
		shards := config.ProvisionConcurrency

		e.gpvWorkerChans = make([]chan gpvWorkItem, shards)
		for i := 0; i < shards; i++ {
			ch := make(chan gpvWorkItem, config.ProvisionQueueDepth)
			e.gpvWorkerChans[i] = ch
			go e.runGPVWorker(i, ch)
		}
		e.logger.Info("gpv workers started",
			zap.Int("shards", shards),
			zap.Int("queue_depth", config.ProvisionQueueDepth))
	})
}

func (e *ProvisioningEngine) runGPVWorker(shard int, ch <-chan gpvWorkItem) {
	for item := range ch {
		ctx := item.ctx
		if ctx == nil {
			ctx = context.Background()
		}
		err := e.handleGPVPayload(ctx, item.payload)
		if err != nil {
			e.logger.Error("gpv worker failed",
				zap.Int("shard", shard),
				zap.String("device_sn", item.payload.DeviceSN),
				zap.Error(err))
		}
		if item.result != nil {
			item.result <- err
		}
	}
}

func (e *ProvisioningEngine) enqueueGPVResponseEvent(ctx context.Context, evt event.Event) error {
	var payload gpvResponsePayload
	if err := evt.DecodePayload(&payload); err != nil {
		e.logger.Error("decode GPV response event", zap.Error(err))
		return nil
	}
	if payload.DeviceSN == "" {
		e.logger.Warn("GPV response event missing device SN, skipping")
		return nil
	}

	e.ensureGPVWorkersStarted()
	if len(e.gpvWorkerChans) == 0 {
		return fmt.Errorf("gpv workers not initialized")
	}
	shard := gpvShardIndex(payload.DeviceSN, len(e.gpvWorkerChans))
	result := make(chan error, 1)
	item := gpvWorkItem{ctx: ctx, payload: payload, result: result}

	select {
	case e.gpvWorkerChans[shard] <- item:
	case <-ctx.Done():
		return fmt.Errorf("enqueue GPV response for %s: %w", payload.DeviceSN, ctx.Err())
	}
	select {
	case err := <-result:
		return err
	case <-ctx.Done():
		return fmt.Errorf("wait GPV response for %s: %w", payload.DeviceSN, ctx.Err())
	}
}

func gpvShardIndex(deviceSN string, shardCount int) int {
	if shardCount <= 1 {
		return 0
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(deviceSN))
	return int(h.Sum32() % uint32(shardCount))
}

func provisionGPVDeviceKey(evt event.Event) (string, error) {
	var payload gpvResponsePayload
	if err := evt.DecodePayload(&payload); err != nil {
		return "", fmt.Errorf("decode GPV response key: %w", err)
	}
	if payload.DeviceSN == "" {
		return "", fmt.Errorf("GPV response key missing device SN")
	}
	return payload.DeviceSN, nil
}

// handleGPVResponse processes GetParameterValuesResponse events from ACS.
//
// T-0098 P5-01：先尝试 Path B 翻译落库（standardPath）；命中即返回，否则
// 用旧 HandleSyncResult 直写 privatePath（兜底，参数保留可见性）。
func (e *ProvisioningEngine) handleGPVResponse(ctx context.Context, evt event.Event) error {
	var payload gpvResponsePayload
	if err := evt.DecodePayload(&payload); err != nil {
		e.logger.Error("decode GPV response event", zap.Error(err))
		return nil
	}
	return e.handleGPVPayload(ctx, payload)
}

func (e *ProvisioningEngine) handleGPVPayload(ctx context.Context, payload gpvResponsePayload) error {
	// Durable parameter-sync owns its result persistence and finalization. The
	// legacy provisioning consumer must not write the same response directly to
	// device_parameters or update last_param_sync_at. CommandKey is retained as
	// a compatibility guard for fault events emitted by older ACS instances.
	if payload.TaskSource == task.TaskSourceParamSync || strings.HasPrefix(payload.CommandKey, "param-sync-") {
		e.logger.Debug("skip durable parameter-sync response in legacy Path B",
			zap.String("device_sn", payload.DeviceSN),
			zap.String("task_source_id", payload.TaskSourceID),
		)
		return nil
	}
	e.logger.Info("received GPV response",
		zap.String("device_sn", payload.DeviceSN),
		zap.Int("parameter_count", len(payload.ParameterValues)),
	)

	dev, err := e.deviceService.GetBySerialNumber(ctx, payload.DeviceSN)
	if err != nil {
		e.logger.Error("find device for GPV response", zap.Error(err), zap.String("device_sn", payload.DeviceSN))
		return fmt.Errorf("find device for GPV response %s: %w", payload.DeviceSN, err)
	}
	if dev == nil {
		e.logger.Warn("device not found for GPV response, skipping", zap.String("device_sn", payload.DeviceSN))
		return nil
	}

	if e.syncService == nil {
		return nil
	}

	if used, err := e.syncService.HandleSyncResultPathB(ctx, dev, payload.ParameterValues, payload.CommandKey); err != nil {
		e.logger.Error("path-b save parameter values", zap.Error(err), zap.String("device_sn", payload.DeviceSN))
		return fmt.Errorf("save path-b GPV response for %s: %w", payload.DeviceSN, err)
	} else if used {
		e.logger.Info("path-b parameter values saved",
			zap.String("device_sn", payload.DeviceSN),
			zap.Int("count", len(payload.ParameterValues)),
		)
		return nil
	}

	if err := e.syncService.HandleSyncResult(ctx, dev, payload.ParameterValues); err != nil {
		e.logger.Error("save parameter values", zap.Error(err), zap.String("device_sn", payload.DeviceSN))
		return fmt.Errorf("save GPV response for %s: %w", payload.DeviceSN, err)
	}
	e.logger.Info("parameter values saved (privatePath direct)",
		zap.String("device_sn", payload.DeviceSN),
		zap.Int("count", len(payload.ParameterValues)),
	)
	return nil
}

// StartTaskReaper launches a background goroutine that periodically fails
// provisioning tasks stuck in non-terminal states beyond the configured timeout.
func (e *ProvisioningEngine) StartTaskReaper() {
	timeout := e.config.TaskTimeout
	if timeout <= 0 {
		timeout = 15 * time.Minute
	}
	interval := timeout / 2
	if interval > 30*time.Second || interval <= 0 {
		interval = 30 * time.Second
	}

	e.logger.Info("provisioning task reaper started",
		zap.Duration("timeout", timeout),
		zap.Duration("interval", interval))

	go func() {
		reap := func() {
			ctx := context.Background()
			if err := e.CheckDueActivationTasks(ctx, time.Now()); err != nil {
				e.logger.Error("provisioning task reaper: check activation state", zap.Error(err))
			}
			n, err := e.taskRepo.FailStale(ctx, timeout)
			if err != nil {
				e.logger.Error("provisioning task reaper: fail stale tasks", zap.Error(err))
				return
			}
			if n > 0 {
				e.logger.Warn("provisioning task reaper: timed out stale tasks",
					zap.Int64("count", n),
					zap.Duration("timeout", timeout))
			}
		}

		reap()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			reap()
		}
	}()
}

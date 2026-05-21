package provision

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/config/template"
	"github.com/omcgo/omcgo/internal/core/appconfig"
	"github.com/omcgo/omcgo/internal/core/carrier"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
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
	productRegistry    *product.Registry
	productRepo        *product.PgRepository
	redisClient        redis.UniversalClient
	config             appconfig.ProvisionConfig
	logger             *zap.Logger
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

// SetRedisClient 注入 Redis 客户端供 device.online 节流与 Path B 同步差异日志使用（T-0123）。
// nil 表示禁用 token bucket 节流（仍可正常处理事件，幂等性由下游 GPV/UPSERT 保证）。
func (e *ProvisioningEngine) SetRedisClient(client redis.UniversalClient) {
	e.redisClient = client
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
func (e *ProvisioningEngine) SetProductBinder(reg *product.Registry, repo *product.PgRepository) {
	e.productRegistry = reg
	e.productRepo = repo
}

// bindDeviceProduct 在 HandleBootstrap identify 阶段调 productRegistry 路由
// productClass 命中产品后，把 product_id / param_model_id 写回 device 行。
// 任何错误都仅记 warn 不阻断后续 provisioning（孤儿设备由前端孤儿设备页处理）。
func (e *ProvisioningEngine) bindDeviceProduct(ctx context.Context, dev *model.Device) {
	if e.productRegistry == nil || e.productRepo == nil || dev == nil || dev.ProductClass == "" {
		return
	}
	match, err := e.productRegistry.MatchProductClass(ctx, dev.ProductClass)
	if err != nil {
		e.logger.Warn("bindDeviceProduct: match productClass failed",
			zap.String("device_sn", dev.SerialNumber),
			zap.String("product_class", dev.ProductClass),
			zap.Error(err))
		return
	}
	if match == nil || match.Product == nil {
		return
	}
	if err := e.productRepo.BindDevice(ctx, dev.ID, match.Product.ID, match.Product.ParamModelID); err != nil {
		e.logger.Warn("bindDeviceProduct: write product_id failed",
			zap.String("device_sn", dev.SerialNumber),
			zap.String("product_id", match.Product.ID.String()),
			zap.Error(err))
		return
	}
	e.logger.Info("device bound to product",
		zap.String("device_sn", dev.SerialNumber),
		zap.String("product_id", match.Product.ID.String()),
		zap.String("product_name", match.Product.Name))
}

// bootstrapEvent represents the data published on device.registered.
type bootstrapEvent struct {
	DeviceID     uuid.UUID `json:"device_id"`
	SerialNumber string    `json:"serial_number"`
	OUI          string    `json:"oui"`
	ProductClass string    `json:"product_class"`
	Carrier      string    `json:"carrier"`
	Technology   string    `json:"technology"`
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

	// Subscribe to datamodel.file.received for Path C model upload processing.
	if _, err := bus.QueueSubscribe(event.SubjectDataModelFileReceived, "provision-model-upload", func(ctx context.Context, evt event.Event) error {
		return e.handleDataModelFileReceived(ctx, evt)
	}); err != nil {
		e.logger.Warn("failed to subscribe to datamodel.file.received", zap.Error(err))
	} else {
		e.logger.Info("provisioning engine subscribed to datamodel.file.received events")
	}

	// Subscribe to GPV response for parameter sync processing.
	if _, err := bus.QueueSubscribe(event.SubjectCommandGetParamsResponse, "provision-gpv", func(ctx context.Context, evt event.Event) error {
		return e.handleGPVResponse(ctx, evt)
	}); err != nil {
		e.logger.Warn("failed to subscribe to GPV response", zap.Error(err))
	} else {
		e.logger.Info("provisioning engine subscribed to GPV response events")
	}

	// T-0098 P5-01：GPN response 订阅删除（two-phase sync 已移除）。

	// T-0123: 订阅 device.online — 已存在设备从 offline 恢复时触发 Path B 全量同步。
	onlineHandler := func(ctx context.Context, evt event.Event) error {
		var onlineEvt device.DeviceOnlineEvent
		if err := evt.DecodePayload(&onlineEvt); err != nil {
			e.logger.Error("decode device.online event", zap.Error(err),
				zap.String("event_id", evt.ID))
			return err
		}
		return e.HandleDeviceOnline(ctx, onlineEvt)
	}
	if e.deduper != nil {
		onlineHandler = e.deduper.Wrap("provision-online-sync", onlineHandler)
	}
	if _, err := bus.QueueSubscribe(event.SubjectDeviceOnline, "provision-online-sync", onlineHandler); err != nil {
		e.logger.Warn("failed to subscribe to device.online", zap.Error(err))
	} else {
		e.logger.Info("provisioning engine subscribed to device.online events")
	}

	// T-0125: 订阅 device.firmware.changed — 固件升级后重新交集 + Path B 同步。
	firmwareHandler := func(ctx context.Context, evt event.Event) error {
		var fwEvt device.DeviceFirmwareChangedEvent
		if err := evt.DecodePayload(&fwEvt); err != nil {
			e.logger.Error("decode device.firmware.changed event", zap.Error(err),
				zap.String("event_id", evt.ID))
			return err
		}
		return e.HandleFirmwareChanged(ctx, fwEvt)
	}
	if e.deduper != nil {
		firmwareHandler = e.deduper.Wrap("provision-firmware-changed", firmwareHandler)
	}
	if _, err := bus.QueueSubscribe(event.SubjectDeviceFirmwareChanged, "provision-firmware-changed", firmwareHandler); err != nil {
		e.logger.Warn("failed to subscribe to device.firmware.changed", zap.Error(err))
	} else {
		e.logger.Info("provisioning engine subscribed to device.firmware.changed events")
	}

	return nil
}

// HandleDeviceOnline 处理已存在设备从 offline 恢复 active 的事件（T-0123）。
//
// 与 HandleBootstrap 的差异：不重跑 FileType=11 上传与产品路由（首次工作已完成），
// 直接调 syncService.StartPathBSync 拉一遍参数检测离线期间漂移。
//
// 节流：Redis token bucket key=provision:online_sync:{deviceID} TTL=60s，
// 60s 内同一设备的重复 device.online 事件直接跳过（防 ACS 抖动/multiple Inform 触发）。
func (e *ProvisioningEngine) HandleDeviceOnline(ctx context.Context, evt device.DeviceOnlineEvent) error {
	ctx, span := tracing.StartSpan(ctx, tracing.ProvisionTracerName, "Provision HandleDeviceOnline",
		attribute.String("provision.device_sn", evt.SerialNumber),
		attribute.String("provision.device_id", evt.DeviceID.String()),
	)
	defer span.End()

	// Token bucket 节流：60s 内重复 device.online 跳过
	if e.redisClient != nil {
		key := fmt.Sprintf("provision:online_sync:%s", evt.DeviceID.String())
		acquired, err := e.redisClient.SetNX(ctx, key, "1", 60*time.Second).Result()
		if err != nil {
			// Redis 失败不阻断主流程：log warn 后继续（容忍 Redis 抖动期间允许重复同步）
			e.logger.Warn("device.online token bucket SetNX failed",
				zap.String("device_id", evt.DeviceID.String()),
				zap.Error(err))
		} else if !acquired {
			e.logger.Debug("device.online throttled by token bucket",
				zap.String("device_id", evt.DeviceID.String()),
				zap.String("serial_number", evt.SerialNumber))
			return nil
		}
	}

	if e.syncService == nil {
		e.logger.Debug("device.online received but syncService unavailable, skipping",
			zap.String("device_id", evt.DeviceID.String()))
		return nil
	}

	dev, err := e.deviceService.GetDevice(ctx, evt.DeviceID)
	if err != nil {
		e.logger.Warn("device.online: lookup device failed",
			zap.String("device_id", evt.DeviceID.String()),
			zap.Error(err))
		return nil
	}
	if dev == nil {
		e.logger.Warn("device.online: device not found, skipping",
			zap.String("device_id", evt.DeviceID.String()))
		return nil
	}

	// SourceID 是裸 UUID（写入 device_tasks.source_id UUID 列做溯源）；reason="device_online"
	// 通过 WithReason 走 Redis 通道传给 HandleSyncResultPathB 打差异日志（T-0127）。
	sourceID := evt.DeviceID.String()
	used, err := e.syncService.StartPathBSync(ctx, dev, sourceID, WithReason("device_online"))
	if err != nil {
		e.logger.Warn("device.online: StartPathBSync failed",
			zap.String("device_id", evt.DeviceID.String()),
			zap.Error(err))
		return nil
	}
	if !used {
		e.logger.Debug("device.online: Path B unavailable (no MappingSet), skipping",
			zap.String("device_id", evt.DeviceID.String()))
		return nil
	}

	e.logger.Info("device.online: Path B sync initiated",
		zap.String("device_id", evt.DeviceID.String()),
		zap.String("serial_number", evt.SerialNumber),
	)
	return nil
}

// HandleFirmwareChanged 处理设备固件版本变化事件（T-0125）。
//
// 流程（设计方案 §3.2）：
//  1. Redis 串行锁 SetNX provision:firmware_handling:{deviceID} TTL=10min — 防设备升级期间
//     不稳定 swVersion 多次 Inform 引发并发交集。锁不主动释放，TTL 自然过期。
//  2. 预设 reason hint Set provision:syncreason:{deviceID}="firmware_changed" TTL=10min —
//     让 handleDataModelFileReceived 内 auto-sync Path B 完成时差异日志能读到正确 reason。
//  3. 调 modelUploadService.RequestModelUpload → 入队 Upload(FileType=11) → 异步回到
//     handleDataModelFileReceived → IntersectCPEModel 写新代次 discovered_param_mappings →
//     auto-sync Path B（reason 由 hint 决定）。
//  4. 若 RequestModelUpload 返 log.Status=DiscoveryCompleted（enable_filetype11=false 跳过）
//     或 err 不为 nil → 兜底直接调 StartPathBSync(WithReason("firmware_changed"))，
//     用 default 映射全量同步（旧 standardPath 不删除，漂移由 T-0127 差异日志记录）。
func (e *ProvisioningEngine) HandleFirmwareChanged(ctx context.Context, evt device.DeviceFirmwareChangedEvent) error {
	ctx, span := tracing.StartSpan(ctx, tracing.ProvisionTracerName, "Provision HandleFirmwareChanged",
		attribute.String("provision.device_sn", evt.SerialNumber),
		attribute.String("provision.device_id", evt.DeviceID.String()),
		attribute.String("provision.old_version", evt.OldVersion),
		attribute.String("provision.new_version", evt.NewVersion),
	)
	defer span.End()

	// 1. Redis 串行锁：防短时间内重复 firmware Inform 引发并发交集
	if e.redisClient != nil {
		lockKey := fmt.Sprintf("provision:firmware_handling:%s", evt.DeviceID.String())
		acquired, err := e.redisClient.SetNX(ctx, lockKey, "1", 10*time.Minute).Result()
		if err != nil {
			// Redis 失败不阻断：log warn 后继续推进（容忍 Redis 抖动）
			e.logger.Warn("firmware.changed lock SetNX failed",
				zap.String("device_id", evt.DeviceID.String()),
				zap.Error(err))
		} else if !acquired {
			e.logger.Debug("firmware handling already in progress, skip",
				zap.String("device_id", evt.DeviceID.String()),
				zap.String("serial_number", evt.SerialNumber))
			return nil
		}
	}

	// 2. 预设 reason hint：让 Upload 完成后 handleDataModelFileReceived auto-sync 也能读到正确 reason
	if e.redisClient != nil {
		reasonKey := fmt.Sprintf("provision:syncreason:%s", evt.DeviceID.String())
		if err := e.redisClient.Set(ctx, reasonKey, "firmware_changed", 10*time.Minute).Err(); err != nil {
			e.logger.Warn("firmware.changed reason hint write failed",
				zap.String("device_id", evt.DeviceID.String()),
				zap.Error(err))
		}
	}

	// 3. 查 device
	dev, err := e.deviceService.GetDevice(ctx, evt.DeviceID)
	if err != nil {
		e.logger.Warn("firmware.changed: lookup device failed",
			zap.String("device_id", evt.DeviceID.String()),
			zap.Error(err))
		return nil
	}
	if dev == nil {
		e.logger.Warn("firmware.changed: device not found, skipping",
			zap.String("device_id", evt.DeviceID.String()))
		return nil
	}

	// 4. 调 RequestModelUpload：log.Status=Discovering → Upload 真入队（handleDataModelFileReceived 会自动触发 Path B）；
	//    log.Status=Completed (enable_filetype11=false) 或 err → 走 step 5 兜底
	// SourceID 是裸 UUID 写入 device_tasks.source_id；reason="firmware_changed" 走 Redis 通道。
	sourceID := evt.DeviceID.String()
	var modelUploadEnqueued bool
	if e.modelUploadService != nil {
		log, uploadErr := e.modelUploadService.RequestModelUpload(ctx, dev, sourceID)
		if uploadErr != nil {
			e.logger.Warn("firmware.changed: RequestModelUpload failed, fallback to direct Path B",
				zap.String("device_id", evt.DeviceID.String()),
				zap.Error(uploadErr))
		} else if log != nil && log.Status == DiscoveryDiscovering {
			modelUploadEnqueued = true
			e.logger.Info("firmware.changed: model upload enqueued (Path B will be triggered after Upload completes)",
				zap.String("device_id", evt.DeviceID.String()),
				zap.String("discovery_id", log.ID.String()),
				zap.String("old_version", evt.OldVersion),
				zap.String("new_version", evt.NewVersion))
		} else if log != nil {
			e.logger.Info("firmware.changed: model upload skipped, will fallback to direct Path B",
				zap.String("device_id", evt.DeviceID.String()),
				zap.String("discovery_status", string(log.Status)),
				zap.String("discovery_message", log.ErrorMessage))
		}
	}

	// 5. 兜底 Path B：当 Upload 未真正入队（enable_filetype11=false / err / modelUploadService nil）时
	//    直接全量同步使用 default 映射；reason 标签已由 step 2 预设
	if !modelUploadEnqueued {
		if e.syncService == nil {
			e.logger.Debug("firmware.changed: syncService nil, skipping Path B fallback",
				zap.String("device_id", evt.DeviceID.String()))
			return nil
		}
		used, syncErr := e.syncService.StartPathBSync(ctx, dev, sourceID, WithReason("firmware_changed"))
		if syncErr != nil {
			e.logger.Warn("firmware.changed: direct Path B fallback failed",
				zap.String("device_id", evt.DeviceID.String()),
				zap.Error(syncErr))
			return nil
		}
		if !used {
			e.logger.Debug("firmware.changed: Path B fallback skipped (no MappingSet)",
				zap.String("device_id", evt.DeviceID.String()))
			return nil
		}
		e.logger.Info("firmware.changed: direct Path B fallback initiated",
			zap.String("device_id", evt.DeviceID.String()),
			zap.String("serial_number", evt.SerialNumber))
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
	e.bindDeviceProduct(ctx, dev)

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

	// Path B: AutoSync enabled + paramMapping available → sync via GPV.
	if e.config.AutoSync.Enabled && e.syncService != nil && e.syncService.PathBEnabled(ctx, dev) {
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

	used, err := e.syncService.StartPathBSync(ctx, dev, task.ID.String())
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
	if t == nil || t.SourceID == "" {
		return
	}
	if t.Status != task.TaskStatusFailed && t.Status != task.TaskStatusExpired {
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

	// After discovered mappings written, auto-sync via Path B if enabled.
	if e.syncService != nil && e.config.AutoSync.Enabled {
		// 反查 active provisioning_task 作为 sourceID，让 task.failed 能联动它（D2）
		var sourceID string
		if pt, _ := e.taskRepo.GetByDeviceID(ctx, dev.ID); pt != nil {
			sourceID = pt.ID.String()
		}
		if used, syncErr := e.syncService.StartPathBSync(ctx, dev, sourceID); syncErr != nil {
			e.logger.Warn("path-b auto-sync after model upload failed",
				zap.Error(syncErr),
				zap.String("device_sn", payload.DeviceSN),
			)
		} else if used {
			e.logger.Info("path-b auto-sync initiated after model upload",
				zap.String("device_sn", payload.DeviceSN),
			)
		} else {
			e.logger.Info("path-b auto-sync skipped (mapping unavailable)",
				zap.String("device_sn", payload.DeviceSN),
			)
		}
	}

	return nil
}

// gpvResponsePayload is the structured event payload for GetParameterValuesResponse.
type gpvResponsePayload struct {
	DeviceSN        string                       `json:"device_sn"`
	Method          string                       `json:"method"`
	CommandKey      string                       `json:"command_key,omitempty"`
	ParameterValues []tr069.ParameterValueStruct `json:"parameter_values"`
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

	e.logger.Info("received GPV response",
		zap.String("device_sn", payload.DeviceSN),
		zap.Int("parameter_count", len(payload.ParameterValues)),
	)

	if len(payload.ParameterValues) == 0 {
		return nil
	}

	dev, err := e.deviceService.GetBySerialNumber(ctx, payload.DeviceSN)
	if err != nil {
		e.logger.Error("find device for GPV response", zap.Error(err), zap.String("device_sn", payload.DeviceSN))
		return nil
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
		return nil
	} else if used {
		e.logger.Info("path-b parameter values saved",
			zap.String("device_sn", payload.DeviceSN),
			zap.Int("count", len(payload.ParameterValues)),
		)
		return nil
	}

	if err := e.syncService.HandleSyncResult(ctx, dev, payload.ParameterValues); err != nil {
		e.logger.Error("save parameter values", zap.Error(err), zap.String("device_sn", payload.DeviceSN))
		return nil
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
	if interval < 30*time.Second {
		interval = 30 * time.Second
	}

	e.logger.Info("provisioning task reaper started",
		zap.Duration("timeout", timeout),
		zap.Duration("interval", interval))

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			ctx := context.Background()
			n, err := e.taskRepo.FailStale(ctx, timeout)
			if err != nil {
				e.logger.Error("provisioning task reaper: fail stale tasks", zap.Error(err))
				continue
			}
			if n > 0 {
				e.logger.Warn("provisioning task reaper: timed out stale tasks",
					zap.Int64("count", n),
					zap.Duration("timeout", timeout))
			}
		}
	}()
}

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
	"github.com/omcgo/omcgo/internal/task"
	"github.com/omcgo/omcgo/pkg/tr069"
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
	modelUploadService *ModelUploadService
	syncService        *SyncService
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
	_, err := bus.QueueSubscribe(event.SubjectDeviceRegistered, "provisioning", func(ctx context.Context, evt event.Event) error {
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
	})
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

	// 3. Try matching template first (Path A).
	if err := e.transitionTask(ctx, task, StateMatching); err != nil {
		return e.failTask(ctx, task, fmt.Errorf("transition to matching: %w", err))
	}

	tmpl, err := e.templateService.Match(ctx, dev)
	if err != nil {
		return e.failTask(ctx, task, fmt.Errorf("match template: %w", err))
	}

	if tmpl != nil && e.config.AutoConfigure {
		return e.handleTemplateProvisioning(ctx, task, dev, tmpl, evt.SerialNumber)
	}
	if tmpl != nil {
		e.logger.Info("template matched but auto_configure disabled, skipping Path A",
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

	if err := EnqueueSteps(ctx, deviceSN, steps, e.taskSvc); err != nil {
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

	_, err := e.modelUploadService.RequestModelUpload(ctx, dev)
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

	used, err := e.syncService.StartPathBSync(ctx, dev)
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
		if used, syncErr := e.syncService.StartPathBSync(ctx, dev); syncErr != nil {
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

	if used, err := e.syncService.HandleSyncResultPathB(ctx, dev, payload.ParameterValues); err != nil {
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

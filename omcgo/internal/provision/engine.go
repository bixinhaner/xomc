package provision

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/acs/cmdqueue"
	"github.com/omcgo/omcgo/internal/core/appconfig"
	"github.com/omcgo/omcgo/internal/core/carrier"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/tracing"
	"github.com/omcgo/omcgo/internal/config/datamodel"
	"github.com/omcgo/omcgo/internal/config/template"
	"github.com/omcgo/omcgo/internal/device"
	"github.com/omcgo/omcgo/pkg/tr069"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
)

// ProvisioningEngine orchestrates the full auto-provisioning workflow.
type ProvisioningEngine struct {
	taskRepo         ProvisioningTaskRepository
	deviceService    *device.DeviceService
	dmRegistry       *datamodel.DataModelRegistry
	templateService  *template.ConfigTemplateService
	carrierRegistry  *carrier.CarrierRegistry
	cmdQueue         cmdqueue.CommandQueue
	eventBus         event.EventBus
	discoveryService *DiscoveryService
	syncService      *SyncService
	config           appconfig.ProvisionConfig
	logger           *zap.Logger
}

// NewProvisioningEngine creates a new ProvisioningEngine with all dependencies.
func NewProvisioningEngine(
	taskRepo ProvisioningTaskRepository,
	deviceService *device.DeviceService,
	dmRegistry *datamodel.DataModelRegistry,
	templateService *template.ConfigTemplateService,
	carrierRegistry *carrier.CarrierRegistry,
	cmdQueue cmdqueue.CommandQueue,
	eventBus event.EventBus,
	config appconfig.ProvisionConfig,
	logger *zap.Logger,
) *ProvisioningEngine {
	return &ProvisioningEngine{
		taskRepo:        taskRepo,
		deviceService:   deviceService,
		dmRegistry:      dmRegistry,
		templateService: templateService,
		carrierRegistry: carrierRegistry,
		cmdQueue:        cmdQueue,
		eventBus:        eventBus,
		config:          config,
		logger:          logger,
	}
}

// SetDiscoveryService sets the discovery service for auto-discovery support.
func (e *ProvisioningEngine) SetDiscoveryService(svc *DiscoveryService) {
	e.discoveryService = svc
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

// Subscribe registers the engine to listen for bootstrap events via queue group.
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

	// Subscribe to GPN response for auto-discovery processing.
	if _, err := bus.QueueSubscribe(event.SubjectCommandGetNamesResponse, "provision-gpn", func(ctx context.Context, evt event.Event) error {
		return e.handleGPNResponse(ctx, evt)
	}); err != nil {
		e.logger.Warn("failed to subscribe to GPN response", zap.Error(err))
	} else {
		e.logger.Info("provisioning engine subscribed to GPN response events")
	}

	// Subscribe to GPV response for parameter sync processing.
	if _, err := bus.QueueSubscribe(event.SubjectCommandGetParamsResponse, "provision-gpv", func(ctx context.Context, evt event.Event) error {
		return e.handleGPVResponse(ctx, evt)
	}); err != nil {
		e.logger.Warn("failed to subscribe to GPV response", zap.Error(err))
	} else {
		e.logger.Info("provisioning engine subscribed to GPV response events")
	}

	return nil
}

// HandleBootstrap processes a device bootstrap event and initiates the provisioning workflow.
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
	// This handles the case where a device went offline mid-provisioning and reconnected.
	// The stale task would otherwise block new provisioning until the reaper cleans it up.
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
		// Clean up residual discovery Redis state (pending counter + accumulated params).
		if e.discoveryService != nil {
			e.discoveryService.CleanupState(ctx, evt.SerialNumber)
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

	// 2. Identify device — resolve data model.
	if err := e.transitionTask(ctx, task, StateIdentifying); err != nil {
		return e.failTask(ctx, task, fmt.Errorf("transition to identifying: %w", err))
	}

	dev, err := e.deviceService.GetDevice(ctx, evt.DeviceID)
	if err != nil {
		return e.failTask(ctx, task, fmt.Errorf("get device: %w", err))
	}

	dm, err := e.dmRegistry.ResolveForDevice(ctx, dev)
	if err != nil {
		return e.failTask(ctx, task, fmt.Errorf("resolve data model: %w", err))
	}
	if dm != nil {
		e.logger.Info("data model resolved",
			zap.String("device_sn", evt.SerialNumber),
			zap.String("model_id", dm.ID.String()),
			zap.String("scope", string(dm.Scope)),
		)
	}

	// Four-path branching based on feature toggles and data model availability.
	//
	// Path A: Has matching template → classic provisioning flow (configuring).
	// Path B: Has DataModel + auto_sync enabled → sync parameter values only.
	// Path C: No DataModel + auto_discovery enabled → discover parameter tree first.
	// Path D: All switches off or no match → fail task (backward compatible).

	// 3. Try matching template first (Path A).
	if err := e.transitionTask(ctx, task, StateMatching); err != nil {
		return e.failTask(ctx, task, fmt.Errorf("transition to matching: %w", err))
	}

	tmpl, err := e.templateService.Match(ctx, dev)
	if err != nil {
		return e.failTask(ctx, task, fmt.Errorf("match template: %w", err))
	}

	if tmpl != nil && e.config.AutoConfigure {
		// Path A: classic provisioning with template (requires parameter path mapping).
		return e.handleTemplateProvisioning(ctx, task, dev, tmpl, evt.SerialNumber)
	}
	if tmpl != nil {
		e.logger.Info("template matched but auto_configure disabled, skipping Path A",
			zap.String("device_sn", evt.SerialNumber),
			zap.String("template_id", tmpl.ID.String()),
		)
	}

	// No template matched. Check auto-sync and auto-discovery paths.

	// Path B: DataModel exists + auto_sync enabled → sync parameters.
	if dm != nil && e.config.AutoSync.Enabled && e.syncService != nil {
		return e.handleAutoSync(ctx, task, dev, dm)
	}

	// Path C: No DataModel + auto_discovery enabled → discover parameter tree.
	if dm == nil && e.config.AutoDiscovery.Enabled && e.discoveryService != nil {
		return e.handleAutoDiscovery(ctx, task, dev)
	}

	// Path D: No template, no applicable feature toggle → fail.
	e.logger.Warn("no matching template found, provisioning skipped",
		zap.String("device_sn", evt.SerialNumber),
	)
	return e.failTask(ctx, task, fmt.Errorf("no matching template for device %s", evt.SerialNumber))
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

	steps, err := BuildProvisioningSteps(tmpl)
	if err != nil {
		return e.failTask(ctx, task, fmt.Errorf("build provisioning steps: %w", err))
	}
	task.TotalSteps = len(steps)
	if err := e.taskRepo.Update(ctx, task); err != nil {
		return e.failTask(ctx, task, fmt.Errorf("update task steps: %w", err))
	}

	if err := EnqueueSteps(ctx, deviceSN, steps, e.cmdQueue); err != nil {
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

// handleAutoDiscovery initiates parameter tree discovery (Path C).
func (e *ProvisioningEngine) handleAutoDiscovery(ctx context.Context, task *ProvisioningTask,
	dev *model.Device) error {

	if err := e.transitionTask(ctx, task, StateDiscovering); err != nil {
		return e.failTask(ctx, task, fmt.Errorf("transition to discovering: %w", err))
	}

	_, err := e.discoveryService.StartDiscovery(ctx, dev)
	if err != nil {
		return e.failTask(ctx, task, fmt.Errorf("start discovery: %w", err))
	}

	e.logger.Info("auto-discovery initiated",
		zap.String("device_sn", dev.SerialNumber),
		zap.String("task_id", task.ID.String()),
	)

	return nil
}

// handleAutoSync initiates parameter value synchronization (Path B).
func (e *ProvisioningEngine) handleAutoSync(ctx context.Context, task *ProvisioningTask,
	dev *model.Device, dm *datamodel.DataModel) error {

	if err := e.transitionTask(ctx, task, StateSyncing); err != nil {
		return e.failTask(ctx, task, fmt.Errorf("transition to syncing: %w", err))
	}

	// Extract parameter paths from the data model's parameter tree.
	paramPaths, err := extractPathsFromParameterTree(dm.ParameterTree)
	if err != nil {
		return e.failTask(ctx, task, fmt.Errorf("extract parameter paths: %w", err))
	}

	if err := e.syncService.StartSync(ctx, dev, paramPaths); err != nil {
		return e.failTask(ctx, task, fmt.Errorf("start sync: %w", err))
	}

	e.logger.Info("auto-sync initiated",
		zap.String("device_sn", dev.SerialNumber),
		zap.String("task_id", task.ID.String()),
		zap.Int("param_count", len(paramPaths)),
	)

	return nil
}

// extractPathsFromParameterTree extracts parameter paths from a DataModel's parameter tree JSON.
func extractPathsFromParameterTree(tree json.RawMessage) ([]string, error) {
	var params []struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(tree, &params); err != nil {
		return nil, fmt.Errorf("unmarshal parameter tree: %w", err)
	}

	paths := make([]string, 0, len(params))
	for _, p := range params {
		if p.Path != "" {
			paths = append(paths, p.Path)
		}
	}
	return paths, nil
}

// HandleRPCResult processes an RPC response and advances the provisioning state.
func (e *ProvisioningEngine) HandleRPCResult(ctx context.Context, deviceSN string, method string, success bool, errMsg string) error {
	dev, err := e.deviceService.GetBySerialNumber(ctx, deviceSN)
	if err != nil {
		return fmt.Errorf("find device by SN %s: %w", deviceSN, err)
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

	return cause
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

// gpnResponsePayload is the structured event payload for GetParameterNamesResponse.
// ACS handler parses the SOAP XML and sends structured data to avoid NATS message size limits.
type gpnResponsePayload struct {
	DeviceSN       string                      `json:"device_sn"`
	Method         string                      `json:"method"`
	ParameterInfos []tr069.ParameterInfoStruct `json:"parameter_infos"`
}

// gpvResponsePayload is the structured event payload for GetParameterValuesResponse.
type gpvResponsePayload struct {
	DeviceSN        string                       `json:"device_sn"`
	Method          string                       `json:"method"`
	ParameterValues []tr069.ParameterValueStruct `json:"parameter_values"`
}

// handleGPNResponse processes GetParameterNamesResponse events from ACS.
func (e *ProvisioningEngine) handleGPNResponse(ctx context.Context, evt event.Event) error {
	var payload gpnResponsePayload
	if err := evt.DecodePayload(&payload); err != nil {
		e.logger.Error("decode GPN response event", zap.Error(err))
		return nil
	}

	e.logger.Info("received GPN response",
		zap.String("device_sn", payload.DeviceSN),
		zap.Int("parameter_count", len(payload.ParameterInfos)),
	)

	// Note: empty GPN responses (parameter_count=0) must NOT be skipped.
	// They still need to flow through HandleLevelGPNResponse to decrement
	// the pending counter, otherwise discovery stalls permanently.

	// Look up the device.
	dev, err := e.deviceService.GetBySerialNumber(ctx, payload.DeviceSN)
	if err != nil {
		e.logger.Error("find device for GPN response", zap.Error(err), zap.String("device_sn", payload.DeviceSN))
		return nil
	}

	// Call discovery service to process this level's GPN response.
	// HandleLevelGPNResponse returns a non-nil DataModel only when all levels are done.
	if e.discoveryService != nil {
		dm, err := e.discoveryService.HandleLevelGPNResponse(ctx, dev, payload.ParameterInfos)
		if err != nil {
			e.logger.Error("handle discovery result", zap.Error(err), zap.String("device_sn", payload.DeviceSN))
			return nil
		}
		if dm != nil {
			e.logger.Info("data model created/found from GPN response",
				zap.String("device_sn", payload.DeviceSN),
				zap.String("model_id", dm.ID.String()),
			)

			// After discovery, auto-sync parameters if sync service is available.
			if e.syncService != nil && e.config.AutoSync.Enabled {
				paramPaths, err := extractPathsFromParameterTree(dm.ParameterTree)
				if err == nil && len(paramPaths) > 0 {
					if syncErr := e.syncService.StartSync(ctx, dev, paramPaths); syncErr != nil {
						e.logger.Warn("auto-sync after discovery failed",
							zap.Error(syncErr),
							zap.String("device_sn", payload.DeviceSN),
						)
					} else {
						e.logger.Info("auto-sync initiated after discovery",
							zap.String("device_sn", payload.DeviceSN),
							zap.Int("param_count", len(paramPaths)),
						)
					}
				}
			}
		}
	}

	return nil
}

// handleGPVResponse processes GetParameterValuesResponse events from ACS.
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

	// Look up the device.
	dev, err := e.deviceService.GetBySerialNumber(ctx, payload.DeviceSN)
	if err != nil {
		e.logger.Error("find device for GPV response", zap.Error(err), zap.String("device_sn", payload.DeviceSN))
		return nil
	}

	// Save parameter values to database via sync service.
	if e.syncService != nil {
		if err := e.syncService.HandleSyncResult(ctx, dev, payload.ParameterValues); err != nil {
			e.logger.Error("save parameter values", zap.Error(err), zap.String("device_sn", payload.DeviceSN))
			return nil
		}
		e.logger.Info("parameter values saved",
			zap.String("device_sn", payload.DeviceSN),
			zap.Int("count", len(payload.ParameterValues)),
		)
	}

	return nil
}

// StartTaskReaper launches a background goroutine that periodically fails
// provisioning tasks stuck in non-terminal states beyond the configured timeout.
// This prevents stale tasks from permanently blocking new provisioning for a device.
func (e *ProvisioningEngine) StartTaskReaper() {
	timeout := e.config.TaskTimeout
	if timeout <= 0 {
		timeout = 15 * time.Minute
	}
	// Scan interval = half the timeout, so stale tasks are caught reasonably fast.
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

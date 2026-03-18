package provision

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/acs/cmdqueue"
	"github.com/omcgo/omcgo/internal/core/carrier"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/config/datamodel"
	"github.com/omcgo/omcgo/internal/config/template"
	"github.com/omcgo/omcgo/internal/device"
	"go.uber.org/zap"
)

// ProvisioningEngine orchestrates the full auto-provisioning workflow.
type ProvisioningEngine struct {
	taskRepo        ProvisioningTaskRepository
	deviceService   *device.DeviceService
	dmRegistry      *datamodel.DataModelRegistry
	templateService *template.ConfigTemplateService
	carrierRegistry *carrier.CarrierRegistry
	cmdQueue        cmdqueue.CommandQueue
	eventBus        event.EventBus
	logger          *zap.Logger
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
		logger:          logger,
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
}

// Subscribe registers the engine to listen for bootstrap events via queue group.
func (e *ProvisioningEngine) Subscribe(bus event.EventBus) error {
	_, err := bus.QueueSubscribe(event.SubjectDeviceRegistered, "provisioning", func(ctx context.Context, evt event.Event) error {
		var bsEvt bootstrapEvent
		if err := evt.DecodePayload(&bsEvt); err != nil {
			e.logger.Error("decode bootstrap event", zap.Error(err))
			return err
		}
		return e.HandleBootstrap(ctx, bsEvt)
	})
	return err
}

// HandleBootstrap processes a device bootstrap event and initiates the provisioning workflow.
func (e *ProvisioningEngine) HandleBootstrap(ctx context.Context, evt bootstrapEvent) error {
	e.logger.Info("provisioning started for device",
		zap.String("device_sn", evt.SerialNumber),
		zap.String("device_id", evt.DeviceID.String()),
	)

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

	// 3. Match template.
	if err := e.transitionTask(ctx, task, StateMatching); err != nil {
		return e.failTask(ctx, task, fmt.Errorf("transition to matching: %w", err))
	}

	tmpl, err := e.templateService.Match(ctx, dev)
	if err != nil {
		return e.failTask(ctx, task, fmt.Errorf("match template: %w", err))
	}
	if tmpl == nil {
		e.logger.Warn("no matching template found, provisioning skipped",
			zap.String("device_sn", evt.SerialNumber),
		)
		return e.failTask(ctx, task, fmt.Errorf("no matching template for device %s", evt.SerialNumber))
	}

	task.TemplateID = &tmpl.ID
	if err := e.taskRepo.Update(ctx, task); err != nil {
		return e.failTask(ctx, task, fmt.Errorf("update task template: %w", err))
	}

	// 4. Build and enqueue provisioning steps.
	if err := e.transitionTask(ctx, task, StateConfiguring); err != nil {
		return e.failTask(ctx, task, fmt.Errorf("transition to configuring: %w", err))
	}

	steps := BuildProvisioningSteps(tmpl)
	task.TotalSteps = len(steps)
	if err := e.taskRepo.Update(ctx, task); err != nil {
		return e.failTask(ctx, task, fmt.Errorf("update task steps: %w", err))
	}

	if err := EnqueueSteps(ctx, evt.SerialNumber, steps, e.cmdQueue); err != nil {
		return e.failTask(ctx, task, fmt.Errorf("enqueue steps: %w", err))
	}

	e.logger.Info("provisioning steps enqueued",
		zap.String("device_sn", evt.SerialNumber),
		zap.Int("steps", len(steps)),
	)

	// 5. Transition to verifying — we'll wait for RPC results via HandleRPCResult.
	if err := e.transitionTask(ctx, task, StateVerifying); err != nil {
		return e.failTask(ctx, task, fmt.Errorf("transition to verifying: %w", err))
	}

	return nil
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

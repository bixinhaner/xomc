package backup

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/acs/connreq"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/device"
	devtask "github.com/omcgo/omcgo/internal/task"
)

// BackupExecutor subscribes to backup.task.created events and executes
// backup tasks by pushing Upload commands to the unified device task queue.
//
// T-0073 Phase 1 additions: when configured with a PolicyService and metrics,
// the executor publishes alarm.raised on failure paths according to the
// active BackupPolicy.AlertOnFailure flag. Both fields are optional (nil-safe)
// to keep the constructor signature backward-compatible.
type BackupExecutor struct {
	taskRepo      TaskRepository
	deviceRepo    device.DeviceRepository
	taskSvc       devtask.Enqueuer
	connReq       *connreq.Client
	eventBus      event.EventBus
	policyService PolicyGetter   // optional (T-0073)
	metrics       *PolicyMetrics // optional (T-0073)
	logger        *zap.Logger
}

// NewBackupExecutor creates a new BackupExecutor.
func NewBackupExecutor(
	taskRepo TaskRepository,
	deviceRepo device.DeviceRepository,
	taskSvc devtask.Enqueuer,
	connReq *connreq.Client,
	eventBus event.EventBus,
	logger *zap.Logger,
) *BackupExecutor {
	return &BackupExecutor{
		taskRepo:   taskRepo,
		deviceRepo: deviceRepo,
		taskSvc:    taskSvc,
		connReq:    connReq,
		eventBus:   eventBus,
		logger:     logger.Named("backup-executor"),
	}
}

// SetPolicyEnforcement wires PolicyService + metrics post-construction so the
// failure path can publish alarm.raised when AlertOnFailure=true. Either
// argument may be nil to disable that side-effect (default is disabled).
func (e *BackupExecutor) SetPolicyEnforcement(policyService PolicyGetter, metrics *PolicyMetrics) {
	e.policyService = policyService
	e.metrics = metrics
}

// Subscribe registers the executor to listen for backup task created events.
func (e *BackupExecutor) Subscribe(bus event.EventBus) error {
	_, err := bus.QueueSubscribe(
		event.SubjectBackupTaskCreated,
		"backup-executors",
		e.handleTaskCreated,
	)
	if err != nil {
		return fmt.Errorf("subscribe backup.task.created: %w", err)
	}
	e.logger.Info("backup executor subscribed",
		zap.String("subject", event.SubjectBackupTaskCreated))
	return nil
}

type backupTaskPayload struct {
	TaskID string `json:"task_id"`
}

func (e *BackupExecutor) handleTaskCreated(ctx context.Context, evt event.Event) error {
	var payload backupTaskPayload
	if err := evt.DecodePayload(&payload); err != nil {
		return fmt.Errorf("decode backup task payload: %w", err)
	}

	e.logger.Info("executing backup task", zap.String("task_id", payload.TaskID))

	// Get task
	taskID, err := uuid.Parse(payload.TaskID)
	if err != nil {
		return fmt.Errorf("parse task_id: %w", err)
	}

	task, err := e.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return fmt.Errorf("get backup task: %w", err)
	}

	if task.Status != TaskPending {
		e.logger.Info("backup task not pending, skipping",
			zap.String("task_id", payload.TaskID),
			zap.String("status", string(task.Status)))
		return nil
	}

	// Update to running
	now := time.Now()
	task.Status = TaskRunning
	task.StartedAt = &now
	if err := e.taskRepo.Update(ctx, task); err != nil {
		return fmt.Errorf("update backup task to running: %w", err)
	}

	// Process each target device
	total := len(task.TargetIDs)
	successCount := 0

	for i, targetSN := range task.TargetIDs {
		dev, err := e.deviceRepo.GetBySerialNumber(ctx, targetSN)
		if err != nil {
			e.logger.Warn("device lookup failed for backup",
				zap.String("device_sn", targetSN),
				zap.Error(err))
			continue
		}
		if dev == nil {
			e.logger.Warn("device not found for backup, skipping",
				zap.String("device_sn", targetSN))
			continue
		}

		// Build Upload command (TR-069 FileType "2" = VendorConfigurationFile)
		paramsJSON, marshalErr := json.Marshal(map[string]interface{}{
			"file_type": "2",
		})
		if marshalErr != nil {
			e.logger.Warn("marshal upload params", zap.String("device_sn", targetSN), zap.Error(marshalErr))
			continue
		}
		if _, err := e.taskSvc.CreateTask(ctx, &devtask.CreateTaskRequest{
			DeviceSN: dev.SerialNumber,
			Method:   "Upload",
			Params:   paramsJSON,
			Source:   devtask.TaskSourceSystem,
		}); err != nil {
			e.logger.Warn("push upload command",
				zap.String("device_sn", dev.SerialNumber),
				zap.Error(err))
			continue
		}

		// Wake device via Connection Request
		if dev.ConnectionRequestURL != "" {
			if err := e.connReq.Send(ctx, dev.SerialNumber, dev.ConnectionRequestURL); err != nil {
				e.logger.Warn("send connection request",
					zap.String("device_sn", dev.SerialNumber),
					zap.Error(err))
			}
		}

		successCount++

		// Update progress
		task.Progress = (i + 1) * 100 / total
		if updateErr := e.taskRepo.Update(ctx, task); updateErr != nil {
			e.logger.Warn("update backup task progress", zap.Error(updateErr))
		}
	}

	// Complete task
	completedAt := time.Now()
	if successCount == 0 && total > 0 {
		task.Status = TaskFailed
		errMsg := "no devices were successfully queued"
		task.ErrorMessage = &errMsg
	} else {
		task.Status = TaskCompleted
		task.Progress = 100
	}
	task.CompletedAt = &completedAt
	if err := e.taskRepo.Update(ctx, task); err != nil {
		return fmt.Errorf("update backup task completion: %w", err)
	}

	// T-0073 Phase 1: opt-in failure alarm publish.
	// Only fire on TaskFailed AND when BackupPolicy.AlertOnFailure=true.
	// Best-effort: alarm publish errors are logged, never blocking.
	if task.Status == TaskFailed && e.policyService != nil {
		if pubErr := PublishFailureAlarm(ctx, e.policyService, e.eventBus, e.metrics, task); pubErr != nil {
			e.logger.Warn("publish backup failure alarm",
				zap.String("task_id", payload.TaskID),
				zap.Error(pubErr))
		}
	}

	e.logger.Info("backup task completed",
		zap.String("task_id", payload.TaskID),
		zap.String("status", string(task.Status)),
		zap.Int("success", successCount),
		zap.Int("total", total))

	// Publish done event
	doneEvt, err := event.NewEvent(event.SubjectBackupTaskDone, map[string]interface{}{
		"task_id": payload.TaskID,
		"status":  string(task.Status),
	})
	if err == nil {
		if pubErr := e.eventBus.Publish(ctx, event.SubjectBackupTaskDone, doneEvt); pubErr != nil {
			e.logger.Warn("publish backup.task.done event", zap.Error(pubErr))
		}
	}

	return nil
}

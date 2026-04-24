package software

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/acs/connreq"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/device"
	devtask "github.com/omcgo/omcgo/internal/task"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// UpgradeExecutor handles the event-driven upgrade lifecycle for individual devices.
// It follows the same pattern as provision/Engine: cmdqueue.Push → wait for EventBus callbacks.
type UpgradeExecutor struct {
	taskRepo     TaskRepository
	subTaskRepo  SubTaskRepository
	deviceRepo   device.DeviceRepository
	firmwareRepo FirmwareRepository
	cmdQueue     devtask.Enqueuer
	connReq      *connreq.Client
	redis        redis.UniversalClient
	eventBus     event.EventBus
	logger       *zap.Logger
}

// NewUpgradeExecutor creates a new UpgradeExecutor.
func NewUpgradeExecutor(
	taskRepo TaskRepository,
	subTaskRepo SubTaskRepository,
	deviceRepo device.DeviceRepository,
	firmwareRepo FirmwareRepository,
	cmdQueue devtask.Enqueuer,
	connReq *connreq.Client,
	redisClient redis.UniversalClient,
	eventBus event.EventBus,
	logger *zap.Logger,
) *UpgradeExecutor {
	return &UpgradeExecutor{
		taskRepo:     taskRepo,
		subTaskRepo:  subTaskRepo,
		deviceRepo:   deviceRepo,
		firmwareRepo: firmwareRepo,
		cmdQueue:     cmdQueue,
		connReq:      connReq,
		redis:        redisClient,
		eventBus:     eventBus,
		logger:       logger.Named("upgrade-executor"),
	}
}

// ExecuteOne runs the upgrade flow for a single sub-task:
// conflict check (Redis SETNX) → record ori_version → push Download Command → update status.
func (e *UpgradeExecutor) ExecuteOne(ctx context.Context, subTask *UpgradeSubTask, fw *FirmwareVersion) {
	dev, err := e.deviceRepo.GetByID(ctx, subTask.DeviceID)
	if err != nil {
		e.failSubTask(ctx, subTask, fmt.Sprintf("device not found: %v", err))
		return
	}

	// Acquire device-level lock via Redis SETNX
	acquired, err := e.acquireDeviceLock(ctx, dev.SerialNumber, subTask.ID)
	if err != nil {
		e.logger.Warn("acquire device lock failed, proceeding without lock",
			zap.String("device_sn", dev.SerialNumber), zap.Error(err))
		// Continue without lock — degraded but functional
	} else if !acquired {
		e.failSubTask(ctx, subTask, "device already has an active upgrade")
		return
	}

	// Record device SN and original version
	subTask.DeviceSN = dev.SerialNumber
	subTask.OriVersion = dev.FirmwareVersion
	if err := e.subTaskRepo.Update(ctx, subTask); err != nil {
		e.logger.Error("update sub-task device info", zap.Error(err))
	}

	// Push Download command
	downloadURL := fmt.Sprintf("/firmware/%s/download", fw.ID.String())
	paramsJSON, err := json.Marshal(map[string]interface{}{
		"url":       downloadURL,
		"file_type": "1", // firmware
		"file_size": fw.FileSize,
		"file_name": fw.FileName,
		"target_filename": fw.FileName,
		"command_key": subTask.ID.String(),
	})
	if err != nil {
		e.releaseDeviceLock(ctx, dev.SerialNumber)
		e.failSubTask(ctx, subTask, fmt.Sprintf("marshal download params: %v", err))
		return
	}

	_, err = e.cmdQueue.CreateTask(ctx, &devtask.CreateTaskRequest{
		DeviceSN:   dev.SerialNumber,
		Method:     "Download",
		Params:     paramsJSON,
		Source:     devtask.TaskSourceSystem,
		CommandKey: subTask.ID.String(),
	})
	if err != nil {
		e.releaseDeviceLock(ctx, dev.SerialNumber)
		e.failSubTask(ctx, subTask, fmt.Sprintf("push download command: %v", err))
		return
	}

	// Store command key for reverse lookup
	subTask.CommandKey = subTask.ID.String()
	if err := e.subTaskRepo.Update(ctx, subTask); err != nil {
		e.logger.Error("update sub-task command_key", zap.Error(err))
	}

	// Send Connection Request to wake device
	if dev.ConnectionRequestURL != "" {
		if err := e.connReq.Send(ctx, dev.SerialNumber, dev.ConnectionRequestURL); err != nil {
			e.logger.Warn("send connection request",
				zap.String("device_sn", dev.SerialNumber), zap.Error(err))
		}
	}

	// Transition to downloading
	if err := e.subTaskRepo.UpdateStatus(ctx, subTask.ID, UpgradeDownloading, ""); err != nil {
		e.logger.Error("update sub-task to downloading", zap.Error(err))
	}

	e.logger.Info("upgrade download pushed",
		zap.String("sub_task_id", subTask.ID.String()),
		zap.String("device_sn", dev.SerialNumber),
		zap.String("firmware_version", fw.Version))
}

// HandleDownloadResponse handles command.download.response events.
// On fault → fail the sub-task. On success → wait for TransferComplete.
func (e *UpgradeExecutor) HandleDownloadResponse(ctx context.Context, evt event.Event) error {
	var payload struct {
		DeviceSN   string `json:"device_sn"`
		CommandKey string `json:"command_key"`
		FaultCode  int    `json:"fault_code"`
		FaultStr   string `json:"fault_string"`
	}
	if err := evt.DecodePayload(&payload); err != nil || payload.CommandKey == "" {
		return nil
	}

	subTask, err := e.subTaskRepo.GetByCommandKey(ctx, payload.CommandKey)
	if err != nil {
		return nil // Not our task
	}

	// Only handle downloading state
	if subTask.Status != UpgradeDownloading {
		return nil
	}

	if payload.FaultCode != 0 {
		e.failSubTask(ctx, subTask, fmt.Sprintf("download fault %d: %s", payload.FaultCode, payload.FaultStr))
		return nil
	}

	e.logger.Debug("download accepted, waiting for TransferComplete",
		zap.String("sub_task_id", subTask.ID.String()),
		zap.String("device_sn", payload.DeviceSN))
	return nil
}

// HandleRebootComplete handles device.inform.reboot_complete events.
// Used for rollback completion detection.
func (e *UpgradeExecutor) HandleRebootComplete(ctx context.Context, evt event.Event) error {
	var payload struct {
		DeviceSN string `json:"device_sn"`
	}
	if err := evt.DecodePayload(&payload); err != nil || payload.DeviceSN == "" {
		return nil
	}

	dev, err := e.deviceRepo.GetBySerialNumber(ctx, payload.DeviceSN)
	if err != nil || dev == nil {
		return nil
	}

	// Find active sub-task for this device that is in rebooting state
	subTask, err := e.subTaskRepo.GetActiveByDeviceID(ctx, dev.ID)
	if err != nil {
		return nil
	}

	if subTask.Status != UpgradeRebooting {
		return nil
	}

	// Reboot complete → verify version
	e.completeSubTask(ctx, subTask, dev.SerialNumber)

	e.logger.Info("reboot complete, upgrade finalized",
		zap.String("sub_task_id", subTask.ID.String()),
		zap.String("device_sn", payload.DeviceSN))
	return nil
}

// HandleDeviceOnline checks for pending upgrade wait keys when a device comes online.
func (e *UpgradeExecutor) HandleDeviceOnline(ctx context.Context, evt event.Event) error {
	var payload struct {
		DeviceSN string `json:"device_sn"`
	}
	if err := evt.DecodePayload(&payload); err != nil || payload.DeviceSN == "" {
		return nil
	}

	waitKey := fmt.Sprintf("software:upgrade:wait:%s", payload.DeviceSN)
	val, err := e.redis.Get(ctx, waitKey).Result()
	if err == redis.Nil {
		return nil // No pending upgrade
	}
	if err != nil {
		e.logger.Error("check upgrade wait key", zap.Error(err))
		return nil
	}

	// Delete the wait key
	e.redis.Del(ctx, waitKey)

	subTaskID, err := uuid.Parse(val)
	if err != nil {
		e.logger.Error("parse wait key sub-task ID", zap.String("value", val), zap.Error(err))
		return nil
	}

	subTask, err := e.subTaskRepo.GetByID(ctx, subTaskID)
	if err != nil {
		return nil
	}

	// Only resume suspended or pending tasks
	if subTask.Status != UpgradeSuspended && subTask.Status != UpgradePending {
		return nil
	}

	e.logger.Info("device online, resuming pending upgrade",
		zap.String("sub_task_id", subTask.ID.String()),
		zap.String("device_sn", payload.DeviceSN))

	// Get firmware and resume execution
	if subTask.FirmwareID != nil {
		fw, err := e.firmwareRepo.GetByID(ctx, *subTask.FirmwareID)
		if err != nil {
			e.failSubTask(ctx, subTask, fmt.Sprintf("get firmware for resume: %v", err))
			return nil
		}
		go e.ExecuteOne(context.Background(), subTask, fw)
	}

	return nil
}

// acquireDeviceLock tries to acquire a Redis SETNX lock for a device upgrade.
func (e *UpgradeExecutor) acquireDeviceLock(ctx context.Context, deviceSN string, taskID uuid.UUID) (bool, error) {
	key := fmt.Sprintf("software:upgrade:active:%s", deviceSN)
	ok, err := e.redis.SetNX(ctx, key, taskID.String(), time.Hour).Result()
	if err != nil {
		return false, fmt.Errorf("acquire device lock: %w", err)
	}
	return ok, nil
}

// releaseDeviceLock releases the Redis lock for a device upgrade.
func (e *UpgradeExecutor) releaseDeviceLock(ctx context.Context, deviceSN string) {
	key := fmt.Sprintf("software:upgrade:active:%s", deviceSN)
	e.redis.Del(ctx, key)
}

// failSubTask marks a sub-task as failed and updates the main task counts.
func (e *UpgradeExecutor) failSubTask(ctx context.Context, subTask *UpgradeSubTask, reason string) {
	if err := e.subTaskRepo.UpdateStatus(ctx, subTask.ID, UpgradeFailed, reason); err != nil {
		e.logger.Error("fail sub-task", zap.String("sub_task_id", subTask.ID.String()), zap.Error(err))
	}
	if subTask.DeviceSN != "" {
		e.releaseDeviceLock(ctx, subTask.DeviceSN)
	}
	if err := e.taskRepo.IncrementCounts(ctx, subTask.TaskID, 0, 1); err != nil {
		e.logger.Error("increment fail count", zap.Error(err))
	}
}

// completeSubTask marks a sub-task as completed and updates main task counts.
func (e *UpgradeExecutor) completeSubTask(ctx context.Context, subTask *UpgradeSubTask, deviceSN string) {
	if err := e.subTaskRepo.UpdateStatus(ctx, subTask.ID, UpgradeCompleted, ""); err != nil {
		e.logger.Error("complete sub-task", zap.String("sub_task_id", subTask.ID.String()), zap.Error(err))
	}
	e.releaseDeviceLock(ctx, deviceSN)
	if err := e.taskRepo.IncrementCounts(ctx, subTask.TaskID, 1, 0); err != nil {
		e.logger.Error("increment success count", zap.Error(err))
	}
}

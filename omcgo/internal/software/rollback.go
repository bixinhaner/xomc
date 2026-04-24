package software

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/omcgo/omcgo/internal/acs/connreq"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/device"
	devtask "github.com/omcgo/omcgo/internal/task"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// RollbackExecutor handles version rollback via SetParameterValues.
// Unlike upgrades which use Download RPC, rollback sets a parameter to trigger
// the device to revert to its previous firmware version.
type RollbackExecutor struct {
	taskRepo    TaskRepository
	subTaskRepo SubTaskRepository
	deviceRepo  device.DeviceRepository
	cmdQueue    devtask.Enqueuer
	connReq     *connreq.Client
	redis       redis.UniversalClient
	eventBus    event.EventBus
	logger      *zap.Logger
}

// NewRollbackExecutor creates a new RollbackExecutor.
func NewRollbackExecutor(
	taskRepo TaskRepository,
	subTaskRepo SubTaskRepository,
	deviceRepo device.DeviceRepository,
	cmdQueue devtask.Enqueuer,
	connReq *connreq.Client,
	redisClient redis.UniversalClient,
	eventBus event.EventBus,
	logger *zap.Logger,
) *RollbackExecutor {
	return &RollbackExecutor{
		taskRepo:    taskRepo,
		subTaskRepo: subTaskRepo,
		deviceRepo:  deviceRepo,
		cmdQueue:    cmdQueue,
		connReq:     connReq,
		redis:       redisClient,
		eventBus:    eventBus,
		logger:      logger.Named("rollback-executor"),
	}
}

// RollbackOne executes a rollback for a single device sub-task.
// Flow: acquire lock → push SetParameterValues command → wait for RebootComplete.
func (e *RollbackExecutor) RollbackOne(ctx context.Context, subTask *UpgradeSubTask, rollbackParamPath, rollbackParamValue string) {
	dev, err := e.deviceRepo.GetByID(ctx, subTask.DeviceID)
	if err != nil {
		e.failRollbackSubTask(ctx, subTask, fmt.Sprintf("device not found: %v", err))
		return
	}

	// Acquire device-level lock
	acquired, err := e.acquireDeviceLock(ctx, dev.SerialNumber, subTask.ID)
	if err != nil {
		e.logger.Warn("acquire device lock for rollback failed",
			zap.String("device_sn", dev.SerialNumber), zap.Error(err))
	} else if !acquired {
		e.failRollbackSubTask(ctx, subTask, "device already has an active task")
		return
	}

	// Record device info
	subTask.DeviceSN = dev.SerialNumber
	subTask.OriVersion = dev.FirmwareVersion
	if err := e.subTaskRepo.Update(ctx, subTask); err != nil {
		e.logger.Error("update rollback sub-task device info", zap.Error(err))
	}

	// Push SetParameterValues command
	paramsJSON, err := json.Marshal(map[string]interface{}{
		"parameter_path":  rollbackParamPath,
		"parameter_value": rollbackParamValue,
		"command_key":     subTask.ID.String(),
	})
	if err != nil {
		e.releaseDeviceLock(ctx, dev.SerialNumber)
		e.failRollbackSubTask(ctx, subTask, fmt.Sprintf("marshal set params: %v", err))
		return
	}

	_, err = e.cmdQueue.CreateTask(ctx, &devtask.CreateTaskRequest{
		DeviceSN:   dev.SerialNumber,
		Method:     "SetParameterValues",
		Params:     paramsJSON,
		Source:     devtask.TaskSourceSystem,
		CommandKey: subTask.ID.String(),
	})
	if err != nil {
		e.releaseDeviceLock(ctx, dev.SerialNumber)
		e.failRollbackSubTask(ctx, subTask, fmt.Sprintf("push set params command: %v", err))
		return
	}

	// Store command key
	subTask.CommandKey = subTask.ID.String()
	if err := e.subTaskRepo.Update(ctx, subTask); err != nil {
		e.logger.Error("update rollback sub-task command_key", zap.Error(err))
	}

	// Send Connection Request
	if dev.ConnectionRequestURL != "" {
		if err := e.connReq.Send(ctx, dev.SerialNumber, dev.ConnectionRequestURL); err != nil {
			e.logger.Warn("send connection request for rollback",
				zap.String("device_sn", dev.SerialNumber), zap.Error(err))
		}
	}

	// Transition to rebooting (waiting for device to reboot with old firmware)
	if err := e.subTaskRepo.UpdateStatus(ctx, subTask.ID, UpgradeRebooting, ""); err != nil {
		e.logger.Error("update rollback sub-task to rebooting", zap.Error(err))
	}

	e.logger.Info("rollback command pushed",
		zap.String("sub_task_id", subTask.ID.String()),
		zap.String("device_sn", dev.SerialNumber),
		zap.String("param_path", rollbackParamPath))
}

// acquireDeviceLock tries to acquire a Redis SETNX lock.
func (e *RollbackExecutor) acquireDeviceLock(ctx context.Context, deviceSN string, taskID interface{ String() string }) (bool, error) {
	key := fmt.Sprintf("software:upgrade:active:%s", deviceSN)
	ok, err := e.redis.SetNX(ctx, key, taskID.String(), 3600000000000).Result() // 1 hour
	if err != nil {
		return false, fmt.Errorf("acquire device lock: %w", err)
	}
	return ok, nil
}

// releaseDeviceLock releases the Redis lock.
func (e *RollbackExecutor) releaseDeviceLock(ctx context.Context, deviceSN string) {
	key := fmt.Sprintf("software:upgrade:active:%s", deviceSN)
	e.redis.Del(ctx, key)
}

// failRollbackSubTask marks a sub-task as failed.
func (e *RollbackExecutor) failRollbackSubTask(ctx context.Context, subTask *UpgradeSubTask, reason string) {
	if err := e.subTaskRepo.UpdateStatus(ctx, subTask.ID, UpgradeFailed, reason); err != nil {
		e.logger.Error("fail rollback sub-task", zap.String("sub_task_id", subTask.ID.String()), zap.Error(err))
	}
	if subTask.DeviceSN != "" {
		e.releaseDeviceLock(ctx, subTask.DeviceSN)
	}
	if err := e.taskRepo.IncrementCounts(ctx, subTask.TaskID, 0, 1); err != nil {
		e.logger.Error("increment rollback fail count", zap.Error(err))
	}
}

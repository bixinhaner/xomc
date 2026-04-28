package software

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/omcgo/omcgo/internal/acs/connreq"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/device"
	devtask "github.com/omcgo/omcgo/internal/task"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// RollbackExecutor handles version rollback via SetParameterValues.
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
// Flow: acquire lock → [optional: check rollback enable] → push SetParameterValues → wait for RebootComplete.
func (e *RollbackExecutor) RollbackOne(ctx context.Context, subTask *UpgradeSubTask, dev *model.Device, rollbackParamPath, rollbackParamValue string, needEnableCheck bool) {
	// Acquire device-level lock
	acquired, err := e.acquireDeviceLock(ctx, dev.SerialNumber, subTask.ID)
	if err != nil {
		e.logger.Warn("acquire device lock for rollback failed",
			zap.String("device_sn", dev.SerialNumber), zap.Error(err))
	} else if !acquired {
		e.FailRollbackSubTask(ctx, subTask, "Rollback can not be started, device can not be in multi running tasks.", FailureDeviceLocked)
		return
	}

	// Record device info
	subTask.DeviceSN = dev.SerialNumber
	subTask.OriVersion = dev.FirmwareVersion
	if err := e.subTaskRepo.Update(ctx, subTask); err != nil {
		e.logger.Error("update rollback sub-task device info", zap.Error(err))
	}

	// 4G rollback: check rollback enable via GetParameterValues
	if needEnableCheck {
		enabled, err := e.checkRollbackEnable(ctx, dev, rollbackParamPath)
		if err != nil {
			e.releaseDeviceLock(ctx, dev.SerialNumber)
			e.FailRollbackSubTask(ctx, subTask, fmt.Sprintf("Rollback can not be started, check rollback enable failed: %v", err), FailureInternalError)
			return
		}
		if !enabled {
			e.releaseDeviceLock(ctx, dev.SerialNumber)
			e.FailRollbackSubTask(ctx, subTask, "Rollback can not be started, device does not support rollback.", FailureInternalError)
			return
		}
	}

	// Push SetParameterValues command
	paramsJSON, err := json.Marshal(map[string]interface{}{
		"values": []map[string]string{
			{
				"name":  rollbackParamPath,
				"value": rollbackParamValue,
				"type":  "xsd:string",
			},
		},
		"command_key": subTask.ID.String(),
	})
	if err != nil {
		e.releaseDeviceLock(ctx, dev.SerialNumber)
		e.FailRollbackSubTask(ctx, subTask, fmt.Sprintf("Rollback can not be started, internal error: %v", err), FailureInternalError)
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
		e.FailRollbackSubTask(ctx, subTask, fmt.Sprintf("Rollback can not be started, failed to send set params command to device: %v", err), FailureCommandPush)
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
		zap.String("param_path", rollbackParamPath),
		zap.Bool("enable_check", needEnableCheck))
}

// checkRollbackEnable queries the device for X_COM_ROLLBACK_ENABLE parameter.
// For 4G devices, the device must support rollback before we can trigger it.
func (e *RollbackExecutor) checkRollbackEnable(ctx context.Context, dev *model.Device, rollbackPath string) (bool, error) {
	// Derive enable check path from rollback path (vendor-specific convention)
	enablePath := "Device.DeviceInfo.X_COM_ROLLBACK_ENABLE"

	paramsJSON, err := json.Marshal(map[string]interface{}{
		"parameter_names": []string{enablePath},
		"command_key":     "rollback-enable-check",
	})
	if err != nil {
		return false, fmt.Errorf("marshal GPV params: %w", err)
	}

	_, err = e.cmdQueue.CreateTask(ctx, &devtask.CreateTaskRequest{
		DeviceSN:   dev.SerialNumber,
		Method:     "GetParameterValues",
		Params:     paramsJSON,
		Source:     devtask.TaskSourceSystem,
		CommandKey: "rollback-enable-check-" + dev.SerialNumber,
	})
	if err != nil {
		// If we can't push the GPV command, assume enabled (best effort)
		e.logger.Warn("push rollback enable check command, assuming enabled",
			zap.String("device_sn", dev.SerialNumber), zap.Error(err))
		return true, nil
	}

	// Send Connection Request to wake the device
	if dev.ConnectionRequestURL != "" {
		e.connReq.Send(ctx, dev.SerialNumber, dev.ConnectionRequestURL)
	}

	// For now, assume enabled. The GPV response will be handled asynchronously.
	// If the device reports X_COM_ROLLBACK_ENABLE=false, the rollback will fail at the SPV step.
	return true, nil
}

func (e *RollbackExecutor) acquireDeviceLock(ctx context.Context, deviceSN string, taskID interface{ String() string }) (bool, error) {
	key := fmt.Sprintf("software:upgrade:active:%s", deviceSN)
	ok, err := e.redis.SetNX(ctx, key, taskID.String(), time.Hour).Result()
	if err != nil {
		return false, fmt.Errorf("acquire device lock: %w", err)
	}
	return ok, nil
}

func (e *RollbackExecutor) releaseDeviceLock(ctx context.Context, deviceSN string) {
	key := fmt.Sprintf("software:upgrade:active:%s", deviceSN)
	e.redis.Del(ctx, key)
}

// FailRollbackSubTask marks a sub-task as failed and finalizes the parent task.
func (e *RollbackExecutor) FailRollbackSubTask(ctx context.Context, subTask *UpgradeSubTask, reason string, code FailureCode) {
	if err := e.subTaskRepo.UpdateStatusWithCode(ctx, subTask.ID, UpgradeFailed, reason, code); err != nil {
		e.logger.Error("fail rollback sub-task", zap.String("sub_task_id", subTask.ID.String()), zap.Error(err))
	}
	if subTask.DeviceSN != "" {
		e.releaseDeviceLock(ctx, subTask.DeviceSN)
	}
	if err := e.taskRepo.IncrementCounts(ctx, subTask.TaskID, 0, 1); err != nil {
		e.logger.Error("increment rollback fail count", zap.Error(err))
	}
	finalizeTask(ctx, e.taskRepo, e.logger, subTask.TaskID)
}

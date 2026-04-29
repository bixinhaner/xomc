package software

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/acs/connreq"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/device"
	devtask "github.com/omcgo/omcgo/internal/task"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// UpgradeExecutor handles the event-driven upgrade lifecycle for individual devices.
type UpgradeExecutor struct {
	taskRepo     TaskRepository
	subTaskRepo  SubTaskRepository
	deviceRepo   device.DeviceRepository
	firmwareRepo FirmwareRepository
	cmdQueue     devtask.Enqueuer
	connReq      *connreq.Client
	redis        redis.UniversalClient
	eventBus     event.EventBus
	adapter      UpgradeAdapter
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
		adapter:      NewDefaultUpgradeAdapter(),
		logger:       logger.Named("upgrade-executor"),
	}
}

// ExecuteOne runs the upgrade flow for a single sub-task.
// Flow: Step 1 (online check) → Step 2 (send Download cmd) → Step 3 (monitor download) → wait for events.
func (e *UpgradeExecutor) ExecuteOne(ctx context.Context, subTask *UpgradeSubTask, fw *FirmwareVersion, isKeepConfig bool) {
	dev, err := e.deviceRepo.GetByID(ctx, subTask.DeviceID)
	if err != nil {
		e.failSubTask(ctx, subTask, "Upgrade can not be started, device not found.", FailureDeviceNotFound)
		return
	}

	// Step 1: Device online check
	if dev.Status != model.DeviceActive {
		e.logger.Info("device offline, entering wait state",
			zap.String("device_sn", dev.SerialNumber),
			zap.String("sub_task_id", subTask.ID.String()))

		// Set Redis wait key for HandleDeviceOnline to resume
		waitKey := fmt.Sprintf("software:upgrade:wait:%s", dev.SerialNumber)
		e.redis.Set(ctx, waitKey, subTask.ID.String(), time.Hour)

		// Record device info and mark as suspended (waiting for device)
		subTask.DeviceSN = dev.SerialNumber
		subTask.OriVersion = dev.FirmwareVersion
		e.subTaskRepo.Update(ctx, subTask)
		e.subTaskRepo.UpdateStatus(ctx, subTask.ID, UpgradeSuspended, "waiting for device online")
		return
	}

	// Acquire device-level lock via Redis SETNX
	acquired, err := e.acquireDeviceLock(ctx, dev.SerialNumber, subTask.ID)
	if err != nil {
		e.logger.Warn("acquire device lock failed, proceeding without lock",
			zap.String("device_sn", dev.SerialNumber), zap.Error(err))
	} else if !acquired {
		e.failLockedSubTask(ctx, subTask, dev.SerialNumber)
		return
	}

	// Record device SN and original version
	subTask.DeviceSN = dev.SerialNumber
	subTask.OriVersion = dev.FirmwareVersion
	if err := e.subTaskRepo.Update(ctx, subTask); err != nil {
		e.logger.Error("update sub-task device info", zap.Error(err))
	}

	// Step 2: Build and push Download command
	commandKey := e.adapter.DownloadCommandKey(subTask.ID.String())
	downloadURL := "firmware/" + fw.MinIOPath

	rawMode := "true"
	if isKeepConfig {
		rawMode = "false"
	}

	paramsJSON, err := json.Marshal(map[string]interface{}{
		"command_key":     commandKey,
		"file_type":       e.adapter.DownloadFileType(fw.FileType),
		"url":             downloadURL,
		"file_size":       fw.FileSize,
		"file_name":       fw.FileName,
		"target_filename": fw.FileName,
		"md5":             fw.MD5Val,
		"raw_mode":        rawMode,
	})
	if err != nil {
		e.releaseDeviceLock(ctx, dev.SerialNumber)
		e.failSubTask(ctx, subTask, fmt.Sprintf("Upgrade can not be started, internal error: %v", err), FailureInternalError)
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
		e.failSubTask(ctx, subTask, "Upgrade can not be started, failed to send download command to device.", FailureCommandPush)
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

	// Set Redis flags for download monitoring and TC matching
	tcKey := fmt.Sprintf("TransferCompleteReq_%s", subTask.ID.String())
	e.redis.Set(ctx, tcKey, "0", 30*time.Minute)

	dlKey := fmt.Sprintf("DownloadingFlag_%s_%s", subTask.TaskID.String(), dev.SerialNumber)
	e.redis.Set(ctx, dlKey, "0", 10*time.Minute)

	// Transition to downloading
	if err := e.subTaskRepo.UpdateStatus(ctx, subTask.ID, UpgradeDownloading, ""); err != nil {
		e.logger.Error("update sub-task to downloading", zap.Error(err))
	}

	e.logger.Info("upgrade download pushed",
		zap.String("sub_task_id", subTask.ID.String()),
		zap.String("device_sn", dev.SerialNumber),
		zap.String("command_key", commandKey),
		zap.String("download_url", downloadURL),
		zap.String("file_type", e.adapter.DownloadFileType(fw.FileType)),
		zap.String("file_name", fw.FileName),
		zap.Int64("file_size", fw.FileSize),
		zap.String("md5", fw.MD5Val),
		zap.Bool("is_keep_config", isKeepConfig),
		zap.String("firmware_version", fw.Version),
		zap.Bool("is_5g", Is5G(dev)))

	// Step 3: Start download progress monitor in background
	go e.monitorDownloadProgress(context.Background(), subTask, dev.SerialNumber)
}

// monitorDownloadProgress polls the DownloadingFlag Redis key to track download progress.
// Values: null=not started, "0"=downloading, "2"=interrupted, "3"=file not found, other=complete.
func (e *UpgradeExecutor) monitorDownloadProgress(ctx context.Context, subTask *UpgradeSubTask, deviceSN string) {
	key := fmt.Sprintf("DownloadingFlag_%s_%s", subTask.TaskID.String(), deviceSN)
	deadline := time.Now().Add(10 * time.Minute)
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case t := <-ticker.C:
			if t.After(deadline) {
				// Check if sub-task is still in downloading state before failing
				current, err := e.subTaskRepo.GetByID(ctx, subTask.ID)
				if err != nil || current.Status != UpgradeDownloading {
					return
				}
				e.failSubTask(ctx, subTask, "Download can not be started, there is no DownloadResponse msg from device.", FailureDownloadTimeout)
				return
			}

			val, err := e.redis.Get(ctx, key).Result()
			if err == redis.Nil {
				// Not started yet, keep waiting
				continue
			}
			if err != nil {
				e.logger.Error("check download flag", zap.Error(err))
				continue
			}

			switch val {
			case "0": // downloading
				continue
			case "2": // interrupted (resume)
				continue
			case "3": // file not found
				e.failSubTask(ctx, subTask, "Download failed, target version file can not be found.", FailureDownloadFile)
				return
			default: // download complete — clear flag and stop monitoring
				e.redis.Del(ctx, key)
				return
			}
		}
	}
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
		return nil
	}

	if subTask.Status != UpgradeDownloading {
		return nil
	}

	if payload.FaultCode != 0 {
		e.failSubTask(ctx, subTask, fmt.Sprintf("Download failed, device rejected download. FaultCode: %d, FaultString: %s", payload.FaultCode, payload.FaultStr), FailureDownloadFault)
		return nil
	}

	e.logger.Debug("download accepted, waiting for TransferComplete",
		zap.String("sub_task_id", subTask.ID.String()),
		zap.String("device_sn", payload.DeviceSN))
	return nil
}

// HandleRebootComplete handles device.inform.reboot_complete events.
// Used for rollback completion detection and 4G upgrade completion.
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

	subTask, err := e.subTaskRepo.GetActiveByDeviceID(ctx, dev.ID)
	if err != nil {
		return nil
	}

	if subTask.Status != UpgradeRebooting {
		return nil
	}

	// For 5G upgrade: skip — HandleUpgradeFinish handles the 102 event
	task, err := e.taskRepo.GetByID(ctx, subTask.TaskID)
	if err != nil {
		return nil
	}
	if task.TaskType == TaskTypeUpgrade && Is5G(dev) {
		e.logger.Debug("skip reboot_complete for 5G upgrade, waiting for 102 event",
			zap.String("sub_task_id", subTask.ID.String()))
		return nil
	}

	// Reboot complete → finalize (rollback or 4G upgrade)
	e.completeSubTask(ctx, subTask, dev.SerialNumber)

	e.logger.Info("reboot complete, task finalized",
		zap.String("sub_task_id", subTask.ID.String()),
		zap.String("device_sn", payload.DeviceSN))
	return nil
}

// HandleUpgradeFinish handles the 5G 102 UPGRADE FINISH event.
// Only processes sub-tasks in rebooting state for 5G devices.
func (e *UpgradeExecutor) HandleUpgradeFinish(ctx context.Context, evt event.Event) error {
	var payload struct {
		DeviceID struct {
			SerialNumber string `json:"SerialNumber"`
		} `json:"device_id"`
		Events        []string                 `json:"events"`
		ParameterList []map[string]interface{} `json:"parameter_list"`
	}
	if err := evt.DecodePayload(&payload); err != nil || payload.DeviceID.SerialNumber == "" {
		return nil
	}

	deviceSN := payload.DeviceID.SerialNumber
	dev, err := e.deviceRepo.GetBySerialNumber(ctx, deviceSN)
	if err != nil || dev == nil {
		return nil
	}

	subTask, err := e.subTaskRepo.GetActiveByDeviceID(ctx, dev.ID)
	if err != nil {
		return nil
	}

	if subTask.Status != UpgradeRebooting {
		return nil
	}

	// Extract upgradeStatus from Inform parameter list
	upgradeStatus := extractParamValue(payload.ParameterList, "UpgradeStatus")

	e.logger.Info("5G upgrade finish received",
		zap.String("sub_task_id", subTask.ID.String()),
		zap.String("device_sn", deviceSN),
		zap.String("command_key", subTask.CommandKey),
		zap.String("upgrade_status", upgradeStatus))

	// Per design doc: "2" or "3" → failed, other (including "1") → success
	if upgradeStatus == "2" || upgradeStatus == "3" {
		e.failSubTask(ctx, subTask, fmt.Sprintf("Upgrade failed, 5G upgrade status: %s", upgradeStatus), Failure5GInstall)
	} else {
		e.completeSubTask(ctx, subTask, deviceSN)
	}

	return nil
}

// extractParamValue searches a TR-069 parameter list for a parameter whose name
// contains the given suffix and returns its value.
func extractParamValue(params []map[string]interface{}, nameSuffix string) string {
	for _, p := range params {
		name, _ := p["name"].(string)
		if name == "" {
			continue
		}
		// Match parameter names ending with the suffix (e.g., "...UpgradeStatus]")
		if len(name) >= len(nameSuffix) && name[len(name)-len(nameSuffix):] == nameSuffix {
			val, _ := p["value"].(string)
			return val
		}
	}
	return ""
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
		return nil
	}
	if err != nil {
		e.logger.Error("check upgrade wait key", zap.Error(err))
		return nil
	}

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

	if subTask.Status != UpgradeSuspended && subTask.Status != UpgradePending {
		return nil
	}

	e.logger.Info("device online, resuming pending upgrade",
		zap.String("sub_task_id", subTask.ID.String()),
		zap.String("device_sn", payload.DeviceSN))

	if subTask.FirmwareID != nil {
		fw, err := e.firmwareRepo.GetByID(ctx, *subTask.FirmwareID)
		if err != nil {
			e.failSubTask(ctx, subTask, fmt.Sprintf("Upgrade can not be started, the target version file can not be found: %v", err), FailureFirmwareGone)
			return nil
		}

		// Get isKeepConfig from parent task
		isKeepConfig := true
		if task, err := e.taskRepo.GetByID(ctx, subTask.TaskID); err == nil {
			isKeepConfig = task.IsKeepConfig
		}

		// Reset status to pending for re-execution
		e.subTaskRepo.UpdateStatus(ctx, subTask.ID, UpgradePending, "")
		go e.ExecuteOne(context.Background(), subTask, fw, isKeepConfig)
	}

	return nil
}

func (e *UpgradeExecutor) acquireDeviceLock(ctx context.Context, deviceSN string, taskID uuid.UUID) (bool, error) {
	key := fmt.Sprintf("software:upgrade:active:%s", deviceSN)
	ok, err := e.redis.SetNX(ctx, key, taskID.String(), time.Hour).Result()
	if err != nil {
		return false, fmt.Errorf("acquire device lock: %w", err)
	}
	return ok, nil
}

func (e *UpgradeExecutor) releaseDeviceLock(ctx context.Context, deviceSN string) {
	key := fmt.Sprintf("software:upgrade:active:%s", deviceSN)
	e.redis.Del(ctx, key)
}

func (e *UpgradeExecutor) failSubTask(ctx context.Context, subTask *UpgradeSubTask, reason string, code FailureCode) {
	if err := e.subTaskRepo.UpdateStatusWithCode(ctx, subTask.ID, UpgradeFailed, reason, code); err != nil {
		e.logger.Error("fail sub-task", zap.String("sub_task_id", subTask.ID.String()), zap.Error(err))
	}
	if subTask.DeviceSN != "" {
		e.releaseDeviceLock(ctx, subTask.DeviceSN)
	}
	if err := e.taskRepo.IncrementCounts(ctx, subTask.TaskID, 0, 1); err != nil {
		e.logger.Error("increment fail count", zap.Error(err))
	}
	finalizeTask(ctx, e.taskRepo, e.logger, subTask.TaskID)
}

// failLockedSubTask handles the DEVICE_LOCKED case by looking up the blocking task name.
func (e *UpgradeExecutor) failLockedSubTask(ctx context.Context, subTask *UpgradeSubTask, deviceSN string) {
	reason := "Upgrade can not be started, device can not be in multi running tasks."

	e.failSubTask(ctx, subTask, reason, FailureDeviceLocked)
}

func (e *UpgradeExecutor) completeSubTask(ctx context.Context, subTask *UpgradeSubTask, deviceSN string) {
	if err := e.subTaskRepo.UpdateStatus(ctx, subTask.ID, UpgradeCompleted, ""); err != nil {
		e.logger.Error("complete sub-task", zap.String("sub_task_id", subTask.ID.String()), zap.Error(err))
	}
	e.releaseDeviceLock(ctx, deviceSN)

	// Clean up Redis flags
	tcKey := fmt.Sprintf("TransferCompleteReq_%s", subTask.ID.String())
	e.redis.Del(ctx, tcKey)
	if subTask.DeviceSN != "" {
		dlKey := fmt.Sprintf("DownloadingFlag_%s_%s", subTask.TaskID.String(), subTask.DeviceSN)
		e.redis.Del(ctx, dlKey)
	}

	if err := e.taskRepo.IncrementCounts(ctx, subTask.TaskID, 1, 0); err != nil {
		e.logger.Error("increment success count", zap.Error(err))
	}
	finalizeTask(ctx, e.taskRepo, e.logger, subTask.TaskID)
}

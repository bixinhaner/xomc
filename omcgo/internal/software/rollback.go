package software

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
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

// RollbackExecutor handles version rollback via SetParameterValues.
//
// 业务规则（用户口径，2026-05-22 重写）：
//
//	4G（非 5G / BNQ）：
//	  Stage 1: GET standardPath Device.DeviceInfo.ROLLBACK_ENABLE（经 Translator 翻译到设备私有 path）
//	    · 返回 "1" / "true" → 进 Stage 2
//	    · 返回 "0" / "false" → fail "设备不支持回退"
//	    · 5min 内无响应 → reaper 兜底 fail
//	  Stage 2: SPV standardPath Device.DeviceInfo.ROLLBACK_CONTROL（同样 Translator 翻译）
//	    · 值与 xsd 类型由 adapter.RollbackParameterValue 根据 privatePath 决定
//	      （TR-098 系 InternetGatewayDevice.*RollBackEnable → "true"/boolean；
//	      其它如 X_COM_ROLLBACK_CONTROL → "1"/string）
//
//	5G/NR：
//	  直接 Stage 2，无需 GET。standardPath = Device.SoftwareCtrl.ActivateEnable，值固定 "1"/string。
//
// 完成路径：CPE 收 SPV → 回 SPVResponse → 设备重启 → BOOT Inform → ACS publish
// device.inform.reboot_complete → HandleRebootComplete 标 Completed。
//
// 失败兜底：
//   - SOAP Fault（CPE 拒收 SPV/GPV）→ HandleSetParamsResponse / HandleGetParamsResponseForRollback
//     按 command_key 反查后立即标 failed
//   - 30min 等不到 reboot → reaper（rebooting 长超时窗口）
type RollbackExecutor struct {
	taskRepo       TaskRepository
	subTaskRepo    SubTaskRepository
	deviceRepo     device.DeviceRepository
	cmdQueue       devtask.Enqueuer
	connReq        *connreq.Client
	redis          redis.UniversalClient
	eventBus       event.EventBus
	adapter        UpgradeAdapter
	pathTranslator ParamPathTranslator // 可为 nil；nil 时所有 path 走 passthrough（不翻译）
	logger         *zap.Logger
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
		adapter:     NewDefaultUpgradeAdapter(),
		logger:      logger.Named("rollback-executor"),
	}
}

// SetParamPathTranslator 注入 standardPath → privatePath 翻译器。
// nil 时所有 path 走 passthrough（与历史行为兼容，但生产环境强烈建议注入）。
func (e *RollbackExecutor) SetParamPathTranslator(t ParamPathTranslator) {
	e.pathTranslator = t
}

// rollbackEnableCmdKeyPrefix 用 sub_task_id 后缀，让 HandleGetParamsResponseForRollback
// 能按 command_key 前缀 + sub_task_id 后缀准确反查 sub_task 推进 Stage 2。
const rollbackEnableCmdKeyPrefix = "rollback-enable-check-"

// RollbackOne 执行单设备的回退。
//
// 入参 tech 是设备 Technology（LTE / NR），用来分流 4G 两阶段 / 5G 一阶段。
// adapter 提供 standardPath；translator 把 standardPath → privatePath。
func (e *RollbackExecutor) RollbackOne(ctx context.Context, subTask *UpgradeSubTask, dev *model.Device, tech model.Technology) {
	// 1) 设备级锁 + 在线检查
	acquired, err := e.acquireDeviceLock(ctx, dev.SerialNumber, subTask.ID)
	if err != nil {
		e.logger.Warn("acquire device lock for rollback failed",
			zap.String("device_sn", dev.SerialNumber), zap.Error(err))
	} else if !acquired {
		e.FailRollbackSubTask(ctx, subTask, "Rollback can not be started, device can not be in multi running tasks.", FailureDeviceLocked)
		return
	}
	if dev.Status != model.DeviceActive {
		e.releaseDeviceLock(ctx, dev.SerialNumber, subTask.ID)
		e.FailRollbackSubTask(ctx, subTask, fmt.Sprintf("Rollback can not be started, device is %s. Please retry when device is online.", dev.Status), FailureDeviceOffline)
		return
	}

	// 2) 记录设备信息
	subTask.DeviceSN = dev.SerialNumber
	subTask.OriVersion = dev.FirmwareVersion
	if err := e.subTaskRepo.Update(ctx, subTask); err != nil {
		e.logger.Error("update rollback sub-task device info", zap.Error(err))
	}

	// 3) 4G：Stage 1 - 发 GPV 查 ROLLBACK_ENABLE，由 GPV response handler 推进 Stage 2
	//    5G：直接 Stage 2
	if e.adapter.RollbackNeedsEnableCheck(tech) {
		enablePath := e.adapter.RollbackEnableCheckPath(tech)
		if enablePath == "" {
			// 不应走到（NeedsEnableCheck=true 但 path 空），兜底跳过
			e.logger.Warn("RollbackNeedsEnableCheck=true but EnableCheckPath empty; skip stage 1",
				zap.String("device_sn", dev.SerialNumber))
			e.dispatchRollbackSPV(ctx, subTask, dev, tech)
			return
		}
		e.pushEnableCheckGPV(ctx, subTask, dev, enablePath)
		return
	}
	e.dispatchRollbackSPV(ctx, subTask, dev, tech)
}

// pushEnableCheckGPV 推送 ROLLBACK_ENABLE GPV 命令，状态翻 Downloading
// （借用短 RPCResponse 超时分支让 reaper 兜底）。CommandKey 形如
// "rollback-enable-check-{sub_task_id}"，方便后续 GPV response 反查。
func (e *RollbackExecutor) pushEnableCheckGPV(ctx context.Context, subTask *UpgradeSubTask, dev *model.Device, standardEnablePath string) {
	enablePath := e.translatePath(ctx, dev, standardEnablePath)
	cmdKey := rollbackEnableCmdKeyPrefix + subTask.ID.String()

	paramsJSON, err := json.Marshal(map[string]interface{}{
		"names":       []string{enablePath},
		"command_key": cmdKey,
	})
	if err != nil {
		e.releaseDeviceLock(ctx, dev.SerialNumber, subTask.ID)
		e.FailRollbackSubTask(ctx, subTask, fmt.Sprintf("Rollback can not be started, internal error: %v", err), FailureInternalError)
		return
	}
	if _, err := e.cmdQueue.CreateTask(ctx, &devtask.CreateTaskRequest{
		DeviceSN:   dev.SerialNumber,
		Method:     "GetParameterValues",
		Params:     paramsJSON,
		Source:     devtask.TaskSourceSystem,
		CommandKey: cmdKey,
	}); err != nil {
		e.releaseDeviceLock(ctx, dev.SerialNumber, subTask.ID)
		e.FailRollbackSubTask(ctx, subTask, fmt.Sprintf("Rollback can not be started, failed to push enable-check command: %v", err), FailureCommandPush)
		return
	}

	// 记 CommandKey 让后续 HandleGetParamsResponseForRollback 按 cmdKey 反查
	subTask.CommandKey = cmdKey
	if err := e.subTaskRepo.Update(ctx, subTask); err != nil {
		e.logger.Error("update rollback sub-task command_key (stage 1)", zap.Error(err))
	}

	// 状态翻 Downloading：DB 状态保持兼容，UFTE 展示层会按回退任务映射为
	// rollback_checking；reaper 也会按 task_type + command_key 输出回退专用超时原因。
	if err := e.subTaskRepo.UpdateStatus(ctx, subTask.ID, UpgradeDownloading, ""); err != nil {
		e.logger.Error("update rollback sub-task to downloading (stage 1)", zap.Error(err))
	}

	// 异步唤醒设备
	if dev.ConnectionRequestURL != "" {
		deviceSN := dev.SerialNumber
		connURL := dev.ConnectionRequestURL
		go func() {
			if err := e.connReq.Send(context.Background(), deviceSN, connURL); err != nil {
				e.logger.Warn("send connection request for rollback enable check",
					zap.String("device_sn", deviceSN), zap.Error(err))
			}
		}()
	}

	e.logger.Info("rollback stage 1 (enable-check GPV) pushed",
		zap.String("sub_task_id", subTask.ID.String()),
		zap.String("device_sn", dev.SerialNumber),
		zap.String("standard_enable_path", standardEnablePath),
		zap.String("dispatch_enable_path", enablePath),
		zap.String("command_key", cmdKey))
}

// dispatchRollbackSPV 推送 ROLLBACK_CONTROL（4G）或 ActivateEnable（5G）SPV 命令。
// 5G 直接进入；4G 阶段一通过后由 GPV response handler 调用。
//
// Translator: standardPath → privatePath；privatePath 进一步决定 value+xsdType
// （adapter.RollbackParameterValue）。
func (e *RollbackExecutor) dispatchRollbackSPV(ctx context.Context, subTask *UpgradeSubTask, dev *model.Device, tech model.Technology) {
	standardPath := e.adapter.RollbackParameterPath(tech)
	privatePath := e.translatePath(ctx, dev, standardPath)
	value, xsdType := e.adapter.RollbackParameterValue(tech, privatePath)

	paramsJSON, err := json.Marshal(map[string]interface{}{
		"values": []map[string]string{
			{
				"name":  privatePath,
				"value": value,
				"type":  xsdType,
			},
		},
		"command_key": subTask.ID.String(),
	})
	if err != nil {
		e.releaseDeviceLock(ctx, dev.SerialNumber, subTask.ID)
		e.FailRollbackSubTask(ctx, subTask, fmt.Sprintf("Rollback can not be started, internal error: %v", err), FailureInternalError)
		return
	}
	if _, err := e.cmdQueue.CreateTask(ctx, &devtask.CreateTaskRequest{
		DeviceSN:   dev.SerialNumber,
		Method:     "SetParameterValues",
		Params:     paramsJSON,
		Source:     devtask.TaskSourceSystem,
		CommandKey: subTask.ID.String(),
	}); err != nil {
		e.releaseDeviceLock(ctx, dev.SerialNumber, subTask.ID)
		e.FailRollbackSubTask(ctx, subTask, fmt.Sprintf("Rollback can not be started, failed to send set params command to device: %v", err), FailureCommandPush)
		return
	}

	// Stage 2 用 sub_task UUID 当 CommandKey；HandleSetParamsResponse / HandleRebootComplete
	// 都按这个反查 sub_task。
	subTask.CommandKey = subTask.ID.String()
	if err := e.subTaskRepo.Update(ctx, subTask); err != nil {
		e.logger.Error("update rollback sub-task command_key (stage 2)", zap.Error(err))
	}

	// 状态翻 Rebooting —— 必须在 connReq 前；同 ExecuteOne 注释，避免 race。
	if err := e.subTaskRepo.UpdateStatus(ctx, subTask.ID, UpgradeRebooting, ""); err != nil {
		e.logger.Error("update rollback sub-task to rebooting (stage 2)", zap.Error(err))
	}

	// 异步唤醒设备
	if dev.ConnectionRequestURL != "" {
		deviceSN := dev.SerialNumber
		connURL := dev.ConnectionRequestURL
		go func() {
			if err := e.connReq.Send(context.Background(), deviceSN, connURL); err != nil {
				e.logger.Warn("send connection request for rollback",
					zap.String("device_sn", deviceSN), zap.Error(err))
			}
		}()
	}

	e.logger.Info("rollback stage 2 (SPV) pushed",
		zap.String("sub_task_id", subTask.ID.String()),
		zap.String("device_sn", dev.SerialNumber),
		zap.String("tech", string(tech)),
		zap.String("standard_path", standardPath),
		zap.String("dispatch_path", privatePath),
		zap.String("value", value),
		zap.String("xsd_type", xsdType))
}

// translatePath 把 standardPath 翻译为 device 私有路径。Translator 未注入 / 翻译失败 →
// passthrough 返回原 standardPath，并 warn。
func (e *RollbackExecutor) translatePath(ctx context.Context, dev *model.Device, standardPath string) string {
	if e.pathTranslator == nil || standardPath == "" {
		return standardPath
	}
	translated, err := e.pathTranslator.TranslateForDevice(ctx, dev.ProductClass, dev.FirmwareVersion, []string{standardPath})
	if err != nil {
		e.logger.Warn("rollback path translator failed; passthrough",
			zap.String("device_sn", dev.SerialNumber),
			zap.String("product_class", dev.ProductClass),
			zap.String("standard_path", standardPath),
			zap.Error(err))
		return standardPath
	}
	if len(translated) == 0 || translated[0].Private == "" {
		return standardPath
	}
	return translated[0].Private
}

// HandleEnableCheckResponse 是 Stage 1 GPV 响应处理：按 command_key 前缀过滤 rollback-enable-check-，
// 反查 sub_task，检查值 → 推进 Stage 2 或 fail。
//
// 调用方：UpgradeExecutor.HandleGetParamsResponseForRollback（订阅 command.get_parameters.response）
// 把 payload 反序列化后传进来。
func (e *RollbackExecutor) HandleEnableCheckResponse(ctx context.Context, deviceSN, commandKey string, faultCode int, faultStr string, parameterValues []ParameterValueResponse) {
	if !strings.HasPrefix(commandKey, rollbackEnableCmdKeyPrefix) {
		return // 不是 rollback 流程
	}
	subTaskIDStr := strings.TrimPrefix(commandKey, rollbackEnableCmdKeyPrefix)
	subTaskID, err := uuid.Parse(subTaskIDStr)
	if err != nil {
		e.logger.Warn("invalid sub_task_id in rollback enable command_key",
			zap.String("command_key", commandKey), zap.Error(err))
		return
	}
	subTask, err := e.subTaskRepo.GetByID(ctx, subTaskID)
	if err != nil || subTask == nil {
		return
	}
	if subTask.Status != UpgradeDownloading {
		return // 已被别的路径推进
	}

	// CPE 直接 Fault（路径不存在 / 不可读）→ fail
	if isCPEFault(faultCode, faultStr) {
		e.FailRollbackSubTask(ctx, subTask,
			fmt.Sprintf("Rollback can not be started, enable check rejected by device. FaultCode: %d, FaultString: %s",
				faultCode, faultStr),
			FailureRollbackEnableCheckFault)
		return
	}

	// 找到 ROLLBACK_ENABLE 的值（不区分大小写匹配 path 末尾 ROLLBACK_ENABLE，
	// 因为返回的 name 是 privatePath，可能是 X_COM_ROLLBACK_ENABLE / 等其它形态）
	var enableValue string
	for _, pv := range parameterValues {
		lower := strings.ToLower(pv.Name)
		if strings.HasSuffix(lower, "rollback_enable") || strings.HasSuffix(lower, "rollbackenable") {
			enableValue = strings.TrimSpace(pv.Value)
			break
		}
	}
	if !isRollbackEnabled(enableValue) {
		e.FailRollbackSubTask(ctx, subTask,
			fmt.Sprintf("Rollback can not be started, device does not support rollback (ROLLBACK_ENABLE=%q).", enableValue),
			FailureRollbackNotSupported)
		return
	}

	// Stage 2：重新 fetch 设备拿最新 tech / 在线状态
	dev, err := e.deviceRepo.GetByID(ctx, subTask.DeviceID)
	if err != nil {
		e.FailRollbackSubTask(ctx, subTask, fmt.Sprintf("Rollback can not be started, device not found: %v", err), FailureDeviceNotFound)
		return
	}
	// 三态识别：GSM(2G) / NR(5G) / LTE(4G 兜底)，与 service.go 回退派发保持一致。
	tech := ResolveDeviceTech(dev)
	e.logger.Info("rollback stage 1 passed, dispatching SPV (stage 2)",
		zap.String("sub_task_id", subTask.ID.String()),
		zap.String("device_sn", deviceSN),
		zap.String("enable_value", enableValue))
	e.dispatchRollbackSPV(ctx, subTask, dev, tech)
}

// ParameterValueResponse 是 GPV 响应 payload 里 parameter_values 数组的一项。
// 跟 tr069.ParameterValueStruct 结构同款，单独定义避免反向依赖 pkg/tr069。
type ParameterValueResponse struct {
	Name  string `json:"name"`
	Value string `json:"value"`
	Type  string `json:"type,omitempty"`
}

// isRollbackEnabled 把 CPE 回的 enable 值映射为 bool。
// 支持 4 种格式（CPE 厂商各家不一）：
//
//	"1" / "true" / "True" / "TRUE" → true
//	"0" / "false" / "False" / 空    → false
func isRollbackEnabled(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true":
		return true
	default:
		return false
	}
}

func (e *RollbackExecutor) acquireDeviceLock(ctx context.Context, deviceSN string, taskID interface{ String() string }) (bool, error) {
	key := upgradeDeviceLockKey(deviceSN)
	ok, err := e.redis.SetNX(ctx, key, taskID.String(), time.Hour).Result()
	if err != nil {
		return false, fmt.Errorf("acquire device lock: %w", err)
	}
	return ok, nil
}

func (e *RollbackExecutor) releaseDeviceLock(ctx context.Context, deviceSN string, subTaskID uuid.UUID) {
	if err := releaseOwnedDeviceLock(ctx, e.redis, deviceSN, subTaskID); err != nil {
		e.logger.Warn("release rollback device lock",
			zap.String("sub_task_id", subTaskID.String()),
			zap.String("device_sn", deviceSN),
			zap.Error(err))
	}
}

// FailRollbackSubTask marks a sub-task as failed and finalizes the parent task.
func (e *RollbackExecutor) FailRollbackSubTask(ctx context.Context, subTask *UpgradeSubTask, reason string, code FailureCode) {
	if err := e.subTaskRepo.UpdateStatusWithCode(ctx, subTask.ID, UpgradeFailed, reason, code); err != nil {
		e.logger.Error("fail rollback sub-task", zap.String("sub_task_id", subTask.ID.String()), zap.Error(err))
	}
	if subTask.DeviceSN != "" && code != FailureDeviceLocked {
		// DEVICE_LOCKED means this sub-task never acquired the lock.
		e.releaseDeviceLock(ctx, subTask.DeviceSN, subTask.ID)
	}
	if err := e.taskRepo.IncrementCounts(ctx, subTask.TaskID, 0, 1); err != nil {
		e.logger.Error("increment rollback fail count", zap.Error(err))
	}
	finalizeTask(ctx, e.taskRepo, e.logger, subTask.TaskID)
}

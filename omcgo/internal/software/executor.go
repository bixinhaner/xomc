package software

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/acs/connreq"
	"github.com/omcgo/omcgo/internal/acs/transfercfg"
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
	// Upload 配置（日志采集 / 备份 Upload RPC 使用）。
	// 实际下发给 CPE 的 base URL / Username / Password 优先从 transferProvider（运行时 sys_config
	// 'acs_transfer' 类别）拿，对应前端"系统管理 → ACS 传输"页面。未注入或运行时未配时退化到
	// acsUploadBaseURL（YAML 静态配置），仍为空则 Warn——CPE 拿到纯路径必然拒绝上传。
	transferProvider transfercfg.Provider
	acsUploadBaseURL string // CPE 可达的 ACS 上传服务基础 URL，如 "http://localhost:8080"
	// logCollectResumer 在设备上线时唤醒挂在 redis wait key 的 LogCollect 类子任务
	// （备份 / 日志采集，FirmwareID=NULL，没有"重新执行"所需的固件信息）。实现位于
	// ufte 包，避免 software 反向依赖 ufte 的 catalog——通过接口注入解耦。
	// nil-safe：未注入时 LogCollect 子任务上线唤醒退化为 no-op + 警告日志。
	logCollectResumer LogCollectResumer
}

// LogCollectResumer 接口承载 LogCollect 类子任务"设备上线即重试"的能力。
// 注入位置：cmd/app/provider/modules.go initUFTEModule，由 ufte.Service 实现。
// 详见 docs/project/backup-display-fix-20260520.md F6。
type LogCollectResumer interface {
	// ResumeLogCollectSubTask 由 HandleDeviceOnline 在挂起的 LogCollect 子任务
	// 被唤醒时回调；实现方需要根据 parent.DownloadFileType 解析 UFTE catalog 的
	// transport_path，再调 SoftwareService.ExecuteOneUploadDirect 重启 Upload RPC。
	ResumeLogCollectSubTask(ctx context.Context, subTask *UpgradeSubTask, parent *UpgradeTask) error
}

// SetUploadConfig 注入日志采集所需的 ACS 上传基础 URL（CPE 可达地址）。
// 仅在 transferProvider 未注入或运行时配置缺失时作为兜底，YAML 静态默认值。
func (e *UpgradeExecutor) SetUploadConfig(acsUploadBaseURL string) {
	e.acsUploadBaseURL = acsUploadBaseURL
}

// SetTransferProvider 注入运行时 ACS 传输配置（来源：sys_config 'acs_transfer' 类别，
// 由前端"系统管理 → ACS 传输"页面维护）。注入后 ExecuteOneUpload 会优先从这里拿
// BaseURL / Username / Password。改配置不用重启进程，30 秒缓存内自动生效。
func (e *UpgradeExecutor) SetTransferProvider(p transfercfg.Provider) {
	e.transferProvider = p
}

// SetLogCollectResumer 注入 LogCollect 类子任务"设备上线即重试"的回调实现。
// 未注入时 HandleDeviceOnline 对 LogCollect 子任务退化为 no-op（保持原有 升级类 行为）。
func (e *UpgradeExecutor) SetLogCollectResumer(r LogCollectResumer) {
	e.logCollectResumer = r
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
func (e *UpgradeExecutor) ExecuteOne(ctx context.Context, subTask *UpgradeSubTask, fw *FirmwareVersion, isKeepConfig bool, downloadFileType string) {
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
	effectiveDownloadFileType := downloadFileType
	if effectiveDownloadFileType == "" {
		effectiveDownloadFileType = e.adapter.DownloadFileType(fw.FileType)
	}

	rawMode := "true"
	if isKeepConfig {
		rawMode = "false"
	}

	paramsJSON, err := json.Marshal(map[string]interface{}{
		"command_key":     commandKey,
		"file_type":       effectiveDownloadFileType,
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

	// Set Redis flags for download monitoring and TC matching
	tcKey := fmt.Sprintf("TransferCompleteReq_%s", subTask.ID.String())
	e.redis.Set(ctx, tcKey, "0", 30*time.Minute)

	dlKey := fmt.Sprintf("DownloadingFlag_%s_%s", subTask.TaskID.String(), dev.SerialNumber)
	e.redis.Set(ctx, dlKey, "0", 10*time.Minute)

	// Transition to downloading —— 必须放在 connReq 之前。详见 ExecuteOneUpload 同名注释：
	// connReq 同步 30s 重试会跟 UpdateStatus 串行，设备秒级响应回 TC 时 status 还是
	// pending，handleTCBody 直接 return 导致 fault 丢失、状态卡死。
	if err := e.subTaskRepo.UpdateStatus(ctx, subTask.ID, UpgradeDownloading, ""); err != nil {
		e.logger.Error("update sub-task to downloading", zap.Error(err))
	}

	// Send Connection Request to wake device —— 异步，不阻塞 ExecuteOne。
	if dev.ConnectionRequestURL != "" {
		deviceSN := dev.SerialNumber
		connURL := dev.ConnectionRequestURL
		go func() {
			if err := e.connReq.Send(context.Background(), deviceSN, connURL); err != nil {
				e.logger.Warn("send connection request",
					zap.String("device_sn", deviceSN), zap.Error(err))
			}
		}()
	}

	e.logger.Info("upgrade download pushed",
		zap.String("sub_task_id", subTask.ID.String()),
		zap.String("device_sn", dev.SerialNumber),
		zap.String("command_key", commandKey),
		zap.String("download_url", downloadURL),
		zap.String("file_type", effectiveDownloadFileType),
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

// resolveTemplate 替换模板变量，支持 {task_id8}、{sn}、{fileType}、{targetFileName}。
func resolveTemplate(tmpl string, vars map[string]string) string {
	result := tmpl
	for k, v := range vars {
		result = strings.ReplaceAll(result, "{"+k+"}", v)
	}
	return result
}

// taskIDPrefix 提取任务 UUID 的前 8 位十六进制字符（不含连字符）。
func taskIDPrefix(id uuid.UUID) string {
	hex := strings.ReplaceAll(id.String(), "-", "")
	if len(hex) > 8 {
		return hex[:8]
	}
	return hex
}

// deriveUploadCommandKey 根据已渲染的 fileType 推断厂商私有约定的 CommandKey 格式。
// baicells 系列 CPE firmware 内部按 CommandKey 前缀分发处理逻辑——
// 用纯 UUID 当 CommandKey 时设备识别不出这是 NV/XML 备份请求，会直接静默丢弃
// Upload RPC（既不回 UploadResponse 也不向 FileUploadService 发 PUT）。
//
// 真实工作样本（来自厂商抓包）：
//   - NV 备份：CommandKey = "Collect NV|<OUI>_<SN>,<sub_task_uuid>"
//   - XML 备份：同上 ACTION 改为 "XML"（保守按 NV 同款，等真实样本验证）
//   - 其它 FileType（日志、数据模型 upload）：保持原 UUID，避免影响存量已工作流程
//
// OUI 是这里的关键——厂商样本里 `MMMM` 位置对应的是设备 OUI 而不是 manufacturer
// 字段，CPE 内部按 OUI 校验 CommandKey 来源。
//
// fileType 形如 "10 48BF74 Configuration File" / "12 48BF74 Configuration File" /
// "4 Vendor Log File"——以第一个 token 的数字部分区分 NV(12)/XML(10)/其它。
func deriveUploadCommandKey(resolvedFileType, oui, serialNumber, subTaskID string) string {
	parts := strings.SplitN(strings.TrimSpace(resolvedFileType), " ", 2)
	if len(parts) < 2 {
		return subTaskID
	}
	var action string
	switch parts[0] {
	case "12":
		action = "NV"
	case "10":
		action = "XML"
	default:
		return subTaskID
	}
	ouiKey := oui
	if ouiKey == "" {
		ouiKey = fallbackOUI
	}
	return fmt.Sprintf("Collect %s|%s_%s,%s", action, ouiKey, serialNumber, subTaskID)
}

// fallbackOUI 是设备 OUI 字段为空时下发 Upload SOAP 用的兜底值。
// 48BF74 是 Baicells（主流小基站厂商）官方分配的 OUI；OMC 当前管的基本都是
// Baicells 系列，用这个兜底比留字面值 "{OUI}" 给 CPE 更安全（CPE 一般要求
// 形如 "10 48BF74 Configuration File" 这种带真实 OUI 的 FileType 字符串才能
// 正确识别）。若以后接入其它厂商设备，应当确保设备注册时正确填了 OUI 字段。
const fallbackOUI = "48BF74"

// resolveUploadBaseURL 拿 Upload RPC 实际下发给 CPE 的 base URL。
// 优先级：
//  1. transferProvider（运行时 sys_config 'acs_transfer'.uploadBaseURL，前端"系统管理 → ACS 传输"维护）
//  2. SetUploadConfig 注入的 YAML 静态默认值（acs_upload_base_url）
//  3. 空（ExecuteOneUpload 会 Warn）
//
// 注：Username / Password 一律不下发——CPE 拿到 <cwmp:Username></cwmp:Username>
// 空标签即可（产品线明确要求 Upload / Download 都不走 HTTP Basic Auth）。
// nil-safe：transferProvider 未注入时直接走 YAML 兜底。
func (e *UpgradeExecutor) resolveUploadBaseURL(ctx context.Context) string {
	if e.transferProvider != nil {
		if baseURL := strings.TrimRight(e.transferProvider.Snapshot(ctx).Upload.BaseURL, "/"); baseURL != "" {
			return baseURL
		}
	}
	return strings.TrimRight(e.acsUploadBaseURL, "/")
}

// resolveOUIPlaceholders 在 fileType / transportPath 模板里把 "{OUI}" 替换为
// 设备真实 OUI；设备未上报 OUI 时退化到 fallbackOUI 并 warn 一次。
// 必须在 Upload 命令入队前做——CPE 拿到的 FileType 字符串里再有字面 "{OUI}"
// 就完全无法识别，必然被拒绝（FaultCode≠0）。
func (e *UpgradeExecutor) resolveOUIPlaceholders(deviceSN, deviceOUI string, raw string) string {
	if !strings.Contains(raw, "{OUI}") {
		return raw
	}
	oui := deviceOUI
	if oui == "" {
		e.logger.Warn("device OUI not populated; falling back to default OUI for upload template",
			zap.String("device_sn", deviceSN),
			zap.String("fallback_oui", fallbackOUI),
			zap.String("template", raw))
		oui = fallbackOUI
	}
	return strings.ReplaceAll(raw, "{OUI}", oui)
}

// ExecuteOneUpload 执行日志采集类 Upload RPC 子任务。
// 流程：设备在线检查 → 获取设备锁 → 构造上传 URL → 推送 Upload 命令 → 发送 Connection Request。
// 当 CPE 上传完成后，ACS 收到 TransferComplete 事件，通过 handleTCBody 将子任务标记为 completed。
func (e *UpgradeExecutor) ExecuteOneUpload(ctx context.Context, subTask *UpgradeSubTask, fileType, targetFileNameTemplate, transportPath string) {
	dev, err := e.deviceRepo.GetByID(ctx, subTask.DeviceID)
	if err != nil {
		e.failSubTask(ctx, subTask, "Log collect can not be started, device not found.", FailureDeviceNotFound)
		return
	}

	// 设备在线检查
	if dev.Status != model.DeviceActive {
		e.logger.Info("device offline, log collect sub-task pending",
			zap.String("device_sn", dev.SerialNumber),
			zap.String("sub_task_id", subTask.ID.String()))
		waitKey := fmt.Sprintf("software:upgrade:wait:%s", dev.SerialNumber)
		e.redis.Set(ctx, waitKey, subTask.ID.String(), time.Hour)
		subTask.DeviceSN = dev.SerialNumber
		e.subTaskRepo.Update(ctx, subTask)
		e.subTaskRepo.UpdateStatus(ctx, subTask.ID, UpgradeSuspended, "waiting for device online")
		return
	}

	// 获取设备级锁
	acquired, err := e.acquireDeviceLock(ctx, dev.SerialNumber, subTask.ID)
	if err != nil {
		e.logger.Warn("acquire device lock failed for log collect, proceeding without lock",
			zap.String("device_sn", dev.SerialNumber), zap.Error(err))
	} else if !acquired {
		e.failLockedSubTask(ctx, subTask, dev.SerialNumber)
		return
	}

	subTask.DeviceSN = dev.SerialNumber
	if err := e.subTaskRepo.Update(ctx, subTask); err != nil {
		e.logger.Error("update log collect sub-task device info", zap.Error(err))
	}

	// 渲染 {OUI} —— 必须在所有 {OUI} 出现的字符串上都做替换，否则 CPE 会收到
	// 字面 "{OUI}" 的 FileType / URL，无法识别后回 FaultCode 直接拒收。
	// UFTE 内置 catalog 里 BACKUP / PATCH 类 FileType 都带 "{OUI}" 占位符（详见 ufte/model.go）。
	resolvedFileType := e.resolveOUIPlaceholders(dev.SerialNumber, dev.OUI, fileType)
	resolvedTransport := e.resolveOUIPlaceholders(dev.SerialNumber, dev.OUI, transportPath)

	// 构造目标文件名：替换 {task_id8} 和 {sn}
	task_id8 := taskIDPrefix(subTask.TaskID)
	targetFileName := resolveTemplate(targetFileNameTemplate, map[string]string{
		"task_id8": task_id8,
		"sn":       dev.SerialNumber,
	})

	// 拿运行时 ACS 上传 base URL（前端"系统管理 → ACS 传输 → 上传服务"维护）。
	// 优先级：transferProvider.Snapshot.Upload.BaseURL → SetUploadConfig 静态兜底 → 空。
	// 凭据不下发——产品线要求 Upload / Download 都不走 HTTP Basic Auth，CPE 拿到空 Username/Password 标签即可。
	uploadBaseURL := e.resolveUploadBaseURL(ctx)

	// 构造完整上传 URL：先解析 transportPath 模板，再拼接基础 URL。
	// 占位符 {sn} 与 {taskId} 必须渲染——UFTE 内置 transportPath 模板会带这两个
	// query 参数（ACS upload handler 用 sn 标识设备、taskId 关联 backup_tasks 行；
	// 详见 docs/project/backup-display-fix-20260520.md F14）。漏渲染将导致 ACS
	// 收到字面值 "{sn}"/"{taskId}" 误判设备身份。
	resolvedPath := resolveTemplate(resolvedTransport, map[string]string{
		"fileType":       resolvedFileType,
		"targetFileName": targetFileName,
		"sn":             dev.SerialNumber,
		"taskId":         subTask.TaskID.String(),
	})
	if uploadBaseURL == "" {
		// 没配置会让 URL 变成纯路径（"/smallcell/..."），CPE 拿到后无法解析为绝对地址；
		// 失败发生在 CPE 侧 → TransferComplete 永远不来 → 30min reaper 兜底。
		// 抛 warn 比静默拼空字符串好排查得多。
		e.logger.Warn("upload base URL is empty (check sys_config 'acs_transfer'.uploadBaseURL or YAML upgrade.acs_upload_base_url); CPE will receive a path-only URL and likely reject the upload",
			zap.String("device_sn", dev.SerialNumber),
			zap.String("sub_task_id", subTask.ID.String()))
	}
	uploadURL := uploadBaseURL + resolvedPath

	// CommandKey 格式必须对齐厂商私有约定，否则 baicells/MMMM 系列 CPE 拿到纯 UUID
	// 的 CommandKey 后**完全静默**（既不回 UploadResponse 也不上传文件，看似设备死机）。
	// 实测路径："12 ... Configuration File" / "10 ... Configuration File" 这类 NV/XML
	// 备份必须用 "Collect NV|<MFR>_<SN>,<UUID>" / "Collect XML|<MFR>_<SN>,<UUID>" 业务串；
	// 其他 FileType（日志采集 / 数据模型 upload）保持原 UUID 格式，避免影响存量流程。
	commandKey := deriveUploadCommandKey(resolvedFileType, dev.OUI, dev.SerialNumber, subTask.ID.String())
	// username / password 不传——产品线要求 Upload 不走 Basic Auth，
	// soap.UploadData 零值字段会渲染成空 <cwmp:Username></cwmp:Username>。
	paramsJSON, err := json.Marshal(map[string]interface{}{
		"command_key":      commandKey,
		"file_type":        resolvedFileType, // 已渲染 {OUI}，避免 CPE 收到字面占位符
		"url":              uploadURL,
		"target_file_name": targetFileName,
	})
	if err != nil {
		e.releaseDeviceLock(ctx, dev.SerialNumber)
		e.failSubTask(ctx, subTask, fmt.Sprintf("Log collect can not be started, internal error: %v", err), FailureInternalError)
		return
	}

	if _, err := e.cmdQueue.CreateTask(ctx, &devtask.CreateTaskRequest{
		DeviceSN:   dev.SerialNumber,
		Method:     "Upload",
		Params:     paramsJSON,
		Source:     devtask.TaskSourceSystem,
		CommandKey: commandKey,
	}); err != nil {
		e.releaseDeviceLock(ctx, dev.SerialNumber)
		e.failSubTask(ctx, subTask, "Log collect can not be started, failed to push Upload command.", FailureCommandPush)
		return
	}

	subTask.CommandKey = commandKey
	if err := e.subTaskRepo.Update(ctx, subTask); err != nil {
		e.logger.Error("update log collect sub-task command_key", zap.Error(err))
	}

	// 立即把 status 翻成 Uploading —— 必须放在 connReq 之前，否则会有 race：
	// 1) push Upload RPC 入队 → 2) connReq 3 次重试阻塞 ~30s → 3) UpdateStatus(uploading)
	// 期间设备如果**自然 Inform 周期触发** ACS PopTask 派发了 SOAP，设备秒级响应回 TC，
	// 这时 sub_task.status 还是 pending，handleTCBody 的"只处理在途状态"分支会
	// 直接 return，导致 fault 信息丢失、状态卡死。Upload RPC 走独立的 Uploading 状态
	// （不复用 Downloading）—— 历史拆分缘由详见 ufte/normalizeDeviceStatus、reaper SQL。
	if err := e.subTaskRepo.UpdateStatus(ctx, subTask.ID, UpgradeUploading, ""); err != nil {
		e.logger.Error("update log collect sub-task to uploading", zap.Error(err))
	}

	// Connection Request 唤醒设备 —— 异步发送，不阻塞 ExecuteOneUpload。
	// connReq 主要服务"设备空闲时主动拉取任务"场景，对正在 Inform 周期里的设备是冗余的。
	// 同步阻塞 30s 会跟上面的 UpdateStatus 串行，反而拖慢响应被 TC race condition 反咬。
	if dev.ConnectionRequestURL != "" {
		deviceSN := dev.SerialNumber
		connURL := dev.ConnectionRequestURL
		go func() {
			if err := e.connReq.Send(context.Background(), deviceSN, connURL); err != nil {
				e.logger.Warn("send connection request for log collect",
					zap.String("device_sn", deviceSN), zap.Error(err))
			}
		}()
	}

	e.logger.Info("log collect Upload RPC pushed",
		zap.String("sub_task_id", subTask.ID.String()),
		zap.String("device_sn", dev.SerialNumber),
		zap.String("device_oui", dev.OUI),
		zap.String("file_type_template", fileType),         // 原始模板，便于核对配置
		zap.String("file_type_resolved", resolvedFileType), // 真正给 CPE 的串
		zap.String("upload_url", uploadURL),
		zap.String("upload_base_url", uploadBaseURL), // 单独打 base，方便核对 sys_config 是否生效
		zap.String("target_file_name", targetFileName))
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

// HandleUploadResponse handles command.upload.response events.
// On fault → fail the sub-task. On success → wait for TransferComplete.
func (e *UpgradeExecutor) HandleUploadResponse(ctx context.Context, evt event.Event) error {
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

	// Upload 链路自 ExecuteOneUpload 起就处于 UpgradeUploading；这里看到非 Uploading
	// 的子任务说明事件迟到 / 误投递（比如已被 reaper 标失败），直接忽略。
	if subTask.Status != UpgradeUploading {
		return nil
	}

	if payload.FaultCode != 0 {
		e.failSubTask(ctx, subTask, fmt.Sprintf("Upload rejected by device. FaultCode: %d, FaultString: %s", payload.FaultCode, payload.FaultStr), FailureUploadFault)
		return nil
	}

	e.logger.Info("upload accepted, waiting for TransferComplete",
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
//
// 兼容三种事件 payload schema（2026-05-20 修复）：
//   - device.inform.periodic（ACS publish）→ `device_id.SerialNumber` 嵌套对象
//   - device.online（device 模块 publish）→ 顶层 `serial_number`
//   - 旧 / 测试代码 → 顶层 `device_sn`
// 详见 docs/project/backup-display-fix-20260520.md F12。
func (e *UpgradeExecutor) HandleDeviceOnline(ctx context.Context, evt event.Event) error {
	var payload struct {
		DeviceSN     string `json:"device_sn"`
		SerialNumber string `json:"serial_number"`
		DeviceID     struct {
			// tr069.DeviceId 的 json tag 是 lowercase `serial_number`，
			// 详见 pkg/tr069/types.go DeviceId 定义。
			SerialNumber string `json:"serial_number"`
		} `json:"device_id"`
	}
	if err := evt.DecodePayload(&payload); err != nil {
		return nil
	}
	deviceSN := payload.DeviceSN
	if deviceSN == "" {
		deviceSN = payload.SerialNumber
	}
	if deviceSN == "" {
		deviceSN = payload.DeviceID.SerialNumber
	}
	if deviceSN == "" {
		return nil
	}

	waitKey := fmt.Sprintf("software:upgrade:wait:%s", deviceSN)
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
		zap.String("device_sn", deviceSN))

	if subTask.FirmwareID != nil {
		fw, err := e.firmwareRepo.GetByID(ctx, *subTask.FirmwareID)
		if err != nil {
			e.failSubTask(ctx, subTask, fmt.Sprintf("Upgrade can not be started, the target version file can not be found: %v", err), FailureFirmwareGone)
			return nil
		}

		isKeepConfig := true
		downloadFileType := ""
		if task, err := e.taskRepo.GetByID(ctx, subTask.TaskID); err == nil {
			isKeepConfig = task.IsKeepConfig
			downloadFileType = task.DownloadFileType
		}

		// Reset status to pending for re-execution
		e.subTaskRepo.UpdateStatus(ctx, subTask.ID, UpgradePending, "")

		go e.ExecuteOne(context.Background(), subTask, fw, isKeepConfig, downloadFileType)
		return nil
	}

	// LogCollect 类（备份 / 日志采集 / 配置恢复）：FirmwareID=NULL，没有"固件下载"
	// 路径。通过 ufte.Service 注入的 Resumer 回调拿 transport_path 后重启 Upload RPC。
	// 详见 docs/project/backup-display-fix-20260520.md F6。
	if e.logCollectResumer == nil {
		e.logger.Warn("log collect sub-task awoke on device.online but resumer not configured; sub-task will remain suspended",
			zap.String("sub_task_id", subTask.ID.String()),
			zap.String("device_sn", deviceSN))
		return nil
	}
	parent, err := e.taskRepo.GetByID(ctx, subTask.TaskID)
	if err != nil {
		e.logger.Warn("get parent task on log collect online resume",
			zap.String("sub_task_id", subTask.ID.String()), zap.Error(err))
		return nil
	}
	// Reset to pending so ExecuteOneUpload 视为首次执行；下面 goroutine 失败时
	// status 会被相应 Failure 路径覆盖。
	if err := e.subTaskRepo.UpdateStatus(ctx, subTask.ID, UpgradePending, ""); err != nil {
		e.logger.Error("reset log collect sub-task to pending on resume", zap.Error(err))
	}
	go func() {
		defer func() {
			if r := recover(); r != nil {
				e.logger.Error("log collect resume panic",
					zap.String("sub_task_id", subTask.ID.String()), zap.Any("recover", r))
			}
		}()
		if err := e.logCollectResumer.ResumeLogCollectSubTask(context.Background(), subTask, parent); err != nil {
			e.logger.Error("resume log collect sub-task on device.online",
				zap.String("sub_task_id", subTask.ID.String()), zap.Error(err))
		}
	}()
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

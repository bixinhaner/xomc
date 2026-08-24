package software

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/acs/connreq"
	"github.com/omcgo/omcgo/internal/acs/transfercfg"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/device"
	devtask "github.com/omcgo/omcgo/internal/task"
	"github.com/omcgo/omcgo/pkg/tr069"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

const upsSoftwareVersionParamPath = "InternetGatewayDevice.DeviceInfo.SoftwareVersion"

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
	uploadResolver   transferAddressResolver
	downloadResolver transferAddressResolver
	acsUploadBaseURL string // CPE 可达的 ACS 上传服务基础 URL，如 "http://localhost:8080"
	// logCollectResumer 在设备上线时唤醒挂在 redis wait key 的 LogCollect 类子任务
	// （备份 / 日志采集，FirmwareID=NULL，没有"重新执行"所需的固件信息）。实现位于
	// ufte 包，避免 software 反向依赖 ufte 的 catalog——通过接口注入解耦。
	// nil-safe：未注入时 LogCollect 子任务上线唤醒退化为 no-op + 警告日志。
	logCollectResumer LogCollectResumer
	// pathTranslator 把 standardPath → privatePath（per-device），SPV 触发的
	// 日志采集（如 FAULT_LOG_COLLECT）下发 paramName 前必须翻译，否则不同
	// param_model 的设备会拒收。nil 退化为 passthrough。
	pathTranslator ParamPathTranslator
	// verifier 在固件 Download 下发前做完整性 / 签名校验（issue #8）。构造期默认
	// HashOnlyVerifier（强制固件具备 SHA-256 或回退 MD5 的完整性根）；DI 可经
	// SetFirmwareVerifier 换成 SignatureVerifier 叠加厂商验签。永不为 nil。
	verifier FirmwareVerifier
	// firmwareMetrics 记录校验 pass/fail/legacy_md5；nil-safe。
	firmwareMetrics *FirmwareMetrics
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

// ParamPathTranslator 把 standardPath 翻译为目标设备的 privatePath（per-device，按
// productClass + swVersion 路由 ProductRegistry → ParamRegistry → Translator）。
// 与 mml.PathTranslator 同款消费者驱动接口；CLAUDE.md §5.3 明确「模板 / SPV / GPV
// 输入侧用 standardPath，下发 / 持久化用 privatePath」，所有走 SPV 下发的链路都
// 必须先经此翻译，否则不同 param_model 的设备会拒收。
//
// 注入：cmd/app/provider/modules.go 用 mml_adapters.go 同款 adapter 装配；nil 表示
// 翻译禁用（向后兼容 / 单测），ExecuteOneSetParamCollect 退化为 passthrough。
type ParamPathTranslator interface {
	// TranslateForDevice 翻译一组 standardPaths。未命中 → Private=Standard，Source="passthrough"。
	// productClass 解析失败 → 返回 error；调用方退化到 passthrough 并 warn。
	TranslateForDevice(ctx context.Context, productClass, softwareVersion string, standardPaths []string) ([]TranslatedParamPath, error)
}

// TranslatedParamPath 是 ParamPathTranslator 的单条结果。
type TranslatedParamPath struct {
	Standard string
	Private  string
	Source   string // discovered / default / passthrough
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

type transferAddressResolver interface {
	Resolve(ctx context.Context, deviceID uuid.UUID, direction transfercfg.TransferDirection) (transfercfg.AddressDecision, error)
}

// SetUploadAddressResolver injects the unified HTTP/HTTPS address decision
// used by station log upload dispatch. Config backup upload remains on the
// legacy base-url path until its own rollout issue wires it in.
func (e *UpgradeExecutor) SetUploadAddressResolver(r transferAddressResolver) {
	e.uploadResolver = r
}

// SetDownloadAddressResolver injects the unified HTTP/HTTPS address decision
// used by IMG/PATCH/FPGA Download dispatch.
func (e *UpgradeExecutor) SetDownloadAddressResolver(r transferAddressResolver) {
	e.downloadResolver = r
}

// SetLogCollectResumer 注入 LogCollect 类子任务"设备上线即重试"的回调实现。
// 未注入时 HandleDeviceOnline 对 LogCollect 子任务退化为 no-op（保持原有 升级类 行为）。
func (e *UpgradeExecutor) SetLogCollectResumer(r LogCollectResumer) {
	e.logCollectResumer = r
}

// SetParamPathTranslator 注入 standardPath → privatePath 翻译器，供 SPV 触发的日志
// 采集（如 FAULT_LOG_COLLECT）下发前翻译参数名。nil 则保持 passthrough。
func (e *UpgradeExecutor) SetParamPathTranslator(t ParamPathTranslator) {
	e.pathTranslator = t
}

// HandleGetParamsResponseForRollback 桥接 command.get_parameters.response 事件到
// RollbackExecutor.HandleEnableCheckResponse —— 仅当 command_key 带 rollback enable
// 检查前缀时才推进，其它 GPV 响应（device_parameters 自动同步等）由其它订阅者处理。
//
// 注入：service.Subscribe 时把 rollbackExec 绑定到该 handler；payload 形态：
//
//	{
//	  "device_sn": "...",
//	  "command_key": "rollback-enable-check-<uuid>",
//	  "fault_code": 0 | 9xxx,
//	  "fault_string": "...",
//	  "parameter_values": [{"name": "...", "value": "1|true|0|false", "type": "..."}]
//	}
func (e *UpgradeExecutor) HandleGetParamsResponseForRollback(ctx context.Context, evt event.Event, rb *RollbackExecutor) error {
	if !softwareRollbackOwnsGPVResponse(evt) {
		return nil
	}
	var payload struct {
		DeviceSN        string                   `json:"device_sn"`
		CommandKey      string                   `json:"command_key"`
		FaultCode       int                      `json:"fault_code"`
		FaultStr        string                   `json:"fault_string"`
		ParameterValues []ParameterValueResponse `json:"parameter_values"`
	}
	if err := evt.DecodePayload(&payload); err != nil || payload.CommandKey == "" {
		return nil
	}
	rb.HandleEnableCheckResponse(ctx, payload.DeviceSN, payload.CommandKey, payload.FaultCode, payload.FaultStr, payload.ParameterValues)
	return nil
}

func softwareRollbackOwnsGPVResponse(evt event.Event) bool {
	owner := evt.Metadata[event.MetadataGPVOwner]
	return owner == "" || owner == event.GPVOwnerSoftwareRollback
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
		// 默认强制完整性校验（issue #8）；DI 可经 SetFirmwareVerifier 升级为带签名校验。
		verifier: NewHashOnlyVerifier(),
		logger:   logger.Named("upgrade-executor"),
	}
}

// SetFirmwareVerifier 替换固件下发前的完整性 / 签名校验器（issue #8）。
// 传 nil 退化为默认 HashOnlyVerifier，保证 verifier 永不为空、校验永不被绕过。
func (e *UpgradeExecutor) SetFirmwareVerifier(v FirmwareVerifier) {
	if v == nil {
		v = NewHashOnlyVerifier()
	}
	e.verifier = v
}

// SetFirmwareMetrics 注入固件校验 Prometheus 指标（nil-safe）。
func (e *UpgradeExecutor) SetFirmwareMetrics(m *FirmwareMetrics) {
	e.firmwareMetrics = m
}

// ExecuteOne runs the upgrade flow for a single sub-task.
// Flow: Step 1 (online check) → Step 2 (send Download cmd) → Step 3 (monitor download) → wait for events.
func (e *UpgradeExecutor) ExecuteOne(ctx context.Context, subTask *UpgradeSubTask, fw *FirmwareVersion, isKeepConfig bool, downloadFileType string) {
	e.executeOne(ctx, subTask, fw, isKeepConfig, downloadFileType)
}

func (e *UpgradeExecutor) executeOne(ctx context.Context, subTask *UpgradeSubTask, fw *FirmwareVersion, isKeepConfig bool, downloadFileType string) {
	// #59 Problem 3 紧急叫停（第一道）：ctx 已被取消（任务被 Suspend/Terminate/阈值暂停）
	// 时整批 goroutine 还没轮到执行就提前退出，绝不下发。ctx.Err() 非阻塞，比 select 更直白。
	if ctx.Err() != nil {
		e.logger.Info("upgrade dispatch skipped: execution canceled before start",
			zap.String("sub_task_id", subTask.ID.String()))
		return
	}

	dev, err := e.deviceRepo.GetByID(ctx, subTask.DeviceID)
	if err != nil {
		e.failSubTask(ctx, subTask, "Upgrade can not be started, device not found.", FailureDeviceNotFound)
		return
	}

	// Step 0: 固件完整性 / 签名校验（issue #8）——必须在派发 Download 命令之前。
	// 校验失败绝不静默放行：标 sub_task 失败 + 记 metric + 打日志，不下发。
	// 放在锁 / 命令推送之前，失败时无需释放任何已占资源。
	if err := e.verifyFirmware(subTask, fw, dev.SerialNumber); err != nil {
		return
	}

	// Step 1: Device online check
	if dev.Status != model.DeviceActive {
		e.logger.Info("device offline, entering wait state",
			zap.String("device_sn", dev.SerialNumber),
			zap.String("sub_task_id", subTask.ID.String()))

		// Set Redis wait key for HandleDeviceOnline to resume
		// Redis wait key TTL = 10min，跟 reaper.DeviceOnline 阈值保持一致：
		// 设备 10min 内不上线 → reaper 标 sub_task failed，wait key 也同时过期
		// 避免出现 wait key 过期不会自动唤醒、reaper 又没兜底失败的中间态。
		waitKey := fmt.Sprintf("software:upgrade:wait:%s", dev.SerialNumber)
		e.redis.Set(ctx, waitKey, subTask.ID.String(), 10*time.Minute)

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

	// #59 Problem 3 紧急叫停（第二道，权威）：拿到设备锁、即将构造并下发 Download 之前，
	// 回库复查 sub_task 与 parent task 的最新状态。SuspendUpgrade 只改 DB 不动在飞
	// goroutine（历史 bug），而本 goroutine 可能在 cancel 之前就越过了第一道 ctx 检查、
	// 卡在 30s connReq / 锁竞争上；等它继续往下时 cancel 信号或 DB 暂停状态可能已经写入。
	// 这道 DB 复查是兜底防线：只要子任务已被挂起 / 终止，或父任务已挂起，就释放锁并退出，
	// 不再 enqueue Download。也再看一次 ctx.Err()，覆盖锁后才发生的 cancel。
	if e.shouldAbortDispatch(ctx, subTask) {
		// 用 detached ctx 释放锁：此刻 ctx 很可能已被急停 cancel，带它调 Redis Del 会
		// 因 context canceled 失败、把设备锁留到 1h TTL 才过期，挡住后续重试。
		e.releaseDeviceLock(context.Background(), dev.SerialNumber, subTask.ID)
		e.logger.Info("upgrade dispatch aborted: sub-task suspended/terminated before download",
			zap.String("sub_task_id", subTask.ID.String()),
			zap.String("device_sn", dev.SerialNumber))
		return
	}

	// Step 2: Build and push Download command
	commandKey := e.adapter.DownloadCommandKey(subTask.ID.String())
	downloadURL := "firmware/" + fw.MinIOPath
	effectiveDownloadFileType := downloadFileType
	if effectiveDownloadFileType == "" {
		effectiveDownloadFileType = e.adapter.DownloadFileType(fw.FileType)
	}
	transferPolicyManaged := shouldResolveManagedFirmwareDownloadURL(fw) && e.downloadResolver != nil
	if transferPolicyManaged {
		resolvedURL, err := e.resolveManagedFirmwareDownloadURL(ctx, dev.ID, fw)
		if err != nil {
			e.releaseDeviceLock(context.Background(), dev.SerialNumber, subTask.ID)
			e.failSubTask(ctx, subTask, fmt.Sprintf("Upgrade can not be started, invalid download URL: %v", err), FailureInternalError)
			return
		}
		downloadURL = resolvedURL
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
		// ACS SOAP 渲染时用这个标记区分 OMC 内部 FileDownloadService 地址和外部厂商 URL。
		"transfer_policy_managed": transferPolicyManaged,
	})
	if err != nil {
		e.releaseDeviceLock(ctx, dev.SerialNumber, subTask.ID)
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
		e.releaseDeviceLock(ctx, dev.SerialNumber, subTask.ID)
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

func shouldResolveManagedFirmwareDownloadURL(fw *FirmwareVersion) bool {
	if fw == nil {
		return false
	}
	switch fw.FileType {
	case FileTypeIMG, FileTypePATCH, FileTypeAP, FileTypeFPGA:
		return true
	default:
		return false
	}
}

func (e *UpgradeExecutor) resolveManagedFirmwareDownloadURL(ctx context.Context, deviceID uuid.UUID, fw *FirmwareVersion) (string, error) {
	if e.downloadResolver == nil {
		return "", fmt.Errorf("download address resolver is not configured")
	}
	decision, err := e.downloadResolver.Resolve(ctx, deviceID, transfercfg.TransferDirectionDownload)
	if err != nil {
		return "", fmt.Errorf("resolve download address: %w", err)
	}

	servicePath := ""
	if e.transferProvider != nil {
		servicePath = e.transferProvider.Snapshot(ctx).Download.Path
	}
	segments, err := firmwareDownloadObjectSegments(fw.MinIOPath)
	if err != nil {
		return "", err
	}
	downloadURL, err := transfercfg.BuildURL(decision.BaseURL, servicePath, segments, nil)
	if err != nil {
		return "", fmt.Errorf("build download URL: %w", err)
	}
	return downloadURL, nil
}

func firmwareDownloadObjectSegments(minioPath string) ([]string, error) {
	trimmed := strings.Trim(minioPath, "/")
	if trimmed == "" {
		return nil, fmt.Errorf("firmware object path is empty")
	}
	parts := strings.Split(trimmed, "/")
	segments := make([]string, 0, len(parts)+1)
	if parts[0] != "firmware" {
		segments = append(segments, "firmware")
	}
	for _, part := range parts {
		if part == "" {
			return nil, fmt.Errorf("firmware object path contains an empty segment")
		}
		segments = append(segments, part)
	}
	return segments, nil
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

// imsParamUploadFileName 生成 IMS 采集的目标文件名：
// paramType 去 FT_ImsCore_ 前缀 + 年月日时分秒，如
// FT_ImsCore_User_Setting_UD → User_Setting_UD_20260820153201.dat；
// 日志类（*_Logs_U）用 .log 后缀。无法识别前缀时用原值兜底。
// 日志判断本地实现（后缀 _Logs_U）——imsparam → software 已有依赖，反向 import 成环。
func imsParamUploadFileName(paramType string, now time.Time) string {
	trimmed := strings.TrimSpace(paramType)
	name := strings.TrimPrefix(trimmed, "FT_ImsCore_")
	ext := ".dat"
	if strings.HasSuffix(trimmed, "_Logs_U") {
		ext = ".log"
	}
	return fmt.Sprintf("%s_%s%s", name, now.Format("20060102150405"), ext)
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
// 用纯 UUID 当 CommandKey 时设备识别不出这是 NV/XML/LOG 备份请求，会直接静默丢弃
// Upload RPC（既不回 UploadResponse 也不向 FileUploadService 发 PUT）。
//
// 真实工作样本（来自厂商抓包）：
//   - NV 备份：CommandKey = "Collect NV|<OUI>_<SN>,<sub_task_uuid>"     （FileType 第一段 "12"）
//   - XML 备份：CommandKey = "Collect XML|<OUI>_<SN>,<sub_task_uuid>"   （FileType 第一段 "10"，保守按 NV 同款）
//   - LOG 收集：CommandKey = "Collect LOG,<uuid_前13字符>"               （FileType 第一段 "4"，无 OUI/SN 段，UUID 截断短串）
//   - 其它 FileType（如数据模型 upload）：保持原 UUID，避免影响存量已工作流程
//
// fileType 形如 "10 48BF74 Configuration File" / "12 48BF74 Configuration File" /
// "4 Vendor Log File 1,2,3,4"——以第一个 token 的数字部分区分。
func deriveUploadCommandKey(resolvedFileType, oui, serialNumber, subTaskID string, runtimeLog bool) string {
	if runtimeLog {
		short := subTaskID
		if len(short) > 13 {
			short = short[:13]
		}
		return fmt.Sprintf("Collect LOG,%s", short)
	}
	parts := strings.SplitN(strings.TrimSpace(resolvedFileType), " ", 2)
	if len(parts) < 2 {
		return subTaskID
	}
	switch parts[0] {
	case "12":
		ouiKey := oui
		if ouiKey == "" {
			ouiKey = fallbackOUI
		}
		return fmt.Sprintf("Collect NV|%s_%s,%s", ouiKey, serialNumber, subTaskID)
	case "10":
		ouiKey := oui
		if ouiKey == "" {
			ouiKey = fallbackOUI
		}
		return fmt.Sprintf("Collect XML|%s_%s,%s", ouiKey, serialNumber, subTaskID)
	default:
		return subTaskID
	}
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

func encodeQueryTemplateValue(value string) string {
	return url.QueryEscape(value)
}

func (e *UpgradeExecutor) resolveLogUploadAddress(
	ctx context.Context,
	dev *model.Device,
) (string, transfercfg.AddressDecision, error) {
	if e.uploadResolver == nil {
		baseURL := e.resolveUploadBaseURL(ctx)
		return baseURL, transfercfg.AddressDecision{
			Direction:  transfercfg.TransferDirectionUpload,
			Protocol:   transfercfg.TransferProtocolHTTP,
			BaseURL:    baseURL,
			Capability: transfercfg.HTTPSCapabilityNotRead,
			Reason:     transfercfg.AddressReasonForceHTTP,
		}, nil
	}
	decision, err := e.uploadResolver.Resolve(ctx, dev.ID, transfercfg.TransferDirectionUpload)
	if err != nil {
		return "", transfercfg.AddressDecision{}, err
	}
	return strings.TrimRight(decision.BaseURL, "/"), decision, nil
}

func isRuntimeLogUpload(fileType, transportPath string) bool {
	normalized := strings.ToUpper(strings.TrimSpace(fileType))
	normalizedPath := strings.ToUpper(strings.TrimSpace(transportPath))
	return normalized == "4" ||
		strings.Contains(normalized, "VENDOR LOG FILE") ||
		strings.Contains(normalizedPath, "FILETYPE=LOG")
}

// isImsParamUpload 判断是否核心网文件采集（TransportPath 含 fileType=IMS_PARAM）。
// 独立判断而非并入 isRuntimeLogUpload：避免核心网任务被误标 runtimeLog 而改变
// CommandKey 派生（"Collect LOG," 前缀）与 reaper 语义。仅用于决定是否走
// 传输地址决策链路（与运行日志/配置备份一致）。
func isImsParamUpload(fileType, transportPath string) bool {
	return strings.Contains(strings.ToUpper(strings.TrimSpace(transportPath)), "FILETYPE=IMS_PARAM")
}

func isConfigBackupUpload(fileType, transportPath string) bool {
	normalized := strings.ToUpper(strings.TrimSpace(fileType))
	normalizedPath := strings.ToUpper(strings.TrimSpace(transportPath))
	return strings.Contains(normalized, "CONFIGURATION FILE") ||
		strings.Contains(normalizedPath, "FILETYPE=CONFIGBACKUP_XML") ||
		strings.Contains(normalizedPath, "FILETYPE=CONFIGBACKUP_NV")
}

func buildTransferUploadURL(baseURL, resolvedTransport string) (string, error) {
	return transfercfg.BuildTemplateURL(baseURL, resolvedTransport)
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
//
// fileType 是任务行 download_file_type 落库值：IMS 参数采集为
// "ImsCore Parameters File:FT_ImsCore_*"（CWMP 字面值 + ParamType 尾段），此处 split 后
// 真正下发 CWMP 的 FileType 是 "ImsCore Parameters File"，ParamType 单独塞进
// SOAP 报文的 <ParamType> 标签（见 soap.UploadData.ParamType）。
func (e *UpgradeExecutor) ExecuteOneUpload(ctx context.Context, subTask *UpgradeSubTask, fileType, targetFileNameTemplate, transportPath string) {
	cwmpFileType, paramType := splitStoredFileType(fileType)
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
		// Redis wait key TTL = 10min，跟 reaper.DeviceOnline 阈值保持一致：
		// 设备 10min 内不上线 → reaper 标 sub_task failed，wait key 也同时过期
		// 避免出现 wait key 过期不会自动唤醒、reaper 又没兜底失败的中间态。
		waitKey := fmt.Sprintf("software:upgrade:wait:%s", dev.SerialNumber)
		e.redis.Set(ctx, waitKey, subTask.ID.String(), 10*time.Minute)
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
	resolvedFileType := e.resolveOUIPlaceholders(dev.SerialNumber, dev.OUI, cwmpFileType)
	resolvedTransport := e.resolveOUIPlaceholders(dev.SerialNumber, dev.OUI, transportPath)

	// 构造目标文件名：替换 {task_id8} 和 {sn}
	task_id8 := taskIDPrefix(subTask.TaskID)
	targetFileName := resolveTemplate(targetFileNameTemplate, map[string]string{
		"task_id8": task_id8,
		"sn":       dev.SerialNumber,
	})
	// IMS 采集类（参数/日志）：目标文件名固定为 "类型去 FT_ImsCore_ 前缀_年月日时分秒
	// + .dat/.log"（如 Policy_Setting_UD_20260820153201.dat / Operation_Logs_U_...log），
	// 不依赖 catalog 模板值 —— 既有库 ufte_task_types 里残留的旧模板不影响文件名格式。
	// License/恢复采集无子类型尾段，沿用模板渲染。
	if paramType != "" {
		targetFileName = imsParamUploadFileName(paramType, time.Now())
	}

	// 拿运行时 ACS 上传 base URL（前端"系统管理 → ACS 传输 → 上传服务"维护）。
	// 优先级：transferProvider.Snapshot.Upload.BaseURL → SetUploadConfig 静态兜底 → 空。
	// 凭据不下发——产品线要求 Upload / Download 都不走 HTTP Basic Auth，CPE 拿到空 Username/Password 标签即可。
	runtimeLogUpload := isRuntimeLogUpload(resolvedFileType, resolvedTransport)
	useTransferDecision := runtimeLogUpload || isConfigBackupUpload(resolvedFileType, resolvedTransport) ||
		isImsParamUpload(resolvedFileType, resolvedTransport)
	uploadBaseURL := e.resolveUploadBaseURL(ctx)
	transferDecision := transfercfg.AddressDecision{
		Direction:  transfercfg.TransferDirectionUpload,
		Protocol:   transfercfg.TransferProtocolHTTP,
		BaseURL:    uploadBaseURL,
		Capability: transfercfg.HTTPSCapabilityNotRead,
		Reason:     transfercfg.AddressReasonForceHTTP,
	}
	if useTransferDecision {
		var resolveErr error
		uploadBaseURL, transferDecision, resolveErr = e.resolveLogUploadAddress(ctx, dev)
		if resolveErr != nil {
			e.releaseDeviceLock(ctx, dev.SerialNumber, subTask.ID)
			e.failSubTask(ctx, subTask, fmt.Sprintf("Log collect can not be started, resolve upload address failed: %v", resolveErr), FailureInternalError)
			return
		}
	}

	// 构造完整上传 URL：先解析 transportPath 模板，再拼接基础 URL。
	// 占位符 {sn} 与 {taskId} 必须渲染——UFTE 内置 transportPath 模板会带这两个
	// query 参数（ACS upload handler 用 sn 标识设备、taskId 关联 backup_tasks 行；
	// 详见 docs/project/backup-display-fix-20260520.md F14）。漏渲染将导致 ACS
	// 收到字面值 "{sn}"/"{taskId}" 误判设备身份。
	resolvedPath := resolveTemplate(resolvedTransport, map[string]string{
		"fileType":       resolvedFileType,
		"targetFileName": targetFileName,
		"sn":             encodeQueryTemplateValue(dev.SerialNumber),
		"taskId":         subTask.TaskID.String(),
		// {taskId32}: UUID 去连字符的 32 字符纯 hex 形式。
		// 厂商 baicells/MMMM 真实样本的 Upload URL 用这种格式（详见 migrations/000141
		// 运行日志收集对齐），普通 {taskId} (含连字符 36 字符) 用于 NV/XML 备份等保持兼容。
		"taskId32": strings.ReplaceAll(subTask.TaskID.String(), "-", ""),
	})
	uploadURL, err := buildTransferUploadURL(uploadBaseURL, resolvedPath)
	if err != nil {
		e.releaseDeviceLock(ctx, dev.SerialNumber, subTask.ID)
		e.failSubTask(ctx, subTask, fmt.Sprintf("Log collect can not be started, invalid upload URL: %v", err), FailureInternalError)
		return
	}

	// CommandKey 格式必须对齐厂商私有约定，否则 baicells/MMMM 系列 CPE 拿到纯 UUID
	// 的 CommandKey 后**完全静默**（既不回 UploadResponse 也不上传文件，看似设备死机）。
	// 实测路径："12 ... Configuration File" / "10 ... Configuration File" 这类 NV/XML
	// 备份必须用 "Collect NV|<MFR>_<SN>,<UUID>" / "Collect XML|<MFR>_<SN>,<UUID>" 业务串；
	// 其他 FileType（日志采集 / 数据模型 upload）保持原 UUID 格式，避免影响存量流程。
	commandKey := deriveUploadCommandKey(resolvedFileType, dev.OUI, dev.SerialNumber, subTask.ID.String(), runtimeLogUpload)
	// username / password 不传——产品线要求 Upload 不走 Basic Auth，
	// soap.UploadData 零值字段会渲染成空 <cwmp:Username></cwmp:Username>。
	params := map[string]interface{}{
		"command_key":      commandKey,
		"file_type":        resolvedFileType, // 已渲染 {OUI}，避免 CPE 收到字面占位符
		"url":              uploadURL,
		"target_file_name": targetFileName,
	}
	// IMS 采集类子类型标签：统一进 <ParameterType>（两个任务模板合并后日志类
	// 也走同一标签）。非空才带，其它任务报文结构不变。
	if paramType != "" {
		params["param_type"] = paramType
	}
	paramsJSON, err := json.Marshal(params)
	if err != nil {
		e.releaseDeviceLock(ctx, dev.SerialNumber, subTask.ID)
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
		e.releaseDeviceLock(ctx, dev.SerialNumber, subTask.ID)
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

	logFields := []zap.Field{
		zap.String("sub_task_id", subTask.ID.String()),
		zap.String("device_sn", dev.SerialNumber),
		zap.String("device_oui", dev.OUI),
		zap.String("file_type_template", fileType),         // 原始模板，便于核对配置
		zap.String("file_type_resolved", resolvedFileType), // 真正给 CPE 的串
		zap.String("target_file_name", targetFileName),
	}
	if useTransferDecision {
		logFields = append(logFields,
			zap.String("transfer_protocol", string(transferDecision.Protocol)),
			zap.String("transfer_reason", string(transferDecision.Reason)),
			zap.String("https_capability", string(transferDecision.Capability)),
		)
	} else {
		logFields = append(logFields,
			zap.String("upload_url", uploadURL),
			zap.String("upload_base_url", uploadBaseURL), // 单独打 base，方便核对 sys_config 是否生效
		)
	}
	e.logger.Info("log collect Upload RPC pushed", logFields...)
}

// ExecuteOneSetParamCollect 通过 SetParameterValues 触发设备主动上传日志文件。
//
// 与 ExecuteOneUpload 的区别：不走 TR-069 Upload RPC（厂商私有 ACS 协议未实现 Upload），
// 而是把 ACS 端 FileUploadService 的完整 URL 写入设备私有参数（如
// Device.DeviceInfo.X_COM_Log.FaultLogURL）。设备收到 SPV 后会异步把日志 HTTP PUT 到该 URL，
// 整体效果等价于厂商触发的"反向上传"。
//
// paramPath  → SetParameterValues 的参数名，如 "Device.DeviceInfo.X_COM_Log.FaultLogURL"
// transportPath → 含 {id}/{sn} 占位符的 URL 路径模板，渲染后拼到 uploadBaseURL 前面
//
// 完成路径：CPE 上传文件 → ACS 端 upload handler 落 MinIO → 发布 backup.file.received →
// SoftwareService.HandleFileLandedForCollect 把 sub_task 标 Completed。
func (e *UpgradeExecutor) ExecuteOneSetParamCollect(ctx context.Context, subTask *UpgradeSubTask, paramPath, transportPath string) {
	dev, err := e.deviceRepo.GetByID(ctx, subTask.DeviceID)
	if err != nil {
		e.failSubTask(ctx, subTask, "Log collect can not be started, device not found.", FailureDeviceNotFound)
		return
	}

	// 设备在线检查（与 ExecuteOneUpload 同款）：离线 → 挂 wait key 等 device.online 唤醒。
	if dev.Status != model.DeviceActive {
		waitKey := fmt.Sprintf("software:upgrade:wait:%s", dev.SerialNumber)
		e.redis.Set(ctx, waitKey, subTask.ID.String(), 10*time.Minute)
		subTask.DeviceSN = dev.SerialNumber
		e.subTaskRepo.Update(ctx, subTask)
		e.subTaskRepo.UpdateStatus(ctx, subTask.ID, UpgradeSuspended, "waiting for device online")
		return
	}

	acquired, err := e.acquireDeviceLock(ctx, dev.SerialNumber, subTask.ID)
	if err != nil {
		e.logger.Warn("acquire device lock failed for set-param collect, proceeding without lock",
			zap.String("device_sn", dev.SerialNumber), zap.Error(err))
	} else if !acquired {
		e.failLockedSubTask(ctx, subTask, dev.SerialNumber)
		return
	}

	subTask.DeviceSN = dev.SerialNumber
	if err := e.subTaskRepo.Update(ctx, subTask); err != nil {
		e.logger.Error("update set-param collect sub-task device info", zap.Error(err))
	}

	// 渲染上传 URL。{id} 用主任务 UUID，ACS upload handler 后续用它发布事件并
	// 写入 backup_restore_file.task_id，便于按 (sn, parent_task_id) 反查真实落地文件。
	// {sn} 用设备序列号。{fileName} 留空让设备自己决定上传名（与现网 Upload 链路一致）。
	uploadBaseURL, transferDecision, resolveErr := e.resolveLogUploadAddress(ctx, dev)
	if resolveErr != nil {
		e.releaseDeviceLock(ctx, dev.SerialNumber, subTask.ID)
		e.failSubTask(ctx, subTask, fmt.Sprintf("Log collect can not be started, resolve upload address failed: %v", resolveErr), FailureInternalError)
		return
	}
	resolvedPath := resolveTemplate(transportPath, map[string]string{
		"id": subTask.TaskID.String(),
		"sn": encodeQueryTemplateValue(dev.SerialNumber),
	})
	uploadURL, err := buildTransferUploadURL(uploadBaseURL, resolvedPath)
	if err != nil {
		e.releaseDeviceLock(ctx, dev.SerialNumber, subTask.ID)
		e.failSubTask(ctx, subTask, fmt.Sprintf("Log collect can not be started, invalid upload URL: %v", err), FailureInternalError)
		return
	}

	// standardPath → privatePath 翻译（per-device，按 productClass + swVersion 路由
	// ProductRegistry → ParamRegistry → Translator）。UFTE 模板里 url_template 字段
	// 存的是 standardPath（如 Device.DeviceInfo.FaultLogURL），实际下发给 CPE 的
	// 必须是该 param_model 对应的 privatePath（不同型号映射可能不一样，详见
	// CLAUDE.md §5.3 与 mml/service.go 同款翻译链路）。翻译器未注入或翻译失败 →
	// passthrough 用原 standardPath，跟旧行为兼容。
	dispatchPath := paramPath
	if e.pathTranslator != nil && paramPath != "" {
		translated, terr := e.pathTranslator.TranslateForDevice(ctx, dev.ProductClass, dev.FirmwareVersion, []string{paramPath})
		switch {
		case terr != nil:
			e.logger.Warn("path translator failed; using standardPath as-is (passthrough)",
				zap.String("device_sn", dev.SerialNumber),
				zap.String("product_class", dev.ProductClass),
				zap.String("standard_path", paramPath),
				zap.Error(terr))
		case len(translated) > 0:
			// 容错：翻译表里若存在 standardPath → "" 这类脏数据（不应发生但要兜底），
			// 不能把空字符串当作合法 privatePath 下发——CPE 会回 "Empty parameter list"。
			// 此处直接退化为 standardPath passthrough，并 WARN 出来便于查脏数据。
			if translated[0].Private == "" {
				e.logger.Warn("translator returned empty privatePath; falling back to standardPath",
					zap.String("device_sn", dev.SerialNumber),
					zap.String("product_class", dev.ProductClass),
					zap.String("software_version", dev.FirmwareVersion),
					zap.String("standard_path", translated[0].Standard),
					zap.String("source", translated[0].Source))
			} else {
				dispatchPath = translated[0].Private
			}
			e.logger.Info("path translated for SPV collect",
				zap.String("device_sn", dev.SerialNumber),
				zap.String("product_class", dev.ProductClass),
				zap.String("software_version", dev.FirmwareVersion),
				zap.String("standard_path", translated[0].Standard),
				zap.String("private_path", translated[0].Private),
				zap.String("dispatch_path", dispatchPath),
				zap.String("source", translated[0].Source))
		}
	}

	// 最终防线：dispatchPath 必须非空。SOAP 渲染会照搬 Name 字段，空串
	// 直接进 <Name></Name> → CPE 回 "Empty parameter list"，任务报错后还
	// 不知道为啥失败。此处直接 fail，并把 productClass / standardPath 全打到
	// failure_reason 里，便于运维查脏数据 / 漏映射。
	if dispatchPath == "" {
		e.releaseDeviceLock(ctx, dev.SerialNumber, subTask.ID)
		msg := fmt.Sprintf("SPV log-collect aborted: empty dispatch path (productClass=%s, fw=%s, standardPath=%q). Check param_mappings or task_type url_template.",
			dev.ProductClass, dev.FirmwareVersion, paramPath)
		e.logger.Error(msg, zap.String("device_sn", dev.SerialNumber))
		e.failSubTask(ctx, subTask, msg, FailureInternalError)
		return
	}

	// 构造 SetParameterValues 命令：参数 = dispatchPath（翻译后的 privatePath），值 = uploadURL。
	// CommandKey 用 sub_task ID，后续 set_parameters.response 事件按 command_key 反查 sub_task。
	paramsJSON, err := json.Marshal(map[string]interface{}{
		"values": []map[string]string{
			{
				"name":  dispatchPath,
				"value": uploadURL,
				"type":  "xsd:string",
			},
		},
		"command_key": subTask.ID.String(),
	})
	if err != nil {
		e.releaseDeviceLock(ctx, dev.SerialNumber, subTask.ID)
		e.failSubTask(ctx, subTask, fmt.Sprintf("Log collect can not be started, internal error: %v", err), FailureInternalError)
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
		e.failSubTask(ctx, subTask, "Log collect can not be started, failed to push SetParameterValues command.", FailureCommandPush)
		return
	}

	subTask.CommandKey = subTask.ID.String()
	if err := e.subTaskRepo.Update(ctx, subTask); err != nil {
		e.logger.Error("update set-param collect sub-task command_key", zap.Error(err))
	}

	// 状态翻到 Uploading —— 必须在 connReq 之前，详见 ExecuteOneUpload 同名注释。
	// SPV 走和 Upload 同款的 Uploading 状态：UI 通过 fileLanded 判定是 uploading / awaiting_tc，
	// 文件落地后 HandleFileLandedForCollect 推进到 Completed。
	if err := e.subTaskRepo.UpdateStatus(ctx, subTask.ID, UpgradeUploading, ""); err != nil {
		e.logger.Error("update set-param collect sub-task to uploading", zap.Error(err))
	}

	if dev.ConnectionRequestURL != "" {
		deviceSN := dev.SerialNumber
		connURL := dev.ConnectionRequestURL
		go func() {
			if err := e.connReq.Send(context.Background(), deviceSN, connURL); err != nil {
				e.logger.Warn("send connection request for set-param collect",
					zap.String("device_sn", deviceSN), zap.Error(err))
			}
		}()
	}

	e.logger.Info("set-param collect SPV pushed",
		zap.String("sub_task_id", subTask.ID.String()),
		zap.String("device_sn", dev.SerialNumber),
		zap.String("standard_path", paramPath),
		zap.String("dispatch_path", dispatchPath),
		zap.String("transfer_protocol", string(transferDecision.Protocol)),
		zap.String("transfer_reason", string(transferDecision.Reason)),
		zap.String("https_capability", string(transferDecision.Capability)))
}

// HandleSetParamsResponse 处理 SetParameterValues 响应事件。SPV 在 OMC 内有两条
// 业务使用路径，按 parent.TaskType + sub_task.Status 分发：
//
//   - LogCollect / Uploading：FAULT_LOG_COLLECT 等 SPV 触发的日志采集
//     · fault → fail，记 UPLOAD_FAULT
//     · success → 保持 Uploading 等文件落地（OnLogFileLanded hook 推进）
//
//   - Rollback / Rebooting：基站版本回退（adapter 决定 4G/5G 路径与值）
//     · fault → fail，记 ROLLBACK_SET_FAULT（避免 30 min 后才被 reaper 兜底）
//     · success → 保持 Rebooting 等 reboot_complete 事件
//
// 历史教训：原版只服务 LogCollect 一条路径（带 parent.TaskType==LogCollect 过滤），
// rollback 路径的 SPV Fault 拿不到回调，sub_task 在 Rebooting 状态卡 30 min 直到
// reaper 兜底，UI 上"失败原因"也只显示模糊的"等 TC 超时"，丢失真实 CPE 错误。
func (e *UpgradeExecutor) HandleSetParamsResponse(ctx context.Context, evt event.Event) error {
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
	parent, err := e.taskRepo.GetByID(ctx, subTask.TaskID)
	if err != nil {
		return nil
	}

	// 判失败：fault_code 非 0 → 标准 CWMP fault；fault_code 为 0 但 fault_string 非空 →
	// 兼容用 SOAP 1.1 <faultcode>Server.Internal</faultcode> 替代 <cwmp:FaultCode> 的厂商
	// （如 baicells/FAP，回 "RPC handler failed: Empty parameter list"），ACS 解出 code=0
	// 但 string 携带真实原因。任一非空都视为失败，把原因落到 sub_task.error_message。
	isFault := isCPEFault(payload.FaultCode, payload.FaultStr)

	switch parent.TaskType {
	case TaskTypeLogCollect:
		if subTask.Status != UpgradeUploading {
			return nil
		}
		if isFault {
			e.failSubTask(ctx, subTask,
				fmt.Sprintf("SetParameterValues rejected by device. FaultCode: %d, FaultString: %s",
					payload.FaultCode, payload.FaultStr),
				FailureUploadFault)
			return nil
		}
		e.logger.Info("SPV accepted, waiting for log file upload",
			zap.String("sub_task_id", subTask.ID.String()),
			zap.String("device_sn", payload.DeviceSN))

	case TaskTypeRollback:
		// rollback 路径 RollbackOne 推送 SPV 后立刻把 sub_task 翻到 Rebooting；
		// 这里只在该状态接受 fault 推进，避免重复处理或脏推送（reaper / 多副本场景）。
		if subTask.Status != UpgradeRebooting {
			return nil
		}
		if isFault {
			e.failSubTask(ctx, subTask,
				fmt.Sprintf("Rollback failed, device rejected SetParameterValues. FaultCode: %d, FaultString: %s",
					payload.FaultCode, payload.FaultStr),
				FailureRollbackSetFault)
			return nil
		}
		e.logger.Info("SPV accepted, waiting for device reboot",
			zap.String("sub_task_id", subTask.ID.String()),
			zap.String("device_sn", payload.DeviceSN))

	default:
		// 其它 TaskType（升级 / 配置备份等）不通过 SPV 单独触发，无需处理。
		return nil
	}
	return nil
}

// isCPEFault 综合判定 CPE 是否真的失败。
// 单看 fault_code != 0 不够：部分厂商 CPE（如 baicells）SOAP Fault 用 SOAP 1.1 的
// <faultcode>Server.Internal</faultcode>（字符串）而不是 CWMP 标准的
// <cwmp:FaultCode>9xxx</cwmp:FaultCode>（数值），ACS 解出来 fault_code=0，但 fault_string
// 仍然带真实错误描述（"RPC handler failed: ..."）。此时若只看 code 会把失败当成功，
// sub_task 永远卡在 in_flight 状态。
func isCPEFault(faultCode int, faultStr string) bool {
	return faultCode != 0 || strings.TrimSpace(faultStr) != ""
}

// HandleFileLandedForCollect 处理 backup.file.received 事件，按 (device_sn, parent_task_id)
// 反查在途 LogCollect 子任务，把状态推进到 Completed。
//
// 触发场景：FAULT_LOG_COLLECT 通过 SPV 让设备把日志 PUT 到 ACS 端 FileUploadService。
// 设备 PUT 完成 → upload handler 写 MinIO → 发 backup.file.received（含 task_id 与 device_sn）。
//
// 与 TC 路径并存：传统 Upload RPC 链路（RUNTIME_LOG_COLLECT）走 TransferComplete →
// handleTCBody 推进。该路径下也会进到这里，但因 sub_task 已被 TC 标 Completed，
// GetActiveByDeviceID 拿不到在途任务 → 自然 no-op。
func (e *UpgradeExecutor) HandleFileLandedForCollect(ctx context.Context, evt event.Event) error {
	var payload struct {
		DeviceSN string `json:"device_sn"`
		TaskID   string `json:"task_id"`
	}
	if err := evt.DecodePayload(&payload); err != nil {
		return nil
	}
	if payload.DeviceSN == "" || payload.TaskID == "" {
		return nil
	}
	parentID, err := uuid.Parse(payload.TaskID)
	if err != nil {
		return nil
	}
	parent, err := e.taskRepo.GetByID(ctx, parentID)
	if err != nil || parent == nil || parent.TaskType != TaskTypeLogCollect {
		return nil
	}

	dev, err := e.deviceRepo.GetBySerialNumber(ctx, payload.DeviceSN)
	if err != nil || dev == nil {
		return nil
	}
	// 按 (parent_task_id, device_id) 精确查 sub_task，不能用 GetActiveByDeviceID。
	// 后者的 fan-out 顺序是 upgrade → config_backup → runtime_log → fault_log → config_restore；
	// 一旦设备在前面的表里残留 pending sub_task（多业务表并存场景实测会发生：
	// 老的 CONFIG_BACKUP / CONFIG_RESTORE pending 没清理），fan-out 直接返回那个，
	// 永远走不到 fault_log_collect_sub_tasks → 当前 SPV 任务的 sub_task 推不动 →
	// 反器 15 分钟后兜底标 failed → 用户看到"Awaiting TransferComplete"15 分钟后变失败。
	// 翻页取全量子任务再按 device_id 反查：默认页只取前 20 台，>20 台的任务里
	// 超页设备文件落地后会找不到 sub_task → 推不动 → 15 分钟后被反器误标失败。
	allSubs, err := listAllSubTasksByTaskID(ctx, e.subTaskRepo, parentID)
	if err != nil {
		e.logger.Warn("file landed: list sub_tasks by parent_task_id failed",
			zap.String("parent_task_id", parentID.String()),
			zap.Error(err))
		return nil
	}
	var subTask *UpgradeSubTask
	for i := range allSubs {
		if allSubs[i].DeviceID == dev.ID {
			subTask = &allSubs[i].UpgradeSubTask
			break
		}
	}
	if subTask == nil {
		e.logger.Warn("file landed: no sub_task matches (parent_task_id, device_id)",
			zap.String("parent_task_id", parentID.String()),
			zap.String("device_id", dev.ID.String()),
			zap.String("device_sn", payload.DeviceSN),
			zap.Int("scanned", len(allSubs)))
		return nil
	}
	if subTask.Status != UpgradeUploading {
		e.logger.Info("file landed: sub_task not in uploading status, skip",
			zap.String("sub_task_id", subTask.ID.String()),
			zap.String("status", string(subTask.Status)))
		return nil
	}

	e.completeSubTask(ctx, subTask, payload.DeviceSN)
	e.logger.Info("log file landed, sub_task completed",
		zap.String("sub_task_id", subTask.ID.String()),
		zap.String("device_sn", payload.DeviceSN),
		zap.String("parent_task_id", parentID.String()))
	return nil
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

	// 判失败兼容 SOAP 1.1 fault（fault_code=0 但 fault_string 非空），详见 isCPEFault 注释。
	if isCPEFault(payload.FaultCode, payload.FaultStr) {
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

	// 判失败兼容 SOAP 1.1 fault（fault_code=0 但 fault_string 非空），详见 isCPEFault 注释。
	if isCPEFault(payload.FaultCode, payload.FaultStr) {
		e.failSubTask(ctx, subTask, fmt.Sprintf("Upload rejected by device. FaultCode: %d, FaultString: %s", payload.FaultCode, payload.FaultStr), FailureUploadFault)
		return nil
	}

	e.logger.Info("upload accepted, waiting for TransferComplete",
		zap.String("sub_task_id", subTask.ID.String()),
		zap.String("device_sn", payload.DeviceSN))
	return nil
}

// HandleRebootComplete handles device.inform.reboot_complete events.
// Used for rollback completion detection, UPS post-boot version verification,
// and legacy upgrade completion.
func (e *UpgradeExecutor) HandleRebootComplete(ctx context.Context, evt event.Event) error {
	var payload struct {
		DeviceSN string `json:"device_sn"`
		DeviceID struct {
			SerialNumber string `json:"serial_number"`
		} `json:"device_id"`
		ParameterList []tr069.ParameterValueStruct `json:"parameter_list"`
	}
	if err := evt.DecodePayload(&payload); err != nil {
		return nil
	}
	deviceSN := payload.DeviceSN
	if deviceSN == "" {
		deviceSN = payload.DeviceID.SerialNumber
	}
	if deviceSN == "" {
		return nil
	}

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

	// For 5G upgrade: skip — HandleUpgradeFinish handles the 102 event
	task, err := e.taskRepo.GetByID(ctx, subTask.TaskID)
	if err != nil {
		return nil
	}
	if task.TaskType == TaskTypeUpgrade && isUPSUpgradeDevice(dev) {
		return e.handleUPSUpgradeBoot(ctx, subTask, dev, payload.ParameterList)
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
		zap.String("device_sn", deviceSN))
	return nil
}

func (e *UpgradeExecutor) handleUPSUpgradeBoot(ctx context.Context, subTask *UpgradeSubTask, dev *model.Device, params []tr069.ParameterValueStruct) error {
	targetVersion := e.upsUpgradeTargetVersion(ctx, subTask)
	reportedVersion := firstNonBlank(
		extractTR069ParamExact(params, upsSoftwareVersionParamPath),
		extractTR069ParamSuffix(params, "DeviceInfo.SoftwareVersion"),
		dev.FirmwareVersion,
	)

	if targetVersion == "" {
		e.failSubTask(ctx, subTask, "UPS upgrade version verification failed: target version is empty.", FailureInternalError)
		return nil
	}
	if reportedVersion == "" {
		e.failSubTask(ctx, subTask, fmt.Sprintf("UPS upgrade version verification failed: no software version reported after reboot; target version: %s", targetVersion), FailureVersionMismatch)
		return nil
	}
	if !versionsEquivalent(reportedVersion, targetVersion) {
		e.failSubTask(ctx, subTask, fmt.Sprintf("UPS upgrade version verification failed: reported version %s does not match target version %s", reportedVersion, targetVersion), FailureVersionMismatch)
		return nil
	}

	e.completeSubTask(ctx, subTask, dev.SerialNumber)
	e.logger.Info("UPS reboot complete, software version verified",
		zap.String("sub_task_id", subTask.ID.String()),
		zap.String("device_sn", dev.SerialNumber),
		zap.String("software_version", reportedVersion))
	return nil
}

func (e *UpgradeExecutor) upsUpgradeTargetVersion(ctx context.Context, subTask *UpgradeSubTask) string {
	if subTask == nil {
		return ""
	}
	if v := strings.TrimSpace(subTask.DestVersion); v != "" {
		return v
	}
	if subTask.FirmwareID == nil || e.firmwareRepo == nil {
		return ""
	}
	fw, err := e.firmwareRepo.GetByID(ctx, *subTask.FirmwareID)
	if err != nil || fw == nil {
		return ""
	}
	return strings.TrimSpace(fw.Version)
}

func isUPSUpgradeDevice(dev *model.Device) bool {
	if dev == nil {
		return false
	}
	return strings.HasPrefix(strings.TrimSpace(dev.ProductClass), "UPS")
}

func extractTR069ParamExact(params []tr069.ParameterValueStruct, exactName string) string {
	for _, p := range params {
		if strings.TrimSpace(p.Name) == exactName {
			return strings.TrimSpace(p.Value)
		}
	}
	return ""
}

func extractTR069ParamSuffix(params []tr069.ParameterValueStruct, nameSuffix string) string {
	for _, p := range params {
		name := strings.TrimSpace(p.Name)
		if strings.HasSuffix(name, nameSuffix) {
			return strings.TrimSpace(p.Value)
		}
	}
	return ""
}

func versionsEquivalent(left, right string) bool {
	return strings.TrimSpace(left) == strings.TrimSpace(right)
}

func firstNonBlank(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
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
//
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

		go e.executeOne(context.Background(), subTask, fw, isKeepConfig, downloadFileType)
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

// shouldAbortDispatch 在派发 Download 之前做最后一道急停复查（#59 Problem 3）。返回 true
// 表示必须中止下发。三个判据，任一命中即中止：
//  1. ctx 已取消（任务被 Suspend/Terminate/canary 阈值暂停触发 cancelReg.cancel）。
//  2. sub_task 回库最新状态已是 Suspended 或 terminal（completed/failed/terminated）——
//     覆盖「SuspendUpgrade 只改 DB、cancel 漏掉了卡在锁/connReq 上的本 goroutine」的竞态。
//  3. parent task 已 Suspended/ended——整任务级急停，子任务行可能还没翻状态时也要拦住。
//
// 读 DB 失败时按「不中止」处理（fail-open）：宁可多发一次也不要因为一次瞬时读错就漏发，
// 与既有 acquireDeviceLock 出错也继续下发的保守取向一致；真正的权威终态仍由后续 reaper /
// 状态机兜底。读到的状态用 GetByID（routing repo 已按表分流）。
func (e *UpgradeExecutor) shouldAbortDispatch(ctx context.Context, subTask *UpgradeSubTask) bool {
	if ctx.Err() != nil {
		return true
	}
	current, err := e.subTaskRepo.GetByID(ctx, subTask.ID)
	if err != nil {
		e.logger.Warn("pre-dispatch sub-task re-check failed, proceeding",
			zap.String("sub_task_id", subTask.ID.String()), zap.Error(err))
		return false
	}
	if current.Status == UpgradeSuspended || IsUpgradeTerminal(current.Status) {
		return true
	}
	parent, err := e.taskRepo.GetByID(ctx, subTask.TaskID)
	if err != nil {
		e.logger.Warn("pre-dispatch parent task re-check failed, proceeding",
			zap.String("task_id", subTask.TaskID.String()), zap.Error(err))
		return false
	}
	return parent.Status == TaskSuspended || parent.Status == TaskEnded
}

func (e *UpgradeExecutor) acquireDeviceLock(ctx context.Context, deviceSN string, taskID uuid.UUID) (bool, error) {
	key := upgradeDeviceLockKey(deviceSN)
	ok, err := e.redis.SetNX(ctx, key, taskID.String(), time.Hour).Result()
	if err != nil {
		return false, fmt.Errorf("acquire device lock: %w", err)
	}
	return ok, nil
}

func (e *UpgradeExecutor) releaseDeviceLock(ctx context.Context, deviceSN string, subTaskID uuid.UUID) {
	if err := releaseOwnedDeviceLock(ctx, e.redis, deviceSN, subTaskID); err != nil {
		e.logger.Warn("release device lock",
			zap.String("sub_task_id", subTaskID.String()),
			zap.String("device_sn", deviceSN),
			zap.Error(err))
	}
}

// verifyFirmware 在 Download 下发前对固件做完整性 / 签名校验（issue #8）。
// 通过 → 返回 nil（顺带记 pass / legacy_md5 指标 + 降级 warn）；失败 → 标 sub_task
// 失败 + 记 fail 指标 + error 日志并返回非 nil，调用方据此中止下发，绝不静默放行。
func (e *UpgradeExecutor) verifyFirmware(subTask *UpgradeSubTask, fw *FirmwareVersion, deviceSN string) error {
	v := e.verifier
	if v == nil {
		// 防御：构造期已默认 HashOnlyVerifier，理论上不为 nil。兜底避免校验被绕过。
		v = NewHashOnlyVerifier()
	}
	if err := v.Verify(fw); err != nil {
		code := VerificationFailureCode(err)
		e.firmwareMetrics.RecordFail(code)
		fwID := ""
		fwVersion := ""
		if fw != nil {
			fwID = fw.ID.String()
			fwVersion = fw.Version
		}
		e.logger.Error("firmware verification failed, aborting download dispatch",
			zap.String("sub_task_id", subTask.ID.String()),
			zap.String("device_sn", deviceSN),
			zap.String("firmware_id", fwID),
			zap.String("firmware_version", fwVersion),
			zap.String("failure_code", string(code)),
			zap.Error(err))
		e.failSubTask(context.Background(), subTask,
			fmt.Sprintf("Upgrade can not be started, firmware verification failed: %v", err), code)
		return err
	}

	legacy := IsLegacyMD5Only(fw)
	e.firmwareMetrics.RecordPass(legacy)
	if legacy {
		// 仅 MD5 的存量固件：放行但提示运营尽快补 SHA-256 / 重新上传。
		fwID := ""
		if fw != nil {
			fwID = fw.ID.String()
		}
		e.logger.Warn("firmware passed verification on legacy MD5 fallback (no sha256); consider re-uploading to compute sha256",
			zap.String("sub_task_id", subTask.ID.String()),
			zap.String("device_sn", deviceSN),
			zap.String("firmware_id", fwID))
	}
	return nil
}

func (e *UpgradeExecutor) failSubTask(ctx context.Context, subTask *UpgradeSubTask, reason string, code FailureCode) {
	e.failSubTaskWithLockRelease(ctx, subTask, reason, code, true)
}

func (e *UpgradeExecutor) failSubTaskWithLockRelease(ctx context.Context, subTask *UpgradeSubTask, reason string, code FailureCode, releaseLock bool) {
	if err := e.subTaskRepo.UpdateStatusWithCode(ctx, subTask.ID, UpgradeFailed, reason, code); err != nil {
		e.logger.Error("fail sub-task", zap.String("sub_task_id", subTask.ID.String()), zap.Error(err))
	}
	if releaseLock && subTask.DeviceSN != "" {
		e.releaseDeviceLock(ctx, subTask.DeviceSN, subTask.ID)
	}
	if err := e.taskRepo.IncrementCounts(ctx, subTask.TaskID, 0, 1); err != nil {
		e.logger.Error("increment fail count", zap.Error(err))
	}
	finalizeTask(ctx, e.taskRepo, e.logger, subTask.TaskID)
	e.publishUpgradeResult(ctx, event.SubjectUpgradeFailed, subTask, subTask.DeviceSN, reason)
}

// failLockedSubTask handles the DEVICE_LOCKED case by looking up the blocking task name.
func (e *UpgradeExecutor) failLockedSubTask(ctx context.Context, subTask *UpgradeSubTask, deviceSN string) {
	reason := fmt.Sprintf("Upgrade can not be started, device %s is busy in another task%s",
		deviceSN, e.describeDeviceLockHolder(ctx, deviceSN))

	// This contender never acquired the lock, so it must not release it.
	e.failSubTaskWithLockRelease(ctx, subTask, reason, FailureDeviceLocked, false)
}

// describeDeviceLockHolder 读 Redis 设备锁 value（占用者 subTaskID），反查其主任务名，
// 生成 " (held by task \"xxx\")" 供排查；查不到返回空串（错误信息仍可辨识）。
func (e *UpgradeExecutor) describeDeviceLockHolder(ctx context.Context, deviceSN string) string {
	val, err := e.redis.Get(ctx, upgradeDeviceLockKey(deviceSN)).Result()
	if err != nil || strings.TrimSpace(val) == "" {
		return ""
	}
	holderID, parseErr := uuid.Parse(strings.TrimSpace(val))
	if parseErr != nil {
		return ""
	}
	holder, lookupErr := e.subTaskRepo.GetByID(ctx, holderID)
	if lookupErr != nil || holder == nil {
		return ""
	}
	parent, parentErr := e.taskRepo.GetByID(ctx, holder.TaskID)
	if parentErr != nil || parent == nil || parent.TaskName == "" {
		return ""
	}
	return fmt.Sprintf(" (held by task %q, wait for it to finish or terminate it first)", parent.TaskName)
}

func (e *UpgradeExecutor) completeSubTask(ctx context.Context, subTask *UpgradeSubTask, deviceSN string) {
	if err := e.subTaskRepo.UpdateStatus(ctx, subTask.ID, UpgradeCompleted, ""); err != nil {
		e.logger.Error("complete sub-task", zap.String("sub_task_id", subTask.ID.String()), zap.Error(err))
	}

	// qa-614 #371: 完成后回填"目标版本"(dest_version)。手动回退/升级时 OMC 事先不知道
	// 回退后的具体版本号（走设备上一个 bank），创建子任务时 DestVersion 为空，前端"目标版本"
	// 列恒显示"-"。完成时设备已重启并上报新 FirmwareVersion，按设备 SN 取最新值回填。
	// 仅当 DestVersion 仍为空时才回填，避免覆盖创建时已显式指定的目标固件版本。
	e.backfillDestVersion(ctx, subTask, deviceSN)

	e.releaseDeviceLock(ctx, deviceSN, subTask.ID)

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
	e.publishUpgradeResult(ctx, event.SubjectUpgradeCompleted, subTask, deviceSN, "")
}

func (e *UpgradeExecutor) publishUpgradeResult(
	ctx context.Context,
	subject string,
	subTask *UpgradeSubTask,
	deviceSN string,
	reason string,
) {
	if e.eventBus == nil || e.deviceRepo == nil || subTask == nil || strings.TrimSpace(deviceSN) == "" {
		return
	}
	dev, err := e.deviceRepo.GetBySerialNumber(ctx, deviceSN)
	if err != nil || dev == nil {
		e.logger.Warn("lookup device before publishing upgrade result",
			zap.String("subject", subject),
			zap.String("device_sn", deviceSN),
			zap.Error(err))
		return
	}
	evt, err := event.NewEvent(subject, map[string]interface{}{
		"sub_task_id": subTask.ID.String(),
		"task_id":     subTask.TaskID.String(),
		"device_id":   dev.ID.String(),
		"reason":      reason,
	})
	if err != nil {
		e.logger.Warn("build upgrade result event", zap.String("subject", subject), zap.Error(err))
		return
	}
	if err := e.eventBus.Publish(ctx, subject, evt); err != nil {
		e.logger.Warn("publish upgrade result event", zap.String("subject", subject), zap.Error(err))
	}
}

// backfillDestVersion 在子任务完成后，用设备最新上报的固件版本回填 dest_version（qa-614 #371）。
// 仅当 subTask.DestVersion 为空时才回填——创建时已显式指定目标固件版本的不覆盖。
// 任何失败都仅记日志、不影响完成流程（dest_version 仅用于展示）。
func (e *UpgradeExecutor) backfillDestVersion(ctx context.Context, subTask *UpgradeSubTask, deviceSN string) {
	if subTask.DestVersion != "" || deviceSN == "" {
		return
	}
	dev, err := e.deviceRepo.GetBySerialNumber(ctx, deviceSN)
	if err != nil || dev == nil || dev.FirmwareVersion == "" {
		return
	}
	if err := e.subTaskRepo.UpdateDestVersionByID(ctx, subTask.ID, dev.FirmwareVersion); err != nil {
		e.logger.Warn("backfill dest_version after completion",
			zap.String("sub_task_id", subTask.ID.String()),
			zap.String("device_sn", deviceSN),
			zap.Error(err))
		return
	}
	subTask.DestVersion = dev.FirmwareVersion
}

package backup

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/acs/connreq"
	"github.com/omcgo/omcgo/internal/acs/transfercfg"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/device"
	devtask "github.com/omcgo/omcgo/internal/task"
	"github.com/omcgo/omcgo/internal/ufte"
)

// BackupTypeSpec 是配置备份类型的路由表条目。
// 仅保存"选哪个 UFTE 模板"所需的路由信息（TypeCode + ProductClass 前缀）和
// URL/文件命名辅助参数。
// TR-069 FileType 字符串来自 ufte.BuiltInTaskType(TypeCode).FileType，
// 在执行时用设备 OUI 替换占位符 {OUI}，不在此处硬编码。
type BackupTypeSpec struct {
	TypeCode             string   // 对应 UFTE TypeCode，如 "CONFIG_BACKUP_XML"
	URLFileTypeParam     string   // 上传 URL fileType= 参数值，供 upload handler 路由到正确 bucket
	FileExtension        string   // 目标文件扩展名，如 ".xml" 或 ".nv"
	ProductClassPrefixes []string // 匹配设备 ProductClass 的前缀列表；nil 表示兜底默认类型
}

// backupTypeSpecs 是配置备份类型路由表，按匹配优先级排列。
// 仅负责 ProductClass → UFTE TypeCode 的映射；
// TR-069 FileType 值从 ufte.BuiltInTaskType(TypeCode).FileType 读取，
// 以 ufte/model.go 内置模板定义为唯一数据源，此处不重复维护。
var backupTypeSpecs = []BackupTypeSpec{
	{
		TypeCode:         "CONFIG_BACKUP_NV",
		URLFileTypeParam: "CONFIGBACKUP_NV",
		FileExtension:    ".nv",
		// 前缀须与 products.xml / param-model-routing.xml 的真实 ProductClass 对齐：
		// MLQ 全系（FAP/MLQ/SC）、MLN 全系（FAP/MLN/SC|CA|DC）、BM（FAP/BU1810）均导出 NV 配置。
		// 历史 bug：曾写成 "FAP/MLN_SC"（下划线）导致 MLN 全系不命中而误落 XML；
		// BM(FAP/BU1810) 此前完全未登记 → 永远兜底 XML，NV 路径不可达（qa-614 #376）。
		ProductClassPrefixes: []string{"FAP/MLQ/", "FAP/MLN/", "FAP/BU1810"},
	},
	{
		TypeCode:         "CONFIG_BACKUP_XML",
		URLFileTypeParam: "CONFIGBACKUP_XML",
		FileExtension:    ".xml",
		// nil ProductClassPrefixes = 兜底，覆盖所有其他平台
	},
}

// selectBackupType 根据设备 ProductClass 选择对应的备份类型规格。
// 按注册顺序匹配前缀；始终返回非 nil（兜底为 CONFIG_BACKUP_XML）。
func selectBackupType(productClass string) *BackupTypeSpec {
	for i := range backupTypeSpecs {
		spec := &backupTypeSpecs[i]
		if len(spec.ProductClassPrefixes) == 0 {
			return spec // 兜底类型
		}
		for _, prefix := range spec.ProductClassPrefixes {
			if strings.HasPrefix(productClass, prefix) {
				return spec
			}
		}
	}
	return &backupTypeSpecs[len(backupTypeSpecs)-1]
}

// buildBackupUploadURL 根据上传配置（来自 transfercfg.Policy / sys_configs）、备份类型规格、
// 设备 SN、任务 ID 和文件名构造 ACS 上传 URL。
// URL 中的 fileType 参数来自 spec.URLFileTypeParam（如 CONFIGBACKUP_XML / CONFIGBACKUP_NV），
// upload handler 通过该参数路由到 config_backup bucket。
func buildBackupUploadURL(upload transfercfg.UploadSettings, spec *BackupTypeSpec, sn, taskID, filename string) (string, error) {
	servicePath := upload.Path
	if servicePath == "" {
		servicePath = "/smallcell/FileUploadService"
	}
	if err := transfercfg.ValidateServicePath(servicePath); err != nil {
		return "", fmt.Errorf("validate backup upload service path: %w", err)
	}
	query := "fileType=" + url.QueryEscape(spec.URLFileTypeParam) +
		"&sn=" + url.QueryEscape(sn) +
		"&taskId=" + url.QueryEscape(taskID) +
		"&filename=" + url.QueryEscape(filename)
	return transfercfg.BuildTemplateURL(upload.BaseURL, servicePath+"?"+query)
}

// BackupExecutor subscribes to backup.task.created events and executes
// backup tasks by pushing Upload commands to the unified device task queue.
//
// T-0073 Phase 1 additions: when configured with a PolicyService and metrics,
// the executor publishes alarm.raised on failure paths according to the
// active BackupPolicy.AlertOnFailure flag. Both fields are optional (nil-safe)
// to keep the constructor signature backward-compatible.
type BackupExecutor struct {
	taskRepo         TaskRepository
	deviceRepo       device.DeviceRepository
	taskSvc          devtask.Enqueuer
	connReq          *connreq.Client
	eventBus         event.EventBus
	transferProvider transfercfg.Provider // ACS 上传配置（来自 sys_configs acs_transfer，UI 可配置）
	uploadResolver   transferAddressResolver
	policyService    PolicyGetter        // optional (T-0073)
	ftpConfigRepo    FTPConfigRepository // optional; used when remote storage backend is selected
	metrics          *PolicyMetrics      // optional (T-0073)
	logger           *zap.Logger
}

type transferAddressResolver interface {
	Resolve(context.Context, uuid.UUID, transfercfg.TransferDirection) (transfercfg.AddressDecision, error)
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

// SetTransferProvider 注入 transfercfg.Provider，用于运行时读取 ACS 上传配置。
// 配置来源为 sys_configs.category='acs_transfer'（界面「系统管理 → ACS 传输」），
// 支持运行时修改无需重启。须在 Subscribe 前调用；未注入时 URL 为空。
func (e *BackupExecutor) SetTransferProvider(p transfercfg.Provider) {
	e.transferProvider = p
}

func (e *BackupExecutor) SetUploadAddressResolver(r transferAddressResolver) {
	e.uploadResolver = r
}

// SetFTPConfigRepository injects the remote FTP/SFTP config source used when
// backup policy selects a non-local storage backend.
func (e *BackupExecutor) SetFTPConfigRepository(repo FTPConfigRepository) {
	e.ftpConfigRepo = repo
}

// SetPolicyEnforcement wires PolicyService + metrics post-construction so the
// failure path can publish alarm.raised when AlertOnFailure=true. Either
// argument may be nil to disable that side-effect (default is disabled).
func (e *BackupExecutor) SetPolicyEnforcement(policyService PolicyGetter, metrics *PolicyMetrics) {
	e.policyService = policyService
	e.metrics = metrics
}

func normalizeRemoteProtocol(protocol string, storageBackend string) string {
	if protocol != "" {
		return strings.ToLower(protocol)
	}
	return strings.ToLower(storageBackend)
}

func buildRemoteUploadURL(cfg *FTPConfig, filename string, storageBackend string) string {
	protocol := normalizeRemoteProtocol(cfg.Protocol, storageBackend)
	host := cfg.Host
	if cfg.Port > 0 {
		host = fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	}
	remotePath := strings.TrimSpace(cfg.RemotePath)
	if remotePath == "" {
		remotePath = "/"
	}
	remotePath = "/" + strings.Trim(remotePath, "/")
	if remotePath == "/" {
		remotePath = ""
	}
	return fmt.Sprintf("%s://%s%s/%s", protocol, host, remotePath, url.PathEscape(filename))
}

type transferDecisionLogFields struct {
	Protocol   string
	Reason     string
	Capability string
}

func (e *BackupExecutor) resolveUploadTarget(ctx context.Context, spec *BackupTypeSpec, dev *model.Device, task *BackupTask, targetFilename string) (string, string, string, transferDecisionLogFields, error) {
	if e.policyService != nil {
		policy, err := e.policyService.Get(ctx)
		if err != nil {
			return "", "", "", transferDecisionLogFields{}, fmt.Errorf("get backup policy: %w", err)
		}
		if policy != nil {
			switch policy.StorageBackend {
			case "ftp", "sftp":
				if policy.FTPConfigID == nil {
					return "", "", "", transferDecisionLogFields{}, fmt.Errorf("backup policy storage_backend=%s missing ftp_config_id", policy.StorageBackend)
				}
				if e.ftpConfigRepo == nil {
					return "", "", "", transferDecisionLogFields{}, fmt.Errorf("backup policy storage_backend=%s but ftp config repo is not wired", policy.StorageBackend)
				}
				cfg, err := e.ftpConfigRepo.GetByID(ctx, *policy.FTPConfigID)
				if err != nil {
					return "", "", "", transferDecisionLogFields{}, fmt.Errorf("load ftp config %s: %w", policy.FTPConfigID.String(), err)
				}
				if cfg == nil {
					return "", "", "", transferDecisionLogFields{}, fmt.Errorf("ftp config %s not found", policy.FTPConfigID.String())
				}
				if !cfg.Enabled {
					return "", "", "", transferDecisionLogFields{}, fmt.Errorf("ftp config %s is disabled", cfg.ID.String())
				}
				password := ""
				if cfg.PasswordEncrypted != nil {
					password = *cfg.PasswordEncrypted
				}
				fields := transferDecisionLogFields{
					Protocol: normalizeRemoteProtocol(cfg.Protocol, policy.StorageBackend),
					Reason:   "backup_policy_remote_storage",
				}
				return buildRemoteUploadURL(cfg, targetFilename, policy.StorageBackend), cfg.Username, password, fields, nil
			}
		}
	}

	var uploadSettings transfercfg.UploadSettings
	decisionFields := transferDecisionLogFields{
		Protocol:   string(transfercfg.TransferProtocolHTTP),
		Reason:     "legacy_upload_config",
		Capability: string(transfercfg.HTTPSCapabilityNotRead),
	}
	baseURL := ""
	if e.transferProvider != nil {
		uploadSettings = e.transferProvider.Snapshot(ctx).Upload
	}
	if e.uploadResolver != nil {
		decision, err := e.uploadResolver.Resolve(ctx, dev.ID, transfercfg.TransferDirectionUpload)
		if err != nil {
			return "", "", "", transferDecisionLogFields{}, fmt.Errorf("resolve ACS backup upload address: %w", err)
		}
		baseURL = decision.BaseURL
		decisionFields = transferDecisionLogFields{
			Protocol:   string(decision.Protocol),
			Reason:     string(decision.Reason),
			Capability: string(decision.Capability),
		}
	} else {
		baseURL = uploadSettings.BaseURL
	}
	uploadSettings.BaseURL = baseURL
	uploadURL, err := buildBackupUploadURL(uploadSettings, spec, dev.SerialNumber, task.ID.String(), targetFilename)
	if err != nil {
		return "", "", "", transferDecisionLogFields{}, fmt.Errorf("build ACS backup upload URL: %w", err)
	}
	return uploadURL, uploadSettings.Username, uploadSettings.Password, decisionFields, nil
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

		// Build Upload command. TR-069 Upload RPC 需要：
		// - FileType：厂商扩展格式，XML 平台 "10 {OUI} Configuration File"，
		//             NV 平台（MLQ/MLN/BM 等）"12 {OUI} Configuration File"。
		// - URL：CPE 上传目标，格式
		//   {baseURL}{path}?fileType=CONFIGBACKUP_XML|CONFIGBACKUP_NV&sn={sn}&taskId={taskID}&filename={file}
		//   upload handler 通过 fileType query 参数路由到 config_backup bucket。
		//
		// issue #585: 真实落盘文件名由 ACS upload handler 在收 POST 时
		// 强制覆写为 {sn}_CFG.{xml|nv}（TR-069 Upload RPC 无 TargetFileName
		// 字段，设备端无法控制）。这里仅作为 URL query 中的 filename=
		// 提示传给设备/handler，handler 完全忽略并按 sn+fileType 重命名。
		// task_id 仍由 URL query taskId= 透传，event 兜底从 query 抽 prefix。
		spec := selectBackupType(dev.ProductClass)
		targetFilename := fmt.Sprintf("%s_CFG%s", dev.SerialNumber, spec.FileExtension)

		uploadURL, uploadUsername, uploadPassword, transferFields, uploadErr := e.resolveUploadTarget(ctx, spec, dev, task, targetFilename)
		if uploadErr != nil {
			e.logger.Warn("resolve backup upload target",
				zap.String("device_sn", dev.SerialNumber),
				zap.String("task_id", task.ID.String()),
				zap.Error(uploadErr))
			continue
		}

		// TR-069 Upload FileType 来自 UFTE 内置模板（ufte/model.go 为唯一数据源），
		// 运行时将模板中的 {OUI} 占位符替换为设备实际 OUI。
		var fileType string
		if tt, ok := ufte.BuiltInTaskType(spec.TypeCode); ok {
			fileType = strings.ReplaceAll(tt.FileType, "{OUI}", dev.OUI)
		} else {
			// 不应命中：内置模板缺失时降级，避免下发空 FileType
			e.logger.Warn("ufte built-in type not found, using fallback file_type",
				zap.String("type_code", spec.TypeCode))
			fileType = fmt.Sprintf("10 %s Configuration File", dev.OUI)
		}

		params := map[string]interface{}{
			"file_type": fileType,
			"url":       uploadURL,
		}
		if uploadUsername != "" {
			params["username"] = uploadUsername
			params["password"] = uploadPassword
		}
		paramsJSON, marshalErr := json.Marshal(params)
		if marshalErr != nil {
			e.logger.Warn("marshal upload params", zap.String("device_sn", targetSN), zap.Error(marshalErr))
			continue
		}
		if _, err := e.taskSvc.CreateTask(ctx, &devtask.CreateTaskRequest{
			DeviceSN: dev.SerialNumber,
			Method:   "Upload",
			Params:   paramsJSON,
			Source:   devtask.TaskSourceSystem,
			SourceID: task.ID.String(),
			// M2 of backup-restore-alignment-plan: 规范要求 CommandKey 格式
			// 为 `{cellCode}_BACKUP`；为支持 TransferComplete 回写定位任务行，
			// 实际写为 `{cellCode}_BACKUP_{taskID8}`。device.SiteID 作为 cellCode
			// 兜底（无 site 时使用 SN）。
			CommandKey: BuildBackupCommandKey(dev.SiteID, dev.SerialNumber, task.ID.String()),
		}); err != nil {
			e.logger.Warn("push upload command",
				zap.String("device_sn", dev.SerialNumber),
				zap.Error(err))
			continue
		}
		e.logger.Info("backup upload command queued",
			zap.String("device_sn", dev.SerialNumber),
			zap.String("file_type", fileType),
			zap.String("backup_type", spec.TypeCode),
			zap.String("transfer_protocol", transferFields.Protocol),
			zap.String("transfer_reason", transferFields.Reason),
			zap.String("https_capability", transferFields.Capability))

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

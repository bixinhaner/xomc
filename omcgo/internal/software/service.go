package software

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	stderrors "errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/minio/minio-go/v7"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/acs/connreq"
	"github.com/omcgo/omcgo/internal/acs/transfercfg"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/storage"
	"github.com/omcgo/omcgo/internal/device"
	devtask "github.com/omcgo/omcgo/internal/task"
)

// finalizeSubTaskFailure 是 BatchCollect / BatchUpgrade / RollbackDevices 共用的"sub_task
// 创建失败后的兜底"。这里只清理已经 commit 的 mainTask，并把 PG unique violation
// 翻译成业务层 ErrAlreadyExists 让 handler 自动映射到 HTTP 409。
//
// 为什么有这个函数：repository 没有暴露 tx 接口，无法把 mainTask + subTask 两次写库
// 真正原子化。次优解是失败时手动回滚 mainTask，避免 UI 上出现"主任务存在但子任务缺失"
// 的孤儿任务——这种孤儿任务列表能看到、start 会因为没 sub_task 直接 400，用户无解。
//
// 触发场景：同一设备已经有 active sub_task 占着 idx_upgrade_sub_tasks_device_active_uniq
// 索引位（status NOT IN completed/failed/terminated）。前端重复点击或并发请求时会反复命中。
func (s *SoftwareService) finalizeSubTaskFailure(ctx context.Context, mainID uuid.UUID, cause error) error {
	if delErr := s.taskRepo.Delete(ctx, mainID); delErr != nil {
		s.logger.Error("rollback main task after sub-task create failure",
			zap.String("task_id", mainID.String()),
			zap.NamedError("cause", cause),
			zap.Error(delErr))
	}
	var pgErr *pgconn.PgError
	if stderrors.As(cause, &pgErr) && pgErr.Code == "23505" {
		return fmt.Errorf("%w: 该设备已有进行中的同类任务，请等待完成或先终止后再试", commonerrors.ErrAlreadyExists)
	}
	return cause
}

// SoftwareService provides firmware upload and device upgrade functionality.
type SoftwareService struct {
	firmwareRepo    FirmwareRepository
	taskRepo        TaskRepository
	subTaskRepo     SubTaskRepository
	deviceRepo      device.DeviceRepository
	taskSvc         devtask.Enqueuer
	connReq         *connreq.Client
	minioClient     *minio.Client
	firmwareBkt     string
	eventBus        event.EventBus
	redis           redis.UniversalClient
	executor        *UpgradeExecutor
	rollbackExec    *RollbackExecutor
	adapter         UpgradeAdapter
	canaryMetrics   *CanaryMetrics   // optional; nil-safe via metrics methods
	rollbackMetrics *RollbackMetrics // optional; nil-safe via metrics methods (T-0021)
	logger          *zap.Logger
	// taskFileCleaner 可选 hook：DeleteUpgrade 时回收任务关联的 MinIO 对象 +
	// backup_restore_file 元数据。由 cmd/app/provider/modules.go 用 backup.FileRepository
	// + MinIO 客户端装配。未注入时仅删 upgrade_tasks/upgrade_sub_tasks（向后兼容）。
	taskFileCleaner func(ctx context.Context, taskID uuid.UUID) error
}

// SetTaskFileCleaner 注入"任务删除时清理 MinIO 对象 + backup_restore_file 元数据"的回调。
// 由 wiring 层（cmd/app/provider/modules.go）装配；不调用则 DeleteUpgrade 仅删 DB 任务行，
// MinIO 上的备份/日志文件会成为孤儿。Best-effort 语义：清理失败仅 warn，不阻断主删除。
func (s *SoftwareService) SetTaskFileCleaner(fn func(ctx context.Context, taskID uuid.UUID) error) {
	s.taskFileCleaner = fn
}

// SetCanaryMetrics wires Prometheus metrics for canary stage transitions.
// Safe to call after construction (DI container picks one MetricsReg).
func (s *SoftwareService) SetCanaryMetrics(m *CanaryMetrics) {
	s.canaryMetrics = m
}

// SetRollbackMetrics wires Prometheus metrics for rollback audit (T-0021).
// Safe to call after construction so the DI container drives registration.
func (s *SoftwareService) SetRollbackMetrics(m *RollbackMetrics) {
	s.rollbackMetrics = m
}

// NewSoftwareService creates a new SoftwareService.
func NewSoftwareService(
	firmwareRepo FirmwareRepository,
	taskRepo TaskRepository,
	subTaskRepo SubTaskRepository,
	deviceRepo device.DeviceRepository,
	taskSvc devtask.Enqueuer,
	connReq *connreq.Client,
	minioClient *minio.Client,
	firmwareBucket string,
	eventBus event.EventBus,
	redisClient redis.UniversalClient,
	logger *zap.Logger,
) *SoftwareService {
	s := &SoftwareService{
		firmwareRepo: firmwareRepo,
		taskRepo:     taskRepo,
		subTaskRepo:  subTaskRepo,
		deviceRepo:   deviceRepo,
		taskSvc:      taskSvc,
		connReq:      connReq,
		minioClient:  minioClient,
		firmwareBkt:  firmwareBucket,
		eventBus:     eventBus,
		redis:        redisClient,
		adapter:      NewDefaultUpgradeAdapter(),
		logger:       logger.Named("software"),
	}

	s.executor = NewUpgradeExecutor(
		taskRepo, subTaskRepo, deviceRepo, firmwareRepo,
		taskSvc, connReq, redisClient, eventBus, logger,
	)
	s.rollbackExec = NewRollbackExecutor(
		taskRepo, subTaskRepo, deviceRepo,
		taskSvc, connReq, redisClient, eventBus, logger,
	)

	return s
}

// UploadFirmware stores a firmware file to MinIO and creates a firmware version record.
func (s *SoftwareService) UploadFirmware(ctx context.Context, fw *FirmwareVersion, file io.Reader, fileSize int64) error {
	category := "img"
	switch fw.FileType {
	case FileTypePATCH:
		category = "patch"
	case FileTypeAP:
		category = "ap"
	case FileTypeFPGA:
		category = "fpga"
	}
	objectPath := storage.FirmwarePath(category, fw.ProductClass, fw.Version, fw.FileName)

	hash := md5.New()
	teeReader := io.TeeReader(file, hash)

	_, err := s.minioClient.PutObject(ctx, s.firmwareBkt, objectPath, teeReader, fileSize, minio.PutObjectOptions{
		ContentType: "application/octet-stream",
	})
	if err != nil {
		return fmt.Errorf("upload firmware to MinIO: %w", err)
	}

	fw.MinIOPath = objectPath
	fw.FileSize = fileSize
	fw.MD5Val = hex.EncodeToString(hash.Sum(nil))
	if fw.Status == "" {
		fw.Status = "active"
	}

	if err := s.firmwareRepo.Create(ctx, fw); err != nil {
		if delErr := s.minioClient.RemoveObject(ctx, s.firmwareBkt, objectPath, minio.RemoveObjectOptions{}); delErr != nil {
			s.logger.Error("cleanup orphaned firmware file", zap.String("path", objectPath), zap.Error(delErr))
		}
		return fmt.Errorf("create firmware record: %w", err)
	}

	if evt, err := event.NewEvent(event.SubjectFirmwareUploaded, map[string]interface{}{
		"firmware_id": fw.ID.String(),
		"version":     fw.Version,
	}); err == nil {
		if pubErr := s.eventBus.Publish(ctx, event.SubjectFirmwareUploaded, evt); pubErr != nil {
			s.logger.Warn("publish firmware.uploaded event", zap.Error(pubErr))
		}
	}

	return nil
}

// SetUploadConfig 配置日志采集 Upload RPC 的 ACS 上传基础 URL（CPE 可达地址）。
// 应在 NewSoftwareService 之后、第一次 BatchCollect 之前调用（通常由 DI 容器注入）。
// SetLogCollectResumer 装配 LogCollect 类子任务"设备上线即重试"回调（透传给 executor）。
// ufte.Service 在 cmd/app/provider/modules.go initUFTEModule 中注入；详见
// docs/project/backup-display-fix-20260520.md F7。
func (s *SoftwareService) SetLogCollectResumer(r LogCollectResumer) {
	s.executor.SetLogCollectResumer(r)
}

// SetParamPathTranslator 装配 standardPath → privatePath 翻译器（透传给 executor
// 和 rollback executor），供 SPV / GPV 下发前翻译参数名。
//
// 业务场景：
//   - FAULT_LOG_COLLECT（executor）：SPV 写 FaultLogURL 时翻译
//   - VERSION_ROLLBACK（rollbackExec）：4G 阶段一 GPV ROLLBACK_ENABLE +
//     阶段二 SPV ROLLBACK_CONTROL 都翻译；5G SPV ActivateEnable 翻译
func (s *SoftwareService) SetParamPathTranslator(t ParamPathTranslator) {
	s.executor.SetParamPathTranslator(t)
	if s.rollbackExec != nil {
		s.rollbackExec.SetParamPathTranslator(t)
	}
}

// OnLogFileLanded 实现 backup.FileLandedNotifier 接口。
// 由 backup.FilePathRecorder 在 backup_restore_file 元数据 upsert 完成后同进程
// 直接回调，按 (deviceSN, taskID) 推进 LogCollect 类（FAULT_LOG_COLLECT /
// RUNTIME_LOG_COLLECT / CONFIG_BACKUP_*）sub_task 到 Completed。
//
// SPV 触发的 FAULT_LOG_COLLECT 链路不走 TR-069 TransferComplete，"文件落地即成功"
// 是业务定义的结束条件——见 ufte/model.go FAULT_LOG_COLLECT 的 StepChain
// WAIT_FILE_UPLOAD（区别于 RUNTIME_LOG_COLLECT 的 WAIT_TRANSFER_COMPLETE）。
//
// 复用 executor.HandleFileLandedForCollect：同款 (parent_task_id + device_sn)
// 反查逻辑 + parent.TaskType==LogCollect 过滤 + status==Uploading 防误推。
func (s *SoftwareService) OnLogFileLanded(ctx context.Context, deviceSN, taskID string) {
	// 把 hook 参数转回 event payload 形式，复用既有 handler，避免逻辑双份维护。
	evt, err := event.NewEvent(event.SubjectBackupFileReceived, map[string]interface{}{
		"device_sn": deviceSN,
		"task_id":   taskID,
	})
	if err != nil {
		s.logger.Warn("OnLogFileLanded: build event", zap.Error(err))
		return
	}
	if err := s.executor.HandleFileLandedForCollect(ctx, evt); err != nil {
		s.logger.Warn("OnLogFileLanded: handle file landed",
			zap.String("device_sn", deviceSN),
			zap.String("task_id", taskID),
			zap.Error(err))
	}
}

// ExecuteOneUploadDirect 是 executor.ExecuteOneUpload 的对外门面，供 ufte 包在
// LogCollectResumer 回调中重启子任务的 Upload RPC 使用。与 startCollectExecution
// 走的是同一份执行路径，区别仅在于一次只跑一个子任务、不限并发。
func (s *SoftwareService) ExecuteOneUploadDirect(ctx context.Context, subTask *UpgradeSubTask, fileType, targetFileNameTemplate, transportPath string) {
	s.executor.ExecuteOneUpload(ctx, subTask, fileType, targetFileNameTemplate, transportPath)
}

// ExecuteOneSetParamCollectDirect 是 executor.ExecuteOneSetParamCollect 的对外门面，
// 供 ufte 包在 LogCollectResumer 回调中重启子任务的 SPV 触发上传使用。
func (s *SoftwareService) ExecuteOneSetParamCollectDirect(ctx context.Context, subTask *UpgradeSubTask, paramPath, transportPath string) {
	s.executor.ExecuteOneSetParamCollect(ctx, subTask, paramPath, transportPath)
}

func (s *SoftwareService) SetUploadConfig(acsUploadBaseURL string) {
	s.executor.SetUploadConfig(acsUploadBaseURL)
}

// SetTransferProvider 把"系统管理 → ACS 传输"系统配置接进来，Upload RPC 下发时
// 用它的 BaseURL / Username / Password。改配置 30 秒内自动生效，无需重启。
func (s *SoftwareService) SetTransferProvider(p transfercfg.Provider) {
	s.executor.SetTransferProvider(p)
}

// BatchCollectRequest 日志采集任务创建请求。
type BatchCollectRequest struct {
	DeviceIDs              []uuid.UUID
	TaskName               string
	FileType               string // Upload RPC FileType，如 "6"（运行日志）、"8"（故障日志）
	TargetFileNameTemplate string // 目标文件名模板，如 "runtime-{task_id8}-{sn}.tar.gz"
	TransportPath          string // 上传路径模板（Upload 或 SPV 模式各自的 URL 模板，渲染规则不同）
	// RPCType 决定使用 TR-069 Upload RPC（"UPLOAD"，默认）还是 SetParameterValues
	// 直接给设备私有参数写 URL（"SET_PARAM_VALUES"，如 FAULT_LOG_COLLECT 用 FaultLogURL）。
	// 留空 → "UPLOAD"，兼容现有 RUNTIME_LOG_COLLECT / CONFIG_BACKUP 链路。
	RPCType string
	// ParamPath 仅在 RPCType="SET_PARAM_VALUES" 时使用，是 SPV 下发的参数名，例如
	// "Device.DeviceInfo.X_COM_Log.FaultLogURL"。其值由 executor 用 transport_path
	// 渲染后写入。
	ParamPath       string
	CreateUser      string
	CreateSuspended bool
}

// BatchCollect 创建日志采集主任务及各设备子任务，然后启动执行。
// 与 BatchUpgrade 类似，但不需要固件，执行时发送 Upload RPC。
//
// 多表分发（按业务拆表 — docs/design/task-tables-split-by-business-20260521.md）：
// 通过 WithRouteHint 把 req.FileType 注入 ctx，下面所有 s.taskRepo / s.subTaskRepo
// 调用都会被 RoutingTaskRepository / RoutingSubTaskRepository wrapper 据此路由到
// 对应业务的物理表（config_backup_* / runtime_log_collect_* / fault_log_collect_*）。
// 旧表 upgrade_tasks / upgrade_sub_tasks 仅保留给升级 / 回退（走 BatchUpgrade / RollbackDevices，
// 那两个方法不注入 hint → wrapper fallback 到旧表）。
func (s *SoftwareService) BatchCollect(ctx context.Context, req BatchCollectRequest) (*UpgradeTask, error) {
	ctx = WithRouteHint(ctx, req.FileType)
	concurrency := 5

	mainTask := &UpgradeTask{
		TaskName:         req.TaskName,
		TaskType:         TaskTypeLogCollect,
		DownloadFileType: req.FileType,               // 复用字段存放 Upload FileType（如 "6"）
		FileName:         req.TargetFileNameTemplate, // 复用字段存放目标文件名模板
		Status:           TaskPending,
		CreateStatus:     "active",
		CreateUser:       req.CreateUser,
		TotalCount:       len(req.DeviceIDs),
		MaxConcurrent:    concurrency,
	}
	if err := s.taskRepo.Create(ctx, mainTask); err != nil {
		return nil, fmt.Errorf("create log collect main task: %w", err)
	}

	subTasks := make([]*UpgradeSubTask, 0, len(req.DeviceIDs))
	for _, deviceID := range req.DeviceIDs {
		subTask := &UpgradeSubTask{
			TaskID:     mainTask.ID,
			DeviceID:   deviceID,
			Status:     UpgradePending,
			MaxRetries: 3,
		}
		if dev, err := s.deviceRepo.GetByID(ctx, deviceID); err == nil {
			subTask.DeviceSN = dev.SerialNumber
		}
		subTasks = append(subTasks, subTask)
	}

	if err := s.subTaskRepo.BatchCreate(ctx, subTasks); err != nil {
		return nil, s.finalizeSubTaskFailure(ctx, mainTask.ID, fmt.Errorf("batch create log collect sub-tasks: %w", err))
	}

	if req.CreateSuspended {
		s.logger.Info("log collect task created in pending (suspended) mode",
			zap.String("task_id", mainTask.ID.String()),
			zap.Int("device_count", len(req.DeviceIDs)))
		return mainTask, nil
	}

	if err := s.taskRepo.UpdateStatus(ctx, mainTask.ID, TaskInProgress, ""); err != nil {
		s.logger.Error("update log collect task to in_progress", zap.Error(err))
	}
	mainTask.Status = TaskInProgress

	s.startCollectExecution(mainTask, subTasks, req.TransportPath, req.RPCType, req.ParamPath, concurrency)

	return mainTask, nil
}

// PlaceholderTrackingRequest 是 CreatePlaceholderTrackingTask 的入参。
type PlaceholderTrackingRequest struct {
	DeviceIDs []uuid.UUID
	TaskName  string
	// TypeCode 是 UFTE 任务类型编码（如 "LICENSE_UPGRADE" / "CONFIG_RESTORE"）。
	// 仅用于派生 sub_task.CommandKey 前缀，让派发 device_task 时能用同一份 key
	// 让 TC 回流能命中 sub_task → 自动推进。
	TypeCode string
	// FileType 存进 upgrade_tasks.download_file_type 列，给 UFTE 列表的
	// resolveTaskType(catalog, taskType, productClass, fileType) 精确回找
	// catalog 条目。对 CONFIG_RESTORE 是 "10 <OUI> Configuration File" 这种
	// 字面值（含 <OUI> 占位），与 dispatcher 真实下发 FileType（含真实 OUI）解耦。
	FileType   string
	CreateUser string
	Suspended  bool
}

// CreatePlaceholderTrackingTask 创建一个"已派发完毕"的占位 upgrade_tasks / sub_tasks，
// 给 LICENSE_UPGRADE / CONFIG_RESTORE 等"绕过 software executor 直接派发 device_tasks"
// 的任务类型在 UFTE 列表里可见。
//
// 行为：
//   - 主任务 TaskType=LogCollect（让 ufte loadAllTasks 的 typeSet 过滤包含它）
//   - 主任务 Status=ended / Result=success（因为派发是同步、瞬时的）
//   - 子任务 Status=completed（一并算 ended）
//   - 实际设备下发结果在 device_tasks 里独立跟踪，本占位行不会自动更新
//
// 这是为可见性服务的 MVP；真正的端到端进度跟踪需要扩展 TransferCompleteRouter
// 识别 LICENSE_UPGRADE / CONFIG_RESTORE 的 CommandKey 并回写 sub_task 状态。
func (s *SoftwareService) CreatePlaceholderTrackingTask(
	ctx context.Context, req PlaceholderTrackingRequest,
) (*UpgradeTask, error) {
	now := model.Time(time.Now())
	// 三态：
	//   - Suspended=true  → 任务"已挂起" / sub_tasks "待派发"；不增计数
	//   - Suspended=false → 任务"进行中" / sub_tasks "下载中"；不增计数
	//     （派发成功 ≠ 设备已应用，真正完成态要靠 TransferComplete 路由推进，
	//     这条线尚未接，先用 in_progress 表示"已派发等设备落地"，进度 0%）
	initialStatus := TaskInProgress
	subStatus := UpgradeDownloading
	if req.Suspended {
		initialStatus = TaskSuspended
		subStatus = UpgradePending
	}
	main := &UpgradeTask{
		TaskName:         req.TaskName,
		TaskType:         TaskTypeLogCollect, // 复用使 typeSet 过滤覆盖到本任务
		DownloadFileType: req.FileType,       // 用 catalog FileType 字面值（含 <OUI> 占位）
		Status:           initialStatus,
		CreateStatus:     "active",
		CreateUser:       req.CreateUser,
		TotalCount:       len(req.DeviceIDs),
		MaxConcurrent:    1,
		StartedAt:        &now,
	}
	// route hint 用 FileType（catalog 字面）：当前 router 没匹配的话 fallback 到默认 upgrade_tasks 表
	ctx = WithRouteHint(ctx, req.FileType)
	if err := s.taskRepo.Create(ctx, main); err != nil {
		return nil, fmt.Errorf("create placeholder tracking task: %w", err)
	}
	// pg_task_repository.go::Create 只插了部分列（status / total_count），
	// 这里再用 UpdateStatus 把 status 真正落库（Create 已写但保险起见）。
	if err := s.taskRepo.UpdateStatus(ctx, main.ID, initialStatus, ""); err != nil {
		s.logger.Warn("update placeholder task status failed",
			zap.String("task_id", main.ID.String()), zap.Error(err))
	}

	subs := make([]*UpgradeSubTask, 0, len(req.DeviceIDs))
	for _, did := range req.DeviceIDs {
		sub := &UpgradeSubTask{
			TaskID:   main.ID,
			DeviceID: did,
			Status:   subStatus,
		}
		if dev, derr := s.deviceRepo.GetByID(ctx, did); derr == nil && dev != nil {
			sub.DeviceSN = dev.SerialNumber
			// 关键：CommandKey 用统一格式，让派发 device_tasks 时使用同一 key →
			// TC 到达后 handleTCBody → GetByCommandKey 命中 sub_task → 自动推进。
			sub.CommandKey = BuildDirectDispatchCommandKey(req.TypeCode, main.ID, dev.SerialNumber)
		}
		subs = append(subs, sub)
	}
	if err := s.subTaskRepo.BatchCreate(ctx, subs); err != nil {
		return nil, s.finalizeSubTaskFailure(ctx, main.ID,
			fmt.Errorf("create placeholder sub_tasks: %w", err))
	}
	return main, nil
}

// BuildDirectDispatchCommandKey 给 CONFIG_RESTORE / LICENSE_UPGRADE 等"直接派发
// device_tasks 的 LogCollect 类任务"生成统一 CommandKey。sub_task 和 device_task
// 都用同一份 key，TC 到达时 handleTCBody → GetByCommandKey 才能命中 sub_task 推进。
//
// 格式：<typeCode>_<upgradeTaskID 前 8 hex>_<deviceSN>
func BuildDirectDispatchCommandKey(typeCode string, upgradeTaskID uuid.UUID, deviceSN string) string {
	tid := strings.ReplaceAll(upgradeTaskID.String(), "-", "")
	if len(tid) >= 8 {
		tid = tid[:8]
	}
	return fmt.Sprintf("%s_%s_%s", typeCode, tid, deviceSN)
}

// FinalizePlaceholderTrackingTask 把一个 suspended 状态的占位任务推进到"已派发"
// 状态（task=in_progress，sub_tasks=downloading）—— 与 immediate 模式创建后的状态
// 完全一致。在 UFTE StartTask 内 direct-dispatch 路径派发完后调用。
//
// 故意不推到 TaskEnded：派发成功 ≠ 设备已下载完成并应用 license/config，
// 真正完成态要靠后续接入 TransferCompleteRouter 据 device_tasks TC 推进。
func (s *SoftwareService) FinalizePlaceholderTrackingTask(
	ctx context.Context, taskID uuid.UUID, totalCount int,
) error {
	_ = totalCount // 保留入参以保后向兼容；状态推进仅依赖 ID
	if err := s.taskRepo.UpdateStatus(ctx, taskID, TaskInProgress, ""); err != nil {
		return fmt.Errorf("update status to in_progress: %w", err)
	}
	// 子任务从 pending → downloading
	page, err := s.subTaskRepo.ListByTaskID(ctx, taskID, SubTaskFilter{})
	if err != nil {
		s.logger.Warn("list sub_tasks for finalize failed", zap.Error(err))
		return nil
	}
	for _, sub := range page.Items {
		if sub.Status != UpgradePending {
			continue // 已不是 pending（可能已 completed/failed 由其他链路推进过）
		}
		if upErr := s.subTaskRepo.UpdateStatus(ctx, sub.ID, UpgradeDownloading, ""); upErr != nil {
			s.logger.Warn("update sub_task to downloading failed",
				zap.String("sub_task_id", sub.ID.String()), zap.Error(upErr))
		}
	}
	return nil
}

// startCollectExecution 启动日志采集子任务的并发执行 goroutine。按 RPCType 分流：
// "SET_PARAM_VALUES" 走 ExecuteOneSetParamCollect（如 FAULT_LOG_COLLECT），其余走
// ExecuteOneUpload（默认 Upload RPC 链路，包含 RUNTIME_LOG_COLLECT / CONFIG_BACKUP_*）。
func (s *SoftwareService) startCollectExecution(mainTask *UpgradeTask, subTasks []*UpgradeSubTask, transportPath, rpcType, paramPath string, concurrency int) {
	if concurrency < 1 {
		concurrency = 5
	}
	useSPV := strings.EqualFold(strings.TrimSpace(rpcType), "SET_PARAM_VALUES")
	sem := make(chan struct{}, concurrency)
	for i := range subTasks {
		sem <- struct{}{}
		go func(st *UpgradeSubTask) {
			defer func() { <-sem }()
			defer func() {
				if r := recover(); r != nil {
					s.logger.Error("log collect executor panic",
						zap.String("sub_task_id", st.ID.String()),
						zap.Any("recover", r))
				}
			}()
			if useSPV {
				s.executor.ExecuteOneSetParamCollect(context.Background(), st, paramPath, transportPath)
				return
			}
			s.executor.ExecuteOneUpload(context.Background(), st, mainTask.DownloadFileType, mainTask.FileName, transportPath)
		}(subTasks[i])
	}
}

// BatchUpgrade creates a main upgrade task and sub-tasks for each device, then starts execution.
func (s *SoftwareService) BatchUpgrade(ctx context.Context, req BatchUpgradeRequest) (*UpgradeTask, error) {
	fw, err := s.firmwareRepo.GetByID(ctx, req.FirmwareID)
	if err != nil {
		return nil, fmt.Errorf("get firmware: %w", err)
	}

	concurrency := req.Concurrency
	if concurrency < 1 {
		concurrency = 5
	}

	taskType := req.TaskType
	if taskType == 0 {
		taskType = TaskTypeUpgrade
	}

	mainTask := &UpgradeTask{
		TaskName:         req.TaskName,
		TaskType:         taskType,
		FirmwareID:       &req.FirmwareID,
		DownloadFileType: req.DownloadFileType,
		FileName:         fw.FileName,
		FileMD5:          fw.MD5Val,
		Status:           TaskPending,
		ProductClass:     fw.ProductClass,
		IsKeepConfig:     req.IsKeepConfig,
		CreateStatus:     "active",
		CreateUser:       "system",
		TotalCount:       len(req.DeviceIDs),
		MaxConcurrent:    concurrency,
	}
	if err := s.taskRepo.Create(ctx, mainTask); err != nil {
		return nil, fmt.Errorf("create main task: %w", err)
	}

	subTasks := make([]*UpgradeSubTask, 0, len(req.DeviceIDs))
	for _, deviceID := range req.DeviceIDs {
		subTask := &UpgradeSubTask{
			TaskID:      mainTask.ID,
			DeviceID:    deviceID,
			FirmwareID:  &req.FirmwareID,
			Status:      UpgradePending,
			MaxRetries:  3,
			DestVersion: fw.Version,
		}
		if dev, err := s.deviceRepo.GetByID(ctx, deviceID); err == nil {
			subTask.DeviceSN = dev.SerialNumber
			subTask.OriVersion = dev.FirmwareVersion
		}
		subTasks = append(subTasks, subTask)
	}

	if err := s.subTaskRepo.BatchCreate(ctx, subTasks); err != nil {
		return nil, s.finalizeSubTaskFailure(ctx, mainTask.ID, fmt.Errorf("batch create sub-tasks: %w", err))
	}

	if req.CreateSuspended {
		s.logger.Info("upgrade task created in pending (suspended) mode",
			zap.String("task_id", mainTask.ID.String()),
			zap.Int("device_count", len(req.DeviceIDs)))
		return mainTask, nil
	}

	// Canary strategy (T-0018 / R-101): persist stage metadata before kicking
	// execution. The execution layer still launches all sub-tasks today; the
	// canary monitor (canary_monitor.go) gates stage progression by failure-
	// rate thresholds and triggers SuspendUpgrade when crossed. Subsequent PR
	// extends startExecution to start only stage-N devices.
	if req.Strategy == StrategyCanary {
		stages := req.CanaryStages
		if len(stages) == 0 {
			stages = append([]CanaryStage(nil), DefaultCanaryStages...)
		}
		if err := ValidateStages(stages); err != nil {
			return nil, fmt.Errorf("validate canary stages: %w", err)
		}
		if err := s.taskRepo.UpdateCanaryFields(ctx, mainTask.ID, &CanaryFields{
			TaskID:             mainTask.ID,
			Strategy:           StrategyCanary,
			Stages:             stages,
			CurrentStage:       1,
			StageStatus:        StageStatusRunning,
			StageHistory:       []StageHistoryEntry{},
			AutoAdvance:        req.AutoAdvance,
			AutoAdvanceMinutes: req.AutoAdvanceMinutes,
			RollbackOnFailure:  req.RollbackOnFailure,
			TotalCount:         mainTask.TotalCount,
		}); err != nil {
			return nil, fmt.Errorf("init canary fields: %w", err)
		}
		s.logger.Info("canary upgrade started",
			zap.String("task_id", mainTask.ID.String()),
			zap.Int("total_devices", mainTask.TotalCount),
			zap.Int("stage_count", len(stages)),
			zap.Bool("auto_advance", req.AutoAdvance),
			zap.Bool("rollback_on_failure", req.RollbackOnFailure),
		)
	}

	if err := s.taskRepo.UpdateStatus(ctx, mainTask.ID, TaskInProgress, ""); err != nil {
		s.logger.Error("update main task to in_progress", zap.Error(err))
	}
	mainTask.Status = TaskInProgress

	s.startExecution(mainTask, subTasks, fw, concurrency)

	return mainTask, nil
}

// =============================================================================
// Canary stage transitions (T-0018 / R-101)
// =============================================================================

// AdvanceCanaryStage promotes a paused/running canary task to the next stage.
// Returns ErrInvalidInput when the task is not on the canary path or already
// completed/aborted.
func (s *SoftwareService) AdvanceCanaryStage(ctx context.Context, taskID uuid.UUID) error {
	fields, err := s.taskRepo.GetCanaryFields(ctx, taskID)
	if err != nil {
		return fmt.Errorf("get canary fields: %w", err)
	}
	if !fields.IsCanary() {
		return fmt.Errorf("task is not canary: %w", commonerrors.ErrInvalidInput)
	}
	if fields.StageStatus == StageStatusCompleted || fields.StageStatus == StageStatusAborted {
		return fmt.Errorf("canary task already terminal (%s): %w", fields.StageStatus, commonerrors.ErrInvalidInput)
	}
	next := fields.CurrentStage + 1
	if next > len(fields.Stages) {
		// All stages exhausted → mark completed.
		fields.StageStatus = StageStatusCompleted
		fields.StageHistory = append(fields.StageHistory, StageHistoryEntry{
			Stage: fields.CurrentStage, Action: "completed", At: nowFunc(),
		})
		s.logger.Info("canary task completed all stages", zap.String("task_id", taskID.String()))
	} else {
		fields.StageHistory = append(fields.StageHistory, StageHistoryEntry{
			Stage:   fields.CurrentStage,
			Percent: fields.Stages[fields.CurrentStage-1].Percent,
			Action:  "advanced",
			At:      nowFunc(),
		})
		fields.CurrentStage = next
		fields.StageStatus = StageStatusRunning
		s.logger.Info("canary task advanced",
			zap.String("task_id", taskID.String()),
			zap.Int("new_stage", next),
			zap.Int("percent", fields.Stages[next-1].Percent),
		)
	}
	if s.canaryMetrics != nil {
		s.canaryMetrics.RecordAdvance("advanced")
	}
	return s.taskRepo.UpdateCanaryFields(ctx, taskID, fields)
}

// PauseCanaryStage pauses a running canary task. In-flight sub-tasks continue;
// no new stage promotion until ResumeCanaryStage / AbortCanary is called.
func (s *SoftwareService) PauseCanaryStage(ctx context.Context, taskID uuid.UUID) error {
	return s.transitionCanaryStatus(ctx, taskID, StageStatusPaused, "paused", "manual_pause")
}

// ResumeCanaryStage moves a paused canary back to running.
func (s *SoftwareService) ResumeCanaryStage(ctx context.Context, taskID uuid.UUID) error {
	return s.transitionCanaryStatus(ctx, taskID, StageStatusRunning, "resumed", "manual_resume")
}

// AbortCanary terminates remaining stages of a canary task. Already-running
// sub-tasks are not killed — operators must call SuspendUpgrade if they
// also want to halt in-flight executions.
func (s *SoftwareService) AbortCanary(ctx context.Context, taskID uuid.UUID) error {
	return s.transitionCanaryStatus(ctx, taskID, StageStatusAborted, "aborted", "manual_abort")
}

func (s *SoftwareService) transitionCanaryStatus(ctx context.Context, taskID uuid.UUID, target, action, reason string) error {
	fields, err := s.taskRepo.GetCanaryFields(ctx, taskID)
	if err != nil {
		return fmt.Errorf("get canary fields: %w", err)
	}
	if !fields.IsCanary() {
		return fmt.Errorf("task is not canary: %w", commonerrors.ErrInvalidInput)
	}
	if fields.StageStatus == StageStatusCompleted || fields.StageStatus == StageStatusAborted {
		return fmt.Errorf("canary task already terminal (%s): %w", fields.StageStatus, commonerrors.ErrInvalidInput)
	}
	prev := fields.StageStatus
	fields.StageStatus = target
	fields.StageHistory = append(fields.StageHistory, StageHistoryEntry{
		Stage:  fields.CurrentStage,
		Action: action,
		At:     nowFunc(),
		Reason: reason,
	})
	s.logger.Info("canary task status changed",
		zap.String("task_id", taskID.String()),
		zap.String("from", prev),
		zap.String("to", target),
		zap.String("reason", reason),
	)
	if s.canaryMetrics != nil {
		s.canaryMetrics.RecordAdvance(action)
	}
	return s.taskRepo.UpdateCanaryFields(ctx, taskID, fields)
}

// nowFunc is overridable for tests; production reads time.Now.
var nowFunc = func() time.Time { return time.Now() }

// startExecution launches goroutines to execute upgrade sub-tasks with bounded concurrency.
func (s *SoftwareService) startExecution(mainTask *UpgradeTask, subTasks []*UpgradeSubTask, fw *FirmwareVersion, concurrency int) {
	if concurrency < 1 {
		concurrency = 5
	}
	isKeepConfig := mainTask.IsKeepConfig
	sem := make(chan struct{}, concurrency)
	for i := range subTasks {
		sem <- struct{}{}
		go func(st *UpgradeSubTask) {
			defer func() { <-sem }()
			defer func() {
				if r := recover(); r != nil {
					s.logger.Error("upgrade executor panic",
						zap.String("sub_task_id", st.ID.String()),
						zap.Any("recover", r))
				}
			}()
			s.executor.ExecuteOne(context.Background(), st, fw, isKeepConfig, mainTask.DownloadFileType)
		}(subTasks[i])
	}
}

// startRollbackExecution launches goroutines to execute rollback sub-tasks with bounded concurrency.
func (s *SoftwareService) startRollbackExecution(subTasks []*UpgradeSubTask) {
	concurrency := 5
	sem := make(chan struct{}, concurrency)
	for i := range subTasks {
		sem <- struct{}{}
		go func(st *UpgradeSubTask) {
			defer func() { <-sem }()
			defer func() {
				if r := recover(); r != nil {
					s.logger.Error("rollback executor panic",
						zap.String("sub_task_id", st.ID.String()),
						zap.Any("recover", r))
				}
			}()

			dev, err := s.deviceRepo.GetByID(context.Background(), st.DeviceID)
			if err != nil {
				s.rollbackExec.FailRollbackSubTask(context.Background(), st, fmt.Sprintf("device not found: %v", err), FailureDeviceNotFound)
				return
			}

			tech := model.TechLTE
			if Is5G(dev) {
				tech = model.TechNR
			}

			// 4G 走两阶段（GPV ROLLBACK_ENABLE → SPV ROLLBACK_CONTROL）；5G 直接 SPV。
			// 路径与值统一在 RollbackExecutor 内部经 adapter + Translator 决定。
			s.rollbackExec.RollbackOne(context.Background(), st, dev, tech)
		}(subTasks[i])
	}
}

// HandleTransferComplete advances the upgrade state machine when a device reports transfer complete.
// ACS publishes two kinds of TC events on the same subject:
//  1. Inform-level (from publishInformEvents): payload has device_sn but no TC body data
//  2. TC SOAP body level (from handleTransferComplete): payload is tr069.TransferComplete with
//     command_key, fault_struct (FaultCode/FaultString), start_time, complete_time — but no device_sn
//
// Both are needed: #2 carries fault information, #1 carries device identity.
// We try to decode both formats and route accordingly.
func (s *SoftwareService) HandleTransferComplete(ctx context.Context, evt event.Event) error {
	s.logger.Info("handling TransferComplete event", zap.String("subject", evt.Subject))

	// Try decoding as TC SOAP body (has command_key + optional fault_struct)
	var tcPayload struct {
		CommandKey  string `json:"command_key"`
		FaultStruct *struct {
			FaultCode   int    `json:"fault_code"`
			FaultString string `json:"fault_string"`
		} `json:"fault_struct,omitempty"`
		StartTime    time.Time `json:"start_time"`
		CompleteTime time.Time `json:"complete_time"`
	}
	hasTCBody := evt.DecodePayload(&tcPayload) == nil && tcPayload.CommandKey != ""

	// Try decoding as Inform-level event (has device_sn)
	var informPayload struct {
		DeviceID struct {
			SerialNumber string `json:"SerialNumber"`
		} `json:"device_id"`
	}
	hasInformBody := evt.DecodePayload(&informPayload) == nil && informPayload.DeviceID.SerialNumber != ""

	// Route to the appropriate handler
	if hasTCBody {
		return s.handleTCBody(ctx, tcPayload.CommandKey, tcPayload.FaultStruct)
	}
	if hasInformBody {
		return s.handleTCInform(ctx, informPayload.DeviceID.SerialNumber)
	}
	return nil
}

// handleTCBody processes TC SOAP body events (from ACS handleTransferComplete).
// These carry command_key and optional fault_struct.
func (s *SoftwareService) handleTCBody(ctx context.Context, commandKey string, fault *struct {
	FaultCode   int    `json:"fault_code"`
	FaultString string `json:"fault_string"`
}) error {
	// Find sub-task by command key
	subTask, err := s.subTaskRepo.GetByCommandKey(ctx, commandKey)
	if err != nil {
		return nil // Not our task
	}

	// Fault 判定不能只看 FaultCode！部分厂商 CPE（实测 baicells/MMMM 系列）
	// 在 TC 失败场景下回传 <FaultCode></FaultCode> 空字符串（违反 TR-069 spec：
	// spec 要求成功填 0、失败填具体 code 数字），仅在 FaultString 里塞失败描述
	// 如 "Download fail with exit status 1"。旧实现 `fault.FaultCode != 0` 把这种
	// 空 code 当成 0/成功，导致设备其实没传文件、sub_task 却被标 completed、
	// managed_files 落地零文件 —— 前端"已完成"但下载列表空。
	// 修复：FaultCode != 0 OR FaultString trim 后非空，二者满足其一即视为失败。
	faultString := ""
	if fault != nil {
		faultString = strings.TrimSpace(fault.FaultString)
	}
	hasFault := fault != nil && (fault.FaultCode != 0 || faultString != "")

	s.logger.Info("TC matched sub-task",
		zap.String("sub_task_id", subTask.ID.String()),
		zap.String("command_key", commandKey),
		zap.String("status", string(subTask.Status)),
		zap.Bool("has_fault", hasFault))

	// 只处理在途状态：Downloading（升级 / 回滚类）或 Uploading（备份 / 日志采集类）。
	// 其它状态意味着事件迟到或已被 reaper 兜底，忽略即可。
	if subTask.Status != UpgradeDownloading && subTask.Status != UpgradeUploading {
		return nil
	}

	// TC with fault → fail immediately
	if hasFault {
		reason := fmt.Sprintf("Upgrade failed, there is FaultString in TransferComplete msg. FaultCode: %d, FaultString: %s", fault.FaultCode, faultString)
		s.executor.failSubTask(ctx, subTask, reason, FailureTCFault)
		return nil
	}

	// TC without fault — find device to determine 4G/5G
	if subTask.DeviceSN == "" {
		s.logger.Warn("TC body has no device SN, cannot determine 4G/5G",
			zap.String("sub_task_id", subTask.ID.String()))
		return nil
	}

	dev, err := s.deviceRepo.GetBySerialNumber(ctx, subTask.DeviceSN)
	if err != nil || dev == nil {
		return nil
	}

	return s.advanceAfterTC(ctx, subTask, dev)
}

// handleTCInform processes Inform-level TC events (from ACS publishInformEvents).
// These carry device_sn but no fault information.
func (s *SoftwareService) handleTCInform(ctx context.Context, deviceSN string) error {
	dev, err := s.deviceRepo.GetBySerialNumber(ctx, deviceSN)
	if err != nil {
		return fmt.Errorf("get device by SN: %w", err)
	}
	if dev == nil {
		return fmt.Errorf("device not found: %s", deviceSN)
	}

	subTask, err := s.subTaskRepo.GetActiveByDeviceID(ctx, dev.ID)
	if err != nil {
		return nil
	}

	// 同 handleTCBody：在途阶段（下载 / 上传）才接受 TC 事件推进。
	if subTask.Status != UpgradeDownloading && subTask.Status != UpgradeUploading {
		return nil
	}

	return s.advanceAfterTC(ctx, subTask, dev)
}

// advanceAfterTC handles the 4G/5G branching after a successful TransferComplete.
func (s *SoftwareService) advanceAfterTC(ctx context.Context, subTask *UpgradeSubTask, dev *model.Device) error {
	// For log collection (Upload RPC), TransferComplete means the file was successfully
	// uploaded by the device. No reboot is needed — just complete the subtask.
	parentTask, err := s.taskRepo.GetByID(ctx, subTask.TaskID)
	if err == nil && parentTask.TaskType == TaskTypeLogCollect {
		s.executor.completeSubTask(ctx, subTask, dev.SerialNumber)
		return nil
	}

	is5G := Is5G(dev)
	nextState := NextStateAfterTC(is5G)

	if err := ValidateUpgradeTransition(subTask.Status, nextState); err != nil {
		s.logger.Warn("invalid upgrade transition",
			zap.String("sub_task_id", subTask.ID.String()),
			zap.String("current", string(subTask.Status)),
			zap.String("target", string(nextState)))
		return nil
	}

	// Clean up download progress flag
	if s.redis != nil {
		dlKey := fmt.Sprintf("DownloadingFlag_%s_%s", subTask.TaskID.String(), dev.SerialNumber)
		s.redis.Del(ctx, dlKey)
	}

	if nextState == UpgradeCompleted {
		// 4G: TC success = upgrade complete
		s.executor.completeSubTask(ctx, subTask, dev.SerialNumber)

		if completedEvt, err := event.NewEvent(event.SubjectUpgradeCompleted, map[string]interface{}{
			"sub_task_id": subTask.ID.String(),
			"task_id":     subTask.TaskID.String(),
			"device_id":   dev.ID.String(),
		}); err == nil {
			if pubErr := s.eventBus.Publish(ctx, event.SubjectUpgradeCompleted, completedEvt); pubErr != nil {
				s.logger.Warn("publish upgrade.completed event", zap.Error(pubErr))
			}
		}
	} else {
		// 5G: TC success = file downloaded, device will now install and reboot
		if err := s.subTaskRepo.UpdateStatus(ctx, subTask.ID, nextState, ""); err != nil {
			return fmt.Errorf("update sub-task to rebooting: %w", err)
		}

		// Clean up TC flag
		if s.redis != nil {
			tcKey := fmt.Sprintf("TransferCompleteReq_%s", subTask.ID.String())
			s.redis.Del(ctx, tcKey)
		}

		s.logger.Info("5G download complete, waiting for 102 UPGRADE FINISH",
			zap.String("sub_task_id", subTask.ID.String()),
			zap.String("device_sn", dev.SerialNumber))
	}

	return nil
}

// checkAndFinalizeTask checks if a main task is complete and updates its final status.
func (s *SoftwareService) checkAndFinalizeTask(ctx context.Context, taskID uuid.UUID) {
	finalizeTask(ctx, s.taskRepo, s.logger, taskID)
}

// SuspendUpgrade suspends all active sub-tasks under a main task.
// 历史 bug：旧实现只改 upgrade_tasks 主任务 status，**完全不动 upgrade_sub_tasks**——
// 前端「设备状态」列读 sub_task.status，所以暂停后页面依然显示"上传中"/"下载中"，
// 用户觉得"操作没生效"，再点终止时前端按状态过滤又挡住了，链路彻底卡住。
//
// 修复：同步把 active sub_tasks（downloading / uploading / rebooting / pending）翻成
// Suspended。已 Terminal（completed / failed / terminated）的跳过；已 Suspended 的
// 也跳过避免无效写库。
func (s *SoftwareService) SuspendUpgrade(ctx context.Context, taskID uuid.UUID) error {
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return fmt.Errorf("get upgrade task: %w", err)
	}

	if task.Status == TaskEnded || task.Status == TaskSuspended {
		return commonerrors.NewBusinessError(8003, fmt.Sprintf("cannot suspend task in %s state", task.Status), commonerrors.ErrInvalidInput)
	}

	if err := s.taskRepo.UpdateStatus(ctx, taskID, TaskSuspended, ""); err != nil {
		return fmt.Errorf("suspend main task: %w", err)
	}

	suspendedCount := 0
	for page := 1; ; page++ {
		subResult, err := s.subTaskRepo.ListByTaskID(ctx, taskID, SubTaskFilter{
			TaskID:      taskID,
			ListRequest: model.ListRequest{Page: page, PageSize: 100},
		})
		if err != nil {
			return fmt.Errorf("list sub-tasks for suspend: %w", err)
		}
		for i := range subResult.Items {
			subTask := subResult.Items[i]
			if IsUpgradeTerminal(subTask.Status) || subTask.Status == UpgradeSuspended {
				continue
			}
			if err := s.subTaskRepo.UpdateStatus(ctx, subTask.ID, UpgradeSuspended, "task suspended by operator"); err != nil {
				return fmt.Errorf("suspend sub-task %s: %w", subTask.ID.String(), err)
			}
			suspendedCount++
		}
		if len(subResult.Items) == 0 || page >= subResult.TotalPages {
			break
		}
	}

	s.logger.Info("upgrade task suspended",
		zap.String("task_id", taskID.String()),
		zap.Int("sub_tasks_suspended", suspendedCount))
	return nil
}

// ResumeUpgrade resumes a suspended or pending main task and starts execution.
func (s *SoftwareService) ResumeUpgrade(ctx context.Context, taskID uuid.UUID) error {
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return fmt.Errorf("get upgrade task: %w", err)
	}

	if task.Status != TaskSuspended && task.Status != TaskPending {
		return commonerrors.NewBusinessError(8004, "task is not suspended or pending", commonerrors.ErrInvalidInput)
	}

	// 同时接受 Pending 和 Suspended sub-tasks——SuspendUpgrade 修复后会把 active 的
	// downloading/rebooting 子任务翻成 Suspended，Resume 需要把它们也捞回来。
	// 这与 ResumeCollect 行为对齐。
	subResult, err := s.subTaskRepo.ListByTaskID(ctx, taskID, SubTaskFilter{
		TaskID:   taskID,
		Statuses: []UpgradeState{UpgradePending, UpgradeSuspended},
	})
	if err != nil {
		return fmt.Errorf("list pending/suspended sub-tasks: %w", err)
	}

	if len(subResult.Items) == 0 {
		return commonerrors.NewBusinessError(8010, "no pending or suspended sub-tasks to execute", commonerrors.ErrInvalidInput)
	}

	subTasks := make([]*UpgradeSubTask, len(subResult.Items))
	for i := range subResult.Items {
		subTasks[i] = &subResult.Items[i].UpgradeSubTask
	}

	if err := s.taskRepo.UpdateStatus(ctx, taskID, TaskInProgress, ""); err != nil {
		return fmt.Errorf("resume main task: %w", err)
	}

	if task.TaskType == TaskTypeRollback {
		s.startRollbackExecution(subTasks)
	} else {
		var fw *FirmwareVersion
		if task.FirmwareID != nil {
			fw, err = s.firmwareRepo.GetByID(ctx, *task.FirmwareID)
			if err != nil {
				return fmt.Errorf("get firmware for resume: %w", err)
			}
		}
		if fw == nil {
			return commonerrors.NewBusinessError(8010, "firmware not found for upgrade task", commonerrors.ErrInvalidInput)
		}
		s.startExecution(task, subTasks, fw, task.MaxConcurrent)
	}

	s.logger.Info("upgrade task resumed", zap.String("task_id", taskID.String()))
	return nil
}

// TerminateUpgrade force-stops a main task and all its sub-tasks.
func (s *SoftwareService) TerminateUpgrade(ctx context.Context, taskID uuid.UUID) error {
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return fmt.Errorf("get upgrade task: %w", err)
	}

	if task.Status == TaskEnded {
		return commonerrors.NewBusinessError(8005, "task already ended", commonerrors.ErrInvalidInput)
	}

	for page := 1; ; page++ {
		subResult, err := s.subTaskRepo.ListByTaskID(ctx, taskID, SubTaskFilter{
			TaskID: taskID,
			ListRequest: model.ListRequest{
				Page:     page,
				PageSize: 100,
			},
		})
		if err != nil {
			return fmt.Errorf("list sub-tasks for terminate: %w", err)
		}
		for i := range subResult.Items {
			subTask := subResult.Items[i]
			if IsUpgradeTerminal(subTask.Status) {
				continue
			}
			if err := s.subTaskRepo.UpdateStatus(ctx, subTask.ID, UpgradeTerminated, "task terminated by operator"); err != nil {
				return fmt.Errorf("terminate sub-task %s: %w", subTask.ID.String(), err)
			}
			// 释放设备级 Redis 锁，否则用户终止后 1 小时（lock TTL）内重试都会撞
			// "升级无法启动，设备不能运行多个升级任务"。failSubTask 路径会顺带 Del 是巧合，
			// 不能依赖。注意 sub_task 行刚拿出来时 DeviceSN 可能为空（还没进 ExecuteOne）→
			// 此时也没拿过锁，跳过即可。
			if subTask.DeviceSN != "" {
				key := fmt.Sprintf("software:upgrade:active:%s", subTask.DeviceSN)
				if err := s.redis.Del(ctx, key).Err(); err != nil {
					s.logger.Warn("release device lock on terminate",
						zap.String("sub_task_id", subTask.ID.String()),
						zap.String("device_sn", subTask.DeviceSN),
						zap.Error(err))
				}
			}
		}
		if len(subResult.Items) == 0 || page >= subResult.TotalPages {
			break
		}
	}

	if err := s.taskRepo.UpdateStatus(ctx, taskID, TaskEnded, TaskResultTerminated); err != nil {
		return fmt.Errorf("terminate main task: %w", err)
	}

	s.logger.Info("upgrade task terminated", zap.String("task_id", taskID.String()))
	return nil
}

// DeleteUpgrade permanently deletes a task and its sub-tasks.
// Allowed for ended, pending, and suspended tasks; in-progress tasks must be
// terminated first.
func (s *SoftwareService) DeleteUpgrade(ctx context.Context, taskID uuid.UUID) error {
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return fmt.Errorf("get upgrade task: %w", err)
	}

	if task.Status == TaskInProgress {
		return commonerrors.NewBusinessError(8011, "cannot delete a running task; terminate it first", commonerrors.ErrInvalidInput)
	}

	// 先回收任务关联的 MinIO 对象 + backup_restore_file 元数据（LogCollect 类
	// 任务：配置备份 / 运行日志 / 故障日志）。best-effort：失败仅 warn，不阻断
	// 主任务删除——孤儿文件比卡住删除按钮更可接受。
	if s.taskFileCleaner != nil {
		if err := s.taskFileCleaner(ctx, taskID); err != nil {
			s.logger.Warn("task file cleanup failed (best-effort)",
				zap.String("task_id", taskID.String()), zap.Error(err))
		}
	}

	if err := s.subTaskRepo.DeleteByTaskID(ctx, taskID); err != nil {
		return fmt.Errorf("delete sub-tasks: %w", err)
	}
	if err := s.taskRepo.Delete(ctx, taskID); err != nil {
		return fmt.Errorf("delete upgrade task: %w", err)
	}

	s.logger.Info("upgrade task deleted", zap.String("task_id", taskID.String()))
	return nil
}

// ResumeCollect resumes a suspended or pending log-collect / config-backup task.
// rpcType + paramPath 仅 SPV 模式（FAULT_LOG_COLLECT）使用；Upload 模式留空即可。
// transportPath is the Upload RPC path template stored in the UFTE task-type catalog;
// callers (ufte.Service.StartTask) must look it up before calling this method.
func (s *SoftwareService) ResumeCollect(ctx context.Context, taskID uuid.UUID, transportPath, rpcType, paramPath string) error {
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return fmt.Errorf("get upgrade task: %w", err)
	}

	if task.Status != TaskSuspended && task.Status != TaskPending {
		return commonerrors.NewBusinessError(8004, "task is not suspended or pending", commonerrors.ErrInvalidInput)
	}

	// Resume both pending and suspended sub-tasks (suspended = device was offline
	// when first attempted; they should be retried on resume).
	subResult, err := s.subTaskRepo.ListByTaskID(ctx, taskID, SubTaskFilter{
		TaskID:   taskID,
		Statuses: []UpgradeState{UpgradePending, UpgradeSuspended},
	})
	if err != nil {
		return fmt.Errorf("list pending sub-tasks: %w", err)
	}
	if len(subResult.Items) == 0 {
		return commonerrors.NewBusinessError(8010, "no pending sub-tasks to execute", commonerrors.ErrInvalidInput)
	}

	subTasks := make([]*UpgradeSubTask, len(subResult.Items))
	for i := range subResult.Items {
		subTasks[i] = &subResult.Items[i].UpgradeSubTask
	}

	if err := s.taskRepo.UpdateStatus(ctx, taskID, TaskInProgress, ""); err != nil {
		return fmt.Errorf("resume collect task: %w", err)
	}

	s.startCollectExecution(task, subTasks, transportPath, rpcType, paramPath, task.MaxConcurrent)
	s.logger.Info("collect task resumed", zap.String("task_id", taskID.String()))
	return nil
}

// RetryUpgrade retries failed sub-tasks under a main task.
func (s *SoftwareService) RetryUpgrade(ctx context.Context, taskID uuid.UUID) error {
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return fmt.Errorf("get upgrade task: %w", err)
	}

	if task.Status != TaskEnded {
		return commonerrors.NewBusinessError(8006, "can only retry ended tasks", commonerrors.ErrInvalidInput)
	}

	if err := s.taskRepo.UpdateStatus(ctx, taskID, TaskInProgress, ""); err != nil {
		return fmt.Errorf("retry main task: %w", err)
	}

	s.logger.Info("upgrade task retry initiated", zap.String("task_id", taskID.String()))
	return nil
}

// RollbackDevices creates a rollback task for the specified devices.
// Per-device technology detection determines 4G/5G-specific parameters.
//
// Audit fields (T-0021): Reason / Source / TargetFirmwareID are persisted on
// the parent task. Source defaults to "manual" when blank; an explicit value
// is validated against the canonical RollbackSource* set. When TargetFirmwareID
// is non-nil, all sub_tasks adopt that firmware's version as DestVersion (the
// device's previous firmware is still recorded as OriVersion).
func (s *SoftwareService) RollbackDevices(ctx context.Context, req RollbackRequest) (*UpgradeTask, error) {
	// Source validation: blank → manual; non-blank must be canonical.
	source := req.Source
	if source == "" {
		source = RollbackSourceManual
	} else if !IsValidRollbackSource(source) {
		return nil, fmt.Errorf("invalid rollback source %q: %w", source, commonerrors.ErrInvalidInput)
	}

	// Optional target firmware override: validate existence + capture version.
	var targetVersion string
	if req.TargetFirmwareID != nil {
		fw, err := s.firmwareRepo.GetByID(ctx, *req.TargetFirmwareID)
		if err != nil {
			return nil, fmt.Errorf("get target firmware: %w", err)
		}
		targetVersion = fw.Version
	}

	var productClass string
	if len(req.DeviceIDs) > 0 {
		if dev, err := s.deviceRepo.GetByID(ctx, req.DeviceIDs[0]); err == nil {
			productClass = dev.ProductClass
		}
	}

	mainTask := &UpgradeTask{
		TaskName:                 req.TaskName,
		TaskType:                 TaskTypeRollback,
		Status:                   TaskPending,
		ProductClass:             productClass,
		CreateStatus:             "active",
		CreateUser:               req.CreateUser,
		TotalCount:               len(req.DeviceIDs),
		MaxConcurrent:            5,
		RollbackReason:           req.Reason,
		RollbackSource:           source,
		RollbackTargetFirmwareID: req.TargetFirmwareID,
	}
	if err := s.taskRepo.Create(ctx, mainTask); err != nil {
		return nil, fmt.Errorf("create rollback task: %w", err)
	}

	subTasks := make([]*UpgradeSubTask, 0, len(req.DeviceIDs))
	for _, deviceID := range req.DeviceIDs {
		subTask := &UpgradeSubTask{
			TaskID:     mainTask.ID,
			DeviceID:   deviceID,
			Status:     UpgradePending,
			MaxRetries: 3,
		}
		if dev, err := s.deviceRepo.GetByID(ctx, deviceID); err == nil {
			subTask.DeviceSN = dev.SerialNumber
			subTask.OriVersion = dev.FirmwareVersion
		}
		// Target firmware override: explicit version trumps the legacy "back to
		// OriVersion" path. OriVersion still records the device's current state
		// for audit / failure recovery.
		if targetVersion != "" {
			subTask.DestVersion = targetVersion
			subTask.FirmwareID = req.TargetFirmwareID
		}
		subTasks = append(subTasks, subTask)
	}

	if err := s.subTaskRepo.BatchCreate(ctx, subTasks); err != nil {
		return nil, s.finalizeSubTaskFailure(ctx, mainTask.ID, fmt.Errorf("batch create rollback sub-tasks: %w", err))
	}

	// Metrics fire only once the task is fully persisted (parent + sub-tasks);
	// suspended-mode rollbacks are still real persisted artefacts so they count too.
	recordMetrics := func() {
		s.rollbackMetrics.RecordRollback(source, len(req.DeviceIDs))
		if req.TargetFirmwareID != nil {
			s.rollbackMetrics.RecordWithTarget()
		}
	}

	if req.CreateSuspended {
		recordMetrics()
		s.logger.Info("rollback task created in pending (suspended) mode",
			zap.String("task_id", mainTask.ID.String()),
			zap.Int("device_count", len(req.DeviceIDs)),
			zap.String("source", source),
			zap.String("reason", req.Reason),
			zap.Bool("with_target", req.TargetFirmwareID != nil))
		return mainTask, nil
	}

	if err := s.taskRepo.UpdateStatus(ctx, mainTask.ID, TaskInProgress, ""); err != nil {
		s.logger.Error("update rollback task to in_progress", zap.Error(err))
	}
	mainTask.Status = TaskInProgress

	// Metrics after the task transitioned to in_progress so dashboards reflect
	// rollbacks that actually entered execution (not just creation attempts).
	recordMetrics()

	s.startRollbackExecution(subTasks)

	s.logger.Info("rollback task created",
		zap.String("task_id", mainTask.ID.String()),
		zap.Int("device_count", len(req.DeviceIDs)),
		zap.String("source", source),
		zap.String("reason", req.Reason),
		zap.Bool("with_target", req.TargetFirmwareID != nil))

	return mainTask, nil
}

// TriggerCanaryFailureRollback is invoked by the canary monitor when a stage
// crosses its failure threshold AND the canary task opted into RollbackOnFailure.
//
// Devices already promoted (sub_tasks with status=Completed) are rolled back
// automatically with source=canary_failure. Failed/pending devices are skipped:
// failed ones never received the new firmware, pending ones haven't started.
func (s *SoftwareService) TriggerCanaryFailureRollback(ctx context.Context, taskID uuid.UUID, reason string) error {
	parent, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return fmt.Errorf("get canary task: %w", err)
	}

	// canaryAutoRollbackPageCap caps the auto-rollback batch size at 1000 devices.
	// Larger canary stages (>1000 promoted devices) require operators to fall
	// back to a manual paginated rollback — log.warn flags this so dashboards
	// surface the truncation. TODO(T-future): paginate this loop when canary
	// rollouts routinely exceed 1000 devices.
	const canaryAutoRollbackPageCap = 1000

	completed := UpgradeCompleted
	subResp, err := s.subTaskRepo.ListByTaskID(ctx, taskID, SubTaskFilter{
		TaskID: taskID,
		Status: &completed,
		ListRequest: model.ListRequest{
			Page:     1,
			PageSize: canaryAutoRollbackPageCap,
		},
	})
	if err != nil {
		return fmt.Errorf("list completed sub-tasks: %w", err)
	}
	if subResp == nil || len(subResp.Items) == 0 {
		s.logger.Info("canary auto-rollback skipped: no completed devices to roll back",
			zap.String("task_id", taskID.String()))
		return nil
	}
	if len(subResp.Items) >= canaryAutoRollbackPageCap {
		s.logger.Warn("canary auto-rollback page cap reached: rolling back first N devices only — operators must paginate manually for the rest",
			zap.String("task_id", taskID.String()),
			zap.Int("page_cap", canaryAutoRollbackPageCap),
			zap.Int64("total_completed", subResp.Total),
		)
	}

	deviceIDs := make([]uuid.UUID, 0, len(subResp.Items))
	for _, st := range subResp.Items {
		deviceIDs = append(deviceIDs, st.DeviceID)
	}

	rollbackName := fmt.Sprintf("%s-auto-rollback", parent.TaskName)
	if len(rollbackName) > 200 {
		rollbackName = rollbackName[:200]
	}
	req := RollbackRequest{
		DeviceIDs:  deviceIDs,
		TaskName:   rollbackName,
		CreateUser: "canary-monitor",
		Reason:     reason,
		Source:     RollbackSourceCanaryFailure,
	}
	if _, err := s.RollbackDevices(ctx, req); err != nil {
		return fmt.Errorf("auto-rollback: %w", err)
	}
	s.logger.Warn("canary task auto-rolled back (threshold exceeded + opt-in)",
		zap.String("canary_task_id", taskID.String()),
		zap.Int("device_count", len(deviceIDs)),
		zap.String("reason", reason),
	)
	return nil
}

// DownloadFirmware streams a firmware file from MinIO.
func (s *SoftwareService) DownloadFirmware(ctx context.Context, id uuid.UUID) (*minio.Object, error) {
	fw, err := s.firmwareRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get firmware: %w", err)
	}

	obj, err := s.minioClient.GetObject(ctx, s.firmwareBkt, fw.MinIOPath, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("get object from MinIO: %w", err)
	}
	return obj, nil
}

// DeleteFirmware removes a firmware record and deletes the file from MinIO.
func (s *SoftwareService) DeleteFirmware(ctx context.Context, id uuid.UUID) error {
	fw, err := s.firmwareRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("get firmware for delete: %w", err)
	}

	if err := s.firmwareRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete firmware record: %w", err)
	}

	if fw.MinIOPath != "" {
		if err := s.minioClient.RemoveObject(ctx, s.firmwareBkt, fw.MinIOPath, minio.RemoveObjectOptions{}); err != nil {
			s.logger.Warn("failed to delete firmware file from MinIO, orphaned object",
				zap.String("minio_path", fw.MinIOPath),
				zap.Error(err))
		}
	}

	return nil
}

// UpdateFirmwareMetadata updates a firmware version's metadata.
func (s *SoftwareService) UpdateFirmwareMetadata(ctx context.Context, fw *FirmwareVersion) error {
	return s.firmwareRepo.Update(ctx, fw)
}

// Subscribe registers all event subscriptions for the software service.
//
// 每个 subject 用独立 queue 名（durable consumer name），避免 "subject does
// not match consumer" 错误：NATS Durable Consumer 一旦绑定 FilterSubject，
// 其他 subject 用同名 durable 注册会被拒绝。
func (s *SoftwareService) Subscribe(eventBus event.EventBus) error {
	// TransferComplete — upgrade state advancement
	_, err := eventBus.QueueSubscribe(event.SubjectDeviceTransferComplete, "software-upgrade-transfer", func(ctx context.Context, evt event.Event) error {
		return s.HandleTransferComplete(ctx, evt)
	})
	if err != nil {
		s.logger.Warn("subscribe transfer_complete", zap.Error(err))
	}

	// Download response — detect download failures
	_, err = eventBus.QueueSubscribe(event.SubjectCommandDownloadResponse, "software-upgrade-download-resp", func(ctx context.Context, evt event.Event) error {
		return s.executor.HandleDownloadResponse(ctx, evt)
	})
	if err != nil {
		s.logger.Warn("subscribe download response", zap.Error(err))
	}

	// Upload response — detect upload rejection (config backup / log collect)
	_, err = eventBus.QueueSubscribe(event.SubjectCommandUploadResponse, "software-upgrade-upload-resp", func(ctx context.Context, evt event.Event) error {
		return s.executor.HandleUploadResponse(ctx, evt)
	})
	if err != nil {
		s.logger.Warn("subscribe upload response", zap.Error(err))
	}

	// RebootComplete — rollback completion and 4G upgrade
	_, err = eventBus.QueueSubscribe(event.SubjectDeviceRebootComplete, "software-upgrade-reboot", func(ctx context.Context, evt event.Event) error {
		return s.executor.HandleRebootComplete(ctx, evt)
	})
	if err != nil {
		s.logger.Warn("subscribe reboot_complete", zap.Error(err))
	}

	// 5G Upgrade Finish — 102 UPGRADE FINISH event
	_, err = eventBus.QueueSubscribe(event.SubjectDeviceUpgradeFinish, "software-upgrade-finish", func(ctx context.Context, evt event.Event) error {
		return s.executor.HandleUpgradeFinish(ctx, evt)
	})
	if err != nil {
		s.logger.Warn("subscribe upgrade_finish", zap.Error(err))
	}

	// Device periodic — check for pending upgrades on reconnect.
	// 每次心跳都触发；payload 含 device_id 嵌套对象（ACS publish）。
	_, err = eventBus.QueueSubscribe(event.SubjectDevicePeriodic, "software-upgrade-periodic", func(ctx context.Context, evt event.Event) error {
		return s.executor.HandleDeviceOnline(ctx, evt)
	})
	if err != nil {
		s.logger.Warn("subscribe device periodic", zap.Error(err))
	}

	// Device online — offline→active 切换的确定性信号（device 模块 publish，T-0123），
	// 比 periodic 更精准。两者同时订阅可形成兜底。详见
	// docs/project/backup-display-fix-20260520.md F13。
	_, err = eventBus.QueueSubscribe(event.SubjectDeviceOnline, "software-upgrade-online", func(ctx context.Context, evt event.Event) error {
		return s.executor.HandleDeviceOnline(ctx, evt)
	})
	if err != nil {
		s.logger.Warn("subscribe device online", zap.Error(err))
	}

	// SetParameterValues response — 覆盖两条 SPV 业务路径：
	//   · LogCollect（FAULT_LOG_COLLECT）：fault → 立即 fail，success → 等文件落地
	//   · Rollback（VERSION_ROLLBACK）：fault → 立即 fail，success → 等 reboot_complete
	// 详见 executor.HandleSetParamsResponse 注释。
	_, err = eventBus.QueueSubscribe(event.SubjectCommandSetParamsResponse, "software-setparams-resp", func(ctx context.Context, evt event.Event) error {
		return s.executor.HandleSetParamsResponse(ctx, evt)
	})
	if err != nil {
		s.logger.Warn("subscribe set_parameters response", zap.Error(err))
	}

	// GetParameterValues response — 仅服务 VERSION_ROLLBACK 4G 阶段一 ROLLBACK_ENABLE 校验：
	// 按 command_key 前缀 "rollback-enable-check-" 过滤，其它 GPV 响应（device_parameters
	// 自动同步等）由 device-rpc-resp-sub 等其它订阅者各自处理，互不打扰。
	_, err = eventBus.QueueSubscribe(event.SubjectCommandGetParamsResponse, "software-rollback-gpv-resp", func(ctx context.Context, evt event.Event) error {
		return s.executor.HandleGetParamsResponseForRollback(ctx, evt, s.rollbackExec)
	})
	if err != nil {
		s.logger.Warn("subscribe get_parameters response for rollback", zap.Error(err))
	}

	// 文件落地通知不走 NATS 订阅：BACKUP stream 是 WorkQueuePolicy retention，
	// 同 subject 只能 1 个 filter consumer——backup.FilePathRecorder 已独占。
	// 改走 hook：FilePathRecorder.SetFileLandedNotifier(softwareService) 在
	// cmd/app/provider/modules.go 装配，FilePathRecorder 处理完 metadata 后
	// 直接同进程回调 SoftwareService.OnLogFileLanded，按 (deviceSN, taskID) 推进
	// LogCollect sub_task 到 Completed。

	return nil
}

// StartTaskReaper starts a background goroutine that periodically scans for
// stale (timed-out) upgrade sub-tasks and marks them as failed.
func (s *SoftwareService) StartTaskReaper() {
	interval := 2 * time.Minute
	// DeviceOnline: 设备 inform_interval 默认 300s（5 min），offline detector
	// 留 2× 缓冲（10 min）才标 offline。
	//   · 历史：原 10 min 在 2026-05-20 改 60 min（避免 inform 延迟一次就错杀）；
	//   · 现在改回 10 min —— 运维诉求"挂起一小时太久不可接受"，且设备实际 inform
	//     周期普遍 60s（远低于 inform_interval=300s 配置默认值），10 min 足够容错 5+
	//     个心跳周期，再上不来就 failed 让用户重试更直观。
	//   · executor 端 Redis wait key TTL 同步保持 10 min，避免 Redis 已过期不会
	//     自动唤醒、reaper 又没兜底失败的中间态。
	timeouts := StaleTimeouts{
		RPCResponse:      5 * time.Minute,
		DeviceOnline:     10 * time.Minute,
		TransferComplete: 30 * time.Minute,
		// FAULT_LOG_COLLECT 走 SPV → CPE 主动 PUT 故障日志，秒～分钟级即可。15min 充裕，
		// 比 30min TC 通用值短一半，配合"文件落地即成功"语义快速失败更直观。
		FaultLogUpload: 15 * time.Minute,
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			taskCounts, err := s.subTaskRepo.FailStale(context.Background(), timeouts)
			if err != nil {
				s.logger.Error("reap stale upgrade tasks", zap.Error(err))
				continue
			}
			var total int64
			for _, cnt := range taskCounts {
				total += cnt
			}
			if total > 0 {
				s.logger.Warn("reaped stale upgrade tasks", zap.Int64("count", total))
				for taskID, cnt := range taskCounts {
					if err := s.subTaskRepo.UpdateFailureReasonByTask(context.Background(), taskID, FailureTaskTimeout); err != nil {
						s.logger.Error("set failure_reason for reaped sub-tasks",
							zap.String("task_id", taskID.String()), zap.Error(err))
					}
					if err := s.taskRepo.IncrementCounts(context.Background(), taskID, 0, int(cnt)); err != nil {
						s.logger.Error("increment fail count for reaped task",
							zap.String("task_id", taskID.String()), zap.Error(err))
					}
					finalizeTask(context.Background(), s.taskRepo, s.logger, taskID)
				}
			}
		}
	}()
	s.logger.Info("upgrade task reaper started",
		zap.Duration("interval", interval),
		zap.Duration("rpc_response_timeout", timeouts.RPCResponse),
		zap.Duration("device_online_timeout", timeouts.DeviceOnline),
		zap.Duration("transfer_complete_timeout", timeouts.TransferComplete),
		zap.Duration("fault_log_upload_timeout", timeouts.FaultLogUpload))
}

// RestorePendingUpgrades recovers upgrade tasks that were in-progress when
// the process crashed.
func (s *SoftwareService) RestorePendingUpgrades(ctx context.Context) {
	s.logger.Info("upgrade restore check completed")
}

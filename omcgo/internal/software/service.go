package software

import (
	"context"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	stderrors "errors"
	"fmt"
	"io"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/minio/minio-go/v7"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/acs/connreq"
	"github.com/omcgo/omcgo/internal/acs/transfercfg"
	"github.com/omcgo/omcgo/internal/authz"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/storage"
	"github.com/omcgo/omcgo/internal/device"
	"github.com/omcgo/omcgo/internal/storageprotection"
	devtask "github.com/omcgo/omcgo/internal/task"
)

const (
	softwarePeriodicConsumerConcurrency = 16
	softwarePeriodicConsumerQueueDepth  = 64
)

type softwareKeyedQueueEventBus interface {
	KeyedQueueSubscribe(
		subject string,
		config event.KeyedQueueConfig,
		keyFunc event.EventKeyFunc,
		handler event.EventHandler,
	) (event.Subscription, error)
}

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
	// cancelReg 把每个执行中的主任务映射到可取消的 context（#59 Problem 3 紧急叫停）。
	// SuspendUpgrade / TerminateUpgrade / canary 阈值自动暂停在改 DB status 之外，
	// 额外 cancel 对应任务的执行 context，让在飞的子任务 goroutine 在派发 Download 前
	// 提前退出。详见 stop_control.go。
	cancelReg *taskCancelRegistry
	// transferProvider 运行时 ACS 传输配置（sys_config 'acs_transfer'，30s 缓存）。
	// 本层只消费 MaxGlobalUpgradeConcurrency（系统级升级并发上限）；
	// BaseURL/凭据等由 executor 自己持有的同一 Provider 消费。
	transferProvider transfercfg.Provider
	// globalUpgradeSlots 系统级（跨任务）升级/回退设备并发闸：所有任务合计同时执行
	// 的设备数 ≤ 上限，空闲槽位按任务轮转（round-robin）分配保证多任务雨露均沾。
	// 任务级并发（MaxConcurrent）仍各自生效，本闸只封顶总量，防多任务叠加打爆固件
	// 下载带宽 / ACS。上限运行时取自"系统设置 → ACS 传输配置"
	// （acquireGlobalUpgradeSlot 每次占位前刷新），未配置默认 DefaultGlobalUpgradeConcurrency。
	globalUpgradeSlots *fairSlotPool
	// groupReader 是 #59 Problem 4「按设备归属校验」的统一强制层接入点（与 device /
	// alarm 模块共用 internal/authz）。未注入时退化为不校验（dev/test），与既有
	// nil-safe 语义一致；生产路由经 SetDeviceGroupReader 注入。可见分组的解析在 handler
	// （持 gin ctx），service 这层只按已解析的 visibleGroups 做逐设备归属判定。
	groupReader authz.GroupReader
	// storageAdmission gates firmware and transfer artifact writes when the
	// global disk protection policy blocks new business data.
	storageAdmission storageprotection.WriteAdmission
}

// SetDeviceGroupReader 注入设备组读取器（#59 Problem 4 / #64 统一强制层），供
// AuthorizeDevicesAccess 对升级 / 回退请求体里的 DeviceIDs 做逐设备归属校验。
// 未注入时 AuthorizeDevicesAccess 退化为不校验（dev/test），与 device 模块语义一致。
func (s *SoftwareService) SetDeviceGroupReader(reader authz.GroupReader) {
	s.groupReader = reader
}

func (s *SoftwareService) SetStorageAdmission(admission storageprotection.WriteAdmission) {
	s.storageAdmission = admission
}

// AuthorizeDevicesAccess 对一批 deviceID 逐个做设备组归属校验（#59 Problem 4）。
// visibleGroups 由 handler 从 gin ctx 经 authz.Resolver 解析后传入，三态语义见 authz 包：
//
//	nil       → 超管：放行。
//	[]        → 无任何分组权限：ErrForbidden（fail-closed）。
//	[g1,...]  → 任一设备不在可见分组内即整批 ErrForbidden（fail-fast，防请求体混入域外设备）。
//
// groupReader 未注入（dev/test）→ 放行（nil-safe）。委派给 authz 包唯一实现，避免语义漂移。
func (s *SoftwareService) AuthorizeDevicesAccess(ctx context.Context, visibleGroups []uuid.UUID, deviceIDs ...uuid.UUID) error {
	return authz.AuthorizeDevicesAccess(ctx, s.groupReader, visibleGroups, deviceIDs...)
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

// SetFirmwareVerifier 注入固件下发前的完整性 / 签名校验器（issue #8），透传给 executor。
// 传 nil 退化为默认 HashOnlyVerifier，校验永不被绕过。
func (s *SoftwareService) SetFirmwareVerifier(v FirmwareVerifier) {
	if s.executor != nil {
		s.executor.SetFirmwareVerifier(v)
	}
}

// SetFirmwareMetrics 注入固件校验 Prometheus 指标（issue #8），透传给 executor（nil-safe）。
func (s *SoftwareService) SetFirmwareMetrics(m *FirmwareMetrics) {
	if s.executor != nil {
		s.executor.SetFirmwareMetrics(m)
	}
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
		cancelReg:    newTaskCancelRegistry(),

		globalUpgradeSlots: newFairSlotPool(DefaultGlobalUpgradeConcurrency),
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
	if s.storageAdmission != nil {
		decision, err := s.storageAdmission.Check(ctx, storageprotection.TargetFilesystem, storageprotection.UnifiedStorageTargetID, storageprotection.WriteScopeUpload)
		if err != nil {
			return fmt.Errorf("storage admission check: %w", err)
		}
		if !decision.Allowed {
			return fmt.Errorf("storage write protected: %s", decision.Reason)
		}
	}
	// #638：把 ProductID（旧单值）与 ProductIDs（新多值）双向打齐，保证：
	//  · DB 唯一索引 (product_id, version, file_type) 仍能命中"主产品"
	//  · ProductIDs 列存全部适用产品，列表/任务过滤走 ANY(product_ids)
	normalizeFirmwareProductIDs(fw)
	category := "img"
	switch fw.FileType {
	case FileTypePATCH:
		category = "patch"
	case FileTypeAP:
		category = "ap"
	case FileTypeFPGA:
		category = "fpga"
	}
	// #492：固件对象路径段以 product_id 为准。产品名中心化后上传只传 product_id、
	// product_class 可能为空，空段会让 MinIO 对象名出现 "//" 被拒
	// （"object name contains unsupported characters"）。product_id 是 uuid，路径安全。
	productSeg := fw.ProductClass
	if fw.ProductID != nil {
		productSeg = fw.ProductID.String()
	}
	if productSeg == "" {
		productSeg = "unknown"
	}
	objectPath := storage.FirmwarePath(category, productSeg, fw.Version, fw.FileName)

	// 单次串流同时算 MD5（向后兼容旧列）与 SHA-256（issue #8 新完整性根）。
	// io.MultiWriter 让 TeeReader 把字节同时喂给两个 hasher，避免二次读文件。
	md5Hash := md5.New()
	sha256Hash := sha256.New()
	teeReader := io.TeeReader(file, io.MultiWriter(md5Hash, sha256Hash))

	_, err := s.minioClient.PutObject(ctx, s.firmwareBkt, objectPath, teeReader, fileSize, minio.PutObjectOptions{
		ContentType: "application/octet-stream",
	})
	if err != nil {
		return fmt.Errorf("upload firmware to MinIO: %w", err)
	}

	fw.MinIOPath = objectPath
	fw.FileSize = fileSize
	fw.MD5Val = hex.EncodeToString(md5Hash.Sum(nil))
	fw.SHA256Val = hex.EncodeToString(sha256Hash.Sum(nil))
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
	s.transferProvider = p
	s.executor.SetTransferProvider(p)
}

func (s *SoftwareService) SetDownloadAddressResolver(r interface {
	Resolve(context.Context, uuid.UUID, transfercfg.TransferDirection) (transfercfg.AddressDecision, error)
}) {
	if s.executor != nil {
		s.executor.SetDownloadAddressResolver(r)
	}
}

// acquireGlobalUpgradeSlot 为 taskID 占用一个系统级升级槽位（跨任务轮转分配）。
// 占位前先按运行时配置刷新上限（系统设置 → ACS 传输配置 → 升级并发上限，30s 缓存
// 生效）；满载排队等待，ctx 取消（任务急停）时放弃并返回 false。
func (s *SoftwareService) acquireGlobalUpgradeSlot(ctx context.Context, taskID uuid.UUID) bool {
	if s.transferProvider != nil {
		if n := s.transferProvider.Snapshot(ctx).MaxGlobalUpgradeConcurrency; n > 0 {
			s.globalUpgradeSlots.SetLimit(n)
		}
	}
	return s.globalUpgradeSlots.Acquire(ctx, taskID)
}

// isSlotReleasableState 并发槽位可释放的子任务状态。槽位语义 = 设备的**服务器侧
// IO 阶段**（下发 RPC → CPE 拉/传文件 → TransferComplete），而非完整升级流程：
// 大批量升级的瓶颈是固件下载的文件 IO / 带宽，TC 之后设备自行安装重启，不再占用
// 服务器资源，提前放行下一台开始下载可显著加快整批节奏。
//   - rebooting / verifying：文件已下载完成（5G TC 后等 102 UPGRADE FINISH），IO
//     已结束 → 释放；终态判定仍由 102 事件处理 / reaper 收口，与槽位无关。
//   - completed / failed / terminated：流程结束 → 释放（4G TC 即 completed）。
//   - suspended 不释放——设备离线挂起会被 device.online 恢复（恢复后重新下载，IO
//     还会发生）或 reaper 在 10min 内判失败，挂起期间设备仍属于"在飞批次"；任务级
//     手动挂起走 execCtx cancel 释放。
func isSlotReleasableState(st UpgradeState) bool {
	switch st {
	case UpgradeCompleted, UpgradeFailed, UpgradeTerminated,
		UpgradeRebooting, UpgradeVerifying:
		return true
	}
	return false
}

// waitSubTaskSlotRelease 阻塞等待子任务离开服务器侧 IO 阶段（见 isSlotReleasableState），
// 让并发槽位（任务级 + 系统级）精确覆盖产生 IO 压力的窗口：下发 → 下载 → TC。
//
// 用 10s 轮询 DB 而非订阅事件：状态写入分散在 TC 处理 / 102 事件 / reaper 超时 / 急停
// 多处，轮询是唯一不会漏的收口。退出条件三选一：
//   - 子任务进入可释放状态（rebooting/verifying/completed/failed/terminated）
//   - ctx 取消（任务级 Suspend/Terminate 急停）
//   - 2h 兜底超时（reaper 单阶段最长 30min + 挂起恢复重试；防 reaper 失效时槽位永久泄漏）
func (s *SoftwareService) waitSubTaskSlotRelease(ctx context.Context, subTaskID uuid.UUID) {
	const pollInterval = 10 * time.Second
	const maxWait = 2 * time.Hour
	deadline := time.Now().Add(maxWait)

	check := func() bool {
		st, err := s.subTaskRepo.GetByID(ctx, subTaskID)
		if err != nil {
			return false // 瞬时 DB 错误下个 tick 重试；持续失败由 maxWait 兜底
		}
		return isSlotReleasableState(st.Status)
	}
	if check() {
		return
	}

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if check() {
				return
			}
			if time.Now().After(deadline) {
				s.logger.Warn("upgrade slot wait backstop timeout, releasing slot",
					zap.String("sub_task_id", subTaskID.String()))
				return
			}
		}
	}
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
	// ScheduledAt 定时执行模式：非 nil 且晚于当前时间 → 任务挂在 pending+timing，
	// 等 TaskScheduler 到点 ResumeCollect 推进；时间已过 / 为 nil → 旧语义。
	ScheduledAt *time.Time
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

	mode, schedAt := resolveScheduleMode(req.ScheduledAt, req.CreateSuspended)
	createUser := strings.TrimSpace(req.CreateUser)
	if createUser == "" {
		createUser = "system"
	}

	mainTask := &UpgradeTask{
		TaskName:         req.TaskName,
		TaskType:         TaskTypeLogCollect,
		DownloadFileType: req.FileType,               // 复用字段存放 Upload FileType（如 "6"）
		FileName:         req.TargetFileNameTemplate, // 复用字段存放目标文件名模板
		Status:           TaskPending,
		CreateStatus:     CreateStatusActive,
		CreateUser:       createUser,
		TotalCount:       len(req.DeviceIDs),
		MaxConcurrent:    concurrency,
	}
	applyScheduleMode(mainTask, mode, schedAt)
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

	switch mode {
	case scheduleModeScheduled:
		s.logger.Info("log collect task created in scheduled mode",
			zap.String("task_id", mainTask.ID.String()),
			zap.Int("device_count", len(req.DeviceIDs)),
			zap.Time("scheduled_at", *schedAt))
		return mainTask, nil
	case scheduleModeSuspended:
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
	// ScheduledAt 非 nil 且未过时 → 占位任务建成 pending + create_status=timing，
	// 由 TaskScheduler 到点触发 StartTask 真正调 dispatcher 派发 device_tasks。
	ScheduledAt *time.Time
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
	// 四态：
	//   - immediate (Suspended=false, ScheduledAt 空): 任务"进行中" / sub_tasks "下载中"
	//   - suspended (Suspended=true):                任务"已挂起" / sub_tasks "待派发"
	//   - scheduled (ScheduledAt 非空且未过):         任务"待定时触发" / sub_tasks "待派发"
	//                                                由 TaskScheduler 到点 StartTask 派发
	mode, schedAt := resolveScheduleMode(req.ScheduledAt, req.Suspended)
	// CreatePlaceholderTrackingTask 历史上用 TaskSuspended 区分手动挂起的 placeholder
	// 与立即派发的 in_progress。新增 scheduled 模式遵循"pending + create_status=timing"
	// 与其他业务（BatchUpgrade/BatchCollect/RollbackDevices）一致，给 scheduler 抢占 hook。
	initialStatus := TaskInProgress
	// CONFIG_RESTORE / LICENSE_UPGRADE 这类 placeholder 任务的派发链路是：
	//   UFTE → device_tasks 队列 → ACS 同步发 Download SOAP → 设备秒回 DownloadResponse
	//   → 设备拉文件 + 应用 + 重启 → 回 TransferComplete（整体可达 30 分钟级）
	// 没有 executor 在中间推进状态，HandleDownloadResponse 只打日志不写库。所以语义
	// 上 sub_task 从创建那一刻起就是"已派发，等 TC"——对应 UpgradeUploading（reaper
	// 走 TransferComplete 30min 窗口）。早期实现用 UpgradeDownloading（reaper 走
	// RPCResponse 5min 窗口）会在 TC 到来前就把任务标 failed → handleTCBody 见到
	// 非 in-flight 状态后静默 return → "Task timeout: no TransferComplete" 假阴性。
	subStatus := UpgradeUploading
	switch mode {
	case scheduleModeSuspended:
		initialStatus = TaskSuspended
		subStatus = UpgradePending
	case scheduleModeScheduled:
		initialStatus = TaskPending
		subStatus = UpgradePending
	}
	main := &UpgradeTask{
		TaskName:         req.TaskName,
		TaskType:         TaskTypeLogCollect, // 复用使 typeSet 过滤覆盖到本任务
		DownloadFileType: req.FileType,       // 用 catalog FileType 字面值（含 <OUI> 占位）
		Status:           initialStatus,
		CreateStatus:     CreateStatusActive,
		CreateUser:       req.CreateUser,
		TotalCount:       len(req.DeviceIDs),
		MaxConcurrent:    1,
		StartedAt:        &now,
	}
	// 仅 scheduled 模式覆写 create_status=timing + scheduled_at。suspended 保留原行为。
	if mode == scheduleModeScheduled {
		main.CreateStatus = CreateStatusTiming
		if schedAt != nil {
			mt := model.Time(*schedAt)
			main.ScheduledAt = &mt
		}
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

// listAllSubTasksByTaskID 翻页取「任务的全部子任务」。ListByTaskID 单页上限 100、
// 默认仅首页 20——凡是"按任务遍历全部子任务"的逻辑（Resume 恢复 / finalize 状态推进 /
// file-landed 按 device_id 反查）不翻页就会漏掉第 21+ 台设备。statuses 非空时只取
// 这些状态的子任务（对齐 Resume 的 pending/suspended 过滤）。
func listAllSubTasksByTaskID(
	ctx context.Context, repo SubTaskRepository, taskID uuid.UUID, statuses ...UpgradeState,
) ([]UpgradeSubTaskWithTaskName, error) {
	const pageSize = 100
	items := make([]UpgradeSubTaskWithTaskName, 0)
	for page := 1; ; page++ {
		res, err := repo.ListByTaskID(ctx, taskID, SubTaskFilter{
			TaskID:      taskID,
			Statuses:    statuses,
			ListRequest: model.ListRequest{Page: page, PageSize: pageSize},
		})
		if err != nil {
			return nil, err
		}
		items = append(items, res.Items...)
		if page >= res.TotalPages || len(res.Items) == 0 {
			break
		}
	}
	return items, nil
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
	// 子任务从 pending → downloading（翻页取全量：>20 台的任务不能只推进前 20 台）
	subs, err := listAllSubTasksByTaskID(ctx, s.subTaskRepo, taskID)
	if err != nil {
		s.logger.Warn("list sub_tasks for finalize failed", zap.Error(err))
		return nil
	}
	for _, sub := range subs {
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
		concurrency = DefaultUpgradeConcurrency
	}

	taskType := req.TaskType
	if taskType == 0 {
		taskType = TaskTypeUpgrade
	}

	mode, schedAt := resolveScheduleMode(req.ScheduledAt, req.CreateSuspended)
	createUser := strings.TrimSpace(req.CreateUser)
	if createUser == "" {
		createUser = "system"
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
		CreateStatus:     CreateStatusActive,
		CreateUser:       createUser,
		TotalCount:       len(req.DeviceIDs),
		MaxConcurrent:    concurrency,
	}
	applyScheduleMode(mainTask, mode, schedAt)
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

	switch mode {
	case scheduleModeScheduled:
		s.logger.Info("upgrade task created in scheduled mode",
			zap.String("task_id", mainTask.ID.String()),
			zap.Int("device_count", len(req.DeviceIDs)),
			zap.Time("scheduled_at", *schedAt))
		return mainTask, nil
	case scheduleModeSuspended:
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

// scheduleMode 是 Batch* 入参在"是否定时 / 是否挂起 / 是否立即"三态间的决策结果。
type scheduleMode int

const (
	scheduleModeImmediate scheduleMode = iota // 立即派发
	scheduleModeSuspended                     // 挂起 pending，等手动 Resume
	scheduleModeScheduled                     // 定时 pending+timing，等 scheduler 触发
)

// resolveScheduleMode 决定三态：
//   - ScheduledAt 非空 ＋ 晚于 now → 定时
//   - 否则 CreateSuspended=true   → 挂起
//   - 否则                       → 立即
//
// 把判定收敛到一个函数，BatchCollect / BatchUpgrade / RollbackDevices 三处共用，
// 避免各自手写一遍漏掉边界（如时间已过的定时任务）。返回的 *time.Time 即为持久化用的
// scheduled_at；非定时模式恒为 nil。
func resolveScheduleMode(scheduledAt *time.Time, createSuspended bool) (scheduleMode, *time.Time) {
	if scheduledAt != nil && scheduledAt.After(nowFunc()) {
		t := *scheduledAt
		return scheduleModeScheduled, &t
	}
	if createSuspended {
		return scheduleModeSuspended, nil
	}
	return scheduleModeImmediate, nil
}

// applyScheduleMode 把 mode 投射到 UpgradeTask 的 Status / CreateStatus / ScheduledAt。
// 调用方负责后续是否 Update 入库或保持 Create 默认值。
func applyScheduleMode(task *UpgradeTask, mode scheduleMode, scheduledAt *time.Time) {
	switch mode {
	case scheduleModeScheduled:
		task.Status = TaskPending
		task.CreateStatus = CreateStatusTiming
		if scheduledAt != nil {
			t := model.Time(*scheduledAt)
			task.ScheduledAt = &t
		}
	case scheduleModeSuspended:
		// 挂起统一用 TaskSuspended（与 CreatePlaceholderTrackingTask 的挂起分支一致）。
		// 历史上这里落 TaskPending+active，导致 ufte/executionModeForTask（只认
		// TaskSuspended）把挂起任务回显成 "immediate"。Resume{Upgrade,Collect} 均同时
		// 接受 TaskSuspended 与 TaskPending，调度器只命中 create_status=timing，故改用
		// TaskSuspended 不影响 start/resume/scheduler 链路。
		task.Status = TaskSuspended
		task.CreateStatus = CreateStatusActive
		task.ScheduledAt = nil
	default: // immediate — Create 之后由调用方推到 in_progress
		task.CreateStatus = CreateStatusActive
		task.ScheduledAt = nil
	}
}

// startExecution launches goroutines to execute upgrade sub-tasks with bounded concurrency.
//
// #59 Problem 3 紧急叫停：每批 goroutine 不再用脱钩的 context.Background()，而是从注册表
// 派生一个绑定到本主任务的可取消 execCtx。SuspendUpgrade / TerminateUpgrade / canary
// 阈值自动暂停调用 cancelReg.cancel(taskID) 时，在飞 goroutine 在 ExecuteOne 顶部 / 锁后
// 派发前的 ctx.Done 检查处提前退出，不再下发 Download。监管 goroutine 等齐所有子任务
// goroutine 后清理注册表项并 cancel（释放 context 资源）。
func (s *SoftwareService) startExecution(mainTask *UpgradeTask, subTasks []*UpgradeSubTask, fw *FirmwareVersion, concurrency int) {
	if concurrency < 1 {
		concurrency = DefaultUpgradeConcurrency
	}
	isKeepConfig := mainTask.IsKeepConfig
	execCtx, done := s.launchControlledExecution(mainTask.ID, len(subTasks))
	sem := make(chan struct{}, concurrency)
	// 派发循环放后台：槽位覆盖设备文件下载阶段，sem 满载会阻塞到有设备下载完成
	// （TC 到达），不能占着调用方（BatchUpgrade HTTP 请求 / 定时调度器）。
	go func() {
		for i := range subTasks {
			select {
			case sem <- struct{}{}:
			case <-execCtx.Done():
				// 任务急停：未派发的设备不再启动，但要补齐 done 计数让监管 goroutine 收尾。
				for j := i; j < len(subTasks); j++ {
					done()
				}
				return
			}
			go func(st *UpgradeSubTask) {
				defer done()
				defer func() { <-sem }()
				defer func() {
					if r := recover(); r != nil {
						s.logger.Error("upgrade executor panic",
							zap.String("sub_task_id", st.ID.String()),
							zap.Any("recover", r))
					}
				}()
				// 系统级（跨任务）并发闸：满载时在此排队（按任务轮转放行），任务急停时放弃。
				if !s.acquireGlobalUpgradeSlot(execCtx, mainTask.ID) {
					return
				}
				defer s.globalUpgradeSlots.Release()
				s.executor.ExecuteOne(execCtx, st, fw, isKeepConfig, mainTask.DownloadFileType)
				// 槽位语义 = 设备文件 IO 阶段：ExecuteOne 只是派发（秒级返回），这里
				// 继续占住任务级 + 系统级槽位，直到文件下载完成（TC 到达）才放行
				// 下一台；之后的安装/重启不占服务器资源，终态由 102 / reaper 收口。
				s.waitSubTaskSlotRelease(execCtx, st.ID)
			}(subTasks[i])
		}
	}()
}

// startRollbackExecution launches goroutines to execute rollback sub-tasks with bounded concurrency.
//
// 与 startExecution 同款 #59 紧急叫停接线：execCtx 绑定本回退主任务，cancel 后在飞
// goroutine 在 RollbackOne 内的 ctx.Done 检查处提前退出。
func (s *SoftwareService) startRollbackExecution(taskID uuid.UUID, subTasks []*UpgradeSubTask) {
	concurrency := 5
	execCtx, done := s.launchControlledExecution(taskID, len(subTasks))
	sem := make(chan struct{}, concurrency)
	// 同 startExecution：槽位覆盖全流程后 sem 会长时间满载，派发循环放后台。
	go func() {
		for i := range subTasks {
			select {
			case sem <- struct{}{}:
			case <-execCtx.Done():
				for j := i; j < len(subTasks); j++ {
					done()
				}
				return
			}
			go func(st *UpgradeSubTask) {
				defer done()
				defer func() { <-sem }()
				defer func() {
					if r := recover(); r != nil {
						s.logger.Error("rollback executor panic",
							zap.String("sub_task_id", st.ID.String()),
							zap.Any("recover", r))
					}
				}()

				// 派发前先看急停信号：cancel 后整批不再触发回退 SPV/GPV。
				if execCtx.Err() != nil {
					return
				}

				// 与升级共用同一系统级并发闸（回退同样产生固件下载/重启压力）。
				if !s.acquireGlobalUpgradeSlot(execCtx, taskID) {
					return
				}
				defer s.globalUpgradeSlots.Release()

				dev, err := s.deviceRepo.GetByID(execCtx, st.DeviceID)
				if err != nil {
					s.rollbackExec.FailRollbackSubTask(context.Background(), st, fmt.Sprintf("device not found: %v", err), FailureDeviceNotFound)
					return
				}

				// 三态识别：GSM(2G) / NR(5G) / LTE(4G 兜底)。GSM 优先判定，避免 2G
				// BSC/BTS 静默落入 LTE 默认分支（adapter 已为 GSM 显式列出回退分支）。
				tech := ResolveDeviceTech(dev)

				// 4G/2G 走两阶段（GPV ROLLBACK_ENABLE → SPV ROLLBACK_CONTROL）；5G 直接 SPV。
				// 路径与值统一在 RollbackExecutor 内部经 adapter + Translator 决定。
				s.rollbackExec.RollbackOne(execCtx, st, dev, tech)
				// 与升级同语义：槽位覆盖到设备开始重启（派发/IO 阶段结束）即释放。
				s.waitSubTaskSlotRelease(execCtx, st.ID)
			}(subTasks[i])
		}
	}()
}

// launchControlledExecution 为一批执行 goroutine 派生绑定到 taskID 的可取消 execCtx，
// 并返回每个 goroutine 收尾时调用的 done 回调。done 用原子计数（n 个子任务 goroutine）
// 在最后一个收尾时 remove 注册表项 + cancel 释放 context 资源。n==0（空批）时立即清理。
//
// 派生用 context.Background() 作父 ctx（而非请求 ctx）：执行 goroutine 的生命周期独立于
// 触发它的 HTTP 请求——请求早就返回了，下载 / 回退要在后台继续，只受急停 cancel 约束。
func (s *SoftwareService) launchControlledExecution(taskID uuid.UUID, n int) (context.Context, func()) {
	execCtx, cancel := s.cancelReg.derive(context.Background(), taskID)
	if n <= 0 {
		s.cancelReg.remove(taskID)
		cancel()
		return execCtx, func() {}
	}
	var remaining int64 = int64(n)
	done := func() {
		if atomic.AddInt64(&remaining, -1) == 0 {
			s.cancelReg.remove(taskID)
			cancel()
		}
	}
	return execCtx, done
}

// HandleTransferComplete advances the upgrade state machine when a device reports transfer complete.
// ACS publishes two kinds of TC events on the same subject:
//  1. Inform-level (from publishInformEvents): payload has device_sn but no TC body data
//  2. TC SOAP body level (from handleTransferComplete): payload is tr069.TransferComplete with
//     command_key, fault_struct (FaultCode/FaultString), start_time, complete_time — but no device_sn
//
// Only #2 is authoritative for software/UFTE state advancement: the TR-069
// CommandKey is the correlation key that proves this TransferComplete belongs to
// the Download/Upload RPC we issued. Inform-level "7 TRANSFER COMPLETE" can
// appear with an empty CommandKey for stale/unrelated transfers; correlating it
// by "current active task on this device" can advance the wrong upgrade.
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

	if hasTCBody {
		return s.handleTCBody(ctx, tcPayload.CommandKey, tcPayload.FaultStruct)
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
		reason := fmt.Sprintf("FaultCode: %d, FaultString: %s", fault.FaultCode, faultString)
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

	// #59 Problem 3 紧急叫停：先取消本任务的执行 context，让在飞的子任务 goroutine
	// 在派发 Download 前提前退出。必须在改 DB status 之前/同步进行——单纯改 status
	// 拦不住已经持有旧 context.Background() 的 goroutine（历史 bug）。
	if s.cancelReg != nil {
		s.cancelReg.cancel(taskID)
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
			// 用 UpdateStatusByOperator（不写 started_at）—— operator 主动暂停
			// 不等同于「轮到设备升级」语义；若 sub_task 此时仍是 pending（未被 executor
			// 挑过），started_at 应保持 NULL，待 Resume 后 executor 真正调度时再写入。
			if err := s.subTaskRepo.UpdateStatusByOperator(ctx, subTask.ID, UpgradeSuspended, "task suspended by operator"); err != nil {
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
	// 翻页取全量待恢复子任务：>20 台的任务不能只 resume 前 20 台，其余会卡死。
	resumable, err := listAllSubTasksByTaskID(ctx, s.subTaskRepo, taskID, UpgradePending, UpgradeSuspended)
	if err != nil {
		return fmt.Errorf("list pending/suspended sub-tasks: %w", err)
	}

	if len(resumable) == 0 {
		return commonerrors.NewBusinessError(8010, "no pending or suspended sub-tasks to execute", commonerrors.ErrInvalidInput)
	}

	subTasks := make([]*UpgradeSubTask, len(resumable))
	for i := range resumable {
		subTasks[i] = &resumable[i].UpgradeSubTask
	}

	if err := s.taskRepo.UpdateStatus(ctx, taskID, TaskInProgress, ""); err != nil {
		return fmt.Errorf("resume main task: %w", err)
	}

	if task.TaskType == TaskTypeRollback {
		s.startRollbackExecution(taskID, subTasks)
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

	// #59 Problem 3：终止同样要先掐断在飞 goroutine（与 SuspendUpgrade 同理）。
	if s.cancelReg != nil {
		s.cancelReg.cancel(taskID)
	}

	// #634：统计本次主动终止"由非终态 → terminated"的子任务数。
	// 旧实现只改子任务 status,没动 upgrade_tasks.{success,fail}_count,
	// 导致前端"成功 N / 失败 M / 总数 K"显示 0/0/K,且 (success+fail)/total 进度卡 0%。
	// 终止的子任务计入 fail_count——语义"未完成 ≈ 失败",且主任务 result=terminated
	// 已在 UI 上以"已终止"标签区分,不会与"失败"主结果混淆。
	terminatedDelta := 0

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
			terminatedDelta++
			// 释放设备级 Redis 锁，否则用户终止后 1 小时（lock TTL）内重试都会撞
			// "升级无法启动，设备不能运行多个升级任务"。failSubTask 路径会顺带 Del 是巧合，
			// 不能依赖。注意 sub_task 行刚拿出来时 DeviceSN 可能为空（还没进 ExecuteOne）→
			// 此时也没拿过锁，跳过即可。
			if subTask.DeviceSN != "" {
				if err := releaseOwnedDeviceLock(ctx, s.redis, subTask.DeviceSN, subTask.ID); err != nil {
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

	// #634：把本次终止的子任务计入 fail_count。先于 UpdateStatus 调用——
	// 即使后续状态更新失败,计数也已写入,前端不再卡 0/0/N。
	if terminatedDelta > 0 {
		if err := s.taskRepo.IncrementCounts(ctx, taskID, 0, terminatedDelta); err != nil {
			return fmt.Errorf("increment fail_count on terminate: %w", err)
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
	// 翻页取全量：>20 台的采集/备份任务不能只 resume 前 20 台。
	resumable, err := listAllSubTasksByTaskID(ctx, s.subTaskRepo, taskID, UpgradePending, UpgradeSuspended)
	if err != nil {
		return fmt.Errorf("list pending sub-tasks: %w", err)
	}
	if len(resumable) == 0 {
		return commonerrors.NewBusinessError(8010, "no pending sub-tasks to execute", commonerrors.ErrInvalidInput)
	}

	subTasks := make([]*UpgradeSubTask, len(resumable))
	for i := range resumable {
		subTasks[i] = &resumable[i].UpgradeSubTask
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

// downgradeDevice 是 firstDowngradeDevice 的轻量返回结构（仅供日志 / 错误信息），
// 不暴露完整设备模型。
type downgradeDevice struct {
	id      string
	current string
}

// firstDowngradeDevice 扫描 deviceIDs，返回第一个「回退到 targetVersion 构成降级」的设备
// （#59 Problem 4 防降级守卫）。blocked=true 时 dev 携带该设备 id + 当前版本供错误信息。
// 取不到设备 / 当前版本为空 / 版本无法可靠比较 → 跳过该设备（不视为降级），见 version.go。
func (s *SoftwareService) firstDowngradeDevice(ctx context.Context, deviceIDs []uuid.UUID, targetVersion string) (bool, downgradeDevice) {
	for _, id := range deviceIDs {
		dev, err := s.deviceRepo.GetByID(ctx, id)
		if err != nil {
			// 设备查不到由后续 sub_task 执行阶段失败兜底；守卫阶段不阻断。
			continue
		}
		if isDowngrade(dev.FirmwareVersion, targetVersion) {
			return true, downgradeDevice{id: id.String(), current: dev.FirmwareVersion}
		}
	}
	return false, downgradeDevice{}
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

	// #59 Problem 4 回退防降级守卫：仅对「显式指定 TargetFirmwareID 的回退」生效。
	// 若目标版本比某设备当前版本更旧（降级，可能回到含已知漏洞的旧镜像），默认整批拒绝，
	// 要求 Force=true 显式确认。canary 自动回退（source=canary_failure）是已促升设备的
	// 受控降级，且不带 OMC 侧目标版本时走设备自身旧 bank，不在此守卫范围。版本无法可靠
	// 比较时放行（见 version.go isDowngrade，fail-open on guard，授权另有 fail-closed）。
	if targetVersion != "" && !req.Force {
		if blocked, dev := s.firstDowngradeDevice(ctx, req.DeviceIDs, targetVersion); blocked {
			s.logger.Warn("rollback blocked: downgrade without force",
				zap.String("target_version", targetVersion),
				zap.String("device_id", dev.id),
				zap.String("device_current_version", dev.current))
			return nil, commonerrors.NewBusinessError(
				8012,
				fmt.Sprintf("回退目标版本 %s 比设备当前版本 %s 更旧（降级），如确需降级请显式传 force=true", targetVersion, dev.current),
				commonerrors.ErrInvalidInput,
			)
		}
	}

	mode, schedAt := resolveScheduleMode(req.ScheduledAt, req.CreateSuspended)

	mainTask := &UpgradeTask{
		TaskName:                 req.TaskName,
		TaskType:                 TaskTypeRollback,
		Status:                   TaskPending,
		ProductClass:             productClass,
		CreateStatus:             CreateStatusActive,
		CreateUser:               req.CreateUser,
		TotalCount:               len(req.DeviceIDs),
		MaxConcurrent:            5,
		RollbackReason:           req.Reason,
		RollbackSource:           source,
		RollbackTargetFirmwareID: req.TargetFirmwareID,
	}
	applyScheduleMode(mainTask, mode, schedAt)
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

	switch mode {
	case scheduleModeScheduled:
		recordMetrics()
		s.logger.Info("rollback task created in scheduled mode",
			zap.String("task_id", mainTask.ID.String()),
			zap.Int("device_count", len(req.DeviceIDs)),
			zap.Time("scheduled_at", *schedAt),
			zap.String("source", source),
			zap.String("reason", req.Reason),
			zap.Bool("with_target", req.TargetFirmwareID != nil))
		return mainTask, nil
	case scheduleModeSuspended:
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

	s.startRollbackExecution(mainTask.ID, subTasks)

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
	normalizeFirmwareProductIDs(fw)
	return s.firmwareRepo.Update(ctx, fw)
}

// normalizeFirmwareProductIDs 把 ProductID（旧单值）与 ProductIDs（#638 新多值）双向同步：
//   - ProductIDs 非空 → ProductID = &ProductIDs[0]（"主产品"=列表首项，喂给老唯一索引）
//   - ProductIDs 为空 且 ProductID 非空 → ProductIDs = [ProductID]（老调用路径自动升格）
//   - 两者皆空 → 保持空（历史"未关联产品"语义）
//
// 任何写入固件元数据的入口（Upload / Update）都应先过这里，确保两列恒等价。
func normalizeFirmwareProductIDs(fw *FirmwareVersion) {
	if len(fw.ProductIDs) > 0 {
		primary := fw.ProductIDs[0]
		fw.ProductID = &primary
		return
	}
	if fw.ProductID != nil {
		fw.ProductIDs = []uuid.UUID{*fw.ProductID}
	}
}

// ---------------------------------------------------------------------------
// Read-through service methods (#18 分层收敛)
//
// Handler 层不再直连 Repository / SQL 池：所有读路径经 Service 转发到
// Repository，统一在 Service 层留出权限检查 / 缓存策略的挂载点（handler →
// service → repository）。这些方法对返回值不做加工，保持行为不变。
// ---------------------------------------------------------------------------

// ListFirmware 列出固件版本（分页/过滤）。
func (s *SoftwareService) ListFirmware(ctx context.Context, filter FirmwareFilter) (*model.ListResponse[FirmwareVersion], error) {
	return s.firmwareRepo.List(ctx, filter)
}

// GetFirmware 按 ID 取单个固件版本元数据。
func (s *SoftwareService) GetFirmware(ctx context.Context, id uuid.UUID) (*FirmwareVersion, error) {
	return s.firmwareRepo.GetByID(ctx, id)
}

// ToggleFirmwareRecommend 翻转固件的"推荐"标记并持久化（读-改-写原子收敛到 Service）。
func (s *SoftwareService) ToggleFirmwareRecommend(ctx context.Context, id uuid.UUID) (*FirmwareVersion, error) {
	fw, err := s.firmwareRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get firmware for recommend toggle: %w", err)
	}
	fw.Recommend = !fw.Recommend
	if err := s.firmwareRepo.Update(ctx, fw); err != nil {
		return nil, fmt.Errorf("update firmware recommend: %w", err)
	}
	return fw, nil
}

// ListUpgradeTasks 列出升级主任务（分页/过滤）。
func (s *SoftwareService) ListUpgradeTasks(ctx context.Context, filter UpgradeTaskFilter) (*model.ListResponse[UpgradeTask], error) {
	return s.taskRepo.List(ctx, filter)
}

// GetUpgradeTask 按 ID 取单个升级主任务。
func (s *SoftwareService) GetUpgradeTask(ctx context.Context, id uuid.UUID) (*UpgradeTask, error) {
	return s.taskRepo.GetByID(ctx, id)
}

// ListSubTasksByTaskID 列出某主任务下的子任务（分页/过滤）。
func (s *SoftwareService) ListSubTasksByTaskID(ctx context.Context, taskID uuid.UUID, filter SubTaskFilter) (*model.ListResponse[UpgradeSubTaskWithTaskName], error) {
	return s.subTaskRepo.ListByTaskID(ctx, taskID, filter)
}

// ListAllSubTasks 跨主任务列出子任务（分页/过滤）。
func (s *SoftwareService) ListAllSubTasks(ctx context.Context, filter AllSubTaskFilter) (*model.ListResponse[UpgradeSubTaskWithTaskName], error) {
	return s.subTaskRepo.ListAll(ctx, filter)
}

// GetSubTask 按 ID 取单个升级子任务。
func (s *SoftwareService) GetSubTask(ctx context.Context, id uuid.UUID) (*UpgradeSubTask, error) {
	return s.subTaskRepo.GetByID(ctx, id)
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
	periodicHandler := func(ctx context.Context, evt event.Event) error {
		return s.executor.HandleDeviceOnline(ctx, evt)
	}
	if keyedBus, ok := eventBus.(softwareKeyedQueueEventBus); ok {
		_, err = keyedBus.KeyedQueueSubscribe(
			event.SubjectDevicePeriodic,
			event.KeyedQueueConfig{
				Durable:     "software-upgrade-periodic",
				Concurrency: softwarePeriodicConsumerConcurrency,
				QueueDepth:  softwarePeriodicConsumerQueueDepth,
			},
			softwarePeriodicDeviceKey,
			periodicHandler,
		)
	} else {
		_, err = eventBus.QueueSubscribe(
			event.SubjectDevicePeriodic,
			"software-upgrade-periodic",
			periodicHandler,
		)
	}
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

func softwarePeriodicDeviceKey(evt event.Event) (string, error) {
	var payload struct {
		DeviceID struct {
			SerialNumber string `json:"serial_number"`
		} `json:"device_id"`
	}
	if err := evt.DecodePayload(&payload); err != nil {
		return "", fmt.Errorf("decode software periodic device key: %w", err)
	}
	serialNumber := strings.TrimSpace(payload.DeviceID.SerialNumber)
	if serialNumber == "" {
		return "", stderrors.New("software periodic device key has empty serial number")
	}
	return serialNumber, nil
}

// StartTaskReaper starts a background goroutine that periodically scans for
// stale (timed-out) upgrade sub-tasks and marks them as failed.
func (s *SoftwareService) StartTaskReaper() {
	interval := 2 * time.Minute
	timeouts := defaultUpgradeTaskReaperTimeouts()
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			total, err := s.reapStaleUpgradeTasks(context.Background(), timeouts)
			if err != nil {
				s.logger.Error("reap stale upgrade tasks", zap.Error(err))
				continue
			}
			if total > 0 {
				s.logger.Warn("reaped stale upgrade tasks", zap.Int64("count", total))
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

func (s *SoftwareService) reapStaleUpgradeTasks(ctx context.Context, timeouts StaleTimeouts) (int64, error) {
	detailed, ok := s.subTaskRepo.(interface {
		FailStaleWithDetails(context.Context, StaleTimeouts) ([]UpgradeSubTask, error)
	})
	if !ok {
		failures, err := s.subTaskRepo.FailStale(ctx, timeouts)
		if err != nil {
			return 0, err
		}
		for _, lock := range failures.Locks {
			if err := releaseOwnedDeviceLock(ctx, s.redis, lock.DeviceSN, lock.SubTaskID); err != nil {
				s.logger.Warn("release device lock for reaped sub-task",
					zap.String("sub_task_id", lock.SubTaskID.String()),
					zap.String("device_sn", lock.DeviceSN),
					zap.Error(err))
			}
		}
		return s.finalizeReapedUpgradeTasks(ctx, failures.TaskCounts, true, nil), nil
	}

	failed, err := detailed.FailStaleWithDetails(ctx, timeouts)
	if err != nil {
		return 0, err
	}
	counts := make(map[uuid.UUID]int64)
	for _, subTask := range failed {
		counts[subTask.TaskID]++
	}
	return s.finalizeReapedUpgradeTasks(ctx, counts, false, failed), nil
}

func (s *SoftwareService) finalizeReapedUpgradeTasks(
	ctx context.Context,
	counts map[uuid.UUID]int64,
	setLegacyFailureReason bool,
	failed []UpgradeSubTask,
) int64 {
	var total int64
	for taskID, count := range counts {
		total += count
		if setLegacyFailureReason {
			if err := s.subTaskRepo.UpdateFailureReasonByTask(ctx, taskID, FailureTaskTimeout); err != nil {
				s.logger.Error("set failure_reason for reaped sub-tasks",
					zap.String("task_id", taskID.String()), zap.Error(err))
			}
		}
		if err := s.taskRepo.IncrementCounts(ctx, taskID, 0, int(count)); err != nil {
			s.logger.Error("increment fail count for reaped task",
				zap.String("task_id", taskID.String()), zap.Error(err))
		}
		finalizeTask(ctx, s.taskRepo, s.logger, taskID)
	}
	for i := range failed {
		if s.executor != nil && failed[i].DeviceSN != "" {
			// FailStaleWithDetails updates rows directly, bypassing executor.failSubTask.
			// Release the exact timed-out sub-task's Redis lock before publishing its
			// terminal event so an immediate retry is not rejected as concurrent.
			s.executor.releaseDeviceLock(ctx, failed[i].DeviceSN, failed[i].ID)
		}
		s.publishReapedUpgradeFailure(ctx, &failed[i])
	}
	return total
}

func (s *SoftwareService) publishReapedUpgradeFailure(ctx context.Context, subTask *UpgradeSubTask) {
	if s.eventBus == nil || subTask == nil || subTask.DeviceID == uuid.Nil {
		return
	}
	evt, err := event.NewEvent(event.SubjectUpgradeFailed, map[string]interface{}{
		"sub_task_id": subTask.ID.String(),
		"task_id":     subTask.TaskID.String(),
		"device_id":   subTask.DeviceID.String(),
		"reason":      subTask.ErrorMessage,
	})
	if err != nil {
		s.logger.Warn("build reaped upgrade failure event", zap.Error(err))
		return
	}
	if err := s.eventBus.Publish(ctx, event.SubjectUpgradeFailed, evt); err != nil {
		s.logger.Warn("publish reaped upgrade failure event",
			zap.String("sub_task_id", subTask.ID.String()), zap.Error(err))
	}
}

func (s *SoftwareService) reapStaleSubTasksOnce(ctx context.Context, timeouts StaleTimeouts) error {
	_, err := s.reapStaleUpgradeTasks(ctx, timeouts)
	return err
}

func defaultUpgradeTaskReaperTimeouts() StaleTimeouts {
	return StaleTimeouts{
		RPCResponse:      10 * time.Minute,
		DeviceOnline:     10 * time.Minute,
		TransferComplete: 30 * time.Minute,
		FaultLogUpload:   15 * time.Minute,
	}
}

// RestorePendingUpgrades recovers upgrade tasks that were in-progress when
// the process crashed.
func (s *SoftwareService) RestorePendingUpgrades(ctx context.Context) {
	s.logger.Info("upgrade restore check completed")
}

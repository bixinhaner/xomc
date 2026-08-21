package software

import (
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
)

// JSONTime converts a time.Time to model.Time (nil-safe).
// Used in scan functions to bridge database time values to the JSON-serializable model.Time.
func JSONTime(t time.Time) model.Time {
	return model.Time(t)
}

// UpgradeState represents the current state of a firmware upgrade sub-task.
type UpgradeState string

const (
	UpgradePending UpgradeState = "pending"
	// UpgradeDownloading 表示"短 RPC 响应窗口"：
	// 普通升级为 Download RPC 等 DownloadResponse；回退阶段一复用该状态等 GPV 响应，
	// 展示层与 reaper 会按 task_type/command_key 改写为回退语义。
	UpgradeDownloading UpgradeState = "downloading"
	// UpgradeUploading 给 Upload RPC（备份 / 日志采集 / 配置恢复）使用——Upload 命令已派发，
	// 涵盖：等 CPE 回 UploadResponse + CPE HTTP PUT 文件到 ACS + 等 CPE 主动发 TransferComplete。
	// 这三段在 TR-069 上没有独立的 ACS 侧信号；UI 在 mapping 层结合 backup_restore_file 是否到位
	// 再把这一段细化为"上传中 / 等待 TransferComplete"两段展示（详见 ufte/model.go normalizeDeviceStatus）。
	UpgradeUploading  UpgradeState = "uploading"
	UpgradeRebooting  UpgradeState = "rebooting"
	UpgradeVerifying  UpgradeState = "verifying"
	UpgradeCompleted  UpgradeState = "completed"
	UpgradeFailed     UpgradeState = "failed"
	UpgradeSuspended  UpgradeState = "suspended"
	UpgradeTerminated UpgradeState = "terminated"
)

// FileType 升级文件类型
type FileType int

const (
	FileTypeIMG   FileType = 0 // 软件主镜像（默认）
	FileTypePATCH FileType = 1 // 补丁包
	FileTypeAP    FileType = 5 // AP 固件
	FileTypeFPGA  FileType = 6 // FPGA
)

// FailureCode identifies why an upgrade sub-task failed.
// Format: {STAGE}_{TYPE} — stage tells which step, type tells device/system/timeout.
type FailureCode string

const (
	// Stage: device check
	FailureDeviceNotFound FailureCode = "DEVICE_NOT_FOUND" // system — device record missing
	FailureDeviceLocked   FailureCode = "DEVICE_LOCKED"    // system — concurrent upgrade conflict
	FailureDeviceOffline  FailureCode = "DEVICE_OFFLINE"   // system — device not active

	// Stage: command push
	FailureCommandPush FailureCode = "COMMAND_PUSH_FAILED" // system — failed to enqueue Download command

	// Stage: integrity / signature verification (issue #8) — runs before Download dispatch
	FailureIntegrityCheck FailureCode = "INTEGRITY_CHECK_FAILED" // system — firmware lacks a usable integrity hash
	FailureSignatureCheck FailureCode = "SIGNATURE_CHECK_FAILED" // system — vendor signature verification failed

	// Stage: download
	FailureDownloadTimeout FailureCode = "DOWNLOAD_TIMEOUT"    // timeout — download progress timed out
	FailureDownloadFile    FailureCode = "DOWNLOAD_FILE_ERROR" // system — firmware file not found
	FailureDownloadFault   FailureCode = "DOWNLOAD_FAULT"      // device — CPE rejected Download RPC
	FailureUploadFault     FailureCode = "UPLOAD_FAULT"        // device — CPE rejected Upload RPC

	// Stage: version rollback
	FailureRollbackEnableCheckTimeout FailureCode = "ROLLBACK_ENABLE_CHECK_TIMEOUT" // timeout — no GPV response for rollback enable check
	FailureRollbackApplyTimeout       FailureCode = "ROLLBACK_APPLY_TIMEOUT"        // timeout — no reboot completion after rollback SPV
	FailureRollbackEnableCheckFault   FailureCode = "ROLLBACK_ENABLE_CHECK_FAULT"   // device — CPE rejected rollback enable GPV
	FailureRollbackSetFault           FailureCode = "ROLLBACK_SET_FAULT"            // device — CPE rejected rollback SPV
	FailureRollbackNotSupported       FailureCode = "ROLLBACK_NOT_SUPPORTED"        // device — rollback enable value is false/empty

	// Stage: transfer complete
	FailureTCFault FailureCode = "TC_FAULT" // device — CPE returned TransferComplete with fault

	// Stage: install/reboot
	Failure5GInstall       FailureCode = "UPGRADE_5G_FAILED" // device — 5G UpgradeStatus = 2 or 3
	FailureVersionMismatch FailureCode = "VERSION_MISMATCH"  // device — reported version differs from target after reboot

	// Stage: general
	FailureTaskTimeout   FailureCode = "TASK_TIMEOUT"       // timeout — reaper marked stale task
	FailureFirmwareGone  FailureCode = "FIRMWARE_NOT_FOUND" // system — firmware record missing
	FailureInternalError FailureCode = "INTERNAL_ERROR"     // system — unexpected error
)

// TaskType 任务类型
type TaskType int

const (
	TaskTypeUpgrade    TaskType = 1  // IMG 升级
	TaskTypeRollback   TaskType = 2  // 版本回退
	TaskTypePatch      TaskType = 4  // PATCH 升级
	TaskTypeFPGA       TaskType = 6  // FPGA 升级
	TaskTypeReserved   TaskType = 8  // 预留
	TaskTypeLogCollect TaskType = 10 // 日志采集 (Upload RPC to ACS upload service)
)

// TaskStatus 主任务状态
type TaskStatus string

const (
	TaskPending    TaskStatus = "pending"
	TaskInProgress TaskStatus = "in_progress"
	TaskSuspended  TaskStatus = "suspended"
	TaskEnded      TaskStatus = "ended"
)

// CreateStatus 标记任务的"创建意图"，与 TaskStatus 正交：
//
//	· CreateStatusActive  — 立即执行 / 挂起待手动 Resume（旧默认）
//	· CreateStatusSuspend — 保留位，未使用
//	· CreateStatusTiming  — 定时执行；调度器到点把 status pending → in_progress
//
// 三态决策表（status × create_status × scheduled_at）：
//
//	┌──────────┬──────────────┬──────────────┬─────────────┐
//	│   status │ create_status│ scheduled_at │  含义       │
//	├──────────┼──────────────┼──────────────┼─────────────┤
//	│ in_progress │ active     │ NULL         │ 立即执行    │
//	│ pending     │ active     │ NULL         │ 挂起        │
//	│ pending     │ timing     │ 非 NULL       │ 定时（待触发）│
//	│ in_progress │ timing     │ 非 NULL       │ 定时已触发  │
//	└──────────┴──────────────┴──────────────┴─────────────┘
//
// 字段值与 migrations/000033 加的 CHECK 约束 ('active','suspend','timing') 一致。
const (
	CreateStatusActive  = "active"
	CreateStatusSuspend = "suspend"
	CreateStatusTiming  = "timing"
)

// TaskResult 主任务结果（ended 时才有值）
type TaskResult string

const (
	TaskResultSuccess    TaskResult = "success"    // 全部成功
	TaskResultPartial    TaskResult = "partial"    // 部分成功
	TaskResultFailed     TaskResult = "failed"     // 全部失败
	TaskResultTerminated TaskResult = "terminated" // 已终止
)

// FirmwareVersion represents a firmware image stored in MinIO.
type FirmwareVersion struct {
	ID uuid.UUID `json:"id"`
	// ProductID #492：固件主产品（products.id），= ProductIDs 列表首项。保留为兼容字段，
	// 旧唯一索引 (product_id, version, file_type) 与现有引用照旧；新逻辑应优先使用 ProductIDs。
	ProductID *uuid.UUID `json:"product_id,omitempty"`
	// ProductIDs #638：固件适用的多个产品 ID 列表（products.id）。上传支持多选；列表/任务
	// 按产品过滤命中条件 :pid = ANY(product_ids)。写入时由 service 同步 ProductID = ProductIDs[0]。
	ProductIDs    []uuid.UUID `json:"product_ids,omitempty"`
	ProductClass  string      `json:"product_class"`
	Version       string      `json:"version"`
	FileName      string      `json:"file_name"`
	FileSize      int64       `json:"file_size"`
	FileType      FileType    `json:"file_type"`
	MinIOPath     string      `json:"minio_path"`
	CompatibleOUI []string    `json:"compatible_oui"`
	MD5Val        string      `json:"md5_val"`
	// SHA256Val 是上传时计算的 SHA-256 完整性摘要（hex）。新上传必填；存量旧行为空，
	// 校验时回退到 MD5Val（见 verifier.go HashOnlyVerifier）。
	SHA256Val string `json:"sha256_val,omitempty"`
	// Signature 是厂商对固件的数字签名（base64）。可空：空 = 无签名固件，验签跳过
	// （向后兼容现网未签名固件）。非空时由 SignatureVerifier 在下发前强制校验。
	Signature string `json:"signature,omitempty"`
	// SignatureAlg 标识 Signature 的签名算法（如 rsa-pss-sha256）。仅 Signature 非空时有意义。
	SignatureAlg string `json:"signature_alg,omitempty"`
	// PublicKeyID 标识验签所用厂商公钥（key id / 指纹），便于密钥轮换与吊销时定位公钥。
	PublicKeyID  string     `json:"public_key_id,omitempty"`
	Recommend    bool       `json:"recommend"`
	Uploader     string     `json:"uploader"`
	Manufacturer string     `json:"manufacturer"`
	ReleaseNotes string     `json:"release_notes"`
	Description  string     `json:"description"`
	Status       string     `json:"status"`
	CreatedAt    model.Time `json:"created_at"`
	UpdatedAt    model.Time `json:"updated_at"`
}

// UpgradeTask represents a main upgrade task (upgrade_tasks table).
// Frontend "任务列表" Tab queries this.
type UpgradeTask struct {
	ID               uuid.UUID   `json:"id"`
	TaskName         string      `json:"task_name"`
	TaskType         TaskType    `json:"task_type"`
	FirmwareID       *uuid.UUID  `json:"firmware_id,omitempty"`
	DownloadFileType string      `json:"download_file_type,omitempty"`
	FileName         string      `json:"file_name,omitempty"`
	FileMD5          string      `json:"file_md5,omitempty"`
	Status           TaskStatus  `json:"status"`
	Result           TaskResult  `json:"result,omitempty"`
	ProductClass     string      `json:"product_class"`
	IsKeepConfig     bool        `json:"is_keep_config"`
	CreateStatus     string      `json:"create_status"`
	CreateUser       string      `json:"create_user"`
	TotalCount       int         `json:"total_count"`
	SuccessCount     int         `json:"success_count"`
	FailCount        int         `json:"fail_count"`
	MaxConcurrent    int         `json:"max_concurrent"`
	StartedAt        *model.Time `json:"started_at,omitempty"`
	EndedAt          *model.Time `json:"ended_at,omitempty"`
	// ScheduledAt 是"定时执行"任务的计划启动时刻。
	//   · 非 nil ＋ CreateStatus=timing ＋ Status=pending → 等待 TaskScheduler 触发
	//   · 触发后由 scheduler 调 ResumeUpgrade / ResumeCollect 推进，本字段保留作审计
	//   · 立即 / 挂起模式恒为 nil
	ScheduledAt *model.Time `json:"scheduled_at,omitempty"`
	CreatedAt   model.Time  `json:"created_at"`
	UpdatedAt   model.Time  `json:"updated_at"`

	// Rollback metadata (T-0021 / R-101). Populated for TaskTypeRollback rows;
	// for upgrade rows the columns carry their DB defaults (rollback_source='manual',
	// reason/target null) and are not surfaced to consumers.
	RollbackReason           string     `json:"rollback_reason,omitempty"`
	RollbackSource           string     `json:"rollback_source,omitempty"`
	RollbackTargetFirmwareID *uuid.UUID `json:"rollback_target_firmware_id,omitempty"`
}

// UpgradeSubTask represents a per-device upgrade sub-task (upgrade_sub_tasks table).
// Frontend "设备列表" Tab queries this via GET /upgrade-tasks/:id/tasks.
type UpgradeSubTask struct {
	ID               uuid.UUID    `json:"id"`
	TaskID           uuid.UUID    `json:"task_id"`
	DeviceID         uuid.UUID    `json:"device_id"`
	FirmwareID       *uuid.UUID   `json:"firmware_id,omitempty"`
	Status           UpgradeState `json:"status"`
	ErrorMessage     string       `json:"error_message,omitempty"`
	RetryCount       int          `json:"retry_count"`
	MaxRetries       int          `json:"max_retries"`
	DeviceSN         string       `json:"device_sn,omitempty"`
	OriVersion       string       `json:"ori_version,omitempty"`
	DestVersion      string       `json:"dest_version,omitempty"`
	CommandKey       string       `json:"command_key,omitempty"`
	FailureReason    string       `json:"failure_reason,omitempty"`
	PreSuspendStatus string       `json:"pre_suspend_status,omitempty"`
	StartedAt        *model.Time  `json:"started_at,omitempty"`
	CompletedAt      *model.Time  `json:"completed_at,omitempty"`
	CreatedAt        model.Time   `json:"created_at"`
	UpdatedAt        model.Time   `json:"updated_at"`
}

// UpgradeSubTaskWithTaskName extends UpgradeSubTask with the parent task's name (via JOIN).
type UpgradeSubTaskWithTaskName struct {
	UpgradeSubTask
	TaskName string `json:"task_name,omitempty"`
}

// FirmwareFilter specifies criteria for listing firmware versions.
type FirmwareFilter struct {
	ProductClass *string `form:"product_class"`
	// ProductID #492：按产品过滤固件（升级抽屉选产品名 → 传 product_id）。用 string 而非
	// *uuid.UUID —— gin 的 query 绑定不支持 uuid（会 400），仓储层再 uuid.Parse。
	ProductID string    `form:"product_id"`
	FileType  *FileType `form:"file_type"`
	model.ListRequest
}

// UpgradeTaskFilter specifies criteria for listing main upgrade tasks.
type UpgradeTaskFilter struct {
	TaskType     *TaskType   `form:"task_type"`
	Status       *TaskStatus `form:"status"`
	ProductClass *string     `form:"product_class"`
	// ProductID #638：按产品名过滤升级任务。任务本身不存 product_id（来自 firmware），
	// repository 用 JOIN firmware_versions 命中 :pid = ANY(fv.product_ids)；用 string 而非
	// *uuid.UUID —— gin query 绑定不支持 uuid，repository 层再 uuid.Parse，解析失败跳过。
	ProductID  string  `form:"product_id"`
	CreateUser *string `form:"create_user"`
	model.ListRequest
}

// SubTaskFilter specifies criteria for listing sub-tasks under a main task.
type SubTaskFilter struct {
	TaskID   uuid.UUID
	Status   *UpgradeState  `form:"status"`
	Statuses []UpgradeState // multi-status filter (takes precedence over Status)
	model.ListRequest
}

// AllSubTaskFilter specifies criteria for listing sub-tasks across all tasks.
type AllSubTaskFilter struct {
	TaskName *string       `form:"task_name"`
	DeviceSN *string       `form:"device_sn"`
	Status   *UpgradeState `form:"status"`
	TaskType *TaskType     `form:"task_type"`
	// TaskID 精确按主任务 ID 收窄。#615 ufte 任务详情 Drawer 走「单任务设备列表」
	// 时下推到 SQL（WHERE ust.task_id = ?），避免内存层全 type 扫描。
	TaskID *uuid.UUID `form:"-"`
	model.ListRequest
}

// DefaultUpgradeConcurrency 升级任务内设备并发执行数的兜底默认值：
// 创建请求未传 / 传 0 时按此值并发派发（前端文件传输中心默认也填 20）。
const DefaultUpgradeConcurrency = 20

// DefaultGlobalUpgradeConcurrency 系统级（跨任务）升级/回退设备并发上限默认值：
// 所有任务合计同时执行的设备数封顶，防多任务叠加打爆固件下载带宽 / ACS。
// 运行时可通过"系统设置 → ACS 传输配置"（sys_config acs_transfer.maxGlobalUpgradeConcurrency）
// 调整，30 秒内生效；未配置时用本默认值。
const DefaultGlobalUpgradeConcurrency = 100

// BatchUpgradeRequest is the JSON body for triggering a batch upgrade.
type BatchUpgradeRequest struct {
	DeviceIDs        []uuid.UUID `json:"device_ids" binding:"required,min=1"`
	FirmwareID       uuid.UUID   `json:"firmware_id" binding:"required"`
	Concurrency      int         `json:"concurrency"`
	TaskName         string      `json:"task_name" binding:"required"`
	TaskType         TaskType    `json:"task_type"`
	DownloadFileType string      `json:"download_file_type,omitempty"`
	// ProductClassHint is used by UFTE when firmware metadata does not carry a
	// concrete product_class. It keeps read-side task classification aligned
	// with the selected task template/product scope.
	ProductClassHint string `json:"product_class_hint,omitempty"`
	IsKeepConfig     bool   `json:"is_keep_config"`
	CreateUser       string `json:"-"`
	CreateSuspended  bool   `json:"create_suspended"`
	// ScheduledAt 指定执行时间。非 nil 且晚于当前时间 → 定时模式（CreateSuspended 被忽略）。
	// 时间已过 / 为 nil → 按 CreateSuspended 走老语义。
	ScheduledAt *time.Time `json:"scheduled_at,omitempty"`

	// Canary strategy (T-0018 / R-101). Strategy defaults to "full" (legacy
	// path); when "canary", CanaryStages drives stage progression. Empty
	// CanaryStages falls back to DefaultCanaryStages [1,10,50,100].
	Strategy           string        `json:"strategy,omitempty"`
	CanaryStages       []CanaryStage `json:"canary_stages,omitempty"`
	AutoAdvance        bool          `json:"auto_advance,omitempty"`
	AutoAdvanceMinutes int           `json:"auto_advance_minutes,omitempty"`

	// RollbackOnFailure (T-0021): when true (and Strategy="canary"), a stage
	// that exceeds its failure threshold will pause AND auto-trigger a
	// rollback for the already-promoted devices. Default false.
	RollbackOnFailure bool `json:"rollback_on_failure,omitempty"`
}

// RollbackSource enumerates why a rollback was triggered (T-0021 / R-101).
// Tracked on upgrade_tasks.rollback_source for audit and metrics.
const (
	RollbackSourceManual        = "manual"
	RollbackSourceCanaryFailure = "canary_failure"
	RollbackSourceCompatibility = "compatibility"
	RollbackSourceScheduled     = "scheduled"
)

// IsValidRollbackSource reports whether s is one of the four canonical sources.
func IsValidRollbackSource(s string) bool {
	switch s {
	case RollbackSourceManual, RollbackSourceCanaryFailure, RollbackSourceCompatibility, RollbackSourceScheduled:
		return true
	}
	return false
}

// RollbackRequest is the JSON body for triggering a batch rollback.
type RollbackRequest struct {
	DeviceIDs       []uuid.UUID `json:"device_ids" binding:"required,min=1"`
	TaskName        string      `json:"task_name" binding:"required"`
	CreateUser      string      `json:"create_user" binding:"required"`
	CreateSuspended bool        `json:"create_suspended"`
	// ScheduledAt 指定回退任务的计划执行时间，语义与 BatchUpgradeRequest 一致。
	ScheduledAt *time.Time `json:"scheduled_at,omitempty"`

	// Audit fields (T-0021). All optional for backwards compatibility.
	// Source defaults to RollbackSourceManual when empty.
	Reason           string     `json:"reason,omitempty"`
	Source           string     `json:"source,omitempty"`
	TargetFirmwareID *uuid.UUID `json:"target_firmware_id,omitempty"`

	// Force 显式允许降级回退（#59 Problem 4 防降级守卫）。默认 false：当回退目标版本
	// 比设备当前版本更旧（降级，可能回到含已知漏洞的旧镜像）时，整批拒绝并要求显式
	// 确认。Force=true 表示运维已知情并批准降级。仅作用于「带 TargetFirmwareID 的显式
	// 目标版本回退」——回退到设备自身旧 bank（不指定 target）天然就是回旧版本，是回退
	// 的本意，不受此守卫约束。
	Force bool `json:"force,omitempty"`
}

// BatchActionRequest is the JSON body for batch actions (suspend/resume/terminate).
type BatchActionRequest struct {
	TaskID uuid.UUID `json:"task_id" binding:"required"`
}

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
	UpgradePending     UpgradeState = "pending"
	UpgradeDownloading UpgradeState = "downloading"
	UpgradeRebooting   UpgradeState = "rebooting"
	UpgradeVerifying   UpgradeState = "verifying"
	UpgradeCompleted   UpgradeState = "completed"
	UpgradeFailed      UpgradeState = "failed"
	UpgradeSuspended   UpgradeState = "suspended"
	UpgradeTerminated  UpgradeState = "terminated"
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

	// Stage: download
	FailureDownloadTimeout FailureCode = "DOWNLOAD_TIMEOUT"    // timeout — download progress timed out
	FailureDownloadFile    FailureCode = "DOWNLOAD_FILE_ERROR" // system — firmware file not found
	FailureDownloadFault   FailureCode = "DOWNLOAD_FAULT"      // device — CPE rejected Download RPC

	// Stage: transfer complete
	FailureTCFault FailureCode = "TC_FAULT" // device — CPE returned TransferComplete with fault

	// Stage: install/reboot
	Failure5GInstall FailureCode = "UPGRADE_5G_FAILED" // device — 5G UpgradeStatus = 2 or 3

	// Stage: general
	FailureTaskTimeout   FailureCode = "TASK_TIMEOUT"       // timeout — reaper marked stale task
	FailureFirmwareGone  FailureCode = "FIRMWARE_NOT_FOUND" // system — firmware record missing
	FailureInternalError FailureCode = "INTERNAL_ERROR"     // system — unexpected error
)

// TaskType 任务类型
type TaskType int

const (
	TaskTypeUpgrade  TaskType = 1 // IMG 升级
	TaskTypeRollback TaskType = 2 // 版本回退
	TaskTypePatch    TaskType = 4 // PATCH 升级
	TaskTypeFPGA     TaskType = 6 // FPGA 升级
	TaskTypeReserved TaskType = 8 // 预留
)

// TaskStatus 主任务状态
type TaskStatus string

const (
	TaskPending    TaskStatus = "pending"
	TaskInProgress TaskStatus = "in_progress"
	TaskSuspended  TaskStatus = "suspended"
	TaskEnded      TaskStatus = "ended"
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
	ID            uuid.UUID  `json:"id"`
	ProductClass  string     `json:"product_class"`
	Version       string     `json:"version"`
	FileName      string     `json:"file_name"`
	FileSize      int64      `json:"file_size"`
	FileType      FileType   `json:"file_type"`
	MinIOPath     string     `json:"minio_path"`
	CompatibleOUI []string   `json:"compatible_oui"`
	MD5Val        string     `json:"md5_val"`
	Recommend     bool       `json:"recommend"`
	Uploader      string     `json:"uploader"`
	Manufacturer  string     `json:"manufacturer"`
	ReleaseNotes  string     `json:"release_notes"`
	Description   string     `json:"description"`
	Status        string     `json:"status"`
	CreatedAt     model.Time `json:"created_at"`
	UpdatedAt     model.Time `json:"updated_at"`
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
	CreatedAt        model.Time  `json:"created_at"`
	UpdatedAt        model.Time  `json:"updated_at"`

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
	ProductClass *string   `form:"product_class"`
	FileType     *FileType `form:"file_type"`
	model.ListRequest
}

// UpgradeTaskFilter specifies criteria for listing main upgrade tasks.
type UpgradeTaskFilter struct {
	TaskType     *TaskType   `form:"task_type"`
	Status       *TaskStatus `form:"status"`
	ProductClass *string     `form:"product_class"`
	CreateUser   *string     `form:"create_user"`
	model.ListRequest
}

// SubTaskFilter specifies criteria for listing sub-tasks under a main task.
type SubTaskFilter struct {
	TaskID uuid.UUID
	Status *UpgradeState `form:"status"`
	model.ListRequest
}

// AllSubTaskFilter specifies criteria for listing sub-tasks across all tasks.
type AllSubTaskFilter struct {
	TaskName *string       `form:"task_name"`
	DeviceSN *string       `form:"device_sn"`
	Status   *UpgradeState `form:"status"`
	TaskType *TaskType     `form:"task_type"`
	model.ListRequest
}

// BatchUpgradeRequest is the JSON body for triggering a batch upgrade.
type BatchUpgradeRequest struct {
	DeviceIDs        []uuid.UUID `json:"device_ids" binding:"required,min=1"`
	FirmwareID       uuid.UUID   `json:"firmware_id" binding:"required"`
	Concurrency      int         `json:"concurrency"`
	TaskName         string      `json:"task_name" binding:"required"`
	TaskType         TaskType    `json:"task_type"`
	DownloadFileType string      `json:"download_file_type,omitempty"`
	IsKeepConfig     bool        `json:"is_keep_config"`
	CreateSuspended  bool        `json:"create_suspended"`

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

	// Audit fields (T-0021). All optional for backwards compatibility.
	// Source defaults to RollbackSourceManual when empty.
	Reason           string     `json:"reason,omitempty"`
	Source           string     `json:"source,omitempty"`
	TargetFirmwareID *uuid.UUID `json:"target_firmware_id,omitempty"`
}

// BatchActionRequest is the JSON body for batch actions (suspend/resume/terminate).
type BatchActionRequest struct {
	TaskID uuid.UUID `json:"task_id" binding:"required"`
}

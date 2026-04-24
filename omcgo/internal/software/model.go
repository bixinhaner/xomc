package software

import (
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
)

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
	FileTypeFPGA  FileType = 6 // FPGA
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
	ID            uuid.UUID         `json:"id"`
	Carrier       model.CarrierCode `json:"carrier"`
	ProductClass  string            `json:"product_class"`
	Version       string            `json:"version"`
	FileName      string            `json:"file_name"`
	FileSize      int64             `json:"file_size"`
	FileType      FileType          `json:"file_type"`
	MinIOPath     string            `json:"minio_path"`
	CompatibleOUI []string          `json:"compatible_oui"`
	MD5Val        string            `json:"md5_val"`
	Recommend     bool              `json:"recommend"`
	Uploader      string            `json:"uploader"`
	Manufacturer  string            `json:"manufacturer"`
	ReleaseNotes  string            `json:"release_notes"`
	Description   string            `json:"description"`
	Status        string            `json:"status"`
	CreatedAt     time.Time         `json:"created_at"`
	UpdatedAt     time.Time         `json:"updated_at"`
}

// UpgradeTask represents a main upgrade task (upgrade_tasks table).
// Frontend "任务列表" Tab queries this.
type UpgradeTask struct {
	ID           uuid.UUID      `json:"id"`
	TaskName     string         `json:"task_name"`
	TaskType     TaskType       `json:"task_type"`
	FirmwareID   *uuid.UUID     `json:"firmware_id,omitempty"`
	FileName     string         `json:"file_name,omitempty"`
	FileMD5      string         `json:"file_md5,omitempty"`
	Status       TaskStatus     `json:"status"`
	Result       TaskResult     `json:"result,omitempty"`
	OperatorCode model.CarrierCode `json:"operator_code"`
	ProductClass string         `json:"product_class"`
	IsKeepConfig bool           `json:"is_keep_config"`
	CreateStatus string         `json:"create_status"`
	CreateUser   string         `json:"create_user"`
	TotalCount   int            `json:"total_count"`
	SuccessCount int            `json:"success_count"`
	FailCount    int            `json:"fail_count"`
	MaxConcurrent int           `json:"max_concurrent"`
	StartedAt    *time.Time     `json:"started_at,omitempty"`
	EndedAt      *time.Time     `json:"ended_at,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
}

// UpgradeSubTask represents a per-device upgrade sub-task (upgrade_sub_tasks table).
// Frontend "设备列表" Tab queries this via GET /upgrade-tasks/:id/tasks.
type UpgradeSubTask struct {
	ID              uuid.UUID    `json:"id"`
	TaskID          uuid.UUID    `json:"task_id"`
	DeviceID        uuid.UUID    `json:"device_id"`
	FirmwareID      *uuid.UUID   `json:"firmware_id,omitempty"`
	Status          UpgradeState `json:"status"`
	ErrorMessage    string       `json:"error_message,omitempty"`
	RetryCount      int          `json:"retry_count"`
	MaxRetries      int          `json:"max_retries"`
	DeviceSN        string       `json:"device_sn,omitempty"`
	OriVersion      string       `json:"ori_version,omitempty"`
	DestVersion     string       `json:"dest_version,omitempty"`
	CommandKey      string       `json:"command_key,omitempty"`
	FailureReason   string       `json:"failure_reason,omitempty"`
	PreSuspendStatus string      `json:"pre_suspend_status,omitempty"`
	StartedAt       *time.Time   `json:"started_at,omitempty"`
	CompletedAt     *time.Time   `json:"completed_at,omitempty"`
	CreatedAt       time.Time    `json:"created_at"`
	UpdatedAt       time.Time    `json:"updated_at"`
}

// FirmwareFilter specifies criteria for listing firmware versions.
type FirmwareFilter struct {
	Carrier      *model.CarrierCode
	ProductClass *string
	FileType     *FileType
	model.ListRequest
}

// UpgradeTaskFilter specifies criteria for listing main upgrade tasks.
type UpgradeTaskFilter struct {
	TaskType     *TaskType
	Status       *TaskStatus
	OperatorCode *model.CarrierCode
	ProductClass *string
	CreateUser   *string
	model.ListRequest
}

// SubTaskFilter specifies criteria for listing sub-tasks under a main task.
type SubTaskFilter struct {
	TaskID uuid.UUID
	Status *UpgradeState
	model.ListRequest
}

// BatchUpgradeRequest is the JSON body for triggering a batch upgrade.
type BatchUpgradeRequest struct {
	DeviceIDs    []uuid.UUID `json:"device_ids" binding:"required,min=1"`
	FirmwareID   uuid.UUID   `json:"firmware_id" binding:"required"`
	Concurrency  int         `json:"concurrency"`
	TaskName     string      `json:"task_name" binding:"required"`
	TaskType     TaskType    `json:"task_type"`
	IsKeepConfig bool        `json:"is_keep_config"`
}

// RollbackRequest is the JSON body for triggering a batch rollback.
type RollbackRequest struct {
	DeviceIDs    []uuid.UUID       `json:"device_ids" binding:"required,min=1"`
	TaskName     string            `json:"task_name" binding:"required"`
	OperatorCode model.CarrierCode `json:"operator_code" binding:"required"`
	CreateUser   string            `json:"create_user" binding:"required"`
}

// BatchActionRequest is the JSON body for batch actions (suspend/resume/terminate).
type BatchActionRequest struct {
	TaskID uuid.UUID `json:"task_id" binding:"required"`
}

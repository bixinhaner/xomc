package software

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
)

// StaleTimeouts configures per-phase reaper timeouts.
type StaleTimeouts struct {
	RPCResponse     time.Duration // waiting for Upload/Download Response
	DeviceOnline    time.Duration // waiting for offline device to come back
	TransferComplete time.Duration // waiting for TransferComplete after RPC accepted
	// FaultLogUpload 是 FAULT_LOG_COLLECT 专用的 uploading 状态超时（SPV 触发后等
	// 设备主动 PUT 故障日志到 ACS）。语义和 TransferComplete 不同：CPE 收到 SPV 后
	// 应立刻 PUT 一个故障日志文件（一般几 MB～几十 MB，过程秒～分钟级），不像
	// RUNTIME_LOG_COLLECT 走 Upload RPC 可能上传大日志包要更久。该值仅作用于
	// fault_log_collect_sub_tasks 表；为 0 时退化到 TransferComplete。
	FaultLogUpload time.Duration
}

// FirmwareRepository provides persistence for firmware versions.
type FirmwareRepository interface {
	Create(ctx context.Context, fw *FirmwareVersion) error
	GetByID(ctx context.Context, id uuid.UUID) (*FirmwareVersion, error)
	List(ctx context.Context, filter FirmwareFilter) (*model.ListResponse[FirmwareVersion], error)
	Update(ctx context.Context, fw *FirmwareVersion) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// TaskRepository provides persistence for main upgrade tasks (upgrade_tasks table).
type TaskRepository interface {
	Create(ctx context.Context, task *UpgradeTask) error
	GetByID(ctx context.Context, id uuid.UUID) (*UpgradeTask, error)
	Update(ctx context.Context, task *UpgradeTask) error
	List(ctx context.Context, filter UpgradeTaskFilter) (*model.ListResponse[UpgradeTask], error)
	IncrementCounts(ctx context.Context, taskID uuid.UUID, successDelta, failDelta int) error
	UpdateStatus(ctx context.Context, id uuid.UUID, status TaskStatus, result TaskResult) error
	Delete(ctx context.Context, id uuid.UUID) error

	// Canary fields (T-0018 / R-101). Read/written via dedicated methods so
	// the legacy scanUpgradeTask + Create paths stay byte-identical.
	GetCanaryFields(ctx context.Context, id uuid.UUID) (*CanaryFields, error)
	UpdateCanaryFields(ctx context.Context, id uuid.UUID, fields *CanaryFields) error
	ListActiveCanaryTaskIDs(ctx context.Context) ([]uuid.UUID, error)
}

// SubTaskRepository provides persistence for per-device upgrade sub-tasks (upgrade_sub_tasks table).
type SubTaskRepository interface {
	Create(ctx context.Context, task *UpgradeSubTask) error
	GetByID(ctx context.Context, id uuid.UUID) (*UpgradeSubTask, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status UpgradeState, errorMsg string) error
	UpdateStatusWithCode(ctx context.Context, id uuid.UUID, status UpgradeState, errorMsg string, code FailureCode) error
	Update(ctx context.Context, task *UpgradeSubTask) error
	List(ctx context.Context, filter SubTaskFilter) (*model.ListResponse[UpgradeSubTaskWithTaskName], error)
	ListByTaskID(ctx context.Context, taskID uuid.UUID, filter SubTaskFilter) (*model.ListResponse[UpgradeSubTaskWithTaskName], error)
	ListAll(ctx context.Context, filter AllSubTaskFilter) (*model.ListResponse[UpgradeSubTaskWithTaskName], error)
	GetActiveByDeviceID(ctx context.Context, deviceID uuid.UUID) (*UpgradeSubTask, error)
	GetByCommandKey(ctx context.Context, commandKey string) (*UpgradeSubTask, error)
	BatchCreate(ctx context.Context, tasks []*UpgradeSubTask) error
	DeleteByTaskID(ctx context.Context, taskID uuid.UUID) error
	FailStale(ctx context.Context, cutoffs StaleTimeouts) (map[uuid.UUID]int64, error)
	UpdateFailureReasonByTask(ctx context.Context, taskID uuid.UUID, code FailureCode) error
}

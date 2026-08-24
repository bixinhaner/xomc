// Package repo 提供按业务拆分的文件传输任务 Repository。
//
// 设计稿：docs/design/task-tables-split-by-business-20260521.md
// 迁移：omcgo/migrations/000144_split_task_tables_by_business.sql
//
// 物理表（每业务一对，与 upgrade_tasks/upgrade_sub_tasks schema 一致）：
//
//	config_backup_tasks        / config_backup_sub_tasks
//	config_restore_tasks       / config_restore_sub_tasks
//	runtime_log_collect_tasks  / runtime_log_collect_sub_tasks
//	fault_log_collect_tasks    / fault_log_collect_sub_tasks
//
// 设计要点：
//   - Model 复用 software.UpgradeTask / UpgradeSubTask（字段都一样不重定义）
//   - Repo 实现一份，通过构造函数 tableName 参数适配 4 张表
//   - software 包代码完全不动，独立维护 upgrade_tasks/upgrade_sub_tasks
//   - 跨业务设备唯一锁通过 device_active_tasks 表（DeviceLockRepo）实现
package repo

import (
	"context"

	"github.com/google/uuid"

	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/software"
)

// TaskRepo 抽象 main task 表的持久化能力。
// 接口签名与 software.TaskRepository 一致，但通过 tableName 配置物理表。
type TaskRepo interface {
	Create(ctx context.Context, task *software.UpgradeTask) error
	GetByID(ctx context.Context, id uuid.UUID) (*software.UpgradeTask, error)
	Update(ctx context.Context, task *software.UpgradeTask) error
	List(ctx context.Context, filter software.UpgradeTaskFilter) (*model.ListResponse[software.UpgradeTask], error)
	IncrementCounts(ctx context.Context, taskID uuid.UUID, successDelta, failDelta int) error
	UpdateStatus(ctx context.Context, id uuid.UUID, status software.TaskStatus, result software.TaskResult) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// SubTaskRepo 抽象 sub_task 表的持久化能力。
// 与 software.SubTaskRepository 兼容，新增 ResetStaleByCutoff 用于多业务通用 reaper。
type SubTaskRepo interface {
	Create(ctx context.Context, task *software.UpgradeSubTask) error
	GetByID(ctx context.Context, id uuid.UUID) (*software.UpgradeSubTask, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status software.UpgradeState, errorMsg string) error
	UpdateStatusWithCode(ctx context.Context, id uuid.UUID, status software.UpgradeState, errorMsg string, code software.FailureCode) error
	Update(ctx context.Context, task *software.UpgradeSubTask) error
	List(ctx context.Context, filter software.SubTaskFilter) (*model.ListResponse[software.UpgradeSubTaskWithTaskName], error)
	ListByTaskID(ctx context.Context, taskID uuid.UUID, filter software.SubTaskFilter) (*model.ListResponse[software.UpgradeSubTaskWithTaskName], error)
	GetActiveByDeviceID(ctx context.Context, deviceID uuid.UUID) (*software.UpgradeSubTask, error)
	GetByCommandKey(ctx context.Context, commandKey string) (*software.UpgradeSubTask, error)
	BatchCreate(ctx context.Context, tasks []*software.UpgradeSubTask) error
	DeleteByTaskID(ctx context.Context, taskID uuid.UUID) error
	FailStale(ctx context.Context, cutoffs software.StaleTimeouts) (software.StaleFailures, error)
	UpdateFailureReasonByTask(ctx context.Context, taskID uuid.UUID, code software.FailureCode) error
}

// DeviceLockRepo 维护 device_active_tasks 表 — 跨业务设备唯一锁。
// 任何 sub_task 进入活跃状态（pending → in_progress 之间）前 Acquire；
// 进入终态（completed / failed / terminated）时 ReleaseBySubTask。
type DeviceLockRepo interface {
	// Acquire 抢占设备。已被占用返回 ErrDeviceBusy。
	Acquire(ctx context.Context, lock DeviceLock) error
	// Release 释放设备锁（按设备 ID）。
	Release(ctx context.Context, deviceID uuid.UUID) error
	// ReleaseBySubTask 通过 sub_task_id 释放（终态回写时不一定知道 device_id）。
	ReleaseBySubTask(ctx context.Context, subTaskID uuid.UUID) error
	// GetByDevice 查询设备当前占用情况；未占用返回 nil, nil。
	GetByDevice(ctx context.Context, deviceID uuid.UUID) (*DeviceLock, error)
}

// DeviceLock 对应 device_active_tasks 行。
type DeviceLock struct {
	DeviceID     uuid.UUID
	SubTaskID    uuid.UUID
	BusinessType string // upgrade / rollback / config_backup / config_restore / runtime_log_collect / fault_log_collect
	SubTaskTable string // 物理 sub_task 表名（reaper / FK 反查用）
}

// Business 业务标识枚举。同步 device_active_tasks.business_type CHECK 约束。
const (
	BusinessUpgrade           = "upgrade"
	BusinessRollback          = "rollback"
	BusinessConfigBackup      = "config_backup"
	BusinessConfigRestore     = "config_restore"
	BusinessRuntimeLogCollect = "runtime_log_collect"
	BusinessFaultLogCollect   = "fault_log_collect"
	BusinessImsParamCollect   = "ims_param_collect"
)

// 物理表名常量。Repo 实例化、跨业务 dispatch、device_active_tasks.sub_task_table 都用这套。
const (
	TableConfigBackupTasks         = "config_backup_tasks"
	TableConfigBackupSubTasks      = "config_backup_sub_tasks"
	TableConfigRestoreTasks        = "config_restore_tasks"
	TableConfigRestoreSubTasks     = "config_restore_sub_tasks"
	TableRuntimeLogCollectTasks    = "runtime_log_collect_tasks"
	TableRuntimeLogCollectSubTasks = "runtime_log_collect_sub_tasks"
	TableFaultLogCollectTasks      = "fault_log_collect_tasks"
	TableFaultLogCollectSubTasks   = "fault_log_collect_sub_tasks"
	TableImsParamCollectTasks      = "ims_param_collect_tasks"
	TableImsParamCollectSubTasks   = "ims_param_collect_sub_tasks"
	// 旧表仍保留给 upgrade / rollback 用，列在这里方便 reaper / dispatch 集中引用。
	TableUpgradeTasks    = "upgrade_tasks"
	TableUpgradeSubTasks = "upgrade_sub_tasks"
)

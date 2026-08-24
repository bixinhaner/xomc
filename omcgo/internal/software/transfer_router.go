package software

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"github.com/omcgo/omcgo/internal/core/model"
)

// 本文件提供"按业务把任务写入正确物理表"的最小路由层。
//
// 背景：upgrade_tasks / upgrade_sub_tasks 两张表过去承载 6 类业务，task_type 字段
// 区分。docs/design/task-tables-split-by-business-20260521.md 决定 4 类新业务
// （配置备份 / 配置下发 / 运行日志 / 异常日志）各占一对独立物理表；升级 / 回退
// 继续用旧表。建表迁移已在 migrations/000144_split_task_tables_by_business.sql。
//
// 这里"分流"只做最薄的两件事：
//   1. BasicTaskRepo / BasicSubTaskRepo：把 software.TaskRepository / SubTaskRepository
//      接口里 4 类新业务用得到的方法子集抽出来，让 software.PgTaskRepository 与
//      transfer/repo.PgTaskRepo 都能用同一类型变量装载（Go 鸭子类型，签名一致）。
//   2. TransferRepoRouter：根据上传业务的 FileType 字符串前导数字识别业务并返回
//      对应的（main, sub）repo 二元组。BatchCollect 内部用 router 选 repo 调用，
//      不再硬编码 s.taskRepo / s.subTaskRepo。
//
// 升级 / 回退的 Canary 路径仍调用具体 *PgTaskRepository（含 canary 三方法），
// 不走 router；router 只在 4 类新业务的 BatchCollect 路径生效。

// BasicTaskRepo 是 4 类新业务足够用的 main task 仓储子集。
// software.PgTaskRepository 和 transfer/repo.PgTaskRepo 都自动满足。
type BasicTaskRepo interface {
	Create(ctx context.Context, task *UpgradeTask) error
	GetByID(ctx context.Context, id uuid.UUID) (*UpgradeTask, error)
	Update(ctx context.Context, task *UpgradeTask) error
	UpdateStatus(ctx context.Context, id uuid.UUID, status TaskStatus, result TaskResult) error
	List(ctx context.Context, filter UpgradeTaskFilter) (*model.ListResponse[UpgradeTask], error)
	IncrementCounts(ctx context.Context, taskID uuid.UUID, successDelta, failDelta int) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// BasicSubTaskRepo 是 4 类新业务足够用的 sub_task 仓储子集。
// 跟 SubTaskRepository 接口签名一致 → software.PgSubTaskRepository 和
// transfer/repo.PgSubTaskRepo 都自动满足。
type BasicSubTaskRepo interface {
	Create(ctx context.Context, task *UpgradeSubTask) error
	GetByID(ctx context.Context, id uuid.UUID) (*UpgradeSubTask, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status UpgradeState, errorMsg string) error
	UpdateStatusWithCode(ctx context.Context, id uuid.UUID, status UpgradeState, errorMsg string, code FailureCode) error
	UpdateStatusByOperator(ctx context.Context, id uuid.UUID, status UpgradeState, errorMsg string) error
	Update(ctx context.Context, task *UpgradeSubTask) error
	List(ctx context.Context, filter SubTaskFilter) (*model.ListResponse[UpgradeSubTaskWithTaskName], error)
	ListByTaskID(ctx context.Context, taskID uuid.UUID, filter SubTaskFilter) (*model.ListResponse[UpgradeSubTaskWithTaskName], error)
	ListAll(ctx context.Context, filter AllSubTaskFilter) (*model.ListResponse[UpgradeSubTaskWithTaskName], error)
	GetActiveByDeviceID(ctx context.Context, deviceID uuid.UUID) (*UpgradeSubTask, error)
	GetByCommandKey(ctx context.Context, commandKey string) (*UpgradeSubTask, error)
	BatchCreate(ctx context.Context, tasks []*UpgradeSubTask) error
	DeleteByTaskID(ctx context.Context, taskID uuid.UUID) error
	FailStale(ctx context.Context, cutoffs StaleTimeouts) (StaleFailures, error)
	UpdateFailureReasonByTask(ctx context.Context, taskID uuid.UUID, code FailureCode) error
	UpdateDestVersionByCommandKey(ctx context.Context, commandKey, destVersion string) error
	UpdateDestVersionByID(ctx context.Context, id uuid.UUID, destVersion string) error
}

// TransferRepoSet 一对（main + sub）业务仓储。
type TransferRepoSet struct {
	Task    BasicTaskRepo
	SubTask BasicSubTaskRepo
	// SubTaskTable 是 SubTask repo 操作的物理表名，写入 device_active_tasks.sub_task_table
	// 用，便于 reaper 反查；为空表示走默认（旧表 upgrade_sub_tasks）。
	SubTaskTable string
	// BusinessType 为 device_active_tasks.business_type 取值（upgrade / config_backup ...）；
	// 为空 → "upgrade"（旧表默认）。
	BusinessType string
}

// TransferRepoRouter 把"FileType 字符串"映射到目标业务的 main+sub repo 二元组。
// 升级 / 回退 / 不识别的 FileType 全部 fallback 到 Default（即旧表 upgrade_*）。
type TransferRepoRouter struct {
	Default           TransferRepoSet // 旧表 upgrade_tasks / upgrade_sub_tasks
	ConfigBackup      TransferRepoSet // FileType 前导 10 / 11 / 12（XML / NV）
	ConfigRestore     TransferRepoSet // 暂未启用 — CONFIG_RESTORE Download 路径
	RuntimeLogCollect TransferRepoSet // FileType 前导 4 / 6（Vendor Log File / 运行日志）
	FaultLogCollect   TransferRepoSet // FileType 前导 8（异常日志）
	ImsParamCollect   TransferRepoSet // FileType 字面值 Ims*（核心网参数/日志/License/恢复采集，docs/design/imscore-file-transfer.md）
}

// ForUploadFileType 根据 Upload 业务的 FileType 字符串选 repo。
// fileType 可能是 "10 {OUI} Configuration File" 或纯数字 "8"，仅看前导数字 token。
// IMS 采集类（CWMP FileType 统一 "ImsCore Parameters File"，历史值 "Ims Log
// File" / "Ims License File" / "Ims Recovery File" / "Ims File" 等保留兼容存量
// 任务行），或带子类型尾段）无前导数字，按字面值前缀识别（大小写不敏感）。
// 注意只列采集类字面值 —— 下发占位任务的反查键（"IMS_FILE_DISTRIBUTE"）也以
// IMS 开头，但必须走 Default（旧表），与 CONFIG_RESTORE / LICENSE_UPGRADE 占位
// 行为一致，不能用宽泛的 "IMS" 前缀。不识别 → Default（旧表）兜底。
func (r *TransferRepoRouter) ForUploadFileType(fileType string) TransferRepoSet {
	upper := strings.ToUpper(strings.TrimSpace(fileType))
	for _, prefix := range [...]string{"IMSCORE PARAMETERS FILE", "IMS FILE", "IMS LOG FILE", "IMS LICENSE FILE", "IMS RECOVERY FILE"} {
		if strings.HasPrefix(upper, prefix) {
			if r.ImsParamCollect.Task != nil {
				return r.ImsParamCollect
			}
			break
		}
	}
	switch leadingNumericToken(fileType) {
	case "4", "6":
		if r.RuntimeLogCollect.Task != nil {
			return r.RuntimeLogCollect
		}
	case "8":
		if r.FaultLogCollect.Task != nil {
			return r.FaultLogCollect
		}
	case "10", "11", "12":
		if r.ConfigBackup.Task != nil {
			return r.ConfigBackup
		}
	}
	return r.Default
}

// ForBusiness 直接按业务字符串选 repo。Reaper / RouteByMainID 等场景用。
func (r *TransferRepoRouter) ForBusiness(business string) TransferRepoSet {
	switch business {
	case "config_backup":
		if r.ConfigBackup.Task != nil {
			return r.ConfigBackup
		}
	case "config_restore":
		if r.ConfigRestore.Task != nil {
			return r.ConfigRestore
		}
	case "runtime_log_collect":
		if r.RuntimeLogCollect.Task != nil {
			return r.RuntimeLogCollect
		}
	case "fault_log_collect":
		if r.FaultLogCollect.Task != nil {
			return r.FaultLogCollect
		}
	case "ims_param_collect":
		if r.ImsParamCollect.Task != nil {
			return r.ImsParamCollect
		}
	}
	return r.Default
}

// AllSets 返回当前已配置的所有 repo set（含 Default）。reaper 调用方迭代用。
func (r *TransferRepoRouter) AllSets() []TransferRepoSet {
	out := []TransferRepoSet{r.Default}
	for _, s := range []TransferRepoSet{r.ConfigBackup, r.ConfigRestore, r.RuntimeLogCollect, r.FaultLogCollect, r.ImsParamCollect} {
		if s.Task != nil {
			out = append(out, s)
		}
	}
	return out
}

// leadingNumericToken 取字符串首段连续数字。"10 {OUI}..." → "10"；"8" → "8"；
// 找不到 → 空串。注意：FileType 字符串可能含空格分隔或全数字，逻辑都覆盖。
func leadingNumericToken(s string) string {
	s = strings.TrimSpace(s)
	end := 0
	for end < len(s) && s[end] >= '0' && s[end] <= '9' {
		end++
	}
	return s[:end]
}

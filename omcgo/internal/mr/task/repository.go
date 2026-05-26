package task

import (
	"context"

	"github.com/google/uuid"

	"github.com/omcgo/omcgo/internal/core/model"
)

// Repository 提供 MR 任务持久层抽象。
//
// 设计原则：
//   - 接口小、单一职责（与 internal/mr/IndicatorRepository / MappingRepository
//     一致风格）
//   - 不持有事务，事务边界由 Service 控制
//   - 错误统一返回带上下文的 wrapped error，业务码（如 ErrNotFound）由
//     internal/core/errors 提供
type Repository interface {
	// CreateTask 仅写入任务主记录（无 cell 进度）。
	// 进度行由 scheduler 在 task 开启时从 mr_device_mappings 动态枚举写入。
	// 这种解耦让"任务创建"与"目标 cell 选择"在时间上分离 — 任务创建时
	// mr_device_mappings 可能还在变化（如新设备上线），到 start_time 时才
	// 锁定目标集合最贴合用户意图。
	CreateTask(ctx context.Context, task *Task) error

	// InsertProgressRows 在 scheduler.tickOpen 时批量插入 cell 进度行（status=pending）。
	// targets 为空时是 no-op，不报错。
	InsertProgressRows(ctx context.Context, taskID uuid.UUID, targets []CellTarget) error

	// GetTask 按 task_id 查任务。不再支持运营商隔离（多租户已下线）。
	// 不存在返回 (nil, errors.ErrNotFound) 包装错误。
	GetTask(ctx context.Context, taskID uuid.UUID) (*Task, error)

	// ListTasks 按过滤条件分页查询任务列表。
	ListTasks(ctx context.Context, filter TaskListFilter) (*model.ListResponse[Task], error)

	// UpdateTaskStatus 更新任务状态（含 task_result）。乐观更新，调用方负责判断
	// 当前状态是否允许转换（waitting→on / on→off / *→termination）。
	UpdateTaskStatus(ctx context.Context, taskID uuid.UUID, status TaskStatus, result *string) error

	// DeleteTask 物理删除任务（CASCADE 自动清理 progress）。
	// 调用方需先确认任务处于可删状态（off / termination）。
	DeleteTask(ctx context.Context, taskID uuid.UUID) error

	// ListProgress 分页查询某任务的 cell 进度。
	ListProgress(ctx context.Context, filter ProgressListFilter) (*model.ListResponse[Progress], error)

	// UpdateProgressDispatch 写下发结果（open/close 共用）；
	// open 调用方设 status=openSuccess/openFailure 等；close 调用方设
	// closeSuccess/closeFailure；faultCode 可为 nil。
	UpdateProgressDispatch(ctx context.Context, taskID uuid.UUID, smallCellCode string, status ProgressStatus, faultCode *string) error

	// TouchHeartbeat 设备成功上报 MR 文件时，由 transfer/bridge.go 调用。
	// 将该 cell 的 last_heartbeat 更新为 now、missed_heartbeat 清零、
	// health_status 置为 normal。可能匹配 0 行（cell 不在任何活跃任务中），
	// 返回的 matched 表示是否找到对应任务进度行。
	TouchHeartbeat(ctx context.Context, smallCellCode string) (matched bool, err error)

	// ListActiveTasksByCell 给出某 cell 当前所属的 'on' 状态任务（用于反查
	// UploadPeriod 计算 Redis TTL）。一般只有 1 条，但允许 N 条（多任务并发
	// 场景，取最近 start_time）。
	ListActiveTasksByCell(ctx context.Context, smallCellCode string) ([]Task, error)

	// ListDueWaitingTasks 返回 task_status='waitting' 且 start_time<=now 的任务，
	// 供 scheduler 触发"开启"动作。limit 控制单批最大数量，避免单次扫描过大。
	ListDueWaitingTasks(ctx context.Context, limit int) ([]Task, error)

	// ListDueOnTasks 返回 task_status='on' 且 end_time<=now 的任务，
	// 供 scheduler 触发"关闭"动作。
	ListDueOnTasks(ctx context.Context, limit int) ([]Task, error)

	// IncrementMissedHeartbeat 给某 cell 的 missed_heartbeat +1（原子操作）。
	// 当 missed_heartbeat 达到 threshold 时，同时把 health_status 切到 'abnormal'。
	// 阈值通过 abnormalThreshold 传入，让 repo 一次 UPDATE 完成累加 + 状态判断，
	// 避免 SELECT-then-UPDATE 的竞态。
	//
	// 仅匹配活跃 (task.task_status='on' AND progress.progress_status='openSuccess') 的行。
	// 返回 (matched, newMissed, becameAbnormal, err)：
	//   matched=false → 该 cell 不在任何活跃任务中（scheduler 应跳过）
	//   becameAbnormal=true → 本次累加导致 health 切到 abnormal（指标增量上报点）
	IncrementMissedHeartbeat(ctx context.Context, smallCellCode string, abnormalThreshold int) (matched bool, newMissed int, becameAbnormal bool, err error)
}

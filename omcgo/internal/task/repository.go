package task

import (
	"context"
)

// TaskRepository PostgreSQL 任务仓库接口
// 用于任务持久化和历史查询
type TaskRepository interface {
	// Create 创建任务记录
	Create(ctx context.Context, task *Task) error

	// Update 更新任务记录
	Update(ctx context.Context, task *Task) error

	// GetByID 根据 ID 获取任务
	GetByID(ctx context.Context, id string) (*Task, error)

	// GetByCWMPID 根据 CWMP ID 获取任务
	GetByCWMPID(ctx context.Context, cwmpID string) (*Task, error)

	// GetHistory 获取设备任务历史
	GetHistory(ctx context.Context, deviceSN string, opts *TaskHistoryOptions) ([]*Task, int64, error)

	// GetPendingByDevice 获取设备的所有 pending 状态任务
	GetPendingByDevice(ctx context.Context, deviceSN string) ([]*Task, error)

	// Delete 删除任务
	Delete(ctx context.Context, id string) error

	// BatchCreate 批量创建任务
	BatchCreate(ctx context.Context, tasks []*Task) error

	// CountByStatus 统计各状态任务数量
	CountByStatus(ctx context.Context, deviceSN string) (map[TaskStatus]int64, error)

	// PurgeOldTasks 清理指定时间之前的已完成任务
	PurgeOldTasks(ctx context.Context, before string) (int64, error)
}

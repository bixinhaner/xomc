package task

import (
	"context"
)

// TaskQueue Redis 任务队列接口
// 用于运行时任务管理，提供高性能的入队、出队操作
type TaskQueue interface {
	// Push 推送任务到设备队列
	// 任务按 priority 升序排列（0=最高优先级）
	Push(ctx context.Context, task *Task) error

	// Pop 弹出最高优先级任务（从队列中移除）
	// 返回 nil 表示队列为空
	Pop(ctx context.Context, deviceSN string) (*Task, error)

	// Peek 查看队首任务（不移除）
	Peek(ctx context.Context, deviceSN string) (*Task, error)

	// Len 获取队列长度
	Len(ctx context.Context, deviceSN string) (int64, error)

	// GetByID 根据 ID 获取任务详情
	GetByID(ctx context.Context, taskID string) (*Task, error)

	// GetByCWMPID 根据 CWMP ID 获取任务
	GetByCWMPID(ctx context.Context, cwmpID string) (*Task, error)

	// Update 更新任务
	Update(ctx context.Context, task *Task) error

	// Delete 从队列和详情存储中删除任务
	Delete(ctx context.Context, taskID string) error

	// GetStaleSentTasks 获取 sent 状态超过指定时间的任务
	// 用于任务恢复机制
	GetStaleSentTasks(ctx context.Context, deviceSN string, staleDuration string) ([]*Task, error)

	// SetCWMPIDMapping 设置 CWMP ID 到 Task ID 的映射
	SetCWMPIDMapping(ctx context.Context, cwmpID, taskID string) error

	// DeleteCWMPIDMapping 删除 CWMP ID 映射
	DeleteCWMPIDMapping(ctx context.Context, cwmpID string) error
}

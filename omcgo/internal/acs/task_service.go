package acs

import (
	"context"
	"encoding/json"

	"github.com/omcgo/omcgo/internal/task"
)

// TaskService defines the interface for task management operations
// used by the ACS Handler. This interface allows for testing with mocks.
type TaskService interface {
	// CreateTask creates a new task for a device.
	CreateTask(ctx context.Context, req *task.CreateTaskRequest) (*task.Task, error)

	// PopTask retrieves and removes the next pending task for a device.
	PopTask(ctx context.Context, deviceSN string) (*task.Task, error)

	// MarkTaskSent marks a task as sent to the CPE with the given CWMP ID.
	MarkTaskSent(ctx context.Context, taskID, cwmpID string) error

	// MarkTaskCompleted marks a task as successfully completed with the result.
	MarkTaskCompleted(ctx context.Context, taskID string, result json.RawMessage) error

	// MarkTaskFailed marks a task as failed with error details.
	MarkTaskFailed(ctx context.Context, taskID string, errorCode int, errorMsg string) error

	// MarkTaskFailedWithResult marks a task as failed AND attaches structured result
	// (e.g. per-parameter SetParameterValuesFault detail). result 为空时等价于
	// MarkTaskFailed —— 调用方按需选用。T-0174 引入。
	MarkTaskFailedWithResult(ctx context.Context, taskID string, errorCode int, errorMsg string, result json.RawMessage) error

	// GetTaskByCWMPID retrieves a task by its CWMP ID.
	GetTaskByCWMPID(ctx context.Context, cwmpID string) (*task.Task, error)

	// GetTask retrieves a task by its primary key (task_id).
	// 用于 handleSOAPFault 在 CPE 自生成 cwmp_id 时按 session.LastTaskID fallback 关联。
	GetTask(ctx context.Context, taskID string) (*task.Task, error)

	// GetQueueLength returns the number of pending tasks for a device.
	GetQueueLength(ctx context.Context, deviceSN string) (int64, error)

	// RecoverPendingTasks 把该设备上 status=sent 且 sent_at>5min 的僵死任务
	// 按 CanRetry() 重置 pending 重入队 / 或标记 failed。
	// 调用方：handler.handleInform —— CPE 每次重连即触发兜底（参 docs/消息队列
	// 全流程流转说明书.md §4.3.2）。RestorePendingQueues 只覆盖 pending 状态，
	// sent 状态必须靠本方法在设备重连时恢复。
	RecoverPendingTasks(ctx context.Context, deviceSN string) error
}

// GPVFaultRecoverer extends a durable parameter-sync run after a leaf-path
// 9005 fault and before the original task becomes terminal.
type GPVFaultRecoverer interface {
	// A non-nil replacement with a non-nil error means the durable recovery
	// committed but immediate queue admission failed; the outbox will retry.
	Recover(ctx context.Context, original *task.Task, remaining []string) (*task.Task, error)
}

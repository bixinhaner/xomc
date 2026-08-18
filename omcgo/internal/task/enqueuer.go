package task

import "context"

// Enqueuer is the narrow interface business modules depend on to submit
// device RPC tasks and inspect queue depth. The concrete *TaskService
// satisfies it; tests may substitute a fake with only the methods below.
//
// Keeping the surface intentionally small follows "accept interfaces,
// return structs": modules don't see task lifecycle APIs (MarkTaskSent /
// MarkTaskCompleted / RestorePendingQueues etc) they have no business
// touching.
type Enqueuer interface {
	CreateTask(ctx context.Context, req *CreateTaskRequest) (*Task, error)
	GetQueueLength(ctx context.Context, deviceSN string) (int64, error)
}

// IdempotentEnqueuer creates at most one durable task for a deterministic
// command key. It is used by at-least-once business event consumers.
type IdempotentEnqueuer interface {
	EnsureTaskByCommandKey(ctx context.Context, req *CreateTaskRequest) (*Task, error)
}

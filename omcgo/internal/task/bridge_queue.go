package task

import (
	"context"
	"encoding/json"
	"time"

	"github.com/omcgo/omcgo/internal/acs/cmdqueue"
)

// BridgeQueue adapts the legacy cmdqueue.CommandQueue interface to use
// the new device_tasks table and Redis task queue.
// Modules that previously pushed to the old cmdqueue can use this adapter
// to transparently route commands through device_tasks.
type BridgeQueue struct {
	svc *TaskService
}

// NewBridgeQueue creates a BridgeQueue backed by the given TaskService.
func NewBridgeQueue(svc *TaskService) *BridgeQueue {
	return &BridgeQueue{svc: svc}
}

// Push converts a cmdqueue.Command into a device_task and enqueues it.
func (b *BridgeQueue) Push(ctx context.Context, deviceSN string, cmd *cmdqueue.Command) error {
	if cmd == nil {
		return nil
	}

	params := cmd.Params
	if params == nil {
		params = json.RawMessage("{}")
	}

	req := &CreateTaskRequest{
		DeviceSN:   deviceSN,
		Method:     cmd.Method,
		Params:     params,
		Priority:   cmd.Priority,
		Source:     TaskSourceAPI,
		CommandKey: cmd.CommandKey,
	}

	if cmd.ID != "" {
		req.Description = cmd.ID
	}

	if cmd.ExpiresAt != nil && !cmd.ExpiresAt.IsZero() {
		expiresIn := int(time.Until(*cmd.ExpiresAt).Seconds())
		if expiresIn > 0 {
			req.ExpiresIn = expiresIn
		}
	}

	_, err := b.svc.CreateTask(ctx, req)
	return err
}

// Pop delegates to TaskService.PopTask.
func (b *BridgeQueue) Pop(ctx context.Context, deviceSN string) (*cmdqueue.Command, error) {
	t, err := b.svc.PopTask(ctx, deviceSN)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, nil
	}
	return taskToCommand(t), nil
}

// Peek returns the highest-priority pending task without removing it.
func (b *BridgeQueue) Peek(ctx context.Context, deviceSN string) (*cmdqueue.Command, error) {
	tasks, err := b.svc.GetPendingTasks(ctx, deviceSN, 1)
	if err != nil {
		return nil, err
	}
	if len(tasks) == 0 {
		return nil, nil
	}
	return taskToCommand(tasks[0]), nil
}

// Len returns the queue length for a device.
func (b *BridgeQueue) Len(ctx context.Context, deviceSN string) (int64, error) {
	return b.svc.GetQueueLength(ctx, deviceSN)
}

// Clear removes all pending tasks for a device (not implemented for safety).
func (b *BridgeQueue) Clear(ctx context.Context, deviceSN string) error {
	return nil
}

func taskToCommand(t *Task) *cmdqueue.Command {
	cmd := &cmdqueue.Command{
		ID:         t.ID,
		Method:     t.Method,
		Params:     t.Params,
		Priority:   t.Priority,
		CreatedAt:  t.CreatedAt,
		CommandKey: t.CommandKey,
		CWMPID:     t.CWMPID,
	}
	if t.ExpiresAt != nil {
		cmd.ExpiresAt = t.ExpiresAt
	}
	return cmd
}

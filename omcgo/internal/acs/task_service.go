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

	// GetTaskByCWMPID retrieves a task by its CWMP ID.
	GetTaskByCWMPID(ctx context.Context, cwmpID string) (*task.Task, error)
}

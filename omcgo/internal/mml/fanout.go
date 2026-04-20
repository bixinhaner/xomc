package mml

import (
	"context"
	"encoding/json"
	"fmt"

	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/task"
)

// DeviceTaskCreator creates device_tasks from an MML task.
// Implemented by task.TaskService; defined here to avoid circular dependency.
type DeviceTaskCreator interface {
	BatchCreateTasks(ctx context.Context, reqs []*task.CreateTaskRequest) ([]*task.Task, error)
}

// Fanouter fans out an MML task into individual device_tasks.
type Fanouter struct {
	taskCreator DeviceTaskCreator
	logger      *zap.Logger
}

// NewFanouter creates a new Fanouter.
func NewFanouter(taskCreator DeviceTaskCreator, logger *zap.Logger) *Fanouter {
	return &Fanouter{
		taskCreator: taskCreator,
		logger:      logger.Named("mml-fanout"),
	}
}

// Fanout creates device_tasks for each (command, device) pair in the MML task.
// Only called for immediate execution; scheduled/periodic tasks are fan-outed when started.
func (f *Fanouter) Fanout(ctx context.Context, mmlTask *MMLTask) (int, error) {
	if len(mmlTask.DeviceSNs) == 0 || len(mmlTask.Commands) == 0 {
		return 0, nil
	}

	reqs := buildDeviceTaskRequests(mmlTask)
	if len(reqs) == 0 {
		return 0, nil
	}

	created, err := f.taskCreator.BatchCreateTasks(ctx, reqs)
	if err != nil {
		return 0, fmt.Errorf("fanout mml task %s: %w", mmlTask.ID, err)
	}

	f.logger.Info("mml task fan-out completed",
		zap.String("mml_task_id", mmlTask.ID.String()),
		zap.Int("device_count", len(mmlTask.DeviceSNs)),
		zap.Int("command_count", len(mmlTask.Commands)),
		zap.Int("device_tasks_created", len(created)),
	)

	return len(created), nil
}

// buildDeviceTaskRequests converts an MML task into individual device task creation requests.
func buildDeviceTaskRequests(mmlTask *MMLTask) []*task.CreateTaskRequest {
	var reqs []*task.CreateTaskRequest
	parentID := mmlTask.ID.String()

	for cmdIdx, cmd := range mmlTask.Commands {
		rpcMethod, _ := cmd["rpc_method"].(string)
		if rpcMethod == "" {
			continue
		}

		var params json.RawMessage
		if p, ok := cmd["parameters"]; ok {
			params, _ = json.Marshal(p)
		}
		if params == nil {
			params = json.RawMessage("{}")
		}

		commandCode, _ := cmd["command_code"].(string)
		description := fmt.Sprintf("MML %s", commandCode)
		if mmlTask.TaskName != "" {
			description = fmt.Sprintf("MML %s: %s", commandCode, mmlTask.TaskName)
		}

		for devIdx, sn := range mmlTask.DeviceSNs {
			reqs = append(reqs, &task.CreateTaskRequest{
				DeviceSN:    sn,
				Method:      rpcMethod,
				Params:      params,
				Priority:    10,
				Source:      task.TaskSourceMML,
				CreatorID:   mmlTask.Creator,
				Description: description,

				ParentTaskID: parentID,
				CommandIndex: cmdIdx,
				DeviceIndex:  devIdx,
			})
		}
	}

	return reqs
}

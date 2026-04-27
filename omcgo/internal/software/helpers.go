package software

import (
	"context"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// finalizeTask checks if all sub-tasks are terminal and finalizes the main task.
// Shared by UpgradeExecutor, RollbackExecutor, and StartTaskReaper.
func finalizeTask(ctx context.Context, taskRepo TaskRepository, logger *zap.Logger, taskID uuid.UUID) {
	task, err := taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return
	}
	if task.SuccessCount+task.FailCount < task.TotalCount {
		return
	}
	result := TaskResultSuccess
	if task.FailCount == task.TotalCount {
		result = TaskResultFailed
	} else if task.FailCount > 0 {
		result = TaskResultPartial
	}
	if err := taskRepo.UpdateStatus(ctx, taskID, TaskEnded, result); err != nil {
		logger.Error("finalize main task", zap.String("task_id", taskID.String()), zap.Error(err))
	}
}

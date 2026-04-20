package mml

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/task"
)

// ResultAggregator updates MML task statistics when device_tasks complete.
type ResultAggregator struct {
	taskRepo TaskRepository
	hub      SSEPublisher
	logger   *zap.Logger
}

// NewResultAggregator creates a new ResultAggregator.
func NewResultAggregator(taskRepo TaskRepository, hub SSEPublisher, logger *zap.Logger) *ResultAggregator {
	return &ResultAggregator{
		taskRepo: taskRepo,
		hub:      hub,
		logger:   logger.Named("mml-result-aggregator"),
	}
}

// OnTaskCompleted is called when a device_task reaches a terminal state.
func (a *ResultAggregator) OnTaskCompleted(ctx context.Context, dt *task.Task) {
	parentID := dt.ParentTaskID
	if parentID == "" {
		return
	}

	mmlID, err := uuid.Parse(parentID)
	if err != nil {
		a.logger.Error("parse parent_task_id", zap.Error(err))
		return
	}

	successDelta, failedDelta := 0, 0
	switch dt.Status {
	case task.TaskStatusCompleted:
		successDelta = 1
	case task.TaskStatusFailed, task.TaskStatusExpired:
		failedDelta = 1
	default:
		return
	}

	if err := a.taskRepo.IncrementStats(ctx, mmlID, successDelta, failedDelta); err != nil {
		a.logger.Error("increment mml task stats", zap.Error(err))
		return
	}

	a.finalizeIfComplete(ctx, mmlID)
}

// finalizeIfComplete transitions the MML task to completed/failed when all sub-tasks finish.
func (a *ResultAggregator) finalizeIfComplete(ctx context.Context, mmlID uuid.UUID) {
	mmlTask, err := a.taskRepo.GetByID(ctx, mmlID)
	if err != nil || mmlTask == nil {
		return
	}

	total := mmlTask.TotalDevices * len(mmlTask.Commands)
	done := mmlTask.SuccessCount + mmlTask.FailedCount
	if done < total {
		return
	}

	var finalStatus TaskStatus
	var finalResult TaskResult
	if mmlTask.FailedCount == 0 {
		finalStatus = TaskCompleted
		finalResult = ResultSuccess
	} else if mmlTask.SuccessCount == 0 {
		finalStatus = TaskFailed
		finalResult = ResultFailed
	} else {
		finalStatus = TaskCompleted
		finalResult = ResultPartial
	}

	now := time.Now()
	mmlTask.Status = finalStatus
	mmlTask.Result = &finalResult
	mmlTask.FinishedAt = &now
	if err := a.taskRepo.Update(ctx, mmlTask); err != nil {
		a.logger.Error("finalize mml task", zap.Error(err))
		return
	}

	a.logger.Info("mml task finalized",
		zap.String("mml_task_id", mmlID.String()),
		zap.String("status", string(finalStatus)),
		zap.String("result", string(finalResult)),
		zap.Int("success", mmlTask.SuccessCount),
		zap.Int("failed", mmlTask.FailedCount),
	)

	if a.hub != nil && mmlTask.Executor != "" {
		data, _ := json.Marshal(map[string]interface{}{
			"task_id":       mmlTask.ID.String(),
			"status":        string(mmlTask.Status),
			"result":        string(*mmlTask.Result),
			"success_count": mmlTask.SuccessCount,
			"failed_count":  mmlTask.FailedCount,
		})
		a.hub.PublishSimple(mmlTask.Executor, "mml_task_completed", data)
	}
}

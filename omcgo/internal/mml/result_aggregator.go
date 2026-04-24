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
// P1 扩展：当 mml_task.script_id != nil 时，把最终 status 回写到 mml_scripts 的
// last_run_status / last_run_at 字段，供脚本列表/详情页展示"最近一次执行态"。
type ResultAggregator struct {
	taskRepo   TaskRepository
	scriptRepo ScriptRepository
	hub        SSEPublisher
	logger     *zap.Logger
}

// NewResultAggregator creates a new ResultAggregator. scriptRepo 可为 nil——
// 单测或不需要回写 mml_scripts 的场景下传 nil，跳过脚本态更新逻辑。
func NewResultAggregator(
	taskRepo TaskRepository,
	scriptRepo ScriptRepository,
	hub SSEPublisher,
	logger *zap.Logger,
) *ResultAggregator {
	return &ResultAggregator{
		taskRepo:   taskRepo,
		scriptRepo: scriptRepo,
		hub:        hub,
		logger:     logger.Named("mml-result-aggregator"),
	}
}

// OnTaskCompleted is called when a device_task reaches a terminal state.
func (a *ResultAggregator) OnTaskCompleted(ctx context.Context, dt *task.Task) {
	sourceID := dt.SourceID
	if sourceID == "" {
		return
	}

	mmlID, err := uuid.Parse(sourceID)
	if err != nil {
		a.logger.Error("parse source_id", zap.Error(err))
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

	// P1 扩展：若本任务关联脚本（script_id != nil），回写
	// mml_scripts.last_run_status / last_run_at。每次执行详情由 mml_tasks 独立
	// 持有；脚本层只存"最近一次"指针，前端脚本详情页用来快速显示。
	a.updateScriptLastRunIfNeeded(ctx, mmlTask, finalResult, now)

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

// updateScriptLastRunIfNeeded 当 mml_task 来自脚本时，回写 mml_scripts 指针列。
// scriptRepo 为 nil（单测 / 无脚本场景）或 task.ScriptID 为 nil 时 no-op。
func (a *ResultAggregator) updateScriptLastRunIfNeeded(
	ctx context.Context,
	mmlTask *MMLTask,
	result TaskResult,
	at time.Time,
) {
	if a.scriptRepo == nil || mmlTask.ScriptID == nil {
		return
	}
	if err := a.scriptRepo.UpdateLastRun(ctx, *mmlTask.ScriptID, string(result), at); err != nil {
		a.logger.Warn("update mml_script last_run",
			zap.String("script_id", mmlTask.ScriptID.String()),
			zap.String("mml_task_id", mmlTask.ID.String()),
			zap.String("result", string(result)),
			zap.Error(err))
		return
	}
	a.logger.Debug("mml_script last_run updated",
		zap.String("script_id", mmlTask.ScriptID.String()),
		zap.String("result", string(result)))
}

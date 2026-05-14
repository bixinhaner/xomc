package ops

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/task"
)

// ============================================================
// result_aggregator.go — T-0102 umbrella 残债 2 / B′
//
// completion 侧 ACS→ops 桥：当 device_task (source=ops) 进入终态
// (completed/failed/expired) 时：
//   1. 更新 ops_task_executions 行 (status running→终态 + completed_at +
//      duration_ms + response/error_message)
//   2. 发 per-task SSE 事件 (command.completed / command.failed)
//      到 channel "command:<opsTaskID>"，让 /commands/:id/stream 订阅者
//      看到每台设备的最终结果（闭合 T-0102-b/c 打通的入队侧 SSE 回路）
//
// 设计配对：
//   - TaskExecutor.dispatchInlineRPC (T-0102-c) 在入队时 Create 行 status="running"
//   - 本 Aggregator 在终态时 Update 同一行到终态
//
// 接线（modules.go）：
//   - 跨进程：completionRouter.Register(task.TaskSourceOps, opsAggregator)
//   - 单进程 fallback：taskSvc.AddCompletionCallback(opsAggregator)
//
// 与 mml.ResultAggregator 对照：
//   - mml 聚合 mml_task 级 success/failed count 并发 SSE 帧（per-device frame）
//   - ops 不维护 ops_task 级聚合（GWT V1 单命令 N 设备一次性 fan-out，
//     无多 command 串行需求），只更 execution 行 + 发 per-device 终态事件
// ============================================================

// ResultAggregator 实现 task.TaskCompletionCallback for source=ops。
type ResultAggregator struct {
	execRepo TaskExecutionRepository
	sseHub   *SSEHub
	logger   *zap.Logger
}

// NewResultAggregator 构造 ResultAggregator。sseHub 可为 nil（单测/不需推送场景）。
func NewResultAggregator(execRepo TaskExecutionRepository, sseHub *SSEHub, logger *zap.Logger) *ResultAggregator {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &ResultAggregator{
		execRepo: execRepo,
		sseHub:   sseHub,
		logger:   logger.Named("ops-result-aggregator"),
	}
}

// OnTaskCompleted 是 task.TaskCompletionCallback 入口。
//
// 行为：
//   - dt 非 nil 且 dt.Source == TaskSourceOps 才处理（防御性 — CompletionRouter
//     已按 source 过滤，但 AddCompletionCallback 通路是全量分发）
//   - dt.Status ∈ {completed, failed, expired} 才更新；其他视为非终态短路
//   - SSE 推送是 best-effort：sseHub 为 nil / 推送失败不阻塞行更新
//   - 行更新失败仅 log warn 不 panic — 完成事件不能丢，但持久化容许重试
func (a *ResultAggregator) OnTaskCompleted(ctx context.Context, dt *task.Task) {
	if dt == nil || dt.Source != task.TaskSourceOps {
		return
	}
	if dt.SourceID == "" {
		return
	}
	opsTaskID, err := uuid.Parse(dt.SourceID)
	if err != nil {
		a.logger.Warn("parse ops source_id",
			zap.String("source_id", dt.SourceID),
			zap.Error(err))
		return
	}

	execStatus, ok := mapTaskStatusToExecStatus(dt.Status)
	if !ok {
		return // 非终态：pending/running/sent/paused/cancelled — 不更新
	}

	a.updateExecutionRow(ctx, opsTaskID, dt, execStatus)
	a.publishCompletionEvent(opsTaskID, dt, execStatus)
}

// mapTaskStatusToExecStatus 把 device_task 终态映射到 ops_task_executions.status。
// 仅终态返 (status, true)；非终态返 ("", false)，调用方据此短路。
func mapTaskStatusToExecStatus(s task.TaskStatus) (string, bool) {
	switch s {
	case task.TaskStatusCompleted:
		return "completed", true
	case task.TaskStatusFailed:
		return "failed", true
	case task.TaskStatusExpired:
		return "expired", true
	}
	return "", false
}

// updateExecutionRow 查找入队期创建的 ops_task_executions 行并翻转到终态。
// 查找键：(task_id=source_id, device_sn) — dispatchInlineRPC 每设备 1 行
// step_index=0 step_type="rpc"，足以唯一定位。
//
// 持久化失败仅 log warn 不向上传播（best-effort + 后续 sweep 可补偿）。
func (a *ResultAggregator) updateExecutionRow(
	ctx context.Context,
	opsTaskID uuid.UUID,
	dt *task.Task,
	execStatus string,
) {
	listResp, err := a.execRepo.List(ctx, TaskExecutionFilter{
		TaskID:      &opsTaskID,
		DeviceSN:    dt.DeviceSN,
		ListRequest: model.ListRequest{Page: 1, PageSize: 1},
	})
	if err != nil {
		a.logger.Warn("list ops_task_executions for completion",
			zap.String("ops_task_id", opsTaskID.String()),
			zap.String("device_sn", dt.DeviceSN),
			zap.Error(err))
		return
	}
	if listResp == nil || len(listResp.Items) == 0 {
		// 入队期 Create 行失败留下的孤儿 device_task，或并行 sweep 已删除。
		// 不阻塞 SSE 推送，但日志留痕便于排查。
		a.logger.Warn("ops_task_execution row not found at completion",
			zap.String("ops_task_id", opsTaskID.String()),
			zap.String("device_sn", dt.DeviceSN))
		return
	}

	row := listResp.Items[0]
	row.Status = execStatus
	if dt.CompletedAt != nil {
		row.CompletedAt = dt.CompletedAt
	} else {
		now := time.Now()
		row.CompletedAt = &now
	}
	if row.StartedAt != nil && row.CompletedAt != nil {
		row.DurationMS = int(row.CompletedAt.Sub(*row.StartedAt).Milliseconds())
	}
	if len(dt.Result) > 0 {
		row.Response = dt.Result
	}
	if dt.ErrorMessage != "" {
		row.ErrorMessage = dt.ErrorMessage
	}

	if err := a.execRepo.Update(ctx, &row); err != nil {
		a.logger.Warn("update ops_task_execution terminal",
			zap.String("execution_id", row.ID.String()),
			zap.String("ops_task_id", opsTaskID.String()),
			zap.String("device_sn", dt.DeviceSN),
			zap.String("target_status", execStatus),
			zap.Error(err))
	}
}

// publishCompletionEvent 发 per-task SSE 完成事件。
//
// channel: "command:<opsTaskID>" — 与 publishDispatchEvent 同 channel 复用，
// 让单次 /commands/:id/stream 订阅可收齐"派发 → 各设备终态"完整生命周期。
//
// eventName:
//   - status=completed → "command.completed"
//   - status=failed/expired → "command.failed"（前端不需区分细分原因，
//     error_message 字段已携带；保持 eventName 简单避免 UI 分支爆炸）
func (a *ResultAggregator) publishCompletionEvent(
	opsTaskID uuid.UUID,
	dt *task.Task,
	execStatus string,
) {
	if a.sseHub == nil {
		return
	}
	eventName := "command.completed"
	if execStatus != "completed" {
		eventName = "command.failed"
	}

	payload := map[string]interface{}{
		"ops_task_id":    opsTaskID.String(),
		"device_sn":      dt.DeviceSN,
		"device_task_id": dt.ID,
		"method":         dt.Method,
		"status":         execStatus,
	}
	if len(dt.Result) > 0 {
		payload["result"] = json.RawMessage(dt.Result)
	}
	if dt.ErrorMessage != "" {
		payload["error_message"] = dt.ErrorMessage
	}
	if dt.CompletedAt != nil {
		payload["completed_at"] = dt.CompletedAt
	}

	data, err := json.Marshal(payload)
	if err != nil {
		a.logger.Warn("marshal ops completion event",
			zap.String("ops_task_id", opsTaskID.String()),
			zap.Error(err))
		return
	}

	a.sseHub.Publish("command:"+opsTaskID.String(), SSEEvent{
		Event: eventName,
		Data:  data,
	})
}

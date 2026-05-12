// Package ops 任务断点续传（T-0101-i）。
//
// PRD §5.3.2 要求：进程重启 / 节点切换后 dispatcher 能从 ops_task_executions
// 恢复 status=running 的任务，避免任务永久停留 running 状态。
//
// 启动期流程（在 cmd/app/main.go / cmd/worker/main.go 服务启动钩子调用）：
//   1. SELECT * FROM ops_tasks WHERE status='running'
//   2. 对每个 task 做恢复决策：
//      - 若上次 dispatcher 真在跑（看 ops_task_executions 最近一行时间戳），重新挂回 dispatcher
//      - 若 dispatcher 已退（节点切换），标记 task 为 failed + 写 audit + 发 alarm
//   3. MVP 实现：保守路线 — 把所有 running task 标记 failed（"上次 dispatcher 异常退出"），
//      future 接节点心跳 + dispatcher state checkpoint 可改恢复
package ops

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// RecoverPendingTasks 启动期扫描并处理上次 dispatcher 异常退出的 running 任务。
//
// MVP 行为：把 status=running 的 task 全部 TransitionStatus(running→failed) +
// 写 OpsAuditLog "task_recovered_as_failed" + 返恢复任务计数。
//
// future T-0101-i 扩展：引入 dispatcher 心跳 + checkpoint，可识别"仍在跑"
// 任务挂回（heartbeat 在 freshness 阈值内）；超阈值视为节点 crash 标 failed。
//
// 注：本方法应在服务启动期单线程调用；并发场景下多 goroutine 调可能重复处理
// 同一 task —— 调用方保证单调用点（cmd/app/main.go 启动钩子）。
func (e *TaskExecutor) RecoverPendingTasks(ctx context.Context) (int, error) {
	filter := TaskFilter{
		Status: statusPtr(OpsTaskRunning),
	}
	// 用大 PageSize 一次性拿（生产典型 < 100 running task；超大量留 future paging）
	filter.Page = 1
	filter.PageSize = 1000

	resp, err := e.taskRepo.List(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("list running tasks for recovery: %w", err)
	}

	recovered := 0
	for _, task := range resp.Items {
		// MVP 保守路线：标记 failed，写 audit + execution
		transErr := e.taskRepo.TransitionStatus(ctx, task.ID,
			[]OpsTaskStatus{OpsTaskRunning},
			OpsTaskFailed, false, true)
		if transErr != nil {
			e.logger.Warn("recovery transition failed",
				zap.String("task_id", task.ID.String()),
				zap.Error(transErr))
			continue
		}

		e.auditSvc.Log(ctx, &OpsAuditLog{
			OpType:       "task_recovered_as_failed",
			TargetType:   "task",
			TargetID:     task.ID.String(),
			OperatorName: "system",
			RiskLevel:    RiskCautious,
			Result:       "recovered",
			OutputSummary: "上次 dispatcher 异常退出，task 标记为 failed",
		})
		e.logger.Info("running task recovered as failed",
			zap.String("task_id", task.ID.String()),
			zap.String("creator", task.Creator),
		)
		recovered++
	}

	if recovered > 0 {
		e.logger.Info("startup recovery completed",
			zap.Int("recovered_count", recovered),
			zap.Int("scanned_total", len(resp.Items)),
		)
	}
	return recovered, nil
}

// statusPtr 辅助：返 OpsTaskStatus 指针（filter struct 需要 *）。
func statusPtr(s OpsTaskStatus) *OpsTaskStatus { return &s }

// 编译期类型断言：避免 unused import 警告（uuid 当前未直接使用但 future 扩展用）
var _ = uuid.Nil

package provider

import (
	"context"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/admin"
	"github.com/omcgo/omcgo/internal/task"
)

// taskLogObserver 把每个到达终态的 device_task 写入 sys_oper_logs 的姊妹表
// sys_task_logs（#122），支撑 /admin/logs/task 视图。
//
// 设计要点：
//   - 它实现 task.TaskCompletionCallback，作为 CompletionRouter 的 source 无关
//     观察者（RegisterObserver）注册一次即覆盖所有 source 的终态任务；单进程
//     部署再额外 AddCompletionCallback 兜底（与业务聚合器同范式）。
//   - fire-and-forget：写失败仅 Warn 不阻塞业务聚合器；用 context.Background +
//     超时，避免上游 ctx 取消后日志丢失。
//   - 仅记录终态（completed/failed/expired/cancelled）。pending/sent 在途态由
//     CompletionRouter 天然过滤（只在终态 Dispatch），此处再防御性跳过。
//
// 这是 provider 装配层的桥（admin.LogRepository ← task.Task），刻意不放进
// internal/task —— 避免 task → admin 反向依赖，与项目其它跨模块适配器同风格。
type taskLogObserver struct {
	logRepo admin.LogRepository
	logger  *zap.Logger
}

// newTaskLogObserver 构造观察者。logRepo / logger 均不能为 nil（调用方保证）。
func newTaskLogObserver(logRepo admin.LogRepository, logger *zap.Logger) *taskLogObserver {
	return &taskLogObserver{
		logRepo: logRepo,
		logger:  logger.Named("task-log-observer"),
	}
}

var _ task.TaskCompletionCallback = (*taskLogObserver)(nil)

// OnTaskCompleted 在任务到达终态时异步写一条 sys_task_logs。
func (o *taskLogObserver) OnTaskCompleted(_ context.Context, t *task.Task) {
	if t == nil || o.logRepo == nil {
		return
	}

	// 防御性：只记录终态。pending/sent 不应到达这里（CompletionRouter 仅终态
	// Dispatch），但保险跳过避免把在途态误记为日志。
	switch t.Status {
	case task.TaskStatusCompleted, task.TaskStatusFailed,
		task.TaskStatusExpired, task.TaskStatusCancelled:
	default:
		return
	}

	operatorID := parseOperatorID(t.CreatorID)
	// PARAM_SYNC currently carries its request correlation UUID in CreatorID;
	// it is not a user/operator foreign key and must never be written as one.
	if t.Source == task.TaskSourceParamSync {
		operatorID = nil
	}
	req := admin.CreateTaskLogRequest{
		TaskType:   t.Method,
		TaskID:     t.ID,
		Status:     string(t.Status),
		OperatorID: operatorID,
		Operator:   t.CreatorID,
		Target:     t.DeviceSN,
		Detail:     t.Description,
		ErrorMsg:   t.ErrorMessage,
		CostMs:     taskCostMs(t),
	}

	// Fire and forget — 任务日志写入不阻塞业务回调链；background ctx + 超时。
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := o.logRepo.CreateTaskLog(ctx, req); err != nil {
			o.logger.Warn("failed to write task log",
				zap.String("task_id", t.ID),
				zap.String("status", string(t.Status)),
				zap.Error(err))
		}
	}()
}

// taskCostMs 计算任务 sent → completed 的耗时（毫秒）。任一时间戳缺失则返回 0。
func taskCostMs(t *task.Task) int {
	if t.SentAt == nil || t.SentAt.IsZero() || t.CompletedAt == nil || t.CompletedAt.IsZero() {
		return 0
	}
	d := t.CompletedAt.Sub(*t.SentAt)
	if d < 0 {
		return 0
	}
	return int(d.Milliseconds())
}

// parseOperatorID 把 Task.CreatorID（字符串）解析为 *uuid.UUID。
// 非 UUID（系统/匿名任务的空串或非标识符）返回 nil —— operator_id 列允许 NULL。
// 全零 UUID（uuid.Nil）同样视为"无操作人"返回 nil：users 表不会存在 id 全 0 的
// 占位记录，一旦有调用方误把它当成"系统任务"标记传进来（曾经
// internal/alarm/sync_service.go 就这么用过），写库会稳定触发
// sys_task_logs_operator_id_fkey 外键违反；在这里兜底比要求每个调用方都记得用
// 空串更可靠。
func parseOperatorID(creatorID string) *uuid.UUID {
	if creatorID == "" {
		return nil
	}
	id, err := uuid.Parse(creatorID)
	if err != nil || id == uuid.Nil {
		return nil
	}
	return &id
}

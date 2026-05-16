package task

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/omcgo/omcgo/internal/core/components/logger"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/tracing"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
)

// DeviceLookup retrieves a device's Connection Request URL by serial number.
type DeviceLookup interface {
	GetConnectionRequestURL(ctx context.Context, deviceSN string) (httpURL string, err error)
}

// ConnectionRequestSender wakes a device via Connection Request.
// The implementation should handle serverAddr, isENB, and other transport-level details internally.
type ConnectionRequestSender interface {
	Send(ctx context.Context, deviceSN, httpURL string) error
}

// TaskCompletionCallback is invoked when a task reaches a terminal state.
// Used by MML to aggregate results back to the parent mml_task.
type TaskCompletionCallback interface {
	OnTaskCompleted(ctx context.Context, task *Task)
}

// TaskService 任务管理服务
// 协调 Redis 队列（运行时）和 PostgreSQL（持久化）
type TaskService struct {
	queue        *RedisTaskQueue
	repo         *PgTaskRepository
	metrics      *TaskMetrics
	deviceLookup DeviceLookup
	connReq      ConnectionRequestSender
	callbacks    []TaskCompletionCallback
	eventBus     event.EventBus
	logger       *zap.Logger
}

// NewTaskService 创建任务服务
func NewTaskService(queue *RedisTaskQueue, repo *PgTaskRepository, log *zap.Logger) *TaskService {
	return &TaskService{
		queue:  queue,
		repo:   repo,
		logger: log,
	}
}

// AggregatePathTranslationMissBySourceID 透传到底层 PgTaskRepository，
// 供 mml.Service 在 GET /mml/tasks/:id 聚合 device_tasks 的 path translation
// miss 元数据（整改方案 Stage 3 — UI 警告标签）。
func (s *TaskService) AggregatePathTranslationMissBySourceID(
	ctx context.Context, sourceID string,
) (PathTranslationMissStats, error) {
	return s.repo.AggregatePathTranslationMissBySourceID(ctx, sourceID)
}

// SetMetrics attaches Prometheus metrics to the service.
func (s *TaskService) SetMetrics(m *TaskMetrics) {
	s.metrics = m
}

// Metrics returns the registered TaskMetrics; other subsystems (CompletionRouter,
// Scheduler) share the same metric instance so "mml_task_total" stays unified.
// Returns nil if SetMetrics hasn't been called (e.g. unit tests).
func (s *TaskService) Metrics() *TaskMetrics {
	return s.metrics
}

// SetConnectionRequester enables automatic device wake-up on task creation.
func (s *TaskService) SetConnectionRequester(dl DeviceLookup, cr ConnectionRequestSender) {
	s.deviceLookup = dl
	s.connReq = cr
}

// AddCompletionCallback registers a callback invoked when tasks reach terminal states.
func (s *TaskService) AddCompletionCallback(cb TaskCompletionCallback) {
	s.callbacks = append(s.callbacks, cb)
}

// SetEventBus enables cross-process task terminal-state broadcasting via NATS.
// ACS writes terminal states then publishes task.completed / task.failed so
// APP/Worker subscribers (e.g. MML ResultAggregator) can react without being
// in the same process.
func (s *TaskService) SetEventBus(bus event.EventBus) {
	s.eventBus = bus
}

// CreateTask 创建新任务
func (s *TaskService) CreateTask(ctx context.Context, req *CreateTaskRequest) (*Task, error) {
	ctx, span := tracing.StartSpan(ctx, tracing.TaskTracerName, "Task CreateTask",
		attribute.String("task.device_sn", req.DeviceSN),
		attribute.String("task.method", req.Method),
	)
	defer span.End()

	task := NewTask(req)

	// 1. 持久化到 PostgreSQL
	if err := s.repo.Create(ctx, task); err != nil {
		return nil, fmt.Errorf("persist task: %w", err)
	}

	// 2. 推送到 Redis 队列
	if err := s.queue.Push(ctx, task); err != nil {
		// 回滚 PostgreSQL 记录
		s.repo.Delete(ctx, task.ID)
		return nil, fmt.Errorf("enqueue task: %w", err)
	}

	if s.metrics != nil {
		s.metrics.PendingTotal.Inc()
	}

	// 3. 异步触发 Connection Request 唤醒设备
	s.wakeDevice(task.DeviceSN)

	logger.L(ctx).Info("task created",
		zap.String("task_id", task.ID),
		zap.String("device_sn", task.DeviceSN),
		zap.String("method", task.Method))

	return task, nil
}

// GetTask 获取任务详情
func (s *TaskService) GetTask(ctx context.Context, taskID string) (*Task, error) {
	// 优先从 Redis 获取（更实时）
	task, err := s.queue.GetByID(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("get task from queue: %w", err)
	}
	if task != nil {
		return task, nil
	}

	// 从 PostgreSQL 获取
	return s.repo.GetByID(ctx, taskID)
}

// GetTaskByCWMPID 根据 CWMP ID 获取任务
func (s *TaskService) GetTaskByCWMPID(ctx context.Context, cwmpID string) (*Task, error) {
	// 从 Redis 获取
	task, err := s.queue.GetByCWMPID(ctx, cwmpID)
	if err != nil {
		return nil, fmt.Errorf("get task by cwmp_id from queue: %w", err)
	}
	if task != nil {
		return task, nil
	}

	// 从 PostgreSQL 获取
	return s.repo.GetByCWMPID(ctx, cwmpID)
}

// GetPendingTasks 获取设备待处理任务
func (s *TaskService) GetPendingTasks(ctx context.Context, deviceSN string, limit int) ([]*Task, error) {
	// 从 PostgreSQL 获取所有 pending 状态任务
	tasks, err := s.repo.GetPendingByDevice(ctx, deviceSN)
	if err != nil {
		return nil, fmt.Errorf("get pending tasks: %w", err)
	}

	if limit > 0 && len(tasks) > limit {
		tasks = tasks[:limit]
	}

	return tasks, nil
}

// PopTask 弹出队首任务（供 ACS Handler 调用）
func (s *TaskService) PopTask(ctx context.Context, deviceSN string) (*Task, error) {
	ctx, span := tracing.StartSpan(ctx, tracing.TaskTracerName, "Task PopTask",
		attribute.String("task.device_sn", deviceSN),
	)
	defer span.End()

	t, err := s.queue.Pop(ctx, deviceSN)
	if err != nil {
		tracing.RecordError(span, err)
	} else if t != nil {
		span.SetAttributes(
			attribute.String("task.id", t.ID),
			attribute.String("task.method", t.Method),
		)
	}
	return t, err
}

// GetQueueLength 获取队列长度
func (s *TaskService) GetQueueLength(ctx context.Context, deviceSN string) (int64, error) {
	return s.queue.Len(ctx, deviceSN)
}

// MarkTaskSent 标记任务已发送
func (s *TaskService) MarkTaskSent(ctx context.Context, taskID, cwmpID string) error {
	// 更新 Redis
	if err := s.queue.MarkTaskSent(ctx, taskID, cwmpID); err != nil {
		return fmt.Errorf("mark task sent in queue: %w", err)
	}

	// 同步更新 PostgreSQL
	task, err := s.queue.GetByID(ctx, taskID)
	if err != nil {
		return fmt.Errorf("get sent task: %w", err)
	}
	if task != nil {
		if err := s.repo.Update(ctx, task); err != nil {
			logger.L(ctx).Error("sync task to db", zap.Error(err), zap.String("task_id", taskID))
		}
	}

	logger.L(ctx).Info("task sent",
		zap.String("task_id", taskID),
		zap.String("cwmp_id", cwmpID))

	return nil
}

// MarkTaskCompleted 标记任务完成
func (s *TaskService) MarkTaskCompleted(ctx context.Context, taskID string, result json.RawMessage) error {
	// 更新 Redis
	if err := s.queue.MarkTaskCompleted(ctx, taskID, result); err != nil {
		return fmt.Errorf("mark task completed in queue: %w", err)
	}

	// 同步更新 PostgreSQL
	task, err := s.queue.GetByID(ctx, taskID)
	if err != nil {
		return fmt.Errorf("get completed task: %w", err)
	}
	if task != nil {
		if err := s.repo.Update(ctx, task); err != nil {
			logger.L(ctx).Error("sync task to db", zap.Error(err), zap.String("task_id", taskID))
		}
	}

	if s.metrics != nil {
		s.metrics.CompletedTotal.WithLabelValues("success").Inc()
		s.metrics.PendingTotal.Dec()
	}

	logger.L(ctx).Info("task completed",
		zap.String("task_id", taskID),
		zap.String("method", task.Method))

	s.notifyCompletion(ctx, task)

	return nil
}

// MarkTaskFailed 标记任务失败
func (s *TaskService) MarkTaskFailed(ctx context.Context, taskID string, errorCode int, errorMsg string) error {
	// 更新 Redis
	if err := s.queue.MarkTaskFailed(ctx, taskID, errorCode, errorMsg); err != nil {
		return fmt.Errorf("mark task failed in queue: %w", err)
	}

	// 同步更新 PostgreSQL
	task, err := s.queue.GetByID(ctx, taskID)
	if err != nil {
		return fmt.Errorf("get failed task: %w", err)
	}
	if task != nil {
		if err := s.repo.Update(ctx, task); err != nil {
			logger.L(ctx).Error("sync task to db", zap.Error(err), zap.String("task_id", taskID))
		}
	}

	if s.metrics != nil {
		s.metrics.CompletedTotal.WithLabelValues("failed").Inc()
		s.metrics.PendingTotal.Dec()
	}

	logger.L(ctx).Info("task failed",
		zap.String("task_id", taskID),
		zap.Int("error_code", errorCode),
		zap.String("error_message", errorMsg))

	s.notifyCompletion(ctx, task)

	return nil
}

// CancelTask 取消任务
func (s *TaskService) CancelTask(ctx context.Context, taskID string) error {
	task, err := s.GetTask(ctx, taskID)
	if err != nil {
		return fmt.Errorf("get task for cancel: %w", err)
	}
	if task == nil {
		return fmt.Errorf("task not found: %s", taskID)
	}

	// 只能取消 pending 状态的任务
	if task.Status != TaskStatusPending {
		return fmt.Errorf("cannot cancel task with status: %s", task.Status)
	}

	// 更新状态
	now := time.Now()
	task.Status = TaskStatusCancelled
	task.CompletedAt = &now

	// 从 Redis 删除
	if err := s.queue.Delete(ctx, taskID); err != nil {
		return fmt.Errorf("delete task from queue: %w", err)
	}

	// 更新 PostgreSQL
	if err := s.repo.Update(ctx, task); err != nil {
		return fmt.Errorf("update cancelled task: %w", err)
	}

	if s.metrics != nil {
		s.metrics.CompletedTotal.WithLabelValues("expired").Inc()
		s.metrics.PendingTotal.Dec()
	}

	logger.L(ctx).Info("task cancelled", zap.String("task_id", taskID))

	return nil
}

// GetTaskHistory 获取任务历史
func (s *TaskService) GetTaskHistory(ctx context.Context, deviceSN string, opts *TaskHistoryOptions) (*TaskListResponse, error) {
	tasks, total, err := s.repo.GetHistory(ctx, deviceSN, opts)
	if err != nil {
		return nil, err
	}

	return &TaskListResponse{
		Tasks:    tasks,
		Total:    total,
		Page:     opts.Page,
		PageSize: opts.PageSize,
	}, nil
}

// RecoverPendingTasks 恢复未完成任务（CPE 重连时调用）
// 检查 sent 状态超过指定时间的任务，重置为 pending
func (s *TaskService) RecoverPendingTasks(ctx context.Context, deviceSN string) error {
	// 获取 sent 状态超过 5 分钟的任务
	staleTasks, err := s.queue.GetStaleSentTasks(ctx, deviceSN, "5m")
	if err != nil {
		return fmt.Errorf("get stale tasks: %w", err)
	}

	for _, task := range staleTasks {
		if !task.CanRetry() {
			// 超过最大重试次数，标记为失败
			if err := s.MarkTaskFailed(ctx, task.ID, 0, "exceeded max retries"); err != nil {
				logger.L(ctx).Error("mark task failed", zap.Error(err), zap.String("task_id", task.ID))
			}
			continue
		}

		// 重置任务状态
		task.ResetForRetry()

		// 更新 Redis
		if err := s.queue.Update(ctx, task); err != nil {
			logger.L(ctx).Error("reset stale task", zap.Error(err), zap.String("task_id", task.ID))
			continue
		}

		// 重新入队
		if err := s.queue.Push(ctx, task); err != nil {
			logger.L(ctx).Error("requeue task", zap.Error(err), zap.String("task_id", task.ID))
			continue
		}

		// 同步 PostgreSQL
		if err := s.repo.Update(ctx, task); err != nil {
			logger.L(ctx).Error("sync task to db", zap.Error(err), zap.String("task_id", task.ID))
		}

		logger.L(ctx).Info("task recovered",
			zap.String("task_id", task.ID),
			zap.Int("retry_count", task.RetryCount))
	}

	return nil
}

// RestoreStats 汇报 RestorePendingQueues 的处理结果。
type RestoreStats struct {
	Scanned int // 从 PG 查出的 pending 任务数
	Pushed  int // 实际推送到 Redis 的任务数（Redis 中不存在）
	Skipped int // Redis 中已存在跳过的任务数
	Failed  int // 推送或检查失败的任务数
}

// pendingTaskLister 抽象 RestorePendingQueues 依赖的 PG 查询能力（便于单测 mock）。
type pendingTaskLister interface {
	ListPendingAllDevices(ctx context.Context, limit int) ([]*Task, error)
}

// taskEnqueuer 抽象 RestorePendingQueues 依赖的 Redis 队列能力（便于单测 mock）。
type taskEnqueuer interface {
	Exists(ctx context.Context, deviceSN, taskID string) (bool, error)
	Push(ctx context.Context, task *Task) error
}

// RestorePendingQueues 在进程启动时把 PostgreSQL 中仍为 pending 的任务重新灌入
// Redis 设备队列。对每个任务先用 ZScore 检查对应设备队列里是否已有该任务 ID，
// 已存在则跳过（Redis 重启持久化 / 其它实例并发启动都可能导致队列非空）。
// 只处理 pending 状态：sent 状态任务已在 CPE 在途，重新入队会引起重复下发，
// 由设备重连时的 RecoverPendingTasks 按 sent_at 陈旧阈值走正常恢复路径。
func (s *TaskService) RestorePendingQueues(ctx context.Context, limit int) (RestoreStats, error) {
	return restorePendingQueues(ctx, s.repo, s.queue, s.logger, limit)
}

// restorePendingQueues 是 RestorePendingQueues 的可测试实现，接受小接口。
func restorePendingQueues(
	ctx context.Context,
	lister pendingTaskLister,
	enq taskEnqueuer,
	log *zap.Logger,
	limit int,
) (RestoreStats, error) {
	tasks, err := lister.ListPendingAllDevices(ctx, limit)
	if err != nil {
		return RestoreStats{}, fmt.Errorf("list pending tasks: %w", err)
	}

	stats := RestoreStats{Scanned: len(tasks)}
	for _, t := range tasks {
		exists, err := enq.Exists(ctx, t.DeviceSN, t.ID)
		if err != nil {
			log.Warn("check queue existence",
				zap.String("task_id", t.ID),
				zap.String("device_sn", t.DeviceSN),
				zap.Error(err))
			stats.Failed++
			continue
		}
		if exists {
			stats.Skipped++
			continue
		}
		if err := enq.Push(ctx, t); err != nil {
			log.Warn("push pending task to queue",
				zap.String("task_id", t.ID),
				zap.String("device_sn", t.DeviceSN),
				zap.Error(err))
			stats.Failed++
			continue
		}
		stats.Pushed++
	}

	log.Info("restore pending task queues done",
		zap.Int("scanned", stats.Scanned),
		zap.Int("pushed", stats.Pushed),
		zap.Int("skipped", stats.Skipped),
		zap.Int("failed", stats.Failed))

	return stats, nil
}

// GetTaskStats 获取任务统计
func (s *TaskService) GetTaskStats(ctx context.Context, deviceSN string) (map[TaskStatus]int64, error) {
	return s.repo.CountByStatus(ctx, deviceSN)
}

// RetryTask 手动重试任务
func (s *TaskService) RetryTask(ctx context.Context, task *Task) error {
	if task == nil {
		return fmt.Errorf("task is nil")
	}

	// 更新 Redis
	if err := s.queue.Update(ctx, task); err != nil {
		return fmt.Errorf("update task in queue: %w", err)
	}

	// 重新入队
	if err := s.queue.Push(ctx, task); err != nil {
		return fmt.Errorf("requeue task: %w", err)
	}

	// 同步 PostgreSQL
	if err := s.repo.Update(ctx, task); err != nil {
		logger.L(ctx).Error("sync task to db", zap.Error(err), zap.String("task_id", task.ID))
	}

	logger.L(ctx).Info("task retried",
		zap.String("task_id", task.ID),
		zap.Int("retry_count", task.RetryCount))

	return nil
}

// BatchCreateTasks 批量创建任务
func (s *TaskService) BatchCreateTasks(ctx context.Context, reqs []*CreateTaskRequest) ([]*Task, error) {
	var tasks []*Task
	for _, req := range reqs {
		task := NewTask(req)
		tasks = append(tasks, task)
	}

	// 批量持久化
	if err := s.repo.BatchCreate(ctx, tasks); err != nil {
		return nil, fmt.Errorf("batch persist tasks: %w", err)
	}

	// 逐个推送到队列
	wakeDevices := make(map[string]struct{})
	var pushed []*Task
	for _, task := range tasks {
		if err := s.queue.Push(ctx, task); err != nil {
			logger.L(ctx).Error("enqueue task", zap.Error(err), zap.String("task_id", task.ID))
			continue
		}
		pushed = append(pushed, task)
		wakeDevices[task.DeviceSN] = struct{}{}
	}

	// 唤醒所有涉及的设备（去重）
	for sn := range wakeDevices {
		s.wakeDevice(sn)
	}

	return pushed, nil
}

// PurgeOldTasks 清理旧任务
func (s *TaskService) PurgeOldTasks(ctx context.Context, retentionDays int) (int64, error) {
	before := time.Now().AddDate(0, 0, -retentionDays).Format(time.RFC3339)
	return s.repo.PurgeOldTasks(ctx, before)
}

// wakeDevice sends a Connection Request to wake the device asynchronously.
// This is fire-and-forget: errors are logged but do not block task creation.
func (s *TaskService) wakeDevice(deviceSN string) {
	if s.deviceLookup == nil || s.connReq == nil {
		return
	}

	go func() {
		ctx := context.Background()
		httpURL, err := s.deviceLookup.GetConnectionRequestURL(ctx, deviceSN)
		if err != nil {
			s.logger.Warn("lookup device for connection request",
				zap.String("device_sn", deviceSN),
				zap.Error(err))
			return
		}

		if err := s.connReq.Send(ctx, deviceSN, httpURL); err != nil {
			s.logger.Warn("send connection request",
				zap.String("device_sn", deviceSN),
				zap.Error(err))
		}
	}()
}

// notifyCompletion broadcasts a terminal task state.
//
// 二选一：注入了 EventBus 则只广播（跨进程由订阅者调用聚合器）；否则 fallback
// 到同进程 callbacks（单进程部署、单测）。避免同一 TaskService 既发事件又回
// 调导致下游聚合器（如 mml_tasks 统计）重复计数。
//
// P1 重构（docs/design/mml-task-flow-design-20260424.md §3.3 C2）：
// 去掉 source == TaskSourceMML 硬编码过滤。任何带 source_id 的任务终态都会
// 广播，由 APP 侧 CompletionRouter 按 source 分发到对应聚合器；未注册的
// source 走 router 的 unknownHandler（记 warn 日志，不中断）。
// 仍保留 source_id 非空判断——匿名 / 临时任务（如 Console 执行命令按钮
// 产生的 api-source 任务）没有回流目标，不必占用广播带宽。
func (s *TaskService) notifyCompletion(ctx context.Context, task *Task) {
	if task.SourceID == "" {
		return
	}

	if s.eventBus != nil {
		subject := SubjectForStatus(task.Status)
		if subject == "" {
			return
		}
		evt, err := event.NewEvent(subject, task)
		if err != nil {
			s.logger.Warn("build task event",
				zap.String("task_id", task.ID),
				zap.String("status", string(task.Status)),
				zap.Error(err))
			return
		}
		if err := s.eventBus.Publish(ctx, subject, evt); err != nil {
			s.logger.Warn("publish task event",
				zap.String("subject", subject),
				zap.String("task_id", task.ID),
				zap.Error(err))
		}
		return
	}

	for _, cb := range s.callbacks {
		cb.OnTaskCompleted(ctx, task)
	}
}

// SubjectForStatus maps a terminal TaskStatus to its event subject.
// Returns "" for non-terminal statuses so callers can skip publishing.
func SubjectForStatus(status TaskStatus) string {
	switch status {
	case TaskStatusCompleted:
		return event.SubjectTaskCompleted
	case TaskStatusFailed, TaskStatusExpired:
		return event.SubjectTaskFailed
	default:
		return ""
	}
}

// TaskHistoryOptions 任务历史查询选项（定义在 model.go）


package task

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/omcgo/omcgo/internal/core/components/logger"
	coreerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/tracing"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
	"golang.org/x/sync/semaphore"
)

// defaultWakeConcurrency 是 wakeDevice 异步唤醒并发上界的安全默认值。
//
// 问题（issue #12）：原 wakeDevice 每次 CreateTask / BatchCreateTasks 去重设备都
// 无界 `go func()`，10 万 Inform 风暴 + 批量任务下 goroutine 暴涨 → 峰值 OOM，
// 与连接池耗尽互相放大级联。
//
// 取值权衡：唤醒是 fire-and-forget（失败仅 log，设备下个 periodic inform 会自愈），
// 不在任务创建关键路径上。256 给批量唤醒留足并行度（远高于单设备 Connection Request
// 的网络往返耗时所需），又把峰值 goroutine + 出站连接钉在常数级，避免发散。
// 生产可经 SetWakeConcurrency 调整。
const defaultWakeConcurrency = 256

const taskCreateRollbackTimeout = 3 * time.Second

// ErrQueueFull is returned by CreateTask when a device's pending-task queue has
// reached its configured depth cap. It is a backpressure signal (issue #7): an
// untrusted / offline device must not let its queue grow without bound and
// exhaust Redis/PostgreSQL. Callers should surface this as a 429-style refusal,
// not a 500.
var ErrQueueFull = errors.New("device task queue at capacity")

// ErrTaskNotPending means a Redis queue entry lost the PostgreSQL pending-state
// fence before ACS could send it. Callers should discard the stale execution
// copy and continue with the next queued task instead of aborting the CWMP
// session.
var ErrTaskNotPending = errors.New("task is no longer pending")

// ErrTaskNotFound is returned when a task ID does not resolve to an existing
// task (e.g. cancelling / marking a non-existent task). It wraps the core
// errors.ErrNotFound sentinel so that handlers mapping via
// coreerrors.HTTPStatusFromError surface it as HTTP 404 rather than 500, while
// errors.Is(err, ErrTaskNotFound) keeps a task-scoped check at call sites. The
// message deliberately carries no SQL / storage detail.
var ErrTaskNotFound = fmt.Errorf("task not found: %w", coreerrors.ErrNotFound)

// defaultMaxQueueDepth caps the number of pending tasks per device. It is
// generous enough for legitimate batch operations (sweep / template apply) yet
// bounds runaway growth toward an unreachable device. 0 (unset) disables the
// cap; production wiring sets it via SetMaxQueueDepth.
const defaultMaxQueueDepth = 1000

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

	// defaultExpiresIn: T-0157 C1 — device task 创建兜底默认超时秒数。
	// 调用方语义见 appconfig.TaskConfig.DefaultExpiresInSeconds。0 表示未配置（不兜底）。
	defaultExpiresIn int

	// createFailureNotifier: T-0157 C6 — CreateTask 入队失败兜底通知。
	// 由 cmd/app/bootstrap.go 通过 SetCreateFailureNotifier 注入；
	// repo.Create / queue.Push 失败时调用，让消息中心 100% 覆盖用户操作。
	// nil = 未注入（如 acs/worker 进程不调 CreateTask，无需注入）。
	createFailureNotifier CreateFailureNotifier

	// issue #12 — wakeDevice 异步唤醒的有界并发控制。
	// wakeSem 限制同时在飞的唤醒 goroutine 数（背压：满载时 TryAcquire 失败即丢弃，
	// 不阻塞 CreateTask 关键路径——设备下个 periodic inform 会自愈）。
	// 经 wakeSemOnce 懒初始化，使既有 `&TaskService{...}` 字面量构造（含单测）零改动仍受保护。
	wakeSem         *semaphore.Weighted
	wakeSemOnce     sync.Once
	wakeConcurrency int // <=0 用 defaultWakeConcurrency

	// maxQueueDepth: issue #7 — 每设备 pending 队列深度上限（背压）。
	// 0 表示不限制（向后兼容历史行为）；生产由 SetMaxQueueDepth 注入。
	maxQueueDepth int
}

// CreateFailureNotifier 在 CreateTask 入队失败时被调用，把失败信息写入消息中心。
// 实现端（internal/notification）负责按 task.CreatorID 隔离 + 渲染文案 + UpsertByDedup。
// closure 不返回 error —— 通知失败不应影响 CreateTask 的错误返回链路。
type CreateFailureNotifier func(ctx context.Context, task *Task, errMsg string)

type createdTaskRollbackRepository interface {
	Delete(ctx context.Context, id string) error
}

func deleteCreatedTaskAfterEnqueueFailure(
	ctx context.Context,
	repo createdTaskRollbackRepository,
	taskID string,
) error {
	rollbackCtx, cancel := context.WithTimeout(
		context.WithoutCancel(ctx),
		taskCreateRollbackTimeout,
	)
	defer cancel()
	return repo.Delete(rollbackCtx, taskID)
}

// NewTaskService 创建任务服务
func NewTaskService(queue *RedisTaskQueue, repo *PgTaskRepository, log *zap.Logger) *TaskService {
	return &TaskService{
		queue:         queue,
		repo:          repo,
		logger:        log,
		maxQueueDepth: defaultMaxQueueDepth, // issue #7: 默认开启每设备队列背压，可经 SetMaxQueueDepth 调整
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

// ListResultsBySourceID 透传到底层 PgTaskRepository，供 mml.Service 在
// GET /api/v1/mml/tasks/:id/results 展示设备级执行结果（任务记录页查看 modal）。
//
// 历史问题（2026-05-23 修复）：原 mml.GetTaskResults 读 mml_tasks.results JSONB，
// 但执行结果实际全在 device_tasks，导致 UI 永远"暂无执行结果"。
func (s *TaskService) ListResultsBySourceID(
	ctx context.Context, sourceID string, page, pageSize int,
) ([]DeviceTaskResultRow, int64, error) {
	return s.repo.ListResultsBySourceID(ctx, sourceID, page, pageSize)
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

// SetWakeConcurrency 配置 wakeDevice 异步唤醒的并发上界（issue #12）。
// <=0 时回退到 defaultWakeConcurrency。须在首次 wakeDevice 调用前设置
// （信号量懒初始化只读一次本值）。仅 cmd/app/bootstrap.go 在装配期调用。
func (s *TaskService) SetWakeConcurrency(n int) {
	s.wakeConcurrency = n
}

// wakeSemaphore 懒初始化并返回唤醒并发信号量。
// 用 sync.Once 保证多 goroutine 首次并发 wakeDevice 时只建一个信号量。
func (s *TaskService) wakeSemaphore() *semaphore.Weighted {
	s.wakeSemOnce.Do(func() {
		limit := s.wakeConcurrency
		if limit <= 0 {
			limit = defaultWakeConcurrency
		}
		s.wakeSem = semaphore.NewWeighted(int64(limit))
	})
	return s.wakeSem
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

// SetCreateFailureNotifier 注入入队失败通知 closure（T-0157 C6）。
// 仅 cmd/app/bootstrap.go 调用（app 是唯一 CreateTask 入口）。
func (s *TaskService) SetCreateFailureNotifier(fn CreateFailureNotifier) {
	s.createFailureNotifier = fn
}

// SetMaxQueueDepth 配置每设备 pending 队列深度上限（issue #7 背压）。
// n <= 0 表示禁用上限（向后兼容历史行为）。命中上限时 CreateTask 返回 ErrQueueFull。
func (s *TaskService) SetMaxQueueDepth(n int) {
	if n < 0 {
		n = 0
	}
	s.maxQueueDepth = n
}

// SetDefaultExpiresIn 配置 device task 创建的默认超时兜底秒数（T-0157 C1）。
// 仅当 CreateTaskRequest.ExpiresIn == 0 时生效；调用方显式传 0 等价于声明"永不超时"
// 但本兜底仍会覆盖（如需真正永不超时，调用方需显式传一个极大值如 86400）。
// 负值或 0 表示不启用兜底，等价于历史行为。
func (s *TaskService) SetDefaultExpiresIn(seconds int) {
	if seconds < 0 {
		seconds = 0
	}
	s.defaultExpiresIn = seconds
}

func (s *TaskService) applyDefaultExpiresIn(req *CreateTaskRequest) {
	if req != nil && req.ExpiresIn == 0 && s.defaultExpiresIn > 0 {
		req.ExpiresIn = s.defaultExpiresIn
	}
}

// CreateTask 创建新任务
func (s *TaskService) CreateTask(ctx context.Context, req *CreateTaskRequest) (*Task, error) {
	ctx, span := tracing.StartSpan(ctx, tracing.TaskTracerName, "Task CreateTask",
		attribute.String("task.device_sn", req.DeviceSN),
		attribute.String("task.method", req.Method),
	)
	defer span.End()

	// issue #7: 每设备 pending 队列深度背压 —— 防止面向不可达 / 不可信设备的任务
	// 无界堆积耗尽 Redis/PG。命中上限直接拒绝（不落库、不入队），调用方按 429 处理。
	if !req.FailImmediately && s.maxQueueDepth > 0 {
		depth, err := s.queue.Len(ctx, req.DeviceSN)
		if err != nil {
			return nil, fmt.Errorf("check queue depth: %w", err)
		}
		if depth >= int64(s.maxQueueDepth) {
			logger.L(ctx).Warn("device task queue at capacity, rejecting new task",
				zap.String("device_sn", req.DeviceSN),
				zap.Int64("depth", depth),
				zap.Int("max_depth", s.maxQueueDepth))
			return nil, fmt.Errorf("device %s: %w", req.DeviceSN, ErrQueueFull)
		}
	}

	// T-0157 C1: 兜底默认超时（调用方未传 → 用配置默认；保留显式覆盖能力）
	s.applyDefaultExpiresIn(req)

	task := NewTask(req)

	// 1. 持久化到 PostgreSQL
	if err := s.repo.Create(ctx, task); err != nil {
		// T-0157 C6: 入队失败兜底 → 写一条 status=failed 的消息（避免用户感知"点了没反应"）
		s.notifyCreateFailure(ctx, task, err)
		return nil, fmt.Errorf("persist task: %w", err)
	}
	if req.FailImmediately {
		s.recordCompletion(task, TaskStatusFailed)
		s.notifyCompletion(ctx, task)
		return task, nil
	}

	// 2. 推送到 Redis 队列
	if err := s.queue.Push(ctx, task); err != nil {
		// 回滚 PostgreSQL 记录。回滚失败 → PG 留下 pending 孤儿（#13）：记 error + metric，
		// 由 RestorePendingQueues（启动期）/ ExpiredSweeper（过期）兜底，不让其静默漂移。
		if derr := deleteCreatedTaskAfterEnqueueFailure(ctx, s.repo, task.ID); derr != nil {
			s.recordDualWriteFail("create_rollback")
			logger.L(ctx).Error("rollback task pg record after enqueue failure",
				zap.String("task_id", task.ID),
				zap.NamedError("enqueue_err", err),
				zap.NamedError("rollback_err", derr))
		}
		// T-0157 C6: 同上
		s.notifyCreateFailure(ctx, task, err)
		return nil, fmt.Errorf("enqueue task: %w", err)
	}

	if s.metrics != nil {
		s.metrics.PendingTotal.Inc()
	}

	// 3. 异步触发 Connection Request 唤醒设备
	s.wakeDevice(task.DeviceSN)

	// 4. T-0157 C5: publish task.created 让消息中心订阅器写"进行中"消息
	// 触发条件 CreatorID 非空（用户操作）—— 与订阅器 upsertFromTask 的 CreatorID 跳过逻辑一致。
	// 系统任务（PeriodicSyncer / F09 等内部触发）CreatorID 为空，不广播也不打扰用户。
	if s.eventBus != nil && task.CreatorID != "" {
		if evt, err := event.NewEvent(event.SubjectTaskCreated, task); err == nil {
			if perr := s.eventBus.Publish(ctx, event.SubjectTaskCreated, evt); perr != nil {
				s.logger.Warn("publish task.created",
					zap.String("task_id", task.ID),
					zap.Error(perr))
			}
		}
	}

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
	if task == nil {
		// 从 PostgreSQL 获取
		return s.repo.GetByID(ctx, taskID)
	}
	resolved, err := resolveTaskDetails(ctx, task, s.repo.GetByID)
	if err != nil {
		return nil, fmt.Errorf("get terminal task from repository: %w", err)
	}
	return resolved, nil
}

func resolveTaskDetails(
	ctx context.Context,
	queueTask *Task,
	loadDurable func(context.Context, string) (*Task, error),
) (*Task, error) {
	if queueTask == nil || !isTerminal(queueTask.Status) {
		return queueTask, nil
	}
	durable, err := loadDurable(ctx, queueTask.ID)
	if err != nil {
		return nil, err
	}
	if durable != nil {
		return durable, nil
	}
	return queueTask, nil
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
	if s.repo != nil {
		count, err := s.repo.CountOpenByDevice(ctx, deviceSN, time.Now())
		if err == nil {
			return count, nil
		}
		logger.L(ctx).Warn("count open tasks from pg failed, falling back to redis queue length",
			zap.String("device_sn", deviceSN),
			zap.Error(err))
	}
	return s.queue.Len(ctx, deviceSN)
}

// ReleasePlannedTask is the single execution-plane admission path for a task
// whose durable row was created by an upstream transactional planner. It owns
// queue capacity enforcement, idempotent Redis admission, metrics and wake-up,
// while the planner remains responsible only for the business plan.
func (s *TaskService) ReleasePlannedTask(ctx context.Context, planned *Task) (bool, error) {
	return s.releasePlannedTask(ctx, planned, true)
}

// ReleasePlannedTaskWithoutWake admits a durable planned task without sending
// a per-task Connection Request. Batch dispatchers use it to enqueue all work
// for one device first and then wake that device once via WakePlannedDevice.
func (s *TaskService) ReleasePlannedTaskWithoutWake(ctx context.Context, planned *Task) (bool, error) {
	return s.releasePlannedTask(ctx, planned, false)
}

func (s *TaskService) releasePlannedTask(ctx context.Context, planned *Task, wake bool) (bool, error) {
	if planned == nil || planned.Status != TaskStatusPending {
		return false, nil
	}
	exists, err := s.queue.Exists(ctx, planned.DeviceSN, planned.ID)
	if err != nil {
		return false, fmt.Errorf("check planned task queue membership: %w", err)
	}
	if exists {
		return true, nil
	}
	if s.maxQueueDepth > 0 {
		depth, err := s.queue.Len(ctx, planned.DeviceSN)
		if err != nil {
			return false, fmt.Errorf("check planned task queue depth: %w", err)
		}
		if depth >= int64(s.maxQueueDepth) {
			return false, fmt.Errorf("device %s: %w", planned.DeviceSN, ErrQueueFull)
		}
	}
	if err := s.queue.Push(ctx, planned); err != nil {
		return false, fmt.Errorf("release planned task: %w", err)
	}
	if s.metrics != nil {
		s.metrics.PendingTotal.Inc()
	}
	if wake {
		s.wakeDevice(planned.DeviceSN)
	}
	return true, nil
}

// WakePlannedDevice performs the single best-effort wake after a batch of
// durable tasks has been admitted to the execution queue.
func (s *TaskService) WakePlannedDevice(deviceSN string) {
	s.wakeDevice(deviceSN)
}

// EvictPlannedTask removes an unreleased/cancelled planned task from the
// execution plane. It is deliberately idempotent so an outbox cancellation can
// safely race an earlier enqueue delivery or be replayed after Redis recovery.
func (s *TaskService) EvictPlannedTask(ctx context.Context, planned *Task) error {
	if planned == nil {
		return nil
	}
	if err := s.queue.Delete(ctx, planned.ID); err != nil {
		return fmt.Errorf("evict planned task: %w", err)
	}
	return nil
}

func (s *TaskService) LatestOpenTaskByDeviceAndMethod(ctx context.Context, deviceSN, method, description string) (*Task, error) {
	if s.repo == nil {
		return nil, nil
	}
	return s.repo.LatestOpenByDeviceMethodDescription(ctx, deviceSN, method, description)
}

func (s *TaskService) ListOpenTasksByDeviceAndMethods(
	ctx context.Context,
	deviceSN string,
	methods []string,
) ([]*Task, error) {
	if s.repo == nil {
		return nil, nil
	}
	return s.repo.ListOpenByDeviceAndMethods(ctx, deviceSN, methods)
}

func (s *TaskService) LatestCompletedTaskByDeviceAndCommandKey(
	ctx context.Context,
	deviceSN, commandKey string,
) (*Task, error) {
	if s.repo == nil {
		return nil, nil
	}
	return s.repo.LatestCompletedByDeviceCommandKey(ctx, deviceSN, commandKey)
}

func (s *TaskService) LatestSyncGPVSummaryByDevice(ctx context.Context, deviceSN string) (*SyncGPVSummary, error) {
	if s.repo == nil {
		return nil, nil
	}
	return s.repo.LatestSyncGPVSummaryByDevice(ctx, deviceSN)
}

func (s *TaskService) CountOpenSyncGPVByDevice(ctx context.Context, deviceSN string) (int64, error) {
	if s.repo == nil {
		return 0, nil
	}
	return s.repo.CountOpenSyncGPVByDevice(ctx, deviceSN)
}

func (s *TaskService) HasOpenSyncGPVTasksByDevice(ctx context.Context, deviceSN string) (bool, error) {
	if s.repo == nil {
		return false, nil
	}
	return s.repo.HasOpenSyncGPVTasksByDevice(ctx, deviceSN)
}

func (s *TaskService) AcquireSyncGPVDeviceLock(ctx context.Context, deviceSN string) (func(), error) {
	if s.repo == nil {
		return func() {}, nil
	}
	return s.repo.AcquireSyncGPVDeviceLock(ctx, deviceSN)
}

// MarkTaskSent 标记任务已发送
func (s *TaskService) MarkTaskSent(ctx context.Context, taskID, cwmpID string) error {
	if s.repo != nil {
		acquired, err := s.repo.MarkSentIfPending(ctx, taskID, cwmpID, time.Now())
		if err != nil {
			return err
		}
		if !acquired {
			// The queue may still contain a stale copy after the PG task or its
			// durable run became terminal. Drop it here so every subsequent Inform
			// does not pop and reject the same task forever.
			if err := s.queue.Delete(ctx, taskID); err != nil {
				return fmt.Errorf("task %s is no longer pending; remove stale queue copy: %w", taskID, err)
			}
			return fmt.Errorf("task %s: %w", taskID, ErrTaskNotPending)
		}
	}
	// 更新 Redis
	if err := s.queue.MarkTaskSent(ctx, taskID, cwmpID); err != nil {
		if s.repo != nil {
			return s.releaseUnwrittenSendClaim(ctx, taskID, cwmpID, err)
		}
		return fmt.Errorf("mark task sent in queue: %w", err)
	}

	logger.L(ctx).Info("task sent",
		zap.String("task_id", taskID),
		zap.String("cwmp_id", cwmpID))

	return nil
}

func (s *TaskService) releaseUnwrittenSendClaim(ctx context.Context, taskID, cwmpID string, cause error) error {
	released, err := s.repo.ReleaseSentClaimIfUnwritten(ctx, taskID, cwmpID)
	if err != nil {
		s.recordDualWriteFail("release_send_claim")
		return errors.Join(fmt.Errorf("mark task sent in queue: %w", cause), err)
	}
	if !released {
		return errors.Join(
			fmt.Errorf("mark task sent in queue: %w", cause),
			fmt.Errorf("task %s send claim changed before compensation", taskID),
		)
	}

	var repairErrs []error
	if err := s.queue.DeleteCWMPIDMapping(ctx, cwmpID); err != nil {
		repairErrs = append(repairErrs, fmt.Errorf("delete unwritten cwmp mapping: %w", err))
	}
	pending, err := s.repo.GetByID(ctx, taskID)
	if err != nil {
		repairErrs = append(repairErrs, fmt.Errorf("load released task send claim: %w", err))
	} else if pending != nil {
		var transitionErr *taskTransitionError
		if errors.As(cause, &transitionErr) {
			err = s.queue.rollbackSentTransition(ctx, pending, cwmpID, transitionErr.token)
		} else {
			err = s.queue.Update(ctx, pending)
		}
		if err != nil {
			repairErrs = append(repairErrs, fmt.Errorf("restore released task queue entry: %w", err))
		}
	}
	if len(repairErrs) > 0 {
		s.recordDualWriteFail("release_send_claim_queue")
	}
	return errors.Join(append([]error{fmt.Errorf("mark task sent in queue: %w", cause)}, repairErrs...)...)
}

// commitTransition owns the Redis/PG state transition protocol:
//  1. Redis CAS elects exactly one winner and persists its snapshot.
//  2. PostgreSQL conditionally applies it from the same observed source state.
//  3. Only after PG confirmation are cross-slot indexes cleaned and a TTL set.
//
// A PG or cleanup outage therefore leaves a durable Redis compensation record
// for the reconciler instead of losing the only winning state after a short TTL.
func (s *TaskService) commitTransition(
	ctx context.Context,
	current, candidate *Task,
	materializeMissing bool,
) (bool, error) {
	token, changed, err := s.queue.prepareTransition(ctx, current, candidate, materializeMissing)
	if err != nil || !changed {
		return changed, err
	}
	pgChanged, err := s.repo.TransitionIfStatus(ctx, candidate, current.Status)
	if err != nil {
		s.recordDualWriteFail("sync_transition")
		return false, fmt.Errorf("persist task transition: %w", err)
	}
	if !pgChanged {
		return false, nil
	}
	if err := s.queue.acknowledgeTransition(ctx, candidate.ID, token); err != nil {
		// PG is already authoritative. Keep the persistent pending index so the
		// worker can retry cross-slot cleanup without replaying the business event.
		s.recordDualWriteFail("cleanup_transition")
		logger.L(ctx).Warn("defer task transition cleanup",
			zap.String("task_id", candidate.ID),
			zap.Error(err))
	}
	if isTerminal(candidate.Status) && (candidate.SourceID != "" || candidate.CreatorID != "") {
		if err := s.PublishTransitionEvent(ctx, candidate, token); err != nil {
			s.recordDualWriteFail("publish_transition")
			logger.L(ctx).Warn("defer task transition event",
				zap.String("task_id", candidate.ID),
				zap.String("transition_token", token),
				zap.Error(err))
		}
	}
	return true, nil
}

// MarkTaskCompleted 标记任务完成
func (s *TaskService) MarkTaskCompleted(ctx context.Context, taskID string, result json.RawMessage) error {
	current, err := s.queue.GetByID(ctx, taskID)
	if err != nil {
		return fmt.Errorf("get task for completion: %w", err)
	}
	if current == nil {
		return ErrTaskNotFound
	}
	task := cloneTaskSnapshot(current)
	task.MarkCompleted(result)
	changed, transitionErr := s.commitTransition(ctx, current, task, false)
	if transitionErr != nil {
		// ACS already accepted the device response. The persistent transition
		// record lets the worker retry PG sync; do not make the CWMP response fail.
		logger.L(ctx).Error("defer completed task persistence",
			zap.String("task_id", taskID), zap.Error(transitionErr))
		return nil
	}
	if !changed {
		return nil
	}

	s.recordCompletion(task, TaskStatusCompleted)

	logger.L(ctx).Info("task completed",
		zap.String("task_id", taskID),
		zap.String("method", task.Method))

	return nil
}

// MarkTaskFailed 标记任务失败
func (s *TaskService) MarkTaskFailed(ctx context.Context, taskID string, errorCode int, errorMsg string) error {
	return s.MarkTaskFailedWithResult(ctx, taskID, errorCode, errorMsg, nil)
}

func shouldAutoRetryOnFailure(t *Task) bool {
	return t != nil &&
		(t.Source == TaskSourceMML || t.Source == TaskSourceGeofence) &&
		t.CanRetry()
}

func cloneTaskSnapshot(task *Task) *Task {
	if task == nil {
		return nil
	}
	clone := *task
	clone.Params = append(json.RawMessage(nil), task.Params...)
	clone.Result = append(json.RawMessage(nil), task.Result...)
	return &clone
}

// MarkTaskFailedWithResult 标记任务失败并附带结构化 result（如 per-param SetParameterValuesFault 详情）。
// result 为空时等价于 MarkTaskFailed —— 不会清空已有 task.Result。
//
// T-0174 引入：ACS handleSOAPFault 在 SetParameterValues 失败时把每个失败 path 的
// (parameter_name / fault_code / fault_string) 序列化到 result，下游 MML
// ResultAggregator 据此触发 is_supported=false auto-learn。
func (s *TaskService) MarkTaskFailedWithResult(ctx context.Context, taskID string, errorCode int, errorMsg string, result json.RawMessage) error {
	task, err := s.queue.GetByID(ctx, taskID)
	if err != nil {
		return fmt.Errorf("get failed task: %w", err)
	}
	if task != nil && isTerminal(task.Status) {
		// 已完成任务收到迟到重复 Fault 时保持第一个终态；尤其不能把仍有
		// retry budget 的 MML completed task 重新放回 pending。
		return nil
	}
	if task == nil {
		return ErrTaskNotFound
	}
	if shouldAutoRetryOnFailure(task) {
		current := cloneTaskSnapshot(task)
		candidate := cloneTaskSnapshot(task)
		if len(result) > 0 {
			candidate.Result = result
		}
		candidate.ErrorCode = errorCode
		candidate.ErrorMessage = errorMsg
		candidate.ResetForRetryAfter(candidate.RetryInterval())
		changed, transitionErr := s.commitTransition(ctx, current, candidate, false)
		if transitionErr != nil {
			logger.L(ctx).Error("defer retry task persistence",
				zap.String("task_id", taskID), zap.Error(transitionErr))
			return nil
		}
		if !changed {
			return nil
		}
		logger.L(ctx).Info("mml task scheduled for retry",
			zap.String("task_id", taskID),
			zap.Int("retry_count", candidate.RetryCount),
			zap.Int("max_retries", candidate.MaxRetries),
			zap.Int("retry_interval_seconds", candidate.RetryIntervalSeconds),
			zap.Time("next_attempt_at", deref(candidate.NextAttemptAt)))
		return nil
	}

	current := cloneTaskSnapshot(task)
	candidate := cloneTaskSnapshot(task)
	candidate.MarkFailedWithResult(errorCode, errorMsg, result)
	changed, transitionErr := s.commitTransition(ctx, current, candidate, false)
	if transitionErr != nil {
		logger.L(ctx).Error("defer failed task persistence",
			zap.String("task_id", taskID), zap.Error(transitionErr))
		return nil
	}
	if !changed {
		return nil
	}
	task = candidate

	s.recordCompletion(task, TaskStatusFailed)

	logger.L(ctx).Info("task failed",
		zap.String("task_id", taskID),
		zap.Int("error_code", errorCode),
		zap.String("error_message", errorMsg),
		zap.Int("result_bytes", len(result)))

	return nil
}

// ExpireTask 把单个任务标记为 expired（T-0157 C2）。
//
// 与 MarkTaskFailed 不同：调用方已通过 repo.ListExpiredCandidates 持有完整 Task 对象，
// 跳过 GetByID 一次往返。流程：MarkExpired → repo.Update → queue.Update（原子写入短 TTL
// 终态并清理索引，warn 不中断）→ metrics 计数 → notifyCompletion 广播（复用 task.failed 主题，
// 订阅器按 task.Status 区分 failed / expired —— 详见 SubjectForStatus）。
//
// 用于 worker 进程的 ExpiredSweeper；其他场景请用 MarkTaskFailed 走 Redis 真相源。
func (s *TaskService) ExpireTask(ctx context.Context, task *Task) error {
	if task == nil {
		return nil
	}
	current := cloneTaskSnapshot(task)
	candidate := cloneTaskSnapshot(task)
	candidate.MarkExpired()
	changed, err := s.commitTransition(ctx, current, candidate, true)
	if err != nil {
		return fmt.Errorf("expire task: %w", err)
	}
	if !changed {
		return nil
	}
	task = candidate
	s.recordCompletion(candidate, TaskStatusExpired)
	s.logger.Info("task expired by sweeper",
		zap.String("task_id", task.ID),
		zap.String("device_sn", task.DeviceSN),
		zap.Time("expires_at", deref(task.ExpiresAt)))
	return nil
}

func deref(t *time.Time) time.Time {
	if t == nil {
		return time.Time{}
	}
	return *t
}

// recordCompletion 在任务进入终态时按 (source, status) 递增 CompletedTotal 并递减
// PendingTotal（issue #116）。CompletedTotal 是 {source, status} 双标签 CounterVec
// （与 CompletionRouter.Dispatch 的用法对齐），少传任一标签会 panic
// "inconsistent label cardinality"——曾导致 ExpiredSweeper 每 10s panic 整个 worker。
// task 为 nil 时 source 记空串兜底；metrics 未注入（单测 / 未调 SetMetrics）时安全跳过。
func (s *TaskService) recordCompletion(task *Task, status TaskStatus) {
	if s.metrics == nil {
		return
	}
	var source TaskSource
	if task != nil {
		source = task.Source
	}
	s.metrics.CompletedTotal.WithLabelValues(string(source), string(status)).Inc()
	s.metrics.PendingTotal.Dec()
}

// recordDualWriteFail 在双写中断（写一半失败）时递增可观测指标（#13）。
// metrics 未注入（单测 / 未调 SetMetrics）时安全跳过。
func (s *TaskService) recordDualWriteFail(op string) {
	if s.metrics != nil {
		s.metrics.DualWriteFailTotal.WithLabelValues(op).Inc()
		s.metrics.QueueWriteFailuresTotal.WithLabelValues(op).Inc()
	}
}

// notifyCreateFailure 在 CreateTask 失败路径上调 createFailureNotifier（T-0157 C6）。
// 系统任务（CreatorID 为空）不通知；notifier 未注入也跳过；panic 隔离不影响主返回。
func (s *TaskService) notifyCreateFailure(ctx context.Context, task *Task, err error) {
	if s.createFailureNotifier == nil || task == nil || task.CreatorID == "" || err == nil {
		return
	}
	defer func() {
		if r := recover(); r != nil {
			s.logger.Warn("create failure notifier panicked",
				zap.String("task_id", task.ID),
				zap.Any("recover", r))
		}
	}()
	s.createFailureNotifier(ctx, task, err.Error())
}

// CancelTask 取消任务
func (s *TaskService) CancelTask(ctx context.Context, taskID string) error {
	task, err := s.GetTask(ctx, taskID)
	if err != nil {
		return fmt.Errorf("get task for cancel: %w", err)
	}
	if task == nil {
		return ErrTaskNotFound
	}

	// 只能取消 pending 状态的任务
	if task.Status != TaskStatusPending {
		return fmt.Errorf("cannot cancel task with status: %s", task.Status)
	}

	current := cloneTaskSnapshot(task)
	candidate := cloneTaskSnapshot(task)
	now := time.Now()
	candidate.Status = TaskStatusCancelled
	candidate.CompletedAt = &now
	changed, err := s.commitTransition(ctx, current, candidate, false)
	if err != nil {
		return fmt.Errorf("cancel task: %w", err)
	}
	if !changed {
		return nil
	}
	task = candidate
	s.recordCompletion(candidate, TaskStatusCancelled)
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

const recoverSentTaskBatchSize = 500

// RecoverPendingTasks 恢复未完成任务（CPE 重连/新 Inform 时调用）。
//
// 新 Inform 表示 CPE 已开启新的 CWMP 会话；上一会话里仍处于 sent 的 RPC
// 不会再返回响应。如果继续等待 5 分钟 stale 阈值，参数同步页面会在“待处理 N 次 GPV”
// 上无谓卡住。sent 任务已从 Redis 队列弹出，所以这里以 PG 为准取回同设备 sent 任务，
// 并按重试预算恢复为 pending，让当前会话可以继续 PopTask。
func (s *TaskService) RecoverPendingTasks(ctx context.Context, deviceSN string) error {
	staleTasks, err := s.repo.ListSentByDeviceBefore(ctx, deviceSN, time.Now(), recoverSentTaskBatchSize)
	if err != nil {
		return fmt.Errorf("get stale tasks: %w", err)
	}

	for _, task := range staleTasks {
		if !task.CanRetry() {
			// 超过最大重试次数，标记为失败
			current := cloneTaskSnapshot(task)
			candidate := cloneTaskSnapshot(task)
			candidate.MarkFailed(0, "exceeded max retries")
			changed, err := s.commitTransition(ctx, current, candidate, true)
			if err != nil {
				logger.L(ctx).Error("defer exhausted task transition", zap.Error(err), zap.String("task_id", task.ID))
				continue
			}
			if !changed {
				continue
			}
			s.recordCompletion(candidate, TaskStatusFailed)
			continue
		}
		if interval := task.RetryInterval(); interval > 0 && task.SentAt != nil {
			next := task.SentAt.Add(interval)
			if time.Now().Before(next) {
				logger.L(ctx).Debug("stale task retry interval not reached",
					zap.String("task_id", task.ID),
					zap.Time("next_retry_at", next),
					zap.Int("retry_interval_seconds", task.RetryIntervalSeconds))
				continue
			}
		}

		current := cloneTaskSnapshot(task)
		candidate := cloneTaskSnapshot(task)
		candidate.ResetForRetry()
		changed, err := s.commitTransition(ctx, current, candidate, true)
		if err != nil {
			logger.L(ctx).Error("defer stale task retry", zap.Error(err), zap.String("task_id", task.ID))
			continue
		}
		if !changed {
			continue
		}

		logger.L(ctx).Info("task recovered",
			zap.String("task_id", task.ID),
			zap.Int("retry_count", candidate.RetryCount))
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

// pendingPageLister 是 keyset 流式分页能力（#11）。生产 PgTaskRepository 实现它，
// RestorePendingQueues 在调用方未给显式上界（limit<=0）时按页恢复，避免把全量
// pending 任务一次性读入内存。仅实现 pendingTaskLister 的 mock 退回单批路径。
type pendingPageLister interface {
	ListPendingPage(ctx context.Context, after PendingCursor, batchSize int) ([]*Task, error)
}

// restorePageBatchSize 是流式恢复的每页行数。每页处理完即可被 GC，内存占用恒定。
const restorePageBatchSize = 1000

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
	stats, err := restorePendingQueues(ctx, s.repo, s.queue, s.logger, limit)
	// issue #20：把启动期实际重灌进 Redis 的 pending 任务数记为 recovery 动作。
	// 突增说明上一次停机时有大量 pending 未被消费 / 双写漂移被此处兜底。
	if s.metrics != nil && stats.Pushed > 0 {
		s.metrics.RecoveryActionTotal.WithLabelValues(RecoveryActionRestorePending).Add(float64(stats.Pushed))
	}
	return stats, err
}

// restorePendingQueues 是 RestorePendingQueues 的可测试实现，接受小接口。
//
// limit<=0 且 lister 支持 keyset 分页（pendingPageLister）时走流式恢复：按 (created_at, id)
// 游标逐页拉取并灌入 Redis，每页处理完即释放，内存占用恒定，避免百万 pending 任务 OOM（#11）。
// limit>0（调用方给了显式上界）或 lister 不支持分页时，退回单批 ListPendingAllDevices。
func restorePendingQueues(
	ctx context.Context,
	lister pendingTaskLister,
	enq taskEnqueuer,
	log *zap.Logger,
	limit int,
) (RestoreStats, error) {
	var stats RestoreStats

	if pager, ok := lister.(pendingPageLister); ok && limit <= 0 {
		cursor := PendingCursor{}
		for {
			tasks, err := pager.ListPendingPage(ctx, cursor, restorePageBatchSize)
			if err != nil {
				return stats, fmt.Errorf("list pending tasks: %w", err)
			}
			if len(tasks) == 0 {
				break
			}
			for _, t := range tasks {
				restoreOnePendingTask(ctx, enq, log, t, &stats)
			}
			// 末页：返回行数不足一批，无更多数据。
			if len(tasks) < restorePageBatchSize {
				break
			}
			last := tasks[len(tasks)-1]
			cursor = PendingCursor{CreatedAt: last.CreatedAt, ID: last.ID}
		}
	} else {
		tasks, err := lister.ListPendingAllDevices(ctx, limit)
		if err != nil {
			return stats, fmt.Errorf("list pending tasks: %w", err)
		}
		for _, t := range tasks {
			restoreOnePendingTask(ctx, enq, log, t, &stats)
		}
	}

	log.Info("restore pending task queues done",
		zap.Int("scanned", stats.Scanned),
		zap.Int("pushed", stats.Pushed),
		zap.Int("skipped", stats.Skipped),
		zap.Int("failed", stats.Failed))

	return stats, nil
}

// restoreOnePendingTask 把单条 pending 任务幂等灌入 Redis 队列并累加统计。
// Redis 中已存在则跳过（多实例并发启动 / Redis 持久化重启都可能导致队列非空）。
func restoreOnePendingTask(
	ctx context.Context,
	enq taskEnqueuer,
	log *zap.Logger,
	t *Task,
	stats *RestoreStats,
) {
	stats.Scanned++
	exists, err := enq.Exists(ctx, t.DeviceSN, t.ID)
	if err != nil {
		log.Warn("check queue existence",
			zap.String("task_id", t.ID),
			zap.String("device_sn", t.DeviceSN),
			zap.Error(err))
		stats.Failed++
		return
	}
	if exists {
		stats.Skipped++
		return
	}
	if err := enq.Push(ctx, t); err != nil {
		log.Warn("push pending task to queue",
			zap.String("task_id", t.ID),
			zap.String("device_sn", t.DeviceSN),
			zap.Error(err))
		stats.Failed++
		return
	}
	stats.Pushed++
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

	current, err := s.repo.GetByID(ctx, task.ID)
	if err != nil {
		return fmt.Errorf("load durable task for retry: %w", err)
	}
	if current == nil {
		return ErrTaskNotFound
	}
	changed, err := s.commitTransition(ctx, current, task, true)
	if err != nil {
		return fmt.Errorf("retry task: %w", err)
	}
	if !changed {
		return ErrTaskNotPending
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
		s.applyDefaultExpiresIn(req)
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
		if task.Status == TaskStatusFailed {
			s.recordCompletion(task, TaskStatusFailed)
			s.notifyCompletion(ctx, task)
			continue
		}
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
//
// issue #12：派生的 goroutine 受 wakeSemaphore 有界并发控制。满载时 TryAcquire
// 失败即丢弃本次唤醒（不阻塞调用方关键路径，也不无界堆积 goroutine）——唤醒本就
// 是 best-effort，被丢弃的设备会在下一个 periodic inform 周期自行重连。这把峰值
// goroutine 数 + 出站 Connection Request 连接钉在常数级，防 10 万风暴下发散 OOM。
func (s *TaskService) wakeDevice(deviceSN string) {
	if s.deviceLookup == nil || s.connReq == nil {
		return
	}

	sem := s.wakeSemaphore()
	if !sem.TryAcquire(1) {
		// 背压：在飞唤醒已达上界，丢弃本次（设备下个 periodic inform 自愈）。
		if s.metrics != nil {
			s.metrics.WakeDropped.Inc()
		}
		s.logger.Warn("wake device dropped: concurrency limit reached",
			zap.String("device_sn", deviceSN))
		return
	}

	go func() {
		defer sem.Release(1)
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
	// T-0157 C5: 原条件 SourceID=="" 跳过排除了普通用户操作 (device.SetParameters 等
	// 不填 SourceID)，导致消息中心 subscriber 永远收不到 task.completed/failed 事件。
	// 新条件: SourceID 或 CreatorID 任一非空都广播——前者承担业务回流 (mml/ops 聚合器),
	// 后者承担消息中心通知。匿名/临时任务两个都空 → 跳过 (不占用广播带宽)。
	if task.SourceID == "" && task.CreatorID == "" {
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

// PublishTransitionEvent publishes the durable transition outbox entry using
// the transition token as a stable event id. Subscriber deduplication therefore
// makes publish-then-ack crash recovery safe.
func (s *TaskService) PublishTransitionEvent(
	ctx context.Context,
	task *Task,
	token string,
) error {
	if task == nil || token == "" {
		return fmt.Errorf("task and transition token are required")
	}
	if task.SourceID == "" && task.CreatorID == "" {
		return s.queue.acknowledgeTransitionEvent(ctx, task.ID, token)
	}
	if s.eventBus != nil {
		subject := SubjectForStatus(task.Status)
		if subject == "" {
			return s.queue.acknowledgeTransitionEvent(ctx, task.ID, token)
		}
		evt, err := event.NewEvent(subject, task)
		if err != nil {
			return fmt.Errorf("build transition event: %w", err)
		}
		evt.ID = "task-transition-" + token
		evt.Metadata = map[string]string{"transition_token": token}
		if err := s.eventBus.Publish(ctx, subject, evt); err != nil {
			return fmt.Errorf("publish transition event: %w", err)
		}
	} else {
		for _, cb := range s.callbacks {
			cb.OnTaskCompleted(ctx, task)
		}
	}
	return s.queue.acknowledgeTransitionEvent(ctx, task.ID, token)
}

// StaleNotificationLookup 实现 notification.StaleTaskLookup 接口（T-0157 stale sync）。
// 把 PgTaskRepository.GetByID 包装成与 notification 包解耦的查询函数，
// 解决"消息中心 stale sync 要反查 task 状态"的跨包依赖（notification → task 单向）。
type StaleNotificationLookup struct {
	repo *PgTaskRepository
}

func NewStaleNotificationLookup(repo *PgTaskRepository) *StaleNotificationLookup {
	return &StaleNotificationLookup{repo: repo}
}

// LookupTaskStatus 返回 task 的当前 status / error message。
// found=false 表示 task 已不存在（被 PurgeOldTasks 清理 / 测试数据被重置等）。
func (l *StaleNotificationLookup) LookupTaskStatus(ctx context.Context, taskID string) (string, string, bool, error) {
	t, err := l.repo.GetByID(ctx, taskID)
	if err != nil {
		return "", "", false, err
	}
	if t == nil {
		return "", "", false, nil
	}
	return string(t.Status), t.ErrorMessage, true, nil
}

// TaskStatusInfo 是 stale sync 批量反查的去包装结果（只含 status / error message）。
// 定义在 task 包：notification 已单向依赖 task，可直接引用；反向（task→notification）禁止，
// 故不能把该类型放 notification 包，否则 LookupTaskStatuses 适配器会引入 import cycle。
type TaskStatusInfo struct {
	Status   string
	ErrorMsg string
}

// LookupTaskStatuses 批量反查多个 task 的状态（#16 消除 stale sync N+1）。
// 一条 IN 查询替代逐 ID 的 LookupTaskStatus；返回 map 只含存在的 taskID，
// 缺失的 taskID（已被清理）由调用方按 not-found 处理。
func (l *StaleNotificationLookup) LookupTaskStatuses(ctx context.Context, taskIDs []string) (map[string]TaskStatusInfo, error) {
	rows, err := l.repo.LookupStatusesByIDs(ctx, taskIDs)
	if err != nil {
		return nil, err
	}
	out := make(map[string]TaskStatusInfo, len(rows))
	for id, row := range rows {
		out[id] = TaskStatusInfo{
			Status:   string(row.Status),
			ErrorMsg: row.ErrorMessage,
		}
	}
	return out, nil
}

// SubjectForStatus maps a terminal TaskStatus to its event subject.
// Returns "" for non-terminal statuses so callers can skip publishing.
func SubjectForStatus(status TaskStatus) string {
	switch status {
	case TaskStatusCompleted:
		return event.SubjectTaskCompleted
	case TaskStatusFailed, TaskStatusExpired:
		return event.SubjectTaskFailed
	case TaskStatusCancelled:
		return event.SubjectTaskCancelled
	default:
		return ""
	}
}

// TaskHistoryOptions 任务历史查询选项（定义在 model.go）

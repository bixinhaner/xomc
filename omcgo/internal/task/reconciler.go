package task

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Reconciler 周期对账 Redis（运行时真相源）与 PostgreSQL（持久化权威源）的任务状态分叉（#13）。
//
// 背景：统一任务队列对 Redis + PG 双写，但两次写之间无事务/补偿安全：
//   - MarkTaskCompleted/Failed/Sent 先写 Redis（ACS 会话内的活真相源）再 sync PG；
//     若 PG.Update 失败（连接抖动 / 超时），错误被 service 层降级为 warn 吞掉 →
//     PG 永远停在 pending/sent，而 Redis 已是终态。Redis 24h TTL 过期后，PG 行
//     成为永久陈旧的孤儿记录（派发/完成路由据 PG 判活会错乱）。
//   - CreateTask 入队失败回滚 PG.Delete 也可能失败，留下 pending 孤儿（由
//     RestorePendingQueues / ExpiredSweeper 兜底，不在本对账器职责内）。
//
// 务实方案（非跨存储分布式事务）：PG 为权威源，Redis 为最佳努力 + 可对账重建。
// Reconciler 扫描 PG 活跃态(pending/sent)任务，读 Redis 真相：
//   - Redis 持有该任务的终态 → PG 滞后 → 把 PG 同步到终态（修复分叉）。
//   - Redis 无记录 / 仍活跃 → 无须修复（孤儿 pending 由 Restore/Sweeper 处理）。
//
// 设计沿用 ExpiredSweeper 范式：
//   - 小接口注入（ActiveTaskLister + QueueStateReader + TaskRepairer），便于单测脱离 PG/Redis
//   - ReconcileOnce 暴露同步单轮接口供单测直接断言；Run 包装 Ticker 循环
//   - 单条修复失败仅记 warn + metric，不中断整轮；仅 lister 报错才是致命错误
//   - grace 宽限期避开在飞双写，batchSize 防 worker 长查询
type Reconciler struct {
	lister   ActiveTaskLister
	reader   QueueStateReader
	repairer TaskRepairer
	metrics  *TaskMetrics

	interval          time.Duration
	grace             time.Duration
	batchSize         int
	logger            *zap.Logger
	publishTransition func(context.Context, *Task, string) error

	scanMu sync.Mutex
	cursor *ActiveTaskCursor
}

// ActiveTaskLister 抽象 repo.ListActiveTasks，便于单测。
type ActiveTaskLister interface {
	ListActiveTasks(ctx context.Context, olderThan time.Time, limit int) ([]*Task, error)
}

type activeTaskPageLister interface {
	ListActiveTasksAfter(
		ctx context.Context,
		olderThan time.Time,
		after *ActiveTaskCursor,
		limit int,
	) ([]*Task, error)
}

// QueueStateReader 抽象 queue.GetByID（读 Redis 真相），便于单测。
// 返回 nil, nil 表示 Redis 中无该任务（TTL 过期 / 从未入队）。
type QueueStateReader interface {
	GetByID(ctx context.Context, taskID string) (*Task, error)
}

// TaskRepairer 抽象 repo.Update（把修复后的终态写回 PG），便于单测。
type TaskRepairer interface {
	Update(ctx context.Context, task *Task) error
}

type conditionalTaskRepairer interface {
	TransitionIfStatus(ctx context.Context, task *Task, from TaskStatus) (bool, error)
	GetByID(ctx context.Context, id string) (*Task, error)
}

type partitionedTaskGetter interface {
	GetByDeviceAndID(ctx context.Context, deviceSN, id string) (*Task, error)
}

type pendingTransitionStore interface {
	listPendingTransitions(ctx context.Context, limit int64) ([]pendingTransitionRef, error)
	loadPendingTransition(ctx context.Context, taskID, token string) (*preparedTaskTransition, error)
	acknowledgeTransition(ctx context.Context, taskID, token string) error
	deferPendingTransition(ctx context.Context, taskID, token string) error
	removePendingTransition(ctx context.Context, taskID, token string) error
	resolvePendingTransition(ctx context.Context, durable *Task, token string) error
	rollbackSentTransition(ctx context.Context, pending *Task, oldCWMPID, token string) error
}

func (r *Reconciler) WithTransitionPublisher(
	publisher func(context.Context, *Task, string) error,
) *Reconciler {
	r.publishTransition = publisher
	return r
}

// ReconcileStats 汇报一轮对账的处理结果。
type ReconcileStats struct {
	Scanned      int // 从 PG 查出的活跃态任务数
	Repaired     int // PG 滞后于 Redis 终态、已成功同步到终态的任务数
	RepairFailed int // 检出分叉但 PG 回写失败的任务数（下一轮重试）
}

// isTerminal 报告状态是否为终态（不会再变）。
func isTerminal(s TaskStatus) bool {
	switch s {
	case TaskStatusCompleted, TaskStatusFailed, TaskStatusExpired, TaskStatusCancelled:
		return true
	default:
		return false
	}
}

// NewReconciler 创建 Reconciler。
//
// interval/grace/batchSize <= 0 时使用安全默认值（30s / 60s / 100）。
// grace 必须 >= 一次正常双写 sync 的耗时上限，避免把在飞任务误判为分叉。
// logger 为 nil 时使用 zap.NewNop()。metrics 可为 nil（单测）。
func NewReconciler(
	lister ActiveTaskLister,
	reader QueueStateReader,
	repairer TaskRepairer,
	metrics *TaskMetrics,
	interval, grace time.Duration,
	batchSize int,
	logger *zap.Logger,
) *Reconciler {
	if interval <= 0 {
		interval = 30 * time.Second
	}
	if grace <= 0 {
		grace = 60 * time.Second
	}
	if batchSize <= 0 {
		batchSize = 100
	}
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Reconciler{
		lister:    lister,
		reader:    reader,
		repairer:  repairer,
		metrics:   metrics,
		interval:  interval,
		grace:     grace,
		batchSize: batchSize,
		logger:    logger.Named("task-reconciler"),
	}
}

// Run 阻塞循环执行 ReconcileOnce 直到 ctx.Done()。
// 首轮在第一个 interval tick 触发（避开启动期与其它初始化抢资源）。
func (r *Reconciler) Run(ctx context.Context) {
	r.logger.Info("task reconciler running",
		zap.Duration("interval", r.interval),
		zap.Duration("grace", r.grace),
		zap.Int("batch_size", r.batchSize))

	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			r.logger.Info("task reconciler stopped", zap.Error(ctx.Err()))
			return
		case <-ticker.C:
			if _, err := r.ReconcileOnce(ctx); err != nil {
				r.logger.Warn("reconcile round failed", zap.Error(err))
			}
		}
	}
}

// ReconcileOnce 跑一轮对账，返回处理统计 + 第一个致命错误。
// 仅 lister 报错才是致命错误；单条任务的读 Redis / 回写 PG 失败仅记 warn + metric，继续下一条。
func (r *Reconciler) ReconcileOnce(ctx context.Context) (ReconcileStats, error) {
	pendingStats := r.reconcilePendingTransitions(ctx)
	olderThan := time.Now().Add(-r.grace)
	tasks, err := r.listActivePage(ctx, olderThan)
	if err != nil {
		return pendingStats, err
	}

	stats := pendingStats
	stats.Scanned += len(tasks)
	staleDetected := 0
	// issue #20：本轮观测到的活跃任务积压绝对快照（不漂移的权威背压信号）。
	// 注意：受 grace + batchSize 限制，这是"超过 grace 仍活跃且本批可见"的下界，
	// 足以驱动"积压持续走高"告警；精确全量统计成本更高，按需再加。
	if r.metrics != nil {
		r.metrics.BacklogTotal.Set(float64(len(tasks)))
		r.metrics.RedisPGDiff.Set(0)
	}
	for _, t := range tasks {
		if t == nil {
			continue
		}

		// 读 Redis 真相。读失败不致命：Redis 不可用时本轮跳过，待恢复后再对账。
		live, err := r.reader.GetByID(ctx, t.ID)
		if err != nil {
			r.logger.Warn("reconcile: read redis state failed",
				zap.String("task_id", t.ID),
				zap.String("device_sn", t.DeviceSN),
				zap.Error(err))
			continue
		}

		// Redis 无记录（TTL 过期或从未入队），或仍非终态 → 无分叉可修。
		// 孤儿 pending 由 RestorePendingQueues / ExpiredSweeper 兜底，不在此职责内。
		if live == nil || !isTerminal(live.Status) {
			continue
		}

		// 分叉确认：PG 活跃态 vs Redis 终态。先按"检出即计"记 stale 指标
		// （即便随后修复失败也已计入，反映分叉发生频率本身）。
		if r.metrics != nil {
			r.metrics.StaleDetectedTotal.Inc()
			staleDetected++
			r.metrics.RedisPGDiff.Set(float64(staleDetected))
		}

		// 把 Redis 终态同步回 PG。复制终态字段到 PG 侧对象后 Update ——
		// 直接用 live（已是终态完整对象）回写。
		var repairErr error
		if conditional, ok := r.repairer.(conditionalTaskRepairer); ok {
			var changed bool
			changed, repairErr = conditional.TransitionIfStatus(ctx, live, t.Status)
			if repairErr == nil && !changed {
				continue
			}
		} else {
			repairErr = r.repairer.Update(ctx, live)
		}
		if repairErr != nil {
			stats.RepairFailed++
			if r.metrics != nil {
				r.metrics.ReconcileTotal.WithLabelValues("repair_failed").Inc()
			}
			r.logger.Warn("reconcile: repair pg state failed",
				zap.String("task_id", t.ID),
				zap.String("device_sn", t.DeviceSN),
				zap.String("pg_status", string(t.Status)),
				zap.String("redis_status", string(live.Status)),
				zap.Error(repairErr))
			continue
		}

		stats.Repaired++
		if r.metrics != nil {
			r.metrics.ReconcileTotal.WithLabelValues("repaired").Inc()
			r.metrics.RecoveryActionTotal.WithLabelValues(RecoveryActionReconcileRepair).Inc()
		}
		r.logger.Info("reconcile: repaired pg state divergence",
			zap.String("task_id", t.ID),
			zap.String("device_sn", t.DeviceSN),
			zap.String("from_status", string(t.Status)),
			zap.String("to_status", string(live.Status)))
	}

	if stats.Repaired > 0 || stats.RepairFailed > 0 {
		r.logger.Info("task reconcile round done",
			zap.Int("scanned", stats.Scanned),
			zap.Int("repaired", stats.Repaired),
			zap.Int("repair_failed", stats.RepairFailed))
	}
	return stats, nil
}

func (r *Reconciler) listActivePage(ctx context.Context, olderThan time.Time) ([]*Task, error) {
	paged, ok := r.lister.(activeTaskPageLister)
	if !ok {
		return r.lister.ListActiveTasks(ctx, olderThan, r.batchSize)
	}

	r.scanMu.Lock()
	defer r.scanMu.Unlock()

	tasks, err := paged.ListActiveTasksAfter(ctx, olderThan, r.cursor, r.batchSize)
	if err != nil {
		return nil, err
	}
	if len(tasks) == 0 {
		r.cursor = nil
		return tasks, nil
	}
	last := tasks[len(tasks)-1]
	r.cursor = &ActiveTaskCursor{CreatedAt: last.CreatedAt, ID: last.ID}
	return tasks, nil
}

// reconcilePendingTransitions consumes the durable Redis compensation index
// before the legacy PG-active scan. Failed entries are moved to the tail so a
// long PG outage for one task cannot starve newer transitions in a fixed batch.
func (r *Reconciler) reconcilePendingTransitions(ctx context.Context) ReconcileStats {
	store, ok := r.reader.(pendingTransitionStore)
	if !ok {
		return ReconcileStats{}
	}
	repairer, ok := r.repairer.(conditionalTaskRepairer)
	if !ok {
		return ReconcileStats{}
	}
	refs, err := store.listPendingTransitions(ctx, int64(r.batchSize))
	if err != nil {
		r.logger.Warn("reconcile: list pending transitions failed", zap.Error(err))
		return ReconcileStats{RepairFailed: 1}
	}
	stats := ReconcileStats{Scanned: len(refs)}
	for _, ref := range refs {
		id, token := ref.TaskID, ref.Token
		prepared, err := store.loadPendingTransition(ctx, id, token)
		if err != nil {
			stats.RepairFailed++
			_ = store.deferPendingTransition(ctx, id, token)
			continue
		}
		if prepared == nil || prepared.Task == nil {
			// Hash was explicitly deleted after the index write. Remove the
			// stale index member; normal transitions are PERSISTed and cannot
			// disappear because of TTL while PG is unavailable.
			_ = store.removePendingTransition(ctx, id, token)
			continue
		}
		if prepared.Task.Status == TaskStatusSent {
			durable, loadErr := loadDurableTaskForRepair(ctx, repairer, prepared.Task, id)
			if loadErr != nil {
				stats.RepairFailed++
				_ = store.deferPendingTransition(ctx, id, token)
				continue
			}
			if durable != nil && durable.Status == TaskStatusPending {
				if err := store.rollbackSentTransition(
					ctx, durable, prepared.Task.CWMPID, token,
				); err != nil {
					stats.RepairFailed++
					_ = store.deferPendingTransition(ctx, id, token)
					continue
				}
				stats.Repaired++
				continue
			}
		}
		if prepared.PGSyncPending {
			changed, transitionErr := repairer.TransitionIfStatus(
				ctx, prepared.Task, prepared.From,
			)
			if transitionErr != nil {
				stats.RepairFailed++
				_ = store.deferPendingTransition(ctx, id, token)
				continue
			}
			if !changed {
				durable, loadErr := loadDurableTaskForRepair(ctx, repairer, prepared.Task, id)
				if loadErr != nil || durable == nil {
					stats.RepairFailed++
					_ = store.deferPendingTransition(ctx, id, token)
					continue
				}
				if durable.Status != prepared.Task.Status {
					if !isTerminal(durable.Status) ||
						store.resolvePendingTransition(ctx, durable, token) != nil {
						stats.RepairFailed++
						_ = store.deferPendingTransition(ctx, id, token)
						continue
					}
					prepared.Task = durable
				}
			}
		}
		if err := store.acknowledgeTransition(ctx, id, token); err != nil {
			stats.RepairFailed++
			_ = store.deferPendingTransition(ctx, id, token)
			continue
		}
		if prepared.EventPending {
			if r.publishTransition == nil ||
				r.publishTransition(ctx, prepared.Task, token) != nil {
				stats.RepairFailed++
				_ = store.deferPendingTransition(ctx, id, token)
				continue
			}
		}
		stats.Repaired++
	}
	return stats
}

func loadDurableTaskForRepair(
	ctx context.Context,
	repairer conditionalTaskRepairer,
	prepared *Task,
	id string,
) (*Task, error) {
	if getter, ok := repairer.(partitionedTaskGetter); ok && prepared != nil && prepared.DeviceSN != "" {
		return getter.GetByDeviceAndID(ctx, prepared.DeviceSN, id)
	}
	return repairer.GetByID(ctx, id)
}

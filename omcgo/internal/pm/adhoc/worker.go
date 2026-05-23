package adhoc

import (
	"context"
	"errors"
	"time"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

// Worker 是 G7 adhoc 任务消费者。
//
// 工作循环：
//  1. tick (默认 3s)：LockNextPending 抢一个 pending 任务
//  2. UpdateStatus(running) → executor.ExecuteOneshot → 终态切换：
//     - oneshot 成功 → status=succeeded
//     - continuous 成功 → status=scheduled（等下次 cron 触发）
//     - 任何失败 → status=failed
//  3. PublishCompleted 事件（SSE handler 用）
//
// 多实例：worker 池里 N 个 Worker goroutine 并行，LockNextPending 的 SKIP LOCKED
// 保证一个任务只被一个 worker 抢。跨进程多 worker 同样有效。
//
// continuous 模式的 scheduled→pending 切换由 ContinuousScheduler 单独驱动。
type Worker struct {
	repo      Repository
	executor  *Executor
	lockOwner string
	tickEvery time.Duration
	logger    *zap.Logger
}

// NewWorker 构造 Worker。tickEvery <= 0 走默认 3s。
func NewWorker(repo Repository, executor *Executor, lockOwner string, tickEvery time.Duration, logger *zap.Logger) *Worker {
	if tickEvery <= 0 {
		tickEvery = 3 * time.Second
	}
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Worker{
		repo:      repo,
		executor:  executor,
		lockOwner: lockOwner,
		tickEvery: tickEvery,
		logger:    logger.Named("pm.adhoc.worker"),
	}
}

// Run 阻塞循环。ctx 取消后返回。
func (w *Worker) Run(ctx context.Context) {
	ticker := time.NewTicker(w.tickEvery)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// 一次 tick 排空（多个 pending 时连续抢）
			for {
				ran := w.tryRunOne(ctx)
				if !ran {
					break
				}
			}
		}
	}
}

// tryRunOne 抢一个 pending 跑一遍。返 true 表示跑了一个（可能成功也可能失败）。
func (w *Worker) tryRunOne(ctx context.Context) bool {
	task, err := w.repo.LockNextPending(ctx, w.lockOwner)
	if err != nil {
		if errors.Is(err, ErrNoPendingTask) {
			return false
		}
		w.logger.Warn("lock next pending failed", zap.Error(err))
		return false
	}
	w.runOne(ctx, task)
	return true
}

// runOne 执行单个任务。executor 内部已上报中间 progress；本函数只负责终态切换 + completed 事件。
func (w *Worker) runOne(ctx context.Context, task *Task) {
	w.logger.Info("adhoc task started",
		zap.String("task_id", task.ID.String()),
		zap.String("mode", string(task.Mode)),
		zap.Int("device_count", len(task.DeviceSNs)),
		zap.Int("granularity_count", len(task.Granularities)),
	)

	rowsTotal, err := w.executor.ExecuteOneshot(ctx, task)

	finalStatus := w.terminalStatus(task, err)
	errMsg := ""
	if err != nil {
		errMsg = err.Error()
	}

	// 切终态时不动 progress（executor 内已设到 100）
	if updateErr := w.repo.UpdateStatus(context.Background(), task.ID, finalStatus, nil, errMsg); updateErr != nil {
		w.logger.Error("update terminal status failed",
			zap.String("task_id", task.ID.String()),
			zap.String("status", string(finalStatus)),
			zap.Error(updateErr))
	}

	w.executor.PublishCompleted(context.Background(), task.ID, finalStatus, rowsTotal, errMsg)
	w.logger.Info("adhoc task finished",
		zap.String("task_id", task.ID.String()),
		zap.String("status", string(finalStatus)),
		zap.Int("rows_total", rowsTotal),
		zap.Error(err))
}

// terminalStatus 按 mode + error 决定终态。
func (w *Worker) terminalStatus(task *Task, err error) Status {
	if err != nil {
		return StatusFailed
	}
	if task.Mode == ModeContinuous {
		return StatusScheduled
	}
	return StatusSucceeded
}

// ── ContinuousScheduler ──────────────────────────────────────────────────

// ContinuousScheduler 周期性扫描 status=scheduled 的 continuous 任务，
// 按 cron_expr 评估"上次跑 + cron interval"是否到达，到达则切回 pending
// 让 worker 重新抢。
//
// 简化设计（vs 每任务一 cron entry）：单 ticker 每分钟扫一次，per-task evaluate。
// 多 worker 实例都跑此扫描器无害（同 task 多 worker 同时更新 scheduled→pending
// 是幂等的 — UPDATE WHERE status='scheduled' 只匹配一次）。
type ContinuousScheduler struct {
	repo     ContinuousRepository // 缩小依赖契约，便于单测
	tickEvery time.Duration
	parser    cron.Parser
	logger    *zap.Logger
}

// ContinuousRepository 是 ContinuousScheduler 所需的最小契约。
//
// 真实实现由本包外的 SQL 提供（worker main 装配时用 pgxpool 直接实现）。
type ContinuousRepository interface {
	// ListReschedulable 返 status=scheduled 且 mode=continuous 的 task。
	// 每行的 LastFireAt 为"上次 cron 触发时刻"——优先 last_fire_at 列，
	// 为 NULL 时 fallback 到 created_at。Worker 跑完任务后不更新 last_fire_at，
	// 由本 Scheduler 在 MarkPending 时显式推进。
	ListReschedulable(ctx context.Context) ([]ContinuousTask, error)
	// MarkPending 单条原子切 scheduled→pending（CAS），同时把 last_fire_at 推到 fireAt。
	// fireAt 表示本次 cron 触发对应的窗口时间（cron.Next(prevLastFireAt) 算出）。
	// 推一格而非直接到 NOW，让长时间停机后启动能逐格补齐漏桶（"lossless catchup"，G7-Gap-9）。
	MarkPending(ctx context.Context, id ContinuousTaskID, fireAt time.Time) error
}

// ContinuousTaskID 是 marker；具体类型由实现决定（避免 uuid 包污染 contract）。
type ContinuousTaskID = string

// ContinuousTask 是 ContinuousScheduler 看到的最小任务投影。
type ContinuousTask struct {
	ID         ContinuousTaskID // string 版本的 task UUID
	CronExpr   string
	LastFireAt time.Time // 上次 cron 触发时刻（last_fire_at 优先；NULL fallback 到 created_at）
}

// NewContinuousScheduler 构造调度器。tickEvery <= 0 走默认 60s。
func NewContinuousScheduler(repo ContinuousRepository, tickEvery time.Duration, logger *zap.Logger) *ContinuousScheduler {
	if tickEvery <= 0 {
		tickEvery = 60 * time.Second
	}
	if logger == nil {
		logger = zap.NewNop()
	}
	// 用标准 5 字段 cron parser（与 G5 cron + backup/period_scheduler 一致）
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	return &ContinuousScheduler{
		repo:      repo,
		tickEvery: tickEvery,
		parser:    parser,
		logger:    logger.Named("pm.adhoc.continuous-scheduler"),
	}
}

// Run 阻塞循环。
func (s *ContinuousScheduler) Run(ctx context.Context) {
	ticker := time.NewTicker(s.tickEvery)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			s.sweepOnce(ctx, now)
		}
	}
}

// sweepOnce 单轮扫描。暴露便于单测。
//
// G7-Gap-9 lossless catchup：last_fire_at 与 status 解耦，调度器每次只推进一个 cron 窗口。
// 长时间停机场景下，每个 sweep cycle + 一次 worker 跑完，会前进一个漏桶；
// 直至 last_fire_at 追平 NOW()，恢复正常 cadence。
// 单轮内 status=scheduled→pending 只能切一次（CAS），所以本函数不在循环内对同一 task 反复 MarkPending —
// 推进 1 格即可，下一 tick 会再来。
func (s *ContinuousScheduler) sweepOnce(ctx context.Context, now time.Time) {
	tasks, err := s.repo.ListReschedulable(ctx)
	if err != nil {
		s.logger.Warn("list reschedulable tasks failed", zap.Error(err))
		return
	}
	for _, t := range tasks {
		sched, err := s.parser.Parse(t.CronExpr)
		if err != nil {
			s.logger.Warn("invalid cron expr; skip",
				zap.String("task_id", t.ID), zap.String("cron", t.CronExpr), zap.Error(err))
			continue
		}
		// 从 LastFireAt 之后下一个 cron 触发时刻
		next := sched.Next(t.LastFireAt)
		if next.IsZero() || next.After(now) {
			continue
		}
		// 统计本次需要追多少格（仅 log 用，实际 MarkPending 只推 1 格）
		missed := 0
		for cur := next; !cur.IsZero() && !cur.After(now); cur = sched.Next(cur) {
			missed++
			if missed > 10000 {
				// 极端防御：cron 表达式异常导致无限循环
				break
			}
		}
		if err := s.repo.MarkPending(ctx, t.ID, next); err != nil {
			s.logger.Warn("mark pending failed",
				zap.String("task_id", t.ID), zap.Error(err))
			continue
		}
		if missed > 1 {
			s.logger.Warn("continuous task catching up missed windows",
				zap.String("task_id", t.ID),
				zap.String("cron", t.CronExpr),
				zap.Int("missed", missed),
				zap.Time("fire_at", next))
		} else {
			s.logger.Info("continuous task rescheduled",
				zap.String("task_id", t.ID),
				zap.String("cron", t.CronExpr),
				zap.Time("fire_at", next))
		}
	}
}

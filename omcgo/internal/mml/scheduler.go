package mml

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// -----------------------------------------------------------------------------
// Scheduler —— MML 任务调度器
// docs/design/mml-task-flow-design-20260424.md P2 / P3。
//
// 职责：
//   · 定时（每 30s）扫描 mml_tasks.next_trigger_at <= NOW() 且 status='pending'
//     的行，按 execute_type 做不同处理：
//       - scheduled：call StartTask（复用现有路径）+ 清空 next_trigger_at
//       - periodic：P3 克隆子实例 + 推进模板的 next_trigger_at 到下一次命中
//   · 启动时一次性扫描 scheduled_at < NOW() 的遗留任务，补触发（Q2）
//   · 多副本部署下通过 PG FOR UPDATE SKIP LOCKED 保证同一行不会被两副本同时占用
//
// 组装：cmd/app/provider/modules.go 装配 Scheduler 并调用 Start(ctx)；
// Shutdown 时调用 Stop 让 cron 进程干净退出。
// -----------------------------------------------------------------------------

// SchedulerClock 允许单测注入模拟时钟；默认用真实时间。
type SchedulerClock interface {
	Now() time.Time
}

type systemClock struct{}

func (systemClock) Now() time.Time { return time.Now() }

// Scheduler 每 30 秒扫一次 PG，把到期的 scheduled/periodic 任务拉起。
type Scheduler struct {
	service  *Service
	taskRepo ScheduledTaskRepository
	clock    SchedulerClock
	interval time.Duration
	batch    int
	logger   *zap.Logger

	stopCh chan struct{}
	doneCh chan struct{}
}

// ScheduledTaskRepository 是 Scheduler 需要的 repo 能力子集。定义成独立接口
// 便于单测注入 mock；生产侧由 PgTaskRepository 扩展实现。
type ScheduledTaskRepository interface {
	// ClaimDueTasks 原子性地认领到期任务：在事务内 SELECT FOR UPDATE SKIP LOCKED
	// 并把命中行的 status 置 running（scheduled）或保留 pending（periodic）+
	// 清空/推进 next_trigger_at。返回被认领的任务快照。
	ClaimDueTasks(ctx context.Context, now time.Time, limit int) ([]*MMLTask, error)
	// RecordPeriodicChild 在同一事务里：插入子实例 + 推进模板 next_trigger_at。
	// P3 使用，P2 阶段 ClaimDueTasks 跳过 periodic，本方法可留空实现。
	RecordPeriodicChild(ctx context.Context, parent, child *MMLTask, parentNext *time.Time) error
	// FinalizePeriodicParent 当 period_end < now 时把模板置 completed，
	// 清空 next_trigger_at。
	FinalizePeriodicParent(ctx context.Context, id uuid.UUID, finishedAt time.Time) error
}

// NewScheduler 构造 Scheduler。interval<=0 时使用默认 30s。
func NewScheduler(
	svc *Service,
	repo ScheduledTaskRepository,
	clock SchedulerClock,
	interval time.Duration,
	logger *zap.Logger,
) *Scheduler {
	if interval <= 0 {
		interval = 30 * time.Second
	}
	if clock == nil {
		clock = systemClock{}
	}
	return &Scheduler{
		service:  svc,
		taskRepo: repo,
		clock:    clock,
		interval: interval,
		batch:    100,
		logger:   logger.Named("mml-scheduler"),
		stopCh:   make(chan struct{}),
		doneCh:   make(chan struct{}),
	}
}

// Start 启动后台轮询；先做一次启动补触发（catch-up），然后进入 ticker 循环。
// 返回后后台 goroutine 继续运行，直到 Stop 被调用。
func (s *Scheduler) Start(ctx context.Context) {
	s.logger.Info("scheduler starting",
		zap.Duration("interval", s.interval),
		zap.Int("batch", s.batch))

	// 启动补触发：把 scheduled_at 已过但仍 pending 的任务一次性认领掉。
	s.runOnce(ctx)

	go func() {
		defer close(s.doneCh)
		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				s.logger.Info("scheduler context cancelled")
				return
			case <-s.stopCh:
				s.logger.Info("scheduler stopped")
				return
			case <-ticker.C:
				s.runOnce(ctx)
			}
		}
	}()
}

// Stop 请求 Scheduler 停止；调用方应 block 在 Wait 上直到后台 goroutine 退出。
func (s *Scheduler) Stop() {
	select {
	case <-s.stopCh:
		// already closed
	default:
		close(s.stopCh)
	}
}

// Wait 阻塞直到后台 goroutine 退出；适合在 main 的 shutdown 阶段调用。
func (s *Scheduler) Wait() { <-s.doneCh }

// runOnce 执行一次认领 + 派发。错误仅记日志，不终止调度。
func (s *Scheduler) runOnce(ctx context.Context) {
	now := s.clock.Now()
	tasks, err := s.taskRepo.ClaimDueTasks(ctx, now, s.batch)
	if err != nil {
		s.logger.Error("claim due tasks", zap.Error(err))
		return
	}
	if len(tasks) == 0 {
		return
	}
	s.logger.Info("scheduler claimed tasks", zap.Int("count", len(tasks)))

	for _, task := range tasks {
		s.dispatch(ctx, task, now)
	}
}

// dispatch 按 execute_type 分支处理已被 Claim 的任务。
func (s *Scheduler) dispatch(ctx context.Context, task *MMLTask, now time.Time) {
	// Imported scripts are immutable snapshots, but device availability and
	// command visibility can change between creation and a scheduled trigger.
	// Re-run the dynamic validator before fanout. Blocking errors transition the
	// claimed instance to failed so it cannot silently remain pending/running.
	if result, err := s.service.preflightTask(ctx, task); err != nil {
		s.logger.Error("preflight scheduled mml task", zap.String("task_id", task.ID.String()), zap.Error(err))
		_ = s.service.failPreflightTask(ctx, task, &ScriptValidationResult{Issues: []ScriptIssue{{Code: "MML_EXECUTION_PREFLIGHT_FAILED", Severity: IssueError, Message: err.Error()}}}, now)
		return
	} else if result != nil && (result.Summary.ErrorCount > 0 || hasScriptErrors(result.Issues)) {
		s.logger.Warn("scheduled mml task blocked by preflight", zap.String("task_id", task.ID.String()), zap.Int("issue_count", len(result.Issues)))
		if err := s.service.failPreflightTask(ctx, task, result, now); err != nil {
			s.logger.Error("persist preflight failure", zap.String("task_id", task.ID.String()), zap.Error(err))
		}
		return
	}

	switch task.ExecuteType {
	case ExecuteScheduled:
		// Claim 阶段已经把 status 改为 running + 清空 next_trigger_at。
		// 这里仅负责 fanout；复用 Service.fanouter（通过导出的轻量方法）。
		if err := s.service.fanoutClaimed(ctx, task); err != nil {
			s.logger.Error("fanout scheduled task",
				zap.String("task_id", task.ID.String()),
				zap.Error(err))
		}
	case ExecutePeriodic:
		// P3 路径：Claim 阶段已把 parent 的 next_trigger_at 推进 + 插入 child。
		// 此处对 child（clone 后的新任务）做 fanout。
		if err := s.service.fanoutClaimed(ctx, task); err != nil {
			s.logger.Error("fanout periodic child",
				zap.String("child_task_id", task.ID.String()),
				zap.Error(err))
		}
	case ExecuteImmediate:
		if task.PeriodicParentID == nil {
			s.logger.Warn("scheduler got unexpected execute_type",
				zap.String("task_id", task.ID.String()),
				zap.String("execute_type", string(task.ExecuteType)))
			break
		}
		// PgTaskRepository.cloneAsPeriodicChild 会把周期子实例降为 immediate，
		// 表示子任务本身无需再被调度，只需立即 fanout。
		if err := s.service.fanoutClaimed(ctx, task); err != nil {
			s.logger.Error("fanout periodic child",
				zap.String("child_task_id", task.ID.String()),
				zap.String("parent_task_id", task.PeriodicParentID.String()),
				zap.Error(err))
		}
	default:
		s.logger.Warn("scheduler got unexpected execute_type",
			zap.String("task_id", task.ID.String()),
			zap.String("execute_type", string(task.ExecuteType)))
	}

	// 若 periodic 模板的 period_end 已过，收敛成 completed。
	// 注意：此时 task 可能是 child（不是模板），所以用 PeriodicParentID 指回模板。
	if task.PeriodicParentID != nil {
		s.maybeFinalizeParent(ctx, *task.PeriodicParentID, now)
	}
}

// maybeFinalizeParent 查询父（模板）任务，若 period_end 已过则置 completed。
// 实现放在 repo 里做单次 UPDATE；本方法仅做条件判断 + 调用。
func (s *Scheduler) maybeFinalizeParent(ctx context.Context, parentID uuid.UUID, now time.Time) {
	if err := s.taskRepo.FinalizePeriodicParent(ctx, parentID, now); err != nil {
		s.logger.Warn("finalize periodic parent (best-effort)",
			zap.String("parent_id", parentID.String()),
			zap.Error(err))
	}
}

// -----------------------------------------------------------------------------
// computeNextPeriodicTrigger —— 根据 period_start / period_end / period_time
// 推算下一次命中时刻。
//
// period_time 格式 "HH:MM:SS"；计算规则：
//   1. baseDay := max(task.PeriodStart 日期, 当前日期)
//   2. 取 baseDay 的 period_time 点；若已过当前时刻，顺延一天
//   3. 若 > PeriodEnd，返回 nil（表示无下次触发）
//
// 时区：统一使用 task 原字段携带的 Location（若均为 UTC 则全链路 UTC）。
// -----------------------------------------------------------------------------

func computeNextPeriodicTrigger(task *MMLTask, now time.Time) *time.Time {
	if task == nil || task.PeriodTime == "" {
		return nil
	}
	hh, mm, ss, err := parsePeriodTime(task.PeriodTime)
	if err != nil {
		return nil
	}
	loc := now.Location()
	if task.PeriodStart != nil {
		loc = task.PeriodStart.Location()
	}

	baseDay := now.In(loc)
	if task.PeriodStart != nil && task.PeriodStart.After(baseDay) {
		baseDay = task.PeriodStart.In(loc)
	}

	candidate := time.Date(baseDay.Year(), baseDay.Month(), baseDay.Day(), hh, mm, ss, 0, loc)
	if !candidate.After(now) {
		candidate = candidate.AddDate(0, 0, 1)
	}

	if task.PeriodEnd != nil && candidate.After(*task.PeriodEnd) {
		return nil
	}
	return &candidate
}

// parsePeriodTime 解析 "HH:MM:SS"，返回 hour/min/sec。
func parsePeriodTime(s string) (int, int, int, error) {
	var hh, mm, ss int
	if _, err := fmt.Sscanf(s, "%d:%d:%d", &hh, &mm, &ss); err != nil {
		return 0, 0, 0, err
	}
	if hh < 0 || hh > 23 || mm < 0 || mm > 59 || ss < 0 || ss > 59 {
		return 0, 0, 0, fmt.Errorf("invalid period_time %q", s)
	}
	return hh, mm, ss, nil
}

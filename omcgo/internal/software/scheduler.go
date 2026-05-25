package software

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// 本文件实现"定时任务调度器"——把 status=pending + create_status=timing + scheduled_at<=now
// 的任务自动推进到执行态。
//
// 设计选择：
//   · 多源轮询：4 张派生表 + 旧表（upgrade_tasks）各有一个 ListDueScheduled 入口；
//     调度器把它们抽象成 ScheduledTaskSource，每个 tick 顺序扫一遍。
//   · 单 goroutine：tick 周期 30s，扫描出的任务在同 goroutine 内串行触发，避免与
//     SoftwareService.startExecution / startCollectExecution 内部的并发池冲突。
//   · 幂等：每条任务在触发前用 UpdateStatus 把 status 从 pending 翻成 in_progress；
//     如返回 ErrNotFound（被其它实例抢到）或 RowsAffected==0（前一 tick 已处理）
//     就跳过，保证多副本部署下也不会重复触发。
//   · 触发分流：
//       - TaskTypeUpgrade / Rollback / Patch / FPGA → SoftwareService.ResumeUpgrade
//       - TaskTypeLogCollect                       → 走外部注入的 collectTrigger 回调
//         （UFTE 知道 transportPath / rpcType / paramPath，调度器自己拿不到）
//     未注入 collectTrigger 时 LogCollect 类任务保持 pending，并打 warn 日志。

// ScheduledTaskSource 抽象一张物理表的"列出到期任务"能力。
// software.PgTaskRepository 和 transfer/repo.PgTaskRepo 都通过 ListDueScheduled 自动满足。
type ScheduledTaskSource interface {
	ListDueScheduled(ctx context.Context, before time.Time, limit int) ([]*UpgradeTask, error)
}

// CollectResumeFn 是 LogCollect 类任务（配置备份 / 运行日志 / 故障日志 / 配置下发）
// 的触发回调。由 UFTE 服务从 catalog 反查 transportPath / rpcType / paramPath 后
// 调用 SoftwareService.ResumeCollect。
//
// 调度器仅传 taskID；taskType / fileType 等信息回调方自取。
type CollectResumeFn func(ctx context.Context, taskID uuid.UUID) error

// TaskScheduler 周期扫描所有 task 源表，触发到期的定时任务。
type TaskScheduler struct {
	svc             *SoftwareService
	sources         []ScheduledTaskSource
	collectTrigger  CollectResumeFn
	interval        time.Duration
	batchPerTickCap int

	logger *zap.Logger

	mu     sync.Mutex
	cancel context.CancelFunc
}

// NewTaskScheduler 用 SoftwareService（触发 Upgrade/Rollback）+ 多个 source（5 张表）
// 构造调度器。interval ≤ 0 时使用默认值 30s。
func NewTaskScheduler(svc *SoftwareService, sources []ScheduledTaskSource, interval time.Duration, logger *zap.Logger) *TaskScheduler {
	if interval <= 0 {
		interval = 30 * time.Second
	}
	return &TaskScheduler{
		svc:             svc,
		sources:         sources,
		interval:        interval,
		batchPerTickCap: 100,
		logger:          logger.Named("task-scheduler"),
	}
}

// SetCollectTrigger 注入 LogCollect 类任务的触发回调。由 UFTE 装配（cmd/app/provider）。
// 不调用 → 该类任务定时模式失效（保持 pending，日志 warn）。
func (s *TaskScheduler) SetCollectTrigger(fn CollectResumeFn) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.collectTrigger = fn
}

// Start 起一个后台 goroutine 周期 tick。重复调用幂等（先 Stop 再起）。
func (s *TaskScheduler) Start(parent context.Context) {
	s.mu.Lock()
	if s.cancel != nil {
		s.cancel()
	}
	ctx, cancel := context.WithCancel(parent)
	s.cancel = cancel
	s.mu.Unlock()

	go func() {
		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()
		// 启动后立即跑一次，避免错过刚好已到期的任务。
		s.tick(ctx)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.tick(ctx)
			}
		}
	}()
	s.logger.Info("task scheduler started",
		zap.Duration("interval", s.interval),
		zap.Int("sources", len(s.sources)))
}

// Stop 结束调度。
func (s *TaskScheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancel != nil {
		s.cancel()
		s.cancel = nil
	}
}

// tick 执行一次扫描 + 触发。panic 全部 recover，不让单条任务把整个循环搞挂。
func (s *TaskScheduler) tick(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			s.logger.Error("task scheduler tick panic", zap.Any("recover", r))
		}
	}()

	now := nowFunc()
	for _, src := range s.sources {
		tasks, err := src.ListDueScheduled(ctx, now, s.batchPerTickCap)
		if err != nil {
			s.logger.Warn("list due scheduled tasks", zap.Error(err))
			continue
		}
		for _, t := range tasks {
			s.fire(ctx, t)
		}
	}
}

// fire 触发一条到期任务。失败仅日志告警，不重抛——下一 tick 还会再扫到。
func (s *TaskScheduler) fire(ctx context.Context, task *UpgradeTask) {
	if task == nil {
		return
	}
	// 在调度器自己的 svc 里没有"按 TaskType 分流到对应 source"的能力——
	// taskRepo 现在挂的是 RoutingTaskRepository，能根据 ID 反查到正确物理表。
	// 这里先尝试把 create_status 从 timing 翻回 active 占位 → 防止其它实例 / 下一 tick
	// 重复抢占。失败说明已被抢，直接 return。
	if err := s.markAsActive(ctx, task.ID); err != nil {
		s.logger.Debug("scheduler skip (already claimed)",
			zap.String("task_id", task.ID.String()), zap.Error(err))
		return
	}

	taskType := task.TaskType
	logFields := []zap.Field{
		zap.String("task_id", task.ID.String()),
		zap.Int("task_type", int(taskType)),
		zap.String("task_name", task.TaskName),
	}
	if task.ScheduledAt != nil {
		logFields = append(logFields, zap.Time("scheduled_at", time.Time(*task.ScheduledAt)))
	}

	switch taskType {
	case TaskTypeUpgrade, TaskTypePatch, TaskTypeFPGA, TaskTypeRollback, TaskTypeReserved:
		if err := s.svc.ResumeUpgrade(ctx, task.ID); err != nil {
			s.logger.Error("scheduler resume upgrade failed", append(logFields, zap.Error(err))...)
			return
		}
		s.logger.Info("scheduler triggered upgrade/rollback task", logFields...)

	case TaskTypeLogCollect:
		s.mu.Lock()
		trigger := s.collectTrigger
		s.mu.Unlock()
		if trigger == nil {
			s.logger.Warn("scheduler reached log-collect task but no trigger installed; task stays paused",
				logFields...)
			// 把 create_status 翻回 timing，下一 tick 再试（等 UFTE 注入 trigger）。
			_ = s.markAsTiming(ctx, task.ID)
			return
		}
		if err := trigger(ctx, task.ID); err != nil {
			s.logger.Error("scheduler collect trigger failed", append(logFields, zap.Error(err))...)
			return
		}
		s.logger.Info("scheduler triggered log-collect task", logFields...)

	default:
		s.logger.Warn("scheduler unknown task type, skipping",
			append(logFields, zap.Int("task_type", int(taskType)))...)
	}
}

// markAsActive 原子地把 create_status: timing → active（仅在 status 仍 pending 时生效），
// 用作"抢占 lock"。成功（RowsAffected=1）即由当前进程负责触发；其他 tick / 实例此后看到
// create_status=active 会自动忽略（partial index 不再命中）。
//
// 这里直连 fallback PgTaskRepository 的 pool 完成 UPDATE。RoutingTaskRepository 没有
// 暴露"按 ID 翻表"的事务方法，但因为 5 张表的 schema 一致，所以我们让调度器跨表
// 尝试 UPDATE—某个表的行存在就生效，其它表 RowsAffected=0，跨表 UPDATE 行为可控。
func (s *TaskScheduler) markAsActive(ctx context.Context, id uuid.UUID) error {
	return s.transitionCreateStatus(ctx, id, CreateStatusTiming, CreateStatusActive)
}

// markAsTiming 恢复回 timing，用于"被调度器拿出来但无 trigger 可用"的回退场景。
func (s *TaskScheduler) markAsTiming(ctx context.Context, id uuid.UUID) error {
	return s.transitionCreateStatus(ctx, id, CreateStatusActive, CreateStatusTiming)
}

// transitionCreateStatus 按 sources 顺序尝试 UPDATE，只要任意一张表命中即成功。
func (s *TaskScheduler) transitionCreateStatus(ctx context.Context, id uuid.UUID, from, to string) error {
	type stmExecutor interface {
		UpdateCreateStatusGuarded(ctx context.Context, id uuid.UUID, from, to string) (bool, error)
	}
	for _, src := range s.sources {
		ex, ok := src.(stmExecutor)
		if !ok {
			continue
		}
		ok2, err := ex.UpdateCreateStatusGuarded(ctx, id, from, to)
		if err != nil {
			return fmt.Errorf("transition create_status: %w", err)
		}
		if ok2 {
			return nil
		}
	}
	return fmt.Errorf("task %s not found in any source for transition %s→%s", id, from, to)
}

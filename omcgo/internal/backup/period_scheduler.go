package backup

// M3: 周期备份调度器 (backup-restore-alignment-plan M3)
//
// PeriodScheduler 在启动时从 DB 加载全部 enabled=true 的 backup_schedules，并
// 通过 robfig/cron/v3 按照各行的 cron_expr 定时调用 ScheduledTaskCreator.CreateTask。
// CreateTask 会向 EventBus 发布 SubjectBackupTaskCreated，由 BackupExecutor 消费执行。
//
// 热更新：schedule CRUD 后调用 Reload(ctx)，调度器会无停机替换内部 cron 实例。
//
// 设计约束：
//   - 依赖接口定义在消费者侧（consumer-side interface），不向现有 ScheduleRepository
//     或 TaskRepository 接口添加方法，避免破坏包内 5 个已有 mock。
//   - SchedulerMetrics 同时服务于 PeriodScheduler 和 TaskReaper（共用同一份注册）。

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/event"
)

// ScheduleLister 是 PeriodScheduler 所需的最小调度仓库契约。
// *PgScheduleRepository 在 pg_repository.go 中实现了此接口（ListEnabled 方法）。
type ScheduleLister interface {
	ListEnabled(ctx context.Context) ([]*BackupSchedule, error)
}

// ScheduledTaskCreator 是 PeriodScheduler 所需的最小任务创建契约。
// *Service 已实现 CreateTask，天然满足此接口。
type ScheduledTaskCreator interface {
	CreateTask(ctx context.Context, task *BackupTask) (*BackupTask, error)
}

// PeriodScheduler 周期备份调度器。
type PeriodScheduler struct {
	scheduleRepo ScheduleLister
	taskCreator  ScheduledTaskCreator
	metrics      *SchedulerMetrics
	logger       *zap.Logger

	mu   sync.Mutex
	cron *cron.Cron
}

// NewPeriodScheduler 构造一个 PeriodScheduler。metrics 可为 nil。
func NewPeriodScheduler(
	scheduleRepo ScheduleLister,
	taskCreator ScheduledTaskCreator,
	metrics *SchedulerMetrics,
	logger *zap.Logger,
) *PeriodScheduler {
	return &PeriodScheduler{
		scheduleRepo: scheduleRepo,
		taskCreator:  taskCreator,
		metrics:      metrics,
		logger:       logger.Named("backup-period-scheduler"),
	}
}

// Start 从 DB 加载调度并启动 cron。进程启动时调用一次。
func (s *PeriodScheduler) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.reload(ctx)
}

// Reload 热更新调度：停止旧 cron，从 DB 重新加载全部 enabled 调度后启动新 cron。
// schedule CRUD 完成后调用此方法，无需重启进程。Goroutine 安全。
func (s *PeriodScheduler) Reload(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.reload(ctx); err != nil {
		s.logger.Warn("period scheduler reload failed", zap.Error(err))
	}
}

// Stop 停止 cron。幂等，可多次调用。
func (s *PeriodScheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cron != nil {
		s.cron.Stop()
		s.cron = nil
	}
}

// SubscribeReload 订阅 SubjectBackupScheduleChanged，收到事件后异步触发 Reload。
// 用于跨进程热更新：app 进程的 Service.{Create,Update,Delete}Schedule 发布事件，
// worker 进程的 PeriodScheduler 收到后无停机重载 cron。bus 为 nil 时静默跳过。
func (s *PeriodScheduler) SubscribeReload(bus event.EventBus) error {
	if bus == nil {
		return nil
	}
	_, err := bus.Subscribe(event.SubjectBackupScheduleChanged,
		func(ctx context.Context, _ event.Event) error {
			reloadCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			s.Reload(reloadCtx)
			return nil
		})
	if err != nil {
		return fmt.Errorf("subscribe schedule.changed: %w", err)
	}
	s.logger.Info("backup period scheduler subscribed to schedule.changed")
	return nil
}

// reload 是 Start/Reload 的共享实现；调用方需持有 mu。
func (s *PeriodScheduler) reload(ctx context.Context) error {
	if s.cron != nil {
		s.cron.Stop()
		s.cron = nil
	}

	schedules, err := s.scheduleRepo.ListEnabled(ctx)
	if err != nil {
		return fmt.Errorf("load enabled backup schedules: %w", err)
	}

	c := cron.New()
	registered := 0
	for _, sched := range schedules {
		sc := sched // 捕获循环变量
		if _, addErr := c.AddFunc(sc.CronExpr, func() {
			fireCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			s.fire(fireCtx, sc)
		}); addErr != nil {
			s.logger.Warn("invalid cron expression; skipping schedule",
				zap.String("schedule_id", sc.ID.String()),
				zap.String("name", sc.Name),
				zap.String("cron_expr", sc.CronExpr),
				zap.Error(addErr))
			continue
		}
		registered++
	}
	c.Start()
	s.cron = c

	s.logger.Info("backup period scheduler reloaded",
		zap.Int("enabled_schedules", len(schedules)),
		zap.Int("registered", registered))
	return nil
}

// fire 在 cron tick 时触发，为指定调度创建一条 backup_task。
func (s *PeriodScheduler) fire(ctx context.Context, sched *BackupSchedule) {
	targetType := "device"
	if sched.TargetType != nil && *sched.TargetType != "" {
		targetType = *sched.TargetType
	}
	taskName := fmt.Sprintf("scheduled_%s", sched.Name)
	task := &BackupTask{
		TaskType:   sched.TaskType,
		TargetType: targetType,
		TargetIDs:  sched.TargetIDs,
		TaskName:   &taskName,
	}

	_, err := s.taskCreator.CreateTask(ctx, task)
	if err != nil {
		s.metrics.RecordScheduleFired("error")
		s.logger.Warn("period scheduler failed to create backup task",
			zap.String("schedule_id", sched.ID.String()),
			zap.String("name", sched.Name),
			zap.Error(err))
		return
	}
	s.metrics.RecordScheduleFired("success")
	s.logger.Info("period scheduler fired backup task",
		zap.String("schedule_id", sched.ID.String()),
		zap.String("name", sched.Name),
		zap.String("task_id", task.ID.String()))
}

// SchedulerMetrics 同时服务于 PeriodScheduler 和 TaskReaper（共用注册，避免重复 MustRegister）。
type SchedulerMetrics struct {
	scheduleFiredTotal *prometheus.CounterVec
	taskReapedTotal    prometheus.Counter
}

// NewSchedulerMetrics 注册并返回调度器+reaper 的 Prometheus 指标。
// reg 为 nil 时（单元测试）仅构造结构体，不注册。所有方法均 nil-safe。
func NewSchedulerMetrics(reg prometheus.Registerer) *SchedulerMetrics {
	m := &SchedulerMetrics{
		scheduleFiredTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "omc_backup_schedule_fired_total",
			Help: "Backup periodic schedule cron fires, labeled by result (success|error).",
		}, []string{"result"}),
		taskReapedTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "omc_backup_task_reaped_total",
			Help: "Backup pending tasks republished by the task reaper for event-loss recovery.",
		}),
	}
	if reg != nil {
		reg.MustRegister(m.scheduleFiredTotal, m.taskReapedTotal)
	}
	return m
}

// RecordScheduleFired 递增调度触发计数。result 为 "success" 或 "error"。nil-safe。
func (m *SchedulerMetrics) RecordScheduleFired(result string) {
	if m == nil {
		return
	}
	m.scheduleFiredTotal.WithLabelValues(result).Inc()
}

// RecordTaskReaped 递增 reaper 回收计数。nil-safe。
func (m *SchedulerMetrics) RecordTaskReaped() {
	if m == nil {
		return
	}
	m.taskReapedTotal.Inc()
}

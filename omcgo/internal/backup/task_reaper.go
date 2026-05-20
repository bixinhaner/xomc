package backup

// M3: 备份任务 Reaper (backup-restore-alignment-plan M3)
//
// TaskReaper 每 30 秒扫描一次 backup_tasks，找出 status='pending' 且
// created_at < now-2min 的行，对每行重新发布 SubjectBackupTaskCreated 事件。
// 这实现了对 NATS 事件丢失（进程重启、连接抖动）的兜底恢复机制，
// 对应规范中 BackupRestoreTaskJob 的语义。
//
// 幂等性：BackupExecutor 在处理事件前会检查 task.Status，若已不是 pending
// 则直接跳过（写 "not pending, skipping" 日志），不会重复执行。

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/event"
)

// PendingTaskLister 是 TaskReaper 所需的最小任务仓库契约。
// 定义在消费者侧，不修改已有 TaskRepository 接口，避免破坏包内 5 个 mock。
// *PgTaskRepository 实现了此接口（ListPendingOlderThan 方法在 pg_repository.go 中）。
type PendingTaskLister interface {
	ListPendingOlderThan(ctx context.Context, olderThan time.Time, limit int) ([]*BackupTask, error)
}

// TaskReaper 备份任务兜底恢复器。
type TaskReaper struct {
	taskRepo   PendingTaskLister
	bus        event.EventBus
	metrics    *SchedulerMetrics
	scanPeriod time.Duration
	threshold  time.Duration
	logger     *zap.Logger

	stopOnce sync.Once
	stop     chan struct{}
}

// NewTaskReaper 构造 TaskReaper。
//   - scanPeriod <= 0 时使用默认值 30s
//   - threshold <= 0 时使用默认值 2min
//   - metrics 可为 nil
func NewTaskReaper(
	taskRepo PendingTaskLister,
	bus event.EventBus,
	metrics *SchedulerMetrics,
	scanPeriod, threshold time.Duration,
	logger *zap.Logger,
) *TaskReaper {
	if scanPeriod <= 0 {
		scanPeriod = 30 * time.Second
	}
	if threshold <= 0 {
		threshold = 2 * time.Minute
	}
	return &TaskReaper{
		taskRepo:   taskRepo,
		bus:        bus,
		metrics:    metrics,
		scanPeriod: scanPeriod,
		threshold:  threshold,
		logger:     logger.Named("backup-task-reaper"),
		stop:       make(chan struct{}),
	}
}

// Start 在后台 goroutine 中启动 reaper 循环。立即返回。
func (r *TaskReaper) Start() {
	go r.loop()
	r.logger.Info("backup task reaper started",
		zap.Duration("scan_period", r.scanPeriod),
		zap.Duration("pending_threshold", r.threshold))
}

// Stop 通知 reaper goroutine 退出。幂等，可多次调用（也可在 Start 之前调用）。
func (r *TaskReaper) Stop() {
	r.stopOnce.Do(func() { close(r.stop) })
}

func (r *TaskReaper) loop() {
	ticker := time.NewTicker(r.scanPeriod)
	defer ticker.Stop()
	for {
		select {
		case <-r.stop:
			return
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			if err := r.RunOnce(ctx); err != nil {
				r.logger.Warn("backup task reaper tick error", zap.Error(err))
			}
			cancel()
		}
	}
}

// RunOnce 执行一次 reaper 扫描。对外暴露便于测试和手动触发。
func (r *TaskReaper) RunOnce(ctx context.Context) error {
	if r.bus == nil {
		return nil
	}
	olderThan := time.Now().Add(-r.threshold)
	tasks, err := r.taskRepo.ListPendingOlderThan(ctx, olderThan, 100)
	if err != nil {
		return fmt.Errorf("list pending backup tasks for reaper: %w", err)
	}

	for _, t := range tasks {
		taskID := t.ID.String()
		evt, buildErr := event.NewEvent(event.SubjectBackupTaskCreated, map[string]interface{}{
			"task_id": taskID,
		})
		if buildErr != nil {
			r.logger.Warn("reaper: build event failed",
				zap.String("task_id", taskID), zap.Error(buildErr))
			continue
		}
		if pubErr := r.bus.Publish(ctx, event.SubjectBackupTaskCreated, evt); pubErr != nil {
			r.logger.Warn("reaper: re-publish event failed",
				zap.String("task_id", taskID), zap.Error(pubErr))
			continue
		}
		r.metrics.RecordTaskReaped()
		r.logger.Info("reaper: re-published stuck pending backup task",
			zap.String("task_id", taskID))
	}
	return nil
}

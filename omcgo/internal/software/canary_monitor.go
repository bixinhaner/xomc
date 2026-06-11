// Package software — Canary monitor (cron-driven failure-rate guardrail).
//
// The Monitor runs one periodic check:
//
//	every 1 min   CheckCanaryTasks
//
//	  1. ListActiveCanaryTaskIDs (running + paused)
//	  2. For each task, GetCanaryFields
//	  3. Compute failure rate of the active stage
//	     (success_count + fail_count vs total devices for stage)
//	  4. If failure_rate > stage threshold AND status='running' → pause +
//	     log + metric (operators decide resume / abort)
//	  5. If status='running' AND auto_advance AND stage success-completed
//	     AND elapsed > auto_advance_minutes → advance to next stage
//
// Decision: rollback is intentionally NOT automatic (T-0021 separate task).
package software

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

// canaryRepoView is the narrow contract Monitor consumes — defined here so
// tests can mock without pulling the whole TaskRepository surface.
type canaryRepoView interface {
	ListActiveCanaryTaskIDs(ctx context.Context) ([]uuid.UUID, error)
	GetCanaryFields(ctx context.Context, id uuid.UUID) (*CanaryFields, error)
	UpdateCanaryFields(ctx context.Context, id uuid.UUID, fields *CanaryFields) error
	GetByID(ctx context.Context, id uuid.UUID) (*UpgradeTask, error)
}

// canaryRollbackTrigger is the narrow contract for triggering an automatic
// rollback when a canary stage exceeds its failure threshold AND the task
// opted into RollbackOnFailure (T-0021). Optional: nil disables auto-rollback,
// preserving the T-0018 default ("不擅自回滚").
type canaryRollbackTrigger interface {
	TriggerCanaryFailureRollback(ctx context.Context, taskID uuid.UUID, reason string) error
}

// canarySuspender 是「阈值自动暂停真正止血」（#59 Problem 3）的窄契约：阈值越限时，
// 除了把 canary StageStatus 标 Paused，还必须真正掐断在飞 / 待派发的子任务下发。
// 由 SoftwareService.SuspendUpgrade 实现（取消执行 context + 翻 sub_task 状态）。
// 可选：nil 时退化为旧行为（只标 StageStatus，不止血）——但生产装配始终注入。
type canarySuspender interface {
	SuspendUpgrade(ctx context.Context, taskID uuid.UUID) error
}

// CanaryMonitor periodically checks failure rates and auto-advance windows.
type CanaryMonitor struct {
	repo            canaryRepoView
	rollbackTrigger canaryRollbackTrigger // optional; nil = auto-rollback disabled
	suspender       canarySuspender       // optional; nil = threshold pause won't halt dispatch (#59)
	metrics         *CanaryMetrics
	logger          *zap.Logger

	cron   *cron.Cron
	cancel context.CancelFunc
}

// NewCanaryMonitor constructs a Monitor without auto-rollback wiring.
// Use SetRollbackTrigger to enable T-0021 auto-rollback after construction
// (avoids a constructor-cycle: SoftwareService creates the monitor, then wires
// itself in as the trigger).
func NewCanaryMonitor(repo canaryRepoView, metrics *CanaryMetrics, logger *zap.Logger) *CanaryMonitor {
	return &CanaryMonitor{
		repo:    repo,
		metrics: metrics,
		logger:  logger.Named("canary-monitor"),
	}
}

// SetRollbackTrigger wires the auto-rollback callback. nil disables it.
// Safe to call once at startup before Start().
func (m *CanaryMonitor) SetRollbackTrigger(t canaryRollbackTrigger) {
	m.rollbackTrigger = t
}

// SetSuspender wires the threshold-pause止血 callback（#59 Problem 3）。nil 退化为旧
// 行为（阈值越限只标 StageStatus，不真正掐断下发）。生产装配始终注入 SoftwareService。
// Safe to call once at startup before Start().
func (m *CanaryMonitor) SetSuspender(s canarySuspender) {
	m.suspender = s
}

// Start launches the cron schedule (every 1 min).
func (m *CanaryMonitor) Start(ctx context.Context) error {
	scoped, cancel := context.WithCancel(ctx)
	m.cancel = cancel
	m.cron = cron.New()

	if _, err := m.cron.AddFunc("@every 1m", func() {
		c, c2 := context.WithTimeout(scoped, 30*time.Second)
		defer c2()
		if err := m.CheckCanaryTasks(c); err != nil {
			m.logger.Warn("canary tick failed", zap.Error(err))
		}
	}); err != nil {
		return fmt.Errorf("schedule canary monitor: %w", err)
	}
	m.cron.Start()
	m.logger.Info("canary monitor cron started", zap.String("schedule", "@every 1m"))
	return nil
}

// Stop gracefully shuts down the cron.
func (m *CanaryMonitor) Stop() {
	if m.cancel != nil {
		m.cancel()
	}
	if m.cron != nil {
		m.cron.Stop()
	}
}

// CheckCanaryTasks scans active canary tasks and applies stage-rate gating.
// Exposed for direct test invocation; cron calls it on schedule.
func (m *CanaryMonitor) CheckCanaryTasks(ctx context.Context) error {
	ids, err := m.repo.ListActiveCanaryTaskIDs(ctx)
	if err != nil {
		return fmt.Errorf("list active canary tasks: %w", err)
	}
	m.metrics.SetActiveTasks(len(ids))

	for _, id := range ids {
		if err := m.evaluateOne(ctx, id); err != nil {
			m.logger.Warn("evaluate canary task",
				zap.String("task_id", id.String()),
				zap.Error(err))
			// continue with remaining tasks; one failure does not stop the loop
		}
	}
	return nil
}

// evaluateOne checks failure rate for a single task and triggers pause if
// over threshold. auto_advance is processed when the stage completes cleanly.
func (m *CanaryMonitor) evaluateOne(ctx context.Context, id uuid.UUID) error {
	fields, err := m.repo.GetCanaryFields(ctx, id)
	if err != nil {
		return fmt.Errorf("get canary fields: %w", err)
	}
	if !fields.IsCanary() {
		return nil
	}

	stage := fields.CurrentStageDescriptor()
	if stage == nil {
		return nil
	}

	// success/fail counts come from the parent UpgradeTask row.
	task, err := m.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("get upgrade task: %w", err)
	}
	devicesInStage := DevicesForStage(fields.CurrentStage-1, fields.TotalCount, fields.Stages)
	completed := task.SuccessCount + task.FailCount
	rate := FailureRate(task.FailCount, completed)

	m.metrics.SetStageFailureRate(fields.CurrentStage, rate)
	m.metrics.SetDevicesInStage(fields.CurrentStage, devicesInStage)

	thresholdFraction := float64(stage.FailureThreshold) / 100.0

	// Threshold guardrail: only pause when running (avoid double-action on already-paused).
	if fields.StageStatus == StageStatusRunning && completed >= 1 && rate > thresholdFraction {
		reason := fmt.Sprintf("failure_rate %.2f%% > threshold %d%%", rate*100, stage.FailureThreshold)
		fields.StageStatus = StageStatusPaused
		fields.StageHistory = append(fields.StageHistory, StageHistoryEntry{
			Stage:          fields.CurrentStage,
			Percent:        stage.Percent,
			DevicesInStage: devicesInStage,
			SuccessCount:   task.SuccessCount,
			FailCount:      task.FailCount,
			FailureRate:    rate,
			Action:         "paused",
			At:             nowFunc(),
			Reason:         reason,
		})
		m.metrics.RecordAdvance("threshold_exceeded")
		m.logger.Warn("canary task auto-paused (threshold exceeded)",
			zap.String("task_id", id.String()),
			zap.Int("stage", fields.CurrentStage),
			zap.Float64("failure_rate", rate),
			zap.Int("threshold_pct", stage.FailureThreshold),
		)
		if err := m.repo.UpdateCanaryFields(ctx, id, fields); err != nil {
			return err
		}

		// #59 Problem 3 紧急叫停：阈值越限不能只标 StageStatus=Paused（历史 bug——那只是
		// 让 canary 阶段进度停下，待派发 / 在飞的子任务照样继续下发 Download）。这里真正
		// 调 SuspendUpgrade 取消执行 context + 把 active sub_task 翻 Suspended，止住下发。
		// best-effort：失败仅 error 日志，不阻断——StageStatus 已落 Paused，且后续 rollback
		// （若 opt-in）独立运行；运维仍可手动 Suspend/Terminate 兜底。
		if m.suspender != nil {
			if err := m.suspender.SuspendUpgrade(ctx, id); err != nil {
				m.logger.Error("canary threshold auto-suspend failed",
					zap.String("task_id", id.String()),
					zap.Error(err))
			} else {
				m.logger.Warn("canary threshold auto-suspend applied (dispatch halted)",
					zap.String("task_id", id.String()))
			}
		}

		// T-0021 auto-rollback (opt-in only). Pause already happened above —
		// rollback runs *in addition*, so even if the trigger errors the canary
		// task stays safely paused.
		if fields.RollbackOnFailure && m.rollbackTrigger != nil {
			if err := m.rollbackTrigger.TriggerCanaryFailureRollback(ctx, id, reason); err != nil {
				m.logger.Error("canary auto-rollback trigger failed",
					zap.String("task_id", id.String()),
					zap.Error(err))
				// swallow: the pause is already persisted; operators can rollback manually
			} else {
				m.metrics.RecordAdvance("auto_rollback")
			}
		}
		return nil
	}

	// Auto-advance: stage devices fully completed, low failure, auto_advance set.
	if fields.StageStatus == StageStatusRunning &&
		fields.AutoAdvance &&
		completed >= devicesInStage &&
		rate <= thresholdFraction {
		next := fields.CurrentStage + 1
		if next > len(fields.Stages) {
			fields.StageStatus = StageStatusCompleted
			fields.StageHistory = append(fields.StageHistory, StageHistoryEntry{
				Stage:        fields.CurrentStage,
				Action:       "completed",
				FailureRate:  rate,
				SuccessCount: task.SuccessCount,
				FailCount:    task.FailCount,
				At:           nowFunc(),
			})
			m.metrics.RecordAdvance("completed")
		} else {
			fields.StageHistory = append(fields.StageHistory, StageHistoryEntry{
				Stage:        fields.CurrentStage,
				Percent:      stage.Percent,
				FailureRate:  rate,
				SuccessCount: task.SuccessCount,
				FailCount:    task.FailCount,
				Action:       "advanced",
				At:           nowFunc(),
				Reason:       "auto_advance",
			})
			fields.CurrentStage = next
			m.metrics.RecordAdvance("advanced")
		}
		m.logger.Info("canary task auto-advanced",
			zap.String("task_id", id.String()),
			zap.Int("from_stage", fields.CurrentStage),
		)
		return m.repo.UpdateCanaryFields(ctx, id, fields)
	}
	return nil
}

// Package software — Canary monitor (cron-driven failure-rate guardrail).
//
// The Monitor runs one periodic check:
//   every 1 min   CheckCanaryTasks
//
//     1. ListActiveCanaryTaskIDs (running + paused)
//     2. For each task, GetCanaryFields
//     3. Compute failure rate of the active stage
//        (success_count + fail_count vs total devices for stage)
//     4. If failure_rate > stage threshold AND status='running' → pause +
//        log + metric (operators decide resume / abort)
//     5. If status='running' AND auto_advance AND stage success-completed
//        AND elapsed > auto_advance_minutes → advance to next stage
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

// CanaryMonitor periodically checks failure rates and auto-advance windows.
type CanaryMonitor struct {
	repo    canaryRepoView
	metrics *CanaryMetrics
	logger  *zap.Logger

	cron   *cron.Cron
	cancel context.CancelFunc
}

// NewCanaryMonitor constructs a Monitor.
func NewCanaryMonitor(repo canaryRepoView, metrics *CanaryMetrics, logger *zap.Logger) *CanaryMonitor {
	return &CanaryMonitor{
		repo:    repo,
		metrics: metrics,
		logger:  logger.Named("canary-monitor"),
	}
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
			Reason:         fmt.Sprintf("failure_rate %.2f%% > threshold %d%%", rate*100, stage.FailureThreshold),
		})
		m.metrics.RecordAdvance("threshold_exceeded")
		m.logger.Warn("canary task auto-paused (threshold exceeded)",
			zap.String("task_id", id.String()),
			zap.Int("stage", fields.CurrentStage),
			zap.Float64("failure_rate", rate),
			zap.Int("threshold_pct", stage.FailureThreshold),
		)
		return m.repo.UpdateCanaryFields(ctx, id, fields)
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

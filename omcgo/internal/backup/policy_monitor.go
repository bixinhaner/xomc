// Package backup — BackupPolicy enforcement Monitor (T-0073 Phase 1).
//
// The Monitor runs one cron tick per day:
//
//	@daily   RunCleanupOnce — reads the active BackupPolicy and deletes
//	                          terminal-status backup_tasks rows older than
//	                          retention_days while keeping the most-recent
//	                          keep_last_n rows per target_type partition.
//
// Phase 1 boundary: this Monitor only deletes DB rows. Physical file deletion
// (MinIO / local FS path indicated by `file_path`) is deferred to T-0076 once
// the transfer-module file storage path is audited. Storage-quota threshold
// alarms (alert_threshold_percent) are also Phase 2.
package backup

import (
	"context"
	"fmt"
	"time"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

// PolicyMonitor periodically enforces BackupPolicy.AutoCleanup against
// backup_tasks rows. Failure alarm publishing is handled inline by
// PublishFailureAlarm in the executor's failure path, not by this cron.
type PolicyMonitor struct {
	policyService *PolicyService
	taskRepo      TaskRepository
	metrics       *PolicyMetrics
	logger        *zap.Logger

	cron   *cron.Cron
	cancel context.CancelFunc
}

// NewPolicyMonitor constructs a Monitor. metrics and any sub-dependency may
// be nil for tests; the methods are nil-safe.
func NewPolicyMonitor(policyService *PolicyService, taskRepo TaskRepository, metrics *PolicyMetrics, logger *zap.Logger) *PolicyMonitor {
	return &PolicyMonitor{
		policyService: policyService,
		taskRepo:      taskRepo,
		metrics:       metrics,
		logger:        logger.Named("backup-policy-monitor"),
	}
}

// Start launches the cron schedule (@daily at 00:00 server time).
// The current implementation does not use BackupPolicy.CleanupTime — the
// cron fires daily; cleanup itself is idempotent so the actual hour matters
// little. Honoring CleanupTime exactly is a Phase 2 enhancement.
func (m *PolicyMonitor) Start(ctx context.Context) error {
	scoped, cancel := context.WithCancel(ctx)
	m.cancel = cancel
	m.cron = cron.New()

	if _, err := m.cron.AddFunc("@daily", func() {
		c, c2 := context.WithTimeout(scoped, 5*time.Minute)
		defer c2()
		if _, err := m.RunCleanupOnce(c); err != nil {
			m.logger.Warn("backup cleanup tick failed", zap.Error(err))
		}
	}); err != nil {
		// Review fix HIGH-2: release the scoped context so we don't leak it on
		// failed AddFunc paths.
		cancel()
		m.cancel = nil
		m.cron = nil
		return fmt.Errorf("schedule backup policy monitor: %w", err)
	}
	m.cron.Start()
	m.logger.Info("backup policy monitor started", zap.String("schedule", "@daily"))
	return nil
}

// Stop gracefully shuts down the cron.
func (m *PolicyMonitor) Stop() {
	if m.cancel != nil {
		m.cancel()
	}
	if m.cron != nil {
		m.cron.Stop()
	}
}

// RunCleanupOnce executes a single cleanup pass: reads the active BackupPolicy,
// honours auto_cleanup, and dispatches to TaskRepository.CleanupOldRows.
// Returns rows deleted. Exposed for direct test invocation; cron calls it on
// schedule.
func (m *PolicyMonitor) RunCleanupOnce(ctx context.Context) (int64, error) {
	policy, err := m.policyService.Get(ctx)
	if err != nil {
		m.metrics.RecordCleanupRun("failure")
		return 0, fmt.Errorf("get backup policy: %w", err)
	}

	if !policy.AutoCleanup {
		m.metrics.RecordCleanupRun("skipped")
		m.logger.Debug("backup cleanup skipped: auto_cleanup=false")
		return 0, nil
	}

	cutoff := time.Now().Add(-time.Duration(policy.RetentionDays) * 24 * time.Hour)
	deleted, err := m.taskRepo.CleanupOldRows(ctx, cutoff, policy.KeepLastN)
	if err != nil {
		m.metrics.RecordCleanupRun("failure")
		return 0, fmt.Errorf("cleanup old backup_tasks: %w", err)
	}

	m.metrics.RecordCleanupRun("success")
	m.metrics.RecordCleanupDeleted(deleted)
	m.logger.Info("backup cleanup completed",
		zap.Int64("rows_deleted", deleted),
		zap.Int("retention_days", policy.RetentionDays),
		zap.Int("keep_last_n", policy.KeepLastN),
		zap.Time("cutoff", cutoff),
	)
	return deleted, nil
}

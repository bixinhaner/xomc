// Package backup — BackupPolicy enforcement Monitor (T-0073 + T-0076).
//
// The Monitor runs one cron tick per day:
//
//	@daily   RunCleanupOnce — reads the active BackupPolicy and deletes
//	                          terminal-status backup_tasks rows older than
//	                          retention_days while keeping the most-recent
//	                          keep_last_n rows per target_type partition.
//	                          T-0076: physically deletes the corresponding
//	                          MinIO objects (best-effort, by file_path).
//
// T-0076 closed: physical file delete now runs alongside DB cleanup. Each
// deleted backup_tasks row's file_path (where non-NULL — populated by T-0079
// linkage) is fed to MinIO RemoveObject; failures are recorded in metrics
// and logged but never roll back the DB delete (which is the source of
// truth). Storage-quota threshold alarms (alert_threshold_percent) and the
// multi-device orphan reaper remain deferred to T-0082 / T-0083.
package backup

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

// MinIOObjectRemover is the narrow contract PolicyMonitor needs to physically
// delete backup objects (T-0076). *minio.Client satisfies it; declared at the
// consumer per Go convention to keep test mocks small.
type MinIOObjectRemover interface {
	RemoveObject(ctx context.Context, bucketName, objectName string, opts minio.RemoveObjectOptions) error
}

// PolicyMonitor periodically enforces BackupPolicy.AutoCleanup against
// backup_tasks rows. Failure alarm publishing is handled inline by
// PublishFailureAlarm in the executor's failure path, not by this cron.
//
// T-0076: when the optional MinIO field is wired, the cron also runs
// best-effort physical delete on each cleaned-up backup_task's file_path.
// The MinIO field is optional (nil-safe) so tests can drive the cron
// without a real MinIO and the existing T-0073 behaviour is preserved when
// physical delete is undesired.
type PolicyMonitor struct {
	policyService *PolicyService
	taskRepo      TaskRepository
	metrics       *PolicyMetrics
	logger        *zap.Logger
	minio         MinIOObjectRemover // optional T-0076 — nil disables physical delete

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

// SetMinIO wires an object remover for T-0076 physical delete. Pass nil to
// disable (preserves T-0073 DB-only behaviour). Constructed separately from
// NewPolicyMonitor to keep the signature stable for existing call sites.
func (m *PolicyMonitor) SetMinIO(remover MinIOObjectRemover) {
	m.minio = remover
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
	filePaths, deleted, err := m.taskRepo.CleanupOldRows(ctx, cutoff, policy.KeepLastN)
	if err != nil {
		m.metrics.RecordCleanupRun("failure")
		return 0, fmt.Errorf("cleanup old backup_tasks: %w", err)
	}

	m.metrics.RecordCleanupRun("success")
	m.metrics.RecordCleanupDeleted(deleted)

	// T-0076: physically delete MinIO objects best-effort. file_paths only
	// contains entries where backup_tasks.file_path was non-empty (the
	// repository filters NULL/empty out); legacy rows from before T-0079
	// linkage are skipped here.
	deletedFiles := 0
	if m.minio != nil && len(filePaths) > 0 {
		for _, fp := range filePaths {
			deletedFiles += m.physicallyDeleteOne(ctx, fp)
		}
	}

	// Multi-device orphans: rows deleted minus files we knew to delete (and
	// excluding null/empty file_path rows which we can't blame on multi-device).
	// This count is informational; T-0083 will close the gap.
	rowsWithKnownFile := int64(len(filePaths))
	rowsWithoutFile := deleted - rowsWithKnownFile
	if rowsWithoutFile > 0 {
		m.logger.Info("backup cleanup left rows without known file_path",
			zap.Int64("rows_without_file", rowsWithoutFile),
			zap.String("note", "legacy pre-T-0079 rows or multi-device backup orphans (T-0083 followup)"),
		)
	}

	m.logger.Info("backup cleanup completed",
		zap.Int64("rows_deleted", deleted),
		zap.Int("files_attempted", len(filePaths)),
		zap.Int("files_deleted", deletedFiles),
		zap.Int("retention_days", policy.RetentionDays),
		zap.Int("keep_last_n", policy.KeepLastN),
		zap.Time("cutoff", cutoff),
	)
	return deleted, nil
}

// physicallyDeleteOne tries to RemoveObject for a single "bucket/path"
// concatenated path. Returns 1 on success, 0 on any failure (errors are
// classified into metric reasons and logged at warn level — never propagated).
func (m *PolicyMonitor) physicallyDeleteOne(ctx context.Context, fullPath string) int {
	bucket, objectPath, err := splitBucketAndPath(fullPath)
	if err != nil {
		m.metrics.RecordFileDeleteError("parse_path")
		m.logger.Warn("malformed backup file_path; skipping physical delete",
			zap.String("file_path", fullPath),
			zap.Error(err),
		)
		return 0
	}
	if err := m.minio.RemoveObject(ctx, bucket, objectPath, minio.RemoveObjectOptions{}); err != nil {
		reason := classifyMinIORemoveErr(err)
		m.metrics.RecordFileDeleteError(reason)
		m.logger.Warn("MinIO RemoveObject failed; continuing cleanup",
			zap.String("bucket", bucket),
			zap.String("object_path", objectPath),
			zap.String("reason", reason),
			zap.Error(err),
		)
		return 0
	}
	m.metrics.RecordFileDeleted()
	return 1
}

// classifyMinIORemoveErr maps a RemoveObject error to a coarse metric label.
// "not_found" — already deleted (idempotent benign);
// "network"   — transient transport issue;
// "other"     — unexpected (auth, server-side, etc.).
func classifyMinIORemoveErr(err error) string {
	if err == nil {
		return ""
	}
	resp := minio.ToErrorResponse(err)
	if resp.Code == "NoSuchKey" || resp.Code == "NoSuchBucket" {
		return "not_found"
	}
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "connection") ||
		strings.Contains(msg, "timeout") ||
		strings.Contains(msg, "no route") ||
		strings.Contains(msg, "i/o"):
		return "network"
	case errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled):
		return "network"
	}
	return "other"
}

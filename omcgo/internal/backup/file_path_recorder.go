// Package backup — backup_task ↔ file_path linkage (T-0079).
//
// The ACS upload handler publishes `backup.file.received` after a FileType=3
// (Vendor Configuration File) lands in MinIO; FilePathRecorder subscribes,
// extracts the backup_task UUID prefix from the filename embedded by the
// executor (`backup-{taskID8}-{deviceSN}.xml`), and writes
// backup_tasks.file_path so that `restore_by_task_id` mode has data to work
// with.
//
// First-write-wins semantics for multi-device backup tasks: the first
// device's upload sets file_path; subsequent uploads from siblings see a
// non-null file_path and are skipped (logged + metric). Multi-device
// restore-by-task-id then uses the first device's config as the source for
// all targets — see T-0079 PRD §2.3 for the rationale.
package backup

import (
	"context"
	"errors"
	"fmt"

	"go.uber.org/zap"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/event"
)

// BackupFileReceivedPayload mirrors the JSON map published by the ACS upload
// handler when FileType=3. Defined here so subscribers have a typed surface;
// the publisher constructs the same field names via map[string]interface{}.
type BackupFileReceivedPayload struct {
	Bucket             string `json:"bucket"`
	ObjectPath         string `json:"object_path"`
	Filename           string `json:"filename"`
	BackupTaskIDPrefix string `json:"backup_task_id_prefix"`
	DeviceSN           string `json:"device_sn"`
	FileSize           int64  `json:"file_size"`
}

// FilePathRecorder subscribes to SubjectBackupFileReceived and writes the
// originating backup_tasks row's file_path. Lives on the App side (not ACS)
// so the ACS process stays decoupled from the backup module.
type FilePathRecorder struct {
	repo    TaskRepository
	metrics *RestoreMetrics
	logger  *zap.Logger
}

// NewFilePathRecorder constructs a FilePathRecorder. metrics may be nil
// (Record* short-circuits).
func NewFilePathRecorder(
	repo TaskRepository,
	metrics *RestoreMetrics,
	logger *zap.Logger,
) *FilePathRecorder {
	return &FilePathRecorder{
		repo:    repo,
		metrics: metrics,
		logger:  logger.Named("backup-file-path-recorder"),
	}
}

// Subscribe wires the recorder into the EventBus. Uses QueueSubscribe so
// only one App instance processes each event in a multi-replica deployment.
func (r *FilePathRecorder) Subscribe(bus event.EventBus) error {
	_, err := bus.QueueSubscribe(
		event.SubjectBackupFileReceived,
		"backup-file-recorder",
		r.handleFileReceived,
	)
	if err != nil {
		return fmt.Errorf("subscribe %s: %w", event.SubjectBackupFileReceived, err)
	}
	r.logger.Info("backup file path recorder subscribed",
		zap.String("subject", event.SubjectBackupFileReceived))
	return nil
}

// handleFileReceived is the EventBus callback. Best-effort by design: any
// non-fatal classification (no match, already set, malformed payload) returns
// nil so NATS doesn't redeliver. Real DB errors return the wrapped error.
func (r *FilePathRecorder) handleFileReceived(ctx context.Context, evt event.Event) error {
	var p BackupFileReceivedPayload
	if err := evt.DecodePayload(&p); err != nil {
		r.metrics.RecordFilePathRecord("error")
		return fmt.Errorf("decode backup.file.received: %w", err)
	}

	if p.BackupTaskIDPrefix == "" {
		// Filename did not match `backup-{taskID8}-{sn}.xml` — likely an
		// operator-uploaded ad-hoc config file, not from our executor. No-op.
		r.metrics.RecordFilePathRecord("skipped_no_match")
		r.logger.Debug("backup file received with empty task_id prefix; skipping",
			zap.String("filename", p.Filename))
		return nil
	}

	matches, err := r.repo.FindByIDPrefix(ctx, p.BackupTaskIDPrefix, 2)
	if err != nil {
		r.metrics.RecordFilePathRecord("error")
		return fmt.Errorf("find backup_task by prefix %q: %w", p.BackupTaskIDPrefix, err)
	}
	if len(matches) == 0 {
		// Prefix didn't match any current row — the originating backup_task
		// may have been cleaned up (T-0073) or this file came from a foreign
		// system. Skip without error.
		r.metrics.RecordFilePathRecord("skipped_no_match")
		r.logger.Info("no backup_task matches prefix; skipping",
			zap.String("prefix", p.BackupTaskIDPrefix),
			zap.String("filename", p.Filename))
		return nil
	}
	if len(matches) > 1 {
		// 8-hex-char prefix collision is rare (~1 in 4B) but possible. Pick
		// the most recent (FindByIDPrefix already orders DESC by created_at).
		r.logger.Warn("multiple backup_tasks match prefix; using most recent",
			zap.String("prefix", p.BackupTaskIDPrefix),
			zap.Int("match_count", len(matches)))
	}
	target := matches[0]

	// Fast-path: avoid an unnecessary UPDATE round-trip when we already
	// observed a non-null file_path. The DB-layer CAS in UpdateFilePath is
	// the actual correctness boundary (see review HIGH fix); this read-side
	// check is purely an optimization for the common multi-device case
	// where a sibling already won the race.
	if target.FilePath != nil && *target.FilePath != "" {
		r.metrics.RecordFilePathRecord("skipped_already_set")
		r.logger.Debug("backup_task.file_path already set (fast-path skip)",
			zap.String("task_id", target.ID.String()),
			zap.String("existing_path", *target.FilePath),
			zap.String("incoming_path", p.ObjectPath))
		return nil
	}

	fullPath := p.Bucket + "/" + p.ObjectPath
	err = r.repo.UpdateFilePath(ctx, target.ID, fullPath)
	switch {
	case err == nil:
		r.metrics.RecordFilePathRecord("recorded")
		r.logger.Info("backup_task.file_path recorded",
			zap.String("task_id", target.ID.String()),
			zap.String("path", fullPath),
			zap.String("device_sn", p.DeviceSN))
		return nil
	case errors.Is(err, ErrFilePathAlreadySet):
		// CAS lost: another concurrent recorder won. This is the TOCTOU-safe
		// branch — both recorders observed null in their fast-path read but
		// only one's UPDATE matched. Idempotent no-op for the loser.
		r.metrics.RecordFilePathRecord("skipped_already_set")
		r.logger.Debug("backup_task.file_path CAS lost; sibling recorder won",
			zap.String("task_id", target.ID.String()),
			zap.String("incoming_path", p.ObjectPath))
		return nil
	case errors.Is(err, commonerrors.ErrNotFound):
		// Row vanished between FindByIDPrefix and UpdateFilePath (raced with
		// T-0073 cleanup). Not fatal.
		r.metrics.RecordFilePathRecord("skipped_no_match")
		r.logger.Info("backup_task vanished between find and update",
			zap.String("task_id", target.ID.String()))
		return nil
	default:
		r.metrics.RecordFilePathRecord("error")
		return fmt.Errorf("update backup_task %s file_path: %w", target.ID, err)
	}
}

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
	// MD5 — M1 of backup-restore-alignment-plan: ACS 侧使用 MinIO PutObject
	// 返回的 ETag。单块 PutObject (未启用 multipart) 下 ETag = MD5(hex)；
	// 备份配置文件一般远小于 multipart 阈值 (5MiB)，因此在实际场景
	// 中 ETag 可靠。遇到 multipart ETag (带 -N 后缀) 时消费者应忽略。
	MD5 string `json:"md5,omitempty"`
}

// FilePathRecorder subscribes to SubjectBackupFileReceived and writes the
// originating backup_tasks row's file_path. Lives on the App side (not ACS)
// so the ACS process stays decoupled from the backup module.
//
// M1 of backup-restore-alignment-plan: 额外 upsert backup_restore_file
// 元数据表 (SN/file_name/md5/size/operator_code/update_time)，以供后续
// queryCellInfos / single/exportFile 查询。fileRepo 可为 nil（向下兼容
// 老部署）。
type FilePathRecorder struct {
	repo     TaskRepository
	fileRepo FileRepository // optional (M1)
	metrics  *RestoreMetrics
	logger   *zap.Logger
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

// SetFileRepository wires the backup_restore_file repository post-construction
// so the recorder also persists per-file metadata (M1). Pass nil to disable
// (default behavior — only backup_tasks.file_path is recorded).
func (r *FilePathRecorder) SetFileRepository(fr FileRepository) {
	r.fileRepo = fr
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
		// M1: 同步落库 backup_restore_file 元数据。失败不阻断主路径 ——
		// file_path 已在 backup_tasks 中记录，元数据表是补充查询面。
		r.upsertFileMetadata(ctx, target, p, fullPath)
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

// upsertFileMetadata writes / refreshes a backup_restore_file row keyed by
// (serial_number, file_name). Best-effort: failure is logged but does not
// propagate so the backup_tasks.file_path success path remains the source
// of truth. No-op when fileRepo is nil (deployment did not wire it).
func (r *FilePathRecorder) upsertFileMetadata(
	ctx context.Context,
	target *BackupTask,
	p BackupFileReceivedPayload,
	fullPath string,
) {
	if r.fileRepo == nil {
		return
	}
	if p.DeviceSN == "" || p.Filename == "" {
		return
	}
	var md5 *string
	if p.MD5 != "" {
		v := p.MD5
		md5 = &v
	}
	var operatorCode *string
	if target.OperatorCode != nil && *target.OperatorCode != "" {
		operatorCode = target.OperatorCode
	}
	f := &BackupRestoreFile{
		SerialNumber: p.DeviceSN,
		FileName:     p.Filename,
		ObjectPath:   fullPath,
		MD5:          md5,
		FileSize:     p.FileSize,
		OperatorCode: operatorCode,
	}
	if err := r.fileRepo.Upsert(ctx, f); err != nil {
		r.logger.Warn("upsert backup_restore_file failed (best-effort)",
			zap.String("device_sn", p.DeviceSN),
			zap.String("filename", p.Filename),
			zap.Error(err))
		return
	}
	r.logger.Debug("backup_restore_file upserted",
		zap.String("device_sn", p.DeviceSN),
		zap.String("filename", p.Filename),
		zap.Int64("file_size", p.FileSize))
}

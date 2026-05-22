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
	// TaskID 是 UFTE 主任务的完整 UUID（upgrade_tasks.id）。比 8-char Prefix
	// 更精确，写入 backup_restore_file.task_id 后能用于精确隔离不同任务下
	// 同 SN 同名文件的元数据。空字符串表示老链路 / 非任务路径上传。
	TaskID   string `json:"task_id,omitempty"`
	DeviceSN string `json:"device_sn"`
	FileSize int64  `json:"file_size"`
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
	// notifier 可选 hook：在文件落盘 + metadata 写入完成后被调一次，让上层（如
	// software/UFTE 的 FAULT_LOG_COLLECT 链路）按 (task_id, device_sn) 推进自己的
	// sub_task 状态。
	//
	// 为什么走 hook 而不是订阅 backup.file.received：NATS BACKUP stream 是
	// WorkQueuePolicy retention，同 subject 只能挂 1 个 filter consumer——
	// FilePathRecorder 已经独占，再加 software 端 subscribe 会被 NATS 拒
	// ("filtered consumer not unique on workqueue stream")。所以让已经收到
	// 事件的 FilePathRecorder 转手通知上层。
	notifier FileLandedNotifier
}

// FileLandedNotifier 是 FilePathRecorder 处理完一条 backup.file.received 后回调
// 给上层的 hook（消费者驱动接口，定义在 backup 包）。software 模块实现并注入；
// 详见 file_path_recorder.go 注释里"为什么走 hook"。
type FileLandedNotifier interface {
	// OnLogFileLanded 在 backup_restore_file metadata upsert 完成后调用，best-effort
	// 语义（实现内部处理错误并日志，不影响 FilePathRecorder 主链路 / NATS ack）。
	// deviceSN + taskID 来自 ACS upload handler URL query；taskID 为完整 UUID 字符串，
	// 实现方按 (deviceSN, taskID) 反查自己的 active 子任务并推进。
	OnLogFileLanded(ctx context.Context, deviceSN, taskID string)
}

// SetFileLandedNotifier 注入 hook。nil 表示禁用（向后兼容）。
func (r *FilePathRecorder) SetFileLandedNotifier(n FileLandedNotifier) {
	r.notifier = n
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
//
// 双分支语义（2026-05-20 修复，backup-display-fix-20260520.md B4）：
//   1. 旧 backup_tasks 链路（T-0079 first-write-wins）—— 仅当 prefix 命中
//      backup_tasks 行才写 file_path，未命中跳过。
//   2. backup_restore_file 元数据 —— 自然键 (serial_number, file_name)，
//      *与 backup_tasks 是否命中无关*。UFTE / 自动开站等链路也会上报
//      backup.file.received，它们的"任务"实体不在 backup_tasks 而在
//      software.upgrade_tasks，但前端展示与下载链路需要这份元数据来定位
//      object_path 与 MD5。因此该 upsert 提到任何 backup_tasks 判定之前。
func (r *FilePathRecorder) handleFileReceived(ctx context.Context, evt event.Event) error {
	var p BackupFileReceivedPayload
	if err := evt.DecodePayload(&p); err != nil {
		r.metrics.RecordFilePathRecord("error")
		return fmt.Errorf("decode backup.file.received: %w", err)
	}

	fullPath := p.Bucket + "/" + p.ObjectPath

	// 分支 2：先 upsert metadata —— 与下面的 backup_tasks 匹配无关，
	// nil target 表示找不到原 backup_task 行（UFTE 链路 / 旧任务被清理），
	// 此时 OperatorCode 留空。
	r.upsertFileMetadata(ctx, nil, p, fullPath)

	// 通知上层（如 software UFTE 的 FAULT_LOG_COLLECT）按 (deviceSN, taskID) 推进
	// 自己的 sub_task。一定要放在 backup_tasks 匹配前——UFTE 链路 task 在
	// upgrade_tasks 表，prefix 永远不会命中下面的 backup_tasks 分支。
	r.logger.Info("file_landed hook check",
		zap.Bool("notifier_wired", r.notifier != nil),
		zap.String("device_sn", p.DeviceSN),
		zap.String("task_id", p.TaskID),
		zap.String("filename", p.Filename))
	if r.notifier != nil && p.DeviceSN != "" && p.TaskID != "" {
		r.notifier.OnLogFileLanded(ctx, p.DeviceSN, p.TaskID)
	}

	// 分支 1：尝试匹配旧 backup_tasks 链路。filename 不带 backup- 前缀的
	// 上报（操作员手工上传、外部系统）走不到这里——直接返回。
	if p.BackupTaskIDPrefix == "" {
		r.metrics.RecordFilePathRecord("metadata_only_no_prefix")
		r.logger.Debug("backup file received with empty task_id prefix; metadata-only",
			zap.String("filename", p.Filename))
		return nil
	}

	matches, err := r.repo.FindByIDPrefix(ctx, p.BackupTaskIDPrefix, 2)
	if err != nil {
		r.metrics.RecordFilePathRecord("error")
		return fmt.Errorf("find backup_task by prefix %q: %w", p.BackupTaskIDPrefix, err)
	}
	if len(matches) == 0 {
		// Prefix didn't match —— UFTE 链路（task 在 software.upgrade_tasks）
		// 或旧任务已被 T-0073 清理。metadata 已在分支 2 写入，主表跳过即可。
		r.metrics.RecordFilePathRecord("metadata_only_no_match")
		r.logger.Info("no backup_task matches prefix; metadata-only",
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

	err = r.repo.UpdateFilePath(ctx, target.ID, fullPath)
	switch {
	case err == nil:
		r.metrics.RecordFilePathRecord("recorded")
		r.logger.Info("backup_task.file_path recorded",
			zap.String("task_id", target.ID.String()),
			zap.String("path", fullPath),
			zap.String("device_sn", p.DeviceSN))
		// 命中 backup_tasks 时补一次带 OperatorCode 的 upsert（覆盖 nil 路径），
		// COALESCE 保证已有 operator_code 不会被覆盖为空。
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
//
// target 可为 nil（B4 改造，2026-05-20）：UFTE 链路 / 操作员手工上传等场景
// 找不到对应 backup_tasks 行，仍需要 metadata 来支撑 UI 下载链路；此时
// OperatorCode 留空，其他字段不受影响。
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
	if target != nil && target.OperatorCode != nil && *target.OperatorCode != "" {
		operatorCode = target.OperatorCode
	}
	var taskID *string
	if p.TaskID != "" {
		tid := p.TaskID
		taskID = &tid
	}
	f := &BackupRestoreFile{
		SerialNumber: p.DeviceSN,
		FileName:     p.Filename,
		ObjectPath:   fullPath,
		MD5:          md5,
		FileSize:     p.FileSize,
		OperatorCode: operatorCode,
		TaskID:       taskID,
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

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
	"strings"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
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
	MD5       string `json:"md5,omitempty"`
	ParamType string `json:"param_type,omitempty"`
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
	notifier          FileLandedNotifier
	paramFileRecorder ParamFileLandedRecorder

	// snapshotPromoter 可选 hook（T-0164 / B3）：每次本 SN 的备份文件成功落
	// MinIO + backup_tasks/backup_restore_file 完成写入后，把该文件 server-side
	// copy 到 config-snapshots bucket 并 upsert config_snapshots 表。
	//
	// 失败仅 warn + metric，不影响主链路成功 —— 配置快照表是辅助索引，
	// 任何故障都不应让备份任务被标记为失败。
	snapshotPromoter SnapshotPromoter

	logQuotaRepo    LogFileQuotaRepository
	logQuotaPolicy  LogFileQuotaPolicy
	logQuotaRemover logQuotaObjectRemover
}

// LogFileQuotaPolicy 提供故障日志文件数限额，生产上由 stationlog.RetentionPolicy 实现。
type LogFileQuotaPolicy interface {
	MaxFileCount(ctx context.Context) int
	MaxFileCountPerDevice(ctx context.Context) int
}

type logQuotaObjectRemover interface {
	RemoveObject(ctx context.Context, bucket, object string, opts minio.RemoveObjectOptions) error
}

// SnapshotPromoter 是 FilePathRecorder 对 SnapshotService 的最小依赖。
// *SnapshotService 通过同名方法满足；在 backup 模块内同包定义所以无需 import 反向。
type SnapshotPromoter interface {
	PromoteFromBackup(ctx context.Context, backupTaskID uuid.UUID, deviceSN string) error
}

// SetSnapshotPromoter 注入快照 promote hook（B3）。nil 表示禁用。
func (r *FilePathRecorder) SetSnapshotPromoter(p SnapshotPromoter) {
	r.snapshotPromoter = p
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

type ParamFileLandedRecorder interface {
	RecordCollectedFile(ctx context.Context, paramType, deviceSN, fileName, objectBucket, objectPath string, fileSize int64, md5Value string) error
}

// SetFileLandedNotifier 注入 hook。nil 表示禁用（向后兼容）。
func (r *FilePathRecorder) SetFileLandedNotifier(n FileLandedNotifier) {
	r.notifier = n
}

func (r *FilePathRecorder) SetParamFileLandedRecorder(recorder ParamFileLandedRecorder) {
	r.paramFileRecorder = recorder
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

// SetLogFileQuota 注入故障日志文件配额清理依赖。它只作用于 backup_restore_file 中
// logs/fault 的元数据，不影响运行日志、配置备份，也不写 station_fault_logs。
func (r *FilePathRecorder) SetLogFileQuota(repo LogFileQuotaRepository, policy LogFileQuotaPolicy, remover logQuotaObjectRemover) {
	r.logQuotaRepo = repo
	r.logQuotaPolicy = policy
	r.logQuotaRemover = remover
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
//  1. 旧 backup_tasks 链路（T-0079 first-write-wins）—— 仅当 prefix 命中
//     backup_tasks 行才写 file_path，未命中跳过。
//  2. backup_restore_file 元数据 —— 自然键 (serial_number, file_name)，
//     *与 backup_tasks 是否命中无关*。UFTE / 自动开站等链路也会上报
//     backup.file.received，它们的"任务"实体不在 backup_tasks 而在
//     software.upgrade_tasks，但前端展示与下载链路需要这份元数据来定位
//     object_path 与 MD5。因此该 upsert 提到任何 backup_tasks 判定之前。
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
	metadata := r.upsertFileMetadata(ctx, nil, p, fullPath)
	r.enforceLogFileQuota(ctx, metadata)
	if r.paramFileRecorder != nil && p.ParamType != "" && p.DeviceSN != "" {
		if err := r.paramFileRecorder.RecordCollectedFile(
			ctx, p.ParamType, p.DeviceSN, p.Filename, p.Bucket, p.ObjectPath, p.FileSize, p.MD5,
		); err != nil {
			r.logger.Warn("record collected core file failed",
				zap.String("device_sn", p.DeviceSN), zap.String("filename", p.Filename), zap.Error(err))
		}
	}

	// T-0164 关键：promote 必须在 backup_tasks 匹配**之前**触发，因为现网真实链路
	// 大多走 UFTE 的 CONFIG_BACKUP_XML / CONFIG_BACKUP_NV —— 主任务在
	// software.upgrade_tasks 表，prefix 永远不会命中下面的 backup_tasks 分支。
	// 只要 sn 非空、metadata 已 upsert（pickBackupFileForTask 能拿到这一行），
	// promote 就该执行。UFTE 路径下 backupTaskID 用 event payload.TaskID（如有），
	// 没有则用 zero UUID — PromoteFromBackup 内部会 fallback 到 ListBySerial 的
	// 最新一行（按 update_time DESC），与刚 upsert 的 metadata 一致。
	if p.DeviceSN != "" && p.Filename != "" {
		var promoteTaskID uuid.UUID
		if p.TaskID != "" {
			if parsed, perr := uuid.Parse(p.TaskID); perr == nil {
				promoteTaskID = parsed
			}
		}
		r.promoteSnapshot(ctx, promoteTaskID, p.DeviceSN)
	}

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
		// 注：promote 已在事件处理早期统一执行（详见上面分支 2 后注释），此处
		// 无需重复调用。
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
) *BackupRestoreFile {
	if r.fileRepo == nil {
		return nil
	}
	if p.DeviceSN == "" || p.Filename == "" {
		return nil
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
		return nil
	}
	r.logger.Debug("backup_restore_file upserted",
		zap.String("device_sn", p.DeviceSN),
		zap.String("filename", p.Filename),
		zap.Int64("file_size", p.FileSize))
	return f
}

func (r *FilePathRecorder) enforceLogFileQuota(ctx context.Context, f *BackupRestoreFile) {
	if f == nil || !isQuotaManagedLogObjectPath(f.ObjectPath) {
		return
	}
	if r.logQuotaRepo == nil || r.logQuotaPolicy == nil {
		return
	}

	r.enforceGlobalLogFileQuota(ctx)

	if f.SerialNumber == "" {
		return
	}
	r.enforceSerialLogFileQuota(ctx, f.SerialNumber)
}

// EnforceAllLogFileQuotas 收敛现存故障日志文件数配额。它用于 sys_configs 保存后：
// 当每设备/全局配额被调小，旧的活跃文件也应尽快置灰，而不等下一次设备上传。
func (r *FilePathRecorder) EnforceAllLogFileQuotas(ctx context.Context) {
	if r.logQuotaRepo == nil || r.logQuotaPolicy == nil {
		return
	}
	r.enforceGlobalLogFileQuota(ctx)

	if maxPerDevice := r.logQuotaPolicy.MaxFileCountPerDevice(ctx); maxPerDevice > 0 {
		counts, err := r.logQuotaRepo.ListActiveLogFileSerialCounts(ctx)
		if err != nil {
			r.logger.Warn("list active log file serial counts for quota convergence", zap.Error(err))
			return
		}
		for _, item := range counts {
			if item.Count <= int64(maxPerDevice) || item.SerialNumber == "" {
				continue
			}
			oldest, listErr := r.logQuotaRepo.ListOldestActiveLogFilesBySerial(
				ctx, item.SerialNumber, int(item.Count-int64(maxPerDevice)),
			)
			if listErr != nil {
				r.logger.Warn("list oldest log files by serial for quota convergence",
					zap.String("device_sn", item.SerialNumber), zap.Error(listErr))
				continue
			}
			r.removeQuotaLogFiles(ctx, oldest)
		}
	}
}

func (r *FilePathRecorder) enforceGlobalLogFileQuota(ctx context.Context) {
	if maxCount := r.logQuotaPolicy.MaxFileCount(ctx); maxCount > 0 {
		count, err := r.logQuotaRepo.CountActiveLogFiles(ctx)
		if err != nil {
			r.logger.Warn("count active log files for quota", zap.Error(err))
		} else if count > int64(maxCount) {
			oldest, listErr := r.logQuotaRepo.ListOldestActiveLogFiles(ctx, int(count-int64(maxCount)))
			if listErr != nil {
				r.logger.Warn("list oldest log files for quota", zap.Error(listErr))
			} else {
				r.removeQuotaLogFiles(ctx, oldest)
			}
		}
	}
}

func (r *FilePathRecorder) enforceSerialLogFileQuota(ctx context.Context, serialNumber string) {
	if serialNumber == "" {
		return
	}
	if maxPerDevice := r.logQuotaPolicy.MaxFileCountPerDevice(ctx); maxPerDevice > 0 {
		count, err := r.logQuotaRepo.CountActiveLogFilesBySerial(ctx, serialNumber)
		if err != nil {
			r.logger.Warn("count active log files by serial for quota",
				zap.String("device_sn", serialNumber), zap.Error(err))
		} else if count > int64(maxPerDevice) {
			oldest, listErr := r.logQuotaRepo.ListOldestActiveLogFilesBySerial(ctx, serialNumber, int(count-int64(maxPerDevice)))
			if listErr != nil {
				r.logger.Warn("list oldest log files by serial for quota",
					zap.String("device_sn", serialNumber), zap.Error(listErr))
			} else {
				r.removeQuotaLogFiles(ctx, oldest)
			}
		}
	}
}

func (r *FilePathRecorder) removeQuotaLogFiles(ctx context.Context, files []BackupRestoreFile) {
	for _, old := range files {
		if r.logQuotaRemover != nil && old.ObjectPath != "" {
			bucket, objectPath, splitErr := SplitBucketAndPath(old.ObjectPath)
			if splitErr != nil {
				r.logger.Warn("split log file object path for quota",
					zap.Int64("id", old.ID),
					zap.String("path", old.ObjectPath),
					zap.Error(splitErr))
				continue
			} else if removeErr := r.logQuotaRemover.RemoveObject(ctx, bucket, objectPath, minio.RemoveObjectOptions{}); removeErr != nil {
				r.logger.Warn("remove log file object for quota",
					zap.Int64("id", old.ID),
					zap.String("path", old.ObjectPath),
					zap.Error(removeErr))
				continue
			}
		} else {
			r.logger.Warn("skip quota metadata deletion because minio remover or object path is missing",
				zap.Int64("id", old.ID),
				zap.String("path", old.ObjectPath))
			continue
		}
		if markErr := r.logQuotaRepo.MarkFileDeleted(ctx, old.ID); markErr != nil {
			r.logger.Warn("mark log file metadata deleted for quota",
				zap.Int64("id", old.ID),
				zap.String("path", old.ObjectPath),
				zap.Error(markErr))
		} else {
			r.logger.Info("log file quota: removed old file",
				zap.Int64("id", old.ID),
				zap.String("device_sn", old.SerialNumber),
				zap.String("path", old.ObjectPath))
		}
	}
}

func isQuotaManagedLogObjectPath(path string) bool {
	return strings.Contains(path, "/fault/") || strings.HasPrefix(path, "fault/")
}

// promoteSnapshot 把刚落地的备份文件 promote 到 config_snapshots（B3）。
// 失败仅 warn + metric，不返回 error —— 快照表是辅助索引，故障不应阻塞
// 主备份链路。promoter 未注入时静默跳过（向后兼容）。
func (r *FilePathRecorder) promoteSnapshot(ctx context.Context, taskID uuid.UUID, deviceSN string) {
	if r.snapshotPromoter == nil {
		return
	}
	if deviceSN == "" {
		return
	}
	if err := r.snapshotPromoter.PromoteFromBackup(ctx, taskID, deviceSN); err != nil {
		r.metrics.RecordSnapshotPromote("failed")
		r.logger.Warn("promote config_snapshot failed (best-effort)",
			zap.String("task_id", taskID.String()),
			zap.String("device_sn", deviceSN),
			zap.Error(err))
		return
	}
	r.metrics.RecordSnapshotPromote("success")
}

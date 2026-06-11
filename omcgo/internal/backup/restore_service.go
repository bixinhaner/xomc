// Package backup — restore orchestration service (T-0072).
//
// RestoreService accepts a request {bucket, object_path, target_device_sns}
// and fans out to per-device CWMP Download(FileType=3) device tasks via the
// shared task queue. Each restore creates one row in `restore_tasks` and
// N rows in `device_tasks` (where N = len(target_device_sns) excluding any
// devices that the device repo cannot find).
//
// Path inputs are validated (no traversal, restricted to the config_backup
// bucket by default) before any device task is enqueued, so a 400 surfaces
// before any side effect.
package backup

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	pathpkg "path"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"go.uber.org/zap"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
	devtask "github.com/omcgo/omcgo/internal/task"
)

// DeviceLookup is the narrow contract RestoreService needs from the device
// repository — just "given a serial number, does this device exist?". Defined
// at the consumer per Go convention; *device.PgDeviceRepository satisfies it.
type DeviceLookup interface {
	GetBySerialNumber(ctx context.Context, sn string) (*model.Device, error)
}

// TaskCreator is the narrow contract RestoreService needs from the task
// service — just "enqueue a device task". *task.TaskService satisfies it via
// the broader Enqueuer interface; declared narrowly here per consumer-side
// interface convention to keep test mocks small.
type TaskCreator interface {
	CreateTask(ctx context.Context, req *devtask.CreateTaskRequest) (*devtask.Task, error)
}

// CanonicalRestoreBucket is the only bucket allowed as a restore source by
// default. Future expansion (e.g. allow firmware/* for image rollback) would
// require an explicit allow-list extension on the service.
const CanonicalRestoreBucket = "config_backup"

// MinIOStater abstracts the bucket/object existence check. *minio.Client
// satisfies this via StatObject; tests inject a fake.
type MinIOStater interface {
	StatObject(ctx context.Context, bucket, object string, opts minio.StatObjectOptions) (minio.ObjectInfo, error)
}

// RestoreObjectReader abstracts streaming an object's bytes so RestoreService
// can compute the Download MD5 at dispatch time by reading the file content.
// 需求：配置文件恢复在下发时读取文件流现算 MD5（区别于固件/license 的上传期 MD5）。
// *minio.Object 实现 io.ReadCloser，NewMinIOObjectReader 包装真实 client；
// 测试注入返回 bytes 的 fake。
type RestoreObjectReader interface {
	GetObjectStream(ctx context.Context, bucket, object string) (io.ReadCloser, error)
}

// minioObjectReader adapts *minio.Client to RestoreObjectReader.
type minioObjectReader struct{ c *minio.Client }

// NewMinIOObjectReader wraps a *minio.Client so RestoreService can stream
// objects for MD5 computation. Wired in cmd/app/provider/modules.go.
func NewMinIOObjectReader(c *minio.Client) RestoreObjectReader { return minioObjectReader{c: c} }

func (r minioObjectReader) GetObjectStream(ctx context.Context, bucket, object string) (io.ReadCloser, error) {
	obj, err := r.c.GetObject(ctx, bucket, object, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	return obj, nil
}

// BackupTaskFinder is the narrow contract RestoreService needs to fetch the
// originating backup_task for `restore_by_task_id` mode (T-0079). The full
// TaskRepository satisfies it; declared narrow here to keep test mocks small.
type BackupTaskFinder interface {
	GetByID(ctx context.Context, id uuid.UUID) (*BackupTask, error)
}

// SnapshotLookup is the narrow contract RestoreService needs for
// `restore_by_snapshot` mode (T-0164 / B5): batch lookup latest snapshot rows
// by SN. *SnapshotService satisfies this via BatchGetBySerialNumbers.
type SnapshotLookup interface {
	BatchGetBySerialNumbers(ctx context.Context, sns []string) (map[string]*ConfigSnapshot, error)
}

// SnapshotRestoreSourcePlaceholder 标记 restore_tasks.source_object_path
// 在按快照恢复模式下的占位串。每个 device_task 的 Download URL 各自指向
// 对应 SN 的真实 snapshot 文件，但 restore_tasks 是单行聚合 —— 列表展示
// 时前端见到该串就可识别"按设备快照"模式。
const SnapshotRestoreSourcePlaceholder = "(per-device-latest)"

// RestoreService orchestrates restore_task creation + device task fan-out.
type RestoreService struct {
	repo       RestoreTaskRepository
	deviceRepo DeviceLookup
	taskSvc    TaskCreator
	stater     MinIOStater
	metrics    *RestoreMetrics
	logger     *zap.Logger
	// T-0079: optional — when wired enables POST /backup/restore/by-task-id.
	// nil-safe: nil disables the endpoint (handler returns 503).
	backupTaskFinder BackupTaskFinder
	// T-0164 B5: optional — when wired enables POST /backup/restore/by-snapshot.
	snapshotLookup SnapshotLookup
	// objReader: optional — streams source object bytes to compute the Download
	// MD5 at dispatch time. nil → MD5 is left empty (unit tests that don't wire
	// it still pass); wired with the real *minio.Client in production.
	objReader RestoreObjectReader
}

// NewRestoreService wires the dependencies. metrics may be nil (Record* nil-safe).
func NewRestoreService(
	repo RestoreTaskRepository,
	deviceRepo DeviceLookup,
	taskSvc TaskCreator,
	stater MinIOStater,
	metrics *RestoreMetrics,
	logger *zap.Logger,
) *RestoreService {
	return &RestoreService{
		repo:       repo,
		deviceRepo: deviceRepo,
		taskSvc:    taskSvc,
		stater:     stater,
		metrics:    metrics,
		logger:     logger.Named("backup-restore"),
	}
}

// SetBackupTaskFinder enables the by-task-id restore mode (T-0079) without
// changing NewRestoreService's signature. Pass nil to disable.
func (s *RestoreService) SetBackupTaskFinder(finder BackupTaskFinder) {
	s.backupTaskFinder = finder
}

// SetSnapshotLookup enables the by-snapshot restore mode (T-0164 B5).
func (s *RestoreService) SetSnapshotLookup(lookup SnapshotLookup) {
	s.snapshotLookup = lookup
}

// SetObjectReader wires the source-object streamer used to compute the Download
// MD5 at dispatch time (配置文件恢复读流现算 MD5). Pass nil to disable (MD5 empty).
func (s *RestoreService) SetObjectReader(r RestoreObjectReader) {
	s.objReader = r
}

// computeSourceMD5 streams the source object and returns its lowercase-hex MD5.
// 这是"配置文件恢复在下发时读取文件流计算 MD5"的实现点。objReader 未注入时返回
// 空串（不报错），让未装配该依赖的单元测试照常通过；生产路径已在 modules.go 注入。
//
// 兜底翻译：当 stater 存在性预检被跳过（s.stater 为 nil 或预检漏判）时，缺失对象
// 会在此处的 GetObject 流上首次 Read 时暴露为 MinIO NoSuchKey/NoSuchBucket。把它
// 翻译为 commonerrors.ErrNotFound，让 handler 映射 404 而非 default 500。
func (s *RestoreService) computeSourceMD5(ctx context.Context, bucket, object string) (string, error) {
	if s.objReader == nil {
		return "", nil
	}
	rc, err := s.objReader.GetObjectStream(ctx, bucket, object)
	if err != nil {
		if nfErr := translateMinIONotFound(bucket, object, err); nfErr != nil {
			return "", nfErr
		}
		return "", fmt.Errorf("open %s/%s for md5: %w", bucket, object, err)
	}
	defer rc.Close()
	h := md5.New()
	// minio-go 的 GetObject 不立即发请求；对象/桶不存在的错误在首次 Read 时才暴露，
	// 故 NotFound 翻译要兜在 io.Copy 的返回上（不止 GetObjectStream 的返回）。
	if _, err := io.Copy(h, rc); err != nil {
		if nfErr := translateMinIONotFound(bucket, object, err); nfErr != nil {
			return "", nfErr
		}
		return "", fmt.Errorf("read %s/%s for md5: %w", bucket, object, err)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// translateMinIONotFound 把 MinIO 的 NoSuchKey/NoSuchBucket 错误翻译为
// commonerrors.ErrNotFound（不外泄裸 SDK 错误细节给客户端，仅保留 bucket/object
// 路径上下文）；其它错误返回 nil（让调用方按内部错误处理）。与 policy_monitor.go
// 的 isObjectNotFound 同款判定，避免 if 字符串硬编码。
func translateMinIONotFound(bucket, object string, err error) error {
	if err == nil {
		return nil
	}
	resp := minio.ToErrorResponse(err)
	if resp.Code == "NoSuchKey" || resp.Code == "NoSuchBucket" {
		return fmt.Errorf("source object %s/%s: %w", bucket, object, commonerrors.ErrNotFound)
	}
	return nil
}

// CreateRestoreRequest is the API request body validated and persisted.
type CreateRestoreRequest struct {
	Bucket          string   `json:"bucket" binding:"required"`
	ObjectPath      string   `json:"object_path" binding:"required"`
	TargetDeviceSNs []string `json:"target_device_sns" binding:"required,min=1"`
}

// Create validates and enqueues a restore. Caller may pass createdBy="" for
// system-triggered restores.
func (s *RestoreService) Create(ctx context.Context, req *CreateRestoreRequest, createdBy string) (*RestoreTask, error) {
	if req == nil {
		return nil, fmt.Errorf("nil request: %w", commonerrors.ErrInvalidInput)
	}
	if err := validateRestorePath(req.Bucket, req.ObjectPath); err != nil {
		s.metrics.RecordRequest("rejected_invalid_path")
		return nil, err
	}
	if len(req.TargetDeviceSNs) == 0 {
		s.metrics.RecordRequest("rejected_no_target")
		return nil, fmt.Errorf("at least one target device required: %w", commonerrors.ErrInvalidInput)
	}

	// Verify the source object exists before fanning out (avoids creating a
	// restore_task with N device tasks pointing at a 404 URL). 当 stater 未注入
	// 时（DI 漏装或 typed-nil），这里跳过，由 computeSourceMD5 的 GetObject 兜底
	// 翻译 NotFound → 404，不会落到 default 500。
	if s.stater != nil {
		if _, err := s.stater.StatObject(ctx, req.Bucket, req.ObjectPath, minio.StatObjectOptions{}); err != nil {
			if nfErr := translateMinIONotFound(req.Bucket, req.ObjectPath, err); nfErr != nil {
				s.metrics.RecordRequest("rejected_object_not_found")
				return nil, nfErr
			}
			return nil, fmt.Errorf("stat source object: %w", err)
		}
	}

	// 配置文件恢复：下发时读取源文件流现算 MD5（Download 报文必填，供 CPE 下载后校验）。
	// 放在建 restore_task 行之前 → md5 失败不留孤儿行。同一份文件发给所有目标设备，
	// 故循环外只算一次。
	srcMD5, md5Err := s.computeSourceMD5(ctx, req.Bucket, req.ObjectPath)
	if md5Err != nil {
		return nil, fmt.Errorf("compute restore source md5: %w", md5Err)
	}

	now := time.Now()
	created := &RestoreTask{
		SourceBucket:     req.Bucket,
		SourceObjectPath: req.ObjectPath,
		TargetDeviceSNs:  req.TargetDeviceSNs,
		Status:           RestorePending,
		Progress:         0,
		StartedAt:        &now,
	}
	if createdBy != "" {
		cb := createdBy
		created.CreatedBy = &cb
	}
	if err := s.repo.Create(ctx, created); err != nil {
		return nil, fmt.Errorf("create restore_task: %w", err)
	}

	// Fan out: enqueue one Download device task per (existing) device. Missing
	// SNs are recorded in error_message JSON so the operator sees what was
	// skipped without an aggregate failure.
	restoreURL := req.Bucket + "/" + req.ObjectPath
	skipped := make([]string, 0)
	enqueued := 0
	for _, sn := range req.TargetDeviceSNs {
		dev, err := s.deviceRepo.GetBySerialNumber(ctx, sn)
		if err != nil || dev == nil {
			skipped = append(skipped, sn)
			s.logger.Warn("device not found for restore; skipping",
				zap.String("device_sn", sn),
				zap.Error(err))
			continue
		}
		params, err := json.Marshal(map[string]interface{}{
			// FileType = "10 <OUI> Configuration File"：厂商私有配置文件下行格式，
			// OUI 必须替换为设备真实 OUI（如 48BF74 Baicells），否则 CPE 不识别。
			"file_type":        s.buildRestoreFileType(dev.SerialNumber, dev.OUI),
			"url":              restoreURL,
			"target_file_name": pathpkg.Base(req.ObjectPath),
			"md5":              srcMD5,
		})
		if err != nil {
			// json.Marshal of a map[string]any with primitive values cannot
			// fail in practice, but propagate any error rather than swallow.
			return nil, fmt.Errorf("marshal Download params: %w", err)
		}
		if _, err := s.taskSvc.CreateTask(ctx, &devtask.CreateTaskRequest{
			DeviceSN:  dev.SerialNumber,
			Method:    "Download",
			Params:    params,
			Source:    devtask.TaskSourceSystem,
			SourceID:  created.ID.String(),
			CreatorID: createdBy,
			// M2: 规范化 Download CommandKey 为 `{cellCode}_RESTORE_{taskID8}`，
			// 由 TransferCompleteRouter 反解定位 restore_tasks 行。
			CommandKey: BuildRestoreCommandKey(dev.SiteID, dev.SerialNumber, created.ID.String()),
		}); err != nil {
			skipped = append(skipped, sn)
			s.logger.Warn("enqueue Download device task failed",
				zap.String("device_sn", sn),
				zap.Error(err))
			continue
		}
		enqueued++
	}
	s.metrics.RecordRequest("accepted")
	s.metrics.RecordDevicesEnqueued(int64(enqueued))

	if len(skipped) > 0 {
		msg := "skipped: " + strings.Join(skipped, ",")
		created.ErrorMessage = &msg
		if updErr := s.updateSkipped(ctx, created); updErr != nil {
			s.logger.Warn("update restore_task skipped notes failed",
				zap.String("restore_id", created.ID.String()),
				zap.Error(updErr))
		}
	}
	if enqueued > 0 {
		// Status flips to running once at least one device task is in flight.
		// We don't actually persist the running flip in MVP — progress callbacks
		// are out of scope (PRD N4); a follow-up task may add it.
	}
	s.logger.Info("restore created",
		zap.String("restore_id", created.ID.String()),
		zap.String("bucket", req.Bucket),
		zap.String("path", req.ObjectPath),
		zap.Int("target_count", len(req.TargetDeviceSNs)),
		zap.Int("enqueued", enqueued),
		zap.Int("skipped", len(skipped)))
	return created, nil
}

// updateSkipped persists the skipped device note onto restore_tasks so that a
// later GET /restore-tasks/:id sees the same error_message as the POST
// response (review fix M2 — was previously an in-memory-only mutation).
// Best-effort; logged on failure but does not abort the request.
func (s *RestoreService) updateSkipped(ctx context.Context, t *RestoreTask) error {
	if t.ErrorMessage == nil {
		return nil
	}
	return s.repo.UpdateErrorMessage(ctx, t.ID, *t.ErrorMessage)
}

// CreateByTaskIDRequest is the API body for `POST /backup/restore/by-task-id`.
type CreateByTaskIDRequest struct {
	BackupTaskID    uuid.UUID `json:"backup_task_id" binding:"required"`
	TargetDeviceSNs []string  `json:"target_device_sns" binding:"required,min=1"`
}

// CreateByTaskIDResult bundles the created RestoreTask with an optional
// human-readable warning (e.g. multi-device backup caveat) so the handler
// can include it in the response body without inflating RestoreTask itself.
type CreateByTaskIDResult struct {
	Task    *RestoreTask `json:"task"`
	Warning *string      `json:"warning,omitempty"`
}

// CreateByTaskID resolves backup_tasks.file_path for the given task and
// dispatches a restore (T-0079). Returns ErrNotFound when the backup_task
// hasn't uploaded yet (file_path is null) so the handler can surface 404.
func (s *RestoreService) CreateByTaskID(
	ctx context.Context,
	req *CreateByTaskIDRequest,
	createdBy string,
) (*CreateByTaskIDResult, error) {
	if s.backupTaskFinder == nil {
		return nil, fmt.Errorf("by-task-id mode not configured: %w", commonerrors.ErrInvalidInput)
	}
	if req == nil {
		s.metrics.RecordRestoreByTask("rejected_invalid_input")
		return nil, fmt.Errorf("nil request: %w", commonerrors.ErrInvalidInput)
	}
	if len(req.TargetDeviceSNs) == 0 {
		s.metrics.RecordRestoreByTask("rejected_invalid_input")
		return nil, fmt.Errorf("at least one target device required: %w", commonerrors.ErrInvalidInput)
	}
	bt, err := s.backupTaskFinder.GetByID(ctx, req.BackupTaskID)
	if err != nil {
		if errors.Is(err, commonerrors.ErrNotFound) {
			s.metrics.RecordRestoreByTask("rejected_not_uploaded")
			return nil, err
		}
		return nil, fmt.Errorf("lookup backup_task %s: %w", req.BackupTaskID, err)
	}
	if bt.FilePath == nil || *bt.FilePath == "" {
		s.metrics.RecordRestoreByTask("rejected_not_uploaded")
		return nil, fmt.Errorf("backup_task %s file_path not yet recorded; CPE may not have uploaded yet: %w",
			req.BackupTaskID, commonerrors.ErrNotFound)
	}
	bucket, objectPath, err := splitBucketAndPath(*bt.FilePath)
	if err != nil {
		return nil, fmt.Errorf("parse backup_task.file_path %q: %w", *bt.FilePath, err)
	}
	rt, err := s.Create(ctx, &CreateRestoreRequest{
		Bucket:          bucket,
		ObjectPath:      objectPath,
		TargetDeviceSNs: req.TargetDeviceSNs,
	}, createdBy)
	if err != nil {
		return nil, err
	}

	result := &CreateByTaskIDResult{Task: rt}
	if len(bt.TargetIDs) > 1 {
		w := fmt.Sprintf("multi-device backup (target_count=%d); restore uses only the first device's config (first-write-wins)",
			len(bt.TargetIDs))
		result.Warning = &w
	}
	s.metrics.RecordRestoreByTask("accepted")
	return result, nil
}

// CreateBySnapshotRequest is the body of POST /backup/restore/by-snapshot.
type CreateBySnapshotRequest struct {
	TargetDeviceSNs []string `json:"target_device_sns" binding:"required,min=1"`
	// UpgradeTaskID（可选）：UFTE 侧提前建好的 upgrade_tasks 占位行 ID。
	// 非 Nil 时 device_tasks 的 CommandKey 用 "CONFIG_RESTORE_<id8>_<sn>" 格式
	// （与 software.BuildDirectDispatchCommandKey 对齐），保证 TC 回流可命中 sub_task。
	// Nil 时退化用 restore_tasks.id-derived CommandKey（兼容 /backup/restore/by-snapshot 旧入口）。
	UpgradeTaskID uuid.UUID `json:"-"`
}

// CreateBySnapshotResult bundles the created RestoreTask with the missing-SN
// list (when integral rejection happens) so the handler can return a single
// shape regardless of acceptance outcome.
type CreateBySnapshotResult struct {
	Task    *RestoreTask `json:"task,omitempty"`
	Missing []string     `json:"missing,omitempty"`
	// DispatchedFiles 是每台设备实际下发的快照文件 basename。UFTE 用作"目标文件"列
	// 展示数据来源。仅成功 enqueue 的设备进 map；整批拒绝时为 nil。
	DispatchedFiles map[string]string `json:"dispatched_files,omitempty"`
}

// CreateBySnapshot orchestrates a restore where each target device picks its
// **own** latest config_snapshots row as the source. Integral rejection
// semantics: if any target SN has no snapshot row, the entire batch is
// rejected and the missing SNs are returned to the caller — avoids partial
// success ambiguity ("which devices actually restored").
//
// One restore_tasks row is persisted (source_bucket=config-snapshots,
// source_object_path=SnapshotRestoreSourcePlaceholder). Per-device Download
// commands point at the device's own snapshot key.
func (s *RestoreService) CreateBySnapshot(
	ctx context.Context,
	req *CreateBySnapshotRequest,
	createdBy string,
) (*CreateBySnapshotResult, error) {
	if s.snapshotLookup == nil {
		return nil, fmt.Errorf("by-snapshot mode not configured: %w", commonerrors.ErrInvalidInput)
	}
	if req == nil || len(req.TargetDeviceSNs) == 0 {
		s.metrics.RecordRestoreBySnapshot("rejected_invalid_input")
		return nil, fmt.Errorf("at least one target device required: %w", commonerrors.ErrInvalidInput)
	}

	snaps, err := s.snapshotLookup.BatchGetBySerialNumbers(ctx, req.TargetDeviceSNs)
	if err != nil {
		return nil, fmt.Errorf("lookup config_snapshots: %w", err)
	}
	missing := make([]string, 0)
	for _, sn := range req.TargetDeviceSNs {
		if _, ok := snaps[sn]; !ok {
			missing = append(missing, sn)
		}
	}
	if len(missing) > 0 {
		s.metrics.RecordRestoreBySnapshot("rejected_missing_snapshot")
		// Note: we return missing in the result *and* propagate ErrNotFound so
		// the handler can choose 400/404 mapping; the body carries the list.
		return &CreateBySnapshotResult{Missing: missing},
			fmt.Errorf("the following devices have no config snapshot: %v: %w",
				missing, commonerrors.ErrNotFound)
	}

	// All targets have snapshots — persist one restore_tasks row, then fan out.
	now := time.Now()
	bucket := SnapshotBucketDefault
	if first, ok := snaps[req.TargetDeviceSNs[0]]; ok && first.ObjectBucket != "" {
		bucket = first.ObjectBucket
	}
	created := &RestoreTask{
		SourceBucket:     bucket,
		SourceObjectPath: SnapshotRestoreSourcePlaceholder,
		TargetDeviceSNs:  req.TargetDeviceSNs,
		Status:           RestorePending,
		Progress:         0,
		StartedAt:        &now,
	}
	if createdBy != "" {
		cb := createdBy
		created.CreatedBy = &cb
	}
	if err := s.repo.Create(ctx, created); err != nil {
		return nil, fmt.Errorf("create restore_task: %w", err)
	}

	// UpgradeTaskID 非 Nil 时派生 "CONFIG_RESTORE_<id8>_<sn>" CommandKey
	// （与 software.BuildDirectDispatchCommandKey 对齐，让 TC 回流能命中 UFTE sub_task）。
	// 这条 short id 提前算一次复用。
	upgradeTidShort := ""
	if req.UpgradeTaskID != uuid.Nil {
		s := strings.ReplaceAll(req.UpgradeTaskID.String(), "-", "")
		if len(s) >= 8 {
			s = s[:8]
		}
		upgradeTidShort = s
	}

	enqueued := 0
	skipped := make([]string, 0)
	// dispatchedFiles 用作"目标文件"列展示数据来源；仅 enqueue 成功的设备进 map。
	dispatchedFiles := make(map[string]string, len(req.TargetDeviceSNs))
	for _, sn := range req.TargetDeviceSNs {
		snap := snaps[sn]
		dev, dErr := s.deviceRepo.GetBySerialNumber(ctx, sn)
		if dErr != nil || dev == nil {
			skipped = append(skipped, sn)
			s.logger.Warn("device not found for restore_by_snapshot; skipping",
				zap.String("device_sn", sn), zap.Error(dErr))
			continue
		}
		restoreURL := snap.ObjectBucket + "/" + snap.ObjectPath
		targetFileName := pathpkg.Base(snap.ObjectPath)
		// 下发时读该设备快照文件流现算 MD5（Download 报文必填）。每设备文件不同，
		// 故循环内逐个算；算失败跳过该设备，避免下发缺 MD5 的 Download。
		snapMD5, md5Err := s.computeSourceMD5(ctx, snap.ObjectBucket, snap.ObjectPath)
		if md5Err != nil {
			skipped = append(skipped, sn)
			s.logger.Warn("compute snapshot md5 failed; skipping",
				zap.String("device_sn", sn), zap.Error(md5Err))
			continue
		}
		params, mErr := json.Marshal(map[string]interface{}{
			"file_type":        s.buildRestoreFileType(dev.SerialNumber, dev.OUI),
			"url":              restoreURL,
			"target_file_name": targetFileName,
			"md5":              snapMD5,
		})
		if mErr != nil {
			return nil, fmt.Errorf("marshal Download params for %s: %w", sn, mErr)
		}
		commandKey := BuildRestoreCommandKey(dev.SiteID, dev.SerialNumber, created.ID.String())
		if upgradeTidShort != "" {
			commandKey = fmt.Sprintf("CONFIG_RESTORE_%s_%s", upgradeTidShort, dev.SerialNumber)
		}
		if _, qErr := s.taskSvc.CreateTask(ctx, &devtask.CreateTaskRequest{
			DeviceSN:   dev.SerialNumber,
			Method:     "Download",
			Params:     params,
			Source:     devtask.TaskSourceSystem,
			SourceID:   created.ID.String(),
			CreatorID:  createdBy,
			CommandKey: commandKey,
		}); qErr != nil {
			skipped = append(skipped, sn)
			s.logger.Warn("enqueue Download device task failed (by-snapshot)",
				zap.String("device_sn", sn), zap.Error(qErr))
			continue
		}
		dispatchedFiles[dev.SerialNumber] = targetFileName
		enqueued++
	}
	s.metrics.RecordRestoreBySnapshot("accepted")
	s.metrics.RecordDevicesEnqueued(int64(enqueued))

	if len(skipped) > 0 {
		msg := "skipped: " + strings.Join(skipped, ",")
		created.ErrorMessage = &msg
		if updErr := s.updateSkipped(ctx, created); updErr != nil {
			s.logger.Warn("update restore_task skipped notes failed (by-snapshot)",
				zap.String("restore_id", created.ID.String()),
				zap.Error(updErr))
		}
	}

	s.logger.Info("restore created (by-snapshot)",
		zap.String("restore_id", created.ID.String()),
		zap.Int("target_count", len(req.TargetDeviceSNs)),
		zap.Int("enqueued", enqueued),
		zap.Int("skipped", len(skipped)))
	return &CreateBySnapshotResult{Task: created, DispatchedFiles: dispatchedFiles}, nil
}

// SplitBucketAndPath 是 splitBucketAndPath 的导出别名，供 cmd/app/provider 等
// 外部包构造 MinIO presigned URL 时复用同一份解析逻辑（B5）。
func SplitBucketAndPath(combined string) (bucket, objectPath string, err error) {
	return splitBucketAndPath(combined)
}

// restoreFileTypeFallbackOUI：设备未上报 OUI 时下发 Download 用的兜底 OUI。
// 与 software/executor.go 的 fallbackOUI 同值（Baicells 48BF74）——OMC 当前主流
// 设备来自 Baicells，CPE 拿到形如 "10 48BF74 Configuration File" 的 FileType
// 才能正确识别。接其它厂商时务必保证设备注册时填了真实 OUI。
const restoreFileTypeFallbackOUI = "48BF74"

// buildRestoreFileType 渲染配置恢复 Download RPC 的 FileType 字符串，格式：
//
//	10 <OUI> Configuration File
//
// 其中 OUI 取设备真实 OUI；为空时退化到 restoreFileTypeFallbackOUI 并 warn 一次。
func (s *RestoreService) buildRestoreFileType(deviceSN, deviceOUI string) string {
	oui := deviceOUI
	if oui == "" {
		s.logger.Warn("device OUI not populated; falling back for Download FileType",
			zap.String("device_sn", deviceSN),
			zap.String("fallback_oui", restoreFileTypeFallbackOUI))
		oui = restoreFileTypeFallbackOUI
	}
	return fmt.Sprintf("10 %s Configuration File", oui)
}

// DispatchConfigRestoreBySnapshot 实现 ufte.SnapshotConfigRestoreDispatcher（T-0164）。
//
// 让 UFTE 的 CONFIG_RESTORE 任务复用 CreateBySnapshot 的整批拒绝 + 逐设备 URL 派发
// 语义，无需在 ufte 包反向 import backup。
//
// 返回：
//   - dispatchedID：restore_tasks 主行 UUID
//   - missing：缺失快照的 SN 列表（非空时同时返回 ErrNotFound 包装错误）
//   - err：基础设施错误
func (s *RestoreService) DispatchConfigRestoreBySnapshot(
	ctx context.Context, sns []string, createUser string, upgradeTaskID uuid.UUID,
) (dispatchedID uuid.UUID, dispatchedFiles map[string]string, missing []string, err error) {
	result, err := s.CreateBySnapshot(ctx,
		&CreateBySnapshotRequest{TargetDeviceSNs: sns, UpgradeTaskID: upgradeTaskID}, createUser)
	if err != nil {
		// CreateBySnapshot 在整批拒绝时同时返回 result + ErrNotFound 包装错误。
		if result != nil && len(result.Missing) > 0 {
			return uuid.Nil, nil, result.Missing, err
		}
		return uuid.Nil, nil, nil, err
	}
	if result == nil || result.Task == nil {
		return uuid.Nil, nil, nil, fmt.Errorf("CreateBySnapshot returned nil result")
	}
	return result.Task.ID, result.DispatchedFiles, nil, nil
}

// PreviewConfigRestoreFiles 不创建 restore_task / device_task，只返回 sn → 预期下发的
// snapshot 文件名（basename），给 UFTE 挂起 / 定时占位任务在创建时写 sub_task.dest_version 用。
// 整批拒绝语义同 CreateBySnapshot。
func (s *RestoreService) PreviewConfigRestoreFiles(
	ctx context.Context, sns []string,
) (map[string]string, error) {
	if s.snapshotLookup == nil {
		return nil, fmt.Errorf("by-snapshot mode not configured: %w", commonerrors.ErrInvalidInput)
	}
	if len(sns) == 0 {
		return nil, fmt.Errorf("at least one target device required: %w", commonerrors.ErrInvalidInput)
	}
	snaps, err := s.snapshotLookup.BatchGetBySerialNumbers(ctx, sns)
	if err != nil {
		return nil, fmt.Errorf("lookup config_snapshots (preview): %w", err)
	}
	missing := make([]string, 0)
	for _, sn := range sns {
		if _, ok := snaps[sn]; !ok {
			missing = append(missing, sn)
		}
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("the following devices have no config snapshot: %v: %w",
			missing, commonerrors.ErrNotFound)
	}
	out := make(map[string]string, len(sns))
	for _, sn := range sns {
		out[sn] = pathpkg.Base(snaps[sn].ObjectPath)
	}
	return out, nil
}

// splitBucketAndPath splits a "bucket/path/to/object" string into its parts.
// Returns ErrInvalidInput when the format is unexpected (no '/' separator).
func splitBucketAndPath(combined string) (bucket, objectPath string, err error) {
	idx := strings.IndexByte(combined, '/')
	if idx <= 0 || idx >= len(combined)-1 {
		return "", "", fmt.Errorf("file_path %q missing bucket/path separator: %w",
			combined, commonerrors.ErrInvalidInput)
	}
	return combined[:idx], combined[idx+1:], nil
}

// List proxies to the repo.
func (s *RestoreService) List(ctx context.Context, filter RestoreFilter) (*model.ListResponse[RestoreTask], error) {
	return s.repo.List(ctx, filter)
}

// GetByID proxies to the repo.
func (s *RestoreService) GetByID(ctx context.Context, id uuid.UUID) (*RestoreTask, error) {
	return s.repo.GetByID(ctx, id)
}

// validateRestorePath enforces:
//   - non-empty bucket + object_path
//   - no path traversal (".." / leading "/")
//   - bucket must equal CanonicalRestoreBucket (operators cannot restore from
//     unrelated buckets like firmware/ or pm/)
func validateRestorePath(bucket, objectPath string) error {
	if bucket == "" || objectPath == "" {
		return fmt.Errorf("bucket and object_path required: %w", commonerrors.ErrInvalidInput)
	}
	cleanedBucket := pathpkg.Clean(bucket)
	cleanedPath := pathpkg.Clean(objectPath)
	if strings.Contains(bucket, "..") || strings.Contains(objectPath, "..") ||
		strings.HasPrefix(bucket, "/") || strings.HasPrefix(objectPath, "/") ||
		cleanedBucket != bucket || cleanedPath != objectPath {
		return fmt.Errorf("path traversal or non-canonical path: %w", commonerrors.ErrInvalidInput)
	}
	if bucket != CanonicalRestoreBucket {
		return fmt.Errorf("only %q bucket is allowed for restore: %w",
			CanonicalRestoreBucket, commonerrors.ErrInvalidInput)
	}
	return nil
}

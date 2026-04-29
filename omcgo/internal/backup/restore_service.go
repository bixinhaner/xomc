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
	"encoding/json"
	"errors"
	"fmt"
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

// BackupTaskFinder is the narrow contract RestoreService needs to fetch the
// originating backup_task for `restore_by_task_id` mode (T-0079). The full
// TaskRepository satisfies it; declared narrow here to keep test mocks small.
type BackupTaskFinder interface {
	GetByID(ctx context.Context, id uuid.UUID) (*BackupTask, error)
}

// RestoreService orchestrates restore_task creation + device task fan-out.
type RestoreService struct {
	repo            RestoreTaskRepository
	deviceRepo      DeviceLookup
	taskSvc         TaskCreator
	stater          MinIOStater
	metrics         *RestoreMetrics
	logger          *zap.Logger
	// T-0079: optional — when wired enables POST /backup/restore/by-task-id.
	// nil-safe: nil disables the endpoint (handler returns 503).
	backupTaskFinder BackupTaskFinder
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
	// restore_task with N device tasks pointing at a 404 URL).
	if s.stater != nil {
		if _, err := s.stater.StatObject(ctx, req.Bucket, req.ObjectPath, minio.StatObjectOptions{}); err != nil {
			errResp := minio.ToErrorResponse(err)
			if errResp.Code == "NoSuchKey" || errResp.Code == "NoSuchBucket" {
				s.metrics.RecordRequest("rejected_object_not_found")
				return nil, fmt.Errorf("source object %s/%s: %w", req.Bucket, req.ObjectPath, commonerrors.ErrNotFound)
			}
			return nil, fmt.Errorf("stat source object: %w", err)
		}
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
			"file_type":        "3", // Vendor Configuration File
			"url":              restoreURL,
			"target_file_name": pathpkg.Base(req.ObjectPath),
		})
		if err != nil {
			// json.Marshal of a map[string]any with primitive values cannot
			// fail in practice, but propagate any error rather than swallow.
			return nil, fmt.Errorf("marshal Download params: %w", err)
		}
		if _, err := s.taskSvc.CreateTask(ctx, &devtask.CreateTaskRequest{
			DeviceSN:   dev.SerialNumber,
			Method:     "Download",
			Params:     params,
			Source:     devtask.TaskSourceSystem,
			SourceID:   created.ID.String(),
			CreatorID:  createdBy,
			CommandKey: "restore-" + created.ID.String()[:8],
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


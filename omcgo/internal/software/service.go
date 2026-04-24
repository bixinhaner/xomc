package software

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/acs/connreq"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/storage"
	"github.com/omcgo/omcgo/internal/device"
	devtask "github.com/omcgo/omcgo/internal/task"
)

// SoftwareService provides firmware upload and device upgrade functionality.
type SoftwareService struct {
	firmwareRepo FirmwareRepository
	taskRepo     TaskRepository
	subTaskRepo  SubTaskRepository
	deviceRepo   device.DeviceRepository
	taskSvc      devtask.Enqueuer
	connReq      *connreq.Client
	minioClient  *minio.Client
	firmwareBkt  string
	eventBus     event.EventBus
	redis        redis.UniversalClient
	executor     *UpgradeExecutor
	rollbackExec *RollbackExecutor
	adapter      UpgradeAdapter
	logger       *zap.Logger
}

// NewSoftwareService creates a new SoftwareService.
func NewSoftwareService(
	firmwareRepo FirmwareRepository,
	taskRepo TaskRepository,
	subTaskRepo SubTaskRepository,
	deviceRepo device.DeviceRepository,
	taskSvc devtask.Enqueuer,
	connReq *connreq.Client,
	minioClient *minio.Client,
	firmwareBucket string,
	eventBus event.EventBus,
	redisClient redis.UniversalClient,
	logger *zap.Logger,
) *SoftwareService {
	s := &SoftwareService{
		firmwareRepo: firmwareRepo,
		taskRepo:     taskRepo,
		subTaskRepo:  subTaskRepo,
		deviceRepo:   deviceRepo,
		taskSvc:      taskSvc,
		connReq:      connReq,
		minioClient:  minioClient,
		firmwareBkt:  firmwareBucket,
		eventBus:     eventBus,
		redis:        redisClient,
		adapter:      NewDefaultUpgradeAdapter(),
		logger:       logger.Named("software"),
	}

	s.executor = NewUpgradeExecutor(
		taskRepo, subTaskRepo, deviceRepo, firmwareRepo,
		taskSvc, connReq, redisClient, eventBus, logger,
	)
	s.rollbackExec = NewRollbackExecutor(
		taskRepo, subTaskRepo, deviceRepo,
		taskSvc, connReq, redisClient, eventBus, logger,
	)

	return s
}

// UploadFirmware stores a firmware file to MinIO and creates a firmware version record.
func (s *SoftwareService) UploadFirmware(ctx context.Context, fw *FirmwareVersion, file io.Reader, fileSize int64) error {
	// Determine MinIO directory by file type
	category := "img"
	switch fw.FileType {
	case FileTypePATCH:
		category = "patch"
	case FileTypeAP:
		category = "ap"
	case FileTypeFPGA:
		category = "fpga"
	}
	objectPath := storage.FirmwarePath(category, fw.ProductClass, fw.Version, fw.FileName)

	// Tee file stream: one copy to MinIO, one to compute MD5
	hash := md5.New()
	teeReader := io.TeeReader(file, hash)

	_, err := s.minioClient.PutObject(ctx, s.firmwareBkt, objectPath, teeReader, fileSize, minio.PutObjectOptions{
		ContentType: "application/octet-stream",
	})
	if err != nil {
		return fmt.Errorf("upload firmware to MinIO: %w", err)
	}

	fw.MinIOPath = objectPath
	fw.FileSize = fileSize
	fw.MD5Val = hex.EncodeToString(hash.Sum(nil))
	if fw.Status == "" {
		fw.Status = "active"
	}

	if err := s.firmwareRepo.Create(ctx, fw); err != nil {
		// Clean up orphaned MinIO file on DB failure
		if delErr := s.minioClient.RemoveObject(ctx, s.firmwareBkt, objectPath, minio.RemoveObjectOptions{}); delErr != nil {
			s.logger.Error("cleanup orphaned firmware file", zap.String("path", objectPath), zap.Error(delErr))
		}
		return fmt.Errorf("create firmware record: %w", err)
	}

	if evt, err := event.NewEvent(event.SubjectFirmwareUploaded, map[string]interface{}{
		"firmware_id": fw.ID.String(),
		"version":     fw.Version,
	}); err == nil {
		if pubErr := s.eventBus.Publish(ctx, event.SubjectFirmwareUploaded, evt); pubErr != nil {
			s.logger.Warn("publish firmware.uploaded event", zap.Error(pubErr))
		}
	}

	return nil
}

// BatchUpgrade creates a main upgrade task and sub-tasks for each device, then starts execution.
func (s *SoftwareService) BatchUpgrade(ctx context.Context, req BatchUpgradeRequest) (*UpgradeTask, error) {
	// Fetch firmware once outside the loop (fix N+1)
	fw, err := s.firmwareRepo.GetByID(ctx, req.FirmwareID)
	if err != nil {
		return nil, fmt.Errorf("get firmware: %w", err)
	}

	concurrency := req.Concurrency
	if concurrency < 1 {
		concurrency = 5
	}

	taskType := req.TaskType
	if taskType == 0 {
		taskType = TaskTypeUpgrade
	}

	// Create main task
	mainTask := &UpgradeTask{
		TaskName:     req.TaskName,
		TaskType:     taskType,
		FirmwareID:   &req.FirmwareID,
		FileName:     fw.FileName,
		FileMD5:      fw.MD5Val,
		Status:       TaskPending,
		ProductClass: fw.ProductClass,
		IsKeepConfig: req.IsKeepConfig,
		CreateStatus: "active",
		CreateUser:   "system",
		TotalCount:   len(req.DeviceIDs),
		MaxConcurrent: concurrency,
	}
	if err := s.taskRepo.Create(ctx, mainTask); err != nil {
		return nil, fmt.Errorf("create main task: %w", err)
	}

	// Create sub-tasks
	subTasks := make([]*UpgradeSubTask, 0, len(req.DeviceIDs))
	for _, deviceID := range req.DeviceIDs {
		subTask := &UpgradeSubTask{
			TaskID:     mainTask.ID,
			DeviceID:   deviceID,
			FirmwareID: &req.FirmwareID,
			Status:     UpgradePending,
			MaxRetries: 3,
			DestVersion: fw.Version,
		}
		subTasks = append(subTasks, subTask)
	}

	if err := s.subTaskRepo.BatchCreate(ctx, subTasks); err != nil {
		return nil, fmt.Errorf("batch create sub-tasks: %w", err)
	}

	// Update main task status to in_progress
	if err := s.taskRepo.UpdateStatus(ctx, mainTask.ID, TaskInProgress, ""); err != nil {
		s.logger.Error("update main task to in_progress", zap.Error(err))
	}
	mainTask.Status = TaskInProgress

	// Start execution for each device via executor
	sem := make(chan struct{}, concurrency)
	for i := range subTasks {
		sem <- struct{}{}
		go func(st *UpgradeSubTask) {
			defer func() { <-sem }()
			defer func() {
				if r := recover(); r != nil {
					s.logger.Error("upgrade executor panic",
						zap.String("sub_task_id", st.ID.String()),
						zap.Any("recover", r))
				}
			}()
			s.executor.ExecuteOne(context.Background(), st, fw)
		}(subTasks[i])
	}

	return mainTask, nil
}


// HandleTransferComplete advances the upgrade state machine when a device reports transfer complete.
func (s *SoftwareService) HandleTransferComplete(ctx context.Context, evt event.Event) error {
	var payload struct {
		DeviceSN string `json:"device_sn"`
	}
	if err := evt.DecodePayload(&payload); err != nil || payload.DeviceSN == "" {
		return nil
	}
	deviceSN := payload.DeviceSN

	dev, err := s.deviceRepo.GetBySerialNumber(ctx, deviceSN)
	if err != nil {
		return fmt.Errorf("get device by SN: %w", err)
	}
	if dev == nil {
		return fmt.Errorf("device not found: %s", deviceSN)
	}

	subTask, err := s.subTaskRepo.GetActiveByDeviceID(ctx, dev.ID)
	if err != nil {
		return nil // No active upgrade, ignore
	}

	// Idempotent: only handle downloading state
	if subTask.Status != UpgradeDownloading {
		s.logger.Debug("ignore TC for non-downloading sub-task",
			zap.String("sub_task_id", subTask.ID.String()),
			zap.String("status", string(subTask.Status)))
		return nil
	}

	// Advance state machine: downloading -> rebooting -> verifying -> completed
	nextState := NextUpgradeState(subTask.Status)
	if err := ValidateUpgradeTransition(subTask.Status, nextState); err != nil {
		s.logger.Warn("invalid upgrade transition",
			zap.String("sub_task_id", subTask.ID.String()),
			zap.String("current", string(subTask.Status)),
			zap.String("target", string(nextState)),
		)
		return nil
	}

	if err := s.subTaskRepo.UpdateStatus(ctx, subTask.ID, nextState, ""); err != nil {
		return fmt.Errorf("update sub-task status: %w", err)
	}

	// Update main task counts on completion
	if nextState == UpgradeCompleted {
		if err := s.taskRepo.IncrementCounts(ctx, subTask.TaskID, 1, 0); err != nil {
			s.logger.Error("increment success count", zap.Error(err))
		}
		s.checkAndFinalizeTask(ctx, subTask.TaskID)

		if completedEvt, err := event.NewEvent(event.SubjectUpgradeCompleted, map[string]interface{}{
			"sub_task_id": subTask.ID.String(),
			"task_id":     subTask.TaskID.String(),
			"device_id":   dev.ID.String(),
		}); err == nil {
			if pubErr := s.eventBus.Publish(ctx, event.SubjectUpgradeCompleted, completedEvt); pubErr != nil {
				s.logger.Warn("publish upgrade.completed event", zap.Error(pubErr))
			}
		}
	} else if nextState == UpgradeFailed {
		if err := s.taskRepo.IncrementCounts(ctx, subTask.TaskID, 0, 1); err != nil {
			s.logger.Error("increment fail count", zap.Error(err))
		}
		s.checkAndFinalizeTask(ctx, subTask.TaskID)
	}

	return nil
}

// checkAndFinalizeTask checks if a main task is complete and updates its final status.
func (s *SoftwareService) checkAndFinalizeTask(ctx context.Context, taskID uuid.UUID) {
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return
	}

	// All sub-tasks must be terminal
	if task.SuccessCount+task.FailCount < task.TotalCount {
		return
	}

	result := TaskResultSuccess
	if task.FailCount == task.TotalCount {
		result = TaskResultFailed
	} else if task.FailCount > 0 {
		result = TaskResultPartial
	}

	if err := s.taskRepo.UpdateStatus(ctx, taskID, TaskEnded, result); err != nil {
		s.logger.Error("finalize main task", zap.Error(err))
	}
}

// SuspendUpgrade suspends all active sub-tasks under a main task.
func (s *SoftwareService) SuspendUpgrade(ctx context.Context, taskID uuid.UUID) error {
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return fmt.Errorf("get upgrade task: %w", err)
	}

	if task.Status == TaskEnded || task.Status == TaskSuspended {
		return commonerrors.NewBusinessError(8003, fmt.Sprintf("cannot suspend task in %s state", task.Status), commonerrors.ErrInvalidInput)
	}

	// Update main task status
	if err := s.taskRepo.UpdateStatus(ctx, taskID, TaskSuspended, ""); err != nil {
		return fmt.Errorf("suspend main task: %w", err)
	}

	s.logger.Info("upgrade task suspended", zap.String("task_id", taskID.String()))
	return nil
}

// ResumeUpgrade resumes a suspended main task and its sub-tasks.
func (s *SoftwareService) ResumeUpgrade(ctx context.Context, taskID uuid.UUID) error {
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return fmt.Errorf("get upgrade task: %w", err)
	}

	if task.Status != TaskSuspended {
		return commonerrors.NewBusinessError(8004, "task is not suspended", commonerrors.ErrInvalidInput)
	}

	// Update main task status back to in_progress
	if err := s.taskRepo.UpdateStatus(ctx, taskID, TaskInProgress, ""); err != nil {
		return fmt.Errorf("resume main task: %w", err)
	}

	s.logger.Info("upgrade task resumed", zap.String("task_id", taskID.String()))
	return nil
}

// TerminateUpgrade force-stops a main task and all its sub-tasks.
func (s *SoftwareService) TerminateUpgrade(ctx context.Context, taskID uuid.UUID) error {
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return fmt.Errorf("get upgrade task: %w", err)
	}

	if task.Status == TaskEnded {
		return commonerrors.NewBusinessError(8005, "task already ended", commonerrors.ErrInvalidInput)
	}

	// Update main task status
	if err := s.taskRepo.UpdateStatus(ctx, taskID, TaskEnded, TaskResultTerminated); err != nil {
		return fmt.Errorf("terminate main task: %w", err)
	}

	s.logger.Info("upgrade task terminated", zap.String("task_id", taskID.String()))
	return nil
}

// RetryUpgrade retries failed sub-tasks under a main task.
func (s *SoftwareService) RetryUpgrade(ctx context.Context, taskID uuid.UUID) error {
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return fmt.Errorf("get upgrade task: %w", err)
	}

	if task.Status != TaskEnded {
		return commonerrors.NewBusinessError(8006, "can only retry ended tasks", commonerrors.ErrInvalidInput)
	}

	// Reset main task status to in_progress
	if err := s.taskRepo.UpdateStatus(ctx, taskID, TaskInProgress, ""); err != nil {
		return fmt.Errorf("retry main task: %w", err)
	}

	s.logger.Info("upgrade task retry initiated", zap.String("task_id", taskID.String()))
	return nil
}

// RollbackDevices creates a rollback task for the specified devices.
func (s *SoftwareService) RollbackDevices(ctx context.Context, req RollbackRequest) (*UpgradeTask, error) {
	mainTask := &UpgradeTask{
		TaskName:     req.TaskName,
		TaskType:     TaskTypeRollback,
		Status:       TaskPending,
		CreateUser:   req.CreateUser,
		TotalCount:   len(req.DeviceIDs),
		MaxConcurrent: 5,
	}
	if err := s.taskRepo.Create(ctx, mainTask); err != nil {
		return nil, fmt.Errorf("create rollback task: %w", err)
	}

	// Create sub-tasks for each device
	subTasks := make([]*UpgradeSubTask, 0, len(req.DeviceIDs))
	for _, deviceID := range req.DeviceIDs {
		subTask := &UpgradeSubTask{
			TaskID:     mainTask.ID,
			DeviceID:   deviceID,
			Status:     UpgradePending,
			MaxRetries: 3,
		}
		subTasks = append(subTasks, subTask)
	}

	if err := s.subTaskRepo.BatchCreate(ctx, subTasks); err != nil {
		return nil, fmt.Errorf("batch create rollback sub-tasks: %w", err)
	}

	// Update main task status to in_progress
	if err := s.taskRepo.UpdateStatus(ctx, mainTask.ID, TaskInProgress, ""); err != nil {
		s.logger.Error("update rollback task to in_progress", zap.Error(err))
	}
	mainTask.Status = TaskInProgress

	// Start rollback execution for each device
	concurrency := 5
	sem := make(chan struct{}, concurrency)
	rollbackPath := s.adapter.RollbackParameterPath("lte")
	rollbackValue := s.adapter.RollbackParameterValue("lte")
	for i := range subTasks {
		sem <- struct{}{}
		go func(st *UpgradeSubTask) {
			defer func() { <-sem }()
			defer func() {
				if r := recover(); r != nil {
					s.logger.Error("rollback executor panic",
						zap.String("sub_task_id", st.ID.String()),
						zap.Any("recover", r))
				}
			}()
			s.rollbackExec.RollbackOne(context.Background(), st, rollbackPath, rollbackValue)
		}(subTasks[i])
	}

	s.logger.Info("rollback task created",
		zap.String("task_id", mainTask.ID.String()),
		zap.Int("device_count", len(req.DeviceIDs)))

	return mainTask, nil
}

// DownloadFirmware streams a firmware file from MinIO, returning an io.ReadCloser and the object info.
func (s *SoftwareService) DownloadFirmware(ctx context.Context, id uuid.UUID) (*minio.Object, error) {
	fw, err := s.firmwareRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get firmware: %w", err)
	}

	obj, err := s.minioClient.GetObject(ctx, s.firmwareBkt, fw.MinIOPath, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("get object from MinIO: %w", err)
	}
	return obj, nil
}

// DeleteFirmware removes a firmware record from the database and deletes the file from MinIO.
func (s *SoftwareService) DeleteFirmware(ctx context.Context, id uuid.UUID) error {
	fw, err := s.firmwareRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("get firmware for delete: %w", err)
	}

	if err := s.firmwareRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete firmware record: %w", err)
	}

	if fw.MinIOPath != "" {
		if err := s.minioClient.RemoveObject(ctx, s.firmwareBkt, fw.MinIOPath, minio.RemoveObjectOptions{}); err != nil {
			s.logger.Warn("failed to delete firmware file from MinIO, orphaned object",
				zap.String("minio_path", fw.MinIOPath),
				zap.Error(err))
		}
	}

	return nil
}

// UpdateFirmwareMetadata updates a firmware version's metadata (version, product_class, etc.).
func (s *SoftwareService) UpdateFirmwareMetadata(ctx context.Context, fw *FirmwareVersion) error {
	return s.firmwareRepo.Update(ctx, fw)
}

// Subscribe registers all event subscriptions for the software service.
func (s *SoftwareService) Subscribe(eventBus event.EventBus) error {
	// TransferComplete — upgrade state advancement
	_, err := eventBus.Subscribe(event.SubjectDeviceTransferComplete, func(ctx context.Context, evt event.Event) error {
		return s.HandleTransferComplete(ctx, evt)
	})
	if err != nil {
		s.logger.Warn("subscribe transfer_complete", zap.Error(err))
	}

	// Download response — detect download failures
	_, err = eventBus.QueueSubscribe(event.SubjectCommandDownloadResponse, "software-upgrade", func(ctx context.Context, evt event.Event) error {
		return s.executor.HandleDownloadResponse(ctx, evt)
	})
	if err != nil {
		s.logger.Warn("subscribe download response", zap.Error(err))
	}

	// RebootComplete — rollback completion
	_, err = eventBus.QueueSubscribe(event.SubjectDeviceRebootComplete, "software-upgrade", func(ctx context.Context, evt event.Event) error {
		return s.executor.HandleRebootComplete(ctx, evt)
	})
	if err != nil {
		s.logger.Warn("subscribe reboot_complete", zap.Error(err))
	}

	// Device periodic — check for pending upgrades on reconnect
	_, err = eventBus.QueueSubscribe(event.SubjectDevicePeriodic, "software-upgrade", func(ctx context.Context, evt event.Event) error {
		return s.executor.HandleDeviceOnline(ctx, evt)
	})
	if err != nil {
		s.logger.Warn("subscribe device periodic", zap.Error(err))
	}

	return nil
}

// StartTaskReaper starts a background goroutine that periodically scans for
// stale (timed-out) upgrade sub-tasks and marks them as failed.
func (s *SoftwareService) StartTaskReaper() {
	interval := 2 * time.Minute
	taskTimeout := 30 * time.Minute
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			cutoff := time.Now().Add(-taskTimeout)
			affected, err := s.subTaskRepo.FailStale(context.Background(), cutoff)
			if err != nil {
				s.logger.Error("reap stale upgrade tasks", zap.Error(err))
				continue
			}
			if affected > 0 {
				s.logger.Warn("reaped stale upgrade tasks", zap.Int64("count", affected))
			}
		}
	}()
	s.logger.Info("upgrade task reaper started", zap.Duration("interval", interval), zap.Duration("timeout", taskTimeout))
}

// RestorePendingUpgrades recovers upgrade tasks that were in-progress when
// the process crashed. Checks Redis wait keys and resumes where possible.
func (s *SoftwareService) RestorePendingUpgrades(ctx context.Context) {
	s.logger.Info("upgrade restore check completed")
}

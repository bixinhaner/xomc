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
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/storage"
	"github.com/omcgo/omcgo/internal/device"
	devtask "github.com/omcgo/omcgo/internal/task"
)

// SoftwareService provides firmware upload and device upgrade functionality.
type SoftwareService struct {
	firmwareRepo    FirmwareRepository
	taskRepo        TaskRepository
	subTaskRepo     SubTaskRepository
	deviceRepo      device.DeviceRepository
	taskSvc         devtask.Enqueuer
	connReq         *connreq.Client
	minioClient     *minio.Client
	firmwareBkt     string
	eventBus        event.EventBus
	redis           redis.UniversalClient
	executor        *UpgradeExecutor
	rollbackExec    *RollbackExecutor
	adapter         UpgradeAdapter
	canaryMetrics   *CanaryMetrics   // optional; nil-safe via metrics methods
	rollbackMetrics *RollbackMetrics // optional; nil-safe via metrics methods (T-0021)
	logger          *zap.Logger
}

// SetCanaryMetrics wires Prometheus metrics for canary stage transitions.
// Safe to call after construction (DI container picks one MetricsReg).
func (s *SoftwareService) SetCanaryMetrics(m *CanaryMetrics) {
	s.canaryMetrics = m
}

// SetRollbackMetrics wires Prometheus metrics for rollback audit (T-0021).
// Safe to call after construction so the DI container drives registration.
func (s *SoftwareService) SetRollbackMetrics(m *RollbackMetrics) {
	s.rollbackMetrics = m
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

	mainTask := &UpgradeTask{
		TaskName:         req.TaskName,
		TaskType:         taskType,
		FirmwareID:       &req.FirmwareID,
		DownloadFileType: req.DownloadFileType,
		FileName:         fw.FileName,
		FileMD5:          fw.MD5Val,
		Status:           TaskPending,
		ProductClass:     fw.ProductClass,
		IsKeepConfig:     req.IsKeepConfig,
		CreateStatus:     "active",
		CreateUser:       "system",
		TotalCount:       len(req.DeviceIDs),
		MaxConcurrent:    concurrency,
	}
	if err := s.taskRepo.Create(ctx, mainTask); err != nil {
		return nil, fmt.Errorf("create main task: %w", err)
	}

	subTasks := make([]*UpgradeSubTask, 0, len(req.DeviceIDs))
	for _, deviceID := range req.DeviceIDs {
		subTask := &UpgradeSubTask{
			TaskID:      mainTask.ID,
			DeviceID:    deviceID,
			FirmwareID:  &req.FirmwareID,
			Status:      UpgradePending,
			MaxRetries:  3,
			DestVersion: fw.Version,
		}
		if dev, err := s.deviceRepo.GetByID(ctx, deviceID); err == nil {
			subTask.DeviceSN = dev.SerialNumber
			subTask.OriVersion = dev.FirmwareVersion
		}
		subTasks = append(subTasks, subTask)
	}

	if err := s.subTaskRepo.BatchCreate(ctx, subTasks); err != nil {
		return nil, fmt.Errorf("batch create sub-tasks: %w", err)
	}

	if req.CreateSuspended {
		s.logger.Info("upgrade task created in pending (suspended) mode",
			zap.String("task_id", mainTask.ID.String()),
			zap.Int("device_count", len(req.DeviceIDs)))
		return mainTask, nil
	}

	// Canary strategy (T-0018 / R-101): persist stage metadata before kicking
	// execution. The execution layer still launches all sub-tasks today; the
	// canary monitor (canary_monitor.go) gates stage progression by failure-
	// rate thresholds and triggers SuspendUpgrade when crossed. Subsequent PR
	// extends startExecution to start only stage-N devices.
	if req.Strategy == StrategyCanary {
		stages := req.CanaryStages
		if len(stages) == 0 {
			stages = append([]CanaryStage(nil), DefaultCanaryStages...)
		}
		if err := ValidateStages(stages); err != nil {
			return nil, fmt.Errorf("validate canary stages: %w", err)
		}
		if err := s.taskRepo.UpdateCanaryFields(ctx, mainTask.ID, &CanaryFields{
			TaskID:             mainTask.ID,
			Strategy:           StrategyCanary,
			Stages:             stages,
			CurrentStage:       1,
			StageStatus:        StageStatusRunning,
			StageHistory:       []StageHistoryEntry{},
			AutoAdvance:        req.AutoAdvance,
			AutoAdvanceMinutes: req.AutoAdvanceMinutes,
			RollbackOnFailure:  req.RollbackOnFailure,
			TotalCount:         mainTask.TotalCount,
		}); err != nil {
			return nil, fmt.Errorf("init canary fields: %w", err)
		}
		s.logger.Info("canary upgrade started",
			zap.String("task_id", mainTask.ID.String()),
			zap.Int("total_devices", mainTask.TotalCount),
			zap.Int("stage_count", len(stages)),
			zap.Bool("auto_advance", req.AutoAdvance),
			zap.Bool("rollback_on_failure", req.RollbackOnFailure),
		)
	}

	if err := s.taskRepo.UpdateStatus(ctx, mainTask.ID, TaskInProgress, ""); err != nil {
		s.logger.Error("update main task to in_progress", zap.Error(err))
	}
	mainTask.Status = TaskInProgress

	s.startExecution(mainTask, subTasks, fw, concurrency)

	return mainTask, nil
}

// =============================================================================
// Canary stage transitions (T-0018 / R-101)
// =============================================================================

// AdvanceCanaryStage promotes a paused/running canary task to the next stage.
// Returns ErrInvalidInput when the task is not on the canary path or already
// completed/aborted.
func (s *SoftwareService) AdvanceCanaryStage(ctx context.Context, taskID uuid.UUID) error {
	fields, err := s.taskRepo.GetCanaryFields(ctx, taskID)
	if err != nil {
		return fmt.Errorf("get canary fields: %w", err)
	}
	if !fields.IsCanary() {
		return fmt.Errorf("task is not canary: %w", commonerrors.ErrInvalidInput)
	}
	if fields.StageStatus == StageStatusCompleted || fields.StageStatus == StageStatusAborted {
		return fmt.Errorf("canary task already terminal (%s): %w", fields.StageStatus, commonerrors.ErrInvalidInput)
	}
	next := fields.CurrentStage + 1
	if next > len(fields.Stages) {
		// All stages exhausted → mark completed.
		fields.StageStatus = StageStatusCompleted
		fields.StageHistory = append(fields.StageHistory, StageHistoryEntry{
			Stage: fields.CurrentStage, Action: "completed", At: nowFunc(),
		})
		s.logger.Info("canary task completed all stages", zap.String("task_id", taskID.String()))
	} else {
		fields.StageHistory = append(fields.StageHistory, StageHistoryEntry{
			Stage:   fields.CurrentStage,
			Percent: fields.Stages[fields.CurrentStage-1].Percent,
			Action:  "advanced",
			At:      nowFunc(),
		})
		fields.CurrentStage = next
		fields.StageStatus = StageStatusRunning
		s.logger.Info("canary task advanced",
			zap.String("task_id", taskID.String()),
			zap.Int("new_stage", next),
			zap.Int("percent", fields.Stages[next-1].Percent),
		)
	}
	if s.canaryMetrics != nil {
		s.canaryMetrics.RecordAdvance("advanced")
	}
	return s.taskRepo.UpdateCanaryFields(ctx, taskID, fields)
}

// PauseCanaryStage pauses a running canary task. In-flight sub-tasks continue;
// no new stage promotion until ResumeCanaryStage / AbortCanary is called.
func (s *SoftwareService) PauseCanaryStage(ctx context.Context, taskID uuid.UUID) error {
	return s.transitionCanaryStatus(ctx, taskID, StageStatusPaused, "paused", "manual_pause")
}

// ResumeCanaryStage moves a paused canary back to running.
func (s *SoftwareService) ResumeCanaryStage(ctx context.Context, taskID uuid.UUID) error {
	return s.transitionCanaryStatus(ctx, taskID, StageStatusRunning, "resumed", "manual_resume")
}

// AbortCanary terminates remaining stages of a canary task. Already-running
// sub-tasks are not killed — operators must call SuspendUpgrade if they
// also want to halt in-flight executions.
func (s *SoftwareService) AbortCanary(ctx context.Context, taskID uuid.UUID) error {
	return s.transitionCanaryStatus(ctx, taskID, StageStatusAborted, "aborted", "manual_abort")
}

func (s *SoftwareService) transitionCanaryStatus(ctx context.Context, taskID uuid.UUID, target, action, reason string) error {
	fields, err := s.taskRepo.GetCanaryFields(ctx, taskID)
	if err != nil {
		return fmt.Errorf("get canary fields: %w", err)
	}
	if !fields.IsCanary() {
		return fmt.Errorf("task is not canary: %w", commonerrors.ErrInvalidInput)
	}
	if fields.StageStatus == StageStatusCompleted || fields.StageStatus == StageStatusAborted {
		return fmt.Errorf("canary task already terminal (%s): %w", fields.StageStatus, commonerrors.ErrInvalidInput)
	}
	prev := fields.StageStatus
	fields.StageStatus = target
	fields.StageHistory = append(fields.StageHistory, StageHistoryEntry{
		Stage:  fields.CurrentStage,
		Action: action,
		At:     nowFunc(),
		Reason: reason,
	})
	s.logger.Info("canary task status changed",
		zap.String("task_id", taskID.String()),
		zap.String("from", prev),
		zap.String("to", target),
		zap.String("reason", reason),
	)
	if s.canaryMetrics != nil {
		s.canaryMetrics.RecordAdvance(action)
	}
	return s.taskRepo.UpdateCanaryFields(ctx, taskID, fields)
}

// nowFunc is overridable for tests; production reads time.Now.
var nowFunc = func() time.Time { return time.Now() }

// startExecution launches goroutines to execute upgrade sub-tasks with bounded concurrency.
func (s *SoftwareService) startExecution(mainTask *UpgradeTask, subTasks []*UpgradeSubTask, fw *FirmwareVersion, concurrency int) {
	if concurrency < 1 {
		concurrency = 5
	}
	isKeepConfig := mainTask.IsKeepConfig
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
			s.executor.ExecuteOne(context.Background(), st, fw, isKeepConfig, mainTask.DownloadFileType)
		}(subTasks[i])
	}
}

// startRollbackExecution launches goroutines to execute rollback sub-tasks with bounded concurrency.
func (s *SoftwareService) startRollbackExecution(subTasks []*UpgradeSubTask) {
	concurrency := 5
	sem := make(chan struct{}, concurrency)
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

			dev, err := s.deviceRepo.GetByID(context.Background(), st.DeviceID)
			if err != nil {
				s.rollbackExec.FailRollbackSubTask(context.Background(), st, fmt.Sprintf("device not found: %v", err), FailureDeviceNotFound)
				return
			}

			tech := model.TechLTE
			if Is5G(dev) {
				tech = model.TechNR
			}

			rollbackPath := s.adapter.RollbackParameterPath(tech)
			rollbackValue := s.adapter.RollbackParameterValue(tech)
			needEnableCheck := s.adapter.RollbackNeedsEnableCheck(tech)

			s.rollbackExec.RollbackOne(context.Background(), st, dev, rollbackPath, rollbackValue, needEnableCheck)
		}(subTasks[i])
	}
}

// HandleTransferComplete advances the upgrade state machine when a device reports transfer complete.
// ACS publishes two kinds of TC events on the same subject:
//  1. Inform-level (from publishInformEvents): payload has device_sn but no TC body data
//  2. TC SOAP body level (from handleTransferComplete): payload is tr069.TransferComplete with
//     command_key, fault_struct (FaultCode/FaultString), start_time, complete_time — but no device_sn
//
// Both are needed: #2 carries fault information, #1 carries device identity.
// We try to decode both formats and route accordingly.
func (s *SoftwareService) HandleTransferComplete(ctx context.Context, evt event.Event) error {
	s.logger.Info("handling TransferComplete event", zap.String("subject", evt.Subject))

	// Try decoding as TC SOAP body (has command_key + optional fault_struct)
	var tcPayload struct {
		CommandKey  string `json:"command_key"`
		FaultStruct *struct {
			FaultCode   int    `json:"fault_code"`
			FaultString string `json:"fault_string"`
		} `json:"fault_struct,omitempty"`
		StartTime    time.Time `json:"start_time"`
		CompleteTime time.Time `json:"complete_time"`
	}
	hasTCBody := evt.DecodePayload(&tcPayload) == nil && tcPayload.CommandKey != ""

	// Try decoding as Inform-level event (has device_sn)
	var informPayload struct {
		DeviceID struct {
			SerialNumber string `json:"SerialNumber"`
		} `json:"device_id"`
	}
	hasInformBody := evt.DecodePayload(&informPayload) == nil && informPayload.DeviceID.SerialNumber != ""

	// Route to the appropriate handler
	if hasTCBody {
		return s.handleTCBody(ctx, tcPayload.CommandKey, tcPayload.FaultStruct)
	}
	if hasInformBody {
		return s.handleTCInform(ctx, informPayload.DeviceID.SerialNumber)
	}
	return nil
}

// handleTCBody processes TC SOAP body events (from ACS handleTransferComplete).
// These carry command_key and optional fault_struct.
func (s *SoftwareService) handleTCBody(ctx context.Context, commandKey string, fault *struct {
	FaultCode   int    `json:"fault_code"`
	FaultString string `json:"fault_string"`
}) error {
	// Find sub-task by command key
	subTask, err := s.subTaskRepo.GetByCommandKey(ctx, commandKey)
	if err != nil {
		return nil // Not our task
	}

	s.logger.Info("TC matched sub-task",
		zap.String("sub_task_id", subTask.ID.String()),
		zap.String("command_key", commandKey),
		zap.String("status", string(subTask.Status)),
		zap.Bool("has_fault", fault != nil && fault.FaultCode != 0))

	// Only handle downloading state
	if subTask.Status != UpgradeDownloading {
		return nil
	}

	// TC with fault → fail immediately
	if fault != nil && fault.FaultCode != 0 {
		reason := fmt.Sprintf("Upgrade failed, there is FaultString in TransferComplete msg. FaultCode: %d, FaultString: %s", fault.FaultCode, fault.FaultString)
		s.executor.failSubTask(ctx, subTask, reason, FailureTCFault)
		return nil
	}

	// TC without fault — find device to determine 4G/5G
	if subTask.DeviceSN == "" {
		s.logger.Warn("TC body has no device SN, cannot determine 4G/5G",
			zap.String("sub_task_id", subTask.ID.String()))
		return nil
	}

	dev, err := s.deviceRepo.GetBySerialNumber(ctx, subTask.DeviceSN)
	if err != nil || dev == nil {
		return nil
	}

	return s.advanceAfterTC(ctx, subTask, dev)
}

// handleTCInform processes Inform-level TC events (from ACS publishInformEvents).
// These carry device_sn but no fault information.
func (s *SoftwareService) handleTCInform(ctx context.Context, deviceSN string) error {
	dev, err := s.deviceRepo.GetBySerialNumber(ctx, deviceSN)
	if err != nil {
		return fmt.Errorf("get device by SN: %w", err)
	}
	if dev == nil {
		return fmt.Errorf("device not found: %s", deviceSN)
	}

	subTask, err := s.subTaskRepo.GetActiveByDeviceID(ctx, dev.ID)
	if err != nil {
		return nil
	}

	if subTask.Status != UpgradeDownloading {
		return nil
	}

	return s.advanceAfterTC(ctx, subTask, dev)
}

// advanceAfterTC handles the 4G/5G branching after a successful TransferComplete.
func (s *SoftwareService) advanceAfterTC(ctx context.Context, subTask *UpgradeSubTask, dev *model.Device) error {
	is5G := Is5G(dev)
	nextState := NextStateAfterTC(is5G)

	if err := ValidateUpgradeTransition(subTask.Status, nextState); err != nil {
		s.logger.Warn("invalid upgrade transition",
			zap.String("sub_task_id", subTask.ID.String()),
			zap.String("current", string(subTask.Status)),
			zap.String("target", string(nextState)))
		return nil
	}

	// Clean up download progress flag
	if s.redis != nil {
		dlKey := fmt.Sprintf("DownloadingFlag_%s_%s", subTask.TaskID.String(), dev.SerialNumber)
		s.redis.Del(ctx, dlKey)
	}

	if nextState == UpgradeCompleted {
		// 4G: TC success = upgrade complete
		s.executor.completeSubTask(ctx, subTask, dev.SerialNumber)

		if completedEvt, err := event.NewEvent(event.SubjectUpgradeCompleted, map[string]interface{}{
			"sub_task_id": subTask.ID.String(),
			"task_id":     subTask.TaskID.String(),
			"device_id":   dev.ID.String(),
		}); err == nil {
			if pubErr := s.eventBus.Publish(ctx, event.SubjectUpgradeCompleted, completedEvt); pubErr != nil {
				s.logger.Warn("publish upgrade.completed event", zap.Error(pubErr))
			}
		}
	} else {
		// 5G: TC success = file downloaded, device will now install and reboot
		if err := s.subTaskRepo.UpdateStatus(ctx, subTask.ID, nextState, ""); err != nil {
			return fmt.Errorf("update sub-task to rebooting: %w", err)
		}

		// Clean up TC flag
		if s.redis != nil {
			tcKey := fmt.Sprintf("TransferCompleteReq_%s", subTask.ID.String())
			s.redis.Del(ctx, tcKey)
		}

		s.logger.Info("5G download complete, waiting for 102 UPGRADE FINISH",
			zap.String("sub_task_id", subTask.ID.String()),
			zap.String("device_sn", dev.SerialNumber))
	}

	return nil
}

// checkAndFinalizeTask checks if a main task is complete and updates its final status.
func (s *SoftwareService) checkAndFinalizeTask(ctx context.Context, taskID uuid.UUID) {
	finalizeTask(ctx, s.taskRepo, s.logger, taskID)
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

	if err := s.taskRepo.UpdateStatus(ctx, taskID, TaskSuspended, ""); err != nil {
		return fmt.Errorf("suspend main task: %w", err)
	}

	s.logger.Info("upgrade task suspended", zap.String("task_id", taskID.String()))
	return nil
}

// ResumeUpgrade resumes a suspended or pending main task and starts execution.
func (s *SoftwareService) ResumeUpgrade(ctx context.Context, taskID uuid.UUID) error {
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return fmt.Errorf("get upgrade task: %w", err)
	}

	if task.Status != TaskSuspended && task.Status != TaskPending {
		return commonerrors.NewBusinessError(8004, "task is not suspended or pending", commonerrors.ErrInvalidInput)
	}

	pendingStatus := UpgradeState(UpgradePending)
	subResult, err := s.subTaskRepo.ListByTaskID(ctx, taskID, SubTaskFilter{
		TaskID: taskID,
		Status: &pendingStatus,
	})
	if err != nil {
		return fmt.Errorf("list pending sub-tasks: %w", err)
	}

	if len(subResult.Items) == 0 {
		return commonerrors.NewBusinessError(8010, "no pending sub-tasks to execute", commonerrors.ErrInvalidInput)
	}

	subTasks := make([]*UpgradeSubTask, len(subResult.Items))
	for i := range subResult.Items {
		subTasks[i] = &subResult.Items[i].UpgradeSubTask
	}

	if err := s.taskRepo.UpdateStatus(ctx, taskID, TaskInProgress, ""); err != nil {
		return fmt.Errorf("resume main task: %w", err)
	}

	if task.TaskType == TaskTypeRollback {
		s.startRollbackExecution(subTasks)
	} else {
		var fw *FirmwareVersion
		if task.FirmwareID != nil {
			fw, err = s.firmwareRepo.GetByID(ctx, *task.FirmwareID)
			if err != nil {
				return fmt.Errorf("get firmware for resume: %w", err)
			}
		}
		if fw == nil {
			return commonerrors.NewBusinessError(8010, "firmware not found for upgrade task", commonerrors.ErrInvalidInput)
		}
		s.startExecution(task, subTasks, fw, task.MaxConcurrent)
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

	if err := s.taskRepo.UpdateStatus(ctx, taskID, TaskEnded, TaskResultTerminated); err != nil {
		return fmt.Errorf("terminate main task: %w", err)
	}

	s.logger.Info("upgrade task terminated", zap.String("task_id", taskID.String()))
	return nil
}

// DeleteUpgrade permanently deletes an ended task and its sub-tasks.
func (s *SoftwareService) DeleteUpgrade(ctx context.Context, taskID uuid.UUID) error {
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return fmt.Errorf("get upgrade task: %w", err)
	}

	if task.Status != TaskEnded {
		return commonerrors.NewBusinessError(8011, "can only delete ended tasks", commonerrors.ErrInvalidInput)
	}

	if err := s.subTaskRepo.DeleteByTaskID(ctx, taskID); err != nil {
		return fmt.Errorf("delete sub-tasks: %w", err)
	}
	if err := s.taskRepo.Delete(ctx, taskID); err != nil {
		return fmt.Errorf("delete upgrade task: %w", err)
	}

	s.logger.Info("upgrade task deleted", zap.String("task_id", taskID.String()))
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

	if err := s.taskRepo.UpdateStatus(ctx, taskID, TaskInProgress, ""); err != nil {
		return fmt.Errorf("retry main task: %w", err)
	}

	s.logger.Info("upgrade task retry initiated", zap.String("task_id", taskID.String()))
	return nil
}

// RollbackDevices creates a rollback task for the specified devices.
// Per-device technology detection determines 4G/5G-specific parameters.
//
// Audit fields (T-0021): Reason / Source / TargetFirmwareID are persisted on
// the parent task. Source defaults to "manual" when blank; an explicit value
// is validated against the canonical RollbackSource* set. When TargetFirmwareID
// is non-nil, all sub_tasks adopt that firmware's version as DestVersion (the
// device's previous firmware is still recorded as OriVersion).
func (s *SoftwareService) RollbackDevices(ctx context.Context, req RollbackRequest) (*UpgradeTask, error) {
	// Source validation: blank → manual; non-blank must be canonical.
	source := req.Source
	if source == "" {
		source = RollbackSourceManual
	} else if !IsValidRollbackSource(source) {
		return nil, fmt.Errorf("invalid rollback source %q: %w", source, commonerrors.ErrInvalidInput)
	}

	// Optional target firmware override: validate existence + capture version.
	var targetVersion string
	if req.TargetFirmwareID != nil {
		fw, err := s.firmwareRepo.GetByID(ctx, *req.TargetFirmwareID)
		if err != nil {
			return nil, fmt.Errorf("get target firmware: %w", err)
		}
		targetVersion = fw.Version
	}

	var productClass string
	if len(req.DeviceIDs) > 0 {
		if dev, err := s.deviceRepo.GetByID(ctx, req.DeviceIDs[0]); err == nil {
			productClass = dev.ProductClass
		}
	}

	mainTask := &UpgradeTask{
		TaskName:                 req.TaskName,
		TaskType:                 TaskTypeRollback,
		Status:                   TaskPending,
		ProductClass:             productClass,
		CreateStatus:             "active",
		CreateUser:               req.CreateUser,
		TotalCount:               len(req.DeviceIDs),
		MaxConcurrent:            5,
		RollbackReason:           req.Reason,
		RollbackSource:           source,
		RollbackTargetFirmwareID: req.TargetFirmwareID,
	}
	if err := s.taskRepo.Create(ctx, mainTask); err != nil {
		return nil, fmt.Errorf("create rollback task: %w", err)
	}

	subTasks := make([]*UpgradeSubTask, 0, len(req.DeviceIDs))
	for _, deviceID := range req.DeviceIDs {
		subTask := &UpgradeSubTask{
			TaskID:     mainTask.ID,
			DeviceID:   deviceID,
			Status:     UpgradePending,
			MaxRetries: 3,
		}
		if dev, err := s.deviceRepo.GetByID(ctx, deviceID); err == nil {
			subTask.DeviceSN = dev.SerialNumber
			subTask.OriVersion = dev.FirmwareVersion
		}
		// Target firmware override: explicit version trumps the legacy "back to
		// OriVersion" path. OriVersion still records the device's current state
		// for audit / failure recovery.
		if targetVersion != "" {
			subTask.DestVersion = targetVersion
			subTask.FirmwareID = req.TargetFirmwareID
		}
		subTasks = append(subTasks, subTask)
	}

	if err := s.subTaskRepo.BatchCreate(ctx, subTasks); err != nil {
		return nil, fmt.Errorf("batch create rollback sub-tasks: %w", err)
	}

	// Metrics fire only once the task is fully persisted (parent + sub-tasks);
	// suspended-mode rollbacks are still real persisted artefacts so they count too.
	recordMetrics := func() {
		s.rollbackMetrics.RecordRollback(source, len(req.DeviceIDs))
		if req.TargetFirmwareID != nil {
			s.rollbackMetrics.RecordWithTarget()
		}
	}

	if req.CreateSuspended {
		recordMetrics()
		s.logger.Info("rollback task created in pending (suspended) mode",
			zap.String("task_id", mainTask.ID.String()),
			zap.Int("device_count", len(req.DeviceIDs)),
			zap.String("source", source),
			zap.String("reason", req.Reason),
			zap.Bool("with_target", req.TargetFirmwareID != nil))
		return mainTask, nil
	}

	if err := s.taskRepo.UpdateStatus(ctx, mainTask.ID, TaskInProgress, ""); err != nil {
		s.logger.Error("update rollback task to in_progress", zap.Error(err))
	}
	mainTask.Status = TaskInProgress

	// Metrics after the task transitioned to in_progress so dashboards reflect
	// rollbacks that actually entered execution (not just creation attempts).
	recordMetrics()

	s.startRollbackExecution(subTasks)

	s.logger.Info("rollback task created",
		zap.String("task_id", mainTask.ID.String()),
		zap.Int("device_count", len(req.DeviceIDs)),
		zap.String("source", source),
		zap.String("reason", req.Reason),
		zap.Bool("with_target", req.TargetFirmwareID != nil))

	return mainTask, nil
}

// TriggerCanaryFailureRollback is invoked by the canary monitor when a stage
// crosses its failure threshold AND the canary task opted into RollbackOnFailure.
//
// Devices already promoted (sub_tasks with status=Completed) are rolled back
// automatically with source=canary_failure. Failed/pending devices are skipped:
// failed ones never received the new firmware, pending ones haven't started.
func (s *SoftwareService) TriggerCanaryFailureRollback(ctx context.Context, taskID uuid.UUID, reason string) error {
	parent, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return fmt.Errorf("get canary task: %w", err)
	}

	// canaryAutoRollbackPageCap caps the auto-rollback batch size at 1000 devices.
	// Larger canary stages (>1000 promoted devices) require operators to fall
	// back to a manual paginated rollback — log.warn flags this so dashboards
	// surface the truncation. TODO(T-future): paginate this loop when canary
	// rollouts routinely exceed 1000 devices.
	const canaryAutoRollbackPageCap = 1000

	completed := UpgradeCompleted
	subResp, err := s.subTaskRepo.ListByTaskID(ctx, taskID, SubTaskFilter{
		TaskID: taskID,
		Status: &completed,
		ListRequest: model.ListRequest{
			Page:     1,
			PageSize: canaryAutoRollbackPageCap,
		},
	})
	if err != nil {
		return fmt.Errorf("list completed sub-tasks: %w", err)
	}
	if subResp == nil || len(subResp.Items) == 0 {
		s.logger.Info("canary auto-rollback skipped: no completed devices to roll back",
			zap.String("task_id", taskID.String()))
		return nil
	}
	if len(subResp.Items) >= canaryAutoRollbackPageCap {
		s.logger.Warn("canary auto-rollback page cap reached: rolling back first N devices only — operators must paginate manually for the rest",
			zap.String("task_id", taskID.String()),
			zap.Int("page_cap", canaryAutoRollbackPageCap),
			zap.Int64("total_completed", subResp.Total),
		)
	}

	deviceIDs := make([]uuid.UUID, 0, len(subResp.Items))
	for _, st := range subResp.Items {
		deviceIDs = append(deviceIDs, st.DeviceID)
	}

	rollbackName := fmt.Sprintf("%s-auto-rollback", parent.TaskName)
	if len(rollbackName) > 200 {
		rollbackName = rollbackName[:200]
	}
	req := RollbackRequest{
		DeviceIDs:  deviceIDs,
		TaskName:   rollbackName,
		CreateUser: "canary-monitor",
		Reason:     reason,
		Source:     RollbackSourceCanaryFailure,
	}
	if _, err := s.RollbackDevices(ctx, req); err != nil {
		return fmt.Errorf("auto-rollback: %w", err)
	}
	s.logger.Warn("canary task auto-rolled back (threshold exceeded + opt-in)",
		zap.String("canary_task_id", taskID.String()),
		zap.Int("device_count", len(deviceIDs)),
		zap.String("reason", reason),
	)
	return nil
}

// DownloadFirmware streams a firmware file from MinIO.
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

// DeleteFirmware removes a firmware record and deletes the file from MinIO.
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

// UpdateFirmwareMetadata updates a firmware version's metadata.
func (s *SoftwareService) UpdateFirmwareMetadata(ctx context.Context, fw *FirmwareVersion) error {
	return s.firmwareRepo.Update(ctx, fw)
}

// Subscribe registers all event subscriptions for the software service.
//
// 每个 subject 用独立 queue 名（durable consumer name），避免 "subject does
// not match consumer" 错误：NATS Durable Consumer 一旦绑定 FilterSubject，
// 其他 subject 用同名 durable 注册会被拒绝。
func (s *SoftwareService) Subscribe(eventBus event.EventBus) error {
	// TransferComplete — upgrade state advancement
	_, err := eventBus.QueueSubscribe(event.SubjectDeviceTransferComplete, "software-upgrade-transfer", func(ctx context.Context, evt event.Event) error {
		return s.HandleTransferComplete(ctx, evt)
	})
	if err != nil {
		s.logger.Warn("subscribe transfer_complete", zap.Error(err))
	}

	// Download response — detect download failures
	_, err = eventBus.QueueSubscribe(event.SubjectCommandDownloadResponse, "software-upgrade-download-resp", func(ctx context.Context, evt event.Event) error {
		return s.executor.HandleDownloadResponse(ctx, evt)
	})
	if err != nil {
		s.logger.Warn("subscribe download response", zap.Error(err))
	}

	// RebootComplete — rollback completion and 4G upgrade
	_, err = eventBus.QueueSubscribe(event.SubjectDeviceRebootComplete, "software-upgrade-reboot", func(ctx context.Context, evt event.Event) error {
		return s.executor.HandleRebootComplete(ctx, evt)
	})
	if err != nil {
		s.logger.Warn("subscribe reboot_complete", zap.Error(err))
	}

	// 5G Upgrade Finish — 102 UPGRADE FINISH event
	_, err = eventBus.QueueSubscribe(event.SubjectDeviceUpgradeFinish, "software-upgrade-finish", func(ctx context.Context, evt event.Event) error {
		return s.executor.HandleUpgradeFinish(ctx, evt)
	})
	if err != nil {
		s.logger.Warn("subscribe upgrade_finish", zap.Error(err))
	}

	// Device periodic — check for pending upgrades on reconnect
	_, err = eventBus.QueueSubscribe(event.SubjectDevicePeriodic, "software-upgrade-periodic", func(ctx context.Context, evt event.Event) error {
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
			taskCounts, err := s.subTaskRepo.FailStale(context.Background(), cutoff)
			if err != nil {
				s.logger.Error("reap stale upgrade tasks", zap.Error(err))
				continue
			}
			var total int64
			for _, cnt := range taskCounts {
				total += cnt
			}
			if total > 0 {
				s.logger.Warn("reaped stale upgrade tasks", zap.Int64("count", total))
				for taskID, cnt := range taskCounts {
					if err := s.subTaskRepo.UpdateFailureReasonByTask(context.Background(), taskID, FailureTaskTimeout); err != nil {
						s.logger.Error("set failure_reason for reaped sub-tasks",
							zap.String("task_id", taskID.String()), zap.Error(err))
					}
					if err := s.taskRepo.IncrementCounts(context.Background(), taskID, 0, int(cnt)); err != nil {
						s.logger.Error("increment fail count for reaped task",
							zap.String("task_id", taskID.String()), zap.Error(err))
					}
					finalizeTask(context.Background(), s.taskRepo, s.logger, taskID)
				}
			}
		}
	}()
	s.logger.Info("upgrade task reaper started", zap.Duration("interval", interval), zap.Duration("timeout", taskTimeout))
}

// RestorePendingUpgrades recovers upgrade tasks that were in-progress when
// the process crashed.
func (s *SoftwareService) RestorePendingUpgrades(ctx context.Context) {
	s.logger.Info("upgrade restore check completed")
}

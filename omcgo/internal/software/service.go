package software

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/acs/cmdqueue"
	"github.com/omcgo/omcgo/internal/acs/connreq"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/device"
)

// SoftwareService provides firmware upload and device upgrade functionality.
type SoftwareService struct {
	firmwareRepo FirmwareRepository
	upgradeRepo  UpgradeTaskRepository
	deviceRepo   device.DeviceRepository
	cmdQueue     cmdqueue.CommandQueue
	connReq      *connreq.Client
	minioClient  *minio.Client
	firmwareBkt  string
	eventBus     event.EventBus
	logger       *zap.Logger
}

// NewSoftwareService creates a new SoftwareService.
func NewSoftwareService(
	firmwareRepo FirmwareRepository,
	upgradeRepo UpgradeTaskRepository,
	deviceRepo device.DeviceRepository,
	cmdQueue cmdqueue.CommandQueue,
	connReq *connreq.Client,
	minioClient *minio.Client,
	firmwareBucket string,
	eventBus event.EventBus,
	logger *zap.Logger,
) *SoftwareService {
	return &SoftwareService{
		firmwareRepo: firmwareRepo,
		upgradeRepo:  upgradeRepo,
		deviceRepo:   deviceRepo,
		cmdQueue:     cmdQueue,
		connReq:      connReq,
		minioClient:  minioClient,
		firmwareBkt:  firmwareBucket,
		eventBus:     eventBus,
		logger:       logger.Named("software"),
	}
}

// UploadFirmware stores a firmware file to MinIO and creates a firmware version record.
func (s *SoftwareService) UploadFirmware(ctx context.Context, fw *FirmwareVersion, file io.Reader, fileSize int64) error {
	// Build MinIO path: firmware/{carrier}/{product_class}/{version}/firmware.bin
	objectPath := fmt.Sprintf("firmware/%s/%s/%s/%s",
		fw.Carrier, fw.ProductClass, fw.Version, fw.FileName)

	_, err := s.minioClient.PutObject(ctx, s.firmwareBkt, objectPath, file, fileSize, minio.PutObjectOptions{
		ContentType: "application/octet-stream",
	})
	if err != nil {
		return fmt.Errorf("upload firmware to MinIO: %w", err)
	}

	fw.MinIOPath = objectPath
	fw.FileSize = fileSize
	if fw.Status == "" {
		fw.Status = "active"
	}

	if err := s.firmwareRepo.Create(ctx, fw); err != nil {
		return fmt.Errorf("create firmware record: %w", err)
	}

	if evt, err := event.NewEvent(event.SubjectFirmwareUploaded, map[string]interface{}{
		"firmware_id": fw.ID.String(),
		"carrier":     string(fw.Carrier),
		"version":     fw.Version,
	}); err == nil {
		if pubErr := s.eventBus.Publish(ctx, event.SubjectFirmwareUploaded, evt); pubErr != nil {
			s.logger.Warn("publish firmware.uploaded event", zap.Error(pubErr))
		}
	}

	return nil
}

// StartUpgrade creates an upgrade task for a single device.
func (s *SoftwareService) StartUpgrade(ctx context.Context, deviceID, firmwareID uuid.UUID) (*UpgradeTask, error) {
	// Verify device exists
	dev, err := s.deviceRepo.GetByID(ctx, deviceID)
	if err != nil {
		return nil, fmt.Errorf("get device: %w", err)
	}

	// Verify firmware exists
	fw, err := s.firmwareRepo.GetByID(ctx, firmwareID)
	if err != nil {
		return nil, fmt.Errorf("get firmware: %w", err)
	}

	// Check no active upgrade already running
	_, err = s.upgradeRepo.GetActiveByDeviceID(ctx, deviceID)
	if err == nil {
		return nil, commonerrors.NewBusinessError(8001, "device already has an active upgrade", commonerrors.ErrAlreadyExists)
	}

	task := &UpgradeTask{
		DeviceID:   deviceID,
		FirmwareID: firmwareID,
		Status:     UpgradePending,
		MaxRetries: 3,
	}

	if err := s.upgradeRepo.Create(ctx, task); err != nil {
		return nil, fmt.Errorf("create upgrade task: %w", err)
	}

	// Transition to downloading
	if err := s.upgradeRepo.UpdateStatus(ctx, task.ID, UpgradeDownloading, ""); err != nil {
		s.logger.Error("transition to downloading", zap.Error(err))
	}
	task.Status = UpgradeDownloading

	// Push Download command to command queue
	downloadURL := fmt.Sprintf("minio://%s/%s", s.firmwareBkt, fw.MinIOPath)
	paramsJSON, marshalErr := json.Marshal(map[string]interface{}{
		"url":       downloadURL,
		"file_type": "1", // firmware
		"file_size": fw.FileSize,
		"file_name": fw.FileName,
	})
	if marshalErr != nil {
		return nil, fmt.Errorf("marshal download params: %w", marshalErr)
	}
	cmd := &cmdqueue.Command{
		Method: "Download",
		Params: paramsJSON,
	}

	if err := s.cmdQueue.Push(ctx, dev.SerialNumber, cmd); err != nil {
		s.logger.Error("push download command", zap.Error(err))
	}

	// Send Connection Request to wake device
	if dev.ConnectionRequestURL != "" {
		if err := s.connReq.Send(ctx, dev.SerialNumber, dev.ConnectionRequestURL); err != nil {
			s.logger.Warn("send connection request", zap.Error(err))
		}
	}

	if evt, err := event.NewEvent(event.SubjectUpgradeStarted, map[string]interface{}{
		"task_id":     task.ID.String(),
		"device_id":   deviceID.String(),
		"firmware_id": firmwareID.String(),
	}); err == nil {
		if pubErr := s.eventBus.Publish(ctx, event.SubjectUpgradeStarted, evt); pubErr != nil {
			s.logger.Warn("publish upgrade.started event", zap.Error(pubErr))
		}
	}

	return task, nil
}

// BatchUpgrade triggers upgrades for multiple devices with configurable concurrency.
func (s *SoftwareService) BatchUpgrade(ctx context.Context, deviceIDs []uuid.UUID, firmwareID uuid.UUID, concurrency int) ([]UpgradeTask, error) {
	if concurrency < 1 {
		concurrency = 5
	}

	batchID := uuid.New()
	tasks := make([]UpgradeTask, 0, len(deviceIDs))

	sem := make(chan struct{}, concurrency)
	results := make(chan UpgradeTask, len(deviceIDs))
	errs := make(chan error, len(deviceIDs))

	for _, deviceID := range deviceIDs {
		sem <- struct{}{}
		go func(did uuid.UUID) {
			defer func() { <-sem }()

			task := &UpgradeTask{
				DeviceID:   did,
				FirmwareID: firmwareID,
				BatchID:    &batchID,
				Status:     UpgradePending,
				MaxRetries: 3,
			}

			if err := s.upgradeRepo.Create(ctx, task); err != nil {
				s.logger.Error("create batch upgrade task", zap.String("device_id", did.String()), zap.Error(err))
				errs <- err
				return
			}

			// Start download for this device
			dev, err := s.deviceRepo.GetByID(ctx, did)
			if err != nil {
				s.logger.Error("get device for batch upgrade", zap.String("device_id", did.String()), zap.Error(err))
				if statusErr := s.upgradeRepo.UpdateStatus(ctx, task.ID, UpgradeFailed, err.Error()); statusErr != nil {
				s.logger.Error("update upgrade status to failed", zap.Error(statusErr))
			}
				task.Status = UpgradeFailed
				results <- *task
				return
			}

			fw, fwErr := s.firmwareRepo.GetByID(ctx, firmwareID)
		if fwErr != nil {
			s.logger.Error("get firmware for batch upgrade", zap.Error(fwErr))
		}
			if fw != nil {
				downloadURL := fmt.Sprintf("minio://%s/%s", s.firmwareBkt, fw.MinIOPath)
				batchParamsJSON, marshalErr := json.Marshal(map[string]interface{}{
					"url":       downloadURL,
					"file_type": "1",
					"file_size": fw.FileSize,
					"file_name": fw.FileName,
				})
				if marshalErr != nil {
					s.logger.Error("marshal batch download params", zap.Error(marshalErr))
				} else {
					cmd := &cmdqueue.Command{
						Method: "Download",
						Params: batchParamsJSON,
					}
					if pushErr := s.cmdQueue.Push(ctx, dev.SerialNumber, cmd); pushErr != nil {
						s.logger.Error("push batch download command", zap.String("device_sn", dev.SerialNumber), zap.Error(pushErr))
					}
				}

				if dev.ConnectionRequestURL != "" {
					if crErr := s.connReq.Send(ctx, dev.SerialNumber, dev.ConnectionRequestURL); crErr != nil {
						s.logger.Warn("send connection request in batch", zap.String("device_sn", dev.SerialNumber), zap.Error(crErr))
					}
				}
			}

			if statusErr := s.upgradeRepo.UpdateStatus(ctx, task.ID, UpgradeDownloading, ""); statusErr != nil {
				s.logger.Error("update upgrade status to downloading", zap.Error(statusErr))
			}
			task.Status = UpgradeDownloading
			results <- *task
		}(deviceID)
	}

	// Collect results
	for range deviceIDs {
		select {
		case task := <-results:
			tasks = append(tasks, task)
		case <-errs:
			// Error already logged, continue collecting
		}
	}

	return tasks, nil
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

	task, err := s.upgradeRepo.GetActiveByDeviceID(ctx, dev.ID)
	if err != nil {
		return nil // No active upgrade, ignore
	}

	// Advance state machine: downloading -> rebooting -> verifying -> completed
	nextState := NextUpgradeState(task.Status)
	if err := ValidateUpgradeTransition(task.Status, nextState); err != nil {
		s.logger.Warn("invalid upgrade transition",
			zap.String("task_id", task.ID.String()),
			zap.String("current", string(task.Status)),
			zap.String("target", string(nextState)),
		)
		return nil
	}

	if err := s.upgradeRepo.UpdateStatus(ctx, task.ID, nextState, ""); err != nil {
		return fmt.Errorf("update upgrade status: %w", err)
	}

	if nextState == UpgradeCompleted {
		if completedEvt, err := event.NewEvent(event.SubjectUpgradeCompleted, map[string]interface{}{
			"task_id":   task.ID.String(),
			"device_id": dev.ID.String(),
		}); err == nil {
			if pubErr := s.eventBus.Publish(ctx, event.SubjectUpgradeCompleted, completedEvt); pubErr != nil {
				s.logger.Warn("publish upgrade.completed event", zap.Error(pubErr))
			}
		}
	}

	return nil
}

// Subscribe registers event subscriptions for the software service.
func (s *SoftwareService) Subscribe(eventBus event.EventBus) error {
	_, err := eventBus.Subscribe(event.SubjectDeviceTransferComplete, func(ctx context.Context, evt event.Event) error {
		return s.HandleTransferComplete(ctx, evt)
	})
	return err
}

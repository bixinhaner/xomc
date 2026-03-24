package provision

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"

	"github.com/omcgo/omcgo/internal/acs/cmdqueue"
	"github.com/omcgo/omcgo/internal/config/datamodel"
	"github.com/omcgo/omcgo/internal/core/appconfig"
	"github.com/omcgo/omcgo/internal/core/model"
	"go.uber.org/zap"
)

// ModelUploadService handles parameter model acquisition via TR-069 Upload RPC.
// When a device has no matching DataModel, the service dispatches an Upload RPC
// (FileType "11") to the CPE, which uploads its parameter model XML to the ACS
// upload endpoint. The XML is stored in MinIO and parsed into a DataModel.
type ModelUploadService struct {
	discoveryRepo ParameterDiscoveryLogRepository
	dmImporter    *datamodel.DataModelImporter
	dmRegistry    *datamodel.DataModelRegistry
	cmdQueue      cmdqueue.CommandQueue
	minioClient   *minio.Client
	config        appconfig.ModelUploadConfig
	logger        *zap.Logger
}

// NewModelUploadService creates a new ModelUploadService.
func NewModelUploadService(
	discoveryRepo ParameterDiscoveryLogRepository,
	dmImporter *datamodel.DataModelImporter,
	dmRegistry *datamodel.DataModelRegistry,
	cmdQueue cmdqueue.CommandQueue,
	minioClient *minio.Client,
	config appconfig.ModelUploadConfig,
	logger *zap.Logger,
) *ModelUploadService {
	return &ModelUploadService{
		discoveryRepo: discoveryRepo,
		dmImporter:    dmImporter,
		dmRegistry:    dmRegistry,
		cmdQueue:      cmdQueue,
		minioClient:   minioClient,
		config:        config,
		logger:        logger.Named("model-upload"),
	}
}

// RequestModelUpload dispatches an Upload RPC command (FileType "11") to the CPE,
// instructing it to upload its parameter model XML to the ACS upload endpoint.
func (s *ModelUploadService) RequestModelUpload(ctx context.Context, dev *model.Device) (*ParameterDiscoveryLog, error) {
	log := NewParameterDiscoveryLog(dev.ID, dev.SerialNumber, dev.OUI, dev.ProductClass, dev.FirmwareVersion)
	log.Status = DiscoveryDiscovering

	if err := s.discoveryRepo.Create(ctx, log); err != nil {
		return nil, fmt.Errorf("create discovery log: %w", err)
	}

	// Generate upload URL with device SN and unique ID in filename.
	filename := fmt.Sprintf("datamodel_%s_%s.xml", dev.SerialNumber, uuid.New().String()[:8])
	uploadURL := fmt.Sprintf("%s?fileType=11&filename=%s", s.config.UploadURL, filename)

	// Build Upload RPC command parameters.
	uploadParams, err := json.Marshal(map[string]interface{}{
		"file_type":       "11 " + dev.OUI + " Parameter Model",
		"url":             uploadURL,
		"username":        s.config.UploadUsername,
		"password":        s.config.UploadPassword,
		"file_size":       0,
		"target_filename": filename,
		"delay_seconds":   0,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal upload params: %w", err)
	}

	cmd := &cmdqueue.Command{
		ID:         uuid.New().String(),
		Method:     "Upload",
		Params:     uploadParams,
		Priority:   1,
		CommandKey: fmt.Sprintf("model-upload-%s", dev.SerialNumber),
	}

	if err := s.cmdQueue.Push(ctx, dev.SerialNumber, cmd); err != nil {
		_ = s.discoveryRepo.UpdateStatus(ctx, log.ID, DiscoveryFailed, err.Error())
		return nil, fmt.Errorf("enqueue Upload command: %w", err)
	}

	s.logger.Info("parameter model upload requested",
		zap.String("device_sn", dev.SerialNumber),
		zap.String("discovery_id", log.ID.String()),
		zap.String("upload_url", uploadURL),
	)

	return log, nil
}

// dataModelFilePayload is the event payload for datamodel.file.received.
type dataModelFilePayload struct {
	MinioBucket     string `json:"minio_bucket"`
	MinioPath       string `json:"minio_path"`
	DeviceID        string `json:"device_id"`
	DeviceSN        string `json:"device_sn"`
	Carrier         string `json:"carrier"`
	Technology      string `json:"technology"`
	OUI             string `json:"oui"`
	ProductClass    string `json:"product_class"`
	FirmwareVersion string `json:"firmware_version"`
	FileSize        int64  `json:"file_size"`
	Filename        string `json:"filename"`
}

// HandleModelFileReceived processes a datamodel.file.received event.
// It downloads the XML from MinIO, parses it, and creates a DataModel.
func (s *ModelUploadService) HandleModelFileReceived(ctx context.Context, dev *model.Device, payload dataModelFilePayload) (*datamodel.DataModel, error) {
	s.logger.Info("handling parameter model file",
		zap.String("device_sn", payload.DeviceSN),
		zap.String("bucket", payload.MinioBucket),
		zap.String("path", payload.MinioPath),
		zap.Int64("file_size", payload.FileSize),
	)

	// Get latest discovery log for this device.
	log, _ := s.discoveryRepo.GetByDeviceID(ctx, dev.ID)
	if log == nil {
		// No discovery log — create one for tracking.
		log = NewParameterDiscoveryLog(dev.ID, dev.SerialNumber, dev.OUI, dev.ProductClass, dev.FirmwareVersion)
		log.Status = DiscoveryDiscovering
		if err := s.discoveryRepo.Create(ctx, log); err != nil {
			s.logger.Warn("create discovery log for model file", zap.Error(err))
			log = nil
		}
	}

	// Download XML from MinIO.
	obj, err := s.minioClient.GetObject(ctx, payload.MinioBucket, payload.MinioPath, minio.GetObjectOptions{})
	if err != nil {
		errMsg := fmt.Sprintf("get object from MinIO: %v", err)
		if log != nil {
			_ = s.discoveryRepo.UpdateStatus(ctx, log.ID, DiscoveryFailed, errMsg)
		}
		return nil, fmt.Errorf("get object from MinIO %s/%s: %w", payload.MinioBucket, payload.MinioPath, err)
	}
	defer obj.Close()

	// Parse XML and create DataModel via importer.
	dm, err := s.dmImporter.ImportFromXMLForCPE(ctx, obj,
		dev.Carrier, dev.ProductClass, dev.FirmwareVersion)
	if err != nil {
		errMsg := fmt.Sprintf("import XML: %v", err)
		if log != nil {
			_ = s.discoveryRepo.UpdateStatus(ctx, log.ID, DiscoveryFailed, errMsg)
		}
		return nil, fmt.Errorf("import parameter model XML: %w", err)
	}

	// Update discovery log.
	if log != nil {
		log.DataModelID = &dm.ID
		log.Status = DiscoveryCompleted
		if err := s.discoveryRepo.Update(ctx, log); err != nil {
			s.logger.Warn("update discovery log after model creation", zap.Error(err))
		}
	}

	// Invalidate cache so the new model is immediately available.
	if dm.IsActive {
		if err := s.dmRegistry.InvalidateCache(ctx, dm); err != nil {
			s.logger.Warn("invalidate cache after model upload", zap.Error(err))
		}
	}

	s.logger.Info("data model created from CPE upload",
		zap.String("device_sn", dev.SerialNumber),
		zap.String("model_id", dm.ID.String()),
		zap.String("status", string(dm.Status)),
	)

	return dm, nil
}

// CleanupState is a no-op for model upload (no Redis state to clean up).
// Kept for interface compatibility with the provisioning engine.
func (s *ModelUploadService) CleanupState(_ context.Context, _ string) {
	// Model upload uses Upload RPC + event-driven flow.
	// No intermediate Redis state to clean up (unlike old GPN discovery).
}

// UploadTimeout returns the configured upload timeout, with a default of 5 minutes.
func (s *ModelUploadService) UploadTimeout() time.Duration {
	if s.config.UploadTimeout > 0 {
		return s.config.UploadTimeout
	}
	return 5 * time.Minute
}

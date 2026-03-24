package transfer

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/device"
	"github.com/omcgo/omcgo/internal/core/event"
)

// TransferBridge subscribes to AutonomousTransferComplete events from the ACS,
// downloads files from the CPE transfer URL, stores them in MinIO, and publishes
// pm.file.received / mr.file.received events to trigger downstream processing.
type TransferBridge struct {
	deviceRepo  device.DeviceRepository
	minioClient *minio.Client
	pmBucket    string
	mrBucket    string
	logsBucket  string
	httpClient  *http.Client
	eventBus    event.EventBus
	logger      *zap.Logger
}

// NewTransferBridge creates a new TransferBridge.
func NewTransferBridge(
	deviceRepo device.DeviceRepository,
	minioClient *minio.Client,
	pmBucket, mrBucket, logsBucket string,
	eventBus event.EventBus,
	logger *zap.Logger,
) *TransferBridge {
	return &TransferBridge{
		deviceRepo:  deviceRepo,
		minioClient: minioClient,
		pmBucket:    pmBucket,
		mrBucket:    mrBucket,
		logsBucket:  logsBucket,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		eventBus: eventBus,
		logger:   logger.Named("transfer-bridge"),
	}
}

// Subscribe registers the bridge to listen for autonomous transfer complete events.
func (b *TransferBridge) Subscribe(bus event.EventBus) error {
	_, err := bus.QueueSubscribe(
		event.SubjectDeviceAutonomousTransferComplete,
		"transfer-bridge",
		b.handleAutonomousTransferComplete,
	)
	if err != nil {
		return fmt.Errorf("subscribe autonomous_transfer_complete: %w", err)
	}
	b.logger.Info("transfer bridge subscribed",
		zap.String("subject", event.SubjectDeviceAutonomousTransferComplete))
	return nil
}

// atcPayload matches the payload published by ACS handler.go handleAutonomousTransferComplete.
type atcPayload struct {
	DeviceSN       string      `json:"device_sn"`
	AnnounceURL    string      `json:"announce_url"`
	TransferURL    string      `json:"transfer_url"`
	IsDownload     bool        `json:"is_download"`
	FileType       string      `json:"file_type"`
	FileSize       int64       `json:"file_size"`
	TargetFileName string      `json:"target_filename"`
	StartTime      string      `json:"start_time"`
	CompleteTime   string      `json:"complete_time"`
	Fault          interface{} `json:"fault,omitempty"`
}

func (b *TransferBridge) handleAutonomousTransferComplete(ctx context.Context, evt event.Event) error {
	var payload atcPayload
	if err := evt.DecodePayload(&payload); err != nil {
		return fmt.Errorf("decode ATC payload: %w", err)
	}

	b.logger.Info("handling autonomous transfer complete",
		zap.String("device_sn", payload.DeviceSN),
		zap.String("file_type", payload.FileType),
		zap.String("transfer_url", payload.TransferURL),
		zap.String("target_filename", payload.TargetFileName),
	)

	// Skip if there was a fault
	if payload.Fault != nil {
		b.logger.Warn("ATC has fault, skipping",
			zap.String("device_sn", payload.DeviceSN),
			zap.Any("fault", payload.Fault),
		)
		return nil
	}

	// Skip if no transfer URL
	if payload.TransferURL == "" {
		b.logger.Warn("ATC has no transfer URL, skipping",
			zap.String("device_sn", payload.DeviceSN))
		return nil
	}

	// Look up device
	dev, err := b.deviceRepo.GetBySerialNumber(ctx, payload.DeviceSN)
	if err != nil {
		return fmt.Errorf("get device by SN %s: %w", payload.DeviceSN, err)
	}
	if dev == nil {
		b.logger.Warn("device not found in DB, skipping file processing",
			zap.String("device_sn", payload.DeviceSN))
		return nil
	}

	// Classify file type
	fileCategory := classifyFileType(payload.FileType, payload.TargetFileName)

	// Determine bucket and object path
	var bucket, objectPath string
	fileName := payload.TargetFileName
	if fileName == "" {
		fileName = filepath.Base(payload.TransferURL)
	}
	now := time.Now()
	datePrefix := now.Format("2006/01/02")

	switch fileCategory {
	case "pm":
		bucket = b.pmBucket
		objectPath = fmt.Sprintf("%s/%s/%s", datePrefix, payload.DeviceSN, fileName)
	case "mr":
		bucket = b.mrBucket
		objectPath = fmt.Sprintf("%s/%s/%s", datePrefix, payload.DeviceSN, fileName)
	case "datamodel":
		bucket = b.logsBucket
		objectPath = fmt.Sprintf("datamodel/%s/%s/%s", datePrefix, payload.DeviceSN, fileName)
	default:
		bucket = b.logsBucket
		objectPath = fmt.Sprintf("%s/%s/%s", datePrefix, payload.DeviceSN, fileName)
	}

	// Download from transfer URL and store in MinIO
	fileSize, err := b.downloadAndStore(ctx, payload.TransferURL, bucket, objectPath)
	if err != nil {
		return fmt.Errorf("download and store file: %w", err)
	}

	b.logger.Info("file stored in MinIO",
		zap.String("bucket", bucket),
		zap.String("path", objectPath),
		zap.Int64("size", fileSize),
		zap.String("category", fileCategory),
	)

	// Publish downstream event based on file category
	switch fileCategory {
	case "pm":
		pmPayload := map[string]interface{}{
			"minio_path": objectPath,
			"device_id":  dev.ID.String(),
			"device_sn":  dev.SerialNumber,
			"carrier":    string(dev.Carrier),
			"technology": string(dev.Technology),
		}
		pmEvt, err := event.NewEvent(event.SubjectPMFileReceived, pmPayload)
		if err != nil {
			return fmt.Errorf("create pm event: %w", err)
		}
		if err := b.eventBus.Publish(ctx, event.SubjectPMFileReceived, pmEvt); err != nil {
			return fmt.Errorf("publish pm.file.received: %w", err)
		}
		b.logger.Info("published pm.file.received",
			zap.String("device_sn", payload.DeviceSN),
			zap.String("path", objectPath))

	case "mr":
		mrPayload := map[string]interface{}{
			"minio_path": objectPath,
			"bucket":     bucket,
			"device_id":  dev.ID.String(),
			"device_sn":  dev.SerialNumber,
			"carrier":    string(dev.Carrier),
			"file_name":  fileName,
			"file_size":  fileSize,
		}
		mrEvt, err := event.NewEvent(event.SubjectMRFileReceived, mrPayload)
		if err != nil {
			return fmt.Errorf("create mr event: %w", err)
		}
		if err := b.eventBus.Publish(ctx, event.SubjectMRFileReceived, mrEvt); err != nil {
			return fmt.Errorf("publish mr.file.received: %w", err)
		}
		b.logger.Info("published mr.file.received",
			zap.String("device_sn", payload.DeviceSN),
			zap.String("path", objectPath))

	case "datamodel":
		dmPayload := map[string]interface{}{
			"minio_bucket": bucket,
			"minio_path":   objectPath,
			"device_id":    dev.ID.String(),
			"device_sn":    dev.SerialNumber,
			"carrier":      string(dev.Carrier),
			"technology":   string(dev.Technology),
			"oui":          dev.OUI,
			"product_class": dev.ProductClass,
			"firmware_version": dev.FirmwareVersion,
			"file_size":    fileSize,
		}
		dmEvt, err := event.NewEvent(event.SubjectDataModelFileReceived, dmPayload)
		if err != nil {
			return fmt.Errorf("create datamodel event: %w", err)
		}
		if err := b.eventBus.Publish(ctx, event.SubjectDataModelFileReceived, dmEvt); err != nil {
			return fmt.Errorf("publish datamodel.file.received: %w", err)
		}
		b.logger.Info("published datamodel.file.received",
			zap.String("device_sn", payload.DeviceSN),
			zap.String("path", objectPath))

	default:
		b.logger.Info("log file stored, no downstream event",
			zap.String("device_sn", payload.DeviceSN),
			zap.String("path", objectPath))
	}

	return nil
}

// classifyFileType determines whether a file is PM, MR, DataModel, or Log based on
// the TR-069 FileType code and the target file name.
func classifyFileType(fileType, fileName string) string {
	ft := strings.ToUpper(strings.TrimSpace(fileType))
	fn := strings.ToUpper(fileName)

	// TR-069 FileType codes: "1"=Firmware, "2"=WebContent, "3"=VendorConfig/Log, "4"=PM, "5"=MR, "11"=ParameterModel
	switch {
	case ft == "11" || strings.Contains(ft, "PARAMETER MODEL"):
		return "datamodel"
	case ft == "4" || strings.Contains(ft, "PM"):
		return "pm"
	case ft == "5" || strings.Contains(ft, "MR"):
		return "mr"
	case strings.Contains(fn, "MRO") || strings.Contains(fn, "MRS") || strings.Contains(fn, "MRE"):
		return "mr"
	case strings.Contains(fn, "PM") || strings.Contains(fn, "COUNTER"):
		return "pm"
	case strings.Contains(fn, "DATAMODEL") || strings.Contains(fn, "PARAMETERMODEL"):
		return "datamodel"
	case ft == "3" || strings.Contains(ft, "LOG"):
		return "log"
	default:
		return "log"
	}
}

// downloadAndStore fetches the file from the given URL and stores it in MinIO.
func (b *TransferBridge) downloadAndStore(ctx context.Context, url, bucket, objectPath string) (int64, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, fmt.Errorf("create request: %w", err)
	}

	resp, err := b.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("HTTP GET %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("HTTP GET %s returned status %d", url, resp.StatusCode)
	}

	info, err := b.minioClient.PutObject(ctx, bucket, objectPath, resp.Body, -1,
		minio.PutObjectOptions{ContentType: "application/octet-stream"})
	if err != nil {
		return 0, fmt.Errorf("put object to MinIO %s/%s: %w", bucket, objectPath, err)
	}

	return info.Size, nil
}

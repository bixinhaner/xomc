package transfer

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/appconfig"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/storage"
	"github.com/omcgo/omcgo/internal/device"
	"github.com/omcgo/omcgo/internal/storageprotection"
	"github.com/omcgo/omcgo/pkg/tr069"
)

// TransferBridge subscribes to AutonomousTransferComplete events from the ACS,
// downloads files from the CPE transfer URL, stores them in MinIO, and publishes
// pm.file.received / mr.file.received events to trigger downstream processing.
type TransferBridge struct {
	deviceRepo  device.DeviceRepository
	minioClient *minio.Client
	buckets     appconfig.BucketConfig
	httpClient  *http.Client
	eventBus    event.EventBus
	deduper     *event.Deduper
	logger      *zap.Logger
	admission   storageprotection.WriteAdmission
}

// NewTransferBridge creates a new TransferBridge.
// deduper 为 nil 时关闭幂等检查（向后兼容单进程内存 EventBus 场景）。
func NewTransferBridge(
	deviceRepo device.DeviceRepository,
	minioClient *minio.Client,
	buckets appconfig.BucketConfig,
	eventBus event.EventBus,
	deduper *event.Deduper,
	logger *zap.Logger,
) *TransferBridge {
	return &TransferBridge{
		deviceRepo:  deviceRepo,
		minioClient: minioClient,
		buckets:     buckets,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		eventBus: eventBus,
		deduper:  deduper,
		logger:   logger.Named("transfer-bridge"),
	}
}

func (b *TransferBridge) SetStorageAdmission(admission storageprotection.WriteAdmission) {
	b.admission = admission
}

// Subscribe registers the bridge to listen for autonomous transfer complete events.
func (b *TransferBridge) Subscribe(bus event.EventBus) error {
	handler := event.EventHandler(b.handleAutonomousTransferComplete)
	if b.deduper != nil {
		handler = b.deduper.Wrap("transfer-autonomous", handler)
	}
	_, err := bus.QueueSubscribe(
		event.SubjectDeviceAutonomousTransferComplete,
		"transfer-bridge",
		handler,
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

	// Classify file type and determine bucket/path via storage package
	ft := classifyFileType(payload.FileType, payload.TargetFileName)
	bucket, category := storage.BucketAndCategory(ft, b.buckets)

	fileName := payload.TargetFileName
	if fileName == "" {
		fileName = filepath.Base(payload.TransferURL)
	}
	carrier := string(dev.Carrier)
	objectPath := storage.ObjectPath(category, carrier, payload.DeviceSN, fileName)

	// Download from transfer URL and store in MinIO
	fileSize, err := b.downloadAndStore(ctx, payload.TransferURL, bucket, objectPath)
	if err != nil {
		return fmt.Errorf("download and store file: %w", err)
	}

	b.logger.Info("file stored in MinIO",
		zap.String("bucket", bucket),
		zap.String("path", objectPath),
		zap.Int64("size", fileSize),
		zap.String("file_type", string(ft)),
	)

	// Publish downstream event based on file type
	switch ft {
	case tr069.FileTypePM:
		pmPayload := map[string]interface{}{
			"minio_path": objectPath,
			"device_id":  dev.ID.String(),
			"device_oui": dev.OUI, // T-0164-P3: TR-069 标准设备唯一标识 (oui, sn) 双键
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

	case tr069.FileTypeMR:
		mrPayload := map[string]interface{}{
			"minio_path": objectPath,
			"bucket":     bucket,
			"device_id":  dev.ID.String(),
			"device_oui": dev.OUI, // T-0164-P3: TR-069 标准设备唯一标识 (oui, sn) 双键
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

	case tr069.FileTypeDataModel:
		dmPayload := map[string]interface{}{
			"minio_bucket":     bucket,
			"minio_path":       objectPath,
			"device_id":        dev.ID.String(),
			"device_sn":        dev.SerialNumber,
			"carrier":          string(dev.Carrier),
			"technology":       string(dev.Technology),
			"oui":              dev.OUI,
			"product_class":    dev.ProductClass,
			"firmware_version": dev.FirmwareVersion,
			"file_size":        fileSize,
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

	case tr069.FileTypeRunningLog, tr069.FileTypeFaultLog:
		// #178/#222：自主传输上传运行日志/故障日志此前落进 default 分支被静默丢弃，
		// 从不发「日志文件已接收」事件。这里补上事件发布，与 ACS 直传路径
		// （acs/upload/handler.go publishLogFileReceivedEvent）发同一 SubjectLogFileReceived。
		// stationlog.Service 只持久化运行日志；故障日志任务/文件状态由文件传输链路维护，
		// 不写入重启记录。两条上传路径互斥（ACS 直传 vs CPE 自主传输），同一文件只经其一。
		taskID8, deviceSN := parseLogFilename(fileName)
		if deviceSN == "" {
			deviceSN = payload.DeviceSN
		}
		logPayload := map[string]interface{}{
			"bucket":      bucket,
			"object_path": objectPath,
			"file_name":   fileName,
			"file_type":   string(ft),
			"file_size":   fileSize,
			"task_id8":    taskID8,
			"device_sn":   deviceSN,
		}
		logEvt, err := event.NewEvent(event.SubjectLogFileReceived, logPayload)
		if err != nil {
			return fmt.Errorf("create log.file.received event: %w", err)
		}
		if err := b.eventBus.Publish(ctx, event.SubjectLogFileReceived, logEvt); err != nil {
			return fmt.Errorf("publish log.file.received: %w", err)
		}
		b.logger.Info("published log.file.received",
			zap.String("device_sn", deviceSN),
			zap.String("file_type", string(ft)),
			zap.String("path", objectPath))

	default:
		b.logger.Info("log file stored, no downstream event",
			zap.String("device_sn", payload.DeviceSN),
			zap.String("path", objectPath))
	}

	return nil
}

// classifyFileType determines the TR069 FileType based on the raw file type code
// and the target file name heuristics.
func classifyFileType(fileType, fileName string) tr069.FileType {
	ft := strings.ToUpper(strings.TrimSpace(fileType))
	fn := strings.ToUpper(fileName)

	switch {
	case ft == "11" || strings.Contains(ft, "PARAMETER MODEL"):
		return tr069.FileTypeDataModel
	case ft == "4" || strings.Contains(ft, "PM"):
		return tr069.FileTypePM
	case ft == "5" || strings.Contains(ft, "MR"):
		return tr069.FileTypeMR
	case ft == "1":
		return tr069.FileTypeFirmware
	case ft == "2":
		return tr069.FileTypePatch
	case ft == "9":
		return tr069.FileTypePCAP
	// #178/#222：故障日志（FileType "8"）此前落进 default→RunningLog，类型误分类；
	// 显式归到 FaultLog，与 ACS 直传路径 fileTypeToLogType("8")→fault 对齐。
	case ft == "8":
		return tr069.FileTypeFaultLog
	// 运行日志（FileType "6"）显式归位，避免依赖 default 兜底。
	case ft == "6":
		return tr069.FileTypeRunningLog
	case strings.Contains(fn, "MRO") || strings.Contains(fn, "MRS") || strings.Contains(fn, "MRE"):
		return tr069.FileTypeMR
	case strings.Contains(fn, "PM") || strings.Contains(fn, "COUNTER"):
		return tr069.FileTypePM
	case strings.Contains(fn, "DATAMODEL") || strings.Contains(fn, "PARAMETERMODEL"):
		return tr069.FileTypeDataModel
	case strings.HasPrefix(fn, "FAULT-") || strings.Contains(fn, "FAULTLOG"):
		return tr069.FileTypeFaultLog
	case ft == "3" || strings.Contains(ft, "LOG"):
		return tr069.FileTypeRunningLog
	default:
		return tr069.FileTypeRunningLog
	}
}

// logFilenameRe 匹配 executor 生成的日志文件名（与 acs/upload/handler.go 一致）：
//
//	运行日志：runtime-{taskID8}-{deviceSN}.tar.gz
//	故障日志：fault-{taskID8}-{deviceSN}.tar.gz
var logFilenameRe = regexp.MustCompile(`^(?:runtime|fault)-([0-9a-f]{8})-(.+)\.tar\.gz$`)

// parseLogFilename 从日志文件名解析 (taskID8, deviceSN)。
// 文件名不匹配（如临时上传）时返回 ("", "")，调用方回退到 payload.DeviceSN。
func parseLogFilename(filename string) (taskID8, deviceSN string) {
	m := logFilenameRe.FindStringSubmatch(filename)
	if len(m) >= 3 {
		return m[1], m[2]
	}
	return "", ""
}

// downloadAndStore fetches the file from the given URL and stores it in MinIO.
func (b *TransferBridge) downloadAndStore(ctx context.Context, url, bucket, objectPath string) (int64, error) {
	if b.admission != nil {
		decision, err := b.admission.Check(ctx, storageprotection.TargetFilesystem, storageprotection.UnifiedStorageTargetID, b.writeScope(bucket))
		if err != nil {
			return 0, fmt.Errorf("storage admission check: %w", err)
		}
		if !decision.Allowed {
			return 0, fmt.Errorf("storage write protected: %s", decision.Reason)
		}
	}
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

func (b *TransferBridge) writeScope(bucket string) storageprotection.WriteScope {
	switch bucket {
	case b.buckets.PMFiles:
		return storageprotection.WriteScopePM
	case b.buckets.MRFiles:
		return storageprotection.WriteScopeMR
	case b.buckets.ConfigBackup:
		return storageprotection.WriteScopeBackup
	case b.buckets.Reports:
		return storageprotection.WriteScopeReport
	default:
		return storageprotection.WriteScopeUpload
	}
}

package provision

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/config/parammodel"
	"github.com/omcgo/omcgo/internal/core/appconfig"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/product"
	"github.com/omcgo/omcgo/internal/task"
)

// ModelUploadService handles parameter model acquisition via TR-069 Upload RPC.
//
// T-0098 P5-01：旧 dmImporter / dmRegistry / datamodel.DataModel 路径已删除。
// 设备上传的参数模型 XML 由 parseCPEEntries 直接解析为 []parammodel.CPEEntry，
// 通过 IntersectService 写 discovered_param_mappings；不再生成 datamodel.DataModel。
//
// enable_filetype11 决策（设计 §1.10）：
//   - false → 不下发 Upload(FileType=11)，discovery_log 标 skipped；Translator 自动降级 default mapping
//   - true 设备支持 → 解析 XML → IntersectService.IntersectCPEModel 写 discovered_param_mappings
//   - true 设备不支持（SOAP Fault / TransferComplete Fault）→ HandleUploadFailed 标 used_default
type ModelUploadService struct {
	discoveryRepo    ParameterDiscoveryLogRepository
	taskSvc          task.Enqueuer
	minioClient      *minio.Client
	productRegistry  *product.Registry
	intersectService *parammodel.IntersectService
	config           appconfig.ModelUploadConfig
	logger           *zap.Logger
	metrics          *Metrics
}

// NewModelUploadService creates a new ModelUploadService.
func NewModelUploadService(
	discoveryRepo ParameterDiscoveryLogRepository,
	taskSvc task.Enqueuer,
	minioClient *minio.Client,
	productReg *product.Registry,
	intersect *parammodel.IntersectService,
	config appconfig.ModelUploadConfig,
	logger *zap.Logger,
) *ModelUploadService {
	return &ModelUploadService{
		discoveryRepo:    discoveryRepo,
		taskSvc:          taskSvc,
		minioClient:      minioClient,
		productRegistry:  productReg,
		intersectService: intersect,
		config:           config,
		logger:           logger.Named("model-upload"),
	}
}

// SetMetrics 注入 provisioning 指标集合，供 discovery_log 状态写库失败计数
// （HIGH-27）。nil 表示禁用（updateDiscoveryStatus 仍记 warn，仅不打点）。
func (s *ModelUploadService) SetMetrics(m *Metrics) {
	s.metrics = m
}

// updateDiscoveryStatus 统一封装 housekeeping 类 discovery_log 状态写库：
// 失败时记 warn + 打点（discovery_status_update_errors_total{operation}），但不
// 阻断主流程（HIGH-27 建议——非关键路径 log+metric，关键路径返回错误由调用方处理）。
func (s *ModelUploadService) updateDiscoveryStatus(
	ctx context.Context, logID uuid.UUID, status DiscoveryStatus, msg, operation, deviceSN string,
) {
	if err := s.discoveryRepo.UpdateStatus(ctx, logID, status, msg); err != nil {
		s.metrics.discoveryStatusUpdateErr(operation)
		s.logger.Warn("failed to update discovery log status",
			zap.String("device_sn", deviceSN),
			zap.String("operation", operation),
			zap.String("log_id", logID.String()),
			zap.String("target_status", string(status)),
			zap.Error(err))
	}
}

// resolveProduct 查找 device 对应的 product 装配件（含 EnableFileType11）。
// 任一前置失败 → 返回 (nil, false)。
func (s *ModelUploadService) resolveProduct(ctx context.Context, dev *model.Device) (*product.Product, bool) {
	if s.productRegistry == nil || dev == nil || dev.ProductClass == "" {
		return nil, false
	}
	match, err := s.productRegistry.MatchProductClass(ctx, dev.ProductClass)
	if err != nil || match == nil || match.Product == nil {
		return nil, false
	}
	return match.Product, true
}

// RequestModelUpload dispatches an Upload RPC command (FileType "11") to the CPE.
//
// 设备 product.EnableFileType11=false → 直接跳过 Upload，标 discovery_log 为
// completed (reason=disabled_by_product_config)，让 Translator 自动降级到默认映射。
func (s *ModelUploadService) RequestModelUpload(ctx context.Context, dev *model.Device, sourceID string) (*ParameterDiscoveryLog, error) {
	log := NewParameterDiscoveryLog(dev.ID, dev.SerialNumber, dev.OUI, dev.ProductClass, dev.FirmwareVersion)
	log.Status = DiscoveryDiscovering

	if err := s.discoveryRepo.Create(ctx, log); err != nil {
		return nil, fmt.Errorf("create discovery log: %w", err)
	}

	if prod, ok := s.resolveProduct(ctx, dev); ok && !prod.EnableFileType11 {
		s.updateDiscoveryStatus(ctx, log.ID, DiscoveryCompleted,
			"skipped: enable_filetype11=false", "skipped_fileupload", dev.SerialNumber)
		s.logger.Info("model upload skipped per product policy",
			zap.String("device_sn", dev.SerialNumber),
			zap.String("product_id", prod.ID.String()),
			zap.String("product_name", prod.Name),
		)
		return log, nil
	}

	filename := fmt.Sprintf("datamodel_%s_%s.xml", dev.SerialNumber, uuid.New().String()[:8])
	uploadURL := fmt.Sprintf("%s?fileType=11&filename=%s", s.config.UploadURL, filename)

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

	if _, err := s.taskSvc.CreateTask(ctx, &task.CreateTaskRequest{
		DeviceSN:   dev.SerialNumber,
		Method:     "Upload",
		Params:     uploadParams,
		Priority:   1,
		CommandKey: fmt.Sprintf("model-upload-%s", dev.SerialNumber),
		Source:     task.TaskSourceSystem,
		SourceID:   sourceID,
	}); err != nil {
		// 关键路径：CreateTask 失败的错误向上传播给调用方（HIGH-27）。
		// 写 DiscoveryFailed 是 best-effort 旁路标记，其自身写库失败不应掩盖
		// 真正的 CreateTask 错误，故用 helper 记 warn+metric 后仍返回原错误。
		s.updateDiscoveryStatus(ctx, log.ID, DiscoveryFailed, err.Error(),
			"create_task_failed", dev.SerialNumber)
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
//
// 注意：subject 字符串保留 "datamodel.file.received"（transfer/bridge.go 发出），
// 仅 Go 类型变量名延用 dataModelFilePayload。
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
//
// 流程：
//  1. GetObject 从 MinIO 拉取上传的 XML
//  2. parseCPEEntries 解析为 []CPEEntry
//  3. resolveProduct 反查 product，IntersectService.IntersectCPEModel 写 discovered_param_mappings
//  4. discovery_log → completed
//
// 任一前置失败 → discovery_log → failed；不再返回 datamodel.DataModel。
func (s *ModelUploadService) HandleModelFileReceived(ctx context.Context, dev *model.Device, payload dataModelFilePayload) error {
	s.logger.Info("handling parameter model file",
		zap.String("device_sn", payload.DeviceSN),
		zap.String("bucket", payload.MinioBucket),
		zap.String("path", payload.MinioPath),
		zap.Int64("file_size", payload.FileSize),
	)

	log, _ := s.discoveryRepo.GetByDeviceID(ctx, dev.ID)
	if log == nil {
		log = NewParameterDiscoveryLog(dev.ID, dev.SerialNumber, dev.OUI, dev.ProductClass, dev.FirmwareVersion)
		log.Status = DiscoveryDiscovering
		if err := s.discoveryRepo.Create(ctx, log); err != nil {
			s.logger.Warn("create discovery log for model file", zap.Error(err))
			log = nil
		}
	}

	obj, err := s.minioClient.GetObject(ctx, payload.MinioBucket, payload.MinioPath, minio.GetObjectOptions{})
	if err != nil {
		errMsg := fmt.Sprintf("get object from MinIO: %v", err)
		if log != nil {
			s.updateDiscoveryStatus(ctx, log.ID, DiscoveryFailed, errMsg, "minio_get_failed", dev.SerialNumber)
		}
		return fmt.Errorf("get object from MinIO %s/%s: %w", payload.MinioBucket, payload.MinioPath, err)
	}
	defer obj.Close()

	entries, err := parseCPEEntries(obj)
	if err != nil {
		errMsg := fmt.Sprintf("parse XML: %v", err)
		if log != nil {
			s.updateDiscoveryStatus(ctx, log.ID, DiscoveryFailed, errMsg, "parse_xml_failed", dev.SerialNumber)
		}
		return fmt.Errorf("parse parameter model XML: %w", err)
	}

	prod, ok := s.resolveProduct(ctx, dev)
	if !ok {
		errMsg := fmt.Sprintf("cannot resolve product for productClass=%s", dev.ProductClass)
		if log != nil {
			s.updateDiscoveryStatus(ctx, log.ID, DiscoveryFailed, errMsg, "product_unresolved", dev.SerialNumber)
		}
		return fmt.Errorf("%s", errMsg)
	}
	if dev.FirmwareVersion == "" {
		errMsg := "empty firmware version"
		if log != nil {
			s.updateDiscoveryStatus(ctx, log.ID, DiscoveryFailed, errMsg, "empty_firmware", dev.SerialNumber)
		}
		return fmt.Errorf("%s", errMsg)
	}

	if s.intersectService == nil {
		errMsg := "intersect service not configured"
		if log != nil {
			s.updateDiscoveryStatus(ctx, log.ID, DiscoveryFailed, errMsg, "intersect_unconfigured", dev.SerialNumber)
		}
		return fmt.Errorf("%s", errMsg)
	}

	res, err := s.intersectService.IntersectCPEModel(ctx, parammodel.IntersectInput{
		ProductID:       prod.ID,
		SoftwareVersion: dev.FirmwareVersion,
		Entries:         entries,
	})
	if err != nil {
		errMsg := fmt.Sprintf("intersect cpe model: %v", err)
		if log != nil {
			s.updateDiscoveryStatus(ctx, log.ID, DiscoveryFailed, errMsg, "intersect_failed", dev.SerialNumber)
		}
		return fmt.Errorf("intersect cpe model: %w", err)
	}

	if log != nil {
		log.Status = DiscoveryCompleted
		if err := s.discoveryRepo.Update(ctx, log); err != nil {
			s.logger.Warn("update discovery log after intersect", zap.Error(err))
		}
	}

	s.logger.Info("path-c intersect completed",
		zap.String("device_sn", dev.SerialNumber),
		zap.String("product_id", res.ProductID.String()),
		zap.String("software_version", res.SoftwareVersion),
		zap.Int("default_count", res.DefaultCount),
		zap.Int("uploaded_count", res.UploadedCount),
		zap.Int("matched", res.Matched),
	)
	return nil
}

// HandleUploadFailed 处理"设备不支持 Upload(FileType=11)"的兜底（设计 §1.10）。
//
// 触发场景：CPE 返回 SOAP Fault / TransferComplete Fault；调用方监听
// command.upload.response 与 device.inform.transfer_complete 事件，匹配 model-upload
// CommandKey 时调本方法。
func (s *ModelUploadService) HandleUploadFailed(ctx context.Context, dev *model.Device, reason string) error {
	log, _ := s.discoveryRepo.GetByDeviceID(ctx, dev.ID)
	if log == nil {
		return nil
	}
	msg := "used_default: " + reason
	if err := s.discoveryRepo.UpdateStatus(ctx, log.ID, DiscoveryCompleted, msg); err != nil {
		return fmt.Errorf("update discovery log used_default: %w", err)
	}
	s.logger.Info("model upload failed; falling back to default mapping",
		zap.String("device_sn", dev.SerialNumber),
		zap.String("reason", reason),
	)
	return nil
}

// CleanupState is a no-op for model upload (no Redis state to clean up).
// Kept for interface compatibility with the provisioning engine.
func (s *ModelUploadService) CleanupState(_ context.Context, _ string) {}

// UploadTimeout returns the configured upload timeout, with a default of 5 minutes.
func (s *ModelUploadService) UploadTimeout() time.Duration {
	if s.config.UploadTimeout > 0 {
		return s.config.UploadTimeout
	}
	return 5 * time.Minute
}

package provision

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/config/datamodel"
	"github.com/omcgo/omcgo/internal/config/parammodel"
	"github.com/omcgo/omcgo/internal/core/appconfig"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/product"
	"github.com/omcgo/omcgo/internal/task"
)

// ModelUploadService handles parameter model acquisition via TR-069 Upload RPC.
// When a device has no matching DataModel, the service dispatches an Upload RPC
// (FileType "11") to the CPE, which uploads its parameter model XML to the ACS
// upload endpoint. The XML is stored in MinIO and parsed into a DataModel.
//
// T-0098 P2-06：双栈期 dataModel 与 paramRegistry 共存。enable_filetype11 决策：
//   - false → 不下发 Upload(FileType=11)，discovery_log 标 skipped；Translator 自动
//     降级 default mapping
//   - true 设备支持 → 解析 XML → IntersectService.IntersectCPEModel 写 discovered_param_mappings
//   - true 设备不支持（SOAP Fault / TransferComplete Fault）→ HandleUploadFailed
//     标 used_default
type ModelUploadService struct {
	discoveryRepo        ParameterDiscoveryLogRepository
	dmImporter           *datamodel.DataModelImporter
	dmRegistry           *datamodel.DataModelRegistry
	taskSvc              task.Enqueuer
	minioClient          *minio.Client
	productRegistry      *product.Registry
	intersectService     *parammodel.IntersectService
	paramRegistryEnabled bool
	config               appconfig.ModelUploadConfig
	logger               *zap.Logger
}

// NewModelUploadService creates a new ModelUploadService.
func NewModelUploadService(
	discoveryRepo ParameterDiscoveryLogRepository,
	dmImporter *datamodel.DataModelImporter,
	dmRegistry *datamodel.DataModelRegistry,
	taskSvc task.Enqueuer,
	minioClient *minio.Client,
	config appconfig.ModelUploadConfig,
	logger *zap.Logger,
) *ModelUploadService {
	return &ModelUploadService{
		discoveryRepo: discoveryRepo,
		dmImporter:    dmImporter,
		dmRegistry:    dmRegistry,
		taskSvc:       taskSvc,
		minioClient:   minioClient,
		config:        config,
		logger:        logger.Named("model-upload"),
	}
}

// WithParamRegistry 启用 T-0098 P2-06 dual-stack（enable_filetype11 决策 + Intersect 落库）。
//
// enabled=false 或 prodReg / intersect 任一 nil → 等价于不调用本方法（沿用旧行为）。
func (s *ModelUploadService) WithParamRegistry(prodReg *product.Registry, intersect *parammodel.IntersectService, enabled bool) *ModelUploadService {
	s.productRegistry = prodReg
	s.intersectService = intersect
	s.paramRegistryEnabled = enabled && prodReg != nil && intersect != nil
	return s
}

// resolveProduct 公共助手：返回 device 对应的 product 装配件（含 EnableFileType11）。
//
// 任一前置失败 → 返回 (nil, false)。
func (s *ModelUploadService) resolveProduct(ctx context.Context, dev *model.Device) (*product.Product, bool) {
	if !s.paramRegistryEnabled || s.productRegistry == nil {
		return nil, false
	}
	if dev == nil || dev.ProductClass == "" {
		return nil, false
	}
	match, err := s.productRegistry.MatchProductClass(ctx, dev.ProductClass)
	if err != nil || match == nil || match.Product == nil {
		return nil, false
	}
	return match.Product, true
}

// RequestModelUpload dispatches an Upload RPC command (FileType "11") to the CPE,
// instructing it to upload its parameter model XML to the ACS upload endpoint.
//
// T-0098 P2-06：dual-stack 启用且 product.EnableFileType11=false → 直接跳过 Upload，
// 标 discovery_log 为 completed（reason=disabled_by_product_config），让上游 Translator
// 自动降级到默认映射。
func (s *ModelUploadService) RequestModelUpload(ctx context.Context, dev *model.Device) (*ParameterDiscoveryLog, error) {
	log := NewParameterDiscoveryLog(dev.ID, dev.SerialNumber, dev.OUI, dev.ProductClass, dev.FirmwareVersion)
	log.Status = DiscoveryDiscovering

	if err := s.discoveryRepo.Create(ctx, log); err != nil {
		return nil, fmt.Errorf("create discovery log: %w", err)
	}

	// T-0098 P2-06：enable_filetype11 三态决策。false → skip + return early。
	if prod, ok := s.resolveProduct(ctx, dev); ok && !prod.EnableFileType11 {
		_ = s.discoveryRepo.UpdateStatus(ctx, log.ID, DiscoveryCompleted, "skipped: enable_filetype11=false")
		s.logger.Info("model upload skipped per product policy",
			zap.String("device_sn", dev.SerialNumber),
			zap.String("product_id", prod.ID.String()),
			zap.String("product_name", prod.Name),
		)
		return log, nil
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

	if _, err := s.taskSvc.CreateTask(ctx, &task.CreateTaskRequest{
		DeviceSN:   dev.SerialNumber,
		Method:     "Upload",
		Params:     uploadParams,
		Priority:   1,
		CommandKey: fmt.Sprintf("model-upload-%s", dev.SerialNumber),
		Source:     task.TaskSourceSystem,
	}); err != nil {
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

	// T-0098 P2-06：dual-stack 启用时，并行调用 Intersect 写 discovered_param_mappings。
	// 失败仅 WARN（旧 datamodel 已落地，不阻塞主流程）。
	if s.paramRegistryEnabled && s.intersectService != nil {
		if intErr := s.intersectFromDataModel(ctx, dev, dm); intErr != nil {
			s.logger.Warn("path-c intersect failed (non-fatal)",
				zap.Error(intErr),
				zap.String("device_sn", dev.SerialNumber),
			)
		}
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

// HandleUploadFailed 处理"设备不支持 Upload(FileType=11)"的兜底（设计 §1.10）。
//
// 触发场景：CPE 返回 SOAP Fault / TransferComplete Fault；调用方监听
// command.upload.response 与 device.inform.transfer_complete 事件，匹配 model-upload
// CommandKey 时调本方法。
//
// 行为：discovery_log → completed（带 used_default 标记），不写 discovered_param_mappings；
// Translator 后续自动降级到 param_mappings 默认映射。
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

// intersectFromDataModel 把已解析的 DataModel.ParameterTree 转为 CPEEntry 集合并调
// IntersectService.IntersectCPEModel 写 discovered_param_mappings。
//
// 复用 dmImporter 已经解析出的 datamodel.Parameter 列表（含 path/type/writable + Constraints
// min/max），减少 XML 二次解析。
func (s *ModelUploadService) intersectFromDataModel(ctx context.Context, dev *model.Device, dm *datamodel.DataModel) error {
	prod, ok := s.resolveProduct(ctx, dev)
	if !ok {
		return fmt.Errorf("intersect skipped: cannot resolve product for %s", dev.ProductClass)
	}
	if dev.FirmwareVersion == "" {
		return fmt.Errorf("intersect skipped: empty firmware version")
	}

	var dmParams []datamodel.Parameter
	if err := json.Unmarshal(dm.ParameterTree, &dmParams); err != nil {
		return fmt.Errorf("unmarshal data model parameter_tree: %w", err)
	}
	entries := make([]parammodel.CPEEntry, 0, len(dmParams))
	for _, p := range dmParams {
		entries = append(entries, parammodel.CPEEntry{
			PrivatePath:   p.Path,
			EntryType:     "parameter",
			Access:        accessFromWritable(p.Writable),
			DataType:      p.Type,
			ChangeApplies: p.ChangeApplies,
			MinValue:      cpeMinValue(p.Constraints),
			MaxValue:      cpeMaxValue(p.Constraints),
		})
	}

	// 也把 Objects 转为 CPEEntry（entry_type=object）便于 mapping 匹配
	var objs []datamodel.ObjectInfo
	if dm.ObjectTree != nil {
		_ = json.Unmarshal(dm.ObjectTree, &objs)
	}
	for _, o := range objs {
		entries = append(entries, parammodel.CPEEntry{
			PrivatePath: o.Name,
			EntryType:   "object",
			Access:      o.Access,
		})
	}

	res, err := s.intersectService.IntersectCPEModel(ctx, parammodel.IntersectInput{
		ProductID:       prod.ID,
		SoftwareVersion: dev.FirmwareVersion,
		Entries:         entries,
	})
	if err != nil {
		return fmt.Errorf("intersect cpe model: %w", err)
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

// accessFromWritable / cpeMinValue / cpeMaxValue 是 datamodel.Parameter → CPEEntry 适配 helper。
func accessFromWritable(w bool) string {
	if w {
		return "readWrite"
	}
	return "readOnly"
}

func cpeMinValue(c *datamodel.Constraints) *int64 {
	if c == nil || c.MinValue == nil {
		return nil
	}
	v := *c.MinValue
	return &v
}

func cpeMaxValue(c *datamodel.Constraints) *int64 {
	if c == nil || c.MaxValue == nil {
		return nil
	}
	v := *c.MaxValue
	return &v
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

package collector

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
	coremodel "github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/mr"
	"github.com/omcgo/omcgo/internal/mr/parser"
	"go.uber.org/zap"
)

// MRFilePayload is the event payload for MR file received events.
type MRFilePayload struct {
	MinioPath  string `json:"minio_path"`
	Bucket     string `json:"bucket"`
	DeviceID   string `json:"device_id"`
	DeviceSN   string `json:"device_sn"`
	Carrier    string `json:"carrier"`
	FileName   string `json:"file_name"`
	FileSize   int64  `json:"file_size"`
}

// DeviceLookup 抽象按 SN 查设备的能力。device.DeviceRepository 满足。
// 用于 payload.DeviceID 为空时（如 acs/upload 直传路径）由 collector 自己查。
type DeviceLookup interface {
	GetBySerialNumber(ctx context.Context, sn string) (*coremodel.Device, error)
}

// MRCollector handles MR file download, type detection, parsing, and storage.
type MRCollector struct {
	minioClient *minio.Client
	bucket      string
	parsers     map[string]parser.MRParser
	store       mr.MRStore
	devices     DeviceLookup // 可 nil；nil 时强制要求 payload.DeviceID 非空
	eventBus    event.EventBus
	logger      *zap.Logger
}

// NewMRCollector creates a new MR file collector.
func NewMRCollector(
	minioClient *minio.Client,
	bucket string,
	store mr.MRStore,
	eventBus event.EventBus,
	logger *zap.Logger,
) *MRCollector {
	parsers := map[string]parser.MRParser{
		MRTypeMRO: parser.NewMROParser(),
		MRTypeMRS: parser.NewMRSParser(),
		MRTypeMRE: parser.NewMREParser(),
	}
	return &MRCollector{
		minioClient: minioClient,
		bucket:      bucket,
		parsers:     parsers,
		store:       store,
		eventBus:    eventBus,
		logger:      logger,
	}
}

// SetDeviceLookup 注入设备查询能力（可选）。注入后 payload.DeviceID 为空时
// collector 会按 device_sn 反查 device_id / carrier。
// 不注入则 payload.DeviceID 必须非空（保留旧 transfer/bridge 路径行为）。
func (c *MRCollector) SetDeviceLookup(d DeviceLookup) {
	c.devices = d
}

// Subscribe registers the collector for MR file received events.
func (c *MRCollector) Subscribe(eventBus event.EventBus) error {
	_, err := eventBus.QueueSubscribe(
		event.SubjectMRFileReceived,
		"mr-workers",
		c.handleFileReceived,
	)
	if err != nil {
		return fmt.Errorf("subscribe mr.file.received: %w", err)
	}
	c.logger.Info("MR collector subscribed", zap.String("subject", event.SubjectMRFileReceived))
	return nil
}

func (c *MRCollector) handleFileReceived(ctx context.Context, evt event.Event) error {
	var payload MRFilePayload
	if err := evt.DecodePayload(&payload); err != nil {
		c.logger.Error("decode MR file payload", zap.Error(err))
		return err
	}

	c.logger.Info("processing MR file",
		zap.String("file", payload.FileName),
		zap.String("device_sn", payload.DeviceSN))

	// Detect MR type
	mrType, err := DetectMRType(payload.FileName)
	if err != nil {
		c.logger.Error("detect MR type", zap.Error(err), zap.String("file", payload.FileName))
		return err
	}

	var deviceID uuid.UUID
	carrier := payload.Carrier
	if payload.DeviceID != "" {
		var perr error
		deviceID, perr = uuid.Parse(payload.DeviceID)
		if perr != nil {
			return fmt.Errorf("parse device_id: %w", perr)
		}
	} else if c.devices != nil && payload.DeviceSN != "" {
		// upload handler 直传路径：payload 不带 device_id，按 SN 反查
		dev, derr := c.devices.GetBySerialNumber(ctx, payload.DeviceSN)
		if derr != nil {
			return fmt.Errorf("lookup device by sn %s: %w", payload.DeviceSN, derr)
		}
		if dev == nil {
			c.logger.Warn("MR file uploaded but device not found; skipping",
				zap.String("device_sn", payload.DeviceSN),
				zap.String("file", payload.FileName))
			return nil
		}
		deviceID = dev.ID
		if carrier == "" {
			carrier = string(dev.Carrier)
		}
	} else {
		return fmt.Errorf("mr file payload missing device_id and no DeviceLookup wired")
	}

	// Save file metadata
	fileID := uuid.New()
	now := time.Now()
	bucket := payload.Bucket
	if bucket == "" {
		bucket = c.bucket
	}

	fileInfo := &mr.MRFileInfo{
		ID:          fileID,
		DeviceID:    deviceID,
		DeviceSN:    payload.DeviceSN,
		Carrier:     carrier,
		MRType:      mrType,
		FileName:    payload.FileName,
		FileSize:    payload.FileSize,
		CollectTime: now,
		MinioPath:   payload.MinioPath,
		CreatedAt:   now,
	}

	if err := c.store.SaveFile(ctx, fileInfo); err != nil {
		return fmt.Errorf("save MR file info: %w", err)
	}

	// Download from MinIO
	obj, err := c.minioClient.GetObject(ctx, bucket, payload.MinioPath, minio.GetObjectOptions{})
	if err != nil {
		return fmt.Errorf("get MinIO object %s: %w", payload.MinioPath, err)
	}
	defer obj.Close()

	// Parse
	p, ok := c.parsers[mrType]
	if !ok {
		return fmt.Errorf("no parser for MR type: %s", mrType)
	}

	carrierCode := model.CarrierCode(carrier)
	data, err := p.Parse(obj, carrierCode)
	if err != nil {
		c.logger.Warn("parse MR file",
			zap.Error(err),
			zap.String("file", payload.FileName),
			zap.String("type", mrType))
		return err
	}

	// Store parsed records
	if len(data.Records) > 0 {
		if err := c.store.BatchInsertRecords(ctx, fileID, deviceID, mrType, data.Records); err != nil {
			return fmt.Errorf("batch insert MR records: %w", err)
		}
	}

	// Update file as parsed
	if err := c.store.UpdateFileParsed(ctx, fileID, len(data.Records)); err != nil {
		c.logger.Warn("update file parsed status", zap.Error(err))
	}

	// Publish parsed event
	parsedPayload := map[string]interface{}{
		"file_id":      fileID.String(),
		"device_id":    payload.DeviceID,
		"device_sn":    payload.DeviceSN,
		"mr_type":      mrType,
		"record_count": len(data.Records),
		"file_name":    filepath.Base(payload.FileName),
	}
	parsedEvt, err := event.NewEvent(event.SubjectMRFileParsed, parsedPayload)
	if err == nil {
		if pubErr := c.eventBus.Publish(ctx, event.SubjectMRFileParsed, parsedEvt); pubErr != nil {
			c.logger.Warn("publish mr.file.parsed", zap.Error(pubErr))
		}
	}

	c.logger.Info("MR file processed",
		zap.String("file_id", fileID.String()),
		zap.String("type", mrType),
		zap.Int("records", len(data.Records)))

	return nil
}

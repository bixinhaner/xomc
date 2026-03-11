package collector

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/omcgo/omcgo/internal/event"
	"github.com/omcgo/omcgo/internal/model"
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

// MRCollector handles MR file download, type detection, parsing, and storage.
type MRCollector struct {
	minioClient *minio.Client
	bucket      string
	parsers     map[string]parser.MRParser
	store       mr.MRStore
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

	deviceID, err := uuid.Parse(payload.DeviceID)
	if err != nil {
		return fmt.Errorf("parse device_id: %w", err)
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
		Carrier:     payload.Carrier,
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

	carrierCode := model.CarrierCode(payload.Carrier)
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

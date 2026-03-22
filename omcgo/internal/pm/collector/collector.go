package collector

import (
	"context"
	"fmt"
	"path"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/pm"
	"github.com/omcgo/omcgo/internal/pm/counter"
	"github.com/omcgo/omcgo/internal/pm/kpi"
	"go.uber.org/zap"
)

// FileReceivedPayload is the event payload for pm.file.received.
type FileReceivedPayload struct {
	MinIOPath  string `json:"minio_path"`
	DeviceID   string `json:"device_id"`
	DeviceSN   string `json:"device_sn"`
	Carrier    string `json:"carrier"`
	Technology string `json:"technology"`
}

// PMCollector handles PM file processing: download from MinIO, parse XML, store counters.
type PMCollector struct {
	minioClient *minio.Client
	bucket      string
	parser      *PMXMLParser
	counterRepo counter.CounterRepository
	kpiEngine   *kpi.KPIEngine
	fileStore   pm.PMFileStore
	eventBus    event.EventBus
	metrics     *pm.PMMetrics
	logger      *zap.Logger
}

// NewPMCollector creates a new PM collector.
func NewPMCollector(
	minioClient *minio.Client, bucket string, parser *PMXMLParser,
	counterRepo counter.CounterRepository, kpiEngine *kpi.KPIEngine,
	fileStore pm.PMFileStore,
	eventBus event.EventBus, logger *zap.Logger,
) *PMCollector {
	return &PMCollector{
		minioClient: minioClient, bucket: bucket, parser: parser,
		counterRepo: counterRepo, kpiEngine: kpiEngine,
		fileStore: fileStore,
		eventBus: eventBus, logger: logger,
	}
}

// SetMetrics attaches Prometheus metrics to the collector.
func (c *PMCollector) SetMetrics(m *pm.PMMetrics) {
	c.metrics = m
}

// Subscribe registers the collector to listen for PM file received events.
func (c *PMCollector) Subscribe(bus event.EventBus) error {
	_, err := bus.QueueSubscribe(event.SubjectPMFileReceived, "pm-workers", c.handleFileReceived)
	if err != nil {
		return fmt.Errorf("subscribe pm.file.received: %w", err)
	}
	c.logger.Info("PM collector subscribed to pm.file.received")
	return nil
}

func (c *PMCollector) handleFileReceived(ctx context.Context, evt event.Event) error {
	startTime := time.Now()

	var payload FileReceivedPayload
	if err := evt.DecodePayload(&payload); err != nil {
		return fmt.Errorf("decode payload: %w", err)
	}
	c.logger.Info("processing PM file",
		zap.String("path", payload.MinIOPath),
		zap.String("device_sn", payload.DeviceSN),
	)

	deviceID, err := uuid.Parse(payload.DeviceID)
	if err != nil {
		return fmt.Errorf("parse device_id: %w", err)
	}

	obj, err := c.minioClient.GetObject(ctx, c.bucket, payload.MinIOPath, minio.GetObjectOptions{})
	if err != nil {
		return fmt.Errorf("download pm file: %w", err)
	}
	defer obj.Close()

	// Save file metadata to pm_files table before parsing
	var fileID uuid.UUID
	if c.fileStore != nil {
		var fileSize int64
		if stat, statErr := obj.Stat(); statErr == nil {
			fileSize = stat.Size
		}
		now := time.Now()
		pmFile := &pm.PMFileInfo{
			DeviceID:    deviceID,
			DeviceSN:    payload.DeviceSN,
			Carrier:     payload.Carrier,
			Technology:  payload.Technology,
			FileName:    path.Base(payload.MinIOPath),
			FileSize:    fileSize,
			CollectTime: now,
			MinioPath:   payload.MinIOPath,
		}
		if err := c.fileStore.SaveFile(ctx, pmFile); err != nil {
			c.logger.Warn("save PM file metadata", zap.Error(err))
		} else {
			fileID = pmFile.ID
		}
	}

	content, err := c.parser.Parse(obj, deviceID)
	if err != nil {
		if c.metrics != nil {
			c.metrics.FilesProcessedTotal.WithLabelValues("failed").Inc()
			c.metrics.ProcessingDurationSecs.Observe(time.Since(startTime).Seconds())
		}
		return fmt.Errorf("parse pm xml: %w", err)
	}

	c.logger.Info("parsed PM file", zap.Int("counters", len(content.Counters)))

	if err := c.counterRepo.BatchInsert(ctx, content.Counters); err != nil {
		if c.metrics != nil {
			c.metrics.FilesProcessedTotal.WithLabelValues("failed").Inc()
			c.metrics.ProcessingDurationSecs.Observe(time.Since(startTime).Seconds())
		}
		return fmt.Errorf("batch insert counters: %w", err)
	}

	// Record success metrics
	if c.metrics != nil {
		c.metrics.FilesProcessedTotal.WithLabelValues("success").Inc()
		c.metrics.ProcessingDurationSecs.Observe(time.Since(startTime).Seconds())
	}

	// Update file parsed status
	if c.fileStore != nil && fileID != uuid.Nil {
		if err := c.fileStore.UpdateFileParsed(ctx, fileID, len(content.Counters)); err != nil {
			c.logger.Warn("update PM file parsed status", zap.Error(err))
		}
	}

	// Calculate KPIs
	if c.kpiEngine != nil {
		cellIDs := extractUniqueCellIDs(content.Counters)
		carrier := model.CarrierCode(payload.Carrier)
		tech := model.Technology(payload.Technology)
		for _, cellID := range cellIDs {
			if _, err := c.kpiEngine.CalculateAndStore(ctx, deviceID, cellID, content.CollectTime, carrier, tech); err != nil {
				c.logger.Warn("calculate kpi", zap.String("cell_id", cellID), zap.Error(err))
			}
		}
	}

	parsedEvt, err := event.NewEvent(event.SubjectPMFileParsed, map[string]interface{}{
		"minio_path": payload.MinIOPath, "device_id": payload.DeviceID, "counter_count": len(content.Counters),
	})
	if err == nil {
		_ = c.eventBus.Publish(ctx, event.SubjectPMFileParsed, parsedEvt)
	}
	return nil
}

func extractUniqueCellIDs(counters []model.PMCounter) []string {
	seen := make(map[string]struct{})
	var result []string
	for _, c := range counters {
		if _, ok := seen[c.CellID]; !ok {
			seen[c.CellID] = struct{}{}
			result = append(result, c.CellID)
		}
	}
	return result
}

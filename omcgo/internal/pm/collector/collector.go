package collector

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/omcgo/omcgo/internal/event"
	"github.com/omcgo/omcgo/internal/model"
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
	eventBus    event.EventBus
	logger      *zap.Logger
}

// NewPMCollector creates a new PM collector.
func NewPMCollector(
	minioClient *minio.Client, bucket string, parser *PMXMLParser,
	counterRepo counter.CounterRepository, kpiEngine *kpi.KPIEngine,
	eventBus event.EventBus, logger *zap.Logger,
) *PMCollector {
	return &PMCollector{
		minioClient: minioClient, bucket: bucket, parser: parser,
		counterRepo: counterRepo, kpiEngine: kpiEngine,
		eventBus: eventBus, logger: logger,
	}
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

	content, err := c.parser.Parse(obj, deviceID)
	if err != nil {
		return fmt.Errorf("parse pm xml: %w", err)
	}

	c.logger.Info("parsed PM file", zap.Int("counters", len(content.Counters)))

	if err := c.counterRepo.BatchInsert(ctx, content.Counters); err != nil {
		return fmt.Errorf("batch insert counters: %w", err)
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

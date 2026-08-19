package report

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/alarm"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/pm/kpi"
	"github.com/omcgo/omcgo/internal/storageprotection"
)

// ReportGenerator subscribes to report.generate.requested events and
// produces JSON report files stored in MinIO.
type ReportGenerator struct {
	recordRepo       RecordRepository
	defRepo          DefinitionRepository
	kpiRepo          kpi.KPIRepository
	alarmStore       alarm.AlarmStore
	minioClient      *minio.Client
	reportBkt        string
	eventBus         event.EventBus
	logger           *zap.Logger
	storageAdmission storageprotection.WriteAdmission
}

func (g *ReportGenerator) SetStorageAdmission(admission storageprotection.WriteAdmission) {
	g.storageAdmission = admission
}

// NewReportGenerator creates a new ReportGenerator.
func NewReportGenerator(
	recordRepo RecordRepository,
	defRepo DefinitionRepository,
	kpiRepo kpi.KPIRepository,
	alarmStore alarm.AlarmStore,
	minioClient *minio.Client,
	reportBkt string,
	eventBus event.EventBus,
	logger *zap.Logger,
) *ReportGenerator {
	return &ReportGenerator{
		recordRepo:  recordRepo,
		defRepo:     defRepo,
		kpiRepo:     kpiRepo,
		alarmStore:  alarmStore,
		minioClient: minioClient,
		reportBkt:   reportBkt,
		eventBus:    eventBus,
		logger:      logger.Named("report-generator"),
	}
}

// Subscribe registers the generator as listener for report generation events.
func (g *ReportGenerator) Subscribe(bus event.EventBus) error {
	_, err := bus.QueueSubscribe(event.SubjectReportGenerateRequested, "report-generators", g.handleGenerateRequest)
	return err
}

func (g *ReportGenerator) handleGenerateRequest(ctx context.Context, evt event.Event) error {
	var payload struct {
		RecordID     string `json:"record_id"`
		DefinitionID string `json:"definition_id"`
	}
	if err := (&evt).DecodePayload(&payload); err != nil {
		g.logger.Error("decode report generate payload", zap.Error(err))
		return nil // don't retry on bad payload
	}

	recordID, err := uuid.Parse(payload.RecordID)
	if err != nil {
		g.logger.Error("invalid record_id", zap.String("record_id", payload.RecordID))
		return nil
	}
	definitionID, err := uuid.Parse(payload.DefinitionID)
	if err != nil {
		g.logger.Error("invalid definition_id", zap.String("definition_id", payload.DefinitionID))
		return nil
	}

	g.logger.Info("generating report",
		zap.String("record_id", recordID.String()),
		zap.String("definition_id", definitionID.String()),
	)

	record, err := g.recordRepo.GetByID(ctx, recordID)
	if err != nil {
		return fmt.Errorf("get report record: %w", err)
	}

	def, err := g.defRepo.GetByID(ctx, definitionID)
	if err != nil {
		return fmt.Errorf("get report definition: %w", err)
	}

	// Collect data based on report type
	reportData, err := g.collectData(ctx, def, record.Period)
	if err != nil {
		g.logger.Error("collect report data failed", zap.Error(err))
		record.Status = RecordFailed
		_ = g.recordRepo.Update(ctx, record)
		return nil
	}

	// Build the full report JSON
	reportJSON := map[string]interface{}{
		"report_name":   record.ReportName,
		"report_type":   def.ReportType,
		"period":        record.Period,
		"generated_at":  time.Now().UTC().Format(time.RFC3339),
		"definition_id": def.ID.String(),
		"kpi_codes":     def.KPICodes,
		"device_groups": def.DeviceGroups,
		"data":          reportData,
	}

	jsonBytes, err := json.MarshalIndent(reportJSON, "", "  ")
	if err != nil {
		g.logger.Error("marshal report JSON", zap.Error(err))
		record.Status = RecordFailed
		_ = g.recordRepo.Update(ctx, record)
		return nil
	}

	// Upload to MinIO
	objectPath := fmt.Sprintf("reports/%s/%s/%s.json", def.ID.String(), record.Period, record.ID.String())
	reader := bytes.NewReader(jsonBytes)
	if g.storageAdmission != nil {
		decision, admissionErr := g.storageAdmission.CheckPath(ctx, storageprotection.ProtectedPathIDMinIO, storageprotection.WriteScopeReport)
		if admissionErr != nil {
			return fmt.Errorf("storage admission check: %w", admissionErr)
		}
		if !decision.Allowed {
			return fmt.Errorf("storage write protected: %s", decision.Reason)
		}
	}
	_, err = g.minioClient.PutObject(ctx, g.reportBkt, objectPath, reader, int64(len(jsonBytes)),
		minio.PutObjectOptions{ContentType: "application/json"})
	if err != nil {
		g.logger.Error("upload report to minio", zap.Error(err))
		record.Status = RecordFailed
		_ = g.recordRepo.Update(ctx, record)
		return nil
	}

	// Update record as ready
	record.MinioPath = objectPath
	record.FileSize = int64(len(jsonBytes))
	record.Status = RecordReady
	if err := g.recordRepo.Update(ctx, record); err != nil {
		return fmt.Errorf("update report record: %w", err)
	}

	g.logger.Info("report generated successfully",
		zap.String("record_id", recordID.String()),
		zap.String("minio_path", objectPath),
		zap.Int("file_size", len(jsonBytes)),
	)

	// Publish completion event
	if g.eventBus != nil {
		doneEvt, _ := event.NewEvent(event.SubjectReportGenerateDone, map[string]interface{}{
			"record_id":     recordID.String(),
			"definition_id": definitionID.String(),
			"minio_path":    objectPath,
		})
		_ = g.eventBus.Publish(ctx, event.SubjectReportGenerateDone, doneEvt)
	}

	return nil
}

// collectData gathers data from appropriate repositories based on report type.
func (g *ReportGenerator) collectData(ctx context.Context, def *ReportDefinition, period string) (interface{}, error) {
	switch def.ReportType {
	case ReportPerformance:
		return g.collectPerformanceData(ctx, def, period)
	case ReportAlarm:
		return g.collectAlarmData(ctx, def, period)
	default:
		// For device/capacity/security types, return summary placeholder
		return g.collectGenericData(ctx, def, period)
	}
}

func (g *ReportGenerator) collectPerformanceData(ctx context.Context, def *ReportDefinition, period string) (interface{}, error) {
	startTime, endTime := parsePeriodRange(period)

	filter := kpi.KPIFilter{
		StartTime: startTime,
		EndTime:   endTime,
		ListRequest: model.ListRequest{
			Page:     1,
			PageSize: 1000,
		},
	}

	result, err := g.kpiRepo.Query(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("query KPI data: %w", err)
	}

	return map[string]interface{}{
		"kpi_values":   result.Items,
		"total_count":  result.Total,
		"period_start": startTime.Format(time.RFC3339),
		"period_end":   endTime.Format(time.RFC3339),
	}, nil
}

func (g *ReportGenerator) collectAlarmData(ctx context.Context, def *ReportDefinition, period string) (interface{}, error) {
	startTime, endTime := parsePeriodRange(period)

	filter := alarm.AlarmFilter{
		StartTime: &startTime,
		EndTime:   &endTime,
		ListRequest: model.ListRequest{
			Page:     1,
			PageSize: 1000,
		},
	}

	history, err := g.alarmStore.ListHistory(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("query alarm history: %w", err)
	}

	stats, err := g.alarmStore.Statistics(ctx, filter)
	if err != nil {
		g.logger.Warn("alarm statistics query failed, continuing without stats", zap.Error(err))
		stats = &alarm.AlarmStatistics{}
	}

	return map[string]interface{}{
		"alarms":       history.Items,
		"total_count":  history.Total,
		"statistics":   stats,
		"period_start": startTime.Format(time.RFC3339),
		"period_end":   endTime.Format(time.RFC3339),
	}, nil
}

func (g *ReportGenerator) collectGenericData(ctx context.Context, def *ReportDefinition, period string) (interface{}, error) {
	startTime, endTime := parsePeriodRange(period)

	return map[string]interface{}{
		"report_type":  string(def.ReportType),
		"period_start": startTime.Format(time.RFC3339),
		"period_end":   endTime.Format(time.RFC3339),
		"description":  def.Description,
		"kpi_codes":    def.KPICodes,
	}, nil
}

// parsePeriodRange converts a period string like "2026-03-10" or "2026-03" to start/end times.
func parsePeriodRange(period string) (time.Time, time.Time) {
	// Try full date first: "2006-01-02"
	if t, err := time.Parse("2006-01-02", period); err == nil {
		return t, t.AddDate(0, 0, 1)
	}
	// Try month: "2006-01"
	if t, err := time.Parse("2006-01", period); err == nil {
		return t, t.AddDate(0, 1, 0)
	}
	// Try year: "2006"
	if t, err := time.Parse("2006", period); err == nil {
		return t, t.AddDate(1, 0, 0)
	}
	// Fallback: last 24 hours
	now := time.Now()
	return now.AddDate(0, 0, -1), now
}

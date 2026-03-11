package report

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/model"
)

// Service provides business logic for report management.
type Service struct {
	defRepo    DefinitionRepository
	recordRepo RecordRepository
	logger     *zap.Logger
}

// NewService creates a new report Service.
func NewService(defRepo DefinitionRepository, recordRepo RecordRepository, logger *zap.Logger) *Service {
	return &Service{
		defRepo:    defRepo,
		recordRepo: recordRepo,
		logger:     logger.Named("report"),
	}
}

// ---- Definition operations ----

// CreateDefinition creates a new report definition.
func (s *Service) CreateDefinition(ctx context.Context, def *ReportDefinition) (*ReportDefinition, error) {
	if def.Status == "" {
		def.Status = ReportDraft
	}
	if def.Format == nil {
		def.Format = []string{"pdf"}
	}
	if def.KPICodes == nil {
		def.KPICodes = []string{}
	}
	if def.DeviceGroups == nil {
		def.DeviceGroups = []string{}
	}

	if err := s.defRepo.Create(ctx, def); err != nil {
		return nil, fmt.Errorf("create report definition: %w", err)
	}

	s.logger.Info("report definition created",
		zap.String("id", def.ID.String()),
		zap.String("name", def.ReportName),
		zap.String("type", string(def.ReportType)),
	)

	return def, nil
}

// GetDefinition retrieves a report definition by ID.
func (s *Service) GetDefinition(ctx context.Context, id uuid.UUID) (*ReportDefinition, error) {
	return s.defRepo.GetByID(ctx, id)
}

// UpdateDefinition updates an existing report definition.
func (s *Service) UpdateDefinition(ctx context.Context, id uuid.UUID, def *ReportDefinition) (*ReportDefinition, error) {
	existing, err := s.defRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get report definition: %w", err)
	}

	existing.ReportName = def.ReportName
	existing.ReportType = def.ReportType
	existing.Description = def.Description
	existing.Format = def.Format
	existing.Period = def.Period
	existing.KPICodes = def.KPICodes
	existing.DeviceGroups = def.DeviceGroups
	existing.AutoGenerate = def.AutoGenerate
	existing.CronExpression = def.CronExpression
	existing.Status = def.Status
	existing.Creator = def.Creator

	if err := s.defRepo.Update(ctx, existing); err != nil {
		return nil, fmt.Errorf("update report definition: %w", err)
	}

	s.logger.Info("report definition updated",
		zap.String("id", id.String()),
	)

	return existing, nil
}

// DeleteDefinition deletes a report definition by ID.
func (s *Service) DeleteDefinition(ctx context.Context, id uuid.UUID) error {
	return s.defRepo.Delete(ctx, id)
}

// ListDefinitions returns a paginated list of report definitions.
func (s *Service) ListDefinitions(ctx context.Context, filter DefinitionFilter) (*model.ListResponse[ReportDefinition], error) {
	return s.defRepo.List(ctx, filter)
}

// ---- Record operations ----

// ListRecords returns a paginated list of report records.
func (s *Service) ListRecords(ctx context.Context, filter RecordFilter) (*model.ListResponse[ReportRecord], error) {
	return s.recordRepo.List(ctx, filter)
}

// GetRecord retrieves a report record by ID.
func (s *Service) GetRecord(ctx context.Context, id uuid.UUID) (*ReportRecord, error) {
	return s.recordRepo.GetByID(ctx, id)
}

// Generate creates a new report record with status=generating and returns immediately.
// Actual report generation would be handled asynchronously by a worker.
func (s *Service) Generate(ctx context.Context, definitionID uuid.UUID, period string) (*ReportRecord, error) {
	def, err := s.defRepo.GetByID(ctx, definitionID)
	if err != nil {
		return nil, fmt.Errorf("get report definition: %w", err)
	}

	now := time.Now()
	record := &ReportRecord{
		ReportDefinitionID: definitionID,
		ReportName:         fmt.Sprintf("%s-%s", def.ReportName, period),
		Period:             period,
		GenerateTime:       now,
		Format:             "pdf",
		Status:             RecordGenerating,
	}
	if len(def.Format) > 0 {
		record.Format = def.Format[0]
	}

	if err := s.recordRepo.Create(ctx, record); err != nil {
		return nil, fmt.Errorf("create report record: %w", err)
	}

	// Update last_gen_time on definition
	def.LastGenTime = &now
	_ = s.defRepo.Update(ctx, def)

	s.logger.Info("report generation triggered",
		zap.String("record_id", record.ID.String()),
		zap.String("definition_id", definitionID.String()),
		zap.String("period", period),
	)

	return record, nil
}

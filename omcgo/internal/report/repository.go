package report

import (
	"context"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/model"
)

// DefinitionRepository provides persistence for report definitions.
type DefinitionRepository interface {
	Create(ctx context.Context, def *ReportDefinition) error
	GetByID(ctx context.Context, id uuid.UUID) (*ReportDefinition, error)
	Update(ctx context.Context, def *ReportDefinition) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, filter DefinitionFilter) (*model.ListResponse[ReportDefinition], error)
}

// RecordRepository provides persistence for report records.
type RecordRepository interface {
	Create(ctx context.Context, record *ReportRecord) error
	GetByID(ctx context.Context, id uuid.UUID) (*ReportRecord, error)
	Update(ctx context.Context, record *ReportRecord) error
	List(ctx context.Context, filter RecordFilter) (*model.ListResponse[ReportRecord], error)
}

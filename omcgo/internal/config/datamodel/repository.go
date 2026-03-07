package datamodel

import (
	"context"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/common/model"
)

// DataModelRepository defines the persistence interface for data model definitions.
type DataModelRepository interface {
	Create(ctx context.Context, dm *DataModel) error
	GetByID(ctx context.Context, id uuid.UUID) (*DataModel, error)
	Update(ctx context.Context, dm *DataModel) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, filter DataModelFilter) (*model.ListResponse[DataModel], error)
	FindActive(ctx context.Context, carrier model.CarrierCode, tech model.Technology,
		oui, productClass string, scope model.DataModelScope) (*DataModel, error)
	Activate(ctx context.Context, id uuid.UUID) error
	Deprecate(ctx context.Context, id uuid.UUID) error
	Statistics(ctx context.Context) (*DataModelStats, error)
}

// ImportLogRepository defines the persistence interface for import audit logs.
type ImportLogRepository interface {
	Create(ctx context.Context, entry *ImportLogEntry) error
	ListByModel(ctx context.Context, modelID uuid.UUID) ([]ImportLogEntry, error)
}

// OUIRepository defines the persistence interface for the OUI registry.
type OUIRepository interface {
	GetByOUI(ctx context.Context, oui string) (*OUIEntry, error)
	List(ctx context.Context) ([]OUIEntry, error)
	Create(ctx context.Context, entry *OUIEntry) error
}

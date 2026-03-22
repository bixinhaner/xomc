package datamodel

import (
	"context"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
)

// DataModelReader provides read-only access to data model definitions.
type DataModelReader interface {
	GetByID(ctx context.Context, id uuid.UUID) (*DataModel, error)
	List(ctx context.Context, filter DataModelFilter) (*model.ListResponse[DataModel], error)
	FindActive(ctx context.Context, carrier model.CarrierCode, tech model.Technology,
		oui, productClass string, scope model.DataModelScope) (*DataModel, error)
	Statistics(ctx context.Context) (*DataModelStats, error)
}

// DataModelWriter provides write and lifecycle operations for data model definitions.
type DataModelWriter interface {
	Create(ctx context.Context, dm *DataModel) error
	Update(ctx context.Context, dm *DataModel) error
	Delete(ctx context.Context, id uuid.UUID) error
	Activate(ctx context.Context, id uuid.UUID) error
	Deprecate(ctx context.Context, id uuid.UUID) error
}

// DataModelRepository defines the full persistence interface for data model definitions.
// It composes smaller interfaces for backward compatibility.
type DataModelRepository interface {
	DataModelReader
	DataModelWriter
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

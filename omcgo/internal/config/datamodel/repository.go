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
	FindActiveWithFirmware(ctx context.Context, carrier model.CarrierCode, tech model.Technology,
		oui, productClass, firmwareVersion string, scope model.DataModelScope) (*DataModel, error)
	// FindActiveForMatch performs two-level precise matching for parameter templates:
	//   Level 1: OUI + ProductClass + FirmwareVersion (exact)
	//   Level 2: OUI + ProductClass (firmware_version IS NULL or empty, manual > auto)
	// Returns nil, nil if no matching template is found.
	FindActiveForMatch(ctx context.Context, carrier model.CarrierCode, tech model.Technology,
		oui, productClass, firmwareVersion string) (*DataModel, error)
	Statistics(ctx context.Context) (*DataModelStats, error)
}

// DataModelWriter provides write and lifecycle operations for data model definitions.
type DataModelWriter interface {
	Create(ctx context.Context, dm *DataModel) error
	Update(ctx context.Context, dm *DataModel) error
	Delete(ctx context.Context, id uuid.UUID) error
	Activate(ctx context.Context, id uuid.UUID) error
	Deprecate(ctx context.Context, id uuid.UUID) error
	// TouchLastAccessed updates last_accessed_at to NOW() for the given data model.
	TouchLastAccessed(ctx context.Context, id uuid.UUID) error
	// DeleteExpired removes data models that have not been accessed within the
	// configured idle period. autoMaxAge is the max idle days for auto_discovered
	// templates (e.g. 15), manualMaxAge for manual templates (e.g. 60).
	DeleteExpired(ctx context.Context, autoMaxAge, manualMaxAge int) (int64, error)
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

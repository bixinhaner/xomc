package kpi

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/common/model"
)

// KPIFilter defines query parameters for KPI value retrieval.
type KPIFilter struct {
	DeviceID   *uuid.UUID
	CellID     *string
	KPIName    *string
	Carrier    *model.CarrierCode
	Technology *model.Technology
	StartTime  time.Time
	EndTime    time.Time
	model.ListRequest
}

// KPIRepository defines the interface for KPI value persistence.
type KPIRepository interface {
	BatchInsert(ctx context.Context, values []model.KPIValue) error
	Query(ctx context.Context, filter KPIFilter) (*model.ListResponse[model.KPIValue], error)
	ListDefinitions(ctx context.Context, carrier *model.CarrierCode, tech *model.Technology) ([]model.KPIDefinition, error)
	SyncDefinitions(ctx context.Context, defs []model.KPIDefinition) error
}

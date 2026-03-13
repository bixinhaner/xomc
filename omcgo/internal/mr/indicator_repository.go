package mr

import (
	"context"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
)

// IndicatorRepository provides persistence for MR indicators.
type IndicatorRepository interface {
	List(ctx context.Context, filter IndicatorFilter) (*model.ListResponse[MRIndicator], error)
	ListAll(ctx context.Context) ([]MRIndicator, error)
	GetByCode(ctx context.Context, code string) (*MRIndicator, error)
}

// MappingRepository provides persistence for MR device mappings.
type MappingRepository interface {
	List(ctx context.Context, filter MappingFilter) (*model.ListResponse[MRDeviceMapping], error)
	Update(ctx context.Context, mapping *MRDeviceMapping) error
	ToggleEnabled(ctx context.Context, id uuid.UUID, enabled bool) (*MRDeviceMapping, error)
}

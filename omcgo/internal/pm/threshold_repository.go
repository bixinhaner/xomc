package pm

import (
	"context"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
)

// ThresholdRepository defines the interface for KPI threshold persistence.
type ThresholdRepository interface {
	Create(ctx context.Context, threshold *KPIThreshold) error
	GetByID(ctx context.Context, id uuid.UUID) (*KPIThreshold, error)
	Update(ctx context.Context, threshold *KPIThreshold) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, filter KPIThresholdFilter) (*model.ListResponse[KPIThreshold], error)
}

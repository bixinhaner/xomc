package baseline

import (
	"context"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/common/model"
)

// BaselineRepository provides persistence for baseline configs.
type BaselineRepository interface {
	Create(ctx context.Context, baseline *BaselineConfig) error
	GetByID(ctx context.Context, id uuid.UUID) (*BaselineConfig, error)
	Update(ctx context.Context, baseline *BaselineConfig) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, filter BaselineFilter) (*model.ListResponse[BaselineConfig], error)
}

// ConfigTaskRepository provides persistence for config tasks.
type ConfigTaskRepository interface {
	Create(ctx context.Context, task *ConfigTask) error
	List(ctx context.Context, filter ConfigTaskFilter) (*model.ListResponse[ConfigTask], error)
}

// NeighborRepository provides persistence for neighbor params.
type NeighborRepository interface {
	List(ctx context.Context, filter NeighborFilter) (*model.ListResponse[NeighborParam], error)
}

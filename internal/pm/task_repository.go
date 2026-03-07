package pm

import (
	"context"

	"github.com/omcgo/omcgo/internal/common/model"
)

// TaskRepository defines the interface for PM task persistence.
type TaskRepository interface {
	Create(ctx context.Context, task *PerformanceTask) error
	List(ctx context.Context, filter TaskFilter) (*model.ListResponse[PerformanceTask], error)
}

package ops

import (
	"context"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/common/model"
)

// TemplateRepository provides persistence for ops templates.
type TemplateRepository interface {
	Create(ctx context.Context, template *OpsTemplate) error
	GetByID(ctx context.Context, id uuid.UUID) (*OpsTemplate, error)
	Update(ctx context.Context, template *OpsTemplate) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, filter TemplateFilter) (*model.ListResponse[OpsTemplate], error)
	IncrementUseCount(ctx context.Context, id uuid.UUID) error
}

// TaskRepository provides persistence for ops tasks.
type TaskRepository interface {
	Create(ctx context.Context, task *OpsTask) error
	GetByID(ctx context.Context, id uuid.UUID) (*OpsTask, error)
	UpdateStatus(ctx context.Context, task *OpsTask) error
	List(ctx context.Context, filter TaskFilter) (*model.ListResponse[OpsTask], error)
}

// CommandRecordRepository provides persistence for ops command records.
type CommandRecordRepository interface {
	Create(ctx context.Context, record *OpsCommandRecord) error
	List(ctx context.Context, filter CommandRecordFilter) (*model.ListResponse[OpsCommandRecord], error)
}

package ops

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
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
	// UpdateApproval 持久化任务审批结果 + 联动执行状态转移（T-0101-d）。
	UpdateApproval(ctx context.Context, taskID, approverID uuid.UUID, approve bool, decidedAt time.Time) error
}

// CommandRecordRepository provides persistence for ops command records.
type CommandRecordRepository interface {
	Create(ctx context.Context, record *OpsCommandRecord) error
	List(ctx context.Context, filter CommandRecordFilter) (*model.ListResponse[OpsCommandRecord], error)
}

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
	// TransitionStatus 原子状态转换（T-0101-g 状态机原子转换）。
	// 单 SQL UPDATE 含 `WHERE status = ANY(validFrom)` CAS guard，并发 race-safe。
	// 若当前 status 不在 validFrom 集合中（或 task 不存在），返 ErrInvalidStateTransition。
	TransitionStatus(ctx context.Context, taskID uuid.UUID, validFrom []OpsTaskStatus, to OpsTaskStatus, setStartedAt, setCompletedAt bool) error
}

// CommandRecordRepository provides persistence for ops command records.
type CommandRecordRepository interface {
	Create(ctx context.Context, record *OpsCommandRecord) error
	List(ctx context.Context, filter CommandRecordFilter) (*model.ListResponse[OpsCommandRecord], error)
}

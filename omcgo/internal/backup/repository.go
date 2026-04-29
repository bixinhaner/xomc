package backup

import (
	"context"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
)

// TaskRepository provides persistence for backup tasks.
type TaskRepository interface {
	Create(ctx context.Context, task *BackupTask) error
	GetByID(ctx context.Context, id uuid.UUID) (*BackupTask, error)
	Update(ctx context.Context, task *BackupTask) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, filter TaskFilter) (*model.ListResponse[BackupTask], error)
}

// ScheduleRepository provides persistence for backup schedules.
type ScheduleRepository interface {
	Create(ctx context.Context, schedule *BackupSchedule) error
	GetByID(ctx context.Context, id uuid.UUID) (*BackupSchedule, error)
	Update(ctx context.Context, schedule *BackupSchedule) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, filter ScheduleFilter) (*model.ListResponse[BackupSchedule], error)
}

// PolicyRepository provides singleton persistence for the backup policy
// (T-0071). Application layer enforces the singleton: Get returns the latest
// row (ORDER BY updated_at DESC LIMIT 1) or nil if the table is empty; Upsert
// inserts on first save and updates the existing row thereafter.
type PolicyRepository interface {
	Get(ctx context.Context) (*BackupPolicy, error)
	Upsert(ctx context.Context, policy *BackupPolicy) error
}

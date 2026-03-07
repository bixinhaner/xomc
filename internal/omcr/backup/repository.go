package backup

import (
	"context"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/common/model"
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

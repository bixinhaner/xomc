package backup

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
)

// ErrFilePathAlreadySet signals that UpdateFilePath skipped the row because
// file_path was already populated. Used by the FilePathRecorder (T-0079) to
// differentiate first-write-wins skips from genuine errors. Atomic at the DB
// layer (CAS via WHERE file_path IS NULL) — TOCTOU-free under multi-device
// concurrent uploads.
var ErrFilePathAlreadySet = errors.New("backup_task.file_path already set")

// TaskRepository provides persistence for backup tasks.
type TaskRepository interface {
	Create(ctx context.Context, task *BackupTask) error
	GetByID(ctx context.Context, id uuid.UUID) (*BackupTask, error)
	Update(ctx context.Context, task *BackupTask) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, filter TaskFilter) (*model.ListResponse[BackupTask], error)

	// CleanupOldRows deletes terminal-status rows older than `cutoff` while
	// preserving the most-recent `keepLastN` rows per target_type partition.
	// Returns the number of rows actually deleted. (T-0073 cleanup cron.)
	CleanupOldRows(ctx context.Context, cutoff time.Time, keepLastN int) (int64, error)

	// UpdateFilePath sets backup_tasks.file_path on the given row using an
	// atomic CAS — only writes when the column is currently NULL. Used by
	// FilePathRecorder (T-0079) for first-write-wins semantics under
	// concurrent multi-device uploads. Returns ErrFilePathAlreadySet when
	// another writer won the race; ErrNotFound when the id does not exist.
	UpdateFilePath(ctx context.Context, id uuid.UUID, filePath string) error

	// FindByIDPrefix returns up to `limit` backup_tasks whose UUID (textual
	// form, dashes stripped) starts with the given hex prefix. Used by the
	// FilePathRecorder to map upload filename → backup_task.
	// Empty prefix returns no rows. limit < 1 is treated as 1.
	FindByIDPrefix(ctx context.Context, prefix string, limit int) ([]*BackupTask, error)
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

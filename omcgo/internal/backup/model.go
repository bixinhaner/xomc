package backup

import (
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/model"
)

// TaskStatus represents the current state of a backup task.
type TaskStatus string

const (
	TaskPending   TaskStatus = "pending"
	TaskRunning   TaskStatus = "running"
	TaskCompleted TaskStatus = "completed"
	TaskFailed    TaskStatus = "failed"
	TaskCancelled TaskStatus = "cancelled"
)

// TaskType represents the type of backup to perform.
type TaskType string

const (
	TaskFull        TaskType = "full"
	TaskIncremental TaskType = "incremental"
	TaskConfigOnly  TaskType = "config_only"
)

// BackupTask represents a single backup operation.
type BackupTask struct {
	ID           uuid.UUID  `json:"id"`
	TaskType     TaskType   `json:"task_type"`
	TargetType   string     `json:"target_type"`             // "device" or "group"
	TargetIDs    []string   `json:"target_ids"`
	Status       TaskStatus `json:"status"`
	Progress     int        `json:"progress"`                // 0-100
	FilePath     *string    `json:"file_path,omitempty"`
	ErrorMessage *string    `json:"error_message,omitempty"`
	StartedAt    *time.Time `json:"started_at,omitempty"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// BackupSchedule represents a scheduled recurring backup.
type BackupSchedule struct {
	ID         uuid.UUID `json:"id"`
	Name       string    `json:"name"`
	CronExpr   string    `json:"cron_expr"`
	Enabled    bool      `json:"enabled"`
	TaskType   TaskType  `json:"task_type"`
	TargetType *string   `json:"target_type,omitempty"`
	TargetIDs  []string  `json:"target_ids,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// TaskFilter specifies criteria for listing backup tasks.
type TaskFilter struct {
	Status   *TaskStatus
	TaskType *TaskType
	model.ListRequest
}

// ScheduleFilter specifies criteria for listing backup schedules.
type ScheduleFilter struct {
	Enabled *bool
	model.ListRequest
}

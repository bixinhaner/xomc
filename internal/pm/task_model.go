package pm

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/common/model"
)

// TaskStatus represents the current state of a PM task.
type TaskStatus string

const (
	PMTaskPending   TaskStatus = "pending"
	PMTaskRunning   TaskStatus = "running"
	PMTaskCompleted TaskStatus = "completed"
	PMTaskFailed    TaskStatus = "failed"
	PMTaskCancelled TaskStatus = "cancelled"
)

// PMTaskType represents the type of PM task.
type PMTaskType string

const (
	PMTaskExtraction     PMTaskType = "extraction"
	PMTaskReport         PMTaskType = "report"
	PMTaskThresholdCheck PMTaskType = "threshold-check"
)

// PerformanceTask represents a single PM task.
type PerformanceTask struct {
	ID          uuid.UUID       `json:"id"`
	TaskName    string          `json:"task_name"`
	TaskType    PMTaskType      `json:"task_type"`
	DeviceSNs   json.RawMessage `json:"device_sns"`
	KPICodes    json.RawMessage `json:"kpi_codes"`
	Granularity string          `json:"granularity"`
	TimeRange   json.RawMessage `json:"time_range"`
	Status      TaskStatus      `json:"status"`
	Progress    int             `json:"progress"`
	Creator     string          `json:"creator"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

// TaskFilter specifies criteria for listing PM tasks.
type TaskFilter struct {
	Status   *TaskStatus
	TaskType *PMTaskType
	model.ListRequest
}

// CreatePerformanceTaskRequest is the JSON body for creating a PM task.
type CreatePerformanceTaskRequest struct {
	TaskName    string          `json:"task_name" binding:"required"`
	TaskType    PMTaskType      `json:"task_type"`
	DeviceSNs   json.RawMessage `json:"device_sns"`
	KPICodes    json.RawMessage `json:"kpi_codes"`
	Granularity string          `json:"granularity"`
	TimeRange   json.RawMessage `json:"time_range"`
	Creator     string          `json:"creator"`
}

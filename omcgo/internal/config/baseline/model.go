package baseline

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
)

// BaselineStatus represents the lifecycle state of a baseline config.
type BaselineStatus string

const (
	BaselineDraft      BaselineStatus = "draft"
	BaselineActive     BaselineStatus = "active"
	BaselineDeprecated BaselineStatus = "deprecated"
)

// ConfigTaskStatus represents the execution state of a config task.
type ConfigTaskStatus string

const (
	ConfigTaskPending   ConfigTaskStatus = "pending"
	ConfigTaskRunning   ConfigTaskStatus = "running"
	ConfigTaskCompleted ConfigTaskStatus = "completed"
	ConfigTaskFailed    ConfigTaskStatus = "failed"
	ConfigTaskCancelled ConfigTaskStatus = "cancelled"
)

// ConfigTaskType represents the type of config task.
type ConfigTaskType string

const (
	ConfigTaskParamSync      ConfigTaskType = "param-sync"
	ConfigTaskBatchConfig    ConfigTaskType = "batch-config"
	ConfigTaskBaselineApply  ConfigTaskType = "baseline-apply"
	ConfigTaskNeighborUpdate ConfigTaskType = "neighbor-update"
)

// NeighborType represents the type of neighbor relationship.
type NeighborType string

const (
	NeighborIntraFreq NeighborType = "intra-freq"
	NeighborInterFreq NeighborType = "inter-freq"
	NeighborInterRAT  NeighborType = "inter-rat"
)

// BaselineConfig represents a baseline configuration entry.
type BaselineConfig struct {
	ID           uuid.UUID       `json:"id"`
	BaselineName string          `json:"baseline_name"`
	Description  *string         `json:"description,omitempty"`
	DeviceType   *string         `json:"device_type,omitempty"`
	Version      *string         `json:"version,omitempty"`
	Params       json.RawMessage `json:"params"`
	Creator      *string         `json:"creator,omitempty"`
	Status       BaselineStatus  `json:"status"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

// ConfigTask represents a configuration task.
type ConfigTask struct {
	ID           uuid.UUID        `json:"id"`
	TaskName     string           `json:"task_name"`
	TaskType     ConfigTaskType   `json:"task_type"`
	DeviceSns    json.RawMessage  `json:"device_sns"`
	TemplateID   *uuid.UUID       `json:"template_id,omitempty"`
	BaselineID   *uuid.UUID       `json:"baseline_id,omitempty"`
	Params       json.RawMessage  `json:"params,omitempty"`
	Status       ConfigTaskStatus `json:"status"`
	Progress     int              `json:"progress"`
	SuccessCount int              `json:"success_count"`
	FailCount    int              `json:"fail_count"`
	TotalCount   int              `json:"total_count"`
	Creator      *string          `json:"creator,omitempty"`
	Message      *string          `json:"message,omitempty"`
	CreatedAt    time.Time        `json:"created_at"`
	UpdatedAt    time.Time        `json:"updated_at"`
}

// NeighborParam represents a neighbor relationship between cells.
type NeighborParam struct {
	ID             uuid.UUID       `json:"id"`
	SourceCellID   string          `json:"source_cell_id"`
	SourceCellName *string         `json:"source_cell_name,omitempty"`
	TargetCellID   string          `json:"target_cell_id"`
	TargetCellName *string         `json:"target_cell_name,omitempty"`
	NeighborType   NeighborType    `json:"neighbor_type"`
	Params         json.RawMessage `json:"params"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

// BaselineFilter specifies criteria for listing baseline configs.
type BaselineFilter struct {
	DeviceType *string
	Status     *BaselineStatus
	model.ListRequest
}

// ConfigTaskFilter specifies criteria for listing config tasks.
type ConfigTaskFilter struct {
	Status *ConfigTaskStatus
	model.ListRequest
}

// NeighborFilter specifies criteria for listing neighbor params.
type NeighborFilter struct {
	SourceCellID *string
	model.ListRequest
}

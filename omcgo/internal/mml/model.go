package mml

import (
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
)

// TaskStatus represents the current state of an MML task.
type TaskStatus string

const (
	TaskPending   TaskStatus = "pending"
	TaskRunning   TaskStatus = "running"
	TaskCompleted TaskStatus = "completed"
	TaskFailed    TaskStatus = "failed"
)

// MMLCommand represents a predefined MML command template.
type MMLCommand struct {
	ID            uuid.UUID              `json:"id"`
	CommandName   string                 `json:"command_name"`
	CommandCode   string                 `json:"command_code"`
	Category      string                 `json:"category"`
	Description   string                 `json:"description"`
	RPCMethod     string                 `json:"rpc_method"`
	ParamTemplate map[string]interface{} `json:"param_template"`
	ProductTypes  []string               `json:"product_types"`
	CreatedAt     time.Time              `json:"created_at"`
}

// MMLScript represents a user-defined MML command script.
type MMLScript struct {
	ID          uuid.UUID `json:"id"`
	ScriptName  string    `json:"script_name"`
	Description string    `json:"description"`
	Content     string    `json:"content"`
	DeviceType  string    `json:"device_type"`
	Creator     string    `json:"creator"`
	Tags        []string  `json:"tags"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// MMLTask represents an MML command execution task.
type MMLTask struct {
	ID        uuid.UUID                `json:"id"`
	TaskName  string                   `json:"task_name"`
	ScriptID  *uuid.UUID               `json:"script_id,omitempty"`
	DeviceSNs []string                 `json:"device_sns"`
	Commands  []map[string]interface{} `json:"commands"`
	Status    TaskStatus               `json:"status"`
	Results   []map[string]interface{} `json:"results"`
	Creator   string                   `json:"creator"`
	CreatedAt time.Time                `json:"created_at"`
	UpdatedAt time.Time                `json:"updated_at"`
}

// CommandFilter specifies criteria for listing MML commands.
type CommandFilter struct {
	Category *string
	Search   *string
	model.ListRequest
}

// ScriptFilter specifies criteria for listing MML scripts.
type ScriptFilter struct {
	DeviceType *string
	Creator    *string
	Search     *string
	model.ListRequest
}

// TaskFilter specifies criteria for listing MML tasks.
type TaskFilter struct {
	Status *TaskStatus
	model.ListRequest
}

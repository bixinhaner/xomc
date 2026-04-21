package mml

import (
	"encoding/json"
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
	TaskPaused    TaskStatus = "paused"
	TaskCancelled TaskStatus = "cancelled"
)

// ExecuteType defines how a task is scheduled for execution.
type ExecuteType string

const (
	ExecuteImmediate ExecuteType = "immediate"
	ExecuteScheduled ExecuteType = "scheduled"
	ExecutePeriodic  ExecuteType = "periodic"
	ExecuteSuspended ExecuteType = "suspended"
)

// TaskResult represents the outcome of a completed task.
type TaskResult string

const (
	ResultSuccess TaskResult = "success"
	ResultPartial TaskResult = "partial"
	ResultFailed  TaskResult = "failed"
)

// MMLCommand represents a predefined MML command template.
type MMLCommand struct {
	ID                  uuid.UUID              `json:"id"`
	CommandName         string                 `json:"command_name"`
	CommandCode         string                 `json:"command_code"`
	Category            string                 `json:"category"`
	Description         string                 `json:"description"`
	RPCMethod           string                 `json:"rpc_method"`
	OperationType       string                 `json:"operation_type" db:"operation_type"`
	ParamTemplate       map[string]interface{} `json:"param_template"`
	ParamPaths          json.RawMessage        `json:"param_paths" db:"param_paths"`
	SupportedOperations []string               `json:"supported_operations" db:"supported_operations"`
	HelpDoc             string                 `json:"help_doc" db:"help_doc"`
	Notes               string                 `json:"notes" db:"notes"`
	ProductTypes        []string               `json:"product_types"`
	CreatedAt           time.Time              `json:"created_at"`
	SubCommands         []SubCommand           `json:"sub_commands,omitempty"`
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
	Executor  string                   `json:"executor,omitempty"`
	CreatedAt time.Time                `json:"created_at"`
	UpdatedAt time.Time                `json:"updated_at"`

	// Scheduling
	ExecuteType ExecuteType  `json:"execute_type"`
	ScheduledAt *time.Time   `json:"scheduled_at,omitempty"`
	PeriodStart *time.Time   `json:"period_start,omitempty"`
	PeriodEnd   *time.Time   `json:"period_end,omitempty"`
	PeriodTime  string       `json:"period_time,omitempty"`

	// Retry strategy
	OfflineRetry      bool `json:"offline_retry"`
	OfflineRetryWait  int  `json:"offline_retry_wait"`
	FailedRetry       bool `json:"failed_retry"`
	FailedRetryCount  int  `json:"failed_retry_count"`
	FailedRetryInterval int `json:"failed_retry_interval"`

	// Execution timestamps
	StartedAt  *time.Time `json:"started_at,omitempty"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`

	// Statistics
	TotalDevices int       `json:"total_devices"`
	SuccessCount int       `json:"success_count"`
	FailedCount  int       `json:"failed_count"`
	Result       *TaskResult `json:"result,omitempty"`
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
	Status      *TaskStatus
	ExecuteType *ExecuteType
	Result      *TaskResult
	model.ListRequest
}

// MMLTemplate represents a user-defined command parameter template.
type MMLTemplate struct {
	ID            uuid.UUID              `json:"id"`
	TemplateName  string                 `json:"template_name"`
	CommandCode   string                 `json:"command_code"`
	OperationType string                 `json:"operation_type"`
	TemplateScope string                 `json:"template_scope"` // private or public
	CategoryGroup string                 `json:"category_group,omitempty"`
	Parameters    map[string]interface{} `json:"parameters"`
	ParamPaths    []string               `json:"param_paths"`
	Description   string                 `json:"description"`
	ProductTypes  []string               `json:"product_types"`
	Creator       string                 `json:"creator"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
}

// TemplateFilter specifies criteria for listing MML templates.
type TemplateFilter struct {
	CommandCode   *string
	OperationType *string
	TemplateScope *string
	CategoryGroup *string
	Creator       *string
	model.ListRequest
}

// SubCommand represents a sub-command bound to an MML command (N:M relationship).
type SubCommand struct {
	ID          uuid.UUID       `json:"id"`
	Name        string          `json:"name"`
	Code        string          `json:"code"`
	Tr069Path   string          `json:"tr069_path"`
	Description string          `json:"description"`
	ValueType   string          `json:"value_type"`
	IsWritable  bool            `json:"is_writable"`
	Options     []SubCmdOption  `json:"options"`
	Unit        string          `json:"unit,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
}

// SubCmdOption represents an option for enum-type sub-commands.
type SubCmdOption struct {
	Label string      `json:"label"`
	Value interface{} `json:"value"`
}

// MMLAuditLog records a single command execution for compliance auditing.
type MMLAuditLog struct {
	ID            uuid.UUID              `json:"id"`
	TaskID        *uuid.UUID             `json:"task_id,omitempty"`
	CommandCode   string                 `json:"command_code"`
	OperationType string                 `json:"operation_type"`
	DeviceSN      string                 `json:"device_sn"`
	Parameters    map[string]interface{} `json:"parameters"`
	ParamPaths    []string               `json:"param_paths"`
	ResultStatus  string                 `json:"result_status"`
	ResultMessage string                 `json:"result_message,omitempty"`
	Creator       string                 `json:"creator"`
	ExecutedAt    time.Time              `json:"executed_at"`
	DurationMs    *int                   `json:"duration_ms,omitempty"`
}

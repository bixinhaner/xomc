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
	Params              []MMLParamRef          `json:"params,omitempty"`
}

// ScriptStatus represents the current state of an MML script.
//
// Two families of values:
//   - 定义态: active / archived（脚本本身是否启用）
//   - 执行态: pending / running / paused / completed / failed / cancelled
//     （脚本最近一次执行的生命周期阶段，与 TaskStatus 对齐）
// mml_scripts.status 列无 CHECK 约束，允许跨族流转。
type ScriptStatus string

const (
	ScriptActive    ScriptStatus = "active"
	ScriptArchived  ScriptStatus = "archived"
	ScriptPending   ScriptStatus = "pending"
	ScriptRunning   ScriptStatus = "running"
	ScriptPaused    ScriptStatus = "paused"
	ScriptCompleted ScriptStatus = "completed"
	ScriptFailed    ScriptStatus = "failed"
	ScriptCancelled ScriptStatus = "cancelled"
)

// ScriptType defines how a script is categorized.
type ScriptType string

const (
	ScriptTypeManual ScriptType = "manual"
	ScriptTypeBatch  ScriptType = "batch"
)

// MMLScript represents a user-defined MML command script.
type MMLScript struct {
	ID          uuid.UUID    `json:"id"`
	ScriptName  string       `json:"script_name"`
	Description string       `json:"description"`
	Content     string       `json:"content"`
	Creator     string       `json:"creator"`
	Tags        []string     `json:"tags"`
	Status      ScriptStatus `json:"status"`
	StartTime   *time.Time   `json:"start_time,omitempty"`
	EndTime     *time.Time   `json:"end_time,omitempty"`
	Type        ScriptType   `json:"type"`
	Progress    float64      `json:"progress"`
	Result      JSONMap      `json:"result,omitempty"`
	// 最近一次执行终态（由 MMLAggregator 在对应 mml_task 收敛后回写）。
	// per-execution 详情查 mml_tasks，脚本层只持有"最近一次"指针。
	LastRunStatus *string    `json:"last_run_status,omitempty"`
	LastRunAt     *time.Time `json:"last_run_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// JSONMap is a helper type for nullable JSONB map fields.
type JSONMap map[string]interface{}

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
	TotalDevices int         `json:"total_devices"`
	SuccessCount int         `json:"success_count"`
	FailedCount  int         `json:"failed_count"`
	Result       *TaskResult `json:"result,omitempty"`

	// Scheduler 调度字段（P2/P3，docs/design/mml-task-flow-design-20260424.md）
	// NextTriggerAt: 下次触发时刻；PeriodicParentID: periodic 子实例指向模板。
	NextTriggerAt    *time.Time `json:"next_trigger_at,omitempty"`
	PeriodicParentID *uuid.UUID `json:"parent_task_id,omitempty"`
}

// CommandFilter specifies criteria for listing MML commands.
type CommandFilter struct {
	Category *string
	Search   *string
	model.ListRequest
}

// ScriptFilter specifies criteria for listing MML scripts.
type ScriptFilter struct {
	Creator *string
	Search  *string
	model.ListRequest
}

// TaskFilter specifies criteria for listing MML tasks.
type TaskFilter struct {
	Status      *TaskStatus
	ExecuteType *ExecuteType
	Result      *TaskResult
	TaskName    *string // case-insensitive substring match on task_name
	model.ListRequest
}

// MMLCustomCommand represents a user-defined custom command (renamed from MMLTemplate).
type MMLCustomCommand struct {
	ID            uuid.UUID              `json:"id"`
	CommandName   string                 `json:"command_name"`
	CommandCode   string                 `json:"command_code"`
	OperationType string                 `json:"operation_type"`
	CommandScope  string                 `json:"command_scope"` // private or public
	CategoryGroup string                 `json:"category_group,omitempty"`
	Parameters    map[string]interface{} `json:"parameters"`
	ParamPaths    []string               `json:"param_paths"`
	Description   string                 `json:"description"`
	Creator       string                 `json:"creator"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
}

// CustomCommandFilter specifies criteria for listing MML custom commands.
//
// 私有命令可见性（T-0090-c）：
//   - public 命令：始终可见
//   - private 命令：仅在以下任一条件满足时可见
//       (a) creator == Creator （self fallback，防止脱离 group 后看不见自己创建的）
//       (b) creator's RBAC group(s) 与 VisibleGroupIDs 交集非空 （group-share）
//   - 若 Creator 与 VisibleGroupIDs 均为空 → 默认 deny private（仅 public 可见）
//
// UserID 由 service 层接收后调 RoleQuerier 派生 VisibleGroupIDs；repo 层仅消费派生结果。
type CustomCommandFilter struct {
	CommandCode     *string
	OperationType   *string
	CommandScope    *string
	CategoryGroup   *string
	Creator         *string      // 当前 admin 的 username；用于 private 命令 self-fallback 可见性
	UserID          *uuid.UUID   // 当前 admin 的 user_id；service 层据此派生 VisibleGroupIDs
	VisibleGroupIDs []uuid.UUID  // service 派生后填入；repo 层用作 group-share 可见性 SQL 参数
	model.ListRequest
}

// MMLParamRef represents a lightweight reference to an mml_param bound to a command.
// Replaces the old SubCommand concept — commands now directly reference mml_params.
//
// DefaultValue / JsRegex 暴露给前端做 placeholder 与初始值提示（to-do-list MOD 类需求）。
type MMLParamRef struct {
	ID              uuid.UUID              `json:"id"`
	ParamCode       string                 `json:"param_code"`
	ParamNameZh     string                 `json:"param_name_zh"`
	Tr069Path       string                 `json:"tr069_path"`
	ValueType       string                 `json:"value_type"`
	IsWritable      bool                   `json:"is_writable"`
	DefaultValue    string                 `json:"default_value,omitempty"`
	JsRegex         string                 `json:"js_regex,omitempty"`
	ValueConstraint map[string]interface{} `json:"value_constraint,omitempty"`
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

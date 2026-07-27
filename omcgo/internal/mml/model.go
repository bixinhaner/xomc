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

// TaskOrigin describes where an MML task was launched from.
type TaskOrigin string

const (
	TaskOriginConsole TaskOrigin = "console"
	TaskOriginScript  TaskOrigin = "script"
)

// TaskExecuteMode defines whether an MML task uses common broadcast semantics
// or the device-bound execution plan introduced by the script task redesign.
type TaskExecuteMode string

const (
	TaskExecuteModeCommon      TaskExecuteMode = "common"
	TaskExecuteModeDeviceBound TaskExecuteMode = "device_bound"
)

// TaskResult represents the outcome of a completed task.
type TaskResult string

const (
	ResultSuccess TaskResult = "success"
	ResultPartial TaskResult = "partial"
	ResultFailed  TaskResult = "failed"
)

// MMLCommand represents a predefined MML command, generated from standard-model.xml
// by mmlstandardloader (Sprint A schema rebuild — migration 000090).
//
// Old fields (param_template / param_paths / supported_operations / product_types)
// were dropped in 000090. New layout pivots on:
//   - target_paths  : TR-069 paths the command operates on (JSONB array of strings)
//   - target_object : add/delete object path (for ADD/RMV) — nullable
//   - group_id      : FK → mml_command_groups.id (NULL for ungrouped legacy)
//   - command_name_i18n / confirm_msg_i18n : {"zh-CN": "...", "en-US": "..."}
//   - require_confirm : 危险命令二次确认标志
//
// One command now binds to exactly one operation_type (LST/MOD/ADD/RMV/...);
// the "supported_operations" multi-op concept is gone because the standard model
// already emits separate command_codes per operation.
type MMLCommand struct {
	ID            uuid.UUID `json:"id"`
	CommandName   string    `json:"command_name"`
	CommandCode   string    `json:"command_code"`
	Category      string    `json:"category"`
	Description   string    `json:"description"`
	RPCMethod     string    `json:"rpc_method"`
	OperationType string    `json:"operation_type" db:"operation_type"`
	HelpDoc       string    `json:"help_doc" db:"help_doc"`
	Notes         string    `json:"notes" db:"notes"`

	// TargetPaths：自 migration 000095 起语义降级为"派生缓存"，
	// 由 mml_command_sub_fields trigger (trg_mml_sub_fields_target_paths) 自动维护；
	// 仍可直接 SELECT，但 admin/import 写入 sub_fields 后自动重算，避免手动同步。
	TargetPaths []string `json:"target_paths"`

	TargetObject    string            `json:"target_object,omitempty"`
	GroupID         *uuid.UUID        `json:"group_id,omitempty"`
	CommandNameI18n map[string]string `json:"command_name_i18n"`
	RequireConfirm  bool              `json:"require_confirm"`
	ConfirmMsgI18n  map[string]string `json:"confirm_msg_i18n"`
	CreatedAt       time.Time         `json:"created_at"`
	Params          []MMLParamRef     `json:"params,omitempty"`

	// LogicalCode：去 op 前缀的逻辑命令码（"LST DEVICE_INFO" → "DEVICE_INFO"）；
	// 同 logical 不同 op 是命令树叶子同分支。
	// 不再持久化为 DB 列（已 DROP）；由代码从 command_code 派生填充
	// （见 deriveLogicalCodeFromCommandCode）。JSON 契约保持不变。
	LogicalCode string `json:"logical_code"`
	// LogicalNameI18n：逻辑命令显示名（命令树叶子 label 前缀）
	// 例：{"en-US":"Device info","zh-CN":"设备信息"} → 叶子 "Device info(LST DEVICE_INFO)"
	LogicalNameI18n  map[string]string `json:"logical_name_i18n" db:"logical_name_i18n"`
	Source           string            `json:"source" db:"source"`                       // standard / admin
	CatalogProtected bool              `json:"catalog_protected" db:"catalog_protected"` // standard 行不可删除

	// SubFields：T-0123-P0 命令 → sub-field 多对多关系
	// 由 admin handler GET /sub-fields 端点加载；按 sort_order 排序；mml_command_sub_fields 表
	SubFields []MMLCommandSubField `json:"sub_fields,omitempty"`
}

// ScriptStatus represents the current state of an MML script.
//
// Two families of values:
//   - 定义态: active / archived（脚本本身是否启用）
//   - 执行态: pending / running / paused / completed / failed / cancelled
//     （脚本最近一次执行的生命周期阶段，与 TaskStatus 对齐）
//
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
	ID                uuid.UUID     `json:"id"`
	ImportSessionID   uuid.UUID     `json:"-"`
	ScriptName        string        `json:"script_name"`
	Description       string        `json:"description"`
	Content           string        `json:"content"`
	OriginalFilename  string        `json:"original_filename"`
	ContentSHA256     string        `json:"content_sha256"`
	ValidationVersion string        `json:"validation_version"`
	ValidatedAt       *time.Time    `json:"validated_at,omitempty"`
	PlanItems         []MMLPlanItem `json:"plan_items"`
	ValidationSummary JSONMap       `json:"validation_summary"`
	Creator           string        `json:"creator"`
	Tags              []string      `json:"tags"`
	Status            ScriptStatus  `json:"status"`
	StartTime         *time.Time    `json:"start_time,omitempty"`
	EndTime           *time.Time    `json:"end_time,omitempty"`
	Type              ScriptType    `json:"type"`
	Progress          float64       `json:"progress"`
	Result            JSONMap       `json:"result,omitempty"`
	// 最近一次执行终态（由 MMLAggregator 在对应 mml_task 收敛后回写）。
	// per-execution 详情查 mml_tasks，脚本层只持有"最近一次"指针。
	LastRunStatus *string    `json:"last_run_status,omitempty"`
	LastRunAt     *time.Time `json:"last_run_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// JSONMap is a helper type for nullable JSONB map fields.
type JSONMap map[string]interface{}

// MMLPlanItem is one normalized row in a device-bound execution plan.
type MMLPlanItem struct {
	LineNo   int                    `json:"line_no"`
	DeviceSN string                 `json:"device_sn"`
	Order    int                    `json:"order"`
	RawLine  string                 `json:"raw_line,omitempty"`
	Command  map[string]interface{} `json:"command"`

	// Flat fields are accepted for import/API compatibility. They are folded
	// into Command before persistence and fanout.
	CommandCode   string                 `json:"command_code,omitempty"`
	OperationType string                 `json:"operation_type,omitempty"`
	Parameters    map[string]interface{} `json:"parameters,omitempty"`
}

// MMLTask represents an MML command execution task.
type MMLTask struct {
	ID                      uuid.UUID                `json:"id"`
	TaskName                string                   `json:"task_name"`
	RequestID               string                   `json:"request_id,omitempty"`
	ScriptID                *uuid.UUID               `json:"script_id,omitempty"`
	ScriptName              string                   `json:"script_name,omitempty"`
	TaskOrigin              TaskOrigin               `json:"task_origin"`
	ScriptContentSHA256     string                   `json:"script_content_sha256"`
	ScriptValidationVersion string                   `json:"script_validation_version"`
	DeviceSNs               []string                 `json:"device_sns"`
	Commands                []map[string]interface{} `json:"commands"`
	CommandCount            int                      `json:"command_count,omitempty"`
	ExecuteMode             TaskExecuteMode          `json:"execute_mode"`
	PlanItems               []MMLPlanItem            `json:"plan_items"`
	PlanItemCount           int                      `json:"plan_item_count,omitempty"`
	Status                  TaskStatus               `json:"status"`
	Results                 []map[string]interface{} `json:"results"`
	Creator                 string                   `json:"creator"`
	Executor                string                   `json:"executor,omitempty"`
	CreatedAt               time.Time                `json:"created_at"`
	UpdatedAt               time.Time                `json:"updated_at"`

	// Scheduling
	ExecuteType ExecuteType `json:"execute_type"`
	ScheduledAt *time.Time  `json:"scheduled_at,omitempty"`
	PeriodStart *time.Time  `json:"period_start,omitempty"`
	PeriodEnd   *time.Time  `json:"period_end,omitempty"`
	PeriodTime  string      `json:"period_time,omitempty"`

	// Retry strategy
	OfflineRetry        bool `json:"offline_retry"`
	OfflineRetryWait    int  `json:"offline_retry_wait"`
	FailedRetry         bool `json:"failed_retry"`
	FailedRetryCount    int  `json:"failed_retry_count"`
	FailedRetryInterval int  `json:"failed_retry_interval"`

	// Execution timestamps
	StartedAt  *time.Time `json:"started_at,omitempty"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`

	// Statistics
	TotalDevices int         `json:"total_devices"`
	SuccessCount int         `json:"success_count"`
	FailedCount  int         `json:"failed_count"`
	Result       *TaskResult `json:"result,omitempty"`
	LatestRun    *MMLTaskRun `json:"latest_run,omitempty"`

	// Scheduler 调度字段（P2/P3，docs/design/mml-task-flow-design-20260424.md）
	// NextTriggerAt: 下次触发时刻；PeriodicParentID: periodic 子实例指向模板。
	NextTriggerAt    *time.Time `json:"next_trigger_at,omitempty"`
	PeriodicParentID *uuid.UUID `json:"parent_task_id,omitempty"`

	// PathTranslationWarning — 整改方案 Stage 3 — UI 警告标签。
	// 不入库，由 GetTask handler 聚合 device_tasks 的 has_path_translation_miss
	// 计数后填充；nil = 任务非 MML 来源或者聚合未启用；AnyMiss=true 时前端
	// 任务详情页显示"路径翻译警告"标签。
	PathTranslationWarning *PathTranslationWarning `json:"path_translation_warning,omitempty"`

	// T-0168: 路径翻译审计元数据，持久化到 mml_tasks 表 4 列（migration 000171）。
	// 由 Service.translateTaskPaths 在 fanout 前根据 TranslationOutcome 写入。
	//
	// ProductResolved=false 表示设备 productClass 未匹配任何 product（激进路线下
	// 任务仍正常 fanout，所有 path 走 orphan_passthrough 原路径下发），并触发
	// Prometheus 告警 mml_path_translation_orphan_total。
	ProductResolved       bool       `json:"product_resolved"`
	MatchedProductID      *uuid.UUID `json:"matched_product_id,omitempty"`
	MatchedProductClass   string     `json:"matched_product_class,omitempty"`
	PathTranslationSource string     `json:"path_translation_source,omitempty"` // discovered/default/passthrough/orphan_passthrough/mixed
}

// MMLTaskRun is a lightweight child-execution summary embedded in periodic
// parent list rows so the UI can show the latest run progress without N+1 calls.
type MMLTaskRun struct {
	ID            uuid.UUID       `json:"id"`
	ExecuteType   ExecuteType     `json:"execute_type"`
	ExecuteMode   TaskExecuteMode `json:"execute_mode"`
	Status        TaskStatus      `json:"status"`
	Result        *TaskResult     `json:"result,omitempty"`
	TotalDevices  int             `json:"total_devices"`
	SuccessCount  int             `json:"success_count"`
	FailedCount   int             `json:"failed_count"`
	CommandCount  int             `json:"command_count"`
	PlanItemCount int             `json:"plan_item_count"`
	StartedAt     *time.Time      `json:"started_at,omitempty"`
	FinishedAt    *time.Time      `json:"finished_at,omitempty"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

// PathTranslationWarning 是 MML 任务详情中关于 standardPath ↔ privatePath 翻译
// 未命中的聚合视图。来源：device_tasks 表的 has_path_translation_miss /
// path_translation_miss_count 列（migration 000114）。
type PathTranslationWarning struct {
	AnyMiss     bool  `json:"any_miss"`     // 任意 device 出现 fallback
	DeviceCount int   `json:"device_count"` // 受影响的 device 数
	PathCount   int64 `json:"path_count"`   // 全部 miss 的 path 累计计数
}

// CommandFilter specifies criteria for listing MML commands.
//
// admin 端列表（T-Mml-Admin）通过 GroupID / Source 过滤实现"按分组浏览 + 仅看
// standard 来源"的视图；为空 = 不过滤，保持现有 console 用法不变。
type CommandFilter struct {
	Category *string
	Search   *string
	GroupID  *uuid.UUID
	Source   *string // 'standard' / 'extension' / 'admin'；用于 admin 视图过滤 customized 行
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
	ScriptName  *string // case-insensitive substring match on current mml_scripts.script_name
	TaskOrigin  *TaskOrigin
	model.ListRequest
}

// MMLCustomCommand represents a user-defined custom command (renamed from MMLTemplate).
//
// OwnerUserID 是 migration 000134 加入的 FK 字段，正在替代历史 Creator(username)：
//   - 写：Repository.Create 同时设 Creator + OwnerUserID（dual-write）
//   - 读：Phase 1 visibility 过滤仍用 Creator；编辑 / 删除鉴权优先用 OwnerUserID（更准确）
//   - 历史行可能 OwnerUserID=nil（000134 backfill 未命中）；isOwnerOrSuper 用 Creator 兜底
type MMLCustomCommand struct {
	ID            uuid.UUID              `json:"id"`
	CommandName   string                 `json:"command_name"`
	CommandCode   string                 `json:"command_code"`
	OperationType string                 `json:"operation_type"`
	CommandScope  string                 `json:"command_scope"` // private or public
	CategoryGroup string                 `json:"category_group,omitempty"`
	Parameters    map[string]interface{} `json:"parameters"`
	ParamPaths    []string               `json:"param_paths"`
	// ParamPathsProvided 仅用于 Update 的字段掩码；防止省略 param_paths 的 PUT
	// 用 service 层旧快照覆盖并发 Path 关联写入。
	ParamPathsProvided bool       `json:"-"`
	Description        string     `json:"description"`
	Creator            string     `json:"creator"`
	OwnerUserID        *uuid.UUID `json:"owner_user_id,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

// CustomCommandFilter specifies criteria for listing MML custom commands.
//
// 私有命令可见性（T-0090-c）：
//   - public 命令：始终可见
//   - private 命令：仅在以下任一条件满足时可见
//     (a) creator == Creator （self fallback，防止脱离 group 后看不见自己创建的）
//     (b) creator's RBAC group(s) 与 VisibleGroupIDs 交集非空 （group-share）
//   - 若 Creator 与 VisibleGroupIDs 均为空 → 默认 deny private（仅 public 可见）
//
// UserID 由 service 层接收后调 RoleQuerier 派生 VisibleGroupIDs；repo 层仅消费派生结果。
type CustomCommandFilter struct {
	CommandCode   *string
	OperationType *string
	CommandScope  *string
	CategoryGroup *string
	// ProductID 非空时，列表只返回当前产品参数模型仍支持至少一个 path 的模板，
	// 并把模板中的不支持 path 一并裁掉。管理端不传此字段，仍返回完整模板。
	ProductID       *uuid.UUID
	Creator         *string     // 当前 admin 的 username；用于 private 命令 self-fallback 可见性
	UserID          *uuid.UUID  // 当前 admin 的 user_id；service 层据此派生 VisibleGroupIDs
	VisibleGroupIDs []uuid.UUID // service 派生后填入；repo 层用作 group-share 可见性 SQL 参数
	model.ListRequest
}

// MMLParamRef represents a lightweight reference to an mml_param bound to a command.
// Replaces the old SubCommand concept — commands now directly reference mml_params.
//
// DefaultValue / JsRegex 暴露给前端做 placeholder 与初始值提示（to-do-list MOD 类需求）。
type MMLParamRef struct {
	ID                uuid.UUID              `json:"id"`
	ParamCode         string                 `json:"param_code"`
	ParamNameZh       string                 `json:"param_name_zh"`
	Tr069Path         string                 `json:"tr069_path"`
	ValueType         string                 `json:"value_type"`
	IsWritable        bool                   `json:"is_writable"`
	IsRequired        bool                   `json:"is_required"`
	DefaultValue      string                 `json:"default_value,omitempty"`
	JsRegex           string                 `json:"js_regex,omitempty"`
	ValueConstraint   map[string]interface{} `json:"value_constraint,omitempty"`
	PathMode          string                 `json:"path_mode,omitempty"`
	PrivatePath       string                 `json:"private_path,omitempty"`
	TranslationSource string                 `json:"translation_source,omitempty"`
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

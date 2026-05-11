package ops

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
)

// ============================================================
// F06-ops-management T-0112..T-0111 扩展类型
// 来源 PRD: docs/project/prd/F06-ops-management.md §7
// 推进计划: docs/project/F06-ops-management-implementation-plan.md
// ============================================================

// RiskLevel 模板 / 任务 / 命令风险等级（PRD §4.2.2）。
type RiskLevel string

const (
	RiskSafe      RiskLevel = "safe"      // L1 只读 / 查询
	RiskCautious  RiskLevel = "cautious"  // L2 单设备写 / 重启
	RiskDangerous RiskLevel = "dangerous" // L3 factory_reset / 跨多设备批量写
)

// ApprovalState 任务审批状态（PRD §8.2）。
type ApprovalState string

const (
	ApprovalNotRequired ApprovalState = "not_required"
	ApprovalPending     ApprovalState = "pending"
	ApprovalApproved    ApprovalState = "approved"
	ApprovalRejected    ApprovalState = "rejected"
)

// FailurePolicy 任务失败策略（PRD §5.3.2）。
type FailurePolicy string

const (
	FailureContinue FailurePolicy = "continue"
	FailureAbort    FailurePolicy = "abort"
	FailureRetry    FailurePolicy = "retry"
	FailureRollback FailurePolicy = "rollback"
)

// DiagnosticStatus 诊断执行状态（PRD §5.4）。
type DiagnosticStatus string

const (
	DiagPending  DiagnosticStatus = "pending"
	DiagRunning  DiagnosticStatus = "running"
	DiagComplete DiagnosticStatus = "complete"
	DiagFailed   DiagnosticStatus = "failed"
	DiagTimeout  DiagnosticStatus = "timeout"
)

// DownloadStatus 运维下载状态（PRD §5.5）。
type DownloadStatus string

const (
	DownloadPending   DownloadStatus = "pending"
	DownloadUploading DownloadStatus = "uploading"
	DownloadComplete  DownloadStatus = "complete"
	DownloadFailed    DownloadStatus = "failed"
	DownloadExpired   DownloadStatus = "expired"
)

// MaintenanceWindowStatus 维护窗口状态（PRD §6.2）。
type MaintenanceWindowStatus string

const (
	MWPlanned   MaintenanceWindowStatus = "planned"
	MWApproved  MaintenanceWindowStatus = "approved"
	MWActive    MaintenanceWindowStatus = "active"
	MWEnded     MaintenanceWindowStatus = "ended"
	MWCancelled MaintenanceWindowStatus = "cancelled"
)

// OpsTaskExecution 任务在单台设备上单个步骤的执行记录（PRD §7.2）。
type OpsTaskExecution struct {
	ID           uuid.UUID       `json:"id"`
	TaskID       uuid.UUID       `json:"task_id"`
	DeviceSN     string          `json:"device_sn"`
	StepIndex    int             `json:"step_index"`
	StepName     string          `json:"step_name"`
	StepType     string          `json:"step_type"`
	Status       string          `json:"status"`
	StartedAt    *time.Time      `json:"started_at,omitempty"`
	CompletedAt  *time.Time      `json:"completed_at,omitempty"`
	DurationMS   int             `json:"duration_ms"`
	Request      json.RawMessage `json:"request,omitempty"`
	Response     json.RawMessage `json:"response,omitempty"`
	ErrorMessage string          `json:"error_message,omitempty"`
	CreatedAt    time.Time       `json:"created_at"`
}

// OpsDiagnostic 诊断记录。
type OpsDiagnostic struct {
	ID           uuid.UUID        `json:"id"`
	DeviceSN     *string          `json:"device_sn,omitempty"`
	DiagType     string           `json:"diag_type"`
	Initiator    string           `json:"initiator"`
	Request      json.RawMessage  `json:"request"`
	Result       json.RawMessage  `json:"result,omitempty"`
	Status       DiagnosticStatus `json:"status"`
	StartedAt    time.Time        `json:"started_at"`
	CompletedAt  *time.Time       `json:"completed_at,omitempty"`
	DurationMS   int              `json:"duration_ms"`
	Operator     string           `json:"operator,omitempty"`
	TaskID       *uuid.UUID       `json:"task_id,omitempty"`
	FilePath     string           `json:"file_path,omitempty"`
	ErrorMessage string           `json:"error_message,omitempty"`
	CreatedAt    time.Time        `json:"created_at"`
}

// OpsDownload 下载记录。
type OpsDownload struct {
	ID          uuid.UUID      `json:"id"`
	DeviceSN    string         `json:"device_sn"`
	ContentType string         `json:"content_type"`
	FilePath    string         `json:"file_path"`
	FileSize    int64          `json:"file_size"`
	Checksum    string         `json:"checksum,omitempty"`
	Status      DownloadStatus `json:"status"`
	Operator    string         `json:"operator,omitempty"`
	TaskID      *uuid.UUID     `json:"task_id,omitempty"`
	ExpiresAt   *time.Time     `json:"expires_at,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
}

// OpsAuditLog 运维审计日志（独立高频写，区别于通用 audit_logs）。
type OpsAuditLog struct {
	ID              uuid.UUID       `json:"id"`
	OpType          string          `json:"op_type"`
	TargetType      string          `json:"target_type"`
	TargetID        string          `json:"target_id"`
	OperatorUserID  *uuid.UUID      `json:"operator_user_id,omitempty"`
	OperatorName    string          `json:"operator_name"`
	RiskLevel       RiskLevel       `json:"risk_level"`
	Input           json.RawMessage `json:"input,omitempty"`
	OutputSummary   string          `json:"output_summary,omitempty"`
	Result          string          `json:"result"`
	ApproverUserID  *uuid.UUID      `json:"approver_user_id,omitempty"`
	BreakGlass      bool            `json:"break_glass"`
	ClientIP        string          `json:"client_ip,omitempty"`
	UserAgent       string          `json:"user_agent,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
}

// OpsMaintenanceWindow 维护窗口。
type OpsMaintenanceWindow struct {
	ID              uuid.UUID               `json:"id"`
	Name            string                  `json:"name"`
	ScopeType       string                  `json:"scope_type"`
	ScopeIDs        json.RawMessage         `json:"scope_ids"`
	StartAt         time.Time               `json:"start_at"`
	EndAt           time.Time               `json:"end_at"`
	SuppressAlarms  bool                    `json:"suppress_alarms"`
	PauseProvision  bool                    `json:"pause_provision"`
	AllowDangerous  bool                    `json:"allow_dangerous"`
	Reason          string                  `json:"reason,omitempty"`
	CreatorUserID   *uuid.UUID              `json:"creator_user_id,omitempty"`
	ApproverUserID  *uuid.UUID              `json:"approver_user_id,omitempty"`
	Status          MaintenanceWindowStatus `json:"status"`
	CreatedAt       time.Time               `json:"created_at"`
	UpdatedAt       time.Time               `json:"updated_at"`
}

// OpsPlaybook 知识库条目（PRD §6.5）。
type OpsPlaybook struct {
	ID                   uuid.UUID       `json:"id"`
	AlarmPattern         json.RawMessage `json:"alarm_pattern"`
	RecommendedTemplates json.RawMessage `json:"recommended_templates"`
	Title                string          `json:"title"`
	Docs                 string          `json:"docs,omitempty"`
	Tags                 json.RawMessage `json:"tags"`
	SuccessRate          *float64        `json:"success_rate,omitempty"`
	UseCount             int             `json:"use_count"`
	CreatedAt            time.Time       `json:"created_at"`
	UpdatedAt            time.Time       `json:"updated_at"`
}

// ---- Filters ----

type DiagnosticFilter struct {
	DeviceSN string
	DiagType string
	Status   *DiagnosticStatus
	model.ListRequest
}

type DownloadFilter struct {
	DeviceSN    string
	ContentType string
	Status      *DownloadStatus
	model.ListRequest
}

type AuditLogFilter struct {
	OpType         string
	TargetType     string
	TargetID       string
	OperatorUserID *uuid.UUID
	RiskLevel      *RiskLevel
	BreakGlass     *bool
	FromTime       *time.Time
	ToTime         *time.Time
	model.ListRequest
}

type MaintenanceWindowFilter struct {
	Status *MaintenanceWindowStatus
	model.ListRequest
}

type PlaybookFilter struct {
	Keyword string
	model.ListRequest
}

type TaskExecutionFilter struct {
	TaskID   *uuid.UUID
	DeviceSN string
	Status   string
	model.ListRequest
}

// ---- Diagnostic request payloads ----

type DiagPingRequest struct {
	DeviceSN string `json:"device_sn" binding:"required"`
	Host     string `json:"host" binding:"required"`
	Count    int    `json:"count,omitempty"`
	Timeout  int    `json:"timeout,omitempty"`
}

type DiagTracerouteRequest struct {
	DeviceSN string `json:"device_sn" binding:"required"`
	Host     string `json:"host" binding:"required"`
}

type DiagThroughputRequest struct {
	DeviceSN  string `json:"device_sn" binding:"required"`
	URL       string `json:"url" binding:"required"`
	Direction string `json:"direction,omitempty"` // download / upload
}

// ---- Download request payloads ----

type CollectDownloadRequest struct {
	DeviceSN     string   `json:"device_sn" binding:"required"`
	ContentTypes []string `json:"content_types" binding:"required"`
}

// ---- Maintenance window payloads ----

type CreateMaintenanceWindowRequest struct {
	Name            string          `json:"name" binding:"required"`
	ScopeType       string          `json:"scope_type" binding:"required"`
	ScopeIDs        json.RawMessage `json:"scope_ids"`
	StartAt         time.Time       `json:"start_at" binding:"required"`
	EndAt           time.Time       `json:"end_at" binding:"required"`
	SuppressAlarms  bool            `json:"suppress_alarms"`
	PauseProvision  bool            `json:"pause_provision"`
	AllowDangerous  bool            `json:"allow_dangerous"`
	Reason          string          `json:"reason,omitempty"`
}

// ---- Playbook payloads ----

type MatchPlaybookRequest struct {
	AlarmCode string `json:"alarm_code" binding:"required"`
	Severity  string `json:"severity,omitempty"`
}

// ---- Break-glass payloads ----

type BreakGlassRequest struct {
	TicketID string `json:"ticket_id" binding:"required"`
	Reason   string `json:"reason" binding:"required"`
}

package ops

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
)

// OpsTaskStatus represents the current state of an ops task.
type OpsTaskStatus string

const (
	OpsTaskPending   OpsTaskStatus = "pending"
	OpsTaskRunning   OpsTaskStatus = "running"
	OpsTaskSuccess   OpsTaskStatus = "success"
	OpsTaskFailed    OpsTaskStatus = "failed"
	OpsTaskCancelled OpsTaskStatus = "cancelled"
	OpsTaskPaused    OpsTaskStatus = "paused"
)

// OpsTemplate represents a reusable operations procedure template.
type OpsTemplate struct {
	ID                uuid.UUID       `json:"id"`
	TemplateName      string          `json:"template_name"`
	Description       string          `json:"description"`
	Category          string          `json:"category"`
	TargetDeviceTypes json.RawMessage `json:"target_device_types"`
	Steps             json.RawMessage `json:"steps"`
	EstimatedDuration int             `json:"estimated_duration"`
	Creator           string          `json:"creator"`
	UseCount          int             `json:"use_count"`
	Tags              json.RawMessage `json:"tags"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
}

// OpsTask represents an execution of an ops template against devices.
//
// T-0101-d 状态机集成（PRD §5.3.1 + §8.2）：
//   - Status 是任务执行状态轴（pending / running / success / failed / cancelled / paused）
//   - ApprovalState 是任务审批状态轴（not_required / pending / approved / rejected）
//   - 两轴正交：任务创建时按 RiskLevel 决定 ApprovalState 初值；不需要审批的直接走 router
//     拉起；需要审批的等 Approve → ApprovalState=approved → Status=running
type OpsTask struct {
	ID           uuid.UUID       `json:"id"`
	TaskName     string          `json:"task_name"`
	TemplateID   *uuid.UUID      `json:"template_id,omitempty"`
	DeviceSNs    json.RawMessage `json:"device_sns"`
	Status       OpsTaskStatus   `json:"status"`
	CurrentStep  int             `json:"current_step"`
	TotalSteps   int             `json:"total_steps"`
	Progress     int             `json:"progress"`
	SuccessCount int             `json:"success_count"`
	FailCount    int             `json:"fail_count"`
	TotalCount   int             `json:"total_count"`
	Creator      string          `json:"creator"`
	Message      string          `json:"message"`
	StartedAt    *time.Time      `json:"started_at,omitempty"`
	CompletedAt  *time.Time      `json:"completed_at,omitempty"`
	// T-0101-d 状态机字段（DB 列 from migration 000080）
	RiskLevel      RiskLevel     `json:"risk_level"`
	ApprovalState  ApprovalState `json:"approval_state"`
	ApproverUserID *uuid.UUID    `json:"approver_user_id,omitempty"`
	ApprovedAt     *time.Time    `json:"approved_at,omitempty"`
	CreatedAt      time.Time     `json:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at"`
}

// OpsCommandRecord represents a single command execution record.
type OpsCommandRecord struct {
	ID           uuid.UUID `json:"id"`
	CommandText  string    `json:"command_text"`
	DeviceSN     string    `json:"device_sn"`
	DeviceName   string    `json:"device_name"`
	Operator     string    `json:"operator"`
	ExecuteTime  time.Time `json:"execute_time"`
	Duration     int       `json:"duration"`
	Success      bool      `json:"success"`
	Output       string    `json:"output"`
	ErrorMessage *string   `json:"error_message,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

// TemplateFilter specifies criteria for listing ops templates.
type TemplateFilter struct {
	Category string
	Keyword  string
	model.ListRequest
}

// TaskFilter specifies criteria for listing ops tasks.
type TaskFilter struct {
	Status     *OpsTaskStatus
	TemplateID *uuid.UUID
	Keyword    string
	Creator    string
	model.ListRequest
}

// CommandRecordFilter specifies criteria for listing ops command records.
type CommandRecordFilter struct {
	DeviceSN string
	Operator string
	Success  *bool
	model.ListRequest
}

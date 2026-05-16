// Package trace implements TR069 message trace (报文跟踪) — T-0137 移植自老 OMC。
// 设计文档：docs/design/TR069报文跟踪-设计.md
// PRD：docs/project/prd/F01-tr069-message-trace.md
//
// 设计决策（D1-D10 见设计文档 §10）：
//   - D6 独立 internal/trace/ 模块，不复用 internal/task/ 队列
//   - D8 trace_messages hypertable retention 3 天
//   - D3 报文原样落库，不做 mask
//   - D2 仅冗余存 device_sn，不存 device_id（避免设备删除导致历史丢失）
package trace

import (
	"time"

	"github.com/google/uuid"

	"github.com/omcgo/omcgo/internal/core/model"
)

// TaskStatus 抓包任务状态。
type TaskStatus string

const (
	TaskStatusRunning TaskStatus = "running"
	TaskStatusStopped TaskStatus = "stopped"
	TaskStatusPurged  TaskStatus = "purged"
)

// Direction 报文方向。
type Direction string

const (
	DirectionIn  Direction = "in"  // CPE → ACS
	DirectionOut Direction = "out" // ACS → CPE
)

// MaxInlinePayloadBytes M1 阶段所有报文 inline 存储；超过此阈值时截断后写入（M2 改为外置 MinIO）。
// 32KB 与设计文档 §4.3 阈值一致。
const MaxInlinePayloadBytes = 32 * 1024

// DefaultDurationMinutes 默认抓包窗口时长（分钟）。
const DefaultDurationMinutes = 5

// MaxDurationMinutes 单任务最长窗口时长（分钟），防误操作。
const MaxDurationMinutes = 60

// Task 抓包任务（trace_tasks 表行）。
type Task struct {
	ID           uuid.UUID  `json:"id"`
	DeviceSN     string     `json:"device_sn"`
	OperatorCode string     `json:"operator_code"`
	Status       TaskStatus `json:"status"`
	StartTime    time.Time  `json:"start_time"`
	ExpiresAt    time.Time  `json:"expires_at"`
	StoppedAt    *time.Time `json:"stopped_at,omitempty"`
	PurgedAt     *time.Time `json:"purged_at,omitempty"`
	CreatedBy    string     `json:"created_by"`
	MessageCount int        `json:"message_count"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// IsActive 任务是否在抓包窗口内。
func (t *Task) IsActive(now time.Time) bool {
	return t.Status == TaskStatusRunning && now.Before(t.ExpiresAt)
}

// Message 单条报文（trace_messages hypertable 行）。
type Message struct {
	CapturedAt       time.Time `json:"captured_at"`
	ID               uuid.UUID `json:"id"`
	TaskID           uuid.UUID `json:"task_id"`
	DeviceSN         string    `json:"device_sn"`
	Direction        Direction `json:"direction"`
	RPCMethod        string    `json:"rpc_method,omitempty"`
	CwmpID           string    `json:"cwmp_id,omitempty"`
	SessionID        string    `json:"session_id,omitempty"`
	HTTPStatus       int       `json:"http_status,omitempty"`
	PayloadSizeBytes int       `json:"payload_size_bytes"`
	PayloadInline    string    `json:"payload_inline,omitempty"`
	PayloadObjectKey string    `json:"payload_object_key,omitempty"` // M2 启用 MinIO 时填充
}

// CreateTaskRequest 创建抓包任务请求体。
type CreateTaskRequest struct {
	DeviceSN        string `json:"device_sn" binding:"required"`
	DurationMinutes int    `json:"duration_minutes"`
}

// StopTaskRequest 停止任务请求体。
type StopTaskRequest struct {
	Purge bool `json:"purge"`
}

// TaskFilter 任务列表过滤。
type TaskFilter struct {
	DeviceSN     string
	Status       TaskStatus
	OperatorCode string
	model.ListRequest
}

// MessageFilter 报文列表过滤。
type MessageFilter struct {
	TaskID    uuid.UUID
	DeviceSN  string
	Direction Direction
	RPCMethod string
	StartTime *time.Time
	EndTime   *time.Time
	model.ListRequest
}

// TaskEvent 是 trace.task.{started,stopped,purged} NATS 事件 payload。
type TaskEvent struct {
	TaskID       uuid.UUID  `json:"task_id"`
	DeviceSN     string     `json:"device_sn"`
	Status       TaskStatus `json:"status"`
	OperatorCode string     `json:"operator_code,omitempty"`
	CreatedBy    string     `json:"created_by,omitempty"`
	StartTime    time.Time  `json:"start_time,omitempty"`
	ExpiresAt    time.Time  `json:"expires_at,omitempty"`
	StoppedAt    *time.Time `json:"stopped_at,omitempty"`
	// Reason 解释 stopped 触发方（manual / timeout / new_task_replaced）。
	Reason string `json:"reason,omitempty"`
}

// ToTaskEvent 把 Task 转换成事件 payload。
func ToTaskEvent(t *Task, reason string) TaskEvent {
	return TaskEvent{
		TaskID:       t.ID,
		DeviceSN:     t.DeviceSN,
		Status:       t.Status,
		OperatorCode: t.OperatorCode,
		CreatedBy:    t.CreatedBy,
		StartTime:    t.StartTime,
		ExpiresAt:    t.ExpiresAt,
		StoppedAt:    t.StoppedAt,
		Reason:       reason,
	}
}

// ExportJobStatus 异步导出任务状态。
type ExportJobStatus string

const (
	ExportJobQueued  ExportJobStatus = "queued"
	ExportJobRunning ExportJobStatus = "running"
	ExportJobDone    ExportJobStatus = "done"
	ExportJobFailed  ExportJobStatus = "failed"
)

// ExportJob 异步下载任务（trace_export_jobs 行）。
type ExportJob struct {
	ID           uuid.UUID       `json:"id"`
	TaskID       uuid.UUID       `json:"task_id"`
	RequestedBy  string          `json:"requested_by"`
	Status       ExportJobStatus `json:"status"`
	ObjectKey    string          `json:"object_key,omitempty"`
	ObjectBucket string          `json:"object_bucket,omitempty"`
	MessageCount int             `json:"message_count"`
	SizeBytes    int64           `json:"size_bytes"`
	ErrorMessage string          `json:"error_message,omitempty"`
	StartedAt    *time.Time      `json:"started_at,omitempty"`
	CompletedAt  *time.Time      `json:"completed_at,omitempty"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
	// DownloadURL 由 handler 在响应时按需生成（预签名），不存 PG。
	DownloadURL string `json:"download_url,omitempty"`
}

// ExportRequestedEvent trace.export.requested NATS 事件 payload。
type ExportRequestedEvent struct {
	JobID  uuid.UUID `json:"job_id"`
	TaskID uuid.UUID `json:"task_id"`
}

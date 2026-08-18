package task

import (
	"encoding/json"
	"fmt"
	"time"
)

// TaskStatus 任务状态
type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"   // 等待下发
	TaskStatusSent      TaskStatus = "sent"      // 已发送给 CPE
	TaskStatusCompleted TaskStatus = "completed" // CPE 执行成功
	TaskStatusFailed    TaskStatus = "failed"    // CPE 执行失败
	TaskStatusExpired   TaskStatus = "expired"   // 任务超时
	TaskStatusCancelled TaskStatus = "cancelled" // 任务取消
)

type AdmissionClass string

const (
	AdmissionClassNormal         AdmissionClass = "normal"
	AdmissionClassAccessProbe    AdmissionClass = "access_probe"
	AdmissionClassSecurityAction AdmissionClass = "security_action"
)

// TaskSource 任务来源
type TaskSource string

const (
	TaskSourceAPI          TaskSource = "api"           // REST API 创建
	TaskSourceScheduler    TaskSource = "scheduler"     // 定时任务创建
	TaskSourceSystem       TaskSource = "system"        // 系统内部创建
	TaskSourceMML          TaskSource = "mml"           // MML 批量任务扇出
	TaskSourceOps          TaskSource = "ops"           // F06 运维即时命令 (T-0102-c)
	TaskSourceParamSync    TaskSource = "param_sync"    // durable parameter-sync run task
	TaskSourceGeofence     TaskSource = "geofence"      // geofence location control task
	TaskSourceDeviceAccess TaskSource = "device_access" // access evidence probe task
)

// Task 表示一个设备任务
type Task struct {
	// 基本信息
	ID       string          `json:"id"`        // UUID
	DeviceSN string          `json:"device_sn"` // 设备序列号
	Method   string          `json:"method"`    // RPC 方法名
	Params   json.RawMessage `json:"params"`    // 方法参数
	Priority int             `json:"priority"`  // 优先级 (0=最高, 默认=10)

	// TR069 相关
	CommandKey string `json:"command_key,omitempty"` // TR069 CommandKey
	CWMPID     string `json:"cwmp_id,omitempty"`     // SOAP Header ID

	// 状态管理
	Status     TaskStatus `json:"status"`      // 当前状态
	RetryCount int        `json:"retry_count"` // 当前重试次数
	MaxRetries int        `json:"max_retries"` // 最大重试次数
	// RetryIntervalSeconds 控制自动重试间隔。0 表示立即可重试。
	RetryIntervalSeconds int `json:"retry_interval_seconds"`

	// 时间戳
	CreatedAt     time.Time  `json:"created_at"`                // 创建时间
	SentAt        *time.Time `json:"sent_at,omitempty"`         // 发送时间
	CompletedAt   *time.Time `json:"completed_at,omitempty"`    // 完成时间
	ExpiresAt     *time.Time `json:"expires_at,omitempty"`      // 过期时间
	NextAttemptAt *time.Time `json:"next_attempt_at,omitempty"` // 下一次允许出队时间

	// 结果
	Result       json.RawMessage `json:"result,omitempty"`
	ErrorCode    int             `json:"error_code,omitempty"`
	ErrorMessage string          `json:"error_message,omitempty"`

	// 元数据
	Source      TaskSource `json:"source"`                // 任务来源
	CreatorID   string     `json:"creator_id,omitempty"`  // 创建者 ID
	Description string     `json:"description,omitempty"` // 任务描述

	// 任务来源外键（Source='mml' 时为 mml_tasks.id；其它来源按需扩展语义）
	SourceID     string `json:"source_id,omitempty"`
	CommandIndex int    `json:"command_index"` // MML 扇出时在 commands[] 中的索引
	DeviceIndex  int    `json:"device_index"`  // MML 扇出时在 device_sns[] 中的索引

	// 路径翻译元数据（Stage 1 — fanout 入队前 standardPath → privatePath 翻译）
	// 仅 MML fanout 阶段写入；其它来源任务保持默认 false / 0。
	HasPathTranslationMiss   bool `json:"has_path_translation_miss"`   // 任意 path 未命中 → 用 standardPath 兜底
	PathTranslationMissCount int  `json:"path_translation_miss_count"` // 未命中的 path 数量

	// T-0168: per-device 翻译来源（同 mml_tasks.path_translation_source 枚举）。
	// 首版由 fanout 从 task 直接复制（R-8.4 保证 task 内 product_class 一致）。
	// 枚举值：discovered/default/passthrough/orphan_passthrough/mixed；空值=非 MML 来源。
	PathTranslationSource string         `json:"path_translation_source,omitempty"`
	AdmissionClass        AdmissionClass `json:"admission_class"`
}

// CreateTaskRequest 创建任务请求
type CreateTaskRequest struct {
	DeviceSN  string          `json:"device_sn"`
	Method    string          `json:"method" binding:"required"`
	Params    json.RawMessage `json:"params"`
	Priority  int             `json:"priority"`
	ExpiresIn int             `json:"expires_in"` // 过期时间（秒）
	// MaxRetries 用指针区分"未设置"（nil → 默认 3）与"显式 0"（禁止重试）。
	// 旧实现用 int + (>0) 判定，无法表达"调用方显式要 0 次重试"，0 被静默改回 3。
	MaxRetries           *int       `json:"max_retries"`            // 最大重试次数（nil=默认 3，0=禁止重试）
	RetryIntervalSeconds int        `json:"retry_interval_seconds"` // 自动重试间隔（秒）
	Source               TaskSource `json:"source"`
	CreatorID            string     `json:"creator_id"`
	Description          string     `json:"description"`

	// TR069 CommandKey（预设的命令标识）
	CommandKey string `json:"command_key,omitempty"`

	// 任务来源外键（对应 device_tasks.source_id；Source='mml' 时为 mml_tasks.id）
	SourceID     string `json:"source_id"`
	CommandIndex int    `json:"command_index"`
	DeviceIndex  int    `json:"device_index"`

	// 路径翻译元数据（MML fanout 阶段写入；详见 Task.HasPathTranslationMiss 注释）
	HasPathTranslationMiss   bool           `json:"has_path_translation_miss"`
	PathTranslationMissCount int            `json:"path_translation_miss_count"`
	PathTranslationSource    string         `json:"path_translation_source,omitempty"` // T-0168
	AdmissionClass           AdmissionClass `json:"admission_class"`

	// FailImmediately creates a terminal failed task row without enqueueing it.
	// It is used by callers that already know an RPC cannot be delivered, such
	// as an MML task targeting an offline device without offline-wait enabled.
	FailImmediately bool   `json:"fail_immediately,omitempty"`
	FailReason      string `json:"fail_reason,omitempty"`
}

// TaskHistoryOptions 任务历史查询选项
type TaskHistoryOptions struct {
	Status   TaskStatus `form:"status"`
	Start    *time.Time `form:"start"`
	End      *time.Time `form:"end"`
	Page     int        `form:"page"`
	PageSize int        `form:"page_size"`
}

// TaskListResponse 任务列表响应
type TaskListResponse struct {
	Tasks    []*Task `json:"tasks"`
	Total    int64   `json:"total"`
	Page     int     `json:"page,omitempty"`
	PageSize int     `json:"page_size,omitempty"`
}

// SyncGPVSummary summarizes the latest Path B GetParameterValues sync task group for a device.
type SyncGPVSummary struct {
	SourceID            string           `json:"source_id"`
	TaskCount           int              `json:"task_count"`
	SuccessfulCommands  int              `json:"successful_commands"`
	FailedCommands      int              `json:"failed_commands"`
	RequestedPathCount  int              `json:"requested_path_count"`
	SuccessfulPathCount int              `json:"successful_path_count"`
	FailedPathCount     int              `json:"failed_path_count"`
	FailedPaths         []SyncGPVFailure `json:"failed_paths,omitempty"`
	FirstCreatedAt      time.Time        `json:"first_created_at"`
	LastCompletedAt     *time.Time       `json:"last_completed_at,omitempty"`
	WallClockSeconds    float64          `json:"wall_clock_seconds"`
}

type SyncGPVFailure struct {
	Path       string `json:"path,omitempty"`
	FaultCode  int    `json:"fault_code,omitempty"`
	FaultText  string `json:"fault_text,omitempty"`
	CommandKey string `json:"command_key,omitempty"`
	Status     string `json:"status,omitempty"`
}

// RetryBudgetCoveringExpiry returns a retry budget that cannot be exhausted
// before a task's TTL solely because successive CWMP sessions recover a sent
// task. Invalid/unscheduled inputs retain the historical default.
func RetryBudgetCoveringExpiry(expiresInSeconds, retryIntervalSeconds int) int {
	const defaultMaxRetries = 3
	if expiresInSeconds <= 0 || retryIntervalSeconds <= 0 {
		return defaultMaxRetries
	}
	return (expiresInSeconds + retryIntervalSeconds - 1) / retryIntervalSeconds
}

// NewTask 创建新任务
func NewTask(req *CreateTaskRequest) *Task {
	now := time.Now()
	task := &Task{
		ID:                       generateUUID(),
		DeviceSN:                 req.DeviceSN,
		Method:                   req.Method,
		Params:                   req.Params,
		Priority:                 req.Priority,
		CommandKey:               req.CommandKey,
		Status:                   TaskStatusPending,
		MaxRetries:               3,
		RetryIntervalSeconds:     req.RetryIntervalSeconds,
		CreatedAt:                now,
		Source:                   TaskSourceAPI,
		CreatorID:                req.CreatorID,
		Description:              req.Description,
		SourceID:                 req.SourceID,
		CommandIndex:             req.CommandIndex,
		DeviceIndex:              req.DeviceIndex,
		HasPathTranslationMiss:   req.HasPathTranslationMiss,
		PathTranslationMissCount: req.PathTranslationMissCount,
		PathTranslationSource:    req.PathTranslationSource,
		AdmissionClass:           req.AdmissionClass,
	}

	// 设置默认值
	if task.Priority <= 0 {
		task.Priority = 10
	}
	// nil → 保留默认 3；非 nil → 采纳调用方显式值（含显式 0 = 禁止重试）。
	// 负值无意义，钳到 0。
	if req.MaxRetries != nil {
		mr := *req.MaxRetries
		if mr < 0 {
			mr = 0
		}
		task.MaxRetries = mr
	}
	if task.RetryIntervalSeconds < 0 {
		task.RetryIntervalSeconds = 0
	}
	if req.ExpiresIn > 0 {
		expiresAt := now.Add(time.Duration(req.ExpiresIn) * time.Second)
		task.ExpiresAt = &expiresAt
	}
	if req.Source != "" {
		task.Source = req.Source
	}
	if task.AdmissionClass == "" {
		task.AdmissionClass = AdmissionClassNormal
	}
	if req.FailImmediately {
		reason := req.FailReason
		if reason == "" {
			reason = "task failed before enqueue"
		}
		task.MarkFailed(0, reason)
	}

	return task
}

// IsExpired 检查任务是否过期
func (t *Task) IsExpired() bool {
	if t.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*t.ExpiresAt)
}

// CanRetry 检查任务重试预算是否未耗尽（仅看次数，不看状态）。
//
// 用于 RecoverPendingTasks 对僵死 sent 任务的恢复决策：sent 任务并非"终态可重试"，
// 而是在途任务的恢复，故此处只校验 retry_count<max_retries。用户主动 retry 端点
// 的状态校验走 CanManualRetry。
func (t *Task) CanRetry() bool {
	return t.RetryCount < t.MaxRetries
}

// CanManualRetry 检查任务是否可被用户主动 retry（POST /:task_id/retry）。
//
// 仅终态失败类（failed/expired）可重试；cancelled/completed 为终态成功/人工终结，
// pending/sent 仍在途，均不可主动 retry（否则会把已取消/已完成任务"复活"回 pending）。
// 同时仍需满足重试预算（retry_count<max_retries），显式 max_retries=0 即禁止重试。
func (t *Task) CanManualRetry() bool {
	switch t.Status {
	case TaskStatusFailed, TaskStatusExpired:
		return t.CanRetry()
	default:
		return false
	}
}

// MarkSent 标记任务已发送
func (t *Task) MarkSent(cwmpID string) {
	now := time.Now()
	t.Status = TaskStatusSent
	t.CWMPID = cwmpID
	t.SentAt = &now
	t.NextAttemptAt = nil
}

// MarkCompleted 标记任务完成
func (t *Task) MarkCompleted(result json.RawMessage) {
	now := time.Now()
	t.Status = TaskStatusCompleted
	t.Result = result
	t.CompletedAt = &now
	t.NextAttemptAt = nil
}

// MarkFailed 标记任务失败
func (t *Task) MarkFailed(errorCode int, errorMessage string) {
	now := time.Now()
	t.Status = TaskStatusFailed
	t.ErrorCode = errorCode
	t.ErrorMessage = errorMessage
	t.CompletedAt = &now
	t.NextAttemptAt = nil
}

// MarkFailedWithResult 标记任务失败并附带结构化结果（如 per-param SetParameterValuesFault 详情）。
// result 为空则等价 MarkFailed。
//
// T-0174 引入：ACS 收到 SetParameterValues SOAP Fault 时把每个失败 path 的
// (parameter_name / fault_code / fault_string) 序列化到 t.Result，下游 MML
// ResultAggregator 据此触发 is_supported=false auto-learn。
func (t *Task) MarkFailedWithResult(errorCode int, errorMessage string, result json.RawMessage) {
	t.MarkFailed(errorCode, errorMessage)
	if len(result) > 0 {
		t.Result = result
	}
}

// MarkExpired 标记任务过期
func (t *Task) MarkExpired() {
	now := time.Now()
	t.Status = TaskStatusExpired
	if t.Source == TaskSourceMML && t.ErrorMessage == "" {
		t.ErrorMessage = t.commandTimeoutMessage()
	}
	t.CompletedAt = &now
}

func (t *Task) commandTimeoutMessage() string {
	if t == nil || t.ExpiresAt == nil || t.CreatedAt.IsZero() {
		return "执行命令超时"
	}
	timeout := t.ExpiresAt.Sub(t.CreatedAt)
	seconds := int(timeout.Round(time.Second) / time.Second)
	if seconds <= 0 {
		return "执行命令超时"
	}
	return fmt.Sprintf("执行命令超时，超时时间 %d 秒", seconds)
}

// ResetForRetry 重置任务以进行重试
func (t *Task) ResetForRetry() {
	t.ResetForRetryAfter(0)
}

// ResetForRetryAfter 重置任务以进行重试，并可设置下一次允许出队时间。
func (t *Task) ResetForRetryAfter(delay time.Duration) {
	t.Status = TaskStatusPending
	t.CWMPID = ""
	t.SentAt = nil
	t.RetryCount++
	t.CompletedAt = nil
	if delay > 0 {
		next := time.Now().Add(delay)
		t.NextAttemptAt = &next
	} else {
		t.NextAttemptAt = nil
	}
}

// RetryInterval 返回任务配置的自动重试间隔。
func (t *Task) RetryInterval() time.Duration {
	if t.RetryIntervalSeconds <= 0 {
		return 0
	}
	return time.Duration(t.RetryIntervalSeconds) * time.Second
}

// IsReadyForAttempt 检查任务是否已到允许出队时间。
func (t *Task) IsReadyForAttempt(now time.Time) bool {
	return t.NextAttemptAt == nil || !t.NextAttemptAt.After(now)
}

// QueueTime 返回用于队列排序的可出队时间。
func (t *Task) QueueTime() time.Time {
	if t.NextAttemptAt != nil && !t.NextAttemptAt.IsZero() {
		return *t.NextAttemptAt
	}
	return t.CreatedAt
}

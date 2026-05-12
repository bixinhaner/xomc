package task

import (
	"encoding/json"
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

// TaskSource 任务来源
type TaskSource string

const (
	TaskSourceAPI       TaskSource = "api"       // REST API 创建
	TaskSourceScheduler TaskSource = "scheduler" // 定时任务创建
	TaskSourceSystem    TaskSource = "system"    // 系统内部创建
	TaskSourceMML       TaskSource = "mml"       // MML 批量任务扇出
	TaskSourceOps       TaskSource = "ops"       // F06 运维即时命令 (T-0102-c)
)

// Task 表示一个设备任务
type Task struct {
	// 基本信息
	ID         string          `json:"id"`         // UUID
	DeviceSN   string          `json:"device_sn"`  // 设备序列号
	Method     string          `json:"method"`     // RPC 方法名
	Params     json.RawMessage `json:"params"`     // 方法参数
	Priority   int             `json:"priority"`   // 优先级 (0=最高, 默认=10)

	// TR069 相关
	CommandKey string `json:"command_key,omitempty"` // TR069 CommandKey
	CWMPID     string `json:"cwmp_id,omitempty"`     // SOAP Header ID

	// 状态管理
	Status     TaskStatus `json:"status"`               // 当前状态
	RetryCount int        `json:"retry_count"`          // 当前重试次数
	MaxRetries int        `json:"max_retries"`          // 最大重试次数

	// 时间戳
	CreatedAt   time.Time  `json:"created_at"`             // 创建时间
	SentAt      *time.Time `json:"sent_at,omitempty"`      // 发送时间
	CompletedAt *time.Time `json:"completed_at,omitempty"` // 完成时间
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`   // 过期时间

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
}

// CreateTaskRequest 创建任务请求
type CreateTaskRequest struct {
	DeviceSN    string          `json:"device_sn" binding:"required"`
	Method      string          `json:"method" binding:"required"`
	Params      json.RawMessage `json:"params"`
	Priority    int             `json:"priority"`
	ExpiresIn   int             `json:"expires_in"`   // 过期时间（秒）
	MaxRetries  int             `json:"max_retries"`  // 最大重试次数
	Source      TaskSource      `json:"source"`
	CreatorID   string          `json:"creator_id"`
	Description string          `json:"description"`

	// TR069 CommandKey（预设的命令标识）
	CommandKey string `json:"command_key,omitempty"`

	// 任务来源外键（对应 device_tasks.source_id；Source='mml' 时为 mml_tasks.id）
	SourceID     string `json:"source_id"`
	CommandIndex int    `json:"command_index"`
	DeviceIndex  int    `json:"device_index"`
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

// NewTask 创建新任务
func NewTask(req *CreateTaskRequest) *Task {
	now := time.Now()
	task := &Task{
		ID:           generateUUID(),
		DeviceSN:     req.DeviceSN,
		Method:       req.Method,
		Params:       req.Params,
		Priority:     req.Priority,
		CommandKey:   req.CommandKey,
		Status:       TaskStatusPending,
		MaxRetries:   3,
		CreatedAt:    now,
		Source:       TaskSourceAPI,
		CreatorID:    req.CreatorID,
		Description:  req.Description,
		SourceID:     req.SourceID,
		CommandIndex: req.CommandIndex,
		DeviceIndex:  req.DeviceIndex,
	}

	// 设置默认值
	if task.Priority <= 0 {
		task.Priority = 10
	}
	if req.MaxRetries > 0 {
		task.MaxRetries = req.MaxRetries
	}
	if req.ExpiresIn > 0 {
		expiresAt := now.Add(time.Duration(req.ExpiresIn) * time.Second)
		task.ExpiresAt = &expiresAt
	}
	if req.Source != "" {
		task.Source = req.Source
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

// CanRetry 检查任务是否可以重试
func (t *Task) CanRetry() bool {
	return t.RetryCount < t.MaxRetries
}

// MarkSent 标记任务已发送
func (t *Task) MarkSent(cwmpID string) {
	now := time.Now()
	t.Status = TaskStatusSent
	t.CWMPID = cwmpID
	t.SentAt = &now
}

// MarkCompleted 标记任务完成
func (t *Task) MarkCompleted(result json.RawMessage) {
	now := time.Now()
	t.Status = TaskStatusCompleted
	t.Result = result
	t.CompletedAt = &now
}

// MarkFailed 标记任务失败
func (t *Task) MarkFailed(errorCode int, errorMessage string) {
	now := time.Now()
	t.Status = TaskStatusFailed
	t.ErrorCode = errorCode
	t.ErrorMessage = errorMessage
	t.CompletedAt = &now
}

// MarkExpired 标记任务过期
func (t *Task) MarkExpired() {
	now := time.Now()
	t.Status = TaskStatusExpired
	t.CompletedAt = &now
}

// ResetForRetry 重置任务以进行重试
func (t *Task) ResetForRetry() {
	t.Status = TaskStatusPending
	t.CWMPID = ""
	t.SentAt = nil
	t.RetryCount++
}

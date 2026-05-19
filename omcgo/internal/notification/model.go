// Package notification provides the notification CRUD service and HTTP handlers
// for the OMC notification system.
package notification

import (
	"time"

	"github.com/google/uuid"

	"github.com/omcgo/omcgo/internal/core/model"
)

// NotificationType defines the category of a notification.
type NotificationType string

const (
	NotifTypeAlarm        NotificationType = "alarm"
	NotifTypeTaskComplete NotificationType = "task_complete"
	NotifTypeSystem       NotificationType = "system"
	NotifTypeApproval     NotificationType = "approval"
	NotifTypeDeviceStatus NotificationType = "device_status"
)

// NotificationPriority defines the urgency level of a notification.
type NotificationPriority string

const (
	PriorityCritical NotificationPriority = "critical"
	PriorityHigh     NotificationPriority = "high"
	PriorityNormal   NotificationPriority = "normal"
	PriorityLow      NotificationPriority = "low"
)

// NotificationStatus 与 §4.4 五态文案分层对齐（T-0157 C3）。
// 前端图标 / 颜色 / 过滤直接消费该字段；非 task 关联消息（如告警 / 系统公告）
// 通常用 StatusCompleted 表示终态。
type NotificationStatus string

const (
	StatusQueued    NotificationStatus = "queued"    // 任务已入队，等待 CPE 应答
	StatusSent      NotificationStatus = "sent"      // 已发送给 CPE，等待回应
	StatusCompleted NotificationStatus = "completed" // 终态：成功
	StatusFailed    NotificationStatus = "failed"    // 终态：失败（应答失败 / 入队失败）
	StatusExpired   NotificationStatus = "expired"   // 终态：超时
	StatusCancelled NotificationStatus = "cancelled" // 终态：取消
)

// Notification represents a single notification persisted in the database.
//
// DedupKey: T-0157 C3 — task → notification 映射的去重键。订阅器按
// (user_id, dedup_key) upsert 同一条消息，状态升级（queued → sent → completed/failed）
// 不重复插入。NULL 表示该消息非 task 关联（如告警 / 系统公告），不参与去重。
type Notification struct {
	ID        uuid.UUID            `json:"id"`
	UserID    string               `json:"user_id"`
	Type      NotificationType     `json:"type"`
	Status    NotificationStatus   `json:"status"`
	Priority  NotificationPriority `json:"priority"`
	Title     string               `json:"title"`
	Content   string               `json:"content,omitempty"`
	Link      string               `json:"link,omitempty"`
	Sender    string               `json:"sender"`
	DedupKey  *string              `json:"dedup_key,omitempty"`
	IsRead    bool                 `json:"is_read"`
	ReadAt    *time.Time           `json:"read_at,omitempty"`
	CreatedAt time.Time            `json:"created_at"`
}

// NotificationFilter specifies criteria for listing notifications.
type NotificationFilter struct {
	UserID string               // required: always filter by user
	Type   *NotificationType    // optional: filter by type
	IsRead *bool                // optional: filter by read status
	model.ListRequest
}

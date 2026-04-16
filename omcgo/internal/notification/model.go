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

// Notification represents a single notification persisted in the database.
type Notification struct {
	ID        uuid.UUID           `json:"id"`
	UserID    string              `json:"user_id"`
	Type      NotificationType    `json:"type"`
	Priority  NotificationPriority `json:"priority"`
	Title     string              `json:"title"`
	Content   string              `json:"content,omitempty"`
	Link      string              `json:"link,omitempty"`
	Sender    string              `json:"sender"`
	IsRead    bool                `json:"is_read"`
	ReadAt    *time.Time          `json:"read_at,omitempty"`
	CreatedAt time.Time           `json:"created_at"`
}

// NotificationFilter specifies criteria for listing notifications.
type NotificationFilter struct {
	UserID string               // required: always filter by user
	Type   *NotificationType    // optional: filter by type
	IsRead *bool                // optional: filter by read status
	model.ListRequest
}

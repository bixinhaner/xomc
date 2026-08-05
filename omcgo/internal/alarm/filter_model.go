package alarm

import (
	"time"
	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
)

// 过滤动作常量
const (
	FilterActionDefault         = "default"
	FilterActionIgnore          = "ignore"
	FilterActionAutoAcknowledge = "auto_acknowledge"
	FilterActionAutoClear       = "auto_clear"
	// FilterActionNotifyWebhook W1.5 冒烟：匹配规则即向 WebhookURL 发送 HTTP POST。
	// W2.A.2/T-0011 已扩展 retry/dead-letter/HMAC + FilterEngine 接生产路径。
	FilterActionNotifyWebhook = "notify_webhook"
	// FilterActionNotifyEmail W2.A.1/T-0007 整合：匹配规则向 EmailRecipients 列表发邮件。
	// 迁移期与通知中心共享 appconfig SMTP transport；Task 12 切换后移除旧入口。
	FilterActionNotifyEmail = "notify_email"
)

// 过滤类型常量
const (
	FilterTypeAlarmSource = "alarm_source"
	FilterTypeAlarmIdentifier   = "alarm_identifier"
	FilterTypeDeviceGroup = "device_group"
	FilterTypeDevice      = "device"
)

// AlarmFilterRule 告警过滤规则（数据库模型）
type AlarmFilterRule struct {
	ID              uuid.UUID   `json:"id" db:"id"`
	Name            string      `json:"name" db:"name"`
	FilterType      string      `json:"filter_type" db:"filter_type"`
	AlarmSources    []string    `json:"alarm_sources" db:"alarm_sources"`
	AlarmIdentifiers      []string    `json:"alarm_identifiers" db:"alarm_identifiers"`
	DeviceIDs       []uuid.UUID `json:"device_ids" db:"device_ids"`
	DeviceGroupIDs  []uuid.UUID `json:"device_group_ids" db:"device_group_ids"`
	Action          string      `json:"action" db:"action"`
	AcknowledgeDesc string      `json:"acknowledge_desc,omitempty" db:"acknowledge_desc"`
	WebhookURL       *string     `json:"webhook_url,omitempty" db:"webhook_url"`
	WebhookSecret    *string     `json:"webhook_secret,omitempty" db:"webhook_secret"`
	EmailRecipients  []string    `json:"email_recipients,omitempty" db:"email_recipients"`
	Priority         int         `json:"priority" db:"priority"`
	Enabled         bool        `json:"enabled" db:"enabled"`
	CreatedBy       string      `json:"created_by,omitempty" db:"created_by"`
	CreatedAt       time.Time   `json:"created_at" db:"created_at"`
	UpdatedBy       string      `json:"updated_by,omitempty" db:"updated_by"`
	UpdatedAt       time.Time   `json:"updated_at" db:"updated_at"`
}

// AlarmFilterRuleFilter 过滤规则查询过滤
type AlarmFilterRuleFilter struct {
	FilterTypes []string
	Action     *string
	Enabled    *bool
	Keyword    *string
	model.ListRequest
}

// CreateAlarmFilterRuleRequest 创建过滤规则请求
type CreateAlarmFilterRuleRequest struct {
	Name            string      `json:"name" binding:"required"`
	FilterType      string      `json:"filter_type" binding:"required"`
	AlarmSources    []string    `json:"alarm_sources"`
	AlarmIdentifiers      []string    `json:"alarm_identifiers"`
	DeviceIDs       []uuid.UUID `json:"device_ids"`
	DeviceGroupIDs  []uuid.UUID `json:"device_group_ids"`
	Action           string      `json:"action" binding:"required,oneof=default ignore auto_acknowledge auto_clear notify_webhook notify_email"`
	AcknowledgeDesc  string      `json:"acknowledge_desc"`
	WebhookURL       *string     `json:"webhook_url"`
	WebhookSecret    *string     `json:"webhook_secret"`
	EmailRecipients  []string    `json:"email_recipients"`
	Priority         int         `json:"priority"`
	Enabled          *bool       `json:"enabled"`
}

// UpdateAlarmFilterRuleRequest 更新过滤规则请求
type UpdateAlarmFilterRuleRequest struct {
	Name             *string      `json:"name"`
	FilterType       *string      `json:"filter_type"`
	AlarmSources     []string     `json:"alarm_sources"`
	AlarmIdentifiers []string     `json:"alarm_identifiers"`
	DeviceIDs        []uuid.UUID  `json:"device_ids"`
	DeviceGroupIDs   []uuid.UUID  `json:"device_group_ids"`
	Action           *string      `json:"action" binding:"omitempty,oneof=default ignore auto_acknowledge auto_clear notify_webhook notify_email"`
	AcknowledgeDesc  *string      `json:"acknowledge_desc"`
	WebhookURL       *string      `json:"webhook_url"`
	WebhookSecret    *string      `json:"webhook_secret"`
	EmailRecipients  []string     `json:"email_recipients"`
	Priority         *int         `json:"priority"`
	Enabled          *bool        `json:"enabled"`
}

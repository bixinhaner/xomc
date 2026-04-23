package alarm

import (
	"time"
	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
)

// 过滤动作常量
const (
	FilterActionDefault        = "default"
	FilterActionIgnore         = "ignore"
	FilterActionAutoAcknowledge = "auto_acknowledge"
	FilterActionAutoClear      = "auto_clear"
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
	Priority        int         `json:"priority" db:"priority"`
	Enabled         bool        `json:"enabled" db:"enabled"`
	CreatedBy       string      `json:"created_by,omitempty" db:"created_by"`
	CreatedAt       time.Time   `json:"created_at" db:"created_at"`
	UpdatedBy       string      `json:"updated_by,omitempty" db:"updated_by"`
	UpdatedAt       time.Time   `json:"updated_at" db:"updated_at"`
}

// AlarmFilterRuleFilter 过滤规则查询过滤
type AlarmFilterRuleFilter struct {
	FilterType *string
	Action     *string
	Enabled    *bool
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
	Action          string      `json:"action" binding:"required,oneof=default ignore auto_acknowledge auto_clear"`
	AcknowledgeDesc string      `json:"acknowledge_desc"`
	Priority        int         `json:"priority"`
	Enabled         *bool       `json:"enabled"`
}

// UpdateAlarmFilterRuleRequest 更新过滤规则请求
type UpdateAlarmFilterRuleRequest struct {
	Name            *string      `json:"name"`
	FilterType      *string      `json:"filter_type"`
	AlarmSources    []string     `json:"alarm_sources"`
	AlarmIdentifiers      []string     `json:"alarm_identifiers"`
	DeviceIDs       []uuid.UUID  `json:"device_ids"`
	DeviceGroupIDs  []uuid.UUID  `json:"device_group_ids"`
	Action          *string      `json:"action" binding:"omitempty,oneof=default ignore auto_acknowledge auto_clear"`
	AcknowledgeDesc *string      `json:"acknowledge_desc"`
	Priority        *int         `json:"priority"`
	Enabled         *bool        `json:"enabled"`
}
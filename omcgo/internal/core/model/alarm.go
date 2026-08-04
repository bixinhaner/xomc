package model

import (
	"time"

	"github.com/google/uuid"
)

// Alarm 表示被管设备上报的一条告警。
// 对应数据库 alarms 表，由 alarm.AlarmEngine 写入。
// 告警生命周期：活跃 → 确认 → 清除，对应 AlarmStatus。
type Alarm struct {
	ID              uuid.UUID     `json:"id" db:"id"`
	Version         int64         `json:"alarm_version" db:"alarm_version"`
	DeviceID        uuid.UUID     `json:"device_id" db:"device_id"`
	DeviceSN        string        `json:"device_sn" db:"device_sn"`
	Carrier         CarrierCode   `json:"carrier" db:"carrier"`
	Severity        AlarmSeverity `json:"severity" db:"severity"`
	AlarmType       string        `json:"alarm_type" db:"alarm_type"`
	AlarmIdentifier string        `json:"alarm_identifier" db:"alarm_identifier"`
	Description     string        `json:"description" db:"description"`
	Status          AlarmStatus   `json:"status" db:"status"`
	RaisedAt        time.Time     `json:"raised_at" db:"raised_at"`
	AcknowledgedAt  *time.Time    `json:"acknowledged_at,omitempty" db:"acknowledged_at"`
	ClearedAt       *time.Time    `json:"cleared_at,omitempty" db:"cleared_at"`
	// 增强字段 (v2.0)
	DeviceName      *string           `json:"device_name,omitempty" db:"device_name"`
	Technology      *string           `json:"technology,omitempty" db:"technology"`
	AlarmSource     *string           `json:"alarm_source,omitempty" db:"alarm_source"`
	EventType       *string           `json:"event_type,omitempty" db:"event_type"`
	NetworkLocation *string           `json:"network_location,omitempty" db:"network_location"`
	ExplicitCause   *string           `json:"explicit_cause,omitempty" db:"explicit_cause"`
	IsRead          bool              `json:"is_read" db:"is_read"`
	AckCount        int               `json:"ack_count" db:"ack_count"`
	FirstRaisedAt   time.Time         `json:"first_raised_at,omitempty" db:"first_raised_at"`
	LastUpdatedAt   time.Time         `json:"last_updated_at,omitempty" db:"last_updated_at"`
	ProbableCause   *string           `json:"probable_cause,omitempty" db:"probable_cause"`
	AcknowledgedBy  *string           `json:"acknowledged_by,omitempty" db:"acknowledged_by"`
	AckNote         *string           `json:"ack_note,omitempty" db:"ack_note"`
	ClearedBy       *string           `json:"cleared_by,omitempty" db:"cleared_by"`
	ClearNote       *string           `json:"clear_note,omitempty" db:"clear_note"`
	AdditionalInfo  map[string]string `json:"additional_info,omitempty" db:"additional_info"`
	// IsUnknown 标记 T-0098 P2-10 fallback 写入的告警（identifier 不在 alarm_definitions
	// 但 product.enable_unknown_alarm=true）。dashboard / 治理闭环依据此过滤。
	IsUnknown bool      `json:"is_unknown" db:"is_unknown"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

package eventlog

import (
	"time"

	"github.com/google/uuid"
)

// 事件类型常量。本期 MVP 只接 1 BOOT 普通重启（type=boot），其他类型留扩展。
const (
	EventTypeBoot = "boot"
)

// 事件级别（与前端 mock 设计对齐：info / warning / error / success）。
const (
	EventLevelInfo    = "info"
	EventLevelWarning = "warning"
	EventLevelError   = "error"
	EventLevelSuccess = "success"
)

// EventLog 对应 event_logs 表一行。
type EventLog struct {
	ID              uuid.UUID  `json:"id"`
	DeviceID        *uuid.UUID `json:"device_id,omitempty"`
	DeviceSN        string     `json:"device_sn"`
	DeviceName      string     `json:"device_name,omitempty"`
	DeviceType      string     `json:"device_type,omitempty"`
	IsGNB           bool       `json:"is_gnb"`
	OperateIP       string     `json:"operate_ip,omitempty"`
	SoftwareVersion string     `json:"software_version,omitempty"`
	EventType       string     `json:"event_type"`
	EventReason     string     `json:"event_reason,omitempty"`
	EventLevel      string     `json:"event_level"`
	EventData       []byte     `json:"event_data,omitempty"` // 原始 JSONB
	OccurredAt      time.Time  `json:"occurred_at"`
	CreatedAt       time.Time  `json:"created_at"`
}

// Filter 查询过滤条件
type Filter struct {
	DeviceSN   string
	EventType  string
	StartTime  *time.Time
	EndTime    *time.Time
	Page       int
	PageSize   int
}

func (f Filter) Offset() int {
	if f.Page <= 1 {
		return 0
	}
	return (f.Page - 1) * f.PageSize
}

func (f Filter) Limit() int {
	if f.PageSize <= 0 {
		return 20
	}
	return f.PageSize
}

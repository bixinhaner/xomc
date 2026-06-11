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

// DeviceRebootStat 按设备聚合的重启次数统计（event_logs GROUP BY device_sn）。
// LatestAt 为该设备最近一次事件时间，可能为空（理论上聚合后必有值，留指针以防边界）。
type DeviceRebootStat struct {
	DeviceSN    string     `json:"device_sn"`
	DeviceName  string     `json:"device_name,omitempty"`
	RebootCount int64      `json:"reboot_count"`
	LatestAt    *time.Time `json:"latest_at,omitempty"`
}

// Filter 查询过滤条件
type Filter struct {
	DeviceSN  string
	EventType string
	StartTime *time.Time
	EndTime   *time.Time
	Page      int
	PageSize  int
	// VisibleGroups 是 #63 设备组可见性强制层注入的可见分组集合（三态契约见 authz 包）：
	// nil=超管不过滤，[]=无权限空集 fail-closed，[ids]=限定。仓库层按 device_id 列收窄。
	VisibleGroups []uuid.UUID
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

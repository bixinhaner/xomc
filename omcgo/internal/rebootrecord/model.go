package rebootrecord

import (
	"time"

	"github.com/google/uuid"
)

// RebootType 重启类型过滤维度。
//
// 统一重启记录由两张互斥的表合成：
//   - event_logs          普通 1 BOOT（无 HaltReason）
//   - station_fault_logs  异常 1 BOOT（带 HaltReason）
//
// 两表不重叠，UNION ALL 即得全部重启，无需去重。
type RebootType string

const (
	RebootTypeAll      RebootType = ""         // 全部（两表合并）
	RebootTypeNormal   RebootType = "normal"   // 仅正常（event_logs）
	RebootTypeAbnormal RebootType = "abnormal" // 仅异常（station_fault_logs）
)

// ParseRebootType 把查询参数归一化为 RebootType（未知值回退为全部）。
func ParseRebootType(s string) RebootType {
	switch s {
	case string(RebootTypeNormal):
		return RebootTypeNormal
	case string(RebootTypeAbnormal):
		return RebootTypeAbnormal
	default:
		return RebootTypeAll
	}
}

// RebootRecord 统一重启记录的一行（event_logs ∪ station_fault_logs）。
//
// 异常专属字段（DetailReason / RuntimeBeforeReboot）在普通重启行为空 / 0。
type RebootRecord struct {
	ID                  string    `json:"id"`     // 原表 uuid
	Source              string    `json:"source"` // "event"（普通）| "fault"（异常）
	IsAbnormal          bool      `json:"is_abnormal"`
	DeviceSN            string    `json:"device_sn"`
	DeviceName          string    `json:"device_name,omitempty"`
	DeviceType          string    `json:"device_type,omitempty"` // eNB / gNB / GSM / UPS
	OperateIP           string    `json:"operate_ip,omitempty"`
	SoftwareVersion     string    `json:"software_version,omitempty"`
	Reason              string    `json:"reason,omitempty"`        // 普通: event_reason；异常: fault_reason(HaltMainReason)
	DetailReason        string    `json:"detail_reason,omitempty"` // 仅异常: fault_detail(HaltDetailReason)
	RuntimeBeforeReboot int64     `json:"runtime_before_reboot"`   // 重启前设备运行时长（秒）；普通和异常重启均有
	RebootTime          time.Time `json:"reboot_time"`             // 普通: occurred_at；异常: detected created_at
}

// Filter 统一查询过滤条件。
type Filter struct {
	DeviceSN   string
	DeviceType string
	RebootType RebootType
	StartTime  *time.Time
	EndTime    *time.Time
	Page       int
	PageSize   int
	// VisibleGroups 是 #63 设备组可见性强制层注入的可见分组集合（三态契约见 authz 包）：
	// nil=超管不过滤，[]=无权限空集 fail-closed，[ids]=限定。UNION 两半各按 device_id
	// 关联 device_group_members 收窄（device_id 为 NULL 的记录对非超管不可见）。
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

// DeviceRebootStat 按设备聚合：总重启次数 + 其中异常次数（跟随过滤条件）。
type DeviceRebootStat struct {
	DeviceSN      string     `json:"device_sn"`
	DeviceName    string     `json:"device_name,omitempty"`
	DeviceType    string     `json:"device_type,omitempty"`
	TotalCount    int64      `json:"total_count"`
	AbnormalCount int64      `json:"abnormal_count"`
	LatestAt      *time.Time `json:"latest_at,omitempty"`
}

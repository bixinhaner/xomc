package device

import (
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
)

// DeviceWithInfo 组合设备核心信息与扩展信息，用于列表展示。
// 字段来自 devices 表 LEFT JOIN device_info 表 LEFT JOIN device_groups 表。
// 前端无需感知多表架构，直接使用扁平化结构。
// device_info 和 group 字段可能为空（设备刚注册，尚未同步参数或未分配分组）。
type DeviceWithInfo struct {
	// 嵌入 devices 表核心字段
	model.Device

	// ===== 设备分组信息（可能为空）=====

	// GroupName 设备分组名称
	GroupName *string `json:"group_name"`

	// ===== device_info 运维标识（可能为空）=====

	// DeviceName 设备名称
	DeviceName *string `json:"device_name"`

	// InfoAddress 安装地址（原 InfoAddress，改名消除歧义）
	InfoAddress *string `json:"device_address"`

	// Remark 备注
	Remark *string `json:"remark"`

	// ProjectStatus 项目状态
	ProjectStatus *string `json:"project_status"`

	// Height 安装高度（米）
	Height *float64 `json:"height"`

	// ===== device_info 无线参数（可能为空）=====

	// ECI E-UTRAN 小区标识符
	ECI *string `json:"eci"`

	// PCI 物理小区标识
	PCI *string `json:"pci"`

	// CellID 逻辑小区 ID
	CellID *string `json:"cell_id"`

	// FreqPoint 频点号
	FreqPoint *string `json:"freq_point"`

	// Bandwidth 载波带宽 (MHz)
	Bandwidth *float64 `json:"bandwidth"`

	// TransmitPower 发射功率 (dBm)
	TransmitPower *float64 `json:"transmit_power"`

	// PLMN 公众陆地移动网标识
	PLMN *string `json:"plmn"`

	// ===== device_info 状态字段（可能为空）=====

	// RFStatus 射频状态
	RFStatus *string `json:"rf_status"`

	// CellStatus 小区状态
	CellStatus *string `json:"cell_status"`

	// MMEStatus MME 连接状态
	MMEStatus *string `json:"mme_status"`

	// SyncStatus 时钟同步源状态
	SyncStatus *string `json:"sync_status"`

	// KPIStatus KPI 状态
	KPIStatus *string `json:"kpi_status"`

	// NumOfCells 载波数量
	NumOfCells *int `json:"num_of_cells"`

	// GPSStatus GPS 状态
	GPSStatus *string `json:"gps_status"`

	// AlarmSeverity 最高告警级别
	AlarmSeverity *string `json:"alarm_severity"`

	// LicenseStatus 许可证状态
	LicenseStatus *string `json:"license_status"`

	// ===== device_info 硬件信息（可能为空）=====

	// MAC MAC 地址
	MAC *string `json:"mac"`

	// HardwareVersion 硬件版本
	HardwareVersion *string `json:"hardware_version"`

	// ===== device_info 时间信息（可能为空）=====

	// FirstOnlineTime 首次上线时间（设备生命周期内只记录一次）
	FirstOnlineTime *time.Time `json:"first_online_time"`

	// LastOnlineTime 最后上线时间（设备从离线变为在线的时间，对应前端"接入时间"）
	LastOnlineTime *time.Time `json:"last_online_time"`

	// LastOfflineTime 最后离线时间（设备从在线变为离线的时间，对应前端"断开时间"）
	LastOfflineTime *time.Time `json:"last_offline_time"`

	// RunTime 累计运行时长（秒）
	RunTime *int64 `json:"run_time"`

	// ===== 离线时长（动态计算，不存储）=====

	// OfflineSeconds 离线总秒数（仅离线设备有值）
	OfflineSeconds *int64 `json:"offline_seconds,omitempty"`

	// OfflineDays 离线天数
	OfflineDays *int64 `json:"offline_days,omitempty"`

	// OfflineHours 剩余小时数（0-23）
	OfflineHours *int64 `json:"offline_hours,omitempty"`

	// OfflineMinutes 剩余分钟数（0-59）
	OfflineMinutes *int64 `json:"offline_minutes,omitempty"`
}

// DeviceListItem 简化的设备列表项，用于前端表格展示。
// 相比 DeviceWithInfo，移除了不常用的字段，减少数据传输量。
type DeviceListItem struct {
	ID             uuid.UUID          `json:"id"`
	SerialNumber   string             `json:"serial_number"`
	DeviceName     *string            `json:"device_name"`
	Status         model.DeviceStatus `json:"status"`
	Carrier        model.CarrierCode  `json:"carrier"`
	Technology     model.Technology   `json:"technology"`
	Manufacturer   string             `json:"manufacturer"`
	ModelName      string             `json:"model_name"`
	FirmwareVersion string            `json:"firmware_version"`
	IPAddress      string             `json:"ip_address"`
	SiteName       string             `json:"site_name"`
	LastInformAt   *time.Time         `json:"last_inform_at"`
	RFStatus       *string            `json:"rf_status"`
	AlarmSeverity  *string            `json:"alarm_severity"`
	GroupName      *string            `json:"group_name"`
	CreatedAt      time.Time          `json:"created_at"`
}

// DeviceSummary 设备汇总统计，用于仪表盘展示。
type DeviceSummary struct {
	// Total 总设备数
	Total int64 `json:"total"`

	// ActiveCount 活跃设备数
	ActiveCount int64 `json:"active_count"`

	// OfflineCount 离线设备数
	OfflineCount int64 `json:"offline_count"`

	// AlarmCount 有告警的设备数
	AlarmCount int64 `json:"alarm_count"`

	// ByCarrier 按运营商统计
	ByCarrier map[string]int64 `json:"by_carrier"`

	// ByTechnology 按制式统计
	ByTechnology map[string]int64 `json:"by_technology"`
}

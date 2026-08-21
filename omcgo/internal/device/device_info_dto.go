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

	// LocationSync combines the accepted device coordinates with the latest
	// valid device-reported observation for list/detail reconciliation.
	LocationSync *LocationSync `json:"location_sync,omitempty"`

	// ControlSummary is the current OMC-owned control projection. It is nil
	// when no geofence control action currently owns device deactivation.
	ControlSummary *DeviceControlSummary `json:"control_summary"`

	// DeviceType is the list tab dimension. UPS is identified by ProductClass
	// prefix only; other existing devices stay in the base-station tab.
	DeviceType string `json:"device_type"`

	// UPSSummary contains list-level UPS power and battery status fields.
	UPSSummary *UPSRuntimeSummary `json:"ups_summary,omitempty"`

	// ===== 设备分组信息（可能为空）=====

	// GroupID 设备分组 ID
	GroupID *uuid.UUID `json:"group_id,omitempty"`

	// GroupName 设备分组名称
	GroupName *string `json:"group_name"`

	// SourceType 分组归属来源。无真实归属行的默认组视图由查询层派生为 auto。
	SourceType *string `json:"source_type,omitempty"`

	// ===== device_info 运维标识（可能为空）=====

	// InfoDeviceName device_info 表的运维设备名。
	// 注意与嵌入的 model.Device.DeviceName（devices 表，json device_name）区分：
	// 那个是设备名称主字段；本字段是 device_info 运维扩展里另填的名字。
	InfoDeviceName *string `json:"info_device_name"`

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

	// MMEPool 当前页 LTE 设备的 MME 池明细，由 device_parameters 批量装配。
	MMEPool []MMEEntry `json:"mme_pool,omitempty"`

	// SyncStatus 时钟同步源状态
	SyncStatus *string `json:"sync_status"`

	// KPIStatus KPI 状态
	KPIStatus *string `json:"kpi_status"`

	// NumOfCells 载波数量
	NumOfCells *int `json:"num_of_cells"`

	// UECount 当前接入 UE 数，由 device_info.ue_count 透传到设备列表。
	UECount int `json:"ue_count"`

	// GPSStatus GPS 状态
	GPSStatus *string `json:"gps_status"`

	// AlarmSeverity 最高告警级别
	// #361: 来自 alarms_active 实时聚合（未 cleared 活动告警 MIN(severity) 映成
	// critical/major/minor/warning 文本），无活动告警 → nil → 前端归 'none'。
	AlarmSeverity *string `json:"alarm_severity"`

	// ActiveAlarmCount 该设备未 cleared 活动告警数（#361）；无活动告警 → nil → 前端归 0。
	ActiveAlarmCount *int `json:"active_alarm_count"`

	// ParamSyncRunning 当前设备是否存在非终态的参数同步请求或运行记录。
	// 用于设备列表显示“参数同步中”的动态状态图标。
	ParamSyncRunning bool `json:"param_sync_running"`

	// LicenseStatus 许可证状态
	LicenseStatus *string `json:"license_status"`

	// ===== device_info 硬件信息（可能为空）=====

	// MAC MAC 地址
	MAC *string `json:"mac"`

	// HardwareVersion 硬件版本
	HardwareVersion *string `json:"hardware_version"`

	// ===== Phase 2/3 (设计文档 §4.2 Layer E)：扩展基础信息列 =====
	// 与 model.DeviceInfo 同名字段保持一致;list 接口 SELECT 这些列后扁平
	// 化暴露给前端,无需走 /devices/:id/detail composite 接口。

	TAC                *string  `json:"tac"`
	Lac                *string  `json:"lac,omitempty"`
	Band               *string  `json:"band"`
	ULEarfcn           *string  `json:"ul_earfcn"`
	SubframeAssignment *string  `json:"subframe_assignment"`
	SpecialSubframe    *string  `json:"special_subframe"`
	RootIndex          *string  `json:"root_index"`
	GPSSatellites      *int     `json:"gps_satellites"`
	GPSHeight          *float64 `json:"gps_height"`
	LockStatus         *string  `json:"lock_status"`
	// AdminState NR 管理状态（NR.RAN.Common.AdminState 1/2/3 → Locked/Unlocked/ShuttingDown）。
	// 前端 device list 'Admin State' 列；LTE 设备为 NULL。
	AdminState *string `json:"admin_state"`
	// IpsecAddr Device.DeviceInfo.SERVING_UNIT1_IPSEC_Address，前端 'IPSec 地址' 列。
	IpsecAddr    *string `json:"ipsec_addr"`
	EnbID        *string `json:"enb_id"`
	NetworkModel *string `json:"network_model"`

	// ===== GSM/BTS 专属（migration 000003，DeviceGSM.* TR069 路径同步）=====
	// 前端 mapBackendDevice 早已声明 bd.bsc_select / bd.oml_remote_ip 等并消费，
	// 此前 device_info 物理列缺失 → 后端始终回 NULL → 详情页恒 "-"；
	// 000003 引入物理列后由这 4 个字段透传到前端列表/详情。

	BscSelect      *string `json:"bsc_select,omitempty"`
	OmlRemoteIp    *string `json:"oml_remote_ip,omitempty"`
	OmlRemoteIpBak *string `json:"oml_remote_ip_bak,omitempty"`
	IpaUnitId      *string `json:"ipa_unit_id,omitempty"`

	// BscLinkStatus 由 SELECT 派生（oml_remote_ip 非空 + d.is_online → "connected"
	// 否则 "disconnected"），不对应物理列。前端 Tag 映射 connected/disconnected。
	BscLinkStatus *string `json:"bsc_link_status,omitempty"`

	// ===== 设备名称同步（Issue #758）=====

	// NameSyncPending 设备名称同步待处理标记。
	// true = LMT 名称与网管名称不一致且 prompt=true，需人工确认（前端显示小红点）。
	NameSyncPending *bool `json:"name_sync_pending,omitempty"`

	// LMTDeviceName 从 LMT 读取的设备名称（HNBName / gNBName）缓存。
	// 供前端在名称冲突时对比展示"LMT 名称"与"网管名称"。
	LMTDeviceName *string `json:"lmt_device_name,omitempty"`

	// ===== device_info 时间信息（可能为空）=====

	// FirstOnlineTime 首次上线时间（设备生命周期内只记录一次）
	FirstOnlineTime *time.Time `json:"first_online_time"`

	// LastOnlineTime 最后上线时间（设备从离线变为在线的时间，对应前端"接入时间"）
	LastOnlineTime *time.Time `json:"last_online_time"`

	// LastOfflineTime 最后离线时间（设备从在线变为离线的时间，对应前端"断开时间"）
	LastOfflineTime *time.Time `json:"last_offline_time"`

	// RunTime 设备自报运行时长（秒，来自 TR-181 Device.DeviceInfo.UpTime）。
	// 与"OMC 视角累计在线时长"（CumulativeOnlineDuration）含义不同。
	RunTime *int64 `json:"run_time"`

	// CumulativeOnlineDuration 累计在线总时长（秒）。
	// 由 DeviceStatusReconciler 在 online→offline 边沿事务性累加。
	// 前端"总在线时长"应用 = cumulative_online_duration + (is_online ? NOW - last_online_time : 0)。
	CumulativeOnlineDuration *int64 `json:"cumulative_online_duration,omitempty"`

	// ===== 在线时长（动态计算，不存储）=====
	//
	// OnlineDuration 当前在线/最近在线区间秒数（SQL 派生列）：
	//   - is_online=TRUE 时：NOW() - last_online_time
	//   - is_online=FALSE 且 last_offline_time > last_online_time 时：last_offline_time - last_online_time
	//   - last_online_time IS NULL 时：null
	// 用于前端「累计时长」UI 字段（与 RunTime「设备本次开机时长」语义区分）。
	OnlineDuration *int64 `json:"online_duration,omitempty"`

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

// UPSRuntimeSummary is the UPS list/detail read model projected from TR-069
// parameters. Raw values remain in device_parameters and the parameter model.
type UPSRuntimeSummary struct {
	ExternalIP       *string    `json:"external_ip,omitempty"`
	TotalVoltage     *string    `json:"total_voltage,omitempty"`
	TotalTemperature *string    `json:"total_temperature,omitempty"`
	TotalCurrent     *string    `json:"total_current,omitempty"`
	SoftwareVersion  *string    `json:"software_version,omitempty"`
	HardwareVersion  *string    `json:"hardware_version,omitempty"`
	Manufacturer     *string    `json:"manufacturer,omitempty"`
	ManufacturerOUI  *string    `json:"manufacturer_oui,omitempty"`
	UpTimeSeconds    *int64     `json:"up_time_seconds,omitempty"`
	BMSCharging      *string    `json:"bms_charging,omitempty"`
	ACPower          *string    `json:"ac_power,omitempty"`
	ACVoltage        *string    `json:"ac_voltage,omitempty"`
	DCVoltage        *string    `json:"dc_voltage,omitempty"`
	DCCurrent        *string    `json:"dc_current,omitempty"`
	BoardTemperature *string    `json:"board_temperature,omitempty"`
	SFPState         *string    `json:"sfp_state,omitempty"`
	Port0State       *string    `json:"port0_state,omitempty"`
	Port1State       *string    `json:"port1_state,omitempty"`
	Port2State       *string    `json:"port2_state,omitempty"`
	Port3State       *string    `json:"port3_state,omitempty"`
	AverageSOC       *int       `json:"average_soc,omitempty"`
	PackCounts       *int       `json:"pack_counts,omitempty"`
	LastInformAt     *time.Time `json:"last_inform_at,omitempty"`
}

// DeviceListItem 简化的设备列表项，用于前端表格展示。
// 相比 DeviceWithInfo，移除了不常用的字段，减少数据传输量。
type DeviceListItem struct {
	ID              uuid.UUID          `json:"id"`
	SerialNumber    string             `json:"serial_number"`
	InfoDeviceName  *string            `json:"info_device_name"` // device_info 运维设备名
	Status          model.DeviceStatus `json:"status"`
	Carrier         model.CarrierCode  `json:"carrier"`
	Technology      model.Technology   `json:"technology"`
	Manufacturer    string             `json:"manufacturer"`
	ModelName       string             `json:"model_name"`
	FirmwareVersion string             `json:"firmware_version"`
	IPAddress       string             `json:"ip_address"`
	DeviceName      string             `json:"device_name"` // devices 表设备名称（原 SiteName）
	LastInformAt    *time.Time         `json:"last_inform_at"`
	RFStatus        *string            `json:"rf_status"`
	AlarmSeverity   *string            `json:"alarm_severity"`
	GroupName       *string            `json:"group_name"`
	CreatedAt       time.Time          `json:"created_at"`
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

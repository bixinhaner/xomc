package device

import (
	"time"

	"github.com/google/uuid"
)

// DeviceInfo 存储设备的扩展运维信息。
// 对应数据库 device_info 表，与 devices 表一对一关系。
// 字段分为三类：
//   - 运维标识（手动填写）：DeviceName, Address, Remark, ProjectStatus, Height
//   - 无线参数（TR069 同步）：ECI, PCI, CellID, FreqPoint, Bandwidth, TransmitPower, PLMN
//   - 状态字段（TR069 计算）：RFStatus, CellStatus, MMEStatus, SyncStatus, GPSStatus, LicenseStatus
type DeviceInfo struct {
	// DeviceID 关联 devices.id，主键
	DeviceID uuid.UUID `json:"device_id"`

	// ===== 运维标识（操作员手动填写）=====

	// DeviceName 设备名称，操作员自定义的易读名称
	DeviceName string `json:"device_name"`

	// Address 安装地址，详细地址描述
	Address string `json:"address"`

	// Remark 备注信息
	Remark string `json:"remark"`

	// ProjectStatus 项目状态，运营商自定义状态标识
	// 示例值: "在建", "调试", "商用", "退网"
	ProjectStatus string `json:"project_status"`

	// Height 安装高度（米）
	Height *float64 `json:"height"`

	// ===== 无线参数（从 TR069 参数自动同步）=====

	// ECI E-UTRAN 小区标识符 (E-UTRAN Cell ID)
	// 格式: PLMN(6位) + CellID(22位)，共 28 位
	ECI string `json:"eci"`

	// PCI 物理小区标识 (Physical Cell ID)
	// 范围: LTE 0-503, NR 0-1007
	PCI string `json:"pci"`

	// CellID 逻辑小区 ID
	CellID string `json:"cell_id"`

	// FreqPoint 频点号
	// LTE: EARFCN, NR: NRARFCN
	FreqPoint string `json:"freq_point"`

	// Bandwidth 载波带宽 (MHz)
	Bandwidth *float64 `json:"bandwidth"`

	// TransmitPower 发射功率 (dBm)
	TransmitPower *float64 `json:"transmit_power"`

	// PLMN 公众陆地移动网标识
	// 格式: MCC(3位) + MNC(2-3位)，如 "46000"
	PLMN string `json:"plmn"`

	// ===== 状态字段（从 TR069 参数计算）=====

	// RFStatus 射频状态
	// 可能值: RFStatusOn("on"), RFStatusOff("off"), RFStatusError("error")
	// 计算逻辑: CalcRFStatus()
	RFStatus string `json:"rf_status"`

	// CellStatus 小区状态
	// 可能值: CellStatusNormal, CellStatusInactive, CellStatusFault, CellStatusDecommissioned
	// 计算逻辑: CalcCellStatus()
	CellStatus string `json:"cell_status"`

	// OpState 激活状态(前端展示用,'1'=激活 / '0'=未激活)。
	// 派生口径同 CellStatus —— 任一 cell active(FAPControl/CellConfig 下
	// {LTE|NR}.OpState/CellOpState 任一 trpath 为 "1"/"true"/"active")则
	// 设备激活,否则未激活。由 InfoSyncer.CalcOpState 在每次参数同步时刷新。
	// 取代历史 model.DeriveOpStateActivated(first_online_time) 的一次性持久语义。
	OpState string `json:"op_state"`

	// MMEStatus MME/AMF 连接状态
	// 可能值: MMEStatusConnected, MMEStatusPartial, MMEStatusDisconnected
	// 计算逻辑: CalcMMEStatus()
	MMEStatus string `json:"mme_status"`

	// SyncStatus 时钟同步源状态
	// 可能值: SyncStatusGPS, SyncStatusBeidou, SyncStatusNTP, SyncStatusError；
	// 对于设备直接上报的文本态（如 LOCKED / HOLDOVER / SYNCED），保留原始值。
	// 计算逻辑: CalcSyncStatus()
	SyncStatus string `json:"sync_status"`

	// KPIStatus KPI 状态（保留字段）
	KPIStatus string `json:"kpi_status"`

	// NumOfCells 载波数量
	// 通常 1-4 个载波
	NumOfCells int `json:"num_of_cells"`

	// GPSStatus GPS 定位状态
	// 可能值: GPSStatusNormal, GPSStatusAbnormal, GPSStatusNoSignal
	// 计算逻辑: CalcGPSStatus()
	GPSStatus string `json:"gps_status"`

	// AlarmSeverity 当前最高告警级别
	// 可能值: "critical", "major", "minor", "warning", ""
	AlarmSeverity string `json:"alarm_severity"`

	// LicenseStatus 许可证状态
	// 可能值: LicenseStatusActive, LicenseStatusExpiring, LicenseStatusExpired
	// 计算逻辑: CalcLicenseStatus()
	LicenseStatus string `json:"license_status"`

	// ===== 硬件信息 =====

	// MAC MAC 地址
	MAC string `json:"mac"`

	// HardwareVersion 硬件版本号
	HardwareVersion string `json:"hardware_version"`

	// ===== 时间信息 =====

	// FirstOnlineTime 首次上线时间（设备生命周期内只记录一次）
	FirstOnlineTime *time.Time `json:"first_online_time"`

	// LastOnlineTime 最后上线时间（设备从离线变为在线的时间）
	LastOnlineTime *time.Time `json:"last_online_time"`

	// LastOfflineTime 最后离线时间（设备从在线变为离线的时间）
	LastOfflineTime *time.Time `json:"last_offline_time"`

	// RunTime 设备自报运行时长（秒），来自 TR-181 Device.DeviceInfo.UpTime；
	// 由 InfoSyncer.SyncFromParameters 每次参数同步刷新。
	// 注意：这是"设备从上次本地重启算起"的时长，与 OMC 视角的累计在线时长（CumulativeOnlineDuration）含义不同。
	RunTime int64 `json:"run_time"`

	// CumulativeOnlineDuration 累计在线总时长（秒）。
	// 由 DeviceStatusReconciler 在 online→offline 边沿事务性累加
	// (NOW - last_online_time)。前端展示"总在线时长"应用:
	//   cumulative_online_duration + (is_online ? NOW - last_online_time : 0)
	CumulativeOnlineDuration int64 `json:"cumulative_online_duration"`

	// ===== Phase 2/3 (设计文档 §4.2)：扩展基础信息字段（指针 nullable）=====
	// 这些列在 migration 000181 加入；InfoSyncer 按 universalInformMapping
	// 和派生计算回填。前端 BasicTab「小区信息 / 状态信息 / 其他信息」分组消费。

	// TAC Tracking Area Code（TR-181 EPC.TAC）
	TAC *string `json:"tac,omitempty"`

	// Band LTE 频段编号（TR-181 RAN.RF.FreqBandIndicator）
	Band *string `json:"band,omitempty"`

	// ULEarfcn 上行频点号（TR-181 RAN.RF.EARFCNUL；与 FreqPoint 对称）
	ULEarfcn *string `json:"ul_earfcn,omitempty"`

	// SubframeAssignment TDD 帧配比（RAN.PHY.TDDFrame.SubFrameAssignment）
	SubframeAssignment *string `json:"subframe_assignment,omitempty"`

	// SpecialSubframe TDD 特殊子帧（RAN.PHY.TDDFrame.SpecialSubframePatterns）
	SpecialSubframe *string `json:"special_subframe,omitempty"`

	// RootIndex PRACH 根索引（RAN.PHY.PRACH.ZeroCorrelationZoneConfig）
	RootIndex *string `json:"root_index,omitempty"`

	// GPSSatellites GPS 卫星数（FAP.GPS.NumberOfSatellites）
	GPSSatellites *int `json:"gps_satellites,omitempty"`

	// GPSHeight GPS 高度（米；CPE 上报 path 优先 altidute → Altitude → Height）
	GPSHeight *float64 `json:"gps_height,omitempty"`

	// LockStatus 小区锁状态（FAPControl.LTE.AdminState："true"=unlocked/"false"=locked）
	LockStatus *string `json:"lock_status,omitempty"`

	// EnbID eNodeB ID（派生：ECI >> 8）
	EnbID *string `json:"enb_id,omitempty"`

	// NetworkModel "TDD" / "FDD"（派生：根据 PHY 子帧 path 是否存在）
	NetworkModel *string `json:"network_model,omitempty"`

	// AMFStatus 详情页展示别名。
	// NR 设备把 AMF 状态回写到统一的 mme_status 口径后，这里仅作为详情页别名透出，
	// 避免前端文案层必须感知 LTE/NR 共用字段名。
	AMFStatus string `json:"amf_status,omitempty"`

	// MultiPlmnEnable NR Multi PLMN 开关状态（详情页按参数现值投影，非 device_info 持久列）。
	// 取值规范化为 "enabled" / "disabled"。
	MultiPlmnEnable string `json:"multi_plmn_enable,omitempty"`

	// GPSVersion GPS 软件版本（详情页按 Device.FAP.GPS.SoftVersion 参数临时投影）。
	GPSVersion string `json:"gps_version,omitempty"`

	// PPSTimeMode PPS 授时模式（详情页按 Device.FAP.Synchronization.PpsTimeMode 参数临时投影）。
	PPSTimeMode string `json:"pps_time_mode,omitempty"`

	// RollbackVersion 备用系统/回滚版本（详情页按 Device.SoftwareCtrl.SystemBackupVersion 参数临时投影）。
	RollbackVersion string `json:"rollback_version,omitempty"`

	// WANStatus WAN 口链路状态（详情页按 Device.Ethernet.Interface.{i}.Status 参数临时投影）。
	WANStatus string `json:"wan_status,omitempty"`

	// ===== 审计字段 =====

	// Creator 创建人用户名
	Creator string `json:"creator"`

	// Updater 最后更新人用户名
	Updater string `json:"updater"`

	// CreatedAt 创建时间
	CreatedAt time.Time `json:"created_at"`

	// UpdatedAt 最后更新时间
	UpdatedAt time.Time `json:"updated_at"`
}

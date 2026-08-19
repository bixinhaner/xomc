package device

// ===== 射频状态 (RFStatus) =====

// RFStatus 表示设备射频发射状态。
// 从 TR069 参数 FAPControl.AdminState 和 RFTxStatus 计算。
type RFStatus string

const (
	// RFStatusOn 射频开启，正在发射
	RFStatusOn RFStatus = "on"
	// RFStatusOff 射频关闭（管理态禁用）
	RFStatusOff RFStatus = "off"
	// RFStatusError 射频故障
	RFStatusError RFStatus = "error"
)

// ValidRFStatuses 返回所有有效的射频状态
func ValidRFStatuses() []RFStatus {
	return []RFStatus{RFStatusOn, RFStatusOff, RFStatusError}
}

// IsValid 检查射频状态是否有效
func (s RFStatus) IsValid() bool {
	switch s {
	case RFStatusOn, RFStatusOff, RFStatusError:
		return true
	}
	return false
}

// String 返回字符串形式
func (s RFStatus) String() string {
	return string(s)
}

// ===== 小区状态 (CellStatus) =====

// CellStatus 表示小区服务状态。
// 从 TR069 参数 FAPControl.AdminState、OpState、CellOpState 计算。
type CellStatus string

const (
	// CellStatusNormal 正常服务
	CellStatusNormal CellStatus = "normal"
	// CellStatusInactive 管理态未激活
	CellStatusInactive CellStatus = "inactive"
	// CellStatusFault 故障
	CellStatusFault CellStatus = "fault"
	// CellStatusDecommissioned 已退服
	CellStatusDecommissioned CellStatus = "decommissioned"
)

// ValidCellStatuses 返回所有有效的小区状态
func ValidCellStatuses() []CellStatus {
	return []CellStatus{CellStatusNormal, CellStatusInactive, CellStatusFault, CellStatusDecommissioned}
}

// IsValid 检查小区状态是否有效
func (s CellStatus) IsValid() bool {
	switch s {
	case CellStatusNormal, CellStatusInactive, CellStatusFault, CellStatusDecommissioned:
		return true
	}
	return false
}

// String 返回字符串形式
func (s CellStatus) String() string {
	return string(s)
}

// ===== MME/AMF 连接状态 (MMEStatus) =====

// MMEStatus 表示设备级 MME/AMF 连接状态。
// LTE 从真实 MME 池实例状态汇总，任一实例连接即为 connected。
type MMEStatus string

const (
	// MMEStatusConnected 至少一个 MME/AMF 连接正常
	MMEStatusConnected MMEStatus = "connected"
	// MMEStatusPartial 历史兼容值；新 LTE 汇总不再生成
	MMEStatusPartial MMEStatus = "partial"
	// MMEStatusDisconnected 无 MME 连接
	MMEStatusDisconnected MMEStatus = "disconnected"
)

// ValidMMEStatuses 返回所有有效的 MME 连接状态
func ValidMMEStatuses() []MMEStatus {
	return []MMEStatus{MMEStatusConnected, MMEStatusPartial, MMEStatusDisconnected}
}

// IsValid 检查 MME 连接状态是否有效
func (s MMEStatus) IsValid() bool {
	switch s {
	case MMEStatusConnected, MMEStatusPartial, MMEStatusDisconnected:
		return true
	}
	return false
}

// String 返回字符串形式
func (s MMEStatus) String() string {
	return string(s)
}

// ===== 时钟同步状态 (SyncStatus) =====

// SyncStatus 表示时钟同步源状态。
// 从 TR069 参数 GPS/BDS/1588 Status 和 tfcsSyncState 计算。
type SyncStatus string

const (
	// SyncStatusGPS GPS 同步
	SyncStatusGPS SyncStatus = "gps"
	// SyncStatusBeidou 北斗同步
	SyncStatusBeidou SyncStatus = "beidou"
	// SyncStatusNTP NTP/1588v2 同步
	SyncStatusNTP SyncStatus = "ntp"
	// SyncStatusError 同步失败
	SyncStatusError SyncStatus = "error"
)

// ValidSyncStatuses 返回所有有效的时钟同步状态
func ValidSyncStatuses() []SyncStatus {
	return []SyncStatus{SyncStatusGPS, SyncStatusBeidou, SyncStatusNTP, SyncStatusError}
}

// IsValid 检查时钟同步状态是否有效
func (s SyncStatus) IsValid() bool {
	switch s {
	case SyncStatusGPS, SyncStatusBeidou, SyncStatusNTP, SyncStatusError:
		return true
	}
	return false
}

// String 返回字符串形式
func (s SyncStatus) String() string {
	return string(s)
}

// ===== GPS 定位状态 (GPSStatus) =====

// GPSStatus 表示 GPS 定位模块状态。
// 从 TR069 参数 X_COM_GPS_Status 计算。
type GPSStatus string

const (
	// GPSStatusNormal GPS 正常
	GPSStatusNormal GPSStatus = "normal"
	// GPSStatusAbnormal GPS 异常
	GPSStatusAbnormal GPSStatus = "abnormal"
	// GPSStatusNoSignal 无 GPS 信号
	GPSStatusNoSignal GPSStatus = "no_signal"
)

// ValidGPSStatuses 返回所有有效的 GPS 状态
func ValidGPSStatuses() []GPSStatus {
	return []GPSStatus{GPSStatusNormal, GPSStatusAbnormal, GPSStatusNoSignal}
}

// IsValid 检查 GPS 状态是否有效
func (s GPSStatus) IsValid() bool {
	switch s {
	case GPSStatusNormal, GPSStatusAbnormal, GPSStatusNoSignal:
		return true
	}
	return false
}

// String 返回字符串形式
func (s GPSStatus) String() string {
	return string(s)
}

// ===== 许可证状态 (LicenseStatus) =====

// LicenseStatus 表示设备许可证状态。
// 从 TR069 参数 X_COM_LICENSE.Capacity.*.State 和 RemainingPeriod 计算。
type LicenseStatus string

const (
	// LicenseStatusActive 许可证有效
	LicenseStatusActive LicenseStatus = "active"
	// LicenseStatusExpiring 许可证即将过期 (<=30天)
	LicenseStatusExpiring LicenseStatus = "expiring"
	// LicenseStatusExpired 许可证已过期
	LicenseStatusExpired LicenseStatus = "expired"
)

// ValidLicenseStatuses 返回所有有效的许可证状态
func ValidLicenseStatuses() []LicenseStatus {
	return []LicenseStatus{LicenseStatusActive, LicenseStatusExpiring, LicenseStatusExpired}
}

// IsValid 检查许可证状态是否有效
func (s LicenseStatus) IsValid() bool {
	switch s {
	case LicenseStatusActive, LicenseStatusExpiring, LicenseStatusExpired:
		return true
	}
	return false
}

// String 返回字符串形式
func (s LicenseStatus) String() string {
	return string(s)
}

// ===== 预注册状态 (RegistrationStatus) =====

// RegistrationStatus 表示设备预注册状态。
type RegistrationStatus string

const (
	// RegistrationStatusPending 待上线
	RegistrationStatusPending RegistrationStatus = "pending"
	// RegistrationStatusOnline 已上线
	RegistrationStatusOnline RegistrationStatus = "online"
	// RegistrationStatusExpired 已过期
	RegistrationStatusExpired RegistrationStatus = "expired"
)

// ValidRegistrationStatuses 返回所有有效的预注册状态
func ValidRegistrationStatuses() []RegistrationStatus {
	return []RegistrationStatus{RegistrationStatusPending, RegistrationStatusOnline, RegistrationStatusExpired}
}

// IsValid 检查预注册状态是否有效
func (s RegistrationStatus) IsValid() bool {
	switch s {
	case RegistrationStatusPending, RegistrationStatusOnline, RegistrationStatusExpired:
		return true
	}
	return false
}

// String 返回字符串形式
func (s RegistrationStatus) String() string {
	return string(s)
}

// Package model 已在 device.go 中声明包注释。
// 此文件将 global 包中的枚举常量和类型别名再导出，提供导入路径统一。
// 内部代码优先导入此包常量，不直接引用 global 包。
//
// 枚举类型：
//   - CarrierCode：运营商编码（CMCC/CTCC/CUCC）
//   - Technology：无线技术（LTE/NR）
//   - DeviceStatus：设备生命周期状态（已发现→已注册→开站中→已激活→维护中→离线→已报废）
//   - AlarmSeverity：告警严重级（居害/主要/次要/警告）
//   - AlarmStatus：告警生命周期（活跃/已确认/已清除）
//   - ParameterType： TR-069 参数数据类型（string/int/uint/bool/dateTime）
//   - DataModelScope：数据模型作用域（万能/OUI/运营商默认）
package model

import "github.com/omcgo/omcgo/global"

// Type aliases — re-export from global for backwards compatibility.
type CarrierCode = global.CarrierCode

const (
	CarrierCMCC = global.CarrierCMCC
	CarrierCTCC = global.CarrierCTCC
	CarrierCUCC = global.CarrierCUCC
)

// ValidCarriers returns all valid carrier codes.
func ValidCarriers() []CarrierCode { return global.ValidCarriers() }

type Technology = global.Technology

const (
	TechLTE = global.TechLTE
	TechNR  = global.TechNR
)

type DeviceStatus = global.DeviceStatus

const (
	DeviceDiscovered     = global.DeviceDiscovered
	DeviceRegistered     = global.DeviceRegistered
	DeviceProvisioning   = global.DeviceProvisioning
	DeviceActive         = global.DeviceActive
	DeviceMaintenance    = global.DeviceMaintenance
	DeviceOffline        = global.DeviceOffline
	DeviceDecommissioned = global.DeviceDecommissioned
)

type AlarmSeverity = global.AlarmSeverity

const (
	AlarmCritical = global.AlarmCritical
	AlarmMajor    = global.AlarmMajor
	AlarmMinor    = global.AlarmMinor
	AlarmWarning  = global.AlarmWarning
)

type AlarmStatus = global.AlarmStatus

const (
	AlarmActive       = global.AlarmActive
	AlarmAcknowledged = global.AlarmAcknowledged
	AlarmCleared      = global.AlarmCleared
)

type ParameterType = global.ParameterType

const (
	ParamString   = global.ParamString
	ParamInt      = global.ParamInt
	ParamUint     = global.ParamUint
	ParamBool     = global.ParamBool
	ParamDateTime = global.ParamDateTime
)

type DataModelScope = global.DataModelScope

const (
	ScopeProduct        = global.ScopeProduct
	ScopeOUI            = global.ScopeOUI
	ScopeCarrierDefault = global.ScopeCarrierDefault
)

package global

import (
	"encoding/json"
	"strings"
)

// CarrierCode identifies a mobile network operator.
type CarrierCode string

const (
	CarrierCMCC CarrierCode = "cmcc" // China Mobile
	CarrierCTCC CarrierCode = "ctcc" // China Telecom
	CarrierCUCC CarrierCode = "cucc" // China Unicom
)

// ValidCarriers returns all valid carrier codes.
func ValidCarriers() []CarrierCode {
	return []CarrierCode{CarrierCMCC, CarrierCTCC, CarrierCUCC}
}

// IsValid checks if the carrier code is recognized.
func (c CarrierCode) IsValid() bool {
	switch c {
	case CarrierCMCC, CarrierCTCC, CarrierCUCC:
		return true
	}
	return false
}

// UnmarshalJSON 让外部 JSON 入参 case-insensitive：北向 OSS / 第三方系统按
// 习惯送 "CMCC"/"Cmcc"/"cmcc" 任一形式都解到内部 canonical 小写。后续
// binding:"oneof=cmcc ctcc cucc" 校验、SQL 比较、字典对照都按小写走 —— 同
// 一处地方做规范化，下游链路保持简单。
func (c *CarrierCode) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	*c = CarrierCode(strings.ToLower(strings.TrimSpace(s)))
	return nil
}

// Technology identifies a radio access technology.
type Technology string

const (
	TechLTE Technology = "lte" // 4G LTE
	TechNR  Technology = "nr"  // 5G NR SA
	TechGSM Technology = "gsm" // 2G GSM（小基站 BSC/BTS 等）
)

// IsValid checks if the technology is recognized.
func (t Technology) IsValid() bool {
	switch t {
	case TechLTE, TechNR, TechGSM:
		return true
	}
	return false
}

// NormalizeTechnology 把人类可读 / 厂商习惯字符串（"4G" / "LTE" / "5G NR" / "2G" 等）
// 归一为本系统内部 canonical 小写代码（lte/nr/gsm）。
//
// 用途：
//   - products.xml Loader 写 DB 前归一（XML 习惯写 "4G"，DB 列存 "lte"）
//   - 北向 / OSS 接口入参兼容
//
// 不认识的输入：返回 trim+lower 后的原值，让上层用 IsValid 判定是否拒绝。
func NormalizeTechnology(in string) Technology {
	s := strings.ToLower(strings.TrimSpace(in))
	switch s {
	case "lte", "4g", "4g lte", "lte fdd", "lte tdd", "fdd-lte", "tdd-lte":
		return TechLTE
	case "nr", "5g", "5g nr", "5g nr sa", "5g sa":
		return TechNR
	case "gsm", "2g", "2g gsm":
		return TechGSM
	}
	return Technology(s)
}

// UnmarshalJSON 让外部 JSON 入参 case-insensitive 且统一别名。
//
// 3GPP/TR-181 标准约定 "LTE"/"NR" 大写，本系统内部 canonical 小写。
// 北向 OSS / 第三方系统按 TR-181 习惯送 "LTE" 不能被 oneof binding 拒掉，
// 同时设备 Inform 推断侧 (detectTechnology) 也保持只产生小写，全链路一致。
// 别名（"4G"/"5G"/"2G"）走 NormalizeTechnology 折算为 canonical 代码。
func (t *Technology) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	*t = NormalizeTechnology(s)
	return nil
}

// DeviceStatus represents the legacy device state (deprecated).
//
// DEPRECATED (T-0162): 此类型混淆了"生命周期"与"实时在线"两个正交概念。
// 已被 DeviceLifecycle（生命周期）+ bool is_online（实时在线）替代。保留过渡
// 期供 P3 阶段渐进改造所有调用方，**不要在新代码里再用**。P3 完成后整体删除。
type DeviceStatus string

const (
	DeviceDiscovered     DeviceStatus = "discovered"
	DeviceRegistered     DeviceStatus = "registered"
	DeviceProvisioning   DeviceStatus = "provisioning"
	DeviceActive         DeviceStatus = "active"
	DeviceMaintenance    DeviceStatus = "maintenance"
	DeviceOffline        DeviceStatus = "offline"
	DeviceDecommissioned DeviceStatus = "decommissioned"
)

// DeviceLifecycle represents the business lifecycle stage of a device.
//
// 与"实时在线状态"（devices.is_online BOOLEAN，由 HeartbeatMonitor 维护）完全
// 正交：lifecycle 由业务流程 / 运维决策推进，is_online 由系统自动维护。
//
// 合法转移（state_machine.go validLifecycleTransitions）：
//
//	discovered     → registered, commissioned
//	registered     → provisioning, commissioned, decommissioned
//	provisioning   → commissioned, registered, decommissioned
//	commissioned   → maintenance, decommissioned
//	maintenance    → commissioned, decommissioned
//	decommissioned → (终态，无转出)
//
// 设计文档：docs/design/device-lifecycle-online-status-decouple-20260520.md
type DeviceLifecycle string

const (
	LifecycleDiscovered     DeviceLifecycle = "discovered"
	LifecycleRegistered     DeviceLifecycle = "registered"
	LifecycleProvisioning   DeviceLifecycle = "provisioning"
	LifecycleCommissioned   DeviceLifecycle = "commissioned"
	LifecycleMaintenance    DeviceLifecycle = "maintenance"
	LifecycleDecommissioned DeviceLifecycle = "decommissioned"
)

// IsValid checks if the lifecycle value is recognized.
func (l DeviceLifecycle) IsValid() bool {
	switch l {
	case LifecycleDiscovered, LifecycleRegistered, LifecycleProvisioning,
		LifecycleCommissioned, LifecycleMaintenance, LifecycleDecommissioned:
		return true
	}
	return false
}

// AlarmSeverity represents the urgency level of an alarm.
type AlarmSeverity int

const (
	AlarmCritical AlarmSeverity = 1 // Critical
	AlarmMajor    AlarmSeverity = 2 // Major
	AlarmMinor    AlarmSeverity = 3 // Minor
	AlarmWarning  AlarmSeverity = 4 // Warning
)

// AlarmStatus represents the lifecycle state of an alarm.
type AlarmStatus string

const (
	AlarmActive       AlarmStatus = "active"
	AlarmAcknowledged AlarmStatus = "acknowledged"
	AlarmCleared      AlarmStatus = "cleared"
)

// ParameterType represents a TR069 parameter data type.
type ParameterType string

const (
	ParamString   ParameterType = "string"
	ParamInt      ParameterType = "int"
	ParamUint     ParameterType = "unsignedInt"
	ParamBool     ParameterType = "boolean"
	ParamDateTime ParameterType = "dateTime"
)

// DataModelScope represents the specificity level of a data model definition.
// Resolution priority: Product > OUI > CarrierDefault.
type DataModelScope string

const (
	ScopeProduct        DataModelScope = "product"
	ScopeOUI            DataModelScope = "oui"
	ScopeCarrierDefault DataModelScope = "carrier_default"
)

// Default device group UUIDs (fixed, referenced in migration 000065).
const (
	DefaultLevel1GroupID = "00000000-0000-0000-0000-000000000001"
	DefaultLevel2GroupID = "00000000-0000-0000-0000-000000000002"
)

// RegistrationStatus represents the state of a pre-registered device.
type RegistrationStatus string

const (
	RegistrationPending RegistrationStatus = "pending"
	RegistrationOnline  RegistrationStatus = "online"
	RegistrationExpired RegistrationStatus = "expired"
)

// DeviceGroupStatus represents the state of a device group.
type DeviceGroupStatus string

const (
	GroupStatusActive   DeviceGroupStatus = "active"
	GroupStatusDisabled DeviceGroupStatus = "disabled"
)

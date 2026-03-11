package global

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

// Technology identifies a radio access technology.
type Technology string

const (
	TechLTE Technology = "lte" // 4G LTE
	TechNR  Technology = "nr"  // 5G NR SA
)

// IsValid checks if the technology is recognized.
func (t Technology) IsValid() bool {
	switch t {
	case TechLTE, TechNR:
		return true
	}
	return false
}

// DeviceStatus represents the lifecycle state of a device.
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

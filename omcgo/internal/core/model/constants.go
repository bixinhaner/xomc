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

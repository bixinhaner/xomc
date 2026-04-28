package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_ValidCarriers_NotEmpty(t *testing.T) {
	carriers := ValidCarriers()
	assert.NotEmpty(t, carriers, "expected at least one carrier in ValidCarriers()")
}

func Test_ValidCarriers_ContainsAllThree(t *testing.T) {
	carriers := ValidCarriers()
	set := make(map[CarrierCode]bool, len(carriers))
	for _, c := range carriers {
		set[c] = true
	}

	assert.True(t, set[CarrierCMCC], "missing CMCC")
	assert.True(t, set[CarrierCTCC], "missing CTCC")
	assert.True(t, set[CarrierCUCC], "missing CUCC")
}

func Test_CarrierConstants_DistinctValues(t *testing.T) {
	// Re-exported constants must hold distinct underlying values to avoid mix-ups.
	codes := map[CarrierCode]string{
		CarrierCMCC: "CMCC",
		CarrierCTCC: "CTCC",
		CarrierCUCC: "CUCC",
	}
	assert.Len(t, codes, 3, "carrier codes must be distinct")
}

func Test_TechConstants_DistinctValues(t *testing.T) {
	techs := map[Technology]string{
		TechLTE: "lte",
		TechNR:  "nr",
	}
	assert.Len(t, techs, 2)
}

func Test_DeviceStatus_DistinctValues(t *testing.T) {
	statuses := map[DeviceStatus]string{
		DeviceDiscovered:     "discovered",
		DeviceRegistered:     "registered",
		DeviceProvisioning:   "provisioning",
		DeviceActive:         "active",
		DeviceMaintenance:    "maintenance",
		DeviceOffline:        "offline",
		DeviceDecommissioned: "decommissioned",
	}
	assert.Len(t, statuses, 7, "device status enum should have 7 distinct values")
}

func Test_AlarmSeverity_DistinctValues(t *testing.T) {
	sevs := map[AlarmSeverity]string{
		AlarmCritical: "critical",
		AlarmMajor:    "major",
		AlarmMinor:    "minor",
		AlarmWarning:  "warning",
	}
	assert.Len(t, sevs, 4)
}

func Test_AlarmStatus_DistinctValues(t *testing.T) {
	statuses := map[AlarmStatus]string{
		AlarmActive:       "active",
		AlarmAcknowledged: "acknowledged",
		AlarmCleared:      "cleared",
	}
	assert.Len(t, statuses, 3)
}

func Test_ParameterType_DistinctValues(t *testing.T) {
	types := map[ParameterType]string{
		ParamString:   "string",
		ParamInt:      "int",
		ParamUint:     "uint",
		ParamBool:     "bool",
		ParamDateTime: "dateTime",
	}
	assert.Len(t, types, 5)
}

func Test_DataModelScope_DistinctValues(t *testing.T) {
	scopes := map[DataModelScope]string{
		ScopeProduct:        "product",
		ScopeOUI:            "oui",
		ScopeCarrierDefault: "carrier_default",
	}
	assert.Len(t, scopes, 3)
}

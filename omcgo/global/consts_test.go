package global

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_CarrierCode_IsValid(t *testing.T) {
	tests := []struct {
		name    string
		carrier CarrierCode
		want    bool
	}{
		{"cmcc is valid", CarrierCMCC, true},
		{"ctcc is valid", CarrierCTCC, true},
		{"cucc is valid", CarrierCUCC, true},
		{"empty is invalid", CarrierCode(""), false},
		{"unknown is invalid", CarrierCode("unknown"), false},
		{"uppercase CMCC is invalid", CarrierCode("CMCC"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.carrier.IsValid())
		})
	}
}

func Test_Technology_IsValid(t *testing.T) {
	tests := []struct {
		name string
		tech Technology
		want bool
	}{
		{"lte is valid", TechLTE, true},
		{"nr is valid", TechNR, true},
		{"empty is invalid", Technology(""), false},
		{"unknown is invalid", Technology("wifi"), false},
		{"uppercase LTE is invalid", Technology("LTE"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.tech.IsValid())
		})
	}
}

func Test_ValidCarriers(t *testing.T) {
	carriers := ValidCarriers()

	assert.Len(t, carriers, 3)
	assert.Contains(t, carriers, CarrierCMCC)
	assert.Contains(t, carriers, CarrierCTCC)
	assert.Contains(t, carriers, CarrierCUCC)
}

func Test_CarrierCode_Values(t *testing.T) {
	assert.Equal(t, CarrierCode("cmcc"), CarrierCMCC)
	assert.Equal(t, CarrierCode("ctcc"), CarrierCTCC)
	assert.Equal(t, CarrierCode("cucc"), CarrierCUCC)
}

func Test_Technology_Values(t *testing.T) {
	assert.Equal(t, Technology("lte"), TechLTE)
	assert.Equal(t, Technology("nr"), TechNR)
}

func Test_AlarmSeverity_Ordering(t *testing.T) {
	// Critical should be most severe (lowest number)
	assert.Less(t, int(AlarmCritical), int(AlarmMajor))
	assert.Less(t, int(AlarmMajor), int(AlarmMinor))
	assert.Less(t, int(AlarmMinor), int(AlarmWarning))
}

func Test_ParameterType_Values(t *testing.T) {
	tests := []struct {
		name string
		pt   ParameterType
		want string
	}{
		{"string", ParamString, "string"},
		{"int", ParamInt, "int"},
		{"uint", ParamUint, "unsignedInt"},
		{"bool", ParamBool, "boolean"},
		{"dateTime", ParamDateTime, "dateTime"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, string(tt.pt))
		})
	}
}

func Test_DataModelScope_Values(t *testing.T) {
	assert.Equal(t, DataModelScope("product"), ScopeProduct)
	assert.Equal(t, DataModelScope("oui"), ScopeOUI)
	assert.Equal(t, DataModelScope("carrier_default"), ScopeCarrierDefault)
}

func Test_DeviceStatus_Values(t *testing.T) {
	statuses := []DeviceStatus{
		DeviceDiscovered,
		DeviceRegistered,
		DeviceProvisioning,
		DeviceActive,
		DeviceMaintenance,
		DeviceOffline,
		DeviceDecommissioned,
	}

	assert.Len(t, statuses, 7)

	// Ensure all are distinct
	seen := make(map[DeviceStatus]bool)
	for _, s := range statuses {
		assert.False(t, seen[s], "duplicate status: %s", s)
		seen[s] = true
	}
}

func Test_AlarmStatus_Values(t *testing.T) {
	assert.Equal(t, AlarmStatus("active"), AlarmActive)
	assert.Equal(t, AlarmStatus("acknowledged"), AlarmAcknowledged)
	assert.Equal(t, AlarmStatus("cleared"), AlarmCleared)
}

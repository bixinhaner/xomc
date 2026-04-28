package cmcc

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/core/model"
)

func Test_New_ReturnsAdapter(t *testing.T) {
	c := New()
	require.NotNil(t, c)
	assert.Equal(t, model.CarrierCMCC, c.Code())
	assert.Equal(t, "中国移动", c.Name())
}

func Test_SupportedTechnologies(t *testing.T) {
	c := New()
	techs := c.SupportedTechnologies()
	assert.Contains(t, techs, model.TechLTE)
	assert.Contains(t, techs, model.TechNR)
}

func Test_DefaultDataModelVersions(t *testing.T) {
	c := New()
	tests := []struct {
		name     string
		tech     model.Technology
		wantNon0 bool
	}{
		{"LTE", model.TechLTE, true},
		{"NR", model.TechNR, true},
		{"unknown returns nil", model.Technology("unknown"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			versions := c.DefaultDataModelVersions(tt.tech)
			if tt.wantNon0 {
				assert.NotEmpty(t, versions)
			} else {
				assert.Empty(t, versions)
			}
		})
	}
}

func Test_KnownOUIProductClasses(t *testing.T) {
	c := New()
	for _, tech := range []model.Technology{model.TechLTE, model.TechNR} {
		t.Run(string(tech), func(t *testing.T) {
			info := c.KnownOUIProductClasses(tech)
			// The result might be empty for some implementations - just verify it doesn't panic.
			_ = info
		})
	}
}

func Test_MapParameterToUnified_Unknown(t *testing.T) {
	c := New()
	// Unknown path should return itself.
	got := c.MapParameterToUnified("UnknownPath")
	assert.Equal(t, "UnknownPath", got)
}

func Test_MapUnifiedToParameter_Unknown(t *testing.T) {
	c := New()
	got := c.MapUnifiedToParameter("UnknownName")
	assert.Equal(t, "UnknownName", got)
}

func Test_ProvisioningTemplates(t *testing.T) {
	c := New()
	for _, tech := range []model.Technology{model.TechLTE, model.TechNR} {
		_ = c.ProvisioningTemplates(tech)
	}
}

func Test_KPIDefinitions(t *testing.T) {
	c := New()
	for _, tech := range []model.Technology{model.TechLTE, model.TechNR} {
		_ = c.KPIDefinitions(tech)
	}
}

func Test_AlarmSeverityMapping(t *testing.T) {
	c := New()
	tests := []struct {
		code string
		want model.AlarmSeverity
	}{
		{"CELL_UNAVAILABLE", model.AlarmCritical},
		{"S1_LINK_FAILURE", model.AlarmCritical},
		{"X2_LINK_FAILURE", model.AlarmMajor},
		{"RADIO_FAILURE", model.AlarmMajor},
		{"GPS_FAILURE", model.AlarmMajor},
		{"TEMP_HIGH", model.AlarmMinor},
		{"VSWR_HIGH", model.AlarmMinor},
		{"MEM_OVERLOAD", model.AlarmWarning},
		{"CONFIG_MISMATCH", model.AlarmWarning},
		{"UNKNOWN_ALARM", model.AlarmWarning}, // unknown defaults to warning
	}
	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			got := c.AlarmSeverityMapping(tt.code)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_ValidateParameter_PLMNID(t *testing.T) {
	c := New()
	plmnPath := "Device.Services.FAPService.1.CellConfig.LTE.EPC.PLMNList.1.PLMNID"

	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"valid 5 digits", "46000", false},
		{"valid 6 digits", "460001", false},
		{"too short", "1234", true},
		{"too long", "1234567", true},
		{"non-digit", "abcde", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := c.ValidateParameter(plmnPath, tt.value)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_ValidateParameter_NoValidator(t *testing.T) {
	c := New()
	// A path without a validator should return nil error.
	err := c.ValidateParameter("Some.Other.Path", "anyvalue")
	assert.NoError(t, err)
}

func Test_SupportsDirectConnection(t *testing.T) {
	c := New()
	assert.True(t, c.SupportsDirectConnection())
}

func Test_GetInfoParamMapping_LTE(t *testing.T) {
	c := New()
	m := c.GetInfoParamMapping(model.TechLTE)
	assert.NotEmpty(t, m)
	// CMCC LTE uses X_CMCC_MACAddress
	_, hasMAC := m["Device.DeviceInfo.X_CMCC_MACAddress"]
	assert.True(t, hasMAC, "expected CMCC LTE mapping to include X_CMCC_MACAddress")
}

func Test_GetInfoParamMapping_NR(t *testing.T) {
	c := New()
	m := c.GetInfoParamMapping(model.TechNR)
	assert.NotEmpty(t, m)
}

func Test_RoundTripParameterMapping(t *testing.T) {
	c := New()
	// Pick first key from the internal mapping and round-trip through both directions
	for path, unified := range c.paramMapping {
		t.Run(path, func(t *testing.T) {
			gotUnified := c.MapParameterToUnified(path)
			assert.Equal(t, unified, gotUnified)

			gotPath := c.MapUnifiedToParameter(unified)
			assert.Equal(t, path, gotPath)
		})
		break // single iteration sufficient for round-trip smoke test
	}
}

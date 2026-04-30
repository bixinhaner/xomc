package cucc

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/core/model"
)

func Test_New_ReturnsAdapter(t *testing.T) {
	c := New()
	require.NotNil(t, c)
	assert.Equal(t, model.CarrierCUCC, c.Code())
	assert.Equal(t, "中国联通", c.Name())
}

func Test_SupportedTechnologies_NROnly(t *testing.T) {
	c := New()
	techs := c.SupportedTechnologies()
	// CUCC only supports NR, no LTE.
	assert.Contains(t, techs, model.TechNR)
	assert.NotContains(t, techs, model.TechLTE)
}

func Test_DefaultDataModelVersions(t *testing.T) {
	c := New()

	t.Run("NR has versions", func(t *testing.T) {
		versions := c.DefaultDataModelVersions(model.TechNR)
		assert.NotEmpty(t, versions)
	})

	t.Run("LTE returns nil", func(t *testing.T) {
		versions := c.DefaultDataModelVersions(model.TechLTE)
		assert.Empty(t, versions)
	})
}

func Test_KnownOUIProductClasses(t *testing.T) {
	c := New()
	_ = c.KnownOUIProductClasses(model.TechNR)
}

func Test_MapParameterToUnified_Unknown(t *testing.T) {
	c := New()
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
	_ = c.ProvisioningTemplates(model.TechNR)
}

func Test_KPIDefinitions(t *testing.T) {
	c := New()
	_ = c.KPIDefinitions(model.TechNR)
}

func Test_AlarmSeverityMapping(t *testing.T) {
	c := New()
	tests := []struct {
		code string
		want model.AlarmSeverity
	}{
		{"CELL_UNAVAILABLE", model.AlarmCritical},
		{"NG_LINK_FAILURE", model.AlarmCritical},
		{"SITE_POWER_FAILURE", model.AlarmCritical},
		{"XN_LINK_FAILURE", model.AlarmMajor},
		{"BACKHAUL_DEGRADED", model.AlarmMajor},
		{"RF_TX_FAILURE", model.AlarmMajor},
		{"VSWR_HIGH", model.AlarmMinor},
		{"POWER_DEGRADED", model.AlarmMinor},
		{"SW_VERSION_MISMATCH", model.AlarmWarning},
		{"LICENSE_EXPIRING", model.AlarmWarning},
		{"UNKNOWN_ALARM", model.AlarmWarning},
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
	plmnPath := "Device.Services.FAPService.1.CellConfig.NR.Core.PLMNList.1.PLMNID"

	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"valid 5 digits", "46001", false},
		{"valid 6 digits", "460001", false},
		{"too short", "123", true},
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

func Test_ValidateParameter_SST(t *testing.T) {
	c := New()
	sstPath := "Device.Services.FAPService.1.CellConfig.NR.Core.SNSSAI.1.SST"

	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"valid digit", "1", false},
		{"empty", "", true},
		{"non-digit", "ab", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := c.ValidateParameter(sstPath, tt.value)
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
	err := c.ValidateParameter("Some.Other.Path", "anyvalue")
	assert.NoError(t, err)
}

func Test_GetInfoParamMapping_NR(t *testing.T) {
	c := New()
	m := c.GetInfoParamMapping(model.TechNR)
	assert.NotEmpty(t, m)
}

func Test_GetInfoParamMapping_LTE_Empty(t *testing.T) {
	c := New()
	// CUCC doesn't support LTE
	m := c.GetInfoParamMapping(model.TechLTE)
	assert.Empty(t, m)
}

// T-0029: CUCC NR-only — LTE returns "" so device layer fails fast with
// a clear error rather than queue an unkeyed FAPControl.LTE command.
func Test_RFControlPath_T0029(t *testing.T) {
	c := New()
	assert.Equal(t, "Device.Services.FAPService.1.FAPControl.NR.AdminState", c.RFControlPath(model.TechNR))
	assert.Equal(t, "", c.RFControlPath(model.TechLTE), "CUCC NR-only: LTE must return empty path")
	assert.Equal(t, "", c.RFControlPath(model.Technology("unknown")))
}

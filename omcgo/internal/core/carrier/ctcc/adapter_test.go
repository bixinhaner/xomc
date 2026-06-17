package ctcc

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/core/model"
)

func Test_New_ReturnsAdapter(t *testing.T) {
	c := New()
	require.NotNil(t, c)
	assert.Equal(t, model.CarrierCTCC, c.Code())
	assert.Equal(t, "中国电信", c.Name())
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
		_ = c.KnownOUIProductClasses(tech)
	}
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
	for _, tech := range []model.Technology{model.TechLTE, model.TechNR} {
		_ = c.ProvisioningTemplates(tech)
	}
}

func Test_AlarmSeverityMapping(t *testing.T) {
	c := New()
	tests := []struct {
		code string
		want model.AlarmSeverity
	}{
		{"CELL_UNAVAILABLE", model.AlarmCritical},
		{"SCTP_LINK_FAILURE", model.AlarmCritical},
		{"BACKHAUL_FAILURE", model.AlarmMajor},
		{"X2_LINK_FAILURE", model.AlarmMajor},
		{"VSWR_HIGH", model.AlarmMinor},
		{"POWER_DEGRADED", model.AlarmMinor},
		{"LICENSE_EXPIRING", model.AlarmWarning},
		{"UNKNOWN_ALARM", model.AlarmSeverity(0)}, // unknown has no explicit override
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
		{"valid 5 digits", "46011", false},
		{"valid 6 digits", "460011", false},
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
	err := c.ValidateParameter("Some.Other.Path", "anyvalue")
	assert.NoError(t, err)
}

func Test_GetInfoParamMapping_LTE(t *testing.T) {
	c := New()
	m := c.GetInfoParamMapping(model.TechLTE)
	assert.NotEmpty(t, m)
}

func Test_GetInfoParamMapping_NR(t *testing.T) {
	c := New()
	m := c.GetInfoParamMapping(model.TechNR)
	assert.NotEmpty(t, m)
	// Baicells/Dengyo NR 设备实际上报的 .CellConfig.1.NR.* 索引路径必须覆盖到。
	for _, want := range []struct {
		path string
		col  string
	}{
		{"Device.Services.FAPService.1.CellConfig.1.NR.RAN.RF.PhyCellID", "pci"},
		{"Device.Services.FAPService.1.CellConfig.1.NR.RAN.RF.NRARFCNDL", "freq_point"},
		{"Device.Services.FAPService.1.CellConfig.1.NR.RAN.RF.DLBandwidth", "bandwidth"},
		{"Device.Services.FAPService.1.CellConfig.1.NR.RAN.Common.CellLocalId", "cell_id"},
		// #362 NR 版：取 RW PowerModify（快速设置「功率调整」）作为 transmit_power。
		// 不使用只读硬件能力上限 MaxTxPower，语义与 LTE ReferenceSignalPower 对齐。
		{"Device.Services.FAPService.1.CellConfig.1.NR.RAN.PowerModify", "transmit_power"},
	} {
		got, ok := m[want.path]
		assert.Truef(t, ok, "CTCC NR mapping missing path %s", want.path)
		assert.Equalf(t, want.col, got, "CTCC NR mapping %s expected column %s", want.path, want.col)
	}
}

// T-0029: CTCC RF control path follows TR-181 standard.
func Test_RFControlPath_T0029(t *testing.T) {
	c := New()
	assert.Equal(t, "Device.Services.FAPService.1.FAPControl.LTE.AdminState", c.RFControlPath(model.TechLTE))
	assert.Equal(t, "Device.Services.FAPService.1.FAPControl.NR.AdminState", c.RFControlPath(model.TechNR))
	assert.Equal(t, "", c.RFControlPath(model.Technology("unknown")))
}

// #17: CTCC 采集全部三类 MR 报告（MRO/MRS/MRE）。
func Test_SupportsMRType_CTCC_All(t *testing.T) {
	c := New()
	assert.True(t, c.SupportsMRType(model.MRTypeMRO))
	assert.True(t, c.SupportsMRType(model.MRTypeMRS))
	assert.True(t, c.SupportsMRType(model.MRTypeMRE))
	assert.False(t, c.SupportsMRType("unknown"), "unknown MR type must be rejected")
}

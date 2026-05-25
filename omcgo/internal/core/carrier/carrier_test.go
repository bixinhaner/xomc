package carrier_test

import (
	"testing"

	"github.com/omcgo/omcgo/internal/core/carrier"
	"github.com/omcgo/omcgo/internal/core/carrier/cmcc"
	"github.com/omcgo/omcgo/internal/core/carrier/ctcc"
	"github.com/omcgo/omcgo/internal/core/carrier/cucc"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestRegistry() *carrier.CarrierRegistry {
	r := carrier.NewRegistry()
	r.Register(cmcc.New())
	r.Register(ctcc.New())
	r.Register(cucc.New())
	return r
}

func TestRegistryGet(t *testing.T) {
	r := newTestRegistry()

	tests := []struct {
		code    model.CarrierCode
		wantErr bool
	}{
		{model.CarrierCMCC, false},
		{model.CarrierCTCC, false},
		{model.CarrierCUCC, false},
		{"unknown", true},
	}

	for _, tt := range tests {
		c, err := r.Get(tt.code)
		if tt.wantErr {
			assert.Error(t, err)
			assert.Nil(t, c)
		} else {
			require.NoError(t, err)
			assert.Equal(t, tt.code, c.Code())
		}
	}
}

func TestRegistryAll(t *testing.T) {
	r := newTestRegistry()
	all := r.All()
	assert.Len(t, all, 3)
}

func TestRegistryResolveByOUI(t *testing.T) {
	r := newTestRegistry()

	tests := []struct {
		oui  string
		want model.CarrierCode
	}{
		{"00E0FC", model.CarrierCMCC}, // Huawei (in CMCC known list)
		{"001E7E", model.CarrierCMCC}, // ZTE (in CMCC known list)
		{"FFFFFF", ""},                 // unknown OUI
	}

	for _, tt := range tests {
		got := r.ResolveByOUI(tt.oui)
		assert.Equal(t, tt.want, got, "OUI: %s", tt.oui)
	}
}

func TestCMCCCarrier(t *testing.T) {
	c := cmcc.New()

	t.Run("identity", func(t *testing.T) {
		assert.Equal(t, model.CarrierCMCC, c.Code())
		assert.Equal(t, "中国移动", c.Name())
	})

	t.Run("technologies", func(t *testing.T) {
		techs := c.SupportedTechnologies()
		assert.Contains(t, techs, model.TechLTE)
		assert.Contains(t, techs, model.TechNR)
	})

	t.Run("data_model_versions", func(t *testing.T) {
		lteVersions := c.DefaultDataModelVersions(model.TechLTE)
		assert.Contains(t, lteVersions, "V2.3")

		nrVersions := c.DefaultDataModelVersions(model.TechNR)
		assert.Contains(t, nrVersions, "V1.9.4")
	})

	t.Run("parameter_mapping_roundtrip", func(t *testing.T) {
		testPaths := []string{
			"Device.X_CMCC.ENBId",
			"Device.X_CMCC.SiteName",
			"Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.PhyCellID",
			"Device.DeviceInfo.SerialNumber",
		}
		for _, path := range testPaths {
			unified := c.MapParameterToUnified(path)
			assert.NotEmpty(t, unified, "path: %s", path)
			back := c.MapUnifiedToParameter(unified)
			assert.Equal(t, path, back, "roundtrip failed for path: %s -> %s -> %s", path, unified, back)
		}
	})

	// T-0164-P1: KPIDefinitions 已从 carrier 接口下线（KPI 元数据走 DB / pm/kpi/router），
	// 原 kpi_definitions 子测试已删除。

	t.Run("provisioning_templates", func(t *testing.T) {
		lteTemplates := c.ProvisioningTemplates(model.TechLTE)
		require.NotEmpty(t, lteTemplates)
		assert.Equal(t, "cmcc_lte_default", lteTemplates[0].Name)
		assert.NotEmpty(t, lteTemplates[0].Parameters)
		assert.NotEmpty(t, lteTemplates[0].Required)

		nrTemplates := c.ProvisioningTemplates(model.TechNR)
		require.NotEmpty(t, nrTemplates)
		assert.Equal(t, "cmcc_nr_default", nrTemplates[0].Name)
	})

	t.Run("alarm_severity", func(t *testing.T) {
		assert.Equal(t, model.AlarmCritical, c.AlarmSeverityMapping("CELL_UNAVAILABLE"))
		assert.Equal(t, model.AlarmMajor, c.AlarmSeverityMapping("RADIO_FAILURE"))
		assert.Equal(t, model.AlarmSeverity(0), c.AlarmSeverityMapping("UNKNOWN_CODE"))
	})

	t.Run("known_oui_products", func(t *testing.T) {
		lteProducts := c.KnownOUIProductClasses(model.TechLTE)
		assert.NotEmpty(t, lteProducts)
		// Huawei should be known
		found := false
		for _, p := range lteProducts {
			if p.OUI == "00E0FC" {
				found = true
			}
		}
		assert.True(t, found, "Huawei OUI should be in known products")
	})
}

func TestCTCCCarrier(t *testing.T) {
	c := ctcc.New()

	t.Run("identity", func(t *testing.T) {
		assert.Equal(t, model.CarrierCTCC, c.Code())
		assert.Equal(t, "中国电信", c.Name())
	})

	t.Run("technologies", func(t *testing.T) {
		techs := c.SupportedTechnologies()
		assert.Contains(t, techs, model.TechLTE)
		assert.Contains(t, techs, model.TechNR)
	})

	t.Run("data_model_versions", func(t *testing.T) {
		lteVersions := c.DefaultDataModelVersions(model.TechLTE)
		assert.Contains(t, lteVersions, "V1.0.2")

		nrVersions := c.DefaultDataModelVersions(model.TechNR)
		assert.Contains(t, nrVersions, "V2.8.7")
	})

	t.Run("parameter_mapping_roundtrip", func(t *testing.T) {
		testPaths := []string{
			"Device.X_CTCC.ENBId",
			"Device.X_CTCC.SiteName",
			"Device.X_CTCC.PMEnable",
			"Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.PhyCellID",
			"Device.DeviceInfo.SerialNumber",
			"Device.Services.FAPService.1.CellConfig.NR.RAN.RF.NRPCI",
		}
		for _, path := range testPaths {
			unified := c.MapParameterToUnified(path)
			assert.NotEmpty(t, unified, "path: %s", path)
			back := c.MapUnifiedToParameter(unified)
			assert.Equal(t, path, back, "roundtrip failed for path: %s -> %s -> %s", path, unified, back)
		}
	})

	// T-0164-P1: kpi_definitions 子测试已删除（同上）。

	t.Run("provisioning_templates", func(t *testing.T) {
		lteTemplates := c.ProvisioningTemplates(model.TechLTE)
		require.NotEmpty(t, lteTemplates)
		assert.Equal(t, "ctcc_lte_default", lteTemplates[0].Name)
		assert.NotEmpty(t, lteTemplates[0].Parameters)
		assert.NotEmpty(t, lteTemplates[0].Required)

		nrTemplates := c.ProvisioningTemplates(model.TechNR)
		require.NotEmpty(t, nrTemplates)
		assert.Equal(t, "ctcc_nr_default", nrTemplates[0].Name)
	})

	t.Run("alarm_severity", func(t *testing.T) {
		assert.Equal(t, model.AlarmCritical, c.AlarmSeverityMapping("CELL_UNAVAILABLE"))
		assert.Equal(t, model.AlarmCritical, c.AlarmSeverityMapping("SCTP_LINK_FAILURE"))
		assert.Equal(t, model.AlarmMajor, c.AlarmSeverityMapping("RADIO_FAILURE"))
		assert.Equal(t, model.AlarmMajor, c.AlarmSeverityMapping("BACKHAUL_FAILURE"))
		assert.Equal(t, model.AlarmMinor, c.AlarmSeverityMapping("POWER_DEGRADED"))
		assert.Equal(t, model.AlarmSeverity(0), c.AlarmSeverityMapping("UNKNOWN_CODE"))
	})

	t.Run("known_oui_products", func(t *testing.T) {
		lteProducts := c.KnownOUIProductClasses(model.TechLTE)
		assert.NotEmpty(t, lteProducts)
		// Huawei should be known for CTCC
		found := false
		for _, p := range lteProducts {
			if p.OUI == "00E0FC" {
				found = true
			}
		}
		assert.True(t, found, "Huawei OUI should be in CTCC known products")
	})

	t.Run("validate_parameter", func(t *testing.T) {
		// Valid PLMNID
		err := c.ValidateParameter("Device.Services.FAPService.1.CellConfig.LTE.EPC.PLMNList.1.PLMNID", "46011")
		assert.NoError(t, err)
		// Invalid PLMNID (too short)
		err = c.ValidateParameter("Device.Services.FAPService.1.CellConfig.LTE.EPC.PLMNList.1.PLMNID", "460")
		assert.Error(t, err)
	})
}

func TestCUCCCarrier(t *testing.T) {
	c := cucc.New()

	assert.Equal(t, model.CarrierCUCC, c.Code())
	assert.Equal(t, "中国联通", c.Name())

	techs := c.SupportedTechnologies()
	assert.Contains(t, techs, model.TechNR)
	assert.NotContains(t, techs, model.TechLTE, "CUCC should not support LTE")
}

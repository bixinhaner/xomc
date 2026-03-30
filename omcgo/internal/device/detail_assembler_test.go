package device

import (
	"testing"

	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/stretchr/testify/assert"
)

func TestAssembleMMEPool(t *testing.T) {
	params := []model.DeviceParameter{
		{ParameterPath: "Device.Services.FAPService.1.CellConfig.LTE.EPC.MmePoolConfigParam.1.MME1Status", ParameterValue: "1"},
		{ParameterPath: "Device.Services.FAPService.1.CellConfig.LTE.EPC.MmePoolConfigParam.1.MME1Address", ParameterValue: "10.0.0.1"},
		{ParameterPath: "Device.Services.FAPService.1.CellConfig.LTE.EPC.MmePoolConfigParam.1.PLMNID", ParameterValue: "46000"},
		{ParameterPath: "Device.Services.FAPService.1.CellConfig.LTE.EPC.MmePoolConfigParam.2.MME1Status", ParameterValue: "0"},
		{ParameterPath: "Device.Services.FAPService.1.CellConfig.LTE.EPC.MmePoolConfigParam.2.MME1Address", ParameterValue: "10.0.0.2"},
	}

	entries := AssembleMMEPool(params)
	assert.Len(t, entries, 2)

	// Find active entry
	var active, inactive *MMEEntry
	for i := range entries {
		if entries[i].Index == 1 {
			active = &entries[i]
		} else if entries[i].Index == 2 {
			inactive = &entries[i]
		}
	}
	assert.NotNil(t, active)
	assert.Equal(t, "active", active.Status)
	assert.Equal(t, "10.0.0.1", active.IP)
	assert.Equal(t, "46000", active.PLMNID)

	assert.NotNil(t, inactive)
	assert.Equal(t, "inactive", inactive.Status)
}

func TestAssembleMMEPool_Empty(t *testing.T) {
	entries := AssembleMMEPool(nil)
	assert.Empty(t, entries)
}

func TestAssembleLicenseDetail(t *testing.T) {
	params := []model.DeviceParameter{
		{ParameterPath: "Device.DeviceInfo.X_COM_LICENSE.LicenseCode", ParameterValue: "ABC-123"},
		{ParameterPath: "Device.DeviceInfo.X_COM_LICENSE.GenerateDate", ParameterValue: "2024-01-15"},
		{ParameterPath: "Device.DeviceInfo.X_COM_LICENSE.Capacity.1.Description", ParameterValue: "LTE Band 40"},
		{ParameterPath: "Device.DeviceInfo.X_COM_LICENSE.Capacity.1.State", ParameterValue: "1"},
		{ParameterPath: "Device.DeviceInfo.X_COM_LICENSE.Capacity.1.ValidPeriod", ParameterValue: "365"},
		{ParameterPath: "Device.DeviceInfo.X_COM_LICENSE.Capacity.1.RemainingPeriod", ParameterValue: "200"},
	}

	detail := AssembleLicenseDetail(params)
	assert.NotNil(t, detail)
	assert.Equal(t, "ABC-123", detail.Code)
	assert.Equal(t, "2024-01-15", detail.GenerateDate)
	assert.Len(t, detail.Capacities, 1)
	assert.Equal(t, "LTE Band 40", detail.Capacities[0].Description)
	assert.Equal(t, 200, detail.Capacities[0].RemainingPeriod)
}

func TestAssembleLicenseDetail_Empty(t *testing.T) {
	detail := AssembleLicenseDetail(nil)
	assert.Nil(t, detail)
}

func TestAssembleAntennaInfo(t *testing.T) {
	params := []model.DeviceParameter{
		{ParameterPath: "Device.Services.FAPService.1.CellConfig.LTE.RAN.AntennaInfo.Azimuth", ParameterValue: "120"},
		{ParameterPath: "Device.Services.FAPService.1.CellConfig.LTE.RAN.AntennaInfo.Gain", ParameterValue: "15"},
		{ParameterPath: "Device.Services.FAPService.1.CellConfig.LTE.RAN.AntennaInfo.Height", ParameterValue: "30"},
		{ParameterPath: "Device.Services.FAPService.1.CellConfig.LTE.RAN.AntennaInfo.Downtilt", ParameterValue: "6"},
	}

	info := AssembleAntennaInfo(params)
	assert.NotNil(t, info)
	assert.Equal(t, "120", info.Azimuth)
	assert.Equal(t, "15", info.Gain)
	assert.Equal(t, "30", info.Height)
	assert.Equal(t, "6", info.Downtilt)
}

func TestAssembleAntennaInfo_Empty(t *testing.T) {
	info := AssembleAntennaInfo(nil)
	assert.Nil(t, info)
}

func TestAssembleCells(t *testing.T) {
	params := []model.DeviceParameter{
		// Cell 1 (LTE)
		{ParameterPath: "Device.Services.FAPService.1.CellConfig.LTE.RAN.Common.CellIdentity", ParameterValue: "12345"},
		{ParameterPath: "Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.PhyCellID", ParameterValue: "100"},
		{ParameterPath: "Device.Services.FAPService.1.CellConfig.LTE.RAN.Common.EARFCNDL", ParameterValue: "38950"},
		{ParameterPath: "Device.Services.FAPService.1.FAPControl.LTE.OpState", ParameterValue: "true"},
		// Cell 2 (LTE CA)
		{ParameterPath: "Device.Services.FAPService.2.CellConfig.LTE.RAN.Common.CellIdentity", ParameterValue: "12346"},
		{ParameterPath: "Device.Services.FAPService.2.CellConfig.LTE.RAN.RF.PhyCellID", ParameterValue: "101"},
		{ParameterPath: "Device.Services.FAPService.2.FAPControl.LTE.OpState", ParameterValue: "false"},
	}

	cells := AssembleCells(params, 2)
	assert.Len(t, cells, 2)

	assert.Equal(t, 1, cells[0].Index)
	assert.Equal(t, "12345", cells[0].ECI)
	assert.Equal(t, "100", cells[0].PCI)
	assert.Equal(t, "38950", cells[0].FreqPoint)
	assert.Equal(t, "true", cells[0].OpState)

	assert.Equal(t, 2, cells[1].Index)
	assert.Equal(t, "12346", cells[1].ECI)
	assert.Equal(t, "false", cells[1].OpState)
}

func TestAssembleCells_SingleCell(t *testing.T) {
	cells := AssembleCells(nil, 1)
	assert.Len(t, cells, 1)
	assert.Equal(t, 1, cells[0].Index)
}

func TestExtractIndexAndField(t *testing.T) {
	tests := []struct {
		path      string
		marker    string
		wantIdx   int
		wantField string
	}{
		{
			path:      "Device.Services.FAPService.1.CellConfig.LTE.EPC.MmePoolConfigParam.3.MME1Status",
			marker:    "MmePoolConfigParam.",
			wantIdx:   3,
			wantField: "MME1Status",
		},
		{
			path:      "Device.DeviceInfo.X_COM_LICENSE.Capacity.12.RemainingPeriod",
			marker:    "Capacity.",
			wantIdx:   12,
			wantField: "RemainingPeriod",
		},
		{
			path:    "Device.DeviceInfo.SoftwareVersion",
			marker:  "Capacity.",
			wantIdx: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			idx, field := extractIndexAndField(tt.path, tt.marker)
			assert.Equal(t, tt.wantIdx, idx)
			if tt.wantIdx > 0 {
				assert.Equal(t, tt.wantField, field)
			}
		})
	}
}

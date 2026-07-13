package device

import (
	"testing"

	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/stretchr/testify/require"
)

func TestAssembleAntennaSectors_AssemblesUnnumberedAntennaInfo(t *testing.T) {
	sectors := AssembleAntennaSectors([]model.DeviceParameter{
		{ParameterPath: "Device.Services.FAPService.1.CellConfig.LTE.AntennaInfo.Azimuth", ParameterValue: "120"},
		{ParameterPath: "Device.Services.FAPService.1.CellConfig.LTE.AntennaInfo.Height", ParameterValue: "18"},
		{ParameterPath: "Device.Services.FAPService.1.CellConfig.LTE.AntennaInfo.Downtilt", ParameterValue: "6"},
		{ParameterPath: "Device.Services.FAPService.1.CellConfig.LTE.AntennaInfo.Beamwidth", ParameterValue: "65"},
		{ParameterPath: "Device.Services.FAPService.1.CellConfig.LTE.AntennaInfo.VerticalBeamwidth", ParameterValue: "8"},
		{ParameterPath: "Device.Services.FAPService.1.CellConfig.LTE.AntennaInfo.ElectronicDowntilt", ParameterValue: "true"},
	})

	require.Len(t, sectors, 1)
	sector := sectors[0]
	require.Equal(t, 1, sector.Number)
	require.Equal(t, 120.0, sector.Azimuth)
	require.Equal(t, 18.0, sector.AntennaHeight)
	require.Equal(t, 6.0, sector.MechanicalDowntilt)
	require.Equal(t, 65.0, sector.HorizontalBeamwidth)
	require.Equal(t, 8.0, sector.VerticalBeamwidth)
	require.Equal(t, "true", sector.ElectronicDowntilt)
	require.True(t, sector.CoverageAvailable)
	require.InDelta(t, 102.08, sector.NearRadiusMeters, 0.01)
	require.InDelta(t, 515.45, sector.FarRadiusMeters, 0.01)
	require.Empty(t, sector.MissingFields)
}

func TestAssembleAntennaSectors_MarksMissingAzimuthUnavailable(t *testing.T) {
	sectors := AssembleAntennaSectors([]model.DeviceParameter{
		{ParameterPath: "Device.Services.FAPService.1.CellConfig.LTE.AntennaInfo.Height", ParameterValue: "18"},
		{ParameterPath: "Device.Services.FAPService.1.CellConfig.LTE.AntennaInfo.Downtilt", ParameterValue: "6"},
		{ParameterPath: "Device.Services.FAPService.1.CellConfig.LTE.AntennaInfo.Beamwidth", ParameterValue: "65"},
		{ParameterPath: "Device.Services.FAPService.1.CellConfig.LTE.AntennaInfo.VerticalBeamwidth", ParameterValue: "12"},
	})

	require.Len(t, sectors, 1)
	require.False(t, sectors[0].DirectionAvailable)
	require.False(t, sectors[0].CoverageAvailable)
	require.Equal(t, []string{"azimuth"}, sectors[0].MissingFields)
}

func TestAssembleAntennaSectors_MarksInvalidNumericValueMissing(t *testing.T) {
	sectors := AssembleAntennaSectors([]model.DeviceParameter{
		{ParameterPath: "Device.DeviceInfo.AntennaInfo.Azimuth", ParameterValue: "not-a-number"},
		{ParameterPath: "Device.DeviceInfo.AntennaInfo.Height", ParameterValue: "18"},
		{ParameterPath: "Device.DeviceInfo.AntennaInfo.Downtilt", ParameterValue: "6"},
		{ParameterPath: "Device.DeviceInfo.AntennaInfo.Beamwidth", ParameterValue: "65"},
		{ParameterPath: "Device.DeviceInfo.AntennaInfo.VerticalBeamwidth", ParameterValue: "8"},
	})

	require.Len(t, sectors, 1)
	require.False(t, sectors[0].DirectionAvailable)
	require.False(t, sectors[0].CoverageAvailable)
	require.Equal(t, []string{"azimuth"}, sectors[0].MissingFields)
}

func TestAssembleAntennaSectors_NormalizesEditableFieldSources(t *testing.T) {
	sectors := AssembleAntennaSectors([]model.DeviceParameter{
		{ParameterPath: "Device.DeviceInfo.AntennaInfo.MechanicalDowntilt", ParameterValue: "6"},
		{ParameterPath: "Device.DeviceInfo.AntennaInfo.AntennaHeight", ParameterValue: "18"},
		{ParameterPath: "Device.DeviceInfo.AntennaInfo.HorizontalBeamwidth", ParameterValue: "65"},
	})

	require.Len(t, sectors, 1)
	require.Equal(t, "Device.DeviceInfo.AntennaInfo.MechanicalDowntilt", sectors[0].FieldSources["downtilt"])
	require.Equal(t, "Device.DeviceInfo.AntennaInfo.AntennaHeight", sectors[0].FieldSources["height"])
	require.Equal(t, "Device.DeviceInfo.AntennaInfo.HorizontalBeamwidth", sectors[0].FieldSources["beamwidth"])
}

func TestAssembleAntennaSectors_RejectsNonFiniteNumericValues(t *testing.T) {
	sectors := AssembleAntennaSectors([]model.DeviceParameter{
		{ParameterPath: "Device.DeviceInfo.AntennaInfo.Azimuth", ParameterValue: "NaN"},
		{ParameterPath: "Device.DeviceInfo.AntennaInfo.Height", ParameterValue: "+Inf"},
	})

	require.Len(t, sectors, 1)
	require.NotContains(t, sectors[0].FieldSources, "azimuth")
	require.NotContains(t, sectors[0].FieldSources, "height")
}

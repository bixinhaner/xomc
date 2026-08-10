package carrier

import (
	"testing"

	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/stretchr/testify/require"
)

func TestBuildGeofenceControlParameters(t *testing.T) {
	tests := []struct {
		name         string
		productClass string
		technology   model.Technology
		enabled      bool
		rfPath       string
	}{
		{
			name:         "BLN LTE",
			productClass: "BLN",
			technology:   model.TechLTE,
			enabled:      false,
			rfPath:       "Device.Services.FAPService.1.FAPControl.LTE.RFTxStatus",
		},
		{
			name:         "MLN LTE",
			productClass: "MLN",
			technology:   model.TechLTE,
			enabled:      true,
			rfPath:       "Device.Services.FAPService.1.FAPControl.LTE.RFTxStatus",
		},
		{
			name:         "BLQ LTE",
			productClass: "BLQ",
			technology:   model.TechLTE,
			enabled:      false,
			rfPath:       "Device.Services.FAPService.1.FAPControl.LTE.RFTxStatus",
		},
		{
			name:         "MLQ LTE",
			productClass: "MLQ",
			technology:   model.TechLTE,
			enabled:      true,
			rfPath:       "Device.Services.FAPService.1.FAPControl.LTE.RFTxStatus",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			parameters, err := BuildGeofenceControlParametersForInstances(
				test.productClass,
				test.technology,
				test.enabled,
				[]int{1},
			)

			require.NoError(t, err)
			require.Len(t, parameters, 2)
			require.Equal(t, "Device.Services.FAPService.Ipsec.IPSEC_ENABLE", parameters[0].Path)
			require.Equal(t, test.rfPath, parameters[1].Path)
			require.Equal(t, expectedControlValue(test.enabled), parameters[0].Value)
			require.Equal(t, expectedControlValue(test.enabled), parameters[1].Value)
		})
	}
}

func TestBuildGeofenceControlParametersForMBS31001(t *testing.T) {
	parameters, err := BuildGeofenceControlParametersForSnapshot(
		"FAP/mBS31001/DC",
		model.TechLTE,
		false,
		[]model.DeviceParameter{
			{ParameterPath: "Device.DeviceInfo.SAS.RadioEnable", ParameterValue: "true", Writable: true},
			{ParameterPath: "Device.FAP.Ipsec.1.TUNNEL_CONFIG_TUNNELENABLE", ParameterValue: "true", Writable: true},
			{ParameterPath: "Device.FAP.Ipsec.2.TUNNEL_CONFIG_TUNNELENABLE", ParameterValue: "true", Writable: true},
		},
	)

	require.NoError(t, err)
	require.Equal(t, []GeofenceControlParameter{
		{Path: "Device.DeviceInfo.SAS.RadioEnable", Value: "0"},
		{Path: "Device.FAP.Ipsec.1.TUNNEL_ENABLE", Value: "0"},
		{Path: "Device.FAP.Ipsec.2.TUNNEL_ENABLE", Value: "0"},
	}, parameters)
}

func TestBuildGeofenceControlParametersForMBS31001MultiIPSec(t *testing.T) {
	parameters, err := BuildGeofenceControlParametersForSnapshot(
		"FAP/mBS31001/DC",
		model.TechLTE,
		false,
		[]model.DeviceParameter{
			{ParameterPath: "Device.DeviceInfo.SAS.RadioEnable", ParameterValue: "true", Writable: true},
			{ParameterPath: "Device.Services.FAPService.1.CellConfig.LTE.MultiIpsecConfigParam.1.Enable", ParameterValue: "true", Writable: true},
			{ParameterPath: "Device.Services.FAPService.1.CellConfig.LTE.MultiIpsecConfigParam.2.Enable", ParameterValue: "true", Writable: true},
		},
	)

	require.NoError(t, err)
	require.Equal(t, []GeofenceControlParameter{
		{Path: "Device.DeviceInfo.SAS.RadioEnable", Value: "0"},
		{Path: "Device.Services.FAPService.1.CellConfig.LTE.MultiIpsecConfigParam.1.Enable", Value: "0"},
		{Path: "Device.Services.FAPService.1.CellConfig.LTE.MultiIpsecConfigParam.2.Enable", Value: "0"},
	}, parameters)
}

func TestBuildGeofenceControlParametersForMBS31001PrefersMappedRFOverSAS(t *testing.T) {
	parameters, err := BuildGeofenceControlParametersForSnapshot(
		"FAP/mBS31001/DC",
		model.TechLTE,
		false,
		[]model.DeviceParameter{
			{ParameterPath: "Device.DeviceInfo.SAS.RadioEnable", ParameterValue: "false", Writable: true},
			{ParameterPath: "Device.Services.FAPService.1.FAPControl.LTE.RFTxStatus", ParameterValue: "true", Writable: false},
			{ParameterPath: "Device.FAP.Ipsec.1.TUNNEL_CONFIG_TUNNELENABLE", ParameterValue: "true", Writable: true},
			{ParameterPath: "Device.FAP.Ipsec.2.TUNNEL_CONFIG_TUNNELENABLE", ParameterValue: "true", Writable: true},
		},
	)

	require.NoError(t, err)
	require.Equal(t, []GeofenceControlParameter{
		{Path: "Device.Services.FAPService.1.FAPControl.LTE.RFTxStatus", Value: "0"},
		{Path: "Device.FAP.Ipsec.1.TUNNEL_ENABLE", Value: "0"},
		{Path: "Device.FAP.Ipsec.2.TUNNEL_ENABLE", Value: "0"},
	}, parameters)
}

func TestBuildGeofenceControlParametersForSnapshotUsesParamModelRFMapping(t *testing.T) {
	parameters, err := BuildGeofenceControlParametersForSnapshotWithMappings(
		"FAP/MLN/DC",
		model.TechLTE,
		false,
		[]model.DeviceParameter{
			{ParameterPath: "Device.DeviceInfo.SAS.RadioEnable", ParameterValue: "false", Writable: true},
			{ParameterPath: "Device.Services.FAPService.2.CellConfig.LTE.RAN.RF.AdminCellState", ParameterValue: "true", Writable: false},
			{ParameterPath: "Device.FAP.Ipsec.1.TUNNEL_ENABLE", ParameterValue: "true", Writable: true},
		},
		[]GeofenceControlMapping{
			{
				StandardPath: "Device.DeviceInfo.SAS.RadioEnable",
				PrivatePath:  "Device.DeviceInfo.SAS.RadioEnable",
				EntryType:    "parameter",
				Access:       "READ_WRITE",
				IsActive:     true,
				IsSupported:  true,
			},
			{
				StandardPath: "Device.Services.FAPService.{i}.FAPControl.LTE.RFTxStatus",
				PrivatePath:  "Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.AdminCellState",
				EntryType:    "parameter",
				Access:       "READ_WRITE",
				IsActive:     true,
				IsSupported:  true,
			},
		},
	)

	require.NoError(t, err)
	require.Equal(t, []GeofenceControlParameter{
		{Path: "Device.Services.FAPService.2.FAPControl.LTE.RFTxStatus", Value: "0"},
		{Path: "Device.FAP.Ipsec.1.TUNNEL_ENABLE", Value: "0"},
	}, parameters)
}

func TestBuildGeofenceControlParametersFromSnapshotPrefersOneRFPathFamily(t *testing.T) {
	parameters, err := BuildGeofenceControlParametersForSnapshot(
		"BLQ",
		model.TechLTE,
		false,
		[]model.DeviceParameter{
			{ParameterPath: "Device.Services.FAPService.1.FAPControl.LTE.RFTxStatus", ParameterValue: "true", Writable: true},
			{ParameterPath: "Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.X_COM_RadioEnable", ParameterValue: "true", Writable: true},
			{ParameterPath: "Device.DeviceInfo.SAS.RadioEnable", ParameterValue: "true", Writable: true},
			{ParameterPath: "Device.FAP.Ipsec.1.TUNNEL_CONFIG_TUNNELENABLE", ParameterValue: "true", Writable: true},
		},
	)

	require.NoError(t, err)
	require.Equal(t, []GeofenceControlParameter{
		{Path: "Device.Services.FAPService.1.FAPControl.LTE.RFTxStatus", Value: "0"},
		{Path: "Device.FAP.Ipsec.1.TUNNEL_ENABLE", Value: "0"},
	}, parameters)
}

func TestBuildGeofenceControlParametersRejectsBMGSMReadOnlyState(t *testing.T) {
	_, err := BuildGeofenceControlParametersForInstances(
		"BM",
		model.TechGSM,
		false,
		[]int{1},
	)

	require.Error(t, err)
	require.Contains(t, err.Error(), "read-only")
}

func TestDetectGeofenceControlInstancesSortsWritableInstances(t *testing.T) {
	instances, err := DetectGeofenceControlInstances([]model.DeviceParameter{
		{ParameterPath: "Device.Services.FAPService.5.FAPControl.LTE.RFTxStatus", Writable: true},
		{ParameterPath: "Device.Services.FAPService.2.CellConfig.LTE.RAN.RF.X_COM_RadioEnable", Writable: true},
		{ParameterPath: "Device.Services.FAPService.3.CellConfig.LTE.RAN.RF.AdminCellState", Writable: false},
	}, "BLQ", model.TechLTE)

	require.NoError(t, err)
	require.Equal(t, []int{2, 5}, instances)
}

func TestDetectGeofenceControlInstancesAcceptsPrivateAdminStatePath(t *testing.T) {
	instances, err := DetectGeofenceControlInstances([]model.DeviceParameter{
		{ParameterPath: "Device.Services.FAPService.4.CellConfig.LTE.RAN.RF.AdminCellState", Writable: true},
	}, "BLN", model.TechLTE)

	require.NoError(t, err)
	require.Equal(t, []int{4}, instances)
}

func TestDetectGeofenceControlInstancesRejectsMissingWritableInstance(t *testing.T) {
	_, err := DetectGeofenceControlInstances([]model.DeviceParameter{
		{ParameterPath: "Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.X_COM_RadioEnable", Writable: false},
	}, "BLQ", model.TechLTE)

	require.Error(t, err)
	require.Contains(t, err.Error(), "no writable")
}

func TestDetectGeofenceControlInstancesForMBS31001UsesWritableIPSecTunnels(t *testing.T) {
	instances, err := DetectGeofenceControlInstances([]model.DeviceParameter{
		{ParameterPath: "Device.DeviceInfo.SAS.RadioEnable", Writable: true},
		{ParameterPath: "Device.FAP.Ipsec.1.TUNNEL_CONFIG_TUNNELENABLE", Writable: true},
		{ParameterPath: "Device.FAP.Ipsec.2.TUNNEL_CONFIG_TUNNELENABLE", Writable: true},
	}, "FAP/mBS31001/DC", model.TechLTE)

	require.NoError(t, err)
	require.Equal(t, []int{1, 2}, instances)
}

func expectedControlValue(enabled bool) string {
	if enabled {
		return "1"
	}
	return "0"
}

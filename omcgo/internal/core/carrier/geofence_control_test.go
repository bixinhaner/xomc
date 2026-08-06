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

func expectedControlValue(enabled bool) string {
	if enabled {
		return "1"
	}
	return "0"
}

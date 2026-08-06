package geofence

import (
	"testing"

	"github.com/omcgo/omcgo/internal/core/carrier"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/stretchr/testify/require"
)

func TestBuildGeofenceControlPlanChangesOnlyValuesOwnedByThisAction(t *testing.T) {
	targets := []carrier.GeofenceControlParameter{
		{Path: "Device.Services.FAPService.Ipsec.IPSEC_ENABLE", Value: "0"},
		{Path: "Device.Services.FAPService.1.FAPControl.LTE.RFTxStatus", Value: "0"},
	}
	snapshot := []model.DeviceParameter{
		{ParameterPath: targets[0].Path, ParameterValue: "false", Writable: true},
		{
			ParameterPath:  "Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.X_COM_RadioEnable",
			ParameterValue: "true", Writable: true,
		},
	}

	plan, err := buildGeofenceControlPlan(snapshot, targets)

	require.NoError(t, err)
	require.Equal(t, []ControlParameterState{
		{Path: targets[0].Path, Value: "0"},
		{Path: targets[1].Path, Value: "1"},
	}, plan.Before)
	require.Equal(t, []ControlParameterState{{Path: targets[1].Path, Value: "0"}}, plan.Requested)
}

func TestBuildGeofenceControlPlanRequiresWritableKnownCurrentValue(t *testing.T) {
	target := []carrier.GeofenceControlParameter{{
		Path: "Device.Services.FAPService.Ipsec.IPSEC_ENABLE", Value: "0",
	}}

	_, err := buildGeofenceControlPlan([]model.DeviceParameter{{
		ParameterPath: target[0].Path, ParameterValue: "unknown", Writable: true,
	}}, target)
	require.ErrorContains(t, err, "unsupported boolean value")

	_, err = buildGeofenceControlPlan([]model.DeviceParameter{{
		ParameterPath: target[0].Path, ParameterValue: "1", Writable: false,
	}}, target)
	require.ErrorContains(t, err, "not writable")
}

func TestRestoreTargetsRestoresOnlyParametersChangedByDeactivation(t *testing.T) {
	ipsec := "Device.Services.FAPService.Ipsec.IPSEC_ENABLE"
	rf := "Device.Services.FAPService.1.FAPControl.LTE.RFTxStatus"
	targets := restoreTargets(
		[]ControlParameterState{{Path: ipsec, Value: "0"}, {Path: rf, Value: "1"}},
		[]ControlParameterState{{Path: rf, Value: "0"}},
	)

	require.Equal(t, []carrier.GeofenceControlParameter{{Path: rf, Value: "1"}}, targets)
}

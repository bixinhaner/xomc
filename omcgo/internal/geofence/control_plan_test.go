package geofence

import (
	"fmt"
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

func TestBuildGeofenceControlPlanUsesParamModelAccessOverStaleSnapshotWritable(t *testing.T) {
	admin := carrier.GeofenceControlParameter{
		Path:         "Device.Services.FAPService.1.FAPControl.LTE.AdminState",
		Value:        "0",
		Role:         carrier.GeofenceRoleAdmin,
		AccessProven: true,
	}
	opState := carrier.GeofenceControlParameter{
		Path:         "Device.Services.FAPService.1.FAPControl.LTE.OpState",
		Value:        "0",
		Role:         carrier.GeofenceRoleOpState,
		AccessProven: true,
	}

	plan, err := buildGeofenceControlPlanWithTerminal([]model.DeviceParameter{
		{ParameterPath: admin.Path, ParameterValue: "1", Writable: false},
		{ParameterPath: opState.Path, ParameterValue: "1", Writable: true},
	}, []carrier.GeofenceControlParameter{admin}, []carrier.GeofenceControlParameter{opState}, false)

	require.NoError(t, err)
	require.Equal(t, []ControlParameterState{{
		Path: admin.Path, Value: "0", Role: carrier.GeofenceRoleAdmin,
	}}, plan.Requested)
	require.Equal(t, []ControlParameterState{{
		Path: opState.Path, Value: "0", Role: carrier.GeofenceRoleOpState,
	}}, plan.Terminals)
}

func TestBuildGeofenceControlPlanUsesAndPreservesPrivateObservedPath(t *testing.T) {
	target := carrier.GeofenceControlParameter{
		Path:              "Device.Services.FAPService.1.FAPControl.LTE.AdminState",
		SnapshotPath:      "Device.DeviceInfo.FAP_adminstate",
		Value:             "0",
		Role:              carrier.GeofenceRoleAdmin,
		AccessProven:      true,
		AppliesToAllCells: true,
	}
	plan, err := buildGeofenceControlPlan([]model.DeviceParameter{{
		ParameterPath: target.SnapshotPath, ParameterValue: "true", Writable: true,
	}}, []carrier.GeofenceControlParameter{target})

	require.NoError(t, err)
	require.Equal(t, []ControlParameterState{{
		Path: target.Path, ObservedPath: target.SnapshotPath,
		Value: "1", Role: carrier.GeofenceRoleAdmin, AppliesToAllCells: true,
	}}, plan.Before)
	require.Equal(t, []ControlParameterState{{
		Path: target.Path, Value: "0", Role: carrier.GeofenceRoleAdmin,
		AppliesToAllCells: true,
	}}, plan.Requested)
	require.Equal(t, []carrier.GeofenceControlParameter{{
		Path: target.Path, SnapshotPath: target.SnapshotPath,
		Value: "1", Role: carrier.GeofenceRoleAdmin, AccessProven: true,
		AppliesToAllCells: true,
	}}, restoreTargets(plan.Before, plan.Requested))
}

func TestBuildGeofenceControlPlanMatchesMBS31001PrivateSingleIPSecPath(t *testing.T) {
	target := carrier.GeofenceControlParameter{
		Path:  "Device.FAP.Ipsec.1.TUNNEL_ENABLE",
		Value: "0",
	}
	plan, err := buildGeofenceControlPlan([]model.DeviceParameter{
		{ParameterPath: "Device.FAP.Ipsec.1.TUNNEL_CONFIG_TUNNELENABLE", ParameterValue: "true", Writable: true},
	}, []carrier.GeofenceControlParameter{target})

	require.NoError(t, err)
	require.Equal(t, []ControlParameterState{{Path: target.Path, Value: "1"}}, plan.Before)
	require.Equal(t, []ControlParameterState{{Path: target.Path, Value: "0"}}, plan.Requested)
}

func TestBuildGeofenceControlPlanAllowsMappedStandardRFReadOnlySnapshot(t *testing.T) {
	target := carrier.GeofenceControlParameter{
		Path:  "Device.Services.FAPService.1.FAPControl.LTE.RFTxStatus",
		Value: "0",
	}
	plan, err := buildGeofenceControlPlan([]model.DeviceParameter{
		{ParameterPath: target.Path, ParameterValue: "true", Writable: false},
	}, []carrier.GeofenceControlParameter{target})

	require.NoError(t, err)
	require.Equal(t, []ControlParameterState{{Path: target.Path, Value: "1"}}, plan.Before)
	require.Equal(t, []ControlParameterState{{Path: target.Path, Value: "0"}}, plan.Requested)
}

func TestBuildGeofenceControlPlanRestoresTargetDespiteStaleCurrentValue(t *testing.T) {
	target := carrier.GeofenceControlParameter{
		Path:  "Device.Services.FAPService.1.FAPControl.LTE.RFTxStatus",
		Value: "1",
	}
	plan, err := buildGeofenceControlPlanWithRestore([]model.DeviceParameter{
		{ParameterPath: target.Path, ParameterValue: "true", Writable: false},
	}, []carrier.GeofenceControlParameter{target}, true)

	require.NoError(t, err)
	require.Equal(t, []ControlParameterState{{Path: target.Path, Value: "1"}}, plan.Before)
	require.Equal(t, []ControlParameterState{{Path: target.Path, Value: "1"}}, plan.Requested)
}

func TestRestoreTargetsRestoresOnlyParametersChangedByDeactivation(t *testing.T) {
	ipsec := "Device.Services.FAPService.Ipsec.IPSEC_ENABLE"
	rf := "Device.Services.FAPService.1.FAPControl.LTE.RFTxStatus"
	targets := restoreTargets(
		[]ControlParameterState{{Path: ipsec, Value: "0"}, {Path: rf, Value: "1"}},
		[]ControlParameterState{{Path: rf, Value: "0"}},
	)

	require.Equal(t, []carrier.GeofenceControlParameter{{
		Path: rf, Value: "1", AccessProven: true,
	}}, targets)
}

func TestRestoreTargetsPlacesMultiIPSecBeforeRF(t *testing.T) {
	multiIPSec := "Device.Services.FAPService.1.CellConfig.LTE.MultiIpsecConfigParam.1.Enable"
	rf := "Device.DeviceInfo.SAS.RadioEnable"
	targets := restoreTargets(
		[]ControlParameterState{{Path: rf, Value: "1"}, {Path: multiIPSec, Value: "1"}},
		[]ControlParameterState{{Path: rf, Value: "0"}, {Path: multiIPSec, Value: "0"}},
	)

	require.Equal(t, []carrier.GeofenceControlParameter{
		{Path: multiIPSec, Value: "1", AccessProven: true},
		{Path: rf, Value: "1", AccessProven: true},
	}, targets)
}

func TestRestoreTargetsUsesVerifiedOwnershipAndRestoresAdminLast(t *testing.T) {
	ipsec := ControlParameterState{Path: "Device.Services.FAPService.Ipsec.IPSEC_ENABLE", Value: "1", Role: carrier.GeofenceRoleIPSec}
	rf := ControlParameterState{Path: "Device.Services.FAPService.1.FAPControl.LTE.RFTxStatus", Value: "1", Role: carrier.GeofenceRoleRF}
	admin := ControlParameterState{Path: "Device.Services.FAPService.1.FAPControl.LTE.AdminState", Value: "1", Role: carrier.GeofenceRoleAdmin}
	opState := ControlParameterState{Path: "Device.Services.FAPService.1.FAPControl.LTE.OpState", Value: "1", Role: carrier.GeofenceRoleOpState}

	targets := restoreTargets(
		[]ControlParameterState{admin, opState, rf, ipsec},
		[]ControlParameterState{
			{Path: admin.Path, Value: "0", Role: admin.Role},
			{Path: rf.Path, Value: "0", Role: rf.Role},
			{Path: ipsec.Path, Value: "0", Role: ipsec.Role},
			{Path: opState.Path, Value: "0", Role: opState.Role},
		},
	)

	require.Equal(t, []carrier.GeofenceControlParameter{
		{Path: ipsec.Path, Value: "1", Role: carrier.GeofenceRoleIPSec, AccessProven: true},
		{Path: rf.Path, Value: "1", Role: carrier.GeofenceRoleRF, AccessProven: true},
		{Path: admin.Path, Value: "1", Role: carrier.GeofenceRoleAdmin, AccessProven: true},
	}, targets)
}

func TestRestoreTargetsSkipsChangedButUnverifiedControl(t *testing.T) {
	rf := ControlParameterState{Path: "Device.Services.FAPService.1.FAPControl.LTE.RFTxStatus", Value: "1", Role: carrier.GeofenceRoleRF}
	admin := ControlParameterState{Path: "Device.Services.FAPService.1.FAPControl.LTE.AdminState", Value: "1", Role: carrier.GeofenceRoleAdmin}

	targets := restoreTargets(
		[]ControlParameterState{rf, admin},
		[]ControlParameterState{{Path: rf.Path, Value: "0", Role: rf.Role}},
	)

	require.Equal(t, []carrier.GeofenceControlParameter{
		{Path: rf.Path, Value: "1", Role: carrier.GeofenceRoleRF, AccessProven: true},
	}, targets)
}

func TestFilterRestoreToEffectiveCellsUsesCurrentConfiguredCount(t *testing.T) {
	ipsec := carrier.GeofenceControlParameter{
		Path: "Device.Services.FAPService.Ipsec.IPSEC_ENABLE",
		Role: carrier.GeofenceRoleIPSec,
	}
	targets := []carrier.GeofenceControlParameter{ipsec}
	terminals := make([]ControlParameterState, 0, 3)
	for instance := 1; instance <= 3; instance++ {
		targets = append(targets, carrier.GeofenceControlParameter{
			Path: fmt.Sprintf("Device.Services.FAPService.%d.FAPControl.LTE.RFTxStatus", instance),
			Role: carrier.GeofenceRoleRF,
		})
		terminals = append(terminals, ControlParameterState{
			Path: fmt.Sprintf("Device.Services.FAPService.%d.FAPControl.LTE.OpState", instance),
			Role: carrier.GeofenceRoleOpState,
		})
	}
	filteredTargets, filteredTerminals := filterRestoreToEffectiveCells(
		[]model.DeviceParameter{{
			ParameterPath:  "Device.Services.FAPService.1.CellConfig.LTE.RAN.CA.PARAMS.NumOfCells",
			ParameterValue: "1",
		}},
		model.TechLTE,
		targets,
		terminals,
	)

	require.Equal(t, []carrier.GeofenceControlParameter{ipsec, targets[1]}, filteredTargets)
	require.Equal(t, terminals[:1], filteredTerminals)
}

func TestFilterRestoreToEffectiveCellsFallsBackToOwnedMaximum(t *testing.T) {
	targets := []carrier.GeofenceControlParameter{
		{Path: "Device.Services.FAPService.1.FAPControl.LTE.RFTxStatus", Role: carrier.GeofenceRoleRF},
		{Path: "Device.Services.FAPService.2.FAPControl.LTE.RFTxStatus", Role: carrier.GeofenceRoleRF},
	}
	terminals := []ControlParameterState{
		{Path: "Device.Services.FAPService.1.FAPControl.LTE.OpState", Role: carrier.GeofenceRoleOpState},
		{Path: "Device.Services.FAPService.2.FAPControl.LTE.OpState", Role: carrier.GeofenceRoleOpState},
	}

	filteredTargets, filteredTerminals := filterRestoreToEffectiveCells(
		nil, model.TechLTE, targets, terminals,
	)

	require.Equal(t, targets, filteredTargets)
	require.Equal(t, terminals, filteredTerminals)
}

func TestFilterRestoreToEffectiveCellsKeepsGlobalAdmin(t *testing.T) {
	globalAdmin := carrier.GeofenceControlParameter{
		Path: "Device.Services.FAPService.1.FAPControl.LTE.AdminState",
		Role: carrier.GeofenceRoleAdmin, AppliesToAllCells: true,
	}
	targets := []carrier.GeofenceControlParameter{
		globalAdmin,
		{Path: "Device.Services.FAPService.1.FAPControl.LTE.RFTxStatus", Role: carrier.GeofenceRoleRF},
		{Path: "Device.Services.FAPService.2.FAPControl.LTE.RFTxStatus", Role: carrier.GeofenceRoleRF},
	}
	terminals := []ControlParameterState{
		{Path: "Device.Services.FAPService.1.FAPControl.LTE.OpState", Role: carrier.GeofenceRoleOpState},
		{Path: "Device.Services.FAPService.2.FAPControl.LTE.OpState", Role: carrier.GeofenceRoleOpState},
	}

	filteredTargets, filteredTerminals := filterRestoreToEffectiveCells(
		[]model.DeviceParameter{
			{ParameterPath: "Device.Services.FAPService.1.FAPControl.LTE.InUse", ParameterValue: "0"},
			{ParameterPath: "Device.Services.FAPService.2.FAPControl.LTE.InUse", ParameterValue: "1"},
		},
		model.TechLTE, targets, terminals,
	)

	require.Equal(t, []carrier.GeofenceControlParameter{globalAdmin, targets[2]}, filteredTargets)
	require.Equal(t, terminals[1:], filteredTerminals)
}

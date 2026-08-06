package geofence

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestBuildCreateControlActionQueryPersistsOwnershipEvidence(t *testing.T) {
	now := time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC)
	bindingID := uuid.New()
	action := &ControlAction{
		ID: uuid.New(), ActionKey: "geofence:device:8:deactivate",
		DeviceID: uuid.New(), DeviceSN: "SN-1", BindingID: &bindingID,
		EffectiveStateVersion: 8, ActionType: ControlActionDeactivate,
		Status: ControlActionPending,
		BeforeState: []ControlParameterState{
			{Path: "Device.Services.FAPService.Ipsec.IPSEC_ENABLE", Value: "0"},
			{Path: "Device.Services.FAPService.1.FAPControl.LTE.RFTxStatus", Value: "1"},
		},
		RequestedState: []ControlParameterState{{
			Path: "Device.Services.FAPService.1.FAPControl.LTE.RFTxStatus", Value: "0",
		}},
		CreatedAt: now, UpdatedAt: now,
	}

	query, args, err := buildCreateControlActionQuery(action)

	require.NoError(t, err)
	require.Contains(t, query, "INSERT INTO geofence_control_actions")
	require.Contains(t, query, "ON CONFLICT (action_key) DO NOTHING")
	require.Contains(t, args, action.ActionKey)
	require.Contains(t, args, action.DeviceID)
	require.Contains(t, args, ControlActionDeactivate)
}

func TestBuildGetControlActionByKeyQueryUsesStableIdempotencyKey(t *testing.T) {
	query, args, err := buildGetControlActionByKeyQuery("geofence:device:8:deactivate")

	require.NoError(t, err)
	require.Contains(t, query, "FROM geofence_control_actions")
	require.Contains(t, query, "action_key =")
	require.Equal(t, []any{"geofence:device:8:deactivate"}, args)
}

func TestBuildListControlActionsByGeofenceIncludesBindingAndLifecycleActions(
	t *testing.T,
) {
	geofenceID := uuid.New()
	query, args, err := buildListControlActionsByGeofenceQuery(geofenceID, 100)

	require.NoError(t, err)
	require.Contains(t, query, "LEFT JOIN device_geofence_bindings")
	require.Contains(t, query, "LEFT JOIN geofence_control_actions parent")
	require.Contains(t, query, "action.geofence_id =")
	require.Contains(t, query, "binding.geofence_id =")
	require.Contains(t, query, "parent.geofence_id =")
	require.Contains(t, query, "parent_binding.geofence_id =")
	require.Contains(t, query, "ORDER BY action.created_at DESC")
	require.Equal(t, []any{
		geofenceID.String(), geofenceID.String(), geofenceID.String(), geofenceID.String(),
	}, args)
}

func TestBuildFindRecoverableDeactivationRequiresVerifiedUnrecoveredOwnership(t *testing.T) {
	deviceID := uuid.New()
	query, args, err := buildFindRecoverableDeactivationQuery(deviceID)

	require.NoError(t, err)
	require.Contains(t, query, "action.action_type =")
	require.Contains(t, query, "action.status =")
	require.Contains(t, query, "NOT EXISTS")
	require.Contains(t, query, "child.parent_action_id = action.id")
	require.Contains(t, args, deviceID.String())
	require.Contains(t, args, ControlActionVerified)
}

func TestBuildUpdateControlActionStatusMovesState(t *testing.T) {
	actionID := uuid.New()
	query, args, err := buildUpdateControlActionStatusQuery(actionID, ControlActionExecuting)

	require.NoError(t, err)
	require.Contains(t, query, "UPDATE geofence_control_actions")
	require.Contains(t, args, actionID.String())
	require.Contains(t, args, ControlActionExecuting)
}

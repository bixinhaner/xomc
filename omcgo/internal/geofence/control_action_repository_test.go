package geofence

import (
	"encoding/json"
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
	require.Contains(t, args, json.RawMessage("[]"))
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
	require.Contains(t, query, "action.contract_version >=")
	require.Contains(t, query, "jsonb_array_length(action.terminal_state) > 0")
	require.Contains(t, args, deviceID.String())
	require.Contains(t, args, ControlActionVerified)
}

func TestBuildListDueControlVerificationsUsesDurableSchedule(t *testing.T) {
	now := time.Date(2026, 8, 12, 18, 0, 0, 0, time.UTC)
	query, args, err := buildListDueControlVerificationsQuery(now, 100)

	require.NoError(t, err)
	require.Contains(t, query, "status =")
	require.Contains(t, query, "contract_version >=")
	require.Contains(t, query, "next_verification_at <=")
	require.Contains(t, query, "ORDER BY next_verification_at ASC")
	require.Contains(t, args, ControlActionVerifying)
	require.Contains(t, args, now)
}

func TestBuildUpdateControlActionStatusMovesState(t *testing.T) {
	actionID := uuid.New()
	query, args, err := buildUpdateControlActionStatusQuery(actionID, ControlActionExecuting)

	require.NoError(t, err)
	require.Contains(t, query, "UPDATE geofence_control_actions")
	require.Contains(t, query, "status IN")
	require.Contains(t, args, actionID.String())
	require.Contains(t, args, ControlActionExecuting)
	require.Contains(t, args, ControlActionPending)
	require.NotContains(t, args, ControlActionVerified)
}

func TestBuildBeginControlVerificationStartsWindowOnlyOnce(t *testing.T) {
	actionID := uuid.New()
	deadline := time.Date(2026, 8, 12, 18, 2, 0, 0, time.UTC)
	nextAt := time.Date(2026, 8, 12, 18, 0, 0, 0, time.UTC)

	query, args, err := buildBeginControlVerificationQuery(actionID, deadline, nextAt)

	require.NoError(t, err)
	require.Contains(t, query, "verification_deadline IS NULL")
	require.Contains(t, query, "status IN")
	require.Contains(t, args, actionID.String())
	require.Contains(t, args, deadline)
	require.Contains(t, args, nextAt)
	require.NotContains(t, args, ControlActionVerified)
}

func TestBuildCompleteControlVerificationCannotOverwriteTerminalAction(t *testing.T) {
	actionID := uuid.New()
	completedAt := time.Date(2026, 8, 12, 18, 2, 0, 0, time.UTC)

	query, args, err := buildCompleteControlVerificationQuery(
		actionID, nil, ControlActionFailed, "late failure", completedAt,
	)

	require.NoError(t, err)
	require.Contains(t, query, "status IN")
	require.Contains(t, args, ControlActionFailed)
	require.Contains(t, args, ControlActionVerifying)
	require.NotContains(t, args, ControlActionVerified)
}

func TestBuildScheduleControlVerificationRequiresCurrentAttempt(t *testing.T) {
	actionID := uuid.New()
	nextAt := time.Date(2026, 8, 12, 18, 0, 5, 0, time.UTC)

	query, args, err := buildScheduleControlVerificationQuery(
		actionID, nil, "OpState active", 2, nextAt,
	)

	require.NoError(t, err)
	require.Contains(t, query, "verification_attempt =")
	require.Contains(t, args, 2)
	require.Contains(t, args, 1)
	require.Contains(t, args, ControlActionVerifying)
}

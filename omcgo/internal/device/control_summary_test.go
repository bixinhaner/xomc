package device

import (
	"testing"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestBuildCurrentDeviceControlSummariesQueryIsBatchAndSourceAware(t *testing.T) {
	deviceIDs := []uuid.UUID{uuid.New(), uuid.New()}

	query, args, err := buildCurrentDeviceControlSummariesQuery(deviceIDs)

	require.NoError(t, err)
	require.Contains(t, query, "DISTINCT ON (deactivation.device_id)")
	require.Contains(t, query, "LEFT JOIN LATERAL")
	require.Contains(t, query, "device_geofence_bindings")
	require.Contains(t, query, "geofence_definitions")
	require.Contains(t, query, "deactivation.device_id IN")
	require.Contains(t, args, deviceIDs[0])
	require.Contains(t, args, deviceIDs[1])
	require.Contains(t, args, "activate")
	require.Contains(t, args, "deactivate")
}

func TestCurrentDeviceControlFilterConditionUsesLatestActionAndRecovery(t *testing.T) {
	source := DeviceControlSourceGeofence
	condition := currentDeviceControlFilterCondition(DeviceFilter{
		ControlSource: &source,
		ControlPhases: []string{DeviceControlPhaseDeactivated, DeviceControlPhaseRecoveryFailed},
	})

	query, args, err := sq.Select("d.id").From("devices d").Where(condition).PlaceholderFormat(sq.Dollar).ToSql()

	require.NoError(t, err)
	require.Contains(t, query, "EXISTS")
	require.Contains(t, query, "LEFT JOIN LATERAL")
	require.Contains(t, query, "latest.device_id = d.id")
	require.Contains(t, query, "recovery.status IN")
	require.Contains(t, args, "activate")
	require.Contains(t, args, "verified")
	require.Contains(t, args, "partial_failed")
}

func TestCurrentDeviceControlFilterConditionRejectsUnknownSource(t *testing.T) {
	source := "alarm"
	condition := currentDeviceControlFilterCondition(DeviceFilter{ControlSource: &source})

	query, _, err := sq.Select("d.id").From("devices d").Where(condition).ToSql()

	require.NoError(t, err)
	require.Contains(t, query, "WHERE FALSE")
}

func TestResolveDeviceControlPhase(t *testing.T) {
	tests := []struct {
		name               string
		deactivationStatus string
		recoveryStatus     *string
		wantPhase          string
		wantVisible        bool
	}{
		{name: "queued", deactivationStatus: "pending", wantPhase: DeviceControlPhaseDeactivating, wantVisible: true},
		{name: "readback", deactivationStatus: "verifying", wantPhase: DeviceControlPhaseVerifying, wantVisible: true},
		{name: "owned", deactivationStatus: "verified", wantPhase: DeviceControlPhaseDeactivated, wantVisible: true},
		{name: "partial", deactivationStatus: "partial_failed", wantPhase: DeviceControlPhasePartialFailed, wantVisible: true},
		{name: "failed", deactivationStatus: "failed", wantPhase: DeviceControlPhaseFailed, wantVisible: true},
		{name: "recovering", deactivationStatus: "verified", recoveryStatus: stringPointer("executing"), wantPhase: DeviceControlPhaseRecovering, wantVisible: true},
		{name: "recovery failed", deactivationStatus: "verified", recoveryStatus: stringPointer("failed"), wantPhase: DeviceControlPhaseRecoveryFailed, wantVisible: true},
		{name: "recovered", deactivationStatus: "verified", recoveryStatus: stringPointer("verified"), wantVisible: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			phase, visible := resolveDeviceControlPhase(test.deactivationStatus, test.recoveryStatus)
			require.Equal(t, test.wantPhase, phase)
			require.Equal(t, test.wantVisible, visible)
		})
	}
}

func stringPointer(value string) *string {
	return &value
}

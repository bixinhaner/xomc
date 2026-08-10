package geofence

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type controlActionListReaderStub struct {
	geofenceID uuid.UUID
	limit      int
	actions    []ControlAction
}

func (s *controlActionListReaderStub) ListByGeofence(
	_ context.Context,
	geofenceID uuid.UUID,
	limit int,
) ([]ControlAction, error) {
	s.geofenceID = geofenceID
	s.limit = limit
	return s.actions, nil
}

func TestListControlActionsReturnsDurableReadbackEvidence(t *testing.T) {
	geofenceID := uuid.New()
	reader := &controlActionListReaderStub{actions: []ControlAction{{
		ID: uuid.New(), ActionKey: "geofence:device:8:deactivate",
		DeviceSN: "SN-1", ActionType: ControlActionDeactivate,
		Status:         ControlActionPartialFailed,
		RequestedState: []ControlParameterState{{Path: "rf", Value: "0"}},
		VerifiedState:  []ControlParameterState{{Path: "rf", Value: "1"}},
		LastError:      "rf expected 0 got 1",
		CreatedAt:      time.Now().UTC(),
	}}}
	service := NewService(nil, nil)
	service.SetGeofenceControlActionReader(reader)

	actions, err := service.ListControlActions(context.Background(), geofenceID)

	require.NoError(t, err)
	require.Equal(t, geofenceID, reader.geofenceID)
	require.Equal(t, 100, reader.limit)
	require.Equal(t, ControlActionPartialFailed, actions[0].Status)
	require.Equal(t, "rf expected 0 got 1", actions[0].LastError)
	require.Equal(t, "1", actions[0].VerifiedState[0].Value)
}

package geofence

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

type GeofenceControlActionReader interface {
	ListByGeofence(context.Context, uuid.UUID, int) ([]ControlAction, error)
}

func (s *Service) ListControlActions(
	ctx context.Context,
	geofenceID uuid.UUID,
) ([]ControlAction, error) {
	if geofenceID == uuid.Nil {
		return nil, fmt.Errorf("geofence is required")
	}
	if s.controlActionReader == nil {
		return nil, fmt.Errorf("geofence control action reader is not configured")
	}
	actions, err := s.controlActionReader.ListByGeofence(ctx, geofenceID, 100)
	if err != nil {
		return nil, fmt.Errorf("list geofence control actions: %w", err)
	}
	return actions, nil
}

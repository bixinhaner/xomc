package device

import (
	"context"

	"github.com/google/uuid"
)

type LocationObservationRepository interface {
	UpsertLatest(ctx context.Context, deviceID uuid.UUID, observation ReportedLocation) error
	GetLatest(ctx context.Context, deviceID uuid.UUID) (*ReportedLocation, error)
}

type LocationSyncRepository interface {
	LocationObservationRepository
	Accept(ctx context.Context, deviceID uuid.UUID, reportedVersion int64) (*LocationSync, error)
}

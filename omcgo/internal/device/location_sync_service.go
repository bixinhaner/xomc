package device

import (
	"context"

	"github.com/google/uuid"
)

type LocationSyncService struct {
	repo LocationSyncRepository
}

func NewLocationSyncService(repo LocationSyncRepository) *LocationSyncService {
	return &LocationSyncService{repo: repo}
}

func (s *LocationSyncService) Accept(ctx context.Context, deviceID uuid.UUID, reportedVersion int64) (*LocationSync, error) {
	return s.repo.Accept(ctx, deviceID, reportedVersion)
}

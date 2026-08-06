package device

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
)

type fakeLocationSyncRepository struct {
	result     *LocationSync
	gotID      uuid.UUID
	gotVersion int64
	err        error
}

func (f *fakeLocationSyncRepository) SaveLatestWithOutbox(
	context.Context,
	uuid.UUID,
	ReportedLocation,
) (LocationObservationWriteResult, error) {
	return LocationObservationWriteResult{}, nil
}

func (f *fakeLocationSyncRepository) GetLatest(context.Context, uuid.UUID) (*ReportedLocation, error) {
	return nil, nil
}

func (f *fakeLocationSyncRepository) Accept(_ context.Context, deviceID uuid.UUID, version int64) (*LocationSync, error) {
	f.gotID = deviceID
	f.gotVersion = version
	return f.result, f.err
}

func TestLocationSyncServiceAcceptUsesReportedVersion(t *testing.T) {
	deviceID := uuid.New()
	repo := &fakeLocationSyncRepository{result: &LocationSync{Status: LocationSyncInSync}}
	service := NewLocationSyncService(repo)

	result, err := service.Accept(context.Background(), deviceID, 7)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != LocationSyncInSync || repo.gotID != deviceID || repo.gotVersion != 7 {
		t.Fatalf("unexpected accept call/result: result=%#v id=%s version=%d", result, repo.gotID, repo.gotVersion)
	}
}

func TestLocationSyncServiceAcceptPropagatesVersionConflict(t *testing.T) {
	conflict := errors.New("reported location version conflict")
	repo := &fakeLocationSyncRepository{err: conflict}
	service := NewLocationSyncService(repo)

	_, err := service.Accept(context.Background(), uuid.New(), 2)
	if !errors.Is(err, conflict) {
		t.Fatalf("expected version conflict to propagate, got %v", err)
	}
}

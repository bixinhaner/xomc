package geofence

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/stretchr/testify/require"
)

func TestBuildListVersionsQueryOrdersNewestFirst(t *testing.T) {
	geofenceID := uuid.MustParse(
		"11111111-1111-1111-1111-111111111111",
	)

	query, args, err := buildListVersionsQuery(geofenceID)

	require.NoError(t, err)
	require.Contains(t, query, "WHERE geofence_id = $1")
	require.Contains(t, query, "ORDER BY version DESC")
	require.Equal(t, []any{geofenceID.String()}, args)
}

func (f *fakeRepository) ListVersions(
	_ context.Context,
	geofenceID uuid.UUID,
) ([]Version, error) {
	f.listVersionsGeofenceID = geofenceID
	return f.versions, f.listVersionsErr
}

func TestServiceListVersionsRequiresExistingDefinition(t *testing.T) {
	repository := &fakeRepository{}

	_, err := NewService(repository, nil).
		ListVersions(context.Background(), uuid.New())

	require.ErrorIs(t, err, commonerrors.ErrNotFound)
	require.Equal(t, uuid.Nil, repository.listVersionsGeofenceID)
}

func TestServiceListVersionsReturnsRepositoryHistory(t *testing.T) {
	geofenceID := uuid.New()
	repository := &fakeRepository{
		definition: &Definition{ID: geofenceID},
		versions: []Version{
			{ID: uuid.New(), GeofenceID: geofenceID, Version: 2},
			{ID: uuid.New(), GeofenceID: geofenceID, Version: 1},
		},
	}

	got, err := NewService(repository, nil).
		ListVersions(context.Background(), geofenceID)

	require.NoError(t, err)
	require.Equal(t, repository.versions, got)
	require.Equal(t, geofenceID, repository.listVersionsGeofenceID)
}

func TestServiceListVersionsWrapsRepositoryError(t *testing.T) {
	geofenceID := uuid.New()
	databaseErr := errors.New("database unavailable")
	repository := &fakeRepository{
		definition:      &Definition{ID: geofenceID},
		listVersionsErr: databaseErr,
	}

	_, err := NewService(repository, nil).
		ListVersions(context.Background(), geofenceID)

	require.ErrorIs(t, err, databaseErr)
	require.Contains(t, err.Error(), "list geofence versions")
}

package geofence

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func (f *fakeRepository) ListMapDefinitions(
	_ context.Context,
	filter MapDefinitionFilter,
) ([]MapDefinition, error) {
	f.mapDefinitionFilter = filter
	return f.mapDefinitions, f.mapDefinitionErr
}

func TestServiceListMapDefinitionsDelegatesValidatedFilter(t *testing.T) {
	repository := &fakeRepository{
		mapDefinitions: []MapDefinition{{
			Definition: Definition{Name: "west"},
		}},
	}
	filter := MapDefinitionFilter{
		Bounds: &MapBounds{
			MinLongitude: 120,
			MaxLongitude: 121,
			MinLatitude:  30,
			MaxLatitude:  31,
		},
		Carrier: "cmcc",
		Status:  DefinitionStatusEnabled,
		Name:    "west",
	}

	got, err := NewService(repository, nil).
		ListMapDefinitions(context.Background(), filter)

	require.NoError(t, err)
	require.Equal(t, repository.mapDefinitions, got)
	require.Equal(t, filter, repository.mapDefinitionFilter)
}

func TestServiceListMapDefinitionsWrapsRepositoryError(t *testing.T) {
	databaseErr := errors.New("database unavailable")
	repository := &fakeRepository{mapDefinitionErr: databaseErr}

	_, err := NewService(repository, nil).
		ListMapDefinitions(context.Background(), MapDefinitionFilter{})

	require.ErrorIs(t, err, databaseErr)
	require.Contains(t, err.Error(), "list geofence map definitions")
}

package geofence

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/stretchr/testify/require"
)

func TestNormalizeBindingDetailFilterDefaultsPagination(t *testing.T) {
	got, err := normalizeBindingDetailFilter(BindingDetailFilter{
		GeofenceID: uuid.New(),
	})

	require.NoError(t, err)
	require.Equal(t, 1, got.Page)
	require.Equal(t, DefaultBindingPageSize, got.PageSize)
}

func TestNormalizeBindingDetailFilterRejectsCurrentScopeWithExplicitStatus(
	t *testing.T,
) {
	_, err := normalizeBindingDetailFilter(BindingDetailFilter{
		GeofenceID:  uuid.New(),
		Status:      BindingStatusActive,
		CurrentOnly: true,
	})

	require.ErrorIs(t, err, commonerrors.ErrInvalidInput)
}

func TestNormalizeBindingDetailFilterRejectsInvalidInputs(t *testing.T) {
	tests := []struct {
		name   string
		filter BindingDetailFilter
		target error
	}{
		{
			name:   "missing geofence",
			filter: BindingDetailFilter{},
			target: commonerrors.ErrInvalidInput,
		},
		{
			name: "empty visibility",
			filter: BindingDetailFilter{
				GeofenceID:    uuid.New(),
				VisibleGroups: []uuid.UUID{},
			},
			target: commonerrors.ErrForbidden,
		},
		{
			name: "unknown status",
			filter: BindingDetailFilter{
				GeofenceID: uuid.New(),
				Status:     BindingStatus("unknown"),
			},
			target: commonerrors.ErrInvalidInput,
		},
		{
			name: "page size above limit",
			filter: BindingDetailFilter{
				GeofenceID: uuid.New(),
				PageSize:   MaxBindingPageSize + 1,
			},
			target: commonerrors.ErrInvalidInput,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := normalizeBindingDetailFilter(tt.filter)
			require.ErrorIs(t, err, tt.target)
		})
	}
}

func TestBuildListBindingDetailsQueryAppliesVisibilityAndKeyword(
	t *testing.T,
) {
	groupID := uuid.MustParse(
		"22222222-2222-2222-2222-222222222222",
	)
	filter := BindingDetailFilter{
		GeofenceID: uuid.MustParse(
			"11111111-1111-1111-1111-111111111111",
		),
		Status:        BindingStatusActive,
		Keyword:       "station",
		Page:          2,
		PageSize:      50,
		VisibleGroups: []uuid.UUID{groupID},
	}

	query, args, err := buildListBindingDetailsQuery(filter)

	require.NoError(t, err)
	require.Contains(t, query, "device_group_members")
	require.Contains(t, query, "d.deleted_at IS NULL")
	require.Contains(t, query, "d.serial_number ILIKE")
	require.Contains(t, query, "device_info.device_name ILIKE")
	require.Contains(t, query, "device_geofence_states binding_state")
	require.Contains(t, query, "device_geofence_effective_states effective_state")
	require.Contains(t, query, "binding_state.confirmed_state")
	require.Contains(t, query, "effective_state.evaluation_health")
	require.NotContains(t, query, "geofence_control_actions")
	require.NotContains(t, query, "geofence_control_steps")
	require.Contains(t, query, "COUNT(*) OVER()")
	require.Contains(t, query, "LIMIT 50 OFFSET 50")
	require.Contains(t, args, groupID)
	require.Contains(t, args, "%station%")
}

func TestBuildListBindingDetailsQueryDoesNotFilterSuperAdministrator(
	t *testing.T,
) {
	query, args, err := buildListBindingDetailsQuery(
		BindingDetailFilter{
			GeofenceID: uuid.MustParse(
				"11111111-1111-1111-1111-111111111111",
			),
			Page:     1,
			PageSize: 50,
		},
	)

	require.NoError(t, err)
	require.NotContains(t, query, "d.id IN (SELECT device_id")
	require.NotContains(t, query, "WHERE FALSE")
	require.Len(t, args, 1)
}

func TestBuildListBindingDetailsQueryFiltersCurrentBindings(t *testing.T) {
	query, args, err := buildListBindingDetailsQuery(BindingDetailFilter{
		GeofenceID:  uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		CurrentOnly: true,
		Page:        1,
		PageSize:    50,
	})

	require.NoError(t, err)
	require.Contains(t, query, "binding.status IN")
	require.Contains(t, args, BindingStatusActive)
	require.Contains(t, args, BindingStatusSuspended)
	require.NotContains(t, args, BindingStatusRemoved)
}

func (f *fakeRepository) ListBindingDetails(
	_ context.Context,
	filter BindingDetailFilter,
) (BindingDetailPage, error) {
	f.bindingDetailFilter = filter
	return f.bindingDetailPage, f.bindingDetailErr
}

func TestServiceListBindingDetailsRequiresExistingDefinition(t *testing.T) {
	repository := &fakeRepository{}

	_, err := NewService(repository, nil).ListBindingDetails(
		context.Background(),
		BindingDetailFilter{GeofenceID: uuid.New()},
	)

	require.ErrorIs(t, err, commonerrors.ErrNotFound)
	require.Equal(t, uuid.Nil, repository.bindingDetailFilter.GeofenceID)
}

func TestServiceListBindingDetailsReturnsRepositoryPage(t *testing.T) {
	geofenceID := uuid.New()
	repository := &fakeRepository{
		definition: &Definition{ID: geofenceID},
		bindingDetailPage: BindingDetailPage{
			Items: []BindingDetail{{
				Binding:  Binding{ID: uuid.New(), GeofenceID: geofenceID},
				DeviceSN: "SN001",
				Evaluation: BindingEvaluationDetail{
					ConfirmedState:   ConfirmedStateInside,
					EvaluationHealth: "healthy",
				},
			}},
			Total: 1, Page: 1, PageSize: 50,
		},
	}

	got, err := NewService(repository, nil).ListBindingDetails(
		context.Background(),
		BindingDetailFilter{GeofenceID: geofenceID},
	)

	require.NoError(t, err)
	require.Equal(t, repository.bindingDetailPage, got)
	require.Equal(t, 1, repository.bindingDetailFilter.Page)
	require.Equal(
		t,
		DefaultBindingPageSize,
		repository.bindingDetailFilter.PageSize,
	)
}

func TestServiceListBindingDetailsWrapsRepositoryError(t *testing.T) {
	geofenceID := uuid.New()
	databaseErr := errors.New("database unavailable")
	repository := &fakeRepository{
		definition:       &Definition{ID: geofenceID},
		bindingDetailErr: databaseErr,
	}

	_, err := NewService(repository, nil).ListBindingDetails(
		context.Background(),
		BindingDetailFilter{GeofenceID: geofenceID},
	)

	require.ErrorIs(t, err, databaseErr)
	require.Contains(t, err.Error(), "list geofence binding details")
}

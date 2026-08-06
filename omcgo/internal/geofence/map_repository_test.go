package geofence

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestBuildListMapDefinitionsQueryFiltersPublishedBBoxesByIntersection(
	t *testing.T,
) {
	query, args, err := buildListMapDefinitionsQuery(MapDefinitionFilter{
		Bounds: &MapBounds{
			MinLongitude: 120,
			MaxLongitude: 121,
			MinLatitude:  30,
			MaxLatitude:  31,
		},
		Carrier: "cmcc",
		Status:  DefinitionStatusEnabled,
		Name:    "west",
	})

	require.NoError(t, err)
	require.Contains(t, query, "version.bbox_min_longitude <= ")
	require.Contains(t, query, "version.bbox_max_longitude >= ")
	require.Contains(t, query, "version.bbox_min_latitude <= ")
	require.Contains(t, query, "version.bbox_max_latitude >= ")
	require.Contains(t, query, "definition.current_version_id IS NOT NULL")
	require.Equal(t, []any{
		"cmcc",
		DefinitionStatusEnabled,
		"%west%",
		float64(121),
		float64(120),
		float64(31),
		float64(30),
	}, args)
}

func TestBuildListMapDefinitionsQueryKeepsDraftsWithoutBounds(
	t *testing.T,
) {
	query, args, err := buildListMapDefinitionsQuery(MapDefinitionFilter{})

	require.NoError(t, err)
	require.Contains(t, query, "LEFT JOIN geofence_versions version")
	require.NotContains(t, query, "definition.current_version_id IS NOT NULL")
	require.Contains(t, query, "definition.status <> ")
	require.Equal(t, []any{DefinitionStatusArchived}, args)
}

func TestBuildListMapDefinitionsQueryAllowsExplicitArchiveFilter(
	t *testing.T,
) {
	query, args, err := buildListMapDefinitionsQuery(MapDefinitionFilter{
		Status: DefinitionStatusArchived,
	})

	require.NoError(t, err)
	require.NotContains(t, query, "definition.status <> ")
	require.Contains(t, query, "definition.status = ")
	require.Equal(t, []any{DefinitionStatusArchived}, args)
}

func TestBuildListMapDefinitionsQueryScopesCarrierToVisibleDeviceGroups(
	t *testing.T,
) {
	groupID := uuid.MustParse("11111111-1111-1111-1111-111111111111")

	query, args, err := buildListMapDefinitionsQuery(MapDefinitionFilter{
		VisibleGroups: []uuid.UUID{groupID},
	})

	require.NoError(t, err)
	require.Contains(t, query, "scope_device.carrier")
	require.Contains(t, query, "device_group_members")
	require.Contains(t, query, "$1")
	require.Contains(t, query, "$2")
	require.Equal(
		t,
		[]any{groupID, DefinitionStatusArchived},
		args,
	)
}

func TestBuildListMapDefinitionsQueryNumbersNestedScopeArgumentsOnce(
	t *testing.T,
) {
	groupID := uuid.MustParse("11111111-1111-1111-1111-111111111111")

	query, args, err := buildListMapDefinitionsQuery(MapDefinitionFilter{
		Carrier:       "cmcc",
		Status:        DefinitionStatusEnabled,
		VisibleGroups: []uuid.UUID{groupID},
	})

	require.NoError(t, err)
	require.Contains(t, query, "$1")
	require.Contains(t, query, "$2")
	require.Contains(t, query, "$3")
	require.Equal(
		t,
		[]any{groupID, "cmcc", DefinitionStatusEnabled},
		args,
	)
}

func TestBuildListMapDefinitionsQueryFailsClosedWithoutVisibleGroups(
	t *testing.T,
) {
	query, _, err := buildListMapDefinitionsQuery(MapDefinitionFilter{
		VisibleGroups: []uuid.UUID{},
	})

	require.NoError(t, err)
	require.Contains(t, query, "WHERE FALSE")
}

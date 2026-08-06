package geofence

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newUniqueViolation(constraint string) error {
	return &pgconn.PgError{Code: "23505", ConstraintName: constraint}
}

func TestPgRepository_BuildCreateDefinitionWithDraftQueriesMapsImmutableSnapshot(t *testing.T) {
	now := time.Date(2026, 7, 29, 10, 0, 0, 0, time.UTC)
	actorID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	definition := &Definition{
		ID:        uuid.MustParse("22222222-2222-2222-2222-222222222222"),
		Name:      "杭州测试围栏",
		Carrier:   "cmcc",
		RuleType:  RuleTypePolygonAllowZone,
		Status:    DefinitionStatusDraft,
		CreatedBy: actorID,
		UpdatedBy: actorID,
		CreatedAt: now,
		UpdatedAt: now,
	}
	version := &Version{
		ID:           uuid.MustParse("33333333-3333-3333-3333-333333333333"),
		GeofenceID:   definition.ID,
		Version:      1,
		Status:       VersionStatusDraft,
		GeometryJSON: json.RawMessage(`{"type":"Polygon","coordinates":[[[121.1,31.1],[121.2,31.1],[121.2,31.2],[121.1,31.1]]]}`),
		BoundingBox: BoundingBox{
			MinLongitude: 121.1,
			MinLatitude:  31.1,
			MaxLongitude: 121.2,
			MaxLatitude:  31.2,
		},
		PolicyJSON: json.RawMessage(`{"exit_action":"notify_only"}`),
		CreatedBy:  actorID,
		CreatedAt:  now,
	}

	definitionSQL, definitionArgs, versionSQL, versionArgs, err :=
		buildCreateDefinitionWithDraftQueries(definition, version)
	require.NoError(t, err)
	assert.Contains(t, definitionSQL, "INSERT INTO geofence_definitions")
	assert.Contains(t, definitionSQL, "owner_device_id")
	assert.Contains(t, versionSQL, "INSERT INTO geofence_versions")
	assert.Contains(t, versionSQL, "geometry_json")
	assert.Contains(t, versionSQL, "bbox_min_longitude")
	assert.Contains(t, versionSQL, "policy_json")
	assert.Contains(t, definitionArgs, definition.Name)
	assert.Contains(t, versionArgs, version.GeometryJSON)
	assert.Contains(t, versionArgs, version.BoundingBox.MaxLatitude)
}

func TestIsActiveBindingConflict_OnlyMatchesNamedConstraint(t *testing.T) {
	assert.True(t, isActiveBindingConflict(newUniqueViolation(
		"uq_device_geofence_active_rule_type",
	)))
	assert.False(t, isActiveBindingConflict(newUniqueViolation(
		"unrelated_unique_constraint",
	)))
}

func TestBuildListDefinitionsQueryScopesCarrierToVisibleDeviceGroups(
	t *testing.T,
) {
	groupID := uuid.MustParse("11111111-1111-1111-1111-111111111111")

	query, args, err := buildListDefinitionsQuery(DefinitionFilter{
		VisibleGroups: []uuid.UUID{groupID},
	})

	require.NoError(t, err)
	require.Contains(t, query, "scope_device.carrier")
	require.Contains(t, query, "device_group_members")
	require.Contains(t, args, groupID)
}

func TestBuildCarrierVisibilityQueryUsesCanonicalDeviceScope(t *testing.T) {
	groupID := uuid.MustParse("11111111-1111-1111-1111-111111111111")

	query, args, err := buildCarrierVisibilityQuery(
		"cmcc",
		[]uuid.UUID{groupID},
	)

	require.NoError(t, err)
	require.Contains(t, query, "FROM devices scope_device")
	require.Contains(t, query, "device_group_members")
	require.Contains(t, args, "cmcc")
	require.Contains(t, args, groupID)
}

func TestBuildDeviceVisibilityQueryUsesCanonicalDeviceScope(t *testing.T) {
	groupID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	deviceID := uuid.MustParse("22222222-2222-2222-2222-222222222222")

	query, args, err := buildDeviceVisibilityQuery(
		deviceID,
		[]uuid.UUID{groupID},
	)

	require.NoError(t, err)
	require.Contains(t, query, "FROM devices scope_device")
	require.Contains(t, query, "device_group_members")
	require.Contains(t, args, deviceID.String())
	require.Contains(t, args, groupID)
}

func TestBuildPublishVersionQueriesPreservesLockAndStateTransitionSemantics(t *testing.T) {
	geofenceID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	versionID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	currentVersionID := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	actorID := uuid.MustParse("44444444-4444-4444-4444-444444444444")
	publishedAt := time.Date(2026, 7, 30, 4, 0, 0, 0, time.UTC)

	queries, err := buildPublishVersionQueries(
		geofenceID,
		versionID,
		&currentVersionID,
		actorID,
		publishedAt,
	)

	require.NoError(t, err)
	require.Len(t, queries, 5)
	assert.Contains(t, queries[0].sql, "FROM geofence_definitions")
	assert.Contains(t, queries[0].sql, "FOR UPDATE")
	assert.Contains(t, queries[1].sql, "FROM geofence_versions")
	assert.Contains(t, queries[1].sql, "FOR UPDATE")
	assert.Contains(t, queries[2].sql, "UPDATE geofence_versions")
	assert.Contains(t, queries[2].args, currentVersionID.String())
	assert.Contains(t, queries[3].sql, "published_by")
	assert.Contains(t, queries[4].sql, "UPDATE geofence_definitions")
	assert.Contains(t, queries[4].args, geofenceID.String())
}

func TestBuildBindingStateQueriesUsesIdempotentEffectiveStateUpsert(t *testing.T) {
	binding := &Binding{
		ID:       uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		DeviceID: uuid.MustParse("22222222-2222-2222-2222-222222222222"),
	}

	bindingState, effectiveState, err := buildBindingStateQueries(binding)

	require.NoError(t, err)
	assert.Contains(t, bindingState.sql, "INSERT INTO device_geofence_states")
	assert.Contains(t, bindingState.args, binding.ID)
	assert.Contains(t, effectiveState.sql, "INSERT INTO device_geofence_effective_states")
	assert.Contains(t, effectiveState.sql, "ON CONFLICT (device_id) DO UPDATE")
	assert.Contains(t, effectiveState.args, binding.DeviceID)
}

func TestBuildCreateDraftVersionQueriesLockDefinitionAndAllocateNextVersion(
	t *testing.T,
) {
	geofenceID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	version := &Version{
		ID:         uuid.MustParse("22222222-2222-2222-2222-222222222222"),
		GeofenceID: geofenceID,
		Status:     VersionStatusDraft,
		GeometryJSON: json.RawMessage(
			`{"type":"Polygon","coordinates":[[[120,30],[121,30],[121,31],[120,30]]]}`,
		),
		BoundingBox: BoundingBox{
			MinLongitude: 120, MinLatitude: 30,
			MaxLongitude: 121, MaxLatitude: 31,
		},
		PolicyJSON: json.RawMessage(`{"exit_action":"notify_only"}`),
		CreatedBy:  uuid.MustParse("33333333-3333-3333-3333-333333333333"),
		CreatedAt:  time.Date(2026, 7, 30, 14, 0, 0, 0, time.UTC),
	}

	queries, err := buildCreateDraftVersionQueries(version)

	require.NoError(t, err)
	require.Len(t, queries, 2)
	assert.Contains(t, queries[0].sql, "FROM geofence_definitions")
	assert.Contains(t, queries[0].sql, "FOR UPDATE")
	assert.Contains(t, queries[1].sql, "INSERT INTO geofence_versions")
	assert.Contains(t, queries[1].sql, "MAX(version)")
	assert.Contains(t, queries[1].sql, "RETURNING version")
	assert.Contains(t, queries[1].args, geofenceID)
	assert.NotContains(t, queries[1].sql, "UPDATE geofence_definitions")
}

func TestBuildLifecycleQueriesCountImpactAndGuardArchive(t *testing.T) {
	geofenceID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	actorID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	now := time.Date(2026, 7, 30, 15, 0, 0, 0, time.UTC)

	impactSQL, impactArgs, err := buildLifecycleImpactQuery(geofenceID)
	require.NoError(t, err)
	assert.Contains(t, impactSQL, "COUNT(")
	assert.Contains(t, impactSQL, "COUNT(DISTINCT")
	assert.Contains(t, impactSQL, "device_geofence_bindings")
	assert.Contains(t, impactArgs, geofenceID.String())

	lockDefinitionSQL, _, err := buildLockLifecycleDefinitionQuery(geofenceID)
	require.NoError(t, err)
	assert.Contains(t, lockDefinitionSQL, "FOR UPDATE")

	lockBindingsSQL, _, err := buildLockLifecycleBindingsQuery(geofenceID)
	require.NoError(t, err)
	assert.Contains(t, lockBindingsSQL, "ORDER BY id")
	assert.Contains(t, lockBindingsSQL, "FOR UPDATE")

	deactivationSQL, deactivationArgs, err :=
		buildLifecycleDeactivationCandidatesQuery(geofenceID)
	require.NoError(t, err)
	assert.Contains(t, deactivationSQL, "JOIN device_geofence_states s")
	assert.Contains(t, deactivationSQL, "s.confirmed_state =")
	assert.Contains(t, deactivationSQL, "b.status =")
	assert.Contains(t, deactivationSQL, "gv.policy_json ->> 'exit_action'")
	assert.Contains(t, deactivationSQL, "FOR UPDATE OF s")
	assert.Contains(t, deactivationArgs, geofenceID.String())
	assert.Contains(t, deactivationArgs, ConfirmedStateInside)
	assert.Contains(t, deactivationArgs, BindingStatusActive)
	assert.Contains(t, deactivationArgs, string(ActionLevelDeactivate))

	updateSQL, updateArgs, err := buildUpdateDefinitionStatusQuery(
		geofenceID,
		DefinitionStatusArchived,
		actorID,
		now,
	)
	require.NoError(t, err)
	assert.Contains(t, updateSQL, "UPDATE geofence_definitions")
	assert.Contains(t, updateArgs, DefinitionStatusArchived)

	removeSQL, removeArgs, err := buildArchiveBindingsQuery(
		geofenceID,
		actorID,
		"operator archive",
		now,
	)
	require.NoError(t, err)
	assert.Contains(t, removeSQL, "UPDATE device_geofence_bindings")
	assert.Contains(t, removeSQL, "removed_by")
	assert.Contains(t, removeArgs, "operator archive")
}

func TestBuildBindingLifecycleQueriesLockParentAndPersistRemovalAudit(
	t *testing.T,
) {
	bindingID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	actorID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	now := time.Date(2026, 7, 30, 16, 0, 0, 0, time.UTC)

	suspendSQL, suspendArgs, err := buildUpdateBindingLifecycleQuery(
		bindingID,
		BindingStatusSuspended,
		actorID,
		"maintenance",
		now,
	)
	require.NoError(t, err)
	assert.Contains(t, suspendSQL, "UPDATE device_geofence_bindings")
	assert.Contains(t, suspendArgs, BindingStatusSuspended)
	assert.NotContains(t, suspendSQL, "removed_by =")

	removeSQL, removeArgs, err := buildUpdateBindingLifecycleQuery(
		bindingID,
		BindingStatusRemoved,
		actorID,
		"retired",
		now,
	)
	require.NoError(t, err)
	assert.Contains(t, removeSQL, "removed_by")
	assert.Contains(t, removeSQL, "removed_at")
	assert.Contains(t, removeSQL, "remove_reason")
	assert.Contains(t, removeArgs, actorID)
	assert.Contains(t, removeArgs, "retired")
	assert.Contains(t, removeSQL, "RETURNING")
}

func TestBuildBindingFactMutationLockQueriesUseDefinitionDeviceBindingOrder(
	t *testing.T,
) {
	geofenceID := uuid.New()
	deviceIDs := []uuid.UUID{
		uuid.MustParse("ffffffff-ffff-ffff-ffff-ffffffffffff"),
		uuid.MustParse("00000000-0000-0000-0000-000000000001"),
	}
	bindingID := uuid.New()

	queries, err := buildBindingFactMutationLockQueries(
		geofenceID,
		deviceIDs,
		&bindingID,
	)

	require.NoError(t, err)
	require.Len(t, queries, 3)
	assert.Contains(t, queries[0].sql, "FROM geofence_definitions")
	assert.Contains(t, queries[0].sql, "FOR UPDATE")
	assert.Contains(t, queries[1].sql, "FROM devices")
	assert.Contains(t, queries[1].sql, "ORDER BY id")
	assert.Contains(t, queries[1].sql, "FOR UPDATE")
	assert.Contains(t, queries[2].sql, "FROM device_geofence_bindings")
	assert.Contains(t, queries[2].sql, "ORDER BY id")
	assert.Contains(t, queries[2].sql, "FOR UPDATE")
	assert.Contains(t, queries[2].args, bindingID.String())
}

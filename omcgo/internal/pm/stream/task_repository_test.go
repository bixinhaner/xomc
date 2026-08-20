package stream

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestSaveEffectiveFromDefaultsToNextNaturalHour(t *testing.T) {
	now := time.Date(2026, 7, 27, 6, 4, 30, 0, time.UTC)

	got := saveEffectiveFrom(SaveTaskRequest{}, now)

	require.True(t, got.Equal(time.Date(2026, 7, 27, 7, 0, 0, 0, time.UTC)))
}

func TestSaveEffectiveFromUsesExplicitTime(t *testing.T) {
	now := time.Date(2026, 7, 27, 6, 4, 30, 0, time.UTC)
	explicit := time.Date(2026, 7, 27, 6, 0, 0, 0, time.FixedZone("CST", 8*60*60))

	got := saveEffectiveFrom(SaveTaskRequest{EffectiveFrom: explicit}, now)

	require.True(t, got.Equal(time.Date(2026, 7, 26, 22, 0, 0, 0, time.UTC)))
	require.Equal(t, time.UTC, got.Location())
}

func TestShouldAdjustEffectiveFromBackdatesInitialUnchangedVersion(t *testing.T) {
	req := SaveTaskRequest{EffectiveFrom: time.Date(2026, 7, 27, 5, 0, 0, 0, time.UTC)}
	current := time.Date(2026, 7, 27, 5, 15, 0, 0, time.UTC)
	target := time.Date(2026, 7, 27, 5, 0, 0, 0, time.UTC)

	require.True(t, shouldAdjustEffectiveFrom(req, 1, current, target))
	require.False(t, shouldAdjustEffectiveFrom(SaveTaskRequest{}, 1, current, target))
	require.False(t, shouldAdjustEffectiveFrom(req, 2, current, target))
	require.False(t, shouldAdjustEffectiveFrom(req, 1, target, target))
	require.False(t, shouldAdjustEffectiveFrom(req, 1, target.Add(-slotDuration), target))
}

func TestBuildPurgeObsoleteBuiltinDeviceTasksSQLRetiresOnlyLegacyIDs(t *testing.T) {
	query, args, err := buildRetireTasksSQL(obsoleteBuiltinDeviceTaskIDs)

	require.NoError(t, err)
	require.Contains(t, query, "UPDATE pm_aggregation_tasks")
	require.Contains(t, query, "enabled = $1")
	require.Contains(t, query, "deleted_at")
	require.NotContains(t, query, "DELETE FROM")
	require.Equal(t, []interface{}{
		false,
		uuid.MustParse("0184dddd-0005-4000-8000-000000000001"),
		uuid.MustParse("0184dddd-0005-4000-8000-000000000002"),
		uuid.MustParse("0184dddd-0005-4000-8000-000000000003"),
	}, args)
}

func TestBuildUpsertTaskReactivationForcesNewCurrentVersion(t *testing.T) {
	taskID := uuid.MustParse("10000000-0000-4000-8000-000000000001")
	req := SaveTaskRequest{
		Name: "ISSUE354", Enabled: true, Visibility: "private", Creator: "test",
		SourceUpdatedAt: time.Date(2026, 8, 20, 9, 0, 0, 0, time.UTC),
	}

	query, _, err := buildUpsertTaskSQL(taskID, req, nil)

	require.NoError(t, err)
	require.Contains(t, query, "ON CONFLICT (id) DO UPDATE SET")
	require.Contains(t, query, "deleted_at = NULL")
	require.Contains(t, query, "current_version_id = NULL")
	require.Contains(t, query, "pm_aggregation_tasks.deleted_at IS NOT NULL")
}

func TestBuildUpdateTaskMetadataSQLSkipsIdenticalValues(t *testing.T) {
	taskID := uuid.MustParse("10000000-0000-4000-8000-000000000001")
	plannedEnd := time.Date(2026, 8, 2, 3, 4, 5, 0, time.UTC)
	req := SaveTaskRequest{
		Name: "LTE network", Enabled: true, Visibility: "public",
		PlannedEndAt: &plannedEnd,
	}

	query, args, err := buildUpdateTaskMetadataSQL(taskID, req, &plannedEnd)

	require.NoError(t, err)
	require.Contains(t, query, "name IS DISTINCT FROM $7")
	require.Contains(t, query, "enabled IS DISTINCT FROM $8")
	require.Contains(t, query, "visibility IS DISTINCT FROM $9")
	require.Contains(t, query, "planned_end_at IS DISTINCT FROM $10")
	require.Contains(t, query, "source_updated_at IS DISTINCT FROM $11")
	require.Equal(t, []interface{}{
		req.Name, req.Enabled, req.Visibility, &plannedEnd, nil, taskID.String(),
		req.Name, req.Enabled, req.Visibility, &plannedEnd, nil,
	}, args)
}

func TestBuildLoadMatchableRevisionSQLUsesSemanticFields(t *testing.T) {
	query, args, err := buildLoadMatchableRevisionSQL()

	require.NoError(t, err)
	require.NotContains(t, query, "updated_at")
	require.NotContains(t, query, "row_to_json")
	require.Contains(t, query, "current_version_id")
	require.Contains(t, query, "planned_end_at")
	require.Contains(t, query, "effective_from")
	require.Contains(t, query, "effective_to")
	require.Contains(t, query, "content_hash")
	require.Contains(t, query, "LEFT JOIN pm_aggregation_task_versions")
	require.Empty(t, args)
}

func TestPgProgressTaskLoaderExcludesVersionMembers(t *testing.T) {
	loader := NewPgProgressTaskLoader(nil)

	require.NotNil(t, loader)
	require.False(t, loader.includeMembers)
}

func TestBuildLoadTaskVersionsByIDSQLTargetsExactImmutableVersions(t *testing.T) {
	first := uuid.MustParse("10000000-0000-4000-8000-000000000001")
	second := uuid.MustParse("10000000-0000-4000-8000-000000000002")

	query, args, err := buildLoadTaskVersionsByIDSQL([]uuid.UUID{first, second})

	require.NoError(t, err)
	require.Contains(t, query, "FROM pm_aggregation_task_versions v")
	require.Contains(t, query, "JOIN pm_aggregation_tasks t")
	require.Contains(t, query, "v.id IN ($1,$2)")
	require.NotContains(t, query, "effective_to >=")
	require.Equal(t, []interface{}{first, second}, args)
}

func TestNormalizeTaskVersionIDsRemovesNilAndDuplicates(t *testing.T) {
	first := uuid.MustParse("10000000-0000-4000-8000-000000000001")
	second := uuid.MustParse("10000000-0000-4000-8000-000000000002")

	got := normalizeTaskVersionIDs([]uuid.UUID{second, uuid.Nil, first, second})

	require.Equal(t, []uuid.UUID{first, second}, got)
}

func TestBuildMissingSourceTaskCandidatesIsBoundedAndLocked(t *testing.T) {
	query, args, err := buildMissingSourceTaskCandidatesSQL(
		"adhoc_aggregation", "continuous", 200,
	)

	require.NoError(t, err)
	require.Contains(t, query, "NOT EXISTS")
	require.Contains(t, query, "source.id = pm_aggregation_tasks.id")
	require.Contains(t, query, "deleted_at IS NULL")
	require.Contains(t, query, "LIMIT 200")
	require.Contains(t, query, "FOR UPDATE SKIP LOCKED")
	require.Equal(t, []interface{}{"adhoc_aggregation", "continuous"}, args)
}

func TestCloseRetiredTaskVersionsNeverEndsBeforeEffectiveFrom(t *testing.T) {
	taskID := uuid.MustParse("10000000-0000-4000-8000-000000000001")
	effectiveTo := time.Date(2026, 8, 20, 9, 45, 0, 0, time.UTC)

	query, args, err := buildCloseRetiredTaskVersionsSQL([]uuid.UUID{taskID}, effectiveTo)

	require.NoError(t, err)
	require.Contains(t, query, "effective_to = GREATEST(effective_from, $1)")
	require.Equal(t, []interface{}{effectiveTo, taskID}, args)
}

func TestTaskMemberBatchesStayBelowPostgresParameterLimit(t *testing.T) {
	members := make([]TaskMember, 2501)

	batches := taskMemberBatches(members)

	require.Len(t, batches, 3)
	require.Len(t, batches[0], 1000)
	require.Len(t, batches[1], 1000)
	require.Len(t, batches[2], 501)
	require.LessOrEqual(t, len(batches[0])*taskMemberColumnCount, 65535)
	require.LessOrEqual(t, len(batches[1])*taskMemberColumnCount, 65535)
	require.LessOrEqual(t, len(batches[2])*taskMemberColumnCount, 65535)
}

func TestTaskMemberBatchesHandleEmptyInput(t *testing.T) {
	require.Empty(t, taskMemberBatches(nil))
}

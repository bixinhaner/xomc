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

func TestBuildPurgeObsoleteBuiltinDeviceTasksSQLTargetsOnlyLegacyIDs(t *testing.T) {
	query, args, err := buildPurgeObsoleteBuiltinDeviceTasksSQL()

	require.NoError(t, err)
	require.Equal(
		t,
		"DELETE FROM pm_aggregation_tasks WHERE id IN ($1,$2,$3)",
		query,
	)
	require.Equal(t, []interface{}{
		uuid.MustParse("0184dddd-0005-4000-8000-000000000001"),
		uuid.MustParse("0184dddd-0005-4000-8000-000000000002"),
		uuid.MustParse("0184dddd-0005-4000-8000-000000000003"),
	}, args)
}

func TestBuildLoadMatchableRevisionSQLIncludesDeletedTasks(t *testing.T) {
	query, args, err := buildLoadMatchableRevisionSQL()

	require.NoError(t, err)
	require.Equal(
		t,
		"SELECT COUNT(*), COALESCE(MAX(updated_at), to_timestamp(0)) FROM pm_aggregation_tasks",
		query,
	)
	require.Empty(t, args)
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

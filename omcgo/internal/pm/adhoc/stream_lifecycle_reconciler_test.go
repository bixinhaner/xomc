package adhoc

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestStreamingTaskShouldBeEnabledUsesStatusAndPlannedEnd(t *testing.T) {
	now := time.Date(2026, 8, 20, 9, 0, 0, 0, time.UTC)
	past, future := now.Add(-time.Minute), now.Add(time.Minute)

	require.True(t, streamingTaskShouldBeEnabled(&Task{Status: StatusScheduled}, now))
	require.True(t, streamingTaskShouldBeEnabled(
		&Task{Status: StatusRunning, PlannedEndAt: &future}, now,
	))
	require.False(t, streamingTaskShouldBeEnabled(&Task{Status: StatusCanceled}, now))
	require.False(t, streamingTaskShouldBeEnabled(
		&Task{Status: StatusScheduled, PlannedEndAt: &past}, now,
	))
	require.False(t, streamingTaskShouldBeEnabled(nil, now))
}

func TestBuildStreamingLifecycleCandidatesFindsDatabaseDrift(t *testing.T) {
	now := time.Date(2026, 8, 20, 9, 0, 0, 0, time.UTC)

	query, args, err := buildStreamingLifecycleCandidateIDsSQL(now, 200)

	require.NoError(t, err)
	query = strings.Join(strings.Fields(query), " ")
	require.Contains(t, query, "LEFT JOIN pm_aggregation_tasks streaming")
	require.Contains(t, query, "streaming.id IS NULL")
	require.Contains(t, query, "streaming.source_updated_at IS NULL")
	require.NotContains(t, query, "streaming.source_updated_at < t.updated_at")
	require.Contains(t, query, "streaming.enabled IS DISTINCT FROM")
	require.Contains(t, query, "streaming.deleted_at IS NOT NULL")
	require.Contains(t, query, "t.is_builtin =")
	require.Contains(t, query, "LIMIT 200")
	require.Equal(t, 5, len(args))
}

package stream

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestMarkPublicationPreparedIsIdempotentlyScopedToWindowRevision(t *testing.T) {
	key := WindowKey{
		TaskVersionID: uuid.New(), EntityKey: "SN-1", Granularity: GranularityHourly,
		Start: time.Date(2026, 8, 1, 14, 0, 0, 0, time.UTC),
		End:   time.Date(2026, 8, 1, 15, 0, 0, 0, time.UTC),
	}

	query, args, err := markPublicationPreparedQuery(key, 1).ToSql()

	require.NoError(t, err)
	require.Contains(t, query, "INSERT INTO pm_aggregation_publications")
	require.Contains(t, query, "ON CONFLICT (task_version_id, granularity, window_start)")
	require.Contains(t, query, "prepared_entities")
	require.Contains(t, args, key.TaskVersionID)
	require.Contains(t, args, key.Start)
}

func TestPublishReadyQueryWaitsForWatermarkAndAllPreparedEntities(t *testing.T) {
	now := time.Date(2026, 8, 1, 15, 12, 30, 0, time.UTC)

	query, args, err := publishReadyCandidatesQuery(now, 12*time.Minute, 32).ToSql()

	require.NoError(t, err)
	require.Contains(t, query, "status =")
	require.Contains(t, query, "window_end <=")
	require.Contains(t, query, "NOT EXISTS")
	require.Contains(t, query, "pm_aggregation_windows")
	require.Contains(t, query, "status NOT IN")
	require.Contains(t, query, "FOR UPDATE SKIP LOCKED")
	require.True(t, strings.Contains(query, "LIMIT 32") || strings.Contains(query, "LIMIT $"))
	require.Contains(t, args, now.Add(-12*time.Minute))
}

func TestEveryGranularityAndRevisionIsPreparedBeforePublication(t *testing.T) {
	require.Equal(t, "prepared", finalWindowStatus(GranularityHourly, 1))
	require.Equal(t, "prepared", finalWindowStatus(GranularityHourly, 2))
	require.Equal(t, "prepared", finalWindowStatus(GranularityDaily, 1))
	require.Equal(t, "prepared", finalWindowStatus(GranularityWeekly, 3))
}

func TestInitialRevisionDoesNotSerializeIndependentEntitiesOnPublicationLock(t *testing.T) {
	require.False(t, requiresPublicationLock(1),
		"the first revision has no visible generation to protect and must finalize entities concurrently")
	require.True(t, requiresPublicationLock(2),
		"late rebuild revisions must remain serialized with the visible publication generation")
}

func TestEnsureInitialPublicationIsIdempotent(t *testing.T) {
	key := WindowKey{
		TaskID: uuid.New(), TaskVersionID: uuid.New(), EntityKey: "SN-1",
		Granularity: GranularityHourly,
		Start:       time.Date(2026, 8, 1, 14, 0, 0, 0, time.UTC),
		End:         time.Date(2026, 8, 1, 15, 0, 0, 0, time.UTC),
	}

	query, args, err := ensureInitialPublicationQuery(key).ToSql()

	require.NoError(t, err)
	require.Contains(t, query, "INSERT INTO pm_aggregation_publications")
	require.Contains(t, query, "ON CONFLICT (task_version_id, granularity, window_start) DO NOTHING")
	require.Contains(t, args, key.TaskVersionID)
	require.Contains(t, args, 1)
}

func TestPublishReadyAcceptsPublishedEntitiesFromPreviousRevision(t *testing.T) {
	query, _, err := publishReadyCandidatesQuery(time.Now().UTC(), 12*time.Minute, 32).ToSql()
	require.NoError(t, err)
	require.Contains(t, query, "status NOT IN")
	require.Contains(t, query, "prepared")
	require.Contains(t, query, "published")
}

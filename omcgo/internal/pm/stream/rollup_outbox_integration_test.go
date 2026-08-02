//go:build integration

package stream

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

func TestIntegrationVisitSnapshotsForPeriodKeepsPublishedRevisionAcrossPages(t *testing.T) {
	dsn := os.Getenv("OMCGO_TSDB_DSN")
	if dsn == "" {
		t.Skip("OMCGO_TSDB_DSN not set; skipping TimescaleDB integration test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)
	defer pool.Close()

	taskID := uuid.New()
	sourceVersionID := uuid.New()
	publicationVersionID := uuid.New()
	entityKey := "integration-" + uuid.NewString()
	start := time.Date(2026, 8, 1, 14, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour)

	_, err = pool.Exec(ctx, `
INSERT INTO pm_aggregation_windows (
  task_id, task_version_id, entity_key, granularity, window_start, window_end,
  status, expected_slots, received_slots, published_revision, revision, published_at
) VALUES ($1, $2, $3, 'hourly', $4, $5, 'published', 4, 4, 2, 2, now())`,
		taskID, publicationVersionID, entityKey, start, end)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `
DELETE FROM pm_aggregation_counter_rollups
WHERE task_version_id=$1 AND publication_task_version_id=$2 AND entity_key=$3`,
			sourceVersionID, publicationVersionID, entityKey)
		_, _ = pool.Exec(context.Background(), `
DELETE FROM pm_aggregation_windows
WHERE task_version_id=$1 AND entity_key=$2 AND granularity='hourly' AND window_start=$3`,
			publicationVersionID, entityKey, start)
	})

	const rowCount = int(rollupSnapshotPageSize) + 1
	for chunk := 0; chunk < rowCount; chunk++ {
		eventID := uuid.New()
		payload, marshalErr := json.Marshal(RollupPayload{
			SchemaVersion: SchemaVersion, EventID: eventID, TaskID: taskID,
			TaskVersionID: sourceVersionID, SourceGranularity: GranularityHourly,
			EntityKey: entityKey, WindowStart: start, WindowEnd: end,
			SourceExpectedSlots: 4, SourceReceivedSlots: 4, Complete: true,
			ChunkIndex: chunk, ChunkCount: rowCount,
			Values: []ContributionValue{{
				Dimension: DimensionNetwork, DimensionKey: "network",
				MetricPath: "C1", MetricType: "counter", Operation: AggregationSum,
				Sum: float64(chunk + 1), Count: 1, Composed: true,
			}},
		})
		require.NoError(t, marshalErr)
		_, err = pool.Exec(ctx, `
INSERT INTO pm_aggregation_counter_rollups (
  event_id, task_id, task_version_id, publication_task_version_id, entity_key,
  granularity, window_start, window_end, chunk_index, chunk_count, complete,
  revision, payload
) VALUES ($1,$2,$3,$4,$5,'hourly',$6,$7,$8,$9,true,2,$10)`,
			eventID, taskID, sourceVersionID, publicationVersionID, entityKey,
			start, end, chunk, rowCount, payload)
		require.NoError(t, err)
	}

	repo := NewRollupOutboxRepository(pool)
	visited := 0
	err = repo.VisitSnapshotsForPeriod(
		ctx, []uuid.UUID{sourceVersionID}, GranularityHourly, start, end,
		func(RollupPayload) error {
			visited++
			if visited == 1 {
				_, updateErr := pool.Exec(ctx, `
UPDATE pm_aggregation_windows SET published_revision=1
WHERE task_version_id=$1 AND entity_key=$2 AND granularity='hourly' AND window_start=$3`,
					publicationVersionID, entityKey, start)
				return updateErr
			}
			return nil
		},
	)
	require.NoError(t, err)
	require.Equal(t, rowCount, visited,
		"all pages must retain the revision visible when the repeatable-read transaction began")

	_, err = pool.Exec(ctx, `
UPDATE pm_aggregation_windows SET published_revision=2
WHERE task_version_id=$1 AND entity_key=$2 AND granularity='hourly' AND window_start=$3`,
		publicationVersionID, entityKey, start)
	require.NoError(t, err)
	wantVisitErr := errors.New("stop rebuild visit")
	err = repo.VisitSnapshotsForPeriod(
		ctx, []uuid.UUID{sourceVersionID}, GranularityHourly, start, end,
		func(RollupPayload) error { return wantVisitErr },
	)
	require.ErrorIs(t, err, wantVisitErr)
	require.NoError(t, pool.Ping(ctx), "callback failure must roll back and release the connection")

	canceledCtx, cancelVisit := context.WithCancel(ctx)
	cancelVisit()
	err = repo.VisitSnapshotsForPeriod(
		canceledCtx, []uuid.UUID{sourceVersionID}, GranularityHourly, start, end,
		func(RollupPayload) error { return nil },
	)
	require.ErrorIs(t, err, context.Canceled)
	require.NoError(t, pool.Ping(ctx), "context cancellation must release the connection")
}

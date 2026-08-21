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

	"github.com/omcgo/omcgo/internal/core/event"
)

func TestIntegrationOutboxClaimCommitsBeforeNetworkPublish(t *testing.T) {
	dsn := os.Getenv("OMCGO_TSDB_DSN")
	if dsn == "" {
		t.Skip("OMCGO_TSDB_DSN not set; skipping TimescaleDB integration test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)
	defer pool.Close()

	repo := NewOutboxRepository(pool)
	deleteOutboxRelayFixtures(ctx, pool)
	claimAt := time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC)
	row := insertOutboxRelayFixture(
		t, ctx, pool, time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC), claimAt,
	)
	t.Cleanup(func() { deleteOutboxRelayFixture(context.Background(), pool, row.EventID) })
	requirePendingOutboxRelayFixture(t, ctx, pool, row.EventID, claimAt)

	claimed, err := repo.ClaimBatch(ctx, 1, claimAt, 30*time.Second)
	require.NoError(t, err)
	require.Len(t, claimed, 1)
	require.Equal(t, row.EventID, claimed[0].EventID)
	require.NotEqual(t, uuid.Nil, claimed[0].ClaimToken)

	probeCtx, probeCancel := context.WithTimeout(ctx, 500*time.Millisecond)
	defer probeCancel()
	started := time.Now()
	tag, err := pool.Exec(probeCtx, `
UPDATE pm_aggregation_outbox SET last_error = 'lock-probe'
WHERE event_id = $1`, row.EventID)
	require.NoError(t, err)
	require.Equal(t, int64(1), tag.RowsAffected())
	lockProbeElapsed := time.Since(started)
	t.Logf("outbox claim lock probe: update_same_row_elapsed=%s claim_token=%s", lockProbeElapsed, claimed[0].ClaimToken)
	require.Less(t, lockProbeElapsed, 500*time.Millisecond,
		"claim transaction must be committed before NATS Publish waits outside the database")

	ok, err := repo.MarkPublished(ctx, row.EventID, uuid.New(), time.Now().UTC())
	require.NoError(t, err)
	require.False(t, ok, "wrong claim token must not publish another worker's claim")

	ok, err = repo.MarkPublished(ctx, row.EventID, claimed[0].ClaimToken, time.Now().UTC())
	require.NoError(t, err)
	require.True(t, ok)

	var publishedAt *time.Time
	err = pool.QueryRow(ctx, `
SELECT published_at FROM pm_aggregation_outbox WHERE event_id = $1`, row.EventID).Scan(&publishedAt)
	require.NoError(t, err)
	require.NotNil(t, publishedAt)
}

func TestIntegrationOutboxPublishFailureFinalizesByClaimToken(t *testing.T) {
	dsn := os.Getenv("OMCGO_TSDB_DSN")
	if dsn == "" {
		t.Skip("OMCGO_TSDB_DSN not set; skipping TimescaleDB integration test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)
	defer pool.Close()

	repo := NewOutboxRepository(pool)
	deleteOutboxRelayFixtures(ctx, pool)
	claimAt := time.Date(2099, 1, 1, 0, 1, 0, 0, time.UTC)
	row := insertOutboxRelayFixture(
		t, ctx, pool, time.Date(1900, 1, 1, 0, 0, 1, 0, time.UTC), claimAt,
	)
	t.Cleanup(func() { deleteOutboxRelayFixture(context.Background(), pool, row.EventID) })
	requirePendingOutboxRelayFixture(t, ctx, pool, row.EventID, claimAt)

	claimed, err := repo.ClaimBatch(ctx, 1, claimAt, 30*time.Second)
	require.NoError(t, err)
	require.Len(t, claimed, 1)
	require.Equal(t, row.EventID, claimed[0].EventID)

	publishErr := errors.New("nats publish unavailable")
	ok, err := repo.MarkFailed(ctx, row.EventID, uuid.New(), publishErr, time.Now().UTC(), time.Second)
	require.NoError(t, err)
	require.False(t, ok, "wrong claim token must not mark another worker's claim failed")

	now := time.Now().UTC()
	retryAfter := time.Hour
	ok, err = repo.MarkFailed(ctx, row.EventID, claimed[0].ClaimToken, publishErr, now, retryAfter)
	require.NoError(t, err)
	require.True(t, ok)

	var claimToken *uuid.UUID
	var claimExpiresAt *time.Time
	var lastError string
	var nextAttemptAt time.Time
	var attempts int
	err = pool.QueryRow(ctx, `
SELECT claim_token, claim_expires_at, last_error, next_attempt_at, publish_attempts
FROM pm_aggregation_outbox WHERE event_id = $1`, row.EventID).
		Scan(&claimToken, &claimExpiresAt, &lastError, &nextAttemptAt, &attempts)
	require.NoError(t, err)
	require.Nil(t, claimToken)
	require.Nil(t, claimExpiresAt)
	require.Equal(t, publishErr.Error(), lastError)
	require.GreaterOrEqual(t, nextAttemptAt.Sub(now), retryAfter-100*time.Millisecond)
	require.Equal(t, 1, attempts)
	t.Logf("outbox publish failure retry: attempts=%d retry_after=%s next_attempt_delta=%s",
		attempts, retryAfter, nextAttemptAt.Sub(now))
}

type outboxRelayFixtureRow struct {
	EventID uuid.UUID
}

func insertOutboxRelayFixture(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	createdAt time.Time,
	nextAttemptAt time.Time,
) outboxRelayFixtureRow {
	t.Helper()
	eventID := uuid.New()
	sourceFileID := uuid.New()
	ingestBatchID := uuid.New()
	deviceID := uuid.New()
	windowStart := time.Date(2026, 8, 21, 9, 0, 0, 0, time.UTC)
	payload := event.PMAggregationNormalizedPayload{
		SchemaVersion: SchemaVersion,
		EventID:       eventID,
		SourceFileID:  sourceFileID,
		IngestBatchID: ingestBatchID,
		DeviceID:      deviceID,
		DeviceOUI:     "OUTBOXIT",
		DeviceSN:      "OUTBOX-" + deviceID.String(),
		Technology:    "lte",
		WindowStart:   windowStart,
		WindowEnd:     windowStart.Add(15 * time.Minute),
		Measurements: []event.PMAggregationMeasurement{{
			ObjectLDN:    "",
			CounterGroup: "CG",
			Metrics: []event.PMAggregationMetric{{
				MetricPath: "C_OUTBOX", MetricType: "counter",
				StatisType: "sum", Value: 1,
			}},
		}},
	}
	raw, err := json.Marshal(payload)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `
INSERT INTO pm_aggregation_outbox (
  event_id, source_file_id, ingest_batch_id, payload, created_at,
  event_window_start, event_window_end, device_id, barrier_eligible,
  next_attempt_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,true,$9)`,
		eventID, sourceFileID, ingestBatchID, raw, createdAt,
		windowStart, windowStart.Add(15*time.Minute), deviceID, nextAttemptAt)
	require.NoError(t, err)
	return outboxRelayFixtureRow{EventID: eventID}
}

func deleteOutboxRelayFixture(ctx context.Context, pool *pgxpool.Pool, eventID uuid.UUID) {
	_, _ = pool.Exec(ctx, `DELETE FROM pm_aggregation_outbox WHERE event_id = $1`, eventID)
}

func deleteOutboxRelayFixtures(ctx context.Context, pool *pgxpool.Pool) {
	_, _ = pool.Exec(ctx, `
DELETE FROM pm_aggregation_outbox
WHERE payload->>'device_oui' = 'OUTBOXIT'`)
}

func requirePendingOutboxRelayFixture(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	eventID uuid.UUID,
	claimAt time.Time,
) {
	t.Helper()
	var count int
	err := pool.QueryRow(ctx, `
SELECT count(*) FROM pm_aggregation_outbox
WHERE event_id = $1
  AND published_at IS NULL
  AND consumed_at IS NULL
  AND next_attempt_at <= $2
  AND (claim_token IS NULL OR claim_expires_at IS NULL OR claim_expires_at <= $2)`,
		eventID, claimAt,
	).Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 1, count, "fixture row must match the outbox claim predicate")
}

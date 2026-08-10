package outbox

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omcgo/omcgo/internal/core/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPgRelayRepositoryClaimAndLeaseGuardIntegration(t *testing.T) {
	dsn := os.Getenv("TEST_PG_URL")
	if dsn == "" {
		t.Skip("TEST_PG_URL not set")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	now := time.Now().UTC().Truncate(time.Microsecond)
	pendingID := uuid.New()
	failedID := uuid.New()
	expiredID := uuid.New()
	ids := []uuid.UUID{pendingID, failedID, expiredID}
	t.Cleanup(func() {
		query, args, buildErr := storage.Psql.
			Delete("event_outbox").
			Where(sq.Eq{"id": ids}).
			ToSql()
		if buildErr == nil {
			_, _ = pool.Exec(context.Background(), query, args...)
		}
	})

	insertSQL, insertArgs, err := storage.Psql.
		Insert("event_outbox").
		Columns(
			"id",
			"aggregate_type",
			"aggregate_id",
			"subject",
			"payload",
			"dedupe_key",
			"status",
			"attempts",
			"next_attempt_at",
			"claim_token",
			"claim_expires_at",
			"created_at",
			"updated_at",
		).
		Values(
			pendingID,
			"device",
			pendingID.String(),
			"device.location.observed",
			json.RawMessage(`{"version":1}`),
			"integration:"+pendingID.String(),
			StatusPending,
			0,
			now.Add(-time.Minute),
			nil,
			nil,
			now.Add(-3*time.Minute),
			now.Add(-3*time.Minute),
		).
		Values(
			failedID,
			"device",
			failedID.String(),
			"device.location.observed",
			json.RawMessage(`{"version":2}`),
			"integration:"+failedID.String(),
			StatusFailed,
			1,
			now.Add(-time.Minute),
			nil,
			nil,
			now.Add(-2*time.Minute),
			now.Add(-2*time.Minute),
		).
		Values(
			expiredID,
			"device",
			expiredID.String(),
			"device.location.observed",
			json.RawMessage(`{"version":3}`),
			"integration:"+expiredID.String(),
			StatusPublishing,
			1,
			now.Add(-time.Minute),
			uuid.New(),
			now.Add(-time.Second),
			now.Add(-time.Minute),
			now.Add(-time.Minute),
		).
		ToSql()
	require.NoError(t, err)
	_, err = pool.Exec(ctx, insertSQL, insertArgs...)
	require.NoError(t, err)

	repo := NewPgRelayRepository(storage.NewPoolDB(pool))
	claimed, err := repo.ClaimDue(ctx, ClaimOptions{
		Now:         now,
		Lease:       30 * time.Second,
		Limit:       10,
		MaxAttempts: 3,
	})
	require.NoError(t, err)
	require.Len(t, claimed, 3)

	claimedByID := make(map[uuid.UUID]Entry, len(claimed))
	for _, entry := range claimed {
		claimedByID[entry.ID] = entry
		assert.Equal(t, StatusPublishing, entry.Status)
		assert.NotEqual(t, uuid.Nil, entry.ClaimToken)
		assert.Equal(t, now.Add(30*time.Second), entry.ClaimExpiresAt)
	}
	assert.Equal(t, 1, claimedByID[pendingID].Attempts)
	assert.Equal(t, 2, claimedByID[failedID].Attempts)
	assert.Equal(t, 2, claimedByID[expiredID].Attempts)

	published, err := repo.MarkPublished(
		ctx,
		pendingID,
		claimedByID[pendingID].ClaimToken,
		now.Add(time.Second),
	)
	require.NoError(t, err)
	assert.True(t, published)

	staleFinalization, err := repo.MarkFailed(
		ctx,
		pendingID,
		uuid.New(),
		"stale worker",
		now.Add(time.Minute),
	)
	require.NoError(t, err)
	assert.False(t, staleFinalization)

	failed, err := repo.MarkFailed(
		ctx,
		failedID,
		claimedByID[failedID].ClaimToken,
		"temporary",
		now.Add(time.Minute),
	)
	require.NoError(t, err)
	assert.True(t, failed)

	dead, err := repo.MarkDead(
		ctx,
		expiredID,
		claimedByID[expiredID].ClaimToken,
		"exhausted",
		now.Add(time.Second),
	)
	require.NoError(t, err)
	assert.True(t, dead)
}

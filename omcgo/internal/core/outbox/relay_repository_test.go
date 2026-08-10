package outbox

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildClaimSelectUsesDueRowsExpiredLeasesAndSkipLocked(t *testing.T) {
	now := time.Date(2026, 7, 30, 12, 0, 0, 0, time.UTC)
	query, args, err := buildClaimSelect(ClaimOptions{
		Now:         now,
		Lease:       30 * time.Second,
		Limit:       100,
		MaxAttempts: 10,
	})

	require.NoError(t, err)
	assert.Contains(t, query, "FROM event_outbox")
	assert.Contains(t, query, "status IN")
	assert.Contains(t, query, "next_attempt_at <=")
	assert.Contains(t, query, "claim_expires_at <=")
	assert.Contains(t, query, "attempts <")
	assert.Contains(t, query, "FOR UPDATE SKIP LOCKED")
	assert.Contains(t, query, "CASE WHEN status = 'publishing' THEN 0 ELSE 1 END")
	assert.Contains(t, args, StatusPending)
	assert.Contains(t, args, StatusFailed)
	assert.Contains(t, args, StatusPublishing)
	assert.Contains(t, args, now)
	assert.Contains(t, args, 10)
}

func TestBuildClaimUpdateAssignsLeaseAndIncrementsAttempts(t *testing.T) {
	ids := []uuid.UUID{uuid.New(), uuid.New()}
	token := uuid.New()
	expiresAt := time.Date(2026, 7, 30, 12, 0, 30, 0, time.UTC)

	query, args, err := buildClaimUpdate(ids, token, expiresAt)

	require.NoError(t, err)
	assert.Contains(t, query, "UPDATE event_outbox")
	assert.Contains(t, query, "status =")
	assert.Contains(t, query, "attempts + 1")
	assert.Contains(t, query, "claim_token =")
	assert.Contains(t, query, "claim_expires_at =")
	assert.Contains(t, args, StatusPublishing)
	assert.Contains(t, args, token)
	assert.Contains(t, args, expiresAt)
	for _, id := range ids {
		assert.Contains(t, args, id)
	}
}

func TestBuildExpireExhaustedClaimsMovesRowsToDead(t *testing.T) {
	now := time.Date(2026, 7, 30, 12, 0, 0, 0, time.UTC)

	query, args, err := buildExpireExhaustedClaims(now, 10)

	require.NoError(t, err)
	assert.Contains(t, query, "UPDATE event_outbox")
	assert.Contains(t, query, "status =")
	assert.Contains(t, query, "claim_expires_at <=")
	assert.Contains(t, query, "attempts >=")
	assert.Contains(t, query, "claim_token =")
	assert.Contains(t, args, StatusDead)
	assert.Contains(t, args, StatusPublishing)
	assert.Contains(t, args, now)
	assert.Contains(t, args, 10)
}

func TestBuildFinalizersRequireCurrentPublishingLease(t *testing.T) {
	id := uuid.New()
	token := uuid.New()
	now := time.Date(2026, 7, 30, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name   string
		build  func() (string, []any, error)
		status Status
	}{
		{
			name: "published",
			build: func() (string, []any, error) {
				return buildMarkPublished(id, token, now)
			},
			status: StatusPublished,
		},
		{
			name: "failed",
			build: func() (string, []any, error) {
				return buildMarkFailed(id, token, "temporary", now.Add(time.Minute), now)
			},
			status: StatusFailed,
		},
		{
			name: "dead",
			build: func() (string, []any, error) {
				return buildMarkDead(id, token, "exhausted", now)
			},
			status: StatusDead,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query, args, err := tt.build()
			require.NoError(t, err)
			assert.Contains(t, query, "UPDATE event_outbox")
			assert.Contains(t, query, "status =")
			assert.Contains(t, query, "claim_token =")
			assert.Contains(t, query, "claim_expires_at =")
			assert.Contains(t, args, id.String())
			assert.Contains(t, args, token.String())
			assert.Contains(t, args, StatusPublishing)
			assert.Contains(t, args, tt.status)
		})
	}
}

type beginRecordingDB struct {
	beginCalls int
}

func (d *beginRecordingDB) Begin(context.Context) (pgx.Tx, error) {
	d.beginCalls++
	return nil, assert.AnError
}

func (d *beginRecordingDB) Query(context.Context, string, ...any) (pgx.Rows, error) {
	panic("unexpected Query")
}

func (d *beginRecordingDB) QueryRow(context.Context, string, ...any) pgx.Row {
	panic("unexpected QueryRow")
}

func (d *beginRecordingDB) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	panic("unexpected Exec")
}

func TestClaimDueRejectsInvalidOptionsBeforeBeginningTransaction(t *testing.T) {
	tests := []ClaimOptions{
		{Now: time.Now(), Lease: 0, Limit: 1, MaxAttempts: 1},
		{Now: time.Now(), Lease: time.Second, Limit: 0, MaxAttempts: 1},
		{Now: time.Now(), Lease: time.Second, Limit: 1, MaxAttempts: 0},
	}

	for _, options := range tests {
		db := &beginRecordingDB{}
		repo := NewPgRelayRepository(db)

		_, err := repo.ClaimDue(context.Background(), options)

		require.Error(t, err)
		assert.Zero(t, db.beginCalls)
	}
}

func TestTruncateOutboxErrorPreservesValidUTF8AndBoundsLength(t *testing.T) {
	message := strings.Repeat("故障", maxStoredErrorRunes)

	truncated := truncateOutboxError(message)

	assert.Len(t, []rune(truncated), maxStoredErrorRunes)
	assert.True(t, strings.HasPrefix(message, truncated))
}

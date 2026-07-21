package admin

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

func configApplyTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("OMCGO_TEST_DB_DSN")
	if dsn == "" {
		t.Skip("OMCGO_TEST_DB_DSN not set")
	}
	pool, err := pgxpool.New(t.Context(), dsn)
	require.NoError(t, err)
	require.NoError(t, pool.Ping(t.Context()))
	t.Cleanup(pool.Close)
	var migrated bool
	require.NoError(t, pool.QueryRow(t.Context(), `SELECT to_regclass('config_apply_targets') IS NOT NULL`).Scan(&migrated))
	if !migrated {
		t.Skip("config apply migration is not applied")
	}
	return pool
}

func insertApplyTargetForTest(t *testing.T, pool *pgxpool.Pool, category, target, status string, leaseExpiresAt *time.Time) uuid.UUID {
	t.Helper()
	batchID := uuid.New()
	_, err := pool.Exec(t.Context(), `
INSERT INTO config_apply_batches (id, category, config_version, status, created_at, updated_at)
VALUES ($1, $2, $3, 'pending', NOW() - INTERVAL '10 minutes', NOW() - INTERVAL '10 minutes')
`, batchID, category, time.Now().UnixNano())
	require.NoError(t, err)
	_, err = pool.Exec(t.Context(), `
INSERT INTO config_apply_targets (batch_id, category, target, status, lease_expires_at, updated_at)
VALUES ($1, $2, $3, $4, $5, NOW() - INTERVAL '10 minutes')
`, batchID, category, target, status, leaseExpiresAt)
	require.NoError(t, err)
	return batchID
}

func cleanupApplyCategory(t *testing.T, pool *pgxpool.Pool, category string) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `DELETE FROM config_apply_batches WHERE category = $1`, category)
	require.NoError(t, err)
}

func TestClaimNextApplyTargetReclaimsExpiredLeaseBeforeOlderPendingSibling(t *testing.T) {
	pool := configApplyTestPool(t)
	category := "test-lease-" + uuid.NewString()
	t.Cleanup(func() { cleanupApplyCategory(t, pool, category) })
	insertApplyTargetForTest(t, pool, category, "runtime", ConfigApplyStatusPending, nil)
	expired := time.Now().Add(-time.Minute)
	expiredBatch := insertApplyTargetForTest(t, pool, category, "runtime", ConfigApplyStatusApplying, &expired)

	work, err := NewPgSysConfigRepository(pool).ClaimNextApplyTarget(t.Context())
	require.NoError(t, err)
	require.NotNil(t, work)
	require.Equal(t, expiredBatch, work.Batch.ID)
}

func TestClaimNextApplyTargetSkipsBlockedSiblingAndClaimsOtherTarget(t *testing.T) {
	pool := configApplyTestPool(t)
	blockedCategory := "test-blocked-" + uuid.NewString()
	otherCategory := "test-other-" + uuid.NewString()
	t.Cleanup(func() {
		cleanupApplyCategory(t, pool, blockedCategory)
		cleanupApplyCategory(t, pool, otherCategory)
	})
	insertApplyTargetForTest(t, pool, blockedCategory, "runtime", ConfigApplyStatusPending, nil)
	future := time.Now().Add(time.Minute)
	insertApplyTargetForTest(t, pool, blockedCategory, "runtime", ConfigApplyStatusApplying, &future)
	otherBatch := insertApplyTargetForTest(t, pool, otherCategory, "other", ConfigApplyStatusPending, nil)

	work, err := NewPgSysConfigRepository(pool).ClaimNextApplyTarget(t.Context())
	require.NoError(t, err)
	require.NotNil(t, work)
	require.Equal(t, otherBatch, work.Batch.ID)
}

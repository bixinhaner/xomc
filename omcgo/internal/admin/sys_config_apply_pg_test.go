package admin

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/acs/transfercfg"
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

func TestPgSysConfigRepositorySupportsAtomicCategoryValidation(t *testing.T) {
	var repo any = (*PgSysConfigRepository)(nil)
	_, ok := repo.(SysConfigAtomicCategoryValidationRepository)
	require.True(t, ok)
}

func TestPgSysConfigRepositorySerializesCategoryValidationAndWrites(t *testing.T) {
	pool := configApplyTestPool(t)
	category := "test-atomic-category-" + uuid.NewString()
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM config_apply_batches WHERE category = $1`, category)
		_, _ = pool.Exec(context.Background(), `DELETE FROM sys_configs WHERE category = $1`, category)
		_, _ = pool.Exec(context.Background(), `DELETE FROM config_apply_versions WHERE category = $1`, category)
	})
	repo := NewPgSysConfigRepository(pool)

	firstValidating := make(chan struct{})
	releaseFirst := make(chan struct{})
	firstDone := make(chan error, 1)
	go func() {
		_, err := repo.BatchUpsertWithApplyValidated(
			context.Background(),
			category,
			[]BatchItem{
				{Key: transfercfg.KeyProtocolPolicy, Value: transfercfg.ProtocolPolicyForceHTTP},
				{Key: transfercfg.KeyHTTPSUploadBaseURL, Value: "https://acs.example.com:8443/upload"},
			},
			nil,
			func(values map[string]string) error {
				close(firstValidating)
				<-releaseFirst
				return transfercfg.ValidateConfig(values)
			},
		)
		firstDone <- err
	}()
	select {
	case <-firstValidating:
	case <-time.After(5 * time.Second):
		t.Fatal("first category validator did not start")
	}

	secondValidating := make(chan struct{})
	secondDone := make(chan error, 1)
	go func() {
		_, err := repo.BatchUpsertWithApplyValidated(
			context.Background(),
			category,
			[]BatchItem{
				{Key: transfercfg.KeyProtocolPolicy, Value: transfercfg.ProtocolPolicyPreferHTTPS},
				{Key: transfercfg.KeyHTTPSDownloadBaseURL, Value: "https://acs.example.com:8443/download"},
			},
			nil,
			func(values map[string]string) error {
				close(secondValidating)
				return transfercfg.ValidateConfig(values)
			},
		)
		secondDone <- err
	}()

	select {
	case <-secondValidating:
		t.Fatal("second validator entered before the first category transaction released its lock")
	case <-time.After(150 * time.Millisecond):
	}
	close(releaseFirst)
	require.NoError(t, <-firstDone)
	require.NoError(t, <-secondDone)
	select {
	case <-secondValidating:
	default:
		t.Fatal("second validator was not called after the first transaction committed")
	}

	configs, err := repo.List(t.Context(), category, false)
	require.NoError(t, err)
	values := make(map[string]string, len(configs))
	for _, config := range configs {
		values[config.Key] = config.Value
	}
	require.Equal(t, transfercfg.ProtocolPolicyPreferHTTPS, values[transfercfg.KeyProtocolPolicy])
	require.Equal(t, "https://acs.example.com:8443/upload", values[transfercfg.KeyHTTPSUploadBaseURL])
	require.Equal(t, "https://acs.example.com:8443/download", values[transfercfg.KeyHTTPSDownloadBaseURL])
}

func TestPgSysConfigRepositoryRollsBackRejectedCategoryValidation(t *testing.T) {
	pool := configApplyTestPool(t)
	category := "test-atomic-rollback-" + uuid.NewString()
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM config_apply_batches WHERE category = $1`, category)
		_, _ = pool.Exec(context.Background(), `DELETE FROM sys_configs WHERE category = $1`, category)
		_, _ = pool.Exec(context.Background(), `DELETE FROM config_apply_versions WHERE category = $1`, category)
	})
	repo := NewPgSysConfigRepository(pool)

	_, err := repo.BatchUpsertWithApplyValidated(
		t.Context(),
		category,
		[]BatchItem{{Key: "key", Value: "value"}},
		nil,
		func(map[string]string) error { return errors.New("rejected final state") },
	)
	require.ErrorContains(t, err, "rejected final state")

	var configCount, versionCount, batchCount int
	require.NoError(t, pool.QueryRow(t.Context(), `
SELECT
    (SELECT COUNT(*) FROM sys_configs WHERE category = $1),
    (SELECT COUNT(*) FROM config_apply_versions WHERE category = $1),
    (SELECT COUNT(*) FROM config_apply_batches WHERE category = $1)
`, category).Scan(&configCount, &versionCount, &batchCount))
	require.Zero(t, configCount)
	require.Zero(t, versionCount)
	require.Zero(t, batchCount)
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

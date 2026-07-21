package main

import (
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
)

func TestSysConfigApplyMigrationUpgradesIntermediateSchema(t *testing.T) {
	dsn := os.Getenv("OMCGO_TEST_DB_DSN")
	if dsn == "" {
		t.Skip("OMCGO_TEST_DB_DSN not set")
	}

	adminDB, err := sql.Open("pgx", dsn)
	require.NoError(t, err)
	defer adminDB.Close()

	schema := "migration_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	_, err = adminDB.Exec(fmt.Sprintf(`CREATE SCHEMA %q`, schema))
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = adminDB.Exec(fmt.Sprintf(`DROP SCHEMA IF EXISTS %q CASCADE`, schema))
	})

	parsedDSN, err := url.Parse(dsn)
	require.NoError(t, err)
	query := parsedDSN.Query()
	query.Set("search_path", schema)
	parsedDSN.RawQuery = query.Encode()
	db, err := sql.Open("pgx", parsedDSN.String())
	require.NoError(t, err)
	defer db.Close()

	intermediateStatements := []string{`
CREATE TABLE config_apply_versions (
    category varchar(64) PRIMARY KEY,
    config_version bigint NOT NULL DEFAULT 0,
    updated_at timestamptz NOT NULL DEFAULT now()
)`, `
CREATE TABLE config_apply_batches (
    id uuid PRIMARY KEY,
    category varchar(64) NOT NULL,
    config_version bigint NOT NULL,
    status varchar(16) NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (category, config_version)
)`, `
CREATE TABLE config_apply_targets (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    batch_id uuid NOT NULL REFERENCES config_apply_batches(id) ON DELETE CASCADE,
    target varchar(128) NOT NULL,
    status varchar(16) NOT NULL,
    attempts integer NOT NULL DEFAULT 0,
    applied_at timestamptz,
    last_error text NOT NULL DEFAULT '',
    expected_value jsonb NOT NULL DEFAULT '{}'::jsonb,
    actual_value jsonb NOT NULL DEFAULT '{}'::jsonb,
    lease_token uuid,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (batch_id, target)
)`, `
CREATE INDEX idx_config_apply_targets_recovering
    ON config_apply_targets (updated_at) WHERE status = 'applying'`,
	}
	for _, statement := range intermediateStatements {
		_, err = db.Exec(statement)
		require.NoError(t, err)
	}

	firstBatch, secondBatch := uuid.New(), uuid.New()
	_, err = db.Exec(`
INSERT INTO config_apply_batches (id, category, config_version, status)
VALUES ($1, 'storage', 1, 'applying'), ($2, 'storage', 2, 'applying')
`, firstBatch, secondBatch)
	require.NoError(t, err)
	_, err = db.Exec(`
INSERT INTO config_apply_targets (batch_id, target, status)
VALUES ($1, 'alarm_history_retention', 'applying'), ($2, 'alarm_history_retention', 'applying')
`, firstBatch, secondBatch)
	require.NoError(t, err)

	migrationDir := t.TempDir()
	contents, err := os.ReadFile("../../migrations/000002_sys_config_apply_status.sql")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(migrationDir, "000002_sys_config_apply_status.sql"), contents, 0o600))

	require.NoError(t, goose.Up(db, migrationDir, goose.WithAllowMissing()))

	var targetCount, categorizedCount, applyingCount, failedCount int
	err = db.QueryRow(`
SELECT count(*),
       count(*) FILTER (WHERE category = 'storage'),
       count(*) FILTER (WHERE status = 'applying'),
       count(*) FILTER (WHERE status = 'failed')
FROM config_apply_targets
`).Scan(&targetCount, &categorizedCount, &applyingCount, &failedCount)
	require.NoError(t, err)
	require.Equal(t, 2, targetCount)
	require.Equal(t, 2, categorizedCount)
	require.Equal(t, 1, applyingCount)
	require.Equal(t, 1, failedCount)

	var leaseColumnExists bool
	require.NoError(t, db.QueryRow(`
SELECT EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema = current_schema()
      AND table_name = 'config_apply_targets'
      AND column_name = 'lease_expires_at'
      AND is_nullable = 'YES'
)
`).Scan(&leaseColumnExists))
	require.True(t, leaseColumnExists)

	var runningIndex, recoveringIndex string
	require.NoError(t, db.QueryRow(`SELECT indexdef FROM pg_indexes WHERE schemaname = current_schema() AND indexname = 'uq_config_apply_target_running'`).Scan(&runningIndex))
	require.Contains(t, runningIndex, "category, target")
	require.Contains(t, runningIndex, "status")
	require.NoError(t, db.QueryRow(`SELECT indexdef FROM pg_indexes WHERE schemaname = current_schema() AND indexname = 'idx_config_apply_targets_recovering'`).Scan(&recoveringIndex))
	require.Contains(t, recoveringIndex, "lease_expires_at")

	var currentVersion int64
	require.NoError(t, db.QueryRow(`SELECT config_version FROM config_apply_versions WHERE category = 'storage'`).Scan(&currentVersion))
	require.Equal(t, int64(2), currentVersion)

	var nextBatchID uuid.UUID
	require.NoError(t, db.QueryRow(`
INSERT INTO config_apply_batches (category, config_version, status)
VALUES ('storage', $1, 'pending')
RETURNING id
`, currentVersion+1).Scan(&nextBatchID))
	require.NotEqual(t, uuid.Nil, nextBatchID)
}

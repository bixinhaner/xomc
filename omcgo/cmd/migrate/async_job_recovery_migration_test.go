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

func TestAsyncJobRecoveryMigrationGooseUpDownUp(t *testing.T) {
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

	_, err = db.Exec(`
CREATE TABLE async_jobs (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    job_type text NOT NULL,
    status text NOT NULL,
    bucket_start timestamptz,
    bucket_end timestamptz
)`)
	require.NoError(t, err)

	migrationDir := t.TempDir()
	contents, err := os.ReadFile("../../migrations/000005_async_job_recovery.sql")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(
		filepath.Join(migrationDir, "000005_async_job_recovery.sql"),
		contents,
		0o600,
	))

	require.NoError(t, goose.Up(db, migrationDir, goose.WithAllowMissing()))
	assertAsyncJobRecoverySchema(t, db, true)

	require.NoError(t, goose.Down(db, migrationDir))
	assertAsyncJobRecoverySchema(t, db, false)

	require.NoError(t, goose.Up(db, migrationDir, goose.WithAllowMissing()))
	assertAsyncJobRecoverySchema(t, db, true)
}

func assertAsyncJobRecoverySchema(t *testing.T, db *sql.DB, want bool) {
	t.Helper()
	var columns, indexes int
	require.NoError(t, db.QueryRow(`
SELECT count(*)
  FROM information_schema.columns
 WHERE table_schema=current_schema()
   AND table_name='async_jobs'
   AND column_name IN ('recovery_count','last_recovered_at')`).Scan(&columns))
	require.NoError(t, db.QueryRow(`
SELECT count(*)
  FROM pg_indexes
 WHERE schemaname=current_schema()
   AND indexname='idx_async_jobs_hourly_failed_recovery'`).Scan(&indexes))
	if want {
		require.Equal(t, 2, columns)
		require.Equal(t, 1, indexes)
		return
	}
	require.Equal(t, 0, columns)
	require.Equal(t, 0, indexes)
}

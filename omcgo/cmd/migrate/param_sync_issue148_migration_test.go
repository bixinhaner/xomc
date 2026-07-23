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

func TestParamSyncIssue148MigrationDownPreservesNewTriggerRows(t *testing.T) {
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

	for _, table := range []string{"parameter_sync_requests", "parameter_sync_runs"} {
		_, err = db.Exec(fmt.Sprintf(`
CREATE TABLE %s (
    trigger_reason varchar(32) NOT NULL,
    CONSTRAINT %s_trigger_reason_chk CHECK (trigger_reason = 'bootstrap')
)`, table, table))
		require.NoError(t, err)
	}

	migrationDir := t.TempDir()
	contents, err := os.ReadFile("../../migrations/000004_param_sync_issue_148_trigger_reasons.sql")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(
		filepath.Join(migrationDir, "000004_param_sync_issue_148_trigger_reasons.sql"),
		contents,
		0o600,
	))

	require.NoError(t, goose.Up(db, migrationDir, goose.WithAllowMissing()))
	_, err = db.Exec(`
INSERT INTO parameter_sync_requests (trigger_reason) VALUES ('device_registered');
INSERT INTO parameter_sync_runs (trigger_reason) VALUES ('omc_upgrade');
`)
	require.NoError(t, err)

	require.NoError(t, goose.Down(db, migrationDir))

	var requestCount, runCount int
	require.NoError(t, db.QueryRow(`
SELECT
    (SELECT count(*) FROM parameter_sync_requests WHERE trigger_reason = 'device_registered'),
    (SELECT count(*) FROM parameter_sync_runs WHERE trigger_reason = 'omc_upgrade')
`).Scan(&requestCount, &runCount))
	require.Equal(t, 1, requestCount)
	require.Equal(t, 1, runCount)
}

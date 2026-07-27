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

func TestStandardParamsUpdatedFieldsMigrationUpgradesLegacySchema(t *testing.T) {
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
CREATE TABLE standard_params (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    standard_path text NOT NULL UNIQUE,
    entry_type varchar(16) NOT NULL,
    access varchar(16),
    data_type varchar(16),
    change_applies varchar(16),
    min_value bigint,
    max_value bigint,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    description text NOT NULL DEFAULT ''
)`)
	require.NoError(t, err)
	_, err = db.Exec(`
INSERT INTO standard_params (standard_path, entry_type, access, data_type)
VALUES ('Device.Test.Legacy', 'parameter', 'READ_ONLY', 'STRING')`)
	require.NoError(t, err)

	migrationDir := t.TempDir()
	contents, err := os.ReadFile("../../migrations/000003_add_standard_params_updated_fields.sql")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(migrationDir, "000003_add_standard_params_updated_fields.sql"), contents, 0o600))

	require.NoError(t, goose.Up(db, migrationDir, goose.WithAllowMissing()))

	var columnExists bool
	require.NoError(t, db.QueryRow(`
SELECT EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema = current_schema()
      AND table_name = 'standard_params'
      AND column_name = 'updated_fields'
      AND is_nullable = 'NO'
      AND column_default = '''{}''::text[]'
)`).Scan(&columnExists))
	require.True(t, columnExists)

	var updatedFields string
	require.NoError(t, db.QueryRow(`
SELECT updated_fields FROM standard_params WHERE standard_path = 'Device.Test.Legacy'
`).Scan(&updatedFields))
	require.Equal(t, "{}", updatedFields)
}

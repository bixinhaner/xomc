package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
)

func TestExtractMainReconcileSQLCollectsOrderedSections(t *testing.T) {
	contents := `ignored
-- +omcgo MainReconcileBegin
SELECT 1;
-- +omcgo MainReconcileEnd
ignored
-- +omcgo MainReconcileBegin
SELECT 2;
-- +omcgo MainReconcileEnd`

	actual, err := extractMainReconcileSQL(contents)
	require.NoError(t, err)
	require.Equal(t, "SELECT 1;\n\nSELECT 2;", actual)
}

func TestExtractMainReconcileSQLRejectsMalformedMarkers(t *testing.T) {
	for name, contents := range map[string]string{
		"missing sections": "SELECT 1;",
		"missing end":      mainReconcileBegin + "\nSELECT 1;",
		"orphan end":       mainReconcileEnd,
		"empty section":    mainReconcileBegin + "\n" + mainReconcileEnd,
	} {
		t.Run(name, func(t *testing.T) {
			_, err := extractMainReconcileSQL(contents)
			require.Error(t, err)
		})
	}
}

func TestMainBaselineReconcileSectionsAreAdditiveAndIdempotent(t *testing.T) {
	schemaPath := filepath.Join("..", "..", "migrations", "000001_init_schema.sql")
	seedPath := filepath.Join("..", "..", "migrations", "seed", "000001_init_seed.sql")

	schemaContents, err := os.ReadFile(schemaPath)
	require.NoError(t, err)
	schemaSQL, err := extractMainReconcileSQL(string(schemaContents))
	require.NoError(t, err)

	for _, contract := range []string{
		"CREATE TABLE IF NOT EXISTS public.plug_and_play_policies",
		"ADD COLUMN IF NOT EXISTS current_step_name",
		"ADD COLUMN IF NOT EXISTS location_source_mode",
		"CREATE TABLE IF NOT EXISTS public.geofence_carrier_settings",
		"CREATE TABLE IF NOT EXISTS public.geofence_definitions",
		"CREATE TABLE IF NOT EXISTS public.event_outbox",
		"CREATE TABLE IF NOT EXISTS public.geofence_batch_items",
		"CREATE UNIQUE INDEX IF NOT EXISTS uq_async_jobs_geofence_manual_bind_request",
		"main baseline reconcile missing devices.location_source_mode",
	} {
		require.Contains(t, schemaSQL, contract)
	}
	require.NotContains(t, schemaSQL, "DROP TABLE")
	require.NotContains(t, schemaSQL, "DROP COLUMN")
	require.NotContains(t, schemaSQL, "TRUNCATE")
	require.NotContains(t, schemaSQL, "DELETE FROM")

	seedContents, err := os.ReadFile(seedPath)
	require.NoError(t, err)
	seedSQL, err := extractMainReconcileSQL(string(seedContents))
	require.NoError(t, err)
	require.Contains(t, seedSQL, "INSERT INTO public.geofence_carrier_settings")
	require.Contains(t, seedSQL, "INSERT INTO public.api_endpoints")
	require.Contains(t, seedSQL, "INSERT INTO public.role_api_permissions")
	require.Contains(t, seedSQL, "INSERT INTO public.menus")
	require.Contains(t, seedSQL, "ON CONFLICT")
}

func TestMainSeedMigrationDirTriggersBaselineReconcile(t *testing.T) {
	require.True(t, isMainSeedMigrationDir("/etc/omcgo/migrations/seed"))
	require.True(t, isMainSeedMigrationDir("migrations/seed/"))
	require.False(t, isMainSeedMigrationDir("/etc/omcgo/migrations"))
	require.False(t, isMainSeedMigrationDir("/etc/omcgo/migrations/tsdb"))
}

func TestMainBaselineReconcileRepairsAndReplaysPostgreSQL16(t *testing.T) {
	dsn := os.Getenv("OMCGO_TEST_DB_DSN")
	if dsn == "" {
		t.Skip("OMCGO_TEST_DB_DSN not set")
	}

	adminDB, err := sql.Open("pgx", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, adminDB.Close()) })
	require.NoError(t, adminDB.Ping())

	databaseName := "main_reconcile_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	_, err = adminDB.Exec(fmt.Sprintf(`CREATE DATABASE %q`, databaseName))
	require.NoError(t, err)

	databaseDSN := geofenceBatchMigrationDatabaseDSN(t, dsn, databaseName)
	db, err := sql.Open("pgx", databaseDSN)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, db.Close())
		_, cleanupErr := adminDB.Exec(fmt.Sprintf(`DROP DATABASE IF EXISTS %q WITH (FORCE)`, databaseName))
		require.NoError(t, cleanupErr)
	})
	require.NoError(t, db.Ping())

	migrationDir := geofenceBatchMigrationDirectory(t)
	require.NoError(t, goose.Up(db, migrationDir))
	goose.SetTableName("goose_db_version_seed")
	t.Cleanup(func() { goose.SetTableName("goose_db_version") })
	require.NoError(t, goose.Up(db, filepath.Join(migrationDir, "seed")))

	_, err = db.Exec(`
DROP TABLE public.geofence_carrier_settings;
ALTER TABLE public.devices DROP COLUMN location_source_mode;
ALTER TABLE public.provisioning_tasks
    DROP COLUMN policy_id,
    DROP COLUMN current_step_name;
`)
	require.NoError(t, err)

	seedDir := filepath.Join(migrationDir, "seed")
	for range 2 {
		require.NoError(t, reconcileMainBaselineSchema(db, seedDir))
		require.NoError(t, goose.Up(db, seedDir))
		require.NoError(t, reconcileMainBaselineSeed(db, seedDir))
	}

	for _, relation := range []string{
		"public.geofence_carrier_settings",
		"public.geofence_definitions",
		"public.event_outbox",
	} {
		var exists bool
		require.NoError(t, db.QueryRow(`SELECT to_regclass($1) IS NOT NULL`, relation).Scan(&exists))
		require.True(t, exists, relation)
	}
	for table, column := range map[string]string{
		"devices":            "location_source_mode",
		"provisioning_tasks": "policy_id",
	} {
		var exists bool
		require.NoError(t, db.QueryRow(`
SELECT EXISTS (
    SELECT 1
    FROM information_schema.columns
    WHERE table_schema = 'public' AND table_name = $1 AND column_name = $2
)`, table, column).Scan(&exists))
		require.True(t, exists, table+"."+column)
	}

	var carrierRows int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM public.geofence_carrier_settings`).Scan(&carrierRows))
	require.Equal(t, 3, carrierRows)
}

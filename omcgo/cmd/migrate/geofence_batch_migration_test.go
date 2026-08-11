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

func TestGeofenceBatchMigrationContract(t *testing.T) {
	contents, err := os.ReadFile("../../migrations/000001_init_schema.sql")
	require.NoError(t, err)
	sql := string(contents)

	require.Contains(t, sql, "-- +goose Up")
	require.Contains(t, sql, "CREATE TABLE IF NOT EXISTS public.geofence_batch_items")
	require.Contains(t, sql, "REFERENCES public.async_jobs(id) ON DELETE RESTRICT")
	require.Contains(t, sql, "CHECK (status IN ('pending', 'succeeded', 'skipped', 'failed'))")
	require.Contains(t, sql, "UNIQUE (job_id, input_key)")
	require.Contains(t, sql, "uq_geofence_batch_items_job_device")
	require.Contains(t, sql, "uq_async_jobs_geofence_manual_bind_request")
	require.Contains(t, sql, "payload->>'preview_fingerprint'")
	require.Contains(t, sql, "-- +goose Down")
	require.FileExists(t, "../../migrations/000001_init_schema.sql")
}

func TestGeofenceBatchBaselineFreshUpDownUpPostgreSQL16(t *testing.T) {
	dsn := os.Getenv("OMCGO_TEST_DB_DSN")
	if dsn == "" {
		t.Skip("OMCGO_TEST_DB_DSN not set")
	}

	adminDB, err := sql.Open("pgx", dsn)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, adminDB.Close())
	})
	require.NoError(t, adminDB.Ping())

	databaseName := "geofence_batch_migration_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	_, err = adminDB.Exec(fmt.Sprintf(`CREATE DATABASE %q`, databaseName))
	require.NoError(t, err)

	databaseDSN := geofenceBatchMigrationDatabaseDSN(t, dsn, databaseName)
	var db *sql.DB
	t.Cleanup(func() {
		if db != nil {
			require.NoError(t, db.Close())
		}
		_, cleanupErr := adminDB.Exec(fmt.Sprintf(`DROP DATABASE IF EXISTS %q WITH (FORCE)`, databaseName))
		require.NoError(t, cleanupErr)
	})

	db, err = sql.Open("pgx", databaseDSN)
	require.NoError(t, err)
	require.NoError(t, db.Ping())

	var serverVersion string
	require.NoError(t, db.QueryRow(`SHOW server_version_num`).Scan(&serverVersion))
	require.True(t, strings.HasPrefix(serverVersion, "16"), "requires PostgreSQL 16, got %s", serverVersion)

	migrationDir := geofenceBatchMigrationDirectory(t)

	require.NoError(t, goose.Up(db, migrationDir))
	requireBaselineVersion(t, db, 1)
	requireGeofenceBatchMigrationContractInDatabase(t, db, "first")

	require.NoError(t, goose.Down(db, migrationDir))
	requireGeofenceBatchObjectsAbsent(t, db)

	require.NoError(t, goose.Up(db, migrationDir))
	requireBaselineVersion(t, db, 1)
	requireGeofenceBatchMigrationContractInDatabase(t, db, "second")
}

func geofenceBatchMigrationDatabaseDSN(t *testing.T, dsn, databaseName string) string {
	t.Helper()
	parsedDSN, err := url.Parse(dsn)
	require.NoError(t, err)
	parsedDSN.Path = "/" + databaseName
	return parsedDSN.String()
}

func geofenceBatchMigrationDirectory(t *testing.T) string {
	t.Helper()
	migrationDir, err := filepath.Abs("../../migrations")
	require.NoError(t, err)
	return migrationDir
}

func requireBaselineVersion(t *testing.T, db *sql.DB, want int64) {
	t.Helper()
	var got int64
	require.NoError(t, db.QueryRow(`
SELECT COALESCE(MAX(version_id), 0)
FROM public.goose_db_version
WHERE is_applied`).Scan(&got))
	require.Equal(t, want, got)
}

func requireGeofenceBatchMigrationContractInDatabase(t *testing.T, db *sql.DB, payloadNamespace string) {
	t.Helper()
	requireGeofenceBatchItemSchema(t, db)
	requireGeofenceBatchIndexes(t, db)
	requireGeofenceManualBindDuplicateBehavior(t, db, payloadNamespace)
}

type geofenceBatchColumnSpec struct {
	Name       string
	DataType   string
	UDTName    string
	Nullable   string
	DefaultSQL string
}

func requireGeofenceBatchItemSchema(t *testing.T, db *sql.DB) {
	t.Helper()
	rows, err := db.Query(`
SELECT column_name, data_type, udt_name, is_nullable, column_default
FROM information_schema.columns
WHERE table_schema = 'public' AND table_name = 'geofence_batch_items'
ORDER BY ordinal_position`)
	require.NoError(t, err)
	defer rows.Close()

	var actual []geofenceBatchColumnSpec
	for rows.Next() {
		var column geofenceBatchColumnSpec
		var columnDefault sql.NullString
		require.NoError(t, rows.Scan(
			&column.Name,
			&column.DataType,
			&column.UDTName,
			&column.Nullable,
			&columnDefault,
		))
		column.DefaultSQL = normalizeCatalogSQL(columnDefault.String)
		actual = append(actual, column)
	}
	require.NoError(t, rows.Err())
	require.Equal(t, []geofenceBatchColumnSpec{
		{"id", "uuid", "uuid", "NO", "gen_random_uuid()"},
		{"job_id", "uuid", "uuid", "NO", ""},
		{"geofence_id", "uuid", "uuid", "NO", ""},
		{"input_key", "character varying", "varchar", "NO", ""},
		{"input_kind", "character varying", "varchar", "NO", ""},
		{"input_value", "character varying", "varchar", "NO", ""},
		{"device_id", "uuid", "uuid", "YES", ""},
		{"device_sn_snapshot", "character varying", "varchar", "YES", ""},
		{"status", "character varying", "varchar", "NO", "'pending'::charactervarying"},
		{"reason_code", "character varying", "varchar", "YES", ""},
		{"error_message", "text", "text", "YES", ""},
		{"expected_source_binding_id", "uuid", "uuid", "YES", ""},
		{"binding_id", "uuid", "uuid", "YES", ""},
		{"attempt", "integer", "int4", "NO", "0"},
		{"started_at", "timestamp with time zone", "timestamptz", "YES", ""},
		{"finished_at", "timestamp with time zone", "timestamptz", "YES", ""},
		{"created_at", "timestamp with time zone", "timestamptz", "NO", "now()"},
		{"updated_at", "timestamp with time zone", "timestamptz", "NO", "now()"},
	}, actual)

	requireGeofenceBatchForeignKeys(t, db)
	requireGeofenceBatchChecks(t, db)
}

type geofenceBatchForeignKeySpec struct {
	SourceColumn string
	TargetSchema string
	TargetTable  string
	TargetColumn string
	DeleteRule   string
}

func requireGeofenceBatchForeignKeys(t *testing.T, db *sql.DB) {
	t.Helper()
	rows, err := db.Query(`
SELECT source_attribute.attname,
       target_namespace.nspname,
       target_relation.relname,
       target_attribute.attname,
       item.confdeltype
FROM pg_constraint AS item
JOIN pg_class AS relation ON relation.oid = item.conrelid
JOIN pg_namespace AS namespace ON namespace.oid = relation.relnamespace
JOIN pg_class AS target_relation ON target_relation.oid = item.confrelid
JOIN pg_namespace AS target_namespace ON target_namespace.oid = target_relation.relnamespace
JOIN unnest(item.conkey) WITH ORDINALITY AS source_key(attnum, ordinal) ON true
JOIN unnest(item.confkey) WITH ORDINALITY AS target_key(attnum, ordinal)
  ON target_key.ordinal = source_key.ordinal
JOIN pg_attribute AS source_attribute
  ON source_attribute.attrelid = relation.oid AND source_attribute.attnum = source_key.attnum
JOIN pg_attribute AS target_attribute
  ON target_attribute.attrelid = target_relation.oid AND target_attribute.attnum = target_key.attnum
WHERE namespace.nspname = 'public'
  AND relation.relname = 'geofence_batch_items'
  AND item.contype = 'f'
ORDER BY source_attribute.attname`)
	require.NoError(t, err)
	defer rows.Close()

	var actual []geofenceBatchForeignKeySpec
	for rows.Next() {
		var foreignKey geofenceBatchForeignKeySpec
		require.NoError(t, rows.Scan(
			&foreignKey.SourceColumn,
			&foreignKey.TargetSchema,
			&foreignKey.TargetTable,
			&foreignKey.TargetColumn,
			&foreignKey.DeleteRule,
		))
		actual = append(actual, foreignKey)
	}
	require.NoError(t, rows.Err())
	require.Equal(t, []geofenceBatchForeignKeySpec{
		{"binding_id", "public", "device_geofence_bindings", "id", "r"},
		{"expected_source_binding_id", "public", "device_geofence_bindings", "id", "r"},
		{"geofence_id", "public", "geofence_definitions", "id", "r"},
		{"job_id", "public", "async_jobs", "id", "r"},
	}, actual)
}

func requireGeofenceBatchChecks(t *testing.T, db *sql.DB) {
	t.Helper()
	rows, err := db.Query(`
SELECT item.conname, pg_get_constraintdef(item.oid)
FROM pg_constraint AS item
JOIN pg_class AS relation ON relation.oid = item.conrelid
JOIN pg_namespace AS namespace ON namespace.oid = relation.relnamespace
WHERE namespace.nspname = 'public'
  AND relation.relname = 'geofence_batch_items'
  AND item.contype = 'c'
ORDER BY item.conname`)
	require.NoError(t, err)
	defer rows.Close()

	actual := make(map[string]string)
	for rows.Next() {
		var name, definition string
		require.NoError(t, rows.Scan(&name, &definition))
		actual[name] = normalizeCatalogSQL(definition)
	}
	require.NoError(t, rows.Err())
	require.Equal(t, map[string]string{
		"geofence_batch_items_attempt_check":         "check((attempt>=0))",
		"geofence_batch_items_input_kind_check":      "check(((input_kind)::text=any((array['device_id'::charactervarying,'device_sn'::charactervarying])::text[])))",
		"geofence_batch_items_status_check":          "check(((status)::text=any((array['pending'::charactervarying,'succeeded'::charactervarying,'skipped'::charactervarying,'failed'::charactervarying])::text[])))",
		"geofence_batch_items_success_binding_check": "check((((status)::text<>'succeeded'::text)or(binding_idisnotnull)))",
	}, actual)
}

func requireGeofenceBatchIndexes(t *testing.T, db *sql.DB) {
	t.Helper()
	rows, err := db.Query(`
SELECT indexname, indexdef
FROM pg_indexes
WHERE schemaname = 'public'
  AND indexname = ANY ($1)
ORDER BY indexname`, []string{
		"geofence_batch_items_pkey",
		"uq_geofence_batch_items_job_input",
		"uq_geofence_batch_items_job_device",
		"idx_geofence_batch_items_job_status",
		"idx_geofence_batch_items_geofence_status",
		"uq_async_jobs_geofence_manual_bind_request",
	})
	require.NoError(t, err)
	defer rows.Close()

	actual := make(map[string]string)
	for rows.Next() {
		var name, definition string
		require.NoError(t, rows.Scan(&name, &definition))
		actual[name] = normalizeCatalogSQL(definition)
	}
	require.NoError(t, rows.Err())
	require.Equal(t, map[string]string{
		"geofence_batch_items_pkey":                  "createuniqueindexgeofence_batch_items_pkeyonpublic.geofence_batch_itemsusingbtree(id)",
		"uq_geofence_batch_items_job_input":          "createuniqueindexuq_geofence_batch_items_job_inputonpublic.geofence_batch_itemsusingbtree(job_id,input_key)",
		"uq_geofence_batch_items_job_device":         "createuniqueindexuq_geofence_batch_items_job_deviceonpublic.geofence_batch_itemsusingbtree(job_id,device_id)where(device_idisnotnull)",
		"idx_geofence_batch_items_job_status":        "createindexidx_geofence_batch_items_job_statusonpublic.geofence_batch_itemsusingbtree(job_id,status,created_at,id)",
		"idx_geofence_batch_items_geofence_status":   "createindexidx_geofence_batch_items_geofence_statusonpublic.geofence_batch_itemsusingbtree(geofence_id,status)",
		"uq_async_jobs_geofence_manual_bind_request": "createuniqueindexuq_async_jobs_geofence_manual_bind_requestonpublic.async_jobsusingbtree(job_type,((payload->>'geofence_id'::text)),((payload->>'requested_by'::text)),((payload->>'preview_fingerprint'::text)))where((job_type='geofence_manual_bind'::text)and(status=any(array['pending'::text,'running'::text,'succeeded'::text])))",
	}, actual)
}

func requireGeofenceManualBindDuplicateBehavior(t *testing.T, db *sql.DB, payloadNamespace string) {
	t.Helper()
	for _, status := range []string{"pending", "running", "succeeded"} {
		status := status
		t.Run("rejects duplicate "+status+" request", func(t *testing.T) {
			payload := geofenceManualBindPayload(payloadNamespace, status)
			insertGeofenceManualBindJob(t, db, status, payload)
			_, err := db.Exec(`
INSERT INTO public.async_jobs (job_type, status, scheduled_at, payload)
VALUES ('geofence_manual_bind', 'pending', now(), $1::jsonb)`, payload)
			require.Error(t, err)
		})
	}

	for _, status := range []string{"failed", "canceled", "zombie"} {
		status := status
		t.Run("allows duplicate "+status+" request", func(t *testing.T) {
			payload := geofenceManualBindPayload(payloadNamespace, status)
			insertGeofenceManualBindJob(t, db, status, payload)
			insertGeofenceManualBindJob(t, db, status, payload)
		})
	}
}

func geofenceManualBindPayload(namespace, status string) string {
	return fmt.Sprintf(`{"geofence_id":"geofence-%s-%s","requested_by":"operator","preview_fingerprint":"fingerprint-%s-%s"}`, namespace, status, namespace, status)
}

func insertGeofenceManualBindJob(t *testing.T, db *sql.DB, status, payload string) {
	t.Helper()
	_, err := db.Exec(`
INSERT INTO public.async_jobs (job_type, status, scheduled_at, payload)
VALUES ('geofence_manual_bind', $1, now(), $2::jsonb)`, status, payload)
	require.NoError(t, err)
}

func requireGeofenceBatchObjectsAbsent(t *testing.T, db *sql.DB) {
	t.Helper()
	for _, object := range []string{
		"public.geofence_batch_items",
		"public.uq_async_jobs_geofence_manual_bind_request",
		"public.async_jobs",
		"public.geofence_definitions",
	} {
		var exists bool
		require.NoError(t, db.QueryRow(`SELECT to_regclass($1) IS NOT NULL`, object).Scan(&exists))
		require.False(t, exists, "%s should be absent after Down", object)
	}
}

func normalizeCatalogSQL(value string) string {
	return strings.ToLower(strings.Join(strings.Fields(value), ""))
}

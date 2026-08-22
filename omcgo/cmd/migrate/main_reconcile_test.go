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

func TestExtractMainReconcileSectionsPreservesBoundaries(t *testing.T) {
	contents := `ignored
-- +omcgo MainReconcileBegin
SELECT 1;
-- +omcgo MainReconcileEnd
ignored
-- +omcgo MainReconcileBegin
SELECT 2;
-- +omcgo MainReconcileEnd`

	actual, err := extractMainReconcileSections(contents)
	require.NoError(t, err)
	require.Equal(t, []string{"SELECT 1;", "SELECT 2;"}, actual)
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
		"CREATE TABLE IF NOT EXISTS public.device_access_policy_sets",
		"CREATE TABLE IF NOT EXISTS public.device_access_actions",
		"ADD COLUMN IF NOT EXISTS admission_class",
		"CREATE UNIQUE INDEX IF NOT EXISTS uq_async_jobs_geofence_manual_bind_request",
		"CREATE TABLE IF NOT EXISTS public.alarm_email_global_settings",
		"CREATE TABLE IF NOT EXISTS public.alarm_email_subscriptions",
		"ADD COLUMN IF NOT EXISTS legacy_filter_id",
		"CREATE TABLE IF NOT EXISTS public.alarm_email_runs",
		"CREATE TABLE IF NOT EXISTS public.alarm_email_deliveries",
		"ADD COLUMN IF NOT EXISTS subscription_snapshot",
		"ADD COLUMN IF NOT EXISTS recipients_snapshot",
		"uq_alarm_email_runs_subscription_window",
		"uq_alarm_email_deliveries_run_recipient",
		"uq_alarm_email_subscriptions_legacy_filter",
		"main baseline reconcile missing devices.location_source_mode",
		"'public.device_access_states'",
		"main baseline reconcile missing relation %",
		"main baseline reconcile missing alarm email run snapshots",
		"main baseline reconcile missing legacy alarm email linkage",
		"main baseline reconcile missing device_tasks.admission_class",
		"CREATE TABLE IF NOT EXISTS public.device_task_locations",
		"CREATE OR REPLACE FUNCTION public.sync_device_task_location",
		"trg_device_tasks_location_sync",
	} {
		require.Contains(t, schemaSQL, contract)
	}
	require.NotContains(t, schemaSQL, "DROP TABLE")
	require.NotContains(t, schemaSQL, "DROP COLUMN")
	require.NotContains(t, schemaSQL, "TRUNCATE")
	schemaSQLWithoutLocationTriggerCleanup := strings.ReplaceAll(
		schemaSQL,
		"DELETE FROM public.device_task_locations WHERE task_id = OLD.id;",
		"",
	)
	require.NotContains(t, schemaSQLWithoutLocationTriggerCleanup, "DELETE FROM")
	require.NotContains(t, schemaSQL, "INSERT INTO public.device_task_locations (task_id, device_sn)\nSELECT id, device_sn\nFROM public.device_tasks")

	seedContents, err := os.ReadFile(seedPath)
	require.NoError(t, err)
	seedSQL, err := extractMainReconcileSQL(string(seedContents))
	require.NoError(t, err)
	require.Contains(t, seedSQL, "INSERT INTO public.geofence_carrier_settings")
	require.Contains(t, seedSQL, "INSERT INTO public.api_endpoints")
	require.Contains(t, seedSQL, "INSERT INTO public.role_api_permissions")
	require.Contains(t, seedSQL, "INSERT INTO public.menus")
	require.Contains(t, seedSQL, "'device:access-control'")
	require.Contains(t, seedSQL, "'/api/v1/device-access/states'")
	require.Contains(t, seedSQL, "'notification.email', 'enabled'")
	require.Contains(t, seedSQL, "'/api/v1/admin/notification/email/test'")
	require.Contains(t, seedSQL, "'/api/v1/alarms/email-subscriptions'")
	require.Contains(t, seedSQL, "'/api/v1/notifications/email-runs'")
	require.Contains(t, seedSQL, "filter_rule.action = 'notify_email'")
	require.Contains(t, seedSQL, "legacy_filter_id")
	require.Contains(t, seedSQL, "migrated_legacy_email_rules")
	require.Contains(t, seedSQL, "ON CONFLICT")
}

func TestMainSeedMigrationDirTriggersBaselineReconcile(t *testing.T) {
	require.True(t, isMainSeedMigrationDir("/etc/omcgo/migrations/seed"))
	require.True(t, isMainSeedMigrationDir("migrations/seed/"))
	require.False(t, isMainSeedMigrationDir("/etc/omcgo/migrations"))
	require.False(t, isMainSeedMigrationDir("/etc/omcgo/migrations/tsdb"))
	require.False(t, isMainSeedMigrationDir("/etc/omcgo/migrations/tsdb/seed"))
	require.True(t, isMainSchemaMigrationDir("/etc/omcgo/migrations"))
	require.True(t, isMainSchemaMigrationDir("migrations/"))
	require.False(t, isMainSchemaMigrationDir("/etc/omcgo/migrations/seed"))
	require.False(t, isMainSchemaMigrationDir("/etc/omcgo/migrations/tsdb"))
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
DROP TABLE
	public.device_access_action_attempts,
    public.device_access_actions,
    public.device_access_decision_checks,
	public.device_access_decision_archives,
	public.device_access_identity_snapshots,
    public.device_access_decisions,
    public.device_access_states,
    public.device_access_evidence,
    public.device_access_candidates,
	public.device_access_import_rows,
    public.device_access_conditions,
    public.device_access_rules,
	public.device_access_import_batches,
    public.device_access_policy_versions,
    public.device_access_policy_sets,
    public.device_access_list_entries,
    public.device_access_outbox;
ALTER TABLE public.device_tasks DROP COLUMN admission_class;
ALTER TABLE public.devices DROP COLUMN location_source_mode;
ALTER TABLE public.provisioning_tasks
    DROP COLUMN policy_id,
    DROP COLUMN current_step_name;

DELETE FROM public.role_menus
WHERE menu_id::text LIKE 'da000001-%';
DELETE FROM public.menus
WHERE id::text LIKE 'da000001-%';
DELETE FROM public.role_api_permissions
WHERE endpoint_id IN (
    SELECT id FROM public.api_endpoints WHERE api_group = 'device-access'
);
DELETE FROM public.api_endpoints
WHERE api_group = 'device-access';
`)
	require.NoError(t, err)

	legacyFilterID := uuid.New()
	_, err = db.Exec(`
INSERT INTO public.alarm_filters (
    id, name, filter_type, alarm_sources, alarm_identifiers,
    action, email_recipients, priority, enabled
) VALUES (
    $1, 'legacy-email-rule', 'alarm_identifier', ARRAY['device']::varchar[], ARRAY['11500']::varchar[],
    'notify_email', ARRAY['OPS@example.com', 'ops@example.com'], 10, true
)
`, legacyFilterID)
	require.NoError(t, err)

	seedDir := filepath.Join(migrationDir, "seed")
	for range 2 {
		require.NoError(t, reconcileMainBaselineSchema(db, seedDir))
		require.NoError(t, goose.Up(db, seedDir))
		require.NoError(t, reconcileMainBaselineSeed(db, seedDir))
	}

	// Older pre-release databases could retain the reject-only policy check
	// under an unexpected name. Reconciliation must remove every stale check,
	// not only the canonical constraint name.
	_, err = db.Exec(`
ALTER TABLE public.device_access_policy_versions
    ADD CONSTRAINT legacy_reject_only_default_action
    CHECK (default_action = 'reject')
`)
	require.NoError(t, err)
	require.NoError(t, reconcileMainBaselineSchema(db, seedDir))

	var defaultActionChecks int
	require.NoError(t, db.QueryRow(`
SELECT COUNT(*)
FROM pg_catalog.pg_constraint constraint_row
WHERE constraint_row.conrelid = 'public.device_access_policy_versions'::regclass
  AND constraint_row.contype = 'c'
  AND pg_get_constraintdef(constraint_row.oid) ILIKE '%default_action%'
`).Scan(&defaultActionChecks))
	require.Equal(t, 1, defaultActionChecks)

	policySetID := uuid.New()
	_, err = db.Exec(`
INSERT INTO public.device_access_policy_sets (id, name, carrier)
VALUES ($1, 'review-policy', 'cmcc')
`, policySetID)
	require.NoError(t, err)
	_, err = db.Exec(`
INSERT INTO public.device_access_policy_versions (
    policy_set_id, version, status, default_action, failure_mode,
    collection_timeout_seconds, content_hash
) VALUES ($1, 1, 'draft', 'review', 'review_hold', 900, $2)
`, policySetID, strings.Repeat("a", 64))
	require.NoError(t, err)

	candidateID, decisionID, actionID := uuid.New(), uuid.New(), uuid.New()
	_, err = db.Exec(`
INSERT INTO public.device_access_candidates (id, carrier, serial_number, oui, expires_at)
VALUES ($1, 'cmcc', 'LEGACY-RF-1', 'AABBCC', now() + interval '1 day')
`, candidateID)
	require.NoError(t, err)
	_, err = db.Exec(`
INSERT INTO public.device_access_decisions (
    id, carrier, serial_number, candidate_id, trigger_type, trigger_event_id,
    new_state, decision, reason_code, decision_version
) VALUES ($2, 'cmcc', 'LEGACY-RF-1', $1, 'inform', 'legacy-event', 'rejected', 'reject', 'denylist', 1)
`, candidateID, decisionID)
	require.NoError(t, err)
	_, err = db.Exec(`
INSERT INTO public.device_access_actions (
    id, candidate_id, decision_id, action_type, direction, status,
    idempotency_key, attempts, completed_at
) VALUES ($3, $1, $2, 'rf_off', 'contain', 'succeeded', 'legacy-action', 0, now())
`, candidateID, decisionID, actionID)
	require.NoError(t, err)
	require.NoError(t, reconcileMainBaselineSchema(db, seedDir))

	var actionStatus, failureCode string
	var repairRequired bool
	require.NoError(t, db.QueryRow(`
SELECT status, last_failure_code, manual_repair_required
FROM public.device_access_actions
WHERE id = $1
`, actionID).Scan(&actionStatus, &failureCode, &repairRequired))
	require.Equal(t, "dead", actionStatus)
	require.Equal(t, "evidence_missing", failureCode)
	require.True(t, repairRequired)

	for _, relation := range []string{
		"public.geofence_carrier_settings",
		"public.geofence_definitions",
		"public.event_outbox",
		"public.device_access_policy_sets",
		"public.device_access_states",
		"public.device_access_actions",
	} {
		var exists bool
		require.NoError(t, db.QueryRow(`SELECT to_regclass($1) IS NOT NULL`, relation).Scan(&exists))
		require.True(t, exists, relation)
	}
	for table, column := range map[string]string{
		"devices":            "location_source_mode",
		"device_tasks":       "admission_class",
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

	var accessMenus, accessEndpoints int
	require.NoError(t, db.QueryRow(`
SELECT COUNT(*) FROM public.menus WHERE permission_key = 'device:access-control'
`).Scan(&accessMenus))
	require.Equal(t, 1, accessMenus)
	require.NoError(t, db.QueryRow(`
SELECT COUNT(*) FROM public.api_endpoints WHERE api_group = 'device-access'
`).Scan(&accessEndpoints))
	require.Equal(t, 39, accessEndpoints)

	var emailConfigRows int
	require.NoError(t, db.QueryRow(`
SELECT COUNT(*)
FROM public.sys_configs
WHERE category = 'notification.email'
`).Scan(&emailConfigRows))
	require.Equal(t, 10, emailConfigRows)

	var emailEndpointRows int
	require.NoError(t, db.QueryRow(`
SELECT COUNT(*)
FROM public.api_endpoints
WHERE path IN (
    '/api/v1/admin/notification/email/test',
    '/api/v1/alarms/email-settings',
    '/api/v1/alarms/email-subscriptions',
    '/api/v1/alarms/email-subscriptions/:id',
    '/api/v1/notifications/email-runs',
    '/api/v1/notifications/email-runs/:business/:id/deliveries'
)
`).Scan(&emailEndpointRows))
	require.Equal(t, 9, emailEndpointRows)

	var emailPermissionRows int
	require.NoError(t, db.QueryRow(`
SELECT COUNT(*)
FROM public.role_api_permissions AS permission
JOIN public.api_endpoints AS endpoint ON endpoint.id = permission.endpoint_id
WHERE endpoint.path IN (
    '/api/v1/admin/notification/email/test',
    '/api/v1/alarms/email-settings',
    '/api/v1/alarms/email-subscriptions',
    '/api/v1/alarms/email-subscriptions/:id',
    '/api/v1/notifications/email-runs',
    '/api/v1/notifications/email-runs/:business/:id/deliveries'
)
`).Scan(&emailPermissionRows))
	require.Equal(t, 22, emailPermissionRows)

	var migratedLegacyRules int
	require.NoError(t, db.QueryRow(`
SELECT COUNT(*)
FROM public.alarm_email_subscriptions
WHERE legacy_filter_id = $1
  AND enabled
  AND interval_minutes = 0
  AND tolerance_minutes = 0
  AND NOT include_default_recipients
  AND recipients = ARRAY['ops@example.com']::text[]
  AND alarm_identifiers = ARRAY['11500']::text[]
`, legacyFilterID).Scan(&migratedLegacyRules))
	require.Equal(t, 1, migratedLegacyRules)

	var alarmEmailEnabled bool
	require.NoError(t, db.QueryRow(`
SELECT enabled FROM public.alarm_email_global_settings WHERE id = 1
`).Scan(&alarmEmailEnabled))
	require.True(t, alarmEmailEnabled)
}

package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDeviceAccessBaselineDefinesAuthorityTables(t *testing.T) {
	sql := readDeviceAccessBaseline(t)
	tables := []string{
		"device_access_policy_sets",
		"device_access_policy_versions",
		"device_access_rules",
		"device_access_conditions",
		"device_access_import_batches",
		"device_access_import_rows",
		"device_access_list_entries",
		"device_access_candidates",
		"device_access_evidence",
		"device_access_states",
		"device_access_decisions",
		"device_access_decision_checks",
		"device_access_decision_archives",
		"device_access_identity_snapshots",
		"device_access_actions",
		"device_access_action_attempts",
		"device_access_outbox",
	}

	for _, table := range tables {
		block := deviceAccessTableBlock(t, sql, table)
		require.Contains(t, block, "id uuid DEFAULT gen_random_uuid() NOT NULL", table)
	}
}

func TestDeviceAccessBaselineEnforcesTenantAndHistoryIntegrity(t *testing.T) {
	sql := readDeviceAccessBaseline(t)

	require.Contains(t, sql,
		"ADD CONSTRAINT device_access_candidates_carrier_serial_key UNIQUE (carrier, serial_number)")
	require.Contains(t, sql,
		"ADD CONSTRAINT device_access_states_carrier_serial_key UNIQUE (carrier, serial_number)")
	require.Contains(t, sql,
		"ADD CONSTRAINT device_access_decisions_version_key UNIQUE (carrier, serial_number, decision_version)")
	require.Contains(t, sql,
		"ADD CONSTRAINT device_access_list_entries_identity_key UNIQUE (carrier, entry_type, identity_type, identity_value)")
	require.Contains(t, sql,
		"ADD CONSTRAINT device_access_evidence_version_type_key UNIQUE (carrier, serial_number, evidence_version, evidence_type)")
	require.Contains(t, sql,
		"ADD CONSTRAINT device_access_policy_versions_set_version_key UNIQUE (policy_set_id, version)")
	require.Contains(t, sql,
		"CREATE UNIQUE INDEX uq_device_access_policy_sets_enabled_carrier ON public.device_access_policy_sets USING btree (carrier) WHERE (enabled = true)")
	require.Contains(t, sql,
		"CREATE UNIQUE INDEX uq_device_access_rules_version_priority ON public.device_access_rules USING btree (policy_version_id, priority)")
	require.Contains(t, sql,
		"CREATE UNIQUE INDEX uq_device_access_import_batches_idempotency ON public.device_access_import_batches USING btree (carrier, idempotency_key) WHERE (idempotency_key IS NOT NULL)")
	rules := deviceAccessTableBlock(t, sql, "device_access_rules")
	require.Contains(t, rules, "priority integer DEFAULT 0 NOT NULL")
	require.Contains(t, rules, "device_access_rules_priority_check CHECK ((priority >= 0))")
	imports := deviceAccessTableBlock(t, sql, "device_access_import_batches")
	require.Contains(t, imports, "device_access_import_batches_target_check")
	require.Contains(t, imports, "(target_policy_version_id IS NULL AND target_rule_id IS NULL)")
	require.Contains(t, imports, "device_access_import_batches_count_check")
	require.Contains(t, sql, "DROP CONSTRAINT IF EXISTS device_access_import_batches_target_check")
	rows := deviceAccessTableBlock(t, sql, "device_access_import_rows")
	require.Contains(t, rows, "device_access_import_rows_batch_row_key UNIQUE (batch_id, row_number)")
	listEntries := deviceAccessTableBlock(t, sql, "device_access_list_entries")
	require.Contains(t, listEntries, "source_batch_id uuid")
	require.Contains(t, listEntries, "device_access_list_entries_source_batch_id_fkey")
	candidates := deviceAccessTableBlock(t, sql, "device_access_candidates")
	require.Contains(t, candidates, "technology character varying(16) DEFAULT 'lte'::character varying NOT NULL")
	require.Contains(t, candidates, "rf_control_paths jsonb DEFAULT '[]'::jsonb NOT NULL")
	actions := deviceAccessTableBlock(t, sql, "device_access_actions")
	require.Contains(t, actions, "device_id uuid,")
	require.Contains(t, actions, "candidate_id uuid,")
	require.Contains(t, actions, "bound_session_id character varying(128)")
	require.Contains(t, actions, "bound_request_id character varying(128)")
	require.Contains(t, actions, "device_access_actions_target_check")
	require.Contains(t, actions, "device_access_actions_candidate_type_check")

	for _, table := range []string{"device_access_decisions", "device_access_decision_checks"} {
		block := deviceAccessTableBlock(t, sql, table)
		require.NotContains(t, block, "updated_at", table+" is immutable history")
	}
	decisions := deviceAccessTableBlock(t, sql, "device_access_decisions")
	require.Contains(t, decisions, "decision_version bigint NOT NULL")
	require.Contains(t, decisions, "device_access_decisions_version_check CHECK ((decision_version > 0))")

	for _, fragment := range []string{
		"device_access_policy_versions_policy_set_id_fkey FOREIGN KEY (policy_set_id) REFERENCES public.device_access_policy_sets(id)",
		"device_access_rules_policy_version_id_fkey FOREIGN KEY (policy_version_id) REFERENCES public.device_access_policy_versions(id)",
		"device_access_conditions_rule_id_fkey FOREIGN KEY (rule_id) REFERENCES public.device_access_rules(id)",
		"device_access_states_candidate_id_fkey FOREIGN KEY (candidate_id) REFERENCES public.device_access_candidates(id)",
		"device_access_decisions_candidate_id_fkey FOREIGN KEY (candidate_id) REFERENCES public.device_access_candidates(id)",
		"device_access_decision_checks_decision_id_fkey FOREIGN KEY (decision_id) REFERENCES public.device_access_decisions(id)",
		"device_access_decision_archives_decision_fkey FOREIGN KEY (decision_id) REFERENCES public.device_access_decisions(id)",
		"device_access_identity_snapshots_decision_fkey FOREIGN KEY (decision_id) REFERENCES public.device_access_decisions(id)",
		"device_access_actions_decision_id_fkey FOREIGN KEY (decision_id) REFERENCES public.device_access_decisions(id)",
		"device_access_actions_candidate_id_fkey FOREIGN KEY (candidate_id) REFERENCES public.device_access_candidates(id)",
		"device_access_actions_recovery_of_action_id_fkey FOREIGN KEY (recovery_of_action_id) REFERENCES public.device_access_actions(id)",
		"device_access_action_attempts_action_fkey FOREIGN KEY (action_id) REFERENCES public.device_access_actions(id)",
	} {
		require.Contains(t, sql, fragment)
	}
}

func TestDeviceAccessBaselineConstrainsEvidenceAndOutbox(t *testing.T) {
	sql := readDeviceAccessBaseline(t)
	evidence := deviceAccessTableBlock(t, sql, "device_access_evidence")
	outbox := deviceAccessTableBlock(t, sql, "device_access_outbox")

	require.Contains(t, evidence, "expires_at timestamp with time zone")
	require.Contains(t, evidence, "normalized_value jsonb NOT NULL")
	require.Contains(t, evidence, "value_hash character varying(128) NOT NULL")
	for _, evidenceType := range []string{
		"identity", "asset", "tac", "ecgi", "observed_ip", "gps", "security",
	} {
		require.Contains(t, evidence, "'"+evidenceType+"'")
	}
	require.NotContains(t, evidence, "password")
	require.NotContains(t, evidence, "private_key")
	require.NotContains(t, evidence, "certificate_content")

	require.Contains(t, outbox, "event_key character varying(192) NOT NULL")
	require.Contains(t, outbox, "status character varying(24) DEFAULT 'pending'")
	require.Contains(t, outbox, "next_attempt_at timestamp with time zone DEFAULT now() NOT NULL")
	require.Contains(t, sql,
		"ADD CONSTRAINT device_access_outbox_event_key_key UNIQUE (event_key)")
	require.Contains(t, sql,
		"CREATE INDEX idx_device_access_outbox_dispatch ON public.device_access_outbox USING btree (status, next_attempt_at, created_at)")
}

func TestDeviceAccessBaselineAddsAdmissionClassToEveryTaskPartition(t *testing.T) {
	sql := readDeviceAccessBaseline(t)
	tables := []string{"device_tasks"}
	for i := 0; i < 16; i++ {
		tables = append(tables, fmt.Sprintf("device_tasks_p%02d", i))
	}

	for _, table := range tables {
		block := deviceAccessTableBlock(t, sql, table)
		require.Contains(t, block,
			"admission_class character varying(24) DEFAULT 'normal'::character varying NOT NULL",
			table)
	}

	for _, class := range []string{"normal", "access_probe", "security_action"} {
		require.Contains(t, sql, "'"+class+"'")
	}
	require.Contains(t, sql, "ADD CONSTRAINT device_tasks_admission_class_check")
}

func TestDeviceAccessBaselineHasBoundedWorkerIndexes(t *testing.T) {
	sql := readDeviceAccessBaseline(t)
	for _, index := range []string{
		"idx_device_access_candidates_expiry",
		"idx_device_access_states_recheck",
		"idx_device_access_policy_versions_set_status",
		"idx_device_access_import_batches_carrier_time",
		"idx_device_access_import_batches_status",
		"idx_device_access_import_rows_batch_status",
		"idx_device_access_identity_snapshots_identity_time",
		"idx_device_access_decision_archives_decision",
		"idx_device_access_list_entries_effective",
		"idx_device_access_actions_status",
		"idx_device_access_actions_retry",
		"idx_device_access_action_attempts_action",
		"idx_device_access_outbox_dispatch",
	} {
		require.Contains(t, sql, "CREATE INDEX "+index)
	}
	require.Contains(t, sql, "CREATE UNIQUE INDEX IF NOT EXISTS uq_notification_history_event")
	require.Contains(t, sql, "ADD COLUMN IF NOT EXISTS source_type varchar(32)")
	require.Contains(t, sql, "'not_configured'")
}

func TestDeviceAccessBaselineConstrainsBusinessEnums(t *testing.T) {
	sql := readDeviceAccessBaseline(t)
	expectations := map[string][]string{
		"device_access_policy_versions": {"draft", "published", "retired"},
		"device_access_list_entries":    {"deny", "allow", "revoked", "active", "disabled"},
		"device_access_import_batches": {
			"access_list", "rule_dimension", "append", "replace",
			"strict", "valid_only", "uploaded", "validated", "committing", "committed", "failed", "rolled_back",
		},
		"device_access_import_rows":        {"valid", "invalid", "duplicate", "no_change"},
		"device_access_candidates":         {"pending", "approved", "rejected", "expired"},
		"device_access_states":             {"review_required", "collecting", "accepted", "rejected", "revalidating", "revoked"},
		"device_access_decision_checks":    {"passed", "failed", "missing", "stale", "error", "skipped"},
		"device_access_identity_snapshots": {"resolved", "unresolved", "conflict", "ambiguous"},
		"device_access_actions": {
			"rf_off", "rf_on",
			"contain", "release", "pending_dispatch", "dispatching", "verifying", "retry_wait", "succeeded", "failed", "dead",
		},
		"device_access_action_attempts": {
			"baseline_gpv", "spv", "readback_gpv", "queued", "sent", "succeeded", "failed", "timeout", "cancelled",
		},
		"device_access_outbox": {"pending", "delivering", "delivered", "failed", "dead"},
	}

	for table, values := range expectations {
		block := deviceAccessTableBlock(t, sql, table)
		for _, value := range values {
			require.Containsf(t, block, "'"+value+"'", "%s must constrain %q", table, value)
		}
	}

	versions := deviceAccessTableBlock(t, sql, "device_access_policy_versions")
	require.Contains(t, versions, "default_action character varying(16) DEFAULT 'reject'")
	require.Contains(t, versions, "'reject'")
	require.Contains(t, versions, "'review'")
	require.Contains(t, versions, "failure_mode character varying(16) DEFAULT 'fail_closed'")
	require.Contains(t, versions, "'review_hold'")
	require.Contains(t, versions, "collection_timeout_seconds integer DEFAULT 900 NOT NULL")
	require.Contains(t, versions, "bypass_profiles jsonb DEFAULT '[]'::jsonb NOT NULL")
	require.Contains(t, versions, "content_hash character varying(64) NOT NULL")
	require.Contains(t, versions, "device_access_policy_versions_content_hash_check")
	require.Contains(t, sql, "pg_get_constraintdef(constraint_row.oid) ILIKE '%default_action%'")
	require.Contains(t, sql, "last_failure_code = 'evidence_missing'")
	policySets := deviceAccessTableBlock(t, sql, "device_access_policy_sets")
	require.NotContains(t, policySets, "default_action")
	conditions := deviceAccessTableBlock(t, sql, "device_access_conditions")
	require.Contains(t, conditions, "'ip_range'")
	require.Contains(t, conditions, "'within_bounds'")
	actions := deviceAccessTableBlock(t, sql, "device_access_actions")
	require.Contains(t, actions, "max_attempts integer DEFAULT 3 NOT NULL")
	require.Contains(t, actions, "next_attempt_at timestamp with time zone")
	require.Contains(t, actions, "manual_repair_required boolean DEFAULT false NOT NULL")
	attempts := deviceAccessTableBlock(t, sql, "device_access_action_attempts")
	require.Contains(t, attempts, "device_access_action_attempts_action_phase_key UNIQUE (action_id, attempt_no, phase)")
}

func TestDeviceAccessSeedDefinesDynamicMenuAndBuiltInRoleBaseline(t *testing.T) {
	sql := readDeviceAccessSeed(t)

	menuID := "da000001-0000-0000-0000-000000000001"
	deviceMenuID := "11111111-1111-1111-1111-111111111101"
	require.Regexp(t, regexp.MustCompile(
		`(?s)\('`+menuID+`',\s*'接入控制',\s*'menu',\s*'device:access-control',\s*'`+deviceMenuID+`',\s*6,\s*'/device/access-control',\s*'device/AccessControl'`,
	), sql)

	for _, roleID := range []string{
		"10000000-0000-0000-0000-000000000001",
		"10000000-0000-0000-0000-000000000002",
		"10000000-0000-0000-0000-000000000003",
	} {
		require.Regexp(t, regexp.MustCompile(
			`\('`+roleID+`'::uuid,\s*'`+menuID+`'::uuid\)`,
		), sql, "built-in role must receive the access-control page menu")
	}

	viewerID := "10000000-0000-0000-0000-000000000003"
	for suffix := 2; suffix <= 5; suffix++ {
		buttonID := fmt.Sprintf("da000001-0000-0000-0000-%012d", suffix)
		require.NotRegexp(t, regexp.MustCompile(
			`\('`+viewerID+`'::uuid,\s*'`+buttonID+`'::uuid\)`,
		), sql, "viewer must not receive access-control write buttons")
	}
}

func TestDeviceAccessSeedRegistersEveryManagementRoute(t *testing.T) {
	sql := readDeviceAccessSeed(t)
	for _, route := range []string{
		"'/api/v1/device-access/policies/:versionID', 'GET'",
		"'/api/v1/device-access/access-list/batch-disable', 'POST'",
		"'/api/v1/device-access/access-list/template', 'GET'",
		"'/api/v1/device-access/imports/preview', 'POST'",
		"'/api/v1/device-access/imports/:batchID/commit', 'POST'",
		"'/api/v1/device-access/imports/:batchID/rollback', 'POST'",
		"'/api/v1/device-access/imports/:batchID', 'GET'",
		"'/api/v1/device-access/imports/:batchID/errors', 'GET'",
		"'/api/v1/device-access/imports', 'GET'",
		"'/api/v1/device-access/import-templates/:importType', 'GET'",
		"'/api/v1/device-access/policies/:versionID/rules/:ruleID/dimensions/:dimension/export', 'GET'",
		"'/api/v1/device-access/policies/:versionID/rules/:ruleID/dimensions/:dimension', 'DELETE'",
		"'/api/v1/device-access/policies/:versionID', 'PUT'",
		"'/api/v1/device-access/states/:serialNumber/notifications', 'GET'",
		"'/api/v1/device-access/policies/:versionID/difference', 'GET'",
		"'/api/v1/device-access/policies/:versionID/rollback', 'POST'",
	} {
		require.Contains(t, sql, route)
	}
}

func TestDeviceAccessSeedUsesLeastPrivilegeRoleMatrix(t *testing.T) {
	sql := readDeviceAccessSeed(t)
	operatorStart := strings.Index(sql, "-- Operator: read access plus scoped day-to-day governance")
	require.NotEqual(t, -1, operatorStart)
	viewerStart := strings.Index(sql[operatorStart:], "SELECT '10000000-0000-0000-0000-000000000003'::uuid, id")
	require.NotEqual(t, -1, viewerStart)
	operatorGrant := sql[operatorStart : operatorStart+viewerStart]

	require.Contains(t, sql, "DELETE FROM public.role_api_permissions")
	require.Contains(t, operatorGrant, "method = 'GET'")
	for _, allowed := range []string{
		"da000002-0000-0000-0000-000000000006", // list maintenance
		"da000002-0000-0000-0000-000000000012", // candidate review
		"da000002-0000-0000-0000-000000000019", // batch disable
		"da000002-0000-0000-0000-000000000021", // import preview
		"da000002-0000-0000-0000-000000000022", // scoped import commit
	} {
		require.Contains(t, operatorGrant, allowed)
	}
	for _, adminOnly := range []string{
		"da000002-0000-0000-0000-000000000002", // policy publish
		"da000002-0000-0000-0000-000000000015", // action manual retry
		"da000002-0000-0000-0000-000000000018", // global switch update
		"da000002-0000-0000-0000-000000000023", // import rollback
		"da000002-0000-0000-0000-000000000035", // archive
		"da000002-0000-0000-0000-000000000036", // restore
		"da000002-0000-0000-0000-000000000042", // policy rollback
	} {
		require.NotContains(t, operatorGrant, adminOnly)
	}
	require.Contains(t, sql, "WHERE api_group = 'device-access' AND method = 'GET'")
}

func readDeviceAccessBaseline(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile("../../migrations/000001_init_schema.sql")
	require.NoError(t, err)
	return string(data)
}

func readDeviceAccessSeed(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile("../../migrations/seed/000001_init_seed.sql")
	require.NoError(t, err)
	return string(data)
}

func deviceAccessTableBlock(t *testing.T, sql, table string) string {
	t.Helper()
	pattern := regexp.MustCompile(`(?s)CREATE TABLE public\.` + regexp.QuoteMeta(table) + ` \((.*?)\n\)(?:\nPARTITION BY [^;]+)?;`)
	match := pattern.FindStringSubmatch(sql)
	require.Lenf(t, match, 2, "missing or malformed CREATE TABLE for %s", table)
	return strings.TrimSpace(match[1])
}

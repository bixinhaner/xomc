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
		"device_access_list_entries",
		"device_access_candidates",
		"device_access_evidence",
		"device_access_states",
		"device_access_decisions",
		"device_access_decision_checks",
		"device_access_actions",
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
		"device_access_actions_decision_id_fkey FOREIGN KEY (decision_id) REFERENCES public.device_access_decisions(id)",
		"device_access_actions_recovery_of_action_id_fkey FOREIGN KEY (recovery_of_action_id) REFERENCES public.device_access_actions(id)",
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
		"idx_device_access_list_entries_effective",
		"idx_device_access_actions_status",
		"idx_device_access_outbox_dispatch",
	} {
		require.Contains(t, sql, "CREATE INDEX "+index)
	}
}

func TestDeviceAccessBaselineConstrainsBusinessEnums(t *testing.T) {
	sql := readDeviceAccessBaseline(t)
	expectations := map[string][]string{
		"device_access_policy_versions": {"draft", "published", "retired"},
		"device_access_list_entries":    {"deny", "allow", "revoked", "active", "disabled"},
		"device_access_candidates":      {"pending", "approved", "rejected", "expired"},
		"device_access_states":          {"review_required", "collecting", "accepted", "rejected", "revalidating", "revoked"},
		"device_access_decision_checks": {"passed", "failed", "missing", "stale", "error", "skipped"},
		"device_access_actions": {
			"rf_off", "rf_on",
			"contain", "release", "pending_dispatch", "dispatching", "succeeded", "failed",
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
	require.Contains(t, versions, "content_hash character varying(64) NOT NULL")
	require.Contains(t, versions, "device_access_policy_versions_content_hash_check")
	policySets := deviceAccessTableBlock(t, sql, "device_access_policy_sets")
	require.NotContains(t, policySets, "default_action")
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
	} {
		require.Contains(t, sql, route)
	}
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

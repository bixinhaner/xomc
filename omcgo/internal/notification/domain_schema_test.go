package notification

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDomainSchemaTables(t *testing.T) {
	schema := readNotificationMigrationBaseline(t)
	for _, table := range []string{
		"notification_events",
		"notification_occurrences",
		"notification_rules",
		"notification_rule_versions",
		"notification_rule_recipients",
		"notification_rule_channels",
		"notification_contact_groups",
		"notification_contact_group_members",
		"notification_templates",
		"notification_template_versions",
		"notification_channel_configs",
		"notification_channel_health",
		"notification_schedules",
		"notification_deliveries",
		"notification_delivery_attempts",
		"notification_aggregation_buckets",
	} {
		require.NotEmpty(t, extractNotificationTableBody(t, schema, table))
	}
}

func TestDomainSchemaIdentityAndEncryptionGuards(t *testing.T) {
	schema := readNotificationMigrationBaseline(t)

	events := extractNotificationTableBody(t, schema, "notification_events")
	require.Regexp(t, `(?i)event_id\s+uuid\s+not\s+null`, events)
	require.Regexp(t, `(?i)payload\s+jsonb\s+not\s+null`, events)
	require.Contains(t, schema, "notification_events_event_id_key UNIQUE (event_id)")

	occurrences := extractNotificationTableBody(t, schema, "notification_occurrences")
	require.Regexp(t, `(?i)occurrence_id\s+uuid\s+not\s+null`, occurrences)
	require.Regexp(t, `(?i)last_applied_version\s+bigint\s+default\s+0\s+not\s+null`, occurrences)
	require.Regexp(t, `(?i)schedule_generation\s+bigint\s+default\s+1\s+not\s+null`, occurrences)

	for _, table := range []string{
		"notification_rule_recipients",
		"notification_contact_group_members",
		"notification_deliveries",
	} {
		body := extractNotificationTableBody(t, schema, table)
		require.Regexp(t, `(?i)address_ciphertext\s+bytea`, body, table)
		require.Regexp(t, `(?i)address_key_version\s+integer`, body, table)
		require.Regexp(t, `(?i)recipient_fingerprint\s+bytea`, body, table)
		require.NotRegexp(t, `(?i)(email_address|phone_number)\s+(text|character varying)`, body, table)
	}
}

func TestDomainSchemaIdempotencyAndClaimGuards(t *testing.T) {
	schema := readNotificationMigrationBaseline(t)
	for _, fragment := range []string{
		"notification_occurrences_occurrence_id_key UNIQUE (occurrence_id)",
		"notification_deliveries_dedup_key UNIQUE (event_id, dispatch_kind, sequence_no, channel, recipient_fingerprint)",
		"notification_delivery_attempts_delivery_attempt_key UNIQUE (delivery_id, attempt_no)",
		"notification_schedules_dedup_key UNIQUE (occurrence_id, rule_version_id, channel, recipient_fingerprint, schedule_kind, sequence_no, generation)",
		"CREATE INDEX idx_notification_events_process",
		"CREATE INDEX idx_notification_schedules_claim",
		"CREATE INDEX idx_notification_deliveries_claim",
	} {
		require.Contains(t, schema, fragment)
	}

	for _, table := range []string{"notification_schedules", "notification_deliveries"} {
		body := extractNotificationTableBody(t, schema, table)
		require.Regexp(t, `(?i)locked_by\s+character varying`, body, table)
		require.Regexp(t, `(?i)locked_at\s+timestamp with time zone`, body, table)
		require.Regexp(t, `(?i)lease_expires_at\s+timestamp with time zone`, body, table)
	}
}

func readNotificationMigrationBaseline(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	require.True(t, ok, "resolve test filename")
	path := filepath.Join(filepath.Dir(filename), "..", "..", "migrations", "000001_init_schema.sql")
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	return string(data)
}

func extractNotificationTableBody(t *testing.T, schema, table string) string {
	t.Helper()
	pattern := regexp.MustCompile(`(?is)CREATE TABLE public\.` + regexp.QuoteMeta(table) + `\s*\((.*?)\n\);`)
	match := pattern.FindStringSubmatch(schema)
	require.Len(t, match, 2, "CREATE TABLE public.%s not found", table)
	return strings.TrimSpace(match[1])
}

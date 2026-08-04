package alarm

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAlarmOutboxSchema_MainBaseline(t *testing.T) {
	mainSchema := readAlarmMigrationBaseline(t, "000001_init_schema.sql")
	activeBody := extractCreateTableBody(t, mainSchema, "alarms_active")
	outboxBody := extractCreateTableBody(t, mainSchema, "alarm_event_outbox")

	require.Regexp(t, `(?i)alarm_version\s+bigint\s+default\s+1\s+not\s+null`, activeBody)
	require.Regexp(t, `(?i)event_id\s+uuid\s+not\s+null`, outboxBody)
	require.Regexp(t, `(?i)aggregate_version\s+bigint\s+not\s+null`, outboxBody)
	require.Regexp(t, `(?i)payload\s+jsonb\s+not\s+null`, outboxBody)
	require.Regexp(t, `(?i)status\s+character varying\(16\)\s+default\s+'pending'::character varying\s+not\s+null`, outboxBody)
	for _, status := range []OutboxStatus{
		OutboxStatusPending,
		OutboxStatusPublishing,
		OutboxStatusPublished,
		OutboxStatusFailed,
		OutboxStatusDead,
	} {
		require.Contains(t, outboxBody, "'"+string(status)+"'::character varying")
	}
	require.Contains(t, outboxBody, "alarm_event_outbox_aggregate_version_check")
	require.Contains(t, outboxBody, "alarm_event_outbox_attempt_count_check")

	require.Contains(t, mainSchema, "ADD CONSTRAINT alarm_event_outbox_event_id_key UNIQUE (event_id);")
	require.Contains(t, mainSchema, "ADD CONSTRAINT alarm_event_outbox_aggregate_version_key UNIQUE (aggregate_id, aggregate_version);")
	require.Contains(t, mainSchema, "CREATE INDEX idx_alarm_event_outbox_dispatch")
	require.Contains(t, mainSchema, "CREATE INDEX idx_alarm_event_outbox_locked_at")
	require.Contains(t, mainSchema, "CREATE INDEX idx_alarm_event_outbox_published_at")
}

func TestAlarmOutboxSchema_TSDBBaseline(t *testing.T) {
	tsdbSchema := readAlarmMigrationBaseline(t, "tsdb", "000001_tsdb_schema.sql")
	historyBody := extractCreateTableBody(t, tsdbSchema, "alarms_history")
	require.Regexp(t, `(?i)alarm_version\s+bigint\s+default\s+1\s+not\s+null`, historyBody)
}

func readAlarmMigrationBaseline(t *testing.T, parts ...string) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	require.True(t, ok, "resolve test filename")
	pathParts := append([]string{filepath.Dir(filename), "..", "..", "migrations"}, parts...)
	data, err := os.ReadFile(filepath.Join(pathParts...))
	require.NoError(t, err)
	return string(data)
}

func extractCreateTableBody(t *testing.T, schema, table string) string {
	t.Helper()
	pattern := regexp.MustCompile(`(?is)CREATE TABLE public\.` + regexp.QuoteMeta(table) + `\s*\((.*?)\n\);`)
	match := pattern.FindStringSubmatch(schema)
	require.Len(t, match, 2, "CREATE TABLE public.%s not found", table)
	return strings.TrimSpace(match[1])
}

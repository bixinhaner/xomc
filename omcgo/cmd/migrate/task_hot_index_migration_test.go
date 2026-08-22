package main

import (
	"os"
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTaskHotPathIndexesMigration(t *testing.T) {
	baseline, err := os.ReadFile("../../migrations/000001_init_schema.sql")
	require.NoError(t, err)

	sql := string(baseline)

	openByDevice := regexp.MustCompile(`(?is)CREATE\s+INDEX\s+IF\s+NOT\s+EXISTS\s+idx_device_tasks_open_by_device\s+ON\s+public\.device_tasks\s*\(\s*device_sn\s*,\s*status\s*,\s*expires_at\s*\)\s+WHERE\s+status\s+IN\s*\(\s*'pending'\s*,\s*'sent'\s*\)`)
	require.Regexp(t, openByDevice, sql,
		"device reconnect path requires a partial index for CountOpenByDevice")

	sentByDevice := regexp.MustCompile(`(?is)CREATE\s+INDEX\s+IF\s+NOT\s+EXISTS\s+idx_device_tasks_sent_by_device_sent_at\s+ON\s+public\.device_tasks\s*\(\s*device_sn\s*,\s*sent_at\s*\)\s+WHERE\s+status\s*=\s*'sent'\s+AND\s+sent_at\s+IS\s+NOT\s+NULL`)
	require.Regexp(t, sentByDevice, sql,
		"sent task recovery requires a per-device sent_at index")

	expiredPending := regexp.MustCompile(`(?is)CREATE\s+INDEX\s+IF\s+NOT\s+EXISTS\s+idx_device_tasks_expired_pending\s+ON\s+public\.device_tasks\s*\(\s*expires_at\s*\)\s+WHERE\s+status\s*=\s*'pending'\s+AND\s+expires_at\s+IS\s+NOT\s+NULL`)
	require.Regexp(t, expiredPending, sql,
		"expired pending sweep must be able to use expires_at ordering")

	expiredSent := regexp.MustCompile(`(?is)CREATE\s+INDEX\s+IF\s+NOT\s+EXISTS\s+idx_device_tasks_expired_sent\s+ON\s+public\.device_tasks\s*\(\s*expires_at\s*,\s*sent_at\s*\)\s+WHERE\s+status\s*=\s*'sent'\s+AND\s+expires_at\s+IS\s+NOT\s+NULL`)
	require.Regexp(t, expiredSent, sql,
		"expired sent sweep must be able to use expires_at and sent_at filtering")
}

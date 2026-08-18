package main

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParamSyncBaselineSupportsIssue148TriggerReasons(t *testing.T) {
	contents, err := os.ReadFile("../../migrations/000001_init_schema.sql")
	require.NoError(t, err)
	sql := string(contents)

	require.GreaterOrEqual(t, strings.Count(sql, "parameter_sync_requests_trigger_reason_chk"), 2)
	require.GreaterOrEqual(t, strings.Count(sql, "parameter_sync_runs_trigger_reason_chk"), 2)
	require.GreaterOrEqual(t, strings.Count(sql, "'device_registered'"), 2)
	require.GreaterOrEqual(t, strings.Count(sql, "'omc_upgrade'"), 2)
	require.Contains(t, sql, "ALTER TABLE public.parameter_sync_requests")
	require.Contains(t, sql, "ADD COLUMN IF NOT EXISTS admission_queued_at")
	require.Contains(t, sql, "idx_parameter_sync_requests_auto_backpressure")
}

package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSysConfigApplySchemaIsFoldedIntoPreReleaseBaseline(t *testing.T) {
	baseline, err := os.ReadFile("../../migrations/000001_init_schema.sql")
	require.NoError(t, err)
	sql := string(baseline)

	require.Contains(t, sql, "CREATE TABLE public.config_apply_versions")
	require.Contains(t, sql, "CREATE TABLE public.config_apply_batches")
	require.Contains(t, sql, "CREATE TABLE public.config_apply_targets")
	require.Contains(t, sql, "lease_expires_at timestamp with time zone")
	require.Contains(t, sql, "CREATE UNIQUE INDEX uq_config_apply_target_running")
	require.Contains(t, sql, "CREATE INDEX idx_config_apply_targets_recovering")
	require.Contains(t, sql, "lease_expires_at")
}

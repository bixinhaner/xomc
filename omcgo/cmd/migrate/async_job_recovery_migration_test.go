package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAsyncJobRecoveryIsPartOfConsolidatedBaseline(t *testing.T) {
	contents, err := os.ReadFile("../../migrations/000001_init_schema.sql")
	require.NoError(t, err)
	sql := string(contents)

	require.Contains(t, sql, "recovery_count integer DEFAULT 0 NOT NULL")
	require.Contains(t, sql, "last_recovered_at timestamp with time zone")
	require.Contains(t, sql, "idx_async_jobs_hourly_failed_recovery")
}

package main

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGeofenceLocationOutboxMigrationContract(t *testing.T) {
	contents, err := os.ReadFile("../../migrations/000001_init_schema.sql")
	require.NoError(t, err)
	sql := string(contents)

	for _, column := range []string{
		"received_at",
		"device_reported_at",
		"gps_lock_status",
		"satellite_count",
		"accuracy_meters",
	} {
		require.Contains(t, sql, column)
	}
	require.Contains(t, sql, "location_source_mode")
	require.Contains(t, sql, "devices_location_source_mode_check")
	require.Equal(t, 5, strings.Count(
		sql,
		"location_source_mode character varying(16)",
	))
	require.Equal(t, 5, strings.Count(sql, "CONSTRAINT devices_location_source_mode_check"))
	require.Contains(t, sql, "'tr069'::character varying, 'external'::character varying")
	require.Contains(t, sql, "CREATE TABLE public.event_outbox")
	require.Contains(t, sql, "UNIQUE (dedupe_key)")
	require.Contains(t, sql, "'pending', 'publishing', 'published', 'failed', 'dead'")
	require.Contains(t, sql, "claim_token uuid")
	require.Contains(t, sql, "claim_expires_at timestamptz")
	require.Contains(t, sql, "idx_event_outbox_claimable")
	require.Contains(t, sql, "WHERE status IN ('pending', 'failed', 'publishing')")
	require.Contains(t, sql, "-- +goose Down")
	require.Contains(t, sql, "DROP SCHEMA IF EXISTS public CASCADE")
}

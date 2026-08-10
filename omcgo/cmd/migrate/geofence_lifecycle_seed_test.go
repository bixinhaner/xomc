package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGeofenceLifecycleSeedContract(t *testing.T) {
	contents, err := os.ReadFile(
		"../../migrations/seed/000001_init_seed.sql",
	)
	require.NoError(t, err)
	sql := string(contents)

	require.Contains(t, sql, "-- +goose Up")
	require.Contains(t, sql, "INSERT INTO public.api_endpoints")
	for _, route := range []string{
		"/api/v1/geofences/:id/versions",
		"/api/v1/geofences/:id/enable-preview",
		"/api/v1/geofences/:id/enable",
		"/api/v1/geofences/:id/disable-preview",
		"/api/v1/geofences/:id/disable",
		"/api/v1/geofences/:id/archive-preview",
		"/api/v1/geofences/:id/archive",
		"/api/v1/geofence-bindings/:id/suspend",
		"/api/v1/geofence-bindings/:id/resume",
		"/api/v1/geofence-bindings/:id",
	} {
		require.Contains(t, sql, route)
	}
	for _, method := range []string{"'POST'", "'DELETE'"} {
		require.Contains(t, sql, method)
	}
	require.Contains(t, sql, "ON CONFLICT DO NOTHING")
	require.Contains(t, sql, "INSERT INTO public.role_api_permissions")
	require.Contains(
		t,
		sql,
		"'10000000-0000-0000-0000-000000000001'::uuid",
	)
	geofenceSection := geofenceSeedBaselineSection(t, sql)
	require.NotContains(
		t,
		geofenceSection,
		"'10000000-0000-0000-0000-000000000002'::uuid",
	)
	require.NotContains(
		t,
		geofenceSection,
		"'10000000-0000-0000-0000-000000000003'::uuid",
	)

	require.Contains(t, sql, "-- +goose Down")
	require.Contains(t, sql, "consolidated seed 无安全的逐行回滚")
}

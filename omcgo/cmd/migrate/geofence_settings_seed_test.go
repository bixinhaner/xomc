package main

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGeofenceSettingsSeedContract(t *testing.T) {
	contents, err := os.ReadFile("../../migrations/seed/000001_init_seed.sql")
	require.NoError(t, err)
	sql := string(contents)

	require.Contains(t, sql, "-- +goose Up")
	require.Contains(t, sql, "INSERT INTO public.sys_configs")
	require.Contains(t, sql, "'geofence'")
	require.Contains(t, sql, "'mode'")
	require.Contains(t, sql, "'off'")
	require.Contains(t, sql, "INSERT INTO public.geofence_carrier_settings")
	for _, carrier := range []string{"cmcc", "ctcc", "cucc"} {
		require.Contains(t, sql, "'"+carrier+"'")
	}
	require.Contains(t, sql, "ON CONFLICT DO NOTHING")

	for _, route := range []string{
		"/api/v1/geofences/settings",
		"/api/v1/geofences/settings/preview",
	} {
		require.Contains(t, sql, route)
	}
	for _, method := range []string{"'GET'", "'POST'", "'PUT'"} {
		require.Contains(t, sql, method)
	}
	require.Contains(t, sql, "INSERT INTO public.role_api_permissions")
	require.Contains(t, sql, "'topology:gis-map:geofence:view'")
	require.Contains(t, sql, "'topology:gis-map:geofence:manage'")
	require.Contains(t, sql, "'10000000-0000-0000-0000-000000000001'::uuid")
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

func geofenceSeedBaselineSection(t *testing.T, sql string) string {
	t.Helper()
	const marker = "-- Consolidated pre-release geofence defaults and administrator permissions."
	index := strings.LastIndex(sql, marker)
	require.NotEqual(t, -1, index)
	section := sql[index:]
	end := strings.Index(section, "-- +omcgo MainReconcileEnd")
	require.NotEqual(t, -1, end)
	return section[:end]
}

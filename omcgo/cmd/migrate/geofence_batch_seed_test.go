package main

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGeofenceBatchSeedContract(t *testing.T) {
	contents, err := os.ReadFile(
		"../../migrations/seed/000001_init_seed.sql",
	)
	require.NoError(t, err)
	sql := string(contents)

	require.Contains(t, sql, "-- +goose Up")
	require.Contains(t, sql, "INSERT INTO public.api_endpoints")
	require.Contains(t, sql, "INSERT INTO public.role_api_permissions")
	require.Contains(t, sql, "ON CONFLICT DO NOTHING")

	endpoints := []struct {
		id     string
		path   string
		method string
	}{
		{
			id:     "6d1f0b32-37ac-4e58-98e8-32a90c648eb1",
			path:   "/api/v1/geofences/:id/binding-preview",
			method: "POST",
		},
		{
			id:     "6d1f0b32-37ac-4e58-98e8-32a90c648eb2",
			path:   "/api/v1/geofences/:id/bindings",
			method: "POST",
		},
		{
			id:     "6d1f0b32-37ac-4e58-98e8-32a90c648eb3",
			path:   "/api/v1/geofence-jobs/:id",
			method: "GET",
		},
		{
			id:     "6d1f0b32-37ac-4e58-98e8-32a90c648eb4",
			path:   "/api/v1/geofence-jobs/:id/items",
			method: "GET",
		},
	}
	for _, endpoint := range endpoints {
		require.Contains(
			t,
			sql,
			"'"+endpoint.path+"',\n        '"+endpoint.method+"'",
		)
		require.Equal(
			t,
			2,
			strings.Count(sql, endpoint.id),
			"endpoint ID must be inserted and granted in the fresh baseline",
		)
	}

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

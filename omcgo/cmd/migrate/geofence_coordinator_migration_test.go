package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGeofenceCoordinatorMigrationContract(t *testing.T) {
	contents, err := os.ReadFile("../../migrations/000001_init_schema.sql")
	require.NoError(t, err)
	sql := string(contents)

	require.Contains(t, sql, "CREATE TABLE IF NOT EXISTS public.geofence_evaluations")
	for _, column := range []string{
		"binding_id",
		"device_id",
		"geofence_id",
		"geofence_version_id",
		"observation_version",
		"latitude",
		"longitude",
		"gps_height",
		"observed_at",
		"received_at",
		"device_reported_at",
		"gps_lock_status",
		"satellite_count",
		"accuracy_meters",
		"source_path",
		"previous_observation_version",
		"movement_distance_meters",
		"elapsed_seconds",
		"implied_speed_mps",
		"rule_type",
		"raw_position",
		"signed_distance_meters",
		"previous_confirmed_state",
		"confirmed_state",
		"candidate_state",
		"candidate_count",
		"candidate_since",
		"state_edge",
		"status",
		"reason_code",
		"failure_stage",
		"error_code",
		"error_summary",
		"evaluated_at",
	} {
		require.Contains(t, sql, column)
	}
	require.Contains(
		t,
		sql,
		"UNIQUE (binding_id, geofence_version_id, observation_version)",
	)
	require.Contains(t, sql, "geofence_evaluations_status_check")
	require.Contains(t, sql, "'completed', 'failed'")
	require.Contains(t, sql, "device_geofence_states_last_evaluation_fk")
	require.Contains(t, sql, "idx_geofence_evaluations_device_time")
	require.Contains(t, sql, "idx_geofence_evaluations_status_time")
	require.Contains(t, sql, "-- +goose Down")
	require.Contains(t, sql, "DROP SCHEMA IF EXISTS public CASCADE")
}

func TestGeofenceControlSchemaKeepsOneMinimalOwnershipTable(t *testing.T) {
	contents, err := os.ReadFile("../../migrations/000001_init_schema.sql")
	require.NoError(t, err)
	sql := string(contents)

	require.Contains(t, sql, "CREATE TABLE IF NOT EXISTS public.geofence_control_actions")
	for _, evidenceColumn := range []string{
		"action_key varchar(255)",
		"parent_action_id uuid",
		"before_state jsonb",
		"requested_state jsonb",
		"verified_state jsonb",
	} {
		require.Contains(t, sql, evidenceColumn)
	}

	for _, speculativeContract := range []string{
		"CREATE TABLE public.geofence_control_steps",
		"last_successful_action_id uuid",
		"disabled_by_action_id uuid",
		"ADD COLUMN idempotency_key varchar(160)",
		"ADD COLUMN admission_class varchar(64)",
		"CREATE TABLE public.geofence_change_requests",
	} {
		require.NotContains(t, sql, speculativeContract)
	}
}

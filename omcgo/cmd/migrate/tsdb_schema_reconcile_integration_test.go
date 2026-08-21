//go:build integration

package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

func TestIntegrationTSDBSchemaReconcileIsIdempotentAndPreservesVersionOneRows(t *testing.T) {
	dsn := os.Getenv("OMCGO_TSDB_DSN")
	if dsn == "" {
		t.Skip("OMCGO_TSDB_DSN not set; skipping TSDB schema reconciliation integration test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)
	defer pool.Close()

	schema := "slot_reconcile_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	ident := pgx.Identifier{schema}.Sanitize()
	_, err = pool.Exec(ctx, "CREATE SCHEMA "+ident)
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), "DROP SCHEMA "+ident+" CASCADE") })

	_, err = pool.Exec(ctx, `CREATE TABLE `+ident+`.pm_files (
		id uuid PRIMARY KEY,
		device_id uuid NOT NULL,
		technology varchar(16) NOT NULL,
		carrier varchar(16) NOT NULL,
		parsed boolean NOT NULL DEFAULT false
	)`)
	require.NoError(t, err)
	fileID := uuid.New()
	_, err = pool.Exec(ctx, `INSERT INTO `+ident+`.pm_files
		(id,device_id,technology,carrier,parsed) VALUES ($1,$2,'lte','blq',true)`, fileID, uuid.New())
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `CREATE TABLE `+ident+`.pm_aggregation_outbox (
		event_id uuid PRIMARY KEY,
		event_window_start timestamptz NOT NULL,
		payload jsonb NOT NULL,
		barrier_eligible boolean NOT NULL DEFAULT false
	)`)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `CREATE TABLE `+ident+`.pm_aggregation_replay_sources (
		event_window_start timestamptz NOT NULL,
		event_id uuid NOT NULL,
		payload jsonb NOT NULL,
		created_at timestamptz NOT NULL DEFAULT now(),
		PRIMARY KEY (event_window_start,event_id)
	)`)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `SELECT create_hypertable(
		$1::regclass, by_range('event_window_start', INTERVAL '1 day'), if_not_exists => true
	)`, schema+".pm_aggregation_replay_sources")
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `ALTER TABLE `+ident+`.pm_aggregation_replay_sources SET (
		timescaledb.compress,
		timescaledb.compress_orderby = 'event_window_start,event_id'
	)`)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `SELECT add_compression_policy(
		$1::regclass, INTERVAL '1 day', if_not_exists => true
	)`, schema+".pm_aggregation_replay_sources")
	require.NoError(t, err)

	deviceID := uuid.New()
	eventID := uuid.New()
	eventAt := time.Now().UTC().Add(-72 * time.Hour).Truncate(time.Hour)
	_, err = pool.Exec(ctx, `INSERT INTO `+ident+`.pm_aggregation_outbox
		(event_id,event_window_start,payload,barrier_eligible)
		VALUES ($1,$2,jsonb_build_object('device_id',$3::text),false)`,
		eventID, eventAt, deviceID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO `+ident+`.pm_aggregation_replay_sources
		(event_window_start,event_id,payload)
		VALUES ($1,$2,jsonb_build_object('device_id',$3::text))`,
		eventAt, eventID, deviceID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `SELECT compress_chunk(c,true)
		FROM show_chunks($1::regclass) c`, schema+".pm_aggregation_replay_sources")
	require.NoError(t, err)

	sqlPath := filepath.Join("..", "..", "..", "deployments", "release", "bundle", "deploy", "tsdb-schema-reconcile.sql")
	reconcileSQL, err := os.ReadFile(sqlPath)
	require.NoError(t, err)
	scopedSQL := strings.ReplaceAll(string(reconcileSQL), "public.", ident+".")
	scopedSQL = strings.ReplaceAll(scopedSQL, "hypertable_schema = 'public'", "hypertable_schema = '"+schema+"'")

	_, err = pool.Exec(ctx, scopedSQL)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, scopedSQL)
	require.NoError(t, err, "reconciliation must be idempotent")

	var measurementStart, measurementEnd *time.Time
	err = pool.QueryRow(ctx, `SELECT measurement_start,measurement_end FROM `+ident+`.pm_files WHERE id=$1`, fileID).
		Scan(&measurementStart, &measurementEnd)
	require.NoError(t, err)
	require.Nil(t, measurementStart)
	require.Nil(t, measurementEnd)

	var tableExists, fileIndexExists, latestIndexExists, statusConstraintExists bool
	err = pool.QueryRow(ctx, `SELECT
		to_regclass($1) IS NOT NULL,
		to_regclass($2) IS NOT NULL,
		to_regclass($3) IS NOT NULL,
		EXISTS (
			SELECT 1 FROM pg_constraint c
			JOIN pg_namespace n ON n.oid=c.connamespace
			WHERE n.nspname=$4 AND c.conname='chk_pm_slot_health_status'
		)`,
		schema+".pm_slot_health",
		schema+".idx_pm_files_measurement_slot",
		schema+".idx_pm_slot_health_latest",
		schema,
	).Scan(&tableExists, &fileIndexExists, &latestIndexExists, &statusConstraintExists)
	require.NoError(t, err)
	require.True(t, tableExists)
	require.True(t, fileIndexExists)
	require.True(t, latestIndexExists)
	require.True(t, statusConstraintExists)

	var outboxDeviceID, replayDeviceID uuid.UUID
	err = pool.QueryRow(ctx, `SELECT device_id FROM `+ident+`.pm_aggregation_outbox WHERE event_id=$1`, eventID).
		Scan(&outboxDeviceID)
	require.NoError(t, err)
	err = pool.QueryRow(ctx, `SELECT device_id FROM `+ident+`.pm_aggregation_replay_sources WHERE event_id=$1`, eventID).
		Scan(&replayDeviceID)
	require.NoError(t, err)
	require.Equal(t, deviceID, outboxDeviceID)
	require.Equal(t, deviceID, replayDeviceID)

	var outboxIndex, replayIndex, outboxNotNull, replayNotNull, compressed, policyScheduled bool
	var segmentBy string
	err = pool.QueryRow(ctx, `SELECT
		to_regclass($1) IS NOT NULL,
		to_regclass($2) IS NOT NULL,
		(SELECT attnotnull FROM pg_attribute WHERE attrelid=$3::regclass AND attname='device_id'),
		(SELECT attnotnull FROM pg_attribute WHERE attrelid=$4::regclass AND attname='device_id'),
		EXISTS (SELECT 1 FROM timescaledb_information.chunks
			WHERE hypertable_schema=$5 AND hypertable_name='pm_aggregation_replay_sources' AND is_compressed),
		COALESCE((SELECT segmentby FROM timescaledb_information.chunk_compression_settings
			WHERE hypertable=$4::regclass LIMIT 1),''),
		COALESCE((SELECT scheduled FROM timescaledb_information.jobs
			WHERE hypertable_schema=$5 AND hypertable_name='pm_aggregation_replay_sources'
			  AND proc_name='policy_compression' LIMIT 1),false)`,
		schema+".idx_pm_aggregation_outbox_device_period_replay",
		schema+".idx_pm_replay_sources_device_period",
		schema+".pm_aggregation_outbox",
		schema+".pm_aggregation_replay_sources",
		schema,
	).Scan(&outboxIndex, &replayIndex, &outboxNotNull, &replayNotNull, &compressed, &segmentBy, &policyScheduled)
	require.NoError(t, err)
	require.True(t, outboxIndex)
	require.True(t, replayIndex)
	require.True(t, outboxNotNull)
	require.True(t, replayNotNull)
	require.True(t, compressed)
	require.Equal(t, "device_id", segmentBy)
	require.True(t, policyScheduled)
}

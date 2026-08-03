package slothealth

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepositoryUpsertSnapshots_PreservesAuthoritativeSnapshotOrdering(t *testing.T) {
	dsn := os.Getenv("OMCGO_TSDB_DSN")
	if dsn == "" {
		t.Skip("OMCGO_TSDB_DSN not set; skipping TimescaleDB integration test")
	}
	ctx := context.Background()
	config, err := pgxpool.ParseConfig(dsn)
	require.NoError(t, err)
	config.MaxConns = 1
	pool, err := pgxpool.NewWithConfig(ctx, config)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	_, err = pool.Exec(ctx, `CREATE TEMP TABLE pm_slot_health (
		slot_start timestamptz NOT NULL, slot_end timestamptz NOT NULL,
		technology varchar(16) NOT NULL, carrier varchar(16) NOT NULL,
		expected_devices bigint NOT NULL, received_devices bigint NOT NULL,
		coverage_ratio double precision NOT NULL,
		expected_snapshot_version text NOT NULL DEFAULT '',
		evaluated_at timestamptz NOT NULL, status varchar(32) NOT NULL,
		PRIMARY KEY (slot_end, technology, carrier)
	)`)
	require.NoError(t, err)

	repository := NewRepository(nil, pool)
	slotEnd := time.Date(2026, 8, 3, 16, 0, 0, 0, time.UTC)
	base := Snapshot{
		SlotStart: slotEnd.Add(-SlotDuration), SlotEnd: slotEnd,
		Technology: "lte", Carrier: "cmcc", ExpectedDevices: 20000,
		ExpectedSnapshotVersion: "v1",
	}
	bootstrap := base
	bootstrap.EvaluatedAt = slotEnd.Add(20 * time.Minute)
	bootstrap.Status = StatusBootstrapIgnored
	complete := base
	complete.ReceivedDevices = 20000
	complete.CoverageRatio = 1
	complete.EvaluatedAt = slotEnd.Add(12 * time.Minute)
	complete.Status = StatusComplete
	olderPartial := base
	olderPartial.ReceivedDevices = 10000
	olderPartial.CoverageRatio = 0.5
	olderPartial.EvaluatedAt = slotEnd.Add(11 * time.Minute)
	olderPartial.Status = StatusPartial
	newerPartial := olderPartial
	newerPartial.ReceivedDevices = 19000
	newerPartial.CoverageRatio = 0.95
	newerPartial.EvaluatedAt = slotEnd.Add(13 * time.Minute)

	stored, err := repository.UpsertSnapshots(ctx, []Snapshot{bootstrap})
	require.NoError(t, err)
	require.Equal(t, []Snapshot{bootstrap}, stored)
	stored, err = repository.UpsertSnapshots(ctx, []Snapshot{complete})
	require.NoError(t, err)
	require.Equal(t, []Snapshot{complete}, stored, "normal observation must replace bootstrap regardless of timestamp")
	stored, err = repository.UpsertSnapshots(ctx, []Snapshot{bootstrap})
	require.NoError(t, err)
	require.Equal(t, []Snapshot{complete}, stored, "bootstrap must never replace a normal observation")
	stored, err = repository.UpsertSnapshots(ctx, []Snapshot{olderPartial})
	require.NoError(t, err)
	require.Equal(t, []Snapshot{complete}, stored, "older normal observation must not roll back newer state")
	stored, err = repository.UpsertSnapshots(ctx, []Snapshot{newerPartial})
	require.NoError(t, err)
	require.Len(t, stored, 1)
	assert.Equal(t, newerPartial, stored[0], "newer observation remains eligible for correction")
}

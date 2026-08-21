//go:build integration

package stream

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/core/event"
)

func TestIntegrationDevicePeriodReplayUsesTargetDeviceWindowIndex(t *testing.T) {
	dsn := os.Getenv("OMCGO_TSDB_DSN")
	if dsn == "" {
		t.Skip("OMCGO_TSDB_DSN not set; skipping TimescaleDB integration test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)
	defer pool.Close()

	tx, err := pool.Begin(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(context.Background()) }()

	targetDeviceID := uuid.New()
	start := time.Date(2026, 8, 21, 8, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour)
	allDeviceIDs := []uuid.UUID{targetDeviceID}
	for range 96 {
		allDeviceIDs = append(allDeviceIDs, uuid.New())
	}
	for _, deviceID := range allDeviceIDs {
		for slot := range 4 {
			windowStart := start.Add(time.Duration(slot) * 15 * time.Minute)
			require.NoError(t, insertReplayFixturePayload(
				ctx, tx, "pm_aggregation_replay_sources", deviceID, windowStart,
			))
			require.NoError(t, insertReplayFixturePayload(
				ctx, tx, "pm_aggregation_outbox", deviceID, windowStart,
			))
		}
	}

	query, args, err := devicePeriodReplaySelect(targetDeviceID, start, end).ToSql()
	require.NoError(t, err)

	rows, err := tx.Query(ctx, query, args...)
	require.NoError(t, err)
	visited := 0
	err = visitReplayPayloadRows(rows, func(payload event.PMAggregationNormalizedPayload) error {
		require.Equal(t, targetDeviceID, payload.DeviceID)
		require.False(t, payload.WindowStart.Before(start))
		require.True(t, payload.WindowStart.Before(end))
		visited++
		return nil
	})
	require.NoError(t, err)
	require.Equal(t, 8, visited, "target device has four 15-minute slots in each durable source")
	t.Logf("device-period replay pressure fixture: sql_reads=1 returned_rows=%d fixture_devices=%d window_slots=4 durable_sources=2", visited, len(allDeviceIDs))

	explainRows, err := tx.Query(ctx, "EXPLAIN (ANALYZE, BUFFERS, FORMAT TEXT) "+query, args...)
	require.NoError(t, err)
	defer explainRows.Close()
	var planLines []string
	for explainRows.Next() {
		var line string
		require.NoError(t, explainRows.Scan(&line))
		planLines = append(planLines, line)
	}
	require.NoError(t, explainRows.Err())
	plan := strings.Join(planLines, "\n")
	t.Logf("device-period replay EXPLAIN:\n%s", plan)
	require.Contains(t, plan, "idx_pm_replay_sources_device_period", "replay_sources scan must use the device/window index")
	require.Contains(t, plan, "idx_pm_aggregation_outbox_device_period_replay", "legacy outbox scan must use the device/window replay index")
	require.Contains(t, plan, "device_id", "plan must keep target-device predicate in the scan")
	require.Contains(t, plan, "event_window_start", "plan must keep window predicate in the scan")
	require.Contains(t, plan, "Buffers:", "EXPLAIN must include shared block evidence")

	require.NoError(t, tx.Rollback(ctx))
	assertNoReplayFixtureRows(t, ctx, pool, targetDeviceID)
}

type replayFixtureTx interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

func insertReplayFixturePayload(
	ctx context.Context,
	tx replayFixtureTx,
	table string,
	deviceID uuid.UUID,
	windowStart time.Time,
) error {
	eventID := uuid.New()
	payload := event.PMAggregationNormalizedPayload{
		SchemaVersion: SchemaVersion,
		EventID:       eventID,
		SourceFileID:  uuid.New(),
		IngestBatchID: uuid.New(),
		DeviceID:      deviceID,
		DeviceOUI:     "INTREPLAY",
		DeviceSN:      "REPLAY-" + deviceID.String(),
		Technology:    "lte",
		WindowStart:   windowStart,
		WindowEnd:     windowStart.Add(15 * time.Minute),
		Measurements: []event.PMAggregationMeasurement{{
			ObjectLDN:    "",
			CounterGroup: "CG",
			Metrics: []event.PMAggregationMetric{{
				MetricPath: "C_REPLAY", MetricType: "counter",
				StatisType: "sum", Value: 1,
			}},
		}},
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	switch table {
	case "pm_aggregation_replay_sources":
		_, err = tx.Exec(ctx, `
INSERT INTO pm_aggregation_replay_sources (
  event_window_start, event_id, device_id, payload, created_at
) VALUES ($1,$2,$3,$4,now())`,
			windowStart, eventID, deviceID, raw)
	case "pm_aggregation_outbox":
		_, err = tx.Exec(ctx, `
INSERT INTO pm_aggregation_outbox (
  event_id, source_file_id, ingest_batch_id, payload, created_at,
  event_window_start, event_window_end, device_id, barrier_eligible
) VALUES ($1,$2,$3,$4,now(),$5,$6,$7,false)`,
			eventID, payload.SourceFileID, payload.IngestBatchID, raw,
			windowStart, windowStart.Add(15*time.Minute), deviceID)
	default:
		err = ErrInvalidEvent
	}
	return err
}

func assertNoReplayFixtureRows(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	deviceID uuid.UUID,
) {
	t.Helper()
	for _, table := range []string{"pm_aggregation_replay_sources", "pm_aggregation_outbox"} {
		var count int
		err := pool.QueryRow(ctx,
			"SELECT count(*) FROM "+table+" WHERE device_id = $1",
			deviceID,
		).Scan(&count)
		require.NoError(t, err)
		require.Zero(t, count, "integration fixture rows leaked after rollback from %s", table)
	}
}

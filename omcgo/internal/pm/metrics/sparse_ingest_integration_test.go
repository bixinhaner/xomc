//go:build integration

package metrics

import (
	"context"
	"encoding/binary"
	"math"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/core/model"
)

func TestIntegrationCopyIngestStoresSparseValuesAndLogicalMissingRows(t *testing.T) {
	dsn := os.Getenv("OMCGO_DB_DSN")
	if dsn == "" {
		t.Skip("OMCGO_DB_DSN not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)
	defer pool.Close()

	deviceID := uuid.New()
	sn := "SPARSE-INGEST-" + deviceID.String()
	_, err = pool.Exec(ctx, `
		INSERT INTO device_dim (id,oui,serial_number,technology,carrier)
		VALUES ($1,'INT002',$2,'lte','cmcc')`, deviceID, sn)
	require.NoError(t, err)
	end := time.Date(2020, 1, 1, 0, 15, 0, 0, time.UTC).
		Add(time.Duration(binary.BigEndian.Uint32(deviceID[:4])%40000) * time.Hour)
	marker := FileMarker{
		ID: uuid.New(), DeviceID: deviceID, DeviceSN: sn,
		Carrier: "cmcc", Technology: "lte", FileName: deviceID.String() + ".xml",
		CollectTime: end, MinioPath: "integration/" + deviceID.String(),
		ContentSHA256: make([]byte, 32),
		RawCompressed: true,
	}
	counters := []model.PMCounter{
		{DeviceID: deviceID, OUI: "INT002", DeviceSN: sn, CounterGroup: "INT", CounterName: "CSPARSE1", CounterValue: 7, StatisType: "sum", Granularity: 15, Time: end},
		{DeviceID: deviceID, OUI: "INT002", DeviceSN: sn, CounterGroup: "INT", CounterName: "CSPARSE2", CounterValue: math.NaN(), StatisType: "sum", Granularity: 15, Time: end},
	}
	repo := NewPgRepository(pool)
	ingested, err := repo.CopyIngest(ctx, marker, counters, nil)
	require.NoError(t, err)
	require.True(t, ingested)
	var rawCompressed bool
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT raw_compressed FROM pm_files WHERE id=$1`, marker.ID,
	).Scan(&rawCompressed))
	require.True(t, rawCompressed)

	type dictionaryVersion struct {
		path      string
		xmin      string
		updatedAt time.Time
	}
	dictionaryRows, err := pool.Query(ctx, `
		SELECT metric_path, xmin::text, updated_at
		FROM pm_metric_dictionary
		WHERE metric_path = ANY($1::text[])
		ORDER BY metric_path`, []string{"CSPARSE1", "CSPARSE2"})
	require.NoError(t, err)
	var dictionaryBefore []dictionaryVersion
	for dictionaryRows.Next() {
		var version dictionaryVersion
		require.NoError(t, dictionaryRows.Scan(&version.path, &version.xmin, &version.updatedAt))
		dictionaryBefore = append(dictionaryBefore, version)
	}
	require.NoError(t, dictionaryRows.Err())
	dictionaryRows.Close()
	require.Len(t, dictionaryBefore, 2)

	var metricSetID int64
	var metricSetXMinBefore string
	err = pool.QueryRow(ctx, `
		SELECT s.metric_set_id, s.xmin::text
		FROM pm_metric_sets s
		JOIN pm_measurement_anchors a ON a.metric_set_id = s.metric_set_id
		WHERE a.source_file_id = $1`, marker.ID).Scan(&metricSetID, &metricSetXMinBefore)
	require.NoError(t, err)

	var anchors, physical, logical, missing int
	err = pool.QueryRow(ctx, `
		SELECT (SELECT count(*) FROM pm_measurement_anchors WHERE source_file_id=$1),
		       (SELECT count(*) FROM pm_metric_values v JOIN pm_measurement_anchors a
		          ON a."time"=v."time" AND a.anchor_id=v.anchor_id WHERE a.source_file_id=$1),
		       (SELECT count(*) FROM pm_metrics WHERE device_sn=$2),
		       (SELECT count(*) FROM pm_metrics WHERE device_sn=$2 AND metric_value IS NULL)`,
		marker.ID, sn).Scan(&anchors, &physical, &logical, &missing)
	require.NoError(t, err)
	require.Equal(t, 1, anchors)
	require.Equal(t, 1, physical)
	require.Equal(t, 2, logical)
	require.Equal(t, 1, missing)
	targeted, err := repo.Query(ctx, QueryRequest{
		DeviceSNs: []string{sn}, MetricPaths: []string{"CSPARSE1", "CSPARSE2"},
		StartTime: end.Add(-15 * time.Minute), EndTime: end, Limit: 10,
	})
	require.NoError(t, err)
	require.Len(t, targeted, 2)
	total, err := repo.Count(ctx, QueryRequest{
		DeviceSNs: []string{sn}, MetricPaths: []string{"CSPARSE1", "CSPARSE2"},
		StartTime: end.Add(-15 * time.Minute), EndTime: end,
	})
	require.NoError(t, err)
	require.Equal(t, int64(2), total)

	ingested, err = repo.CopyIngest(ctx, marker, counters, nil)
	require.NoError(t, err)
	require.False(t, ingested)

	renamed := marker
	renamed.ID = uuid.New()
	renamed.FileName = "renamed-" + marker.FileName
	ingested, err = repo.CopyIngest(ctx, renamed, counters, nil)
	require.NoError(t, err)
	require.False(t, ingested, "same device/content with another name must be deduplicated")

	changed := marker
	changed.ContentSHA256 = append([]byte(nil), marker.ContentSHA256...)
	changed.ContentSHA256[0] = 1
	changedCounters := append([]model.PMCounter(nil), counters...)
	changedCounters[0].CounterValue = 9
	ingested, err = repo.CopyIngest(ctx, changed, changedCounters, nil)
	require.NoError(t, err)
	require.True(t, ingested, "same source identity with changed content must replace atomically")

	var value float64
	err = pool.QueryRow(ctx, `
		SELECT metric_value FROM pm_metrics
		 WHERE device_sn=$1 AND metric_path='CSPARSE1'`, sn).Scan(&value)
	require.NoError(t, err)
	require.Equal(t, float64(9), value)

	hourStart := end.Add(-15 * time.Minute).Truncate(time.Hour)
	_, err = pool.Exec(ctx, `
		INSERT INTO pm_hourly_bucket_versions (bucket_start,bucket_end,status,dirty)
		VALUES ($1,$2,'active',false)`, hourStart, hourStart.Add(time.Hour))
	require.NoError(t, err)
	empty := changed
	empty.ContentSHA256 = append([]byte(nil), changed.ContentSHA256...)
	empty.ContentSHA256[1] = 2
	ingested, err = repo.CopyIngest(ctx, empty, nil, nil)
	require.NoError(t, err)
	require.True(t, ingested)
	var dirty bool
	err = pool.QueryRow(ctx, `
		SELECT dirty FROM pm_hourly_bucket_versions
		 WHERE bucket_start=$1 AND status='active'`, hourStart).Scan(&dirty)
	require.NoError(t, err)
	require.True(t, dirty, "removing old measurements must dirty their previously published hour")

	// Resolve the same metadata through the hot write path again. This is kept
	// after the compatibility-view assertions because it intentionally creates a
	// second anchor; the point here is to isolate metadata immutability from
	// file-marker idempotency.
	tx, err := pool.Begin(ctx)
	require.NoError(t, err)
	require.NoError(t, writeSparseMeasurements(ctx, tx, nil, nil, BuildSparseMeasurements(counters, nil)))
	require.NoError(t, tx.Commit(ctx))

	dictionaryRows, err = pool.Query(ctx, `
		SELECT metric_path, xmin::text, updated_at
		FROM pm_metric_dictionary
		WHERE metric_path = ANY($1::text[])
		ORDER BY metric_path`, []string{"CSPARSE1", "CSPARSE2"})
	require.NoError(t, err)
	var dictionaryAfter []dictionaryVersion
	for dictionaryRows.Next() {
		var version dictionaryVersion
		require.NoError(t, dictionaryRows.Scan(&version.path, &version.xmin, &version.updatedAt))
		dictionaryAfter = append(dictionaryAfter, version)
	}
	require.NoError(t, dictionaryRows.Err())
	dictionaryRows.Close()
	require.Equal(t, dictionaryBefore, dictionaryAfter, "repeat metadata resolution must not update dictionary rows")

	var metricSetXMinAfter string
	err = pool.QueryRow(ctx, `SELECT xmin::text FROM pm_metric_sets WHERE metric_set_id = $1`, metricSetID).Scan(&metricSetXMinAfter)
	require.NoError(t, err)
	require.Equal(t, metricSetXMinBefore, metricSetXMinAfter, "repeat metadata resolution must not update metric sets")
	var dictionaryCount, metricSetCount int64
	err = pool.QueryRow(ctx, `
		SELECT
		  (SELECT count(*) FROM pm_metric_dictionary WHERE metric_path = ANY($1::text[])),
		  (SELECT count(*) FROM pm_metric_sets
		     WHERE (product_key, counter_group, content_hash) =
		       (SELECT product_key, counter_group, content_hash
		          FROM pm_metric_sets WHERE metric_set_id = $2))`,
		[]string{"CSPARSE1", "CSPARSE2"}, metricSetID).Scan(&dictionaryCount, &metricSetCount)
	require.NoError(t, err)
	require.Equal(t, int64(2), dictionaryCount)
	require.Equal(t, int64(1), metricSetCount)
}

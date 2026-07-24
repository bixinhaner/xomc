package aggregator

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/asyncjob"
)

const DefaultLateDataWindow = 7 * 24 * time.Hour

func ParseLateDataWindow(raw string) time.Duration {
	d, err := time.ParseDuration(raw)
	if err != nil || d <= 0 {
		return DefaultLateDataWindow
	}
	return d
}

// RunSparseMaintenance keeps dirty buckets queued and performs only
// watermark-safe compression/retention. It exits when ctx is cancelled.
func RunSparseMaintenance(
	ctx context.Context,
	tsdb *pgxpool.Pool,
	enqueuer JobEnqueuer,
	lateWindow time.Duration,
	m *Metrics,
	logger *zap.Logger,
) {
	if tsdb == nil {
		return
	}
	if lateWindow <= 0 {
		lateWindow = DefaultLateDataWindow
	}
	if logger == nil {
		logger = zap.NewNop()
	}
	sampleTicker := time.NewTicker(time.Minute)
	maintenanceTicker := time.NewTicker(time.Hour)
	defer sampleTicker.Stop()
	defer maintenanceTicker.Stop()

	sample := func() {
		if err := sampleSparseState(ctx, tsdb, m); err != nil {
			logger.Warn("sample sparse PM state failed", zap.Error(err))
		}
		if enqueuer != nil {
			if err := enqueueDirtyHourlyBuckets(ctx, tsdb, enqueuer, lateWindow); err != nil {
				logger.Warn("enqueue dirty PM buckets failed", zap.Error(err))
			}
		}
	}
	maintain := func() {
		if err := cleanupObsoleteHourlyVersions(ctx, tsdb); err != nil {
			logger.Warn("cleanup obsolete hourly versions failed", zap.Error(err))
		}
		for _, table := range []string{"pm_measurement_anchors", "pm_metric_values"} {
			if err := compressEligibleChunks(ctx, tsdb, table, lateWindow); err != nil {
				logger.Warn("compress sparse PM chunks failed", zap.String("table", table), zap.Error(err))
			}
			if err := dropEligibleChunks(ctx, tsdb, table, 30*24*time.Hour); err != nil {
				logger.Warn("drop retained sparse PM chunks failed", zap.String("table", table), zap.Error(err))
			}
		}
		for _, table := range []string{"pm_hourly_anchors", "pm_hourly_values"} {
			if err := compressEligibleChunks(ctx, tsdb, table, lateWindow); err != nil {
				logger.Warn("compress hourly sparse PM chunks failed", zap.String("table", table), zap.Error(err))
			}
			if err := dropEligibleChunks(ctx, tsdb, table, 180*24*time.Hour); err != nil {
				logger.Warn("drop retained hourly sparse PM chunks failed", zap.String("table", table), zap.Error(err))
			}
		}
	}
	sample()
	for {
		select {
		case <-ctx.Done():
			return
		case <-sampleTicker.C:
			sample()
		case <-maintenanceTicker.C:
			maintain()
		}
	}
}

func buildDirtyBucketSQL() string {
	return `SELECT bucket_start, bucket_end
FROM pm_hourly_bucket_versions
WHERE status='active' AND dirty=true
ORDER BY bucket_start
LIMIT 100`
}

func enqueueDirtyHourlyBuckets(ctx context.Context, db *pgxpool.Pool, enq JobEnqueuer, late time.Duration) error {
	_ = late
	rows, err := db.Query(ctx, buildDirtyBucketSQL())
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var start, end time.Time
		if err := rows.Scan(&start, &end); err != nil {
			return err
		}
		payload, err := BuildPayload(start, end)
		if err != nil {
			return err
		}
		var enqueueErr error
		if requeue, ok := enq.(interface {
			RequeueSucceededBucket(context.Context, string, time.Time, time.Time, json.RawMessage) (uuid.UUID, error)
		}); ok {
			_, enqueueErr = requeue.RequeueSucceededBucket(ctx, JobTypeHourly, start, end, payload)
		} else {
			bucketStart, bucketEnd := start, end
			_, enqueueErr = enq.Insert(ctx, asyncjob.InsertRequest{
				JobType: JobTypeHourly, Payload: payload,
				BucketStart: &bucketStart, BucketEnd: &bucketEnd,
			})
		}
		if enqueueErr != nil {
			return enqueueErr
		}
	}
	return rows.Err()
}

func sampleSparseState(ctx context.Context, db *pgxpool.Pool, m *Metrics) error {
	if m == nil {
		return nil
	}
	var dirty float64
	if err := db.QueryRow(ctx,
		`SELECT count(*)::float8 FROM pm_hourly_bucket_versions WHERE dirty=true AND status IN ('active','building')`,
	).Scan(&dirty); err != nil {
		return err
	}
	m.SetDirtyBuckets(dirty)

	var watermark *time.Time
	var amplification *float64
	var tempBytes float64
	if err := db.QueryRow(ctx, `
		SELECT max(bucket_end) FILTER (WHERE status='active' AND dirty=false),
		       (SELECT logical_rows::float8 / NULLIF(physical_rows,0)
		          FROM (SELECT sum(cardinality(s.metric_ids)) AS logical_rows
		                  FROM pm_measurement_anchors a
		                  JOIN pm_metric_sets s ON s.metric_set_id=a.metric_set_id
		                 WHERE a."time" >= now()-interval '1 hour') logical,
		               (SELECT count(*) AS physical_rows
		                  FROM pm_metric_values
		                 WHERE "time" >= now()-interval '1 hour') physical),
		       (SELECT temp_bytes::float8 FROM pg_stat_database WHERE datname=current_database())
		  FROM pm_hourly_bucket_versions`,
	).Scan(&watermark, &amplification, &tempBytes); err != nil {
		return err
	}
	if watermark != nil {
		m.SetWatermark(watermark.Unix())
	}
	if amplification != nil {
		m.SetSparseAmplification(*amplification)
	}
	m.SetTempBytes(tempBytes)
	return nil
}

func buildEligibleChunkSQL(onlyUncompressed bool) string {
	compressionFilter := ""
	if onlyUncompressed {
		compressionFilter = "\n   AND NOT c.is_compressed"
	}
	return `
SELECT format('%I.%I', c.chunk_schema, c.chunk_name), c.range_start, c.range_end
  FROM timescaledb_information.chunks c
 WHERE c.hypertable_schema='public' AND c.hypertable_name=$1
` + compressionFilter + `
   AND c.range_end < now() - $2::interval
   AND NOT EXISTS (
       SELECT 1
         FROM generate_series(date_trunc('hour', c.range_start),
                              c.range_end - interval '1 hour',
                              interval '1 hour') hour_start
         LEFT JOIN pm_hourly_bucket_versions v
           ON v.bucket_start=hour_start AND v.status='active' AND v.dirty=false
        WHERE v.bucket_version IS NULL
   )
 ORDER BY c.range_end
 LIMIT 4`
}

func eligibleChunks(
	ctx context.Context,
	db *pgxpool.Pool,
	table string,
	age time.Duration,
	onlyUncompressed bool,
) ([]chunkRange, error) {
	rows, err := db.Query(ctx, buildEligibleChunkSQL(onlyUncompressed), table, age.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []chunkRange
	for rows.Next() {
		var chunk chunkRange
		if err := rows.Scan(&chunk.name, &chunk.start, &chunk.end); err != nil {
			return nil, err
		}
		out = append(out, chunk)
	}
	return out, rows.Err()
}

type chunkRange struct {
	name       string
	start, end time.Time
}

func compressEligibleChunks(ctx context.Context, db *pgxpool.Pool, table string, late time.Duration) error {
	chunks, err := eligibleChunks(ctx, db, table, late, true)
	if err != nil {
		return err
	}
	for _, chunk := range chunks {
		if _, err := db.Exec(ctx, `SELECT compress_chunk($1::regclass, if_not_compressed => true)`, chunk.name); err != nil {
			return fmt.Errorf("compress %s: %w", chunk.name, err)
		}
	}
	return nil
}

func dropEligibleChunks(ctx context.Context, db *pgxpool.Pool, table string, retention time.Duration) error {
	chunks, err := eligibleChunks(ctx, db, table, retention, false)
	if err != nil {
		return err
	}
	for _, chunk := range chunks {
		if _, err := db.Exec(ctx,
			`SELECT drop_chunks($1::regclass, older_than => $2::timestamptz, newer_than => $3::timestamptz)`,
			table, chunk.end, chunk.start); err != nil {
			return fmt.Errorf("drop %s: %w", chunk.name, err)
		}
	}
	return nil
}

func cleanupObsoleteHourlyVersions(ctx context.Context, db *pgxpool.Pool) error {
	_, err := db.Exec(ctx, `
WITH doomed AS (
    SELECT bucket_version FROM pm_hourly_bucket_versions
     WHERE (status IN ('failed','superseded') AND created_at < now()-interval '24 hours')
        OR (status='building' AND created_at < now()-interval '1 hour')
),
deleted_values AS (
    DELETE FROM pm_hourly_values v
     USING pm_hourly_anchors a, doomed d
     WHERE a.bucket_version=d.bucket_version
       AND v."time"=a."time" AND v.anchor_id=a.anchor_id
),
deleted_anchors AS (
    DELETE FROM pm_hourly_anchors a USING doomed d
     WHERE a.bucket_version=d.bucket_version
),
deleted_batches AS (
    DELETE FROM pm_hourly_rollup_batches b USING doomed d
     WHERE b.bucket_version=d.bucket_version
)
DELETE FROM pm_hourly_bucket_versions v USING doomed d
 WHERE v.bucket_version=d.bucket_version`)
	return err
}

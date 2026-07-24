package aggregator

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/asyncjob"
)

const (
	DefaultLateDataWindow          = 7 * 24 * time.Hour
	DefaultHourlyRecoveryCooldown  = time.Hour
	DefaultHourlyRecoveryMax       = 3
	DefaultHourlyRecoveryScanLimit = 200
	MaxHourlyRecoveryScanHorizon   = 7 * 24 * time.Hour
	DefaultStaleBuildingTimeout    = time.Hour
)

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
		cleanupOK := true
		recoveryRepo, supportsRecovery := enqueuer.(HourlyRecoveryRepository)
		if supportsRecovery {
			if err := failStaleBuildingVersions(
				ctx, tsdb, recoveryRepo, DefaultStaleBuildingTimeout, logger,
			); err != nil {
				cleanupOK = false
				logger.Warn("fail stale building PM versions failed", zap.Error(err))
			}
			if err := recoverFailedHourlyBuckets(
				ctx, tsdb, recoveryRepo, lateWindow, m, logger,
			); err != nil {
				logger.Warn("recover failed hourly PM buckets failed", zap.Error(err))
			}
		}
		if err := sampleSparseState(ctx, tsdb, m); err != nil {
			logger.Warn("sample sparse PM state failed", zap.Error(err))
		}
		if enqueuer != nil && cleanupOK {
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

func enqueueDirtyHourlyBuckets(ctx context.Context, db DBTX, enq JobEnqueuer, late time.Duration) error {
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

func buildStaleBuildingVersionsSQL() string {
	return `
SELECT bucket_version, bucket_start, bucket_end
  FROM pm_hourly_bucket_versions
 WHERE status='building'
   AND created_at < now() - $1::interval
 ORDER BY created_at
 LIMIT 100`
}

func buildFailStaleBuildingVersionSQL() string {
	return `
UPDATE pm_hourly_bucket_versions
   SET status='failed'
 WHERE bucket_version=$1
   AND status='building'`
}

type staleBuildingVersion struct {
	version    int64
	start, end time.Time
}

func failStaleBuildingVersions(
	ctx context.Context,
	db DBTX,
	repo HourlyRecoveryRepository,
	timeout time.Duration,
	logger *zap.Logger,
) error {
	rows, err := db.Query(ctx, buildStaleBuildingVersionsSQL(), timeout.String())
	if err != nil {
		return fmt.Errorf("list stale building hourly versions: %w", err)
	}
	defer rows.Close()
	var versions []staleBuildingVersion
	for rows.Next() {
		var version staleBuildingVersion
		if err := rows.Scan(&version.version, &version.start, &version.end); err != nil {
			return fmt.Errorf("scan stale building hourly version: %w", err)
		}
		versions = append(versions, version)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate stale building hourly versions: %w", err)
	}

	for _, version := range versions {
		job, found, err := repo.FindNaturalBucketJob(
			ctx, JobTypeHourly, version.start, version.end,
		)
		if err != nil {
			return fmt.Errorf("find job for stale hourly version %d: %w", version.version, err)
		}
		if found && !terminalAsyncJobStatus(job.Status) {
			continue
		}
		tag, err := db.Exec(ctx, buildFailStaleBuildingVersionSQL(), version.version)
		if err != nil {
			return fmt.Errorf("fail stale building hourly version %d: %w", version.version, err)
		}
		if tag.RowsAffected() > 0 {
			logger.Warn("stale hourly building version marked failed",
				zap.Int64("bucket_version", version.version),
				zap.Time("bucket_start", version.start),
				zap.Time("bucket_end", version.end),
				zap.Bool("job_found", found),
			)
		}
	}
	return nil
}

func terminalAsyncJobStatus(status asyncjob.Status) bool {
	switch status {
	case asyncjob.StatusSucceeded, asyncjob.StatusFailed, asyncjob.StatusZombie, asyncjob.StatusCanceled:
		return true
	default:
		return false
	}
}

func recoverFailedHourlyBuckets(
	ctx context.Context,
	db DBTX,
	repo HourlyRecoveryRepository,
	horizon time.Duration,
	m *Metrics,
	logger *zap.Logger,
) error {
	if horizon <= 0 {
		horizon = DefaultLateDataWindow
	}
	now := time.Now()
	req := asyncjob.FailedBucketMaintenanceRequest{
		JobType:       JobTypeHourly,
		Since:         now.Add(-horizon),
		Limit:         DefaultHourlyRecoveryScanLimit,
		MaxRecoveries: DefaultHourlyRecoveryMax,
		Cooldown:      DefaultHourlyRecoveryCooldown,
		AgedAfter:     DefaultHourlyRecoveryCooldown,
	}
	m.SetFailedBuckets(0)
	m.SetAgedFailedBuckets(0)
	m.SetRecoveryExhaustedBuckets(0)
	stats, err := repo.GetFailedBucketStats(ctx, req)
	if err != nil {
		return fmt.Errorf("sample failed hourly buckets: %w", err)
	}
	m.SetFailedBuckets(float64(stats.FailedCount))
	m.SetAgedFailedBuckets(float64(stats.AgedCount))
	m.SetRecoveryExhaustedBuckets(float64(stats.ExhaustedCount))

	recoveryReq := req
	recoveryReq.Since = now.Add(-min(horizon, MaxHourlyRecoveryScanHorizon))
	jobs, err := repo.ListRecoverableFailedNaturalBuckets(ctx, recoveryReq)
	if err != nil {
		return fmt.Errorf("discover recoverable failed hourly buckets: %w", err)
	}
	for _, job := range jobs {
		if !retriableFailedBucketMarker(job.ErrorMessage) ||
			job.BucketStart == nil || job.BucketEnd == nil {
			continue
		}
		hasSource, hasActive, err := hourlyBucketRecoveryState(
			ctx, db, *job.BucketStart, *job.BucketEnd,
		)
		if err != nil {
			return fmt.Errorf("inspect failed hourly bucket %s: %w", job.ID, err)
		}
		if !hasSource || hasActive {
			continue
		}
		payload, err := BuildPayload(*job.BucketStart, *job.BucketEnd)
		if err != nil {
			return fmt.Errorf("build failed hourly bucket payload: %w", err)
		}
		recoveredID, recovered, err := repo.RequeueRetriableFailedBucket(
			ctx,
			asyncjob.FailedBucketRecoveryRequest{
				JobType:       JobTypeHourly,
				BucketStart:   *job.BucketStart,
				BucketEnd:     *job.BucketEnd,
				Payload:       payload,
				MaxRecoveries: DefaultHourlyRecoveryMax,
				Cooldown:      DefaultHourlyRecoveryCooldown,
			},
		)
		if err != nil {
			return fmt.Errorf("requeue failed hourly bucket %s: %w", job.ID, err)
		}
		if recovered {
			m.IncHourlyRecovery()
			logger.Info("failed hourly bucket recovered",
				zap.String("job_id", recoveredID.String()),
				zap.Time("bucket_start", *job.BucketStart),
				zap.Time("bucket_end", *job.BucketEnd),
				zap.Int("recovery_count", job.RecoveryCount+1),
			)
		}
	}
	return nil
}

var retriableFailedBucketMarkerRE = regexp.MustCompile(
	`(^|[^[:alnum:]])SQLSTATE (40P01|40001)([^[:alnum:]]|$)`,
)

func retriableFailedBucketMarker(message string) bool {
	return retriableFailedBucketMarkerRE.MatchString(message)
}

func hourlyBucketRecoveryState(
	ctx context.Context,
	db DBTX,
	start, end time.Time,
) (hasSource, hasActive bool, err error) {
	err = db.QueryRow(ctx, `
SELECT EXISTS (
           SELECT 1
             FROM pm_measurement_anchors
            WHERE "time" >= $1 AND "time" < $2
       ),
       EXISTS (
           SELECT 1
             FROM pm_hourly_bucket_versions
            WHERE bucket_start=$1 AND bucket_end=$2 AND status='active'
       )`, start, end).Scan(&hasSource, &hasActive)
	if err != nil {
		return false, false, fmt.Errorf("query hourly bucket recovery state: %w", err)
	}
	return hasSource, hasActive, nil
}

func sampleSparseState(ctx context.Context, db DBTX, m *Metrics) error {
	if m == nil {
		return nil
	}
	var dirty, failedVersions, staleBuilding float64
	if err := db.QueryRow(ctx,
		`SELECT count(*) FILTER (WHERE dirty=true AND status IN ('active','building'))::float8,
		        count(*) FILTER (WHERE status='failed')::float8,
		        count(*) FILTER (
		            WHERE status='building' AND created_at < now()-$1::interval
		        )::float8
		   FROM pm_hourly_bucket_versions`,
		DefaultStaleBuildingTimeout.String(),
	).Scan(&dirty, &failedVersions, &staleBuilding); err != nil {
		return fmt.Errorf("sample hourly version state: %w", err)
	}
	m.SetDirtyBuckets(dirty)
	m.SetFailedVersions(failedVersions)
	m.SetStaleBuildingVersions(staleBuilding)

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
		return fmt.Errorf("sample sparse watermark state: %w", err)
	}
	if watermark != nil {
		m.SetWatermark(watermark.Unix())
		lag := time.Since(*watermark)
		if lag < 0 {
			lag = 0
		}
		m.SetWatermarkLag(lag)
	} else {
		m.SetWatermarkLag(time.Since(time.Unix(0, 0)))
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
     WHERE status IN ('failed','superseded') AND created_at < now()-interval '24 hours'
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

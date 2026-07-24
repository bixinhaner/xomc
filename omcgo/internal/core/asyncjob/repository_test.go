package asyncjob

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestBuildInsertSQL_BucketJobsUseWindowDedupe(t *testing.T) {
	start := time.Date(2026, 7, 7, 16, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)

	sql, args, err := buildInsertSQL(InsertRequest{
		JobType:     "pm_aggregate_daily",
		ScheduledAt: start,
		BucketStart: &start,
		BucketEnd:   &end,
	})
	require.NoError(t, err)
	require.Contains(t, sql, "bucket_start")
	require.Contains(t, sql, "bucket_end")
	require.Contains(t, sql, "ON CONFLICT (job_type, bucket_start, bucket_end)")
	require.Contains(t, sql, "WHERE bucket_start IS NOT NULL AND bucket_end IS NOT NULL")
	require.NotContains(t, sql, "'failed', 'canceled', 'zombie'")
	require.Contains(t, sql, "async_jobs.status IN ('canceled', 'zombie')")
	require.Contains(t, sql, "async_jobs.job_type <> 'pm_aggregate_hourly'")
	require.NotContains(t, sql, "recovery_count")
	require.NotContains(t, sql, "last_recovered_at")
	require.Contains(t, sql, "THEN 'pending'")
	require.Contains(t, sql, "RETURNING id")
	require.Equal(t, "pm_aggregate_daily", args[0])
	require.Equal(t, start, args[3])
	require.Same(t, &start, args[4])
	require.Same(t, &end, args[5])
	require.Equal(t, DefaultMaxAttempts, args[7])
}

func TestBuildInsertSQL_NonBucketJobsLeaveWindowNull(t *testing.T) {
	scheduledAt := time.Date(2026, 7, 7, 16, 0, 0, 0, time.UTC)

	_, args, err := buildInsertSQL(InsertRequest{
		JobType:     "pm_kpi_export",
		ScheduledAt: scheduledAt,
	})
	require.NoError(t, err)
	require.Nil(t, args[4])
	require.Nil(t, args[5])
}

func TestBuildRequeueRetriableFailedBucketSQLIsAtomicAndBounded(t *testing.T) {
	start := time.Date(2026, 7, 24, 10, 0, 0, 0, time.UTC)
	req := FailedBucketRecoveryRequest{
		JobType:       "pm_aggregate_hourly",
		BucketStart:   start,
		BucketEnd:     start.Add(time.Hour),
		Payload:       json.RawMessage(`{"start":"2026-07-24T10:00:00Z"}`),
		MaxRecoveries: 3,
		Cooldown:      time.Hour,
	}

	sql, args, err := buildRequeueRetriableFailedBucketSQL(req)
	require.NoError(t, err)
	require.Contains(t, sql, "UPDATE async_jobs")
	require.Contains(t, sql, "status = 'pending'")
	require.Contains(t, sql, "recovery_count = recovery_count + 1")
	require.Contains(t, sql, "last_recovered_at = NOW()")
	require.Contains(t, sql, "status = 'failed'")
	require.Contains(t, sql, "finished_at IS NOT NULL")
	require.Contains(t, sql, "bucket_start = date_trunc('hour', bucket_start)")
	require.Contains(t, sql, "bucket_end = bucket_start + interval '1 hour'")
	require.Contains(t, sql, "bucket_end <= NOW()")
	require.Contains(t, sql, "error_message LIKE '%SQLSTATE 40P01%'")
	require.Contains(t, sql, "error_message LIKE '%SQLSTATE 40001%'")
	require.Contains(t, sql, "recovery_count <")
	require.Contains(t, sql, "GREATEST")
	require.Contains(t, sql, "RETURNING id")
	require.NotContains(t, sql, "error_message = NULL")
	require.NotContains(t, sql, "recovery_count = 0")
	require.Equal(t, "pm_aggregate_hourly", args[0])
	require.Equal(t, start, args[1])
	require.Equal(t, start.Add(time.Hour), args[2])
	require.Equal(t, 3, args[4])
	require.Equal(t, time.Hour.String(), args[5])
}

func TestBuildRequeueRetriableFailedBucketSQLRejectsNonHourlyWindows(t *testing.T) {
	start := time.Date(2026, 7, 24, 10, 0, 0, 0, time.UTC)
	for _, req := range []FailedBucketRecoveryRequest{
		{JobType: "pm_aggregate_daily", BucketStart: start, BucketEnd: start.Add(time.Hour), MaxRecoveries: 3},
		{JobType: "pm_aggregate_hourly", BucketStart: start, BucketEnd: start.Add(2 * time.Hour), MaxRecoveries: 3},
		{JobType: "pm_aggregate_hourly", BucketStart: start.Add(time.Minute), BucketEnd: start.Add(time.Hour + time.Minute), MaxRecoveries: 3},
		{JobType: "pm_aggregate_hourly", BucketStart: start, BucketEnd: start.Add(time.Hour), MaxRecoveries: 0},
		{JobType: "pm_aggregate_hourly", BucketStart: start, BucketEnd: start.Add(time.Hour), MaxRecoveries: 3, Cooldown: -time.Second},
	} {
		_, _, err := buildRequeueRetriableFailedBucketSQL(req)
		require.Error(t, err)
	}
}

func TestFailedBucketScanIsBoundedToRecentNaturalHourlyBuckets(t *testing.T) {
	sql := buildListFailedNaturalBucketsSQL()
	require.Contains(t, sql, "job_type=$1")
	require.Contains(t, sql, "status='failed'")
	require.Contains(t, sql, "bucket_start >= $2")
	require.Contains(t, sql, "bucket_end=bucket_start+interval '1 hour'")
	require.Contains(t, sql, "LIMIT $3")
}

func TestAsyncJobRecoveryMigrationMatchesFreshInstallSchema(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	migrations := filepath.Join(filepath.Dir(file), "..", "..", "..", "migrations")

	incremental, err := os.ReadFile(filepath.Join(migrations, "000005_async_job_recovery.sql"))
	require.NoError(t, err)
	baseline, err := os.ReadFile(filepath.Join(migrations, "000001_init_schema.sql"))
	require.NoError(t, err)

	for _, sql := range []string{string(incremental), string(baseline)} {
		require.Contains(t, sql, "recovery_count")
		require.Contains(t, sql, "last_recovered_at")
	}
	require.Contains(t, string(incremental), "ADD COLUMN IF NOT EXISTS recovery_count")
	require.Contains(t, string(incremental), "ADD COLUMN IF NOT EXISTS last_recovered_at")
	require.Contains(t, string(incremental), "CREATE INDEX IF NOT EXISTS idx_async_jobs_hourly_failed_recovery")
}

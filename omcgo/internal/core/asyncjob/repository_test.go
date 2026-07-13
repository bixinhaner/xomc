package asyncjob

import (
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
	require.Contains(t, sql, "async_jobs.status IN ('failed', 'canceled', 'zombie')")
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

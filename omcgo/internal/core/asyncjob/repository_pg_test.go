package asyncjob

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

func TestPgRepository_InsertBucketDedupe_WithPostgres(t *testing.T) {
	dsn := os.Getenv("OMCGO_ASYNCJOB_TEST_DSN")
	if dsn == "" {
		t.Skip("set OMCGO_ASYNCJOB_TEST_DSN to run PostgreSQL repository behavior test")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)
	defer pool.Close()

	_, err = pool.Exec(ctx, `
DROP TABLE IF EXISTS async_jobs;
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE TABLE async_jobs (
    id uuid DEFAULT gen_random_uuid() NOT NULL PRIMARY KEY,
    job_type text NOT NULL,
    status text DEFAULT 'pending'::text NOT NULL,
    schedule_expr text,
    scheduled_at timestamp with time zone NOT NULL,
    bucket_start timestamp with time zone,
    bucket_end timestamp with time zone,
    started_at timestamp with time zone,
    finished_at timestamp with time zone,
    heartbeat_at timestamp with time zone,
    lock_owner text,
    attempt integer DEFAULT 1 NOT NULL,
    max_attempts integer DEFAULT 3 NOT NULL,
    payload jsonb,
    result jsonb,
    error_message text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);
CREATE UNIQUE INDEX uq_async_jobs_bucket
    ON async_jobs (job_type, bucket_start, bucket_end)
    WHERE bucket_start IS NOT NULL AND bucket_end IS NOT NULL;`)
	require.NoError(t, err)

	repo := NewPgRepository(pool)
	start := time.Date(2026, 7, 7, 16, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)
	req := InsertRequest{
		JobType:     "pm_aggregate_daily",
		ScheduledAt: start,
		BucketStart: &start,
		BucketEnd:   &end,
		Payload:     json.RawMessage(`{"start":"a"}`),
	}

	firstID, err := repo.Insert(ctx, req)
	require.NoError(t, err)
	secondID, err := repo.Insert(ctx, req)
	require.NoError(t, err)
	require.Equal(t, firstID, secondID, "pending duplicate should not create another row")

	_, err = pool.Exec(ctx, `UPDATE async_jobs SET status='running', started_at=NOW(), heartbeat_at=NOW(), lock_owner='test' WHERE id=$1`, firstID)
	require.NoError(t, err)
	runningID, err := repo.Insert(ctx, req)
	require.NoError(t, err)
	require.Equal(t, firstID, runningID, "running duplicate should not create another row")
	job, err := repo.GetByID(ctx, firstID)
	require.NoError(t, err)
	require.Equal(t, StatusRunning, job.Status)

	_, err = pool.Exec(ctx, `UPDATE async_jobs SET status='succeeded', finished_at=NOW(), result='{}'::jsonb WHERE id=$1`, firstID)
	require.NoError(t, err)
	succeededID, err := repo.Insert(ctx, req)
	require.NoError(t, err)
	require.Equal(t, firstID, succeededID, "succeeded duplicate should not auto-rerun")
	job, err = repo.GetByID(ctx, firstID)
	require.NoError(t, err)
	require.Equal(t, StatusSucceeded, job.Status)

	for _, status := range []Status{StatusFailed, StatusCanceled, StatusZombie} {
		_, err = pool.Exec(ctx, `
UPDATE async_jobs SET
    status=$2, started_at=NOW(), finished_at=NOW(), heartbeat_at=NOW(), lock_owner='stale',
    attempt=3, result='{}'::jsonb, error_message='old'
WHERE id=$1`, firstID, string(status))
		require.NoError(t, err)

		restoredID, err := repo.Insert(ctx, req)
		require.NoError(t, err)
		require.Equal(t, firstID, restoredID)
		job, err = repo.GetByID(ctx, firstID)
		require.NoError(t, err)
		require.Equal(t, StatusPending, job.Status, "%s should restore to pending", status)
		require.Equal(t, 1, job.Attempt)
		require.Nil(t, job.StartedAt)
		require.Nil(t, job.FinishedAt)
		require.Nil(t, job.HeartbeatAt)
		require.Empty(t, job.LockOwner)
		require.Empty(t, job.ErrorMessage)
	}
}

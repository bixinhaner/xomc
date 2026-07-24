-- Bounded recovery metadata for terminal hourly aggregation jobs.
ALTER TABLE public.async_jobs
    ADD COLUMN IF NOT EXISTS recovery_count integer NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS last_recovered_at timestamptz;

CREATE INDEX IF NOT EXISTS idx_async_jobs_hourly_failed_recovery
    ON public.async_jobs (bucket_start, recovery_count, last_recovered_at)
    WHERE job_type = 'pm_aggregate_hourly'
      AND status = 'failed'
      AND bucket_start IS NOT NULL
      AND bucket_end IS NOT NULL;

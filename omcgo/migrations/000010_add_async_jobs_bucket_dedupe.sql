-- +goose Up
-- +goose StatementBegin
ALTER TABLE public.async_jobs
    ADD COLUMN IF NOT EXISTS bucket_start timestamp with time zone,
    ADD COLUMN IF NOT EXISTS bucket_end timestamp with time zone;

COMMENT ON COLUMN public.async_jobs.bucket_start IS 'Bucket job source window start, for idempotent enqueue.';
COMMENT ON COLUMN public.async_jobs.bucket_end IS 'Bucket job source window end, for idempotent enqueue.';

CREATE UNIQUE INDEX IF NOT EXISTS uq_async_jobs_bucket
    ON public.async_jobs (job_type, bucket_start, bucket_end)
    WHERE bucket_start IS NOT NULL AND bucket_end IS NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS public.uq_async_jobs_bucket;

ALTER TABLE public.async_jobs
    DROP COLUMN IF EXISTS bucket_start,
    DROP COLUMN IF EXISTS bucket_end;
-- +goose StatementEnd

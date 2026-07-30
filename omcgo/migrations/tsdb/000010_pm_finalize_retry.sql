-- +goose NO TRANSACTION
-- +goose Up

ALTER TABLE public.pm_aggregation_windows
    ADD COLUMN IF NOT EXISTS finalize_attempts integer NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS finalize_next_attempt_at timestamptz NOT NULL DEFAULT '-infinity';

-- The previous due-claim index was published by migration 000009. Replace it
-- without editing migration history so deferred poison windows remain cheap to
-- skip on upgraded databases.
DROP INDEX CONCURRENTLY IF EXISTS public.idx_pm_windows_due_claim;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_pm_windows_due_claim
    ON public.pm_aggregation_windows (
        granularity, finalize_next_attempt_at, window_end,
        task_version_id, entity_key, window_start
    )
    WHERE status IN ('open', 'failed');

DROP INDEX CONCURRENTLY IF EXISTS public.idx_pm_windows_oldest_due;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_pm_windows_oldest_due
    ON public.pm_aggregation_windows (granularity, window_end)
    WHERE status IN ('open', 'failed');

-- +goose Down

DROP INDEX CONCURRENTLY IF EXISTS public.idx_pm_windows_oldest_due;
DROP INDEX CONCURRENTLY IF EXISTS public.idx_pm_windows_due_claim;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_pm_windows_due_claim
    ON public.pm_aggregation_windows (
        granularity, window_end, task_version_id, entity_key, window_start
    )
    WHERE status IN ('open', 'failed');

ALTER TABLE public.pm_aggregation_windows
    DROP COLUMN IF EXISTS finalize_next_attempt_at,
    DROP COLUMN IF EXISTS finalize_attempts;

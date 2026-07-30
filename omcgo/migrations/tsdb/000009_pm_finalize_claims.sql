-- +goose NO TRANSACTION
-- +goose Up

ALTER TABLE public.pm_aggregation_windows
    ADD COLUMN IF NOT EXISTS finalize_lease_owner uuid,
    ADD COLUMN IF NOT EXISTS finalize_lease_until timestamptz;

-- NO TRANSACTION migrations can stop after ALTER TABLE. Drop either a valid
-- or invalid leftover index so retry always converges on the same definition.
DROP INDEX CONCURRENTLY IF EXISTS public.idx_pm_windows_due_claim;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_pm_windows_due_claim
    ON public.pm_aggregation_windows (
        granularity, window_end, task_version_id, entity_key, window_start
    )
    WHERE status IN ('open', 'failed');

-- +goose Down

DROP INDEX CONCURRENTLY IF EXISTS public.idx_pm_windows_due_claim;

ALTER TABLE public.pm_aggregation_windows
    DROP COLUMN IF EXISTS finalize_lease_until,
    DROP COLUMN IF EXISTS finalize_lease_owner;

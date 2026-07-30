-- +goose NO TRANSACTION
-- +goose Up

ALTER TABLE public.pm_aggregation_windows
    ADD COLUMN IF NOT EXISTS version_audit_fingerprint text;

-- A failed CREATE INDEX CONCURRENTLY leaves an invalid index with the same
-- name. Drop either form first so retry converges after partial migration.
DROP INDEX CONCURRENTLY IF EXISTS public.idx_pm_windows_version_audit;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_pm_windows_version_audit
    ON public.pm_aggregation_windows (
        task_version_id, version_audit_fingerprint,
        window_start, entity_key, granularity
    )
    WHERE status = 'published'
      AND granularity IN ('daily', 'weekly', 'monthly');

-- +goose Down

DROP INDEX CONCURRENTLY IF EXISTS public.idx_pm_windows_version_audit;

ALTER TABLE public.pm_aggregation_windows
    DROP COLUMN IF EXISTS version_audit_fingerprint;

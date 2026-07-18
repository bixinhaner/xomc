-- +goose NO TRANSACTION

-- +goose Up
-- DispatchPending orders ready rows globally by created_at. The older
-- (status, next_attempt_at, created_at) index cannot satisfy that order across
-- multiple statuses, so PostgreSQL scans and sorts the complete backlog.
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_parameter_sync_outbox_ready_created
    ON public.parameter_sync_outbox (created_at, id)
    WHERE status IN ('pending', 'failed');

-- +goose Down
DROP INDEX CONCURRENTLY IF EXISTS public.idx_parameter_sync_outbox_ready_created;

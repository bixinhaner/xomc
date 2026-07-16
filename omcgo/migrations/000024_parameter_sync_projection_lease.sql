-- +goose Up
ALTER TABLE public.parameter_sync_runs
    ADD COLUMN projection_lease_token uuid,
    ADD COLUMN projection_lease_until timestamptz,
    ADD COLUMN projection_next_attempt_at timestamptz NOT NULL DEFAULT now();

CREATE INDEX idx_parameter_sync_runs_projection_due
    ON public.parameter_sync_runs (projection_next_attempt_at, completed_at, id)
    WHERE status = 'succeeded'
      AND sync_scope = 'full'
      AND projection_status <> 'completed';

-- +goose Down
DROP INDEX IF EXISTS public.idx_parameter_sync_runs_projection_due;

ALTER TABLE public.parameter_sync_runs
    DROP COLUMN IF EXISTS projection_next_attempt_at,
    DROP COLUMN IF EXISTS projection_lease_until,
    DROP COLUMN IF EXISTS projection_lease_token;

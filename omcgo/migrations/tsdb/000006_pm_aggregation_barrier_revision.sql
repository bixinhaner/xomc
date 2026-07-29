-- +goose Up
ALTER TABLE public.pm_aggregation_outbox
    ADD COLUMN event_window_start timestamptz,
    ADD COLUMN event_window_end timestamptz,
    ADD COLUMN consumed_at timestamptz,
    ADD COLUMN barrier_eligible boolean;

UPDATE public.pm_aggregation_outbox
SET event_window_start = (payload->>'window_start')::timestamptz,
    event_window_end = (payload->>'window_end')::timestamptz
WHERE event_window_start IS NULL;

-- Consumption cannot be inferred from publication. Rows that predate the
-- durable consumer acknowledgement protocol are excluded from the close
-- barrier; if one arrives late, the normal late-event rebuild path repairs it.
UPDATE public.pm_aggregation_outbox SET barrier_eligible = false;

ALTER TABLE public.pm_aggregation_outbox
    ALTER COLUMN event_window_start SET NOT NULL,
    ALTER COLUMN event_window_end SET NOT NULL,
    ALTER COLUMN barrier_eligible SET DEFAULT false,
    ALTER COLUMN barrier_eligible SET NOT NULL;

CREATE INDEX idx_pm_aggregation_outbox_consume_barrier
    ON public.pm_aggregation_outbox (event_window_start, created_at)
    WHERE consumed_at IS NULL AND barrier_eligible;

ALTER TABLE public.pm_aggregation_rollup_outbox
    ADD COLUMN consumed_at timestamptz,
    ADD COLUMN barrier_eligible boolean;

UPDATE public.pm_aggregation_rollup_outbox
SET barrier_eligible = false;

ALTER TABLE public.pm_aggregation_rollup_outbox
    ALTER COLUMN barrier_eligible SET DEFAULT false,
    ALTER COLUMN barrier_eligible SET NOT NULL;

CREATE INDEX idx_pm_aggregation_rollup_consume_barrier
    ON public.pm_aggregation_rollup_outbox (subject, window_start)
    WHERE consumed_at IS NULL AND barrier_eligible;

ALTER TABLE public.pm_aggregation_windows
    ADD COLUMN revision integer NOT NULL DEFAULT 1,
    ADD COLUMN rebuild_requested_at timestamptz,
    ADD COLUMN version_effective_from timestamptz,
    ADD COLUMN version_effective_to timestamptz;

ALTER TABLE public.pm_aggregation_windows
    DROP CONSTRAINT chk_pm_aggregation_windows_status;
ALTER TABLE public.pm_aggregation_windows
    ADD CONSTRAINT chk_pm_aggregation_windows_status
        CHECK (status IN ('open', 'finalizing', 'published', 'failed', 'rebuilding'));

ALTER TABLE public.pm_aggregation_results
    ADD COLUMN revision integer NOT NULL DEFAULT 1,
    ADD COLUMN version_effective_from timestamptz,
    ADD COLUMN version_effective_to timestamptz,
    ADD COLUMN received_slots bigint NOT NULL DEFAULT 0,
    ADD COLUMN expected_slots bigint NOT NULL DEFAULT 0,
    ADD COLUMN version_expected_slots bigint NOT NULL DEFAULT 0,
    ADD COLUMN natural_expected_slots bigint NOT NULL DEFAULT 0,
    ADD COLUMN version_slice_complete boolean NOT NULL DEFAULT false,
    ADD COLUMN period_complete boolean NOT NULL DEFAULT false;

CREATE OR REPLACE VIEW public.pm_adhoc_aggregation_results AS
SELECT
    r.id,
    r.task_id,
    r.device_oui,
    CASE WHEN r.dimension = 'device' THEN r.device_sn ELSE 'AGGREGATED' END::text AS device_sn,
    CASE WHEN r.dimension = 'product' THEN r.dimension_key::uuid ELSE NULL::uuid END AS product_id,
    r.metric_path,
    r.metric_type,
    r.metric_value,
    r.aggregation_op::text AS statis_type,
    r.granularity::text,
    r.window_start AS "time",
    r.window_start AS start_time,
    r.window_end AS end_time,
    r.created_at AS ingest_time,
    CASE
        WHEN r.dimension = 'device' THEN NULLIF(r.object_ldn, '')
        WHEN r.dimension = 'network' THEN 'Network'
        ELSE r.dimension_key
    END::text AS object_ldn,
    jsonb_build_object(
        'task_version_id', r.task_version_id,
        'complete', r.period_complete,
        'missing_slots', r.missing_slots,
        'dimension', r.dimension,
        'revision', r.revision,
        'version_effective_from', r.version_effective_from,
        'version_effective_to', r.version_effective_to,
        'received_slots', r.received_slots,
        'expected_slots', r.expected_slots,
        'version_expected_slots', r.version_expected_slots,
        'natural_expected_slots', r.natural_expected_slots,
        'version_slice_complete', r.version_slice_complete,
        'period_complete', r.period_complete,
        'active_version', r.task_version_id = (
            SELECT candidate.task_version_id
            FROM public.pm_aggregation_results candidate
            WHERE candidate.task_id = r.task_id
              AND candidate.granularity = r.granularity
              AND candidate.window_start = r.window_start
            GROUP BY candidate.task_version_id, candidate.version_effective_from
            ORDER BY candidate.version_effective_from DESC NULLS LAST,
                     MAX(candidate.revision) DESC,
                     MAX(candidate.created_at) DESC,
                     candidate.task_version_id DESC
            LIMIT 1
        ) AND NOT EXISTS (
            SELECT 1
            FROM public.pm_aggregation_windows active_window
            WHERE active_window.task_id = r.task_id
              AND active_window.granularity = r.granularity
              AND active_window.window_start = r.window_start
              AND active_window.task_version_id <> r.task_version_id
              AND active_window.status IN ('open', 'finalizing', 'rebuilding', 'failed')
              AND active_window.version_effective_from IS NOT NULL
              AND (
                  r.version_effective_from IS NULL
                  OR active_window.version_effective_from > r.version_effective_from
              )
        ),
        'partial', false
    ) AS extra
FROM public.pm_aggregation_results r;

CREATE TABLE public.pm_aggregation_rebuilds (
    id bigserial PRIMARY KEY,
    task_id uuid NOT NULL,
    task_version_id uuid NOT NULL,
    entity_key text NOT NULL,
    granularity varchar(16) NOT NULL,
    window_start timestamptz NOT NULL,
    window_end timestamptz NOT NULL,
    source_event_id text NOT NULL,
    request_generation bigint NOT NULL DEFAULT 1,
    status varchar(16) NOT NULL DEFAULT 'pending',
    attempts integer NOT NULL DEFAULT 0,
    last_error text,
    requested_at timestamptz NOT NULL DEFAULT now(),
    next_attempt_at timestamptz NOT NULL DEFAULT now(),
    started_at timestamptz,
    lease_expires_at timestamptz,
    lease_owner uuid,
    completed_at timestamptz,
    CONSTRAINT chk_pm_aggregation_rebuilds_granularity
        CHECK (granularity IN ('hourly', 'daily', 'weekly', 'monthly')),
    CONSTRAINT chk_pm_aggregation_rebuilds_status
        CHECK (status IN ('pending', 'running', 'completed', 'failed')),
    UNIQUE (task_version_id, entity_key, granularity, window_start)
);

CREATE INDEX idx_pm_aggregation_rebuilds_pending
    ON public.pm_aggregation_rebuilds (next_attempt_at, requested_at, id)
    WHERE status IN ('pending', 'failed', 'running');

-- +goose Down
CREATE OR REPLACE VIEW public.pm_adhoc_aggregation_results AS
SELECT
    r.id, r.task_id, r.device_oui,
    CASE WHEN r.dimension = 'device' THEN r.device_sn ELSE 'AGGREGATED' END::text AS device_sn,
    CASE WHEN r.dimension = 'product' THEN r.dimension_key::uuid ELSE NULL::uuid END AS product_id,
    r.metric_path, r.metric_type, r.metric_value,
    r.aggregation_op::text AS statis_type, r.granularity::text,
    r.window_start AS "time", r.window_start AS start_time, r.window_end AS end_time,
    r.created_at AS ingest_time,
    CASE
        WHEN r.dimension = 'device' THEN NULLIF(r.object_ldn, '')
        WHEN r.dimension = 'network' THEN 'Network'
        ELSE r.dimension_key
    END::text AS object_ldn,
    jsonb_build_object(
        'task_version_id', r.task_version_id,
        'complete', r.complete,
        'missing_slots', r.missing_slots,
        'dimension', r.dimension
    ) AS extra
FROM public.pm_aggregation_results r;
DROP TABLE IF EXISTS public.pm_aggregation_rebuilds;
ALTER TABLE public.pm_aggregation_results
    DROP COLUMN IF EXISTS period_complete,
    DROP COLUMN IF EXISTS version_slice_complete,
    DROP COLUMN IF EXISTS natural_expected_slots,
    DROP COLUMN IF EXISTS version_expected_slots,
    DROP COLUMN IF EXISTS expected_slots,
    DROP COLUMN IF EXISTS received_slots,
    DROP COLUMN IF EXISTS version_effective_to,
    DROP COLUMN IF EXISTS version_effective_from,
    DROP COLUMN IF EXISTS revision;
ALTER TABLE public.pm_aggregation_windows
    DROP CONSTRAINT chk_pm_aggregation_windows_status;
ALTER TABLE public.pm_aggregation_windows
    ADD CONSTRAINT chk_pm_aggregation_windows_status
        CHECK (status IN ('open', 'finalizing', 'published', 'failed'));
ALTER TABLE public.pm_aggregation_windows
    DROP COLUMN IF EXISTS version_effective_to,
    DROP COLUMN IF EXISTS version_effective_from,
    DROP COLUMN IF EXISTS rebuild_requested_at,
    DROP COLUMN IF EXISTS revision;
DROP INDEX IF EXISTS public.idx_pm_aggregation_rollup_consume_barrier;
ALTER TABLE public.pm_aggregation_rollup_outbox
    DROP COLUMN IF EXISTS barrier_eligible,
    DROP COLUMN IF EXISTS consumed_at;
DROP INDEX IF EXISTS public.idx_pm_aggregation_outbox_consume_barrier;
ALTER TABLE public.pm_aggregation_outbox
    DROP COLUMN IF EXISTS barrier_eligible,
    DROP COLUMN IF EXISTS consumed_at,
    DROP COLUMN IF EXISTS event_window_end,
    DROP COLUMN IF EXISTS event_window_start;

-- +goose Up
-- Upgrade bridge for installations that already applied v6. Pre-v7 rows did
-- not carry a durable consumer acknowledgement, so publication must not be
-- presented as consumption.
ALTER TABLE public.pm_aggregation_outbox
    ADD COLUMN IF NOT EXISTS barrier_eligible boolean;
UPDATE public.pm_aggregation_outbox
SET barrier_eligible = false
WHERE barrier_eligible IS NULL;
ALTER TABLE public.pm_aggregation_outbox
    ALTER COLUMN barrier_eligible SET DEFAULT false,
    ALTER COLUMN barrier_eligible SET NOT NULL;

ALTER TABLE public.pm_aggregation_rollup_outbox
    ADD COLUMN IF NOT EXISTS barrier_eligible boolean;
UPDATE public.pm_aggregation_rollup_outbox
SET barrier_eligible = false
WHERE barrier_eligible IS NULL;
ALTER TABLE public.pm_aggregation_rollup_outbox
    ALTER COLUMN barrier_eligible SET DEFAULT false,
    ALTER COLUMN barrier_eligible SET NOT NULL;

DROP INDEX IF EXISTS public.idx_pm_aggregation_outbox_consume_barrier;
CREATE INDEX idx_pm_aggregation_outbox_consume_barrier
    ON public.pm_aggregation_outbox (event_window_start, created_at)
    WHERE consumed_at IS NULL AND barrier_eligible;
DROP INDEX IF EXISTS public.idx_pm_aggregation_rollup_consume_barrier;
CREATE INDEX idx_pm_aggregation_rollup_consume_barrier
    ON public.pm_aggregation_rollup_outbox (subject, window_start)
    WHERE consumed_at IS NULL AND barrier_eligible;

-- The transport outbox stays small. Rebuild sources live in a dedicated
-- hypertable whose old chunks can be compressed and dropped independently.
CREATE TABLE public.pm_aggregation_replay_sources (
    event_window_start timestamptz NOT NULL,
    event_id uuid NOT NULL,
    payload jsonb NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (event_window_start, event_id)
);

SELECT create_hypertable(
    'public.pm_aggregation_replay_sources',
    by_range('event_window_start', INTERVAL '1 day'),
    if_not_exists => TRUE,
    migrate_data => TRUE
);

ALTER TABLE public.pm_aggregation_replay_sources SET (
    timescaledb.compress,
    timescaledb.compress_orderby = 'event_window_start, event_id'
);
SELECT add_compression_policy(
    'public.pm_aggregation_replay_sources',
    INTERVAL '1 day',
    if_not_exists => TRUE
);

-- Do not bulk-copy the legacy JSON outbox here: at carrier scale that would
-- create a multi-million-row WAL-heavy migration transaction. Legacy rows
-- remain readable through the replay UNION until their original 45-day
-- horizon expires; new-protocol writes explicitly opt into the barrier and
-- are copied transactionally into this compressed store.

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

-- +goose Down
SELECT remove_compression_policy(
    'public.pm_aggregation_replay_sources',
    if_exists => TRUE
);
DROP TABLE IF EXISTS public.pm_aggregation_replay_sources;
-- barrier_eligible is retained on downgrade because v6 installations may
-- already depend on the truthful cutover marker.

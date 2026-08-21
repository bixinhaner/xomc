-- Pre-release baseline compatibility reconciliation.
--
-- Before versioned migrations are enabled, schema changes are folded into the
-- 000001 baseline. Goose correctly does not rerun an already-applied baseline,
-- so existing test/production databases need this small idempotent bridge.
-- Keep these statements in sync with migrations/tsdb/000001_tsdb_schema.sql.
BEGIN;

ALTER TABLE public.pm_files
    ADD COLUMN IF NOT EXISTS measurement_start timestamptz,
    ADD COLUMN IF NOT EXISTS measurement_end timestamptz;

CREATE INDEX IF NOT EXISTS idx_pm_files_measurement_slot
    ON public.pm_files (measurement_end, technology, carrier, device_id)
    WHERE measurement_end IS NOT NULL AND parsed = true;

CREATE TABLE IF NOT EXISTS public.pm_slot_health (
    slot_start timestamptz NOT NULL,
    slot_end timestamptz NOT NULL,
    technology varchar(16) NOT NULL,
    carrier varchar(16) NOT NULL,
    expected_devices bigint NOT NULL,
    received_devices bigint NOT NULL,
    coverage_ratio double precision NOT NULL,
    expected_snapshot_version text NOT NULL DEFAULT '',
    evaluated_at timestamptz NOT NULL,
    status varchar(32) NOT NULL,
    PRIMARY KEY (slot_end, technology, carrier),
    CONSTRAINT chk_pm_slot_health_window CHECK (slot_end > slot_start),
    CONSTRAINT chk_pm_slot_health_counts CHECK (
        expected_devices >= 0 AND received_devices >= 0
    ),
    CONSTRAINT chk_pm_slot_health_coverage CHECK (
        coverage_ratio >= 0 AND coverage_ratio <= 1
    ),
    CONSTRAINT chk_pm_slot_health_status CHECK (
        status IN ('complete', 'partial', 'missing', 'bootstrap_ignored')
    )
);

CREATE INDEX IF NOT EXISTS idx_pm_slot_health_latest
    ON public.pm_slot_health (technology, carrier, slot_end DESC);

-- Early observer versions marked every startup-crossing slot as ignored even
-- when the complete expected device set had already arrived. That state is
-- provably complete and safe to repair idempotently; partial startup slots stay
-- ignored because their missing segment cannot be reconstructed.
UPDATE public.pm_slot_health
SET status = 'complete'
WHERE status = 'bootstrap_ignored'
  AND coverage_ratio >= 0.98;

-- #360: durable PM replay must filter by device before decoding payloads.
-- Existing one-day chunks can already be compressed, so bulk backfill follows
-- the Timescale-safe order: pause policy -> decompress captured chunks ->
-- backfill/validate -> change segmentby/indexes -> recompress -> restore policy.
ALTER TABLE public.pm_aggregation_outbox
    ADD COLUMN IF NOT EXISTS device_id uuid;
ALTER TABLE public.pm_aggregation_replay_sources
    ADD COLUMN IF NOT EXISTS device_id uuid;

-- +goose StatementBegin
DO $pm_replay_reconcile$
DECLARE
    reconcile_needed boolean;
    policy_job record;
    paused_jobs integer[] := ARRAY[]::integer[];
    chunk_row record;
    compressed_chunks regclass[] := ARRAY[]::regclass[];
    compressed_chunk regclass;
    paused_job_id integer;
BEGIN
    PERFORM pg_advisory_xact_lock(
        hashtextextended('omcgo-tsdb-pm-replay-device-reconcile', 0)
    );

    SELECT
        NOT COALESCE((
            SELECT a.attnotnull
            FROM pg_attribute a
            WHERE a.attrelid = 'public.pm_aggregation_outbox'::regclass
              AND a.attname = 'device_id' AND NOT a.attisdropped
        ), false)
        OR NOT COALESCE((
            SELECT a.attnotnull
            FROM pg_attribute a
            WHERE a.attrelid = 'public.pm_aggregation_replay_sources'::regclass
              AND a.attname = 'device_id' AND NOT a.attisdropped
        ), false)
        OR to_regclass('public.idx_pm_aggregation_outbox_device_period_replay') IS NULL
        OR to_regclass('public.idx_pm_replay_sources_device_period') IS NULL
        OR NOT EXISTS (
            SELECT 1
            FROM timescaledb_information.compression_settings
            WHERE hypertable_schema = 'public'
              AND hypertable_name = 'pm_aggregation_replay_sources'
              AND attname = 'device_id'
              AND segmentby_column_index = 1
        )
    INTO reconcile_needed;

    IF NOT reconcile_needed THEN
        RETURN;
    END IF;

    FOR policy_job IN
        SELECT job_id
        FROM timescaledb_information.jobs
        WHERE proc_name = 'policy_compression'
          AND hypertable_schema = 'public'
          AND hypertable_name = 'pm_aggregation_replay_sources'
          AND scheduled
    LOOP
        PERFORM alter_job(policy_job.job_id, scheduled => false);
        paused_jobs := array_append(paused_jobs, policy_job.job_id);
    END LOOP;

    FOR chunk_row IN
        SELECT format('%I.%I', chunk_schema, chunk_name)::regclass AS chunk
        FROM timescaledb_information.chunks
        WHERE hypertable_schema = 'public'
          AND hypertable_name = 'pm_aggregation_replay_sources'
          AND is_compressed
        ORDER BY range_start, chunk_name
    LOOP
        compressed_chunks := array_append(compressed_chunks, chunk_row.chunk);
        PERFORM decompress_chunk(chunk_row.chunk, true);
    END LOOP;

    UPDATE public.pm_aggregation_outbox
    SET device_id = (payload->>'device_id')::uuid
    WHERE device_id IS NULL;

    UPDATE public.pm_aggregation_replay_sources
    SET device_id = (payload->>'device_id')::uuid
    WHERE device_id IS NULL;

    IF EXISTS (SELECT 1 FROM public.pm_aggregation_outbox WHERE device_id IS NULL) THEN
        RAISE EXCEPTION 'pm_aggregation_outbox device_id backfill left NULL rows';
    END IF;
    IF EXISTS (SELECT 1 FROM public.pm_aggregation_replay_sources WHERE device_id IS NULL) THEN
        RAISE EXCEPTION 'pm_aggregation_replay_sources device_id backfill left NULL rows';
    END IF;

    ALTER TABLE public.pm_aggregation_outbox
        ALTER COLUMN device_id SET NOT NULL;
    ALTER TABLE public.pm_aggregation_replay_sources
        ALTER COLUMN device_id SET NOT NULL;
    ALTER TABLE public.pm_aggregation_replay_sources SET (
        timescaledb.compress,
        timescaledb.compress_segmentby = 'device_id',
        timescaledb.compress_orderby = 'event_window_start, event_id'
    );

    CREATE INDEX IF NOT EXISTS idx_pm_aggregation_outbox_device_period_replay
        ON public.pm_aggregation_outbox (device_id, event_window_start, event_id)
        WHERE NOT barrier_eligible;
    CREATE INDEX IF NOT EXISTS idx_pm_replay_sources_device_period
        ON public.pm_aggregation_replay_sources (device_id, event_window_start, event_id);

    FOREACH compressed_chunk IN ARRAY compressed_chunks
    LOOP
        PERFORM compress_chunk(compressed_chunk, true);
    END LOOP;

    IF EXISTS (
        SELECT 1
        FROM timescaledb_information.chunks c
        LEFT JOIN timescaledb_information.chunk_compression_settings s
          ON s.chunk = format('%I.%I', c.chunk_schema, c.chunk_name)::regclass
        WHERE c.hypertable_schema = 'public'
          AND c.hypertable_name = 'pm_aggregation_replay_sources'
          AND c.is_compressed
          AND COALESCE(s.segmentby, '') <> 'device_id'
    ) THEN
        RAISE EXCEPTION 'compressed PM replay chunks do not use device_id segmentby';
    END IF;

    FOREACH paused_job_id IN ARRAY paused_jobs
    LOOP
        PERFORM alter_job(paused_job_id, scheduled => true);
    END LOOP;
END
$pm_replay_reconcile$;
-- +goose StatementEnd

COMMIT;

-- +goose Up
-- Existing databases that had already applied the former incremental
-- parameter-sync migrations do not replay 000001 after the migrations were
-- consolidated. Bridge durable-routing objects added to the baseline later.

ALTER TABLE public.parameter_sync_requests
    ADD COLUMN IF NOT EXISTS source_event_id varchar(128),
    ADD COLUMN IF NOT EXISTS origin_event_type varchar(64),
    ADD COLUMN IF NOT EXISTS model_upload_intent_id uuid,
    ADD COLUMN IF NOT EXISTS model_upload_status varchar(24),
    ADD COLUMN IF NOT EXISTS admission_class varchar(32),
    ADD COLUMN IF NOT EXISTS admission_reason text,
    ADD COLUMN IF NOT EXISTS admission_snapshot jsonb,
    ADD COLUMN IF NOT EXISTS deduplicated_to_request_id uuid;

CREATE TABLE IF NOT EXISTS public.model_upload_intents (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id uuid NOT NULL,
    upload_task_id uuid NOT NULL,
    discovery_log_id uuid,
    source_event_id varchar(128) NOT NULL,
    model_version varchar(128),
    model_hash varchar(128),
    status varchar(24) NOT NULL DEFAULT 'requested'
        CHECK (status IN ('requested', 'uploaded', 'not_supported', 'failed',
                          'sync_queued', 'sync_submitted', 'manual_review')),
    failure_code varchar(64),
    failure_message text,
    attempts integer NOT NULL DEFAULT 0,
    next_attempt_at timestamptz NOT NULL DEFAULT now(),
    last_error text,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_model_upload_intents_device_task
    ON public.model_upload_intents (device_id, upload_task_id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_model_upload_intents_source_event
    ON public.model_upload_intents (source_event_id);
CREATE INDEX IF NOT EXISTS idx_model_upload_intents_dispatch
    ON public.model_upload_intents (status, next_attempt_at, created_at);

CREATE TABLE IF NOT EXISTS public.parameter_sync_admission_state (
    admission_class varchar(32) NOT NULL,
    bucket_id smallint NOT NULL,
    active_run_limit integer NOT NULL DEFAULT 0,
    active_task_limit integer NOT NULL DEFAULT 0,
    missing_result_limit integer NOT NULL DEFAULT 0,
    create_rate_per_minute integer NOT NULL DEFAULT 0,
    reserved_runs integer NOT NULL DEFAULT 0,
    reserved_tasks integer NOT NULL DEFAULT 0,
    version bigint NOT NULL DEFAULT 0,
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (admission_class, bucket_id),
    CHECK (bucket_id >= 0),
    CHECK (admission_class IN ('global', 'model_upload', 'periodic', 'manual')),
    CHECK (active_run_limit >= 0 AND active_task_limit >= 0
           AND missing_result_limit >= 0 AND create_rate_per_minute >= 0
           AND reserved_runs >= 0 AND reserved_tasks >= 0)
);

CREATE TABLE IF NOT EXISTS public.parameter_sync_admission_reservations (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    request_id uuid NOT NULL,
    admission_class varchar(32) NOT NULL,
    bucket_id smallint NOT NULL,
    reserved_runs integer NOT NULL DEFAULT 0,
    reserved_tasks integer NOT NULL DEFAULT 0,
    status varchar(16) NOT NULL DEFAULT 'reserved'
        CHECK (status IN ('reserved', 'released', 'expired')),
    lease_until timestamptz NOT NULL DEFAULT now(),
    released_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CHECK (reserved_runs >= 0 AND reserved_tasks >= 0)
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_parameter_sync_admission_reservation_request
    ON public.parameter_sync_admission_reservations (request_id, admission_class);
CREATE INDEX IF NOT EXISTS idx_parameter_sync_admission_reservations_bucket
    ON public.parameter_sync_admission_reservations (admission_class, bucket_id, status);
CREATE INDEX IF NOT EXISTS idx_parameter_sync_admission_reservations_due
    ON public.parameter_sync_admission_reservations (status, lease_until, updated_at);

-- +goose StatementBegin
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'parameter_sync_admission_reservations_request_id_fkey'
          AND conrelid = 'public.parameter_sync_admission_reservations'::regclass
    ) THEN
        ALTER TABLE public.parameter_sync_admission_reservations
            ADD CONSTRAINT parameter_sync_admission_reservations_request_id_fkey
            FOREIGN KEY (request_id) REFERENCES public.parameter_sync_requests(id)
            ON DELETE CASCADE;
    END IF;
END
$$;
-- +goose StatementEnd

CREATE TABLE IF NOT EXISTS public.parameter_sync_event_failures (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    subject varchar(255) NOT NULL,
    event_id varchar(128) NOT NULL,
    device_id uuid,
    device_sn varchar(64),
    request_id uuid,
    run_id uuid,
    task_id uuid,
    raw_payload jsonb NOT NULL DEFAULT '{}'::jsonb,
    delivery_count integer NOT NULL DEFAULT 0,
    status varchar(24) NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'replayed', 'recovered', 'manual_review')),
    last_error text,
    next_retry_at timestamptz,
    replayed_at timestamptz,
    recovered_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_parameter_sync_event_failure_event
    ON public.parameter_sync_event_failures (subject, event_id);
CREATE INDEX IF NOT EXISTS idx_parameter_sync_event_failures_replay
    ON public.parameter_sync_event_failures (status, next_retry_at, created_at);
CREATE INDEX IF NOT EXISTS idx_parameter_sync_event_failures_run_task
    ON public.parameter_sync_event_failures (run_id, task_id)
    WHERE run_id IS NOT NULL OR task_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS public.parameter_sync_recovery_state (
    run_id uuid NOT NULL,
    task_id uuid NOT NULL,
    status varchar(24) NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'processing', 'processed', 'failed', 'manual_review')),
    attempts integer NOT NULL DEFAULT 0,
    next_retry_at timestamptz NOT NULL DEFAULT now(),
    lease_token uuid,
    lease_until timestamptz,
    last_error text,
    claimed_at timestamptz,
    processed_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (run_id, task_id)
);

CREATE INDEX IF NOT EXISTS idx_parameter_sync_recovery_claim
    ON public.parameter_sync_recovery_state (status, next_retry_at, lease_until, updated_at);
CREATE INDEX IF NOT EXISTS idx_parameter_sync_requests_source_event
    ON public.parameter_sync_requests (source_event_id)
    WHERE source_event_id IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS uq_parameter_sync_requests_model_upload_intent
    ON public.parameter_sync_requests (model_upload_intent_id)
    WHERE model_upload_intent_id IS NOT NULL;

-- +goose Down
-- This migration repairs pre-existing installations. Keep rollback
-- non-destructive because newer binaries may already have written these fields.
SELECT 1;

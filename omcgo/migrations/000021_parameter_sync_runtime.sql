-- +goose Up
ALTER TABLE public.product_unsupported_paths
    ADD COLUMN firmware_version varchar(128) NOT NULL DEFAULT '';

ALTER TABLE public.product_unsupported_paths
    DROP CONSTRAINT uq_product_unsupported_path;

ALTER TABLE public.product_unsupported_paths
    ADD CONSTRAINT uq_product_unsupported_path
    UNIQUE (product_id, firmware_version, standard_path);

CREATE INDEX idx_product_unsupported_paths_product_firmware
    ON public.product_unsupported_paths (product_id, firmware_version)
    WHERE read_unsupported = true;

CREATE TABLE public.parameter_sync_requests (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id uuid NOT NULL,
    device_sn varchar(64) NOT NULL,
    caller_type varchar(32) NOT NULL DEFAULT 'system',
    trigger_reason varchar(32) NOT NULL,
    sync_scope varchar(24) NOT NULL,
    requested_paths jsonb NOT NULL DEFAULT '[]'::jsonb,
    status varchar(24) NOT NULL DEFAULT 'accepted',
    run_id uuid,
    active_run_id uuid,
    priority integer NOT NULL DEFAULT 10,
    next_attempt_at timestamptz NOT NULL DEFAULT now(),
    deadline_at timestamptz,
    idempotency_key varchar(160),
    result_code varchar(64),
    result_summary jsonb,
    error_message text,
    campaign_id uuid,
    created_at timestamptz NOT NULL DEFAULT now(),
    started_at timestamptz,
    completed_at timestamptz,
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT parameter_sync_requests_status_chk CHECK (status IN (
        'accepted','queued','running','succeeded','failed','timed_out',
        'cancelled','deduplicated','rejected'
    )),
    CONSTRAINT parameter_sync_requests_scope_chk CHECK (sync_scope IN (
        'full','partial','readback','policy_probe'
    )),
	CONSTRAINT parameter_sync_requests_trigger_reason_chk CHECK (trigger_reason IN (
		'bootstrap','model_upload','device_online','firmware_changed','periodic','manual',
		'config_pull','license','spv_readback','add_object_readback','inform_period_probe'
	))
);

CREATE UNIQUE INDEX uq_parameter_sync_requests_idempotency
    ON public.parameter_sync_requests (caller_type, idempotency_key)
    WHERE idempotency_key IS NOT NULL;
CREATE INDEX idx_parameter_sync_requests_schedule
    ON public.parameter_sync_requests (status, priority, next_attempt_at, created_at);
CREATE INDEX idx_parameter_sync_requests_device_history
    ON public.parameter_sync_requests (device_id, created_at DESC);

CREATE TABLE public.parameter_sync_runs (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    request_id uuid NOT NULL REFERENCES public.parameter_sync_requests(id) ON DELETE CASCADE,
    device_id uuid NOT NULL,
    device_sn varchar(64) NOT NULL,
    trigger_reason varchar(32) NOT NULL,
    sync_scope varchar(24) NOT NULL,
    mapping_source varchar(128),
    mapping_version varchar(128),
    coverage jsonb NOT NULL DEFAULT '[]'::jsonb,
    status varchar(24) NOT NULL DEFAULT 'planning',
    expected_task_count integer NOT NULL DEFAULT 0,
    terminal_task_count integer NOT NULL DEFAULT 0,
    processed_task_count integer NOT NULL DEFAULT 0,
    failed_task_count integer NOT NULL DEFAULT 0,
    error_message text,
    started_at timestamptz NOT NULL DEFAULT now(),
    completed_at timestamptz,
    version bigint NOT NULL DEFAULT 0,
    projection_status varchar(16) NOT NULL DEFAULT 'pending',
    projection_attempts integer NOT NULL DEFAULT 0,
    projection_error text,
    projection_completed_at timestamptz,
    CONSTRAINT parameter_sync_runs_status_chk CHECK (status IN (
        'planning','enqueuing','waiting_device','executing','processing','cancelling',
        'succeeded','failed','cancelled'
    )),
    CONSTRAINT parameter_sync_runs_scope_chk CHECK (sync_scope IN (
        'full','partial','readback','policy_probe'
    )),
	CONSTRAINT parameter_sync_runs_trigger_reason_chk CHECK (trigger_reason IN (
		'bootstrap','model_upload','device_online','firmware_changed','periodic','manual',
		'config_pull','license','spv_readback','add_object_readback','inform_period_probe'
	)),
    CONSTRAINT parameter_sync_runs_counts_chk CHECK (
        expected_task_count >= 0 AND terminal_task_count >= 0 AND
        processed_task_count >= 0 AND failed_task_count >= 0 AND
        terminal_task_count <= expected_task_count AND
        processed_task_count <= terminal_task_count AND
        failed_task_count <= processed_task_count
    )
);

CREATE UNIQUE INDEX uq_parameter_sync_runs_active_device
    ON public.parameter_sync_runs (device_id)
    WHERE status IN ('planning','enqueuing','waiting_device','executing','processing','cancelling');
CREATE INDEX idx_parameter_sync_runs_device_history
    ON public.parameter_sync_runs (device_id, started_at DESC);
CREATE INDEX idx_parameter_sync_runs_request ON public.parameter_sync_runs (request_id);
CREATE INDEX idx_parameter_sync_runs_projection_pending
    ON public.parameter_sync_runs (completed_at, id)
    WHERE status = 'succeeded' AND sync_scope = 'full' AND projection_status <> 'completed';

ALTER TABLE public.parameter_sync_requests
    ADD CONSTRAINT parameter_sync_requests_run_fk
    FOREIGN KEY (run_id) REFERENCES public.parameter_sync_runs(id) ON DELETE SET NULL;
ALTER TABLE public.parameter_sync_requests
    ADD CONSTRAINT parameter_sync_requests_active_run_fk
    FOREIGN KEY (active_run_id) REFERENCES public.parameter_sync_runs(id) ON DELETE SET NULL;

CREATE TABLE public.parameter_sync_request_bindings (
    request_id uuid NOT NULL REFERENCES public.parameter_sync_requests(id) ON DELETE CASCADE,
    run_id uuid NOT NULL REFERENCES public.parameter_sync_runs(id) ON DELETE CASCADE,
	provisioning_task_id uuid NOT NULL REFERENCES public.provisioning_tasks(id) ON DELETE CASCADE,
    status varchar(24) NOT NULL DEFAULT 'waiting',
    created_at timestamptz NOT NULL DEFAULT now(),
    completed_at timestamptz,
    PRIMARY KEY (request_id, provisioning_task_id),
    CONSTRAINT parameter_sync_bindings_status_chk CHECK (status IN ('waiting','completed','failed','cancelled'))
);
CREATE INDEX idx_parameter_sync_bindings_run ON public.parameter_sync_request_bindings (run_id, status);

CREATE TABLE public.parameter_sync_task_results (
    run_id uuid NOT NULL REFERENCES public.parameter_sync_runs(id) ON DELETE CASCADE,
	task_id uuid NOT NULL REFERENCES public.device_tasks(id) ON DELETE CASCADE,
    event_id varchar(128) NOT NULL,
    success boolean NOT NULL,
    result_ref text,
    status varchar(24) NOT NULL DEFAULT 'received',
    error_code varchar(64),
    error_message text,
    processed_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (run_id, task_id),
    CONSTRAINT parameter_sync_task_results_status_chk CHECK (status IN ('received','processed','failed'))
);
CREATE UNIQUE INDEX uq_parameter_sync_task_results_event ON public.parameter_sync_task_results (event_id);

CREATE TABLE public.parameter_sync_staging_values (
    run_id uuid NOT NULL REFERENCES public.parameter_sync_runs(id) ON DELETE CASCADE,
    parameter_path text NOT NULL,
    private_path text NOT NULL,
    value jsonb,
    value_type varchar(64),
    writable boolean NOT NULL DEFAULT false,
	fap_instance integer NOT NULL DEFAULT 0,
	param_group varchar(32) NOT NULL DEFAULT 'other',
    coverage_scope text,
	task_id uuid REFERENCES public.device_tasks(id) ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (run_id, parameter_path)
);

CREATE TABLE public.parameter_sync_outbox (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type varchar(64) NOT NULL,
    aggregate_type varchar(32) NOT NULL,
    aggregate_id uuid NOT NULL,
    dedupe_key varchar(192) NOT NULL,
    payload jsonb NOT NULL,
    status varchar(24) NOT NULL DEFAULT 'pending',
    attempt_count integer NOT NULL DEFAULT 0,
    next_attempt_at timestamptz NOT NULL DEFAULT now(),
    last_error text,
    delivered_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT parameter_sync_outbox_status_chk CHECK (status IN ('pending','delivering','delivered','failed','dead')),
    CONSTRAINT parameter_sync_outbox_attempt_chk CHECK (attempt_count >= 0),
    CONSTRAINT uq_parameter_sync_outbox_dedupe UNIQUE (dedupe_key)
);
CREATE INDEX idx_parameter_sync_outbox_dispatch
    ON public.parameter_sync_outbox (status, next_attempt_at, created_at)
    WHERE status IN ('pending','failed');

CREATE TABLE public.parameter_sync_device_state (
    device_id uuid PRIMARY KEY,
    consecutive_failures integer NOT NULL DEFAULT 0 CHECK (consecutive_failures >= 0),
    last_attempt_at timestamptz,
    last_success_at timestamptz,
    last_failure_at timestamptz,
    next_auto_sync_at timestamptz,
    last_error text,
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_parameter_sync_device_state_due
    ON public.parameter_sync_device_state (next_auto_sync_at)
    WHERE next_auto_sync_at IS NOT NULL;

-- +goose Down
ALTER TABLE public.parameter_sync_requests DROP CONSTRAINT IF EXISTS parameter_sync_requests_active_run_fk;
ALTER TABLE public.parameter_sync_requests DROP CONSTRAINT IF EXISTS parameter_sync_requests_run_fk;
DROP TABLE IF EXISTS public.parameter_sync_device_state;
DROP TABLE IF EXISTS public.parameter_sync_outbox;
DROP TABLE IF EXISTS public.parameter_sync_staging_values;
DROP TABLE IF EXISTS public.parameter_sync_task_results;
DROP TABLE IF EXISTS public.parameter_sync_request_bindings;
DROP TABLE IF EXISTS public.parameter_sync_runs;
DROP TABLE IF EXISTS public.parameter_sync_requests;

DROP INDEX IF EXISTS public.idx_product_unsupported_paths_product_firmware;

DELETE FROM public.product_unsupported_paths newer
USING public.product_unsupported_paths older
WHERE newer.product_id = older.product_id
  AND newer.standard_path = older.standard_path
  AND newer.id > older.id;

ALTER TABLE public.product_unsupported_paths
    DROP CONSTRAINT uq_product_unsupported_path;

ALTER TABLE public.product_unsupported_paths
    DROP COLUMN firmware_version;

ALTER TABLE public.product_unsupported_paths
    ADD CONSTRAINT uq_product_unsupported_path
    UNIQUE (product_id, standard_path);

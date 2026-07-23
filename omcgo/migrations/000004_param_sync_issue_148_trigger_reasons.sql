-- +goose Up

ALTER TABLE parameter_sync_requests
    DROP CONSTRAINT parameter_sync_requests_trigger_reason_chk,
    ADD CONSTRAINT parameter_sync_requests_trigger_reason_chk
        CHECK (trigger_reason::text = ANY (ARRAY[
            'bootstrap', 'model_upload', 'device_online', 'firmware_changed',
            'periodic', 'manual', 'config_pull', 'license', 'spv_readback',
            'add_object_readback', 'inform_period_probe', 'device_registered',
            'omc_upgrade'
        ]::text[]));

ALTER TABLE parameter_sync_runs
    DROP CONSTRAINT parameter_sync_runs_trigger_reason_chk,
    ADD CONSTRAINT parameter_sync_runs_trigger_reason_chk
        CHECK (trigger_reason::text = ANY (ARRAY[
            'bootstrap', 'model_upload', 'device_online', 'firmware_changed',
            'periodic', 'manual', 'config_pull', 'license', 'spv_readback',
            'add_object_readback', 'inform_period_probe', 'device_registered',
            'omc_upgrade'
        ]::text[]));

-- +goose Down

-- The expanded CHECK constraints are backward-compatible with older code.
-- Keep them on rollback so existing device_registered / omc_upgrade history
-- remains valid and can still be updated without deleting operational data.
SELECT 1;

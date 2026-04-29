-- +goose Up
-- T-0021 / R-101 (rollback half): rollback metadata for audit + canary auto-rollback link.
-- New columns are nullable / defaulted so the legacy RollbackDevices flow stays compatible.
--
-- rollback_reason          : free-form audit text ("threshold exceeded at 1%% stage")
-- rollback_source          : enum, taggable from API + canary monitor; default 'manual'
-- rollback_target_firmware_id : optional override; when set, sub_tasks dest_version
--                            comes from this firmware instead of devices.firmware_version
-- rollback_on_failure      : opt-in flag persisted on the *upgrade* task so the canary
--                            monitor can decide whether to call RollbackDevices when
--                            a stage exceeds its failure threshold (default false; T-0018
--                            invariant "不擅自回滚" preserved).

ALTER TABLE upgrade_tasks
    ADD COLUMN IF NOT EXISTS rollback_reason TEXT,
    ADD COLUMN IF NOT EXISTS rollback_source VARCHAR(32) NOT NULL DEFAULT 'manual'
        CHECK (rollback_source IN ('manual', 'canary_failure', 'compatibility', 'scheduled')),
    ADD COLUMN IF NOT EXISTS rollback_target_firmware_id UUID
        REFERENCES firmware_versions(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS rollback_on_failure BOOLEAN NOT NULL DEFAULT false;

CREATE INDEX IF NOT EXISTS idx_upgrade_tasks_rollback_source
    ON upgrade_tasks(rollback_source)
    WHERE task_type = 2;  -- TaskTypeRollback

-- +goose Down
DROP INDEX IF EXISTS idx_upgrade_tasks_rollback_source;
ALTER TABLE upgrade_tasks
    DROP COLUMN IF EXISTS rollback_on_failure,
    DROP COLUMN IF EXISTS rollback_target_firmware_id,
    DROP COLUMN IF EXISTS rollback_source,
    DROP COLUMN IF EXISTS rollback_reason;

-- +goose Up
-- T-0018 / R-101: Software canary upgrade strategy
-- Adds canary fields to upgrade_tasks; existing rows default to strategy='full'
-- (backwards compatible - BatchUpgrade with no Strategy stays on the legacy path).

ALTER TABLE upgrade_tasks
    ADD COLUMN IF NOT EXISTS strategy VARCHAR(16) NOT NULL DEFAULT 'full'
        CHECK (strategy IN ('full', 'canary')),
    ADD COLUMN IF NOT EXISTS canary_stages JSONB,
    ADD COLUMN IF NOT EXISTS current_stage INT NOT NULL DEFAULT 0
        CHECK (current_stage >= 0 AND current_stage <= 100),
    ADD COLUMN IF NOT EXISTS stage_status VARCHAR(20) NOT NULL DEFAULT 'pending'
        CHECK (stage_status IN ('pending','running','paused','aborted','completed')),
    ADD COLUMN IF NOT EXISTS stage_history JSONB NOT NULL DEFAULT '[]'::jsonb,
    ADD COLUMN IF NOT EXISTS auto_advance BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS auto_advance_minutes INT NOT NULL DEFAULT 0
        CHECK (auto_advance_minutes >= 0 AND auto_advance_minutes <= 1440);

CREATE INDEX IF NOT EXISTS idx_upgrade_tasks_canary_active
    ON upgrade_tasks(strategy, stage_status)
    WHERE strategy = 'canary' AND stage_status IN ('running', 'paused');

-- +goose Down
DROP INDEX IF EXISTS idx_upgrade_tasks_canary_active;
ALTER TABLE upgrade_tasks
    DROP COLUMN IF EXISTS auto_advance_minutes,
    DROP COLUMN IF EXISTS auto_advance,
    DROP COLUMN IF EXISTS stage_history,
    DROP COLUMN IF EXISTS stage_status,
    DROP COLUMN IF EXISTS current_stage,
    DROP COLUMN IF EXISTS canary_stages,
    DROP COLUMN IF EXISTS strategy;

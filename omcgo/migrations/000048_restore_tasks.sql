-- T-0072: backup restore tracking — one row per restore request, regardless
-- of how many target devices the request fans out to. Per-device progress is
-- joined from `device_tasks` (method=Download) at query time; no FK because
-- device_tasks is a partition table (CLAUDE.md §5.5.3).

-- +goose Up
CREATE TABLE IF NOT EXISTS restore_tasks (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_bucket       VARCHAR(64) NOT NULL,
    source_object_path  TEXT        NOT NULL,
    target_device_sns   JSONB       NOT NULL DEFAULT '[]'::jsonb,
    status              VARCHAR(16) NOT NULL DEFAULT 'pending'
                        CHECK (status IN ('pending','running','completed','failed','cancelled')),
    progress            INT         NOT NULL DEFAULT 0
                        CHECK (progress >= 0 AND progress <= 100),
    error_message       TEXT,
    started_at          TIMESTAMPTZ,
    completed_at        TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by          VARCHAR(64)
);

CREATE INDEX IF NOT EXISTS idx_restore_tasks_status     ON restore_tasks(status);
CREATE INDEX IF NOT EXISTS idx_restore_tasks_created_at ON restore_tasks(created_at DESC);

-- Reuse the shared updated_at trigger function created in 000001.
CREATE TRIGGER trg_restore_tasks_updated
BEFORE UPDATE ON restore_tasks
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- +goose Down
DROP TRIGGER IF EXISTS trg_restore_tasks_updated ON restore_tasks;
DROP INDEX IF EXISTS idx_restore_tasks_created_at;
DROP INDEX IF EXISTS idx_restore_tasks_status;
DROP TABLE IF EXISTS restore_tasks;

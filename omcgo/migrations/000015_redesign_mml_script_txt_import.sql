-- TXT import redesign: this migration deliberately and irreversibly removes
-- historical MML scripts/tasks. It preserves all non-MML device_tasks.
-- +goose Up
DELETE FROM device_tasks WHERE source = 'mml';
DELETE FROM mml_tasks;
DELETE FROM mml_scripts;

ALTER TABLE mml_scripts
    ADD COLUMN import_session_id uuid NOT NULL,
    ADD COLUMN original_filename text NOT NULL DEFAULT '',
    ADD COLUMN content_sha256 text NOT NULL DEFAULT '',
    ADD COLUMN validation_version text NOT NULL DEFAULT '',
    ADD COLUMN validated_at timestamptz,
    ADD COLUMN plan_items jsonb NOT NULL DEFAULT '[]'::jsonb,
    ADD COLUMN validation_summary jsonb NOT NULL DEFAULT '{}'::jsonb;

ALTER TABLE mml_tasks
    ADD COLUMN script_content_sha256 text NOT NULL DEFAULT '',
    ADD COLUMN script_validation_version text NOT NULL DEFAULT '';

CREATE INDEX idx_mml_scripts_content_sha256 ON mml_scripts(content_sha256);
CREATE UNIQUE INDEX uq_mml_scripts_import_session_id ON mml_scripts(import_session_id);
CREATE INDEX idx_mml_scripts_plan_items_gin
    ON mml_scripts USING gin (plan_items jsonb_path_ops);

-- +goose Down
-- Historical MML data deleted by Up cannot be restored.
DROP INDEX idx_mml_scripts_plan_items_gin;
DROP INDEX uq_mml_scripts_import_session_id;
DROP INDEX idx_mml_scripts_content_sha256;

ALTER TABLE mml_tasks
    DROP COLUMN script_validation_version,
    DROP COLUMN script_content_sha256;

ALTER TABLE mml_scripts
    DROP COLUMN validation_summary,
    DROP COLUMN plan_items,
    DROP COLUMN validated_at,
    DROP COLUMN validation_version,
    DROP COLUMN content_sha256,
    DROP COLUMN original_filename,
    DROP COLUMN import_session_id;

-- +goose Up
-- MML templates: add category_group for custom command directory organization
ALTER TABLE mml_templates ADD COLUMN IF NOT EXISTS category_group VARCHAR(50);
CREATE INDEX IF NOT EXISTS idx_mml_templates_scope_group
    ON mml_templates(template_scope, category_group);

-- MML tasks: add executor for SSE push targeting
ALTER TABLE mml_tasks ADD COLUMN IF NOT EXISTS executor VARCHAR(100);

-- +goose Down
DROP INDEX IF EXISTS idx_mml_templates_scope_group;
ALTER TABLE mml_templates DROP COLUMN IF EXISTS category_group;
ALTER TABLE mml_tasks DROP COLUMN IF EXISTS executor;

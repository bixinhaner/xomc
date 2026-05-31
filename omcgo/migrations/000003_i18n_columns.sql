-- +goose Up
-- +goose StatementBegin
-- 给 4 张用户可见表加 JSONB i18n 列,沿用 menus.name_i18n / mml_command_groups.name_i18n
-- 的 {"zh-CN": "...", "en-US": "..."} 形态。读取兜底顺序: i18n[locale] -> i18n['zh-CN'] -> legacy 列。

ALTER TABLE device_groups ADD COLUMN IF NOT EXISTS name_i18n JSONB NOT NULL DEFAULT '{}'::jsonb;
ALTER TABLE device_groups ADD COLUMN IF NOT EXISTS description_i18n JSONB NOT NULL DEFAULT '{}'::jsonb;
ALTER TABLE device_groups ADD COLUMN IF NOT EXISTS remark_i18n JSONB NOT NULL DEFAULT '{}'::jsonb;

ALTER TABLE sys_dictionaries ADD COLUMN IF NOT EXISTS name_i18n JSONB NOT NULL DEFAULT '{}'::jsonb;
ALTER TABLE sys_dictionaries ADD COLUMN IF NOT EXISTS description_i18n JSONB NOT NULL DEFAULT '{}'::jsonb;

ALTER TABLE sys_dictionary_details ADD COLUMN IF NOT EXISTS label_i18n JSONB NOT NULL DEFAULT '{}'::jsonb;

ALTER TABLE sys_configs ADD COLUMN IF NOT EXISTS description_i18n JSONB NOT NULL DEFAULT '{}'::jsonb;

-- 历史数据回填: 把现有单语言列内容作为 zh-CN 和 en-US 兜底,后续 seed 000002 会覆盖英文翻译。
UPDATE device_groups
SET name_i18n = jsonb_build_object('zh-CN', name, 'en-US', name)
WHERE name_i18n = '{}'::jsonb AND name IS NOT NULL;

UPDATE device_groups
SET description_i18n = jsonb_build_object('zh-CN', description, 'en-US', description)
WHERE description_i18n = '{}'::jsonb AND description IS NOT NULL AND description <> '';

UPDATE device_groups
SET remark_i18n = jsonb_build_object('zh-CN', remark, 'en-US', remark)
WHERE remark_i18n = '{}'::jsonb AND remark IS NOT NULL AND remark <> '';

UPDATE sys_dictionaries
SET name_i18n = jsonb_build_object('zh-CN', name, 'en-US', name)
WHERE name_i18n = '{}'::jsonb AND name IS NOT NULL;

UPDATE sys_dictionaries
SET description_i18n = jsonb_build_object('zh-CN', description, 'en-US', description)
WHERE description_i18n = '{}'::jsonb AND description IS NOT NULL AND description <> '';

UPDATE sys_dictionary_details
SET label_i18n = jsonb_build_object('zh-CN', label, 'en-US', label)
WHERE label_i18n = '{}'::jsonb AND label IS NOT NULL;

UPDATE sys_configs
SET description_i18n = jsonb_build_object('zh-CN', description, 'en-US', description)
WHERE description_i18n = '{}'::jsonb AND description IS NOT NULL AND description <> '';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE device_groups DROP COLUMN IF EXISTS name_i18n;
ALTER TABLE device_groups DROP COLUMN IF EXISTS description_i18n;
ALTER TABLE device_groups DROP COLUMN IF EXISTS remark_i18n;
ALTER TABLE sys_dictionaries DROP COLUMN IF EXISTS name_i18n;
ALTER TABLE sys_dictionaries DROP COLUMN IF EXISTS description_i18n;
ALTER TABLE sys_dictionary_details DROP COLUMN IF EXISTS label_i18n;
ALTER TABLE sys_configs DROP COLUMN IF EXISTS description_i18n;
-- +goose StatementEnd

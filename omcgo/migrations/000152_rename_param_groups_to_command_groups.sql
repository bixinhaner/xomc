-- +goose Up
ALTER TABLE mml_param_groups RENAME TO mml_command_groups;

-- 索引名跟随更新
ALTER INDEX IF EXISTS mml_param_groups_pkey RENAME TO mml_command_groups_pkey;

-- 清空数据（CASCADE 删除关联的 mml_commands + mml_command_sub_fields）
TRUNCATE mml_command_groups CASCADE;

-- +goose Down
ALTER TABLE mml_command_groups RENAME TO mml_param_groups;
ALTER INDEX IF EXISTS mml_command_groups_pkey RENAME TO mml_param_groups_pkey;

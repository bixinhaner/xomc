-- +goose Up
-- ============================================================
-- 000110_mml_commands_platform_tags.sql
-- MML 老 catalog 导入前置：mml_commands 增加 platform_tags 列
--
-- 背景：T-0123 完成 schema 重建后，准备从 docs/files/db/mml/{small_cell_param.sql,
-- small_cell_param_group.sql} 导入老 OMC 1919 group + 7226 param 的业务分类
-- 与聚合命令清单。老表里有 mobile_support / broadband_support / platform_support
-- 三个标记，需要在 mml_commands 上有一个对应列保存，前端按当前皮肤/设备类型过滤
-- 命令树叶子。
--
-- 设计决策（B1）：单列 JSONB platform_tags，结构形如：
--   {"mobile": true, "broadband": true, "platform_codes": ["1"]}
--
-- 关于 confirm 文案（C1 决策）：复用既有 confirm_msg_i18n（000090 已有），
-- 不新增 confirm_text_i18n。
-- ============================================================

ALTER TABLE mml_commands
    ADD COLUMN IF NOT EXISTS platform_tags JSONB NOT NULL DEFAULT '{}';

COMMENT ON COLUMN mml_commands.platform_tags IS
    '平台支持标记（JSONB），形如 {"mobile":true,"broadband":true,"platform_codes":["1"]}；'
    '由老 OMC small_cell_param_group 的 mobile_support/broadband_support/platform_support '
    '三列合并而来。前端按当前皮肤/设备类型过滤命令树叶子。';

CREATE INDEX IF NOT EXISTS idx_mml_commands_platform_tags_gin
    ON mml_commands USING GIN (platform_tags);


-- +goose Down
DROP INDEX IF EXISTS idx_mml_commands_platform_tags_gin;
ALTER TABLE mml_commands DROP COLUMN IF EXISTS platform_tags;

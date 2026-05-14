-- +goose Up
-- ============================================================================
-- 000094: 放宽 mml_param_groups.group_code 与 mml_commands.command_code 列宽
-- (原编号 000092，与 c1f95f66 feat(...) config_templates_auto_dispatch 并行 PR
--  撞号；按 CLAUDE.md §5.5 "后合并者改大版本号"规则上调；093 已被
--  000093_mml_param_versions_content_hash 占用，故跳到 094)
--
-- Sprint A 重建后 group_code 由完整 path 转 UPPER_SNAKE_CASE 派生（例：
-- "DEVICE_SERVICES_FAPSERVICE_CELLCONFIG_NR_RAN_PHY_BWP_BWPDL_PDCCH_..."），
-- 实测最长 107 字符；command_code 为 "LST_<group_code>" / "MOD_<group_code>"
-- 等前缀拼接，会更长（>111）。VARCHAR(100) 直接溢出，Loader UPSERT 时
-- 报 "value too long for type character varying(100) (SQLSTATE 22001)"。
--
-- 修复：把两列放宽到 VARCHAR(255)。standard-model.xml 实际深度还有空间
-- 演进，留 ~2.3x 的 buffer。同步把 *_name 列也放宽到 500 防御。
-- ============================================================================

-- mml_param_groups
ALTER TABLE mml_param_groups
    ALTER COLUMN group_code    TYPE VARCHAR(255),
    ALTER COLUMN group_name_zh TYPE VARCHAR(500),
    ALTER COLUMN group_name_en TYPE VARCHAR(500);

-- mml_commands
ALTER TABLE mml_commands
    ALTER COLUMN command_code TYPE VARCHAR(255),
    ALTER COLUMN command_name TYPE VARCHAR(500);

-- +goose Down
-- 注意：缩窄列宽可能在已写入超长行时失败；Down 段保留语义对称仅供回滚
-- 演练，生产数据可能需要先 TRUNCATE。
ALTER TABLE mml_param_groups
    ALTER COLUMN group_code    TYPE VARCHAR(100),
    ALTER COLUMN group_name_zh TYPE VARCHAR(200),
    ALTER COLUMN group_name_en TYPE VARCHAR(200);

ALTER TABLE mml_commands
    ALTER COLUMN command_code TYPE VARCHAR(100),
    ALTER COLUMN command_name TYPE VARCHAR(200);

-- 000153_standard_commands.sql
--
-- 改号说明（2026-05-22）：原 000151 与 eventlog 分支（commit 8c148c81）的
--   000151_event_logs.sql 撞号 → 按 CLAUDE.md §5.5「后合入者重命名为更大版本号」
--   规则改为 000153（000152 已被 mml_param_groups → mml_command_groups rename 占用）。
--
-- 任务 #5 修复说明（与 000058 冲突解决）：
--   原始 000151 试图重建 standard_params 表，与 000058 创建的同名表（UUID PK，
--   被 mml_command_sub_fields.standard_path_id FK in 000113 引用）冲突。
--
--   分析：
--     1. standard_params 已由 000058 创建（standard_path UNIQUE / entry_type / access /
--        data_type / change_applies / min_value / max_value）。
--     2. mmlstandardloader/seed_importer.go 仅写入 000058 schema 列（不依赖
--        version_code / name_cn / constraints / group_code / command_name）。
--     3. ParamModel Translator (parammodel/translator.go) 通过 param_mappings 表
--        实现 standardPath ↔ privatePath 翻译，不直接读 standard_params。
--     4. 现有 mml seed (000111 / 000126) JOIN standard_params 时只用 standard_path /
--        entry_type / access 三列。
--   结论：standard_params 不需新增列；删除冲突的 CREATE TABLE。
--
-- 本迁移仅保留 standard_commands 这张全新表（南向数据模型命令权威表，
-- 用于未来按 version_code 缓存命令树元数据，不与现有 schema 冲突）。

-- +goose Up

CREATE TABLE IF NOT EXISTS standard_commands (
    id              BIGSERIAL PRIMARY KEY,
    version_code    VARCHAR(50) NOT NULL REFERENCES mml_param_versions(version_code) ON DELETE CASCADE,
    group_code      TEXT NOT NULL,                    -- 分组编码：SA/SB/.../SR
    group_name      TEXT NOT NULL,                    -- 纯中文分组名（如"设备信息参数管理"）
    object_path     TEXT NOT NULL,                    -- 对象路径，如 Device.DeviceInfo.
    command_name    TEXT NOT NULL,                    -- 命令中文名
    has_rw_params   BOOLEAN NOT NULL DEFAULT false,   -- 是否有RW参数（决定MOD是否生成）
    has_instance    BOOLEAN NOT NULL DEFAULT false,   -- 对象路径是否含{i}
    is_creatable    BOOLEAN NOT NULL DEFAULT true,    -- 是否可创建实例（非白名单对象）
    param_count     INTEGER NOT NULL DEFAULT 0,       -- 关联参数总数
    rw_param_count  INTEGER NOT NULL DEFAULT 0,       -- RW参数数量
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(version_code, object_path)
);

CREATE INDEX IF NOT EXISTS idx_standard_commands_group ON standard_commands(version_code, group_code);

COMMENT ON TABLE  standard_commands IS '南向数据模型标准命令权威表，按 version_code 存储 MML 命令树的命令元数据';
COMMENT ON COLUMN standard_commands.has_rw_params IS '为 true 时才生成 MOD 操作叶子';
COMMENT ON COLUMN standard_commands.is_creatable IS '为 false 表示在非可创建白名单中，不生成 ADD/RMV';

-- +goose Down

DROP INDEX IF EXISTS idx_standard_commands_group;
DROP TABLE IF EXISTS standard_commands;

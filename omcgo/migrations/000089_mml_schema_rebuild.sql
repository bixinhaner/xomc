-- +goose Up
-- ============================================================
-- 000089_mml_schema_rebuild.sql
-- MML 表结构重建 — 基于 standard-model.xml 的命令字典化
--
-- 设计依据：docs/design/mml-rebuild-plan-20260513.md v3.1 APPROVED
--
-- 改动总览：
--   1. DROP TABLE mml_audit_log（迁 ops_audit_log）
--   2. DROP TABLE mml_command_params_rel（target_paths JSONB GIN 替代）
--   3. mml_param_versions: 改 UUID 主键 + 硬删 24 老 versions
--   4. mml_param_groups: 删 15 列（parent_id/level/4 flag/cell_*/confirm_*/...）+ 加 name_i18n JSONB
--   5. mml_params: 删 16 列（4 flag/platform_*/title_*/explanation_*/...）+ 加 name_i18n / explanation_i18n JSONB
--   6. mml_commands: 删 4 字段 + 加 target_paths / target_object / rpc_method 调整
--   7. mml_scripts: 删 4 个单次执行字段（保留 status 表 active/disabled）
--   8. TRUNCATE 旧数据，等 Loader 启动期填充
-- ============================================================

-- +goose StatementBegin
DO $$ BEGIN

-- ============================================================
-- Step 1: DROP 不再需要的表
-- ============================================================
DROP TABLE IF EXISTS mml_audit_log CASCADE;
DROP TABLE IF EXISTS mml_command_params_rel CASCADE;

-- ============================================================
-- Step 2: mml_scripts 删单次执行字段（保留 status 为模板态）
--   Q5 决议：保留 status='active'/'disabled' 模板启用态；
--           删 start_time/end_time/type/progress/result（单次执行字段，归 mml_tasks）
-- ============================================================
ALTER TABLE mml_scripts DROP COLUMN IF EXISTS start_time;
ALTER TABLE mml_scripts DROP COLUMN IF EXISTS end_time;
ALTER TABLE mml_scripts DROP COLUMN IF EXISTS type;
ALTER TABLE mml_scripts DROP COLUMN IF EXISTS progress;
ALTER TABLE mml_scripts DROP COLUMN IF EXISTS result;
DROP INDEX IF EXISTS idx_mml_scripts_type;

-- ============================================================
-- Step 3: TRUNCATE 字典层数据（待 Loader 重填）
--   先 TRUNCATE 子表（rel） → 再 TRUNCATE 主表（commands/groups/params）
--   mml_command_params_rel 已 DROP，跳过
-- ============================================================
TRUNCATE mml_group_param_rel CASCADE;
TRUNCATE mml_commands       CASCADE;
TRUNCATE mml_param_groups   CASCADE;
TRUNCATE mml_params         CASCADE;

-- ============================================================
-- Step 4: mml_param_versions 改 UUID 主键 + 硬删 24 老 versions（Q7）
--   保留表结构；增 id UUID 列做新主键；version_code 改 UNIQUE
--   注意：mml_param_groups / mml_params 用 param_version VARCHAR(50) 外键引用
--          我们保留 param_version FK（仍按 code 引用 STANDARD），底层 PK 改 UUID 不影响
-- ============================================================

-- 先删依赖的外键
ALTER TABLE mml_param_groups DROP CONSTRAINT IF EXISTS mml_param_groups_param_version_fkey;
ALTER TABLE mml_params       DROP CONSTRAINT IF EXISTS mml_params_param_version_fkey;

-- TRUNCATE 老 versions 数据
TRUNCATE mml_param_versions CASCADE;

-- 改 PK：先删旧 PK，加 id UUID 列做新 PK，version_code 改 UNIQUE
ALTER TABLE mml_param_versions DROP CONSTRAINT IF EXISTS mml_param_versions_pkey;
ALTER TABLE mml_param_versions ADD COLUMN IF NOT EXISTS id UUID NOT NULL DEFAULT gen_random_uuid();
ALTER TABLE mml_param_versions ADD PRIMARY KEY (id);
ALTER TABLE mml_param_versions ADD CONSTRAINT uq_mml_param_versions_code UNIQUE (version_code);

-- 加 source 列标识来源（standard / product）
ALTER TABLE mml_param_versions ADD COLUMN IF NOT EXISTS source VARCHAR(20) NOT NULL DEFAULT 'standard';

-- 删不再需要的字段
ALTER TABLE mml_param_versions DROP COLUMN IF EXISTS product_models;
ALTER TABLE mml_param_versions DROP COLUMN IF EXISTS software_versions;
ALTER TABLE mml_param_versions DROP COLUMN IF EXISTS release_date;
ALTER TABLE mml_param_versions DROP COLUMN IF EXISTS group_count;
ALTER TABLE mml_param_versions DROP COLUMN IF EXISTS param_count;

-- 恢复 FK 关系
ALTER TABLE mml_param_groups
    ADD CONSTRAINT mml_param_groups_param_version_fkey
    FOREIGN KEY (param_version) REFERENCES mml_param_versions(version_code) ON DELETE RESTRICT;
ALTER TABLE mml_params
    ADD CONSTRAINT mml_params_param_version_fkey
    FOREIGN KEY (param_version) REFERENCES mml_param_versions(version_code) ON DELETE RESTRICT;

-- ============================================================
-- Step 5: mml_param_groups schema 瘦身（25 → 10 列）
-- ============================================================
-- 删 parent_id（用 path ltree 替代）+ level（path 派生）
ALTER TABLE mml_param_groups DROP COLUMN IF EXISTS parent_id;
ALTER TABLE mml_param_groups DROP COLUMN IF EXISTS level;
DROP INDEX IF EXISTS idx_mml_param_groups_parent;
-- 删 4 操作 flag（操作权限是命令属性，不是 group 属性）
ALTER TABLE mml_param_groups DROP COLUMN IF EXISTS is_listable;
ALTER TABLE mml_param_groups DROP COLUMN IF EXISTS is_modifiable;
ALTER TABLE mml_param_groups DROP COLUMN IF EXISTS is_addable;
ALTER TABLE mml_param_groups DROP COLUMN IF EXISTS is_removable;
-- 删 ADD/RMV 路径字段（移到 mml_commands.target_object）
ALTER TABLE mml_param_groups DROP COLUMN IF EXISTS add_object_path;
ALTER TABLE mml_param_groups DROP COLUMN IF EXISTS delete_object_path;
-- 删 product/platform 散布字段
ALTER TABLE mml_param_groups DROP COLUMN IF EXISTS platform_support;
ALTER TABLE mml_param_groups DROP COLUMN IF EXISTS mobile_support;
ALTER TABLE mml_param_groups DROP COLUMN IF EXISTS broadband_support;
-- 删历史遗留字段
ALTER TABLE mml_param_groups DROP COLUMN IF EXISTS cell_number;
ALTER TABLE mml_param_groups DROP COLUMN IF EXISTS cell_index_location;
-- 删二次确认字段（搬到 mml_commands）
ALTER TABLE mml_param_groups DROP COLUMN IF EXISTS require_second_confirm;
ALTER TABLE mml_param_groups DROP COLUMN IF EXISTS confirm_message_zh;
ALTER TABLE mml_param_groups DROP COLUMN IF EXISTS confirm_message_en;

-- 合并 i18n：group_name_zh / group_name_en → JSONB name_i18n
ALTER TABLE mml_param_groups ADD COLUMN IF NOT EXISTS name_i18n JSONB NOT NULL DEFAULT '{}';
-- 老字段保留只读兼容（FE 切换后再删）：暂时保留 group_name_zh / group_name_en

-- ============================================================
-- Step 6: mml_params schema 瘦身（31 → 15 列）
-- ============================================================
-- 删 4 操作 flag（同 group，操作是命令属性）
ALTER TABLE mml_params DROP COLUMN IF EXISTS is_listable;
ALTER TABLE mml_params DROP COLUMN IF EXISTS is_modifiable;
ALTER TABLE mml_params DROP COLUMN IF EXISTS is_addable;
ALTER TABLE mml_params DROP COLUMN IF EXISTS is_removable;
-- 删 dynamic flag（用不到）
ALTER TABLE mml_params DROP COLUMN IF EXISTS is_dynamic;
-- 删 product/platform 散布字段
ALTER TABLE mml_params DROP COLUMN IF EXISTS platform_support;
ALTER TABLE mml_params DROP COLUMN IF EXISTS mobile_support;
ALTER TABLE mml_params DROP COLUMN IF EXISTS broadband_support;
ALTER TABLE mml_params DROP COLUMN IF EXISTS software_version;
-- 删二次确认字段（搬到 mml_commands）
ALTER TABLE mml_params DROP COLUMN IF EXISTS require_second_confirm;
ALTER TABLE mml_params DROP COLUMN IF EXISTS confirm_message_zh;
ALTER TABLE mml_params DROP COLUMN IF EXISTS confirm_message_en;
-- 删 memo（用 explanation 即可）
ALTER TABLE mml_params DROP COLUMN IF EXISTS memo;
-- 合并 i18n
ALTER TABLE mml_params ADD COLUMN IF NOT EXISTS name_i18n JSONB NOT NULL DEFAULT '{}';
ALTER TABLE mml_params ADD COLUMN IF NOT EXISTS explanation_i18n JSONB NOT NULL DEFAULT '{}';
-- 老字段保留兼容（param_name_zh/en, title_zh/en, explanation_zh/en）— FE 切换后再删

-- ============================================================
-- Step 7: mml_commands schema 重构
--   删 product_types / param_template / param_paths / supported_operations 4 字段
--   加 target_paths JSONB / target_object / category VARCHAR(10) / command_name_i18n / require_confirm / confirm_msg_i18n
--   保留 rpc_method（单 RPC，v3 已撤回 rpc_steps）
-- ============================================================
-- 老字段：drop
ALTER TABLE mml_commands DROP COLUMN IF EXISTS product_types;
ALTER TABLE mml_commands DROP COLUMN IF EXISTS param_template;
ALTER TABLE mml_commands DROP COLUMN IF EXISTS param_paths;
ALTER TABLE mml_commands DROP COLUMN IF EXISTS supported_operations;
DROP INDEX IF EXISTS idx_mml_commands_product_types_gin;
DROP INDEX IF EXISTS idx_mml_commands_param_paths_gin;

-- 加新字段
ALTER TABLE mml_commands ADD COLUMN IF NOT EXISTS target_paths JSONB NOT NULL DEFAULT '[]';
ALTER TABLE mml_commands ADD COLUMN IF NOT EXISTS target_object VARCHAR(500);
ALTER TABLE mml_commands ADD COLUMN IF NOT EXISTS group_id UUID;
ALTER TABLE mml_commands ADD COLUMN IF NOT EXISTS command_name_i18n JSONB NOT NULL DEFAULT '{}';
ALTER TABLE mml_commands ADD COLUMN IF NOT EXISTS require_confirm BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE mml_commands ADD COLUMN IF NOT EXISTS confirm_msg_i18n JSONB NOT NULL DEFAULT '{}';

-- category 改为 VARCHAR(10) 存 1-7 既有 7 类 code（Q1）
-- 原 category VARCHAR(50) 兼容保留；新数据写 '1'..'7' 字符串
-- 若想强约束加 CHECK：CHECK (category IN ('1','2','3','4','5','6','7')) —
-- 但兼容老 25 条 seed/000006 数据；本次先放宽，Loader 写新数据守约。

-- operation_type 已在 seed/000004 加；保证存在
ALTER TABLE mml_commands ALTER COLUMN operation_type SET NOT NULL;

-- 加 group_id FK + 索引
ALTER TABLE mml_commands DROP CONSTRAINT IF EXISTS mml_commands_group_id_fkey;
ALTER TABLE mml_commands
    ADD CONSTRAINT mml_commands_group_id_fkey
    FOREIGN KEY (group_id) REFERENCES mml_param_groups(id) ON DELETE SET NULL;

-- GIN 索引：替代删除的 mml_command_params_rel 的反查能力
CREATE INDEX IF NOT EXISTS idx_mml_commands_target_paths_gin ON mml_commands USING GIN (target_paths);
CREATE INDEX IF NOT EXISTS idx_mml_commands_category ON mml_commands(category);
CREATE INDEX IF NOT EXISTS idx_mml_commands_group_id ON mml_commands(group_id);
CREATE INDEX IF NOT EXISTS idx_mml_commands_operation_type ON mml_commands(operation_type);

END $$;
-- +goose StatementEnd


-- +goose Down
-- ============================================================
-- 回滚：仅恢复 schema，数据无法逆转（24 老 versions / mml_audit_log / mml_command_params_rel 已删）
-- 警告：down 后 mml_commands 数据丢失；需重跑 seed/000006 才能恢复老 25 行
-- ============================================================

-- +goose StatementBegin
DO $$ BEGIN

-- 删新加字段
ALTER TABLE mml_commands DROP COLUMN IF EXISTS target_paths;
ALTER TABLE mml_commands DROP COLUMN IF EXISTS target_object;
ALTER TABLE mml_commands DROP COLUMN IF EXISTS group_id;
ALTER TABLE mml_commands DROP COLUMN IF EXISTS command_name_i18n;
ALTER TABLE mml_commands DROP COLUMN IF EXISTS require_confirm;
ALTER TABLE mml_commands DROP COLUMN IF EXISTS confirm_msg_i18n;

-- 恢复 mml_commands 老字段（空表，靠 seed 重填）
ALTER TABLE mml_commands ADD COLUMN IF NOT EXISTS product_types JSONB DEFAULT '[]';
ALTER TABLE mml_commands ADD COLUMN IF NOT EXISTS param_template JSONB;
ALTER TABLE mml_commands ADD COLUMN IF NOT EXISTS param_paths JSONB DEFAULT '[]';
ALTER TABLE mml_commands ADD COLUMN IF NOT EXISTS supported_operations JSONB DEFAULT '["LST"]';

-- 恢复 mml_param_groups 老字段
ALTER TABLE mml_param_groups ADD COLUMN IF NOT EXISTS parent_id UUID REFERENCES mml_param_groups(id) ON DELETE CASCADE;
ALTER TABLE mml_param_groups ADD COLUMN IF NOT EXISTS level INT NOT NULL DEFAULT 0;
ALTER TABLE mml_param_groups ADD COLUMN IF NOT EXISTS is_listable BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE mml_param_groups ADD COLUMN IF NOT EXISTS is_modifiable BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE mml_param_groups ADD COLUMN IF NOT EXISTS is_addable BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE mml_param_groups ADD COLUMN IF NOT EXISTS is_removable BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE mml_param_groups ADD COLUMN IF NOT EXISTS add_object_path VARCHAR(500);
ALTER TABLE mml_param_groups ADD COLUMN IF NOT EXISTS delete_object_path VARCHAR(500);
ALTER TABLE mml_param_groups ADD COLUMN IF NOT EXISTS platform_support VARCHAR(10)[];
ALTER TABLE mml_param_groups ADD COLUMN IF NOT EXISTS mobile_support BOOLEAN NOT NULL DEFAULT true;
ALTER TABLE mml_param_groups ADD COLUMN IF NOT EXISTS broadband_support BOOLEAN NOT NULL DEFAULT true;
ALTER TABLE mml_param_groups ADD COLUMN IF NOT EXISTS cell_number INT NOT NULL DEFAULT 1;
ALTER TABLE mml_param_groups ADD COLUMN IF NOT EXISTS cell_index_location INT NOT NULL DEFAULT 0;
ALTER TABLE mml_param_groups ADD COLUMN IF NOT EXISTS require_second_confirm BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE mml_param_groups ADD COLUMN IF NOT EXISTS confirm_message_zh TEXT;
ALTER TABLE mml_param_groups ADD COLUMN IF NOT EXISTS confirm_message_en TEXT;
ALTER TABLE mml_param_groups DROP COLUMN IF EXISTS name_i18n;

-- 恢复 mml_params 老字段
ALTER TABLE mml_params ADD COLUMN IF NOT EXISTS is_listable BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE mml_params ADD COLUMN IF NOT EXISTS is_modifiable BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE mml_params ADD COLUMN IF NOT EXISTS is_addable BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE mml_params ADD COLUMN IF NOT EXISTS is_removable BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE mml_params ADD COLUMN IF NOT EXISTS is_dynamic BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE mml_params ADD COLUMN IF NOT EXISTS platform_support VARCHAR(10)[];
ALTER TABLE mml_params ADD COLUMN IF NOT EXISTS mobile_support BOOLEAN NOT NULL DEFAULT true;
ALTER TABLE mml_params ADD COLUMN IF NOT EXISTS broadband_support BOOLEAN NOT NULL DEFAULT true;
ALTER TABLE mml_params ADD COLUMN IF NOT EXISTS software_version VARCHAR(100);
ALTER TABLE mml_params ADD COLUMN IF NOT EXISTS require_second_confirm BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE mml_params ADD COLUMN IF NOT EXISTS confirm_message_zh TEXT;
ALTER TABLE mml_params ADD COLUMN IF NOT EXISTS confirm_message_en TEXT;
ALTER TABLE mml_params ADD COLUMN IF NOT EXISTS memo TEXT;
ALTER TABLE mml_params DROP COLUMN IF EXISTS name_i18n;
ALTER TABLE mml_params DROP COLUMN IF EXISTS explanation_i18n;

-- 恢复 mml_param_versions schema
ALTER TABLE mml_param_versions DROP CONSTRAINT IF EXISTS uq_mml_param_versions_code;
ALTER TABLE mml_param_versions DROP CONSTRAINT IF EXISTS mml_param_versions_pkey;
ALTER TABLE mml_param_versions DROP COLUMN IF EXISTS id;
ALTER TABLE mml_param_versions DROP COLUMN IF EXISTS source;
ALTER TABLE mml_param_versions ADD COLUMN IF NOT EXISTS product_models VARCHAR(200)[];
ALTER TABLE mml_param_versions ADD COLUMN IF NOT EXISTS software_versions VARCHAR(100)[];
ALTER TABLE mml_param_versions ADD COLUMN IF NOT EXISTS release_date DATE;
ALTER TABLE mml_param_versions ADD COLUMN IF NOT EXISTS group_count INT NOT NULL DEFAULT 0;
ALTER TABLE mml_param_versions ADD COLUMN IF NOT EXISTS param_count INT NOT NULL DEFAULT 0;
ALTER TABLE mml_param_versions ADD PRIMARY KEY (version_code);

-- 恢复 mml_scripts 单次执行字段
ALTER TABLE mml_scripts ADD COLUMN IF NOT EXISTS start_time TIMESTAMPTZ;
ALTER TABLE mml_scripts ADD COLUMN IF NOT EXISTS end_time TIMESTAMPTZ;
ALTER TABLE mml_scripts ADD COLUMN IF NOT EXISTS type VARCHAR(20) NOT NULL DEFAULT 'manual';
ALTER TABLE mml_scripts ADD COLUMN IF NOT EXISTS progress NUMERIC(5,2) NOT NULL DEFAULT 0;
ALTER TABLE mml_scripts ADD COLUMN IF NOT EXISTS result JSONB DEFAULT '{}';
CREATE INDEX IF NOT EXISTS idx_mml_scripts_type ON mml_scripts(type);

-- 重建 mml_command_params_rel
CREATE TABLE IF NOT EXISTS mml_command_params_rel (
    command_id UUID NOT NULL REFERENCES mml_commands(id) ON DELETE CASCADE,
    param_id   UUID NOT NULL REFERENCES mml_params(id) ON DELETE CASCADE,
    sort_order INT NOT NULL DEFAULT 0,
    PRIMARY KEY (command_id, param_id)
);

-- 重建 mml_audit_log（最简结构，仅供回滚）
CREATE TABLE IF NOT EXISTS mml_audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    operator VARCHAR(100),
    op_type VARCHAR(50),
    target_id UUID,
    detail JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

END $$;
-- +goose StatementEnd

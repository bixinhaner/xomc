-- +goose Up
-- ============================================================
-- 000129 — MML 控制台 CMCC TD-LTE v2.3 命令树重构 (P1.a 基础设施)
--
-- 方案文档：docs/design/mml-console-cmcc-tdlte-v23-adjustment-plan-20260519.md
-- 涉及需求：R-1 chapter_code + object_path_template / R-2 instance_arity
--           R-5 Customized visibility + owner_user_id
--           R-9 catalog deprecated_at 软删
--           sub_field_overrides per-user 配置保护
--
-- 兼容策略：
-- · 现有 mml_param_groups.group_code 含义为业务大类（绑定 param_version），保留不动
-- · 新加 object_path_template 列作为 v2.3 catalog 的 TR-181 路径模板标识
-- · 现有 mml_command_sub_fields (command_id, param_id) 维度保留，按 D24 兼容期方案
--   不改 FK，BuildTree 在 GROUP 维度聚合查询
-- ============================================================

-- ─── 1. mml_param_groups：catalog 元数据扩展 ───────────────────
ALTER TABLE mml_param_groups
    ADD COLUMN IF NOT EXISTS object_path_template VARCHAR(256),
    ADD COLUMN IF NOT EXISTS chapter_code VARCHAR(8),
    ADD COLUMN IF NOT EXISTS instance_arity SMALLINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS instance_levels TEXT[],
    ADD COLUMN IF NOT EXISTS deprecated_at TIMESTAMPTZ;

COMMENT ON COLUMN mml_param_groups.object_path_template IS
    'v2.3 catalog: object 维度归一化 TR-181 路径模板（如 Device.DeviceInfo.*），R-1 标识';
COMMENT ON COLUMN mml_param_groups.chapter_code IS
    'v2.3 catalog: spec 章节代码（SA…SR），仅用于排序元数据，不渲染';
COMMENT ON COLUMN mml_param_groups.instance_arity IS
    'v2.3 catalog: {i} 占位符层级数；0=无实例；多层如 SR 段达 5';
COMMENT ON COLUMN mml_param_groups.instance_levels IS
    'v2.3 catalog: 多层 {i} 的层级语义名（["MU","Slot","EU","RU","RFChannel"]）';

-- 部分唯一索引：standard 范围内的 object_path_template 唯一
CREATE UNIQUE INDEX IF NOT EXISTS uniq_mml_param_groups_object_path
    ON mml_param_groups(object_path_template)
    WHERE object_path_template IS NOT NULL AND deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_mml_param_groups_chapter
    ON mml_param_groups(chapter_code, display_order)
    WHERE chapter_code IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_mml_param_groups_deprecated
    ON mml_param_groups(deprecated_at)
    WHERE deprecated_at IS NOT NULL;

-- ─── 2. mml_commands：Customized 可见性 + 软删 ────────────────
ALTER TABLE mml_commands
    ADD COLUMN IF NOT EXISTS visibility VARCHAR(8),
    ADD COLUMN IF NOT EXISTS owner_user_id UUID,
    ADD COLUMN IF NOT EXISTS deprecated_at TIMESTAMPTZ;

-- visibility 取值约束（NULL 允许 — standard 行 visibility 无意义）
-- +goose StatementBegin
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.constraint_column_usage
        WHERE table_name = 'mml_commands' AND constraint_name = 'chk_mml_commands_visibility'
    ) THEN
        ALTER TABLE mml_commands
            ADD CONSTRAINT chk_mml_commands_visibility
            CHECK (visibility IS NULL OR visibility IN ('public', 'private'));
    END IF;
END $$;
-- +goose StatementEnd

COMMENT ON COLUMN mml_commands.visibility IS
    'R-5: Customized 命令可见性 public/private；source=admin 时必填，standard 时 NULL';
COMMENT ON COLUMN mml_commands.owner_user_id IS
    'R-5: Customized 命令创建者；private 时按 owner 隔离，admin 角色可跨用户访问';
COMMENT ON COLUMN mml_commands.deprecated_at IS
    'v2.3 catalog: Loader 软删标记；列表查询默认过滤非空';

-- 现有 source='admin' 行回填 visibility='private'（保守默认）
UPDATE mml_commands
    SET visibility = 'private'
    WHERE source = 'admin' AND visibility IS NULL;

CREATE INDEX IF NOT EXISTS idx_mml_commands_owner
    ON mml_commands(visibility, owner_user_id)
    WHERE owner_user_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_mml_commands_deprecated
    ON mml_commands(deprecated_at)
    WHERE deprecated_at IS NOT NULL;

-- ─── 3. mml_command_sub_fields：访问类型冗余列 + 软删 ────────
ALTER TABLE mml_command_sub_fields
    ADD COLUMN IF NOT EXISTS access_type VARCHAR(4),
    ADD COLUMN IF NOT EXISTS deprecated_at TIMESTAMPTZ;

-- +goose StatementBegin
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.constraint_column_usage
        WHERE table_name = 'mml_command_sub_fields' AND constraint_name = 'chk_sub_field_access_type'
    ) THEN
        ALTER TABLE mml_command_sub_fields
            ADD CONSTRAINT chk_sub_field_access_type
            CHECK (access_type IS NULL OR access_type IN ('RO', 'RW'));
    END IF;
END $$;
-- +goose StatementEnd

COMMENT ON COLUMN mml_command_sub_fields.access_type IS
    'R-4: RO/RW 冗余列加速前端按 op 过滤（MOD 隐藏 RO）；Loader 写入时按 standardPath 元属性拷贝';

-- ─── 4. mml_command_sub_field_overrides：per-user 配置保护（D29）──
CREATE TABLE IF NOT EXISTS mml_command_sub_field_overrides (
    id                          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sub_field_id                UUID NOT NULL REFERENCES mml_command_sub_fields(id) ON DELETE CASCADE,
    owner_user_id               UUID NOT NULL,
    default_selected_override   BOOLEAN,
    label_i18n_override         JSONB,
    created_at                  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_subfield_owner UNIQUE (sub_field_id, owner_user_id)
);

COMMENT ON TABLE mml_command_sub_field_overrides IS
    'R-6/D29: per-user 保护用户调过的 default_selected / label_i18n，Loader UPSERT 不覆盖。运行期读取走 LEFT JOIN + COALESCE';

CREATE INDEX IF NOT EXISTS idx_sub_field_overrides_owner
    ON mml_command_sub_field_overrides(owner_user_id);

DROP TRIGGER IF EXISTS trigger_sub_field_overrides_updated_at ON mml_command_sub_field_overrides;
CREATE TRIGGER trigger_sub_field_overrides_updated_at
    BEFORE UPDATE ON mml_command_sub_field_overrides
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- +goose Down
-- ============================================================
-- 反向：删除 000129 引入的所有对象
-- ============================================================

DROP TRIGGER IF EXISTS trigger_sub_field_overrides_updated_at ON mml_command_sub_field_overrides;
DROP TABLE IF EXISTS mml_command_sub_field_overrides;

ALTER TABLE mml_command_sub_fields
    DROP CONSTRAINT IF EXISTS chk_sub_field_access_type;
ALTER TABLE mml_command_sub_fields
    DROP COLUMN IF EXISTS access_type,
    DROP COLUMN IF EXISTS deprecated_at;

DROP INDEX IF EXISTS idx_mml_commands_owner;
DROP INDEX IF EXISTS idx_mml_commands_deprecated;
ALTER TABLE mml_commands
    DROP CONSTRAINT IF EXISTS chk_mml_commands_visibility;
ALTER TABLE mml_commands
    DROP COLUMN IF EXISTS visibility,
    DROP COLUMN IF EXISTS owner_user_id,
    DROP COLUMN IF EXISTS deprecated_at;

DROP INDEX IF EXISTS uniq_mml_param_groups_object_path;
DROP INDEX IF EXISTS idx_mml_param_groups_chapter;
DROP INDEX IF EXISTS idx_mml_param_groups_deprecated;
ALTER TABLE mml_param_groups
    DROP COLUMN IF EXISTS object_path_template,
    DROP COLUMN IF EXISTS chapter_code,
    DROP COLUMN IF EXISTS instance_arity,
    DROP COLUMN IF EXISTS instance_levels,
    DROP COLUMN IF EXISTS deprecated_at;

-- ============================================================
-- 000095_mml_command_catalog_v2.sql
-- MML 老交互恢复 v2 数据层（T-0123-P0）
--
-- 设计依据：docs/design/mml-restore-old-interaction-plan-20260514.md v2 APPROVED §4 §M
-- 关联风险：R-206（MML 老交互被 Sprint A 数据模型重构破坏）
--
-- 改动总览：
--   1. mml_params 扩展 8 列元数据 + is_writable 转 GENERATED STORED 派生列（Q1=B）
--   2. mml_commands 扩展 4 列（logical_code / logical_name_i18n / source / catalog_protected）
--   3. mml_param_groups 扩展 2 列（source / catalog_protected）
--   4. 新建 mml_command_sub_fields 表（替代 000090 DROP 的 mml_command_params_rel）
--   5. 3 个触发器：sub_fields 写后回填 target_paths / updated_at / is_writable 派生
-- ============================================================

-- +goose Up

-- ============================================================
-- Section 1: mml_params 扩展元数据 + is_writable GENERATED 转换
--
-- Q1=B 决议：is_writable 由普通 BOOLEAN 列 → GENERATED ALWAYS AS STORED 派生列
--   - 强保证与 access_type 永不漂移
--   - 老 SQL `WHERE is_writable=true` 透明继续工作
--   - 任何写入 is_writable 的代码会被 PG 自动拒绝（符合"派生字段"语义）
-- ============================================================

-- 1.1 新增 8 元数据列
ALTER TABLE mml_params
    ADD COLUMN IF NOT EXISTS access_type          VARCHAR(20) NOT NULL DEFAULT 'READ_ONLY'
        CHECK (access_type IN ('READ_ONLY','READ_WRITE','WRITE_ONLY')),
    ADD COLUMN IF NOT EXISTS is_object            BOOLEAN     NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS supports_add         BOOLEAN     NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS supports_delete      BOOLEAN     NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS change_applies       VARCHAR(20) NOT NULL DEFAULT 'Immediate',
    ADD COLUMN IF NOT EXISTS constraint_text_i18n JSONB       NOT NULL DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS catalog_protected    BOOLEAN     NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS source               VARCHAR(20) NOT NULL DEFAULT 'admin';

-- 1.2 backfill access_type 从既有 is_writable 普通列（保护现有 Sprint A loader 写入数据）
UPDATE mml_params SET access_type = 'READ_WRITE' WHERE is_writable = true;

-- 1.3 DROP 旧 is_writable 普通列（既有 schema 未发现单独 is_writable 索引；如有 IF EXISTS 守护）
DROP INDEX IF EXISTS idx_mml_params_is_writable;
ALTER TABLE mml_params DROP COLUMN is_writable;

-- 1.4 重建 is_writable 为 GENERATED STORED 派生列
ALTER TABLE mml_params
    ADD COLUMN is_writable BOOLEAN
        GENERATED ALWAYS AS (access_type IN ('READ_WRITE','WRITE_ONLY')) STORED;

-- 1.5 索引
CREATE INDEX IF NOT EXISTS idx_mml_params_access_type ON mml_params(access_type);
CREATE INDEX IF NOT EXISTS idx_mml_params_is_object   ON mml_params(is_object);
CREATE INDEX IF NOT EXISTS idx_mml_params_source      ON mml_params(source);

-- 1.6 列注释
COMMENT ON COLUMN mml_params.access_type IS
    'TR-069 访问权限：READ_ONLY / READ_WRITE / WRITE_ONLY；驱动 UI 显示形态（只读灰显 / 输入框）';
COMMENT ON COLUMN mml_params.is_object IS
    'true=TR-069 object 容器（path 以 . 结尾，支持 AddObject/DeleteObject）；false=leaf param';
COMMENT ON COLUMN mml_params.supports_add IS
    '仅 is_object=true 有效：UI 是否显 "+" 添加实例按钮';
COMMENT ON COLUMN mml_params.supports_delete IS
    '仅 is_object=true 有效：UI 是否显 🗑 删除实例按钮';
COMMENT ON COLUMN mml_params.change_applies IS
    '生效时机：Immediate / OnReboot；OnReboot 触发前端二次确认 + 黄 badge 提示';
COMMENT ON COLUMN mml_params.constraint_text_i18n IS
    '多语言约束提示文本（JSONB：{"en-US":"...","zh-CN":"..."}），MOD 输入框右侧灰字渲染';
COMMENT ON COLUMN mml_params.catalog_protected IS
    'true=standard 来源行（防 admin UI 误删/锁定关键字段）；Q2=C 决议 catalog_protected 字段不可 PATCH';
COMMENT ON COLUMN mml_params.source IS
    '来源：standard（XML import）/ admin（手动创建）；决定 catalog_protected 的逻辑等价语义';
COMMENT ON COLUMN mml_params.is_writable IS
    'GENERATED STORED：派生自 access_type IN (READ_WRITE,WRITE_ONLY)；不可直接写入（Q1=B）';


-- ============================================================
-- Section 2: mml_commands 扩展 logical_code + i18n + source/protected
-- ============================================================

ALTER TABLE mml_commands
    ADD COLUMN IF NOT EXISTS logical_code      VARCHAR(100),
    ADD COLUMN IF NOT EXISTS logical_name_i18n JSONB       NOT NULL DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS source            VARCHAR(20) NOT NULL DEFAULT 'admin',
    ADD COLUMN IF NOT EXISTS catalog_protected BOOLEAN     NOT NULL DEFAULT false;

CREATE INDEX IF NOT EXISTS idx_mml_commands_logical_code ON mml_commands(logical_code);

COMMENT ON COLUMN mml_commands.logical_code IS
    '去除 op 前缀的逻辑命令码（command_code "LST_DEVICE_INFO" → logical_code "DEVICE_INFO"）；同 logical 不同 op 是命令树叶子同分支';
COMMENT ON COLUMN mml_commands.logical_name_i18n IS
    '逻辑命令显示名（JSONB），如 {"en-US":"Device info","zh-CN":"设备信息"}；命令树叶子 label 前缀来源';
COMMENT ON COLUMN mml_commands.source IS
    '来源：standard（XML import）/ admin（手动创建）';
COMMENT ON COLUMN mml_commands.catalog_protected IS
    'true=admin UI 不可删除（Q2=C 决议）；不可 PATCH';


-- ============================================================
-- Section 3: mml_param_groups 扩展 source/protected
-- ============================================================

ALTER TABLE mml_param_groups
    ADD COLUMN IF NOT EXISTS source            VARCHAR(20) NOT NULL DEFAULT 'admin',
    ADD COLUMN IF NOT EXISTS catalog_protected BOOLEAN     NOT NULL DEFAULT false;

COMMENT ON COLUMN mml_param_groups.source IS
    '来源：standard / admin';
COMMENT ON COLUMN mml_param_groups.catalog_protected IS
    'true=admin UI 不可删除关键 group';


-- ============================================================
-- Section 4: mml_command_sub_fields 新表
--
-- 替代 000090 DROP 的 mml_command_params_rel；提供 (command_id, param_id) 多对多
-- 关系 + 命令上下文的 mml_code / label_i18n / sort_order / default_selected / is_required
-- ============================================================

CREATE TABLE IF NOT EXISTS mml_command_sub_fields (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    command_id       UUID NOT NULL REFERENCES mml_commands(id) ON DELETE CASCADE,
    param_id         UUID NOT NULL REFERENCES mml_params(id)   ON DELETE RESTRICT,

    -- 老系统 MML 字符串内部使用的 code（命令上下文相关）
    -- 例：path=Device.DeviceInfo.X_COM_MODULE_TYPE 在 DEVICE_INFO 命令叫 LTE_GSM_MODEL_NAME
    mml_code         VARCHAR(100) NOT NULL,

    -- 命令上下文的 sub-field 显示标签（可覆盖 mml_params.name_i18n 的全局默认）
    label_i18n       JSONB NOT NULL DEFAULT '{}',

    -- LST：默认是否勾选；MOD/ADD：本字段是否必填
    default_selected BOOLEAN NOT NULL DEFAULT true,
    is_required      BOOLEAN NOT NULL DEFAULT false,

    sort_order       INT NOT NULL DEFAULT 0,

    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_command_mml_code UNIQUE (command_id, mml_code),
    CONSTRAINT uq_command_param    UNIQUE (command_id, param_id)
);

CREATE INDEX IF NOT EXISTS idx_mml_command_sub_fields_command
    ON mml_command_sub_fields(command_id, sort_order);
CREATE INDEX IF NOT EXISTS idx_mml_command_sub_fields_param
    ON mml_command_sub_fields(param_id);

COMMENT ON TABLE mml_command_sub_fields IS
    'MML 命令 → sub-field 多对多关系（替代 000090 DROP 的 mml_command_params_rel）；'
    'T-0123 老交互恢复方案核心数据结构';
COMMENT ON COLUMN mml_command_sub_fields.mml_code IS
    '老系统 MML 字符串内部 code（如 LTE_GSM_MODEL_NAME）；命令上下文相关';
COMMENT ON COLUMN mml_command_sub_fields.label_i18n IS
    '命令上下文的 sub-field 显示标签（覆盖 mml_params.name_i18n 全局默认）';
COMMENT ON COLUMN mml_command_sub_fields.default_selected IS
    'LST 命令：UI 是否默认勾选；老系统所有 sub-field 默认全勾选';
COMMENT ON COLUMN mml_command_sub_fields.is_required IS
    'MOD/ADD 命令：sub-field 是否必填（前端校验）';

-- updated_at 自动维护（复用 000001 共享函数）
CREATE TRIGGER trg_mml_command_sub_fields_updated_at
    BEFORE UPDATE ON mml_command_sub_fields
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();


-- ============================================================
-- Section 5: target_paths 派生触发器
--
-- 每当 sub_fields INSERT/UPDATE/DELETE → 重算所属 command 的 target_paths JSONB 数组
-- mml_commands.target_paths 语义从"权威源"降级为"派生缓存"（由 sub_fields 维护）
-- ============================================================

-- 5.1 重算函数
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION refresh_mml_command_target_paths(p_command_id UUID)
RETURNS VOID AS $$
BEGIN
    UPDATE mml_commands c
    SET target_paths = COALESCE((
            SELECT jsonb_agg(p.tr069_path ORDER BY csf.sort_order)
            FROM mml_command_sub_fields csf
            JOIN mml_params p ON p.id = csf.param_id
            WHERE csf.command_id = p_command_id
        ), '[]'::jsonb),
        updated_at = NOW()
    WHERE c.id = p_command_id;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

-- 5.2 触发器函数
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION trg_mml_sub_fields_refresh_paths()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        PERFORM refresh_mml_command_target_paths(OLD.command_id);
        RETURN OLD;
    ELSE
        PERFORM refresh_mml_command_target_paths(NEW.command_id);
        RETURN NEW;
    END IF;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

-- 5.3 触发器（AFTER 时机，写入完成后回填）
CREATE TRIGGER trg_mml_sub_fields_target_paths
    AFTER INSERT OR UPDATE OR DELETE ON mml_command_sub_fields
    FOR EACH ROW EXECUTE FUNCTION trg_mml_sub_fields_refresh_paths();


-- +goose Down

-- ============================================================
-- 反向卸载（按 Up 反向顺序）
-- ============================================================

-- 5. 触发器 + 函数
DROP TRIGGER IF EXISTS trg_mml_sub_fields_target_paths ON mml_command_sub_fields;
DROP TRIGGER IF EXISTS trg_mml_command_sub_fields_updated_at ON mml_command_sub_fields;
DROP FUNCTION IF EXISTS trg_mml_sub_fields_refresh_paths();
DROP FUNCTION IF EXISTS refresh_mml_command_target_paths(UUID);

-- 4. mml_command_sub_fields 表
DROP INDEX IF EXISTS idx_mml_command_sub_fields_param;
DROP INDEX IF EXISTS idx_mml_command_sub_fields_command;
DROP TABLE IF EXISTS mml_command_sub_fields;

-- 3. mml_param_groups 反向
ALTER TABLE mml_param_groups
    DROP COLUMN IF EXISTS source,
    DROP COLUMN IF EXISTS catalog_protected;

-- 2. mml_commands 反向
DROP INDEX IF EXISTS idx_mml_commands_logical_code;
ALTER TABLE mml_commands
    DROP COLUMN IF EXISTS logical_code,
    DROP COLUMN IF EXISTS logical_name_i18n,
    DROP COLUMN IF EXISTS source,
    DROP COLUMN IF EXISTS catalog_protected;

-- 1. mml_params 反向（GENERATED → 普通列 → DROP 其他 7 列）

-- 1.5 索引
DROP INDEX IF EXISTS idx_mml_params_access_type;
DROP INDEX IF EXISTS idx_mml_params_is_object;
DROP INDEX IF EXISTS idx_mml_params_source;

-- 1.4 → 1.3 → 1.2 反向：临时列捕获 → DROP generated → rename 回来
ALTER TABLE mml_params ADD COLUMN _is_writable_temp BOOLEAN NOT NULL DEFAULT false;
UPDATE mml_params SET _is_writable_temp = (access_type IN ('READ_WRITE','WRITE_ONLY'));
ALTER TABLE mml_params DROP COLUMN is_writable;
ALTER TABLE mml_params RENAME COLUMN _is_writable_temp TO is_writable;

-- 1.1 DROP 8 新列
ALTER TABLE mml_params
    DROP COLUMN IF EXISTS access_type,
    DROP COLUMN IF EXISTS is_object,
    DROP COLUMN IF EXISTS supports_add,
    DROP COLUMN IF EXISTS supports_delete,
    DROP COLUMN IF EXISTS change_applies,
    DROP COLUMN IF EXISTS constraint_text_i18n,
    DROP COLUMN IF EXISTS catalog_protected,
    DROP COLUMN IF EXISTS source;

-- +goose Up
-- ============================================================
-- 000149 — MML 控制台 spec v2.3 (v2) catalog schema 升级 (P1)
--
-- 规范源：omcgo/规范/移动/南向数据模型/cmcc-tdlte-southbound-data-model-v2.3.md
-- 关联实施：P0 已落（parse_cmcc_tdlte_v23_v2.py + cmcc-tdlte-v2.3.v2.json），
--           P2 Loader 重写消费本迁移产出的新列。
--
-- 引入：
--   1. mml_commands.tree_node_refs JSONB  — §R-2.5 引用 standard_params.standardPath
--   2. mml_commands.instance_range_meta JSONB — §R-4.1.1 每层 {i} 范围 metadata
--   3. mml_catalog_link_health 表 — §R-2.5.2 标准参数树关联失败清单
--
-- 兼容窗口：
--   · 旧 target_paths 列保留不变，v1 Loader 继续可工作；
--   · 新列默认值 '[]'::jsonb，旧 v1 行可平滑共存；
--   · §R-2.4 命令中文名跨章节唯一由 Loader 端 panic 校验，不在本迁移加 DB UNIQUE
--     （原因：v1 数据存在多处 zh-CN 重名如 SA/SR 共用 "设备版本升级"，需 P2 v2
--     Loader 重写数据后才能加上 UNIQUE 约束）；
--   · §R-1 mml_param_groups.path depth ≤ 1 由 Loader 端启动自检 panic，不在 DB
--     加 CHECK（同上原因，v1 数据可能存多层 path）。
-- ============================================================

-- ─── 1. mml_commands：tree_node_refs + instance_range_meta ─────
ALTER TABLE mml_commands
    ADD COLUMN IF NOT EXISTS tree_node_refs JSONB NOT NULL DEFAULT '[]'::jsonb;

COMMENT ON COLUMN mml_commands.tree_node_refs IS
    '§R-2.5: 引用标准参数树(standard_params.standardPath) 的列表（JSONB array of strings）。'
    '运行时 JOIN standard_params 拉 path 类型/范围/权限/中文描述。'
    '替代旧 target_paths 字符串数组（兼容窗口期，target_paths 仍由 v1 Loader 写入；'
    'P2 v2 Loader 同时写两列，P3 切流量到 tree_node_refs 后再 DROP target_paths）';

CREATE INDEX IF NOT EXISTS idx_mml_commands_tree_node_refs_gin
    ON mml_commands USING GIN (tree_node_refs);

ALTER TABLE mml_commands
    ADD COLUMN IF NOT EXISTS instance_range_meta JSONB NOT NULL DEFAULT '[]'::jsonb;

COMMENT ON COLUMN mml_commands.instance_range_meta IS
    '§R-4.1.1: 每层 {i} 占位符的取值范围元数据，按 layer 顺序排列。'
    '每个元素含 layer/rangeExpr/rangeMin/rangeMax/dynamic/nSource/description；'
    '前端 LST Control Panel 渲染 InstancePicker 时用于校验：静态范围 [min,max] 闭区间硬校验，'
    '动态范围（dynamic=true，nSource 指向某 NumberOfEntries 参数）由设备 LST 缓存上限 + 后端兜底';

-- ─── 2. mml_catalog_link_health：§R-2.5.2 关联失败清单 ─────────
CREATE TABLE IF NOT EXISTS mml_catalog_link_health (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    spec_version        VARCHAR(64)  NOT NULL,
    standard_path       VARCHAR(512) NOT NULL,
    group_code_object   VARCHAR(512) NOT NULL,
    operation_type      VARCHAR(8),
    failure_reason      VARCHAR(8)   NOT NULL,
    notes               TEXT,
    decision            TEXT,
    owner               VARCHAR(64),
    detected_at         TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    resolved_at         TIMESTAMPTZ,
    created_at          TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_mml_catalog_link_health_path
        UNIQUE (spec_version, standard_path, group_code_object),
    CONSTRAINT chk_mml_catalog_link_health_reason
        CHECK (failure_reason IN ('A', 'B', 'C', 'D'))
);

COMMENT ON TABLE mml_catalog_link_health IS
    '§R-2.5.2: catalog Loader 解析 spec MD 时把"无法在 standard_params 命中"的 path 写入。'
    '每次 Loader 启动按 (spec_version, standard_path, group_code_object) 幂等 UPSERT；'
    'admin API GET /api/v1/mml/catalog/link-health 列出 resolved_at IS NULL 的项；'
    '维护者修正 spec / standard_params 后，下次 Loader 启动如果该 path 命中标准参数树，'
    '自动把对应记录的 resolved_at 写为 NOW()';

COMMENT ON COLUMN mml_catalog_link_health.failure_reason IS
    'A=标准参数树未覆盖（spec 合法 path 但 standard_params 未导入）/ '
    'B=spec 写错（拼写/大小写/{i} 嵌套不符 TR-181）/ '
    'C=厂商私有路径泄漏到 standardPath 列（应移到 privatePath 或 X_VENDOR_*）/ '
    'D=命令已废弃（新协议版本已移除）';

CREATE INDEX IF NOT EXISTS idx_mml_catalog_link_health_unresolved
    ON mml_catalog_link_health(spec_version, detected_at)
    WHERE resolved_at IS NULL;

DROP TRIGGER IF EXISTS trigger_mml_catalog_link_health_updated_at ON mml_catalog_link_health;
CREATE TRIGGER trigger_mml_catalog_link_health_updated_at
    BEFORE UPDATE ON mml_catalog_link_health
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- +goose Down
-- ============================================================
-- 反向：删除 000149 引入的所有对象
-- ============================================================

DROP TRIGGER IF EXISTS trigger_mml_catalog_link_health_updated_at ON mml_catalog_link_health;
DROP INDEX IF EXISTS idx_mml_catalog_link_health_unresolved;
DROP TABLE IF EXISTS mml_catalog_link_health;

DROP INDEX IF EXISTS idx_mml_commands_tree_node_refs_gin;

ALTER TABLE mml_commands
    DROP COLUMN IF EXISTS instance_range_meta,
    DROP COLUMN IF EXISTS tree_node_refs;

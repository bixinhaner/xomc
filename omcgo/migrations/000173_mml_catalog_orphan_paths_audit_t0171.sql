-- +goose Up
-- ============================================================
-- 000173_mml_catalog_orphan_paths_audit_t0171.sql
-- T-0171（草案）—— standard_params 孤儿 path 审计（不污染业务表）
--
-- 背景：
--   standard_params 当前 2168 行，其中 1544 行从未被任何 mml_command_sub_fields
--   引用（"孤儿 path"）。这些 path 主要是 BAICELLS BLQ/MLN 等厂商私有扩展
--   + 5G NR / GSM / SAS / IPsec 扩展子树，**不属于** cmcc-tdlte-v2.3 spec md
--   §R-2.4 71 个 group_code 任一个。
--
-- 设计决策（用户与 AI 协商，2026-05-24）：
--   1. **不自动派生 mml_commands**：因为章节归属 / 命令命名 / 派生 op 类型
--      都是业务决策，不能 SQL 自动猜。盲派生会污染 cmcc-tdlte-v2.3 catalog。
--   2. **审计型迁移**：把孤儿 path 按 object_prefix 聚合到独立审计表，业务方
--      review 后再决定如何处置（独立私有 catalog / 手工 INSERT / 忽略）。
--   3. **Down 完全可逆**：DROP 审计表即可，不动 mml_commands / sub_fields。
--
-- 升级路径（业务方拍板后）：
--   方案 A — 完全忽略：保留这些 path 仅作字典存档（前端 quicksettings tab 通过
--           param_mappings 反查仍可用），不进 MML 命令树
--   方案 B — 独立私有 catalog：业务方编写 BLQ-private-v1.0.md → omcctl mml
--           import-spec-md 生成对应 seed（与 T-0169 同款工具）
--   方案 C — 精确补漏：admin Tab 3 UI 手工 INSERT 个别 path 到现有命令的 sub_field
-- ============================================================

-- ----------------------------------------------------------------------------
-- Step 1: 建审计表
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS mml_catalog_orphan_paths_audit_t0171 (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    standard_path     TEXT NOT NULL UNIQUE,
    standard_param_id UUID NOT NULL REFERENCES standard_params(id) ON DELETE CASCADE,

    -- 派生字段：object_prefix（path 去末段后加 .*），用于按对象聚合
    -- 例: Device.DeviceInfo.AntennaInfo.Azimuth → Device.DeviceInfo.AntennaInfo.*
    object_prefix TEXT NOT NULL,

    -- 元属性快照（standard_params 同名列；冗余存储便于审计查询不 JOIN）
    access     VARCHAR(16),
    data_type  VARCHAR(16),

    -- 业务方处置字段：留 NULL 待人工填写
    -- TODO: 业务方 review 后填 'ignore' / 'private_catalog' / 'manual_subfield'
    -- TODO: 若 disposition='private_catalog'，suggested_group_code 填建议归属
    -- TODO: 若 disposition='manual_subfield'，suggested_command_code 填目标命令
    disposition            VARCHAR(32),
    suggested_group_code   TEXT,
    suggested_command_code TEXT,
    reviewer               VARCHAR(100),
    reviewed_at            TIMESTAMPTZ,
    notes                  TEXT,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_orphan_paths_object_prefix
    ON mml_catalog_orphan_paths_audit_t0171(object_prefix);
CREATE INDEX IF NOT EXISTS idx_orphan_paths_disposition_null
    ON mml_catalog_orphan_paths_audit_t0171(disposition)
    WHERE disposition IS NULL;  -- 部分索引：待处置项加速查询

CREATE TRIGGER trigger_orphan_paths_updated_at
    BEFORE UPDATE ON mml_catalog_orphan_paths_audit_t0171
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

COMMENT ON TABLE mml_catalog_orphan_paths_audit_t0171 IS
    'T-0171 审计：standard_params 中未被任何 mml_command_sub_fields 引用的孤儿 path '
    '快照（1544 行 BAICELLS 私有扩展 + 5G/GSM/SAS/IPsec 子树）。业务方 review 后填 '
    'disposition 决定处置；本表不污染 mml_commands / sub_fields 业务表。';

COMMENT ON COLUMN mml_catalog_orphan_paths_audit_t0171.disposition IS
    '业务方决策：ignore=忽略仅作字典存档 / private_catalog=进独立私有 catalog（用 '
    'omcctl mml import-spec-md 生成 seed） / manual_subfield=admin UI 手工补关联到 '
    '现有命令。NULL=待 review。';

-- ----------------------------------------------------------------------------
-- Step 2: 写入孤儿 path 快照
-- ----------------------------------------------------------------------------
-- LEFT JOIN 基准：standard_params 为左表，mml_command_sub_fields 为右表；
-- NOT EXISTS 等价于 LEFT JOIN ... WHERE csf.id IS NULL，但语义更直接。
--
-- 幂等性：ON CONFLICT (standard_path) DO NOTHING 保证重跑不报错（CLAUDE.md §5.5.7）；
-- 不 DO UPDATE 因为审计字段（disposition/reviewer 等）由业务方手工填，重跑不应覆盖。
-- +goose StatementBegin
INSERT INTO mml_catalog_orphan_paths_audit_t0171 (
    standard_path, standard_param_id, object_prefix, access, data_type
)
SELECT
    sp.standard_path,
    sp.id,
    -- object_prefix 派生：去末段后追加 .*
    -- 例: Device.X.Y.Z       → Device.X.Y.*
    -- 例: Device.X.{i}.Y     → Device.X.{i}.*
    regexp_replace(sp.standard_path, '\.[^.]+$', '.*') AS object_prefix,
    sp.access,
    sp.data_type
  FROM standard_params sp
 WHERE NOT EXISTS (
     SELECT 1 FROM mml_command_sub_fields csf
      WHERE csf.standard_path_id = sp.id
 )
ON CONFLICT (standard_path) DO NOTHING;
-- +goose StatementEnd

-- ----------------------------------------------------------------------------
-- Step 3: 输出审计统计（写入 NOTICE 让 goose stdout 可见）
-- ----------------------------------------------------------------------------
-- +goose StatementBegin
DO $$
DECLARE
    total_count       INT;
    by_object_count   INT;
    sample_paths      TEXT;
BEGIN
    SELECT COUNT(*) INTO total_count FROM mml_catalog_orphan_paths_audit_t0171;
    SELECT COUNT(DISTINCT object_prefix) INTO by_object_count FROM mml_catalog_orphan_paths_audit_t0171;
    SELECT string_agg(object_prefix || ' (' || cnt || ')', E'\n  ' ORDER BY cnt DESC)
      INTO sample_paths
      FROM (
          SELECT object_prefix, COUNT(*) AS cnt
            FROM mml_catalog_orphan_paths_audit_t0171
           GROUP BY object_prefix
           ORDER BY COUNT(*) DESC
           LIMIT 10
      ) t;

    RAISE NOTICE 'T-0171 orphan paths audit complete';
    RAISE NOTICE '  Total orphan paths: %', total_count;
    RAISE NOTICE '  Distinct object prefixes: %', by_object_count;
    RAISE NOTICE '  Top 10 object prefixes:';
    RAISE NOTICE '  %', sample_paths;
    RAISE NOTICE 'Next: 业务方 review mml_catalog_orphan_paths_audit_t0171 表，';
    RAISE NOTICE '       为每行填 disposition 字段（ignore / private_catalog / manual_subfield）';
END $$;
-- +goose StatementEnd

-- ============================================================
-- 不动 mml_commands / mml_command_sub_fields / standard_params
-- 任何业务表 — 本迁移是纯审计型，安全可重跑。
-- ============================================================


-- +goose Down
-- ============================================================
-- 安全回滚：仅 DROP 审计表，不动业务数据。
-- 由于本 Up 没修改 mml_commands / mml_command_sub_fields / standard_params 任一行，
-- Down 阶段无需"恢复"任何业务数据。
-- ============================================================
DROP TABLE IF EXISTS mml_catalog_orphan_paths_audit_t0171;

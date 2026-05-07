-- T-0098-P1-03 — 参数字典 schema（设计 §1.2 / 实施计划 §1.1）
--
-- 目标：建立"参数模型 + 默认映射 + 设备发现映射 + 标准参数树"四件套，承载新范式
-- standardPath ↔ privatePath 双向翻译。下游 Path A 模板下发、Path B 自动同步、Path C 模型上传
-- 全部从此 schema 取数。
--
-- 设计依据：docs/design/参数-KPI-告警-整合设计方案.md §1.2.1 / §1.2.2 / §1.2.3 / §1.2.4 / §1.2.5
--
-- 表关系：
--   param_models(9 行)              — 参数模型主表（以 XML 文件 paramModel 维度落地）
--   param_mappings(~4781 行)        — 默认映射（每模型 N 条 standardPath ↔ privatePath）
--   discovered_param_mappings       — 设备发现映射（按 product_id + swVersion 隔离的交集结果）
--   standard_params(2001 行)        — 标准参数树（统一 standardPath 字典）
--
-- 既有表 ALTER：
--   devices.param_model_id          UUID（nullable，无 FK；与既有 data_model_id 模式一致，分区表）
--   parameter_discovery_log.param_model_id  UUID FK ON DELETE SET NULL
--
-- 跨域 FK 收口（P1-02 留待此处补齐）：
--   products.param_model_id → param_models(id) ON DELETE SET NULL
--     原因：P1-02 创建 products 时 param_models 尚未存在；本迁移用 ALTER ... ADD CONSTRAINT 追加
--     SET NULL 而非 CASCADE：products 是装配件，param_model 删除不应链式删除整个产品
--
-- is_storable 语义（设计 §1.11 Path B 关键）：
--   param_mappings.is_storable / discovered_param_mappings.is_storable
--   = false 时 sync 不写库（XML 来源属性 store="false"，缺省视 true）
--   discovered 表的 is_storable 从 default 复制，不参与 device_attrs_override 覆盖

-- +goose Up

-- 1. param_models — 参数模型主表
CREATE TABLE IF NOT EXISTS param_models (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name          VARCHAR(64) NOT NULL UNIQUE,
    total_entries INT NOT NULL DEFAULT 0,
    total_objects INT NOT NULL DEFAULT 0,
    total_params  INT NOT NULL DEFAULT 0,
    description   TEXT,
    is_active     BOOLEAN NOT NULL DEFAULT TRUE,
    loaded_from   VARCHAR(256),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_param_models_active ON param_models(is_active);

CREATE TRIGGER trigger_param_models_updated_at
    BEFORE UPDATE ON param_models
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

COMMENT ON TABLE  param_models IS 'T-0098 参数模型主表（设计 §1.2.1）；9 行典型规模，由 9 个 XML 文件载入';
COMMENT ON COLUMN param_models.name IS '模型名（唯一），如 "BLQ" / "MLN" / "BaiBNQ"';
COMMENT ON COLUMN param_models.loaded_from IS 'XML 源文件名（如 BLQ.xml），溯源用';

-- 2. param_mappings — 默认映射（XML 导入）
CREATE TABLE IF NOT EXISTS param_mappings (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    param_model_id  UUID NOT NULL REFERENCES param_models(id) ON DELETE CASCADE,
    standard_path   TEXT NOT NULL,
    private_path    TEXT NOT NULL,
    entry_type      VARCHAR(8) NOT NULL CHECK (entry_type IN ('object', 'parameter')),
    access          VARCHAR(16),
    data_type       VARCHAR(16),
    change_applies  VARCHAR(16),
    min_value       BIGINT,
    max_value       BIGINT,
    is_storable     BOOLEAN NOT NULL DEFAULT TRUE,
    is_active       BOOLEAN NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS uniq_param_mappings_model_standard
    ON param_mappings(param_model_id, standard_path);
CREATE INDEX IF NOT EXISTS idx_param_mappings_model_private_active
    ON param_mappings(param_model_id, private_path) WHERE is_active;
CREATE INDEX IF NOT EXISTS idx_param_mappings_storable
    ON param_mappings(param_model_id) WHERE is_storable;

CREATE TRIGGER trigger_param_mappings_updated_at
    BEFORE UPDATE ON param_mappings
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

COMMENT ON TABLE  param_mappings IS 'T-0098 默认映射（设计 §1.2.2）；~4781 行典型规模，每条 standardPath ↔ privatePath 双向映射';
COMMENT ON COLUMN param_mappings.is_storable IS '是否纳入 OMC 自动同步；XML store="false" → false，缺省 true（Path B sync 据此过滤）';
COMMENT ON COLUMN param_mappings.entry_type IS 'object（容器对象）或 parameter（叶子参数）';

-- 3. discovered_param_mappings — 设备发现映射（按 product_id + swVersion 隔离）
CREATE TABLE IF NOT EXISTS discovered_param_mappings (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id        UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    software_version  VARCHAR(64) NOT NULL,
    standard_path     TEXT NOT NULL,
    private_path      TEXT NOT NULL,
    entry_type        VARCHAR(8) NOT NULL CHECK (entry_type IN ('object', 'parameter')),
    access            VARCHAR(16),
    data_type         VARCHAR(16),
    change_applies    VARCHAR(16),
    min_value         BIGINT,
    max_value         BIGINT,
    is_storable       BOOLEAN NOT NULL DEFAULT TRUE,
    is_active         BOOLEAN NOT NULL DEFAULT TRUE,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS uniq_discovered_product_swver_standard
    ON discovered_param_mappings(product_id, software_version, standard_path);
CREATE INDEX IF NOT EXISTS idx_discovered_product_swver_private_active
    ON discovered_param_mappings(product_id, software_version, private_path) WHERE is_active;

CREATE TRIGGER trigger_discovered_param_mappings_updated_at
    BEFORE UPDATE ON discovered_param_mappings
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

COMMENT ON TABLE  discovered_param_mappings IS 'T-0098 设备发现映射（设计 §1.2.3）；按 product_id + swVersion 隔离，FileType=11 交集落地';
COMMENT ON COLUMN discovered_param_mappings.product_id IS 'CASCADE：product 删除时一并清理本产品全部 swVersion 的交集数据';
COMMENT ON COLUMN discovered_param_mappings.is_storable IS '从默认映射继承；不参与 device_attrs_override 覆盖（设计 §1.2.3）';

-- 4. standard_params — 标准参数树
CREATE TABLE IF NOT EXISTS standard_params (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    standard_path   TEXT NOT NULL UNIQUE,
    entry_type      VARCHAR(8) NOT NULL CHECK (entry_type IN ('object', 'parameter')),
    access          VARCHAR(16),
    data_type       VARCHAR(16),
    change_applies  VARCHAR(16),
    min_value       BIGINT,
    max_value       BIGINT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_standard_params_entry_type ON standard_params(entry_type);

CREATE TRIGGER trigger_standard_params_updated_at
    BEFORE UPDATE ON standard_params
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

COMMENT ON TABLE standard_params IS 'T-0098 标准参数树（设计 §1.2.4）；2001 行典型规模，OMC 统一标准 path 字典';

-- 5. 既有表 ALTER：devices 与 parameter_discovery_log 新增 param_model_id 列
-- devices 是分区表（迁移 000003 LIST PARTITION BY carrier），与既有 data_model_id 一致不加 FK；引用完整性由 ParamRegistry + 应用层校验
ALTER TABLE devices ADD COLUMN IF NOT EXISTS param_model_id UUID;
COMMENT ON COLUMN devices.param_model_id IS 'T-0098 参数模型 UUID 软引用；分区表无 FK（与 data_model_id / product_id 模式一致）';

-- parameter_discovery_log 非分区表，使用 FK ON DELETE SET NULL（与既有 data_model_id 行为对齐）
ALTER TABLE parameter_discovery_log ADD COLUMN IF NOT EXISTS param_model_id UUID;
-- +goose StatementBegin
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'fk_pdl_param_model'
    ) THEN
        ALTER TABLE parameter_discovery_log
            ADD CONSTRAINT fk_pdl_param_model
            FOREIGN KEY (param_model_id) REFERENCES param_models(id) ON DELETE SET NULL;
    END IF;
END $$;
-- +goose StatementEnd
CREATE INDEX IF NOT EXISTS idx_pdl_param_model ON parameter_discovery_log(param_model_id) WHERE param_model_id IS NOT NULL;
COMMENT ON COLUMN parameter_discovery_log.param_model_id IS 'T-0098 参数模型解析结果；与 data_model_id 共存（旧通路）直至 P5 清理';

-- 6. 跨域 FK：补齐 products.param_model_id → param_models（P1-02 留待此处）
-- +goose StatementBegin
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'fk_products_param_model'
    ) THEN
        ALTER TABLE products
            ADD CONSTRAINT fk_products_param_model
            FOREIGN KEY (param_model_id) REFERENCES param_models(id) ON DELETE SET NULL;
    END IF;
END $$;
-- +goose StatementEnd

-- +goose Down

-- 反向：先 drop 跨域 FK，再 drop 既有表新增列，最后 drop 新建表（外键依赖顺序）
ALTER TABLE products DROP CONSTRAINT IF EXISTS fk_products_param_model;

DROP INDEX IF EXISTS idx_pdl_param_model;
ALTER TABLE parameter_discovery_log DROP CONSTRAINT IF EXISTS fk_pdl_param_model;
ALTER TABLE parameter_discovery_log DROP COLUMN IF EXISTS param_model_id;
ALTER TABLE devices DROP COLUMN IF EXISTS param_model_id;

DROP TRIGGER IF EXISTS trigger_standard_params_updated_at ON standard_params;
DROP TABLE IF EXISTS standard_params;

DROP TRIGGER IF EXISTS trigger_discovered_param_mappings_updated_at ON discovered_param_mappings;
DROP TABLE IF EXISTS discovered_param_mappings;

DROP TRIGGER IF EXISTS trigger_param_mappings_updated_at ON param_mappings;
DROP TABLE IF EXISTS param_mappings;

DROP TRIGGER IF EXISTS trigger_param_models_updated_at ON param_models;
DROP TABLE IF EXISTS param_models;

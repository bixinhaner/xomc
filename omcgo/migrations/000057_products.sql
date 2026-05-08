-- T-0098-P1-02 — 产品装配件 schema（设计 §4.2 / 实施计划 §1.4）
--
-- 目标：建立"产品"装配件，作为 productClass 路由入口与三字典（参数 / KPI / 告警）的引用枢纽。
-- 设计依据：docs/design/参数-KPI-告警-整合设计方案.md §4.2.1 / §4.2.2 / §4.2.3
--
-- 表关系：
--   products(15 行)            — 产品主表，含三字典软引用 + 三策略字段
--   product_class_patterns(29 行) — productClass 正则路由（全局 sort_order 升序匹配）
--   devices.product_id 新增列  — 设备 Bootstrap 时由 ProductRegistry 路由写入
--
-- 软引用 vs 硬 FK：
--   - product_class_patterns.product_id → products.id  使用 FK CASCADE（强一致）
--   - products.param_model_id          硬 FK，由 P1-03 在创建 param_models 后追加（本迁移先以 UUID 列预留，nullable）
--   - products.indicator_platform / alarm_ne_type 用 VARCHAR 软引用（设计 §4.2.1 注：跨表字符串维度，硬 FK 反造成依赖混乱；引用完整性由 handler 校验）
--   - devices.product_id 新增 UUID 列 NOT NULL FK：因 devices 是分区表，参照 devices.data_model_id 既有模式不加 FK（应用层校验）
--
-- 三策略字段（设计 §4.2.1）：
--   enable_filetype11      该产品 Bootstrap 时是否下发 Upload(FileType=11) 拉取设备实际参数模型
--   device_attrs_override  JSONB；交集时哪些元属性用设备上传值覆盖默认；data_type=true 业务上禁用（handler 拒绝）
--   enable_unknown_alarm   identifier 不在告警库时是否仍写入活动告警表（fallback severity=Warning, is_unknown=true）
--
-- products.xml lenient unmarshal（实施计划 §P1-06 行 Notes）：
--   <enableUnknownAlarm> 属性可缺省，Go XML unmarshal 时 bool 默认 false，与 DEFAULT false 一致

-- +goose Up

CREATE TABLE IF NOT EXISTS products (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_name          VARCHAR(128) NOT NULL UNIQUE,
    vendor                VARCHAR(64),
    tech                  VARCHAR(8),
    radio_modes           VARCHAR(64),
    description           TEXT,
    param_model_id        UUID,                        -- FK 由 P1-03 追加（指向尚未创建的 param_models）
    indicator_device_type VARCHAR(8) NOT NULL,         -- "enb" / "gsm" / "gnb"，与 indicator_platform 联合定位 KPI 表
    indicator_platform    VARCHAR(32) NOT NULL,        -- KPI 平台名（软引用）
    alarm_ne_type         VARCHAR(16) NOT NULL,        -- 告警 ne_type（软引用 alarm_definitions.ne_type）
    enable_filetype11     BOOLEAN NOT NULL DEFAULT TRUE,
    device_attrs_override JSONB NOT NULL DEFAULT '{}'::jsonb,
    enable_unknown_alarm  BOOLEAN NOT NULL DEFAULT FALSE,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_products_param_model     ON products(param_model_id) WHERE param_model_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_products_indicator       ON products(indicator_device_type, indicator_platform);
CREATE INDEX IF NOT EXISTS idx_products_alarm_ne_type   ON products(alarm_ne_type);

CREATE TRIGGER trigger_products_updated_at
    BEFORE UPDATE ON products
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

COMMENT ON TABLE  products IS 'T-0098 产品装配件主表（设计 §4.2.1）；15 行典型规模，三字典软引用 + 三策略字段';
COMMENT ON COLUMN products.param_model_id IS '参数模型软引用；硬 FK 由 P1-03 创建 param_models 后追加';
COMMENT ON COLUMN products.indicator_platform IS 'KPI 平台名软引用，handler 校验存在于 platform_indicator_formulas_{deviceType}';
COMMENT ON COLUMN products.alarm_ne_type IS '告警 ne_type 软引用，handler 校验存在于 alarm_definitions.ne_type';
COMMENT ON COLUMN products.enable_filetype11 IS '该产品 Bootstrap 时是否下发 Upload(FileType=11)';
COMMENT ON COLUMN products.device_attrs_override IS '交集时哪些元属性以设备上传值覆盖默认；data_type=true 业务禁止';
COMMENT ON COLUMN products.enable_unknown_alarm IS 'identifier 不在告警库时是否 fallback 写活动告警表（is_unknown=true）';

CREATE TABLE IF NOT EXISTS product_class_patterns (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id    UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    product_class TEXT NOT NULL,
    sort_order    INT  NOT NULL,
    is_active     BOOLEAN NOT NULL DEFAULT TRUE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_product_class_patterns_sort     ON product_class_patterns(sort_order ASC) WHERE is_active;
CREATE INDEX IF NOT EXISTS idx_product_class_patterns_product  ON product_class_patterns(product_id);
CREATE UNIQUE INDEX IF NOT EXISTS uniq_product_class_patterns_global_order ON product_class_patterns(sort_order) WHERE is_active;

CREATE TRIGGER trigger_product_class_patterns_updated_at
    BEFORE UPDATE ON product_class_patterns
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

COMMENT ON TABLE  product_class_patterns IS 'productClass → product 路由正则；运行时按 sort_order 全局升序匹配，首命中返回 product_id';
COMMENT ON COLUMN product_class_patterns.sort_order IS '全局唯一 sort_order；FAP 兜底正则必须排在最末';

-- devices 分区表新增 product_id 列（参照既有 data_model_id 模式，不加 FK；引用完整性由应用层 + ProductRegistry 校验）
ALTER TABLE devices ADD COLUMN IF NOT EXISTS product_id UUID;
COMMENT ON COLUMN devices.product_id IS 'T-0098 由 ProductRegistry 路由 productClass 后写入；分区表无 FK 约束（与 data_model_id 既有模式一致）';

-- +goose Down

ALTER TABLE devices DROP COLUMN IF EXISTS product_id;

DROP TRIGGER IF EXISTS trigger_product_class_patterns_updated_at ON product_class_patterns;
DROP TABLE IF EXISTS product_class_patterns;

DROP TRIGGER IF EXISTS trigger_products_updated_at ON products;
DROP TABLE IF EXISTS products;

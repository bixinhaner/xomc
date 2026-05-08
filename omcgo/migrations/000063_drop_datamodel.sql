-- +goose Up
-- ============================================================
-- 000063_drop_datamodel.sql
-- T-0098-P5-02：DROP 旧 data_model 表族 + 列
--
-- 决议依据：
--   - 设计文档 `docs/design/参数-KPI-告警-整合设计方案.md` D5=B
--   - 新参数字典走 param_models / param_mappings / discovered_param_mappings（migrations/000058）
--   - 路由改由 product_class + ProductRegistry 完成（migrations/000057），不再依赖 data_model_id
--
-- DROP 范围：
--   - data_model_import_log（FK → data_model_definitions.id ON DELETE CASCADE，先 DROP）
--   - oui_registry（厂商查询表，已无消费者）
--   - parameter_discovery_log.data_model_id 列（FK → data_model_definitions.id；先 DROP 列）
--   - data_model_definitions
--   - devices.data_model_id 列（分区表 ALTER 直接 DROP COLUMN，无 FK 约束）
--
-- 兼容性影响：
--   - core/model/device.go 已移除 DataModelID 字段
--   - device_repository.go / device_info_pg_repository.go 已从 SELECT 列表 / Scan 移除 data_model_id
--   - provision/model.go ParameterDiscoveryLog.DataModelID 已删除
--   - provision/discovery_repository.go INSERT/UPDATE/SELECT 已剥离 data_model_id
-- ============================================================

-- 1. 先 DROP FK 关系最深的表
DROP TABLE IF EXISTS data_model_import_log CASCADE;

-- 2. parameter_discovery_log.data_model_id 列上 FK → data_model_definitions
--    必须在 DROP TABLE data_model_definitions 前显式删列（避免 CASCADE 错伤）
ALTER TABLE parameter_discovery_log DROP COLUMN IF EXISTS data_model_id;

-- 3. DROP datamodel 主表
DROP TABLE IF EXISTS data_model_definitions CASCADE;

-- 4. DROP OUI 表（已无消费者）
DROP TABLE IF EXISTS oui_registry CASCADE;

-- 5. devices 分区表删 data_model_id 列
ALTER TABLE devices DROP COLUMN IF EXISTS data_model_id;

-- +goose Down
-- ============================================================
-- 回滚：重建空表 + 恢复列。数据无法恢复（参数字典已迁至 param_mappings 体系）。
-- 如确需恢复运行旧栈，需手动从 git history 找回 datamodel 包代码并 revert 9 个消费者改造。
-- ============================================================

CREATE TABLE data_model_definitions (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    carrier          VARCHAR(4) NOT NULL,
    technology       VARCHAR(3) NOT NULL,
    version          VARCHAR(16) NOT NULL,
    oui              VARCHAR(6),
    product_class    VARCHAR(64),
    firmware_version TEXT,
    scope            VARCHAR(16) NOT NULL CHECK (scope IN ('product', 'oui', 'carrier_default')),
    status           VARCHAR(12) NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'active', 'deprecated')),
    is_active        BOOLEAN NOT NULL DEFAULT false,
    root_object      VARCHAR(64) NOT NULL DEFAULT 'Device.',
    parameter_tree   JSONB NOT NULL,
    source           VARCHAR(128),
    source_type      TEXT NOT NULL DEFAULT 'manual' CHECK (source_type IN ('manual', 'auto_discovered', 'cpe_uploaded')),
    imported_by      VARCHAR(128),
    spec_document_ref VARCHAR(256),
    description      TEXT,
    object_tree      JSONB,
    model_metadata   JSONB,
    last_accessed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE oui_registry (
    oui              VARCHAR(6) PRIMARY KEY,
    manufacturer     VARCHAR(128) NOT NULL,
    short_name       VARCHAR(32) NOT NULL,
    country          VARCHAR(64),
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE data_model_import_log (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    data_model_id    UUID NOT NULL REFERENCES data_model_definitions(id) ON DELETE CASCADE,
    action           VARCHAR(16) NOT NULL,
    performed_by     VARCHAR(128) NOT NULL,
    changes_summary  JSONB,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE parameter_discovery_log
    ADD COLUMN data_model_id UUID REFERENCES data_model_definitions(id);

ALTER TABLE devices
    ADD COLUMN data_model_id UUID;

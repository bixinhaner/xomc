-- +goose Up
-- ============================================================
-- 000191_pm_query_templates.sql
-- ----------------------------------------------------------------
-- T-0174 阶段 1：指标查询页"查询模板"持久化。
--
-- 表用于存储"指标查询"页面的可复用查询配置（设备 SN 列表 + 指标路径列表
-- + 粒度 + 时间窗预设 + 其它前端可序列化的查询表单状态）。
--
-- 设计：
--   - visibility=public 公共模板，super_admin 角色可读写；普通用户只读
--   - visibility=private 私有模板，仅创建者可读写
--   - payload JSONB：前端可序列化的全表单状态，schema 由前端约定
--   - UNIQUE(creator_id, name)：同一创建者下名字不重；不同 creator 之间可同名
--   - 不建外键关联 admin_users（admin_users 历史上是分区表，外键限制）
--
-- 与 T-0164 G7 adhoc 任务的关系：
--   - adhoc 是"提交一次跑出结果存到 pm_adhoc_aggregation_results 表"
--   - query_template 是"保存查询配置，不跑，每次手动选模板再触发查询"
--   - 两者数据流不同，独立表
-- ============================================================

CREATE TABLE IF NOT EXISTS pm_query_templates (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(128) NOT NULL,
    visibility  VARCHAR(16)  NOT NULL CHECK (visibility IN ('public', 'private')),
    creator_id  UUID         NOT NULL,
    description TEXT,
    payload     JSONB        NOT NULL DEFAULT '{}'::jsonb,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS uniq_pm_query_templates_creator_name
    ON pm_query_templates (creator_id, name);

CREATE INDEX IF NOT EXISTS idx_pm_query_templates_visibility_created
    ON pm_query_templates (visibility, created_at DESC);

COMMENT ON TABLE  pm_query_templates IS 'T-0174 指标查询页可复用查询模板；payload 为前端表单完整序列化状态';
COMMENT ON COLUMN pm_query_templates.visibility  IS 'public（公共模板，super_admin 可写） / private（私有模板，仅 creator 可写）';
COMMENT ON COLUMN pm_query_templates.creator_id  IS '逻辑关联 admin_users.id；不建 FK（admin_users 分区表）';
COMMENT ON COLUMN pm_query_templates.payload     IS 'JSONB 表单序列化：{ device_sns, metric_paths, granularity, time_range_preset, custom_start, custom_end, ... }';

-- +goose Down
DROP INDEX IF EXISTS idx_pm_query_templates_visibility_created;
DROP INDEX IF EXISTS uniq_pm_query_templates_creator_name;
DROP TABLE IF EXISTS pm_query_templates;

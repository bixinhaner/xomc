-- +goose Up
-- issue #213 S1：Dashboard KPI 首页全局布局表（全局单套·按制式各一行）。
--
-- 背景：首页 KPI 折线图区原写死在前端静态配置；本表把布局收为后端全局可配置存储，
-- 管理员配一次、所有用户看同一份。主键 = 制式（lte/nr/gsm），故最多三行。
--
-- layout JSONB 形如：
--   { "panels": [
--       { "title": "...", "metrics": ["LTE_PDCP_VOLUME_DL", ...],
--         "x": 0, "y": 0, "w": 12, "h": 8, "chartType": "line" },
--       ...
--   ] }
-- 每张图记四样：标题 / 指标 symbolic key 列表 / 网格位置(x,y) / 网格大小(w,h)。
-- chartType 字段预留扩展（本版恒为 line）。

CREATE TABLE IF NOT EXISTS public.dashboard_kpi_layouts (
    tech       text PRIMARY KEY,
    layout     jsonb NOT NULL DEFAULT '{"panels": []}'::jsonb,
    updated_at timestamp with time zone NOT NULL DEFAULT now(),
    updated_by uuid,
    CONSTRAINT dashboard_kpi_layouts_tech_check CHECK (tech IN ('lte', 'nr', 'gsm'))
);

COMMENT ON TABLE public.dashboard_kpi_layouts IS 'issue #213：Dashboard 首页 KPI 折线图区全局布局，按制式各一行（lte/nr/gsm），全局单套所有用户共享。';
COMMENT ON COLUMN public.dashboard_kpi_layouts.tech IS '制式主键：lte / nr / gsm。';
COMMENT ON COLUMN public.dashboard_kpi_layouts.layout IS '布局体 JSONB：panels 数组，每图含 title / metrics(symbolic key 列表) / x,y(网格位置) / w,h(网格大小) / chartType(预留，恒 line)。';
COMMENT ON COLUMN public.dashboard_kpi_layouts.updated_by IS '最近一次保存的管理员用户 ID（nullable：seed 灌入的初始行无来源用户）。';

-- +goose Down
DROP TABLE IF EXISTS public.dashboard_kpi_layouts;

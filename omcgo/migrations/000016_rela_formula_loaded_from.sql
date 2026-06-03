-- +goose Up
-- KPI 指标库一级列表恢复「来源 / 加载源」两列:给平台↔公式关系表补「该公式/平台来自哪个 XML 文件」。
-- 一个 XML 文件即一个平台,故同一 platform_name 的所有 formula 行 loaded_from 单值;
-- SummaryByTech 按 platform 聚合时取 MAX(loaded_from) 即得该平台的文件,source 由其前缀派生。
-- 既有行由下次 Loader.run(启动期 dictload,在 HTTP 服务前同步执行)的 TRUNCATE + 全量重写自动回填,
-- 故无需数据 backfill。formula 表为小目录数据,loaded_from 不进 WHERE/GROUP BY,无需索引。
ALTER TABLE rela_platform_indicator_formula_enb ADD COLUMN IF NOT EXISTS loaded_from TEXT;
ALTER TABLE rela_platform_indicator_formula_gsm ADD COLUMN IF NOT EXISTS loaded_from TEXT;
ALTER TABLE rela_platform_indicator_formula_gnb ADD COLUMN IF NOT EXISTS loaded_from TEXT;

-- +goose Down
ALTER TABLE rela_platform_indicator_formula_enb DROP COLUMN IF EXISTS loaded_from;
ALTER TABLE rela_platform_indicator_formula_gsm DROP COLUMN IF EXISTS loaded_from;
ALTER TABLE rela_platform_indicator_formula_gnb DROP COLUMN IF EXISTS loaded_from;

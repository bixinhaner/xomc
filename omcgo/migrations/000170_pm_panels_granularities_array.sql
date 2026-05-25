-- +goose Up
-- G6-Gap-6 (T-0164 收尾 P2 第 2 批)
--
-- pm_panels.granularity TEXT → granularities TEXT[]
--
-- 设计：panel 可以同时绑定多个粒度（hourly / daily / weekly / monthly），
-- 前端 PanelHeader 显示 Tab 让用户切换粒度浏览同一指标的不同时间分辨率，
-- 不需要重新请求 panel CRUD 接口。
--
-- 迁移路径：
--   1) ADD COLUMN granularities TEXT[]
--   2) backfill granularities = ARRAY[granularity]（保留原单粒度行为）
--   3) DROP COLUMN granularity
--   4) NOT NULL + CHECK 元素白名单（15min/hourly/daily/weekly/monthly）+ array_length >= 1

ALTER TABLE pm_panels ADD COLUMN IF NOT EXISTS granularities TEXT[];

UPDATE pm_panels
   SET granularities = ARRAY[granularity]
 WHERE granularities IS NULL AND granularity IS NOT NULL;

ALTER TABLE pm_panels DROP COLUMN IF EXISTS granularity;

ALTER TABLE pm_panels ALTER COLUMN granularities SET NOT NULL;

ALTER TABLE pm_panels DROP CONSTRAINT IF EXISTS pm_panels_granularities_check;
ALTER TABLE pm_panels ADD CONSTRAINT pm_panels_granularities_check CHECK (
    granularities <@ ARRAY['15min','hourly','daily','weekly','monthly']::TEXT[]
    AND array_length(granularities, 1) >= 1
);

-- +goose Down
ALTER TABLE pm_panels ADD COLUMN IF NOT EXISTS granularity TEXT;

UPDATE pm_panels
   SET granularity = COALESCE(granularities[1], 'hourly')
 WHERE granularity IS NULL;

ALTER TABLE pm_panels DROP CONSTRAINT IF EXISTS pm_panels_granularities_check;
ALTER TABLE pm_panels DROP COLUMN IF EXISTS granularities;

ALTER TABLE pm_panels ALTER COLUMN granularity SET NOT NULL;

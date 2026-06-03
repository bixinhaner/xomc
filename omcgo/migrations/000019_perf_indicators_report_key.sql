-- +goose Up
-- PM 原始计数指标落库编号化 P1:给指标表补 report_key 列(上报名锚点)。
-- reportKey 是 PM 文件里不可变的计数器上报名,解析翻译须以它为锚点匹配编号;
-- 指标表此前只存 enName(可改展示名),装载器解析了 reportKey 却丢弃,本列补落库。
-- 可空、无默认:少数指标 reportKey 为空(用 nullIfEmpty 落 NULL),且按 §5.5.11 铁律,
-- 加列不得裸加无默认 NOT NULL(否则打爆已 applied 的初始种子致 migrate-seed 全失败)。
-- 既有行由下次 Loader.run(启动期 dictload,HTTP 服务前同步执行)的 UPSERT 自动回填,无需数据 backfill。
ALTER TABLE perf_indicators_enb ADD COLUMN IF NOT EXISTS report_key varchar(256);
ALTER TABLE perf_indicators_gnb ADD COLUMN IF NOT EXISTS report_key varchar(256);
ALTER TABLE perf_indicators_gsm ADD COLUMN IF NOT EXISTS report_key varchar(256);

-- +goose Down
ALTER TABLE perf_indicators_enb DROP COLUMN IF EXISTS report_key;
ALTER TABLE perf_indicators_gnb DROP COLUMN IF EXISTS report_key;
ALTER TABLE perf_indicators_gsm DROP COLUMN IF EXISTS report_key;

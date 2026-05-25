-- +goose Up
-- +goose StatementBegin
-- T-0164 BUG-6 修复：把 object_ldn 纳入 pm_metrics 自然键
--
-- 根因：
--   uq_pm_metrics_natural 原列清单 (device_oui, device_sn, metric_path, granularity, end_time, time)
--   不含 object_ldn。同一 PM 文件中同 metric_path 跨多个 cell（不同 object_ldn）的行
--   自然键完全相同 → INSERT ... ON CONFLICT DO UPDATE 同语句两次命中同一目标行 →
--   PG 抛 SQLSTATE 21000「ON CONFLICT DO UPDATE command cannot affect row a second time」。
--
-- 现象：批量 INSERT 全量 retry 3 次进 DLQ，pm_metrics 0 行。
--
-- 决策：
--   1. object_ldn 由 NULLABLE 改 NOT NULL DEFAULT ''
--      原因：UNIQUE 索引中两个 NULL 列值视为「不冲突」，相同 (oui, sn, path, gran, end_time, time, NULL)
--      可同时存在 → 破坏幂等。改 NOT NULL DEFAULT '' 后唯一性严格。
--   2. 重建 uq_pm_metrics_natural 索引，把 object_ldn 加到列末。
--   3. 聚合表 pm_metrics_hourly/daily/weekly/monthly 不需要改：
--      aggregator 通过 GROUP BY (oui, sn, metric_path, statis_type) 已把多 cell 合并为单行
--      （object_ldn 取 MIN 代表值），聚合层语义不需要按 cell 区分。

-- 1. 把现存的 NULL 填成 ''（DEFAULT 'd 不影响已有 NULL 行）
UPDATE pm_metrics SET object_ldn = '' WHERE object_ldn IS NULL;

-- 2. NOT NULL + DEFAULT
ALTER TABLE pm_metrics ALTER COLUMN object_ldn SET DEFAULT '';
ALTER TABLE pm_metrics ALTER COLUMN object_ldn SET NOT NULL;

-- 3. 重建 UNIQUE 索引，列末追加 object_ldn
--    TimescaleDB 要求 UNIQUE 含分区列 time（保留）
DROP INDEX IF EXISTS uq_pm_metrics_natural;
CREATE UNIQUE INDEX uq_pm_metrics_natural
    ON pm_metrics (device_oui, device_sn, metric_path, granularity, end_time, time, object_ldn);

-- 4. 部分索引 idx_pm_metrics_object_ldn 的 WHERE object_ldn IS NOT NULL 失效（永真）
--    改成无条件普通索引，并避免与新 UNIQUE 索引前缀重复浪费空间。
--    保留以兼容已有按 ldn 查询的代码（pm/counter/pg_repository.go 用 object_ldn 过滤 cellID）。
DROP INDEX IF EXISTS idx_pm_metrics_object_ldn;
CREATE INDEX idx_pm_metrics_object_ldn ON pm_metrics (object_ldn) WHERE object_ldn <> '';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_pm_metrics_object_ldn;
CREATE INDEX idx_pm_metrics_object_ldn ON pm_metrics (object_ldn) WHERE object_ldn IS NOT NULL;

DROP INDEX IF EXISTS uq_pm_metrics_natural;
CREATE UNIQUE INDEX uq_pm_metrics_natural
    ON pm_metrics (device_oui, device_sn, metric_path, granularity, end_time, time);

ALTER TABLE pm_metrics ALTER COLUMN object_ldn DROP NOT NULL;
ALTER TABLE pm_metrics ALTER COLUMN object_ldn DROP DEFAULT;
-- +goose StatementEnd

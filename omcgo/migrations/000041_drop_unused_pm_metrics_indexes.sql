-- +goose Up
-- PM 写吞吐优化：删除 pm_metrics 两个"读侧无人用、纯写放大"的二级索引。
--
-- 审计（2026-06-11，全代码库 grep）：
--   - ingest_time：无任何 SQL 按它 WHERE/ORDER BY（仅写入时 SET NOW()）→ 纯写税。
--   - object_ldn：小区过滤要么在内存（QueryForKPI / handler groupKey），要么走
--     substring(object_ldn FROM 'Cellid=([0-9]+)')（聚合器 query.go，plain btree 不可用）；
--     无 object_ldn= 等值查询 → 该 partial btree 不服务任何查询，纯写税。
--
-- pm_metrics 每行原维护 7 个索引；删这 2 个后 7→5，实测全 GSM 600 文件（9.2M 行）入库
-- 166s→112s（8 核 PG），写吞吐 55K→82K 行/s。幂等唯一索引 uq_pm_metrics_natural 与
-- 设备/指标读索引（device_time / path_time / time）保留不动。
DROP INDEX IF EXISTS idx_pm_metrics_ingest_time;
DROP INDEX IF EXISTS idx_pm_metrics_object_ldn;

-- +goose Down
CREATE INDEX IF NOT EXISTS idx_pm_metrics_ingest_time ON pm_metrics USING btree (ingest_time DESC);
CREATE INDEX IF NOT EXISTS idx_pm_metrics_object_ldn ON pm_metrics USING btree (object_ldn) WHERE (object_ldn <> '');

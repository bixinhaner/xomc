-- +goose Up
-- pm_metrics_hourly_cagg（000001 建的 TimescaleDB 原生连续聚合视图）与应用自己的
-- Go 聚合管线（internal/pm/aggregator，T-0164-P5/G5 自然日历桶预聚合，写入独立的
-- pm_metrics_hourly 普通 hypertable）功能完全重复——全代码库没有任何查询读
-- pm_metrics_hourly_cagg，是建表时遗留、后来被 Go 侧方案取代但未清理的废弃对象。
--
-- 压测实测：pm_metrics 涨到 2000万+ 行后，它的刷新策略（每15分钟一次，
-- start_offset=24小时）单次运行超过20分钟仍未完成，且与同一 chunk 上的
-- autovacuum 互相抢 IO/Buffer，导致：
--   1. 应用自己的 pm_aggregate_hourly 聚合任务被拖到35分钟以上才失败/超时
--   2. 连 `select count(*) from pm_metrics` 这种简单查询都要等5分钟以上
-- 是当前KPI处理链路"数据进得来、算不出来"的根因，不是并发度或索引问题。
--
-- +goose StatementBegin
SELECT remove_continuous_aggregate_policy('public.pm_metrics_hourly_cagg', if_exists => TRUE);
-- +goose StatementEnd
DROP MATERIALIZED VIEW IF EXISTS public.pm_metrics_hourly_cagg;

-- +goose Down
-- +goose StatementBegin
CREATE MATERIALIZED VIEW public.pm_metrics_hourly_cagg
WITH (timescaledb.continuous) AS
SELECT
    time_bucket('1 hour', time) AS bucket_time,
    metric_path,
    statis_type,
    SUM(metric_value) as sum_val,
    AVG(metric_value) as avg_val,
    MAX(metric_value) as max_val,
    MIN(metric_value) as min_val
FROM public.pm_metrics
WHERE granularity = '15min'
GROUP BY bucket_time, metric_path, statis_type
WITH NO DATA;

SELECT add_continuous_aggregate_policy('public.pm_metrics_hourly_cagg',
    start_offset => INTERVAL '24 hours',
    end_offset => INTERVAL '15 minutes',
    schedule_interval => INTERVAL '15 minutes');
-- +goose StatementEnd

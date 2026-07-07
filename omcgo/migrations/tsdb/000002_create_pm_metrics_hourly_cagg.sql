-- 000002_create_pm_metrics_hourly_cagg.sql
-- 为解决仪表盘尾部 15 分钟聚合逻辑缓慢和维护困难的问题，引入 TimescaleDB 的连续聚合特性

-- +goose Up
-- +goose StatementBegin
-- 创建连续聚合视图，按小时通用聚合，替换掉以前 Go 里面复杂的 fallback 现场汇总和全网表扫描
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

-- 设定自动刷新策略（由于实时聚合开启，15分钟刷新即可保障大部分场景的物化，
-- 查询视图时引擎会自动拼接上过去 15 分钟尚未物化的热数据，对应用透明）
SELECT add_continuous_aggregate_policy('public.pm_metrics_hourly_cagg',
    start_offset => INTERVAL '24 hours',
    end_offset => INTERVAL '15 minutes',
    schedule_interval => INTERVAL '15 minutes');
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT remove_continuous_aggregate_policy('public.pm_metrics_hourly_cagg', if_exists => TRUE);
DROP MATERIALIZED VIEW IF EXISTS public.pm_metrics_hourly_cagg;
-- +goose StatementEnd

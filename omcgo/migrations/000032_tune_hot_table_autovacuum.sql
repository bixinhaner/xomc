-- +goose Up
-- device_info / devices_cmcc 是小而热的表（行数≈全网设备数，量级固定），但每次
-- TR-069 上报都会更新 mme_status/gps_status/sync_status/rf_status/
-- alarm_severity 等字段，churn 速度远超默认 autovacuum 按比例触发的假设——
-- 默认 autovacuum_vacuum_scale_factor=20% 意味着要等 20% 行变成死元组才触发
-- 一次 vacuum，压测实测中 device_info 死元组比例曾被拖到 87.4%，拖慢了每次
-- 扫描/更新的成本。调低 scale_factor + 降低固定阈值，让 autovacuum 更早介入。
ALTER TABLE public.device_info SET (
    autovacuum_vacuum_scale_factor = 0.02,
    autovacuum_vacuum_threshold = 200,
    autovacuum_analyze_scale_factor = 0.02,
    autovacuum_analyze_threshold = 200
);

ALTER TABLE public.devices_cmcc SET (
    autovacuum_vacuum_scale_factor = 0.02,
    autovacuum_vacuum_threshold = 200,
    autovacuum_analyze_scale_factor = 0.02,
    autovacuum_analyze_threshold = 200
);

-- +goose Down
ALTER TABLE public.device_info RESET (
    autovacuum_vacuum_scale_factor,
    autovacuum_vacuum_threshold,
    autovacuum_analyze_scale_factor,
    autovacuum_analyze_threshold
);

ALTER TABLE public.devices_cmcc RESET (
    autovacuum_vacuum_scale_factor,
    autovacuum_vacuum_threshold,
    autovacuum_analyze_scale_factor,
    autovacuum_analyze_threshold
);

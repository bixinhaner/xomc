-- +goose Up
-- +goose StatementBegin

-- Issue #827：从背压决策逻辑移除 CPU 负载信号，只保留磁盘使用率。
-- 清理 sys_configs 中遗留的 cpu_high_per_core / cpu_low_per_core 两条配置行。
-- CPU 负载指标（acs_host_load_per_core）仍保留用于 Prometheus/Grafana 监控，
-- 但配置项不再有意义，删除以保持初始数据库干净。

DELETE FROM public.sys_configs
WHERE category = 'acs.backpressure'
  AND key IN ('cpu_high_per_core', 'cpu_low_per_core');

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- 回滚：恢复原 seed 中的两条 CPU 背压配置行。
INSERT INTO public.sys_configs (id, category, key, value, value_type, description, is_secret, created_at, updated_at, metadata)
VALUES
    ('8c34e838-9a85-456f-a572-9d6bf3a81694', 'acs.backpressure', 'cpu_high_per_core', '0.9', 'float',
     '每核 1 分钟负载高水位，超过停收 PM 上传（#318）', false,
     '2026-06-17 20:08:35.34786+08', '2026-06-17 20:08:35.34786+08', '{}'),
    ('b1a65b76-ba66-436a-8260-5f723924c920', 'acs.backpressure', 'cpu_low_per_core', '0.7', 'float',
     '每核 1 分钟负载低水位，回落到此自动恢复（#318）', false,
     '2026-06-17 20:08:35.34786+08', '2026-06-17 20:08:35.34786+08', '{}')
ON CONFLICT (id) DO NOTHING;

-- +goose StatementEnd

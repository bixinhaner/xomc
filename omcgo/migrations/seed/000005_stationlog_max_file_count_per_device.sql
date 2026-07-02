-- +goose Up
-- #798：新增每设备故障日志文件数配额（默认 5，0=禁用），与已有的全局配额
-- （stationlog.retention.max_file_count，默认 20）并存、互不替代——全局兜底总量失控，
-- 本配额防止单台设备刷屏挤占其他设备的保留空间。

INSERT INTO public.sys_configs (category, key, value, value_type, description, is_public, created_at, updated_at, description_i18n)
VALUES ('stationlog.retention', 'max_file_count_per_device', '5', 'int', '每设备故障日志文件数配额，0=禁用（#798）', false, now(), now(), '{}'::jsonb)
ON CONFLICT (category, key) DO NOTHING;

-- +goose Down
DELETE FROM public.sys_configs WHERE category = 'stationlog.retention' AND key = 'max_file_count_per_device';

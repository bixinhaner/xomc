-- +goose Up
-- issue #318-321：保留/背压相关后端特性的 sys_configs 默认值。
-- 让这些配置出现在前端「系统配置」页可改、并由各模块热加载。代码侧均有同值兜底默认，
-- 故本 seed 缺失不影响功能，仅为可见/可配。
-- 幂等：ON CONFLICT (category,key) DO NOTHING —— 已存在的键不覆盖运维改过的值
-- （如 plan-resources.sh 据磁盘容量写过的 minio.retention）。
INSERT INTO public.sys_configs (category, key, value, value_type, description, is_public) VALUES
  -- #319 MinIO 原始件 ILM 保留天数（pm-files/mr-files 桶过期），默认对齐 60 天保留场景
  ('minio.retention', 'raw_object_days', '60', 'int', 'MinIO 原始 PM/MR 文件 ILM 过期天数（#319）', false),

  -- #320 基站日志保留：时间保留(天) 与文件数配额(0=禁用) 并存
  ('stationlog.retention', 'max_retention_days', '60', 'int', '基站日志按时间保留天数（#320）', false),
  ('stationlog.retention', 'max_file_count', '20', 'int', '故障日志文件数配额，0=禁用仅按时间保留（#320）', false),

  -- #321 入库后原始 XML 压缩回写开关（真机已 gzip 的零成本跳过）
  ('raw_archive', 'compress_after_ingest', 'true', 'bool', '入库成功后把明文 PM/MR 原始 XML 压缩回写 MinIO 省盘（#321）', false),

  -- #318 PM 上传背压 watchdog（磁盘% + 每核负载，迟滞）
  ('acs.backpressure', 'enabled', 'true', 'bool', 'PM 上传资源背压总开关（#318）', false),
  ('acs.backpressure', 'disk_high_pct', '85', 'int', '数据盘使用率高水位%，超过停收 PM 上传（#318）', false),
  ('acs.backpressure', 'disk_low_pct', '75', 'int', '数据盘使用率低水位%，回落到此自动恢复（#318）', false),
  ('acs.backpressure', 'cpu_high_per_core', '0.9', 'float', '每核 1 分钟负载高水位，超过停收 PM 上传（#318）', false),
  ('acs.backpressure', 'cpu_low_per_core', '0.7', 'float', '每核 1 分钟负载低水位，回落到此自动恢复（#318）', false),
  ('acs.backpressure', 'check_interval_sec', '30', 'int', 'PM 上传背压 watchdog 采样周期（秒）（#318）', false)
ON CONFLICT (category, key) DO NOTHING;

-- +goose Down
DELETE FROM public.sys_configs
WHERE category IN ('minio.retention', 'stationlog.retention', 'raw_archive', 'acs.backpressure');

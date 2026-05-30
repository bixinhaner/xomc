-- +goose Up
-- T-0184 内置任务预置：4 维度（network/device_group/product/band）× 3 制式（lte/nr/gsm）= 12 个内置任务。
--
-- 版本号说明：DDL 用 000222，本 seed 用 000223（**不能与 DDL 同号**）。
--   migrate-up 走 `--paths migrations,migrations/seed` 对两目录在同一 goose_db_version 表里
--   各跑一次 goose.Up；DDL 目录先把 222 标为已应用后，seed 目录的同号 222 会被 goose 判为
--   "已应用"而静默跳过（实测复现）。故 seed 取 DDL 之后的下一个空号 223。
--
-- 统一参数（设计 §2.2 + 账本「12 个内置任务清单」锁定）：
--   - mode='continuous'、cron_expr='5 * * * *'（每小时第 5 分滚动触发，由 ContinuousScheduler 捞）
--   - granularity='hourly'（老列）、granularities='{hourly}'（adhoc 真正读的粒度数组，单粒度；
--     值须对齐 metrics.GranularityHourly='hourly' 与结果表 granularity CHECK，账本「hour」是笔误）
--   - status='scheduled'（持续型靠它被 scheduler 捞 → MarkPending → worker 跑聚合）
--   - is_builtin=true、expire_days=60（持续型其实不消费 expire_days，仅占位）
--   - task_type='extraction'（老列默认）、task_subtype='adhoc_aggregation'（走 adhoc 路径）
--   - device_sns='[]'::jsonb（内置任务不限设备）、kpi_codes='[]'::jsonb（老列空）
--   - metric_paths（text[]）= 各制式核心指标全集（真实编号，已核对 perf_indicators_{enb,gnb,gsm}）
--   - window_start/window_end 留 NULL：持续型每次滚动聚合最新可用桶（不限时窗）
--
-- 固定 UUID + ON CONFLICT (id) DO NOTHING 幂等：重启 / 重部署不重复插入。
-- UUID 方案 0184dddd-<dim>-4000-8000-00000000000<tech>（全 hex）：
--   dim 段 0001=network / 0002=device_group / 0003=product / 0004=band；末位 1=lte/2=nr/3=gsm。

-- ── network（全网） ───────────────────────────────────────────────────────
INSERT INTO pm_tasks (id, task_name, task_type, task_subtype, mode, cron_expr,
    device_sns, kpi_codes, metric_paths, granularity, granularities,
    dimension, technology, is_builtin, expire_days, status, progress)
VALUES
  ('0184dddd-0001-4000-8000-000000000001', '内置-全网-LTE', 'extraction', 'adhoc_aggregation', 'continuous', '5 * * * *',
    '[]'::jsonb, '[]'::jsonb,
    ARRAY['K900010015','K900010016','C000060216','K900010014','K900010013','K900010006','K900010002','K900010005','K900010029','K900010027','K900010017','K900010022','K900010021','K900010026'],
    'hourly', '{hourly}', 'network', 'lte', true, 60, 'scheduled', 0),
  ('0184dddd-0001-4000-8000-000000000002', '内置-全网-NR', 'extraction', 'adhoc_aggregation', 'continuous', '5 * * * *',
    '[]'::jsonb, '[]'::jsonb,
    ARRAY['KGNB0511','KGNB0510','KGNB0506','KGNB0505'],
    'hourly', '{hourly}', 'network', 'nr', true, 60, 'scheduled', 0),
  ('0184dddd-0001-4000-8000-000000000003', '内置-全网-GSM', 'extraction', 'adhoc_aggregation', 'continuous', '5 * * * *',
    '[]'::jsonb, '[]'::jsonb,
    ARRAY['KGSM0102','KGSM0103','KGSM0101'],
    'hourly', '{hourly}', 'network', 'gsm', true, 60, 'scheduled', 0)
ON CONFLICT (id) DO NOTHING;

-- ── device_group（设备组） ────────────────────────────────────────────────
INSERT INTO pm_tasks (id, task_name, task_type, task_subtype, mode, cron_expr,
    device_sns, kpi_codes, metric_paths, granularity, granularities,
    dimension, technology, is_builtin, expire_days, status, progress)
VALUES
  ('0184dddd-0002-4000-8000-000000000001', '内置-设备组-LTE', 'extraction', 'adhoc_aggregation', 'continuous', '5 * * * *',
    '[]'::jsonb, '[]'::jsonb,
    ARRAY['K900010015','K900010016','C000060216','K900010014','K900010013','K900010006','K900010002','K900010005','K900010029','K900010027','K900010017','K900010022','K900010021','K900010026'],
    'hourly', '{hourly}', 'device_group', 'lte', true, 60, 'scheduled', 0),
  ('0184dddd-0002-4000-8000-000000000002', '内置-设备组-NR', 'extraction', 'adhoc_aggregation', 'continuous', '5 * * * *',
    '[]'::jsonb, '[]'::jsonb,
    ARRAY['KGNB0511','KGNB0510','KGNB0506','KGNB0505'],
    'hourly', '{hourly}', 'device_group', 'nr', true, 60, 'scheduled', 0),
  ('0184dddd-0002-4000-8000-000000000003', '内置-设备组-GSM', 'extraction', 'adhoc_aggregation', 'continuous', '5 * * * *',
    '[]'::jsonb, '[]'::jsonb,
    ARRAY['KGSM0102','KGSM0103','KGSM0101'],
    'hourly', '{hourly}', 'device_group', 'gsm', true, 60, 'scheduled', 0)
ON CONFLICT (id) DO NOTHING;

-- ── product（产品） ───────────────────────────────────────────────────────
INSERT INTO pm_tasks (id, task_name, task_type, task_subtype, mode, cron_expr,
    device_sns, kpi_codes, metric_paths, granularity, granularities,
    dimension, technology, is_builtin, expire_days, status, progress)
VALUES
  ('0184dddd-0003-4000-8000-000000000001', '内置-产品-LTE', 'extraction', 'adhoc_aggregation', 'continuous', '5 * * * *',
    '[]'::jsonb, '[]'::jsonb,
    ARRAY['K900010015','K900010016','C000060216','K900010014','K900010013','K900010006','K900010002','K900010005','K900010029','K900010027','K900010017','K900010022','K900010021','K900010026'],
    'hourly', '{hourly}', 'product', 'lte', true, 60, 'scheduled', 0),
  ('0184dddd-0003-4000-8000-000000000002', '内置-产品-NR', 'extraction', 'adhoc_aggregation', 'continuous', '5 * * * *',
    '[]'::jsonb, '[]'::jsonb,
    ARRAY['KGNB0511','KGNB0510','KGNB0506','KGNB0505'],
    'hourly', '{hourly}', 'product', 'nr', true, 60, 'scheduled', 0),
  ('0184dddd-0003-4000-8000-000000000003', '内置-产品-GSM', 'extraction', 'adhoc_aggregation', 'continuous', '5 * * * *',
    '[]'::jsonb, '[]'::jsonb,
    ARRAY['KGSM0102','KGSM0103','KGSM0101'],
    'hourly', '{hourly}', 'product', 'gsm', true, 60, 'scheduled', 0)
ON CONFLICT (id) DO NOTHING;

-- ── band（频段） ──────────────────────────────────────────────────────────
INSERT INTO pm_tasks (id, task_name, task_type, task_subtype, mode, cron_expr,
    device_sns, kpi_codes, metric_paths, granularity, granularities,
    dimension, technology, is_builtin, expire_days, status, progress)
VALUES
  ('0184dddd-0004-4000-8000-000000000001', '内置-频段-LTE', 'extraction', 'adhoc_aggregation', 'continuous', '5 * * * *',
    '[]'::jsonb, '[]'::jsonb,
    ARRAY['K900010015','K900010016','C000060216','K900010014','K900010013','K900010006','K900010002','K900010005','K900010029','K900010027','K900010017','K900010022','K900010021','K900010026'],
    'hourly', '{hourly}', 'band', 'lte', true, 60, 'scheduled', 0),
  ('0184dddd-0004-4000-8000-000000000002', '内置-频段-NR', 'extraction', 'adhoc_aggregation', 'continuous', '5 * * * *',
    '[]'::jsonb, '[]'::jsonb,
    ARRAY['KGNB0511','KGNB0510','KGNB0506','KGNB0505'],
    'hourly', '{hourly}', 'band', 'nr', true, 60, 'scheduled', 0),
  ('0184dddd-0004-4000-8000-000000000003', '内置-频段-GSM', 'extraction', 'adhoc_aggregation', 'continuous', '5 * * * *',
    '[]'::jsonb, '[]'::jsonb,
    ARRAY['KGSM0102','KGSM0103','KGSM0101'],
    'hourly', '{hourly}', 'band', 'gsm', true, 60, 'scheduled', 0)
ON CONFLICT (id) DO NOTHING;

-- +goose Down
DELETE FROM pm_tasks WHERE is_builtin = true AND task_subtype = 'adhoc_aggregation'
  AND id IN (
    '0184dddd-0001-4000-8000-000000000001','0184dddd-0001-4000-8000-000000000002','0184dddd-0001-4000-8000-000000000003',
    '0184dddd-0002-4000-8000-000000000001','0184dddd-0002-4000-8000-000000000002','0184dddd-0002-4000-8000-000000000003',
    '0184dddd-0003-4000-8000-000000000001','0184dddd-0003-4000-8000-000000000002','0184dddd-0003-4000-8000-000000000003',
    '0184dddd-0004-4000-8000-000000000001','0184dddd-0004-4000-8000-000000000002','0184dddd-0004-4000-8000-000000000003'
  );

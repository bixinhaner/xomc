-- +goose Up
-- KPI-ALL-IND 阶段1：把 3 条内置「全网」聚合任务（network 维度）的指标列表放开到全库。
--
-- 背景：内置 network 任务（0184dddd-0001-*）当前只列了精选二十来个指标（LTE/NR/GSM），
-- 没贯彻「算全库」的设计初衷；放开任意指标后，非精选指标在全网聚合结果表里没有线可读。
--
-- 方案：清空这 3 条任务的 metric_paths（置为空数组 '{}'）走「全聚」语义——
--   - 聚合器 network 维度的 metric_path 过滤是「列表非空才过滤、空则不过滤」(applyScalarFilters)，
--     空列表即全库不限指标；
--   - 配合默认开着的「全部落库」开关（pm.storage.store_all_metrics，落库前不再按 metric_paths 过滤），
--     全库每个 counter 都会被聚成全网线并落库。
--
-- 只动 network 三条，不碰 device_group/product/band 维度（它们仍保留精选范围，非本任务范围）。
--
-- 幂等：UPDATE 是幂等的（重复应用结果一致）；对全新库（seed 跑在 baseline INSERT 之后）与
-- 既有库重复前向应用均安全。
UPDATE public.pm_tasks
SET metric_paths = '{}',
    updated_at = now()
WHERE id IN (
    '0184dddd-0001-4000-8000-000000000001',  -- 内置-全网-LTE
    '0184dddd-0001-4000-8000-000000000002',  -- 内置-全网-NR
    '0184dddd-0001-4000-8000-000000000003'   -- 内置-全网-GSM
);

-- +goose Down
-- 回滚：把 3 条 network 任务的指标列表还原回放开前的精选子集（与 seed/000001 baseline 一致）。
UPDATE public.pm_tasks
SET metric_paths = '{K900010015,K900010016,C000060216,K900010014,K900010013,K900010006,K900010002,K900010005,K900010029,K900010027,K900010017,K900010022,K900010021,K900010026}',
    updated_at = now()
WHERE id = '0184dddd-0001-4000-8000-000000000001';

UPDATE public.pm_tasks
SET metric_paths = '{KGNB0511,KGNB0510,KGNB0506,KGNB0505}',
    updated_at = now()
WHERE id = '0184dddd-0001-4000-8000-000000000002';

UPDATE public.pm_tasks
SET metric_paths = '{KGSM0102,KGSM0103,KGSM0101}',
    updated_at = now()
WHERE id = '0184dddd-0001-4000-8000-000000000003';

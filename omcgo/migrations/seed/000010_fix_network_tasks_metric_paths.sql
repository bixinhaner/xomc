-- +goose Up
-- ISSUE-474：内置「全网」(network 维度) 3 条聚合任务 metric_paths 原为空数组 '{}'，
-- 在聚合落库层被当作「不按指标过滤、全库每个 counter 都聚成全网线」语义，配合默认开着的
-- pm.storage.store_all_metrics，使全网任务名下落了上百上千个指标；性能仪表盘按任务实际落库
-- 的指标铺图 → 全网任务铺上百张图 → 卡顿。其余 9 条（设备组/产品/频段 × LTE/NR/GSM）因
-- 指标限定在十几个以内，铺图可控。
--
-- 修复：给 3 条全网任务配上与同制式其它维度一致的精选清单（LTE 14 / NR 4 / GSM 3），
-- 收敛「空 = 全库」语义。空=全库能力仍保留给用户自定义 network 任务（聚合器分支不变）。
--
-- 与 seed/000001_init_seed.sql 配套：000001 只在全新建库生效，修不到已部署库（含 dev 库），
-- 故本迁移对现存数据做 UPDATE。
--
-- 幂等：WHERE metric_paths = '{}'，仅当现值仍为空（未被运维手改）时回填，不覆盖运维设置；
-- 重复执行第二次起 WHERE 不命中、零行更新。
UPDATE public.pm_tasks
   SET metric_paths = '{K900010015,K900010016,C000060216,K900010014,K900010013,K900010006,K900010002,K900010005,K900010029,K900010027,K900010017,K900010022,K900010021,K900010026}',
       updated_at = NOW()
 WHERE id = '0184dddd-0001-4000-8000-000000000001'
   AND metric_paths = '{}';

UPDATE public.pm_tasks
   SET metric_paths = '{KGNB0511,KGNB0510,KGNB0506,KGNB0505}',
       updated_at = NOW()
 WHERE id = '0184dddd-0001-4000-8000-000000000002'
   AND metric_paths = '{}';

UPDATE public.pm_tasks
   SET metric_paths = '{KGSM0102,KGSM0103,KGSM0101}',
       updated_at = NOW()
 WHERE id = '0184dddd-0001-4000-8000-000000000003'
   AND metric_paths = '{}';

-- +goose Down
-- 回滚为空数组 '{}'（恢复全库聚合语义）。配对 .up，演练用。
UPDATE public.pm_tasks
   SET metric_paths = '{}',
       updated_at = NOW()
 WHERE id IN (
   '0184dddd-0001-4000-8000-000000000001',
   '0184dddd-0001-4000-8000-000000000002',
   '0184dddd-0001-4000-8000-000000000003'
 );

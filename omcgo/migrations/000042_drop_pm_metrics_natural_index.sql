-- +goose Up
-- PM 写吞吐 + 低 CPU 优化：删除 pm_metrics 自然键唯一索引 uq_pm_metrics_natural。
--
-- 背景：PM 入库已切 copy-direct 写模式（worker 单事务 plain COPY，幂等下沉到每文件一次
-- pm_files 唯一约束 + 文件内 last-wins 去重，见 internal/pm/metrics/copy_ingest.go）。原来挂在
-- 该 7 列唯一索引上的"每行自然键 ON CONFLICT DO UPDATE"补传幂等不再需要——它纯粹是写税：
-- 单 chunk 灌 9.2M 行时维护这一 7 列唯一 btree 占满 GSM 入库约 60% 的时间。
--
-- 实测（dev PG 8 核/10g）：满 GSM 600 文件（9.2M 行单 chunk）copy 模式 94.5s → 37.2s
-- （247K 行/s），且 PG CPU 反而更低（~780% → ~450%）。
--
-- 行身份仍由主键 pm_metrics_pkey (id, time) 保证（id 为 gen_random_uuid），删唯一索引不影响
-- 行唯一性；读路径走 device_time / path_time / time 索引，均不依赖本索引。
--
-- 配套代码改造（同 PR）：
--   - metrics.BatchInsert 改 plain INSERT（去 ON CONFLICT，无索引可冲突）；
--   - admin RecomputeKPIs（kpi.CalculateAndStore）改 scoped DELETE + INSERT 保持重算幂等；
--   - worker 统一走 copy 写模式（退役 upsert 写模式）。
-- 副带收益：plain INSERT 不再受 TimescaleDB 压缩 chunk 拒绝 ON CONFLICT（SQLSTATE 0A000）限制，
-- 迟到补传写入压缩 chunk 不再被降级跳过（issue #14 的写侧约束随之解除）。
DROP INDEX IF EXISTS uq_pm_metrics_natural;

-- +goose Down
-- 回滚需重建唯一索引；若此时表内已存在重复自然键行（copy 模式跨文件可能产生），需先去重
-- 才能重建（CREATE UNIQUE INDEX 会因重复键失败）。
CREATE UNIQUE INDEX IF NOT EXISTS uq_pm_metrics_natural ON public.pm_metrics
    USING btree (device_oui, device_sn, metric_path, granularity, end_time, "time", object_ldn);

-- +goose Up
-- PM 入库 CPU 优化（运营规模 10k 文件 / 单 chunk 突发）：删 pm_metrics 随机 uuid 主键 + 放宽
-- insert-triggered autovacuum。实测真实占比 10030 设备（30 GSM/7000 LTE/3000 NR，10.79M 行单 chunk）
-- 单次采集周期入库 304s → 76.7s（~4x），PG 入库峰值 CPU ~800% → ~350%。durable copy，0 失败/DLQ/迟到。
--
-- ① DROP 主键 (id, time)：id 是 gen_random_uuid 随机 surrogate，审计确认 0 外键引用、代码从不按 id 查
--    （读全走 device_oui/device_sn/metric_path/time）。每插一行往随机 uuid btree 塞随机键 → 满 chunk
--    随机翻页 + 页分裂，单 chunk 越大越贵，是 CPU 单一最大头（解释约 2/3 的提升：93.9s vs 304s）。copy
--    写模式幂等靠 pm_files 标记 + 文件内 last-wins 去重，pm_metrics 不需要任何唯一索引；id 列保留（仍由
--    worker 生成写入，只是不再建唯一索引）。读路径走 device_time / path_time / time 索引，均不依赖本主键。
-- ② 放宽 insert-triggered autovacuum（阈值 1000 → 5,000,000）：实测单 chunk 一次 10.79M 插入突发会触发
--    ~9 次 autovacuum + autoanalyze 抢 PG CPU；抬高阈值让突发期不频繁触发（仍周期清理；append-only 时序
--    chunk 的 stale stats 对时间桶查询影响小）。贡献额外约 18% 提升（93.9s → 76.7s）。
--
-- 注：autovacuum reloptions 设在 hypertable，新建 chunk 继承（每天新 chunk 即生效）；既有 chunk 保持默认，
--    随老化/压缩/保留淡出，不单独回灌以保持迁移简单低风险。
-- 已否决的第三个杠杆 commit_delay（PG 原生 group commit）：实测删主键后瓶颈已从 PG 转到 worker 解析/过滤，
--    commit_delay 只加提交延迟反而 -12%（76.7s → 86.1s），故不引入。
ALTER TABLE pm_metrics DROP CONSTRAINT IF EXISTS pm_metrics_pkey;
ALTER TABLE pm_metrics SET (
    autovacuum_vacuum_insert_scale_factor = 0,
    autovacuum_vacuum_insert_threshold = 5000000,
    autovacuum_analyze_scale_factor = 0,
    autovacuum_analyze_threshold = 5000000
);

-- +goose Down
ALTER TABLE pm_metrics RESET (
    autovacuum_vacuum_insert_scale_factor,
    autovacuum_vacuum_insert_threshold,
    autovacuum_analyze_scale_factor,
    autovacuum_analyze_threshold
);
-- 回滚重建主键：id 为随机 uuid，(id, time) 实际唯一；若历史数据存在重复 (id,time) 需先去重才能重建。
ALTER TABLE pm_metrics ADD CONSTRAINT pm_metrics_pkey PRIMARY KEY (id, "time");

-- +goose Up
-- pm_metrics chunk_time_interval 由 1 天 → 4 小时。运营规模（~1万站）下 1 天 chunk 过粗：
-- 10k LTE @15min 上报，单日 chunk ~13 亿行 / ~370GB，其上 3 个 btree 索引随 chunk 增大而维护变贵，
-- 正是 2026-06-12 入库压测实测的瓶颈（PG 打满 8 核在单 chunk 索引维护，见报告
-- docs/qa-report/pm-ingest-stress-report-20260612.md）。连续压测验证单 chunk 撑到 ~65M 行不退化；
-- 4 小时 chunk 把活跃 chunk 压到 ~2 亿行量级（10k LTE）/ 更小（混合或低规模），把写入热点保持在可控范围，
-- 是不动 schema、不上 space/SN 哈希分区（单机 TimescaleDB 不推荐）的最简写入侧杠杆。
--
-- 安全性：set_chunk_time_interval 幂等，且【只影响之后新建的 chunk】——存量 chunk 不重写、不锁表、零停机；
-- 对查询层完全透明（无 SQL/API 变化，前端按 granularity 路由：15min→pm_metrics、hourly+→rollup 表，
-- raw 查询均短范围 + 单设备 + 带 time 谓词，chunk exclusion 照常生效）。
-- 与既有压缩（compress_after 7d）/保留（drop_after 30d）策略不冲突：策略作用于 chunk，4h 后 drop_chunks
-- 更细粒度（过度保留更少）。autovacuum reloptions（000044）继续随新 chunk 继承。
--
-- 选 4 小时为后续统一默认：在低规模不至过度碎片化、在 1 万站量级又把活跃 chunk 压回压测验证过的可控区间，
-- 兼顾各规模的折中。更高规模（10k LTE 满载）仍建议叠加内存档（26-32G）/上报周期（15min→1h，数据率÷4）。
SELECT set_chunk_time_interval('pm_metrics', INTERVAL '4 hours');

-- +goose Down
-- 回滚到基线的 1 天间隔（同样仅影响之后新建的 chunk，存量不动）。
SELECT set_chunk_time_interval('pm_metrics', INTERVAL '1 day');

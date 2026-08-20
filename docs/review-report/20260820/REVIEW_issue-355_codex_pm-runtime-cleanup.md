# Review: Issue #355 PM 运行数据清理分批化

固定点：`main`

规格来源：GitLab Issue #355、父 PRD #352。

规范来源：`AGENTS.md`、`CLAUDE.md`、`omcgo/CLAUDE.md`、`CONTEXT.md`、`docs/project/dod.md`、`docs/ref/pm-metrics-knowledge.md`、ADR 0001/0002/0007。

## Standards

未发现阻塞或非阻塞规范问题。

- 数据删除 SQL 使用 Squirrel，CTE 候选集有硬 `LIMIT`，行级任务使用 `FOR UPDATE SKIP LOCKED`。
- 多 worker 通过稳定 PostgreSQL advisory lock 避免重复运行整轮维护。
- TSDB 索引和状态变化折回未封版基线，没有新增迁移编号。
- 日志和 Prometheus label 只有固定 target，不引入任务、设备或指标等高基数标签。
- context 控制整轮最大时长，VACUUM 默认关闭且只允许静态表白名单。

## Spec

未发现阻塞或非阻塞规格偏差。

- outbox、rollup outbox、replay source 和 terminal window 均改为有界清理。
- replay source 使用单 Timescale chunk 作为批次，避免在压缩 hypertable 上做大 DELETE。
- published、retired、orphaned 和 abandoned 终态均有清理路径；有效 active window 不在候选条件中。
- 仍有 `pm_aggregation_results` 的 published window 使用 `NOT EXISTS` 保护，不因任务清理被误删。
- 每轮有 batch、总时长、advisory lock、匹配索引、行数/错误/耗时/积压指标，并在有删除时执行 ANALYZE。
- Redis 和结果 retention 复用现有能力，没有重复实现或使用 `KEYS`。

结论：Standards 0 项，Spec 0 项；可以进入提交和 MR。

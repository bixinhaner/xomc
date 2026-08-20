# Review: Issue #353 PM recovery 无效任务版本兜底

固定点：`main`

规格来源：GitLab Issue #353、父 PRD #352。

规范来源：`AGENTS.md`、`CLAUDE.md`、`omcgo/CLAUDE.md`、`CONTEXT.md`、`docs/project/dod.md`、`docs/ref/pm-metrics-knowledge.md`、`docs/adr/0001-modular-monolith-not-microservices.md`、`docs/adr/0002-squirrel-pgx-not-orm.md`。

## Standards

未发现阻塞或非阻塞规范问题。

- SQL 继续使用 Squirrel 与 pgx，错误均补充上下文。
- TSDB 变化折回未封版基线迁移，没有新增迁移编号。
- recovery 每页只做一次主库版本状态批量查询；终态更新、Redis 清理和日志量都有硬上限。
- 新增测试覆盖未知版本、删除/停用/有效期边界、重叠窗口和版本定义分批清理。

## Spec

未发现阻塞或非阻塞规格偏差。

- 主库缺失、任务删除/停用、版本停用、计划结束和有效期越界均在 replay 前被拦截。
- 窗口终态记录原状态、原因和处理时间，并退出 active recovery 范围。
- Redis 使用 TSDB 索引化待清理队列和精确 key，不使用 `KEYS`；窗口状态与版本 definitions 均按批次清理并可重试。
- 没有删除或修改 `pm_aggregation_results`。
- 本机真实夹具证明未进入 replay、跨周期不重入，证据见 `docs/superpowers/evidence/2026-08-20-issue-353-pm-recovery-acceptance.md`。

结论：Standards 0 项，Spec 0 项；可以进入提交和 MR。

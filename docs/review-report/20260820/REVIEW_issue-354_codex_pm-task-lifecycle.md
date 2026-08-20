# Review: Issue #354 PM 聚合任务生命周期闭环

固定点：`main`

规格来源：GitLab Issue #354、父 PRD #352。

规范来源：`AGENTS.md`、`CLAUDE.md`、`omcgo/CLAUDE.md`、`CONTEXT.md`、`docs/project/dod.md`、`docs/ref/pm-metrics-knowledge.md`、ADR 0001/0002/0007。

## Standards

未发现阻塞或非阻塞规范问题。

- 主库 SQL 使用 Squirrel/pgx；复杂 missing-source 判定参数化且有事务锁与批次上限。
- 新字段折回未封版 schema 基线，没有新增迁移编号。
- 生命周期 reconcile 位于 PM 模块内部，不引入新的部署单元或跨域服务。
- 错误携带上下文，周期任务汇总记录 candidates/reconciled/retired/failed，没有高基数日志或指标。

## Spec

未发现阻塞或非阻塞规格偏差。

- 内置 source UUID 已稳定，reconcile 使用 source task ID 保存 streaming task。
- 内容未变时复用 current version；规则变化时创建新版本并关闭旧版本。
- source 删除、过期定义清理和废弃任务均只软退役 streaming task，版本历史不级联删除。
- planned end、cancel/resume 和 source/streaming 缺失由 source marker + enabled 状态周期扫描兜底；运行进度更新时间不会触发定义重算。
- source 消失后使用事务内 NOT EXISTS 判断，避免 source ID 快照竞态。
- 同 ID source 恢复会在原 task 下创建新版本，不复用已关闭版本。
- snapshot/recovery 周期性数据库扫描保证控制事件丢失后仍收敛 TSDB/Redis。
- 普通任务删除不触碰 `pm_aggregation_results`。
- 内置任务保存、自建任务创建/编辑/取消/删除均取得真实浏览器请求与数据库版本证据，并恢复或清理测试数据。

结论：Standards 0 项，Spec 0 项；可以进入提交和 MR。

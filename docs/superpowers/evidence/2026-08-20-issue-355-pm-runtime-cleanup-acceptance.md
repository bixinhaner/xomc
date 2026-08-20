# Issue #355 PM 运行数据清理分批化真实验收

## 测试目标

证明 PM streaming 历史运行数据按固定批次和总时限清理，能够跨多个批次完成积压，同时保留未消费、未过期、有效 active window 和仍有历史结果的 published window。

## 测试环境

- 日期：2026-08-20
- 环境：本机 OMC Docker 全栈
- 分支：`fix/355-pm-runtime-cleanup-batching`
- 基线：包含 MR !666 的 `main`
- TSDB：清空后由当前分支基线迁移重新创建
- 默认限制：每批 500，单轮最多 30 秒，VACUUM 默认关闭

## 测试数据

- 过期普通 outbox：501 条。
- 过期 legacy outbox：1 条。
- 保护 outbox：未消费 1 条、未过期 1 条。
- 过期 rollup outbox：501 条。
- 保护 rollup outbox：未消费 1 条。
- 过期终态窗口：orphaned 501 条、abandoned 1 条、无结果 published 1 条。
- 保护窗口：主库有效且 Redis 状态存在的 open 1 条、仍有 `pm_aggregation_results` 的 published 1 条。
- replay source：2 个过期日 chunk、1 个当前 chunk。

## 验收结果

- outbox 删除 502 条，剩余积压采样 0；两个保护哨兵均保留。
- rollup outbox 删除 501 条，剩余积压采样 0；未消费哨兵保留。
- replay source 每批只 drop 一个 chunk，共删除 2 个过期 chunk；当前 chunk 和当前数据保留。
- 终态窗口删除 503 条，剩余积压采样 0；有效 open 窗口保持 `open`，Redis meta 保持存在。
- 有历史结果的 published window 保留，对应 `pm_aggregation_results` 仍为 1 条。
- 四类清理在约 0.39 秒内完成；502/501/503 均超过单批 500，证明同一轮可由多个有界批次追赶完成。
- `pm_aggregation_outbox`、`pm_aggregation_rollup_outbox`、`pm_aggregation_windows` 的 `last_analyze` 均已更新。
- worker 启动日志没有 runtime cleanup 错误；四类 backlog 指标均为 0。

## 自动化验证

- SQL 形态测试验证 outbox、rollup outbox、terminal window 均使用 CTE、`LIMIT` 和 `FOR UPDATE SKIP LOCKED`。
- replay source 测试验证每次只选择一个过期 Timescale chunk。
- 配置测试验证 batch 硬上限为 5000、间隔和总时长有上下界、VACUUM 必须显式开启。
- `go test ./internal/pm/... ./cmd/worker ./cmd/migrate -count=1`：通过。
- `go build ./...`：通过。

## Redis 与结果保留边界

- published Redis 窗口继续由现有 bounded Redis sweeper 清理。
- retired/orphaned Redis 状态由 Issue #353 的 TSDB 索引化待清理队列处理。
- 正常过期 Redis key 继续依赖按粒度 TTL；没有新增全量 `KEYS`。
- `pm_aggregation_results` 继续由现有 PM retention cleanup 处理，本 Issue 未实现第二套结果删除。

## PM 入口矩阵

本次不修改 PM 统计口径、页面聚合查询、定时聚合计算、自定义聚合计算、导出或启用指标旁路。真实验证覆盖数据库、worker 日志和 Prometheus 指标，页面浏览器验证不适用。

## 清理方式

验收后删除 payload、entity_key、metric_id 或任务名称带 `ISSUE355` 的主库、TSDB 与 Redis 夹具。

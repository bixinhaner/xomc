# 0007 — PM 跨粒度聚合由上游完成事件链式触发

## Status

Accepted。决策于 2026-07-09 落地。

## Context

PM 自然桶聚合按 `pm_metrics -> pm_metrics_hourly -> pm_metrics_daily -> pm_metrics_weekly/monthly` 逐级生成。此前 hourly、daily、weekly、monthly 各自按 cron 入队；当 hourly 的最后一个源桶与 daily 同时在零点后入队时，daily 可能先执行，导致日桶固定漏掉业务日最后一小时。

单纯把 daily cron 延后几分钟只能降低竞态概率，不能证明上游源桶已经成功提交。保留每个粒度独立 cron 也会让启动补跑、重试和组级聚合维护多套触发游标，长期容易漂移。

## Decision

PM 跨粒度聚合采用链式触发：

- 只保留 hourly 独立 cron 和 hourly catchup。
- hourly 设备级桶成功后，若该桶是业务日最后一个小时桶，则触发同窗口 daily 设备级桶。
- daily 设备级桶成功后，若该桶是业务周或业务月最后一天，则触发对应 weekly 或 monthly 设备级桶。
- 每个设备级桶成功后继续触发同粒度设备组桶；组级聚合不拥有独立 cron。
- monthly 直接从 daily 聚合，不从 weekly 聚合，避免 ISO 周和自然月边界不对齐。
- 所有窗口继续使用业务时区自然边界，并保持 `[start_time, end_time)` 半开区间语义。

## Consequences

**正向**：

- daily 不会早于目标业务日最后一个 hourly 源桶成功完成。
- 停机补跑只需补 hourly 漏桶，下游粒度由成功事件继续推进，触发链更单一。
- chain 入队失败可以让上游任务失败并重试，避免下游桶静默丢失。
- monthly 的来源与自然月边界一致，不再受跨月周桶影响。

**代价 / 约束**：

- async job 入队需要支持 bucket 级幂等，避免重试或补跑重复排队。
- 聚合写入必须保持 UPSERT 幂等，允许上游 retry 后重复计算同一 bucket。
- 本决策只消除“最后源桶未完成”的竞态；中间源桶因采集或历史失败缺失，不作为下游聚合的阻塞条件。
- 历史已成功但缺数的 daily/weekly/monthly 是否补算，是独立运维决策，不随本决策自动改写。

## Rejected Alternatives

- **固定延迟 daily/weekly/monthly cron**：实现简单，但无法提供源桶成功完成的正确性保证。
- **要求 daily 前检查 24 个 hourly 桶全部成功**：能发现更多缺失，但会把中间历史缺失或采集异常扩大成下游长期阻塞，不符合本次修复目标。
- **monthly 从 weekly 聚合**：减少读取行数，但自然周可能跨月，无法稳定表达自然月窗口。

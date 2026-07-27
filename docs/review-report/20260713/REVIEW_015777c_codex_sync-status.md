# Review: sync-status PG count

结论: PASS

## 范围

- 后端任务状态计数从 Redis 队列长度转向 PG open task 统计。
- 参数树 sync-status 仅统计近期未完成的 sync-gpv 任务，并在 PG 查询失败时回退到队列长度。
- Redis Pop 清理缺失详情或损坏详情的队列成员，避免脏队列阻塞后续任务。
- 参数树和快速设置页调整同步中/失败态展示与请求启用时机。

## 发现

- CRITICAL: 无。
- WARNING: 无阻断项。`CountOpenSyncGPVByDevice` 使用 24h 近期窗口，符合当前同步状态展示语义，但后续如果存在超过 24h 的超长同步场景，需要重新评估窗口。
- INFO: `LatestSyncGPVSummaryByDevice` 通过 root command_key 和 30s 边界识别同一轮同步，已补 retry 与旧 run 回归测试。

## 验证

- `go test ./internal/task ./internal/device ./internal/provision ./internal/acs` 通过。
- `npm run typecheck` 通过。
- `git diff --check` 通过。

## 风险

- 无数据迁移。
- 无公开 API 字段破坏；`pending_commands` 语义更精确，前端仍按原字段消费。

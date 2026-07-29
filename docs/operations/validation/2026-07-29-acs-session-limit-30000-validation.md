# ACS 30,000 全局会话上限验证

## 部署

- 服务器：`172.24.224.197`
- 版本：`0.1.6-20260729-1352`
- 配置：`session.max_concurrent: 30000`
- 升级保留现有数据卷，健康检查 25/25。

## 14:00–14:06 结果

- `admission denied`、设备限流、PM 背压拒绝和会话交换失败均为 0。
- Redis 准入槽位直接采样峰值 14,123；Prometheus 活跃会话峰值 13,587，未达到
  30,000。
- Inform 峰值约 368.38 次/秒。
- PM 队列峰值：pending 109、ack pending 128、oldest 16.8 秒、redelivered 0；
  窗口结束全部归零。
- 聚合发布 outbox 直接采样峰值 690，随后归零。
- Worker 短时接近 8 核配额，随后回落到约 2.6–3.8 核；未形成队列持续积压。
- 宿主 CPU 峰值 53.65%，内存峰值 28.17%。
- `sda` I/O 活动时钟峰值 81.07%，平均等待峰值 0.70 ms，无磁盘饱和。
- Redis 使用内存从约 1.15 GiB 回落到 941 MiB，`maxmemory=2 GiB`；AOF 写入和最近
  重写状态正常，无 pending/delayed fsync。数据盘使用率 8%。
- 主库和 TimescaleDB 无超过 30 秒长查询、无慢查询增量；窗口结束聚合 outbox 为 0。
- PM 文件成功处理平均约 47.52 文件/秒，处理 P95 约 479 ms。
- 小时、天、周 LTE 网络聚合各有 122 个完整指标，`complete=false` 和 missing slots
  均为 0。
- firing 告警只有预期的 `DeadMansSwitch` 和既存
  `PMCountersDiscoveredOutsideLibrary`。

## 结论

30,000 上限消除了此前 10,000 上限导致的整点准入 503。14:00 的主要瞬时瓶颈转移到
Worker CPU，但队列、最老消息和 outbox 均在窗口内排空，当前容量可以承载本轮负载。

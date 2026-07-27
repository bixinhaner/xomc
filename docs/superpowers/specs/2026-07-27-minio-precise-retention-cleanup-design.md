# MinIO 原始文件精确分批清理设计

## 状态

- 日期：2026-07-27
- 状态：已确认，待开发计划
- 目标：取消 PM/MR 原始文件对 MinIO ILM 全桶扫描的依赖，改为按数据库精确路径持续、分批、自动限速删除。
- 保留语义：`minio.retention.raw_object_days` 继续作为原始对象保留期，默认 60 天；到期后允许最多 24 小时删除宽限。
- 非范围：修改 PM/MR 时序数据保留期、历史对象打包、对象存储迁移、关闭 MinIO 内部 scanner 进程。

## 背景

10000 基站每 15 分钟上报一个 PM 文件时，每天约产生：

```text
10000 × 96 = 960000 objects/day
960000 ÷ 86400 ≈ 11.1 objects/second
```

MinIO 当前使用 bucket lifecycle scanner 遍历 `pm-files`、`mr-files`，再判断对象是否超过
`minio.retention.raw_object_days`。在百万级小对象场景，这种全桶扫描持续消耗随机读 IOPS。
gzip 压缩由 ACS/worker 完成，与 scanner 无关。

当前 MR cleaner 只删除 `mr_files` 数据库记录，对象本体完全依赖 MinIO ILM。直接移除 ILM
会导致对象永久增长，因此必须先提供应用侧精确删除能力。

## 方案比较

### A. 持续低速精确删除（采用）

从 `pm_files`、`mr_files` 查询到期对象的精确 `minio_path`，单实例、单并发、批量删除。
删除能力随设备规模和实际文件产生速率自动调整，并受业务压力保护。

优点：

- 不执行 `ListObjects`，不为寻找过期对象遍历整个 bucket；
- 删除压力均匀分散到全天；
- 每个对象有明确成功/失败状态，可恢复、可审计；
- 能在设备规模增长时自动调整清理能力。

### B. 固定夜间窗口分批删除（不采用）

虽然单批可控，但白天到期对象集中积累，夜间仍形成删除高峰，并与数据库维护任务竞争。

### C. 保留 ILM 并使用 `slowest`（仅作回退）

改动小，但 scanner 仍需遍历全部对象，不能消除小对象随机读根因。

## 架构

worker 增加独立的 `RawObjectCleanupRunner`，包含三个边界清晰的组件：

1. `CandidateRepository`
   - 分别从 `pm_files`、`mr_files` 按 `(collect_time,id)` keyset 顺序读取到期记录；
   - 不使用 OFFSET，不执行 MinIO 列表查询；
   - 单次最多返回 `batch_size` 条。
2. `ObjectDeleter`
   - 使用 MinIO/S3 批量删除接口删除精确 key；
   - 返回逐对象结果，单个对象失败不回滚其他成功对象；
   - 删除不存在的对象视为成功，保证崩溃重试幂等。
3. `RateGovernor`
   - 根据设备规模、最近一小时实际新增文件速率和业务压力计算下一批等待时间；
   - 只控制速率，不改变保留期；
   - 任何信号异常都回落到保守速度，不自动无限提速。

多 worker 部署时通过 PostgreSQL advisory lock 保证同一时刻只有一个 runner 工作。数据库
事务不得跨越 MinIO 网络请求：先读取候选并结束查询，再删对象，最后批量更新成功/失败状态。

## 数据状态

`pm_files`、`mr_files` 增加：

```text
raw_deleted_at            timestamptz null
raw_delete_attempts       integer not null default 0
raw_delete_next_attempt_at timestamptz null
raw_delete_last_error     varchar(512) null
```

候选部分索引：

```text
(collect_time,id)
WHERE raw_deleted_at IS NULL
```

另为 `mr_files(created_at)` 增加普通索引；`pm_files` 已有 `created_at` 索引。最近一小时实际
新增速率只做索引计数，不扫描对象存储。

查询条件：

```text
collect_time < now() - raw_object_days
AND raw_deleted_at IS NULL
AND (
  raw_delete_next_attempt_at IS NULL
  OR raw_delete_next_attempt_at <= now()
)
```

成功对象使用一次批量 UPDATE 设置 `raw_deleted_at=now()` 并清空错误。失败对象增加 attempts，
写入截断后的错误摘要，并按 1m、2m、4m……最多 1h 计算下一次重试时间。

不能在对象删除后立即无条件删除 `pm_files`：它是 PM 入库幂等锚点。文件元数据由同一 runner
使用相同批次上限和速率低速清理，但必须满足：

- 原始对象已经删除；
- 对应 PM/MR 时序数据也已超过自己的保留期；
- PM 不再存在引用该 `source_file_id` 的 anchor；
- MR 不再存在引用该 `file_id` 的 record。

PM 元数据删除事务同时清理已经发布且不再被 counter rollup 引用的
`pm_aggregation_outbox`、对应 `pm_ingest_batches`，最后删除 `pm_files`；任一依赖仍存在则
整组跳过。MR 在 `mr_records` 已不存在后删除 `mr_files`。不做无条件级联删除。

这样 MinIO 原始对象保留期与时序数据库保留期继续独立配置，也不会提前破坏 PM 幂等。

## 自动速率

每 5 分钟重新计算：

```text
predicted_rate =
  sum(每个已启用 PM 任务的目标设备数 ÷ 该任务上报周期秒数)

observed_rate =
  (最近一小时新增 pm_files + mr_files) ÷ 3600

base_rate = max(predicted_rate, observed_rate)
target_rate = clamp(ceil(base_rate × 1.5), 5, 100)
batch_interval = batch_size ÷ target_rate
```

默认：

- `batch_size=100`
- `min_rate=5 objects/s`
- `max_rate=100 objects/s`
- `rate_headroom=1.5`
- `recalculate_interval=5m`
- `expiration_lag_slo=24h`

10000 台、15 分钟 PM 周期时，预测约 11.1 objects/s，目标取约 20 objects/s，即约每 5 秒
删除一批 100 个。每日能力约 172 万，能够覆盖 96 万 PM 对象并给 MR 和失败重试留出余量。

任务目标设备数取启用任务的去重设备快照；同一设备被多个独立任务调度时按任务分别计入，因为
每个任务都会产生文件。任务配置暂不可用时，冷启动预测使用在线设备数和系统默认 PM 周期；
最近一小时实际新增速率是运行期主要依据，因此设备离线、PM 周期变化或 MR 量变化都会自动反映。

## 业务压力保护

RateGovernor 对目标速率乘以最严格的保护系数：

- 15 分钟 PM 边界前后各 2 分钟：`0.25`
- NATS PM/聚合 consumer 出现 pending 或 ack pending 持续增长：暂停
- worker/宿主 CPU > 80%：`0.5`
- 磁盘 await > 20ms 或平均队列 > 1：`0.5`
- 磁盘 await > 40ms 或平均队列 > 2：暂停
- MinIO 批量删除出现 503、限流或超时：指数退避，最长暂停 5 分钟
- 单批删除耗时持续超过 2 秒：下一周期速率减半

CPU/磁盘信号从已部署的 node-exporter 指标读取；完整监控未启用或指标不可达时，不中断清理，
但目标速率不得超过保守上限 20 objects/s。NATS 状态直接通过现有 JetStream 管理接口读取，
不依赖 Prometheus。

到期对象最老延迟超过 24 小时时记录告警，但不突破 `max_rate`，不以牺牲实时业务换取追赶速度。

## 调度和关闭

- runner 全天运行，不使用夜间集中 cron；
- 启动后随机等待 0–60 秒，避免多个后台任务同时启动；
- 每轮最多处理一批，之后按 governor 计算结果等待；
- context 取消后不再领取新批次，等待当前批量请求完成或超时；
- 单批 MinIO 请求超时 10 秒，数据库状态更新超时 5 秒；
- advisory lock 丢失后立即停止领取新批次。

## ILM 与 scanner

应用侧精确清理通过业务验证后：

1. app 不再给 `pm-files`、`mr-files` 下发原始文件生命周期规则；
2. 移除当前 OMC 创建的 raw-file lifecycle 规则；
3. `MINIO_SCANNER_SPEED` 从 `slow` 调整为 `slowest`；
4. scanner 不做非官方关闭。MinIO 仍可能将其用于内部使用量统计、复制和 healing，但不再依赖
   scanner 查找 PM/MR 过期对象。

回退时可重新应用原有 lifecycle 规则，精确清理器停止领取新批次；两种删除均为幂等。

## 可观测性

新增指标：

```text
omc_raw_cleanup_candidates
omc_raw_cleanup_deleted_total{kind="pm|mr"}
omc_raw_cleanup_failed_total{kind,reason}
omc_raw_cleanup_rate_target
omc_raw_cleanup_batch_duration_seconds
omc_raw_cleanup_oldest_expired_age_seconds{kind}
omc_raw_cleanup_paused{reason}
```

告警：

- 最老到期对象延迟超过 24 小时；
- 连续 30 分钟没有成功删除且存在候选；
- 删除失败率 15 分钟内超过 5%；
- 清理器 advisory lock 连续 10 分钟无人持有。

## 验证

单元测试覆盖：

- 设备预测、实际速率、上下限和批次间隔；
- 15 分钟边界降速；
- CPU、磁盘、NATS 和 MinIO 错误的降速/暂停；
- 部分成功、对象不存在、重试退避；
- PM/MR 元数据不会早于时序数据删除。

集成测试覆盖：

- 候选 keyset 分页无重复、无遗漏；
- MinIO 删除成功后状态落库；
- 在“对象已删、状态未写”处模拟崩溃，重启后幂等恢复；
- 两个 runner 竞争时只有 advisory lock 持有者执行；
- 10000 设备等价速率下，删除能力大于新增速率且业务队列无新增积压。

部署验证采用两阶段：

1. shadow 模式只计算候选和目标速率，不执行删除，观察 24 小时；
2. 开启真实删除但保留旧 ILM 规则 24 小时作为兜底，验证后再移除 ILM 并把 scanner 调为
   `slowest`。

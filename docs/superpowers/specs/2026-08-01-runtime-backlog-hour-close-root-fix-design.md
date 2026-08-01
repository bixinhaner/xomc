# 运行时积压、主库写放大与小时窗口准时发布根治设计

日期：2026-08-01

状态：总体方案已确认，待书面设计复核

## 1. 现场证据

20,000 台设备清洁部署后的当前问题不是 PM 主消费队列阻塞，而是三条相互放大的
后台链路：

- `pm-workers`、`pm-registration-wait` 和小时 rollup consumer 均为 0 pending；
- `device-mgr-periodic` 曾积压超过 140 万，观测窗口内生产约 427 条/秒、消费约
  975 条/秒，当前虽在回落，但 `ack_pending=1000` 已达到上限；
- `param-sync-results-pull` 积压约 102 万，观测吞吐约 12.25 条/秒；
- 主 PostgreSQL 使用约 7.8~11.1 个 CPU 核，27.3 秒内产生约 18.1 GiB
  PostgreSQL buffer miss，区间缓存命中率约 62.5%，并出现
  `device_parameters` DELETE/INSERT 的 transactionid 锁等待；
- 10:00 小时窗口共有 20,003 个实体窗口，第一条在 10:12:21 发布，P50 为
  +20.61 分钟，P95 为 +25.90 分钟，最后一条为 +26.87 分钟；
- 14,347 条 PM 死信全部产生于 02:36~03:04，错误为设备注册宽限 30 分钟后仍
  找不到设备，03:04 后没有新增。

## 2. 已确认业务决策

### 2.1 非法启动期 PM

`pm.file.received` 到达后，设备在 30 分钟注册宽限期内始终不存在，则该文件是
非法数据，不具备重放价值：

- 删除 MinIO 原始对象；
- 不写 `dead_letters`；
- 不写 `pm_files`、Counter、KPI 或聚合窗口；
- 不计入小时、日、周覆盖率的“应收槽位”；
- 仅保留按原因分类的 Prometheus 丢弃计数和不含完整 SN/对象路径的脱敏日志。

删除对象成功或对象已经不存在后才能 ACK。删除失败返回可重试错误，不能用 ACK
掩盖存储泄漏。当前 14,347 条同类历史死信在新版本部署后执行一次性清理；这是明确
的数据切点操作，不作为修复是否成功的证据。

### 2.2 小时关闭语义

12 分钟是迟到数据接收宽限期。业务验收含义为：

- `hour_end + 12m` 前结果可以持续修订，不对外声明自然周期最终完成；
- `hour_end + 12m30s` 内完成最终版本切换；
- Dashboard 在一次版本切换后看到该小时同一版本的全部可用结果，不看到逐设备
  发布形成的 15 分钟长尾；
- 12 分钟后到达的合法迟到数据必须创建新 revision 并触发增量修正，不能静默丢弃。

## 3. 方案选择

### 方案 A：只增大并发并清空积压

不采用。它能暂时缩短队列和小时发布尾部，但保留了每个结果重复聚合计数、每个
心跳重复分组、每个窗口单独发布的根因。

### 方案 B：三条有界链路逐项根治

采用：非法 PM 明确终结；周期事件去掉重复副作用；参数同步改为增量计数和一次性
收尾；小时窗口使用预结算 revision 和原子发布水位。每条链路都可独立测试、部署和
回滚。

### 方案 C：重建设备事件、参数同步和 PM 聚合三套基础设施

不采用。本轮已有可复用的 durable consumer、设备分片结果消费者和流式 PM 窗口，
全面重建会扩大上线风险，且不是解决当前证据所必需。

## 4. 详细设计

### 4.1 非法 PM 终结器

在 `PMCollector` 的设备解析边界增加窄接口 `RawObjectDiscarder`。处理顺序为：

1. 设备查询返回 `ErrDeferred` 且事件年龄未满 30 分钟：继续进入现有
   `pm-registration-wait` durable；
2. 已满 30 分钟：调用 `Discard(ctx, bucket, object)`；
3. `RemoveObject` 成功或确认返回 `NoSuchKey`：增加
   `pm_files_discarded_total{reason="device_not_registered"}`，写脱敏日志并返回 nil；
4. `NoSuchBucket` 或其他 MinIO 错误：返回带上下文的 `ErrDeferred`，让 deferred
   consumer NAK 后重试并触发存储告警；缺桶属于配置故障，不能伪装成文件已删除；
5. 该分支不返回普通错误或 `ErrPermanent`，因此不会进入通用 runner 的死信写入路径。

历史死信清理只选择以下严格条件：

```text
source_module = pm
source_subject = pm.file.deferred
error 包含 device registration grace exceeded
```

逐条解析 payload、幂等删除对象后再删除对应死信行；单条失败不影响其他条目，并输出
成功、对象已不存在、失败三类计数。

### 4.2 周期事件去副作用

保留现有 `BatchInformProcessor` 的“同一 worker flush 周期内按 SN 只保留最新更新”以及
`device_parameters` unchanged guard，不重复实现第二套合并器。

删除 `InformHandler.handlePeriodic` 的心跳自动分组调用和 fire-and-forget goroutine。
设备分组已有三条可靠触发：

- `device.registered`；
- `device.attributes.changed`（LAC/TAC 等分组字段变化）；
- `GroupMatchEngine` 每小时全量兜底。

因此每次 Periodic Inform 再做一次匹配是重复副作用。移除后不改变首次归组、属性变化
归组和规则编辑后的全量重评估。

周期消费者继续按设备稳定分片；增加生产、处理、合并覆盖、批处理通道满、数据库 flush
时长指标。线上保护要求通道满时产生高优先级告警，不能把 drop 只写 WARN。

### 4.3 参数同步结果增量收敛

保留当前按设备哈希分片，保证同一设备的结果串行；根治以下两个重复成本：

1. 每个结果都执行一次 `loadAuthoritativeRunCounts` 聚合任务/结果表；
2. full sync 最终完成时按 coverage 循环执行多条 DELETE。

新的处理边界：

- `parameter_sync_task_results` 幂等插入成功后，在同一事务中原子增加 run 的
  `processed_task_count`；重复事件不增加；
- 终态任务已在 `loadTaskValues` 验证，因此正常结果可原子增加
  `terminal_task_count`；
- 只有计数接近 `expected_task_count`、进入 cancelling、恢复器修复或状态不一致时，
  才执行一次权威聚合校验；普通中间结果不再重复 COUNT；
- full sync 收尾把所有 complete coverage 谓词合并为一条带 `device_id` 前导条件的
  DELETE，并保留现有 LIKE 前缀 + 正则精确匹配；
- staging merge、缺失参数清理、run/request 终态和 terminal outbox 仍在同一事务，
  不牺牲原子性；
- 先以现有 8 个设备分片验证。只有单事务 P95 达标且主库有余量，才逐级调到 16/32，
  不以并发掩盖慢 SQL。

冷启动 full sync 的 GPV batch 默认从 50 调到 100，仍在现有建议范围内，以减少新环境
任务和结果事件数量。9005 隔离路径继续单独成批，不因扩大普通 batch 而改变错误隔离语义。

### 4.4 小时窗口预结算与原子发布

沿用现有事件驱动逐级聚合，不回查原始 PM，不恢复定时全表扫描。

为同一个 `(task_version_id, granularity, window_end)` 引入 publication 水位：

- 窗口数据仍随每个 15 分钟槽持续写 Redis 聚合状态；
- 小时结束后进入 `preparing`，结算器按批领取实体窗口并生成不可见的目标 revision；
- 聚合结果业务唯一键包含 revision，使 preparing revision 与当前 published revision
  可同时存在，预结算不会覆盖线上正在读取的结果；
- 迟到槽在 12 分钟宽限期内更新状态并把受影响实体标记为 dirty，只重算受影响实体；
- 到水位时先完成最后一轮 dirty entity，再在一个短事务中把 publication revision 从
  `preparing` 切换为 `published`；
- Dashboard 和下级 rollup 只读取 publication 指向的 revision，因此不会看到逐实体
  发布长尾；
- 水位后合法迟到事件创建下一 revision，增量修正完成后再次原子切换；
- rollup outbox 同样携带 publication revision，只有该 revision 原子发布后才允许向
  日窗口和其他维度传播，避免下级聚合提前消费半成品。

结算领取使用现有 lease/SKIP LOCKED；单次最多 32 个实体，计算并发先从 4 提升到 32，
TSDB 连接预算保持 96。并发是预结算吞吐保护，publication 水位才是消除页面长尾的
正确性边界。

日、周、月同样登记 publication；它们保持现有 grace，并在到期周期内完成 prepare +
publish，从而所有粒度共用同一读取契约。日、周进行中结果继续消费已发布小时/日 revision；publication 记录明确保存版本有效
区间、应有槽位、已收到槽位、自然周期是否完整，避免把旧 revision 当作当前完整结果。

## 5. 失败处理与回滚

- MinIO 丢弃失败：消息重试，原文件和事件均保留；不写死信。
- 参数同步事务失败：NATS 不 ACK；幂等 result key 防止重复计数。
- 参数同步计数不一致：回退权威聚合校验并修正 stored counters，记录 repair metric。
- 预结算失败：publication 保持旧 revision 或 preparing，Dashboard 不读取半成品。
- publication 切换失败：结果仍不可见，下轮重试短事务；不需要删除已写 revision。
- 回滚应用版本时，旧代码忽略 publication 新字段但数据库基线兼容；部署验证期间不删除
  旧 revision，确认稳定后再按 retention 清理。

## 6. 可观测性与线上保护

新增或补齐：

- 非法 PM 丢弃数、MinIO 删除失败数；
- device periodic 生产/完成/合并覆盖/通道满/oldest age；
- 参数同步 result process P50/P95/P99、权威计数 fallback、单 run finalize 时长；
- 主库慢 SQL、锁等待、buffer hit、临时文件和事务率；
- PM publication preparing age、dirty entities、revision switch latency、最终版本覆盖率；
- Dashboard 查询超时、并发限制和慢查询告警保持启用。

## 7. TDD 与验证门禁

### 自动化

- 每个行为先写失败测试并确认按预期失败，再写最小实现；
- 目标包至少覆盖 `internal/pm/collector`、`internal/device`、
  `internal/paramsync`、`internal/pm/stream`；
- `cd omcgo && go build ./... && go test ./... -count=1`；
- 现有 PM stream e2e、发布包构建和 Docker health 门禁全部通过。

### 20,000 设备部署验收

- 非法未注册 PM：原文件删除成功，`dead_letters` 不新增，PM/小时/日/周覆盖率不计入；
- `pm-workers`、`pm-registration-wait` 稳态 pending 为 0；
- device periodic 在峰值下净增长不大于 0，`ack_pending` 不持续顶到上限，历史积压
  在一个小时内清零；
- parameter-sync results 在无新冷启动洪峰时持续下降，目标一小时内清零；新部署不再
  产生百万级结果债务；
- 参数同步 task/result/run/request 最终收敛，running 无永久滞留，server-attributable
  expired 低于 0.1%；
- 主库稳态 CPU 低于分配核的 70%，5 分钟窗口 buffer hit 大于 95%，lock wait 为 0，
  宿主机 iowait P95 小于 5%；
- 小时 publication 在 `hour_end + 12m30s` 内切换，版本内结果一致；迟到修正生成新
  revision；
- 当前日/周进行中结果在下一次 5 分钟 Dashboard 定时刷新时可见，覆盖率与源窗口一致；
- ACS 502/503/504 为 0，容器 restart/OOM 为 0，Prometheus 无业务告警。

## 8. 实施与交付顺序

1. 获取最新 `main` 并合入当前隔离分支，先跑基线测试；
2. 非法 PM 终结器及一次性历史清理；
3. 移除周期心跳重复分组；
4. 参数同步增量计数和单语句收尾；
5. 小时预结算 revision 与 publication 水位；
6. 完整自动化测试、构建、部署；
7. 清理已授权的非法历史 PM 数据，执行 20,000 设备验收；
8. 只有全部门禁通过才提交 MR；MR 通过后合入，再做合入版本部署复验。

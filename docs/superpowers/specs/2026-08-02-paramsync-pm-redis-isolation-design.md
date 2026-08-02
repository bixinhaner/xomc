# 参数同步收敛与 PM/KPI Redis 隔离设计

## 背景

2026-08-02 对 20,000 台设备环境进行实时巡检后，主业务链、ACS、PM 文件采集、NATS、CPU、内存和磁盘均未出现宿主机级饱和，但确认了两个需要根治的问题：

1. 参数同步存在大量已经具备终态条件、却仍停留在 `executing` 的 run。
2. 小时窗口在结束 12 分钟后正确开始关闭，但约 20,003 个设备和汇总窗口全部发布完成需要额外约 12 分钟，网络级 KPI 到达时间约为 `window_end + 24m`；同时 PM 聚合和 ACS/任务队列共用一个 Redis，小时边界负载曾使 ACS Redis pool 短时达到 100/100。

本设计同时处理状态机正确性、Redis 资源隔离和关窗发布吞吐。只拆 Redis 不能消除 TSDB 逐窗口发布开销，因此三部分必须作为同一版本的完整改造交付。

## 现场证据

### 参数同步

- 活跃 run 约 1,044 个。
- 其中 500 个满足 `expected_task_count = terminal_task_count = processed_task_count`，但状态仍为 `executing`。
- 其余 run 记录的未完成任务合计约 1,200 个，而数据库中真实活跃的参数同步设备任务只有约 26 个。
- 最近 30 分钟没有新增设备任务失败或参数同步 run 失败。
- 当前 `ReconcileRunCounts` 每 30 秒只按 UUID 游标扫描 20 个历史 run，修正计数后不会推进 run 终态。

### PM 聚合

- 10:00–11:00 小时窗口在 11:12:26 开始发布，符合 12 分钟关闭水位。
- 20,003 个窗口全部发布到 11:24:04，网络 KPI 在 11:24:03 生成。
- finalize concurrency 为 32。
- TSDB 累计统计中，逐窗口删除旧 rollup 的 SQL 已执行约 28.8 万次，平均耗时约 94ms；revision 1 没有历史 revision，仍执行删除。
- 日/周重算快照查询累计 20 次，平均约 232 秒，说明全周期物化仍是恢复路径的潜在重负载。

### Redis

- 单 Redis 峰值约 40,000 ops/s。
- ACS primary pool 曾短时达到 100/100，虽然没有产生 503 或准入拒绝，但已证明 PM 峰值会影响核心业务连接池。
- Redis 当前同时承载 ACS 会话、准入、cmdq/taskq、告警、设备缓存、PM 在线窗口和 KPI L2 缓存。
- 使用 Redis 逻辑 DB 不能隔离单线程命令执行、内存、AOF 和连接处理，必须使用独立 Redis 进程。

## 目标

1. 参数同步最后一个任务终态后 30 秒内，run 必须进入 `succeeded`、`failed` 或 `cancelled`。
2. `ready_but_not_finalized` 必须长期保持为 0。
3. PM/KPI Redis 与 ACS、任务队列和普通业务缓存物理隔离。
4. 小时窗口在 `window_end + 12m ± 30s` 开始关闭，20,000 设备及网络/产品/设备组结果在 `window_end + 15m` 前完成发布。
5. 拆分过程不得丢失活动窗口、PM 文件、NATS 事件或现有业务数据。
6. 首页继续只读取现有聚合数据，小时/天/周均保留，仅每 5 分钟定时刷新。

## 非目标

- 本阶段不把 PM 聚合拆成新的独立微服务。
- 不引入 Redis Cluster、Sentinel 或第三方队列。
- 不改变 KPI 公式、Counter 映射、窗口自然周期定义或 Dashboard API 契约。
- 不通过清库、删除历史 run 或丢弃合法 PM 数据掩盖问题。

## 总体架构

系统新增第二个 Redis 实例和 Worker 内的第二个 Redis client：

```text
redis-core
├── ACS session / admission
├── cmdq / taskq
├── device cache
├── alarm state
└── common business cache

redis-pm
├── PM streaming aggregation windows
├── PM aggregation locks and watermarks
├── late-event correction state
├── PM Redis sweeper state
└── KPI router L2 cache
```

`workerInfra.Redis` 保持为核心业务 Redis；新增 `workerInfra.PMRedis`。PM 聚合组件和 KPI Router 明确依赖 `PMRedis`，其他 worker 模块继续依赖 `Redis`。如果 `pm_redis` 未配置，`PMRedis` 回退到 `Redis`，保证旧配置可启动和滚动升级。

## 参数同步状态收敛

### 单一收敛入口

新增事务级统一函数 `ConvergeRunTx`。结果事件处理器、主动维护和故障恢复都调用同一实现，禁止各自复制状态推进规则。

收敛事务执行以下步骤：

1. `SELECT ... FOR UPDATE` 锁定目标 run。
2. 从 `device_tasks` 和 `parameter_sync_task_results` 计算权威计数：实际任务数、终态任务数、已处理结果数和失败数。
3. 覆盖 run 上的四个计数器。
4. 根据权威状态进行单次状态转换：
   - 全部终态并全部处理，失败数为 0：`succeeded`。
   - 全部终态并全部处理，失败数大于 0：`failed`，保留规范化失败摘要。
   - run 已进入取消流程：复用现有取消收敛规则进入 `cancelled` 或 `failed`。
   - 仍有真实活跃任务：保持 `waiting_device`/`executing`。
5. 设置 `completed_at`、增加 version，并通过唯一 dedupe key 写入 terminal outbox。
6. binding、provisioning task 和 projection 继续通过现有 durable outbox/投影器推进。

终态转换必须具备幂等性；同一 run 被事件消费和维护任务并发处理时，只允许一个事务成功推进，其他事务读取终态后直接返回。

### 活跃 run 优先修复

新增 `ReconcileActiveRuns`，只扫描：

```sql
status IN ('planning','enqueuing','waiting_device','executing','processing','cancelling')
```

按 `(started_at, id)` 最老优先，每批 200，使用 `FOR UPDATE SKIP LOCKED`。每一行都执行权威计数和 `ConvergeRunTx`，避免再次出现“只修计数不推进状态”。

原有全历史 UUID 扫描降为低频完整性审计，不再占用 30 秒业务维护循环。历史审计只修复终态历史计数，不参与在线 run 的完成时延。

### 缺口分类

收敛逻辑必须区分三类缺口：

1. **计划未落任务**：run 记录期望任务，但没有相应 `device_tasks`。按照现有 stalled dispatch 规则失败并进入退避，不能永久 executing。
2. **任务已终态但结果缺失**：从 `device_tasks` 重建幂等结果，再进入收敛事务。
3. **真实设备任务仍活跃**：保持运行态；达到任务 expiry 后由现有过期流程收敛。

## PM/KPI Redis 物理隔离

### 配置契约

WorkerConfig 新增：

```yaml
pm_redis:
  addrs:
    - redis-pm:6379
  password: ${PM_REDIS_PASSWORD}
  db: 0
  pool_size: 200
```

配置规则：

- `pm_redis` 完全缺失时回退到 `redis`。
- 只要填写了 `pm_redis`，就必须通过独立连接和启动期健康检查。
- 生产环境拒绝 `pm_redis.addrs` 与 `redis.addrs` 完全相同，除非显式启用兼容回退模式。
- 日志、指标和 health component 使用 `redis-core`、`redis-pm` 固定低基数标签。

### 依赖路由

迁移到 `PMRedis`：

- `pmstream.NewRedisWindowStore`
- PM aggregation recovery/finalizer/sweeper 使用的窗口 store
- KPI Router L2 cache

保留在 `Redis`：

- ACS session/admission
- cmdq/taskq 和 terminal TTL
- alarm Redis store
- device/product/common cache
- PM 在线配置 admission gate

产品元数据缓存默认留在 core Redis，因为它同时被非 KPI 业务消费；KPI Router 的计算结果 L2 缓存独立迁移。

### Redis 实例资源

`redis-core`：

- CPU limit：2 cores
- container memory：4GiB
- maxmemory：3GiB
- AOF everysec、noeviction、独立 volume

`redis-pm`：

- CPU limit：2 cores
- container memory：8GiB
- maxmemory：6GiB
- AOF everysec、noeviction、`no-appendfsync-on-rewrite yes`、关闭 RDB save
- 独立 volume，仅加入内部 compose network，不暴露宿主机端口

两个实例都必须满足 `container memory >= maxmemory + 1GiB`，为 AOF rewrite 的 fork/COW 留出空间。

### 无损切换

切换采用短维护窗口，不依赖清库：

1. 部署支持 `pm_redis` 但仍指向 core Redis 的兼容代码。
2. 启动 `redis-pm`，验证 AOF、容量、健康检查和监控。
3. 在上一小时结果完成发布后停止 Worker；NATS 和 TSDB outbox 继续持久保存新 PM 事件。
4. 以 SCAN + pipeline DUMP/RESTORE 复制 `pmagg:*` 活动窗口键并保留 TTL；复制期间 Worker 不写聚合窗口。
5. 对比源/目标键数、抽样哈希内容和 TTL，执行窗口状态一致性检查。
6. 修改 Worker `pm_redis` 指向 `redis-pm` 并启动。
7. 验证恢复活动窗口、NATS backlog 排空和前一/当前小时覆盖率。
8. core Redis 中旧 `pmagg:*` 键保留一个完整窗口 TTL 安全期后，再由有界清理任务删除。

复制失败时直接重新指向 core Redis 启动 Worker；源数据在安全期内不删除，因此回滚不丢状态。

## 小时发布链路优化

### 消除无意义删除

`deleteStaleRollupRevisionTx` 在 revision 1 没有旧 revision 可清理。revision 1 直接跳过两个 DELETE。

revision 大于 1 时保留定点清理，并为以下两表增加覆盖其完整谓词的索引：

```text
(publication_task_version_id,
 entity_key,
 granularity,
 window_start,
 revision,
 event_id)
```

- `pm_aggregation_counter_rollups`
- `pm_aggregation_rollup_outbox`

索引必须通过迁移创建，并用查询形态测试防止后续回退到不匹配索引。

### 并发调优顺序

先完成 DELETE/索引优化，在 20,000 设备环境测量 finalize 吞吐和 TSDB 等待，再决定是否将 finalize concurrency 从 32 提升到 64。禁止在 SQL 未优化前先增并发。

提升并发必须同步校验：

- Worker GOMAXPROCS 和 CPU limit
- TSDB max_conns
- TSDB CPU、锁等待、WAL 和磁盘 await
- Redis-pm pool size 和命令延迟

若 64 并发不能在 +15 分钟内完成，则下一步采用 set-based 批量发布状态更新；本阶段不通过无上限并发压迫 TSDB。

### 日/周重算

正常进行中日/周结果继续消费小时 rollup 增量推进。全周期快照只用于版本变更、迟到修正和显式重建。

重建查询按 task version 和时间范围 keyset 分页，禁止在单个事务中把整个周期 payload 全量物化到临时表。重建任务保留现有幂等 generation 和版本审计语义。

## 监控与告警

新增参数同步指标：

- `omc_paramsync_runs_ready_but_not_finalized`
- `omc_paramsync_run_counter_drift`
- `omc_paramsync_active_run_oldest_age_seconds`
- `omc_paramsync_reconcile_finalized_total{result}`
- `omc_paramsync_reconcile_duration_seconds`

新增/拆分 Redis 指标标签：

- pool in-use/max/idle
- operation latency
- ops/s
- used/max memory
- rejected connections、evictions
- AOF rewrite 状态和最近 rewrite 耗时
- keyspace scan duration

每个指标必须带固定 `redis_role="core|pm"`，禁止地址、key 或 device 作为标签。

新增聚合 SLO：

- close claim lag：`claim_at - (window_end + grace)`
- device publication completion lag
- hierarchy/network publication completion lag
- revision cleanup duration和影响行数
- rebuild snapshot page duration和行数

## 错误处理与降级

- redis-core 失败：ACS/任务队列按现有保护策略降级；不得自动转移到 redis-pm。
- redis-pm 失败：PM 聚合 readiness 置 0，停止消费聚合 subject，PM 文件和 outbox 继续持久化；不得写入 core Redis。
- TSDB 发布失败：保留 finalize claim/retry 语义，不能删除 Redis 窗口。
- 参数同步收敛失败：保留 run 原状态并记录具体阶段；单个坏 run 不能阻塞同批其他 run。
- 所有 terminal outbox 写入使用唯一 dedupe key，重试不得重复发送业务终态。

## 测试策略

### 参数同步

- run 已完全处理但处于 executing 时，维护任务推进到 succeeded。
- 存在失败结果时推进到 failed，并只写一次 terminal outbox。
- 事件处理和维护任务并发时只发生一次终态转换。
- 计划未创建任务、任务终态结果缺失、真实活跃任务三类缺口分别测试。
- 活跃扫描只访问非终态 run，并按最老优先有界处理。

### Redis 隔离

- `pm_redis` 缺失时兼容回退。
- 独立配置时 PM store/KPI cache 使用 PM client，task/alarm/cache 使用 core client。
- 两个实例健康状态和指标标签独立。
- redis-pm 故障不会耗尽 ACS core pool。
- 键复制工具验证 TTL、类型、内容和幂等重跑。

### 聚合发布

- revision 1 不执行 stale rollup DELETE。
- revision 大于 1 只删除非当前 event IDs。
- 迁移包含匹配 DELETE 谓词的两个索引。
- 20,000 窗口压力测试记录 +12m claim 和 +15m 完整发布。
- 日/周进行中覆盖率在重建、迟到事件和版本切换后保持正确。

## 部署与验收

按以下顺序部署：

1. 参数同步收敛代码、Redis 双 client 支持、SQL/索引优化和监控。
2. 完整单元测试、集成测试、race、build、vet 和前端 typecheck。
3. 构建发布包并先以兼容模式部署，确认没有行为回归。
4. 启动 redis-pm，执行无损活动窗口迁移后切换 Worker。
5. 验证 20,000 设备至少一个完整小时周期，以及当前日/周进行中结果。

最终验收条件：

- 所有服务健康，无新增 503、拒绝、死信和数据库锁等待。
- `ready_but_not_finalized = 0`，不存在无真实任务却超过 5 分钟的活跃 run。
- 参数同步 outbox、NATS PARAM_SYNC/TASK 无持续积压。
- PM NATS 突发可在告警 firing 前排空。
- 小时 close claim 在 +12m ±30s；小时所有层级结果在 +15m 前完成。
- K900010006、K900010076 小时结果完整，当前日/周覆盖率连续更新。
- redis-core pool 持续低于 70%，redis-pm 无 rejected/evicted keys。
- 宿主机 CPU、内存、磁盘 await 和 iowait 均不出现持续饱和。

## 回滚

- 应用回滚时将 `pm_redis` 配置回退到 core Redis，并使用安全期内保留的 `pmagg:*` 源键。
- 新增索引可以保留，不影响旧代码；如确需删除，使用并发删除索引，不能阻塞聚合表。
- redis-pm volume 在确认回滚完成前不删除。
- 参数同步状态推进属于合法终态，不进行反向恢复；terminal outbox 的幂等键防止旧版本重复发送。

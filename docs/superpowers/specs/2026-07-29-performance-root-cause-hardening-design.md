# OMC 性能根因治理设计

## 背景

2026-07-29 对 `172.24.224.197` 的线上只读诊断确认，宿主机整体 CPU、内存和即时磁盘
I/O 均未饱和，但 PM 数据链路存在三个相互独立的局部故障：

1. ACS 曾因 PM 队列达到高水位进入背压；队列归零后，未曾触发的磁盘信号仍因处于
   `disk_low_pct` 与 `disk_high_pct` 之间而阻止解除，导致 PM 上传持续返回 503。
2. Redis 的 PM 聚合窗口状态达到 `maxmemory=2GiB`，`noeviction` 正确地拒绝了新写入，
   但 Worker 恢复窗口持续重试并产生大量 OOM。现场约 53 万 Redis 键中，PM 聚合的
   daily、weekly、monthly 键各约 7 万；聚合 Hash 平均约 99 个字段，单个窗口的 98 个
   指标会分别保存 definition、sum、count、min、max，字段开销是主要容量来源。
3. TimescaleDB 影子维表同步把完整快照复制到临时 staging 表后执行 anti-join DELETE。
   staging 表没有索引，复合主键比较还使用 `IS NOT DISTINCT FROM`，现场单表删除需要
   20–30 秒并且每分钟重复。

另有一个监控契约问题：app、worker 和 ACS 都注册
`omc_pm_queue_sample_timestamp_seconds`，只有 ACS 真正采样，导致 app/worker 的零值触发
`OMCPMQueueSampleStale` 误报。

## 目标

- PM 队列压力恢复后，ACS 能在队列低水位条件满足时解除队列背压；磁盘和 I/O 仍保持各自
  独立的高低水位保护。
- 在不改变小时、天、周、月聚合语义和 `noeviction` 数据完整性策略的前提下，显著降低
  Redis 聚合窗口的字段数和内存占用。
- 影子维表同步的 stale-row 删除能够使用 staging 主键索引，消除每分钟 20–30 秒长查询。
- 队列采样过期告警只覆盖实际采样进程，并继续保留 Dashboard 查询超时、并发限制、
  慢查询、资源和 Redis 内存告警。
- 用清空后的 OMC 环境部署最新代码，完成从空库迁移、数据接入、队列、聚合和监控验证。

## 非目标

- 不取消 PM 上传背压，也不放宽磁盘硬保护。
- 不把聚合窗口状态迁移到 PostgreSQL。
- 不把 Redis 改为会静默淘汰聚合状态的 LRU 策略。
- 不重构 Dashboard 查询链路；现有五分钟定时刷新、查询超时和并发限制保持不变。
- 不删除服务器操作系统文件或非 `omcgo` compose 项目的数据。

## 服务器清理边界

目标 compose 项目为 `omcgo`，部署目录为 `/opt/omc/current/deploy`。清理动作包括：

- 停止并删除 `omcgo` 的应用、基础设施和监控容器；
- 删除 compose 网络；
- 删除以下 `omcgo` 命名卷：
  - `omcgo_pgdata`
  - `omcgo_tsdbdata`
  - `omcgo_redisdata`
  - `omcgo_natsdata`
  - `omcgo_miniodata`
  - `omcgo_prometheusdata`
  - `omcgo_lokidata`
  - `omcgo_tempodata`
  - `omcgo_grafanadata`
- 删除经容器挂载确认属于 OMC 的运行日志目录。

保留 `/opt/omc/current` 发布程序、配置文件和非 OMC Docker 项目。清理完成后验证：

- 不再存在带 `com.docker.compose.project=omcgo` 标签的容器；
- 上述命名卷均不存在；
- 后续部署从空 PostgreSQL、TimescaleDB、Redis、NATS、MinIO 和监控存储启动。

## 设计

### 1. 背压信号独立迟滞

用位状态表示 `disk`、`io`、`queue` 三个独立压力源，不再用单一布尔值表达“曾经有任意
压力”。

每次 watchdog 采样分别更新三个 latch：

- 磁盘：
  - 未激活且 `disk_pct >= disk_high_pct` 时激活；
  - 已激活且 `disk_pct <= disk_low_pct` 时解除；
  - 未触发磁盘高水位时，中间区间不会创建新的磁盘 latch。
- I/O：
  - 使用 `io_some_high_pct` 和 `io_some_low_pct`，行为与磁盘一致。
- 队列：
  - pending 或 oldest 达到高水位时激活；
  - pending、oldest 都回到低水位，且已观测到非正 backlog slope 后解除；
  - 采样失败时不创建新 latch，但保留已经存在的 queue latch。

最终背压状态是三个 latch 的 OR。状态变化指标继续使用 `engaged/released`，进入时记录实际
新触发原因，全部解除时记录 `recovered`。

兼容约束：

- watchdog 启动前没有确认过压力时，探测失败继续 fail-open；
- 已确认的对应压力信号在采样失败时不得被错误释放；
- CPU load 继续只监控，不参与背压决策。

### 2. Redis 聚合窗口紧凑格式

当前每个 accumulator definition ID 使用五个 Hash 字段：

```text
defs[definition_id] = encoded_definition
acc[definition_id|sum]
acc[definition_id|count]
acc[definition_id|min]
acc[definition_id|max]
```

新格式将 definition 和数值状态合并到 `acc` 的一个字段：

```text
acc[definition_id] = v1|encoded_definition|sum|count|min|max
```

`encoded_definition` 使用 Raw URL Base64，不包含 `|`，可安全作为分隔字段。Lua 脚本在一次
原子执行中完成读取、数值合并和写回。相同的 98 个指标从约 490 个 Hash 字段降为 98 个，
并且新窗口不再创建独立 `defs` Hash。

升级兼容：

- 读取路径同时识别新格式字段和旧 `|sum`、`|count`、`|min`、`|max` 字段。
- 新版本首次累加旧窗口的某个指标时，读取旧 definition 和四个数值字段，合并本次增量，
  写成新格式并删除该指标的旧字段。
- Window 删除逻辑在兼容期内仍同时删除 `acc` 和 `defs` 键。
- 新旧格式产生相同的 `WindowState` 和最终聚合结果。

容量策略：

- Redis 保持 AOF everysec、`maxmemory-policy=noeviction`。
- 开发和发布资源规划脚本不得再生成 `allkeys-lru` 或 `volatile-lru`。
- 删除“Redis 仅使用数 MB、maxmemory 封顶 512MB”的过期假设。
- `maxmemory` 与容器内存保持 AOF rewrite/COW 余量，并由同一资源预算派生。
- 现有 Redis 85% 内存告警保留，新增持续写入失败/聚合失败告警，确保在窗口失败前可见。

### 3. TimescaleDB staging 索引

影子维表同步保持“完整源快照 + 目标增量 merge”的语义，但在 CopyFrom 完成后：

1. 为 staging 表按目标表主键列创建唯一索引；
2. 对 staging 表执行 `ANALYZE`；
3. anti-join 对非空主键使用普通等值比较；
4. 再执行 UPSERT 和 stale-row DELETE。

目标表主键天然非空，因此 `=` 与当前 `IS NOT DISTINCT FROM` 语义等价，同时更容易使用
B-tree 索引。没有可用主键的旧影子表仍保留原有 TRUNCATE + COPY 回退行为。

同步日志增加每张表的行数和耗时，使后续可直接定位退化表。

### 4. 告警与线上保护

- `OMCPMQueueSampleStale` 增加 `deployment_unit="acs"` 限定。
- Redis 内存告警继续以 `used_memory/maxmemory` 为准，而不是容器内存上限。
- 增加 PM 聚合事件失败率、窗口 finalize/recovery 错误的告警。
- Dashboard 保护保持：
  - 查询超时；
  - 并发限制；
  - 查询拒绝与超时指标；
  - TSDB 慢查询；
  - 主机、容器、磁盘、Redis 资源告警。

## 错误处理

- 紧凑 Redis 值格式不合法时返回带 definition ID 上下文的错误，不以零值继续聚合。
- 旧格式迁移在 Lua 中原子完成，避免并发消费产生双计数。
- staging 索引创建或 ANALYZE 失败时回滚该表事务，其他影子表继续同步。
- 背压状态更新保持无外部 I/O 的纯决策函数，watchdog 只负责采样和原子发布状态。

## 测试

### 单元测试

- 磁盘、I/O、队列 latch 分别进入、保持和解除。
- 队列触发后在磁盘中间区间能够解除。
- 磁盘已经触发时即使队列恢复仍保持磁盘背压。
- 队列采样失败的启动 fail-open 和已激活 fail-closed。
- 新 Redis 格式幂等累加、完整性统计、rollup chunk 和 finalize 结果。
- 新窗口每个指标只占一个 accumulator Hash 字段且不创建 definition Hash。
- 旧格式窗口读取和首次累加迁移。
- staging 唯一索引、ANALYZE 和等值 anti-join SQL。
- 告警规则限定 ACS，并包含 Redis/聚合失败保护。

### 仓库级验证

- `cd omcgo && go build ./...`
- `cd omcgo && go test ./...`
- `cd omcmb && npm run typecheck`
- 监控规则语法检查。
- compose/release shell 测试。

### 空环境部署验证

- 所有数据库迁移从零成功。
- compose 服务全部 healthy，健康检查 25/25。
- Redis 为 `noeviction`，AOF 开启，`maxmemory < container limit`。
- 构造 PM 输入后 NATS consumer 无持续 pending、ack pending 或 redelivery。
- 小时、天、周聚合窗口能够建立；通过可控测试时间或直接集成测试验证各粒度 finalize，
  不等待自然周结束。
- Redis 新聚合 key 使用紧凑格式，内存使用随窗口数量线性增长且无 OOM。
- ACS 队列背压触发和恢复测试通过，不再被未触发的磁盘 latch 阻挡。
- 影子维表同步查询不再持续触发 20–30 秒长查询。
- Dashboard 每五分钟定时刷新，不发生事件即时刷新；查询超时和并发保护指标正常。

## MR 验收标准

- 所有新增和现有相关测试通过。
- Go 全量 build/test 与前端 typecheck 通过。
- 清空环境部署成功且全部服务健康。
- PM 上传、队列消费、Redis 聚合、TSDB 维表同步和 Dashboard 查询均完成线上验证。
- 线上没有 Redis OOM、持续 PM 上传背压、队列积压或新增慢查询告警。
- 提交使用 Conventional Commits，MR 包含问题证据、设计、测试结果、部署版本和回滚说明。

# PM 10000 基站持续负载加固设计

## 状态

已批准，待实施。

## 背景

PM 稀疏存储版本部署到 `172.24.224.197` 后，10000 基站压测证明了稀疏模型的空间方向正确：

- 平均每个 PM 文件约 5092 个逻辑指标；
- 稀疏逻辑行与物理值行比例约为 3.02:1；
- 只物理保存约 33% 的真实非空值；
- 小时版本发布能隔离失败版本，没有暴露半成品结果。

压测同时暴露了持续吞吐和自动恢复问题：

- 仅约 2043 个指标字典行发生了超过 211 万次更新；
- 每个 PM 文件在热路径对约 1400 个稳定字典项执行 `ON CONFLICT DO UPDATE`；
- `pm_metric_sets` 只有 26 个稳定集合，却发生约 8 万次更新；
- TimescaleDB 长期有 20–30 个入库连接等待同一批字典行；
- PM 入库按“字典 → 小时桶”取锁，小时汇总按“小时桶 → 字典”取锁，形成反向锁序；
- 小时汇总连续 3 次死锁，耗尽任务重试，没有活动版本，聚合水位为 0；
- NATS PM consumer 的 128 个 ack-pending 槽位持续占满，最终观测到 9571 条 pending；
- 当前反压只把 backlog 折算成磁盘未来占用，在大容量空闲磁盘上不会因消费能力不足及时限流；
- OTEL collector 未启动、备份 bucket 配置无效、未知指标指标名误导，影响可观测性。

## 目标

1. 消除 PM 热路径对稳定字典和稳定指标集的重复写入。
2. 消除 PM 入库与小时汇总之间的反向锁序。
3. 让瞬时死锁或串行化冲突只重试当前数据库批次，不消耗整个小时任务的全部重试。
4. 让失败小时桶在满足可重试条件时自动恢复并推进水位。
5. 让上传反压直接感知队列深度、最老消息年龄和生产消费速率差。
6. 补齐 monitoring、OTEL、备份检查和关键告警，消除已知运行噪声。
7. 使用同一批 PM 原始文件完成旧实现与稀疏实现双跑，验证逻辑等价和物理空间目标。
8. 通过 10000 文件突发与 11.1 文件/秒持续负载验收。

## 非目标

- 不通过增加 CPU、内存或 TimescaleDB 连接数掩盖锁竞争。
- 不引入独立字典微服务。
- 不改变 PM 逻辑查询语义、保留周期或设备重传协议。
- 不删除缺少真实查询计划证据的索引。
- 不在压测期间通过手工删除队列、跳过小时任务或直接修改数据库状态制造通过结果。

## 方案选择

### 采用：根因修复与闭环反压

消除稳定数据热写、统一锁序、增加批次级瞬时错误重试、自动恢复失败桶，并把队列状态纳入上传准入。

该方案直接解决数据库吞吐和队列增长的根因，保持现有模块边界，不需要拆分服务。

### 不采用：仅降低 worker 并发

降低 128 个并发消费槽位可以减少锁竞争，但单文件仍会更新大量稳定行，稳态吞吐仍不能达到 11.1 文件/秒。这只能作为线上临时止血，不能作为验收结果。

### 不采用：独立字典服务

独立服务可以集中缓存与登记字典，但引入额外一致性、部署和故障域，超过本轮问题所需范围。

## P0：PM 入库热路径

### 指标字典

将字典解析拆成三个步骤：

1. 使用输入路径数组批量查询已经存在的 `metric_path → metric_id`。
2. 只对查询结果中缺失的路径执行批量 `INSERT ... ON CONFLICT DO NOTHING`。
3. 再查询缺失路径并合并 ID 结果。

PM 热路径不得更新已存在字典行的 `metric_type`、`statis_type`、`unit` 或 `updated_at`。这些元数据由指标库同步和专门的字典补全链路维护。

同一进程可使用有界缓存减少稳定路径的重复查询，但数据库的“只插缺失项”语义必须独立成立，不能依赖缓存保证正确性。

并发发现同一未知指标时，所有事务都允许执行 `DO NOTHING`，提交后统一查询最终 ID，不产生重复记录。

### 指标集

`pm_metric_sets` 的稳定键为：

```text
(product_key, counter_group, content_hash)
```

入库先按稳定键查询；缺失时执行 `INSERT ... ON CONFLICT DO NOTHING`，随后查询 `metric_set_id`。命中时不再执行 `DO UPDATE metric_ids`。

`content_hash` 必须继续由排序后的完整 `metric_ids` 计算。若同一 hash 查询到不同数组，返回一致性错误并告警，不覆盖已有集合。

### 锁顺序

所有同时涉及小时桶和字典的事务统一使用：

```text
小时桶/版本 → 字典 → 指标集 → anchor/value
```

原始 PM 入库在写字典前执行 `markHourlyBucketsDirty`。该更新仍位于同一个入库事务中，后续失败会整体回滚，不破坏原子性。

小时公式 KPI 字典登记必须在持有小时桶版本写锁的批次事务之前完成。批次事务只读取已经解析的 `metric_id` 并写小时 anchor/value 与批次计数。

### 瞬时错误重试

为小时批次增加数据库级重试器，只处理：

- `40P01` deadlock detected；
- `40001` serialization failure。

每个批次最多重试 3 次，使用带抖动的短退避。每次重试创建全新事务，旧事务必须完成回滚。

语法错误、约束错误、数据一致性错误不重试，直接交给任务级失败处理。

## P0：失败小时桶自动恢复

小时维护循环扫描以下情况：

- 主库中 `pm_aggregate_hourly` 任务为 `failed`；
- 错误属于可重试数据库错误；
- 对应时间桶存在原始 anchor；
- 当前不存在 `active` 版本；
- 距离上次失败超过退避时间；
- 当前没有同桶 pending/running 任务。

满足条件时把同一任务安全地恢复为 pending，重置本轮执行锁和时间字段，保留累计恢复次数和最后错误用于审计。

自动恢复设置独立上限。达到上限后保持失败并告警，避免永久 poison bucket 无限循环。

维护循环还要清理孤立 `building` 版本，并确保失败版本不会计入 dirty-bucket 正常值。新增失败版本和无水位告警，避免 `dirty=0` 掩盖汇总不可用。

## P1：队列闭环反压

### 队列状态

NATS event bus 增加 PM durable consumer 状态读取接口，至少返回：

```text
pending
ack_pending
redelivered
oldest_pending_age
stream_last_sequence
consumer_ack_sequence
sampled_at
```

watchdog 保存连续样本，计算：

- publish rate；
- ack rate；
- pending delta；
- backlog slope。

### 迟滞准入

默认阈值：

```text
pending_high = 2000
pending_low = 500
oldest_age_high = 10m
oldest_age_low = 2m
```

进入反压满足任一条件：

- 磁盘或 IO PSI 达到既有高水位；
- pending 达到高水位；
- oldest pending age 达到高水位。

解除反压要求所有可用信号都回到低水位。信号读取失败时保留当前反压状态，不允许在无法确认队列健康时自动解除。

backlog slope 用于指标和告警，不单独触发 503，避免短突发产生抖动。pending 和 oldest age 通过迟滞控制准入。

被拒绝的 PM 上传继续返回 503，不写去重键，设备重试后不会丢文件。

### 指标与告警

新增：

- `acs_pm_queue_pending`
- `acs_pm_queue_ack_pending`
- `acs_pm_queue_oldest_age_seconds`
- `acs_pm_queue_publish_rate`
- `acs_pm_queue_ack_rate`
- `acs_pm_queue_backlog_slope`
- `omc_pm_dictionary_insert_total`
- `omc_pm_dictionary_existing_total`
- `omc_pm_dictionary_conflict_total`
- `omc_pm_db_retry_total`
- `omc_pm_hourly_failed_versions`
- `omc_pm_hourly_watermark_lag_seconds`

告警覆盖：

- pending 或 oldest age 超阈值；
- backlog slope 持续为正；
- 字典 existing/insert 比例异常；
- TimescaleDB deadlock 增长；
- 小时任务失败、无 active 版本或水位不推进；
- 反压持续时间过长。

## P1：运行质量

### 未知指标指标名

新增 `omc_pm_discovered_counters_total` 表示“指标库外发现但仍被保留”的计数。

现有 `omc_pm_dropped_counters_total{reason="whitelist_miss"}` 保留一个兼容周期并在 HELP 中标记 deprecated。所有项目内面板和告警切换到新指标。

### OTEL

发布部署默认启动 collector。业务服务只有在 OTEL endpoint 配置存在时才启用 exporter；明确禁用监控时不反复连接不存在的 `otelcol`。

collector、Prometheus 和业务 metrics target 都必须有健康检查。

### 备份 bucket

启动时校验备份 bucket 名称。空值表示禁用 bucket 容量巡检；非法名称返回明确配置错误，不允许后台每分钟重复打印相同警告。

### Dashboard

在 PM 写入和小时汇总并发时验证 Dashboard 查询超时。允许客户端主动取消产生 `context canceled`，但服务端必须区分客户端取消、数据库超时和真实查询错误，避免误告警。

## P2：索引与双跑

### 索引

对以下真实查询执行 `EXPLAIN (ANALYZE, BUFFERS)`：

- 设备时间范围；
- 指标时间范围；
- Dashboard 最新 KPI；
- CSV 导出；
- 小时查询。

记录每个候选索引的扫描次数、实际执行时间和 buffer 命中。只有在覆盖全部核心查询后确认无收益，才允许删除索引。

特别检查约 145 MiB 的 `idx_pm_metric_values_metric_time` 和 TimescaleDB 自动时间索引。不得只依据 `idx_scan=0` 删除。

### 同批双跑

使用固定 PM 原始文件集，分别运行：

- `main` 基线提交的 legacy schema/实现；
- 当前分支的 sparse schema/实现。

两个环境必须隔离数据库、MinIO、NATS、端口和数据目录。双跑工具比较：

- Counter、KPI、Dashboard、CSV 和小时汇总；
- 逻辑行数、null、non-null、filled；
- 分页顺序；
- 历史指标集变化；
- 未知指标；
- 重复投递；
- 迟到数据。

物理空间报告包含：

- 表、索引、TOAST、WAL；
- anchor/value 数；
- legacy 逻辑行数；
- sparse 逻辑与物理比；
- 新旧总物理空间比例。

稀疏实现的 PM 表、索引和 TOAST 总空间必须不超过 legacy 的 20%。

## 测试策略

所有行为修改使用 TDD：

1. 先写能稳定复现现有热写、锁序、失败恢复或反压缺口的测试。
2. 确认测试因目标缺失而失败。
3. 实现最小修复。
4. 运行相关包测试。
5. 最终运行全量构建、测试和部署验收。

必须包含：

- 字典已有路径不执行 UPDATE；
- 并发登记未知指标得到同一 ID；
- 指标集命中不产生 UPDATE；
- 入库先标 dirty 再访问字典；
- 小时公式字典登记不持有桶版本事务；
- 40P01/40001 批次重试，其他错误不重试；
- 可重试 failed bucket 自动恢复，永久错误不恢复；
- pending 与 oldest-age 迟滞进入/解除；
- 队列状态读取失败时不错误解除反压；
- 503 不写去重键；
- 新旧未知指标 metric 兼容；
- OTEL 显式禁用不产生连接错误；
- backup bucket 空值禁用、非法值启动失败；
- 双跑逻辑一致和空间比例门禁。

## 验收标准

### 本地

- `git diff --check`
- `cd omcgo && go build ./...`
- `cd omcgo && go test ./...`
- `cd omcmb && npm run typecheck`

### 线上

- 10000 PM 文件突发在 15 分钟内排空；
- 以 11.1 文件/秒持续 30 分钟，pending 线性斜率不大于 0；
- oldest pending age 小于 5 分钟；
- ack pending 不长期顶满；
- TimescaleDB 无新增 deadlock；
- 持续锁等待接近 0，不形成稳定锁队列；
- 小时任务成功，没有 failed/building 遗留版本；
- 小时 active 版本唯一且水位推进；
- 主机 CPU 保留至少 20% 空闲；
- 容器无 OOM，内存和磁盘低于既有阈值；
- app、ACS、worker、web、数据库、NATS、Redis、MinIO、Prometheus、OTEL 全部健康；
- 业务日志无重复 OTEL DNS 错误和 backup bucket 警告；
- 同批双跑逻辑一致；
- sparse PM 物理空间不超过 legacy 的 20%。

## 发布与 MR

实现继续使用 `codex/pm-sparse-storage-rollup` 分支，并更新现有 MR `!324`。

发布步骤：

1. 本地全量验证；
2. 构建完整版本包；
3. 备份当前部署配置和数据库统计基线；
4. 执行数据库迁移；
5. 滚动部署；
6. 自动恢复现有失败小时桶；
7. 运行突发与持续负载；
8. 生成新旧双跑、资源、队列和空间报告；
9. 所有验收项通过后提交最终代码并推送到 MR。

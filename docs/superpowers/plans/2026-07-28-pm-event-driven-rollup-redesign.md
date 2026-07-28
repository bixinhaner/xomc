# PM 事件驱动逐级汇聚重构实施计划

> **For Codex:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 删除“可运行聚合任务”和隐藏设备任务，以固定公共流水线实现设备与规则维度的小时、日、周、月 Counter/KPI 逐级汇聚，并把 10000 设备运行时汇聚状态控制在有界内存和 I/O 范围内。

**Architecture:** 15 分钟 PM 入库事务只发布设备 Counter 事件；公共设备流水线将其累计为设备小时，并以紧凑 Counter 状态逐级生成设备日、周、月。设备小时事件同时经过规则版本匹配，累计设备组、产品、频段、全网和自定义维度，再逐级生成规则日、周、月。Redis 只保存按预计状态大小自动分片的活动窗口，窗口关闭后批量写结果并删除活动状态；恢复只扫描有界元数据，不读取完整累计 Hash。

**Tech Stack:** Go、pgx、Squirrel、Redis Lua、NATS JetStream、PostgreSQL/TimescaleDB、Docker Compose。

---

## 统一实现约束

- 先完成全部代码，再执行完整构建、全量测试、部署和 10000 设备验收。
- 开发期间只运行用于 TDD 的最小目标测试，不逐模块部署或做业务压测。
- 每个生产改动先添加失败测试并确认失败原因，再写最小实现使其通过。
- KPI 只能从当前粒度的 `sum/count/min/max` Counter 状态计算，禁止聚合上一级 KPI。
- 规则更新创建不可变版本，默认从下一个自然小时生效；不回算历史。
- 不兼容旧任务运行状态、旧 Redis 窗口、旧 NATS 消息或旧测试数据。
- 物理表和 API 删除调度语义；公共消费者、关闭器、恢复器不是规则任务。
- Redis 禁止 `KEYS`、`HGETALL` 和无界扫描；所有读写必须按分片和页限制。
- 时序数据库保留由现有 PM 系统配置控制；本次只固定 NATS 聚合流保留。

## Task 1：建立规则与公共流水线领域模型

**Files:**

- Modify: `omcgo/internal/pm/stream/task_model.go`
- Modify: `omcgo/internal/pm/stream/task_repository.go`
- Modify: `omcgo/internal/pm/stream/snapshot.go`
- Modify: `omcgo/internal/pm/stream/snapshot_test.go`
- Modify: `omcgo/internal/pm/stream/task_repository_test.go`
- Create: `omcgo/internal/pm/stream/rule_model.go`

**Steps:**

1. 新增失败测试，覆盖：
   - 规则不包含 `mode/cron/status scheduled` 等调度字段；
   - 更新规则生成新版本而不改写旧版本；
   - 未显式指定生效时间时向上取整到下一个自然小时；
   - 版本快照能同时返回内置规则和自定义规则。
2. 运行：

   ```bash
   cd omcgo && go test ./internal/pm/stream -run 'Test(Rule|Snapshot)'
   ```

   确认测试因领域模型和小时生效规则未实现而失败。
3. 实现 `AggregationRule`、`RuleVersionSnapshot`、成员和指标选择器。
4. 将存储层的版本创建边界改为下一自然小时，已生效版本保持只读。
5. 保留必要的内部 ID，但删除运行模式、cron 和任务状态对运行路径的控制。
6. 重跑目标测试并通过。

## Task 2：实现设备基础小时汇聚，不再生成隐藏设备任务

**Files:**

- Create: `omcgo/internal/pm/stream/device_pipeline.go`
- Create: `omcgo/internal/pm/stream/device_pipeline_test.go`
- Modify: `omcgo/internal/pm/stream/event.go`
- Modify: `omcgo/internal/pm/stream/matcher.go`
- Modify: `omcgo/internal/pm/stream/consumer.go`
- Modify: `omcgo/cmd/worker/pm_streaming.go`
- Delete: `omcgo/internal/pm/adhoc/builtin_reconciler.go`
- Delete: `omcgo/internal/pm/adhoc/builtin_reconciler_test.go`

**Steps:**

1. 新增失败测试，覆盖：
   - 同一设备同一小时的 4 个 15 分钟槽只累计一次；
   - 收齐 4 槽立即关闭，未收齐按宽限时间超时关闭；
   - 设备基础汇聚无需任何规则或隐藏任务记录；
   - LTE/NR/GSM 均走同一公共设备流水线；
   - 重复文件、重复槽不重复累计。
2. 运行设备流水线目标测试并确认红灯。
3. 实现固定设备流水线入口：
   - 从标准化 15 分钟 Counter 事件构造设备小时贡献；
   - 以 `device_id + technology + object + hour` 作为业务窗口；
   - 由固定 Counter/KPI 目录解析所需原始 Counter；
   - 关闭时产生设备小时 Counter 状态、KPI 结果和上卷事件。
4. 从 worker 启动路径删除启动/每 5 分钟内置任务协调器。
5. 删除隐藏 `内置-设备-LTE/NR/GSM` 的生成、对齐和运行代码。
6. 重跑目标测试并通过。

## Task 3：实现紧凑、有界、自动分片的活动状态

**Files:**

- Modify: `omcgo/internal/pm/stream/redis_store.go`
- Modify: `omcgo/internal/pm/stream/redis_store_test.go`
- Modify: `omcgo/internal/pm/stream/accumulate.lua`
- Create: `omcgo/internal/pm/stream/sharding.go`
- Create: `omcgo/internal/pm/stream/sharding_test.go`
- Modify: `omcgo/internal/pm/stream/window_repository.go`
- Modify: `omcgo/internal/pm/stream/window_repository_test.go`

**Steps:**

1. 新增失败测试，覆盖：
   - 根据设备数、Counter 状态数和 16 MiB 目标自动计算 1..256 分片；
   - 同一实体稳定落入同一分片；
   - Redis 字段不重复保存完整 JSON 定义；
   - 状态以紧凑 Counter ID 保存 `sum/count/min/max`；
   - 分片读取使用 `HSCAN COUNT` 分页，代码中不存在 `HGETALL`/`KEYS`；
   - 单页估算不超过 32 MiB。
2. 确认目标测试失败。
3. 为窗口键增加 `pipeline/rule_version + granularity + window + shard`。
4. 将重复定义移动为每版本一次的目录映射，累计 Hash 只保存紧凑 ID 和数值状态。
5. 用游标分页 API 取代窗口全量读取；最终计算器逐页消费并及时释放内存。
6. 窗口元数据保存分片数、预计成员数、已收子项、水位和关闭状态。
7. 重跑目标测试并通过。

## Task 4：实现设备小时到日、周、月的逐级 Counter 上卷

**Files:**

- Modify: `omcgo/internal/pm/stream/rollup_event.go`
- Modify: `omcgo/internal/pm/stream/direct_rollup.go`
- Modify: `omcgo/internal/pm/stream/direct_rollup_test.go`
- Modify: `omcgo/internal/pm/stream/finalizer.go`
- Modify: `omcgo/internal/pm/stream/finalizer_test.go`
- Modify: `omcgo/internal/pm/stream/finalize_values.go`
- Modify: `omcgo/internal/pm/stream/finalize_values_test.go`

**Steps:**

1. 新增失败测试，覆盖：
   - 小时 Counter `sum/count/min/max` 正确组合到日；
   - 日同时驱动周和自然月；
   - 周跨年、月天数和时区边界正确；
   - avg KPI 使用组合后的 `sum/count`，不平均小时 KPI；
   - 每个粒度关闭后都重新计算 KPI；
   - 数据完整性与单个公式依赖完整性分别记录。
2. 确认目标测试失败。
3. 定义紧凑上卷事件，只携带版本、实体、窗口、分片、Counter 状态和完整性摘要。
4. 实现：
   - 设备小时 → 设备日；
   - 设备日 → 设备周；
   - 设备日 → 设备月。
5. 最终计算改为分片流式读取、批量写入；完整或超时共用同一计算路径。
6. 将全局 `formulaIncomplete` 改为逐 KPI 的公式状态。
7. 重跑目标测试并通过。

## Task 5：实现规则维度匹配和逐级上卷

**Files:**

- Create: `omcgo/internal/pm/stream/rule_matcher.go`
- Create: `omcgo/internal/pm/stream/rule_matcher_test.go`
- Modify: `omcgo/internal/pm/stream/matcher.go`
- Modify: `omcgo/internal/pm/stream/consumer.go`
- Modify: `omcgo/internal/pm/stream/direct_rollup.go`
- Modify: `omcgo/internal/pm/stream/direct_rollup_test.go`
- Modify: `omcgo/internal/pm/stream/snapshot.go`

**Steps:**

1. 新增失败测试，覆盖：
   - 一个设备可同时命中多个设备组；
   - 无显式分组设备进入稳定的默认未分组组；
   - 产品、频段、全网和自定义规则匹配正确；
   - 新规则版本只接收生效小时及之后的设备小时事件；
   - 规则小时状态只有目标维度 Counter，不保存成员设备展开状态；
   - 规则小时 → 日，规则日 → 周/月；
   - 不消费设备 KPI，不查询 15 分钟数据库。
2. 确认目标测试失败。
3. 实现规则版本快照和小时事件匹配器。
4. 以设备小时 Counter 事件为唯一规则小时输入，按所有命中维度扇出紧凑贡献。
5. 规则小时按成员水位完整关闭，或按宽限时间超时关闭。
6. 实现规则维度的小时、日、周、月 Counter/KPI 上卷。
7. 重跑目标测试并通过。

## Task 6：修复恢复、关闭和幂等边界

**Files:**

- Modify: `omcgo/internal/pm/stream/recovery.go`
- Modify: `omcgo/internal/pm/stream/recovery_test.go`
- Modify: `omcgo/internal/pm/stream/window_repository.go`
- Modify: `omcgo/internal/pm/stream/finalizer.go`
- Modify: `omcgo/internal/pm/stream/outbox_repository.go`
- Modify: `omcgo/internal/pm/stream/consumer_test.go`

**Steps:**

1. 新增失败测试，覆盖：
   - 恢复只读活动窗口元数据/存在性，不读累计数据；
   - 一个窗口恢复失败不会中止其余窗口；
   - 关闭抢占、结果写入、上卷 outbox 和窗口完成具有幂等性；
   - 重启和 NATS 重投不会产生重复结果或重复上卷；
   - 超时窗口保留实际收到/预期子项和缺失摘要。
2. 确认目标测试失败。
3. 将恢复器改为分页读取活动窗口元数据并逐窗口隔离错误。
4. 将关闭器改为基于租约/状态 CAS 抢占；失败可安全重试。
5. 结果和上卷 outbox 保持同一数据库事务，成功后再删除 Redis 活动状态。
6. 重跑目标测试并通过。

## Task 7：重整基线 SQL、API 投影和运行配置

**Files:**

- Modify: `omcgo/migrations/000001_init_schema.sql`
- Modify: `omcgo/migrations/seed/000001_init_seed.sql`
- Modify: `omcgo/migrations/tsdb/000001_tsdb_schema.sql`
- Modify: `omcgo/internal/pm/aggregator/query.go`
- Modify: `omcgo/internal/pm/aggregator/query_test.go`
- Modify: `omcgo/internal/core/components/nats/nats.go`
- Modify: `omcgo/internal/core/components/nats/nats_test.go`
- Modify: `deployments/docker/docker-compose.yml`
- Modify: `deployments/docker/release/app/docker-compose.yml`

**Steps:**

1. 新增/更新失败测试，覆盖：
   - 新库不播种 12 个可运行内置任务；
   - 规则表不含调度状态语义；
   - 设备与规则结果视图能按小时/日/周/月查询；
   - 15m NATS 为 2h、小时为 48h、日为 40d 且有硬容量与 S2 压缩；
   - 周/月没有持久上卷流；
   - TSDB 保留仍读取 PM 系统配置。
2. 确认目标测试失败。
3. 重写 `000001` 基线中的聚合定义为规则/版本数据，不保留历史迁移兼容 SQL。
4. 删除 12 个内置运行任务种子，写入必要的内置规则定义。
5. 调整窗口、结果、索引和查询视图以支持公共设备流水线及规则版本。
6. 删除部署配置中“是否运行聚合任务”的开关语义，公共 PM 上卷服务随 worker 固定启动。
7. 重跑目标测试并通过。

## Task 8：补齐指标、端到端和负载验收工具

**Files:**

- Modify: `omcgo/cmd/pm-stream-e2etest/main.go`
- Modify: `omcgo/cmd/pm-stream-loadtest/main.go`
- Modify: `omcgo/internal/pm/streamtest/harness.go`
- Modify: `omcgo/internal/pm/stream/metrics.go`
- Modify: `omcgo/internal/pm/stream/metrics_test.go`
- Modify: `deployments/monitoring/grafana/dashboards/omc-dashboard.json`

**Steps:**

1. 新增失败测试，要求暴露：
   - 各粒度输入、关闭、超时、重试、落库计数；
   - 活动窗口/分片数、预计/实际字节；
   - 恢复单窗口耗时和错误数；
   - 最终计算页数、批次数、耗时；
   - 事件端到端延迟和 NATS pending/redelivery。
2. 扩展端到端工具，验证设备和规则维度 15m→小时→日→周/月的 Counter/KPI 正确性。
3. 扩展负载工具，输出 10000 设备吞吐、积压、Redis 峰值、数据库写入量和窗口关闭延迟。
4. 重跑目标测试并通过。

## Task 9：统一代码验证和提交

**Steps:**

1. 格式化全部修改的 Go 文件：

   ```bash
   cd omcgo && gofmt -w <modified-go-files>
   ```

2. 检查无界 Redis 命令和旧隐藏任务语义：

   ```bash
   rg -n 'HGETALL|KEYS|内置-设备|builtinDevice|cron_expr|oneshot|continuous' omcgo/internal/pm omcgo/cmd/worker
   ```

   只允许测试中的否定断言或与 PM 聚合无关的明确用法。
3. 执行后端完整验证：

   ```bash
   cd omcgo && go build ./... && go test ./...
   ```

4. 校验工作区：

   ```bash
   git diff --check
   git status --short
   ```

5. 提交：

   ```bash
   git add <implementation-files>
   git commit -m "feat(pm): 重构事件驱动逐级汇聚"
   ```

## Task 10：构建、清理旧聚合状态并部署

**Steps:**

1. 记录本地提交、服务器当前 release、容器状态和磁盘余量。
2. 使用仓库发布脚本构建当前提交的软件包；核对包内提交号和 compose 配置。
3. 停止 `/opt/omc/current` 的 compose 业务。
4. 只清理以下已确认可丢弃的测试数据：
   - 旧 `pmagg:*` Redis 活动状态；
   - 旧 PM 聚合 NATS streams/consumers；
   - 主库旧规则任务、版本、窗口；
   - 时序库旧聚合结果和 Counter rollup；
   - 不删除与本次无关的设备、配置和账号数据。
5. 按新 `000001` 基线重新初始化相关数据库结构和种子。
6. 安装新 release，启动 compose 栈。
7. 等待全部容器进入 healthy；如失败立即保留日志和现场，不反复重启掩盖问题。

## Task 11：统一业务与 10000 设备性能验收

**Correctness checks:**

- 不存在隐藏设备任务和 scheduled 聚合任务。
- 10000 台设备均注册，PM 文件接收、解析、15 分钟入库连续。
- 每台设备小时结果在 4 槽齐全或超时后生成。
- 抽样核对小时 KPI 来自 4 个 15 分钟原始 Counter。
- 抽样核对日来自小时 Counter、周/月来自日 Counter。
- 设备组、产品、频段、全网结果来自设备小时 Counter。
- 一个多组设备在所有命中组中各出现一次；未分组设备进入默认组。
- 公式缺依赖只影响对应 KPI。

**Performance checks:**

- 连续观察至少一个完整 15 分钟窗口和一个小时关闭窗口。
- 记录 worker、主库、时序库、Redis、NATS、MinIO 的 CPU、RSS、磁盘 IOPS/延迟和网络。
- Redis 无单个超大 Hash；单分片不超过 16 MiB，全部活动汇聚状态目标不超过 512 MiB。
- 恢复和最终计算不再每分钟产生 I/O timeout。
- NATS pending 最终回落，redelivery 无持续增长。
- 主库无持续锁等待、连接耗尽和异常高写放大。
- 时序库按 PM 配置保留；NATS 15m/小时/日保留分别为 2h/48h/40d。
- 磁盘 `%util`、await、队列深度与业务吞吐一起判断，不只依据 `%util=100%`。

**Failure handling:**

- 功能错误：停止压测，保留日志、窗口元数据和抽样 Counter，不提交 MR。
- 性能未达标：先定位 Redis 分片、NATS 消息体、主库批量写或 TSDB 压缩中的主因，再修复并重新执行统一验证。
- 全部通过后汇总提交号、release、业务结果、资源峰值、队列和数据库容量；等待用户明确要求后再 push/创建 MR。

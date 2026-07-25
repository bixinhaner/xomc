# PM 在线流式聚合 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan as one development batch followed by one consolidated verification gate. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 用基于 Outbox、NATS、Redis 窗口累加和统一结果表的在线聚合替换全部 PM 数据库扫描式聚合。

**Architecture:** 现有 PM 解析和 15 分钟原始入库保持主链路不变，在同一时序库事务写标准事件 outbox。Relay 将事件发布到 JetStream；规则匹配器加载不可变任务版本快照；Redis Lua 原子执行文件级去重和窗口累加；Finalizer 在完整或超时关闭时一次性写统一结果表。运行路径禁止查询原始 PM 表，新任务和新版本不补历史。

**Tech Stack:** Go、pgx/Squirrel、PostgreSQL、TimescaleDB、NATS JetStream、Redis 7、Lua、Docker Compose、Prometheus。

## Global Constraints

- 本计划采用“一次性完整开发、最后集中验证、验证通过后再提交 MR”。
- 开发阶段编写全部测试，但不在每个任务完成后反复执行构建、单测、部署或压测。
- 所有验证统一放在代码全部完成之后执行；任一验证失败则继续修复，全部重新通过前不得提交、推送或创建 MR。
- 在线聚合不得读取原始 PM 表，不补算历史，不保留旧扫描链路。
- 用户编辑任务时生成新的不可变内部版本，从下一个完整 15 分钟窗口生效。
- NATS 标准事件保留至少 40 天，Redis 使用 AOF 和 `noeviction`。
- SQL 使用 Squirrel + pgx，错误使用 `%w` 包装上下文。
- 设计依据：`docs/superpowers/specs/2026-07-25-pm-streaming-aggregation-design.md`。

---

## 一、开发节奏和提交门禁

本次只有三个阶段：

1. **完整开发阶段：** 一次完成数据库结构、事件契约、任务版本、Redis 引擎、Finalizer、API/查询、旧链路删除、配置和测试代码。
2. **集中验证阶段：** 统一运行格式化、构建、全量单测、集成测试、Docker 验收、10000 基站压测和故障注入。
3. **提交 MR 阶段：** 只有集中验证全部通过，才创建分支提交、推送并创建 MR；MR 描述附完整验证证据。

开发阶段禁止：

- 每写一个文件就跑一轮全量测试。
- 每完成一个小模块就部署服务器验证。
- 验证未完成就创建“先看代码”的 MR。
- 为了赶进度跳过失败测试、使用 `--no-verify` 或保留旧扫描链路兜底。

允许在开发阶段做不启动测试环境的快速静态自检，例如 `gofmt`、编辑器诊断、搜索未实现符号和审阅 diff；这些不替代最终集中验证。

## 二、完整开发阶段

### 任务 1：建立新数据库结构并明确删除旧结构

- [ ] **任务交付：** 新旧环境均可得到唯一的新聚合 schema，旧聚合数据只删除、不迁移。

**修改：**

- `omcgo/migrations/000006_pm_streaming_aggregation.sql`
- `omcgo/migrations/tsdb/000002_pm_streaming_aggregation.sql`
- `omcgo/migrations/000001_init_schema.sql`
- `omcgo/migrations/tsdb/000001_tsdb_schema.sql`
- `omcgo/migrations/README.md`

**测试：**

- 新增 `omcgo/cmd/migrate/pm_streaming_aggregation_migration_test.go`
- 新增 `omcgo/test/integration/pm_streaming_aggregation_schema_test.go`
- 修改 `omcgo/test/integration/migration_idempotency_test.go`

**实现内容：**

1. 主库新增：
   - `pm_aggregation_tasks`
   - `pm_aggregation_task_versions`
   - `pm_aggregation_version_metrics`
   - `pm_aggregation_version_members`
2. 时序库新增：
   - `pm_aggregation_outbox`
   - `pm_aggregation_windows`
   - `pm_aggregation_results`
3. 为 outbox 待发布、版本有效期、版本成员、窗口状态、结果查询和结果幂等建立必要索引。
4. 结果表按 `window_start` 建 TimescaleDB hypertable，并配置压缩和保留策略。
5. 增量迁移显式删除旧聚合表、旧聚合状态和临时阻断触发器；只删除，不复制旧数据。
6. 同步更新两个基线 schema，保证全新环境和已有环境升级后的最终结构一致。
7. 不删除 15 分钟原始稀疏表、`pm_files`、`pm_ingest_batches` 和 KPI 元数据。
8. 不删除 KPI 导出仍需使用的通用 `async_jobs` 表。

**迁移测试代码：**

- 新增 schema 契约测试，断言新表、约束、唯一键和索引存在。
- 断言旧聚合结果/水位表和临时阻断触发器不存在。
- 断言重复执行清理语句不会报错。

### 任务 2：定义标准事件和事务 Outbox

- [ ] **任务交付：** 每个成功提交的 PM 文件在同一事务内得到一条可独立消费的标准事件。

**修改：**

- `omcgo/internal/core/event/subjects.go`
- `omcgo/internal/core/event/types.go`
- `omcgo/internal/pm/metrics/model.go`
- `omcgo/internal/pm/metrics/copy_ingest.go`
- `omcgo/internal/pm/collector/collector.go`
- `omcgo/cmd/worker/main.go`

**新增：**

- `omcgo/internal/pm/stream/event.go`
- `omcgo/internal/pm/stream/outbox_repository.go`
- `omcgo/internal/pm/stream/outbox_relay.go`
- `omcgo/internal/pm/stream/event_test.go`
- `omcgo/internal/pm/stream/outbox_repository_test.go`
- `omcgo/internal/pm/stream/outbox_relay_test.go`

**接口：**

```go
type IngestResult struct {
    Ingested     bool
    SourceFileID uuid.UUID
    IngestBatchID uuid.UUID
}

func (r *metrics.PgRepository) CopyIngest(
    ctx context.Context,
    marker metrics.FileMarker,
    counters []model.PMCounter,
    kpis []model.KPIValue,
) (IngestResult, error)

type OutboxRelay interface {
    Run(ctx context.Context) error
}
```

**实现内容：**

1. 增加 `pmaggregation.normalized` subject 和版本化事件结构（使用独立 retained stream）。
2. 从 `BuildSparseMeasurements` 的同一份内存结果生成标准事件，不再次解析 XML，不查询数据库。
3. 将 `CopyIngest` 返回值扩展为包含 `source_file_id` 和 `ingest_batch_id` 的入库结果。
4. 在 `CopyIngest` 现有时序库事务内写 outbox；任何一步失败均回滚文件标记、原始值和 outbox。
5. 相同文件内容命中幂等标记时不新增 outbox；同一来源标识已提交后出现不同内容时拒绝替换，消除“消费前替换”和“消费后替换”的竞态。
6. Relay 使用 `FOR UPDATE SKIP LOCKED` 批量领取、JetStream 发布确认和带退避的失败重试。
7. 事件序列化前校验时间、ID、有限数值和最大载荷；失败必须使原始入库失败并进入现有重试/DLQ。
8. 发布成功后更新 `published_at`，允许“已发布但标记提交失败”导致重复发布，消费端负责幂等。

**测试代码覆盖：**

- 原始值与 outbox 同事务提交/回滚。
- 文件重投和内容替换语义。
- Relay 重复发布、NATS 暂停和恢复。
- 事件校验、序列化和载荷限制。

### 任务 3：重构任务 CRUD 为逻辑任务 + 不可变版本

- [ ] **任务交付：** 用户可以编辑逻辑任务，每次保存生成从下一完整窗口生效的不可变版本快照。

**修改：**

- `omcgo/internal/pm/adhoc/model.go`
- `omcgo/internal/pm/adhoc/handler.go`
- `omcgo/internal/pm/adhoc/repository.go`
- `omcgo/internal/pm/export/source.go`
- `omcgo/internal/pm/export/repository.go`
- `omcmb/frontend-core/src/services/api/pmAdhocApi.ts`
- `omcmb/frontend-core/src/types/pmAdhoc.ts`
- `omcmb/frontend-core/src/hooks/api/usePmAdhoc.ts`
- `omcmb/frontend-core/src/i18n/zh-CN/index.ts`
- `omcmb/frontend-core/src/i18n/en-US/index.ts`
- `omcmb/webcode/src/pages/performance/PmAdhoc/index.tsx`
- `omcmb/webcode/src/pages/performance/PmAdhoc/CreateAdhocTaskDrawer.tsx`
- `omcmb/webcode/src/pages/performance/PmAdhoc/BuiltinMetricEditModal.tsx`
- `omcmb/webcode/src/pages/performance/PmAdhoc/AdhocResultPanel.tsx`

**新增：**

- `omcgo/internal/pm/stream/task_model.go`
- `omcgo/internal/pm/stream/task_repository.go`
- `omcgo/internal/pm/stream/task_service.go`
- `omcgo/internal/pm/stream/task_handler.go`
- `omcgo/internal/pm/stream/task_repository_test.go`
- `omcgo/internal/pm/stream/task_service_test.go`

**接口：**

```go
type TaskService interface {
    Save(ctx context.Context, req SaveTaskRequest) (TaskVersion, error)
    SetEnabled(ctx context.Context, taskID uuid.UUID, enabled bool, actor string) error
    Delete(ctx context.Context, taskID uuid.UUID, actor string) error
}

type TaskVersionRepository interface {
    LoadMatchable(ctx context.Context, at time.Time) ([]TaskVersionSnapshot, error)
}
```

**实现内容：**

1. 保留用户可编辑任务的创建、查看、更新、启用、禁用、删除和权限控制。
2. 每次创建或更新在一个主库事务内：
   - 锁定逻辑任务。
   - 计算连续 `version_no`。
   - 解析指标元数据并固化 `aggregation_op`。
   - 查询并物化设备成员与 `dimension_key`。
   - 将 `effective_from` 对齐到下一个完整 15 分钟边界。
   - 关闭旧版本有效期并创建新版本。
   - 更新 `current_version_id`。
3. 禁止原地修改版本表。
4. 去掉 oneshot、continuous cron、任意历史窗口、重新执行、恢复、运行进度和历史补算语义。
5. 任务保存 API 返回 `version_no`、`effective_from` 和成员数量。
6. 前端提示“从下一个完整窗口生效，不补算历史”。
7. 任务成员快照包含 device、aggregate_group、device_group、product、band、network 六种维度所需键。
8. 任务变更成功后发布轻量 `pmaggregation.task_version.changed` 控制事件；发布失败不影响版本持久化，worker 通过周期刷新兜底。

**测试代码覆盖：**

- 并发更新版本号不重复。
- 新旧版本有效期不重叠。
- 保存时成员和指标规则被完整物化。
- 各维度键正确。
- 禁用、删除不影响已打开窗口。
- 任务权限和前端 API 契约。

### 任务 4：实现任务快照和规则匹配器

- [ ] **任务交付：** 标准事件只通过内存索引匹配任务版本并生成窗口贡献，不访问原始 PM 表。

**新增：**

- `omcgo/internal/pm/stream/snapshot.go`
- `omcgo/internal/pm/stream/matcher.go`
- `omcgo/internal/pm/stream/window.go`
- `omcgo/internal/pm/stream/snapshot_test.go`
- `omcgo/internal/pm/stream/matcher_test.go`
- `omcgo/internal/pm/stream/window_test.go`

**接口：**

```go
type SnapshotStore interface {
    Reload(ctx context.Context) error
    Current() *TaskSnapshot
}

type Matcher interface {
    Match(event NormalizedEvent, snapshot *TaskSnapshot) ([]Contribution, error)
}

func WindowFor(slotStart time.Time, granularity Granularity, location *time.Location) Window
```

**实现内容：**

1. worker 启动时加载当前有效版本和仍有活动窗口的旧版本。
2. 建立 `technology -> device_id -> versions` 倒排索引以及每版本指标集合。
3. 监听任务版本变更事件并原子替换内存快照；每分钟从主库刷新一次作为控制事件丢失兜底。
4. 按事件 15 分钟起点和版本有效期判断是否匹配。
5. 为匹配版本计算小时、日、周、月窗口，统一使用业务时区边界并在事件/存储中使用 UTC。
6. 按 `object_ldns` 和指标规则过滤事件内容。
7. 输出最小化 `Contribution`，包含任务版本、维度键、窗口、指标值和预期完整性信息。
8. 匹配器不得访问时序库原始 PM 表。

**测试代码覆盖：**

- 生效边界前后事件。
- 更新时旧窗口和新窗口分别命中正确版本。
- 各时区的小时、日、周、月边界。
- 指标、设备、制式和 LDN 过滤。
- 六种维度的 contribution。

### 任务 5：实现 Redis 原子窗口累加

- [ ] **任务交付：** Redis 在一个 Lua 原子操作中完成文件去重、基础统计量累加和窗口完整性判断。

**新增：**

- `omcgo/internal/pm/stream/redis_store.go`
- `omcgo/internal/pm/stream/accumulate.lua`
- `omcgo/internal/pm/stream/redis_store_test.go`
- `omcgo/internal/pm/stream/accumulate_integration_test.go`

**接口：**

```go
type WindowStore interface {
    Accumulate(ctx context.Context, contribution Contribution) (AccumulateResult, error)
    Read(ctx context.Context, key WindowKey) (WindowState, error)
    TryFinalizeLock(ctx context.Context, key WindowKey, ttl time.Duration) (Lock, error)
    Delete(ctx context.Context, key WindowKey) error
}
```

**实现内容：**

1. 使用 hash tag 生成窗口内同槽位 Redis 键。
2. Lua 原子完成：
   - `source_file_id` 去重。
   - `sum/count/min/max` 更新。
   - `(device_id, 15min_slot)` 到达标记。
   - `received_slots` 更新。
   - 完整性判断。
3. 预期时隙按版本成员数乘以窗口内 15 分钟时隙数计算；同一设备同一时隙只计一次。
4. 窗口首次贡献时在时序库创建 `pm_aggregation_windows(open)`，唯一冲突视为已存在。
5. Redis 键 TTL 必须覆盖最大窗口、5 分钟宽限和故障恢复时间；窗口发布后主动删除。
6. Redis `noeviction` 和 AOF 配置在启动时检查，不符合要求则聚合 worker 拒绝就绪并告警，原始 PM 入库仍可运行。
7. 暴露重复事件、Lua 耗时、活动窗口、Redis 错误和内存水位指标。

**测试代码覆盖：**

- 重复、乱序和并发相同文件。
- 多指标、多对象、多维度的 sum/avg/min/max。
- 完整时隙判断。
- Redis 重启后的 AOF 状态恢复。
- Redis 内存策略不合规时的启动守门。

### 任务 6：实现消费器、窗口关闭和 Finalizer

- [ ] **任务交付：** 完整或超时窗口只发布一次，失败可重试，Redis 状态缺失可从 NATS 重建。

**新增：**

- `omcgo/internal/pm/stream/consumer.go`
- `omcgo/internal/pm/stream/window_repository.go`
- `omcgo/internal/pm/stream/finalizer.go`
- `omcgo/internal/pm/stream/recovery.go`
- `omcgo/internal/pm/stream/consumer_test.go`
- `omcgo/internal/pm/stream/finalizer_test.go`
- `omcgo/internal/pm/stream/recovery_test.go`

**接口：**

```go
type Finalizer interface {
    Finalize(ctx context.Context, key WindowKey, reason CloseReason) error
}

type Recovery interface {
    RestoreActiveWindows(ctx context.Context) error
}

type StreamConsumer interface {
    Run(ctx context.Context) error
}
```

**实现内容：**

1. durable pull consumer 并发拉取标准事件。
2. 只有全部 contribution 成功写入 Redis 后才 ACK NATS 消息。
3. 完整窗口立即投递 finalizer；每 30 秒扫描到期 `open/failed` 窗口，执行 `window_end + 5min` 强制关闭。
4. Finalizer 获取带租约 Redis 锁，并通过数据库状态条件更新防止并发重复发布。
5. 一次读取窗口累加器，批量写入统一结果表；结果与 `published` 状态在同一时序库事务提交。
6. 写库失败保留 Redis 状态并可重试。
7. 已关闭窗口收到迟到事件时只增加指标并 ACK，不重开、不修正。
8. worker 启动时恢复 `open/finalizing/failed` 窗口。
9. Redis 窗口缺失时，按窗口范围从 NATS retained stream 创建 replay consumer 重建；严禁查询原始 PM 表。
10. NATS 保留不足导致无法重建时将窗口标记 `failed` 并产生高优先级告警，不伪造结果。

**测试代码覆盖：**

- 完整关闭、超时关闭、缺失时隙。
- concurrent finalizer。
- 最终写库失败、提交成功后进程崩溃。
- worker 重启、Redis 重启和 NATS replay。
- 关闭后的迟到事件。

### 任务 7：切换查询、报表和导出到统一结果

- [ ] **任务交付：** 15 分钟继续查原始值，其余粒度和维度统一查新结果表。

**修改：**

- `omcgo/internal/pm/metrics/pg_repository.go`
- `omcgo/internal/pm/device_query_service.go`
- `omcgo/internal/pm/export/query.go`
- `omcgo/internal/pm/export/source.go`
- `omcgo/internal/pm/export/runner.go`
- `omcgo/internal/pm/handler.go`
- `omcgo/internal/pm/adhoc/handler.go`
- `omcgo/internal/pm/export/handler.go`
- `omcmb/frontend-core/src/services/api/pmAggregationApi.ts`
- `omcmb/frontend-core/src/services/api/pmQueryApi.ts`
- `omcmb/frontend-core/src/services/api/kpiExportApi.ts`
- `omcmb/frontend-core/src/types/pmQuery.ts`
- `omcmb/frontend-core/src/types/kpiExport.ts`
- `omcmb/webcode/src/pages/performance/KPIQuery/index.tsx`
- `omcmb/webcode/src/pages/performance/PmDashboard/TaskDashboardPane.tsx`
- `omcmb/webcode/src/pages/performance/PmAdhoc/AdhocResultPanel.tsx`

**新增：**

- `omcgo/internal/pm/stream/result_repository.go`
- `omcgo/internal/pm/stream/result_service.go`
- `omcgo/internal/pm/stream/result_repository_test.go`
- `omcgo/internal/pm/stream/result_service_test.go`

**接口：**

```go
type ResultRepository interface {
    Query(ctx context.Context, filter ResultFilter) ([]AggregationResult, error)
}

type ResultService interface {
    Query(ctx context.Context, filter ResultFilter, actor string) ([]AggregationResult, error)
}
```

**实现内容：**

1. 15 分钟明细保持现有稀疏原始查询。
2. 小时、日、周、月和任务结果统一查询 `pm_aggregation_results`。
3. 查询条件支持任务、版本、粒度、维度、维度键、LDN、指标和时间范围。
4. API 和导出增加 `complete`、`missing_slots`、`task_version_id`。
5. 导出 runner 保留通用 async job 基础设施，但数据源不再读取旧自然桶和 adhoc 结果表。
6. 删除前端“补跑、重试、恢复、历史执行进度”等不再存在的操作。
7. 用户可见文案全部进入 i18n。

**测试代码覆盖：**

- 六种维度和四种粒度查询。
- 完整/不完整窗口展示。
- 权限过滤和导出。
- 15 分钟查询未受影响。

### 任务 8：彻底删除旧聚合运行链路

- [ ] **任务交付：** worker 只装配新流式聚合，仓库中不存在可启动的旧扫描、补跑或 adhoc 执行路径。

**删除：**

- `omcgo/internal/pm/aggregator/` 中自然桶、组聚合、rollup、watermark、sparse maintenance、补跑及其专属测试
- `omcgo/internal/pm/adhoc/executor.go`
- `omcgo/internal/pm/adhoc/worker.go`
- `omcgo/internal/pm/adhoc/window.go`
- `omcgo/internal/pm/adhoc/progress_bridge.go`
- `omcgo/internal/pm/adhoc/progress_hub.go`
- 仅服务旧执行模型的 scheduler、run history、结果写入代码及测试

**修改：**

- `omcgo/cmd/worker/aggregator.go`
- `omcgo/cmd/worker/adhoc.go`
- `omcgo/cmd/worker/main.go`
- `omcgo/cmd/worker/timezone.go`
- `omcgo/cmd/worker/retention.go`

**实现内容：**

1. 用新的 `startPMAggregationStream` 装配 Relay、snapshot、consumer、Redis store、timeout scanner、finalizer 和 recovery。
2. 删除旧 8 类自然/组聚合 runner、cron、启动补跑、水位推进和 sparse maintenance 启动代码。
3. 删除 adhoc worker pool 和 continuous scheduler。
4. KPI 导出 worker、其他业务 async jobs 和 retention 独立保留。
5. 全仓搜索并删除对旧聚合表、旧 job type、旧水位和旧环境变量的引用。
6. 不保留运行时开关回退到旧聚合方案。

### 任务 9：部署配置、健康检查和监控

- [ ] **任务交付：** Docker 环境具备持久化专用 Redis、足够 NATS 保留、健康检查和完整指标。

**修改：**

- `deployments/docker/docker-compose.yml`
- `deployments/docker/docker-compose.local.yml`
- `deployments/docker/docker-compose.test.yml`
- `deployments/docker/README.md`
- `omcgo/cmd/worker/etc/config.dev.yaml`
- `omcgo/cmd/worker/etc/config.prod.yaml`
- `omcgo/cmd/worker/etc/config.test.yaml`
- `omcgo/internal/pm/metrics.go`

**新增：**

- `omcgo/internal/pm/stream/config.go`
- `omcgo/internal/pm/stream/metrics.go`
- `omcgo/internal/pm/stream/health.go`
- `omcgo/internal/pm/stream/config_test.go`
- `omcgo/internal/pm/stream/metrics_test.go`
- `omcgo/internal/pm/stream/health_test.go`

**实现内容：**

1. Redis 开启 AOF、`appendfsync everysec`、`noeviction`，使用持久卷。
2. NATS JetStream 为聚合 subject 配置至少 40 天保留、文件流压缩和经过真实 PM 文件验证的 `max_payload`；保留期不得短于最长聚合窗口加关闭宽限和计划恢复时间。
3. 增加聚合开关、宽限期、outbox 批量、消费并发和 finalizer 并发配置。
4. readiness 区分原始 PM 入库和聚合子系统状态，避免 Redis 故障阻止原始入库 worker 处理文件。
5. 增加设计文档列出的全部 Prometheus 指标和结构化日志字段。
6. 更新部署说明、故障恢复步骤和正式切换步骤。

### 任务 10：补齐集中验证所需测试工具与证据采集

- [ ] **任务交付：** 万站压测和全部故障场景可一次运行并产出固定格式证据。

**新增或修改：**

- 新增 `omcgo/cmd/pm-stream-loadtest/main.go`
- 新增 `omcgo/internal/pm/streamtest/generator.go`
- 新增 `omcgo/internal/pm/streamtest/faults.go`
- 新增 `omcgo/internal/pm/streamtest/observer.go`
- 新增 `omcgo/internal/pm/streamtest/testdata/lte-pm-template.xml`
- 新增 `omcgo/internal/pm/streamtest/generator_test.go`
- 新增 `omcgo/internal/pm/streamtest/faults_test.go`
- 新增 `docs/superpowers/evidence/2026-07-25-pm-streaming-aggregation-validation.md`

**实现内容：**

1. 压测工具能够记录：
   - 首个/最后一个文件接收和提交时间。
   - 文件处理时延分位数。
   - NATS pending/ack/redelivery。
   - outbox backlog。
   - Redis 活动窗口和内存。
   - 数据库临时文件、CPU、I/O。
   - 聚合窗口发布时刻和缺失情况。
2. 故障注入支持：
   - 重复消息。
   - 乱序消息。
   - 缺失设备/时隙。
   - 关闭后迟到。
   - 任务更新。
   - worker 重启。
   - Redis 重启和状态重建。
   - NATS 暂停/恢复。
   - 最终数据库写失败。
   - 并发 finalizer。
3. 增加 SQL 审计采样，证明聚合 worker 没有读取原始 PM 表。
4. 验证输出采用固定文件名，便于附到 MR。

## 三、集中验证阶段

代码、配置、迁移和测试代码全部完成后，只在此阶段统一验证。

### 任务 11：一次性执行静态检查、构建和全量测试

- [ ] **验证门禁：** 后端格式化、构建、全量测试、前端类型检查、空库和升级迁移、集成测试全部通过。

按顺序执行：

```bash
cd omcgo
go fmt ./...
go build ./...
go test ./...

cd ../omcmb
npm run typecheck
```

然后启动完整测试栈，执行：

- 主库和时序库从空库应用基线 schema。
- 从当前生产结构应用 `000006` / `tsdb 000002` 增量迁移。
- 新聚合包集成测试。
- API、导出、权限和前端契约测试。

失败处理：

- 记录失败原因并修复所有相关代码。
- 修复完成后重新执行本任务的完整验证集合，不只单独重跑失败用例。
- 全部通过前不得进入万站压测。

### 任务 12：一次性执行 10000 基站性能和故障验收

- [ ] **验证门禁：** 万站 SLO、无原始表读取和全部故障恢复场景通过。

在与生产配置相当的 Docker 环境执行至少一个完整 10000 基站 15 分钟周期：

1. 创建覆盖 device、device_group、product、band、network、aggregate_group 的代表性任务。
2. 发送 10000 个 PM 文件。
3. 等待原始入库和所有预期聚合窗口完成。
4. 采集固定证据并核对设计 SLO。
5. 执行全部故障注入场景。
6. 执行 SQL 审计，确认聚合进程对原始 PM 表读取为 0。
7. 比较聚合开启/关闭时原始入库耗时，确认聚合没有让 10000 文件总耗时超过 4 分钟。

必须满足：

- 10000 文件全部提交，耗时不超过 4 分钟。
- 99% 单文件处理小于 1 秒。
- 原始 PM NATS pending 最终为 0，无异常重投。
- 完整窗口在最后预期事件后 60 秒内发布。
- 超时窗口在宽限期后 60 秒内发布。
- 重复、乱序、重启和写库重试不产生重复结果。
- Redis 丢失窗口可从 NATS 恢复。
- 聚合不读取原始 PM 表。
- 数据库临时空间没有由聚合导致的持续增长。

任一指标不满足：

- 停止 MR 流程。
- 修复后重新执行任务 11 和任务 12 的完整验证。

## 四、提交与 MR 阶段

### 任务 13：全部验证通过后提交并创建 MR

- [ ] **提交门禁：** 只有任务 11、12 的完整证据均通过后，才允许创建提交、推送和 MR。

只有任务 11、12 全部通过且证据齐全后执行：

1. 复查 `git status` 和完整 diff，确认没有本机 AI 配置、凭据、压测数据或无关改动。
2. 创建分支：

```bash
git switch -c codex/pm-streaming-aggregation
```

3. 用 `git diff --name-only` 审核本次精确文件清单，逐项暂存，不使用 `git add -A`，然后提交一个完整功能提交：

```bash
git commit -m "feat(pm): 改为在线流式聚合"
```

4. 推送分支。
5. 使用 `glab mr create` 创建 MR。
6. MR 描述必须包含：
   - 方案摘要和不兼容切换声明。
   - 删除的旧聚合路径。
   - schema 变更和无历史迁移说明。
   - 全量构建/测试结果。
   - 10000 基站性能数据。
   - 故障测试结果。
   - SQL 审计结果。
   - 部署、回退和数据清理步骤。

如果提交钩子、推送或 MR 检查失败，不得绕过；修复后重新执行必要的集中验证，再继续提交。

## 五、正式部署清单

MR 合并且准备上线后：

1. 停止业务入口和 worker。
2. 确认没有在途 PM 文件和未完成数据库事务。
3. 应用主库 `000006` 和时序库 `000002`；旧聚合任务、结果、状态直接删除。
4. 删除生产环境临时聚合阻断触发器。
5. 启动持久化 Redis 和 JetStream，检查 AOF、noeviction、保留期和磁盘。
6. 部署 app、ACS、worker 和前端。
7. 检查原始入库、outbox relay、consumer、Redis、finalizer readiness。
8. 创建新的聚合任务，确认返回下一个完整窗口的 `effective_from`。
9. 观察一个完整 15 分钟周期和一个小时窗口。
10. 未达到验收条件时停止新聚合 worker，修复软件后重新清空聚合状态部署；不恢复旧扫描链路和旧聚合数据。

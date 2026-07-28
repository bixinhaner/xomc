# OMC Resource Monitoring, Queue Governance, and Disk Write Protection Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在现有 OMC 监控基础上，补齐 CPU、内存、磁盘、MinIO、监控数据盘和持久化队列的可观测性；在 Grafana 中提供可验证的资源与队列治理视图；在 OMC 管理页面配置磁盘写入保护和数据有效期，并将准入策略真正接入日志、上报文件、备份、报表、Trace 等业务写入路径。

**Architecture:** Prometheus 负责指标采集和告警规则，Grafana 负责展示；OMC app/acs/worker 负责产生 NATS JetStream、Redis、PostgreSQL、PM/MR/Trace 和写入保护指标；`storageprotection` 领域服务负责策略、状态机、准入判定和审计；业务写入者通过统一 `WriteAdmission` 接口在创建对象、打开文件或提交持久化任务前检查；清理服务按统一 retention policy 执行，并上报释放空间和失败指标。

**Tech Stack:** Go、Gin、pgx/Squirrel、PostgreSQL/Goose、Redis、NATS JetStream、MinIO、Prometheus client_golang、Prometheus rule files、Grafana dashboard JSON、React/TypeScript、React Query、Ant Design、Vitest、Playwright。

## Global Constraints

- 应用内存 Channel、goroutine 内部队列、SSE 推送缓存不纳入本期队列监控；本期只监控 NATS JetStream、Redis 任务队列、PostgreSQL 持久化任务/Outbox、PM/MR/Trace/备份等可恢复队列。
- Redis 队列扫描必须使用 `SCAN`，禁止 `KEYS`；Prometheus/Grafana 标签禁止使用设备号、对象名、完整路径等高基数值。
- Docker Compose 环境统一使用 `service`、`container_id`、`instance`、`mountpoint` 等标签；不再新增依赖 `namespace="omcgo"` 或 cAdvisor `name` 标签的查询。
- 指标缺失、采集失败、队列为空三种状态必须可区分：空队列输出 0，采集失败保留失败/新鲜度指标，查询无 series 时 Grafana 明确显示 No data。
- Prometheus rules 是告警事实源，Grafana 只展示状态和历史；Alertmanager 负责通知，不在 Grafana JSON 中复制告警判定逻辑。
- 磁盘写入保护只拦截 OMC 业务可控写入，不能宣称阻止 PostgreSQL WAL、Docker daemon、Prometheus/Loki/Tempo 内部写入；日志在策略允许时执行降级，错误日志、安全审计和写入保护审计不可静默丢失。
- 三条 Goose 迁移流独立编号：主库 schema 使用 `migrations/000002`，seed 按现有 `000002` 之后使用 `migrations/seed/000003`；TSDB 只有在实际增加时序对象时才增加 `migrations/tsdb/000002`。
- 前端所有用户可见文字进入中英文 i18n；页面、菜单和浏览器验收以 `omcmb/webcode` 的 V1 实际路由和 DOM 为准。

---

## 现状基线与目标差距

| 能力 | 当前状态 | 本计划处理方式 |
|---|---|---|
| 主机 CPU/内存/基础磁盘 | 已有 node_exporter 指标、部分 Grafana 面板和告警 | 保留现有能力，补 inode、增长预测、数据盘/监控盘维度和统一标签 |
| 容器资源 | cAdvisor 指标存在，但现有面板按 `name` 查询，现场 `name` 无数据 | 修正为实际 cAdvisor 标签或在采集层规范化，删除无效 K8s `namespace` 查询 |
| MinIO | total/free 指标和 `/system/info` 采集已有 | 新增 Grafana 展示、bucket/数据分类视图、容量告警，并纳入写入准入 |
| NATS | 连接健康和 JetStream 总存储已有；consumer pending 查询无数据 | 增加 OMC JetStream observer，输出 stream/consumer backlog、最老消息年龄、速率和采集失败 |
| Redis 队列 | 旧 `acs:cmdq:{device_sn}`、新 `acs:taskq:{device_sn}` 有实现但无统一指标/面板 | 使用 SCAN 周期采样总长度、活跃设备数、最大长度、最老年龄，并保留对账/恢复指标 |
| PostgreSQL 持久化队列 | `omc_tasks_pending_total`、`omc_async_jobs_queue_depth` 已有，但缺统一队列维度面板 | 增加 queue/status、oldest、failed、dead-letter、processed、observer failure 指标 |
| PM 队列 | 指标代码已有，但现场 `omc_pm_queue_pending` 无 series | worker 启动时初始化标签，空队列保持 0，采样失败通过 failure/freshness 表达 |
| 磁盘告警 | HostDisk 告警已有 | 新增 OMC 管理策略、状态机、准入拒写/降级、恢复迟滞和审计 |
| 有效期/清理 | PM、日志、告警、备份各自维护 | 建立统一 policy 模型和清理编排，逐项迁移已有实现 |

---

## Phase 0 — 契约、基线和验证工具

### Task 1: 固化指标/标签/队列目录契约

**Files:**

- Modify: `docs/design/资源监控与队列治理及磁盘写入保护功能设计-20260728.md`
- Modify: `deployments/monitoring/README.md`
- Modify: `omcgo/internal/core/event/metrics.go`
- Modify: `omcgo/internal/task/metrics.go`
- Test: `omcgo/internal/core/event/metrics_test.go`
- Test: `omcgo/internal/task/metrics_test.go`

**Steps:**

- [ ] 将指标名称、标签、单位、是否允许 0、失败时的可用性语义整理成代码注释和 README 表格；队列目录固定为 `device_tasks`、`async_jobs`、`parameter_sync_outbox`、`northbound_outbox`、`pm_kpi_export`、`trace_export`、`backup_tasks`、`dead_letters`。
- [ ] 在 event/task 现有 metrics 测试中增加低基数断言，禁止 device SN、完整 Redis key 和对象路径出现在新指标标签中。
- [ ] 为 PM、任务和新持久化队列约定 `pending`、`oldest_age_seconds`、`failed_total`、`dead_letter_total`、`processed_total`、`observer_failures_total` 的命名和状态值。
- [ ] 在 README 记录本期排除的应用内存队列，并说明 PromQL 查询失败与真实 0 的区别。

**Verification:**

```bash
cd omcgo && go test ./internal/core/event ./internal/task
```

**Done when:** 后续任务只使用这一套指标/标签契约，测试能阻止高基数标签和同名异义指标回归。

### Task 2: 建立 Grafana/PromQL 静态校验入口

**Files:**

- Create: `deployments/monitoring/tests/validate-dashboards.sh`
- Create: `deployments/monitoring/tests/promql-probes.txt`
- Modify: `deployments/monitoring/README.md`

**Steps:**

- [ ] 脚本校验四个现有 dashboard JSON 和新增 dashboard JSON 可被 `jq` 解析，UID 唯一，panel query 不含 `namespace="omcgo"`、`name=~` 这类已确认失效筛选。
- [ ] 将 CPU、内存、node filesystem、cAdvisor、MinIO、NATS、Redis、PG、PM 和写入保护查询写入 probe 清单，区分“必须有数据”“允许无数据但必须显示 No data”“空队列必须为 0”。
- [ ] README 写明验证顺序：Prometheus `/api/v1/query` → Grafana panel → 浏览器实际 dashboard；禁止只凭 JSON 中存在 panel 判定功能完成。

**Verification:**

```bash
chmod +x deployments/monitoring/tests/validate-dashboards.sh
deployments/monitoring/tests/validate-dashboards.sh
```

**Done when:** 每次 dashboard 或 PromQL 修改都能在本地先发现 JSON、UID、旧标签和关键查询错误。

---

## Phase 1 — 修复现有资源监控并补齐队列指标

### Task 3: 修复现有 Grafana 资源面板和标签口径

**Files:**

- Modify: `deployments/monitoring/grafana/dashboards/omc-resources.json`
- Modify: `deployments/monitoring/grafana/dashboards/omc-infra.json`
- Modify: `deployments/monitoring/grafana/dashboards/nginx-host-overview.json`
- Modify: `deployments/monitoring/grafana/dashboards/omc-overview.json`
- Modify: `deployments/monitoring/prometheus.yml`
- Test: `deployments/monitoring/tests/validate-dashboards.sh`

**Steps:**

- [ ] 将容器 CPU、内存、重启、OOM、网络和 IO 查询改为现场 cAdvisor 实际存在的标签；优先使用 `container_id`/`id` 与 `service`，不得依赖 `name`。
- [ ] 删除或改写所有 `namespace="omcgo"` 的旧 Kubernetes 查询，统一 Docker Compose 的 `job`、`instance`、`service` 维度。
- [ ] 保留现有 node CPU、内存、磁盘告警语义，增加 inode 使用率、文件系统增长率、预测耗尽时间和目标挂载点筛选。
- [ ] 在 Prometheus scrape/metric relabel 配置中补充可控的 cAdvisor → `container_id`、Compose service 标签映射；保留原始标签仅用于兼容过渡。
- [ ] 检查每个 panel 的 No data、零值、单位、时间范围和刷新周期，统一默认 `now-1h`、刷新 30 秒。

**Verification:**

```bash
deployments/monitoring/tests/validate-dashboards.sh
jq empty deployments/monitoring/grafana/dashboards/*.json
```

**Done when:** 现场 `container_cpu_usage_seconds_total{...}` 和容器内存查询均能返回 series，Grafana 不再出现“面板存在但无数据”。

### Task 4: 完善存储目标和 MinIO/监控数据盘指标

**Files:**

- Modify: `omcgo/internal/core/components/storage_metrics.go`
- Modify: `omcgo/internal/core/components/storage_metrics_test.go`
- Modify: `omcgo/internal/core/components/sysinfo.go`
- Modify: `deployments/monitoring/prometheus.yml`
- Create: `deployments/monitoring/storage-targets.yml`
- Modify: `deployments/monitoring/grafana/dashboards/omc-resources.json`
- Modify: `deployments/monitoring/grafana/dashboards/omc-infra.json`

**Steps:**

- [ ] 保留 `/system/info` 的来源状态和采集时间，扩展 host filesystem 的 mountpoint、inode 总量/可用量、使用比例和 stale 状态。
- [ ] 为 Prometheus recording rules/目标映射定义 `target_type`、`target_id`、`mountpoint`、`write_scope`；至少覆盖 application、PostgreSQL、MinIO、Prometheus、Loki、Tempo 和 Docker data root。
- [ ] 增加 MinIO 总容量、已用/可用容量和有限 bucket 分类指标；bucket 标签只允许固定业务分类，不允许任意对象路径。
- [ ] 对 `/var/lib/postgresql`、`/var/lib/minio`、监控数据目录等部署目标提供明确映射；未配置目标时仍可展示 node filesystem，不把缺配置伪装成 0。
- [ ] 为容量、inode、增长预测和 MinIO bucket 增加 dashboard panel，并在 `/system/info` 中将不可用来源显示为 unavailable。

**Verification:**

```bash
cd omcgo && go test ./internal/core/components
deployments/monitoring/tests/validate-dashboards.sh
```

**Done when:** MinIO 已有的 total/free 指标在 Grafana 可见；主机数据盘和监控数据盘可按目标区分；容量采集失败不显示为“剩余 0”。

### Task 5: 增加 NATS JetStream backlog observer

**Files:**

- Create: `omcgo/internal/core/components/nats/queue_metrics.go`
- Create: `omcgo/internal/core/components/nats/queue_metrics_test.go`
- Modify: `omcgo/internal/core/components/nats/nats.go`
- Modify: `omcgo/internal/core/components/nats/conn_metrics.go`
- Modify: `omcgo/internal/core/components/infra.go`
- Modify: `omcgo/internal/core/event/queue_sampler.go`
- Modify: `omcgo/internal/core/event/queue_sampler_test.go`
- Modify: `omcgo/cmd/app/provider/modules.go`

**Steps:**

- [ ] 使用现有 `NATSClient.JS` 周期读取固定 stream 和 durable consumer 的 `StreamInfo`/`ConsumerInfo`，不额外建立连接，不使用任意用户输入作为标签。
- [ ] 注册 `omc_nats_stream_messages`、`omc_nats_stream_bytes`、`omc_nats_stream_max_bytes`、`omc_nats_consumer_pending`、`omc_nats_consumer_ack_pending`、`omc_nats_consumer_redelivered_total`、`omc_nats_consumer_oldest_age_seconds`、速率、observer failure 指标。
- [ ] 对已知 consumer 在启动时初始化 0；查询失败时不把旧值改成 0，同时增加 failure 和 sample timestamp/freshness 指标。
- [ ] 复用现有 PM queue sampler 的 subject/durable 约定，保证 PM 指标 empty=0、失败可辨识，并保持背压读取接口兼容。
- [ ] 在 app/worker 生命周期中启动和停止 observer，避免重复注册、goroutine 泄漏和应用关闭阻塞。

**Verification:**

```bash
cd omcgo && go test ./internal/core/components/nats ./internal/core/event
```

**Done when:** 停止一个测试 consumer 后 Prometheus 能看到 pending/oldest/failure 变化，正常空队列能看到 0；`nats_jetstream_consumer_num_pending` 不再作为 OMC backlog 的唯一依赖。

### Task 6: 增加 Redis 旧/新任务队列观测和对账指标

**Files:**

- Create: `omcgo/internal/task/queue_metrics.go`
- Create: `omcgo/internal/task/queue_metrics_test.go`
- Modify: `omcgo/internal/task/redis_queue.go`
- Modify: `omcgo/internal/task/service.go`
- Modify: `omcgo/internal/task/reconciler.go`
- Modify: `omcgo/internal/task/metrics.go`
- Modify: `omcgo/internal/task/redis_queue_ttl_test.go`
- Modify: `omcgo/cmd/app/provider/modules.go`

**Steps:**

- [ ] 为 `acs:cmdq:{device_sn}` 和 `acs:taskq:{device_sn}` 建立固定 `queue_family`，用 `SCAN` 统计总长度、活跃设备数、最大长度、最老任务年龄和扫描耗时。
- [ ] 将任务内容中的设备号只用于内部统计，不进入 Prometheus label；Top N 通过受控 API 返回脱敏/哈希后的设备标识，不做全量 label 化。
- [ ] 增加 `omc_task_queue_redis_pg_diff`、`omc_task_queue_recovery_action_total`、`omc_task_queue_write_failures_total`，并与现有 Redis↔PG reconciler 结果对齐。
- [ ] 为 Redis 超时、扫描失败、TTL 异常、空 key 集合分别写单元测试；明确扫描结果为 0 与 Redis 不可用的区别。
- [ ] 将采样周期、SCAN count、最大扫描耗时和失败处理放入可配置项，默认值不能导致生产环境使用 `KEYS` 或无限扫描。

**Verification:**

```bash
cd omcgo && go test ./internal/task
```

**Done when:** 写入 old/new 两类 Redis task key 后，Prometheus 能按 `queue_family` 展示积压；Redis 不可用时 panel 显示采集失败而不是 0。

### Task 7: 增加 PostgreSQL 持久化队列和 PM/MR/Trace 队列指标

**Files:**

- Create: `omcgo/internal/core/components/persistent_queue_metrics.go`
- Create: `omcgo/internal/core/components/persistent_queue_metrics_test.go`
- Modify: `omcgo/internal/core/asyncjob/metrics.go`
- Modify: `omcgo/internal/core/asyncjob/repository.go`
- Modify: `omcgo/internal/northbound/push/outbox.go`
- Modify: `omcgo/internal/paramsync/outbox.go`
- Modify: `omcgo/internal/pm/metrics.go`
- Modify: `omcgo/internal/pm/stream/metrics.go`
- Modify: `omcgo/internal/core/event/metrics.go`
- Modify: `omcgo/internal/core/event/queue_sampler.go`
- Test: `omcgo/internal/core/asyncjob/repository_test.go`
- Test: `omcgo/internal/pm/metrics_test.go`
- Test: `omcgo/internal/pm/stream/metrics_test.go`

**Steps:**

- [ ] 新增统一持久化队列 observer，按 `queue`、`status` 输出 pending、oldest age、failed、dead-letter、processed 和 observer failure；查询使用固定 SQL 和允许列表，禁止拼接外部表名。
- [ ] 将 `device_tasks`、`async_jobs`、parameter sync outbox、northbound outbox、PM KPI export、Trace export、backup tasks、dead letters 映射到统一 queue 名称。
- [ ] 不删除 `omc_tasks_pending_total`、`omc_tasks_backlog_total`、`omc_async_jobs_queue_depth`；通过兼容层或 recording rule 让旧面板继续工作，新增面板使用统一指标。
- [ ] PM/MR/Trace worker 启动时为已知 subject/durable 注册 0，采样成功更新时间戳，采样失败保留 unavailable 语义；确认 aggregation `Ready` 与 queue backlog 不混用。
- [ ] 对空表、失败状态、死信、旧任务、数据库超时分别做 repository/observer 测试。

**Verification:**

```bash
cd omcgo && go test ./internal/core/asyncjob ./internal/core/components ./internal/pm ./internal/pm/stream ./internal/paramsync ./internal/northbound/push
```

**Done when:** PostgreSQL 中存在待处理任务时可按 queue/status 查到 backlog，PM queue empty 有 0，PM observer 失败可通过 freshness/failure 指标识别。

---

## Phase 1.5 — 统一 Grafana 视图和告警

### Task 8: 新增“存储与队列治理”Dashboard 和规则

**Files:**

- Create: `deployments/monitoring/grafana/dashboards/omc-storage-queue-governance.json`
- Create: `deployments/monitoring/alerts/storage-queue-alerts.yml`
- Modify: `deployments/monitoring/prometheus.yml`
- Modify: `deployments/monitoring/grafana/provisioning/dashboards/default.yml`
- Modify: `deployments/monitoring/README.md`
- Test: `deployments/monitoring/tests/validate-dashboards.sh`

**Steps:**

- [ ] Dashboard 固定五行：主机/容器资源、文件系统/MinIO、NATS、Redis、PostgreSQL/PM/写入保护与清理；所有 panel 使用统一 labels 和 30 秒刷新。
- [ ] 每个 queue panel 同时展示 pending、oldest age、增长速率、失败/重投递、freshness；无数据、0 和采集失败使用不同的显示配置。
- [ ] Prometheus rules 覆盖 CPU、内存、inode、预测磁盘耗尽、NATS pending/oldest/stall、Redis queue、PG queue、PM freshness、MinIO 容量、write rejected、cleanup failure。
- [ ] 默认阈值采用设计文档约定：warning 80%、block 90%、recover 85%；告警表达式使用状态机/策略实际指标，不在 Grafana 中硬编码另一套阈值。
- [ ] 将新 dashboard 纳入 provisioning，并确保 dashboard UID、folder、datasource UID 与当前环境一致。

**Verification:**

```bash
deployments/monitoring/tests/validate-dashboards.sh
promtool check rules deployments/monitoring/alerts/*.yml
```

**Done when:** 新 dashboard 在现有 Grafana OMC folder 可打开；上述各类队列和存储面板能定位“没有数据”是无指标、查询错误还是实际 0。

---

## Phase 2 — 磁盘写入保护领域能力

### Task 9: 建立存储保护策略、状态机和准入服务

**Files:**

- Create: `omcgo/migrations/000002_storage_protection.sql`
- Create: `omcgo/internal/storageprotection/model.go`
- Create: `omcgo/internal/storageprotection/repository.go`
- Create: `omcgo/internal/storageprotection/pg_repository.go`
- Create: `omcgo/internal/storageprotection/evaluator.go`
- Create: `omcgo/internal/storageprotection/metrics.go`
- Create: `omcgo/internal/storageprotection/service.go`
- Create: `omcgo/internal/storageprotection/*_test.go`
- Modify: `omcgo/internal/core/components/storage_metrics.go`
- Modify: `omcgo/internal/core/appconfig/config.go`
- Modify: `omcgo/cmd/app/provider/modules.go`

**Steps:**

- [ ] 主库迁移新增 `storage_protection_policies`、状态/事件审计所需表和约束；策略字段包括 target、write scope、enabled、warn/block/recover、check interval、unknown behavior、version、timestamps。
- [ ] 加入数据库约束 `0 <= warn < recover < block <= 100`，同一 target/scope 只能有一个启用策略；无策略时仅监控，不改变业务写入行为。
- [ ] 实现 `normal → warning → blocked`、`blocked → warning → normal` 和 `unknown` 状态机；连续两次检查确认，恢复使用 hysteresis，防止磁盘抖动导致反复拒写/放行。
- [ ] 实现设计文档规定的接口：`WriteAdmission.Check(ctx, StorageTarget, WriteScope) AdmissionDecision`；返回 Allowed、State、Reason、RetryAfter、ObservedAt。
- [ ] 注册 `omc_storage_capacity_bytes`、`omc_storage_used_bytes`、`omc_storage_used_ratio`、`omc_storage_admission_state`、write rejected/degraded、check failures、policy info、cleanup released bytes 指标。
- [ ] unknown 行为可配置为 fail-open 或 fail-closed，但必须记录原因、审计事件和 Prometheus failure；默认值在部署文档中明确。

**Verification:**

```bash
cd omcgo && go test ./internal/storageprotection ./internal/core/components
go test ./test/integration -run Migration
```

**Done when:** 策略可以持久化、热加载，状态机在测试中经过 warning/block/recover/unknown 全路径；准入服务不依赖具体业务写入者。

### Task 10: 将准入检查接入业务写入路径

**Files:**

- Modify: `omcgo/internal/acs/upload/handler.go`
- Modify: `omcgo/internal/acs/upload/uploader.go`
- Modify: `omcgo/internal/transfer/bridge.go`
- Modify: `omcgo/internal/core/storage/router.go`
- Modify: `omcgo/internal/backup/executor.go`
- Modify: `omcgo/internal/backup/snapshot_service.go`
- Modify: `omcgo/internal/report/generator.go`
- Modify: `omcgo/internal/trace/exporter.go`
- Modify: `omcgo/internal/provision/model_upload.go`
- Modify: `omcgo/internal/software/transfer_router.go`
- Modify: `omcgo/internal/logretention/cleanup_runner.go`
- Test: 对应目录已有 handler/service/bridge/executor/exporter tests

**Steps:**

- [ ] 在 MinIO `PutObject`、本地文件 `OpenFile/Create`、大文件临时文件创建、PG 持久化任务提交前统一调用 `WriteAdmission.Check`，检查发生在产生不可回收数据之前。
- [ ] P2 首先接入 PM/MR 上报文件、备份/config snapshot、report/exchange/trace、firmware/config/certificate/license 等明确业务写入；为每类写入设置稳定 `write_scope`。
- [ ] 被阻止时返回稳定业务错误/HTTP 语义、Retry-After 和审计事件；不得先写对象再异步删除。
- [ ] 写入失败后的对象与元数据补偿保持幂等；拒写计数带 target/write_scope/reason，不能带设备号或对象路径。
- [ ] PM/日志路径按设计实现降级策略：日志优先降级，错误日志、安全审计和 storage-protection audit 仍必须写入；无法保证时转为明确 error。
- [ ] 对所有调用点增加“normal 放行、warning 放行并记录、blocked 拒绝、unknown 按策略、恢复后放行”测试。

**Verification:**

```bash
cd omcgo && go test ./internal/acs/upload ./internal/transfer ./internal/core/storage ./internal/backup ./internal/report ./internal/trace ./internal/provision ./internal/software ./internal/logretention
```

**Done when:** 通过任意已接入业务路径写入时，策略达到 block 水位可阻止新数据落盘；恢复到 recover 水位并满足确认周期后自动放行。

---

## Phase 3 — OMC 管理页面、RBAC 和审计

### Task 11: 提供存储保护和有效期管理 API

**Files:**

- Create: `omcgo/internal/storageprotection/handler.go`
- Modify: `omcgo/cmd/app/provider/router.go`
- Modify: `omcgo/cmd/app/provider/modules.go`
- Modify: `omcgo/internal/admin/permission_service.go`
- Create: `omcgo/migrations/seed/000003_storage_protection_permissions.sql`
- Create: `omcgo/internal/storageprotection/handler_test.go`
- Create: `omcgo/internal/storageprotection/service_test.go`

**Steps:**

- [ ] 注册设计文档规定的 API：targets、policies GET/POST/PUT、policy release、events、retention policies；统一使用 `/api/v1/admin/...`、现有 response/error/RBAC 约定。
- [ ] targets 返回 capacity、used ratio、state、last check、write scopes；policies 写入前做百分比、间隔、scope、target 和 unknown behavior 校验。
- [ ] release 接口只允许 super-admin 或明确的 storage protection 操作权限，并记录操作者、旧状态、新状态、原因和策略版本；不能绕过 block 直接永久放行。
- [ ] seed 新增菜单、API 权限和默认角色授权；非授权用户必须得到 403，不能仅依赖前端隐藏按钮。
- [ ] retention API 与 storage protection API 分离资源模型，但共用 target/type 枚举和审计模型。

**Verification:**

```bash
cd omcgo && go test ./internal/storageprotection ./internal/admin
go test ./test/integration -run Migration
```

**Done when:** API 可通过鉴权管理策略，权限和审计在后端生效；重复保存、非法阈值、并发更新和版本冲突均有稳定错误。

### Task 12: 实现 V1 存储保护管理页面

**Files:**

- Create: `omcmb/frontend-core/src/services/api/storageProtectionApi.ts`
- Create: `omcmb/frontend-core/src/hooks/api/useStorageProtection.ts`
- Modify: `omcmb/frontend-core/src/types/system.ts`
- Create: `omcmb/webcode/src/pages/system/StorageProtection/index.tsx`
- Create: `omcmb/webcode/src/pages/system/StorageProtection/index.test.tsx`
- Modify: `omcmb/webcode/src/router/routes.tsx`
- Modify: `omcmb/frontend-core/src/i18n/zh-CN/index.ts`
- Modify: `omcmb/frontend-core/src/i18n/en-US/index.ts`
- Modify: `omcmb/webcode/e2e/smoke/system.spec.ts`

**Steps:**

- [ ] 新增“资源监控与存储保护”页面，至少包含容量概览、保护策略、当前阻断、清理/有效期、审计五个区域；展示目标、占用比例、状态、最近检查和不可用原因。
- [ ] 策略表单支持 warn/block/recover、检查周期、unknown behavior、scope 和启停；前端校验不能替代后端校验，保存后刷新服务端真实状态。
- [ ] 当前阻断支持查看原因、Retry-After、受影响 write scope 和受控 release；所有操作显示成功/失败/权限提示。
- [ ] retention 表单支持 data class、target、days、max bytes、max objects、schedule、batch size、delete order、orphan policy 和 enabled。
- [ ] 页面使用现有 `PrivateRoute`、React Query 和 API 结构；所有文案、空态、错误态、确认框走 i18n。
- [ ] 用 mock API 单测覆盖数据映射、非法阈值、空状态、blocked 状态和无权限状态；用浏览器 smoke 覆盖进入、查询、编辑、保存和刷新。

**Verification:**

```bash
cd omcmb && npm run typecheck
npm run test -- --run src/pages/system/StorageProtection/index.test.tsx
npm run test:e2e -- webcode/e2e/smoke/system.spec.ts
```

**Done when:** admin 可在 OMC 页面配置并看到真实策略/状态，普通用户受到后端权限控制；页面不出现硬编码用户可见文案。

---

## Phase 4 — 统一有效期和清理治理

### Task 13: 建立 retention policy 和清理编排

**Files:**

- Modify: `omcgo/migrations/000002_storage_protection.sql`
- Create: `omcgo/internal/storageretention/model.go`
- Create: `omcgo/internal/storageretention/repository.go`
- Create: `omcgo/internal/storageretention/pg_repository.go`
- Create: `omcgo/internal/storageretention/service.go`
- Create: `omcgo/internal/storageretention/cleanup_runner.go`
- Create: `omcgo/internal/storageretention/*_test.go`
- Modify: `omcgo/internal/pm/retention/service.go`
- Modify: `omcgo/internal/pm/retention/cleanup_runner.go`
- Modify: `omcgo/internal/logretention/cleanup_runner.go`
- Modify: `omcgo/internal/alarm/history_retention.go`
- Modify: `omcgo/internal/backup/policy_orphan_reaper.go`
- Modify: `omcgo/cmd/app/provider/pm_retention.go`
- Modify: `omcgo/cmd/app/provider/minio_ilm.go`
- Modify: `omcgo/cmd/worker/retention.go`

**Steps:**

- [ ] 主库策略表统一保存 data class、target、retention days、max bytes、max objects、schedule、batch size、delete order、orphan policy、enabled 和 version。
- [ ] 统一覆盖 MinIO PM/MR/log/backup/config/exchange/trace、TSDB PM/MR/Trace/alarm、应用日志、NATS、Redis TTL、PG task metadata、Prometheus/Loki/Tempo 等数据类别；不能由 OMC 直接控制的内部数据明确标记为 monitor-only。
- [ ] 清理 runner 按小批量、可重入、可暂停、幂等执行；先删除过期/孤儿对象，再按容量上限做最老优先清理，并记录释放 bytes/objects、失败、耗时和最后成功时间。
- [ ] 迁移已有 PM/log/alarm/backup 清理逻辑为 adapter，保留现有业务语义和默认值；禁止新旧 runner 同时删除同一类数据。
- [ ] 清理失败不改变写入准入状态，但触发 cleanup failure 告警；连续失败时在管理页面显示 remediation 建议。

**Verification:**

```bash
cd omcgo && go test ./internal/storageretention ./internal/pm/retention ./internal/logretention ./internal/alarm ./internal/backup
go test ./test/integration -run Migration
```

**Done when:** 各数据类别有唯一有效期来源，清理任务可重试且有释放空间指标；容量达到 block 前能通过清理释放空间并在审计中可追溯。

---

## Phase 5 — 联调、故障注入和交付验收

### Task 14: 完成跨组件端到端验收

**Files:**

- Modify: `deployments/monitoring/tests/promql-probes.txt`
- Modify: `deployments/monitoring/README.md`
- Create: `docs/runbook/resource-queue-storage-protection.md`
- Modify: `docs/design/资源监控与队列治理及磁盘写入保护功能设计-20260728.md`
- Modify: `omcmb/webcode/e2e/smoke/system.spec.ts`

**Steps:**

- [ ] 用运行环境验证 host CPU/memory/disk、inode、container、MinIO、NATS、Redis、PG、PM 和写入保护查询；记录每个 probe 的返回 series、单位和 freshness。
- [ ] 注入 NATS consumer backlog，确认 pending/oldest/alert/Grafana panel 链路；恢复消费后确认告警恢复。
- [ ] 写入 old/new Redis task key，确认 queue family、SCAN 采样、对账和恢复指标；模拟 Redis 不可用，确认不是 0。
- [ ] 插入 PostgreSQL pending/failed/dead-letter 任务，确认统一 queue/status 指标和 panel；清空队列确认输出 0。
- [ ] 模拟 MinIO/应用数据盘 80% warning、90% block、85% recover，验证两次确认、hysteresis、拒写/降级、Retry-After、审计和恢复。
- [ ] 修改 retention policy，创建过期对象/记录，确认清理释放指标、失败重试和页面状态；验证无法控制的监控内部数据未被误报为业务写入已阻止。
- [ ] 浏览器验收必须检查真实 Grafana panel、真实 OMC 页面 DOM、保存后的再次加载结果和权限 403，不以 mock 或源码存在作为验收依据。

**Verification:**

```bash
deployments/monitoring/tests/validate-dashboards.sh
promtool check rules deployments/monitoring/alerts/*.yml
cd omcgo && go build ./... && go test ./...
cd ../omcmb && npm run typecheck
```

**Done when:** 需求 1/2/3 的功能链路均有可复现证据，且文档明确列出已实现、优化、未纳入本期的边界和运行手册。

---

## 依赖关系与建议开发顺序

```text
Task 1/2
   ↓
Task 3/4 ───────────────┐
   ↓                    │
Task 5/6/7 ─────────────┤
   ↓                    ↓
Task 8            Task 9
                       ↓
                 Task 10
                       ↓
                 Task 11 → Task 12
                       ↓
                 Task 13
                       ↓
                 Task 14
```

建议按以下可独立验收的交付批次推进：

1. **P0 监控修复批次：** Task 1–4。先让已有 CPU/内存/磁盘/容器/MinIO 数据在 Grafana 真实可见。
2. **P1 队列观测批次：** Task 5–8。完成 NATS、Redis、PG、PM 队列指标、统一 dashboard 和告警。
3. **P2 写入保护批次：** Task 9–12。完成策略、状态机、准入接入、API、RBAC、OMC 页面。
4. **P3 有效期治理批次：** Task 13–14。统一清理策略，完成故障注入、端到端验收和运行手册。

每个批次完成后应先在测试环境验证，再进入下一批次；P2 在没有 Task 4/8 的真实容量指标和 Grafana 观察能力前不应直接开启生产拒写。

## 交付提交建议

- 每个 Task 或同一批次内的垂直切片使用一个 Conventional Commit，提交信息使用中文并包含能力范围，例如 `feat: 增加 NATS 队列积压监控`。
- Dashboard、Prometheus rules、Go 指标和前端页面必须与对应测试同一提交，避免出现“面板先合入但指标尚未存在”的中间状态。
- 迁移、RBAC seed、后端 API、前端页面和 runbook 必须在发布清单中成组核对；不直接推送或合并主分支。

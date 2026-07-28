# Dashboard 全网预聚合直读与线上保护 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 让首页 summary 和 KPI 时序直接读取现有 eNB、gNB、GSM 全网小时/天预聚合结果，彻底移除首页原始 PM 扫描与即席全网汇总，并用超时、并发限制、慢查询监控和资源告警把线上影响限制在可控范围内。

**Architecture:** 在 Dashboard 内建立只读 `NetworkRollupReader`，以 `pm_aggregation_results` 的 `dimension='network'` 结果作为唯一 PM 数据源；Service 通过带 singleflight、短缓存、stale 缓存和有界并发的 `KPIQueryGuard` 调用 Repository。首页仍只接受 hourly/daily，缺失时返回空点或可识别错误，绝不回退原始表。全网小时/天发布事件精准失效缓存；应用、PostgreSQL、容器指标统一进入现有 Prometheus/Grafana 告警链路。

**Tech Stack:** Go、pgx/pgxpool、Squirrel、x/sync/singleflight、Prometheus client、PostgreSQL/TimescaleDB、OpenTelemetry Collector、Prometheus/Alertmanager、Grafana、React Query、Vitest

## Global Constraints

- 首页只保留 `hourly`、`daily`；不增加 15 分钟或周粒度入口。
- 直接复用 LTE/eNB、NR/gNB、GSM 已有全网上卷任务和 `pm_aggregation_results`；不新增聚合任务、结果表或 Dashboard 内补算。
- `/dashboard/summary` 和 `/dashboard/kpi-time-series` 的 PM 路径不得读取 `pm_metric_values`、`pm_measurement_anchors`、`pm_metric_dictionary`、`pm_metrics_*` 兼容视图，也不得调用 PM Aggregator 的即席 network 聚合。
- 预聚合缺失、不完整、超时或过载时不得回退原始表；只能返回空点、最近成功 stale 缓存或明确的 503/504。
- summary 的设备数、告警数、最近告警等非 PM 子查询保持现状。
- 所有生产代码先由失败测试驱动；每个任务完成后运行该任务的目标测试并提交小步 Conventional Commit。
- 不回滚仓库内无关改动；上线变更先经过覆盖门禁和隔离副本性能门禁，再切生产流量。

---

## File Structure

### Backend read path

- Create: `omcgo/internal/dashboard/network_rollup_repository.go`
  - 定义 `NetworkRollupReader`、查询参数、结果点和 TSDB 只读实现。
  - 只查询 network 维度全网任务，SQL 内完成最新版本去重和数据库 statement timeout。
- Create: `omcgo/internal/dashboard/network_rollup_repository_test.go`
  - 锁定 SQL 物理数据源、过滤条件、版本去重、小时/天粒度和拒绝非法粒度。
- Modify: `omcgo/internal/pm/stream/device_pipeline.go`
  - 导出内置全网任务 ID 查询函数，Dashboard 不复制 UUID 常量。
- Modify: `omcgo/internal/pm/stream/device_pipeline_test.go`
  - 覆盖 LTE/NR/GSM 映射、大小写和未知制式。
- Modify: `omcgo/internal/dashboard/service.go`
  - 用 `NetworkRollupReader` 替换 `dashboardKPIAggregator`。
  - summary 和时序统一消费最终全网结果。
- Modify: `omcgo/internal/dashboard/kpi_series_service_test.go`
  - 用 fake reader 覆盖小时、天、当前期/对比期、缺失桶、不完整窗口和无回退。
- Delete: `omcgo/internal/dashboard/kpi_network_query.go`
- Delete: `omcgo/internal/dashboard/kpi_network_query_test.go`
- Delete: `omcgo/internal/dashboard/kpi_summary_query.go`
- Delete: `omcgo/internal/dashboard/kpi_summary_query_test.go`

### Backend protection and API

- Create: `omcgo/internal/dashboard/kpi_query_guard.go`
  - 实现规范化 key、fresh/stale 缓存、singleflight、有界并发、排队和查询超时。
- Create: `omcgo/internal/dashboard/kpi_query_guard_test.go`
  - 使用可控 clock/blocking reader 测试保护语义。
- Create: `omcgo/internal/dashboard/metrics.go`
  - 注册低基数 Dashboard 查询、缓存、缺失、完整性和 lag 指标。
- Create: `omcgo/internal/dashboard/metrics_test.go`
  - 验证 nil registry、指标注册和标签集合。
- Modify: `omcgo/internal/dashboard/handler.go`
  - 将过载/超时映射为 503/504；stale 设置响应头；过载设置 `Retry-After: 1`。
- Modify: `omcgo/internal/dashboard/handler_test.go`
  - 锁定 HTTP 状态、响应头和原有 JSON 契约。
- Modify: `omcgo/internal/dashboard/sse_notifier.go`
  - 仅在内置全网小时/天结果发布后失效 KPI 缓存并通知页面。
- Create: `omcgo/internal/dashboard/sse_notifier_test.go`
  - 证明 PM 文件解析和设备级上卷不再触发 Dashboard 刷新。

### Configuration and wiring

- Modify: `omcgo/internal/core/appconfig/config.go`
  - 增加 `DashboardConfig` 及其 `Defaults()`，设置 3s/2.5s/4/100ms/30s/60s/5m 默认值。
- Modify: `omcgo/internal/core/appconfig/validate.go`
  - 校验超时、并发和 TTL 边界。
- Modify: `omcgo/internal/core/appconfig/validate_test.go`
- Modify: `omcgo/internal/core/appconfig/config_expand_test.go`
  - 覆盖默认值、环境覆盖和非法值。
- Modify: `omcgo/cmd/app/etc/config.dev.yaml`
- Modify: `omcgo/cmd/app/etc/config.local.yaml`
- Modify: `omcgo/cmd/app/etc/config.test.yaml`
- Modify: `omcgo/cmd/app/etc/config.prod.yaml`
  - 显式记录 Dashboard 保护参数。
- Modify: `omcgo/cmd/app/provider/modules.go`
  - 注入 reader、guard、metrics，并在 service 建好后构造 SSE notifier。

### Database, monitoring and frontend

- Create: `omcgo/migrations/tsdb/000004_dashboard_network_rollup_read_index.up.sql`
- Create: `omcgo/migrations/tsdb/000004_dashboard_network_rollup_read_index.down.sql`
  - 增加 network 结果局部读取索引；启用 `pg_stat_statements` 扩展。
- Modify: `deployments/docker/docker-compose.yml`
  - 保留 TimescaleDB preload，并追加 `pg_stat_statements`；开启 I/O timing、慢 SQL 和临时文件日志默认值。
- Modify: `omcgo/internal/core/components/postgres/postgres.go`
- Modify: `omcgo/internal/core/components/infra.go`
- Modify: `omcgo/internal/core/components/postgres/slow_query_test.go`
  - 将现有 `SlowQueryTracer` 接到 TSDB pool；即使 `log_sql=false` 也保留慢查询 hash、表名和计数。
- Modify: `deployments/monitoring/otelcol/config.yaml`
  - 为 TSDB 增加 `sqlquery/tsdb`，采集 temp bytes/files 和长查询数。
- Create: `deployments/monitoring/alerts/dashboard-kpi-alerts.yml`
  - 增加 Dashboard、全网上卷和 TSDB 临时写入告警。
- Modify: `deployments/monitoring/alerts/host-container-alerts.yml`
  - 补充 postgres-tsdb CPU/内存专用告警。
- Modify: `deployments/monitoring/grafana/dashboards/omc-overview.json`
  - 在当前面板底部 `y=92` 起新增 Dashboard 延迟、并发/拒绝、缓存和完整性四块面板。
- Modify: `deployments/monitoring/grafana/dashboards/omc-infra.json`
  - 在 `y=36` 起新增 TSDB 临时写入、长查询、CPU/内存和磁盘读写四块面板。
- Modify: `omcmb/frontend-core/src/hooks/api/useDashboard.ts`
  - 禁止固定轮询和自动重试重查询，保留窗口聚焦与 SSE 刷新。
- Modify: `omcmb/webcode/src/components/dashboard/useKPIPanelData.ts`
  - 删除遗留的 60 秒 per-KPI 轮询。
- Modify: `omcmb/webcode/src/test/useDashboard.test.ts`
  - 假定时器验证不轮询、不重试、仍批量查询。

### Verification and runbook

- Create: `omcgo/scripts/verify-dashboard-network-rollups.sql`
  - 上线前检查三制式、首页 KPI、小时/天窗口、完整性和延迟。
- Create: `docs/operations/dashboard-network-rollup-rollout.md`
  - 记录发布顺序、观察项、停止条件和回滚方式。
- Create: `docs/superpowers/evidence/2026-07-28-dashboard-network-rollup-validation.md`
  - 保存隔离副本与上线后实际查询计划、延迟和资源证据。

---

### Task 1: 导出并锁定现有三制式全网任务映射

**Files:**
- Modify: `omcgo/internal/pm/stream/device_pipeline.go`
- Modify: `omcgo/internal/pm/stream/device_pipeline_test.go`

**Interfaces:**
- Produces: `func BuiltinNetworkTaskID(technology string) (uuid.UUID, bool)`
- Consumes: 当前私有 `builtinNetworkRuleIDs`

- [ ] **Step 1: 写映射失败测试**

表驱动断言 `lte/LTE`、`nr/NR`、`gsm/GSM` 返回现有三个固定 UUID，未知值返回 `uuid.Nil, false`。同时断言调用方修改返回值不会影响后续读取。

- [ ] **Step 2: 运行测试确认 RED**

Run:

```bash
cd omcgo
go test ./internal/pm/stream -run TestBuiltinNetworkTaskID
```

Expected: FAIL，导出函数尚不存在。

- [ ] **Step 3: 实现唯一映射入口**

在 `device_pipeline.go` 中增加只读查询函数，内部统一 `strings.ToLower(strings.TrimSpace(technology))`；现有 pipeline 也改用该函数，避免出现两套映射逻辑。

- [ ] **Step 4: 运行测试确认 GREEN**

Run:

```bash
cd omcgo
go test ./internal/pm/stream -run 'TestBuiltinNetworkTaskID|Test.*DevicePipeline'
```

Expected: PASS。

- [ ] **Step 5: 提交**

```bash
git add omcgo/internal/pm/stream/device_pipeline.go omcgo/internal/pm/stream/device_pipeline_test.go
git commit -m "refactor(pm): 导出全网聚合任务映射"
```

### Task 2: 用失败测试定义全网结果 Repository

**Files:**
- Create: `omcgo/internal/dashboard/network_rollup_repository.go`
- Create: `omcgo/internal/dashboard/network_rollup_repository_test.go`
- Create: `omcgo/migrations/tsdb/000004_dashboard_network_rollup_read_index.up.sql`
- Create: `omcgo/migrations/tsdb/000004_dashboard_network_rollup_read_index.down.sql`

**Interfaces:**

```go
type NetworkRollupPoint struct {
    Technology  model.Technology
    MetricPath  string
    Granularity metrics.Granularity
    WindowStart time.Time
    WindowEnd   time.Time
    Value       jsonx.Float
    Complete    bool
    MissingSlots int64
    CreatedAt   time.Time
}

type NetworkRollupQuery struct {
    Technology  model.Technology
    Granularity metrics.Granularity
    MetricPaths []string
    StartTime   time.Time
    EndTime     time.Time
}

type NetworkRollupReader interface {
    ListSeries(context.Context, NetworkRollupQuery) ([]NetworkRollupPoint, error)
    ListLatestHourly(context.Context, time.Time, time.Time) ([]NetworkRollupPoint, error)
}
```

- [ ] **Step 1: 写 SQL 形状失败测试**

把 SQL 生成拆成纯函数供测试。`ListSeries` 断言：

```sql
FROM pm_aggregation_results r
WHERE r.dimension = 'network'
  AND r.task_id = ANY($1)
  AND r.technology = $2
  AND r.granularity = $3
  AND r.metric_path = ANY($4)
  AND r.window_start >= $5
  AND r.window_start < $6
```

并断言 `DISTINCT ON (r.technology, r.metric_path, r.window_start)` 与
`ORDER BY r.technology, r.metric_path, r.window_start, r.created_at DESC` 同时存在。SQL 中不得出现 raw 表、兼容视图或主库任务表。

- [ ] **Step 2: 写参数和防御测试**

覆盖：

- LTE、NR、GSM 使用 Task 1 的任务 ID；
- 仅 `hourly`、`daily` 可用；
- 空指标列表、开始时间不早于结束时间、未知制式直接返回参数错误且不触库；
- metric paths 去空、去重、排序，保证稳定 SQL 参数和缓存 key；
- `ListLatestHourly` 固定 network/hourly/KPI 类型，在 24 小时窗口内按 `technology + metric_path` 取 `window_start, created_at` 最新记录。

- [ ] **Step 3: 运行测试确认 RED**

Run:

```bash
cd omcgo
go test ./internal/dashboard -run 'TestNetworkRollupRepository|TestBuildNetworkRollup'
```

Expected: FAIL，reader 与 SQL builder 尚不存在。

- [ ] **Step 4: 实现只读 Repository**

Repository 持有 `*pgxpool.Pool` 和 `statementTimeout time.Duration`。每次读取：

1. `BeginTx` 使用只读事务；
2. `SELECT set_config('statement_timeout', $1, true)`，参数为 `2500ms` 形式；
3. 执行有界 SQL 并扫描结果；
4. 检查 `rows.Err()`；
5. 正常提交，任何错误包装查询上下文。

不得连接主库查任务版本；版本重算只按结果表 `created_at DESC` 去重。

- [ ] **Step 5: 增加局部索引迁移**

Up:

```sql
CREATE INDEX IF NOT EXISTS idx_pm_aggregation_results_dashboard_network
ON pm_aggregation_results
    (task_id, granularity, technology, metric_path, window_start DESC, created_at DESC)
WHERE dimension = 'network';

CREATE EXTENSION IF NOT EXISTS pg_stat_statements;
```

Down 只删除本迁移命名的索引；不删除 `pg_stat_statements`，避免回滚时破坏可能已被其他模块使用的共享扩展。

- [ ] **Step 6: 运行测试确认 GREEN**

Run:

```bash
cd omcgo
go test ./internal/dashboard -run 'TestNetworkRollupRepository|TestBuildNetworkRollup'
```

Expected: PASS。

- [ ] **Step 7: 提交**

```bash
git add omcgo/internal/dashboard/network_rollup_repository.go omcgo/internal/dashboard/network_rollup_repository_test.go omcgo/migrations/tsdb/000004_dashboard_network_rollup_read_index.up.sql omcgo/migrations/tsdb/000004_dashboard_network_rollup_read_index.down.sql
git commit -m "feat(dashboard): 增加全网预聚合只读仓库"
```

### Task 3: 将 KPI 时序彻底切到全网结果

**Files:**
- Modify: `omcgo/internal/dashboard/service.go`
- Modify: `omcgo/internal/dashboard/kpi_series_service_test.go`
- Delete: `omcgo/internal/dashboard/kpi_network_query.go`
- Delete: `omcgo/internal/dashboard/kpi_network_query_test.go`

**Interfaces:**
- `NewService(..., networkRollups NetworkRollupReader, logger *zap.Logger) *Service`
- Existing: `GetKPITimeSeries`, `GetKPITrendComparison`, Active UE delta helper

- [ ] **Step 1: 把测试 fake 改为 NetworkRollupReader**

在 `kpi_series_service_test.go` 用 fake reader 记录每次 `NetworkRollupQuery`，先断言：

- hourly 请求只读 hourly；
- daily 请求只读 daily；
- 当前期和对比期各产生一个批量查询，不按 KPI 拆分；
- LTE/NR/GSM 正确透传；
- 结果按 metric path/window 排序；
- 缺失桶保持缺失，不补 0；
- `Complete=false` 和 `MissingSlots>0` 不丢失内部状态；
- reader 返回空结果或错误时，没有第二次原始/aggregator 调用。

- [ ] **Step 2: 运行目标测试确认 RED**

Run:

```bash
cd omcgo
go test ./internal/dashboard -run 'Test.*KPITimeSeries|Test.*TrendComparison|Test.*ActiveUE'
```

Expected: FAIL，Service 仍依赖 `dashboardKPIAggregator`。

- [ ] **Step 3: 替换 Service 依赖和 helper**

删除 `dashboardKPIAggregator` 字段；`fetchNetworkKCodeSeries` 与 daily 分支直接构造 `NetworkRollupQuery`。公式 KPI 与直接 KPI 一律把 Repository 返回值视为最终值，不调用 `Aggregator.Query`，不再在 Dashboard 内做公式或 sum/avg/min/max。

同步让 `GetKPITrendComparison` 和 Active UE delta 复用新 helper。旧 `/dashboard/kpi-trend` 若未经过这些 helper，保持现状但不得被首页新代码调用。

- [ ] **Step 4: 删除即席聚合适配文件**

删除 `kpi_network_query.go` 和对应测试，并用：

```bash
rg -n 'dashboardKPIAggregator|queryDirectRollupKPIs|queryNetworkTable|aggregator\\.Query' omcgo/internal/dashboard
```

Expected: 新首页时序路径无匹配；允许的非首页旧接口必须在代码注释中明确边界。

- [ ] **Step 5: 运行测试确认 GREEN**

Run:

```bash
cd omcgo
go test ./internal/dashboard -run 'Test.*KPITimeSeries|Test.*TrendComparison|Test.*ActiveUE'
```

Expected: PASS。

- [ ] **Step 6: 提交**

```bash
git add -A omcgo/internal/dashboard
git commit -m "refactor(dashboard): 时序直读全网预聚合"
```

### Task 4: 将 summary KPI overview 切到最近全网小时结果

**Files:**
- Modify: `omcgo/internal/dashboard/service.go`
- Modify: `omcgo/internal/dashboard/service_test.go`
- Delete: `omcgo/internal/dashboard/kpi_summary_query.go`
- Delete: `omcgo/internal/dashboard/kpi_summary_query_test.go`

**Interfaces:**
- Consumes: `NetworkRollupReader.ListLatestHourly(ctx, now-24h, now)`
- Preserves: `DashboardSummary` JSON contract

- [ ] **Step 1: 写 summary 失败测试**

fake reader 返回跨 LTE/NR/GSM、跨窗口和重复 metric path 的结果，断言：

- 每个 `technology + metric_path` 只使用最新窗口；
- 指标编号在不同制式唯一时仍写入现有 map；
- 同一指标同窗口的最新 `created_at` 已由 Repository 解决，Service 不自行覆盖；
- 空结果不伪造 0；
- 非 PM 的设备、告警、最近告警协程仍被调用；
- reader 报错只影响 KPI overview 的既有容错边界，不触发 raw SQL。

增加冲突防御：同一 metric path 被多个制式返回时记录冲突并跳过该 path，禁止按遍历顺序隐式覆盖。

- [ ] **Step 2: 运行测试确认 RED**

Run:

```bash
cd omcgo
go test ./internal/dashboard -run 'Test.*Summary'
```

Expected: FAIL，summary 仍构造 raw KPI SQL。

- [ ] **Step 3: 替换 summary 查询**

把原始 KPI goroutine 改为 `ListLatestHourly`。删除 raw query builder 文件，保留 summary 其他并发子查询。错误必须包装 `"load dashboard latest network rollups"` 上下文。

- [ ] **Step 4: 执行物理数据源静态门禁**

Run:

```bash
rg -n 'pm_metric_values|pm_measurement_anchors|pm_metric_dictionary|pm_metrics_(hourly|daily|weekly|monthly)' omcgo/internal/dashboard
```

Expected: summary 与 KPI 时序生产代码无匹配；若测试 fixture 出现，只允许用于负向断言。

- [ ] **Step 5: 运行测试确认 GREEN**

Run:

```bash
cd omcgo
go test ./internal/dashboard -run 'Test.*Summary'
```

Expected: PASS。

- [ ] **Step 6: 提交**

```bash
git add -A omcgo/internal/dashboard
git commit -m "refactor(dashboard): summary 直读全网小时结果"
```

### Task 5: 增加可配置的超时、并发、singleflight 和缓存保护

**Files:**
- Create: `omcgo/internal/dashboard/kpi_query_guard.go`
- Create: `omcgo/internal/dashboard/kpi_query_guard_test.go`
- Modify: `omcgo/internal/dashboard/service.go`

**Interfaces:**

```go
type KPIQueryGuardConfig struct {
    QueryTimeout  time.Duration
    MaxConcurrent int
    QueueTimeout  time.Duration
    SummaryTTL    time.Duration
    SeriesTTL     time.Duration
    StaleTTL      time.Duration
}

type KPIQueryMetadata struct {
    Stale bool
}
```

- [ ] **Step 1: 写并发与超时失败测试**

使用容量为 1 的 guard 和 blocking loader：

- 第一个查询占用名额；
- 同 key 第二个调用被 singleflight 合并，不再占名额；
- 不同 key 第二个调用等待 100ms 后返回包装 `commonerrors.ErrUnavailable`；
- loader 超过 3s 返回包装 `commonerrors.ErrTimeout`；
- 查询 context 被客户端取消后，已合并查询仍最多执行到 guard 超时，不产生无界后台 SQL；
- inflight 在成功、失败、panic 防御路径都回到 0。

- [ ] **Step 2: 写缓存失败测试**

使用 fake clock 验证：

- summary fresh 30s、series fresh 60s；
- key 包含 endpoint、technology、granularity、排序去重后的指标、UTC start/end；
- fresh 命中不触发 loader；
- loader 失败或过载时只返回年龄不超过 5m 的最近成功值并标记 `Stale=true`；
- 超过 stale TTL 后返回原错误；
- `Invalidate()` 清 fresh 和 stale，下一次必须重新读取；
- 失败结果不写入成功缓存。

- [ ] **Step 3: 运行测试确认 RED**

Run:

```bash
cd omcgo
go test ./internal/dashboard -run TestKPIQueryGuard
```

Expected: FAIL，guard 尚不存在。

- [ ] **Step 4: 实现 guard**

实现顺序固定为：

1. 规范化 key；
2. 查 fresh cache；
3. 进入 `singleflight.Group.Do`；
4. 再查一次 fresh cache；
5. 在 `QueueTimeout` 内获取 buffered channel 名额；
6. 用 `context.WithTimeout(context.WithoutCancel(ctx), QueryTimeout)` 执行 reader；
7. 成功写 fresh/stale 共用的最近成功项；
8. 失败时尝试 stale；
9. defer 释放名额并更新 inflight。

缓存使用互斥锁保护，存储不可变副本；返回切片前复制，防止调用方修改缓存。`Invalidate` 与查询完成竞争时使用 generation，禁止失效前启动的旧查询在失效后重新写回。

- [ ] **Step 5: 接入 summary 和时序**

Service 只通过 guard 调 reader。summary key 固定 endpoint `summary`；时序 key 固定 `series`。对比期仍是独立 key，重复客户端会被合并。

- [ ] **Step 6: 运行测试确认 GREEN 和 race 安全**

Run:

```bash
cd omcgo
go test -race ./internal/dashboard -run 'TestKPIQueryGuard|Test.*KPITimeSeries|Test.*Summary'
```

Expected: PASS，无 race。

- [ ] **Step 7: 提交**

```bash
git add omcgo/internal/dashboard/kpi_query_guard.go omcgo/internal/dashboard/kpi_query_guard_test.go omcgo/internal/dashboard/service.go
git commit -m "feat(dashboard): 增加查询超时并发与缓存保护"
```

### Task 6: 增加保护配置并完成依赖注入

**Files:**
- Modify: `omcgo/internal/core/appconfig/config.go`
- Modify: `omcgo/internal/core/appconfig/validate.go`
- Modify: `omcgo/internal/core/appconfig/validate_test.go`
- Modify: `omcgo/internal/core/appconfig/config_expand_test.go`
- Modify: `omcgo/cmd/app/etc/config.dev.yaml`
- Modify: `omcgo/cmd/app/etc/config.local.yaml`
- Modify: `omcgo/cmd/app/etc/config.test.yaml`
- Modify: `omcgo/cmd/app/etc/config.prod.yaml`
- Modify: `omcgo/cmd/app/provider/modules.go`

**Interfaces:**

```go
type DashboardConfig struct {
    QueryTimeout    time.Duration `mapstructure:"query_timeout"`
    StatementTimeout time.Duration `mapstructure:"statement_timeout"`
    MaxConcurrent   int           `mapstructure:"max_concurrent"`
    QueueTimeout    time.Duration `mapstructure:"queue_timeout"`
    SummaryCacheTTL time.Duration `mapstructure:"summary_cache_ttl"`
    SeriesCacheTTL  time.Duration `mapstructure:"series_cache_ttl"`
    StaleTTL        time.Duration `mapstructure:"stale_ttl"`
}
```

- [ ] **Step 1: 写默认值和校验失败测试**

期望默认值：

```yaml
dashboard:
  query_timeout: 3s
  statement_timeout: 2500ms
  max_concurrent: 4
  queue_timeout: 100ms
  summary_cache_ttl: 30s
  series_cache_ttl: 60s
  stale_ttl: 5m
```

校验规则：所有 duration > 0；statement timeout < query timeout；max concurrent 在 1..64；stale TTL 不短于两个 fresh TTL。测试环境变量 `OMCGO_DASHBOARD_MAX_CONCURRENT=2` 能覆盖 YAML。

- [ ] **Step 2: 运行测试确认 RED**

Run:

```bash
cd omcgo
go test ./internal/core/appconfig -run 'Test.*Dashboard'
```

Expected: FAIL，配置尚不存在。

- [ ] **Step 3: 实现配置、默认值与四套 YAML**

在 `config.go` 实现 `(DashboardConfig{}).Defaults()`，provider 只消费 defaults 后的值，不重复常量。四套 YAML 都显式写出同一安全默认；测试可按需进一步收紧 timeout，但不得关闭保护。

- [ ] **Step 4: 重构 provider 注入**

`initDashboardModule` 先执行 `dashboardCfg := c.Cfg.Dashboard.Defaults()`，再按以下顺序：

1. `dashboard.NewNetworkRollupRepository(c.TsPool, c.Cfg.Dashboard.StatementTimeout)`；
2. `dashboard.NewMetrics(c.MetricsReg)`；
3. `dashboard.NewKPIQueryGuard(...)`；
4. `dashboard.NewService(...)`；
5. `dashboard.NewSSENotifier(..., dashboardService)` 并订阅；
6. 创建 handler。

移除 `dashPMAggregator` 注入，indicator repository 仍保留给 KPI 定义接口。

- [ ] **Step 5: 运行测试确认 GREEN 和编译**

Run:

```bash
cd omcgo
go test ./internal/core/appconfig ./internal/dashboard ./cmd/app/provider
go build ./cmd/app
```

Expected: PASS。

- [ ] **Step 6: 提交**

```bash
git add omcgo/internal/core/appconfig omcgo/cmd/app/etc/config.*.yaml omcgo/cmd/app/provider/modules.go
git commit -m "feat(dashboard): 配置查询保护并完成注入"
```

### Task 7: 增加应用指标和 HTTP 过载/stale 语义

**Files:**
- Create: `omcgo/internal/dashboard/metrics.go`
- Create: `omcgo/internal/dashboard/metrics_test.go`
- Modify: `omcgo/internal/dashboard/kpi_query_guard.go`
- Modify: `omcgo/internal/dashboard/service.go`
- Modify: `omcgo/internal/dashboard/handler.go`
- Modify: `omcgo/internal/dashboard/handler_test.go`

**Interfaces:**

```text
dashboard_kpi_query_duration_seconds{endpoint,granularity,status}
dashboard_kpi_query_inflight
dashboard_kpi_query_timeout_total{endpoint}
dashboard_kpi_query_rejected_total{endpoint}
dashboard_kpi_query_cache_total{endpoint,result}
dashboard_kpi_query_coalesced_total{endpoint}
dashboard_kpi_missing_result_total{technology,granularity}
dashboard_kpi_incomplete_window_total{technology,granularity}
pm_network_rollup_lag_seconds{technology,granularity}
```

- [ ] **Step 1: 写 metrics 失败测试**

使用新 registry 收集指标并断言：

- nil registry 全部方法安全 no-op；
- label 只允许 endpoint/granularity/status/result/technology；
- 不出现 KPI、request ID、SQL、时间范围标签；
- timeout、reject、fresh/stale/miss、coalesced、inflight、duration 都可观测；
- 每次读取更新 missing/incomplete counter 和最新窗口 lag gauge。

- [ ] **Step 2: 写 handler 失败测试**

锁定：

- overload → 503，`Retry-After: 1`；
- query timeout → 504；
- stale success → 200，`X-OMC-Data-Stale: true`；
- fresh success 不设置 stale 头；
- 现有 JSON 字段、空数组和 granularity 校验不变；
- weekly/15min 仍为 400。

- [ ] **Step 3: 运行测试确认 RED**

Run:

```bash
cd omcgo
go test ./internal/dashboard -run 'TestDashboardMetrics|Test.*Handler.*(Stale|Timeout|Unavailable|Granularity)'
```

Expected: FAIL。

- [ ] **Step 4: 实现 metrics 与错误映射**

复用 `commonerrors.HTTPStatusFromError`。只对并发获取超时设置 `Retry-After`；数据库 statement/app timeout 返回 504。让时序 Service 返回 `KPIQueryMetadata`，handler 根据 metadata 设置 stale 头。summary 内部可用 stale，但若 summary 仍是混合响应，至少记录 stale metric 和结构化日志，不改变 summary JSON。

- [ ] **Step 5: 运行测试确认 GREEN**

Run:

```bash
cd omcgo
go test ./internal/dashboard
```

Expected: PASS。

- [ ] **Step 6: 提交**

```bash
git add omcgo/internal/dashboard
git commit -m "feat(dashboard): 暴露查询保护与完整性指标"
```

### Task 8: 用全网窗口发布事件精准刷新，移除 PM 文件刷新风暴

**Files:**
- Modify: `omcgo/internal/dashboard/sse_notifier.go`
- Create: `omcgo/internal/dashboard/sse_notifier_test.go`

**Interfaces:**

```go
type KPIQueryInvalidator interface {
    InvalidateKPIQueries()
}
```

- [ ] **Step 1: 写事件过滤失败测试**

覆盖：

- `SubjectPMFileParsed` 不再订阅、不发布 `dashboard_update`；
- 告警 raised/cleared 仍发布更新，但不清 PM KPI 缓存；
- 内置 LTE/NR/GSM task 的 hourly/daily rollup 解码后失效缓存并发布一次更新；
- 设备级 task 或随机 task UUID 的 rollup 不失效、不发布；
- 非法 payload 记录 warning，不 panic、不发布。

测试 payload 使用 `pmstream.RollupPayload` 的紧凑 JSON 字段，尤其是 `rid`。

- [ ] **Step 2: 运行测试确认 RED**

Run:

```bash
cd omcgo
go test ./internal/dashboard -run TestSSENotifier
```

Expected: FAIL，当前仍订阅 `PMFileParsed`。

- [ ] **Step 3: 修改订阅和 invalidator**

订阅：

- `SubjectAlarmRaised`
- `SubjectAlarmCleared`
- `SubjectPMAggregationHourlyRollup`
- `SubjectPMAggregationDailyRollup`

rollup handler 只比较 Task 1 导出的三个 task ID。全网结果发布后调用 `InvalidateKPIQueries()`，再通过现有 debounce/SSE 通知前端。

- [ ] **Step 4: 运行测试确认 GREEN**

Run:

```bash
cd omcgo
go test ./internal/dashboard -run TestSSENotifier
```

Expected: PASS。

- [ ] **Step 5: 提交**

```bash
git add omcgo/internal/dashboard/sse_notifier.go omcgo/internal/dashboard/sse_notifier_test.go
git commit -m "fix(dashboard): 按全网窗口发布精准刷新"
```

### Task 9: 清除前端固定轮询和重试风暴

**Files:**
- Modify: `omcmb/frontend-core/src/hooks/api/useDashboard.ts`
- Modify: `omcmb/webcode/src/components/dashboard/useKPIPanelData.ts`
- Modify: `omcmb/webcode/src/test/useDashboard.test.ts`

**Interfaces:**
- Existing: `buildDashboardKPIQueryOptions`
- Existing: `useMultiKPITrendComparison`
- Existing: `useDashboardRealtime`

- [ ] **Step 1: 写前端失败测试**

在 `useDashboard.test.ts` 用假定时器和 API mock 断言：

- 首次渲染当前期和对比期最多各一个批量请求；
- 推进 60s、120s 不新增请求；
- 503/504/429 不自动 retry；
- window focus 允许刷新；
- `dashboard_update` debounce 后失效并刷新；
- weekly UI 模式仍转换成 daily 请求，不向 API 发送 weekly；
- 请求不产生 15min。

- [ ] **Step 2: 运行测试确认 RED**

Run:

```bash
cd omcmb
npm run test --workspace webcode -- src/test/useDashboard.test.ts
```

Expected: 至少固定轮询或 retry 断言 FAIL。

- [ ] **Step 3: 修改 React Query 选项**

`buildDashboardKPIQueryOptions` 显式设置：

```ts
retry: false,
refetchInterval: false,
refetchIntervalInBackground: false,
refetchOnWindowFocus: true,
```

保留 `staleTime: 30_000` 和当前批量请求结构。删除 `useKPIPanelData.ts` 内 `refetchInterval: 60_000`；若该遗留 hook 已无引用，单独确认后删除 hook，而不是保留危险默认。

- [ ] **Step 4: 运行测试和类型检查**

Run:

```bash
cd omcmb
npm run test --workspace webcode -- src/test/useDashboard.test.ts
npm run typecheck
```

Expected: PASS。

- [ ] **Step 5: 提交**

```bash
git add omcmb/frontend-core/src/hooks/api/useDashboard.ts omcmb/webcode/src/components/dashboard/useKPIPanelData.ts omcmb/webcode/src/test/useDashboard.test.ts
git commit -m "fix(dashboard): 移除 KPI 固定轮询与自动重试"
```

### Task 10: 补齐 TSDB 慢查询、临时写入和资源采集

**Files:**
- Modify: `deployments/docker/docker-compose.yml`
- Modify: `deployments/monitoring/otelcol/config.yaml`
- Modify: `deployments/release/bundle/deploy/storage-compose_test.sh`
- Modify: `omcgo/internal/core/components/postgres/postgres.go`
- Modify: `omcgo/internal/core/components/infra.go`
- Modify: `omcgo/internal/core/components/postgres/slow_query_test.go`

**Interfaces:**
- Produces metrics:
  - `pg_stat_database_temp_bytes{instance="postgres-tsdb",database="omcgo"}`
  - `pg_stat_database_temp_files{instance="postgres-tsdb",database="omcgo"}`
  - `pg_active_long_queries{instance="postgres-tsdb"}`

- [ ] **Step 1: 写部署配置失败测试**

扩展 `storage-compose_test.sh`，断言：

- postgres-tsdb command 包含 `track_io_timing=${TSDB_TRACK_IO_TIMING:-on}`；
- `log_min_duration_statement=${TSDB_LOG_MIN_DURATION_STATEMENT:-1000}`；
- `log_temp_files=${TSDB_LOG_TEMP_FILES:-0}`；
- preload 同时包含 `timescaledb,pg_stat_statements`，不覆盖 TimescaleDB；
- OTel `metrics/tsdb` pipeline 同时包含 `postgresql/tsdb` 与 `sqlquery/tsdb`。

- [ ] **Step 2: 运行测试确认 RED**

Run:

```bash
bash deployments/release/bundle/deploy/storage-compose_test.sh
```

Expected: FAIL，保护参数和 sqlquery receiver 尚未配置。

- [ ] **Step 3: 写并接通应用侧慢查询 tracer**

先扩展 `slow_query_test.go`，证明 `log_sql=false`、`log_sql_slow_threshold=1000` 时，超过阈值的 TSDB 查询仍由现有 `SlowQueryTracer` 记录 `query_hash/table/request_id` 并增加 `pgx_slow_query_total`；普通查询不记录。

在 `postgres.go` 增加保留旧签名的带 registry 构造入口：

```go
func NewPostgresPoolWithRegisterer(
    ctx context.Context,
    cfg appconfig.PostgresConfig,
    log *zap.Logger,
    reg prometheus.Registerer,
) (*pgxpool.Pool, error)
```

当 `LogSQL=true` 时保留现有 `OTELSQLTracer`；当 `LogSQLSlowThreshold>0` 时无论 `LogSQL` 是否开启，都用 `ChainTracer` 追加 `WithSlowQueryTracer`。`infra.go` 的 main/tsdb 连接分别传入带 `pool=main/tsdb` 常量标签的 registry，旧 `NewPostgresPool` 作为 nil registry 兼容包装。

Run:

```bash
cd omcgo
go test ./internal/core/components/postgres ./internal/core/components -run 'Test.*SlowQuery|Test.*Postgres'
```

Expected: PASS；生产配置关闭全量 SQL 日志时，慢 SQL 仍可见。

- [ ] **Step 4: 修改 TSDB PostgreSQL 参数**

在 postgres-tsdb command 增加：

```text
-c shared_preload_libraries=timescaledb,pg_stat_statements
-c track_io_timing=${TSDB_TRACK_IO_TIMING:-on}
```

并把 TSDB 专用慢 SQL、temp file 默认值分别改为 `1000` 和 `0`，将 `config.prod.yaml` 的 `tsdb.log_sql_slow_threshold` 从 2000 调为 1000。保持现有日志轮转限制，避免慢查询日志填满磁盘。

- [ ] **Step 5: 增加 sqlquery/tsdb receiver**

使用现有 `otel/opentelemetry-collector-contrib:0.103.0` 的 sqlquery receiver：

```yaml
sqlquery/tsdb:
  driver: postgres
  datasource: "host=postgres-tsdb port=5432 user=omcgo password=${env:POSTGRES_TSDB_PASSWORD} dbname=omcgo sslmode=disable"
  collection_interval: 30s
```

查询 `pg_stat_database` 的累计 `temp_bytes/temp_files`，以及 `pg_stat_activity` 中持续超过 5 秒的 active query 数。累计指标声明为 monotonic cumulative sum，长查询声明为 gauge；只保留 database 低基数属性。将 receiver 加入 `metrics/tsdb`，沿用该 pipeline 的 instance processor 和 remote write exporter。

- [ ] **Step 6: 校验 collector 配置和部署测试**

Run:

```bash
bash deployments/release/bundle/deploy/storage-compose_test.sh
docker run --rm -v "$PWD/deployments/monitoring/otelcol/config.yaml:/etc/otelcol-contrib/config.yaml:ro" otel/opentelemetry-collector-contrib:0.103.0 validate --config=/etc/otelcol-contrib/config.yaml
```

Expected: PASS。Docker 命令需要按仓库权限规则提权；如果当前镜像命令不支持 `validate`，使用 `--config` 启动并检查 10 秒内无配置错误后主动停止。

- [ ] **Step 7: 提交**

```bash
git add deployments/docker/docker-compose.yml deployments/monitoring/otelcol/config.yaml deployments/release/bundle/deploy/storage-compose_test.sh omcgo/internal/core/components/postgres/postgres.go omcgo/internal/core/components/infra.go omcgo/internal/core/components/postgres/slow_query_test.go omcgo/cmd/app/etc/config.prod.yaml
git commit -m "ops(tsdb): 开启慢查询临时写入监控"
```

### Task 11: 增加 Dashboard、上卷延迟和 TSDB 资源告警

**Files:**
- Create: `deployments/monitoring/alerts/dashboard-kpi-alerts.yml`
- Modify: `deployments/monitoring/alerts/host-container-alerts.yml`
- Modify: `deployments/monitoring/grafana/dashboards/omc-overview.json`
- Modify: `deployments/monitoring/grafana/dashboards/omc-infra.json`
- Modify: `deployments/monitoring/README.md`

- [ ] **Step 1: 先写精确告警规则**

`dashboard-kpi-alerts.yml` 包含：

```text
DashboardKPIQuerySlow:
  histogram_quantile(0.95, sum by (le) (rate(dashboard_kpi_query_duration_seconds_bucket[5m]))) > 2
  for: 10m, severity: warning

DashboardKPIQueryTimeout:
  increase(dashboard_kpi_query_timeout_total[5m]) > 0
  for: 2m, severity: critical

DashboardKPIQueryRejected:
  increase(dashboard_kpi_query_rejected_total[5m]) > 0
  for: 2m, severity: warning

DashboardKPIResultMissing:
  increase(dashboard_kpi_missing_result_total[10m]) > 0
  for: 10m, severity: warning

PMNetworkRollupLagHigh:
  pm_network_rollup_lag_seconds{granularity="hourly"} > 5400
  for: 10m, severity: warning

PMNetworkRollupLagCritical:
  pm_network_rollup_lag_seconds{granularity="hourly"} > 7200
  for: 5m, severity: critical

TSDBTempWriteHigh:
  rate(pg_stat_database_temp_bytes{instance="postgres-tsdb"}[5m]) > 10 * 1024 * 1024
  for: 5m, severity: warning

TSDBTempWriteCritical:
  rate(pg_stat_database_temp_bytes{instance="postgres-tsdb"}[5m]) > 50 * 1024 * 1024
  for: 5m, severity: critical

TSDBLongQueryActive:
  pg_active_long_queries{instance="postgres-tsdb"} > 0
  for: 5m, severity: warning
```

在 `host-container-alerts.yml` 增加 postgres-tsdb CPU 配额 70% 持续 10m warning、90% 持续 5m critical，以及内存限额 85% 持续 10m warning。保留现有 `HostDiskIOSaturated`，不增加硬件相关的绝对磁盘吞吐告警。

- [ ] **Step 2: 增加 Grafana 面板**

当前 JSON 面板没有显式 `id`，保持现有格式，不虚构 panel id。

`omc-overview.json` 在现有最后一行之后从 `y=92` 排列四个 12×8 面板：

1. Dashboard KPI query p50/p95/p99；
2. inflight、timeout、rejected；
3. fresh/stale/miss/coalesced rate；
4. missing/incomplete 和 rollup lag。

`omc-infra.json` 从 `y=36` 排列四个 12×8 面板：

1. TSDB temp bytes/s 与 files/s；
2. active long queries；
3. postgres-tsdb CPU/内存饱和度；
4. postgres-tsdb 容器 block read/write bytes/s。

- [ ] **Step 3: 校验 YAML、PromQL 和 JSON**

Run:

```bash
promtool check rules deployments/monitoring/alerts/*.yml
jq empty deployments/monitoring/grafana/dashboards/omc-overview.json deployments/monitoring/grafana/dashboards/omc-infra.json
```

Expected: PASS。若本机无 promtool，用仓库 Prometheus 镜像执行同一 `check rules`，不得跳过。

- [ ] **Step 4: 更新 monitoring README**

记录新指标来源、默认阈值、Dashboard/TSDB 面板位置，以及通过环境变量调低日志强度的方法。明确 temp bytes 是自 PostgreSQL 启动/统计重置以来累计值，告警使用 rate。

- [ ] **Step 5: 提交**

```bash
git add deployments/monitoring
git commit -m "feat(monitoring): 增加 Dashboard 与 TSDB 资源告警"
```

### Task 12: 建立上线覆盖门禁和运维回滚手册

**Files:**
- Create: `omcgo/scripts/verify-dashboard-network-rollups.sql`
- Create: `docs/operations/dashboard-network-rollup-rollout.md`

- [ ] **Step 1: 编写只读覆盖核验 SQL**

脚本必须输出：

- 三个内置 task ID 在 `pm_aggregation_results` 的 technology/granularity 行数；
- 每种制式 hourly/daily 最新 `window_start` 与 lag；
- 首页布局所需 metric path 在最近 24 小时 hourly 和最近 7 天 daily 的缺失清单；
- `complete=false` 或 `missing_slots>0` 清单；
- 重复逻辑窗口按 `technology, granularity, metric_path, window_start` 分组的版本数；
- 推荐查询的 `EXPLAIN (ANALYZE, BUFFERS, FORMAT TEXT)`。

脚本只执行 `SELECT/EXPLAIN`，不得更新、补数、建索引或重算。

- [ ] **Step 2: 写发布与回滚手册**

发布顺序固定为：

1. 在隔离生产规模副本应用迁移；
2. 执行覆盖 SQL；
3. 若 metric 缺失，只修已有内置全网任务的 metric selection/mapping，等待上卷补齐后重新核验；
4. 重启 postgres-tsdb 使 preload/track_io_timing 生效；
5. 启动 OTel/Prometheus/Grafana；
6. 部署 app 后先单实例观察 15 分钟；
7. 再恢复全部实例并观察 60 分钟。

停止条件：

- 任何首页 KPI 发生口径冲突；
- 覆盖清单非空；
- 查询计划访问 raw/兼容视图；
- 单查询产生 temp file；
- API P95 ≥ 1s 或 DB P95 ≥ 200ms；
- TSDB CPU ≥ 70% 持续 10m。

回滚只回滚 app 到旧版本并保留新增索引、扩展和监控；由于旧版本会恢复重查询，回滚期间必须暂时限制首页访问并保留 TSDB 告警。禁止通过删除预聚合结果或禁用现有全网任务回滚。

- [ ] **Step 3: 复核脚本无写操作**

Run:

```bash
rg -ni '\\b(insert|update|delete|alter|drop|truncate|create)\\b' omcgo/scripts/verify-dashboard-network-rollups.sql
```

Expected: 无匹配。

- [ ] **Step 4: 提交**

```bash
git add omcgo/scripts/verify-dashboard-network-rollups.sql docs/operations/dashboard-network-rollup-rollout.md
git commit -m "docs(dashboard): 增加全网结果上线门禁"
```

### Task 13: 全量自动化验证

**Files:**
- Test all modified backend, frontend, migration and monitoring files

- [ ] **Step 1: 后端目标测试**

Run:

```bash
cd omcgo
go test ./internal/pm/stream ./internal/dashboard ./internal/core/appconfig ./cmd/app/provider
```

Expected: PASS。

- [ ] **Step 2: 后端编译和全量测试**

Run:

```bash
cd omcgo
go build ./...
go test ./...
```

Expected: PASS。涉及 miniredis/httptest 本地监听时按仓库规则直接提权复跑，区分权限问题和真实失败。

- [ ] **Step 3: race 验证**

Run:

```bash
cd omcgo
go test -race ./internal/dashboard ./internal/pm/stream
```

Expected: PASS。

- [ ] **Step 4: 前端验证**

Run:

```bash
cd omcmb
npm run test --workspace webcode -- src/test/useDashboard.test.ts
npm run typecheck
npm run lint
```

Expected: PASS。

- [ ] **Step 5: 部署资产验证**

Run:

```bash
bash deployments/release/bundle/deploy/storage-compose_test.sh
promtool check rules deployments/monitoring/alerts/*.yml
jq empty deployments/monitoring/grafana/dashboards/omc-overview.json deployments/monitoring/grafana/dashboards/omc-infra.json
docker compose -f deployments/docker/docker-compose.yml config --quiet
```

Expected: PASS。

- [ ] **Step 6: 静态根治门禁**

Run:

```bash
rg -n 'pm_metric_values|pm_measurement_anchors|pm_metric_dictionary|pm_metrics_(hourly|daily|weekly|monthly)|dashboardKPIAggregator' omcgo/internal/dashboard
rg -n 'refetchInterval:\\s*60_?000|retry:\\s*[1-9]' omcmb/frontend-core/src/hooks/api/useDashboard.ts omcmb/webcode/src/components/dashboard
```

Expected: 第一条只允许负向测试断言；第二条无匹配。

### Task 14: 隔离生产规模副本性能验收与上线观察

**Files:**
- Create: `docs/superpowers/evidence/2026-07-28-dashboard-network-rollup-validation.md`

- [ ] **Step 1: 执行切换前覆盖门禁**

在隔离副本执行：

```bash
psql "$TSDB_DSN" -v ON_ERROR_STOP=1 -f omcgo/scripts/verify-dashboard-network-rollups.sql
```

Expected:

- LTE/NR/GSM 的首页 KPI hourly/daily 缺失清单为空；
- 最新 hourly lag < 90m；
- 不完整窗口清单符合业务已知状态，否则停止；
- EXPLAIN 只访问 `pm_aggregation_results` 与 `idx_pm_aggregation_results_dashboard_network`；
- `Buffers: temp` 为 0。

- [ ] **Step 2: 生产规模性能门禁**

隔离副本原始规模至少等价于 10,000 设备、3 亿条 value。对 16 KPI 执行：

- 今天/昨天 hourly 对比；
- daily 时间范围；
- 10 个客户端并发；
- 冷缓存一次、暖缓存至少 100 次。

采集 API histogram、Repository duration、`pg_stat_statements`、temp bytes/files、容器 CPU/内存/block I/O。验收：

- 暖态 DB P95 < 200ms；
- API P95 < 1s；
- 10 客户端并发无 raw 表访问；
- temp bytes/files 增量为 0；
- 没有持续 >5s active query；
- 查询扫描行数随 KPI×桶数近似线性，不随设备数增长。

- [ ] **Step 3: 记录实际证据**

在 evidence 文档写入：

- 测试环境规模和数据时间窗；
- migration 版本；
- 两类 EXPLAIN 完整输出；
- P50/P95/P99；
- 并发时 timeout/rejected/coalesced/cache；
- temp bytes/files 前后值；
- CPU、内存、block I/O 峰值；
- 覆盖、不完整和 lag 结果；
- 明确 PASS/FAIL 结论。

不得保留未填写项或空表格。

- [ ] **Step 4: 分阶段上线并观察**

按 runbook 单实例发布。观察 15 分钟后满足：

- Dashboard P95 < 1s；
- timeout/rejected 无增长；
- stale 只在预期切换窗口出现；
- TSDB temp write rate 接近 0；
- CPU 未持续超过 70%；
- rollup lag < 90m；

再恢复全部实例，继续观察 60 分钟。

- [ ] **Step 5: 最终回归与提交证据**

```bash
git add docs/superpowers/evidence/2026-07-28-dashboard-network-rollup-validation.md
git commit -m "test(dashboard): 记录全网预聚合性能验收"
```

---

## Plan Self-Review Checklist

- [ ] Design coverage: summary、KPI 时序、三制式、小时/天、禁止 15 分钟/周、无 raw fallback、超时、并发、慢查询、资源告警全部有实施任务和验证。
- [ ] Data ownership: Dashboard 只读现有 network 结果；任务 UUID 由 PM stream 单一入口提供；未新增聚合体系。
- [ ] Type consistency: technology 使用 `model.Technology`，granularity 使用 `metrics.Granularity`，task/version 使用 `uuid.UUID`，时间统一 UTC。
- [ ] Failure semantics: 503/504/stale/empty/missing/incomplete 的 API、日志和指标语义均被测试。
- [ ] Concurrency safety: singleflight、cache、generation invalidation、semaphore 均有 race 测试。
- [ ] Rollout safety: 覆盖门禁、隔离副本、单实例灰度、停止条件和可操作回滚均已定义。
- [ ] Placeholder scan:

```bash
rg -n 'T[B]D|T[O]DO|F[I]XME|待[补]|占[位]|<[^>]+>' docs/superpowers/plans/2026-07-28-dashboard-network-rollup-read-hardening.md
```

Expected: 无输出。

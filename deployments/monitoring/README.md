# OMC Monitoring Stack

Prometheus + AlertManager + Grafana + Loki + **otelcol + Tempo** 一键起栈，
覆盖 OMC 三进程（app / acs / worker）的**指标 + 日志 + 链路追踪**三通道可观测性。

Promtail 已于 T-0155 Phase 3 下线，日志采集 → Loki 改由 otelcol filelog
receiver + loki exporter 链路承担。

## 文件结构

```
deployments/monitoring/
├── prometheus.yml                # Prometheus 主配置（scrape + 告警路由）
├── storage-targets.yml           # 统一物理存储目标与逻辑分类（部署契约）
├── alertmanager.yml              # AlertManager 路由 + receiver（占位 webhook）
├── alerts/
│   ├── omc-rules.yml             # starter 告警规则（三进程存活）
│   ├── connection-pool-alerts.yml
│   ├── infra-alerts.yml          # pg/redis/nats/minio 基础服务（T-0155 P2b 改写）
│   ├── dashboard-kpi-alerts.yml  # Dashboard 查询、全网上卷与 TSDB 临时写入
│   ├── otelcol-alerts.yml        # otelcol 自身管道健康（T-0155 收尾）
│   └── storage-queue-alerts.yml  # 存储与 Redis/NATS/PG 队列治理
├── loki/
│   ├── loki-config.yml           # Loki filesystem 存储 + 7d retention
│   └── loki-size-retention.sh    # /loki 超过 10GiB 时按 OMC 时区最早日期提交删除请求
├── promtail/                      # 旧 Promtail 配置（T-0155 P3 后已下线，保留作历史参考）
│   └── promtail-config.yml
├── otelcol/
│   └── config.yaml               # OTel Collector：
│                                 #   ─ traces:  OTLP gRPC :4317 → Tempo（P1）
│                                 #   ─ metrics: postgresql/redis receiver
│                                 #             → metricstransform → prometheusremotewrite
│                                 #             → Prometheus（P2b）
│                                 #   ─ logs:   filelog receiver (/run/logs/*)
│                                 #             → loki exporter → Loki（P3）
├── tempo/
│   └── tempo.yaml                # Grafana Tempo monolithic + 本地存储 + metrics_generator
│                                 #   service_graphs + span_metrics → Prometheus
│                                 #   产 RED + service map 指标供 SLO 度量
├── grafana/
│   ├── provisioning/
│   │   ├── datasources/
│   │   │   ├── prometheus.yml    # 自动注册 Prometheus 数据源
│   │   │   ├── loki.yml          # 自动注册 Loki 数据源
│   │   │   └── tempo.yml         # 自动注册 Tempo 数据源 + trace-to-logs 跳 Loki
│   │   └── dashboards/default.yml
│   └── dashboards/
│       ├── omc-overview.json
│       ├── omc-storage-queue-governance.json
│       └── omc-alert-overview.json
├── grafana-dashboard.json        # 历史 dashboard 原件
└── README.md                     # 本文件
```

## 端口映射（dev）

| 服务 | 容器端口 | 宿主端口 | 说明 |
|------|--------|--------|------|
| Prometheus | 9090 | **9090** | 标准端口（acs metrics 已让出宿主 9090 → 9095） |
| Grafana | 3000 | **3030** | 宿主 3030（避开 webcode vite dev :3000） |
| AlertManager | 9093 | 9093 | 无冲突 |
| Loki HTTP API | 3100 | 3100 | Grafana 通过此端口查日志 |
| Tempo HTTP API | 3200 | — | 仅容器内（Grafana 走 docker network 直连，不对外） |
| otelcol OTLP gRPC | 4317 | — | 仅容器内（app/acs/worker SDK 在同 network 内推送，不对外） |
| omcgo-acs metrics | 9090 | **9095** | 让出 9090 给 Prometheus 服务 |

宿主访问入口：

- Prometheus UI：<http://localhost:9090>
- Grafana UI：<http://localhost:3030>（admin / admin，dev 默认值）
- AlertManager UI：<http://localhost:9093>
- Loki API：<http://localhost:3100>（无 UI，通过 Grafana 查询）

## 用法

```bash
# 启动监控 + 日志 + 链路追踪栈（8 个服务一起起）
docker-compose -f deployments/docker/docker-compose.yml up -d \
  prometheus alertmanager grafana loki loki-size-retention tempo otelcol

# 健康自检
curl -fsSL http://localhost:9090/-/healthy        # Prometheus
curl -fsSL http://localhost:3030/api/health       # Grafana
curl -fsSL http://localhost:9093/-/healthy        # AlertManager
curl -fsSL http://localhost:3100/ready            # Loki
docker exec docker-tempo-1   wget --spider -q http://localhost:3200/ready    # Tempo（仅容器内）
docker exec docker-otelcol-1 wget --spider -q http://localhost:13133/        # otelcol（仅容器内）

# 查看 scrape target 状态（包含 omc-app / omc-acs / omc-worker 三个 job）
curl -s http://localhost:9090/api/v1/targets | jq '.data.activeTargets[] | {job:.labels.job, health:.health}'

# 查看 Loki 已收到的 label（验证 promtail 推送成功）
curl -s http://localhost:3100/loki/api/v1/labels | jq

# 直接用 Loki API 验证日志写入（不经过 Grafana）
curl -s -G 'http://localhost:3100/loki/api/v1/query_range' \
  --data-urlencode 'query={service="omcgo-app"}' \
  --data-urlencode "start=$(date -d '5 min ago' -u +%s)000000000" \
  --data-urlencode "end=$(date -u +%s)000000000" | jq '.data.result[0].values[:3]'

# 关闭
docker-compose -f deployments/docker/docker-compose.yml down \
  prometheus alertmanager grafana loki loki-size-retention tempo otelcol
```

## 链路追踪（Trace）查询

Grafana → Explore → 选 Tempo 数据源 → 三种查法：

```
# 1. 按 traceID 查（如果在日志里看到 trace_id 字段）
<traceID 32 位 hex>

# 2. TraceQL 按服务名搜
{ resource.service.name = "omcgo-app" }
{ resource.service.name = "omcgo-acs" && duration > 100ms }

# 3. 按 HTTP 路由搜（要求 SDK 注入了 http.route 属性）
{ name =~ "POST /api/.+" }
```

trace-to-logs 关联：span 详情页右上角 → "Logs for this span"，自动按 `service`
label 跳 Loki 查同时段日志（详见 `grafana/provisioning/datasources/tempo.yml`
的 `tracesToLogsV2` 配置）。

trace-to-logs 关联：`logger.L(ctx)` 已在 T-0157 收尾时从 OTel context 抽取
`trace_id` / `span_id` 注入 zap 字段；任何经过 Tracing middleware 的 HTTP
请求所写日志都自动携带 trace_id，Grafana Tempo 数据源点 span → "Logs for
this span" 用 `|= "<traceID>"` 精确匹配 Loki 日志。

## 应用侧 tracer 开关

app/acs/worker 三进程通过 `tracer.enabled` 控制 OTel SDK 上报：

```yaml
# omcgo/cmd/{app,acs,worker}/etc/config.{dev,test,prod}.yaml
tracer:
  enabled: true              # local 环境保持 false（宿主直跑 go run，无 otelcol）
  endpoint: "otelcol:4317"   # 容器内 service name + OTLP gRPC 标准端口
  sample_rate: 1.0           # dev/test 全采；prod 0.1
```

tracer 关闭时 SDK 用 no-op Provider，零开销，与 trace 栈停机互不影响。

## SLO 度量（Tempo metrics_generator）

Tempo 通过 `metrics_generator` 把实时 trace 转化为 RED 指标推 Prometheus。
配置见 `tempo/tempo.yaml` 末段，产出指标：

| 指标 | 用途 |
|------|------|
| `traces_spanmetrics_calls_total{service, span_kind, status_code}` | 请求速率 |
| `traces_spanmetrics_latency_bucket{...}` | 延迟直方图（P50/P90/P99） |
| `traces_service_graph_request_total{client, server}` | 服务依赖边 |
| `traces_service_graph_request_failed_total{...}` | 失败边（错误率） |
| `traces_service_graph_request_server_seconds_*` | 边延迟分布 |

PromQL 算 RED：
```promql
# Rate（每个服务的 QPS）
sum by(service) (rate(traces_spanmetrics_calls_total[5m]))

# Error rate（错误率）
sum(rate(traces_spanmetrics_calls_total{status_code="STATUS_CODE_ERROR"}[5m]))
  / sum(rate(traces_spanmetrics_calls_total[5m]))

# P99 latency
histogram_quantile(0.99,
  sum by(le)(rate(traces_spanmetrics_latency_bucket[5m])))
```

Grafana 内置 "Tempo Service Graph" 面板（Explore → Tempo → Service Graph
tab）会自动用上述指标渲染服务依赖图。

## PG / Redis 指标链路（Phase 2b 改造）

T-0155 Phase 2b 把 `postgres-exporter` / `redis-exporter` 替换为 otelcol
原生 receiver 直连数据库采集，再 prometheusremotewrite 推到 Prometheus：

```
postgres :5432 ──┐  ┌─ metricstransform (重命名 pg_*) ─┐
                 ├─►│                                   ├─► prometheusremotewrite ─► prometheus :9090/api/v1/write
redis    :6379 ──┘  └─ transform/promote_pg_resource ──┘
                       (resource attrs → datapoint attrs，避免 duplicate sample)
```

**关键配置点（坑过）**：
- `target_info.enabled: false`：默认开会生成额外 metadata 指标，pg
  per-table 多个 resource collision
- `transform/promote_pg_resource`：把 `postgresql.database.name` /
  `table.name` / `index.name` 从 resource attr 提到 datapoint attr，
  否则 prometheusremotewrite 不会把这些转成 Prom label → 同名指标多个
  表的样本碰撞 → HTTP 400 duplicate sample
- `postgresql.index.size` / `postgresql.index.scans` 关掉：receiver 0.103
  对多表同名 index（`pkey`）的处理不带 table 维度，会触发 collision
- `add_metric_suffixes: false`：保持指标名与告警表达式精确一致，不让
  prometheusremotewrite 自动加 `_total` 后缀

**告警重设计**：
- `pg_up == 0` / `redis_up == 0` boolean gauge 不再存在（otelcol receiver
  不发健康指示，连不上就静默 fail scrape）→ 重写为
  `absent_over_time(pg_stat_database_numbackends[2m]) == 1`
  / `absent_over_time(redis_uptime_in_seconds[2m]) == 1`
- `up{job=~"postgres|redis"}` 不再合成（remote_write 不产 synthetic up）
  → InfraExporterDown 收窄到 `job=~"nats|minio|otelcol"`
- `pg_stat_database_numbackends` / `pg_settings_max_connections` /
  `redis_memory_used_bytes` / `redis_memory_max_bytes` 经 metricstransform
  别名保留，原 PostgresConnectionsHigh / RedisMemoryHigh 表达式不变

> ⚠️ **OMC 三进程未启动时**，Prometheus targets 会显示 `down`，这是预期行为，
> 不影响监控栈自身 healthy。启动 `app/acs/worker` 三进程后，target 会在
> 一个 scrape interval（15s）内变为 `up`。

## 存储目标映射

`storage-targets.yml` 是部署侧的存储契约，不会被 Prometheus 当作
scrape target 自动加载。当前部署只定义一个物理目标 `filesystem/root`（宿主机
`/`）；PostgreSQL、MinIO、Prometheus、Loki、Tempo 等 named volume 作为同一物理
目标下的逻辑归属和 retention 分类，并限制 MinIO bucket 只使用固定业务分类。

Prometheus 仍从 node-exporter 的 `mountpoint`、容量和 inode 指标，以及 MinIO
cluster endpoint 获取实测值。目标未配置、采集失败或样本过期时，Grafana 必须显示
No data / unavailable，不能用 `0` 代替。部署到非默认 Docker data root 时，应同时
更新 `storage-targets.yml` 的物理 `mountpoint`，再由 OMC 写入保护模块读取同一映射。
逻辑组件不得新增独立容量阈值；只有确认挂载了独立磁盘或接入外部存储时，才新增物理目标。

## 队列治理观测

OMC 业务观测器在 app/worker 启动时分别采集 Redis 和 PostgreSQL 持久化队列：

- Redis 使用 `SCAN`，固定 `queue_family=cmdq|taskq`，输出总长度、活动设备数、最大队列长度、最老任务年龄、扫描耗时和失败状态；设备 SN 只用于内部查询，不进入指标标签。
- PostgreSQL 使用固定 SQL 采集 `device_tasks`、`async_jobs`、parameter-sync/northbound outbox、PM 导出、Trace 导出、备份任务和 dead letters；查询失败保留上次业务快照，并将 `*_up=0`、失败计数递增。
- Grafana 总览为 `OMC - 存储与队列治理`（UID `omc-storage-queue-governance`）；告警页为 `OMC - 告警总览`（UID `omc-alert-overview`），读取 Prometheus `ALERTS`/`ALERTS_FOR_STATE` 展示当前 Firing/Pending 告警。应用内 Go Channel、worker 内存切片、SSE 缓存等不纳入队列积压指标。

## Dashboard 与 TSDB 保护指标

首页 KPI 只读取现有全网小时、天、周聚合结果。`omcgo-app` 暴露查询延迟、
并发、超时/拒绝、缓存、缺失/不完整窗口和上卷延迟指标；OTel Collector 的
`sqlquery/tsdb` 每 30 秒采集 TSDB 临时写入累计值与超过 5 秒的活跃查询。

- `pg_stat_database_temp_bytes` / `temp_files` 是 PostgreSQL 启动或统计重置以来的累计值，看板和告警必须使用 `rate()`。
- `omc-overview` 底部展示 Dashboard P50/P95/P99、保护状态、缓存和完整性。
- `omc-infra` 底部展示 TSDB 临时写入、长查询、CPU/内存和容器磁盘读写。
- 默认告警阈值见 `alerts/dashboard-kpi-alerts.yml`；生产基线稳定后可按容量调整。
- `TSDB_LOG_MIN_DURATION_STATEMENT` 可调慢 SQL 日志阈值，`TSDB_LOG_TEMP_FILES=-1` 可临时关闭临时文件日志；不要关闭 Prometheus 指标采集。

新增 dashboard 或队列指标后，先执行：

```bash
deployments/monitoring/tests/validate-dashboards.sh
jq empty deployments/monitoring/grafana/dashboards/*.json
```

## 加新告警规则

1. 在 `alerts/` 目录下新增 `*.yml`（按业务域命名，例：`acs-session.yml`、`f04-alarm.yml`）。
2. 文件结构遵循 [Prometheus alerting rules](https://prometheus.io/docs/prometheus/latest/configuration/alerting_rules/)：

   ```yaml
   groups:
     - name: <business.area>
       interval: 30s
       rules:
         - alert: <CamelCaseName>
           expr: <PromQL>
           for: <持续时间>
           labels:
             severity: critical | warning | info
             domain: omc
           annotations:
             summary: "<一句话>"
             description: "<多行，包含 Impact / Action>"
   ```
3. 热加载（无需重启）：`curl -X POST http://localhost:9090/-/reload`。
4. 在 Prometheus UI **Status → Rules** 下确认新规则已加载。

## 加新 Grafana Dashboard

1. 在 Grafana UI 编辑 dashboard，**Share → Export → Save to file** 导出 JSON。
2. 把 JSON 放进 `grafana/dashboards/` 目录。
3. 30s 内 Grafana provisioner 自动加载（`updateIntervalSeconds`）。
4. 注意 dashboard 顶部 `uid` 字段必须唯一，否则会覆盖已有 dashboard。

提交前先运行静态入口，新增的 JSON 会被自动发现；四个现有 provisioned
dashboard 缺失、JSON 无效、UID 重复，或仍使用
`namespace="omcgo"` / `name=~` 失效筛选时都会失败：

```bash
chmod +x deployments/monitoring/tests/validate-dashboards.sh
deployments/monitoring/tests/validate-dashboards.sh
```

同时校验存储目标契约及固定 MinIO bucket 分类：

```bash
chmod +x deployments/monitoring/tests/validate-storage-targets.sh
deployments/monitoring/tests/validate-storage-targets.sh
```

容器服务级 CPU/内存面板依赖 cAdvisor 导出的有限 Compose service 标签；提交
cAdvisor 或容器资源面板改动时，同时执行：

```bash
chmod +x deployments/monitoring/tests/validate-cadvisor-service-labels.sh
deployments/monitoring/tests/validate-cadvisor-service-labels.sh
```

`tests/promql-probes.txt` 是资源、队列和写入保护的查询清单。它不是 dashboard
或告警规则的替代品；其中标为 `MUST-HAVE` 的 probe 在对应 exporter/service 启动后
必须返回时间序列。

### PromQL 到浏览器的验证顺序

每次改动 PromQL、Grafana panel 或指标导出时，必须按以下顺序验收：

1. 先经 Prometheus `/api/v1/query` 验证查询本身。例如：

   ```bash
   curl -sG http://localhost:9090/api/v1/query \
     --data-urlencode 'query=omc_pm_queue_pending{subject="pm.file.received",durable="pm-workers"}' \
     | jq
   ```

2. 再在 Grafana 的对应 panel 中确认同一时间范围、数据源和 legend 显示的值与
   Prometheus 返回一致。
3. 最后在浏览器打开实际 provisioned dashboard，确认 panel 已加载、No data 和
   错误状态可见、刷新后仍保持正确。**不得只因 JSON 中存在 panel 就判定功能完成。**

应用进程内部内存队列不纳入这些 queue probes：包括 Go Channel、Worker Channel、
参数同步内存 Channel、Trace 本地 Capture Queue，以及其他没有持久化权威来源的临时
缓冲。诊断这类队列应使用进程运行时指标或日志，不应伪造成持久化队列的 Prometheus
时间序列。

面板和告警必须区分三种状态：Prometheus 返回一个值为 `0` 的样本，才是队列为空、
没有拒绝写入或资源使用为零的真实 `zero`；查询没有返回时间序列时必须显示 `No data`，
不能补零；Prometheus、exporter 或 Grafana 查询报错时是 `failure`，应显示错误并排查
采集链路。写入保护指标在 Task 8 暴露前允许 `No data`，但不得据此推断写入被允许。

## 生产部署注意事项

> 本目录的所有配置仅适用于 **dev 环境**。生产部署前必须修改：

| 项 | dev 默认值 | 生产要求 |
|----|----------|---------|
| Grafana admin 密码 | `admin` (env `GF_SECURITY_ADMIN_PASSWORD`) | 通过 K8s Secret / Vault 注入强密码，不进 git |
| Grafana 匿名访问 | 关闭 | 保持关闭，对接公司 SSO（OAuth2 / LDAP） |
| AlertManager webhook | `http://example.invalid/webhook` (占位) | 指向真实 omcgo-app 告警入口或公司 IM 通道 |
| Prometheus 保留期 | 15d | 视容量调整（推荐 30d，并启用 remote_write 落入长存） |
| TLS | 无 | 接入 ingress / TLS termination |

## 与 omcgo 的指标契约

### 资源与队列指标契约（Task 1）

后续资源监控、队列观测器和 Grafana 面板必须复用下面的命名、标签和语义。
`queue`、`status`、`result`、`subject`、`durable` 只能取配置或代码中登记的有限枚举；
不得把 device SN、完整 Redis key、对象路径、数据库 row ID 或请求 ID 作为标签。

| 指标后缀/指标 | 标签 | 单位 | 空值/0 语义 | 采集失败语义 |
|---|---|---|---|---|
| `pending` | `queue,status` | 条 | 队列为空时为真实 `0` | 保留上次值 |
| `oldest_age_seconds` | `queue,status` | 秒 | 无积压时为真实 `0` | 保留上次值 |
| `overdue_oldest_age_seconds` | `queue,status` | 秒 | 未超过任务自身 `expires_at` 时为真实 `0` | 保留上次值 |
| `failed_total` | `queue` | 次 | 尚无失败时可从 `0` 开始 | 观测失败不冒充业务失败 |
| `dead_letter_total` | `queue` | 条 | 无死信时为真实 `0` | 保留上次值并记录观测失败 |
| `processed_total` | `queue,result` | 次 | 尚未处理时可从 `0` 开始 | 观测失败不冒充处理结果 |
| `observer_failures_total` | `queue` | 次 | 尚无失败时可从 `0` 开始 | 每次查询/采集失败递增 |
| `omc_pm_queue_pending` | `subject,durable` | 条 | Worker 启动和空队列均为真实 `0` | 保留上次值 |
| `omc_pm_queue_oldest_age_seconds` | `subject,durable` | 秒 | 无积压时为真实 `0` | 保留上次值 |
| `omc_pm_queue_sample_failures_total` | `subject,durable` | 次 | 尚无失败时可从 `0` 开始 | 失败时递增 |

统一持久化队列目录固定为：`device_tasks`、`async_jobs`、
`parameter_sync_outbox`、`northbound_outbox`、`pm_kpi_export`、`trace_export`、
`backup_tasks`、`dead_letters`。状态值固定为 `pending`、`sent`、`running`、
`succeeded`、`failed`、`dead_letter`；`processed_total` 的 `result` 使用
`succeeded` 或 `failed`。

查询语义必须区分“真实 0”和“不可用”：PromQL 返回存在且值为 `0` 的时间序列，
表示观测器成功采集到空队列；查询失败、序列缺失或样本过期表示指标不可用，面板和告警
不得把它转换成 `0`。NATS/PM 等观测器应保留上次业务值，并递增对应的
`*_observer_failures_total` 或 `omc_pm_queue_sample_failures_total`。

本期明确排除应用进程内部内存队列：Go Channel、Worker Channel、参数同步内存
Channel、Trace 本地 Capture Queue，以及其他仅存在于进程内且没有持久化权威来源的
临时缓冲。它们不创建 Prometheus 队列时间序列；如需诊断，应使用进程级运行时指标或日志。

omcgo 三进程通过以下端口暴露 `/metrics`（容器内）：

| 进程 | metrics 端口 | 验证 |
|------|------------|------|
| omcgo-app | `:9091` | `omcgo/cmd/app/etc/config.dev.yaml` `metrics.port` |
| omcgo-acs | `:9090` | `omcgo/cmd/acs/etc/config.dev.yaml` `metrics.port` |
| omcgo-worker | `:9092` | `omcgo/cmd/worker/etc/config.dev.yaml` `metrics.port` |

修改任一 metrics 端口时，必须同步修改 `prometheus.yml` 的 `scrape_configs`。

## 日志查询（Loki + Grafana）

打开 Grafana <http://localhost:3030> → 左侧 **Explore** → 数据源选 **Loki**。

### 日志清理策略

- **时间条件**：`loki-config.yml` 的 `retention_period: 168h` 由 Loki compactor
  清理超过 7 天的日志。
- **容量条件**：`loki-size-retention` 每 10 分钟检查 `/loki` Docker 卷；实际占用
  超过 10 GiB 时，通过 Loki 删除 API 提交一个按 OMC 系统时区计算的最早完整日期区间
  删除请求，下一轮再继续推进，直到容量回落。时区来源是系统配置页的
  `timezoneCode`，app 会同步到 `/var/lib/omcgo/timezone/system-timezone`。
  时区文件尚未生成、为空或内容无效时，清理器使用容器绑定的系统本地时间继续清理。
- 两个条件相互独立，任一条件满足都会启动对应清理流程。删除请求不会直接删除 TSDB
  文件；当前 `retention_delete_delay: 2h`，因此容量清理开始后，磁盘空间通常会在
  compactor 处理并完成延迟删除后下降。
- 容量清理进度保存在 `/loki/compactor/size-retention.state`，重启后不会反复提交同一
  日期区间。

### Promtail 注入的 label 体系

| label | 取值范围 | 用途 |
|-------|---------|------|
| `service` | `omcgo-app` / `omcgo-acs` / `omcgo-worker` / `nginx` / `frontend` | 主要过滤维度 |
| `job` | 同 service | 历史习惯 label |
| `deployment_unit` | `app` / `acs` / `worker` | 与 Prometheus 标签对齐 |
| `level` | `info` / `warn` / `error` / `debug`（从 zap JSON 抽取） | 严重级过滤 |
| `log_type` | `access` / `error`（仅 nginx） | nginx 日志类型 |

> ⚠️ **不要**把 `request_id` / `device_sn` / `trace_id` 提为 label——这些是高基数字段，
> 会让 Loki stream 数爆炸。它们保留在 message body，用 `| json` 解析后过滤。

### 常用 LogQL

```logql
# 全文搜 panic（最常用，事故现场）
{service="omcgo-app"} |= "panic"

# 三进程 error 全量
{level="error"}

# ACS 模块 error
{service="omcgo-acs", level="error"}

# 按设备 SN 过滤（device_sn 在 JSON 体内）
{service="omcgo-app"} | json | device_sn="120200024719AAB0039"

# 按 request_id 追踪一次完整请求
{service="omcgo-app"} | json | request_id="app-20260511153032-36520199"

# 各服务 5 分钟内 error 速率（指标化）
sum by (service) (count_over_time({level="error"}[5m]))

# nginx 5xx 响应
{service="nginx", log_type="access"} |~ " (5[0-9]{2}) "

# 排除某些噪声日志
{service="omcgo-app"} != "health" != "metrics"
```

### Loki → 指标联动

Loki 支持把 LogQL 转成 Prometheus-like metric，可以在同一个 Grafana 面板里：
- 上面板用 Prometheus 指标看 QPS / 延迟
- 下面板用 Loki 看同时段错误日志
- 点击指标尖刺 → Grafana 自动跳到对应时段的日志

## 关联文档

- [docker-compose.yml](../docker/docker-compose.yml) — service 定义
- [grafana-dashboard.json](./grafana-dashboard.json) — OMC 既有 dashboard 原件
- [docs/methodology/AI承诺对峙清单.md](../../docs/methodology/AI承诺对峙清单.md) — W1.7 验证规格

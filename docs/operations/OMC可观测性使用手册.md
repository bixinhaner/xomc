# OMC 可观测性使用手册

> 面向运维、交付、客户现场工程师的统一可观测性平台使用说明。
> 适用版本：OMC Go（app / acs / worker 三进程）+ 监控栈（otelcol / Prometheus / Loki / Tempo / Grafana）。
> 入口：**http://localhost:3030**（默认账号 `admin` / 密码 `admin123`）。

---

## 0. 文档导航

| 章节 | 内容 |
|------|------|
| §1 | 可观测性总览（三支柱 + 架构图） |
| §2 | 组件清单与集成方式 |
| §3 | 访问入口 `:3030` 系统说明（登录、首屏、能看到什么） |
| §4 | 使用指南：查指标 / 查日志 / 查链路 / 三支柱联动 |
| §5 | 建议配置方案（快捷入口、Dashboard 矩阵、告警、权限） |
| §6 | 商业化能力提升（SLO / 多租户 / 报表 / 品牌化） |
| §7 | 行业最佳实践 |
| §8 | 运维与排障 |
| §9 | 附录（端口表、查询速查、安全基线） |

---

## 1. 可观测性总览

OMC 监控栈遵循**可观测性三支柱**模型，并由 **OpenTelemetry Collector（otelcol）** 做统一采集中枢，**Grafana** 做统一展示与关联下钻：

| 支柱 | 回答的问题 | 组件 | 存什么 |
|------|-----------|------|--------|
| **Metrics（指标）** | "系统现在健不健康？" | Prometheus | ACS 会话数、设备总数、告警数、RPC 延迟、连接池、CPU/内存…… |
| **Logs（日志）** | "到底发生了什么事？" | Loki | app/acs/worker 的 zap JSON 日志 + nginx access/error 日志 |
| **Traces（链路）** | "一次请求在三进程间怎么流转的？" | Tempo | 一次 API / TR-069 会话跨 app↔acs↔worker 的完整调用链 |

### 1.1 整体架构图

```
                         ┌──────────────── Grafana :3030 ────────────────┐
                         │   统一查询 + Dashboard + trace↔logs↔metrics 联动  │
                         └───┬───────────────┬───────────────┬───────────┘
                      PromQL │          LogQL │        TraceQL │
                       ┌─────▼─────┐   ┌──────▼──────┐  ┌──────▼──────┐
                       │ Prometheus │   │    Loki     │  │    Tempo    │
                       │   :9090    │   │   :3100     │  │   :3200     │
                       │ 指标+告警   │   │   日志       │  │  链路+RED   │
                       └──▲──▲───▲──┘   └──────▲──────┘  └──────▲──────┘
            pull /metrics │  │   │ remote_write │ push          │ OTLP gRPC
        ┌─────────────────┘  │   │              │               │
        │ nats-exporter :7777 │   │       ┌──────┴───────────────┴──────┐
        │ minio /metrics      │   └───────┤   OpenTelemetry Collector     │
        │ otelcol :8888       │   RED 指标 │   (otelcol :4317 / :8888)     │
        │ app/acs/worker      │ ◄─────────┤  · OTLP traces → Tempo        │
        │   /metrics          │  Tempo    │  · pg/redis receiver → Prom   │
        └─────────────────────┘ metrics_gen│  · filelog → Loki            │
                                            └──────▲───────────────────────┘
                          PG/Redis 直采 + /run/logs/{app,acs,worker,nginx}/*.log
                                            │
                                   告警 → Alertmanager :9093
```

---

## 2. 组件清单与集成方式

### 2.1 端口与角色一览

| 组件 | 容器端口 | 宿主端口 | 角色 | 集成方式 |
|------|---------|---------|------|---------|
| **Grafana** | 3000 | **3030** | 统一可视化门面 | 浏览器访问；provisioning 自动注册 3 个数据源 |
| **Prometheus** | 9090 | 9090 | 指标存储 + 告警评估 | pull + remote_write 双模式 |
| **Loki** | 3100 | 3100 | 日志聚合 + LogQL | 由 otelcol `loki` exporter 推送 |
| **Tempo** | 3200 / 4317 | — | 链路存储 + RED 指标发动机 | 由 otelcol `otlp/tempo` exporter 推送 |
| **otelcol** | 4317 / 8888 / 13133 | — | 统一采集中枢 | 见下 §2.2 |
| **Alertmanager** | 9093 | 9093 | 告警路由 / 去重 / 通知 | Prometheus 评估后推送 |
| nats-exporter | 7777 | — | NATS 指标桥 | Prometheus 直接 pull |
| minio | 9000 | — | 自带 `/minio/v2/metrics/cluster` | Prometheus 直接 pull |

> **端口约定**：宿主 `3030 → 容器 3000` 是为了避开前端 vite dev server 的 `:3000`。请始终用 **http://localhost:3030** 访问 Grafana。

### 2.2 三条集成链路（数据怎么进来的）

**① Traces（链路）— OTLP 直推**
```
app / acs / worker  --(OTLP gRPC)-->  otelcol :4317  -->  Tempo :4317  -->  本地 wal/blocks
```
业务代码用 OpenTelemetry Go SDK 上报 span，统一进 otelcol，再转 Tempo 存储。

**② Metrics（指标）— pull + remote_write 混合**
- **Pull（Prometheus 主动拉）**：app `:9091`、acs `:9090`、worker `:9092` 的 `/metrics`；nats-exporter `:7777`；minio `/metrics`；otelcol 自身 `:8888`。
- **Remote Write（被动接收）**：
  - PostgreSQL / Redis 指标 → otelcol 原生 receiver 直连数据库采集 → `prometheusremotewrite` 推到 `:9090/api/v1/write`；
  - Tempo `metrics_generator` 从链路实时算出的 **RED 指标**（rate/error/duration）也 remote_write 进 Prometheus。
- otelcol 用 `metricstransform` 把 OTel 语义名改回旧 exporter 名（`pg_*` / `redis_*`），**现有告警与面板零改动**。

**③ Logs（日志）— filelog 采集（Promtail 已下线）**
```
/run/logs/{app,acs,worker,nginx}/*.log  -->  otelcol filelog receiver
   --> json_parser 提取 timestamp/level/service/trace_id  -->  loki exporter  -->  Loki :3100
```
> **基数纪律**：仅 `service.name` 提升为 Loki label；`trace_id` / `request_id` / `device_sn` 等高基数字段留在日志正文，用 LogQL `| json | device_sn="..."` 过滤，避免 label 爆炸。

**④ 三支柱关联（T-0157）**
`logger.L(ctx)` 把 `trace_id` / `span_id` 注入 zap 日志字段；Grafana 中点击 Tempo 的 span 即可按 `service.name` + 时间窗口跳转到 Loki 对应日志，实现 trace → logs 一键下钻。

---

## 3. 访问入口 `:3030` 系统说明

### 3.1 登录

| 项 | 值 |
|----|----|
| 地址 | **http://localhost:3030**（远程：`http://<服务器IP>:3030`） |
| 账号 | `admin` |
| 密码 | `admin123` |
| 注册 | 已禁用（`GF_USERS_ALLOW_SIGN_UP=false`） |
| 匿名访问 | 已禁用（`GF_AUTH_ANONYMOUS_ENABLED=false`） |

> ⚠️ **首登改密**：首次登录会强制提示修改口令。生产环境务必改强口令（见 §9.3）。
> ⚠️ **凭据差异提示**：dev 版 `deployments/docker/docker-compose.yml` 内置默认是 `admin/admin`；release 离线包通过 `GRAFANA_ADMIN_PASSWORD` 环境变量注入。**本手册以现场约定 `admin123` 为准**——若登录失败，请核对部署所用 compose 文件中的 `GF_SECURITY_ADMIN_PASSWORD`。

### 3.2 在 `:3030` 能看到什么

登录后默认已就绪以下内容（全部由 provisioning 自动加载，开箱即用）：

**数据源（Connections → Data sources）**：`Prometheus`、`Loki`、`Tempo` 三个已自动注册、已配好关联。

**Dashboard（Dashboards → OMC 文件夹）**：内置 `OMCGo - Operations Dashboard`（`omc-overview.json`），包含面板：

| 类别 | 面板 |
|------|------|
| 业务概览 | ACS Active Sessions、Total Devices、Active Alarms、Pod Count |
| 南向 / 接口 | Inform Rate (req/s)、RPC Latency、HTTP Request Rate by Component |
| 消息队列 | NATS JetStream Queue Depth |
| 资源 | CPU Usage by Pod、Memory Usage by Pod |
| Go 运行时 | Goroutines、GC Pause |

**Explore（左侧指南针图标）**：临时排障的自由查询入口，可切换三个数据源写 PromQL / LogQL / TraceQL。

---

## 4. 使用指南

### 4.1 查指标（Metrics）

**方式 A — 看大盘（推荐日常巡检）**
`Dashboards` → `OMC` 文件夹 → `OMCGo - Operations Dashboard`，右上角选时间范围（如 Last 1 hour）。

**方式 B — 自由查询（排障）**
`Explore` → 选 `Prometheus` → 输入 PromQL：

| 目的 | PromQL |
|------|--------|
| ACS 全局实时在线会话 | `omc_acs_global_active_sessions` |
| Inform 速率 | `rate(omc_acs_inform_total[5m])` |
| API P99 延迟（RED，来自 Tempo） | `histogram_quantile(0.99, sum by(le)(rate(traces_spanmetrics_duration_seconds_bucket[5m])))` |
| API 错误率（RED） | `sum(rate(traces_spanmetrics_calls_total{status_code="STATUS_CODE_ERROR"}[5m])) / sum(rate(traces_spanmetrics_calls_total[5m]))` |
| PG 连接数占比 | `pg_stat_database_numbackends / pg_settings_max_connections` |
| Redis 内存占比 | `redis_memory_used_bytes / redis_memory_max_bytes` |

> `acs_global_active_sessions` 是并发、容量和准入判断的权威指标。
> `acs_active_sessions` 仅为兼容旧查询而保留，已弃用且数值与全局指标一致；
> `acs_local_tracked_sessions` 是单个 ACS 进程最多保留五分钟的会话 ID 数，仅用于诊断，
> 可能高于实时并发，不能用于容量判断。

### 4.2 查日志（Logs）

`Explore` → 选 `Loki` → 写 LogQL：

| 目的 | LogQL |
|------|-------|
| App 全量日志 | `{service="omcgo-app"}` |
| ACS error 级 | `{service="omcgo-acs"} \| json \| level="error"` |
| 全文搜 panic | `{service="omcgo-app"} \|= "panic"` |
| 按设备过滤 | `{service="omcgo-app"} \| json \| device_sn="120200024719AAB0039"` |
| 各服务 5min error 速率 | `sum by (service) (count_over_time({level="error"}[5m]))` |
| 按 trace 关联日志 | `{service="omcgo-app"} \|= "<traceId>"` |

> Loki label 只有 `service`（omcgo-app / omcgo-acs / omcgo-worker / nginx）和 `level`，其余字段先 `| json` 再过滤。

### 4.3 查链路（Traces）

`Explore` → 选 `Tempo` → TraceQL 或直接粘 traceID：

| 目的 | TraceQL |
|------|---------|
| App 全部 trace | `{ resource.service.name = "omcgo-app" }` |
| ACS 慢请求（>100ms） | `{ resource.service.name = "omcgo-acs" && duration > 100ms }` |
| 按接口名 | `{ name =~ "POST /api/.+" }` |
| 服务依赖拓扑 | Tempo 面板的 `Service Graph` 标签页（自动绘制） |

### 4.4 三支柱联动（核心价值）

典型排障动线 —— **从"指标异常"一路下钻到"那一行日志"**：

```
1. Dashboard 看到 "RPC Latency" P99 飙高 / "Active Alarms" 突增
          ↓ 点面板 → Explore
2. Tempo 用 TraceQL 找慢/错 trace：{ duration > 1s }
          ↓ 点开某条 trace 的 span
3. 点 span 上的 "Logs for this span" → 自动跳 Loki
          ↓ 按 service + 时间窗口 + traceId 召回
4. Loki 看到该次调用链的精确日志上下文，定位根因
```

反向：在 Loki 看到一条带 `trace_id` 的 error 日志，复制 trace_id 到 Tempo 即可还原完整调用链。

---

## 5. 建议配置方案

### 5.1 快捷入口配置（提升日常效率）

| 配置项 | 操作路径 | 建议值 |
|--------|---------|--------|
| **设为首页大盘** | Dashboard 右上角 ⭐ 收藏 → Profile → Preferences → Home Dashboard | `OMCGo - Operations Dashboard` |
| **收藏常用大盘** | 各 Dashboard 点 ⭐ | 概览盘、ACS 盘、基础设施盘 |
| **Explore 查询历史** | Explore → Query history → ⭐ Star | 常用 LogQL / PromQL |
| **时间范围默认** | Preferences → Timezone | `Asia/Shanghai`（与容器 TZ 一致） |
| **自动刷新** | Dashboard 右上角刷新下拉 | 巡检盘设 `30s`，排障时关闭 |

### 5.2 推荐 Dashboard 矩阵（分层建设）

> 现状仅 1 个综合盘，建议按"业务 → 三进程 → 基础设施"分层扩展：

| 层 | Dashboard | 关键面板 | 受众 |
|----|-----------|---------|------|
| **L1 业务总览** | OMC Overview（已有） | 设备总数、在线率、活动告警、Inform 速率 | 值班 / 管理层 |
| **L2 南向 ACS** | ACS Detail（建议新增） | 会话数、RPC 延迟分位、限流拒绝、SOAP Fault 率 | ACS 工程师 |
| **L2 数据管线** | PM/MR Pipeline（建议新增） | KPI 聚合时延、队列深度、文件处理失败率 | 数据工程师 |
| **L3 基础设施** | Infra（建议新增） | PG 连接/慢查询、Redis 内存/命中率、NATS 积压、MinIO 容量 | DBA / 运维 |
| **L3 SLO** | RED / SLO（建议新增） | 各接口 Rate/Error/Duration + 错误预算 | SRE / 交付 |

### 5.3 告警与通知配置

**已内置告警规则**（`deployments/monitoring/alerts/`，Prometheus 自动加载）：

| 文件 | 规则 |
|------|------|
| `otelcol-alerts.yml` | OtelcolTraceExportFailing、OtelcolMetricExportFailing、OtelcolReceiverRefusing、OtelcolMemoryLimiterDropping、OtelcolExporterQueueHigh |
| `infra-alerts.yml` | InfraExporterDown、PostgresMetricsAbsent、RedisMetricsAbsent、PostgresConnectionsHigh（>80%）、RedisMemoryHigh（>85%） |

**建议补充的业务告警**：ACS 会话数骤降、Inform 速率异常、活动告警激增、KPI 聚合任务连续失败、API 错误率 > 阈值（基于 RED 指标）。

**通知渠道配置**：编辑 Alertmanager 配置接入邮件 / 钉钉 / 企业微信 / Webhook → OMC 通知中心（`internal/notification/`），与 F04 告警体系打通，形成"指标告警 + 设备告警"统一出口。

### 5.4 团队 / 权限 / 多组织

| 角色 | Grafana Role | 可见范围 |
|------|-------------|---------|
| 管理员 | Admin | 全部 + 数据源 / 用户管理 |
| 运维工程师 | Editor | 编辑大盘、配置告警 |
| 值班 / 客服 | Viewer | 只读大盘 |
| 客户（如开放） | Viewer + 限定 Folder | 仅其租户大盘（见 §6.2） |

---

## 6. 商业化能力提升

> 让可观测性从"内部排障工具"升级为"可交付、可度量、可对客的商用能力"。

### 6.1 SLA / SLO 可度量（运营商验收的硬通货）
- 基于 Tempo `metrics_generator` 的 RED 指标，定义并展示 OMC 自身的 SLO：
  - **可用性**：API 成功率 ≥ 99.9%
  - **时延**：管理面 API P99 ≤ 500ms；ACS Inform 处理 P99 ≤ 200ms
  - **数据及时性**：PM/KPI 聚合延迟 ≤ 15min
- 建立"错误预算（Error Budget）"看板，把抽象的"系统稳定"变成可向客户出示的量化承诺。

### 6.2 多租户 / 按运营商隔离视图
- 利用 Grafana Folder + Team + 数据源变量，为 CMCC / CTCC / CUCC 或不同地市建立**独立大盘视图**，客户只看到自己的数据。
- 指标 / 日志 / 链路均已带 `cluster` / `service` 标签，可扩展 `carrier` / `region` 维度做过滤。

### 6.3 运营报表与定期导出
- Grafana 报表（或 PromQL 定时导出）生成**日报 / 周报 / 月报**：设备在线率趋势、Top 告警、KPI 达标率、容量水位。
- 作为交付物随版本附带，体现"可运营"成熟度。

### 6.4 品牌化与白标（White-label）
- 替换 Grafana Logo / 标题 / 主题色为 OMC 品牌，登录页与导航统一视觉。
- 内嵌（iframe / panel embed）到 OMC Web 控制台，让监控成为产品的一部分而非外挂工具。

### 6.5 容量与成本可视化
- MinIO 容量、PG/TimescaleDB 磁盘、Tempo/Loki 存储水位上盘，支撑容量规划与扩容决策（10 万 → 100 万设备演进路径）。

---

## 7. 行业最佳实践对照

| 最佳实践 | 行业共识 | OMC 现状 / 建议 |
|---------|---------|----------------|
| **三支柱关联** | Metrics→Traces→Logs 一键下钻（Grafana LGTM Stack） | ✅ 已配 tracesToLogsV2 + serviceMap |
| **OTel 作为统一采集标准** | 厂商中立、一次接入多后端 | ✅ otelcol 已是唯一中枢 |
| **日志低基数 label** | 高基数字段进正文不进 label | ✅ 已遵循（仅 service/level 为 label） |
| **RED / USE 方法论** | RED 看请求、USE 看资源 | ⚠️ RED 已有数据源，建议补专用 SLO 盘 |
| **告警分级 + 抑制** | Critical/Warning 分级、避免告警风暴 | ⚠️ 建议补 severity 分级 + Alertmanager 抑制规则 |
| **采样策略** | dev 全采、prod 尾部采样降本 | ✅ 已规划（dev 1.0 / prod 0.1，保留 7d） |
| **Dashboard as Code** | 大盘 JSON 进 Git、provisioning 下发 | ✅ 已 provisioning，建议大盘变更走 PR |
| **可观测性即发布门禁** | 新指标必须有告警规则才算上线 | ⚠️ 建议纳入 Release Gate（见 `docs/project/release-gate.md`） |
| **生产横向扩展** | Collector agent+gateway 两层、后端微服务化 | ✅ 已写明 100 万规模演进路径 |

---

## 8. 运维与排障

### 8.1 健康检查
```bash
curl -fsSL http://localhost:3030/api/health        # Grafana
curl -fsSL http://localhost:9090/-/healthy         # Prometheus
curl -fsSL http://localhost:3100/ready             # Loki
curl -fsSL http://localhost:3200/ready             # Tempo
curl -fsSL http://localhost:13133                  # otelcol health_check
```

### 8.2 常见问题速查

| 现象 | 排查方向 |
|------|---------|
| Grafana 打不开 | 端口确认 `3030`（非 3000）；`docker ps` 看 grafana 容器；防火墙放行 |
| 指标空白 | Prometheus Targets 页（`:9090/targets`）看 scrape 状态；otelcol `:8888` 自监控 |
| PG/Redis 指标断流 | 触发 `PostgresMetricsAbsent` / `RedisMetricsAbsent`；查 otelcol 与数据库连通性 |
| 日志查不到 | `/run/logs/*/` 是否有文件；otelcol filelog 是否 `start_at: end`（启动后才采新行）；otelcol 容器需有日志卷读权限（注意：otelcol 以 root 运行以读业务日志） |
| Trace 缺失 | app/acs/worker OTLP 上报配置；Tempo `:4317` 连通；otelcol 队列水位告警 |
| Trace 跳 Logs 无结果 | 时间窗口（±1min）；service 名是否一致（omcgo-app/acs/worker）；trace_id 是否已注入日志 |

### 8.3 自监控
otelcol 自身指标在 Prometheus `job="otelcol"`，已配 5 条告警监控管道健康（导出失败、receiver 拒收、内存丢数据、队列水位）。**监控系统自己也被监控**。

---

## 9. 附录

### 9.1 端口速查表

| 服务 | 宿主端口 | 用途 |
|------|---------|------|
| Grafana | **3030** | 可视化入口（→容器 3000） |
| Prometheus | 9090 | 指标 + 告警 + remote_write 接收 |
| Loki | 3100 | 日志 API |
| Tempo | 3200 | 链路查询 API（摄入 4317） |
| Alertmanager | 9093 | 告警通知 |
| app / acs / worker metrics | 9091 / 9090 / 9092 | 业务指标 `/metrics` |
| otelcol | 4317 / 8888 / 13133 | OTLP 摄入 / 自监控 / 健康检查 |

### 9.2 查询语言速查

- **PromQL（指标）**：`rate()`、`histogram_quantile()`、`sum by()`
- **LogQL（日志）**：`{service="..."} | json | field="..."`、`|= "关键词"`、`count_over_time()`
- **TraceQL（链路）**：`{ resource.service.name = "..." && duration > 100ms }`

### 9.3 安全基线（生产必做）

- [ ] 修改 Grafana 默认口令（不要用 admin/admin123 上生产）
- [ ] 关闭匿名访问与注册（已默认关闭）
- [ ] `:3030` / `:9090` / `:3100` / `:3200` 不直接暴露公网，走反向代理 + TLS + 鉴权
- [ ] 数据源连接（otelcol→PG/Redis）口令从环境变量 / Secret 注入，勿硬编码
- [ ] 定期备份 `grafanadata` 卷（自定义大盘、用户、告警）
- [ ] 设置 Tempo / Loki 保留期与容量上限，防磁盘打满

### 9.4 相关文档

| 文档 | 路径 |
|------|------|
| otelcol 迁移设计 | `docs/design/observability-otelcol-migration-plan-20260520.md` |
| 监控栈 README | `deployments/monitoring/README.md` |
| 内网离线部署手册 | `docs/operations/OMC内网离线部署手册（运维侧）.md` |
| 发布门控 | `docs/project/release-gate.md` |
| 配置文件 | `deployments/monitoring/{otelcol,prometheus.yml,tempo,loki,grafana}` |

---

*本手册随监控栈配置演进，配置变更请同步更新本文档。*

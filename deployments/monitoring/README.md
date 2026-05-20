# OMC Monitoring Stack

Prometheus + AlertManager + Grafana + Loki + Promtail + **otelcol + Tempo**
一键起栈，覆盖 OMC 三进程（app / acs / worker）的**指标 + 日志 + 链路追踪**
三通道可观测性。

## 文件结构

```
deployments/monitoring/
├── prometheus.yml                # Prometheus 主配置（scrape + 告警路由）
├── alertmanager.yml              # AlertManager 路由 + receiver（占位 webhook）
├── alerts/
│   ├── omc-rules.yml             # starter 告警规则（三进程存活）
│   └── connection-pool-alerts.yml
├── loki/
│   └── loki-config.yml           # Loki 单节点 filesystem 存储 + 7d retention
├── promtail/
│   └── promtail-config.yml       # 日志采集（zap JSON 解析 + level/service label）
├── otelcol/
│   └── config.yaml               # OTel Collector：OTLP gRPC receiver → batch → Tempo（T-0155 Phase 1）
├── tempo/
│   └── tempo.yaml                # Grafana Tempo 单进程 monolithic 配置 + 本地存储
├── grafana/
│   ├── provisioning/
│   │   ├── datasources/
│   │   │   ├── prometheus.yml    # 自动注册 Prometheus 数据源
│   │   │   ├── loki.yml          # 自动注册 Loki 数据源
│   │   │   └── tempo.yml         # 自动注册 Tempo 数据源 + trace-to-logs 跳 Loki
│   │   └── dashboards/default.yml
│   └── dashboards/
│       └── omc-overview.json
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
# 启动监控 + 日志 + 链路追踪栈（7 个服务一起起）
docker-compose -f deployments/docker/docker-compose.yml up -d \
  prometheus alertmanager grafana loki promtail tempo otelcol

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
  prometheus alertmanager grafana loki promtail tempo otelcol
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

> ⚠️ **当前阶段 trace_id 尚未注入 zap 日志字段**（待 T-0157 续）。
> 暂时 trace-to-logs 会按服务名+时间窗口召回近似日志，精确匹配能力等
> T-0157 完成后自动到位。

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

> ⚠️ **OMC 三进程未启动时**，Prometheus targets 会显示 `down`，这是预期行为，
> 不影响监控栈自身 healthy。启动 `app/acs/worker` 三进程后，target 会在
> 一个 scrape interval（15s）内变为 `up`。

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

omcgo 三进程通过以下端口暴露 `/metrics`（容器内）：

| 进程 | metrics 端口 | 验证 |
|------|------------|------|
| omcgo-app | `:9091` | `omcgo/cmd/app/etc/config.dev.yaml` `metrics.port` |
| omcgo-acs | `:9090` | `omcgo/cmd/acs/etc/config.dev.yaml` `metrics.port` |
| omcgo-worker | `:9092` | `omcgo/cmd/worker/etc/config.dev.yaml` `metrics.port` |

修改任一 metrics 端口时，必须同步修改 `prometheus.yml` 的 `scrape_configs`。

## 日志查询（Loki + Grafana）

打开 Grafana <http://localhost:3030> → 左侧 **Explore** → 数据源选 **Loki**。

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

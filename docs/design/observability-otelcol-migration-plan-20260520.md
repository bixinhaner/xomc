# OMC 观测层迁移到 OpenTelemetry Collector — 方案审议

> **状态**：草案，待审批
> **作者**：Claude（与维护者协作）
> **日期**：2026-05-20
> **关联 Backlog**：建议落 **T-0155**（接 T-0151/T-0153/T-0154 监控系列）
> **关联文档**：
> - `omcgo/CLAUDE.md §3 技术栈表`（OpenTelemetry 已列入）
> - `deployments/monitoring/prometheus.yml`
> - `deployments/monitoring/alerts/infra-alerts.yml`
> - `internal/core/components/tracer.go`
> - `internal/core/components/monitor/metrics.go`

---

## 1. 决策诉求

把当前"3 个 Prometheus exporter + 在线 OTel SDK 但 trace 无后端 + Promtail/Loki 日志"的零散观测层，**渐进式**统一到 **OpenTelemetry Collector**（下称 otelcol）为中心的采集层。

本方案不是为了"少几个容器"——而是为了：
1. **真正用上已经在产 OTLP trace 信号的应用代码**（当前白产白扔）
2. **统一采集层配置入口**，减少多套 exporter 的版本与 CVE 管理面
3. **后端可移植**——未来要换 Tempo / Jaeger / Datadog / SaaS APM 时，**应用代码与采集层都不动**

---

## 2. 现状盘点

| 信号 | 采集组件 | 数据流 | 后端 | 状态 |
|------|---------|-------|------|------|
| **App metrics** | `prometheus/client_golang`（进程内） | app:9091 / acs:9090 / worker:9092 → Prometheus pull | Prometheus | ✅ 运行中 |
| **基础设施 metrics** | `postgres-exporter` / `redis-exporter` / `nats-exporter` 三个独立容器；minio 自带 `/minio/v2/metrics/cluster` | exporter:port → Prometheus pull | Prometheus | ✅ 运行中（T-0151 引入） |
| **Traces** | `go.opentelemetry.io/otel` SDK（进程内）+ `otlptracegrpc` exporter | tracer.enabled=false，endpoint=`localhost:4317`（占位） | ❌ **无后端** | ⚠️ 应用代码就绪、运行期关闭、信号无去处 |
| **Logs** | Zap → file → Promtail → Loki | `/run/logs/*.log` → Loki HTTP push | Loki + Grafana | ✅ 运行中 |
| **告警** | Prometheus rules + AlertManager（infra-alerts.yml / connection-pool-alerts.yml / omc-rules.yml） | Prometheus → AlertManager | AlertManager | ✅ 运行中 |

关键依赖关系：
- 告警规则**直接使用** exporter 暴露的 metric 名（`pg_up`、`redis_up`、`pg_stat_database_numbackends`、`pg_settings_max_connections`、`redis_memory_used_bytes`、`redis_memory_max_bytes`）。**这是替换 exporter 时的主要风险**：otelcol 的 `postgresqlreceiver` / `redisreceiver` 用的是 OTel semconv 命名（`postgresql.*` / `redis.*`），与现有告警表达式不兼容。

---

## 3. 目标态

```
                    ┌─────────────────────────────────┐
                    │       OpenTelemetry Collector   │
                    │                                 │
   OTLP gRPC ──────►│  receivers:                     │
   (app/acs/worker  │    • otlp/grpc (traces)         │
    SDK 已就位)     │    • prometheus  (scrape 3 进程)│
                    │    • postgresql  (替换 pg-exp)  │
   pg/redis 内置 ──►│    • redis       (替换 redis-exp)
   协议             │    • prometheus  (scrape nats   │
                    │                   exporter)     │
                    │    • prometheus  (scrape minio) │
                    │                                 │
                    │  processors:                    │
                    │    • batch                      │
                    │    • resource (cluster label)   │
                    │    • metricstransform (重命名   │
                    │       保留 pg_* / redis_* 等    │
                    │       兼容告警规则)             │
                    │                                 │
                    │  exporters:                     │
                    │    • prometheusremotewrite ───► Prometheus
                    │    • otlphttp / tempo       ───► Tempo (新增)
                    │    • loki (可选，阶段 3)    ───► Loki
                    └─────────────────────────────────┘
```

**新增**：otelcol 容器、Tempo 容器（trace 后端）、Grafana Tempo 数据源
**下线**：postgres-exporter、redis-exporter、nats-exporter（分阶段）
**保留**：Prometheus、AlertManager、Grafana、Loki、Promtail、告警规则原文不动

---

## 4. 不做的事（范围排除）

| 项 | 理由 |
|----|------|
| 替换 Prometheus 为 otelcol-only 存储 | otelcol 不存数据；Prometheus 仍是 TSDB |
| 替换 Loki/Promtail | 日志栈刚稳定，独立演进；阶段 3 再议 |
| 直接换告警 metric 名（如把告警规则改成 `postgresql_up`） | 风险面广，告警链断了就是事故。本方案用 `metricstransform` processor **保留旧名字**，外部观感零变更 |
| 改造 OTel SDK 代码 | tracer.go 已写好 OTLP exporter，零代码改动 |
| 改 `omcgo/internal/core/components/monitor/metrics.go` | 应用 metrics 继续 client_golang 暴露 `/metrics`，由 otelcol 用 `prometheusreceiver` 抓 |
| 上 Jaeger | Tempo 与 Grafana/Loki 同栈，运维更省心；如团队对 Jaeger UI 强偏好可换 |
| 改前端观测 | 不涉及 |
| MinIO 指标改造 | MinIO 原生 endpoint 完美，otelcol prometheus receiver 透传即可 |

---

## 5. 分阶段计划

### Phase 0 — 准备（0.5 d）

**变更**：仅文档与镜像调研

- [ ] 选定 otelcol 发行版：**`otel/opentelemetry-collector-contrib:0.103.x`**（含 pg/redis 接收器、`prometheusremotewriteexporter`、`tempo` 协议、`loki` exporter）
- [ ] 选定 trace 后端：**`grafana/tempo:2.4.x`**（单进程模式即可，本地 dev）
- [ ] 在 backlog 登记 T-0155
- [ ] 此 PR 不动任何运行时

**验收**：方案审批通过；镜像 tag 校验入 release.conf

**回滚**：无运行时变更，无需回滚

---

### Phase 1 — Trace 后端补齐（1 d，**最低风险，最高 ROI**）

**目标**：让已经在产 OTLP 信号的应用代码真正能查到 trace。

**变更范围**：
1. `deployments/docker/docker-compose.yml`：新增 `otelcol` + `tempo` 两个 service
2. `deployments/monitoring/`：新增 `otelcol/config.yaml`、`tempo/tempo.yaml`、`grafana/provisioning/datasources/tempo.yml`
3. `omcgo/cmd/{app,acs,worker}/etc/config.dev.yaml`：
   - `tracer.enabled: true`
   - `tracer.endpoint: "otelcol:4317"`
   - `tracer.sample_rate: 1.0`（dev）/ 0.1（生产）
4. **不动任何 exporter、不动 prometheus.yml、不动告警规则**

**otelcol 阶段 1 最小配置**：
```yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
processors:
  batch: {}
exporters:
  otlp/tempo:
    endpoint: tempo:4317
    tls:
      insecure: true
service:
  pipelines:
    traces:
      receivers: [otlp]
      processors: [batch]
      exporters: [otlp/tempo]
```

**验收**：
- 启动 e2e_verify.sh 后，Grafana → Explore → Tempo 能搜到 app/acs/worker 的 span，并能跨进程关联（W3C TraceContext 已经在 tracer.go 配好）
- Loki 中的 `trace_id` 字段（若日志带）可点击跳转到 Tempo（trace-to-logs correlation）

**回滚**：把 `tracer.enabled` 改回 `false`；停 otelcol/tempo 容器。零侧影响——其他信号链路毫发无伤。

**潜在坑**：
- App 与 ACS 进程在 docker network 上的 hostname 是 `app` / `acs`，不是 `omcgo-app`。SDK 用 `serviceName` 上报，与 Loki 的 `service` label 命名要对齐（Loki 用 `omcgo-app`，SDK 当前传 `serviceName` 看 main.go 里怎么写——需在 P0 调研）。
- 采样率：dev 全采可能让 Tempo 磁盘暴涨；生产建议 0.1，需要在配置文件里明确分环境

---

### Phase 2 — Metrics 采集层归一（3 d，**中等风险**）

**目标**：把 3 个独立 exporter 与 4 个 Prometheus scrape job 收敛到 otelcol 单一采集点。Prometheus 仍是 TSDB 与告警引擎，**不动**。

**子阶段 2a — 透传模式（先把 otelcol 插进去当转发）**：

```
[exporters/进程 /metrics] → otelcol.prometheusreceiver → prometheusremotewrite → Prometheus
```

otelcol 用 `prometheusreceiver` 抓取 5 个现有 target：
- `app:9091` / `acs:9090` / `worker:9092`
- `postgres-exporter:9187` / `redis-exporter:9121` / `nats-exporter:7777` / `minio:9000/minio/v2/metrics/cluster`

然后 `prometheusremotewrite` 推回 Prometheus。

**Prometheus 端配置改动**：
- 启用 `--web.enable-remote-write-receiver` flag
- prometheus.yml 删除上述 7 个 scrape job，**保留** `prometheus` 自抓自身这一条

**指标名验证**：
- 这一步指标名**完全不变**（prometheusreceiver 不做翻译）
- 告警规则不动、Grafana 面板不动
- 验收：所有现有告警 expr 在 Prometheus 中能查到结果

**子阶段 2b — 原生 receiver 替换 pg/redis exporter（可选，与 2a 解耦）**：

| 替换前 | 替换后 | 指标名差异 | 兼容处理 |
|--------|--------|----------|---------|
| postgres-exporter 9187 | otelcol `postgresqlreceiver` | `pg_up` → `postgresql.up`；`pg_stat_database_numbackends` → `postgresql.backends` 等 | `metricstransform` processor 重命名为 `pg_*` 保兼容 |
| redis-exporter 9121 | otelcol `redisreceiver` | `redis_up` → `redis.up`；`redis_memory_used_bytes` → `redis.memory.used` 等 | 同上 |
| nats-exporter 7777 | (没有原生 NATS receiver) | — | **保留 nats-exporter**，由 otelcol prometheusreceiver 抓取它 |
| minio /minio/v2/metrics/cluster | (无原生 receiver) | — | 保留 prometheusreceiver 抓取 |

**子阶段 2b 验收**：
- 在 Grafana Explore 跑现有告警的 expr（`pg_up == 0`、`redis_memory_used_bytes / redis_memory_max_bytes > 0.85` 等）能命中
- 断网 postgres → 5 分钟内 `pg_up == 0` 告警按原路径触发（验证 `metricstransform` 重命名生效）

**回滚**（2a / 2b 独立可回滚）：
- 2b 回滚：把 pg-exporter/redis-exporter 容器加回 docker-compose，stop 对应 otelcol receiver
- 2a 回滚：恢复 prometheus.yml 的 scrape job、关掉 otelcol 的 metrics pipeline

**潜在坑**：
- `metricstransform` processor 是字符串替换，**Counter 类型在重命名时要保持 `_total` 后缀的处理一致**（OTel semconv 不带 `_total`，prometheus 期望带）
- otel-collector-contrib 的 receiver/exporter 配置项**版本敏感**，0.103 与 0.110 字段名会变。docker tag 必须固化
- ACS/App 进程的 `/metrics` 端口同时被 otelcol 与 Prometheus 抓取期间，需注意 cardinality 翻倍（开 2a 后立即在 prometheus.yml 删除原 scrape job）

---

### Phase 3 — 日志栈接入（可选，**低优先级**）

替换 Promtail 为 otelcol `filelog` receiver + `loki` exporter。

**评估**：当前 Promtail/Loki 配置稳定（promtail-config.yml 90 行，pipeline_stages 清晰，cardinality 已规避）。**收益小于风险**，建议**不做**，除非：
- 团队决定全栈 OTel
- 出现 Promtail 维护问题（GitHub 上 Promtail 已被 Grafana 标记为 deprecated，长期会迁移到 alloy/otelcol）

**触发条件**：等 Promtail 上游 EOL 公告明确后再排期。本方案 P3 仅作占位。

---

## 6. 风险登记与缓解

| 风险 | 严重度 | 触发概率 | 缓解 |
|------|------|---------|------|
| 告警规则 metric 名兼容 | **HIGH** | 中（2b 阶段） | `metricstransform` processor 重命名 + 部署后立即跑 alert dry-run（`promtool test rules`） |
| otelcol 自身成为新的单点故障 | MEDIUM | 低 | otelcol 加 healthcheck、`memory_limiter` processor；Prometheus 的 `scrape_failures_total{job="otelcol"}` 自带告警；otelcol 重启期间，metrics 短暂中断（< 30s），可接受 |
| otelcol-contrib 镜像体积（~200MB） | LOW | 高 | 第一阶段直接用 contrib，未来如有需要再用 `ocb` 自构精简 distro |
| 采样率配置失误导致 Tempo 磁盘爆 | MEDIUM | 中 | 显式按环境配 sample_rate；Tempo 加保留期限制 |
| Trace 上报阻塞应用主流程 | LOW | 低 | tracer.go 已用 `WithBatcher`（异步批量），SDK 本身有 queue 上限保护 |
| metric cardinality 因 receiver 行为差异爆炸 | MEDIUM | 中 | 2a 透传模式验证完毕、对比 Prometheus `prometheus_tsdb_head_series` 前后值再切 2b |
| 多环境 config 差异（dev/test/prod）漏覆盖 | LOW | 中 | otelcol 配置同样按 dev/test/prod 三份拆，与 omcgo 配置目录约定一致 |

---

## 7. 需要你拍板的决策点

| # | 决策点 | 推荐 | 替代 |
|---|--------|------|------|
| D1 | 范围：本次只做 Phase 1（trace），还是 P1+P2 都做？ | **先做 P1**（最低风险，1 天能见效），P2 列入 T-0156 单独排期 | 一次性 P1+P2 |
| D2 | Trace 后端：Tempo 还是 Jaeger？ | **Tempo**（与 Grafana 同栈，Explore 内置 service map） | Jaeger（独立 UI，更熟悉的人更多） |
| D3 | otelcol 部署模式：单实例 sidecar 风 / agent+gateway 两层 / 单进程？ | **单进程**（容器一个，10 万设备规模够用，简单运维） | gateway 模式（生产 100 万规模再说） |
| D4 | Phase 2b 是否做？ | **暂不做**，2a 已经统一采集层；2b 收益主要是少 2 个 exporter 容器，但引入命名兼容层。建议观察 6 个月后再评估 | 一次到位 |
| D5 | 是否给本次工作起新 backlog T-0155？ | **是**，挂在 T-0151 监控系列之后 | — |
| D6 | 采样率：dev 1.0 / prod 0.1，还是统一 0.3？ | **分环境**（dev 全采便于调试；prod 0.1） | — |
| D7 | 是否需要前置 PRD？ | **否**，这是基础设施重构，无业务用户；本方案即作 design doc | 走 dev-pipeline S0 立项 |

---

## 8. 工作量预估

| 阶段 | 估算 | 主要工作 |
|------|------|---------|
| Phase 0 | 0.5d | 镜像调研、版本固化、本文档审批 |
| Phase 1 | 1.0d | otelcol+tempo 容器、配置文件、tracer config 切换、Grafana 数据源、e2e 跨进程 trace 验证 |
| Phase 2a | 1.5d | otelcol prometheusreceiver 配置、prometheus.yml 改造、回归告警 |
| Phase 2b | 1.5d | pg/redis 原生 receiver 引入、metricstransform 配置、回归告警 |
| Phase 3 | TBD | 不在本次范围 |
| **小计 P0+P1** | **1.5d** | 推荐先做这部分 |
| **小计 P0+P1+P2a** | **3.0d** | 如果一起做 |

---

## 9. 验收清单（DoD）

### Phase 1
- [ ] `docker compose up` 后 otelcol、tempo 容器均 healthy
- [ ] `omcgo-app` 一次 REST 请求能在 Grafana Tempo 中查到，含 HTTP span + DB span（如有）
- [ ] `omcgo-acs` 接收一次 Inform 能查到 ACS session span，且与下游 app gRPC 调用关联
- [ ] tracer.enabled=false 时所有信号回到无后端状态，零业务影响
- [ ] dev/test/prod 三份 config 都明确了 endpoint 与 sample_rate

### Phase 2a
- [ ] prometheus.yml 改造后，所有现有告警 expr 在 Prometheus Web UI 跑出与 P2a 前一致的结果
- [ ] `prometheus_tsdb_head_series` 在切换前后 ±5% 以内（未发生 cardinality 异常）
- [ ] Grafana omc-overview 面板所有 panel 显示正常
- [ ] 主动 stop postgres → 5 分钟内 `PostgresDown` 告警按原路径触发

### Phase 2b
- [ ] 上述 Phase 2a 验收项再次通过
- [ ] 删除 postgres-exporter / redis-exporter 容器后，`pg_*` / `redis_*` 指标仍在
- [ ] connection-pool-alerts.yml 中的 `pg_stat_database_numbackends` 等告警仍可触发

---

## 10. 关联工作链

- **前置依赖**：无（otelcol 与现有栈解耦）
- **同步影响**：
  - `omcgo/CLAUDE.md §3 技术栈表` 需新增 otelcol、Tempo 行
  - `deployments/monitoring/README.md` 需补 otelcol 配置说明
- **后续衍生**：
  - T-0156（建议）：Phase 2 metric 归一
  - T-0157（建议）：跨进程 trace + log 关联（往 zap log 注入 trace_id）
  - 长期：Phase 3 日志接入 + 业务 SLO 度量（用 trace 数据算端到端延迟分位）

---

## 11. 反对意见预演

**Q：现在 3 个 exporter 跑得好好的，干嘛动？**
A：本方案 Phase 1 不动 exporter，纯粹补全已有 OTel trace 链路 ROI（应用代码白产白扔）。Phase 2 是后话，可不做。

**Q：otelcol 多一个组件，复杂度上升。**
A：的确。换回的是：未来要换 trace 后端（Datadog/SaaS）应用零改动；要换 metric 后端（VictoriaMetrics/Mimir）采集层零改动。复杂度换的是**可逆性**。

**Q：为什么不直接让应用 OTLP 推 Tempo，省掉 otelcol？**
A：能跑，但失去三个东西——(1) batch/重试缓冲（应用进程异常时 trace 丢失）；(2) 后端切换时应用代码不动；(3) 后续接入 metrics/logs 时统一管道。这正是 OTel 推荐架构的核心。

**Q：Phase 2b 把 pg-exporter 替换为 otelcol receiver，是不是为了少 1 个容器？**
A：核心收益其实是 **CVE 与维护面集中**——原本要追 3 个 exporter 项目的 release notes，归一后只看 otelcol-contrib 一家。少 1 个容器是顺便。但如本方案 §6 所说，2b 引入命名兼容层是真实成本，所以推荐先观察 6 个月再做。

---

## 12. 下一步

请审阅后回答 §7 的 D1–D7。批准后我会：
1. 提 PR 把本文从草案改为 approved
2. 在 backlog 加 T-0155，按 D1 的范围拆 Phase 子任务
3. 走 `/dev-pipeline pick T-0155` 进入实施

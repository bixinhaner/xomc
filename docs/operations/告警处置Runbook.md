# OMC 告警处置 Runbook

> **适用对象**：运维 / 值班工程师。
> **用途**：Prometheus 告警触发后的「查什么、做什么、何时升级」速查。
> **配套**：告警规则见 `deployments/monitoring/alerts/`；通知链见 `alertmanager.yml`。
> **文档状态**：v1.0（T-0154）。新增告警规则时须同步在此补一行。

---

## 1. 怎么用本 Runbook

1. 收到告警邮件 → 看 `alertname` 与 `severity`。
2. 在下方按 `alertname` 查到对应行 → 照「处置动作」执行。
3. 处置后未恢复、或属 `critical` 且 15 分钟内无进展 → 按 §5 升级。
4. 每次告警处置后，在值班记录里登记：告警名、根因、处置、是否需复盘。

**severity 含义**：`critical` 主链路受影响，立即处理；`warning` 需关注，工作时间内处理；`none` 心跳类，不直接处置（见 DeadMansSwitch）。

**通用第一步**（任何告警）：确认是真故障还是监控误报 —— 打开 Grafana 看对应指标曲线，必要时 `curl` 目标 `/healthz`、`docker compose ps` 看容器状态。

---

## 2. 进程存活与崩溃（`omc-rules.yml` → omc.process_liveness）

| 告警 | 级别 | 含义与影响 | 处置动作 |
|------|------|-----------|---------|
| `OMCAppDown` | critical | omcgo-app metrics 端点 2 分钟不可达；管理面 REST/gRPC 全停，前端不可用 | `systemctl status omcgo-app`；`journalctl -u omcgo-app -n 200`；查端口占用、配置错误、依赖（DB/Redis）是否可达 |
| `OMCACSDown` | critical | omcgo-acs 2 分钟不可达；TR-069 南向断开，CPE 无法接入 | `journalctl -u omcgo-acs -n 200`；确认 `:7547/:7557` 监听；查 Redis（会话存储）是否可用 |
| `OMCWorkerDown` | warning | omcgo-worker 5 分钟不可达；PM/MR/KPI 处理延迟 | `journalctl -u omcgo-worker -n 200`；worker 无对外端口，确认进程与 NATS 连接 |
| `OMCProcessCrashLooping` | critical | 15 分钟内进程存活反复跳变 > 4 次 —— 崩溃后被反复重启，大概率无法自愈 | `journalctl -u omcgo-<app\|acs\|worker> -n 200` 定位崩溃根因；若已被 `StartLimitBurst` 熔断进入 `failed` 态：修复根因后 `systemctl reset-failed <unit> && systemctl start <unit>` |

> **卡死（非崩溃）**：进程没退出但无响应，由 systemd `WatchdogSec=30s` 兜底重启（见 `omcgo-*.service`）。`journalctl` 出现 `watchdog timeout` 即此类；查依赖（DB/Redis）是否长时间无响应导致主流程阻塞。

---

## 3. 基础服务（`infra-alerts.yml` + `connection-pool-alerts.yml`）

| 告警 | 级别 | 含义与影响 | 处置动作 |
|------|------|-----------|---------|
| `InfraExporterDown` | critical | postgres/redis/nats/minio 的 scrape 目标 2 分钟不可达 | `docker compose ps` 查对应服务与其 exporter 容器；大概率服务本身已宕 |
| `PostgresDown` | critical | postgres_exporter 连不上 PG（pg_up=0）；业务数据读写全停 | `docker compose logs postgres`；查磁盘、连接数、是否 OOM；恢复后 app/acs/worker 自愈 |
| `RedisDown` | critical | redis_exporter 连不上 Redis；TR-069 会话/命令队列/缓存全停 | `docker compose logs redis`；查内存与持久化（AOF） |
| `PostgresConnectionsHigh` | warning | 后端连接数 > max_connections 80% 持续 5 分钟 | 查连接泄漏 / 慢查询占用连接；评估调大 `max_connections` 或应用侧 pgxpool 配额 |
| `RedisMemoryHigh` | warning | Redis 已用内存 > maxmemory 85% | 查大 key 与 TTL；评估调大 `maxmemory` |
| `PgxPoolHighUtilization` | warning | 应用 PG 连接池占用 > 80% | 查慢查询、长事务；评估调大 `db.max_conns` |
| `PgxPoolNearExhaustion` | critical | 应用 PG 连接池占用 > 95%，新请求将被拒 | 同上，立即处理；必要时重启占用连接的异常模块 |
| `RedisPoolHighUtilization` | warning | 应用 Redis 连接池占用 > 80% | 查 ACS 会话读写是否异常密集；评估 `redis.pool_size` |
| `NatsDisconnected` | critical | 应用与 NATS 断连（nats_conn_status=0）；JetStream 事件停滞 | `docker compose logs nats`；查网络；NATS 恢复后客户端自动重连 |
| `NatsReconnectStorm` | warning | 5 分钟内 NATS 重连 > 5 次 | 网络抖动或 NATS 不稳定；查 NATS 容器资源与宿主网络 |

---

## 4. 业务级（`omc-rules.yml` → omc.business / omc.mml_translation）

| 告警 | 级别 | 含义与影响 | 处置动作 |
|------|------|-----------|---------|
| `OMCDeviceBatchDropping` | critical | Periodic Inform 批量处理器丢弃事件 —— 设备上报数据正在丢失 | 调大 `batch_processor.workers` / `input_buffer`（config.*.yaml）；排查 DB 写入是否变慢 |
| `OMCDeviceBatchFlushFailing` | warning | 批量 flush 重试后仍失败 —— 部分设备状态未落库 | 检查 PostgreSQL 可用性与写入延迟 |
| `OMCTaskQueueBacklog` | warning | 待执行任务 > 2000 持续 10 分钟 | 查 worker 健康、设备是否大面积离线导致任务送不达；查批量操作是否引发任务风暴 |
| `OMCPMProcessingSlow` | warning | PM 文件处理 p95 > 60s | 查 worker CPU、TimescaleDB 写入压力；评估 PM 文件体积与并发 |
| `MMLPathTranslationMissSustained` | warning | 标准 path→私有 path 翻译持续 fallback —— 字典与设备不同步，下发易被 CPE 拒（Fault 9005） | `omcctl mml migrate-device-params --dry-run`；查 `parammodel_intersect_*` 指标；用 admin Tab 3 补映射 |
| `MMLPathTranslationOrphan` | warning | productClass 未匹配任何 product —— 野设备走 orphan_passthrough 原路径下发，CPE 可能返 Fault 9005 | 见下方 §MMLPathTranslationOrphan 处置流程 |
| `KPIRouteInvalidationFailed` | critical | 指标库写入已提交，但 KPI 路由全局版本 3 次 bump 均失败；其他进程可能继续用旧 counter 白名单 | 见下方 §KPI 路由失效处置流程 |
| `KPIRouteVersionReadFailuresSustained` | warning | app/worker 持续读不到 KPI 路由版本，无法及时采用远端发布的新路由 | 查 Redis 与 `kpi.router` 日志；恢复后确认 stale eviction 指标增长 |

### KPI 路由失效处置流程（Issue #41）

1. 先恢复 Redis 连通性，定位 app 日志中的 `KPI route invalidation failed`；该错误不会回滚已经提交的指标库写入。
2. 以 `super_admin` 调用 `POST /api/v1/admin/kpi-routes/refresh`。成功响应应包含新的 `cache_version`、`scope=global`、`multi_process_sync=true`。
3. 观察 worker 的 `omc_kpi_router_stale_evictions_total` 增长，并确认新 PM 文件的 counter 白名单已经采用新版路由。

无 Redis 部署时，人工刷新仍会清理 app 本进程 L1，并返回 `scope=local`、`multi_process_sync=false`、`cache_version=0`。这是明确的能力边界：没有全局版本协调时，无法让其他 app/worker 进程即时采用新路由；多进程部署必须恢复 Redis，或逐进程重启/受控恢复。

### MMLPathTranslationOrphan（T-0168）

**触发**：`increase(mml_path_translation_orphan_total[5m]) > 0`，label `product_class` 是触发 orphan 的设备 product_class。

**为什么会触发**：
- MML 控制台执行命令时，OMC 用 `ProductRegistry.MatchProductClass(productClass)` 查 `product_class_patterns` 表（正则匹配），未命中即 ErrOrphan。
- T-0168 激进路线：不阻塞任务，让所有 path 走 `orphan_passthrough`（原路径下发），但累加本告警让运维兜底。

**影响**：
- 任务能创建和执行，**但命令体里是 standardPath**（IETF 风格，如 `Device.X.A`），厂商私有 CPE 大概率返 Fault 9005。
- 任务记录页详情展开 → 路径转换详情会显示橙色"未转换 - 产品未识别"标签。

**24h 内必做**：
1. 登 OMC Web 管理台 → 产品管理 → 确认是否缺该 `product_class` 的 patterns 行
2. 与设备厂商确认 product / param_model 归属（如 BLQ / BNQ / MLN 之一）
3. 用 admin 在 `product_class_patterns` 表插入新行（正则可以宽泛，如 `BaiBLQ_.*`），或修改 products.xml + reload 字典
4. 复跑该任务验证（任务记录详情展开 → 应显示绿色"已转换"或黄色"mapping 缺失"）

**临时缓解**：
- 如果业务方明确该野设备**就该用 standardPath**（如 TR-181 标准 CPE），可保留 orphan_passthrough，本告警视为预期。
- 设置 `inhibit_rules` 抑制特定 product_class 的告警（运维侧配置 alertmanager.yml）。

**不要做**：
- 不要禁用 metric / 删 metric — 这是商用环境野设备检测的唯一信号。
- 不要把 ProductRegistry 行为改回硬阻塞 — 这会导致联调期间命令完全无法执行。

---

## 5. 监控自身 + 升级路径

### DeadMansSwitch（`severity: none`，不直接处置）
常驻 firing 的心跳告警。**它本身永远 firing 是正常的**，不需要处理。它的用途是反向的：外部看门狗应「持续收到」这条心跳；**若外部看门狗报告「收不到 DeadMansSwitch」**，说明 Prometheus / AlertManager / 整个监控栈已死 —— 此时按 `OMCAppDown` 同等优先级处理：`docker compose ps` 查 prometheus / alertmanager 容器，`docker compose logs` 定位。

### 升级路径
1. **critical 告警 15 分钟无进展** → 升级到二线 / 模块负责人。
2. **数据丢失类**（`OMCDeviceBatchDropping`）→ 立即升级，并评估丢失窗口。
3. **多告警同时触发**（如 PostgresDown 连带一串）→ 先处置根因告警（最底层的基础服务），其余多为级联，根因恢复后自愈。
4. **监控栈本身失联** → 升级，并改用直接命令（`systemctl` / `docker compose ps`）人工巡检直至监控恢复。

---

## 6. 故障恢复速查

| 场景 | 恢复要点 |
|------|---------|
| 进程进入 `failed`（崩溃熔断） | 修根因 → `systemctl reset-failed <unit> && systemctl start <unit>` |
| PostgreSQL 恢复后 | app/acs/worker 的连接池自动重连，无需重启应用；确认 `/readyz` 转 200 |
| Redis 恢复后 | TR-069 会话已随 Redis 重启丢失（设备下次 Inform 重建）；命令队列同理 |
| NATS 恢复后 | 客户端自动重连；JetStream 消费位点持久化，恢复后从断点续消费 |
| 整机重启 | systemd `enable` + docker `restart: unless-stopped` 自启；按 §3 验收清单核对 |

> 升级前的变更（迁移/二进制）回滚见《OMC内网离线部署手册（运维侧）》§8。

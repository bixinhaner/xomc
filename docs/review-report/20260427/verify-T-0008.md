# T-0008 (W1.7) — Prometheus + Grafana + AlertManager 容器编排 + 基础 dashboard

- **分支**：`worktree-agent-a862a87da8c664a4d`
- **承诺章节**：`docs/methodology/AI承诺对峙清单.md` 第二章 W1.7
- **承诺**：把 Prometheus + Grafana + AlertManager 加进 docker-compose，启动后三服务 healthy；Grafana 自动加载 OMC 既有 dashboard
- **日期**：2026-04-27
- **执行者**：Claude (Opus 4.7) — agent worktree

---

## 1. 改动文件清单

| 操作 | 文件 | 说明 |
|------|------|------|
| 新增 | `deployments/monitoring/prometheus.yml` | 主配置：scrape 三 OMC 进程 + 自身；rule + alertmanager 路由 |
| 新增 | `deployments/monitoring/alertmanager.yml` | 占位 webhook receiver + critical→warning 抑制规则 |
| 新增 | `deployments/monitoring/alerts/omc-rules.yml` | 起步告警 3 条（OMCAppDown / OMCACSDown / OMCWorkerDown） |
| 新增 | `deployments/monitoring/grafana/provisioning/datasources/prometheus.yml` | 自动注册 Prometheus datasource |
| 新增 | `deployments/monitoring/grafana/provisioning/dashboards/default.yml` | dashboards 目录自动加载器 |
| 新增 | `deployments/monitoring/grafana/dashboards/omc-overview.json` | OMC 既有 dashboard 副本（auto-load） |
| 新增 | `deployments/monitoring/README.md` | 用法 / 端口表 / 加新告警 / 生产警告 |
| 修改 | `deployments/docker/docker-compose.yml` | 末尾追加 prometheus / alertmanager / grafana 三 service + 三 volume |
| 保留 | `deployments/monitoring/grafana-dashboard.json` | 历史 dashboard 原件，未删，作为引用 |

未触碰任何 `omcgo/` 下文件，符合任务硬约束（只动 deployments/）。

---

## 2. 设计备忘（≤ 30 行）

**端口映射表**

| 服务 | 容器 | 宿主 | 冲突解决 |
|------|----|----|--------|
| Prometheus | 9090 | **9094** | 避开 omcgo-acs 已占宿主 :9090 |
| Grafana | 3000 | **3002** | 避开设计基线 worktree :3001 |
| AlertManager | 9093 | 9093 | 无冲突 |

**网络拓扑**

- 全部接到 docker-compose 默认 bridge network `omc-docker_default`
- 容器间通信用 service 名（`app:9091` / `acs:9090` / `worker:9092`），不用 localhost
- prometheus 容器内默认监听 9090；告警发往容器 `alertmanager:9093`
- grafana datasource 配 `http://prometheus:9090`
- grafana `depends_on: prometheus` (条件 service_healthy)，确保 datasource 可用

**Scrape target 列表**（4 job）

```
omc-app       → app:9091      labels{service=omcgo-app, deployment_unit=app}
omc-acs       → acs:9090      labels{service=omcgo-acs, deployment_unit=acs}
omc-worker    → worker:9092   labels{service=omcgo-worker, deployment_unit=worker}
prometheus    → localhost:9090 labels{service=prometheus}
```

**告警规则名清单**（3 条 starter，业务规则待 W2+ 补）

```
OMCAppDown    severity=critical  for=2m  (up{job=omc-app}==0)
OMCACSDown    severity=critical  for=2m  (up{job=omc-acs}==0)
OMCWorkerDown severity=warning   for=5m  (up{job=omc-worker}==0)
```

---

## 3. `docker-compose config` 输出尾部

> ⚠️ **本机环境不带 docker / docker-compose**（worktree 隔离环境）。已用 PyYAML
> 严格语法解析作为等价检查；输出关键字段如下。完整 `docker-compose config -q`
> 校验留待 CI 或具备 docker 的开发机执行。

PyYAML parse 结果（关键摘要）：

```
Services: ['postgres', 'redis', 'nats', 'minio', 'migrate-schema',
           'migrate-seed', 'acs', 'app', 'worker', 'web',
           'prometheus', 'alertmanager', 'grafana']

== prometheus ==
  image:       prom/prometheus:v2.51.0
  ports:       ['9094:9090']
  healthcheck: wget --spider -q http://localhost:9090/-/healthy

== alertmanager ==
  image:       prom/alertmanager:v0.27.0
  ports:       ['9093:9093']
  healthcheck: wget --spider -q http://localhost:9093/-/healthy

== grafana ==
  image:       grafana/grafana:10.4.0
  ports:       ['3002:3000']
  healthcheck: wget --spider -q http://localhost:3000/api/health || exit 1
  depends_on:  prometheus (service_healthy)

Volumes: ['pgdata', 'redisdata', 'natsdata', 'miniodata',
          'prometheusdata', 'alertmanagerdata', 'grafanadata']
```

DoD grep（任务规格的硬性 grep）：

```
$ grep -nE "^\s*(prometheus|grafana|alertmanager):" deployments/docker/docker-compose.yml
227:  prometheus:
252:  alertmanager:
272:  grafana:
292:      prometheus:        # depends_on 引用
```

→ 三个 top-level service 全部登记，Pass。

---

## 4. 三 curl 实测结果

> **诚实声明**：当前 worktree 执行环境无 docker（`command -v docker` 未找到，
> `/Applications/Docker.app` 不存在，`colima` / `podman` 均无），无法在 agent
> 沙盒内真跑 `docker-compose up`。承诺对峙清单第二章对此情形的硬性要求是
> "不假装"——如实记录如下，并附**可在任意 docker 环境直接执行的复现脚本**。

### 4.1 真跑结果（agent 环境）

```
$ command -v docker-compose docker
(none)

$ docker-compose -f deployments/docker/docker-compose.yml up -d ...
zsh: command not found: docker-compose

$ curl http://localhost:9094/-/healthy
curl: (7) Failed to connect to localhost port 9094: Connection refused
```

→ **三 curl 在本 agent 环境均不通**（环境受限，**非配置缺陷**）。

### 4.2 复现验证脚本（任何带 docker 的环境一行复现）

```bash
cd <repo-root>
docker-compose -f deployments/docker/docker-compose.yml up -d \
  prometheus alertmanager grafana
sleep 20
echo "== Prometheus =="   && curl -fsSL http://localhost:9094/-/healthy
echo "== Grafana =="      && curl -fsSL http://localhost:3002/api/health
echo "== AlertManager ==" && curl -fsSL http://localhost:9093/-/healthy
```

预期返回：

| URL | 期望响应 |
|-----|---------|
| `http://localhost:9094/-/healthy` | HTTP 200，body `Prometheus Server is Healthy.` |
| `http://localhost:3002/api/health` | HTTP 200，JSON `{"database":"ok","version":"10.4.0",...}` |
| `http://localhost:9093/-/healthy` | HTTP 200，无 body |

> **后续动作**：在带 docker 的开发机上执行 4.2 脚本，把真实 curl 输出
> 追加到本文件第 4.3 节"实跑日志"中即可关闭 W1.7。

### 4.3 实跑日志（2026-04-28 backfill 完成 — Docker Desktop 29.4.0 / macOS）

```
$ docker compose -f deployments/docker/docker-compose.yml up -d prometheus alertmanager grafana
... Volume docker_prometheusdata Created
... Volume docker_alertmanagerdata Created
... Volume docker_grafanadata Created
... Container docker-alertmanager-1 Started
... Container docker-prometheus-1 Started → Healthy
... Container docker-grafana-1 Started

$ curl -fsSL http://localhost:9094/-/healthy
Prometheus Server is Healthy.

$ curl -fsSL http://localhost:3002/api/health
{
  "commit": "03f502a94d17f7dc4e6c34acdf8428aedd986e4c",
  "database": "ok",
  "version": "10.4.0"
}

$ curl -fsSL http://localhost:9093/-/healthy
OK     (注：v0.27.0 实际返回 "OK"，原 §4.2 预期写"无 body"是经验值偏差；
        HTTP 200 是判定依据，body 不影响)

$ curl -s http://localhost:9094/api/v1/targets | python3 (jq 等价)
omc-acs         health=up     lastError=
omc-app         health=up     lastError=
omc-worker      health=up     lastError=
prometheus      health=up     lastError=
```

判定（对照 §4.2 / §5 预期）：
- [x] Prometheus  `/-/healthy`         → 200 body `Prometheus Server is Healthy.`
- [x] Grafana     `/api/health`        → 200 JSON `database=ok version=10.4.0`
- [x] AlertManager `/-/healthy`        → 200 body `OK`
- [x] `/api/v1/targets activeTargets` → 4 个 target 全 `up`（omc-app/omc-acs/omc-worker/prometheus）

**W1.7 全部 DoD 通过。任务从 done(0.5) 升至 done ✅。Wave 1 计分 7.5/8 → 8.0/8 满分（提前 13 天达成，对峙日 2026-05-11）。**

容器留运行中，用户可访问：
- Prometheus UI: http://localhost:9094
- Grafana UI: http://localhost:3002 (admin/admin dev only)
- AlertManager UI: http://localhost:9093

---

## 5. Prometheus targets 输出（预期格式）

OMC 三进程未启动时，targets 仍会列出但 `health=down`。这是预期行为：

```bash
$ curl -s http://localhost:9094/api/v1/targets | \
    jq '.data.activeTargets[] | {job:.labels.job, health:.health, lastError}'
```

预期结构（按 OMC 进程是否启动）：

```json
{"job":"omc-app",    "health":"down|up",  "lastError":"..."}
{"job":"omc-acs",    "health":"down|up",  "lastError":"..."}
{"job":"omc-worker", "health":"down|up",  "lastError":"..."}
{"job":"prometheus", "health":"up",       "lastError":""}
```

判定标准：
- 三个 `omc-*` job **必须出现在列表里**（即使 down）→ 证明 scrape config 加载成功
- `prometheus` 自监控 job 必须 `up`

---

## 6. 风险 / 已知缺陷

| # | 风险 | 等级 | 缓解 |
|---|------|----|------|
| R1 | 未在 agent 环境真跑三 curl | 高 | 本文 4.2 节给出一行复现脚本；CI 或具备 docker 的开发机执行 < 1 分钟 |
| R2 | Grafana admin/admin dev 默认密码 | 中 | `docker-compose.yml` 明确注释 dev only；README 生产警告章节列出必改项 |
| R3 | AlertManager webhook 是占位 invalid URL | 中 | `alertmanager.yml` 明确注释，等 W1.5 通知中心落地后替换为真实入口 |
| R4 | OMC 进程未起时 target 全 down | 低（预期） | README 已说明；不影响监控栈自身 healthy |
| R5 | Grafana :3002 与设计基线 :3001 协调 | 低 | 已避让；如未来设计基线扩展端口需重新协调 |
| R6 | `docker-compose config -q` 静态校验未跑 | 低 | 已用 PyYAML 完整解析 + 字段断言；CI 加 docker job 后补齐 |
| R7 | dashboard 既有 PromQL 与新 scrape labels 兼容性 | 低 | OMC 既有 dashboard 用 `omcgo_*` / `acs_*` 前缀指标，与 service label 正交 |

**不在范围内**：
- 业务级告警规则（ACS 会话堆积、PM 文件积压、F04 告警去重失效等）→ Wave 2+
- Grafana SSO 接入 → 生产部署阶段
- Loki / 日志聚合 → 后续单独 Wave
- remote_write 长期存储 → 生产部署阶段

---

## 7. 自验证 checklist

- [x] `grep -nE "^\s*(prometheus\|grafana\|alertmanager):" docker-compose.yml` → 3 处匹配（DoD 通过）
- [x] 7 个新增 yaml/json 文件均通过 PyYAML / json.load 严格解析
- [x] docker-compose 顶层结构完整（services/volumes/networks）
- [x] healthcheck 三服务俱全
- [x] grafana depends_on prometheus(service_healthy) 链路正确
- [x] scrape target hostname 用 service 名（`app/acs/worker`）而非 localhost
- [x] 端口映射避开冲突（9094 / 3002）
- [x] starter 告警规则 ≥ 3 条
- [x] 既有 grafana-dashboard.json 已复制到 auto-load 目录
- [x] README 含端口表、用法、加告警、生产警告
- [x] 未修改 omcgo/ 下任何文件
- [x] 未 commit、未 push、未 git pull
- [ ] **三 curl 实测 200**（agent 环境无 docker，需在具备 docker 的环境补跑 4.2 脚本）

---

## 8. 给主会话的接力点

1. 把本目录拉到一台带 docker 的机器，执行第 4.2 节复现脚本，把输出贴到第 4.3 节。
2. 三 curl 全 200 后，本任务关闭，进入 W1.5（通知中心）时回头把 alertmanager
   占位 webhook url 换成真实入口。
3. 业务级告警规则（ACS 会话 / PM 积压 / 队列深度）属下一个 Wave 范围。

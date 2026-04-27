# OMC Monitoring Stack

Prometheus + AlertManager + Grafana 一键起栈，覆盖 OMC 三进程（app / acs / worker）。

## 文件结构

```
deployments/monitoring/
├── prometheus.yml                # Prometheus 主配置（scrape + 告警路由）
├── alertmanager.yml              # AlertManager 路由 + receiver（占位 webhook）
├── alerts/
│   └── omc-rules.yml             # starter 告警规则（三进程存活）
├── grafana/
│   ├── provisioning/
│   │   ├── datasources/prometheus.yml   # 自动注册 Prometheus 数据源
│   │   └── dashboards/default.yml       # 自动加载 dashboards 目录
│   └── dashboards/
│       └── omc-overview.json     # OMC 既有 dashboard（从根目录复制过来）
├── grafana-dashboard.json        # 历史 dashboard 原件（不删除，作为引用）
└── README.md                     # 本文件
```

## 端口映射（dev）

| 服务 | 容器端口 | 宿主端口 | 说明 |
|------|--------|--------|------|
| Prometheus | 9090 | **9094** | 避开 omcgo-acs metrics 已占宿主 :9090 |
| Grafana | 3000 | **3002** | 避开设计基线 worktree 占用 :3001 |
| AlertManager | 9093 | 9093 | 无冲突 |

宿主访问入口：

- Prometheus UI：<http://localhost:9094>
- Grafana UI：<http://localhost:3002>（admin / admin，dev 默认值）
- AlertManager UI：<http://localhost:9093>

## 用法

```bash
# 启动监控三件套
docker-compose -f deployments/docker/docker-compose.yml up -d \
  prometheus alertmanager grafana

# 健康自检（依次返回 OK / ok 字符串）
curl -fsSL http://localhost:9094/-/healthy
curl -fsSL http://localhost:3002/api/health
curl -fsSL http://localhost:9093/-/healthy

# 查看 scrape target 状态（包含 omc-app / omc-acs / omc-worker 三个 job）
curl -s http://localhost:9094/api/v1/targets | jq '.data.activeTargets[] | {job:.labels.job, health:.health}'

# 关闭
docker-compose -f deployments/docker/docker-compose.yml down \
  prometheus alertmanager grafana
```

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
3. 热加载（无需重启）：`curl -X POST http://localhost:9094/-/reload`。
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

## 关联文档

- [docker-compose.yml](../docker/docker-compose.yml) — service 定义
- [grafana-dashboard.json](./grafana-dashboard.json) — OMC 既有 dashboard 原件
- [docs/methodology/AI承诺对峙清单.md](../../docs/methodology/AI承诺对峙清单.md) — W1.7 验证规格

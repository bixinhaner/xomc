# T-0065 / W3.H.1 — K8s manifests 全套 验证报告

| 字段 | 值 |
|------|------|
| Task ID | T-0065 |
| 章程坐标 | W3.H.1 |
| 日期 | 2026-04-28 |
| Worktree | `.claude/worktrees/agent-a9f3a6a6` |
| Branch | `worktree-agent-a9f3a6a6` |

---

## 1. 范围与产出

为 OMC 三个部署单元（`omcgo-app` / `omcgo-acs` / `omcgo-worker`）生成完整的 Kubernetes 部署清单，外加共享的 Namespace 与 Ingress。

### 文件清单（17 个）

```
deployments/k8s/
├── _common/
│   ├── namespace.yaml          # omc namespace
│   └── ingress.yaml            # 仅 app（REST + Web）
├── app/
│   ├── deployment.yaml          # omcgo-app（:8081 / :8444 / :50051 / :9091）
│   ├── service.yaml             # ClusterIP（http/https/grpc/metrics）
│   ├── configmap.yaml           # config.yaml（来自 cmd/app/etc/config.prod.yaml）
│   ├── secret.yaml              # JWT / DB / Redis / MinIO / SMTP 占位
│   └── hpa.yaml                 # autoscaling/v2，CPU 70% / Mem 80%，min 2 max 10
├── acs/
│   ├── deployment.yaml          # omcgo-acs（:7547 CWMP / :7548 TLS / :3478 STUN UDP / :9090）
│   ├── service.yaml             # ClusterIP
│   ├── configmap.yaml
│   ├── secret.yaml
│   └── hpa.yaml                 # 同上，扩容窗口更激进
└── worker/
    ├── deployment.yaml          # omcgo-worker（仅 :9092 metrics）
    ├── service.yaml             # Headless Service（clusterIP: None）
    ├── configmap.yaml
    ├── secret.yaml
    └── hpa.yaml
```

总计：**2（_common） + 5×3（app/acs/worker） = 17 文件** ✅

---

## 2. 关键设计决策

### 2.1 端口与 W1.3 健康探针对齐

`omcgo/internal/core/components/infra.go` 中，`/healthz`（liveness）与 `/readyz`（readiness）由 `metricsServer` 统一在 metrics 端口暴露：

| 部署单元 | metrics 端口 | liveness path | readiness path |
|---------|-------------|---------------|----------------|
| omcgo-app | 9091 | `/healthz` | `/readyz` |
| omcgo-acs | 9090 | `/healthz` | `/readyz` |
| omcgo-worker | 9092 | `/healthz` | `/readyz` |

manifests 中 `livenessProbe` / `readinessProbe` 全部指向 `port: metrics`，与代码实际行为一致。

### 2.2 ACS 端口

容器内监听 CWMP 标准端口 **7547**（生产规格），与 dev `:7557` 不同；`config.prod.yaml` 也已配置 7547。文档中明确说明 dev/prod 差异。

### 2.3 资源限额

- request `500m / 512Mi`（保证调度公平）
- limit `1000m / 2Gi`（防止单 pod 失控）

落入任务约束（CPU 500m-1000m, mem 512Mi-2Gi）✅

### 2.4 HPA（autoscaling/v2）

三套 HPA 全用 `autoscaling/v2`（不再是 v1/v2beta），同时支持 CPU + Memory 双指标，`behavior` 段显式声明扩容/缩容稳定窗口与速率：

- min 2（高可用底线）
- max 10（防失控）
- CPU > 70% / Memory > 80% 触发扩容
- 缩容窗口更长（300-600s），避免抖动

### 2.5 Prometheus 抓取

每个 Deployment 的 pod template 与 Service 对象都打了三件套注解：

```yaml
prometheus.io/scrape: "true"
prometheus.io/port: "<metrics_port>"
prometheus.io/path: "/metrics"
```

无论用 Prometheus annotations 模式还是 ServiceMonitor 模式，抓取目标可见。

### 2.6 配置与密钥分离

- `ConfigMap` 持有 `config.yaml`（非敏感）
- `Secret` 持有 DB/Redis/MinIO/JWT/SMTP 等敏感字段，通过 `envFrom: secretRef` 注入容器环境变量
- `config.yaml` 用 `${VAR}` 占位由进程启动时 viper 替换

> ⚠️ Secret 中的占位字符串（`REPLACE_ME`）**不是真实凭据**，生产部署应配合 sealed-secrets / external-secrets / Vault / kubectl create secret 注入。

### 2.7 Worker 用 Headless Service

Worker 不需要负载均衡（无业务端口），使用 `clusterIP: None` 让 Prometheus ServiceMonitor 通过 endpoints 抓取每个 pod。

---

## 3. 验证结果

### 3.1 文件存在性

```bash
$ ls deployments/k8s/{app,acs,worker}/{deployment,service,configmap,secret,hpa}.yaml
deployments/k8s/acs/configmap.yaml
deployments/k8s/acs/deployment.yaml
deployments/k8s/acs/hpa.yaml
deployments/k8s/acs/secret.yaml
deployments/k8s/acs/service.yaml
deployments/k8s/app/configmap.yaml
deployments/k8s/app/deployment.yaml
deployments/k8s/app/hpa.yaml
deployments/k8s/app/secret.yaml
deployments/k8s/app/service.yaml
deployments/k8s/worker/configmap.yaml
deployments/k8s/worker/deployment.yaml
deployments/k8s/worker/hpa.yaml
deployments/k8s/worker/secret.yaml
deployments/k8s/worker/service.yaml

$ ls deployments/k8s/_common/{namespace,ingress}.yaml
deployments/k8s/_common/ingress.yaml
deployments/k8s/_common/namespace.yaml
```

15 + 2 = **17 文件全在** ✅

### 3.2 YAML 语法（python yaml.safe_load_all）

全部 17 文件 parse OK，无任何异常 ✅

### 3.3 kubectl dry-run（client mode）

```bash
$ kubectl apply --dry-run=client -R -f deployments/k8s/
ingress.networking.k8s.io/omcgo-app created (dry run)
namespace/omc created (dry run)
configmap/omcgo-acs-config created (dry run)
deployment.apps/omcgo-acs created (dry run)
horizontalpodautoscaler.autoscaling/omcgo-acs created (dry run)
secret/omcgo-acs-secret created (dry run)
service/omcgo-acs created (dry run)
configmap/omcgo-app-config created (dry run)
deployment.apps/omcgo-app created (dry run)
horizontalpodautoscaler.autoscaling/omcgo-app created (dry run)
secret/omcgo-app-secret created (dry run)
service/omcgo-app created (dry run)
configmap/omcgo-worker-config created (dry run)
deployment.apps/omcgo-worker created (dry run)
horizontalpodautoscaler.autoscaling/omcgo-worker created (dry run)
secret/omcgo-worker-secret created (dry run)
service/omcgo-worker created (dry run)
exit=0
```

**17 个资源 0 errors** ✅

---

## 4. 章程 Pass 标准核对

| 标准 | 状态 |
|------|------|
| 3 部署单元 × {Deployment, Service, ConfigMap, Secret} 全在 | ✅ 12 文件 |
| HPA × 3 | ✅ |
| `_common/namespace.yaml` | ✅ |
| `_common/ingress.yaml`（仅 app） | ✅ |
| dry-run 0 errors（或 yaml.safe_load 不抛错） | ✅ 两者均通过 |

---

## 5. 已知限制 / 后续事项

1. **镜像 tag 占位**：所有 Deployment 使用 `omcgo/{app,acs,worker}:${VERSION}`，部署前需用 `envsubst` 或 CI/CD pipeline 替换为真实 tag（建议 git short SHA）。
2. **Secret 仅占位**：`REPLACE_ME` 不是有效凭据。生产部署须配合 sealed-secrets / external-secrets / Vault 注入。
3. **Ingress host 是示例**：`omc.example.com` 需替换为真实域名，`ingressClassName: nginx` 视集群 Ingress Controller 调整。
4. **未包含 PV/PVC**：依赖（PostgreSQL/Redis/NATS/MinIO）按惯例使用独立 Helm Chart 或 Operator 部署，不在本批 manifests 范围。
5. **未包含 NetworkPolicy / PDB / ServiceMonitor**：基础 manifests 完成后，可在后续工单（W3.H.x 或 W4）补充安全策略与高可用增强。

---

## 6. 触及边界自检

- [x] 仅在 `deployments/k8s/` 下新建文件，未改 `omcgo/*`、`omcmb/*`、`.github/`、`scripts/db_backup.sh`、`docs/project/release-gate.md`
- [x] 未改 `deployments/docker/` 或 `deployments/monitoring/`
- [x] 未 commit / push / pull
- [x] 未新增 go.mod 依赖
- [x] 全程使用相对路径写文件

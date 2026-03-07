# 08 — 基础设施与部署

> etcd 服务发现、数据库策略、Docker/K8s 部署、可观测性

---

## 1. 基础设施组件总览

```
┌─────────────────────────────────────────────────────────────────┐
│                      Kubernetes Cluster                          │
│                                                                  │
│  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌──────────┐ ┌─────────┐  │
│  │device-  │ │monitor- │ │admin-   │ │acs-      │ │ worker  │  │
│  │api ×2   │ │api ×2   │ │api ×1   │ │engine ×2 │ │ ×2      │  │
│  └────┬────┘ └────┬────┘ └────┬────┘ └────┬─────┘ └────┬────┘  │
│       │           │           │            │            │        │
│  ┌────┴────┐ ┌────┴────┐ ┌───┴────┐ ┌────┴────┐ ┌────┴────┐   │
│  │device-  │ │pm-rpc   │ │admin-  │ │config-  │ │alarm-   │   │
│  │rpc ×2   │ │×2       │ │rpc ×1  │ │rpc ×2   │ │rpc ×2   │   │
│  └─────────┘ └─────────┘ └────────┘ └─────────┘ └─────────┘   │
│                                                                  │
├──────────────────── 基础设施层 ──────────────────────────────────┤
│                                                                  │
│  ┌───────────┐  ┌──────────┐  ┌──────────┐  ┌───────┐  ┌─────┐ │
│  │PostgreSQL │  │ Redis 7  │  │  NATS    │  │ MinIO │  │etcd │ │
│  │16 +       │  │ Cluster  │  │JetStream │  │       │  │ ×3  │ │
│  │TimescaleDB│  │          │  │          │  │       │  │     │ │
│  └───────────┘  └──────────┘  └──────────┘  └───────┘  └─────┘ │
│                                                                  │
├──────────────────── 可观测性 ─────────────────────────────────── │
│                                                                  │
│  ┌────────────┐  ┌──────────┐  ┌──────────┐  ┌──────────────┐   │
│  │ Prometheus │  │  Grafana │  │  Jaeger  │  │ Loki (日志)  │   │
│  └────────────┘  └──────────┘  └──────────┘  └──────────────┘   │
└─────────────────────────────────────────────────────────────────┘
```

---

## 2. etcd 服务发现

### 2.1 为什么需要 etcd

go-zero 使用 etcd 作为服务注册与发现的核心组件：

- RPC 服务启动时自动注册到 etcd
- API 网关通过 etcd 发现下游 RPC 服务地址
- 支持多实例负载均衡（go-zero p2c 算法）
- 配置热更新通过 etcd watch 机制

### 2.2 etcd 集群配置

```yaml
# 3 节点 etcd 集群（生产环境最低要求）
etcd:
  endpoints:
    - etcd-0.etcd-headless:2379
    - etcd-1.etcd-headless:2379
    - etcd-2.etcd-headless:2379
  dial_timeout: 5s
  auth:
    username: omcgo
    password: ${ETCD_PASSWORD}
```

### 2.3 服务注册配置示例

```yaml
# service/device/rpc/etc/device-rpc.yaml
Name: device.rpc
ListenOn: 0.0.0.0:50051
Etcd:
  Hosts:
    - etcd-0.etcd-headless:2379
    - etcd-1.etcd-headless:2379
    - etcd-2.etcd-headless:2379
  Key: device.rpc
Timeout: 5000  # 5 秒超时

# 数据库配置
DataSource: postgres://omcgo:${PG_PASSWORD}@postgresql:5432/omcgo?sslmode=disable
Cache:
  - Host: redis-master:6379
    Pass: ${REDIS_PASSWORD}
    Type: node
```

### 2.4 对比模块化单体

| 维度 | 模块化单体（无 etcd） | go-zero（需 etcd） |
|------|---------------------|-------------------|
| 服务发现 | 不需要（同进程调用） | etcd 自动注册发现 |
| 新增依赖 | 无 | 3 节点 etcd 集群 |
| 运维负担 | 无 | etcd 备份、监控、证书管理 |
| 故障影响 | 无 | etcd 不可用 = 新服务无法注册 |

---

## 3. 数据库策略

### 3.1 共享数据库（非 database-per-service）

```
                    ┌──────────────────────┐
                    │ PostgreSQL 16 实例    │
                    │ + TimescaleDB 扩展    │
                    │                      │
                    │ ┌──────────────────┐ │
                    │ │ omcgo 数据库      │ │
                    │ │                  │ │
                    │ │ devices ←────── device-rpc        │
                    │ │ device_params ← device-rpc        │
                    │ │ device_groups ← device-rpc        │
                    │ │ provisioning ←─ device-rpc        │
                    │ │                  │ │
                    │ │ data_models ←── config-rpc        │
                    │ │ templates ←──── config-rpc        │
                    │ │ oui_registry ←─ config-rpc        │
                    │ │                  │ │
                    │ │ pm_counters ←── pm-rpc / worker   │
                    │ │ kpi_values ←─── pm-rpc / worker   │
                    │ │ meas_reports ←─ pm-rpc / worker   │
                    │ │                  │ │
                    │ │ alarms_active ← alarm-rpc         │
                    │ │ alarms_history← alarm-rpc         │
                    │ │                  │ │
                    │ │ users ←──────── admin-rpc         │
                    │ │ roles ←──────── admin-rpc         │
                    │ │ permissions ←── admin-rpc         │
                    │ │ audit_logs ←─── admin-rpc         │
                    │ │ firmware ←───── admin-rpc         │
                    │ └──────────────────┘ │
                    └──────────────────────┘
```

**为什么共享数据库？**

1. `devices` 表被几乎所有域引用（FK 约束）
2. 自动开站跨 devices + data_models + templates（需事务）
3. database-per-service 会导致大量数据同步逻辑
4. 10 万-100 万规模单库完全承受得住

**数据所有权规则**：

| 服务 | 主表（可写） | 只读引用 |
|------|-------------|---------|
| device-rpc | devices, device_params, device_groups, provisioning_tasks | data_model_definitions |
| config-rpc | data_model_definitions, config_templates, oui_registry | — |
| pm-rpc | — | pm_counters, kpi_values, measurement_reports |
| alarm-rpc | alarms_active | alarms_history |
| admin-rpc | users, roles, permissions, audit_logs, firmware_versions | — |
| worker | pm_counters (写), kpi_values (写), measurement_reports (写), alarms_history (写) | — |

### 3.2 连接池配置

```yaml
# 每个服务独立连接池
DataSource: postgres://omcgo:${PG_PASSWORD}@postgresql:5432/omcgo?sslmode=disable
  # pgxpool 参数
  pool_max_conns: 20         # 每服务最大 20 连接
  pool_min_conns: 5
  pool_max_conn_lifetime: 1h
  pool_max_conn_idle_time: 30m
```

全局连接数预算：10 服务 × 20 = 200 连接（PostgreSQL 默认 max_connections = 200，需调高至 300）。

### 3.3 迁移策略

单一迁移链，所有服务共享：

```bash
# 使用 golang-migrate
migrate -database "postgres://..." -path migrations/ up

# 迁移文件命名
migrations/
├── 000001_create_devices.up.sql
├── 000001_create_devices.down.sql
├── 000002_create_data_models.up.sql
├── ...
├── 000010_create_hypertables.up.sql      # TimescaleDB 特有
├── 000011_create_continuous_aggregates.up.sql
└── 000012_create_users_rbac.up.sql
```

---

## 4. Docker 部署

### 4.1 Dockerfile 模板

```dockerfile
# deployments/docker/Dockerfile.device-rpc
FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /device-rpc service/device/rpc/device.go

FROM alpine:3.19
RUN apk --no-cache add ca-certificates tzdata
COPY --from=builder /device-rpc /usr/local/bin/device-rpc
COPY service/device/rpc/etc/ /etc/omcgo/

EXPOSE 50051
CMD ["device-rpc", "-f", "/etc/omcgo/device-rpc.yaml"]
```

### 4.2 docker-compose（本地开发）

```yaml
# deployments/docker/docker-compose.yml
version: '3.8'

services:
  # ===== 基础设施 =====
  postgresql:
    image: timescale/timescaledb:latest-pg16
    ports: ["5432:5432"]
    environment:
      POSTGRES_DB: omcgo
      POSTGRES_USER: omcgo
      POSTGRES_PASSWORD: omcgo_dev
    volumes:
      - pgdata:/var/lib/postgresql/data

  redis:
    image: redis:7-alpine
    ports: ["6379:6379"]
    command: redis-server --requirepass redis_dev

  nats:
    image: nats:2-alpine
    ports: ["4222:4222", "8222:8222"]
    command: "--js --store_dir /data"
    volumes:
      - natsdata:/data

  minio:
    image: minio/minio:latest
    ports: ["9000:9000", "9001:9001"]
    environment:
      MINIO_ROOT_USER: minioadmin
      MINIO_ROOT_PASSWORD: minioadmin
    command: server /data --console-address ":9001"
    volumes:
      - miniodata:/data

  etcd:
    image: bitnami/etcd:3.5
    ports: ["2379:2379"]
    environment:
      ALLOW_NONE_AUTHENTICATION: "yes"
      ETCD_ADVERTISE_CLIENT_URLS: "http://etcd:2379"
      ETCD_LISTEN_CLIENT_URLS: "http://0.0.0.0:2379"

  # ===== 应用服务 =====
  acs-engine:
    build:
      context: ../..
      dockerfile: deployments/docker/Dockerfile.acs
    ports: ["7547:7547", "50050:50050"]
    depends_on: [postgresql, redis, nats, etcd]
    environment:
      PG_PASSWORD: omcgo_dev
      REDIS_PASSWORD: redis_dev

  device-rpc:
    build:
      context: ../..
      dockerfile: deployments/docker/Dockerfile.device-rpc
    ports: ["50051:50051"]
    depends_on: [postgresql, redis, etcd]

  config-rpc:
    build:
      context: ../..
      dockerfile: deployments/docker/Dockerfile.config-rpc
    ports: ["50052:50052"]
    depends_on: [postgresql, redis, etcd]

  pm-rpc:
    build:
      context: ../..
      dockerfile: deployments/docker/Dockerfile.pm-rpc
    ports: ["50053:50053"]
    depends_on: [postgresql, redis, etcd]

  alarm-rpc:
    build:
      context: ../..
      dockerfile: deployments/docker/Dockerfile.alarm-rpc
    ports: ["50054:50054"]
    depends_on: [postgresql, redis, etcd]

  admin-rpc:
    build:
      context: ../..
      dockerfile: deployments/docker/Dockerfile.admin-rpc
    ports: ["50055:50055"]
    depends_on: [postgresql, redis, etcd]

  device-api:
    build:
      context: ../..
      dockerfile: deployments/docker/Dockerfile.device-api
    ports: ["8080:8080"]
    depends_on: [device-rpc, config-rpc, etcd]

  monitor-api:
    build:
      context: ../..
      dockerfile: deployments/docker/Dockerfile.monitor-api
    ports: ["8081:8081"]
    depends_on: [pm-rpc, alarm-rpc, etcd]

  admin-api:
    build:
      context: ../..
      dockerfile: deployments/docker/Dockerfile.admin-api
    ports: ["8082:8082"]
    depends_on: [admin-rpc, etcd]

  worker:
    build:
      context: ../..
      dockerfile: deployments/docker/Dockerfile.worker
    depends_on: [postgresql, redis, nats, minio]

  # ===== 可观测性 =====
  prometheus:
    image: prom/prometheus:latest
    ports: ["9090:9090"]
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml

  grafana:
    image: grafana/grafana:latest
    ports: ["3000:3000"]

  jaeger:
    image: jaegertracing/all-in-one:latest
    ports: ["16686:16686", "14268:14268"]

volumes:
  pgdata:
  natsdata:
  miniodata:
```

---

## 5. Kubernetes 部署

### 5.1 关键 K8s 资源

| 服务 | K8s 资源类型 | 副本数 | HPA |
|------|-------------|:------:|:---:|
| acs-engine | Deployment + LoadBalancer Service | 2 | 是（CPU 70%） |
| device-api | Deployment + ClusterIP | 2 | 是 |
| monitor-api | Deployment + ClusterIP | 2 | 是 |
| admin-api | Deployment + ClusterIP | 1 | 否 |
| device-rpc | Deployment + ClusterIP | 2 | 是 |
| config-rpc | Deployment + ClusterIP | 2 | 是 |
| pm-rpc | Deployment + ClusterIP | 2 | 是 |
| alarm-rpc | Deployment + ClusterIP | 2 | 是 |
| admin-rpc | Deployment + ClusterIP | 1 | 否 |
| worker | Deployment | 2 | KEDA |
| PostgreSQL | StatefulSet | 1（主） | 否 |
| Redis | StatefulSet | 3（Cluster） | 否 |
| etcd | StatefulSet | 3 | 否 |
| NATS | StatefulSet | 3 | 否 |

### 5.2 Ingress 路由

```yaml
# deployments/k8s/ingress.yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: omcgo-ingress
  annotations:
    nginx.ingress.kubernetes.io/rewrite-target: /
spec:
  rules:
    - host: omc.example.com
      http:
        paths:
          - path: /api/v1/devices
            pathType: Prefix
            backend:
              service:
                name: device-api
                port:
                  number: 8080
          - path: /api/v1/topology
            pathType: Prefix
            backend:
              service:
                name: device-api
                port:
                  number: 8080
          - path: /api/v1/datamodels
            pathType: Prefix
            backend:
              service:
                name: device-api
                port:
                  number: 8080
          - path: /api/v1/provisioning
            pathType: Prefix
            backend:
              service:
                name: device-api
                port:
                  number: 8080
          - path: /api/v1/pm
            pathType: Prefix
            backend:
              service:
                name: monitor-api
                port:
                  number: 8081
          - path: /api/v1/alarms
            pathType: Prefix
            backend:
              service:
                name: monitor-api
                port:
                  number: 8081
          - path: /api/v1/users
            pathType: Prefix
            backend:
              service:
                name: admin-api
                port:
                  number: 8082
          - path: /api/v1/auth
            pathType: Prefix
            backend:
              service:
                name: admin-api
                port:
                  number: 8082
```

### 5.3 HPA 示例

```yaml
# deployments/k8s/hpa-acs.yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: acs-engine-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: acs-engine
  minReplicas: 2
  maxReplicas: 20
  metrics:
    - type: Resource
      resource:
        name: cpu
        target:
          type: Utilization
          averageUtilization: 70
    - type: Pods
      pods:
        metric:
          name: acs_active_sessions
        target:
          type: AverageValue
          averageValue: "200"    # 每实例最多 200 并发会话
```

---

## 6. 可观测性

### 6.1 日志

go-zero 内置结构化日志：

```yaml
# 各服务配置
Log:
  ServiceName: device-rpc
  Mode: file           # console / file / volume
  Path: /var/log/omcgo
  Level: info
  KeepDays: 7
  Compress: true
  Encoding: json       # 结构化 JSON 输出
```

日志格式示例：
```json
{
    "level": "info",
    "ts": "2026-03-05T10:30:00.000Z",
    "caller": "logic/registerdevicelogic.go:45",
    "msg": "device registered",
    "device_serial": "SN123456",
    "carrier": "cmcc",
    "duration": "2.3ms",
    "trace": "abc123def456"
}
```

### 6.2 指标（Prometheus）

go-zero 自动采集的指标：

| 指标 | 类型 | 说明 |
|------|------|------|
| `rpc_server_requests_total` | Counter | RPC 请求总数 |
| `rpc_server_requests_duration_ms` | Histogram | RPC 请求延迟 |
| `http_server_requests_total` | Counter | HTTP 请求总数 |
| `http_server_requests_duration_ms` | Histogram | HTTP 请求延迟 |
| `rpc_client_requests_total` | Counter | RPC 客户端请求数 |
| `rpc_client_requests_duration_ms` | Histogram | RPC 客户端延迟 |

自定义指标（ACS 引擎）：

| 指标 | 类型 | 说明 |
|------|------|------|
| `acs_active_sessions` | Gauge | 活跃 TR069 会话数 |
| `acs_inform_total` | Counter | Inform 消息总数 |
| `acs_rpc_duration_seconds` | Histogram | TR069 RPC 执行延迟 |
| `acs_session_duration_seconds` | Histogram | 会话持续时间 |
| `acs_command_queue_size` | Gauge | 命令队列长度 |

### 6.3 链路追踪（OpenTelemetry）

```yaml
# 各服务配置
Telemetry:
  Name: device-rpc
  Endpoint: http://jaeger:14268/api/traces
  Sampler: 0.1    # 10% 采样率
  Batcher: jaeger
```

go-zero 自动在 zRPC 调用链中传播 trace context，跨服务调用自动关联。

### 6.4 健康检查

```yaml
# K8s 探针配置
livenessProbe:
  httpGet:
    path: /healthz
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 10

readinessProbe:
  httpGet:
    path: /readyz
    port: 8080
  initialDelaySeconds: 10
  periodSeconds: 5
```

---

## 7. 基础设施对比

| 组件 | 模块化单体 | go-zero 微服务 | 增量 |
|------|-----------|---------------|------|
| PostgreSQL 16 + TimescaleDB | 1 实例 | 1 实例 | 无 |
| Redis 7 | 1 实例 | 1 集群（或 1 实例） | 无 |
| NATS JetStream | 1-3 节点 | 1-3 节点 | 无 |
| MinIO | 1 实例 | 1 实例 | 无 |
| **etcd** | **不需要** | **3 节点集群** | **+1 组件** |
| Prometheus | 1 实例 | 1 实例 | 无 |
| Jaeger/Zipkin | 可选 | **推荐** | 重要性提升 |
| Docker 镜像数 | 3 | 10 | +7 |
| K8s Deployment | 3 | 10 | +7 |
| 总容器数（最小） | 8 | 15+ | +7 |

# DD-20: 部署架构与运维

> 关联功能域：全部
> 关联 backend-design.md 章节：第七章（可扩展性设计）、第八章（高可用设计）、第九章（部署架构）
> 实施阶段：Phase 1（Docker）→ Phase 4（K8s + 规模化）
> 依赖文档：DD-01

---

## 1. 概述

### 1.1 模块定位

部署架构定义了系统的容器化、编排、扩展和生产运维方案，覆盖从开发环境到生产环境的完整部署链路。

### 1.2 核心职责

- Docker 镜像构建（三个应用 + 迁移工具）
- docker-compose 本地开发环境
- Kubernetes 生产部署（HPA/KEDA 弹性伸缩）
- 10 万 / 100 万规模部署方案

---

## 2. Docker 镜像设计

### 2.1 Dockerfile — 三阶段构建

**Dockerfile.acs**：

```dockerfile
# 构建阶段
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /omcgo-acs ./cmd/acs

# 运行阶段
FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /omcgo-acs /usr/local/bin/
COPY configs/acs.yaml /etc/omcgo/
EXPOSE 7547 7548 9090
ENTRYPOINT ["omcgo-acs"]
CMD ["--config", "/etc/omcgo/acs.yaml"]
```

类似结构用于 `Dockerfile.app`（端口 8080/8443/9091/50051）和 `Dockerfile.worker`（端口 9092）。

### 2.2 docker-compose 本地开发环境

```yaml
# deployments/docker/docker-compose.yml
version: "3.8"

services:
  postgres:
    image: timescale/timescaledb:latest-pg16
    ports: ["5432:5432"]
    environment:
      POSTGRES_USER: omcgo
      POSTGRES_PASSWORD: password
      POSTGRES_DB: omcgo
    volumes:
      - pgdata:/var/lib/postgresql/data

  redis:
    image: redis:7-alpine
    ports: ["6379:6379"]

  nats:
    image: nats:2.10-alpine
    ports: ["4222:4222", "8222:8222"]
    command: ["--js"]

  minio:
    image: minio/minio:latest
    ports: ["9000:9000", "9001:9001"]
    environment:
      MINIO_ROOT_USER: minioadmin
      MINIO_ROOT_PASSWORD: minioadmin
    command: server /data --console-address ":9001"

  acs:
    build:
      context: ../..
      dockerfile: deployments/docker/Dockerfile.acs
    ports: ["7547:7547", "9090:9090"]
    depends_on: [postgres, redis, nats]

  app:
    build:
      context: ../..
      dockerfile: deployments/docker/Dockerfile.app
    ports: ["8080:8080", "9091:9091", "50051:50051"]
    depends_on: [postgres, redis, nats, minio]

  worker:
    build:
      context: ../..
      dockerfile: deployments/docker/Dockerfile.worker
    depends_on: [postgres, redis, nats, minio]

volumes:
  pgdata:
```

---

## 3. Kubernetes 部署

### 3.1 部署拓扑

```yaml
Namespace: omcgo

# ACS 引擎 — HPA 基于活跃连接数
Deployment: acs         (replicas: 2-20)
Service:    acs-lb      (type: LoadBalancer, port 7547/7548)

# 应用服务器 — HPA 基于 CPU
Deployment: app         (replicas: 2-5)
Service:    app-api     (type: ClusterIP, port 8080)
Ingress:    api.omc.example.com → app-api

# Worker — KEDA 基于 NATS 队列深度
Deployment: worker      (replicas: 2-10)

# 基础设施（StatefulSet）
StatefulSet: postgresql  (replicas: 2, Patroni)
StatefulSet: redis       (replicas: 6, Cluster)
StatefulSet: nats        (replicas: 3, JetStream)
StatefulSet: minio       (replicas: 4, distributed)
```

### 3.2 ACS HPA 配置

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: acs-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: acs
  minReplicas: 2
  maxReplicas: 20
  metrics:
  - type: Pods
    pods:
      metric:
        name: acs_active_sessions
      target:
        type: AverageValue
        averageValue: "2000"  # 每实例 2000 并发会话
```

---

## 4. 规模化方案

### 4.1 Tier 1（10 万基站）

| 组件 | 规格 |
|------|------|
| ACS | 2-3 实例 × 2 CPU / 2 GB |
| App | 1 实例 × 4 CPU / 4 GB |
| Worker | 2 实例 × 2 CPU / 2 GB |
| PostgreSQL | 主从 × 8 CPU / 16 GB |
| TimescaleDB | 单实例 × 8 CPU / 16 GB |
| Redis | 3 主 + 3 从 × 2 CPU / 4 GB |
| NATS | 3 节点 × 2 CPU / 2 GB |
| MinIO | 单实例 × 4 CPU / 8 GB / 500 GB SSD |

### 4.2 Tier 2（100 万基站）

| 组件 | 规格 |
|------|------|
| ACS | 10-20 实例 |
| App | 3-5 实例 |
| Worker | 5-10 实例 |
| PostgreSQL | 按运营商分区，区域只读副本 |
| TimescaleDB | 分布式超表 |
| Redis | 更大集群，会话和缓存分离 |
| NATS | 更大集群或切换 Kafka |
| MinIO | 分布式 4+ 节点 |

---

## 5. 高可用策略

| 组件 | HA 策略 |
|------|--------|
| ACS | 多实例无状态，LB 健康检查，任意实例处理任意设备 |
| App | 多实例无状态（状态在 DB/Redis） |
| PostgreSQL | 主从复制 + Patroni 自动 Failover（30s） |
| Redis | Cluster 自动提升副本（~5s） |
| NATS | 3 节点集群 + JetStream 跨节点复制 |
| MinIO | 纠删码模式 |

---

## 6. 实施子阶段

### 阶段 20a：Dockerfile + docker-compose（Phase 1）
### 阶段 20b：K8s 基础清单（Phase 3/4）
### 阶段 20c：HPA/KEDA + 生产加固（Phase 4）

---

## 7. 文件清单

```
deployments/docker/Dockerfile.acs
deployments/docker/Dockerfile.app
deployments/docker/Dockerfile.worker
deployments/docker/docker-compose.yml
deployments/k8s/acs/deployment.yaml
deployments/k8s/acs/service.yaml
deployments/k8s/acs/hpa.yaml
deployments/k8s/app/deployment.yaml
deployments/k8s/app/service.yaml
deployments/k8s/app/ingress.yaml
deployments/k8s/worker/deployment.yaml
deployments/k8s/infra/
```

---

## 8. 参考

- backend-design.md 第七章：可扩展性设计
- backend-design.md 第八章：高可用设计
- backend-design.md 第九章：部署架构

# OMC Docker Compose 部署指南

## 1. 概述

本目录包含 OMC（Operations Management Center）系统的 Docker Compose 完整部署配置。系统由以下层次构成：

```
┌─────────────────────────────────────────────┐
│              Nginx 网关（web）               │
│   port 8081（前端 + REST API）               │
│   port 8080（ACS TR-069 设备接入）           │
└──────────┬─────────────────────┬────────────┘
           │                     │
    ┌──────▼──────┐       ┌──────▼──────┐
    │  app :8081  │       │  acs :7557  │
    │ REST API 服务│       │ TR-069 服务  │
    └──────┬──────┘       └──────┬──────┘
           │                     │
    ┌──────▼──────────────────────▼──────┐
    │           worker :9092             │
    │        后台异步任务处理              │
    └──────────────┬─────────────────────┘
                   │
    ┌──────────────▼─────────────────────────────────┐
    │          基础设施层                              │
    │  PostgreSQL(TimescaleDB) | Redis | NATS | MinIO │
    └────────────────────────────────────────────────┘
```

**所有命令均需在项目根目录 `goomc/` 下执行。**

---

## 2. 服务列表

| 服务名       | 镜像 / Dockerfile        | 对外端口                    | 作用                                     |
|-------------|--------------------------|----------------------------|------------------------------------------|
| `postgres`  | timescale/timescaledb:latest-pg16 | `5432:5432`   | 主数据库（含 TimescaleDB 时序扩展）        |
| `redis-core` | redis:7-alpine          | `6379:6379`                | ACS 会话 / 任务 / 告警 / 核心缓存         |
| `redis-pm` | redis:7-alpine            | 仅容器网络                 | PM 窗口 / KPI 聚合状态                    |
| `nats`      | nats:2.10-alpine         | `4222:4222`, `8222:8222`   | 消息队列（JetStream 模式），8222 为监控端口 |
| `minio`     | `minio/minio:RELEASE.2024-06-13T22-53-53Z` | `9000:9000`, `9001:9001`   | 对象存储，9001 为 Web 控制台              |
| `migrate`   | Dockerfile.app           | —（一次性任务）             | 启动时执行数据库迁移，成功后退出            |
| `acs`       | Dockerfile.acs           | `9095:9090` (metrics), `7557:7557` | ACS 服务，处理 TR-069 CWMP 设备通信。metrics 让出宿主 9090 给 Prometheus |
| `app`       | Dockerfile.app           | `18081:8081` (host loopback), `9091:9091` (metrics) | REST API 管理面服务，业务流量通过 Nginx 代理 |
| `worker`    | Dockerfile.worker        | `9092:9092`                | 后台异步任务 Worker                       |
| `web`       | Dockerfile.web           | `8080:8080`, `8081:8081`    | Nginx 网关：宿主机 8081 前端+API，8080 ACS 代理；默认通过 Docker bridge 访问 app/acs |
| `prometheus` | prom/prometheus:v2.51.0 | `9090:9090`                | 指标存储与查询                            |
| `alertmanager` | prom/alertmanager:v0.27.0 | `9093:9093`            | 告警路由                                  |
| `grafana`   | grafana/grafana:10.4.0   | `3030:3000`                | 可视化（admin/admin，dev 默认）。宿主 3030 避让 vite dev :3000 |
| `loki`      | grafana/loki:3.0.0       | `3100:3100`                | 日志存储与查询                            |
| `promtail`  | grafana/promtail:3.0.0   | —（仅容器内）              | 日志采集 agent，tail run/logs 推到 Loki   |

---

## 3. 前置条件

| 依赖         | 最低版本       | 说明                                 |
|-------------|---------------|--------------------------------------|
| Docker      | 24.0+         | 支持 Compose v2 内置命令              |
| Docker Compose | v2.20+     | 内置于 Docker Desktop，或独立安装      |
| 可用内存      | ≥ 4 GB        | 所有服务同时运行的推荐内存              |
| 可用磁盘      | ≥ 10 GB       | 镜像、数据卷、日志存储空间              |

验证版本：

```bash
docker --version
docker compose version
```

> **国内服务器首次部署**：基础镜像（`alpine` / `golang` / `node` / `nginx`）从 Docker Hub 拉取，
> 国内访问通常会超时。先按 [REGISTRY_SETUP.md](./REGISTRY_SETUP.md) 配宿主机 Docker / BuildKit
> 镜像源（10 分钟一次性配置），再回来执行 §4 启动流程。

---

## 4. 快速启动

### 4.1 首次启动（完整构建）

```bash
# 在项目根目录 goomc/ 下执行

# 1. 使用默认网段构建所有镜像并后台启动
bash deployments/docker/dc.sh up -d --build

# 只有默认网段与客户业务网冲突时，才传入自定义规划：
# DOCKER_BIP=10.250.0.1/16 bash deployments/docker/dc.sh up -d --build

# 如需后续命令不再重复传参，请把同一个客户规划值写入项目根 .env：
# DOCKER_BIP=10.240.0.1/16

# 2. 查看启动状态
bash deployments/docker/dc.sh ps

# 3. 查看实时日志（所有服务）
bash deployments/docker/dc.sh logs -f
```

### 4.2 日常启动（跳过构建）

```bash
# 默认使用 10.240.0.1/16；项目根 .env 中可填写冲突时的自定义 DOCKER_BIP
bash deployments/docker/dc.sh up -d
```

### 4.3 停止服务

```bash
# 停止但保留数据卷
bash deployments/docker/dc.sh down

# 停止并删除所有数据卷（彻底清除数据）
bash deployments/docker/dc.sh down -v
```

### 4.4 重启单个服务

```bash
# 仅重启容器（不重新编译，适用于配置文件变更）
bash deployments/docker/dc.sh restart app
```

### 4.5 重新编译并重启

```bash
# 重新构建镜像（触发 Go 编译）并用新镜像替换旧容器
bash deployments/docker/dc.sh up -d --build app
```

> **注意**：`restart` 只是停止并重启已有容器，不会重新编译代码。修改了 Go 源码或前端代码后，必须使用 `up -d --build` 才能生效。

### 4.6 可选 host network 覆盖

默认 `web` 服务使用 Docker bridge 网络，通过服务名访问 `app`、`acs`，并显式暴露宿主 `8081`、`8080` 端口。

如果部署环境明确需要 `web` 直接绑定宿主网络，可叠加本地覆盖文件：

```bash
source deployments/docker/docker-network-lib.sh
DOCKER_BIP=10.240.0.1/16
docker_network_plan
docker compose -f deployments/docker/docker-compose.yml -f deployments/docker/docker-compose.local.yml up -d --build web
```

该覆盖会使用 `default.local.conf`，把 Nginx 上游切到宿主映射端口 `127.0.0.1:18081` 和 `127.0.0.1:7557`，不改变默认部署拓扑。

---

## 5. 环境变量

### 5.1 全局环境变量

| 变量名          | 默认值    | 说明                                                   |
|----------------|----------|--------------------------------------------------------|
| `OMCGO_ENV`    | `dev`    | 运行环境：`dev` / `test` / `prod`，决定加载哪个配置文件 |
| `TZ`           | `Asia/Shanghai` | 容器时区，所有服务统一设置                        |

### 5.2 服务专用环境变量

**app 服务：**

| 变量名               | 默认值                                          | 说明              |
|---------------------|------------------------------------------------|-------------------|
| `OMCGO_JWT_SECRET`  | `8f7a9b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6q7r8s9t0u1v2w3x4y5z6` | JWT 签名密钥，**生产环境务必修改** |

**postgres 服务（基础设施）：**

| 变量名               | 默认值        | 说明          |
|---------------------|--------------|---------------|
| `POSTGRES_USER`     | `omcgo`      | 数据库用户名   |
| `POSTGRES_PASSWORD` | `omcgo123`   | 数据库密码     |
| `POSTGRES_DB`       | `omcgo`      | 数据库名       |

**minio 服务（基础设施）：**

| 变量名                  | 默认值         | 说明           |
|------------------------|---------------|----------------|
| `MINIO_ROOT_USER`      | `minioadmin`  | MinIO 管理员账号 |
| `MINIO_ROOT_PASSWORD`  | `minioadmin`  | MinIO 管理员密码 |

### 5.3 使用 .env 文件覆盖变量

在项目根目录创建 `.env` 文件：

```bash
# .env（放在 goomc/ 根目录）
OMCGO_ENV=prod
OMCGO_JWT_SECRET=your-strong-secret-here
```

启动时会自动读取（需在 `goomc/` 目录执行 compose 命令）。

---

## 6. 服务详情

### 6.1 postgres（数据库）

- **镜像**：`timescale/timescaledb:latest-pg16`（PostgreSQL 16 + TimescaleDB 时序扩展）
- **端口**：`5432`
- **数据卷**：`pgdata:/var/lib/postgresql/data`（持久化存储）
- **时区**：通过启动参数 `timezone=Asia/Shanghai` 强制设置
- **健康检查**：`pg_isready -U omcgo`，间隔 5s，最多重试 5 次

### 6.2 redis-core / redis-pm（物理隔离）

- **镜像**：两个实例均为 `redis:7-alpine`
- **端口**：`6379`
- **持久化**：启用 AOF（`--appendonly yes`）
- **数据卷**：`redisdata:/data`
- **健康检查**：两个实例分别执行 `redis-cli ping`，间隔 5s
- **隔离原则**：ACS 与普通业务只使用 `redis-core`；PM 窗口和 KPI 路由使用 `redis-pm`，避免关窗、重算及 AOF rewrite 与核心业务争抢内存和磁盘 I/O

### 6.3 nats（消息队列）

- **镜像**：`nats:2.10-alpine`
- **端口**：`4222`（客户端连接），`8222`（HTTP 监控）
- **模式**：JetStream 持久化消息队列，存储目录 `/data`
- **数据卷**：`natsdata:/data`
- **健康检查**：HTTP GET `http://localhost:8222/healthz`
- **监控访问**：`http://localhost:8222`

### 6.4 minio（对象存储）

- **镜像**：`minio/minio:RELEASE.2024-06-13T22-53-53Z`（可用 `IMAGE_MINIO` 显式覆盖）
- **端口**：`9000`（S3 API），`9001`（Web 控制台）
- **数据卷**：`miniodata:/data`
- **Web 控制台**：`http://localhost:9001`（账号 `minioadmin` / `minioadmin`）
- **健康检查**：`curl http://localhost:9000/minio/health/live`

### 6.5 migrate（数据库迁移，一次性任务）

- **入口命令**：`omcgo-migrate`
- **执行逻辑**：`up` 方向应用全部未执行的迁移文件
- **DSN**：`postgres://omcgo:omcgo123@postgres:5432/omcgo?sslmode=disable`
- **迁移文件**：`Dockerfile.app` 构建时已将 `omcgo/migrations/` 内嵌到镜像 `/etc/omcgo/migrations`；compose 中额外以只读 bind mount 将宿主机 `omcgo/migrations/` 覆盖挂载，确保使用最新迁移文件
- **依赖**：等待 `postgres` 健康后执行，成功退出后其他 App 服务才会启动

### 6.6 acs（ACS 服务）

- **端口**：`9090`（管理/监控），`7557`（TR-069 CWMP 设备接入）
- **配置文件**：`/etc/omcgo/acs.${OMCGO_ENV}.yaml`
- **日志挂载**：`run/logs/acs/` → 容器 `/run/logs/acs`
- **依赖**：migrate 成功 + redis-core/nats/minio 健康

### 6.7 app（REST API 服务）

- **端口**：不对外暴露，仅在 Docker 内部网络中监听 `8081`，由 Nginx `app_backend` upstream 代理
- **配置文件**：`/etc/omcgo/app.${OMCGO_ENV}.yaml`
- **日志挂载**：`run/logs/app/` → 容器 `/run/logs/app`
- **依赖**：migrate 成功 + redis-core/redis-pm/nats/minio 健康

### 6.8 worker（后台任务服务）

- **端口**：`9092`
- **配置文件**：`/etc/omcgo/worker.${OMCGO_ENV}.yaml`
- **日志挂载**：`run/logs/worker/` → 容器 `/run/logs/worker`
- **依赖**：migrate 成功 + redis-core/redis-pm/nats/minio 健康

### 6.9 web（Nginx 网关）

- **端口**：`8081`（前端 SPA + REST API 代理），`8080`（ACS TR-069 代理）
- **构建方式**：基于 `Dockerfile.web` 多阶段构建，前端静态资源、`nginx.conf`、`default.conf` 均已在构建时内嵌到镜像中，无需运行时挂载
- **文件描述符限制**：soft 20480 / hard 40960（匹配 nginx `worker_rlimit_nofile`）
- **日志挂载**：`run/logs/nginx/` → 容器 `/var/log/nginx`（仅日志目录需要挂载）
- **依赖**：app 和 acs 容器启动后才启动

---

## 7. Dockerfile 说明

所有后端 Dockerfile 均采用**多阶段构建**，最终镜像基于 `alpine:3.19`，体积小且安全。

### 7.1 Dockerfile.acs

```
阶段1（builder）：golang:1.25-alpine
  └─ GOPROXY=https://goproxy.cn  加速国内依赖下载
  └─ 编译 ./cmd/acs → /build/bin/omcgo-acs

阶段2（runtime）：alpine:3.19
  └─ 安装 ca-certificates tzdata
  └─ 复制二进制文件 omcgo-acs
  └─ 复制三套配置：acs.dev/test/prod.yaml → /etc/omcgo/
  └─ 复制 entrypoint.sh
  └─ 暴露端口：7557、7558、9090
```

### 7.2 Dockerfile.app

```
阶段1（builder）：golang:1.25-alpine
  └─ 同时编译两个二进制：
     - omcgo-app（REST API 服务）
     - omcgo-migrate（数据库迁移工具）

阶段2（runtime）：alpine:3.19
  └─ 复制 omcgo-app + omcgo-migrate
  └─ 复制三套配置：app.dev/test/prod.yaml → /etc/omcgo/
  └─ 复制迁移文件目录 → /etc/omcgo/migrations
  └─ 暴露端口：8081（HTTP）、8444（HTTPS）、9091（Metrics）、50051（gRPC）
```

### 7.3 Dockerfile.worker

```
阶段1（builder）：golang:1.25-alpine
  └─ 编译 ./cmd/worker → /build/bin/omcgo-worker

阶段2（runtime）：alpine:3.19
  └─ 复制 omcgo-worker
  └─ 复制三套配置：worker.dev/test/prod.yaml → /etc/omcgo/
  └─ 暴露端口：9092
```

### 7.4 Dockerfile.web

```
阶段1（builder）：node:20-alpine
  └─ WORKDIR /app
  └─ COPY omcmb/webcode/package.json + package-lock.json
  └─ npm ci 安装依赖（基于 package-lock.json，保证版本锁定）
  └─ COPY omcmb/webcode/ 源码
  └─ npm run build → /app/dist

阶段2（runtime）：nginx:1.31.2-alpine
  └─ 复制构建产物 /app/dist → /usr/share/nginx/html
  └─ COPY nginx.conf → /etc/nginx/nginx.conf（内嵌到镜像，无需运行时挂载）
  └─ COPY default.conf → /etc/nginx/conf.d/default.conf（内嵌到镜像，无需运行时挂载）
  └─ 暴露端口：8080、8081
```

### 7.5 entrypoint.sh（通用入口脚本）

后端三个服务（acs / app / worker）共用同一个入口脚本，逻辑如下：

```
ENV     = ${OMCGO_ENV:-dev}        # 默认 dev
SERVICE = ${OMCGO_SERVICE:-app}    # 由 Dockerfile ENV 预设

配置文件路径 = /etc/omcgo/${SERVICE}.${ENV}.yaml
  → 若不存在，回退到 /etc/omcgo/${SERVICE}.dev.yaml

最终执行：omcgo-${SERVICE} --config <config_path> [额外参数]
```

---

## 8. Nginx 网关配置

### 8.1 主配置（nginx.conf）

| 参数                   | 值       | 说明                                   |
|-----------------------|---------|----------------------------------------|
| `worker_processes`    | auto    | 自动匹配 CPU 核数                        |
| `worker_connections`  | 10240   | 单 worker 最大并发连接                   |
| `worker_rlimit_nofile`| 20480   | 单 worker 文件描述符上限                  |
| `keepalive_timeout`   | 65s     | 客户端空闲连接保持时间                    |
| `keepalive_requests`  | 1000    | 单连接最大请求数                         |
| Gzip 压缩             | 开启     | 仅压缩前端静态资源，不压缩 ACS SOAP/XML   |

**理论并发容量**：`worker_processes × worker_connections`，4 核机器约 40960 并发连接。

### 8.2 Upstream 连接池

| upstream       | 目标地址      | keepalive 连接数 | 说明                         |
|---------------|-------------|----------------|------------------------------|
| `acs_backend` | acs:7557    | 5120           | 高并发设备接入，维持大量长连接  |
| `app_backend` | app:8081    | 64             | 管理面并发较低                |

### 8.3 端口路由规则

**port 8081 — 前端 + REST API 网关**

| 路径           | 转发目标         | 说明                                  |
|---------------|----------------|---------------------------------------|
| `/api/*`      | app:8081        | REST API 代理，超时 60s               |
| `/` (其他)    | 静态文件目录     | SPA fallback，未匹配路径返回 index.html |
| `*.js/css/...`| 静态文件目录     | 30 天缓存，Cache-Control: immutable    |

**port 8080 — ACS TR-069 网关**

| 路径  | 转发目标         | 说明                                         |
|------|----------------|----------------------------------------------|
| `/*` | acs:7557        | TR-069 CWMP 代理，超时 300s，禁用代理缓冲     |
| 上传限制 | —           | `client_max_body_size 100m`（支持大 PM/MR 文件）|

---

## 9. 数据持久化

五个有状态服务均支持通过环境变量切换宿主机 bind mount：
`POSTGRES_DATA_PATH`、`TSDB_DATA_PATH`、`REDIS_DATA_PATH`、`NATS_DATA_PATH`、
`MINIO_DATA_PATH`。变量留空时仍使用下表命名卷，已有开发数据不会因升级被旁路；填写时
必须使用绝对路径。切换已有数据前先停栈并完整复制，不能只修改变量后直接启动。

### 9.1 Docker 命名卷（自动管理）

| 卷名         | 挂载路径                         | 存储内容              |
|-------------|----------------------------------|----------------------|
| `pgdata`    | postgres:/var/lib/postgresql/data | PostgreSQL 数据文件   |
| `tsdbdata`  | postgres-tsdb:/var/lib/postgresql/data | TimescaleDB 时序数据 |
| `redisdata` | redis-core:/data                 | 核心 Redis AOF 持久化文件 |
| `redispmdata` | redis-pm:/data                 | PM Redis AOF 持久化文件 |
| `natsdata`  | nats:/data                       | NATS JetStream 消息  |
| `miniodata` | minio:/data                      | MinIO 对象文件        |

### 9.2 宿主机绑定挂载（日志）

| 宿主机路径（相对项目根）    | 容器路径                        | 说明                           |
|--------------------------|--------------------------------|--------------------------------|
| `run/logs/acs/`          | `/run/logs/acs`                | ACS 服务日志                   |
| `run/logs/app/`          | `/run/logs/app`                | App 服务日志                   |
| `run/logs/worker/`       | `/run/logs/worker`             | Worker 服务日志                |
| `run/logs/nginx/`        | `/var/log/nginx`               | Nginx 访问和错误日志            |
| `omcgo/migrations/`      | `/etc/omcgo/migrations`（只读）| 覆盖镜像内迁移文件，使用最新版本  |
| `/etc/localtime`         | `/etc/localtime`（只读）       | 同步宿主机时区                  |

> **注意**：`web` 服务的前端静态资源、`nginx.conf`、`default.conf` 均已在 `Dockerfile.web` 构建时 `COPY` 进镜像，**不需要**运行时 bind mount。

---

## 10. 日志管理

### 10.1 应用日志

后端服务日志由 **lumberjack** 库管理（在各服务的 `config.yaml` 中配置）：

| 配置项        | 默认值  | 说明                     |
|--------------|--------|--------------------------|
| Max file size | 20 MB  | 单文件超过后自动轮转       |
| Max age       | 7 天   | 超过天数的日志文件自动删除  |
| Max backups   | 100 个 | 保留最多 100 个备份文件    |
| Compress      | 开启   | 历史日志 gzip 压缩        |

### 10.2 实时查看日志

```bash
# 查看 ACS 服务日志
tail -f run/logs/acs/acs.log

# 查看 App 服务日志
tail -f run/logs/app/app.log

# 查看 Worker 服务日志
tail -f run/logs/worker/worker.log

# 查看 Nginx 访问日志
tail -f run/logs/nginx/app_access.log    # 前端/API 访问
tail -f run/logs/nginx/acs_access.log    # ACS 设备访问
tail -f run/logs/nginx/app_error.log     # 前端/API 错误
tail -f run/logs/nginx/acs_error.log     # ACS 错误
tail -f run/logs/nginx/error.log         # Nginx 全局错误
```

### 10.3 Docker 容器日志

```bash
# 查看某服务的容器标准输出
docker compose -f deployments/docker/docker-compose.yml logs -f app
docker compose -f deployments/docker/docker-compose.yml logs -f acs
docker compose -f deployments/docker/docker-compose.yml logs --tail=100 worker
```

---

## 11. 常用操作

### 11.1 构建镜像

```bash
# 构建所有服务镜像
docker compose -f deployments/docker/docker-compose.yml build

# 构建单个服务（不使用缓存）
docker compose -f deployments/docker/docker-compose.yml build --no-cache app

# 构建并启动
docker compose -f deployments/docker/docker-compose.yml up -d --build
```

### 11.2 启动 / 停止 / 重启

```bash
# 启动所有服务（后台）
docker compose -f deployments/docker/docker-compose.yml up -d

# 停止所有服务（保留数据）
docker compose -f deployments/docker/docker-compose.yml down

# 重启单个服务
docker compose -f deployments/docker/docker-compose.yml restart app

# 强制重建并重启单个服务
docker compose -f deployments/docker/docker-compose.yml up -d --build app
```

### 11.3 查看状态

```bash
# 查看所有服务状态
docker compose -f deployments/docker/docker-compose.yml ps

# 查看资源占用（CPU/内存）
docker stats
```

### 11.4 进入容器

```bash
# 进入 app 容器 shell
docker compose -f deployments/docker/docker-compose.yml exec app sh

# 进入 postgres 数据库
docker compose -f deployments/docker/docker-compose.yml exec postgres psql -U omcgo
```

### 11.5 手动执行数据库迁移

```bash
# 重新运行迁移（migrate 容器已退出，需 run 启动新实例）
docker compose -f deployments/docker/docker-compose.yml run --rm migrate
```

### 11.6 切换运行环境

```bash
# 使用生产环境配置启动
OMCGO_ENV=prod docker compose -f deployments/docker/docker-compose.yml up -d

# 或在 .env 文件中设置
echo "OMCGO_ENV=prod" > .env
docker compose -f deployments/docker/docker-compose.yml up -d
```

---

## 12. 测试环境

`docker-compose.test.yml` 提供轻量级测试基础设施，**仅包含三个基础服务**，不含任何应用服务。

### 12.1 与生产配置的区别

| 项目          | docker-compose.yml  | docker-compose.test.yml |
|--------------|---------------------|-------------------------|
| 包含应用服务   | 是                  | **否**（仅基础设施）       |
| PostgreSQL 端口 | `5432`           | **`5433`**（避免冲突）    |
| Redis 端口    | `6379`              | **`6380`**              |
| NATS 端口     | `4222`              | **`4223`**              |
| PostgreSQL 存储 | Docker 命名卷     | **tmpfs 内存盘**（极速）  |
| 健康检查间隔   | 5s                  | **2s**（更快就绪）        |

### 12.2 启动测试环境

```bash
# 在项目根目录 goomc/ 下执行

# 启动测试基础设施（等待所有服务健康）
docker compose -f deployments/docker/docker-compose.test.yml up -d --wait

# 运行集成测试（示例）
cd omcgo
go test ./...

# 测试完成后销毁（tmpfs 数据本就不持久，-v 清理网络等）
docker compose -f deployments/docker/docker-compose.test.yml down -v
```

### 12.3 测试环境连接信息

| 服务         | 连接地址                                                    |
|-------------|-------------------------------------------------------------|
| PostgreSQL  | `postgres://omcgo_test:omcgo_test@localhost:5433/omcgo_test` |
| Redis       | `redis://localhost:6380`                                    |
| NATS        | `nats://localhost:4223`                                     |

---

## 13. 故障排查

### 13.1 服务启动失败

**现象**：某服务 `Exited` 或 `unhealthy`

```bash
# 查看具体错误信息
docker compose -f deployments/docker/docker-compose.yml logs <服务名>

# 查看容器退出原因
docker inspect <容器ID> | grep -A 5 '"State"'
```

### 13.2 migrate 任务失败导致应用无法启动

**原因**：migrate 服务是 acs/app/worker 的前置依赖，迁移失败会阻塞所有应用服务。

```bash
# 查看迁移日志
docker compose -f deployments/docker/docker-compose.yml logs migrate

# 手动重跑迁移
docker compose -f deployments/docker/docker-compose.yml run --rm migrate
```

### 13.3 端口冲突

**现象**：`bind: address already in use`

```bash
# 查看占用端口的进程（macOS/Linux）
lsof -i :5432
lsof -i :6379

# 停止冲突进程后重新启动
docker compose -f deployments/docker/docker-compose.yml up -d
```

**常见冲突**：本机已安装 PostgreSQL（5432）或 Redis（6379），修改 compose 中的宿主机端口即可。

### 13.4 构建失败：COPY 文件不存在

**现象**：`COPY failed: file not found`

**原因**：所有 Dockerfile 的 build context 设置为项目根目录 `goomc/`（`context: ../..`），必须在 `goomc/` 根目录下执行 compose 命令。

```bash
# 错误示例（在 docker 目录执行）
cd deployments/docker && docker compose up  # ❌

# 正确示例（在项目根目录执行）
cd /path/to/goomc && docker compose -f deployments/docker/docker-compose.yml up  # ✅
```

### 13.5 镜像拉取超时 / `failed to resolve source metadata for docker.io/...`

国内服务器访问 Docker Hub 超时。**daemon.json 配 mirror 不够** —— BuildKit 需要单独配 `buildkitd.toml`。
完整流程见 [REGISTRY_SETUP.md](./REGISTRY_SETUP.md)。

### 13.6 容器内 DNS 解析失败 / `dial tcp: lookup xxx on 53: i/o timeout`

不是 mirror 问题（端口 53 是 DNS，端口 443 才是 mirror）。见 [DOCKER_DNS_FIX.md](./DOCKER_DNS_FIX.md)。

### 13.7 前端页面空白 / API 请求失败

```bash
# 检查 nginx 配置是否有语法错误
docker compose -f deployments/docker/docker-compose.yml exec web nginx -t

# 查看 nginx 错误日志
tail -f run/logs/nginx/app_error.log

# 确认 app 服务正常运行
docker compose -f deployments/docker/docker-compose.yml ps app
```

### 13.6 MinIO 无法访问

```bash
# 检查 MinIO 健康状态
curl http://localhost:9000/minio/health/live

# 查看 MinIO 日志
docker compose -f deployments/docker/docker-compose.yml logs minio
```

### 13.7 数据卷数据清除与重置

```bash
# 查看现有数据卷
docker volume ls | grep goomc

# 完全重置（删除所有数据，慎用！）
docker compose -f deployments/docker/docker-compose.yml down -v
docker compose -f deployments/docker/docker-compose.yml up -d --build
```

### 13.8 时区问题

所有容器均挂载宿主机 `/etc/localtime`（只读）并设置 `TZ=Asia/Shanghai`。若日志时间不正确：

```bash
# 确认宿主机时区
date
ls -la /etc/localtime

# Linux 宿主机若无 /etc/localtime，可手动链接
sudo ln -sf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime
```

---

## 14. 资源限制与优雅关闭

为防止单个容器内存泄漏 / 突发负载耗尽宿主机资源拖垮整栈，所有服务在
`docker-compose.yml`（dev）与交付包 `deployments/release/bundle/deploy/docker-compose.{infra,app,monitoring,web}.yml`（prod）
中均声明了 `deploy.resources.limits / reservations`（Compose-spec 字段，`docker compose up`
直接生效），有状态服务与应用进程另配 `stop_grace_period` 保证优雅退出窗口。

> 端口表与基础资源建议见本文件 §2 / §3；此处只补「每服务限额 + 优雅关闭窗口」的取舍依据。

> 🆕 **不想每次换机器手改这些限额？** dev compose 的资源旋钮已全部参数化为 `${VAR:-默认}`
> （默认 = 下表历史值，不传 env 行为不变）。跑 `bash deployments/docker/plan-resources.sh` 按本机
> Docker 引擎（Docker Desktop 下是 VM）可用资源自动算好 `resources.env`，再用包装器
> `bash deployments/docker/dc.sh up -d --build` 一键起栈（自动带 `--env-file`）。
> 详见 [RESOURCE-PLANNING.md](./RESOURCE-PLANNING.md)。

### 14.1 资源限额一览（dev 与 prod 同值）

| 服务 | CPU limit | mem limit | CPU reserve | mem reserve | 说明 |
|------|-----------|-----------|-------------|-------------|------|
| postgres | 2 | 2g | 0.5 | 512m | 数据面主库，TimescaleDB 聚合 / WAL 缓冲需较大内存 |
| redis-core | 2 | 4g | 0.25 | 256m | 会话 / 任务 / 告警 / 核心缓存 |
| redis-pm | 2 | 8g | 0.25 | 512m | PM 双小时窗口与 KPI 状态 |
| nats | 1 | 512m | 0.25 | 256m | JetStream 消息，store 落盘 |
| minio | 1 | 1g | 0.25 | 512m | 对象存储，multipart 缓冲 |
| app | 2 | 1g | 0.5 | 512m | REST + gRPC 主进程 |
| acs | 2 | 1g | 0.5 | 512m | TR-069 长连接，高并发会话 |
| worker | 1.5 | 768m | 0.5 | 384m | PM/MR 文件处理、KPI 计算 |
| web (nginx) | 1 | 512m | 0.25 | 128m | 静态资源 + 反代 |
| prometheus | 1 | 1g | 0.25 | 256m | TSDB 15d 保留 |
| grafana | 1 | 512m | 0.25 | 256m | 可视化 |
| loki / otelcol | 1 | 512m | 0.25 | 256m | 日志 / trace 转发 |
| tempo | 1 | 1g | 0.25 | 256m | trace 存储；`GOMEMLIMIT=768MiB` 约束块压缩峰值 |
| alertmanager | 0.5 | 512m | 0.25 | 256m | 告警路由 |
| migrate-schema / migrate-seed | 1 | 512m | 0.25 | 128m | 一次性任务，限额防失控 |
| nats/nginx-exporter · node-exporter | 0.25 | 128m | 0.05 | 32m | 轻量 exporter |
| cadvisor | 0.5 | 256m | 0.1 | 64m | 容器指标采集 |

> 以上为「单机全栈 dev / 小规模 prod」基线，宿主机建议 ≥ 8 GB 内存。10 万→100 万基站
> 扩展时按实际压测调高 app/acs/postgres 上限，或拆分独立宿主机。

### 14.2 优雅关闭窗口（stop_grace_period）

`stop_grace_period` = docker 发出 `SIGTERM` 到强制 `SIGKILL` 之间的等待时长。配置原则：

- **app / acs / worker = 30s**：与 Go 端 `shutdown_timeout`（见 `cmd/*/etc/config.*.yaml`，默认 `30s`）
  对齐。进程收到 `SIGTERM` 后由 `internal/core/components/shutdown.go` 的 `GracefulShutdown`
  按优先级（HTTP→NATS→Redis→DB→MinIO）依次关闭，内部超时 30s；docker 给足同等窗口，
  确保在被 `SIGKILL` 前能跑完优雅关闭，不丢在途请求 / 任务。
- **postgres = 60s**：留足 checkpoint flush + WAL 落盘，避免被强杀截断触发下次启动 recovery。
- **redis / nats / minio / prometheus / loki / tempo = 30s**：各自完成 AOF/JetStream/对象/
  TSDB-WAL/chunk 落盘。
- **一次性任务（migrate-*）与无状态 exporter**：自行退出 / 无落盘，不单独配 `stop_grace_period`
  （沿用 docker 默认 10s）。

> 若调整 Go 端 `shutdown_timeout`，必须同步调整 app/acs/worker 的 `stop_grace_period`，
> 两者错配会导致优雅关闭被 docker 提前 `SIGKILL` 打断。

## 15. MML TXT 脚本导入切换（Task 13）

本功能使用 `omcgo/migrations/000016_redesign_mml_script_txt_import.sql`。迁移是
不可逆数据切换：先删除 `device_tasks WHERE source = 'mml'`、`mml_tasks` 和
`mml_scripts`，再增加 TXT 摘要、校验版本、`plan_items` 和 `validation_summary`
字段；Down 只撤销字段和索引，不恢复旧数据。发布前必须先停止 MML 新建、调度和消费，
并记录非 MML `device_tasks` 数量。

### 15.1 导入、保存、执行接口

```text
POST /api/v1/mml/scripts/import/validate  (multipart file=.txt)
POST /api/v1/mml/scripts/import          (JSON validation_token + metadata)
POST /api/v1/mml/scripts/:id/executions  (JSON scheduling/retry policy only)
GET  /api/v1/mml/tasks/:id/results
```

校验成功返回一次性 `validation_token`；保存成功返回 `201` 和 `script.id`，同一令牌
重放返回 `409`（`MML_IMPORT_TOKEN_CONSUMED`）。非法 TXT 返回 `422`
（`MML_SCRIPT_VALIDATION_FAILED`）且不返回保存令牌。执行任务从服务端脚本快照复制
`plan_items`，结果应保留 `plan_line_no`、`plan_device_sn`、`plan_order` 和
`script_content_sha256`。

可重复的端到端脚本及固定样例位于：
`omcgo/scripts/e2e_mml_script_import.sh`、
`omcgo/internal/mml/testdata/import-valid.txt` 和 `import-invalid.txt`。
运行时提供 `OMC_TOKEN`（或 `AUTH_TOKEN`）以及已注册设备的 `MML_E2E_SN`：

```bash
OMC_TOKEN="$TOKEN" MML_E2E_SN="<provisioned-sn>" \
  bash omcgo/scripts/e2e_mml_script_import.sh http://localhost:8081
```

脚本不会在缺少服务、凭据或设备时静默跳过，而是输出实际 HTTP 响应并以非零状态结束。

### 15.2 切换与回滚边界

```bash
# 1. 停止 MML 新建、调度和消费
# 2. 只读统计 Redis 中 source=mml 的排队任务
(cd omcgo && go run ./cmd/omcctl mml reset-script-data --dry-run)
# 3. 复核数量后才允许执行（禁止 FLUSHDB）
(cd omcgo && go run ./cmd/omcctl mml reset-script-data \
  --apply --confirm DELETE-MML-RUNTIME
)
# 4. 应用迁移、启动 migrate/app/worker/web，再执行健康检查和 E2E
bash omcgo/scripts/check-migrations.sh --strict
curl -fsS http://localhost:8081/healthz
```

`reset-script-data` 的 `--apply` 必须同时带有精确确认串；PostgreSQL 清理由迁移负责，
不会清理其他来源的设备任务。该迁移不提供数据恢复回滚，若发布中止只能恢复数据库快照，
然后重新执行迁移前的验证和切换演练。
### PM 稀疏存储与小时汇总

新部署将 PM 真实值写入 `pm_metric_values`，缺失但受支持的指标由
`pm_measurement_anchors` 和不可变指标集在查询时恢复。小时汇总按整桶版本发布：
构建中的版本不可见，全部设备批次完成后一次切换为 active。

- `PM_HOURLY_BATCH_DEVICES`：每个小时批次的设备数，默认 `2500`；该值已按 10000
  基站压测下“一小时内完成”和 worker 1 GiB 内存限制联合校准。
- `PM_LATE_DATA_WINDOW`：迟到数据与压缩安全窗口，默认 `168h`，不改变原 7 天压缩等待。

### PM 在线流式聚合

小时、日、周、月聚合由 worker 在 PM 文件完成 15 分钟 Counter/KPI 入库后直接消费同一份
标准事件，不再回查原始 PM 表。任务创建或更新后从下一个完整聚合窗口生效，不补算历史。

- Redis 必须开启 AOF、`appendfsync everysec` 和 `maxmemory-policy noeviction`；worker 启动时会校验。
- `PM_AGGREGATION_ENABLED`：是否启用在线聚合，默认 `true`。
- `PM_AGGREGATION_CLOSE_GRACE`：小时缺数据窗口的最小关闭宽限，默认 `12m`。到期后仍需等待聚合事件水位满足关闭条件。
- `PM_AGGREGATION_OUTBOX_BATCH`：发布/拉取批量，默认 `100`。
- `PM_AGGREGATION_CONSUMER_CONCURRENCY`：标准事件消费并发，默认 `8`。
- `PM_AGGREGATION_FINALIZE_CONCURRENCY`：窗口落库并发预算，默认 `32`；用于保证 2 万设备窗口在水位到达后及时收口。
- `PM_AGGREGATION_MAX_EVENT_BYTES`：单个标准事件上限，默认 `8MiB`，必须小于 NATS `max_payload`。
- `PM_AGGREGATION_WINDOW_TTL`：未知粒度的 Redis 窗口兜底 TTL；小时/日/周/月分别固定为
  `4h`、`72h`、`14d`、`45d`。
- `PM_AGGREGATION_REDIS_V2_WRITE_ENABLED`：Redis v2 紧凑写入门禁，默认 `true`。当前版本支持
  v1/v2 双读并会在后续事件到达时迁移旧窗口；仅在回退到仍具备 v1/v2 双读能力的版本时显式设为
  `false`，禁止回退到不识别 v2 的旧二进制。

JetStream 按计算层级拆分，均使用 LimitsPolicy、S2 压缩、10GiB 硬容量上限和
`DiscardOld`：`PM_AGG_15M` 保留 2 小时，`PM_AGG_HOURLY` 保留 48 小时，
`PM_AGG_DAILY` 保留 40 天。15 分钟事件只进入小时窗口；小时完成后通过事务 outbox
发布紧凑 Counter 状态给日窗口；日完成后发布紧凑 Counter 状态给周和月。周、月没有
下游，不再产生 Rollup 事件。

故障恢复顺序：先恢复 NATS 和 Redis，再启动 worker。小时窗口重放最近 2 小时的 15 分钟
事件，日窗口重放小时 Rollup，周/月窗口重放日 Rollup；紧凑流不足时允许读取
`pm_aggregation_counter_rollups` 快照，但正常计算不扫描原始 PM 表。无法恢复的窗口标为
failed 并告警，不伪造完整结果。
- PM 上传背压以 MinIO 所在文件系统的已用空间加“已接收但尚未物化”的 PM 文件预计
  入库量计算；同文件系统上的 TSDB 临时文件、WAL 和文件预分配已包含在实际已用空间，
  不重复累加。默认预计入库放大系数为保守的 `1.0`。
- 数据库 release 默认只保留 warning/error/fatal，关闭 checkpoint、autovacuum、SQL、
  慢查询、连接和临时文件逐条诊断日志；可用对应 `PG_LOG_*` / `TSDB_LOG_*` 环境变量临时开启。
- PostgreSQL/TimescaleDB 的 Docker JSON 日志默认按 `20m × 5` 轮转。

PM 重建是破坏性操作，但 **不得直接删除整个 TimescaleDB 数据卷**：该卷同时保存
告警历史、MR、trace、adhoc、设备组汇总以及可能由用户维护的 KPI 定义。发布时应先备份
这些非 PM 对象，再仅清理并重建下列 PM 对象：

- `pm_files`、`pm_ingest_batches`、`pm_metric_dictionary`、`pm_metric_sets`；
- `pm_measurement_anchors`、`pm_metric_values`；
- `pm_hourly_bucket_versions`、`pm_hourly_rollup_batches`、
  `pm_hourly_anchors`、`pm_hourly_values`；
- 旧 PM 明细/小时对象（如目标环境仍为旧 schema）。

执行前必须再次核对服务器、数据库和上述精确对象清单，并确认原始 PM 文件可重放。
只有在已经单独备份并验证恢复所有非 PM 对象、且用户再次明确确认精确卷名时，才允许
选择整卷重建；不得把主库业务数据卷纳入任何删除范围。

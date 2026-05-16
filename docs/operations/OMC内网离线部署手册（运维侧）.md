# OMC 内网离线部署手册（运维侧）

> **适用对象**：现场实施工程师、运维工程师。
> **适用场景**：OMC 无线网管系统交付到运营商**内网环境**，目标网络**不通公网**。
> **配套交付物**：离线交付包 `omc-release-<版本号>-<架构>.tar.gz`。
> **配套文档**：交付包的制作见《OMC离线交付包构建手册（构建侧）》（运维侧无需关心）。
> **文档状态**：v1.0，需随产品版本迭代同步维护。

---

## 1. 概述

OMC 采用「构建侧编译、运维侧只跑二进制」的交付模式。**运维侧拿到的是一个自包含的
离线交付包**，照本手册第 5 章逐步执行即可把系统跑起来：

- **不需要 Go**、不需要 130 个 Go 三方组件、不需要 Node/npm——交付物是已编译的静态二进制。
- **不需要联网**——所有依赖（含 Docker 引擎、基础设施镜像）都在交付包内。
- 唯一需要在内网安装的"基础软件"是 **Docker**，它本身也打进交付包离线安装。

| 组件 | 交付形态 | 运维侧操作 |
|------|---------|-----------|
| omcgo-app / omcgo-acs / omcgo-worker | 已编译静态二进制 | systemd 托管运行 |
| omcgo-migrate / omcgo-seed、omcctl | 已编译二进制（运维工具） | 按需执行 |
| 前端 webcode | 已编译静态资源 | nginx 容器提供 |
| PostgreSQL/TimescaleDB、Redis、NATS、MinIO | Docker 镜像离线包 | `docker load` 导入 |
| Docker 引擎 | 静态二进制 + 安装脚本 | 离线安装 |

---

## 2. 部署架构与端口

OMC 由 **3 个业务进程 + 1 个前端 + 4 个基础设施 + 可选监控栈**组成。推荐拓扑：

```
┌──────────────────────────── 单台/多台 Linux 服务器（内网）────────────────────────────┐
│                                                                                      │
│   宿主机进程（二进制 + systemd 托管）            Docker 容器（基础设施）               │
│   ┌───────────────────────────────┐            ┌──────────────────────────────────┐  │
│   │ omcgo-app    :8081 /:8444 TLS │            │ postgres (TimescaleDB pg16) :5432 │  │
│   │              :50051 gRPC      │───┐        │ redis                       :6379 │  │
│   │              :9091 metrics    │   │        │ nats (JetStream)            :4222 │  │
│   ├───────────────────────────────┤   ├───────▶│ minio                  :9000/:9001│  │
│   │ omcgo-acs    :7557 /:7558 TLS │   │        └──────────────────────────────────┘  │
│   │              :3478 STUN       │   │                                              │
│   │              :9090 metrics    │   │        Docker 容器（前端 / 可选监控）         │
│   ├───────────────────────────────┤   │        ┌──────────────────────────────────┐  │
│   │ omcgo-worker :9092 metrics    │───┘        │ nginx(web)            :8080/:8081 │  │
│   └───────────────────────────────┘            │ prometheus/grafana/loki ... 可选  │  │
│                                                 └──────────────────────────────────┘  │
└──────────────────────────────────────────────────────────────────────────────────────┘
        ▲                                   ▲
        │ 北向 OSS                            │ 南向 TR-069（CPE 基站经 :7557 回连 ACS）
```

设计取舍：

- **基础设施容器化**：PostgreSQL（含 TimescaleDB 扩展）、Redis、NATS、MinIO 原生安装版本敏感、依赖复杂，统一用官方镜像容器化。
- **OMC 三进程跑宿主二进制 + systemd**：静态编译无运行时依赖；systemd 是运维最熟悉的进程托管方式（开机自启、崩溃拉起、优雅停止）。
- **前端走 nginx 容器**：静态资源 + 反向代理一体。

### 2.1 组件与端口清单

| 类别 | 组件 | 默认端口 | 说明 |
|------|------|---------|------|
| OMC | omcgo-app | 8081(HTTP) / 8444(TLS) / 50051(gRPC) / 9091(metrics) | 管理面 REST + 北向 |
| OMC | omcgo-acs | 7557(HTTP) / 7558(TLS) / 3478(STUN) / 9090(metrics) | TR-069 ACS，CPE 接入 |
| OMC | omcgo-worker | 9092(metrics) | 后台 PM/MR/KPI 处理 |
| 前端 | nginx(web) | 8080 / 8081 | 静态资源 + `/api` 反代到 app |
| 基础设施 | postgres(TimescaleDB) | 5432 | 业务库 + 时序库 |
| 基础设施 | redis | 6379 | 会话/缓存/命令队列 |
| 基础设施 | nats | 4222 / 8222 | JetStream 消息 / 监控 |
| 基础设施 | minio | 9000 / 9001 | 对象存储 / 控制台 |
| 监控(可选) | prometheus / grafana / alertmanager / loki | 9090 / 3030 / 9093 / 3100 | 指标 + 日志 + 告警 |

---

## 3. 交付包内容说明

### 3.1 交付包目录结构

> **每个版本提供两个交付包**：`omc-release-<版本>-amd64.tar.gz` 与 `-arm64.tar.gz`。
> 两包结构相同，仅 `bin/` 二进制与 `images/` 镜像为对应架构；按目标机架构二选一。

```
omc-release-<版本>-<架构>/
├── README.md                      # 交付包说明 + 版本号
├── VERSION                        # 版本号 / 构建时间 / git commit
├── checksums.sha256               # 全部文件 SHA256，用于校验完整性
│
├── docker/                        # ① Docker 引擎离线安装
│   ├── docker-<ver>.tgz           #   Docker 静态二进制包
│   └── install-docker.sh          #   离线安装脚本
├── images/                        # ② Docker 镜像离线包
│   ├── infra-images-<架构>.tar    #   postgres/redis/nats/minio/nginx
│   ├── monitoring-images-<架构>.tar #  监控栈（可选）
│   └── images.manifest            #   镜像清单
├── bin/                           # ③ OMC 已编译二进制
│   └── omcgo-app / omcgo-acs / omcgo-worker / omcgo-migrate / omcgo-seed / omcctl
├── web/dist/                      # ④ 前端已编译静态资源
├── etc/                           # ⑤ 配置模板（app/acs/worker.prod.yaml，需现场修改）
├── data/                          # ⑥ 启动期加载的字典 XML
├── configs/                       # ⑦ Casbin RBAC 模型 casbin_model.conf
├── migrations/                    # ⑧ 数据库迁移 SQL（schema + seed/）
├── deploy/                        # ⑨ 部署脚本与模板
│   ├── docker-compose.infra.yml / docker-compose.web.yml / .env
│   ├── nginx.conf / default.conf
│   ├── systemd/                   #   omcgo-app/acs/worker.service 模板
│   └── healthcheck.sh             #   健康检查脚本
└── docs/
    └── OMC内网离线部署手册（运维侧）.md   # 本文档
```

### 3.2 关键内容说明

| 目录 | 说明 |
|------|------|
| `bin/` | OMC 核心二进制，静态编译，直接运行 |
| `etc/*.prod.yaml` | 配置**模板**，含开发默认值，部署时**必须修改**密码与连接地址 |
| `data/` `configs/` `migrations/` | 与 `bin/` **强绑定同版本**，严禁跨版本混用 |
| `images/` | 基础设施 Docker 镜像，`docker load` 导入 |

---

## 4. 目标环境要求与交付前检查

### 4.1 硬件（按 10 万基站规模基线）

| 资源 | 最小 | 推荐 | 说明 |
|------|------|------|------|
| CPU | 8 核 | 16 核+ | ACS 热路径 + Worker 并发 |
| 内存 | 16 GB | 32 GB+ | 基础设施 + 三进程 |
| 磁盘 | 200 GB SSD | 1 TB SSD+ | PM/MR/固件文件约 500 GB/月增长 |
| 网络 | 千兆 | 万兆 | 南向 CPE + 北向 OSS |

### 4.2 操作系统与内核

- **架构**：x86-64（amd64）或 ARM64。每个版本均提供两种架构交付包，按目标机
  `uname -m` 选用（`x86_64`→amd64，`aarch64`→arm64）。
- **发行版**：主流 Linux（CentOS 7+/Rocky/Ubuntu 20.04+/麒麟 V10/统信 UOS 等）。
- **内核**：≥ 3.10（Docker 要求），推荐 ≥ 4.x。
- **systemd**：托管 OMC 三进程与 dockerd。
- 关闭或正确配置 **SELinux / 防火墙**（见 §9 #6）。

### 4.3 端口规划

部署前确认 §2.1 全部端口在目标机**未被占用**，且内网防火墙策略放行：

- CPE 基站 → ACS：`7557`（及 `7558` TLS、`3478` STUN）。
- 浏览器 → 前端：`8080`/`8081`。
- 基础设施端口（5432/6379/4222/9000…）建议仅**本机回环**可达。

### 4.4 交付前环境检查清单

实施工程师进场前与客户确认（任一不满足需提前协调）：

- [ ] 目标服务器架构（amd64 / arm64）与数量
- [ ] 操作系统发行版与内核版本
- [ ] 内核 IP 转发可启用（`net.ipv4.ip_forward`，见 §5 步骤 2.1）
- [ ] 是否已安装 Docker；若有，版本是否满足（≥ 20.10）
- [ ] root 或 sudo 权限
- [ ] §2.1 端口是否空闲、防火墙是否可放行
- [ ] 磁盘容量与挂载点（数据盘路径）
- [ ] 是否有内网 NTP 时间源（见 §9 #3）
- [ ] 是否有内网 Harbor 镜像仓库（见 §9 #11）
- [ ] CPE 基站侧的 ACS URL 指向与回连网络是否打通
- [ ] 是否需要 TLS / 企业 CA 证书（见 §6.3）
- [ ] PostgreSQL 用客户现有实例还是容器内自建（见 §9 #9）

---

## 5. 部署操作手册（分步执行）

> 以下命令默认以 **root** 执行。安装根目录约定为 `/opt/omc`，数据盘约定为 `/data`。
> 全程**无需联网**。先读 §5.0 目录布局约定，再按步骤执行。

### 5.0 目录布局与版本管理约定

OMC 采用**「版本目录 + `current` 软链」**方式存放**不同版本的二进制**，便于保留历史
版本、秒级切换、快速回滚——每个交付版本独立一个目录，互不覆盖；`current` 软链指向
当前运行版本。

```
/opt/omc/
├── releases/                       # 各版本独立目录，互不覆盖
│   ├── 1.0.0/                      #   一个交付包解压成一个版本目录
│   │   ├── bin/  web/  data/  configs/  migrations/  deploy/  docker/  images/
│   │   └── VERSION
│   ├── 1.1.0/
│   └── 1.2.0/                      #   ← 最新版本
├── current  ->  releases/1.2.0     # 软链：指向"当前运行版本"，升级/回滚只切它
├── etc/                            # 实例配置（跨版本保留，不随交付包覆盖）
│   ├── app.prod.yaml  acs.prod.yaml  worker.prod.yaml
│   └── keys/                       #   登录 RSA 私钥等，绝不随版本走
├── run/logs/                       # 运行日志（跨版本保留）
└── packages/                       # （可选）交付包 tar.gz 原始存档备查
```

**三类内容的存放原则**：

| 类别 | 内容 | 存放位置 | 升级时行为 |
|------|------|---------|-----------|
| **版本相关** | `bin/`、`web/`、`data/`、`configs/`、`migrations/`、`deploy/` | `releases/<版本>/` | 新版本进新目录，旧目录保留 |
| **实例配置** | `*.prod.yaml`、`keys/`（站点密码/IP/密钥） | `/opt/omc/etc/` | 不覆盖；新版本模板在 `releases/<版本>/etc/`，按需 diff 合并 |
| **持久数据** | 运行日志；数据库 / MinIO 数据（docker 卷内） | `/opt/omc/run/`、docker volume | 不随版本走 |

**关键约定**：

- **`current` 软链 = 当前运行版本**。systemd 单元、nginx、脚本一律走 `/opt/omc/current/...`，**不写死版本号**。
- **`data/`、`configs/`、`migrations/` 必须与 `bin/` 同版本**——字典 XML、Casbin 模型、迁移 SQL 都与二进制强绑定，**严禁跨版本混用**。
- **升级** = 解压新版本目录 → 跑迁移 → 切 `current` 软链 → 重启；**回滚** = `current` 切回旧版本目录 → 重启（详见 §8）。切换是原子的 `ln -sfn`，秒级生效。
- **保留最近 3 个版本**目录，更早的可 `rm -rf` 回收磁盘。
- 下文 `/opt/omc/current/` 即"当前版本目录"，`/opt/omc/etc/` 为跨版本保留的实例配置目录。

### 步骤 1 — 上传与解压交付包

```bash
# 先确认目标机架构，选用对应交付包：x86_64→amd64，aarch64→arm64
uname -m

VER=<版本号>            # 例如 1.2.0，与交付包 VERSION 文件一致
ARCH=<架构>             # amd64 或 arm64

# 按 §5.0 布局：本版本解压到独立的 releases/<版本>/ 目录
mkdir -p /opt/omc/releases/$VER /opt/omc/etc /opt/omc/run/logs /opt/omc/packages
cd /opt/omc/releases/$VER
# 将 omc-release-$VER-$ARCH.tar.gz 上传至此（架构须与上面 uname -m 匹配）
sha256sum -c omc-release-$VER-$ARCH.tar.gz.sha256        # 校验完整性，必须 OK
tar xzf omc-release-$VER-$ARCH.tar.gz --strip-components=1
sha256sum -c checksums.sha256                             # 校验包内文件
mv omc-release-$VER-$ARCH.tar.gz /opt/omc/packages/       # 原始包存档备查（可选）

# 把本版本设为"当前版本"——current 软链是后续所有步骤的统一入口
ln -sfn /opt/omc/releases/$VER /opt/omc/current
```

### 步骤 2 — 主机预配置与离线安装 Docker

#### 2.1 启用内核 IP 转发（**必做**）

Docker 的 bridge 容器网络、容器间通信、宿主端口映射均依赖 Linux **IP 转发**。
加固过的 / 信创 Linux 常默认 `net.ipv4.ip_forward = 0`——此时容器无法联网、
端口映射失效，启动容器会报：
`WARNING: IPv4 forwarding is disabled. Networking will not work.`

```bash
# (1) 加载网桥过滤内核模块，并设为开机自动加载
modprobe br_netfilter
echo br_netfilter > /etc/modules-load.d/br_netfilter.conf

# (2) 持久化内核参数
cat > /etc/sysctl.d/99-omc.conf <<'EOF'
# OMC 部署所需 —— Docker 容器网络 / 端口转发依赖
net.ipv4.ip_forward = 1
net.bridge.bridge-nf-call-iptables = 1
net.bridge.bridge-nf-call-ip6tables = 1
EOF

# (3) 立即生效
sysctl --system

# (4) 校验：三项均须为 1
sysctl net.ipv4.ip_forward net.bridge.bridge-nf-call-iptables net.bridge.bridge-nf-call-ip6tables
```

> ⚠️ 部分环境 `firewalld` 重载后会把 `ip_forward` 重置为 0。若使用 firewalld，
> 需确认其 IP 伪装/转发策略已开启（`firewall-cmd --add-masquerade --permanent`），
> 或在每次 firewalld 重载后重新执行 `sysctl --system`。

#### 2.2 离线安装 Docker

> 若目标机**已装** Docker 且版本满足（≥ 20.10），跳过本小节。

`install-docker.sh` **全程离线、不联网下载**——它只解压交付包 `docker/` 目录内
**随包带来的** `docker-<版本>.tgz`（Docker 官方静态二进制，构建侧已预先下载好）。
若该目录缺 `docker-*.tgz`，脚本会直接报错而非联网，需联系交付方补齐交付包。

```bash
cd /opt/omc/current/docker
bash install-docker.sh        # 解压随包的 docker-*.tgz 到 /usr/local/bin，装 systemd 单元
systemctl enable --now docker
docker version                # 确认 Client/Server 均正常
```

### 步骤 3 — 导入 Docker 镜像

```bash
cd /opt/omc/current/images
docker load -i infra-images-<架构>.tar
docker load -i monitoring-images-<架构>.tar      # 若部署监控
docker images                                     # 对照 images.manifest 核对
```

### 步骤 4 — 规划与修改配置（**安全关键，必做**）

配置模板随版本走（在 `/opt/omc/current/etc/`），实例配置统一放**跨版本保留**的
`/opt/omc/etc/`。首次部署先复制模板再修改：

```bash
# 首次部署：把本版本配置模板复制到 etc/（-n：已存在则不覆盖，保护已有站点配置）
cp -rn /opt/omc/current/etc/. /opt/omc/etc/
```

`/opt/omc/etc/*.prod.yaml` 含开发默认值，**生产部署前必须修改**：

| 配置项 | 默认值（模板） | 必改为 |
|--------|--------------|--------|
| PostgreSQL 密码 | `omcgo123` | 强密码 |
| MinIO 账号/密码 | `minioadmin/minioadmin` | 强密码 |
| JWT 密钥 `jwt.secret` | 示例字符串 | ≥32 位随机串（多副本须一致） |
| Grafana 密码 | `admin/admin` | 强密码 |
| 各组件连接地址 | `postgres:5432` `redis:6379` `nats:4222` `minio:9000` `acs:7547` | 见下方说明 |

**连接地址说明**：OMC 三进程跑在宿主机、基础设施在容器，需把配置里的 docker 服务名改为宿主可达地址：

```yaml
# /opt/omc/etc/{app,acs,worker}.prod.yaml
db:    { dsn: "postgres://omcgo:<强密码>@127.0.0.1:5432/omcgo?sslmode=disable" }
tsdb:  { dsn: "postgres://omcgo:<强密码>@127.0.0.1:5432/omcgo?sslmode=disable" }
redis: { addrs: ["127.0.0.1:6379"] }
nats:  { url: "nats://127.0.0.1:4222" }
minio: { endpoint: "127.0.0.1:9000", access_key: "<强账号>", secret_key: "<强密码>" }
```

同步修改 `/opt/omc/current/deploy/docker-compose.infra.yml` 内 `POSTGRES_PASSWORD`、`MINIO_ROOT_*` 与配置一致。

### 步骤 5 — 启动基础设施容器

```bash
cd /opt/omc/current/deploy
# 数据卷建议落到数据盘（编辑 compose 把 volumes 指到 /data/omc/...）
docker compose -f docker-compose.infra.yml up -d
docker compose -f docker-compose.infra.yml ps        # 4 个服务均 healthy 再继续
```

容器：`postgres`（TimescaleDB pg16）、`redis`、`nats`（JetStream）、`minio`。

### 步骤 6 — 初始化数据库（迁移）

数据库迁移分 **schema（表结构）** 与 **seed（种子数据）** 两步，使用各自独立的 goose 版本表：

```bash
cd /opt/omc/current
DSN="postgres://omcgo:<强密码>@127.0.0.1:5432/omcgo?sslmode=disable"

# 6.1 表结构
./bin/omcgo-migrate --dsn "$DSN" --path migrations up

# 6.2 种子数据（独立 goose 版本表）
GOOSE_TABLE=goose_db_version_seed ./bin/omcgo-migrate --dsn "$DSN" --path migrations/seed up
```

> 迁移幂等，可重复执行。升级版本时同样先跑迁移再换二进制。

### 步骤 7 — 部署 OMC 三进程（二进制 + systemd）

`omcgo-app/acs/worker` 启动时按**相对路径**加载 `data/`（字典 XML）与 `configs/`（Casbin
模型），因此 systemd 单元的 `WorkingDirectory` **必须**设为 `/opt/omc/current`（当前
版本目录软链）。配置用绝对路径指向跨版本保留的 `/opt/omc/etc/`——升级时只切软链、不改单元。

`/opt/omc/current/deploy/systemd/omcgo-app.service` 模板：

```ini
[Unit]
Description=OMC App Service
After=network.target docker.service
Requires=docker.service

[Service]
Type=simple
WorkingDirectory=/opt/omc/current                # 关键：软链，相对路径 data/、configs/ 在此解析
ExecStart=/opt/omc/current/bin/omcgo-app --config /opt/omc/etc/app.prod.yaml
Restart=on-failure
RestartSec=5s
LimitNOFILE=65536                                 # 文件描述符上限（见 §9 #8）
Environment=OMCGO_ENV=prod
Environment=TZ=Asia/Shanghai

[Install]
WantedBy=multi-user.target
```

安装并启动（acs、worker 同理，分别指向 `acs.prod.yaml`、`worker.prod.yaml`）：

```bash
cp /opt/omc/current/deploy/systemd/omcgo-*.service /etc/systemd/system/
systemctl daemon-reload
systemctl enable --now omcgo-app omcgo-acs omcgo-worker
systemctl status omcgo-app omcgo-acs omcgo-worker     # 均 active(running)
```

> **启动顺序**：基础设施 → 迁移 → 三进程。
> systemd 单元一次安装、长期不动；升级换版本只需切 `current` 软链 + `systemctl restart`（见 §8）。

### 步骤 8 — 部署前端

```bash
cd /opt/omc/current/deploy
# docker-compose.web.yml 已挂载 web/dist + nginx.conf + default.conf
docker compose -f docker-compose.web.yml up -d
```

`default.conf` 将 `/api` 反向代理到 OMC app。OMC 三进程跑宿主机时，需把代理
`upstream` 改为 `host.docker.internal:8081`（compose 已配 `extra_hosts`）或宿主机实际 IP。

### 步骤 9 —（可选）部署监控栈

```bash
cd /opt/omc/current/deploy
docker compose -f docker-compose.monitoring.yml up -d
```

Prometheus 采集三进程 metrics（9090/9091/9092），Grafana 看板、Loki 收集日志。非必需，可后置。

### 步骤 10 — 启动校验

```bash
bash /opt/omc/current/deploy/healthcheck.sh
# 或手动：
curl -s http://127.0.0.1:8081/health        # app 健康
curl -s http://127.0.0.1:7557/              # acs 存活
curl -s http://127.0.0.1:9091/metrics | head  # metrics 暴露
docker compose -f /opt/omc/current/deploy/docker-compose.infra.yml ps   # 基础设施 healthy
```

浏览器访问 `http://<服务器IP>:8080`，用初始管理员账号登录（见交付包 `README.md`），完成首次登录改密。

### 5.11 备选：全容器化部署

若运维更倾向统一用 docker-compose 管理，可由构建侧额外把三进程二进制封装为极简镜像，
随 `images/` 交付，运维侧 `docker compose up -d` 一把启动全部服务。仍满足"交付二进制"
——只是二进制被薄镜像包裹。两方案择一，不混用。

---

## 6. 配置说明

### 6.1 安全必改项

见 §5 步骤 4 表格。**严禁**用模板默认密码上生产。

### 6.2 `*.prod.yaml` 关键项

| 段 | 项 | 说明 |
|----|----|----|
| `server` | `port` / `tls_port` / `grpc_port` | 服务监听端口 |
| `db` / `tsdb` | `dsn` / `max_conns` | 数据库连接，连接数按规模调 |
| `redis` | `addrs` / `pool_size` | Redis 地址 |
| `nats` | `url` | NATS 地址 |
| `minio` | `endpoint` / `buckets` | 对象存储；buckets 首启自动创建 |
| `jwt` | `secret` / `*_ttl` | 令牌密钥与有效期 |
| `login_crypto` | `private_key_path` / `allow_plaintext` | 登录密码 RSA 加密；HTTP 内网无 TLS 时可设 `allow_plaintext: true`（仅受信内网） |
| `batch_processor` | `workers` | Periodic Inform 批量并发，建议 = CPU 核数/2 |
| `log` | `level` / `output_paths` / `rotation` | 日志级别（prod 建议 `warn`）、路径、轮转 |
| `license` | `signing.public_key_dir` / `strict` | License 治理；strict=true 需放 OEM 公钥 |

### 6.3 TLS 与密钥

- **TLS**：默认 `tls.enabled: false`（内网 HTTP）。需加密时用 `deploy/certs/` 自签脚本生成证书，或导入企业 CA 证书，置 `enabled: true` 并填 `cert_file`/`key_file`。
- **登录 RSA 私钥**：`login_crypto.private_key_path` 指向的 PEM 需在部署时生成并放入 `/opt/omc/etc/keys/`；**多副本部署必须共用同一份**（keyID 一致）。
- 内网纯 HTTP 访问时浏览器 `crypto.subtle` 不可用，可临时启用 `allow_plaintext: true`（审计日志会标记），但推荐尽快上 TLS。

---

## 7. 验收清单

部署完成后逐项确认：

- [ ] 4 个基础设施容器状态 `healthy`
- [ ] 数据库 schema + seed 迁移均成功（`goose_db_version*` 表有记录）
- [ ] `omcgo-app/acs/worker` 三进程 `active(running)`，且重启宿主后自启
- [ ] `/health` 返回正常，`/metrics` 可抓取
- [ ] 前端页面可访问、可登录、首登改密完成
- [ ] CPE 基站能 Inform 接入 ACS（`:7557`），设备出现在设备列表
- [ ] 默认密码（PG/MinIO/JWT/Grafana/管理员）全部已改
- [ ] 日志正常写入 `/opt/omc/run/logs/`，轮转生效
- [ ] 关键端口防火墙策略已按规划放行/收敛
- [ ] 数据盘容量与告警阈值已设置
- [ ] 备份方案（见 §8）已落实并演练

---

## 8. 升级与回滚

升级/回滚基于 §5.0 的「版本目录 + `current` 软链」：新版本进新的 `releases/<版本>/`
目录，切软链即生效，旧版本目录原样留存、可秒级回退。systemd 单元一次安装后无需再改。

### 8.1 升级

```bash
VER=<新版本号> ; ARCH=<架构>
DSN="postgres://omcgo:<密码>@127.0.0.1:5432/omcgo?sslmode=disable"

# 1) 备份（不可省）：数据库 + 当前实例配置（MinIO 数据按客户备份方案另行快照）
pg_dump "$DSN" > /opt/omc/packages/db-backup-$(date +%F).sql
cp -r /opt/omc/etc /opt/omc/packages/etc-backup-$(date +%F)

# 2) 解压新版本到独立目录（current 暂不动，老版本仍在运行）
mkdir -p /opt/omc/releases/$VER && cd /opt/omc/releases/$VER
tar xzf omc-release-$VER-$ARCH.tar.gz --strip-components=1
sha256sum -c checksums.sha256

# 3) 配置：比对新版本模板有无新增项，按需手工合并到 /opt/omc/etc/（不要整体覆盖）
diff -ru /opt/omc/etc /opt/omc/releases/$VER/etc

# 4) 跑数据库迁移（用新版本的 migrations，幂等、向后兼容）
/opt/omc/releases/$VER/bin/omcgo-migrate --dsn "$DSN" --path /opt/omc/releases/$VER/migrations up
GOOSE_TABLE=goose_db_version_seed /opt/omc/releases/$VER/bin/omcgo-migrate \
  --dsn "$DSN" --path /opt/omc/releases/$VER/migrations/seed up

# 5) 切 current 软链 + 重启三进程（systemd 单元无需改动）
systemctl stop omcgo-app omcgo-acs omcgo-worker
ln -sfn /opt/omc/releases/$VER /opt/omc/current
systemctl start omcgo-app omcgo-acs omcgo-worker

# 6) 前端：current 已指向新版本，重建 nginx 容器即用上新 web/dist
docker compose -f /opt/omc/current/deploy/docker-compose.web.yml up -d --force-recreate

# 7) 跑 §7 验收清单
```

### 8.2 回滚

二进制 / 前端回滚就是**把 `current` 软链切回旧版本目录**，秒级完成：

```bash
systemctl stop omcgo-app omcgo-acs omcgo-worker
ln -sfn /opt/omc/releases/<旧版本> /opt/omc/current
systemctl start omcgo-app omcgo-acs omcgo-worker
docker compose -f /opt/omc/current/deploy/docker-compose.web.yml up -d --force-recreate
```

- **配置**：用 §8.1 步骤 1 备份的 `etc-backup-*` 还原 `/opt/omc/etc/`。
- **数据库**：若新版本迁移含**破坏性变更**（删列/改类型/删表），回滚旧二进制后 schema 不兼容，须用 §8.1 步骤 1 的 `pg_dump` 备份恢复——**因此升级前备份绝不可省**；向后兼容的迁移则无需动数据库。

### 8.3 版本留存策略

- `releases/` 下**保留最近 3 个版本**目录，供回滚与问题排查。
- 清理更早版本：确认 `current` 未指向后 `rm -rf /opt/omc/releases/<旧版本>`。
- 交付包原件存档于 `/opt/omc/packages/`，可长期保留。

> 升级前务必完成 `pg_dump` + MinIO 备份，并已在测试环境演练过回滚流程。

---

## 9. 内网环境依赖与不可控因素分析

> 以下每项都可能导致部署失败，需在交付前 / 部署中逐一排除。

| # | 因素 | 风险 | 应对 |
|---|------|------|------|
| 1 | **CPU 架构** | amd64 包跑到 arm64 机器（或反之）直接无法执行 | 每个版本均提供 amd64 与 arm64 两个交付包；部署前 `uname -m` 确认架构（`x86_64`→amd64，`aarch64`→arm64）并选用对应包 |
| 2 | **操作系统/内核** | 国产化 OS（麒麟/统信）或老旧内核对 Docker、cgroup 支持差异 | 交付前确认发行版+内核；Docker 用静态二进制规避包管理器差异；信创环境提前在同型号机验证 |
| 3 | **时间同步** | 内网无公网 NTP，多机时钟漂移 → JWT 失效、TLS 校验失败、定时任务错乱、日志时序混乱 | 确认是否有内网 NTP；无则指定一台为 NTP 源；容器统一挂 `/etc/localtime`、`TZ=Asia/Shanghai` |
| 4 | **时区** | 容器/宿主时区不一致 → 时序数据、日志、调度偏差 | 全栈固定 `Asia/Shanghai`，容器挂 `/etc/localtime:ro` |
| 5 | **端口冲突** | 目标机已有服务占用 5432/6379/8081 等 | §4.4 提前核查；冲突时改 compose 端口映射 + 同步改 `*.prod.yaml` |
| 6 | **防火墙/SELinux** | iptables/firewalld 拦截容器网络与对外端口；SELinux 阻止容器挂载卷 | 放行 §2.1 端口；SELinux 设宽容模式或为卷打 `:z` 标签；记录变更供安全审计 |
| 7 | **磁盘容量** | PM/MR/固件文件约 500 GB/月增长，pgdata/miniodata 撑爆根盘 | 数据卷落独立数据盘；设容量告警；配置 PM/MR 文件生命周期清理 |
| 8 | **文件描述符上限** | 高并发下 ACS 会话 + nginx 连接耗尽 fd → 拒绝连接 | systemd `LimitNOFILE=65536`；nginx 容器 `ulimits.nofile`；调宿主 `/etc/security/limits.conf` |
| 9 | **数据库选型** | 客户要求用既有 PG 实例，但缺 TimescaleDB 扩展 → 时序表创建失败 | 优先用交付包内 TimescaleDB 容器；若用外部 PG，必须确认已装 TimescaleDB 扩展且版本兼容 pg16 |
| 10 | **容器 DNS** | 容器内 DNS 解析异常（已知痛点，见 `deployments/docker/DOCKER_DNS_FIX.md`） | OMC 跑宿主机用 IP 直连可规避；全容器化时按该文档配置 `docker-daemon-dns.json` |
| 11 | **内网镜像仓库** | 有 Harbor 则镜像分发更优；无则只能逐机 `docker load` | 交付前确认；有 Harbor 可改为推仓库 + compose 引用 |
| 12 | **镜像版本一致性** | 自行 `docker pull` 浮动标签会覆盖交付镜像、导致版本漂移 | 镜像版本已由构建侧固化；运维侧只 `docker load` 交付包内镜像，勿自行 `docker pull` |
| 13 | **CPE 回连网络** | ACS 与 CPE 基站之间 NAT/防火墙阻断 Connection Request / STUN | 确认 CPE 的 ACS URL 指向正确、`:7557`/`:3478` 可达；STUN 用于 NAT 穿透需放行 UDP |
| 14 | **多副本密钥一致** | app 多副本但 JWT/登录 RSA 密钥不一致 → 令牌跨副本失效 | 多副本共用同一份 `jwt.secret` 与 `login_crypto` 私钥 |
| 15 | **现场源码编译需求** | 个别客户安全审计要求内网内编译，而非接收二进制 | 默认不需要；如确有此要求，由构建侧按《构建手册》§8 提供离线源码+依赖包 |
| 16 | **权限** | 无 root/sudo 无法装 Docker、写 systemd | 交付前确认权限；最小权限方案需提前与客户协商 |
| 17 | **优雅停机** | 直接 kill 进程丢失会话/未刷盘数据 | 用 `systemctl stop`（进程响应 SIGTERM 优雅关闭）；`worker` 关闭超时 30s |
| 18 | **内核 IP 转发未开** | 加固/信创 Linux 默认 `net.ipv4.ip_forward=0` → Docker 容器无法联网、端口映射失效，容器启动报 `IPv4 forwarding is disabled` | 部署前按 §5 步骤 2.1 开启 `net.ipv4.ip_forward` 与 `bridge-nf-call-iptables`；firewalld 重载会重置，需配 masquerade 或重载后再 `sysctl --system` |

---

## 10. 常见故障排查

| 现象 | 排查方向 |
|------|---------|
| 三进程起不来 | `journalctl -u omcgo-app -n 100`；检查 `WorkingDirectory` 是否 `/opt/omc/current`、`current` 软链是否指向有效版本目录（否则 `data/`、`configs/` 加载失败） |
| 端点对非超管返回 500 `casbin authorizer not configured` | `configs/casbin_model.conf` 缺失或路径不对 |
| 启动报字典 XML 加载失败 | `data/` 目录缺失或 `WorkingDirectory` 不对 |
| 连不上数据库/Redis | `*.prod.yaml` 地址是否改成 `127.0.0.1`；基础设施容器是否 `healthy` |
| 迁移失败 | 检查 DSN、PG 是否就绪；schema 与 seed 用**不同** goose 版本表（`GOOSE_TABLE`） |
| 前端能开但接口 502 | nginx `default.conf` 的 upstream 是否指向正确的 app 地址/端口 |
| CPE 接不进来 | 防火墙 `:7557`；CPE 的 ACS URL；网络可达性；ACS 日志 `/opt/omc/run/logs/acs/acs.log` |
| 时间相关报错（令牌/证书） | 见 §9 #3 时间同步 |
| 容器 DNS 解析失败 | 见 §9 #10，参考 `deployments/docker/DOCKER_DNS_FIX.md` |
| 容器起不来 / 端口映射不通 / 报 `IPv4 forwarding is disabled` | 内核 IP 转发未开，按 §5 步骤 2.1 配置 `net.ipv4.ip_forward=1` |

日志位置：`/opt/omc/run/logs/{app,acs,worker,nginx}/`，zap JSON 格式，lumberjack 自动轮转。

---

**文档结束。** 端口、配置项、镜像版本以实际交付版本的 `etc/*.prod.yaml`、`deploy/` 为准。

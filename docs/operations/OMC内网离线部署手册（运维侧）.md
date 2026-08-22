# OMC 内网离线部署手册（运维侧）

> **适用对象**：现场实施工程师、运维工程师。
> **适用场景**：OMC 无线网管系统交付到运营商**内网环境**，目标网络**不通公网**。
> **配套交付物**：两个互相独立的离线交付包（可从构建机 HTTP 服务下载）——
> **项目包** `omc-<test|release>-<版本号>-amd64.tar.xz`（OMC 业务镜像 + 部署模板）与
> **基础设施包** `omc-infra-<版本号>-amd64.tar.xz`（Docker 引擎 + 基础镜像）。
> **配套文档**：交付包的制作见《OMC离线交付包构建手册（构建侧）》（运维侧无需关心）。
> **文档状态**：v3.0（全 docker compose 部署 + 双包拆分），需随产品版本迭代同步维护。
>
> ⚠️ **部署模式（v3）**：本版本为**全 docker compose 部署**——所有 OMC 服务
> （app / acs / worker / web / 监控栈）都以 **docker 容器**运行，宿主机上**不再放业务
> 二进制、不再装 systemd 单元**。`docker load` 导入镜像后用 `docker compose` 一键起全栈。
>
> ⚠️ **架构**：当前发布仅有 **amd64（x86_64）** 包。目标机 `uname -m` 应返回
> `x86_64`；若是 `aarch64`/`arm64` 等其它架构，请联系构建侧（不在本期发布范围）。
>
> **严格离线约束**：安装脚本会在启动前校验本次 Compose 的全部镜像，并使用
> `docker compose up --pull never`。缺少任一镜像（包括 `cadvisor`、各 exporter）都会直接失败，
> 不会联网拉取。首次部署必须先导入基础设施包；`--skip-infra` 仅适用于基础设施和监控镜像已经
> 在本机完成 `docker load` 的后续项目升级。

---

## 1. 概述

OMC 采用「构建侧 `docker build` 业务镜像、运维侧全 docker compose 跑容器」的交付模式。
**运维侧拿到两个互相独立的离线交付包**，照本手册第 5 章逐步执行即可把系统跑起来：

- **项目包** `omc-<test|release>-<版本>-<架构>.tar.xz`——OMC 业务镜像（app/acs/worker/web）+ 配置 + 数据库迁移 + 字典 + 部署模板 + 监控配置。发版频繁。
- **基础设施包** `omc-infra-<版本>-<架构>.tar.xz`——Docker 引擎离线安装包 + Compose v2 + Buildx + 基础镜像（+ 默认含监控栈镜像）。不常变更。

两个包**各自独立的版本号**，首次部署两个都要；之后日常升级通常只更新项目包。

- **不需要 Go**、不需要 Node/npm——业务代码已编译进 docker 镜像。
- **不需要联网**——所有依赖（含 Docker 引擎、基础设施镜像、业务镜像）都在两个交付包内。
- 唯一需要在内网安装的"基础软件"是 **Docker（含 Compose v2 / Buildx）**，它本身也打进基础设施包离线安装。

| 组件 | 交付形态 | 运维侧操作 |
|------|---------|-----------|
| omcgo app / acs / worker / web(nginx) | `docker save` 业务镜像 tar | `docker load` 导入，`docker compose up -d` 运行 |
| 数据库迁移 schema + seed | `migrations/`，goose 一次性容器执行 | `deploy.sh` 自动跑 `migrate-schema` / `migrate-seed-sql` 容器 |
| PostgreSQL/TimescaleDB、Redis、NATS、MinIO、监控栈 | Docker 镜像离线包 | `docker load` 导入 |
| Docker 引擎 / Compose v2 / Buildx | 静态二进制 + 安装脚本 | 离线安装 |

---

## 2. 部署架构与端口

OMC 由 **3 个业务容器 + 1 个前端容器 + 4 个基础设施容器 + 可选监控栈**组成，
全部由 `docker compose`（compose project = `omcgo`，网络 = `omcgo-net`）编排：

```
┌──────────────────────── 单台/多台 Linux 服务器（内网）────────────────────────┐
│   docker compose 全栈（compose project = omcgo，network = omcgo-net）          │
│                                                                              │
│   业务容器                                基础设施容器                          │
│   ┌────────────────────────────────┐    ┌────────────────────────────────┐  │
│   │ web(nginx) :8081 Web UI+REST   │    │ postgres(TimescaleDB pg16):5432 │  │
│   │            :8080 TR-069 ACS 反代│    │ redis                     :6379 │  │
│   │ app   127.0.0.1:9091 metrics   │───▶│ nats(JetStream)  :4222/127.0.0.1:8222│
│   │ acs   :7547 CPE TR-069         │    │ minio              :9000/:9001  │  │
│   │       :7557 connection-request │    └────────────────────────────────┘  │
│   │       127.0.0.1:9095 metrics   │    监控容器（默认含，可 --skip-monitoring）│
│   │ worker 127.0.0.1:9092 metrics  │    ┌────────────────────────────────┐  │
│   └────────────────────────────────┘    │ prometheus:9090 alertmanager:9093│ │
│                                          │ grafana:3030 loki:3100 tempo ... │ │
│                                          └────────────────────────────────┘  │
└──────────────────────────────────────────────────────────────────────────────┘
        ▲                                   ▲
        │ 北向 OSS（app）                     │ 南向 TR-069（CPE 基站填 :8080 作 ACS URL）
```

设计取舍：

- **全容器化**：基础设施（PG/Redis/NATS/MinIO）与业务（app/acs/worker/web）统一用 docker
  镜像 + compose 编排，开机自启、崩溃拉起由 docker `restart` 策略负责，运维只需 `docker compose`。
- **前端走 nginx 容器**：静态资源 + 反向代理一体，`:8081` 服务 Web UI 与 `/api`，`:8080` 反代 TR-069 给 acs。

### 2.1 组件与端口清单

> ⚠️ **Web 管理界面在 `:8081`**（nginx server block，`/api` 反代到 app）；**`:8080`** 是
> 给**基站设备**填的 ACS URL（TR-069 CWMP 反代到 acs），**人不浏览**。

**内网可达（0.0.0.0 绑定）**：

| 端口 | 组件 | 用途 / 使用方 |
|------|------|--------------|
| **8081** | web(nginx) | **Web 管理界面 + REST + SSE**，运维浏览器登录（默认 admin/admin123） |
| **8080** | web(nginx) | **基站连接（TR-069 ACS 反代）**，基站设备侧填作 ACS URL |
| 7547 | acs | CPE → ACS（HTTP，TR-069 标准端口；nginx :8080 反代到此） |
| 7557 | acs | 内部 connection-request 接口 |
| 5432 | postgres(TimescaleDB) | 业务库 + 时序库 |
| 6379 | redis | 会话/缓存/命令队列（当前无密码，仅受信内网可接受） |
| 4222 | nats | JetStream 客户端 |
| 9000 / 9001 | minio | S3 API / Console UI |
| 9090 / 9093 / 3030 / 3100 | 监控（可选） | prometheus / alertmanager / grafana(宿主3030→容器3000) / loki |

**仅本机回环 127.0.0.1（从工作机访问需 SSH 隧道）**：

| 端口 | 服务 | 接入方式 |
|------|------|---------|
| 9091 | app 健康 / metrics | `curl 127.0.0.1:9091/healthz` 或 `/metrics` |
| 9095 | acs 健康 / metrics | `curl 127.0.0.1:9095/healthz`（容器内是 9090） |
| 9092 | worker 健康 / metrics | `curl 127.0.0.1:9092/healthz` |
| 8222 | NATS HTTP 监控 | `http://127.0.0.1:8222` |

> ⚠️ `5432 / 6379 / 4222 / 9000 / 9001` 对内网全开（0.0.0.0）——部署前必须改强口令（见 §6.1）。
> Redis 当前无密码，仅受信任内网可接受；公网 / DMZ 须配 `requirepass` 并同步 `etc/*.prod.yaml`。
> 防火墙 / 安全组在出公网前必须 deny 这 5 个端口。

---

## 3. 交付包内容说明

交付物是**两个独立交付包**，各自一份 amd64 包。

### 3.1 项目包 `omc-<test|release>-<版本>-<架构>/`

OMC 本体。发版频繁，走"版本目录 + `current` 软链"管理（见 §5.0）。

```
omc-<test|release>-<版本>-<架构>/
├── README.md                      # 项目包说明 + 版本号 + 渠道
├── VERSION                        # 项目版本 / 渠道 / 架构 / 镜像前缀 / 构建时间 / git commit
├── checksums.sha256               # 全部文件 SHA256，用于校验完整性
│
├── images/                        # ① OMC 业务镜像（docker save）
│   └── business-images-<版本>-<架构>.tar   #   omcgo/{app,acs,worker,web}:<版本>
├── data/                          # ② 启动期加载的字典 XML（可挂载覆盖镜像内默认值）
├── configs/                       # ③ Casbin RBAC 模型等（可挂载覆盖）
├── migrations/                    # ④ 数据库迁移 SQL（schema + seed/，goose 容器执行）
├── etc/                           # ⑤ 配置模板 app/acs/worker.prod.yaml（需现场改口令）
├── deploy/                        # ⑥ 部署脚本与模板
│   ├── docker-compose.infra.yml / .app.yml / .web.yml / .monitoring.yml
│   ├── .env                       #   镜像 tag + 默认口令 + JWT（部署前必改）
│   ├── nginx.conf / default.conf
│   ├── monitoring/                #   prometheus/grafana/loki/tempo/otelcol/alertmanager 配置
│   ├── deploy.sh                  #   一键部署
│   ├── svc.sh                     #   日常启停 / 重启 / 日志
│   └── healthcheck.sh             #   健康检查脚本
└── docs/
    └── OMC内网离线部署手册（运维侧）.md   # 本文档
```

### 3.2 基础设施包 `omc-infra-<版本>-<架构>/`

Docker 引擎 + 基础镜像。不常变更，**一台机器装一次**（不随项目版本走）。

```
omc-infra-<版本>-<架构>/
├── README.md                      # 基础设施包说明 + 安装步骤
├── VERSION                        # 基础设施版本 / 架构 / Docker / Compose / Buildx 版本 / 构建时间
├── checksums.sha256               # 全部文件 SHA256
│
├── setup-mirrors.sh               # 系统加速三合一：Docker / npm / Golang 配 / 换 / 查 / 取消
├── docker/                        # ① Docker 引擎离线安装
│   ├── docker-<ver>.tgz           #   Docker 静态二进制包
│   ├── docker-compose             #   Compose v2 静态二进制
│   ├── docker-buildx              #   Buildx 静态二进制
│   └── install-docker.sh          #   离线安装脚本（装完自动调 ../setup-mirrors.sh 引导加速）
└── images/                        # ② Docker 镜像离线包（docker save）
    ├── infra-images-<架构>.tar     #   postgres / redis / nats / minio / nginx
    ├── monitoring-images-<架构>.tar #   监控栈（默认包含；用 build-images.sh --infra-only 可关闭）
    └── images.manifest             #   镜像清单
```

### 3.3 关键内容说明

| 目录 | 所属包 | 说明 |
|------|--------|------|
| `images/` | 项目包 | OMC 业务镜像 tar（`docker save`），`docker load` 导入即用 |
| `etc/*.prod.yaml` | 项目包 | 配置**模板**，含开发默认值，部署时**必须修改**密码（容器可挂载覆盖镜像内默认值） |
| `data/` `configs/` `migrations/` | 项目包 | 与业务镜像 **强绑定同版本**，严禁跨版本混用 |
| `deploy/.env` | 项目包 | 业务镜像 tag + 基础设施 / 监控镜像标签 + 默认口令 + JWT，`docker compose` 引用 |
| `docker/` | 基础设施包 | Docker 引擎 + Compose + Buildx 离线安装包 + 安装脚本 |
| `images/` | 基础设施包 | 基础设施 / 监控 Docker 镜像，`docker load` 导入 |

> **两个包版本号互相独立**：项目包版本（`omc-test/omc-release-...`）与基础设施包版本
> （`omc-infra-...`）各自演进。`deploy/.env`（项目包内）记录基础设施镜像标签，须与
> 已 `docker load` 的基础设施包镜像一致——同源同构建侧固化，正常无需关心。

---

## 4. 目标环境要求与交付前检查

### 4.1 硬件（按 10 万基站规模基线）

| 资源 | 最小 | 推荐 | 说明 |
|------|------|------|------|
| CPU | 8 核 | 16 核+ | ACS 热路径 + Worker 并发 |
| 内存 | 16 GB | 32 GB+ | 基础设施 + 业务容器 + 监控栈 |
| 磁盘 | 200 GB SSD | 1 TB SSD+ | PM/MR/固件文件约 500 GB/月增长 |
| 网络 | 千兆 | 万兆 | 南向 CPE + 北向 OSS |

### 4.2 操作系统与内核

- **架构**：x86-64（amd64）。目标机 `uname -m` 须为 `x86_64`。
- **发行版**：主流 Linux（CentOS 7+/Rocky/Ubuntu 20.04+/麒麟 V10/统信 UOS 等）。
- **内核**：≥ 3.10（Docker 要求），推荐 ≥ 4.x。
- **内核 IP 转发**：Docker 容器网络依赖，需开启（见 §5 步骤 2.1）。
- 关闭或正确配置 **SELinux / 防火墙**（见 §9 #6）。

### 4.3 端口规划

部署前确认 §2.1 全部端口在目标机**未被占用**，且内网防火墙策略放行：

- CPE 基站 → ACS：`8080`（nginx 反代到 acs `:7547`）。
- 浏览器 → Web 管理界面：`8081`。
- 基础设施端口（5432/6379/4222/9000…）建议仅**受信内网**可达，出公网前必须 deny。

### 4.4 交付前环境检查清单

实施工程师进场前与客户确认（任一不满足需提前协调）：

- [ ] 目标服务器架构（须 amd64）与数量
- [ ] 操作系统发行版与内核版本
- [ ] 内核 IP 转发可启用（`net.ipv4.ip_forward`，见 §5 步骤 2.1）
- [ ] 是否已安装 Docker；若有，版本是否满足（≥ 20.10）且含 compose v2 插件
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

> 🚀 **推荐路径**：按 §5.0 + 步骤 1（解压）+ 步骤 2（装 Docker）准备好后，直接跑
> `sudo bash /opt/omc/current/deploy/deploy.sh` **一键部署**，自动完成镜像 load /
> migrate / seed / 起全栈 / 健康检查。详见 §5.10。下面的「手动步骤」便于理解细节与排错。

### 5.0 目录布局与版本管理约定

OMC 采用**「版本目录 + `current` 软链」**方式存放**不同版本的项目包**，便于保留历史
版本、秒级切换、快速回滚。`current` 软链指向当前运行版本。基础设施包（Docker 引擎 +
镜像）一台机器只装一次，不随项目版本走。

```
/opt/omc/
├── releases/                       # 各项目版本独立目录，互不覆盖
│   ├── 0.1.0-20260518-1030/        #   一个项目包解压成一个版本目录
│   │   ├── images/ data/ configs/ migrations/ etc/ deploy/
│   │   └── VERSION
│   └── 0.1.0-20260601-0900/        #   ← 最新版本
├── current  ->  releases/0.1.0-20260601-0900   # 软链：当前运行版本，升级/回滚只切它
├── infra/                          # 基础设施包解压处（docker/ + images/），装一次
├── etc/                            # 实例配置（跨版本保留，不随交付包覆盖）
│   └── app.prod.yaml  acs.prod.yaml  worker.prod.yaml
├── data/                           # host 主权字典 XML（param-mappings / indicator-library / alarm-definitions，单目录 + .custom sidecar）
├── run/logs/{app,acs,worker,nginx}/ # 运行日志（跨版本保留）
└── packages/                       # （可选）交付包压缩档原始存档备查
```

> 数据库 / MinIO / 监控历史数据落在 **docker 数据卷**（`pgdata`/`miniodata`/`redisdata`/
> `natsdata`/`prometheusdata`/`grafanadata`/`lokidata`/`tempodata`），不随版本走、不在 `/opt/omc` 下。

**关键约定**：

- **`current` 软链 = 当前运行版本**。`deploy.sh` / `svc.sh` / compose 文件一律走 `/opt/omc/current/...`，**不写死版本号**。
- **`data/`、`configs/`、`migrations/` 必须与业务镜像同版本**——字典 XML、Casbin 模型、迁移 SQL 都与镜像强绑定，**严禁跨版本混用**。
- **升级** = 解压新版本目录 → 跑 `deploy.sh` → 切 `current` + 起新镜像容器；**回滚** = 旧版本目录重跑 `deploy.sh --skip-migrate`（详见 §8）。
- **保留最近 3 个版本**目录，更早的可 `rm -rf` 回收磁盘。

### 步骤 1 — 上传与解压交付包

首次部署需**两个包**：基础设施包解压到 `/opt/omc/infra/`（装一次），项目包解压到
版本目录 `/opt/omc/releases/<版本>/`。

```bash
# 先确认目标机架构（须 x86_64 → amd64）
uname -m

ARCH=amd64
PROJ=<项目包文件名>      # 如 omc-test-0.1.0-20260518-1030-amd64（测试阶段）
                         # 或 omc-release-1.0.0-...-amd64（正式发布）
INFRA=<基础设施包文件名> # 如 omc-infra-0.1.0-...-amd64
VER=<项目版本号>         # 项目包 VERSION 文件的 project_version，作为 releases/ 目录名

mkdir -p /opt/omc/releases/$VER /opt/omc/infra /opt/omc/etc /opt/omc/run/logs /opt/omc/packages

# (1) 基础设施包 —— 解压到 /opt/omc/infra/（一台机器装一次；已装过可跳过本步）
cd /opt/omc/infra
# 将 $INFRA.tar.xz 上传至此
sha256sum -c $INFRA.tar.xz.sha256                  # 校验完整性，必须 OK
tar xf $INFRA.tar.xz --strip-components=1          # tar 自动识别 xz/gz 压缩
sha256sum -c checksums.sha256                      # 校验包内文件

# (2) 项目包 —— 解压到独立的 releases/<版本>/ 目录
cd /opt/omc/releases/$VER
# 将 $PROJ.tar.xz 上传至此
sha256sum -c $PROJ.tar.xz.sha256
tar xf $PROJ.tar.xz --strip-components=1
sha256sum -c checksums.sha256

# 把本项目版本设为"当前版本"——current 软链是后续所有步骤的统一入口
ln -sfn /opt/omc/releases/$VER /opt/omc/current
```

### 步骤 2 — 主机预配置与离线安装 Docker

#### 2.1 启用内核 IP 转发（**必做**）

Docker 的 bridge 容器网络、容器间通信、宿主端口映射均依赖 Linux **IP 转发**。
加固过的 / 信创 Linux 常默认 `net.ipv4.ip_forward = 0`——此时容器无法联网、
端口映射失效，启动容器会报 `WARNING: IPv4 forwarding is disabled.`

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

# (3) 立即生效 +（4）校验：三项均须为 1
sysctl --system
sysctl net.ipv4.ip_forward net.bridge.bridge-nf-call-iptables net.bridge.bridge-nf-call-ip6tables
```

> ⚠️ 部分环境 `firewalld` 重载后会把 `ip_forward` 重置为 0。若使用 firewalld，
> 需确认其 IP 伪装/转发策略已开启（`firewall-cmd --add-masquerade --permanent`），
> 或在每次 firewalld 重载后重新执行 `sysctl --system`。

#### 2.2 离线安装 Docker（含 Compose v2 / Buildx）

> 若目标机**已装** Docker 且版本满足（≥ 20.10、含 compose v2 插件），可跳过本小节；
> deploy.sh 检测到只有 compose v1 时会从 infra bundle 自动补装 v2 插件。

`install-docker.sh` **全程离线、不联网下载**——它只解压基础设施包 `docker/` 目录内
**随包带来的** `docker-<版本>.tgz`、`docker-compose`、`docker-buildx`（构建侧已预先下载好）。

```bash
cd /opt/omc/infra/docker
sudo bash install-docker.sh   # 解压随包 docker 三件套，装 systemd 单元 + cli-plugins
systemctl enable --now docker
docker version                # 确认 Client/Server 均正常
docker compose version        # 确认 compose v2 插件就位
```

**自动检测 `/var` 容量**：脚本在解压二进制前会 `df` 检查 `/var` 可用空间。
**可用 < 15G 时弹出提示**，回车默认切到 `/home`（建 `/home/{docker,containerd}-data`
并改 `data-root` / containerd `--root`）；输入 `n` 继续走默认 `/var/lib`；非交互模式
（被 deploy.sh 调起）自动按 Y 切换。装完用 `docker info | grep -E 'Docker Root Dir|Containerd'` 复核。

#### 2.3 Docker 网段规划（**必看**：唯一输入为客户规划的 DOCKER_BIP）

Docker 网桥地址不能使用固定的公司示例网段。客户应根据实际业务地址、路由和安全策略
规划唯一输入 `DOCKER_BIP`；生产安装不会在未配置时自动选择网段，必须先完成客户侧规划。
客户可以规划其它不冲突的 IPv4 网段。为避免再次与公司内网冲突，当前实现拒绝整个
`172.0.0.0/8`。

规划库按 `DOCKER_BIP` 所在网络块连续派生其它 Docker 网络：

| 用途 | 派生规则 | 配置位置 |
|------|---------|---------|
| `docker0`（`bip`） | `DOCKER_BIP` 所在网络块 | `/etc/docker/daemon.json` 的 `bip` |
| `omcgo-net`（业务网） | 下一个同等大小的网络块 | `DOCKER_COMPOSE_SUBNET` |
| 自动/未来网络（池） | 再下一个同等大小的网络块，每个网络至少 `/24` | `default-address-pools` |
| 测试网络 | 再下一个同等大小的网络块 | `DOCKER_TEST_SUBNET` |
| 迁移基线网络 | 再下一个同等大小的网络块 | `DOCKER_MIGRATION_SUBNET` |

例如规划 `10.240.0.1/16`：`docker0=10.240.0.0/16`、业务网为 `10.241.0.0/16`、
自动地址池为 `10.242.0.0/16`、测试网为 `10.243.0.0/16`、迁移基线网为
`10.244.0.0/16`。

交付包部署前只需在 `deploy/.env` 填写：

```bash
DOCKER_BIP=10.240.0.1/16
```

安装脚本会计算并持久化所有派生变量，后续 Compose、`svc.sh`、`healthcheck.sh` 和
密码轮换操作均复用这份规划。不要手工填写派生变量，也不要直接覆盖
`/etc/docker/daemon.json` 丢失 `bip` 或 `default-address-pools`。

已装 Docker 修改规划时：

```bash
export DOCKER_BIP=10.240.0.1/16
cd /opt/omc/infra/docker
sudo -E bash install-docker.sh --skip-if-installed --no-mirror
```

修改前应停止使用旧网段的容器。`--fresh-install` 会删除无容器的规划外网络；仍有容器
挂载的规划外网络必须先人工处理。校验：

```bash
cat /etc/docker/daemon.json
ip -4 route
docker network ls -q | xargs -r docker network inspect --format '{{.Name}} {{range .IPAM.Config}}{{.Subnet}} {{end}}'
```

所有 Docker bridge 都必须属于当前 `DOCKER_BIP` 派生计划；只修改 Compose 文件不会改变
已经存在的 Docker 网络。

### 步骤 3 — 导入 Docker 镜像

镜像来自两个包：基础设施 / 监控镜像来自基础设施包（导一次即可，项目升级不需重导），
业务镜像来自项目包（每次升级随新版本导入）。

```bash
# (1) 基础设施 + 监控镜像（来自基础设施包，一台机器一次）
cd /opt/omc/infra/images
docker load -i infra-images-amd64.tar
docker load -i monitoring-images-amd64.tar           # 若部署监控
# (2) 业务镜像（来自项目包，随版本走）
docker load -i /opt/omc/current/images/business-images-<版本>-amd64.tar
docker images                                         # 对照 images.manifest 核对
```

> **关于 tar 内双 tag**：基础设施 tar `docker load` 后会看到两套引用——原 tag（如
> `redis:7-alpine`）与 `*-amd64-saved` 后缀 tag。**运维只需关心原 tag**，
> `docker-compose.yml` 里 `image: redis:7-alpine` 直接可用；`*-saved` tag 是构建侧
> 本地缓存的命名隔离标识，运维不需手工清理。

> 实操中**推荐直接用 `deploy.sh`**（§5.10）完成镜像 load——脚本会 `docker image inspect`
> 判断镜像是否已存在，已存在则跳过 load 并重启容器，缺失才 load，幂等安全。
> 启动前还会再次校验全部镜像；缺少任一镜像会失败并提示导入基础设施包，不会让 Compose 隐式
> pull。特别是启用监控时，必须确认 `monitoring-images-amd64.tar` 已导入，其中包含 `cadvisor`
> 和各 exporter。

### 步骤 4 — 规划与修改配置（**安全关键，必做**）

默认口令出现在**两处**，**必须同步修改**，否则 OMC 容器连不上 PostgreSQL / MinIO：

1. `/opt/omc/current/deploy/.env`——`docker compose` 起容器时的**初始口令 / 镜像版本**（仅首次 volume 创建时生效）
2. `/opt/omc/etc/{app,acs,worker}.prod.yaml`——OMC 容器连接中间件时的**客户端口令**

首次部署先把模板复制到跨版本保留的 `etc/`（deploy.sh 首次会自动做；手动如下）：

```bash
cp -rn /opt/omc/current/etc/. /opt/omc/etc/         # -n：已存在则不覆盖，保护已有站点配置
```

必改项：

| 配置项 | 默认值（.env / 模板） | 必改为 |
|--------|----------------------|--------|
| PostgreSQL 密码 | `POSTGRES_PASSWORD=omcgo123` | 强密码 |
| MinIO 账号/密码 | `MINIO_ROOT_USER/PASSWORD=minioadmin` | 强密码 |
| Grafana 密码 | `GRAFANA_ADMIN_PASSWORD=admin` | 强密码 |
| JWT 密钥 | `OMCGO_JWT_SECRET=...`（.env）/ `jwt.secret`（yaml） | ≥32 位随机串（多副本须一致） |

**容器间地址（compose 内部网络）**：业务容器与基础设施容器同在 `omcgo-net` 网络，
`*.prod.yaml` 里用 **docker 服务名**互联（无需改 IP）：

```yaml
# /opt/omc/etc/{app,acs,worker}.prod.yaml（容器内挂载覆盖镜像默认值）
db:    { dsn: "postgres://omcgo:<强密码>@postgres:5432/omcgo?sslmode=disable" }
tsdb:  { dsn: "postgres://omcgo:<强密码>@postgres:5432/omcgo?sslmode=disable" }
redis: { addrs: ["redis:6379"] }
nats:  { url: "nats://nats:4222" }
minio: { endpoint: "minio:9000", access_key: "<强账号>", secret_key: "<强密码>" }
```

> ⚠️ `.env` 中的口令与 `*.prod.yaml` 中的口令必须**完全一致**。`.env` 的口令只在
> PostgreSQL/MinIO **首次创建数据卷**时写入；若已起过容器再改 `.env` 口令，需先删数据卷
> 或进容器改库内口令，否则业务容器认证失败。详见下载页 §9「配置文件修改指南」。

### 步骤 5 — 一键起全栈（推荐）

完成步骤 1-4 后，直接跑 `deploy.sh`（§5.10），它会自动 load 镜像、起基础设施、跑
migrate/seed、起业务+web+监控、健康检查。**手动分步**仅用于理解 / 排错：

```bash
cd /opt/omc/current/deploy
COMPOSE="docker compose -p omcgo -f docker-compose.infra.yml -f docker-compose.app.yml -f docker-compose.web.yml -f docker-compose.monitoring.yml"

# 5.1 起基础设施，等 PG / Redis 就绪
$COMPOSE up -d postgres redis nats minio

# 5.2 数据库迁移（goose 一次性容器；schema 与 seed 各自独立 goose 版本表）
$COMPOSE up --exit-code-from migrate-schema migrate-schema  ; $COMPOSE rm -f migrate-schema
$COMPOSE up --exit-code-from migrate-seed-sql migrate-seed-sql ; $COMPOSE rm -f migrate-seed-sql

# 5.3 起业务 + web + 监控
$COMPOSE up -d

# 5.4 健康检查
bash /opt/omc/current/deploy/healthcheck.sh
```

> 迁移走容器内 `omcgo-migrate`（goose 幂等），**无需宿主机二进制**。`migrate-schema` /
> `migrate-seed-sql` 是 compose 中的一次性服务（`network_mode: service:postgres`，跑完即退）。

### 步骤 6 — 启动校验

```bash
bash /opt/omc/current/deploy/healthcheck.sh
# healthcheck 检查：app/acs/worker/postgres/redis/nats/minio（+web+监控）容器 running，
# 以及 5 个核心健康端点（app/acs/worker /healthz、app /metrics、前端 SPA :8081）。
```

浏览器访问 `http://<服务器IP>:8081`，用初始管理员账号登录（`admin`/`admin123`），完成首次登录改密。

### 5.10 一键部署（推荐）

完成 §5.0 目录布局 + 步骤 1（解压）+ 步骤 2（装 Docker）后，直接跑 `deploy.sh`：

```bash
sudo bash /opt/omc/current/deploy/deploy.sh                  # 全套（推荐首次）
sudo bash /opt/omc/current/deploy/deploy.sh --skip-infra     # 日常升级（基础设施镜像已 load）
sudo bash /opt/omc/current/deploy/deploy.sh --skip-monitoring # 不起监控栈
sudo bash /opt/omc/current/deploy/deploy.sh --skip-migrate   # 不跑 migrate/seed
sudo bash /opt/omc/current/deploy/deploy.sh --check-only     # 只做环境 precheck
sudo bash /opt/omc/current/deploy/deploy.sh --overwrite-etc  # 用新包模板覆盖 etc（旧 etc 自动备份）
sudo bash /opt/omc/current/deploy/deploy.sh -h               # 看全部参数
```

**9 步自动化覆盖**：precheck（含 compose v2 自检 / 缺失时从 infra bundle 自动装 v2 插件）→
旧 systemd 单元自动迁移（兼容历史宿主二进制部署，停掉并备份 `omcgo-{app,acs,worker}.service`）→
建立目录布局 + `current` 软链 → load 镜像（基础设施 + 监控 + 业务，从 `/opt/omc/infra/images/`
与版本目录 `images/`；已存在则跳过 load 并重启容器）→ 默认口令检查（含交互确认）→
组装 compose 命令 → 起基础设施 + 等就绪（PG + Redis ping ≤ 90s）+ 容器内 `migrate-schema` /
`migrate-seed-sql`（goose 幂等，migrate 失败自动重试 3 次）→ `docker compose up -d` 全栈 → 调 `healthcheck.sh`。

**幂等**：所有步骤均可重跑。二次运行视情况跳过已完成项（镜像 `docker image inspect`
命中即跳过 load 并 restart、seed 走 goose 版本表自动 catch up、`etc/` 已有实例配置默认保留不覆盖）。

**安全要求**：deploy.sh **会主动检查** `deploy/.env` 内默认口令（`omcgo123` /
`minioadmin` / `admin`），存在时弹交互确认；生产部署务必在跑 deploy 前先按 §4 / §6.1 改强口令。

### 5.11 日常运维 — `svc.sh`

部署完成后，日常启停 / 重启 / 查日志用 `svc.sh`（与 deploy.sh 同目录，无须再跑 deploy.sh）：

```bash
cd /opt/omc/current/deploy
bash svc.sh status                  # 查看所有服务状态（默认）
bash svc.sh start [svc...]          # 启动全栈，或指定服务
bash svc.sh stop  [svc...]          # 停止（容器保留，volume 保留）
bash svc.sh restart [svc...]        # 重启全栈，或指定服务
bash svc.sh logs app --tail 200     # 查看 app 日志
bash svc.sh logs acs -f             # 跟随 acs 日志（Ctrl-C 退出）
bash svc.sh down                    # 关栈并删容器（保留 volume）
bash svc.sh --skip-monitoring restart   # 不操作监控栈
```

> svc.sh 自动按存在性拼 4 个 compose 文件，compose project 固定 `omcgo`（与 deploy.sh 一致），不需 root（除非 docker daemon 需 sudo）。

### 5.12 系统加速设置（可选 / 安装后任意时刻可改）

`install-docker.sh` 装完 Docker 后会引导选择加速；后期想换或单独配置：

```bash
sudo bash /opt/omc/infra/setup-mirrors.sh                    # 交互选单（依次问 Docker / npm / Golang）
sudo bash /opt/omc/infra/setup-mirrors.sh --docker daocloud  # 仅 Docker（非交互）
sudo bash /opt/omc/infra/setup-mirrors.sh --npm taobao       # 仅 npm
sudo bash /opt/omc/infra/setup-mirrors.sh --golang goproxycn # 仅 Golang
sudo bash /opt/omc/infra/setup-mirrors.sh --show / --remove  # 看当前三项 / 取消全部
```

| 目标 | 选项 | 说明 |
|------|------|------|
| Docker | `official` / `daocloud`（推荐 `https://docker.m.daocloud.io`）/ `xuanyuan` | 写 `/etc/docker/daemon.json` 的 `registry-mirrors`（python3 merge 保留其它键，变更时 restart docker） |
| npm | `official` / `taobao`（推荐 `https://registry.npmmirror.com`） | 写系统级 `/etc/npmrc`，不动用户 `~/.npmrc` |
| Golang | `official` / `goproxycn`（推荐 `https://goproxy.cn,direct` + `GOSUMDB=sum.golang.google.cn`） | 写 `/etc/profile.d/goproxy.sh`，新 shell 自动加载 |

> 纯离线场景下三项加速对**首次部署**没有影响（镜像 / 包 / 模块都已随包交付）；
> 配置加速主要利于运维侧后续临时 `docker pull`、`npm install`、`go install` 提速。

### 5.13 部署后访问 + 初始账号

| 端点 | URL | 初始账号 |
|------|-----|---------|
| **Web 管理页** | `http://<服务器IP>:8081` | `admin` / `admin123` |
| 基站 ACS URL | `http://<服务器IP>:8080`（基站设备侧填，人不浏览） | — |
| MinIO Console | `http://<服务器IP>:9001` | `minioadmin` / `minioadmin` |
| PostgreSQL | `<服务器IP>:5432` | `omcgo` / `omcgo123` |
| Grafana（可选监控栈） | `http://<服务器IP>:3030` | `admin` / `admin` |
| app 健康端点 | `http://127.0.0.1:9091/healthz` | — |
| acs 健康端点 | `http://127.0.0.1:9095/healthz`（容器内 9090） | — |

> ⚠️ **生产部署前**：上述默认口令必须改强口令，并同步 `deploy/.env` 与 `*.prod.yaml`；用户首次登录 Web UI 强制改密。

### 5.14 脚本帮助速查

```bash
sudo bash /opt/omc/infra/docker/install-docker.sh -h    # 装 Docker / Compose / Buildx
sudo bash /opt/omc/infra/setup-mirrors.sh -h            # Docker/npm/Golang 加速
sudo bash /opt/omc/current/deploy/deploy.sh -h          # 一键部署
bash /opt/omc/current/deploy/svc.sh -h                  # 日常启停 / 重启 / 日志
bash /opt/omc/current/deploy/healthcheck.sh -h          # 健康校验
```

---

## 6. 配置说明

### 6.1 安全必改项

见 §5 步骤 4 表格。**严禁**用模板默认密码上生产；`.env` 与 `*.prod.yaml` 两处口令必须一致。

### 6.2 `*.prod.yaml` 关键项

| 段 | 项 | 说明 |
|----|----|----|
| `server` | `port` / `tls_port` / `grpc_port` | 服务监听端口 |
| `db` / `tsdb` | `dsn` / `max_conns` | 数据库连接（容器内用服务名 `postgres:5432`），连接数按规模调 |
| `redis` | `addrs` / `pool_size` | Redis 地址（容器内 `redis:6379`） |
| `nats` | `url` | NATS 地址（容器内 `nats:4222`） |
| `minio` | `endpoint` / `buckets` | 对象存储（容器内 `minio:9000`）；buckets 首启自动创建 |
| `jwt` | `secret` / `*_ttl` | 令牌密钥与有效期 |
| `login_crypto` | `private_key_path` / `allow_plaintext` | 登录密码 RSA 加密；HTTP 内网无 TLS 时可设 `allow_plaintext: true`（仅受信内网） |
| `batch_processor` | `workers` | Periodic Inform 批量并发，建议 = CPU 核数/2 |
| `log` | `level` / `output_paths` / `rotation` | 日志级别（prod 建议 `warn`）、路径、轮转 |
| `license` | `signing.public_key_dir` / `strict` | License 治理；strict=true 需放 OEM 公钥 |
| `notification` | `smtp.*` / `alert_webhook.*` | 告警邮件通知。默认 `smtp.enabled: false`；需告警邮件时填 `smtp.host`/`from` 并置 `enabled: true`（详见《告警处置Runbook.md》与 `monitoring/alertmanager.yml`） |

### 6.3 TLS 与密钥

- **TLS**：默认内网 HTTP。需加密时用 `deploy/certs/` 自签脚本生成证书，或导入企业 CA 证书并配置 nginx / 业务 yaml。
- **登录 RSA 私钥**：`login_crypto.private_key_path` 指向的 PEM 需在部署时生成并放入 `/opt/omc/etc/keys/`；**多副本部署必须共用同一份**（keyID 一致）。
- 内网纯 HTTP 访问时浏览器 `crypto.subtle` 不可用，可临时启用 `allow_plaintext: true`（审计日志会标记），但推荐尽快上 TLS。

### 6.4 host 主权的自定义 XML 目录

deploy.sh 首次部署把整个 `data/` 目录外置 bind mount（容器内 UID 10001 写入），不进镜像；
builtin 与运维上传的 custom XML **同住一个目录**，靠空文件 `X.xml.custom` sidecar 标记来源
（ADR 0004 单目录 + sidecar，取代旧的 `*-custom` 双目录）。升级「现网赢、新版补充」反向合并，
带 `.custom` sidecar 的自定义 XML 一律保留：

- `/opt/omc/data/param-mappings/`（paramModel XML，扁平；custom 旁有 `.custom`）
- `/opt/omc/data/indicator-library/{enb,gsm,gnb}/`（indicator XML，三制式分桶；custom 旁有 `.custom`）
- `/opt/omc/data/alarm-definitions/`（alarm XML，扁平；custom 旁有 `.custom`）

---

## 7. 验收清单

部署完成后逐项确认：

- [ ] 4 个基础设施容器 + 3 个业务容器 + web 容器（+监控）均 `running`（`svc.sh status`）
- [ ] 数据库 schema + seed 迁移均成功（`migrate-schema` / `migrate-seed-sql` 容器 exit 0）
- [ ] `healthcheck.sh` 全部 `[OK]`（容器 running + 5 个健康端点）
- [ ] 前端 `http://<IP>:8081` 可访问、可登录、首登改密完成
- [ ] CPE 基站能 Inform 接入 ACS（基站填 `http://<IP>:8080` 作 ACS URL），设备出现在设备列表
- [ ] 默认密码（PG/MinIO/JWT/Grafana/管理员）全部已改，`.env` 与 `*.prod.yaml` 一致
- [ ] 日志正常写入 `/opt/omc/run/logs/`，轮转生效
- [ ] 关键端口防火墙策略已按规划放行/收敛（5432/6379/4222/9000/9001 出公网前 deny）
- [ ] 数据盘容量与告警阈值已设置
- [ ] 备份方案（见 §8）已落实并演练

---

## 8. 升级与回滚

升级/回滚基于 §5.0 的「版本目录 + `current` 软链」：新版本进新的 `releases/<版本>/`
目录，`deploy.sh` 切软链 + 起新镜像容器，旧版本目录原样留存、可快速回退。

> 本节是**项目包升级**（日常）。基础设施包很少升级；需要时（基础设施包出了新版本）
> 重新执行 §5 步骤 1(1) + 步骤 3(1)，把新基础设施包解压到 `/opt/omc/infra/` 并
> `docker load` 新镜像，与项目包升级互不影响。

### 8.1 升级（项目包）

```bash
VER=<新项目版本号>
PROJ=<新项目包文件名>     # 如 omc-test-<版本>-amd64 或 omc-release-<版本>-amd64

# 1) 备份（不可省）：数据库 + 当前实例配置（MinIO 数据按客户备份方案另行快照）
docker exec $(docker ps -qf name=postgres) \
  pg_dump -U omcgo omcgo > /opt/omc/packages/db-backup-$(date +%F).sql
cp -r /opt/omc/etc /opt/omc/packages/etc-backup-$(date +%F)

# 2) 解压新版本到独立目录（current 暂不动，老版本仍在运行）
mkdir -p /opt/omc/releases/$VER && cd /opt/omc/releases/$VER
tar xf $PROJ.tar.xz --strip-components=1
sha256sum -c checksums.sha256
ln -sfn /opt/omc/releases/$VER /opt/omc/current

# 3) 配置：比对新版本模板有无新增项，按需手工合并到 /opt/omc/etc/（不要整体覆盖）
diff -ru /opt/omc/etc /opt/omc/releases/$VER/etc

# 4) 跑 deploy.sh：自动 load 新业务镜像、跑 migrate/seed（幂等、向后兼容）、起新容器
sudo bash /opt/omc/current/deploy/deploy.sh --skip-infra

# 5) 跑 §7 验收清单
```

> 业务镜像 tag 含版本号，新旧镜像并存；`deploy.sh` 起容器时按 `.env` 的新 tag 拉起，
> 旧容器被 `up -d` 滚动替换。新版本若引入新表 / 新 seed，goose 自动追加。

#### 8.1.1（可选）device_parameters 路径翻译迁移

仅首次升级到含 Stage 2 路径翻译的版本时执行：历史 `device_parameters` 行的
`parameter_path` 是基站私有 path，从该版本起改写为系统级 standardPath。`omcctl` 运维 CLI
随 **worker 镜像**发布（`docker exec` 进 worker 容器跑，默认连 `OMCCTL_SERVER=http://app:8081`）：

```bash
WORKER=$(docker ps -qf name=worker)
docker exec "$WORKER" omcctl mml migrate-device-params                       # dry-run，输出 would update N rows
docker exec "$WORKER" omcctl mml migrate-device-params --apply --batch=500   # 真实执行
```

无脏数据 / 老版本本来就走 standardPath 的环境可跳过本步。

### 8.2 回滚

把 `current` 软链切回旧版本目录并用旧镜像重起容器：

```bash
ln -sfn /opt/omc/releases/<旧版本> /opt/omc/current
sudo bash /opt/omc/current/deploy/deploy.sh --skip-infra --skip-migrate
```

- **配置**：用 §8.1 步骤 1 备份的 `etc-backup-*` 还原 `/opt/omc/etc/`。
- **数据库**：若新版本迁移含**破坏性变更**（删列/改类型/删表），回滚旧镜像后 schema 不兼容，须用 §8.1 步骤 1 的 `pg_dump` 备份恢复——**因此升级前备份绝不可省**；向后兼容的迁移则无需动数据库（故回滚加 `--skip-migrate`）。

### 8.3 版本留存策略

- `releases/` 下**保留最近 3 个版本**目录，供回滚与问题排查。
- 清理更早版本：确认 `current` 未指向后 `rm -rf /opt/omc/releases/<旧版本>`，并 `docker rmi` 对应旧业务镜像。
- 交付包原件存档于 `/opt/omc/packages/`，可长期保留。

> 升级前务必完成 `pg_dump` + MinIO 备份，并已在测试环境演练过回滚流程。

### 8.4 SSD/NVMe 数据路径与存量迁移

`deploy/.env` 支持五个独立路径：

```dotenv
POSTGRES_DATA_PATH=/mnt/nvme-b/omc-data/postgres
TSDB_DATA_PATH=/mnt/nvme-a/omc-data/timescaledb
REDIS_DATA_PATH=/mnt/ssd/omc-data/redis
NATS_DATA_PATH=/mnt/ssd/omc-data/nats
MINIO_DATA_PATH=/mnt/nvme-c/omc-data/minio
```

留空时继续使用 Docker 命名卷；运行 `bash deploy/plan-resources.sh` 会以可用空间最大的本地
文件系统生成默认值，但不会覆盖已有人工值，也不会搬迁数据。安装前必须人工检查 `.env`，
确认目录确实位于目标 SSD/NVMe，而不是同一旋转盘的另一个目录。

推荐优先级：TimescaleDB 独占写入能力最强的 NVMe，PostgreSQL 主库使用另一块 NVMe，
MinIO 使用第三块 SSD/NVMe；Redis 与 NATS 放在剩余低延迟盘。只有一块 SSD/NVMe 时可先把
五项都指向该盘，仍能降低机械寻道等待，但不能隔离组件之间的 I/O 竞争。

已有环境以主库为例按以下步骤迁移，其他组件分别替换为
`omcgo_tsdbdata/omcgo_redisdata/omcgo_natsdata/omcgo_miniodata` 和对应环境变量：

```bash
cd /opt/omc/current/deploy
bash svc.sh stop
SRC="$(docker volume inspect -f '{{.Mountpoint}}' omcgo_pgdata)"
DEST=/mnt/nvme-b/omc-data/postgres
install -d "$DEST"
rsync -aHAX --numeric-ids "$SRC"/ "$DEST"/
du -sb "$SRC" "$DEST"                 # 容量应一致；重要库再核对文件数/备份
vi .env                               # 填 POSTGRES_DATA_PATH=$DEST
docker compose -p omcgo --env-file .env --env-file resources.env \
  -f docker-compose.infra.yml -f docker-compose.app.yml config >/dev/null
bash svc.sh up
docker inspect omcgo-postgres-1 --format '{{range .Mounts}}{{println .Source "->" .Destination}}{{end}}'
```

随后检查两个数据库 `pg_isready`、Redis `PING`、NATS `http://127.0.0.1:8222/healthz`、
MinIO `http://127.0.0.1:9000/minio/health/live`、app/ACS/worker 健康接口。旧卷或旧目录至少
保留一个观察周期；回滚时停服，把对应路径改回旧位置（命名卷则清空变量）再启动。

当前单旋转盘压测观察到读取等待约 154 ms、末段 iowait 约 61%。迁移到真实 SSD/NVMe 后
预期存储等待和队列深度显著下降，PM 消费速率更接近输入速率；具体收益受介质、RAID 和
拆盘方式影响，必须按相同 KPI 口径复测，不能用脚本配置生效代替硬件收益验证。

### 8.5 卸载

```bash
sudo bash /opt/omc/current/deploy/deploy.sh --uninstall                 # dry-run，只列将做的动作
sudo bash /opt/omc/current/deploy/deploy.sh --uninstall --no-dry-run    # 真删（再做一次交互确认）
sudo bash /opt/omc/current/deploy/deploy.sh --uninstall --no-dry-run --keep-data --keep-images
                                                                        # 删容器+/opt/omc，但保留数据卷与业务镜像
```

> 卸载默认 **dry-run**，仅打印将停止/删除的容器、数据卷、网络、业务镜像、`/opt/omc`。
> `--no-dry-run` 才真删；`--keep-data` 保留 docker 数据卷（pg/minio/redis/tempo/loki）；
> `--keep-images` 保留 `omcgo/*` 业务镜像。**不卸载 Docker 引擎本身**（如需：`install-docker.sh --uninstall`）。

---

## 9. 内网环境依赖与不可控因素分析

> 以下每项都可能导致部署失败，需在交付前 / 部署中逐一排除。

| # | 因素 | 风险 | 应对 |
|---|------|------|------|
| 1 | **CPU 架构** | 当前仅 amd64 包，跑到 arm64 机器无法执行 | 部署前 `uname -m` 确认为 `x86_64`；arm64 联系构建侧 |
| 2 | **操作系统/内核** | 国产化 OS（麒麟/统信）或老旧内核对 Docker、cgroup 支持差异 | 交付前确认发行版+内核；Docker 用静态二进制规避包管理器差异；信创环境提前在同型号机验证 |
| 3 | **时间同步** | 内网无公网 NTP，多机时钟漂移 → JWT 失效、TLS 校验失败、定时任务错乱 | 确认内网 NTP；无则指定一台为 NTP 源；容器统一 `TZ=Asia/Shanghai` |
| 4 | **时区** | 容器/宿主时区不一致 → 时序数据、日志、调度偏差 | 全栈固定 `Asia/Shanghai` |
| 5 | **端口冲突** | 目标机已有服务占用 8081/8080/5432/6379 等 | §4.4 提前核查；冲突时改 compose 端口映射 + 同步改 `.env`/`*.prod.yaml` |
| 6 | **防火墙/SELinux** | iptables/firewalld 拦截容器网络与对外端口；SELinux 阻止容器挂载卷 | 放行 §2.1 端口；SELinux 设宽容模式或为卷打 `:z` 标签；记录变更供安全审计 |
| 7 | **磁盘容量** | PM/MR/固件文件约 500 GB/月增长，pgdata/miniodata 撑爆根盘 | docker `data-root` 落独立数据盘（install-docker.sh 可切 /home）；设容量告警；配 PM/MR 生命周期清理 |
| 8 | **文件描述符上限** | 高并发下 ACS 会话 + nginx 连接耗尽 fd → 拒绝连接 | compose `ulimits.nofile`；调宿主 `/etc/security/limits.conf` 与 dockerd `LimitNOFILE` |
| 9 | **数据库选型** | 客户要求用既有 PG 实例，但缺 TimescaleDB 扩展 → 时序表创建失败 | 优先用交付包内 TimescaleDB 容器；若用外部 PG，必须确认已装 TimescaleDB 扩展且兼容 pg16 |
| 10 | **容器 DNS** | 容器内 DNS 解析异常（已知痛点，见 `deployments/docker/DOCKER_DNS_FIX.md`） | 按该文档配置 `docker-daemon-dns.json`；compose 内服务名互联走 docker 内部 DNS |
| 11 | **内网镜像仓库** | 有 Harbor 则镜像分发更优；无则只能逐机 `docker load` | 交付前确认；有 Harbor 可改为推仓库 + compose 引用 |
| 12 | **镜像版本一致性** | 自行 `docker pull` 浮动标签会覆盖交付镜像、导致版本漂移 | 镜像版本已由构建侧固化；运维侧只 `docker load` 交付包内镜像，勿自行 `docker pull` |
| 13 | **CPE 回连网络** | ACS 与 CPE 基站之间 NAT/防火墙阻断 Inform / Connection Request | 确认 CPE 的 ACS URL 指向 `http://<IP>:8080`、`:7557` 可达 |
| 14 | **多副本密钥一致** | app 多副本但 JWT/登录 RSA 密钥不一致 → 令牌跨副本失效 | 多副本共用同一份 `jwt.secret` / `OMCGO_JWT_SECRET` 与 `login_crypto` 私钥 |
| 15 | **权限** | 无 root/sudo 无法装 Docker、写 `/opt/omc` | 交付前确认权限；最小权限方案需提前与客户协商 |
| 16 | **优雅停机** | 直接 `docker kill` 丢失会话/未刷盘数据 | 用 `svc.sh stop` / `docker compose stop`（容器响应 SIGTERM 优雅关闭） |
| 17 | **内核 IP 转发未开** | 加固/信创 Linux 默认 `net.ipv4.ip_forward=0` → 容器无法联网、端口映射失效 | 按 §5 步骤 2.1 开启；firewalld 重载会重置，需配 masquerade 或重载后再 `sysctl --system` |
| 18 | **compose 版本** | 系统装的 docker-compose V1（Python）不识别 compose v3 写法 → 报 Unsupported config option | deploy.sh 优先用 `docker compose` V2，缺失时从 infra bundle 自动装 v2 插件；或 `install-docker.sh` 重装 |
| 19 | **`.env` 与 yaml 口令不一致** | 改了 `.env` 但已起过容器 / 没同步 yaml → 业务容器认证失败 | 首次起容器前同步两处口令；已起过需删数据卷或进容器改库内口令（详见下载页 §9） |

---

## 10. 常见故障排查

| 现象 | 排查方向 |
|------|---------|
| 业务容器起不来 / 反复重启 | `cd /opt/omc/current/deploy && bash svc.sh logs app`（或 acs/worker）；看是否连不上 DB/Redis（基础设施容器是否 healthy） |
| 容器认证 PostgreSQL/MinIO 失败 | `.env` 与 `*.prod.yaml` 口令是否一致；`.env` 改过但数据卷已用旧口令创建（见 §9 #19） |
| 端点对非超管返回 500 `casbin authorizer not configured` | `configs/casbin_model.conf` 缺失或挂载路径不对 |
| 启动报字典 XML 加载失败 | `data/` 目录缺失或挂载不对；确认 `current` 软链指向有效版本目录 |
| migrate 容器失败 | 检查 PG 是否就绪（deploy.sh 等 90s）；schema 与 seed 用**不同** goose 版本表；`$COMPOSE logs migrate-schema` |
| 前端能开但接口 502 | nginx `default.conf` 的 `app_backend` upstream；app 容器是否 running |
| CPE 接不进来 | 防火墙 `:8080`（nginx→acs `:7547`）；CPE 的 ACS URL；`svc.sh logs acs` |
| 报 `Unsupported config option` | docker-compose V1 不兼容 compose v3，按 §9 #18 装 v2 插件 |
| 容器起不来 / 端口映射不通 / 报 `IPv4 forwarding is disabled` | 内核 IP 转发未开，按 §5 步骤 2.1 配置 |
| 容器 DNS 解析失败 | 见 §9 #10，参考 `deployments/docker/DOCKER_DNS_FIX.md` |
| 时间相关报错（令牌/证书） | 见 §9 #3 时间同步 |

日志位置：容器日志 `docker compose -p omcgo logs <svc>`（或 `svc.sh logs <svc>`）；
业务日志文件 `/opt/omc/run/logs/{app,acs,worker,nginx}/`，zap JSON 格式，lumberjack 自动轮转。

> **Prometheus 告警的逐条处置**（容器存活、基础服务、连接池、业务积压等告警的含义、
> 排查、处置、升级路径）见 [`告警处置Runbook.md`](./告警处置Runbook.md)。

---

**文档结束。** 端口、配置项、镜像版本以实际交付版本的 `deploy/.env`、`etc/*.prod.yaml`、`deploy/docker-compose.*.yml` 为准。

# OMC 离线交付包构建手册（构建侧）

> **适用对象**：研发 / 发布工程师。
> **执行环境**：有公网、装有 Go / Node / Docker 的**构建机**。
> **产出**：可送入内网的离线交付包 `omc-release-<版本>-<架构>.tar.gz`（amd64 + arm64 各一）。
> **配套文档**：运维侧部署见《OMC内网离线部署手册（运维侧）》。
> **配套工具**：构建工具位于仓库 `deployments/release/`。
> **文档状态**：v1.0，需随产品版本迭代同步维护。

---

## 1. 交付模式

OMC 采用**「构建侧编译、运维侧只跑二进制」**：

- **构建侧（本手册）**：在有公网的构建机一次性完成 Go 编译、前端打包、Docker 镜像收集，产出一个自包含的离线交付包。
- **运维侧**：拿到交付包后**零 Go 依赖、零 Node 依赖、全程不联网**——因为交付物是已编译的静态二进制 + 已打包的镜像。

| 组件 | 交付形态 |
|------|---------|
| omcgo-app / omcgo-acs / omcgo-worker、omcgo-migrate / omcgo-seed、omcctl | 已编译 Linux 静态二进制（`CGO_ENABLED=0`，无动态库依赖） |
| 前端 webcode | 已编译静态资源（HTML/JS/CSS） |
| PostgreSQL/TimescaleDB、Redis、NATS、MinIO、nginx | `docker save` 离线镜像包 |
| Docker 引擎 | 官方静态二进制包（构建侧一次性下载，打入交付包） |

> 构建侧需要 Go 工具链 + 130 个 Go 三方组件、Node/npm、Docker、公网；这些**运维侧一律不需要**。

---

## 2. 产物架构（构建者需了解的目标形态）

构建产物最终在内网组成如下系统（运维侧拓扑）：

```
宿主机进程（二进制 + systemd）          Docker 容器（基础设施）
  omcgo-app   :8081/:8444/:50051/:9091    postgres(TimescaleDB pg16) :5432
  omcgo-acs   :7557/:7558/:3478/:9090     redis                      :6379
  omcgo-worker :9092                      nats(JetStream)            :4222
                                          minio                 :9000/:9001
前端 nginx 容器 :8080/:8081               （可选）prometheus/grafana/loki
```

| 类别 | 组件 | 默认端口 |
|------|------|---------|
| OMC | omcgo-app | 8081 / 8444(TLS) / 50051(gRPC) / 9091(metrics) |
| OMC | omcgo-acs | 7557 / 7558(TLS) / 3478(STUN) / 9090(metrics) |
| OMC | omcgo-worker | 9092(metrics) |
| 前端 | nginx | 8080 / 8081 |
| 基础设施 | postgres / redis / nats / minio | 5432 / 6379 / 4222 / 9000 |

---

## 3. 构建环境要求

| 项 | 要求 |
|----|------|
| Go | **1.25+**（与 `omcgo/go.mod` 的 `go 1.25.0` 一致） |
| Node.js | **20.x** + npm |
| Docker | 支持 `docker pull --platform` 多架构拉取 |
| 网络 | 公网（用于拉取 Go/npm 依赖与 Docker 镜像）；构建机若也离线见 §8 |
| 架构产出 | **每个版本固定同时产出 amd64 与 arm64 两套二进制与镜像**，对应两个交付包 |

---

## 4. 交付包形态（构建产物结构）

> **每个版本产出两个交付包**：`omc-release-<版本>-amd64.tar.gz` 与 `omc-release-<版本>-arm64.tar.gz`。
> 两包结构完全相同，仅 `bin/` 二进制与 `images/` 镜像为对应架构。

```
omc-release-<版本>-<架构>/
├── README.md                      # 交付包说明 + 版本号 + 校验和
├── VERSION                        # 版本号 / 构建时间 / git commit
├── checksums.sha256               # 全部文件 SHA256
│
├── docker/                        # ① Docker 引擎离线安装
│   ├── docker-<ver>.tgz           #   Docker 静态二进制包（官方）
│   └── install-docker.sh          #   离线安装脚本
├── images/                        # ② Docker 镜像离线包（docker save）
│   ├── infra-images-<架构>.tar    #   postgres/redis/nats/minio/nginx
│   ├── monitoring-images-<架构>.tar #  prometheus/grafana/loki/...（可选）
│   └── images.manifest            #   镜像名:tag 清单
├── bin/                           # ③ OMC 已编译二进制（核心交付物）
│   └── omcgo-app / omcgo-acs / omcgo-worker / omcgo-migrate / omcgo-seed / omcctl
├── web/dist/                      # ④ 前端已编译静态资源
├── etc/                           # ⑤ 配置模板（app/acs/worker.prod.yaml，需现场改）
├── data/                          # ⑥ 启动期加载的字典 XML
├── configs/                       # ⑦ Casbin RBAC 模型 casbin_model.conf
├── migrations/                    # ⑧ 数据库迁移 SQL（schema + seed/）
├── deploy/                        # ⑨ 部署脚本与模板
│   ├── docker-compose.infra.yml / docker-compose.web.yml / .env
│   ├── nginx.conf / default.conf
│   ├── systemd/                   #   omcgo-app/acs/worker.service
│   └── healthcheck.sh
└── docs/
    └── OMC内网离线部署手册（运维侧）.md   # 交给运维的部署文档
```

**产物来源对照**：

| 交付包目录 | 来源 |
|-----------|------|
| `bin/` | 构建侧交叉编译（§6.1），每个版本产 amd64 + arm64 两套 |
| `web/dist/` | `npm run build`（§6.2） |
| `images/*.tar` | `docker pull --platform` + `docker save`（§6.3） |
| `data/` `configs/` `migrations/` | 取自源码仓库 `omcgo/` 对应目录，随版本走 |
| `etc/*.prod.yaml` | 源码 `omcgo/cmd/{app,acs,worker}/etc/config.prod.yaml`，**模板** |
| `deploy/` `docker/` | 取自 `deployments/release/bundle/` |

---

## 5. 一键构建（推荐）

构建流程已固化为工具 **`deployments/release/build-release.sh`**。

```
deployments/release/
├── README.md                 # 工具说明
├── release.conf              # 配置：镜像清单、目标架构
├── build-release.sh          # ⭐ 生成交付物的工具
├── bundle/                   # 进交付包的静态模板（docker/ + deploy/）
└── dist/                     # 交付包产出目录（git 忽略）
```

**步骤**：

```bash
cd deployments/release

# 1) 一次性准备：把 Docker 静态二进制包放进 bundle/docker/
#    从 https://download.docker.com/linux/static/stable/ 下载对应架构的
#    docker-<版本>.tgz（amd64 用 x86_64/，arm64 用 aarch64/），
#    详见 bundle/docker/README.md。

# 2) 生成交付包（默认同时产出 amd64 + arm64）
./build-release.sh -v 1.2.0

#    可选参数：
#      -v <版本>          版本号（缺省取 git describe）
#      --arch amd64       只构建指定架构
#      --with-monitoring  额外打包监控栈镜像
```

产出：

```
dist/omc-release-1.2.0-amd64.tar.gz   + .sha256
dist/omc-release-1.2.0-arm64.tar.gz   + .sha256
```

交付包通过移动介质（U 盘 / 光盘 / 堡垒机文件摆渡）送入内网。

---

## 6. 构建流程详解

> 本章是 `build-release.sh` 内部流程的展开说明，便于理解与排错；也可据此手动构建。

### 6.1 编译 Go 二进制（amd64 + arm64 双架构）

Go 原生支持交叉编译，**每个版本必须同时编译 amd64 与 arm64**，与构建机自身架构无关：

```bash
cd omcgo
for ARCH in amd64 arm64; do
  for SVC in app acs worker migrate seed; do
    CGO_ENABLED=0 GOOS=linux GOARCH=$ARCH go build -ldflags="-s -w" \
      -o bin/$ARCH/omcgo-$SVC ./cmd/$SVC
  done
  CGO_ENABLED=0 GOOS=linux GOARCH=$ARCH go build -ldflags="-s -w" \
    -o bin/$ARCH/omcctl ./cmd/omcctl
done
```

> Go 第三方组件（130 个，见 `go.mod`）在此步由 `go build` 自动拉取并**编译进二进制**，运维侧不再需要它们。

### 6.2 构建前端（架构无关，只做一次）

```bash
cd omcmb/webcode
npm ci --include=dev --legacy-peer-deps   # 严格按 package-lock.json 安装
npm run build                              # 产出 dist/（纯静态）
```

两个架构的交付包复用同一份 `web/dist`。

### 6.3 收集 Docker 镜像（amd64 + arm64 双架构）

Docker 镜像按架构区分。镜像导出**不依赖构建机自身架构**——用 `docker pull --platform`
显式逐架构拉取（拉取只下载分层、不执行，跨架构可行）。但 `docker save` 读的是
构建机本地镜像库，故每个架构在 pull 前先 `docker rmi` 清本地同名镜像，保证导出确定：

```bash
IMAGES="timescale/timescaledb:<具体版本>-pg16 redis:7-alpine nats:2.10-alpine \
        minio/minio:RELEASE.<具体日期> nginx:alpine"
for ARCH in amd64 arm64; do
  for IMG in $IMAGES; do
    docker rmi -f "$IMG" >/dev/null 2>&1 || true   # 清缓存，使 save 架构确定
    docker pull --platform linux/$ARCH "$IMG"
  done
  docker save -o images/infra-images-$ARCH.tar $IMAGES
done
```

### 6.4 打包与校验

```bash
for ARCH in amd64 arm64; do
  PKG=omc-release-<版本>-$ARCH
  # 组装：bin/<架构> + images/<架构> + web/data/configs/migrations/etc/deploy/docs
  ...
  ( cd "$PKG" && find . -type f ! -name checksums.sha256 -exec sha256sum {} + > checksums.sha256 )
  tar czf $PKG.tar.gz $PKG/
  sha256sum $PKG.tar.gz > $PKG.tar.gz.sha256
done
```

---

## 7. 镜像版本固化（重要）

`release.conf` / 构建脚本中的基础设施镜像标签，**发布前必须固化为具体版本号**，
不要用 `latest` / `latest-pg16`：

- 浮动标签会导致**不同批次交付包镜像不一致**，难以复现问题。
- 镜像标签同时写入交付包 `deploy/.env`，供 `docker-compose` 引用——必须与
  `docker save` 的镜像保持一致。

> 例：`timescale/timescaledb:latest-pg16` → `timescale/timescaledb:2.17.x-pg16`；
> `minio/minio:latest` → `minio/minio:RELEASE.<具体日期>`。

---

## 8. 附：构建机也离线 / 现场源码编译

默认构建机有公网即可。两类特殊场景：

**A. 构建机本身离线**：先在有网环境 `cd omcgo && go mod vendor` 把 130 个 Go
组件落入 `vendor/`，并预置 Go 1.25 SDK、npm 离线缓存；之后 `go build -mod=vendor`
与 `npm ci --offline` 全程不联网。

**B. 客户安全策略要求内网内自行编译**（拒收二进制，少见）：

1. 构建侧 `go mod vendor`，把 `vendor/` 随源码交付；
2. 额外交付 Go 1.25 离线 SDK 包（`go1.25.linux-<架构>.tar.gz`）；
3. 前端额外交付 npm 离线缓存或 `node_modules` 整包；
4. 内网编译环境装好 Go SDK 后 `make build` + `npm run build` 即可。

> B 方案增加交付包体积与现场复杂度，非必要不推荐——默认交付二进制。

---

## 9. 交付前自检清单

- [ ] 版本号正确，`VERSION` 文件与交付包名一致
- [ ] amd64、arm64 两个交付包均已产出
- [ ] 镜像标签已固化为具体版本（非 `latest`），且 `deploy/.env` 与之一致
- [ ] `bundle/docker/` 已放入对应架构的 `docker-<版本>.tgz`
- [ ] `checksums.sha256` 与 `*.tar.gz.sha256` 校验通过
- [ ] 交付包内 `docs/` 含最新版《OMC内网离线部署手册（运维侧）》
- [ ] 在测试环境用交付包完整走通一遍部署（见运维侧手册）

---

**文档结束。** 端口、配置项、镜像版本以实际交付版本的源码 `config.prod.yaml`、
`deployments/` 为准。

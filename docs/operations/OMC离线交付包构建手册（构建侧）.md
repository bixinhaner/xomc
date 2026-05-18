# OMC 离线交付包构建手册（构建侧）

> **适用对象**：研发 / 发布工程师。
> **执行环境**：有公网、装有 Go / Node / Docker 的**构建机**。
> **产出**：可送入内网的离线交付包 `omc-release-<版本>-<架构>.tar.xz`（amd64 + arm64 各一），按版本归档到 `archive/`，可经 HTTP 服务下载。
> **配套文档**：命令速查见《OMC交付构建快速上手》；运维侧部署见《OMC内网离线部署手册（运维侧）》。
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
| 运行身份 | **普通用户**运行，不要 `sudo`（见下方说明） |

> ⚠️ **不要用 `sudo` 运行 `build-release.sh`**：
> - 它是构建脚本，`go build` / `npm build` 都不需要 root；
> - `sudo` 会重置 `PATH`（sudoers 的 `secure_path`），导致脚本找不到 `go`/`npm`
>   —— 即使 `go version` 在你的 shell 里能跑，仍会报"缺少构建工具：go"；
> - `sudo` 还会让 `dist/`、Go 构建缓存、`node_modules` 归 root，后续难处理。
>
> `docker` 所需权限请用 **docker 组**解决，而非 sudo 整个脚本：
> ```bash
> sudo usermod -aG docker $USER && newgrp docker   # 或重新登录
> ```

---

## 4. 交付包形态（构建产物结构）

> **每个版本产出两个交付包**：`omc-release-<版本>-amd64.tar.xz` 与 `omc-release-<版本>-arm64.tar.xz`
> （压缩方式由 `release.conf` 的 `PKG_COMPRESS` 决定，默认 `xz`）。
> 两包结构完全相同，仅 `bin/` 二进制与 `images/` 镜像为对应架构。下方为**解压后**的目录结构。

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

## 5. 构建工具与流程

构建工具位于 `deployments/release/`，**四个脚本**——基础设施准备（Docker 引擎、
镜像）与项目发版分离（构建周期不同），外加一个下载服务：

```
deployments/release/
├── release.conf              # 配置：版本号、压缩方式、Docker 版本与地址、镜像清单
├── download-docker.sh        # ① Docker 引擎离线包下载（不常跑）
├── build-images.sh           # ② 基础设施镜像构建（不常跑）
├── build-release.sh          # ③ 交付包构建 / 发版（常跑）
├── serve.sh                  # ④ HTTP 下载服务
├── bundle/                   # 进交付包的静态模板（docker/install-docker.sh + deploy/）
├── docker-cache/             # ① 的产物：docker-cache/<arch>/docker-<版本>.tgz（git 忽略）
├── images-cache/             # ② 的产物，③ 复用（git 忽略）
├── dist/                     # 构建临时工作区（git 忽略）
└── archive/                  # 版本化归档 = HTTP 服务根目录（git 忽略）
    ├── index.html            #   自动生成的版本下载索引
    └── <项目版本>/ …          #   交付包按版本归档
```

| 工具 | 职责 | 运行频率 |
|------|------|---------|
| `download-docker.sh` | 按 `release.conf` 下载 Docker 引擎离线包 → `docker-cache/` | **低**：仅 Docker 版本变更时 |
| `build-images.sh` | 拉取并导出基础设施 Docker 镜像 → `images-cache/` | **低**：仅镜像版本变更时 |
| `build-release.sh` | 编译二进制 + 前端 + 组装 + 压缩，按版本归档到 `archive/` | **高**：每次发版 |
| `serve.sh` | 起 HTTP 服务暴露 `archive/`，供使用者浏览器下载 | 常驻 |

> ⚠️ `download-docker.sh` / `build-images.sh` / `build-release.sh` 都**用普通用户运行，不要 sudo**（见 §3）。

### 5.1 第一步：准备基础设施（首次 / Docker 或镜像版本变更时）

```bash
cd deployments/release

# (a) 下载 Docker 引擎离线包（供运维侧离线装 Docker；版本/地址见 release.conf）
./download-docker.sh               # → docker-cache/{amd64,arm64}/docker-<版本>.tgz
#   --force   已存在也重下

# (b) 拉取基础设施镜像
./build-images.sh                  # 基础设施版本取 release.conf 的 INFRA_VERSION
#   -v infra-1.1        手动指定基础设施版本
#   --with-monitoring   额外导出监控栈镜像
```

Docker 包与镜像缓存生成一次后可反复复用，日常发版无需重跑本步。

### 5.2 第二步：构建交付包（每次发版）

```bash
cd deployments/release
./build-release.sh                 # 小版本：版本自动生成
./build-release.sh -v 2.0.0        # 大版本：手动指定
#   --arch amd64        只构建指定架构（缺省 amd64 + arm64）
```

产物按版本归档：

```
archive/<项目版本>/
├── omc-release-<版本>-amd64.tar.xz   + .sha256
├── omc-release-<版本>-arm64.tar.xz   + .sha256
└── RELEASE.txt                       # 版本构建说明
```

并自动刷新 `archive/index.html`。压缩方式由 `release.conf` 的 `PKG_COMPRESS`
控制（默认 `xz`，体积最小）。

### 5.3 版本号规则（基础设施 / 项目 各自独立命名）

| 维度 | 取值 | 维护方式 |
|------|------|---------|
| **基础设施版本** | `release.conf` 的 `INFRA_VERSION`（或 `build-images.sh -v`） | 手动；改镜像清单后递增（如 `infra-1.0`→`infra-1.1`） |
| **项目小版本**（频繁） | `build-release.sh` 不带 `-v` → `<RELEASE_BASE_VERSION>-<YYYYMMDD-HHMM>` | 自动生成 |
| **项目大版本** | `build-release.sh -v 2.0.0` | 手动指定 |

- 二者**独立追溯**：每个交付包内 `VERSION` / `RELEASE.txt` 同时记录项目版本与
  基础设施版本，`RELEASE.txt` 还说明本次基础设施相对上次发布**是否更新**。
- 发大版本时，建议同步把 `release.conf` 的 `RELEASE_BASE_VERSION` 更新为该版本号。

### 5.4 第三步：起 HTTP 下载服务

构建产物归档在 `archive/` 后，用 `serve.sh` 在构建机上起一个 HTTP 服务，
使用者用浏览器（或 wget/curl）即可下载，无需登录构建机拷文件：

```bash
./serve.sh                         # 默认端口 8000
# 后台常驻： nohup ./serve.sh 8000 >/tmp/omc-serve.log 2>&1 &
```

使用者浏览器访问 `http://<构建机IP>:8000/` → `archive/index.html` 列出全部
版本（项目版本 / 基础设施版本 / 构建时间 / 下载链接）→ 点击下载对应架构的包。

---

## 6. 构建流程详解

> 本章是两个构建脚本内部流程的展开说明，便于理解与排错；也可据此手动构建。
> §6.3 属 `build-images.sh`（基础设施，不常跑）；其余属 `build-release.sh`（发版，常跑）。

### 6.1 编译 Go 二进制（amd64 + arm64 双架构）—— build-release.sh

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

### 6.2 构建前端（架构无关，只做一次）—— build-release.sh

```bash
cd omcmb/webcode
npm ci --include=dev --legacy-peer-deps   # 严格按 package-lock.json 安装
npm run build                              # 产出 dist/（纯静态）
```

两个架构的交付包复用同一份 `web/dist`。

### 6.3 收集 Docker 镜像（amd64 + arm64 双架构）—— build-images.sh

> 本步由独立工具 `build-images.sh` 完成，产物落 `images-cache/`，不常跑；
> `build-release.sh` 只是复用 `images-cache/` 里的 tar，不重复拉取。

Docker 镜像按架构区分。镜像导出**不依赖构建机自身架构**——用 `docker pull --platform`
显式逐架构拉取（拉取只下载分层、不执行，跨架构可行）。但 `docker save` 读的是
构建机本地镜像库，故每个架构在 pull 前先 `docker rmi` 清本地同名镜像，保证导出确定：

```bash
for ARCH in amd64 arm64; do
  for IMG in "${INFRA_IMAGES[@]}"; do            # 镜像清单来自 release.conf
    docker rmi -f "$IMG" >/dev/null 2>&1 || true # 清缓存，使 save 架构确定
    docker pull --platform linux/$ARCH "$IMG"
  done
  docker save -o images-cache/infra-images-$ARCH.tar "${INFRA_IMAGES[@]}"
done
```

> **跑完不留痕**：`build-images.sh` 在全部 `docker save` 完成后，会把拉进本地
> docker 镜像库的镜像 `docker rmi` 清掉。因为 `docker pull --platform` 会按架构
> 覆盖同名标签，残留会让本机的 `docker compose` 等操作误用到错架构镜像；清理后
> 本工具与本机其它 docker 操作互不影响。

### 6.4 组装、压缩、归档 —— build-release.sh

`build-release.sh` 逐架构组装交付包：编译产物 + 前端 + `data/configs/migrations/etc`
+ `bundle/` 部署模板 + **从 `images-cache/` 复用的镜像 tar** + 运维侧文档，
生成 `VERSION` / `README` / `checksums.sha256`，再**按 `PKG_COMPRESS` 压缩**
（默认 `xz`，体积最小）并**按版本归档**到 `archive/<版本>/`：

```bash
for ARCH in amd64 arm64; do
  PKG=omc-release-<版本>-$ARCH
  # 组装 bin/ web/ data/ configs/ migrations/ etc/ deploy/ docker/ docs/，
  # images/ 直接 cp images-cache/infra-images-$ARCH.tar
  ( cd "$PKG" && find . -type f ! -name checksums.sha256 -exec sha256sum {} + > checksums.sha256 )
  tar -cJf archive/<版本>/$PKG.tar.xz $PKG/        # xz 压缩，减小体积
  sha256sum $PKG.tar.xz > $PKG.tar.xz.sha256
done
```

打包后写 `archive/<版本>/RELEASE.txt`（版本构建说明：项目版本 / 基础设施版本 /
是否更新 / 镜像清单），并刷新 `archive/index.html` 供 `serve.sh` HTTP 下载。

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

- [ ] `images-cache/` 已由 `build-images.sh` 生成，基础设施版本为本次发布所需
- [ ] 项目版本号正确，`archive/<版本>/` 已生成、`RELEASE.txt` 内容无误
- [ ] amd64、arm64 两个交付包均已产出
- [ ] 镜像标签已固化为具体版本（非 `latest`），且 `deploy/.env` 与之一致
- [ ] `bundle/docker/` 已放入对应架构的 `docker-<版本>.tgz`
- [ ] 交付包与 `*.<压缩>.sha256` 校验通过
- [ ] `RELEASE.txt` 已说明本次基础设施版本及是否更新
- [ ] `archive/index.html` 已刷新，`serve.sh` 可正常下载
- [ ] 交付包内 `docs/` 含最新版《OMC内网离线部署手册（运维侧）》
- [ ] 在测试环境用交付包完整走通一遍部署（见运维侧手册）

---

**文档结束。** 端口、配置项、镜像版本以实际交付版本的源码 `config.prod.yaml`、
`deployments/` 为准。

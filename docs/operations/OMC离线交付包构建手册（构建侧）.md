# OMC 离线交付包构建手册（构建侧）

> **适用对象**：研发 / 发布工程师。
> **执行环境**：有公网、装有 Go / Node / Docker 的**构建机**。
> **产出**：两个互相独立的离线交付包——**项目包** `omc-<test|release>-<版本>-amd64.tar.xz`
> 与**基础设施包** `omc-infra-<版本>-amd64.tar.xz`，按版本归档到 `archive/`，可经
> HTTP 服务下载。
> **配套文档**：命令速查见《OMC交付构建快速上手》；运维侧部署见《OMC内网离线部署手册（运维侧）》。
> **配套工具**：构建工具位于仓库 `deployments/release/`。
> **文档状态**：v2.0（双包拆分），需随产品版本迭代同步维护。
>
> ⚠️ **架构支持**：本工具链当前**只支持 amd64（x86_64）**。`release.conf` 的
> `ARCHES` 固定为 `"amd64"`；三个构建脚本（download-docker.sh /
> build-images.sh / build-release.sh）均在参数解析阶段做了 amd64-only 校验，
> 传非 amd64 直接 die。原因：① 交付目标设备全为 amd64 服务器 ② arm64 跨架构
> 构建在 amd64 host 上要双倍磁盘（全栈 ~40GB）并易撑爆紧凑分区 ③ 运维侧未对
> arm64 包做过验证。**未来重启 arm64**：见 `release.conf` 中"架构支持"注释。

---

## 0. 两个独立交付包（v2.0 起）

交付物从「单个大包」拆成**两个互相独立、各自维护版本号**的包：

| 交付包 | 内容 | 由谁生成 | 文件名 | 发布频率 |
|--------|------|---------|--------|---------|
| **项目包** | OMC 二进制 + 前端 + 配置 + 数据库迁移 + 部署模板 | `build-release.sh` | `omc-<test\|release>-<版本>-<架构>.tar.xz` | **高**：每次发版 |
| **基础设施包** | Docker 引擎离线包 + 基础镜像（PG/Redis/NATS/MinIO/Nginx） | `build-images.sh` | `omc-infra-<版本>-<架构>.tar.xz` | **低**：仅基础设施变更时 |

**拆分动机**：基础设施镜像体积大、变更少；项目代码迭代快。合在一个包里，每次发版都
要重复分发上 GB 的镜像。拆开后日常发版只动项目包，基础设施包按需更新，两者版本号独立追溯。

**发布渠道**：项目包文件名带渠道前缀——`test`（测试阶段，当前默认）/ `release`（正式发布）。

**版本起点**：项目版本与基础设施版本均为纯 semver，当前测试阶段**从 `0.0.1` 起步**，不从 `1.0.0` 起。

---

## 1. 交付模式

OMC 采用**「构建侧编译、运维侧只跑二进制」**：

- **构建侧（本手册）**：在有公网的构建机一次性完成 Go 编译、前端打包、Docker 镜像收集，产出离线交付包。
- **运维侧**：拿到交付包后**零 Go 依赖、零 Node 依赖、全程不联网**——因为交付物是已编译的静态二进制 + 已打包的镜像。

| 组件 | 交付形态 | 归属包 |
|------|---------|--------|
| omcgo-app / omcgo-acs / omcgo-worker、omcgo-migrate / omcgo-seed、omcctl | 已编译 Linux 静态二进制（`CGO_ENABLED=0`，无动态库依赖） | 项目包 |
| 前端 webcode | 已编译静态资源（HTML/JS/CSS） | 项目包 |
| PostgreSQL/TimescaleDB、Redis、NATS、MinIO、nginx | `docker save` 离线镜像包 | 基础设施包 |
| Docker 引擎 | 官方静态二进制包（构建侧一次性下载） | 基础设施包 |

> 构建侧需要 Go 工具链 + Go 三方组件、Node/npm、Docker、公网；这些**运维侧一律不需要**。

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
| 架构产出 | **每个版本固定同时产出 amd64 与 arm64 两套**，对应两个架构的交付包 |
| 运行身份 | **普通用户**运行，不要 `sudo`（见下方说明） |

> ⚠️ **不要用 `sudo` 运行 `build-release.sh` / `build-images.sh`**：
> - 它们是构建脚本，`go build` / `npm build` 都不需要 root；
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

每个版本、每个架构产出对应交付包。下方为**解压后**的目录结构。

### 4.1 项目包 `omc-<test|release>-<版本>-<架构>/`

```
omc-<test|release>-<版本>-<架构>/
├── README.md                      # 项目包说明 + 版本号 + 渠道
├── VERSION                        # 项目版本 / 渠道 / 架构 / 构建时间 / git commit
├── checksums.sha256               # 全部文件 SHA256
│
├── bin/                           # ① OMC 已编译二进制（核心交付物）
│   └── omcgo-app / omcgo-acs / omcgo-worker / omcgo-migrate / omcgo-seed / omcctl
├── web/dist/                      # ② 前端已编译静态资源
├── etc/                           # ③ 配置模板（app/acs/worker.prod.yaml，需现场改）
├── data/                          # ④ 启动期加载的字典 XML
├── configs/                       # ⑤ Casbin RBAC 模型 casbin_model.conf
├── migrations/                    # ⑥ 数据库迁移 SQL（schema + seed/）
├── deploy/                        # ⑦ 部署脚本与模板
│   ├── docker-compose.infra.yml / docker-compose.web.yml / .env
│   ├── nginx.conf / default.conf
│   ├── systemd/                   #   omcgo-app/acs/worker.service
│   └── healthcheck.sh
└── docs/
    └── OMC内网离线部署手册（运维侧）.md   # 交给运维的部署文档
```

> 项目包**不含** Docker 引擎与基础镜像——它们在基础设施包里。`deploy/.env` 仍记录
> 基础设施镜像的标签（供 `docker-compose` 引用），这些镜像由基础设施包提供并 `docker load`。

### 4.2 基础设施包 `omc-infra-<版本>-<架构>/`

```
omc-infra-<版本>-<架构>/
├── README.md                      # 基础设施包说明 + 安装步骤
├── VERSION                        # 基础设施版本 / 架构 / Docker 版本 / 构建时间
├── checksums.sha256               # 全部文件 SHA256
│
├── docker/                        # ① Docker 引擎离线安装
│   ├── docker-<ver>.tgz           #   Docker 静态二进制包（官方）
│   └── install-docker.sh          #   离线安装脚本
└── images/                        # ② Docker 镜像离线包（docker save）
    ├── infra-images-<架构>.tar     #   postgres/redis/nats/minio/nginx
    ├── monitoring-images-<架构>.tar #  prometheus/grafana/loki/...（可选）
    └── images.manifest             #   镜像名:tag 清单
```

**产物来源对照**：

| 交付包目录 | 来源 | 归属 |
|-----------|------|------|
| `bin/` | 构建侧交叉编译（§6.1），每架构一套 | 项目包 |
| `web/dist/` | `npm run build`（§6.2） | 项目包 |
| `data/` `configs/` `migrations/` | 取自源码仓库 `omcgo/` 对应目录，随版本走 | 项目包 |
| `etc/*.prod.yaml` | 源码 `omcgo/cmd/{app,acs,worker}/etc/config.prod.yaml`，**模板** | 项目包 |
| `deploy/` | 取自 `deployments/release/bundle/deploy/` + `deployments/docker/` 的 nginx 配置 | 项目包 |
| `images/*.tar` | `docker pull --platform` + `docker save`（§6.3） | 基础设施包 |
| `docker/` | `docker-cache/`（`download-docker.sh` 下载）+ `bundle/docker/install-docker.sh` | 基础设施包 |

---

## 5. 构建工具与流程

构建工具位于 `deployments/release/`，**五个脚本**——基础设施准备（Docker 引擎、
基础设施包）与项目发版（构建周期不同），加索引生成与下载服务：

```
deployments/release/
├── release.conf              # 配置：版本号、发布渠道、压缩方式、Docker 版本、镜像清单
├── download-docker.sh        # ① Docker 引擎离线包下载（不常跑）
├── build-images.sh           # ② 基础设施包构建（不常跑）→ archive/infra/
├── build-release.sh          # ③ 项目包构建 / 发版（常跑）→ archive/project/
├── gen-index.sh              # ④ 下载索引生成（②③ 自动调用，也可手动）
├── serve.sh                  # ⑤ HTTP 下载服务
├── bundle/                   # 进交付包的静态模板（docker/install-docker.sh + deploy/）
├── docker-cache/             # ① 的产物：docker-cache/<arch>/docker-<版本>.tgz（git 忽略）
├── images-cache/             # ② 的中间产物（基础镜像 tar，git 忽略）
├── dist/                     # 构建临时工作区（git 忽略）
└── archive/                  # 版本化归档 = HTTP 服务根目录（git 忽略）
    ├── index.html            #   自动生成的下载索引（两张表 + 操作步骤）
    ├── project/<项目版本>/ …  #   项目包按版本归档
    └── infra/<基础设施版本>/ … #   基础设施包按版本归档
```

| 工具 | 职责 | 运行频率 |
|------|------|---------|
| `download-docker.sh` | 按 `release.conf` 下载 Docker 引擎离线包 → `docker-cache/` | **低**：仅 Docker 版本变更时 |
| `build-images.sh` | 拉取基础设施镜像 + 组装压缩**基础设施包** → `archive/infra/` | **低**：仅基础设施变更时 |
| `build-release.sh` | 编译二进制 + 前端 + 组装压缩**项目包** → `archive/project/` | **高**：每次发版 |
| `gen-index.sh` | 扫描 `archive/` 生成下载索引 `index.html` | 自动 |
| `serve.sh` | 起 HTTP 服务暴露 `archive/`，供使用者浏览器下载 | 常驻 |

> ⚠️ `download-docker.sh` / `build-images.sh` / `build-release.sh` 都**用普通用户运行，不要 sudo**（见 §3）。

### 5.1 第一步：构建基础设施包（首次 / Docker 或镜像版本变更时）

```bash
cd deployments/release

# (a) 下载 Docker 引擎离线包（会打进基础设施包；版本/地址见 release.conf）
./download-docker.sh               # → docker-cache/{amd64,arm64}/docker-<版本>.tgz
#   --force   已存在也重下

# (b) 拉取基础设施镜像 + 组装压缩基础设施包
./build-images.sh                  # v2 默认：基础设施 + 监控栈全套
#   -v 0.0.2            手动指定基础设施版本
#   --infra-only        仅基础设施（不要监控栈）
#   --monitoring-only   只补监控栈镜像（不重拉基础设施）并重新打包
#   --with-monitoring   【已废弃】保留兼容；v2 默认即含监控栈，本标志为 no-op
```

产出 `archive/infra/<基础设施版本>/omc-infra-<版本>-<架构>.tar.xz`。基础设施不常变，
生成一次后日常发版无需重跑本步。

### 5.2 第二步：构建项目包（每次发版）

```bash
cd deployments/release
./build-release.sh                 # 小版本：版本自动生成；渠道取 release.conf 的 RELEASE_CHANNEL
./build-release.sh -v 1.0.0        # 大版本：手动指定版本
./build-release.sh --channel release   # 临时覆盖渠道（默认 test）
#   --arch amd64        只构建指定架构（缺省 amd64 + arm64）
```

产物按版本归档：

```
archive/project/<项目版本>/
├── omc-<test|release>-<版本>-amd64.tar.xz   + .sha256
├── omc-<test|release>-<版本>-arm64.tar.xz   + .sha256
└── RELEASE.txt                              # 版本构建说明
```

并自动刷新 `archive/index.html`。压缩方式由 `release.conf` 的 `PKG_COMPRESS`
控制（默认 `xz`，体积最小）。

### 5.3 版本号与渠道规则（基础设施 / 项目 各自独立）

| 维度 | 取值 | 维护方式 |
|------|------|---------|
| **基础设施版本** | `release.conf` 的 `INFRA_VERSION`（或 `build-images.sh -v`） | 手动；改镜像/Docker 版本后递增（如 `0.0.1`→`0.0.2`） |
| **项目小版本**（频繁） | `build-release.sh` 不带 `-v` → `<RELEASE_BASE_VERSION>-<YYYYMMDD-HHMM>` | 自动生成 |
| **项目大版本** | `build-release.sh -v 1.0.0` | 手动指定 |
| **发布渠道** | `release.conf` 的 `RELEASE_CHANNEL`（或 `build-release.sh --channel`） | `test`=测试阶段 / `release`=正式发布 |

- 版本号均为纯 semver，当前测试阶段**从 `0.0.1` 起步**。
- 项目版本与基础设施版本**各自独立追溯**，互不耦合；各包内 `VERSION` / `RELEASE.txt`
  记录自己的版本与构建信息。
- 渠道决定项目包文件名前缀：`test` → `omc-test-…`，`release` → `omc-release-…`。
- 发大版本时，建议同步把 `release.conf` 的 `RELEASE_BASE_VERSION` 更新为该版本号。

### 5.4 第三步：起 HTTP 下载服务

构建产物归档在 `archive/` 后，用 `serve.sh` 在构建机上起一个 HTTP 服务，
使用者用浏览器（或 wget/curl）即可下载，无需登录构建机拷文件：

```bash
./serve.sh                         # 默认端口 8000
# 后台常驻： nohup ./serve.sh 8000 >/tmp/omc-serve.log 2>&1 &
```

使用者浏览器访问 `http://<构建机IP>:8000/` → 下载页含**操作步骤指引** +
**项目交付包**、**基础设施包** 两张版本列表 → 按目标机架构点击下载对应包。

---

## 6. 构建流程详解

> 本章是构建脚本内部流程的展开说明，便于理解与排错。
> §6.3 属 `build-images.sh`（基础设施包，不常跑）；§6.1/6.2/6.4 属 `build-release.sh`（项目包，常跑）。

### 6.1 编译 Go 二进制（amd64 单架构）—— build-release.sh

每个版本编译 amd64 一份。构建机若是 amd64 host 自然原生编译；若是其它 host，
Go 原生支持交叉编译到 amd64：

```bash
cd omcgo
for SVC in app acs worker migrate seed; do
  CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" \
    -o bin/amd64/omcgo-$SVC ./cmd/$SVC
done
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" \
  -o bin/amd64/omcctl ./cmd/omcctl
```

> Go 第三方组件（见 `go.mod`）在此步由 `go build` 自动拉取并**编译进二进制**，运维侧不再需要它们。

### 6.2 构建前端（架构无关，只做一次）—— build-release.sh

```bash
cd omcmb/webcode
npm ci --include=dev --legacy-peer-deps   # 严格按 package-lock.json 安装
npm run build                              # 产出 dist/（纯静态）
```

amd64 项目包包含同一份 `web/dist`。

### 6.3 收集 Docker 镜像并打包基础设施包（amd64）—— build-images.sh

> 本步由独立工具 `build-images.sh` 完成，不常跑；产出基础设施包到 `archive/infra/`。

`build-images.sh` 启动时双校验：① `ARCHES`（来自 `release.conf`）必须仅 `amd64`；
② host 架构（`uname -m`）必须是 `x86_64/amd64` —— amd64-only refactor 后已移除跨架构
pull 代码路径，非 amd64 host 直接 die。

**镜像缓存策略（amd64-only）**：

1. **命名兼容（`*-amd64-saved` tag）**：镜像以 `<image>-amd64-saved` 后缀 tag 长期
   保留（如 `redis:7-alpine-amd64-saved`）。下次构建命中即跳过 `docker pull`。
   `-amd64-` 段命名作历史缓存兼容保留；不再有跨架构需求。
2. **不再 `docker rmi`**：脚本结束后**不清理本地镜像库**，`*-saved` tag 留作下次
   增量构建的本地缓存。如需手工清理：
   ```bash
   docker images | grep -- '-saved' | awk '{print $1":"$2}' | xargs docker rmi -f
   ```
3. **tar 双 tag 设计**：`infra-images-amd64.tar` / `monitoring-images-amd64.tar`
   内同时包含 `<image>-amd64-saved` 与原 tag 两套引用。运维侧 `docker load` 后两个
   引用都存在，**`docker-compose.yml` 直接 `image: redis:7-alpine` 即可**，完全不
   必关心 `*-saved`。

伪代码（实现细节见 `build-images.sh` 中 `prepare_image` / `save_with_dual_tags`）：

```bash
ARCH=amd64

# 1) 拉取并打 *-saved tag（命中缓存即跳过）
for IMG in "${INFRA_IMAGES[@]}"; do                # 清单来自 release.conf
  SAVED_TAG="${IMG}-${ARCH}-saved"
  docker image inspect "$SAVED_TAG" >/dev/null 2>&1 && continue   # ✓ 复用本地缓存
  docker pull --platform linux/$ARCH "$IMG"
  docker tag "$IMG" "$SAVED_TAG"
done

# 2) save 前让原 tag 指向 *-saved，使 tar 内同时含两套引用
for IMG in "${INFRA_IMAGES[@]}"; do
  docker tag "${IMG}-${ARCH}-saved" "$IMG"
done
docker save -o images-cache/infra-images-$ARCH.tar \
  $(for IMG in "${INFRA_IMAGES[@]}"; do echo "${IMG}-${ARCH}-saved"; echo "$IMG"; done)

# 3) 组装：images/infra-images-$ARCH.tar + docker/(docker-*.tgz + install-docker.sh)
#         + 顶层 setup-mirrors.sh
#    → 压缩 → archive/infra/<版本>/omc-infra-<版本>-amd64.tar.xz
```

> **不再跑完即清**：`build-images.sh` 不再 `docker rmi` 任何镜像。`*-saved` 后缀
> tag 与原 tag 都指向 amd64 镜像，因此本工具与同台机器上的 `docker compose` 等操作
> 可并行运行，互不影响。

### 6.4 组装、压缩、归档项目包 —— build-release.sh

`build-release.sh` 逐架构组装项目包：编译产物 + 前端 + `data/configs/migrations/etc`
+ `bundle/deploy` 部署模板 + 运维侧文档，生成 `VERSION` / `README` /
`checksums.sha256`，再**按 `PKG_COMPRESS` 压缩**（默认 `xz`）并**按版本归档**：

```bash
for ARCH in amd64 arm64; do
  PKG=omc-<渠道>-<版本>-$ARCH
  # 组装 bin/ web/ data/ configs/ migrations/ etc/ deploy/ docs/（不含 images/docker）
  ( cd "$PKG" && find . -type f ! -name checksums.sha256 -exec sha256sum {} + > checksums.sha256 )
  tar -cJf archive/project/<版本>/$PKG.tar.xz $PKG/        # xz 压缩
  sha256sum $PKG.tar.xz > $PKG.tar.xz.sha256
done
```

打包后写 `archive/project/<版本>/RELEASE.txt`（项目版本 / 渠道 / git commit /
交付文件清单），并调 `gen-index.sh` 刷新 `archive/index.html` 供 `serve.sh` HTTP 下载。

---

## 7. 镜像版本固化（重要）

`release.conf` 中的基础设施镜像标签，**发布前必须固化为具体版本号**，
不要用 `latest` / `latest-pg16`：

- 浮动标签会导致**不同批次基础设施包镜像不一致**，难以复现问题。
- 镜像标签同时写入项目包 `deploy/.env`，供 `docker-compose` 引用——必须与
  基础设施包里 `docker save` 的镜像保持一致。
- 每次改镜像版本，记得同步递增 `release.conf` 的 `INFRA_VERSION`。

> 例：`timescale/timescaledb:latest-pg16` → `timescale/timescaledb:2.17.x-pg16`；
> `minio/minio:latest` → `minio/minio:RELEASE.<具体日期>`。

---

## 8. 附：构建机也离线 / 现场源码编译

默认构建机有公网即可。两类特殊场景：

**A. 构建机本身离线**：先在有网环境 `cd omcgo && go mod vendor` 把 Go
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

**基础设施包**（仅基础设施变更时重出）：

- [ ] `download-docker.sh` 已下载对应 `DOCKER_VERSION` 的 Docker 引擎包
- [ ] `build-images.sh` 已产出 `archive/infra/<版本>/`，amd64 + arm64 两个包齐全
- [ ] 镜像标签已固化为具体版本（非 `latest`），`INFRA_VERSION` 已对应递增
- [ ] 基础设施包内 `images.manifest`、`docker/docker-*.tgz` 齐全

**项目包**（每次发版）：

- [ ] 项目版本号与发布渠道正确（`test` / `release`）
- [ ] `archive/project/<版本>/` 已生成，amd64 + arm64 两个包齐全，`RELEASE.txt` 无误
- [ ] `deploy/.env` 镜像标签与基础设施包内镜像一致
- [ ] 交付包内 `docs/` 含最新版《OMC内网离线部署手册（运维侧）》

**通用**：

- [ ] 交付包与 `*.<压缩>.sha256` 校验通过
- [ ] `archive/index.html` 已刷新，`serve.sh` 可正常下载，下载页操作步骤显示正常
- [ ] 在测试环境用项目包 + 基础设施包完整走通一遍部署（见运维侧手册）

---

## 10. 脚本帮助对照表（`-h` 全覆盖）

每个脚本均支持 `-h | --help` 看完整用法。下表用于快速查阅：

| 脚本 | 角色 | 关键参数 | 用途 |
|------|------|---------|------|
| `download-docker.sh` | 构建侧 | `--arch amd64\|arm64` / `--force` | 下载 Docker 静态二进制到 `docker-cache/` |
| `build-images.sh` | 构建侧 | `-v <版本>` / `--infra-only` / `--monitoring-only`（v2 默认含监控栈） | 拉镜像 + 组装基础设施包 |
| `build-release.sh` | 构建侧 | `-v <版本>` / `--channel test\|release` / `--arch` | 编译 + 组装项目包 |
| `gen-index.sh` | 构建侧 | `--archive <dir>` | 重生成 `archive/index.html`（由前两个脚本自动调） |
| `serve.sh` | 构建侧 | `-p\|--port <PORT>` | 起 HTTP 下载服务（默认 8000） |
| `bundle/docker/install-docker.sh` | 交付侧 | `--mirror <name>` / `--no-mirror` / `--skip-if-installed` | 离线装 Docker + 引导加速（末尾自动调 setup-mirrors.sh） |
| `bundle/setup-mirrors.sh` | 交付侧 | `--docker <official\|daocloud\|xuanyuan>` / `--npm <official\|taobao>` / `--golang <official\|goproxycn>` / `--show` / `--remove` | 单独配 / 换 / 查 / 取消 **Docker / npm / Golang** 三项加速（infra 包顶层；运行时落到 `/opt/omc/infra/setup-mirrors.sh`） |
| `bundle/deploy/deploy.sh` | 交付侧 | `--skip-infra` / `--skip-migrate` / `--skip-web` / `--check-only` / `--yes` | 一键部署 OMC 全栈 |
| `bundle/deploy/healthcheck.sh` | 交付侧 | — | 部署完成后健康校验 |

---

## 11. 系统加速设置（Docker / npm / Golang 三合一）

构建侧 `download-docker.sh` 把 Docker 引擎二进制装进 `docker-cache/`，
`build-images.sh` 同时把 `install-docker.sh` 与 `setup-mirrors.sh` 打入
基础设施包 `docker/` 目录。交付侧 `install-docker.sh` **离线**装完 Docker 后
**主动引导**用户配置加速；`setup-mirrors.sh` 单独运行时一次性配齐三项。

### 11.1 三个目标 / 三个推荐

| 目标 | 内置选项 | 推荐值 | 落地位置 |
|------|---------|--------|---------|
| **Docker** | `official` / `daocloud` / `xuanyuan` | `daocloud` (`https://docker.m.daocloud.io`) | `/etc/docker/daemon.json` 的 `registry-mirrors` |
| **npm** | `official` / `taobao` | `taobao` (`https://registry.npmmirror.com`) | `/etc/npmrc`（含 `disturl` 指向 npmmirror node 镜像） |
| **Golang** | `official` / `goproxycn` | `goproxycn` (`https://goproxy.cn,direct` + `GOSUMDB=sum.golang.google.cn`) | `/etc/profile.d/goproxy.sh` |

三项相互独立，可任意组合启用 / 跳过；任一项选 `official` 即"不设置，回归官方"。

### 11.2 入口

- 入口 1：`install-docker.sh` 末尾交互菜单（仅 Docker；批处理可 `--mirror <name>` 跳过）
- 入口 2：`setup-mirrors.sh` 交互式依次问 Docker / npm / Golang 三项
- 入口 3：`setup-mirrors.sh --docker daocloud --npm taobao --golang goproxycn` 非交互一次过
- 任意时刻可用 `setup-mirrors.sh --show` / `--remove` 查看 / 取消三项配置

### 11.3 实现要点

- **Docker**：用 `python3` 合并 `/etc/docker/daemon.json`，**保留其它键**；自动
  备份 `daemon.json.bak.<时间戳>`；内容确变时 `systemctl restart docker`，未变则跳过
- **npm**：写**系统级** `/etc/npmrc`（所有用户生效），不动用户 `~/.npmrc`
- **Golang**：写 `/etc/profile.d/goproxy.sh`，导出 `GOPROXY` / `GOSUMDB`；新 shell
  自动加载，当前 shell 需 `source` 或重新登录

> 纯离线场景（镜像 / 包 / 模块都已随交付包给出）三项加速对**首次部署**没有影响；
> 主要利于运维侧后续临时 `docker pull` / `npm install` / `go install` 提速。
> **因此 install 默认主动询问**。客户网络绝对不可外出时三项均选 `official` 即可。

---

## 12. 一键部署 `deploy.sh`（交付侧）

交付侧把项目包解压后，进入项目包根目录运行：

```bash
sudo bash deploy/deploy.sh                    # 全套首次部署
sudo bash deploy/deploy.sh --skip-infra       # 已部署基础设施时仅升级 app
sudo bash deploy/deploy.sh --check-only       # 只 precheck 不动手
sudo bash deploy/deploy.sh -h                 # 查看完整参数
```

10 步自动化：precheck → 建立 `/opt/omc/{infra,releases,etc,current}` 布局 →
load 基础镜像 → 默认口令检查 → infra compose up → 等就绪（PG + Redis ≤ 90s）→
db migrate → db seed（首次自动打 mark 文件）→ install systemd（app/acs/worker，
开机自启）→ web compose up → 调 healthcheck.sh。

**幂等**：所有步骤均可重跑；二次运行视情况跳过已完成项：
- seed 已跑过 → 跳过（按 `/opt/omc/etc/.seed.done` mark 文件判断）
- systemd 单元已存在 → 差异比对后备份覆盖（`.bak.<时间戳>`）
- 镜像已 load → docker 自动跳过同 digest

---

**文档结束。** 端口、配置项、镜像版本以实际交付版本的源码 `config.prod.yaml`、
`deployments/` 为准。

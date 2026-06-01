# OMC 交付构建快速上手

> **适用对象**：在构建机上制作、分发 OMC 离线交付包的研发 / 发布工程师。
> **本文定位**：命令驱动的操作速查。完整原理、交付包结构、环境要求见
> 《OMC离线交付包构建手册（构建侧）》；运维部署见《OMC内网离线部署手册（运维侧）》。
> **工具位置**：`deployments/release/`。
>
> ⚠️ **部署模式（v3）**：交付物已切到**全 docker compose 部署**——所有 OMC 服务
> （app / acs / worker / web / 监控栈）都以容器运行，业务镜像由 `docker build` +
> `docker save` 产出并随项目包分发，**构建机不再需要 Go / Node，运维机不再放业务
> 二进制 / systemd 单元**。构建机只需 `docker` + `tar` + `sha256sum`。
>
> ⚠️ **架构**：当前工具链**只支持 amd64（x86_64）**。三个构建脚本均在参数解析阶段
> 校验，传非 amd64 直接拒绝。原因与重启 arm64 的步骤见 `release.conf` 的"架构支持"注释。

---

## 0. 两个独立交付包

交付物拆成**两个互相独立、各自维护版本号**的包：

| 交付包 | 内容 | 由谁生成 | 文件名 |
|--------|------|---------|--------|
| **基础设施包** | Docker 引擎离线包 + Compose v2 + Buildx + 基础镜像（PG/Redis/NATS/MinIO/Nginx）+（默认）监控栈镜像 | `build-images.sh` | `omc-infra-<版本>-amd64.tar.xz` |
| **项目包** | OMC 业务镜像 tar（app/acs/worker/web）+ 配置 + 数据库迁移 + 字典 + 部署模板 + 监控配置 | `build-release.sh` | `omc-<test\|release>-<版本>-amd64.tar.xz` |

基础设施不常变、发布频率低；项目包每次发版都出。运维首次部署两个都下，之后日常升级通常只更新项目包。

---

## 1. 五个脚本一览

| 脚本 | 干什么 | 多久跑一次 | 依赖 |
|------|--------|-----------|------|
| `download-docker.sh` | 下载 Docker 引擎 + Compose v2 + Buildx 离线包 → `docker-cache/` | **低**：仅 Docker 版本变更时 | curl/wget + 公网 |
| `build-images.sh` | 拉取基础设施（默认含监控栈）镜像 + 组装压缩**基础设施包** → `archive/infra/` | **低**：仅基础设施变更时 | docker + 公网 |
| `build-release.sh` | `docker build` 业务镜像 + 组装压缩**项目包** → `archive/project/` | **高**：每次发版 | docker + 公网 |
| `gen-index.sh` | 扫描 `archive/` 生成下载索引 `index.html` | 自动（②③ 调用） | bash |
| `serve.sh` | 起 HTTP 服务，让使用者用浏览器下载交付包 | 常驻 | python3 |

> 一句话：**`download-docker.sh`/`build-images.sh` 偶尔跑、`build-release.sh` 每次发版跑、`serve.sh` 一直开着；`gen-index.sh` 自动调用。**

> ⚠️ `build-images.sh` / `build-release.sh` 都**用普通用户运行，不要 `sudo`**。`sudo`
> 会重置 `PATH` 并让产物（`dist/`、缓存）归 `root`。`docker` 权限用 **docker 组**解决：
> `sudo usermod -aG docker $USER && newgrp docker`（见 §2）。

---

## 2. 首次准备（一次性）

```bash
cd <仓库>/deployments/release

# (1) 把当前用户加入 docker 组（build-images.sh / build-release.sh 都要用 docker）
sudo usermod -aG docker $USER && newgrp docker

# (2) 按需编辑 release.conf：
#     DOCKER_VERSION / COMPOSE_VERSION / BUILDX_VERSION   离线 Docker 三件套版本
#     DOCKER_URL_TEMPLATE                                 Docker 下载地址（按网络环境只放开一行）
#     INFRA_VERSION         基础设施版本号（纯 semver，当前 0.1.0）
#     RELEASE_BASE_VERSION  项目基线版本号（纯 semver，当前 0.1.0）
#     RELEASE_CHANNEL       发布渠道：test=测试阶段 / release=正式发布
#     IMAGE_*               基础设施 / 监控镜像 —— 发布前把 latest 改成具体版本号
#     PKG_COMPRESS          交付包压缩方式（默认 xz，体积最小）

# (3) 下载 Docker 引擎 + Compose v2 + Buildx 离线包（会打进基础设施包，供运维侧离线装 Docker）
./download-docker.sh        # 按 release.conf 下载 amd64 → docker-cache/amd64/
```

---

## 3. 场景化用法

### 场景 A —— 日常发小版本（最常用）

基础设施没变，只出一个新的项目版本：

```bash
./build-release.sh
```

版本号自动生成 = `<RELEASE_BASE_VERSION>-<构建时间戳>`，渠道取 `release.conf` 的
`RELEASE_CHANNEL`（默认 `test`）。产出如 `omc-test-0.1.0-20260518-1030-amd64.tar.xz`，
归档到 `archive/project/0.1.0-20260518-1030/`。脚本内部对每个业务服务跑
`docker build`（app/acs/worker/web），`docker save` 进 `images/`，再组装压缩。

### 场景 B —— 发大版本 / 切正式渠道

```bash
./build-release.sh -v 1.0.0                  # 大版本号（仍自动追加时间戳保证镜像 tag 唯一）
./build-release.sh --channel release         # 切到正式渠道 → omc-release-...
./build-release.sh -v 1.0.0 --channel release # 两者一起
```

发大版本时建议同步把 `release.conf` 的 `RELEASE_BASE_VERSION` 改成 `1.0.0`，让之后的小版本基线对齐。

### 场景 C —— 升级基础设施（镜像 / Docker）

基础设施包与项目包独立，升级基础设施只需重出基础设施包：

```bash
# 1) 编辑 release.conf：递增 INFRA_VERSION（如 0.1.0 → 0.1.1），更新 IMAGE_* / DOCKER_VERSION
# 2) 若改了 DOCKER_VERSION / COMPOSE_VERSION / BUILDX_VERSION，重新下载离线三件套
./download-docker.sh
# 3) 重建基础设施包（默认含监控栈）
./build-images.sh                  # → archive/infra/0.1.1/omc-infra-0.1.1-amd64.tar.xz
```

项目包不受影响，无需重出。

> **镜像缓存复用**：`build-images.sh` 通过 `<image>:<tag>-amd64-saved` 后缀 tag
> 把镜像与本机原 tag **命名隔离**，构建结束后**不再 `docker rmi` 清理**——
> `*-saved` tag 留作本地缓存，下次构建命中即跳过 `docker pull`；本机原 tag 始终
> 指向本机镜像，可与同主机的 `docker compose` 并存运行，互不干扰。详见构建
> 手册 §6.3。

### 场景 D —— 监控栈裁剪

`build-images.sh` **默认即含监控栈**（v2 起）。要去掉监控栈或后补：

```bash
./build-images.sh                       # 默认：基础设施 + 监控栈全套
./build-images.sh --infra-only          # 仅基础设施，不含监控栈
./build-images.sh --monitoring-only     # 已建好基础设施、只补监控栈镜像（不重拉基础设施）并重新打包
# ./build-images.sh --with-monitoring   # 【已废弃】v2 默认即含监控栈，本标志为 no-op + 提示
```

### 场景 E —— 显式指定架构

```bash
./build-images.sh  --arch amd64
./build-release.sh --arch amd64
```

当前仅支持 amd64，传其它值会被拒绝。

---

## 4. 分发：起 HTTP 下载服务

构建产物归档到 `archive/` 后，起服务供使用者下载。`serve.sh` 支持**前台**与**后台守护**两种模式：

```bash
# 前台（Ctrl-C 停止）
./serve.sh                # 默认端口 8000
./serve.sh -p 9000        # 指定端口

# 后台守护（关 ssh 不掉；setsid detach；写 .serve.pid/.serve.port/.serve.log）
./serve.sh start          # 启动
./serve.sh start -p 9000  # 启动并指定端口
./serve.sh status         # 查看运行状态
./serve.sh restart        # 重启（不传 -p 沿用上次端口）
./serve.sh stop           # 停止
```

> `build-release.sh` / `build-images.sh` 结束时会自动 `serve.sh restart`，让新包立即可下载。
> 需要**开机自启 / 崩溃重启**用 systemd unit：`deployments/release/omc-serve.service`，
> 安装方法见 `deployments/release/README.md`「下载服务长期运行」。

使用者（运维）浏览器访问 `http://<构建机IP>:8000/` → 下载页含**操作指引**（系统准备 / 校验 /
解压 / 装 Docker / 加速 / 一键部署 / 验证 / 访问 / 账号 / 配置 / 运维）+ **项目包**、
**基础设施包** 两张版本列表 → 按目标机架构点击下载。也可用 `wget`/`curl`。

---

## 5. 版本号与渠道

| 维度 | 取值 | 谁定 |
|------|------|------|
| 基础设施版本 | `release.conf` 的 `INFRA_VERSION`，或 `build-images.sh -v`（均自动追加时间戳） | 手动，改基础设施后递增 |
| 项目小版本 | `build-release.sh` 不带 `-v` → `<基线>-<时间戳>` | 自动 |
| 项目大版本 | `build-release.sh -v X.Y.Z`（仍自动追加时间戳） | 手动 |
| 发布渠道 | `release.conf` 的 `RELEASE_CHANNEL`，或 `build-release.sh --channel` | `test` / `release` |

版本号均为纯 semver，**从 0.1.0 起步**（当前测试阶段，不从 1.0.0 起）。所有版本号最终都会
追加 `-YYYYMMDD-HHMM` 时间戳，保证镜像 tag 唯一（使 deploy.sh 的 `images_exist` 智能跳过逻辑安全）。
基础设施版本与项目版本**各自独立**，互不耦合；各包内 `VERSION` / `RELEASE.txt` 记录自己的版本与构建信息。

---

## 6. 产物在哪

```
deployments/release/
├── docker-cache/                          # download-docker.sh 产物（Docker 引擎/compose/buildx，可复用）
├── images-cache/                          # build-images.sh 中间产物（基础镜像 + 监控镜像 tar）
└── archive/                               # 交付包归档 = HTTP 服务根目录
    ├── index.html                         #   HTTP 下载首页（自动生成）
    ├── project/<项目版本>/
    │   ├── omc-<test|release>-<版本>-amd64.tar.xz  + .sha256
    │   └── RELEASE.txt
    └── infra/<基础设施版本>/
        ├── omc-infra-<版本>-amd64.tar.xz   + .sha256
        └── RELEASE.txt
```

交给运维：让运维从 HTTP 服务下载（首次部署两个包都下），或把 `archive/` 对应目录拷到
U 盘/介质送入内网。运维侧照项目包内 `docs/OMC内网离线部署手册（运维侧）.md` 部署。

---

## 7. 完整流程示例

```bash
cd deployments/release

# —— 首次：准备 + 下载 Docker 三件套 + 建基础设施包 ——
sudo usermod -aG docker $USER && newgrp docker
# （编辑 release.conf）
./download-docker.sh                     # 下载 Docker 引擎 + compose + buildx 离线包
./build-images.sh                        # → archive/infra/0.1.0-<时间戳>/（默认含监控栈）

# —— 日常：发项目版本 ——
./build-release.sh                       # 小版本（test 渠道，docker build 业务镜像）
#   → archive/project/0.1.0-20260518-1030/

# —— 分发 ——
./serve.sh start                         # 后台起服务，保持开启

# —— 某次升级了基础设施 / Docker ——
# 改 release.conf（递增 INFRA_VERSION）后：
./download-docker.sh                     # 若改了 DOCKER_VERSION / COMPOSE_VERSION / BUILDX_VERSION
./build-images.sh                        # 重出基础设施包，项目包不受影响
```

---

## 8. 常见问题

| 现象 | 原因 / 处理 |
|------|------------|
| `docker 不可用：当前用户可能不在 docker 组` | `sudo usermod -aG docker $USER && newgrp docker` 后重试 |
| `the --mount option requires BuildKit` | 业务 Dockerfile 用了 BuildKit 缓存语法；脚本已默认 `DOCKER_BUILDKIT=1`，老 docker（≥18.09）都支持，确认 docker 版本 |
| `dial tcp: lookup goproxy.cn ... i/o timeout`（build 时容器内 DNS 超时） | bridge 网络被 ufw/fail2ban 改坏；`build-release.sh` 已用 `--network host` 规避，确认宿主机本身能联网 |
| `--channel 取值非法` | `--channel` 只接受 `test` 或 `release` |
| `本工具仅支持 amd64 架构` | 当前工具链 amd64-only，传非 amd64 会被拒绝（见 `release.conf` 注释） |
| `build-images.sh` 警告"docker-cache 无 Docker 引擎包 / 无 compose / 无 buildx" | 先跑一次 `./download-docker.sh` |
| 交付包很大 | `release.conf` 设 `PKG_COMPRESS=xz`（默认，体积最小）；gzip 更快但略大 |
| 浏览器打不开下载页 | `serve.sh status` 看是否在跑；构建机防火墙是否放行该端口 |
| 不同批次镜像不一致 | `release.conf` 的镜像标签别用 `latest`，固化为具体版本号 |
| 本地 `*-saved` 镜像太多想清理 | 这些是 `build-images.sh` 的本地缓存（命名隔离、不影响 docker compose），可保留以加速下次构建；如需手工回收：`docker images \| grep -- '-saved' \| awk '{print $1":"$2}' \| xargs docker rmi -f`，或 `docker image prune` |

---

**速记**：`download-docker.sh` + `build-images.sh`（偶尔，出基础设施包）→ `build-release.sh`（每次发版，`docker build` 出项目包）→ `serve.sh`（常开）。
深入内容见《OMC离线交付包构建手册（构建侧）》。

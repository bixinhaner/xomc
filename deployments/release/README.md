# deployments/release — OMC 离线交付包

> ⚠️ **架构支持**：当前发布工具链**只支持 amd64（x86_64）**。`download-docker.sh` /
> `build-images.sh` / `build-release.sh` 三个脚本都会在参数解析阶段校验，传入
> 非 amd64 会被拒绝。原因见 `release.conf` 中"架构支持"注释；未来若要重启
> arm64 需要：① 改 `ARCHES` 回双架构 ② 移除三脚本的校验 ③ 准备 ≥40GB Docker
> 数据盘 ④ 在 arm64 目标设备上完整跑一遍 deploy + healthcheck 验证。

本目录用于**生成、归档、分发交付给运维的离线交付包**。配套文档：

- **快速上手**（怎么用本目录的脚本）：[`docs/operations/OMC交付构建快速上手.md`](../../docs/operations/OMC交付构建快速上手.md)
- 构建侧完整说明：[`docs/operations/OMC离线交付包构建手册（构建侧）.md`](../../docs/operations/OMC离线交付包构建手册（构建侧）.md)
- 运维侧（随交付包发给运维）：[`docs/operations/OMC内网离线部署手册（运维侧）.md`](../../docs/operations/OMC内网离线部署手册（运维侧）.md)

## 两个独立交付包

交付物拆成**两个互相独立、各自维护版本号**的包：

| 交付包 | 内容 | 由谁生成 | 文件名 |
|--------|------|---------|--------|
| **基础设施包** | Docker 引擎离线安装包 + 基础镜像（PostgreSQL/Redis/NATS/MinIO/Nginx） | `build-images.sh` | `omc-infra-<版本>-<架构>.tar.xz` |
| **项目包** | OMC 二进制 + 前端 + 配置 + 数据库迁移 + 部署模板 | `build-release.sh` | `omc-<test\|release>-<版本>-<架构>.tar.xz` |

基础设施不常变，发布频率低；项目包每次发版都出。首次部署两个都要；之后日常升级通常只更新项目包。

## 五个脚本

| 脚本 | 职责 | 运行频率 |
|------|------|---------|
| `download-docker.sh` | 按 `release.conf` 下载 Docker 引擎离线包 → `docker-cache/` | **低**：仅 Docker 版本变更时 |
| `build-images.sh` | 拉取基础设施镜像 + 组装压缩**基础设施包** → `archive/infra/` | **低**：仅基础设施变更时 |
| `build-release.sh` | 编译二进制 + 前端 + 组装压缩**项目包** → `archive/project/` | **高**：每次发版 |
| `gen-index.sh` | 扫描 `archive/` 生成下载索引 `index.html`（被上面两个脚本自动调用） | 自动 |
| `serve.sh` | 起 HTTP 服务暴露 `archive/`，使用者用浏览器下载 | 常驻 |

## 目录结构

```
deployments/release/
├── release.conf              # 配置：版本号、发布渠道、压缩方式、Docker 版本、镜像清单
├── download-docker.sh        # ① Docker 引擎离线包下载（不常跑）
├── build-images.sh           # ② 基础设施包构建（不常跑）
├── build-release.sh          # ③ 项目包构建 / 发版（常跑）
├── gen-index.sh              # ④ 下载索引生成（②③ 自动调用，也可手动）
├── serve.sh                  # ⑤ HTTP 下载服务
├── bundle/                   # 进交付包的静态模板（docker/install-docker.sh + deploy/）
├── docker-cache/             # ① 的产物：docker-cache/<arch>/docker-<版本>.tgz（git 忽略）
├── images-cache/             # ② 的中间产物（基础镜像 tar，含 *-<arch>-saved 双 tag；git 忽略）
├── dist/                     # 构建临时工作区（git 忽略）
└── archive/                  # 版本化归档 = HTTP 服务根目录（git 忽略）
    ├── index.html            #   自动生成的下载索引（项目包、基础设施包两张表 + 操作步骤）
    ├── project/<项目版本>/
    │   ├── omc-<test|release>-<版本>-amd64.tar.xz (+ .sha256)
    │   ├── omc-<test|release>-<版本>-arm64.tar.xz (+ .sha256)
    │   └── RELEASE.txt
    └── infra/<基础设施版本>/
        ├── omc-infra-<版本>-amd64.tar.xz (+ .sha256)
        ├── omc-infra-<版本>-arm64.tar.xz (+ .sha256)
        └── RELEASE.txt
```

## 使用

> ⚠️ `build-images.sh` / `build-release.sh` **用普通用户运行，不要 `sudo`**
> （sudo 重置 PATH 会找不到 go/npm，产物归 root）。`docker` 权限用
> `sudo usermod -aG docker $USER && newgrp docker` 解决。

### ① 下载 Docker 引擎离线包（首次 / Docker 版本变更时）

```bash
cd deployments/release
./download-docker.sh               # 按 release.conf 的 DOCKER_VERSION 下载 amd64 + arm64
#   → docker-cache/{amd64,arm64}/docker-<版本>.tgz
#   --force   已存在也重下
```

### ② 构建基础设施包（首次 / 基础设施变更时）

```bash
./build-images.sh                  # v2 默认：基础设施 + 监控栈全套
#   -v 0.0.2            手动指定基础设施版本
#   --infra-only        仅基础设施（不要监控栈）
#   --monitoring-only   只补监控栈镜像（不重拉基础设施）
#   --with-monitoring   【已废弃】保留兼容；v2 默认即含监控栈，本标志为 no-op
#   → archive/infra/<版本>/omc-infra-<版本>-<架构>.tar.xz
```

### ③ 构建项目包（每次发版）

```bash
./build-release.sh                 # 小版本：版本自动生成；渠道取 release.conf 的 RELEASE_CHANNEL
./build-release.sh -v 1.0.0        # 大版本：手动指定版本
./build-release.sh --channel release   # 临时覆盖渠道（默认 test）
#   → archive/project/<版本>/omc-<test|release>-<版本>-<架构>.tar.xz
```

产物归档到 `archive/project/<版本>/`，并自动刷新 `archive/index.html`。

### ④ 起下载服务

```bash
./serve.sh                         # 默认端口 8000
```

使用者浏览器访问 `http://<构建机IP>:8000/` → 看到**操作步骤** + 项目包/基础设施包两张版本列表 → 点击下载。

## 版本号规则（基础设施 / 项目 各自独立，均纯 semver，从 0.0.1 起）

| 维度 | 取值 | 说明 |
|------|------|------|
| 基础设施版本 | `release.conf` 的 `INFRA_VERSION`，或 `build-images.sh -v` | 不常变，手动维护；改镜像/Docker 版本后递增 |
| 项目版本（小版本） | `build-release.sh` 不带 `-v` → `<RELEASE_BASE_VERSION>-<时间戳>` | 频繁发布，自动生成 |
| 项目版本（大版本） | `build-release.sh -v X.Y.Z` | 手动指定 |
| 发布渠道 | `release.conf` 的 `RELEASE_CHANNEL`，或 `build-release.sh --channel` | `test`=测试阶段 / `release`=正式发布，决定项目包文件名前缀 |

当前处于测试阶段：`INFRA_VERSION=0.0.1`、`RELEASE_BASE_VERSION=0.0.1`、`RELEASE_CHANNEL=test`。

## 注意

- **镜像版本固化**：`release.conf` 镜像标签发布前改为具体版本号，不要用 `latest`。
- **Docker 版本**：`release.conf` 的 `DOCKER_VERSION` / `DOCKER_URL_TEMPLATE` 控制下载。
- **压缩方式**：`release.conf` 的 `PKG_COMPRESS` 控制交付包压缩（默认 `xz`，体积最小）。
- **首次部署**需基础设施包 + 项目包**两个都下载**（同架构），先装 Docker、导基础镜像，再部署项目包。
- **镜像缓存策略**：`build-images.sh` 通过 `<image>:<tag>-<arch>-saved` 后缀 tag 实现跨架构镜像**命名隔离**；本机原 tag 始终指向本机架构镜像，**可与同主机的 `docker compose` 并存**，互不干扰。构建结束**不再 `docker rmi` 清理**：`*-saved` tag 作为本地缓存，下次构建命中即跳 `docker pull`。打出的 tar 同时包含 `*-saved` 与原 tag（**双 tag tar**），运维侧 `docker load` 后直接用原 tag 即可。详见《构建手册》§6.3。
- `docker-cache/`、`images-cache/`、`dist/`、`archive/` 为构建产物，已被 `.gitignore` 忽略。

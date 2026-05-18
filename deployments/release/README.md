# deployments/release — OMC 离线交付包

本目录用于**生成、归档、分发交付给运维的离线交付包**。配套文档：

- **快速上手**（怎么用本目录的脚本）：[`docs/operations/OMC交付构建快速上手.md`](../../docs/operations/OMC交付构建快速上手.md)
- 构建侧完整说明：[`docs/operations/OMC离线交付包构建手册（构建侧）.md`](../../docs/operations/OMC离线交付包构建手册（构建侧）.md)
- 运维侧（随交付包发给运维）：[`docs/operations/OMC内网离线部署手册（运维侧）.md`](../../docs/operations/OMC内网离线部署手册（运维侧）.md)

## 三个脚本

构建拆分为**基础设施**与**项目版本**两条线（构建周期不同），外加一个下载服务：

| 脚本 | 职责 | 运行频率 |
|------|------|---------|
| `build-images.sh` | 拉取并导出基础设施 Docker 镜像 → `images-cache/`（带独立的基础设施版本号） | **低**：仅镜像版本变更时 |
| `build-release.sh` | 编译二进制 + 前端 + 组装 + 压缩，按版本归档到 `archive/` | **高**：每次发版 |
| `serve.sh` | 起 HTTP 服务暴露 `archive/`，使用者用浏览器下载 | 常驻 |

## 目录结构

```
deployments/release/
├── release.conf              # 配置：基础设施版本、项目基线版本、压缩方式、镜像清单
├── build-images.sh           # ① 基础设施镜像构建（不常跑）
├── build-release.sh          # ② 交付包构建 / 发版（常跑）
├── serve.sh                  # ③ HTTP 下载服务
├── bundle/                   # 进交付包的静态模板（docker/ + deploy/）
├── images-cache/             # ① 的产物，② 复用（git 忽略）
├── dist/                     # 构建临时工作区（git 忽略）
└── archive/                  # 版本化归档 = HTTP 服务根目录（git 忽略）
    ├── index.html            #   自动生成的版本下载索引
    └── <项目版本>/
        ├── omc-release-<版本>-amd64.tar.xz (+ .sha256)
        ├── omc-release-<版本>-arm64.tar.xz (+ .sha256)
        └── RELEASE.txt       #   版本构建说明（项目/基础设施版本、镜像清单）
```

## 使用

> ⚠️ `build-images.sh` / `build-release.sh` **用普通用户运行，不要 `sudo`**
> （sudo 重置 PATH 会找不到 go/npm，产物归 root）。`docker` 权限用
> `sudo usermod -aG docker $USER && newgrp docker` 解决。

### ① 构建基础设施镜像（首次 / 镜像版本变更时）

```bash
cd deployments/release
# 一次性：把 Docker 静态二进制包放进 bundle/docker/（见 bundle/docker/README.md）
./build-images.sh                  # 基础设施版本取 release.conf 的 INFRA_VERSION
#   -v infra-1.1        手动指定基础设施版本
#   --with-monitoring   额外导出监控栈镜像
```

### ② 构建交付包（每次发版）

```bash
./build-release.sh                 # 小版本：版本自动生成
./build-release.sh -v 2.0.0        # 大版本：手动指定
```

产物归档到 `archive/<版本>/`，并自动刷新 `archive/index.html`。

### ③ 起下载服务

```bash
./serve.sh                         # 默认端口 8000
```

使用者浏览器访问 `http://<构建机IP>:8000/` → 看到版本列表 → 点击下载。

## 版本号规则（基础设施 / 项目 各自独立）

| 维度 | 取值 | 说明 |
|------|------|------|
| 基础设施版本 | `release.conf` 的 `INFRA_VERSION`，或 `build-images.sh -v` | 不常变，手动维护；改镜像清单后递增 |
| 项目版本（小版本） | `build-release.sh` 不带 `-v` → `<RELEASE_BASE_VERSION>-<时间戳>` | 频繁发布，自动生成 |
| 项目版本（大版本） | `build-release.sh -v X.Y.Z` | 手动指定 |

每个交付包内 `VERSION` / `RELEASE.txt` 同时记录**项目版本**与**基础设施版本**，
并说明本次基础设施是否更新。

## 注意

- **镜像版本固化**：`release.conf` 镜像标签发布前改为具体版本号，不要用 `latest`。
- **压缩方式**：`release.conf` 的 `PKG_COMPRESS` 控制交付包压缩（默认 `xz`，体积最小）。
- `images-cache/`、`dist/`、`archive/` 为构建产物，已被 `.gitignore` 忽略。

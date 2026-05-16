# deployments/release — OMC 离线交付包

本目录用于**生成、存放交付给运维的离线交付包**。配套文档：

- 构建侧（本目录工具的完整说明）：[`docs/operations/OMC离线交付包构建手册（构建侧）.md`](../../docs/operations/OMC离线交付包构建手册（构建侧）.md)
- 运维侧（随交付包发给运维）：[`docs/operations/OMC内网离线部署手册（运维侧）.md`](../../docs/operations/OMC内网离线部署手册（运维侧）.md)

## 两个独立的构建工具

构建拆成两步，因为**基础设施镜像不常变、二进制要频繁发版**：

| 工具 | 职责 | 运行频率 |
|------|------|---------|
| `build-images.sh` | 拉取并导出基础设施 Docker 镜像 → `images-cache/` | **低**：仅在 `release.conf` 调整镜像版本时 |
| `build-release.sh` | 编译二进制 + 构建前端 + 组装交付包（复用 `images-cache/`） | **高**：每次发版 |

## 目录结构

```
deployments/release/
├── README.md                 # 本文件
├── release.conf              # 配置：基线版本号、基础设施镜像清单
├── build-images.sh           # ① 基础设施镜像构建（不常跑）
├── build-release.sh          # ② 交付包构建 / 发版（常跑）
├── bundle/                   # 进交付包的静态模板（docker/ + deploy/）
├── images-cache/             # build-images.sh 的产物，build-release.sh 复用（git 忽略）
└── dist/                     # 交付包产出目录（git 忽略）
```

## 使用

> ⚠️ **用普通用户运行，不要 `sudo`**。脚本是构建脚本、不需要 root；`sudo` 会重置
> PATH 导致找不到 `go`/`npm`，还会让产物归 root。`docker` 权限请用
> `sudo usermod -aG docker $USER && newgrp docker` 把当前用户加入 docker 组解决。

### 第一步：构建基础设施镜像（首次，或镜像版本变更时）

```bash
cd deployments/release
# 一次性：把 Docker 静态二进制包放进 bundle/docker/（见 bundle/docker/README.md）
./build-images.sh                       # 产出 images-cache/infra-images-{amd64,arm64}.tar
#   --arch amd64        只构建指定架构
#   --with-monitoring   额外导出监控栈镜像
```

### 第二步：构建交付包（每次发版）

```bash
cd deployments/release

# 小版本（日常）：不带 -v，版本自动生成 = <基线版本>-<构建时间戳>
./build-release.sh

# 大版本：用 -v 手动指定
./build-release.sh -v 2.0.0

#   --arch amd64        只构建指定架构（缺省 amd64 + arm64）
```

产出：`dist/omc-release-<版本>-{amd64,arm64}.tar.gz` + `.sha256`。

> `build-release.sh` 复用 `images-cache/` 里的镜像；若缓存不存在会提示先跑
> `build-images.sh`。基础设施镜像不常变，无需每次发版重拉。

## 版本号规则

- **小版本（频繁）**：`build-release.sh` 不带 `-v`，自动生成
  `<release.conf 的 RELEASE_BASE_VERSION>-<YYYYMMDD-HHMM>`，每次构建唯一可追溯。
- **大版本**：`build-release.sh -v X.Y.Z` 手动指定；发大版本时建议同步把
  `release.conf` 的 `RELEASE_BASE_VERSION` 更新为该版本，使之后的小版本基线对齐。

## 注意

- **镜像版本固化**：`release.conf` 的镜像标签发布前改为具体版本号，不要用
  `latest` / `latest-pg16`（详见《构建手册》§7）。
- `images-cache/` 与 `dist/` 是构建产物（含大体积 tar），已被 `.gitignore` 忽略。

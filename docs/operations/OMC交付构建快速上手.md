# OMC 交付构建快速上手

> **适用对象**：在构建机上制作、分发 OMC 离线交付包的研发 / 发布工程师。
> **本文定位**：命令驱动的操作速查。完整原理、交付包结构、环境要求见
> 《OMC离线交付包构建手册（构建侧）》；运维部署见《OMC内网离线部署手册（运维侧）》。
> **工具位置**：`deployments/release/`。

---

## 1. 四个脚本一览

| 脚本 | 干什么 | 多久跑一次 | 依赖 |
|------|--------|-----------|------|
| `download-docker.sh` | 下载 Docker 引擎离线包 → `docker-cache/` | **低**：仅 Docker 版本变更时 | curl/wget + 公网 |
| `build-images.sh` | 拉取并导出基础设施 Docker 镜像 → `images-cache/` | **低**：仅镜像版本变更时 | docker + 公网 |
| `build-release.sh` | 编译二进制 + 前端 + 组装压缩 → 归档到 `archive/` | **高**：每次发版 | go + npm + 公网 |
| `serve.sh` | 起 HTTP 服务，让使用者用浏览器下载交付包 | 常驻 | python3 |

> 一句话：**`download-docker.sh`/`build-images.sh` 偶尔跑、`build-release.sh` 每次发版跑、`serve.sh` 一直开着。**

> ⚠️ 这些脚本都**用普通用户运行，不要 `sudo`**。`sudo` 会重置 `PATH`，导致脚本
> 找不到 `go`/`npm`，并让产物归 `root`。`docker` 权限用 docker 组解决（见 §2）。

---

## 2. 首次准备（一次性）

```bash
cd <仓库>/deployments/release

# (1) 把当前用户加入 docker 组（build-images.sh 要用 docker）
sudo usermod -aG docker $USER && newgrp docker

# (2) 按需编辑 release.conf：
#     DOCKER_VERSION        Docker 引擎版本（download-docker.sh 用）
#     DOCKER_URL_TEMPLATE   Docker 下载地址模板
#     INFRA_VERSION         基础设施版本号（如 infra-1.0）
#     RELEASE_BASE_VERSION  项目基线版本号（如 1.0.0）
#     IMAGE_*               基础设施镜像 —— 发布前把 latest 改成具体版本号
#     PKG_COMPRESS          交付包压缩方式（默认 xz，体积最小）

# (3) 下载 Docker 引擎离线包（供运维侧离线装 Docker；目标机已装可跳过）
./download-docker.sh        # 按 release.conf 下载 amd64 + arm64 → docker-cache/
```

---

## 3. 场景化用法

### 场景 A —— 日常发小版本（最常用）

镜像没变，只是出一个新的项目版本：

```bash
./build-release.sh
```

版本号自动生成 = `<RELEASE_BASE_VERSION>-<构建时间戳>`，如 `1.0.0-20260518-1030`。
产物归档到 `archive/1.0.0-20260518-1030/`。

### 场景 B —— 发大版本

```bash
./build-release.sh -v 2.0.0
```

版本号即 `2.0.0`。建议同时把 `release.conf` 的 `RELEASE_BASE_VERSION` 改成 `2.0.0`，
让之后的小版本基线对齐。

### 场景 C —— 升级了基础设施镜像

改动镜像版本时，先重建镜像缓存，再发版：

```bash
# 1) 编辑 release.conf：递增 INFRA_VERSION（如 infra-1.0 → infra-1.1），更新 IMAGE_*
# 2) 重建基础设施镜像缓存
./build-images.sh
# 3) 正常发版
./build-release.sh                  # 或 -v <大版本>
```

`build-release.sh` 会在 `RELEASE.txt` 标注"基础设施：已更新（infra-1.0 → infra-1.1）"。

### 场景 D —— 只出单架构

```bash
./build-images.sh  --arch amd64
./build-release.sh --arch amd64
```

缺省同时产出 amd64 + arm64。

### 场景 E —— 带监控栈

```bash
./build-images.sh --with-monitoring     # 额外导出 prometheus/grafana/loki 等镜像
```

---

## 4. 分发：起 HTTP 下载服务

构建产物归档在 `archive/` 后，起服务供使用者下载：

```bash
./serve.sh                # 默认端口 8000
./serve.sh 9000           # 指定端口

# 后台常驻：
nohup ./serve.sh 8000 >/tmp/omc-serve.log 2>&1 &
```

使用者（运维）浏览器访问 `http://<构建机IP>:8000/` → `archive/index.html` 列出
所有版本（项目版本 / 基础设施版本 / 构建时间 / 下载链接）→ 点击下载对应架构的包。
也可用 `wget`/`curl` 下载。

---

## 5. 版本号怎么来的

| 维度 | 取值 | 谁定 |
|------|------|------|
| 基础设施版本 | `release.conf` 的 `INFRA_VERSION`，或 `build-images.sh -v` | 手动，改镜像后递增 |
| 项目小版本 | `build-release.sh` 不带 `-v` → `<基线>-<时间戳>` | 自动 |
| 项目大版本 | `build-release.sh -v X.Y.Z` | 手动 |

基础设施版本与项目版本**各自独立**；每个交付包内 `VERSION` / `RELEASE.txt` 同时
记录两者，`RELEASE.txt` 还说明本次基础设施相对上次发布**是否更新**。

---

## 6. 产物在哪

```
deployments/release/
├── docker-cache/                       # download-docker.sh 产物（Docker 引擎包，可复用）
├── images-cache/                       # build-images.sh 产物（基础设施镜像，可复用）
└── archive/                            # build-release.sh 产物（交付包归档）
    ├── index.html                      #   HTTP 下载首页（自动生成）
    └── <项目版本>/
        ├── omc-release-<版本>-amd64.tar.xz  + .sha256
        ├── omc-release-<版本>-arm64.tar.xz  + .sha256
        └── RELEASE.txt                 #   版本构建说明
```

交给运维：让运维从 HTTP 服务下载，或把 `archive/<版本>/` 拷到 U 盘/介质送入内网。
运维侧照交付包内 `docs/OMC内网离线部署手册（运维侧）.md` 部署。

---

## 7. 完整流程示例

```bash
cd deployments/release

# —— 首次：准备 + 下载 Docker + 建镜像 ——
sudo usermod -aG docker $USER && newgrp docker
# （编辑 release.conf）
./download-docker.sh                     # 下载 Docker 引擎离线包
./build-images.sh                        # 拉取基础设施镜像

# —— 日常：发版 ——
./build-release.sh                       # 小版本
#   → archive/1.0.0-20260518-1030/

# —— 分发 ——
./serve.sh &                             # 起服务，保持开启

# —— 某次升级了基础设施 / Docker ——
# 改 release.conf 后：
./download-docker.sh                     # 若改了 DOCKER_VERSION
./build-images.sh                        # 若改了镜像版本
./build-release.sh -v 2.0.0
```

---

## 8. 常见问题

| 现象 | 原因 / 处理 |
|------|------------|
| `缺少构建工具：go`（但 `go version` 能跑） | 用了 `sudo`，PATH 被重置。用**普通用户**直接运行，不要 sudo |
| `docker 不可用：当前用户可能不在 docker 组` | `sudo usermod -aG docker $USER && newgrp docker` 后重试 |
| `build-release.sh` 报"缺少基础设施镜像缓存" | 先跑一次 `./build-images.sh` 生成 `images-cache/` |
| 交付包很大 | `release.conf` 设 `PKG_COMPRESS=xz`（默认，体积最小）；gzip 更快但略大 |
| 浏览器打不开下载页 | `serve.sh` 是否在跑；构建机防火墙是否放行该端口 |
| 不同批次镜像不一致 | `release.conf` 的镜像标签别用 `latest`，固化为具体版本号 |

---

**速记**：`download-docker.sh` + `build-images.sh`（偶尔）→ `build-release.sh`（每次发版）→ `serve.sh`（常开）。
深入内容见《OMC离线交付包构建手册（构建侧）》。

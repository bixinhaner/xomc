# OMC 交付构建快速上手

> **适用对象**：在构建机上制作、分发 OMC 离线交付包的研发 / 发布工程师。
> **本文定位**：命令驱动的操作速查。完整原理、交付包结构、环境要求见
> 《OMC离线交付包构建手册（构建侧）》；运维部署见《OMC内网离线部署手册（运维侧）》。
> **工具位置**：`deployments/release/`。

---

## 0. 两个独立交付包

交付物拆成**两个互相独立、各自维护版本号**的包：

| 交付包 | 内容 | 由谁生成 | 文件名 |
|--------|------|---------|--------|
| **基础设施包** | Docker 引擎离线包 + 基础镜像（PG/Redis/NATS/MinIO/Nginx） | `build-images.sh` | `omc-infra-<版本>-<架构>.tar.xz` |
| **项目包** | OMC 二进制 + 前端 + 配置 + 数据库迁移 + 部署模板 | `build-release.sh` | `omc-<test\|release>-<版本>-<架构>.tar.xz` |

基础设施不常变、发布频率低；项目包每次发版都出。运维首次部署两个都下，之后日常升级通常只更新项目包。

---

## 1. 五个脚本一览

| 脚本 | 干什么 | 多久跑一次 | 依赖 |
|------|--------|-----------|------|
| `download-docker.sh` | 下载 Docker 引擎离线包 → `docker-cache/` | **低**：仅 Docker 版本变更时 | curl/wget + 公网 |
| `build-images.sh` | 拉取基础设施镜像 + 组装压缩**基础设施包** → `archive/infra/` | **低**：仅基础设施变更时 | docker + 公网 |
| `build-release.sh` | 编译二进制 + 前端 + 组装压缩**项目包** → `archive/project/` | **高**：每次发版 | go + npm + 公网 |
| `gen-index.sh` | 扫描 `archive/` 生成下载索引 `index.html` | 自动（②③ 调用） | bash |
| `serve.sh` | 起 HTTP 服务，让使用者用浏览器下载交付包 | 常驻 | python3 |

> 一句话：**`download-docker.sh`/`build-images.sh` 偶尔跑、`build-release.sh` 每次发版跑、`serve.sh` 一直开着；`gen-index.sh` 自动调用。**

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
#     INFRA_VERSION         基础设施版本号（纯 semver，当前 0.0.1）
#     RELEASE_BASE_VERSION  项目基线版本号（纯 semver，当前 0.0.1）
#     RELEASE_CHANNEL       发布渠道：test=测试阶段 / release=正式发布
#     IMAGE_*               基础设施镜像 —— 发布前把 latest 改成具体版本号
#     PKG_COMPRESS          交付包压缩方式（默认 xz，体积最小）

# (3) 下载 Docker 引擎离线包（会打进基础设施包，供运维侧离线装 Docker）
./download-docker.sh        # 按 release.conf 下载 amd64 + arm64 → docker-cache/
```

---

## 3. 场景化用法

### 场景 A —— 日常发小版本（最常用）

基础设施没变，只出一个新的项目版本：

```bash
./build-release.sh
```

版本号自动生成 = `<RELEASE_BASE_VERSION>-<构建时间戳>`，渠道取 `release.conf` 的
`RELEASE_CHANNEL`（默认 `test`）。产出如 `omc-test-0.0.1-20260518-1030-amd64.tar.xz`，
归档到 `archive/project/0.0.1-20260518-1030/`。

### 场景 B —— 发大版本 / 切正式渠道

```bash
./build-release.sh -v 1.0.0                  # 大版本号
./build-release.sh --channel release         # 切到正式渠道 → omc-release-...
./build-release.sh -v 1.0.0 --channel release # 两者一起
```

发大版本时建议同步把 `release.conf` 的 `RELEASE_BASE_VERSION` 改成 `1.0.0`，让之后的小版本基线对齐。

### 场景 C —— 升级基础设施（镜像 / Docker）

基础设施包与项目包独立，升级基础设施只需重出基础设施包：

```bash
# 1) 编辑 release.conf：递增 INFRA_VERSION（如 0.0.1 → 0.0.2），更新 IMAGE_* / DOCKER_VERSION
# 2) 若改了 DOCKER_VERSION，重新下载 Docker 引擎包
./download-docker.sh
# 3) 重建基础设施包
./build-images.sh                  # → archive/infra/0.0.2/omc-infra-0.0.2-<架构>.tar.xz
```

项目包不受影响，无需重出。

> **镜像缓存复用**：`build-images.sh` 通过 `<image>:<tag>-<arch>-saved` 后缀 tag
> 把跨架构镜像与本机原 tag **命名隔离**，构建结束后**不再 `docker rmi` 清理**——
> `*-saved` tag 留作本地缓存，下次构建命中即跳过 `docker pull`；本机原 tag 始终
> 指向本机架构镜像，可与同主机的 `docker compose` 并存运行，互不干扰。详见构建
> 手册 §6.3。

### 场景 D —— 只出单架构

```bash
./build-images.sh  --arch amd64
./build-release.sh --arch amd64
```

缺省同时产出 amd64 + arm64。

### 场景 E —— 带监控栈

```bash
./build-images.sh --with-monitoring     # 基础设施 + 监控栈一起导出并打进基础设施包
# 若已建好基础设施、之后才想补监控（不重拉基础设施）：
./build-images.sh --monitoring-only     # 只补 monitoring-images-*.tar 并重新打包
```

---

## 4. 分发：起 HTTP 下载服务

构建产物归档到 `archive/` 后，起服务供使用者下载：

```bash
./serve.sh                # 默认端口 8000
./serve.sh 9000           # 指定端口

# 后台常驻：
nohup ./serve.sh 8000 >/tmp/omc-serve.log 2>&1 &
```

使用者（运维）浏览器访问 `http://<构建机IP>:8000/` → 下载页含 **7 步操作指引** +
**项目交付包**、**基础设施包** 两张版本列表 → 按目标机架构点击下载。也可用 `wget`/`curl`。

---

## 5. 版本号与渠道

| 维度 | 取值 | 谁定 |
|------|------|------|
| 基础设施版本 | `release.conf` 的 `INFRA_VERSION`，或 `build-images.sh -v` | 手动，改基础设施后递增 |
| 项目小版本 | `build-release.sh` 不带 `-v` → `<基线>-<时间戳>` | 自动 |
| 项目大版本 | `build-release.sh -v X.Y.Z` | 手动 |
| 发布渠道 | `release.conf` 的 `RELEASE_CHANNEL`，或 `build-release.sh --channel` | `test` / `release` |

版本号均为纯 semver，**从 0.0.1 起步**（当前测试阶段，不从 1.0.0 起）。基础设施版本与
项目版本**各自独立**，互不耦合；各包内 `VERSION` / `RELEASE.txt` 记录自己的版本与构建信息。

---

## 6. 产物在哪

```
deployments/release/
├── docker-cache/                          # download-docker.sh 产物（Docker 引擎包，可复用）
├── images-cache/                          # build-images.sh 中间产物（基础镜像 tar）
└── archive/                               # 交付包归档 = HTTP 服务根目录
    ├── index.html                         #   HTTP 下载首页（自动生成）
    ├── project/<项目版本>/
    │   ├── omc-<test|release>-<版本>-amd64.tar.xz  + .sha256
    │   ├── omc-<test|release>-<版本>-arm64.tar.xz  + .sha256
    │   └── RELEASE.txt
    └── infra/<基础设施版本>/
        ├── omc-infra-<版本>-amd64.tar.xz   + .sha256
        ├── omc-infra-<版本>-arm64.tar.xz   + .sha256
        └── RELEASE.txt
```

交给运维：让运维从 HTTP 服务下载（首次部署两个包都下），或把 `archive/` 对应目录拷到
U 盘/介质送入内网。运维侧照项目包内 `docs/OMC内网离线部署手册（运维侧）.md` 部署。

---

## 7. 完整流程示例

```bash
cd deployments/release

# —— 首次：准备 + 下载 Docker + 建基础设施包 ——
sudo usermod -aG docker $USER && newgrp docker
# （编辑 release.conf）
./download-docker.sh                     # 下载 Docker 引擎离线包
./build-images.sh                        # → archive/infra/0.0.1/

# —— 日常：发项目版本 ——
./build-release.sh                       # 小版本（test 渠道）
#   → archive/project/0.0.1-20260518-1030/

# —— 分发 ——
./serve.sh &                             # 起服务，保持开启

# —— 某次升级了基础设施 / Docker ——
# 改 release.conf（递增 INFRA_VERSION）后：
./download-docker.sh                     # 若改了 DOCKER_VERSION
./build-images.sh                        # 重出基础设施包，项目包不受影响
```

---

## 8. 常见问题

| 现象 | 原因 / 处理 |
|------|------------|
| `缺少构建工具：go`（但 `go version` 能跑） | 用了 `sudo`，PATH 被重置。用**普通用户**直接运行，不要 sudo |
| `docker 不可用：当前用户可能不在 docker 组` | `sudo usermod -aG docker $USER && newgrp docker` 后重试 |
| `--channel 取值非法` | `--channel` 只接受 `test` 或 `release` |
| `build-images.sh` 警告"docker-cache 无 Docker 引擎包" | 先跑一次 `./download-docker.sh` |
| 交付包很大 | `release.conf` 设 `PKG_COMPRESS=xz`（默认，体积最小）；gzip 更快但略大 |
| 浏览器打不开下载页 | `serve.sh` 是否在跑；构建机防火墙是否放行该端口 |
| 不同批次镜像不一致 | `release.conf` 的镜像标签别用 `latest`，固化为具体版本号 |
| 本地 `*-saved` 镜像太多想清理 | 这些是 `build-images.sh` 的本地缓存（命名隔离、不影响 docker compose），可保留以加速下次构建；如需手工回收：`docker images \| grep -- '-saved' \| awk '{print $1":"$2}' \| xargs docker rmi -f`，或运行 `docker image prune` |

---

**速记**：`download-docker.sh` + `build-images.sh`（偶尔，出基础设施包）→ `build-release.sh`（每次发版，出项目包）→ `serve.sh`（常开）。
深入内容见《OMC离线交付包构建手册（构建侧）》。

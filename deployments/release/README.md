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

HTTPS 8443 兼容说明：项目包会包含 `deploy/nginx-cert/cert.pem`、
`deploy/nginx-cert/key.pem` 和 `deploy/openssl-legacy.cnf`。当前证书是老 OMC
1024-bit 证书，web 容器通过
`OPENSSL_CONF=/etc/nginx/openssl-legacy.cnf` 降低自身 OpenSSL 安全级别以兼容
该证书。这个例外只作用于 web/nginx 容器，不改基础镜像、不改 Dockerfile
`FROM`、不改宿主机全局 OpenSSL，也不影响 app/acs/worker/db。未来换成
2048-bit 或更强证书后，应移除这个例外配置。

### ④ 起下载服务

```bash
./serve.sh                         # 默认端口 8000，前台运行（Ctrl-C 停止）
```

使用者浏览器访问 `http://<构建机IP>:8000/` → 看到**操作步骤** + 项目包/基础设施包两张版本列表 → 点击下载。

需要长期运行（关 ssh 不掉、机器重启自启）见下一节「下载服务长期运行」。

## 下载服务长期运行 — `./serve.sh` 自管 vs systemd

`serve.sh` 提供两种常驻方式，互不干扰，**只选其一**：

| 方式 | 适用场景 | 关 ssh / 退出登录后 | 机器重启后 | 谁监管崩溃 |
|------|---------|----------|-----------|----------|
| **A. `./serve.sh start`** | 临时常驻（含 `build-release.sh` 末尾自动起的下载服务） | 不掉 | **不自启** | 无 |
| **B. systemd unit** | 长期挂着、要开机自启、要崩溃重启 | 不掉 | 自启 | systemd `Restart=on-failure` |

> ⚠️ **两种方式不要并存**。systemd 接管后再用 `./serve.sh start` 会出现端口冲突。要切换先把另一种停干净。

### 方式 A：`./serve.sh` 自管（脚本内置 daemon）

```bash
./serve.sh start                   # 启动；默认 8000；关 ssh / 退出登录不带走
./serve.sh start -p 9000           # 启动并指定端口
./serve.sh status                  # 查看运行状态（unit/PID / 端口 / 日志路径）
./serve.sh stop                    # 停止
./serve.sh restart                 # 重启；不传 -p 时沿用上次端口
./serve.sh restart -p 9001         # 重启并换端口
```

**后台保活怎么实现的（start / restart 自动选择，无需关心）**：

| 运行环境 | 后台托管方式 | 关 ssh / 退出登录 | 日志 |
|---------|------------|------------------|------|
| **root + systemd**（构建机常见） | **systemd 瞬态服务 `omc-serve-auto`**（脱离登录会话，挺过 `logind` 的 `KillUserProcesses=yes`） | 不掉 | `journalctl -u omc-serve-auto -f` |
| 非 root / 无 systemd | `setsid`+`nohup`（写 `.serve.pid`/`.serve.log`） | 一般不掉；但开了 `KillUserProcesses=yes` 的 systemd 系统登出仍可能被带走 → 用方式 B | `tail -f .serve.log` |

> 历史坑：`build-release.sh` 末尾会自动 `serve.sh restart` 刷新下载服务。早期只用
> `setsid`+`nohup`，在 **root + systemd** 机器上 `KillUserProcesses=yes` 会在登出时
> 连带杀掉它，表现为"构建完关 ssh 后 8000 端口就访问不了"。现已改为 root+systemd 下
> 走 systemd 瞬态服务托管，登出 / 关 ssh 不再带走。

相关文件（均在 `deployments/release/` 下，已加 `.gitignore`）：

| 文件 | 用途 |
|------|------|
| `.serve.unit` | systemd 瞬态服务模式标记（内容=unit 名；存在即走 systemd） |
| `.serve.pid`  | `setsid`/`nohup` 模式的后台进程 PID |
| `.serve.port` | 上次启动的端口（`restart` 沿用） |
| `.serve.log`  | `setsid`/`nohup` 模式的请求日志（systemd 模式日志在 journald） |

**适合谁**：临时把构建机变成下载源，开发 / 测试场景。**不适合**机器重启后自动起来——
瞬态服务和 `setsid`/`nohup` 都不写开机自启，重启就没了；要**重启后自启**用方式 B。

### 方式 B：systemd unit（生产推荐）

unit 文件随仓库交付：[`deployments/release/omc-serve.service`](omc-serve.service)。**这是构建侧文件，不打进 `omc-infra-*.tar.xz`**——客户那边不需要起这个服务。

```bash
# 1) 假设 deployments/release/ 拷贝到了 /opt/omc/release/。如果不是，
#    先编辑 omc-serve.service 把 WorkingDirectory / ExecStart 中两处
#    /opt/omc/release 改成你的实际路径，再继续。
ls /opt/omc/release/serve.sh /opt/omc/release/omc-serve.service

# 2) 装到 systemd
sudo cp /opt/omc/release/omc-serve.service /etc/systemd/system/omc-serve.service
sudo systemctl daemon-reload

# 3) 启动 + 开机自启
sudo systemctl enable --now omc-serve

# 4) 验证
sudo systemctl status omc-serve
sudo ss -ltnp | grep :8000
sudo journalctl -u omc-serve -f       # 实时看请求日志
```

**改端口或路径**（升级安全的做法 — 用 drop-in，不动 `/etc/systemd/system/omc-serve.service`）：

```bash
sudo systemctl edit omc-serve
```

编辑器里写覆盖段，例如换 9000 端口：

```ini
[Service]
ExecStart=
ExecStart=/opt/omc/release/serve.sh -p 9000
```

> 第一行 `ExecStart=` 留空很关键：systemd 不允许累加 `ExecStart`，必须先清零再设新值。

存盘后：

```bash
sudo systemctl daemon-reload
sudo systemctl restart omc-serve
```

**常用维护命令**：

```bash
sudo systemctl start    omc-serve
sudo systemctl stop     omc-serve
sudo systemctl restart  omc-serve
sudo systemctl status   omc-serve
sudo systemctl disable  omc-serve     # 取消开机自启（不卸载 unit）
sudo journalctl -u omc-serve --since "1 hour ago"
```

**unit 关键配置**（要查/改时找这里）：

| 字段 | 当前值 | 说明 |
|------|--------|------|
| `Type` | `simple` | systemd 直接监管前台 python http server，**不走** serve.sh 自带的 `start/stop` 守护逻辑 |
| `WorkingDirectory` | `/opt/omc/release` | serve.sh 所在目录 = archive/ 所在目录 |
| `ExecStart` | `… serve.sh -p 8000` | serve.sh 以前台模式运行；端口在这里改 |
| `Restart` | `on-failure` | 崩溃自动重启；正常退出不重启 |
| `RestartSec` | `3s` | 重启间隔 |
| `KillSignal` / `TimeoutStopSec` | `SIGTERM` / `10s` | 给 python 10 秒优雅退出，超时再 SIGKILL |
| `NoNewPrivileges` / `PrivateTmp` / `ProtectSystem=full` / `ProtectHome` | 安全收紧 | 限制服务权限，`/opt` 不在 `ProtectSystem=full` 范围，`archive/` 与日志仍可读写 |

**防火墙**（启动了但浏览器访问不通时）：

```bash
sudo ufw allow 8000/tcp                              # Ubuntu/Debian
sudo firewall-cmd --add-port=8000/tcp --permanent    # CentOS/RHEL
sudo firewall-cmd --reload
```

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
- **安装严格离线**：项目包安装会在启动前校验本次 Compose 使用的全部基础设施、监控和业务镜像，并以 `--pull never` 启动；缺少 `cadvisor` 等任一镜像会直接失败，不会联网补拉。首次部署或基础设施包尚未导入时不要使用 `--skip-infra`，应先解压并导入包含 `monitoring-images-*.tar` 的基础设施包。
- **镜像缓存策略**：`build-images.sh` 通过 `<image>:<tag>-<arch>-saved` 后缀 tag 实现跨架构镜像**命名隔离**；本机原 tag 始终指向本机架构镜像，**可与同主机的 `docker compose` 并存**，互不干扰。构建结束**不再 `docker rmi` 清理**：`*-saved` tag 作为本地缓存，下次构建命中即跳 `docker pull`。打出的 tar 同时包含 `*-saved` 与原 tag（**双 tag tar**），运维侧 `docker load` 后直接用原 tag 即可。详见《构建手册》§6.3。
- `docker-cache/`、`images-cache/`、`dist/`、`archive/` 为构建产物，已被 `.gitignore` 忽略。

# deployments/release 方案增强 — 交付侧 Docker 安装 / 系统加速 / 一键部署 / 脚本可发现性 / 操作手册

> 状态：**已落地（v2 收口）**
> 提出时间：2026-05-20
> 涉及代码：`deployments/release/**` + `docs/operations/**`

---

## v5 收口（2026-05-20 当日演进 — 跨架构死代码清理 + 历史归档处理）

v3 把 ARCHES 收紧到 amd64 并加入入口校验，但 `build-images.sh` 内部仍保留了
跨架构 pull 的代码路径（HOST_ARCH != ARCH 分支、`save_with_dual_tags` 末尾的
"修复原 tag" 循环、镜像缓存策略注释里"跨架构互不覆盖"等措辞）。这些都成为死
代码 / 误导性文档。本轮清理：

- **build-images.sh 简化**：
  - `prepare_image` 删除 `if [ "$ARCH" != "$HOST_ARCH" ]; then docker pull --platform "linux/$HOST_ARCH" ...` 分支
  - `save_with_dual_tags` 删除末尾跨架构 fixup 循环
  - 头部 "镜像缓存策略" 段重写为 amd64-only 语义（保留 -amd64-saved 后缀命名作历史缓存兼容）
  - 增入 `HOST_ARCH=amd64` 启动断言（非 amd64 host 上跑直接 die 并给指引）
  - `-h` 用法示例去掉 `--arch amd64`（amd64 是唯一允许值，例子里写它徒增疑问）
- **构建侧手册 §6.1 / §6.3 改写**：双架构 → amd64 单架构；伪代码同步删跨架构 fixup
- **gen-index.sh 下载页过滤**：
  - 扫描 glob 从 `omc-*.tar.*` 收紧到 `omc-*-amd64.tar.*` —— 历史 `*-arm64.tar.*`
    物理文件还在磁盘但不再进下载列表，避免给运维造成"两个文件是不是同一个"的迷惑
  - 底部加智能提示：检测到 N 个非 amd64 历史归档时显示一行 `find archive/ -name 'omc-*-arm64.tar.*' -delete` 清理命令

用户场景：今日 baicells 服务器跑 build-release.sh 后，archive/project/0.0.1-XXX/
里既有 -amd64.tar.xz（amd64-only refactor 之后产）也有 -arm64.tar.xz（refactor 之
前产），下载页同时列出两个，被误以为"重复文件"。glob 收紧后只列 amd64，问题解决。

---

## v4 收口（2026-05-20 当日演进 — install-docker.sh 数据目录智能检测）

baicells 构建机 `/var` 仅 4.9G LV，跑 `build-images.sh` 拉镜像直接撑爆
`/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/`。
针对"紧凑 `/var`"这类常见 Linux server 配置，给 `install-docker.sh` 加自动检测：

- **触发**：解压二进制前 `df -k /var` 取可用 KB → 换算 GB → < 15G 触发提示
- **提示内容**：报告 /var 实际可用 + /home 实际可用 + 推荐路径
  - docker → `/home/docker-data`
  - containerd → `/home/containerd-data`
- **默认行为**：回车（默认 Y）切到 /home；输入 n 保留 /var/lib；非 TTY 自动按 Y
- **/home 也紧张时**：仍切，但额外打一条"两个分区都紧张"warn
- **落地**：
  - containerd：`containerd.service` 的 `ExecStart` 拼 `--root /home/containerd-data`
  - docker：python3 merge `/etc/docker/daemon.json`，`data-root: /home/docker-data`
    （与 setup-mirrors.sh 写 `registry-mirrors` 同款 merge，两键共存不冲突）
- **不加 CLI flag**：用户决定后由脚本内置路径执行，简化 UX；CI / 自动化场景靠
  非 TTY → 自动 Y 兜底

阈值 15G 由"基础栈镜像 ~3G + 监控栈 ~5G + 中间层 + 容器运行时余量"推算而来。
未来若要做更激进的容量预估，可在 `build-images.sh` / `deploy.sh` 起始同样加
`df` 预检（目前 `build-images.sh` 仍只把 containerd 原始错误冒泡出来）。

---

## v3 收口（2026-05-20 当日演进 — 仅 amd64）

`release.conf` 的 `ARCHES` 从 `"amd64 arm64"` 收紧为 `"amd64"`，三个构建脚本
（`download-docker.sh` / `build-images.sh` / `build-release.sh`）在参数解析阶
段加 amd64-only 校验，传入非 amd64 直接 die。理由：

1. 现场交付目标设备全部为 amd64 服务器，arm64 没有真实用户
2. arm64 跨架构 pull 在 amd64 host 上要拉 arm64 manifest + 镜像，磁盘翻倍；
   baicells 构建机的 `/var` 切了 4.9G LV，跑双架构全栈直接撑爆 containerd
   snapshot 目录（实测）
3. 运维侧从未对 arm64 交付包做过 deploy + healthcheck 验证

未来重启 arm64 步骤已记录在 `release.conf` 注释（改 ARCHES + 移除三脚本校验
+ 40GB 磁盘 + arm64 设备实测 deploy）。下面 v2 收口段及 §0-§6 提案原文保
留作历史快照。

---

## v2 收口（2026-05-20 当日演进）

本设计的"镜像加速"模块在落地当天进一步收口为**三合一加速**：

- 脚本由 `setup-docker-mirror.sh` 重命名为 **`setup-mirrors.sh`**
- 由 Docker 单一目标扩为 **Docker / npm / Golang** 三目标
- 参数模型从 `--mirror <name>` 改为 `--docker <name>` / `--npm <name>` / `--golang <name>` 三段独立；
  `--mirror` 保留为已废弃别名（等价 `--docker`）
- 三目标各自的"不设置"选项均为 `official`；任意目标可独立跳过
- 配置落地位置：Docker → `/etc/docker/daemon.json`；npm → `/etc/npmrc`；Golang → `/etc/profile.d/goproxy.sh`
- **脚本位置**：从 `bundle/docker/setup-mirrors.sh` 上移到 `bundle/setup-mirrors.sh`（infra 包顶层），
  运行时落到 `/opt/omc/infra/setup-mirrors.sh`。这一调整反映"系统加速 ≠ Docker 子组件"的语义：
  `setup-mirrors.sh` 是基础设施包级别的通用工具，与 `docker/` / `images/` 平级。
  `install-docker.sh` 内部的调用相应改为 `bash "$(dirname "$0")/../setup-mirrors.sh"`。

下面 §0–§6 保留 2026-05-20 提案原文（已把脚本名同步成 `setup-mirrors.sh`），
但语义以 v2 收口为准——npm / Golang 两项是在此设计**落地当日补齐**的。

---

## 0. 用户原始诉求（5 项）

1. Docker **下载后如何安装** — 交付侧拿到 docker tgz 后，怎么装 docker 服务、开机自启 ...
2. Docker 安装后**怎么设置加速镜像** — 让交付侧从 官方 / 阿里云 / 腾讯云 / 其它稳定第三方 中**选择**；安装时引导，**内置几个稳定地址**
3. 怎么**把镜像导入 docker**、把**整个项目部署起来**（基础设施 + 版本镜像）
4. **每个脚本加 `-h`**，看清能传哪些参数、各做什么
5. 实施后完善**构建侧 + 交付侧**文档；并在 `:8000` 下载页加上交付侧**操作手册**，含：选哪些文件、下载清单、执行步骤、验证方式、部署后访问地址、初始账号密码

---

## 1. 现状核查

| 维度 | 现状 | 与诉求差距 |
|---|---|---|
| Docker 离线安装 | `bundle/docker/install-docker.sh` 已支持：解包二进制 → 写 containerd / docker systemd 单元 → `systemctl enable --now` → `docker version` 验证 | ✅ 核心已有；缺：装完后**主动引导设置镜像加速**；缺：交互/非交互双模式 `--mirror <name>` |
| 镜像加速 | **完全没有** | ❌ 需新增 `setup-mirrors.sh` |
| 镜像导入 + 启动 | 仅文档 `docs/operations/OMC内网离线部署手册（运维侧）.md` 描述步骤，无脚本 | ❌ 需新增 `deploy.sh` 一键编排 |
| 脚本 `-h` | `download-docker.sh` / `build-images.sh` / `build-release.sh` 有 `-h` ；`gen-index.sh` / `serve.sh` / `healthcheck.sh` / `install-docker.sh` **无 `-h`** | ⚠️ 部分缺；需补齐 |
| `:8000` 操作手册 | `archive/index.html` 有 7 步流程；缺访问地址 / 初始账号 / 验证方式 / 文件清单 / 首次 vs 升级分流 | ⚠️ 需扩充 |

---

## 2. 决策清单

| # | 决策点 | 选定方案 |
|---|---|---|
| D1 | 镜像加速器内置名单 | **v2 精简到 3 项**：`official`（不设置镜像）/ `daocloud`（`https://docker.m.daocloud.io`）/ `xuanyuan`（`https://docker.xuanyuan.me`）。其它历史选项（aliyun/tencent/ustc/netease/baidu/custom）已下线 |
| D2 | 设置方式 | 修改 `/etc/docker/daemon.json` 的 `registry-mirrors`；若文件已有其它键则 **merge 而非覆盖** |
| D3 | 镜像加速触发时机 | install-docker.sh **执行末尾自动提示** "是否配置加速？" → 是则调 setup-mirrors.sh；批处理可走 `--mirror <name>` 一次过；交付侧也可后期单独运行 setup-mirrors.sh |
| D4 | deploy.sh 自动化粒度 | 默认全套（load 镜像 → infra up → 等就绪 → migrate → seed → app/acs/worker systemd → web up → healthcheck）；提供 `--skip-*` 标志做精细控制 |
| D5 | systemd 单元怎么装 | 项目包 `etc/systemd/*.service` 已存在 → `deploy.sh` 拷到 `/etc/systemd/system/`，`daemon-reload` + `enable --now` |
| D6 | 必须 root 的脚本 | install-docker.sh / setup-mirrors.sh / deploy.sh 全部需要 root |
| D7 | `daemon.json` 兜底 | 不强制使用 `jq`（很多内网无 jq）；用 python3 / awk fallback，按"无则建、有则改"双路径 |
| D8 | 操作手册集成位置 | 主要扩 `gen-index.sh` 渲染的 `index.html`；细节链 docs/operations 两份手册 |
| D9 | 初始账号密码展示口径 | 引用项目 `migrations/seed/000001_seed_data.sql` 的默认；密码标"**首次登录强制改**" |
| D10 | 升级分支 | index.html 显式分"首次部署"和"日常升级"两个流程 |

---

## 3. 实施清单

### 3.1 新增脚本

#### `bundle/setup-mirrors.sh`

职责：写 `/etc/docker/daemon.json` 的 `registry-mirrors`，重启 docker。

用法：
```bash
sudo bash setup-mirrors.sh                    # 交互式选单（3 选项）
sudo bash setup-mirrors.sh --mirror daocloud  # 非交互
sudo bash setup-mirrors.sh --mirror xuanyuan
sudo bash setup-mirrors.sh --mirror official  # 不设置镜像
sudo bash setup-mirrors.sh --remove           # 等价 --mirror official
sudo bash setup-mirrors.sh -h
```

内置 URL（决策 D1，**v2 精简到 3 项**）：
```bash
declare -A MIRRORS=(
  [official]=""
  [daocloud]="https://docker.m.daocloud.io"
  [xuanyuan]="https://docker.xuanyuan.me"
)
```

合并 `daemon.json` 用 python3：
```python
import json, sys, os
p = '/etc/docker/daemon.json'
data = json.load(open(p)) if os.path.exists(p) and os.path.getsize(p) > 0 else {}
data['registry-mirrors'] = sys.argv[1].split(',') if sys.argv[1] else []
json.dump(data, open(p, 'w'), indent=2)
```

#### `bundle/deploy/deploy.sh`

职责：一键端到端部署（首次 + 升级路径覆盖）。

用法：
```bash
sudo bash deploy.sh                              # 默认全套
sudo bash deploy.sh --skip-infra                 # 已部署过基础设施，仅升级 app
sudo bash deploy.sh --skip-migrate               # 不跑 migrate（已迁移过）
sudo bash deploy.sh --check-only                 # 仅检查环境，不动手
sudo bash deploy.sh -h
```

10 步流程：
```
1.  detect_arch & precheck (docker | binaries | etc/*.yaml | systemd)
2.  load_infra_images  ../images/infra-images-*.tar  → docker load
3.  ensure_env  ./deploy/.env (镜像 tag) + 提示改默认口令
4.  start_infra  docker compose -f docker-compose.infra.yml up -d
5.  wait_infra_healthy  60s 内 PG / Redis / NATS / MinIO ready
6.  run_migrate  ./bin/omcgo-migrate up
7.  run_seed     ./bin/omcgo-seed  (only on first deploy)
8.  install_systemd  omcgo-{app,acs,worker}.service → enable --now
9.  start_web   docker compose -f docker-compose.web.yml up -d
10. healthcheck ./deploy/healthcheck.sh
```

### 3.2 增强脚本

#### `bundle/docker/install-docker.sh`

- 增强 `-h` 块（list 所有参数）
- 末尾自动调用 `setup-mirrors.sh`（交互；可 `--mirror <name>` 跳过）
- 新增 `--no-mirror` 不配加速
- 检测当前用户 → 提示加入 docker 组（仅 root 模式下提示 `usermod -aG docker $SUDO_USER`）

#### `gen-index.sh` / `serve.sh` / `healthcheck.sh`

- 各加 `-h|--help` 块
- `serve.sh` 改 `getopts` 风格：`./serve.sh [-p|--port 8000]`，保留位置兼容（首参数视作端口）
- `gen-index.sh` 加 `--archive <dir>` 选项以测试

### 3.3 index.html 扩展（gen-index.sh 改造）

新版结构：

```
🚀 OMC 离线版本下载
├─ 📋 你应该下载哪些文件？
│   ├─ 首次部署：基础设施包 + 项目包（同架构、各下 .tar.xz + .sha256）
│   └─ 日常升级：只下项目包（基础设施已部署过的情况下）
├─ 📐 1. 确认目标机架构（uname -m → amd64 / arm64）
├─ 🔐 2. 校验完整性（sha256sum -c）
├─ 📦 3. 解压交付包（tar -xJf）
├─ 🐳 4. 安装 Docker（首次部署，sudo bash docker/install-docker.sh）
├─ ⚡ 5. 配置系统加速（可选；sudo bash setup-mirrors.sh — 三合一，infra 包顶层）
├─ 🚚 6. 一键部署（sudo bash deploy/deploy.sh）
├─ ✅ 7. 验证（bash deploy/healthcheck.sh）
├─ 🌐 8. 部署后访问地址
│   ├─ Web UI：http://<服务器IP>:8080
│   ├─ MinIO Console：http://<服务器IP>:9001
│   └─ Grafana（如启用监控栈）：http://<服务器IP>:3000
├─ 👤 9. 初始账号密码（首次登录强制改）
│   ├─ Web UI 管理员：admin / admin123
│   ├─ MinIO Console：minioadmin / minioadmin
│   ├─ PostgreSQL：omcgo / omcgo123
│   └─ Grafana：admin / admin
└─ 📦 项目交付包表 / 🛠 基础设施包表（原有）
```

### 3.4 文档更新

**构建侧** `docs/operations/OMC离线交付包构建手册（构建侧）.md`：
- 新增 §X 镜像加速器：bundle/setup-mirrors.sh 入参 / 流程 / 内置 URL
- 新增 §X 一键部署：bundle/deploy/deploy.sh 介绍 + `--skip-*` 用法
- 新增 §X 脚本 -h 对照表（每个脚本 → 关键参数 → 适用场景）
- 与已有"两包独立"章节交叉引用

**交付侧** `docs/operations/OMC内网离线部署手册（运维侧）.md`：
- 替换"手动 10 步"为"一键 deploy.sh + 手动可拆步骤"双路径
- 加镜像加速器章节
- 加验证清单（healthcheck + 访问 URL + 初始登录）
- 加常见问题 FAQ（基于安装 / 镜像加速 / 部署三阶段）

---

## 4. 实施顺序

1. ✅ 本设计文档
2. 新建 `bundle/setup-mirrors.sh` + `-h`
3. 增强 `bundle/docker/install-docker.sh`（调加速 + `-h` 强化）
4. 新建 `bundle/deploy/deploy.sh` + `-h`
5. 补 `gen-index.sh` / `serve.sh` / `healthcheck.sh` 的 `-h`
6. 重写 `gen-index.sh` 渲染段，加新章节
7. 更新构建侧 + 交付侧两份手册
8. 校验：`bash -n` 全部脚本通过；模拟跑 `serve.sh` 渲染 index.html 验证；shell 工具是否齐全（python3 / curl 兜底）

---

## 5. 风险与回滚

| 风险 | 缓解 |
|---|---|
| 客户内网无 python3 → daemon.json 合并失败 | 兜底：仅"无文件"场景写裸 JSON；存在文件时让用户手动 edit 并提示 |
| 镜像加速 URL 失效 | v2 内置 3 选项（official/daocloud/xuanyuan），用户随时可换；如需其它 URL 可单独编辑 `/etc/docker/daemon.json` |
| deploy.sh 在已部署环境重跑 | 全幂等：load 镜像跳过同 digest；compose up -d 自动协调；migrate 内置版本号比对 |
| systemd 单元覆盖已有 | 检测 `/etc/systemd/system/omcgo-*.service` 存在时备份 `.bak.<时间戳>` |
| Docker 重启中断在运行的容器 | setup-mirrors.sh 在 daemon.json 未变化时跳过 restart |
| index.html 默认密码引出安全审计问题 | 显著红色提示"首次登录强制改"；密码可由 release.conf 配置默认值（生产 / 测试不同） |

---

## 6. 测试

| 用例 | 方式 |
|---|---|
| 各脚本 `-h` 输出 | `bash <script> -h` 不报错、内容含全部参数 |
| `bash -n` 语法 | CI / 手动 |
| install-docker.sh 干跑 | docker 容器中模拟（先卸载再装）|
| setup-mirrors.sh merge daemon.json | 单元：临时 daemon.json + python 合并断言 |
| deploy.sh `--check-only` | 在 dev 环境跑通而不修改系统 |
| index.html 渲染 | `bash gen-index.sh` → 检查 HTML 含 8 大段 |
| 端口冲突 | serve.sh 探测 8000 占用时 fail-fast |

---

## 7. 关联资料

- 现有手册：`docs/operations/OMC离线交付包构建手册（构建侧）.md` / `OMC内网离线部署手册（运维侧）.md` / `OMC交付构建快速上手.md`
- 现有脚本：`deployments/release/*.sh` + `bundle/{docker,deploy}/*`
- Docker daemon.json 参考：https://docs.docker.com/engine/daemon/

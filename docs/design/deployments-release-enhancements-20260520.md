# deployments/release 方案增强 — 交付侧 Docker 安装 / 镜像加速 / 一键部署 / 脚本可发现性 / 操作手册

> 状态：**待审核**
> 提出时间：2026-05-20
> 涉及代码：`deployments/release/**` + `docs/operations/**`

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
| 镜像加速 | **完全没有** | ❌ 需新增 `setup-docker-mirror.sh` |
| 镜像导入 + 启动 | 仅文档 `docs/operations/OMC内网离线部署手册（运维侧）.md` 描述步骤，无脚本 | ❌ 需新增 `deploy.sh` 一键编排 |
| 脚本 `-h` | `download-docker.sh` / `build-images.sh` / `build-release.sh` 有 `-h` ；`gen-index.sh` / `serve.sh` / `healthcheck.sh` / `install-docker.sh` **无 `-h`** | ⚠️ 部分缺；需补齐 |
| `:8000` 操作手册 | `archive/index.html` 有 7 步流程；缺访问地址 / 初始账号 / 验证方式 / 文件清单 / 首次 vs 升级分流 | ⚠️ 需扩充 |

---

## 2. 决策清单

| # | 决策点 | 选定方案 |
|---|---|---|
| D1 | 镜像加速器内置名单 | 6 项：`official`（无加速）/ `aliyun` / `tencent` / `ustc`（中科大）/ `netease`（网易）/ `baidu`（百度云）+ `custom`（用户输入 URL） |
| D2 | 设置方式 | 修改 `/etc/docker/daemon.json` 的 `registry-mirrors`；若文件已有其它键则 **merge 而非覆盖** |
| D3 | 镜像加速触发时机 | install-docker.sh **执行末尾自动提示** "是否配置加速？" → 是则调 setup-docker-mirror.sh；批处理可走 `--mirror <name>` 一次过；交付侧也可后期单独运行 setup-docker-mirror.sh |
| D4 | deploy.sh 自动化粒度 | 默认全套（load 镜像 → infra up → 等就绪 → migrate → seed → app/acs/worker systemd → web up → healthcheck）；提供 `--skip-*` 标志做精细控制 |
| D5 | systemd 单元怎么装 | 项目包 `etc/systemd/*.service` 已存在 → `deploy.sh` 拷到 `/etc/systemd/system/`，`daemon-reload` + `enable --now` |
| D6 | 必须 root 的脚本 | install-docker.sh / setup-docker-mirror.sh / deploy.sh 全部需要 root |
| D7 | `daemon.json` 兜底 | 不强制使用 `jq`（很多内网无 jq）；用 python3 / awk fallback，按"无则建、有则改"双路径 |
| D8 | 操作手册集成位置 | 主要扩 `gen-index.sh` 渲染的 `index.html`；细节链 docs/operations 两份手册 |
| D9 | 初始账号密码展示口径 | 引用项目 `migrations/seed/000001_seed_data.sql` 的默认；密码标"**首次登录强制改**" |
| D10 | 升级分支 | index.html 显式分"首次部署"和"日常升级"两个流程 |

---

## 3. 实施清单

### 3.1 新增脚本

#### `bundle/docker/setup-docker-mirror.sh`

职责：写 `/etc/docker/daemon.json` 的 `registry-mirrors`，重启 docker。

用法：
```bash
sudo bash setup-docker-mirror.sh                  # 交互式选单
sudo bash setup-docker-mirror.sh --mirror aliyun  # 非交互
sudo bash setup-docker-mirror.sh --mirror custom --url https://my-mirror.example.com
sudo bash setup-docker-mirror.sh --remove         # 取消加速（回归官方）
sudo bash setup-docker-mirror.sh -h
```

内置 URL（决策 D1）：
```bash
declare -A MIRRORS=(
  [official]=""
  [aliyun]="https://registry.aliyuncs.com,https://hub-mirror.c.163.com"
  [tencent]="https://mirror.ccs.tencentyun.com"
  [ustc]="https://docker.mirrors.ustc.edu.cn"
  [netease]="https://hub-mirror.c.163.com"
  [baidu]="https://mirror.baidubce.com"
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
- 末尾自动调用 `setup-docker-mirror.sh`（交互；可 `--mirror <name>` 跳过）
- 新增 `--no-mirror` 不配加速
- 检测当前用户 → 提示加入 docker 组（仅 root 模式下提示 `usermod -aG docker $SUDO_USER`）

#### `gen-index.sh` / `serve.sh` / `healthcheck.sh`

- 各加 `-h|--help` 块
- `serve.sh` 改 `getopts` 风格：`./serve.sh [-p|--port 8000]`，保留位置兼容（首参数视作端口）
- `gen-index.sh` 加 `--archive <dir>` 选项以测试

### 3.3 index.html 扩展（gen-index.sh 改造）

新版结构：

```
🚀 OMC 离线交付包下载
├─ 📋 你应该下载哪些文件？
│   ├─ 首次部署：基础设施包 + 项目包（同架构、各下 .tar.xz + .sha256）
│   └─ 日常升级：只下项目包（基础设施已部署过的情况下）
├─ 📐 1. 确认目标机架构（uname -m → amd64 / arm64）
├─ 🔐 2. 校验完整性（sha256sum -c）
├─ 📦 3. 解压交付包（tar -xJf）
├─ 🐳 4. 安装 Docker（首次部署，sudo bash docker/install-docker.sh）
├─ ⚡ 5. 配置镜像加速（可选；sudo bash docker/setup-docker-mirror.sh）
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
- 新增 §X 镜像加速器：bundle/docker/setup-docker-mirror.sh 入参 / 流程 / 内置 URL
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
2. 新建 `bundle/docker/setup-docker-mirror.sh` + `-h`
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
| 镜像加速 URL 失效 | 内置 5+ 选项，用户随时可换；提供 `--mirror custom --url ...` 自定义 |
| deploy.sh 在已部署环境重跑 | 全幂等：load 镜像跳过同 digest；compose up -d 自动协调；migrate 内置版本号比对 |
| systemd 单元覆盖已有 | 检测 `/etc/systemd/system/omcgo-*.service` 存在时备份 `.bak.<时间戳>` |
| Docker 重启中断在运行的容器 | setup-docker-mirror.sh 在 daemon.json 未变化时跳过 restart |
| index.html 默认密码引出安全审计问题 | 显著红色提示"首次登录强制改"；密码可由 release.conf 配置默认值（生产 / 测试不同） |

---

## 6. 测试

| 用例 | 方式 |
|---|---|
| 各脚本 `-h` 输出 | `bash <script> -h` 不报错、内容含全部参数 |
| `bash -n` 语法 | CI / 手动 |
| install-docker.sh 干跑 | docker 容器中模拟（先卸载再装）|
| setup-docker-mirror.sh merge daemon.json | 单元：临时 daemon.json + python 合并断言 |
| deploy.sh `--check-only` | 在 dev 环境跑通而不修改系统 |
| index.html 渲染 | `bash gen-index.sh` → 检查 HTML 含 8 大段 |
| 端口冲突 | serve.sh 探测 8000 占用时 fail-fast |

---

## 7. 关联资料

- 现有手册：`docs/operations/OMC离线交付包构建手册（构建侧）.md` / `OMC内网离线部署手册（运维侧）.md` / `OMC交付构建快速上手.md`
- 现有脚本：`deployments/release/*.sh` + `bundle/{docker,deploy}/*`
- Docker daemon.json 参考：https://docs.docker.com/engine/daemon/

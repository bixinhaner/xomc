# deployments/release — OMC 离线交付包

本目录用于**生成、存放交付给运维的离线交付包**。配套文档：

- 构建侧（本目录工具的完整说明）：[`docs/operations/OMC离线交付包构建手册（构建侧）.md`](../../docs/operations/OMC离线交付包构建手册（构建侧）.md)
- 运维侧（随交付包发给运维）：[`docs/operations/OMC内网离线部署手册（运维侧）.md`](../../docs/operations/OMC内网离线部署手册（运维侧）.md)

## 目录结构

```
deployments/release/
├── README.md                 # 本文件
├── release.conf              # 可配置项：版本号、基础设施镜像清单
├── build-release.sh          # ⭐ 生成交付物的工具（构建侧执行）
├── bundle/                   # 进交付包的静态模板（被原样拷入每个交付包）
│   ├── docker/
│   │   ├── install-docker.sh         # 运维侧离线安装 Docker
│   │   └── README.md                 # 说明须放入的 Docker 静态二进制包
│   └── deploy/
│       ├── docker-compose.infra.yml  # 基础设施（postgres/redis/nats/minio）
│       ├── docker-compose.web.yml    # 前端 nginx
│       ├── healthcheck.sh            # 运维侧启动校验
│       └── systemd/                  # OMC 三进程 systemd 单元模板
│           ├── omcgo-app.service
│           ├── omcgo-acs.service
│           └── omcgo-worker.service
└── dist/                     # 交付包产出目录（git 忽略，仅留 .gitkeep）
```

## 使用（构建侧 — 需有公网、Go 1.25+、Node 20+、Docker）

```bash
cd deployments/release

# 1) 把 Docker 静态二进制包放进 bundle/docker/（一次性，见 bundle/docker/README.md）

# 2) 生成交付包（默认同时产出 amd64 + arm64 两个包）
./build-release.sh -v 1.2.0

# 可选参数：
#   -v <版本>          指定版本号（缺省取 git describe）
#   --arch amd64       只构建指定架构（缺省 amd64+arm64）
#   --with-monitoring  额外打包监控栈镜像（prometheus/grafana/loki...）
```

产出：

```
dist/omc-release-1.2.0-amd64.tar.gz       + .sha256
dist/omc-release-1.2.0-arm64.tar.gz       + .sha256
```

## 工具做了什么（对应《构建手册》§6）

1. 交叉编译 OMC 6 个 Go 二进制（`omcgo-app/acs/worker/migrate/seed`、`omcctl`），
   `CGO_ENABLED=0` 静态编译，amd64 与 arm64 各一套。
2. 构建前端静态资源（`omcmb/webcode` → `dist/`，架构无关）。
3. 收集 `data/`（字典 XML）、`configs/`（Casbin 模型）、`migrations/`（迁移 SQL）、
   `etc/`（`*.prod.yaml` 配置模板）。
4. 按架构 `docker pull --platform` + `docker save` 基础设施镜像。
5. 拷入 `bundle/` 下的部署模板（compose / systemd / 脚本）与《运维侧手册》。
6. 生成 `VERSION`、`README.md`、`checksums.sha256`，打包为 `tar.gz` + 校验和。

> 运维侧拿到 `tar.gz` 后**无需 Go、无需 Node、无需联网**，按《运维侧手册》第 5 章操作即可。

## 注意

- **镜像版本固化**：`release.conf` 里的镜像标签建议改为具体版本号，不要用
  `latest` / `latest-pg16`，避免不同批次交付包镜像不一致（《构建手册》§7）。
- `dist/` 是构建产物目录，已被 `.gitignore` 忽略，不会进仓库。

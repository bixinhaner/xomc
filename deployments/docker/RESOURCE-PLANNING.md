# OMC 本地 dev 栈资源动态规划

> 解决「每次换机器 / 调 Docker Desktop VM 大小，都要手改 docker-compose.yml 里十几个
> CPU/内存限额 + 两个 PG 的调优参数」的麻烦。一条命令按本机硬件自动算好并起栈。
>
> 这是生产交付包 `deployments/release/bundle/deploy/`（[plan-resources.sh](../release/bundle/deploy/plan-resources.sh)
> + [RESOURCE-PLANNING.md](../release/bundle/deploy/RESOURCE-PLANNING.md)）那套方案在
> **本地单文件 dev 栈**上的对应物。资源切分依据见
> [docs/operations/OMC内存分配与容量规划-20260611.md](../../docs/operations/OMC内存分配与容量规划-20260611.md)。

---

## 1. TL;DR 日常用法

```bash
# 在仓库根 goomc/ 下执行。dc.sh 是 docker compose 的包装器：
#   首次自动按本机硬件生成 resources.env，并始终带 --env-file 喂给 compose。
bash deployments/docker/dc.sh up -d --build        # 起栈（首次自动规划）
bash deployments/docker/dc.sh ps
bash deployments/docker/dc.sh logs -f --tail=100 acs
bash deployments/docker/dc.sh down

bash deployments/docker/dc.sh --replan up -d        # 换了机器 / 调了 VM：强制重新探测再起
```

不想用 wrapper、手工跑也行：

```bash
docker compose -p omc \
  --env-file deployments/docker/resources.env \
  -f deployments/docker/docker-compose.yml up -d
```

**不传 `--env-file` 时**，compose 取 `${VAR:-默认}` 里的默认值（= 历史「PM 写吞吐优化档」），
行为与改造前**完全一致**——所以这套机制是纯增量、零风险的。

---

## 2. 现在资源是怎么分配的（改造后）

| 维度 | 机制 |
|------|------|
| **CPU** | 每服务 `deploy.resources.limits.cpus`，参数化为 `${PG_CPUS:-8}` 等；CPU 是可突发的软上限 |
| **内存** | 每服务 `limits.memory`，参数化为 `${PG_MEM:-10g}` 等；cgroup 硬限/熔断 |
| **缓存（Redis）** | `--maxmemory ${REDIS_MAXMEMORY:-512mb}` + `allkeys-lru`；OMC 实测用量仅数 MB，封顶 512MB 防过配 |
| **PostgreSQL 调优** | 两个实例各自的 `shared_buffers / effective_cache_size / work_mem / maintenance_work_mem / max_wal_size / max_connections` 全参数化 |
| **Go 运行时** | acs/app/worker 新增 `GOMEMLIMIT`（堆软限，把硬 OOM-kill 换成 GC 背压）+ `GOMAXPROCS`（对齐 CFS 配额）|
| **磁盘 / 存储路径** | 数据走 Docker 命名卷（pgdata/tsdbdata/redisdata/natsdata/miniodata…）落在 Docker 引擎盘；日志 bind-mount 到 `run/logs/`。脚本探测盘可用空间并在偏小时告警（生产时序量大须独立数据盘，见 §6）|

**两个 PostgreSQL 实例**：主库 `postgres`（业务数据）+ 时序库 `postgres-tsdb`（PM/KPI/告警历史，
KPI 物理分离后写压力主要在此）。二者都吃 `shared_buffers`，故规划脚本按「双 PG」做内存切分与 OOM 自检。

---

## 3. 脚本怎么算（plan-resources.sh）

### 3.1 探测口径——用 `docker info`，不是物理内存

容器真正的天花板是 **Docker 引擎可分配的量**：
- **Docker Desktop（Mac/Win）**：是 **VM 上限**（`docker info` 的 MemTotal/NCPU）。物理 Mac 24g
  但 VM 可能仅 ~11.67g，按物理算会严重超分。
- **Linux**：`docker info` = 宿主总量，容器可用到此。

docker 不可达时回退宿主探测（仅供预览），并提示在 docker 可用时重跑。

### 3.2 空闲预算

```
引擎保留     = max(1 GiB, VM 总量 × 10%)              # VM 内核 / dockerd
其它项目占用 = Σ(非本 OMC 栈的运行中容器现用量)         # 共享引擎时替 boss 等栈留出余量
空闲预算     = VM 总量 − 引擎保留 − 其它项目占用
```

`--assume-dedicated` 视引擎为 OMC 独占，不扣其它项目；`--self-project` 指定「算作本栈而跳过」的
compose 项目名（默认 `omc omcgo`）。

### 3.3 floor-first 分配（单一自适应，不分业务 profile）

每个组件先拿到 `floor` 下限（绝不低于，否则重演 Redis/PG OOM），只有「空闲余量」才按权重在
`floor..ceiling` 间向上伸缩。权重已内含相对重要度（**tsdb / worker / postgres 重，web 轻**）——
这就是「按业务倾斜」的部分，只是没有可切换的 profile。

| 组件 | floor | ceiling | 权重 | 依据 |
|------|------:|--------:|:---:|------|
| postgres-tsdb | 1024 | 8192 | 22 | 时序库，PM COPY 入库 + KPI 聚合，写压力主要在此 |
| worker | 768 | 2560 | 20 | PM/MR XML 解析最吃内存 |
| postgres | 1024 | 6144 | 14 | 业务主库；时序已分离，本库写量较小 |
| acs | 512 | 2048 | 12 | TR-069 长连接堆；>1万会话走横向多副本 |
| app | 512 | 1536 | 8 | 运维 UI + OSS 轮询，非设备量驱动 |
| nats | 384 | 1024 | 6 | JetStream + PM 突发 in-flight |
| minio | 512 | 2048 | 6 | 对象存储，瓶颈在磁盘非内存 |
| redis | 384 | 1024 | 4 | OMC 实测仅数 MB，ceiling 低是有意 |
| web | 192 | 512 | 0 | nginx 静态+反代，近似固定 |

监控栈默认**不计入**（dev 本地通常不起）；`--with-monitoring` 把约 4.1 GiB 固定块计入预算。

### 3.4 联动派生（同源、锁死一致）

`limits.memory` 一变，下列进程内上限由同一预算同步算出：

| 派生项 | 公式 |
|--------|------|
| `GOMEMLIMIT` | `0.90 × *_MEM`（软限，10% 余量给二进制/mmap） |
| `GOMAXPROCS` | `floor(cpus)` |
| PG `shared_buffers` | `0.25 × 该实例 MEM`（留 OS page cache 给 Timescale 解压） |
| PG `effective_cache_size` | `0.60 × 该实例 MEM` |
| PG `max_connections` | `300`（覆盖 Go 端 ~180 池 + exporter + psql + 余量） |
| Redis `maxmemory` | `min(512MB, REDIS_MEM − 256)`，且保证 `cap − maxmemory ≥ 256MiB`（AOF rewrite COW 余量） |

### 3.5 双 PG 合计实占 OOM 自检（关键）

只有 PG 会吃满（`shared_buffers + maintenance + backends/work 余量`）。脚本算「留给两 PG 的安全
预算」`PG_AVAIL = VM − 引擎保留 − 其它占用 − 非PG服务实占估`，再校验两库合计实占 ≤ `PG_AVAIL`：
- 超了 → **按比例下调两库 `shared_buffers`**（保 ≥128MB）并告警；
- 连两库最低配都放不下 → **die**，给出建议（调大 VM / 释放其它项目 / 临时只起单库）。

---

## 4. resources.env 怎么用 / 怎么改

`plan-resources.sh` 写出的 [resources.env](resources.env) 是**纯文本含注释、本机相关、不入库**
（已加 `.gitignore`）。可手改，约束写在文件头：

- `*_GOMEMLIMIT` 必须 < 对应 `*_MEM`（建议 0.90×）
- `REDIS_MEM` 必须 ≥ `REDIS_MAXMEMORY + 256MiB`
- 两 PG 的 `shared_buffers` 之和 + maintenance + backends 余量 须 < VM 内存
- `PG/TSDB_MAX_CONNECTIONS` 必须 ≥ Go 端连接池总和（当前 ~180）

改完 `bash deployments/docker/dc.sh up -d`（或手工带 `--env-file`）即按新值重建变化的容器。

> ⚠️ **与 PM 写吞吐压测档的关系**：compose 内的 `:-` 默认值是历史「PM 写吞吐优化档」
> （两 PG 各 10g / shared_buffers 3GB，为单机压测刻意超分）。自动规划给出的是**安全自适应**值
> （按 VM 实际大小，通常远小于 10g）。要跑 PM 压测时：**要么不带 `--env-file`**（用 compose 默认的
> 大档），**要么手动调大** `resources.env` 里的 `PG_MEM/TSDB_MEM/PG_SHARED_BUFFERS/...`。

---

## 5. 命令速查

```bash
# 规划（不起栈）
bash deployments/docker/plan-resources.sh --dry-run        # 只预览
bash deployments/docker/plan-resources.sh                  # 写 resources.env
bash deployments/docker/plan-resources.sh --with-monitoring        # 计入监控栈
bash deployments/docker/plan-resources.sh --assume-dedicated       # 引擎 OMC 独占
bash deployments/docker/plan-resources.sh --self-project omc       # 自定义本栈项目名

# what-if（为别的机器预规划 / 测试）
OMC_PROBE_CPU=8 OMC_PROBE_MEM_TOTAL_MIB=8192 \
  bash deployments/docker/plan-resources.sh --dry-run

# 起栈（wrapper，自动带 --env-file）
bash deployments/docker/dc.sh up -d --build
bash deployments/docker/dc.sh --replan up -d               # 换机器后强制重规划
OMC_SKIP_AUTOPLAN=1 bash deployments/docker/dc.sh up -d     # 跳过规划，用 compose 默认大档
```

---

## 7. 生产服务器：`--maximize` 模式 + 保留期自动测算

dev 默认是 floor-first + 低 ceiling（给 ~12 GB VM）。**独占型生产服务器**（如 40C/80T、64/128 GB）
要最大化吃硬件，用 `--maximize`：

```bash
# 每台服务器独立跑（每台独立全栈）。--assume-dedicated = 整机 OMC 独占。
bash deployments/docker/plan-resources.sh --maximize --assume-dedicated --devices 8000
bash deployments/docker/dc.sh up -d --build      # wrapper 自动带生成的 resources.env
```

`--maximize` 与 dev 的区别：

| 维度 | dev（默认） | `--maximize`（生产独占机） |
|------|------|------|
| **CPU 限额** | 按核数缩放、封顶 | **每服务 = 全部逻辑核**（谁抢到是谁的，不按进程切）|
| `GOMAXPROCS` | `floor(cpus)` | 空（Go 用满所有核）|
| **内存分配** | floor-first，低 ceiling | **PG 最佳实践**：`shared_buffers` 总量 ≈ **25% 物理内存**（其余留 OS page cache，TimescaleDB 列存解压/读全靠它），两库按时序库偏重切（tsdb 60% / 主库 40%）|
| work_mem | 8–16 MB | 32 MB（≥96 GB→48 MB）|
| 其它服务 cap | 低 ceiling | 按物理内存比例给足（worker 10%、acs 6%…）|

实测（dry-run 验证）：

| 服务器 | shared_buffers 总量 | 主库 / tsdb cap | worker | 60 天保留 |
|------|------|------|------|------|
| 64 GB / 4 TB | 16 GB(25%) | 16 / 24 GB | 6.5 GB | 8000 台 → **自动降到 48 天**（盘不够）|
| 128 GB / 8 TB | 32 GB(25%) | 32 / 49 GB | 13 GB | 8000 台 → **60 天 OK** |

> caps 之和允许 > 物理内存（仅 PG/worker 真吃得多，原则1）；其余内存自然成为 OS page cache。

## 8. 保留期自动测算（磁盘 → 15min 原始数据可保留天数）

你要「KPI 数据库数据 / 15 分钟数据保留 60 天，磁盘存不下就自动算」。脚本据**盘可用容量 + 设备数**算出
可行天数（上限 `--retention-days`，默认 60），不够则自动下调，并写 `retention.sql`：

```bash
bash deployments/docker/plan-resources.sh --maximize --assume-dedicated --devices 8000
# → 算出可行天数，写 deployments/docker/retention.sql（UPDATE sys_configs pm.retention.raw_15min_days）
# 应用（sys_configs 改后服务热加载，无需重启）：
docker compose ... exec -T postgres-tsdb psql -U omcgo -d omcgo -f - < deployments/docker/retention.sql
```

- 不给 `--devices` → 只报「本盘 @ 60 天约可承载几台设备」。
- 模型透明（混合制式 ~17 MB/设备/天、7 天后列存压缩 ~12×、MinIO 原始 .gz ~3 MB/天、留 30% watchdog 余量），
  实际随 RAT/压缩比浮动，仅作规划锚点。
- **只设 DB 侧 15min 原始保留**（`pm.retention.raw_15min_days`，本就可配置、热加载）。**以下属后端特性、不是脚本能配的**，已登记 Issue：
  - MinIO 原始件 ILM（现**硬编码 14 天** `internal/core/components/minio/minio.go`）→ 改可配置、对齐 60 天；
  - 基站日志（现 **20 文件配额**，非按时间）→ 加 60 天时间保留；
  - 磁盘超 N% 停收 PM / 回落恢复（watchdog 背压）；入库后原始 XML 压缩回 MinIO。

## 6. 磁盘 / 存储路径

- dev 数据走 Docker 命名卷，落在 Docker 引擎盘（Docker Desktop 下是 VM 盘）。脚本探测盘可用空间，
  < 40 GiB 时告警。Docker Desktop 下宿主读不到 VM 盘真实剩余（显示 0），属预期，请在 Docker
  Desktop 设置里看 Disk image size。
- **生产**：时序数据量大（混合制式约 150MB/设备/30天，256GB 盘约 1.6k 设备即写满），须把
  `pgdata/tsdbdata/miniodata` 改 bind-mount 到独立数据盘 + 配保留期/降采样。见容量规划 §4.3 与
  §5。本地 dev 一般无需改卷路径（改卷会孤立既有数据，慎重）。

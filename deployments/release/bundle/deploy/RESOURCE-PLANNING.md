# OMC 容器资源动态规划方案

> 目的：把写死在 4 个 compose 文件里的资源限额，改为**按目标服务器的空闲资源动态计算**——
> 既不让大机白白闲置，也不在共享主机上超分压垮别的项目。
>
> 本文档 = 设计方案（供评审）。配套脚本 `plan-resources.sh` 已可运行；compose / install.sh
> 的接线（Phase 2）**尚未实施**，待本方案确认后再做。

---

## 1. 背景：为什么要做这件事

当前 `deployments/release/bundle/deploy/docker-compose.*.yml` 的资源限额是**写死的字面量**，
且与本地 dev 栈 `deployments/docker/docker-compose.yml` 完全一致。`install.sh` 的 precheck
（[install.sh:141-230](install.sh)）只校验工具与包结构，**从不读主机 CPU/内存**。后果：

- 8c/16g 的实验机和 64c/256g 的生产机拿到**同一套上限**——大机严重浪费，小机静默 OOM。
- 没有最低配置门禁：主机过小时不是「装不上」，而是装上后容器反复 OOM 重启（晚而隐蔽）。

这正是 Redis 事故（commit `7f990e46`）的同一类病根：**cgroup 硬限设了，但进程内没有
配套上限**，数据集涨到 2.36G 撞 512m 限额被 kill → crash-loop 拖垮 app/acs/worker。
经一轮多视角分析 + 对抗评审，确认还有**两处同类隐患**未爆：

| 隐患 | 现状 | 风险 |
|------|------|------|
| **Go 容器无 `GOMEMLIMIT`** | app/acs/worker 全无（全仓确认） | Go 1.25 能从 cgroup 推 `GOMAXPROCS`，但**不推内存软限**；GC 按 2×存活堆决策、无视 1g/768m 上限，突发即 OOM。这是当前最高危、与规模无关的地雷 |
| **PG 连接池超配** | app(60+40)+acs(30)+worker(25+25)=**180** vs PG 默认 `max_connections=100` | 单副本即可在唤醒风暴下耗尽连接报 `FATAL: too many clients`；`app.prod.yaml` 注释已自承须上 pgbouncer |
| PG 零调优 | command 只设时区，`shared_buffers` 跑 128MB 默认、`effective_cache_size` 4GB 默认对 2g 容器说谎 | 时序扫描成本误判、写入吞吐受限 |

> 评审同时**下调**了两条原判：Redis `限额:maxmemory=1.5×` 比例其实够用（COW 实测峰值 ~2.46g < 3g）；
> worker「1000 并发解析 OOM」机制不成立（push handler 同步逐条处理，并发=副本数而非 1000）。
> 这些修正已并入下方设计。

---

## 2. 需求与落地映射（你的 6 点）

| # | 需求 | 落地 |
|---|------|------|
| ① | 探测服务器配置 | `plan-resources.sh` 第 1 步：`nproc` / `/proc/meminfo` / 磁盘（macOS 用 `sysctl`/`vm_stat` 供本机预览） |
| ② | 探测当前状态/负荷，避免与其它项目共用超分 | 用 **`MemAvailable`**（天然反映其它进程当前占用）为基准；再扫描**非 omcgo 容器**的「已声明但未用」预留额并扣除；读 load average |
| ③ | 按空闲资源算分配 → 生成 .env | 第 2-4 步：算空闲预算 → floor-first 分配 → 写 `resources.env` |
| ④ | 部署人员可查看/调整 | `resources.env` 纯文本、含注释与约束说明；`--dry-run` 只预览不写 |
| ⑤ | install.sh 按 .env 动态部署 | **Phase 2**：compose 字面量改 `${VAR:-默认}`；install.sh 加 `--env-file resources.env` |
| ⑥ | 改 .env 后重启即时生效 | **Phase 2**：`docker compose ... up -d` 重读 env-file，按新限额重建变化的容器 |

---

## 3. 分配算法（脚本核心逻辑）

### 3.1 空闲预算（口径 = 空闲优先）

```
OS 保留      = clamp(总内存 / 8, 下限 4 GiB, 上限 16 GiB)   # 留给内核/dockerd/其它系统进程
其它项目预留 = Σ(非omcgo容器 mem_limit − 当前用量)   # 替别的项目留出它们能涨到的余量
内存空闲预算 = MemAvailable − OS保留 − 其它项目预留
主机 CPU 保留 = clamp(核数 / 8, 下限 4 核, 上限 12 核)
CPU 空闲预算 = nproc − 主机CPU保留 − max(其它容器CPU, ⌈load15⌉)   # 仅作下限参考，限额可突发超分
```

保留量随主机规模缩放：32核/32GiB → 保留4核/4GiB（业务28核/28GiB）；64核/64GiB → 保留8核/8GiB；
再大依此类推，CPU 封顶 12 核（≥96核触顶）、内存封顶 16GiB（≥128GiB 触顶）。

`MemAvailable` 是关键：它已经扣掉了别的项目**当前**占用的内存；再减去它们「声明了上限但还没用满」
的部分，就得到 OMC 真正能安全吃下的空闲量。共享主机由此天然防超分。

### 3.2 floor-first 分配（评审关键修正）

朴素的「权重 × 预算」会在最低配主机上把 Postgres 压到 2g 却仍跑 `max_connections=200`——
**正好重演它要修的那个 OOM**。所以改为：

1. **先发下限**：每个组件先拿到 100k 基线 `floor`（绝不低于）。
2. **门禁（带容忍度）**：`Σfloor`（含监控）是最低门槛；`--floor-tolerance-pct`（默认 30）
   划出一个缓冲带——缺口在容忍度内（即 `空闲预算 ≥ Σfloor × (1 − 容忍度%)`）只降级为
   **WARN + 按下限分配**（不再向上伸缩），不阻断部署；缺口超过容忍度才 **die + 给建议最低配**。
   压测/生产实测组件很少同时打满 floor，留一点容忍度换可用性，比直接拒绝部署更实用。
3. **再分余量**：`剩余 = 空闲预算 − Σfloor`，按权重分给可伸缩组件，每个**封顶到 `ceiling`**。
4. **全量记账**：nats/minio/web/monitoring 也计入预算；最后校验 `Σ限额 ≤ 空闲预算`。

### 3.3 组件 floor / ceiling 表

单位 MiB。`floor` = 100k 基线（单机单副本）下限；`ceiling` = 单机纵向上限，再大走横向扩展。

| 组件 | floor | ceiling | 余量权重 | 依据 |
|------|------:|--------:|:-------:|------|
| app | 1536 | 3072 | 10% | 非设备量驱动（运维UI+OSS轮询），最先让出预算；线上巡检发现 995MiB 配额下常驻内存已到 88%，floor/ceil 上调留余量 |
| acs | 4096 | 6144 | 18% | TR-069 最热堆（1万并发会话+20万限流器映射+50MB SOAP体）；1M 走横向多副本；omc78 压测实测 max_inflight 调大后并发 PM 上传缓冲内存需求上升，1-2GiB 下触发 cgroup OOMKilled 循环重启，floor 提到 4GiB（2026-07-21）|
| worker | 1024 | 2048 | 25% | PM/MR XML 解析最吃内存（111→1111 文件/s）；1M 走横向 |
| **postgres** | **5120** | 16384 | 25% | **须容 `max_connections=200`**（180池+余量）：shared_buffers+maint+200×(10+work_mem) 须舒适放进限额 |
| **postgres-tsdb** | **4096** | 12288 | 22% | **时序库（#347）独立 TimescaleDB 实例**：PM COPY 入库 + KPI 聚合写主要在此；与主库分别计入预算，防双 PG 同机超分 OOM。同源派生 `TSDB_*`（shared_buffers 25% 等），`max_connections=300` 与主库对齐（实际池仅 ~65，余量充足） |
| redis | 5120 | 8192 | 15% | 12 分钟关窗会短时并存相邻两个小时的约 4 万设备窗口；20k 基站实测需 4GiB `maxmemory`，另留 1GiB AOF rewrite COW 余量 |
| nats | 512 | 2048 | 5% | JetStream file store |
| minio | 3072 | 4096 | 8% | 对象存储；压测发现按可见CPU配额自动估算的并发上限过于保守，且线上巡检 2.5GiB 配额下已到 88%，floor/ceil 上调留余量 |
| web | 512 | 512 | 0% | nginx 静态+反代，固定 |
| monitoring | 4224（固定块） | — | — | prometheus/loki/tempo/otelcol/grafana/alertmgr/exporters；`--skip-monitoring` 整块去除 |

### 3.4 联动派生（同源、锁死一致）

**不可违反的不变量**：`limits.memory` 一变，下列进程内上限**必须由同一预算同步算出**，
绝不手敲独立字面量（这正是 Redis 事故的根因）：

| 派生项 | 公式 | 说明 |
|--------|------|------|
| `GOMEMLIMIT` | `0.90 × *_MEM` | **软限**：把硬 OOM-kill 换成 GC 背压。10% 余量给二进制/.rodata/mmap arena/CGO（**不**含 goroutine 栈——栈在 GOMEMLIMIT 内）。⚠️ 软限不防「真实存活集超限」，ACS 须配合准入控制 + `max_request_body_size` |
| `GOMAXPROCS` | `floor(cpus)` | 对齐 CFS 配额；尤其修小数 `1.5cpu` 被 Go 向上取整成 2 导致的节流 |
| PG `shared_buffers` | `0.25 × PG_MEM` | 取 25% 非 40%——留 OS page cache 给 Timescale 列存解压 |
| PG `effective_cache_size` | `0.70 × PG_MEM` | 规划器提示（当前 4GB 默认对 2g 容器说谎） |
| PG `max_connections` | `200`（固定） | 覆盖 Go 端 180 池 + 余量；自检饱和估算逼近限额时告警建议 pgbouncer |
| Redis `maxmemory` | `REDIS_MEM − 1GiB`；`REDIS_MEM` 下限 5GiB | 覆盖 12 分钟关窗造成的双小时窗口重叠，并保留 1GiB COW 余量；策略固定 `noeviction` |
| `TSDB_*`（时序库 #347） | 同各 PG 公式 | 独立实例 postgres-tsdb 同源派生 `shared_buffers`/`effective_cache_size`/`work_mem`/`maint`/`max_wal`；`max_connections=300` 与主库对齐 |

---

## 4. 档位与最低配置

档位按**空闲内存**自动判定（`--tier` 可手动覆盖）：

| 档位 | 空闲内存 | 目标规模 | 拓扑 |
|------|---------|---------|------|
| small | ≥ ~19 GiB（达下限即可，#347 起含两 PG）| 100k 单副本 | 单机；推荐 16c/32g |
| medium | ≥ 24 GiB | 100k 满突发 / 300-500k | 单机 + pgbouncer + acs/worker ×2-3（Phase 2/手动） |
| large | ≥ 48 GiB | 1M | **多机**：acs/worker ×6-8 + pgbouncer + Redis 拆分；脚本仅规划单机切片并告警 |

medium/large 档按 2026-07-23 的 10000 基站压测热点分配 CPU：`POSTGRES_CPUS`
取宿主总核数三分之一、`TSDB_CPUS` 取二分之一、`WORKER_CPUS` 取四分之一，
均向下取整且最低 2 核。当前 32 核 / 32 GiB 生产机型因此得到 10/16/8。

依据：数据库原 4 核配额下主库约 406%、TimescaleDB 约 376% 持续接近上限；
调整为 10/10/3 后数据库瓶颈打开，但 worker 持续触及 3 核。worker 3→5 核使
PM backlog 峰值从约 6511 降到 4491（约 -31%），排空速度从约 18-20 条/s 提升到
约 25-29 条/s；继续将 worker/TSDB 调为 8/16 后，预热结束的完成速率约
45-56 条/s，TimescaleDB 实际使用约 13-15 核，宿主 load 峰值约 24/32，且均无
重投、OOM 或业务处理错误。CPU limit 是上限而非预留，三者限额之和允许超过物理
核数，由调度器按实时负载共享。

**最低配置门禁（带 30% 容忍度）**：`空闲预算 < Σfloor` 时，缺口 ≤ `--floor-tolerance-pct`
（默认 30%）先降级为 WARN 按下限分配放行；缺口超过容忍度才 `die`，给出检测值 vs 需求值 +
建议最低配 + 逃生口（`--skip-monitoring` 约降到 20 GiB / `--assume-dedicated` / 释放其它
项目 / 调大 `--floor-tolerance-pct`）。全栈推荐底线 **≥ 28 GiB**，`--skip-monitoring` 约
**20 GiB**（#347 起含独立时序库 postgres-tsdb，较单 PG 时上调约 4 GiB）；容忍度带内
（如 32 核/31 GiB 机器）仍可放行，但各组件同时打满 limit 时有 OOM 风险，建议尽快扩容内存。

> **1M 明确超出单机范围**（~3333 会话/s + ~1111 PM 文件/s），脚本检出 large 档时
> 给多机拓扑建议而非假装单机能扛。

---

## 5. 使用方式

```bash
cd deployments/release/bundle/deploy

./plan-resources.sh --dry-run              # 只预览规划，不写文件
./plan-resources.sh                        # 探测 + 计算 + 写 resources.env
./plan-resources.sh --skip-monitoring      # 不部署监控栈，降低门槛
./plan-resources.sh --tier medium          # 手动指定档位
./plan-resources.sh --assume-dedicated     # 本机 OMC 独占，不扣其它容器预留

# 在构建机为「目标主机」预规划（what-if）：
OMC_PROBE_CPU=32 OMC_PROBE_MEM_TOTAL_MIB=65536 OMC_PROBE_MEM_AVAIL_MIB=61440 \
  ./plan-resources.sh --dry-run
```

部署人员检视/微调 `resources.env` 后，运行 `install.sh`（Phase 2 后将自动消费该文件）。

---

## 6. Phase 2 接线（✅ 已实施）

> 已落地为可运行代码（`docker compose config` 双向验证：无 resources.env 渲染 = 历史值、零告警；
> 有 resources.env 各旋钮按文件覆盖）。下列为实现要点。

1. **compose 模板化**（✅）：4 个 yml 的 `cpus/memory` 及 redis `--maxmemory/--maxmemory-policy`、
   postgres `-c` 调优、Go `GOMEMLIMIT/GOMAXPROCS` 全改为 `${VAR:-<默认>}`。
   **默认 = 历史等价值**（新增旋钮取无害缺省：`GOMEMLIMIT=off`、`GOMAXPROCS=`空、PG `-c` 取 PG 原生默认、
   Redis 使用 compose 缺省 `2gb/noeviction`，未跑 plan-resources.sh 时仍保护 PM 聚合窗口。
   > 注：本次模板化只接管「机制」，Step 0 的「修正默认值」（GOMEMLIMIT 实际取值、PG 调大、redis
   Redis 容量走 resources.env 注入，淘汰策略在所有部署路径固定为 `noeviction`。
   ```yaml
   # 例：docker-compose.app.yml
   app:
     deploy: { resources: { limits: { cpus: "${APP_CPUS:-2}", memory: "${APP_MEM:-1g}" } } }
     environment:
       GOMEMLIMIT: "${APP_GOMEMLIMIT:-900MiB}"
       GOMAXPROCS: "${APP_GOMAXPROCS:-2}"
   # 例：docker-compose.infra.yml
   redis:
     command: redis-server --appendonly yes --maxmemory ${REDIS_MAXMEMORY:-4gb}
              --maxmemory-policy noeviction
              --no-appendfsync-on-rewrite yes --save ""
   postgres:
     command: ["postgres","-c","max_connections=${PG_MAX_CONNECTIONS:-200}",
               "-c","shared_buffers=${PG_SHARED_BUFFERS:-512MB}",
               "-c","effective_cache_size=${PG_EFFECTIVE_CACHE_SIZE:-1536MB}", ...]
   ```
2. **install.sh 接线**：compose 命令加 `--env-file .env --env-file resources.env`；
   precheck 检出无 `resources.env` 时提示先跑 plan-resources.sh（或回退到字面量默认）。
3. **svc.sh 接线（需求⑥的执行面）**：`svc.sh` 是日常 start/restart 的入口，必须**同样**
   消费 `resources.env`，否则改了文件用 `svc.sh restart` 不生效。当前 `svc.sh` 靠 compose
   **自动加载 `./.env`**（未显式传 `--env-file`），而 compose **只按名自动加载 `.env`、绝不自动
   加载 `resources.env`**；且一旦显式传任一 `--env-file`，`.env` 的自动加载即停止——所以必须
   **两个都显式传**。在 `DC` 数组拼接处（[svc.sh:85-97](svc.sh)）插入：
   ```bash
   ENV_FILES=()
   [ -f .env ]          && ENV_FILES+=( --env-file .env )
   [ -f resources.env ] && ENV_FILES+=( --env-file resources.env )
   # --env-file 是顶层 flag，须在子命令前；放进 DC（-p 之后、-f 之前/后均可）
   DC=( $COMPOSE -p "$COMPOSE_PROJECT" "${ENV_FILES[@]}" "${COMPOSE_FILES[@]}" )
   ```
   这样 `svc.sh start` / `svc.sh restart`（内部 `up -d --force-recreate`）/ `svc.sh up`
   都会按 `resources.env` 的新限额重建容器。无 `resources.env` 时退化为今天的行为（仅 `.env`）。
4. **升级存活**（✅，实现略有调整）：`resources.env` 是 operator 独有、**不随交付包**的独立文件，
   故用**整文件继承**而非 `ENV_PRESERVE_KEYS` 键级合并（后者只作用于 `.env`）。install.sh 在升级时
   快照上一版 `resources.env`（或 `etc/resources.env.saved` 兜底）→ 拷入新 release 的 `deploy/` →
   切 current 软链后再落 `etc/resources.env.saved`，与 `.env.saved` 完全对称。
5. **生效命令（需求 ⑥）**：改完 `resources.env` 后任选其一——
   ```bash
   bash svc.sh restart                  # 推荐：按 depends_on 有序重建，经健康门控
   # 或手工：
   docker compose -p omcgo --env-file .env --env-file resources.env -f ... up -d
   ```

### 与本方案**正交**、建议另立任务（代码改动，不在资源脚本范围）

- worker：`internal/pm`/`internal/mr` 加 `errgroup.SetLimit` + JetStream consumer 显式 `MaxAckPending`，
  并约束单文件解析体积上限。
- task：`taskDetailTTL`/`cwmpMappingTTL` 24h→2-4h（治 Redis 数据集增长的真因）。
- 监控告警：`redis used_memory>70%`、`pg numbackends>0.8×max_connections`、容器 `OOMKilled` 计数。
- 数据盘：已支持通过 `.env` 的五个 `*_DATA_PATH` 把有状态服务 bind-mount 到独立数据盘；
  迁移步骤见下节。MinIO 桶生命周期过期仍需另行配置。
- 300k+ 规模：引入 pgbouncer（transaction pooling）作为 acs/worker 加副本的前置条件。

### 有状态服务拆盘与迁移

可独立配置：

| 服务 | `.env` 键 | 容器目录 |
|---|---|---|
| PostgreSQL | `POSTGRES_DATA_PATH` | `/var/lib/postgresql/data` |
| TimescaleDB | `TSDB_DATA_PATH` | `/var/lib/postgresql/data` |
| Redis | `REDIS_DATA_PATH` | `/data` |
| NATS JetStream | `NATS_DATA_PATH` | `/data` |
| MinIO | `MINIO_DATA_PATH` | `/data` |

键留空时继续使用 `pgdata/tsdbdata/redisdata/natsdata/miniodata` 命名卷，保证升级不隐式
切换数据。`plan-resources.sh` 按可用空间选择最大的本地持久文件系统，只补空值并提示人工
拆盘；它不复制数据。建议 NVMe A 放 TimescaleDB、NVMe B 放主库、SSD/NVMe C 放 MinIO，
Redis/NATS 放剩余低延迟设备。只有一块 SSD/NVMe 时可先全部迁入，寻道等待会下降，但五个
服务仍会争用同一设备。

存量迁移必须停服逐项执行，以下以主库为例，其他组件只替换卷名和目标目录：

```bash
cd /opt/omc/current/deploy
bash svc.sh stop
SRC="$(docker volume inspect -f '{{.Mountpoint}}' omcgo_pgdata)"
DEST=/mnt/nvme-b/omc-data/postgres
install -d "$DEST"
rsync -aHAX --numeric-ids "$SRC"/ "$DEST"/
du -sb "$SRC" "$DEST"
# 编辑 .env：POSTGRES_DATA_PATH=/mnt/nvme-b/omc-data/postgres
docker compose -p omcgo --env-file .env --env-file resources.env \
  -f docker-compose.infra.yml -f docker-compose.app.yml config >/dev/null
bash svc.sh up
docker compose -p omcgo --env-file .env --env-file resources.env \
  -f docker-compose.infra.yml -f docker-compose.app.yml ps
```

验收数据库 `pg_isready`、Redis `PING`、NATS `/healthz`、MinIO `/minio/health/live` 和业务
健康接口后，旧卷/旧目录至少保留一个观察周期。回滚时停服、清空对应 `*_DATA_PATH` 或改回
旧路径，再启动。禁止在容器写入期间直接复制 PostgreSQL/WAL、Redis AOF 或 JetStream。

当前压测旋转盘读取等待约 154 ms、末段 iowait 约 61%。SSD/NVMe 一般可把介质等待降到低
毫秒级，独立设备还能降低队列深度和 checkpoint、PM COPY、MinIO 上传、AOF、JetStream
之间的相互阻塞；实际收益必须在真实硬件迁移后复测，不承诺固定倍数。

---

## 7. 验证记录（本机 dry-run）

| 场景 | 结果 |
|------|------|
| 8c/16g, `--skip-monitoring` | 恰好贴下限（Σ13.0/预算13.0），PG 饱和告警正确触发 |
| 32c/32g 近空闲 | medium 档，主库/TimescaleDB/worker CPU=10/16/8，内存按权重分配 |
| 64c/128g | large 档，主库/TimescaleDB/worker CPU=21/32/16，内存封顶 ceiling + 多机拓扑告警 |
| 11g 可用的繁忙主机 | 命中门禁，die + 建议 ≥23 GiB + 逃生口 |

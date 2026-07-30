# PM 聚合与资源根治验收记录（2026-07-30）

## 结论

- 代码、前后端测试、真实 PostgreSQL/TimescaleDB 事务验证均通过。
- 发布包 `100.0.0-20260730-1509` 已构建完成，未部署。
- 目标服务器 `172.24.224.197` 只有 32 CPU、32100 MiB 内存。完整根治配置的业务容器上限合计 35840 MiB，监控固定上限 4224 MiB，再保留 4096 MiB 操作系统内存后，至少需要 44160 MiB。
- 因此资源安全门禁保持阻断：未更新线上版本、未修改线上资源、未启用 Redis v2 双阶段开关，也未声称完成 20000 KPI 线上验收。

## 代码验证

基线提交：`ecf74662e`

通过项目标准验证：

- `cd omcgo && go build ./... && go test ./...`
- `cd omcmb && npm run typecheck`
- 资源环境验证：53/53
- 存储规划验证：19/19
- storage compose 验证：152/152
- 配置升级验证
- 资源计划指标验证

发布构建产物：

```text
deployments/release/archive/project/100.0.0-20260730-1509/
  omc-test-100.0.0-20260730-1509-amd64.tar.xz  (105 MiB)
```

## 真实数据库验证

在服务器临时数据库中完成以下验证，完成后已删除临时数据库和测试角色：

1. `pm_aggregation_windows` 迁移可在失败后安全重入；并发索引最终条件为：
   `(granularity, window_end, task_version_id, entity_key, window_start)
   WHERE status IN ('open','failed')`。
2. 两个独立 PostgreSQL 连接使用 `FOR UPDATE SKIP LOCKED` 时，第一连接锁定
   `first`，第二连接获取 `second`，没有重复认领。
3. 真实 pgx + TimescaleDB hypertable 验证 `ReplaceWindowResults`：
   revision 1 写入 `K900010006`、`K900010076`；revision 2 更新前者并删除后者的
   旧结果，测试通过。

## cAdvisor 兼容性验证

发现 `ghcr.io/google/cadvisor:v0.55.1` 标签不存在，实际官方标签为
`ghcr.io/google/cadvisor:0.55.1`，已同步修正 release 配置、compose 默认值和测试。

在 Docker 29.5 服务器用与发布 compose 相同挂载启动临时 cAdvisor 0.55.1，
app、acs、worker、postgres、postgres-tsdb、redis、nats、minio、web 九个服务均能
采集以下三类序列，并带 Compose service 标签：

- `container_spec_cpu_quota`
- `container_spec_cpu_period`
- `container_spec_memory_limit_bytes`

服务器 Docker 根目录是 `/home/docker-data`，旧 compose 只挂载
`/var/lib/docker`，因此个别 overlayfs 文件系统统计仍有告警，但不影响本次资源
CPU/内存限制采集。临时验证容器已删除。

## 当前线上三层资源证据

线上仍运行 `100.0.0-20260729-2322`。现有 `resources.env` 只有：

```text
REDIS_CPUS=2
REDIS_MEM=5g
REDIS_MAXMEMORY=4gb
```

Docker inspect 的实际限制：

| 服务 | CPU | 内存 |
| --- | ---: | ---: |
| app | 2 | 1536 MiB |
| acs | 5 | 4096 MiB |
| worker | 8 | 1024 MiB |
| postgres | 10 | 7168 MiB |
| postgres-tsdb | 16 | 7168 MiB |
| redis | 2 | 5120 MiB |
| nats | 1 | 1024 MiB |
| minio | 4 | 4096 MiB |
| web | 1 | 512 MiB |

Redis 运行参数：`maxmemory=4 GiB`、`maxmemory-policy=noeviction`、
`appendonly=yes`、`appendfsync=everysec`。

PostgreSQL：`max_connections=300`、`shared_buffers=2293768kB`、
`effective_cache_size=6553608kB`、`work_mem=4096kB`、
`maintenance_work_mem=65536kB`、`max_wal_size=1024MB`。

TimescaleDB：`max_connections=300`、`shared_buffers=2293768kB`、
`effective_cache_size=6553608kB`、`work_mem=16384kB`、
`maintenance_work_mem=262144kB`、`max_wal_size=4096MB`。

## 资源 dry-run

在不安装新版本的情况下执行资源规划 dry-run。服务器当时内存约 31.3 GiB，
安全业务预算为 17.1 GiB；规划下限（含监控）为 31.1 GiB，规划器返回 FATAL。
固定根治方案则需要：

- 九个业务服务：35840 MiB
- 监控服务：4224 MiB
- 操作系统保留：4096 MiB
- 总计：44160 MiB

## 两套执行方案

### A：根治方案（推荐）

“至少 40 GiB”必须理解为 OMC 可分配预算，不是物理内存。物理内存推荐升级到
48 GiB 或更高，然后重新运行规划器：

| 服务 | CPU | 内存 | 进程内约束 |
| --- | ---: | ---: | --- |
| app | 2 | 1536 MiB | GOMAXPROCS=2，GOMEMLIMIT=1382 MiB |
| acs | 5 | 4096 MiB | GOMAXPROCS=5，GOMEMLIMIT=3686 MiB |
| worker | 8 | 2048 MiB | GOMAXPROCS=8，GOMEMLIMIT=1843 MiB |
| postgres | 10 | 7168 MiB | shared_buffers=1792MB，effective_cache_size=5017MB，work_mem=8MB |
| postgres-tsdb | 16 | 7168 MiB | shared_buffers=1792MB，effective_cache_size=5017MB，work_mem=8MB |
| redis | 2 | 8192 MiB | maxmemory=6144mb，noeviction |
| nats | 1 | 1024 MiB | max_store=256 MiB |
| minio | 4 | 4096 MiB | — |
| web | 1 | 512 MiB | — |

升级后按“完整资源文件校验 → dry-run → 第一阶段部署（Redis v2 关闭）→ 健康和
旧队列归零验证 → 第二阶段开启 v2 → 20000 KPI、小时/天数据和至少 3 小时稳定性”
执行。

### B：保持当前限制的分阶段过渡（仅临时）

把 Docker inspect 的当前实际限制固化成完整 schema v2 `resources.env`，第一阶段
只部署代码并保持 Redis v2 关闭；确认旧 worker 队列归零和数据库迁移成功后，再
小流量启用 v2。

风险：

- 当前容器上限加监控仍超过物理内存，无法通过根治门禁，存在宿主机竞争/OOM 风险。
- worker 只有 1 GiB，低于目标 2 GiB。
- Redis 已使用约 4 GiB，且 `maxmemory` 也是 4 GiB，没有迁移和双读所需余量；
  `noeviction` 下会直接拒绝写入。
- B 只能用于用户明确接受风险后的短期过渡，不能作为根治完成依据，也不能据此
  提交“可上线”结论。

## 未完成项

由于资源门禁失败，以下线上验收尚未开始：

- 最新包部署与 Redis v2 两阶段切换
- 20000 KPI 完整处理时长
- 小时、天聚合新鲜度与覆盖率
- 3 小时稳定性观察及当日窗口验收


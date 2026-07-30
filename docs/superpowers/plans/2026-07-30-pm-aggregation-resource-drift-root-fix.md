# PM Aggregation And Resource Drift Root Fix Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 根治 20,000 设备场景下小时/天 KPI 聚合延迟、Redis 状态膨胀和资源配置漂移，使全网小时结果稳定在窗口结束后 30 分钟内发布，并确保发布包声明、Compose 解析值、容器 cgroup 与进程运行值一致。

**Architecture:** 资源侧把 `resources.env` 从“可缺字段的覆盖文件”升级为带版本、必填键和跨字段约束的完整资源契约，部署前验证、部署后核对。聚合侧使用可领取的有界关闭任务、批量结果替换、按时间桶合并的迟到重算和 Redis v2 紧凑状态；版本关闭时以最新有效区间重新校准期望槽位。

**Tech Stack:** Go、pgx/Squirrel、PostgreSQL 16/TimescaleDB、Redis 7/Lua、NATS JetStream、Docker Compose、Bash、Prometheus。

## Global Constraints

- 小时关闭宽限固定为 12 分钟，不恢复 5 分钟关闭。
- Dashboard 只每 5 分钟定时刷新，不增加事件即时刷新。
- 首页继续读取现有 eNB、gNB、GSM 小时/天/周预聚合结果，不回退原始 PM 扫描。
- Redis 保持 AOF everysec 和 `maxmemory-policy=noeviction`；禁止用淘汰聚合状态掩盖容量问题。
- 迟到数据必须修订结果，但同一窗口的迟到事件必须合并，不能每个事件触发一次全量重算。
- 任务版本不可变；自然周期完整性与版本片段完整性继续分别记录。
- SQL 使用 Squirrel + pgx，不引入 ORM。
- 所有性能验收必须基于 20,000 设备真实负载，不能只用单元测试推断生产吞吐。
- 实施前先更新本地主线；确认已包含线上版本 `100.0.0-20260729-2322` 所使用的迟到重算实现。若本地主线仍不含提交 `2f38691eb` 的等价代码，先完成主线同步，不在旧代码上平行实现。

---

### Task 1: 建立完整资源契约并阻止残缺配置上线

**Files:**
- Create: `deployments/release/bundle/deploy/resource-env-lib.sh`
- Create: `deployments/release/bundle/deploy/resource-env-lib_test.sh`
- Modify: `deployments/release/bundle/deploy/plan-resources.sh`
- Modify: `deployments/release/bundle/deploy/install.sh`
- Modify: `deployments/release/bundle/deploy/svc.sh`
- Modify: `deployments/release/bundle/deploy/healthcheck.sh`
- Modify: `deployments/release/bundle/deploy/storage-compose_test.sh`
- Modify: `deployments/release/bundle/deploy/README.md`
- Modify: `deployments/release/bundle/deploy/RESOURCE-PLANNING.md`

**Interfaces:**
- Produces: `resource_env_validate FILE`，成功返回 0；缺字段、单位非法或跨字段约束不满足时返回非 0。
- Produces: `resource_env_required_keys`，输出当前资源契约的全部必填键。
- Produces: `OMC_RESOURCE_SCHEMA_VERSION=2`、`OMC_RESOURCE_PLAN_HOST_CPU`、`OMC_RESOURCE_PLAN_HOST_MEM_MIB`。
- Consumes: `install.sh`、`svc.sh` 中现有 `.env` 和 `resources.env` 加载流程。

- [ ] **Step 1: 写残缺资源文件必须失败的测试**

在 `resource-env-lib_test.sh` 构造当前线上同形文件：

```bash
cat >"$TMP/partial.env" <<'EOF'
REDIS_CPUS=2
REDIS_MEM=5g
REDIS_MAXMEMORY=4gb
EOF

if resource_env_validate "$TMP/partial.env"; then
  bad "仅含 Redis 的 resources.env 不得被识别为完整资源规划"
fi
```

同时覆盖以下约束：

```text
WORKER_GOMEMLIMIT < WORKER_MEM
ACS_GOMEMLIMIT < ACS_MEM
APP_GOMEMLIMIT < APP_MEM
REDIS_MAXMEMORY <= REDIS_MEM - 1GiB
PG_MAX_CONNECTIONS >= 180
TSDB_MAX_CONNECTIONS >= 180
CPU 和内存值必须为正数且单位可解析
```

- [ ] **Step 2: 运行测试并确认失败**

Run:

```bash
bash deployments/release/bundle/deploy/resource-env-lib_test.sh
```

Expected: FAIL，因为验证库尚不存在。

- [ ] **Step 3: 实现资源契约验证库**

`resource-env-lib.sh` 必须要求以下键成组出现：

```text
APP_CPUS APP_MEM APP_GOMEMLIMIT APP_GOMAXPROCS
ACS_CPUS ACS_MEM ACS_GOMEMLIMIT ACS_GOMAXPROCS
WORKER_CPUS WORKER_MEM WORKER_GOMEMLIMIT WORKER_GOMAXPROCS
POSTGRES_CPUS POSTGRES_MEM PG_SHARED_BUFFERS PG_EFFECTIVE_CACHE_SIZE
PG_MAX_CONNECTIONS PG_WORK_MEM PG_MAINTENANCE_WORK_MEM PG_MAX_WAL_SIZE
TSDB_CPUS TSDB_MEM TSDB_SHARED_BUFFERS TSDB_EFFECTIVE_CACHE_SIZE
TSDB_MAX_CONNECTIONS TSDB_WORK_MEM TSDB_MAINTENANCE_WORK_MEM TSDB_MAX_WAL_SIZE
REDIS_CPUS REDIS_MEM REDIS_MAXMEMORY
NATS_CPUS NATS_MEM NATS_MAX_MEMORY_STORE
MINIO_CPUS MINIO_MEM WEB_CPUS WEB_MEM
OMC_RESOURCE_SCHEMA_VERSION OMC_RESOURCE_PLAN_HOST_CPU OMC_RESOURCE_PLAN_HOST_MEM_MIB
```

验证错误必须列出具体缺失键，不能只打印“配置非法”。

- [ ] **Step 4: 让规划器生成契约元数据**

`plan-resources.sh` 输出：

```bash
OMC_RESOURCE_SCHEMA_VERSION=2
OMC_RESOURCE_PLAN_HOST_CPU=$HOST_CPU
OMC_RESOURCE_PLAN_HOST_MEM_MIB=$MEM_TOTAL_MIB
```

写文件后立即调用 `resource_env_validate "$OUT_FILE"`；验证失败时删除新生成的无效文件并返回非 0。

- [ ] **Step 5: 阻止 install/svc 静默接受残缺文件**

`install.sh` 和 `svc.sh` 在组装 Compose 命令前执行：

```bash
if [ -f resources.env ]; then
  resource_env_validate resources.env ||
    die "resources.env 不是完整资源规划；请重新运行 plan-resources.sh，禁止缺失项静默回退 Compose 默认值"
fi
```

升级继承旧文件后也必须验证。当前仅 3 行 Redis 的历史文件应使安装预检失败，而不是继续上线。

- [ ] **Step 6: 增加部署后实际值核对**

`healthcheck.sh` 对 app、ACS、Worker、主库、TSDB、Redis、NATS、MinIO、Web 核对：

```text
resources.env 声明值
docker compose config 解析值
docker inspect HostConfig.NanoCpus/Memory
Go 服务 go_sched_gomaxprocs_threads
Redis CONFIG GET maxmemory/maxmemory-policy
PostgreSQL SHOW shared_buffers/work_mem/max_connections
```

任一层不一致即失败，并打印服务名、期望值、实际值。

- [ ] **Step 7: 运行部署脚本测试**

Run:

```bash
bash deployments/release/bundle/deploy/resource-env-lib_test.sh
bash deployments/release/bundle/deploy/plan-resources-storage_test.sh
bash deployments/release/bundle/deploy/storage-compose_test.sh
bash deployments/release/bundle/deploy/config-upgrade-lib_test.sh
```

Expected: PASS。

- [ ] **Step 8: 提交**

```bash
git add deployments/release/bundle/deploy
git commit -m "fix(deploy): 阻止残缺资源规划静默上线"
```

---

### Task 2: 校准任务版本关闭后的期望槽位

**Files:**
- Modify: `omcgo/internal/pm/stream/rollup_event.go`
- Modify: `omcgo/internal/pm/stream/rollup_event_test.go`
- Modify: `omcgo/internal/pm/stream/finalizer.go`
- Modify: `omcgo/internal/pm/stream/finalize_values_test.go`
- Modify: `omcgo/internal/pm/stream/window_repository.go`
- Modify: `omcgo/internal/pm/stream/window_repository_test.go`

**Interfaces:**
- Produces: `expectedSlotsForFinalization(key WindowKey, version *TaskVersionSnapshot, accumulated int64, location *time.Location) int64`。
- Consumes: 当前快照中的 `EffectiveFrom`、`EffectiveTo` 和自然周期边界。

- [ ] **Step 1: 写“版本后来关闭”回归测试**

复现线上场景：日窗口 08:00–次日 08:00，版本在 13:00 首次出现，当时 `EffectiveTo=nil`，Redis 已记录 `expected_slots=19`；随后版本在次日 00:00 关闭。Finalize 时必须得到：

```go
require.EqualValues(t, 11, expected)
require.EqualValues(t, 11, result.VersionExpectedSlots)
require.False(t, result.PeriodComplete)
```

- [ ] **Step 2: 运行定向测试并确认失败**

Run:

```bash
cd omcgo
go test ./internal/pm/stream -run 'VersionBoundary|ExpectedSlotsForFinalization' -count=1
```

Expected: FAIL，当前沿用 Redis 中历史的 19。

- [ ] **Step 3: 在 Finalize 时按最新版本区间重新计算**

规则任务的小时→天、天→周/月槽位使用当前快照重新裁剪：

```go
expected := expectedVersionChildWindows(parent, sourceGranularity, version, location)
if expected > 0 {
    state.ExpectedSlots = expected
}
```

设备流水线仍使用自然周期槽位，不套规则版本裁剪。

- [ ] **Step 4: 同步窗口审计字段**

Finalize claim 必须把校准后的 `expected_slots`、`version_effective_from`、`version_effective_to` 一次写回窗口表，保证窗口与结果表口径相同。

- [ ] **Step 5: 运行测试**

Run:

```bash
cd omcgo
go test ./internal/pm/stream -count=1
```

Expected: PASS；11 小时版本片段不再显示 11/19。

- [ ] **Step 6: 提交**

```bash
git add omcgo/internal/pm/stream
git commit -m "fix(pm): 按版本有效区间校准聚合槽位"
```

---

### Task 3: 将窗口关闭改为可领取的有界调度

**Files:**
- Create: `omcgo/migrations/tsdb/000009_pm_finalize_claims.sql`
- Modify: `omcgo/internal/pm/stream/window_repository.go`
- Modify: `omcgo/internal/pm/stream/window_repository_test.go`
- Modify: `omcgo/internal/pm/stream/consumer.go`
- Create: `omcgo/internal/pm/stream/finalize_scheduler_test.go`
- Modify: `omcgo/internal/pm/stream/metrics.go`

**Interfaces:**
- Produces: `ClaimDue(ctx, granularity, dueBefore, limit, leaseOwner, leaseUntil) ([]WindowRecord, error)`。
- Produces: `CompleteClaim`、`ReleaseClaim`。
- Consumes: 每个粒度独立的关闭宽限和 `FinalizeConcurrency`。

- [ ] **Step 1: 写公平性和不重复领取测试**

测试必须证明：

```text
两个 scanner 不能领取同一窗口
08 点仍有 5,000 个设备窗口时，09 点到期窗口仍能取得调度份额
小时任务不会被天级重算长期饿死
进程退出后过期 lease 可重新领取
```

- [ ] **Step 2: 运行测试并确认失败**

Run:

```bash
cd omcgo
go test ./internal/pm/stream -run 'ClaimDue|FinalizeScheduler' -count=1
```

Expected: FAIL。

- [ ] **Step 3: 增加领取字段和索引**

迁移增加：

```sql
ALTER TABLE pm_aggregation_windows
  ADD COLUMN finalize_lease_owner uuid,
  ADD COLUMN finalize_lease_until timestamptz;

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_pm_windows_due_claim
ON pm_aggregation_windows (granularity, window_end, task_version_id, entity_key, window_start)
WHERE status IN ('open', 'failed');
```

- [ ] **Step 4: 实现 SKIP LOCKED 领取**

在一个短事务中使用 `FOR UPDATE SKIP LOCKED` 领取并写 lease。不得像当前 scanner 一样每 30 秒重新分页扫描全部 open/failed 窗口。

- [ ] **Step 5: 实现按粒度公平调度**

使用一个固定 Worker 池，轮询配额：

```text
hourly device: 70%
hourly network/product/group: 15%
daily/weekly/monthly: 15%
```

当某类无任务时，配额可借给其他类。调度循环不能等待整页 200 个任务全部结束后才领取下一页。

- [ ] **Step 6: 增加调度指标**

至少增加：

```text
omc_pm_aggregation_finalize_claims
omc_pm_aggregation_finalize_inflight
omc_pm_aggregation_finalize_oldest_due_seconds
omc_pm_aggregation_finalize_claim_conflicts_total
```

- [ ] **Step 7: 运行测试**

Run:

```bash
cd omcgo
go test ./internal/pm/stream -count=1
go test ./cmd/worker/... -count=1
```

Expected: PASS。

- [ ] **Step 8: 提交**

```bash
git add omcgo/internal/pm/stream omcgo/migrations/tsdb
git commit -m "perf(pm): 改为有界公平窗口关闭调度"
```

---

### Task 4: 消除结果全量 DELETE + INSERT 写放大

**Files:**
- Create: `omcgo/internal/pm/stream/result_repository.go`
- Create: `omcgo/internal/pm/stream/result_repository_test.go`
- Modify: `omcgo/internal/pm/stream/finalizer.go`
- Modify: `omcgo/internal/pm/stream/rebuild_test.go`

**Interfaces:**
- Produces: `ReplaceWindowResults(ctx, tx, key, revision, metrics, completeness) (int, error)`。
- Consumes: immutable task-version metric definitions and pgx transaction.

- [ ] **Step 1: 写首次发布不得 DELETE 的测试**

使用 pgx mock/SQL contract 断言：

```text
revision=1 的首次发布只执行批量 upsert，不执行窗口级 DELETE
revision>1 时只删除 staging 中不存在的旧 metric_id
同一 metric_id 更新 revision/value/completeness
```

- [ ] **Step 2: 运行测试并确认失败**

Run:

```bash
cd omcgo
go test ./internal/pm/stream -run 'ReplaceWindowResults|Finalize' -count=1
```

Expected: FAIL，现实现每次 Finalize 都删除整个窗口结果。

- [ ] **Step 3: 使用临时 staging + CopyFrom**

每个 Finalize 事务：

```text
1. 创建/清空 session 临时 staging 表。
2. pgx.CopyFrom 写入本窗口全部结果。
3. 一条 INSERT ... ON CONFLICT DO UPDATE 合并。
4. 仅 revision>1 时执行 DELETE ... WHERE NOT EXISTS(staging matching metric_id)。
5. 更新窗口 published 状态。
```

禁止逐结果单条 INSERT，禁止首次发布 DELETE。

- [ ] **Step 4: 保证修订语义**

任务版本不可变，因此正常重算指标集合应稳定；如果公式不完整导致某指标本次没有值，必须在 staging 中写入 `metric_value=NULL/complete=false` 或由 stale-delete 明确删除，不能保留旧值。

- [ ] **Step 5: 运行测试和基准**

Run:

```bash
cd omcgo
go test ./internal/pm/stream -count=1
go test ./internal/pm/stream -run '^$' -bench 'Finalize|ReplaceWindowResults' -benchmem
```

Expected: PASS；20,000 设备等价批次 SQL 次数不再随指标数线性增长。

- [ ] **Step 6: 提交**

```bash
git add omcgo/internal/pm/stream
git commit -m "perf(pm): 批量替换聚合窗口结果"
```

---

### Task 5: 合并迟到重算并复用周期快照扫描

**Files:**
- Modify: `omcgo/internal/pm/stream/rebuild.go`
- Modify: `omcgo/internal/pm/stream/rebuild_test.go`
- Modify: `omcgo/internal/pm/stream/rollup_outbox.go`
- Modify: `omcgo/internal/pm/stream/rollup_outbox_query_test.go`
- Modify: `omcgo/internal/pm/stream/metrics.go`

**Interfaces:**
- Produces: `ClaimRebuildBatch(ctx, quietBefore, limit) ([]RebuildJob, error)`。
- Produces: `VisitSnapshotsForPeriod` 每个 `(sourceVersionIDs, granularity, start, end)` 只扫描一次并分派给同批窗口。
- Consumes: `request_generation` 和迟到事件最后更新时间。

- [ ] **Step 1: 写重算合并测试**

测试同一小时 100 个迟到事件：

```text
quiet period 内只增加 request_generation，不启动重算
静默 2 分钟后只执行一次源快照扫描
重算期间又到事件时，完成后最多再执行一次
父级日窗口只在子小时批次稳定后入队一次
```

- [ ] **Step 2: 运行测试并确认失败**

Run:

```bash
cd omcgo
go test ./internal/pm/stream -run 'Rebuild.*Coalesce|VisitSnapshots.*Batch' -count=1
```

Expected: FAIL。

- [ ] **Step 3: 增加 quiet-period 领取条件**

`claimNext` 改为批量领取，并要求：

```sql
requested_at <= now() - interval '2 minutes'
```

同一窗口继续通过唯一键合并；运行中的 generation 变化只安排下一轮，不立即并发重算。

- [ ] **Step 4: 按周期批量扫描**

把同一小时、同一源版本集合的 rebuild jobs 分组，一次顺序读取 `pm_aggregation_counter_rollups`，在内存中只保留当前 payload，并分派给匹配窗口；不得为每个网络/产品/组窗口重复读取约 20,000 个 JSON payload。

- [ ] **Step 5: 增加重算成本指标**

```text
omc_pm_aggregation_rebuild_batches_total
omc_pm_aggregation_rebuild_jobs_per_batch
omc_pm_aggregation_rebuild_snapshot_rows_total
omc_pm_aggregation_rebuild_snapshot_scan_seconds
omc_pm_aggregation_rebuild_coalesced_total
```

- [ ] **Step 6: 运行测试**

Run:

```bash
cd omcgo
go test ./internal/pm/stream -count=1
```

Expected: PASS。

- [ ] **Step 7: 提交**

```bash
git add omcgo/internal/pm/stream
git commit -m "perf(pm): 合并迟到窗口批量重算"
```

---

### Task 6: 将 Redis 聚合状态升级为 v2 紧凑编码并清理已发布残留

**Files:**
- Modify: `omcgo/internal/pm/stream/accumulate.lua`
- Modify: `omcgo/internal/pm/stream/redis_store.go`
- Modify: `omcgo/internal/pm/stream/redis_store_test.go`
- Modify: `omcgo/internal/pm/stream/snapshot.go`
- Create: `omcgo/internal/pm/stream/redis_sweeper.go`
- Create: `omcgo/internal/pm/stream/redis_sweeper_test.go`
- Modify: `omcgo/internal/pm/stream/metrics.go`
- Modify: `omcgo/cmd/worker/pm_streaming.go`

**Interfaces:**
- Produces: accumulator v2 值只保存 `metric identity + sum/count/min/max`，不在每个实体、每个粒度重复保存完整 base64 JSON definition。
- Produces: v1/v2 双读、v2 单写。
- Produces: `SweepPublishedState(ctx, scanLimit, unlinkBatch) (int64, error)`。

- [ ] **Step 1: 写空间回归测试**

构造 20,000 设备、146 指标、小时/天/周/月活动窗口模型，断言：

```text
v2 单 accumulator 编码长度 <= v1 的 45%
定义元数据每个 task version 只保存一次
设备 identity 每窗口只保存一次
读取 v1 后可正确迁移为 v2
```

- [ ] **Step 2: 运行测试并确认失败**

Run:

```bash
cd omcgo
go test ./internal/pm/stream -run 'CompactAccumulatorV2|RedisMemoryModel' -count=1
```

Expected: FAIL。

- [ ] **Step 3: 实现 v2 编码**

Lua 热路径只做数值增量；Go 侧通过 immutable task snapshot 还原指标定义。设备维度元数据从窗口 entity 和设备快照恢复，不复制到每个指标值中。

- [ ] **Step 4: 使用 UNLINK 删除大窗口**

Finalize 成功后以 `UNLINK` 异步释放 seen/slots/meta/entities/chunks/acc 等键，避免 `DEL` 在 Redis 主线程同步回收大对象。

- [ ] **Step 5: 增加安全残留清理器**

Sweeper 只处理：

```text
Redis meta 对应 DB 窗口已经 published
没有活动 finalize/rebuild lock
发布时间早于安全阈值
每轮 SCAN 和 UNLINK 均有固定上限
```

不得根据 TTL 或 key 前缀直接删除未核对状态。

- [ ] **Step 6: 增加内存可观测性**

暴露按粒度采样的活动窗口数、键数、估算字节数、sweeper 删除数和 Redis 写失败数。采样不得执行无节制全库 `--bigkeys`。

- [ ] **Step 7: 运行测试**

Run:

```bash
cd omcgo
go test ./internal/pm/stream -count=1
go test ./cmd/worker/... -count=1
```

Expected: PASS；模型估算在 20,000 设备四粒度活动窗口下低于 2.5GiB。

- [ ] **Step 8: 提交**

```bash
git add omcgo/internal/pm/stream omcgo/cmd/worker
git commit -m "perf(pm): 压缩并回收 Redis 聚合状态"
```

---

### Task 7: 补齐资源漂移、关闭延迟和数据库成本告警

**Files:**
- Modify: `deployments/monitoring/alerts/infra-alerts.yml`
- Modify: `deployments/monitoring/alerts/omc-rules.yml`
- Modify: `deployments/monitoring/alerts/host-container-alerts.yml`
- Modify: `deployments/monitoring/grafana/dashboards/omc-overview.json`
- Modify: `deployments/release/bundle/deploy/storage-compose_test.sh`

**Interfaces:**
- Consumes: Tasks 1、3、5、6 新增指标。
- Produces: 资源配置漂移、Redis 水位、小时关闭尾延迟、重算扫描成本告警。

- [ ] **Step 1: 写规则契约测试**

要求存在：

```text
OMCResourcePlanDrift
RedisAggregationMemoryHigh warning >= 80%
RedisAggregationMemoryCritical critical >= 90%
PMFinalizeOldestDueHigh warning > 30m
PMFinalizeOldestDueCritical critical > 60m
PMRebuildSnapshotScanSlow warning p95 > 10s
PMDailyVersionExpectedSlotsMismatch
```

- [ ] **Step 2: 实现规则和 Dashboard 面板**

Dashboard 同时展示：

```text
声明 CPU/内存、容器实际 limit、实际 usage
Redis maxmemory、dataset bytes、按粒度活动状态估算
小时 finalize inflight/oldest due
重算 batch 大小、扫描行数和耗时
TSDB result replace、rollup snapshot 查询耗时
```

- [ ] **Step 3: 验证规则**

Run:

```bash
docker run --rm \
  -v "$PWD/deployments/monitoring/alerts:/rules:ro" \
  prom/prometheus:v2.51.0 promtool check rules /rules/*.yml
bash deployments/release/bundle/deploy/storage-compose_test.sh
```

Expected: PASS。

- [ ] **Step 4: 提交**

```bash
git add deployments/monitoring deployments/release/bundle/deploy
git commit -m "feat(monitoring): 监控聚合延迟与资源配置漂移"
```

---

### Task 8: 统一测试、生成完整资源方案、部署和 20,000 设备验收

**Files:**
- Create: `docs/superpowers/evidence/2026-07-30-pm-aggregation-resource-root-fix-validation.md`
- Modify only if verification exposes a defect directly caused by Tasks 1–7.

**Interfaces:**
- Consumes: Tasks 1–7 全部提交。
- Produces: 新发布包、完整 `resources.env`、线上验证证据和 MR。

- [ ] **Step 1: 运行代码级验证**

Run:

```bash
cd omcgo
go build ./...
go test ./...
cd ../omcmb
npm run typecheck
cd ..
bash deployments/release/bundle/deploy/resource-env-lib_test.sh
bash deployments/release/bundle/deploy/plan-resources-storage_test.sh
bash deployments/release/bundle/deploy/storage-compose_test.sh
```

Expected: 全部 PASS。

- [ ] **Step 2: 制定本机明确资源清单**

CPU 暂保持已经验证并实际生效的值：

```text
app=2, acs=5, worker=8, postgres=10, postgres-tsdb=16,
redis=2, nats=1, minio=4, web=1
```

第一轮保护性内存清单：

```text
app=1536MiB, acs=4GiB, worker=2GiB,
postgres=7GiB, postgres-tsdb=7GiB,
redis container=8GiB/maxmemory=6GiB,
nats=1GiB, minio=4GiB, web=512MiB
```

其中 Redis 增容只是上线保护；根治验收仍要求 v2 状态在稳定负载下低于 2.5GiB。资源文件必须包含 Task 1 的全部必填键，禁止继续使用三行 Redis 文件。

- [ ] **Step 3: 部署前 dry-run 和配置核对**

停止应用写入后运行规划器，避免把正在运行的 OMC 自身内存再次从 MemAvailable 扣除。对生成值进行人工核对，再写入上述明确清单。

Run:

```bash
bash deploy/plan-resources.sh --dry-run --assume-dedicated
bash deploy/resource-env-lib_test.sh
```

Expected: 完整资源契约验证通过。

- [ ] **Step 4: 部署并核对三层值**

使用 `install.sh`/`svc.sh` 部署后运行 `healthcheck.sh`。必须保存：

```text
resources.env 声明值
Compose config 解析值
docker inspect cgroup 值
Go GOMAXPROCS
Redis maxmemory/policy
PostgreSQL/TSDB 内部内存参数
```

Expected: 三层完全一致，所有服务 healthy，无 OOM/restart。

- [ ] **Step 5: 执行 20,000 设备小时/天验收**

至少连续观察 3 个小时窗口和一次日窗口关闭。验收门槛：

```text
NATS pending/ack_pending/redelivery = 0
HTTP 503 = 0
宿主机稳态 CPU idle >= 20%，IOwait < 10%
Redis 稳态 dataset < 2.5GiB，且活动窗口进入平台期
小时窗口 T+12m 开始关闭，全网结果最迟 T+30m 发布
连续三个小时不得出现延迟递增
日结果最迟自然日结束后 30m 发布
K900010006、K900010076 小时和天结果均存在
版本片段 expected slots 与 EffectiveFrom/EffectiveTo 一致
无重复全周期快照扫描；单次重算快照扫描 p95 < 10s
pm_aggregation_counter_rollups 查询 p95 < 10s、max < 30s
聚合 finalize/rebuild/outbox errors = 0
```

- [ ] **Step 6: 做结果一致性抽样**

对 eNB、gNB、GSM 各抽样设备及 network/product/group：

```text
新旧小时 KPI 值一致，浮点允许误差 <= 1e-9
迟到事件到达后 revision 增加且结果只修订一次
自然周期完整与版本片段完整标记符合槽位证据
Dashboard 五分钟刷新后读取到已发布小时/天结果
```

- [ ] **Step 7: 记录证据并提交 MR**

验证文档必须记录命令时间、镜像版本、资源三层值、窗口时间、发布耗时、KPI 值、Redis/TSDB/队列曲线和所有未满足项。任何 P0 门槛失败都不得创建“可合入”MR。

```bash
git add docs/superpowers/evidence/2026-07-30-pm-aggregation-resource-root-fix-validation.md
git commit -m "docs(qa): 记录聚合与资源根治验收"
git push
glab mr create
```

Expected: MR 包含完整代码、部署契约和生产验收证据。

---

## Self-Review

- Spec coverage: 包含资源配置漂移、CPU/内存实际值、Redis 容量、TSDB 写放大、关闭调度、迟到重算、版本槽位、小时/天 KPI、告警、部署和 MR。
- Placeholder scan: 每个实现任务均包含目标接口、失败测试、实现和验证命令，没有待补内容。
- Type consistency: `resource_env_validate`、`ClaimDue`、`ReplaceWindowResults`、`ClaimRebuildBatch` 和 v2 accumulator 在其首次出现处定义，后续引用一致。
- Safety: 计划阶段不修改服务器；部署阶段先验证完整资源契约，保留 noeviction、12 分钟关闭和 Dashboard 线上保护。

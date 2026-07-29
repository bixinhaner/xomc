# OMC Performance Root-Cause Hardening Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 根治 PM 上传背压交叉锁定、Redis 聚合窗口内存膨胀和 TimescaleDB 影子维表长查询，并在清空的服务器环境完成部署验证。

**Architecture:** ACS 使用磁盘、I/O、队列三个独立迟滞 latch，最终背压取 OR；Redis 把每个指标的 definition 与 sum/count/min/max 合并为单字段紧凑值，并在写入时原子迁移旧格式；TSDB staging 快照在 prune 前建立主键索引并 ANALYZE。部署资源和监控规则与这些运行时不变量同步。

**Tech Stack:** Go 1.24、go-redis/Lua、pgx/PostgreSQL 16/TimescaleDB、Prometheus rules、Bash、Docker Compose、React/TypeScript。

## Global Constraints

- 小时、天、周、月聚合语义和已有结果契约不得改变。
- Redis 必须保持 AOF everysec 和 `maxmemory-policy=noeviction`。
- PM 上传磁盘硬保护不得取消；CPU load 继续只监控。
- Dashboard 只每五分钟定时刷新，不恢复事件即时刷新。
- Dashboard 查询超时、并发限制、慢查询和资源告警必须保留。
- 所有用户可见前端文案走国际化；本计划不新增用户可见文案。
- SQL 继续使用 pgx/Squirrel 既有模式，不引入 ORM。
- 服务器清理只覆盖 `com.docker.compose.project=omcgo` 的资源和经挂载确认的 OMC 日志。

---

### Task 1: 清空服务器 OMC 环境

**Files:**
- No repository files.

**Interfaces:**
- Consumes: `/opt/omc/current/deploy` 下四个 compose 文件和 `omcgo` 项目标签。
- Produces: 不含 OMC 容器、命名卷和运行日志的干净服务器；保留发布程序和非 OMC 项目。

- [ ] **Step 1: 记录删除前目标**

Run:

```bash
docker ps -a --filter label=com.docker.compose.project=omcgo
docker volume ls --filter label=com.docker.compose.project=omcgo
docker inspect omcgo-app-1 omcgo-acs-1 omcgo-worker-1 omcgo-web-1
```

Expected: 所有待删除资源都带 `omcgo` 项目标识；日志 bind mount 的 source 位于 OMC 发布目录。

- [ ] **Step 2: 停止服务并删除 compose 卷**

Run from `/opt/omc/current/deploy`:

```bash
docker compose -p omcgo \
  -f docker-compose.infra.yml \
  -f docker-compose.app.yml \
  -f docker-compose.web.yml \
  -f docker-compose.monitoring.yml \
  down -v --remove-orphans
```

Expected: OMC 容器、网络和十个命名卷被删除。

- [ ] **Step 3: 删除已确认的 OMC 运行日志**

仅对 Step 1 输出中明确属于 OMC 的日志 source 目录执行删除；不使用通配符，不删除
`/opt/omc/current`、`/opt/omc/shared` 根目录或任何非日志目录。

- [ ] **Step 4: 验证清理结果**

Run:

```bash
docker ps -a --filter label=com.docker.compose.project=omcgo
docker volume ls --filter label=com.docker.compose.project=omcgo
```

Expected: 两个列表均无 OMC 资源；非 OMC 容器和文件保持不变。

---

### Task 2: 实现背压信号独立迟滞

**Files:**
- Modify: `omcgo/internal/acs/upload/backpressure.go`
- Modify: `omcgo/internal/acs/upload/backpressure_test.go`

**Interfaces:**
- Consumes: `BackpressureConfig`、`QueueSignal` 和 watchdog 周期采样值。
- Produces: `pressureState` 位状态及 `decideBackpressureState(previous, diskPct, ioPct, queue, cfg) (pressureState, BackpressureDecision)`。

- [ ] **Step 1: 写队列恢复不受磁盘中间区间阻塞的失败测试**

Add a test equivalent to:

```go
func TestBackpressureStateQueueRecoveryIgnoresUnlatchedDiskNeutralBand(t *testing.T) {
    cfg := BackpressureConfig{
        Enabled: true, DiskHighPct: 70, DiskLowPct: 60,
        IOSomeHighPct: 70, IOSomeLowPct: 20,
        QueuePendingHigh: 2000, QueuePendingLow: 500,
        QueueOldestHigh: 10 * time.Minute, QueueOldestLow: 2 * time.Minute,
    }
    engaged, _ := decideBackpressureState(0, 68, 0, QueueSignal{
        Configured: true, Available: true, RatesAvailable: true,
        Stats: event.QueueStats{Pending: 2000},
    }, cfg)
    require.True(t, engaged.has(pressureQueue))

    recovered, decision := decideBackpressureState(engaged, 68, 0, QueueSignal{
        Configured: true, Available: true, RatesAvailable: true,
        Stats: event.QueueStats{Pending: 0},
        Rates: QueueRates{PendingPerSecond: 0},
    }, cfg)
    require.Zero(t, recovered)
    require.False(t, decision.Active)
    require.Equal(t, pressureReasonRecovered, decision.Reason)
}
```

- [ ] **Step 2: 运行失败测试**

Run:

```bash
cd omcgo && go test ./internal/acs/upload -run 'TestBackpressureState' -count=1
```

Expected: FAIL because `pressureState` and `decideBackpressureState` do not exist.

- [ ] **Step 3: 实现三个独立 latch**

Add:

```go
type pressureState uint32

const (
    pressureDisk pressureState = 1 << iota
    pressureIO
    pressureQueue
)

func (s pressureState) has(flag pressureState) bool { return s&flag != 0 }
```

Implement `decideBackpressureState` so disk/I/O independently cross high/low thresholds, queue independently
crosses pending/oldest/slope thresholds, and disabled config clears all bits. Preserve startup fail-open and
an already-latched queue on a failed queue sample.

Replace `rememberedPressure atomic.Bool` with `pressure atomic.Uint32`; watchdog loads/stores the bit state and
sets `active` from `nextState != 0`.

- [ ] **Step 4: 补齐状态组合测试**

Cover:

- disk latch remains active after queue recovers;
- queue latch releases while disk is in an unlatched neutral band;
- disk and I/O unavailable signals retain only their already-confirmed latch semantics from the design;
- queue failure starts fail-open but preserves an existing queue latch;
- disabled immediately opens the gate but retains confirmed latches until a fresh low-watermark sample proves recovery.

- [ ] **Step 5: 运行包测试**

Run:

```bash
cd omcgo && go test ./internal/acs/upload -count=1
```

Expected: PASS.

- [ ] **Step 6: 提交**

```bash
git add omcgo/internal/acs/upload/backpressure.go omcgo/internal/acs/upload/backpressure_test.go
git commit -m "fix(acs): 解耦上传背压信号迟滞状态"
```

---

### Task 3: 压缩 Redis 聚合窗口状态并兼容旧格式

**Files:**
- Modify: `omcgo/internal/pm/stream/accumulate.lua`
- Modify: `omcgo/internal/pm/stream/redis_store.go`
- Modify: `omcgo/internal/pm/stream/redis_store_test.go`

**Interfaces:**
- Consumes: `ContributionValue` 和旧 `defs[id]`、`acc[id|sum/count/min/max]`。
- Produces: `acc[id] = v1|definition|sum|count|min|max`；`Read` 同时支持新旧格式。

- [ ] **Step 1: 写紧凑字段数量失败测试**

After accumulating two metrics, assert:

```go
fields, err := server.HKeys(keys.acc[0])
require.NoError(t, err)
require.Len(t, fields, 2)
require.False(t, server.Exists(keys.defs[0]))
for _, field := range fields {
    require.NotContains(t, field, "|sum")
    require.NotContains(t, field, "|count")
}
```

- [ ] **Step 2: 写旧格式读取和迁移失败测试**

Seed one legacy definition and four legacy numeric fields, call `Read`, then call `Accumulate` for the same
definition. Assert:

- the returned sum/count/min/max include legacy and new contributions exactly once;
- `acc[id]` starts with `v1|`;
- legacy numeric fields and `defs[id]` have been removed.

- [ ] **Step 3: 运行失败测试**

Run:

```bash
cd omcgo && go test ./internal/pm/stream -run 'TestRedisWindowStore.*(Compact|Legacy)' -count=1
```

Expected: FAIL because current format creates five fields.

- [ ] **Step 4: 添加 Go 编解码器**

Implement:

```go
type compactAccumulator struct {
    Definition ContributionValue
    Sum        float64
    Count      int64
    Min        float64
    Max        float64
}

func encodeCompactAccumulator(item compactAccumulator) (string, error)
func decodeCompactAccumulator(raw string) (compactAccumulator, error)
```

Use prefix `v1|`, Raw URL Base64 definition, `strconv.FormatFloat(..., 'g', -1, 64)` and
`strconv.FormatInt`. Reject the wrong field count, unknown version, invalid Base64 or invalid number with
definition/window context.

- [ ] **Step 5: 修改 Lua 原子累加和旧格式迁移**

For each definition ID:

1. `HGET acc id`;
2. when compact value exists, parse `v1|definition|sum|count|min|max`;
3. otherwise read legacy `defs[id]` and four legacy accumulator fields;
4. merge the current delta;
5. `HSET acc id compact_value`;
6. `HDEL` the four old fields and `HDEL defs id`;
7. expire `acc`; expire/delete `defs` only while it still contains legacy fields.

Keep duplicate slot handling before metric accumulation so re-delivery never double-counts.

- [ ] **Step 6: 修改 Read 双格式扫描**

During `HSCAN acc`:

- fields without legacy suffix decode directly as compact values;
- legacy fields retain the existing grouped read path and fetch definitions from `defs`;
- duplicate IDs prefer compact form because a completed migration deletes old fields.

Return the same `WindowState.Accumulators`.

- [ ] **Step 7: 运行 Redis 聚合测试**

Run:

```bash
cd omcgo && go test ./internal/pm/stream -run 'TestRedisWindowStore' -count=1
```

Expected: PASS.

- [ ] **Step 8: 运行完整 stream 包测试**

Run:

```bash
cd omcgo && go test ./internal/pm/stream -count=1
```

Expected: PASS.

- [ ] **Step 9: 提交**

```bash
git add omcgo/internal/pm/stream/accumulate.lua omcgo/internal/pm/stream/redis_store.go omcgo/internal/pm/stream/redis_store_test.go
git commit -m "fix(pm): 压缩 Redis 聚合窗口状态"
```

---

### Task 4: 为 TSDB staging prune 建立索引

**Files:**
- Modify: `omcgo/internal/tsdbsync/runner.go`
- Modify: `omcgo/internal/tsdbsync/runner_sql_test.go`

**Interfaces:**
- Consumes: `dstTable`、`stageTable`、`cols`、`keyCols`。
- Produces: staging unique-index SQL、ANALYZE SQL、使用主键等值匹配的 prune SQL。

- [ ] **Step 1: 写 SQL 失败测试**

Extend composite-key assertions:

```go
indexSQL := buildStageIndexSQL("sync_stage_device_group_member_dim", []string{"group_id", "device_id"})
assert.Contains(t, indexSQL, `CREATE UNIQUE INDEX`)
assert.Contains(t, indexSQL, `("group_id", "device_id")`)
assert.Contains(t, prune, `d."group_id" = s."group_id"`)
assert.Contains(t, prune, `d."device_id" = s."device_id"`)
assert.NotContains(t, prune, "IS NOT DISTINCT FROM")
```

- [ ] **Step 2: 运行失败测试**

Run:

```bash
cd omcgo && go test ./internal/tsdbsync -run 'TestBuildMirrorMergeSQL' -count=1
```

Expected: FAIL because staging index SQL is absent and prune uses `IS NOT DISTINCT FROM`.

- [ ] **Step 3: 实现 staging 索引和统计**

After CopyFrom and before merge:

```go
if _, err := tx.Exec(ctx, buildStageIndexSQL(stageTable, keyCols)); err != nil {
    return 0, fmt.Errorf("index staging for %s: %w", dstTable, err)
}
if _, err := tx.Exec(ctx, "ANALYZE "+pgx.Identifier{stageTable}.Sanitize()); err != nil {
    return 0, fmt.Errorf("analyze staging for %s: %w", dstTable, err)
}
```

Generate a transaction-local unique index name from the sanitized stage table. Build prune predicates with `=`
because primary-key columns are non-null.

- [ ] **Step 4: 为逐表日志增加耗时**

Measure inside `runTable` and log `table`, `rows`, `took` on success and `took` on failure.

- [ ] **Step 5: 运行 tsdbsync 测试**

Run:

```bash
cd omcgo && go test ./internal/tsdbsync -count=1
```

Expected: PASS.

- [ ] **Step 6: 提交**

```bash
git add omcgo/internal/tsdbsync/runner.go omcgo/internal/tsdbsync/runner_sql_test.go
git commit -m "perf(tsdb): 索引影子维表同步快照"
```

---

### Task 5: 对齐 Redis 资源规划和监控规则

**Files:**
- Modify: `deployments/docker/docker-compose.yml`
- Modify: `deployments/docker/plan-resources.sh`
- Modify: `deployments/docker/RESOURCE-PLANNING.md`
- Modify: `deployments/release/bundle/deploy/docker-compose.infra.yml`
- Modify: `deployments/release/bundle/deploy/plan-resources.sh`
- Modify: `deployments/release/bundle/deploy/RESOURCE-PLANNING.md`
- Modify: `deployments/release/bundle/deploy/storage-compose_test.sh`
- Modify: `deployments/monitoring/alerts/infra-alerts.yml`
- Modify: `deployments/monitoring/alerts/omc-rules.yml`

**Interfaces:**
- Consumes: `REDIS_MEM`。
- Produces: `REDIS_MAXMEMORY <= REDIS_MEM - COW reserve`、固定 `noeviction`、准确告警。

- [ ] **Step 1: 写部署契约失败断言**

Add `storage-compose_test.sh` checks for:

```bash
contains "release Redis 禁止淘汰聚合状态" '--maxmemory-policy noeviction' "$RELEASE_COMPOSE"
contains "开发 Redis 禁止淘汰聚合状态" '--maxmemory-policy noeviction' "$DEV_COMPOSE"
contains "队列样本陈旧只检查 ACS" 'deployment_unit="acs"' "$OMC_ALERTS"
contains "Redis 上限告警说明 noeviction" 'noeviction' "$INFRA_ALERTS"
```

Add a planner test that generated `resources.env` never contains `allkeys-lru` or `volatile-lru`, and that
`REDIS_MAXMEMORY < REDIS_MEM`.

- [ ] **Step 2: 运行部署测试并确认失败**

Run:

```bash
bash deployments/release/bundle/deploy/storage-compose_test.sh
```

Expected: FAIL on stale policy/alert assumptions.

- [ ] **Step 3: 修正资源规划**

- Keep compose policy fixed at `noeviction`.
- Remove the development planner's 512MiB `maxmemory` cap and “Redis only uses a few MB” comments.
- Derive maxmemory from the same Redis container budget, preserving at least 256MiB COW reserve in development
  and 1GiB in release.
- Make planner output and resource documents say `noeviction`.
- Do not emit a contradictory `REDIS_MAXMEMORY_POLICY`.

- [ ] **Step 4: 修正告警**

- Restrict `OMCPMQueueSampleStale` to `deployment_unit="acs"`.
- Update `RedisMemoryHigh` annotations: at the limit, `noeviction` rejects writes rather than evicting state.
- Add a critical Redis near-limit threshold at 95% with a shorter hold.
- Add/strengthen a PM aggregation processing failure alert using
  `increase(omc_pm_aggregation_events_failed_total[5m]) > 0`.

- [ ] **Step 5: 运行 shell 与规则检查**

Run:

```bash
bash deployments/release/bundle/deploy/storage-compose_test.sh
docker run --rm -v "$PWD/deployments/monitoring/alerts:/rules:ro" prom/prometheus:v2.51.0 \
  promtool check rules /rules/*.yml
```

Expected: all shell assertions and Prometheus rule files pass.

- [ ] **Step 6: 提交**

```bash
git add deployments/docker deployments/release/bundle/deploy deployments/monitoring/alerts
git commit -m "fix(deploy): 对齐聚合 Redis 容量与告警"
```

---

### Task 6: 统一构建和测试

**Files:**
- Modify only if tests reveal a defect directly caused by Tasks 2–5.

**Interfaces:**
- Consumes: all implementation commits.
- Produces: a release candidate commit with no known regression.

- [ ] **Step 1: 格式化**

Run:

```bash
gofmt -w omcgo/internal/acs/upload/backpressure.go \
  omcgo/internal/acs/upload/backpressure_test.go \
  omcgo/internal/pm/stream/redis_store.go \
  omcgo/internal/pm/stream/redis_store_test.go \
  omcgo/internal/tsdbsync/runner.go \
  omcgo/internal/tsdbsync/runner_sql_test.go
```

- [ ] **Step 2: Go build**

Run:

```bash
cd omcgo && go build ./...
```

Expected: exit 0.

- [ ] **Step 3: Go full test**

Run with host networking permission because miniredis/httptest bind local ports:

```bash
cd omcgo && go test ./...
```

Expected: PASS.

- [ ] **Step 4: Frontend typecheck**

Run:

```bash
cd omcmb && npm run typecheck
```

Expected: PASS.

- [ ] **Step 5: Deployment shell tests and diff checks**

Run:

```bash
bash deployments/release/bundle/deploy/storage-compose_test.sh
bash deployments/release/bundle/deploy/plan-resources-storage_test.sh
bash deployments/release/build-release-bash3_test.sh
git diff --check
```

Expected: PASS.

---

### Task 7: 构建发布包并部署空环境

**Files:**
- Generated release artifacts remain outside Git tracking.

**Interfaces:**
- Consumes: verified branch HEAD.
- Produces: versioned OMC release deployed to `172.24.224.197`.

- [ ] **Step 1: 生成唯一版本并构建发布包**

Use the repository release script with a version containing the current timestamp and branch commit. Verify the
archive checksum before transfer.

- [ ] **Step 2: 上传并安装**

Transfer the release archive to the server's established release staging directory, run the bundled installer,
and keep `/opt/omc/current` pointing to the new version only after installation succeeds.

- [ ] **Step 3: 等待所有服务健康**

Run bundled `healthcheck.sh` and:

```bash
docker compose -p omcgo \
  -f docker-compose.infra.yml \
  -f docker-compose.app.yml \
  -f docker-compose.web.yml \
  -f docker-compose.monitoring.yml ps
```

Expected: healthcheck 25/25 and all long-running services healthy.

---

### Task 8: 线上功能和性能验收

**Files:**
- Create: `docs/operations/validation/2026-07-29-performance-root-cause-hardening-validation.md`

**Interfaces:**
- Consumes: clean deployed environment and monitoring APIs.
- Produces: timestamped validation evidence for the MR.

- [ ] **Step 1: 验证空库和版本**

Record deployed image tags, migration success, initial PostgreSQL/TimescaleDB sizes, Redis DBSIZE, NATS stream
state and MinIO object count.

- [ ] **Step 2: 验证 PM 接入与队列**

Use the repository PM stream integration/e2e tool or controlled test fixture. Record:

- accepted PM upload count;
- ACS backpressure active/rejected deltas;
- NATS pending, ack pending and redelivery;
- worker processing errors.

- [ ] **Step 3: 验证四种聚合粒度**

Create or seed a controlled aggregation task containing hourly, daily, weekly and monthly granularities. Use the
integration test's controllable window timestamps to finalize each granularity without waiting for wall-clock
week/month boundaries. Verify result rows and completeness fields against expected input.

- [ ] **Step 4: 验证 Redis 紧凑状态和容量**

Inspect representative `pmagg:*:acc:*` hashes:

- one field per metric;
- no new `defs` hash;
- no legacy `|sum/count/min/max` fields;
- Redis used/maxmemory remains below warning threshold;
- no `OOM command not allowed` logs.

- [ ] **Step 5: 验证背压独立恢复**

Using the existing unit/integration fixture, reproduce queue-only pressure while disk remains between 60% and
70%. Verify queue latch engages at high watermark and releases after queue low watermark and non-positive slope,
without requiring disk to fall below 60%.

- [ ] **Step 6: 验证 TSDB 同步**

Run at least three shadow-dim sync cycles. Record per-table duration and `pg_stat_activity`; verify
`device_dim`/`device_group_member_dim` prune no longer remains active for 20–30 seconds and no
`TSDBLongQueryActive` alert fires.

- [ ] **Step 7: 验证 Dashboard 和监控**

Verify:

- Dashboard reads existing hourly/daily/weekly results;
- frontend refresh occurs only on the five-minute timer;
- query timeout, concurrency limit and rejection metrics are present;
- no false app/worker `OMCPMQueueSampleStale`;
- active alerts contain no Redis OOM, PM backlog, aggregation failure or long-query alert.

- [ ] **Step 8: 保存验证报告**

Record exact commands, timestamps, inputs, observed values and any environmental limitation in the validation
document.

- [ ] **Step 9: 提交验证证据**

```bash
git add docs/operations/validation/2026-07-29-performance-root-cause-hardening-validation.md
git commit -m "docs(performance): 记录根因治理线上验证"
```

---

### Task 9: 最终审查、推送和 MR

**Files:**
- Modify only for review findings that are in scope.

**Interfaces:**
- Consumes: all passing tests and validation evidence.
- Produces: pushed branch and ready GitLab MR.

- [ ] **Step 1: 最终差异审查**

Run:

```bash
git status --short
git diff --check origin/main...HEAD
git diff --stat origin/main...HEAD
git log --oneline origin/main..HEAD
```

Expected: clean worktree, no whitespace errors, only planned files.

- [ ] **Step 2: 重跑高风险验证**

Run the Go full test, frontend typecheck, deployment shell tests and Prometheus rule checks again from branch HEAD.

- [ ] **Step 3: 推送分支**

```bash
git push -u origin codex/performance-root-cause-hardening
```

- [ ] **Step 4: 创建 ready MR**

Create a non-draft MR targeting `main`. Include:

- incident symptoms and measured root causes;
- code design and compatibility;
- destructive clean-environment test scope;
- test commands and results;
- deployed version and online validation;
- rollback instructions.

- [ ] **Step 5: 确认 MR pipeline**

Query the MR pipeline and report any external failure separately from implementation failures.

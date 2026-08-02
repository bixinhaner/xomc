# Parameter Sync Convergence and PM Redis Isolation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 根治参数同步运行已满足终态条件却长期停留在执行中的问题，将 PM/KPI 热状态从核心 Redis 物理隔离，并消除 PM 小时关窗后 TSDB 写放大，保证 20,000 设备规模下小时结果在关窗后 3 分钟内完成，同时不影响 ACS、设备任务和 Dashboard。

**Architecture:** 参数同步由事件路径和维护路径共同调用一个事务级状态收敛器，维护扫描只处理最老的活跃运行并使用 `FOR UPDATE SKIP LOCKED`；app/worker 建立 `redis-core` 与 `redis-pm` 两个独立客户端，PM 窗口、锁、水位以及 KPI L2 读取和失效通知只走 `redis-pm`，其他业务继续走 `redis-core`；PM 首次发布跳过无意义的旧修订删除，TSDB 使用匹配删除谓词的索引，并在只读一致性快照中直接 keyset 分页，取消整周期临时表物化。

**Tech Stack:** Go 1.x、pgx v5、Squirrel、Redis Universal Client、Prometheus、PostgreSQL/TimescaleDB、Docker Compose、Bash、React/TypeScript（仅回归验证，无 UI 功能改动）。

## Global Constraints

- [ ] 所有工作在分支 `codex/paramsync-pm-redis-isolation` 和隔离 worktree 中完成，不覆盖用户已有改动。
- [ ] 严格 TDD：每个行为先写失败测试，确认失败原因正确，再写最小实现使其通过。
- [ ] 后端遵守 `handler -> service -> repository/model` 分层；SQL 使用 Squirrel + pgx，错误用 `%w` 包装。
- [ ] Dashboard 只读现有 hourly/daily/weekly 聚合结果；不新增 15 分钟粒度，不做事件即时刷新，仍只每 5 分钟刷新。
- [ ] 关窗 grace 保持 12 分钟；验收的是 `window_end + 12m ± 30s` 开始关闭，而不是把处理时间隐藏到 grace 中。
- [ ] 不以清库、丢合法数据或调大资源掩盖状态机、SQL 或队列问题；非法启动期 PM 数据仍按现有策略直接丢弃。
- [ ] Redis 迁移必须可检查、可重跑、保留 TTL、可回滚；迁移期间不允许同时有两个 worker 写 PM 窗口。
- [ ] 不先把 finalize concurrency 从 32 提到 64；只有 SQL 优化后的压测证据证明 TSDB CPU、锁和 I/O 有余量才单独调整。
- [ ] 每个任务提交 Conventional Commit，中文说明；全部验证通过后才 push、创建 MR、合入和部署。
- [ ] 部署后不得立即宣称完成，至少跨越一个完整小时关窗，并核对 hourly/daily/weekly、队列、ACS 503、CPU、内存、磁盘 I/O、数据库锁与慢查询。

---

## Task 1: 建立参数同步统一收敛器

**Files:**

- Create: `omcgo/internal/paramsync/convergence.go`
- Create: `omcgo/internal/paramsync/convergence_test.go`
- Modify: `omcgo/internal/paramsync/result_processor.go`
- Modify: `omcgo/internal/paramsync/result_processor_test.go`
- Modify: `omcgo/internal/paramsync/pg_repository.go`
- Modify: `omcgo/internal/paramsync/pg_repository_integration_test.go`

- [x] **Step 1: 为统一收敛判定写失败单元测试**

  在 `convergence_test.go` 建表驱动测试，至少覆盖：

  ```go
  func TestRunConvergenceDecision(t *testing.T) {
      tests := []struct {
          name string
          run  SyncRun
          want convergenceDecision
      }{
          {
              name: "all terminal and processed succeeds",
              run: SyncRun{Status: RunStatusExecuting, ExpectedTaskCount: 20,
                  TerminalTaskCount: 20, ProcessedTaskCount: 20, FailedTaskCount: 0},
              want: convergenceDecision{Ready: true, Result: RunStatusSucceeded},
          },
          {
              name: "terminal failures converge failed",
              run: SyncRun{Status: RunStatusExecuting, ExpectedTaskCount: 20,
                  TerminalTaskCount: 20, ProcessedTaskCount: 20, FailedTaskCount: 2},
              want: convergenceDecision{Ready: true, Result: RunStatusFailed},
          },
          {
              name: "processed results missing remains active",
              run: SyncRun{Status: RunStatusExecuting, ExpectedTaskCount: 20,
                  TerminalTaskCount: 20, ProcessedTaskCount: 19},
              want: convergenceDecision{},
          },
      }
      // assert exact decision; terminal runs must be no-op.
  }
  ```

- [x] **Step 2: 运行测试确认因收敛器尚不存在而失败**

  Run: `cd omcgo && go test ./internal/paramsync -run 'TestRunConvergenceDecision' -count=1`

  Expected: FAIL，缺少 `convergenceDecision` / `decideRunConvergence`。

- [x] **Step 3: 定义事务级收敛接口和结果**

  在 `convergence.go` 定义：

  ```go
  type convergenceCause string

  const (
      convergenceEvent       convergenceCause = "event"
      convergenceMaintenance convergenceCause = "maintenance"
  )

  type convergenceResult struct {
      Finalized bool
      Status    RunStatus
      Drift     bool
  }

  func convergeRunTx(
      ctx context.Context,
      tx pgx.Tx,
      runID uuid.UUID,
      now time.Time,
      metrics *Metrics,
      cause convergenceCause,
  ) (convergenceResult, error)
  ```

  函数必须在同一事务中：锁定运行、加载 authoritative task/result counts、修正计数、判断 ready、调用现有成功/失败 finalize 逻辑。终态运行幂等返回，不重复记完成指标。

- [x] **Step 4: 将事件处理路径改为调用统一收敛器**

  删除 `Process` 中重复的 `ReadyToFinalize` 分支组合，保留事件落库与幂等判断，然后调用 package-level `convergeRunTx(..., p.now(), p.metrics, convergenceEvent)`。该函数不挂在 `PGResultProcessor` receiver 上，确保 Reconciler 可复用同一实现而不形成接口反向依赖。重投结果不得重复 finalize、不得重复增加 processed 数。

- [x] **Step 5: 为事务与重投行为补集成测试**

  测试至少断言：

  - 20/20/20 在一次事务内变为 succeeded；
  - 含失败任务时变为 failed；
  - 事务回滚后 run、outbox、计数均不部分提交；
  - 同一 result redelivery 不重复完成；
  - 两个并发收敛调用只有一个返回 `Finalized=true`。

- [x] **Step 6: 运行参数同步聚焦测试**

  Run: `cd omcgo && go test ./internal/paramsync -run 'Test(RunConvergence|ResultProcessor|ClaimRunForFinalize)' -count=1`

  Expected: PASS。

- [x] **Step 7: 提交统一收敛器**

  ```bash
  git add omcgo/internal/paramsync
  git commit -m "fix: 统一参数同步运行状态收敛"
  ```

## Task 2: 根治活跃运行扫描饥饿并补齐异常分类

**Files:**

- Modify: `omcgo/internal/paramsync/reconciler.go`
- Modify: `omcgo/internal/paramsync/reconciler_test.go`
- Modify: `omcgo/cmd/app/provider/paramsync.go`
- Modify: `omcgo/cmd/app/provider/paramsync_maintenance_test.go`
- Modify: `omcgo/internal/paramsync/metrics.go`
- Modify: `omcgo/internal/paramsync/metrics_test.go`
- Modify: `deployments/monitoring/alerts/omc-rules.yml`
- Modify: `deployments/monitoring/tests/promql-probes.txt`
- Modify: `deployments/monitoring/grafana/dashboards/omc-overview.json`

- [x] **Step 1: 写 active-only、oldest-first、skip-locked 的失败 SQL 测试**

  ```go
  func TestBuildRunConvergenceCandidateSelectSQL(t *testing.T) {
      query, _, err := buildRunConvergenceCandidateSelect(200).ToSql()
      require.NoError(t, err)
      require.Contains(t, query, "status IN")
      require.Contains(t, query, "ORDER BY started_at")
      require.Contains(t, query, "FOR UPDATE SKIP LOCKED")
      require.Contains(t, query, "LIMIT 200")
      require.NotContains(t, query, "ORDER BY id")
  }
  ```

  同时增加用 UUID 顺序构造“历史旧记录永远占满前 20 条”的回归测试，证明新查询优先拿最老活跃运行。

- [x] **Step 2: 运行测试确认旧的全历史/20 条扫描失败**

  Run: `cd omcgo && go test ./internal/paramsync -run 'TestBuildRunConvergenceCandidateSelectSQL|TestReconcileRunCounts' -count=1`

  Expected: FAIL，旧 SQL 没有 active 过滤、时间排序、行锁，批量仍为 20。

- [x] **Step 3: 用统一收敛器重写维护扫描**

  将批量改为 200。候选状态仅包含 active 状态；优先级为：

  1. 已满足 expected=terminal=processed 但未 finalize；
  2. 计数与 authoritative 状态不一致；
  3. 按 `started_at NULLS FIRST, created_at, id` 从老到新。

  每条候选在事务内调用 `convergeRunTx(..., convergenceMaintenance)`，单条失败不阻塞下一轮；维护函数返回扫描数、漂移数、完成数和最老 active age。

- [x] **Step 4: 区分三类真正未收敛原因**

  在 authoritative 查询中明确输出并记录：

  - `plan_not_dispatched`：expected > 已创建任务数；
  - `terminal_task_missing_result`：terminal task > durable result 数；
  - `device_task_active`：设备任务确实仍处于非终态。

  不通过修改 expected 数“修好”缺任务；前两类进入告警和补偿路径，第三类继续等待现有设备超时/取消机制。

- [x] **Step 5: 接入维护周期并扩展 20 秒预算测试**

  `runParamSyncMaintenance` 继续每 30 秒执行，但 convergence 扫描必须先于低优先级统计维护。测试确保 context deadline 仍为 20 秒、一次调用 batch=200、维护错误只记录不杀 app。

- [x] **Step 6: 增加指标并写注册测试**

  指标采用项目现有前缀风格：

  ```text
  param_sync_runs_ready_but_not_finalized
  param_sync_run_counter_drift
  param_sync_active_run_oldest_age_seconds
  param_sync_reconcile_finalized_total{result}
  param_sync_reconcile_duration_seconds
  param_sync_runs_blocked{reason}
  ```

  `metrics_test.go` 必须 gather registry 并断言名称、label 和一次观测值，避免只定义未注册。

- [x] **Step 7: 增加告警和 Dashboard 运维面板**

  告警阈值：ready 未 finalize 持续 2 分钟为 critical；最老 active >5 分钟且 `reason!="device_task_active"` 为 warning；counter drift >0 持续 2 分钟为 warning。Dashboard 只增加运维指标，不改变首页业务数据查询。

- [x] **Step 8: 运行测试并提交**

  Run:

  ```bash
  cd omcgo && go test ./internal/paramsync ./cmd/app/provider -run 'ParamSync|Reconcile|Metrics' -count=1
  cd .. && promtool check rules deployments/monitoring/alerts/omc-rules.yml
  bash deployments/monitoring/tests/validate-dashboards.sh
  bash deployments/monitoring/tests/validate-queue-governance.sh
  ```

  Expected: PASS；新增查询同时补到 `promql-probes.txt` 供线上逐条探测。

  Commit:

  ```bash
  git add omcgo deployments/monitoring
  git commit -m "fix: 修复参数同步维护扫描饥饿"
  ```

## Task 3: 消除首次 PM 发布的无效 DELETE 和缺失索引

**Files:**

- Modify: `omcgo/internal/pm/stream/rollup_outbox.go`
- Modify: `omcgo/internal/pm/stream/rollup_outbox_query_test.go`
- Modify: `omcgo/migrations/tsdb/000001_tsdb_schema.sql`
- Modify: `omcgo/cmd/worker/pm_streaming.go`

- [x] **Step 1: 写 revision=1 不执行 DELETE 的失败测试**

  使用现有 fake tx 记录执行 SQL：

  ```go
  func TestDeleteStaleRollupRevisionTxSkipsFirstRevision(t *testing.T) {
      tx := newRecordingTx(t)
      err := deleteStaleRollupRevisionTx(ctx, tx, key, 1, []uuid.UUID{uuid.New()})
      require.NoError(t, err)
      require.Empty(t, tx.ExecutedSQL())
  }
  ```

  保留 revision=2 的断言，要求两个表都 DELETE 且含完整谓词。

- [x] **Step 2: 运行测试确认 revision=1 当前仍执行两次 DELETE**

  Run: `cd omcgo && go test ./internal/pm/stream -run 'TestDeleteStaleRollupRevisionTx' -count=1`

  Expected: FAIL，记录到两个 DELETE。

- [x] **Step 3: 最小实现首次发布快速返回**

  `revision <= 1` 直接返回 nil；`revision > 1` 保持 stale cleanup，不改变迟到重算语义。

- [x] **Step 4: 为两个删除谓词增加精确索引**

  在 baseline TSDB schema 增加：

  ```sql
  CREATE INDEX idx_pm_counter_rollups_revision_cleanup
      ON public.pm_aggregation_counter_rollups
      (publication_task_version_id, entity_key, granularity, window_start, revision, event_id);

  CREATE INDEX idx_pm_rollup_outbox_revision_cleanup
      ON public.pm_aggregation_rollup_outbox
      (publication_task_version_id, entity_key, granularity, window_start, revision, event_id);
  ```

  与 `EnsurePeriodRebuildIndex` 同样增加在线升级的 `CREATE INDEX CONCURRENTLY IF NOT EXISTS` 守护；使用独立 advisory lock，不能把 `CONCURRENTLY` 放入事务。

- [x] **Step 5: 增加 SQL 结构和在线升级测试**

  断言索引列顺序完全匹配删除等值前缀、最后覆盖 `event_id`，创建/删除都为 CONCURRENTLY，副本间有 advisory lock。

- [x] **Step 6: 跑 PM 聚焦测试并提交**

  Run: `cd omcgo && go test ./internal/pm/stream ./cmd/worker -run 'Rollup|Revision|Index|Streaming' -count=1`

  Expected: PASS。

  Commit:

  ```bash
  git add omcgo/internal/pm/stream omcgo/migrations/tsdb omcgo/cmd/worker
  git commit -m "perf: 消除PM首次发布无效清理"
  ```

## Task 4: 取消周期重算临时表写放大

**Files:**

- Modify: `omcgo/internal/pm/stream/rollup_outbox.go`
- Modify: `omcgo/internal/pm/stream/rollup_outbox_query_test.go`
- Modify: `omcgo/internal/pm/stream/metrics.go`
- Modify: `omcgo/internal/pm/stream/metrics_test.go`
- Modify: `deployments/monitoring/alerts/omc-rules.yml`
- Modify: `deployments/monitoring/grafana/dashboards/omc-overview.json`

- [x] **Step 1: 写“不物化临时表”的失败测试**

  新测试要求分页 SQL 直接查询 `pm_aggregation_counter_rollups` 并 join published window，禁止：

  ```go
  require.NotContains(t, query, "pm_rebuild_source_snapshot")
  require.NotContains(t, query, "CREATE TEMP TABLE")
  require.Contains(t, query, "pm_aggregation_counter_rollups rollup")
  require.Contains(t, query, "published_window.published_revision = rollup.revision")
  require.Contains(t, query, "published_window.published_revision IS NOT NULL")
  ```

  分页 cursor 仍必须覆盖 `(task_version_id, window_start, entity_key, chunk_index, publication_task_version_id, revision, event_id)`。

- [x] **Step 2: 运行测试确认当前整周期临时表方案失败**

  Run: `cd omcgo && go test ./internal/pm/stream -run 'Test.*Period.*Page|TestVisitSnapshotsForPeriod' -count=1`

  Expected: FAIL，当前 SQL 来源为 `pm_rebuild_source_snapshot` 且执行 TEMP/INDEX/ANALYZE。

- [x] **Step 3: 在只读 Repeatable Read 快照中直接分页**

  `VisitSnapshotsForPeriod` 在专用连接上开启：

  ```go
  pgx.TxOptions{
      IsoLevel:   pgx.RepeatableRead,
      AccessMode: pgx.ReadOnly,
  }
  ```

  所有页面通过该事务查询基表和 `pm_aggregation_windows`。Repeatable Read 的同一 MVCC snapshot 保证分页过程中 `published_revision` 视图不漂移；只读事务不使用 `FOR UPDATE/FOR SHARE`。正常结束 commit，回调/查询失败 rollback。删除 TEMP TABLE、CREATE INDEX、ANALYZE 与 defer DROP。

- [x] **Step 4: 增加一致性、分页和取消测试**

  至少覆盖：

  - 两页边界不丢、不重；
  - 相同 key 不同 revision 只读取 published revision；
  - task version/time range 过滤不扩大；
  - visit callback 失败触发 rollback；
  - context cancel 释放连接；
  - 空结果不产生后续 page query。

- [x] **Step 5: 调整观测指标**

  保留 `omc_pm_aggregation_rebuild_snapshot_scan_seconds` 以兼容告警，新增 `omc_pm_aggregation_rebuild_snapshot_pages_total`；将 help 文案从“temporary snapshot scan”改成“repeatable-read keyset snapshot scan”。Dashboard 同时显示 rows/s、pages/s、scan p95 和 TSDB temp bytes，方便确认写放大消失。

- [x] **Step 6: 跑 PM 全包测试并提交**

  Run: `cd omcgo && go test ./internal/pm/stream -count=1`

  Expected: PASS。

  Commit:

  ```bash
  git add omcgo/internal/pm/stream deployments/monitoring
  git commit -m "perf: 移除PM周期重算临时表写放大"
  ```

## Task 5: 增加双 Redis 配置、连接和路由

**Files:**

- Modify: `omcgo/internal/core/appconfig/config.go`
- Modify: `omcgo/internal/core/appconfig/validate.go`
- Modify: `omcgo/internal/core/appconfig/validate_test.go`
- Modify: `omcgo/internal/core/components/infra.go`
- Modify: `omcgo/internal/core/components/redis/pool_metrics.go`
- Modify: `omcgo/internal/core/components/redis/pool_metrics_test.go`
- Modify: `omcgo/cmd/app/bootstrap.go`
- Modify: `omcgo/cmd/app/provider/dictload.go`
- Modify: `omcgo/cmd/app/provider/pm.go`
- Modify: `omcgo/cmd/app/etc/config.dev.yaml`
- Modify: `omcgo/cmd/app/etc/config.test.yaml`
- Modify: `omcgo/cmd/app/etc/config.local.yaml`
- Modify: `omcgo/cmd/app/etc/config.prod.yaml`
- Modify: `omcgo/cmd/worker/bootstrap.go`
- Modify: `omcgo/cmd/worker/main.go`
- Modify: `omcgo/cmd/worker/pm_streaming.go`
- Modify: `omcgo/cmd/worker/etc/config.dev.yaml`
- Modify: `omcgo/cmd/worker/etc/config.test.yaml`
- Modify: `omcgo/cmd/worker/etc/config.local.yaml`
- Modify: `omcgo/cmd/worker/etc/config.prod.yaml`
- Create: `omcgo/cmd/worker/redis_routing_test.go`

- [x] **Step 1: 写配置 fallback 和生产隔离失败测试**

  ```go
  func TestWorkerPMRedisFallback(t *testing.T) {
      cfg := WorkerConfig{Redis: RedisConfig{Addrs: []string{"redis:6379"}}}
      require.Equal(t, cfg.Redis, cfg.EffectivePMRedis())
  }

  func TestProductionRejectsSharedPMRedis(t *testing.T) {
      t.Setenv("OMCGO_ENV", "prod")
      cfg := validProdWorkerConfig()
      cfg.Redis.Addrs = []string{"redis-core:6379"}
      cfg.PMRedis.Addrs = []string{"redis-core:6379"}
      require.ErrorContains(t, cfg.Validate(), "pm_redis must be physically isolated")
  }
  ```

  dev/test/local 允许 PMRedis 空配置 fallback，生产显式配置时必须与 core 地址集合不同；不得用不同 DB index 冒充物理隔离。

- [x] **Step 2: 写组件路由失败测试**

  使用两个 miniredis 实例：向 PM streaming 写入一条窗口后只允许 `pmagg:*` 出现在 PM Redis；app 和 worker 初始化 KPI Router L2、执行一次版本失效后，只允许 `kpi-route:*` 出现在 PM Redis，且 app 的 version bump 可使 worker 的条目失效；task queue、alarm、product/parammodel、device cache 测试仍只在 core Redis。

- [x] **Step 3: 运行测试确认 AppConfig/WorkerConfig 的 PMRedis 尚不存在**

  Run: `cd omcgo && go test ./internal/core/appconfig ./cmd/worker -run 'PMRedis|RedisRouting' -count=1`

  Expected: FAIL，缺少 `PMRedis`、fallback 和路由。

- [x] **Step 4: 增加配置和第二客户端生命周期**

  在 `AppConfig` 和 `WorkerConfig` 增加：

  ```go
  PMRedis RedisConfig `mapstructure:"pm_redis"`
  ```

  增加 `EffectivePMRedis()`；`appInfra` 和 `workerInfra` 均增加 `PMRedis redis.UniversalClient`。连接、health registration、shutdown hook 与 pool metrics 均复用 core components，但健康检查名称和指标带 `role="core|pm"`，避免第二次注册冲突。

- [x] **Step 5: 精确切换 PM/KPI 调用点**

  改为 `PMRedis`：

  - `NewRedisWindowStore` 以及 finalizer/recovery/rebuilder/sweeper 共享的窗口存储；
  - app 和 worker 的 KPI Router Redis L2 cache；
  - app dict loader/indicator management 的 KPI Router version bumper。

  保持 `Redis`：

  - ACS session/admission；
  - command/task queue；
  - device/common cache；
  - alarm、connection request、在线状态等非 PM 业务。

- [x] **Step 6: 更新 app/worker 配置样例**

  app/worker prod:

  ```yaml
  redis:
    addrs: ["redis-core:6379"]
  pm_redis:
    addrs: ["redis-pm:6379"]
  ```

  dev/local/test 可显式配置两个地址以覆盖真实路径；兼容旧配置的 fallback 只作为滚动升级过渡。

- [x] **Step 7: 跑配置、组件、app、worker 测试并提交**

  Run:

  ```bash
  cd omcgo
  go test ./internal/core/appconfig ./internal/core/components/... ./cmd/app/... ./cmd/worker -count=1
  ```

  Expected: PASS。

  Commit:

  ```bash
  git add omcgo
  git commit -m "feat: 隔离PM与核心Redis客户端"
  ```

## Task 6: 部署两个物理 Redis 并联动资源规划

**Files:**

- Modify: `deployments/docker/docker-compose.yml`
- Modify: `deployments/docker/plan-resources.sh`
- Modify: `deployments/release/bundle/deploy/docker-compose.infra.yml`
- Modify: `deployments/release/bundle/deploy/docker-compose.app.yml`
- Modify: `deployments/release/bundle/deploy/plan-resources.sh`
- Modify: `deployments/release/bundle/deploy/resource-env-lib.sh`
- Modify: `deployments/release/bundle/deploy/storage-paths-lib.sh`
- Modify: `deployments/release/bundle/deploy/healthcheck.sh`
- Modify: `deployments/release/bundle/deploy/install.sh`
- Modify: `deployments/release/build-release.sh`
- Modify: `deployments/release/bundle/deploy/RESOURCE-PLANNING.md`
- Modify: `deployments/docker/README.md`
- Modify: `deployments/release/bundle/deploy/resource-env-lib_test.sh`
- Modify: `deployments/release/bundle/deploy/storage-paths-lib_test.sh`
- Modify: `deployments/release/bundle/deploy/plan-resources-storage_test.sh`
- Modify: `deployments/release/bundle/deploy/storage-compose_test.sh`
- Modify: `deployments/release/bundle/deploy/healthcheck_test.sh`
- Modify: `deployments/release/bundle/deploy/install-resource-preflight_test.sh`
- Modify: `deployments/release/bundle/deploy/resource-plan-metrics_test.sh`

- [x] **Step 1: 先更新 shell/compose 契约测试**

  测试必须要求存在独立 service、volume 和资源键：

  ```text
  redis-core / redis-pm
  rediscoredata / redispmdata
  REDIS_CORE_CPUS / REDIS_CORE_MEM / REDIS_CORE_MAXMEMORY
  REDIS_PM_CPUS / REDIS_PM_MEM / REDIS_PM_MAXMEMORY
  REDIS_DATA_PATH（core 沿用旧键）/ REDIS_PM_DATA_PATH
  OMC_RESOURCE_SCHEMA_VERSION=3
  ```

  断言 `redis-core` 可保留既有宿主 6379 兼容口，`redis-pm` 不发布宿主端口；app/worker 同时 depends_on 两个 healthy Redis，其他服务只依赖 core。

- [x] **Step 2: 运行部署测试确认单 Redis 契约失败**

  Run:

  ```bash
  bash deployments/release/bundle/deploy/resource-env-lib_test.sh
  bash deployments/release/bundle/deploy/storage-compose_test.sh
  bash deployments/release/bundle/deploy/plan-resources-storage_test.sh
  ```

  Expected: FAIL，缺少双实例、schema v3 和 PM 资源键。

- [x] **Step 3: 修改 Compose**

  生产默认：

  - `redis-core`: 2 CPU、4 GiB container、3 GiB maxmemory、AOF everysec、noeviction；
  - `redis-pm`: 2 CPU、8 GiB container、6 GiB maxmemory、AOF everysec、noeviction；
  - 两者独立 data path/volume、healthcheck、stop grace；core 沿用 `REDIS_DATA_PATH` 保证已有数据路径不漂移，PM 新增 `REDIS_PM_DATA_PATH`；
  - `redis-core` 同时提供网络别名 `redis-core` 和兼容别名 `redis`，避免未同时升级的 ACS 配置失联；
  - core AOF rewrite 仍留至少 1 GiB；PM 留 2 GiB 以吸收双窗口和 rewrite COW；
  - app/worker 配置环境能解析 `redis-core:6379` / `redis-pm:6379`。

- [x] **Step 4: 修改资源规划算法和完整性门禁**

  把原 `redis` 组件拆成 `redis-core` 与 `redis-pm`，两者分别进入 floor/ceiling/weight、CPU 列表、总内存预算和输出表。`resource-env-lib.sh` schema 升到 3，分别校验：

  - core maxmemory <= core mem - 1 GiB；
  - PM maxmemory <= PM mem - 2 GiB；
  - 所有键唯一、正数、继承旧 resources.env 时旧 schema 明确拒绝并提示重跑 planner。

- [x] **Step 5: 更新健康检查和资源计划指标**

  `healthcheck.sh` 分别核对两个实例的 `PING`、AOF、policy、实际 maxmemory、Docker CPU/memory limit，并验证 app/worker 配置的两个 endpoint 不同。资源计划 Prometheus 指标为两个 component 输出，不继续把总量记成一个 Redis。

- [x] **Step 6: 更新部署文档和升级提示**

  文档明确：这是物理隔离，不是 Redis DB index；升级前必须重新运行 planner；PM Redis 无宿主端口；回滚时旧 core `pmagg:*` 暂时保留直到窗口 TTL/验收期结束。

- [x] **Step 7: 跑全部部署脚本测试并提交**

  Run:

  ```bash
  for test in deployments/release/bundle/deploy/*_test.sh; do bash "$test"; done
  ```

  Expected: 全部 PASS。

  Commit:

  ```bash
  git add deployments
  git commit -m "feat: 部署独立PM Redis实例"
  ```

## Task 7: 提供可验证、保留 TTL 的 PM Redis 迁移工具

**Files:**

- Create: `omcgo/cmd/omcctl/pm_redis.go`
- Create: `omcgo/cmd/omcctl/pm_redis_test.go`
- Modify: `omcgo/cmd/omcctl/main.go`
- Modify: `deployments/release/bundle/deploy/docker-compose.app.yml`
- Modify: `deployments/release/bundle/deploy/README.md`

- [x] **Step 1: 写迁移算法失败测试**

  用可注入的 Redis command fake 覆盖任意二进制 DUMP/RESTORE、冲突和批处理语义；再用两个 miniredis 对 string key 做端到端覆盖。miniredis 只完整支持 string 的 DUMP/RESTORE，因此 hash/set/zset/list/stream 必须在 Task 8 的真实 Redis Compose 集成烟测中验证，不能伪称单测已覆盖：

  - 只复制权威的 `pmagg:*`；可重建的 `kpi-route:*` 不迁移，切换后按 miss 从 DB 重建；
  - fake 断言所有类型统一走不解析 payload 的 DUMP/RESTORE；真实 Redis 烟测断言 string/hash/set/zset/list/stream 原样迁移；
  - 正 TTL 保留，persistent key 仍 persistent；
  - destination 已存在且 checksum 相同视为幂等；
  - checksum 不同默认失败，只有显式 `--replace` 才替换；
  - source 在复制中 key 消失记 skipped，不中断全批；
  - `--dry-run` 不写目标；
  - 扫描结果输出 scanned/copied/skipped/conflict/failed 和源/目标 key 数。

- [x] **Step 2: 运行测试确认命令不存在**

  Run: `cd omcgo && go test ./cmd/omcctl -run 'TestPMRedis' -count=1`

  Expected: FAIL，缺少 `newPMRedisCmd` / migrator。

- [x] **Step 3: 实现 `omcctl pm-redis migrate`**

  命令参数：

  ```text
  --source redis-core:6379
  --target redis-pm:6379
  --pattern 'pmagg:*'
  --scan-count 500
  --pipeline-size 100
  --dry-run
  --replace
  ```

  使用 Redis `SCAN` + pipelined `PTTL`/`DUMP` + target `RESTORE`；TTL 小于等于 0 时按 Redis 语义区分永久 key 与已消失 key。日志不得输出 value、密码或 DUMP 内容。迁移结束再次 SCAN 比较 key count，并随机/全量（规模允许时）比较 `DUMP` hash 与 TTL 容差。

- [x] **Step 4: 将 omcctl 放进 worker 运维入口并写 runbook**

  升级步骤固定为：

  1. 等待前一小时窗口 published；
  2. 停 worker，确认无 PM 消费者写入；
  3. 启动并健康检查 `redis-pm`；
  4. dry-run；
  5. migrate；
  6. verify key/hash/TTL；
  7. 以双 Redis 配置启动 worker，并滚动重启 app；新 PM Redis 中的 KPI L2 从空缓存安全重建；
  8. 保留 core 中旧 key，不立即删除；
  9. 若 PM 健康失败，停 worker、切回 fallback core、重启。

- [x] **Step 5: 跑命令测试并提交**

  Run: `cd omcgo && go test ./cmd/omcctl -count=1`

  Expected: PASS。

  Commit:

  ```bash
  git add omcgo/cmd/omcctl deployments/release/bundle/deploy
  git commit -m "feat: 增加PM Redis安全迁移工具"
  ```

## Task 8: 完整静态验证、性能回归和代码审查

**Files:**

- Modify if needed: only files already listed above

- [ ] **Step 1: 格式化并检查差异范围**

  Run:

  ```bash
  git diff --name-only --diff-filter=ACM origin/main -- '*.go' | xargs gofmt -w
  cd .. && git diff --check && git status --short
  ```

  Expected: `git diff --check` 无输出；没有无关文件。

- [ ] **Step 2: 后端全量构建、vet 和测试**

  Run:

  ```bash
  cd omcgo
  go build ./...
  go vet ./...
  go test ./... -count=1
  ```

  Expected: 全部退出 0。若 miniredis/httptest 因 sandbox 本地监听失败，按仓库规则提权原命令复跑并记录两次结果。

- [ ] **Step 3: 前端回归验证**

  Run: `cd omcmb && npm run typecheck`

  Expected: PASS；本次无 UI 功能修改，但必须证明共享类型/构建未受影响。

- [ ] **Step 4: 部署与监控配置验证**

  Run:

  ```bash
  for test in deployments/release/bundle/deploy/*_test.sh; do bash "$test"; done
  docker compose -f deployments/docker/docker-compose.yml config --quiet
  ```

  Expected: 全部 PASS，compose 无未解析/非法依赖。

- [ ] **Step 5: 用真实双 Redis 验证迁移类型和 TTL**

  启动 Compose 的 `redis-core` / `redis-pm`，在 source 创建带 TTL 和永久的 string/hash/set/zset/list/stream 测试 key，运行 `omcctl pm-redis migrate --pattern 'pmagg:test:*'`。逐类型比较 `TYPE`、`DUMP` hash 和 TTL 容差，最后删除仅带 `pmagg:test:` 前缀的测试 key。不得删除真实 `pmagg:*`。

- [ ] **Step 6: 做本地/测试环境 20k 合成 PM 性能对比**

  用仓库已有 PM 压测工具（先查 README 和现有脚本，不新造第二套）对同一固定数据集记录 before/after：

  - revision cleanup SQL 次数与累计耗时；
  - hour finalizer wall time、p95/p99；
  - TSDB CPU、`pg_stat_statements`、temp bytes、block read/write；
  - Redis core/PM ops、latency、memory、AOF fsync；
  - queue peak/backlog drain time。

  Gate：revision=1 cleanup DELETE=0；周期重算 temp bytes 不再随整周期结果线性增长；无死锁/锁等待回归；若 +15m 目标未达到，停止上线并回到 Task 3/4 分析，不能直接把并发调到 64。

- [ ] **Step 7: 请求代码审查并处理发现**

  使用 `superpowers:requesting-code-review` 检查设计覆盖、并发正确性、Redis 路由遗漏、迁移回滚、SQL 索引与上线风险。修复必须补回归测试并重新执行受影响验证。

- [ ] **Step 8: 提交验证修正**

  若有修正：

  ```bash
  git add -u
  git commit -m "test: 完善Redis隔离与收敛回归验证"
  ```

## Task 9: Push、创建 MR、合入并构建发布包

**Files:**

- Create: `docs/superpowers/plans/2026-08-02-paramsync-pm-redis-isolation-mr.md`

- [ ] **Step 1: 用完成前验证技能复核证据**

  使用 `superpowers:verification-before-completion`，重新执行 Task 8 的关键命令，不复用旧日志宣称通过。

- [ ] **Step 2: Push 分支并创建 MR**

  ```bash
  git push -u origin codex/paramsync-pm-redis-isolation
  glab mr create --source-branch codex/paramsync-pm-redis-isolation --target-branch main \
    --title "fix: 根治参数同步收敛并隔离PM Redis" \
    --description-file docs/superpowers/plans/2026-08-02-paramsync-pm-redis-isolation-mr.md
  ```

  MR 描述必须列出：根因、状态机不变量、Redis key 路由矩阵、TSDB before/after、迁移/回滚 runbook、测试证据和线上验收表。

- [ ] **Step 3: 等待流水线并处理审查**

  所有 pipeline job 必须通过；收到意见时使用 `superpowers:receiving-code-review`，验证后修改，禁止盲目接受或跳过测试。

- [ ] **Step 4: 合入 MR 并确认 main commit**

  按项目既有合入策略执行，不强推、不 `--no-verify`。记录 merge commit SHA，确认远端 main 包含全部提交。

- [ ] **Step 5: 从已合入 main 构建发布包**

  必须使用 merge 后 commit 构建，不从 feature branch 或工作区脏状态直接部署。执行项目 release README 的标准构建命令并校验镜像 digest/版本标签。

## Task 10: 生产迁移、部署和跨小时验收

**Files:**

- No source changes expected; evidence goes to MR/deployment record

- [ ] **Step 1: 部署前只读基线采集**

  在 `172.24.224.197` 记录：当前 commit/image digest、全部容器健康、ACS 503/session/reject、PM 主队列与 `pm-registration-wait`、设备任务、死信、K900010006/K900010076、PM 文件新鲜度、PostgreSQL/TSDB locks 和 top SQL、CPU/内存/磁盘 I/O、Redis key/memory/AOF，以及 hourly/daily/weekly 最新结果和覆盖率。

- [ ] **Step 2: 执行资源 planner 和部署前门禁**

  重新生成 schema v3 `resources.env`，人工核对整机总账允许超配但不超过物理内存安全线；执行 `install.sh --check-only`。任何失败先修复，不带病部署。

- [ ] **Step 3: 安全迁移 PM Redis**

  严格按 Task 7 runbook：等待已发布窗口、停 worker、启动 redis-pm、dry-run、复制、hash/TTL/key count 验证、切换 worker 并滚动重启 app。旧 core PM key 暂不删除，因此回滚可恢复到旧 endpoint；KPI L2 不迁移，从空缓存按 DB 重建。

- [ ] **Step 4: 部署 merge 后最新版本并做即时烟测**

  核对所有服务 healthy，worker 两个 Redis health 均通过；ACS 无 503、无 session 拒绝；核心任务队列可生产消费；PM 文件可接收解析；非法启动期 PM 直接丢弃且不进死信；K900010006/K900010076 有新鲜数据。

- [ ] **Step 5: 验收参数同步收敛**

  触发代表性同步并检查：

  - `expected=terminal=processed` 后 30 秒内 run 进入 succeeded/failed；
  - `ready_but_not_finalized=0`；
  - `counter_drift=0`；
  - 非真实设备任务的 active run 不超过 5 分钟；
  - 历史 500 个已 ready stuck run 被维护扫描逐批收敛，不清库。

- [ ] **Step 6: 跨越完整小时关窗验收**

  选择部署后第一个完整小时窗口：

  - `window_end + 12m ± 30s` 进入 finalizing；
  - 20,000 设备 hourly 全部发布不晚于 `window_end + 15m`；
  - daily/weekly 当前进行中结果生成，覆盖率/已收到槽位/应有槽位/版本有效区间正确；
  - 不把旧版本片段完整误判为自然周期完整；
  - Dashboard 仍只每 5 分钟刷新，读取聚合结果，无原始明细全量扫描。

- [ ] **Step 7: 验收性能和隔离效果**

  同窗口记录并与基线对比：

  - redis-core 不再出现 PM 波峰带来的 latency/CPU/AOF I/O 峰值；
  - redis-pm 无 rejected connections、OOM、eviction、AOF 错误；
  - 主库 CPU/I/O 无异常，参数同步查询不做历史全表饥饿扫描；
  - TSDB 无 revision=1 cleanup DELETE 热点，无长锁/慢查询堆积，temp write 明显下降；
  - PM 主队列和 registration-wait 在目标时间排空，无死信增长；
  - 所有 Prometheus 业务告警恢复 green。

- [ ] **Step 8: 失败处理与最终结论**

  任一 gate 未达标：保留现场，按 `superpowers:systematic-debugging` 找根因，补失败测试、修复、重新走 MR/合入/部署/跨小时验收；不得清库或只扩资源。全部 gate 达标后，使用 `superpowers:verification-before-completion` 汇总 commit、MR、部署版本、时间窗口和量化结果，再宣告完成。

---

## Final Self-Review Checklist

- [ ] 规格覆盖：参数同步状态机、扫描公平性、异常分类、指标/告警均有任务和测试。
- [ ] 规格覆盖：PM/KPI 与核心 Redis 物理隔离，路由矩阵、资源规划、健康检查、迁移、回滚均有任务。
- [ ] 规格覆盖：12 分钟关窗未改变，首次发布 DELETE、索引、周期重算临时写放大均有修复和性能 gate。
- [ ] 规格覆盖：首页 hourly/daily/weekly、5 分钟定时刷新、无 15 分钟粒度和无事件刷新均被明确保护。
- [ ] 类型一致：`AppConfig.PMRedis` / `WorkerConfig.PMRedis`、`appInfra.PMRedis` / `workerInfra.PMRedis`、Prometheus role label、Redis service 名称在配置/代码/compose/测试一致。
- [ ] 迁移安全：TTL、幂等、冲突、dry-run、停止 writer、验证和回滚全部明确，未要求删除旧 key。
- [ ] 无占位：实施者不需要再猜关键文件、核心接口、测试入口、验收阈值或上线顺序。
- [ ] 完成标准：只有 MR 合入、merge commit 部署并跨完整小时达到业务/性能/数据 gate 才算完成。

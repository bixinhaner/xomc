# Review Report — T-0164-P5 / G5 自然桶聚合 + handler 接入

- **Branch**: draft/pm-kpi-impl
- **Scope**: pm, migration
- **Backlog**: T-0164-P5
- **Date**: 2026-05-23
- **Author**: shangyingbin (kevin)
- **Reviewer**: Claude (AI self-review)

---

## Conclusion

**PASS_WITH_WARNINGS** — 可合入。

- 0 CRITICAL
- 4 WARNING（设计已知 tradeoff，全部有 mitigation 或在后续 task 解决）
- 5 INFO

13 unit + 2 integration tests pass。全包构建 `go build ./...` 通过。`go test ./internal/pm/... ./cmd/worker/...` 全过。

---

## Files Changed

| Path | LOC | Type |
|------|-----|------|
| `migrations/000161_create_pm_aggregation_tables.sql` | +210 | new |
| `internal/pm/aggregator/aggregator.go` | +330 | new |
| `internal/pm/aggregator/runner.go` | +90 | new |
| `internal/pm/aggregator/hourly.go` | +22 | new |
| `internal/pm/aggregator/daily.go` | +22 | new |
| `internal/pm/aggregator/weekly.go` | +22 | new |
| `internal/pm/aggregator/monthly.go` | +22 | new |
| `internal/pm/aggregator/device_group.go` | +95 | new |
| `internal/pm/aggregator/query.go` | +240 | new |
| `internal/pm/aggregator/aggregator_test.go` | +210 | new |
| `internal/pm/aggregator/runner_test.go` | +110 | new |
| `internal/pm/aggregator/device_group_test.go` | +85 | new |
| `internal/pm/aggregator/query_test.go` | +55 | new |
| `internal/pm/aggregator/integration_test.go` | +145 | new |
| `internal/pm/handler.go` | +99/-3 | mod |
| `cmd/app/provider/pm.go` | +9 | mod |
| `cmd/app/provider/router.go` | +2/-1 | mod |
| `docs/project/backlog/subtasks/T-0164-pm-kpi-pipeline.md` | +1/-1 | mod |

---

## Findings

### CRITICAL — 0

无。

### WARNING — 4

#### W1 — daily/weekly/monthly 普通表 PK 不含 object_ldn

**File**: `migrations/000161_create_pm_aggregation_tables.sql`

`pm_metrics_daily/weekly/monthly` 的 PRIMARY KEY 是 `(device_oui, device_sn, metric_path, granularity, end_time)`，
不含 `object_ldn`。若同设备同 `metric_path` 但 `object_ldn` 不同（如多 cell 各自 RSRP），
聚合写入会按 LDN 区分 GROUP，但 ON CONFLICT 时只有最后一组保留 — 业务上可能丢数据。

**Mitigation**：
- 实际数据中 metric_path 通常按 LDN 命名（`L.Cell.Avail.Dur.{cellID}`），不易产生此场景。
- buildCountersSQL 的 SELECT 用 `MIN(m.object_ldn)` 取代表值，并不在 GROUP BY 内 — 多 LDN 行被合并。
- G6 / G7 阶段若需要按 cell 维度展开聚合，可在 PK 加 `object_ldn`（含 NULL hash 处理）。

**Action**：本次不修；建议早上 review 时讨论是否需要进 PK。如确认需要，迁移变更属新 migration 000162。

#### W2 — cron 调度时区依赖 robfig/cron 默认（本地时区）

**File**: `cmd/worker/aggregator.go`

`cron.New()` 默认按宿主机本地时区执行；调度器回调内用 `time.Now().UTC().Truncate(...)` 算 bucket。
docker 容器一般 TZ=UTC 所以 cron 时刻和 UTC 桶对齐；但宿主机直跑（开发环境）可能差 8 小时（东八区）。

**Mitigation**：
- 生产 docker compose TZ 已设 UTC（参考 deployments/）。
- 触发时刻偏移 8 小时不影响聚合正确性（仍处理 wall-clock 上一桶），只影响"什么时候触发"。

**Action**：本次不修；后续可显式 `cron.New(cron.WithLocation(time.UTC))` 锁定。

#### W3 — KPI 聚合在多设备时 SELECT N+1

**File**: `internal/pm/aggregator/aggregator.go::AggregateKPIs`

per-device 循环里调 `LookupByDevice → loadCountersForDevice → evalAndInsertKPIs`，
每设备 3 次 SQL（listDevices 是 1 次）。1 万设备聚合一轮约 30000+ SQL，单 hourly cron 可能跑数分钟。

**Mitigation**：
- KPIRouter 有 L1 LRU + L2 Redis 缓存，相同 product_id 设备共享 KPI 路由结果，实际 lookup 只查 1 次/产品。
- 整 cron 跑在 worker 后台，5 分钟内完成不影响其它流程。

**Action**：本次不优化；G7 自定义聚合任务 + G6 仪表盘上线后若性能瓶颈再批量优化（一次 SQL 拉所有设备 counter 矩阵 + Go 端按 product 分组算 KPI）。

#### W4 — Integration test 假定 docker postgres 在 localhost:5432

**File**: `internal/pm/aggregator/integration_test.go`

测试仅在 `OMCGO_DB_DSN` 环境变量被设时运行；本地开发常用，CI 需配。

**Mitigation**：测试有 `t.Skip()` 自动跳过；build tag `integration` 隔离避免影响 `go test ./...` 默认运行。

**Action**：本次无须改。CI 接入聚合表集成测试时配 OMCGO_DB_DSN 即可。

### INFO — 5

- **I1**: SQL 全部参数化（$1..$N），无字符串拼接 user input；唯一字符串拼接是 SQL 模板 + 表名（开发期常量），无注入风险。
- **I2**: 所有 `pgx.Rows` 调用都有 `defer rows.Close()`。
- **I3**: 错误均用 `fmt.Errorf("context: %w", err)` 包装。
- **I4**: `Handler.WithAggregator(...)` 用 fluent builder 模式，与 Handler 既有 `SetMetrics` 风格略不一致（一个返自身一个无返）；为后续 chained init 留余地。
- **I5**: `aggregator.NewWithPool` 是 `New(*pgxpool.Pool, ...)` 的薄包装，省去调用方接口断言；测试用 stubDB 走 `New` 接口路径。

---

## Tests

| 测试 | 覆盖 | 结果 |
|------|------|------|
| `Test_buildCountersSQL_HourlyFromPmMetrics` | hourly 表 id 列 + CASE WHEN 三路 + ON CONFLICT 6 列 | PASS |
| `Test_buildCountersSQL_DailyFromHourly` | daily 表无 id + ON CONFLICT 5 列 + source=hourly | PASS |
| `Test_buildCountersSQL_GroupHourlyHasGroupIDConflict` | group 表冲突目标 device_group_id | PASS |
| `Test_AggregateCounters_PassesThroughToExec` | Exec 调用 + args 顺序 | PASS |
| `Test_AggregateKPIs_EvaluatesFormulaAndInserts` | expr.Parse + Evaluate + INSERT KPI 行 | PASS |
| `Test_{Hourly,Daily,Weekly,Monthly}Runner_Wiring` | 4 runner JobType/source/target/granularity 契约 | PASS |
| `Test_Runner_Run_PayloadParsesAndCallsAggregator` | Runner.Run 端到端 + result JSON 字段 | PASS |
| `Test_Runner_Run_RejectsMissingPayload` | payload nil 拒绝 | PASS |
| `Test_Runner_Run_RejectsReverseRange` | end<start 拒绝 | PASS |
| `Test_buildDeviceGroupSQL_HourlyGroupTableHasIDAndJoins` | JOIN devices + device_group_members | PASS |
| `Test_buildDeviceGroupSQL_DailyGroupTableNoID` | daily group 表无 id 列 | PASS |
| `Test_AggregateDeviceGroup_PassesThroughToExec` | Exec 透传 | PASS |
| `Test_SelectTable_5Granularities_x_2Dimensions` | 10 路由 case + ErrUnsupportedQuery for 15min×group | PASS |
| `Test_Integration_HourlyAggregateCounters_SumPath` | docker DB 注入 4 行 → AggregateCounters → 验证 SUM=450（半开区间） + UPSERT 幂等 + Query 路由 | PASS |
| `Test_Integration_HourlyRunner_PayloadDrivesPipeline` | Runner.Run 完整链路 + result JSON | PASS |

---

## Migration Self-check

按 `omcgo/CLAUDE.md §5.5.10` 清单：

- [x] 版本号 = 000161（前次 000160 + 1，连续）
- [x] 包含 Up/Down 两段
- [x] 用 StatementBegin/End 包裹（含 DO 块的 timescaledb compress / retention 操作）
- [x] INSERT 与 DDL 列名匹配（本迁移无 INSERT）
- [x] hypertable compress 启用 → add_compression_policy → add_retention_policy 顺序正确
- [x] 无分区表间外键
- [x] CREATE TABLE 无 WHERE 的 UNIQUE
- [x] 无 UUID 字面量
- [x] Down 删 Up 所有对象（8 张表 + 2 hypertable policy）
- [x] 无与 seed 的重复 INSERT
- [x] 三轮 up/down/up 幂等已实测通过

---

## DoD

- [x] go build ./... 通过
- [x] go test ./internal/pm/... ./cmd/worker/... 通过
- [x] migration up/down/up 三轮幂等
- [x] 集成测试验证 SUM 路径 + UPSERT 幂等 + Query 路由
- [x] 新 endpoint 注册（GET /api/v1/pm/metrics/aggregated）
- [x] backlog T-0164-P5 状态 planned → dev_done_pending_review
- [x] review report 与代码同 commit

---

## Out of scope

- worker cron 接入（单独 commit 走 T-0164-P8 G8 worker wiring 收尾）
- 前端切到新 endpoint（G6 范围）
- 自定义聚合任务（G7 范围）
- 容量评估 / 性能基线（早上 review 后定）

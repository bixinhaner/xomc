# PM 上卷设备查询优化 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 让设备维度小时、日、周、月 KPI 查询在过滤目标设备后直接读取统一上卷结果表，消除兼容视图全量去重导致的秒级延迟。

**Architecture:** 保留 `SelectTable` 和所有 API 契约，用 `newRawAwareDeviceSelect` 作为唯一数据源路由点。15 分钟继续读取稀疏原始表；设备上卷粒度改为投影 `pm_aggregation_results`，先按 `dimension_key`、粒度和请求条件过滤，再由现有外层逻辑执行最新结果去重、分页和补空。

**Tech Stack:** Go、Squirrel SQL Builder、pgx、PostgreSQL/TimescaleDB、testify、Docker Compose

## Global Constraints

- 设备上卷查询不得读取 `pm_metrics_hourly/daily/weekly/monthly` 兼容视图。
- 页面、导出、权限、透视分页、补空和 JSON 返回字段保持不变。
- 15 分钟设备查询和非设备维度查询保持现状。
- 不新增数据库表、索引、迁移、NATS 或 Redis 改动。
- 所有生产代码必须先由失败测试驱动。

---

## File Structure

- Modify: `omcgo/internal/pm/aggregator/query.go`
  - 增加设备上卷表识别和统一结果表投影。
  - 扩展设备 UUID 预过滤，使其同时适配原始锚点和统一上卷结果表。
  - 保持现有去重、分页、权限与扫描逻辑不变。
- Modify: `omcgo/internal/pm/aggregator/query_test.go`
  - 覆盖四种上卷粒度的物理数据源、过滤下推、兼容投影及非目标路由回归。
- Create: `docs/superpowers/plans/2026-07-28-pm-rollup-device-query-optimization.md`
  - 记录本次 TDD 开发与统一验收步骤。

### Task 1: 用失败测试锁定上卷设备查询契约

**Files:**
- Modify: `omcgo/internal/pm/aggregator/query_test.go`

**Interfaces:**
- Consumes: `newRawAwareDeviceSelect(builder, table, q, columns...) sq.SelectBuilder`
- Produces: 四种上卷粒度直接查询底表的回归契约

- [ ] **Step 1: 写四粒度物理路由测试**

将现有 `TestRawAwareDeviceSelectUsesCorrectPhysicalSource` 扩展为表驱动测试。对 hourly、daily、weekly、monthly 分别构造请求并断言：

```go
assert.Contains(t, sql, "FROM pm_aggregation_results r")
assert.NotContains(t, sql, "FROM "+table)
assert.Contains(t, sql, "r.dimension =")
assert.Contains(t, sql, "r.granularity =")
```

同时保留 15 分钟分支读取 `pm_measurement_anchors` 的断言。

- [ ] **Step 2: 写过滤下推与投影兼容测试**

构造包含 OUI、SN、制式、KPI、时间窗、对象和可见分组的小时请求，调用 `buildDeviceTableSQL`，使用手工推导的 SQL 片段断言：

```go
assert.Contains(t, sql, "r.dimension_key IN (SELECT id::text FROM device_dim")
assert.Contains(t, sql, "r.metric_path")
assert.Contains(t, sql, "r.window_start")
assert.Contains(t, sql, "r.object_ldn")
assert.Contains(t, sql, "r.created_at AS ingest_time")
assert.Contains(t, sql, "jsonb_build_object")
assert.Contains(t, sql, "DISTINCT ON")
assert.NotContains(t, sql, "FROM pm_metrics_hourly")
```

并核对参数中仍含设备、制式、指标、时间、对象和可见分组。

- [ ] **Step 3: 写非设备查询不改变的回归测试**

调用 `newRawAwareDeviceSelect` 时传入非设备快表名，断言仍从传入表读取；保留 `SelectTable` 对 product、band、network 的既有路由测试，证明本次没有改变它们的表名契约。

- [ ] **Step 4: 运行目标测试并确认 RED**

Run:

```bash
cd omcgo
go test ./internal/pm/aggregator -run 'TestRawAwareDeviceSelect|Test_buildDeviceTableSQL_RolledUp'
```

Expected: FAIL；现有代码仍包含 `FROM pm_metrics_hourly`，没有 `FROM pm_aggregation_results r`。

### Task 2: 实现设备上卷底表查询

**Files:**
- Modify: `omcgo/internal/pm/aggregator/query.go`
- Test: `omcgo/internal/pm/aggregator/query_test.go`

**Interfaces:**
- Consumes: `QueryRequest`、`deviceTableColumns`、`applyDeviceFilters`
- Produces:
  - `isRolledUpDeviceTable(table string) bool`
  - `newRolledUpDeviceSelect(builder sq.StatementBuilderType, q QueryRequest, columns ...string) sq.SelectBuilder`
  - 可同时为 `a.device_dim_id` 和 `r.dimension_key` 生成设备 ID 预过滤的内部辅助逻辑

- [ ] **Step 1: 增加上卷表识别**

实现严格白名单：

```go
func isRolledUpDeviceTable(table string) bool {
	switch table {
	case "pm_metrics_hourly", "pm_metrics_daily", "pm_metrics_weekly", "pm_metrics_monthly":
		return true
	default:
		return false
	}
}
```

不能根据任意字符串拼接表名。

- [ ] **Step 2: 增加统一结果表投影**

`newRolledUpDeviceSelect` 从 `pm_aggregation_results r` 投影与 `deviceTableColumns` 完全相同的列名：

```go
r.device_oui AS device_oui
r.device_sn AS device_sn
r.metric_path AS metric_path
r.metric_type AS metric_type
r.metric_value AS metric_value
r.aggregation_op::text AS statis_type
r.granularity::text AS granularity
r.window_start AS time
r.window_start AS start_time
r.window_end AS end_time
r.created_at AS ingest_time
NULLIF(r.object_ldn, '') AS object_ldn
jsonb_build_object(
  'task_id', r.task_id,
  'task_version_id', r.task_version_id,
  'complete', r.complete,
  'missing_slots', r.missing_slots
) AS extra
```

基础条件固定为 `r.dimension = 'device'`。请求的设备、粒度、指标、时间和对象条件通过列别名可见的内层选择器处理，确保全部位于外层 `DISTINCT ON` 之前。

- [ ] **Step 3: 将设备 UUID 预过滤应用到上卷结果**

复用当前 `deviceDimIDPrefilter` 的设备选择语义，将目标列参数化：

```go
func deviceDimIDPrefilter(column string, q QueryRequest) (string, []any, bool)
```

15 分钟传 `a.device_dim_id` 并生成 `SELECT id`；上卷传 `r.dimension_key` 并生成 `SELECT id::text`。OUI/SN 成对、SN+制式、仅 OUI、仅 SN 四种分支保持与 `applyDeviceFilters` 一致。

- [ ] **Step 4: 接入数据源路由**

`newRawAwareDeviceSelect` 的顺序固定为：

1. `table == "pm_metrics"` 且指定指标：稀疏原始优化路径；
2. `isRolledUpDeviceTable(table)`：统一上卷结果路径；
3. 其他表：原样 `From(table)`。

上卷分支返回带别名的子查询，使现有 `applyDeviceFilters`、`DISTINCT ON`、透视分页和 `queryDeviceTable` 无需改变。

- [ ] **Step 5: 运行目标测试并确认 GREEN**

Run:

```bash
cd omcgo
go test ./internal/pm/aggregator -run 'TestRawAwareDeviceSelect|Test_buildDeviceTableSQL_RolledUp|Test_SelectTable'
```

Expected: PASS。

- [ ] **Step 6: 运行聚合包测试**

Run:

```bash
cd omcgo
go test ./internal/pm/aggregator
```

Expected: PASS。

### Task 3: 统一验证、部署与 MR 更新

**Files:**
- Modify only if verification exposes a defect: `omcgo/internal/pm/aggregator/query.go`
- Modify only if a regression test is needed: `omcgo/internal/pm/aggregator/query_test.go`

**Interfaces:**
- Consumes: 已完成的查询优化
- Produces: 可部署构建、线上性能证据和更新后的 MR

- [ ] **Step 1: 格式化并检查差异**

Run:

```bash
gofmt -w omcgo/internal/pm/aggregator/query.go omcgo/internal/pm/aggregator/query_test.go
git diff --check
git diff --stat
```

Expected: 无格式或空白错误，改动仅限计划范围。

- [ ] **Step 2: 统一执行后端构建与测试**

Run:

```bash
cd omcgo
go build ./...
go test ./...
```

Expected: 两条命令均 exit 0。

- [ ] **Step 3: 验证前端类型契约**

Run:

```bash
cd omcmb
npm run typecheck
```

Expected: exit 0；本次无需修改前端。

- [ ] **Step 4: 提交代码**

Run:

```bash
git add omcgo/internal/pm/aggregator/query.go omcgo/internal/pm/aggregator/query_test.go docs/superpowers/plans/2026-07-28-pm-rollup-device-query-optimization.md
git commit -m "perf(pm): 优化设备上卷查询"
```

- [ ] **Step 5: 构建并部署最新交付包**

沿用仓库 `deployments/release` 的现有交付脚本生成版本包，上传到 `172.24.224.197`，使用 `docker compose -f deployments/docker/docker-compose.yml` 更新容器，不使用裸进程重启脚本。

- [ ] **Step 6: 执行业务与性能验收**

对设备 `120200024118AA01241` 的相同时间窗验证：

- 15 分钟接口仍正常；
- 小时已有 KPI `K900010002` 返回 4 个桶；
- 小时空指标仍返回兼容的空/补空结果；
- 页面和导出字段、分页、补空不变；
- 小时 SQL 的 `EXPLAIN (ANALYZE, BUFFERS)` 命中 `idx_pm_aggregation_results_dimension_time`，不再扫描 440,000 行兼容视图；
- 服务、CPU、内存、磁盘、数据库和 NATS 队列无新增异常。

- [ ] **Step 7: 推送分支并更新 MR**

Run:

```bash
git push origin codex/pm-event-driven-rollup-redesign
```

确认 MR `!370` 指向最新提交、可合并且流水线无失败；只有全部验证通过才报告完成。

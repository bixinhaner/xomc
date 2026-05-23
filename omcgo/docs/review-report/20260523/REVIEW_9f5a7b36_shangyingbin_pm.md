# 代码审查报告 — T-0164-P3 / G3 合表 + 删物化视图

**Base commit**: `9f5a7b36`
**Scope**: `pm` (+ migration / dashboard 配套)
**Author**: shangyingbin (Claude)
**Date**: 2026-05-23
**Backlog**: T-0164-P3
**PRD**: docs/design/pm-kpi-pipeline-improvements.md §4.3
**Sprint**: wave-3
**Risk**: -

---

## 1. 变更范围

| 文件 | 类型 | 说明 |
|------|------|------|
| `migrations/000160_create_pm_metrics_and_drop_legacy.sql` | 新增 | DROP MATERIALIZED VIEW pm_counters_hourly + DROP TABLE pm_counters + kpi_values；CREATE pm_metrics（hypertable + 5 索引 + compression 7d + retention 30d） |
| `internal/pm/metrics/model.go` | 新增 | PMMetric struct + MetricType/StatisType/Granularity 类型常量 |
| `internal/pm/metrics/pg_repository.go` | 新增 | Repository 接口 + PgRepository（Insert/BatchInsert/Query/Count，ON CONFLICT 幂等） |
| `internal/pm/metrics/pg_repository_test.go` | 新增 | applyFilters 7 case + 类型常量稳定性 1 case |
| `internal/pm/kpi/calculator.go` | 新增 | AggregateByStatisType(sum/avg/max/pct) 工具函数（G5 cron 消费 statis_type 的核心算子） |
| `internal/pm/kpi/calculator_test.go` | 新增 | 6 case（4 statis_type + empty + unknown） |
| `internal/pm/counter/pg_repository.go` | 重写 | 改为 metrics.Repository 的薄包装：PMCounter ↔ PMMetric 字段转换；QueryAggregated 降级为查 15min 后内存按小时桶聚合 |
| `internal/pm/kpi/pg_repository.go` | 重写 | 改为 metrics.Repository 的薄包装：KPIValue ↔ PMMetric 字段转换；ListDefinitions/SyncDefinitions 仍走 kpi_definitions 表（未动） |
| `internal/pm/aggregation/*` | 删除 | aggregator.go + aggregator_test.go + .gitkeep（pm_counters_hourly 物化视图已删，G5 hourly 表上线后重做） |
| `internal/pm/counter/counter_test.go` | 重写 | 移除已删私有函数 applyCounterFilters/allowedSortColumns 的测试，新增 PMCounter ↔ PMMetric round-trip 字段守恒测试 |
| `internal/core/model/pm.go` | 改注释 | PMCounter / KPIValue 文档串改为指向 pm_metrics + wrapper 设计说明 |
| `internal/dashboard/service.go` | 改 1 SQL | GetKPITimeSeries: From("kpi_values") → From("pm_metrics") + Where(metric_type='kpi')；列名 kpi_name → metric_path / kpi_value → metric_value |
| `scripts/seed_e2e_testdata.sql` | 改 | DELETE/INSERT/COUNT 全部从 pm_counters/kpi_values 迁到 pm_metrics（device_id uuid → device_sn text；counter_group/carrier/technology 进 extra JSONB；三时间字段填入；ON CONFLICT 加 time 列） |
| `scripts/seed_test_data.sql` | 改 | 20000 行 PM + 20000 行 KPI 适配 pm_metrics 结构（旧脚本本来就因列名不匹配旧 schema 跑不通，本次顺带修复） |

**净变化**：+412 / -507 行（净减 95 行）

---

## 2. 设计决策记录

### 2.1 DeviceSN 过渡策略（关键）

**问题**：设计文档 §4.3 定 pm_metrics.device_sn 为 text 真实序列号，但调用链上游 PMCounter/KPIValue 持 UUID。

**决策（方案 B+C 混合）**：
- 保留 PMCounter/KPIValue 原 struct（model 不动，对外 API 契约不变）
- counter/kpi pg_repository.go 改为 metrics.Repository wrapper：BatchInsert 时 DeviceSN = DeviceID.String()
- 过渡期 device_sn 列存的是 UUID 字符串（如 `b9dfacaf-a15b-4b88-995c-aa6a5aa37c28`）
- 后续 collector 改为传真实 device serial 后清理（本次 G3 不做，单独 PR）

**理由**：
- 项目未上生产，0 行旧数据，过渡可接受
- 改动面最小化：collector / KPIEngine / handler / northbound 接口全不动
- 设计文档列定义保留（pm_metrics.device_sn TEXT）

### 2.2 KPIEngine 实时聚合不动

按 plan §0："G3 阶段实时计算层不做 SQL 聚合（聚合下沉到 G5 cron），保留计算单 PM 文件内的 arithmetic 公式即可。"

- KPIEngine.Calculate 流程不变：调 counter wrapper 的 QueryForKPI（内存聚合）→ 公式求值 → BatchInsert
- AggregateByStatisType 是为 G5 cron 准备的工具函数，G3 阶段尚不在调用链上

### 2.3 QueryAggregated 降级

旧实现走 pm_counters_hourly 物化视图（已删）。

G3 阶段降级：查 15min 粒度 pm_metrics 后在内存按 hour bucket 聚合（与旧 1h 窗口一致）。
- 单文件维度数据量有限，开销可接受
- G5 上线 hourly 聚合表后由路由层替换

### 2.4 UNIQUE 索引含 time 列

**Dry-run 阶段发现的真 bug**：TimescaleDB 要求所有 UNIQUE 索引必须包含分区列。

修正：`UNIQUE (device_sn, metric_path, granularity, end_time, time)`
- 业务上 time = end_time，约束语义不变
- ON CONFLICT 子句同步加 time 列（metrics/pg_repository.go + 2 个 seed sql）

---

## 3. 审查检查清单

### 3.1 Go 后端规范

- [x] **命名**：导出 PascalCase（PMMetric, MetricType, NewPgRepository），未导出 camelCase（applyFilters, counterToMetric）
- [x] **错误处理**：所有 error 用 `fmt.Errorf("ctx: %w", err)` 包装；无裸 panic
- [x] **接口优先**：metrics.Repository 接口定义在 model 之上；PgRepository 通过 `var _ Repository = (*PgRepository)(nil)` 验证
- [x] **SQL 安全**：全 Squirrel 构建（含 ON CONFLICT 通过 .Suffix 拼接）；占位符 `$N`
- [x] **运营商无硬编码**：counter/kpi wrapper 内 Extra JSONB 透传 carrier/technology，无 `if carrier == "cmcc"`
- [x] **资源释放**：所有 rows.Query 后 defer rows.Close()
- [x] **JSON 编解码**：metrics.PgRepository.BatchInsert json.Marshal Extra；Query 反向解码（容错处理：解码失败 m.Extra = nil 不阻塞）
- [x] **测试覆盖**：metrics applyFilters 7 case（DeviceSNs IN / MetricPaths IN / MetricType / Granularity / TimeRange / Combined / Empty）+ 类型常量稳定性 + calculator 6 case（含 pct error path）+ counter round-trip 2 case

### 3.2 Migration 自查（按 omcgo/CLAUDE.md §5.5.10）

- [x] 版本号 = 159 + 1 = 160（无跳跃、无重复，`ls migrations/ | sort | tail -3` 确认）
- [x] 包含 `-- +goose Up` / `-- +goose Down` 两段
- [x] `-- +goose StatementBegin / StatementEnd` 包裹（虽然纯 DDL 非 PL/pgSQL，但 plan 也包了）
- [x] **TimescaleDB 顺序正确**：create_hypertable → ALTER SET (compress) → add_compression_policy → add_retention_policy
- [x] **TS UNIQUE 索引含分区列**（dry-run 阶段抓出修正）
- [x] CHECK 约束完整：metric_type IN ('counter','kpi') / statis_type NULL OR IN ('sum','avg','max','pct') / granularity IN ('15min','hourly','daily','weekly','monthly')
- [x] Down 段完整：删 retention/compression policy → DROP TABLE pm_metrics CASCADE → CREATE 旧表空 schema 兜底
- [x] 无 INSERT / 无 PL/pgSQL DO 块
- [x] **Dry-run 通过**（docker exec psql 事务内跑完整 Up SQL + INSERT + 验证查询 + ROLLBACK，全部正常）

### 3.3 兼容性 / 影响面

- [x] dashboard/service.go GetKPITimeSeries SQL 已配套改造（删表后不会 runtime break）
- [x] report/generator.go 第 197 行 "kpi_values" 是 JSON map key（非 SQL），不影响
- [x] northbound/service.go + sync/service.go 走 counter.CounterRepository / kpi.KPIRepository 接口，wrapper 接口签名不变 → 无影响
- [x] collector / KPIEngine / pm handler 全不动
- [x] `cmd/app/provider/pm.go` wiring 不动（NewPgCounterRepository(pool) 签名保留，wrapper 内部 lazy 建 metrics.NewPgRepository(pool)）

---

## 4. 已知遗留 (INFO 级，不阻塞合入)

| # | 项 | 说明 | 后续处理 |
|---|----|------|----------|
| INFO-1 | device_sn 列存 UUID 字符串 | 过渡设计，等 collector 改造 | 单独 PR（不在 G3 范围） |
| INFO-2 | QueryAggregated 内存聚合 | 数据量大时性能受限 | G5 上线 hourly 聚合表后由路由层替换 |
| INFO-3 | Baseline 3 个 build failure | `internal/nedirect`、`internal/northbound/sync`、`test/integration` 测试用 mock 没跟上接口变化（`device.DeviceParameterRepository.DeleteByPathPrefix` 与 `software.SubTaskRepository.FailStale` 签名变了） | 与 G3 无关；git stash 验证 baseline 9f5a7b36 已存在；不在本次 commit 范围 |

---

## 5. 验证证据

- ✅ `go build ./...` 通过
- ✅ `go test ./internal/pm/...` 全过（metrics + counter + kpi + collector + indicator + retention + pm）
- ✅ Migration dry-run 通过（docker exec psql 事务内验证 DROP/CREATE/hypertable/compression/retention/INSERT/ON CONFLICT/ROLLBACK 全 OK）
- ✅ grep `pm_counters_hourly` / `pm_counters` / `kpi_values` 全仓字面值仅命中：migration 000160 自身、migration 000005 历史 DDL、注释（符合 plan §3 豁免范围）
- ✅ pg_dump 备份在 `/tmp/pm-kpi-pre-G3-backup-20260523.sql`（3567 bytes，schema only，0 行业务数据）

---

## 6. 审查结论

**PASS_WITH_INFO** — 设计意图清晰、改动面控制良好、单测+dry-run 覆盖充分。3 个 INFO 级遗留均已记录后续路径。

无 CRITICAL，无 WARNING。

可合入 `draft/pm-kpi-impl` 分支等用户晨间 review。
